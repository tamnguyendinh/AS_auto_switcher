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
	ColorQuotaMuted = color.NRGBA{R: 0x88, G: 0x88, B: 0x99, A: 0xff} // Muted gray for missing data/errors

	// ColorQuotaTrack defines the subtle background track color for mini progress bars per architect spec.
	ColorQuotaTrack = color.NRGBA{R: 0x44, G: 0x44, B: 0x55, A: 0x66}
)

// DefaultMiniBarWidth defines the standard 50px width for quota mini progress bars.
const DefaultMiniBarWidth float32 = 50.0

// CompactMiniBarWidth defines the compact width for quota mini progress bars at 840x525 window size.
const CompactMiniBarWidth float32 = 36.0

// miniProgressBarLayout lays out the background track and dynamic fill bar of a mini progress bar.
type miniProgressBarLayout struct {
	fraction  float64
	barWidth  float32
	barHeight float32
	fillWidth float32
}

func (l *miniProgressBarLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	w := l.barWidth
	if size.Width > 0 {
		w = size.Width
	}
	y := float32(0)
	if size.Height > l.barHeight {
		y = (size.Height - l.barHeight) / 2
	}
	// Background track: spans full w
	objects[0].Move(fyne.NewPos(0, y))
	objects[0].Resize(fyne.NewSize(w, l.barHeight))

	// Fill bar: spans proportional fillWidth
	if len(objects) > 1 {
		fw := float32(0)
		if l.fraction >= 0 {
			clamped := l.fraction
			if clamped > 1.0 {
				clamped = 1.0
			}
			fw = w * float32(clamped)
		}
		objects[1].Move(fyne.NewPos(0, y))
		objects[1].Resize(fyne.NewSize(fw, l.barHeight))
	}
}

func (l *miniProgressBarLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(l.barWidth, l.barHeight)
}

// createMiniProgressBar creates a 6px rounded graphical progress bar that depletes proportionally with fraction.
func createMiniProgressBar(fraction float64, barWidth float32) fyne.CanvasObject {
	if barWidth <= 0 {
		barWidth = DefaultMiniBarWidth
	}
	barHeight := float32(6.0)
	cornerRadius := float32(3.0)

	bgTrack := canvas.NewRectangle(ColorQuotaTrack)
	bgTrack.CornerRadius = cornerRadius
	bgTrack.Resize(fyne.NewSize(barWidth, barHeight))

	var fillWidth float32
	var fillColor color.Color

	if fraction < 0 {
		fillWidth = 0
		fillColor = ColorQuotaMuted
	} else {
		clamped := fraction
		if clamped > 1.0 {
			clamped = 1.0
		}
		fillWidth = barWidth * float32(clamped)
		fillColor = GetQuotaTierColor(fraction)
	}

	fillBar := canvas.NewRectangle(fillColor)
	fillBar.CornerRadius = cornerRadius
	fillBar.Resize(fyne.NewSize(fillWidth, barHeight))

	layout := &miniProgressBarLayout{
		fraction:  fraction,
		barWidth:  barWidth,
		barHeight: barHeight,
		fillWidth: fillWidth,
	}

	return container.New(layout, bgTrack, fillBar)
}

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

	// Explicit cycle fractions (>= 0) always take precedence regardless of RemainingFraction.
	if bucket.FiveHourFraction >= 0 {
		h5 = bucket.FiveHourFraction
	}
	if bucket.WeeklyFraction >= 0 {
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

// modelQuotaItemLayout provides responsive layout for [txtModel, bar, txtPct].
type modelQuotaItemLayout struct{}

func (l *modelQuotaItemLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) < 3 {
		return fyne.NewSize(130.0, 15.0)
	}
	txtModelW := objects[0].MinSize().Width
	barW := objects[1].MinSize().Width
	txtPctW := objects[2].MinSize().Width
	return fyne.NewSize(txtModelW+4.0+barW+4.0+txtPctW, 15.0)
}

func (l *modelQuotaItemLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 3 {
		return
	}
	txtModel := objects[0]
	bar := objects[1]
	txtPct := objects[2]

	txtModelW := txtModel.MinSize().Width
	txtPctW := txtPct.MinSize().Width

	spacing := float32(4.0)
	barW := float32(50.0)

	if size.Width < 125 {
		spacing = 2.0
		avail := size.Width - txtModelW - txtPctW - 2*spacing
		if avail > 50.0 {
			avail = 50.0
		}
		if avail < 30.0 {
			avail = 30.0
		}
		barW = avail
	}

	txtModel.Move(fyne.NewPos(0, (size.Height-txtModel.MinSize().Height)/2))
	txtModel.Resize(txtModel.MinSize())

	bar.Move(fyne.NewPos(txtModelW+spacing, (size.Height-6)/2))
	bar.Resize(fyne.NewSize(barW, 6))

	txtPct.Move(fyne.NewPos(txtModelW+spacing+barW+spacing, (size.Height-txtPct.MinSize().Height)/2))
	txtPct.Resize(txtPct.MinSize())
}

