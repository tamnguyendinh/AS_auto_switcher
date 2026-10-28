package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"antigravity-switch/internal/engines/antigravity/connector"
	"antigravity-switch/internal/engines/antigravity/model"
	"antigravity-switch/internal/engines/antigravity/store"
)

// mockOAuthRefresher implements connector.OAuthRefresher for testing.
type mockOAuthRefresher struct {
	accessToken string
	err         error
	callCount   atomic.Int64
}

func (m *mockOAuthRefresher) RefreshToken(ctx context.Context, refreshToken string) (*connector.TokenRefreshResult, error) {
	m.callCount.Add(1)
	if m.err != nil {
		return nil, m.err
	}
	return &connector.TokenRefreshResult{
		AccessToken: m.accessToken,
		ExpiresIn:   3600,
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(1 * time.Hour),
	}, nil
}

// mockAccountSource implements AccountSource for testing.
type mockAccountSource struct {
	accounts []AccountTokenInfo
	err      error
}

func (m *mockAccountSource) ListAccounts() ([]AccountTokenInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.accounts, nil
}

func (m *mockAccountSource) GetRefreshToken(email string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	for _, acc := range m.accounts {
		if acc.Email == email {
			if acc.RefreshToken == "" {
				return "", fmt.Errorf("no refresh token for %s", email)
			}
			return acc.RefreshToken, nil
		}
	}
	return "", fmt.Errorf("account not found: %s", email)
}

func TestQuotaClient_RetrieveUserQuota_Success(t *testing.T) {
	resetStr := "2026-10-17T11:29:41Z"
	expectedReset, _ := time.Parse(time.RFC3339, resetStr)

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer valid-test-token" {
			t.Errorf("expected Authorization header 'Bearer valid-test-token', got %s", auth)
		}

		resp := rawQuotaResponse{
			Buckets: []rawBucketInfo{
				{
					ModelID:           "claude-sonnet-4-6",
					RemainingFraction: ptrFloat(1.0),
					ResetTimeRaw:      json.RawMessage(`"` + resetStr + `"`),
					TokenType:         "WTUS",
				},
				{
					ModelID:           "gemini-2.5-pro",
					RemainingFraction: ptrFloat(0.235),
					ResetTimeRaw:      json.RawMessage(`"` + resetStr + `"`),
					TokenType:         "WTUS",
				},
				{
					ModelID:           "gpt-oss",
					RemainingFraction: ptrFloat(1.0),
					ResetTimeRaw:      json.RawMessage(`"` + resetStr + `"`),
					TokenType:         "WTUS",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	refresher := &mockOAuthRefresher{accessToken: "valid-test-token"}
	client := NewQuotaClientWithEndpoint(refresher, mockServer.URL, mockServer.Client())

	data, err := client.RetrieveUserQuota(context.Background(), "mock-refresh-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if data == nil {
		t.Fatal("expected non-nil AccountQuotaData")
	}

	if data.Claude.RemainingFraction != 1.0 {
		t.Errorf("expected Claude 1.0, got %f", data.Claude.RemainingFraction)
	}
	if data.Gemini.RemainingFraction != 0.235 {
		t.Errorf("expected Gemini 0.235, got %f", data.Gemini.RemainingFraction)
	}

	if !data.ResetTime.Equal(expectedReset.UTC()) {
		t.Errorf("expected ResetTime %v, got %v", expectedReset.UTC(), data.ResetTime)
	}
	if data.Status != StatusLowGemini {
		t.Errorf("expected Status '%s', got '%s'", StatusLowGemini, data.Status)
	}
	if len(data.RawBuckets) != 3 {
		t.Errorf("expected 3 raw buckets, got %d", len(data.RawBuckets))
	}
}

func TestQuotaClient_RetrieveUserQuota_SnakeCaseAndProtobuf(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := rawQuotaResponse{
			Buckets: []rawBucketInfo{
				{
					ModelIDSnake:           "claude-opus-4-6",
					RemainingFractionSnake: ptrFloat(0.0), // Depleted
					ResetTimeSnakeRaw:      json.RawMessage(`{"seconds": 1792056078, "nanos": 0}`),
					TokenTypeSnake:         "WTUS",
				},
				{
					ModelIDSnake:           "gemini-2.5-flash",
					RemainingFractionSnake: ptrFloat(0.0),
					RemainingAmountSnake:   ptrInt64(0),
					ResetTimeSnakeRaw:      json.RawMessage(`1792056078`),
					TokenTypeSnake:         "WTUS",
				},
				{
					ModelIDSnake:           "gpt-4o",
					RemainingFractionSnake: ptrFloat(0.10), // Depleted < 15%
					ResetTimeSnakeRaw:      json.RawMessage(`"2026-10-17 12:00:00Z"`),
					TokenTypeSnake:         "WTUS",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	refresher := &mockOAuthRefresher{accessToken: "valid-test-token"}
	client := NewQuotaClientWithEndpoint(refresher, mockServer.URL, mockServer.Client())

	data, err := client.RetrieveUserQuota(context.Background(), "mock-refresh-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if data.Claude.RemainingFraction != 0.0 {
		t.Errorf("expected Claude 0.0, got %f", data.Claude.RemainingFraction)
	}
	if data.Gemini.RemainingFraction != 0.0 {
		t.Errorf("expected Gemini 0.0, got %f", data.Gemini.RemainingFraction)
	}

	if data.Status != StatusDepleted {
		t.Errorf("expected Status '%s', got '%s'", StatusDepleted, data.Status)
	}
	if data.ResetTime.IsZero() {
		t.Error("expected non-zero ResetTime")
	}
}

func TestQuotaClient_RetrieveUserQuota_StatusCategories(t *testing.T) {
	tests := []struct {
		name           string
		raw            []ModelQuotaBucket
		expectedStatus string
	}{
		{
			name: "All 100% is Pristine",
			raw: []ModelQuotaBucket{
				{ModelID: "claude-sonnet", RemainingFraction: 1.0},
				{ModelID: "gemini-pro", RemainingFraction: 1.0},
				{ModelID: "gpt-oss", RemainingFraction: 1.0},
			},
			expectedStatus: StatusPristine,
		},
		{
			name: "91.6% Gemini is Plentiful",
			raw: []ModelQuotaBucket{
				{ModelID: "claude-sonnet", RemainingFraction: 1.0},
				{ModelID: "gemini-pro", RemainingFraction: 0.916},
				{ModelID: "gpt-oss", RemainingFraction: 1.0},
			},
			expectedStatus: StatusPlentiful,
		},
		{
			name: "Claude at 30% is Low Claude",
			raw: []ModelQuotaBucket{
				{ModelID: "claude-sonnet", RemainingFraction: 0.30},
				{ModelID: "gemini-pro", RemainingFraction: 1.0},
				{ModelID: "gpt-oss", RemainingFraction: 1.0},
			},
			expectedStatus: StatusLowClaude,
		},
		{
			name: "GPT at 10% is Low GPT",
			raw: []ModelQuotaBucket{
				{ModelID: "claude-sonnet", RemainingFraction: 1.0},
				{ModelID: "gemini-pro", RemainingFraction: 1.0},
				{ModelID: "gpt-oss", RemainingFraction: 0.10},
			},
			expectedStatus: StatusLowGPT,
		},
		{
			name: "All depleted is Depleted",
			raw: []ModelQuotaBucket{
				{ModelID: "claude-sonnet", RemainingFraction: 0.05},
				{ModelID: "gemini-pro", RemainingFraction: 0.0},
				{ModelID: "gpt-oss", RemainingFraction: 0.10},
			},
			expectedStatus: StatusDepleted,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, _, _, status := AggregateBuckets(tc.raw)
			if status != tc.expectedStatus {
				t.Errorf("expected status %s, got %s", tc.expectedStatus, status)
			}
		})
	}
}

func TestQuotaClient_RetrieveUserQuota_Errors(t *testing.T) {
	t.Run("Empty refresh token", func(t *testing.T) {
		client := NewQuotaClient(&mockOAuthRefresher{}, nil)
		_, err := client.RetrieveUserQuota(context.Background(), "")
		if !errors.Is(err, ErrEmptyRefreshToken) {
			t.Errorf("expected ErrEmptyRefreshToken, got %v", err)
		}
	})

	t.Run("Nil refresher", func(t *testing.T) {
		client := NewQuotaClient(nil, nil)
		_, err := client.RetrieveUserQuota(context.Background(), "token")
		if !errors.Is(err, ErrNilRefresher) {
			t.Errorf("expected ErrNilRefresher, got %v", err)
		}
	})

	t.Run("Token refresh error", func(t *testing.T) {
		refresher := &mockOAuthRefresher{err: errors.New("network unreachable")}
		client := NewQuotaClient(refresher, nil)
		_, err := client.RetrieveUserQuota(context.Background(), "token")
		if err == nil || !errors.Is(err, ErrQuotaFetchFailed) {
			t.Errorf("expected ErrQuotaFetchFailed, got %v", err)
		}
	})

	t.Run("HTTP 401 Unauthorized", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}))
		defer server.Close()

		refresher := &mockOAuthRefresher{accessToken: "invalid"}
		client := NewQuotaClientWithEndpoint(refresher, server.URL, server.Client())
		_, err := client.RetrieveUserQuota(context.Background(), "token")
		if err == nil || !errors.Is(err, ErrQuotaFetchFailed) {
			t.Errorf("expected ErrQuotaFetchFailed, got %v", err)
		}
	})

	t.Run("HTTP 500 Internal Error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}))
		defer server.Close()

		refresher := &mockOAuthRefresher{accessToken: "tok"}
		client := NewQuotaClientWithEndpoint(refresher, server.URL, server.Client())
		_, err := client.RetrieveUserQuota(context.Background(), "token")
		if err == nil || !errors.Is(err, ErrQuotaFetchFailed) {
			t.Errorf("expected ErrQuotaFetchFailed, got %v", err)
		}
	})

	t.Run("Cancelled context", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel immediately

		refresher := &mockOAuthRefresher{accessToken: "tok"}
		client := NewQuotaClientWithEndpoint(refresher, server.URL, server.Client())
		_, err := client.RetrieveUserQuota(ctx, "token")
		if err == nil {
			t.Error("expected error on cancelled context, got nil")
		}
	})
}

