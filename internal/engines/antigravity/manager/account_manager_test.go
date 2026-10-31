package manager

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"as-auto-switcher/internal/engines/antigravity/connector"
	"as-auto-switcher/internal/engines/antigravity/model"
	"as-auto-switcher/internal/engines/antigravity/reloader"
	"as-auto-switcher/internal/engines/antigravity/store"
	"as-auto-switcher/internal/engines/antigravity/wincred"

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
	if overview.Accounts[0].DisplayBadge != "[✔ Dùng]" {
		t.Errorf("Expected [✔ Dùng] badge, got %s", overview.Accounts[0].DisplayBadge)
	}

	// Ready account must be present
	foundReady := false
	for _, acc := range overview.Accounts {
		if acc.Email == readyEmail {
			foundReady = true
			if acc.State != model.StateReady || acc.DisplayBadge != "[Chuyển]" {
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

	// Mock ServerDiscoverer to avoid probing real OS language_server.exe processes
	rel.ServerDiscoverer = func() (*connector.ServerInfo, error) {
		return &connector.ServerInfo{
			PID:       9999,
			Port:      port,
			CSRFToken: "test-csrf-token",
		}, nil
	}

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
		t.Fatalf("Expected language server to be recycled during SwitchAccount, got %d", recycleCount.Load())
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
	mockActiveAtomicEmail := "current@example.com"
	atomicServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"email":"` + mockActiveAtomicEmail + `"}`))
	}))
	defer atomicServer.Close()
	partsAtomic := strings.Split(atomicServer.URL, ":")
	portAtomic, _ := strconv.Atoi(partsAtomic[len(partsAtomic)-1])
	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return connector.NewClientWithHTTP(&connector.ServerInfo{PID: 1234, Port: portAtomic, CSRFToken: "tok"}, atomicServer.Client()), nil
	})
	rel.ServerDiscoverer = func() (*connector.ServerInfo, error) {
		return nil, errors.New("simulated discover error")
	}
	warningEmail := "warning-switch@example.com"
	_ = accStore.SaveAccount(warningEmail, &model.TokenData{
		AccessToken: "warning-access-token",
		IDToken:     makeTestJWT(warningEmail),
	})
	resWarn, err := mgr.SwitchAccount(context.Background(), warningEmail)
	if err == nil || resWarn.Success {
		t.Fatalf("SwitchAccount should fail when recycle/discovery encounters an error, got err=%v, res=%+v", err, resWarn)
	}
	if !strings.Contains(resWarn.Message, "Khởi động lại Language Server thất bại") {
		t.Errorf("Expected failure in message, got: %s", resWarn.Message)
	}

	// Restore discoverer for subsequent tests
	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return client, nil
	})
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

type mockOAuthRefresher struct {
	callCount int
	fn        func(ctx context.Context, refreshToken string) (*connector.TokenRefreshResult, error)
}

func (m *mockOAuthRefresher) RefreshToken(ctx context.Context, refreshToken string) (*connector.TokenRefreshResult, error) {
	m.callCount++
	if m.fn != nil {
		return m.fn(ctx, refreshToken)
	}
	return &connector.TokenRefreshResult{
		AccessToken:  "mock-new-access-token",
		RefreshToken: refreshToken,
		ExpiresIn:    3600,
		Expiry:       time.Now().Add(1 * time.Hour),
	}, nil
}

func TestSwitchAccount_ProactiveTokenRefresh(t *testing.T) {
	tmpDir := t.TempDir()
	accStore, err := store.NewAccountStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed creating store: %v", err)
	}

	rel := reloader.NewSessionReloader()
	rel.WindowFinder = func() ([]windows.HWND, error) {
		return []windows.HWND{windows.HWND(101)}, nil
	}
	rel.ProcessKiller = func(ctx context.Context) error { return nil }
	rel.Sleeper = func(ctx context.Context, d time.Duration) error { return nil }
	rel.ServerDiscoverer = func() (*connector.ServerInfo, error) {
		return &connector.ServerInfo{
			PID:       9999,
			Port:      12345,
			CSRFToken: "test-csrf-token",
		}, nil
	}

	mgr, err := NewAccountManager(accStore, rel)
	if err != nil {
		t.Fatalf("Failed creating manager: %v", err)
	}

	var writtenToken *model.TokenData
	mockActiveEmail := "current@example.com"
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"email":"` + mockActiveEmail + `"}`))
	}))
	defer mockServer.Close()
	parts := strings.Split(mockServer.URL, ":")
	port, _ := strconv.Atoi(parts[len(parts)-1])
	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return connector.NewClientWithHTTP(&connector.ServerInfo{PID: 1234, Port: port, CSRFToken: "tok"}, mockServer.Client()), nil
	})

	mgr.SetWincredHandlers(
		func() (*model.TokenData, error) {
			return &model.TokenData{
				AccessToken: "current-active",
				IDToken:     makeTestJWT(mockActiveEmail),
			}, nil
		},
		func(token *model.TokenData) error {
			writtenToken = token
			if e := ExtractEmailFromToken(token); e != "" {
				mockActiveEmail = e
			}
			return nil
		},
	)

	mockRef := &mockOAuthRefresher{}
	mgr.SetOAuthRefresher(mockRef)

	// 1. Expired token -> must proactively refresh and save to store + WinCred
	expiredEmail := "expired-user@example.com"
	pastExpiry := time.Now().Add(-10 * time.Minute)
	err = accStore.SaveAccount(expiredEmail, &model.TokenData{
		AccessToken:  "stale-access-token",
		RefreshToken: "valid-refresh-token",
		Expiry:       pastExpiry,
		IDToken:      makeTestJWT(expiredEmail),
	})
	if err != nil {
		t.Fatalf("Failed saving expired account: %v", err)
	}

	res, err := mgr.SwitchAccount(context.Background(), expiredEmail, false)
	if err != nil || !res.Success {
		t.Fatalf("SwitchAccount failed for expired account: %v, res=%+v", err, res)
	}

	if mockRef.callCount != 1 {
		t.Fatalf("Expected mockRefresher to be called 1 time, got %d", mockRef.callCount)
	}
	if writtenToken == nil || writtenToken.AccessToken != "mock-new-access-token" {
		t.Fatalf("Expected fresh access token written to wincred, got %+v", writtenToken)
	}

	// Verify disk store was updated
	diskToken, err := accStore.GetAccount(expiredEmail)
	if err != nil {
		t.Fatalf("Failed getting account from store: %v", err)
	}
	if diskToken.AccessToken != "mock-new-access-token" {
		t.Errorf("Expected store to have fresh access token, got %s", diskToken.AccessToken)
	}
	if time.Until(diskToken.Expiry) < 50*time.Minute {
		t.Errorf("Expected store token expiry to be updated to future, got %v", diskToken.Expiry)
	}

	// 2. Token with > 5 minutes remaining -> MUST NOT refresh
	mockRef.callCount = 0
	validEmail := "valid-user@example.com"
	futureExpiry := time.Now().Add(30 * time.Minute)
	_ = accStore.SaveAccount(validEmail, &model.TokenData{
		AccessToken:  "valid-access-token",
		RefreshToken: "valid-refresh-token",
		Expiry:       futureExpiry,
		IDToken:      makeTestJWT(validEmail),
	})

	resValid, err := mgr.SwitchAccount(context.Background(), validEmail, false)
	if err != nil || !resValid.Success {
		t.Fatalf("SwitchAccount failed for valid account: %v", err)
	}
	if mockRef.callCount != 0 {
		t.Fatalf("Expected 0 refresh calls for valid token, got %d", mockRef.callCount)
	}
	if writtenToken.AccessToken != "valid-access-token" {
		t.Errorf("Expected valid-access-token to be preserved, got %s", writtenToken.AccessToken)
	}

	// 3. Refresh failure (e.g. offline/network failure) -> graceful fallback, keeps existing token
	mockRef.callCount = 0
	mockRef.fn = func(ctx context.Context, refreshToken string) (*connector.TokenRefreshResult, error) {
		return nil, errors.New("network unreachable")
	}

	offlineEmail := "offline-user@example.com"
	_ = accStore.SaveAccount(offlineEmail, &model.TokenData{
		AccessToken:  "offline-access-token",
		RefreshToken: "offline-refresh-token",
		Expiry:       time.Now().Add(-5 * time.Minute),
		IDToken:      makeTestJWT(offlineEmail),
	})

	resOffline, err := mgr.SwitchAccount(context.Background(), offlineEmail, false)
	if err != nil || !resOffline.Success {
		t.Fatalf("SwitchAccount should proceed gracefully on refresh network error, got err=%v", err)
	}
	if mockRef.callCount != 1 {
		t.Fatalf("Expected 1 refresh attempt, got %d", mockRef.callCount)
	}
	if writtenToken.AccessToken != "offline-access-token" {
		t.Errorf("Expected existing token to be preserved on refresh failure, got %s", writtenToken.AccessToken)
	}
}

