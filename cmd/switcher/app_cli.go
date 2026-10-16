//go:build !gui

package main

import (
	"fmt"
	"os"

	"antigravity-switch/internal/manager"
)

const hasGUI = false

func launchGUI(mgr *manager.AccountManager, showStatus, showList, doCapture bool, switchTarget string) error {
	fmt.Fprintln(os.Stderr, "⚡ Antigravity Multi-Account Switcher (Pure Go Native Engine)")
	fmt.Fprintln(os.Stderr, "[INFO] GUI is disabled in this pure Go standalone build.")
	fmt.Fprintln(os.Stderr, "[INFO] Use CLI flags (--status, --list, --switch <email>, --capture). Showing status:")
	runCLI(mgr, true, false, false, "")
	return nil
}
