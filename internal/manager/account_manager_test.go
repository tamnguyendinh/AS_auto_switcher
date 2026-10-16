package manager

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"antigravity-switch/internal/connector"
	"antigravity-switch/internal/model"
	"antigravity-switch/internal/reloader"
	"antigravity-switch/internal/store"

	"golang.org/x/sys/windows"
)

func makeTestJWT(email string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	claims := map[string]interface{}{
		"email": email,
		"exp":   time.Now().Add(1 * time.Hour).Unix(),
	}
	claimsBytes, _ := json.Marshal(claims)
	payload := base64.RawURLEncoding.EncodeToString(claimsBytes)
	signature := base64.RawURLEncoding.EncodeToString([]byte("signature"))
	return header + "." + payload + "." + signature
}

func TestExtractEmailFromToken(t *testing.T) {
	// 1. Valid JWT in IDToken
	token := &model.TokenData{
		AccessToken: "access-123",
		IDToken:     makeTestJWT("developer@google.com"),
	}
	email := ExtractEmailFromToken(token)
	if email != "developer@google.com" {
		t.Errorf("expected developer@google.com, got %s", email)
	}

	// 2. Email in RawPayload
	tokenRaw := &model.TokenData{
		AccessToken: "access-456",
		RawPayload:  json.RawMessage(`{"email": "fallback@workspace.com"}`),
	}
	emailRaw := ExtractEmailFromToken(tokenRaw)
	if emailRaw != "fallback@workspace.com" {
		t.Errorf("expected fallback@workspace.com, got %s", emailRaw)
	}

	// 3. Nil token
	if ExtractEmailFromToken(nil) != "" {
		t.Error("expected empty string for nil token")
	}
}

func TestCaptureCurrentSession(t *testing.T) {
	tmpDir := t.TempDir()
	accStore, err := store.NewAccountStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed creating store: %v", err)
	}

	rel := reloader.NewSessionReloader()
	mgr, err := NewAccountManager(accStore, rel)
	if err != nil {
		t.Fatalf("Failed creating manager: %v", err)
	}

	testEmail := "active-session@example.com"
	mgr.SetWincredHandlers(
		func() (*model.TokenData, error) {
			return &model.TokenData{
				AccessToken: "active-access-token",
				IDToken:     makeTestJWT(testEmail),
			}, nil
		},
		func(token *model.TokenData) error {
			return nil
		},
	)

	captured, err := mgr.CaptureCurrentSession()
	if err != nil {
		t.Fatalf("CaptureCurrentSession failed: %v", err)
	}
	if captured != testEmail {
		t.Fatalf("Expected captured %s, got %s", testEmail, captured)
	}

	// Verify store has account
	if !accStore.HasAccount(testEmail) {
		t.Error("Expected account store to contain captured account")
	}
}

func TestGetCurrentActiveEmail(t *testing.T) {
	tmpDir := t.TempDir()
	accStore, _ := store.NewAccountStore(tmpDir)
	rel := reloader.NewSessionReloader()
	mgr, _ := NewAccountManager(accStore, rel)

	// Mock language server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "GetUserStatus") || strings.Contains(r.URL.Path, "user_status") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"email":"live-ram@example.com","name":"Live RAM User"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	parts := strings.Split(server.URL, ":")
	port, _ := strconv.Atoi(parts[len(parts)-1])

	client := connector.NewClientWithHTTP(&connector.ServerInfo{
		PID:       9999,
		Port:      port,
		CSRFToken: "csrf-token-123",
	}, server.Client())

	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return client, nil
	})

	email, err := mgr.GetCurrentActiveEmail(context.Background())
	if err != nil {
		t.Fatalf("GetCurrentActiveEmail failed: %v", err)
	}
	if email != "live-ram@example.com" {
		t.Errorf("Expected live-ram@example.com, got %s", email)
	}

	// Test fallback to wincred when client finder fails
	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return nil, connector.ErrLanguageServerNotFound
	})
	mgr.SetWincredHandlers(
		func() (*model.TokenData, error) {
			return &model.TokenData{
				AccessToken: "token-fallback",
				IDToken:     makeTestJWT("wincred-fallback@example.com"),
			}, nil
		},
		func(token *model.TokenData) error {
			return nil
		},
	)

	fallbackEmail, err := mgr.GetCurrentActiveEmail(context.Background())
	if err != nil {
		t.Fatalf("Fallback GetCurrentActiveEmail failed: %v", err)
	}
	if fallbackEmail != "wincred-fallback@example.com" {
		t.Errorf("Expected wincred-fallback@example.com, got %s", fallbackEmail)
	}
}