func TestSwitchAccount_AtomicAppStorageSync(t *testing.T) {
	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "app_storage.json")

	// Pre-populate app_storage.json with existing configuration
	initialConfig := map[string]interface{}{
		"jetski.ui.layout":                    "split-pane",
		"zoomLevel":                           1.0,
		"jetski.onboarding.lastLoginUsername": "old@example.com",
		"jetski.onboarding.lastLoginIsGcpTos": "false",
	}
	raw, _ := json.MarshalIndent(initialConfig, "", "  ")
	_ = os.WriteFile(storagePath, raw, 0644)

	accStore, _ := store.NewAccountStore(tmpDir)

	gcpEmail := "gcp-developer@company.com"
	_ = accStore.SaveAccount(gcpEmail, &model.TokenData{
		AccessToken: "gcp-access-token",
		Expiry:      time.Now().Add(1 * time.Hour),
		AuthMethod:  "gcp",
		IDToken:     makeTestJWT(gcpEmail),
	})

	consumerEmail := "consumer-user@gmail.com"
	_ = accStore.SaveAccount(consumerEmail, &model.TokenData{
		AccessToken: "consumer-access-token",
		Expiry:      time.Now().Add(1 * time.Hour),
		AuthMethod:  "consumer",
		IDToken:     makeTestJWT(consumerEmail),
	})

	rel := reloader.NewSessionReloader()
	rel.StoragePath = storagePath
	rel.ProcessKiller = func(ctx context.Context) error { return nil }
	rel.Sleeper = func(ctx context.Context, d time.Duration) error { return nil }

	mgr, _ := NewAccountManager(accStore, rel)

	mockActiveAtomicEmail := "current@example.com"
	atomicServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"email":"` + mockActiveAtomicEmail + `"}`))
	}))
	defer atomicServer.Close()
	partsAtomic := strings.Split(atomicServer.URL, ":")
	portAtomic, _ := strconv.Atoi(partsAtomic[len(partsAtomic)-1])
	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return connector.NewClientWithHTTP(&connector.ServerInfo{PID: 1234, Port: portAtomic, CSRFToken: "tok"}, atomicServer.Client()), nil
	})

	rel.ServerDiscoverer = func() (*connector.ServerInfo, error) {
		return &connector.ServerInfo{
			PID:       1234,
			Port:      portAtomic,
			CSRFToken: "tok",
		}, nil
	}

	// Verify that SyncAppStorageToS runs BEFORE wincredWriter
	var appStorageDuringWincred map[string]interface{}
	mgr.SetWincredHandlers(
		func() (*model.TokenData, error) {
			return &model.TokenData{
				AccessToken: "current-tok",
				IDToken:     makeTestJWT("current@example.com"),
			}, nil
		},
		func(token *model.TokenData) error {
			// Read app_storage.json AT THE EXACT MOMENT wincred is written
			if data, err := os.ReadFile(storagePath); err == nil {
				_ = json.Unmarshal(data, &appStorageDuringWincred)
			}
			if e := ExtractEmailFromToken(token); e != "" {
				mockActiveAtomicEmail = e
			}
			return nil
		},
	)

	// 1. Switch to GCP account
	resGcp, err := mgr.SwitchAccount(context.Background(), gcpEmail, false)
	if err != nil || !resGcp.Success {
		t.Fatalf("SwitchAccount to GCP account failed: %v", err)
	}

	// Verify app_storage was written BEFORE wincred writer ran
	if appStorageDuringWincred["jetski.onboarding.lastLoginUsername"] != gcpEmail {
		t.Errorf("Expected app_storage to be synced BEFORE wincred write, got %v", appStorageDuringWincred["jetski.onboarding.lastLoginUsername"])
	}
	if appStorageDuringWincred["jetski.onboarding.lastLoginIsGcpTos"] != "true" {
		t.Errorf("Expected isGcpTos 'true' during wincred write, got %v", appStorageDuringWincred["jetski.onboarding.lastLoginIsGcpTos"])
	}

	// Verify existing keys preserved
	if appStorageDuringWincred["jetski.ui.layout"] != "split-pane" {
		t.Errorf("Expected jetski.ui.layout preserved, got %v", appStorageDuringWincred["jetski.ui.layout"])
	}

	// 2. Switch to Consumer account
	resConsumer, err := mgr.SwitchAccount(context.Background(), consumerEmail, false)
	if err != nil || !resConsumer.Success {
		t.Fatalf("SwitchAccount to Consumer account failed: %v", err)
	}

	if appStorageDuringWincred["jetski.onboarding.lastLoginUsername"] != consumerEmail {
		t.Errorf("Expected consumer email in app_storage, got %v", appStorageDuringWincred["jetski.onboarding.lastLoginUsername"])
	}
	if appStorageDuringWincred["jetski.onboarding.lastLoginIsGcpTos"] != "false" {
		t.Errorf("Expected isGcpTos 'false' for consumer account, got %v", appStorageDuringWincred["jetski.onboarding.lastLoginIsGcpTos"])
	}
}

func TestSwitchAccount_CivilizedReAuth_HotSwapped(t *testing.T) {
	tmpDir := t.TempDir()
	accStore, _ := store.NewAccountStore(tmpDir)

	targetEmail := "reauth-target@example.com"
	currentEmail := "reauth-current@example.com"
	mockEmail := currentEmail

	_ = accStore.SaveAccount(targetEmail, &model.TokenData{
		AccessToken: "reauth-access-token",
		Expiry:      time.Now().Add(1 * time.Hour),
		IDToken:     makeTestJWT(targetEmail),
	})

	var reauthInvoked atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "ReAuth") || strings.Contains(r.URL.Path, "reauth") || strings.Contains(r.URL.Path, "Reload") {
			reauthInvoked.Store(true)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("{}"))
			return
		}
		if strings.Contains(r.URL.Path, "GetUserStatus") || strings.Contains(r.URL.Path, "user_status") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"email":"` + mockEmail + `"}`))
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
		CSRFToken: "test-reauth-csrf",
	}, server.Client())

	rel := reloader.NewSessionReloader()
	rel.StoragePath = filepath.Join(tmpDir, "app_storage.json")
	var killerCallCount atomic.Int32
	rel.ProcessKiller = func(ctx context.Context) error {
		killerCallCount.Add(1)
		return nil
	}
	rel.Sleeper = func(ctx context.Context, d time.Duration) error { return nil }

	mgr, _ := NewAccountManager(accStore, rel)
	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return client, nil
	})
	rel.ServerDiscoverer = func() (*connector.ServerInfo, error) {
		return &connector.ServerInfo{
			PID:       8888,
			Port:      port,
			CSRFToken: "test-reauth-csrf",
		}, nil
	}
	mgr.SetWincredHandlers(
		func() (*model.TokenData, error) {
			return &model.TokenData{
				AccessToken: "current-tok",
				IDToken:     makeTestJWT(currentEmail),
			}, nil
		},
		func(token *model.TokenData) error {
			if email := ExtractEmailFromToken(token); email != "" {
				mockEmail = email
			}
			return nil
		},
	)

	result, err := mgr.SwitchAccount(context.Background(), targetEmail, false)
	if err != nil {
		t.Fatalf("SwitchAccount failed: %v", err)
	}
	if !result.Success {
		t.Fatalf("Expected success, got false: %s", result.Message)
	}
	if killerCallCount.Load() != 1 {
		t.Errorf("Expected 1 killer call for standard Antigravity recycle, got %d", killerCallCount.Load())
	}
	if reauthInvoked.Load() {
		t.Errorf("Expected RPC ReAuth to be eliminated, but TriggerReauth was called")
	}
}

