package ui

import (
	"encoding/json"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"

	"antigravity-switch/internal/telemetry"
)

func assertColorMatches(t *testing.T, themeName, tokenName string, actual color.Color, expected color.NRGBA) {
	t.Helper()
	r, g, b, a := actual.RGBA()
	expR, expG, expB, expA := expected.RGBA()
	if r != expR || g != expG || b != expB || a != expA {
		t.Errorf("[%s] %s mismatch: got RGBA(%d,%d,%d,%d), expected RGBA(%d,%d,%d,%d)",
			themeName, tokenName, r, g, b, a, expR, expG, expB, expA)
	}
}

func TestThemeColorTokens(t *testing.T) {
	// 1. DarkPalette
	if DarkPalette.Name != ThemeDark {
		t.Errorf("expected DarkPalette name %s, got %s", ThemeDark, DarkPalette.Name)
	}
	if DarkPalette.Background.R != 0x11 || DarkPalette.Background.G != 0x11 || DarkPalette.Background.B != 0x1b {
		t.Errorf("unexpected DarkPalette background: %#v", DarkPalette.Background)
	}

	// 2. LightPalette
	if LightPalette.Name != ThemeLight {
		t.Errorf("expected LightPalette name %s, got %s", ThemeLight, LightPalette.Name)
	}
	if LightPalette.Background.R != 0xf8 || LightPalette.Background.G != 0xfa || LightPalette.Background.B != 0xfc {
		t.Errorf("unexpected LightPalette background: %#v", LightPalette.Background)
	}
	if LightPalette.Card.R != 0xff || LightPalette.Card.G != 0xff || LightPalette.Card.B != 0xff {
		t.Errorf("unexpected LightPalette card: %#v", LightPalette.Card)
	}

	// 3. WarmPalette (UI-TR-07: Blue-light reduction sepia/amber)
	if WarmPalette.Name != ThemeWarm {
		t.Errorf("expected WarmPalette name %s, got %s", ThemeWarm, WarmPalette.Name)
	}
	if WarmPalette.Background.R != 0x24 || WarmPalette.Background.G != 0x1e || WarmPalette.Background.B != 0x17 {
		t.Errorf("unexpected WarmPalette background: %#v", WarmPalette.Background)
	}
	if WarmPalette.Primary.R != 0xd9 || WarmPalette.Primary.G != 0x77 || WarmPalette.Primary.B != 0x06 {
		t.Errorf("unexpected WarmPalette primary (amber): %#v", WarmPalette.Primary)
	}
}

func TestThemeInterfaceImplementations(t *testing.T) {
	themes := map[string]struct {
		theme   fyne.Theme
		palette ThemePalette
	}{
		ThemeDark:  {theme: &DarkTheme{}, palette: DarkPalette},
		ThemeLight: {theme: &LightTheme{}, palette: LightPalette},
		ThemeWarm:  {theme: &WarmTheme{}, palette: WarmPalette},
	}

	for name, tc := range themes {
		t.Run(name, func(t *testing.T) {
			th := tc.theme
			pal := tc.palette

			// Exact RGBA token checks per FINDING-03
			assertColorMatches(t, name, "Background", th.Color(theme.ColorNameBackground, theme.VariantDark), pal.Background)
			assertColorMatches(t, name, "InputBackground (Card)", th.Color(theme.ColorNameInputBackground, theme.VariantDark), pal.Card)
			assertColorMatches(t, name, "Button", th.Color(theme.ColorNameButton, theme.VariantDark), pal.Button)
			assertColorMatches(t, name, "Foreground (Text)", th.Color(theme.ColorNameForeground, theme.VariantDark), pal.Text)
			assertColorMatches(t, name, "Primary", th.Color(theme.ColorNamePrimary, theme.VariantDark), pal.Primary)
			assertColorMatches(t, name, "Success", th.Color(theme.ColorNameSuccess, theme.VariantDark), pal.Success)
			assertColorMatches(t, name, "Warning", th.Color(theme.ColorNameWarning, theme.VariantDark), pal.Warning)
			assertColorMatches(t, name, "Error", th.Color(theme.ColorNameError, theme.VariantDark), pal.Danger)

			// Font, Icon, Size
			font := th.Font(fyne.TextStyle{})
			if font == nil {
				t.Errorf("%s: Font returned nil", name)
			}
			icon := th.Icon(theme.IconNameViewRefresh)
			if icon == nil {
				t.Errorf("%s: Icon returned nil", name)
			}
			size := th.Size(theme.SizeNamePadding)
			if size <= 0 {
				t.Errorf("%s: Size returned non-positive value %f", name, size)
			}
		})
	}
}

