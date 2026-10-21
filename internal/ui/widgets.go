package ui

import (
	"image/color"
	"regexp"
	"strings"

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

// switchTextButton represents a flat, clickable text button [Chuyển] styled consistently with [✔ Dùng].
type switchTextButton struct {
	widget.BaseWidget
	Text     string
	OnTapped func()
	hovered  bool
	palette  ThemePalette
	disabled bool

	bg   *canvas.Rectangle
	text *canvas.Text
}

var (
	_ fyne.Widget        = (*switchTextButton)(nil)
	_ fyne.Tappable      = (*switchTextButton)(nil)
	_ desktop.Hoverable  = (*switchTextButton)(nil)
	_ desktop.Cursorable = (*switchTextButton)(nil)
)

func newSwitchTextButton(onTapped func(), palette ThemePalette) *switchTextButton {
	b := &switchTextButton{
		Text:     "[Chuyển]",
		OnTapped: onTapped,
		palette:  palette,
	}
	b.bg = canvas.NewRectangle(color.Transparent)
	b.bg.CornerRadius = 3
	b.text = canvas.NewText("[Chuyển]", palette.Primary)
	b.text.TextSize = 10
	b.text.TextStyle = fyne.TextStyle{Bold: true}
	b.text.Alignment = fyne.TextAlignCenter
	b.ExtendBaseWidget(b)
	return b
}

func newDisabledSwitchTextButton(palette ThemePalette) *switchTextButton {
	b := &switchTextButton{
		Text:     "[Chuyển]",
		palette:  palette,
		disabled: true,
	}
	b.bg = canvas.NewRectangle(color.Transparent)
	b.bg.CornerRadius = 3
	b.text = canvas.NewText("[Chuyển]", palette.TextMuted)
	b.text.TextSize = 10
	b.text.TextStyle = fyne.TextStyle{Bold: true}
	b.text.Alignment = fyne.TextAlignCenter
	b.ExtendBaseWidget(b)
	return b
}

func (b *switchTextButton) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewStack(b.bg, container.NewPadded(b.text))
	return widget.NewSimpleRenderer(c)
}

func (b *switchTextButton) Refresh() {
	if b.disabled {
		if b.text != nil {
			if b.Text != "" {
				b.text.Text = b.Text
			}
			b.text.Color = b.palette.TextMuted
			b.text.Refresh()
		}
		if b.bg != nil {
			b.bg.FillColor = color.Transparent
			b.bg.Refresh()
		}
		b.BaseWidget.Refresh()
		return
	}

	if b.text != nil {
		if b.Text != "" {
			b.text.Text = b.Text
		}
		if b.hovered {
			b.text.Color = lightenColor(b.palette.Primary, 35)
		} else {
			b.text.Color = b.palette.Primary
		}
		b.text.Refresh()
	}
	if b.bg != nil {
		if b.hovered {
			b.bg.FillColor = color.NRGBA{
				R: b.palette.Primary.R,
				G: b.palette.Primary.G,
				B: b.palette.Primary.B,
				A: 0x24,
			}
		} else {
			b.bg.FillColor = color.Transparent
		}
		b.bg.Refresh()
	}
	b.BaseWidget.Refresh()
}

func (b *switchTextButton) Tapped(*fyne.PointEvent) {
	if b.disabled {
		return
	}
	if b.OnTapped != nil {
		b.OnTapped()
	}
}

func (b *switchTextButton) MouseIn(*desktop.MouseEvent) {
	if b.disabled {
		return
	}
	b.hovered = true
	b.Refresh()
}

func (b *switchTextButton) MouseOut() {
	if b.disabled {
		return
	}
	b.hovered = false
	b.Refresh()
}

func (b *switchTextButton) MouseMoved(*desktop.MouseEvent) {}

func (b *switchTextButton) Cursor() desktop.Cursor {
	if b.disabled {
		return desktop.DefaultCursor
	}
	return desktop.PointerCursor
}

func (b *switchTextButton) IsDisabled() bool {
	return b.disabled
}

// statusTextButton represents a flat text toggle button [ON] / [OFF].
type statusTextButton struct {
	widget.BaseWidget
	Status   string // "on" hoặc "off"
	OnTapped func()
	hovered  bool
	palette  ThemePalette

	bg   *canvas.Rectangle
	text *canvas.Text
}

var (
	_ fyne.Widget        = (*statusTextButton)(nil)
	_ fyne.Tappable      = (*statusTextButton)(nil)
	_ desktop.Hoverable  = (*statusTextButton)(nil)
	_ desktop.Cursorable = (*statusTextButton)(nil)
)

