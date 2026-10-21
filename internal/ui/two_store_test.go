package ui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"antigravity-switch/internal/connector"
	"antigravity-switch/internal/manager"
	"antigravity-switch/internal/model"
	"antigravity-switch/internal/store"
	"antigravity-switch/internal/wincred"
	"fyne.io/fyne/v2/test"
)

func TestThreeStateCardUpdatesInPlace(t *testing.T) {
	test.NewApp()
	item := manager.AccountDisplayItem{Email: "target@example.com"}
	calls := 0
	w := NewAccountCardWidgetWithQuota(item, nil, DarkPalette, func(string) { calls++ }, nil)
	root, cell := w.Root, w.gemini5hCell
	w.SetMasked(true)
	if findSwitchButton(root) != nil || w.gemini5hPct.Text != "N/A" || !w.refreshButton.Disabled() {
		t.Fatal("unauth card offered usable actions/data")
	}
	item.HasToken = true
	w.UpdateState(item, DarkPalette)
	w.UpdateQuota(nil, DarkPalette)
	button := findSwitchButton(root)
	if button == nil {
		t.Fatal("token addition did not create switch action")
	}
	button.OnTapped()
	if calls != 1 {
		t.Fatal("switch callback not wired")
	}
	if w.gemini5hPct.Text != "--%" {
		t.Fatal("ready missing quota must stay unknown")
	}
	item.HasToken = false
	w.UpdateState(item, DarkPalette)
	w.UpdateQuota(nil, DarkPalette)
	button.OnTapped()
	if calls != 1 || findSwitchButton(root) != nil || w.gemini5hPct.Text != "N/A" {
		t.Fatal("removed credential still actionable")
	}
	item.IsActive = true
	w.UpdateState(item, DarkPalette)
	w.UpdateQuota(nil, DarkPalette)
	if findSwitchButton(root) != nil || w.refreshButton.Disabled() || w.gemini5hPct.Text == "N/A" {
		t.Fatal("active state lost priority")
	}
	if w.Root != root || w.gemini5hCell != cell || w.EmailLabelText() != MaskEmail(item.Email) {
		t.Fatal("state change rebuilt card or lost masking")
	}
}

func TestUpdateUIObservesCredentialChangesWithSameEmail(t *testing.T) {
	f := test.NewApp()
	a := NewAppWithFyne(f, nil)
	a.preferencesPath = filepath.Join(t.TempDir(), "prefs.json")
	a.Init()
	defer a.Close()
	item := manager.AccountDisplayItem{Email: "same@example.com"}
	view := &manager.AccountOverview{Accounts: []manager.AccountDisplayItem{item}}
	a.updateUI(view)
	original := a.accountCards[0]
	view.Accounts[0].HasToken = true
	a.updateUI(view)
	if a.accountCards[0] != original || findSwitchButton(original.Root) == nil {
		t.Fatal("existing card did not update credential state")
	}
	view.Accounts[0].HasToken = false
	a.updateUI(view)
	if a.accountCards[0] != original || findSwitchButton(original.Root) != nil {
		t.Fatal("credential removal did not remove switch")
	}
}

func TestRefreshRepeatedCaptureAndFailureFeedback(t *testing.T) {
	root := t.TempDir()
	vault, _ := store.NewAccountStore(filepath.Join(root, "accounts_store"))
	m, err := manager.NewAccountManager(vault, nil)
	if err != nil {
		t.Fatal(err)
	}
	m.SetChromeDir(filepath.Join(root, "no-chrome"))
	m.SetClientFinder(func(context.Context) (*connector.Client, error) { return nil, errors.New("fixture offline") })
	m.SetWincredHandlers(func() (*model.TokenData, error) { return nil, wincred.ErrNotFound }, nil)
	f := test.NewApp()
	// This integration exercises production Refresh with the real file store,
	// without a polling loop or a live Language Server.
	a := &App{app: f, mgr: m, currentTheme: ThemeDark, preferencesPath: filepath.Join(root, "prefs.json")}
	a.Init()
	defer a.Close()
	waitForIdle(a, 3*time.Second)
	entered, release := make(chan struct{}), make(chan struct{})
	var reads atomic.Int32
	m.SetWincredHandlers(func() (*model.TokenData, error) {
		if reads.Add(1) == 1 {
			close(entered)
			<-release
		}
		return &model.TokenData{AccessToken: "fixture", IDToken: makeTestJWT("captured@example.com")}, nil
	}, nil)
	a.Refresh()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("capture did not start")
	}
	for i := 0; i < 10; i++ {
		a.Refresh()
	}
	if reads.Load() != 1 {
		t.Fatalf("overlapping refresh: %d", reads.Load())
	}
	close(release)
	waitForIdle(a, 3*time.Second)
	if !vault.HasAccount("captured@example.com") || len(a.overview.Accounts) != 1 {
		t.Fatal("refresh did not persist/render captured identity")
	}
	for i := 0; i < 3; i++ {
		a.Refresh()
		waitForIdle(a, 3*time.Second)
	}
	emails, _ := vault.ListAccounts()
	if len(emails) != 1 || len(a.accountCards) != 1 {
		t.Fatal("repeated capture duplicated identity")
	}
	m.SetWincredHandlers(func() (*model.TokenData, error) { return nil, errors.New("fixture read denied") }, nil)
	a.Refresh()
	waitForIdle(a, 3*time.Second)
	if !strings.Contains(a.statusBar.Text, "fixture read denied") || len(a.accountCards) != 1 {
		t.Fatal("capture warning missing or saved card lost")
	}
	if _, err := os.Stat(filepath.Join(root, "discovered_accounts")); err != nil {
		t.Fatal(err)
	}
}
