package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"antigravity-switch/internal/connector"
	"antigravity-switch/internal/manager"
	"antigravity-switch/internal/model"
	"antigravity-switch/internal/reloader"
	"antigravity-switch/internal/store"
	"antigravity-switch/internal/telemetry"
)

func makeTestJWT(email string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	claims := map[string]interface{}{
		"email": email,
		"exp":   time.Now().Add(1 * time.Hour).Unix(),
	}
	claimsBytes, _ := json.Marshal(claims)
	payload := base64.RawURLEncoding.EncodeToString(claimsBytes)
	signature := base64.RawURLEncoding.EncodeToString([]byte("signature"))
	return header + "." + payload + "." + signature
}

func setupTestAccountManager(t *testing.T, activeEmail string) (*manager.AccountManager, *store.AccountStore) {
	t.Helper()
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "accounts_store")

	accStore, err := store.NewAccountStore(storePath)
	if err != nil {
		t.Fatalf("failed initializing test store: %v", err)
	}

	sessionReloader := reloader.NewSessionReloader()
	sessionReloader.ProcessKiller = func(ctx context.Context) error { return nil }
	sessionReloader.Sleeper = func(ctx context.Context, d time.Duration) error { return nil }
	sessionReloader.ServerDiscoverer = func() (*connector.ServerInfo, error) {
		return &connector.ServerInfo{
			PID:       9999,
			Port:      44444,
			CSRFToken: "test-csrf",
		}, nil
	}

	mgr, err := manager.NewAccountManager(accStore, sessionReloader)
	if err != nil {
		t.Fatalf("failed initializing test manager: %v", err)
	}

	var activeCredToken *model.TokenData
	if activeEmail != "" {
		activeCredToken = &model.TokenData{
			AccessToken: "test_access",
			IDToken:     makeTestJWT(activeEmail),
		}
	}

	mgr.SetWincredHandlers(
		func() (*model.TokenData, error) {
			return activeCredToken, nil
		},
		func(token *model.TokenData) error {
			activeCredToken = token
			return nil
		},
	)

	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return nil, nil
	})

	return mgr, accStore
}

func TestFlagQuotaRegistration(t *testing.T) {
	f := flag.Lookup("quota")
	if f == nil {
		t.Fatalf("expected flag 'quota' to be registered")
	}
	if !strings.Contains(f.Usage, "quota telemetry") {
		t.Errorf("unexpected usage message: %s", f.Usage)
	}
}

