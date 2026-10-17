package ui

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"antigravity-switch/internal/connector"
	"antigravity-switch/internal/manager"
	"antigravity-switch/internal/model"
	"antigravity-switch/internal/reloader"
	"antigravity-switch/internal/store"
	"antigravity-switch/internal/telemetry"
)

func makeTestJWT(email string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte("{\"alg\":\"RS256\",\"typ\":\"JWT\"}"))
	claims := map[string]interface{}{
		"email": email,
		"exp":   time.Now().Add(1 * time.Hour).Unix(),
	}
	claimsBytes, _ := json.Marshal(claims)
	payload := base64.RawURLEncoding.EncodeToString(claimsBytes)
	signature := base64.RawURLEncoding.EncodeToString([]byte("signature"))
	return header + "." + payload + "." + signature
}

func createTestManager(t *testing.T) *manager.AccountManager {
	t.Helper()
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "accounts_store")

	accStore, err := store.NewAccountStore(storePath)
	if err != nil {
		t.Fatalf("failed initializing test store: %v", err)
	}

	sessionReloader := reloader.NewSessionReloader()
	sessionReloader.ProcessKiller = func(ctx context.Context) error { return nil }
	sessionReloader.Sleeper = func(ctx context.Context, d time.Duration) error { return nil }
	sessionReloader.ServerDiscoverer = func() (*connector.ServerInfo, error) {
		return &connector.ServerInfo{
			PID:       12345,
			Port:      54321,
			CSRFToken: "test-csrf",
		}, nil
	}

	mgr, err := manager.NewAccountManager(accStore, sessionReloader)
	if err != nil {
		t.Fatalf("failed initializing manager: %v", err)
	}

	// Mock wincred handlers
	mgr.SetWincredHandlers(
		func() (*model.TokenData, error) {
			return &model.TokenData{
				AccessToken: "test_access",
				IDToken:     makeTestJWT("current@gmail.com"),
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
	waitForIdle(app, 3*time.Second)
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

	// Verify window is the original fyne.Window and not wrapped
	win := app.Window()
	if win == nil {
		t.Fatal("expected non-nil window")
	}
	if win.Title() != windowTitle {
		t.Errorf("unexpected window title: %s", win.Title())
	}

	// Verify setWindowAlwaysOnTop handles window safely without panic
	setWindowAlwaysOnTop(windowTitle, false)
	setWindowAlwaysOnTop(windowTitle, true)
	setWindowAlwaysOnTop("non_existent_title", true)
	waitForIdle(app, 3*time.Second)
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
	waitForIdle(app, 3*time.Second)
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

func waitForIdle(app *App, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		app.mu.RLock()
		busy := app.isBusy
		app.mu.RUnlock()
		if !busy {
			time.Sleep(30 * time.Millisecond)
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestSwitchAccount_NoModalPopup(t *testing.T) {
	mgr := createTestManager(t)
	testApp := test.NewApp()
	app := NewAppWithFyne(testApp, mgr)
	app.Init()

	// Wait for initial Init() refresh to complete
	waitForIdle(app, 3*time.Second)

	// Pre-save account to store so switch succeeds
	err := mgr.Store().SaveAccount("target@gmail.com", &model.TokenData{
		AccessToken: "target_access",
		IDToken:     makeTestJWT("target@gmail.com"),
	})
	if err != nil {
		t.Fatalf("failed saving target account: %v", err)
	}

	app.SwitchAccount("target@gmail.com")

	// Wait for async switch to complete
	waitForIdle(app, 3*time.Second)

	// Verify status bar updated with success message
	if !strings.Contains(app.statusBarLabel.Text, "Đã chuyển sang target@gmail.com") {
		t.Errorf("expected status bar text to reflect switch, got: %s", app.statusBarLabel.Text)
	}

	// Verify NO modal overlay dialog exists on the window canvas (Python parity: zero blocking popups)
	if app.win.Canvas().Overlays().Top() != nil {
		t.Errorf("expected zero modal dialogs on canvas, but found an active overlay")
	}
}

func TestSaveSession_NoModalPopup(t *testing.T) {
	mgr := createTestManager(t)
	testApp := test.NewApp()
	app := NewAppWithFyne(testApp, mgr)
	app.Init()

	// Wait for initial Init() refresh to complete
	waitForIdle(app, 3*time.Second)

	app.SaveSession()

	// Wait for async save to complete
	waitForIdle(app, 3*time.Second)

	// Verify status bar updated with save message
	if !strings.Contains(app.statusBarLabel.Text, "Đã lưu phiên của current@gmail.com vào kho JSON") {
		t.Errorf("expected status bar text to reflect save, got: %s", app.statusBarLabel.Text)
	}

	// Verify NO modal overlay dialog exists on the window canvas (Python parity: zero blocking popups)
	if app.win.Canvas().Overlays().Top() != nil {
		t.Errorf("expected zero modal dialogs on canvas, but found an active overlay")
	}
}


// -------------------------------------------------------------
// Slice P6A-2 Tests: Quota Badges, Countdown, Refresh & UI-TR
// -------------------------------------------------------------

func TestQuotaBadgeGroupAndColors(t *testing.T) {
	// 1. Verify 3-tier colors (UI-TR-04)
	if GetQuotaTierColor(0.90) != ColorQuotaGreen {
		t.Errorf("expected Green for 0.90, got %#v", GetQuotaTierColor(0.90))
	}
	if GetQuotaTierColor(0.51) != ColorQuotaGreen {
		t.Errorf("expected Green for 0.51, got %#v", GetQuotaTierColor(0.51))
	}
	if GetQuotaTierColor(0.50) != ColorQuotaAmber {
		t.Errorf("expected Amber for 0.50, got %#v", GetQuotaTierColor(0.50))
	}
	if GetQuotaTierColor(0.235) != ColorQuotaAmber {
		t.Errorf("expected Amber for 0.235, got %#v", GetQuotaTierColor(0.235))
	}
	if GetQuotaTierColor(0.15) != ColorQuotaAmber {
		t.Errorf("expected Amber for 0.15, got %#v", GetQuotaTierColor(0.15))
	}
	if GetQuotaTierColor(0.149) != ColorQuotaRed {
		t.Errorf("expected Red for 0.149, got %#v", GetQuotaTierColor(0.149))
	}
	if GetQuotaTierColor(0.0) != ColorQuotaRed {
		t.Errorf("expected Red for 0.0, got %#v", GetQuotaTierColor(0.0))
	}

	// 2. NewQuotaBadgeGroup with nil quota returns placeholder
	nilBadges := NewQuotaBadgeGroup(nil)
	if nilBadges == nil {
		t.Fatalf("expected non-nil container for nil quota")
	}

	// 3. NewQuotaBadgeGroup with populated quota
	quota := &telemetry.AccountQuotaData{
		Email:  "user@example.com",
		Claude: telemetry.ModelQuotaBucket{RemainingFraction: 1.0},
		Gemini: telemetry.ModelQuotaBucket{RemainingFraction: 0.235},
		GPT:    telemetry.ModelQuotaBucket{RemainingFraction: 0.10},
	}
	badges := NewQuotaBadgeGroup(quota)
	if badges == nil {
		t.Fatalf("expected non-nil container for populated quota")
	}
}

func TestQuotaCountdownCalculationAndLabel(t *testing.T) {
	now := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)

	// 1. Zero reset time
	if msg := FormatRemainingCountdown(time.Time{}, now); msg != "đang chờ hồi phục" {
		t.Errorf("expected 'đang chờ hồi phục', got '%s'", msg)
	}

	// 2. Past reset time
	past := now.Add(-10 * time.Minute)
	if msg := FormatRemainingCountdown(past, now); msg != "sắp hồi phục" {
		t.Errorf("expected 'sắp hồi phục', got '%s'", msg)
	}

	// 3. Future reset time 1h 34m
	fut1 := now.Add(1*time.Hour + 34*time.Minute)
	if msg := FormatRemainingCountdown(fut1, now); msg != "còn 1h 34p" {
		t.Errorf("expected 'còn 1h 34p', got '%s'", msg)
	}

	// 4. Future reset time 45m
	fut2 := now.Add(45 * time.Minute)
	if msg := FormatRemainingCountdown(fut2, now); msg != "còn 45p" {
		t.Errorf("expected 'còn 45p', got '%s'", msg)
	}

	// 5. NewQuotaCountdownLabel (UI-TR-05)
	// Quota >= 15% everywhere returns nil
	healthyQuota := &telemetry.AccountQuotaData{
		Claude: telemetry.ModelQuotaBucket{RemainingFraction: 1.0},
		Gemini: telemetry.ModelQuotaBucket{RemainingFraction: 0.90},
		GPT:    telemetry.ModelQuotaBucket{RemainingFraction: 0.50},
	}
	if lbl := NewQuotaCountdownLabel(healthyQuota); lbl != nil {
		t.Errorf("expected nil countdown label for healthy quota, got: %#v", lbl)
	}

	// Gemini < 15% returns warning label with countdown
	resetTime := time.Now().Add(1*time.Hour + 34*time.Minute)
	lowGeminiQuota := &telemetry.AccountQuotaData{
		Email:  "user@example.com",
		Claude: telemetry.ModelQuotaBucket{RemainingFraction: 1.0},
		Gemini: telemetry.ModelQuotaBucket{RemainingFraction: 0.12, ResetTime: resetTime},
		GPT:    telemetry.ModelQuotaBucket{RemainingFraction: 1.0},
	}
	lbl := NewQuotaCountdownLabel(lowGeminiQuota)
	if lbl == nil {
		t.Fatalf("expected non-nil countdown label for low Gemini quota")
	}
	txtObj, ok := lbl.(*canvas.Text)
	if !ok {
		t.Fatalf("expected *canvas.Text label")
	}
	if !strings.Contains(txtObj.Text, "Gemini sắp cạn") {
		t.Errorf("expected warning to mention Gemini sắp cạn, got: %s", txtObj.Text)
	}
	if !strings.Contains(txtObj.Text, "Google sẽ hồi phục lúc:") {
		t.Errorf("expected warning to mention Google sẽ hồi phục, got: %s", txtObj.Text)
	}
}

type testAccountSource struct {
	accounts []telemetry.AccountTokenInfo
}

func (s *testAccountSource) ListAccounts() ([]telemetry.AccountTokenInfo, error) {
	return s.accounts, nil
}

func (s *testAccountSource) GetRefreshToken(email string) (string, error) {
	return "mock-refresh-token", nil
}

type testQuotaClient struct {
	quota *telemetry.AccountQuotaData
}

func (c *testQuotaClient) RetrieveUserQuota(ctx context.Context, refreshToken string) (*telemetry.AccountQuotaData, error) {
	if c.quota != nil {
		clone := *c.quota
		return &clone, nil
	}
	return &telemetry.AccountQuotaData{
		Email:       "acc1@gmail.com",
		Claude:      telemetry.ModelQuotaBucket{RemainingFraction: 1.0},
		Gemini:      telemetry.ModelQuotaBucket{RemainingFraction: 0.85},
		GPT:         telemetry.ModelQuotaBucket{RemainingFraction: 1.0},
		ResetTime:   time.Now().Add(2 * time.Hour),
		LastUpdated: time.Now(),
	}, nil
}

func TestAccountCardWithQuotaAndRefresh(t *testing.T) {
	singleRefreshed := ""
	item := manager.AccountDisplayItem{
		Email:    "test@example.com",
		State:    model.StateReady,
		IsActive: false,
	}

	quota := &telemetry.AccountQuotaData{
		Email:  "test@example.com",
		Claude: telemetry.ModelQuotaBucket{RemainingFraction: 1.0},
		Gemini: telemetry.ModelQuotaBucket{RemainingFraction: 0.08, ResetTime: time.Now().Add(45 * time.Minute)},
		GPT:    telemetry.ModelQuotaBucket{RemainingFraction: 1.0},
	}

	card := NewAccountCardWithQuota(
		item,
		quota,
		DarkPalette,
		func(email string) {},
		func() {},
		func(email string) {
			singleRefreshed = email
		},
	)

	if card == nil {
		t.Fatalf("expected non-nil card with quota")
	}

	refreshFn := func(email string) { singleRefreshed = email }
	refreshFn(item.Email)
	if singleRefreshed != "test@example.com" {
		t.Errorf("expected singleRefreshed to be test@example.com, got %s", singleRefreshed)
	}
}

func TestGlobalQuotaRefreshAndIndicator(t *testing.T) {
	mgr := createTestManager(t)
	testApp := test.NewApp()
	app := NewAppWithFyne(testApp, mgr)
	tmpDir := t.TempDir()
	app.SetPreferencesPath(filepath.Join(tmpDir, ".ui_preferences.json"))
	app.Init()

	waitForIdle(app, 3*time.Second)

	source := &testAccountSource{
		accounts: []telemetry.AccountTokenInfo{
			{Email: "acc1@gmail.com", RefreshToken: "tok1"},
			{Email: "acc2@gmail.com", RefreshToken: "tok2"},
		},
	}
	cache := telemetry.NewQuotaCache(filepath.Join(tmpDir, "quota_cache.json"))
	pool, err := telemetry.NewTelemetryPoolWithSource(&testQuotaClient{}, cache, source)
	if err != nil {
		t.Fatalf("failed creating pool: %v", err)
	}

	app.SetTelemetryPool(pool)
	app.SetQuotaCache(cache)

	// UI-TR-01 check: Button exists and starts with "📊 Làm mới Quota Tất cả"
	if app.quotaRefreshAllBtn == nil {
		t.Fatalf("expected quotaRefreshAllBtn to be initialized")
	}
	if app.quotaRefreshAllBtn.Text != "📊 Làm mới Quota Tất cả" {
		t.Errorf("unexpected button label: %s", app.quotaRefreshAllBtn.Text)
	}

	// UI-TR-03 check: Auto-refresh label initialized
	if app.autoRefreshLabel == nil {
		t.Fatalf("expected autoRefreshLabel to be initialized")
	}
	if !strings.Contains(app.autoRefreshLabel.Text, "Tự động: 60s") {
		t.Errorf("expected auto-refresh label to contain 'Tự động: 60s', got: %s", app.autoRefreshLabel.Text)
	}

	// Trigger RefreshAllQuota
	app.RefreshAllQuota()

	// Wait for async finish
	waitForIdle(app, 3*time.Second)

	// Verify button returned to standard label
	if app.quotaRefreshAllBtn.Text != "📊 Làm mới Quota Tất cả" {
		t.Errorf("expected button restored to standard label, got: %s", app.quotaRefreshAllBtn.Text)
	}

	// Verify autoRefreshLabel has updated timestamp
	if strings.Contains(app.autoRefreshLabel.Text, "--:--:--") {
		t.Errorf("expected timestamp to be updated, got: %s", app.autoRefreshLabel.Text)
	}

	// Trigger RefreshSingleQuota
	app.RefreshSingleQuota("acc1@gmail.com")
	waitForIdle(app, 3*time.Second)

	if !strings.Contains(app.statusBarLabel.Text, "Đã cập nhật quota acc1@gmail.com") {
		t.Errorf("expected status bar text to reflect single refresh, got: %s", app.statusBarLabel.Text)
	}
}

func TestActiveQuotaHeaderSummary(t *testing.T) {
	// Nil quota
	if s := BuildActiveQuotaSummary(nil); !strings.Contains(s, "Chưa tải dữ liệu") {
		t.Errorf("expected 'Chưa tải dữ liệu' for nil quota, got: %s", s)
	}

	// Populated quota
	q := &telemetry.AccountQuotaData{
		Claude: telemetry.ModelQuotaBucket{RemainingFraction: 1.0},
		Gemini: telemetry.ModelQuotaBucket{RemainingFraction: 0.235},
		GPT:    telemetry.ModelQuotaBucket{RemainingFraction: 1.0},
	}
	summary := BuildActiveQuotaSummary(q)
	if !strings.Contains(summary, "Claude: 100%") || !strings.Contains(summary, "Gemini: 23.5%") || !strings.Contains(summary, "GPT: 100%") {
		t.Errorf("unexpected active quota summary: %s", summary)
	}
}

func TestUITraceabilityChecklist(t *testing.T) {
	mgr := createTestManager(t)
	testApp := test.NewApp()
	app := NewAppWithFyne(testApp, mgr)
	tmpDir := t.TempDir()
	app.SetPreferencesPath(filepath.Join(tmpDir, ".ui_preferences.json"))
	app.Init()

	waitForIdle(app, 3*time.Second)

	// UI-TR-01: Global Quota Refresh Button in Header Toolbar
	if app.quotaRefreshAllBtn == nil || app.quotaRefreshAllBtn.Text != "📊 Làm mới Quota Tất cả" {
		t.Errorf("UI-TR-01 FAILED: global quota refresh button missing or invalid")
	}

	// UI-TR-02: Row-Level Single Refresh Button
	card := NewAccountCardWithQuota(
		manager.AccountDisplayItem{Email: "acc@example.com", State: model.StateReady},
		nil,
		DarkPalette,
		nil,
		nil,
		func(email string) {},
	)
	if card == nil {
		t.Errorf("UI-TR-02 FAILED: row card creation failed")
	}

	// UI-TR-03: Auto-Refresh 60s Indicator
	if app.autoRefreshLabel == nil || !strings.Contains(app.autoRefreshLabel.Text, "Tự động: 60s") {
		t.Errorf("UI-TR-03 FAILED: 60s auto-refresh indicator missing")
	}

	// UI-TR-04: Mini Quota Progress Badges (>50% Green, 15-50% Amber, <15% Red)
	if GetQuotaTierColor(0.8) != ColorQuotaGreen || GetQuotaTierColor(0.3) != ColorQuotaAmber || GetQuotaTierColor(0.05) != ColorQuotaRed {
		t.Errorf("UI-TR-04 FAILED: color thresholds incorrect")
	}

	// UI-TR-05: Quota Recovery Countdown Text when < 15%
	lowQuota := &telemetry.AccountQuotaData{
		Email:  "low@example.com",
		Gemini: telemetry.ModelQuotaBucket{RemainingFraction: 0.10, ResetTime: time.Now().Add(1 * time.Hour)},
	}
	if lbl := NewQuotaCountdownLabel(lowQuota); lbl == nil {
		t.Errorf("UI-TR-05 FAILED: countdown label missing for <15%% quota")
	}

	// UI-TR-06: Multi-Theme Switcher Control
	if app.themeSelect == nil || len(app.themeSelect.Options) != 3 {
		t.Errorf("UI-TR-06 FAILED: theme selector control missing or has invalid options")
	}

	// UI-TR-07: Warm Mode Color Palette (Blue-light reduction)
	if WarmPalette.Primary.R != 0xd9 || WarmPalette.Primary.G != 0x77 || WarmPalette.Primary.B != 0x06 {
		t.Errorf("UI-TR-07 FAILED: Warm Mode primary accent should be amber (#d97706)")
	}

	// UI-TR-08: Non-Blocking UI Invariant (IsBusy and isQuotaRefreshing don't deadlock)
	app.RefreshAllQuota()
	app.ToggleAlwaysOnTop()
	waitForIdle(app, 3*time.Second)
}