func TestGetAccountStates(t *testing.T) {
	tmpDir := t.TempDir()
	accStore, _ := store.NewAccountStore(tmpDir)

	// Save one ready account to store
	readyEmail := "ready-user@example.com"
	_ = accStore.SaveAccount(readyEmail, &model.TokenData{
		AccessToken: "ready-token",
		IDToken:     makeTestJWT(readyEmail),
	})

	rel := reloader.NewSessionReloader()
	mgr, _ := NewAccountManager(accStore, rel)

	activeEmail := "active-user@example.com"
	mgr.SetWincredHandlers(
		func() (*model.TokenData, error) {
			return &model.TokenData{
				AccessToken: "active-token",
				IDToken:     makeTestJWT(activeEmail),
			}, nil
		},
		func(token *model.TokenData) error {
			return nil
		},
	)
	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return nil, connector.ErrLanguageServerNotFound
	})

	overview, err := mgr.GetAccountStates(context.Background())
	if err != nil {
		t.Fatalf("GetAccountStates failed: %v", err)
	}

	if overview.ActiveEmail != activeEmail {
		t.Errorf("Expected active email %s, got %s", activeEmail, overview.ActiveEmail)
	}

	if len(overview.Accounts) < 2 {
		t.Fatalf("Expected at least 2 accounts, got %d", len(overview.Accounts))
	}

	// Active account must be first
	if !overview.Accounts[0].IsActive || overview.Accounts[0].Email != activeEmail {
		t.Errorf("Expected first account to be active %s, got %+v", activeEmail, overview.Accounts[0])
	}
	if overview.Accounts[0].DisplayBadge != "[ĐANG DÙNG]" {
		t.Errorf("Expected [ĐANG DÙNG] badge, got %s", overview.Accounts[0].DisplayBadge)
	}

	// Ready account must be present
	foundReady := false
	for _, acc := range overview.Accounts {
		if acc.Email == readyEmail {
			foundReady = true
			if acc.State != model.StateReady || acc.DisplayBadge != "[⚡ Chuyển]" {
				t.Errorf("Ready account bad state/badge: %+v", acc)
			}
		}
	}
	if !foundReady {
		t.Errorf("Expected ready account %s in overview", readyEmail)
	}
}

