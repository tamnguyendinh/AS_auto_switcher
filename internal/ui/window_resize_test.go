package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestResizeRectKeepsOppositeEdges(t *testing.T) {
	start := resizeRect{-900, -200, 100, 500}
	for _, tc := range []struct {
		name string
		edge resizeEdge
		want resizeRect
	}{
		{"left", resizeLeft, resizeRect{-850, -200, 100, 500}},
		{"right", resizeRight, resizeRect{-900, -200, 150, 500}},
		{"top", resizeTop, resizeRect{-900, -170, 100, 500}},
		{"bottom", resizeBottom, resizeRect{-900, -200, 100, 530}},
		{"top-left", resizeTop | resizeLeft, resizeRect{-850, -170, 100, 500}},
		{"top-right", resizeTop | resizeRight, resizeRect{-900, -170, 150, 500}},
		{"bottom-left", resizeBottom | resizeLeft, resizeRect{-850, -200, 100, 530}},
		{"bottom-right", resizeBottom | resizeRight, resizeRect{-900, -200, 150, 530}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := resizedWindowRect(start, tc.edge, resizePoint{50, 30}, 480, 360); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestResizeRectClampsWithoutMovingOppositeCorner(t *testing.T) {
	start := resizeRect{100, 100, 1100, 800}
	if got := resizedWindowRect(start, resizeLeft|resizeTop, resizePoint{2000, 2000}, 480, 360); got != (resizeRect{620, 440, 1100, 800}) {
		t.Fatal(got)
	}
	if got := resizedWindowRect(start, resizeRight|resizeBottom, resizePoint{-2000, -2000}, 480, 360); got != (resizeRect{100, 100, 580, 460}) {
		t.Fatal(got)
	}
	// Returning toward the original pointer is always based on the press position,
	// not on accumulated/clamped window coordinates.
	if got := resizedWindowRect(start, resizeLeft|resizeTop, resizePoint{}, 480, 360); got != start {
		t.Fatal(got)
	}
}

func TestResizeFrameReservesEdgesAndMinimumSize(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	win := a.NewWindow("resize frame")
	button := widget.NewButton("Close", nil)
	content := container.NewBorder(nil, nil, nil, button, widget.NewLabel("Content"))
	addWindowResizeFrame(win, content)
	win.SetContent(content)
	content.Resize(fyne.NewSize(920, 630))
	frame := content.Layout.(*windowResizeLayout)
	minimum := content.MinSize()
	if minimum.Width < MinWindowWidth || minimum.Height < MinWindowHeight {
		t.Fatalf("minimum too small: %v", minimum)
	}
	for _, object := range content.Objects[frame.contentCount:] {
		grip := object.(*windowResizeGrip)
		p, s := grip.Position(), grip.Size()
		if p.X < 0 || p.Y < 0 || p.X+s.Width > 920 || p.Y+s.Height > 630 {
			t.Fatalf("grip outside window: %v %v", p, s)
		}
		// No grip may enter the content rectangle, including corner wings.
		if p.X < 920-resizeBorder && p.X+s.Width > resizeBorder && p.Y < 630-resizeBorder && p.Y+s.Height > resizeBorder {
			t.Fatalf("grip overlaps content: %v %v", p, s)
		}
	}
}
