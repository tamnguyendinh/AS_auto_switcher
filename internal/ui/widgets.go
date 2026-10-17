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
	"antigravity-switch/internal/telemetry"
)

var (
	// Legacy color variables preserved for backwards compatibility with tests and callers
	ColorDarkBg       = DarkPalette.Background
	ColorCard         = DarkPalette.Card
	ColorCardActive   = DarkPalette.CardActive
	ColorBorderSubtle = DarkPalette.BorderSubtle
	ColorBorderActive = DarkPalette.BorderActive
	ColorGreen        = DarkPalette.Success
	ColorYellow       = DarkPalette.Warning
	ColorBlue         = DarkPalette.Primary
	ColorMuted        = DarkPalette.TextMuted
	ColorText         = DarkPalette.Text
	ColorWhite        = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
)

// SwitcherTheme delegates to DarkTheme for backwards compatibility.
type SwitcherTheme struct {
	DarkTheme
}

var _ fyne.Theme = (*SwitcherTheme)(nil)

// NewStatusDot creates a circular status dot of a given color and diameter.
func NewStatusDot(c color.Color, diameter float32) fyne.CanvasObject {
	dot := canvas.NewCircle(c)
	return container.NewGridWrap(fyne.NewSize(diameter, diameter), dot)
}

// NewAccountCard creates an individual card row for an account item using default DarkPalette.
func NewAccountCard(item manager.AccountDisplayItem, onSwitch func(email string), onSave func()) fyne.CanvasObject {
	return NewAccountCardWithQuota(item, nil, DarkPalette, onSwitch, onSave, nil)
}

// NewAccountCardWithQuota creates an individual card row with mini quota badges and single refresh button.
func NewAccountCardWithQuota(
	item manager.AccountDisplayItem,
	quota *telemetry.AccountQuotaData,
	palette ThemePalette,
	onSwitch func(email string),
	onSave func(),
	onRefreshSingle func(email string),
) fyne.CanvasObject {
	// Status Dot
	var dotColor color.Color
	if item.IsActive || item.State == model.StateReady {
		dotColor = palette.Success
	} else {
		dotColor = palette.Warning
	}
	dot := NewStatusDot(dotColor, 12)

	// Email Label
	var emailLabel *widget.Label
	if item.IsActive {
		emailLabel = widget.NewLabelWithStyle(item.Email, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	} else {
		emailLabel = widget.NewLabel(item.Email)
	}

	infoVBox := container.NewVBox(emailLabel)
	if item.Name != "" {
		nameText := canvas.NewText(item.Name, palette.TextMuted)
		nameText.TextSize = 11
		infoVBox.Add(nameText)
	}

	leftHeader := container.NewHBox(
		container.NewCenter(dot),
		infoVBox,
	)

	// Row-level refresh button (UI-TR-02)
	refreshBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		if onRefreshSingle != nil {
			onRefreshSingle(item.Email)
		}
	})
	refreshBtn.Importance = widget.LowImportance

	// Right Action
	var actionWidget fyne.CanvasObject
	if item.IsActive {
		activeBadge := widget.NewLabelWithStyle("[ĐANG DÙNG]", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true})
		actionWidget = container.NewHBox(refreshBtn, activeBadge)
	} else if item.State == model.StateReady {
		switchBtn := widget.NewButtonWithIcon("⚡ Chuyển", theme.NavigateNextIcon(), func() {
			if onSwitch != nil {
				onSwitch(item.Email)
			}
		})
		switchBtn.Importance = widget.HighImportance
		actionWidget = container.NewHBox(refreshBtn, switchBtn)
	} else {
		// StateUnauth
		if item.IsActive {
			saveBtn := widget.NewButtonWithIcon("💾 Lưu", theme.DocumentSaveIcon(), func() {
				if onSave != nil {
					onSave()
				}
			})
			actionWidget = container.NewHBox(refreshBtn, saveBtn)
		} else {
			unauthBadge := widget.NewLabelWithStyle("[Chưa lưu]", fyne.TextAlignTrailing, fyne.TextStyle{Italic: true})
			actionWidget = container.NewHBox(refreshBtn, unauthBadge)
		}
	}

	topRow := container.NewBorder(nil, nil, leftHeader, actionWidget)

	// Mini Quota Badges (UI-TR-04)
	quotaBadges := NewQuotaBadgeGroup(quota)

	cardVBox := container.NewVBox(
		topRow,
		quotaBadges,
	)

	// Quota Recovery Countdown Text (UI-TR-05) if quota < 15%
	countdownLabel := NewQuotaCountdownLabel(quota)
	if countdownLabel != nil {
		cardVBox.Add(countdownLabel)
	}

	// Card Background with Elevation & Micro-Border adapted to current theme palette
	cardBg := canvas.NewRectangle(palette.Card)
	cardBg.CornerRadius = 8
	if item.IsActive {
		cardBg.FillColor = palette.CardActive
		cardBg.StrokeColor = palette.BorderActive
		cardBg.StrokeWidth = 1.5
	} else {
		cardBg.FillColor = palette.Card
		cardBg.StrokeColor = palette.BorderSubtle
		cardBg.StrokeWidth = 1.0
	}

	return container.NewStack(cardBg, container.NewPadded(cardVBox))
}
