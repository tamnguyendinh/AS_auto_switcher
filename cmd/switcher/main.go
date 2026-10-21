package main

import (
	"fmt"
	"os"

	"antigravity-switch/internal/manager"
	"antigravity-switch/internal/reloader"
	"antigravity-switch/internal/store"
	"antigravity-switch/internal/ui"
)

func main() {
	// 1. Kiểm tra đơn bản sao (Single Instance Lock)
	lock, isFirst := ui.TryAcquireSingleInstance("⚡ Antigravity Multi-Account Switcher")
	if !isFirst {
		// Ứng dụng đã đang chạy, cửa sổ cũ đã được đưa lên trên màn hình -> Thoát bản sao mới ngay
		os.Exit(0)
	}
	defer lock.Release()

	// 2. Khởi tạo AccountStore & Reloader & Manager
	accStore, err := store.NewAccountStore("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing account store: %v\n", err)
		os.Exit(1)
	}

	sessionReloader := reloader.NewSessionReloader()
	mgr, err := manager.NewAccountManager(accStore, sessionReloader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing account manager: %v\n", err)
		os.Exit(1)
	}

	// 3. Khởi tạo và khởi chạy giao diện GUI Fyne
	appInstance, err := ui.NewApp(mgr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed launching GUI: %v\n", err)
		os.Exit(1)
	}
	appInstance.ShowAndRun()
}
