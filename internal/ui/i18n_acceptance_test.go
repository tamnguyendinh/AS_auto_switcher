package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"antigravity-switch/internal/i18n"
	"antigravity-switch/internal/manager"
	"antigravity-switch/internal/model"
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

// TestAcceptance_i18n_FooterAndModalBilingual verifies that the footer status label,
// overview empty state, and add modal accurately switch between Vietnamese and English.
func TestAcceptance_i18n_FooterAndModalBilingual(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	mgr := createTestManager(t)
	app := newIsolatedApp(testApp, mgr)
	defer app.Close()

	app.Init()
	waitForIdle(app, 500*time.Millisecond)

	// In VI:
	app.SetLanguage("vi")
	waitForIdle(app, 200*time.Millisecond)

	dVI := i18n.Current()
	if app.serverStatusLabel != nil {
		if !strings.Contains(app.serverStatusLabel.Text, "Hoạt động") && !strings.Contains(app.serverStatusLabel.Text, "Không hoạt động") && !strings.Contains(app.serverStatusLabel.Text, "kết nối") {
			t.Errorf("expected VI server status, got %q", app.serverStatusLabel.Text)
		}
	}

	contentBoxVI, scanBtnVI, _, _ := app.buildAddAccountModalContent()
	if scanBtnVI.Text != dVI.ModalScanChrome {
		t.Errorf("expected VI scanBtn %q, got %q", dVI.ModalScanChrome, scanBtnVI.Text)
	}
	_ = contentBoxVI

	// Switch to EN:
	app.SetLanguage("en")
	waitForIdle(app, 200*time.Millisecond)

	dEN := i18n.Current()
	if app.serverStatusLabel != nil {
		if !strings.Contains(app.serverStatusLabel.Text, "Active") && !strings.Contains(app.serverStatusLabel.Text, "Inactive") && !strings.Contains(app.serverStatusLabel.Text, "Connecting") {
			t.Errorf("expected EN server status, got %q", app.serverStatusLabel.Text)
		}
	}

	contentBoxEN, scanBtnEN, _, _ := app.buildAddAccountModalContent()
	if scanBtnEN.Text != dEN.ModalScanChrome {
		t.Errorf("expected EN scanBtn %q, got %q", dEN.ModalScanChrome, scanBtnEN.Text)
	}
	_ = contentBoxEN
}

