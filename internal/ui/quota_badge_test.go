package ui

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"

	"as-auto-switcher/internal/engines/antigravity/connector"
	"as-auto-switcher/internal/engines/antigravity/telemetry"
)

// INV-P2-01: FilterExtensionModels with deduplication seen[cleanLabel] & regex qualifier stripping.
func TestFilterExtensionModels_DeduplicationAndQualifierStripping(t *testing.T) {
	configs := []connector.ClientModelConfig{
		// Baseline Gemini models (should be excluded)
		{ModelOrAlias: connector.ModelOrAlias{Model: "M318"}, Label: "Gemini 3.8 Flash (High)"},
		{ModelOrAlias: connector.ModelOrAlias{Model: "M319"}, Label: "Gemini 3.8 Flash (Medium)"},
		{ModelOrAlias: connector.ModelOrAlias{Model: "M320"}, Label: "Gemini 3.8 Flash (Low)"},
		{ModelOrAlias: connector.ModelOrAlias{Model: "M321"}, Label: "Gemini 2.5 Pro"},
		// Baseline Claude / 3P models (should be excluded)
		{ModelOrAlias: connector.ModelOrAlias{Model: "M25"}, Label: "Claude Sonnet 4.6 (Thinking)"},
		{ModelOrAlias: connector.ModelOrAlias{Model: "M26"}, Label: "Claude Opus 4.6 (Thinking)"},
		{ModelOrAlias: connector.ModelOrAlias{Model: "M27"}, Label: "GPT-OSS 120B (Medium)"},
		// Extension models (Gemini 4 Argon with multiple tiers, should be included and deduplicated to 1)
		{ModelOrAlias: connector.ModelOrAlias{Model: "M421"}, Label: "Gemini 4 Argon", Disabled: true},
		{ModelOrAlias: connector.ModelOrAlias{Model: "M422"}, Label: "Gemini 4 Argon (High)", Disabled: true},
		{ModelOrAlias: connector.ModelOrAlias{Model: "M423"}, Label: "Gemini 4 Argon (Low)", Disabled: true},
	}

	result := FilterExtensionModels(configs)
	if len(result) != 1 {
		t.Fatalf("INV-P2-01 VIOLATION: expected exactly 1 discovered model (Argon), got %d: %+v", len(result), result)
	}

	m := result[0]
	if m.ModelID != "M421" {
		t.Errorf("INV-P2-01 VIOLATION: expected ModelID M421, got %s", m.ModelID)
	}
	if m.Label != "Gemini 4 Argon" {
		t.Errorf("INV-P2-01 VIOLATION: expected Label 'Gemini 4 Argon', got '%s'", m.Label)
	}
	if m.Token != "argon" {
		t.Errorf("INV-P2-01 VIOLATION: expected Token 'argon', got '%s'", m.Token)
	}
	if m.Line1 != "Gemini 4" {
		t.Errorf("INV-P2-01 VIOLATION: expected Line1 'Gemini 4', got '%s'", m.Line1)
	}
	if m.Line2 != "Argon" {
		t.Errorf("INV-P2-01 VIOLATION: expected Line2 'Argon', got '%s'", m.Line2)
	}
	if !m.Disabled {
		t.Errorf("INV-P2-01 VIOLATION: expected Disabled == true")
	}

	// Test single-word extension model
	configs2 := []connector.ClientModelConfig{
		{ModelOrAlias: connector.ModelOrAlias{Model: "M900"}, Label: "Argon (Preview)"},
	}
	result2 := FilterExtensionModels(configs2)
	if len(result2) != 1 {
		t.Fatalf("expected 1 discovered model for single-word label, got %d", len(result2))
	}
	if result2[0].Line1 != "" || result2[0].Line2 != "Argon" || result2[0].Token != "argon" {
		t.Errorf("unexpected single-word separation: Line1=%q, Line2=%q, Token=%q", result2[0].Line1, result2[0].Line2, result2[0].Token)
	}
}

