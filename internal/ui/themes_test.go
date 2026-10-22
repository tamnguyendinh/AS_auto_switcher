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
	"fyne.io/fyne/v2/widget"

	"antigravity-switch/internal/manager"
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

func TestUIPreferences_AlwaysOnTop_BackwardCompatibility(t *testing.T) {
	// Case A: AlwaysOnTop = true -> Marshal to JSON must contain "always_on_top": true
	topTrue := true
	prefA := UIPreferences{
		Theme:        ThemeDark,
		MaskAccounts: false,
		AutoSwitch:   true,
		AlwaysOnTop:  &topTrue,
	}
	bytesA, err := json.Marshal(prefA)
	if err != nil {
		t.Fatalf("failed marshaling prefA: %v", err)
	}
	strA := string(bytesA)
	if !strings.Contains(strA, `"always_on_top":true`) && !strings.Contains(strA, `"always_on_top": true`) {
		t.Errorf("Case A: expected JSON to contain 'always_on_top': true, got: %s", strA)
	}

	// Case B: AlwaysOnTop = false -> Marshal to JSON must contain "always_on_top": false
	topFalse := false
	prefB := UIPreferences{
		Theme:        ThemeWarm,
		MaskAccounts: true,
		AutoSwitch:   false,
		AlwaysOnTop:  &topFalse,
	}
	bytesB, err := json.Marshal(prefB)
	if err != nil {
		t.Fatalf("failed marshaling prefB: %v", err)
	}
	strB := string(bytesB)
	if !strings.Contains(strB, `"always_on_top":false`) && !strings.Contains(strB, `"always_on_top": false`) {
		t.Errorf("Case B: expected JSON to contain 'always_on_top': false, got: %s", strB)
	}

	// Case C: Backward Compatibility - Legacy JSON without "always_on_top"
	legacyJSON := []byte(`{"theme":"warm","mask_accounts":true,"auto_switch":true}`)
	var unmarshaledPref UIPreferences
	if err := json.Unmarshal(legacyJSON, &unmarshaledPref); err != nil {
		t.Fatalf("failed unmarshaling legacy JSON: %v", err)
	}
	if unmarshaledPref.AlwaysOnTop != nil {
		t.Errorf("Case C: direct unmarshal should have nil AlwaysOnTop, got: %v", *unmarshaledPref.AlwaysOnTop)
	}

	// Load via LoadUIPreferences: must automatically assign canonical default true
	tmpDir := t.TempDir()
	legacyFile := filepath.Join(tmpDir, "ui_preferences.json")
	if err := os.WriteFile(legacyFile, legacyJSON, 0644); err != nil {
		t.Fatalf("failed writing legacy json file: %v", err)
	}

	loadedPref, err := LoadUIPreferences(legacyFile)
	if err != nil {
		t.Fatalf("failed loading legacy preferences: %v", err)
	}
	if loadedPref.AlwaysOnTop == nil {
		t.Fatalf("Case C: loadedPref.AlwaysOnTop should not be nil")
	}
	if !*loadedPref.AlwaysOnTop {
		t.Errorf("Case C: expected AlwaysOnTop default to true for legacy file, got false")
	}
	if loadedPref.Theme != ThemeWarm {
		t.Errorf("Case C: expected Theme to remain 'warm', got: %s", loadedPref.Theme)
	}
	if !loadedPref.MaskAccounts {
		t.Errorf("Case C: expected MaskAccounts to remain true, got false")
	}
	if !loadedPref.AutoSwitch {
		t.Errorf("Case C: expected AutoSwitch to remain true, got false")
	}
}

func TestUIPreferences_JSONSerialization_WithAlwaysOnTop(t *testing.T) {
	TestUIPreferences_AlwaysOnTop_BackwardCompatibility(t)
}

