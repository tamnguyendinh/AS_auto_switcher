//go:build !windows

package ui

import "fyne.io/fyne/v2"

func SetMainHWND(hwnd uintptr)             {}
func GetMainHWND() uintptr                 { return 0 }
func bindMainWindowHandle(win fyne.Window) {}
func DragNativeWindow()                    {}
func MinimizeNativeWindow()                {}
func ToggleMaximizeNativeWindow()          {}
func IsNativeWindowMaximized() bool        { return false }
