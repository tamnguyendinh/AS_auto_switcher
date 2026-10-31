package manager

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"as-auto-switcher/internal/engines/codex/model"
	"as-auto-switcher/internal/engines/codex/reloader"
	"as-auto-switcher/internal/engines/codex/store"
)

type mockSessionReloader struct {
	switchedTarget string
	activeEmail    string
	switchErr      error
}

func (m *mockSessionReloader) SwitchSession(targetEmail string) error {
	m.switchedTarget = targetEmail
	return m.switchErr
}

func (m *mockSessionReloader) GetActiveSessionEmail() (string, error) {
	return m.activeEmail, nil
}

func createTestAuthData(t *testing.T, email, accountID string, validTokens bool) []byte {
	t.Helper()
	var tokens model.CodexTokens
	if validTokens {
		tokens = model.CodexTokens{
			AccessToken:  "mock_access_token",
			RefreshToken: "mock_refresh_token",
			IDToken:      createTestJWT(email),
			AccountID:    accountID,
		}
	} else {
		// Empty tokens representing unauthenticated / revoked state
		tokens = model.CodexTokens{
			AccessToken:  "",
			RefreshToken: "",
		}
	}

	authObj := model.CodexAuthJson{
		AuthMode:    "chatgpt",
		LastRefresh: time.Now().UTC().Format(time.RFC3339),
		Tokens:      tokens,
	}
	data, err := json.Marshal(authObj)
	if err != nil {
		t.Fatalf("marshal auth data failed: %v", err)
	}
	return data
}

func TestAccountManager_UnknownActiveBlocksDeleteAndDoesNotClaimActive(t *testing.T) {
	st, _ := setupTestStore(t)
	seedAccount(t, st, "a@example.com", "id-a", "off")
	m, err := NewAccountManager(AccountManagerConfig{Store: st})
	if err != nil {
		t.Fatal(err)
	}
	cards, err := m.ListCards()
	if err != nil || len(cards) != 1 {
		t.Fatal(err)
	}
	if cards[0].State != model.StateUnknown || cards[0].CanDelete {
		t.Fatal("unknown identity treated as safe/active")
	}
	if err := m.DeleteAccount("a@example.com"); !errors.Is(err, reloader.ErrUnsupportedDesktopVerification) {
		t.Fatal(err)
	}
	if !st.HasAccount("a@example.com") {
		t.Fatal("unknown-active account removed")
	}
	if err := m.BootstrapActiveSession(); !errors.Is(err, reloader.ErrUnsupportedDesktopVerification) {
		t.Fatal(err)
	}
	if err := m.SwitchAccount("a@example.com"); !errors.Is(err, reloader.ErrUnsupportedDesktopVerification) {
		t.Fatal(err)
	}
}
func TestAccountManager_VerifiedActiveAndManualOffSwitch(t *testing.T) {
	st, _ := setupTestStore(t)
	seedAccount(t, st, "a@example.com", "id-a", "on")
	seedAccount(t, st, "b@example.com", "id-b", "off")
	rel := &mockSessionReloader{activeEmail: "a@example.com"}
	m, _ := NewAccountManager(AccountManagerConfig{Store: st, Reloader: rel})
	cards, err := m.ListCards()
	if err != nil {
		t.Fatal(err)
	}
	if cards[0].State != model.StateActive || cards[0].CanDelete || !cards[1].CanDelete {
		t.Fatal("active protection wrong")
	}
	if err := m.DeleteAccount("a@example.com"); !errors.Is(err, ErrCannotDeleteActiveAccount) {
		t.Fatal(err)
	}
	if err := m.SwitchAccount("b@example.com"); err != nil || rel.switchedTarget != "b@example.com" {
		t.Fatal("manual OFF switch rejected")
	}
	status, _ := st.GetAccountStatus("b@example.com")
	if status != "off" {
		t.Fatal("manual switch reset preference")
	}
	if err := m.DeleteAccount("b@example.com"); err != nil {
		t.Fatal(err)
	}
}
func TestAccountManager_SharedMutationGate(t *testing.T) {
	st, _ := setupTestStore(t)
	seedAccount(t, st, "a@example.com", "id-a", "off")
	gate := store.Coordinator(st)
	gate.AttemptID = "oauth-active"
	m, _ := NewAccountManager(AccountManagerConfig{Store: st, ActiveResolver: func() (string, error) { return "", nil }})
	if err := m.DeleteAccount("a@example.com"); !errors.Is(err, model.ErrMutationBusy) {
		t.Fatal(err)
	}
	if err := m.ToggleStatus("a@example.com"); !errors.Is(err, model.ErrMutationBusy) {
		t.Fatal(err)
	}
	gate.AttemptID = ""
	gate.RecoveryRequired = true
	if err := m.DeleteAccount("a@example.com"); !errors.Is(err, model.ErrRecoveryRequired) {
		t.Fatal(err)
	}
}
func TestAccountManager_ToggleAndInputValidation(t *testing.T) {
	st, _ := setupTestStore(t)
	seedAccount(t, st, "a@example.com", "id-a", "off")
	m, _ := NewAccountManager(AccountManagerConfig{Store: st})
	if err := m.ToggleStatus("a@example.com"); err != nil {
		t.Fatal(err)
	}
	status, _ := st.GetAccountStatus("a@example.com")
	if status != "on" {
		t.Fatal("toggle not persisted")
	}
	for _, fn := range []func(string) error{m.DeleteAccount, m.ToggleStatus, m.SwitchAccount} {
		if !errors.Is(fn(""), ErrEmptyEmail) {
			t.Fatal("empty identity accepted")
		}
	}
}
