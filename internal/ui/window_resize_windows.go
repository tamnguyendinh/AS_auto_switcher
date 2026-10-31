//go:build windows

package ui

import "unsafe"

var (
	procResizeGetWindowRect = modUser32.NewProc("GetWindowRect")
	procResizeGetCursorPos  = modUser32.NewProc("GetCursorPos")
	procResizeGetCapture    = modUser32.NewProc("GetCapture")
)

func nativeResizeState() (rect resizeRect, pointer resizePoint, ok bool) {
	hwnd := GetMainHWND()
	if hwnd == 0 {
		return
	}
	windowOK, _, _ := procResizeGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))
	pointerOK, _, _ := procResizeGetCursorPos.Call(uintptr(unsafe.Pointer(&pointer)))
	return rect, pointer, windowOK != 0 && pointerOK != 0
}

func nativeResizeCaptured() bool {
	capture, _, _ := procResizeGetCapture.Call()
	return capture != 0 && capture == GetMainHWND()
}

func applyNativeResizeRect(rect resizeRect) bool {
	hwnd := GetMainHWND()
	if hwnd == 0 || rect.right <= rect.left || rect.bottom <= rect.top {
		return false
	}
	// Preserve style, activation and z-order. WM_SIZE/WM_MOVE inform GLFW of
	// the actual client geometry instead of manually adjusting Fyne coordinates.
	ok, _, _ := procSetWindowPos.Call(hwnd, 0, uintptr(rect.left), uintptr(rect.top),
		uintptr(rect.right-rect.left), uintptr(rect.bottom-rect.top), 0x0004|0x0010)
	return ok != 0
}
