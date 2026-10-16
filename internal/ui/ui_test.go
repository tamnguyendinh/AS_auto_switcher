package ui

import (
	"context"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

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
	if app.refreshBtn == nil || app.saveBtn == nil || app.alwaysOnTopBtn == nil {
		t.Fatal("expected control buttons to be initialized")
	}

	if app.headerCard == nil || len(app.headerCard.Objects) == 0 {
		t.Fatal("expected headerCard to have objects")
	}
	headerBg, ok := app.headerCard.Objects[0].(*canvas.Rectangle)
	if !ok {
		t.Fatal("expected headerBg to be canvas.Rectangle")
	}
	if headerBg.StrokeColor != ColorBorderSubtle || headerBg.StrokeWidth != 1 || headerBg.CornerRadius != 8 {
		t.Errorf("unexpected headerBg styling: strokeColor=%#v strokeWidth=%f cornerRadius=%f", headerBg.StrokeColor, headerBg.StrokeWidth, headerBg.CornerRadius)
	}
}

func TestToggleAlwaysOnTop(t *testing.T) {
	mgr := createTestManager(t)
	testApp := test.NewApp()
	app := NewAppWithFyne(testApp, mgr)
	app.Init()

	if !app.IsAlwaysOnTop() {
		t.Error("expected alwaysOnTop to be true initially")
	}
	if app.alwaysOnTopBtn.Text != "📌 Ghim: BẬT" {
		t.Errorf("unexpected button text: %s", app.alwaysOnTopBtn.Text)
	}
	if app.alwaysOnTopBtn.Importance != widget.HighImportance {
		t.Errorf("unexpected button importance: %v", app.alwaysOnTopBtn.Importance)
	}

	// Toggle OFF
	app.ToggleAlwaysOnTop()
	if app.IsAlwaysOnTop() {
		t.Error("expected alwaysOnTop to be false after first toggle")
	}
	if app.alwaysOnTopBtn.Text != "📌 Ghim: TẮT" {
		t.Errorf("unexpected button text: %s", app.alwaysOnTopBtn.Text)
	}
	if app.alwaysOnTopBtn.Importance != widget.MediumImportance {
		t.Errorf("unexpected button importance: %v", app.alwaysOnTopBtn.Importance)
	}

	// Toggle ON
	app.ToggleAlwaysOnTop()
	if !app.IsAlwaysOnTop() {
		t.Error("expected alwaysOnTop to be true after second toggle")
	}
	if app.alwaysOnTopBtn.Text != "📌 Ghim: BẬT" {
		t.Errorf("unexpected button text: %s", app.alwaysOnTopBtn.Text)
	}
	if app.alwaysOnTopBtn.Importance != widget.HighImportance {
		t.Errorf("unexpected button importance: %v", app.alwaysOnTopBtn.Importance)
	}

	// Direct SetAlwaysOnTop verification
	app.SetAlwaysOnTop(false)
	if app.IsAlwaysOnTop() {
		t.Error("expected alwaysOnTop to be false after SetAlwaysOnTop(false)")
	}
	if app.alwaysOnTopBtn.Text != "📌 Ghim: TẮT" {
		t.Errorf("unexpected button text: %s", app.alwaysOnTopBtn.Text)
	}

	app.SetAlwaysOnTop(true)
	if !app.IsAlwaysOnTop() {
		t.Error("expected alwaysOnTop to be true after SetAlwaysOnTop(true)")
	}
	if app.alwaysOnTopBtn.Text != "📌 Ghim: BẬT" {
		t.Errorf("unexpected button text: %s", app.alwaysOnTopBtn.Text)
	}

	// Verify window interface SetAlwaysOnTop
	if customWin, ok := app.Window().(Window); ok {
		customWin.SetAlwaysOnTop(false)
		if customWin.IsAlwaysOnTop() {
			t.Error("expected customWin.IsAlwaysOnTop to be false")
		}
		customWin.SetAlwaysOnTop(true)
		if !customWin.IsAlwaysOnTop() {
			t.Error("expected customWin.IsAlwaysOnTop to be true")
		}
	} else {
		t.Fatal("expected app.Window() to implement Window interface")
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

func TestElevationTokensAndBorders(t *testing.T) {
	// 1. Verify color tokens
	if ColorDarkBg.R != 0x11 || ColorDarkBg.G != 0x11 || ColorDarkBg.B != 0x1b || ColorDarkBg.A != 0xff {
		t.Errorf("unexpected ColorDarkBg: %#v", ColorDarkBg)
	}
	if ColorCard.R != 0x18 || ColorCard.G != 0x18 || ColorCard.B != 0x25 || ColorCard.A != 0xff {
		t.Errorf("unexpected ColorCard: %#v", ColorCard)
	}
	if ColorCardActive.R != 0x1e || ColorCardActive.G != 0x1e || ColorCardActive.B != 0x2e || ColorCardActive.A != 0xff {
		t.Errorf("unexpected ColorCardActive: %#v", ColorCardActive)
	}
	if ColorBorderSubtle.R != 0x31 || ColorBorderSubtle.G != 0x32 || ColorBorderSubtle.B != 0x44 || ColorBorderSubtle.A != 0xff {
		t.Errorf("unexpected ColorBorderSubtle: %#v", ColorBorderSubtle)
	}
	if ColorBorderActive.R != 0xa6 || ColorBorderActive.G != 0xe3 || ColorBorderActive.B != 0xa1 || ColorBorderActive.A != 0xdd {
		t.Errorf("unexpected ColorBorderActive: %#v", ColorBorderActive)
	}

	// 2. Verify Ready account card background
	readyItem := manager.AccountDisplayItem{
		Email:    "ready@example.com",
		State:    model.StateReady,
		IsActive: false,
	}
	readyCard := NewAccountCard(readyItem, nil, nil)
	stack, ok := readyCard.(*fyne.Container)
	if !ok || len(stack.Objects) == 0 {
		t.Fatalf("expected container stack for account card")
	}
	rect, ok := stack.Objects[0].(*canvas.Rectangle)
	if !ok {
		t.Fatalf("expected canvas.Rectangle background")
	}
	if rect.CornerRadius != 8 {
		t.Errorf("expected ready card CornerRadius 8, got %f", rect.CornerRadius)
	}
	if rect.FillColor != ColorCard {
		t.Errorf("expected ready card FillColor ColorCard, got %#v", rect.FillColor)
	}
	if rect.StrokeColor != ColorBorderSubtle {
		t.Errorf("expected ready card StrokeColor ColorBorderSubtle, got %#v", rect.StrokeColor)
	}
	if rect.StrokeWidth != 1.0 {
		t.Errorf("expected ready card StrokeWidth 1.0, got %f", rect.StrokeWidth)
	}

	// 3. Verify Active account card background
	activeItem := manager.AccountDisplayItem{
		Email:    "active@example.com",
		State:    model.StateActive,
		IsActive: true,
	}
	activeCard := NewAccountCard(activeItem, nil, nil)
	activeStack, ok := activeCard.(*fyne.Container)
	if !ok || len(activeStack.Objects) == 0 {
		t.Fatalf("expected container stack for active account card")
	}
	activeRect, ok := activeStack.Objects[0].(*canvas.Rectangle)
	if !ok {
		t.Fatalf("expected canvas.Rectangle background for active card")
	}
	if activeRect.CornerRadius != 8 {
		t.Errorf("expected active card CornerRadius 8, got %f", activeRect.CornerRadius)
	}
	if activeRect.FillColor != ColorCardActive {
		t.Errorf("expected active card FillColor ColorCardActive, got %#v", activeRect.FillColor)
	}
	if activeRect.StrokeColor != ColorBorderActive {
		t.Errorf("expected active card StrokeColor ColorBorderActive, got %#v", activeRect.StrokeColor)
	}
	if activeRect.StrokeWidth != 1.5 {
		t.Errorf("expected active card StrokeWidth 1.5, got %f", activeRect.StrokeWidth)
	}
}
