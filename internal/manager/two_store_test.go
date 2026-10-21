package manager

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"antigravity-switch/internal/connector"
	"antigravity-switch/internal/model"
	"antigravity-switch/internal/store"
	"antigravity-switch/internal/wincred"
)

func catalogManager(t *testing.T) (*AccountManager, string) {
	t.Helper()
	root := t.TempDir()
	s, err := store.NewAccountStore(filepath.Join(root, "accounts_store"))
	if err != nil {
		t.Fatal(err)
	}
	m, err := NewAccountManager(s, nil)
	if err != nil {
		t.Fatal(err)
	}
	m.SetClientFinder(func(context.Context) (*connector.Client, error) { return nil, errors.New("fixture offline") })
	m.SetWincredHandlers(func() (*model.TokenData, error) { return nil, wincred.ErrNotFound }, nil)
	chrome := filepath.Join(root, "chrome", "Default")
	if err := os.MkdirAll(chrome, 0755); err != nil {
		t.Fatal(err)
	}
	m.SetChromeDir(filepath.Dir(chrome))
	return m, root
}

func TestTwoStoreCaptureMergeRepeatAndRemember(t *testing.T) {
	m, root := catalogManager(t)
	pref := filepath.Join(root, "chrome", "Default", "Preferences")
	os.WriteFile(pref, []byte(`{"account_info":[{"email":"NEW@example.com"},{"email":"active@example.com"}]}`), 0600)
	token := &model.TokenData{AccessToken: "fixture", IDToken: makeTestJWT("active@example.com")}
	m.SetWincredHandlers(func() (*model.TokenData, error) { return token, nil }, nil)
	if err := m.Store().SaveAccount("stored@example.com", &model.TokenData{RefreshToken: "fixture"}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		view, err := m.GetAccountStates(context.Background())
		if err != nil || len(view.Warnings) > 0 {
			t.Fatalf("%v %v", err, view.Warnings)
		}
		if len(view.Accounts) != 3 {
			t.Fatalf("duplicates or missing: %+v", view.Accounts)
		}
		if !view.Accounts[0].IsActive || !view.Accounts[0].HasToken {
			t.Fatalf("capture/active: %+v", view.Accounts[0])
		}
		if view.Accounts[2].Email != "new@example.com" || view.Accounts[2].HasToken || view.Accounts[2].DisplayBadge != "[Cần OAuth]" {
			t.Fatalf("unauth: %+v", view.Accounts[2])
		}
	}
	stored, _ := m.Store().ListAccounts()
	if len(stored) != 2 {
		t.Fatalf("catalog polluted vault: %v", stored)
	}
	os.Remove(pref)
	view, err := m.GetAccountStates(context.Background())
	if err != nil || len(view.Accounts) != 3 {
		t.Fatalf("remembered identity lost: %+v %v", view, err)
	}
	if m.reloader.OutputDir != filepath.Join(m.Store().GetStoreDir(), "active_threads_history") {
		t.Fatal("history escaped selected store")
	}
}

func TestTwoStorePartialFailuresAndMissingSession(t *testing.T) {
	m, root := catalogManager(t)
	if err := m.Store().SaveAccount("saved@example.com", &model.TokenData{RefreshToken: "fixture"}); err != nil {
		t.Fatal(err)
	}
	view, err := m.GetAccountStates(context.Background())
	if err != nil || len(view.Warnings) != 0 || len(view.Accounts) != 1 {
		t.Fatalf("missing session must be normal: %+v %v", view, err)
	}
	catalog := filepath.Join(root, "discovered_accounts")
	// Only this test-owned, empty catalog is replaced by a file to simulate I/O failure.
	if err := os.Remove(catalog); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(catalog, []byte("blocked"), 0600)
	os.WriteFile(filepath.Join(root, "chrome", "Default", "Preferences"), []byte(`{"account_info":[{"email":"live@example.com"}]}`), 0600)
	m.SetWincredHandlers(func() (*model.TokenData, error) { return nil, errors.New("fixture credential read denied") }, nil)
	view, err = m.GetAccountStates(context.Background())
	if err != nil || len(view.Accounts) != 2 || len(view.Warnings) < 2 {
		t.Fatalf("partial failure hidden or data dropped: %+v %v", view, err)
	}
	if !m.Store().HasAccount("saved@example.com") {
		t.Fatal("existing credential lost")
	}
}

func TestTwoStoreCaptureWriteFailureKeepsLiveIdentity(t *testing.T) {
	m, _ := catalogManager(t)
	// A directory at the account filename makes replacement fail without permissions tricks.
	blocked := filepath.Join(m.Store().GetStoreDir(), "active@example.com.json")
	os.Mkdir(blocked, 0755)
	os.WriteFile(filepath.Join(blocked, "keep"), []byte("keep"), 0600)
	m.SetWincredHandlers(func() (*model.TokenData, error) {
		return &model.TokenData{AccessToken: "fixture", IDToken: makeTestJWT("active@example.com")}, nil
	}, nil)
	view, err := m.GetAccountStates(context.Background())
	if err != nil || len(view.Warnings) == 0 || len(view.Accounts) != 1 || !view.Accounts[0].IsActive || view.Accounts[0].HasToken {
		t.Fatalf("capture failure: %+v %v", view, err)
	}
}
