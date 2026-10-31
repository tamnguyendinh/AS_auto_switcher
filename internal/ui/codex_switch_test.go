package ui

import (
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	codexmanager "as-auto-switcher/internal/engines/codex/manager"
	codexmodel "as-auto-switcher/internal/engines/codex/model"
	codexreloader "as-auto-switcher/internal/engines/codex/reloader"
)

type mockSwitchAccountManager struct {
	bootstrapActiveSessionFn func() error
	listCardsFn              func() ([]*codexmanager.CodexCardView, error)
	switchAccountFn          func(email string) error
	deleteAccountFn          func(email string) error
	toggleStatusFn           func(email string) error
	getActiveEmailFn         func() (string, error)
}

func (m *mockSwitchAccountManager) BootstrapActiveSession() error {
	if m.bootstrapActiveSessionFn != nil {
		return m.bootstrapActiveSessionFn()
	}
	return nil
}

func (m *mockSwitchAccountManager) ListCards() ([]*codexmanager.CodexCardView, error) {
	if m.listCardsFn != nil {
		return m.listCardsFn()
	}
	return nil, nil
}

func (m *mockSwitchAccountManager) SwitchAccount(email string) error {
	if m.switchAccountFn != nil {
		return m.switchAccountFn(email)
	}
	return nil
}

func (m *mockSwitchAccountManager) DeleteAccount(email string) error {
	if m.deleteAccountFn != nil {
		return m.deleteAccountFn(email)
	}
	return nil
}

func (m *mockSwitchAccountManager) ToggleStatus(email string) error {
	if m.toggleStatusFn != nil {
		return m.toggleStatusFn(email)
	}
	return nil
}

func (m *mockSwitchAccountManager) GetActiveEmail() (string, error) {
	if m.getActiveEmailFn != nil {
		return m.getActiveEmailFn()
	}
	return "", nil
}

func setupTestApp(fa fyne.App) *App {
	a := NewAppWithFyne(fa, nil)
	a.statusBar = widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})
	a.codex = newCodexView(DarkPalette, func() {})
	return a
}

func readStatusBarText(a *App) string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.statusBar != nil {
		return a.statusBar.Text
	}
	return ""
}

// TestHandleCodexSwitch_AsyncAndAntiDoubleClick verifies:
// 1. handleCodexSwitch dispatches asynchronously in a goroutine without blocking UI.
// 2. Concurrency Guard: concurrent switch requests while isBusy == true are dropped.
// 3. isBusy flag is reliably reset to false upon completion via goroutine-local defer.
func TestHandleCodexSwitch_AsyncAndAntiDoubleClick(t *testing.T) {
	fa := test.NewApp()
	defer fa.Quit()

	a := setupTestApp(fa)
	defer a.Close()

	started := make(chan struct{})
	proceed := make(chan struct{})
	var callCount atomic.Int32

	mockMgr := &mockSwitchAccountManager{
		switchAccountFn: func(email string) error {
			callCount.Add(1)
			close(started)
			<-proceed
			return nil
		},
	}
	a.SetCodexServices(mockMgr, nil, nil)

	// 1. Invoke first switch
	a.handleCodexSwitch("account1@example.com")

	// Wait until background goroutine enters SwitchAccount
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for background SwitchAccount to start")
	}

	// 2. Verify isBusy lock guard is active
	a.mu.RLock()
	busy := a.isBusy
	a.mu.RUnlock()
	if !busy {
		t.Fatal("expected a.isBusy to be true while switch is running")
	}

	// 3. Attempt concurrent second switch call - must be dropped immediately
	a.handleCodexSwitch("account2@example.com")

	if count := callCount.Load(); count != 1 {
		t.Fatalf("expected anti-double-click guard to drop request; actual call count: %d", count)
	}

	// 4. Release background goroutine
	close(proceed)

	// Wait for goroutine defer to unlock isBusy
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		a.mu.RLock()
		b := a.isBusy
		a.mu.RUnlock()
		if !b {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	a.mu.RLock()
	finalBusy := a.isBusy
	a.mu.RUnlock()
	if finalBusy {
		t.Fatal("expected a.isBusy to revert to false after switch finishes")
	}
	if count := callCount.Load(); count != 1 {
		t.Fatalf("expected total call count to remain 1, got %d", count)
	}
}