func newStatusTextButton(status string, onTapped func(), palette ThemePalette) *statusTextButton {
	norm := strings.ToLower(strings.TrimSpace(status))
	if norm != "off" {
		norm = "on"
	}
	b := &statusTextButton{
		Status:   norm,
		OnTapped: onTapped,
		palette:  palette,
	}
	b.bg = canvas.NewRectangle(color.Transparent)
	b.bg.CornerRadius = 3

	label := "[ON]"
	txtColor := palette.Success
	if norm == "off" {
		label = "[OFF]"
		txtColor = palette.TextMuted
	}

	b.text = canvas.NewText(label, txtColor)
	b.text.TextSize = 10
	b.text.TextStyle = fyne.TextStyle{Bold: true}
	b.text.Alignment = fyne.TextAlignCenter
	b.ExtendBaseWidget(b)
	return b
}

func (b *statusTextButton) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewStack(b.bg, container.NewPadded(b.text))
	return widget.NewSimpleRenderer(c)
}

func (b *statusTextButton) Refresh() {
	norm := strings.ToLower(strings.TrimSpace(b.Status))
	if norm != "off" {
		norm = "on"
	}
	b.Status = norm

	txtColor := b.palette.Success
	label := "[ON]"
	if norm == "off" {
		txtColor = b.palette.TextMuted
		label = "[OFF]"
	}

	if b.text != nil {
		b.text.Text = label
		if b.hovered {
			b.text.Color = lightenColor(txtColor, 35)
		} else {
			b.text.Color = txtColor
		}
		b.text.Refresh()
	}
	if b.bg != nil {
		if b.hovered {
			b.bg.FillColor = color.NRGBA{
				R: txtColor.R,
				G: txtColor.G,
				B: txtColor.B,
				A: 0x24,
			}
		} else {
			b.bg.FillColor = color.Transparent
		}
		b.bg.Refresh()
	}
	b.BaseWidget.Refresh()
}

func (b *statusTextButton) SetStatus(status string) {
	norm := strings.ToLower(strings.TrimSpace(status))
	if norm != "off" {
		norm = "on"
	}
	b.Status = norm
	b.Refresh()
}

func (b *statusTextButton) SetPalette(palette ThemePalette) {
	b.palette = palette
	b.Refresh()
}

func (b *statusTextButton) Tapped(*fyne.PointEvent) {
	if b.OnTapped != nil {
		b.OnTapped()
	}
}

func (b *statusTextButton) MouseIn(*desktop.MouseEvent) {
	b.hovered = true
	b.Refresh()
}

func (b *statusTextButton) MouseOut() {
	b.hovered = false
	b.Refresh()
}

func (b *statusTextButton) MouseMoved(*desktop.MouseEvent) {}

func (b *statusTextButton) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

func (b *statusTextButton) LabelText() string {
	if b == nil || b.text == nil {
		return ""
	}
	return b.text.Text
}

func lightenColor(c color.NRGBA, delta int) color.NRGBA {
	clamp := func(v int) uint8 {
		if v > 255 {
			return 255
		}
		if v < 0 {
			return 0
		}
		return uint8(v)
	}
	return color.NRGBA{
		R: clamp(int(c.R) + delta),
		G: clamp(int(c.G) + delta),
		B: clamp(int(c.B) + delta),
		A: c.A,
	}
}

// MaskEmail transforms an email like "user@domain.com" into "u***@domain.com".
// If email format is invalid or has no @, original string is returned unchanged.
func MaskEmail(email string) string {
	trimmed := strings.TrimSpace(email)
	atIdx := strings.Index(trimmed, "@")
	if atIdx <= 0 {
		return trimmed
	}
	username := trimmed[:atIdx]
	domain := trimmed[atIdx:]
	runes := []rune(username)
	if len(runes) == 0 {
		return trimmed
	}
	return string(runes[0]) + "***" + domain
}

var emailPattern = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)

// maskMessageEmails scans a string for email addresses and replaces them with their masked forms.
func maskMessageEmails(msg string) string {
	if !strings.Contains(msg, "@") {
		return msg
	}
	return emailPattern.ReplaceAllStringFunc(msg, MaskEmail)
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
	Email          string
	Status         string
	IsActive       bool
	HasToken       bool
	Root           *fyne.Container
	cardBg         *canvas.Rectangle
	emailLabel     *emailLabelWithTooltip
	actionCluster  fyne.CanvasObject
	actionSlot     *fyne.Container
	refreshButton  *widget.Button
	statusButton   *statusTextButton
	onSwitch       func(string)
	onToggleStatus func(string)

	gemini5hPct    *canvas.Text
	gemini5hBar    *canvas.Rectangle
	gemini5hTime   *canvas.Text
	geminiWeekPct  *canvas.Text
	geminiWeekBar  *canvas.Rectangle
	geminiWeekTime *canvas.Text
	claude5hPct    *canvas.Text
	claude5hBar    *canvas.Rectangle
	claude5hTime   *canvas.Text
	claudeWeekPct  *canvas.Text
	claudeWeekBar  *canvas.Rectangle
	claudeWeekTime *canvas.Text

	gemini5hCell   *matrixQuotaCell
	geminiWeekCell *matrixQuotaCell
	claude5hCell   *matrixQuotaCell
	claudeWeekCell *matrixQuotaCell
}

