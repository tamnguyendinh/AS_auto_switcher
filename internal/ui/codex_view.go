package ui

import (
	"antigravity-switch/internal/core/i18n"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// codexView is an intentionally UI-only shell. It never calls an engine.
type codexView struct {
	content     *fyne.Container
	add         *widget.Button
	refresh     *widget.Button
	pin         *widget.Button
	title       *widget.Label
	description *widget.Label
	background  *canvas.Rectangle
}

func newCodexView(palette ThemePalette, onPin func()) *codexView {
	v := &codexView{}
	v.add = widget.NewButtonWithIcon("", theme.ContentAddIcon(), nil)
	v.refresh = widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), nil)
	v.add.Disable()
	v.refresh.Disable()
	v.pin = widget.NewButton("", onPin)
	toolbar := container.NewHBox(compactButton(v.add), compactButton(v.refresh), compactButton(v.pin))
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
	v.content = container.NewBorder(toolbar, nil, nil, nil,
		container.NewPadded(container.NewStack(v.background, container.NewPadded(container.New(&codexEmptyLayout{}, message)))))
	v.setLanguage(i18n.Current())
	return v
}

func (v *codexView) setLanguage(d i18n.Dictionary) {
	v.add.SetText(d.CodexAddAccount)
	v.refresh.SetText(d.CodexRefresh)
	v.pin.SetText("📌 " + d.CodexPin)
	v.title.SetText(d.CodexTitle)
	v.description.SetText(d.CodexDescription)
}

func (v *codexView) applyTheme(palette ThemePalette) {
	v.background.FillColor = palette.Card
	v.background.StrokeColor = palette.BorderSubtle
	v.background.Refresh()
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
