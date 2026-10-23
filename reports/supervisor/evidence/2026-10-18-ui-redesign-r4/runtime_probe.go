//go:build ignore

// Run from the repository root: go run reports/supervisor/evidence/2026-10-18-ui-redesign-r4/runtime_probe.go reports/supervisor/evidence/2026-10-18-ui-redesign-r4/quota_matrix.go
// Independent acceptance probe: synthetic accounts, loopback REST, isolated .tmp storage.
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"antigravity-switch/internal/connector"
	"antigravity-switch/internal/manager"
	"antigravity-switch/internal/model"
	"antigravity-switch/internal/store"
	"antigravity-switch/internal/telemetry"
	"antigravity-switch/internal/ui"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const evidenceDir = "reports/supervisor/evidence/2026-10-18-ui-redesign-r4"
const fullSummary = `{"groups":[{"displayName":"Gemini Models","buckets":[{"bucketId":"gemini-5h","window":"5h","remainingFraction":0.88,"resetTime":"2026-10-18T12:00:00Z"},{"bucketId":"gemini-weekly","window":"weekly","remainingFraction":0.90,"resetTime":"2026-10-24T12:00:00Z"}]},{"displayName":"Claude and GPT models","buckets":[{"bucketId":"3p-5h","window":"5h","remainingFraction":0.46,"resetTime":"2026-10-18T13:00:00Z"},{"bucketId":"3p-weekly","window":"weekly","remainingFraction":0.12,"resetTime":"2026-10-24T13:00:00Z"}]}]}`

type refresher struct{}

func (refresher) RefreshToken(context.Context, string) (*connector.TokenRefreshResult, error) {
	return &connector.TokenRefreshResult{AccessToken: "synthetic"}, nil
}

type source struct{}

func (source) ListAccounts() ([]telemetry.AccountTokenInfo, error) { return nil, nil }
func (source) GetRefreshToken(string) (string, error)              { return "synthetic", nil }

type fallback struct{}

func (fallback) RetrieveUserQuota(context.Context, string) (*telemetry.AccountQuotaData, error) {
	return nil, errors.New("synthetic unavailable provider")
}
func must(err error) {
	if err != nil {
		panic(err)
	}
}
func walk(o fyne.CanvasObject, visit func(fyne.CanvasObject)) {
	visit(o)
	if c, ok := o.(*fyne.Container); ok {
		for _, child := range c.Objects {
			walk(child, visit)
		}
	}
	if s, ok := o.(*container.Scroll); ok {
		walk(s.Content, visit)
	}
}
func texts(o fyne.CanvasObject) []string {
	var out []string
	walk(o, func(o fyne.CanvasObject) {
		switch v := o.(type) {
		case *canvas.Text:
			out = append(out, v.Text)
		case *widget.Label:
			out = append(out, v.Text)
		}
	})
	return out
}
func gridOf(root fyne.CanvasObject) *fyne.Container {
	var out *fyne.Container
	walk(root, func(o fyne.CanvasObject) {
		if c, ok := o.(*fyne.Container); ok && strings.Contains(fmt.Sprintf("%T", c.Layout), "gridLayout") {
			out = c
		}
	})
	return out
}
func waitFor(label string, ready func() bool) {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if ready() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	panic("timeout: " + label)
}
func capture(app *ui.App, name string) {
	f, err := os.Create(filepath.Join(evidenceDir, name))
	must(err)
	must(png.Encode(f, app.Window().Canvas().Capture()))
	must(f.Close())
}
func hex(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
}
func parsed(payload string) *telemetry.AccountQuotaData {
	c, g, _, raw, reset, status, err := telemetry.ParseQuotaSummaryWithBuckets([]byte(payload))
	must(err)
	return &telemetry.AccountQuotaData{Email: "account01@example.com", Claude: c, Gemini: g, RawBuckets: raw, ResetTime: reset, Status: status, LastUpdated: time.Now().UTC()}
}
func fractions(c5, cw, g5, gw float64, has bool) ui.DualCycleQuota {
	return ui.DualCycleQuota{Claude5h: c5, ClaudeWeek: cw, Gemini5h: g5, GeminiWeek: gw, HasData: has}
}
func snapshot(quota *telemetry.AccountQuotaData) map[string]any {
	dq := ui.ExtractDualCycleQuota(quota)
	rows := container.NewVBox(
		ui.CreateQuotaCycleRow("5h:", dq.Claude5h, dq.Gemini5h, dq.HasData, dq.IsError),
		ui.CreateQuotaCycleRow("week:", dq.ClaudeWeek, dq.GeminiWeek, dq.HasData, dq.IsError),
	)
	return map[string]any{"fractions": dq, "badge_text": texts(rows), "raw_bucket_count": len(quota.RawBuckets), "status": quota.Status}
}

