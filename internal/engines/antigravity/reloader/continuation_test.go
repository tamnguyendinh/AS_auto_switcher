package reloader

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"antigravity-switch/internal/engines/antigravity/connector"
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
		if d == 3*time.Second {
			sleepCount++
		}
		return nil
	}

	summary, err := rel.PerformSessionContinuation(context.Background(), client, threads, "")
	if err != nil {
		t.Fatalf("PerformSessionContinuation failed: %v", err)
	}

	if summary == nil {
		t.Fatalf("Expected non-nil summary")
	}
	if summary.TotalThreads != 2 || summary.SuccessCount != 2 || summary.FailedCount != 0 {
		t.Errorf("Unexpected summary: %+v", summary)
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

func TestPerformSessionContinuation_PacingAndFaultTolerance(t *testing.T) {
	var mu sync.Mutex
	var sentThreads []string

	// Thread 1: 200 OK
	// Thread 2: 500 Internal Server Error
	// Thread 3: 200 OK
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		cascadeID, _ := body["cascadeId"].(string)

		if cascadeID == "thread-2" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if r.URL.Path == connector.EndpointSendUserCascadeMessage {
			sentThreads = append(sentThreads, cascadeID)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	parts := strings.Split(server.URL, ":")
	port, _ := strconv.Atoi(parts[len(parts)-1])

	client := connector.NewClientWithHTTP(&connector.ServerInfo{
		PID:       4321,
		Port:      port,
		CSRFToken: "csrf-val",
	}, server.Client())

	threads := []ActiveThread{
		{ThreadID: "thread-1", Project: "proj-1", IsMainPane: true},
		{ThreadID: "thread-2", Project: "proj-2", IsMainPane: false},
		{ThreadID: "thread-3", Project: "proj-3", IsMainPane: false},
	}

	var sleepCalls atomic.Int32
	rel := NewSessionReloader()
	rel.Sleeper = func(ctx context.Context, d time.Duration) error {
		if d == 3*time.Second {
			sleepCalls.Add(1)
		}
		return nil
	}

	summary, err := rel.PerformSessionContinuation(context.Background(), client, threads, "")
	if err != nil {
		t.Fatalf("PerformSessionContinuation returned unexpected error: %v", err)
	}

	if summary == nil {
		t.Fatalf("Expected non-nil ContinuationSummary")
	}

	if summary.TotalThreads != 3 {
		t.Errorf("Expected TotalThreads == 3, got %d", summary.TotalThreads)
	}
	if summary.SuccessCount != 2 {
		t.Errorf("Expected SuccessCount == 2, got %d", summary.SuccessCount)
	}
	if summary.FailedCount != 1 {
		t.Errorf("Expected FailedCount == 1, got %d", summary.FailedCount)
	}
	if len(summary.Errors) != 1 {
		t.Errorf("Expected 1 error in summary, got %v", summary.Errors)
	}

	// Sleeper must be called 2 times with 3s between the 3 threads (before thread 2 and before thread 3)
	if sleepCalls.Load() != 2 {
		t.Errorf("Expected exactly 2 pacing sleep calls of 3s, got %d", sleepCalls.Load())
	}

	mu.Lock()
	defer mu.Unlock()

	// Thread 3 must have been sent despite Thread 2 failing
	if len(sentThreads) != 2 || sentThreads[0] != "thread-1" || sentThreads[1] != "thread-3" {
		t.Errorf("Expected sentThreads to contain thread-1 and thread-3, got %v", sentThreads)
	}
}

func TestPerformSessionContinuation_ResilienceToErrors(t *testing.T) {
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

	summary, err := rel.PerformSessionContinuation(context.Background(), client, threads, "")
	if err != nil {
		t.Errorf("PerformSessionContinuation should return nil error, got %v", err)
	}
	if summary == nil || summary.FailedCount != 1 {
		t.Errorf("Expected summary with 1 failure, got %+v", summary)
	}
}
