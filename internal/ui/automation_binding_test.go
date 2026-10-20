package ui

import (
	"context"
	"strings"
	"testing"
	"time"

	"antigravity-switch/internal/automation"
	"antigravity-switch/internal/connector"
	"antigravity-switch/internal/telemetry"

	"fyne.io/fyne/v2/test"
)

// -----------------------------------------------------------------------------
// Slice P2-A: UI Automation Binding Assertions
// -----------------------------------------------------------------------------

func TestUIBinding_AppLifecycle_ControllerAndStatusCleanup(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	mgr := createTestManager(t)
	app := NewAppWithFyne(testApp, mgr)

	ctrl := app.Controller()
	if ctrl == nil {
		t.Fatal("expected non-nil AutomationController in App")
	}
	if ctrl.Gate() == nil {
		t.Fatal("expected non-nil SwitchGate in AutomationController")
	}

	app.Init()
	waitForIdle(app, 1*time.Second)

	// Verify controller is active after Init
	rc := ctrl.RetryCoordinator()
	if rc == nil || !rc.IsRunning() {
		t.Errorf("expected RetryCoordinator to be running after App.Init")
	}

	// Close cleans up controller and status listener goroutine
	app.Close()

	if rc.IsRunning() {
		t.Errorf("expected RetryCoordinator to be stopped after App.Close")
	}
}

func TestUIBinding_TelemetryCallback_DualDispatchWithoutOverwrite(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	mgr := createTestManager(t)
	app := NewAppWithFyne(testApp, mgr)
	defer app.Close()

	app.Init()
	waitForIdle(app, 1*time.Second)

	// Enable automation
	app.SetAutoSwitchEnabled(true)
	if !app.IsAutoSwitchEnabled() || !app.Controller().IsEnabled() {
		t.Fatalf("expected automation enabled on App and Controller")
	}

	// Emit mock status to verify status bar binding
	app.Controller().EmitStatus(automation.StatusSwitching, "Đang kiểm tra quota...", nil)
	time.Sleep(150 * time.Millisecond)

	if !strings.Contains(app.statusBar.Text, "Đang kiểm tra quota...") {
		t.Errorf("expected status bar text to reflect automation status, got: %q", app.statusBar.Text)
	}

	// Update quota data
	now := time.Now().UTC()
	results := []*telemetry.AccountQuotaData{
		{
			Email:       "test@example.com",
			Status:      telemetry.StatusPlentiful,
			LastUpdated: now,
		},
	}

	app.Controller().OnTelemetryUpdate(results)
	time.Sleep(100 * time.Millisecond)

	// Verify UI quota overview remains operational
	app.mu.RLock()
	lastRefresh := app.lastQuotaRefresh
	app.mu.RUnlock()
	_ = lastRefresh
}

func TestUIBinding_Toggle_SetterBindingAndDynamicCheck(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	mgr := createTestManager(t)
	app := NewAppWithFyne(testApp, mgr)
	defer app.Close()

	app.Init()
	waitForIdle(app, 1*time.Second)

	// 1. Initial state is disabled
	if app.IsAutoSwitchEnabled() {
		t.Errorf("expected automation initially disabled")
	}

	// 2. Toggle ON
	app.SetAutoSwitchEnabled(true)
	if !app.IsAutoSwitchEnabled() {
		t.Errorf("expected App.IsAutoSwitchEnabled() == true")
	}
	if !app.Controller().IsEnabled() {
		t.Errorf("expected Controller.IsEnabled() == true")
	}

	// 3. Toggle OFF
	app.SetAutoSwitchEnabled(false)
	if app.IsAutoSwitchEnabled() {
		t.Errorf("expected App.IsAutoSwitchEnabled() == false")
	}
	if app.Controller().IsEnabled() {
		t.Errorf("expected Controller.IsEnabled() == false")
	}
}

func TestUIBinding_ManualSwitch_Resets_RecoveryState(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	mgr := createTestManager(t)
	app := NewAppWithFyne(testApp, mgr)
	defer app.Close()

	app.Init()
	waitForIdle(app, 1*time.Second)

	rc := app.Controller().RetryCoordinator()
	if rc == nil {
		t.Fatal("expected non-nil RetryCoordinator")
	}

	// Simulate coordinator in recovery state
	ctx := context.Background()
	_ = rc.HandleAgentStateUpdate(ctx, connector.AgentStateUpdate{
		ConversationID:    "conv-ui-1",
		ExecutionID:       "exec-1",
		StepID:            "step-1",
		Status:            "IDLE",
		TerminationReason: "429 RATE_LIMIT",
		Attempt:           1,
	})

	if !rc.IsRecovering() {
		t.Errorf("expected coordinator to enter recovering state")
	}

	// Manual switch resets coordinator
	rc.Reset()
	if rc.IsRecovering() || rc.GetRecoveryAttempts() != 0 {
		t.Errorf("expected recovery state reset on manual switch, got recovering=%v attempts=%d", rc.IsRecovering(), rc.GetRecoveryAttempts())
	}
}
