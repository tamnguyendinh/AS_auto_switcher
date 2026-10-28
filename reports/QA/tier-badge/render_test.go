package qa_tier_badge

import (
	"image/png"
	"os"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"

	"antigravity-switch/internal/engines/antigravity/manager"
	"antigravity-switch/internal/engines/antigravity/model"
	"antigravity-switch/internal/engines/antigravity/telemetry"
	"antigravity-switch/internal/ui"
)

func createSampleCards(palette ui.ThemePalette) (fyne.CanvasObject, []*ui.AccountCardWidget) {
	tiers := []struct {
		Email string
		Tier  string
		Desc  string
	}{
		{"ultra.account@example.com", "ULTRA", "Google AI Ultra (Amber #f59e0b)"},
		{"pro.account@example.com", "PRO", "Google AI Pro (Primary Cyan)"},
		{"biz.account@example.com", "BIZ", "Business / Teams (Green #10b981)"},
		{"free.account@example.com", "FREE", "Free Personal (Muted Gray)"},
		{"unauth.account@example.com", "--", "Unauthenticated / No Token (Muted Gray)"},
	}

	cardContainers := make([]fyne.CanvasObject, 0, len(tiers))
	widgets := make([]*ui.AccountCardWidget, 0, len(tiers))

	for _, item := range tiers {
		displayItem := manager.AccountDisplayItem{
			Email:    item.Email,
			Name:     item.Desc,
			Tier:     item.Tier,
			State:    model.StateReady,
			HasToken: item.Tier != "--",
		}
		quota := &telemetry.AccountQuotaData{
			Email: item.Email,
			Tier:  item.Tier,
			Gemini: telemetry.ModelQuotaBucket{
				RemainingFraction: 0.85,
				FiveHourFraction:  0.85,
				WeeklyFraction:    0.92,
				FiveHourReset:     time.Now().Add(3 * time.Hour),
				WeeklyReset:       time.Now().Add(48 * time.Hour),
			},
			Claude: telemetry.ModelQuotaBucket{
				RemainingFraction: 0.70,
				FiveHourFraction:  0.70,
				WeeklyFraction:    0.50,
				FiveHourReset:     time.Now().Add(2 * time.Hour),
				WeeklyReset:       time.Now().Add(72 * time.Hour),
			},
			Status: "ACTIVE",
		}
		if item.Tier == "--" {
			quota = nil
			displayItem.State = model.StateUnauth
			displayItem.HasToken = false
		}

		card := ui.NewAccountCardWidgetWithQuota(displayItem, quota, palette, nil, nil)
		widgets = append(widgets, card)
		cardContainers = append(cardContainers, card.Root)
	}

	grid := container.NewGridWithColumns(1, cardContainers...)
	return grid, widgets
}

func saveCanvas(t *testing.T, win fyne.Window, filename string) {
	_ = os.MkdirAll("screenshots", 0755)
	img := win.Canvas().Capture()
	if img == nil {
		t.Fatalf("capture returned nil for %s", filename)
	}
	f, err := os.Create(filename)
	if err != nil {
		t.Fatalf("failed to create %s: %v", filename, err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("failed to encode %s: %v", filename, err)
	}
	t.Logf("Saved %s", filename)
}

func TestRenderTierBadges_Themes(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	win := test.NewWindow(container.NewVBox())
	win.Resize(fyne.NewSize(920, 630))

	// 1. Dark Palette (Default)
	gridDark, _ := createSampleCards(ui.DarkPalette)
	win.SetContent(gridDark)
	saveCanvas(t, win, "screenshots/tier_badges_dark_theme.png")

	// 2. Light Palette
	gridLight, _ := createSampleCards(ui.LightPalette)
	win.SetContent(gridLight)
	saveCanvas(t, win, "screenshots/tier_badges_light_theme.png")

	// 3. Warm Palette
	gridWarm, _ := createSampleCards(ui.WarmPalette)
	win.SetContent(gridWarm)
	saveCanvas(t, win, "screenshots/tier_badges_warm_theme.png")
}

func TestRenderTierBadges_InPlaceReactiveMutation(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	win := test.NewWindow(container.NewVBox())
	win.Resize(fyne.NewSize(920, 300))

	palette := ui.DarkPalette
	grid, widgets := createSampleCards(palette)
	win.SetContent(grid)

	// Target widget: first card (starts as ULTRA)
	target := widgets[0]
	if target.TierText() == nil {
		t.Fatal("TierText is nil")
	}
	if target.TierText().Text != "ULTRA" {
		t.Fatalf("expected ULTRA, got %s", target.TierText().Text)
	}

	// Dynamic In-Place Transition: ULTRA -> PRO -> BIZ -> FREE -> --
	transitions := []struct {
		newTier  string
		expected string
	}{
		{"PRO", "PRO"},
		{"TEAMS_TIER_ENTERPRISE", "BIZ"},
		{"free-tier", "FREE"},
		{"unknown-token", "--"},
		{"g1-ultra-tier", "ULTRA"},
	}

	for _, tr := range transitions {
		target.UpdateTier(tr.newTier, palette)
		if target.TierText().Text != tr.expected {
			t.Errorf("transition %s failed: expected %s, got %s", tr.newTier, tr.expected, target.TierText().Text)
		}
	}

	// Capture end of dynamic in-place reactive cycle
	saveCanvas(t, win, "screenshots/tier_badges_inplace_mutation.png")
}
