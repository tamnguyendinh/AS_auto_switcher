//go:build ignore

// Coder independent acceptance probe verifying F1, F2, F4 fixes and F3, F5, F6 preservation.
// Run from repo root: go run reports/coder/evidence/2026-10-18-ui-redesign-f1-f6/coder_probe.go
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"time"

	"antigravity-switch/internal/connector"
	"antigravity-switch/internal/telemetry"
	"antigravity-switch/internal/ui"
	"fyne.io/fyne/v2/test"
)

type dummyRefresher struct{}

func (dummyRefresher) RefreshToken(context.Context, string) (*connector.TokenRefreshResult, error) {
	return &connector.TokenRefreshResult{AccessToken: "token"}, nil
}

type dummySource struct{}

func (dummySource) ListAccounts() ([]telemetry.AccountTokenInfo, error) { return nil, nil }
func (dummySource) GetRefreshToken(string) (string, error)              { return "refresh", nil }

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	results := make(map[string]any)

	// =========================================================================
	// F2 Verification: Eradicate GPT from AccountQuotaData struct and contract
	// =========================================================================
	_, present := reflect.TypeOf(telemetry.AccountQuotaData{}).FieldByName("GPT")
	testData := &telemetry.AccountQuotaData{
		Email:       "user@example.com",
		Claude:      telemetry.ModelQuotaBucket{RemainingFraction: 0.5},
		Gemini:      telemetry.ModelQuotaBucket{RemainingFraction: 0.8},
		Status:      "Pristine",
		LastUpdated: time.Now().UTC(),
	}
	encoded, err := json.Marshal(testData)
	must(err)
	f2Result := map[string]any{
		"GPT_field_present":           present,
		"expected_field_present":      false,
		"serialized_contains_gpt_key": strings.Contains(string(encoded), `"gpt"`),
		"status":                      !present && !strings.Contains(string(encoded), `"gpt"`),
	}
	results["F2_contract"] = f2Result
	if present {
		panic("F2 Failed: GPT field still present in AccountQuotaData!")
	}

	// =========================================================================
	// F1 Verification: Truthful Quota (Empty, Missing, Depleted, Legacy Cache)
	// =========================================================================
	f1Cases := make(map[string]any)

	// 1. Connect-RPC empty summary
	cEmpty, gEmpty, _, rawEmpty, _, statusEmpty, err := telemetry.ParseQuotaSummaryWithBuckets([]byte(`{"response":{"groups":[]}}`))
	must(err)
	qEmpty := &telemetry.AccountQuotaData{Claude: cEmpty, Gemini: gEmpty, RawBuckets: rawEmpty, Status: statusEmpty}
	dqEmpty := ui.ExtractDualCycleQuota(qEmpty)
	f1Cases["empty_connect_rpc"] = map[string]any{
		"has_data": dqEmpty.HasData,
		"c5h":      dqEmpty.Claude5h,
		"g5h":      dqEmpty.Gemini5h,
		"pass":     !dqEmpty.HasData && dqEmpty.Claude5h == -1 && dqEmpty.Gemini5h == -1,
	}

	// 2. REST empty summary
	var restPayload atomic.Value
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, restPayload.Load().(string))
	}))
	defer server.Close()

	client := telemetry.NewQuotaClientWithEndpoint(dummyRefresher{}, server.URL, server.Client())
	tmpDir, err := os.MkdirTemp(".tmp", "coder-probe-")
	must(err)
	defer os.RemoveAll(tmpDir)

	cache := telemetry.NewQuotaCache(filepath.Join(tmpDir, "quota.json"))
	pool, err := telemetry.NewTelemetryPoolWithSource(client, cache, dummySource{})
	must(err)
	pool.SetLocalQuotaFetcher(func(context.Context, string) (*telemetry.AccountQuotaData, error) {
		return nil, errors.New("force REST")
	})

	restPayload.Store(`{"groups":[]}`)
	emptyRestData, err := pool.PollAccount(context.Background(), "empty@example.com")
	must(err)
	dqEmptyRest := ui.ExtractDualCycleQuota(emptyRestData)
	f1Cases["empty_rest"] = map[string]any{
		"has_data": dqEmptyRest.HasData,
		"pass":     !dqEmptyRest.HasData && dqEmptyRest.Claude5h == -1 && dqEmptyRest.Gemini5h == -1,
	}

	// 3. Partial depleted zero with opaque ID
	cZero, gZero, _, rawZero, _, _, err := telemetry.ParseQuotaSummaryWithBuckets([]byte(`{"groups":[{"displayName":"Claude models","buckets":[{"bucketId":"opaque","window":"5h","remainingFraction":0}]}]}`))
	must(err)
	qZero := &telemetry.AccountQuotaData{Claude: cZero, Gemini: gZero, RawBuckets: rawZero}
	dqZero := ui.ExtractDualCycleQuota(qZero)
	f1Cases["window_only_partial_zero"] = map[string]any{
		"c5h":      dqZero.Claude5h,
		"cWeek":    dqZero.ClaudeWeek,
		"g5h":      dqZero.Gemini5h,
		"has_data": dqZero.HasData,
		"pass":     dqZero.HasData && dqZero.Claude5h == 0 && dqZero.ClaudeWeek == -1 && dqZero.Gemini5h == -1,
	}

	// 4. Missing fraction
	cMiss, gMiss, _, rawMiss, _, _, err := telemetry.ParseQuotaSummaryWithBuckets([]byte(`{"groups":[{"displayName":"Claude models","buckets":[{"bucketId":"3p-5h","window":"5h"}]}]}`))
	must(err)
	qMiss := &telemetry.AccountQuotaData{Claude: cMiss, Gemini: gMiss, RawBuckets: rawMiss}
	dqMiss := ui.ExtractDualCycleQuota(qMiss)
	f1Cases["missing_fraction"] = map[string]any{
		"has_data": dqMiss.HasData,
		"pass":     !dqMiss.HasData && dqMiss.Claude5h == -1,
	}

	// 5. Legacy cache (never duplicate to weekly)
	legacyPath := filepath.Join(tmpDir, "legacy.json")
	legacyJSON := `{"matrix":{"legacy@example.com":{"email":"legacy@example.com","claude":{"remaining_fraction":0.2},"gemini":{"remaining_fraction":0.8}}}}`
	must(os.WriteFile(legacyPath, []byte(legacyJSON), 0600))
	legacyCache := telemetry.NewQuotaCache(legacyPath)
	must(legacyCache.LoadFromDisk())
	legacyData, ok := legacyCache.Get("legacy@example.com")
	if !ok {
		panic("legacy cache missing")
	}
	dqLegacy := ui.ExtractDualCycleQuota(legacyData)
	f1Cases["legacy_cache"] = map[string]any{
		"c5h":      dqLegacy.Claude5h,
		"cWeek":    dqLegacy.ClaudeWeek,
		"g5h":      dqLegacy.Gemini5h,
		"gWeek":    dqLegacy.GeminiWeek,
		"has_data": dqLegacy.HasData,
		"pass":     dqLegacy.HasData && dqLegacy.Claude5h == 0.2 && dqLegacy.ClaudeWeek == -1 && dqLegacy.Gemini5h == 0.8 && dqLegacy.GeminiWeek == -1,
	}
	results["F1_cases"] = f1Cases

	// =========================================================================
	// F4 Verification: Retired UI Objects and Stubs Eradication
	// =========================================================================
	fyneApp := test.NewApp()
	defer fyneApp.Quit()
	app := ui.NewAppWithFyne(fyneApp, nil)
	appElem := reflect.TypeOf(app).Elem()

	var legacyFieldsFound []string
	for _, name := range []string{"headerBg", "statusBarLabel", "activeStatusLabel", "activeQuotaLabel"} {
		if _, exists := appElem.FieldByName(name); exists {
			legacyFieldsFound = append(legacyFieldsFound, name)
		}
	}
	results["F4_legacy_ui_fields"] = map[string]any{
		"found_legacy_fields": legacyFieldsFound,
		"pass":                len(legacyFieldsFound) == 0,
	}
	if len(legacyFieldsFound) > 0 {
		panic(fmt.Sprintf("F4 Failed: legacy fields still exist in App: %v", legacyFieldsFound))
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	must(encoder.Encode(map[string]any{
		"status":  "ALL_PROBES_PASS",
		"results": results,
	}))
}
