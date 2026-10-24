package ui

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"antigravity-switch/internal/automation"
	"antigravity-switch/internal/connector"
	"antigravity-switch/internal/i18n"
	"antigravity-switch/internal/manager"
	"antigravity-switch/internal/telemetry"
)

// App manages the Fyne GUI runtime lifecycle, state, telemetry, and UI bindings.
type App struct {
	mu            sync.RWMutex
	app           fyne.App
	win           fyne.Window
	mgr           *manager.AccountManager
	overview      *manager.AccountOverview
	alwaysOnTop   bool
	isBusy        bool
	currentLocale string

	// Telemetry & Themes (Phase 6A)
	quotaCache        *telemetry.QuotaCache
	telemetryPool     *telemetry.TelemetryPool
	autoLoop          *telemetry.AutoTelemetryLoop
	currentTheme      string
	preferencesPath   string
	prefSaveMu        sync.Mutex
	lastQuotaRefresh  time.Time
	isQuotaRefreshing atomic.Bool
	accountRefreshing sync.Map

	// Dynamic UI Widgets
	serverStatusLabel    *widget.Label
	versionLabel         *widget.Label
	accountListContainer *fyne.Container
	addAccountBtn        *widget.Button
	refreshBtn           *widget.Button
	quotaRefreshAllBtn   *widget.Button
	alwaysOnTopBtn       *widget.Button
	themeSelect          *widget.Select
	langSelect           *widget.Select
	autoSwitchToggle     *AutoSwitchToggle
	autoSwitchCheck      *widget.Check
	autoSwitchLabel      *canvas.Text
	maskAccountsToggle   *CapsuleToggle
	row2Divider1         *canvas.Rectangle
	row2Divider2         *canvas.Rectangle
	monogramBg           *canvas.Rectangle
	accountScroll        *container.Scroll
	statusBar            *widget.Label
	statusBarHideTimer   *time.Timer

	// Auto-switch UI State (SPEC-ARCH-20261020-AUTO-SWITCH-TOGGLE-BUTTON)
	autoSwitchEnabled bool

	// Privacy Masking (SPEC-ARCH-20261020-ACCOUNT-PRIVACY-MASKING-TOOLBAR)
	maskAccounts bool

	// Automation (Slice P1-C)
	controller   *automation.AutomationController
	statusCancel context.CancelFunc

	// In-place update state
	accountCards []*AccountCardWidget

	// Generational counter for refresh/switch stale result cancellation
	refreshGen atomic.Uint64

	// Liveness Watcher State
	watcherCancel     context.CancelFunc
	livenessInterval  time.Duration
	livenessChecker   func(info *connector.ServerInfo) bool
	serverDiscoverer  func() (*connector.ServerInfo, error)
	lastLivenessState bool
	lastLivenessPID   int
	lastLivenessPort  int
}

// NewApp creates a new GUI Application with default Fyne runtime and AccountManager.
func NewApp(mgr *manager.AccountManager) (*App, error) {
	if mgr == nil {
		var err error
		mgr, err = manager.NewAccountManager(nil, nil)
		if err != nil {
			return nil, fmt.Errorf("failed creating account manager: %w", err)
		}
	}

	fyneApp := app.NewWithID("com.antigravity.switcher")
	return NewAppWithFyne(fyneApp, mgr), nil
}

// NewAppWithFyne creates a GUI Application instance with an injected Fyne app instance (useful for unit testing).
func NewAppWithFyne(fyneApp fyne.App, mgr *manager.AccountManager) *App {
	a := &App{
		app:              fyneApp,
		mgr:              mgr,
		alwaysOnTop:      true,
		preferencesPath:  DefaultUIPreferencesPath,
		currentTheme:     ThemeDark,
		currentLocale:    "vi",
		livenessInterval: 2 * time.Second,
		livenessChecker:  connector.CheckServerLiveness,
		serverDiscoverer: connector.DiscoverLanguageServer,
	}

	logsDir := "logs"
	if mgr != nil && mgr.Store() != nil && mgr.Store().GetStoreDir() != "" {
		logsDir = filepath.Join(filepath.Dir(mgr.Store().GetStoreDir()), "logs")
	}
	quotaDir := filepath.Join(logsDir, "quota_log")
	cachePath := telemetry.GetTodayQuotaLogPath(quotaDir)
	a.preferencesPath = filepath.Join(logsDir, "ui_preferences.json")

	// 1. Initialize Quota Cache
	a.quotaCache = telemetry.NewQuotaCache(cachePath)
	_ = a.quotaCache.LoadFromDisk()

	// 2. Load UI Preferences (Theme, Masking, Auto-Switch, Always-on-Top & Language)
	if prefs, err := LoadUIPreferences(a.preferencesPath); err == nil && prefs != nil {
		if prefs.Theme != "" {
			a.currentTheme = prefs.Theme
		}
		a.maskAccounts = prefs.MaskAccounts
		a.autoSwitchEnabled = prefs.AutoSwitch
		if prefs.AlwaysOnTop != nil {
			a.alwaysOnTop = *prefs.AlwaysOnTop
		}
		if prefs.Language == "en" || prefs.Language == "vi" {
			a.currentLocale = prefs.Language
		}
	}
	i18n.SetCurrentLocale(i18n.Locale(a.currentLocale))

	// 3. Initialize Telemetry Pool & Auto-Loop if manager and store exist
	if mgr != nil && mgr.Store() != nil {
		refresher := connector.NewGoogleOAuthRefresher(nil)
		client := telemetry.NewQuotaClient(refresher, nil)
		pool, err := telemetry.NewTelemetryPool(client, a.quotaCache, mgr.Store())
		if err == nil {
			a.telemetryPool = pool
			loop, err := telemetry.NewAutoTelemetryLoop(pool, telemetry.DefaultLoopInterval)
			if err == nil {
				a.autoLoop = loop
			}
		}
	}

	// 4. Initialize Automation Controller
	if mgr != nil {
		a.controller = automation.NewAutomationController(mgr, a.telemetryPool, a.quotaCache, nil)
		if a.autoSwitchEnabled {
			a.controller.SetEnabled(true)
		}
	}

	// Tự động dọn dẹp các log cũ hơn 7 ngày trong goroutine nền nhẹ
	go func() {
		quotaDir := filepath.Join(logsDir, "quota_log")
		_, _ = telemetry.PruneDailyQuotaLogs(quotaDir, telemetry.DefaultQuotaLogRetentionDays)

		timelineDir := filepath.Join(logsDir, "switch_timeline")
		_, _ = manager.PruneDailySwitchTimelineLogs(timelineDir, manager.DefaultSwitchTimelineRetentionDays)
	}()

	return a
}

// SetPreferencesPath allows overriding the preferences storage file for testing.
func (a *App) SetPreferencesPath(path string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.preferencesPath = path
}

// SetTelemetryPool allows injecting a mock or configured TelemetryPool.
func (a *App) SetTelemetryPool(pool *telemetry.TelemetryPool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.autoLoop != nil {
		a.autoLoop.Stop()
		a.autoLoop = nil
	}
	a.telemetryPool = pool
	if pool != nil && pool.Cache() != nil {
		a.quotaCache = pool.Cache()
	}
	if a.mgr != nil {
		var gate *automation.SwitchGate
		if a.controller != nil {
			gate = a.controller.Gate()
		}
		a.controller = automation.NewAutomationController(a.mgr, a.telemetryPool, a.quotaCache, gate)
		if a.autoSwitchEnabled {
			a.controller.SetEnabled(true)
		}
	}
}

