//go:build !windows

package ui

func SetMainHWND(hwnd uintptr) {}
func GetMainHWND() uintptr { return 0 }
func MakeWindowFrameless(windowTitle string) uintptr { return 0 }
func DragNativeWindow() {}
func MinimizeNativeWindow() {}
func ToggleMaximizeNativeWindow() {}