// INV-P2-02: SplitModelLabel2Lines & matrixStackedModelCellLayout supporting 1 object (baseline) and 2 objects (stacked) in 48px fixed width.
func TestSplitModelLabel2Lines(t *testing.T) {
	cases := []struct {
		input     string
		wantLine1 string
		wantLine2 string
	}{
		{"Gemini 4 Argon", "Gemini 4", "Argon"},
		{"Gemini 4 Argon (High)", "Gemini 4", "Argon"},
		{"Gemini 4 Argon (Thinking)", "Gemini 4", "Argon"},
		{"Argon", "", "Argon"},
		{"Claude 3.7 Sonnet (Thinking)", "Claude 3.7", "Sonnet"},
		{"", "", ""},
		{"   ", "", ""},
	}

	for _, c := range cases {
		l1, l2 := SplitModelLabel2Lines(c.input)
		if l1 != c.wantLine1 || l2 != c.wantLine2 {
			t.Errorf("SplitModelLabel2Lines(%q) = (%q, %q); want (%q, %q)", c.input, l1, l2, c.wantLine1, c.wantLine2)
		}
	}
}

func TestMatrixStackedModelCellLayout(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	layout := &matrixStackedModelCellLayout{}

	// 1 object (baseline model label, e.g. Gemini or Claude)
	txt1 := canvas.NewText("Gemini", DarkPalette.Text)
	min1 := layout.MinSize([]fyne.CanvasObject{txt1})
	if min1.Width != 48.0 || min1.Height != 28.0 {
		t.Errorf("INV-P2-02 VIOLATION: expected 1-object MinSize 48x28, got %fx%f", min1.Width, min1.Height)
	}

	cont1 := container.New(layout, txt1)
	cont1.Resize(fyne.NewSize(48.0, 28.0))
	layout.Layout([]fyne.CanvasObject{txt1}, fyne.NewSize(48.0, 28.0))
	// Object should be positioned at bottom tier (y >= 13.0)
	if txt1.Position().Y < 13.0 {
		t.Errorf("INV-P2-02 VIOLATION: expected 1-object y >= 13.0, got %f", txt1.Position().Y)
	}
	if txt1.Size().Width != 48.0 {
		t.Errorf("INV-P2-02 VIOLATION: expected width 48.0, got %f", txt1.Size().Width)
	}

	// 2 objects (stacked extension model, e.g. "Gemini 4" on top, "Argon" on bottom)
	txtTop := canvas.NewText("Gemini 4", DarkPalette.TextMuted)
	txtBottom := canvas.NewText("Argon", DarkPalette.Text)
	min2 := layout.MinSize([]fyne.CanvasObject{txtTop, txtBottom})
	if min2.Width != 48.0 || min2.Height != 28.0 {
		t.Errorf("INV-P2-02 VIOLATION: expected 2-object MinSize 48x28, got %fx%f", min2.Width, min2.Height)
	}

	cont2 := container.New(layout, txtTop, txtBottom)
	cont2.Resize(fyne.NewSize(48.0, 28.0))
	layout.Layout([]fyne.CanvasObject{txtTop, txtBottom}, fyne.NewSize(48.0, 28.0))
	// Top object should be at y in [0, 13)
	if txtTop.Position().Y < 0 || txtTop.Position().Y >= 13.0 {
		t.Errorf("INV-P2-02 VIOLATION: expected txtTop y in [0, 13), got %f", txtTop.Position().Y)
	}
	// Bottom object should be at y >= 13.0
	if txtBottom.Position().Y < 13.0 {
		t.Errorf("INV-P2-02 VIOLATION: expected txtBottom y >= 13.0, got %f", txtBottom.Position().Y)
	}
	if txtTop.Size().Width != 48.0 || txtBottom.Size().Width != 48.0 {
		t.Errorf("INV-P2-02 VIOLATION: expected both labels width 48.0, got top=%f, bottom=%f", txtTop.Size().Width, txtBottom.Size().Width)
	}
}