func TestSwitchAccount_ReAuthFallback_TriggerReauthError(t *testing.T) {
	tmpDir := t.TempDir()
	accStore, _ := store.NewAccountStore(tmpDir)

	targetEmail := "fallback-target@example.com"
	currentEmail := "fallback-current@example.com"

	_ = accStore.SaveAccount(targetEmail, &model.TokenData{
		AccessToken: "fallback-access-token",
		Expiry:      time.Now().Add(1 * time.Hour),
		IDToken:     makeTestJWT(targetEmail),
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "ReAuth") || strings.Contains(r.URL.Path, "reauth") || strings.Contains(r.URL.Path, "Reload") {
			// Simulate LS rejecting ReAuth (e.g. 500 internal error)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if strings.Contains(r.URL.Path, "GetUserStatus") || strings.Contains(r.URL.Path, "user_status") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"email":"` + currentEmail + `"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	parts := strings.Split(server.URL, ":")
	port, _ := strconv.Atoi(parts[len(parts)-1])
	client := connector.NewClientWithHTTP(&connector.ServerInfo{
		PID:       8889,
		Port:      port,
		CSRFToken: "test-fallback-csrf",
	}, server.Client())

	rel := reloader.NewSessionReloader()
	rel.StoragePath = filepath.Join(tmpDir, "app_storage.json")
	var killerCallCount atomic.Int32
	rel.ProcessKiller = func(ctx context.Context) error {
		killerCallCount.Add(1)
		return nil
	}
	rel.Sleeper = func(ctx context.Context, d time.Duration) error { return nil }
	rel.ServerDiscoverer = func() (*connector.ServerInfo, error) {
		return &connector.ServerInfo{PID: 9001, Port: port, CSRFToken: "new-csrf"}, nil
	}

	mgr, _ := NewAccountManager(accStore, rel)
	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return client, nil
	})
	rel.ServerDiscoverer = func() (*connector.ServerInfo, error) {
		return &connector.ServerInfo{
			PID:       8888,
			Port:      port,
			CSRFToken: "test-reauth-csrf",
		}, nil
	}
	mgr.SetWincredHandlers(
		func() (*model.TokenData, error) {
			return &model.TokenData{
				AccessToken: "current-tok",
				IDToken:     makeTestJWT(currentEmail),
			}, nil
		},
		func(token *model.TokenData) error {
			return nil
		},
	)

	result, err := mgr.SwitchAccount(context.Background(), targetEmail, false)
	if err == nil || result.Success {
		t.Fatalf("Expected failure due to RAM mismatch when server returns current email, got res=%+v", result)
	}
	if killerCallCount.Load() != 1 {
		t.Errorf("Expected exactly 1 killer call on recycle, got %d", killerCallCount.Load())
	}
}