func TestApp_AlwaysOnTop_RoundTrip(t *testing.T) {
	fyneApp := test.NewApp()
	defer fyneApp.Quit()
	app := newIsolatedApp(fyneApp, nil)
	defer app.Close()

	tmpDir := t.TempDir()
	prefPath := filepath.Join(tmpDir, "ui_preferences.json")
	app.SetPreferencesPath(prefPath)
	app.Init()

	waitForPref := func(check func(*UIPreferences) bool, timeout time.Duration) *UIPreferences {
		deadline := time.Now().Add(timeout)
		for time.Now().Before(deadline) {
			prefs, err := LoadUIPreferences(prefPath)
			if err == nil && prefs != nil && check(prefs) {
				return prefs
			}
			time.Sleep(10 * time.Millisecond)
		}
		prefs, _ := LoadUIPreferences(prefPath)
		return prefs
	}

	// Initial default should be true
	if !app.IsAlwaysOnTop() {
		t.Fatalf("expected initial app.IsAlwaysOnTop() == true, got false")
	}

	// 1. Toggle to false
	app.ToggleAlwaysOnTop()
	if app.IsAlwaysOnTop() {
		t.Fatalf("expected app.IsAlwaysOnTop() == false after toggle, got true")
	}

	// Polling wait for disk persistence
	pref1 := waitForPref(func(p *UIPreferences) bool {
		return p.AlwaysOnTop != nil && *p.AlwaysOnTop == false
	}, 1*time.Second)
	if pref1 == nil || pref1.AlwaysOnTop == nil || *pref1.AlwaysOnTop != false {
		t.Fatalf("expected disk preference always_on_top == false, got: %#v", pref1)
	}

	// 2. Toggle back to true
	app.ToggleAlwaysOnTop()
	if !app.IsAlwaysOnTop() {
		t.Fatalf("expected app.IsAlwaysOnTop() == true after second toggle, got false")
	}

	pref2 := waitForPref(func(p *UIPreferences) bool {
		return p.AlwaysOnTop != nil && *p.AlwaysOnTop == true
	}, 1*time.Second)
	if pref2 == nil || pref2.AlwaysOnTop == nil || *pref2.AlwaysOnTop != true {
		t.Fatalf("expected disk preference always_on_top == true, got: %#v", pref2)
	}

	// 3. SetAlwaysOnTop(false) explicit test
	app.SetAlwaysOnTop(false)
	if app.IsAlwaysOnTop() {
		t.Fatalf("expected app.IsAlwaysOnTop() == false after SetAlwaysOnTop(false), got true")
	}
	pref3 := waitForPref(func(p *UIPreferences) bool {
		return p.AlwaysOnTop != nil && *p.AlwaysOnTop == false
	}, 1*time.Second)
	if pref3 == nil || pref3.AlwaysOnTop == nil || *pref3.AlwaysOnTop != false {
		t.Fatalf("expected disk preference always_on_top == false after SetAlwaysOnTop(false), got: %#v", pref3)
	}
}

func TestApp_AlwaysOnTopPersistence_RoundTrip(t *testing.T) {
	TestApp_AlwaysOnTop_RoundTrip(t)
}

func TestApp_AlwaysOnTop_StartupRestoration(t *testing.T) {
	mgr := createTestManager(t)
	tempDir := filepath.Dir(mgr.Store().GetStoreDir())
	logsDir := filepath.Join(tempDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		t.Fatalf("failed creating logs directory: %v", err)
	}

	prefPath := filepath.Join(logsDir, "ui_preferences.json")
	initialJSON := []byte(`{"theme":"light","mask_accounts":true,"auto_switch":false,"always_on_top":false}`)
	if err := os.WriteFile(prefPath, initialJSON, 0644); err != nil {
		t.Fatalf("failed writing initial preferences: %v", err)
	}

	fyneApp := test.NewApp()
	defer fyneApp.Quit()
	app := newIsolatedApp(fyneApp, mgr)
	defer app.Close()
	app.Init()

	// Assert App state restored from disk
	if app.IsAlwaysOnTop() {
		t.Fatalf("expected app.IsAlwaysOnTop() == false, got true")
	}
	if app.CurrentTheme() != ThemeLight {
		t.Errorf("expected CurrentTheme 'light', got '%s'", app.CurrentTheme())
	}
	if !app.IsMaskAccounts() {
		t.Errorf("expected IsMaskAccounts() true, got false")
	}
	if app.IsAutoSwitchEnabled() {
		t.Errorf("expected IsAutoSwitchEnabled() false, got true")
	}

	// Assert Button widget importance reflects unpinned state (MediumImportance)
	if app.alwaysOnTopBtn == nil {
		t.Fatalf("expected non-nil alwaysOnTopBtn")
	}
	if app.alwaysOnTopBtn.Importance != widget.MediumImportance {
		t.Errorf("expected alwaysOnTopBtn.Importance == MediumImportance (%v), got: %v",
			widget.MediumImportance, app.alwaysOnTopBtn.Importance)
	}
}

func TestApp_StartupRestoresAlwaysOnTopState(t *testing.T) {
	TestApp_AlwaysOnTop_StartupRestoration(t)
}

