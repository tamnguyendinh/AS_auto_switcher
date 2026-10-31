//go:build windows && gui && nativeui

package ui

import (
	"context"
	"encoding/json"
	"image/png"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"as-auto-switcher/internal/engines/antigravity/connector"
	"as-auto-switcher/internal/engines/antigravity/manager"
	"as-auto-switcher/internal/engines/antigravity/model"
	"as-auto-switcher/internal/engines/antigravity/store"
	"as-auto-switcher/internal/engines/antigravity/wincred"
	"fyne.io/fyne/v2"
)

// Native Fyne/GLFW UI with isolated credentials and a loopback IDE contract.
// Never calls real account switching or modifies the user's account stores.
func TestNativeSessionSync(t *testing.T) {
	a := nativeWindowApp
	vault, err := store.NewAccountStore(filepath.Join(t.TempDir(), "accounts_store"))
	if err != nil {
		t.Fatal(err)
	}
	mgr, err := manager.NewAccountManager(vault, nil)
	if err != nil {
		t.Fatal(err)
	}
	var online atomic.Bool
	var live atomic.Value
	live.Store("first@example.com")
	var credential atomic.Pointer[model.TokenData]
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !online.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"userStatus": map[string]string{"email": live.Load().(string)}})
	}))
	defer server.Close()
	u, _ := url.Parse(server.URL)
	port, _ := strconv.Atoi(u.Port())
	info := &connector.ServerInfo{PID: 12345, Port: port, CSRFToken: "native-fixture"}
	mgr.SetClientFinder(func(context.Context) (*connector.Client, error) {
		if !online.Load() {
			return nil, connector.ErrLanguageServerNotFound
		}
		return connector.NewClientWithHTTP(info, server.Client()), nil
	})
	mgr.SetWincredHandlers(func() (*model.TokenData, error) {
		token := credential.Load()
		if token == nil {
			return nil, wincred.ErrNotFound
		}
		return token, nil
	}, nil)
	fyne.DoAndWait(func() {
		a.mgr = mgr
		a.SetLivenessChecker(func(*connector.ServerInfo) bool { return online.Load() })
		a.SetServerDiscoverer(func() (*connector.ServerInfo, error) {
			if online.Load() {
				return info, nil
			}
			return nil, connector.ErrLanguageServerNotFound
		})
		a.Refresh()
	})
	waitNativeResize(t, func() bool { return !a.sessionSyncRunning && a.overview != nil })
	fyne.DoAndWait(func() { a.clearStatusBar() })
	online.Store(true)
	credential.Store(&model.TokenData{AccessToken: "fixture", IDTokenEmail: "first@example.com"})
	// The running 2s watcher, rather than a direct refresh, must populate the card.
	deadline := time.Now().Add(10 * time.Second)
	wait := func(want string) {
		t.Helper()
		for time.Now().Before(deadline) {
			ready := false
			fyne.DoAndWait(func() {
				ready = a.overview != nil && a.overview.ActiveEmail == want && !a.sessionSyncRunning && len(a.accountCards) > 0 && a.accountCards[0].IsActive
			})
			if ready {
				return
			}
			time.Sleep(30 * time.Millisecond)
		}
		t.Fatalf("native active card failed to reach %s", want)
	}
	wait("first@example.com")
	firstCapture := time.Now()
	live.Store("second@example.com")
	credential.Store(&model.TokenData{AccessToken: "fixture-2", IDTokenEmail: "second@example.com"})
	deadline = time.Now().Add(15 * time.Second)
	wait("second@example.com")
	detectionMs := time.Since(firstCapture).Milliseconds()
	var screenshotErr error
	capture := func(name string) {
		// Allow a native render frame after the state assertion before reading pixels.
		time.Sleep(200 * time.Millisecond)
		fyne.DoAndWait(func() {
			output := filepath.Join("..", "..", "reports", "QA", "session-sync", name)
			if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
				screenshotErr = err
				return
			}
			f, err := os.Create(output)
			if err != nil {
				screenshotErr = err
				return
			}
			defer f.Close()
			screenshotErr = png.Encode(f, a.win.Canvas().Capture())
		})
		if screenshotErr != nil {
			t.Fatal(screenshotErr)
		}
	}
	capture("native-active.png")
	fyne.DoAndWait(func() {
		a.showSessionWarnings(&manager.AccountOverview{Warnings: []string{"fixture"}, WarningCodes: []string{"credential_read"}}, true)
	})
	capture("native-warning.png")
	time.Sleep(3200 * time.Millisecond)
	fyne.DoAndWait(func() {
		if a.statusBar.Visible() || !a.serverStatusLabel.Visible() || !a.leftFooter.Visible() {
			t.Error("warning did not restore footer")
		}
	})
	capture("native-footer-restored.png")
	clickNativeControl(t, a.sidebar.codexItem)
	waitNativeResize(t, func() bool { return a.sidebar.ActiveTab() == NavTabCodex })
	capture("native-codex.png")
	live.Store("third@example.com")
	credential.Store(&model.TokenData{AccessToken: "fixture-3", IDTokenEmail: "third@example.com"})
	clickNativeControl(t, a.sidebar.antigravityItem)
	deadline = time.Now().Add(10 * time.Second)
	wait("third@example.com")
	capture("native-return.png")
	result := map[string]any{"native_fyne": true, "isolated_credentials": true, "watcher_startup": true, "same_pid_identity_change": true, "same_pid_detection_ms": detectionMs, "warning_expired": true, "contextual_footer": true, "tab_return_sync": true, "real_account_switch_performed": false}
	data, _ := json.MarshalIndent(result, "", "  ")
	if err := os.WriteFile(filepath.Join("..", "..", "reports", "QA", "session-sync", "native-result.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	fyne.DoAndWait(func() { a.Close() })
}
