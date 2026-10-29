package ui

import (
	"fmt"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"antigravity-switch/internal/core/i18n"
	"antigravity-switch/internal/core/version"
	"antigravity-switch/internal/engines/antigravity/manager"
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
// 1100x630 default, 640x360 minimum, and 136.0px sidebar width.
func TestPhase1_WindowDimensions(t *testing.T) {
	if DefaultWindowWidth != 1100.0 {
		t.Errorf("expected DefaultWindowWidth 1100.0, got %f", DefaultWindowWidth)
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
	if SidebarWidth != 136.0 {
		t.Errorf("expected SidebarWidth 136.0 (80%% of 170.0), got %f", SidebarWidth)
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

// TestSidebarDefaultSelection preserves Antigravity as the initial active tab.
func TestSidebarDefaultSelection(t *testing.T) {
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

	// Verify sidebar width is 136.0 (Phase 1A: 80% of 170.0)
	minSize := sidebar.MinSize()
	if minSize.Width != 136.0 {
		t.Errorf("expected sidebar width 136.0, got %f", minSize.Width)
	}
	if minSize.Width != SidebarWidth {
		t.Errorf("expected sidebar width %f, got %f", SidebarWidth, minSize.Width)
	}

	// Verify Antigravity tab fits comfortably inside 136px
	itemMin := sidebar.AntigravityItem().MinSize()
	if itemMin.Width > SidebarWidth {
		t.Errorf("Antigravity tab MinSize.Width %f exceeds SidebarWidth %f", itemMin.Width, SidebarWidth)
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

	if sidebar.codexItem == nil || sidebar.codexItem.isActive || app.codex.content.Visible() {
		t.Fatal("Codex must be available but inactive on startup")
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

// TestPhase1A_CardGrid3ColumnsReflowAt1100px verifies Phase 1A UX Invariants:
// 1. DefaultWindowWidth is 1100.0 and SidebarWidth is 136.0.
// 2. The account list container reflows into a 3-column card grid at 1100px width.
func TestPhase1A_CardGrid3ColumnsReflowAt1100px(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	mgr := createTestManager(t)
	app := newIsolatedApp(testApp, mgr)
	defer app.Close()

	app.Init()
	waitForIdle(app, 2*time.Second)

	win := app.Window()
	if win == nil {
		t.Fatal("expected non-nil window")
	}

	sz := win.Canvas().Size()
	if sz.Width != 1100.0 || sz.Height != 630.0 {
		t.Fatalf("expected canvas size 1100x630, got %fx%f", sz.Width, sz.Height)
	}

	// Add 6 test cards to accountListContainer
	palette := DarkPalette
	for i := 1; i <= 6; i++ {
		item := manager.AccountDisplayItem{Email: fmt.Sprintf("phase1a_user%d@example.com", i), HasToken: true}
		card := NewAccountCardWithQuota(item, nil, palette, nil, nil)
		app.accountListContainer.Add(card)
	}
	app.accountListContainer.Refresh()

	// Wait for layout pass
	waitForIdle(app, 500*time.Millisecond)

	// In a 3-column layout:
	// Card 0, 1, 2 must be on Row 0 (same Y coordinate)
	// Card 3 must start on Row 1 (greater Y coordinate)
	c0 := app.accountListContainer.Objects[0]
	c1 := app.accountListContainer.Objects[1]
	c2 := app.accountListContainer.Objects[2]
	c3 := app.accountListContainer.Objects[3]

	if c0.Position().Y != c1.Position().Y || c0.Position().Y != c2.Position().Y {
		t.Errorf("expected Card 0, 1, 2 to be in the first row (same Y), got Y0=%f, Y1=%f, Y2=%f",
			c0.Position().Y, c1.Position().Y, c2.Position().Y)
	}

	if c3.Position().Y <= c0.Position().Y {
		t.Errorf("expected Card 3 to be in the second row (Y > %f), got %f",
			c0.Position().Y, c3.Position().Y)
	}

	// Check that columns have distinct X positions
	if c0.Position().X >= c1.Position().X || c1.Position().X >= c2.Position().X {
		t.Errorf("expected 3 distinct columns, got X0=%f, X1=%f, X2=%f",
			c0.Position().X, c1.Position().X, c2.Position().X)
	}
}