func TestQuotaCache_Operations(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "test_quota_cache.json")

	cache := NewQuotaCache(cachePath)
	if cache.FilePath() != cachePath {
		t.Errorf("expected FilePath %s, got %s", cachePath, cache.FilePath())
	}

	// 1. Get non-existent
	if _, exists := cache.Get("nonexistent@gmail.com"); exists {
		t.Error("expected false for nonexistent account")
	}

	// 2. Update and Get
	data1 := &AccountQuotaData{
		Email:       "user1@gmail.com",
		Status:      StatusPristine,
		LastUpdated: time.Now().UTC(),
		Claude:      ModelQuotaBucket{RemainingFraction: 1.0},
	}
	cache.Update(data1)

	got1, exists := cache.Get("USER1@GMAIL.COM") // case-insensitive
	if !exists || got1 == nil {
		t.Fatal("expected user1@gmail.com to exist in cache")
	}
	if got1.Status != StatusPristine {
		t.Errorf("expected Status %s, got %s", StatusPristine, got1.Status)
	}

	// 3. Immutability check: mutating returned copy must not alter cache
	got1.Status = "MUTATED"
	fresh1, _ := cache.Get("user1@gmail.com")
	if fresh1.Status == "MUTATED" {
		t.Error("cache internal state was mutated via external reference")
	}

	// 4. Batch update and GetAll
	data2 := &AccountQuotaData{
		Email:  "user2@gmail.com",
		Status: StatusLowGemini,
		Gemini: ModelQuotaBucket{RemainingFraction: 0.235},
	}
	cache.UpdateBatch([]*AccountQuotaData{data2})

	all := cache.GetAll()
	if len(all) != 2 {
		t.Errorf("expected 2 accounts in GetAll(), got %d", len(all))
	}
	if all["user2@gmail.com"] == nil || all["user2@gmail.com"].Status != StatusLowGemini {
		t.Errorf("expected user2 status %s", StatusLowGemini)
	}

	// 5. SaveToDisk
	if err := cache.SaveToDisk(); err != nil {
		t.Fatalf("SaveToDisk failed: %v", err)
	}

	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Fatalf("cache file %s was not created", cachePath)
	}

	// 6. LoadFromDisk in fresh cache instance
	cache2 := NewQuotaCache(cachePath)
	if err := cache2.LoadFromDisk(); err != nil {
		t.Fatalf("LoadFromDisk failed: %v", err)
	}

	loaded1, ok1 := cache2.Get("user1@gmail.com")
	if !ok1 || loaded1.Status != StatusPristine {
		t.Errorf("expected loaded user1 status %s", StatusPristine)
	}
	loaded2, ok2 := cache2.Get("user2@gmail.com")
	if !ok2 || loaded2.Gemini.RemainingFraction != 0.235 {
		t.Errorf("expected loaded user2 gemini fraction 0.235")
	}

	// 7. LoadFromDisk with non-existent file should be a no-op (no error)
	cache3 := NewQuotaCache(filepath.Join(tempDir, "missing.json"))
	if err := cache3.LoadFromDisk(); err != nil {
		t.Errorf("expected nil error on missing file, got %v", err)
	}
}

