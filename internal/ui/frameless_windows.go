//go:build windows

package ui

import (
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver"
)

var (
	procReleaseCapture = modUser32.NewProc("ReleaseCapture")
	procSendMessageW   = modUser32.NewProc("SendMessageW")
	procIsZoomed       = modUser32.NewProc("IsZoomed")
)

const (
	SW_MINIMIZE = 6
	SW_MAXIMIZE = 3

	WM_NCLBUTTONDOWN = 0x00A1
	HTCAPTION        = 0x0002
)

var (
	mainHWNDLock sync.RWMutex
	mainHWND     uintptr
)

func SetMainHWND(hwnd uintptr) {
	mainHWNDLock.Lock()
	defer mainHWNDLock.Unlock()
	mainHWND = hwnd
}

func GetMainHWND() uintptr {
	mainHWNDLock.RLock()
	defer mainHWNDLock.RUnlock()
	return mainHWND
}

// bindMainWindowHandle must run after Show, when Fyne has created the HWND.
// Native controls use the actual window, without title lookup or style changes.
func bindMainWindowHandle(win fyne.Window) {
	SetMainHWND(0)
	if native, ok := win.(driver.NativeWindow); ok {
		native.RunNative(func(context any) {
			if windowsContext, ok := context.(driver.WindowsWindowContext); ok {
				SetMainHWND(windowsContext.HWND)
			}
		})
	}
}

// DragNativeWindow cho phép bấm giữ chuột vào thanh tiêu đề tự vẽ để kéo cửa sổ native.
func DragNativeWindow() {
	hwnd := GetMainHWND()
	if hwnd != 0 {
		procReleaseCapture.Call()
		procSendMessageW.Call(hwnd, uintptr(WM_NCLBUTTONDOWN), uintptr(HTCAPTION), 0)
	}
}

// MinimizeNativeWindow thu nhỏ cửa sổ xuống Taskbar.
func MinimizeNativeWindow() {
	hwnd := GetMainHWND()
	if hwnd != 0 {
		procShowWindow.Call(hwnd, uintptr(SW_MINIMIZE))
	}
}

func IsNativeWindowMaximized() bool {
	if hwnd := GetMainHWND(); hwnd != 0 {
		maximized, _, _ := procIsZoomed.Call(hwnd)
		return maximized != 0
	}
	return false
}

// ToggleMaximizeNativeWindow uses the monitor work area (leaving the taskbar
// visible). Windows retains the normal window placement for the restore action.
func ToggleMaximizeNativeWindow() {
	if hwnd := GetMainHWND(); hwnd != 0 {
		command := SW_MAXIMIZE
		if IsNativeWindowMaximized() {
			command = swRestore
		}
		procShowWindow.Call(hwnd, uintptr(command))
	}
}
