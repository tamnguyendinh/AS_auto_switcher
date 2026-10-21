package automation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"antigravity-switch/internal/connector"
	"antigravity-switch/internal/manager"
	"antigravity-switch/internal/telemetry"
)

// StatusCode denotes the state machine status of the automation controller.
type StatusCode string

const (
	StatusIdle        StatusCode = "IDLE"
	StatusEvaluating  StatusCode = "EVALUATING"
	StatusRetrying    StatusCode = "RETRYING"
	StatusSwitching   StatusCode = "SWITCHING"
	StatusSwitched    StatusCode = "SWITCHED"
	StatusNoCandidate StatusCode = "NO_CANDIDATE"
	StatusError       StatusCode = "ERROR"
)

// StatusEvent encapsulates a status transition dispatched by the controller.
type StatusEvent struct {
	Code      StatusCode `json:"code"`
	Message   string     `json:"message"`
	Timestamp time.Time  `json:"timestamp"`
	Details   any        `json:"details,omitempty"`
}

var (
	// ErrSwitchInProgress is returned when a switch trigger is dropped by the single-flight gate.
	ErrSwitchInProgress = errors.New("switch already in progress (single-flight dropped)")

	// ErrAutomationDisabled is returned when a trigger is received while automation is disabled.
	ErrAutomationDisabled = errors.New("automation is disabled")

	// ErrNoCandidates is returned when no candidate account passes pre-filtering and preflight.
	ErrNoCandidates = errors.New("no eligible candidates available")
)

// SwitchDecision records the outcome of proactive trigger evaluation.
type SwitchDecision struct {
	ShouldSwitch bool                 `json:"should_switch"`
	Reason       string               `json:"reason"`
	ActiveEmail  string               `json:"active_email"`
	Evaluation   ModelQuotaEvaluation `json:"evaluation"`
}

// AutomationController coordinates the dual-trigger failover engine, single-flight gate,
// retry coordinator, and candidate selection.
type AutomationController struct {
	enabled        atomic.Bool
	isSwitching    atomic.Bool // Single-flight concurrency gate
	statusChan     chan StatusEvent
	mgr            *manager.AccountManager
	pool           *telemetry.TelemetryPool
	cache          *telemetry.QuotaCache
	retryCoord     *RetryCoordinator
	gate           *SwitchGate
	clientFinder   func(ctx context.Context) (*connector.Client, error)
	pbtxtPath      string
	activeModelID  string
	activeModelLbl string
	evaluating     atomic.Bool
	manualSwitch   atomic.Bool
	monitorMu      sync.Mutex
	refreshNow     func()

	ctx     context.Context
	cancel  context.CancelFunc
	running atomic.Bool
	mu      sync.Mutex
}

// NewAutomationController creates a new AutomationController.
func NewAutomationController(
	mgr *manager.AccountManager,
	pool *telemetry.TelemetryPool,
	cache *telemetry.QuotaCache,
	gate *SwitchGate,
) *AutomationController {
	if gate == nil {
		gate = NewSwitchGate()
	}
	c := &AutomationController{
		mgr:        mgr,
		pool:       pool,
		cache:      cache,
		gate:       gate,
		statusChan: make(chan StatusEvent, 64),
		pbtxtPath:  "",
	}
	c.clientFinder = c.defaultFindClient

	// Wire RetryCoordinator hooked into TriggerSwitch and EmitStatus
	c.retryCoord = NewRetryCoordinator(
		func(ctx context.Context) (*connector.Client, error) {
			return c.findClient(ctx)
		},
		func(ctx context.Context, reason string) {
			_, _ = c.TriggerSwitch(c.monitorContext(), reason)
		},
		func(code StatusCode, message string, details any) {
			c.EmitStatus(code, message, details)
		},
	)

	c.retryCoord.SetSendGuard(func(ctx context.Context, action func() error) error {
		if !c.enabled.Load() || c.isSwitching.Load() || c.manualSwitch.Load() {
			return ErrAutomationDisabled
		}
		_, err := c.gate.TryExecute(func(bool) error {
			if !c.enabled.Load() || c.isSwitching.Load() || c.manualSwitch.Load() || ctx.Err() != nil {
				return ErrAutomationDisabled
			}
			return action()
		})
		return err
	})
	return c
}

// SetClientFinder overrides the language server client discovery function (useful in tests).
func (c *AutomationController) SetClientFinder(fn func(ctx context.Context) (*connector.Client, error)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.clientFinder = fn
}

// SetPbtxtPath sets the path to antigravity_state.pbtxt for model discovery.
func (c *AutomationController) SetPbtxtPath(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pbtxtPath = path
}

// RetryCoordinator returns the underlying retry coordinator.
func (c *AutomationController) RetryCoordinator() *RetryCoordinator {
	return c.retryCoord
}

