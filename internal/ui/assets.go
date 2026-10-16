package ui

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed assets/logo.png
var logoBytes []byte

// AppLogoResource provides the high-resolution brand logo static resource
// used across application icon, window icon, dashboard header, and system tray.
var AppLogoResource = fyne.NewStaticResource("app_logo.png", logoBytes)
