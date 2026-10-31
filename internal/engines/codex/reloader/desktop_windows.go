package reloader

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"as-auto-switcher/internal/engines/codex/model"

	"golang.org/x/sys/windows"
)

var (
	modUser32                    = windows.NewLazySystemDLL("user32.dll")
	procEnumWindows              = modUser32.NewProc("EnumWindows")
	procGetWindowThreadProcessId = modUser32.NewProc("GetWindowThreadProcessId")
	procIsWindowVisible          = modUser32.NewProc("IsWindowVisible")
	procPostMessageW             = modUser32.NewProc("PostMessageW")
)

const (
	WM_CLOSE           = 0x0010
	CodexAppAUMID      = "OpenAI.Codex_2p2nqsd0c76g0!App"
	CodexPackagePrefix = "OpenAI.Codex_"
)

var enumWindowsCloseCb = syscall.NewCallback(func(hwnd uintptr, lparam uintptr) uintptr {
	targetPID := int(lparam)
	var windowPID uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&windowPID)))
	if int(windowPID) == targetPID {
		ret, _, _ := procIsWindowVisible.Call(hwnd)
		if ret != 0 {
			procPostMessageW.Call(hwnd, WM_CLOSE, 0, 0)
		}
	}
	return 1
})

type processEntry struct {
	PID          uint32
	ParentPID    uint32
	ExeName      string
	ImagePath    string
	CreationTime time.Time
}

// WindowsChatGPTLifecycle implements VerifiedDesktop for ChatGPT Desktop on Windows.
type WindowsChatGPTLifecycle struct {
	authPath       string
	quiescentDelay time.Duration
	pollInterval   time.Duration

	// Internal hooks for testability and isolation
	procSnapshotFn func() ([]processEntry, error)
	treeKillFn     func(ctx context.Context, pid int) error
	quiescentFn    func(ctx context.Context, path string) error
	startCmdFn     func(ctx context.Context) error
	nowFunc        func() time.Time
}

var (
	_ VerifiedDesktop  = (*WindowsChatGPTLifecycle)(nil)
	_ DesktopLifecycle = (*WindowsChatGPTLifecycle)(nil)
	_ DesktopLifecycle = (*DefaultWindowsDesktopLifecycle)(nil)
)

// NewWindowsChatGPTLifecycle creates a verified desktop lifecycle manager for ChatGPT Desktop.
// If authPath is empty, it falls back to ~/.codex/auth.json.
func NewWindowsChatGPTLifecycle(authPath string) VerifiedDesktop {
	trimmed := strings.TrimSpace(authPath)
	if trimmed == "" {
		home, err := os.UserHomeDir()
		if err == nil && home != "" {
			trimmed = filepath.Join(home, ".codex", "auth.json")
		} else {
			trimmed = filepath.Join(os.Getenv("USERPROFILE"), ".codex", "auth.json")
		}
	}
	return &WindowsChatGPTLifecycle{
		authPath:       filepath.Clean(trimmed),
		quiescentDelay: 1 * time.Second,
		pollInterval:   300 * time.Millisecond,
	}
}

func (l *WindowsChatGPTLifecycle) defaultSnapshot() ([]processEntry, error) {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(snap)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	if err := windows.Process32First(snap, &entry); err != nil {
		return nil, err
	}

	var results []processEntry
	for {
		exeName := windows.UTF16ToString(entry.ExeFile[:])
		if strings.EqualFold(exeName, "chatgpt.exe") {
			pid := entry.ProcessID
			ppid := entry.ParentProcessID

			var imagePath string
			var creationTime time.Time

			hProcess, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
			if err == nil {
				buf := make([]uint16, 1024)
				size := uint32(len(buf))
				if err := windows.QueryFullProcessImageName(hProcess, 0, &buf[0], &size); err == nil {
					imagePath = windows.UTF16ToString(buf[:size])
				}
				var ct, et, kt, ut windows.Filetime
				if err := windows.GetProcessTimes(hProcess, &ct, &et, &kt, &ut); err == nil {
					creationTime = time.Unix(0, ct.Nanoseconds())
				}
				_ = windows.CloseHandle(hProcess)
			}

			results = append(results, processEntry{
				PID:          pid,
				ParentPID:    ppid,
				ExeName:      exeName,
				ImagePath:    imagePath,
				CreationTime: creationTime,
			})
		}

		if err := windows.Process32Next(snap, &entry); err != nil {
			break
		}
	}
	return results, nil
}