// StatusButton returns the status toggle button canvas object.
func (w *AccountCardWidget) StatusButton() fyne.CanvasObject {
	if w == nil {
		return nil
	}
	return w.statusButton
}

// ActionButton returns the current canvas object inside actionSlot.
func (w *AccountCardWidget) ActionButton() fyne.CanvasObject {
	if w == nil || w.actionSlot == nil || len(w.actionSlot.Objects) == 0 {
		return nil
	}
	return w.actionSlot.Objects[0]
}

// RefreshButton returns the single-card refresh button.
func (w *AccountCardWidget) RefreshButton() *widget.Button {
	if w == nil {
		return nil
	}
	return w.refreshButton
}

// CanvasObject returns the root CanvasObject for this account card widget.
func (w *AccountCardWidget) CanvasObject() fyne.CanvasObject {
	if w == nil {
		return nil
	}
	return w.Root
}

// SetMasked toggles the email label between masked and full text in-place.
func (w *AccountCardWidget) SetMasked(masked bool) {
	if w == nil || w.emailLabel == nil {
		return
	}
	if masked {
		w.emailLabel.SetText(MaskEmail(w.Email))
	} else {
		w.emailLabel.SetText(w.Email)
	}
}

// EmailLabelText returns the currently displayed text of the email label.
func (w *AccountCardWidget) EmailLabelText() string {
	if w == nil || w.emailLabel == nil {
		return ""
	}
	return w.emailLabel.Text
}

// EmailFullText returns the underlying full email preserved for hover tooltip.
func (w *AccountCardWidget) EmailFullText() string {
	if w == nil || w.emailLabel == nil {
		return ""
	}
	return w.emailLabel.fullText
}