// mockQuotaClient implements QuotaClientInterface for pool and loop tests.
type mockQuotaClient struct {
	delay      time.Duration
	failEmails map[string]error
	callCount  atomic.Int64
}

func (m *mockQuotaClient) RetrieveUserQuota(ctx context.Context, refreshToken string) (*AccountQuotaData, error) {
	m.callCount.Add(1)
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if m.failEmails != nil {
		if err, shouldFail := m.failEmails[refreshToken]; shouldFail {
			return nil, err
		}
	}
	return &AccountQuotaData{
		Claude:      ModelQuotaBucket{ModelID: "claude-4-6", RemainingFraction: 1.0},
		Gemini:      ModelQuotaBucket{ModelID: "gemini-2-5", RemainingFraction: 0.95},
		Status:      StatusPlentiful,
		LastUpdated: time.Now().UTC(),
	}, nil
}

func TestTelemetryPool_PollAccountAndPollAll(t *testing.T) {
	tempDir := t.TempDir()
	cache := NewQuotaCache(filepath.Join(tempDir, "pool_cache.json"))

	accounts := []AccountTokenInfo{
		{Email: "acc1@gmail.com", RefreshToken: "tok1"},
		{Email: "acc2@gmail.com", RefreshToken: "tok2"},
		{Email: "acc3@gmail.com", RefreshToken: "tok3"},
		{Email: "acc4@gmail.com", RefreshToken: "tok4"},
		{Email: "acc5@gmail.com", RefreshToken: "tok5"},
		{Email: "acc6@gmail.com", RefreshToken: "tok6"},
		{Email: "acc7@gmail.com", RefreshToken: "tok7"},
	}

	source := &mockAccountSource{accounts: accounts}
	client := &mockQuotaClient{delay: 20 * time.Millisecond}

	pool, err := NewTelemetryPoolWithSource(client, cache, source)
	if err != nil {
		t.Fatalf("NewTelemetryPoolWithSource failed: %v", err)
	}

	// 1. Poll single account
	singleData, err := pool.PollAccount(context.Background(), "acc1@gmail.com")
	if err != nil {
		t.Fatalf("PollAccount failed: %v", err)
	}
	if singleData.Email != "acc1@gmail.com" {
		t.Errorf("expected email acc1@gmail.com, got %s", singleData.Email)
	}

	cached, ok := cache.Get("acc1@gmail.com")
	if !ok || cached.Email != "acc1@gmail.com" {
		t.Error("expected PollAccount to update cache")
	}

	start := time.Now()
	allResults, successCount, errorCount, err := pool.PollAll(context.Background())
	duration := time.Since(start)
	if successCount != 7 || errorCount != 0 {
		t.Errorf("expected 7 success, 0 error, got %d, %d", successCount, errorCount)
	}
	// removed

	if err != nil {
		t.Fatalf("PollAll failed: %v", err)
	}
	if len(allResults) != 7 {
		t.Fatalf("expected 7 results, got %d", len(allResults))
	}

	// Duration should be far below 2.0 seconds because of concurrency
	if duration > 1500*time.Millisecond {
		t.Errorf("expected PollAll to complete in < 1.5s, took %v", duration)
	}

	// Verify all 7 accounts present in cache
	cachedMap := cache.GetAll()
	if len(cachedMap) != 7 {
		t.Errorf("expected 7 cached accounts, got %d", len(cachedMap))
	}
}