func (l *WindowsChatGPTLifecycle) findRootProcess() (int, time.Time, error) {
	var procs []processEntry
	var err error
	if l.procSnapshotFn != nil {
		procs, err = l.procSnapshotFn()
	} else {
		procs, err = l.defaultSnapshot()
	}
	if err != nil {
		return 0, time.Time{}, err
	}

	chatgptPIDs := make(map[uint32]bool)
	var candidates []processEntry
	for _, p := range procs {
		if !strings.EqualFold(p.ExeName, "chatgpt.exe") {
			continue
		}
		if p.ImagePath != "" && !strings.Contains(p.ImagePath, CodexPackagePrefix) {
			continue
		}
		chatgptPIDs[p.PID] = true
		candidates = append(candidates, p)
	}

	if len(candidates) == 0 {
		return 0, time.Time{}, nil
	}

	for _, c := range candidates {
		if !chatgptPIDs[c.ParentPID] {
			return int(c.PID), c.CreationTime, nil
		}
	}

	return int(candidates[0].PID), candidates[0].CreationTime, nil
}

func (l *WindowsChatGPTLifecycle) isPIDAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	if l.procSnapshotFn != nil {
		procs, err := l.procSnapshotFn()
		if err != nil {
			return false
		}
		for _, p := range procs {
			if int(p.PID) == pid {
				return true
			}
		}
		return false
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)
	var code uint32
	if err := windows.GetExitCodeProcess(h, &code); err != nil {
		return false
	}
	return code == 259 // STILL_ACTIVE = 259
}

// IsRunning detects if the ChatGPT Desktop root process is currently active.
func (l *WindowsChatGPTLifecycle) IsRunning(ctx context.Context) (bool, int, error) {
	rootPID, _, err := l.findRootProcess()
	if err != nil {
		return false, 0, err
	}
	if rootPID > 0 {
		return true, rootPID, nil
	}
	return false, 0, nil
}

// Stop shuts down the ChatGPT Desktop process tree cleanly: graceful close first, followed by tree-kill.
func (l *WindowsChatGPTLifecycle) Stop(ctx context.Context, pid int) error {
	if pid <= 0 {
		return nil
	}

	if l.treeKillFn != nil {
		return l.treeKillFn(ctx, pid)
	}

	// 1. Graceful close
	procEnumWindows.Call(enumWindowsCloseCb, uintptr(pid))

	hProcess, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if err == nil {
		_, _ = windows.WaitForSingleObject(hProcess, 3000)
		_ = windows.CloseHandle(hProcess)
	}

	// 2. Force tree kill
	cmd := exec.CommandContext(ctx, "taskkill", "/T", "/F", "/PID", strconv.Itoa(pid))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = cmd.Run()

	// Wait up to 2 seconds for process exit
	pollInterval := l.pollInterval
	if pollInterval <= 0 {
		pollInterval = 100 * time.Millisecond
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !l.isPIDAlive(pid) {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
	}

	if l.isPIDAlive(pid) {
		return ErrDesktopStopTimeout
	}
	return nil
}

// Quiescent ensures no active writers or locks exist on the auth file before mutations.
func (l *WindowsChatGPTLifecycle) Quiescent(ctx context.Context, path string) error {
	if l.quiescentFn != nil {
		return l.quiescentFn(ctx, path)
	}

	delay := l.quiescentDelay
	if delay <= 0 {
		delay = 1 * time.Second
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(delay):
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}

	h, err := windows.CreateFile(
		pathPtr,
		windows.GENERIC_READ,
		0, // dwShareMode = 0: exclusive access
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		if errors.Is(err, windows.ERROR_FILE_NOT_FOUND) || errors.Is(err, windows.ERROR_PATH_NOT_FOUND) {
			return nil
		}
		if errors.Is(err, windows.ERROR_SHARING_VIOLATION) || errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
			return ErrAuthHomeBusy
		}
		return err
	}
	_ = windows.CloseHandle(h)
	return nil
}

