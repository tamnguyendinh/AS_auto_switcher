package automation

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
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

type mockQuotaClient struct {
	failEmails map[string]bool
}

func (m *mockQuotaClient) RetrieveUserQuota(ctx context.Context, refreshToken string) (*telemetry.AccountQuotaData, error) {
	if m.failEmails != nil && m.failEmails[refreshToken] {
		return nil, errors.New("simulated preflight token expired")
	}
	return &telemetry.AccountQuotaData{
		Email:  "test@example.com",
		Status: telemetry.StatusPlentiful,
		Gemini: telemetry.ModelQuotaBucket{
			ModelID:          "gemini-2.5-pro",
			FiveHourFraction: 0.80,
			WeeklyFraction:   0.40,
		},
	}, nil
}

func makeTestJWT(email string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"email":"%s"}`, email)))
	return fmt.Sprintf("%s.%s.sig", header, payload)
}

func setupTestEnvironment(t *testing.T, activeEmail string, otherAccounts map[string]float64) (*manager.AccountManager, *telemetry.QuotaCache, *store.AccountStore) {
	t.Helper()
	storeDir := t.TempDir()
	accStore, err := store.NewAccountStore(storeDir)
	if err != nil {
		t.Fatalf("failed creating store: %v", err)
	}

	// Save active account
	_ = accStore.SaveAccount(activeEmail, &model.TokenData{
		AccessToken:  "token-" + activeEmail,
		RefreshToken: "refresh-" + activeEmail,
		IDToken:      makeTestJWT(activeEmail),
	})

	cache := telemetry.NewQuotaCache(t.TempDir() + "/cache.json")

	// Save other accounts
	for email := range otherAccounts {
		_ = accStore.SaveAccount(email, &model.TokenData{
			AccessToken:  "token-" + email,
			RefreshToken: "refresh-" + email,
			IDToken:      makeTestJWT(email),
		})
	}

	rel := reloader.NewSessionReloader()
	rel.StoragePath = filepath.Join(t.TempDir(), "app_storage.json")
	rel.SnapshotCapturer = func(string, string, string) (string, []reloader.ActiveThread, error) { return "", nil, nil }
	rel.ProcessKiller = func(ctx context.Context) error { return nil }
	rel.Sleeper = func(ctx context.Context, d time.Duration) error { return nil }
	rel.ContinuationRunner = func(ctx context.Context, client *connector.Client, activeThreads []reloader.ActiveThread, promptText string) error {
		return nil
	}
	rel.ServerDiscoverer = func() (*connector.ServerInfo, error) {
		return &connector.ServerInfo{PID: 1234, Port: 54321, CSRFToken: "test_csrf"}, nil
	}

	mgr, err := manager.NewAccountManager(accStore, rel)
	if err != nil {
		t.Fatalf("failed creating manager: %v", err)
	}

	mgr.SetOAuthRefresher(syntheticOAuthRefresher{})
	var identityMu sync.Mutex
	currentEmail := activeEmail
	mgr.SetWincredHandlers(
		func() (*model.TokenData, error) {
			identityMu.Lock()
			defer identityMu.Unlock()
			return &model.TokenData{
				AccessToken:  "access-" + currentEmail,
				RefreshToken: "refresh-" + currentEmail,
				IDToken:      makeTestJWT(currentEmail),
			}, nil
		},
		func(token *model.TokenData) error {
			identityMu.Lock()
			defer identityMu.Unlock()
			if token != nil {
				if email := manager.ExtractEmailFromToken(token); email != "" {
					currentEmail = email
				}
			}
			return nil
		},
	)

	return mgr, cache, accStore
}

func TestAutomationController_LifecycleAndEnableToggle(t *testing.T) {
	mgr, cache, _ := setupTestEnvironment(t, "user@example.com", nil)
	gate := NewSwitchGate(fmt.Sprintf("Local\\TestCtrl_Lifecycle_%d", time.Now().UnixNano()))
	ctrl := NewAutomationController(mgr, nil, cache, gate)
	ctrl.SetPbtxtPath(filepath.Join(t.TempDir(), "missing.pbtxt"))
	ctrl.RetryCoordinator().SetConversationsScanner(func(context.Context) ([]string, error) { return nil, nil })

	if ctrl.IsEnabled() {
		t.Errorf("expected controller initially disabled")
	}

	ctx := context.Background()
	if err := ctrl.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer ctrl.Stop()

	// Toggle ON
	ctrl.SetEnabled(true)
	if !ctrl.IsEnabled() {
		t.Errorf("expected IsEnabled=true after SetEnabled(true)")
	}

	// Read event from StatusChannel
	select {
	case evt := <-ctrl.StatusChannel():
		if evt.Code != StatusIdle {
			t.Errorf("expected StatusIdle event on toggle ON, got %s", evt.Code)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timeout waiting for toggle status event")
	}

	// Toggle OFF
	ctrl.SetEnabled(false)
	if ctrl.IsEnabled() {
		t.Errorf("expected IsEnabled=false after SetEnabled(false)")
	}

	ctrl.Stop()
}

func TestAutomationController_EvaluateProactiveTrigger(t *testing.T) {
	activeEmail := "active@example.com"
	mgr, cache, _ := setupTestEnvironment(t, activeEmail, nil)
	gate := NewSwitchGate(fmt.Sprintf("Local\\TestCtrl_Eval_%d", time.Now().UnixNano()))
	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) { return nil, nil })
	ctrl := NewAutomationController(mgr, nil, cache, gate)
	ctrl.SetPbtxtPath(filepath.Join(t.TempDir(), "missing.pbtxt"))
	ctrl.RetryCoordinator().SetConversationsScanner(func(context.Context) ([]string, error) { return nil, nil })
	ctrl.SetEnabled(true)

	// Injected client finder returning Gemini model
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == connector.EndpointCascadeModelConfigData {
			resp := connector.CascadeModelConfigData{
				ClientModelConfigs: []connector.ClientModelConfig{
					{
						ModelOrAlias: connector.ModelOrAlias{Model: "gemini-2.5-pro"},
						Label:        "Gemini 2.5 Pro",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	u, _ := url.Parse(server.URL)
	port, _ := strconv.Atoi(u.Port())
	ctrl.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return connector.NewClientWithHTTP(&connector.ServerInfo{
			PID:       1234,
			Port:      port,
			CSRFToken: "test_tok",
		}, server.Client()), nil
	})

	ctx := context.Background()

	// Case 1: Gemini 5h <= 2% -> ShouldSwitch = true
	quota1 := []*telemetry.AccountQuotaData{
		{
			Email: activeEmail,
			Gemini: telemetry.ModelQuotaBucket{
				ModelID:          "gemini-2.5-pro",
				FiveHourFraction: 0.015, // 1.5% <= 2%
				WeeklyFraction:   0.80,
			},
		},
	}
	dec1, err := ctrl.EvaluateProactiveTrigger(ctx, quota1)
	if err != nil {
		t.Fatalf("eval 1 failed: %v", err)
	}
	if !dec1.ShouldSwitch {
		t.Errorf("expected ShouldSwitch=true when 5h <= 2%%")
	}

	// Case 2: Claude week = 0, but session uses Gemini with 80% quota -> ShouldSwitch = false
	quota2 := []*telemetry.AccountQuotaData{
		{
			Email: activeEmail,
			Gemini: telemetry.ModelQuotaBucket{
				ModelID:          "gemini-2.5-pro",
				FiveHourFraction: 0.80,
				WeeklyFraction:   0.80,
			},
			Claude: telemetry.ModelQuotaBucket{
				ModelID:        "claude-3-5-sonnet",
				WeeklyFraction: 0.0, // Claude depleted!
			},
		},
	}
	dec2, err := ctrl.EvaluateProactiveTrigger(ctx, quota2)
	if err != nil {
		t.Fatalf("eval 2 failed: %v", err)
	}
	if dec2.ShouldSwitch {
		t.Errorf("expected ShouldSwitch=false when Claude is 0%% but session model is Gemini")
	}

	// Case 3: Weekly-only account (5h disabled), weekly = 1% -> ShouldSwitch = false
	quota3 := []*telemetry.AccountQuotaData{
		{
			Email: activeEmail,
			Gemini: telemetry.ModelQuotaBucket{
				ModelID:          "gemini-2.5-pro",
				FiveHourDisabled: true,
				WeeklyFraction:   0.02, // 2% > 1%
			},
		},
	}
	dec3, err := ctrl.EvaluateProactiveTrigger(ctx, quota3)
	if err != nil {
		t.Fatalf("eval 3 failed: %v", err)
	}
	if dec3.ShouldSwitch {
		t.Errorf("expected ShouldSwitch=false when weekly=2%% and 5h disabled")
	}

	// Case 4: Weekly-only account, weekly = 0% -> ShouldSwitch = true
	quota4 := []*telemetry.AccountQuotaData{
		{
			Email: activeEmail,
			Gemini: telemetry.ModelQuotaBucket{
				ModelID:          "gemini-2.5-pro",
				FiveHourDisabled: true,
				WeeklyFraction:   0.0, // 0%
			},
		},
	}
	dec4, err := ctrl.EvaluateProactiveTrigger(ctx, quota4)
	if err != nil {
		t.Fatalf("eval 4 failed: %v", err)
	}
	if !dec4.ShouldSwitch {
		t.Errorf("expected ShouldSwitch=true when weekly=0%%")
	}

	// Case 5: Rule 3 - UNKNOWN_UNCERTAIN (status error or missing quota) -> ShouldSwitch = false
	quota5 := []*telemetry.AccountQuotaData{
		{
			Email:        activeEmail,
			Status:       telemetry.StatusError,
			ErrorMessage: "network timeout",
		},
	}
	dec5, err := ctrl.EvaluateProactiveTrigger(ctx, quota5)
	if err != nil {
		t.Fatalf("eval 5 failed: %v", err)
	}
	if dec5.ShouldSwitch {
		t.Errorf("Rule 3 violation: ShouldSwitch must be false on UNKNOWN_UNCERTAIN error")
	}
}

func TestAutomationController_SingleFlightGate_100ConcurrentTriggers(t *testing.T) {
	activeEmail := "active@example.com"
	otherAccounts := map[string]float64{
		"cand1@example.com": 0.50,
	}
	mgr, cache, _ := setupTestEnvironment(t, activeEmail, otherAccounts)

	// Populate cache with healthy quota for candidate
	cache.Update(&telemetry.AccountQuotaData{
		Email: "cand1@example.com",
		Gemini: telemetry.ModelQuotaBucket{
			ModelID:          "gemini-2.5-pro",
			FiveHourFraction: 0.80,
			WeeklyFraction:   0.50,
		},
	})

	gate := NewSwitchGate(fmt.Sprintf("Local\\TestCtrl_100Concurrent_%d", time.Now().UnixNano()))
	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) { return nil, nil })
	ctrl := NewAutomationController(mgr, nil, cache, gate)
	ctrl.SetPbtxtPath(filepath.Join(t.TempDir(), "missing.pbtxt"))
	ctrl.RetryCoordinator().SetConversationsScanner(func(context.Context) ([]string, error) { return nil, nil })
	ctrl.SetEnabled(true)

	// Mock language server for switch
	var currentActiveEmail atomic.Value
	currentActiveEmail.Store(activeEmail)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == connector.EndpointCascadeModelConfigData {
			_, _ = w.Write([]byte(`{"clientModelConfigs":[{"modelOrAlias":{"model":"gemini-2.5-pro"},"label":"Gemini 2.5 Pro","isSelected":true}]}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		email := currentActiveEmail.Load().(string)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"email":"%s"}`, email)))
	}))
	defer server.Close()

	u, _ := url.Parse(server.URL)
	port, _ := strconv.Atoi(u.Port())
	ctrl.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return connector.NewClientWithHTTP(&connector.ServerInfo{
			PID:       1234,
			Port:      port,
			CSRFToken: "tok",
		}, server.Client()), nil
	})
	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return connector.NewClientWithHTTP(&connector.ServerInfo{
			PID:       1234,
			Port:      port,
			CSRFToken: "tok",
		}, server.Client()), nil
	})

	// When WinCred is written, update currentActiveEmail to match
	mgr.SetWincredHandlers(
		func() (*model.TokenData, error) {
			email := currentActiveEmail.Load().(string)
			return &model.TokenData{
				AccessToken:  "access-" + email,
				RefreshToken: "refresh-" + email,
				IDToken:      makeTestJWT(email),
			}, nil
		},
		func(token *model.TokenData) error {
			if token != nil {
				if email := manager.ExtractEmailFromToken(token); email != "" {
					currentActiveEmail.Store(email)
				}
			}
			return nil
		},
	)

	const concurrentCount = 100
	var successCount atomic.Int32
	var droppedCount atomic.Int32
	var otherErrCount atomic.Int32
	var wg sync.WaitGroup

	startBarrier := make(chan struct{})

	for i := 0; i < concurrentCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-startBarrier
			_, err := ctrl.TriggerSwitch(context.Background(), "100 concurrent test")
			if err == nil {
				successCount.Add(1)
			} else if errors.Is(err, ErrSwitchInProgress) {
				droppedCount.Add(1)
			} else {
				otherErrCount.Add(1)
			}
		}()
	}

	// Release all 100 goroutines simultaneously
	close(startBarrier)
	wg.Wait()

	if successCount.Load() != 1 {
		t.Errorf("expected strictly 1 successful switch, got %d", successCount.Load())
	}
	if droppedCount.Load() != concurrentCount-1 {
		t.Errorf("expected %d dropped executions by single-flight gate, got %d", concurrentCount-1, droppedCount.Load())
	}
	if otherErrCount.Load() != 0 {
		t.Errorf("unexpected other errors: %d", otherErrCount.Load())
	}
}

func TestAutomationController_CandidatePreflightAndFallback(t *testing.T) {
	activeEmail := "active@example.com"
	otherAccounts := map[string]float64{
		"cand-fail@example.com": 0.30,
		"cand-succ@example.com": 0.40,
	}
	mgr, cache, accStore := setupTestEnvironment(t, activeEmail, otherAccounts)

	// Populate cache for both
	cache.Update(&telemetry.AccountQuotaData{
		Email: "cand-fail@example.com",
		Gemini: telemetry.ModelQuotaBucket{
			ModelID:          "gemini-2.5-pro",
			FiveHourFraction: 0.80,
			WeeklyFraction:   0.30, // lower weekly -> prioritised first by sorter!
		},
	})
	cache.Update(&telemetry.AccountQuotaData{
		Email: "cand-succ@example.com",
		Gemini: telemetry.ModelQuotaBucket{
			ModelID:          "gemini-2.5-pro",
			FiveHourFraction: 0.80,
			WeeklyFraction:   0.40,
		},
	})

	quotaClient := &mockQuotaClient{
		failEmails: map[string]bool{
			"refresh-cand-fail@example.com": true,
		},
	}
	pool, err := telemetry.NewTelemetryPool(quotaClient, cache, accStore)
	if err != nil {
		t.Fatalf("failed creating TelemetryPool: %v", err)
	}

	gate := NewSwitchGate(fmt.Sprintf("Local\\TestCtrl_Preflight_%d", time.Now().UnixNano()))
	ctrl := NewAutomationController(mgr, pool, cache, gate)
	ctrl.SetPbtxtPath(filepath.Join(t.TempDir(), "missing.pbtxt"))
	ctrl.RetryCoordinator().SetConversationsScanner(func(context.Context) ([]string, error) { return nil, nil })
	ctrl.SetEnabled(true)

	// Mock language server for switch
	var currentActiveEmail atomic.Value
	currentActiveEmail.Store(activeEmail)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == connector.EndpointCascadeModelConfigData {
			_, _ = w.Write([]byte(`{"clientModelConfigs":[{"modelOrAlias":{"model":"gemini-2.5-pro"},"label":"Gemini 2.5 Pro","isSelected":true}]}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		email := currentActiveEmail.Load().(string)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"email":"%s"}`, email)))
	}))
	defer server.Close()

	u, _ := url.Parse(server.URL)
	port, _ := strconv.Atoi(u.Port())
	ctrl.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return connector.NewClientWithHTTP(&connector.ServerInfo{
			PID:       1234,
			Port:      port,
			CSRFToken: "tok",
		}, server.Client()), nil
	})
	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return connector.NewClientWithHTTP(&connector.ServerInfo{
			PID:       1234,
			Port:      port,
			CSRFToken: "tok",
		}, server.Client()), nil
	})

	// When WinCred is written, update currentActiveEmail to match
	mgr.SetWincredHandlers(
		func() (*model.TokenData, error) {
			email := currentActiveEmail.Load().(string)
			return &model.TokenData{
				AccessToken:  "access-" + email,
				RefreshToken: "refresh-" + email,
				IDToken:      makeTestJWT(email),
			}, nil
		},
		func(token *model.TokenData) error {
			if token != nil {
				if email := manager.ExtractEmailFromToken(token); email != "" {
					currentActiveEmail.Store(email)
				}
			}
			return nil
		},
	)

	res, err := ctrl.TriggerSwitch(context.Background(), "preflight test")
	if err != nil {
		t.Fatalf("TriggerSwitch failed: %v", err)
	}

	// Should have switched to candidate 2 (cand-succ) because candidate 1 failed preflight!
	if res.TargetEmail != "cand-succ@example.com" {
		t.Errorf("expected switch to cand-succ@example.com after cand-fail preflight error, got %s", res.TargetEmail)
	}
}

func TestAutomationController_NoCandidates(t *testing.T) {
	activeEmail := "active@example.com"
	mgr, cache, _ := setupTestEnvironment(t, activeEmail, nil)

	gate := NewSwitchGate(fmt.Sprintf("Local\\TestCtrl_NoCand_%d", time.Now().UnixNano()))
	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) { return nil, nil })
	ctrl := NewAutomationController(mgr, nil, cache, gate)
	ctrl.SetPbtxtPath(filepath.Join(t.TempDir(), "missing.pbtxt"))
	ctrl.RetryCoordinator().SetConversationsScanner(func(context.Context) ([]string, error) { return nil, nil })
	ctrl.SetEnabled(true)

	_, err := ctrl.TriggerSwitch(context.Background(), "no cand test")
	if !errors.Is(err, ErrNoCandidates) {
		t.Errorf("expected ErrNoCandidates, got %v", err)
	}
}

func TestAutomationController_NonBlockingStatusChannel(t *testing.T) {
	mgr, cache, _ := setupTestEnvironment(t, "user@example.com", nil)
	gate := NewSwitchGate(fmt.Sprintf("Local\\TestCtrl_NonBlock_%d", time.Now().UnixNano()))
	ctrl := NewAutomationController(mgr, nil, cache, gate)
	ctrl.SetPbtxtPath(filepath.Join(t.TempDir(), "missing.pbtxt"))
	ctrl.RetryCoordinator().SetConversationsScanner(func(context.Context) ([]string, error) { return nil, nil })

	// Rapidly emit 128 status events with NO reader
	done := make(chan struct{})
	go func() {
		for i := 0; i < 128; i++ {
			ctrl.EmitStatus(StatusEvaluating, fmt.Sprintf("Event %d", i), nil)
		}
		close(done)
	}()

	select {
	case <-done:
		// Succeeded without deadlock or blocking
	case <-time.After(1 * time.Second):
		t.Fatalf("EmitStatus blocked when statusChan was full")
	}
}
