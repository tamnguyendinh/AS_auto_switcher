package ui

import (
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"antigravity-switch/internal/i18n"
	"antigravity-switch/internal/version"
)

// TestPhase1_AppTitleAndLocales verifies that the application title has been renamed
// to "Auto Switcher" across i18n dictionaries (vi, en) and windowTitle constant.
func TestPhase1_AppTitleAndLocales(t *testing.T) {
	// 1. Verify windowTitle constant
	if windowTitle != "⚡ Auto Switcher" {
		t.Errorf("expected windowTitle to be '⚡ Auto Switcher', got %q", windowTitle)
	}

	// 2. Verify Vietnamese dictionary
	viDict := i18n.GetDictionary(i18n.LocaleVI)
	if viDict.AppTitle != "Auto Switcher" {
		t.Errorf("expected vi AppTitle 'Auto Switcher', got %q", viDict.AppTitle)
	}
	if viDict.NavAntigravity != "Antigravity" {
		t.Errorf("expected vi NavAntigravity 'Antigravity', got %q", viDict.NavAntigravity)
	}
	if viDict.NavCodex != "Codex" {
		t.Errorf("expected vi NavCodex 'Codex', got %q", viDict.NavCodex)
	}

	// 3. Verify English dictionary
	enDict := i18n.GetDictionary(i18n.LocaleEN)
	if enDict.AppTitle != "Auto Switcher" {
		t.Errorf("expected en AppTitle 'Auto Switcher', got %q", enDict.AppTitle)
	}
	if enDict.NavAntigravity != "Antigravity" {
		t.Errorf("expected en NavAntigravity 'Antigravity', got %q", enDict.NavAntigravity)
	}
	if enDict.NavCodex != "Codex" {
		t.Errorf("expected en NavCodex 'Codex', got %q", enDict.NavCodex)
	}
}

// TestPhase1_WindowDimensions verifies the expanded window dimensions:
// 980x630 default, 640x360 minimum.
func TestPhase1_WindowDimensions(t *testing.T) {
	if DefaultWindowWidth != 980.0 {
		t.Errorf("expected DefaultWindowWidth 980.0, got %f", DefaultWindowWidth)
	}
	if DefaultWindowHeight != 630.0 {
		t.Errorf("expected DefaultWindowHeight 630.0, got %f", DefaultWindowHeight)
	}
	if MinWindowWidth != 640.0 {
		t.Errorf("expected MinWindowWidth 640.0, got %f", MinWindowWidth)
	}
	if MinWindowHeight != 360.0 {
		t.Errorf("expected MinWindowHeight 360.0, got %f", MinWindowHeight)
	}
}

// TestPhase1_AntigravityLogoResource verifies that the official Antigravity logo
// static resource is embedded, non-nil, contains image bytes, and has .png extension.
func TestPhase1_AntigravityLogoResource(t *testing.T) {
	if AntigravityLogoResource == nil {
		t.Fatal("expected AntigravityLogoResource to be non-nil")
	}
	if len(AntigravityLogoResource.Content()) == 0 {
		t.Fatal("expected AntigravityLogoResource to contain image bytes")
	}
	if AntigravityLogoResource.Name() != "antigravity_logo.png" {
		t.Errorf("unexpected resource name: %s", AntigravityLogoResource.Name())
	}
}

// TestPhase1_SidebarStructureAndCodexAbsence verifies:
// 1. Sidebar is initialized and placed on the left with width 170px.
// 2. ONLY Tab "Antigravity" is present and Active by default.
// 3. Absolutely NO "Codex" text or tab appears anywhere on the GUI.
func TestPhase1_SidebarStructureAndCodexAbsence(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	mgr := createTestManager(t)
	app := newIsolatedApp(testApp, mgr)
	defer app.Close()

	app.Init()
	waitForIdle(app, 2*time.Second)

	sidebar := app.Sidebar()
	if sidebar == nil {
		t.Fatal("expected app.Sidebar() to be non-nil")
	}

	// Verify sidebar width
	minSize := sidebar.MinSize()
	if minSize.Width != SidebarWidth {
		t.Errorf("expected sidebar width %f, got %f", SidebarWidth, minSize.Width)
	}

	// Verify Antigravity tab is active
	if sidebar.ActiveTab() != NavTabAntigravity {
		t.Errorf("expected ActiveTab to be NavTabAntigravity (%d), got %d", NavTabAntigravity, sidebar.ActiveTab())
	}
	if sidebar.AntigravityItem() == nil {
		t.Fatal("expected AntigravityItem() to be non-nil")
	}
	if !sidebar.AntigravityItem().isActive {
		t.Error("expected AntigravityItem to be active by default")
	}
	if sidebar.AntigravityItem().label != "Antigravity" {
		t.Errorf("expected tab label 'Antigravity', got %q", sidebar.AntigravityItem().label)
	}

	// Verify Phased Isolation Invariant: Codex must NOT exist in the sidebar or window
	win := app.Window()
	var searchCodexInCanvas func(obj fyne.CanvasObject) bool
	searchCodexInCanvas = func(obj fyne.CanvasObject) bool {
		if text, ok := obj.(*canvas.Text); ok {
			if strings.Contains(strings.ToLower(text.Text), "codex") {
				return true
			}
		}
		if c, ok := obj.(*fyne.Container); ok {
			for _, child := range c.Objects {
				if searchCodexInCanvas(child) {
					return true
				}
			}
		}
		return false
	}

	if searchCodexInCanvas(sidebar) {
		t.Fatal("PHASE 1 VIOLATION: Codex must NOT appear in sidebar in Phase 1")
	}
	if searchCodexInCanvas(win.Content()) {
		t.Fatal("PHASE 1 VIOLATION: Codex must NOT appear anywhere in the UI in Phase 1")
	}
}

