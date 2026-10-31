package manager

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"as-auto-switcher/internal/engines/codex/connector"
	"as-auto-switcher/internal/engines/codex/model"
	"as-auto-switcher/internal/engines/codex/store"
)

// mockOAuthClient simulates Stdio JSON-RPC communication with codex.exe
type mockOAuthClient struct {
	mu                sync.Mutex
	initCalled        bool
	startLoginCalled  bool
	cancelLoginCalled bool
	closeCalled       bool

	startLoginRes *connector.StartLoginResult
	startLoginErr error

	waitRes *connector.LoginCompletedParams
	waitErr error
	waitCh  chan *connector.LoginCompletedParams

	readEmail    string
	readPlanType string
	readErr      error
}

func (m *mockOAuthClient) Initialize(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.initCalled = true
	return nil
}

func (m *mockOAuthClient) StartLogin(ctx context.Context) (*connector.StartLoginResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startLoginCalled = true
	if m.startLoginErr != nil {
		return nil, m.startLoginErr
	}
	if m.startLoginRes != nil {
		return m.startLoginRes, nil
	}
	return &connector.StartLoginResult{
		Type:    "chatgpt",
		LoginID: "test_login_123",
		AuthURL: "https://auth0.openai.com/authorize?test=1",
	}, nil
}