// Gate returns the shared switch gate used by this controller.
func (c *AutomationController) Gate() *SwitchGate {
	return c.gate
}

// IsEnabled returns whether the controller is actively enabled to perform automated switching.
func (c *AutomationController) IsEnabled() bool {
	return c.enabled.Load()
}

// SetEnabled toggles the automation state. Toggling ON triggers an immediate check.
// Toggling OFF prevents new triggers/retries but does not interrupt an in-flight switch.
func (c *AutomationController) SetEnabled(enabled bool) {
	was := c.enabled.Swap(enabled)
	if was == enabled {
		return
	}
	c.syncMonitoring()
	if enabled {
		c.EmitStatus(StatusIdle, "Tự động chuyển tài khoản: BẬT", nil)
		c.requestFreshQuota()
	} else {
		c.EmitStatus(StatusIdle, "Tự động chuyển tài khoản: TẮT", nil)
	}
}

// SetQuotaRefresh binds the existing telemetry loop; the controller owns no ticker.
func (c *AutomationController) SetQuotaRefresh(fn func()) {
	c.mu.Lock()
	c.refreshNow = fn
	c.mu.Unlock()
}
func (c *AutomationController) requestFreshQuota() {
	c.mu.Lock()
	fn := c.refreshNow
	c.mu.Unlock()
	if fn != nil {
		fn()
	}
}
func (c *AutomationController) monitorContext() context.Context {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ctx != nil {
		return c.ctx
	}
	return context.Background()
}
func (c *AutomationController) syncMonitoring() {
	c.monitorMu.Lock()
	defer c.monitorMu.Unlock()
	if c.retryCoord == nil {
		return
	}
	if c.running.Load() && c.enabled.Load() && !c.isSwitching.Load() && !c.manualSwitch.Load() {
		_ = c.retryCoord.Start(c.monitorContext())
	} else {
		c.retryCoord.Stop()
	}
}

// Manual callers pause recovery before entering the same credential gate.
func (c *AutomationController) BeginManualSwitch() { c.manualSwitch.Store(true); c.syncMonitoring() }
func (c *AutomationController) EndManualSwitch() {
	c.manualSwitch.Store(false)
	c.retryCoord.InvalidateSession()
	c.syncMonitoring()
	if c.enabled.Load() {
		c.requestFreshQuota()
	}
}

// IsSwitching returns whether an account switch is currently executing in the single-flight gate.
func (c *AutomationController) IsSwitching() bool {
	return c.isSwitching.Load()
}

// StatusChannel returns the read-only channel delivering non-blocking status events.
func (c *AutomationController) StatusChannel() <-chan StatusEvent {
	return c.statusChan
}

// EmitStatus broadcasts a status event to statusChan without blocking callers.
func (c *AutomationController) EmitStatus(code StatusCode, message string, details any) {
	evt := StatusEvent{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
		Details:   details,
	}
	select {
	case c.statusChan <- evt:
	default:
		select {
		case <-c.statusChan:
		default:
		}
		select {
		case c.statusChan <- evt:
		default:
		}
	}
}

// Start starts background monitoring and the retry coordinator.
func (c *AutomationController) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.running.Load() {
		c.mu.Unlock()
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	c.ctx, c.cancel = context.WithCancel(ctx)
	c.running.Store(true)
	c.mu.Unlock()
	c.syncMonitoring()
	return nil
}

func (c *AutomationController) Stop() {
	c.enabled.Store(false)
	c.running.Store(false)
	c.mu.Lock()
	cancel := c.cancel
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	c.syncMonitoring()
}

func (c *AutomationController) defaultFindClient(ctx context.Context) (*connector.Client, error) {
	info, err := connector.DiscoverLanguageServer()
	if err != nil {
		return nil, err
	}
	return connector.NewClient(info), nil
}

func (c *AutomationController) findClient(ctx context.Context) (*connector.Client, error) {
	c.mu.Lock()
	fn := c.clientFinder
	c.mu.Unlock()
	if fn != nil {
		return fn(ctx)
	}
	return c.defaultFindClient(ctx)
}

// OnTelemetryUpdate is the non-blocking callback invoked when telemetry data is refreshed.
func (c *AutomationController) OnTelemetryUpdate(quotas []*telemetry.AccountQuotaData) {
	if !c.enabled.Load() || c.isSwitching.Load() || c.manualSwitch.Load() || !c.evaluating.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer c.evaluating.Store(false)
		base := c.monitorContext()
		evalCtx, cancel := context.WithTimeout(base, 15*time.Second)
		decision, err := c.EvaluateProactiveTrigger(evalCtx, quotas)
		cancel()
		if err != nil {
			if c.enabled.Load() {
				c.EmitStatus(StatusError, fmt.Sprintf("Không thể kiểm tra automation: %v", err), nil)
			}
			return
		}
		if !c.enabled.Load() || base.Err() != nil {
			return
		}
		if decision != nil && decision.ShouldSwitch {
			// The operation is synchronous in this worker and does not inherit the
			// evaluation deadline or the lifetime of a stream being restarted.
			_, _ = c.triggerSwitch(base, decision.Reason, decision)
		} else if c.retryCoord != nil {
			c.retryCoord.RescanConversations(base)
		}
	}()
}

