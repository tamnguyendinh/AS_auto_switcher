//go:build ignore

// Targeted follow-up to the unresolved F1 fallback: a count alone cannot prove a percentage.
// Run explicitly from repository root. Product code, credentials and real endpoints are untouched.
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
	"sync/atomic"

	"antigravity-switch/internal/connector"
	"antigravity-switch/internal/telemetry"
	"antigravity-switch/internal/ui"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
)

type amountRefresher struct{}

func (amountRefresher) RefreshToken(context.Context, string) (*connector.TokenRefreshResult, error) {
	return &connector.TokenRefreshResult{AccessToken: "synthetic"}, nil
}

type amountSource struct{}

func (amountSource) ListAccounts() ([]telemetry.AccountTokenInfo, error) { return nil, nil }
func (amountSource) GetRefreshToken(string) (string, error)              { return "synthetic", nil }
func amountMust(err error) {
	if err != nil {
		panic(err)
	}
}
func amountBadge(o fyne.CanvasObject) []string {
	var out []string
	if text, ok := o.(*canvas.Text); ok {
		out = append(out, text.Text)
	}
	if c, ok := o.(*fyne.Container); ok {
		for _, child := range c.Objects {
			out = append(out, amountBadge(child)...)
		}
	}
	return out
}
func main() {
	fyneApp := test.NewApp()
	defer fyneApp.Quit()
	tmp, err := os.MkdirTemp(".tmp", "supervisor-r3-amount-")
	amountMust(err)
	abs, err := filepath.Abs(tmp)
	amountMust(err)
	root, err := filepath.Abs(".tmp")
	amountMust(err)
	if filepath.Dir(abs) != root {
		panic("cleanup target outside repository .tmp")
	}
	defer os.RemoveAll(abs)
	var body atomic.Value
	body.Store("")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body.Load().(string))
	}))
	defer server.Close()
	client := telemetry.NewQuotaClientWithEndpoint(amountRefresher{}, server.URL, server.Client())
	client.SetUserAgent("supervisor-r3-loopback")
	cache := telemetry.NewQuotaCache(filepath.Join(abs, "quota.json"))
	pool, err := telemetry.NewTelemetryPoolWithSource(client, cache, amountSource{})
	amountMust(err)
	pool.SetLocalQuotaFetcher(func(context.Context, string) (*telemetry.AccountQuotaData, error) {
		return nil, errors.New("force REST")
	})
	unknown := ui.DualCycleQuota{Claude5h: -1, ClaudeWeek: -1, Gemini5h: -1, GeminiWeek: -1}
	var results []any
	failed := false
	for _, payload := range []string{
		`{"buckets":[{"modelId":"claude-4-6","remainingAmount":37}]}`,
		`{"buckets":[{"model_id":"gemini-2-5","remaining_amount":37}]}`,
	} {
		body.Store(payload)
		data, err := pool.PollAccount(context.Background(), "amount@example.com")
		amountMust(err)
		reload := telemetry.NewQuotaCache(cache.FilePath())
		amountMust(reload.LoadFromDisk())
		after, ok := reload.Get("amount@example.com")
		if !ok {
			panic("persisted amount case missing")
		}
		entry := map[string]any{"input": payload, "expected": unknown}
		for _, stage := range []struct {
			name string
			data *telemetry.AccountQuotaData
		}{{"HTTP_pool", data}, {"disk_reload", after}} {
			actual := ui.ExtractDualCycleQuota(stage.data)
			match := actual == unknown
			if !match {
				failed = true
			}
			entry[stage.name] = map[string]any{"actual": actual, "badge": amountBadge(ui.NewDualCycleQuotaBadge(stage.data)), "matches_expected": match}
		}
		results = append(results, entry)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	amountMust(encoder.Encode(results))
	if failed {
		os.Exit(1)
	}
}
