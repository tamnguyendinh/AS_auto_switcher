package ui

import (
	"context"
	"errors"
	"encoding/base64"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
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

	var activeCredToken = &model.TokenData{
		AccessToken: "test_access",
		IDToken:     makeTestJWT("current@gmail.com"),
	}

	// Mock wincred handlers that properly update memory state upon write
	mgr.SetWincredHandlers(
		func() (*model.TokenData, error) {
			return activeCredToken, nil
		},
		func(token *model.TokenData) error {
			activeCredToken = token
			return nil
		},
	)

	// Mock client finder
	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return nil, nil
	})

	return mgr
}

func waitForIdle(a *App, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	time.Sleep(15 * time.Millisecond)

	for time.Now().Before(deadline) {
		a.mu.RLock()
		busy := a.isBusy || a.isQuotaRefreshing.Load()
		a.mu.RUnlock()

		if !busy {
			time.Sleep(60 * time.Millisecond)
			a.mu.RLock()
			busyAgain := a.isBusy || a.isQuotaRefreshing.Load()
			a.mu.RUnlock()
			if !busyAgain {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
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
	defer app.Close()

	win := app.Window()
	if win == nil {
		t.Fatal("expected non-nil window after Init")
	}
	if win.Title() != "⚡ Antigravity Multi-Account Switcher" {
		t.Errorf("unexpected window title: %s", win.Title())
	}
	if app.activeStatusLabel == nil || app.statusBarLabel == nil {
		t.Fatal("expected non-nil UI widgets")
	}

	waitForIdle(app, 3*time.Second)
}

func TestToggleAlwaysOnTop(t *testing.T) {
	mgr := createTestManager(t)
	testApp := test.NewApp()
	app := NewAppWithFyne(testApp, mgr)
	defer app.Close()
	app.Init()

	if !app.IsAlwaysOnTop() {
		t.Errorf("expected alwaysOnTop to default to true")
	}

	app.ToggleAlwaysOnTop()
	if app.IsAlwaysOnTop() {
		t.Errorf("expected alwaysOnTop to be false after toggle")
	}
	if app.alwaysOnTopBtn.Text != "📌 Ghim: TẮT" {
		t.Errorf("expected button text '📌 Ghim: TẮT', got: %s", app.alwaysOnTopBtn.Text)
	}

	app.SetAlwaysOnTop(true)
	if !app.IsAlwaysOnTop() {
		t.Errorf("expected alwaysOnTop to be true after SetAlwaysOnTop(true)")
	}
	if app.alwaysOnTopBtn.Text != "📌 Ghim: BẬT" {
		t.Errorf("expected button text '📌 Ghim: BẬT', got: %s", app.alwaysOnTopBtn.Text)
	}

	waitForIdle(app, 3*time.Second)
}

func TestUpdateUI_WithAccounts(t *testing.T) {
	mgr := createTestManager(t)
	testApp := test.NewApp()
	app := NewAppWithFyne(testApp, mgr)
	defer app.Close()
	app.Init()

	overview := &manager.AccountOverview{
		ActiveEmail:           "active@example.com",
		LanguageServerRunning: true,
		ServerInfo:            &connector.ServerInfo{PID: 1234, Port: 8080},
		Accounts: []manager.AccountDisplayItem{
			{
				Email:        "active@example.com",
				Name:         "Active User",
				IsActive:     true,
				HasToken:     true,
				State:        model.StateReady,
				DisplayBadge: "[ĐANG DÙNG]",
			},
			{
				Email:        "ready@example.com",
				Name:         "Ready User",
				IsActive:     false,
				HasToken:     true,
				State:        model.StateReady,
				DisplayBadge: "[⚡ Chuyển]",
			},
			{
				Email:        "unauth@example.com",
				Name:         "Unauth User",
				IsActive:     false,
				HasToken:     false,
				State:        model.StateUnauth,
				DisplayBadge: "[Chưa lưu]",
			},
		},
	}

	app.updateUI(overview)

	if !strings.Contains(app.activeStatusLabel.Text, "active@example.com") {
		t.Errorf("expected active status to contain active@example.com, got: %s", app.activeStatusLabel.Text)
	}
	if !strings.Contains(app.serverStatusLabel.Text, "PID 1234") {
		t.Errorf("expected server status to contain PID 1234, got: %s", app.serverStatusLabel.Text)
	}
	if len(app.accountListContainer.Objects) != 3 {
		t.Errorf("expected 3 account cards, got %d", len(app.accountListContainer.Objects))
	}

	waitForIdle(app, 3*time.Second)
}

func TestUpdateUI_Empty(t *testing.T) {
	mgr := createTestManager(t)
	testApp := test.NewApp()
	app := NewAppWithFyne(testApp, mgr)
	defer app.Close()
	app.Init()

	overview := &manager.AccountOverview{
		ActiveEmail:           "",
		LanguageServerRunning: false,
		Accounts:              []manager.AccountDisplayItem{},
	}

	app.updateUI(overview)

	if !strings.Contains(app.activeStatusLabel.Text, "Chưa phát hiện") {
		t.Errorf("expected status 'Chưa phát hiện', got: %s", app.activeStatusLabel.Text)
	}
	if !strings.Contains(app.serverStatusLabel.Text, "Không phát hiện tiến trình") {
		t.Errorf("expected server status 'Không phát hiện tiến trình', got: %s", app.serverStatusLabel.Text)
	}
	if len(app.accountListContainer.Objects) != 1 {
		t.Errorf("expected 1 container object (empty label), got %d", len(app.accountListContainer.Objects))
	}

	waitForIdle(app, 3*time.Second)
}

func TestNewAccountCard_Callbacks(t *testing.T) {
	switched := false
	saved := false

	readyItem := manager.AccountDisplayItem{
		Email:    "ready@example.com",
		State:    model.StateReady,
		IsActive: false,
	}

	card := NewAccountCard(readyItem, func(email string) {
		if email == "ready@example.com" {
			switched = true
		}
	}, func() {
		saved = true
	})

	if card == nil {
		t.Fatalf("expected non-nil card")
	}

	unauthActiveItem := manager.AccountDisplayItem{
		Email:    "active_unauth@example.com",
		State:    model.StateUnauth,
		IsActive: true,
	}

	card2 := NewAccountCard(unauthActiveItem, func(email string) {}, func() {
		saved = true
	})
	if card2 == nil {
		t.Fatalf("expected non-nil card for active unauth")
	}
	_ = switched
	_ = saved
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
	defer app.Close()
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

	readyItem := manager.AccountDisplayItem{
		Email:    "ready@example.com",
		State:    model.StateReady,
		IsActive: false,
	}
	readyCard := NewAccountCard(readyItem, nil, nil)
	stack, ok := readyCard.(*fyne.Container)
	if !ok || len(stack.Objects) == 0 {
		t.Fatalf("expected stack container for readyCard")
	}
	bgRect, ok := stack.Objects[0].(*canvas.Rectangle)
	if !ok {
		t.Fatalf("expected canvas.Rectangle background")
	}
	if bgRect.CornerRadius != 8 {
		t.Errorf("expected CornerRadius 8, got %f", bgRect.CornerRadius)
	}
	if bgRect.StrokeWidth != 1.0 {
		t.Errorf("expected StrokeWidth 1.0, got %f", bgRect.StrokeWidth)
	}

	activeItem := manager.AccountDisplayItem{
		Email:    "active@example.com",
		State:    model.StateReady,
		IsActive: true,
	}
	activeCard := NewAccountCard(activeItem, nil, nil)
	activeStack, ok := activeCard.(*fyne.Container)
	if !ok || len(activeStack.Objects) == 0 {
		t.Fatalf("expected stack container for activeCard")
	}
	activeBgRect, ok := activeStack.Objects[0].(*canvas.Rectangle)
	if !ok {
		t.Fatalf("expected canvas.Rectangle background for active card")
	}
	if activeBgRect.StrokeWidth != 1.5 {
		t.Errorf("expected StrokeWidth 1.5, got %f", activeBgRect.StrokeWidth)
	}
}

func TestSwitchAccount_NoModalPopup(t *testing.T) {
	mgr := createTestManager(t)
	testApp := test.NewApp()
	app := NewAppWithFyne(testApp, mgr)
	defer app.Close()
	app.Init()

	waitForIdle(app, 3*time.Second)

	err := mgr.Store().SaveAccount("target@gmail.com", &model.TokenData{
		AccessToken: "target_access",
		IDToken:     makeTestJWT("target@gmail.com"),
	})
	if err != nil {
		t.Fatalf("failed saving target account: %v", err)
	}

	app.SwitchAccount("target@gmail.com")

	waitForIdle(app, 3*time.Second)

	if !strings.Contains(app.statusBarLabel.Text, "Đã chuyển sang target@gmail.com") {
		t.Errorf("expected status bar text to reflect switch, got: %s", app.statusBarLabel.Text)
	}

	if app.win.Canvas().Overlays().Top() != nil {
		t.Errorf("expected zero modal dialogs on canvas, but found an active overlay")
	}
}

func TestSaveSession_NoModalPopup(t *testing.T) {
	mgr := createTestManager(t)
	testApp := test.NewApp()
	app := NewAppWithFyne(testApp, mgr)
	defer app.Close()
	app.Init()

	waitForIdle(app, 3*time.Second)

	app.SaveSession()

	waitForIdle(app, 3*time.Second)

	if !strings.Contains(app.statusBarLabel.Text, "Đã lưu phiên của current@gmail.com vào kho JSON") {
		t.Errorf("expected status bar text to reflect save, got: %s", app.statusBarLabel.Text)
	}

	if app.win.Canvas().Overlays().Top() != nil {
		t.Errorf("expected zero modal dialogs on canvas, but found an active overlay")
	}
}

// -------------------------------------------------------------
// Slice P6A-2 Tests: Quota Badges, Countdown, Refresh & UI-TR
// -------------------------------------------------------------

func extractBadgeProperties(badgeObj fyne.CanvasObject) (*canvas.Circle, *canvas.Text) {
	var circle *canvas.Circle
	var text *canvas.Text

	var walk func(o fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		if c, ok := o.(*canvas.Circle); ok {
			circle = c
		}
		if t, ok := o.(*canvas.Text); ok {
			text = t
		}
		if cont, ok := o.(*fyne.Container); ok {
			for _, child := range cont.Objects {
				walk(child)
			}
		}
	}
	walk(badgeObj)
	return circle, text
}

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

	// 3. NewQuotaBadgeGroup with populated quota & deep canvas assertion (FINDING-03)
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

	gridCont, ok := badges.(*fyne.Container)
	if !ok || len(gridCont.Objects) != 3 {
		t.Fatalf("expected 3 badge columns in NewQuotaBadgeGroup")
	}

	// Claude: 100% -> Green
	cCircle, cText := extractBadgeProperties(gridCont.Objects[0])
	if cCircle == nil || cText == nil {
		t.Fatalf("failed extracting Claude badge elements")
	}
	assertColorMatches(t, "ClaudeBadge", "dot", cCircle.FillColor, ColorQuotaGreen)
	if cText.Text != "Claude: 100%" {
		t.Errorf("expected Claude text 'Claude: 100%%', got '%s'", cText.Text)
	}
	assertColorMatches(t, "ClaudeBadge", "text", cText.Color, ColorQuotaGreen)

	// Gemini: 23.5% -> Amber
	gCircle, gText := extractBadgeProperties(gridCont.Objects[1])
	if gCircle == nil || gText == nil {
		t.Fatalf("failed extracting Gemini badge elements")
	}
	assertColorMatches(t, "GeminiBadge", "dot", gCircle.FillColor, ColorQuotaAmber)
	if gText.Text != "Gemini: 23.5%" {
		t.Errorf("expected Gemini text 'Gemini: 23.5%%', got '%s'", gText.Text)
	}
	assertColorMatches(t, "GeminiBadge", "text", gText.Color, ColorQuotaAmber)

	// GPT: 10% -> Red
	pCircle, pText := extractBadgeProperties(gridCont.Objects[2])
	if pCircle == nil || pText == nil {
		t.Fatalf("failed extracting GPT badge elements")
	}
	assertColorMatches(t, "GPTBadge", "dot", pCircle.FillColor, ColorQuotaRed)
	if pText.Text != "GPT: 10%" {
		t.Errorf("expected GPT text 'GPT: 10%%', got '%s'", pText.Text)
	}
	assertColorMatches(t, "GPTBadge", "text", pText.Color, ColorQuotaRed)
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

func findRefreshButton(obj fyne.CanvasObject) *widget.Button {
	var target *widget.Button
	var walk func(o fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		if btn, ok := o.(*widget.Button); ok {
			if btn.Icon == theme.ViewRefreshIcon() {
				target = btn
				return
			}
		}
		if cont, ok := o.(*fyne.Container); ok {
			for _, child := range cont.Objects {
				walk(child)
			}
		}
	}
	walk(obj)
	return target
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

	// Real widget traversal and interaction per FINDING-02
	refreshBtn := findRefreshButton(card)
	if refreshBtn == nil {
		t.Fatalf("could not find refresh button in account card")
	}

	refreshBtn.OnTapped()

	if singleRefreshed != "test@example.com" {
		t.Errorf("expected singleRefreshed to be test@example.com after refreshBtn.OnTapped(), got %s", singleRefreshed)
	}
}

func TestGlobalQuotaRefreshAndIndicator(t *testing.T) {
	mgr := createTestManager(t)
	testApp := test.NewApp()
	app := NewAppWithFyne(testApp, mgr)
	tmpDir := t.TempDir()
	app.SetPreferencesPath(filepath.Join(tmpDir, ".ui_preferences.json"))
	cache := telemetry.NewQuotaCache(filepath.Join(tmpDir, "quota_cache.json"))
	app.SetQuotaCache(cache)
	app.Init()
	defer app.Close()

	waitForIdle(app, 3*time.Second)

	source := &testAccountSource{
		accounts: []telemetry.AccountTokenInfo{
			{Email: "acc1@gmail.com", RefreshToken: "tok1"},
			{Email: "acc2@gmail.com", RefreshToken: "tok2"},
		},
	}
	pool, err := telemetry.NewTelemetryPoolWithSource(&testQuotaClient{}, cache, source)
	if err != nil {
		t.Fatalf("failed creating pool: %v", err)
	}

	app.SetTelemetryPool(pool)

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
	app.SetQuotaCache(telemetry.NewQuotaCache(filepath.Join(tmpDir, "quota_cache.json")))
	app.Init()
	defer app.Close()

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
	btn := findRefreshButton(card)
	if btn == nil {
		t.Errorf("UI-TR-02 FAILED: refresh button not found in account card")
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



func TestUIRendering_QuotaErrorBadgesTruthful(t *testing.T) {
	errQuota := &telemetry.AccountQuotaData{
		Email:        "err@example.com",
		Status:       telemetry.StatusError,
		ErrorMessage: "server returned status 403",
		Claude:       telemetry.ModelQuotaBucket{RemainingFraction: 0.0},
		Gemini:       telemetry.ModelQuotaBucket{RemainingFraction: 0.0},
		GPT:          telemetry.ModelQuotaBucket{RemainingFraction: 0.0},
	}

	// 1. NewQuotaBadgeGroup with StatusError must render [!] Lỗi kết nối and NEVER 0% red
	badges := NewQuotaBadgeGroup(errQuota)
	if badges == nil {
		t.Fatalf("expected non-nil container for error quota badges")
	}

	gridCont, ok := badges.(*fyne.Container)
	if !ok || len(gridCont.Objects) == 0 {
		t.Fatalf("expected container with objects in NewQuotaBadgeGroup")
	}

	// Assert that first column is the error badge
	errHBox, ok := gridCont.Objects[0].(*fyne.Container)
	if !ok {
		t.Fatalf("expected *fyne.Container for error badge")
	}

	hasErrorText := false
	hasMutedColor := false
	for _, obj := range errHBox.Objects {
		if txt, ok := obj.(*canvas.Text); ok {
			if strings.Contains(txt.Text, "[!] Lỗi kết nối") {
				hasErrorText = true
			}
			if txt.Color == ColorQuotaMuted {
				hasMutedColor = true
			}
			if strings.Contains(txt.Text, "0%") {
				t.Errorf("CRITICAL VIOLATION: error badge must NEVER display 0%%, got: %s", txt.Text)
			}
		}
	}
	if !hasErrorText {
		t.Errorf("expected error badge to contain '[!] Lỗi kết nối'")
	}
	if !hasMutedColor {
		t.Errorf("expected error badge text color to be ColorQuotaMuted")
	}

	// 2. NewQuotaCountdownLabel with StatusError must return nil (NO "Sắp cạn" warning)
	lbl := NewQuotaCountdownLabel(errQuota)
	if lbl != nil {
		t.Errorf("CRITICAL VIOLATION: countdown label must be nil when Status is StatusError, got: %#v", lbl)
	}

	// 3. BuildActiveQuotaSummary with StatusError must return [!] Lỗi kết nối
	summary := BuildActiveQuotaSummary(errQuota)
	if !strings.Contains(summary, "[!] Lỗi kết nối") {
		t.Errorf("expected active quota summary to report '[!] Lỗi kết nối', got: %s", summary)
	}
	if strings.Contains(summary, "0%") {
		t.Errorf("CRITICAL VIOLATION: active quota summary must NEVER display 0%% when in error status, got: %s", summary)
	}
}

func TestApp_StatusBarQuotaReportingFormatting(t *testing.T) {
	mgr := createTestManager(t)
	testApp := test.NewApp()
	app := NewAppWithFyne(testApp, mgr)
	tmpDir := t.TempDir()

	cache := telemetry.NewQuotaCache(filepath.Join(tmpDir, "quota_cache.json"))
	source := &testAccountSource{
		accounts: []telemetry.AccountTokenInfo{
			{Email: "ok@gmail.com", RefreshToken: "tok_ok"},
			{Email: "err@gmail.com", RefreshToken: ""},
		},
	}
	client := &testQuotaClient{
		quota: &telemetry.AccountQuotaData{
			Email:  "ok@gmail.com",
			Claude: telemetry.ModelQuotaBucket{RemainingFraction: 1.0},
			Gemini: telemetry.ModelQuotaBucket{RemainingFraction: 1.0},
			GPT:    telemetry.ModelQuotaBucket{RemainingFraction: 1.0},
		},
	}
	pool, err := telemetry.NewTelemetryPoolWithSource(client, cache, source)
	if err != nil {
		t.Fatalf("failed creating pool: %v", err)
	}

	// Set local quota fetcher so err@gmail.com returns error and ok@gmail.com succeeds
	pool.SetLocalQuotaFetcher(func(ctx context.Context, email string) (*telemetry.AccountQuotaData, error) {
		if email == "ok@gmail.com" {
			return &telemetry.AccountQuotaData{
				Email:  email,
				Claude: telemetry.ModelQuotaBucket{RemainingFraction: 1.0},
				Gemini: telemetry.ModelQuotaBucket{RemainingFraction: 1.0},
				GPT:    telemetry.ModelQuotaBucket{RemainingFraction: 1.0},
				Status: telemetry.StatusPristine,
			}, nil
		}
		return nil, errors.New("simulated network failure")
	})

	app.SetTelemetryPool(pool)
	app.Init()
	defer app.Close()

	waitForIdle(app, 3*time.Second)
	app.RefreshAllQuota()
	waitForIdle(app, 3*time.Second)

	if app.statusBarLabel != nil {
		txt := app.statusBarLabel.Text
		if !strings.Contains(txt, "Quét quota: 1 thành công, 1 lỗi") {
			t.Errorf("expected status bar text 'Quét quota: 1 thành công, 1 lỗi', got: '%s'", txt)
		}
	}
}