func (m *mockOAuthClient) WaitForLoginCompleted(ctx context.Context, loginID string) (*connector.LoginCompletedParams, error) {
	if m.waitCh != nil {
		select {
		case res := <-m.waitCh:
			return res, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.waitErr != nil {
		return nil, m.waitErr
	}
	if m.waitRes != nil {
		return m.waitRes, nil
	}
	loginIDCopy := loginID
	return &connector.LoginCompletedParams{
		Success: true,
		LoginID: &loginIDCopy,
	}, nil
}

func (m *mockOAuthClient) ReadAccount(ctx context.Context) (string, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.readEmail, m.readPlanType, m.readErr
}

func (m *mockOAuthClient) CancelLogin(ctx context.Context, loginID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cancelLoginCalled = true
	return nil
}

func (m *mockOAuthClient) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closeCalled = true
	return nil
}

func createTestJWT(email string) string {
	claims := map[string]interface{}{
		"email":          email,
		"name":           "Test User",
		"email_verified": true,
		"exp":            time.Now().Add(24 * time.Hour).Unix(),
	}
	payload, _ := json.Marshal(claims)
	return "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9." + base64.RawURLEncoding.EncodeToString(payload) + ".fake_signature"
}

func writeCandidateAuthFile(t *testing.T, path, email, accountID string) {
	t.Helper()
	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0700)

	jwt := createTestJWT(email)
	authObj := model.CodexAuthJson{
		AuthMode:    "chatgpt",
		LastRefresh: time.Now().UTC().Format(time.RFC3339),
		Tokens: model.CodexTokens{
			AccessToken:  "mock_access_token_12345",
			RefreshToken: "mock_refresh_token_67890",
			IDToken:      jwt,
			AccountID:    accountID,
		},
	}
	data, err := json.MarshalIndent(authObj, "", "  ")
	if err != nil {
		t.Fatalf("marshal auth data: %v", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write candidate auth file: %v", err)
	}
}

func setupTestStore(t *testing.T) (store.AccountStore, string) {
	t.Helper()
	tmpDir := t.TempDir()
	storeDir := filepath.Join(tmpDir, "accounts_store", "codex")
	s, err := store.NewAccountStore(storeDir)
	if err != nil {
		t.Fatalf("setup test store failed: %v", err)
	}
	return s, tmpDir
}

func awaitStage(t *testing.T, a *OAuthAttempt, stage string) AttemptSnapshot {
	t.Helper()
	ch, unsubscribe := a.Subscribe()
	defer unsubscribe()
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()
	for {
		select {
		case snapshot := <-ch:
			if snapshot.Stage == stage {
				return snapshot
			}
		case <-timer.C:
			t.Fatalf("waiting for %s, got %+v", stage, a.Snapshot())
			return AttemptSnapshot{}
		}
	}
}
func completeLogin(c *mockOAuthClient) {
	id := "test_login_123"
	c.waitCh <- &connector.LoginCompletedParams{Success: true, LoginID: &id}
}
func testOAuthManager(t *testing.T, st store.AccountStore, c *mockOAuthClient, ttl time.Duration) OAuthManager {
	t.Helper()
	mgr, err := NewOAuthManager(OAuthManagerConfig{Store: st, CodexExePath: "fixture.exe", ClientFactory: func(connector.OAuthClientConfig) (connector.OAuthClient, error) { return c, nil }, AttemptTimeout: time.Second, SaveRetryTTL: ttl})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	return mgr
}
func beginReady(t *testing.T, m OAuthManager, mode, email, id string) *OAuthAttempt {
	t.Helper()
	a, err := m.BeginLogin(context.Background(), mode, email, id)
	if err != nil {
		t.Fatal(err)
	}
	awaitStage(t, a, "waiting_browser")
	return a
}
func seedAccount(t *testing.T, st store.AccountStore, email, id, status string) *model.CodexAccountEnvelope {
	t.Helper()
	path := filepath.Join(t.TempDir(), "auth.json")
	writeCandidateAuthFile(t, path, email, id)
	raw, _ := os.ReadFile(path)
	if err := st.SaveAccount(email, &model.CodexAccountEnvelope{Email: email, Status: status, AuthData: raw, Generation: "seed"}); err != nil {
		t.Fatal(err)
	}
	e, err := st.GetAccount(email)
	if err != nil {
		t.Fatal(err)
	}
	return e
}
func TestOAuthManager_VerifiedIdentityBeforeCommit(t *testing.T) {
	for _, tc := range []struct {
		name, email, id string
		readErr         error
		wantStage       string
	}{
		{"success", "a@example.com", "id-a", nil, "success"},
		{"null-email-metadata-fallback", "", "id-a", nil, "success"},
		{"read-error", "a@example.com", "id-a", errors.New("provider secret"), "account-read"},
		{"wrong-email", "b@example.com", "id-a", nil, "identity"},
		{"missing-account-id", "a@example.com", "", nil, "identity"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st, _ := setupTestStore(t)
			c := &mockOAuthClient{waitCh: make(chan *connector.LoginCompletedParams, 1), readEmail: tc.email, readErr: tc.readErr}
			m := testOAuthManager(t, st, c, time.Second)
			a := beginReady(t, m, "add", "", "")
			writeCandidateAuthFile(t, a.CandidateAuthPath, "a@example.com", tc.id)
			completeLogin(c)
			result := awaitStage(t, a, tc.wantStage)
			if st.HasAccount("a@example.com") != (tc.wantStage == "success") {
				t.Fatal("commit did not match verification outcome")
			}
			if tc.readErr != nil && result.Err.Error() == tc.readErr.Error() {
				t.Fatal("raw provider error escaped")
			}
			_ = m.Close()
			if _, err := os.Stat(a.CandidateHome); !os.IsNotExist(err) {
				t.Fatal("candidate leaked")
			}
		})
	}
}
func TestOAuthManager_PreservesMetadataAndCreatesGeneration(t *testing.T) {
	for _, mode := range []string{"add", "reauth"} {
		t.Run(mode, func(t *testing.T) {
			st, _ := setupTestStore(t)
			old := seedAccount(t, st, "a@example.com", "id-a", "off")
			c := &mockOAuthClient{waitCh: make(chan *connector.LoginCompletedParams, 1), readEmail: old.Email}
			m := testOAuthManager(t, st, c, time.Second)
			email, id := "", ""
			if mode == "reauth" {
				email, id = old.Email, "id-a"
			}
			a := beginReady(t, m, mode, email, id)
			writeCandidateAuthFile(t, a.CandidateAuthPath, old.Email, "id-a")
			completeLogin(c)
			awaitStage(t, a, "success")
			fresh, err := st.GetAccount(old.Email)
			if err != nil {
				t.Fatal(err)
			}
			if fresh.Status != "off" || !fresh.CreatedAt.Equal(old.CreatedAt) || fresh.Generation == old.Generation || fresh.Generation == "" {
				t.Fatal("metadata/generation violated")
			}
			if err := st.MergeActiveCredential(old.Email, old); !errors.Is(err, store.ErrReauthGenerationProtected) {
				t.Fatalf("old active credential overwrote reauth: %v", err)
			}
		})
	}
}
func TestOAuthManager_ReauthRejectsOtherIdentity(t *testing.T) {
	st, _ := setupTestStore(t)
	old := seedAccount(t, st, "a@example.com", "id-a", "off")
	c := &mockOAuthClient{waitCh: make(chan *connector.LoginCompletedParams, 1), readEmail: old.Email}
	m := testOAuthManager(t, st, c, time.Second)
	a := beginReady(t, m, "reauth", old.Email, "id-a")
	writeCandidateAuthFile(t, a.CandidateAuthPath, old.Email, "id-b")
	completeLogin(c)
	awaitStage(t, a, "identity")
	fresh, _ := st.GetAccount(old.Email)
	if fresh.Revision != old.Revision {
		t.Fatal("wrong identity changed record")
	}
}
func TestOAuthManager_CancelShutdownAndSingleFlight(t *testing.T) {
	for _, closeManager := range []bool{false, true} {
		t.Run(map[bool]string{false: "cancel", true: "shutdown"}[closeManager], func(t *testing.T) {
			st, _ := setupTestStore(t)
			c := &mockOAuthClient{waitCh: make(chan *connector.LoginCompletedParams, 1)}
			m := testOAuthManager(t, st, c, time.Second)
			a := beginReady(t, m, "add", "", "")
			same, err := m.BeginLogin(context.Background(), "add", "", "")
			if err != nil || same != a {
				t.Fatal("same attempt not attached")
			}
			if _, err := m.BeginLogin(context.Background(), "reauth", "other@example.com", ""); !errors.Is(err, ErrOAuthBusy) {
				t.Fatal("different attempt admitted")
			}
			if closeManager {
				_ = m.Close()
			} else if err := m.CancelActiveLogin(); err != nil {
				t.Fatal(err)
			}
			awaitStage(t, a, "canceled")
			if st.HasAccount("a@example.com") {
				t.Fatal("canceled account persisted")
			}
			if _, err := os.Stat(a.CandidateHome); !os.IsNotExist(err) {
				t.Fatal("candidate remains after cancel/shutdown")
			}
			c.mu.Lock()
			closed := c.closeCalled
			c.mu.Unlock()
			if !closed {
				t.Fatal("child not closed")
			}
		})
	}
}
func TestOAuthManager_TimeoutCleansChild(t *testing.T) {
	st, _ := setupTestStore(t)
	c := &mockOAuthClient{waitCh: make(chan *connector.LoginCompletedParams, 1)}
	m := testOAuthManager(t, st, c, time.Second)
	a := beginReady(t, m, "add", "", "")
	result := awaitStage(t, a, "login")
	if !errors.Is(result.Err, ErrOAuthTimeout) {
		t.Fatal(result.Err)
	}
	_ = m.Close()
	if _, err := os.Stat(a.CandidateHome); !os.IsNotExist(err) {
		t.Fatal("timeout leaked candidate")
	}
}

