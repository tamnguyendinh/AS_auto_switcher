package manager

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"antigravity-switch/internal/engines/antigravity/connector"
	"antigravity-switch/internal/engines/antigravity/model"
	"antigravity-switch/internal/engines/antigravity/reloader"
	"antigravity-switch/internal/engines/antigravity/store"

	"golang.org/x/sys/windows"
)

func TestSwitchAccount_8SecondWaitAnd4StepWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	accStore, err := store.NewAccountStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed creating account store: %v", err)
	}

	targetEmail := "target-8sec@example.com"
	currentEmail := "current-before@example.com"

	// Prepare target account in store
	targetTok := &model.TokenData{
		AccessToken: "target-access-token-123",
		IDToken:     makeTestJWT(targetEmail),
		Expiry:      time.Now().Add(1 * time.Hour),
	}
	if err := accStore.SaveAccount(targetEmail, targetTok); err != nil {
		t.Fatalf("Failed saving target account: %v", err)
	}

	// Prepare mock active threads with valid UUIDs (F3)
	historyDir := filepath.Join(tmpDir, "active_threads_history")
	_ = os.MkdirAll(historyDir, 0755)
	convDir := filepath.Join(tmpDir, "conversations")
	_ = os.MkdirAll(convDir, 0755)

	now := time.Now()
	f1 := filepath.Join(convDir, "11111111-1111-4111-8111-111111111111.db")
	_ = os.WriteFile(f1, []byte("data"), 0644)
	_ = os.Chtimes(f1, now.Add(-1*time.Minute), now.Add(-1*time.Minute))

	f2 := filepath.Join(convDir, "22222222-2222-4222-8222-222222222222.db")
	_ = os.WriteFile(f2, []byte("data"), 0644)
	_ = os.Chtimes(f2, now.Add(-2*time.Minute), now.Add(-2*time.Minute))

	var killed atomic.Bool
	var eightSecSleepCalled atomic.Bool
	var totalSleeps atomic.Int32

	var focusCalls atomic.Int32
	var loadCalls atomic.Int32
	var sendCalls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case connector.EndpointSmartFocusConversation:
			focusCalls.Add(1)
			w.WriteHeader(http.StatusOK)
		case connector.EndpointLoadTrajectory:
			loadCalls.Add(1)
			w.WriteHeader(http.StatusOK)
		case connector.EndpointSendUserCascadeMessage:
			sendCalls.Add(1)
			w.WriteHeader(http.StatusOK)
		case connector.EndpointGetUserStatus:
			w.WriteHeader(http.StatusOK)
			active := currentEmail
			if killed.Load() {
				active = targetEmail
			}
			_, _ = w.Write([]byte(`{"email":"` + active + `"}`))
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	parts := strings.Split(server.URL, ":")
	port, _ := strconv.Atoi(parts[len(parts)-1])

	rel := reloader.NewSessionReloader()
	rel.ConversationsDir = convDir
	rel.OutputDir = historyDir
	rel.WindowFinder = func() ([]windows.HWND, error) {
		return []windows.HWND{windows.HWND(101)}, nil
	}
	rel.ProcessKiller = func(ctx context.Context) error {
		killed.Store(true)
		return nil
	}
	rel.ServerDiscoverer = func() (*connector.ServerInfo, error) {
		return &connector.ServerInfo{PID: 5555, Port: port, CSRFToken: "csrf-token"}, nil
	}
	rel.Sleeper = func(ctx context.Context, d time.Duration) error {
		totalSleeps.Add(1)
		if d == 8*time.Second {
			eightSecSleepCalled.Store(true)
		}
		return nil
	}

	mgr, err := NewAccountManager(accStore, rel)
	if err != nil {
		t.Fatalf("Failed creating manager: %v", err)
	}

	var writtenToken *model.TokenData
	mgr.SetWincredHandlers(
		func() (*model.TokenData, error) {
			return &model.TokenData{
				AccessToken: "current-token",
				IDToken:     makeTestJWT(currentEmail),
			}, nil
		},
		func(token *model.TokenData) error {
			writtenToken = token
			return nil
		},
	)

	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return connector.NewClientWithHTTP(&connector.ServerInfo{
			PID:       5555,
			Port:      port,
			CSRFToken: "csrf-token",
		}, server.Client()), nil
	})

	result, err := mgr.SwitchAccount(context.Background(), targetEmail, false)
	if err != nil {
		t.Fatalf("SwitchAccount failed: %v", err)
	}

	if !result.Success {
		t.Fatalf("Expected result.Success = true, got false: %s", result.Message)
	}

	// Verify Step 2: language_server was recycled with taskkill
	if !killed.Load() {
		t.Errorf("Expected language server to be killed via ProcessKiller")
	}

	// Verify Step 2: Sleeper called with 8 seconds
	if !eightSecSleepCalled.Load() {
		t.Errorf("Expected 8-second sleep to be called during SwitchAccount")
	}

	// Verify Step 2: Token written to wincred
	if writtenToken == nil || writtenToken.AccessToken != "target-access-token-123" {
		t.Errorf("Expected target token written to wincred, got: %+v", writtenToken)
	}

	// Verify Step 4: Continuation occurred on the 2 threads
	if result.ContinuationSummary == nil {
		t.Fatalf("Expected non-nil ContinuationSummary")
	}
	if result.ContinuationSummary.TotalThreads != 2 {
		t.Errorf("Expected TotalThreads == 2, got %d", result.ContinuationSummary.TotalThreads)
	}
	if result.ContinuationSummary.SuccessCount != 2 {
		t.Errorf("Expected SuccessCount == 2, got %d", result.ContinuationSummary.SuccessCount)
	}

	// Message should contain transparent thread count
	expectedSub := "Đã gửi prompt tiếp tục phiên vào 2/2 thread."
	if !strings.Contains(result.Message, expectedSub) {
		t.Errorf("Expected message to contain %q, got: %s", expectedSub, result.Message)
	}
}