func (c *AutomationController) resolveModel(ctx context.Context) (string, string, error) {
	client, err := c.findClient(ctx)
	if err != nil {
		return "", "", err
	}
	if client == nil {
		return "", "", errors.New("language server unavailable")
	}
	c.mu.Lock()
	path := c.pbtxtPath
	c.mu.Unlock()
	id, label, _, err := client.ResolveActiveModel(ctx, path)
	if err == nil && ResolveActiveModelCategory(id, label) == CategoryUnknown {
		err = fmt.Errorf("unknown quota family for model %s", id)
	}
	return id, label, err
}

// EvaluateProactiveTrigger inspects active account quota against active model thresholds.
// Returns a decision indicating whether proactive switching should be initiated.
func (c *AutomationController) EvaluateProactiveTrigger(ctx context.Context, quotas []*telemetry.AccountQuotaData) (*SwitchDecision, error) {
	if !c.enabled.Load() {
		return &SwitchDecision{ShouldSwitch: false, Reason: "automation disabled"}, nil
	}

	if c.mgr == nil {
		return nil, errors.New("account manager cannot be nil")
	}

	activeEmail, err := c.mgr.GetCurrentActiveEmail(ctx)
	if err != nil || strings.TrimSpace(activeEmail) == "" {
		return nil, fmt.Errorf("failed resolving current active email: %w", err)
	}

	modelID, label, err := c.resolveModel(ctx)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.activeModelID, c.activeModelLbl = modelID, label
	c.mu.Unlock()
	if c.retryCoord != nil {
		c.retryCoord.SetModelName(modelID)
		c.retryCoord.SetCurrentEmail(activeEmail)
	}

	var activeQuota *telemetry.AccountQuotaData
	for _, q := range quotas {
		if q != nil && strings.EqualFold(q.Email, activeEmail) {
			activeQuota = q
			break
		}
	}
	if activeQuota == nil && c.cache != nil {
		activeQuota, _ = c.cache.Get(activeEmail)
	}

	// Rule 3: Do not switch blindly on UNKNOWN_UNCERTAIN
	if activeQuota == nil || activeQuota.Status == telemetry.StatusError {
		return &SwitchDecision{
			ShouldSwitch: false,
			ActiveEmail:  activeEmail,
			Reason:       "active account quota not available or in error state (UNKNOWN_UNCERTAIN)",
			Evaluation: ModelQuotaEvaluation{
				AccountEmail: activeEmail,
				State:        UnknownUncertain,
			},
		}, nil
	}

	eval := EvaluateAccountQuota(activeQuota, modelID, label)

	decision := &SwitchDecision{
		ShouldSwitch: eval.ShouldSwitch,
		Reason:       eval.Reason,
		ActiveEmail:  activeEmail,
		Evaluation:   eval,
	}

	return decision, nil
}

// TriggerSwitch runs the account switching pipeline through the single-flight gate and SwitchGate.
func (c *AutomationController) TriggerSwitch(ctx context.Context, reason string) (result *manager.SwitchResult, resultErr error) {
	return c.triggerSwitch(ctx, reason, nil)
}

