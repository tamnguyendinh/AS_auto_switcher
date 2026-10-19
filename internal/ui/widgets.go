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

// AccountCardWidget represents an account card with direct references to child widgets for in-place reactive updates.
type AccountCardWidget struct {
	Email         string
	IsActive      bool
	Root          *fyne.Container
	cardBg        *canvas.Rectangle
	emailLabel    *emailLabelWithTooltip
	actionCluster fyne.CanvasObject

	gemini5hPct   *canvas.Text
	gemini5hBar   *canvas.Rectangle
	gemini5hTime  *canvas.Text
	geminiWeekPct *canvas.Text
	geminiWeekBar *canvas.Rectangle
	geminiWeekTime *canvas.Text
	claude5hPct   *canvas.Text
	claude5hBar   *canvas.Rectangle
	claude5hTime  *canvas.Text
	claudeWeekPct *canvas.Text
	claudeWeekBar *canvas.Rectangle
	claudeWeekTime *canvas.Text

	gemini5hCell   *matrixQuotaCell
	geminiWeekCell *matrixQuotaCell
	claude5hCell   *matrixQuotaCell
	claudeWeekCell *matrixQuotaCell
}

// CanvasObject returns the root CanvasObject for this account card widget.
func (w *AccountCardWidget) CanvasObject() fyne.CanvasObject {
	if w == nil {
		return nil
	}
	return w.Root
}

// UpdateQuota performs in-place reactive updates of Quota percentage labels and progress bars
// without rebuilding or destroying Canvas tree objects, preventing layout jank and window resizing.
func (w *AccountCardWidget) UpdateQuota(quota *telemetry.AccountQuotaData, palette ThemePalette) {
	if w == nil {
		return
	}
	dq := ExtractDualCycleQuota(quota)

	hasGemini5h := !dq.IsError && dq.HasData && dq.Gemini5h >= 0
	if w.gemini5hCell != nil {
		w.gemini5hCell.update(dq.Gemini5h, dq.Gemini5hReset, hasGemini5h)
	}

	hasGeminiWeek := !dq.IsError && dq.HasData && dq.GeminiWeek >= 0
	if w.geminiWeekCell != nil {
		w.geminiWeekCell.update(dq.GeminiWeek, dq.GeminiWeekReset, hasGeminiWeek)
	}

	hasClaude5h := !dq.IsError && dq.HasData && dq.Claude5h >= 0
	if w.claude5hCell != nil {
		w.claude5hCell.update(dq.Claude5h, dq.Claude5hReset, hasClaude5h)
	}

	hasClaudeWeek := !dq.IsError && dq.HasData && dq.ClaudeWeek >= 0
	if w.claudeWeekCell != nil {
		w.claudeWeekCell.update(dq.ClaudeWeek, dq.ClaudeWeekReset, hasClaudeWeek)
	}
}

// NewAccountCardWidgetWithQuota creates an AccountCardWidget with direct widget references for in-place reactive updates.
func NewAccountCardWidgetWithQuota(
	item manager.AccountDisplayItem,
	quota *telemetry.AccountQuotaData,
	palette ThemePalette,
	onSwitch func(email string),
	onSave func(),
	onRefreshSingle func(email string),
) *AccountCardWidget {
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
		actionCluster = container.NewHBox(compactButton(switchBtn), refreshBtn)
	}

	topRow := container.NewBorder(nil, nil, nil, actionCluster, emailLabel)

	// Extract Dual Cycle Quota (Claude and Gemini only, no GPT)
	dq := ExtractDualCycleQuota(quota)

	// 3x3 Matrix Quota Table with direct cell references
	matrixTable, cGemini5h, cGeminiWeek, cClaude5h, cClaudeWeek := createMatrixQuotaTableWithCells(dq, palette)

	cardVBox := container.NewVBox(
		topRow,
		matrixTable,
	)

	// Card Background with Elevation & Micro-Border adapted to current theme palette
	cardBg := canvas.NewRectangle(palette.Card)
	cardBg.CornerRadius = 8
	if item.IsActive {
		cardBg.FillColor = palette.CardActive
		cardBg.StrokeColor = palette.BorderActive
		cardBg.StrokeWidth = 2.0
	} else {
		cardBg.FillColor = palette.Card
		cardBg.StrokeColor = palette.BorderSubtle
		cardBg.StrokeWidth = 1.8
	}

	root := container.NewStack(cardBg, container.NewPadded(cardVBox))

	return &AccountCardWidget{
		Email:         item.Email,
		IsActive:      item.IsActive,
		Root:          root,
		cardBg:        cardBg,
		emailLabel:    emailLabel,
		actionCluster: actionCluster,

		gemini5hPct:   cGemini5h.pctText,
		gemini5hBar:   cGemini5h.fillBar,
		gemini5hTime:  cGemini5h.timeText,
		geminiWeekPct: cGeminiWeek.pctText,
		geminiWeekBar: cGeminiWeek.fillBar,
		geminiWeekTime: cGeminiWeek.timeText,
		claude5hPct:   cClaude5h.pctText,
		claude5hBar:   cClaude5h.fillBar,
		claude5hTime:  cClaude5h.timeText,
		claudeWeekPct: cClaudeWeek.pctText,
		claudeWeekBar: cClaudeWeek.fillBar,
		claudeWeekTime: cClaudeWeek.timeText,

		gemini5hCell:   cGemini5h,
		geminiWeekCell: cGeminiWeek,
		claude5hCell:   cClaude5h,
		claudeWeekCell: cClaudeWeek,
	}
}

// NewAccountCardWithQuota creates a standardized account card matching the Owner's specification:
// Line 1: Email (with truncation/tooltip) + Action Cluster: [✔ Dùng]/[Chuyển] + [🔄]
// Line 2: 3x3 Matrix Table: Quota | 5h | Week with Gemini and Claude rows separated by 1px vertical borders.
func NewAccountCardWithQuota(
	item manager.AccountDisplayItem,
	quota *telemetry.AccountQuotaData,
	palette ThemePalette,
	onSwitch func(email string),
	onSave func(),
	onRefreshSingle func(email string),
) fyne.CanvasObject {
	w := NewAccountCardWidgetWithQuota(item, quota, palette, onSwitch, onSave, onRefreshSingle)
	return w.Root
}
