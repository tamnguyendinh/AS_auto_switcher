package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

// TestAccountQuotaData_ZeroGPTSerialization verifies F2 requirement: zero GPT in JSON schema.
func TestAccountQuotaData_ZeroGPTSerialization(t *testing.T) {
	data := &AccountQuotaData{
		Email:       "user@example.com",
		Claude:      ModelQuotaBucket{ModelID: "claude-4-6", RemainingFraction: 0.50},
		Gemini:      ModelQuotaBucket{ModelID: "gemini-2-5", RemainingFraction: 0.80},
		GPT:         ModelQuotaBucket{ModelID: "gpt-oss", RemainingFraction: 0.90},
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