// Start launches the ChatGPT Desktop application via official Windows Shell AUMID.
func (l *WindowsChatGPTLifecycle) Start(ctx context.Context) (int, error) {
	startTime := time.Now()
	if l.nowFunc != nil {
		startTime = l.nowFunc()
	}

	if l.startCmdFn != nil {
		if err := l.startCmdFn(ctx); err != nil {
			return 0, fmt.Errorf("%w: %v", ErrDesktopStartFailed, err)
		}
	} else {
		cmd := exec.CommandContext(ctx, "cmd.exe", "/c", "start", "", "shell:AppsFolder\\"+CodexAppAUMID)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		if err := cmd.Run(); err != nil {
			return 0, fmt.Errorf("%w: failed to start via AUMID: %v", ErrDesktopStartFailed, err)
		}
	}

	pollInterval := l.pollInterval
	if pollInterval <= 0 {
		pollInterval = 300 * time.Millisecond
	}

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}

		rootPID, creationTime, err := l.findRootProcess()
		if err == nil && rootPID > 0 {
			if creationTime.IsZero() || creationTime.After(startTime.Add(-2*time.Second)) {
				return rootPID, nil
			}
		}

		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(pollInterval):
		}
	}

	return 0, ErrDesktopStartFailed
}

// Probe inspects and returns the current DesktopState.
func (l *WindowsChatGPTLifecycle) Probe(ctx context.Context) (DesktopState, error) {
	rootPID, creationTime, err := l.findRootProcess()
	if err != nil {
		return DesktopState{}, err
	}

	if rootPID == 0 {
		return DesktopState{
			Running:          false,
			Backend:          "file",
			AuthPath:         l.authPath,
			LauncherVerified: true,
			BusyKnown:        true,
			Busy:             false,
			WritersSafe:      true,
			Ready:            true,
		}, nil
	}

	data, err := os.ReadFile(l.authPath)
	if err != nil {
		return DesktopState{}, fmt.Errorf("failed to read auth file: %w", err)
	}

	email, accountID, err := model.ValidateAuth(data)
	if err != nil {
		return DesktopState{}, fmt.Errorf("failed to validate auth data: %w", err)
	}

	return DesktopState{
		Running:          true,
		PID:              rootPID,
		ProcessKey:       fmt.Sprintf("%d:%d", rootPID, creationTime.UnixNano()),
		Email:            email,
		AccountID:        accountID,
		Ready:            true,
		BusyKnown:        true,
		Busy:             false,
		WritersSafe:      true,
		LauncherVerified: true,
		Backend:          "file",
		AuthPath:         l.authPath,
	}, nil
}

// VerifyAccount verifies that the target process is alive and that authPath matches the expected email and accountID.
func (l *WindowsChatGPTLifecycle) VerifyAccount(ctx context.Context, pid int, email, accountID string) (bool, error) {
	pollInterval := l.pollInterval
	if pollInterval <= 0 {
		pollInterval = 300 * time.Millisecond
	}

	for {
		select {
		case <-ctx.Done():
			return false, nil
		default:
		}

		if l.isPIDAlive(pid) {
			if data, err := os.ReadFile(l.authPath); err == nil {
				if fileEmail, fileAccountID, err := model.ValidateAuth(data); err == nil {
					if strings.EqualFold(fileEmail, email) && (accountID == "" || fileAccountID == accountID) {
						return true, nil
					}
				}
			}
		}

		select {
		case <-ctx.Done():
			return false, nil
		case <-time.After(pollInterval):
		}
	}
}

// VerifyIdentity delegates directly to VerifyAccount with accountID == "".
func (l *WindowsChatGPTLifecycle) VerifyIdentity(ctx context.Context, pid int, email string) (bool, error) {
	return l.VerifyAccount(ctx, pid, email, "")
}

// DefaultWindowsDesktopLifecycle deliberately exposes no process mutation.
// This installed Desktop has no proven process-bound identity/busy channel in
// the integration. v8.1 requires UnsupportedDesktopVerification in this case.
// A CLI PID or auth.json is not an observation of Desktop state.
type DefaultWindowsDesktopLifecycle struct{}

func NewDefaultWindowsDesktopLifecycle(_ string) DesktopLifecycle {
	return &DefaultWindowsDesktopLifecycle{}
}
func (*DefaultWindowsDesktopLifecycle) IsRunning(context.Context) (bool, int, error) {
	return false, 0, ErrUnsupportedDesktopVerification
}
func (*DefaultWindowsDesktopLifecycle) Stop(context.Context, int) error {
	return ErrUnsupportedDesktopVerification
}
func (*DefaultWindowsDesktopLifecycle) Start(context.Context) (int, error) {
	return 0, ErrUnsupportedDesktopVerification
}
func (*DefaultWindowsDesktopLifecycle) VerifyIdentity(context.Context, int, string) (bool, error) {
	return false, ErrUnsupportedDesktopVerification
}
