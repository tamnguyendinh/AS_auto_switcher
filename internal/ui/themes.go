package ui

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	"antigravity-switch/internal/core/preferences"
)

const (
	// Compact typography shared by all themes and directly drawn quota labels.
	BodyTextSize      float32            = 10
	TitleTextSize     float32            = 10
	titleTextSizeName fyne.ThemeSizeName = "appTitleText"

	// Theme identifiers
	ThemeDark  = "dark"
	ThemeLight = "light"
	ThemeWarm  = "warm"

	// DefaultUIPreferencesPath is the canonical disk path for UI settings.
	DefaultUIPreferencesPath = preferences.DefaultUIPreferencesPath
)

// UIPreferences stores persistent user interface settings.
type UIPreferences = preferences.UIPreferences

// ThemePalette defines semantic color tokens for custom Canvas rendering across themes.
type ThemePalette struct {
	Name         string
	Background   color.NRGBA
	Card         color.NRGBA
	CardActive   color.NRGBA
	BorderSubtle color.NRGBA
	BorderActive color.NRGBA
	Text         color.NRGBA
	TextMuted    color.NRGBA
	Primary      color.NRGBA
	Success      color.NRGBA
	Warning      color.NRGBA
	Danger       color.NRGBA
	Button       color.NRGBA
}

var (
	// DarkPalette: Catppuccin Mocha inspired dark theme
	DarkPalette = ThemePalette{
		Name:         ThemeDark,
		Background:   color.NRGBA{R: 0x11, G: 0x11, B: 0x1b, A: 0xff}, // #11111b (Deep Crust Canvas)
		Card:         color.NRGBA{R: 0x18, G: 0x18, B: 0x25, A: 0xff}, // #181825 (Surface Mantle)
		CardActive:   color.NRGBA{R: 0x1e, G: 0x1e, B: 0x2e, A: 0xff}, // #1e1e2e (Surface Base Active)
		BorderSubtle: color.NRGBA{R: 0x58, G: 0x5b, B: 0x70, A: 0xff}, // #585b70 (High-Contrast Border)
		BorderActive: color.NRGBA{R: 0xa6, G: 0xe3, B: 0xa1, A: 0xff}, // #a6e3a1 (Green Accent sáng rõ)
		Text:         color.NRGBA{R: 0xcd, G: 0xd6, B: 0xf4, A: 0xff}, // #cdd6f4
		TextMuted:    color.NRGBA{R: 0xa6, G: 0xad, B: 0xc8, A: 0xff}, // #a6adc8
		Primary:      color.NRGBA{R: 0x89, G: 0xb4, B: 0xfa, A: 0xff}, // #89b4fa
		Success:      color.NRGBA{R: 0xa6, G: 0xe3, B: 0xa1, A: 0xff}, // #a6e3a1
		Warning:      color.NRGBA{R: 0xf9, G: 0xe2, B: 0xaf, A: 0xff}, // #f9e2af
		Danger:       color.NRGBA{R: 0xef, G: 0x44, B: 0x44, A: 0xff}, // #ef4444
		Button:       color.NRGBA{R: 0x31, G: 0x32, B: 0x44, A: 0xff}, // #313244
	}

	// LightPalette: Clean paper slate light theme
	LightPalette = ThemePalette{
		Name:         ThemeLight,
		Background:   color.NRGBA{R: 0xf8, G: 0xfa, B: 0xfc, A: 0xff}, // #f8fafc
		Card:         color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}, // #ffffff
		CardActive:   color.NRGBA{R: 0xf1, G: 0xf5, B: 0xf9, A: 0xff}, // #f1f5f9
		BorderSubtle: color.NRGBA{R: 0x94, G: 0xa3, B: 0xb8, A: 0xff}, // #94a3b8 (Slate-400 sắc sảo)
		BorderActive: color.NRGBA{R: 0x25, G: 0x63, B: 0xeb, A: 0xff}, // #2563eb
		Text:         color.NRGBA{R: 0x0f, G: 0x17, B: 0x2a, A: 0xff}, // #0f172a
		TextMuted:    color.NRGBA{R: 0x64, G: 0x74, B: 0x8b, A: 0xff}, // #64748b
		Primary:      color.NRGBA{R: 0x25, G: 0x63, B: 0xeb, A: 0xff}, // #2563eb
		Success:      color.NRGBA{R: 0x16, G: 0xa3, B: 0x4a, A: 0xff}, // #16a34a
		Warning:      color.NRGBA{R: 0xd9, G: 0x77, B: 0x06, A: 0xff}, // #d97706
		Danger:       color.NRGBA{R: 0xdc, G: 0x26, B: 0x26, A: 0xff}, // #dc2626
		Button:       color.NRGBA{R: 0xe2, G: 0xe8, B: 0xf0, A: 0xff}, // #e2e8f0
	}

	// WarmPalette: Warm sepia/amber theme for eye fatigue & blue-light reduction (UI-TR-07)
	WarmPalette = ThemePalette{
		Name:         ThemeWarm,
		Background:   color.NRGBA{R: 0x24, G: 0x1e, B: 0x17, A: 0xff}, // #241e17
		Card:         color.NRGBA{R: 0x33, G: 0x2a, B: 0x20, A: 0xff}, // #332a20
		CardActive:   color.NRGBA{R: 0x3d, G: 0x32, B: 0x26, A: 0xff}, // #3d3226
		BorderSubtle: color.NRGBA{R: 0x7d, G: 0x66, B: 0x4d, A: 0xff}, // #7d664d (Coffee Border)
		BorderActive: color.NRGBA{R: 0xd9, G: 0x77, B: 0x06, A: 0xff}, // #d97706 (Amber)
		Text:         color.NRGBA{R: 0xe8, G: 0xd8, B: 0xbe, A: 0xff}, // #e8d8be
		TextMuted:    color.NRGBA{R: 0xb3, G: 0x9f, B: 0x82, A: 0xff}, // #b39f82
		Primary:      color.NRGBA{R: 0xd9, G: 0x77, B: 0x06, A: 0xff}, // #d97706
		Success:      color.NRGBA{R: 0x84, G: 0xcc, B: 0x16, A: 0xff}, // #84cc16
		Warning:      color.NRGBA{R: 0xf5, G: 0x9e, B: 0x0b, A: 0xff}, // #f59e0b
		Danger:       color.NRGBA{R: 0xe1, G: 0x1d, B: 0x48, A: 0xff}, // #e11d48
		Button:       color.NRGBA{R: 0x42, G: 0x36, B: 0x29, A: 0xff}, // #423629
	}
)

