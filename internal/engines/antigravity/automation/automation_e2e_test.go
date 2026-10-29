package automation

import (
	"context"
	"encoding/json"
	"errors"
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

	"antigravity-switch/internal/engines/antigravity/connector"
	"antigravity-switch/internal/engines/antigravity/manager"
	"antigravity-switch/internal/engines/antigravity/model"
	"antigravity-switch/internal/engines/antigravity/reloader"
	"antigravity-switch/internal/engines/antigravity/store"
	"antigravity-switch/internal/engines/antigravity/telemetry"
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
	emailGetter   func() string
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
		if r.Header.Get("x-codeium-csrf-token") != mls.csrfToken {
			http.Error(w, "missing CSRF", 401)
			return
		}
		frame, err := connector.ReadConnectFrame(r.Body)
		if err != nil {
			http.Error(w, "invalid envelope", 400)
			return
		}
		var request map[string]any
		if json.Unmarshal(frame.Payload, &request) != nil || request["initialStepsPageBounds"] == nil {
			http.Error(w, "invalid bounds", 400)
			return
		}
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
				payload, err := json.Marshal(nativeUpdateFixture(update))
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

	// 4. GetUserStatus endpoint (for AccountManager identity verification)
	userStatusHandler := func(w http.ResponseWriter, r *http.Request) {
		mls.mu.Lock()
		fn := mls.emailGetter
		mls.mu.Unlock()
		email := ""
		if fn != nil {
			email = fn()
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"email": email,
			"name":  "Test User",
		})
	}
	mux.HandleFunc(connector.EndpointGetUserStatus, userStatusHandler)
	mux.HandleFunc("/GetUserStatus", userStatusHandler)
	mux.HandleFunc("/user_status", userStatusHandler)

	server := httptest.NewServer(mux)
	mls.server = server

	u, _ := url.Parse(server.URL)
	port, _ := strconv.Atoi(u.Port())
	mls.port = port

	return mls
}

func (mls *MockLangServer) SetEmailGetter(fn func() string) {
	mls.mu.Lock()
	defer mls.mu.Unlock()
	mls.emailGetter = fn
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
	qc := telemetry.NewQuotaClientWithEndpoint(
		syntheticOAuthRefresher{},
		mt.URL(),
		mt.Client(),
	)
	qc.SetUserAgent("antigravity/2.24.0")
	telemetry.SetDiscoveredUserAgent("antigravity/2.24.0")
	return qc
}

// E2ETestHarness: Orchestrates hermetic end-to-end testing with real components
// -----------------------------------------------------------------------------