func TestApp_UIPreferences_4WayCrossPreservation(t *testing.T) {
	fyneApp := test.NewApp()
	defer fyneApp.Quit()
	app := newIsolatedApp(fyneApp, nil)
	defer app.Close()

	tmpDir := t.TempDir()
	prefPath := filepath.Join(tmpDir, "ui_preferences.json")
	app.SetPreferencesPath(prefPath)
	app.Init()

	waitForPref := func(check func(*UIPreferences) bool, timeout time.Duration) *UIPreferences {
		deadline := time.Now().Add(timeout)
		for time.Now().Before(deadline) {
			prefs, err := LoadUIPreferences(prefPath)
			if err == nil && prefs != nil && check(prefs) {
				return prefs
			}
			time.Sleep(10 * time.Millisecond)
		}
		prefs, _ := LoadUIPreferences(prefPath)
		if prefs != nil && check(prefs) {
			return prefs
		}
		return nil
	}

	// Set initial 4-way specific state per SPEC Section IV:
	// Theme: warm, MaskAccounts: true, AutoSwitch: true, AlwaysOnTop: false
	topFalse := false
	initialPref := &UIPreferences{
		Theme:        ThemeWarm,
		MaskAccounts: true,
		AutoSwitch:   true,
		AlwaysOnTop:  &topFalse,
	}
	if err := SaveUIPreferences(prefPath, initialPref); err != nil {
		t.Fatalf("failed saving initial preferences: %v", err)
	}

	app.SetAlwaysOnTop(false)
	app.SetTheme(ThemeWarm)
	app.SetMaskAccounts(true)
	app.SetAutoSwitchEnabled(true)

	prefInit := waitForPref(func(p *UIPreferences) bool {
		return p.Theme == ThemeWarm && p.MaskAccounts == true && p.AutoSwitch == true &&
			p.AlwaysOnTop != nil && *p.AlwaysOnTop == false
	}, 1*time.Second)
	if prefInit == nil {
		t.Fatalf("failed establishing initial 4-way preferences")
	}

	// Step 1: SetTheme("light") -> AlwaysOnTop must remain false
	app.SetTheme(ThemeLight)
	pTheme := waitForPref(func(p *UIPreferences) bool { return p.Theme == ThemeLight }, 1*time.Second)
	if pTheme == nil || pTheme.AlwaysOnTop == nil || *pTheme.AlwaysOnTop != false {
		t.Errorf("SetTheme violated AlwaysOnTop: got %#v", pTheme)
	}
	if pTheme == nil || !pTheme.MaskAccounts || !pTheme.AutoSwitch {
		t.Errorf("SetTheme violated MaskAccounts or AutoSwitch: got %#v", pTheme)
	}

	// Step 2: SetMaskAccounts(false) -> AlwaysOnTop must remain false
	app.SetMaskAccounts(false)
	pMask := waitForPref(func(p *UIPreferences) bool { return p.MaskAccounts == false }, 1*time.Second)
	if pMask == nil || pMask.AlwaysOnTop == nil || *pMask.AlwaysOnTop != false {
		t.Errorf("SetMaskAccounts violated AlwaysOnTop: got %#v", pMask)
	}
	if pMask == nil || pMask.Theme != ThemeLight || !pMask.AutoSwitch {
		t.Errorf("SetMaskAccounts violated Theme or AutoSwitch: got %#v", pMask)
	}

	// Step 3: SetAutoSwitchEnabled(false) -> AlwaysOnTop must remain false
	app.SetAutoSwitchEnabled(false)
	pAuto := waitForPref(func(p *UIPreferences) bool { return p.AutoSwitch == false }, 1*time.Second)
	if pAuto == nil || pAuto.AlwaysOnTop == nil || *pAuto.AlwaysOnTop != false {
		t.Errorf("SetAutoSwitchEnabled violated AlwaysOnTop: got %#v", pAuto)
	}
	if pAuto == nil || pAuto.Theme != ThemeLight || pAuto.MaskAccounts != false {
		t.Errorf("SetAutoSwitchEnabled violated Theme or MaskAccounts: got %#v", pAuto)
	}

	// Step 4: ToggleAlwaysOnTop() -> Theme, MaskAccounts, AutoSwitch must remain intact
	app.ToggleAlwaysOnTop()
	pPin := waitForPref(func(p *UIPreferences) bool {
		return p.AlwaysOnTop != nil && *p.AlwaysOnTop == true
	}, 1*time.Second)
	if pPin == nil || pPin.AlwaysOnTop == nil || *pPin.AlwaysOnTop != true {
		t.Errorf("ToggleAlwaysOnTop failed to persist true: got %#v", pPin)
	}
	if pPin == nil || pPin.Theme != ThemeLight {
		t.Errorf("ToggleAlwaysOnTop changed Theme: expected 'light', got '%s'", pPin.Theme)
	}
	if pPin == nil || pPin.MaskAccounts != false {
		t.Errorf("ToggleAlwaysOnTop changed MaskAccounts: expected false, got true")
	}
	if pPin == nil || pPin.AutoSwitch != false {
		t.Errorf("ToggleAlwaysOnTop changed AutoSwitch: expected false, got true")
	}
}

func TestApp_CrossSettingPreservation_4Way(t *testing.T) {
	TestApp_UIPreferences_4WayCrossPreservation(t)
}