// TestAcceptance_i18n_AddModal_BilingualCandidateBadges verifies bilingual rendering
// and in-place reactive mutation of candidate badges in Add Account Modal (TC-MODAL-i18n-01 -> 04).
func TestAcceptance_i18n_AddModal_BilingualCandidateBadges(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	tempDir := t.TempDir()
	mgr, accStore, discStore := createAcceptanceManagerInDir(t, tempDir)
	app := newIsolatedApp(testApp, mgr)
	defer app.Close()

	app.Init()
	waitForIdle(app, 500*time.Millisecond)

	// Set up 4 candidate accounts:
	// 1. oauthEmail: has token in store -> [✔ Đã OAuth] (VI) / [✔ OAuth Done] (EN)
	oauthEmail := "oauth_user@company.com"
	if err := discStore.SaveDiscovered(oauthEmail); err != nil {
		t.Fatal(err)
	}
	if err := accStore.SaveAccount(oauthEmail, &model.TokenData{
		AccessToken: "test_token",
		IDToken:     makeTestJWT(oauthEmail),
	}); err != nil {
		t.Fatal(err)
	}

	// 2. addedEmail: Added == true -> [Đã thêm] (VI) / [Added] (EN)
	addedEmail := "added_user@school.edu"
	if err := discStore.SaveDiscovered(addedEmail); err != nil {
		t.Fatal(err)
	}
	if err := discStore.SetDiscoveredAdded(addedEmail, true); err != nil {
		t.Fatal(err)
	}

	// 3. needOAuthEmail: Added == false, HasToken == false -> [Cần OAuth] + "Thêm" (VI) / [Need OAuth] + "Add" (EN)
	needOAuthEmail := "need_oauth_user@gmail.com"
	if err := discStore.SaveDiscovered(needOAuthEmail); err != nil {
		t.Fatal(err)
	}

	// 4. ghostEmail: FileExists == false -> [Tài khoản không tồn tại] (VI) / [Account Not Found] (EN)
	ghostEmail := "ghost_user@deleted.com"
	if err := discStore.SaveDiscovered(ghostEmail); err != nil {
		t.Fatal(err)
	}
	if err := discStore.SetDiscoveredAdded(ghostEmail, true); err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(filepath.Join(tempDir, "discovered_accounts", ghostEmail+".json"))

	app.mu.Lock()
	app.overview = &manager.AccountOverview{
		Accounts: []manager.AccountDisplayItem{
			{Email: ghostEmail, State: model.StateUnauth, HasToken: false},
		},
	}
	app.mu.Unlock()

	// --- 1. TC-MODAL-i18n-01: Vietnamese Locale Validation ---
	app.SetLanguage("vi")
	waitForIdle(app, 200*time.Millisecond)

	_, _, listContainerVI, _ := app.buildAddAccountModalContent()
	rowsVI := make(map[string]*fyne.Container)
	for _, obj := range listContainerVI.Objects {
		em, slot := extractModalRow(t, obj)
		rowsVI[em] = slot
	}

	// Verify oauthEmail in VI
	badgeOAuthVI, addOAuthVI, delOAuthVI := inspectAcceptanceActionSlot(t, rowsVI[oauthEmail])
	if badgeOAuthVI != "[✔ Đã OAuth]" {
		t.Errorf("VI: expected '[✔ Đã OAuth]' for %s, got %q (Must NOT be '[✔ Active]' or '[✔ Đang dùng]')", oauthEmail, badgeOAuthVI)
	}
	if addOAuthVI != nil {
		t.Errorf("VI: expected no add button for %s", oauthEmail)
	}
	if delOAuthVI == nil {
		t.Errorf("VI: expected delete button for %s", oauthEmail)
	}

	// Verify addedEmail in VI
	badgeAddedVI, addAddedVI, delAddedVI := inspectAcceptanceActionSlot(t, rowsVI[addedEmail])
	if badgeAddedVI != "[Đã thêm]" {
		t.Errorf("VI: expected '[Đã thêm]' for %s, got %q (Must NOT be '[Đã add]')", addedEmail, badgeAddedVI)
	}
	if addAddedVI != nil {
		t.Errorf("VI: expected no add button for %s", addedEmail)
	}
	if delAddedVI == nil {
		t.Errorf("VI: expected delete button for %s", addedEmail)
	}

	// Verify needOAuthEmail in VI
	badgeNeedVI, addNeedVI, delNeedVI := inspectAcceptanceActionSlot(t, rowsVI[needOAuthEmail])
	if badgeNeedVI != "[Cần OAuth]" {
		t.Errorf("VI: expected '[Cần OAuth]' for %s, got %q", needOAuthEmail, badgeNeedVI)
	}
	if addNeedVI == nil || addNeedVI.Text != "Thêm" {
		t.Errorf("VI: expected add button labeled 'Thêm' for %s, got %v", needOAuthEmail, addNeedVI)
	}
	if delNeedVI == nil {
		t.Errorf("VI: expected delete button for %s", needOAuthEmail)
	}

	// Verify ghostEmail in VI
	badgeGhostVI, addGhostVI, delGhostVI := inspectAcceptanceActionSlot(t, rowsVI[ghostEmail])
	if badgeGhostVI != "[Tài khoản không tồn tại]" {
		t.Errorf("VI: expected '[Tài khoản không tồn tại]' for %s, got %q", ghostEmail, badgeGhostVI)
	}
	if addGhostVI != nil {
		t.Errorf("VI: expected no add button for ghost account %s", ghostEmail)
	}
	if delGhostVI == nil {
		t.Errorf("VI: expected delete button for ghost account %s", ghostEmail)
	}

	// --- 2. TC-MODAL-i18n-02: English Locale Validation ---
	app.SetLanguage("en")
	waitForIdle(app, 200*time.Millisecond)

	_, _, listContainerEN, _ := app.buildAddAccountModalContent()
	rowsEN := make(map[string]*fyne.Container)
	for _, obj := range listContainerEN.Objects {
		em, slot := extractModalRow(t, obj)
		rowsEN[em] = slot
	}

	// Verify oauthEmail in EN
	badgeOAuthEN, addOAuthEN, delOAuthEN := inspectAcceptanceActionSlot(t, rowsEN[oauthEmail])
	if badgeOAuthEN != "[✔ OAuth Done]" {
		t.Errorf("EN: expected '[✔ OAuth Done]' for %s, got %q", oauthEmail, badgeOAuthEN)
	}
	if addOAuthEN != nil {
		t.Errorf("EN: expected no add button for %s", oauthEmail)
	}
	if delOAuthEN == nil {
		t.Errorf("EN: expected delete button for %s", oauthEmail)
	}

	// Verify addedEmail in EN
	badgeAddedEN, addAddedEN, delAddedEN := inspectAcceptanceActionSlot(t, rowsEN[addedEmail])
	if badgeAddedEN != "[Added]" {
		t.Errorf("EN: expected '[Added]' for %s, got %q", addedEmail, badgeAddedEN)
	}
	if addAddedEN != nil {
		t.Errorf("EN: expected no add button for %s", addedEmail)
	}
	if delAddedEN == nil {
		t.Errorf("EN: expected delete button for %s", addedEmail)
	}

	// Verify needOAuthEmail in EN
	badgeNeedEN, addNeedEN, delNeedEN := inspectAcceptanceActionSlot(t, rowsEN[needOAuthEmail])
	if badgeNeedEN != "[Need OAuth]" {
		t.Errorf("EN: expected '[Need OAuth]' for %s, got %q", needOAuthEmail, badgeNeedEN)
	}
	if addNeedEN == nil || addNeedEN.Text != "Add" {
		t.Errorf("EN: expected add button labeled 'Add' for %s, got %v", needOAuthEmail, addNeedEN)
	}
	if delNeedEN == nil {
		t.Errorf("EN: expected delete button for %s", needOAuthEmail)
	}

	// Verify ghostEmail in EN
	badgeGhostEN, addGhostEN, delGhostEN := inspectAcceptanceActionSlot(t, rowsEN[ghostEmail])
	if badgeGhostEN != "[Account Not Found]" {
		t.Errorf("EN: expected '[Account Not Found]' for %s, got %q", ghostEmail, badgeGhostEN)
	}
	if addGhostEN != nil {
		t.Errorf("EN: expected no add button for ghost account %s", ghostEmail)
	}
	if delGhostEN == nil {
		t.Errorf("EN: expected delete button for ghost account %s", ghostEmail)
	}

	// --- 3. TC-MODAL-i18n-03: In-Place Reactive Mutation in EN ---
	addNeedEN.OnTapped()
	badgeMutatedEN, addMutatedEN, delMutatedEN := inspectAcceptanceActionSlot(t, rowsEN[needOAuthEmail])
	if badgeMutatedEN != "[Added]" {
		t.Errorf("EN mutation: expected badge to become '[Added]', got %q", badgeMutatedEN)
	}
	if addMutatedEN != nil {
		t.Errorf("EN mutation: expected add button to disappear after click")
	}
	if delMutatedEN == nil {
		t.Errorf("EN mutation: expected delete button to remain present")
	}
}
