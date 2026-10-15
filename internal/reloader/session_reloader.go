package reloader

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"antigravity-switch/internal/connector"

	"golang.org/x/sys/windows"
)

var (
	modUser32                   = windows.NewLazySystemDLL("user32.dll")
	procEnumWindows             = modUser32.NewProc("EnumWindows")
	procGetWindowThreadProcessId = modUser32.NewProc("GetWindowThreadProcessId")
	procGetWindowTextW          = modUser32.NewProc("GetWindowTextW")
	procIsWindowVisible         = modUser32.NewProc("IsWindowVisible")
	procSetForegroundWindow     = modUser32.NewProc("SetForegroundWindow")
	procKeybdEvent              = modUser32.NewProc("keybd_event")
)

const (
	VK_CONTROL       = 0x11
	VK_R             = 0x52
	VK_RETURN        = 0x0D
	KEYEVENTF_KEYUP  = 0x0002
)

var (
	ErrNoWindowsFound   = errors.New("no visible Antigravity windows found")
	ErrInvalidEmail     = errors.New("invalid email address")
	ErrStorageNotFound  = errors.New("app_storage.json not found")
)

// SessionReloader provides window targeting, keystroke dispatch, ToS patching,
// and coordinated reauth for Antigravity sessions.
type SessionReloader struct {
	mu           sync.RWMutex
	PIDFinder    func() ([]int, error)
	WindowFinder func() ([]windows.HWND, error)
	KeySender    func(hwnds []windows.HWND) error
	StoragePath  string
}

// NewSessionReloader creates a default SessionReloader.
func NewSessionReloader() *SessionReloader {
	return &SessionReloader{
		PIDFinder:    FindAntigravityPIDs,
		WindowFinder: FindAntigravityWindows,
		KeySender:    ReloadAntigravityWindow,
	}
}

// FindAntigravityPIDs discovers running PIDs belonging to Antigravity.exe or language_server.exe
// using native Win32 toolhelp process snapshots with zero external process spawning overhead.
func FindAntigravityPIDs() ([]int, error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, fmt.Errorf("failed creating toolhelp snapshot: %w", err)
	}
	defer windows.CloseHandle(snapshot)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))

	if err := windows.Process32First(snapshot, &entry); err != nil {
		return nil, fmt.Errorf("failed reading first process entry: %w", err)
	}

	pids := make([]int, 0)
	antigravityPIDs := make(map[int]bool)

	for {
		exeName := windows.UTF16ToString(entry.ExeFile[:])
		lowerName := strings.ToLower(exeName)

		if strings.Contains(lowerName, "antigravity") {
			pid := int(entry.ProcessID)
			if pid > 0 && !antigravityPIDs[pid] {
				antigravityPIDs[pid] = true
				pids = append(pids, pid)
			}
		}

		err = windows.Process32Next(snapshot, &entry)
		if err != nil {
			if errors.Is(err, windows.ERROR_NO_MORE_FILES) {
				break
			}
			return nil, fmt.Errorf("failed reading next process entry: %w", err)
		}
	}

	return pids, nil
}

// FindAntigravityWindows finds all visible window handles (HWND) belonging to Antigravity.
// It cross-references window PIDs with Antigravity PIDs, with title substring fallback.
func FindAntigravityWindows() ([]windows.HWND, error) {
	pids, _ := FindAntigravityPIDs()
	pidMap := make(map[uint32]bool)
	for _, p := range pids {
		if p > 0 {
			pidMap[uint32(p)] = true
		}
	}

	var found []windows.HWND

	cb := syscall.NewCallback(func(hwnd windows.HWND, lParam uintptr) uintptr {
		// Only check visible windows
		r1, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
		if r1 == 0 {
			return 1 // Continue enumeration
		}

		var processID uint32
		procGetWindowThreadProcessId.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&processID)))

		if len(pidMap) > 0 && pidMap[processID] {
			found = append(found, hwnd)
			return 1
		}

		// Fallback check on window title if PID match didn't catch it
		var titleBuf [512]uint16
		rLen, _, _ := procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&titleBuf[0])), uintptr(len(titleBuf)))
		if rLen > 0 {
			title := strings.ToLower(syscall.UTF16ToString(titleBuf[:]))
			if strings.Contains(title, "antigravity") {
				found = append(found, hwnd)
			}
		}

		return 1
	})

	procEnumWindows.Call(cb, 0)

	return found, nil
}

