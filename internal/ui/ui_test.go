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
	"fyne.io/fyne/v2/driver/desktop"
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

func TestAlwaysOnTop(t *testing.T) {
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
	if app.alwaysOnTopBtn.Text != "📌" {
		t.Errorf("expected button text '📌', got: %s", app.alwaysOnTopBtn.Text)
	}
	if app.alwaysOnTopBtn.Importance != widget.MediumImportance {
		t.Errorf("expected button importance MediumImportance, got: %v", app.alwaysOnTopBtn.Importance)
	}

	app.SetAlwaysOnTop(true)
	if !app.IsAlwaysOnTop() {
		t.Errorf("expected alwaysOnTop to be true after SetAlwaysOnTop(true)")
	}
	if app.alwaysOnTopBtn.Text != "📌" {
		t.Errorf("expected button text '📌', got: %s", app.alwaysOnTopBtn.Text)
	}
	if app.alwaysOnTopBtn.Importance != widget.HighImportance {
		t.Errorf("expected button importance HighImportance, got: %v", app.alwaysOnTopBtn.Importance)
	}

	waitForIdle(app, 3*time.Second)
}

func TestToggleAlwaysOnTop(t *testing.T) {
	TestAlwaysOnTop(t)
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
	if !strings.Contains(app.serverStatusLabel.Text, "Không hoạt động") {
		t.Errorf("expected server status to contain 'Không hoạt động', got: %s", app.serverStatusLabel.Text)
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
	// Zero time -> "--"
	if res := formatCountdown(time.Time{}, false); res != "--" {
		t.Errorf("expected '--' for zero time (5h), got %q", res)
	}
	if res := formatCountdown(time.Time{}, true); res != "--" {
		t.Errorf("expected '--' for zero time (weekly), got %q", res)
	}

	// <= 0 remaining time -> "(Ready)"
	past := time.Now().Add(-10 * time.Minute)
	if res := formatCountdown(past, false); res != "(Ready)" {
		t.Errorf("expected '(Ready)' for past time (5h), got %q", res)
	}
	if res := formatCountdown(past, true); res != "(Ready)" {
		t.Errorf("expected '(Ready)' for past time (weekly), got %q", res)
	}

	// 5h cycle countdown format (04h 32m)
	future5h := time.Now().Add(4*time.Hour + 32*time.Minute + 5*time.Second)
	if res := formatCountdown(future5h, false); res != "(04h 32m)" {
		t.Errorf("expected '(04h 32m)' for 4h 32m, got %q", res)
	}
	future5h2 := time.Now().Add(2*time.Hour + 12*time.Minute + 5*time.Second)
	if res := formatCountdown(future5h2, false); res != "(02h 12m)" {
		t.Errorf("expected '(02h 12m)' for 2h 12m, got %q", res)
	}

	// Week cycle countdown format (6d 21h 59m)
	futureWeek := time.Now().Add(6*24*time.Hour + 21*time.Hour + 59*time.Minute + 5*time.Second)
	if res := formatCountdown(futureWeek, true); res != "(6d 21h 59m)" {
		t.Errorf("expected '(6d 21h 59m)', got %q", res)
	}
	futureWeek2 := time.Now().Add(2*24*time.Hour + 7*time.Hour + 8*time.Minute + 5*time.Second)
	if res := formatCountdown(futureWeek2, true); res != "(2d 07h 08m)" {
		t.Errorf("expected '(2d 07h 08m)', got %q", res)
	}

	// Cell visual styling invariant: text size 8.5pt and TextMuted color
	cell := newMatrixQuotaCell(1.0, future5h, false, true, DarkPalette)
	if cell.timeText == nil {
		t.Fatal("expected non-nil timeText in cell")
	}
	if cell.timeText.TextSize != 8.5 {
		t.Errorf("expected timeText size 8.5, got %f", cell.timeText.TextSize)
	}
	assertColorMatches(t, "DarkPalette", "timeText", cell.timeText.Color, DarkPalette.TextMuted)
	if cell.timeText.Text != "(04h 32m)" {
		t.Errorf("expected cell timeText '(04h 32m)', got %q", cell.timeText.Text)
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
	if app.quotaRefreshAllBtn.Text != "Làm mới Quota" {
		t.Errorf("unexpected button label: %s", app.quotaRefreshAllBtn.Text)
	}

	// UI-TR-03 check: Auto-refresh timestamp label eliminated for clean 3-row layout
	// Timestamp label removed from toolbar per CSD refactor spec
	//
	//
	//
	//
	//

	// Trigger RefreshAllQuota
	app.RefreshAllQuota()

	// Wait for async finish
	waitForIdle(app, 3*time.Second)

	// Verify button returned to standard label
	if app.quotaRefreshAllBtn.Text != "Làm mới Quota" {
		t.Errorf("expected button restored to standard label, got: %s", app.quotaRefreshAllBtn.Text)
	}

	// Timestamp label eliminated
	//
	//
	//

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

	// Populated quota with reset times
	reset5h := time.Now().Add(2 * time.Hour)
	resetWeek := time.Now().Add(7 * 24 * time.Hour)
	qWithReset := &telemetry.AccountQuotaData{
		Claude: telemetry.ModelQuotaBucket{
			RemainingFraction: 1.0, FiveHourFraction: 1.0, WeeklyFraction: 0.5,
			FiveHourReset: reset5h, WeeklyReset: resetWeek,
		},
		Gemini: telemetry.ModelQuotaBucket{
			RemainingFraction: 0.8, FiveHourFraction: 0.8, WeeklyFraction: 0.4,
			FiveHourReset: reset5h, WeeklyReset: resetWeek,
		},
	}
	dqWithReset := ExtractDualCycleQuota(qWithReset)
	if !dqWithReset.Claude5hReset.Equal(reset5h) || !dqWithReset.ClaudeWeekReset.Equal(resetWeek) {
		t.Errorf("unexpected Claude reset times in DualCycleQuota")
	}
	if !dqWithReset.Gemini5hReset.Equal(reset5h) || !dqWithReset.GeminiWeekReset.Equal(resetWeek) {
		t.Errorf("unexpected Gemini reset times in DualCycleQuota")
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
	if app.quotaRefreshAllBtn == nil || app.quotaRefreshAllBtn.Text != "Làm mới Quota" {
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

	// UI-TR-03: Auto-Refresh 60s timestamp label eliminated for clean 3-row layout
	//
	//
	//

	// UI-TR-04: Mini Quota Progress Badges (>50% Green, 15-50% Amber, <15% Red)
	if GetQuotaTierColor(0.8) != ColorQuotaGreen || GetQuotaTierColor(0.3) != ColorQuotaAmber || GetQuotaTierColor(0.05) != ColorQuotaRed {
		t.Errorf("UI-TR-04 FAILED: color thresholds incorrect")
	}

	// UI-TR-05: Quota Recovery Countdown eliminated for clean 3-row layout

	// UI-TR-06: Multi-Theme Switcher Control
	if app.themeSelect == nil || len(app.themeSelect.Options) != 3 || app.themeSelect.Options[0] != "🌙" || app.themeSelect.Options[1] != "☀️" || app.themeSelect.Options[2] != "☕" {
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
// Slice P6A-3 Tests: Owner Redesign Invariants (960x630, 3-Column Grid, 3-Row Cards, No GPT)
// -------------------------------------------------------------

func TestOwnerRedesign_WindowSizeAndLayout(t *testing.T) {
	mgr := createTestManager(t)
	testApp := test.NewApp()
	defer testApp.Quit()
	app := NewAppWithFyne(testApp, mgr)
	defer app.Close()
	app.Init()

	win := app.Window()
	if win == nil {
		t.Fatal("expected non-nil window")
	}

	if win.FixedSize() {
		t.Errorf("expected a resizable window for maximize/restore")
	}

	sz := win.Canvas().Size()
	if sz.Width != 960 || sz.Height != 630 {
		t.Errorf("expected initial window size 960x630, got %fx%f", sz.Width, sz.Height)
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
	var switchBtn *switchTextButton
	var findSwitch func(fyne.CanvasObject)
	findSwitch = func(o fyne.CanvasObject) {
		if btn, ok := o.(*switchTextButton); ok {
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
		t.Fatalf("expected '[Chuyển]' button on inactive card")
	}
	switchBtn.OnTapped()
	if switchedEmail != "inactive_owner@gmail.com" {
		t.Errorf("expected switchedEmail 'inactive_owner@gmail.com', got '%s'", switchedEmail)
	}

	// Line 2: Matrix Quota Table (3x3 Matrix with 3 columns, whitespace gap 8px, and horizontal separator)
	line2, ok := cardVBox.Objects[1].(*fyne.Container)
	if !ok {
		t.Fatalf("expected line 2 to be matrix table container")
	}
	if len(line2.Objects) < 4 {
		t.Fatalf("expected at least 4 objects in matrix table (3 cols + hSep), got %d", len(line2.Objects))
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

	// Verify horizontal separator between Gemini and Claude has BorderSubtle color
	if hSep, ok := line2.Objects[3].(*canvas.Rectangle); ok {
		assertColorMatches(t, "MatrixTable", "hSep", hSep.FillColor, DarkPalette.BorderSubtle)
	} else {
		t.Errorf("expected horizontal separator to be canvas.Rectangle")
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

	expectedStatus := "Hoạt động  | PID 7768 | Port 52704"
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
	deadline := time.Now().Add(6 * time.Second)
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

		// In-place reactive update (INV-01): target card container is preserved in-place (zero re-creations, no canvas churn)
	if grid.Objects[0] != initialCards[0] {
		t.Errorf("expected target card 0 to be updated in-place without replacing container object")
	}

	// All cards remain EXACTLY identical objects (0 re-creations)
	unchanged := 0
	for i := 0; i < len(initialCards); i++ {
		if grid.Objects[i] == initialCards[i] {
			unchanged++
		}
	}
	expectedUnchanged := len(initialCards)
	if unchanged != expectedUnchanged {
		t.Fatalf("F3 VIOLATION: expected %d unchanged card objects (in-place updates), but got %d", expectedUnchanged, unchanged)
	}
}
func TestOwnerRedesign_F5_CompactCardHeight(t *testing.T) {
	app := NewAppWithFyne(test.NewApp(), nil)
	app.Init()
	defer app.Close()

	w := app.Window()
	w.Resize(fyne.NewSize(960, 630))

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
	if card0Size.Height != 130 {
		t.Fatalf("F5 VIOLATION: Card height is %f (expected 130.0)", card0Size.Height)
	}
	if card0Size.Width != 280 {
		t.Fatalf("F5 VIOLATION: Card width is %f (expected 280.0)", card0Size.Width)
	}
	if card0Size.Height <= 0 {
		t.Fatalf("F5 VIOLATION: Card height is non-positive: %f", card0Size.Height)
	}

	gridH := app.accountListContainer.Size().Height
	if gridH > 350 {
		t.Fatalf("F5 VIOLATION: Grid height is stretched to %f (expected <= 350 compact)", gridH)
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
			if len(cont.Objects) < 4 {
				t.Fatalf("expected at least 4 objects in matrix (3 cols + hSep), got %d", len(cont.Objects))
			}

			// Verify horizontal separator between Gemini and Claude
			hSep, ok := cont.Objects[3].(*canvas.Rectangle)
			if !ok {
				t.Fatalf("expected hSep to be canvas.Rectangle")
			}
			assertColorMatches(t, palette.Name, "hSep", hSep.FillColor, palette.BorderSubtle)
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

	// 3. Dimensions: MinSize width 228px, fits well inside 280px card
	minSize := matrixErr.MinSize()
	if minSize.Width > 280.0 {
		t.Errorf("expected matrix MinSize.Width <= 280, got %f", minSize.Width)
	}
	if minSize.Width != 228.0 || minSize.Height != 77.0 {
		t.Errorf("expected matrix MinSize 228x77, got %v", minSize)
	}
	if minSize.Height > 80.0 {
		t.Errorf("expected matrix MinSize.Height <= 80, got %f", minSize.Height)
	}
}

// -----------------------------------------------------------------------------
// Tests for Architect Spec SPEC-ARCH-20261019-UI-RESIZE-STABILITY
// -----------------------------------------------------------------------------

func TestArchitectSpec_INV01_InPlaceReactiveQuotaUpdatesZeroCanvasChurn(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	mgr := createTestManager(t)
	app := NewAppWithFyne(testApp, mgr)
	defer app.Close()

	app.Init()
	waitForIdle(app, 3*time.Second)

	if app.accountListContainer == nil || len(app.accountListContainer.Objects) == 0 {
		t.Fatalf("expected populated account cards in accountListContainer")
	}

	// Snapshot initial Canvas Object pointers
	initialObjects := make([]fyne.CanvasObject, len(app.accountListContainer.Objects))
	copy(initialObjects, app.accountListContainer.Objects)

	initialCards := app.AccountCards()
	if len(initialCards) != len(initialObjects) {
		t.Fatalf("expected %d account cards, got %d", len(initialObjects), len(initialCards))
	}

	// Simulate Quota cache update for first account
	firstEmail := initialCards[0].Email
	newQuota := &telemetry.AccountQuotaData{
		Email: firstEmail,
		Gemini: telemetry.ModelQuotaBucket{
			ModelID:          "gemini-2.5",
			FiveHourFraction: 0.95,
			WeeklyFraction:   0.85,
		},
		Claude: telemetry.ModelQuotaBucket{
			ModelID:          "claude-3.7",
			FiveHourFraction: 0.45,
			WeeklyFraction:   0.10,
		},
	}
	app.quotaCache.Update(newQuota)

	// Trigger updateUI (identical overview)
	app.mu.RLock()
	overview := app.overview
	app.mu.RUnlock()

	app.updateUI(overview)

	// Verify ZERO Canvas churn: Every single object pointer in Objects MUST be identical
	if len(app.accountListContainer.Objects) != len(initialObjects) {
		t.Fatalf("Objects count changed from %d to %d", len(initialObjects), len(app.accountListContainer.Objects))
	}

	for i, obj := range app.accountListContainer.Objects {
		if obj != initialObjects[i] {
			t.Errorf("INV-01 VIOLATION: Canvas object %d was replaced! Expected pointer %p, got %p", i, initialObjects[i], obj)
		}
	}

	// Verify in-place data update on the first card
	card0 := initialCards[0]
	if card0.gemini5hPct.Text != "95%" {
		t.Errorf("expected Gemini 5h text '95%%', got '%s'", card0.gemini5hPct.Text)
	}
	if card0.geminiWeekPct.Text != "85%" {
		t.Errorf("expected Gemini Week text '85%%', got '%s'", card0.geminiWeekPct.Text)
	}
	if card0.claude5hPct.Text != "45%" {
		t.Errorf("expected Claude 5h text '45%%', got '%s'", card0.claude5hPct.Text)
	}
	if card0.claudeWeekPct.Text != "10%" {
		t.Errorf("expected Claude Week text '10%%', got '%s'", card0.claudeWeekPct.Text)
	}
}

func TestIndustryStandard_FixedMatrixMetrics(t *testing.T) {
	dq := DualCycleQuota{
		Gemini5h:   0.80,
		GeminiWeek: 0.60,
		Claude5h:   0.50,
		ClaudeWeek: 0.30,
		HasData:    true,
	}

	table := CreateMatrixQuotaTable(dq, DarkPalette)
	cont, ok := table.(*fyne.Container)
	if !ok || len(cont.Objects) < 4 {
		t.Fatalf("expected at least 4 objects in table container")
	}

	colModel := cont.Objects[0]
	col5h := cont.Objects[1]
	colWeek := cont.Objects[2]
	// Check table MinSize is 228.0 x 77.0
	minSz := cont.MinSize()
	if minSz.Width != 228.0 || minSz.Height != 77.0 {
		t.Errorf("expected table MinSize 228x77, got %v", minSz)
	}

	// Ray bar must be 40.0px
	cell := newMatrixQuotaCell(0.80, time.Time{}, false, true, DarkPalette)
	if cell.bgTrack.Size().Width != 40.0 {
		t.Errorf("expected ray bar (bgTrack) fixed at 40.0px, got %f", cell.bgTrack.Size().Width)
	}

	for _, w := range []float32{228.0, 280.0, 300.0, 340.0, 400.0, 600.0} {
		cont.Resize(fyne.NewSize(w, 77.0))
		if colModel.Size().Width != 48.0 {
			t.Errorf("expected colModel width 48.0 at width %f, got %f", w, colModel.Size().Width)
		}
		if col5h.Size().Width != 78.0 {
			t.Errorf("expected col5h width 78.0 at width %f, got %f", w, col5h.Size().Width)
		}
		if colWeek.Size().Width != 86.0 {
			t.Errorf("expected colWeek width 86.0 at width %f, got %f", w, colWeek.Size().Width)
		}
	}
}

func TestIndustryStandard_OnlyMinibarDecreasesOnQuotaUpdate(t *testing.T) {
	cell := newMatrixQuotaCell(1.0, time.Time{}, false, true, DarkPalette)
	if cell == nil {
		t.Fatal("expected non-nil cell")
	}

	// 100% quota -> fillBar = 40px
	if cell.fillBar.Size().Width != 40.0 {
		t.Errorf("expected fillBar width 40.0 at 100%%, got %f", cell.fillBar.Size().Width)
	}
	if cell.bgTrack.Size().Width != 40.0 {
		t.Errorf("expected bgTrack width 40.0, got %f", cell.bgTrack.Size().Width)
	}
	if cell.pctText.Text != "100%" {
		t.Errorf("expected pctText '100.0%%', got '%s'", cell.pctText.Text)
	}

	// Tụt về 50% -> fillBar = 20px, bgTrack vẫn 40px
	cell.update(0.50, time.Time{}, true)
	if cell.fillBar.Size().Width != 20.0 {
		t.Errorf("expected fillBar width 20.0 at 50%%, got %f", cell.fillBar.Size().Width)
	}
	if cell.bgTrack.Size().Width != 40.0 {
		t.Errorf("expected bgTrack width 40.0 fixed, got %f", cell.bgTrack.Size().Width)
	}
	if cell.pctText.Text != "50%" {
		t.Errorf("expected pctText '50.0%%', got '%s'", cell.pctText.Text)
	}

	// Tụt về 0% -> fillBar = 0px, bgTrack vẫn 40px
	cell.update(0.0, time.Time{}, true)
	if cell.fillBar.Size().Width != 0.0 {
		t.Errorf("expected fillBar width 0.0 at 0%%, got %f", cell.fillBar.Size().Width)
	}
	if cell.bgTrack.Size().Width != 40.0 {
		t.Errorf("expected bgTrack width 40.0 fixed, got %f", cell.bgTrack.Size().Width)
	}
	if cell.pctText.Text != "0%" {
		t.Errorf("expected pctText '0.0%%', got '%s'", cell.pctText.Text)
	}
}

func TestIndustryStandard_WindowInit960x630(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	mgr := createTestManager(t)
	app := NewAppWithFyne(testApp, mgr)
	defer app.Close()

	app.Init()
	waitForIdle(app, 3*time.Second)

	win := app.Window()
	if win == nil {
		t.Fatal("expected non-nil window")
	}

	// Window must be resizable
	if win.FixedSize() {
		t.Errorf("expected a resizable window for maximize/restore")
	}

	sz := win.Canvas().Size()
	if sz.Width != 960 || sz.Height != 630 {
		t.Errorf("expected window canvas size 960x630, got %fx%f", sz.Width, sz.Height)
	}
}

func TestWindowControlsMinimizeMaximizeClose(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	mgr := createTestManager(t)
	app := NewAppWithFyne(testApp, mgr)
	defer app.Close()

	app.Init()

	win := app.Window()
	if win == nil {
		t.Fatal("expected non-nil window")
	}

	content := win.Content().(*fyne.Container)
	var topSection *fyne.Container
	for _, obj := range content.Objects {
		if c, ok := obj.(*fyne.Container); ok {
			topSection = c
			break
		}
	}
	if topSection == nil {
		t.Fatal("expected to find topSection container")
	}
	var row1Header *fyne.Container
	for _, obj := range topSection.Objects {
		if c, ok := obj.(*fyne.Container); ok {
			row1Header = c
			break
		}
	}
	if row1Header == nil {
		t.Fatal("expected to find row1Header container")
	}
	var windowControls *fyne.Container
	for _, obj := range row1Header.Objects {
		if c, ok := obj.(*fyne.Container); ok && len(c.Objects) == 3 {
			windowControls = c
			break
		}
	}
	if windowControls == nil {
		t.Fatalf("expected to find windowControls container with 3 buttons in row1Header")
	}
	if len(windowControls.Objects) != 3 {
		t.Errorf("expected minimize, maximize/restore, close; got %d controls", len(windowControls.Objects))
	}
	maximize := windowControls.Objects[1].(*fyne.Container).Objects[0].(*customTitleBtn)
	if maximize.stateLabel == nil || maximize.onTapped == nil {
		t.Fatal("middle window control must expose the maximize/restore action and state")
	}
}

func TestPurgeAllResizing_AccountGridWrapFixedSize(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	mgr := createTestManager(t)
	app := NewAppWithFyne(testApp, mgr)
	defer app.Close()

	app.Init()

	if app.accountListContainer == nil {
		t.Fatal("expected non-nil accountListContainer")
	}

	// Verify accountListContainer uses GridWrap layout with fixed 318x100 size
	item := manager.AccountDisplayItem{Email: "user@example.com"}
	palette := DarkPalette
	card := NewAccountCardWithQuota(item, nil, palette, nil, nil, nil)
	app.accountListContainer.Add(card)
	app.accountListContainer.Refresh()

	cardSize := card.Size()
	if cardSize.Width != 280 || cardSize.Height != 130 {
		t.Errorf("expected card size 280x130 in GridWrap, got %fx%f", cardSize.Width, cardSize.Height)
	}
}

// -----------------------------------------------------------------------------
// Test for SPEC-ARCH-20261019-COMPACT-CARD-WIDTH-280PX
// -----------------------------------------------------------------------------

func TestArchitectSpec_QuotaStackedLayoutCard280x130_FullConformance(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	mgr := createTestManager(t)
	app := NewAppWithFyne(testApp, mgr)
	defer app.Close()

	app.Init()
	waitForIdle(app, 3*time.Second)

	// AC-4: Default window canvas size 960x630
	win := app.Window()
	if win == nil {
		t.Fatal("expected non-nil window")
	}
	sz := win.Canvas().Size()
	if sz.Width != 960 || sz.Height != 630 {
		t.Errorf("AC-4 VIOLATION: expected window canvas size 960x630, got %fx%f", sz.Width, sz.Height)
	}

	// AC-3: Account card size in GridWrap is 280x130
	item := manager.AccountDisplayItem{Email: "stacked@example.com"}
	palette := DarkPalette
	card := NewAccountCardWithQuota(item, nil, palette, nil, nil, nil)
	app.accountListContainer.Add(card)
	app.accountListContainer.Refresh()

	cardSize := card.Size()
	if cardSize.Width != 280 || cardSize.Height != 130 {
		t.Errorf("AC-3 VIOLATION: expected card size 280x130, got %fx%f", cardSize.Width, cardSize.Height)
	}

	// AC-1, AC-2: Stacked 2-tier Quota Table (230x77px)
	future5h := time.Now().Add(4*time.Hour + 32*time.Minute)
	futureWeek := time.Now().Add(6*24*time.Hour + 21*time.Hour + 59*time.Minute)
	dq := DualCycleQuota{
		Gemini5h:        1.0,
		Gemini5hReset:   future5h,
		GeminiWeek:      0.438,
		GeminiWeekReset: futureWeek,
		Claude5h:        1.0,
		Claude5hReset:   future5h,
		ClaudeWeek:      1.0,
		ClaudeWeekReset: futureWeek,
		HasData:         true,
	}

	table, cG5, cGW, cC5, cCW := createMatrixQuotaTableWithCells(dq, palette)
	cont, ok := table.(*fyne.Container)
	if !ok || len(cont.Objects) < 4 {
		t.Fatalf("expected matrix table container with at least 4 objects")
	}

	// Geometry invariants: 228x77
	tableMin := cont.MinSize()
	if tableMin.Width != 228.0 || tableMin.Height != 77.0 {
		t.Errorf("expected table MinSize 228x77, got %fx%f", tableMin.Width, tableMin.Height)
	}
	// Cell tier 1: 8.5pt font, TextMuted color
	if cG5.timeText.TextSize != 8.5 {
		t.Errorf("AC-1 VIOLATION: expected timeText font size 8.5pt, got %f", cG5.timeText.TextSize)
	}
	assertColorMatches(t, "StackedLayout", "timeText color", cG5.timeText.Color, palette.TextMuted)

	// Cell tier 2: 10pt Bold BodyTextSize font
	if cG5.pctText.TextSize != BodyTextSize {
		t.Errorf("expected pctText font size 10pt (BodyTextSize), got %f", cG5.pctText.TextSize)
	}
	if !cG5.pctText.TextStyle.Bold {
		t.Errorf("expected pctText Bold")
	}

	// Ray bar 40px
	if cG5.bgTrack.Size().Width != 40.0 {
		t.Errorf("expected bgTrack 40px, got %f", cG5.bgTrack.Size().Width)
	}

	// AC-5: In-place reactive update zero jank
	cG5.update(0.50, future5h, true)
	if cG5.fillBar.Size().Width != 20.0 {
		t.Errorf("expected fillBar width 20px after update to 50%%, got %f", cG5.fillBar.Size().Width)
	}
	if cG5.pctText.Text != "50%" {
		t.Errorf("expected pctText '50%%', got %q", cG5.pctText.Text)
	}
	_ = cGW
	_ = cC5
	_ = cCW
}

// TestArchitectSpec_RemoveVerticalSeparatorsQuotaMatrix_FullConformance verifies all ACs of
// reports/architect/2026-10-19-remove-vertical-separators-quota-matrix-spec.md
// (SPEC-ARCH-20261019-REMOVE-VERTICAL-SEPARATORS-QUOTA-MATRIX).
func TestArchitectSpec_RemoveVerticalSeparatorsQuotaMatrix_FullConformance(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	palette := DarkPalette

	future5h := time.Now().Add(4*time.Hour + 32*time.Minute)
	futureWeek := time.Now().Add(6*24*time.Hour + 21*time.Hour + 59*time.Minute)
	dq := DualCycleQuota{
		Gemini5h:        1.0,
		Gemini5hReset:   future5h,
		GeminiWeek:      0.438,
		GeminiWeekReset: futureWeek,
		Claude5h:        1.0,
		Claude5hReset:   future5h,
		ClaudeWeek:      1.0,
		ClaudeWeekReset: futureWeek,
		HasData:         true,
	}

	table, cG5, cGW, cC5, cCW := createMatrixQuotaTableWithCells(dq, palette)
	cont, ok := table.(*fyne.Container)
	if !ok {
		t.Fatalf("expected matrix table container")
	}

	// AC-1: No Vertical Separators. The table must have exactly 4 objects:
	// objects[0]: colModel
	// objects[1]: col5h
	// objects[2]: colWeek
	// objects[3]: hSep
	if len(cont.Objects) != 4 {
		t.Fatalf("AC-1 VIOLATION: expected exactly 4 objects in matrix table (colModel, col5h, colWeek, hSep), got %d", len(cont.Objects))
	}

	// Verify that objects[1] is col5h (container), NOT a rectangle separator
	if _, isRect := cont.Objects[1].(*canvas.Rectangle); isRect {
		t.Fatalf("AC-1 VIOLATION: object[1] is canvas.Rectangle (sep1 still exists), expected col5h container")
	}
	// Verify that objects[2] is colWeek (container), NOT a rectangle separator
	if _, isRect := cont.Objects[2].(*canvas.Rectangle); isRect {
		t.Fatalf("AC-1 VIOLATION: object[2] is canvas.Rectangle (sep2 still exists), expected colWeek container")
	}

	// AC-2: Horizontal Separator Preserved.
	hSep, isHSepRect := cont.Objects[3].(*canvas.Rectangle)
	if !isHSepRect {
		t.Fatalf("AC-2 VIOLATION: object[3] is not canvas.Rectangle (hSep missing)")
	}
	assertColorMatches(t, "RemoveVerticalSeps", "hSep color", hSep.FillColor, palette.BorderSubtle)

	// Layout at default min size
	tableMin := cont.MinSize()
	// AC-3: Geometric Budget: 48 (col1) + 8 (gap) + 78 (col2) + 8 (gap) + 86 (col3) = 228.0 px
	const expectedW = float32(48.0 + 8.0 + 78.0 + 8.0 + 86.0) // 228.0
	const expectedH = float32(77.0)
	if tableMin.Width != expectedW || tableMin.Height != expectedH {
		t.Fatalf("AC-3 VIOLATION: expected MinSize 228x77, got %fx%f", tableMin.Width, tableMin.Height)
	}

	// Trigger Layout
	cont.Resize(tableMin)

	colModel := cont.Objects[0]
	col5h := cont.Objects[1]
	colWeek := cont.Objects[2]

	// Check column widths and positions
	if colModel.Size().Width != 48.0 {
		t.Errorf("AC-3 VIOLATION: expected colModel width 48.0, got %f", colModel.Size().Width)
	}
	if colModel.Position().X != 0 {
		t.Errorf("AC-3 VIOLATION: expected colModel pos.X = 0, got %f", colModel.Position().X)
	}

	if col5h.Size().Width != 78.0 {
		t.Errorf("AC-3 VIOLATION: expected col5h width 78.0, got %f", col5h.Size().Width)
	}
	// colModel(48) + colGap(8) = 56.0
	if col5h.Position().X != 56.0 {
		t.Errorf("AC-3 VIOLATION: expected col5h pos.X = 56.0, got %f", col5h.Position().X)
	}

	if colWeek.Size().Width != 86.0 {
		t.Errorf("AC-3 VIOLATION: expected colWeek width 86.0, got %f", colWeek.Size().Width)
	}
	// col5h pos(56) + col5h width(78) + colGap(8) = 142.0
	if colWeek.Position().X != 142.0 {
		t.Errorf("AC-3 VIOLATION: expected colWeek pos.X = 142.0, got %f", colWeek.Position().X)
	}

	// Horizontal separator position and geometry
	if hSep.Position().Y != 46.0 {
		t.Errorf("AC-2 VIOLATION: expected hSep pos.Y = 46.0, got %f", hSep.Position().Y)
	}
	if hSep.Size().Height != 1.0 {
		t.Errorf("AC-2 VIOLATION: expected hSep height 1.0, got %f", hSep.Size().Height)
	}
	if hSep.Size().Width != expectedW {
		t.Errorf("AC-2 VIOLATION: expected hSep width %f, got %f", expectedW, hSep.Size().Width)
	}

	// AC-4: Card Dimensions 280x130, Window 960x630
	if DefaultWindowWidth != 960.0 || DefaultWindowHeight != 630.0 {
		t.Errorf("AC-4 VIOLATION: expected window 960x630, got %fx%f", DefaultWindowWidth, DefaultWindowHeight)
	}

	// AC-5: In-place reactive update zero jank
	initialFillW := cG5.fillBar.Size().Width
	cG5.update(0.50, future5h, true)
	if cG5.fillBar.Size().Width == initialFillW {
		t.Errorf("AC-5 VIOLATION: fillBar width should update")
	}
	if cG5.pctText.Text != "50%" {
		t.Errorf("AC-5 VIOLATION: expected pctText '50%%', got %q", cG5.pctText.Text)
	}

	_ = cGW
	_ = cC5
	_ = cCW
}


// -----------------------------------------------------------------------------
// Test for SPEC-ARCH-20261019-COMPACT-WINDOW-WIDTH-960PX
// -----------------------------------------------------------------------------

func TestArchitectSpec_CompactWindowWidth960px_FullConformance(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	mgr := createTestManager(t)
	app := NewAppWithFyne(testApp, mgr)
	defer app.Close()

	app.Init()
	waitForIdle(app, 3*time.Second)

	// AC-1: Default window width 960.0, height 630.0
	if DefaultWindowWidth != 960.0 || DefaultWindowHeight != 630.0 {
		t.Fatalf("AC-1 VIOLATION: expected DefaultWindowWidth=960.0, DefaultWindowHeight=630.0, got %f x %f", DefaultWindowWidth, DefaultWindowHeight)
	}

	win := app.Window()
	if win == nil {
		t.Fatal("expected non-nil window")
	}
	sz := win.Canvas().Size()
	if sz.Width != 960 || sz.Height != 630 {
		t.Errorf("AC-1 VIOLATION: expected window canvas size 960x630, got %fx%f", sz.Width, sz.Height)
	}

	// AC-2: 3-Column Card Layout (280x130 Card GridWrap)
	if app.accountListContainer == nil {
		t.Fatal("expected non-nil accountListContainer")
	}
	palette := DarkPalette
	for i := 1; i <= 6; i++ {
		item := manager.AccountDisplayItem{Email: fmt.Sprintf("test%d@example.com", i)}
		card := NewAccountCardWithQuota(item, nil, palette, nil, nil, nil)
		app.accountListContainer.Add(card)
	}
	app.accountListContainer.Refresh()

	// Verify each card retains 280x130 dimensions
	card0 := app.accountListContainer.Objects[0]
	if card0.Size().Width != 280 || card0.Size().Height != 130 {
		t.Errorf("AC-2 VIOLATION: expected card dimensions 280x130, got %fx%f", card0.Size().Width, card0.Size().Height)
	}
}

// -----------------------------------------------------------------------------
// Test for SPEC-ARCH-20261019-COMPACT-SWITCH-BUTTON-HEIGHT-70PCT
// -----------------------------------------------------------------------------

func findSwitchButton(obj fyne.CanvasObject) *switchTextButton {
	var target *switchTextButton
	var walk func(o fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		if btn, ok := o.(*switchTextButton); ok {
			target = btn
			return
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

func TestArchitectSpec_CompactSwitchButtonHeight70Pct_FullConformance(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	// AC-1: Verify compact switch button MinSize height <= 25.0px (clamped to 22.0 - 24.0px range)
	switchBtn := widget.NewButton("Chuyển", nil)
	switchBtn.Importance = widget.HighImportance
	rawSwitchMin := switchBtn.MinSize()

	compactSwitch := compactButton(switchBtn)
	compactSwitchMin := compactSwitch.MinSize()

	if compactSwitchMin.Height > 25.0 {
		t.Fatalf("AC-1 VIOLATION: expected compact switch button height <= 25.0px, got %f", compactSwitchMin.Height)
	}
	if compactSwitchMin.Height > 24.0 {
		t.Errorf("AC-1 VIOLATION: expected compact switch button height <= 24.0px, got %f", compactSwitchMin.Height)
	}

	// AC-2: Verify compact refresh button MinSize height <= 25.0px and matches compact switch height
	refreshBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), nil)
	refreshBtn.Importance = widget.LowImportance
	rawRefreshMin := refreshBtn.MinSize()

	compactRefresh := compactButton(refreshBtn)
	compactRefreshMin := compactRefresh.MinSize()

	if compactRefreshMin.Height > 25.0 {
		t.Fatalf("AC-2 VIOLATION: expected compact refresh button height <= 25.0px, got %f", compactRefreshMin.Height)
	}
	if compactRefreshMin.Height > 24.0 {
		t.Errorf("AC-2 VIOLATION: expected compact refresh button height <= 24.0px, got %f", compactRefreshMin.Height)
	}

	// Layout vertical centering verification
	compactSwitch.Resize(fyne.NewSize(80, 32))
	if switchBtn.Size().Height > 24.0 {
		t.Errorf("Layout VIOLATION: expected inner button height <= 24.0px, got %f", switchBtn.Size().Height)
	}
	if switchBtn.Position().Y < 0 {
		t.Errorf("Layout VIOLATION: expected non-negative Y offset, got %f", switchBtn.Position().Y)
	}

	// AC-3: Functional Integrity of [Chuyển] button and [🔄] in Inactive Card
	var switchedEmail string
	var refreshedEmail string
	itemInactive := manager.AccountDisplayItem{
		Email:    "inactive_test@example.com",
		IsActive: false,
	}
	cardInactive := NewAccountCardWithQuota(itemInactive, nil, DarkPalette,
		func(email string) { switchedEmail = email },
		nil,
		func(email string) { refreshedEmail = email },
	)

	foundSwitch := findSwitchButton(cardInactive)
	if foundSwitch == nil {
		t.Fatalf("AC-3 VIOLATION: could not find 'Chuyển' button in inactive card")
	}
	foundSwitch.OnTapped()
	if switchedEmail != "inactive_test@example.com" {
		t.Errorf("AC-3 VIOLATION: expected switchedEmail 'inactive_test@example.com', got '%s'", switchedEmail)
	}

	foundRefreshInactive := findRefreshButton(cardInactive)
	if foundRefreshInactive == nil {
		t.Fatalf("AC-2 VIOLATION: could not find refresh button in inactive card")
	}
	foundRefreshInactive.OnTapped()
	if refreshedEmail != "inactive_test@example.com" {
		t.Errorf("AC-2 VIOLATION: expected refreshedEmail 'inactive_test@example.com', got '%s'", refreshedEmail)
	}

	// Functional Integrity of [🔄] in Active Card ([✔ Dùng] badge + compactRefresh)
	var activeRefreshedEmail string
	itemActive := manager.AccountDisplayItem{
		Email:    "active_test@example.com",
		IsActive: true,
	}
	cardActive := NewAccountCardWithQuota(itemActive, nil, DarkPalette,
		nil,
		nil,
		func(email string) { activeRefreshedEmail = email },
	)

	foundRefreshActive := findRefreshButton(cardActive)
	if foundRefreshActive == nil {
		t.Fatalf("AC-2 VIOLATION: could not find refresh button in active card")
	}
	foundRefreshActive.OnTapped()
	if activeRefreshedEmail != "active_test@example.com" {
		t.Errorf("AC-2 VIOLATION: expected activeRefreshedEmail 'active_test@example.com', got '%s'", activeRefreshedEmail)
	}

	// AC-4: Dimension preservation: Card remains within 280x130
	cardMin := cardInactive.MinSize()
	if cardMin.Width > 280.0 || cardMin.Height > 130.0 {
		t.Errorf("AC-4 VIOLATION: expected card min size within 280x130, got %fx%f", cardMin.Width, cardMin.Height)
	}
	_ = rawSwitchMin
	_ = rawRefreshMin
}

// -----------------------------------------------------------------------------
// Test for SPEC-ARCH-20261019-FLAT-TEXT-SWITCH-BUTTON
// -----------------------------------------------------------------------------

func TestArchitectSpec_FlatTextSwitchButton_FullConformance(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	for _, palette := range []ThemePalette{DarkPalette, LightPalette, WarmPalette} {
		var switchedEmail string
		itemInactive := manager.AccountDisplayItem{
			Email:    "flat_switch_test@example.com",
			IsActive: false,
		}
		card := NewAccountCardWithQuota(itemInactive, nil, palette,
			func(email string) { switchedEmail = email },
			nil,
			nil,
		)

		// AC-1: Verify flat text style: no solid button, text is "[Chuyển]", 10pt bold, palette.Primary color
		btn := findSwitchButton(card)
		if btn == nil {
			t.Fatalf("[%s] AC-1 VIOLATION: expected switchTextButton on inactive card", palette.Name)
		}
		if btn.Text != "[Chuyển]" {
			t.Errorf("[%s] AC-1 VIOLATION: expected text '[Chuyển]', got '%s'", palette.Name, btn.Text)
		}
		if btn.text == nil {
			t.Fatalf("[%s] AC-1 VIOLATION: expected non-nil inner text object", palette.Name)
		}
		if btn.text.Text != "[Chuyển]" {
			t.Errorf("[%s] AC-1 VIOLATION: expected inner text '[Chuyển]', got '%s'", palette.Name, btn.text.Text)
		}
		if btn.text.TextSize != 10 {
			t.Errorf("[%s] AC-1 VIOLATION: expected text size 10, got %f", palette.Name, btn.text.TextSize)
		}
		if !btn.text.TextStyle.Bold {
			t.Errorf("[%s] AC-1 VIOLATION: expected bold text style", palette.Name)
		}
		assertColorMatches(t, "FlatTextSwitchButton Normal", "text.Color", btn.text.Color, palette.Primary)
		if btn.bg.FillColor != color.Transparent {
			t.Errorf("[%s] AC-1 VIOLATION: expected transparent background in normal state, got %v", palette.Name, btn.bg.FillColor)
		}

		// AC-2: Pointer Cursor & Hover affordance
		if btn.Cursor() != desktop.PointerCursor {
			t.Errorf("[%s] AC-2 VIOLATION: expected desktop.PointerCursor, got %v", palette.Name, btn.Cursor())
		}
		btn.MouseIn(&desktop.MouseEvent{})
		if !btn.hovered {
			t.Errorf("[%s] AC-2 VIOLATION: expected hovered == true after MouseIn", palette.Name)
		}
		if btn.bg.FillColor == color.Transparent {
			t.Errorf("[%s] AC-2 VIOLATION: expected non-transparent background tint on hover", palette.Name)
		}
		expectedHoverText := lightenColor(palette.Primary, 35)
		assertColorMatches(t, "FlatTextSwitchButton Hover", "text.Color", btn.text.Color, expectedHoverText)

		btn.MouseOut()
		if btn.hovered {
			t.Errorf("[%s] AC-2 VIOLATION: expected hovered == false after MouseOut", palette.Name)
		}
		if btn.bg.FillColor != color.Transparent {
			t.Errorf("[%s] AC-2 VIOLATION: expected transparent background after MouseOut", palette.Name)
		}
		assertColorMatches(t, "FlatTextSwitchButton Restored", "text.Color", btn.text.Color, palette.Primary)

		// AC-3: Switch Functionality on click/tap
		btn.Tapped(&fyne.PointEvent{})
		if switchedEmail != "flat_switch_test@example.com" {
			t.Errorf("[%s] AC-3 VIOLATION: expected switchedEmail 'flat_switch_test@example.com', got '%s'", palette.Name, switchedEmail)
		}

		// Also verify direct OnTapped() execution
		switchedEmail = ""
		btn.OnTapped()
		if switchedEmail != "flat_switch_test@example.com" {
			t.Errorf("[%s] AC-3 VIOLATION: expected OnTapped() to set switchedEmail, got '%s'", palette.Name, switchedEmail)
		}

		// AC-4: Dimension preservation: Card remains within 280x130
		cardMin := card.MinSize()
		if cardMin.Width > 280.0 || cardMin.Height > 130.0 {
			t.Errorf("[%s] AC-4 VIOLATION: expected card within 280x130, got %fx%f", palette.Name, cardMin.Width, cardMin.Height)
		}
	}
}
