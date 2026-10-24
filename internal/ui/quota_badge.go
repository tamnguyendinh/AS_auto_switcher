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
	ColorQuotaMuted = color.NRGBA{R: 0x88, G: 0x88, B: 0x99, A: 0xff} // Muted gray for missing data/errors

	// ColorQuotaTrack defines the subtle background track color for mini progress bars per architect spec.
	ColorQuotaTrack = color.NRGBA{R: 0x44, G: 0x44, B: 0x55, A: 0x66}

	// ColorTierUltra: Vàng Amber nổi bật rực rỡ (#f59e0b)
	ColorTierUltra = color.NRGBA{R: 0xf5, G: 0x9e, B: 0x0b, A: 0xff}

	// ColorTierBiz: Xanh lục doanh nghiệp (#10b981)
	ColorTierBiz = color.NRGBA{R: 0x10, G: 0xb9, B: 0x81, A: 0xff}
)

// TierColor determines the visual hierarchy color for a given account tier code.
func TierColor(tier string, palette ThemePalette) color.Color {
	switch strings.ToUpper(strings.TrimSpace(tier)) {
	case "ULTRA":
		return ColorTierUltra
	case "PRO":
		return palette.Primary
	case "BIZ":
		return ColorTierBiz
	case "FREE":
		return palette.TextMuted
	default:
		return palette.TextMuted
	}
}

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
	Claude5h        float64
	ClaudeWeek      float64
	Gemini5h        float64
	GeminiWeek      float64
	Claude5hReset   time.Time
	ClaudeWeekReset time.Time
	Gemini5hReset   time.Time
	GeminiWeekReset time.Time
	HasData         bool
	IsError         bool
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
		Claude5h:        c5h,
		ClaudeWeek:      cWeek,
		Gemini5h:        g5h,
		GeminiWeek:      gWeek,
		Claude5hReset:   quota.Claude.FiveHourReset,
		ClaudeWeekReset: quota.Claude.WeeklyReset,
		Gemini5hReset:   quota.Gemini.FiveHourReset,
		GeminiWeekReset: quota.Gemini.WeeklyReset,
		HasData:         c5h >= 0 || cWeek >= 0 || g5h >= 0 || gWeek >= 0,
	}

	return dq
}

// formatCountdown formats the time remaining until quota bucket reset per architect spec.
func formatCountdown(resetTime time.Time, isWeekly bool) string {
	if resetTime.IsZero() {
		return "--"
	}
	duration := time.Until(resetTime)
	if duration <= 0 {
		return "(Ready)"
	}
	if !isWeekly {
		h := int(duration.Hours())
		m := int(duration.Minutes()) % 60
		return fmt.Sprintf("(%02dh %02dm)", h, m)
	}
	d := int(duration.Hours()) / 24
	h := int(duration.Hours()) % 24
	m := int(duration.Minutes()) % 60
	return fmt.Sprintf("(%dd %02dh %02dm)", d, h, m)
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
		txtModel.TextSize = BodyTextSize
		txtModel.TextStyle = fyne.TextStyle{Bold: true}

		txtPct := canvas.NewText("--%", ColorQuotaMuted)
		txtPct.TextSize = BodyTextSize
		txtPct.TextStyle = fyne.TextStyle{Bold: true}

		return container.New(&modelQuotaItemLayout{}, txtModel, bar, txtPct)
	}

	tierColor := GetQuotaTierColor(fraction)
	pctStr := formatFraction(fraction)
	bar := createMiniProgressBar(fraction, DefaultMiniBarWidth)

	txtModel := canvas.NewText(modelName, tierColor)
	txtModel.TextSize = BodyTextSize
	txtModel.TextStyle = fyne.TextStyle{Bold: true}

	txtPct := canvas.NewText(pctStr, tierColor)
	txtPct.TextSize = BodyTextSize
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
	lbl.TextSize = BodyTextSize
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

// MatrixMiniBarWidth defines the 40px width for 3x3 matrix quota mini progress bars per architect spec.
const MatrixMiniBarWidth float32 = 40.0