// ReloadAntigravityWindow dispatches Ctrl+R keybd_event to each target Antigravity window.
func ReloadAntigravityWindow(hwnds []windows.HWND) error {
	if len(hwnds) == 0 {
		return ErrNoWindowsFound
	}

	for _, hwnd := range hwnds {
		procSetForegroundWindow.Call(uintptr(hwnd))
		time.Sleep(50 * time.Millisecond)

		// Key down: Ctrl + R
		procKeybdEvent.Call(uintptr(VK_CONTROL), 0, 0, 0)
		procKeybdEvent.Call(uintptr(VK_R), 0, 0, 0)
		time.Sleep(50 * time.Millisecond)

		// Key up: R, then Ctrl
		procKeybdEvent.Call(uintptr(VK_R), 0, uintptr(KEYEVENTF_KEYUP), 0)
		procKeybdEvent.Call(uintptr(VK_CONTROL), 0, uintptr(KEYEVENTF_KEYUP), 0)
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// GetDefaultAppStoragePath returns the path to Antigravity's app_storage.json.
func GetDefaultAppStoragePath() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return ""
	}
	return filepath.Join(appData, "Antigravity", "app_storage.json")
}

// PrePatchToSAcceptance pre-patches app_storage.json to mark Google Cloud ToS as accepted
// for targetEmail, setting lastLoginUsername and lastLoginIsGcpTos = "false" to prevent modal prompts.
func PrePatchToSAcceptance(targetEmail string, storagePath string) error {
	normEmail := strings.TrimSpace(targetEmail)
	if normEmail == "" {
		return ErrInvalidEmail
	}

	path := storagePath
	if path == "" {
		path = GetDefaultAppStoragePath()
	}
	if path == "" {
		return ErrStorageNotFound
	}

	// Read existing or create empty
	data := make(map[string]interface{})
	if raw, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(raw, &data)
	}

	data["jetski.onboarding.lastLoginUsername"] = normEmail
	data["jetski.onboarding.lastLoginIsGcpTos"] = "false"

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed creating storage directory: %w", err)
	}

	// Atomic write
	tmpFile, err := os.CreateTemp(dir, "app_storage_tmp_*.json")
	if err != nil {
		return fmt.Errorf("failed creating temp file for app_storage: %w", err)
	}
	tmpName := tmpFile.Name()

	enc := json.NewEncoder(tmpFile)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("failed encoding app_storage: %w", err)
	}
	_ = tmpFile.Sync()
	_ = tmpFile.Close()

	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(path)
		if retryErr := os.Rename(tmpName, path); retryErr != nil {
			_ = os.Remove(tmpName)
			return fmt.Errorf("failed moving temp app_storage to target: %w", retryErr)
		}
	}

	return nil
}

// ReloadAntigravitySession performs coordinated session reloading:
// 1. If client is non-nil, triggers loopback ReAuth RPC.
// 2. If reloadWindow is true (or if ReAuth fails), finds HWND and sends Ctrl+R.
func (r *SessionReloader) ReloadAntigravitySession(ctx context.Context, client *connector.Client, reloadWindow bool) error {
	r.mu.RLock()
	windowFinder := r.WindowFinder
	keySender := r.KeySender
	r.mu.RUnlock()

	if windowFinder == nil {
		windowFinder = FindAntigravityWindows
	}
	if keySender == nil {
		keySender = ReloadAntigravityWindow
	}

	var reauthErr error
	reauthSucceeded := false

	if client != nil {
		if err := client.TriggerReauth(ctx); err == nil {
			reauthSucceeded = true
		} else {
			reauthErr = err
		}
	}

	var windowErr error
	if reloadWindow || !reauthSucceeded {
		hwnds, err := windowFinder()
		if err == nil && len(hwnds) > 0 {
			windowErr = keySender(hwnds)
		} else if err != nil {
			windowErr = err
		} else {
			windowErr = ErrNoWindowsFound
		}
	}

	if reauthSucceeded || windowErr == nil {
		return nil
	}

	if reauthErr != nil && windowErr != nil {
		return fmt.Errorf("both reauth (%v) and window reload (%v) failed", reauthErr, windowErr)
	}
	if windowErr != nil {
		return windowErr
	}
	return reauthErr
}
