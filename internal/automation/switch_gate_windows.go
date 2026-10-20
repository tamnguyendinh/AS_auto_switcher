//go:build windows

package automation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"antigravity-switch/internal/manager"
	"golang.org/x/sys/windows"
)

var (
	// ErrGateBusy is returned by TryExecute when the switch gate is locked by another caller or process.
	ErrGateBusy = errors.New("switch gate is currently busy with another operation")

	// ErrNilManager is returned when ExecuteSwitchWithGate receives a nil AccountManager.
	ErrNilManager = errors.New("account manager cannot be nil")
)

// SwitchGate coordinates caller boundary execution across in-process goroutines and Windows processes.
// It wraps an in-process channel semaphore and a Windows Named Mutex with strict OS-thread ownership.
type SwitchGate struct {
	mutexName string
	inProcSem chan struct{}
}

// NewSwitchGate creates a new SwitchGate. If mutexName is omitted or empty, GetDefaultSwitchGateName() is used.
func NewSwitchGate(mutexName ...string) *SwitchGate {
	name := ""
	if len(mutexName) > 0 {
		name = strings.TrimSpace(mutexName[0])
	}
	if name == "" {
		name = GetDefaultSwitchGateName()
	}
	return &SwitchGate{
		mutexName: name,
		inProcSem: make(chan struct{}, 1),
	}
}

// MutexName returns the Windows named mutex identifier for this gate.
func (g *SwitchGate) MutexName() string {
	return g.mutexName
}

// GetDefaultSwitchGateName returns the user-session scoped Windows Named Mutex name.
// It avoids colliding with the GUI single-instance mutex and isolates across user sessions.
func GetDefaultSwitchGateName() string {
	user := strings.TrimSpace(os.Getenv("USERNAME"))
	if user == "" {
		user = "DefaultUser"
	}
	user = strings.ReplaceAll(user, "\\", "_")
	user = strings.ReplaceAll(user, "/", "_")
	return fmt.Sprintf("Local\\AntigravitySwitch_CallerBoundary_%s", user)
}

// Execute acquires the in-process and cross-process gates, binds OS-thread affinity,
// executes action, and guarantees deferred cleanup and release.
// wasAbandoned is passed as true if a previous holder terminated abruptly.
func (g *SwitchGate) Execute(ctx context.Context, action func(wasAbandoned bool) error) error {
	if action == nil {
		return errors.New("action cannot be nil")
	}

	// 1. In-process acquire with context cancellation check
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case g.inProcSem <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-g.inProcSem }()

	// 2. Lock OS thread for Windows mutex ownership requirement
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// 3. Create or open Named Mutex
	namePtr, err := windows.UTF16PtrFromString(g.mutexName)
	if err != nil {
		return fmt.Errorf("invalid mutex name %q: %w", g.mutexName, err)
	}

	handle, err := windows.CreateMutex(nil, false, namePtr)
	if err != nil && !errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		return fmt.Errorf("failed creating named mutex %q: %w", g.mutexName, err)
	}
	defer windows.CloseHandle(handle)

	// 4. Bounded wait slices checking context cancellation
	wasAbandoned := false
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		waitMs := uint32(50)
		event, waitErr := windows.WaitForSingleObject(handle, waitMs)
		if waitErr != nil {
			return fmt.Errorf("WaitForSingleObject failed on mutex %q: %w", g.mutexName, waitErr)
		}

		switch event {
		case windows.WAIT_OBJECT_0:
			defer func() {
				_ = windows.ReleaseMutex(handle)
			}()
			return action(wasAbandoned)

		case windows.WAIT_ABANDONED:
			wasAbandoned = true
			defer func() {
				_ = windows.ReleaseMutex(handle)
			}()
			return action(wasAbandoned)

		case uint32(windows.WAIT_TIMEOUT):
			continue

		default:
			return fmt.Errorf("unexpected wait event %d on mutex %q", event, g.mutexName)
		}
	}
}

// TryExecute attempts to acquire the gate immediately without blocking.
// Returns (true, actionErr) if acquired and executed, or (false, ErrGateBusy) if currently held.
func (g *SwitchGate) TryExecute(action func(wasAbandoned bool) error) (bool, error) {
	if action == nil {
		return false, errors.New("action cannot be nil")
	}

	select {
	case g.inProcSem <- struct{}{}:
	default:
		return false, ErrGateBusy
	}
	defer func() { <-g.inProcSem }()

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	namePtr, err := windows.UTF16PtrFromString(g.mutexName)
	if err != nil {
		return false, fmt.Errorf("invalid mutex name %q: %w", g.mutexName, err)
	}

	handle, err := windows.CreateMutex(nil, false, namePtr)
	if err != nil && !errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		return false, fmt.Errorf("failed creating named mutex %q: %w", g.mutexName, err)
	}
	defer windows.CloseHandle(handle)

	event, waitErr := windows.WaitForSingleObject(handle, 0)
	if waitErr != nil {
		return false, fmt.Errorf("WaitForSingleObject failed on mutex %q: %w", g.mutexName, waitErr)
	}

	wasAbandoned := false
	switch event {
	case windows.WAIT_OBJECT_0:
		defer func() {
			_ = windows.ReleaseMutex(handle)
		}()
		return true, action(wasAbandoned)

	case windows.WAIT_ABANDONED:
		wasAbandoned = true
		defer func() {
			_ = windows.ReleaseMutex(handle)
		}()
		return true, action(wasAbandoned)

	case uint32(windows.WAIT_TIMEOUT):
		return false, ErrGateBusy

	default:
		return false, fmt.Errorf("unexpected wait event %d on mutex %q", event, g.mutexName)
	}
}

// ExecuteSwitchWithGate is the canonical shared switch wrapper used by all callers
// (Auto controller, GUI manual switch, and CLI --switch).
// It acquires the gate, handles abandoned recovery, calls mgr.SwitchAccount, and releases the gate.
func ExecuteSwitchWithGate(
	ctx context.Context,
	gate *SwitchGate,
	mgr *manager.AccountManager,
	targetEmail string,
	forceLogout ...bool,
) (*manager.SwitchResult, error) {
	if gate == nil {
		gate = NewSwitchGate()
	}
	if mgr == nil {
		return nil, ErrNilManager
	}

	var result *manager.SwitchResult
	var switchErr error

	fl := false
	if len(forceLogout) > 0 {
		fl = forceLogout[0]
	}

	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	_, err := gate.TryExecute(func(wasAbandoned bool) error {
		// If the lock was abandoned by a previous process crash, re-verify the active email
		if wasAbandoned {
			current, _ := mgr.GetCurrentActiveEmail(ctx)
			if current != "" && strings.EqualFold(current, targetEmail) {
				result = &manager.SwitchResult{
					Success:       true,
					Message:       fmt.Sprintf("Tài khoản %s hiện đang hoạt động sau khi phục hồi lock bỏ rơi.", targetEmail),
					TargetEmail:   targetEmail,
					PreviousEmail: current,
				}
				return nil
			}
		}

		result, switchErr = mgr.SwitchAccount(ctx, targetEmail, fl)
		return switchErr
	})

	if err != nil && result == nil {
		return nil, err
	}
	return result, switchErr
}