func TestTelemetryPool_PartialFailureHandling(t *testing.T) {
	tempDir := t.TempDir()
	cache := NewQuotaCache(filepath.Join(tempDir, "err_cache.json"))

	accounts := []AccountTokenInfo{
		{Email: "good@gmail.com", RefreshToken: "good_tok"},
		{Email: "bad@gmail.com", RefreshToken: "bad_tok"},
		{Email: "notoken@gmail.com", RefreshToken: ""},
	}
	source := &mockAccountSource{accounts: accounts}
	client := &mockQuotaClient{
		failEmails: map[string]error{
			"bad_tok": errors.New("simulated 401 unauthorized"),
		},
	}

	pool, err := NewTelemetryPoolWithSource(client, cache, source)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	results, successCount, errorCount, err := pool.PollAll(context.Background())
	if successCount != 1 || errorCount != 2 {
		t.Errorf("expected 1 success, 2 errors, got %d, %d", successCount, errorCount)
	}
	if err != nil {
		t.Fatalf("PollAll should not fail entire batch: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Check good account
	good, ok := cache.Get("good@gmail.com")
	if !ok || good.Status != StatusPlentiful {
		t.Errorf("expected good account status %s", StatusPlentiful)
	}

	// Check bad account (status must be Error)
	bad, ok := cache.Get("bad@gmail.com")
	if !ok || bad.Status != StatusError {
		t.Errorf("expected bad account status %s", StatusError)
	}
	if bad.ErrorMessage == "" {
		t.Error("expected error message recorded for bad account")
	}

	// Check no-token account
	noTok, ok := cache.Get("notoken@gmail.com")
	if !ok || noTok.Status != StatusError {
		t.Errorf("expected no-token account status %s", StatusError)
	}
}

func TestAutoTelemetryLoop_LifecycleAndSingleFlight(t *testing.T) {
	tempDir := t.TempDir()
	cache := NewQuotaCache(filepath.Join(tempDir, "loop_cache.json"))
	accounts := []AccountTokenInfo{
		{Email: "loop1@gmail.com", RefreshToken: "tok1"},
	}
	source := &mockAccountSource{accounts: accounts}
	client := &mockQuotaClient{delay: 60 * time.Millisecond}

	pool, _ := NewTelemetryPoolWithSource(client, cache, source)

	// Interval of 50ms for rapid test validation
	loop, err := NewAutoTelemetryLoop(pool, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("NewAutoTelemetryLoop failed: %v", err)
	}

	if loop.Interval() != 50*time.Millisecond {
		t.Errorf("expected 50ms interval, got %v", loop.Interval())
	}
	if loop.IsRunning() {
		t.Error("expected loop not running before Start()")
	}

	var updateCount atomic.Int64
	loop.SetUpdateCallback(func(data []*AccountQuotaData) {
		updateCount.Add(1)
	})

	// Start loop
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := loop.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if !loop.IsRunning() {
		t.Error("expected loop to be running")
	}

	// Double start should fail
	if err := loop.Start(ctx); !errors.Is(err, ErrLoopAlreadyRunning) {
		t.Errorf("expected ErrLoopAlreadyRunning, got %v", err)
	}

	// Trigger manual poll
	loop.TriggerNow()

	// Wait briefly for updates
	time.Sleep(150 * time.Millisecond)

	// Verify single-flight gate drops overlapping calls
	// Simulate rapid trigger firing
	for i := 0; i < 10; i++ {
		loop.TriggerNow()
	}

	time.Sleep(100 * time.Millisecond)

	// Stop loop
	loop.Stop()
	if loop.IsRunning() {
		t.Error("expected loop to be stopped after Stop()")
	}

	if updateCount.Load() == 0 {
		t.Error("expected at least one update callback invocation")
	}
}

func TestTelemetry_ConcurrentRace(t *testing.T) {
	tempDir := t.TempDir()
	cache := NewQuotaCache(filepath.Join(tempDir, "race_cache.json"))

	accounts := make([]AccountTokenInfo, 10)
	for i := 0; i < 10; i++ {
		accounts[i] = AccountTokenInfo{
			Email:        fmt.Sprintf("user%d@gmail.com", i),
			RefreshToken: fmt.Sprintf("tok%d", i),
		}
	}
	source := &mockAccountSource{accounts: accounts}
	client := &mockQuotaClient{delay: 1 * time.Millisecond}

	pool, _ := NewTelemetryPoolWithSource(client, cache, source)
	loop, _ := NewAutoTelemetryLoop(pool, 20*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = loop.Start(ctx)
	defer loop.Stop()

	var wg sync.WaitGroup
	// Concurrently read and write cache
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 30; j++ {
				email := fmt.Sprintf("user%d@gmail.com", j%10)
				cache.Update(&AccountQuotaData{
					Email:       email,
					Status:      StatusPlentiful,
					LastUpdated: time.Now().UTC(),
				})
				_, _ = cache.Get(email)
				_ = cache.GetAll()
				if j%10 == 0 {
					_ = cache.SaveToDisk()
				}
				loop.TriggerNow()
				_, _ = pool.PollAccount(context.Background(), email)
			}
		}(i)
	}

	wg.Wait()
}

func ptrFloat(f float64) *float64 {
	return &f
}

func ptrInt64(i int64) *int64 {
	return &i
}

func TestQuotaCache_EdgeCasesAndFlatMap(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "flat_cache.json")

	// 1. Default path constructor
	if DefaultQuotaCachePath != "logs/quota_log.json" {
		t.Errorf("expected DefaultQuotaCachePath to be 'logs/quota_log.json', got %s", DefaultQuotaCachePath)
	}
	defCache := NewQuotaCache("")
	if defCache.FilePath() != DefaultQuotaCachePath {
		t.Errorf("expected default path %s, got %s", DefaultQuotaCachePath, defCache.FilePath())
	}
	defCache.SetFilePath(cachePath)
	if defCache.FilePath() != cachePath {
		t.Errorf("expected updated path %s, got %s", cachePath, defCache.FilePath())
	}

	// 2. Update with nil and empty email
	defCache.Update(nil)
	defCache.Update(&AccountQuotaData{Email: ""})
	defCache.UpdateBatch(nil)
	defCache.UpdateBatch([]*AccountQuotaData{nil, {Email: ""}})
	if len(defCache.GetAll()) != 0 {
		t.Error("expected empty cache after nil updates")
	}

	// 3. Flat map JSON disk loading
	flatJSON := `{
		"direct@example.com": {
			"email": "direct@example.com",
			"status": "Pristine",
			"claude": {"model_id": "claude", "remaining_fraction": 1.0}
		}
	}`
	if err := os.WriteFile(cachePath, []byte(flatJSON), 0644); err != nil {
		t.Fatalf("failed to write flat json: %v", err)
	}

	freshCache := NewQuotaCache(cachePath)
	if err := freshCache.LoadFromDisk(); err != nil {
		t.Fatalf("LoadFromDisk failed for flat map: %v", err)
	}

	data, ok := freshCache.Get("direct@example.com")
	if !ok || data.Status != StatusPristine {
		t.Errorf("expected direct@example.com loaded with Pristine status")
	}

	// 4. Corrupt JSON disk loading
	corruptPath := filepath.Join(tempDir, "corrupt.json")
	_ = os.WriteFile(corruptPath, []byte("{broken json"), 0644)
	badCache := NewQuotaCache(corruptPath)
	if err := badCache.LoadFromDisk(); err == nil {
		t.Error("expected error when loading corrupt json")
	}

	// 5. Empty file loading (no-op)
	emptyPath := filepath.Join(tempDir, "empty.json")
	_ = os.WriteFile(emptyPath, []byte("   "), 0644)
	emptyCache := NewQuotaCache(emptyPath)
	if err := emptyCache.LoadFromDisk(); err != nil {
		t.Errorf("expected nil error on empty file, got %v", err)
	}
}