type failingSaveStore struct {
	store.AccountStore
	mu               sync.Mutex
	fail             bool
	entered, proceed chan struct{}
}

func (s *failingSaveStore) SaveAccount(email string, e *model.CodexAccountEnvelope) error {
	if s.entered != nil {
		close(s.entered)
		<-s.proceed
	}
	s.mu.Lock()
	fail := s.fail
	s.mu.Unlock()
	if fail {
		return &os.PathError{Op: "write", Path: "fixture", Err: os.ErrPermission}
	}
	return s.AccountStore.SaveAccount(email, e)
}
func TestOAuthManager_RetrySaveAndDeadline(t *testing.T) {
	for _, retry := range []bool{true, false} {
		t.Run(map[bool]string{true: "retry", false: "expiry"}[retry], func(t *testing.T) {
			base, _ := setupTestStore(t)
			st := &failingSaveStore{AccountStore: base, fail: true}
			c := &mockOAuthClient{waitCh: make(chan *connector.LoginCompletedParams, 1), readEmail: "a@example.com"}
			m := testOAuthManager(t, st, c, 150*time.Millisecond)
			a := beginReady(t, m, "add", "", "")
			writeCandidateAuthFile(t, a.CandidateAuthPath, "a@example.com", "id-a")
			completeLogin(c)
			awaitStage(t, a, "save_failed")
			if !m.IsAttemptActive() {
				t.Fatal("SaveFailed released single-flight")
			}
			if retry {
				st.mu.Lock()
				st.fail = false
				st.mu.Unlock()
				if err := m.RetrySave(context.Background(), a.AttemptID); err != nil {
					t.Fatal(err)
				}
				awaitStage(t, a, "success")
				if !base.HasAccount("a@example.com") {
					t.Fatal("retry didn't save")
				}
			} else {
				awaitStage(t, a, "expired")
				if base.HasAccount("a@example.com") {
					t.Fatal("expiry persisted candidate")
				}
			}
			_ = m.Close()
			if _, err := os.Stat(a.CandidateHome); !os.IsNotExist(err) {
				t.Fatal("retry/expiry leaked candidate")
			}
		})
	}
}
func TestOAuthManager_CommitWinsCancelExactlyOnce(t *testing.T) {
	base, _ := setupTestStore(t)
	st := &failingSaveStore{AccountStore: base, entered: make(chan struct{}), proceed: make(chan struct{})}
	c := &mockOAuthClient{waitCh: make(chan *connector.LoginCompletedParams, 1), readEmail: "a@example.com"}
	m := testOAuthManager(t, st, c, time.Second)
	a := beginReady(t, m, "add", "", "")
	writeCandidateAuthFile(t, a.CandidateAuthPath, "a@example.com", "id-a")
	completeLogin(c)
	<-st.entered
	cancelResult := make(chan error, 1)
	go func() { cancelResult <- m.CancelActiveLogin() }()
	close(st.proceed)
	if err := <-cancelResult; !errors.Is(err, ErrAlreadyCommitted) {
		t.Fatalf("cancel after durable commit: %v", err)
	}
	awaitStage(t, a, "success")
	if !base.HasAccount("a@example.com") || a.IsCanceled() {
		t.Fatal("inconsistent commit/cancel")
	}
}
func TestOAuthManager_SubscriptionsReplayAndDoNotCompete(t *testing.T) {
	st, _ := setupTestStore(t)
	c := &mockOAuthClient{waitCh: make(chan *connector.LoginCompletedParams, 1), readEmail: "a@example.com"}
	m := testOAuthManager(t, st, c, time.Second)
	a := beginReady(t, m, "add", "", "")
	one, un1 := a.Subscribe()
	defer un1()
	two, un2 := a.Subscribe()
	defer un2()
	if (<-one).Stage != "waiting_browser" || (<-two).Stage != "waiting_browser" {
		t.Fatal("state replay failed")
	}
	writeCandidateAuthFile(t, a.CandidateAuthPath, "a@example.com", "id-a")
	completeLogin(c)
	awaitStage(t, a, "success")
	late, un3 := a.Subscribe()
	defer un3()
	if (<-late).Stage != "success" {
		t.Fatal("terminal state lost")
	}
}