func TestRenderQuotaTable_EmptyCache(t *testing.T) {
	mgr, accStore := setupTestAccountManager(t, "alpha@test.com")

	// Save one test account
	acc := &model.TokenData{
		AccessToken:  makeTestJWT("alpha@test.com"),
		RefreshToken: "refresh-token-alpha",
		Expiry:       time.Now().Add(1 * time.Hour),
	}
	if err := accStore.SaveAccount("alpha@test.com", acc); err != nil {
		t.Fatalf("failed saving account: %v", err)
	}

	cache := telemetry.NewQuotaCache(filepath.Join(t.TempDir(), "empty_cache.json"))
	var buf bytes.Buffer
	ctx := context.Background()

	err := renderQuotaTable(&buf, mgr, cache, ctx)
	if err != nil {
		t.Fatalf("renderQuotaTable returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "ANTIGRAVITY MULTI-ACCOUNT QUOTA TELEMETRY") {
		t.Errorf("expected header banner in output:\n%s", out)
	}
	if !strings.Contains(out, "EMAIL") || !strings.Contains(out, "CLAUDE 4.6") || !strings.Contains(out, "GEMINI 2.5") || !strings.Contains(out, "GPT-OSS") || !strings.Contains(out, "RESET TIME (UTC)") || !strings.Contains(out, "HEALTH STATUS") {
		t.Errorf("expected standard table columns in output:\n%s", out)
	}
	if !strings.Contains(out, "alpha@test.com") {
		t.Errorf("expected alpha@test.com in output:\n%s", out)
	}
	if !strings.Contains(out, "N/A") {
		t.Errorf("expected N/A for missing quota in output:\n%s", out)
	}
}

func TestRenderQuotaTable_PopulatedAndStatuses(t *testing.T) {
	mgr, accStore := setupTestAccountManager(t, "active@example.com")

	// Add 3 accounts: 1 active/healthy, 1 ready/healthy, 1 ready/low quota
	acc1 := &model.TokenData{
		AccessToken:  makeTestJWT("active@example.com"),
		RefreshToken: "rt1",
		Expiry:       time.Now().Add(1 * time.Hour),
	}
	acc2 := &model.TokenData{
		AccessToken:  makeTestJWT("ready@example.com"),
		RefreshToken: "rt2",
		Expiry:       time.Now().Add(1 * time.Hour),
	}
	acc3 := &model.TokenData{
		AccessToken:  makeTestJWT("low@example.com"),
		RefreshToken: "rt3",
		Expiry:       time.Now().Add(1 * time.Hour),
	}

	_ = accStore.SaveAccount("active@example.com", acc1)
	_ = accStore.SaveAccount("ready@example.com", acc2)
	_ = accStore.SaveAccount("low@example.com", acc3)

	resetTime := time.Date(2026, 10, 17, 18, 30, 0, 0, time.UTC)
	cache := telemetry.NewQuotaCache(filepath.Join(t.TempDir(), "populated_cache.json"))

	cache.Update(&telemetry.AccountQuotaData{
		Email:       "active@example.com",
		Claude:      telemetry.ModelQuotaBucket{RemainingFraction: 1.0},
		Gemini:      telemetry.ModelQuotaBucket{RemainingFraction: 0.85},
		ResetTime:   resetTime,
		Status:      telemetry.StatusPristine,
		LastUpdated: time.Now().UTC(),
	})

	cache.Update(&telemetry.AccountQuotaData{
		Email:       "ready@example.com",
		Claude:      telemetry.ModelQuotaBucket{RemainingFraction: 0.90},
		Gemini:      telemetry.ModelQuotaBucket{RemainingFraction: 0.70},
		ResetTime:   resetTime,
		Status:      telemetry.StatusPlentiful,
		LastUpdated: time.Now().UTC(),
	})

	cache.Update(&telemetry.AccountQuotaData{
		Email:       "low@example.com",
		Claude:      telemetry.ModelQuotaBucket{RemainingFraction: 0.50},
		Gemini:      telemetry.ModelQuotaBucket{RemainingFraction: 0.08, ResetTime: resetTime},
		ResetTime:   resetTime,
		Status:      telemetry.StatusLowGemini,
		LastUpdated: time.Now().UTC(),
	})

	var buf bytes.Buffer
	ctx := context.Background()

	err := renderQuotaTable(&buf, mgr, cache, ctx)
	if err != nil {
		t.Fatalf("renderQuotaTable returned error: %v", err)
	}

	out := buf.String()

	// Verify columns and data format
	if !strings.Contains(out, "active@example.com") || !strings.Contains(out, "ready@example.com") || !strings.Contains(out, "low@example.com") {
		t.Errorf("missing accounts in output:\n%s", out)
	}
	if !strings.Contains(out, "85.0%") || !strings.Contains(out, "8.0%") {
		t.Errorf("expected percentage values in output:\n%s", out)
	}
	if !strings.Contains(out, "18:30:00 UTC") {
		t.Errorf("expected UTC reset time in output:\n%s", out)
	}
	if !strings.Contains(out, "Active") {
		t.Errorf("expected 'Active' status for active@example.com:\n%s", out)
	}
	if !strings.Contains(out, "Ready") {
		t.Errorf("expected 'Ready' status for ready@example.com:\n%s", out)
	}
	if !strings.Contains(out, "Low Quota") {
		t.Errorf("expected 'Low Quota' status for low@example.com:\n%s", out)
	}
	if !strings.Contains(out, "1 Active, 1 Ready, 1 Low Quota") {
		t.Errorf("expected summary with 1 Active, 1 Ready, 1 Low Quota in output:\n%s", out)
	}
}

func TestRenderQuotaTable_ActiveLowQuotaStatus(t *testing.T) {
	mgr, accStore := setupTestAccountManager(t, "current@example.com")

	acc := &model.TokenData{
		AccessToken:  makeTestJWT("current@example.com"),
		RefreshToken: "rt_curr",
		Expiry:       time.Now().Add(1 * time.Hour),
	}
	_ = accStore.SaveAccount("current@example.com", acc)

	resetTime := time.Date(2026, 10, 17, 21, 0, 0, 0, time.UTC)
	cache := telemetry.NewQuotaCache(filepath.Join(t.TempDir(), "active_low.json"))

	ctx := context.Background()

	cache.Update(&telemetry.AccountQuotaData{
		Email:       "current@example.com",
		Claude:      telemetry.ModelQuotaBucket{RemainingFraction: 0.05, ResetTime: resetTime},
		Gemini:      telemetry.ModelQuotaBucket{RemainingFraction: 0.05, ResetTime: resetTime},
		ResetTime:   resetTime,
		Status:      telemetry.StatusDepleted,
		LastUpdated: time.Now().UTC(),
	})

	var buf bytes.Buffer
	err := renderQuotaTable(&buf, mgr, cache, ctx)
	if err != nil {
		t.Fatalf("renderQuotaTable error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Active (Low Quota)") {
		t.Errorf("expected 'Active (Low Quota)' for depleted active account, got:\n%s", out)
	}
}