// TestHandleCodexSwitch_PrivacyMasking verifies:
// 1. When SetMaskAccounts(true), status bar displays masked email during and after switch.
// 2. When SetMaskAccounts(false), status bar displays raw email.
func TestHandleCodexSwitch_PrivacyMasking(t *testing.T) {
	fa := test.NewApp()
	defer fa.Quit()

	t.Run("Masked", func(t *testing.T) {
		a := setupTestApp(fa)
		defer a.Close()
		a.SetMaskAccounts(true)

		var capturedStartingMsg string
		var mu sync.Mutex

		inFlight := make(chan struct{})
		mockMgr := &mockSwitchAccountManager{
			switchAccountFn: func(email string) error {
				deadline := time.Now().Add(1 * time.Second)
				for time.Now().Before(deadline) {
					txt := readStatusBarText(a)
					if txt != "" {
						mu.Lock()
						capturedStartingMsg = txt
						mu.Unlock()
						break
					}
					time.Sleep(10 * time.Millisecond)
				}
				close(inFlight)
				return nil
			},
		}
		a.SetCodexServices(mockMgr, nil, nil)

		a.handleCodexSwitch("developer@openai.com")
		<-inFlight
		waitForIdle(a, 2*time.Second)

		mu.Lock()
		startMsg := capturedStartingMsg
		mu.Unlock()

		expectedMasked := "d***@openai.com"
		if !strings.Contains(startMsg, expectedMasked) {
			t.Errorf("expected starting status bar to contain masked email %q, got: %q", expectedMasked, startMsg)
		}
		if strings.Contains(startMsg, "developer@openai.com") {
			t.Errorf("starting status bar leaked raw email: %q", startMsg)
		}

		finalMsg := readStatusBarText(a)
		if !strings.Contains(finalMsg, expectedMasked) {
			t.Errorf("expected completion status bar to contain masked email %q, got: %q", expectedMasked, finalMsg)
		}
		if strings.Contains(finalMsg, "developer@openai.com") {
			t.Errorf("completion status bar leaked raw email: %q", finalMsg)
		}
	})

	t.Run("Unmasked", func(t *testing.T) {
		a := setupTestApp(fa)
		defer a.Close()
		a.SetMaskAccounts(false)

		var capturedStartingMsg string
		var mu sync.Mutex

		inFlight := make(chan struct{})
		mockMgr := &mockSwitchAccountManager{
			switchAccountFn: func(email string) error {
				deadline := time.Now().Add(1 * time.Second)
				for time.Now().Before(deadline) {
					txt := readStatusBarText(a)
					if txt != "" {
						mu.Lock()
						capturedStartingMsg = txt
						mu.Unlock()
						break
					}
					time.Sleep(10 * time.Millisecond)
				}
				close(inFlight)
				return nil
			},
		}
		a.SetCodexServices(mockMgr, nil, nil)

		rawEmail := "developer@openai.com"
		a.handleCodexSwitch(rawEmail)
		<-inFlight
		waitForIdle(a, 2*time.Second)

		mu.Lock()
		startMsg := capturedStartingMsg
		mu.Unlock()

		if !strings.Contains(startMsg, rawEmail) {
			t.Errorf("expected starting status bar to contain raw email %q, got: %q", rawEmail, startMsg)
		}

		finalMsg := readStatusBarText(a)
		if !strings.Contains(finalMsg, rawEmail) {
			t.Errorf("expected completion status bar to contain raw email %q, got: %q", rawEmail, finalMsg)
		}
	})
}

// TestHandleCodexSwitch_SuccessRefreshesCardsAndAutoHides verifies:
// 1. Successful SwitchAccount triggers RefreshCodexCards.
// 2. Status bar displays "✅ <email>".
// 3. autoHideStatusBar is scheduled and automatically clears the status bar after timeout.
func TestHandleCodexSwitch_SuccessRefreshesCardsAndAutoHides(t *testing.T) {
	fa := test.NewApp()
	defer fa.Quit()

	a := setupTestApp(fa)
	defer a.Close()
	a.SetMaskAccounts(false)

	var listCardsCalled atomic.Bool
	mockMgr := &mockSwitchAccountManager{
		switchAccountFn: func(email string) error {
			return nil
		},
		listCardsFn: func() ([]*codexmanager.CodexCardView, error) {
			listCardsCalled.Store(true)
			return []*codexmanager.CodexCardView{
				{Email: "user@example.com", State: codexmodel.StateActive, Status: "on"},
			}, nil
		},
	}
	a.SetCodexServices(mockMgr, nil, nil)

	a.handleCodexSwitch("user@example.com")
	waitForIdle(a, 2*time.Second)

	// 1. RefreshCodexCards was invoked
	if !listCardsCalled.Load() {
		t.Fatal("expected RefreshCodexCards to be called on success")
	}

	// 2. Status bar has checkmark and email
	sbText := readStatusBarText(a)
	if !strings.Contains(sbText, "✅ user@example.com") {
		t.Fatalf("expected status bar text to contain '✅ user@example.com', got: %q", sbText)
	}

	// 3. autoHideStatusBar timer is active
	a.mu.RLock()
	timerActive := a.statusBarHideTimer != nil
	a.mu.RUnlock()
	if !timerActive {
		t.Fatal("expected statusBarHideTimer to be scheduled")
	}

	// 4. Status bar automatically clears after the 3-second timeout
	time.Sleep(3200 * time.Millisecond)
	if txt := readStatusBarText(a); txt != "" {
		t.Fatalf("expected status bar to auto-hide after 3 seconds, still: %q", txt)
	}
}

