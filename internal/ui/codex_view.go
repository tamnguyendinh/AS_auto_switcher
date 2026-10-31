package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"as-auto-switcher/internal/core/i18n"
	"as-auto-switcher/internal/engines/codex/manager"
	"as-auto-switcher/internal/engines/codex/model"
)

// codexView manages the Codex section UI: toolbar actions, empty placeholder, and account cards list.
type codexView struct {
	content         *fyne.Container
	add             *widget.Button
	refresh         *widget.Button
	pin             *widget.Button
	title           *widget.Label
	description     *widget.Label
	background      *canvas.Rectangle
	emptyContainer  *fyne.Container
	scrollContainer *container.Scroll
	cardContainer   *fyne.Container
	bodyStack       *fyne.Container

	currentPalette ThemePalette
	currentCards   []*manager.CodexCardView

	onAdd     func()
	onRefresh func()
	onSwitch  func(email string)
	onReauth  func(email string)
	onToggle  func(email string)
	onDelete  func(email string)
}

func newCodexView(palette ThemePalette, onPin func()) *codexView {
	v := &codexView{
		currentPalette: palette,
	}

	// 1. Toolbar buttons
	v.add = widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
		if v.onAdd != nil {
			v.onAdd()
		}
	})
	v.refresh = widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		if v.onRefresh != nil {
			v.onRefresh()
		}
	})
	v.add.Disable()
	v.refresh.Disable()

	v.pin = widget.NewButton("", onPin)
	toolbar := container.NewHBox(compactButton(v.add), compactButton(v.refresh), compactButton(v.pin))

	// 2. Empty placeholder message
	logo := canvas.NewImageFromResource(CodexLogoResource)
	logo.FillMode = canvas.ImageFillContain
	logo.SetMinSize(fyne.NewSize(64, 64))

	v.title = widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	v.description = widget.NewLabel("")
	v.description.Alignment = fyne.TextAlignCenter
	v.description.Wrapping = fyne.TextWrapWord

	v.background = canvas.NewRectangle(palette.Card)
	v.background.CornerRadius = 12
	v.background.StrokeWidth = 1
	v.background.StrokeColor = palette.BorderSubtle

	message := container.NewVBox(container.NewCenter(logo), v.title, v.description)
	v.emptyContainer = container.NewPadded(container.NewStack(
		v.background,
		container.NewPadded(container.New(&codexEmptyLayout{}, message)),
	))

	// 3. Card list container
	v.cardContainer = container.NewVBox()
	v.scrollContainer = container.NewVScroll(container.NewPadded(v.cardContainer))
	v.scrollContainer.Hide()

	// 4. Content body
	v.bodyStack = container.NewStack(v.emptyContainer, v.scrollContainer)
	v.content = container.NewBorder(toolbar, nil, nil, nil, v.bodyStack)

	v.setLanguage(i18n.Current())
	return v
}

// setToolbarEnabled toggles the enabled state of the Add and Refresh toolbar buttons.
func (v *codexView) setToolbarEnabled(enabled bool) {
	if enabled {
		v.add.Enable()
		v.refresh.Enable()
	} else {
		v.add.Disable()
		v.refresh.Disable()
	}
}

// setCallbacks configures action handlers for Codex operations.
func (v *codexView) setCallbacks(
	onAdd, onRefresh func(),
	onSwitch, onReauth, onToggle, onDelete func(string),
) {
	v.onAdd = onAdd
	v.onRefresh = onRefresh
	v.onSwitch = onSwitch
	v.onReauth = onReauth
	v.onToggle = onToggle
	v.onDelete = onDelete

	if onAdd != nil {
		v.add.Enable()
	}
	if onRefresh != nil {
		v.refresh.Enable()
	}
}

// renderCards projects the list of CodexCardView objects into reactive Card widgets.
func (v *codexView) renderCards(cards []*manager.CodexCardView, palette ThemePalette) {
	v.currentPalette = palette
	v.currentCards = cards
	v.cardContainer.Objects = nil

	if len(cards) == 0 {
		v.emptyContainer.Show()
		v.scrollContainer.Hide()
		v.bodyStack.Refresh()
		return
	}

	d := i18n.Current()
	for _, card := range cards {
		cardObj := v.buildCardObject(card, palette, d)
		v.cardContainer.Add(cardObj)
	}

	v.emptyContainer.Hide()
	v.scrollContainer.Show()
	v.cardContainer.Refresh()
	v.scrollContainer.Refresh()
	v.bodyStack.Refresh()
}