func TestSwitchAccount_ReAuthFallback_EmailMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	accStore, _ := store.NewAccountStore(tmpDir)

	targetEmail := "mismatch-target@example.com"
	currentEmail := "mismatch-current@example.com"

	_ = accStore.SaveAccount(targetEmail, &model.TokenData{
		AccessToken: "mismatch-access-token",
		Expiry:      time.Now().Add(1 * time.Hour),
		IDToken:     makeTestJWT(targetEmail),
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "ReAuth") || strings.Contains(r.URL.Path, "reauth") || strings.Contains(r.URL.Path, "Reload") {
			// TriggerReauth returns OK, but RAM never updates
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("{}"))
			return
		}
		if strings.Contains(r.URL.Path, "GetUserStatus") || strings.Contains(r.URL.Path, "user_status") {
			// Stale email returned from RAM
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"email":"stale-unswitched@example.com"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	parts := strings.Split(server.URL, ":")
	port, _ := strconv.Atoi(parts[len(parts)-1])
	client := connector.NewClientWithHTTP(&connector.ServerInfo{
		PID:       8890,
		Port:      port,
		CSRFToken: "test-mismatch-csrf",
	}, server.Client())

	rel := reloader.NewSessionReloader()
	rel.StoragePath = filepath.Join(tmpDir, "app_storage.json")
	var killerCallCount atomic.Int32
	rel.ProcessKiller = func(ctx context.Context) error {
		killerCallCount.Add(1)
		return nil
	}
	rel.Sleeper = func(ctx context.Context, d time.Duration) error { return nil }
	rel.ServerDiscoverer = func() (*connector.ServerInfo, error) {
		return &connector.ServerInfo{PID: 9002, Port: port, CSRFToken: "new-csrf"}, nil
	}

	mgr, _ := NewAccountManager(accStore, rel)
	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return client, nil
	})
	rel.ServerDiscoverer = func() (*connector.ServerInfo, error) {
		return &connector.ServerInfo{
			PID:       8888,
			Port:      port,
			CSRFToken: "test-reauth-csrf",
		}, nil
	}
	mgr.SetWincredHandlers(
		func() (*model.TokenData, error) {
			return &model.TokenData{
				AccessToken: "current-tok",
				IDToken:     makeTestJWT(currentEmail),
			}, nil
		},
		func(token *model.TokenData) error {
			return nil
		},
	)

	result, err := mgr.SwitchAccount(context.Background(), targetEmail, false)
	if err == nil || result.Success {
		t.Fatalf("Expected failure due to RAM mismatch when server returns stale email, got res=%+v", result)
	}
	if killerCallCount.Load() != 1 {
		t.Errorf("Expected exactly 1 killer call on recycle, got %d", killerCallCount.Load())
	}
}