// UpdateQuota performs in-place reactive updates of Quota percentage labels and progress bars
// without rebuilding or destroying Canvas tree objects, preventing layout jank and window resizing.
func (w *AccountCardWidget) UpdateQuota(quota *telemetry.AccountQuotaData, palette ThemePalette) {
	if w == nil {
		return
	}
	if !w.IsActive && !w.HasToken {
		for _, cell := range []*matrixQuotaCell{w.gemini5hCell, w.geminiWeekCell, w.claude5hCell, w.claudeWeekCell} {
			if cell != nil {
				cell.setUnavailable()
			}
		}
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

// UpdateState updates the existing card's actions without replacing its quota
// cells or email label (including the current privacy masking state).
func (w *AccountCardWidget) UpdateState(item manager.AccountDisplayItem, palette ThemePalette) {
	if w == nil || w.actionSlot == nil {
		return
	}
	normStatus := strings.ToLower(strings.TrimSpace(item.Status))
	if normStatus != "off" {
		normStatus = "on"
	}
	if w.IsActive == item.IsActive && w.HasToken == item.HasToken && w.Status == normStatus && len(w.actionSlot.Objects) > 0 {
		return
	}
	w.IsActive, w.HasToken, w.Status = item.IsActive, item.HasToken, normStatus

	// Update statusButton visibility and state
	if w.statusButton != nil {
		if !w.HasToken && !w.IsActive {
			// Thẻ [Cần OAuth]: Tuyệt đối không hiển thị nút [ON/OFF]
			w.statusButton.Hide()
		} else {
			// Thẻ có token hoặc đang active: hiển thị [ON] hoặc [OFF] tương ứng
			w.statusButton.SetPalette(palette)
			w.statusButton.SetStatus(w.Status)
			w.statusButton.Show()
		}
	}

	var action fyne.CanvasObject
	if w.IsActive || !w.HasToken {
		text, tint := "[Cần OAuth]", palette.Warning
		if w.IsActive {
			text, tint = "[✔ Dùng]", palette.Primary
		}
		badge := canvas.NewText(text, tint)
		badge.TextSize = BodyTextSize
		badge.TextStyle = fyne.TextStyle{Bold: true}
		badge.Alignment = fyne.TextAlignTrailing
		action = container.NewCenter(badge)
	} else {
		if w.Status == "off" {
			// Khi w.Status == "off": Nút [Chuyển] bị làm mờ (màu chữ palette.TextMuted, desktop.DefaultCursor, click no-op)
			action = newDisabledSwitchTextButton(palette)
		} else {
			// Khi w.Status == "on": Nút [Chuyển] hoạt động bình thường
			action = newSwitchTextButton(func() {
				if w.HasToken && !w.IsActive && w.Status != "off" && w.onSwitch != nil {
					w.onSwitch(w.Email)
				}
			}, palette)
		}
	}
	w.actionSlot.Objects = []fyne.CanvasObject{action}
	w.actionSlot.Refresh()

	// Update refreshButton
	if w.refreshButton != nil {
		if (w.IsActive || w.HasToken) && w.Status != "off" {
			w.refreshButton.Enable()
		} else {
			w.refreshButton.Disable()
		}
	}

	if w.emailLabel != nil {
		w.emailLabel.TextStyle.Bold = w.IsActive
		w.emailLabel.Refresh()
	}
	if w.cardBg != nil {
		w.cardBg.FillColor, w.cardBg.StrokeColor, w.cardBg.StrokeWidth = palette.Card, palette.BorderSubtle, 1.8
		if w.IsActive {
			w.cardBg.FillColor, w.cardBg.StrokeColor, w.cardBg.StrokeWidth = palette.CardActive, palette.BorderActive, 2
		}
		w.cardBg.Refresh()
	}
}

// NewAccountCardWidgetWithQuota creates a card with in-place state and quota updates.
func NewAccountCardWidgetWithQuota(
	item manager.AccountDisplayItem,
	quota *telemetry.AccountQuotaData,
	palette ThemePalette,
	onSwitch func(email string),
	onRefreshSingle func(email string),
	onToggleStatus ...func(email string),
) *AccountCardWidget {
	var toggleFn func(email string)
	if len(onToggleStatus) > 0 {
		toggleFn = onToggleStatus[0]
	}

	st := strings.ToLower(strings.TrimSpace(item.Status))
	if st != "off" {
		st = "on"
	}

	w := &AccountCardWidget{
		Email:          item.Email,
		Status:         st,
		onSwitch:       onSwitch,
		onToggleStatus: toggleFn,
	}
	emailLabel := newEmailLabelWithTooltip(item.Email, item.IsActive)
	refreshBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		if (w.IsActive || w.HasToken) && w.Status != "off" && onRefreshSingle != nil {
			onRefreshSingle(w.Email)
		}
	})
	refreshBtn.Importance = widget.LowImportance

	statusBtn := newStatusTextButton(st, func() {
		if w.onToggleStatus != nil {
			w.onToggleStatus(w.Email)
		}
	}, palette)

	actionSlot := container.NewStack()
	actionCluster := container.NewHBox(actionSlot, compactButton(refreshBtn), statusBtn)
	topRow := container.NewBorder(nil, nil, nil, actionCluster, emailLabel)

	dq := ExtractDualCycleQuota(quota)
	matrixTable, cGemini5h, cGeminiWeek, cClaude5h, cClaudeWeek := createMatrixQuotaTableWithCells(dq, palette)
	cardVBox := container.NewVBox(topRow, matrixTable)
	cardBg := canvas.NewRectangle(palette.Card)
	cardBg.CornerRadius = 8
	root := container.NewStack(cardBg, container.NewPadded(cardVBox))

	w.Root, w.cardBg, w.emailLabel = root, cardBg, emailLabel
	w.actionCluster, w.actionSlot, w.refreshButton, w.statusButton = actionCluster, actionSlot, refreshBtn, statusBtn
	w.gemini5hCell, w.geminiWeekCell = cGemini5h, cGeminiWeek
	w.claude5hCell, w.claudeWeekCell = cClaude5h, cClaudeWeek
	w.gemini5hPct, w.gemini5hBar, w.gemini5hTime = cGemini5h.pctText, cGemini5h.fillBar, cGemini5h.timeText
	w.geminiWeekPct, w.geminiWeekBar, w.geminiWeekTime = cGeminiWeek.pctText, cGeminiWeek.fillBar, cGeminiWeek.timeText
	w.claude5hPct, w.claude5hBar, w.claude5hTime = cClaude5h.pctText, cClaude5h.fillBar, cClaude5h.timeText
	w.claudeWeekPct, w.claudeWeekBar, w.claudeWeekTime = cClaudeWeek.pctText, cClaudeWeek.fillBar, cClaudeWeek.timeText
	w.UpdateState(item, palette)
	w.UpdateQuota(quota, palette)
	return w
}

// NewAccountCardWithQuota wraps the reactive card for callers needing only its root.
func NewAccountCardWithQuota(
	item manager.AccountDisplayItem,
	quota *telemetry.AccountQuotaData,
	palette ThemePalette,
	onSwitch func(email string),
	onRefreshSingle func(email string),
	onToggleStatus ...func(email string),
) fyne.CanvasObject {
	return NewAccountCardWidgetWithQuota(item, quota, palette, onSwitch, onRefreshSingle, onToggleStatus...).Root
}
