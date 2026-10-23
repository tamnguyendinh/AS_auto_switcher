//go:build ignore

// Independent, build-excluded supervisor reproduction. Run from the repo root:
// go run reports/supervisor/evidence/2026-10-18-ui-redesign/runtime_probe.go
// Only synthetic accounts, loopback HTTP, and .tmp state are used.
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
	"strings"
	"sync"
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

const summary = `{"groups":[{"displayName":"Gemini Models","buckets":[{"bucketId":"gemini-5h","window":"5h","remainingFraction":0.88},{"bucketId":"gemini-weekly","window":"weekly","remainingFraction":0.90}]},{"displayName":"Claude and GPT models","buckets":[{"bucketId":"3p-5h","window":"5h","remainingFraction":0.46},{"bucketId":"3p-weekly","window":"weekly","remainingFraction":0.12}]}]}`

type fakeRefresher struct{}

func (fakeRefresher) RefreshToken(context.Context, string) (*connector.TokenRefreshResult, error) {
	return &connector.TokenRefreshResult{AccessToken: "synthetic-token"}, nil
}

type fakeSource struct{}

func (fakeSource) ListAccounts() ([]telemetry.AccountTokenInfo, error) { return nil, nil }
func (fakeSource) GetRefreshToken(string) (string, error)              { return "synthetic-refresh", nil }

type unusedClient struct{}

