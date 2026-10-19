//go:build windows

package ui

import (
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	procGetWindowLongW = modUser32.NewProc("GetWindowLongW")
	procSetWindowLongW = modUser32.NewProc("SetWindowLongW")
	procIsZoomed       = modUser32.NewProc("IsZoomed")
	procReleaseCapture = modUser32.NewProc("ReleaseCapture")
	procSendMessageW   = modUser32.NewProc("SendMessageW")

	gwlStyleVal int32 = -16 // GWL_STYLE = -16
)

const (
	WS_CAPTION       = 0x00C00000
	WS_THICKFRAME    = 0x00040000
	SWP_FRAMECHANGED = 0x0020
	SWP_NOMOVE       = 0x0002
	SWP_NOSIZE       = 0x0001
	SWP_NOZORDER     = 0x0004

	SW_MINIMIZE = 6
	SW_MAXIMIZE = 3
	SW_RESTORE  = 9

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

// MakeWindowFrameless loại bỏ thanh tiêu đề xám của Windows nhưng giữ WS_THICKFRAME để co giãn cửa sổ.
func MakeWindowFrameless(windowTitle string) uintptr {
	titlePtr := windows.StringToUTF16Ptr(windowTitle)
	hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(titlePtr)))
	if hwnd == 0 {
		// Thăm dò FindWindowW mỗi 30-50ms trong tối đa 3-5 giây phòng trường hợp hệ điều hành đang ánh xạ non-client area
		for i := 0; i < 60; i++ {
			time.Sleep(50 * time.Millisecond)
			hwnd, _, _ = procFindWindowW.Call(0, uintptr(unsafe.Pointer(titlePtr)))
			if hwnd != 0 {
				break
			}
		}
	}
	if hwnd == 0 {
		return 0
	}
	SetMainHWND(hwnd)

	style, _, _ := procGetWindowLongW.Call(hwnd, uintptr(gwlStyleVal))
	newStyle := (style &^ uintptr(WS_CAPTION)) | uintptr(WS_THICKFRAME)
	procSetWindowLongW.Call(hwnd, uintptr(gwlStyleVal), newStyle)
	procSetWindowPos.Call(hwnd, 0, 0, 0, 0, 0, uintptr(SWP_FRAMECHANGED|SWP_NOMOVE|SWP_NOSIZE|SWP_NOZORDER))
	return hwnd
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

// ToggleMaximizeNativeWindow phóng to hoặc khôi phục kích thước cửa sổ.
func ToggleMaximizeNativeWindow() {
	hwnd := GetMainHWND()
	if hwnd != 0 {
		zoomed, _, _ := procIsZoomed.Call(hwnd)
		if zoomed != 0 {
			procShowWindow.Call(hwnd, uintptr(SW_RESTORE))
		} else {
			procShowWindow.Call(hwnd, uintptr(SW_MAXIMIZE))
		}
	}
}