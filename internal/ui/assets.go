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

//go:embed assets/antigravity_logo.png
var antigravityLogoBytes []byte

// AntigravityLogoResource cung cấp logo chính thức của Google Antigravity IDE cho thanh Sidebar.
var AntigravityLogoResource = fyne.NewStaticResource("antigravity_logo.png", antigravityLogoBytes)