func TestSwitchAccount_ReAuthFallback_ClientNil(t *testing.T) {
	tmpDir := t.TempDir()
	accStore, _ := store.NewAccountStore(tmpDir)

	targetEmail := "clientnil-target@example.com"
	currentEmail := "clientnil-current@example.com"

	_ = accStore.SaveAccount(targetEmail, &model.TokenData{
		AccessToken: "clientnil-access-token",
		Expiry:      time.Now().Add(1 * time.Hour),
		IDToken:     makeTestJWT(targetEmail),
	})

	rel := reloader.NewSessionReloader()
	rel.StoragePath = filepath.Join(tmpDir, "app_storage.json")
	var killerCallCount atomic.Int32
	rel.ProcessKiller = func(ctx context.Context) error {
		killerCallCount.Add(1)
		return nil
	}
	rel.Sleeper = func(ctx context.Context, d time.Duration) error { return nil }
	rel.ServerDiscoverer = func() (*connector.ServerInfo, error) {
		return &connector.ServerInfo{PID: 9003, Port: 54321, CSRFToken: "new-csrf"}, nil
	}

	mgr, _ := NewAccountManager(accStore, rel)
	// client is nil (LS not running)
	mockActiveAtomicEmail := "current@example.com"
	atomicServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"email":"` + mockActiveAtomicEmail + `"}`))
	}))
	defer atomicServer.Close()
	partsAtomic := strings.Split(atomicServer.URL, ":")
	portAtomic, _ := strconv.Atoi(partsAtomic[len(partsAtomic)-1])
	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return connector.NewClientWithHTTP(&connector.ServerInfo{PID: 1234, Port: portAtomic, CSRFToken: "tok"}, atomicServer.Client()), nil
	})
	mgr.SetWincredHandlers(
		func() (*model.TokenData, error) {
			return &model.TokenData{
				AccessToken: "current-tok",
				IDToken:     makeTestJWT(currentEmail),
			}, nil
		},
		func(token *model.TokenData) error {
			return nil
		},
	)

	result, err := mgr.SwitchAccount(context.Background(), targetEmail, false)
	if err == nil || result.Success {
		t.Fatalf("Expected failure when respawned language server client is nil, got res=%+v", result)
	}
	if killerCallCount.Load() != 1 {
		t.Errorf("Expected exactly 1 killer call on recycle, got %d", killerCallCount.Load())
	}
}

func TestAccountManager_SetAndGetAccountStatus(t *testing.T) {
	tmpDir := t.TempDir()
	accStore, err := store.NewAccountStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed creating store: %v", err)
	}

	mgr, err := NewAccountManager(accStore, nil)
	if err != nil {
		t.Fatalf("Failed creating manager: %v", err)
	}

	email := "mgr.status@example.com"
	token := &model.TokenData{
		AccessToken:  "mgr_access",
		RefreshToken: "mgr_refresh",
	}

	// 1. Initially non-existent account -> GetAccountStatus returns "on"
	st := mgr.GetAccountStatus(email)
	if st != "on" {
		t.Errorf("expected default 'on' for missing account, got: %q", st)
	}

	// 2. Save account to store
	if err := accStore.SaveAccount(email, token); err != nil {
		t.Fatalf("SaveAccount failed: %v", err)
	}

	// Should still be "on"
	st = mgr.GetAccountStatus(email)
	if st != "on" {
		t.Errorf("expected 'on' after initial save, got: %q", st)
	}

	// 3. SetAccountStatus to "off"
	if err := mgr.SetAccountStatus(email, "off"); err != nil {
		t.Fatalf("SetAccountStatus failed: %v", err)
	}
	st = mgr.GetAccountStatus(email)
	if st != "off" {
		t.Errorf("expected 'off' after SetAccountStatus, got: %q", st)
	}

	// 4. SetAccountStatus to "on"
	if err := mgr.SetAccountStatus(email, "on"); err != nil {
		t.Fatalf("SetAccountStatus to 'on' failed: %v", err)
	}
	st = mgr.GetAccountStatus(email)
	if st != "on" {
		t.Errorf("expected 'on' after SetAccountStatus to 'on', got: %q", st)
	}

	// 5. Test with nil store
	nilMgr := &AccountManager{}
	if err := nilMgr.SetAccountStatus(email, "off"); err == nil {
		t.Error("expected error from SetAccountStatus when store is nil")
	}
	if st := nilMgr.GetAccountStatus(email); st != "on" {
		t.Errorf("expected 'on' when store is nil, got: %q", st)
	}

	// 6. AccountStore() accessor
	if mgr.AccountStore() != accStore {
		t.Errorf("expected AccountStore() to return accStore instance")
	}
}

func TestAccountManager_GetAccountStates_WithStatus(t *testing.T) {
	tmpDir := t.TempDir()
	accStore, err := store.NewAccountStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed creating store: %v", err)
	}

	mgr, err := NewAccountManager(accStore, nil)
	if err != nil {
		t.Fatalf("Failed creating manager: %v", err)
	}

	// Override ChromeDir to empty temp dir so no real local Chrome profiles interfere
	emptyChromeDir := t.TempDir()
	mgr.SetChromeDir(emptyChromeDir)

	// Account 1: on
	email1 := "acc1.on@example.com"
	tok1 := &model.TokenData{AccessToken: "tok1", RefreshToken: "ref1"}
	if err := accStore.SaveAccount(email1, tok1); err != nil {
		t.Fatalf("SaveAccount 1 failed: %v", err)
	}

	// Account 2: off
	email2 := "acc2.off@example.com"
	tok2 := &model.TokenData{AccessToken: "tok2", RefreshToken: "ref2"}
	if err := accStore.SaveAccount(email2, tok2); err != nil {
		t.Fatalf("SaveAccount 2 failed: %v", err)
	}
	if err := accStore.SetAccountStatus(email2, "off"); err != nil {
		t.Fatalf("SetAccountStatus 2 failed: %v", err)
	}

	// Account 3: legacy file without status field
	email3 := "acc3.legacy@example.com"
	legacyFile := filepath.Join(tmpDir, "acc3.legacy@example.com.json")
	legacyJSON := `{
		"email": "acc3.legacy@example.com",
		"saved_at": 1700000000.0,
		"token_data": {
			"access_token": "tok3",
			"refresh_token": "ref3"
		}
	}`
	if err := os.WriteFile(legacyFile, []byte(legacyJSON), 0644); err != nil {
		t.Fatalf("failed to write legacy file: %v", err)
	}

	// Mock wincred handler so CaptureCurrentSession doesn't fail with real wincred
	mgr.SetWincredHandlers(
		func() (*model.TokenData, error) {
			return nil, wincred.ErrNotFound
		},
		func(token *model.TokenData) error {
			return nil
		},
	)

	overview, err := mgr.GetAccountStates(context.Background())
	if err != nil {
		t.Fatalf("GetAccountStates failed: %v", err)
	}

	statusMap := make(map[string]string)
	for _, acc := range overview.Accounts {
		statusMap[acc.Email] = acc.Status
	}

	if statusMap[email1] != "on" {
		t.Errorf("expected account 1 status 'on', got: %q", statusMap[email1])
	}
	if statusMap[email2] != "off" {
		t.Errorf("expected account 2 status 'off', got: %q", statusMap[email2])
	}
	if statusMap[email3] != "on" {
		t.Errorf("expected legacy account 3 status 'on', got: %q", statusMap[email3])
	}
}

func TestAccountManager_RemoveAccount(t *testing.T) {
	tmpDir := t.TempDir()
	storeDir := filepath.Join(tmpDir, "accounts_store")

	accStore, err := store.NewAccountStore(storeDir)
	if err != nil {
		t.Fatalf("Failed creating store: %v", err)
	}
	discDir := accStore.GetDiscoveredStoreDir()
	discStore, err := store.NewDiscoveredStore(discDir)
	if err != nil {
		t.Fatalf("Failed creating discovered store: %v", err)
	}

	mgr, err := NewAccountManager(accStore, nil)
	if err != nil {
		t.Fatalf("Failed creating manager: %v", err)
	}
	mgr.SetDiscoveredStore(discStore)
	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return nil, errors.New("no language server")
	})

	ctx := context.Background()

	// 1. Invariant 1: Cấm xóa tài khoản Active
	activeEmail := "active@example.com"
	mgr.SetWincredHandlers(
		func() (*model.TokenData, error) {
			return &model.TokenData{
				AccessToken: "active-access",
				IDToken:     makeTestJWT(activeEmail),
			}, nil
		},
		func(token *model.TokenData) error { return nil },
	)
	if err := mgr.RemoveAccount(ctx, activeEmail); !errors.Is(err, ErrCannotDeleteActiveAccount) {
		t.Fatalf("expected ErrCannotDeleteActiveAccount, got %v", err)
	}

	// 2. Empty email error
	if err := mgr.RemoveAccount(ctx, "   "); !errors.Is(err, store.ErrInvalidEmail) {
		t.Fatalf("expected store.ErrInvalidEmail for empty email, got %v", err)
	}

	// 3. Xóa tài khoản có token trong accounts_store + có bản ghi trong discovered_accounts
	inactiveEmail := "inactive@example.com"
	tok := &model.TokenData{AccessToken: "tok", RefreshToken: "ref"}
	if err := accStore.SaveAccount(inactiveEmail, tok); err != nil {
		t.Fatalf("failed saving account: %v", err)
	}
	if err := discStore.SetDiscoveredAdded(inactiveEmail, true); err != nil {
		t.Fatalf("failed setting discovered account added: %v", err)
	}

	// Xác nhận trước khi xóa
	if !accStore.HasAccount(inactiveEmail) {
		t.Fatalf("expected account to exist before deletion")
	}
	recordsBefore, _ := discStore.ListDiscoveredRecords()
	foundBefore := false
	for _, rec := range recordsBefore {
		if strings.EqualFold(rec.Email, inactiveEmail) && rec.Added {
			foundBefore = true
			break
		}
	}
	if !foundBefore {
		t.Fatalf("expected discovered record to have Added=true before deletion")
	}

	// Thực hiện xóa
	if err := mgr.RemoveAccount(ctx, inactiveEmail); err != nil {
		t.Fatalf("RemoveAccount failed: %v", err)
	}

	// Xác nhận sau khi xóa: accounts_store không còn, discovered có Added=false
	if accStore.HasAccount(inactiveEmail) {
		t.Fatalf("expected account to be deleted from accounts_store")
	}
	recordsAfter, err := discStore.ListDiscoveredRecords()
	if err != nil {
		t.Fatalf("failed listing discovered records: %v", err)
	}
	foundAfter := false
	for _, rec := range recordsAfter {
		if strings.EqualFold(rec.Email, inactiveEmail) {
			foundAfter = true
			if rec.Added {
				t.Fatalf("expected discovered record Added to be false after deletion")
			}
			break
		}
	}
	if !foundAfter {
		t.Fatalf("expected discovered record to exist after deletion")
	}

	// 4. Xóa tài khoản [Cần OAuth] (không có trong accounts_store, chỉ có trong discovered_accounts)
	oauthEmail := "need.oauth@example.com"
	if err := discStore.SetDiscoveredAdded(oauthEmail, true); err != nil {
		t.Fatalf("failed setting discovered account added: %v", err)
	}

	if err := mgr.RemoveAccount(ctx, oauthEmail); err != nil {
		t.Fatalf("RemoveAccount for need-oauth failed: %v", err)
	}

	recordsOAuth, err := discStore.ListDiscoveredRecords()
	if err != nil {
		t.Fatalf("failed listing discovered records: %v", err)
	}
	foundOAuth := false
	for _, rec := range recordsOAuth {
		if strings.EqualFold(rec.Email, oauthEmail) {
			foundOAuth = true
			if rec.Added {
				t.Fatalf("expected oauth record Added to be false after deletion")
			}
			break
		}
	}
	if !foundOAuth {
		t.Fatalf("expected oauth discovered record to exist")
	}
}

func syncManager(t *testing.T) *AccountManager {
	t.Helper()
	st, err := store.NewAccountStore(filepath.Join(t.TempDir(), "accounts_store"))
	if err != nil {
		t.Fatal(err)
	}
	m, err := NewAccountManager(st, nil)
	if err != nil {
		t.Fatal(err)
	}
	m.SetClientFinder(func(context.Context) (*connector.Client, error) { return nil, connector.ErrLanguageServerNotFound })
	m.SetWincredHandlers(func() (*model.TokenData, error) { return nil, wincred.ErrNotFound }, nil)
	return m
}

func TestSessionIdentityFormatsAndConflict(t *testing.T) {
	for _, tc := range []struct {
		name, raw, want string
		conflict        bool
	}{
		{"current", `{"token":{"access_token":"fixture"},"id_token_email":" User@Example.com "}`, "user@example.com", false},
		{"legacy", `{"access_token":"fixture","email":"legacy@example.com"}`, "legacy@example.com", false},
		{"bad_hint_fallback", `{"access_token":"fixture","id_token_email":{},"email":"legacy@example.com"}`, "legacy@example.com", false},
		{"conflict", `{"access_token":"fixture","id_token_email":"a@example.com","email":"b@example.com"}`, "", true},
		{"missing", `{"access_token":"fixture","id_token_email":[]}`, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var token model.TokenData
			if err := json.Unmarshal([]byte(tc.raw), &token); err != nil {
				t.Fatal(err)
			}
			email, err := credentialEmail(&token)
			if email != tc.want || errors.Is(err, ErrCredentialEmailConflict) != tc.conflict {
				t.Fatalf("identity=%q err=%v", email, err)
			}
			if tc.want != "" {
				token.UpdateTokens("refreshed", "", "", time.Now().Add(time.Hour))
				b, err := json.Marshal(&token)
				if err != nil {
					t.Fatal(err)
				}
				var restored model.TokenData
				if err = json.Unmarshal(b, &restored); err != nil {
					t.Fatal(err)
				}
				if ExtractEmailFromToken(&restored) != tc.want {
					t.Fatal("identity lost after token refresh")
				}
			}
		})
	}
	token := &model.TokenData{AccessToken: "fixture", IDToken: makeTestJWT("a@example.com"), IDTokenEmail: "b@example.com"}
	if _, err := credentialEmail(token); !errors.Is(err, ErrCredentialEmailConflict) {
		t.Fatal("JWT conflict not rejected")
	}
}

func TestSessionCaptureNoRewriteAndStoreIsolation(t *testing.T) {
	m := syncManager(t)
	token := &model.TokenData{AccessToken: "fixture", IDTokenEmail: "capture@example.com"}
	m.SetWincredHandlers(func() (*model.TokenData, error) { return token, nil }, nil)
	view, err := m.GetAccountStates(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if view.ActiveEmail != "capture@example.com" || view.IdentitySource != "credential" || len(view.Warnings) != 0 {
		t.Fatalf("bad overview: %+v", view)
	}
	if err = m.SetAccountStatus(view.ActiveEmail, "off"); err != nil {
		t.Fatal(err)
	}
	path, _ := store.SafeAccountPath(m.store.GetStoreDir(), view.ActiveEmail)
	before, _ := os.ReadFile(path)
	info, _ := os.Stat(path)
	for i := 0; i < 3; i++ {
		if _, err = m.GetAccountStates(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	after, _ := os.ReadFile(path)
	later, _ := os.Stat(path)
	if string(before) != string(after) || !info.ModTime().Equal(later.ModTime()) {
		t.Fatal("unchanged capture rewrote file")
	}
	records, _ := m.discoveredStore.ListDiscoveredRecords()
	if len(records) != 0 {
		t.Fatal("capture wrote Chrome catalog")
	}
	token.AccessToken = "changed"
	if _, err = m.CaptureCurrentSession(); err != nil {
		t.Fatal(err)
	}
	saved, _ := m.store.GetAccount(view.ActiveEmail)
	if saved.AccessToken != "changed" || m.GetAccountStatus(view.ActiveEmail) != "off" {
		t.Fatal("changed token/metadata lost")
	}
	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err = m.CaptureCurrentSession(); err != nil || !m.store.HasAccount(view.ActiveEmail) {
		t.Fatal("missing file was not recaptured", err)
	}
}

func TestSessionMutationCancelsCaptureBeforeWrite(t *testing.T) {
	m := syncManager(t)
	entered, unblock := make(chan struct{}), make(chan struct{})
	m.SetWincredHandlers(func() (*model.TokenData, error) {
		close(entered)
		<-unblock
		return &model.TokenData{AccessToken: "stale", IDTokenEmail: "stale@example.com"}, nil
	}, nil)
	done := make(chan error, 1)
	go func() { _, err := m.GetAccountStates(context.Background()); done <- err }()
	<-entered
	mutated := make(chan error, 1)
	go func() { mutated <- m.AddDiscoveredAccount("catalog@example.com") }()
	deadline := time.Now().Add(time.Second)
	for !m.IsMutating() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !m.IsMutating() {
		t.Fatal("mutation did not register")
	}
	close(unblock)
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("stale read not canceled: %v", err)
	}
	if err := <-mutated; err != nil {
		t.Fatal(err)
	}
	if m.store.HasAccount("stale@example.com") {
		t.Fatal("stale credential persisted after mutation requested")
	}
}

func TestSessionPartialStorePreservesAccountsThenHonorsDeletion(t *testing.T) {
	m := syncManager(t)
	if err := m.store.SaveAccount("saved@example.com", &model.TokenData{AccessToken: "fixture"}); err != nil {
		t.Fatal(err)
	}
	if err := m.AddDiscoveredAccount("catalog@example.com"); err != nil {
		t.Fatal(err)
	}
	initial, err := m.GetAccountStates(context.Background())
	if err != nil || len(initial.Accounts) != 2 {
		t.Fatal("initial", err)
	}
	path, _ := store.SafeAccountPath(m.store.GetStoreDir(), "saved@example.com")
	if err = os.WriteFile(path, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	partial, err := m.GetAccountStates(context.Background())
	if err != nil || len(partial.Accounts) != 2 || len(partial.Warnings) == 0 {
		t.Fatal("partial read dropped prior account", err)
	}
	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	deleted, err := m.GetAccountStates(context.Background())
	if err != nil || len(deleted.Accounts) != 1 || deleted.Accounts[0].HasToken {
		t.Fatal("complete deletion retained stale credential", err)
	}
}

func TestSessionCanceledWaitDoesNotAcquireTransaction(t *testing.T) {
	m := syncManager(t)
	release, err := m.session.mutate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err = m.GetAccountStates(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("wait ignored deadline: %v", err)
	}
}

// Opt-in real integration reads the user's credential and live IDE but stores
// only in t.TempDir. It never writes WinCred, switches accounts or prints secrets.
func TestSessionLiveReadCapture(t *testing.T) {
	if os.Getenv("ANTIGRAVITY_SESSION_LIVE_QA") != "1" {
		t.Skip("opt-in live read")
	}
	st, err := store.NewAccountStore(filepath.Join(t.TempDir(), "accounts_store"))
	if err != nil {
		t.Fatal(err)
	}
	m, err := NewAccountManager(st, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	view, err := m.GetAccountStates(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if view.ActiveEmail == "" || view.IdentitySource != "live" || !view.LanguageServerRunning {
		t.Fatal("live IDE identity not resolved")
	}
	if len(view.Warnings) != 0 {
		t.Fatalf("warning codes: %v", view.WarningCodes)
	}
	token, err := wincred.ReadToken()
	if err != nil {
		t.Fatal("credential unavailable")
	}
	captured := ExtractEmailFromToken(token)
	if captured == "" || !st.HasAccount(captured) {
		t.Fatal("credential not captured")
	}
	path, _ := store.SafeAccountPath(st.GetStoreDir(), captured)
	before, _ := os.ReadFile(path)
	if _, err = m.GetAccountStates(ctx); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("unchanged live credential rewritten")
	}
	records, err := m.DiscoveredStore().ListDiscoveredRecords()
	if err != nil || len(records) != 0 {
		t.Fatal("live capture touched catalog")
	}
	t.Logf("live_identity=true credential_captured=true id_token_email_present=%t unchanged_capture_no_write=true catalog_untouched=true", token.IDTokenEmail != "")
}

func TestSessionDeletedRecordNotRestoredByOtherCorruption(t *testing.T) {
	m := syncManager(t)
	if err := m.store.SaveAccount("gone@example.com", &model.TokenData{AccessToken: "fixture"}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.GetAccountStates(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := m.store.DeleteAccount("gone@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(m.store.GetStoreDir(), "bad.json"), []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	view, err := m.GetAccountStates(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Accounts) != 0 || len(view.Warnings) == 0 {
		t.Fatal("partial read resurrected a confirmed deletion")
	}
}
