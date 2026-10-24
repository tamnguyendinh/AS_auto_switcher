package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"

	"antigravity-switch/internal/i18n"
)

// setupTray initializes the system tray menu and minimize-to-tray close intercept.
func (a *App) setupTray() {
	desk, ok := a.app.(desktop.App)
	if !ok {
		return
	}

	a.updateTrayMenu()

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

// updateTrayMenu updates or sets the system tray menu localized per current i18n settings.
func (a *App) updateTrayMenu() {
	desk, ok := a.app.(desktop.App)
	if !ok {
		return
	}

	d := i18n.Current()
	menu := fyne.NewMenu(d.AppTitle,
		fyne.NewMenuItem(d.TrayOpen, func() {
			if a.win != nil {
				a.win.Show()
				a.win.RequestFocus()
				if a.IsAlwaysOnTop() {
					setWindowAlwaysOnTop(windowTitle, true)
				}
			}
		}),
		fyne.NewMenuItem(d.TrayRefresh, func() {
			a.Refresh()
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem(d.TrayExit, func() {
			a.app.Quit()
		}),
	)

	desk.SetSystemTrayMenu(menu)
}
