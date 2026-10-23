//go:build ignore

// R5 zero-cycle acceptance: use exported production parsers, cache, extractor and card.
// Run with runtime_probe.go and quota_matrix.go from the repository root.
package main

import (
	"encoding/json"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"antigravity-switch/internal/manager"
	"antigravity-switch/internal/telemetry"
	"antigravity-switch/internal/ui"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
)

func quotaRows(card fyne.CanvasObject) map[string][]string {
	rows := map[string][]string{}
	walk(card, func(o fyne.CanvasObject) {
		c, ok := o.(*fyne.Container)
		if !ok || len(c.Objects) == 0 {
			return
		}
		label, ok := c.Objects[0].(*canvas.Text)
		if ok && (label.Text == "5h:" || label.Text == "week:") {
			rows[label.Text] = texts(c)
		}
	})
	return rows
}

func fractionLabel(model string, f float64) string {
	if f < 0 {
		return model + " --%"
	}
	return model + " " + strconv.FormatFloat(f*100, 'f', -1, 64) + "%"
}

func runExplicitZeroMatrix(tmp string) []any {
	// Match the injected palette to the app theme so labels remain readable in evidence.
	priorTheme := fyne.CurrentApp().Settings().Theme()
	fyne.CurrentApp().Settings().SetTheme(ui.NewThemeInstance(ui.ThemeLight))
	defer fyne.CurrentApp().Settings().SetTheme(priorTheme)
	type fixture struct {
		name, body, summary string
		want                ui.DualCycleQuota
		illustrate          bool
	}
	unknown := fractions(-1, -1, -1, -1, false)
	claudeZero := fractions(0, 0, -1, -1, true)
	geminiZero := fractions(-1, -1, 0, 0, true)
	bothZero := fractions(0, 0, 0, 0, true)
	const cSummary = `{"groups":[{"displayName":"Claude models","buckets":[{"bucketId":"opaque-a","window":"5h","remainingFraction":0},{"bucketId":"opaque-b","window":"weekly","remainingFraction":0}]}]}`
	gSummary := strings.ReplaceAll(cSummary, "Claude models", "Gemini Models")
	fixtures := []fixture{
		{name: "claude-parser-zero", summary: cSummary, want: claudeZero, illustrate: true},
		{name: "gemini-parser-zero", summary: gSummary, want: geminiZero, illustrate: true},
		{name: "both-parser-zero", summary: strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(fullSummary, "0.88", "0"), "0.90", "0"), "0.46", "0"), "0.12", "0"), want: bothZero},
		{name: "both-explicit-zero", body: `{"claude":{"remaining_fraction":0,"five_hour_fraction":0,"weekly_fraction":0},"gemini":{"remaining_fraction":0,"five_hour_fraction":0,"weekly_fraction":0}}`, want: bothZero, illustrate: true},
		{name: "claude-no-legacy-fraction", body: `{"claude":{"five_hour_fraction":0,"weekly_fraction":0}}`, want: claudeZero},
		{name: "gemini-no-legacy-fraction", body: `{"gemini":{"five_hour_fraction":0,"weekly_fraction":0}}`, want: geminiZero},
		{name: "both-camel-zero", body: `{"claude":{"remainingFraction":0,"fiveHourFraction":0,"weeklyFraction":0},"gemini":{"remainingFraction":0,"fiveHourFraction":0,"weeklyFraction":0}}`, want: bothZero},
		{name: "explicit-unknown", body: `{"claude":{"remaining_fraction":-1,"five_hour_fraction":-1,"weekly_fraction":-1},"gemini":{"remaining_fraction":-1,"five_hour_fraction":-1,"weekly_fraction":-1}}`, want: unknown, illustrate: true},
		{name: "missing-families", body: `{}`, want: unknown},
		{name: "null-families", body: `{"claude":null,"gemini":null}`, want: unknown},
		{name: "legacy-only-zero", body: `{"claude":{"remaining_fraction":0}}`, want: fractions(0, -1, -1, -1, true), illustrate: true},
		{name: "legacy-only-positive", body: `{"claude":{"remaining_fraction":0.2}}`, want: fractions(.2, -1, -1, -1, true)},
		{name: "weekly-only-zero", body: `{"claude":{"weekly_fraction":0}}`, want: fractions(-1, 0, -1, -1, true), illustrate: true},
		{name: "five-hour-only-zero", body: `{"gemini":{"five_hour_fraction":0}}`, want: fractions(-1, -1, 0, -1, true)},
		// Same invariant, independent legacy value: the JSON explicitly supplies both cycles.
		// Unmarshal normalizes missing fields to -1, so these are not default struct zeroes.
		{name: "claude-legacy-positive", body: `{"claude":{"remaining_fraction":0.2,"five_hour_fraction":0,"weekly_fraction":0}}`, want: claudeZero, illustrate: true},
		{name: "gemini-legacy-positive", body: `{"gemini":{"remaining_fraction":0.8,"five_hour_fraction":0,"weekly_fraction":0}}`, want: geminiZero, illustrate: true},
		{name: "both-legacy-positive", body: `{"claude":{"remaining_fraction":0.2,"five_hour_fraction":0,"weekly_fraction":0},"gemini":{"remaining_fraction":0.8,"five_hour_fraction":0,"weekly_fraction":0}}`, want: bothZero, illustrate: true},
		{name: "claude-positive-with-reset", body: `{"claude":{"remaining_fraction":0.2,"five_hour_fraction":0,"weekly_fraction":0,"five_hour_reset":"2026-10-19T00:00:00Z"}}`, want: claudeZero},
		{name: "gemini-positive-with-reset", body: `{"gemini":{"remaining_fraction":0.8,"five_hour_fraction":0,"weekly_fraction":0,"weekly_reset":"2026-10-24T00:00:00Z"}}`, want: geminiZero},
		{name: "claude-positive-with-raw", body: `{"claude":{"remaining_fraction":0.2,"five_hour_fraction":0,"weekly_fraction":0},"raw_buckets":[{"model_id":"3p-5h","remaining_fraction":0},{"model_id":"3p-weekly","remaining_fraction":0}]}`, want: claudeZero},
	}
	var results []any
	var cards []fyne.CanvasObject
	for _, tc := range fixtures {
		q := &telemetry.AccountQuotaData{}
		if tc.summary != "" {
			c, g, _, reset, status, err := telemetry.ParseQuotaSummary([]byte(tc.summary))
			must(err)
			q.Claude, q.Gemini, q.ResetTime, q.Status = c, g, reset, status
		} else {
			must(json.Unmarshal([]byte(tc.body), q))
		}
		q.Email = tc.name + "@example.com"
		entry := map[string]any{"case": tc.name, "input_json": tc.body, "input_summary": tc.summary, "expected": tc.want}
		var observations []any
		path := filepath.Join(tmp, "explicit-"+tc.name+".json")
		for stage := 0; stage < 3; stage++ {
			dq := ui.ExtractDualCycleQuota(q)
			card := ui.NewAccountCardWithQuota(manager.AccountDisplayItem{Email: q.Email}, q, ui.LightPalette, nil, nil, nil)
			actualRows := quotaRows(card)
			wantedRows := map[string][]string{
				"5h:":   {"5h:", fractionLabel("Claude", tc.want.Claude5h), fractionLabel("Gemini", tc.want.Gemini5h)},
				"week:": {"week:", fractionLabel("Claude", tc.want.ClaudeWeek), fractionLabel("Gemini", tc.want.GeminiWeek)},
			}
			match := dq == tc.want
			cardMatch := reflect.DeepEqual(actualRows, wantedRows)
			label := fmt.Sprintf("F1/explicit/%s/stage%d", tc.name, stage)
			require(label+"/extract", match)
			require(label+"/card_rows", cardMatch)
			if !match || !cardMatch {
				quotaMismatches = append(quotaMismatches, label)
			}
			observations = append(observations, map[string]any{
				"stage":                   []string{"normalized_input", "save_reload_1", "save_reload_2"}[stage],
				"stored_cycles":           [4]float64{q.Claude.FiveHourFraction, q.Claude.WeeklyFraction, q.Gemini.FiveHourFraction, q.Gemini.WeeklyFraction},
				"stored_legacy_fractions": [2]float64{q.Claude.RemainingFraction, q.Gemini.RemainingFraction},
				"extracted":               dq, "extract_matches": match, "card_rows": actualRows, "card_matches": cardMatch,
				"raw_count": len(q.RawBuckets), "all_resets_absent": q.Claude.FiveHourReset.IsZero() && q.Claude.WeeklyReset.IsZero() && q.Gemini.FiveHourReset.IsZero() && q.Gemini.WeeklyReset.IsZero(),
			})
			if stage == 0 && tc.illustrate {
				cards = append(cards, card)
			}
			if stage < 2 {
				cache := telemetry.NewQuotaCache(path)
				cache.Update(q)
				must(cache.SaveToDisk())
				reloaded := telemetry.NewQuotaCache(path)
				must(reloaded.LoadFromDisk())
				var ok bool
				q, ok = reloaded.Get(q.Email)
				if !ok {
					panic("zero-cycle cache missing account")
				}
			}
		}
		entry["observations"] = observations
		results = append(results, entry)
	}
	window := test.NewWindow(container.NewPadded(container.NewVBox(container.NewGridWithColumns(3, cards...))))
	defer window.Close()
	window.Resize(fyne.NewSize(1440, 300))
	file, err := os.Create(filepath.Join(evidenceDir, "E07-explicit-cycle-cards.png"))
	must(err)
	must(png.Encode(file, window.Canvas().Capture()))
	must(file.Close())
	return results
}
