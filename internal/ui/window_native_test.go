//go:build windows && gui && nativeui

package ui

import (
	"fmt"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/theme"
	"golang.org/x/sys/windows"

	"antigravity-switch/internal/manager"
)

// Run explicitly on an interactive Windows desktop:
// go test -tags "gui nativeui" ./internal/ui -run '^TestNativeWindow' -count=1
// Unlike the headless tests, this exercises the production GLFW/Win32 window.
var nativeWindowApp *App

func TestMain(m *testing.M) {
	// The production pin helper resolves the single-instance window by title.
	// Refuse to run alongside the user's app so this test cannot target it.
	title := windows.StringToUTF16Ptr(windowTitle)
	if hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(title))); hwnd != 0 {
		os.Stderr.WriteString("Close Antigravity Switcher before running the native window test.\n")
		os.Exit(2)
	}
	root := filepath.Join("..", "..", ".tmp")
	if err := os.MkdirAll(root, 0755); err != nil {
		panic(err)
	}
	profile, err := os.MkdirTemp(root, "native-window-")
	if err != nil {
		panic(err)
	}
	nativeWindowApp = &App{
		app:             app.NewWithID("com.antigravity.switcher.native-window-test"),
		alwaysOnTop:     true,
		currentTheme:    ThemeDark,
		preferencesPath: filepath.Join(profile, "preferences.json"),
	}
	nativeWindowApp.Init()
	var savedPointer resizePoint
	procResizeGetCursorPos.Call(uintptr(unsafe.Pointer(&savedPointer)))
	go func() {
		code := m.Run()
		// This harness verifies window geometry/input, not the system-tray exit
		// lifecycle. Finish after tests and temporary-profile cleanup rather than
		// waiting indefinitely for the tray's desktop message loop to terminate.
		nativeWindowApp.Close()
		fyne.DoAndWait(func() {
			modUser32.NewProc("SetCursorPos").Call(uintptr(savedPointer.x), uintptr(savedPointer.y))
		})
		_ = os.RemoveAll(profile)
		os.Exit(code)
	}()
	nativeWindowApp.ShowAndRun()
	_ = os.RemoveAll(profile)
	os.Exit(1) // The native runtime stopped before the tests completed.
}

// Send client-coordinate pointer input through the real GLFW WndProc.
func clickNativeControl(t *testing.T, object fyne.CanvasObject) {
	t.Helper()
	a := nativeWindowApp
	var x, y int
	fyne.DoAndWait(func() {
		position := a.app.Driver().AbsolutePositionForObject(object)
		center := position.Add(fyne.NewPos(object.Size().Width/2, object.Size().Height/2))
		x, y = a.win.Canvas().PixelCoordinateForPosition(center)
		pointer := resizePoint{int32(x), int32(y)}
		modUser32.NewProc("ClientToScreen").Call(GetMainHWND(), uintptr(unsafe.Pointer(&pointer)))
		modUser32.NewProc("SetCursorPos").Call(uintptr(pointer.x), uintptr(pointer.y))
	})
	point := uintptr(uint32(uint16(x)) | uint32(uint16(y))<<16)
	postMessage := modUser32.NewProc("PostMessageW")
	for _, event := range []struct{ message, state uintptr }{{0x0200, 0}, {0x0201, 1}, {0x0202, 0}} {
		if ok, _, err := postMessage.Call(GetMainHWND(), event.message, event.state, point); ok == 0 {
			t.Fatalf("posting native pointer event failed: %v", err)
		}
	}
}

