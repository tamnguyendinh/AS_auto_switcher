package automation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"as-auto-switcher/internal/engines/antigravity/connector"
	"as-auto-switcher/internal/engines/antigravity/reloader"
)

const MaxRecoveryAttempts = 3

var (
	ErrCoordinatorStopped = errors.New("retry coordinator is stopped")
	ErrMaxRetriesExceeded = errors.New("maximum native retry attempts exceeded")
)

type recovery struct {
	attempts  int
	execution string
	step      string
	started   bool
	awaiting  bool
	switching bool
	cancel    context.CancelFunc
	ctx       context.Context
}
type subscription struct {
	cancel     context.CancelFunc
	generation uint64
}

// RetryCoordinator owns recovery per conversation. A completion on another
// conversation cannot consume or reset this conversation's three actions.
type RetryCoordinator struct {
	clientProvider  func(context.Context) (*connector.Client, error)
	conversationsFn func(context.Context) ([]string, error)
	onSwitchNeeded  func(context.Context, string)
	onStatusEvent   func(StatusCode, string, any)
	sendGuard       func(context.Context, func() error) error
	sleeper         func(context.Context, time.Duration) error
	mu              sync.Mutex
	recoveries      map[string]*recovery
	seenEvents      map[string]bool
	activeStreams   map[string]*subscription
	activeModelName string
	currentEmail    string
	generation      uint64
	skipInitial     bool
	ctx             context.Context
	cancel          context.CancelFunc
	running         atomic.Bool
}

func NewRetryCoordinator(provider func(context.Context) (*connector.Client, error), onSwitch func(context.Context, string), onStatus func(StatusCode, string, any)) *RetryCoordinator {
	rc := &RetryCoordinator{clientProvider: provider, onSwitchNeeded: onSwitch, onStatusEvent: onStatus,
		recoveries: make(map[string]*recovery), seenEvents: make(map[string]bool), activeStreams: make(map[string]*subscription)}
	rc.conversationsFn = rc.defaultScanConversations
	return rc
}
func (rc *RetryCoordinator) defaultScanConversations(ctx context.Context) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	threads, err := reloader.ScanActiveConversations("", "", time.Now())
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(threads))
	for _, t := range threads {
		if t.ThreadID != "" {
			ids = append(ids, t.ThreadID)
		}
	}
	return ids, nil
}
func (rc *RetryCoordinator) emit(code StatusCode, message string) {
	if rc.onStatusEvent != nil {
		rc.onStatusEvent(code, message, nil)
	}
}
func (rc *RetryCoordinator) SetConversationsScanner(fn func(context.Context) ([]string, error)) {
	rc.mu.Lock()
	rc.conversationsFn = fn
	rc.mu.Unlock()
}

// SetSleeper is an optional scheduling seam for tests; native recovery has no fixed delay.
func (rc *RetryCoordinator) SetSleeper(fn func(context.Context, time.Duration) error) {
	rc.mu.Lock()
	rc.sleeper = fn
	rc.mu.Unlock()
}
func (rc *RetryCoordinator) SetSendGuard(fn func(context.Context, func() error) error) {
	rc.mu.Lock()
	rc.sendGuard = fn
	rc.mu.Unlock()
}
func (rc *RetryCoordinator) SetModelName(name string) {
	rc.mu.Lock()
	rc.activeModelName = name
	rc.mu.Unlock()
}
func (rc *RetryCoordinator) SetCurrentEmail(email string) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	if !strings.EqualFold(rc.currentEmail, email) {
		if rc.currentEmail != "" {
			rc.invalidateLocked()
		}
		rc.currentEmail = email
	}
}
func (rc *RetryCoordinator) clearRecoveryLocked() {
	for _, r := range rc.recoveries {
		r.cancel()
	}
	rc.recoveries = make(map[string]*recovery)
}
func (rc *RetryCoordinator) Reset() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.clearRecoveryLocked()
	rc.seenEvents = make(map[string]bool)
}
func (rc *RetryCoordinator) invalidateLocked() {
	rc.generation++
	rc.clearRecoveryLocked()
	rc.seenEvents = make(map[string]bool)
	rc.skipInitial = true
	for cid, sub := range rc.activeStreams {
		sub.cancel()
		delete(rc.activeStreams, cid)
	}
}

