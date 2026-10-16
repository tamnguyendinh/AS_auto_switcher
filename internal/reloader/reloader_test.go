package reloader

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"antigravity-switch/internal/connector"

)

func TestFindAntigravityPIDs(t *testing.T) {
	pids, err := FindAntigravityPIDs()
	if err != nil {
		t.Fatalf("FindAntigravityPIDs failed: %v", err)
	}
	t.Logf("Discovered Antigravity PIDs: %v", pids)
}

func TestFindAntigravityWindows(t *testing.T) {
	hwnds, err := FindAntigravityWindows()
	if err != nil {
		t.Fatalf("FindAntigravityWindows failed: %v", err)
	}
	t.Logf("Discovered Antigravity HWNDs: %d", len(hwnds))
}

func TestEnsureAntigravityForeground(t *testing.T) {
	// Should execute safely without panic, returning ErrNoWindowsFound if no window exists
	err := EnsureAntigravityForeground()
	if err != nil && !errors.Is(err, ErrNoWindowsFound) {
		t.Logf("EnsureAntigravityForeground returned: %v", err)
	}
}

func TestPrePatchToSAcceptance(t *testing.T) {
	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "app_storage.json")

	// 1. Initial write to non-existent file
	targetEmail := "developer@example.com"
	if err := PrePatchToSAcceptance(targetEmail, storagePath); err != nil {
		t.Fatalf("PrePatchToSAcceptance failed on new file: %v", err)
	}

	data, err := os.ReadFile(storagePath)
	if err != nil {
		t.Fatalf("Failed reading patched file: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed unmarshaling patched json: %v", err)
	}

	if parsed["jetski.onboarding.lastLoginUsername"] != targetEmail {
		t.Errorf("Expected username %s, got %v", targetEmail, parsed["jetski.onboarding.lastLoginUsername"])
	}
	if parsed["jetski.onboarding.lastLoginIsGcpTos"] != "false" {
		t.Errorf("Expected isGcpTos 'false', got %v", parsed["jetski.onboarding.lastLoginIsGcpTos"])
	}

	// 2. Overwrite / update existing file
	newEmail := "target2@example.com"
	if err := PrePatchToSAcceptance(newEmail, storagePath); err != nil {
		t.Fatalf("PrePatchToSAcceptance failed on existing file: %v", err)
	}

	data2, _ := os.ReadFile(storagePath)
	var parsed2 map[string]interface{}
	_ = json.Unmarshal(data2, &parsed2)
	if parsed2["jetski.onboarding.lastLoginUsername"] != newEmail {
		t.Errorf("Expected updated username %s, got %v", newEmail, parsed2["jetski.onboarding.lastLoginUsername"])
	}

	// 3. Error on empty email
	if err := PrePatchToSAcceptance("", storagePath); err == nil {
		t.Error("Expected error for empty email, got nil")
	}
}

func TestPrepatchToS(t *testing.T) {
	// PrepatchToS should reject empty email
	if err := PrepatchToS(""); err == nil {
		t.Error("Expected error for empty email, got nil")
	}
}

func TestRecycleLanguageServer_Success(t *testing.T) {
	rel := NewSessionReloader()

	var killed atomic.Bool
	var sleptDuration time.Duration
	var discovered atomic.Bool

	rel.ProcessKiller = func(ctx context.Context) error {
		killed.Store(true)
		return nil
	}

	rel.Sleeper = func(ctx context.Context, d time.Duration) error {
		sleptDuration = d
		return nil
	}

	rel.ServerDiscoverer = func() (*connector.ServerInfo, error) {
		discovered.Store(true)
		return &connector.ServerInfo{
			PID:       12345,
			Port:      54321,
			CSRFToken: "new-csrf-token",
		}, nil
	}

	ctx := context.Background()
	err := rel.RecycleLanguageServer(ctx, 2*time.Second)
	if err != nil {
		t.Fatalf("RecycleLanguageServer failed: %v", err)
	}

	if !killed.Load() {
		t.Error("Expected ProcessKiller to be invoked")
	}
	if sleptDuration != 2*time.Second {
		t.Errorf("Expected sleptDuration 2s, got %v", sleptDuration)
	}
	if !discovered.Load() {
		t.Error("Expected ServerDiscoverer to be invoked")
	}
}

func TestRecycleLanguageServer_DiscovererFails(t *testing.T) {
	rel := NewSessionReloader()

	rel.ProcessKiller = func(ctx context.Context) error {
		return nil
	}

	rel.Sleeper = func(ctx context.Context, d time.Duration) error {
		return nil
	}

	rel.ServerDiscoverer = func() (*connector.ServerInfo, error) {
		return nil, connector.ErrLanguageServerNotFound
	}

	// Use short deadline context so probe loop completes quickly in test
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := rel.RecycleLanguageServer(ctx, 10*time.Millisecond)
	if err == nil {
		t.Fatal("Expected error when ServerDiscoverer fails, got nil")
	}
	if !errors.Is(err, ErrLanguageServerRestart) && !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected ErrLanguageServerRestart or context deadline error, got: %v", err)
	}
}

