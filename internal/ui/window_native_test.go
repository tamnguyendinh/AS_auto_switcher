//go:build windows && gui && nativeui

package ui

import (
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"golang.org/x/sys/windows"
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
	result := make(chan int, 1)
	go func() {
		code := m.Run()
		result <- code
		fyne.Do(nativeWindowApp.app.Quit)
	}()
	nativeWindowApp.ShowAndRun()
	code := <-result
	_ = os.RemoveAll(profile)
	os.Exit(code)
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
	postMessage := modUser32.NewProc("PostMessageW")
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
		point := uintptr(uint32(uint16(x)) | uint32(uint16(y))<<16)
		// Route real client-coordinate mouse messages through GLFW's WndProc.
		// Calling Tapped directly would miss the original coordinate mismatch.
		for _, event := range []struct{ message, state uintptr }{{0x0200, 0}, {0x0201, 1}, {0x0202, 0}} {
			ok, _, err := postMessage.Call(hwnd, event.message, event.state, point)
			if ok == 0 {
				t.Fatalf("posting native pointer event failed: %v", err)
			}
		}
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
