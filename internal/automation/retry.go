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
	"antigravity-switch/internal/reloader"
)

const (
	// MaxRecoveryAttempts defines the maximum number of native in-memory retry attempts (3).
	MaxRecoveryAttempts = 3

	// DefaultRetryPacingDelay is the delay between limit error detection and dispatching empty-items retry (2s).
	DefaultRetryPacingDelay = 2 * time.Second
)

var (
	ErrCoordinatorStopped = errors.New("retry coordinator is stopped")
	ErrMaxRetriesExceeded = errors.New("maximum native retry attempts exceeded")
)

// RetryCoordinator orchestrates native in-memory 3-retry handling and agent state stream monitoring.
type RetryCoordinator struct {
	clientProvider  func(ctx context.Context) (*connector.Client, error)
	conversationsFn func(ctx context.Context) ([]string, error)
	onSwitchNeeded  func(ctx context.Context, reason string)
	onStatusEvent   func(code StatusCode, message string, details any)
	sleeper         func(ctx context.Context, d time.Duration) error

	mu               sync.Mutex
	recoveryAttempts int
	isRecovering     atomic.Bool
	seenEvents       map[string]bool
	activeStreams    map[string]context.CancelFunc
	activeModelName  string
	currentEmail     string

	ctx     context.Context
	cancel  context.CancelFunc
	running atomic.Bool
}

// NewRetryCoordinator creates a new RetryCoordinator.
func NewRetryCoordinator(
	clientProvider func(ctx context.Context) (*connector.Client, error),
	onSwitchNeeded func(ctx context.Context, reason string),
	onStatusEvent func(code StatusCode, message string, details any),
) *RetryCoordinator {
	rc := &RetryCoordinator{
		clientProvider: clientProvider,
		onSwitchNeeded:  onSwitchNeeded,
		onStatusEvent:   onStatusEvent,
		seenEvents:      make(map[string]bool),
		activeStreams:   make(map[string]context.CancelFunc),
	}
	rc.conversationsFn = rc.defaultScanConversations
	rc.sleeper = defaultContextSleep
	return rc
}

func (rc *RetryCoordinator) defaultScanConversations(ctx context.Context) ([]string, error) {
	threads, err := reloader.ScanActiveConversations2h("", "", time.Now())
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(threads))
	for _, th := range threads {
		if strings.TrimSpace(th.ThreadID) != "" {
			ids = append(ids, th.ThreadID)
		}
	}
	return ids, nil
}

func defaultContextSleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// SetConversationsScanner overrides the conversation discovery function (useful in tests).
func (rc *RetryCoordinator) SetConversationsScanner(fn func(ctx context.Context) ([]string, error)) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.conversationsFn = fn
}

// SetSleeper overrides the pacing sleep function (useful in tests).
func (rc *RetryCoordinator) SetSleeper(fn func(ctx context.Context, d time.Duration) error) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.sleeper = fn
}

// SetModelName configures the active model name to supply in native retry requests.
func (rc *RetryCoordinator) SetModelName(name string) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.activeModelName = name
}

// SetCurrentEmail updates the active email session. If email changes, stale events and retries are purged.
func (rc *RetryCoordinator) SetCurrentEmail(email string) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	if !strings.EqualFold(rc.currentEmail, email) {
		rc.currentEmail = email
		rc.seenEvents = make(map[string]bool)
		rc.recoveryAttempts = 0
		rc.isRecovering.Store(false)
	}
}

// Reset clears the recovery attempt counter, event deduplication map, and recovering state.
func (rc *RetryCoordinator) Reset() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.recoveryAttempts = 0
	rc.isRecovering.Store(false)
	rc.seenEvents = make(map[string]bool)
}

// GetRecoveryAttempts returns the current count of failed recovery actions.
func (rc *RetryCoordinator) GetRecoveryAttempts() int {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return rc.recoveryAttempts
}

// IsRecovering returns whether a recovery sequence is currently active.
func (rc *RetryCoordinator) IsRecovering() bool {
	return rc.isRecovering.Load()
}

// IsRunning returns whether the coordinator is currently active.
func (rc *RetryCoordinator) IsRunning() bool {
	return rc.running.Load()
}

