//go:build !windows

package ui

type SingleInstanceLock struct{}

func TryAcquireSingleInstance(windowTitle string) (*SingleInstanceLock, bool) {
	return &SingleInstanceLock{}, true
}

func (l *SingleInstanceLock) Release() {}