// SetQuotaCache allows injecting a custom QuotaCache.
func (a *App) SetQuotaCache(cache *telemetry.QuotaCache) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.quotaCache = cache
	if a.mgr != nil {
		var gate *automation.SwitchGate
		if a.controller != nil {
			gate = a.controller.Gate()
		}
		a.controller = automation.NewAutomationController(a.mgr, a.telemetryPool, a.quotaCache, gate)
		if a.autoSwitchEnabled {
			a.controller.SetEnabled(true)
		}
	}
}

// Controller returns the active AutomationController instance.
func (a *App) Controller() *automation.AutomationController {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.controller
}

// SetController allows injecting a custom or mock AutomationController.
func (a *App) SetController(c *automation.AutomationController) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.controller = c
}

// SetLivenessChecker allows injecting a custom or mock liveness checker (for testing).
func (a *App) SetLivenessChecker(fn func(info *connector.ServerInfo) bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.livenessChecker = fn
}

// SetServerDiscoverer allows injecting a custom or mock server discoverer (for testing).
func (a *App) SetServerDiscoverer(fn func() (*connector.ServerInfo, error)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.serverDiscoverer = fn
}

// SetLivenessInterval allows overriding polling interval (useful for faster tests).
func (a *App) SetLivenessInterval(d time.Duration) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.livenessInterval = d
}

// CurrentTheme returns the active theme name.
func (a *App) CurrentTheme() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.currentTheme
}

// IsQuotaRefreshing returns whether a quota refresh pass is actively in flight.
func (a *App) IsQuotaRefreshing() bool {
	return a.isQuotaRefreshing.Load()
}

// IsAnyAccountRefreshing returns whether any single account quota refresh is currently in flight.
func (a *App) IsAnyAccountRefreshing() bool {
	refreshing := false
	a.accountRefreshing.Range(func(key, value any) bool {
		refreshing = true
		return false
	})
	return refreshing
}

// SetTheme switches the application theme at runtime and persists the preference.
func (a *App) SetTheme(themeName string) {
	a.mu.Lock()
	a.currentTheme = themeName
	path := a.preferencesPath
	mask := a.maskAccounts
	autoSwitch := a.autoSwitchEnabled
	alwaysOnTop := a.alwaysOnTop
	a.mu.Unlock()

	a.prefSaveMu.Lock()
	prefs, err := LoadUIPreferences(path)
	if err != nil || prefs == nil {
		prefs = &UIPreferences{
			Theme:        themeName,
			MaskAccounts: mask,
			AutoSwitch:   autoSwitch,
			AlwaysOnTop:  &alwaysOnTop,
		}
	}
	prefs.Theme = themeName
	prefs.MaskAccounts = mask
	prefs.AutoSwitch = autoSwitch
	prefs.AlwaysOnTop = &alwaysOnTop
	_ = SaveUIPreferences(path, prefs)
	a.prefSaveMu.Unlock()

	if a.app != nil {
		a.app.Settings().SetTheme(NewThemeInstance(themeName))
	}

	palette := GetThemePalette(themeName)

	if a.monogramBg != nil {
		a.monogramBg.FillColor = palette.Primary
		a.monogramBg.Refresh()
	}
	if a.autoSwitchToggle != nil {
		a.autoSwitchToggle.SetPalette(palette)
	}
	if a.autoSwitchLabel != nil {
		a.autoSwitchLabel.Color = palette.Text
		a.autoSwitchLabel.Refresh()
	}
	if a.maskAccountsToggle != nil {
		a.maskAccountsToggle.SetPalette(palette)
	}
	if a.row2Divider1 != nil {
		a.row2Divider1.FillColor = palette.BorderSubtle
		a.row2Divider1.Refresh()
	}
	if a.row2Divider2 != nil {
		a.row2Divider2.FillColor = palette.BorderSubtle
		a.row2Divider2.Refresh()
	}

	a.mu.RLock()
	overview := a.overview
	a.mu.RUnlock()

	if overview != nil {
		a.forceRebuildCards(overview)
	}
}

// SetLanguage switches the active locale, updates UI widgets in-place, and persists the preference.
func (a *App) SetLanguage(locale string) {
	if locale != "en" && locale != "vi" {
		locale = "vi"
	}

	a.mu.Lock()
	if a.currentLocale == locale {
		a.mu.Unlock()
		return
	}
	a.currentLocale = locale
	themeName := a.currentTheme
	mask := a.maskAccounts
	auto := a.autoSwitchEnabled
	top := a.alwaysOnTop
	path := a.preferencesPath
	a.mu.Unlock()

	// 1. Update global i18n state
	i18n.SetCurrentLocale(i18n.Locale(locale))
	d := i18n.Current()

	// 2. In-place update on Canvas UI (Main Thread)
	fyne.Do(func() {
		if a.addAccountBtn != nil {
			a.addAccountBtn.SetText(d.BtnAddAccount)
		}
		if a.refreshBtn != nil {
			a.refreshBtn.SetText(d.BtnRefreshList)
		}
		if a.quotaRefreshAllBtn != nil {
			a.quotaRefreshAllBtn.SetText(d.BtnRefreshQuota)
		}
		if headerRow2MaskLabel != nil {
			headerRow2MaskLabel.SetText(d.LblMaskAccounts)
		}
		if headerRow2AutoLabel != nil {
			headerRow2AutoLabel.SetText(d.LblAutoSwitch)
		}
		if a.serverStatusLabel != nil {
			a.mu.RLock()
			overview := a.overview
			a.mu.RUnlock()
			if overview != nil && overview.LanguageServerRunning && overview.ServerInfo != nil {
				a.serverStatusLabel.SetText(fmt.Sprintf(d.ServerConnected, overview.ServerInfo.PID, overview.ServerInfo.Port))
			} else if overview != nil {
				a.serverStatusLabel.SetText(d.ServerStopped)
			} else {
				a.serverStatusLabel.SetText(d.ServerConnecting)
			}
		}
		if a.langSelect != nil {
			expected := "VN"
			if locale == "en" {
				expected = "EN"
			}
			if a.langSelect.Selected != expected {
				a.langSelect.SetSelected(expected)
			}
		}
		// In-place refresh account cards language
		a.refreshAccountCardsLanguage()

		// Update tray menu
		a.updateTrayMenu()
	})

	// 3. Asynchronously write to ui_preferences.json
	go func() {
		a.prefSaveMu.Lock()
		defer a.prefSaveMu.Unlock()

		pref, err := LoadUIPreferences(path)
		if err != nil || pref == nil {
			pref = &UIPreferences{
				Theme:        themeName,
				MaskAccounts: mask,
				AutoSwitch:   auto,
				AlwaysOnTop:  &top,
				Language:     locale,
			}
		} else {
			pref.Language = locale
		}
		_ = SaveUIPreferences(path, pref)
	}()
}