func waitForAppIdle(a *App) {
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		a.mu.RLock()
		busy := a.isBusy
		a.mu.RUnlock()
		if !busy {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestSyncVisualScaleAndTextures_PreservesPreferences(t *testing.T) {
	mgr := createTestManager(t)
	tempDir := filepath.Dir(mgr.Store().GetStoreDir())
	logsDir := filepath.Join(tempDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		t.Fatalf("failed creating logs directory: %v", err)
	}

	prefPath := filepath.Join(logsDir, "ui_preferences.json")
	topTrue := true
	initialPref := &UIPreferences{
		Theme:        ThemeDark,
		MaskAccounts: true,
		AutoSwitch:   false,
		AlwaysOnTop:  &topTrue,
	}
	if err := SaveUIPreferences(prefPath, initialPref); err != nil {
		t.Fatalf("failed saving initial preferences: %v", err)
	}

	fyneApp := test.NewApp()
	defer fyneApp.Quit()
	app := newIsolatedApp(fyneApp, mgr)
	defer app.Close()

	app.Init()
	waitForAppIdle(app)

	// Ensure any async initialization write settles before testing sync
	app.prefSaveMu.Lock()
	app.prefSaveMu.Unlock()

	infoBefore, err := os.Stat(prefPath)
	if err != nil {
		t.Fatalf("failed stating preferences file before sync: %v", err)
	}

	app.SyncVisualScaleAndTextures()
	waitForAppIdle(app)

	infoAfter, err := os.Stat(prefPath)
	if err != nil {
		t.Fatalf("failed stating preferences file after sync: %v", err)
	}

	if infoAfter.ModTime() != infoBefore.ModTime() {
		t.Errorf("preferences file was modified during sync: before %v, after %v", infoBefore.ModTime(), infoAfter.ModTime())
	}

	loaded, err := LoadUIPreferences(prefPath)
	if err != nil {
		t.Fatalf("failed loading preferences after sync: %v", err)
	}

	if loaded.Theme != ThemeDark {
		t.Errorf("expected Theme %s, got %s", ThemeDark, loaded.Theme)
	}
	if loaded.MaskAccounts != true {
		t.Errorf("expected MaskAccounts true, got false")
	}
	if loaded.AutoSwitch != false {
		t.Errorf("expected AutoSwitch false, got true")
	}
	if loaded.AlwaysOnTop == nil || *loaded.AlwaysOnTop != true {
		t.Errorf("expected AlwaysOnTop true, got %v", loaded.AlwaysOnTop)
	}
}

func TestSyncVisualScaleAndTextures_RebuildsCardsWhenOverviewPresent(t *testing.T) {
	fyneApp := test.NewApp()
	defer fyneApp.Quit()
	app := newIsolatedApp(fyneApp, nil)
	defer app.Close()

	app.Init()
	waitForAppIdle(app)

	overview := &manager.AccountOverview{
		Accounts: []manager.AccountDisplayItem{
			{Email: "user1@example.com", Name: "User 1"},
			{Email: "user2@example.com", Name: "User 2"},
		},
	}

	app.updateUI(overview)

	if len(app.accountCards) != 2 {
		t.Fatalf("expected 2 account cards before sync, got %d", len(app.accountCards))
	}
	firstCardBefore := app.accountCards[0]

	app.SyncVisualScaleAndTextures()
	waitForAppIdle(app)

	if len(app.accountCards) != 2 {
		t.Fatalf("expected 2 account cards after sync, got %d", len(app.accountCards))
	}
	firstCardAfter := app.accountCards[0]

	if firstCardBefore == firstCardAfter {
		t.Errorf("expected first account card to be rebuilt into a new instance, but pointer was unchanged (%p)", firstCardBefore)
	}
}

func TestSyncVisualScaleAndTextures_SafeWhenOverviewNil(t *testing.T) {
	fyneApp := test.NewApp()
	defer fyneApp.Quit()
	app := newIsolatedApp(fyneApp, nil)
	defer app.Close()

	app.Init()
	waitForAppIdle(app)

	// Ensure overview is nil
	app.mu.Lock()
	app.overview = nil
	app.mu.Unlock()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("SyncVisualScaleAndTextures panicked with nil overview: %v", r)
		}
	}()

	app.SyncVisualScaleAndTextures()
	waitForAppIdle(app)

	// Verify header canvas objects and toggles are intact and refreshed
	if app.monogramBg == nil {
		t.Errorf("expected non-nil monogramBg")
	}
	if app.autoSwitchToggle == nil {
		t.Errorf("expected non-nil autoSwitchToggle")
	}
	if app.maskAccountsToggle == nil {
		t.Errorf("expected non-nil maskAccountsToggle")
	}
}
