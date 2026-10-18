package telemetry

import (
	"context"
	"encoding/json"
	"reflect"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"os"
	"path/filepath"
	"testing"
	"time"

	"antigravity-switch/internal/connector"
)

type testRefresher struct{}

func (testRefresher) RefreshToken(context.Context, string) (*connector.TokenRefreshResult, error) {
	return &connector.TokenRefreshResult{AccessToken: "synthetic-token"}, nil
}

// TestDualCycleQuota_IndependentFractionsAndMissingBuckets tests F1 regression.
func TestDualCycleQuota_IndependentFractionsAndMissingBuckets(t *testing.T) {
	// 1. Independent dual-cycle Connect-RPC payload
	summaryPayload := []byte(`{
		"response": {
			"groups": [
				{
					"displayName": "Gemini Models",
					"buckets": [
						{"bucketId": "gemini-5h", "window": "5h", "remainingFraction": 0.88},
						{"bucketId": "gemini-weekly", "window": "weekly", "remainingFraction": 0.90}
					]
				},
				{
					"displayName": "Claude and GPT models",
					"buckets": [
						{"bucketId": "3p-5h", "window": "5h", "remainingFraction": 0.46},
						{"bucketId": "3p-weekly", "window": "weekly", "remainingFraction": 0.12}
					]
				}
			]
		}
	}`)

	claude, gemini, _, rawBuckets, _, status, err := ParseQuotaSummaryWithBuckets(summaryPayload)
	if err != nil {
		t.Fatalf("ParseQuotaSummaryWithBuckets failed: %v", err)
	}

	if claude.FiveHourFraction != 0.46 {
		t.Errorf("expected Claude 5h fraction 0.46, got %f", claude.FiveHourFraction)
	}
	if claude.WeeklyFraction != 0.12 {
		t.Errorf("expected Claude weekly fraction 0.12, got %f", claude.WeeklyFraction)
	}
	if gemini.FiveHourFraction != 0.88 {
		t.Errorf("expected Gemini 5h fraction 0.88, got %f", gemini.FiveHourFraction)
	}
	if gemini.WeeklyFraction != 0.90 {
		t.Errorf("expected Gemini weekly fraction 0.90, got %f", gemini.WeeklyFraction)
	}
	if len(rawBuckets) != 4 {
		t.Errorf("expected 4 raw buckets, got %d", len(rawBuckets))
	}
	if status != StatusLowClaude && status != StatusDepleted {
		t.Errorf("expected status Low Claude/Depleted for Claude at 12%%, got %s", status)
	}

	// 2. Missing weekly bucket test (should retain -1 and NOT fake 100%)
	partialPayload := []byte(`{
		"groups": [
			{
				"displayName": "Claude and GPT models",
				"buckets": [
					{"bucketId": "3p-5h", "window": "5h", "remainingFraction": 0.46}
				]
			}
		]
	}`)

	cPartial, gPartial, _, partialBuckets, _, _, err := ParseQuotaSummaryWithBuckets(partialPayload)
	if err != nil {
		t.Fatalf("ParseQuotaSummaryWithBuckets failed on partial payload: %v", err)
	}
	if cPartial.FiveHourFraction != 0.46 {
		t.Errorf("expected Claude 5h fraction 0.46, got %f", cPartial.FiveHourFraction)
	}
	if cPartial.WeeklyFraction != -1 {
		t.Errorf("expected missing weekly fraction to be -1, got %f", cPartial.WeeklyFraction)
	}
	if gPartial.FiveHourFraction != -1 || gPartial.WeeklyFraction != -1 {
		t.Errorf("expected missing Gemini buckets to be -1, got 5h=%f, week=%f", gPartial.FiveHourFraction, gPartial.WeeklyFraction)
	}
	if len(partialBuckets) != 1 {
		t.Errorf("expected 1 raw bucket, got %d", len(partialBuckets))
	}
}

