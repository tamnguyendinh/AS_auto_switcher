package manager

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"errors"
	"sync/atomic"
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

	rel.ServerDiscoverer = func() (*connector.ServerInfo, error) {
		return &connector.ServerInfo{
			PID:       9999,
			Port:      port,
			CSRFToken: "test-csrf-token",
		}, nil
	}

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
	var recycleCount atomic.Int32
	rel.ProcessKiller = func(ctx context.Context) error {
		recycleCount.Add(1)
		return nil
	}
	rel.Sleeper = func(ctx context.Context, d time.Duration) error {
		return nil
	}

	mgr, _ := NewAccountManager(accStore, rel)

	currentEmail := "current-user@example.com"
	mockActiveEmail := currentEmail
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
			if email := ExtractEmailFromToken(token); email != "" {
				mockActiveEmail = email
			}
			return nil
		},
	)

	// Mock language server to track whether AuthLogout is called (P0-02 verification)
	var authLogoutCount int
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
	if recycleCount.Load() == 0 {
		t.Fatal("Expected language server to be recycled during SwitchAccount (P0-03 parity)")
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

	// 2d. Verify graceful handling when language server recycling encounters an error
	rel.ServerDiscoverer = func() (*connector.ServerInfo, error) {
		return nil, errors.New("simulated discover error")
	}
	warningEmail := "warning-switch@example.com"
	_ = accStore.SaveAccount(warningEmail, &model.TokenData{
		AccessToken: "warning-access-token",
		IDToken:     makeTestJWT(warningEmail),
	})
	resWarn, err := mgr.SwitchAccount(context.Background(), warningEmail)
	if err != nil || !resWarn.Success {
		t.Fatalf("SwitchAccount should succeed even with recycle warning, got err=%v, res=%+v", err, resWarn)
	}
	if !strings.Contains(resWarn.Message, "cảnh báo khởi động lại Language Server") {
		t.Errorf("Expected warning in message, got: %s", resWarn.Message)
	}

	// Restore discoverer for subsequent tests
	rel.ServerDiscoverer = func() (*connector.ServerInfo, error) {
		return &connector.ServerInfo{
			PID:       9999,
			Port:      port,
			CSRFToken: "test-csrf-token",
		}, nil
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


func TestSwitchAccount_WithSessionContinuation(t *testing.T) {
	tmpDir := t.TempDir()
	accStore, _ := store.NewAccountStore(tmpDir)

	targetEmail := "switch-continuation@example.com"
	currentEmail := "current-before-switch@example.com"
	mockActiveEmail := currentEmail

	_ = accStore.SaveAccount(targetEmail, &model.TokenData{
		AccessToken: "target-access-token",
		IDToken:     makeTestJWT(targetEmail),
	})

	var snapshotCaptured bool
	var continuationExecuted bool
	var continuationThreadID string

	rel := reloader.NewSessionReloader()
	rel.WindowFinder = func() ([]windows.HWND, error) {
		return []windows.HWND{windows.HWND(101)}, nil
	}
	rel.ProcessKiller = func(ctx context.Context) error { return nil }
	rel.Sleeper = func(ctx context.Context, d time.Duration) error { return nil }

	rel.SnapshotCapturer = func(storagePath, brainDir, outputDir string) (string, []reloader.ActiveThread, error) {
		snapshotCaptured = true
		return "test_snapshot.json", []reloader.ActiveThread{
			{ThreadID: "thread-active-999", Project: "G:/project", IsMainPane: true},
		}, nil
	}
	rel.ContinuationRunner = func(ctx context.Context, client *connector.Client, activeThreads []reloader.ActiveThread, promptText string) error {
		continuationExecuted = true
		if len(activeThreads) > 0 {
			continuationThreadID = activeThreads[0].ThreadID
		}
		return nil
	}

	mgr, _ := NewAccountManager(accStore, rel)
	mgr.SetWincredHandlers(
		func() (*model.TokenData, error) {
			return &model.TokenData{
				AccessToken: "current-access-token",
				IDToken:     makeTestJWT(mockActiveEmail),
			}, nil
		},
		func(token *model.TokenData) error {
			if email := ExtractEmailFromToken(token); email != "" {
				mockActiveEmail = email
			}
			return nil
		},
	)

	// Mock language server for GetUserStatus
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		PID:       8888,
		Port:      port,
		CSRFToken: "test-csrf-token",
	}, server.Client())

	rel.ServerDiscoverer = func() (*connector.ServerInfo, error) {
		return &connector.ServerInfo{
			PID:       8888,
			Port:      port,
			CSRFToken: "test-csrf-token",
		}, nil
	}

	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return client, nil
	})

	result, err := mgr.SwitchAccount(context.Background(), targetEmail, false)
	if err != nil {
		t.Fatalf("SwitchAccount failed: %v", err)
	}

	if !result.Success {
		t.Fatalf("Expected switch to succeed, message: %s", result.Message)
	}

	if !snapshotCaptured {
		t.Error("Expected active threads snapshot to be captured before recycle (P0-04)")
	}

	if !continuationExecuted {
		t.Error("Expected continuation to be executed after language server restart (P0-04)")
	}

	if continuationThreadID != "thread-active-999" {
		t.Errorf("Expected continuation on thread-active-999, got %s", continuationThreadID)
	}

	if len(result.ActiveThreads) != 1 || result.ActiveThreads[0].ThreadID != "thread-active-999" {
		t.Errorf("Expected result.ActiveThreads to contain thread-active-999, got %+v", result.ActiveThreads)
	}
}