// InvalidateSession discards old workers and treats rehydrated terminal states
// after a credential switch as history, not as a new failed request.
func (rc *RetryCoordinator) InvalidateSession() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.invalidateLocked()
}
func (rc *RetryCoordinator) GetRecoveryAttempts() int {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	n := 0
	for _, r := range rc.recoveries {
		if r.attempts > n {
			n = r.attempts
		}
	}
	return n
}
func (rc *RetryCoordinator) IsRecovering() bool {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return len(rc.recoveries) > 0
}
func (rc *RetryCoordinator) IsRunning() bool { return rc.running.Load() }
func eventKey(u connector.AgentStateUpdate) string {
	return fmt.Sprintf("%s:%s:%s:%s", u.ConversationID, u.ExecutionID, u.StepID, u.TerminationReason)
}

func (rc *RetryCoordinator) HandleAgentStateUpdate(ctx context.Context, u connector.AgentStateUpdate) error {
	rc.mu.Lock()
	if !rc.running.Load() || ctx.Err() != nil {
		rc.mu.Unlock()
		return ErrCoordinatorStopped
	}
	if strings.TrimSpace(u.ConversationID) == "" {
		rc.mu.Unlock()
		return errors.New("agent update has no conversation ID")
	}
	r := rc.recoveries[u.ConversationID]
	if u.IsRunning() || u.HasRetryAt() {
		if r != nil && r.awaiting {
			r.started = true
		}
		rc.mu.Unlock()
		return nil
	}
	if !u.IsTerminalError() && !u.IsSuccessfulCompletion() {
		rc.mu.Unlock()
		return nil
	}
	key := eventKey(u)
	if rc.seenEvents[key] {
		rc.mu.Unlock()
		return nil
	}
	// Require an actual execution transition after the previous Retry send.
	if r != nil && r.awaiting && !r.started && u.ExecutionID == r.execution && u.StepID == r.step {
		rc.mu.Unlock()
		return nil
	}
	rc.seenEvents[key] = true
	if u.IsSuccessfulCompletion() || !u.IsRateLimitOrQuotaExhausted() {
		if r != nil {
			r.cancel()
			delete(rc.recoveries, u.ConversationID)
		}
		rc.mu.Unlock()
		if r != nil {
			if u.IsSuccessfulCompletion() {
				rc.emit(StatusIdle, "Khôi phục thành công; tiếp tục theo dõi.")
			} else {
				rc.emit(StatusError, "Phiên dừng với lỗi không thuộc quota; ngừng tự thử lại.")
			}
		}
		return nil
	}
	if r == nil {
		workCtx, cancel := context.WithCancel(rc.ctx)
		r = &recovery{ctx: workCtx, cancel: cancel}
		rc.recoveries[u.ConversationID] = r
	}
	if r.switching {
		rc.mu.Unlock()
		return nil
	}
	if r.attempts >= MaxRecoveryAttempts {
		r.switching = true
		generation := rc.generation
		rc.mu.Unlock()
		rc.emit(StatusSwitching, "Ba lần khôi phục vẫn gặp lỗi giới hạn; chuyển tài khoản.")
		go func() {
			rc.mu.Lock()
			valid := rc.running.Load() && rc.generation == generation && rc.recoveries[u.ConversationID] == r
			rc.mu.Unlock()
			if valid && rc.onSwitchNeeded != nil {
				rc.onSwitchNeeded(context.Background(), "3 native retries exhausted on rate limit/quota error")
			}
		}()
		return ErrMaxRetriesExceeded
	}
	r.attempts++
	r.execution, r.step, r.started, r.awaiting = u.ExecutionID, u.StepID, false, true
	attempt, generation, model := r.attempts, rc.generation, rc.activeModelName
	provider, guard, before := rc.clientProvider, rc.sendGuard, rc.sleeper
	rc.mu.Unlock()
	rc.emit(StatusRetrying, fmt.Sprintf("Khôi phục bằng Native Retry lần %d/%d...", attempt, MaxRecoveryAttempts))
	go func() {
		var err error
		if before != nil {
			err = before(r.ctx, 0)
		}
		var client *connector.Client
		if err == nil {
			if provider == nil {
				err = errors.New("retry client provider unavailable")
			} else {
				client, err = provider(r.ctx)
			}
		}
		send := func() error {
			rc.mu.Lock()
			valid := rc.running.Load() && rc.generation == generation && rc.recoveries[u.ConversationID] == r && r.attempts == attempt
			rc.mu.Unlock()
			if !valid || r.ctx.Err() != nil {
				return context.Canceled
			}
			if client == nil {
				return errors.New("retry language server unavailable")
			}
			return client.SendEmptyItemsRetry(r.ctx, u.ConversationID, model, nil)
		}
		if err == nil {
			if guard != nil {
				err = guard(r.ctx, send)
			} else {
				err = send()
			}
		}
		if err != nil {
			rc.mu.Lock()
			valid := rc.generation == generation && rc.recoveries[u.ConversationID] == r && r.attempts == attempt
			if valid {
				r.cancel()
				delete(rc.recoveries, u.ConversationID)
				delete(rc.seenEvents, key)
			}
			rc.mu.Unlock()
			if valid && !errors.Is(err, context.Canceled) && !errors.Is(err, ErrAutomationDisabled) && !errors.Is(err, ErrGateBusy) {
				rc.emit(StatusError, fmt.Sprintf("Không gửi được Native Retry: %v", err))
			}
		}
	}()
	return nil
}