func (unusedClient) RetrieveUserQuota(context.Context, string) (*telemetry.AccountQuotaData, error) {
	return nil, errors.New("unexpected fallback; probe requires local fake fetcher")
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

func gridOf(root fyne.CanvasObject) *fyne.Container {
	var grid *fyne.Container
	walk(root, func(o fyne.CanvasObject) {
		if c, ok := o.(*fyne.Container); ok && len(c.Objects) == 6 && strings.Contains(fmt.Sprintf("%T", c.Layout), "gridLayout") {
			grid = c
		}
	})
	return grid
}

func waitFor(label string, ready func() bool) {
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		if ready() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	panic("timeout waiting for " + label)
}

func savePNG(path string, app *ui.App) {
	f, err := os.Create(path)
	must(err)
	must(png.Encode(f, app.Window().Canvas().Capture()))
	must(f.Close())
}

func hex(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
}

func main() {
	result := map[string]any{"driver": "Fyne test.NewApp headless; not a native desktop screenshot"}
	defer func() {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		must(encoder.Encode(result))
	}()
	wrapped := []byte(`{"response":` + summary + `}`)
	c, g, p, reset, status, err := telemetry.ParseQuotaSummary(wrapped)
	must(err)
	localData := &telemetry.AccountQuotaData{Email: "account1@example.com", Claude: c, Gemini: g, GPT: p, ResetTime: reset, Status: status}
	expected := ui.DualCycleQuota{Claude5h: 0.46, ClaudeWeek: 0.12, Gemini5h: 0.88, GeminiWeek: 0.90, HasData: true}
	actual := ui.ExtractDualCycleQuota(localData)
	encodedData, err := json.Marshal(localData)
	must(err)
	result["connect_rpc"] = map[string]any{
		"expected": expected, "actual": actual, "matches_expected": actual == expected,
		"raw_bucket_count": len(localData.RawBuckets), "serialized_gpt_field_present": strings.Contains(string(encodedData), `"gpt":`),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, summary)
	}))
	defer server.Close()
	client := telemetry.NewQuotaClientWithEndpoint(fakeRefresher{}, server.URL, server.Client())
	client.SetUserAgent("supervisor-synthetic-probe")
	cloudData, err := client.RetrieveUserQuota(context.Background(), "synthetic-refresh")
	must(err)
	result["cloud_summary_payload"] = map[string]any{
		"actual": ui.ExtractDualCycleQuota(cloudData), "expected": expected,
		"raw_bucket_count": len(cloudData.RawBuckets), "production_endpoint": telemetry.DefaultGoogleQuotaURL,
	}

	tmp, err := os.MkdirTemp(".tmp", "supervisor-ui-probe-")
	must(err)
	tmp, err = filepath.Abs(tmp)
	must(err)
	tmpRoot, err := filepath.Abs(".tmp")
	must(err)
	if filepath.Dir(tmp) != tmpRoot {
		panic("temporary cleanup target escaped workspace .tmp")
	}
	defer os.RemoveAll(tmp)
	accountStore, err := store.NewAccountStore(filepath.Join(tmp, "accounts"))
	must(err)
	for i := 1; i <= 6; i++ {
		email := fmt.Sprintf("account%d@example.com", i)
		if i == 2 {
			email = "very.long.enterprise.account.name@antigravity.example.com"
		}
		must(accountStore.SaveAccount(email, &model.TokenData{AccessToken: "synthetic-access", RefreshToken: "synthetic-refresh"}))
	}
	mgr, err := manager.NewAccountManager(accountStore, nil)
	must(err)
	mgr.SetChromeDir(filepath.Join(tmp, "no-real-chrome"))
	mgr.SetClientFinder(func(context.Context) (*connector.Client, error) { return nil, errors.New("probe offline") })
	mgr.SetWincredHandlers(func() (*model.TokenData, error) {
		return &model.TokenData{AccessToken: "synthetic-access", RefreshToken: "synthetic-refresh", IDToken: "e30." + base64.RawURLEncoding.EncodeToString([]byte(`{"email":"account1@example.com"}`)) + ".synthetic"}, nil
	}, func(*model.TokenData) error { return errors.New("probe must never write real credentials") })
	cache := telemetry.NewQuotaCache(filepath.Join(tmp, "quota.json"))
	emails, err := accountStore.ListAccounts()
	must(err)
	for _, email := range emails {
		cache.Update(&telemetry.AccountQuotaData{Email: email, Claude: telemetry.ModelQuotaBucket{RemainingFraction: 1}, Gemini: telemetry.ModelQuotaBucket{RemainingFraction: 0.88}, Status: telemetry.StatusPlentiful})
	}
	pool, err := telemetry.NewTelemetryPoolWithSource(unusedClient{}, cache, fakeSource{})
	must(err)
	var callsMu sync.Mutex
	var polledEmails []string
	pool.SetLocalQuotaFetcher(func(ctx context.Context, email string) (*telemetry.AccountQuotaData, error) {
		callsMu.Lock()
		polledEmails = append(polledEmails, email)
		callsMu.Unlock()
		select {
		case <-time.After(50 * time.Millisecond):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return &telemetry.AccountQuotaData{Email: email, Claude: telemetry.ModelQuotaBucket{RemainingFraction: 0.46}, Gemini: telemetry.ModelQuotaBucket{RemainingFraction: 0.23}, Status: telemetry.StatusPlentiful}, nil
	})
	fyneApp := test.NewApp()
	defer fyneApp.Quit()
	app := ui.NewAppWithFyne(fyneApp, mgr)
	app.SetPreferencesPath(filepath.Join(tmp, "preferences.json"))
	app.SetTelemetryPool(pool)
	app.Init()
	defer app.Close()
	waitFor("six initial cards", func() bool { return gridOf(app.Window().Content()) != nil })
	grid := gridOf(app.Window().Content())
	app.Window().Resize(fyne.NewSize(1440, 900))
	savePNG("reports/supervisor/evidence/2026-10-18-ui-redesign/headless-dark.png", app)
	initial := append([]fyne.CanvasObject(nil), grid.Objects...)
	var refresh *widget.Button
	walk(initial[0], func(o fyne.CanvasObject) {
		if b, ok := o.(*widget.Button); ok && b.Icon == theme.ViewRefreshIcon() {
			refresh = b
		}
	})
	if refresh == nil {
		panic("no card refresh button")
	}
	started := time.Now()
	refresh.OnTapped()
	callbackDuration := time.Since(started)
	waitFor("single refresh UI update", func() bool {
		return len(grid.Objects) == len(initial) && grid.Objects[0] != initial[0]
	})
	completionDuration := time.Since(started)
	unchanged := 0
	for i := 1; i < len(initial); i++ {
		if grid.Objects[i] == initial[i] {
			unchanged++
		}
	}
	callsMu.Lock()
	result["single_card_refresh"] = map[string]any{
		"polled_emails": append([]string(nil), polledEmails...), "expected_unchanged_sibling_cards": 5,
		"actual_unchanged_sibling_cards": unchanged, "callback_ms": float64(callbackDuration.Microseconds()) / 1000,
		"mock_completion_ms": float64(completionDuration.Microseconds()) / 1000,
		"latency_scope":      "synthetic fetcher sleeps 50ms; no claim about real provider latency",
	}
	callsMu.Unlock()
	var monogram *canvas.Rectangle
	walk(app.Window().Content(), func(o fyne.CanvasObject) {
		if c, ok := o.(*fyne.Container); ok && len(c.Objects) == 2 {
			if r, ok := c.Objects[0].(*canvas.Rectangle); ok && r.CornerRadius == 6 {
				monogram = r
			}
		}
	})
	if monogram == nil {
		panic("no AS monogram rectangle")
	}
	beforeColor := hex(monogram.FillColor)
	var selector *widget.Select
	walk(app.Window().Content(), func(o fyne.CanvasObject) {
		if s, ok := o.(*widget.Select); ok {
			selector = s
		}
	})
	if selector == nil {
		panic("no theme selector")
	}
	selector.SetSelected("☀️ Sáng")
	result["theme_monogram"] = map[string]any{"before": beforeColor, "after_light": hex(monogram.FillColor), "expected_light": hex(ui.LightPalette.Primary)}
	savePNG("reports/supervisor/evidence/2026-10-18-ui-redesign/headless-light.png", app)
	result["window"] = map[string]any{
		"fixed_size": app.Window().FixedSize(), "content_min": app.Window().Content().MinSize(),
		"headless_canvas_size": app.Window().Canvas().Size(), "grid_card_count": len(grid.Objects),
		"grid_size": grid.Size(), "grid_min": grid.MinSize(),
		"first_card_size": grid.Objects[0].Size(), "first_card_min": grid.Objects[0].MinSize(),
	}
}