// TestSwitchAccount_F1_RAMConfirmation_Mismatch verifies that if RAM keeps the stale account,
// SwitchAccount reports failure and NEVER executes session continuation (F1).
func TestSwitchAccount_F1_RAMConfirmation_Mismatch(t *testing.T) {
	tmpDir := t.TempDir()
	accStore, _ := store.NewAccountStore(tmpDir)

	targetEmail := "target-ram-mismatch@example.com"
	currentEmail := "current-before@example.com"

	_ = accStore.SaveAccount(targetEmail, &model.TokenData{
		AccessToken: "target-tok",
		IDToken:     makeTestJWT(targetEmail),
		Expiry:      time.Now().Add(1 * time.Hour),
	})

	convDir := filepath.Join(tmpDir, "conversations")
	_ = os.MkdirAll(convDir, 0755)
	f1 := filepath.Join(convDir, "11111111-2222-4333-8444-555555555555.db")
	_ = os.WriteFile(f1, []byte("data"), 0644)

	var continuationCalled atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case connector.EndpointGetUserStatus:
			w.WriteHeader(http.StatusOK)
			// Language Server stale: still returns old email even after kill
			_, _ = w.Write([]byte(`{"email":"stale-account@example.com"}`))
		case connector.EndpointSendUserCascadeMessage:
			continuationCalled.Store(true)
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	parts := strings.Split(server.URL, ":")
	port, _ := strconv.Atoi(parts[len(parts)-1])

	rel := reloader.NewSessionReloader()
	rel.ConversationsDir = convDir
	rel.OutputDir = filepath.Join(tmpDir, "history")
	rel.WindowFinder = func() ([]windows.HWND, error) { return []windows.HWND{windows.HWND(101)}, nil }
	rel.ProcessKiller = func(ctx context.Context) error { return nil }
	rel.ServerDiscoverer = func() (*connector.ServerInfo, error) {
		return &connector.ServerInfo{PID: 1234, Port: port, CSRFToken: "tok"}, nil
	}
	rel.Sleeper = func(ctx context.Context, d time.Duration) error { return nil }

	mgr, _ := NewAccountManager(accStore, rel)
	mgr.SetWincredHandlers(
		func() (*model.TokenData, error) { return &model.TokenData{IDToken: makeTestJWT(currentEmail)}, nil },
		func(token *model.TokenData) error { return nil },
	)
	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return connector.NewClientWithHTTP(&connector.ServerInfo{PID: 1234, Port: port, CSRFToken: "tok"}, server.Client()), nil
	})

	result, err := mgr.SwitchAccount(context.Background(), targetEmail, false)
	if err == nil {
		t.Fatalf("Expected error due to RAM identity mismatch, got nil")
	}
	if result == nil || result.Success {
		t.Fatalf("Expected result.Success = false on RAM mismatch, got %+v", result)
	}
	if continuationCalled.Load() {
		t.Fatalf("CRITICAL SECURITY VIOLATION: Continuation was dispatched on the wrong RAM identity!")
	}
	if !strings.Contains(result.Message, "Xác thực RAM thất bại") {
		t.Errorf("Expected RAM failure message, got: %s", result.Message)
	}
}

