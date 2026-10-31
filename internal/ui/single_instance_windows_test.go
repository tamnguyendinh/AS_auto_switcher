//go:build windows

package ui

import (
	"testing"

	"golang.org/x/sys/windows"
)

func TestSingleInstanceRecognizesApplicationMutex(t *testing.T) {
	// Hold the public OS identity independently of the implementation constant.
	handle, err := windows.CreateMutex(nil, false, windows.StringToUTF16Ptr(`Global\AS_Auto_Switcher_SingleInstance_Mutex`))
	if err != nil && err != windows.ERROR_ALREADY_EXISTS {
		t.Fatal(err)
	}
	defer windows.CloseHandle(handle)
	lock, first := TryAcquireSingleInstance("AS switcher mutex test: no matching window")
	defer lock.Release()
	if first || lock != nil {
		t.Fatal("a second application instance did not recognize the held OS mutex")
	}
}
