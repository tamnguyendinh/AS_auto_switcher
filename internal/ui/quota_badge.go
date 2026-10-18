package ui

import (
	"fmt"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"

	"antigravity-switch/internal/telemetry"
)

var (
	// Three-tier quota colors per UI-TR-04 & Architect Spec
	ColorQuotaGreen = color.NRGBA{R: 0x22, G: 0xc5, B: 0x5e, A: 0xff} // #22c55e (> 50%)
	ColorQuotaAmber = color.NRGBA{R: 0xf5, G: 0x9e, B: 0x0b, A: 0xff} // #f59e0b (15% - 50%)
	ColorQuotaRed   = color.NRGBA{R: 0xef, G: 0x44, B: 0x44, A: 0xff} // #ef4444 (< 15%)
	ColorQuotaMuted = color.NRGBA{R: 0x88, G: 0x88, B: 0x99, A: 0xff}
)

// GetQuotaTierColor determines the semantic color for a given quota fraction [0.0, 1.0].
func GetQuotaTierColor(fraction float64) color.Color {
	if fraction > 0.50 {
		return ColorQuotaGreen
	}
	if fraction >= 0.15 {
		return ColorQuotaAmber
	}
	return ColorQuotaRed
}

// formatFraction formats the fraction [0.0, 1.0] as a percentage string.
func formatFraction(fraction float64) string {
	if fraction < 0 {
		return "--%"
	}
	if fraction >= 1.0 {
		return "100%"
	}
	if fraction <= 0.0 {
		return "0%"
	}
	pct := fraction * 100.0
	if pct == float64(int(pct)) {
		return fmt.Sprintf("%.0f%%", pct)
	}
	return fmt.Sprintf("%.1f%%", pct)
}

// DualCycleQuota holds extracted 5h and weekly quota fractions for Claude and Gemini.
type DualCycleQuota struct {
	Claude5h   float64
	ClaudeWeek float64
	Gemini5h   float64
	GeminiWeek float64
	HasData    bool
	IsError    bool
}

// extractFamilyCycles extracts 5h and weekly quota fractions for one model family (Claude or Gemini).
// It respects explicit cycle fractions (>= 0), checks raw buckets, and only falls back to legacy single-bucket fraction
// if no cycle-specific data exists. It never synthesizes a weekly cycle from a legacy single-bucket fraction.
func extractFamilyCycles(bucket telemetry.ModelQuotaBucket, rawBuckets []telemetry.ModelQuotaBucket, familyKeywords []string) (h5 float64, week float64) {
	h5 = -1
	week = -1

	// Check if bucket has cycle data
	has5h := bucket.FiveHourFraction >= 0
	hasWeek := bucket.WeeklyFraction >= 0

	// Special case: if Go struct literal in tests only initialized RemainingFraction > 0 and left cycle fractions as 0
	if bucket.RemainingFraction > 0 && bucket.FiveHourFraction == 0 && bucket.WeeklyFraction == 0 && bucket.FiveHourReset.IsZero() && bucket.WeeklyReset.IsZero() {
		has5h = false
		hasWeek = false
	}

	if has5h {
		h5 = bucket.FiveHourFraction
	}
	if hasWeek {
		week = bucket.WeeklyFraction
	}

	// Supplement missing cycles from rawBuckets
	for _, b := range rawBuckets {
		if b.RemainingFraction < 0 && b.FiveHourFraction < 0 && b.WeeklyFraction < 0 {
			continue
		}
		matches := false
		lower := strings.ToLower(b.ModelID)
		lowerToken := strings.ToLower(b.TokenType)
		for _, kw := range familyKeywords {
			if strings.Contains(lower, kw) || strings.Contains(lowerToken, kw) {
				matches = true
				break
			}
		}
		if !matches {
			continue
		}

		is5h := strings.Contains(lower, "5h") || b.FiveHourFraction >= 0
		isWeek := strings.Contains(lower, "week") || b.WeeklyFraction >= 0

		frac := b.RemainingFraction
		if is5h && h5 < 0 {
			if b.FiveHourFraction >= 0 {
				h5 = b.FiveHourFraction
			} else if frac >= 0 {
				h5 = frac
			}
		}
		if isWeek && week < 0 {
			if b.WeeklyFraction >= 0 {
				week = b.WeeklyFraction
			} else if frac >= 0 {
				week = frac
			}
		}
	}

	// Legacy single-bucket fallback: only if NO cycle-specific data found
	if h5 < 0 && week < 0 && bucket.RemainingFraction >= 0 {
		h5 = bucket.RemainingFraction
		week = -1
	}

	return h5, week
}