// matrixStackedCellLayout lays out 2 tiers:
// Tier 1 (top): timeText (height 13px)
// Tier 2 (bottom): [miniBar (40px)] [gap 4px] [pctLabel] (height 15px)
// Total cell height = 28px per architect spec.
type matrixStackedCellLayout struct {
	barWidth float32
}

func (l *matrixStackedCellLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	bw := l.barWidth
	if bw <= 0 {
		bw = MatrixMiniBarWidth
	}
	var timeW, pctW float32
	for _, o := range objects {
		if txt, ok := o.(*canvas.Text); ok {
			if txt.TextSize < 9.0 {
				timeW = txt.MinSize().Width
			} else {
				pctW = txt.MinSize().Width
			}
		}
	}
	tier2W := bw + 4.0 + pctW
	maxW := tier2W
	if timeW > maxW {
		maxW = timeW
	}
	return fyne.NewSize(maxW, 28.0)
}

func (l *matrixStackedCellLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	bw := l.barWidth
	if bw <= 0 {
		bw = MatrixMiniBarWidth
	}
	const spacing = float32(4.0)

	var timeObj, pctObj *canvas.Text
	var barCont fyne.CanvasObject

	for _, o := range objects {
		if c, ok := o.(*fyne.Container); ok {
			barCont = c
		} else if txt, ok := o.(*canvas.Text); ok {
			if txt.TextSize < 9.0 {
				timeObj = txt
			} else {
				pctObj = txt
			}
		}
	}

	// Fallback to index-based if needed
	if timeObj == nil && len(objects) >= 1 {
		if txt, ok := objects[0].(*canvas.Text); ok {
			timeObj = txt
		}
	}
	if barCont == nil && len(objects) >= 2 {
		barCont = objects[1]
	}
	if pctObj == nil && len(objects) >= 3 {
		if txt, ok := objects[2].(*canvas.Text); ok {
			pctObj = txt
		}
	}

	// Tier 1: timeText (height 13px, y = 0)
	if timeObj != nil {
		tH := timeObj.MinSize().Height
		y := (13.0 - tH) / 2
		if y < 0 {
			y = 0
		}
		timeObj.Move(fyne.NewPos(0, y))
		timeObj.Resize(fyne.NewSize(size.Width, 13.0))
	}

	// Tier 2: barContainer + pctText (height 15px, y = 13.0)
	const tier2Y = float32(13.0)
	const tier2H = float32(15.0)

	if barCont != nil {
		barH := float32(6.0)
		yBar := tier2Y + (tier2H-barH)/2
		barCont.Move(fyne.NewPos(0, yBar))
		barCont.Resize(fyne.NewSize(bw, barH))
	}

	if pctObj != nil {
		pH := pctObj.MinSize().Height
		pW := pctObj.MinSize().Width
		yPct := tier2Y + (tier2H-pH)/2
		pctObj.Move(fyne.NewPos(bw+spacing, yPct))
		pctObj.Resize(fyne.NewSize(pW, pH))
	}
}

// matrixStackedModelCellLayout aligns model names (Gemini, Claude) with the bottom tier (y = 13px)
// of stacked quota cells, leaving the top tier (0..13px) blank for visual baseline symmetry.
type matrixStackedModelCellLayout struct{}

func (l *matrixStackedModelCellLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.NewSize(48.0, 28.0)
	}
	return fyne.NewSize(objects[0].MinSize().Width, 28.0)
}

func (l *matrixStackedModelCellLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	txt := objects[0]
	txtH := txt.MinSize().Height
	y := float32(13.0) + (15.0-txtH)/2
	if y < 13.0 {
		y = 13.0
	}
	txt.Move(fyne.NewPos(0, y))
	txt.Resize(fyne.NewSize(size.Width, txtH))
}

type matrixQuotaCell struct {
	container *fyne.Container
	layout    *miniProgressBarLayout
	bgTrack   *canvas.Rectangle
	fillBar   *canvas.Rectangle
	pctText   *canvas.Text
	timeText  *canvas.Text
	isWeekly  bool
}