// TestSwitchAccount_F1_RAMConfirmation_GetUserStatusError verifies failure if GetUserStatus returns HTTP error.
func TestSwitchAccount_F1_RAMConfirmation_GetUserStatusError(t *testing.T) {
	tmpDir := t.TempDir()
	accStore, _ := store.NewAccountStore(tmpDir)

	targetEmail := "target-err@example.com"
	currentEmail := "current-before@example.com"

	_ = accStore.SaveAccount(targetEmail, &model.TokenData{
		AccessToken: "target-tok",
		IDToken:     makeTestJWT(targetEmail),
		Expiry:      time.Now().Add(1 * time.Hour),
	})

	var continuationCalled atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == connector.EndpointGetUserStatus {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if r.URL.Path == connector.EndpointSendUserCascadeMessage {
			continuationCalled.Store(true)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	parts := strings.Split(server.URL, ":")
	port, _ := strconv.Atoi(parts[len(parts)-1])

	rel := reloader.NewSessionReloader()
	rel.ConversationsDir = tmpDir
	rel.OutputDir = filepath.Join(tmpDir, "history")
	rel.WindowFinder = func() ([]windows.HWND, error) { return []windows.HWND{windows.HWND(101)}, nil }
	rel.ProcessKiller = func(ctx context.Context) error { return nil }
	rel.ServerDiscoverer = func() (*connector.ServerInfo, error) {
		return &connector.ServerInfo{PID: 1234, Port: port, CSRFToken: "tok"}, nil
	}
	rel.Sleeper = func(ctx context.Context, d time.Duration) error { return nil }

	mgr, _ := NewAccountManager(accStore, rel)
	mgr.SetWincredHandlers(
		func() (*model.TokenData, error) { return &model.TokenData{IDToken: makeTestJWT(currentEmail)}, nil },
		func(token *model.TokenData) error { return nil },
	)
	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return connector.NewClientWithHTTP(&connector.ServerInfo{PID: 1234, Port: port, CSRFToken: "tok"}, server.Client()), nil
	})

	result, err := mgr.SwitchAccount(context.Background(), targetEmail, false)
	if err == nil {
		t.Fatalf("Expected error when GetUserStatus fails, got nil")
	}
	if result == nil || result.Success {
		t.Fatalf("Expected result.Success = false when GetUserStatus fails")
	}
	if continuationCalled.Load() {
		t.Fatalf("Continuation must not be called when GetUserStatus fails")
	}
}

// TestSwitchAccount_F1_RAMConfirmation_ClientNil verifies failure when respawned language server cannot be found.
func TestSwitchAccount_F1_RAMConfirmation_ClientNil(t *testing.T) {
	tmpDir := t.TempDir()
	accStore, _ := store.NewAccountStore(tmpDir)

	targetEmail := "target-nil@example.com"
	currentEmail := "current-before@example.com"

	_ = accStore.SaveAccount(targetEmail, &model.TokenData{
		AccessToken: "target-tok",
		IDToken:     makeTestJWT(targetEmail),
		Expiry:      time.Now().Add(1 * time.Hour),
	})

	rel := reloader.NewSessionReloader()
	rel.ConversationsDir = tmpDir
	rel.OutputDir = filepath.Join(tmpDir, "history")
	rel.WindowFinder = func() ([]windows.HWND, error) { return []windows.HWND{windows.HWND(101)}, nil }
	rel.ProcessKiller = func(ctx context.Context) error { return nil }
	rel.ServerDiscoverer = func() (*connector.ServerInfo, error) {
		return &connector.ServerInfo{PID: 1234, Port: 54321, CSRFToken: "tok"}, nil
	}
	rel.Sleeper = func(ctx context.Context, d time.Duration) error { return nil }

	mgr, _ := NewAccountManager(accStore, rel)
	mgr.SetWincredHandlers(
		func() (*model.TokenData, error) { return &model.TokenData{IDToken: makeTestJWT(currentEmail)}, nil },
		func(token *model.TokenData) error { return nil },
	)
	// Finder returns nil
	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return nil, connector.ErrLanguageServerNotFound
	})

	result, err := mgr.SwitchAccount(context.Background(), targetEmail, false)
	if err == nil {
		t.Fatalf("Expected error when newClient is nil, got nil")
	}
	if result == nil || result.Success {
		t.Fatalf("Expected result.Success = false when newClient is nil")
	}
}

