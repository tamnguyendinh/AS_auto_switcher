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
	ShouldSwitch bool                  `json:"should_switch"`
	Reason       string                `json:"reason"`
	ActiveEmail  string                `json:"active_email"`
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
	failedInRound  map[string]bool
	failedMu       sync.Mutex

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
		mgr:           mgr,
		pool:          pool,
		cache:         cache,
		gate:          gate,
		statusChan:    make(chan StatusEvent, 64),
		failedInRound: make(map[string]bool),
		pbtxtPath:     "",
	}
	c.clientFinder = c.defaultFindClient

	// Wire RetryCoordinator hooked into TriggerSwitch and EmitStatus
	c.retryCoord = NewRetryCoordinator(
		func(ctx context.Context) (*connector.Client, error) {
			return c.findClient(ctx)
		},
		func(ctx context.Context, reason string) {
			_, _ = c.TriggerSwitch(ctx, reason)
		},
		func(code StatusCode, message string, details any) {
			c.EmitStatus(code, message, details)
		},
	)

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
	if !was && enabled {
		c.EmitStatus(StatusIdle, "Tự động chuyển tài khoản: BẬT", nil)
		if c.cache != nil {
			quotas := make([]*telemetry.AccountQuotaData, 0)
			for _, q := range c.cache.GetAll() {
				quotas = append(quotas, q)
			}
			c.OnTelemetryUpdate(quotas)
		}
	} else if was && !enabled {
		c.EmitStatus(StatusIdle, "Tự động chuyển tài khoản: TẮT", nil)
		if c.retryCoord != nil {
			c.retryCoord.Reset()
		}
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
	defer c.mu.Unlock()

	if c.running.Load() {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	c.ctx, c.cancel = context.WithCancel(ctx)
	c.running.Store(true)

	if c.retryCoord != nil {
		_ = c.retryCoord.Start(c.ctx)
	}
	return nil
}

// Stop terminates the controller, cancels background contexts, and stops the retry coordinator.
func (c *AutomationController) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running.Load() {
		return
	}
	c.running.Store(false)
	if c.cancel != nil {
		c.cancel()
	}
	if c.retryCoord != nil {
		c.retryCoord.Stop()
	}
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
	if !c.enabled.Load() || c.isSwitching.Load() {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		decision, err := c.EvaluateProactiveTrigger(ctx, quotas)
		if err == nil && decision != nil && decision.ShouldSwitch {
			if c.retryCoord != nil {
				c.retryCoord.SupersedeByProactive(ctx, decision.Reason)
			} else {
				_, _ = c.TriggerSwitch(ctx, decision.Reason)
			}
		}
	}()
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

	modelID := connector.DefaultActiveAgentModel
	label := connector.DefaultActiveAgentModel
	client, err := c.findClient(ctx)
	if err == nil && client != nil {
		mID, mLbl, _, rErr := client.ResolveActiveModel(ctx, c.pbtxtPath)
		if rErr == nil && mID != "" {
			modelID = mID
			label = mLbl
		}
	}

	c.mu.Lock()
	c.activeModelID = modelID
	c.activeModelLbl = label
	if c.retryCoord != nil {
		c.retryCoord.SetModelName(modelID)
		c.retryCoord.SetCurrentEmail(activeEmail)
	}
	c.mu.Unlock()

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
func (c *AutomationController) TriggerSwitch(ctx context.Context, reason string) (*manager.SwitchResult, error) {
	if !c.enabled.Load() {
		return nil, ErrAutomationDisabled
	}

	// Single-flight gate: only one switch execution allowed at a time
	if !c.isSwitching.CompareAndSwap(false, true) {
		return nil, ErrSwitchInProgress
	}
	defer c.isSwitching.Store(false)

	c.EmitStatus(StatusSwitching, fmt.Sprintf("Kích hoạt chuyển tài khoản: %s", reason), nil)

	var switchResult *manager.SwitchResult

	err := c.gate.Execute(ctx, func(wasAbandoned bool) error {
		activeEmail, err := c.mgr.GetCurrentActiveEmail(ctx)
		if err != nil {
			return fmt.Errorf("failed retrieving active email: %w", err)
		}

		modelID := connector.DefaultActiveAgentModel
		label := connector.DefaultActiveAgentModel
		client, err := c.findClient(ctx)
		if err == nil && client != nil {
			mID, mLbl, _, rErr := client.ResolveActiveModel(ctx, c.pbtxtPath)
			if rErr == nil && mID != "" {
				modelID = mID
				label = mLbl
			}
		}
		category := ResolveActiveModelCategory(modelID, label)

		if c.mgr.Store() == nil {
			return errors.New("account store is nil")
		}
		emails, err := c.mgr.Store().ListAccounts()
		if err != nil {
			return fmt.Errorf("failed listing accounts from store: %w", err)
		}

		c.failedMu.Lock()
		failedMap := make(map[string]bool)
		for k, v := range c.failedInRound {
			failedMap[k] = v
		}
		c.failedMu.Unlock()

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
			candidates = append(candidates, BuildCandidateAccount(
				email,
				idx,
				tokenValid,
				refreshToken,
				qData,
				category,
			))
		}

		eligible, exclusions := FilterEligibleCandidates(candidates, activeEmail, failedMap)
		if len(eligible) == 0 {
			c.EmitStatus(StatusNoCandidate, "Không có ứng viên nào đủ điều kiện chuyển", exclusions)
			return fmt.Errorf("%w: %+v", ErrNoCandidates, exclusions)
		}

		sorted := SortCandidatesBurnDown(eligible)

		var lastSwitchErr error
		for _, cand := range sorted {
			// Preflight check with PollAccount
			if c.pool != nil {
				freshQuota, pollErr := c.pool.PollAccount(ctx, cand.Email)
				if pollErr != nil || (freshQuota != nil && freshQuota.Status == telemetry.StatusError) {
					c.failedMu.Lock()
					c.failedInRound[cand.Email] = true
					c.failedMu.Unlock()
					c.EmitStatus(StatusEvaluating, fmt.Sprintf("Preflight kiểm tra tài khoản %s thất bại, bỏ qua trong đợt này", cand.Email), pollErr)
					continue
				}
				if freshQuota != nil {
					freshEval := ClassifyQuotaState(freshQuota, category)
					if !freshEval.IsEligible {
						c.failedMu.Lock()
						c.failedInRound[cand.Email] = true
						c.failedMu.Unlock()
						c.EmitStatus(StatusEvaluating, fmt.Sprintf("Quota tài khoản %s sau preflight không đủ điều kiện: %s", cand.Email, freshEval.Reason), freshEval)
						continue
					}
				}
			}

			// Preflight passed! Execute switch
			c.EmitStatus(StatusSwitching, fmt.Sprintf("Đang chuyển đổi sang %s...", cand.Email), cand)
			res, sErr := c.mgr.SwitchAccount(ctx, cand.Email, false)
			if sErr != nil {
				lastSwitchErr = sErr
				c.failedMu.Lock()
				c.failedInRound[cand.Email] = true
				c.failedMu.Unlock()
				c.EmitStatus(StatusError, fmt.Sprintf("SwitchAccount sang %s thất bại: %v", cand.Email, sErr), sErr)
				return sErr
			}

			// Switch succeeded
			switchResult = res
			c.failedMu.Lock()
			c.failedInRound = make(map[string]bool)
			c.failedMu.Unlock()

			if c.retryCoord != nil {
				c.retryCoord.Reset()
				c.retryCoord.SetCurrentEmail(cand.Email)
				c.retryCoord.RescanConversations(ctx)
			}

			c.EmitStatus(StatusSwitched, fmt.Sprintf("Đã tự động chuyển thành công sang %s", cand.Email), res)
			return nil
		}

		if lastSwitchErr != nil {
			return lastSwitchErr
		}
		c.EmitStatus(StatusNoCandidate, "Tất cả ứng viên đã thất bại preflight trong đợt này", nil)
		return ErrNoCandidates
	})

	if err != nil && switchResult == nil {
		return nil, err
	}
	return switchResult, nil
}