func newMatrixQuotaCell(fraction float64, resetTime time.Time, isWeekly bool, hasData bool, palette ThemePalette) *matrixQuotaCell {
	barWidth := MatrixMiniBarWidth
	barHeight := float32(6.0)
	cornerRadius := float32(3.0)

	bgTrack := canvas.NewRectangle(ColorQuotaTrack)
	bgTrack.CornerRadius = cornerRadius
	bgTrack.Resize(fyne.NewSize(barWidth, barHeight))

	var fillWidth float32
	var fillColor color.Color
	var pctStr string
	var pctColor color.Color

	if !hasData || fraction < 0 {
		fillWidth = 0
		fillColor = ColorQuotaMuted
		pctStr = "--%"
		pctColor = ColorQuotaMuted
	} else {
		clamped := fraction
		if clamped > 1.0 {
			clamped = 1.0
		} else if clamped < 0 {
			clamped = 0
		}
		fillWidth = barWidth * float32(clamped)
		fillColor = GetQuotaTierColor(fraction)
		pctStr = formatFraction(fraction)
		pctColor = fillColor
	}

	fillBar := canvas.NewRectangle(fillColor)
	fillBar.CornerRadius = cornerRadius
	fillBar.Resize(fyne.NewSize(fillWidth, barHeight))

	pbLayout := &miniProgressBarLayout{
		fraction:  fraction,
		barWidth:  barWidth,
		barHeight: barHeight,
		fillWidth: fillWidth,
	}

	barContainer := container.New(pbLayout, bgTrack, fillBar)

	pct := canvas.NewText(pctStr, pctColor)
	pct.TextSize = BodyTextSize
	pct.TextStyle = fyne.TextStyle{Bold: true}

	var timeStr string
	if !hasData {
		timeStr = "--"
	} else {
		timeStr = formatCountdown(resetTime, isWeekly)
	}
	timeTxt := canvas.NewText(timeStr, palette.TextMuted)
	timeTxt.TextSize = 8.5

	cellContainer := container.New(&matrixStackedCellLayout{barWidth: MatrixMiniBarWidth}, timeTxt, barContainer, pct)

	return &matrixQuotaCell{
		container: cellContainer,
		layout:    pbLayout,
		bgTrack:   bgTrack,
		fillBar:   fillBar,
		pctText:   pct,
		timeText:  timeTxt,
		isWeekly:  isWeekly,
	}
}

func (c *matrixQuotaCell) update(fraction float64, resetTime time.Time, hasData bool) {
	if c == nil {
		return
	}
	const barWidth = float32(MatrixMiniBarWidth) // Ray tiến trình CỐ ĐỊNH 40.0 px
	c.layout.fraction = fraction
	if !hasData {
		c.layout.fraction = -1
	}

	if !hasData || fraction < 0 {
		c.pctText.Text = "--%"
		c.pctText.Color = ColorQuotaMuted
		c.fillBar.FillColor = ColorQuotaMuted
		c.fillBar.Resize(fyne.NewSize(0, c.layout.barHeight))
	} else {
		clamped := fraction
		if clamped > 1.0 {
			clamped = 1.0
		} else if clamped < 0 {
			clamped = 0
		}
		tierColor := GetQuotaTierColor(fraction)
		c.pctText.Text = formatFraction(fraction)
		c.pctText.Color = tierColor

		// CHỈ DUY NHẤT fillBar THAY ĐỔI CHIỀU DÀI THEO % QUOTA (TỤT DẦN)
		fw := barWidth * float32(clamped)
		c.fillBar.FillColor = tierColor
		c.fillBar.Resize(fyne.NewSize(fw, c.layout.barHeight))
	}

	if c.timeText != nil {
		if !hasData {
			c.timeText.Text = "--"
		} else {
			c.timeText.Text = formatCountdown(resetTime, c.isWeekly)
		}
		c.timeText.Refresh()
	}

	c.fillBar.Refresh()
	c.pctText.Refresh()
	// TUYỆT ĐỐI KHÔNG GỌI c.container.Refresh()
}