// TestSwitchAccount_F4_StoragePathIntegration verifies that configuring rel.StoragePath for ToS
// does NOT corrupt the conversations directory or crash snapshot capture (F4).
func TestSwitchAccount_F4_StoragePathIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	accStore, _ := store.NewAccountStore(tmpDir)

	targetEmail := "target-tos-integration@example.com"
	currentEmail := "current-before@example.com"

	_ = accStore.SaveAccount(targetEmail, &model.TokenData{
		AccessToken: "target-tok",
		IDToken:     makeTestJWT(targetEmail),
		Expiry:      time.Now().Add(1 * time.Hour),
	})

	convDir := filepath.Join(tmpDir, "conversations")
	_ = os.MkdirAll(convDir, 0755)
	historyDir := filepath.Join(tmpDir, "active_threads_history")
	_ = os.MkdirAll(historyDir, 0755)

	f1 := filepath.Join(convDir, "77777777-7777-4777-8777-777777777777.db")
	_ = os.WriteFile(f1, []byte("data"), 0644)

	storageFile := filepath.Join(tmpDir, "app_storage.json")
	_ = os.WriteFile(storageFile, []byte(`{"test":true}`), 0644)

	var killed atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case connector.EndpointGetUserStatus:
			w.WriteHeader(http.StatusOK)
			active := currentEmail
			if killed.Load() {
				active = targetEmail
			}
			_, _ = w.Write([]byte(`{"email":"` + active + `"}`))
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	parts := strings.Split(server.URL, ":")
	port, _ := strconv.Atoi(parts[len(parts)-1])

	rel := reloader.NewSessionReloader()
	// F4: Setting StoragePath must not break snapshot
	rel.StoragePath = storageFile
	rel.ConversationsDir = convDir
	rel.OutputDir = historyDir
	rel.WindowFinder = func() ([]windows.HWND, error) { return []windows.HWND{windows.HWND(101)}, nil }
	rel.ProcessKiller = func(ctx context.Context) error {
		killed.Store(true)
		return nil
	}
	rel.ServerDiscoverer = func() (*connector.ServerInfo, error) {
		return &connector.ServerInfo{PID: 1234, Port: port, CSRFToken: "tok"}, nil
	}
	rel.Sleeper = func(ctx context.Context, d time.Duration) error { return nil }

	mgr, _ := NewAccountManager(accStore, rel)
	mgr.SetWincredHandlers(
		func() (*model.TokenData, error) { return &model.TokenData{IDToken: makeTestJWT(currentEmail)}, nil },
		func(token *model.TokenData) error { return nil },
	)
	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return connector.NewClientWithHTTP(&connector.ServerInfo{PID: 1234, Port: port, CSRFToken: "tok"}, server.Client()), nil
	})

	result, err := mgr.SwitchAccount(context.Background(), targetEmail, false)
	if err != nil {
		t.Fatalf("SwitchAccount failed with StoragePath configured: %v", err)
	}
	if !result.Success {
		t.Fatalf("Expected success, got false: %s", result.Message)
	}
	if len(result.ActiveThreads) != 1 || result.ActiveThreads[0].ThreadID != "77777777-7777-4777-8777-777777777777" {
		t.Fatalf("Expected 1 active thread captured, got %+v", result.ActiveThreads)
	}
}

