package automation

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"antigravity-switch/internal/connector"
	"antigravity-switch/internal/manager"
	"antigravity-switch/internal/model"
	"antigravity-switch/internal/reloader"
	"antigravity-switch/internal/store"
	"antigravity-switch/internal/telemetry"
)

// -----------------------------------------------------------------------------
// MockLangServer: Simulates Antigravity Language Server with Connect-RPC endpoints
// -----------------------------------------------------------------------------

type MockLangServer struct {
	t             *testing.T
	server        *httptest.Server
	port          int
	csrfToken     string
	mu            sync.Mutex
	modelConfig   connector.CascadeModelConfigData
	retryCalls    atomic.Int32
	lastRetryReq  []byte
	streamClients map[chan connector.AgentStateUpdate]struct{}
}

func NewMockLangServer(t *testing.T) *MockLangServer {
	t.Helper()
	mls := &MockLangServer{
		t:             t,
		csrfToken:     "mock-ls-csrf-token",
		streamClients: make(map[chan connector.AgentStateUpdate]struct{}),
		modelConfig: connector.CascadeModelConfigData{
			ClientModelConfigs: []connector.ClientModelConfig{
				{
					ModelOrAlias: connector.ModelOrAlias{Model: "gemini-2.5-pro", Alias: connector.DefaultActiveAgentModel},
					Label:        "Gemini 2.5 Pro",
					Disabled:     false,
					IsSelected:   true,
				},
				{
					ModelOrAlias: connector.ModelOrAlias{Model: "claude-3-5-sonnet", Alias: "claude-sonnet"},
					Label:        "Claude 3.5 Sonnet",
					Disabled:     false,
				},
			},
		},
	}

	mux := http.NewServeMux()

	// 1. Cascade Model Config endpoint
	mux.HandleFunc(connector.EndpointCascadeModelConfigData, func(w http.ResponseWriter, r *http.Request) {
		mls.mu.Lock()
		cfg := mls.modelConfig
		mls.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(cfg)
	})

	// 2. StreamAgentStateUpdates Connect-RPC endpoint
	mux.HandleFunc(connector.EndpointStreamAgentStateUpdates, func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/connect+json")
		w.Header().Set("Connect-Protocol-Version", "1")
		w.WriteHeader(http.StatusOK)
		flusher.Flush()

		ch := make(chan connector.AgentStateUpdate, 32)
		mls.mu.Lock()
		mls.streamClients[ch] = struct{}{}
		mls.mu.Unlock()

		defer func() {
			mls.mu.Lock()
			delete(mls.streamClients, ch)
			mls.mu.Unlock()
		}()

		for {
			select {
			case <-r.Context().Done():
				return
			case update, open := <-ch:
				if !open {
					return
				}
				payload, err := json.Marshal(update)
				if err != nil {
					continue
				}
				frame := connector.EncodeConnectFrame(payload, false)
				if _, writeErr := w.Write(frame); writeErr != nil {
					return
				}
				flusher.Flush()
			}
		}
	})

	// 3. SendUserCascadeMessage (Empty-items retry endpoint)
	mux.HandleFunc(connector.EndpointSendUserCascadeMessage, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mls.mu.Lock()
		mls.lastRetryReq = body
		mls.mu.Unlock()
		mls.retryCalls.Add(1)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	server := httptest.NewServer(mux)
	mls.server = server

	u, _ := url.Parse(server.URL)
	port, _ := strconv.Atoi(u.Port())
	mls.port = port

	return mls
}

func (mls *MockLangServer) ServerInfo() *connector.ServerInfo {
	return &connector.ServerInfo{
		PID:       99999,
		Port:      mls.port,
		CSRFToken: mls.csrfToken,
	}
}

func (mls *MockLangServer) NewClient() *connector.Client {
	return connector.NewClientWithHTTP(mls.ServerInfo(), mls.server.Client())
}

func (mls *MockLangServer) SetModelConfig(cfg connector.CascadeModelConfigData) {
	mls.mu.Lock()
	defer mls.mu.Unlock()
	mls.modelConfig = cfg
}

func (mls *MockLangServer) SendStreamUpdate(update connector.AgentStateUpdate) {
	mls.mu.Lock()
	defer mls.mu.Unlock()
	for ch := range mls.streamClients {
		select {
		case ch <- update:
		default:
		}
	}
}