func TestSwitchAccount(t *testing.T) {
	tmpDir := t.TempDir()
	accStore, _ := store.NewAccountStore(tmpDir)

	targetEmail := "target-switch@example.com"
	_ = accStore.SaveAccount(targetEmail, &model.TokenData{
		AccessToken: "target-access-token",
		IDToken:     makeTestJWT(targetEmail),
	})

	rel := reloader.NewSessionReloader()
	rel.WindowFinder = func() ([]windows.HWND, error) {
		return []windows.HWND{windows.HWND(101)}, nil
	}
	rel.KeySender = func(hwnds []windows.HWND) error {
		return nil
	}

	mgr, _ := NewAccountManager(accStore, rel)

	currentEmail := "current-user@example.com"
	var writtenToken *model.TokenData
	mgr.SetWincredHandlers(
		func() (*model.TokenData, error) {
			return &model.TokenData{
				AccessToken: "current-access-token",
				IDToken:     makeTestJWT(currentEmail),
			}, nil
		},
		func(token *model.TokenData) error {
			writtenToken = token
			return nil
		},
	)

	// Mock language server to track whether AuthLogout is called (P0-02 verification)
	var authLogoutCount int
	mockActiveEmail := "current-user@example.com"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "AuthLogout") {
			authLogoutCount++
			w.WriteHeader(http.StatusOK)
			return
		}
		if strings.Contains(r.URL.Path, "GetUserStatus") || strings.Contains(r.URL.Path, "user_status") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"email":"` + mockActiveEmail + `"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	parts := strings.Split(server.URL, ":")
	port, _ := strconv.Atoi(parts[len(parts)-1])
	client := connector.NewClientWithHTTP(&connector.ServerInfo{
		PID:       9999,
		Port:      port,
		CSRFToken: "test-csrf-token",
	}, server.Client())

	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return client, nil
	})

	// 1. Switch to missing account -> fails
	_, errMissing := mgr.SwitchAccount(context.Background(), "non-existent@example.com", false)
	if errMissing == nil {
		t.Fatal("Expected error switching to non-existent account, got nil")
	}

	// 2. Switch to target account -> succeeds and MUST NOT call AuthLogout (P0-02 parity)
	result, err := mgr.SwitchAccount(context.Background(), targetEmail, false)
	if err != nil {
		t.Fatalf("SwitchAccount failed: %v", err)
	}
	if !result.Success {
		t.Errorf("Expected result.Success = true, got false with message: %s", result.Message)
	}
	if result.TargetEmail != targetEmail {
		t.Errorf("Expected target email %s, got %s", targetEmail, result.TargetEmail)
	}
	if writtenToken == nil || writtenToken.AccessToken != "target-access-token" {
		t.Errorf("Expected target token written to wincred, got %+v", writtenToken)
	}
	if authLogoutCount > 0 {
		t.Fatalf("AuthLogout was invoked (%d times) during SwitchAccount! Expected AuthLogout to be bypassed (P0-02)", authLogoutCount)
	}

	// 2b. Verify default SwitchAccount (variadic, no forceLogout argument) also bypasses AuthLogout
	authLogoutCount = 0
	anotherEmail := "another-switch@example.com"
	_ = accStore.SaveAccount(anotherEmail, &model.TokenData{
		AccessToken: "another-access-token",
		IDToken:     makeTestJWT(anotherEmail),
	})
	resDefault, err := mgr.SwitchAccount(context.Background(), anotherEmail)
	if err != nil || !resDefault.Success {
		t.Fatalf("Default SwitchAccount failed: %v", err)
	}
	if authLogoutCount > 0 {
		t.Fatalf("AuthLogout was invoked (%d times) during default SwitchAccount! Expected AuthLogout to be bypassed", authLogoutCount)
	}

	// 2c. Verify that when forceLogout=true is explicitly requested, AuthLogout is called
	authLogoutCount = 0
	resForce, err := mgr.SwitchAccount(context.Background(), targetEmail, true)
	if err != nil || !resForce.Success {
		t.Fatalf("Force logout SwitchAccount failed: %v", err)
	}
	if authLogoutCount != 1 {
		t.Fatalf("Expected AuthLogout to be invoked exactly once when forceLogout=true, got %d", authLogoutCount)
	}

	// 3. Switch to already active account -> no-op success
	mockActiveEmail = targetEmail
	mgr.SetWincredHandlers(
		func() (*model.TokenData, error) {
			return &model.TokenData{
				AccessToken: "target-access-token",
				IDToken:     makeTestJWT(targetEmail),
			}, nil
		},
		func(token *model.TokenData) error {
			return nil
		},
	)
	noopResult, err := mgr.SwitchAccount(context.Background(), targetEmail, false)
	if err != nil {
		t.Fatalf("No-op switch failed: %v", err)
	}
	if !noopResult.Success || !strings.Contains(noopResult.Message, "hiện đang hoạt động") {
		t.Errorf("Unexpected no-op message: %s", noopResult.Message)
	}
}
