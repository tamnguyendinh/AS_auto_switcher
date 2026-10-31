package main

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"

	"as-auto-switcher/internal/core/version"
	"as-auto-switcher/internal/engines/antigravity/manager"
	"as-auto-switcher/internal/engines/antigravity/reloader"
	"as-auto-switcher/internal/engines/antigravity/store"
	"as-auto-switcher/internal/ui"
)

func main() {
	// Report build metadata before acquiring the GUI lock or opening account data.
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Println(version.GetBuildInfo())
		return
	}

	// 1. Kiểm tra đơn bản sao (Single Instance Lock)
	lock, isFirst := ui.TryAcquireSingleInstance("⚡ AS Auto Switcher")
	if !isFirst {
		// Ứng dụng đã đang chạy, cửa sổ cũ đã được đưa lên trên màn hình -> Thoát bản sao mới ngay
		os.Exit(0)
	}
	defer lock.Release()

	// 2. Di chuyển kho tài khoản legacy vào accounts_store/antigravity nếu cần
	legacyRoot := store.GetDefaultRootStoreDir()
	if legacyRoot != "" {
		if err := store.MigrateLegacyRootStore(legacyRoot); err != nil {
			showStartupError(fmt.Errorf("Không di chuyển được kho tài khoản cũ: %w", err))
			os.Exit(1)
		}
		// Di chuyển discovered_accounts ở root vào accounts_store/antigravity/discovered_accounts
		if err := store.MigrateLegacyDiscoveredStore(legacyRoot); err != nil {
			showStartupError(fmt.Errorf("Không di chuyển được danh sách tài khoản quét Chrome cũ: %w", err))
			os.Exit(1)
		}
	}

	// 3. Khởi tạo AccountStore & Reloader & Manager
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

	// 4. Khởi tạo và khởi chạy giao diện GUI Fyne
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
	title, _ := windows.UTF16PtrFromString("AS Auto Switcher")
	_, _ = windows.MessageBox(0, text, title, windows.MB_OK|windows.MB_ICONERROR)
}
