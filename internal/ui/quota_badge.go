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

// MatrixMiniBarWidth defines the 40px width for 3x3 matrix quota mini progress bars per architect spec.
const MatrixMiniBarWidth float32 = 40.0

// matrixCellLayout lays out [miniBar, pctLabel] horizontally with vertical centering.
type matrixCellLayout struct {
	barWidth float32
}

func (l *matrixCellLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) < 2 {
		return fyne.NewSize(76.0, 15.0)
	}
	bw := l.barWidth
	if bw <= 0 {
		bw = MatrixMiniBarWidth
	}
	return fyne.NewSize(bw+4.0+objects[1].MinSize().Width, 15.0)
}

func (l *matrixCellLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 2 {
		return
	}
	bar := objects[0]
	pct := objects[1]

	bw := l.barWidth
	if bw <= 0 {
		bw = MatrixMiniBarWidth
	}
	spacing := float32(4.0)

	bar.Move(fyne.NewPos(0, (size.Height-6.0)/2))
	bar.Resize(fyne.NewSize(bw, 6.0))

	pctH := pct.MinSize().Height
	pct.Move(fyne.NewPos(bw+spacing, (size.Height-pctH)/2))
	pct.Resize(fyne.NewSize(pct.MinSize().Width, pctH))
}

type matrixQuotaCell struct {
	container *fyne.Container
	layout    *miniProgressBarLayout
	bgTrack   *canvas.Rectangle
	fillBar   *canvas.Rectangle
	pctText   *canvas.Text
}

func newMatrixQuotaCell(fraction float64, hasData bool, palette ThemePalette) *matrixQuotaCell {
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
	pct.TextSize = 11
	pct.TextStyle = fyne.TextStyle{Bold: true}

	cellContainer := container.New(&matrixCellLayout{barWidth: MatrixMiniBarWidth}, barContainer, pct)

	return &matrixQuotaCell{
		container: cellContainer,
		layout:    pbLayout,
		bgTrack:   bgTrack,
		fillBar:   fillBar,
		pctText:   pct,
	}
}

func (c *matrixQuotaCell) update(fraction float64, hasData bool) {
	if c == nil {
		return
	}
	barWidth := float32(MatrixMiniBarWidth) // 40.0 px cố định
	if barWidth <= 0 {
		barWidth = 40.0
	}

	if !hasData || fraction < 0 {
		c.pctText.Text = "--%"
		c.pctText.Color = ColorQuotaMuted
		c.layout.fraction = -1
		c.layout.fillWidth = 0
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
		c.layout.fraction = fraction

		// DUY NHẤT fillWidth CO GIÃN THEO FRACTION
		fw := barWidth * float32(clamped)
		c.layout.fillWidth = fw
		c.fillBar.FillColor = tierColor
		c.fillBar.Resize(fyne.NewSize(fw, c.layout.barHeight))
	}

	c.fillBar.Refresh()
	c.pctText.Refresh()
	// KHÔNG GỌI c.container.Refresh() để tránh kích hoạt layout cascade của Fyne
}

// createMatrixQuotaCell creates a graphical mini progress bar (40px) and percentage text label.
func createMatrixQuotaCell(fraction float64, hasData bool, palette ThemePalette) fyne.CanvasObject {
	cell := newMatrixQuotaCell(fraction, hasData, palette)
	return cell.container
}

// matrixColumnLayout lays out 3 rows in a column with fixed row heights and subtle spacing.
type matrixColumnLayout struct{}

func (l *matrixColumnLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	maxW := float32(0)
	totalH := float32(0)
	for i, o := range objects {
		ms := o.MinSize()
		if ms.Width > maxW {
			maxW = ms.Width
		}
		totalH += 15.0
		if i > 0 {
			totalH += 2.0
		}
	}
	return fyne.NewSize(maxW, totalH)
}

func (l *matrixColumnLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	rowH := float32(15.0)
	spacing := float32(2.0)
	y := float32(0)
	for _, o := range objects {
		o.Move(fyne.NewPos(0, y))
		o.Resize(fyne.NewSize(size.Width, rowH))
		y += rowH + spacing
	}
}

// matrixTableLayout provides responsive layout for the 3x3 quota matrix:
// [colModel, sep1, col5h, sep2, colWeek]
type matrixTableLayout struct{}

func (l *matrixTableLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	col1W := float32(50.0)
	sepW := float32(1.0)
	colCycleW := float32(88.0)
	colGap := float32(4.0)
	totalW := col1W + sepW + colCycleW + sepW + colCycleW + 4*colGap
	return fyne.NewSize(totalW, 49.0)
}