func (c *AutomationController) triggerSwitch(ctx context.Context, reason string, expected *SwitchDecision) (result *manager.SwitchResult, resultErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if c.mgr == nil {
		return nil, errors.New("account manager cannot be nil")
	}
	if c.manualSwitch.Load() {
		return nil, ErrSwitchInProgress
	}
	defer func() {
		if resultErr != nil && !errors.Is(resultErr, ErrAutomationDisabled) && !errors.Is(resultErr, ErrSwitchInProgress) && !errors.Is(resultErr, ErrNoCandidates) {
			c.EmitStatus(StatusError, fmt.Sprintf("Tự động chuyển chưa hoàn tất: %v", resultErr), nil)
		}
	}()
	if !c.enabled.Load() {
		return nil, ErrAutomationDisabled
	}

	// Single-flight gate: only one switch execution allowed at a time
	if !c.isSwitching.CompareAndSwap(false, true) {
		return nil, ErrSwitchInProgress
	}
	c.syncMonitoring()
	defer func() { c.isSwitching.Store(false); c.syncMonitoring() }()

	c.EmitStatus(StatusSwitching, fmt.Sprintf("Kích hoạt chuyển tài khoản: %s", reason), nil)

	var switchResult *manager.SwitchResult

	_, err := c.gate.TryExecute(func(wasAbandoned bool) error {
		if !c.enabled.Load() || c.manualSwitch.Load() {
			return ErrAutomationDisabled
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		activeEmail, err := c.mgr.GetCurrentActiveEmail(ctx)
		if err != nil {
			return fmt.Errorf("failed retrieving active email: %w", err)
		}
		if activeEmail == "" {
			return errors.New("cannot identify active account")
		}
		if expected != nil && !strings.EqualFold(activeEmail, expected.ActiveEmail) {
			return nil
		}

		modelID, label, err := c.resolveModel(ctx)
		if err != nil {
			return err
		}
		category := ResolveActiveModelCategory(modelID, label)
		if expected != nil && category != expected.Evaluation.Category {
			var current *telemetry.AccountQuotaData
			if c.cache != nil {
				current, _ = c.cache.Get(activeEmail)
			}
			if !ClassifyQuotaState(current, category).ShouldSwitch {
				return nil
			}
		}

		if c.mgr.Store() == nil {
			return errors.New("account store is nil")
		}
		emails, err := c.mgr.Store().ListAccounts()
		if err != nil {
			return fmt.Errorf("failed listing accounts from store: %w", err)
		}

		failedMap := make(map[string]bool) // Exclusions live only for this operation.

		candidates := make([]CandidateAccount, 0, len(emails))
		for idx, email := range emails {
			if strings.TrimSpace(email) == "" {
				continue
			}
			token, _ := c.mgr.Store().GetAccount(email)
			tokenValid := token != nil && strings.TrimSpace(token.RefreshToken) != ""
			var refreshToken string
			if token != nil {
				refreshToken = token.RefreshToken
			}
			var qData *telemetry.AccountQuotaData
			if c.cache != nil {
				qData, _ = c.cache.Get(email)
			}
			cand := BuildCandidateAccount(
				email,
				idx,
				tokenValid,
				refreshToken,
				qData,
				category,
			)
			if c.mgr != nil {
				cand.Status = c.mgr.GetAccountStatus(email)
			}
			candidates = append(candidates, cand)
		}

		eligible, exclusions := FilterEligibleCandidates(candidates, activeEmail, failedMap)
		if len(eligible) == 0 {
			c.enabled.Store(false)
			c.syncMonitoring()
			c.EmitStatus(StatusNoCandidate, "Không có tài khoản để chuyển.", exclusions)
			return fmt.Errorf("%w: %+v", ErrNoCandidates, exclusions)
		}

		sorted := SortCandidatesBurnDown(eligible)

		for _, cand := range sorted {
			// Preflight check with PollAccount
			if c.pool != nil {
				freshQuota, pollErr := c.pool.PollAccount(ctx, cand.Email)
				if pollErr != nil || (freshQuota != nil && freshQuota.Status == telemetry.StatusError) {
					failedMap[cand.Email] = true
					c.EmitStatus(StatusEvaluating, fmt.Sprintf("Preflight kiểm tra tài khoản %s thất bại, bỏ qua trong đợt này", cand.Email), pollErr)
					continue
				}
				if freshQuota != nil {
					freshEval := ClassifyQuotaState(freshQuota, category)
					if !freshEval.IsEligible {
						failedMap[cand.Email] = true
						c.EmitStatus(StatusEvaluating, fmt.Sprintf("Quota tài khoản %s sau preflight không đủ điều kiện: %s", cand.Email, freshEval.Reason), freshEval)
						continue
					}
				}
			}

			// Preflight passed! Execute switch
			c.EmitStatus(StatusSwitching, fmt.Sprintf("Đang chuyển đổi sang %s...", cand.Email), cand)
			if !c.enabled.Load() || ctx.Err() != nil {
				return ErrAutomationDisabled
			}
			// Once credential mutation begins, OFF/stream cancellation must not
			// cut short restart, identity verification or continuation.
			res, sErr := c.mgr.SwitchAccount(context.WithoutCancel(ctx), cand.Email, false)
			if sErr != nil {
				failedMap[cand.Email] = true
				c.EmitStatus(StatusError, fmt.Sprintf("SwitchAccount sang %s thất bại: %v", cand.Email, sErr), sErr)
				return sErr
			}

			// Switch succeeded
			switchResult = res

			if c.retryCoord != nil {
				c.retryCoord.InvalidateSession()
				c.retryCoord.SetCurrentEmail(cand.Email)
			}

			c.EmitStatus(StatusSwitched, fmt.Sprintf("Đã tự động chuyển thành công sang %s", cand.Email), res)
			return nil
		}

		c.enabled.Store(false)
		c.syncMonitoring()
		c.EmitStatus(StatusNoCandidate, "Không có tài khoản để chuyển.", nil)
		return ErrNoCandidates
	})

	if err != nil && switchResult == nil {
		return nil, err
	}
	return switchResult, nil
}