func (rc *RetryCoordinator) SupersedeByProactive(ctx context.Context, reason string) {
	rc.Reset()
	if rc.onSwitchNeeded != nil {
		rc.onSwitchNeeded(ctx, reason)
	}
}
func (rc *RetryCoordinator) Start(ctx context.Context) error {
	rc.mu.Lock()
	if rc.running.Load() {
		rc.mu.Unlock()
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	rc.ctx, rc.cancel = context.WithCancel(ctx)
	rc.running.Store(true)
	rc.mu.Unlock()
	rc.RescanConversations(ctx)
	return nil
}
func (rc *RetryCoordinator) Stop() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.running.Store(false)
	rc.generation++
	if rc.cancel != nil {
		rc.cancel()
	}
	rc.clearRecoveryLocked()
	for cid, sub := range rc.activeStreams {
		sub.cancel()
		delete(rc.activeStreams, cid)
	}
}

// The existing telemetry callback supplies the reconnect/rescan cadence.
func (rc *RetryCoordinator) RescanConversations(ctx context.Context) {
	rc.mu.Lock()
	scan, generation, running := rc.conversationsFn, rc.generation, rc.running.Load()
	rc.mu.Unlock()
	if !running || scan == nil || ctx.Err() != nil {
		return
	}
	ids, err := scan(ctx)
	if err != nil {
		rc.emit(StatusError, fmt.Sprintf("Không đọc được danh sách phiên để theo dõi: %v", err))
		return
	}
	rc.mu.Lock()
	defer rc.mu.Unlock()
	if !rc.running.Load() || rc.generation != generation {
		return
	}
	present := make(map[string]bool)
	for _, id := range ids {
		present[id] = true
		rc.subscribeLocked(id)
	}
	for id, sub := range rc.activeStreams {
		if !present[id] {
			sub.cancel()
			delete(rc.activeStreams, id)
		}
	}
}
func (rc *RetryCoordinator) subscribeLocked(cid string) {
	if cid == "" || rc.activeStreams[cid] != nil {
		return
	}
	ctx, cancel := context.WithCancel(rc.ctx)
	sub := &subscription{cancel: cancel, generation: rc.generation}
	rc.activeStreams[cid] = sub
	provider, skipInitial := rc.clientProvider, rc.skipInitial
	go func() {
		defer cancel()
		defer func() {
			rc.mu.Lock()
			if rc.activeStreams[cid] == sub {
				delete(rc.activeStreams, cid)
			}
			rc.mu.Unlock()
		}()
		if provider == nil {
			return
		}
		client, err := provider(ctx)
		if err != nil || client == nil {
			if ctx.Err() == nil {
				rc.emit(StatusError, "Chưa kết nối được Language Server; sẽ nối lại ở lượt quota tiếp theo.")
			}
			return
		}
		updates, errs, err := client.StreamAgentStateUpdates(ctx, connector.StreamAgentStateUpdatesRequest{ConversationID: cid, SubscriberID: fmt.Sprintf("switcher-%d", time.Now().UnixNano()), DisableRehydration: true, PageBounds: 1})
		if err != nil {
			if ctx.Err() == nil {
				rc.emit(StatusError, fmt.Sprintf("Không mở được agent stream: %v", err))
			}
			return
		}
		first := true
		for updates != nil || errs != nil {
			select {
			case u, ok := <-updates:
				if !ok {
					updates = nil
					continue
				}
				rc.mu.Lock()
				valid := rc.running.Load() && rc.generation == sub.generation && rc.activeStreams[cid] == sub
				if valid && first && skipInitial && !u.IsRunning() {
					rc.seenEvents[eventKey(u)] = true
				}
				rc.mu.Unlock()
				first = false
				if !valid {
					return
				}
				if u.ConversationID != cid {
					rc.emit(StatusError, "Agent stream trả sai conversation ID.")
					return
				}
				_ = rc.HandleAgentStateUpdate(ctx, u)
			case err, ok := <-errs:
				if !ok {
					errs = nil
					continue
				}
				if err != nil && ctx.Err() == nil {
					rc.emit(StatusError, fmt.Sprintf("Agent stream bị ngắt: %v", err))
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}
