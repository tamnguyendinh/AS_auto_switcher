//go:build ignore

// Reproduce from the repository root with go run <this file>.
// Imports production UI; uses Fyne's software canvas, synthetic accounts and .tmp storage.
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"image/color"
	"image/png"
	"math"
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

const evidenceDir = "reports/supervisor/evidence/2026-10-18-ui-scaling-70-percent"
const reviewedCommit = "92b324542302f2fe311b886650d89ffbfef78d1b"

var results = map[string]any{"reviewed_commit": reviewedCommit, "driver": "Fyne test.NewApp software canvas, scale 1; synthetic offline accounts; no native desktop capture"}
var assertions []map[string]any
var failures []string

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func require(name string, pass bool) {
	assertions = append(assertions, map[string]any{"name": name, "pass": pass})
	if !pass {
		failures = append(failures, name)
	}
}
func closeTo(a, b float32) bool { return math.Abs(float64(a-b)) < 0.1 }
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
func buttonIn(o fyne.CanvasObject, refresh bool) *widget.Button {
	var out *widget.Button
	walk(o, func(o fyne.CanvasObject) {
		if b, ok := o.(*widget.Button); ok && ((refresh && b.Text == "" && b.Icon == theme.ViewRefreshIcon()) || (!refresh && b.Text == "Chuyển")) {
			out = b
		}
	})
	if out == nil {
		panic("required card button missing")
	}
	return out
}
func waitFor(label string, condition func() bool) {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	panic("timeout: " + label)
}
func hex(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
}
func capture(app *ui.App, name string, width, height int) {
	img := app.Window().Canvas().Capture()
	require("image/"+name+"/dimensions", img.Bounds().Dx() == width && img.Bounds().Dy() == height)
	f, err := os.Create(filepath.Join(evidenceDir, name))
	must(err)
	must(png.Encode(f, img))
	must(f.Close())
}
func quota(body string) *telemetry.AccountQuotaData {
	q := &telemetry.AccountQuotaData{}
	must(json.Unmarshal([]byte(body), q))
	q.LastUpdated = time.Now().UTC()
	return q
}
func tuple(c5, cw, g5, gw float64) ui.DualCycleQuota {
	return ui.DualCycleQuota{Claude5h: c5, ClaudeWeek: cw, Gemini5h: g5, GeminiWeek: gw, HasData: c5 >= 0 || cw >= 0 || g5 >= 0 || gw >= 0}
}
func label(family string, fraction float64) string {
	if fraction < 0 {
		return family + " --%"
	}
	return fmt.Sprintf("%s %g%%", family, fraction*100)
}
func cardRows(card fyne.CanvasObject) *fyne.Container {
	return card.(*fyne.Container).Objects[1].(*fyne.Container).Objects[0].(*fyne.Container)
}
func quotaMatrix(tmp string) []*telemetry.AccountQuotaData {
	cases := []struct {
		name, body string
		want       ui.DualCycleQuota
	}{
		{"independent", `{"claude":{"five_hour_fraction":0.46,"weekly_fraction":0.12},"gemini":{"five_hour_fraction":0.88,"weekly_fraction":0.90}}`, tuple(.46, .12, .88, .90)},
		{"zero-wins-positive-legacy", `{"claude":{"remaining_fraction":0.2,"five_hour_fraction":0,"weekly_fraction":0},"gemini":{"remaining_fraction":0.8,"five_hour_fraction":0,"weekly_fraction":0}}`, tuple(0, 0, 0, 0)},
		{"missing", `{}`, tuple(-1, -1, -1, -1)},
		{"null", `{"claude":null,"gemini":{"five_hour_fraction":null,"weekly_fraction":null}}`, tuple(-1, -1, -1, -1)},
		{"partial", `{"claude":{"five_hour_fraction":0,"weekly_fraction":-1},"gemini":{"five_hour_fraction":-1,"weekly_fraction":0.90}}`, tuple(0, -1, -1, .90)},
		{"maximum-width", `{"claude":{"five_hour_fraction":1,"weekly_fraction":1},"gemini":{"five_hour_fraction":1,"weekly_fraction":1}}`, tuple(1, 1, 1, 1)},
		{"legacy", `{"claude":{"remaining_fraction":0.35},"gemini":{"remaining_fraction":0.65}}`, tuple(.35, -1, .65, -1)},
		{"zero-wins-raw", `{"claude":{"remaining_fraction":0.2,"five_hour_fraction":0,"weekly_fraction":0},"raw_buckets":[{"model_id":"3p-5h","remaining_fraction":1},{"model_id":"3p-weekly","remaining_fraction":1}]}`, tuple(0, 0, -1, -1)},
	}
	var observations []any
	var quotas []*telemetry.AccountQuotaData
	for i, tc := range cases {
		q := quota(tc.body)
		q.Email = fmt.Sprintf("account%02d@example.com", i+1)
		if i == 2 {
			q.Email = "account03.very.long.enterprise.account.with.subdomains@company.example"
		}
		quotas = append(quotas, q)
		cachePath := filepath.Join(tmp, tc.name+".json")
		for stage := 0; stage < 3; stage++ {
			actual := ui.ExtractDualCycleQuota(q)
			card := ui.NewAccountCardWithQuota(manager.AccountDisplayItem{Email: q.Email}, q, ui.DarkPalette, nil, nil, nil)
			rows := cardRows(card)
			want5 := []string{"5h:", label("Claude", tc.want.Claude5h), label("Gemini", tc.want.Gemini5h)}
			wantW := []string{"week:", label("Claude", tc.want.ClaudeWeek), label("Gemini", tc.want.GeminiWeek)}
			prefix := fmt.Sprintf("F1/%s/stage%d", tc.name, stage)
			require(prefix+"/tuple", actual == tc.want)
			require(prefix+"/5h", reflect.DeepEqual(texts(rows.Objects[1]), want5))
			require(prefix+"/week", reflect.DeepEqual(texts(rows.Objects[2]), wantW))
			observations = append(observations, map[string]any{"case": tc.name, "stage": stage, "actual": actual, "expected": tc.want, "5h": texts(rows.Objects[1]), "week": texts(rows.Objects[2])})
			if stage < 2 {
				cache := telemetry.NewQuotaCache(cachePath)
				cache.Update(q)
				must(cache.SaveToDisk())
				reloaded := telemetry.NewQuotaCache(cachePath)
				must(reloaded.LoadFromDisk())
				var ok bool
				q, ok = reloaded.Get(q.Email)
				if !ok {
					panic("quota cache entry missing")
				}
			}
		}
	}
	_, gptPresent := reflect.TypeOf(telemetry.AccountQuotaData{}).FieldByName("GPT")
	legacy := quota(`{"email":"legacy@example.com","gpt":{"remaining_fraction":1},"claude":{"five_hour_fraction":0}}`)
	encoded, err := json.Marshal(legacy)
	must(err)
	require("F2/Go-field-absent", !gptPresent)
	require("F2/serialization-clean", !strings.Contains(strings.ToLower(string(encoded)), `"gpt"`))
	results["F1_quota_matrix"] = observations
	results["F2_contract"] = map[string]any{"GPT_field_present": gptPresent, "serialized_legacy_input": string(encoded)}
	return quotas
}
func retiredHelpers() {
	names := map[string]bool{"BuildActiveQuotaSummary": false, "NewQuotaBadgeGroup": false, "NewDualCycleQuotaBadge": false, "NewAccountCard": false, "RefreshSingleQuota": false}
	paths, err := filepath.Glob("internal/ui/*.go")
	must(err)
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		must(err)
		for _, decl := range f.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok {
				if _, wanted := names[fn.Name.Name]; wanted {
					names[fn.Name.Name] = true
				}
			}
		}
	}
	for name, present := range names {
		require("F4/retired/"+name, !present)
	}
	results["F4_retired_helpers_present"] = names
}
func geometry(app *ui.App, count int, size fyne.Size) map[string]any {
	app.Window().Resize(size)
	app.Window().Canvas().Capture()
	grid := gridOf(app.Window().Content())
	prefix := fmt.Sprintf("F5/%d/%g", count, size.Width)
	entry := map[string]any{"count": count, "requested_canvas": size, "actual_canvas": app.Window().Canvas().Size(), "root_min": app.Window().Content().MinSize(), "grid_size": grid.Size(), "card_size": grid.Objects[0].Size(), "last_position": grid.Objects[len(grid.Objects)-1].Position()}
	require(prefix+"/canvas", app.Window().Canvas().Size() == size)
	require(prefix+"/natural-grid-width", closeTo(grid.Size().Width, size.Width-16))
	var rowGeometry []any
	for i, card := range grid.Objects {
		rows := cardRows(card)
		require(fmt.Sprintf("%s/card%d/height", prefix, i), card.Size().Height >= 80 && card.Size().Height <= 84)
		require(fmt.Sprintf("%s/card%d/width", prefix, i), closeTo(card.Size().Width, (grid.Size().Width-8)/3))
		require(fmt.Sprintf("%s/card%d/column", prefix, i), closeTo(card.Position().X, float32(i%3)*(card.Size().Width+4)))
		require(fmt.Sprintf("%s/card%d/rows", prefix, i), len(rows.Objects) == 3)
		for j, row := range rows.Objects {
			if j > 0 {
				require(fmt.Sprintf("%s/card%d/row%d/fits", prefix, i, j), row.MinSize().Width <= row.Size().Width+.1)
			}
			if j > 0 {
				previous := rows.Objects[j-1]
				require(fmt.Sprintf("%s/card%d/row%d/no-overlap", prefix, i, j), row.Position().Y+.1 >= previous.Position().Y+previous.Size().Height)
			}
		}
		top := rows.Objects[0].(*fyne.Container)
		var email fyne.CanvasObject
		var actions *fyne.Container
		for _, o := range top.Objects {
			if strings.Contains(fmt.Sprintf("%T", o), "emailLabelWithTooltip") {
				email = o
			}
			if c, ok := o.(*fyne.Container); ok {
				actions = c
			}
		}
		require(fmt.Sprintf("%s/card%d/email-actions", prefix, i), email != nil && actions != nil && email.Position().X+email.Size().Width <= actions.Position().X+.1)
		if i < 6 {
			rowGeometry = append(rowGeometry, map[string]any{"card": i, "size": card.Size(), "row1": rows.Objects[0].Size(), "5h_min": rows.Objects[1].MinSize(), "week_min": rows.Objects[2].MinSize(), "5h_text": texts(rows.Objects[1]), "week_text": texts(rows.Objects[2])})
		}
	}
	entry["rows"] = rowGeometry
	root := app.Window().Content().(*fyne.Container)
	main := root.Objects[0].(*fyne.Container)
	headerFound := false
	for _, o := range main.Objects {
		if c, ok := o.(*fyne.Container); ok && len(c.Objects) == 4 {
			headerFound = true
			var headers []any
			for i, row := range c.Objects[:3] {
				require(fmt.Sprintf("%s/header%d/fits", prefix, i), row.MinSize().Width <= row.Size().Width+.1)
				headers = append(headers, map[string]any{"row": i, "size": row.Size(), "min": row.MinSize()})
			}
			entry["header_rows"] = headers
		}
	}
	require(prefix+"/header-found", headerFound)
	return entry
}

