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
	// Under the decoupled architecture, Chrome scanning is on-demand:
	if _, err := m.ScanChromeAccounts(); err != nil {
		t.Fatalf("ScanChromeAccounts failed: %v", err)
	}
	if err := m.AddDiscoveredAccount("NEW@example.com"); err != nil {
		t.Fatalf("AddDiscoveredAccount failed: %v", err)
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
	expectedHistory := filepath.Join(filepath.Dir(m.Store().GetStoreDir()), "logs", "active_threads_history")
	if m.reloader.OutputDir != expectedHistory {
		t.Fatalf("history escaped selected logs directory: got %s, want %s", m.reloader.OutputDir, expectedHistory)
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
	// Corrupted catalog entry simulates partial I/O failure in discoveredStore.
	if err := os.WriteFile(filepath.Join(catalog, "corrupt.json"), []byte("corrupt-data"), 0600); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(root, "chrome", "Default", "Preferences"), []byte(`{"account_info":[{"email":"live@example.com"}]}`), 0600)
	m.SetWincredHandlers(func() (*model.TokenData, error) { return nil, errors.New("fixture credential read denied") }, nil)
	view, err = m.GetAccountStates(context.Background())
	if err != nil || len(view.Accounts) != 1 || len(view.Warnings) < 2 {
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

func TestGetAccountStates_DoesNotScanChrome(t *testing.T) {
	m, root := catalogManager(t)
	pref := filepath.Join(root, "chrome", "Default", "Preferences")
	os.WriteFile(pref, []byte(`{"account_info":[{"email":"unscanned@example.com"}]}`), 0600)
	if err := m.Store().SaveAccount("stored@example.com", &model.TokenData{RefreshToken: "fixture"}); err != nil {
		t.Fatal(err)
	}

	view, err := m.GetAccountStates(context.Background())
	if err != nil {
		t.Fatalf("GetAccountStates failed: %v", err)
	}

	// Only stored@example.com should be present; unscanned@example.com must NOT appear
	if len(view.Accounts) != 1 {
		t.Fatalf("expected exactly 1 account, got %d: %+v", len(view.Accounts), view.Accounts)
	}
	if view.Accounts[0].Email != "stored@example.com" {
		t.Fatalf("expected stored@example.com, got %s", view.Accounts[0].Email)
	}

	// Verify no files were created in discovered_accounts/
	discoveredDir := filepath.Join(root, "discovered_accounts")
	entries, readErr := os.ReadDir(discoveredDir)
	if readErr == nil && len(entries) > 0 {
		t.Fatalf("GetAccountStates generated discovered records when it must not scan: found %d files", len(entries))
	}
}

func TestScanChromeAccounts_OnDemand(t *testing.T) {
	m, root := catalogManager(t)
	pref := filepath.Join(root, "chrome", "Default", "Preferences")
	os.WriteFile(pref, []byte(`{"account_info":[{"email":"scanned1@example.com"},{"email":"scanned2@example.com"}]}`), 0600)

	scanned, err := m.ScanChromeAccounts()
	if err != nil {
		t.Fatalf("ScanChromeAccounts failed: %v", err)
	}
	if len(scanned) != 2 {
		t.Fatalf("expected 2 scanned emails, got %d: %v", len(scanned), scanned)
	}

	// Verify accounts saved in discoveredStore with Added == false
	candidates, err := m.GetCandidateAccounts()
	if err != nil {
		t.Fatalf("GetCandidateAccounts failed: %v", err)
	}
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidate accounts, got %d", len(candidates))
	}
	for _, cand := range candidates {
		if cand.Added {
			t.Fatalf("candidate %s should have Added == false by default", cand.Email)
		}
		if cand.HasToken {
			t.Fatalf("candidate %s should have HasToken == false", cand.Email)
		}
	}

	// UI dashboard GetAccountStates should still show 0 accounts because Added == false
	view, err := m.GetAccountStates(context.Background())
	if err != nil {
		t.Fatalf("GetAccountStates failed: %v", err)
	}
	if len(view.Accounts) != 0 {
		t.Fatalf("expected 0 accounts on main dashboard before add, got %d", len(view.Accounts))
	}
}

func TestAddDiscoveredAccount(t *testing.T) {
	m, root := catalogManager(t)
	pref := filepath.Join(root, "chrome", "Default", "Preferences")
	os.WriteFile(pref, []byte(`{"account_info":[{"email":"candidate@example.com"}]}`), 0600)

	if _, err := m.ScanChromeAccounts(); err != nil {
		t.Fatalf("ScanChromeAccounts failed: %v", err)
	}

	// Mark as added
	if err := m.AddDiscoveredAccount("candidate@example.com"); err != nil {
		t.Fatalf("AddDiscoveredAccount failed: %v", err)
	}

	// Verify candidate now has Added == true
	candidates, err := m.GetCandidateAccounts()
	if err != nil {
		t.Fatalf("GetCandidateAccounts failed: %v", err)
	}
	if len(candidates) != 1 || !candidates[0].Added {
		t.Fatalf("expected candidate with Added == true, got: %+v", candidates)
	}

	// Now GetAccountStates must include candidate@example.com as [Cần OAuth]
	view, err := m.GetAccountStates(context.Background())
	if err != nil {
		t.Fatalf("GetAccountStates failed: %v", err)
	}
	if len(view.Accounts) != 1 {
		t.Fatalf("expected 1 account in overview, got %d", len(view.Accounts))
	}
	if view.Accounts[0].Email != "candidate@example.com" || view.Accounts[0].DisplayBadge != "[Cần OAuth]" || view.Accounts[0].State != model.StateUnauth {
		t.Fatalf("expected [Cần OAuth] card for candidate, got: %+v", view.Accounts[0])
	}

	// Re-scan Chrome to ensure Added flag is preserved across scans
	if _, err := m.ScanChromeAccounts(); err != nil {
		t.Fatalf("re-scan failed: %v", err)
	}
	candidates, err = m.GetCandidateAccounts()
	if err != nil || len(candidates) != 1 || !candidates[0].Added {
		t.Fatalf("re-scan wiped Added flag: %+v %v", candidates, err)
	}
}

func TestGetCandidateAccounts_SortingAndEnrichment(t *testing.T) {
	m, root := catalogManager(t)
	pref := filepath.Join(root, "chrome", "Default", "Preferences")
	os.WriteFile(pref, []byte(`{"account_info":[
		{"email":"zebra_no_token@example.com"},
		{"email":"alpha_no_token@example.com"},
		{"email":"token_user@example.com"},
		{"email":"added_user@example.com"}
	]}`), 0600)

	// token_user has token in account store
	if err := m.Store().SaveAccount("token_user@example.com", &model.TokenData{RefreshToken: "fixture"}); err != nil {
		t.Fatal(err)
	}

	if _, err := m.ScanChromeAccounts(); err != nil {
		t.Fatalf("ScanChromeAccounts failed: %v", err)
	}

	// added_user is marked added
	if err := m.AddDiscoveredAccount("added_user@example.com"); err != nil {
		t.Fatalf("AddDiscoveredAccount failed: %v", err)
	}

	candidates, err := m.GetCandidateAccounts()
	if err != nil {
		t.Fatalf("GetCandidateAccounts failed: %v", err)
	}
	if len(candidates) != 4 {
		t.Fatalf("expected 4 candidates, got %d", len(candidates))
	}

	// Expected order:
	// 1. HasToken == true: token_user@example.com
	// 2. Added == true: added_user@example.com
	// 3. alpha_no_token@example.com (A-Z)
	// 4. zebra_no_token@example.com (A-Z)
	if candidates[0].Email != "token_user@example.com" || !candidates[0].HasToken {
		t.Errorf("candidate[0] mismatch: %+v", candidates[0])
	}
	if candidates[1].Email != "added_user@example.com" || !candidates[1].Added || candidates[1].HasToken {
		t.Errorf("candidate[1] mismatch: %+v", candidates[1])
	}
	if candidates[2].Email != "alpha_no_token@example.com" || candidates[2].Added || candidates[2].HasToken {
		t.Errorf("candidate[2] mismatch: %+v", candidates[2])
	}
	if candidates[3].Email != "zebra_no_token@example.com" || candidates[3].Added || candidates[3].HasToken {
		t.Errorf("candidate[3] mismatch: %+v", candidates[3])
	}
}

func TestCandidateAccounts_UninitializedStore(t *testing.T) {
	mgr := &AccountManager{}
	if _, err := mgr.ScanChromeAccounts(); err == nil {
		t.Error("expected error for uninitialized discovered store in ScanChromeAccounts")
	}
	if _, err := mgr.GetCandidateAccounts(); err == nil {
		t.Error("expected error for uninitialized discovered store in GetCandidateAccounts")
	}
	if err := mgr.AddDiscoveredAccount("test@example.com"); err == nil {
		t.Error("expected error for uninitialized discovered store in AddDiscoveredAccount")
	}
	if err := mgr.DeleteDiscoveredAccount("test@example.com"); err == nil {
		t.Error("expected error for uninitialized discovered store in DeleteDiscoveredAccount")
	}
}

func TestDeleteDiscoveredAccount_Isolation(t *testing.T) {
	m, root := catalogManager(t)
	email := "isolated@example.com"

	// 1. Lưu token vào accounts_store/
	tok := &model.TokenData{
		AccessToken:  "token_iso",
		RefreshToken: "ref_iso",
		IDToken:      makeTestJWT(email),
	}
	if err := m.Store().SaveAccount(email, tok); err != nil {
		t.Fatalf("SaveAccount failed: %v", err)
	}

	// 2. Lưu vào discoveredStore
	if err := m.DiscoveredStore().SaveDiscovered(email); err != nil {
		t.Fatalf("SaveDiscovered failed: %v", err)
	}
	discPath := filepath.Join(root, "discovered_accounts", email+".json")
	accPath := filepath.Join(m.Store().GetStoreDir(), email+".json")

	if _, err := os.Stat(discPath); err != nil {
		t.Fatalf("discovered file should exist before delete: %v", err)
	}
	if _, err := os.Stat(accPath); err != nil {
		t.Fatalf("accounts_store file should exist before delete: %v", err)
	}

	// 3. Thực hiện DeleteDiscoveredAccount
	if err := m.DeleteDiscoveredAccount(email); err != nil {
		t.Fatalf("DeleteDiscoveredAccount failed: %v", err)
	}

	// 4. Invariant-Store-Isolation: File trong discovered_accounts bị xóa
	if _, err := os.Stat(discPath); !os.IsNotExist(err) {
		t.Fatalf("discovered file must be deleted, got err: %v", err)
	}
	if m.DiscoveredStore().HasDiscovered(email) {
		t.Fatalf("HasDiscovered must return false after delete")
	}

	// 5. Invariant-Store-Isolation: accounts_store TUYỆT ĐỐI không bị ảnh hưởng!
	if _, err := os.Stat(accPath); err != nil {
		t.Fatalf("accounts_store file MUST be preserved, but got: %v", err)
	}
	if !m.Store().HasAccount(email) {
		t.Fatalf("m.Store().HasAccount must return true, credentials must not be deleted!")
	}

	// UI Overview vẫn hiển thị tài khoản từ accounts_store
	view, err := m.GetAccountStates(context.Background())
	if err != nil {
		t.Fatalf("GetAccountStates failed: %v", err)
	}
	var foundInOverview bool
	for _, acc := range view.Accounts {
		if acc.Email == email {
			foundInOverview = true
			if !acc.HasToken {
				t.Errorf("expected HasToken=true on main UI card")
			}
			break
		}
	}
	if !foundInOverview {
		t.Fatalf("account %s must still be present on main UI overview", email)
	}
}

func TestGetCandidateAccounts_NonExistentDetection(t *testing.T) {
	m, _ := catalogManager(t)

	tokGhost := "tok_ghost@example.com"
	unauthGhost := "unauth_ghost@example.com"
	normalUser := "normal@example.com"

	// 1. tokGhost có trong accounts_store nhưng KHÔNG có file trong discoveredStore
	tok := &model.TokenData{
		AccessToken:  "token_ghost",
		RefreshToken: "ref_ghost",
		IDToken:      makeTestJWT(tokGhost),
	}
	if err := m.Store().SaveAccount(tokGhost, tok); err != nil {
		t.Fatal(err)
	}
	m.TrackAccount(tokGhost)

	// 2. unauthGhost được track trên UI (ví dụ unauth card) nhưng KHÔNG có file trong discoveredStore
	m.TrackAccount(unauthGhost)

	// 3. normalUser có file hợp lệ trong discoveredStore
	if err := m.DiscoveredStore().SaveDiscovered(normalUser); err != nil {
		t.Fatal(err)
	}

	// Lấy danh sách candidates
	candidates, err := m.GetCandidateAccounts()
	if err != nil {
		t.Fatalf("GetCandidateAccounts failed: %v", err)
	}
	if len(candidates) != 3 {
		t.Fatalf("expected 3 candidates, got %d: %+v", len(candidates), candidates)
	}

	candMap := make(map[string]CandidateAccountItem)
	for _, c := range candidates {
		candMap[c.Email] = c
	}

	// Kiểm tra tokGhost: FileExists == false, HasToken == true, Added == true
	cTok, ok := candMap[tokGhost]
	if !ok {
		t.Fatalf("candidate %s not found", tokGhost)
	}
	if cTok.FileExists {
		t.Errorf("expected tokGhost FileExists == false, got true")
	}
	if !cTok.HasToken {
		t.Errorf("expected tokGhost HasToken == true")
	}
	if !cTok.Added {
		t.Errorf("expected tokGhost Added == true")
	}

	// Kiểm tra unauthGhost: FileExists == false, HasToken == false, Added == true
	cUnauth, ok := candMap[unauthGhost]
	if !ok {
		t.Fatalf("candidate %s not found", unauthGhost)
	}
	if cUnauth.FileExists {
		t.Errorf("expected unauthGhost FileExists == false, got true")
	}
	if cUnauth.HasToken {
		t.Errorf("expected unauthGhost HasToken == false")
	}
	if !cUnauth.Added {
		t.Errorf("expected unauthGhost Added == true")
	}

	// Kiểm tra normalUser: FileExists == true
	cNormal, ok := candMap[normalUser]
	if !ok {
		t.Fatalf("candidate %s not found", normalUser)
	}
	if !cNormal.FileExists {
		t.Errorf("expected normalUser FileExists == true, got false")
	}

	// Sau khi gọi DeleteDiscoveredAccount trên unauthGhost (gỡ khỏi tracking)
	if err := m.DeleteDiscoveredAccount(unauthGhost); err != nil {
		t.Fatal(err)
	}
	candidatesAfter, err := m.GetCandidateAccounts()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range candidatesAfter {
		if c.Email == unauthGhost {
			t.Errorf("unauthGhost %s should be removed after DeleteDiscoveredAccount", unauthGhost)
		}
	}
}

