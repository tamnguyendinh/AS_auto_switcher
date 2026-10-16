package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"antigravity-switch/internal/manager"
	"antigravity-switch/internal/model"
)

var (
	// Catppuccin Mocha-inspired Dark Palette with Elevation Depth Tokens
	ColorDarkBg       = color.NRGBA{R: 0x11, G: 0x11, B: 0x1b, A: 0xff} // #11111b (Deep Crust Canvas)
	ColorCard         = color.NRGBA{R: 0x18, G: 0x18, B: 0x25, A: 0xff} // #181825 (Surface Mantle)
	ColorCardActive   = color.NRGBA{R: 0x1e, G: 0x1e, B: 0x2e, A: 0xff} // #1e1e2e (Surface Base Active)
	ColorBorderSubtle = color.NRGBA{R: 0x31, G: 0x32, B: 0x44, A: 0xff} // #313244 (Surface0 - Specular Highlight)
	ColorBorderActive = color.NRGBA{R: 0xa6, G: 0xe3, B: 0xa1, A: 0xdd} // #a6e3a1 (Green Ambient Accent)
	ColorGreen        = color.NRGBA{R: 0xa6, G: 0xe3, B: 0xa1, A: 0xff} // #a6e3a1
	ColorYellow       = color.NRGBA{R: 0xf9, G: 0xe2, B: 0xaf, A: 0xff} // #f9e2af
	ColorBlue         = color.NRGBA{R: 0x89, G: 0xb4, B: 0xfa, A: 0xff} // #89b4fa
	ColorMuted        = color.NRGBA{R: 0xa6, G: 0xad, B: 0xc8, A: 0xff} // #a6adc8
	ColorText         = color.NRGBA{R: 0xcd, G: 0xd6, B: 0xf4, A: 0xff} // #cdd6f4
	ColorWhite        = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
)

// SwitcherTheme implements a custom modern dark theme matching Antigravity aesthetics.
type SwitcherTheme struct{}

var _ fyne.Theme = (*SwitcherTheme)(nil)

func (t *SwitcherTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return ColorDarkBg
	case theme.ColorNameInputBackground:
		return ColorCard
	case theme.ColorNameButton:
		return color.NRGBA{R: 0x31, G: 0x32, B: 0x44, A: 0xff}
	case theme.ColorNameForeground:
		return ColorText
	case theme.ColorNamePrimary:
		return ColorBlue
	case theme.ColorNameSuccess:
		return ColorGreen
	case theme.ColorNameWarning:
		return ColorYellow
	default:
		return theme.DefaultTheme().Color(name, theme.VariantDark)
	}
}

func (t *SwitcherTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *SwitcherTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *SwitcherTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}

// NewStatusDot creates a circular status dot of a given color and diameter.
func NewStatusDot(c color.Color, diameter float32) fyne.CanvasObject {
	dot := canvas.NewCircle(c)
	return container.NewGridWrap(fyne.NewSize(diameter, diameter), dot)
}

// NewAccountCard creates an individual card row for an account item.
func NewAccountCard(item manager.AccountDisplayItem, onSwitch func(email string), onSave func()) fyne.CanvasObject {
	// Status Dot
	var dotColor color.Color
	if item.IsActive || item.State == model.StateReady {
		dotColor = ColorGreen
	} else {
		dotColor = ColorYellow
	}
	dot := NewStatusDot(dotColor, 12)

	// Center Info
	var emailLabel *widget.Label
	if item.IsActive {
		emailLabel = widget.NewLabelWithStyle(item.Email, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	} else {
		emailLabel = widget.NewLabel(item.Email)
	}

	infoVBox := container.NewVBox(emailLabel)
	if item.Name != "" {
		nameText := canvas.NewText(item.Name, ColorMuted)
		nameText.TextSize = 11
		infoVBox.Add(nameText)
	}

	leftSection := container.NewHBox(
		container.NewCenter(dot),
		infoVBox,
	)

	// Right Action
	var actionWidget fyne.CanvasObject
	if item.IsActive {
		activeBadge := widget.NewLabelWithStyle("[ĐANG DÙNG]", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true})
		actionWidget = activeBadge
	} else if item.State == model.StateReady {
		switchBtn := widget.NewButtonWithIcon("⚡ Chuyển", theme.NavigateNextIcon(), func() {
			if onSwitch != nil {
				onSwitch(item.Email)
			}
		})
		switchBtn.Importance = widget.HighImportance
		actionWidget = switchBtn
	} else {
		// StateUnauth
		if item.IsActive {
			saveBtn := widget.NewButtonWithIcon("💾 Lưu", theme.DocumentSaveIcon(), func() {
				if onSave != nil {
					onSave()
				}
			})
			actionWidget = saveBtn
		} else {
			unauthBadge := widget.NewLabelWithStyle("[Chưa lưu]", fyne.TextAlignTrailing, fyne.TextStyle{Italic: true})
			actionWidget = unauthBadge
		}
	}

	cardContent := container.NewBorder(nil, nil, leftSection, actionWidget)

	// Card Background with Elevation & Micro-Border
	cardBg := canvas.NewRectangle(ColorCard)
	cardBg.CornerRadius = 8
	if item.IsActive {
		cardBg.FillColor = ColorCardActive
		cardBg.StrokeColor = ColorBorderActive
		cardBg.StrokeWidth = 1.5
	} else {
		cardBg.FillColor = ColorCard
		cardBg.StrokeColor = ColorBorderSubtle
		cardBg.StrokeWidth = 1.0
	}

	return container.NewStack(cardBg, container.NewPadded(cardContent))
}