// HandleAgentStateUpdate evaluates an update from StreamAgentStateUpdates and drives the recovery state machine.
func (rc *RetryCoordinator) HandleAgentStateUpdate(ctx context.Context, update connector.AgentStateUpdate) error {
	if !rc.running.Load() {
		return ErrCoordinatorStopped
	}

	rc.mu.Lock()
	// 1. Deduplication key
	eventKey := fmt.Sprintf("%s:%s:%s:%s:%d:%s",
		update.ConversationID,
		update.ExecutionID,
		update.StepID,
		update.Status,
		update.Attempt,
		update.TerminationReason,
	)

	if rc.seenEvents[eventKey] {
		rc.mu.Unlock()
		return nil // Drop duplicate/stale update
	}
	rc.seenEvents[eventKey] = true

	// 2. Check if language server is auto-retrying
	if update.HasRetryAt() {
		rc.mu.Unlock()
		if rc.onStatusEvent != nil {
			rc.onStatusEvent(StatusRetrying, fmt.Sprintf("Antigravity đang tự động retry (thời điểm: %s), đang chờ.", update.RetryAt), update)
		}
		return nil
	}

	// 3. Check for successful completion
	if update.IsSuccessfulCompletion() {
		wasRecovering := rc.isRecovering.Load() || rc.recoveryAttempts > 0
		rc.recoveryAttempts = 0
		rc.isRecovering.Store(false)
		rc.mu.Unlock()

		if wasRecovering && rc.onStatusEvent != nil {
			rc.onStatusEvent(StatusIdle, "Khôi phục thành công! Phiên hoạt động trở lại bình thường.", update)
		}
		return nil
	}

	// 4. Check for in-flight running
	if update.IsRunning() {
		rc.mu.Unlock()
		return nil
	}

	// 5. Check for Rate Limit or Quota Exhaustion
	if update.IsRateLimitOrQuotaExhausted() {
		rc.isRecovering.Store(true)

		if rc.recoveryAttempts >= MaxRecoveryAttempts {
			rc.mu.Unlock()
			if rc.onStatusEvent != nil {
				rc.onStatusEvent(StatusSwitching, "Đã thất bại 3 lần thử khôi phục với lỗi giới hạn. Kích hoạt chuyển tài khoản.", update)
			}
			if rc.onSwitchNeeded != nil {
				go rc.onSwitchNeeded(ctx, "3 native retries exhausted on rate limit/quota error")
			}
			return ErrMaxRetriesExceeded
		}

		rc.recoveryAttempts++
		attemptNum := rc.recoveryAttempts
		model := rc.activeModelName
		sleeper := rc.sleeper
		clientProvider := rc.clientProvider
		rc.mu.Unlock()

		if rc.onStatusEvent != nil {
			rc.onStatusEvent(StatusRetrying, fmt.Sprintf("Bắt lỗi giới hạn (429/QUOTA). Thực hiện Native Retry lần %d/%d...", attemptNum, MaxRecoveryAttempts), update)
		}

		// Dispatch native empty items retry asynchronously
		go func() {
			if sleeper != nil {
				_ = sleeper(ctx, DefaultRetryPacingDelay)
			}
			if !rc.running.Load() {
				return
			}
			if clientProvider == nil {
				return
			}
			client, err := clientProvider(ctx)
			if err != nil || client == nil {
				return
			}
			// Send empty items retry without waiting or counting HTTP ACK as recovery
			_ = client.SendEmptyItemsRetry(ctx, update.ConversationID, model, nil)
		}()

		return nil
	}

	rc.mu.Unlock()
	return nil
}

// SupersedeByProactive aborts any in-flight native recovery and triggers an immediate proactive switch.
func (rc *RetryCoordinator) SupersedeByProactive(ctx context.Context, reason string) {
	rc.mu.Lock()
	rc.isRecovering.Store(false)
	rc.recoveryAttempts = 0
	rc.mu.Unlock()

	if rc.onStatusEvent != nil {
		rc.onStatusEvent(StatusSwitching, fmt.Sprintf("Van chủ động kích hoạt (%s), dừng thử lại và chuyển ngay.", reason), nil)
	}
	if rc.onSwitchNeeded != nil {
		go rc.onSwitchNeeded(ctx, reason)
	}
}

// Start initiates monitoring on active conversations.
func (rc *RetryCoordinator) Start(ctx context.Context) error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if rc.running.Load() {
		return nil
	}

	if ctx == nil {
		ctx = context.Background()
	}
	rc.ctx, rc.cancel = context.WithCancel(ctx)
	rc.running.Store(true)

	if rc.conversationsFn != nil {
		convs, err := rc.conversationsFn(rc.ctx)
		if err == nil {
			for _, cid := range convs {
				rc.subscribeConversationLocked(rc.ctx, cid)
			}
		}
	}

	return nil
}

// Stop shuts down the coordinator and cancels all active stream subscriptions.
func (rc *RetryCoordinator) Stop() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if !rc.running.Load() {
		return
	}
	rc.running.Store(false)
	if rc.cancel != nil {
		rc.cancel()
	}
	for cid, cancelFn := range rc.activeStreams {
		cancelFn()
		delete(rc.activeStreams, cid)
	}
}

// RescanConversations rescans active conversations and connects any new streams.
func (rc *RetryCoordinator) RescanConversations(ctx context.Context) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if !rc.running.Load() {
		return
	}
	if rc.conversationsFn == nil {
		return
	}
	convs, err := rc.conversationsFn(ctx)
	if err != nil {
		return
	}
	for _, cid := range convs {
		rc.subscribeConversationLocked(rc.ctx, cid)
	}
}

func (rc *RetryCoordinator) subscribeConversationLocked(ctx context.Context, conversationID string) {
	if conversationID == "" {
		return
	}
	if _, exists := rc.activeStreams[conversationID]; exists {
		return
	}

	streamCtx, streamCancel := context.WithCancel(ctx)
	rc.activeStreams[conversationID] = streamCancel

	go func() {
		defer func() {
			rc.mu.Lock()
			delete(rc.activeStreams, conversationID)
			rc.mu.Unlock()
		}()

		if rc.clientProvider == nil {
			return
		}
		client, err := rc.clientProvider(streamCtx)
		if err != nil || client == nil {
			return
		}

		req := connector.StreamAgentStateUpdatesRequest{
			ConversationID:     conversationID,
			SubscriberID:       fmt.Sprintf("antigravity-switch-sub-%d", time.Now().UnixNano()),
			DisableRehydration: true,
			PageBounds:         10,
		}

		updateChan, errChan, err := client.StreamAgentStateUpdates(streamCtx, req)
		if err != nil {
			return
		}

		for {
			select {
			case update, ok := <-updateChan:
				if !ok {
					return
				}
				_ = rc.HandleAgentStateUpdate(streamCtx, update)
			case _, ok := <-errChan:
				if !ok {
					return
				}
				return
			case <-streamCtx.Done():
				return
			}
		}
	}()
}