// INV-P2-03: Dynamic layout math (+32px): MinSize() = 77.0 + K*32.0 in both matrixColumnLayout and matrixTableLayout.
func TestMatrixDynamicLayoutMath_MinSize(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	colLayout := &matrixColumnLayout{}

	// matrixColumnLayout MinSize tests:
	// K = 0 (3 objects) -> 77.0
	obj0 := canvas.NewText("H", DarkPalette.Text)
	obj1 := canvas.NewText("G", DarkPalette.Text)
	obj2 := canvas.NewText("C", DarkPalette.Text)
	minK0 := colLayout.MinSize([]fyne.CanvasObject{obj0, obj1, obj2})
	if minK0.Height != 77.0 {
		t.Errorf("INV-P2-03 VIOLATION: expected colLayout height 77.0 for K=0, got %f", minK0.Height)
	}

	// K = 1 (4 objects) -> 109.0
	obj3 := canvas.NewText("E1", DarkPalette.Text)
	minK1 := colLayout.MinSize([]fyne.CanvasObject{obj0, obj1, obj2, obj3})
	if minK1.Height != 109.0 {
		t.Errorf("INV-P2-03 VIOLATION: expected colLayout height 109.0 for K=1, got %f", minK1.Height)
	}

	// K = 2 (5 objects) -> 141.0
	obj4 := canvas.NewText("E2", DarkPalette.Text)
	minK2 := colLayout.MinSize([]fyne.CanvasObject{obj0, obj1, obj2, obj3, obj4})
	if minK2.Height != 141.0 {
		t.Errorf("INV-P2-03 VIOLATION: expected colLayout height 141.0 for K=2, got %f", minK2.Height)
	}

	// matrixTableLayout MinSize tests:
	tableLayout := &matrixTableLayout{}
	colM := container.New(colLayout, obj0, obj1, obj2)
	col5 := container.New(colLayout, obj0, obj1, obj2)
	colW := container.New(colLayout, obj0, obj1, obj2)
	hSep := canvas.NewRectangle(DarkPalette.BorderSubtle)

	tMinK0 := tableLayout.MinSize([]fyne.CanvasObject{colM, col5, colW, hSep})
	if tMinK0.Width != 228.0 || tMinK0.Height != 77.0 {
		t.Errorf("INV-P2-03 VIOLATION: expected table MinSize 228x77 for K=0, got %fx%f", tMinK0.Width, tMinK0.Height)
	}

	// Add extension row to colM
	colM.Add(obj3)
	col5.Add(obj3)
	colW.Add(obj3)
	hSep1 := canvas.NewRectangle(DarkPalette.BorderSubtle)
	tMinK1 := tableLayout.MinSize([]fyne.CanvasObject{colM, col5, colW, hSep, hSep1})
	if tMinK1.Width != 228.0 || tMinK1.Height != 109.0 {
		t.Errorf("INV-P2-03 VIOLATION: expected table MinSize 228x109 for K=1, got %fx%f", tMinK1.Width, tMinK1.Height)
	}

	// Add second extension row
	colM.Add(obj4)
	col5.Add(obj4)
	colW.Add(obj4)
	hSep2 := canvas.NewRectangle(DarkPalette.BorderSubtle)
	tMinK2 := tableLayout.MinSize([]fyne.CanvasObject{colM, col5, colW, hSep, hSep1, hSep2})
	if tMinK2.Width != 228.0 || tMinK2.Height != 141.0 {
		t.Errorf("INV-P2-03 VIOLATION: expected table MinSize 228x141 for K=2, got %fx%f", tMinK2.Width, tMinK2.Height)
	}

	// Verify Layout geometry positions for K=1
	tableLayout.Layout([]fyne.CanvasObject{colM, col5, colW, hSep, hSep1}, fyne.NewSize(228.0, 109.0))
	if hSep.Position().Y != 46.0 {
		t.Errorf("INV-P2-03 VIOLATION: expected hSep y=46.0, got %f", hSep.Position().Y)
	}
	if hSep1.Position().Y != 78.0 {
		t.Errorf("INV-P2-03 VIOLATION: expected hSep1 y=78.0, got %f", hSep1.Position().Y)
	}
}

