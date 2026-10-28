//go:build ignore

// Run from repo root: go run reports/supervisor/evidence/2026-10-18-ui-redesign-r4/zero_cycle_card_probe.go
// Focused F1 reproduction through public parser/cache/card APIs. Synthetic data only.
package main

import (
	"encoding/json"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"

	"antigravity-switch/internal/engines/antigravity/manager"
	"antigravity-switch/internal/engines/antigravity/telemetry"
	"antigravity-switch/internal/ui"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func cardTexts(o fyne.CanvasObject) []string {
	var result []string
	if text, ok := o.(*canvas.Text); ok {
		result = append(result, text.Text)
	}
	if group, ok := o.(*fyne.Container); ok {
		for _, child := range group.Objects {
			result = append(result, cardTexts(child)...)
		}
	}
	return result
}

func main() {
	failures := 0
	// Register exit before cleanup defers so failed probes still remove temporary storage.
	defer func() {
		if failures > 0 {
			os.Exit(1)
		}
	}()
	const output = "reports/supervisor/evidence/2026-10-18-ui-redesign-r4"
	tmp, err := os.MkdirTemp(".tmp", "supervisor-r4-zero-card-")
	check(err)
	tmp, err = filepath.Abs(tmp)
	check(err)
	root, err := filepath.Abs(".tmp")
	check(err)
	if filepath.Dir(tmp) != root {
		panic("cleanup target outside workspace .tmp")
	}
	defer os.RemoveAll(tmp)
	app := test.NewApp()
	defer app.Quit()
	app.Settings().SetTheme(ui.NewThemeInstance(ui.ThemeLight))
	const both = `{"groups":[{"displayName":"Claude models","buckets":[{"bucketId":"opaque-a","window":"5h","remainingFraction":0},{"bucketId":"opaque-b","window":"weekly","remainingFraction":0}]}]}`
	const weekly = `{"groups":[{"displayName":"Claude models","buckets":[{"bucketId":"opaque-b","window":"weekly","remainingFraction":0}]}]}`
	cases := []struct {
		name, payload string
		retainRaw     bool
		want          [4]float64
	}{
		{"claude-both-zero-no-raw", both, false, [4]float64{0, 0, -1, -1}},
		{"claude-both-zero-with-raw", both, true, [4]float64{0, 0, -1, -1}},
		{"claude-weekly-zero-no-raw", weekly, false, [4]float64{-1, 0, -1, -1}},
		{"gemini-both-zero-no-raw", strings.ReplaceAll(both, "Claude models", "Gemini Models"), false, [4]float64{-1, -1, 0, 0}},
	}
	var cards []fyne.CanvasObject
	var results []map[string]any
	for _, tc := range cases {
		c, g, _, reset, status, err := telemetry.ParseQuotaSummary([]byte(tc.payload))
		check(err)
		q := &telemetry.AccountQuotaData{Email: tc.name + "@example.com", Claude: c, Gemini: g, ResetTime: reset, Status: status, LastUpdated: time.Now().UTC()}
		if tc.retainRaw {
			_, _, _, q.RawBuckets, _, _, err = telemetry.ParseQuotaSummaryWithBuckets([]byte(tc.payload))
			check(err)
		}
		cache := telemetry.NewQuotaCache(filepath.Join(tmp, tc.name+".json"))
		cache.Update(q)
		check(cache.SaveToDisk())
		reloaded := telemetry.NewQuotaCache(cache.FilePath())
		check(reloaded.LoadFromDisk())
		persisted, ok := reloaded.Get(q.Email)
		if !ok {
			panic("cache missing")
		}
		dq := ui.ExtractDualCycleQuota(persisted)
		actual := [4]float64{dq.Claude5h, dq.ClaudeWeek, dq.Gemini5h, dq.GeminiWeek}
		storage := [4]float64{persisted.Claude.FiveHourFraction, persisted.Claude.WeeklyFraction, persisted.Gemini.FiveHourFraction, persisted.Gemini.WeeklyFraction}
		card := ui.NewAccountCardWithQuota(manager.AccountDisplayItem{Email: q.Email}, persisted, ui.LightPalette, nil, nil, nil)
		cards = append(cards, card)
		results = append(results, map[string]any{"case": tc.name, "parser_cycles": [4]float64{c.FiveHourFraction, c.WeeklyFraction, g.FiveHourFraction, g.WeeklyFraction}, "disk_storage_cycles": storage, "expected": tc.want, "badge_cycles": actual, "raw_bucket_count": len(persisted.RawBuckets), "card_text": cardTexts(card), "matches": actual == tc.want})
		if actual != tc.want {
			failures++
		}
	}
	window := test.NewWindow(container.NewPadded(container.NewVBox(container.NewGridWithColumns(2, cards...))))
	defer window.Close()
	window.Resize(fyne.NewSize(1000, 220))
	file, err := os.Create(filepath.Join(output, "E10-zero-cycle-cards.png"))
	check(err)
	check(png.Encode(file, window.Canvas().Capture()))
	check(file.Close())
	encoded, err := json.MarshalIndent(map[string]any{"driver": "Fyne software canvas, public ParseQuotaSummary -> Update -> SaveToDisk -> LoadFromDisk -> actual NewAccountCardWithQuota", "cases": results, "failures": failures}, "", "  ")
	check(err)
	fmt.Println(string(encoded))
}
