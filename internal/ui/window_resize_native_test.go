//go:build windows && gui && nativeui

package ui

import (
	"math"
	"testing"
	"time"
	"unsafe"

	"fyne.io/fyne/v2"
)

func readNativeResizeRect() resizeRect {
	var rect resizeRect
	fyne.DoAndWait(func() { rect, _, _ = nativeResizeState() })
	return rect
}

func waitNativeResize(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		var ready bool
		fyne.DoAndWait(func() { ready = condition() })
		if ready {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("native resize did not settle")
}

func dragNativeResizeGrip(t *testing.T, edge resizeEdge, dx, dy int32) (resizeRect, resizeRect) {
	t.Helper()
	a := nativeWindowApp
	var grip *windowResizeGrip
	var start resizeRect
	var pointer resizePoint
	postMessage := modUser32.NewProc("PostMessageW")
	setCursor := modUser32.NewProc("SetCursorPos")
	fyne.DoAndWait(func() {
		start, _, _ = nativeResizeState()
		for _, object := range a.win.Content().(*fyne.Container).Objects {
			if candidate, ok := object.(*windowResizeGrip); ok && candidate.edge == edge && candidate.Size().Width > 0 {
				grip = candidate
				break
			}
		}
		if grip == nil {
			return
		}
		position := grip.Position().Add(fyne.NewPos(grip.Size().Width/2, grip.Size().Height/2))
		x, y := a.win.Canvas().PixelCoordinateForPosition(position)
		pointer = resizePoint{start.left + int32(x), start.top + int32(y)}
		setCursor.Call(uintptr(pointer.x), uintptr(pointer.y))
		point := uintptr(uint32(uint16(x)) | uint32(uint16(y))<<16)
		postMessage.Call(GetMainHWND(), 0x0200, 0, point)
		postMessage.Call(GetMainHWND(), 0x0201, 1, point)
	})
	if grip == nil {
		t.Fatalf("missing resize grip for edge %d", edge)
	}
	waitNativeResize(t, func() bool { return grip.drag.active })
	fyne.DoAndWait(func() { pointer = grip.drag.pointer })
	t.Cleanup(func() {
		fyne.DoAndWait(func() {
			postMessage.Call(GetMainHWND(), 0x0202, 0, 0)
			grip.drag.active = false
		})
	})
	// Two moves also verify that a left/top resize is based on the original
	// screen position instead of accumulating deltas after the window moves.
	for _, fraction := range []float32{0.5, 1} {
		x, y := pointer.x+int32(float32(dx)*fraction), pointer.y+int32(float32(dy)*fraction)
		var want resizeRect
		deadline := time.Now().Add(time.Second)
		settled := false
		for time.Now().Before(deadline) {
			fyne.DoAndWait(func() {
				minimum, scale := a.win.Content().MinSize(), a.win.Canvas().Scale()
				want = resizedWindowRect(start, edge, resizePoint{x - pointer.x, y - pointer.y},
					int32(math.Ceil(float64(minimum.Width*scale))), int32(math.Ceil(float64(minimum.Height*scale))))
				setCursor.Call(uintptr(x), uintptr(y))
				current, _, _ := nativeResizeState()
				point := uintptr(uint32(uint16(x-current.left)) | uint32(uint16(y-current.top))<<16)
				postMessage.Call(GetMainHWND(), 0x0200, 1, point)
			})
			time.Sleep(20 * time.Millisecond)
			if readNativeResizeRect() == want {
				settled = true
				break
			}
		}
		if !settled {
			t.Fatalf("drag did not settle at requested screen delta: got %v, want %v", readNativeResizeRect(), want)
		}
	}
	var after resizeRect
	fyne.DoAndWait(func() {
		after, _, _ = nativeResizeState()
		point := uintptr(uint32(uint16(pointer.x+dx-after.left)) | uint32(uint16(pointer.y+dy-after.top))<<16)
		postMessage.Call(GetMainHWND(), 0x0202, 0, point)
	})
	waitNativeResize(t, func() bool { return !grip.drag.active })
	return start, after
}

func assertNativeResizeCanvas(t *testing.T) {
	t.Helper()
	var client resizeRect
	var width, height int32
	fyne.DoAndWait(func() {
		modUser32.NewProc("GetClientRect").Call(GetMainHWND(), uintptr(unsafe.Pointer(&client)))
		c := nativeWindowApp.win.Canvas()
		width = int32(math.Round(float64(c.Size().Width * c.Scale())))
		height = int32(math.Round(float64(c.Size().Height * c.Scale())))
	})
	if client.right != width || client.bottom != height {
		t.Fatalf("native client %v differs from canvas %dx%d", client, width, height)
	}
}

func TestNativeWindowManualResize(t *testing.T) {
	a := nativeWindowApp
	original := readNativeResizeRect()
	var originalPointer resizePoint
	fyne.DoAndWait(func() { _, originalPointer, _ = nativeResizeState() })
	t.Cleanup(func() {
		fyne.DoAndWait(func() {
			applyNativeResizeRect(original)
			modUser32.NewProc("SetCursorPos").Call(uintptr(originalPointer.x), uintptr(originalPointer.y))
		})
	})
	for _, tc := range []struct {
		name   string
		edge   resizeEdge
		dx, dy int32
	}{
		{"left", resizeLeft, -40, 0}, {"right", resizeRight, 40, 0},
		{"top", resizeTop, 0, -30}, {"bottom", resizeBottom, 0, 30},
		{"top-left", resizeTop | resizeLeft, -40, -30}, {"top-right", resizeTop | resizeRight, 40, -30},
		{"bottom-left", resizeBottom | resizeLeft, -40, 30}, {"bottom-right", resizeBottom | resizeRight, 40, 30},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fyne.DoAndWait(func() { applyNativeResizeRect(original) })
			start, after := dragNativeResizeGrip(t, tc.edge, tc.dx, tc.dy)
			want := resizedWindowRect(start, tc.edge, resizePoint{tc.dx, tc.dy}, 1, 1)
			if after != want {
				t.Fatalf("drag %s: got %v, want %v", tc.name, after, want)
			}
			assertNativeResizeCanvas(t)
			t.Logf("drag %s: %v -> %v; native/canvas agree", tc.name, start, after)
		})
	}
	// Shrink from the top-left past the layout minimum. The lower-right
	// corner must stay fixed, with no oscillation at the minimum boundary.
	fyne.DoAndWait(func() { applyNativeResizeRect(original) })
	var minWidth, minHeight int32
	fyne.DoAndWait(func() {
		minimum, scale := a.win.Content().MinSize(), a.win.Canvas().Scale()
		minWidth = int32(math.Ceil(float64(minimum.Width * scale)))
		minHeight = int32(math.Ceil(float64(minimum.Height * scale)))
	})
	start, clamped := dragNativeResizeGrip(t, resizeLeft|resizeTop, 1000, 700)
	if clamped.right != start.right || clamped.bottom != start.bottom || clamped.right-clamped.left != minWidth || clamped.bottom-clamped.top != minHeight {
		t.Fatalf("minimum-size clamp moved the opposite corner or used wrong bounds: %v, minimum %dx%d", clamped, minWidth, minHeight)
	}
	assertNativeResizeCanvas(t)
	t.Logf("minimum clamp: %dx%d; opposite corner held", minWidth, minHeight)
	fyne.DoAndWait(func() { applyNativeResizeRect(original) })
	_, resized := dragNativeResizeGrip(t, resizeRight|resizeBottom, 80, 40)
	fyne.DoAndWait(ToggleMaximizeNativeWindow)
	waitNativeResize(t, IsNativeWindowMaximized)
	fyne.DoAndWait(func() {
		for _, object := range a.win.Content().(*fyne.Container).Objects {
			if grip, ok := object.(*windowResizeGrip); ok && grip.Size() != (fyne.Size{}) {
				t.Errorf("resize grip remains active when maximized: %v", grip.Size())
			}
		}
	})
	fyne.DoAndWait(ToggleMaximizeNativeWindow)
	waitNativeResize(t, func() bool { return !IsNativeWindowMaximized() })
	if got := readNativeResizeRect(); got != resized {
		t.Fatalf("restore lost manually resized bounds: %v, want %v", got, resized)
	}
	assertNativeResizeCanvas(t)
	// Clicking a normal control after dragging must retain the same hit map.
	clickNativeControl(t, a.alwaysOnTopBtn)
	waitNativeResize(t, func() bool { return !a.IsAlwaysOnTop() })
	clickNativeControl(t, a.alwaysOnTopBtn)
	waitNativeResize(t, a.IsAlwaysOnTop)
}
