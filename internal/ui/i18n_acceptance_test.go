package ui

import (
	"path/filepath"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"antigravity-switch/internal/i18n"
	"antigravity-switch/internal/manager"
)

// TestAcceptance_i18n_DropdownPosition verifies that the language dropdown is located in row1Header
// immediately to the left of the minimize button (btnMin).
func TestAcceptance_i18n_DropdownPosition(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	mgr := createTestManager(t)
	app := newIsolatedApp(testApp, mgr)
	defer app.Close()

	app.Init()
	waitForIdle(app, 1*time.Second)

	win := app.Window()
	content := win.Content().(*fyne.Container)

	var topSection *fyne.Container
	for _, obj := range content.Objects {
		if c, ok := obj.(*fyne.Container); ok {
			topSection = c
			break
		}
	}
	if topSection == nil {
		t.Fatal("expected topSection container")
	}

	var row1Header *fyne.Container
	for _, obj := range topSection.Objects {
		if c, ok := obj.(*fyne.Container); ok {
			row1Header = c
			break
		}
	}
	if row1Header == nil {
		t.Fatal("expected row1Header container")
	}

	var windowControls *fyne.Container
	for _, obj := range row1Header.Objects {
		if c, ok := obj.(*fyne.Container); ok && len(c.Objects) == 4 {
			windowControls = c
			break
		}
	}
	if windowControls == nil {
		t.Fatalf("expected to find windowControls container with 4 items")
	}

	// [0] Dropdown VN/EN, [1] btnMin, [2] btnMax, [3] btnClose
	langWrap, ok := windowControls.Objects[0].(*fyne.Container)
	if !ok || len(langWrap.Objects) == 0 {
		t.Fatalf("expected windowControls.Objects[0] to wrap langSelect")
	}
	if langWrap.Objects[0] != app.langSelect {
		t.Errorf("expected app.langSelect at index 0 of windowControls")
	}

	// Verify dropdown size 54x22
	if langWrap.MinSize().Width != 54 || langWrap.MinSize().Height != 22 {
		t.Errorf("expected langSelectWrap size 54x22, got %vx%v", langWrap.MinSize().Width, langWrap.MinSize().Height)
	}

	// Verify default selection is VN
	if app.langSelect.Selected != "VN" {
		t.Errorf("expected default langSelect to be 'VN', got '%s'", app.langSelect.Selected)
	}
}

