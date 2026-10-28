package main

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"

	"antigravity-switch/internal/manager"
	"antigravity-switch/internal/reloader"
	"antigravity-switch/internal/store"
	"antigravity-switch/internal/ui"
)

func main() {
	// 1. Kiểm tra đơn bản sao (Single Instance Lock)
	lock, isFirst := ui.TryAcquireSingleInstance("⚡ Auto Switcher")
	if !isFirst {
		// Ứng dụng đã đang chạy, cửa sổ cũ đã được đưa lên trên màn hình -> Thoát bản sao mới ngay
		os.Exit(0)
	}
	defer lock.Release()

	// 2. Khởi tạo AccountStore & Reloader & Manager
	accStore, err := store.NewAccountStore("")
	if err != nil {
		showStartupError(fmt.Errorf("Không mở được kho tài khoản: %w", err))
		os.Exit(1)
	}

	sessionReloader := reloader.NewSessionReloader()
	mgr, err := manager.NewAccountManager(accStore, sessionReloader)
	if err != nil {
		showStartupError(fmt.Errorf("Không khởi tạo được danh sách tài khoản: %w", err))
		os.Exit(1)
	}

	// 3. Khởi tạo và khởi chạy giao diện GUI Fyne
	appInstance, err := ui.NewApp(mgr)
	if err != nil {
		showStartupError(fmt.Errorf("Không mở được giao diện: %w", err))
		os.Exit(1)
	}
	appInstance.ShowAndRun()
}

func showStartupError(err error) {
	fmt.Fprintln(os.Stderr, err)
	text, _ := windows.UTF16PtrFromString(err.Error())
	title, _ := windows.UTF16PtrFromString("Antigravity Switcher")
	_, _ = windows.MessageBox(0, text, title, windows.MB_OK|windows.MB_ICONERROR)
}
