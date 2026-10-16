//go:build gui

package main

import (
	"fmt"
	"os"

	"antigravity-switch/internal/manager"
	"antigravity-switch/internal/ui"
)

const hasGUI = true

func launchGUI(mgr *manager.AccountManager, showStatus, showList, doCapture bool, switchTarget string) error {
	appInstance, err := ui.NewApp(mgr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed launching GUI: %v\nFalling back to CLI mode.\n", err)
		runCLI(mgr, showStatus, showList, doCapture, switchTarget)
		return nil
	}
	appInstance.ShowAndRun()
	return nil
}
