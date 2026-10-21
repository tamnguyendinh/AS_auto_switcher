package manager

import (
	"context"
	"fmt"
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

func TestPruneDailySwitchTimelineLogs_UnderLimit(t *testing.T) {
	tempDir := t.TempDir()

	files := []string{
		"2026-10-19_switch_timeline.log",
		"2026-10-20_switch_timeline.log",
		"2026-10-21_switch_timeline.log",
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(tempDir, f), []byte("dummy log entry\n"), 0644); err != nil {
			t.Fatalf("failed to create fixture file %s: %v", f, err)
		}
	}

	deleted, err := PruneDailySwitchTimelineLogs(tempDir, 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleted != 0 {
		t.Errorf("expected 0 deleted, got %d", deleted)
	}

	for _, f := range files {
		if _, err := os.Stat(filepath.Join(tempDir, f)); os.IsNotExist(err) {
			t.Errorf("file %q should still exist", f)
		}
	}
}

func TestPruneDailySwitchTimelineLogs_OverLimit(t *testing.T) {
	tempDir := t.TempDir()

	// Create 10 files from 2026-10-01 to 2026-10-10
	var allFiles []string
	for i := 1; i <= 10; i++ {
		filename := fmt.Sprintf("2026-10-%02d_switch_timeline.log", i)
		allFiles = append(allFiles, filename)
		if err := os.WriteFile(filepath.Join(tempDir, filename), []byte("log entry\n"), 0644); err != nil {
			t.Fatalf("failed to create fixture file %s: %v", filename, err)
		}
	}

	deleted, err := PruneDailySwitchTimelineLogs(tempDir, 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleted != 3 {
		t.Errorf("expected 3 deleted files, got %d", deleted)
	}

	// 3 oldest files (2026-10-01, 2026-10-02, 2026-10-03) must be deleted
	for i := 1; i <= 3; i++ {
		filename := fmt.Sprintf("2026-10-%02d_switch_timeline.log", i)
		if _, err := os.Stat(filepath.Join(tempDir, filename)); !os.IsNotExist(err) {
			t.Errorf("expected old file %q to be deleted", filename)
		}
	}

	// 7 newest files (2026-10-04 to 2026-10-10) must remain
	for i := 4; i <= 10; i++ {
		filename := fmt.Sprintf("2026-10-%02d_switch_timeline.log", i)
		if _, err := os.Stat(filepath.Join(tempDir, filename)); os.IsNotExist(err) {
			t.Errorf("expected retained file %q to still exist", filename)
		}
	}
}

func TestLogSwitchEvent_CreatesDailyFile(t *testing.T) {
	tempDir := t.TempDir()

	LogSwitchEvent(tempDir, "TEST_EVENT", "Test message", 150*time.Millisecond)

	today := time.Now().Format("2006-01-02")
	expectedFileName := fmt.Sprintf("%s_switch_timeline.log", today)
	expectedFilePath := filepath.Join(tempDir, expectedFileName)

	content, err := os.ReadFile(expectedFilePath)
	if err != nil {
		t.Fatalf("failed to read created timeline log file: %v", err)
	}

	strContent := string(content)
	if !strings.Contains(strContent, "[TEST_EVENT") {
		t.Errorf("expected log line to contain '[TEST_EVENT', got %q", strContent)
	}
	if !strings.Contains(strContent, "Test message") {
		t.Errorf("expected log line to contain 'Test message', got %q", strContent)
	}
	if !strings.Contains(strContent, "150ms") {
		t.Errorf("expected log line to contain '150ms', got %q", strContent)
	}
}

func TestPruneDailySwitchTimelineLogs_DefaultRetentionAndUnrelatedFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create 9 daily timeline logs
	for i := 1; i <= 9; i++ {
		filename := fmt.Sprintf("2026-10-%02d_switch_timeline.log", i)
		_ = os.WriteFile(filepath.Join(tempDir, filename), []byte("log\n"), 0644)
	}

	// Create unrelated files and subdirectories
	unrelatedFile := filepath.Join(tempDir, "unrelated_timeline.txt")
	_ = os.WriteFile(unrelatedFile, []byte("ignore"), 0644)
	subDir := filepath.Join(tempDir, "2026-10-99_switch_timeline.log")
	_ = os.MkdirAll(subDir, 0755)

	// keepCount <= 0 should default to DefaultSwitchTimelineRetentionDays (7)
	deleted, err := PruneDailySwitchTimelineLogs(tempDir, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleted != 2 {
		t.Errorf("expected 2 deleted files, got %d", deleted)
	}

	// Unrelated file and directory should not be touched
	if _, err := os.Stat(unrelatedFile); os.IsNotExist(err) {
		t.Errorf("unrelated file should not have been pruned")
	}
	if _, err := os.Stat(subDir); os.IsNotExist(err) {
		t.Errorf("sub-directory should not have been pruned")
	}
}

func TestPruneDailySwitchTimelineLogs_NonExistentDir(t *testing.T) {
	nonExistent := filepath.Join(t.TempDir(), "non_existent_timeline_folder")
	deleted, err := PruneDailySwitchTimelineLogs(nonExistent, 7)
	if err != nil {
		t.Errorf("expected nil error for non-existent directory, got %v", err)
	}
	if deleted != 0 {
		t.Errorf("expected 0 deleted, got %d", deleted)
	}
}

func TestGetTodaySwitchTimelinePath(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	expectedFileName := fmt.Sprintf("%s_switch_timeline.log", today)

	// Case 1: Custom base directory
	customDir := "custom/path"
	p := GetTodaySwitchTimelinePath(customDir)
	expectedPath := filepath.Join(customDir, expectedFileName)
	if p != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, p)
	}

	// Case 2: Empty string base directory falls back to default
	pDefault := GetTodaySwitchTimelinePath("")
	expectedDefault := filepath.Join(DefaultSwitchTimelineDir, expectedFileName)
	if pDefault != expectedDefault {
		t.Errorf("expected default path %q, got %q", expectedDefault, pDefault)
	}

	// Case 3: No arguments falls back to default
	pNoArgs := GetTodaySwitchTimelinePath()
	if pNoArgs != expectedDefault {
		t.Errorf("expected default path %q, got %q", expectedDefault, pNoArgs)
	}
}

