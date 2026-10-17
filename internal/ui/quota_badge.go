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
	// Three-tier quota colors per UI-TR-04
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
	// Show one decimal if not whole number
	if pct == float64(int(pct)) {
		return fmt.Sprintf("%.0f%%", pct)
	}
	return fmt.Sprintf("%.1f%%", pct)
}

// createModelBadge creates a compact status dot and percentage text label for one model.
func createModelBadge(modelName string, fraction float64, hasData bool) fyne.CanvasObject {
	if !hasData {
		txt := canvas.NewText(fmt.Sprintf("%s: --%%", modelName), ColorQuotaMuted)
		txt.TextSize = 11
		return txt
	}

	tierColor := GetQuotaTierColor(fraction)
	pctStr := formatFraction(fraction)

	dot := canvas.NewCircle(tierColor)
	dotWrap := container.NewGridWrap(fyne.NewSize(8, 8), dot)

	txt := canvas.NewText(fmt.Sprintf("%s: %s", modelName, pctStr), tierColor)
	txt.TextSize = 11
	txt.TextStyle = fyne.TextStyle{Bold: true}

	return container.NewHBox(container.NewCenter(dotWrap), txt)
}

// createErrorBadge creates a compact badge indicating a connection error in muted gray.
func createErrorBadge() fyne.CanvasObject {
	dot := canvas.NewCircle(ColorQuotaMuted)
	dotWrap := container.NewGridWrap(fyne.NewSize(8, 8), dot)
	txt := canvas.NewText("[!] Lỗi kết nối", ColorQuotaMuted)
	txt.TextSize = 11
	txt.TextStyle = fyne.TextStyle{Bold: true}
	return container.NewHBox(container.NewCenter(dotWrap), txt)
}

// NewQuotaBadgeGroup constructs a 3-column widget containing mini quota badges for Claude, Gemini, and GPT.
// If quota is nil or has an Error status, displays muted placeholders or error badges (never 0% red).
func NewQuotaBadgeGroup(quota *telemetry.AccountQuotaData) fyne.CanvasObject {
	if quota == nil {
		c1 := createModelBadge("Claude", 0, false)
		c2 := createModelBadge("Gemini", 0, false)
		c3 := createModelBadge("GPT", 0, false)
		return container.NewGridWithColumns(3, c1, c2, c3)
	}

	if quota.Status == telemetry.StatusError || quota.ErrorMessage != "" {
		errBadge := createErrorBadge()
		c2 := createModelBadge("Gemini", 0, false)
		c3 := createModelBadge("GPT", 0, false)
		return container.NewGridWithColumns(3, errBadge, c2, c3)
	}

	claudeBadge := createModelBadge("Claude", quota.Claude.RemainingFraction, true)
	geminiBadge := createModelBadge("Gemini", quota.Gemini.RemainingFraction, true)
	gptBadge := createModelBadge("GPT", quota.GPT.RemainingFraction, true)

	return container.NewGridWithColumns(3, claudeBadge, geminiBadge, gptBadge)
}

// NewQuotaCountdownLabel returns a countdown warning text if any model has quota < 15%.
// Returns nil if all models have >= 15% quota, if quota is nil, or if quota is in Error status.
func NewQuotaCountdownLabel(quota *telemetry.AccountQuotaData) fyne.CanvasObject {
	if quota == nil || quota.Status == telemetry.StatusError || quota.ErrorMessage != "" {
		return nil
	}

	var lowModels []string
	var targetReset time.Time

	if quota.Gemini.RemainingFraction < 0.15 {
		lowModels = append(lowModels, "Gemini")
		if !quota.Gemini.ResetTime.IsZero() {
			targetReset = quota.Gemini.ResetTime
		}
	}
	if quota.Claude.RemainingFraction < 0.15 {
		lowModels = append(lowModels, "Claude")
		if targetReset.IsZero() && !quota.Claude.ResetTime.IsZero() {
			targetReset = quota.Claude.ResetTime
		}
	}
	if quota.GPT.RemainingFraction < 0.15 {
		lowModels = append(lowModels, "GPT-OSS")
		if targetReset.IsZero() && !quota.GPT.ResetTime.IsZero() {
			targetReset = quota.GPT.ResetTime
		}
	}

	if len(lowModels) == 0 {
		return nil
	}

	if targetReset.IsZero() && !quota.ResetTime.IsZero() {
		targetReset = quota.ResetTime
	}

	modelStr := strings.Join(lowModels, ", ")
	now := time.Now()

	var msg string
	if !targetReset.IsZero() {
		countdown := FormatRemainingCountdown(targetReset, now)
		resetStr := targetReset.UTC().Format("15:04")
		msg = fmt.Sprintf("⚠️ %s sắp cạn (Google sẽ hồi phục lúc: %s UTC - %s)", modelStr, resetStr, countdown)
	} else {
		msg = fmt.Sprintf("⚠️ %s sắp cạn (Đang chờ thông tin hồi phục từ Google)", modelStr)
	}

	txt := canvas.NewText(msg, ColorQuotaRed)
	txt.TextSize = 11
	txt.TextStyle = fyne.TextStyle{Italic: true}

	return txt
}

// BuildActiveQuotaSummary builds a one-line summary string of active quota for the header card.
func BuildActiveQuotaSummary(quota *telemetry.AccountQuotaData) string {
	if quota == nil {
		return "Hạn ngạch: Chưa tải dữ liệu"
	}
	if quota.Status == telemetry.StatusError || quota.ErrorMessage != "" {
		return "Hạn ngạch: [!] Lỗi kết nối"
	}
	return fmt.Sprintf("Hạn ngạch: 🟣 Claude: %s | 🔵 Gemini: %s | ⚪ GPT: %s",
		formatFraction(quota.Claude.RemainingFraction),
		formatFraction(quota.Gemini.RemainingFraction),
		formatFraction(quota.GPT.RemainingFraction),
	)
}