func TestNewThemeInstance(t *testing.T) {
	if _, ok := NewThemeInstance(ThemeLight).(*LightTheme); !ok {
		t.Errorf("expected *LightTheme for %s", ThemeLight)
	}
	if _, ok := NewThemeInstance(ThemeWarm).(*WarmTheme); !ok {
		t.Errorf("expected *WarmTheme for %s", ThemeWarm)
	}
	if _, ok := NewThemeInstance(ThemeDark).(*DarkTheme); !ok {
		t.Errorf("expected *DarkTheme for %s", ThemeDark)
	}
	if _, ok := NewThemeInstance("unknown").(*DarkTheme); !ok {
		t.Errorf("expected *DarkTheme for unknown")
	}
}

func TestDefaultUIPreferencesPath(t *testing.T) {
	if DefaultUIPreferencesPath != "logs/ui_preferences.json" {
		t.Errorf("expected DefaultUIPreferencesPath to be 'logs/ui_preferences.json', got %s", DefaultUIPreferencesPath)
	}
}

func TestUIPreferencesPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	prefFile := filepath.Join(tmpDir, "ui_preferences.json")

	// 1. Missing file returns default dark theme
	pref, err := LoadUIPreferences(prefFile)
	if err != nil {
		t.Fatalf("unexpected error on missing file: %v", err)
	}
	if pref.Theme != ThemeDark {
		t.Errorf("expected default theme %s, got %s", ThemeDark, pref.Theme)
	}

	// 2. Save warm theme
	if err := SaveUIPreferences(prefFile, &UIPreferences{Theme: ThemeWarm}); err != nil {
		t.Fatalf("failed saving preferences: %v", err)
	}

	// 3. Load back warm theme
	loaded, err := LoadUIPreferences(prefFile)
	if err != nil {
		t.Fatalf("failed loading preferences: %v", err)
	}
	if loaded.Theme != ThemeWarm {
		t.Errorf("expected loaded theme %s, got %s", ThemeWarm, loaded.Theme)
	}

	// 4. Corrupted file fallback
	_ = os.WriteFile(prefFile, []byte("{corrupted json"), 0644)
	fallback, err := LoadUIPreferences(prefFile)
	if err != nil {
		t.Fatalf("unexpected error on corrupted file: %v", err)
	}
	if fallback.Theme != ThemeDark {
		t.Errorf("expected fallback theme %s, got %s", ThemeDark, fallback.Theme)
	}
}

func TestAppSetThemeRuntime(t *testing.T) {
	fyneApp := test.NewApp()
	app := newIsolatedApp(fyneApp, nil)
	tmpDir := t.TempDir()
	app.SetPreferencesPath(filepath.Join(tmpDir, "ui_preferences.json"))
	app.SetQuotaCache(telemetry.NewQuotaCache(filepath.Join(tmpDir, "quota_log.json")))

	app.Init()

	if app.CurrentTheme() != ThemeDark {
		t.Errorf("expected initial theme %s, got %s", ThemeDark, app.CurrentTheme())
	}
	assertColorMatches(t, "Init(Dark)", "monogramBg.FillColor", app.monogramBg.FillColor, DarkPalette.Primary)

	// Switch to Light (UI-TR-06)
	app.SetTheme(ThemeLight)
	if app.CurrentTheme() != ThemeLight {
		t.Errorf("expected theme %s, got %s", ThemeLight, app.CurrentTheme())
	}
	assertColorMatches(t, "SetTheme(Light)", "monogramBg.FillColor", app.monogramBg.FillColor, LightPalette.Primary)

	// Switch to Warm (UI-TR-07)
	app.SetTheme(ThemeWarm)
	if app.CurrentTheme() != ThemeWarm {
		t.Errorf("expected theme %s, got %s", ThemeWarm, app.CurrentTheme())
	}
	assertColorMatches(t, "SetTheme(Warm)", "monogramBg.FillColor", app.monogramBg.FillColor, WarmPalette.Primary)

	// Switch back to Dark
	app.SetTheme(ThemeDark)
	if app.CurrentTheme() != ThemeDark {
		t.Errorf("expected theme %s, got %s", ThemeDark, app.CurrentTheme())
	}
	assertColorMatches(t, "SetTheme(Dark)", "monogramBg.FillColor", app.monogramBg.FillColor, DarkPalette.Primary)
}

func TestUIPreferences_JSONSerialization_WithAutoSwitch(t *testing.T) {
	pref := UIPreferences{
		Theme:        "warm",
		MaskAccounts: true,
		AutoSwitch:   true,
	}

	data, err := json.Marshal(pref)
	if err != nil {
		t.Fatalf("failed marshaling preferences: %v", err)
	}

	raw := string(data)
	if !strings.Contains(raw, `"auto_switch":true`) && !strings.Contains(raw, `"auto_switch": true`) {
		t.Errorf("expected JSON to contain 'auto_switch': true, got: %s", raw)
	}

	var decoded UIPreferences
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed unmarshaling preferences: %v", err)
	}

	if decoded.Theme != "warm" {
		t.Errorf("expected Theme 'warm', got %s", decoded.Theme)
	}
	if !decoded.MaskAccounts {
		t.Errorf("expected MaskAccounts true, got false")
	}
	if !decoded.AutoSwitch {
		t.Errorf("expected AutoSwitch true, got false")
	}
}