// INV-P2-04: Column symmetry: elimination of compatWeek, len(colModel.Objects) == len(col5h.Objects) == len(colWeek.Objects) == 3 + K.
func TestMatrixColumnSymmetry_NoCompatWeek(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	dq := DualCycleQuota{
		Gemini5h:   0.80,
		GeminiWeek: 0.90,
		Claude5h:   0.50,
		ClaudeWeek: 0.60,
		HasData:    true,
	}

	// Test K=0
	tableK0, _, _, _, _, _, extRows0 := createDynamicMatrixQuotaTableWithCells(dq, "PRO", DarkPalette, nil)
	if len(extRows0) != 0 {
		t.Errorf("INV-P2-04 VIOLATION: expected 0 extRows for K=0, got %d", len(extRows0))
	}
	tContK0, ok := tableK0.(*fyne.Container)
	if !ok || len(tContK0.Objects) < 4 {
		t.Fatalf("expected valid table container")
	}
	colModel0 := tContK0.Objects[0].(*fyne.Container)
	col5h0 := tContK0.Objects[1].(*fyne.Container)
	colWeek0 := tContK0.Objects[2].(*fyne.Container)

	if len(colModel0.Objects) != 3 || len(col5h0.Objects) != 3 || len(colWeek0.Objects) != 3 {
		t.Fatalf("INV-P2-04 VIOLATION: expected len 3 for all columns when K=0; got colModel=%d, col5h=%d, colWeek=%d",
			len(colModel0.Objects), len(col5h0.Objects), len(colWeek0.Objects))
	}

	// Test K=1 (Argon)
	discovered := []DiscoveredModelInfo{
		{ModelID: "M421", Label: "Gemini 4 Argon", Token: "argon", Line1: "Gemini 4", Line2: "Argon", Disabled: true},
	}
	tableK1, _, _, _, _, _, extRows1 := createDynamicMatrixQuotaTableWithCells(dq, "PRO", DarkPalette, discovered)
	if len(extRows1) != 1 {
		t.Errorf("INV-P2-04 VIOLATION: expected 1 extRow for K=1, got %d", len(extRows1))
	}
	tContK1 := tableK1.(*fyne.Container)
	colModel1 := tContK1.Objects[0].(*fyne.Container)
	col5h1 := tContK1.Objects[1].(*fyne.Container)
	colWeek1 := tContK1.Objects[2].(*fyne.Container)

	if len(colModel1.Objects) != 4 || len(col5h1.Objects) != 4 || len(colWeek1.Objects) != 4 {
		t.Fatalf("INV-P2-04 VIOLATION: expected len 4 for all columns when K=1; got colModel=%d, col5h=%d, colWeek=%d",
			len(colModel1.Objects), len(col5h1.Objects), len(colWeek1.Objects))
	}

	// Test K=2
	discovered2 := []DiscoveredModelInfo{
		{ModelID: "M421", Label: "Gemini 4 Argon", Token: "argon", Line1: "Gemini 4", Line2: "Argon", Disabled: true},
		{ModelID: "M500", Label: "Reasoning Extra", Token: "extra", Line1: "Reasoning", Line2: "Extra", Disabled: false},
	}
	tableK2, _, _, _, _, _, extRows2 := createDynamicMatrixQuotaTableWithCells(dq, "PRO", DarkPalette, discovered2)
	if len(extRows2) != 2 {
		t.Errorf("INV-P2-04 VIOLATION: expected 2 extRows for K=2, got %d", len(extRows2))
	}
	tContK2 := tableK2.(*fyne.Container)
	colModel2 := tContK2.Objects[0].(*fyne.Container)
	col5h2 := tContK2.Objects[1].(*fyne.Container)
	colWeek2 := tContK2.Objects[2].(*fyne.Container)

	if len(colModel2.Objects) != 5 || len(col5h2.Objects) != 5 || len(colWeek2.Objects) != 5 {
		t.Fatalf("INV-P2-04 VIOLATION: expected len 5 for all columns when K=2; got colModel=%d, col5h=%d, colWeek=%d",
			len(colModel2.Objects), len(col5h2.Objects), len(colWeek2.Objects))
	}
}

