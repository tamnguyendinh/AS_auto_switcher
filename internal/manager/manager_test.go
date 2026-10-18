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

	"antigravity-switch/internal/connector"
	"antigravity-switch/internal/model"
	"antigravity-switch/internal/reloader"
	"antigravity-switch/internal/store"

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

	// Prepare mock active threads file for Step 1 -> Step 3
	historyDir := filepath.Join(tmpDir, "active_threads_history")
	_ = os.MkdirAll(historyDir, 0755)
	convDir := filepath.Join(tmpDir, "conversations")
	_ = os.MkdirAll(convDir, 0755)

	now := time.Now()
	f1 := filepath.Join(convDir, "thread-step4-1.db")
	_ = os.WriteFile(f1, []byte("data"), 0644)
	_ = os.Chtimes(f1, now.Add(-10*time.Minute), now.Add(-10*time.Minute))

	f2 := filepath.Join(convDir, "thread-step4-2.db")
	_ = os.WriteFile(f2, []byte("data"), 0644)
	_ = os.Chtimes(f2, now.Add(-20*time.Minute), now.Add(-20*time.Minute))

	// Track 8-second sleep duration
	var eightSecSleepCalled atomic.Bool
	var totalSleeps atomic.Int32

	var killed atomic.Bool
	// Track HTTP RPC calls on new instance
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
	rel.SnapshotCapturer = func(storagePath, brainDir, outputDir string) (string, []reloader.ActiveThread, error) { return reloader.CaptureActiveThreadsSnapshot(convDir, "", historyDir) }
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