func (c *matrixQuotaCell) setUnavailable() {
	c.update(-1, time.Time{}, false)
	c.pctText.Text = "N/A"
	c.pctText.Refresh()
}

// createMatrixQuotaCell creates a graphical mini progress bar (40px) and percentage text label.
func createMatrixQuotaCell(fraction float64, hasData bool, palette ThemePalette) fyne.CanvasObject {
	cell := newMatrixQuotaCell(fraction, time.Time{}, false, hasData, palette)
	return cell.container
}

// matrixColumnLayout lays out 3 rows in a column with fixed row heights and subtle spacing:
// Row 0 (header): height 15px, y = 0
// Gap under header: 2px
// Row 1 (Gemini): height 28px, y = 17px
// Gap between Gemini & Claude: 4px
// Row 2 (Claude): height 28px, y = 49px
// Total column height = 77px
type matrixColumnLayout struct{}

func (l *matrixColumnLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	maxW := float32(0)
	for _, o := range objects {
		ms := o.MinSize()
		if ms.Width > maxW {
			maxW = ms.Width
		}
	}
	return fyne.NewSize(maxW, 77.0)
}

func (l *matrixColumnLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 3 {
		return
	}
	// Row 0: Header (15px)
	objects[0].Move(fyne.NewPos(0, 0))
	objects[0].Resize(fyne.NewSize(size.Width, 15.0))

	// Row 1: Gemini (28px)
	objects[1].Move(fyne.NewPos(0, 17.0))
	objects[1].Resize(fyne.NewSize(size.Width, 28.0))

	// Row 2: Claude (28px)
	objects[2].Move(fyne.NewPos(0, 49.0))
	objects[2].Resize(fyne.NewSize(size.Width, 28.0))
}

// matrixTableLayout provides responsive layout for the 3x3 quota matrix:
// [colModel, col5h, colWeek, (optional) hSep]
type matrixTableLayout struct{}

func (l *matrixTableLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	const totalW = float32(48.0 + 8.0 + 78.0 + 8.0 + 86.0) // 228.0 px
	return fyne.NewSize(totalW, 77.0)
}

func (l *matrixTableLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 3 {
		return
	}
	colModel := objects[0]
	col5h := objects[1]
	colWeek := objects[2]
	h := size.Height
	if h < 77.0 {
		h = 77.0
	}
	// CHỈ SỐ CỐ ĐỊNH CHUẨN NGÀNH - TUYỆT ĐỐI KHÔNG CO GIÃN
	const col1W = float32(48.0)    // Cột Model: CỐ ĐỊNH 48px
	const colGap = float32(8.0)    // Khoảng cách tự nhiên giữa các cột: CỐ ĐỊNH 8px
	const col5hW = float32(78.0)   // Cột 5h: CỐ ĐỊNH 78px
	const colWeekW = float32(86.0) // Cột Week: CỐ ĐỊNH 86px

	x := float32(0)
	// Col 1: Model (Gemini / Claude)
	colModel.Move(fyne.NewPos(x, 0))
	colModel.Resize(fyne.NewSize(col1W, h))
	x += col1W + colGap

	// Col 2: 5h
	col5h.Move(fyne.NewPos(x, 0))
	col5h.Resize(fyne.NewSize(col5hW, h))
	x += col5hW + colGap

	// Col 3: Week
	colWeek.Move(fyne.NewPos(x, 0))
	colWeek.Resize(fyne.NewSize(colWeekW, h))

	// Horizontal Separator (between Gemini and Claude)
	if len(objects) >= 4 {
		hSep := objects[3]
		hSep.Move(fyne.NewPos(0, 46.0))
		hSep.Resize(fyne.NewSize(x+colWeekW, 1.0))
	}
}

