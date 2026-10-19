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
)

const (
	SW_MINIMIZE = 6

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

// ToggleMaximizeNativeWindow vô hiệu hóa chức năng phóng to (cửa sổ cố định 1008x630).
func ToggleMaximizeNativeWindow() {}
