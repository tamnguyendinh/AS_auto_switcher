package ui

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"

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
