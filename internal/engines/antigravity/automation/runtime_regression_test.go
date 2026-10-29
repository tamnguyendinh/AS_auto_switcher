package automation

import (
	"antigravity-switch/internal/engines/antigravity/connector"
	"antigravity-switch/internal/engines/antigravity/telemetry"
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// A terminal stream event may arrive before the retry RPC unwinds on the client.
// Hold the real send guard past cancellation to make that ordering deterministic.
func TestSwitchWaitsForCancelledRetryToReleaseGate(t *testing.T) {
	h := NewE2ETestHarness(t, "active@example.com", []string{"next@example.com"})
	defer h.Close()
	ctx := context.Background()
	h.Telemetry.SetAccountQuota("refresh-next@example.com", 0.9, 0.6, false, 0.9, 0.6, false)
	if _, err := h.Pool.PollAccount(ctx, "next@example.com"); err != nil {
		t.Fatal(err)
	}
	h.Controller.SetEnabled(true)
	if err := h.Controller.Start(ctx); err != nil {
		t.Fatal(err)
	}
	rc := h.Controller.RetryCoordinator()
	rc.mu.Lock()
	sendCtx, guard := rc.ctx, rc.sendGuard
	rc.mu.Unlock()
	held, cancelled, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	sendDone := make(chan error, 1)
	go func() {
		sendDone <- guard(sendCtx, func() error {
			close(held)
			<-sendCtx.Done()
			close(cancelled)
			<-release
			return sendCtx.Err()
		})
	}()
	select {
	case <-held:
	case <-time.After(3 * time.Second):
		t.Fatal("retry did not enter the shared gate")
	}
	switchDone := make(chan error, 1)
	go func() {
		result, err := h.Controller.TriggerSwitch(ctx, "retry outcome arrived before RPC completion")
		if err == nil && (result == nil || !result.Success) {
			err = errors.New("switch returned no successful result")
		}
		switchDone <- err
	}()
	select {
	case <-cancelled:
	case <-time.After(3 * time.Second):
		t.Fatal("switch did not cancel outstanding retry")
	}
	select {
	case err := <-switchDone:
		t.Fatalf("switch returned before cancelled retry released its gate: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	unblock()
	select {
	case err := <-switchDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("switch did not finish after retry released its gate")
	}
	if err := <-sendDone; !errors.Is(err, context.Canceled) {
		t.Fatalf("retry was not cancelled: %v", err)
	}
	if h.GetActiveEmail() != "next@example.com" {
		t.Fatal("candidate was not activated")
	}
}

func TestRetrySendQueuePreservesNextAttemptAndHonorsOff(t *testing.T) {
	for _, disable := range []bool{false, true} {
		name := "next-attempt"
		if disable {
			name = "off-cancels-queued-attempt"
		}
		t.Run(name, func(t *testing.T) {
			h := NewE2ETestHarness(t, "active@example.com", nil)
			defer h.Close()
			h.Controller.SetEnabled(true)
			if err := h.Controller.Start(context.Background()); err != nil {
				t.Fatal(err)
			}
			rc := h.Controller.RetryCoordinator()
			rc.mu.Lock()
			ctx, guard := rc.ctx, rc.sendGuard
			rc.mu.Unlock()
			held, release := make(chan struct{}), make(chan struct{})
			var once sync.Once
			unblock := func() { once.Do(func() { close(release) }) }
			defer unblock()
			firstDone, secondDone := make(chan error, 1), make(chan error, 1)
			go func() {
				firstDone <- guard(ctx, func() error { close(held); <-release; return nil })
			}()
			select {
			case <-held:
			case <-time.After(3 * time.Second):
				t.Fatal("first retry did not acquire gate")
			}
			sent := make(chan struct{}, 1)
			go func() {
				secondDone <- guard(ctx, func() error { sent <- struct{}{}; return nil })
			}()
			select {
			case err := <-secondDone:
				t.Fatalf("next attempt was dropped instead of waiting for the previous send: %v", err)
			case <-time.After(50 * time.Millisecond):
			}
			if disable {
				h.Controller.SetEnabled(false)
			}
			unblock()
			select {
			case err := <-secondDone:
				if disable {
					if !errors.Is(err, ErrAutomationDisabled) || len(sent) != 0 {
						t.Fatalf("OFF did not suppress queued retry: %v", err)
					}
				} else if err != nil || len(sent) != 1 {
					t.Fatalf("queued retry did not send exactly once: %v", err)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("queued retry did not complete")
			}
			if err := <-firstDone; err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestManualSwitchDrainsCancelledRetryBeforeGateEntry(t *testing.T) {
	h := NewE2ETestHarness(t, "active@example.com", nil)
	defer h.Close()
	h.Controller.SetEnabled(true)
	if err := h.Controller.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	rc := h.Controller.RetryCoordinator()
	rc.mu.Lock()
	ctx, guard := rc.ctx, rc.sendGuard
	rc.mu.Unlock()
	held, cancelled, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	sendDone := make(chan error, 1)
	go func() {
		sendDone <- guard(ctx, func() error {
			close(held)
			<-ctx.Done()
			close(cancelled)
			<-release
			return ctx.Err()
		})
	}()
	select {
	case <-held:
	case <-time.After(3 * time.Second):
		t.Fatal("retry did not enter gate")
	}
	manualReady := make(chan struct{})
	go func() { h.Controller.BeginManualSwitch(); close(manualReady) }()
	select {
	case <-cancelled:
	case <-time.After(3 * time.Second):
		t.Fatal("manual switch did not cancel retry")
	}
	select {
	case <-manualReady:
		t.Fatal("manual switch became ready while retry still owned the gate")
	case <-time.After(50 * time.Millisecond):
	}
	unblock()
	select {
	case <-manualReady:
	case <-time.After(3 * time.Second):
		t.Fatal("manual switch did not finish draining retry")
	}
	defer h.Controller.EndManualSwitch()
	if acquired, err := h.Gate.TryExecute(func(bool) error { return nil }); !acquired || err != nil {
		t.Fatalf("manual switch could not acquire released gate: %v", err)
	}
	if err := <-sendDone; !errors.Is(err, context.Canceled) {
		t.Fatalf("retry was not cancelled: %v", err)
	}
}

func TestQuotaCallbackCompletesEngineAfterEvaluationAndToggleOff(t *testing.T) {
	h := NewE2ETestHarness(t, "active@example.com", []string{"next@example.com"})
	defer h.Close()
	h.Telemetry.SetAccountQuota("refresh-active@example.com", 0.01, 0.5, false, 0.8, 0.5, false)
	h.Telemetry.SetAccountQuota("refresh-next@example.com", 0.9, 0.6, false, 0.9, 0.6, false)
	q, _ := h.Pool.PollAccount(context.Background(), "active@example.com")
	_, _ = h.Pool.PollAccount(context.Background(), "next@example.com")
	entered, release := make(chan struct{}), make(chan struct{})
	rel := h.Reloader
	rel.Sleeper = func(ctx context.Context, _ time.Duration) error { close(entered); <-release; return ctx.Err() }
	h.Controller.SetEnabled(true)
	h.Controller.OnTelemetryUpdate([]*telemetry.AccountQuotaData{q})
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("quota callback never reached engine")
	}
	h.Controller.SetEnabled(false)
	close(release)
	waitUntil(t, func() bool { return !h.Controller.IsSwitching() })
	switched := false
	for len(h.Controller.statusChan) > 0 {
		if (<-h.Controller.statusChan).Code == StatusSwitched {
			switched = true
		}
	}
	if !switched || h.GetActiveEmail() != "next@example.com" {
		t.Fatal("evaluation/OFF canceled active switch")
	}
}
func TestCandidatePreflightFailureExpiresWithOperation(t *testing.T) {
	h := NewE2ETestHarness(t, "active@example.com", []string{"next@example.com"})
	defer h.Close()
	ctx := context.Background()
	h.Telemetry.SetAccountQuota("refresh-next@example.com", 0.9, 0.6, false, 0.9, 0.6, false)
	_, _ = h.Pool.PollAccount(ctx, "next@example.com")
	h.Controller.SetEnabled(true)
	h.Telemetry.SetAccountError("refresh-next@example.com", 503)
	_, err := h.Controller.TriggerSwitch(ctx, "first round")
	if !errors.Is(err, ErrNoCandidates) {
		t.Fatalf("expected preflight rejection: %v", err)
	}
	if h.Controller.IsEnabled() {
		t.Fatalf("expected controller to auto-stop when all candidates fail preflight")
	}
	h.Telemetry.SetAccountQuota("refresh-next@example.com", 0.9, 0.6, false, 0.9, 0.6, false)
	_, _ = h.Pool.PollAccount(ctx, "next@example.com")
	h.Controller.SetEnabled(true)
	res, err := h.Controller.TriggerSwitch(ctx, "new telemetry round")
	if err != nil || res == nil || !res.Success {
		t.Fatalf("temporary failure permanently excluded candidate: %v", err)
	}
}
func TestControllerOffDoesNotSubscribeOrRetry(t *testing.T) {
	h := NewE2ETestHarness(t, "active@example.com", nil)
	defer h.Close()
	_ = h.Controller.Start(context.Background())
	err := h.Controller.RetryCoordinator().HandleAgentStateUpdate(context.Background(), quotaFailure("conv-e2e-1", 0))
	if !errors.Is(err, ErrCoordinatorStopped) || h.LangServer.GetRetryCalls() != 0 {
		t.Fatal("OFF accepted recovery")
	}
}
func TestNativeStreamThroughCoordinatorAndRescan(t *testing.T) {
	h := NewE2ETestHarness(t, "active@example.com", nil)
	defer h.Close()
	ctx := context.Background()
	h.Controller.SetEnabled(true)
	_ = h.Controller.Start(ctx)
	waitUntil(t, func() bool {
		h.LangServer.mu.Lock()
		defer h.LangServer.mu.Unlock()
		return len(h.LangServer.streamClients) == 1
	})
	h.LangServer.SendStreamUpdate(quotaFailure("conv-e2e-1", 0))
	waitUntil(t, func() bool { return h.LangServer.GetRetryCalls() == 1 })
	h.LangServer.SendStreamUpdate(connector.AgentStateUpdate{ConversationID: "conv-e2e-1", ExecutionID: "exec-ok", Status: "IDLE", TerminationReason: "NO_TOOL_CALL"})
	waitUntil(t, func() bool { return !h.Controller.RetryCoordinator().IsRecovering() })
	// EOF and then a new conversation are picked up by the existing rescan entrypoint.
	h.LangServer.mu.Lock()
	for ch := range h.LangServer.streamClients {
		close(ch)
		delete(h.LangServer.streamClients, ch)
	}
	h.LangServer.mu.Unlock()
	waitUntil(t, func() bool {
		rc := h.Controller.RetryCoordinator()
		rc.mu.Lock()
		defer rc.mu.Unlock()
		return len(rc.activeStreams) == 0
	})
	h.Controller.RetryCoordinator().SetConversationsScanner(func(context.Context) ([]string, error) { return []string{"conv-new"}, nil })
	h.Controller.RetryCoordinator().RescanConversations(ctx)
	waitUntil(t, func() bool {
		h.LangServer.mu.Lock()
		defer h.LangServer.mu.Unlock()
		return len(h.LangServer.streamClients) == 1
	})
	h.LangServer.SendStreamUpdate(quotaFailure("conv-new", 0))
	waitUntil(t, func() bool { return h.LangServer.GetRetryCalls() == 2 })
}