func (mls *MockLangServer) GetRetryCalls() int {
	return int(mls.retryCalls.Load())
}

func (mls *MockLangServer) ResetRetryCalls() {
	mls.retryCalls.Store(0)
}

func (mls *MockLangServer) Close() {
	mls.mu.Lock()
	for ch := range mls.streamClients {
		close(ch)
		delete(mls.streamClients, ch)
	}
	mls.mu.Unlock()
	if mls.server != nil {
		mls.server.Close()
	}
}

// -----------------------------------------------------------------------------
// MockTelemetry: Simulates Google Cloud Code Quota Summary endpoint
// -----------------------------------------------------------------------------

type MockTelemetry struct {
	t             *testing.T
	server        *httptest.Server
	mu            sync.Mutex
	accountJSON   map[string][]byte
	accountStatus map[string]int
	pollCounts    map[string]int
}

func NewMockTelemetry(t *testing.T) *MockTelemetry {
	t.Helper()
	mt := &MockTelemetry{
		t:             t,
		accountJSON:   make(map[string][]byte),
		accountStatus: make(map[string]int),
		pollCounts:    make(map[string]int),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1internal:retrieveUserQuotaSummary", func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		token := strings.TrimPrefix(authHeader, "Bearer ")
		token = strings.TrimSpace(token)

		mt.mu.Lock()
		mt.pollCounts[token]++
		status, hasStatus := mt.accountStatus[token]
		payload, hasPayload := mt.accountJSON[token]
		mt.mu.Unlock()

		if hasStatus && status != http.StatusOK {
			http.Error(w, fmt.Sprintf("simulated error %d", status), status)
			return
		}

		if !hasPayload {
			// Default healthy quota payload if not explicitly set
			payload = buildQuotaSummaryJSON(0.80, 0.50, false, 0.80, 0.50, false)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	})

	server := httptest.NewServer(mux)
	mt.server = server
	return mt
}

func buildQuotaSummaryJSON(gemini5h, geminiWeekly float64, gemini5hDisabled bool, claude5h, claudeWeekly float64, claude5hDisabled bool) []byte {
	geminiBuckets := []map[string]interface{}{}
	if gemini5h >= 0 {
		geminiBuckets = append(geminiBuckets, map[string]interface{}{
			"bucketId":          "gemini-5h",
			"window":            "5h",
			"remainingFraction": gemini5h,
			"disabled":          gemini5hDisabled,
		})
	}
	if geminiWeekly >= 0 {
		geminiBuckets = append(geminiBuckets, map[string]interface{}{
			"bucketId":          "gemini-weekly",
			"window":            "weekly",
			"remainingFraction": geminiWeekly,
		})
	}

	claudeBuckets := []map[string]interface{}{}
	if claude5h >= 0 {
		claudeBuckets = append(claudeBuckets, map[string]interface{}{
			"bucketId":          "3p-5h",
			"window":            "5h",
			"remainingFraction": claude5h,
			"disabled":          claude5hDisabled,
		})
	}
	if claudeWeekly >= 0 {
		claudeBuckets = append(claudeBuckets, map[string]interface{}{
			"bucketId":          "3p-weekly",
			"window":            "weekly",
			"remainingFraction": claudeWeekly,
		})
	}

	doc := map[string]interface{}{
		"response": map[string]interface{}{
			"groups": []map[string]interface{}{
				{
					"displayName": "Gemini Models",
					"buckets":     geminiBuckets,
				},
				{
					"displayName": "Claude and GPT models",
					"buckets":     claudeBuckets,
				},
			},
		},
	}

	bytes, _ := json.Marshal(doc)
	return bytes
}

func (mt *MockTelemetry) SetAccountQuota(tokenOrEmail string, gemini5h, geminiWeekly float64, gemini5hDisabled bool, claude5h, claudeWeekly float64, claude5hDisabled bool) {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	mt.accountJSON[tokenOrEmail] = buildQuotaSummaryJSON(gemini5h, geminiWeekly, gemini5hDisabled, claude5h, claudeWeekly, claude5hDisabled)
	mt.accountStatus[tokenOrEmail] = http.StatusOK
}