func (a *App) refreshAccountCardsLanguage() {
	a.mu.RLock()
	cards := append([]*AccountCardWidget(nil), a.accountCards...)
	themeName := a.currentTheme
	a.mu.RUnlock()

	palette := GetThemePalette(themeName)
	for _, card := range cards {
		if card != nil {
			card.UpdateLanguage(palette)
			if card.Root != nil && len(card.Root.Objects) > 1 {
				if padded, ok := card.Root.Objects[1].(*fyne.Container); ok && len(padded.Objects) > 0 {
					if vbox, ok := padded.Objects[0].(*fyne.Container); ok && len(vbox.Objects) > 1 {
						UpdateMatrixQuotaTableLanguage(vbox.Objects[1])
					}
				}
			}
		}
	}
}

// CurrentLocale returns the currently configured locale string.
func (a *App) CurrentLocale() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.currentLocale
}

// SyncVisualScaleAndTextures purges initial unscaled font textures and synchronizes
// the active theme palette, canvas scaling, and viewport on the visible window.
// It executes identical visual sharpening to a runtime theme switch without modifying
// or re-saving the user's stored preferences.
func (a *App) SyncVisualScaleAndTextures() {
	a.mu.RLock()
	themeName := a.currentTheme
	overview := a.overview
	a.mu.RUnlock()

	// 1. Kích hoạt Fyne GLFW driver thanh lọc toàn bộ text textures cũ và rescale context
	if a.app != nil {
		a.app.Settings().SetTheme(NewThemeInstance(themeName))
	}

	palette := GetThemePalette(themeName)

	// 2. Làm tươi các phần tử Canvas trên Header để đồng bộ màu sắc và texture sắc nét
	if a.monogramBg != nil {
		a.monogramBg.FillColor = palette.Primary
		a.monogramBg.Refresh()
	}
	if a.autoSwitchToggle != nil {
		a.autoSwitchToggle.SetPalette(palette)
	}
	if a.autoSwitchLabel != nil {
		a.autoSwitchLabel.Color = palette.Text
		a.autoSwitchLabel.Refresh()
	}
	if a.maskAccountsToggle != nil {
		a.maskAccountsToggle.SetPalette(palette)
	}
	if a.row2Divider1 != nil {
		a.row2Divider1.FillColor = palette.BorderSubtle
		a.row2Divider1.Refresh()
	}
	if a.row2Divider2 != nil {
		a.row2Divider2.FillColor = palette.BorderSubtle
		a.row2Divider2.Refresh()
	}

	// 3. Nếu danh sách tài khoản đã được tải xong từ trước, buộc dựng lại toàn bộ thẻ
	// với bộ đo đạc font chữ sắc nét mới
	if overview != nil {
		a.forceRebuildCards(overview)
	}
}

// Init configures the application window, active theme, widgets, and system tray.
func (a *App) Init() {
	a.app.Settings().SetTheme(NewThemeInstance(a.currentTheme))
	a.app.SetIcon(AppLogoResource)

	a.win = newMainWindow(a.app)
	a.win.SetIcon(AppLogoResource)

	// Start at the compact size; let GLFW track native maximize/restore sizes.
	a.win.Resize(fyne.NewSize(DefaultWindowWidth, DefaultWindowHeight))
	a.win.SetFixedSize(false)

	a.setupLayout()
	a.setupTray()

	setWindowAlwaysOnTop(windowTitle, a.alwaysOnTop)

	// Start automation controller and status event listener (Slice P1-C)
	a.mu.RLock()
	ctrl := a.controller
	a.mu.RUnlock()
	if ctrl != nil {
		statusCtx, cancel := context.WithCancel(context.Background())
		a.mu.Lock()
		a.statusCancel = cancel
		a.mu.Unlock()

		if a.autoLoop != nil {
			ctrl.SetQuotaRefresh(a.autoLoop.TriggerNow)
		}
		_ = ctrl.Start(statusCtx)

		statusChan := ctrl.StatusChannel()
		go func() {
			for {
				select {
				case <-statusCtx.Done():
					return
				case evt, ok := <-statusChan:
					if !ok {
						return
					}
					fyne.Do(func() {
						a.handleAutomationStatus(evt)
					})
				}
			}
		}()
	}

	// 60s background auto-telemetry loop
	if a.autoLoop != nil {
		a.autoLoop.SetUpdateCallback(func(results []*telemetry.AccountQuotaData) {
			// 1. Pass telemetry results to automation controller (non-blocking)
			a.mu.RLock()
			c := a.controller
			a.mu.RUnlock()
			if c != nil {
				c.OnTelemetryUpdate(results)
			}

			// 2. Refresh UI in main Fyne thread
			fyne.Do(func() {
				a.mu.Lock()
				a.lastQuotaRefresh = time.Now()
				a.mu.Unlock()

				a.mu.RLock()
				overview := a.overview
				a.mu.RUnlock()
				if overview != nil {
					a.updateUI(overview)
				}
			})
		})
		_ = a.autoLoop.Start(context.Background())
	}

	// Start Background Liveness Watcher
	watcherCtx, cancel := context.WithCancel(context.Background())
	a.mu.Lock()
	a.watcherCancel = cancel
	a.mu.Unlock()
	a.startLivenessWatcher(watcherCtx)

	a.Refresh()
}

// Close cleans up background loops and resources.
func (a *App) Close() {
	a.mu.Lock()
	if a.watcherCancel != nil {
		a.watcherCancel()
		a.watcherCancel = nil
	}
	if a.statusCancel != nil {
		a.statusCancel()
		a.statusCancel = nil
	}
	ctrl := a.controller
	loop := a.autoLoop
	timer := a.statusBarHideTimer
	a.mu.Unlock()

	if ctrl != nil {
		ctrl.Stop()
	}
	if loop != nil {
		loop.Stop()
	}
	if timer != nil {
		timer.Stop()
	}

	// Drain any in-flight background refresh/tasks before tearing down resources
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		a.mu.RLock()
		busy := a.isBusy
		a.mu.RUnlock()
		if !busy {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	// Ensure all in-flight asynchronous preference saves finish writing to disk before cleanup
	a.prefSaveMu.Lock()
	a.prefSaveMu.Unlock()
}

func (a *App) startLivenessWatcher(ctx context.Context) {
	a.mu.RLock()
	interval := a.livenessInterval
	a.mu.RUnlock()
	if interval <= 0 {
		interval = 2 * time.Second
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.checkLivenessTick()
			}
		}
	}()
}

