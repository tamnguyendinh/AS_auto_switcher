package ui

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"as-auto-switcher/internal/engines/antigravity/connector"
	"as-auto-switcher/internal/engines/antigravity/manager"
	"as-auto-switcher/internal/engines/antigravity/model"
	"as-auto-switcher/internal/engines/antigravity/wincred"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func waitSessionSync(t *testing.T, a *App) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		a.mu.RLock()
		busy := a.sessionSyncRunning || a.sessionSyncPending || a.isBusy
		a.mu.RUnlock()
		if !busy {
			time.Sleep(15 * time.Millisecond)
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("session sync did not become idle")
}

func waitModalMutation(t *testing.T, button *widget.Button) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for button.Visible() && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if button.Visible() {
		t.Fatal("modal mutation did not finish")
	}
}

func newSessionTestApp(t *testing.T) *App {
	t.Helper()
	f := test.NewApp()
	t.Cleanup(f.Quit)
	a := newIsolatedApp(f, createTestManager(t))
	t.Cleanup(a.Close)
	a.SetLivenessInterval(time.Hour)
	a.Init()
	waitSessionSync(t, a)
	a.clearStatusBar()
	return a
}

func TestSessionSyncIdentityChangesWithoutPIDChange(t *testing.T) {
	a := newSessionTestApp(t)
	var live atomic.Value
	live.Store("first@example.com")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"userStatus":{"email":"` + live.Load().(string) + `"}}`))
	}))
	defer server.Close()
	u, _ := url.Parse(server.URL)
	port, _ := strconv.Atoi(u.Port())
	info := &connector.ServerInfo{PID: 123, Port: port, CSRFToken: "fixture"}
	a.mgr.SetClientFinder(func(context.Context) (*connector.Client, error) {
		return connector.NewClientWithHTTP(info, server.Client()), nil
	})
	a.mgr.SetWincredHandlers(func() (*model.TokenData, error) { return nil, wincred.ErrNotFound }, nil)
	a.requestSessionSync(false, false)
	waitSessionSync(t, a)
	if a.overview.ActiveEmail != "first@example.com" {
		t.Fatal("first live identity not presented")
	}
	live.Store("second@example.com")
	a.mu.Lock()
	a.lastIdentityPoll = time.Now().Add(-identityPollInterval)
	a.mu.Unlock()
	a.pollSessionIdentity()
	waitSessionSync(t, a)
	if a.overview.ActiveEmail != "second@example.com" || len(a.accountCards) == 0 || !a.accountCards[0].IsActive {
		t.Fatal("same-PID identity change did not reach card")
	}
	if a.statusBar.Text != "" {
		t.Fatal("background success spammed footer")
	}
	a.SwitchTab(NavTabCodex)
	live.Store("third@example.com")
	test.Tap(a.sidebar.antigravityItem)
	waitSessionSync(t, a)
	if a.overview.ActiveEmail != "third@example.com" {
		t.Fatal("sidebar return did not resync")
	}
}

func TestSessionSyncManualJoinsBackgroundWithoutParallelRead(t *testing.T) {
	a := newSessionTestApp(t)
	entered, release := make(chan struct{}), make(chan struct{})
	var reads atomic.Int32
	a.mgr.SetWincredHandlers(func() (*model.TokenData, error) {
		if reads.Add(1) == 1 {
			close(entered)
			<-release
		}
		return &model.TokenData{AccessToken: "fixture", IDTokenEmail: "joined@example.com"}, nil
	}, nil)
	a.mgr.SetClientFinder(func(context.Context) (*connector.Client, error) { return nil, connector.ErrLanguageServerNotFound })
	a.requestSessionSync(false, false)
	<-entered
	for i := 0; i < 10; i++ {
		a.Refresh()
		a.requestSessionSync(false, true)
	}
	if reads.Load() != 1 {
		t.Fatal("parallel credential reads")
	}
	close(release)
	waitSessionSync(t, a)
	if a.overview.ActiveEmail != "joined@example.com" || a.statusBar.Text == "" {
		t.Fatal("manual request lost feedback")
	}
	if reads.Load() != 2 {
		t.Fatalf("expected capture + fallback only, got %d", reads.Load())
	}
}

func TestSessionWarningExpiryAndTimerOwnership(t *testing.T) {
	a := newSessionTestApp(t)
	view := &manager.AccountOverview{Warnings: []string{"fixture warning"}, WarningCodes: []string{"credential_identity_missing"}}
	a.showSessionWarnings(view, false)
	if a.leftFooter.Visible() {
		t.Fatal("warning not shown")
	}
	a.autoHideStatusBar(20 * time.Millisecond)
	a.showStatusBar("new operation")
	time.Sleep(45 * time.Millisecond)
	if a.statusBar.Text != "new operation" {
		t.Fatal("old timer cleared newer message")
	}
	a.clearStatusBar()
	a.showSessionWarnings(view, false)
	if a.statusBar.Text != "" {
		t.Fatal("repeated background warning replayed")
	}
	a.showSessionWarnings(&manager.AccountOverview{}, false)
	a.showSessionWarnings(view, false)
	a.autoHideStatusBar(20 * time.Millisecond)
	a.SwitchTab(NavTabCodex)
	time.Sleep(45 * time.Millisecond)
	if a.statusBar.Visible() || !a.codexStatusLabel.Visible() {
		t.Fatal("warning leaked into Codex")
	}
	a.SwitchTab(NavTabAntigravity)
	if !a.leftFooter.Visible() || !a.serverStatusLabel.Visible() {
		t.Fatal("footer failed to recover")
	}
}

func TestSessionSyncRejectsOldRevisionAndClosedCallback(t *testing.T) {
	a := newSessionTestApp(t)
	old := a.overview
	if err := a.mgr.AddDiscoveredAccount("added@example.com"); err != nil {
		t.Fatal(err)
	}
	a.updateUI(old)
	waitSessionSync(t, a)
	if a.overview.Revision == old.Revision {
		t.Fatal("old result replaced current state")
	}
	before := a.overview
	a.Close()
	a.updateUI(&manager.AccountOverview{ActiveEmail: "should-not-render@example.com"})
	a.showStatusBar("closed callback")
	if a.overview != before {
		t.Fatal("closed UI accepted result")
	}
}

func TestSessionSyncCaptureFailureStillShowsLiveIdentity(t *testing.T) {
	a := newSessionTestApp(t)
	a.mgr.SetWincredHandlers(func() (*model.TokenData, error) { return nil, errors.New("fixture credential denied") }, nil)
	a.Refresh()
	waitSessionSync(t, a)
	if a.overview.ActiveEmail == "" || len(a.overview.Warnings) == 0 {
		t.Fatal("capture failure lost live identity or warning")
	}
	if a.statusBarHideTimer == nil {
		t.Fatal("warning has no expiry")
	}
}