func (mt *MockTelemetry) SetAccountError(tokenOrEmail string, statusCode int) {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	mt.accountStatus[tokenOrEmail] = statusCode
}

func (mt *MockTelemetry) GetPollCount(tokenOrEmail string) int {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	return mt.pollCounts[tokenOrEmail]
}

func (mt *MockTelemetry) URL() string {
	return mt.server.URL + "/v1internal:retrieveUserQuotaSummary"
}

func (mt *MockTelemetry) Client() *http.Client {
	return mt.server.Client()
}

func (mt *MockTelemetry) Close() {
	if mt.server != nil {
		mt.server.Close()
	}
}

type syntheticOAuthRefresher struct{}

func (syntheticOAuthRefresher) RefreshToken(ctx context.Context, refreshToken string) (*connector.TokenRefreshResult, error) {
	// Pass through refreshToken as synthetic access token so MockTelemetry can identify caller
	return &connector.TokenRefreshResult{
		AccessToken: refreshToken,
		Expiry:      time.Now().Add(1 * time.Hour),
	}, nil
}

func (mt *MockTelemetry) NewQuotaClient() *telemetry.QuotaClient {
	return telemetry.NewQuotaClientWithEndpoint(
		syntheticOAuthRefresher{},
		mt.URL(),
		mt.Client(),
	)
}

// -----------------------------------------------------------------------------
// E2ETestHarness: Orchestrates hermetic end-to-end testing with real components
// -----------------------------------------------------------------------------

type E2ETestHarness struct {
	t            *testing.T
	LangServer   *MockLangServer
	Telemetry    *MockTelemetry
	Store        *store.AccountStore
	Manager      *manager.AccountManager
	Pool         *telemetry.TelemetryPool
	Cache        *telemetry.QuotaCache
	Gate         *SwitchGate
	Controller   *AutomationController
	CurrentEmail string
	emailMu      sync.Mutex
}

func NewE2ETestHarness(t *testing.T, activeEmail string, otherAccounts []string) *E2ETestHarness {
	t.Helper()
	tempDir := t.TempDir()

	mls := NewMockLangServer(t)
	mt := NewMockTelemetry(t)

	accStore, err := store.NewAccountStore(filepath.Join(tempDir, "accounts_store"))
	if err != nil {
		t.Fatalf("failed initializing test store: %v", err)
	}

	harness := &E2ETestHarness{
		t:            t,
		LangServer:   mls,
		Telemetry:    mt,
		Store:        accStore,
		CurrentEmail: activeEmail,
	}

	// Save active account
	if activeEmail != "" {
		_ = accStore.SaveAccount(activeEmail, &model.TokenData{
			AccessToken:  "access-" + activeEmail,
			RefreshToken: "refresh-" + activeEmail,
			IDToken:      makeTestJWT(activeEmail),
		})
	}

	// Save candidate accounts
	for _, email := range otherAccounts {
		_ = accStore.SaveAccount(email, &model.TokenData{
			AccessToken:  "access-" + email,
			RefreshToken: "refresh-" + email,
			IDToken:      makeTestJWT(email),
		})
	}

	rel := reloader.NewSessionReloader()
	rel.ProcessKiller = func(ctx context.Context) error { return nil }
	rel.Sleeper = func(ctx context.Context, d time.Duration) error { return nil }
	rel.ContinuationRunner = func(ctx context.Context, client *connector.Client, activeThreads []reloader.ActiveThread, promptText string) error {
		return nil
	}
	rel.ServerDiscoverer = func() (*connector.ServerInfo, error) {
		return mls.ServerInfo(), nil
	}

	mgr, err := manager.NewAccountManager(accStore, rel)
	if err != nil {
		t.Fatalf("failed initializing AccountManager: %v", err)
	}

	mgr.SetWincredHandlers(
		func() (*model.TokenData, error) {
			harness.emailMu.Lock()
			em := harness.CurrentEmail
			harness.emailMu.Unlock()
			return &model.TokenData{
				AccessToken:  "access-" + em,
				RefreshToken: "refresh-" + em,
				IDToken:      makeTestJWT(em),
			}, nil
		},
		func(token *model.TokenData) error {
			if token != nil {
				if email := manager.ExtractEmailFromToken(token); email != "" {
					harness.emailMu.Lock()
					harness.CurrentEmail = email
					harness.emailMu.Unlock()
				}
			}
			return nil
		},
	)
	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return mls.NewClient(), nil
	})
	harness.Manager = mgr

	cachePath := filepath.Join(tempDir, "quota_cache.json")
	cache := telemetry.NewQuotaCache(cachePath)
	harness.Cache = cache

	quotaClient := mt.NewQuotaClient()
	pool, err := telemetry.NewTelemetryPool(quotaClient, cache, accStore)
	if err != nil {
		t.Fatalf("failed initializing TelemetryPool: %v", err)
	}
	// Direct pool queries to MockTelemetry
	pool.SetLocalQuotaFetcher(nil)
	harness.Pool = pool

	gateName := fmt.Sprintf("Local\\E2ETestGate_%d", time.Now().UnixNano())
	gate := NewSwitchGate(gateName)
	harness.Gate = gate

	ctrl := NewAutomationController(mgr, pool, cache, gate)
	ctrl.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return mls.NewClient(), nil
	})

	rc := ctrl.RetryCoordinator()
	if rc != nil {
		rc.SetConversationsScanner(func(ctx context.Context) ([]string, error) {
			return []string{"conv-e2e-1"}, nil
		})
		rc.SetSleeper(func(ctx context.Context, d time.Duration) error {
			return nil // Fast sleep in tests
		})
	}
	harness.Controller = ctrl

	return harness
}

