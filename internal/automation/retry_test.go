package automation

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"antigravity-switch/internal/connector"
)

func TestRetryCoordinator_HandleAgentStateUpdate_Deduplication(t *testing.T) {
	var retryCount atomic.Int32
	clientProvider := func(ctx context.Context) (*connector.Client, error) {
		retryCount.Add(1)
		return nil, nil
	}

	rc := NewRetryCoordinator(clientProvider, nil, nil)
	rc.SetSleeper(func(ctx context.Context, d time.Duration) error {
		return nil // Instant sleep for tests
	})

	_ = rc.Start(context.Background())
	defer rc.Stop()

	update := connector.AgentStateUpdate{
		ConversationID:    "conv-dup-1",
		ExecutionID:       "exec-1",
		StepID:            "step-1",
		Status:            "IDLE",
		Attempt:           1,
		TerminationReason: "ERROR",
		LastStepError: &connector.StepError{
			Code:    "RESOURCE_EXHAUSTED",
			Message: "429 Rate limit exceeded",
		},
	}

	// First delivery: processes error and increments attempts
	err := rc.HandleAgentStateUpdate(context.Background(), update)
	if err != nil {
		t.Fatalf("first update failed: %v", err)
	}

	if rc.GetRecoveryAttempts() != 1 {
		t.Errorf("expected recoveryAttempts=1, got %d", rc.GetRecoveryAttempts())
	}

	// Second delivery (duplicate): should be dropped
	err = rc.HandleAgentStateUpdate(context.Background(), update)
	if err != nil {
		t.Fatalf("second update failed: %v", err)
	}

	if rc.GetRecoveryAttempts() != 1 {
		t.Errorf("expected recoveryAttempts to remain 1 after duplicate, got %d", rc.GetRecoveryAttempts())
	}
}

func TestRetryCoordinator_ServerAutoRetry_RetryAt(t *testing.T) {
	rc := NewRetryCoordinator(nil, nil, nil)
	_ = rc.Start(context.Background())
	defer rc.Stop()

	update := connector.AgentStateUpdate{
		ConversationID: "conv-server-retry",
		Status:         "PAUSED",
		RetryAt:        "2026-10-20T15:00:00Z",
	}

	err := rc.HandleAgentStateUpdate(context.Background(), update)
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	if rc.GetRecoveryAttempts() != 0 {
		t.Errorf("server auto-retry should not increment attempts, got %d", rc.GetRecoveryAttempts())
	}
	if rc.IsRecovering() {
		t.Errorf("expected isRecovering=false")
	}
}

func TestRetryCoordinator_SuccessfulOutcome_ResetsCounter(t *testing.T) {
	rc := NewRetryCoordinator(nil, nil, nil)
	rc.SetSleeper(func(ctx context.Context, d time.Duration) error {
		return nil
	})
	_ = rc.Start(context.Background())
	defer rc.Stop()

	// 1. Encounter rate limit error
	errUpdate := connector.AgentStateUpdate{
		ConversationID:    "conv-recovery",
		ExecutionID:       "exec-1",
		StepID:            "step-1",
		Status:            "IDLE",
		TerminationReason: "ERROR",
		LastStepError: &connector.StepError{
			Code: "429",
		},
	}
	_ = rc.HandleAgentStateUpdate(context.Background(), errUpdate)

	if rc.GetRecoveryAttempts() != 1 || !rc.IsRecovering() {
		t.Fatalf("expected recoveryAttempts=1 and isRecovering=true, got %d, %v", rc.GetRecoveryAttempts(), rc.IsRecovering())
	}

	// 2. Subsequent execution succeeds
	succUpdate := connector.AgentStateUpdate{
		ConversationID:    "conv-recovery",
		ExecutionID:       "exec-2",
		StepID:            "step-1",
		Status:            "IDLE",
		TerminationReason: "DONE",
	}
	_ = rc.HandleAgentStateUpdate(context.Background(), succUpdate)

	if rc.GetRecoveryAttempts() != 0 {
		t.Errorf("expected recoveryAttempts reset to 0, got %d", rc.GetRecoveryAttempts())
	}
	if rc.IsRecovering() {
		t.Errorf("expected isRecovering=false after success")
	}
}

