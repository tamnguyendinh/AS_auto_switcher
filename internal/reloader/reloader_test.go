package reloader

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
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

func TestSyncAppStorageToS_PreservesOtherKeys(t *testing.T) {
	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "app_storage.json")

	// Pre-populate app_storage.json with rich user settings
	initialConfig := map[string]interface{}{
		"jetski.ui.layout":                    "three-panel",
		"zoomLevel":                           1.25,
		"telemetry.enabled":                   true,
		"conversations":                       []interface{}{"conv-1", "conv-2"},
		"nested":                              map[string]interface{}{"theme": "dark", "fontSize": float64(14)},
		"jetski.onboarding.lastLoginUsername": "old-user@example.com",
		"jetski.onboarding.lastLoginIsGcpTos": "false",
	}
	initialBytes, err := json.MarshalIndent(initialConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed marshaling initial config: %v", err)
	}
	if err := os.WriteFile(storagePath, initialBytes, 0644); err != nil {
		t.Fatalf("Failed writing initial storage: %v", err)
	}

	targetEmail := "updated-user@example.com"
	if err := SyncAppStorageToS(targetEmail, true, storagePath); err != nil {
		t.Fatalf("SyncAppStorageToS failed: %v", err)
	}

	data, err := os.ReadFile(storagePath)
	if err != nil {
		t.Fatalf("Failed reading updated storage: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed unmarshaling updated storage: %v", err)
	}

	// Verify target fields are updated
	if parsed["jetski.onboarding.lastLoginUsername"] != targetEmail {
		t.Errorf("Expected username %s, got %v", targetEmail, parsed["jetski.onboarding.lastLoginUsername"])
	}
	if parsed["jetski.onboarding.lastLoginIsGcpTos"] != "true" {
		t.Errorf("Expected isGcpTos 'true', got %v", parsed["jetski.onboarding.lastLoginIsGcpTos"])
	}

	// Verify 100% preservation of unrelated user configurations
	if parsed["jetski.ui.layout"] != "three-panel" {
		t.Errorf("Expected layout preserved, got %v", parsed["jetski.ui.layout"])
	}
	if parsed["zoomLevel"] != 1.25 {
		t.Errorf("Expected zoomLevel 1.25, got %v", parsed["zoomLevel"])
	}
	if parsed["telemetry.enabled"] != true {
		t.Errorf("Expected telemetry.enabled true, got %v", parsed["telemetry.enabled"])
	}
	convs, ok := parsed["conversations"].([]interface{})
	if !ok || len(convs) != 2 {
		t.Errorf("Expected conversations slice preserved, got %v", parsed["conversations"])
	}
	nested, ok := parsed["nested"].(map[string]interface{})
	if !ok || nested["theme"] != "dark" || nested["fontSize"] != float64(14) {
		t.Errorf("Expected nested config preserved, got %v", parsed["nested"])
	}
}

func TestSyncAppStorageToS_BothGcpTosStates(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("GCP account isGcpTos=true", func(t *testing.T) {
		storagePath := filepath.Join(tmpDir, "storage_gcp.json")
		email := "enterprise-gcp@company.com"
		if err := SyncAppStorageToS(email, true, storagePath); err != nil {
			t.Fatalf("SyncAppStorageToS failed: %v", err)
		}
		data, _ := os.ReadFile(storagePath)
		var parsed map[string]interface{}
		_ = json.Unmarshal(data, &parsed)
		if parsed["jetski.onboarding.lastLoginIsGcpTos"] != "true" {
			t.Errorf("Expected isGcpTos 'true', got %v", parsed["jetski.onboarding.lastLoginIsGcpTos"])
		}
		if parsed["jetski.onboarding.lastLoginUsername"] != email {
			t.Errorf("Expected username %s, got %v", email, parsed["jetski.onboarding.lastLoginUsername"])
		}
	})

	t.Run("Consumer account isGcpTos=false", func(t *testing.T) {
		storagePath := filepath.Join(tmpDir, "storage_consumer.json")
		email := "consumer@gmail.com"
		if err := SyncAppStorageToS(email, false, storagePath); err != nil {
			t.Fatalf("SyncAppStorageToS failed: %v", err)
		}
		data, _ := os.ReadFile(storagePath)
		var parsed map[string]interface{}
		_ = json.Unmarshal(data, &parsed)
		if parsed["jetski.onboarding.lastLoginIsGcpTos"] != "false" {
			t.Errorf("Expected isGcpTos 'false', got %v", parsed["jetski.onboarding.lastLoginIsGcpTos"])
		}
		if parsed["jetski.onboarding.lastLoginUsername"] != email {
			t.Errorf("Expected username %s, got %v", email, parsed["jetski.onboarding.lastLoginUsername"])
		}
	})
}

