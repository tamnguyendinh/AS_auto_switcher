package reloader

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"antigravity-switch/internal/connector"
)

func TestPerformSessionContinuation_HappyPath(t *testing.T) {
	var mu sync.Mutex
	var focusCalls []string
	var loadCalls []string
	var sendCalls []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)

		cascadeID, _ := body["cascadeId"].(string)

		switch r.URL.Path {
		case connector.EndpointSmartFocusConversation:
			focusCalls = append(focusCalls, cascadeID)
			w.WriteHeader(http.StatusOK)
		case connector.EndpointLoadTrajectory:
			loadCalls = append(loadCalls, cascadeID)
			w.WriteHeader(http.StatusOK)
		case connector.EndpointSendUserCascadeMessage:
			sendCalls = append(sendCalls, cascadeID)
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	parts := strings.Split(server.URL, ":")
	port, _ := strconv.Atoi(parts[len(parts)-1])

	client := connector.NewClientWithHTTP(&connector.ServerInfo{
		PID:       1234,
		Port:      port,
		CSRFToken: "test-csrf-token",
	}, server.Client())

	threads := []ActiveThread{
		{ThreadID: "cascade-1", Project: "proj-1", IsMainPane: true},
		{ThreadID: "cascade-2", Project: "proj-2", IsMainPane: false},
	}

	var sleepCount int
	rel := NewSessionReloader()
	rel.Sleeper = func(ctx context.Context, d time.Duration) error {
		sleepCount++
		return nil
	}

	err := rel.PerformSessionContinuation(context.Background(), client, threads, "")
	if err != nil {
		t.Fatalf("PerformSessionContinuation failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(focusCalls) != 2 || focusCalls[0] != "cascade-1" || focusCalls[1] != "cascade-2" {
		t.Errorf("Unexpected focus calls: %v", focusCalls)
	}
	if len(loadCalls) != 2 || loadCalls[0] != "cascade-1" || loadCalls[1] != "cascade-2" {
		t.Errorf("Unexpected load calls: %v", loadCalls)
	}
	if len(sendCalls) != 2 || sendCalls[0] != "cascade-1" || sendCalls[1] != "cascade-2" {
		t.Errorf("Unexpected send calls: %v", sendCalls)
	}
	if sleepCount != 1 {
		t.Errorf("Expected 1 pacing delay call between 2 threads, got %d", sleepCount)
	}
}

func TestPerformSessionContinuation_ResilienceToErrors(t *testing.T) {
	// Server returns 500 internal server error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	parts := strings.Split(server.URL, ":")
	port, _ := strconv.Atoi(parts[len(parts)-1])

	client := connector.NewClientWithHTTP(&connector.ServerInfo{
		PID:       1234,
		Port:      port,
		CSRFToken: "test-csrf-token",
	}, server.Client())

	threads := []ActiveThread{
		{ThreadID: "cascade-err", Project: "proj-err", IsMainPane: true},
	}

	rel := NewSessionReloader()
	rel.Sleeper = func(ctx context.Context, d time.Duration) error { return nil }

	// Continuation must be resilient and not fail the account switch flow
	err := rel.PerformSessionContinuation(context.Background(), client, threads, "")
	if err != nil {
		t.Errorf("PerformSessionContinuation should return nil even if RPC returns error, got %v", err)
	}
}
