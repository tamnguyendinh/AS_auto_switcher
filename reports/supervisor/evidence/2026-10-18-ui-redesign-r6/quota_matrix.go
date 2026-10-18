//go:build ignore

// R6 reruns the independent Supervisor matrix without changing historical evidence.
// Run with runtime_probe.go from the repository root. All HTTP traffic is loopback and synthetic.
package main

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"

	"antigravity-switch/internal/telemetry"
	"antigravity-switch/internal/ui"
)

var quotaMismatches []string

func runQuotaMatrix(tmp string) []any {
	unknown := fractions(-1, -1, -1, -1, false)
	type quotaCase struct {
		name, payload string
		expect        ui.DualCycleQuota
		legacyREST    bool
	}
	cases := []quotaCase{
		{"full", fullSummary, fractions(.46, .12, .88, .90, true), false},
		{"empty_rest", `{"groups":[]}`, unknown, false},
		{"empty_rpc", `{"response":{"groups":[]}}`, unknown, false},
		{"missing_fraction", `{"groups":[{"displayName":"Claude models","buckets":[{"bucketId":"3p-5h","window":"5h"}]}]}`, unknown, false},
		{"null_fraction", `{"groups":[{"displayName":"Claude models","buckets":[{"bucketId":"3p-5h","window":"5h","remainingFraction":null}]}]}`, unknown, false},
		{"zero_5h_standard_id", `{"groups":[{"displayName":"Claude models","buckets":[{"bucketId":"3p-5h","window":"5h","remainingFraction":0}]}]}`, fractions(0, -1, -1, -1, true), false},
		{"zero_week_standard_id", `{"groups":[{"displayName":"Claude models","buckets":[{"bucketId":"3p-weekly","window":"weekly","remainingFraction":0}]}]}`, fractions(-1, 0, -1, -1, true), false},
		{"zero_5h_window_only", `{"groups":[{"displayName":"Claude models","buckets":[{"bucketId":"opaque","window":"5h","remainingFraction":0}]}]}`, fractions(0, -1, -1, -1, true), false},
		{"zero_week_window_only", `{"groups":[{"displayName":"Claude models","buckets":[{"bucketId":"opaque","window":"weekly","remainingFraction":0}]}]}`, fractions(-1, 0, -1, -1, true), false},
		{"nonzero_week_window_only", `{"groups":[{"displayName":"Gemini Models","buckets":[{"bucketId":"opaque","window":"weekly","remainingFraction":0.37}]}]}`, fractions(-1, -1, -1, .37, true), false},
		{"mixed_missing", `{"groups":[{"displayName":"Claude models","buckets":[{"bucketId":"3p-5h","window":"5h","remainingFraction":0.23},{"bucketId":"3p-weekly","window":"weekly"}]},{"displayName":"Gemini Models","buckets":[{"bucketId":"gemini-5h","window":"5h"}]}]}`, fractions(.23, -1, -1, -1, true), false},
		{"legacy_REST_missing_fraction", `{"buckets":[{"modelId":"claude-4-6"}]}`, unknown, true},
		{"legacy_REST_zero_no_cycle", `{"buckets":[{"modelId":"claude-4-6","remainingFraction":0}]}`, fractions(0, -1, -1, -1, true), true},
		{"legacy_REST_amount_camel", `{"buckets":[{"modelId":"claude-4-6","remainingAmount":37}]}`, unknown, true},
		{"legacy_REST_amount_snake", `{"buckets":[{"model_id":"gemini-2-5","remaining_amount":37}]}`, unknown, true},
		{"legacy_REST_zero_amount", `{"buckets":[{"modelId":"claude-4-6","remainingAmount":0}]}`, unknown, true},
		{"legacy_REST_null_fraction", `{"buckets":[{"modelId":"claude-4-6","remainingFraction":null,"remainingAmount":37}]}`, unknown, true},
		{"legacy_REST_unknown_then_valid", `{"buckets":[{"modelId":"claude-a"},{"modelId":"claude-b","remainingFraction":0.2}]}`, fractions(.2, -1, -1, -1, true), true},
		{"legacy_REST_valid_then_unknown", `{"buckets":[{"modelId":"claude-b","remainingFraction":0.2},{"modelId":"claude-a"}]}`, fractions(.2, -1, -1, -1, true), true},
		{"both_zero_without_resets", `{"groups":[{"displayName":"Claude models","buckets":[{"bucketId":"opaque-1","window":"5h","remainingFraction":0},{"bucketId":"opaque-2","window":"weekly","remainingFraction":0}]}]}`, fractions(0, 0, -1, -1, true), false},
		{"gemini_zero_week_window_only", `{"groups":[{"displayName":"Gemini Models","buckets":[{"bucketId":"opaque","window":"weekly","remainingFraction":0}]}]}`, fractions(-1, -1, -1, 0, true), false},
	}
	var payload atomic.Value
	payload.Store(fullSummary)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, payload.Load().(string))
	}))
	defer server.Close()
	client := telemetry.NewQuotaClientWithEndpoint(refresher{}, server.URL+"/v1internal:retrieveUserQuotaSummary", server.Client())
	client.SetUserAgent("supervisor-r6-loopback")
	cache := telemetry.NewQuotaCache(filepath.Join(tmp, "matrix.json"))
	pool, err := telemetry.NewTelemetryPoolWithSource(client, cache, source{})
	must(err)
	pool.SetLocalQuotaFetcher(func(context.Context, string) (*telemetry.AccountQuotaData, error) {
		return nil, errors.New("force loopback REST")
	})
	var results []any
	check := func(name, stage string, data *telemetry.AccountQuotaData, expect ui.DualCycleQuota) map[string]any {
		actual := ui.ExtractDualCycleQuota(data)
		match := actual == expect
		require("F1/matrix/"+name+"/"+stage, match)
		if !match {
			quotaMismatches = append(quotaMismatches, name+"/"+stage)
		}
		out := snapshot(data)
		out["stage"], out["expected"], out["matches_expected"] = stage, expect, match
		out["cycle_storage"] = map[string]any{"claude": data.Claude, "gemini": data.Gemini}
		return out
	}
	for _, tc := range cases {
		entry := map[string]any{"case": tc.name, "input": tc.payload}
		if !tc.legacyREST {
			entry["summary_parser"] = check(tc.name, "summary_parser", parsed(tc.payload), tc.expect)
		}
		payload.Store(tc.payload)
		email := tc.name + "@example.com"
		data, err := pool.PollAccount(context.Background(), email)
		must(err)
		entry["HTTP_pool"] = check(tc.name, "HTTP_pool", data, tc.expect)
		reloaded := telemetry.NewQuotaCache(cache.FilePath())
		must(reloaded.LoadFromDisk())
		persisted, ok := reloaded.Get(email)
		if !ok {
			panic("matrix disk cache missing")
		}
		entry["disk_reload"] = check(tc.name, "disk_reload", persisted, tc.expect)
		results = append(results, entry)
	}
	legacyCases := []struct {
		name, body string
		expect     ui.DualCycleQuota
	}{
		{"legacy_cache_nonzero", `{"claude":{"remaining_fraction":0.2},"gemini":{"remaining_fraction":0.8}}`, fractions(.2, -1, .8, -1, true)},
		{"legacy_cache_zero", `{"claude":{"remaining_fraction":0},"gemini":{"remaining_fraction":0.8}}`, fractions(0, -1, .8, -1, true)},
		{"legacy_cache_missing_model", `{"claude":{"remaining_fraction":0.2}}`, fractions(.2, -1, -1, -1, true)},
		{"legacy_cache_missing_fraction", `{"claude":{"model_id":"claude-4-6"},"gemini":{"remaining_fraction":-1}}`, unknown},
		{"legacy_cache_missing_claude", `{"gemini":{"remaining_fraction":0.8}}`, fractions(-1, -1, .8, -1, true)},
		{"legacy_cache_null_family", `{"claude":{"remaining_fraction":0.2},"gemini":null}`, fractions(.2, -1, -1, -1, true)},
		{"cache_no_families", `{}`, unknown},
		{"cache_null_fraction", `{"claude":{"remaining_fraction":null}}`, unknown},
		{"cache_weekly_zero_without_raw", `{"claude":{"remaining_fraction":0,"five_hour_fraction":-1,"weekly_fraction":0}}`, fractions(-1, 0, -1, -1, true)},
		{"cache_both_zero_without_raw", `{"claude":{"remaining_fraction":0,"five_hour_fraction":0,"weekly_fraction":0}}`, fractions(0, 0, -1, -1, true)},
		{"cache_gemini_both_zero_without_raw", `{"gemini":{"remaining_fraction":0,"five_hour_fraction":0,"weekly_fraction":0}}`, fractions(-1, -1, 0, 0, true)},
		{"cache_camel_weekly_zero", `{"claude":{"remainingFraction":0,"fiveHourFraction":-1,"weeklyFraction":0}}`, fractions(-1, 0, -1, -1, true)},
	}
	for _, tc := range legacyCases {
		path := filepath.Join(tmp, tc.name+".json")
		must(os.WriteFile(path, []byte(`{"matrix":{"legacy@example.com":`+tc.body+`}}`), 0600))
		legacy := telemetry.NewQuotaCache(path)
		must(legacy.LoadFromDisk())
		data, ok := legacy.Get("legacy@example.com")
		if !ok {
			panic("legacy matrix entry missing")
		}
		entry := map[string]any{"case": tc.name, "input": tc.body, "disk_reload": check(tc.name, "disk_reload", data, tc.expect)}
		data.Email = "legacy@example.com"
		legacy.Update(data)
		must(legacy.SaveToDisk())
		reloaded := telemetry.NewQuotaCache(path)
		must(reloaded.LoadFromDisk())
		again, ok := reloaded.Get(data.Email)
		if !ok {
			panic("cache resave missing")
		}
		entry["disk_resave_reload"] = check(tc.name, "disk_resave_reload", again, tc.expect)
		results = append(results, entry)
	}
	return results
}

func retiredUISurface() map[string]any {
	fields := map[string]bool{}
	for _, name := range []string{"headerBg", "statusBarLabel", "activeStatusLabel", "activeQuotaLabel"} {
		_, present := reflect.TypeOf(ui.App{}).FieldByName(name)
		fields[name] = present
		require("F4/field/"+name, !present)
	}
	functions := map[string]bool{"NewQuotaCountdownLabel": false, "FormatRemainingCountdown": false, "BuildActiveQuotaSummary": false, "NewQuotaBadgeGroup": false, "NewDualCycleQuotaBadge": false, "NewAccountCard": false, "RefreshSingleQuota": false}
	set := token.NewFileSet()
	files, err := filepath.Glob("internal/ui/*.go")
	must(err)
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(set, path, nil, 0)
		must(err)
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok {
				if _, wanted := functions[fn.Name.Name]; wanted {
					functions[fn.Name.Name] = true
				}
			}
		}
	}
	for name, present := range functions {
		require("F4/function/"+name, !present)
	}
	return map[string]any{"retired_fields_present": fields, "retired_functions_present": functions}
}