var invariantFailures []string

func require(name string, pass bool) {
	if !pass {
		invariantFailures = append(invariantFailures, name)
	}
}
func buttonIn(o fyne.CanvasObject, refresh bool) *widget.Button {
	var out *widget.Button
	walk(o, func(o fyne.CanvasObject) {
		if b, ok := o.(*widget.Button); ok && ((refresh && b.Icon == theme.ViewRefreshIcon()) || (!refresh && b.Text == "Chuyển")) {
			out = b
		}
	})
	if out == nil {
		panic("button not found")
	}
	return out
}
func main() {
	result := map[string]any{"driver": "Fyne test.NewApp software canvas; synthetic HTTP; not native OS capture", "reviewed_commit": "5996891479f8271bfdbe5822f57dbd81b7161d0f"}
	defer func() {
		result["F1_mismatches"] = quotaMismatches
		result["other_invariant_failures"] = invariantFailures
		e := json.NewEncoder(os.Stdout)
		e.SetIndent("", "  ")
		must(e.Encode(result))
		if len(quotaMismatches) > 0 || len(invariantFailures) > 0 {
			os.Exit(1)
		}
	}()
	tmp, err := os.MkdirTemp(".tmp", "supervisor-ui-r4-")
	must(err)
	tmp, err = filepath.Abs(tmp)
	must(err)
	tmpRoot, err := filepath.Abs(".tmp")
	must(err)
	if filepath.Dir(tmp) != tmpRoot {
		panic("cleanup target outside workspace .tmp")
	}
	defer os.RemoveAll(tmp)
	fyneApp := test.NewApp()
	defer fyneApp.Quit()
	full := parsed(`{"response":` + fullSummary + `}`)
	expected := fractions(.46, .12, .88, .90, true)
	fullResult := snapshot(full)
	fullResult["expected"] = expected
	fullResult["matches_expected"] = ui.ExtractDualCycleQuota(full) == expected
	fullResult["claude_resets_distinct"] = !full.Claude.FiveHourReset.Equal(full.Claude.WeeklyReset)
	fullResult["gemini_resets_distinct"] = !full.Gemini.FiveHourReset.Equal(full.Gemini.WeeklyReset)
	result["F1_connect_rpc_parser"] = fullResult
	require("F1/full_summary", ui.ExtractDualCycleQuota(full) == expected)

	cases := []struct {
		name, payload string
		expected      ui.DualCycleQuota
	}{
		{"partial_nonzero", `{"groups":[{"displayName":"Claude and GPT models","buckets":[{"bucketId":"3p-5h","window":"5h","remainingFraction":0.46}]}]}`, fractions(.46, -1, -1, -1, true)},
		{"partial_depleted_zero", `{"groups":[{"displayName":"Claude and GPT models","buckets":[{"bucketId":"3p-5h","window":"5h","remainingFraction":0}]}]}`, fractions(0, -1, -1, -1, true)},
		{"all_depleted_zero", strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(fullSummary, "0.88", "0"), "0.90", "0"), "0.46", "0"), "0.12", "0"), fractions(0, 0, 0, 0, true)},
		{"window_only_opaque_ids", `{"groups":[{"displayName":"Gemini Models","buckets":[{"bucketId":"a","window":"5h","remainingFraction":0.88},{"bucketId":"b","window":"weekly","remainingFraction":0.90}]},{"displayName":"Claude models","buckets":[{"bucketId":"c","window":"5h","remainingFraction":0.46},{"bucketId":"d","window":"weekly","remainingFraction":0.12}]}]}`, expected},
		{"window_only_partial_zero", `{"groups":[{"displayName":"Claude models","buckets":[{"bucketId":"opaque","window":"5h","remainingFraction":0}]}]}`, fractions(0, -1, -1, -1, true)},
		{"missing_fraction", `{"groups":[{"displayName":"Claude models","buckets":[{"bucketId":"3p-5h","window":"5h"}]}]}`, fractions(-1, -1, -1, -1, false)},
		{"empty_summary", `{"groups":[]}`, fractions(-1, -1, -1, -1, false)},
	}
	var caseResults []any
	for _, tc := range cases {
		q := parsed(`{"response":` + tc.payload + `}`)
		entry := snapshot(q)
		entry["name"] = tc.name
		entry["expected"] = tc.expected
		entry["matches_expected"] = ui.ExtractDualCycleQuota(q) == tc.expected
		require("F1/"+tc.name, ui.ExtractDualCycleQuota(q) == tc.expected)
		caseResults = append(caseResults, entry)
	}
	result["F1_missing_and_zero_cases"] = caseResults

	var restPayload atomic.Value
	restPayload.Store(fullSummary)
	var restPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		restPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, restPayload.Load().(string))
	}))
	defer server.Close()
	client := telemetry.NewQuotaClientWithEndpoint(refresher{}, server.URL+"/v1internal:retrieveUserQuotaSummary", server.Client())
	client.SetUserAgent("supervisor-r4-synthetic")
	cache := telemetry.NewQuotaCache(filepath.Join(tmp, "quota.json"))
	pool, err := telemetry.NewTelemetryPoolWithSource(client, cache, source{})
	must(err)
	pool.SetLocalQuotaFetcher(func(context.Context, string) (*telemetry.AccountQuotaData, error) {
		return nil, errors.New("force REST")
	})
	restData, err := pool.PollAccount(context.Background(), "account01@example.com")
	must(err)
	fromDisk := telemetry.NewQuotaCache(cache.FilePath())
	must(fromDisk.LoadFromDisk())
	roundTrip, ok := fromDisk.Get("account01@example.com")
	if !ok {
		panic("cache reload missing")
	}
	restResult := snapshot(restData)
	restResult["path"] = restPath
	restResult["production_endpoint"] = telemetry.DefaultGoogleQuotaURL
	restResult["matches_expected"] = ui.ExtractDualCycleQuota(restData) == expected
	restResult["disk_reload"] = snapshot(roundTrip)
	result["F1_REST_pool_cache_badges"] = restResult
	require("F1/full_REST", ui.ExtractDualCycleQuota(restData) == expected)
	require("F1/full_REST_reload", ui.ExtractDualCycleQuota(roundTrip) == expected)
	restPayload.Store(`{"groups":[]}`)
	empty, err := pool.PollAccount(context.Background(), "empty@example.com")
	must(err)
	result["F1_empty_REST"] = snapshot(empty)
	encoded, err := json.Marshal(restData)
	must(err)
	field, present := reflect.TypeOf(*restData).FieldByName("GPT")
	result["F2_contract"] = map[string]any{"GPT_field_present": present, "GPT_json_tag": field.Tag.Get("json"), "serialized_gpt_key_present": strings.Contains(string(encoded), `"gpt":`), "expected_field_present": false}
	require("F2/Go_field_absent", !present)
	require("F2/JSON_key_absent", !strings.Contains(string(encoded), `"gpt":`))
	legacy := `{"matrix":{"legacy@example.com":{"email":"legacy@example.com","claude":{"remaining_fraction":0.2},"gemini":{"remaining_fraction":0.8},"gpt":{"remaining_fraction":0.9}}}}`
	legacyPath := filepath.Join(tmp, "legacy.json")
	must(os.WriteFile(legacyPath, []byte(legacy), 0600))
	legacyCache := telemetry.NewQuotaCache(legacyPath)
	must(legacyCache.LoadFromDisk())
	legacyData, ok := legacyCache.Get("legacy@example.com")
	if !ok {
		panic("legacy cache missing")
	}
	result["F1_legacy_cache"] = snapshot(legacyData)
	result["F1_R4_HTTP_cache_matrix"] = runQuotaMatrix(tmp)
	result["F4_R4_reflection_AST"] = retiredUISurface()

	accountStore, err := store.NewAccountStore(filepath.Join(tmp, "accounts"))
	must(err)
	token := &model.TokenData{AccessToken: "synthetic-access", RefreshToken: "synthetic-refresh"}
	for i := 1; i <= 6; i++ {
		must(accountStore.SaveAccount(fmt.Sprintf("account%02d@example.com", i), token))
	}
	mgr, err := manager.NewAccountManager(accountStore, nil)
	must(err)
	mgr.SetChromeDir(filepath.Join(tmp, "no-real-chrome"))
	mgr.SetClientFinder(func(context.Context) (*connector.Client, error) { return nil, errors.New("offline synthetic") })
	mgr.SetWincredHandlers(func() (*model.TokenData, error) {
		return &model.TokenData{AccessToken: "synthetic", IDToken: "e30." + base64.RawURLEncoding.EncodeToString([]byte(`{"email":"account01@example.com"}`)) + ".synthetic"}, nil
	}, func(*model.TokenData) error { return errors.New("real writes forbidden") })
	for i := 1; i <= 6; i++ {
		q := *full
		q.Email = fmt.Sprintf("account%02d@example.com", i)
		cache.Update(&q)
	}
	uiPool, err := telemetry.NewTelemetryPoolWithSource(fallback{}, cache, source{})
	must(err)
	var mu sync.Mutex
	var calls []string
	var errorMode atomic.Bool
	uiPool.SetLocalQuotaFetcher(func(ctx context.Context, email string) (*telemetry.AccountQuotaData, error) {
		mu.Lock()
		calls = append(calls, email)
		mu.Unlock()
		select {
		case <-time.After(50 * time.Millisecond):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		if errorMode.Load() {
			return nil, errors.New("synthetic failure")
		}
		q := parsed(`{"response":` + fullSummary + `}`)
		q.Email = email
		return q, nil
	})
	app := ui.NewAppWithFyne(fyneApp, mgr)
	app.SetPreferencesPath(filepath.Join(tmp, "preferences.json"))
	app.SetTelemetryPool(uiPool)
	app.Init()
	defer app.Close()
	waitFor("six initial cards", func() bool { g := gridOf(app.Window().Content()); return g != nil && len(g.Objects) == 6 })
	grid := gridOf(app.Window().Content())
	app.Window().Resize(fyne.NewSize(1440, 900))
	capture(app, "E06-headless-dark.png")
	var refreshResults []any
	for _, idx := range []int{0, 2, 5, 3} {
		errorMode.Store(idx == 3)
		initial := append([]fyne.CanvasObject(nil), grid.Objects...)
		gridBefore := grid
		disabledIndex := (idx + 1) % 6
		if disabledIndex == 0 {
			disabledIndex = 1
		}
		if disabledIndex == idx {
			disabledIndex = 2
		}
		disabled := buttonIn(initial[disabledIndex], false)
		disabled.Disable()
		focusIndex := (idx + 2) % 6
		if focusIndex == 0 {
			focusIndex = 1
		}
		if focusIndex == idx || focusIndex == disabledIndex {
			focusIndex = 4
		}
		focused := buttonIn(initial[focusIndex], false)
		mu.Lock()
		calls = nil
		mu.Unlock()
		start := time.Now()
		test.Tap(buttonIn(initial[idx], true))
		callbackMS := float64(time.Since(start).Microseconds()) / 1000
		app.Window().Canvas().Focus(focused)
		waitFor("target replacement", func() bool { return grid.Objects[idx] != initial[idx] })
		waitFor("request idle", func() bool { return !app.IsAnyAccountRefreshing() })
		unchanged := 0
		for i := range initial {
			if i != idx && grid.Objects[i] == initial[i] {
				unchanged++
			}
		}
		mu.Lock()
		polled := append([]string(nil), calls...)
		mu.Unlock()
		refreshResults = append(refreshResults, map[string]any{"target_index": idx, "error_path": idx == 3, "polled_emails": polled, "unchanged_siblings": unchanged, "expected_unchanged_siblings": 5, "disabled_sibling_preserved": disabled.Disabled(), "focused_sibling_preserved": app.Window().Canvas().Focused() == focused, "callback_ms": callbackMS, "mock_completion_ms": float64(time.Since(start).Microseconds()) / 1000, "target_text": texts(grid.Objects[idx])})
		require(fmt.Sprintf("F3/target_%d/siblings", idx), unchanged == 5)
		require(fmt.Sprintf("F3/target_%d/grid", idx), gridOf(app.Window().Content()) == gridBefore)
		require(fmt.Sprintf("F3/target_%d/disabled", idx), disabled.Disabled())
		require(fmt.Sprintf("F3/target_%d/focus", idx), app.Window().Canvas().Focused() == focused)
		require(fmt.Sprintf("F3/target_%d/single_email", idx), len(polled) == 1 && polled[0] == fmt.Sprintf("account%02d@example.com", idx+1))
		if idx == 3 {
			require("F3/error_badge", strings.Contains(strings.Join(texts(grid.Objects[idx]), " "), "[!] Lỗi kết nối"))
		}
		disabled.Enable()
	}
	result["F3_actual_button_refresh"] = refreshResults

	var monogram *canvas.Rectangle
	var selector *widget.Select
	walk(app.Window().Content(), func(o fyne.CanvasObject) {
		if r, ok := o.(*canvas.Rectangle); ok && r.CornerRadius == 6 {
			monogram = r
		}
		if s, ok := o.(*widget.Select); ok {
			selector = s
		}
	})
	if monogram == nil || selector == nil {
		panic("theme controls missing")
	}
	var themeResults []any
	for _, tt := range []struct {
		selection string
		expected  color.Color
	}{{"☀️ Sáng", ui.LightPalette.Primary}, {"☕ Ấm", ui.WarmPalette.Primary}, {"🌙 Tối", ui.DarkPalette.Primary}} {
		selector.SetSelected(tt.selection)
		themeResults = append(themeResults, map[string]any{"selector": tt.selection, "actual": hex(monogram.FillColor), "expected": hex(tt.expected), "matches_expected": hex(monogram.FillColor) == hex(tt.expected)})
		require("F6/"+tt.selection, hex(monogram.FillColor) == hex(tt.expected))
		if tt.selection == "☀️ Sáng" {
			capture(app, "E06-headless-light.png")
		}
	}
	result["F6_theme_selector"] = themeResults

	var layoutResults []any
	for _, count := range []int{6, 1, 2, 7, 31} {
		emails, err := accountStore.ListAccounts()
		must(err)
		for _, email := range emails {
			must(accountStore.DeleteAccount(email))
		}
		for i := 1; i <= count; i++ {
			must(accountStore.SaveAccount(fmt.Sprintf("account%02d@example.com", i), token))
		}
		app.Refresh()
		waitFor("account count and refresh completion", func() bool {
			busy := false
			walk(app.Window().Content(), func(o fyne.CanvasObject) {
				if b, ok := o.(*widget.Button); ok && b.Text == "⏳ Đang quét..." {
					busy = true
				}
			})
			return len(grid.Objects) == count && !busy
		})
		for _, size := range []fyne.Size{fyne.NewSize(1440, 900), fyne.NewSize(1200, 750)} {
			app.Window().Resize(size)
			app.Window().Canvas().Capture()
			maxH := float32(0)
			for _, card := range grid.Objects {
				if card.Size().Height > maxH {
					maxH = card.Size().Height
				}
			}
			var scroll *container.Scroll
			walk(app.Window().Content(), func(o fyne.CanvasObject) {
				if s, ok := o.(*container.Scroll); ok {
					scroll = s
				}
			})
			entry := map[string]any{"account_count": count, "canvas_size": app.Window().Canvas().Size(), "grid_size": grid.Size(), "grid_min": grid.MinSize(), "first_card_size": grid.Objects[0].Size(), "first_card_min": grid.Objects[0].MinSize(), "max_card_height": maxH, "compact_height_matches_min": maxH == grid.Objects[0].MinSize().Height, "first_card_position": grid.Objects[0].Position(), "last_card_position": grid.Objects[len(grid.Objects)-1].Position()}
			require(fmt.Sprintf("F5/%d/%v/compact", count, size), maxH >= 80 && maxH <= 84 && maxH == grid.Objects[0].MinSize().Height)
			if scroll != nil {
				entry["scroll_viewport"] = scroll.Size()
				entry["scroll_content"] = scroll.Content.Size()
				if count == 31 {
					scroll.ScrollToBottom()
					entry["bottom_scroll_offset"] = scroll.Offset
					scroll.ScrollToTop()
				}
			}
			layoutResults = append(layoutResults, entry)
		}
	}
	result["F5_natural_layout_sweep"] = layoutResults
}