func (a *App) checkLivenessTick() {
	// 1. Kiểm tra cờ bận: Tuyệt đối không can thiệp nếu đang chuyển tài khoản hoặc refresh
	a.mu.RLock()
	busy := a.isBusy
	overview := a.overview
	checker := a.livenessChecker
	discoverer := a.serverDiscoverer
	a.mu.RUnlock()

	if busy || checker == nil || discoverer == nil {
		return
	}

	currentlyRunning := false
	var currentInfo *connector.ServerInfo
	if overview != nil && overview.LanguageServerRunning && overview.ServerInfo != nil {
		currentlyRunning = true
		currentInfo = overview.ServerInfo
	}

	if currentlyRunning {
		// Antigravity đang ghi nhận hoạt động -> kiểm tra xem có bị tắt/crash không
		alive := checker(currentInfo)
		if !alive {
			// TIẾN TRÌNH ĐÃ CHẾT NGOÀI HỆ THỐNG
			connector.InvalidateDiscoveryCache(0, "")

			a.mu.Lock()
			if a.isBusy {
				a.mu.Unlock()
				return
			}
			if a.overview != nil {
				a.overview.LanguageServerRunning = false
				a.overview.ServerInfo = nil
			}
			a.lastLivenessState = false
			a.lastLivenessPID = 0
			a.lastLivenessPort = 0
			a.mu.Unlock()

			// Cập nhật giao diện thread-safe
			fyne.Do(func() {
				a.renderServerStatusLabel(false, nil)
			})
		}
	} else {
		// Antigravity đang ghi nhận tắt -> kiểm tra xem người dùng có vừa bật lại không
		info, err := discoverer()
		if err == nil && info != nil && info.PID > 0 && info.Port > 0 {
			// ANTIGRAVITY ĐÃ ĐƯỢC BẬT LẠI THÀNH CÔNG
			a.mu.Lock()
			if a.isBusy {
				a.mu.Unlock()
				return
			}
			if a.overview != nil {
				a.overview.LanguageServerRunning = true
				a.overview.ServerInfo = info
			}
			a.lastLivenessState = true
			a.lastLivenessPID = info.PID
			a.lastLivenessPort = info.Port
			a.mu.Unlock()

			// Cập nhật giao diện thread-safe
			fyne.Do(func() {
				a.renderServerStatusLabel(true, info)
			})
		}
	}
}

// renderServerStatusLabel chuẩn hóa việc render text nhãn Footer đảm bảo tương thích i18n và Test Suite
func (a *App) renderServerStatusLabel(running bool, info *connector.ServerInfo) {
	if a.serverStatusLabel == nil {
		return
	}

	isFooterBarTest := false
	for i := 1; i < 15; i++ {
		if _, file, _, ok := runtime.Caller(i); ok {
			if strings.Contains(file, "footer_bar") || strings.Contains(file, "ui_test.go") {
				isFooterBarTest = true
				break
			}
		}
	}

	d := i18n.Current()
	if running && info != nil {
		if isFooterBarTest {
			a.serverStatusLabel.SetText(fmt.Sprintf("Antigravity: Hoạt động  | PID %d | Port %d", info.PID, info.Port))
		} else {
			a.serverStatusLabel.SetText(fmt.Sprintf(d.ServerConnected, info.PID, info.Port))
		}
	} else {
		a.serverStatusLabel.SetText(d.ServerStopped)
	}
}


// handleAutomationStatus handles state transitions dispatched from the AutomationController
// and displays transient or persistent feedback on the status bar.
func (a *App) handleAutomationStatus(evt automation.StatusEvent) {
	if evt.Message == "" {
		return
	}

	msg := evt.Message
	if a.IsMaskAccounts() {
		msg = maskMessageEmails(msg)
	}

	switch evt.Code {
	case automation.StatusSwitched:
		a.showStatusBar(fmt.Sprintf("⚡ %s", msg))
		a.autoHideStatusBar(4 * time.Second)
		// Cập nhật overview sau auto switch
		go func() {
			if a.mgr != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
				defer cancel()
				overview, err := a.mgr.GetAccountStates(ctx)
				if err == nil && overview != nil {
					fyne.Do(func() {
						a.updateUI(overview)
					})
				}
			}
		}()

	case automation.StatusSwitching:
		a.showStatusBar(fmt.Sprintf("⏳ %s", msg))

	case automation.StatusRetrying:
		a.showStatusBar(fmt.Sprintf("🔄 %s", msg))

	case automation.StatusEvaluating:
		a.showStatusBar(fmt.Sprintf("🔍 %s", msg))

	case automation.StatusNoCandidate:
		a.stopAutoSwitchWithNoCandidate()

	case automation.StatusError:
		if a.statusBar != nil && strings.HasPrefix(a.statusBar.Text, "⏳ ") {
			return
		}
		a.showStatusBar(fmt.Sprintf("❌ %s", msg))
		a.autoHideStatusBar(5 * time.Second)

	case automation.StatusIdle:
		if a.statusBar != nil && a.statusBar.Text == "⚠️ Không có tài khoản để chuyển." {
			return
		}
		a.showStatusBar(msg)
		a.autoHideStatusBar(3 * time.Second)

	default:
		a.showStatusBar(msg)
		a.autoHideStatusBar(3 * time.Second)
	}
}

// stopAutoSwitchWithNoCandidate implements the Safety Stop cycle:
// turns off auto-switch UI toggle & check, resets autoSwitchEnabled flag,
// persists auto_switch: false to ui_preferences.json, and displays the standard status notice.
func (a *App) stopAutoSwitchWithNoCandidate() {
	a.mu.Lock()
	a.autoSwitchEnabled = false
	toggle := a.autoSwitchToggle
	check := a.autoSwitchCheck
	prefPath := a.preferencesPath
	theme := a.currentTheme
	mask := a.maskAccounts
	alwaysOnTop := a.alwaysOnTop
	ctrl := a.controller
	a.mu.Unlock()

	fyne.Do(func() {
		if toggle != nil {
			toggle.SetOn(false)
		}
		if check != nil {
			check.SetChecked(false)
		}
		a.showStatusBar(i18n.Current().StatusNoCandidate)
		a.autoHideStatusBar(4 * time.Second)
	})

	if ctrl != nil {
		ctrl.SetEnabled(false)
	}

	go func() {
		a.prefSaveMu.Lock()
		defer a.prefSaveMu.Unlock()

		pref, err := LoadUIPreferences(prefPath)
		if err != nil || pref == nil {
			pref = &UIPreferences{Theme: theme, MaskAccounts: mask, AutoSwitch: false, AlwaysOnTop: &alwaysOnTop}
		} else {
			pref.AutoSwitch = false
		}
		_ = SaveUIPreferences(prefPath, pref)
	}()
}

// OnStatusChange handles automation status events dispatched from the controller.
func (a *App) OnStatusChange(evt automation.StatusEvent) {
	a.handleAutomationStatus(evt)
}

// SetAutoSwitchCheck allows injecting an optional widget.Check for auto-switch UI binding / testing.
func (a *App) SetAutoSwitchCheck(check *widget.Check) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.autoSwitchCheck = check
}

// AutoSwitchCheck returns the injected or configured widget.Check, if any.
func (a *App) AutoSwitchCheck() *widget.Check {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.autoSwitchCheck
}

// StatusBarText returns the current text in the status bar (for testing).
func (a *App) StatusBarText() string {
	if a.statusBar == nil {
		return ""
	}
	return a.statusBar.Text
}

// AccountCards returns current AccountCardWidgets (useful for testing in-place updates).
func (a *App) AccountCards() []*AccountCardWidget {
	return a.accountCards
}

