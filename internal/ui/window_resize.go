package ui

import (
	"image/color"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

type resizeEdge uint8

const (
	resizeLeft resizeEdge = 1 << iota
	resizeRight
	resizeTop
	resizeBottom
	resizeBorder float32 = 6
	resizeCorner float32 = 12
)

// Native coordinates remain in physical screen pixels, including negative
// coordinates on secondary monitors. Only the minimum size needs Fyne scaling.
type resizeRect struct{ left, top, right, bottom int32 }
type resizePoint struct{ x, y int32 }

func resizedWindowRect(start resizeRect, edge resizeEdge, delta resizePoint, minWidth, minHeight int32) resizeRect {
	rect := start
	if edge&resizeLeft != 0 {
		rect.left = min(start.left+delta.x, start.right-minWidth)
	}
	if edge&resizeRight != 0 {
		rect.right = max(start.right+delta.x, start.left+minWidth)
	}
	if edge&resizeTop != 0 {
		rect.top = min(start.top+delta.y, start.bottom-minHeight)
	}
	if edge&resizeBottom != 0 {
		rect.bottom = max(start.bottom+delta.y, start.top+minHeight)
	}
	return rect
}

type windowResizeDrag struct {
	win         fyne.Window
	active      bool
	edge        resizeEdge
	start, last resizeRect
	pointer     resizePoint
}

func (d *windowResizeDrag) begin(edge resizeEdge) {
	d.active = false
	if IsNativeWindowMaximized() {
		return
	}
	rect, pointer, ok := nativeResizeState()
	if !ok {
		return
	}
	d.start, d.last, d.pointer, d.edge, d.active = rect, rect, pointer, edge, true
}

func (d *windowResizeDrag) move() {
	if !d.active {
		return
	}
	if IsNativeWindowMaximized() || !nativeResizeCaptured() {
		d.active = false
		return
	}
	_, pointer, ok := nativeResizeState()
	if !ok {
		d.active = false
		return
	}
	minimum := d.win.Content().MinSize()
	scale := d.win.Canvas().Scale()
	rect := resizedWindowRect(d.start, d.edge,
		resizePoint{pointer.x - d.pointer.x, pointer.y - d.pointer.y},
		int32(math.Ceil(float64(minimum.Width*scale))),
		int32(math.Ceil(float64(minimum.Height*scale))))
	if rect != d.last {
		if !applyNativeResizeRect(rect) {
			d.active = false
			return
		}
		d.last = rect
	}
}

type windowResizeGrip struct {
	widget.BaseWidget
	drag *windowResizeDrag
	edge resizeEdge
}

func newWindowResizeGrip(drag *windowResizeDrag, edge resizeEdge) *windowResizeGrip {
	g := &windowResizeGrip{drag: drag, edge: edge}
	g.ExtendBaseWidget(g)
	return g
}

func (g *windowResizeGrip) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(canvas.NewRectangle(color.Transparent))
}
func (g *windowResizeGrip) MinSize() fyne.Size { return fyne.Size{} }
func (g *windowResizeGrip) MouseDown(e *desktop.MouseEvent) {
	if e.Button == desktop.MouseButtonPrimary {
		g.drag.begin(g.edge)
	}
}
func (g *windowResizeGrip) MouseUp(*desktop.MouseEvent) { g.drag.active = false }
func (g *windowResizeGrip) Dragged(*fyne.DragEvent)     { g.drag.move() }
func (g *windowResizeGrip) DragEnd()                    { g.drag.active = false }
func (g *windowResizeGrip) Cursor() desktop.Cursor {
	if IsNativeWindowMaximized() {
		return desktop.DefaultCursor
	}
	switch g.edge {
	case resizeLeft, resizeRight:
		return desktop.HResizeCursor
	case resizeTop, resizeBottom:
		return desktop.VResizeCursor
	case resizeLeft | resizeTop, resizeRight | resizeBottom:
		return desktop.NWSEResizeCursor
	default:
		return desktop.NESWResizeCursor
	}
}

// Keep the existing content tree intact. Twelve rectangles form eight resize
// directions: each corner is L-shaped so it never covers a title button.
func addWindowResizeFrame(win fyne.Window, content *fyne.Container) {
	drag := &windowResizeDrag{win: win}
	frame := &windowResizeLayout{inner: content.Layout, contentCount: len(content.Objects), drag: drag}
	for _, edge := range []resizeEdge{
		resizeLeft, resizeRight, resizeTop, resizeBottom,
		resizeLeft | resizeTop, resizeLeft | resizeTop,
		resizeRight | resizeTop, resizeRight | resizeTop,
		resizeLeft | resizeBottom, resizeLeft | resizeBottom,
		resizeRight | resizeBottom, resizeRight | resizeBottom,
	} {
		content.Objects = append(content.Objects, newWindowResizeGrip(drag, edge))
	}
	content.Layout = frame
	win.SetPadded(false) // The frame owns the padding and reaches the client edges.
}

type windowResizeLayout struct {
	inner        fyne.Layout
	contentCount int
	drag         *windowResizeDrag
}

func (l *windowResizeLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	minimum := l.inner.MinSize(objects[:l.contentCount]).Add(fyne.NewSquareSize(2 * resizeBorder))
	return fyne.NewSize(max(MinWindowWidth, minimum.Width), max(MinWindowHeight, minimum.Height))
}

func (l *windowResizeLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	inner := objects[:l.contentCount]
	l.inner.Layout(inner, fyne.NewSize(max(0, size.Width-2*resizeBorder), max(0, size.Height-2*resizeBorder)))
	for _, object := range inner {
		object.Move(object.Position().Add(fyne.NewPos(resizeBorder, resizeBorder)))
	}
	grips := objects[l.contentCount:]
	if IsNativeWindowMaximized() {
		l.drag.active = false
		for _, grip := range grips {
			grip.Resize(fyne.Size{})
		}
		return
	}
	w, h, b, c := size.Width, size.Height, resizeBorder, resizeCorner
	boxes := [][4]float32{
		{0, c, b, h - 2*c}, {w - b, c, b, h - 2*c}, {c, 0, w - 2*c, b}, {c, h - b, w - 2*c, b},
		{0, 0, c, b}, {0, b, b, c - b}, {w - c, 0, c, b}, {w - b, b, b, c - b},
		{0, h - b, c, b}, {0, h - c, b, c - b}, {w - c, h - b, c, b}, {w - b, h - c, b, c - b},
	}
	for i, box := range boxes {
		grips[i].Move(fyne.NewPos(box[0], box[1]))
		grips[i].Resize(fyne.NewSize(max(0, box[2]), max(0, box[3])))
	}
}
