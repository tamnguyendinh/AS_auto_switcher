package ui

import (
	"context"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2/test"

	"antigravity-switch/internal/connector"
	"antigravity-switch/internal/manager"
	"antigravity-switch/internal/model"
	"antigravity-switch/internal/reloader"
	"antigravity-switch/internal/store"
)

func createTestManager(t *testing.T) *manager.AccountManager {
	t.Helper()
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "accounts_store")

	accStore, err := store.NewAccountStore(storePath)
	if err != nil {
		t.Fatalf("failed initializing test store: %v", err)
	}

	sessionReloader := reloader.NewSessionReloader()
	mgr, err := manager.NewAccountManager(accStore, sessionReloader)
	if err != nil {
		t.Fatalf("failed initializing manager: %v", err)
	}

	// Mock wincred handlers
	mgr.SetWincredHandlers(
		func() (*model.TokenData, error) {
			return &model.TokenData{
				AccessToken: "test_access",
				IDToken:     "header.eyBlbWFpbCI6ICJjdXJyZW50QGdtYWlsLmNvbSJ9.sig",
			}, nil
		},
		func(token *model.TokenData) error {
			return nil
		},
	)

	// Mock language server finder
	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return nil, nil
	})

	return mgr
}

func TestNewAppAndInit(t *testing.T) {
	mgr := createTestManager(t)
	testApp := test.NewApp()

	app := NewAppWithFyne(testApp, mgr)
	if app == nil {
		t.Fatal("expected non-nil App")
	}
	if app.Manager() != mgr {
		t.Fatal("expected manager to match")
	}

	app.Init()
	win := app.Window()
	if win == nil {
		t.Fatal("expected non-nil window after Init")
	}
	if win.Title() != "⚡ Antigravity Multi-Account Switcher" {
		t.Errorf("unexpected window title: %s", win.Title())
	}
	if app.activeStatusLabel == nil || app.statusBarLabel == nil {
		t.Fatal("expected status labels to be initialized")
	}
	if app.refreshBtn == nil || app.saveBtn == nil || app.failoverBtn == nil {
		t.Fatal("expected control buttons to be initialized")
	}
}

func TestToggleAutoFailover(t *testing.T) {
	mgr := createTestManager(t)
	testApp := test.NewApp()
	app := NewAppWithFyne(testApp, mgr)
	app.Init()

	if !app.autoFailover {
		t.Error("expected autoFailover to be true initially")
	}

	// Toggle OFF
	app.ToggleAutoFailover()
	if app.autoFailover {
		t.Error("expected autoFailover to be false after first toggle")
	}
	if app.failoverBtn.Text != "⚡ Auto-Failover: TẮT" {
		t.Errorf("unexpected button text: %s", app.failoverBtn.Text)
	}

	// Toggle ON
	app.ToggleAutoFailover()
	if !app.autoFailover {
		t.Error("expected autoFailover to be true after second toggle")
	}
	if app.failoverBtn.Text != "⚡ Auto-Failover: BẬT" {
		t.Errorf("unexpected button text: %s", app.failoverBtn.Text)
	}
}

func TestUpdateUI_WithAccounts(t *testing.T) {
	mgr := createTestManager(t)
	testApp := test.NewApp()
	app := NewAppWithFyne(testApp, mgr)
	app.Init()

	overview := &manager.AccountOverview{
		ActiveEmail: "active@gmail.com",
		LanguageServerRunning: true,
		ServerInfo: &connector.ServerInfo{
			PID:  12345,
			Port: 54321,
		},
		Accounts: []manager.AccountDisplayItem{
			{
				Email:        "active@gmail.com",
				Name:         "Active User",
				State:        model.StateActive,
				IsActive:     true,
				HasToken:     true,
				DisplayBadge: "[ĐANG DÙNG]",
			},
			{
				Email:        "ready@gmail.com",
				Name:         "Ready User",
				State:        model.StateReady,
				IsActive:     false,
				HasToken:     true,
				DisplayBadge: "[⚡ Chuyển]",
			},
			{
				Email:        "unauth@gmail.com",
				State:        model.StateUnauth,
				IsActive:     false,
				HasToken:     false,
				DisplayBadge: "[Chưa lưu]",
			},
		},
	}

	app.updateUI(overview)

	if app.activeStatusLabel.Text != "● Đang hoạt động: active@gmail.com" {
		t.Errorf("unexpected active label: %s", app.activeStatusLabel.Text)
	}
	if app.serverStatusLabel.Text != "● Language Server: PID 12345 | Port 54321 | Sẵn sàng" {
		t.Errorf("unexpected server label: %s", app.serverStatusLabel.Text)
	}
	if len(app.accountListContainer.Objects) != 3 {
		t.Fatalf("expected 3 account cards, got %d", len(app.accountListContainer.Objects))
	}
}