// GetThemePalette returns the ThemePalette corresponding to themeName.
func GetThemePalette(themeName string) ThemePalette {
	switch strings.ToLower(strings.TrimSpace(themeName)) {
	case ThemeLight:
		return LightPalette
	case ThemeWarm:
		return WarmPalette
	default:
		return DarkPalette
	}
}

// DarkTheme satisfies fyne.Theme for Dark Mode.
type DarkTheme struct{}

var _ fyne.Theme = (*DarkTheme)(nil)

func (t *DarkTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return DarkPalette.Background
	case theme.ColorNameInputBackground:
		return DarkPalette.Card
	case theme.ColorNameButton:
		return DarkPalette.Button
	case theme.ColorNameForeground:
		return DarkPalette.Text
	case theme.ColorNamePrimary:
		return DarkPalette.Primary
	case theme.ColorNameSuccess:
		return DarkPalette.Success
	case theme.ColorNameWarning:
		return DarkPalette.Warning
	case theme.ColorNameError:
		return DarkPalette.Danger
	default:
		return theme.DefaultTheme().Color(name, theme.VariantDark)
	}
}

func (t *DarkTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *DarkTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *DarkTheme) Size(name fyne.ThemeSizeName) float32 {
	return compactThemeSize(name)
}

// LightTheme satisfies fyne.Theme for Light Mode.
type LightTheme struct{}

var _ fyne.Theme = (*LightTheme)(nil)

func (t *LightTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return LightPalette.Background
	case theme.ColorNameInputBackground:
		return LightPalette.Card
	case theme.ColorNameButton:
		return LightPalette.Button
	case theme.ColorNameForeground:
		return LightPalette.Text
	case theme.ColorNamePrimary:
		return LightPalette.Primary
	case theme.ColorNameSuccess:
		return LightPalette.Success
	case theme.ColorNameWarning:
		return LightPalette.Warning
	case theme.ColorNameError:
		return LightPalette.Danger
	default:
		return theme.DefaultTheme().Color(name, theme.VariantLight)
	}
}

func (t *LightTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *LightTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *LightTheme) Size(name fyne.ThemeSizeName) float32 {
	return compactThemeSize(name)
}

// WarmTheme satisfies fyne.Theme for Warm Mode.
type WarmTheme struct{}

var _ fyne.Theme = (*WarmTheme)(nil)

func (t *WarmTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return WarmPalette.Background
	case theme.ColorNameInputBackground:
		return WarmPalette.Card
	case theme.ColorNameButton:
		return WarmPalette.Button
	case theme.ColorNameForeground:
		return WarmPalette.Text
	case theme.ColorNamePrimary:
		return WarmPalette.Primary
	case theme.ColorNameSuccess:
		return WarmPalette.Success
	case theme.ColorNameWarning:
		return WarmPalette.Warning
	case theme.ColorNameError:
		return WarmPalette.Danger
	default:
		return theme.DefaultTheme().Color(name, theme.VariantDark)
	}
}

func (t *WarmTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *WarmTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *WarmTheme) Size(name fyne.ThemeSizeName) float32 {
	return compactThemeSize(name)
}

func compactThemeSize(name fyne.ThemeSizeName) float32 {
	switch name {
	case titleTextSizeName:
		return TitleTextSize
	case theme.SizeNameText, theme.SizeNameCaptionText:
		return BodyTextSize
	}
	return theme.DefaultTheme().Size(name)
}

// NewThemeInstance returns a new fyne.Theme matching the specified theme name.
func NewThemeInstance(themeName string) fyne.Theme {
	switch strings.ToLower(strings.TrimSpace(themeName)) {
	case ThemeLight:
		return &LightTheme{}
	case ThemeWarm:
		return &WarmTheme{}
	default:
		return &DarkTheme{}
	}
}

// LoadUIPreferences loads UI settings from disk. Returns default preferences if file is missing.
func LoadUIPreferences(path string) (*UIPreferences, error) {
	return preferences.LoadUIPreferences(path)
}

// SaveUIPreferences atomically writes UI preferences to disk.
func SaveUIPreferences(path string, pref *UIPreferences) error {
	return preferences.SaveUIPreferences(path, pref)
}