func (l *matrixTableLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 5 {
		return
	}
	colModel := objects[0]
	sep1 := objects[1]
	col5h := objects[2]
	sep2 := objects[3]
	colWeek := objects[4]

	h := size.Height
	col1W := float32(50.0)    // Cột Model: CỐ ĐỊNH 50px
	sepW := float32(1.0)     // Vách ngăn: CỐ ĐỊNH 1px
	colGap := float32(4.0)   // Khoảng cách: CỐ ĐỊNH 4px
	colCycleW := float32(88.0) // Cột 5h và Week: KHÓA CỨNG 88px TUYỆT ĐỐI, CẤM CO GIÃN ĐỘNG!

	x := float32(0)
	// Col 1: Model
	colModel.Move(fyne.NewPos(x, 0))
	colModel.Resize(fyne.NewSize(col1W, h))
	x += col1W + colGap

	// Sep 1
	sep1.Move(fyne.NewPos(x, 1))
	sep1.Resize(fyne.NewSize(sepW, h-2))
	x += sepW + colGap

	// Col 2: 5h
	col5h.Move(fyne.NewPos(x, 0))
	col5h.Resize(fyne.NewSize(colCycleW, h))
	x += colCycleW + colGap

	// Sep 2
	sep2.Move(fyne.NewPos(x, 1))
	sep2.Resize(fyne.NewSize(sepW, h-2))
	x += sepW + colGap

	// Col 3: Week
	colWeek.Move(fyne.NewPos(x, 0))
	colWeek.Resize(fyne.NewSize(colCycleW, h))
}

func createMatrixQuotaTableWithCells(dq DualCycleQuota, palette ThemePalette) (fyne.CanvasObject, *matrixQuotaCell, *matrixQuotaCell, *matrixQuotaCell, *matrixQuotaCell) {
	// Col 1: Model Labels
	txtQuota := canvas.NewText("Quota", palette.TextMuted)
	txtQuota.TextSize = 11
	txtQuota.TextStyle = fyne.TextStyle{Bold: true}

	txtGemini := canvas.NewText("Gemini", palette.Text)
	txtGemini.TextSize = 11
	txtGemini.TextStyle = fyne.TextStyle{Bold: true}

	txtClaude := canvas.NewText("Claude", palette.Text)
	txtClaude.TextSize = 11
	txtClaude.TextStyle = fyne.TextStyle{Bold: true}

	colModel := container.New(&matrixColumnLayout{}, txtQuota, txtGemini, txtClaude)

	// Vertical Separator 1 (1px)
	sep1 := canvas.NewRectangle(palette.BorderSubtle)

	// Col 2: 5h Cycle
	txt5h := canvas.NewText("5h", palette.TextMuted)
	txt5h.TextSize = 11
	txt5h.TextStyle = fyne.TextStyle{Bold: true}

	hasGemini5h := !dq.IsError && dq.HasData && dq.Gemini5h >= 0
	cellGemini5h := newMatrixQuotaCell(dq.Gemini5h, hasGemini5h, palette)

	hasClaude5h := !dq.IsError && dq.HasData && dq.Claude5h >= 0
	cellClaude5h := newMatrixQuotaCell(dq.Claude5h, hasClaude5h, palette)

	col5h := container.New(&matrixColumnLayout{}, txt5h, cellGemini5h.container, cellClaude5h.container)

	// Vertical Separator 2 (1px)
	sep2 := canvas.NewRectangle(palette.BorderSubtle)

	// Col 3: Week Cycle
	txtWeek := canvas.NewText("Week", palette.TextMuted)
	txtWeek.TextSize = 11
	txtWeek.TextStyle = fyne.TextStyle{Bold: true}

	hasGeminiWeek := !dq.IsError && dq.HasData && dq.GeminiWeek >= 0
	cellGeminiWeek := newMatrixQuotaCell(dq.GeminiWeek, hasGeminiWeek, palette)

	hasClaudeWeek := !dq.IsError && dq.HasData && dq.ClaudeWeek >= 0
	cellClaudeWeek := newMatrixQuotaCell(dq.ClaudeWeek, hasClaudeWeek, palette)

	colWeek := container.New(&matrixColumnLayout{}, txtWeek, cellGeminiWeek.container, cellClaudeWeek.container)

	table := container.New(&matrixTableLayout{}, colModel, sep1, col5h, sep2, colWeek)
	return table, cellGemini5h, cellGeminiWeek, cellClaude5h, cellClaudeWeek
}

// CreateMatrixQuotaTable builds a 3x3 matrix table for model quota display per architect spec:
// Row 0: Quota  | 5h           | Week
// Row 1: Gemini | [bar] <pct>% | [bar] <pct>%
// Row 2: Claude | [bar] <pct>% | [bar] <pct>%
func CreateMatrixQuotaTable(dq DualCycleQuota, palette ThemePalette) fyne.CanvasObject {
	table, _, _, _, _ := createMatrixQuotaTableWithCells(dq, palette)
	return table
}