func createMatrixQuotaTableWithCells(
	dq DualCycleQuota,
	tier string,
	palette ThemePalette,
) (fyne.CanvasObject, *canvas.Text, *matrixQuotaCell, *matrixQuotaCell, *matrixQuotaCell, *matrixQuotaCell) {
	// Col 1: Tier Label & Model Labels
	displayTier := telemetry.NormalizeTier(tier)
	txtTier := canvas.NewText(displayTier, TierColor(displayTier, palette))
	txtTier.TextSize = BodyTextSize
	txtTier.TextStyle = fyne.TextStyle{Bold: true}

	txtGemini := canvas.NewText("Gemini", palette.Text)
	txtGemini.TextSize = BodyTextSize
	txtGemini.TextStyle = fyne.TextStyle{Bold: true}

	txtClaude := canvas.NewText("Claude", palette.Text)
	txtClaude.TextSize = BodyTextSize
	txtClaude.TextStyle = fyne.TextStyle{Bold: true}

	cellGeminiModel := container.New(&matrixStackedModelCellLayout{}, txtGemini)
	cellClaudeModel := container.New(&matrixStackedModelCellLayout{}, txtClaude)

	// Col 1: Thay txtQuota bằng txtTier
	colModel := container.New(&matrixColumnLayout{}, txtTier, cellGeminiModel, cellClaudeModel)

	// Col 2: 5h Cycle
	txt5h := canvas.NewText("5h", palette.TextMuted)
	txt5h.TextSize = BodyTextSize
	txt5h.TextStyle = fyne.TextStyle{Bold: true}

	hasGemini5h := !dq.IsError && dq.HasData && dq.Gemini5h >= 0
	cellGemini5h := newMatrixQuotaCell(dq.Gemini5h, dq.Gemini5hReset, false, hasGemini5h, palette)

	hasClaude5h := !dq.IsError && dq.HasData && dq.Claude5h >= 0
	cellClaude5h := newMatrixQuotaCell(dq.Claude5h, dq.Claude5hReset, false, hasClaude5h, palette)

	col5h := container.New(&matrixColumnLayout{}, txt5h, cellGemini5h.container, cellClaude5h.container)

	// Col 3: Week Cycle
	txtWeek := canvas.NewText("Week", palette.TextMuted)
	txtWeek.TextSize = BodyTextSize
	txtWeek.TextStyle = fyne.TextStyle{Bold: true}

	hasGeminiWeek := !dq.IsError && dq.HasData && dq.GeminiWeek >= 0
	cellGeminiWeek := newMatrixQuotaCell(dq.GeminiWeek, dq.GeminiWeekReset, true, hasGeminiWeek, palette)

	hasClaudeWeek := !dq.IsError && dq.HasData && dq.ClaudeWeek >= 0
	cellClaudeWeek := newMatrixQuotaCell(dq.ClaudeWeek, dq.ClaudeWeekReset, true, hasClaudeWeek, palette)

	colWeek := container.New(&matrixColumnLayout{}, txtWeek, cellGeminiWeek.container, cellClaudeWeek.container)

	// Horizontal Separator (between Gemini and Claude)
	hSep := canvas.NewRectangle(palette.BorderSubtle)

	table := container.New(&matrixTableLayout{}, colModel, col5h, colWeek, hSep)
	return table, txtTier, cellGemini5h, cellGeminiWeek, cellClaude5h, cellClaudeWeek
}

// CreateMatrixQuotaTableWithTier builds a 3x3 matrix table for model quota and tier display:
// Row 0: <Tier> | 5h           | Week
// Row 1: Gemini | [bar] <pct>% | [bar] <pct>%
// Row 2: Claude | [bar] <pct>% | [bar] <pct>%
func CreateMatrixQuotaTableWithTier(dq DualCycleQuota, tier string, palette ThemePalette) fyne.CanvasObject {
	table, _, _, _, _, _ := createMatrixQuotaTableWithCells(dq, tier, palette)
	return table
}

// CreateMatrixQuotaTable builds a 3x3 matrix table for model quota display per architect spec:
// Row 0: <Tier> | 5h           | Week
// Row 1: Gemini | [bar] <pct>% | [bar] <pct>%
// Row 2: Claude | [bar] <pct>% | [bar] <pct>%
func CreateMatrixQuotaTable(dq DualCycleQuota, palette ThemePalette) fyne.CanvasObject {
	return CreateMatrixQuotaTableWithTier(dq, "", palette)
}