func TestApp_AutoSwitchPersistence_RoundTrip(t *testing.T) {
	fyneApp := test.NewApp()
	defer fyneApp.Quit()
	app := newIsolatedApp(fyneApp, nil)
	defer app.Close()
	tmpDir := t.TempDir()
	tempPrefPath := filepath.Join(tmpDir, "ui_preferences.json")
	app.SetPreferencesPath(tempPrefPath)
	app.Init()

	waitForPref := func(check func(*UIPreferences) bool, timeout time.Duration) *UIPreferences {
		deadline := time.Now().Add(timeout)
		for time.Now().Before(deadline) {
			prefs, err := LoadUIPreferences(tempPrefPath)
			if err == nil && prefs != nil && check(prefs) {
				return prefs
			}
			time.Sleep(10 * time.Millisecond)
		}
		prefs, _ := LoadUIPreferences(tempPrefPath)
		return prefs
	}

	// 1. Set to true
	app.SetAutoSwitchEnabled(true)
	prefs := waitForPref(func(p *UIPreferences) bool { return p.AutoSwitch == true }, 1*time.Second)
	if prefs == nil || !prefs.AutoSwitch {
		t.Errorf("expected AutoSwitch to be persisted as true, got %#v", prefs)
	}

	// 2. Set to false
	app.SetAutoSwitchEnabled(false)
	prefs2 := waitForPref(func(p *UIPreferences) bool { return p.AutoSwitch == false }, 1*time.Second)
	if prefs2 == nil || prefs2.AutoSwitch {
		t.Errorf("expected AutoSwitch to be persisted as false, got %#v", prefs2)
	}

	// 3. Cross-setting preservation: Toggle auto-switch back to true, then change Theme and Mask
	app.SetAutoSwitchEnabled(true)
	_ = waitForPref(func(p *UIPreferences) bool { return p.AutoSwitch == true }, 1*time.Second)

	app.SetTheme(ThemeLight)
	prefsTheme := waitForPref(func(p *UIPreferences) bool { return p.Theme == ThemeLight }, 1*time.Second)
	if prefsTheme == nil || !prefsTheme.AutoSwitch {
		t.Errorf("expected AutoSwitch to remain true after SetTheme, got %#v", prefsTheme)
	}
	if prefsTheme == nil || prefsTheme.Theme != ThemeLight {
		t.Errorf("expected Theme to be %s, got %#v", ThemeLight, prefsTheme)
	}

	app.SetMaskAccounts(true)
	prefsMask := waitForPref(func(p *UIPreferences) bool { return p.MaskAccounts == true }, 1*time.Second)
	if prefsMask == nil || !prefsMask.AutoSwitch {
		t.Errorf("expected AutoSwitch to remain true after SetMaskAccounts, got %#v", prefsMask)
	}
	if prefsMask == nil || !prefsMask.MaskAccounts {
		t.Errorf("expected MaskAccounts to be true, got %#v", prefsMask)
	}
}

func TestApp_StartupRestoresAutoSwitchState(t *testing.T) {
	mgr := createTestManager(t)
	tempDir := filepath.Dir(mgr.Store().GetStoreDir())
	logsDir := filepath.Join(tempDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		t.Fatalf("failed creating logs directory: %v", err)
	}

	prefPath := filepath.Join(logsDir, "ui_preferences.json")
	initialJSON := []byte(`{"theme":"dark","mask_accounts":false,"auto_switch":true}`)
	if err := os.WriteFile(prefPath, initialJSON, 0644); err != nil {
		t.Fatalf("failed writing initial preferences: %v", err)
	}

	fyneApp := test.NewApp()
	defer fyneApp.Quit()
	app := newIsolatedApp(fyneApp, mgr)
	defer app.Close()
	app.Init()

	// Assert App state restored
	if !app.IsAutoSwitchEnabled() {
		t.Fatalf("expected app.IsAutoSwitchEnabled() == true, got false")
	}

	// Assert Toggle Widget state restored
	if app.autoSwitchToggle == nil {
		t.Fatalf("expected non-nil app.autoSwitchToggle")
	}
	if !app.autoSwitchToggle.IsOn() {
		t.Fatalf("expected app.autoSwitchToggle.IsOn() == true, got false")
	}

	// Assert AutomationController state restored
	if app.Controller() == nil {
		t.Fatalf("expected non-nil app.Controller()")
	}
	if !app.Controller().IsEnabled() {
		t.Fatalf("expected app.Controller().IsEnabled() == true, got false")
	}
}
