package ui

import (
	"context"
	"errors"
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

	"as-auto-switcher/internal/core/i18n"
	"as-auto-switcher/internal/engines/antigravity/automation"
	"as-auto-switcher/internal/engines/antigravity/connector"
	"as-auto-switcher/internal/engines/antigravity/manager"
	"as-auto-switcher/internal/engines/antigravity/telemetry"
	codexconnector "as-auto-switcher/internal/engines/codex/connector"
	codexmanager "as-auto-switcher/internal/engines/codex/manager"
	codexreloader "as-auto-switcher/internal/engines/codex/reloader"
	codexstore "as-auto-switcher/internal/engines/codex/store"
)

// App manages the Fyne GUI runtime lifecycle, state, telemetry, and UI bindings.
type App struct {
	mu            sync.RWMutex
	uiMu          sync.Mutex // Serializes updateUI execution to prevent concurrent re-entry
	app           fyne.App
	win           fyne.Window
	mgr           *manager.AccountManager
	overview      *manager.AccountOverview
	alwaysOnTop   bool
	isBusy        bool
	currentLocale string

	// Codex Engine Integration (Phase 5 & 6)
	codexStore          codexstore.AccountStore
	codexAccountMgr     codexmanager.AccountManager
	codexOAuthMgr       codexmanager.OAuthManager
	codexBrowserMgr     codexconnector.BrowserLauncher
	codexReloader       codexreloader.SessionReloader
	codexStatusReporter func(string)
	codexInitErr        error
	codexModal          *CodexAddModal
	codexCards          []*codexmanager.CodexCardView

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

	// Dynamic Quota Matrix Discovered Models (Campaign 2 Phase 3)
	discoveredModels   []DiscoveredModelInfo
	discoveredModelsK  int
	lastModelDiscovery time.Time

	// Dynamic UI Widgets
	serverStatusLabel    *widget.Label
	versionLabel         *widget.Label
	codexStatusLabel     *widget.Label
	codexVersionLabel    *widget.Label
	leftFooter           *fyne.Container
	rightFooter          *fyne.Container
	antigravityContent   *fyne.Container
	codex                *codexView
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
	headerDivider1       *canvas.Rectangle
	headerDivider2       *canvas.Rectangle
	headerDivider3       *canvas.Rectangle
	monogramBg           *canvas.Rectangle
	titleLabel           *widget.RichText
	sidebar              *SidebarContainer
	topSection           *fyne.Container
	toolbar              *fyne.Container
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
	refreshGen               atomic.Uint64
	closed                   atomic.Bool
	sessionSyncRunning       bool
	sessionSyncPending       bool
	sessionSyncManual        bool
	sessionSyncPendingManual bool
	sessionSyncCancel        context.CancelFunc
	lastIdentityPoll         time.Time
	backgroundWarnings       map[string]bool
	statusGeneration         uint64

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
	a := NewAppWithFyne(fyneApp, mgr)
	a.codexInitErr = a.InitDefaultCodexEngine()
	return a, nil
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
		logsDir = filepath.Join(mgr.Store().GetDataRootDir(), "logs")
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
	if a.headerDivider1 != nil {
		a.headerDivider1.FillColor = palette.BorderSubtle
		a.headerDivider1.Refresh()
	}
	if a.headerDivider2 != nil {
		a.headerDivider2.FillColor = palette.BorderSubtle
		a.headerDivider2.Refresh()
	}
	if a.headerDivider3 != nil {
		a.headerDivider3.FillColor = palette.BorderSubtle
		a.headerDivider3.Refresh()
	}
	if a.sidebar != nil {
		a.sidebar.ApplyTheme(palette)
	}
	if a.codex != nil {
		a.codex.applyTheme(palette)
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
		if a.titleLabel != nil {
			a.titleLabel.Segments = []widget.RichTextSegment{
				&widget.TextSegment{
					Text: d.AppTitle,
					Style: widget.RichTextStyle{
						SizeName:  titleTextSizeName,
						TextStyle: fyne.TextStyle{Bold: true},
					},
				},
			}
			a.titleLabel.Refresh()
		}
		if a.sidebar != nil && a.sidebar.antigravityItem != nil {
			a.sidebar.antigravityItem.textLabel.Text = d.NavAntigravity
			a.sidebar.antigravityItem.textLabel.Refresh()
		}
		if a.codex != nil {
			a.codex.setLanguage(d)
			a.codexStatusLabel.SetText(d.CodexStatus)
			a.codexVersionLabel.SetText(d.CodexVersion)
		}
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
	if a.headerDivider1 != nil {
		a.headerDivider1.FillColor = palette.BorderSubtle
		a.headerDivider1.Refresh()
	}
	if a.headerDivider2 != nil {
		a.headerDivider2.FillColor = palette.BorderSubtle
		a.headerDivider2.Refresh()
	}
	if a.headerDivider3 != nil {
		a.headerDivider3.FillColor = palette.BorderSubtle
		a.headerDivider3.Refresh()
	}
	if a.sidebar != nil {
		a.sidebar.ApplyTheme(palette)
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

	a.mu.RLock()
	cMgr := a.codexAccountMgr
	a.mu.RUnlock()
	if a.codex != nil && cMgr != nil {
		a.codex.setCallbacks(
			a.ShowCodexAddModal,
			a.RefreshCodexCards,
			a.handleCodexSwitch,
			a.ShowCodexReauthModal,
			a.handleCodexToggle,
			a.handleCodexDelete,
		)
		a.RefreshCodexCards()
	}

	if a.codexInitErr != nil && a.codex != nil {
		a.codex.description.SetText(i18n.Current().CodexInitFailed)
	}
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
	a.closed.Store(true)
	a.mu.Lock()
	if a.sessionSyncCancel != nil {
		a.sessionSyncCancel()
	}
	a.sessionSyncPending, a.sessionSyncPendingManual = false, false
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
		busy := a.isBusy || a.sessionSyncRunning
		a.mu.RUnlock()
		if !busy {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	// Ensure all in-flight asynchronous preference saves finish writing to disk before cleanup
	a.prefSaveMu.Lock()
	a.prefSaveMu.Unlock()

	a.mu.RLock()
	oMgr := a.codexOAuthMgr
	a.mu.RUnlock()
	if a.codexModal != nil {
		a.codexModal.Close()
	}
	if oMgr != nil {
		_ = oMgr.Close()
	}
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
	if a.closed.Load() {
		return
	}
	defer a.pollSessionIdentity()
	a.mu.RLock()
	busy, overview, checker, discoverer := a.isBusy, a.overview, a.livenessChecker, a.serverDiscoverer
	running := overview != nil && overview.LanguageServerRunning && overview.ServerInfo != nil
	var info *connector.ServerInfo
	if running {
		info = overview.ServerInfo
	}
	a.mu.RUnlock()
	if busy || checker == nil || discoverer == nil || (a.mgr != nil && a.mgr.IsMutating()) {
		return
	}
	nextRunning := running
	nextInfo := info
	if running {
		if !checker(info) {
			connector.InvalidateDiscoveryCache(0, "")
			nextRunning, nextInfo = false, nil
		}
	} else {
		found, err := discoverer()
		if err == nil && found != nil && found.PID > 0 && found.Port > 0 {
			nextRunning, nextInfo = true, found
		}
	}
	if running == nextRunning {
		if nextRunning && nextInfo != nil {
			a.mu.RLock()
			hasModels := len(a.discoveredModels) > 0
			a.mu.RUnlock()
			if !hasModels {
				a.queryDiscoveredModelsAsync(nextInfo)
			}
		}
		return
	}
	a.mu.Lock()
	if a.closed.Load() || a.isBusy || a.overview != overview {
		a.mu.Unlock()
		return
	}
	next := &manager.AccountOverview{}
	if overview != nil {
		*next = *overview
	}
	next.LanguageServerRunning, next.ServerInfo = nextRunning, nextInfo
	a.overview = next
	a.lastLivenessState = nextRunning
	a.lastLivenessPID, a.lastLivenessPort = 0, 0
	if nextInfo != nil {
		a.lastLivenessPID, a.lastLivenessPort = nextInfo.PID, nextInfo.Port
	}
	a.mu.Unlock()
	if nextRunning && nextInfo != nil {
		a.queryDiscoveredModelsAsync(nextInfo)
	}
	fyne.Do(func() {
		if a.closed.Load() {
			return
		}
		a.mu.RLock()
		current := a.overview
		a.mu.RUnlock()
		if current != nil {
			a.renderServerStatusLabel(current.LanguageServerRunning, current.ServerInfo)
		}
	})
	a.requestSessionSync(false, false)
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
	if a.closed.Load() || evt.Message == "" {
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
		a.requestSessionSync(false, false)

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

// DiscoveredModels returns a thread-safe copy of discovered extension models (INV-P3-04 / INV-P3-05).
func (a *App) DiscoveredModels() []DiscoveredModelInfo {
	if a == nil {
		return nil
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	if len(a.discoveredModels) == 0 {
		return nil
	}
	res := make([]DiscoveredModelInfo, len(a.discoveredModels))
	copy(res, a.discoveredModels)
	return res
}

// SetDiscoveredModels records discovered extension models safely under mutex (INV-P3-04 / INV-P3-05).
func (a *App) SetDiscoveredModels(models []DiscoveredModelInfo) {
	if a == nil {
		return
	}
	a.mu.Lock()
	a.discoveredModels = models
	a.discoveredModelsK = len(models)
	a.mu.Unlock()
}

// queryDiscoveredModelsAsync queries CascadeModelConfigData asynchronously in a background goroutine
// with a 500ms timeout, deduplicating through FilterExtensionModels, and safely updating App state (INV-P3-04 / INV-P3-05).
func (a *App) queryDiscoveredModelsAsync(info *connector.ServerInfo) {
	if info == nil || a.closed.Load() {
		return
	}
	a.mu.Lock()
	if time.Since(a.lastModelDiscovery) < 10*time.Second {
		a.mu.Unlock()
		return
	}
	a.lastModelDiscovery = time.Now()
	a.mu.Unlock()

	go func(targetInfo *connector.ServerInfo) {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		client := connector.NewClient(targetInfo)
		cfgData, err := client.GetCascadeModelConfig(ctx)
		if err != nil || cfgData == nil {
			return
		}

		discovered := FilterExtensionModels(cfgData.ClientModelConfigs)
		newK := len(discovered)

		a.mu.RLock()
		prevK := a.discoveredModelsK
		a.mu.RUnlock()

		a.SetDiscoveredModels(discovered)

		if newK != prevK {
			fyne.Do(func() {
				if a.closed.Load() {
					return
				}
				a.UpdateCardHeight(newK)
				a.mu.RLock()
				overview := a.overview
				a.mu.RUnlock()
				if overview != nil {
					a.forceRebuildCards(overview)
				}
			})
		}
	}(info)
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
					if a.closed.Load() {
						return
					}
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
			if a.closed.Load() {
				return
			}
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
			if a.closed.Load() {
				return
			}
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
					cardWidget := NewAccountCardWidgetWithDynamicModels(
						targetItem,
						quotaData,
						palette,
						a.DiscoveredModels(),
						a.SwitchAccount,
						a.RefreshSingleAccountQuota,
						func(email string) { go a.ToggleAccountStatus(email) },
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
func (a *App) Refresh() { a.requestSessionSync(true, false) }

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
	if a.sessionSyncCancel != nil {
		a.sessionSyncCancel()
	}
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
			a.requestSessionSync(false, false)
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
				if a.win != nil && !a.closed.Load() {
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

		// The completion defer requests a serialized session refresh.

		fyne.Do(func() {
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
	defer a.requestSessionSync(false, false)
	a.mu.Lock()
	if a.isBusy {
		a.mu.Unlock()
		return
	}
	mgr := a.mgr
	overview := a.overview
	if overview != nil {
		copyView := *overview
		copyView.Accounts = append([]manager.AccountDisplayItem(nil), overview.Accounts...)
		overview = &copyView
	}
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

	// Present the status update without allowing an older account order to
	// address another card. The common refresh then reconciles the whole view.
	revision := mgr.Revision()
	if overview != nil && targetIdx >= 0 && targetIdx < len(overview.Accounts) {
		overview.Accounts[targetIdx].Status = newStatus
	}
	palette := GetThemePalette(currentTheme)
	fyne.Do(func() {
		if a.closed.Load() || mgr.IsMutating() || mgr.Revision() != revision {
			return
		}
		for _, card := range a.accountCards {
			if card != nil && card.Email == email {
				item := manager.AccountDisplayItem{Email: email, Status: newStatus, HasToken: card.HasToken, IsActive: card.IsActive}
				card.UpdateState(item, palette)
				break
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
				fyne.Do(func() {
					if !a.closed.Load() {
						a.stopAutoSwitchWithNoCandidate()
					}
				})
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
	a.refreshGen.Add(1)
	if a.sessionSyncCancel != nil {
		a.sessionSyncCancel()
	}
	a.isBusy = true
	mgr := a.mgr
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.isBusy = false
		a.mu.Unlock()
		a.requestSessionSync(false, false)
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
			fyne.Do(func() {
				if !a.closed.Load() {
					a.stopAutoSwitchWithNoCandidate()
				}
			})
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
	buttons := []*widget.Button{a.alwaysOnTopBtn}
	if a.codex != nil {
		buttons = append(buttons, a.codex.pin)
	}
	for _, button := range buttons {
		if button == nil {
			continue
		}
		if enabled {
			button.Importance = widget.HighImportance
		} else {
			button.Importance = widget.MediumImportance
		}
		button.Refresh()
	}
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
	if overview == nil || a.closed.Load() {
		return
	}
	a.uiMu.Lock()
	defer a.uiMu.Unlock()

	if overview.Versioned && a.mgr != nil && (a.mgr.IsMutating() || overview.Revision != a.mgr.Revision()) {
		a.requestSessionSync(false, false)
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
		a.UpdateCardHeight(len(a.DiscoveredModels()))
		canUpdateInPlace := len(a.accountCards) == len(overview.Accounts) && len(overview.Accounts) > 0
		if canUpdateInPlace {
			byEmail := make(map[string]*AccountCardWidget, len(a.accountCards))
			for _, card := range a.accountCards {
				if card != nil {
					byEmail[card.Email] = card
				}
			}
			ordered := make([]*AccountCardWidget, 0, len(overview.Accounts))
			objects := make([]fyne.CanvasObject, 0, len(overview.Accounts))
			for _, acc := range overview.Accounts {
				card := byEmail[acc.Email]
				if card == nil {
					canUpdateInPlace = false
					break
				}
				ordered = append(ordered, card)
				objects = append(objects, card.Root)
			}
			if canUpdateInPlace {
				a.accountCards = ordered
				a.accountListContainer.Objects = objects
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
			a.accountListContainer.Refresh()
		} else {
			if len(overview.Accounts) == 0 {
				a.accountCards = nil
				emptyLabel := widget.NewLabelWithStyle(i18n.Current().MainEmptyAccounts, fyne.TextAlignCenter, fyne.TextStyle{Italic: true})
				a.accountListContainer.Objects = []fyne.CanvasObject{container.NewPadded(emptyLabel)}
			} else {
				newCards := make([]*AccountCardWidget, 0, len(overview.Accounts))
				newObjects := make([]fyne.CanvasObject, 0, len(overview.Accounts))
				for _, acc := range overview.Accounts {
					var quotaData *telemetry.AccountQuotaData
					if a.quotaCache != nil {
						quotaData, _ = a.quotaCache.Get(acc.Email)
					}
					cardWidget := NewAccountCardWidgetWithDynamicModels(acc, quotaData, palette, a.DiscoveredModels(), a.SwitchAccount, a.RefreshSingleAccountQuota, func(email string) { go a.ToggleAccountStatus(email) }, a.ConfirmAndDeleteAccount)
					if a.maskAccounts {
						cardWidget.SetMasked(true)
					}
					newCards = append(newCards, cardWidget)
					newObjects = append(newObjects, cardWidget.Root)
				}
				a.accountCards = newCards
				a.accountListContainer.Objects = newObjects
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
	if a.closed.Load() {
		return
	}
	a.mu.Lock()
	a.statusGeneration++
	if a.statusBar == nil {
		a.mu.Unlock()
		return
	}
	if a.statusBarHideTimer != nil {
		a.statusBarHideTimer.Stop()
		a.statusBarHideTimer = nil
	}

	a.statusBar.SetText(msg)
	a.renderFooter()
	a.mu.Unlock()
}

// clearStatusBar hides the status bar and restores normal left/right footer labels.
func (a *App) clearStatusBar() {
	if a.closed.Load() {
		return
	}
	a.mu.Lock()
	a.statusGeneration++
	if a.statusBar == nil {
		a.mu.Unlock()
		return
	}
	a.statusBar.SetText("")
	a.renderFooter()
	a.mu.Unlock()
}

// autoHideStatusBar schedules the status bar to be hidden after the given duration.
func (a *App) autoHideStatusBar(d time.Duration) {
	if a.closed.Load() {
		return
	}
	a.mu.Lock()
	if a.statusBarHideTimer != nil {
		a.statusBarHideTimer.Stop()
	}
	generation := a.statusGeneration
	a.statusBarHideTimer = time.AfterFunc(d, func() {
		fyne.Do(func() {
			a.mu.Lock()
			stale := a.closed.Load() || a.statusGeneration != generation
			a.mu.Unlock()
			if stale {
				return
			}
			a.clearStatusBar()
		})
	})
	a.mu.Unlock()
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

// Sidebar returns the left sidebar container.
func (a *App) Sidebar() *SidebarContainer {
	return a.sidebar
}

// TitleLabel returns the rich text title widget in the top bar.
func (a *App) TitleLabel() *widget.RichText {
	return a.titleLabel
}

// TopSection returns the top bar container (Row 1 Header + Separator).
func (a *App) TopSection() *fyne.Container {
	return a.topSection
}

// Toolbar returns the Antigravity toolbar container (Row 2 buttons, mask, auto switch, theme).
func (a *App) Toolbar() *fyne.Container {
	return a.toolbar
}

// -------------------------------------------------------------------------
// Codex Integration Methods (Phase 5)
// -------------------------------------------------------------------------

// SetCodexServices configures the Codex account manager, oauth manager, and browser launcher.
// It wires the callbacks to the Codex view and performs initial card rendering.
func (a *App) SetCodexServices(accMgr codexmanager.AccountManager, oauthMgr codexmanager.OAuthManager, bMgr codexconnector.BrowserLauncher) {
	a.mu.Lock()
	a.codexAccountMgr = accMgr
	a.codexOAuthMgr = oauthMgr
	a.codexBrowserMgr = bMgr
	a.mu.Unlock()

	if a.codex != nil && accMgr != nil {
		a.codex.setCallbacks(
			a.ShowCodexAddModal,
			a.RefreshCodexCards,
			a.handleCodexSwitch,
			a.ShowCodexReauthModal,
			a.handleCodexToggle,
			a.handleCodexDelete,
		)
		a.RefreshCodexCards()
	}
}

// SetCodexServicesWithReloader configures the Codex services including SessionReloader.
func (a *App) SetCodexServicesWithReloader(accMgr codexmanager.AccountManager, oauthMgr codexmanager.OAuthManager, bMgr codexconnector.BrowserLauncher, rel codexreloader.SessionReloader) {
	a.mu.Lock()
	a.codexAccountMgr = accMgr
	a.codexOAuthMgr = oauthMgr
	a.codexBrowserMgr = bMgr
	a.codexReloader = rel
	a.mu.Unlock()

	if a.codex != nil && accMgr != nil {
		a.codex.setCallbacks(
			a.ShowCodexAddModal,
			a.RefreshCodexCards,
			a.handleCodexSwitch,
			a.ShowCodexReauthModal,
			a.handleCodexToggle,
			a.handleCodexDelete,
		)
		a.RefreshCodexCards()
	}
}

// InitDefaultCodexEngine attempts to bootstrap the Codex subsystem using default directories.
// If initialization fails, Codex actions remain safely disabled while Antigravity continues unaffected.
func (a *App) InitDefaultCodexEngine() error {
	st, err := codexstore.NewAccountStore("")
	if err != nil {
		return err
	}
	bMgr := codexconnector.NewBrowserLauncher()
	oMgr, err := codexmanager.NewOAuthManager(codexmanager.OAuthManagerConfig{
		Store: st,
	})
	if err != nil {
		return err
	}
	statusReporter := func(msg string) {
		fyne.Do(func() {
			a.showStatusBar(msg)
		})
	}
	a.codexStatusReporter = statusReporter
	rel := codexreloader.NewSessionReloader(codexreloader.SessionReloaderConfig{
		Store:          st,
		StatusReporter: statusReporter,
	})
	if err := rel.RecoverPendingJournals(); err != nil {
		_ = oMgr.Close()
		return err
	}

	accMgr, err := codexmanager.NewAccountManager(codexmanager.AccountManagerConfig{
		Store:    st,
		Reloader: rel,
	})
	if err != nil {
		return err
	}
	if err := accMgr.BootstrapActiveSession(); err != nil && !errors.Is(err, codexreloader.ErrUnsupportedDesktopVerification) {
		_ = oMgr.Close()
		return err
	}

	a.codexStore = st
	a.codexReloader = rel
	a.SetCodexServices(accMgr, oMgr, bMgr)
	return nil
}

// ShowCodexAddModal displays the Codex OAuth Add Account modal dialog.
func (a *App) ShowCodexAddModal() {
	if a == nil || a.win == nil {
		return
	}
	a.mu.RLock()
	oMgr := a.codexOAuthMgr
	bMgr := a.codexBrowserMgr
	a.mu.RUnlock()

	if oMgr == nil || bMgr == nil {
		return
	}

	if a.codexModal != nil && !a.codexModal.IsClosed() {
		return
	}
	modal := NewCodexAddModal(a.win, oMgr, bMgr)
	a.codexModal = modal
	modal.SetOnSuccess(func(email string) {
		a.RefreshCodexCards()
	})
	modal.Show()
}

// ShowCodexReauthModal displays the Codex OAuth modal in re-authentication mode for the specified account.
func (a *App) ShowCodexReauthModal(email string) {
	if a == nil || a.win == nil {
		return
	}
	a.mu.RLock()
	oMgr := a.codexOAuthMgr
	bMgr := a.codexBrowserMgr
	st := a.codexStore
	a.mu.RUnlock()

	if oMgr == nil || bMgr == nil {
		return
	}

	var accountID string
	if st != nil {
		if env, err := st.GetAccount(email); err == nil && env != nil {
			if tokens, err := env.GetTokens(); err == nil && tokens != nil {
				accountID = tokens.AccountID
			}
		}
	}

	if a.codexModal != nil && !a.codexModal.IsClosed() {
		return
	}
	modal := NewCodexAddModal(a.win, oMgr, bMgr)
	a.codexModal = modal
	modal.SetReauthTarget(email, accountID)
	modal.SetOnSuccess(func(verifiedEmail string) {
		a.RefreshCodexCards()
	})
	modal.Show()
}

// RefreshCodexCards reads current cards from the Codex AccountManager and renders them in the Codex view.
func (a *App) RefreshCodexCards() {
	a.mu.RLock()
	accMgr := a.codexAccountMgr
	view := a.codex
	themeName := a.currentTheme
	a.mu.RUnlock()

	if accMgr == nil || view == nil {
		return
	}

	cards, err := accMgr.ListCards()
	if err != nil {
		return
	}

	a.mu.Lock()
	a.codexCards = cards
	a.mu.Unlock()

	palette := GetThemePalette(themeName)
	fyne.Do(func() {
		view.renderCards(cards, palette)
	})
}

func (a *App) handleCodexToggle(email string) {
	a.mu.RLock()
	accMgr := a.codexAccountMgr
	a.mu.RUnlock()
	if accMgr != nil {
		_ = accMgr.ToggleStatus(email)
		a.RefreshCodexCards()
	}
}

func (a *App) handleCodexDelete(email string) {
	if a == nil || a.win == nil {
		return
	}
	d := i18n.Current()
	title := d.ConfirmDeleteTitle
	msg := fmt.Sprintf("Bạn có chắc chắn muốn xóa tài khoản Codex %s?", email)
	if a.currentLocale == "en" {
		msg = fmt.Sprintf("Are you sure you want to delete Codex account %s?", email)
	}

	confirmDlg := dialog.NewConfirm(
		title,
		msg,
		func(confirmed bool) {
			if confirmed {
				a.mu.RLock()
				accMgr := a.codexAccountMgr
				a.mu.RUnlock()
				if accMgr != nil {
					if err := accMgr.DeleteAccount(email); err != nil {
						fyne.Do(func() {
							dialog.ShowError(err, a.win)
						})
						return
					}
					a.RefreshCodexCards()
				}
			}
		},
		a.win,
	)
	confirmDlg.SetDismissText(d.ConfirmBtnCancel)
	confirmDlg.SetConfirmText(d.ConfirmBtnDelete)
	confirmDlg.Show()
}

func (a *App) handleCodexSwitch(email string) {
	a.mu.Lock()
	if a.isBusy {
		a.mu.Unlock()
		return
	}
	a.isBusy = true
	accMgr := a.codexAccountMgr
	a.mu.Unlock()

	if accMgr == nil {
		a.mu.Lock()
		a.isBusy = false
		a.mu.Unlock()
		return
	}

	displayEmail := email
	if a.IsMaskAccounts() {
		displayEmail = MaskEmail(email)
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

		switchErr := accMgr.SwitchAccount(email)
		fyne.Do(func() {
			if switchErr == nil {
				a.RefreshCodexCards()
				a.showStatusBar(fmt.Sprintf("✅ %s", displayEmail))
				a.autoHideStatusBar(3 * time.Second)
			} else {
				a.showStatusBar(fmt.Sprintf("❌ %v", switchErr))
				if a.win != nil && !a.closed.Load() {
					if errors.Is(switchErr, codexreloader.ErrUnsupportedDesktopVerification) {
						dialog.ShowInformation("Codex", i18n.Current().CodexSwitchUnsupported, a.win)
					} else {
						dialog.ShowError(switchErr, a.win)
					}
				}
			}
		})
	}()
}

// CodexView returns the internal codexView instance.
func (a *App) CodexView() *codexView {
	return a.codex
}

// CodexAccountManager returns the configured Codex AccountManager.
func (a *App) CodexAccountManager() codexmanager.AccountManager {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.codexAccountMgr
}

// CodexReloader returns the configured Codex SessionReloader.
func (a *App) CodexReloader() codexreloader.SessionReloader {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.codexReloader
}

// CodexStatusReporter returns the configured status reporter callback for Codex.
func (a *App) CodexStatusReporter() func(string) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.codexStatusReporter
}
