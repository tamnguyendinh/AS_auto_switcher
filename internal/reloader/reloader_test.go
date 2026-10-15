package reloader

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"antigravity-switch/internal/connector"

	"golang.org/x/sys/windows"
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

func TestReloadAntigravitySession_ReauthSuccess(t *testing.T) {
	// Create mock HTTP server that handles /ReAuth
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "ReAuth") || strings.Contains(r.URL.Path, "reauth") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("{}"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	parts := strings.Split(server.URL, ":")
	port, _ := strconv.Atoi(parts[len(parts)-1])

	client := connector.NewClientWithHTTP(&connector.ServerInfo{
		PID:       1234,
		Port:      port,
		CSRFToken: "test-token",
	}, server.Client())

	reloader := NewSessionReloader()
	windowCalled := false
	reloader.WindowFinder = func() ([]windows.HWND, error) {
		windowCalled = true
		return []windows.HWND{windows.HWND(1001)}, nil
	}
	reloader.KeySender = func(hwnds []windows.HWND) error {
		return nil
	}

	// When reloadWindow=false and reauth succeeds, window reload should NOT be called
	err := reloader.ReloadAntigravitySession(context.Background(), client, false)
	if err != nil {
		t.Fatalf("ReloadAntigravitySession returned error: %v", err)
	}
	if windowCalled {
		t.Errorf("Expected window reload not to be called when reauth succeeds and reloadWindow=false")
	}
}

func TestReloadAntigravitySession_WindowFallback(t *testing.T) {
	// Reauth fails (404)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	parts := strings.Split(server.URL, ":")
	port, _ := strconv.Atoi(parts[len(parts)-1])

	client := connector.NewClientWithHTTP(&connector.ServerInfo{
		PID:       1234,
		Port:      port,
		CSRFToken: "test-token",
	}, server.Client())

	reloader := NewSessionReloader()
	var dispatchedHWNDs []windows.HWND
	reloader.WindowFinder = func() ([]windows.HWND, error) {
		return []windows.HWND{windows.HWND(42), windows.HWND(43)}, nil
	}
	reloader.KeySender = func(hwnds []windows.HWND) error {
		dispatchedHWNDs = hwnds
		return nil
	}

	err := reloader.ReloadAntigravitySession(context.Background(), client, false)
	if err != nil {
		t.Fatalf("ReloadAntigravitySession fallback failed: %v", err)
	}
	if len(dispatchedHWNDs) != 2 {
		t.Fatalf("Expected 2 HWNDs dispatched on fallback, got %d", len(dispatchedHWNDs))
	}
}

func TestReloadAntigravitySession_BothFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	parts := strings.Split(server.URL, ":")
	port, _ := strconv.Atoi(parts[len(parts)-1])

	client := connector.NewClientWithHTTP(&connector.ServerInfo{
		PID:       1234,
		Port:      port,
		CSRFToken: "test-token",
	}, server.Client())

	reloader := NewSessionReloader()
	reloader.WindowFinder = func() ([]windows.HWND, error) {
		return nil, ErrNoWindowsFound
	}

	err := reloader.ReloadAntigravitySession(context.Background(), client, true)
	if err == nil {
		t.Fatal("Expected error when both reauth and window reload fail, got nil")
	}
}