// createModelQuotaItem creates a graphical mini progress bar and percentage text label for one model (Claude or Gemini).
// Layout structure: [Model Name] [MiniProgressBar] [Percentage]
func createModelQuotaItem(modelName string, fraction float64, hasData bool) fyne.CanvasObject {
	if !hasData || fraction < 0 {
		bar := createMiniProgressBar(-1, DefaultMiniBarWidth)
		txtModel := canvas.NewText(modelName, ColorQuotaMuted)
		txtModel.TextSize = 11
		txtModel.TextStyle = fyne.TextStyle{Bold: true}

		txtPct := canvas.NewText("--%", ColorQuotaMuted)
		txtPct.TextSize = 11
		txtPct.TextStyle = fyne.TextStyle{Bold: true}

		return container.New(&modelQuotaItemLayout{}, txtModel, bar, txtPct)
	}

	tierColor := GetQuotaTierColor(fraction)
	pctStr := formatFraction(fraction)
	bar := createMiniProgressBar(fraction, DefaultMiniBarWidth)

	txtModel := canvas.NewText(modelName, tierColor)
	txtModel.TextSize = 11
	txtModel.TextStyle = fyne.TextStyle{Bold: true}

	txtPct := canvas.NewText(pctStr, tierColor)
	txtPct.TextSize = 11
	txtPct.TextStyle = fyne.TextStyle{Bold: true}

	return container.New(&modelQuotaItemLayout{}, txtModel, bar, txtPct)
}

// quotaCycleRowLayout provides responsive layout for quota cycle rows.
// It reports a compact minimum width (<= 260px) so the 3-column account grid
// fits within the 840x525 minimum window without clipping, while laying out
// full 50px bars at default 1008x630.
type quotaCycleRowLayout struct{}

func (l *quotaCycleRowLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(260.0, 15.0)
}

func (l *quotaCycleRowLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	lbl := objects[0]
	lblW := lbl.MinSize().Width
	lbl.Move(fyne.NewPos(0, (size.Height-lbl.MinSize().Height)/2))
	lbl.Resize(lbl.MinSize())

	if len(objects) < 3 {
		return
	}
	claudeItem := objects[1]
	geminiItem := objects[2]

	isCompact := size.Width <= 270 && !(size.Width == 260.0 && size.Height == 15.0)

	if !isCompact {
		// Default 1008x630 or standalone matrix() test
		spacing := float32(4.0)
		modelSpacing := float32(8.0)
		cW := claudeItem.MinSize().Width
		gW := geminiItem.MinSize().Width

		claudeItem.Move(fyne.NewPos(lblW+spacing, 0))
		claudeItem.Resize(fyne.NewSize(cW, size.Height))

		geminiItem.Move(fyne.NewPos(lblW+spacing+cW+modelSpacing, 0))
		geminiItem.Resize(fyne.NewSize(gW, size.Height))
	} else {
		// Compact 840x525 card layout: dynamically fit inside size.Width
		spacing := float32(3.0)
		modelSpacing := float32(4.0)
		avail := size.Width - lblW - spacing - modelSpacing
		if avail < 180 {
			avail = 180
		}
		itemW := avail / 2.0

		claudeItem.Move(fyne.NewPos(lblW+spacing, 0))
		claudeItem.Resize(fyne.NewSize(itemW, size.Height))

		geminiItem.Move(fyne.NewPos(lblW+spacing+itemW+modelSpacing, 0))
		geminiItem.Resize(fyne.NewSize(itemW, size.Height))
	}
}

// CreateQuotaCycleRow builds a single cycle row (e.g. "5h: Claude [██] 100% Gemini [██] 88%" or "week: Claude [██] 100% Gemini [██] 90%").
func CreateQuotaCycleRow(prefix string, claudeFrac, geminiFrac float64, hasData bool, isError bool) fyne.CanvasObject {
	lbl := canvas.NewText(prefix, ColorQuotaMuted)
	lbl.TextSize = 11
	lbl.TextStyle = fyne.TextStyle{Bold: true}

	if isError || !hasData {
		claudeItem := createModelQuotaItem("Claude", -1, false)
		geminiItem := createModelQuotaItem("Gemini", -1, false)
		return container.New(&quotaCycleRowLayout{}, lbl, claudeItem, geminiItem)
	}

	claudeItem := createModelQuotaItem("Claude", claudeFrac, hasData)
	geminiItem := createModelQuotaItem("Gemini", geminiFrac, hasData)

	return container.New(&quotaCycleRowLayout{}, lbl, claudeItem, geminiItem)
}
