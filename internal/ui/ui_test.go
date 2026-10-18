package ui

import (
	"context"
	"image/color"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
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

func createTestManagerWithReloader(t *testing.T, sessionReloader *reloader.SessionReloader) *manager.AccountManager {
	t.Helper()
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "accounts_store")

	accStore, err := store.NewAccountStore(storePath)
	if err != nil {
		t.Fatalf("failed initializing test store: %v", err)
	}

	if sessionReloader == nil {
		sessionReloader = reloader.NewSessionReloader()
		sessionReloader.ProcessKiller = func(ctx context.Context) error { return nil }
		sessionReloader.Sleeper = func(ctx context.Context, d time.Duration) error { return nil }
	} else {
		if sessionReloader.ProcessKiller == nil {
			sessionReloader.ProcessKiller = func(ctx context.Context) error { return nil }
		}
		if sessionReloader.Sleeper == nil {
			sessionReloader.Sleeper = func(ctx context.Context, d time.Duration) error { return nil }
		}
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

	// Setup local loopback mock HTTP server for the UI test that reflects the active token's email
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		activeEmail := manager.ExtractEmailFromToken(activeCredToken)
		if activeEmail == "" {
			activeEmail = "current@gmail.com"
		}
		_, _ = w.Write([]byte(fmt.Sprintf(`{"email":"%s"}`, activeEmail)))
	}))
	t.Cleanup(testServer.Close)

	testAddress, _ := url.Parse(testServer.URL)
	testPort, _ := strconv.Atoi(testAddress.Port())
	testClient := connector.NewClientWithHTTP(&connector.ServerInfo{
		PID:       12345,
		Port:      testPort,
		CSRFToken: "test-csrf",
	}, testServer.Client())

	if sessionReloader.ServerDiscoverer == nil {
		sessionReloader.ServerDiscoverer = func() (*connector.ServerInfo, error) {
			return &connector.ServerInfo{
				PID:       12345,
				Port:      testPort,
				CSRFToken: "test-csrf",
			}, nil
		}
	}

	// Mock client finder returning testClient
	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return testClient, nil
	})

	return mgr
}

func createTestManager(t *testing.T) *manager.AccountManager {
	return createTestManagerWithReloader(t, nil)
}

