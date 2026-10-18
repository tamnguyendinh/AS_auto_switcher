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
	activeDotContainer   *fyne.Container
	serverStatusLabel    *widget.Label
	accountListContainer *fyne.Container
	refreshBtn           *widget.Button
	quotaRefreshAllBtn   *widget.Button
	autoRefreshLabel     *widget.Label
	lastUpdatedLabel     *widget.Label
	alwaysOnTopBtn       *widget.Button
	themeSelect          *widget.Select
	monogramBg           *canvas.Rectangle
	accountScroll        *container.Scroll
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
	if prefs, err := LoadUIPreferences(a.preferencesPath); err == nil && prefs != nil && prefs.Theme != "" {
		a.currentTheme = prefs.Theme
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

	_ = SaveUIPreferences(path, &UIPreferences{Theme: themeName})

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
		a.updateUI(overview)
	}
}

// Init configures the application window, active theme, widgets, and system tray.
func (a *App) Init() {
	a.app.Settings().SetTheme(NewThemeInstance(a.currentTheme))
	a.app.SetIcon(AppLogoResource)

	a.win = a.app.NewWindow(windowTitle)
	a.win.SetIcon(AppLogoResource)
	// Standard desktop size 1440x900 px, resizable (min size 1200x750 px handled by layout)
	a.win.Resize(fyne.NewSize(1440, 900))
	a.win.SetFixedSize(false)

	a.setupLayout()
	a.setupTray()

	if a.alwaysOnTop {
		setWindowAlwaysOnTop(windowTitle, true)
	}

	// Start 60s background auto-telemetry loop if configured
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

				nowStr := time.Now().Format("15:04:05")
				if a.lastUpdatedLabel != nil {
					a.lastUpdatedLabel.SetText(fmt.Sprintf("Cập nhật : %s", nowStr))
				}
				if a.autoRefreshLabel != nil {
					a.autoRefreshLabel.SetText(fmt.Sprintf("Cập nhật : %s", nowStr))
				}

			})
		})
		_ = a.autoLoop.Start(context.Background())
	}

	// Initial data sync
	a.Refresh()
}

// Close cleans up background loops and resources.
func (a *App) Close() {
	if a.autoLoop != nil {
		a.autoLoop.Stop()
	}
}