type source struct{}

func (source) ListAccounts() ([]telemetry.AccountTokenInfo, error) { return nil, nil }
func (source) GetRefreshToken(string) (string, error)              { return "synthetic", nil }

type unavailable struct{}

func (unavailable) RetrieveUserQuota(context.Context, string) (*telemetry.AccountQuotaData, error) {
	return nil, errors.New("synthetic unavailable provider")
}

func main() {
	defer func() {
		if p := recover(); p != nil {
			failures = append(failures, fmt.Sprint(p))
		}
		results["assertions"] = assertions
		results["assertions_total"] = len(assertions)
		results["failures"] = failures
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		must(encoder.Encode(results))
		if len(failures) > 0 {
			os.Exit(1)
		}
	}()
	must(os.MkdirAll(".tmp", 0755))
	tmp, err := os.MkdirTemp(".tmp", "supervisor-ui-scaling-")
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
	quotas := quotaMatrix(tmp)
	retiredHelpers()
	accountStore, err := store.NewAccountStore(filepath.Join(tmp, "accounts"))
	must(err)
	token := &model.TokenData{AccessToken: "synthetic", RefreshToken: "synthetic"}
	for i := 0; i < 6; i++ {
		must(accountStore.SaveAccount(quotas[i].Email, token))
	}
	mgr, err := manager.NewAccountManager(accountStore, nil)
	must(err)
	mgr.SetChromeDir(filepath.Join(tmp, "no-real-chrome"))
	mgr.SetClientFinder(func(context.Context) (*connector.Client, error) { return nil, errors.New("offline synthetic") })
	mgr.SetWincredHandlers(func() (*model.TokenData, error) {
		return &model.TokenData{AccessToken: "synthetic", IDToken: "e30." + base64.RawURLEncoding.EncodeToString([]byte(`{"email":"account01@example.com"}`)) + ".synthetic"}, nil
	}, func(*model.TokenData) error { return errors.New("real writes forbidden") })
	cache := telemetry.NewQuotaCache(filepath.Join(tmp, "ui-quota.json"))
	for _, q := range quotas[:6] {
		cache.Update(q)
	}
	pool, err := telemetry.NewTelemetryPoolWithSource(unavailable{}, cache, source{})
	must(err)
	var mu sync.Mutex
	var calls []string
	var errorMode atomic.Bool
	pool.SetLocalQuotaFetcher(func(ctx context.Context, email string) (*telemetry.AccountQuotaData, error) {
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
		q := *quotas[0]
		q.Email = email
		return &q, nil
	})
	app := ui.NewAppWithFyne(fyneApp, mgr)
	app.SetPreferencesPath(filepath.Join(tmp, "preferences.json"))
	app.SetTelemetryPool(pool)
	app.Init()
	defer app.Close()
	waitFor("initial account refresh", func() bool {
		g := gridOf(app.Window().Content())
		if g == nil || len(g.Objects) != 6 {
			return false
		}
		busy := false
		walk(app.Window().Content(), func(o fyne.CanvasObject) {
			if b, ok := o.(*widget.Button); ok && b.Text == "⏳ Đang quét..." {
				busy = true
			}
		})
		return !busy
	})
	require("window/default-exact", app.Window().Canvas().Size() == fyne.NewSize(1008, 630))
	require("window/content-min-exact", app.Window().Content().MinSize() == fyne.NewSize(840, 525))
	require("window/resizable", !app.Window().FixedSize())
	results["initial_geometry"] = geometry(app, 6, fyne.NewSize(1008, 630))
	capture(app, "E06-headless-dark-1008x630.png", 1008, 630)
	grid := gridOf(app.Window().Content())
	var refreshResults []any
	for _, idx := range []int{0, 2, 5, 3} {
		errorMode.Store(idx == 3)
		initial := append([]fyne.CanvasObject(nil), grid.Objects...)
		disabledIdx, focusIdx := 1, 4
		disabled := buttonIn(initial[disabledIdx], false)
		disabled.Disable()
		focused := buttonIn(initial[focusIdx], false)
		mu.Lock()
		calls = nil
		mu.Unlock()
		started := time.Now()
		test.Tap(buttonIn(initial[idx], true))
		callbackMS := float64(time.Since(started).Microseconds()) / 1000
		app.Window().Canvas().Focus(focused)
		waitFor("target replacement", func() bool { return grid.Objects[idx] != initial[idx] })
		waitFor("refresh idle", func() bool { return !app.IsAnyAccountRefreshing() })
		unchanged := 0
		for i := range initial {
			if i != idx && initial[i] == grid.Objects[i] {
				unchanged++
			}
		}
		mu.Lock()
		polled := append([]string(nil), calls...)
		mu.Unlock()
		prefix := fmt.Sprintf("F3/target%d", idx)
		require(prefix+"/five-siblings", unchanged == 5)
		require(prefix+"/grid", gridOf(app.Window().Content()) == grid)
		require(prefix+"/disable", disabled.Disabled())
		require(prefix+"/focus", app.Window().Canvas().Focused() == focused)
		require(prefix+"/one-email", len(polled) == 1 && polled[0] == quotas[idx].Email)
		if idx == 3 {
			require(prefix+"/error", strings.Contains(strings.Join(texts(grid.Objects[idx]), " "), "[!] Lỗi kết nối"))
		}
		refreshResults = append(refreshResults, map[string]any{"target": idx, "error_path": idx == 3, "unchanged_siblings": unchanged, "disabled_preserved": disabled.Disabled(), "focused_preserved": app.Window().Canvas().Focused() == focused, "polled": polled, "callback_ms": callbackMS, "completion_ms": float64(time.Since(started).Microseconds()) / 1000})
		disabled.Enable()
	}
	results["F3_button_refresh"] = refreshResults
	for _, q := range quotas[:6] {
		cache.Update(q)
	}
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
	for _, tc := range []struct{ choice, want string }{{"☀️ Sáng", "#2563eb"}, {"☕ Ấm", "#d97706"}, {"🌙 Tối", "#89b4fa"}} {
		selector.SetSelected(tc.choice)
		require("F6/"+tc.choice, hex(monogram.FillColor) == tc.want)
		themeResults = append(themeResults, map[string]any{"choice": tc.choice, "actual": hex(monogram.FillColor), "expected": tc.want})
		if tc.choice == "☀️ Sáng" {
			capture(app, "E06-headless-light-1008x630.png", 1008, 630)
		}
	}
	results["F6_theme_selector"] = themeResults
	var layoutResults []any
	for _, count := range []int{6, 1, 2, 7, 31} {
		emails, err := accountStore.ListAccounts()
		must(err)
		for _, email := range emails {
			must(accountStore.DeleteAccount(email))
		}
		for i := 1; i <= count; i++ {
			email := fmt.Sprintf("account%02d@example.com", i)
			if i <= 6 {
				email = quotas[i-1].Email
			}
			must(accountStore.SaveAccount(email, token))
		}
		app.Refresh()
		waitFor("layout account refresh", func() bool {
			busy := false
			walk(app.Window().Content(), func(o fyne.CanvasObject) {
				if b, ok := o.(*widget.Button); ok && b.Text == "⏳ Đang quét..." {
					busy = true
				}
			})
			return len(grid.Objects) == count && !busy
		})
		for _, size := range []fyne.Size{fyne.NewSize(1008, 630), fyne.NewSize(840, 525)} {
			layoutResults = append(layoutResults, geometry(app, count, size))
			if count == 6 && size.Width == 840 {
				capture(app, "E07-headless-dark-840x525.png", 840, 525)
			}
		}
	}
	results["F5_layout_sweep"] = layoutResults
}