type E2ETestHarness struct {
	Reloader     *reloader.SessionReloader
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
	mls.SetEmailGetter(func() string { return harness.GetActiveEmail() })

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
	rel.StoragePath = filepath.Join(tempDir, "app_storage.json")
	rel.SnapshotCapturer = func(string, string, string) (string, []reloader.ActiveThread, error) { return "", nil, nil }
	rel.ProcessKiller = func(ctx context.Context) error { return nil }
	rel.Sleeper = func(ctx context.Context, d time.Duration) error { return nil }
	rel.ContinuationRunner = func(ctx context.Context, client *connector.Client, activeThreads []reloader.ActiveThread, promptText string) error {
		return nil
	}
	rel.ServerDiscoverer = func() (*connector.ServerInfo, error) {
		return mls.ServerInfo(), nil
	}

	harness.Reloader = rel
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
	mgr.SetOAuthRefresher(syntheticOAuthRefresher{})
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
	ctrl.SetPbtxtPath(filepath.Join(tempDir, "missing.pbtxt"))
	ctrl.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return mls.NewClient(), nil
	})

	rc := ctrl.RetryCoordinator()
	if rc != nil {
		rc.SetModelName("gemini-2.5-pro")
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
	modelID, label, disabled, err := client.ResolveActiveModel(context.Background(), filepath.Join(t.TempDir(), "missing.pbtxt"))
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

// -----------------------------------------------------------------------------
// Task 1.2: Verification Condition - Edge Cases 1 to 7
// -----------------------------------------------------------------------------

func TestEdgeCases1To7(t *testing.T) {
	// Case 1: Gemini đang dùng, 5h <= 2% -> Chuyển một lần
	t.Run("EdgeCase1_Gemini5hLow_SwitchesOnce", func(t *testing.T) {
		harness := NewE2ETestHarness(t, "active@example.com", []string{"cand1@example.com"})
		defer harness.Close()

		ctx := context.Background()
		harness.Controller.SetEnabled(true)

		harness.Telemetry.SetAccountQuota("refresh-active@example.com", 0.015, 0.50, false, 0.80, 0.50, false)
		harness.Telemetry.SetAccountQuota("refresh-cand1@example.com", 0.80, 0.70, false, 0.80, 0.70, false)

		// Seed cache
		qActive, _ := harness.Pool.PollAccount(ctx, "active@example.com")
		_, _ = harness.Pool.PollAccount(ctx, "cand1@example.com")

		decision, err := harness.Controller.EvaluateProactiveTrigger(ctx, []*telemetry.AccountQuotaData{qActive})
		if err != nil {
			t.Fatalf("EvaluateProactiveTrigger failed: %v", err)
		}
		if !decision.ShouldSwitch {
			t.Fatalf("expected ShouldSwitch=true for 5h <= 2%%, got false: %s", decision.Reason)
		}

		res, err := harness.Controller.TriggerSwitch(ctx, decision.Reason)
		if err != nil {
			t.Fatalf("TriggerSwitch failed: %v", err)
		}
		if res == nil || !res.Success {
			t.Fatalf("expected successful switch, got: %+v", res)
		}
		if harness.GetActiveEmail() != "cand1@example.com" {
			t.Fatalf("expected active email switched to cand1@example.com, got %s", harness.GetActiveEmail())
		}
	})

	// Case 2: Claude tuần = 0, phiên dùng Gemini còn quota -> Không chuyển do Claude
	t.Run("EdgeCase2_ClaudeWeeklyDepleted_GeminiActive_NoSwitch", func(t *testing.T) {
		harness := NewE2ETestHarness(t, "active@example.com", []string{"cand1@example.com"})
		defer harness.Close()

		ctx := context.Background()
		// Active model is Gemini; Claude weekly is 0%, but Gemini has 5h=80%, weekly=50%
		harness.Telemetry.SetAccountQuota("refresh-active@example.com", 0.80, 0.50, false, 0.0, 0.0, false)
		qActive, _ := harness.Pool.PollAccount(ctx, "active@example.com")

		harness.Controller.SetEnabled(true)
		decision, err := harness.Controller.EvaluateProactiveTrigger(ctx, []*telemetry.AccountQuotaData{qActive})
		if err != nil {
			t.Fatalf("EvaluateProactiveTrigger failed: %v", err)
		}
		if decision.ShouldSwitch {
			t.Fatalf("expected ShouldSwitch=false because Gemini is active and healthy, got true: %s", decision.Reason)
		}
		if harness.GetActiveEmail() != "active@example.com" {
			t.Fatalf("expected active email unchanged, got %s", harness.GetActiveEmail())
		}
	})

	// Case 3: Chỉ tuần, tuần > 1% (ví dụ: tuần = 2%) -> Dùng tiếp
	t.Run("EdgeCase3_WeeklyOnly_Above1Percent_Continues", func(t *testing.T) {
		harness := NewE2ETestHarness(t, "active@example.com", []string{"cand1@example.com"})
		defer harness.Close()

		ctx := context.Background()
		// Only weekly bucket present, 5h is missing (-1), weekly = 0.02 (2% > 1%)
		harness.Telemetry.SetAccountQuota("refresh-active@example.com", -1, 0.02, false, -1, 0.50, false)
		qActive, _ := harness.Pool.PollAccount(ctx, "active@example.com")

		harness.Controller.SetEnabled(true)
		decision, err := harness.Controller.EvaluateProactiveTrigger(ctx, []*telemetry.AccountQuotaData{qActive})
		if err != nil {
			t.Fatalf("EvaluateProactiveTrigger failed: %v", err)
		}
		if decision.ShouldSwitch {
			t.Fatalf("expected ShouldSwitch=false for weekly=1%% > 0%%, got true: %s", decision.Reason)
		}
		if !decision.Evaluation.IsEligible {
			t.Fatalf("expected IsEligible=true for weekly=1%% > 0%%, got false")
		}
		if harness.GetActiveEmail() != "active@example.com" {
			t.Fatalf("expected active email unchanged, got %s", harness.GetActiveEmail())
		}
	})

	// Case 4: Chỉ tuần, tuần <= 1% (ví dụ: tuần = 1% hoặc 0%) -> Chuyển
	t.Run("EdgeCase4_WeeklyOnly_AtOrBelow1Percent_Switches", func(t *testing.T) {
		harness := NewE2ETestHarness(t, "active@example.com", []string{"cand1@example.com"})
		defer harness.Close()

		ctx := context.Background()
		harness.Controller.SetEnabled(true)

		// Only weekly bucket present, 5h is missing (-1), weekly = 0.01 (1% <= 1%)
		harness.Telemetry.SetAccountQuota("refresh-active@example.com", -1, 0.01, false, -1, 0.50, false)
		harness.Telemetry.SetAccountQuota("refresh-cand1@example.com", 0.80, 0.60, false, 0.80, 0.60, false)

		qActive, _ := harness.Pool.PollAccount(ctx, "active@example.com")
		_, _ = harness.Pool.PollAccount(ctx, "cand1@example.com")

		decision, err := harness.Controller.EvaluateProactiveTrigger(ctx, []*telemetry.AccountQuotaData{qActive})
		if err != nil {
			t.Fatalf("EvaluateProactiveTrigger failed: %v", err)
		}
		if !decision.ShouldSwitch {
			t.Fatalf("expected ShouldSwitch=true for weekly=1%% (<= 1%%), got false: %s", decision.Reason)
		}

		res, err := harness.Controller.TriggerSwitch(ctx, decision.Reason)
		if err != nil {
			t.Fatalf("TriggerSwitch failed: %v", err)
		}
		if res == nil || !res.Success {
			t.Fatalf("expected successful switch, got: %+v", res)
		}
		if harness.GetActiveEmail() != "cand1@example.com" {
			t.Fatalf("expected active email switched to cand1@example.com, got %s", harness.GetActiveEmail())
		}
	})

	// Case 5: API bucket 5h disabled -> Cờ đi qua parser/cache, bỏ qua 5h và xét tuần
	t.Run("EdgeCase5_Disabled5hBucket_Ignored_EvaluatesWeekly", func(t *testing.T) {
		harness := NewE2ETestHarness(t, "active@example.com", []string{"cand1@example.com"})
		defer harness.Close()

		ctx := context.Background()
		// 5h bucket disabled: true, remainingFraction: 0.01 (1%), but Weekly: 0.50 (50%)
		harness.Telemetry.SetAccountQuota("refresh-active@example.com", 0.01, 0.50, true, 0.40, 0.60, false)
		qActive, _ := harness.Pool.PollAccount(ctx, "active@example.com")

		if !qActive.Gemini.FiveHourDisabled {
			t.Fatalf("expected Gemini 5h bucket to have FiveHourDisabled=true")
		}

		harness.Controller.SetEnabled(true)
		decision, err := harness.Controller.EvaluateProactiveTrigger(ctx, []*telemetry.AccountQuotaData{qActive})
		if err != nil {
			t.Fatalf("EvaluateProactiveTrigger failed: %v", err)
		}
		if decision.ShouldSwitch {
			t.Fatalf("expected ShouldSwitch=false because 5h is disabled and weekly=50%% > 0%%, got true: %s", decision.Reason)
		}
		if harness.GetActiveEmail() != "active@example.com" {
			t.Fatalf("expected active email unchanged, got %s", harness.GetActiveEmail())
		}
	})

	// Case 6: Recovery 1 hoặc 2 kết thúc không lỗi -> Dừng recovery, không switch
	t.Run("EdgeCase6_RecoveryOutcomeSuccess_Resets_NoSwitch", func(t *testing.T) {
		harness := NewE2ETestHarness(t, "active@example.com", []string{"cand1@example.com"})
		defer harness.Close()

		ctx := context.Background()
		harness.Controller.SetEnabled(true)
		_ = harness.Controller.Start(ctx)

		rc := harness.Controller.RetryCoordinator()
		if rc == nil {
			t.Fatal("expected non-nil RetryCoordinator")
		}

		// 1. First error: Rate limit
		update1 := connector.AgentStateUpdate{
			ConversationID:    "conv-e2e-1",
			ExecutionID:       "exec-6-1",
			StepID:            "step-1",
			Status:            "IDLE",
			TerminationReason: "ERROR",
			LastStepError:     &connector.StepError{Code: "RESOURCE_EXHAUSTED"},
			Attempt:           1,
		}
		if err := rc.HandleAgentStateUpdate(ctx, update1); err != nil {
			t.Fatalf("HandleAgentStateUpdate err: %v", err)
		}
		if !rc.IsRecovering() || rc.GetRecoveryAttempts() != 1 {
			t.Fatalf("expected recovering=true, attempts=1, got recovering=%v attempts=%d", rc.IsRecovering(), rc.GetRecoveryAttempts())
		}

		// Wait briefly for asynchronous retry dispatch
		time.Sleep(50 * time.Millisecond)
		if harness.LangServer.GetRetryCalls() < 1 {
			t.Errorf("expected empty-items retry sent to Language Server, got %d", harness.LangServer.GetRetryCalls())
		}

		// 2. Recovery succeeds on next turn (IsSuccessfulCompletion)
		updateSuccess := connector.AgentStateUpdate{
			ConversationID:    "conv-e2e-1",
			ExecutionID:       "exec-6-2",
			StepID:            "step-2",
			Status:            "IDLE",
			TerminationReason: "COMPLETED",
			Attempt:           2,
		}
		if err := rc.HandleAgentStateUpdate(ctx, updateSuccess); err != nil {
			t.Fatalf("HandleAgentStateUpdate success err: %v", err)
		}

		// 3. Assert recovery state is cleared and no switch was performed
		if rc.IsRecovering() || rc.GetRecoveryAttempts() != 0 {
			t.Fatalf("expected recovery cleared, got recovering=%v attempts=%d", rc.IsRecovering(), rc.GetRecoveryAttempts())
		}
		if harness.GetActiveEmail() != "active@example.com" {
			t.Fatalf("expected active email unchanged, got %s", harness.GetActiveEmail())
		}
	})

	// Case 7: Ba recovery actions kết thúc lỗi giới hạn -> Một quyết định switch
	t.Run("EdgeCase7_ThreeRecoveryFailures_TriggersOneSwitch", func(t *testing.T) {
		harness := NewE2ETestHarness(t, "active@example.com", []string{"cand1@example.com"})
		defer harness.Close()

		ctx := context.Background()
		harness.Telemetry.SetAccountQuota("refresh-cand1@example.com", 0.90, 0.80, false, 0.90, 0.80, false)
		_, _ = harness.Pool.PollAccount(ctx, "cand1@example.com")

		harness.Controller.SetEnabled(true)
		_ = harness.Controller.Start(ctx)

		rc := harness.Controller.RetryCoordinator()

		// Attempt 1 fails with rate limit
		_ = rc.HandleAgentStateUpdate(ctx, connector.AgentStateUpdate{
			ConversationID:    "conv-e2e-1",
			ExecutionID:       "exec-7-1",
			StepID:            "step-1",
			Status:            "IDLE",
			TerminationReason: "ERROR",
			LastStepError:     &connector.StepError{Code: "RESOURCE_EXHAUSTED"},
			Attempt:           1,
		})
		if rc.GetRecoveryAttempts() != 1 {
			t.Fatalf("expected attempt 1, got %d", rc.GetRecoveryAttempts())
		}

		waitUntil(t, func() bool { return harness.LangServer.GetRetryCalls() >= 1 })
		// Attempt 2 fails with rate limit
		_ = rc.HandleAgentStateUpdate(ctx, connector.AgentStateUpdate{
			ConversationID:    "conv-e2e-1",
			ExecutionID:       "exec-7-2",
			StepID:            "step-2",
			Status:            "IDLE",
			TerminationReason: "ERROR",
			LastStepError:     &connector.StepError{Code: "RESOURCE_EXHAUSTED"},
			Attempt:           2,
		})
		if rc.GetRecoveryAttempts() != 2 {
			t.Fatalf("expected attempt 2, got %d", rc.GetRecoveryAttempts())
		}

		waitUntil(t, func() bool { return harness.LangServer.GetRetryCalls() >= 2 })
		// Attempt 3 fails with rate limit
		_ = rc.HandleAgentStateUpdate(ctx, connector.AgentStateUpdate{
			ConversationID:    "conv-e2e-1",
			ExecutionID:       "exec-7-3",
			StepID:            "step-3",
			Status:            "IDLE",
			TerminationReason: "ERROR",
			LastStepError:     &connector.StepError{Code: "RESOURCE_EXHAUSTED"},
			Attempt:           3,
		})
		if rc.GetRecoveryAttempts() != 3 {
			t.Fatalf("expected attempt 3, got %d", rc.GetRecoveryAttempts())
		}

		waitUntil(t, func() bool { return harness.LangServer.GetRetryCalls() >= 3 })
		// Attempt 4 triggers switch because 3 retries are exhausted
		err := rc.HandleAgentStateUpdate(ctx, connector.AgentStateUpdate{
			ConversationID:    "conv-e2e-1",
			ExecutionID:       "exec-7-4",
			StepID:            "step-4",
			Status:            "IDLE",
			TerminationReason: "ERROR",
			LastStepError:     &connector.StepError{Code: "RESOURCE_EXHAUSTED"},
			Attempt:           4,
		})
		if err != ErrMaxRetriesExceeded {
			t.Fatalf("expected ErrMaxRetriesExceeded, got %v", err)
		}

		// Credential mutation precedes timeline/continuation completion. Wait
		// for the whole operation before closing fixtures and deleting TempDir.
		deadline := time.Now().Add(2 * time.Second)
		switched := false
		for time.Now().Before(deadline) {
			if harness.GetActiveEmail() == "cand1@example.com" && !harness.Controller.IsSwitching() {
				switched = true
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		if !switched {
			t.Fatalf("expected account to switch to cand1@example.com after 3 recovery failures, current: %s", harness.GetActiveEmail())
		}
		switches := 0
		for len(harness.Controller.statusChan) > 0 {
			if (<-harness.Controller.statusChan).Code == StatusSwitched {
				switches++
			}
		}
		if switches != 1 || harness.LangServer.GetRetryCalls() != MaxRecoveryAttempts {
			t.Fatalf("expected three retries and one completed switch, got retries=%d switches=%d", harness.LangServer.GetRetryCalls(), switches)
		}
	})
}

// -----------------------------------------------------------------------------
// Task 1.3: Verification Condition - Edge Cases 8 to 14 & Binding Assertions
// -----------------------------------------------------------------------------

func TestEdgeCases8To14(t *testing.T) {
	// Case 8: Proactive và reactive cùng kích hoạt -> Một đợt; không phát retry tiếp sau switch
	t.Run("EdgeCase8_ProactiveSupersedesReactive_NoDuplicateRetry", func(t *testing.T) {
		harness := NewE2ETestHarness(t, "active@example.com", []string{"cand1@example.com"})
		defer harness.Close()

		ctx := context.Background()
		harness.Telemetry.SetAccountQuota("refresh-cand1@example.com", 0.90, 0.80, false, 0.90, 0.80, false)
		_, _ = harness.Pool.PollAccount(ctx, "cand1@example.com")

		harness.Controller.SetEnabled(true)
		_ = harness.Controller.Start(ctx)

		rc := harness.Controller.RetryCoordinator()

		// 1. Reactive trigger starts recovery attempt 1
		_ = rc.HandleAgentStateUpdate(ctx, connector.AgentStateUpdate{
			ConversationID:    "conv-e2e-1",
			ExecutionID:       "exec-8-1",
			StepID:            "step-1",
			Status:            "IDLE",
			TerminationReason: "ERROR",
			LastStepError:     &connector.StepError{Code: "RESOURCE_EXHAUSTED"},
			Attempt:           1,
		})
		if !rc.IsRecovering() || rc.GetRecoveryAttempts() != 1 {
			t.Fatalf("expected recovering=true, attempts=1, got recovering=%v attempts=%d", rc.IsRecovering(), rc.GetRecoveryAttempts())
		}

		initialRetries := harness.LangServer.GetRetryCalls()

		// 2. Proactive trigger fires concurrently and supersedes reactive recovery
		rc.SupersedeByProactive(ctx, "proactive quota exhausted during retry")

		// Assert recovery is immediately reset
		if rc.IsRecovering() || rc.GetRecoveryAttempts() != 0 {
			t.Fatalf("expected recovery reset by proactive trigger, got recovering=%v attempts=%d", rc.IsRecovering(), rc.GetRecoveryAttempts())
		}

		// Wait for switch to complete
		deadline := time.Now().Add(2 * time.Second)
		switched := false
		for time.Now().Before(deadline) {
			if harness.GetActiveEmail() == "cand1@example.com" {
				switched = true
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		if !switched {
			t.Fatalf("expected active email switched to cand1@example.com, got %s", harness.GetActiveEmail())
		}

		// 3. Confirm no subsequent native retries were dispatched after switch decision
		time.Sleep(100 * time.Millisecond)
		afterRetries := harness.LangServer.GetRetryCalls()
		if afterRetries > initialRetries+1 {
			t.Errorf("expected no additional retries dispatched after proactive switch, initial=%d, after=%d", initialRetries, afterRetries)
		}
	})

	// Case 9: Manual GUI/CLI và auto cùng yêu cầu -> Mutual exclusion qua SwitchGate
	t.Run("EdgeCase9_ManualAndAutoContention_TwoWay", func(t *testing.T) {
		harness := NewE2ETestHarness(t, "active@example.com", []string{"cand1@example.com"})
		defer harness.Close()

		ctx := context.Background()
		harness.Controller.SetEnabled(true)

		// 9a: Manual-first contention: manual holds gate, TryExecute fails with ErrGateBusy
		// and TriggerSwitch with bounded timeout cannot enter concurrently
		manualHolding := make(chan struct{})
		manualRelease := make(chan struct{})

		go func() {
			_ = harness.Gate.Execute(ctx, func(wasAbandoned bool) error {
				close(manualHolding)
				<-manualRelease
				return nil
			})
		}()

		<-manualHolding

		// Manual switch holds gate -> TryExecute fails with ErrGateBusy
		executed, tryErr := harness.Gate.TryExecute(func(wasAbandoned bool) error {
			return nil
		})
		if executed || tryErr != ErrGateBusy {
			t.Errorf("expected TryExecute to fail with ErrGateBusy when manual holds gate, got executed=%v, err=%v", executed, tryErr)
		}

		// Auto trigger rejects contention instead of queuing stale intent
		ctxShort, cancelShort := context.WithTimeout(ctx, 100*time.Millisecond)
		_, autoErr := harness.Controller.TriggerSwitch(ctxShort, "auto contention test")
		cancelShort()
		if autoErr == nil || !errors.Is(autoErr, ErrGateBusy) {
			t.Errorf("expected ErrGateBusy while manual holds gate, got: %v", autoErr)
		}

		close(manualRelease)
		time.Sleep(50 * time.Millisecond)

		// 9b: Auto-first contention: auto switch holds the gate, manual TryExecute fails with ErrGateBusy
		autoHolding := make(chan struct{})
		autoRelease := make(chan struct{})

		go func() {
			_ = harness.Gate.Execute(ctx, func(wasAbandoned bool) error {
				close(autoHolding)
				<-autoRelease
				return nil
			})
		}()

		<-autoHolding

		executedManual, manualErr := harness.Gate.TryExecute(func(wasAbandoned bool) error {
			return nil
		})
		if executedManual || manualErr != ErrGateBusy {
			t.Errorf("expected manual TryExecute to fail with ErrGateBusy while auto holds gate, got executed=%v err=%v", executedManual, manualErr)
		}

		close(autoRelease)
	})

	// Case 10: Model đổi trước switch -> Resolve và tính lại nhóm/ứng viên
	t.Run("EdgeCase10_ModelChangedBeforeSwitch_ReranksCandidates", func(t *testing.T) {
		harness := NewE2ETestHarness(t, "active@example.com", []string{"cand_a@example.com", "cand_b@example.com"})
		defer harness.Close()

		ctx := context.Background()
		harness.Controller.SetEnabled(true)

		// Active account has depleted quota for both models (5h = 1%)
		harness.Telemetry.SetAccountQuota("refresh-active@example.com", 0.01, 0.50, false, 0.01, 0.50, false)

		// cand_a has Gemini Weekly=20% (burn-down first for Gemini), Claude Weekly=90%
		harness.Telemetry.SetAccountQuota("refresh-cand_a@example.com", 0.80, 0.20, false, 0.80, 0.90, false)
		// cand_b has Gemini Weekly=90%, Claude Weekly=20% (burn-down first for Claude)
		harness.Telemetry.SetAccountQuota("refresh-cand_b@example.com", 0.80, 0.90, false, 0.80, 0.20, false)

		_, _ = harness.Pool.PollAccount(ctx, "active@example.com")
		_, _ = harness.Pool.PollAccount(ctx, "cand_a@example.com")
		_, _ = harness.Pool.PollAccount(ctx, "cand_b@example.com")

		// Language Server model config is dynamically set to Gemini 2.5 Pro
		harness.LangServer.SetModelConfig(connector.CascadeModelConfigData{
			ClientModelConfigs: []connector.ClientModelConfig{
				{
					ModelOrAlias: connector.ModelOrAlias{Model: "gemini-2.5-pro", Alias: connector.DefaultActiveAgentModel},
					Label:        "Gemini 2.5 Pro",
					Disabled:     false,
					IsSelected:   true,
				},
			},
		})

		// Trigger switch - should dynamically rank cand_a first because active model is Gemini and cand_a has lower weekly (20% vs 90%)
		res, err := harness.Controller.TriggerSwitch(ctx, "model changed to Gemini test")
		if err != nil {
			t.Fatalf("TriggerSwitch failed: %v", err)
		}
		if res == nil || !res.Success {
			t.Fatalf("expected successful switch, got: %+v", res)
		}
		if harness.GetActiveEmail() != "cand_a@example.com" {
			t.Fatalf("expected switch to cand_a@example.com based on active Gemini model, got %s", harness.GetActiveEmail())
		}
	})

	// Case 11: Preflight ứng viên đầu lỗi -> Bỏ qua trong đợt, xét kế tiếp trước khi thay credential
	t.Run("EdgeCase11_PreflightCandidate1Fails_SkipsToCandidate2", func(t *testing.T) {
		harness := NewE2ETestHarness(t, "active@example.com", []string{"cand1_broken@example.com", "cand2_healthy@example.com"})
		defer harness.Close()

		ctx := context.Background()
		harness.Controller.SetEnabled(true)

		harness.Telemetry.SetAccountQuota("refresh-active@example.com", 0.01, 0.50, false, 0.80, 0.50, false)
		// Pre-seed cache so cand1 looks better than cand2 initially
		harness.Telemetry.SetAccountQuota("refresh-cand1_broken@example.com", 0.95, 0.90, false, 0.95, 0.90, false)
		harness.Telemetry.SetAccountQuota("refresh-cand2_healthy@example.com", 0.70, 0.60, false, 0.70, 0.60, false)

		_, _ = harness.Pool.PollAccount(ctx, "active@example.com")
		_, _ = harness.Pool.PollAccount(ctx, "cand1_broken@example.com")
		_, _ = harness.Pool.PollAccount(ctx, "cand2_healthy@example.com")

		// Now simulate cand1 token dying (preflight failure with HTTP 401 or 500)
		harness.Telemetry.SetAccountError("refresh-cand1_broken@example.com", http.StatusInternalServerError)

		res, err := harness.Controller.TriggerSwitch(ctx, "preflight fallback test")
		if err != nil {
			t.Fatalf("TriggerSwitch failed: %v", err)
		}
		if res == nil || !res.Success {
			t.Fatalf("expected successful switch to candidate 2, got: %+v", res)
		}
		if harness.GetActiveEmail() != "cand2_healthy@example.com" {
			t.Fatalf("expected candidate 1 skipped and switched to cand2_healthy@example.com, got %s", harness.GetActiveEmail())
		}
	})

	// Case 12: SwitchAccount trả lỗi -> Dừng đợt, hiện lỗi, không xoay vô điều kiện
	t.Run("EdgeCase12_SwitchAccountError_HaltsWithoutIndefiniteRotation", func(t *testing.T) {
		harness := NewE2ETestHarness(t, "active@example.com", []string{"cand1@example.com", "cand2@example.com"})
		defer harness.Close()

		ctx := context.Background()
		harness.Controller.SetEnabled(true)

		harness.Telemetry.SetAccountQuota("refresh-active@example.com", 0.01, 0.50, false, 0.80, 0.50, false)
		harness.Telemetry.SetAccountQuota("refresh-cand1@example.com", 0.80, 0.70, false, 0.80, 0.70, false)
		harness.Telemetry.SetAccountQuota("refresh-cand2@example.com", 0.80, 0.70, false, 0.80, 0.70, false)

		_, _ = harness.Pool.PollAccount(ctx, "active@example.com")
		_, _ = harness.Pool.PollAccount(ctx, "cand1@example.com")
		_, _ = harness.Pool.PollAccount(ctx, "cand2@example.com")

		// Inject SwitchAccount failure by failing wincredWriter
		harness.Manager.SetWincredHandlers(
			func() (*model.TokenData, error) {
				return &model.TokenData{
					AccessToken:  "access-active@example.com",
					RefreshToken: "refresh-active@example.com",
					IDToken:      makeTestJWT("active@example.com"),
				}, nil
			},
			func(token *model.TokenData) error {
				return errors.New("simulated wincred write hardware error")
			},
		)

		res, err := harness.Controller.TriggerSwitch(ctx, "switch error test")
		if err == nil {
			t.Fatalf("expected TriggerSwitch to return error when SwitchAccount fails, got res: %+v", res)
		}
		if !strings.Contains(err.Error(), "simulated wincred write hardware error") {
			t.Errorf("expected wincred write error in error string, got: %v", err)
		}

		// Verify it stopped after cand1 and did not switch to cand2
		if harness.GetActiveEmail() != "active@example.com" {
			t.Errorf("expected active account to remain active@example.com, got %s", harness.GetActiveEmail())
		}
	})

	// Case 13: Không có ứng viên -> Hiện lý do, chờ telemetry hiện có
	t.Run("EdgeCase13_NoCandidates_EmitsStatusNoCandidate", func(t *testing.T) {
		harness := NewE2ETestHarness(t, "active@example.com", []string{"depleted1@example.com"})
		defer harness.Close()

		ctx := context.Background()
		harness.Controller.SetEnabled(true)

		harness.Telemetry.SetAccountQuota("refresh-active@example.com", 0.01, 0.50, false, 0.80, 0.50, false)
		// depleted1 has 0% weekly quota (ineligible)
		harness.Telemetry.SetAccountQuota("refresh-depleted1@example.com", 0.00, 0.00, false, 0.00, 0.00, false)

		_, _ = harness.Pool.PollAccount(ctx, "active@example.com")
		_, _ = harness.Pool.PollAccount(ctx, "depleted1@example.com")

		res, err := harness.Controller.TriggerSwitch(ctx, "no candidates test")
		if !errors.Is(err, ErrNoCandidates) {
			t.Fatalf("expected ErrNoCandidates, got res=%+v err=%v", res, err)
		}
		if harness.GetActiveEmail() != "active@example.com" {
			t.Errorf("expected active email unchanged, got %s", harness.GetActiveEmail())
		}
	})

	// Case 14: Event cũ/trùng tới muộn -> Không tăng retry hoặc switch vì event cũ
	t.Run("EdgeCase14_StaleDuplicateEvents_DroppedByDeduplication", func(t *testing.T) {
		harness := NewE2ETestHarness(t, "active@example.com", []string{"cand1@example.com"})
		defer harness.Close()

		ctx := context.Background()
		harness.Controller.SetEnabled(true)
		_ = harness.Controller.Start(ctx)

		rc := harness.Controller.RetryCoordinator()

		staleUpdate := connector.AgentStateUpdate{
			ConversationID:    "conv-e2e-1",
			ExecutionID:       "exec-14-1",
			StepID:            "step-1",
			Status:            "IDLE",
			TerminationReason: "ERROR",
			LastStepError:     &connector.StepError{Code: "RESOURCE_EXHAUSTED"},
			Attempt:           1,
		}

		// First delivery -> increments recovery attempt to 1
		err1 := rc.HandleAgentStateUpdate(ctx, staleUpdate)
		if err1 != nil {
			t.Fatalf("first delivery error: %v", err1)
		}
		if rc.GetRecoveryAttempts() != 1 {
			t.Fatalf("expected recovery attempts=1, got %d", rc.GetRecoveryAttempts())
		}

		// Duplicate delivery (same execution/step/attempt) -> dropped cleanly
		err2 := rc.HandleAgentStateUpdate(ctx, staleUpdate)
		if err2 != nil {
			t.Fatalf("duplicate delivery error: %v", err2)
		}
		if rc.GetRecoveryAttempts() != 1 {
			t.Errorf("expected recovery attempts to remain 1 on duplicate event, got %d", rc.GetRecoveryAttempts())
		}

		// Third delivery of same event -> also dropped
		err3 := rc.HandleAgentStateUpdate(ctx, staleUpdate)
		if err3 != nil {
			t.Fatalf("triplicate delivery error: %v", err3)
		}
		if rc.GetRecoveryAttempts() != 1 {
			t.Errorf("expected recovery attempts to remain 1, got %d", rc.GetRecoveryAttempts())
		}
	})
}

// -----------------------------------------------------------------------------
// Binding Assertions (P1-C to P2-A integration verification)
// -----------------------------------------------------------------------------

func TestBindings_HTTPAcceptance_Versus_ExecutionOutcome(t *testing.T) {
	harness := NewE2ETestHarness(t, "active@example.com", nil)
	defer harness.Close()

	ctx := context.Background()
	harness.Controller.SetEnabled(true)
	_ = harness.Controller.Start(ctx)
	client := harness.LangServer.NewClient()

	// 1. SendEmptyItemsRetry returns HTTP 200 OK
	err := client.SendEmptyItemsRetry(ctx, "conv-1", "gemini-2.5-pro", nil)
	if err != nil {
		t.Fatalf("SendEmptyItemsRetry returned error: %v", err)
	}

	// 2. Assert HTTP 200 OK does NOT mean recovery succeeded; RetryCoordinator remains recovering
	rc := harness.Controller.RetryCoordinator()
	_ = rc.HandleAgentStateUpdate(ctx, connector.AgentStateUpdate{
		ConversationID:    "conv-1",
		ExecutionID:       "exec-binding-1",
		StepID:            "step-1",
		Status:            "IDLE",
		TerminationReason: "ERROR",
		LastStepError:     &connector.StepError{Code: "RESOURCE_EXHAUSTED"},
		Attempt:           1,
	})

	if !rc.IsRecovering() {
		t.Errorf("expected coordinator to be recovering while awaiting execution outcome")
	}

	// 3. Only when actual execution outcome arrives does recovery complete
	_ = rc.HandleAgentStateUpdate(ctx, connector.AgentStateUpdate{
		ConversationID:    "conv-1",
		ExecutionID:       "exec-binding-2",
		StepID:            "step-2",
		Status:            "IDLE",
		TerminationReason: "COMPLETED",
		Attempt:           2,
	})

	if rc.IsRecovering() || rc.GetRecoveryAttempts() != 0 {
		t.Errorf("expected recovery completed only after execution outcome COMPLETED, got recovering=%v attempts=%d", rc.IsRecovering(), rc.GetRecoveryAttempts())
	}
}

func TestBindings_NativeRetryWaiting_HasRetryAt(t *testing.T) {
	harness := NewE2ETestHarness(t, "active@example.com", nil)
	defer harness.Close()

	ctx := context.Background()
	harness.Controller.SetEnabled(true)
	_ = harness.Controller.Start(ctx)
	rc := harness.Controller.RetryCoordinator()

	// Language Server scheduled internal auto-retry
	retryAtUpdate := connector.AgentStateUpdate{
		ConversationID: "conv-1",
		ExecutionID:    "exec-retryat-1",
		StepID:         "step-1",
		Status:         "RUNNING",
		RetryAt:        "2026-10-20T17:30:00Z",
	}

	err := rc.HandleAgentStateUpdate(ctx, retryAtUpdate)
	if err != nil {
		t.Fatalf("HandleAgentStateUpdate error: %v", err)
	}

	// Does NOT count as a failed recovery attempt
	if rc.GetRecoveryAttempts() != 0 {
		t.Errorf("expected 0 recovery attempts when server is auto-retrying, got %d", rc.GetRecoveryAttempts())
	}
}

func waitUntil(t *testing.T, ready func() bool) {
	t.Helper()
	end := time.Now().Add(3 * time.Second)
	for time.Now().Before(end) {
		if ready() {
			return
		}
		time.Sleep(time.Millisecond * 5)
	}
	t.Fatal("condition did not become true")
}
func nativeUpdateFixture(u connector.AgentStateUpdate) map[string]any {
	return map[string]any{"update": map[string]any{"conversationId": u.ConversationID, "status": "CASCADE_RUN_STATUS_" + u.Status,
		"executorMetadata":     map[string]any{"executionId": u.ExecutionID, "terminationReason": "EXECUTOR_TERMINATION_REASON_" + u.TerminationReason},
		"mainTrajectoryUpdate": map[string]any{"lastStepError": u.LastStepError, "stepsUpdate": map[string]any{"totalLength": 2}}}}
}
