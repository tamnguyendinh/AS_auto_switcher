//go:build windows

package ui

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modKernel32      = windows.NewLazySystemDLL("kernel32.dll")
	procCreateMutexW = modKernel32.NewProc("CreateMutexW")
	procCloseHandle  = modKernel32.NewProc("CloseHandle")

	procShowWindow          = modUser32.NewProc("ShowWindow")
	procSetForegroundWindow = modUser32.NewProc("SetForegroundWindow")
)

const (
	swRestore               = 9
	singleInstanceMutexName = "Global\\AS_Auto_Switcher_SingleInstance_Mutex"
)

type SingleInstanceLock struct {
	handle uintptr
}

// TryAcquireSingleInstance kiểm tra xem app đã đang chạy chưa.
// Nếu đã chạy: Đưa cửa sổ cũ lên trên cùng màn hình và trả về false.
// Nếu là bản đầu tiên: Nắm giữ khóa và trả về true.
func TryAcquireSingleInstance(windowTitle string) (*SingleInstanceLock, bool) {
	namePtr := windows.StringToUTF16Ptr(singleInstanceMutexName)
	hMutex, _, err := procCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(namePtr)))

	if err == windows.ERROR_ALREADY_EXISTS || hMutex == 0 {
		titlePtr := windows.StringToUTF16Ptr(windowTitle)
		hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(titlePtr)))
		if hwnd != 0 {
			procShowWindow.Call(hwnd, uintptr(swRestore))
			procSetForegroundWindow.Call(hwnd)
		}
		if hMutex != 0 {
			procCloseHandle.Call(hMutex)
		}
		return nil, false
	}

	return &SingleInstanceLock{handle: hMutex}, true
}

func (l *SingleInstanceLock) Release() {
	if l != nil && l.handle != 0 {
		procCloseHandle.Call(l.handle)
		l.handle = 0
	}
}