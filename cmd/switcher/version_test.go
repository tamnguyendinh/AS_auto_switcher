package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"as-auto-switcher/internal/core/version"

	"golang.org/x/sys/windows"
)

func TestVersionExitsBeforeGUIAndStoreInitialization(t *testing.T) {
	const helperEnv = "AS_SWITCHER_VERSION_TEST_PROCESS"
	if os.Getenv(helperEnv) == "1" {
		version.Version = "9.8.7.6"
		version.GitCommit = "test-build-commit"
		os.Args = []string{"AS_Auto_Switcher.exe", "--version"}
		main()
		os.Exit(0)
	}

	// The diagnostic must still work while another GUI instance holds the lock.
	handle, err := windows.CreateMutex(nil, false, windows.StringToUTF16Ptr(`Global\AS_Auto_Switcher_SingleInstance_Mutex`))
	if err != nil && err != windows.ERROR_ALREADY_EXISTS {
		t.Fatal(err)
	}
	defer windows.CloseHandle(handle)

	// A file cannot serve as the account directory; --version must not open it.
	storePath := filepath.Join(t.TempDir(), "not-a-store")
	if err := os.WriteFile(storePath, []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestVersionExitsBeforeGUIAndStoreInitialization$")
	cmd.Env = append(os.Environ(), helperEnv+"=1", "ANTIGRAVITY_STORE_DIR="+storePath)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("version diagnostic did not exit successfully: %v", err)
	}
	if got, want := strings.TrimSpace(string(out)), "Version 9.8.7.6 (commit test-build-commit)"; got != want {
		t.Fatalf("version output = %q, want %q", got, want)
	}
	if body, err := os.ReadFile(storePath); err != nil || string(body) != "preserve" {
		t.Fatalf("version diagnostic touched account data: %q, %v", body, err)
	}
}
