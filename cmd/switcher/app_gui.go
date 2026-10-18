//go:build gui

package main

import (
	"fmt"
	"os"

	"antigravity-switch/internal/manager"
	"antigravity-switch/internal/ui"
)

func launchGUI(mgr *manager.AccountManager, showStatus, showList, doCapture bool, switchTarget string, showQuota bool) error {
	// BƯỚC KIỂM TRA ĐƠN BẢN SAO:
	lock, isFirst := ui.TryAcquireSingleInstance("⚡ Antigravity Multi-Account Switcher")
	if !isFirst {
		// App đã đang chạy, cửa sổ cũ đã được kéo lên trước mặt -> Bản sao mới thoát ngay
		os.Exit(0)
		return nil
	}
	defer lock.Release()

	// KHỞI CHẠY GIAO DIỆN BÌNH THƯỜNG:
	appInstance, err := ui.NewApp(mgr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed launching GUI: %v\nFalling back to CLI mode.\n", err)
		runCLI(mgr, showStatus, showList, doCapture, switchTarget, showQuota)
		return nil
	}
	appInstance.ShowAndRun()
	return nil
}