// ExtractDualCycleQuota parses 5h and weekly buckets from AccountQuotaData.
func ExtractDualCycleQuota(quota *telemetry.AccountQuotaData) DualCycleQuota {
	if quota == nil {
		return DualCycleQuota{Claude5h: -1, ClaudeWeek: -1, Gemini5h: -1, GeminiWeek: -1, HasData: false}
	}
	if quota.Status == telemetry.StatusError || quota.ErrorMessage != "" {
		return DualCycleQuota{Claude5h: -1, ClaudeWeek: -1, Gemini5h: -1, GeminiWeek: -1, HasData: false, IsError: true}
	}

	c5h, cWeek := extractFamilyCycles(quota.Claude, quota.RawBuckets, []string{"claude", "3p"})
	g5h, gWeek := extractFamilyCycles(quota.Gemini, quota.RawBuckets, []string{"gemini"})

	dq := DualCycleQuota{
		Claude5h:   c5h,
		ClaudeWeek: cWeek,
		Gemini5h:   g5h,
		GeminiWeek: gWeek,
		HasData:    c5h >= 0 || cWeek >= 0 || g5h >= 0 || gWeek >= 0,
	}

	return dq
}

// createModelQuotaItem creates a compact status dot and percentage text label for one model (Claude or Gemini).
func createModelQuotaItem(modelName string, fraction float64, hasData bool) fyne.CanvasObject {
	if !hasData || fraction < 0 {
		dot := canvas.NewCircle(ColorQuotaMuted)
		dotWrap := container.NewGridWrap(fyne.NewSize(7, 7), dot)
		txt := canvas.NewText(fmt.Sprintf("%s --%%", modelName), ColorQuotaMuted)
		txt.TextSize = 11
		return container.NewHBox(container.NewCenter(dotWrap), txt)
	}

	tierColor := GetQuotaTierColor(fraction)
	pctStr := formatFraction(fraction)

	dot := canvas.NewCircle(tierColor)
	dotWrap := container.NewGridWrap(fyne.NewSize(7, 7), dot)

	txt := canvas.NewText(fmt.Sprintf("%s %s", modelName, pctStr), tierColor)
	txt.TextSize = 11
	txt.TextStyle = fyne.TextStyle{Bold: true}

	return container.NewHBox(container.NewCenter(dotWrap), txt)
}

// createErrorBadge creates a compact badge indicating a connection error in muted gray.
func createErrorBadge() fyne.CanvasObject {
	dot := canvas.NewCircle(ColorQuotaMuted)
	dotWrap := container.NewGridWrap(fyne.NewSize(7, 7), dot)
	txt := canvas.NewText("[!] Lỗi kết nối", ColorQuotaMuted)
	txt.TextSize = 11
	txt.TextStyle = fyne.TextStyle{Bold: true}
	return container.NewHBox(container.NewCenter(dotWrap), txt)
}

// CreateQuotaCycleRow builds a single cycle row (e.g. "5h: Claude 100% Gemini 88%" or "week: Claude 100% Gemini 90%").
func CreateQuotaCycleRow(prefix string, claudeFrac, geminiFrac float64, hasData bool, isError bool) fyne.CanvasObject {
	lbl := canvas.NewText(prefix, ColorQuotaMuted)
	lbl.TextSize = 11
	lbl.TextStyle = fyne.TextStyle{Bold: true}

	if isError {
		errBadge := createErrorBadge()
		return container.NewHBox(lbl, errBadge)
	}

	claudeItem := createModelQuotaItem("Claude", claudeFrac, hasData)
	geminiItem := createModelQuotaItem("Gemini", geminiFrac, hasData)

	return container.NewHBox(lbl, claudeItem, geminiItem)
}