// TestAccountQuotaData_ZeroGPTSerialization verifies F2 requirement: zero GPT in Go struct and JSON schema.
func TestAccountQuotaData_ZeroGPTSerialization(t *testing.T) {
	field, present := reflect.TypeOf(AccountQuotaData{}).FieldByName("GPT")
	if present {
		t.Fatalf("CRITICAL VIOLATION: struct AccountQuotaData still contains GPT field: %+v", field)
	}

	data := &AccountQuotaData{
		Email:       "user@example.com",
		Claude:      ModelQuotaBucket{ModelID: "claude-4-6", RemainingFraction: 0.50},
		Gemini:      ModelQuotaBucket{ModelID: "gemini-2-5", RemainingFraction: 0.80},
		Status:      StatusPlentiful,
		LastUpdated: time.Now().UTC(),
	}

	encoded, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	encodedStr := string(encoded)
	if strings.Contains(encodedStr, `"gpt":`) || strings.Contains(encodedStr, `"gpt"`) {
		t.Fatalf("CRITICAL VIOLATION: serialized AccountQuotaData contains GPT key: %s", encodedStr)
	}
}

// TestParseQuotaSummaryWithBuckets_F1EdgeCases verifies F1 requirements for empty, missing, and zero quota.
func TestParseQuotaSummaryWithBuckets_F1EdgeCases(t *testing.T) {
	// 1. Empty summary returns -1 fractions and empty status
	c, g, _, raw, _, status, err := ParseQuotaSummaryWithBuckets([]byte(`{"groups":[]}`))
	if err != nil {
		t.Fatalf("ParseQuotaSummaryWithBuckets failed on empty summary: %v", err)
	}
	if len(raw) != 0 {
		t.Errorf("expected 0 raw buckets, got %d", len(raw))
	}
	if c.RemainingFraction != -1 || g.RemainingFraction != -1 {
		t.Errorf("expected -1 fractions for empty summary, got c=%f, g=%f", c.RemainingFraction, g.RemainingFraction)
	}
	if status != "" {
		t.Errorf("expected empty status for empty summary, got '%s'", status)
	}

	// 2. Missing fraction returns -1
	c, g, _, raw, _, _, err = ParseQuotaSummaryWithBuckets([]byte(`{"groups":[{"displayName":"Claude models","buckets":[{"bucketId":"3p-5h","window":"5h"}]}]}`))
	if err != nil {
		t.Fatalf("ParseQuotaSummaryWithBuckets failed on missing fraction: %v", err)
	}
	if len(raw) != 1 || raw[0].RemainingFraction != -1 {
		t.Errorf("expected raw bucket with -1 fraction, got %+v", raw)
	}
	if c.FiveHourFraction != -1 || c.RemainingFraction != -1 {
		t.Errorf("expected -1 fraction for missing fraction bucket, got c5=%f, rem=%f", c.FiveHourFraction, c.RemainingFraction)
	}

	// 3. Legacy cache unmarshals with FiveHourFraction = -1 and WeeklyFraction = -1
	var bucket ModelQuotaBucket
	if err := json.Unmarshal([]byte(`{"remaining_fraction":0.2}`), &bucket); err != nil {
		t.Fatalf("unmarshal legacy bucket failed: %v", err)
	}
	if bucket.RemainingFraction != 0.2 {
		t.Errorf("expected remaining fraction 0.2, got %f", bucket.RemainingFraction)
	}
	if bucket.FiveHourFraction != -1 || bucket.WeeklyFraction != -1 {
		t.Errorf("expected legacy bucket to have -1 for cycle fractions, got 5h=%f, week=%f", bucket.FiveHourFraction, bucket.WeeklyFraction)
	}
}

