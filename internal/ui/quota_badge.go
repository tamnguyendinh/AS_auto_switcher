package ui

import (
	"fmt"
	"image/color"
	"strings"
	"time"

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

// FormatRemainingCountdown formats the time remaining until resetTime as (còn Xh Yp).
func FormatRemainingCountdown(resetTime time.Time, now time.Time) string {
	if resetTime.IsZero() {
		return "đang chờ hồi phục"
	}

	dur := resetTime.Sub(now)
	if dur <= 0 {
		return "sắp hồi phục"
	}

	hours := int(dur.Hours())
	mins := int(dur.Minutes()) % 60

	if hours > 0 {
		return fmt.Sprintf("còn %dh %dp", hours, mins)
	}
	if mins > 0 {
		return fmt.Sprintf("còn %dp", mins)
	}
	return "dưới 1 phút"
}

// formatFraction formats the fraction [0.0, 1.0] as a percentage string.
func formatFraction(fraction float64) string {
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
		return DualCycleQuota{HasData: false}
	}
	if quota.Status == telemetry.StatusError || quota.ErrorMessage != "" {
		return DualCycleQuota{HasData: false, IsError: true}
	}

	dq := DualCycleQuota{
		HasData:    true,
		Claude5h:   quota.Claude.RemainingFraction,
		ClaudeWeek: quota.Claude.RemainingFraction,
		Gemini5h:   quota.Gemini.RemainingFraction,
		GeminiWeek: quota.Gemini.RemainingFraction,
	}

	for _, b := range quota.RawBuckets {
		lower := strings.ToLower(b.ModelID)
		if strings.Contains(lower, "gemini") {
			if strings.Contains(lower, "5h") {
				dq.Gemini5h = b.RemainingFraction
			} else if strings.Contains(lower, "week") {
				dq.GeminiWeek = b.RemainingFraction
			}
		} else if strings.Contains(lower, "claude") || strings.Contains(lower, "3p") {
			if strings.Contains(lower, "5h") {
				dq.Claude5h = b.RemainingFraction
			} else if strings.Contains(lower, "week") {
				dq.ClaudeWeek = b.RemainingFraction
			}
		}
	}

	return dq
}

// createModelQuotaItem creates a compact status dot and percentage text label for one model (Claude or Gemini).
func createModelQuotaItem(modelName string, fraction float64, hasData bool) fyne.CanvasObject {
	if !hasData {
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

// NewQuotaCountdownLabel returns nil as countdown warnings are removed in the redesigned UI.
func NewQuotaCountdownLabel(quota *telemetry.AccountQuotaData) fyne.CanvasObject {
	return nil
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