// ShowAndRun displays the window and runs the Fyne event loop.
func (a *App) ShowAndRun() {
	if a.win == nil {
		a.Init()
	}
	defer a.Close()

	if a.alwaysOnTop {
		setWindowAlwaysOnTop(windowTitle, true)
	}
	a.win.ShowAndRun()
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
						a.quotaRefreshAllBtn.SetText("🔄 Làm mới Quota")
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

			nowStr := time.Now().Format("15:04:05")
			if a.lastUpdatedLabel != nil {
				a.lastUpdatedLabel.SetText(fmt.Sprintf("Cập nhật : %s", nowStr))
			}
			if a.autoRefreshLabel != nil {
				a.autoRefreshLabel.SetText(fmt.Sprintf("Cập nhật : %s", nowStr))
			}

			if a.quotaRefreshAllBtn != nil {
				a.quotaRefreshAllBtn.SetText("🔄 Làm mới Quota")
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

				if targetIdx >= 0 && targetIdx < len(a.accountListContainer.Objects) {
					var quotaData *telemetry.AccountQuotaData
					if a.quotaCache != nil {
						quotaData, _ = a.quotaCache.Get(email)
					}
					if quotaData == nil {
						quotaData = data
					}
					newCard := NewAccountCardWithQuota(
						targetItem,
						quotaData,
						palette,
						a.SwitchAccount,
						a.SaveSession,
						a.RefreshSingleAccountQuota,
					)
					a.accountListContainer.Objects[targetIdx] = newCard
					a.accountListContainer.Refresh()
				}
			}

			nowStr := time.Now().Format("15:04:05")
			if a.lastUpdatedLabel != nil {
				a.lastUpdatedLabel.SetText(fmt.Sprintf("Cập nhật : %s", nowStr))
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
					a.refreshBtn.SetText("🔄 Làm mới Chrome")
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
			nowStr := time.Now().Format("15:04:05")
			if a.lastUpdatedLabel != nil {
				a.lastUpdatedLabel.SetText(fmt.Sprintf("Cập nhật : %s", nowStr))
			}
			if a.autoRefreshLabel != nil {
				a.autoRefreshLabel.SetText(fmt.Sprintf("Cập nhật : %s", nowStr))
			}
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

	if a.alwaysOnTopBtn != nil {
		if enabled {
			a.alwaysOnTopBtn.SetText("📌 Ghim : Bật")
			a.alwaysOnTopBtn.Importance = widget.HighImportance
		} else {
			a.alwaysOnTopBtn.SetText("📌 Ghim : Tắt")
			a.alwaysOnTopBtn.Importance = widget.MediumImportance
		}
		a.alwaysOnTopBtn.Refresh()
	}


}

// SetAlwaysOnTop explicitly sets the Always-on-Top state.
func (a *App) SetAlwaysOnTop(enabled bool) {
	a.mu.Lock()
	a.alwaysOnTop = enabled
	a.mu.Unlock()

	setWindowAlwaysOnTop(windowTitle, enabled)

	if a.alwaysOnTopBtn != nil {
		if enabled {
			a.alwaysOnTopBtn.SetText("📌 Ghim : Bật")
			a.alwaysOnTopBtn.Importance = widget.HighImportance
		} else {
			a.alwaysOnTopBtn.SetText("📌 Ghim : Tắt")
			a.alwaysOnTopBtn.Importance = widget.MediumImportance
		}
		a.alwaysOnTopBtn.Refresh()
	}
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
			a.serverStatusLabel.SetText(fmt.Sprintf("Language Server : PID %d | Port %d | Hoạt động", overview.ServerInfo.PID, overview.ServerInfo.Port))
		} else {
			a.serverStatusLabel.SetText("Language Server : Không phát hiện (Chế độ dự phòng WinCred)")
		}
	}

	// 2. Active Account info in-memory tracking
	if overview.ActiveEmail != "" {
		if a.activeDotContainer != nil {
			a.activeDotContainer.Objects = []fyne.CanvasObject{NewStatusDot(palette.Success, 12)}
			a.activeDotContainer.Refresh()
		}
	} else {
		if a.activeDotContainer != nil {
			a.activeDotContainer.Objects = []fyne.CanvasObject{NewStatusDot(palette.Warning, 12)}
			a.activeDotContainer.Refresh()
		}
	}

	// 3. 3-Column Account Grid with standardized 3-row cards
	if a.accountListContainer != nil {
		a.accountListContainer.Objects = nil
		if len(overview.Accounts) == 0 {
			emptyLabel := widget.NewLabelWithStyle("Không tìm thấy tài khoản Google nào trên Chrome.", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})
			a.accountListContainer.Add(container.NewPadded(emptyLabel))
		} else {
			for _, acc := range overview.Accounts {
				var quotaData *telemetry.AccountQuotaData
				if a.quotaCache != nil {
					quotaData, _ = a.quotaCache.Get(acc.Email)
				}
				card := NewAccountCardWithQuota(acc, quotaData, palette, a.SwitchAccount, a.SaveSession, a.RefreshSingleAccountQuota)
				a.accountListContainer.Add(card)
			}
		}
		minH := a.accountListContainer.MinSize().Height
		w := a.accountListContainer.Size().Width
		if w <= 0 {
			w = 1424
		}
		a.accountListContainer.Resize(fyne.NewSize(w, minH))
		a.accountListContainer.Refresh()
		if a.accountScroll != nil {
			a.accountScroll.Refresh()
		}
	}

	// 4. Update timestamp in toolbar
	nowStr := time.Now().Format("15:04:05")
	if a.lastUpdatedLabel != nil {
		a.lastUpdatedLabel.SetText(fmt.Sprintf("Cập nhật : %s", nowStr))
	}
	if a.autoRefreshLabel != nil {
		a.autoRefreshLabel.SetText(fmt.Sprintf("Cập nhật : %s", nowStr))
	}


}

// Logo returns the brand logo static resource.
func (a *App) Logo() fyne.Resource {
	return AppLogoResource
}