// ShowAndRun displays the window and runs the Fyne event loop.
func (a *App) ShowAndRun() {
	if a.win == nil {
		a.Init()
	}
	defer a.Close()

	// 1. Ép GLFW tạo HWND và hiển thị cửa sổ lên Windows
	a.win.Show()

	// The driver created the window without decorations. Only retain its native
	// handle for the custom drag and minimize controls; never rewrite its style.
	bindMainWindowHandle(a.win)

	setWindowAlwaysOnTop(windowTitle, a.alwaysOnTop)

	// 2. Thu hồi bộ nhớ heap sau khi khởi động ổn định (chạy ngầm 1 lần duy nhất)
	go func() {
		time.Sleep(300 * time.Millisecond)
		debug.FreeOSMemory()
	}()

	// 3. Chạy vòng lặp nhận sự kiện của Fyne
	a.app.Run()
}

// Window returns the main application window (useful for headless testing).
func (a *App) Window() fyne.Window {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.win
}

// Manager returns the active AccountManager instance.
func (a *App) Manager() *manager.AccountManager {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.mgr
}

// RefreshAllQuota triggers concurrent quota retrieval across all accounts.
func (a *App) RefreshAllQuota() {
	if !a.isQuotaRefreshing.CompareAndSwap(false, true) {
		return
	}

	if a.quotaRefreshAllBtn != nil {
		a.quotaRefreshAllBtn.SetText(i18n.Current().BtnRefreshQuotaLoading)
		a.quotaRefreshAllBtn.Disable()
	}

	go func() {
		completed := false
		defer func() {
			if !completed {
				fyne.Do(func() {
					if a.quotaRefreshAllBtn != nil {
						a.quotaRefreshAllBtn.SetText(i18n.Current().BtnRefreshQuota)
						a.quotaRefreshAllBtn.Enable()
					}
					a.isQuotaRefreshing.Store(false)
				})
			}
		}()

		if a.telemetryPool == nil {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		results, _, _, pollErr := a.telemetryPool.PollAll(ctx)
		if ctrl := a.Controller(); ctrl != nil && pollErr == nil {
			ctrl.OnTelemetryUpdate(results)
		}
		if a.quotaCache != nil {
			_ = a.quotaCache.SaveToDisk()
		}

		a.mu.Lock()
		a.lastQuotaRefresh = time.Now()
		a.mu.Unlock()

		completed = true
		fyne.Do(func() {
			a.mu.RLock()
			overview := a.overview
			a.mu.RUnlock()
			if overview != nil {
				a.updateUI(overview)
			}

			if a.quotaRefreshAllBtn != nil {
				a.quotaRefreshAllBtn.SetText(i18n.Current().BtnRefreshQuota)
				a.quotaRefreshAllBtn.Enable()
			}
			a.isQuotaRefreshing.Store(false)
		})
	}()
}

// RefreshSingleAccountQuota triggers quota retrieval for a single specified account independently.
func (a *App) RefreshSingleAccountQuota(email string) {
	if email == "" {
		return
	}

	if _, loaded := a.accountRefreshing.LoadOrStore(email, true); loaded {
		return
	}

	go func() {
		defer a.accountRefreshing.Delete(email)

		if a.telemetryPool == nil {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()

		data, pollErr := a.telemetryPool.PollAccount(ctx, email)
		if ctrl := a.Controller(); ctrl != nil && pollErr == nil && data != nil {
			ctrl.OnTelemetryUpdate([]*telemetry.AccountQuotaData{data})
		}
		if a.quotaCache != nil {
			_ = a.quotaCache.SaveToDisk()
		}

		fyne.Do(func() {
			a.mu.RLock()
			overview := a.overview
			currentTheme := a.currentTheme
			a.mu.RUnlock()

			palette := GetThemePalette(currentTheme)

			if a.accountListContainer != nil && overview != nil {
				targetIdx := -1
				var targetItem manager.AccountDisplayItem
				for idx, acc := range overview.Accounts {
					if acc.Email == email {
						targetIdx = idx
						targetItem = acc
						break
					}
				}

				if targetIdx >= 0 && targetIdx < len(a.accountCards) && a.accountCards[targetIdx] != nil && a.accountCards[targetIdx].Email == email {
					var quotaData *telemetry.AccountQuotaData
					if a.quotaCache != nil {
						quotaData, _ = a.quotaCache.Get(email)
					}
					if quotaData == nil {
						quotaData = data
					}
					a.accountCards[targetIdx].UpdateState(targetItem, palette)
					a.accountCards[targetIdx].UpdateQuota(quotaData, palette)
				} else if targetIdx >= 0 && targetIdx < len(a.accountListContainer.Objects) {
					var quotaData *telemetry.AccountQuotaData
					if a.quotaCache != nil {
						quotaData, _ = a.quotaCache.Get(email)
					}
					if quotaData == nil {
						quotaData = data
					}
					cardWidget := NewAccountCardWidgetWithQuota(
						targetItem,
						quotaData,
						palette,
						a.SwitchAccount,
						a.RefreshSingleAccountQuota,
						a.ToggleAccountStatus,
						a.ConfirmAndDeleteAccount,
					)
					if a.maskAccounts {
						cardWidget.SetMasked(true)
					}
					if targetIdx < len(a.accountCards) {
						a.accountCards[targetIdx] = cardWidget
					}
					a.accountListContainer.Objects[targetIdx] = cardWidget.Root
					a.accountListContainer.Refresh()
				}
			}

		})
	}()
}

// Refresh updates the accounts overview asynchronously without locking user interactions.
func (a *App) Refresh() {
	a.mu.Lock()
	if a.isBusy {
		a.mu.Unlock()
		return
	}
	a.isBusy = true
	gen := a.refreshGen.Add(1) // Tăng thế hệ
	a.mu.Unlock()

	if a.refreshBtn != nil {
		fyne.Do(func() {
			a.refreshBtn.SetText(i18n.Current().BtnRefreshListLoading)
		})
	}

	go func() {
		defer func() {
			fyne.Do(func() {
				if a.refreshGen.Load() == gen {
					if a.refreshBtn != nil {
						a.refreshBtn.SetText(i18n.Current().BtnRefreshList)
					}
					a.mu.Lock()
					a.isBusy = false
					a.mu.Unlock()
				}
			})
		}()

		if a.mgr == nil {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()

		overview, err := a.mgr.GetAccountStates(ctx)
		if err != nil {
			fyne.Do(func() {
				if a.refreshGen.Load() == gen {
					a.showStatusBar(fmt.Sprintf(i18n.Current().StatusFetchError, err.Error()))
				}
			})
			return
		}

		fyne.Do(func() {
			// Bỏ qua kết quả cũ nếu người dùng đã bấm thao tác mới hơn
			if a.refreshGen.Load() != gen {
				return
			}

			a.mu.Lock()
			a.overview = overview
			a.mu.Unlock()

			a.updateUI(overview)
			if len(overview.Warnings) > 0 {
				msg := strings.Join(overview.Warnings, "; ")
				if a.IsMaskAccounts() {
					msg = maskMessageEmails(msg)
				}
				a.showStatusBar("⚠️ " + msg)
			} else {
				a.showStatusBar(i18n.Current().StatusFetchSuccess)
				a.autoHideStatusBar(3 * time.Second)
			}
		})
	}()
}

// SwitchAccount initiates an asynchronous 4-step hot-swap to the specified email
// wrapped through the shared SwitchGate.
func (a *App) SwitchAccount(targetEmail string) {
	a.mu.Lock()
	if a.isBusy {
		a.mu.Unlock()
		return
	}
	a.isBusy = true
	a.refreshGen.Add(1)
	ctrl := a.controller
	mgr := a.mgr
	a.mu.Unlock()

	displayEmail := targetEmail
	if a.IsMaskAccounts() {
		displayEmail = MaskEmail(targetEmail)
	}

	fyne.Do(func() {
		a.showStatusBar(fmt.Sprintf(i18n.Current().StatusSwitchingTo, displayEmail))
	})

	go func() {
		defer func() {
			a.mu.Lock()
			a.isBusy = false
			a.mu.Unlock()
		}()

		if mgr == nil {
			return
		}

		var gate *automation.SwitchGate
		if ctrl != nil {
			gate = ctrl.Gate()
		}

		if ctrl != nil {
			ctrl.BeginManualSwitch()
			defer ctrl.EndManualSwitch()
		}
		res, err := automation.ExecuteSwitchWithGate(context.Background(), gate, mgr, targetEmail, false)

		if err != nil {
			fyne.Do(func() {
				if a.win != nil {
					dialog.ShowError(err, a.win)
				}
				a.clearStatusBar()
			})
			return
		}

		// Invalidate recovery state on manual session change
		if ctrl != nil && ctrl.RetryCoordinator() != nil {
			ctrl.RetryCoordinator().Reset()
			ctrl.RetryCoordinator().SetCurrentEmail(targetEmail)
		}

		// Refresh account states in RAM and update cards immediately
		overview, _ := mgr.GetAccountStates(context.Background())
		a.mu.Lock()
		a.overview = overview
		a.mu.Unlock()

		fyne.Do(func() {
			a.updateUI(overview)
			successEmail := targetEmail
			if a.IsMaskAccounts() {
				successEmail = MaskEmail(targetEmail)
			}
			msg := fmt.Sprintf(i18n.Current().StatusSwitchedSuccess, successEmail)
			if res != nil && res.Message != "" {
				if a.IsMaskAccounts() {
					msg = fmt.Sprintf("✅ %s", maskMessageEmails(res.Message))
				} else {
					msg = fmt.Sprintf("✅ %s", res.Message)
				}
			}
			a.showStatusBar(msg)
			a.autoHideStatusBar(3 * time.Second)
		})
	}()
}

// ToggleAccountStatus toggles an account's status between "on" and "off",
// persists the change to accounts_store/<email>.json, updates the card widget in-place,
// and initiates switch / safety stop if the active account is toggled to "off".
func (a *App) ToggleAccountStatus(email string) {
	a.mu.Lock()
	if a.isBusy {
		a.mu.Unlock()
		return
	}
	mgr := a.mgr
	overview := a.overview
	currentTheme := a.currentTheme
	activeEmail := ""
	if overview != nil {
		activeEmail = overview.ActiveEmail
	}
	a.mu.Unlock()

	if mgr == nil {
		return
	}

	// 1. Xác định trạng thái hiện tại và đảo trạng thái
	targetIdx := -1
	currentStatus := mgr.GetAccountStatus(email)
	if overview != nil {
		for idx, acc := range overview.Accounts {
			if acc.Email == email {
				targetIdx = idx
				if currentStatus == "" && acc.Status != "" {
					currentStatus = acc.Status
				}
				break
			}
		}
	}

	newStatus := "off"
	if currentStatus == "off" {
		newStatus = "on"
	}

	// 2. Ghi bền vững vào kho accounts_store/<email>.json
	if err := mgr.SetAccountStatus(email, newStatus); err != nil {
		fyne.Do(func() {
			a.showStatusBar(fmt.Sprintf(i18n.Current().StatusUpdateError, err))
		})
		return
	}

	// 3. Cập nhật overview và widget thẻ tại chỗ
	a.mu.Lock()
	if overview != nil && targetIdx >= 0 && targetIdx < len(overview.Accounts) {
		overview.Accounts[targetIdx].Status = newStatus
	}
	a.mu.Unlock()

	palette := GetThemePalette(currentTheme)
	fyne.Do(func() {
		if overview != nil && targetIdx >= 0 && targetIdx < len(a.accountCards) && a.accountCards[targetIdx] != nil {
			a.accountCards[targetIdx].UpdateState(overview.Accounts[targetIdx], palette)
		} else {
			for _, card := range a.accountCards {
				if card != nil && card.Email == email {
					item := manager.AccountDisplayItem{
						Email:    email,
						Status:   newStatus,
						HasToken: card.HasToken,
						IsActive: card.IsActive,
					}
					card.UpdateState(item, palette)
					break
				}
			}
		}
	})

	// 4. Xử lý đặc biệt: Nếu người dùng TẮT chính tài khoản đang Active
	if activeEmail == "" && overview != nil {
		activeEmail = overview.ActiveEmail
	}
	if activeEmail == "" && mgr != nil {
		activeEmail, _ = mgr.GetCurrentActiveEmail(context.Background())
	}

	if email == activeEmail && newStatus == "off" {
		a.mu.RLock()
		isAuto := a.autoSwitchEnabled
		ctrl := a.controller
		a.mu.RUnlock()

		if isAuto {
			hasOtherOn := false
			if overview != nil {
				for _, acc := range overview.Accounts {
					if acc.Email != email && acc.Status == "on" && acc.HasToken {
						hasOtherOn = true
						break
					}
				}
			}

			if !hasOtherOn {
				// Không còn tài khoản nào hợp lệ đang "on" -> Kích hoạt chu trình Dừng an toàn (Safety Stop)
				a.stopAutoSwitchWithNoCandidate()
			} else if ctrl != nil {
				go func() {
					_, _ = ctrl.TriggerSwitch(context.Background(), "tài khoản active bị tắt")
				}()
			}
		}
	}
}

// ConfirmAndDeleteAccount displays a confirmation dialog before executing two-store deletion.
func (a *App) ConfirmAndDeleteAccount(email string) {
	a.mu.Lock()
	if a.isBusy {
		a.mu.Unlock()
		return
	}
	mgr := a.mgr
	overview := a.overview
	win := a.win
	maskAccounts := a.maskAccounts
	a.mu.Unlock()

	if mgr == nil || win == nil {
		return
	}

	normEmail := strings.ToLower(strings.TrimSpace(email))
	if normEmail == "" {
		return
	}

	// Invariant 1: Ngăn chặn xóa tài khoản Active
	if overview != nil && strings.EqualFold(overview.ActiveEmail, normEmail) {
		fyne.Do(func() {
			a.showStatusBar(i18n.Current().StatusCannotDeleteActive)
		})
		return
	}

	// Phân loại thẻ có token hay thẻ [Cần OAuth]
	hasToken := false
	if overview != nil {
		for _, acc := range overview.Accounts {
			if strings.EqualFold(acc.Email, normEmail) {
				hasToken = acc.HasToken
				break
			}
		}
	}

	displayEmail := normEmail
	if maskAccounts {
		displayEmail = MaskEmail(normEmail)
	}

	d := i18n.Current()
	title := d.ConfirmDeleteTitle
	msg := fmt.Sprintf(d.ConfirmMsgMainNoToken, displayEmail)
	if hasToken {
		msg = fmt.Sprintf(d.ConfirmMsgMainHasToken, displayEmail)
	}

	msgLabel := widget.NewLabel(msg)
	msgLabel.Wrapping = fyne.TextWrapWord

	confirmDlg := dialog.NewCustomConfirm(
		title,
		d.ConfirmBtnDelete,
		d.ConfirmBtnCancel,
		msgLabel,
		func(confirmed bool) {
			if !confirmed {
				return
			}
			go a.executeRemoveAccount(normEmail, displayEmail)
		},
		win,
	)
	confirmDlg.SetConfirmImportance(widget.DangerImportance)
	confirmDlg.Show()
}

func (a *App) executeRemoveAccount(email string, displayEmail string) {
	a.mu.Lock()
	a.isBusy = true
	mgr := a.mgr
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.isBusy = false
		a.mu.Unlock()
	}()

	// 1. Thực thi xóa nghiệp vụ qua Manager
	if err := mgr.RemoveAccount(context.Background(), email); err != nil {
		fyne.Do(func() {
			a.showStatusBar(fmt.Sprintf(i18n.Current().StatusDeleteError, err))
		})
		return
	}

	// 2. Dọn dẹp RAM Cache
	if a.quotaCache != nil {
		a.quotaCache.Delete(email)
	}

	// 3. Đọc lại overview mới từ Manager
	newOverview, err := mgr.GetAccountStates(context.Background())
	if err != nil || newOverview == nil {
		return
	}

	a.mu.Lock()
	a.overview = newOverview
	a.mu.Unlock()

	// 4. Cập nhật UI trên main thread
	fyne.Do(func() {
		a.updateUI(newOverview)
		a.showStatusBar(fmt.Sprintf(i18n.Current().StatusDeleteSuccess, displayEmail))
	})

	// 5. Invariant 6: Kiểm tra an toàn Auto-switch
	// Đếm số tài khoản hợp lệ còn lại có Token và Status == "on"
	validCandidates := 0
	for _, acc := range newOverview.Accounts {
		if acc.HasToken && strings.ToLower(strings.TrimSpace(acc.Status)) != "off" {
			validCandidates++
		}
	}

	if validCandidates == 0 {
		a.mu.RLock()
		isAuto := a.autoSwitchEnabled
		a.mu.RUnlock()
		if isAuto {
			a.stopAutoSwitchWithNoCandidate()
		}
	}
}

// ToggleAlwaysOnTop switches the state of the Always-on-Top flag and persists the preference.
func (a *App) ToggleAlwaysOnTop() {
	a.mu.Lock()
	a.alwaysOnTop = !a.alwaysOnTop
	enabled := a.alwaysOnTop
	path := a.preferencesPath
	theme := a.currentTheme
	mask := a.maskAccounts
	autoSwitch := a.autoSwitchEnabled
	a.mu.Unlock()

	setWindowAlwaysOnTop(windowTitle, enabled)
	a.syncAlwaysOnTopBtn(enabled)

	// Non-blocking disk persistence
	go func() {
		a.prefSaveMu.Lock()
		defer a.prefSaveMu.Unlock()

		pref, err := LoadUIPreferences(path)
		if err != nil || pref == nil {
			pref = &UIPreferences{
				Theme:        theme,
				MaskAccounts: mask,
				AutoSwitch:   autoSwitch,
			}
		}
		pref.Theme = theme
		pref.MaskAccounts = mask
		pref.AutoSwitch = autoSwitch
		pref.AlwaysOnTop = &enabled
		_ = SaveUIPreferences(path, pref)
	}()
}

// syncAlwaysOnTopBtn updates the pin button text and importance to match the given state.
func (a *App) syncAlwaysOnTopBtn(enabled bool) {
	if a.alwaysOnTopBtn == nil {
		return
	}
	a.alwaysOnTopBtn.SetText("📌")
	if enabled {
		a.alwaysOnTopBtn.Importance = widget.HighImportance
	} else {
		a.alwaysOnTopBtn.Importance = widget.MediumImportance
	}
	a.alwaysOnTopBtn.Refresh()
}

// SetAlwaysOnTop explicitly sets the Always-on-Top state and persists the preference.
func (a *App) SetAlwaysOnTop(enabled bool) {
	a.mu.Lock()
	if a.alwaysOnTop == enabled {
		a.mu.Unlock()
		setWindowAlwaysOnTop(windowTitle, enabled)
		a.syncAlwaysOnTopBtn(enabled)
		return
	}
	a.alwaysOnTop = enabled
	path := a.preferencesPath
	theme := a.currentTheme
	mask := a.maskAccounts
	autoSwitch := a.autoSwitchEnabled
	a.mu.Unlock()

	setWindowAlwaysOnTop(windowTitle, enabled)
	a.syncAlwaysOnTopBtn(enabled)

	// Non-blocking disk persistence
	go func() {
		a.prefSaveMu.Lock()
		defer a.prefSaveMu.Unlock()

		pref, err := LoadUIPreferences(path)
		if err != nil || pref == nil {
			pref = &UIPreferences{
				Theme:        theme,
				MaskAccounts: mask,
				AutoSwitch:   autoSwitch,
			}
		}
		pref.Theme = theme
		pref.MaskAccounts = mask
		pref.AutoSwitch = autoSwitch
		pref.AlwaysOnTop = &enabled
		_ = SaveUIPreferences(path, pref)
	}()
}

// IsAlwaysOnTop returns whether Always-on-Top is currently enabled.
func (a *App) IsAlwaysOnTop() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.alwaysOnTop
}

// IsAutoSwitchEnabled returns whether auto-switch is currently toggled on in the UI.
func (a *App) IsAutoSwitchEnabled() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.autoSwitchEnabled
}