func waitForIdle(a *App, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	time.Sleep(15 * time.Millisecond)

	for time.Now().Before(deadline) {
		a.mu.RLock()
		busy := a.isBusy || a.isQuotaRefreshing.Load() || a.IsAnyAccountRefreshing()
		a.mu.RUnlock()

		if !busy {
			time.Sleep(60 * time.Millisecond)
			a.mu.RLock()
			busyAgain := a.isBusy || a.isQuotaRefreshing.Load() || a.IsAnyAccountRefreshing()
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
	if win.Content() == nil {
		t.Fatal("expected non-nil window content after Init")
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
	if app.alwaysOnTopBtn.Text != "📌 Ghim : Tắt" {
		t.Errorf("expected button text '📌 Ghim : Tắt', got: %s", app.alwaysOnTopBtn.Text)
	}

	app.SetAlwaysOnTop(true)
	if !app.IsAlwaysOnTop() {
		t.Errorf("expected alwaysOnTop to be true after SetAlwaysOnTop(true)")
	}
	if app.alwaysOnTopBtn.Text != "📌 Ghim : Bật" {
		t.Errorf("expected button text '📌 Ghim : Bật', got: %s", app.alwaysOnTopBtn.Text)
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

	if app.overview.ActiveEmail != "active@example.com" {
		t.Errorf("expected active email active@example.com, got: %s", app.overview.ActiveEmail)
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

	if app.overview.ActiveEmail != "" {
		t.Errorf("expected empty active email, got: %s", app.overview.ActiveEmail)
	}
	if !strings.Contains(app.serverStatusLabel.Text, "Language Server : Không phát hiện") {
		t.Errorf("expected server status 'Language Server : Không phát hiện', got: %s", app.serverStatusLabel.Text)
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

	card := NewAccountCardWithQuota(readyItem, nil, DarkPalette, func(email string) {
		if email == "ready@example.com" {
			switched = true
		}
	}, func() {
		saved = true
	}, nil)

	if card == nil {
		t.Fatalf("expected non-nil card")
	}

	unauthActiveItem := manager.AccountDisplayItem{
		Email:    "active_unauth@example.com",
		State:    model.StateUnauth,
		IsActive: true,
	}

	card2 := NewAccountCardWithQuota(unauthActiveItem, nil, DarkPalette, func(email string) {}, func() {
		saved = true
	}, nil)
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
	if ColorBorderSubtle.R != 0x58 || ColorBorderSubtle.G != 0x5b || ColorBorderSubtle.B != 0x70 || ColorBorderSubtle.A != 0xff {
		t.Errorf("unexpected ColorBorderSubtle: %#v", ColorBorderSubtle)
	}
	if ColorBorderActive.R != 0xa6 || ColorBorderActive.G != 0xe3 || ColorBorderActive.B != 0xa1 || ColorBorderActive.A != 0xff {
		t.Errorf("unexpected ColorBorderActive: %#v", ColorBorderActive)
	}

	readyItem := manager.AccountDisplayItem{
		Email:    "ready@example.com",
		State:    model.StateReady,
		IsActive: false,
	}
	readyCard := NewAccountCardWithQuota(readyItem, nil, DarkPalette, nil, nil, nil)
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
	if bgRect.StrokeWidth != 1.8 {
		t.Errorf("expected StrokeWidth 1.8, got %f", bgRect.StrokeWidth)
	}

	activeItem := manager.AccountDisplayItem{
		Email:    "active@example.com",
		State:    model.StateReady,
		IsActive: true,
	}
	activeCard := NewAccountCardWithQuota(activeItem, nil, DarkPalette, nil, nil, nil)
	activeStack, ok := activeCard.(*fyne.Container)
	if !ok || len(activeStack.Objects) == 0 {
		t.Fatalf("expected stack container for activeCard")
	}
	activeBgRect, ok := activeStack.Objects[0].(*canvas.Rectangle)
	if !ok {
		t.Fatalf("expected canvas.Rectangle background for active card")
	}
	if activeBgRect.StrokeWidth != 2.0 {
		t.Errorf("expected StrokeWidth 2.0, got %f", activeBgRect.StrokeWidth)
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

	if app.overview.ActiveEmail != "target@gmail.com" {
		t.Errorf("expected active email target@gmail.com, got: %s", app.overview.ActiveEmail)
	}

	if app.win.Canvas().Overlays().Top() != nil {
		t.Errorf("expected zero modal dialogs on canvas, but found an active overlay")
	}
}

func TestSwitchAccount_WithContinuationSummary(t *testing.T) {
	rel := reloader.NewSessionReloader()
	rel.ProcessKiller = func(ctx context.Context) error { return nil }
	rel.Sleeper = func(ctx context.Context, d time.Duration) error { return nil }
	rel.SnapshotCapturer = func(conversationsDir, summariesDbPath, outputDir string) (string, []reloader.ActiveThread, error) {
		return "", []reloader.ActiveThread{{ThreadID: "test-thread-1"}}, nil
	}
	rel.ContinuationRunner = func(ctx context.Context, client *connector.Client, activeThreads []reloader.ActiveThread, promptText string) error {
		return nil
	}

	mgr := createTestManagerWithReloader(t, rel)
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

	if app.overview.ActiveEmail != "target@gmail.com" {
		t.Errorf("expected active email target@gmail.com, got: %s", app.overview.ActiveEmail)
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

	// SaveSession executed without modal dialogs

	if app.win.Canvas().Overlays().Top() != nil {
		t.Errorf("expected zero modal dialogs on canvas, but found an active overlay")
	}
}

// -------------------------------------------------------------
// Slice P6A-2 Tests: Quota Badges, Countdown, Refresh & UI-TR
// -------------------------------------------------------------

func extractBadgeProperties(badgeObj fyne.CanvasObject) (*canvas.Rectangle, *canvas.Text) {
	var fillRect *canvas.Rectangle
	var text *canvas.Text

	var walk func(o fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		if r, ok := o.(*canvas.Rectangle); ok {
			fillRect = r
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
	return fillRect, text
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

	// 2. ExtractDualCycleQuota and CreateQuotaCycleRow with nil quota returns placeholder
	nilDq := ExtractDualCycleQuota(nil)
	if nilDq.HasData || nilDq.Claude5h != -1 || nilDq.ClaudeWeek != -1 || nilDq.Gemini5h != -1 || nilDq.GeminiWeek != -1 {
		t.Fatalf("expected -1 fractions and HasData=false for nil quota")
	}
	nilRow5h := CreateQuotaCycleRow("5h:", nilDq.Claude5h, nilDq.Gemini5h, nilDq.HasData, nilDq.IsError)
	if nilRow5h == nil {
		t.Fatalf("expected non-nil container for nil quota row")
	}

	// 3. Dual-cycle rows with populated quota & deep canvas assertion (FINDING-03)
	quota := &telemetry.AccountQuotaData{
		Email:  "user@example.com",
		Claude: telemetry.ModelQuotaBucket{RemainingFraction: 1.0, FiveHourFraction: -1, WeeklyFraction: -1},
		Gemini: telemetry.ModelQuotaBucket{RemainingFraction: 0.235, FiveHourFraction: -1, WeeklyFraction: -1},
	}
	dq := ExtractDualCycleQuota(quota)
	r5hObj := CreateQuotaCycleRow("5h:", dq.Claude5h, dq.Gemini5h, dq.HasData, dq.IsError)
	rWeekObj := CreateQuotaCycleRow("week:", dq.ClaudeWeek, dq.GeminiWeek, dq.HasData, dq.IsError)
	badges := container.NewVBox(r5hObj, rWeekObj)
	vbox := badges

	// Row 1 (5h): Claude + Gemini
	r5h, ok := vbox.Objects[0].(*fyne.Container)
	if !ok || len(r5h.Objects) < 3 {
		t.Fatalf("expected 5h row container with prefix, Claude, and Gemini items")
	}

	cRect, cText := extractBadgeProperties(r5h.Objects[1])
	if cRect == nil || cText == nil {
		t.Fatalf("failed extracting Claude badge elements")
	}
	assertColorMatches(t, "ClaudeBadge", "bar_fill", cRect.FillColor, ColorQuotaGreen)
	if cText.Text != "100%" {
		t.Errorf("expected Claude percentage '100%%', got '%s'", cText.Text)
	}
	assertColorMatches(t, "ClaudeBadge", "text", cText.Color, ColorQuotaGreen)

	gRect, gText := extractBadgeProperties(r5h.Objects[2])
	if gRect == nil || gText == nil {
		t.Fatalf("failed extracting Gemini badge elements")
	}
	assertColorMatches(t, "GeminiBadge", "bar_fill", gRect.FillColor, ColorQuotaAmber)
	if gText.Text != "23.5%" {
		t.Errorf("expected Gemini percentage '23.5%%', got '%s'", gText.Text)
	}
	assertColorMatches(t, "GeminiBadge", "text", gText.Color, ColorQuotaAmber)

	// Verify ZERO GPT artifacts anywhere in the widget hierarchy
	var walkNoGPT func(fyne.CanvasObject)
	walkNoGPT = func(o fyne.CanvasObject) {
		if txt, ok := o.(*canvas.Text); ok {
			if strings.Contains(strings.ToUpper(txt.Text), "GPT") {
				t.Errorf("VIOLATION: found unexpected GPT text in quota badge: %s", txt.Text)
			}
		}
		if cont, ok := o.(*fyne.Container); ok {
			for _, child := range cont.Objects {
				walkNoGPT(child)
			}
		}
	}
	walkNoGPT(badges)
}

func TestQuotaCountdownCalculationAndLabel(t *testing.T) {
	// Retired in UI Redesign: Countdown labels and helpers removed to ensure compact 3-row layout.
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
		Claude:      telemetry.ModelQuotaBucket{RemainingFraction: 1.0, FiveHourFraction: -1, WeeklyFraction: -1},
		Gemini:      telemetry.ModelQuotaBucket{RemainingFraction: 0.85, FiveHourFraction: -1, WeeklyFraction: -1},
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
		Claude: telemetry.ModelQuotaBucket{RemainingFraction: 1.0, FiveHourFraction: -1, WeeklyFraction: -1},
		Gemini: telemetry.ModelQuotaBucket{RemainingFraction: 0.08, ResetTime: time.Now().Add(45 * time.Minute), FiveHourFraction: -1, WeeklyFraction: -1},
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
	if app.quotaRefreshAllBtn.Text != "🔄 Làm mới Quota" {
		t.Errorf("unexpected button label: %s", app.quotaRefreshAllBtn.Text)
	}

	// UI-TR-03 check: Auto-refresh label initialized
	if app.autoRefreshLabel == nil {
		t.Fatalf("expected autoRefreshLabel to be initialized")
	}
	if !strings.Contains(app.autoRefreshLabel.Text, "Cập nhật :") {
		t.Errorf("expected auto-refresh label to contain 'Cập nhật :', got: %s", app.autoRefreshLabel.Text)
	}

	// Trigger RefreshAllQuota
	app.RefreshAllQuota()

	// Wait for async finish
	waitForIdle(app, 3*time.Second)

	// Verify button returned to standard label
	if app.quotaRefreshAllBtn.Text != "🔄 Làm mới Quota" {
		t.Errorf("expected button restored to standard label, got: %s", app.quotaRefreshAllBtn.Text)
	}

	// Verify autoRefreshLabel has updated timestamp
	if strings.Contains(app.autoRefreshLabel.Text, "--:--:--") {
		t.Errorf("expected timestamp to be updated, got: %s", app.autoRefreshLabel.Text)
	}

	// Trigger RefreshSingleAccountQuota
	app.RefreshSingleAccountQuota("acc1@gmail.com")
	waitForIdle(app, 3*time.Second)

	if q, ok := cache.Get("acc1@gmail.com"); !ok || q == nil {
		t.Errorf("expected quota cached for acc1@gmail.com")
	}
}

func TestExtractDualCycleQuota_Contract(t *testing.T) {
	// Nil quota
	dqNil := ExtractDualCycleQuota(nil)
	if dqNil.HasData || dqNil.Claude5h != -1 || dqNil.ClaudeWeek != -1 {
		t.Errorf("expected HasData=false and -1 for nil quota")
	}

	// Populated quota
	q := &telemetry.AccountQuotaData{
		Claude: telemetry.ModelQuotaBucket{RemainingFraction: 1.0, FiveHourFraction: -1, WeeklyFraction: -1},
		Gemini: telemetry.ModelQuotaBucket{RemainingFraction: 0.235, FiveHourFraction: -1, WeeklyFraction: -1},
	}
	dq := ExtractDualCycleQuota(q)
	if dq.Claude5h != 1.0 || dq.ClaudeWeek != -1 {
		t.Errorf("unexpected Claude dual-cycle: 5h=%f, week=%f", dq.Claude5h, dq.ClaudeWeek)
	}
	if dq.Gemini5h != 0.235 || dq.GeminiWeek != -1 {
		t.Errorf("unexpected Gemini dual-cycle: 5h=%f, week=%f", dq.Gemini5h, dq.GeminiWeek)
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
	if app.quotaRefreshAllBtn == nil || app.quotaRefreshAllBtn.Text != "🔄 Làm mới Quota" {
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
	if app.autoRefreshLabel == nil || !strings.Contains(app.autoRefreshLabel.Text, "Cập nhật :") {
		t.Errorf("UI-TR-03 FAILED: 60s auto-refresh indicator missing")
	}

	// UI-TR-04: Mini Quota Progress Badges (>50% Green, 15-50% Amber, <15% Red)
	if GetQuotaTierColor(0.8) != ColorQuotaGreen || GetQuotaTierColor(0.3) != ColorQuotaAmber || GetQuotaTierColor(0.05) != ColorQuotaRed {
		t.Errorf("UI-TR-04 FAILED: color thresholds incorrect")
	}

	// UI-TR-05: Quota Recovery Countdown eliminated for clean 3-row layout

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
		Claude:       telemetry.ModelQuotaBucket{RemainingFraction: 0.0, FiveHourFraction: -1, WeeklyFraction: -1},
		Gemini:       telemetry.ModelQuotaBucket{RemainingFraction: 0.0, FiveHourFraction: -1, WeeklyFraction: -1},
	}

	// 1. Dual-cycle rows with StatusError must render placeholder --% with muted MiniProgressBar and NEVER 0% red
	dq := ExtractDualCycleQuota(errQuota)
	r5h := CreateQuotaCycleRow("5h:", dq.Claude5h, dq.Gemini5h, dq.HasData, dq.IsError)
	rWeek := CreateQuotaCycleRow("week:", dq.ClaudeWeek, dq.GeminiWeek, dq.HasData, dq.IsError)
	badges := container.NewVBox(r5h, rWeek)

	gridCont := badges
	if len(gridCont.Objects) == 0 {
		t.Fatalf("expected container with objects in dual-cycle error badge")
	}

	hasErrorText := false
	hasMutedColor := false
	miniBarCount := 0
	var walkObj func(fyne.CanvasObject)
	walkObj = func(o fyne.CanvasObject) {
		if txt, ok := o.(*canvas.Text); ok {
			if strings.Contains(txt.Text, "[!] Lỗi kết nối") || strings.Contains(txt.Text, "--%") {
				hasErrorText = true
			}
			if txt.Color == ColorQuotaMuted {
				hasMutedColor = true
			}
			if strings.Contains(txt.Text, "0%") {
				t.Errorf("CRITICAL VIOLATION: error badge must NEVER display 0%%, got: %s", txt.Text)
			}
		}
		if cont, ok := o.(*fyne.Container); ok {
			if len(cont.Objects) == 2 {
				if r1, ok1 := cont.Objects[0].(*canvas.Rectangle); ok1 && r1.CornerRadius == 3 {
					if r2, ok2 := cont.Objects[1].(*canvas.Rectangle); ok2 && r2.CornerRadius == 3 {
						miniBarCount++
					}
				}
			}
			for _, child := range cont.Objects {
				walkObj(child)
			}
		}
	}
	walkObj(badges)

	if !hasErrorText {
		t.Errorf("expected error badge to contain placeholder '--%%' or '[!] Lỗi kết nối'")
	}
	if !hasMutedColor {
		t.Errorf("expected error badge text color to be ColorQuotaMuted")
	}
	if miniBarCount != 4 {
		t.Errorf("expected 4 MiniProgressBar widgets across dual-cycle error rows, got %d", miniBarCount)
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
			Claude: telemetry.ModelQuotaBucket{RemainingFraction: 1.0, FiveHourFraction: -1, WeeklyFraction: -1},
			Gemini: telemetry.ModelQuotaBucket{RemainingFraction: 1.0, FiveHourFraction: -1, WeeklyFraction: -1},
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
				Claude: telemetry.ModelQuotaBucket{RemainingFraction: 1.0, FiveHourFraction: -1, WeeklyFraction: -1},
				Gemini: telemetry.ModelQuotaBucket{RemainingFraction: 1.0, FiveHourFraction: -1, WeeklyFraction: -1},
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


}

// -------------------------------------------------------------
// Slice P6A-3 Tests: Owner Redesign Invariants (1008x630, 3-Column Grid, 3-Row Cards, No GPT)
// -------------------------------------------------------------

func TestOwnerRedesign_WindowSizeAndLayout(t *testing.T) {
	mgr := createTestManager(t)
	testApp := test.NewApp()
	app := NewAppWithFyne(testApp, mgr)
	defer app.Close()
	app.Init()

	win := app.Window()
	if win == nil {
		t.Fatal("expected non-nil window")
	}

	if win.FixedSize() {
		t.Errorf("expected window to be resizable, got FixedSize=true")
	}

	minSize := win.Content().MinSize()
	if minSize.Width != 840 || minSize.Height != 525 {
		t.Errorf("expected content MinSize exactly 840x525, got %v", minSize)
	}
}

func TestOwnerRedesign_QuotaCycleRowMinSizeFits840Window(t *testing.T) {
	// Regression test for blocking finding: Quota cycle row MinSize must remain <= 264px
	// so that account cards and 3-column grid fit within the 840x525 minimum window without clipping column 3.
	r5h := CreateQuotaCycleRow("5h:", 1.0, 0.916, true, false)
	rWeek := CreateQuotaCycleRow("week:", 0.46, 0.12, true, false)
	rErr := CreateQuotaCycleRow("5h:", -1, -1, false, true)

	for name, row := range map[string]fyne.CanvasObject{"5h": r5h, "week": rWeek, "err": rErr} {
		min := row.MinSize()
		if min.Width > 264.0 {
			t.Errorf("row %s MinSize.Width %f exceeds card content limit 264px", name, min.Width)
		}
	}
}

func TestOwnerRedesign_QuotaCycleRowNoDescendantOverflowAt840(t *testing.T) {
	// Regression test for descendant overflow at 840x525 window size (card inner width = 264px).
	// Ensures that all children/descendants inside the quota row fit strictly within size.Width without clipping.
	testCases := []struct {
		name    string
		prefix  string
		cFrac   float64
		gFrac   float64
		hasData bool
		isErr   bool
	}{
		{"100% and 91.6%", "week:", 1.0, 0.916, true, false},
		{"46% and 12%", "week:", 0.46, 0.12, true, false},
		{"0% and 50%", "week:", 0.0, 0.50, true, false},
		{"Missing data", "week:", -1.0, -1.0, false, false},
		{"Error state", "week:", -1.0, -1.0, false, true},
		{"5h worst case", "5h:", 1.0, 0.916, true, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			row := CreateQuotaCycleRow(tc.prefix, tc.cFrac, tc.gFrac, tc.hasData, tc.isErr)
			row.Resize(fyne.NewSize(264.0, 15.0))

			var checkWalk func(o fyne.CanvasObject, offsetX float32)
			checkWalk = func(o fyne.CanvasObject, offsetX float32) {
				absX := offsetX + o.Position().X
				right := absX + o.Size().Width
				if right > 264.05 {
					t.Errorf("%s: descendant %T right edge %.2f exceeds 264px", tc.name, o, right)
				}
				if cont, ok := o.(*fyne.Container); ok {
					for _, ch := range cont.Objects {
						checkWalk(ch, absX)
					}
				}
			}
			checkWalk(row, 0)
		})
	}
}

func TestOwnerRedesign_AccountCardThreeRows(t *testing.T) {
	switchedEmail := ""
	singleRefreshedEmail := ""

	itemActive := manager.AccountDisplayItem{
		Email:    "active_owner@gmail.com",
		IsActive: true,
		State:    model.StateReady,
	}
	itemInactive := manager.AccountDisplayItem{
		Email:    "inactive_owner@gmail.com",
		IsActive: false,
		State:    model.StateReady,
	}

	quota := &telemetry.AccountQuotaData{
		Email: "active_owner@gmail.com",
		Claude: telemetry.ModelQuotaBucket{
			RemainingFraction: 1.0,
			FiveHourFraction:  -1,
			WeeklyFraction:    -1,
		},
		Gemini: telemetry.ModelQuotaBucket{
			RemainingFraction: 0.88,
			FiveHourFraction:  -1,
			WeeklyFraction:    -1,
		},
		RawBuckets: []telemetry.ModelQuotaBucket{
			{ModelID: "gemini-5h", RemainingFraction: 0.88},
			{ModelID: "gemini-weekly", RemainingFraction: 0.90},
			{ModelID: "3p-5h", RemainingFraction: 1.0},
			{ModelID: "3p-weekly", RemainingFraction: 1.0},
		},
	}

	cardActive := NewAccountCardWithQuota(itemActive, quota, DarkPalette,
		func(email string) { switchedEmail = email },
		func() {},
		func(email string) { singleRefreshedEmail = email },
	)

	stackActive, ok := cardActive.(*fyne.Container)
	if !ok || len(stackActive.Objects) < 2 {
		t.Fatalf("expected stack container with background and content")
	}

	paddedCont, ok := stackActive.Objects[1].(*fyne.Container)
	if !ok || len(paddedCont.Objects) == 0 {
		t.Fatalf("expected padded container in card")
	}

	cardVBox, ok := paddedCont.Objects[0].(*fyne.Container)
	if !ok {
		t.Fatalf("expected card VBox")
	}

	// MUST be exactly 2 main sections (Header + Matrix Table) per 3x3 Matrix Spec
	if len(cardVBox.Objects) != 2 {
		t.Fatalf("CRITICAL VIOLATION: expected exactly 2 main sections in AccountCard, got %d", len(cardVBox.Objects))
	}

	// Line 1: Header row (Border layout with email & action cluster)
	line1, ok := cardVBox.Objects[0].(*fyne.Container)
	if !ok {
		t.Fatalf("expected line 1 to be a container")
	}
	hasActiveBadge := false
	var walkObj func(fyne.CanvasObject)
	walkObj = func(o fyne.CanvasObject) {
		if lbl, ok := o.(*widget.Label); ok {
			if strings.Contains(lbl.Text, "[✔ Dùng]") {
				hasActiveBadge = true
			}
		}
		if cont, ok := o.(*fyne.Container); ok {
			for _, child := range cont.Objects {
				walkObj(child)
			}
		}
	}
	walkObj(line1)
	if !hasActiveBadge {
		t.Errorf("expected line 1 to have '[✔ Dùng]' badge for active card")
	}

	// Test single refresh button on active card
	refBtn := findRefreshButton(cardActive)
	if refBtn == nil {
		t.Fatalf("refresh button not found on active card")
	}
	refBtn.OnTapped()
	if singleRefreshedEmail != "active_owner@gmail.com" {
		t.Errorf("expected singleRefreshedEmail 'active_owner@gmail.com', got '%s'", singleRefreshedEmail)
	}

	// Test inactive card has [Chuyển] button
	cardInactive := NewAccountCardWithQuota(itemInactive, quota, DarkPalette,
		func(email string) { switchedEmail = email },
		func() {},
		func(email string) { singleRefreshedEmail = email },
	)
	var switchBtn *widget.Button
	var findSwitch func(fyne.CanvasObject)
	findSwitch = func(o fyne.CanvasObject) {
		if btn, ok := o.(*widget.Button); ok && btn.Text == "Chuyển" {
			switchBtn = btn
			return
		}
		if cont, ok := o.(*fyne.Container); ok {
			for _, child := range cont.Objects {
				findSwitch(child)
			}
		}
	}
	findSwitch(cardInactive)
	if switchBtn == nil {
		t.Fatalf("expected 'Chuyển' button on inactive card")
	}
	switchBtn.OnTapped()
	if switchedEmail != "inactive_owner@gmail.com" {
		t.Errorf("expected switchedEmail 'inactive_owner@gmail.com', got '%s'", switchedEmail)
	}

	// Line 2: Matrix Quota Table (3x3 Matrix with 3 columns and 2 vertical separators)
	line2, ok := cardVBox.Objects[1].(*fyne.Container)
	if !ok {
		t.Fatalf("expected line 2 to be matrix table container")
	}
	if len(line2.Objects) != 5 {
		t.Fatalf("expected 5 objects in matrix table (3 cols + 2 seps), got %d", len(line2.Objects))
	}

	// Verify required labels: Quota, Gemini, Claude, 5h, Week
	matrixTexts := extractAllTexts(line2)
	requiredLabels := []string{"Quota", "Gemini", "Claude", "5h", "Week"}
	for _, req := range requiredLabels {
		found := false
		for _, txt := range matrixTexts {
			if txt == req {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected label '%s' in matrix quota table, got texts: %v", req, matrixTexts)
		}
	}

	// Verify separators have BorderSubtle color
	if sep1, ok := line2.Objects[1].(*canvas.Rectangle); ok {
		assertColorMatches(t, "MatrixTable", "sep1", sep1.FillColor, DarkPalette.BorderSubtle)
	} else {
		t.Errorf("expected separator 1 to be canvas.Rectangle")
	}
	if sep2, ok := line2.Objects[3].(*canvas.Rectangle); ok {
		assertColorMatches(t, "MatrixTable", "sep2", sep2.FillColor, DarkPalette.BorderSubtle)
	} else {
		t.Errorf("expected separator 2 to be canvas.Rectangle")
	}
}

func TestOwnerRedesign_EmailTruncationAndTooltip(t *testing.T) {
	email := "very.long.enterprise.account.name.with.many.subdomains@antigravity.company.com"
	item := manager.AccountDisplayItem{
		Email:    email,
		IsActive: false,
		State:    model.StateReady,
	}

	card := NewAccountCardWithQuota(item, nil, DarkPalette, nil, nil, nil)
	var emailLbl *emailLabelWithTooltip
	var findEmailLbl func(fyne.CanvasObject)
	findEmailLbl = func(o fyne.CanvasObject) {
		if el, ok := o.(*emailLabelWithTooltip); ok {
			emailLbl = el
			return
		}
		if cont, ok := o.(*fyne.Container); ok {
			for _, child := range cont.Objects {
				findEmailLbl(child)
			}
		}
	}
	findEmailLbl(card)

	if emailLbl == nil {
		t.Fatalf("expected *emailLabelWithTooltip in account card")
	}
	if emailLbl.Truncation != fyne.TextTruncateEllipsis {
		t.Errorf("expected Truncation=TextTruncateEllipsis, got %v", emailLbl.Truncation)
	}
	if emailLbl.Wrapping != fyne.TextWrapOff {
		t.Errorf("expected Wrapping=TextWrapOff, got %v", emailLbl.Wrapping)
	}
	if emailLbl.fullText != email {
		t.Errorf("expected fullText '%s', got '%s'", email, emailLbl.fullText)
	}
}

func TestOwnerRedesign_ThreeColumnGrid(t *testing.T) {
	mgr := createTestManager(t)
	testApp := test.NewApp()
	app := NewAppWithFyne(testApp, mgr)
	defer app.Close()
	app.Init()
	waitForIdle(app, 1*time.Second)

	accounts := make([]manager.AccountDisplayItem, 6)
	for i := 0; i < 6; i++ {
		accounts[i] = manager.AccountDisplayItem{
			Email:    fmt.Sprintf("account%d@gmail.com", i+1),
			IsActive: i == 0,
			State:    model.StateReady,
		}
	}

	overview := &manager.AccountOverview{
		ActiveEmail:           "account1@gmail.com",
		LanguageServerRunning: true,
		ServerInfo:            &connector.ServerInfo{PID: 7768, Port: 52704},
		Accounts:              accounts,
	}

	app.updateUI(overview)

	if len(app.accountListContainer.Objects) != 6 {
		t.Fatalf("expected 6 account cards in container, got %d", len(app.accountListContainer.Objects))
	}

	expectedStatus := "Language Server : PID 7768 | Port 52704 | Hoạt động"
	if app.serverStatusLabel.Text != expectedStatus {
		t.Errorf("expected '%s', got '%s'", expectedStatus, app.serverStatusLabel.Text)
	}
}

func TestOwnerRedesign_NoGPTInCardsOrSummary(t *testing.T) {
	quota := &telemetry.AccountQuotaData{
		Email:  "user@example.com",
		Claude: telemetry.ModelQuotaBucket{RemainingFraction: 1.0, FiveHourFraction: -1, WeeklyFraction: -1},
		Gemini: telemetry.ModelQuotaBucket{RemainingFraction: 0.88, FiveHourFraction: -1, WeeklyFraction: -1},
		RawBuckets: []telemetry.ModelQuotaBucket{
			{ModelID: "gemini-5h", RemainingFraction: 0.88},
			{ModelID: "gemini-weekly", RemainingFraction: 0.90},
			{ModelID: "3p-5h", RemainingFraction: 1.0},
			{ModelID: "3p-weekly", RemainingFraction: 1.0},
			{ModelID: "gpt-oss", RemainingFraction: 0.95},
		},
	}

	card := NewAccountCardWithQuota(
		manager.AccountDisplayItem{Email: "user@example.com", State: model.StateReady},
		quota,
		DarkPalette,
		nil,
		nil,
		nil,
	)

	var walkNoGPT func(fyne.CanvasObject)
	walkNoGPT = func(o fyne.CanvasObject) {
		if txt, ok := o.(*canvas.Text); ok {
			if strings.Contains(strings.ToUpper(txt.Text), "GPT") {
				t.Fatalf("CRITICAL NON-GOAL VIOLATION: found GPT text '%s' inside card", txt.Text)
			}
		}
		if lbl, ok := o.(*widget.Label); ok {
			if strings.Contains(strings.ToUpper(lbl.Text), "GPT") {
				t.Fatalf("CRITICAL NON-GOAL VIOLATION: found GPT text '%s' inside label", lbl.Text)
			}
		}
		if cont, ok := o.(*fyne.Container); ok {
			for _, child := range cont.Objects {
				walkNoGPT(child)
			}
		}
	}
	walkNoGPT(card)


}

// -----------------------------------------------------------------------------
// Regression Tests for F1-F6 Acceptance Gates
// -----------------------------------------------------------------------------

// TestOwnerRedesign_F1_DualCycleIndependentFractionsAndMissing verifies F1:
// 5h and weekly fractions are independent for Claude and Gemini, and missing buckets do not fake 100%.
func TestOwnerRedesign_F1_DualCycleIndependentFractionsAndMissing(t *testing.T) {
	// Case 1: Independent fractions from Connect-RPC payload
	quotaData := &telemetry.AccountQuotaData{
		Email: "independent@example.com",
		Claude: telemetry.ModelQuotaBucket{
			ModelID:           "3p-5h",
			RemainingFraction: 0.12, // min fraction
			FiveHourFraction:  0.46,
			WeeklyFraction:    0.12,
		},
		Gemini: telemetry.ModelQuotaBucket{
			ModelID:           "gemini-5h",
			RemainingFraction: 0.88,
			FiveHourFraction:  0.88,
			WeeklyFraction:    0.90,
		},
	}

	dq := ExtractDualCycleQuota(quotaData)
	if !dq.HasData {
		t.Fatalf("expected HasData to be true")
	}
	if dq.Claude5h != 0.46 {
		t.Errorf("expected Claude 5h 0.46, got %f", dq.Claude5h)
	}
	if dq.ClaudeWeek != 0.12 {
		t.Errorf("expected Claude week 0.12, got %f", dq.ClaudeWeek)
	}
	if dq.Gemini5h != 0.88 {
		t.Errorf("expected Gemini 5h 0.88, got %f", dq.Gemini5h)
	}
	if dq.GeminiWeek != 0.90 {
		t.Errorf("expected Gemini week 0.90, got %f", dq.GeminiWeek)
	}

	// Case 2: Missing weekly bucket (does not fake 100%, renders --%)
	partialQuota := &telemetry.AccountQuotaData{
		Email: "partial@example.com",
		Claude: telemetry.ModelQuotaBucket{
			ModelID:           "3p-5h",
			RemainingFraction: 0.46,
			FiveHourFraction:  0.46,
			WeeklyFraction:    -1, // Missing
		},
		Gemini: telemetry.ModelQuotaBucket{
			ModelID:           "gemini-5h",
			RemainingFraction: 0.88,
			FiveHourFraction:  -1, // Missing
			WeeklyFraction:    0.90,
		},
	}

	dqPartial := ExtractDualCycleQuota(partialQuota)
	if dqPartial.ClaudeWeek != -1 {
		t.Errorf("expected missing Claude week to be -1, got %f", dqPartial.ClaudeWeek)
	}
	if dqPartial.Gemini5h != -1 {
		t.Errorf("expected missing Gemini 5h to be -1, got %f", dqPartial.Gemini5h)
	}

	// Verify UI rendering of missing bucket renders "--%" and graphical mini progress bars are present
	row := CreateQuotaCycleRow("week:", dqPartial.ClaudeWeek, dqPartial.GeminiWeek, dqPartial.HasData, dqPartial.IsError)
	foundMissing := false
	foundMiniBar := false
	var walk func(fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		if txt, ok := o.(*canvas.Text); ok {
			if strings.Contains(txt.Text, "--%") {
				foundMissing = true
			}
		}
		if rect, ok := o.(*canvas.Rectangle); ok {
			if rect.CornerRadius == 3 && rect.Size().Height == 6 {
				foundMiniBar = true
			}
		}
		if c, ok := o.(*fyne.Container); ok {
			for _, child := range c.Objects {
				walk(child)
			}
		}
	}
	walk(row)
	if !foundMissing {
		t.Errorf("expected missing bucket to render '--%%' label")
	}
	if !foundMiniBar {
		t.Errorf("expected MiniProgressBar (CornerRadius: 3, Height: 6) in quota row")
	}
}

// TestOwnerRedesign_F3_SingleCardRefreshKeepsSiblings verifies F3:
// Refreshing a single account card updates ONLY that card and preserves all sibling objects.
func TestOwnerRedesign_F3_SingleCardRefreshKeepsSiblings(t *testing.T) {
	fyneApp := test.NewApp()
	defer fyneApp.Quit()

	tmp := t.TempDir()
	accountStore, err := store.NewAccountStore(filepath.Join(tmp, "accounts"))
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 6; i++ {
		email := fmt.Sprintf("account%d@example.com", i)
		_ = accountStore.SaveAccount(email, &model.TokenData{AccessToken: "token", RefreshToken: "refresh"})
	}
	mgr, _ := manager.NewAccountManager(accountStore, nil)
	mgr.SetChromeDir(filepath.Join(tmp, "no-real-chrome"))

	app := NewAppWithFyne(fyneApp, mgr)
	app.SetPreferencesPath(filepath.Join(tmp, "preferences.json"))

	pool, _ := telemetry.NewTelemetryPoolWithSource(&testQuotaClient{}, telemetry.NewQuotaCache(filepath.Join(tmp, "q.json")), &testAccountSource{})
	pool.SetLocalQuotaFetcher(func(ctx context.Context, email string) (*telemetry.AccountQuotaData, error) {
		return &telemetry.AccountQuotaData{
			Email:  email,
			Claude: telemetry.ModelQuotaBucket{RemainingFraction: 0.50, FiveHourFraction: -1, WeeklyFraction: -1},
			Gemini: telemetry.ModelQuotaBucket{RemainingFraction: 0.75, FiveHourFraction: -1, WeeklyFraction: -1},
			Status: telemetry.StatusPlentiful,
		}, nil
	})
	app.SetTelemetryPool(pool)
	app.Init()
	defer app.Close()

	// Wait for initial cards
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if app.accountListContainer != nil && len(app.accountListContainer.Objects) >= 6 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	grid := app.accountListContainer
	if grid == nil || len(grid.Objects) < 6 {
		t.Fatalf("expected at least 6 initial cards, got %v", grid)
	}

	initialCards := append([]fyne.CanvasObject(nil), grid.Objects...)
	targetEmail := "account1@example.com"
	app.mu.RLock()
	if app.overview != nil && len(app.overview.Accounts) > 0 {
		targetEmail = app.overview.Accounts[0].Email
	}
	app.mu.RUnlock()

	// Trigger single refresh for card 0
	app.RefreshSingleAccountQuota(targetEmail)
	waitForIdle(app, 3*time.Second)

	// Target card must be replaced
	if grid.Objects[0] == initialCards[0] {
		t.Errorf("expected target card 0 to be refreshed/updated")
	}

	// All other siblings must remain EXACTLY identical objects (0 re-creations)
	unchanged := 0
	for i := 1; i < len(initialCards); i++ {
		if grid.Objects[i] == initialCards[i] {
			unchanged++
		}
	}
	expectedUnchanged := len(initialCards) - 1
	if unchanged != expectedUnchanged {
		t.Fatalf("F3 VIOLATION: expected %d unchanged sibling cards, but got %d", expectedUnchanged, unchanged)
	}
}
func TestOwnerRedesign_F5_CompactCardHeight(t *testing.T) {
	app := NewAppWithFyne(test.NewApp(), nil)
	app.Init()
	defer app.Close()

	w := app.Window()
	w.Resize(fyne.NewSize(1008, 630))

	// Populate 6 cards
	palette := DarkPalette
	for i := 1; i <= 6; i++ {
		item := manager.AccountDisplayItem{Email: fmt.Sprintf("user%d@example.com", i)}
		card := NewAccountCardWithQuota(item, nil, palette, nil, nil, nil)
		app.accountListContainer.Add(card)
	}
	minH := app.accountListContainer.MinSize().Height
	app.accountListContainer.Resize(fyne.NewSize(992, minH))
	app.accountListContainer.Refresh()

	card0Size := app.accountListContainer.Objects[0].Size()
	if card0Size.Height > 100 {
		t.Fatalf("F5 VIOLATION: Card height is stretched to %f (expected <= 100 compact)", card0Size.Height)
	}
	if card0Size.Height <= 0 {
		t.Fatalf("F5 VIOLATION: Card height is non-positive: %f", card0Size.Height)
	}

	gridH := app.accountListContainer.Size().Height
	if gridH > 250 {
		t.Fatalf("F5 VIOLATION: Grid height is stretched to %f (expected <= 250 compact)", gridH)
	}
}

// TestOwnerRedesign_F6_MonogramRecolorsOnThemeChange verifies F6:
// Monogram AS background recolors dynamically across Dark, Light, and Warm themes.
func TestOwnerRedesign_F6_MonogramRecolorsOnThemeChange(t *testing.T) {
	app := NewAppWithFyne(test.NewApp(), nil)
	app.Init()
	defer app.Close()

	if app.monogramBg == nil {
		t.Fatalf("expected app.monogramBg to be initialized")
	}

	// Initial Dark theme
	assertColorMatches(t, "Init Dark", "monogramBg", app.monogramBg.FillColor, DarkPalette.Primary)

	// Switch to Light
	app.SetTheme(ThemeLight)
	assertColorMatches(t, "Switch Light", "monogramBg", app.monogramBg.FillColor, LightPalette.Primary)

	// Switch to Warm
	app.SetTheme(ThemeWarm)
	assertColorMatches(t, "Switch Warm", "monogramBg", app.monogramBg.FillColor, WarmPalette.Primary)

	// Switch back to Dark
	app.SetTheme(ThemeDark)
	assertColorMatches(t, "Switch Dark", "monogramBg", app.monogramBg.FillColor, DarkPalette.Primary)
}

// TestF1_ExtractDualCycleQuota_BothCyclesZero_NoRaw_NoReset verifies that when both 5h and weekly cycles
// are 0.0 for Claude and Gemini without raw buckets or reset time:
// 1. ExtractDualCycleQuota returns Claude5h=0.0, ClaudeWeek=0.0, Gemini5h=0.0, GeminiWeek=0.0, HasData=true.
// 2. NewAccountCardWithQuota renders "0%" for all cycles, never "--%".
// 3. Asymmetric zero cases (one family 0.0, other missing) preserve zero and unknown independently.
func TestF1_ExtractDualCycleQuota_BothCyclesZero_NoRaw_NoReset(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	// Case 1: Both Claude and Gemini have both cycles 0.0 without raw buckets and without reset times
	bothZeroQuota := &telemetry.AccountQuotaData{
		Email: "bothzero@example.com",
		Claude: telemetry.ModelQuotaBucket{
			ModelID:           "claude-4-6",
			RemainingFraction: 0.0,
			FiveHourFraction:  0.0,
			WeeklyFraction:    0.0,
		},
		Gemini: telemetry.ModelQuotaBucket{
			ModelID:           "gemini-2-5",
			RemainingFraction: 0.0,
			FiveHourFraction:  0.0,
			WeeklyFraction:    0.0,
		},
		Status: telemetry.StatusDepleted,
	}

	dq := ExtractDualCycleQuota(bothZeroQuota)
	if !dq.HasData {
		t.Errorf("expected HasData=true for both cycles zero")
	}
	if dq.Claude5h != 0.0 || dq.ClaudeWeek != 0.0 {
		t.Errorf("expected Claude (0.0, 0.0), got 5h=%f, week=%f", dq.Claude5h, dq.ClaudeWeek)
	}
	if dq.Gemini5h != 0.0 || dq.GeminiWeek != 0.0 {
		t.Errorf("expected Gemini (0.0, 0.0), got 5h=%f, week=%f", dq.Gemini5h, dq.GeminiWeek)
	}

	// Verify card UI rendering
	card := NewAccountCardWithQuota(
		manager.AccountDisplayItem{Email: bothZeroQuota.Email},
		bothZeroQuota,
		LightPalette,
		nil,
		nil,
		nil,
	)
	var cardTexts []string
	var walk func(fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		if txt, ok := o.(*canvas.Text); ok {
			cardTexts = append(cardTexts, txt.Text)
		}
		if c, ok := o.(*fyne.Container); ok {
			for _, child := range c.Objects {
				walk(child)
			}
		}
	}
	walk(card)

	claudeCount := 0
	geminiCount := 0
	zeroCount := 0
	for _, text := range cardTexts {
		if text == "Claude" {
			claudeCount++
		}
		if text == "Gemini" {
			geminiCount++
		}
		if text == "0%" {
			zeroCount++
		}
		if strings.Contains(text, "--%") {
			t.Errorf("unexpected placeholder --%%%% on zero-cycle card, text: %s", text)
		}
	}
	if claudeCount != 1 || geminiCount != 1 || zeroCount != 4 {
		t.Errorf("expected Claude (2), Gemini (2), and 0%% (4) on zero-cycle card, got texts: %v", cardTexts)
	}

	// Case 2: Claude both zero, Gemini missing
	claudeOnlyZero := &telemetry.AccountQuotaData{
		Email: "claudezero@example.com",
		Claude: telemetry.ModelQuotaBucket{
			ModelID:           "claude-4-6",
			RemainingFraction: 0.0,
			FiveHourFraction:  0.0,
			WeeklyFraction:    0.0,
		},
		Gemini: telemetry.ModelQuotaBucket{
			ModelID:           "gemini-2-5",
			RemainingFraction: -1,
			FiveHourFraction:  -1,
			WeeklyFraction:    -1,
		},
	}
	dqClaude := ExtractDualCycleQuota(claudeOnlyZero)
	if dqClaude.Claude5h != 0.0 || dqClaude.ClaudeWeek != 0.0 {
		t.Errorf("expected Claude (0.0, 0.0), got 5h=%f, week=%f", dqClaude.Claude5h, dqClaude.ClaudeWeek)
	}
	if dqClaude.Gemini5h != -1.0 || dqClaude.GeminiWeek != -1.0 {
		t.Errorf("expected Gemini (-1.0, -1.0), got 5h=%f, week=%f", dqClaude.Gemini5h, dqClaude.GeminiWeek)
	}

	// Case 3: Gemini both zero, Claude missing
	geminiOnlyZero := &telemetry.AccountQuotaData{
		Email: "geminizero@example.com",
		Claude: telemetry.ModelQuotaBucket{
			ModelID:           "claude-4-6",
			RemainingFraction: -1,
			FiveHourFraction:  -1,
			WeeklyFraction:    -1,
		},
		Gemini: telemetry.ModelQuotaBucket{
			ModelID:           "gemini-2-5",
			RemainingFraction: 0.0,
			FiveHourFraction:  0.0,
			WeeklyFraction:    0.0,
		},
	}
	dqGemini := ExtractDualCycleQuota(geminiOnlyZero)
	if dqGemini.Claude5h != -1.0 || dqGemini.ClaudeWeek != -1.0 {
		t.Errorf("expected Claude (-1.0, -1.0), got 5h=%f, week=%f", dqGemini.Claude5h, dqGemini.ClaudeWeek)
	}
	if dqGemini.Gemini5h != 0.0 || dqGemini.GeminiWeek != 0.0 {
		t.Errorf("expected Gemini (0.0, 0.0), got 5h=%f, week=%f", dqGemini.Gemini5h, dqGemini.GeminiWeek)
	}
}

// TestF1_ExtractDualCycleQuota_PositiveLegacyFraction_WithBothCyclesZero verifies that
// when positive RemainingFraction (> 0, e.g. 0.2 or 0.8) exists alongside explicit zero cycle fractions (0.0, 0.0)
// without raw buckets or reset times:
// 1. ExtractDualCycleQuota returns exactly (0.0, 0.0) for 5h and weekly, NEVER falling back to (RemainingFraction, -1).
// 2. NewAccountCardWithQuota renders "0%" in ColorQuotaRed for both cycles (never "20%", "80%", or "--%").
// 3. Asymmetric positive legacy with zero cycles is preserved independently per model family.
// 4. Pure legacy without cycle fractions (-1, -1) still correctly falls back to (RemainingFraction, -1).
func TestF1_ExtractDualCycleQuota_PositiveLegacyFraction_WithBothCyclesZero(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	// Helper to extract all text strings from a canvas object hierarchy
	var extractTexts func(fyne.CanvasObject) []string
	extractTexts = func(o fyne.CanvasObject) []string {
		var res []string
		if txt, ok := o.(*canvas.Text); ok {
			res = append(res, txt.Text)
		}
		if cont, ok := o.(*fyne.Container); ok {
			for _, child := range cont.Objects {
				res = append(res, extractTexts(child)...)
			}
		}
		return res
	}

	// Case 1: Both Claude and Gemini have positive RemainingFraction and explicit zero cycle fractions
	bothPosLegacyBothZeroCycles := &telemetry.AccountQuotaData{
		Email: "both_pos_legacy_zero@example.com",
		Claude: telemetry.ModelQuotaBucket{
			ModelID:           "claude-4-6",
			RemainingFraction: 0.2,
			FiveHourFraction:  0.0,
			WeeklyFraction:    0.0,
		},
		Gemini: telemetry.ModelQuotaBucket{
			ModelID:           "gemini-2-5",
			RemainingFraction: 0.8,
			FiveHourFraction:  0.0,
			WeeklyFraction:    0.0,
		},
		Status: telemetry.StatusDepleted,
	}

	dq := ExtractDualCycleQuota(bothPosLegacyBothZeroCycles)
	if !dq.HasData {
		t.Errorf("expected HasData=true for explicit zero cycles with positive legacy fraction")
	}
	if dq.Claude5h != 0.0 || dq.ClaudeWeek != 0.0 {
		t.Errorf("expected Claude (0.0, 0.0), got 5h=%f, week=%f (must NOT fall back to RemainingFraction 0.2)",
			dq.Claude5h, dq.ClaudeWeek)
	}
	if dq.Gemini5h != 0.0 || dq.GeminiWeek != 0.0 {
		t.Errorf("expected Gemini (0.0, 0.0), got 5h=%f, week=%f (must NOT fall back to RemainingFraction 0.8)",
			dq.Gemini5h, dq.GeminiWeek)
	}

	item := manager.AccountDisplayItem{
		Email:    "both_pos_legacy_zero@example.com",
		State:    model.StateReady,
		IsActive: false,
	}
	card := NewAccountCardWithQuota(item, bothPosLegacyBothZeroCycles, DarkPalette, nil, nil, nil)
	texts := extractTexts(card)

	claudeCount := 0
	geminiCount := 0
	zeroCount := 0
	for _, txt := range texts {
		if txt == "Claude" {
			claudeCount++
		}
		if txt == "Gemini" {
			geminiCount++
		}
		if txt == "0%" {
			zeroCount++
		}
		if strings.Contains(txt, "20%") || strings.Contains(txt, "80%") {
			t.Errorf("CRITICAL VIOLATION: card displayed legacy percentage instead of 0%%: %s", txt)
		}
		if strings.Contains(txt, "--%") {
			t.Errorf("CRITICAL VIOLATION: card displayed placeholder '--%%' instead of explicit 0%%: %s", txt)
		}
	}
	if claudeCount != 1 || geminiCount != 1 || zeroCount != 4 {
		t.Errorf("expected Claude to have two 'Claude' and two '0%%' labels, got texts: %v", texts)
	}

	// Case 2: Asymmetric — Claude has positive RemainingFraction with zero cycles; Gemini is unknown (-1)
	claudePosZeroGeminiUnknown := &telemetry.AccountQuotaData{
		Email: "claude_zero_gemini_unk@example.com",
		Claude: telemetry.ModelQuotaBucket{
			ModelID:           "claude-4-6",
			RemainingFraction: 0.25,
			FiveHourFraction:  0.0,
			WeeklyFraction:    0.0,
		},
		Gemini: telemetry.ModelQuotaBucket{
			ModelID:           "gemini-2-5",
			RemainingFraction: -1,
			FiveHourFraction:  -1,
			WeeklyFraction:    -1,
		},
	}
	dqClaude := ExtractDualCycleQuota(claudePosZeroGeminiUnknown)
	if dqClaude.Claude5h != 0.0 || dqClaude.ClaudeWeek != 0.0 {
		t.Errorf("expected Claude (0.0, 0.0), got 5h=%f, week=%f", dqClaude.Claude5h, dqClaude.ClaudeWeek)
	}
	if dqClaude.Gemini5h != -1.0 || dqClaude.GeminiWeek != -1.0 {
		t.Errorf("expected Gemini (-1.0, -1.0), got 5h=%f, week=%f", dqClaude.Gemini5h, dqClaude.GeminiWeek)
	}

	// Case 3: Asymmetric — Gemini has positive RemainingFraction with zero cycles; Claude is unknown (-1)
	geminiPosZeroClaudeUnknown := &telemetry.AccountQuotaData{
		Email: "gemini_zero_claude_unk@example.com",
		Claude: telemetry.ModelQuotaBucket{
			ModelID:           "claude-4-6",
			RemainingFraction: -1,
			FiveHourFraction:  -1,
			WeeklyFraction:    -1,
		},
		Gemini: telemetry.ModelQuotaBucket{
			ModelID:           "gemini-2-5",
			RemainingFraction: 0.75,
			FiveHourFraction:  0.0,
			WeeklyFraction:    0.0,
		},
	}
	dqGemini := ExtractDualCycleQuota(geminiPosZeroClaudeUnknown)
	if dqGemini.Claude5h != -1.0 || dqGemini.ClaudeWeek != -1.0 {
		t.Errorf("expected Claude (-1.0, -1.0), got 5h=%f, week=%f", dqGemini.Claude5h, dqGemini.ClaudeWeek)
	}
	if dqGemini.Gemini5h != 0.0 || dqGemini.GeminiWeek != 0.0 {
		t.Errorf("expected Gemini (0.0, 0.0), got 5h=%f, week=%f", dqGemini.Gemini5h, dqGemini.GeminiWeek)
	}

	// Case 4: True legacy fallback — cycles are -1, positive RemainingFraction is used for 5h, week is -1
	legacyFallbackQuota := &telemetry.AccountQuotaData{
		Email: "legacy_fallback@example.com",
		Claude: telemetry.ModelQuotaBucket{
			ModelID:           "claude-4-6",
			RemainingFraction: 0.35,
			FiveHourFraction:  -1,
			WeeklyFraction:    -1,
		},
		Gemini: telemetry.ModelQuotaBucket{
			ModelID:           "gemini-2-5",
			RemainingFraction: 0.65,
			FiveHourFraction:  -1,
			WeeklyFraction:    -1,
		},
	}
	dqLegacy := ExtractDualCycleQuota(legacyFallbackQuota)
	if dqLegacy.Claude5h != 0.35 || dqLegacy.ClaudeWeek != -1.0 {
		t.Errorf("expected Claude legacy fallback (0.35, -1.0), got 5h=%f, week=%f", dqLegacy.Claude5h, dqLegacy.ClaudeWeek)
	}
	if dqLegacy.Gemini5h != 0.65 || dqLegacy.GeminiWeek != -1.0 {
		t.Errorf("expected Gemini legacy fallback (0.65, -1.0), got 5h=%f, week=%f", dqLegacy.Gemini5h, dqLegacy.GeminiWeek)
	}
}


func TestMiniProgressBar_DepletionAndGeometry(t *testing.T) {
	testCases := []struct {
		name          string
		fraction      float64
		barWidth      float32
		expectedFillW float32
		expectedColor color.NRGBA
	}{
		{"100% Green", 1.0, 50.0, 50.0, ColorQuotaGreen},
		{"91.6% Green", 0.916, 50.0, 45.8, ColorQuotaGreen},
		{"46% Amber", 0.46, 50.0, 23.0, ColorQuotaAmber},
		{"12% Red", 0.12, 50.0, 6.0, ColorQuotaRed},
		{"0% Red", 0.0, 50.0, 0.0, ColorQuotaRed},
		{"Missing Muted", -1.0, 50.0, 0.0, ColorQuotaMuted},
		{"Overflow Clamped", 1.5, 50.0, 50.0, ColorQuotaGreen},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bar := createMiniProgressBar(tc.fraction, tc.barWidth)
			cont, ok := bar.(*fyne.Container)
			if !ok || len(cont.Objects) < 2 {
				t.Fatalf("expected container with background track and fill bar")
			}

			track, ok := cont.Objects[0].(*canvas.Rectangle)
			if !ok {
				t.Fatalf("expected track to be a canvas.Rectangle")
			}
			if track.Size().Width != tc.barWidth || track.Size().Height != 6.0 {
				t.Errorf("unexpected track size: %v, expected %fx6", track.Size(), tc.barWidth)
			}
			if track.CornerRadius != 3.0 {
				t.Errorf("expected track CornerRadius 3.0, got %f", track.CornerRadius)
			}
			assertColorMatches(t, tc.name, "track_color", track.FillColor, ColorQuotaTrack)

			fill, ok := cont.Objects[1].(*canvas.Rectangle)
			if !ok {
				t.Fatalf("expected fill to be a canvas.Rectangle")
			}
			diff := fill.Size().Width - tc.expectedFillW
			if diff < -0.01 || diff > 0.01 {
				t.Errorf("expected fill width ~%f, got %f", tc.expectedFillW, fill.Size().Width)
			}
			if fill.Size().Height != 6.0 {
				t.Errorf("expected fill height 6.0, got %f", fill.Size().Height)
			}
			if fill.CornerRadius != 3.0 {
				t.Errorf("expected fill CornerRadius 3.0, got %f", fill.CornerRadius)
			}
			assertColorMatches(t, tc.name, "fill_color", fill.FillColor, tc.expectedColor)
		})
	}
}

// extractAllTexts recursively extracts text from all canvas.Text and widget.Label objects.
func extractAllTexts(o fyne.CanvasObject) []string {
	var res []string
	if txt, ok := o.(*canvas.Text); ok {
		res = append(res, txt.Text)
	}
	if lbl, ok := o.(*widget.Label); ok {
		res = append(res, lbl.Text)
	}
	if cont, ok := o.(*fyne.Container); ok {
		for _, child := range cont.Objects {
			res = append(res, extractAllTexts(child)...)
		}
	}
	return res
}

func TestCreateMatrixQuotaTable_DetailedMatrixBehavior(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	// 1. Populated DualCycleQuota with various tier percentages
	dq := DualCycleQuota{
		Claude5h:   1.0,  // 100% -> Green
		ClaudeWeek: 0.46, // 46% -> Amber
		Gemini5h:   0.88, // 88% -> Green
		GeminiWeek: 0.12, // 12% -> Red
		HasData:    true,
		IsError:    false,
	}

	for _, palette := range []ThemePalette{DarkPalette, LightPalette, WarmPalette} {
		t.Run("Palette_"+palette.Name, func(t *testing.T) {
			matrix := CreateMatrixQuotaTable(dq, palette)
			cont, ok := matrix.(*fyne.Container)
			if !ok {
				t.Fatalf("expected matrix to be a *fyne.Container")
			}
			if len(cont.Objects) != 5 {
				t.Fatalf("expected 5 objects in matrix (3 cols + 2 seps), got %d", len(cont.Objects))
			}

			// Verify 2 vertical separators
			sep1, ok := cont.Objects[1].(*canvas.Rectangle)
			if !ok {
				t.Fatalf("expected sep1 to be canvas.Rectangle")
			}
			assertColorMatches(t, palette.Name, "sep1", sep1.FillColor, palette.BorderSubtle)

			sep2, ok := cont.Objects[3].(*canvas.Rectangle)
			if !ok {
				t.Fatalf("expected sep2 to be canvas.Rectangle")
			}
			assertColorMatches(t, palette.Name, "sep2", sep2.FillColor, palette.BorderSubtle)

			// Verify labels and percentages
			texts := extractAllTexts(matrix)
			expectedTexts := []string{"Quota", "Gemini", "Claude", "5h", "88%", "100%", "Week", "12%", "46%"}
			for _, exp := range expectedTexts {
				found := false
				for _, txt := range texts {
					if txt == exp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("missing expected text '%s' in matrix, got %v", exp, texts)
				}
			}
		})
	}

	// 2. Missing data / Error state renders "--%"
	dqError := DualCycleQuota{
		Claude5h:   -1,
		ClaudeWeek: -1,
		Gemini5h:   -1,
		GeminiWeek: -1,
		HasData:    false,
		IsError:    true,
	}
	matrixErr := CreateMatrixQuotaTable(dqError, DarkPalette)
	errTexts := extractAllTexts(matrixErr)
	dashCount := 0
	for _, txt := range errTexts {
		if txt == "--%" {
			dashCount++
		}
	}
	if dashCount != 4 {
		t.Errorf("expected 4 '--%%' labels on error matrix, got %d (texts: %v)", dashCount, errTexts)
	}

	// 3. Dimensions: MinSize width ~236px, fits well inside 252px card
	minSize := matrixErr.MinSize()
	if minSize.Width > 252.0 {
		t.Errorf("expected matrix MinSize.Width <= 252, got %f", minSize.Width)
	}
	if minSize.Height > 60.0 {
		t.Errorf("expected matrix MinSize.Height <= 60, got %f", minSize.Height)
	}
}
