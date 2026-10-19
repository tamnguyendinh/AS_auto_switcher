package ui

import (
	"context"
	"fmt"
	"path/filepath"
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
	monogramBg           *canvas.Rectangle
	accountScroll        *container.Scroll
	statusBar            *widget.Label
	statusBarHideTimer   *time.Timer

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

	cachePath := telemetry.DefaultQuotaCachePath
	if mgr != nil && mgr.Store() != nil && mgr.Store().GetStoreDir() != "" {
		cachePath = filepath.Join(mgr.Store().GetStoreDir(), ".quota_cache.json")
		a.preferencesPath = filepath.Join(mgr.Store().GetStoreDir(), ".ui_preferences.json")
	}

	// 1. Initialize Quota Cache
	a.quotaCache = telemetry.NewQuotaCache(cachePath)
	_ = a.quotaCache.LoadFromDisk()

	// 2. Load UI Preferences (Theme)
	if prefs, err := LoadUIPreferences(a.preferencesPath); err == nil && prefs != nil {
		if prefs.Theme != "" {
			a.currentTheme = prefs.Theme
		}
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
}

// SetQuotaCache allows injecting a custom QuotaCache.
func (a *App) SetQuotaCache(cache *telemetry.QuotaCache) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.quotaCache = cache
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
	a.mu.Unlock()

	prefs, err := LoadUIPreferences(path)
	if err != nil || prefs == nil {
		prefs = &UIPreferences{Theme: themeName}
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

	// Khởi tạo kích thước chuẩn 1008x630 px và khóa cứng
	a.win.Resize(fyne.NewSize(DefaultWindowWidth, DefaultWindowHeight))
	a.win.SetFixedSize(true)

	a.setupLayout()
	a.setupTray()

	if a.alwaysOnTop {
		setWindowAlwaysOnTop(windowTitle, true)
	}

	// 60s background auto-telemetry loop
	if a.autoLoop != nil {
		a.autoLoop.SetUpdateCallback(func(results []*telemetry.AccountQuotaData) {
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
	if a.autoLoop != nil {
		a.autoLoop.Stop()
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

		_, _, _, _ = a.telemetryPool.PollAll(ctx)
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

		data, _ := a.telemetryPool.PollAccount(ctx, email)
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
						a.SaveSession,
						a.RefreshSingleAccountQuota,
					)
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

			return
		}

		a.mu.Lock()
		a.overview = overview
		a.mu.Unlock()

		fyne.Do(func() {
			a.updateUI(overview)

		})
	}()
}

// SwitchAccount initiates an asynchronous 4-step hot-swap to the specified email.
func (a *App) SwitchAccount(targetEmail string) {
	a.mu.Lock()
	if a.isBusy {
		a.mu.Unlock()
		return
	}
	a.isBusy = true
	a.mu.Unlock()

	fyne.Do(func() {
		a.showStatusBar(fmt.Sprintf("⏳ Đang chuyển sang %s...", targetEmail))
	})

	go func() {
		defer func() {
			a.mu.Lock()
			a.isBusy = false
			a.mu.Unlock()
		}()

		if a.mgr == nil {
			return
		}

		_, err := a.mgr.SwitchAccount(context.Background(), targetEmail, false)

		if err != nil {
			fyne.Do(func() {
				if a.win != nil {
					dialog.ShowError(err, a.win)
				}
				a.clearStatusBar()
			})
			return
		}

		// Refresh account states in RAM and update cards immediately
		overview, _ := a.mgr.GetAccountStates(context.Background())
		a.mu.Lock()
		a.overview = overview
		a.mu.Unlock()

		fyne.Do(func() {
			a.updateUI(overview)
			a.showStatusBar(fmt.Sprintf("✅ Đã chuyển sang %s", targetEmail))
			a.autoHideStatusBar(3 * time.Second)
		})
	}()
}

// SaveSession saves the currently active Chrome session to disk.
func (a *App) SaveSession() {
	a.mu.Lock()
	if a.isBusy {
		a.mu.Unlock()
		return
	}
	a.isBusy = true
	a.mu.Unlock()

	fyne.Do(func() {
		a.showStatusBar("⏳ Đang lưu phiên...")
	})

	go func() {
		defer func() {
			a.mu.Lock()
			a.isBusy = false
			a.mu.Unlock()
		}()

		if a.mgr == nil {
			return
		}

		_, err := a.mgr.CaptureCurrentSession()
		if err != nil {
			fyne.Do(func() {
				if a.win != nil {
					dialog.ShowError(err, a.win)
				}
				a.clearStatusBar()
			})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()

		overview, _ := a.mgr.GetAccountStates(ctx)
		a.mu.Lock()
		a.overview = overview
		a.mu.Unlock()

		fyne.Do(func() {
			a.updateUI(overview)
			a.showStatusBar("✅ Đã lưu phiên thành công")
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
	if enabled {
		a.alwaysOnTopBtn.SetText("📌 Ghim : Bật")
		a.alwaysOnTopBtn.Importance = widget.HighImportance
	} else {
		a.alwaysOnTopBtn.SetText("📌 Ghim : Tắt")
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
				if a.accountCards[i] == nil || a.accountCards[i].Email != acc.Email || a.accountCards[i].IsActive != acc.IsActive {
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
					cardWidget := NewAccountCardWidgetWithQuota(acc, quotaData, palette, a.SwitchAccount, a.SaveSession, a.RefreshSingleAccountQuota)
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