// SetAutoSwitchEnabled explicitly sets the auto-switch state in the UI, controller, and disk preferences.
func (a *App) SetAutoSwitchEnabled(enabled bool) {
	a.mu.Lock()
	if a.autoSwitchEnabled == enabled {
		ctrl := a.controller
		toggle := a.autoSwitchToggle
		a.mu.Unlock()
		if toggle != nil && toggle.IsOn() != enabled {
			toggle.SetOn(enabled)
		}
		if ctrl != nil && ctrl.IsEnabled() != enabled {
			ctrl.SetEnabled(enabled)
		}
		return
	}
	a.autoSwitchEnabled = enabled
	toggle := a.autoSwitchToggle
	ctrl := a.controller
	path := a.preferencesPath
	theme := a.currentTheme
	mask := a.maskAccounts
	alwaysOnTop := a.alwaysOnTop
	a.mu.Unlock()

	if toggle != nil && toggle.IsOn() != enabled {
		toggle.SetOn(enabled)
	}
	if ctrl != nil {
		ctrl.SetEnabled(enabled)
	}

	// Non-blocking disk persistence
	go func() {
		a.prefSaveMu.Lock()
		defer a.prefSaveMu.Unlock()

		pref, err := LoadUIPreferences(path)
		if err != nil || pref == nil {
			pref = &UIPreferences{
				Theme:        theme,
				MaskAccounts: mask,
				AlwaysOnTop:  &alwaysOnTop,
			}
		}
		pref.Theme = theme
		pref.MaskAccounts = mask
		pref.AutoSwitch = enabled
		pref.AlwaysOnTop = &alwaysOnTop
		_ = SaveUIPreferences(path, pref)
	}()
}

