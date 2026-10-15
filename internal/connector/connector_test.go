package connector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestExtractCSRFToken(t *testing.T) {
	testCases := []struct {
		cmdline  string
		expected string
	}{
		{
			cmdline:  `"language_server.exe" --csrf_token 349515c5-d632-4a50-95cc-a1d6195d7020 --port 1234`,
			expected: "349515c5-d632-4a50-95cc-a1d6195d7020",
		},
		{
			cmdline:  `language_server.exe --csrf_token=abc-123-xyz`,
			expected: "abc-123-xyz",
		},
		{
			cmdline:  `language_server.exe --csrf_token "token_in_quotes"`,
			expected: "token_in_quotes",
		},
		{
			cmdline:  `language_server.exe --csrf_token='token_in_single_quotes'`,
			expected: "token_in_single_quotes",
		},
		{
			cmdline:  `language_server.exe --no_token_here`,
			expected: "",
		},
		{
			cmdline:  "",
			expected: "",
		},
	}

	for _, tc := range testCases {
		actual := ExtractCSRFToken(tc.cmdline)
		if actual != tc.expected {
			t.Errorf("ExtractCSRFToken(%q) = %q; expected %q", tc.cmdline, actual, tc.expected)
		}
	}
}

func TestProbeHTTPPort(t *testing.T) {
	// Spin up a test server returning 200
	tsOk := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer tsOk.Close()

	uOk, err := url.Parse(tsOk.URL)
	if err != nil {
		t.Fatalf("Failed to parse test server url: %v", err)
	}
	portOk, _ := strconv.Atoi(uOk.Port())

	if !ProbeHTTPPort(portOk, 500*time.Millisecond) {
		t.Errorf("ProbeHTTPPort(%d) returned false for 200 OK server", portOk)
	}

	// Spin up a test server returning 400
	tsErr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer tsErr.Close()

	uErr, _ := url.Parse(tsErr.URL)
	portErr, _ := strconv.Atoi(uErr.Port())
	if ProbeHTTPPort(portErr, 500*time.Millisecond) {
		t.Errorf("ProbeHTTPPort(%d) returned true for 400 server", portErr)
	}

	// Non-listening port
	if ProbeHTTPPort(65534, 100*time.Millisecond) {
		t.Errorf("ProbeHTTPPort(65534) returned true for closed port")
	}
}

func TestGetUserStatus(t *testing.T) {
	const expectedCSRF = "test-csrf-token-12345"

	// Mock server handling GetUserStatus
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("X-Csrf-Token") != expectedCSRF || r.Header.Get("x-codeium-csrf-token") != expectedCSRF {
			http.Error(w, "missing or invalid csrf", http.StatusForbidden)
			return
		}
		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "invalid content type", http.StatusUnsupportedMediaType)
			return
		}

		if r.URL.Path == EndpointGetUserStatus {
			resp := map[string]interface{}{
				"userStatus": map[string]interface{}{
					"name":       "Test Developer",
					"email":      "dev@example.com",
					"planStatus": "PLAN_PRO",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		http.NotFound(w, r)
	})

	ts := httptest.NewServer(handler)
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	port, _ := strconv.Atoi(u.Port())

	client := NewClientWithHTTP(&ServerInfo{
		PID:       1234,
		CSRFToken: expectedCSRF,
		Port:      port,
	}, ts.Client())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	status, err := client.GetUserStatus(ctx)
	if err != nil {
		t.Fatalf("GetUserStatus failed: %v", err)
	}

	if status.Email != "dev@example.com" {
		t.Errorf("Expected email 'dev@example.com', got %q", status.Email)
	}
	if status.Name != "Test Developer" {
		t.Errorf("Expected name 'Test Developer', got %q", status.Name)
	}
	if status.Status != "PLAN_PRO" {
		t.Errorf("Expected status 'PLAN_PRO', got %q", status.Status)
	}
}

func TestGetUserStatusFallbackEndpoints(t *testing.T) {
	const expectedCSRF = "fallback-csrf-token"

	// Mock server where primary endpoint 404s, but /user_status returns 200
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == EndpointGetUserStatus {
			http.NotFound(w, r)
			return
		}
		if r.URL.Path == "/user_status" {
			resp := map[string]interface{}{
				"email":  "fallback@example.com",
				"name":   "Fallback User",
				"status": "ACTIVE",
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	})

	ts := httptest.NewServer(handler)
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	port, _ := strconv.Atoi(u.Port())

	client := NewClientWithHTTP(&ServerInfo{
		PID:       4321,
		CSRFToken: expectedCSRF,
		Port:      port,
	}, ts.Client())

	ctx := context.Background()
	status, err := client.GetUserStatus(ctx)
	if err != nil {
		t.Fatalf("GetUserStatus fallback failed: %v", err)
	}

	if status.Email != "fallback@example.com" {
		t.Errorf("Expected fallback email 'fallback@example.com', got %q", status.Email)
	}
}

func TestAuthLogoutAndTriggerReauth(t *testing.T) {
	const expectedCSRF = "auth-csrf-token"
	var logoutCalled, reauthCalled bool

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Csrf-Token") != expectedCSRF {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		if r.URL.Path == EndpointAuthLogout {
			logoutCalled = true
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.URL.Path == EndpointReauth {
			reauthCalled = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.NotFound(w, r)
	})

	ts := httptest.NewServer(handler)
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	port, _ := strconv.Atoi(u.Port())

	client := NewClientWithHTTP(&ServerInfo{
		PID:       9999,
		CSRFToken: expectedCSRF,
		Port:      port,
	}, ts.Client())

	ctx := context.Background()

	// Test AuthLogout
	if err := client.AuthLogout(ctx); err != nil {
		t.Fatalf("AuthLogout failed: %v", err)
	}
	if !logoutCalled {
		t.Errorf("Expected AuthLogout to be invoked")
	}

	// Test TriggerReauth
	if err := client.TriggerReauth(ctx); err != nil {
		t.Fatalf("TriggerReauth failed: %v", err)
	}
	if !reauthCalled {
		t.Errorf("Expected TriggerReauth to be invoked")
	}
}