// INV-P2-05: Ternary display: setNotApplicable() ("--", hide bar), setLocked() ("[LOCKED]", hide bar), text-only positioning at x = 0 without overflow.
func TestMatrixQuotaCell_TernaryStateAndPositioning(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	cell := newMatrixQuotaCell(0.75, time.Now().Add(2*time.Hour), false, true, DarkPalette)
	if cell.barContainer == nil {
		t.Fatalf("INV-P2-05 VIOLATION: cell.barContainer must not be nil")
	}

	// Test setNotApplicable()
	cell.setNotApplicable()
	if cell.pctText.Text != "--" {
		t.Errorf("INV-P2-05 VIOLATION: expected pctText '--', got %q", cell.pctText.Text)
	}
	if cell.pctText.Color != ColorQuotaMuted {
		t.Errorf("INV-P2-05 VIOLATION: expected pctText ColorQuotaMuted")
	}
	if cell.barContainer.Visible() {
		t.Errorf("INV-P2-05 VIOLATION: expected barContainer hidden in setNotApplicable")
	}
	if cell.timeText.Text != "" {
		t.Errorf("INV-P2-05 VIOLATION: expected empty timeText in setNotApplicable, got %q", cell.timeText.Text)
	}

	// Check Text-Only Layout positioning (x = 0)
	cell.container.Resize(fyne.NewSize(78.0, 28.0))
	cell.container.Layout.Layout(cell.container.Objects, fyne.NewSize(78.0, 28.0))
	if cell.pctText.Position().X != 0 {
		t.Errorf("INV-P2-05 VIOLATION: expected pctText at x=0 in text-only mode, got %f", cell.pctText.Position().X)
	}
	if cell.pctText.Size().Width != 78.0 {
		t.Errorf("INV-P2-05 VIOLATION: expected pctText width 78.0, got %f", cell.pctText.Size().Width)
	}

	// Test setLocked()
	cell.setLocked()
	if cell.pctText.Text != "[LOCKED]" {
		t.Errorf("INV-P2-05 VIOLATION: expected pctText '[LOCKED]', got %q", cell.pctText.Text)
	}
	if cell.pctText.Color != ColorQuotaAmber {
		t.Errorf("INV-P2-05 VIOLATION: expected pctText ColorQuotaAmber")
	}
	if cell.barContainer.Visible() {
		t.Errorf("INV-P2-05 VIOLATION: expected barContainer hidden in setLocked")
	}
	if cell.timeText.Text != "--" {
		t.Errorf("INV-P2-05 VIOLATION: expected timeText '--' in setLocked, got %q", cell.timeText.Text)
	}

	// Check Text-Only Layout positioning (x = 0) for [LOCKED]
	cell.container.Resize(fyne.NewSize(86.0, 28.0))
	cell.container.Layout.Layout(cell.container.Objects, fyne.NewSize(86.0, 28.0))
	if cell.pctText.Position().X != 0 {
		t.Errorf("INV-P2-05 VIOLATION: expected pctText at x=0 for [LOCKED], got %f", cell.pctText.Position().X)
	}
	if cell.pctText.Size().Width != 86.0 {
		t.Errorf("INV-P2-05 VIOLATION: expected pctText width 86.0, got %f", cell.pctText.Size().Width)
	}

	// Test setUnavailable()
	cell.setUnavailable()
	if cell.pctText.Text != "N/A" {
		t.Errorf("INV-P2-05 VIOLATION: expected pctText 'N/A', got %q", cell.pctText.Text)
	}
	if cell.pctText.Color != ColorQuotaMuted {
		t.Errorf("INV-P2-05 VIOLATION: expected pctText ColorQuotaMuted")
	}
	if cell.barContainer.Visible() {
		t.Errorf("INV-P2-05 VIOLATION: expected barContainer hidden in setUnavailable")
	}
}

// INV-P2-06: Bar recovery: update() calls barContainer.Show() when valid data (fraction >= 0 && hasData) exists.
func TestMatrixQuotaCell_BarRecovery(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	cell := newMatrixQuotaCell(0.50, time.Now().Add(time.Hour), false, true, DarkPalette)

	// Step 1: Hide bar via setNotApplicable()
	cell.setNotApplicable()
	if cell.barContainer.Visible() {
		t.Fatalf("precondition failed: barContainer should be hidden")
	}

	// Step 2: Call update with valid quota -> barContainer MUST recover visibility
	future := time.Now().Add(3 * time.Hour)
	cell.update(0.85, future, true)
	if !cell.barContainer.Visible() {
		t.Errorf("INV-P2-06 VIOLATION: expected barContainer to recover visibility (Visible == true) after valid update")
	}
	if cell.pctText.Text != "85%" {
		t.Errorf("INV-P2-06 VIOLATION: expected pctText '85%%', got %q", cell.pctText.Text)
	}

	// Check layout positioning when bar is visible: bar at x=0, pct at x=44
	cell.container.Resize(fyne.NewSize(78.0, 28.0))
	cell.container.Layout.Layout(cell.container.Objects, fyne.NewSize(78.0, 28.0))
	if cell.barContainer.Position().X != 0 {
		t.Errorf("INV-P2-06 VIOLATION: expected barContainer pos.X=0, got %f", cell.barContainer.Position().X)
	}
	if cell.pctText.Position().X != 44.0 {
		t.Errorf("INV-P2-06 VIOLATION: expected pctText pos.X=44.0, got %f", cell.pctText.Position().X)
	}

	// Step 3: Hide bar via setLocked(), then update again
	cell.setLocked()
	if cell.barContainer.Visible() {
		t.Fatalf("precondition failed: barContainer should be hidden in setLocked")
	}

	cell.update(0.40, future, true)
	if !cell.barContainer.Visible() {
		t.Errorf("INV-P2-06 VIOLATION: expected barContainer to recover visibility after setLocked update")
	}
	if cell.pctText.Text != "40%" {
		t.Errorf("INV-P2-06 VIOLATION: expected pctText '40%%', got %q", cell.pctText.Text)
	}
}