func TestSyncAppStorageToS_RecoveryFromMissingAndCorrupted(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("Missing file recovery", func(t *testing.T) {
		missingPath := filepath.Join(tmpDir, "subfolder", "nonexistent.json")
		if err := SyncAppStorageToS("fresh@example.com", false, missingPath); err != nil {
			t.Fatalf("Expected recovery on missing file, got error: %v", err)
		}
		data, err := os.ReadFile(missingPath)
		if err != nil {
			t.Fatalf("Failed to read newly created file: %v", err)
		}
		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("Newly created file is not valid JSON: %v", err)
		}
		if parsed["jetski.onboarding.lastLoginUsername"] != "fresh@example.com" {
			t.Errorf("Expected username fresh@example.com, got %v", parsed["jetski.onboarding.lastLoginUsername"])
		}
	})

	t.Run("Corrupted JSON recovery", func(t *testing.T) {
		corruptedPath := filepath.Join(tmpDir, "corrupted.json")
		if err := os.WriteFile(corruptedPath, []byte("{invalid-json: unterminated..."), 0644); err != nil {
			t.Fatalf("Failed writing corrupted file: %v", err)
		}

		if err := SyncAppStorageToS("healed@example.com", true, corruptedPath); err != nil {
			t.Fatalf("Expected graceful recovery on corrupted JSON, got error: %v", err)
		}

		data, err := os.ReadFile(corruptedPath)
		if err != nil {
			t.Fatalf("Failed reading healed file: %v", err)
		}
		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("Healed file is not valid JSON: %v", err)
		}
		if parsed["jetski.onboarding.lastLoginUsername"] != "healed@example.com" {
			t.Errorf("Expected username healed@example.com, got %v", parsed["jetski.onboarding.lastLoginUsername"])
		}
		if parsed["jetski.onboarding.lastLoginIsGcpTos"] != "true" {
			t.Errorf("Expected isGcpTos 'true', got %v", parsed["jetski.onboarding.lastLoginIsGcpTos"])
		}
	})

	t.Run("Empty file recovery", func(t *testing.T) {
		emptyPath := filepath.Join(tmpDir, "empty.json")
		if err := os.WriteFile(emptyPath, []byte(""), 0644); err != nil {
			t.Fatalf("Failed writing empty file: %v", err)
		}

		if err := SyncAppStorageToS("empty-recovered@example.com", false, emptyPath); err != nil {
			t.Fatalf("Expected recovery on empty file, got: %v", err)
		}
		data, _ := os.ReadFile(emptyPath)
		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("File after empty recovery is not valid JSON: %v", err)
		}
		if parsed["jetski.onboarding.lastLoginUsername"] != "empty-recovered@example.com" {
			t.Errorf("Expected username empty-recovered@example.com, got %v", parsed["jetski.onboarding.lastLoginUsername"])
		}
	})
}

func TestSyncAppStorageToS_AtomicWriteAndCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "app_storage.json")

	for i := 0; i < 5; i++ {
		email := "iteration-" + string(rune('a'+i)) + "@example.com"
		isGcp := (i%2 == 0)
		if err := SyncAppStorageToS(email, isGcp, storagePath); err != nil {
			t.Fatalf("Iteration %d failed: %v", i, err)
		}
	}

	// Verify no temporary files remain in the storage directory
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed reading dir: %v", err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "app_storage_tmp_") {
			t.Errorf("Leftover temporary file detected: %s", entry.Name())
		}
	}
}

func TestSyncAppStorageToS_InvalidEmail(t *testing.T) {
	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "app_storage.json")

	if err := SyncAppStorageToS("", false, storagePath); !errors.Is(err, ErrInvalidEmail) {
		t.Errorf("Expected ErrInvalidEmail for empty string, got: %v", err)
	}
	if err := SyncAppStorageToS("   ", true, storagePath); !errors.Is(err, ErrInvalidEmail) {
		t.Errorf("Expected ErrInvalidEmail for whitespace-only string, got: %v", err)
	}
}

func TestSessionReloader_SyncAppStorageToS_Method(t *testing.T) {
	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "method_test.json")

	rel := NewSessionReloader()
	rel.StoragePath = storagePath

	email := "method-user@example.com"
	if err := rel.SyncAppStorageToS(email, true); err != nil {
		t.Fatalf("rel.SyncAppStorageToS failed: %v", err)
	}

	data, err := os.ReadFile(storagePath)
	if err != nil {
		t.Fatalf("Failed reading storage written via method: %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}
	if parsed["jetski.onboarding.lastLoginUsername"] != email {
		t.Errorf("Expected username %s, got %v", email, parsed["jetski.onboarding.lastLoginUsername"])
	}
	if parsed["jetski.onboarding.lastLoginIsGcpTos"] != "true" {
		t.Errorf("Expected isGcpTos 'true', got %v", parsed["jetski.onboarding.lastLoginIsGcpTos"])
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