func TestUpdateUI_Empty(t *testing.T) {
	mgr := createTestManager(t)
	testApp := test.NewApp()
	app := NewAppWithFyne(testApp, mgr)
	app.Init()

	overview := &manager.AccountOverview{
		ActiveEmail:           "",
		LanguageServerRunning: false,
		Accounts:              nil,
	}

	app.updateUI(overview)

	if app.activeStatusLabel.Text != "● Chưa phát hiện phiên Antigravity" {
		t.Errorf("unexpected active label: %s", app.activeStatusLabel.Text)
	}
	if len(app.accountListContainer.Objects) != 1 {
		t.Fatalf("expected 1 empty-notice object, got %d", len(app.accountListContainer.Objects))
	}
}

func TestNewAccountCard_Callbacks(t *testing.T) {
	switchedEmail := ""
	onSwitch := func(email string) {
		switchedEmail = email
	}
	savedCalled := false
	onSave := func() {
		savedCalled = true
	}

	// Ready account card
	readyItem := manager.AccountDisplayItem{
		Email: "ready@gmail.com",
		State: model.StateReady,
	}
	card := NewAccountCard(readyItem, onSwitch, onSave)
	if card == nil {
		t.Fatal("expected non-nil card")
	}

	// Active account card
	activeItem := manager.AccountDisplayItem{
		Email:    "active@gmail.com",
		IsActive: true,
		State:    model.StateActive,
	}
	activeCard := NewAccountCard(activeItem, onSwitch, onSave)
	if activeCard == nil {
		t.Fatal("expected non-nil activeCard")
	}

	// Unauth account card
	unauthItem := manager.AccountDisplayItem{
		Email: "unauth@gmail.com",
		State: model.StateUnauth,
	}
	unauthCard := NewAccountCard(unauthItem, onSwitch, onSave)
	if unauthCard == nil {
		t.Fatal("expected non-nil unauthCard")
	}

	_ = savedCalled
	_ = switchedEmail
}

func TestStatusDot(t *testing.T) {
	dot := NewStatusDot(ColorGreen, 12)
	if dot == nil {
		t.Fatal("expected non-nil status dot")
	}
	if dot.MinSize().Width != 12 || dot.MinSize().Height != 12 {
		t.Errorf("unexpected dot size: %v", dot.MinSize())
	}
}

func TestAppLogoResourceAndWindowIcon(t *testing.T) {
	if AppLogoResource == nil {
		t.Fatal("expected AppLogoResource to be non-nil")
	}
	if len(AppLogoResource.Content()) == 0 {
		t.Fatal("expected AppLogoResource to contain image bytes")
	}
	if AppLogoResource.Name() != "app_logo.png" {
		t.Errorf("unexpected resource name: %s", AppLogoResource.Name())
	}

	mgr := createTestManager(t)
	testApp := test.NewApp()
	app := NewAppWithFyne(testApp, mgr)
	app.Init()

	win := app.Window()
	if win == nil {
		t.Fatal("expected non-nil window after Init")
	}
	if app.Logo() != AppLogoResource {
		t.Errorf("expected app logo to match AppLogoResource")
	}
}