// TestRetrieveUserQuotaSummary_RESTGroups verifies REST endpoint parsing summary format.
func TestRetrieveUserQuotaSummary_RESTGroups(t *testing.T) {
	summaryBody := `{"groups":[{"displayName":"Gemini Models","buckets":[{"bucketId":"gemini-5h","window":"5h","remainingFraction":0.88},{"bucketId":"gemini-weekly","window":"weekly","remainingFraction":0.90}]},{"displayName":"Claude and GPT models","buckets":[{"bucketId":"3p-5h","window":"5h","remainingFraction":0.46},{"bucketId":"3p-weekly","window":"weekly","remainingFraction":0.12}]}]}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, summaryBody)
	}))
	defer server.Close()

	client := NewQuotaClientWithEndpoint(testRefresher{}, server.URL, server.Client())
	data, err := client.RetrieveUserQuota(context.Background(), "test-refresh-token")
	if err != nil {
		t.Fatalf("RetrieveUserQuota failed: %v", err)
	}

	if len(data.RawBuckets) != 4 {
		t.Errorf("expected 4 raw buckets, got %d", len(data.RawBuckets))
	}
	if data.Claude.FiveHourFraction != 0.46 || data.Claude.WeeklyFraction != 0.12 {
		t.Errorf("unexpected Claude fractions: 5h=%f, week=%f", data.Claude.FiveHourFraction, data.Claude.WeeklyFraction)
	}
	if data.Gemini.FiveHourFraction != 0.88 || data.Gemini.WeeklyFraction != 0.90 {
		t.Errorf("unexpected Gemini fractions: 5h=%f, week=%f", data.Gemini.FiveHourFraction, data.Gemini.WeeklyFraction)
	}
}

// -----------------------------------------------------------------------------
// Comprehensive F1 Regression Tests for Edge Cases (Supervisor R3 Feedback)
// -----------------------------------------------------------------------------

// TestF1_RESTLegacyAmountWithoutFraction verifies that remainingAmount without fraction yields fraction -1.
func TestF1_RESTLegacyAmountWithoutFraction(t *testing.T) {
	// Case 1: camelCase remainingAmount: 37 without remainingFraction
	serverCamel := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"buckets":[{"modelId":"claude-4-6","remainingAmount":37}]}`)
	}))
	defer serverCamel.Close()

	clientCamel := NewQuotaClientWithEndpoint(testRefresher{}, serverCamel.URL, serverCamel.Client())
	dataCamel, err := clientCamel.RetrieveUserQuota(context.Background(), "token")
	if err != nil {
		t.Fatalf("RetrieveUserQuota camel failed: %v", err)
	}
	if dataCamel.Claude.RemainingFraction != -1 {
		t.Errorf("expected Claude fraction -1 for remainingAmount:37 without fraction, got %f", dataCamel.Claude.RemainingFraction)
	}
	if dataCamel.Claude.RemainingAmount != 37 {
		t.Errorf("expected Claude remainingAmount 37, got %d", dataCamel.Claude.RemainingAmount)
	}

	// Case 2: snake_case remaining_amount: 37 without remaining_fraction
	serverSnake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"buckets":[{"model_id":"gemini-2-5","remaining_amount":37}]}`)
	}))
	defer serverSnake.Close()

	clientSnake := NewQuotaClientWithEndpoint(testRefresher{}, serverSnake.URL, serverSnake.Client())
	dataSnake, err := clientSnake.RetrieveUserQuota(context.Background(), "token")
	if err != nil {
		t.Fatalf("RetrieveUserQuota snake failed: %v", err)
	}
	if dataSnake.Gemini.RemainingFraction != -1 {
		t.Errorf("expected Gemini fraction -1 for remaining_amount:37 without fraction, got %f", dataSnake.Gemini.RemainingFraction)
	}
	if dataSnake.Gemini.RemainingAmount != 37 {
		t.Errorf("expected Gemini remainingAmount 37, got %d", dataSnake.Gemini.RemainingAmount)
	}
}

// TestF1_RESTLegacyMissingFractionAndAmount verifies that modelId without fraction or amount yields -1.
func TestF1_RESTLegacyMissingFractionAndAmount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"buckets":[{"modelId":"claude-4-6"}]}`)
	}))
	defer server.Close()

	client := NewQuotaClientWithEndpoint(testRefresher{}, server.URL, server.Client())
	data, err := client.RetrieveUserQuota(context.Background(), "token")
	if err != nil {
		t.Fatalf("RetrieveUserQuota failed: %v", err)
	}
	if data.Claude.RemainingFraction != -1 {
		t.Errorf("expected Claude fraction -1 when fraction and amount missing, got %f", data.Claude.RemainingFraction)
	}
	if data.Claude.FiveHourFraction != -1 || data.Claude.WeeklyFraction != -1 {
		t.Errorf("expected cycle fractions -1, got 5h=%f, week=%f", data.Claude.FiveHourFraction, data.Claude.WeeklyFraction)
	}
}