// INV-P2-07: ExtensionQuotas map in DualCycleQuota & bucket isolation in extractFamilyCycles.
func TestDualCycleQuota_ExtensionQuotasIsolation(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	now := time.Now()
	quota := &telemetry.AccountQuotaData{
		Email:  "test@example.com",
		Status: telemetry.StatusPlentiful,
		Claude: telemetry.ModelQuotaBucket{
			ModelID:           "claude",
			FiveHourFraction:  0.80,
			WeeklyFraction:    0.95,
			FiveHourReset:     now.Add(4 * time.Hour),
			WeeklyReset:       now.Add(7 * 24 * time.Hour),
			RemainingFraction: 0.80,
		},
		Gemini: telemetry.ModelQuotaBucket{
			ModelID:           "gemini-5h",
			FiveHourFraction:  0.70,
			WeeklyFraction:    0.85,
			FiveHourReset:     now.Add(4 * time.Hour),
			WeeklyReset:       now.Add(7 * 24 * time.Hour),
			RemainingFraction: 0.70,
		},
		RawBuckets: []telemetry.ModelQuotaBucket{
			{ModelID: "gemini-5h", FiveHourFraction: 0.70, RemainingFraction: 0.70},
			{ModelID: "gemini-weekly", WeeklyFraction: 0.85, RemainingFraction: 0.85},
			// Extension model bucket: gemini-4-argon (must not be swallowed by Gemini baseline!)
			{
				ModelID:           "gemini-4-argon",
				TokenType:         "argon",
				WeeklyFraction:    0.45,
				RemainingFraction: 0.45,
				WeeklyReset:       now.Add(5 * 24 * time.Hour),
				FiveHourFraction:  -1, // Argon has no 5h cycle
			},
		},
		ExtensionBuckets: map[string]telemetry.ModelQuotaBucket{
			"argon": {
				ModelID:           "gemini-4-argon",
				TokenType:         "argon",
				WeeklyFraction:    0.45,
				RemainingFraction: 0.45,
				WeeklyReset:       now.Add(5 * 24 * time.Hour),
				FiveHourFraction:  -1,
			},
		},
	}

	dq := ExtractDualCycleQuota(quota)

	// Baseline Gemini fractions must remain isolated
	if dq.Gemini5h != 0.70 {
		t.Errorf("INV-P2-07 VIOLATION: expected Gemini5h 0.70, got %f (bucket swallowed?)", dq.Gemini5h)
	}
	if dq.GeminiWeek != 0.85 {
		t.Errorf("INV-P2-07 VIOLATION: expected GeminiWeek 0.85, got %f (bucket swallowed?)", dq.GeminiWeek)
	}

	// ExtensionQuotas must contain argon
	if dq.ExtensionQuotas == nil {
		t.Fatalf("INV-P2-07 VIOLATION: ExtensionQuotas map is nil")
	}
	argonBucket, ok := dq.ExtensionQuotas["argon"]
	if !ok {
		t.Fatalf("INV-P2-07 VIOLATION: ExtensionQuotas does not contain 'argon'")
	}
	if argonBucket.WeeklyFraction != 0.45 {
		t.Errorf("INV-P2-07 VIOLATION: expected argon weekly fraction 0.45, got %f", argonBucket.WeeklyFraction)
	}

	// Test rendering in dynamic table:
	// Case A: Argon enabled with quota
	discoveredEnabled := []DiscoveredModelInfo{
		{ModelID: "M421", Label: "Gemini 4 Argon", Token: "argon", Line1: "Gemini 4", Line2: "Argon", Disabled: false},
	}
	_, _, _, _, _, _, extRowsA := createDynamicMatrixQuotaTableWithCells(dq, "PRO", DarkPalette, discoveredEnabled)
	if len(extRowsA) != 1 {
		t.Fatalf("expected 1 extRow")
	}
	// Argon 5h should be "--" (not applicable)
	if extRowsA[0].cell5h.pctText.Text != "--" {
		t.Errorf("expected Argon 5h cell '--', got %q", extRowsA[0].cell5h.pctText.Text)
	}
	// Argon Week should be "45%" with visible bar
	if extRowsA[0].cellWeek.pctText.Text != "45%" {
		t.Errorf("expected Argon Week cell '45%%', got %q", extRowsA[0].cellWeek.pctText.Text)
	}
	if !extRowsA[0].cellWeek.barContainer.Visible() {
		t.Errorf("expected Argon Week barContainer visible")
	}

	// Case B: Argon disabled
	discoveredDisabled := []DiscoveredModelInfo{
		{ModelID: "M421", Label: "Gemini 4 Argon", Token: "argon", Line1: "Gemini 4", Line2: "Argon", Disabled: true},
	}
	_, _, _, _, _, _, extRowsB := createDynamicMatrixQuotaTableWithCells(dq, "PRO", DarkPalette, discoveredDisabled)
	if len(extRowsB) != 1 {
		t.Fatalf("expected 1 extRow")
	}
	// Argon 5h: "--"
	if extRowsB[0].cell5h.pctText.Text != "--" {
		t.Errorf("expected Argon disabled 5h cell '--', got %q", extRowsB[0].cell5h.pctText.Text)
	}
	// Argon Week: "[LOCKED]"
	if extRowsB[0].cellWeek.pctText.Text != "[LOCKED]" {
		t.Errorf("expected Argon disabled Week cell '[LOCKED]', got %q", extRowsB[0].cellWeek.pctText.Text)
	}
	if extRowsB[0].cellWeek.barContainer.Visible() {
		t.Errorf("expected Argon disabled Week barContainer hidden")
	}
}

