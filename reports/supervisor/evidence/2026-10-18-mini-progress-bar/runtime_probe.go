//go:build ignore

// Run explicitly from the repository root. This probe imports unchanged production
// widgets, mounts the real App on Fyne's software canvas and uses only offline data.
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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

const evidenceDir = "reports/supervisor/evidence/2026-10-18-mini-progress-bar"

var results = map[string]any{
	"reviewed_commit": "4da49ab2073ee1920d3dce47085342ccc7901acf",
	"driver":          "Fyne test.NewApp software canvas, scale 1; production App and widgets; synthetic offline accounts",
}
var checks []map[string]any
var failures []string

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func check(name string, pass bool) {
	checks = append(checks, map[string]any{"name": name, "pass": pass})
	if !pass {
		failures = append(failures, name)
	}
}
func near(a, b float32) bool { return math.Abs(float64(a-b)) < .01 }
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
	out := []string{}
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
		if c, ok := o.(*fyne.Container); ok && strings.HasSuffix(fmt.Sprintf("%T", c.Layout), ".gridLayout") {
			out = c
		}
	})
	return out
}
func rowsOf(card fyne.CanvasObject) *fyne.Container {
	return card.(*fyne.Container).Objects[1].(*fyne.Container).Objects[0].(*fyne.Container)
}
func rgba(c color.Color) string {
	n := color.NRGBAModel.Convert(c).(color.NRGBA)
	return fmt.Sprintf("#%02x%02x%02x%02x", n.R, n.G, n.B, n.A)
}
func miniBars(root fyne.CanvasObject) []*fyne.Container {
	out := []*fyne.Container{}
	walk(root, func(o fyne.CanvasObject) {
		if c, ok := o.(*fyne.Container); ok && len(c.Objects) == 2 {
			a, okA := c.Objects[0].(*canvas.Rectangle)
			b, okB := c.Objects[1].(*canvas.Rectangle)
			if okA && okB && a.CornerRadius == 3 && b.CornerRadius == 3 {
				out = append(out, c)
			}
		}
	})
	return out
}
func barFacts(bar *fyne.Container) map[string]any {
	t := bar.Objects[0].(*canvas.Rectangle)
	f := bar.Objects[1].(*canvas.Rectangle)
	return map[string]any{"track": t.Size(), "fill": f.Size(), "track_radius": t.CornerRadius, "fill_radius": f.CornerRadius, "track_color": rgba(t.FillColor), "fill_color": rgba(f.FillColor), "position": bar.Position(), "fill_position": f.Position()}
}
func matrix() {
	var observations []any
	for _, f := range []float64{-1, 0, .12, .14999, .15, .46, .5, .50001, .916, 1, 1.5} {
		row := ui.CreateQuotaCycleRow("5h:", f, f, true, false).(*fyne.Container)
		row.Resize(row.MinSize())
		bars := miniBars(row)
		prefix := fmt.Sprintf("fraction/%g", f)
		check(prefix+"/two-bars", len(bars) == 2)
		if len(bars) != 2 {
			continue
		}
		item := row.Objects[1].(*fyne.Container)
		check(prefix+"/model-bar-percentage-order", len(item.Objects) == 3 && item.Objects[0].(*canvas.Text).Text == "Claude" && item.Objects[1] == bars[0])
		bar := bars[0]
		track := bar.Objects[0].(*canvas.Rectangle)
		fill := bar.Objects[1].(*canvas.Rectangle)
		wantW := float32(math.Min(1, math.Max(0, f))) * 50
		wantColor := "#ef4444ff"
		if f < 0 {
			wantColor = "#888899ff"
		} else if f > .5 {
			wantColor = "#22c55eff"
		} else if f >= .15 {
			wantColor = "#f59e0bff"
		}
		check(prefix+"/geometry", near(track.Size().Width, 50) && near(track.Size().Height, 6) && near(fill.Size().Width, wantW) && near(fill.Size().Height, 6) && track.CornerRadius == 3 && fill.CornerRadius == 3)
		check(prefix+"/color", rgba(fill.FillColor) == wantColor)
		check(prefix+"/vertical-center", near(fill.Position().Y, (bar.Size().Height-6)/2))
		observations = append(observations, map[string]any{"fraction": f, "expected_width": wantW, "facts": barFacts(bar), "texts": texts(item)})
	}
	results["fraction_matrix"] = observations
}
func waitFor(name string, condition func() bool) {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	panic("timeout: " + name)
}
func capture(app *ui.App, name string, size fyne.Size) {
	img := app.Window().Canvas().Capture()
	check(name+"/dimensions", img.Bounds().Dx() == int(size.Width) && img.Bounds().Dy() == int(size.Height))
	f, err := os.Create(filepath.Join(evidenceDir, name))
	must(err)
	must(png.Encode(f, img))
	must(f.Close())
}
func geometry(app *ui.App, size fyne.Size) map[string]any {
	app.Window().Resize(size)
	app.Window().Canvas().Capture()
	grid := gridOf(app.Window().Content())
	prefix := fmt.Sprintf("layout/%gx%g", size.Width, size.Height)
	check(prefix+"/canvas", app.Window().Canvas().Size() == size)
	check(prefix+"/grid-width", near(grid.Size().Width, size.Width-16))
	var cards []any
	for i, card := range grid.Objects {
		rows := rowsOf(card)
		cp := fmt.Sprintf("%s/card%d", prefix, i)
		check(cp+"/height", card.Size().Height >= 80 && card.Size().Height <= 84)
		check(cp+"/three-rows", len(rows.Objects) == 3)
		var rowFacts []any
		for n, row := range rows.Objects {
			check(fmt.Sprintf("%s/row%d/fits", cp, n), row.MinSize().Width <= row.Size().Width+.01 && rows.Position().X+row.Position().X+row.Size().Width <= card.Size().Width+.01)
			if n > 0 {
				prev := rows.Objects[n-1]
				check(fmt.Sprintf("%s/row%d/no-overlap", cp, n), row.Position().Y >= prev.Position().Y+prev.Size().Height)
			}
			var bars []any
			for _, bar := range miniBars(row) {
				bars = append(bars, barFacts(bar))
			}
			rowFacts = append(rowFacts, map[string]any{"index": n, "size": row.Size(), "min": row.MinSize(), "position": row.Position(), "texts": texts(row), "bars": bars})
		}
		cards = append(cards, map[string]any{"index": i, "size": card.Size(), "min": card.MinSize(), "position": card.Position(), "rows_container": rows.Size(), "rows": rowFacts})
	}
	return map[string]any{"requested_canvas": size, "canvas": app.Window().Canvas().Size(), "content_min": app.Window().Content().MinSize(), "grid": grid.Size(), "grid_min": grid.MinSize(), "cards": cards}
}

