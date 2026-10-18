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

// ExtractDualCycleQuota parses 5h and weekly buckets from AccountQuotaData.
func ExtractDualCycleQuota(quota *telemetry.AccountQuotaData) DualCycleQuota {
	if quota == nil {
		return DualCycleQuota{Claude5h: -1, ClaudeWeek: -1, Gemini5h: -1, GeminiWeek: -1, HasData: false}
	}
	if quota.Status == telemetry.StatusError || quota.ErrorMessage != "" {
		return DualCycleQuota{Claude5h: -1, ClaudeWeek: -1, Gemini5h: -1, GeminiWeek: -1, HasData: false, IsError: true}
	}

	dq := DualCycleQuota{
		Claude5h:   -1,
		ClaudeWeek: -1,
		Gemini5h:   -1,
		GeminiWeek: -1,
	}

	hasAnyBucket := false

	// Priority 1: Extract from RawBuckets if available
	for _, b := range quota.RawBuckets {
		if b.RemainingFraction < 0 {
			continue
		}
		lower := strings.ToLower(b.ModelID)
		is5h := strings.Contains(lower, "5h")
		isWeek := strings.Contains(lower, "week")

		if strings.Contains(lower, "gemini") {
			if is5h {
				dq.Gemini5h = b.RemainingFraction
				hasAnyBucket = true
			} else if isWeek {
				dq.GeminiWeek = b.RemainingFraction
				hasAnyBucket = true
			}
		} else if strings.Contains(lower, "claude") || strings.Contains(lower, "3p") {
			if is5h {
				dq.Claude5h = b.RemainingFraction
				hasAnyBucket = true
			} else if isWeek {
				dq.ClaudeWeek = b.RemainingFraction
				hasAnyBucket = true
			}
		}
	}

	// Priority 2: Extract from ModelQuotaBucket FiveHourFraction and WeeklyFraction if explicitly set (>= 0)
	hasClaudeCycles := (quota.Claude.FiveHourFraction > 0 || quota.Claude.WeeklyFraction > 0 || !quota.Claude.FiveHourReset.IsZero() || !quota.Claude.WeeklyReset.IsZero()) ||
		(quota.Claude.FiveHourFraction == 0 && (quota.Claude.WeeklyFraction != 0 || quota.Claude.RemainingFraction == 0))
	if hasClaudeCycles {
		if quota.Claude.FiveHourFraction >= 0 && dq.Claude5h < 0 {
			dq.Claude5h = quota.Claude.FiveHourFraction
			hasAnyBucket = true
		}
		if quota.Claude.WeeklyFraction >= 0 && dq.ClaudeWeek < 0 {
			dq.ClaudeWeek = quota.Claude.WeeklyFraction
			hasAnyBucket = true
		}
	}

	hasGeminiCycles := (quota.Gemini.FiveHourFraction > 0 || quota.Gemini.WeeklyFraction > 0 || !quota.Gemini.FiveHourReset.IsZero() || !quota.Gemini.WeeklyReset.IsZero()) ||
		(quota.Gemini.FiveHourFraction == 0 && (quota.Gemini.WeeklyFraction != 0 || quota.Gemini.RemainingFraction == 0))
	if hasGeminiCycles {
		if quota.Gemini.FiveHourFraction >= 0 && dq.Gemini5h < 0 {
			dq.Gemini5h = quota.Gemini.FiveHourFraction
			hasAnyBucket = true
		}
		if quota.Gemini.WeeklyFraction >= 0 && dq.GeminiWeek < 0 {
			dq.GeminiWeek = quota.Gemini.WeeklyFraction
			hasAnyBucket = true
		}
	}

	// Priority 3: Fallback for legacy single-bucket data if no cycle-specific fraction found
	if !hasAnyBucket && (quota.Claude.RemainingFraction >= 0 || quota.Gemini.RemainingFraction >= 0) {
		if quota.Claude.RemainingFraction >= 0 {
			dq.Claude5h = quota.Claude.RemainingFraction
			dq.ClaudeWeek = -1 // Never synthesize weekly cycle from legacy single bucket
			hasAnyBucket = true
		}
		if quota.Gemini.RemainingFraction >= 0 {
			dq.Gemini5h = quota.Gemini.RemainingFraction
			dq.GeminiWeek = -1 // Never synthesize weekly cycle from legacy single bucket
			hasAnyBucket = true
		}
	}

	dq.HasData = hasAnyBucket
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

// NewDualCycleQuotaBadge constructs a 2-row container:
// Row 1: "5h: Claude <pct>% Gemini <pct>%"
// Row 2: "week: Claude <pct>% Gemini <pct>%"
// GPT is completely excluded.
func NewDualCycleQuotaBadge(quota *telemetry.AccountQuotaData) fyne.CanvasObject {
	dq := ExtractDualCycleQuota(quota)
	r5h := CreateQuotaCycleRow("5h:", dq.Claude5h, dq.Gemini5h, dq.HasData, dq.IsError)
	rWeek := CreateQuotaCycleRow("week:", dq.ClaudeWeek, dq.GeminiWeek, dq.HasData, dq.IsError)
	return container.NewVBox(r5h, rWeek)
}

// NewQuotaBadgeGroup delegates to NewDualCycleQuotaBadge for backwards compatibility with tests.
func NewQuotaBadgeGroup(quota *telemetry.AccountQuotaData) fyne.CanvasObject {
	return NewDualCycleQuotaBadge(quota)
}

// BuildActiveQuotaSummary builds a one-line summary string of active quota (Claude and Gemini only, no GPT).
func BuildActiveQuotaSummary(quota *telemetry.AccountQuotaData) string {
	if quota == nil {
		return "Hạn ngạch: Chưa tải dữ liệu"
	}
	if quota.Status == telemetry.StatusError || quota.ErrorMessage != "" {
		return "Hạn ngạch: [!] Lỗi kết nối"
	}
	return fmt.Sprintf("Hạn ngạch: 🟣 Claude: %s | 🔵 Gemini: %s",
		formatFraction(quota.Claude.RemainingFraction),
		formatFraction(quota.Gemini.RemainingFraction),
	)
}
