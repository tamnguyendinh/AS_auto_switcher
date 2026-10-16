package reloader

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
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
)

var (
	ErrNoWindowsFound        = errors.New("no visible Antigravity windows found")
	ErrInvalidEmail          = errors.New("invalid email address")
	ErrStorageNotFound       = errors.New("app_storage.json not found")
	ErrLanguageServerRestart = errors.New("failed discovering respawned language_server.exe")
)

// SessionReloader provides process lifecycle management, window targeting,
// ToS patching, and language server recycling for Antigravity sessions.
type SessionReloader struct {
	mu               sync.RWMutex
	PIDFinder        func() ([]int, error)
	WindowFinder     func() ([]windows.HWND, error)
	ProcessKiller    func(ctx context.Context) error
	ServerDiscoverer func() (*connector.ServerInfo, error)
	Sleeper          func(ctx context.Context, d time.Duration) error
	StoragePath      string
	BrainDir         string
	OutputDir        string

	SnapshotCapturer   func(storagePath, brainDir, outputDir string) (string, []ActiveThread, error)
	ContinuationRunner func(ctx context.Context, client *connector.Client, activeThreads []ActiveThread, promptText string) error
}

// NewSessionReloader creates a default SessionReloader.
func NewSessionReloader() *SessionReloader {
	return &SessionReloader{
		PIDFinder:        FindAntigravityPIDs,
		WindowFinder:     FindAntigravityWindows,
		ProcessKiller:    DefaultKillLanguageServer,
		ServerDiscoverer: connector.DiscoverLanguageServer,
		Sleeper:          defaultSleep,
	}
}

var defaultReloader = NewSessionReloader()

// DefaultKillLanguageServer terminates running language_server.exe processes using Windows taskkill.
// Exit code errors (e.g. process was not running) are safely ignored matching Python check=False parity.
func DefaultKillLanguageServer(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "taskkill", "/F", "/IM", "language_server.exe")
	cmd.SysProcAttr = &windows.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NO_WINDOW,
	}
	_ = cmd.Run() // Ignore exit code if process was already dead
	return nil
}

func defaultSleep(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

// RecycleLanguageServer terminates language_server.exe cleanly and waits waitDuration
// for Antigravity's monitorLsCrashInternal to respawn a fresh instance with new credentials.
func RecycleLanguageServer(ctx context.Context, waitDuration time.Duration) error {
	return defaultReloader.RecycleLanguageServer(ctx, waitDuration)
}

// RecycleLanguageServer terminates language_server.exe cleanly and waits for Antigravity's
// monitorLsCrashInternal to automatically respawn a fresh instance with new credentials.
// It verifies the new process is alive with a valid PID and loopback port via connector.DiscoverLanguageServer.
func (r *SessionReloader) RecycleLanguageServer(ctx context.Context, waitDuration time.Duration) error {
	if r == nil {
		return defaultReloader.RecycleLanguageServer(ctx, waitDuration)
	}

	r.mu.RLock()
	killer := r.ProcessKiller
	discoverer := r.ServerDiscoverer
	sleeper := r.Sleeper
	r.mu.RUnlock()

	if killer == nil {
		killer = DefaultKillLanguageServer
	}
	if discoverer == nil {
		discoverer = connector.DiscoverLanguageServer
	}
	if sleeper == nil {
		sleeper = defaultSleep
	}

	// 1. Terminate language_server.exe
	if err := killer(ctx); err != nil {
		return fmt.Errorf("failed executing taskkill for language_server.exe: %w", err)
	}

	// 2. Wait for Antigravity parent (monitorLsCrashInternal) to respawn fresh instance
	if waitDuration <= 0 {
		waitDuration = 2 * time.Second
	}
	if err := sleeper(ctx, waitDuration); err != nil {
		return err
	}

	// 3. Confirm new language_server.exe has spawned with valid PID and loopback port
	info, err := discoverer()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrLanguageServerRestart, err)
	}
	if info == nil || info.PID <= 0 || info.Port <= 0 {
		return fmt.Errorf("%w: invalid server info (pid=%d, port=%d)", ErrLanguageServerRestart, nilPid(info), nilPort(info))
	}

	return nil
}

func nilPid(i *connector.ServerInfo) int {
	if i == nil {
		return 0
	}
	return i.PID
}

func nilPort(i *connector.ServerInfo) int {
	if i == nil {
		return 0
	}
	return i.Port
}

// FindAntigravityPIDs discovers running PIDs belonging to Antigravity.exe or language_server.exe
// using native Win32 toolhelp process snapshots with zero external process spawning overhead.
func FindAntigravityPIDs() ([]int, error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, fmt.Errorf("error creating toolhelp snapshot: %w", err)
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

// EnsureAntigravityForeground brings the primary visible Antigravity window to the foreground.
func EnsureAntigravityForeground() error {
	hwnds, err := FindAntigravityWindows()
	if err != nil || len(hwnds) == 0 {
		return ErrNoWindowsFound
	}
	procSetForegroundWindow.Call(uintptr(hwnds[0]))
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

// PrepatchToS pre-patches app_storage.json to mark Google Cloud ToS as accepted for targetEmail.
func PrepatchToS(targetEmail string) error {
	return PrePatchToSAcceptance(targetEmail, "")
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

// CaptureActiveThreadsSnapshot captures active conversation threads from Antigravity's layout state.
func (r *SessionReloader) CaptureActiveThreadsSnapshot(storagePath, brainDir, outputDir string) (string, []ActiveThread, error) {
	if r == nil {
		return defaultReloader.CaptureActiveThreadsSnapshot(storagePath, brainDir, outputDir)
	}
	r.mu.RLock()
	capturer := r.SnapshotCapturer
	if storagePath == "" {
		storagePath = r.StoragePath
	}
	if brainDir == "" {
		brainDir = r.BrainDir
	}
	if outputDir == "" {
		outputDir = r.OutputDir
	}
	r.mu.RUnlock()

	if capturer != nil {
		return capturer(storagePath, brainDir, outputDir)
	}

	return defaultCaptureActiveThreadsSnapshot(storagePath, brainDir, outputDir)
}

// CaptureActiveThreadsSnapshot is the package-level entry point delegating to defaultReloader.
func CaptureActiveThreadsSnapshot(storagePath, brainDir, outputDir string) (string, []ActiveThread, error) {
	return defaultReloader.CaptureActiveThreadsSnapshot(storagePath, brainDir, outputDir)
}

// ReloadAntigravitySession is retained as a compatibility bridge, delegating to RecycleLanguageServer.
func (r *SessionReloader) ReloadAntigravitySession(ctx context.Context, client *connector.Client, reloadWindow bool) error {
	return r.RecycleLanguageServer(ctx, 2*time.Second)
}