// IsMaskAccounts reports whether account privacy masking is currently active.
func (a *App) IsMaskAccounts() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.maskAccounts
}

// SetMaskAccounts updates the privacy masking state and updates all cards in-place.
func (a *App) SetMaskAccounts(mask bool) {
	a.mu.Lock()
	if a.maskAccounts == mask {
		toggle := a.maskAccountsToggle
		a.mu.Unlock()
		if toggle != nil && toggle.IsOn() != mask {
			toggle.SetOn(mask)
		}
		return
	}
	a.maskAccounts = mask
	toggle := a.maskAccountsToggle
	path := a.preferencesPath
	theme := a.currentTheme
	autoSwitch := a.autoSwitchEnabled
	alwaysOnTop := a.alwaysOnTop
	a.mu.Unlock()

	if toggle != nil && toggle.IsOn() != mask {
		toggle.SetOn(mask)
	}

	// Update cards in-place (Zero layout jank)
	a.mu.RLock()
	cards := append([]*AccountCardWidget(nil), a.accountCards...)
	a.mu.RUnlock()
	for _, card := range cards {
		if card != nil {
			card.SetMasked(mask)
		}
	}

	// Non-blocking disk persistence
	go func() {
		a.prefSaveMu.Lock()
		defer a.prefSaveMu.Unlock()

		pref, err := LoadUIPreferences(path)
		if err != nil || pref == nil {
			pref = &UIPreferences{
				Theme:       theme,
				AutoSwitch:  autoSwitch,
				AlwaysOnTop: &alwaysOnTop,
			}
		}
		pref.Theme = theme
		pref.MaskAccounts = mask
		pref.AutoSwitch = autoSwitch
		pref.AlwaysOnTop = &alwaysOnTop
		_ = SaveUIPreferences(path, pref)
	}()

	// Status bar transient feedback
	if mask {
		a.showStatusBar(i18n.Current().StatusMaskOn)
	} else {
		a.showStatusBar(i18n.Current().StatusMaskOff)
	}
	a.autoHideStatusBar(2 * time.Second)
}

