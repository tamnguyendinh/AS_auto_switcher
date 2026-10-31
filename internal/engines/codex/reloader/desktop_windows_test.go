package reloader

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

// 1. TestWindowsChatGPTLifecycle_InterfaceConformance
func TestWindowsChatGPTLifecycle_InterfaceConformance(t *testing.T) {
	var vd VerifiedDesktop = NewWindowsChatGPTLifecycle("")
	if vd == nil {
		t.Fatal("expected non-nil VerifiedDesktop")
	}
	var dl DesktopLifecycle = vd
	if dl == nil {
		t.Fatal("expected non-nil DesktopLifecycle")
	}
}

// 2. TestWindowsChatGPTLifecycle_DefaultAuthPath
func TestWindowsChatGPTLifecycle_DefaultAuthPath(t *testing.T) {
	vd := NewWindowsChatGPTLifecycle("")
	lc, ok := vd.(*WindowsChatGPTLifecycle)
	if !ok {
		t.Fatalf("expected *WindowsChatGPTLifecycle, got %T", vd)
	}

	home, err := os.UserHomeDir()
	var expected string
	if err == nil && home != "" {
		expected = filepath.Clean(filepath.Join(home, ".codex", "auth.json"))
	} else {
		expected = filepath.Clean(filepath.Join(os.Getenv("USERPROFILE"), ".codex", "auth.json"))
	}

	if lc.authPath != expected {
		t.Fatalf("expected authPath %q, got %q", expected, lc.authPath)
	}

	// Also test explicit authPath
	customPath := filepath.Join(t.TempDir(), "custom_auth.json")
	vdCustom := NewWindowsChatGPTLifecycle(customPath)
	lcCustom, ok := vdCustom.(*WindowsChatGPTLifecycle)
	if !ok {
		t.Fatalf("expected *WindowsChatGPTLifecycle, got %T", vdCustom)
	}
	if lcCustom.authPath != filepath.Clean(customPath) {
		t.Fatalf("expected authPath %q, got %q", filepath.Clean(customPath), lcCustom.authPath)
	}
}

// 3. TestWindowsChatGPTLifecycle_Quiescent
func TestWindowsChatGPTLifecycle_Quiescent(t *testing.T) {
	lc := &WindowsChatGPTLifecycle{
		quiescentDelay: 1 * time.Millisecond,
	}
	ctx := context.Background()

	// 3.1 Non-existent file -> nil
	nonExistent := filepath.Join(t.TempDir(), "non_existent_auth.json")
	if err := lc.Quiescent(ctx, nonExistent); err != nil {
		t.Fatalf("expected nil for non-existent file, got %v", err)
	}

	// 3.2 Normal accessible file -> nil
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "auth.json")
	if err := os.WriteFile(tmpFile, []byte(`{}`), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	if err := lc.Quiescent(ctx, tmpFile); err != nil {
		t.Fatalf("expected nil for accessible file, got %v", err)
	}

	// 3.3 File held with exclusive lock (dwShareMode = 0) -> ErrAuthHomeBusy
	pathPtr, err := windows.UTF16PtrFromString(tmpFile)
	if err != nil {
		t.Fatalf("UTF16PtrFromString error: %v", err)
	}
	h, err := windows.CreateFile(
		pathPtr,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		0, // dwShareMode = 0: exclusive
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		t.Fatalf("failed to open exclusive file handle: %v", err)
	}
	defer windows.CloseHandle(h)

	if err := lc.Quiescent(ctx, tmpFile); !errors.Is(err, ErrAuthHomeBusy) {
		t.Fatalf("expected ErrAuthHomeBusy when file is exclusively held, got %v", err)
	}
}

