package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// compactButton keeps the standard button renderer, text, width and input
// handling, while reducing only its height to 70% of the normal theme size.
func compactButton(button *widget.Button) *fyne.Container {
	return container.New(compactButtonLayout{}, button)
}

type compactButtonLayout struct{}

func (compactButtonLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	size := objects[0].MinSize()
	size.Height *= 0.7
	return size
}

func (layout compactButtonLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	height := min(layout.MinSize(objects).Height, size.Height)
	objects[0].Move(fyne.NewPos(0, (size.Height-height)/2))
	objects[0].Resize(fyne.NewSize(size.Width, height))
}