// TestF1_RESTLegacyClaudeFractionZero_NoWeeklySynthesis verifies legacy single bucket 0.0 does not synthesize weekly cycle.
func TestF1_RESTLegacyClaudeFractionZero_NoWeeklySynthesis(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"buckets":[{"modelId":"claude-4-6","remainingFraction":0.0}]}`)
	}))
	defer server.Close()

	client := NewQuotaClientWithEndpoint(testRefresher{}, server.URL, server.Client())
	data, err := client.RetrieveUserQuota(context.Background(), "token")
	if err != nil {
		t.Fatalf("RetrieveUserQuota failed: %v", err)
	}
	if data.Claude.RemainingFraction != 0.0 {
		t.Errorf("expected Claude remainingFraction 0.0, got %f", data.Claude.RemainingFraction)
	}
	if data.Claude.FiveHourFraction != -1 {
		t.Errorf("expected Claude FiveHourFraction -1, got %f", data.Claude.FiveHourFraction)
	}
	if data.Claude.WeeklyFraction != -1 {
		t.Errorf("expected Claude WeeklyFraction -1 (no synthesis), got %f", data.Claude.WeeklyFraction)
	}
}

// TestF1_ConnectRPCSummaryClaudeWeeklyZero verifies opaque bucket ID with window:weekly and fraction 0.
func TestF1_ConnectRPCSummaryClaudeWeeklyZero(t *testing.T) {
	summaryPayload := []byte(`{
		"groups": [
			{
				"displayName": "Claude models",
				"buckets": [
					{"bucketId": "opaque", "window": "weekly", "remainingFraction": 0.0}
				]
			}
		]
	}`)

	claude, gemini, _, raw, _, _, err := ParseQuotaSummaryWithBuckets(summaryPayload)
	if err != nil {
		t.Fatalf("ParseQuotaSummaryWithBuckets failed: %v", err)
	}
	if claude.WeeklyFraction != 0.0 {
		t.Errorf("expected Claude weekly fraction 0.0, got %f", claude.WeeklyFraction)
	}
	if claude.FiveHourFraction != -1 {
		t.Errorf("expected Claude 5h fraction -1, got %f", claude.FiveHourFraction)
	}
	if gemini.FiveHourFraction != -1 || gemini.WeeklyFraction != -1 {
		t.Errorf("expected Gemini fractions -1, got 5h=%f, week=%f", gemini.FiveHourFraction, gemini.WeeklyFraction)
	}
	if len(raw) != 1 {
		t.Fatalf("expected 1 raw bucket, got %d", len(raw))
	}
	if raw[0].WeeklyFraction != 0.0 {
		t.Errorf("expected raw bucket weekly fraction 0.0, got %f", raw[0].WeeklyFraction)
	}
	if raw[0].FiveHourFraction != -1 {
		t.Errorf("expected raw bucket 5h fraction -1, got %f", raw[0].FiveHourFraction)
	}
}

// TestF1_CachePersistence_WeeklyZeroCycle verifies saving and reloading a weekly zero cycle preserves [-1, 0, -1, -1].
func TestF1_CachePersistence_WeeklyZeroCycle(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "quota_cache.json")
	cache := NewQuotaCache(cachePath)

	data := &AccountQuotaData{
		Email: "weeklyzero@example.com",
		Claude: ModelQuotaBucket{
			ModelID:           "claude-4-6",
			RemainingFraction: 0.0,
			FiveHourFraction:  -1,
			WeeklyFraction:    0.0,
		},
		Gemini: ModelQuotaBucket{
			ModelID:           "gemini-2-5",
			RemainingFraction: -1,
			FiveHourFraction:  -1,
			WeeklyFraction:    -1,
		},
		Status:      StatusDepleted,
		LastUpdated: time.Now().UTC(),
	}

	cache.Update(data)
	if err := cache.SaveToDisk(); err != nil {
		t.Fatalf("SaveToDisk failed: %v", err)
	}

	// Reload into fresh cache instance
	cache2 := NewQuotaCache(cachePath)
	if err := cache2.LoadFromDisk(); err != nil {
		t.Fatalf("LoadFromDisk failed: %v", err)
	}

	loaded, ok := cache2.Get("weeklyzero@example.com")
	if !ok || loaded == nil {
		t.Fatalf("expected loaded quota data")
	}

	if loaded.Claude.FiveHourFraction != -1 {
		t.Errorf("expected loaded Claude 5h fraction -1, got %f", loaded.Claude.FiveHourFraction)
	}
	if loaded.Claude.WeeklyFraction != 0.0 {
		t.Errorf("expected loaded Claude weekly fraction 0.0, got %f", loaded.Claude.WeeklyFraction)
	}
	if loaded.Gemini.FiveHourFraction != -1 || loaded.Gemini.WeeklyFraction != -1 {
		t.Errorf("expected loaded Gemini fractions -1, got 5h=%f, week=%f", loaded.Gemini.FiveHourFraction, loaded.Gemini.WeeklyFraction)
	}
}

// TestF1_CachePersistence_LegacyCacheMissingFamily verifies legacy cache with only Claude 20% preserves Claude and keeps Gemini unknown.
func TestF1_CachePersistence_LegacyCacheMissingFamily(t *testing.T) {
	legacyJSON := []byte(`{
		"saved_at": "2026-10-18T12:00:00Z",
		"account_count": 1,
		"matrix": {
			"acc@test.com": {
				"email": "acc@test.com",
				"claude": {
					"model_id": "claude-4-6",
					"remaining_fraction": 0.2
				}
			}
		}
	}`)

	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "legacy_cache.json")
	if err := os.WriteFile(cachePath, legacyJSON, 0644); err != nil {
		t.Fatalf("failed to write legacy cache: %v", err)
	}

	cache := NewQuotaCache(cachePath)
	if err := cache.LoadFromDisk(); err != nil {
		t.Fatalf("LoadFromDisk failed: %v", err)
	}

	loaded, ok := cache.Get("acc@test.com")
	if !ok || loaded == nil {
		t.Fatalf("expected account loaded")
	}

	if loaded.Claude.RemainingFraction != 0.2 {
		t.Errorf("expected Claude fraction 0.2, got %f", loaded.Claude.RemainingFraction)
	}
	if loaded.Claude.FiveHourFraction != -1 || loaded.Claude.WeeklyFraction != -1 {
		t.Errorf("expected Claude cycle fractions -1, got 5h=%f, week=%f", loaded.Claude.FiveHourFraction, loaded.Claude.WeeklyFraction)
	}

	// Missing Gemini MUST have fractions == -1, NEVER fake 0.0!
	if loaded.Gemini.RemainingFraction != -1 {
		t.Errorf("expected missing Gemini RemainingFraction == -1, got %f", loaded.Gemini.RemainingFraction)
	}
	if loaded.Gemini.FiveHourFraction != -1 || loaded.Gemini.WeeklyFraction != -1 {
		t.Errorf("expected missing Gemini cycle fractions == -1, got 5h=%f, week=%f", loaded.Gemini.FiveHourFraction, loaded.Gemini.WeeklyFraction)
	}
}

// TestF1_CachePersistence_ClaudeObjectWithoutRemainingFraction verifies Claude object without fraction produces -1.
func TestF1_CachePersistence_ClaudeObjectWithoutRemainingFraction(t *testing.T) {
	legacyJSON := []byte(`{
		"matrix": {
			"acc2@test.com": {
				"email": "acc2@test.com",
				"claude": {
					"model_id": "claude-4-6"
				}
			}
		}
	}`)

	var snapshot QuotaCacheSnapshot
	if err := json.Unmarshal(legacyJSON, &snapshot); err != nil {
		t.Fatalf("unmarshal snapshot failed: %v", err)
	}

	acc, ok := snapshot.Matrix["acc2@test.com"]
	if !ok || acc == nil {
		t.Fatalf("expected acc2@test.com in snapshot")
	}

	if acc.Claude.RemainingFraction != -1 {
		t.Errorf("expected Claude fraction -1 when remaining_fraction omitted, got %f", acc.Claude.RemainingFraction)
	}
	if acc.Claude.FiveHourFraction != -1 || acc.Claude.WeeklyFraction != -1 {
		t.Errorf("expected Claude cycle fractions -1, got 5h=%f, week=%f", acc.Claude.FiveHourFraction, acc.Claude.WeeklyFraction)
	}
}
