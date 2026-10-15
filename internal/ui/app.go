package ui

import (
	"context"
	"fmt"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"antigravity-switch/internal/manager"
)

// App manages the Fyne GUI runtime lifecycle, state, and UI bindings.
type App struct {
	mu           sync.RWMutex
	app          fyne.App
	win          fyne.Window
	mgr          *manager.AccountManager
	overview     *manager.AccountOverview
	autoFailover bool
	isBusy       bool

	// Dynamic UI Widgets
	activeDotContainer   *fyne.Container
	activeStatusLabel    *widget.Label
	serverStatusLabel    *widget.Label
	accountListContainer *fyne.Container
	statusBarLabel       *widget.Label
	refreshBtn           *widget.Button
	saveBtn              *widget.Button
	failoverBtn          *widget.Button
	headerCard           *fyne.Container
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
		app:          fyneApp,
		mgr:          mgr,
		autoFailover: true,
	}
	return a
}

// Init configures the application window, dark theme, widgets, and system tray.
func (a *App) Init() {
	a.app.Settings().SetTheme(&SwitcherTheme{})
	a.win = a.app.NewWindow("⚡ Antigravity Multi-Account Switcher")
	a.win.Resize(fyne.NewSize(440, 560))

	a.setupLayout()
	a.setupTray()

	// Initial data sync
	a.Refresh()
}

// ShowAndRun displays the window and runs the Fyne event loop.
func (a *App) ShowAndRun() {
	if a.win == nil {
		a.Init()
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
		defer func() {
			a.mu.Lock()
			a.isBusy = false
			a.mu.Unlock()
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()

		overview, err := a.mgr.GetAccountStates(ctx)
		if err != nil {
			fyne.Do(func() {
				if a.statusBarLabel != nil {
					a.statusBarLabel.SetText(fmt.Sprintf("Lỗi làm mới: %v", err))
				}
			})
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

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		res, err := a.mgr.SwitchAccount(ctx, targetEmail, false)

		fyne.Do(func() {
			if err != nil {
				if a.statusBarLabel != nil {
					a.statusBarLabel.SetText(fmt.Sprintf("Chuyển thất bại: %v", err))
				}
				if a.win != nil {
					dialog.ShowError(err, a.win)
				}
			} else {
				if a.statusBarLabel != nil {
					a.statusBarLabel.SetText(fmt.Sprintf("Đã chuyển sang %s (%s)", res.TargetEmail, res.Duration.Round(time.Millisecond)))
				}
				if a.win != nil {
					dialog.ShowInformation("Chuyển thành công", res.Message, a.win)
				}
			}
			// Trigger a refresh to update cards and active status
			a.Refresh()
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

		email, err := a.mgr.CaptureCurrentSession()

		fyne.Do(func() {
			if err != nil {
				if a.statusBarLabel != nil {
					a.statusBarLabel.SetText(fmt.Sprintf("Lưu phiên thất bại: %v", err))
				}
				if a.win != nil {
					dialog.ShowError(err, a.win)
				}
			} else {
				if a.statusBarLabel != nil {
					a.statusBarLabel.SetText(fmt.Sprintf("Đã lưu phiên của %s vào kho JSON", email))
				}
				if a.win != nil {
					dialog.ShowInformation("Lưu phiên", fmt.Sprintf("Đã lưu thành công token của tài khoản %s!", email), a.win)
				}
			}
			a.Refresh()
		})
	}()
}

// ToggleAutoFailover switches the state of the auto-failover flag.
func (a *App) ToggleAutoFailover() {
	a.mu.Lock()
	a.autoFailover = !a.autoFailover
	enabled := a.autoFailover
	a.mu.Unlock()

	if a.failoverBtn != nil {
		if enabled {
			a.failoverBtn.SetText("⚡ Auto-Failover: BẬT")
			a.failoverBtn.Importance = widget.HighImportance
		} else {
			a.failoverBtn.SetText("⚡ Auto-Failover: TẮT")
			a.failoverBtn.Importance = widget.LowImportance
		}
	}

	if a.statusBarLabel != nil {
		if enabled {
			a.statusBarLabel.SetText("Đã kích hoạt chế độ tự động chuyển đổi Quota (Auto-Failover)")
		} else {
			a.statusBarLabel.SetText("Đã tắt chế độ tự động chuyển đổi Quota")
		}
	}
}

// updateUI refreshes all visual components based on the given overview.
func (a *App) updateUI(overview *manager.AccountOverview) {
	if overview == nil {
		return
	}

	// 1. Header Active Status
	if overview.ActiveEmail != "" {
		a.activeStatusLabel.SetText(fmt.Sprintf("● Đang hoạt động: %s", overview.ActiveEmail))
		a.activeDotContainer.Objects = []fyne.CanvasObject{NewStatusDot(ColorGreen, 12)}
	} else {
		a.activeStatusLabel.SetText("● Chưa phát hiện phiên Antigravity")
		a.activeDotContainer.Objects = []fyne.CanvasObject{NewStatusDot(ColorYellow, 12)}
	}
	a.activeDotContainer.Refresh()

	// 2. Server Status
	if overview.LanguageServerRunning && overview.ServerInfo != nil {
		a.serverStatusLabel.SetText(fmt.Sprintf("● Language Server: PID %d | Port %d | Sẵn sàng", overview.ServerInfo.PID, overview.ServerInfo.Port))
	} else {
		a.serverStatusLabel.SetText("● Language Server: Không phát hiện tiến trình (Dùng Credential Fallback)")
	}

	// 3. Account List
	a.accountListContainer.Objects = nil
	if len(overview.Accounts) == 0 {
		emptyLabel := widget.NewLabelWithStyle("Không tìm thấy tài khoản Google nào trên Chrome.", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})
		a.accountListContainer.Add(container.NewPadded(emptyLabel))
	} else {
		for _, acc := range overview.Accounts {
			card := NewAccountCard(acc, a.SwitchAccount, a.SaveSession)
			a.accountListContainer.Add(card)
		}
	}
	a.accountListContainer.Refresh()

	// 4. Status Bar
	storedCount := 0
	for _, acc := range overview.Accounts {
		if acc.HasToken {
			storedCount++
		}
	}
	nowStr := time.Now().Format("15:04:05")
	a.statusBarLabel.SetText(fmt.Sprintf("Đồng bộ lúc %s • %d tài khoản trong kho JSON • %d trên Chrome", nowStr, storedCount, len(overview.Accounts)))
}