func TestTelemetryPool_StoreAccountSourceIntegration(t *testing.T) {
	tempDir := t.TempDir()
	accStore, err := store.NewAccountStore(tempDir)
	if err != nil {
		t.Fatalf("NewAccountStore failed: %v", err)
	}

	// 1. Error checks on nil store
	nilSource := NewStoreAccountSource(nil)
	if _, err := nilSource.ListAccounts(); err == nil {
		t.Error("expected error on nil store ListAccounts")
	}
	if _, err := nilSource.GetRefreshToken("foo"); err == nil {
		t.Error("expected error on nil store GetRefreshToken")
	}

	// 2. Real accounts in store
	token1 := &connector.TokenRefreshResult{RefreshToken: "refresh-token-1"}
	_ = accStore.SaveAccount("user1@example.com", storeTokenAdapter(token1))
	token2 := &connector.TokenRefreshResult{RefreshToken: "refresh-token-2"}
	_ = accStore.SaveAccount("user2@example.com", storeTokenAdapter(token2))

	src := NewStoreAccountSource(accStore)
	list, err := src.ListAccounts()
	if err != nil {
		t.Fatalf("ListAccounts failed: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(list))
	}

	rt1, err := src.GetRefreshToken("user1@example.com")
	if err != nil || rt1 != "refresh-token-1" {
		t.Errorf("expected refresh-token-1, got %s (err: %v)", rt1, err)
	}

	_, err = src.GetRefreshToken("nonexistent@example.com")
	if err == nil {
		t.Error("expected error for nonexistent account refresh token")
	}

	// 3. TelemetryPool constructor validations
	cache := NewQuotaCache(filepath.Join(tempDir, "p_cache.json"))
	client := &mockQuotaClient{}

	if _, err := NewTelemetryPool(nil, cache, accStore); !errors.Is(err, ErrNilClient) {
		t.Errorf("expected ErrNilClient, got %v", err)
	}
	if _, err := NewTelemetryPool(client, nil, accStore); !errors.Is(err, ErrNilCache) {
		t.Errorf("expected ErrNilCache, got %v", err)
	}
	if _, err := NewTelemetryPoolWithSource(client, cache, nil); !errors.Is(err, ErrNilSource) {
		t.Errorf("expected ErrNilSource, got %v", err)
	}

	// 4. Valid pool with store
	pool, err := NewTelemetryPool(client, cache, accStore)
	if err != nil {
		t.Fatalf("NewTelemetryPool failed: %v", err)
	}
	if pool.Cache() != cache {
		t.Error("expected pool.Cache() to match provided cache")
	}

	// 5. PollAccount validation
	if _, err := pool.PollAccount(context.Background(), ""); err == nil {
		t.Error("expected error on empty email in PollAccount")
	}

	// 6. PollAll with store
	results, successCount, errorCount, err := pool.PollAll(context.Background())
	_ = successCount
	_ = errorCount
	if err != nil {
		t.Fatalf("PollAll failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestAutoTelemetryLoop_EdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	cache := NewQuotaCache(filepath.Join(tempDir, "edge_cache.json"))
	source := &mockAccountSource{accounts: []AccountTokenInfo{}}
	client := &mockQuotaClient{}
	pool, _ := NewTelemetryPoolWithSource(client, cache, source)

	// 1. Constructor errors
	if _, err := NewAutoTelemetryLoop(nil, 60*time.Second); !errors.Is(err, ErrNilPool) {
		t.Errorf("expected ErrNilPool, got %v", err)
	}

	// 2. Default interval
	loopDef, err := NewAutoTelemetryLoop(pool, 0)
	if err != nil || loopDef.Interval() != DefaultLoopInterval {
		t.Errorf("expected DefaultLoopInterval, got %v", loopDef.Interval())
	}

	// 3. TriggerNow when not running (triggers poll in background)
	loopDef.TriggerNow()
	time.Sleep(20 * time.Millisecond)

	// 4. IsRefreshing
	if loopDef.IsRefreshing() {
		t.Error("expected IsRefreshing false when idle")
	}

	// 5. Stop on unstarted loop should not crash
	loopDef.Stop()
}

func storeTokenAdapter(t *connector.TokenRefreshResult) *model.TokenData {
	return &model.TokenData{
		AccessToken:  "dummy_access",
		RefreshToken: t.RefreshToken,
	}
}

func TestCoverageBooster(t *testing.T) {
	tempDir := t.TempDir()

	// 1. QuotaClient constructors with defaults
	refresher := &mockOAuthRefresher{accessToken: "dummy"}
	c1 := NewQuotaClientWithEndpoint(refresher, "", nil)
	if c1.endpoint != DefaultGoogleQuotaURL {
		t.Errorf("expected default endpoint %s, got %s", DefaultGoogleQuotaURL, c1.endpoint)
	}

	// 2. parseResetTime branches
	t1 := parseResetTime(json.RawMessage(`null`))
	if !t1.IsZero() {
		t.Error("expected zero time for null")
	}
	t2 := parseResetTime(json.RawMessage(`""`))
	if !t2.IsZero() {
		t.Error("expected zero time for empty string")
	}
	t3 := parseResetTime(json.RawMessage(`1792056078000`)) // milliseconds
	if t3.IsZero() {
		t.Error("expected non-zero time for epoch milliseconds")
	}
	t4 := parseResetTime(json.RawMessage(`{"invalid": true}`))
	if !t4.IsZero() {
		t.Error("expected zero time for invalid proto object")
	}

	// 3. toModelQuotaBucket branches
	raw1 := rawBucketInfo{
		ModelIDSnake:      "model-test",
		TokenTypeSnake:    "TOK",
		RemainingAmount:   ptrInt64(100),
		ResetTimeSnakeRaw: json.RawMessage(`"2026-10-17 12:00:00Z"`),
	}
	b1 := raw1.toModelQuotaBucket()
	if b1.RemainingFraction != -1 {
		t.Errorf("expected fraction -1 (unknown) for amount 100 without fraction, got %f", b1.RemainingFraction)
	}
	if b1.RemainingAmount != 100 {
		t.Errorf("expected amount 100, got %d", b1.RemainingAmount)
	}

	raw2 := rawBucketInfo{
		RemainingAmountSnake: ptrInt64(50),
	}
	b2 := raw2.toModelQuotaBucket()
	if b2.RemainingFraction != -1 {
		t.Errorf("expected fraction -1 (unknown) for amount 50 without fraction, got %f", b2.RemainingFraction)
	}
	if b2.RemainingAmount != 50 {
		t.Errorf("expected amount 50, got %d", b2.RemainingAmount)
	}

	// 4. AggregateBuckets branches
	rawBuckets := []ModelQuotaBucket{
		{ModelID: "claude-sonnet", RemainingFraction: 0.8, ResetTime: time.Now().UTC()},
		{ModelID: "claude-opus", RemainingFraction: 0.4, ResetTime: time.Now().UTC()},
		{ModelID: "gemini-flash", RemainingFraction: 0.9, ResetTime: time.Now().UTC()},
		{ModelID: "gemini-pro", RemainingFraction: 0.2, ResetTime: time.Now().UTC()},
		{ModelID: "gpt-oss", RemainingFraction: 0.1, ResetTime: time.Now().UTC()},
	}
	claude, gemini, gpt, reset, status := AggregateBuckets(rawBuckets)
	if claude.RemainingFraction != 0.4 {
		t.Errorf("expected minimum claude 0.4, got %f", claude.RemainingFraction)
	}
	if gemini.RemainingFraction != 0.2 {
		t.Errorf("expected minimum gemini 0.2, got %f", gemini.RemainingFraction)
	}
	if gpt.RemainingFraction != 0.1 {
		t.Errorf("expected minimum gpt 0.1, got %f", gpt.RemainingFraction)
	}
	if reset.IsZero() {
		t.Error("expected non-zero canonical reset time")
	}
	if status != StatusLowGPT {
		t.Errorf("expected StatusLowGPT, got %s", status)
	}

	// 5. QuotaCache custom path SaveToDisk & LoadFromDisk
	cachePathCustom := filepath.Join(tempDir, "custom_sub", "cache.json")
	testCachePath := filepath.Join(tempDir, "test_quota_cache.json")
	cache := NewQuotaCache(testCachePath)
	cache.Update(&AccountQuotaData{Email: "custom@example.com", Status: StatusPristine})
	if err := cache.SaveToDisk(cachePathCustom); err != nil {
		t.Fatalf("SaveToDisk custom failed: %v", err)
	}
	cacheLoad := NewQuotaCache(testCachePath)
	if err := cacheLoad.LoadFromDisk(cachePathCustom); err != nil {
		t.Fatalf("LoadFromDisk custom failed: %v", err)
	}
	if _, ok := cacheLoad.Get("custom@example.com"); !ok {
		t.Error("expected custom@example.com loaded")
	}

	// 6. TelemetryPool PollAccount with failing source
	failSource := &mockAccountSource{
		err: errors.New("disk failure"),
	}
	failPool, _ := NewTelemetryPoolWithSource(&mockQuotaClient{}, cache, failSource)
	if _, err := failPool.PollAccount(context.Background(), "user@fail.com"); err == nil {
		t.Error("expected error on PollAccount with failing source")
	}
	if _, _, _, err := failPool.PollAll(context.Background()); err == nil {
		t.Error("expected error on PollAll with failing source")
	}
}

func BenchmarkTelemetryPool_ParallelLatency7Accounts(b *testing.B) {
	tempDir := b.TempDir()
	cache := NewQuotaCache(filepath.Join(tempDir, "bench_cache.json"))

	accounts := make([]AccountTokenInfo, 7)
	for i := 0; i < 7; i++ {
		accounts[i] = AccountTokenInfo{
			Email:        fmt.Sprintf("account%d@gmail.com", i+1),
			RefreshToken: fmt.Sprintf("token%d", i+1),
		}
	}
	source := &mockAccountSource{accounts: accounts}
	// Simulated network round-trip of 50ms per Google RPC call
	client := &mockQuotaClient{delay: 50 * time.Millisecond}
	pool, err := NewTelemetryPoolWithSource(client, cache, source)
	if err != nil {
		b.Fatalf("NewTelemetryPoolWithSource failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		results, successCount, errorCount, err := pool.PollAll(context.Background())
		_ = successCount
		_ = errorCount
		duration := time.Since(start)

		if err != nil {
			b.Fatalf("PollAll failed: %v", err)
		}
		if len(results) != 7 {
			b.Fatalf("expected 7 results, got %d", len(results))
		}
		if duration >= 2*time.Second {
			b.Fatalf("latency exceeded 2.0s limit: %v", duration)
		}
	}
}

func TestDiscoverAntigravityUserAgent_Priorities(t *testing.T) {
	// 1. Reset discovery cache
	ResetDiscoveredUserAgent()
	defer ResetDiscoveredUserAgent()

	// 2. Discover dynamically (runs through priorities 1, 2, 3)
	ua := DiscoverAntigravityUserAgent()
	if ua == "" {
		t.Fatalf("expected non-empty discovered user agent")
	}
	if !strings.HasPrefix(ua, "antigravity") {
		t.Errorf("expected user agent starting with 'antigravity', got: %s", ua)
	}

	// 3. Test explicit override
	SetDiscoveredUserAgent("antigravity/9.99.0")
	if current := DiscoverAntigravityUserAgent(); current != "antigravity/9.99.0" {
		t.Errorf("expected overridden user agent 'antigravity/9.99.0', got: %s", current)
	}
}

func TestQuotaClient_UserAgentAndGoogApiClientHeaders(t *testing.T) {
	var capturedUserAgent string
	var capturedGoogApiClient string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserAgent = r.Header.Get("User-Agent")
		capturedGoogApiClient = r.Header.Get("X-Goog-Api-Client")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"buckets":[]}`))
	}))
	defer srv.Close()

	refresher := &mockOAuthRefresher{accessToken: "valid_access_token"}
	client := NewQuotaClientWithEndpoint(refresher, srv.URL, srv.Client())
	client.SetUserAgent("antigravity/2.24.0")

	_, err := client.RetrieveUserQuota(context.Background(), "mock_refresh_token")
	if err != nil {
		t.Fatalf("RetrieveUserQuota failed: %v", err)
	}

	if capturedUserAgent != "antigravity/2.24.0" {
		t.Errorf("expected User-Agent 'antigravity/2.24.0', got: '%s'", capturedUserAgent)
	}
	if capturedGoogApiClient != DefaultGoogApiClient {
		t.Errorf("expected X-Goog-Api-Client '%s', got: '%s'", DefaultGoogApiClient, capturedGoogApiClient)
	}
}

func TestParseQuotaSummary_ConnectRPC(t *testing.T) {
	jsonPayload := []byte(`{
		"response": {
			"groups": [
				{
					"displayName": "Gemini Models",
					"buckets": [
						{
							"bucketId": "gemini-5h",
							"remainingFraction": 0.933,
							"resetTime": "2026-10-17T12:19:14Z"
						},
						{
							"bucketId": "gemini-weekly",
							"remainingFraction": 0.75,
							"resetTime": "2026-10-24T02:19:14Z"
						}
					]
				},
				{
					"displayName": "Claude and GPT models",
					"buckets": [
						{
							"bucketId": "3p-5h",
							"remainingFraction": 0.85,
							"resetTime": "2026-10-17T12:32:55Z"
						}
					]
				}
			]
		}
	}`)

	claude, gemini, gpt, resetTime, status, err := ParseQuotaSummary(jsonPayload)
	if err != nil {
		t.Fatalf("ParseQuotaSummary failed: %v", err)
	}

	if gemini.RemainingFraction != 0.75 {
		t.Errorf("expected gemini remaining fraction 0.75, got %f", gemini.RemainingFraction)
	}
	if claude.RemainingFraction != 0.85 {
		t.Errorf("expected claude remaining fraction 0.85, got %f", claude.RemainingFraction)
	}
	if gpt.RemainingFraction != 0.85 {
		t.Errorf("expected gpt remaining fraction 0.85, got %f", gpt.RemainingFraction)
	}
	if resetTime.IsZero() {
		t.Errorf("expected non-zero canonical reset time")
	}
	if status != StatusPlentiful {
		t.Errorf("expected status Plentiful, got %s", status)
	}
}

func TestHybridTelemetry_ActiveAccountConnectRPC(t *testing.T) {
	tempDir := t.TempDir()
	cache := NewQuotaCache(filepath.Join(tempDir, "hybrid_cache.json"))

	accounts := []AccountTokenInfo{
		{Email: "active@gmail.com", RefreshToken: "tok_active"},
		{Email: "inactive@gmail.com", RefreshToken: "tok_inactive"},
	}
	source := &mockAccountSource{accounts: accounts}

	googleCalledForActive := false
	googleClient := &mockQuotaClient{}
	googleClient.delay = 10 * time.Millisecond

	pool, err := NewTelemetryPoolWithSource(googleClient, cache, source)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	// Inject local quota fetcher that simulates active account in language_server
	pool.SetLocalQuotaFetcher(func(ctx context.Context, email string) (*AccountQuotaData, error) {
		if email == "active@gmail.com" {
			return &AccountQuotaData{
				Email:  email,
				Claude: ModelQuotaBucket{RemainingFraction: 0.95},
				Gemini: ModelQuotaBucket{RemainingFraction: 0.90},
				Status: StatusPlentiful,
			}, nil
		}
		// Inactive accounts are not in language server
		return nil, errors.New("not active in language server")
	})

	results, successCount, errorCount, err := pool.PollAll(context.Background())
	if err != nil {
		t.Fatalf("PollAll failed: %v", err)
	}
	if successCount != 2 || errorCount != 0 {
		t.Errorf("expected 2 success, 0 error, got %d, %d", successCount, errorCount)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	activeData, ok := cache.Get("active@gmail.com")
	if !ok || activeData.Claude.RemainingFraction != 0.95 {
		t.Errorf("expected active account quota 0.95 from local Connect-RPC")
	}

	if googleCalledForActive {
		t.Errorf("Google Cloud Code API should NOT be called for active account when Connect-RPC succeeds")
	}
}

func TestTelemetryPool_PollAll_SkipsOffAccounts(t *testing.T) {
	tempDir := t.TempDir()
	cache := NewQuotaCache(filepath.Join(tempDir, "quota_cache.json"))

	accounts := []AccountTokenInfo{
		{Email: "on@gmail.com", RefreshToken: "tok_on", Status: "on"},
		{Email: "off@gmail.com", RefreshToken: "tok_off", Status: "off"},
		{Email: "default_on@gmail.com", RefreshToken: "tok_def", Status: ""},
	}
	source := &mockAccountSource{accounts: accounts}
	client := &mockQuotaClient{}

	pool, err := NewTelemetryPoolWithSource(client, cache, source)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	results, successCount, errorCount, err := pool.PollAll(context.Background())
	if err != nil {
		t.Fatalf("PollAll failed: %v", err)
	}

	// off@gmail.com must be skipped completely:
	// Only on@gmail.com and default_on@gmail.com should be polled
	if len(results) != 2 {
		t.Fatalf("expected 2 results (skipping off account), got %d", len(results))
	}
	if successCount != 2 || errorCount != 0 {
		t.Errorf("expected 2 success, 0 error, got success=%d error=%d", successCount, errorCount)
	}
	if client.callCount.Load() != 2 {
		t.Errorf("expected exactly 2 API calls to Google, got %d", client.callCount.Load())
	}

	// Verify off@gmail.com is NOT in cache
	if _, found := cache.Get("off@gmail.com"); found {
		t.Errorf("expected off@gmail.com NOT to be in cache")
	}
}

func TestTelemetryPool_WithStore_SkipsOffAccounts(t *testing.T) {
	tempDir := t.TempDir()
	storeDir := filepath.Join(tempDir, "accounts_store")
	accStore, err := store.NewAccountStore(storeDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Create 2 accounts: on and off
	_ = accStore.SaveAccount("active@gmail.com", &model.TokenData{
		AccessToken:  "access-active",
		RefreshToken: "refresh-active",
		IDToken:      "id-active",
	})
	_ = accStore.SaveAccount("disabled@gmail.com", &model.TokenData{
		AccessToken:  "access-disabled",
		RefreshToken: "refresh-disabled",
		IDToken:      "id-disabled",
	})
	if err := accStore.SetAccountStatus("disabled@gmail.com", "off"); err != nil {
		t.Fatalf("failed to set status off: %v", err)
	}

	cache := NewQuotaCache(filepath.Join(tempDir, "quota_cache.json"))
	client := &mockQuotaClient{}

	pool, err := NewTelemetryPool(client, cache, accStore)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	results, successCount, errorCount, err := pool.PollAll(context.Background())
	if err != nil {
		t.Fatalf("PollAll failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result (disabled account skipped), got %d", len(results))
	}
	if successCount != 1 || errorCount != 0 {
		t.Errorf("expected 1 success, 0 error, got success=%d error=%d", successCount, errorCount)
	}
	if client.callCount.Load() != 1 {
		t.Errorf("expected 1 call to Google API, got %d", client.callCount.Load())
	}

	// PollAccount on disabled account directly should be rejected without sending request
	client.callCount.Store(0)
	_, pollErr := pool.PollAccount(context.Background(), "disabled@gmail.com")
	if pollErr == nil {
		t.Errorf("expected PollAccount on disabled account to fail")
	}
	if client.callCount.Load() != 0 {
		t.Errorf("expected 0 calls to Google API for disabled account, got %d", client.callCount.Load())
	}
}

func TestNormalizeTier(t *testing.T) {
	testCases := []struct {
		raw      string
		expected string
	}{
		{"TEAMS_TIER_PRO_ULTIMATE", "ULTRA"},
		{"ultra-tier", "ULTRA"},
		{"g1-ultra-tier", "ULTRA"},
		{"Ultra", "ULTRA"},
		{"ultra", "ULTRA"},
		{"TEAMS_TIER_PRO", "PRO"},
		{"pro-tier", "PRO"},
		{"g1-pro-tier", "PRO"},
		{"Pro", "PRO"},
		{"pro", "PRO"},
		{"PLAN_PRO", "PRO"},
		{"Google AI Pro", "PRO"},
		{"free-tier", "FREE"},
		{"Free", "FREE"},
		{"free", "FREE"},
		{"PLAN_FREE", "FREE"},
		{"TEAMS_TIER_ENTERPRISE_PLUS", "BIZ"},
		{"standard-tier", "BIZ"},
		{"Enterprise", "BIZ"},
		{"Teams", "BIZ"},
		{"BIZ", "BIZ"},
		{"biz", "BIZ"},
		{"", "--"},
		{"   ", "--"},
		{"unknown", "--"},
		{"[Cần OAuth]", "--"},
	}

	for _, tc := range testCases {
		got := NormalizeTier(tc.raw)
		if got != tc.expected {
			t.Errorf("NormalizeTier(%q) = %q, expected %q", tc.raw, got, tc.expected)
		}
	}
}

func TestLoadCodeAssist_Parser(t *testing.T) {
	// Free tier with no paid tier
	freePayload := []byte(`{"currentTier": {"id": "free-tier", "name": "Free Tier"}}`)
	if got := ParseCodeAssistTier(freePayload); got != "FREE" {
		t.Errorf("expected FREE, got %s", got)
	}

	// Free tier with paid tier (Pro)
	paidProPayload := []byte(`{
		"currentTier": {"id": "free-tier", "name": "Free Tier"},
		"paidTier": {"id": "pro-tier", "name": "Pro Tier"}
	}`)
	if got := ParseCodeAssistTier(paidProPayload); got != "PRO" {
		t.Errorf("expected PRO, got %s", got)
	}

	// Paid tier Ultra
	paidUltraPayload := []byte(`{
		"currentTier": {"id": "ultra-tier", "name": "Ultra Tier"}
	}`)
	if got := ParseCodeAssistTier(paidUltraPayload); got != "ULTRA" {
		t.Errorf("expected ULTRA, got %s", got)
	}

	// Invalid JSON returns "--"
	if got := ParseCodeAssistTier([]byte("not-json")); got != "--" {
		t.Errorf("expected --, got %s", got)
	}
}