// TestHandleCodexSwitch_ErrorDisplayAndDialog verifies:
// 1. Simulated SwitchAccount error displays ❌ on the status bar.
// 2. Error dialogs (both generic and ErrUnsupportedDesktopVerification) do not crash the app.
func TestHandleCodexSwitch_ErrorDisplayAndDialog(t *testing.T) {
	fa := test.NewApp()
	defer fa.Quit()

	t.Run("StandardError", func(t *testing.T) {
		a := setupTestApp(fa)
		win := fa.NewWindow("Test")
		a.win = win
		defer a.Close()

		expectedErr := errors.New("network timeout: failed to connect to auth server")
		mockMgr := &mockSwitchAccountManager{
			switchAccountFn: func(email string) error {
				return expectedErr
			},
		}
		a.SetCodexServices(mockMgr, nil, nil)

		a.handleCodexSwitch("err@example.com")
		waitForIdle(a, 2*time.Second)

		sbText := readStatusBarText(a)
		if !strings.Contains(sbText, "❌") || !strings.Contains(sbText, "network timeout") {
			t.Fatalf("expected status bar to contain '❌ network timeout', got: %q", sbText)
		}
	})

	t.Run("UnsupportedDesktopVerification", func(t *testing.T) {
		a := setupTestApp(fa)
		win := fa.NewWindow("Test")
		a.win = win
		defer a.Close()

		mockMgr := &mockSwitchAccountManager{
			switchAccountFn: func(email string) error {
				return codexreloader.ErrUnsupportedDesktopVerification
			},
		}
		a.SetCodexServices(mockMgr, nil, nil)

		a.handleCodexSwitch("unsupported@example.com")
		waitForIdle(a, 2*time.Second)

		sbText := readStatusBarText(a)
		if !strings.Contains(sbText, "❌") || !strings.Contains(sbText, "UnsupportedDesktopVerification") {
			t.Fatalf("expected status bar to contain '❌ UnsupportedDesktopVerification', got: %q", sbText)
		}
	})
}

// TestInitDefaultCodexEngine_WiresStatusReporter verifies:
// 1. InitDefaultCodexEngine wires StatusReporter into SessionReloader.
// 2. Invoking StatusReporter callback safely dispatches to a.statusBar on the UI loop.
func TestInitDefaultCodexEngine_WiresStatusReporter(t *testing.T) {
	fa := test.NewApp()
	defer fa.Quit()

	t.Setenv("CODEX_STORE_DIR", t.TempDir())

	a := setupTestApp(fa)
	defer a.Close()

	if err := a.InitDefaultCodexEngine(); err != nil {
		t.Fatalf("InitDefaultCodexEngine failed: %v", err)
	}

	if a.CodexReloader() == nil {
		t.Fatal("expected codexReloader to be initialized")
	}

	reporter := a.CodexStatusReporter()
	if reporter == nil {
		t.Fatal("expected codexStatusReporter callback to be wired and non-nil")
	}

	// Trigger StatusReporter with sample retry message
	retryMsg := "Đang thử lại kết nối phiên Codex (1/3)..."
	reporter(retryMsg)

	// Allow fyne.Do to dispatch
	deadline := time.Now().Add(1 * time.Second)
	updated := false
	for time.Now().Before(deadline) {
		if readStatusBarText(a) == retryMsg {
			updated = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if !updated {
		t.Fatalf("expected statusBar text to be %q, got: %q", retryMsg, readStatusBarText(a))
	}
}