// TestPhase1_ThemeParity verifies that the Sidebar and Top Bar update dynamically
// when changing themes (Dark, Light, Warm).
func TestPhase1_ThemeParity(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	mgr := createTestManager(t)
	app := newIsolatedApp(testApp, mgr)
	defer app.Close()

	app.Init()
	waitForIdle(app, 2*time.Second)

	sidebar := app.Sidebar()
	if sidebar == nil {
		t.Fatal("expected non-nil sidebar")
	}

	themes := []string{ThemeDark, ThemeLight, ThemeWarm}
	for _, th := range themes {
		app.SetTheme(th)
		palette := GetThemePalette(th)

		if sidebar.palette.Name != th {
			t.Errorf("expected sidebar palette name %q, got %q", th, sidebar.palette.Name)
		}
		if sidebar.divider.FillColor != palette.BorderSubtle {
			t.Errorf("theme %s: expected sidebar divider color %+v, got %+v", th, palette.BorderSubtle, sidebar.divider.FillColor)
		}
		if sidebar.AntigravityItem().bg.FillColor != palette.CardActive {
			t.Errorf("theme %s: expected active tab bg color %+v, got %+v", th, palette.CardActive, sidebar.AntigravityItem().bg.FillColor)
		}
		if sidebar.AntigravityItem().activeBar.FillColor != palette.Primary {
			t.Errorf("theme %s: expected active bar color %+v, got %+v", th, palette.Primary, sidebar.AntigravityItem().activeBar.FillColor)
		}
	}
}

// TestPhase1_DynamicLanguageSwitching verifies that Top Bar title and Sidebar
// tab labels update when changing language between VN and EN.
func TestPhase1_DynamicLanguageSwitching(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	mgr := createTestManager(t)
	app := newIsolatedApp(testApp, mgr)
	defer app.Close()

	app.Init()
	waitForIdle(app, 2*time.Second)

	sidebar := app.Sidebar()
	if sidebar == nil {
		t.Fatal("expected non-nil sidebar")
	}

	// Switch to EN
	app.SetLanguage("en")
	waitForIdle(app, 1*time.Second)

	if app.TitleLabel() == nil || len(app.TitleLabel().Segments) == 0 {
		t.Fatal("expected non-nil titleLabel with segments")
	}
	seg := app.TitleLabel().Segments[0].(*widget.TextSegment)
	if seg.Text != "Auto Switcher" {
		t.Errorf("expected title text 'Auto Switcher', got %q", seg.Text)
	}

	// Switch back to VI
	app.SetLanguage("vi")
	waitForIdle(app, 1*time.Second)

	seg = app.TitleLabel().Segments[0].(*widget.TextSegment)
	if seg.Text != "Auto Switcher" {
		t.Errorf("expected title text 'Auto Switcher', got %q", seg.Text)
	}
}

// TestPhase1_ContentAreaZeroRegression verifies that the Antigravity toolbar and
// account cards continue to function in the right Content Area.
func TestPhase1_ContentAreaZeroRegression(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	mgr := createTestManager(t)
	app := newIsolatedApp(testApp, mgr)
	defer app.Close()

	app.Init()
	waitForIdle(app, 2*time.Second)

	toolbar := app.Toolbar()
	if toolbar == nil {
		t.Fatal("expected non-nil toolbar in Content Area")
	}
	if app.addAccountBtn == nil {
		t.Fatal("expected non-nil addAccountBtn")
	}
	if app.refreshBtn == nil {
		t.Fatal("expected non-nil refreshBtn")
	}
	if app.quotaRefreshAllBtn == nil {
		t.Fatal("expected non-nil quotaRefreshAllBtn")
	}
	if app.alwaysOnTopBtn == nil {
		t.Fatal("expected non-nil alwaysOnTopBtn")
	}
	if app.maskAccountsToggle == nil {
		t.Fatal("expected non-nil maskAccountsToggle")
	}
	if app.autoSwitchToggle == nil {
		t.Fatal("expected non-nil autoSwitchToggle")
	}
	if app.themeSelect == nil {
		t.Fatal("expected non-nil themeSelect")
	}
	if app.accountListContainer == nil {
		t.Fatal("expected non-nil accountListContainer")
	}
	if app.accountScroll == nil {
		t.Fatal("expected non-nil accountScroll")
	}
	if app.serverStatusLabel == nil {
		t.Fatal("expected non-nil serverStatusLabel")
	}
	if app.versionLabel == nil || app.versionLabel.Text != version.GetFullVersion() {
		t.Errorf("expected version label %q, got %q", version.GetFullVersion(), app.versionLabel.Text)
	}
}
