package ui

import (
	"path/filepath"
	"testing"
	"time"

	"antigravity-switch/internal/engines/antigravity/automation"
	"antigravity-switch/internal/engines/antigravity/model"
	"antigravity-switch/internal/engines/antigravity/store"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

// Helper to save a dummy account in the manager's store
func addTestAccount(t *testing.T, st *store.AccountStore, email string, status string) {
	t.Helper()
	tok := &model.TokenData{
		AccessToken:  "tok_" + email,
		RefreshToken: "ref_" + email,
		IDToken:      makeTestJWT(email),
		Expiry:       time.Now().Add(1 * time.Hour),
	}
	if err := st.SaveAccount(email, tok); err != nil {
		t.Fatalf("failed saving test account %s: %v", email, err)
	}
	if status != "" && status != "on" {
		if err := st.SetAccountStatus(email, status); err != nil {
			t.Fatalf("failed setting status %s on %s: %v", status, email, err)
		}
	}
}

func TestApp_ToggleAccountStatus_TogglesAndPersists(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	mgr := createTestManager(t)
	st := mgr.Store()

	// Add test account
	testEmail := "secondary@gmail.com"
	addTestAccount(t, st, testEmail, "on")

	app := newIsolatedApp(testApp, mgr)
	defer app.Close()

	app.SetPreferencesPath(filepath.Join(t.TempDir(), "ui_preferences.json"))
	app.Init()
	waitForIdle(app, 1*time.Second)

	// Verify initial status is "on"
	if st := mgr.GetAccountStatus(testEmail); st != "on" {
		t.Fatalf("expected initial status on, got %s", st)
	}

	// 1. Toggle to "off"
	app.ToggleAccountStatus(testEmail)

	// Verify status in store
	if st := mgr.GetAccountStatus(testEmail); st != "off" {
		t.Fatalf("expected status 'off' after toggle, got %s", st)
	}

	// Verify in-place update on card widget
	cards := app.AccountCards()
	var targetCard *AccountCardWidget
	for _, c := range cards {
		if c != nil && c.Email == testEmail {
			targetCard = c
			break
		}
	}
	if targetCard == nil {
		t.Fatalf("targetCard for %s not found in accountCards", testEmail)
	}
	if targetCard.Status != "off" {
		t.Errorf("expected card Status 'off', got %s", targetCard.Status)
	}
	if targetCard.statusButton.Status != "off" || targetCard.statusButton.text.Text != "[OFF]" {
		t.Errorf("expected statusButton [OFF], got %s / %s", targetCard.statusButton.Status, targetCard.statusButton.text.Text)
	}

	// 2. Toggle back to "on"
	app.ToggleAccountStatus(testEmail)

	if st := mgr.GetAccountStatus(testEmail); st != "on" {
		t.Fatalf("expected status 'on' after second toggle, got %s", st)
	}
	if targetCard.Status != "on" {
		t.Errorf("expected card Status 'on', got %s", targetCard.Status)
	}
	if targetCard.statusButton.Status != "on" || targetCard.statusButton.text.Text != "[ON]" {
		t.Errorf("expected statusButton [ON], got %s / %s", targetCard.statusButton.Status, targetCard.statusButton.text.Text)
	}
}

func TestApp_ToggleAccountStatus_ActiveAccount_AutoSwitchOff(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	mgr := createTestManager(t)
	st := mgr.Store()

	activeEmail := "current@gmail.com"
	addTestAccount(t, st, activeEmail, "on")

	app := newIsolatedApp(testApp, mgr)
	defer app.Close()

	app.SetPreferencesPath(filepath.Join(t.TempDir(), "ui_preferences.json"))
	app.SetAutoSwitchEnabled(false)
	app.Init()
	waitForIdle(app, 1*time.Second)

	// Toggle active account to "off" while auto-switch is off
	app.ToggleAccountStatus(activeEmail)

	if st := mgr.GetAccountStatus(activeEmail); st != "off" {
		t.Fatalf("expected active account status to become 'off', got %s", st)
	}

	// Auto-switch remains disabled
	if app.IsAutoSwitchEnabled() {
		t.Errorf("expected auto-switch to remain disabled")
	}
}

func TestApp_ToggleAccountStatus_ActiveAccount_AutoSwitchOn_NoCandidates_TriggersSafetyStop(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	mgr := createTestManager(t)
	st := mgr.Store()

	activeEmail := "current@gmail.com"
	addTestAccount(t, st, activeEmail, "on")

	prefPath := filepath.Join(t.TempDir(), "ui_preferences.json")
	app := newIsolatedApp(testApp, mgr)
	defer app.Close()

	app.SetPreferencesPath(prefPath)
	app.Init()
	waitForIdle(app, 1*time.Second)

	// Turn auto-switch ON
	app.SetAutoSwitchEnabled(true)
	if !app.IsAutoSwitchEnabled() {
		t.Fatalf("expected auto-switch to be enabled")
	}

	// Toggle active account to "off" when no other accounts exist
	app.ToggleAccountStatus(activeEmail)

	// Status of active account should be "off"
	if st := mgr.GetAccountStatus(activeEmail); st != "off" {
		t.Fatalf("expected active account status to be 'off', got %s", st)
	}

	// Safety Stop must be triggered:
	if app.IsAutoSwitchEnabled() {
		t.Errorf("expected Safety Stop: autoSwitchEnabled == false")
	}
	if app.autoSwitchToggle != nil && app.autoSwitchToggle.IsOn() {
		t.Errorf("expected Safety Stop: autoSwitchToggle.IsOn() == false")
	}
	if app.StatusBarText() != "⚠️ Không có tài khoản để chuyển." {
		t.Errorf("expected status bar '⚠️ Không có tài khoản để chuyển.', got '%s'", app.StatusBarText())
	}

	// Check disk persistence
	deadline := time.Now().Add(2 * time.Second)
	var persistedPref *UIPreferences
	for time.Now().Before(deadline) {
		p, err := LoadUIPreferences(prefPath)
		if err == nil && p != nil && !p.AutoSwitch {
			persistedPref = p
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if persistedPref == nil || persistedPref.AutoSwitch {
		t.Errorf("expected auto_switch: false persisted to disk, got %#v", persistedPref)
	}
}

func TestApp_OnStatusChange_StatusNoCandidate_TriggersSafetyStop(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	mgr := createTestManager(t)
	prefPath := filepath.Join(t.TempDir(), "ui_preferences.json")

	app := newIsolatedApp(testApp, mgr)
	defer app.Close()

	app.SetPreferencesPath(prefPath)
	app.Init()
	waitForIdle(app, 1*time.Second)

	// Enable auto-switch
	app.SetAutoSwitchEnabled(true)
	if !app.IsAutoSwitchEnabled() {
		t.Fatalf("expected autoSwitchEnabled true")
	}

	// Optional check widget injected
	check := widget.NewCheck("", nil)
	check.SetChecked(true)
	app.SetAutoSwitchCheck(check)

	// Dispatch StatusNoCandidate via OnStatusChange
	app.OnStatusChange(automation.StatusEvent{
		Code:    automation.StatusNoCandidate,
		Message: "Không có tài khoản để chuyển.",
	})

	// 1. autoSwitchEnabled flag must be false
	if app.IsAutoSwitchEnabled() {
		t.Errorf("expected IsAutoSwitchEnabled() == false")
	}

	// 2. autoSwitchToggle must be OFF
	if app.autoSwitchToggle != nil && app.autoSwitchToggle.IsOn() {
		t.Errorf("expected autoSwitchToggle.IsOn() == false")
	}

	// 3. autoSwitchCheck must be unchecked
	if check.Checked {
		t.Errorf("expected autoSwitchCheck.Checked == false")
	}

	// 4. Status bar text must match standardized message
	if app.StatusBarText() != "⚠️ Không có tài khoản để chuyển." {
		t.Errorf("expected status bar text '⚠️ Không có tài khoản để chuyển.', got '%s'", app.StatusBarText())
	}

	// 5. Disk preferences must persist auto_switch: false
	deadline := time.Now().Add(2 * time.Second)
	var persistedPref *UIPreferences
	for time.Now().Before(deadline) {
		p, err := LoadUIPreferences(prefPath)
		if err == nil && p != nil && !p.AutoSwitch {
			persistedPref = p
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if persistedPref == nil || persistedPref.AutoSwitch {
		t.Errorf("expected auto_switch: false persisted to disk, got %#v", persistedPref)
	}
}

func TestApp_ToggleAccountStatus_ActiveAccount_AutoSwitchOn_WithOtherCandidate(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	mgr := createTestManager(t)
	st := mgr.Store()

	activeEmail := "current@gmail.com"
	candidateEmail := "candidate@gmail.com"
	addTestAccount(t, st, activeEmail, "on")
	addTestAccount(t, st, candidateEmail, "on")

	prefPath := filepath.Join(t.TempDir(), "ui_preferences.json")
	app := newIsolatedApp(testApp, mgr)
	defer app.Close()

	app.SetPreferencesPath(prefPath)
	app.Init()
	waitForIdle(app, 1*time.Second)

	app.SetAutoSwitchEnabled(true)

	// Toggling active account when candidate is available should not immediately Safety Stop
	app.ToggleAccountStatus(activeEmail)

	if st := mgr.GetAccountStatus(activeEmail); st != "off" {
		t.Fatalf("expected active account status to become 'off', got %s", st)
	}
	// Since another candidate is "on", it initiates switch rather than immediately halting
}

func TestApp_ToggleAccountStatus_NilManagerOrBusyNoCrash(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	// 1. Nil manager
	appNil := NewAppWithFyne(testApp, nil)
	appNil.ToggleAccountStatus("any@example.com") // Should not panic

	// 2. Busy app
	mgr := createTestManager(t)
	appBusy := newIsolatedApp(testApp, mgr)
	defer appBusy.Close()
	appBusy.mu.Lock()
	appBusy.isBusy = true
	appBusy.mu.Unlock()

	appBusy.ToggleAccountStatus("any@example.com") // Should return immediately
}
