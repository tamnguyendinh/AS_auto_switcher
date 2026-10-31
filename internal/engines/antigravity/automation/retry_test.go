package automation

import (
	"as-auto-switcher/internal/engines/antigravity/connector"
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func retryFixture(t *testing.T) (*RetryCoordinator, *MockLangServer, *atomic.Int32) {
	t.Helper()
	ls := NewMockLangServer(t)
	switches := &atomic.Int32{}
	rc := NewRetryCoordinator(func(context.Context) (*connector.Client, error) { return ls.NewClient(), nil }, func(context.Context, string) { switches.Add(1) }, nil)
	rc.SetConversationsScanner(func(context.Context) ([]string, error) { return nil, nil })
	rc.SetModelName("gemini-2.5-pro")
	if err := rc.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rc.Stop(); ls.Close() })
	return rc, ls, switches
}
func quotaFailure(conv string, n int) connector.AgentStateUpdate {
	return connector.AgentStateUpdate{ConversationID: conv, ExecutionID: fmt.Sprintf("exec-%d", n), StepID: fmt.Sprint(n), Status: "IDLE", TerminationReason: "ERROR", LastStepError: &connector.StepError{Code: "RESOURCE_EXHAUSTED"}}
}
func TestRetryCoordinator_ThreeActionsWaitForExecution(t *testing.T) {
	rc, ls, switches := retryFixture(t)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if err := rc.HandleAgentStateUpdate(ctx, quotaFailure("a", i)); err != nil {
			t.Fatal(err)
		}
		waitUntil(t, func() bool { return ls.GetRetryCalls() == i+1 })
		if switches.Load() != 0 {
			t.Fatal("HTTP ACK counted as failed outcome")
		}
		_ = rc.HandleAgentStateUpdate(ctx, quotaFailure("a", i))
		if rc.GetRecoveryAttempts() != i+1 {
			t.Fatal("duplicate consumed retry budget")
		}
	}
	if err := rc.HandleAgentStateUpdate(ctx, quotaFailure("a", 3)); !errors.Is(err, ErrMaxRetriesExceeded) {
		t.Fatal(err)
	}
	waitUntil(t, func() bool { return switches.Load() == 1 })
	_ = rc.HandleAgentStateUpdate(ctx, quotaFailure("a", 3))
	if ls.GetRetryCalls() != 3 || switches.Load() != 1 {
		t.Fatal("duplicate recovery or switch")
	}
}
func TestRetryCoordinator_ConversationOutcomesAreIsolated(t *testing.T) {
	rc, ls, _ := retryFixture(t)
	ctx := context.Background()
	_ = rc.HandleAgentStateUpdate(ctx, quotaFailure("a", 0))
	waitUntil(t, func() bool { return ls.GetRetryCalls() == 1 })
	_ = rc.HandleAgentStateUpdate(ctx, connector.AgentStateUpdate{ConversationID: "b", ExecutionID: "b-1", Status: "IDLE", TerminationReason: "NO_TOOL_CALL"})
	if rc.GetRecoveryAttempts() != 1 {
		t.Fatal("another conversation reset recovery")
	}
	_ = rc.HandleAgentStateUpdate(ctx, connector.AgentStateUpdate{ConversationID: "a", ExecutionID: "exec-1", Status: "IDLE", TerminationReason: "NO_TOOL_CALL"})
	if rc.IsRecovering() {
		t.Fatal("successful execution did not stop recovery")
	}
}
func TestRetryCoordinator_StoppedSupersededAndSessionChangedCancelPendingSend(t *testing.T) {
	for _, action := range []string{"stop", "supersede", "session"} {
		t.Run(action, func(t *testing.T) {
			rc, ls, _ := retryFixture(t)
			entered, finished := make(chan struct{}), make(chan struct{})
			rc.SetSleeper(func(ctx context.Context, _ time.Duration) error {
				close(entered)
				<-ctx.Done()
				close(finished)
				return ctx.Err()
			})
			_ = rc.HandleAgentStateUpdate(context.Background(), quotaFailure("a", 0))
			<-entered
			switch action {
			case "stop":
				rc.Stop()
			case "supersede":
				rc.SupersedeByProactive(context.Background(), "quota")
			case "session":
				rc.InvalidateSession()
			}
			<-finished
			if rc.IsRecovering() || ls.GetRetryCalls() != 0 {
				t.Fatal("invalidated worker sent Retry")
			}
		})
	}
}
func TestRetryCoordinator_NativeRetryAndNonQuotaErrors(t *testing.T) {
	rc, ls, switches := retryFixture(t)
	u := quotaFailure("a", 0)
	u.Status = "RUNNING"
	u.RetryAt = time.Now().Add(time.Minute).UTC().Format(time.RFC3339Nano)
	u.Attempt = 1
	u.MaxAttempts = 3
	_ = rc.HandleAgentStateUpdate(context.Background(), u)
	u.Status = "IDLE"
	u.RetryAt = ""
	u.LastStepError = &connector.StepError{Code: "INTERNAL"}
	_ = rc.HandleAgentStateUpdate(context.Background(), u)
	if rc.IsRecovering() || ls.GetRetryCalls() != 0 || switches.Load() != 0 {
		t.Fatal("native wait or non-quota error initiated recovery")
	}
}
func TestRetryCoordinator_SendFailureDoesNotBecomeQuotaFailure(t *testing.T) {
	statuses := make(chan StatusCode, 10)
	rc := NewRetryCoordinator(func(context.Context) (*connector.Client, error) { return nil, errors.New("offline") }, func(context.Context, string) { t.Error("transport error caused switch") }, func(s StatusCode, _ string, _ any) { statuses <- s })
	rc.SetConversationsScanner(func(context.Context) ([]string, error) { return nil, nil })
	_ = rc.Start(context.Background())
	defer rc.Stop()
	_ = rc.HandleAgentStateUpdate(context.Background(), quotaFailure("a", 0))
	waitUntil(t, func() bool { return !rc.IsRecovering() })
	found := false
	for len(statuses) > 0 {
		if <-statuses == StatusError {
			found = true
		}
	}
	if !found {
		t.Fatal("send failure not visible")
	}
}