func TestRetryCoordinator_ThreeRetriesThenSwitch(t *testing.T) {
	var switchTriggered atomic.Bool
	switchChan := make(chan string, 1)

	onSwitch := func(ctx context.Context, reason string) {
		switchTriggered.Store(true)
		switchChan <- reason
	}

	rc := NewRetryCoordinator(nil, onSwitch, nil)
	rc.SetSleeper(func(ctx context.Context, d time.Duration) error {
		return nil
	})
	_ = rc.Start(context.Background())
	defer rc.Stop()

	// Feed 3 consecutive limit errors (attempts 1, 2, 3)
	for i := 1; i <= 3; i++ {
		upd := connector.AgentStateUpdate{
			ConversationID:    "conv-3retry",
			ExecutionID:       fmt.Sprintf("exec-%d", i),
			StepID:            "step-1",
			Status:            "IDLE",
			TerminationReason: "ERROR",
			LastStepError: &connector.StepError{
				Code: "RESOURCE_EXHAUSTED",
			},
		}
		err := rc.HandleAgentStateUpdate(context.Background(), upd)
		if err != nil {
			t.Fatalf("step %d update error: %v", i, err)
		}
		if rc.GetRecoveryAttempts() != i {
			t.Errorf("expected recoveryAttempts=%d, got %d", i, rc.GetRecoveryAttempts())
		}
	}

	// 4th limit error: attempts was 3, so now it exceeds max attempts and triggers switch
	upd4 := connector.AgentStateUpdate{
		ConversationID:    "conv-3retry",
		ExecutionID:       "exec-4",
		StepID:            "step-1",
		Status:            "IDLE",
		TerminationReason: "ERROR",
		LastStepError: &connector.StepError{
			Code: "429",
		},
	}
	err := rc.HandleAgentStateUpdate(context.Background(), upd4)
	if err != ErrMaxRetriesExceeded {
		t.Errorf("expected ErrMaxRetriesExceeded, got %v", err)
	}

	select {
	case reason := <-switchChan:
		if !switchTriggered.Load() {
			t.Errorf("expected switchTriggered=true")
		}
		if reason == "" {
			t.Errorf("expected non-empty switch reason")
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("timeout waiting for onSwitchNeeded callback")
	}
}

func TestRetryCoordinator_ProactiveSupersedesRecovery(t *testing.T) {
	var switchTriggered atomic.Bool
	switchChan := make(chan string, 1)

	onSwitch := func(ctx context.Context, reason string) {
		switchTriggered.Store(true)
		switchChan <- reason
	}

	rc := NewRetryCoordinator(nil, onSwitch, nil)
	rc.SetSleeper(func(ctx context.Context, d time.Duration) error {
		return nil
	})
	_ = rc.Start(context.Background())
	defer rc.Stop()

	// Start a recovery attempt
	upd := connector.AgentStateUpdate{
		ConversationID:    "conv-proactive",
		ExecutionID:       "exec-1",
		StepID:            "step-1",
		Status:            "IDLE",
		TerminationReason: "ERROR",
		LastStepError: &connector.StepError{
			Code: "429",
		},
	}
	_ = rc.HandleAgentStateUpdate(context.Background(), upd)

	if !rc.IsRecovering() || rc.GetRecoveryAttempts() != 1 {
		t.Fatalf("expected recovery active")
	}

	// Proactive trigger arrives (5h <= 2% or weekly = 0%)
	rc.SupersedeByProactive(context.Background(), "5h quota depleted (1.5% <= 2%)")

	if rc.IsRecovering() {
		t.Errorf("expected isRecovering=false after proactive supersession")
	}
	if rc.GetRecoveryAttempts() != 0 {
		t.Errorf("expected recoveryAttempts=0 after proactive supersession")
	}

	select {
	case reason := <-switchChan:
		if !switchTriggered.Load() {
			t.Errorf("expected switch to be triggered")
		}
		if reason != "5h quota depleted (1.5% <= 2%)" {
			t.Errorf("unexpected reason: %s", reason)
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("timeout waiting for proactive switch callback")
	}
}

func TestRetryCoordinator_SessionChange_PurgesEvents(t *testing.T) {
	rc := NewRetryCoordinator(nil, nil, nil)
	rc.SetSleeper(func(ctx context.Context, d time.Duration) error {
		return nil
	})
	_ = rc.Start(context.Background())
	defer rc.Stop()

	rc.SetCurrentEmail("user1@example.com")

	upd := connector.AgentStateUpdate{
		ConversationID:    "conv-session",
		ExecutionID:       "exec-1",
		StepID:            "step-1",
		Status:            "IDLE",
		TerminationReason: "ERROR",
		LastStepError: &connector.StepError{
			Code: "429",
		},
	}
	_ = rc.HandleAgentStateUpdate(context.Background(), upd)
	if rc.GetRecoveryAttempts() != 1 {
		t.Errorf("expected attempts=1")
	}

	// Switch session to user2
	rc.SetCurrentEmail("user2@example.com")

	if rc.GetRecoveryAttempts() != 0 {
		t.Errorf("expected attempts reset to 0 after session change, got %d", rc.GetRecoveryAttempts())
	}
	if rc.IsRecovering() {
		t.Errorf("expected isRecovering=false after session change")
	}

	// Re-sending same event under new session should NOT be dropped as a duplicate from old session
	_ = rc.HandleAgentStateUpdate(context.Background(), upd)
	if rc.GetRecoveryAttempts() != 1 {
		t.Errorf("expected event to be processed under new session, got attempts=%d", rc.GetRecoveryAttempts())
	}
}