// 4. TestWindowsChatGPTLifecycle_Probe_WhenNotRunning
func TestWindowsChatGPTLifecycle_Probe_WhenNotRunning(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")

	lc := &WindowsChatGPTLifecycle{
		authPath: authPath,
		procSnapshotFn: func() ([]processEntry, error) {
			return []processEntry{}, nil
		},
	}

	state, err := lc.Probe(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.Running {
		t.Fatalf("expected Running=false, got true")
	}
	if !state.Ready {
		t.Fatalf("expected Ready=true, got false")
	}
	if !state.LauncherVerified {
		t.Fatalf("expected LauncherVerified=true, got false")
	}
	if state.Backend != "file" {
		t.Fatalf("expected Backend='file', got %q", state.Backend)
	}
	if state.AuthPath != authPath {
		t.Fatalf("expected AuthPath=%q, got %q", authPath, state.AuthPath)
	}
	if state.PID != 0 {
		t.Fatalf("expected PID=0, got %d", state.PID)
	}
}

// 5. TestWindowsChatGPTLifecycle_VerifyIdentity_Delegation
func TestWindowsChatGPTLifecycle_VerifyIdentity_Delegation(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")

	// Use makeSampleAuthData from session_reloader_test.go
	sampleData := makeSampleAuthData("alice@example.com", "acc-alice-123")
	if err := os.WriteFile(authPath, sampleData, 0600); err != nil {
		t.Fatalf("failed to write sample auth: %v", err)
	}

	lc := &WindowsChatGPTLifecycle{
		authPath:     authPath,
		pollInterval: 5 * time.Millisecond,
		procSnapshotFn: func() ([]processEntry, error) {
			return []processEntry{
				{
					PID:       12345,
					ParentPID: 1000,
					ExeName:   "ChatGPT.exe",
					ImagePath: `C:\Program Files\WindowsApps\OpenAI.Codex_26.1027.3384.0_x64__2p2nqsd0c76g0\app\ChatGPT.exe`,
				},
			}, nil
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 5.1 VerifyIdentity matches email even though accountID was not provided
	ok, err := lc.VerifyIdentity(ctx, 12345, "alice@example.com")
	if err != nil {
		t.Fatalf("VerifyIdentity unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected VerifyIdentity to return true for matching email")
	}

	// 5.2 VerifyIdentity with case insensitivity
	ok, err = lc.VerifyIdentity(ctx, 12345, "ALICE@EXAMPLE.COM")
	if err != nil {
		t.Fatalf("VerifyIdentity unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected VerifyIdentity to return true for case-insensitive email")
	}

	// 5.3 VerifyIdentity with mismatching email fails safely after short timeout
	failCtx, failCancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer failCancel()
	ok, err = lc.VerifyIdentity(failCtx, 12345, "bob@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("expected VerifyIdentity to return false for mismatching email")
	}

	// 5.4 VerifyAccount with matching accountID
	ok, err = lc.VerifyAccount(ctx, 12345, "alice@example.com", "acc-alice-123")
	if err != nil || !ok {
		t.Fatalf("expected VerifyAccount to pass with matching accountID, got ok=%v, err=%v", ok, err)
	}

	// 5.5 VerifyAccount with wrong accountID
	failCtx2, failCancel2 := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer failCancel2()
	ok, err = lc.VerifyAccount(failCtx2, 12345, "alice@example.com", "acc-wrong")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("expected VerifyAccount to return false for mismatching accountID")
	}
}

// 6. TestWindowsChatGPTLifecycle_ProcessTreeIsolation
func TestWindowsChatGPTLifecycle_ProcessTreeIsolation(t *testing.T) {
	// Replicating Section 2.4 telemetry:
	// PID 15556: ChatGPT.exe Root (Parent 10856 - svchost)
	// PID 7812:  ChatGPT.exe Renderer (Parent 15556)
	// PID 276:   ChatGPT.exe GPU (Parent 15556)
	// PID 1600:  codex.exe child of ChatGPT (Parent 15556)
	// PID 4440:  Code.exe (VS Code parent 1000)
	// PID 17692: codex.exe VS Code Extension (Parent 4440)
	mockProcesses := []processEntry{
		{
			PID:          15556,
			ParentPID:    10856,
			ExeName:      "ChatGPT.exe",
			ImagePath:    `C:\Program Files\WindowsApps\OpenAI.Codex_26.1027.3384.0_x64__2p2nqsd0c76g0\app\ChatGPT.exe`,
			CreationTime: time.Now().Add(-10 * time.Minute),
		},
		{
			PID:          7812,
			ParentPID:    15556,
			ExeName:      "ChatGPT.exe",
			ImagePath:    `C:\Program Files\WindowsApps\OpenAI.Codex_26.1027.3384.0_x64__2p2nqsd0c76g0\app\ChatGPT.exe`,
			CreationTime: time.Now().Add(-9 * time.Minute),
		},
		{
			PID:          276,
			ParentPID:    15556,
			ExeName:      "ChatGPT.exe",
			ImagePath:    `C:\Program Files\WindowsApps\OpenAI.Codex_26.1027.3384.0_x64__2p2nqsd0c76g0\app\ChatGPT.exe`,
			CreationTime: time.Now().Add(-9 * time.Minute),
		},
		{
			PID:          1600,
			ParentPID:    15556,
			ExeName:      "codex.exe",
			ImagePath:    `C:\Users\user\AppData\Local\OpenAI\Codex\bin\codex.exe`,
			CreationTime: time.Now().Add(-8 * time.Minute),
		},
		{
			PID:          4440,
			ParentPID:    1000,
			ExeName:      "Code.exe",
			ImagePath:    `C:\Program Files\Microsoft VS Code\Code.exe`,
			CreationTime: time.Now().Add(-30 * time.Minute),
		},
		{
			PID:          17692,
			ParentPID:    4440,
			ExeName:      "codex.exe",
			ImagePath:    `C:\Users\user\.vscode\extensions\openai.chatgpt-0.1.0\bin\codex.exe`,
			CreationTime: time.Now().Add(-25 * time.Minute),
		},
	}

	var killedPIDs []int
	lc := &WindowsChatGPTLifecycle{
		pollInterval: 5 * time.Millisecond,
		procSnapshotFn: func() ([]processEntry, error) {
			return mockProcesses, nil
		},
		treeKillFn: func(ctx context.Context, pid int) error {
			killedPIDs = append(killedPIDs, pid)
			return nil
		},
	}

	// 6.1 findRootProcess correctly identifies PID 15556
	rootPID, creationTime, err := lc.findRootProcess()
	if err != nil {
		t.Fatalf("unexpected findRootProcess error: %v", err)
	}
	if rootPID != 15556 {
		t.Fatalf("expected root PID 15556, got %d", rootPID)
	}
	if creationTime.IsZero() {
		t.Fatalf("expected non-zero creation time")
	}

	// 6.2 IsRunning reports running with PID 15556
	running, pid, err := lc.IsRunning(context.Background())
	if err != nil {
		t.Fatalf("unexpected IsRunning error: %v", err)
	}
	if !running || pid != 15556 {
		t.Fatalf("expected running=true, pid=15556, got %v, %d", running, pid)
	}

	// 6.3 Stop targets ONLY the root PID 15556
	if err := lc.Stop(context.Background(), 15556); err != nil {
		t.Fatalf("Stop error: %v", err)
	}
	if len(killedPIDs) != 1 || killedPIDs[0] != 15556 {
		t.Fatalf("expected killed PID [15556], got %v", killedPIDs)
	}

	// Confirm that VS Code extension PID 17692 was never targeted
	for _, k := range killedPIDs {
		if k == 17692 {
			t.Fatalf("CRITICAL ISOLATION BREACH: VS Code extension PID 17692 was targeted!")
		}
	}
}

// 7. TestDefaultWindowsDesktopLifecycle_RetainsRejection
func TestDefaultWindowsDesktopLifecycle_RetainsRejection(t *testing.T) {
	dl := NewDefaultWindowsDesktopLifecycle("dummy")
	if dl == nil {
		t.Fatal("expected non-nil DesktopLifecycle")
	}

	ctx := context.Background()

	running, pid, err := dl.IsRunning(ctx)
	if running || pid != 0 || !errors.Is(err, ErrUnsupportedDesktopVerification) {
		t.Fatalf("IsRunning: expected (false, 0, ErrUnsupportedDesktopVerification), got (%v, %d, %v)", running, pid, err)
	}

	err = dl.Stop(ctx, 1234)
	if !errors.Is(err, ErrUnsupportedDesktopVerification) {
		t.Fatalf("Stop: expected ErrUnsupportedDesktopVerification, got %v", err)
	}

	startPID, err := dl.Start(ctx)
	if startPID != 0 || !errors.Is(err, ErrUnsupportedDesktopVerification) {
		t.Fatalf("Start: expected (0, ErrUnsupportedDesktopVerification), got (%d, %v)", startPID, err)
	}

	verified, err := dl.VerifyIdentity(ctx, 1234, "test@example.com")
	if verified || !errors.Is(err, ErrUnsupportedDesktopVerification) {
		t.Fatalf("VerifyIdentity: expected (false, ErrUnsupportedDesktopVerification), got (%v, %v)", verified, err)
	}
}

// Additional test: Start and Probe when running
func TestWindowsChatGPTLifecycle_StartAndProbeRunning(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")
	sampleData := makeSampleAuthData("probe@example.com", "acc-probe-456")
	if err := os.WriteFile(authPath, sampleData, 0600); err != nil {
		t.Fatalf("failed to write auth data: %v", err)
	}

	now := time.Now()
	procs := []processEntry{
		{
			PID:          20000,
			ParentPID:    1000,
			ExeName:      "ChatGPT.exe",
			ImagePath:    `C:\Program Files\WindowsApps\OpenAI.Codex_26.1027.3384.0_x64__2p2nqsd0c76g0\app\ChatGPT.exe`,
			CreationTime: now.Add(100 * time.Millisecond),
		},
	}

	startCalled := false
	lc := &WindowsChatGPTLifecycle{
		authPath:     authPath,
		pollInterval: 5 * time.Millisecond,
		nowFunc:      func() time.Time { return now },
		startCmdFn: func(ctx context.Context) error {
			startCalled = true
			return nil
		},
		procSnapshotFn: func() ([]processEntry, error) {
			if !startCalled {
				return []processEntry{}, nil
			}
			return procs, nil
		},
	}

	newPID, err := lc.Start(context.Background())
	if err != nil {
		t.Fatalf("unexpected Start error: %v", err)
	}
	if newPID != 20000 {
		t.Fatalf("expected new PID 20000, got %d", newPID)
	}

	// Now probe
	state, err := lc.Probe(context.Background())
	if err != nil {
		t.Fatalf("unexpected Probe error: %v", err)
	}
	if !state.Running || state.PID != 20000 {
		t.Fatalf("expected running with PID 20000, got %v, %d", state.Running, state.PID)
	}
	if state.Email != "probe@example.com" || state.AccountID != "acc-probe-456" {
		t.Fatalf("expected probe@example.com / acc-probe-456, got %s / %s", state.Email, state.AccountID)
	}
	if !state.Ready || !state.LauncherVerified || state.Backend != "file" {
		t.Fatalf("state flags invalid: %+v", state)
	}
}