type offlineSource struct{}

func (offlineSource) ListAccounts() ([]telemetry.AccountTokenInfo, error) { return nil, nil }
func (offlineSource) GetRefreshToken(string) (string, error)              { return "synthetic", nil }

type offlineClient struct{}

func (offlineClient) RetrieveUserQuota(context.Context, string) (*telemetry.AccountQuotaData, error) {
	return nil, errors.New("offline synthetic provider")
}

func main() {
	defer func() {
		if p := recover(); p != nil {
			failures = append(failures, fmt.Sprint(p))
		}
		results["checks"] = checks
		results["checks_total"] = len(checks)
		results["failures"] = failures
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		must(enc.Encode(results))
		if len(failures) != 0 {
			os.Exit(1)
		}
	}()
	must(os.MkdirAll(".tmp", 0755))
	tmp, err := os.MkdirTemp(".tmp", "supervisor-mini-progress-")
	must(err)
	tmp, err = filepath.Abs(tmp)
	must(err)
	tmpRoot, err := filepath.Abs(".tmp")
	must(err)
	if filepath.Dir(tmp) != tmpRoot {
		panic("cleanup path outside repository .tmp")
	}
	defer os.RemoveAll(tmp)
	fyneApp := test.NewApp()
	defer fyneApp.Quit()
	matrix()
	_, hasGPT := reflect.TypeOf(telemetry.AccountQuotaData{}).FieldByName("GPT")
	check("F2/no-GPT-field", !hasGPT)
	var legacy telemetry.AccountQuotaData
	must(json.Unmarshal([]byte(`{"gpt":{"remaining_fraction":1}}`), &legacy))
	encoded, err := json.Marshal(legacy)
	must(err)
	check("F2/no-GPT-json", !strings.Contains(strings.ToLower(string(encoded)), `"gpt"`))
	results["F2_serialized_legacy"] = string(encoded)
	bodies := []string{
		`{"claude":{"five_hour_fraction":1,"weekly_fraction":0.46},"gemini":{"five_hour_fraction":0.916,"weekly_fraction":0.12}}`,
		`{"claude":{"five_hour_fraction":0.46,"weekly_fraction":0.12},"gemini":{"five_hour_fraction":0.12,"weekly_fraction":0.916}}`,
		`{"claude":{"five_hour_fraction":0,"weekly_fraction":0},"gemini":{"five_hour_fraction":0.50,"weekly_fraction":0.15}}`,
		`{"claude":{"five_hour_fraction":0.15,"weekly_fraction":0.5},"gemini":{"five_hour_fraction":0.50001,"weekly_fraction":0.14999}}`,
		`{}`,
		`{"status":"Error","error_message":"synthetic offline failure"}`,
	}
	var quotas []*telemetry.AccountQuotaData
	accountStore, err := store.NewAccountStore(filepath.Join(tmp, "accounts"))
	must(err)
	cache := telemetry.NewQuotaCache(filepath.Join(tmp, "quota.json"))
	for i, body := range bodies {
		q := &telemetry.AccountQuotaData{}
		must(json.Unmarshal([]byte(body), q))
		q.Email = fmt.Sprintf("account%02d@example.com", i+1)
		q.LastUpdated = time.Now().UTC()
		quotas = append(quotas, q)
		cache.Update(q)
		must(accountStore.SaveAccount(q.Email, &model.TokenData{AccessToken: "synthetic", RefreshToken: "synthetic"}))
	}
	dq := ui.ExtractDualCycleQuota(quotas[0])
	check("F1/independent-cycles", dq.Claude5h == 1 && dq.ClaudeWeek == .46 && dq.Gemini5h == .916 && dq.GeminiWeek == .12)
	mgr, err := manager.NewAccountManager(accountStore, nil)
	must(err)
	mgr.SetChromeDir(filepath.Join(tmp, "no-real-chrome"))
	mgr.SetClientFinder(func(context.Context) (*connector.Client, error) { return nil, errors.New("offline synthetic") })
	mgr.SetWincredHandlers(func() (*model.TokenData, error) {
		return &model.TokenData{AccessToken: "synthetic", IDToken: "e30." + base64.RawURLEncoding.EncodeToString([]byte(`{"email":"account01@example.com"}`)) + ".synthetic"}, nil
	}, func(*model.TokenData) error { return errors.New("real credential writes forbidden") })
	pool, err := telemetry.NewTelemetryPoolWithSource(offlineClient{}, cache, offlineSource{})
	must(err)
	var calls atomic.Int32
	pool.SetLocalQuotaFetcher(func(ctx context.Context, email string) (*telemetry.AccountQuotaData, error) {
		calls.Add(1)
		q := *quotas[0]
		q.Email = email
		q.Claude.FiveHourFraction = .46
		q.Gemini.FiveHourFraction = .12
		return &q, nil
	})
	app := ui.NewAppWithFyne(fyneApp, mgr)
	app.SetPreferencesPath(filepath.Join(tmp, "preferences.json"))
	app.SetTelemetryPool(pool)
	app.Init()
	defer app.Close()
	waitFor("six initial cards", func() bool {
		g := gridOf(app.Window().Content())
		busy := false
		walk(app.Window().Content(), func(o fyne.CanvasObject) {
			if b, ok := o.(*widget.Button); ok && b.Text == "⏳ Đang quét..." {
				busy = true
			}
		})
		return g != nil && len(g.Objects) == 6 && !busy
	})
	check("window/default-exact", app.Window().Canvas().Size() == fyne.NewSize(1008, 630))
	check("window/min-exact", app.Window().Content().MinSize() == fyne.NewSize(840, 525))
	check("window/resizable", !app.Window().FixedSize())
	results["default_geometry"] = geometry(app, fyne.NewSize(1008, 630))
	capture(app, "E06-headless-dark-1008x630.png", fyne.NewSize(1008, 630))
	grid := gridOf(app.Window().Content())
	check("missing/four-muted-empty-mini-bars", len(miniBars(grid.Objects[4])) == 4)
	check("error/four-muted-empty-mini-bars", len(miniBars(grid.Objects[5])) == 4)
	results["error_card_texts"] = texts(grid.Objects[5])
	results["error_card_bar_count"] = len(miniBars(grid.Objects[5]))
	results["minimum_geometry"] = geometry(app, fyne.NewSize(840, 525))
	capture(app, "E07-headless-dark-840x525.png", fyne.NewSize(840, 525))
	app.Window().Resize(fyne.NewSize(1008, 630))
	before := append([]fyne.CanvasObject(nil), grid.Objects...)
	var refresh *widget.Button
	walk(before[0], func(o fyne.CanvasObject) {
		if b, ok := o.(*widget.Button); ok && b.Text == "" && b.Icon == theme.ViewRefreshIcon() {
			refresh = b
		}
	})
	check("F3/refresh-control-present", refresh != nil)
	test.Tap(refresh)
	waitFor("target card replacement", func() bool { return grid.Objects[0] != before[0] && !app.IsAnyAccountRefreshing() })
	unchanged := 0
	for i := 1; i < 6; i++ {
		if grid.Objects[i] == before[i] {
			unchanged++
		}
	}
	check("F3/five-identical-siblings", unchanged == 5)
	check("F3/one-provider-call", calls.Load() == 1)
	app.Window().Canvas().Capture()
	bars := miniBars(grid.Objects[0])
	check("F3/updated-fill-widths", len(bars) == 4 && near(bars[0].Objects[1].Size().Width, 23) && near(bars[1].Objects[1].Size().Width, 6))
	results["F3"] = map[string]any{"unchanged_siblings": unchanged, "provider_calls": calls.Load(), "texts": texts(grid.Objects[0])}
	capture(app, "E08-headless-after-single-refresh.png", fyne.NewSize(1008, 630))
	app.SetTheme(ui.ThemeLight)
	app.Window().Canvas().Capture()
	var monogram *canvas.Rectangle
	walk(app.Window().Content(), func(o fyne.CanvasObject) {
		if c, ok := o.(*fyne.Container); ok && len(c.Objects) == 2 {
			if r, ok := c.Objects[0].(*canvas.Rectangle); ok && r.CornerRadius == 6 && strings.Contains(strings.Join(texts(c), " "), "AS") {
				monogram = r
			}
		}
	})
	check("F6/light-monogram-2563eb", monogram != nil && rgba(monogram.FillColor) == "#2563ebff")
	if monogram != nil {
		results["F6_light_monogram"] = rgba(monogram.FillColor)
	}
	capture(app, "E09-headless-light-1008x630.png", fyne.NewSize(1008, 630))
}