func (h *E2ETestHarness) GetActiveEmail() string {
	h.emailMu.Lock()
	defer h.emailMu.Unlock()
	return h.CurrentEmail
}

func (h *E2ETestHarness) Close() {
	if h.Controller != nil {
		h.Controller.Stop()
	}
	if h.LangServer != nil {
		h.LangServer.Close()
	}
	if h.Telemetry != nil {
		h.Telemetry.Close()
	}
}

// -----------------------------------------------------------------------------
// Task 1.1: Verification Condition
// -----------------------------------------------------------------------------

func TestMockHarnessInit(t *testing.T) {
	harness := NewE2ETestHarness(t, "active@example.com", []string{"cand1@example.com", "cand2@example.com"})
	defer harness.Close()

	if harness.Controller == nil {
		t.Fatal("expected non-nil AutomationController")
	}
	if harness.LangServer == nil {
		t.Fatal("expected non-nil MockLangServer")
	}
	if harness.Telemetry == nil {
		t.Fatal("expected non-nil MockTelemetry")
	}

	// 1. Verify MockLangServer returns dynamic model config
	client := harness.LangServer.NewClient()
	modelID, label, disabled, err := client.ResolveActiveModel(context.Background(), "")
	if err != nil {
		t.Fatalf("ResolveActiveModel failed: %v", err)
	}
	if label != "Gemini 2.5 Pro" || disabled {
		t.Errorf("unexpected resolved model: model=%s label=%s disabled=%v", modelID, label, disabled)
	}

	// 2. Verify MockTelemetry returns parsed quota through real QuotaClient & TelemetryPool
	harness.Telemetry.SetAccountQuota("refresh-cand1@example.com", 0.75, 0.60, false, 0.40, 0.30, false)
	qData, pollErr := harness.Pool.PollAccount(context.Background(), "cand1@example.com")
	if pollErr != nil {
		t.Fatalf("PollAccount failed: %v", pollErr)
	}
	if qData == nil {
		t.Fatal("expected non-nil AccountQuotaData from pool")
	}
	if qData.Gemini.FiveHourFraction != 0.75 || qData.Gemini.WeeklyFraction != 0.60 {
		t.Errorf("unexpected quota data: Gemini 5h=%f, weekly=%f", qData.Gemini.FiveHourFraction, qData.Gemini.WeeklyFraction)
	}

	// 3. Verify MockLangServer empty items retry call tracking
	retryErr := client.SendEmptyItemsRetry(context.Background(), "test-conv-id", "gemini-2.5-pro", nil)
	if retryErr != nil {
		t.Fatalf("SendEmptyItemsRetry failed: %v", retryErr)
	}
	if harness.LangServer.GetRetryCalls() != 1 {
		t.Errorf("expected 1 retry call recorded, got %d", harness.LangServer.GetRetryCalls())
	}
}