// TestSwitchAccount_F1_RAMConfirmation_ClientNilWithoutError verifies that when clientFinder
// returns (nil, nil) even with target credentials written, SwitchAccount strictly returns Success=false and an error (R2-F1).
func TestSwitchAccount_F1_RAMConfirmation_ClientNilWithoutError(t *testing.T) {
	tmpDir := t.TempDir()
	accStore, _ := store.NewAccountStore(tmpDir)

	targetEmail := "target-nil-without-error@example.com"
	currentEmail := "current-before@example.com"

	_ = accStore.SaveAccount(targetEmail, &model.TokenData{
		AccessToken: "target-tok",
		IDToken:     makeTestJWT(targetEmail),
		Expiry:      time.Now().Add(1 * time.Hour),
	})

	rel := reloader.NewSessionReloader()
	rel.ConversationsDir = tmpDir
	rel.OutputDir = filepath.Join(tmpDir, "history")
	rel.WindowFinder = func() ([]windows.HWND, error) { return []windows.HWND{windows.HWND(101)}, nil }
	rel.ProcessKiller = func(ctx context.Context) error { return nil }
	rel.ServerDiscoverer = func() (*connector.ServerInfo, error) {
		return &connector.ServerInfo{PID: 9999, Port: 23456, CSRFToken: "tok"}, nil
	}
	rel.Sleeper = func(ctx context.Context, d time.Duration) error { return nil }

	mgr, _ := NewAccountManager(accStore, rel)
	var activeToken *model.TokenData = &model.TokenData{IDToken: makeTestJWT(currentEmail)}
	mgr.SetWincredHandlers(
		func() (*model.TokenData, error) { return activeToken, nil },
		func(token *model.TokenData) error {
			activeToken = token
			return nil
		},
	)

	// clientFinder returns (nil, nil)
	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return nil, nil
	})

	result, err := mgr.SwitchAccount(context.Background(), targetEmail, false)
	if err == nil {
		t.Fatalf("Expected error when newClient is nil (even without error), got nil")
	}
	if result == nil || result.Success {
		t.Fatalf("Expected result.Success = false when newClient is nil, got %+v", result)
	}
}

// TestSwitchAccount_PrunesHistoryToMax2 verifies that SwitchAccount automatically prunes
// active_threads_history to keep at most 2 snapshots upon successful account switch (SPEC-ARCH-20261021-ACTIVE-THREADS-HISTORY-PRUNING).
func TestSwitchAccount_PrunesHistoryToMax2(t *testing.T) {
	tmpDir := t.TempDir()
	accStore, err := store.NewAccountStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed creating account store: %v", err)
	}

	targetEmail := "switch-prune-test@example.com"
	currentEmail := "current-prune-test@example.com"
	mockActiveEmail := currentEmail

	_ = accStore.SaveAccount(targetEmail, &model.TokenData{
		AccessToken: "target-tok",
		IDToken:     makeTestJWT(targetEmail),
		Expiry:      time.Now().Add(1 * time.Hour),
	})

	historyDir := filepath.Join(tmpDir, "active_threads_history")
	_ = os.MkdirAll(historyDir, 0755)

	// Pre-create 5 older snapshot files in active_threads_history
	for i := 1; i <= 5; i++ {
		name := filepath.Join(historyDir, "2026-10-20-0"+string(rune('0'+i))+"-00-00_active_threads.json")
		if err := os.WriteFile(name, []byte(`{"test":true}`), 0644); err != nil {
			t.Fatalf("Failed writing initial snapshot file: %v", err)
		}
	}

	rel := reloader.NewSessionReloader()
	rel.OutputDir = historyDir
	rel.ConversationsDir = tmpDir
	rel.WindowFinder = func() ([]windows.HWND, error) { return []windows.HWND{windows.HWND(101)}, nil }
	rel.ProcessKiller = func(ctx context.Context) error { return nil }
	rel.Sleeper = func(ctx context.Context, d time.Duration) error { return nil }
	rel.SnapshotCapturer = func(storagePath, brainDir, outputDir string) (string, []reloader.ActiveThread, error) {
		snapPath := filepath.Join(historyDir, "2026-10-21-12-00-00_active_threads.json")
		_ = os.WriteFile(snapPath, []byte(`{"active_threads":[]}`), 0644)
		return snapPath, []reloader.ActiveThread{}, nil
	}

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
		return &connector.ServerInfo{PID: 8888, Port: port, CSRFToken: "test-csrf-token"}, nil
	}

	mgr, _ := NewAccountManager(accStore, rel)
	mgr.SetWincredHandlers(
		func() (*model.TokenData, error) {
			return &model.TokenData{
				AccessToken: "current-tok",
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
	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return client, nil
	})

	result, err := mgr.SwitchAccount(context.Background(), targetEmail, false)
	if err != nil {
		t.Fatalf("SwitchAccount failed: %v", err)
	}
	if !result.Success {
		t.Fatalf("Expected SwitchAccount to succeed, got: %s", result.Message)
	}

	// Verify pruning in active_threads_history
	entries, err := os.ReadDir(historyDir)
	if err != nil {
		t.Fatalf("Failed reading historyDir: %v", err)
	}

	var remainingSnapshots []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), "_active_threads.json") {
			remainingSnapshots = append(remainingSnapshots, e.Name())
		}
	}

	if len(remainingSnapshots) > 2 {
		t.Fatalf("Expected at most 2 snapshots in active_threads_history, found %d: %v", len(remainingSnapshots), remainingSnapshots)
	}
	if len(remainingSnapshots) != 2 {
		t.Errorf("Expected exactly 2 snapshots retained after pruning, found %d: %v", len(remainingSnapshots), remainingSnapshots)
	}
}

