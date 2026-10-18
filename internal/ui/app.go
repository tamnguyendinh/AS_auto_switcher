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

	// Dynamic UI Widgets
	activeDotContainer   *fyne.Container
	activeStatusLabel    *widget.Label
	activeQuotaLabel     *widget.Label
	serverStatusLabel    *widget.Label
	accountListContainer *fyne.Container
	statusBarLabel       *widget.Label
	refreshBtn           *widget.Button
	saveBtn              *widget.Button
	quotaRefreshAllBtn   *widget.Button
	autoRefreshLabel     *widget.Label
	alwaysOnTopBtn       *widget.Button
	themeSelect          *widget.Select
	headerCard           *fyne.Container
	headerBg             *canvas.Rectangle
	statusBg             *canvas.Rectangle
	legendBg             *canvas.Rectangle
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

	if a.headerBg != nil {
		a.headerBg.FillColor = palette.Card
		a.headerBg.StrokeColor = palette.BorderSubtle
		a.headerBg.Refresh()
	}
	if a.statusBg != nil {
		a.statusBg.FillColor = palette.Card
		a.statusBg.StrokeColor = palette.BorderSubtle
		a.statusBg.Refresh()
	}
	if a.legendBg != nil {
		a.legendBg.FillColor = palette.Card
		a.legendBg.StrokeColor = palette.BorderSubtle
		a.legendBg.Refresh()
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
	a.win.Resize(fyne.NewSize(440, 560))

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
				if a.autoRefreshLabel != nil {
					a.autoRefreshLabel.SetText(fmt.Sprintf("Tự động: 60s | Cập nhật: %s", nowStr))
				}
				if a.statusBarLabel != nil {
					a.statusBarLabel.SetText(fmt.Sprintf("Tự động cập nhật quota lúc %s (%d tài khoản)", nowStr, len(results)))
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
	if a.statusBarLabel != nil {
		a.statusBarLabel.SetText("Đang quét hạn ngạch toàn bộ tài khoản...")
	}

	go func() {
		completed := false
		defer func() {
			if !completed {
				fyne.Do(func() {
					if a.quotaRefreshAllBtn != nil {
						a.quotaRefreshAllBtn.SetText("📊 Làm mới Quota Tất cả")
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

		_, successCount, errorCount, err := a.telemetryPool.PollAll(ctx)
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
			if a.autoRefreshLabel != nil {
				a.autoRefreshLabel.SetText(fmt.Sprintf("Tự động: 60s | Cập nhật: %s", nowStr))
			}
			if a.statusBarLabel != nil {
				if err != nil {
					a.statusBarLabel.SetText(fmt.Sprintf("Lỗi quét quota: %v", err))
				} else {
					a.statusBarLabel.SetText(fmt.Sprintf("Quét quota: %d thành công, %d lỗi", successCount, errorCount))
				}
			}
			if a.quotaRefreshAllBtn != nil {
				a.quotaRefreshAllBtn.SetText("📊 Làm mới Quota Tất cả")
				a.quotaRefreshAllBtn.Enable()
			}
			a.isQuotaRefreshing.Store(false)
		})
	}()
}

// RefreshSingleQuota triggers quota retrieval for a single specified account.
func (a *App) RefreshSingleQuota(email string) {
	if !a.isQuotaRefreshing.CompareAndSwap(false, true) {
		return
	}

	if a.statusBarLabel != nil {
		fyne.Do(func() {
			a.statusBarLabel.SetText(fmt.Sprintf("Đang làm mới quota của %s...", email))
		})
	}

	go func() {
		completed := false
		defer func() {
			if !completed {
				a.isQuotaRefreshing.Store(false)
			}
		}()

		if a.telemetryPool == nil {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()

		data, err := a.telemetryPool.PollAccount(ctx, email)
		if a.quotaCache != nil {
			_ = a.quotaCache.SaveToDisk()
		}

		completed = true
		fyne.Do(func() {
			a.mu.RLock()
			overview := a.overview
			a.mu.RUnlock()
			if overview != nil {
				a.updateUI(overview)
			}
			if a.statusBarLabel != nil {
				if err != nil {
					a.statusBarLabel.SetText(fmt.Sprintf("Lỗi làm mới quota %s: %v", email, err))
				} else if data != nil {
					a.statusBarLabel.SetText(fmt.Sprintf("Đã cập nhật quota %s (Claude: %s, Gemini: %s)",
						email, formatFraction(data.Claude.RemainingFraction), formatFraction(data.Gemini.RemainingFraction)))
				}
			}
			a.isQuotaRefreshing.Store(false)
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

	if a.statusBarLabel != nil {
		fyne.Do(func() {
			a.statusBarLabel.SetText("Đang quét tài khoản Chrome & kho lưu trữ...")
		})
	}

	go func() {
		if a.mgr == nil {
			a.mu.Lock()
			a.isBusy = false
			a.mu.Unlock()
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()

		overview, err := a.mgr.GetAccountStates(ctx)
		if err != nil {
			fyne.Do(func() {
				if a.statusBarLabel != nil {
					a.statusBarLabel.SetText(fmt.Sprintf("Lỗi làm mới: %v", err))
				}
				a.mu.Lock()
				a.isBusy = false
				a.mu.Unlock()
			})
			return
		}

		a.mu.Lock()
		a.overview = overview
		a.mu.Unlock()

		fyne.Do(func() {
			a.updateUI(overview)
			a.mu.Lock()
			a.isBusy = false
			a.mu.Unlock()
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

	if a.statusBarLabel != nil {
		fyne.Do(func() {
			a.statusBarLabel.SetText(fmt.Sprintf("Đang chuyển sang %s...", targetEmail))
		})
	}

	go func() {
		defer func() {
			a.mu.Lock()
			a.isBusy = false
			a.mu.Unlock()
		}()

		if a.mgr == nil {
			return
		}

		// Xóa bỏ hoàn toàn bẫy context.WithTimeout(15s).
		// Quy trình 4 bước chạy tự nhiên từ đầu đến thread cuối cùng qua context.Background().
		res, err := a.mgr.SwitchAccount(context.Background(), targetEmail, false) // forceLogout = false: smooth zero-chrome hot-swap

		if err != nil {
			fyne.Do(func() {
				if a.statusBarLabel != nil {
					a.statusBarLabel.SetText(fmt.Sprintf("Chuyển thất bại: %v", err))
				}
				if a.win != nil {
					dialog.ShowError(err, a.win)
				}
			})
			return
		}

		// Refresh account states in RAM and update cards/header immediately
		overview, _ := a.mgr.GetAccountStates(context.Background())
		a.mu.Lock()
		a.overview = overview
		a.mu.Unlock()

		fyne.Do(func() {
			a.updateUI(overview)
			if a.statusBarLabel != nil {
				statusText := fmt.Sprintf("Đã chuyển sang %s (%s)", res.TargetEmail, res.Duration.Round(time.Millisecond))
				if res != nil && res.ContinuationSummary != nil && res.ContinuationSummary.TotalThreads > 0 {
					statusText = fmt.Sprintf("Đã chuyển sang %s (%s) • Đã tiếp tục %d/%d phiên",
						res.TargetEmail, res.Duration.Round(time.Millisecond),
						res.ContinuationSummary.SuccessCount, res.ContinuationSummary.TotalThreads)
				}
				a.statusBarLabel.SetText(statusText)
			}
		})
	}()
}

// SaveSession captures the current Antigravity session from Windows Credential Manager.
func (a *App) SaveSession() {
	a.mu.Lock()
	if a.isBusy {
		a.mu.Unlock()
		return
	}
	a.isBusy = true
	a.mu.Unlock()

	if a.statusBarLabel != nil {
		fyne.Do(func() {
			a.statusBarLabel.SetText("Đang lưu phiên hiện tại vào accounts_store/...")
		})
	}

	go func() {
		defer func() {
			a.mu.Lock()
			a.isBusy = false
			a.mu.Unlock()
		}()

		if a.mgr == nil {
			return
		}
		email, err := a.mgr.CaptureCurrentSession()

		if err != nil {
			fyne.Do(func() {
				if a.statusBarLabel != nil {
					a.statusBarLabel.SetText(fmt.Sprintf("Lưu phiên thất bại: %v", err))
				}
				if a.win != nil {
					dialog.ShowError(err, a.win)
				}
			})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()

		// Refresh account states in RAM and update cards immediately
		overview, _ := a.mgr.GetAccountStates(ctx)
		a.mu.Lock()
		a.overview = overview
		a.mu.Unlock()

		fyne.Do(func() {
			a.updateUI(overview)
			if a.statusBarLabel != nil {
				a.statusBarLabel.SetText(fmt.Sprintf("Đã lưu phiên của %s vào kho JSON", email))
			}
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
			a.alwaysOnTopBtn.SetText("📌 Ghim: BẬT")
			a.alwaysOnTopBtn.Importance = widget.HighImportance
		} else {
			a.alwaysOnTopBtn.SetText("📌 Ghim: TẮT")
			a.alwaysOnTopBtn.Importance = widget.MediumImportance
		}
		a.alwaysOnTopBtn.Refresh()
	}

	if a.statusBarLabel != nil {
		if enabled {
			a.statusBarLabel.SetText("Đã bật chế độ Ghim trên cùng (Always on Top)")
		} else {
			a.statusBarLabel.SetText("Đã tắt chế độ Ghim trên cùng")
		}
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
			a.alwaysOnTopBtn.SetText("📌 Ghim: BẬT")
			a.alwaysOnTopBtn.Importance = widget.HighImportance
		} else {
			a.alwaysOnTopBtn.SetText("📌 Ghim: TẮT")
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

	palette := GetThemePalette(a.currentTheme)

	// 1. Header Active Status
	if overview.ActiveEmail != "" {
		if a.activeStatusLabel != nil {
			a.activeStatusLabel.SetText(fmt.Sprintf("● Đang hoạt động: %s", overview.ActiveEmail))
		}
		if a.activeDotContainer != nil {
			a.activeDotContainer.Objects = []fyne.CanvasObject{NewStatusDot(palette.Success, 12)}
			a.activeDotContainer.Refresh()
		}

		// Active Account Quota Summary in Header Card
		if a.activeQuotaLabel != nil {
			if a.quotaCache != nil {
				if activeQuota, exists := a.quotaCache.Get(overview.ActiveEmail); exists {
					a.activeQuotaLabel.SetText(BuildActiveQuotaSummary(activeQuota))
				} else {
					a.activeQuotaLabel.SetText("Hạn ngạch: Chưa có dữ liệu telemetry")
				}
			} else {
				a.activeQuotaLabel.SetText("Hạn ngạch: Telemetry chưa kích hoạt")
			}
		}
	} else {
		if a.activeStatusLabel != nil {
			a.activeStatusLabel.SetText("● Chưa phát hiện phiên Antigravity")
		}
		if a.activeDotContainer != nil {
			a.activeDotContainer.Objects = []fyne.CanvasObject{NewStatusDot(palette.Warning, 12)}
			a.activeDotContainer.Refresh()
		}
		if a.activeQuotaLabel != nil {
			a.activeQuotaLabel.SetText("Hạn ngạch: Không có tài khoản hoạt động")
		}
	}

	// 2. Server Status
	if a.serverStatusLabel != nil {
		if overview.LanguageServerRunning && overview.ServerInfo != nil {
			a.serverStatusLabel.SetText(fmt.Sprintf("● Language Server: PID %d | Port %d | Sẵn sàng", overview.ServerInfo.PID, overview.ServerInfo.Port))
		} else {
			a.serverStatusLabel.SetText("● Language Server: Không phát hiện tiến trình (Dùng Credential Fallback)")
		}
	}

	// 3. Account List with Quota Badges & Refresh Buttons
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
				card := NewAccountCardWithQuota(acc, quotaData, palette, a.SwitchAccount, a.SaveSession, a.RefreshSingleQuota)
				a.accountListContainer.Add(card)
			}
		}
		a.accountListContainer.Refresh()
	}

	// 4. Status Bar
	if a.statusBarLabel != nil {
		storedCount := 0
		for _, acc := range overview.Accounts {
			if acc.HasToken {
				storedCount++
			}
		}
		nowStr := time.Now().Format("15:04:05")
		a.statusBarLabel.SetText(fmt.Sprintf("Đồng bộ lúc %s • %d tài khoản trong kho JSON • %d trên Chrome", nowStr, storedCount, len(overview.Accounts)))
	}
}

// Logo returns the brand logo static resource.
func (a *App) Logo() fyne.Resource {
	return AppLogoResource
}

