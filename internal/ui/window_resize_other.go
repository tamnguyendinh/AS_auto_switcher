//go:build !windows

package ui

func nativeResizeState() (resizeRect, resizePoint, bool) { return resizeRect{}, resizePoint{}, false }
func nativeResizeCaptured() bool                         { return false }
func applyNativeResizeRect(resizeRect) bool              { return false }
