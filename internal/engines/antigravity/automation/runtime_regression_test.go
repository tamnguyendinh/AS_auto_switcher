package automation

import (
	"antigravity-switch/internal/engines/antigravity/connector"
	"antigravity-switch/internal/engines/antigravity/telemetry"
	"context"
	"errors"
	"testing"
	"time"
)

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