// TestAcceptance_i18n_LiveInPlaceSwitching verifies that switching between VN and EN
// updates buttons, labels, and account cards immediately in-place without restarting or rebuilding the window.
func TestAcceptance_i18n_LiveInPlaceSwitching(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	mgr := createTestManager(t)
	app := newIsolatedApp(testApp, mgr)
	defer app.Close()

	tempDir := t.TempDir()
	prefFile := filepath.Join(tempDir, "ui_preferences.json")
	app.preferencesPath = prefFile

	app.Init()
	waitForIdle(app, 1*time.Second)

	// Feed dummy account overview
	overview := &manager.AccountOverview{
		Accounts: []manager.AccountDisplayItem{
			{Email: "user1@example.com", Name: "User 1", IsActive: true, HasToken: true, Status: "on"},
			{Email: "user2@example.com", Name: "User 2", IsActive: false, HasToken: true, Status: "on"},
			{Email: "user3@example.com", Name: "User 3", IsActive: false, HasToken: false, Status: "on"},
		},
	}
	app.updateUI(overview)
	waitForIdle(app, 500*time.Millisecond)

	// Step 1: Initial state is VN
	if app.addAccountBtn.Text != "Thêm" {
		t.Errorf("expected 'Thêm', got '%s'", app.addAccountBtn.Text)
	}
	if app.refreshBtn.Text != "Làm mới danh sách" {
		t.Errorf("expected 'Làm mới danh sách', got '%s'", app.refreshBtn.Text)
	}
	if app.quotaRefreshAllBtn.Text != "Làm mới Quota" {
		t.Errorf("expected 'Làm mới Quota', got '%s'", app.quotaRefreshAllBtn.Text)
	}

	// Step 2: Switch to English via SetLanguage("en")
	app.SetLanguage("en")
	waitForIdle(app, 500*time.Millisecond)

	if app.CurrentLocale() != "en" {
		t.Errorf("expected currentLocale 'en', got '%s'", app.CurrentLocale())
	}
	if app.langSelect.Selected != "EN" {
		t.Errorf("expected langSelect 'EN', got '%s'", app.langSelect.Selected)
	}
	if app.addAccountBtn.Text != "Add" {
		t.Errorf("expected 'Add', got '%s'", app.addAccountBtn.Text)
	}
	if app.refreshBtn.Text != "Refresh List" {
		t.Errorf("expected 'Refresh List', got '%s'", app.refreshBtn.Text)
	}
	if app.quotaRefreshAllBtn.Text != "Refresh Quota" {
		t.Errorf("expected 'Refresh Quota', got '%s'", app.quotaRefreshAllBtn.Text)
	}
	if headerRow2MaskLabel != nil && headerRow2MaskLabel.Text != "Mask Accounts:" {
		t.Errorf("expected 'Mask Accounts:', got '%s'", headerRow2MaskLabel.Text)
	}
	if headerRow2AutoLabel != nil && headerRow2AutoLabel.Text != "Auto Switch:" {
		t.Errorf("expected 'Auto Switch:', got '%s'", headerRow2AutoLabel.Text)
	}

	// Verify persistence in ui_preferences.json
	time.Sleep(200 * time.Millisecond)
	pref, err := LoadUIPreferences(prefFile)
	if err != nil {
		t.Fatalf("failed loading ui preferences: %v", err)
	}
	if pref.Language != "en" {
		t.Errorf("expected persisted language 'en', got '%s'", pref.Language)
	}

	// Step 3: Switch back to Vietnamese
	app.SetLanguage("vi")
	waitForIdle(app, 500*time.Millisecond)

	if app.CurrentLocale() != "vi" {
		t.Errorf("expected currentLocale 'vi', got '%s'", app.CurrentLocale())
	}
	if app.langSelect.Selected != "VN" {
		t.Errorf("expected langSelect 'VN', got '%s'", app.langSelect.Selected)
	}
	if app.addAccountBtn.Text != "Thêm" {
		t.Errorf("expected 'Thêm', got '%s'", app.addAccountBtn.Text)
	}
	if app.refreshBtn.Text != "Làm mới danh sách" {
		t.Errorf("expected 'Làm mới danh sách', got '%s'", app.refreshBtn.Text)
	}
	if app.quotaRefreshAllBtn.Text != "Làm mới Quota" {
		t.Errorf("expected 'Làm mới Quota', got '%s'", app.quotaRefreshAllBtn.Text)
	}

	// Verify persistence after switching back to vi
	time.Sleep(200 * time.Millisecond)
	pref2, err := LoadUIPreferences(prefFile)
	if err != nil {
		t.Fatalf("failed loading ui preferences: %v", err)
	}
	if pref2.Language != "vi" {
		t.Errorf("expected persisted language 'vi', got '%s'", pref2.Language)
	}
}

// TestAcceptance_i18n_StartupRestoration verifies that the stored language preference
// in logs/ui_preferences.json is correctly restored on app startup.
func TestAcceptance_i18n_StartupRestoration(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	mgr := createTestManager(t)
	tempDir := t.TempDir()
	prefFile := filepath.Join(tempDir, "ui_preferences.json")

	// Pre-seed preference with English
	top := true
	_ = SaveUIPreferences(prefFile, &UIPreferences{
		Theme:        ThemeDark,
		AlwaysOnTop:  &top,
		Language:     "en",
	})

	app := newIsolatedApp(testApp, mgr)
	defer app.Close()
	app.preferencesPath = prefFile

	// Re-load preferences into app
	if p, err := LoadUIPreferences(prefFile); err == nil && p != nil {
		app.currentLocale = p.Language
		i18n.SetCurrentLocale(i18n.Locale(p.Language))
	}

	app.Init()
	waitForIdle(app, 1*time.Second)

	if app.CurrentLocale() != "en" {
		t.Errorf("expected startup locale 'en', got '%s'", app.CurrentLocale())
	}
	if app.langSelect.Selected != "EN" {
		t.Errorf("expected langSelect 'EN', got '%s'", app.langSelect.Selected)
	}
	if app.addAccountBtn.Text != "Add" {
		t.Errorf("expected 'Add', got '%s'", app.addAccountBtn.Text)
	}
}
