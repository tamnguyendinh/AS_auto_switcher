package ui

import (
	"image"
	"image/color"
	"image/png"
	"bytes"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
)

// generateTrayIconBytes creates a simple 32x32 RGBA lightning/dot icon for the system tray.
func generateTrayIconBytes() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	// Draw a rounded circle/dot with green & blue accent
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			dx := float64(x - 16)
			dy := float64(y - 16)
			distSq := dx*dx + dy*dy
			if distSq <= 14*14 {
				if distSq <= 8*8 {
					// Center core: bright green
					img.Set(x, y, color.RGBA{R: 0xa6, G: 0xe3, B: 0xa1, A: 0xff})
				} else {
					// Outer ring: blue
					img.Set(x, y, color.RGBA{R: 0x89, G: 0xb4, B: 0xfa, A: 0xff})
				}
			} else {
				img.Set(x, y, color.RGBA{R: 0, G: 0, B: 0, A: 0})
			}
		}
	}

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

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
			}
		}),
		fyne.NewMenuItem("Làm mới Chrome", func() {
			a.Refresh()
		}),
		fyne.NewMenuItem("Lưu phiên hiện tại", func() {
			a.SaveSession()
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Thoát", func() {
			a.app.Quit()
		}),
	)

	desk.SetSystemTrayMenu(menu)

	// Set tray icon
	iconData := generateTrayIconBytes()
	if len(iconData) > 0 {
		trayRes := fyne.NewStaticResource("tray_icon.png", iconData)
		desk.SetSystemTrayIcon(trayRes)
	} else {
		desk.SetSystemTrayIcon(theme.AccountIcon())
	}

	if a.win != nil {
		desk.SetSystemTrayWindow(a.win)
		// Minimize to tray on close
		a.win.SetCloseIntercept(func() {
			a.win.Hide()
		})
	}
}