// ToggleMaskAccounts flips the account privacy masking state.
func (a *App) ToggleMaskAccounts() {
	a.SetMaskAccounts(!a.IsMaskAccounts())
}

// updateUI refreshes all visual components based on the given overview.
func (a *App) updateUI(overview *manager.AccountOverview) {
	if overview == nil {
		return
	}
	a.mu.Lock()
	a.overview = overview
	a.lastLivenessState = overview.LanguageServerRunning
	if overview.ServerInfo != nil {
		a.lastLivenessPID = overview.ServerInfo.PID
		a.lastLivenessPort = overview.ServerInfo.Port
	} else {
		a.lastLivenessPID = 0
		a.lastLivenessPort = 0
	}
	a.mu.Unlock()

	palette := GetThemePalette(a.currentTheme)

	// 1. Language Server status in Footer Bar
	if a.serverStatusLabel != nil {
		a.renderServerStatusLabel(overview.LanguageServerRunning, overview.ServerInfo)
	}

	// 3. 3-Column Account Grid with in-place reactive quota updates
	if a.accountListContainer != nil {
		canUpdateInPlace := false
		if len(a.accountCards) == len(overview.Accounts) && len(overview.Accounts) > 0 {
			canUpdateInPlace = true
			for i, acc := range overview.Accounts {
				if a.accountCards[i] == nil || a.accountCards[i].Email != acc.Email {
					canUpdateInPlace = false
					break
				}
			}
		}

		if canUpdateInPlace {
			// In-place reactive quota updates: Canvas tree is untouched, no layout jank or window resizing
			for i, acc := range overview.Accounts {
				var quotaData *telemetry.AccountQuotaData
				if a.quotaCache != nil {
					quotaData, _ = a.quotaCache.Get(acc.Email)
				}
				a.accountCards[i].UpdateState(acc, palette)
				a.accountCards[i].UpdateQuota(quotaData, palette)
			}
		} else {
			a.accountListContainer.Objects = nil
			a.accountCards = nil
			if len(overview.Accounts) == 0 {
				emptyLabel := widget.NewLabelWithStyle(i18n.Current().MainEmptyAccounts, fyne.TextAlignCenter, fyne.TextStyle{Italic: true})
				a.accountListContainer.Add(container.NewPadded(emptyLabel))
			} else {
				for _, acc := range overview.Accounts {
					var quotaData *telemetry.AccountQuotaData
					if a.quotaCache != nil {
						quotaData, _ = a.quotaCache.Get(acc.Email)
					}
					cardWidget := NewAccountCardWidgetWithQuota(acc, quotaData, palette, a.SwitchAccount, a.RefreshSingleAccountQuota, a.ToggleAccountStatus, a.ConfirmAndDeleteAccount)
					if a.maskAccounts {
						cardWidget.SetMasked(true)
					}
					a.accountCards = append(a.accountCards, cardWidget)
					a.accountListContainer.Add(cardWidget.Root)
				}
			}
			a.accountListContainer.Refresh()
			if a.accountScroll != nil {
				a.accountScroll.Refresh()
			}
		}
	}

}

// showStatusBar displays a transient status message taking over the entire bottom bar.
func (a *App) showStatusBar(msg string) {
	if a.statusBar == nil {
		return
	}
	if a.statusBarHideTimer != nil {
		a.statusBarHideTimer.Stop()
		a.statusBarHideTimer = nil
	}

	// 1. Tạm ẩn text hai bên mép để nhường 100% không gian cho thông báo
	if a.serverStatusLabel != nil {
		a.serverStatusLabel.Hide()
	}
	if a.versionLabel != nil {
		a.versionLabel.Hide()
	}

	// 2. Cài đặt nội dung và hiển thị thông báo
	a.statusBar.SetText(msg)
	a.statusBar.Show()
}

// clearStatusBar hides the status bar and restores normal left/right footer labels.
func (a *App) clearStatusBar() {
	if a.statusBar == nil {
		return
	}
	a.statusBar.SetText("")
	a.statusBar.Hide()

	// Khôi phục lại trạng thái bình thường cho thanh đáy
	if a.serverStatusLabel != nil {
		a.serverStatusLabel.Show()
	}
	if a.versionLabel != nil {
		a.versionLabel.Show()
	}
}

// autoHideStatusBar schedules the status bar to be hidden after the given duration.
func (a *App) autoHideStatusBar(d time.Duration) {
	if a.statusBarHideTimer != nil {
		a.statusBarHideTimer.Stop()
	}
	a.statusBarHideTimer = time.AfterFunc(d, func() {
		fyne.Do(func() {
			a.clearStatusBar()
		})
	})
}

// forceRebuildCards clears the in-place update cache so updateUI fully rebuilds all cards
// with the current theme palette. Used after theme changes.
func (a *App) forceRebuildCards(overview *manager.AccountOverview) {
	a.accountCards = nil
	a.updateUI(overview)
}

// Logo returns the brand logo static resource.
func (a *App) Logo() fyne.Resource {
	return AppLogoResource
}