func TestLoadTrajectoryAndSmartFocus(t *testing.T) {
	const testCascadeID = "cascade-abc-123"
	var receivedLoadID, receivedFocusID string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var p TrajectoryPayload
		_ = json.Unmarshal(body, &p)

		if r.URL.Path == EndpointLoadTrajectory {
			receivedLoadID = p.CascadeID
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.URL.Path == EndpointSmartFocusConversation {
			receivedFocusID = p.CascadeID
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	})

	ts := httptest.NewServer(handler)
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	port, _ := strconv.Atoi(u.Port())

	client := NewClientWithHTTP(&ServerInfo{
		PID:       5555,
		CSRFToken: "token-5555",
		Port:      port,
	}, ts.Client())

	ctx := context.Background()

	if err := client.LoadTrajectory(ctx, testCascadeID); err != nil {
		t.Fatalf("LoadTrajectory failed: %v", err)
	}
	if receivedLoadID != testCascadeID {
		t.Errorf("Expected LoadTrajectory cascadeID %q, got %q", testCascadeID, receivedLoadID)
	}

	if err := client.SmartFocusConversation(ctx, testCascadeID); err != nil {
		t.Fatalf("SmartFocusConversation failed: %v", err)
	}
	if receivedFocusID != testCascadeID {
		t.Errorf("Expected SmartFocusConversation cascadeID %q, got %q", testCascadeID, receivedFocusID)
	}
}

func TestDiscoverActiveAgentModelAndSendCascadeMessage(t *testing.T) {
	// 1. Test discovering from mock pbtxt file
	tmpDir := t.TempDir()
	pbtxtPath := filepath.Join(tmpDir, "antigravity_state.pbtxt")
	_ = os.WriteFile(pbtxtPath, []byte("some_flag: true\nlast_selected_agent_model: \"MODEL_CUSTOM_PRO\"\n"), 0600)

	dummyClient := NewClient(&ServerInfo{PID: 1, CSRFToken: "dummy", Port: 8080})
	model := dummyClient.DiscoverActiveAgentModel(context.Background(), pbtxtPath)
	if model != "MODEL_CUSTOM_PRO" {
		t.Errorf("Expected model 'MODEL_CUSTOM_PRO', got %q", model)
	}

	// 2. Test querying GetCascadeModelConfigData from mock server
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == EndpointCascadeModelConfigData {
			resp := map[string]interface{}{
				"clientModelConfigs": []map[string]interface{}{
					{
						"isSelected": true,
						"modelOrAlias": map[string]interface{}{
							"model": "MODEL_FROM_SERVER_RPC",
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if r.URL.Path == EndpointSendUserCascadeMessage {
			body, _ := io.ReadAll(r.Body)
			var payload SendCascadeMessagePayload
			if err := json.Unmarshal(body, &payload); err != nil {
				http.Error(w, "invalid json", http.StatusBadRequest)
				return
			}
			if payload.CascadeID != "cascade-999" || len(payload.Items) == 0 || payload.Items[0].Text != "continue" {
				http.Error(w, "payload mismatch", http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		}

		http.NotFound(w, r)
	})

	ts := httptest.NewServer(handler)
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	port, _ := strconv.Atoi(u.Port())

	rpcClient := NewClientWithHTTP(&ServerInfo{
		PID:       7777,
		CSRFToken: "token-7777",
		Port:      port,
	}, ts.Client())

	// Missing pbtxt -> falls back to RPC
	modelRPC := rpcClient.DiscoverActiveAgentModel(context.Background(), filepath.Join(tmpDir, "non_existent.pbtxt"))
	if modelRPC != "MODEL_FROM_SERVER_RPC" {
		t.Errorf("Expected model 'MODEL_FROM_SERVER_RPC', got %q", modelRPC)
	}

	// Test SendUserCascadeMessage
	err := rpcClient.SendUserCascadeMessage(context.Background(), "cascade-999", "continue", "", nil)
	if err != nil {
		t.Fatalf("SendUserCascadeMessage failed: %v", err)
	}
}

func TestLiveLanguageServerDiscoveryAndStatus(t *testing.T) {
	// Proactively test against live running language_server.exe on this host
	info, err := DiscoverLanguageServer()
	if err != nil {
		t.Skipf("Live language_server.exe not running on this host (%v); skipping live test", err)
		return
	}

	if info.PID <= 0 {
		t.Errorf("Discovered invalid PID: %d", info.PID)
	}
	if strings.TrimSpace(info.CSRFToken) == "" {
		t.Errorf("Discovered empty CSRF token")
	}
	if info.Port <= 0 {
		t.Errorf("Discovered invalid port: %d", info.Port)
	}

	// Verify loopback connectivity
	conn, dialErr := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", info.Port), 500*time.Millisecond)
	if dialErr != nil {
		t.Fatalf("Failed dialing discovered loopback port %d: %v", info.Port, dialErr)
	}
	_ = conn.Close()

	// Live GetUserStatus query
	client := NewClient(info)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	status, err := client.GetUserStatus(ctx)
	if err != nil {
		t.Fatalf("Live GetUserStatus failed on discovered server: %v", err)
	}

	if status.Email == "" {
		t.Errorf("Live GetUserStatus returned empty email")
	}
	t.Logf("Live language_server.exe verified: PID=%d, Port=%d, ActiveEmail=%s", info.PID, info.Port, status.Email)
}