// INV-P2-08: Backwards compatibility preserved in createMatrixQuotaTableWithCells calling createDynamicMatrixQuotaTableWithCells(..., nil).
func TestMatrixQuotaTable_BackwardsCompatibility(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	dq := DualCycleQuota{
		Gemini5h:   0.88,
		GeminiWeek: 0.90,
		Claude5h:   1.0,
		ClaudeWeek: 1.0,
		HasData:    true,
	}

	table, txtTier, cG5, cGW, cC5, cCW := createMatrixQuotaTableWithCells(dq, "PRO", DarkPalette)
	if table == nil || txtTier == nil || cG5 == nil || cGW == nil || cC5 == nil || cCW == nil {
		t.Fatalf("INV-P2-08 VIOLATION: createMatrixQuotaTableWithCells returned nil values")
	}

	cont, ok := table.(*fyne.Container)
	if !ok {
		t.Fatalf("INV-P2-08 VIOLATION: expected table container")
	}
	// Should have exactly 4 objects: colModel, col5h, colWeek, hSep
	if len(cont.Objects) != 4 {
		t.Errorf("INV-P2-08 VIOLATION: expected 4 objects in baseline table, got %d", len(cont.Objects))
	}

	minSize := cont.MinSize()
	if minSize.Width != 228.0 || minSize.Height != 77.0 {
		t.Errorf("INV-P2-08 VIOLATION: expected MinSize 228x77, got %fx%f", minSize.Width, minSize.Height)
	}

	// Helper wrapper functions
	tableWithTier := CreateMatrixQuotaTableWithTier(dq, "ULTRA", DarkPalette)
	if tableWithTier == nil || tableWithTier.MinSize().Height != 77.0 {
		t.Errorf("INV-P2-08 VIOLATION: CreateMatrixQuotaTableWithTier invalid")
	}

	tableDefault := CreateMatrixQuotaTable(dq, DarkPalette)
	if tableDefault == nil || tableDefault.MinSize().Height != 77.0 {
		t.Errorf("INV-P2-08 VIOLATION: CreateMatrixQuotaTable invalid")
	}
}
