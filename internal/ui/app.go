package ui

import (
	"antigravity-switch/internal/automation"
	"context"
	"fmt"
	"path/filepath"
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

	"antigravity-switch/internal/connector"
	"antigravity-switch/internal/manager"
	"antigravity-switch/internal/telemetry"
)

// App manages the Fyne GUI runtime lifecycle, state, telemetry, and UI bindings.
type App struct {
	mu          sync.RWMutex
	app         fyne.App
	win         fyne.Window
	mgr         *manager.AccountManager
	overview    *manager.AccountOverview
	alwaysOnTop bool
	isBusy      bool

	// Telemetry & Themes (Phase 6A)
	quotaCache        *telemetry.QuotaCache
	telemetryPool     *telemetry.TelemetryPool
	autoLoop          *telemetry.AutoTelemetryLoop
	currentTheme      string
	preferencesPath   string
	lastQuotaRefresh  time.Time
	isQuotaRefreshing atomic.Bool
	accountRefreshing sync.Map

	// Dynamic UI Widgets
	serverStatusLabel    *widget.Label
	accountListContainer *fyne.Container
	refreshBtn           *widget.Button
	quotaRefreshAllBtn   *widget.Button
	alwaysOnTopBtn       *widget.Button
	themeSelect          *widget.Select
	autoSwitchToggle     *AutoSwitchToggle
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
		app:             fyneApp,
		mgr:             mgr,
		alwaysOnTop:     true,
		preferencesPath: DefaultUIPreferencesPath,
		currentTheme:    ThemeDark,
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

	// 2. Load UI Preferences (Theme & Masking)
	if prefs, err := LoadUIPreferences(a.preferencesPath); err == nil && prefs != nil {
		if prefs.Theme != "" {
			a.currentTheme = prefs.Theme
		}
		a.maskAccounts = prefs.MaskAccounts
	}

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
	a.mu.Unlock()

	prefs, err := LoadUIPreferences(path)
	if err != nil || prefs == nil {
		prefs = &UIPreferences{Theme: themeName, MaskAccounts: mask}
	}
	prefs.Theme = themeName
	_ = SaveUIPreferences(path, prefs)

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

	if a.alwaysOnTop {
		setWindowAlwaysOnTop(windowTitle, true)
	}

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

	a.Refresh()
}