func TestRecycleLanguageServer_ContextCancelled(t *testing.T) {
	rel := NewSessionReloader()

	rel.ProcessKiller = func(ctx context.Context) error {
		return nil
	}

	rel.Sleeper = func(ctx context.Context, d time.Duration) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := rel.RecycleLanguageServer(ctx, 2*time.Second)
	if err == nil {
		t.Fatal("Expected error on cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled, got: %v", err)
	}
}

func TestMain(m *testing.M) {
	// Global test safety barrier: neutralize osTaskKill by default across all tests
	// in this package so that unmocked invocations never spawn real taskkill.exe processes.
	osTaskKill = func(ctx context.Context) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		return nil
	}
	os.Exit(m.Run())
}

func TestDefaultKillLanguageServer(t *testing.T) {
	origTaskKill := osTaskKill
	t.Cleanup(func() {
		osTaskKill = origTaskKill
	})

	var callCount atomic.Int32
	var receivedCtx context.Context

	osTaskKill = func(ctx context.Context) error {
		callCount.Add(1)
		receivedCtx = ctx
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// DefaultKillLanguageServer must delegate to osTaskKill without error and pass ctx.
	err := DefaultKillLanguageServer(ctx)
	if err != nil {
		t.Fatalf("DefaultKillLanguageServer failed: %v", err)
	}

	if callCount.Load() != 1 {
		t.Fatalf("Expected osTaskKill to be called exactly once, got %d", callCount.Load())
	}
	if receivedCtx != ctx {
		t.Errorf("Expected context %v passed to osTaskKill, got %v", ctx, receivedCtx)
	}
}

func TestDefaultKillLanguageServer_ContextCancelled(t *testing.T) {
	origTaskKill := osTaskKill
	t.Cleanup(func() {
		osTaskKill = origTaskKill
	})

	var callCount atomic.Int32
	osTaskKill = func(ctx context.Context) error {
		callCount.Add(1)
		if err := ctx.Err(); err != nil {
			return err
		}
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel before execution

	err := DefaultKillLanguageServer(ctx)
	if err == nil {
		t.Fatal("Expected error on cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled, got: %v", err)
	}
	if callCount.Load() != 1 {
		t.Errorf("Expected osTaskKill to be called once, got %d", callCount.Load())
	}
}

func TestDefaultKillLanguageServer_ContextTimeout(t *testing.T) {
	origTaskKill := osTaskKill
	t.Cleanup(func() {
		osTaskKill = origTaskKill
	})

	var callCount atomic.Int32
	osTaskKill = func(ctx context.Context) error {
		callCount.Add(1)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
			return nil
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := DefaultKillLanguageServer(ctx)
	if err == nil {
		t.Fatal("Expected error on timed out context, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected context.DeadlineExceeded, got: %v", err)
	}
	if callCount.Load() != 1 {
		t.Errorf("Expected osTaskKill to be called once, got %d", callCount.Load())
	}
}

func TestDefaultKillLanguageServer_ErrorPropagation(t *testing.T) {
	origTaskKill := osTaskKill
	t.Cleanup(func() {
		osTaskKill = origTaskKill
	})

	expectedErr := errors.New("mock process kill failure")
	osTaskKill = func(ctx context.Context) error {
		return expectedErr
	}

	ctx := context.Background()
	err := DefaultKillLanguageServer(ctx)
	if !errors.Is(err, expectedErr) {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestRecycleLanguageServer_DefaultProcessKillerDelegation(t *testing.T) {
	origTaskKill := osTaskKill
	t.Cleanup(func() {
		osTaskKill = origTaskKill
	})

	var taskKillCalled atomic.Bool
	osTaskKill = func(ctx context.Context) error {
		taskKillCalled.Store(true)
		return nil
	}

	rel := NewSessionReloader()
	rel.ProcessKiller = nil // Force default fallback to DefaultKillLanguageServer
	rel.Sleeper = func(ctx context.Context, d time.Duration) error {
		return nil
	}
	rel.ServerDiscoverer = func() (*connector.ServerInfo, error) {
		return &connector.ServerInfo{PID: 111, Port: 222}, nil
	}

	ctx := context.Background()
	err := rel.RecycleLanguageServer(ctx, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("RecycleLanguageServer failed: %v", err)
	}
	if !taskKillCalled.Load() {
		t.Fatal("Expected osTaskKill to be called via DefaultKillLanguageServer fallback")
	}
}