// TestSwitchAccount_UsesLogsActiveThreads verifies that SwitchAccount uses logs/active_threads_history
// and leaves accounts_store 100% pure containing only credential files (SPEC-ARCH-20261021-LOGS-AND-DAILY-QUOTA-SPEC Phase 6).
func TestSwitchAccount_UsesLogsActiveThreads(t *testing.T) {
	root := t.TempDir()
	storeDir := filepath.Join(root, "accounts_store")
	accStore, err := store.NewAccountStore(storeDir)
	if err != nil {
		t.Fatalf("Failed creating account store: %v", err)
	}

	targetEmail := "switch-logs-test@example.com"
	currentEmail := "current-logs-test@example.com"
	mockActiveEmail := currentEmail

	_ = accStore.SaveAccount(targetEmail, &model.TokenData{
		AccessToken: "target-tok",
		IDToken:     makeTestJWT(targetEmail),
		Expiry:      time.Now().Add(1 * time.Hour),
	})

	rel := reloader.NewSessionReloader()
	// Do NOT set rel.OutputDir, let NewAccountManager set it to logs/active_threads_history
	rel.ConversationsDir = root
	rel.WindowFinder = func() ([]windows.HWND, error) { return []windows.HWND{windows.HWND(101)}, nil }
	rel.ProcessKiller = func(ctx context.Context) error { return nil }
	rel.Sleeper = func(ctx context.Context, d time.Duration) error { return nil }
	rel.SnapshotCapturer = func(storagePath, brainDir, outputDir string) (string, []reloader.ActiveThread, error) {
		_ = os.MkdirAll(outputDir, 0755)
		snapPath := filepath.Join(outputDir, "2026-10-21-12-00-00_active_threads.json")
		_ = os.WriteFile(snapPath, []byte(`{"active_threads":[]}`), 0644)
		return snapPath, []reloader.ActiveThread{}, nil
	}

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
		return &connector.ServerInfo{PID: 8888, Port: port, CSRFToken: "test-csrf-token"}, nil
	}

	mgr, err := NewAccountManager(accStore, rel)
	if err != nil {
		t.Fatalf("Failed creating AccountManager: %v", err)
	}

	expectedLogsHistoryDir := filepath.Join(root, "logs", "active_threads_history")
	if mgr.reloader.OutputDir != expectedLogsHistoryDir {
		t.Fatalf("Expected reloader.OutputDir to be %s, got %s", expectedLogsHistoryDir, mgr.reloader.OutputDir)
	}

	mgr.SetWincredHandlers(
		func() (*model.TokenData, error) {
			return &model.TokenData{
				AccessToken: "current-tok",
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
	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return client, nil
	})

	result, err := mgr.SwitchAccount(context.Background(), targetEmail, false)
	if err != nil {
		t.Fatalf("SwitchAccount failed: %v", err)
	}
	if !result.Success {
		t.Fatalf("Expected SwitchAccount to succeed, got: %s", result.Message)
	}

	// 1. Verify that logs/active_threads_history contains the snapshot
	logsHistoryEntries, err := os.ReadDir(expectedLogsHistoryDir)
	if err != nil {
		t.Fatalf("Failed reading expectedLogsHistoryDir: %v", err)
	}
	if len(logsHistoryEntries) == 0 {
		t.Fatalf("Expected at least 1 snapshot in %s, got 0", expectedLogsHistoryDir)
	}

	// 2. Verify that accounts_store does NOT have active_threads_history
	badStoreHistory := filepath.Join(storeDir, "active_threads_history")
	if _, err := os.Stat(badStoreHistory); !os.IsNotExist(err) {
		t.Fatalf("Forbidden: %s exists in accounts_store; accounts_store must be 100%% pure!", badStoreHistory)
	}

	// 3. Verify accounts_store contains only .json credential files
	storeEntries, err := os.ReadDir(storeDir)
	if err != nil {
		t.Fatalf("Failed reading storeDir: %v", err)
	}
	for _, entry := range storeEntries {
		if entry.IsDir() {
			t.Fatalf("accounts_store contains unexpected directory: %s", entry.Name())
		}
		if !strings.HasSuffix(entry.Name(), ".json") {
			t.Fatalf("accounts_store contains non-json file: %s", entry.Name())
		}
	}
}
