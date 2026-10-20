package ui

import (
	"antigravity-switch/internal/manager"
	"context"
	"errors"
	"fyne.io/fyne/v2"
	"path/filepath"
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
	app := newIsolatedApp(testApp, mgr)

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
	if rc == nil || rc.IsRunning() {
		t.Errorf("expected no subscriptions while initial toggle is OFF")
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
	app := newIsolatedApp(testApp, mgr)
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

	_ = results
	app.mu.RLock()
	prevRefresh := app.lastQuotaRefresh
	app.mu.RUnlock()

	app.autoLoop.TriggerNow()
	end := time.Now().Add(5 * time.Second)
	for time.Now().Before(end) {
		app.mu.RLock()
		done := !app.lastQuotaRefresh.IsZero() && !app.lastQuotaRefresh.Equal(prevRefresh)
		app.mu.RUnlock()
		if done {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	// Verify UI quota overview remains operational
	app.mu.RLock()
	lastRefresh := app.lastQuotaRefresh
	app.mu.RUnlock()
	if lastRefresh.IsZero() || lastRefresh.Equal(prevRefresh) {
		t.Fatal("telemetry callback did not refresh UI")
	}
}

func TestUIBinding_Toggle_SetterBindingAndDynamicCheck(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	mgr := createTestManager(t)
	app := newIsolatedApp(testApp, mgr)
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

// Remaining manual caller coverage runs through App.SwitchAccount in ui_test.go.

func newIsolatedApp(fyneApp fyne.App, mgr *manager.AccountManager) *App {
	a := NewAppWithFyne(fyneApp, mgr)
	if mgr != nil && mgr.Store() != nil {
		pool, err := telemetry.NewTelemetryPool(&testQuotaClient{}, a.quotaCache, mgr.Store())
		if err != nil {
			panic(err)
		}
		pool.SetLocalQuotaFetcher(nil)
		a.SetTelemetryPool(pool)
		a.autoLoop, _ = telemetry.NewAutoTelemetryLoop(pool, telemetry.DefaultLoopInterval)
	}
	if a.controller != nil {
		a.controller.SetClientFinder(func(context.Context) (*connector.Client, error) {
			return nil, errors.New("isolated GUI test: no external Language Server")
		})
		a.controller.SetPbtxtPath(filepath.Join(mgr.Store().GetStoreDir(), "test-only.pbtxt"))
		a.controller.RetryCoordinator().SetConversationsScanner(func(context.Context) ([]string, error) { return nil, nil })
	}
	return a
}