// Close cleans up background loops and resources.
func (a *App) Close() {
	a.mu.Lock()
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
		a.showStatusBar(fmt.Sprintf("⚠️ %s", msg))
		a.autoHideStatusBar(4 * time.Second)

	case automation.StatusError:
		a.showStatusBar(fmt.Sprintf("❌ %s", msg))
		a.autoHideStatusBar(5 * time.Second)

	case automation.StatusIdle:
		a.showStatusBar(msg)
		a.autoHideStatusBar(3 * time.Second)

	default:
		a.showStatusBar(msg)
		a.autoHideStatusBar(3 * time.Second)
	}
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

	if a.alwaysOnTop {
		setWindowAlwaysOnTop(windowTitle, true)
	}

	// Chạy vòng lặp nhận sự kiện của Fyne
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
		a.quotaRefreshAllBtn.SetText("⏳ Đang quét...")
		a.quotaRefreshAllBtn.Disable()
	}

	go func() {
		completed := false
		defer func() {
			if !completed {
				fyne.Do(func() {
					if a.quotaRefreshAllBtn != nil {
						a.quotaRefreshAllBtn.SetText("Làm mới Quota")
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
				a.quotaRefreshAllBtn.SetText("Làm mới Quota")
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

// Refresh asynchronously updates account states from Chrome and AccountStore.
func (a *App) Refresh() {
	a.mu.Lock()
	if a.isBusy {
		a.mu.Unlock()
		return
	}
	a.isBusy = true
	a.mu.Unlock()

	if a.refreshBtn != nil {
		fyne.Do(func() {
			a.refreshBtn.SetText("⏳ Đang quét...")
		})
	}

	go func() {
		defer func() {
			fyne.Do(func() {
				if a.refreshBtn != nil {
					a.refreshBtn.SetText("Danh sách tài khoản")
				}
				a.mu.Lock()
				a.isBusy = false
				a.mu.Unlock()
			})
		}()

		if a.mgr == nil {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()

		overview, err := a.mgr.GetAccountStates(ctx)
		if err != nil {
			fyne.Do(func() { a.showStatusBar("Không lấy được danh sách tài khoản: " + err.Error()) })
			return
		}

		a.mu.Lock()
		a.overview = overview
		a.mu.Unlock()

		fyne.Do(func() {
			a.updateUI(overview)
			if len(overview.Warnings) > 0 {
				msg := strings.Join(overview.Warnings, "; ")
				if a.IsMaskAccounts() {
					msg = maskMessageEmails(msg)
				}
				a.showStatusBar("⚠️ " + msg)
			} else {
				a.showStatusBar("Đã cập nhật danh sách tài khoản")
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
	ctrl := a.controller
	mgr := a.mgr
	a.mu.Unlock()

	displayEmail := targetEmail
	if a.IsMaskAccounts() {
		displayEmail = MaskEmail(targetEmail)
	}

	fyne.Do(func() {
		a.showStatusBar(fmt.Sprintf("⏳ Đang chuyển sang %s...", displayEmail))
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
			msg := fmt.Sprintf("✅ Đã chuyển sang %s", successEmail)
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

// ToggleAlwaysOnTop switches the state of the Always-on-Top flag.
func (a *App) ToggleAlwaysOnTop() {
	a.mu.Lock()
	a.alwaysOnTop = !a.alwaysOnTop
	enabled := a.alwaysOnTop
	a.mu.Unlock()

	setWindowAlwaysOnTop(windowTitle, enabled)
	a.syncAlwaysOnTopBtn(enabled)
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

// SetAlwaysOnTop explicitly sets the Always-on-Top state.
func (a *App) SetAlwaysOnTop(enabled bool) {
	a.mu.Lock()
	a.alwaysOnTop = enabled
	a.mu.Unlock()

	setWindowAlwaysOnTop(windowTitle, enabled)
	a.syncAlwaysOnTopBtn(enabled)
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

// SetAutoSwitchEnabled explicitly sets the auto-switch state in the UI and controller.
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
	a.mu.Unlock()

	if toggle != nil && toggle.IsOn() != enabled {
		toggle.SetOn(enabled)
	}
	if ctrl != nil {
		ctrl.SetEnabled(enabled)
	}
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
		pref, err := LoadUIPreferences(path)
		if err != nil || pref == nil {
			pref = &UIPreferences{Theme: theme}
		}
		pref.MaskAccounts = mask
		_ = SaveUIPreferences(path, pref)
	}()

	// Status bar transient feedback
	if mask {
		a.showStatusBar("🔒 Đã ẩn thông tin tài khoản")
	} else {
		a.showStatusBar("🔓 Đã hiển thị đầy đủ tài khoản")
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
	a.mu.Unlock()

	palette := GetThemePalette(a.currentTheme)

	// 1. Language Server status in TopSection Row 2
	if a.serverStatusLabel != nil {
		if overview.LanguageServerRunning && overview.ServerInfo != nil {
			a.serverStatusLabel.SetText(fmt.Sprintf("Hoạt động  | PID %d | Port %d", overview.ServerInfo.PID, overview.ServerInfo.Port))
		} else {
			a.serverStatusLabel.SetText("Không hoạt động  | Dự phòng WinCred")
		}
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
				emptyLabel := widget.NewLabelWithStyle("Không tìm thấy tài khoản Google nào trên Chrome.", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})
				a.accountListContainer.Add(container.NewPadded(emptyLabel))
			} else {
				for _, acc := range overview.Accounts {
					var quotaData *telemetry.AccountQuotaData
					if a.quotaCache != nil {
						quotaData, _ = a.quotaCache.Get(acc.Email)
					}
					cardWidget := NewAccountCardWidgetWithQuota(acc, quotaData, palette, a.SwitchAccount, a.RefreshSingleAccountQuota)
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

// showStatusBar displays a transient status message in the status bar.
func (a *App) showStatusBar(msg string) {
	if a.statusBar == nil {
		return
	}
	if a.statusBarHideTimer != nil {
		a.statusBarHideTimer.Stop()
		a.statusBarHideTimer = nil
	}
	a.statusBar.SetText(msg)
	a.statusBar.Show()
}

// clearStatusBar hides the status bar.
func (a *App) clearStatusBar() {
	if a.statusBar == nil {
		return
	}
	a.statusBar.SetText("")
	a.statusBar.Hide()
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