func TestSwitchAccount_WritesDailyTimeline(t *testing.T) {
	tmpDir := t.TempDir()
	storeDir := filepath.Join(tmpDir, "accounts_store")
	accStore, err := store.NewAccountStore(storeDir)
	if err != nil {
		t.Fatalf("Failed creating account store: %v", err)
	}

	targetEmail := "target-daily@example.com"
	currentEmail := "current-daily@example.com"

	// Prepare target account in store
	targetTok := &model.TokenData{
		AccessToken: "target-access-token-xyz",
		IDToken:     makeTestJWT(targetEmail),
		Expiry:      time.Now().Add(1 * time.Hour),
	}
	if err := accStore.SaveAccount(targetEmail, targetTok); err != nil {
		t.Fatalf("Failed saving target account: %v", err)
	}

	historyDir := filepath.Join(tmpDir, "logs", "active_threads_history")
	_ = os.MkdirAll(historyDir, 0755)

	var killed atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case connector.EndpointSmartFocusConversation:
			w.WriteHeader(http.StatusOK)
		case connector.EndpointLoadTrajectory:
			w.WriteHeader(http.StatusOK)
		case connector.EndpointSendUserCascadeMessage:
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
	rel.OutputDir = historyDir
	rel.WindowFinder = func() ([]windows.HWND, error) {
		return []windows.HWND{windows.HWND(101)}, nil
	}
	rel.ProcessKiller = func(ctx context.Context) error {
		killed.Store(true)
		return nil
	}
	rel.ServerDiscoverer = func() (*connector.ServerInfo, error) {
		return &connector.ServerInfo{PID: 1234, Port: port, CSRFToken: "csrf-token"}, nil
	}
	rel.Sleeper = func(ctx context.Context, d time.Duration) error {
		return nil
	}

	mgr, err := NewAccountManager(accStore, rel)
	if err != nil {
		t.Fatalf("Failed creating manager: %v", err)
	}

	mgr.SetWincredHandlers(
		func() (*model.TokenData, error) {
			return &model.TokenData{
				AccessToken: "current-token",
				IDToken:     makeTestJWT(currentEmail),
			}, nil
		},
		func(token *model.TokenData) error {
			return nil
		},
	)

	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return connector.NewClientWithHTTP(&connector.ServerInfo{
			PID:       1234,
			Port:      port,
			CSRFToken: "csrf-token",
		}, server.Client()), nil
	})

	// 1. Successful switch
	result, err := mgr.SwitchAccount(context.Background(), targetEmail, false)
	if err != nil {
		t.Fatalf("SwitchAccount failed: %v", err)
	}
	if !result.Success {
		t.Fatalf("Expected result.Success = true, got: %s", result.Message)
	}

	// 2. Verify daily switch timeline log file
	expectedTimelineDir := filepath.Join(tmpDir, "logs", "switch_timeline")
	today := time.Now().Format("2006-01-02")
	expectedLogFile := filepath.Join(expectedTimelineDir, fmt.Sprintf("%s_switch_timeline.log", today))

	contentBytes, err := os.ReadFile(expectedLogFile)
	if err != nil {
		t.Fatalf("Expected timeline log file %q to exist: %v", expectedLogFile, err)
	}

	logContent := string(contentBytes)
	requiredTags := []string{
		"SWITCH_START",
		"SNAPSHOT_DONE",
		"WINCRED_WRITE",
		"KILL_LS",
		"RAM_CHECK_DONE",
		"SWITCH_SUCCESS",
	}
	for _, tag := range requiredTags {
		if !strings.Contains(logContent, tag) {
			t.Errorf("expected timeline log to contain tag %q, but got:\n%s", tag, logContent)
		}
	}
	if !strings.Contains(logContent, targetEmail) {
		t.Errorf("expected timeline log to contain target email %q", targetEmail)
	}

	// 3. Failed switch: verify SWITCH_FAILED tag is logged
	_, errFailed := mgr.SwitchAccount(context.Background(), "non-existent@example.com", false)
	if errFailed == nil {
		t.Fatal("expected error for non-existent target email, got nil")
	}

	contentAfterFail, err := os.ReadFile(expectedLogFile)
	if err != nil {
		t.Fatalf("failed reading log file after failure: %v", err)
	}
	if !strings.Contains(string(contentAfterFail), "SWITCH_FAILED") {
		t.Errorf("expected timeline log to contain 'SWITCH_FAILED' after failed switch")
	}
}
