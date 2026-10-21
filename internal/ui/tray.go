package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
)

// setupTray initializes the system tray menu and minimize-to-tray close intercept.
func (a *App) setupTray() {
	desk, ok := a.app.(desktop.App)
	if !ok {
		return
	}

	// Tray Menu
	menu := fyne.NewMenu("Antigravity Switcher",
		fyne.NewMenuItem("Hiển thị Switcher", func() {
			if a.win != nil {
				a.win.Show()
				a.win.RequestFocus()
				if a.IsAlwaysOnTop() {
					setWindowAlwaysOnTop(windowTitle, true)
				}
			}
		}),
		fyne.NewMenuItem("Làm mới Chrome", func() {
			a.Refresh()
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Thoát", func() {
			a.app.Quit()
		}),
	)

	desk.SetSystemTrayMenu(menu)

	// Set tray icon to official brand logo
	if AppLogoResource != nil && len(AppLogoResource.Content()) > 0 {
		desk.SetSystemTrayIcon(AppLogoResource)
	} else {
		desk.SetSystemTrayIcon(theme.AccountIcon())
	}

	if a.win != nil {
		desk.SetSystemTrayWindow(a.win)
		// Close window intercept: Thoát sạch hoàn toàn ứng dụng
		a.win.SetCloseIntercept(func() {
			a.Close()    // 1. Dừng mọi background loop, timer, dọn dẹp tài nguyên
			a.app.Quit() // 2. Thoát hoàn toàn tiến trình Fyne GUI và System Tray
		})
	}
}