// buildCardObject constructs a single card adhering to Invariants 1, 2, 3 and Section 8.1.
func (v *codexView) buildCardObject(card *manager.CodexCardView, palette ThemePalette, d i18n.Dictionary) fyne.CanvasObject {
	cardBg := canvas.NewRectangle(palette.Card)
	cardBg.CornerRadius = 8
	cardBg.StrokeWidth = 1
	cardBg.StrokeColor = palette.BorderSubtle

	emailLbl := widget.NewLabelWithStyle(card.Email, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	emailLbl.Truncation = fyne.TextTruncateEllipsis

	var leftSlot fyne.CanvasObject = emailLbl
	if card.PlanType != "" && card.PlanType != "unknown" {
		planBadge := canvas.NewText("["+strings.ToUpper(card.PlanType)+"]", palette.TextMuted)
		planBadge.TextSize = 11
		planBadge.TextStyle = fyne.TextStyle{Bold: true}
		leftSlot = container.NewHBox(emailLbl, container.NewCenter(planBadge))
	}

	var actionSlot fyne.CanvasObject
	email := card.Email

	switch card.State {
	case model.StateActive:
		// [✔ Dùng] - Active label; NO Switch button; NO Delete button
		activeBadge := canvas.NewText(d.BadgeActive, palette.Success)
		activeBadge.TextSize = 12
		activeBadge.TextStyle = fyne.TextStyle{Bold: true}
		actionSlot = container.NewCenter(activeBadge)

	case model.StateReady, model.StateUnknown:
		// Manual switching preserves OFF eligibility.
		switchBtn := widget.NewButton(d.BadgeSwitch, func() {
			if v.onSwitch != nil {
				v.onSwitch(email)
			}
		})
		switchBtn.Importance = widget.HighImportance

		actionSlot = compactButton(switchBtn)

	case model.StateUnauth:
		// [Cần OAuth] - Reauth button; NO ON/OFF button
		reauthBtn := widget.NewButton(d.CodexBtnReauth, func() {
			if v.onReauth != nil {
				v.onReauth(email)
			}
		})
		reauthBtn.Importance = widget.WarningImportance
		actionSlot = compactButton(reauthBtn)

	default:
		actionSlot = widget.NewLabel(string(card.State))
	}

	var clusterObjects []fyne.CanvasObject
	clusterObjects = append(clusterObjects, actionSlot)

	// Status button [ON]/[OFF] - Strictly hidden for StateUnauth (Invariant 2)
	if card.CanToggle {
		statusLabel := "[ON]"
		if card.Status == "off" {
			statusLabel = "[OFF]"
		}
		toggleBtn := widget.NewButton(statusLabel, func() {
			if v.onToggle != nil {
				v.onToggle(email)
			}
		})
		toggleBtn.Importance = widget.LowImportance
		clusterObjects = append(clusterObjects, compactButton(toggleBtn))
	}

	// Delete button 🗑 - Strictly hidden for StateActive (Invariant 1)
	if card.CanDelete {
		deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
			if v.onDelete != nil {
				v.onDelete(email)
			}
		})
		deleteBtn.Importance = widget.LowImportance
		clusterObjects = append(clusterObjects, compactButton(deleteBtn))
	}

	cluster := container.NewHBox(clusterObjects...)
	cardContent := container.NewBorder(nil, nil, leftSlot, cluster)
	return container.NewStack(cardBg, container.NewPadded(cardContent))
}

func (v *codexView) setLanguage(d i18n.Dictionary) {
	v.add.SetText(d.CodexAddAccount)
	v.refresh.SetText(d.CodexRefresh)
	v.pin.SetText("📌 " + d.CodexPin)
	v.title.SetText(d.CodexTitle)
	v.description.SetText(d.CodexDescription)

	if len(v.currentCards) > 0 {
		v.renderCards(v.currentCards, v.currentPalette)
	}
}

func (v *codexView) applyTheme(palette ThemePalette) {
	v.currentPalette = palette
	v.background.FillColor = palette.Card
	v.background.StrokeColor = palette.BorderSubtle
	v.background.Refresh()

	if len(v.currentCards) > 0 {
		v.renderCards(v.currentCards, palette)
	}
}

// codexEmptyLayout centers the message with a bounded reading width; its minimum
// must not force the main window wider than the existing Antigravity toolbar.
type codexEmptyLayout struct{}

func (*codexEmptyLayout) MinSize([]fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(260, 180)
}

func (*codexEmptyLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, obj := range objects {
		width := min(size.Width, float32(460))
		obj.Resize(fyne.NewSize(width, size.Height))
		height := min(obj.MinSize().Height, size.Height)
		obj.Resize(fyne.NewSize(width, height))
		obj.Move(fyne.NewPos((size.Width-width)/2, (size.Height-height)/2))
	}
}
