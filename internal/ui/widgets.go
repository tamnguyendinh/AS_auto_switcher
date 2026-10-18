package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"antigravity-switch/internal/manager"
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

// emailLabelWithTooltip implements an ellipsis-truncated email label with desktop hover tooltip.
type emailLabelWithTooltip struct {
	widget.Label
	fullText string
	popup    *widget.PopUp
}

var _ desktop.Hoverable = (*emailLabelWithTooltip)(nil)

func newEmailLabelWithTooltip(email string, bold bool) *emailLabelWithTooltip {
	l := &emailLabelWithTooltip{fullText: email}
	l.Text = email
	l.Truncation = fyne.TextTruncateEllipsis
	l.Wrapping = fyne.TextWrapOff
	if bold {
		l.TextStyle = fyne.TextStyle{Bold: true}
	}
	l.ExtendBaseWidget(l)
	return l
}

func (l *emailLabelWithTooltip) MouseIn(e *desktop.MouseEvent) {
	if fyne.CurrentApp() == nil {
		return
	}
	drv := fyne.CurrentApp().Driver()
	if drv == nil {
		return
	}
	c := drv.CanvasForObject(l)
	if c == nil {
		return
	}
	tip := widget.NewLabel(l.fullText)
	l.popup = widget.NewPopUp(tip, c)
	l.popup.ShowAtPosition(e.AbsolutePosition.Add(fyne.NewPos(10, 10)))
}

func (l *emailLabelWithTooltip) MouseMoved(e *desktop.MouseEvent) {
	if l.popup != nil {
		l.popup.Move(e.AbsolutePosition.Add(fyne.NewPos(10, 10)))
	}
}

func (l *emailLabelWithTooltip) MouseOut() {
	if l.popup != nil {
		l.popup.Hide()
		l.popup = nil
	}
}

// NewAccountCard creates an individual card row for an account item using default DarkPalette.
func NewAccountCard(item manager.AccountDisplayItem, onSwitch func(email string), onSave func()) fyne.CanvasObject {
	return NewAccountCardWithQuota(item, nil, DarkPalette, onSwitch, onSave, nil)
}

// NewAccountCardWithQuota creates a standardized 3-row account card matching the Owner's specification:
// Line 1: Email (with truncation/tooltip) + Action Cluster: [✔ Dùng]/[Chuyển] + [🔄]
// Line 2: 5h: Claude <pct>% Gemini <pct>%
// Line 3: week: Claude <pct>% Gemini <pct>%
func NewAccountCardWithQuota(
	item manager.AccountDisplayItem,
	quota *telemetry.AccountQuotaData,
	palette ThemePalette,
	onSwitch func(email string),
	onSave func(),
	onRefreshSingle func(email string),
) fyne.CanvasObject {
	// Line 1: Email label with anti-break truncation and hover tooltip
	emailLabel := newEmailLabelWithTooltip(item.Email, item.IsActive)

	// Single Refresh Button [🔄]
	refreshBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		if onRefreshSingle != nil {
			onRefreshSingle(item.Email)
		}
	})
	refreshBtn.Importance = widget.LowImportance

	// Right Action Cluster
	var actionCluster fyne.CanvasObject
	if item.IsActive {
		activeBadge := widget.NewLabelWithStyle("[✔ Dùng]", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true})
		actionCluster = container.NewHBox(activeBadge, refreshBtn)
	} else {
		switchBtn := widget.NewButton("Chuyển", func() {
			if onSwitch != nil {
				onSwitch(item.Email)
			}
		})
		switchBtn.Importance = widget.HighImportance
		actionCluster = container.NewHBox(switchBtn, refreshBtn)
	}

	topRow := container.NewBorder(nil, nil, nil, actionCluster, emailLabel)

	// Extract Dual Cycle Quota (Claude and Gemini only, no GPT)
	dq := ExtractDualCycleQuota(quota)

	// Line 2: 5h Quota
	row5h := CreateQuotaCycleRow("5h:", dq.Claude5h, dq.Gemini5h, dq.HasData, dq.IsError)

	// Line 3: Weekly Quota
	rowWeek := CreateQuotaCycleRow("week:", dq.ClaudeWeek, dq.GeminiWeek, dq.HasData, dq.IsError)

	cardVBox := container.NewVBox(
		topRow,
		row5h,
		rowWeek,
	)

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