func TestNativeWindowMaximizeRestoreAndGrid(t *testing.T) {
	a := nativeWindowApp
	var maximize *customTitleBtn
	var originalSize fyne.Size
	fyne.DoAndWait(func() {
		var find func(fyne.CanvasObject)
		find = func(object fyne.CanvasObject) {
			if button, ok := object.(*customTitleBtn); ok && button.stateLabel != nil {
				maximize = button
			}
			if parent, ok := object.(*fyne.Container); ok {
				for _, child := range parent.Objects {
					find(child)
				}
			}
		}
		find(a.win.Content())
		originalSize = a.win.Canvas().Size()
		for i := 0; i < 12; i++ {
			card := NewAccountCardWithQuota(manager.AccountDisplayItem{Email: fmt.Sprintf("account-%02d@example.invalid", i)}, nil, DarkPalette, nil, nil, nil)
			a.accountListContainer.Add(card)
		}
	})
	if maximize == nil {
		t.Fatal("maximize/restore button is missing")
	}
	t.Cleanup(func() {
		fyne.DoAndWait(func() {
			if IsNativeWindowMaximized() {
				ToggleMaximizeNativeWindow()
			}
			a.accountListContainer.RemoveAll()
		})
	})
	waitState := func(want bool) {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			var ready bool
			fyne.DoAndWait(func() {
				ready = IsNativeWindowMaximized() == want && (want || a.win.Canvas().Size() == originalSize)
			})
			if ready {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatalf("window did not reach maximized=%t with the expected geometry", want)
	}
	checkGrid := func() int {
		t.Helper()
		var columns, expected int
		fyne.DoAndWait(func() {
			expected = int((a.accountListContainer.Size().Width + theme.Padding()) / (280 + theme.Padding()))
			for _, card := range a.accountListContainer.Objects {
				if card.Size() != fyne.NewSize(280, 130) {
					t.Errorf("card stretched to %v", card.Size())
				}
				if card.Position().Y == 0 {
					columns++
				}
			}
		})
		if columns != expected {
			t.Errorf("grid has %d columns, expected %d from available width", columns, expected)
		}
		return columns
	}
	originalColumns := checkGrid()
	for cycle := 0; cycle < 2; cycle++ {
		clickNativeControl(t, maximize)
		waitState(true)
		type rect struct{ left, top, right, bottom int32 }
		var client, window rect
		var monitor struct {
			size         uint32
			bounds, work rect
			flags        uint32
		}
		var canvasSize fyne.Size
		var scale float32
		var icon string
		fyne.DoAndWait(func() {
			hwnd := GetMainHWND()
			modUser32.NewProc("GetClientRect").Call(hwnd, uintptr(unsafe.Pointer(&client)))
			modUser32.NewProc("GetWindowRect").Call(hwnd, uintptr(unsafe.Pointer(&window)))
			hmon, _, _ := modUser32.NewProc("MonitorFromWindow").Call(hwnd, 2)
			monitor.size = uint32(unsafe.Sizeof(monitor))
			modUser32.NewProc("GetMonitorInfoW").Call(hmon, uintptr(unsafe.Pointer(&monitor)))
			canvasSize, scale = a.win.Canvas().Size(), a.win.Canvas().Scale()
			icon = maximize.text.Text
		})
		if window != monitor.work {
			t.Fatalf("maximized bounds %v must match monitor work area %v", window, monitor.work)
		}
		if int(client.right) != int(math.Ceil(float64(canvasSize.Width*scale))) || int(client.bottom) != int(math.Ceil(float64(canvasSize.Height*scale))) {
			t.Fatalf("maximized native client %v differs from canvas %v at scale %f", client, canvasSize, scale)
		}
		if icon != "❐" {
			t.Errorf("maximized button must show restore icon, got %q", icon)
		}
		columns := checkGrid()
		t.Logf("maximize cycle %d: native=canvas=%dx%d, grid %d -> %d columns, cards 280x130", cycle+1, client.right, client.bottom, originalColumns, columns)
		if capturePath := os.Getenv("NATIVE_MAXIMIZE_CAPTURE"); capturePath != "" && cycle == 0 {
			file, err := os.Create(capturePath)
			if err != nil {
				t.Fatal(err)
			}
			fyne.DoAndWait(func() { err = png.Encode(file, a.win.Canvas().Capture()) })
			file.Close()
			if err != nil {
				t.Fatal(err)
			}
		}
		clickNativeControl(t, maximize)
		waitState(false)
		if columns := checkGrid(); columns != originalColumns {
			t.Errorf("restored grid has %d columns, expected %d", columns, originalColumns)
		}
		t.Logf("restore cycle %d: canvas=%v, grid=%d columns", cycle+1, originalSize, originalColumns)
	}
}

func TestNativeWindowGeometryAndPointer(t *testing.T) {
	a := nativeWindowApp
	var hwnd uintptr
	fyne.DoAndWait(func() { hwnd = GetMainHWND() })
	if hwnd == 0 {
		t.Fatal("production startup did not bind the native window handle")
	}
	getClientRect := modUser32.NewProc("GetClientRect")
	getWindowLong := modUser32.NewProc("GetWindowLongW")
	checkGeometry := func() {
		t.Helper()
		var rect struct{ left, top, right, bottom int32 }
		var expectedWidth, expectedHeight int
		var style uintptr
		fyne.DoAndWait(func() {
			getClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))
			index := int32(-16)
			style, _, _ = getWindowLong.Call(hwnd, uintptr(index))
			canvas := a.win.Canvas()
			expectedWidth = int(math.Ceil(float64(canvas.Size().Width * canvas.Scale())))
			expectedHeight = int(math.Ceil(float64(canvas.Size().Height * canvas.Scale())))
		})
		if int(rect.right) != expectedWidth || int(rect.bottom) != expectedHeight {
			t.Fatalf("native client %dx%d differs from canvas viewport %dx%d", rect.right, rect.bottom, expectedWidth, expectedHeight)
		}
		if style&(0x00C00000|0x00040000) != 0 {
			t.Fatalf("native window still has caption or resize frame: %#x", style)
		}
		t.Logf("native client = canvas viewport = %dx%d; no caption/resize frame", expectedWidth, expectedHeight)
	}
	clickPin := func(want bool) {
		t.Helper()
		var x, y int
		fyne.DoAndWait(func() {
			position := a.app.Driver().AbsolutePositionForObject(a.alwaysOnTopBtn)
			center := position.Add(fyne.NewPos(a.alwaysOnTopBtn.Size().Width/2, a.alwaysOnTopBtn.Size().Height/2))
			x, y = a.win.Canvas().PixelCoordinateForPosition(center)
		})
		clickNativeControl(t, a.alwaysOnTopBtn)
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if a.IsAlwaysOnTop() == want {
				index := int32(-20)
				extendedStyle, _, _ := getWindowLong.Call(hwnd, uintptr(index))
				if (extendedStyle&0x8 != 0) == want {
					t.Logf("click pin center (%d,%d): state and native TOPMOST = %t", x, y, want)
					return
				}
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatalf("click at rendered pin center (%d,%d) did not set TOPMOST=%t", x, y, want)
	}

	checkGeometry()
	clickPin(false)
	clickPin(true)
	for _, theme := range []string{ThemeLight, ThemeWarm, ThemeDark} {
		fyne.DoAndWait(func() { a.SetTheme(theme) })
		checkGeometry()
		clickPin(false)
		clickPin(true)
	}
	fyne.DoAndWait(func() {
		MinimizeNativeWindow()
		procShowWindow.Call(hwnd, uintptr(swRestore))
	})
	checkGeometry()
	clickPin(false)
	clickPin(true)
	if capturePath := os.Getenv("NATIVE_WINDOW_CAPTURE"); capturePath != "" {
		file, err := os.Create(capturePath)
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		fyne.DoAndWait(func() { err = png.Encode(file, a.win.Canvas().Capture()) })
		if err != nil {
			t.Fatal(err)
		}
	}
}
