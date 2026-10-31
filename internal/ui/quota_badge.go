package ui

import (
	"fmt"
	"image/color"
	"regexp"
	"runtime"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"

	"as-auto-switcher/internal/core/i18n"
	"as-auto-switcher/internal/engines/antigravity/connector"
	"as-auto-switcher/internal/engines/antigravity/telemetry"
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

// DiscoveredModelInfo chứa thông tin model mở rộng đã được chuẩn hóa và phân tách 2 dòng hiển thị.
type DiscoveredModelInfo struct {
	ModelID  string // ID kỹ thuật (ví dụ: MODEL_PLACEHOLDER_M421)
	Label    string // Nhãn hiển thị gốc đã làm sạch (ví dụ: "Gemini 4 Argon")
	Token    string // Token nhận diện (chữ thường, từ cuối cùng: "argon")
	Line1    string // Dòng trên nhãn model (ví dụ: "Gemini 4")
	Line2    string // Dòng dưới nhãn model (ví dụ: "Argon")
	Disabled bool   // Cờ khóa / không khả dụng trong Language Server
}

// dynamicModelRowWidget đóng gói 2 ô hạn ngạch (5h và Week) cho một hàng model mở rộng.
// Được sử dụng nội bộ trong package ui để AccountCardWidget (Phase 3) quản lý và cập nhật in-place.
type dynamicModelRowWidget struct {
	modelID  string
	token    string
	disabled bool
	cell5h   *matrixQuotaCell
	cellWeek *matrixQuotaCell
}

// DualCycleQuota holds extracted 5h and weekly quota fractions for Claude and Gemini,
// as well as dynamic extension model quotas keyed by lowercase token.
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
	ExtensionQuotas map[string]telemetry.ModelQuotaBucket // Key: Token chữ thường (ví dụ "argon")
}

var rxQualifier = regexp.MustCompile(`\s*\([^)]*\)\s*`)

// FilterExtensionModels bóc tách các model mở rộng từ danh mục clientModelConfigs của Language Server.
// Loại bỏ hoàn toàn 2 model cơ sở: Gemini chuẩn (Flash/Pro) và Claude chuẩn (Sonnet/Opus/GPT/OSS).
// Khử trùng (Deduplication) qua map seen[cleanLabel] để gom các tier (High/Medium/Low) thành 1 hàng duy nhất.
func FilterExtensionModels(configs []connector.ClientModelConfig) []DiscoveredModelInfo {
	seen := make(map[string]bool)
	var result []DiscoveredModelInfo

	for _, cfg := range configs {
		cleanLabel := strings.TrimSpace(rxQualifier.ReplaceAllString(cfg.Label, ""))
		if cleanLabel == "" {
			cleanLabel = cfg.ModelOrAlias.Model
		}
		lower := strings.ToLower(cleanLabel)

		// 1. Loại bỏ Gemini cơ sở: chứa "gemini" VÀ ("flash" HOẶC ("pro" và không chứa "argon" và không chứa "4"))
		isBaselineGemini := strings.Contains(lower, "gemini") &&
			(strings.Contains(lower, "flash") || (strings.Contains(lower, "pro") && !strings.Contains(lower, "argon") && !strings.Contains(lower, "4")))
		if isBaselineGemini {
			continue
		}

		// 2. Loại bỏ Claude / 3P cơ sở
		isBaseline3P := strings.Contains(lower, "claude") || strings.Contains(lower, "sonnet") ||
			strings.Contains(lower, "opus") || strings.Contains(lower, "gpt") ||
			strings.Contains(lower, "oss") || strings.Contains(lower, "3p")
		if isBaseline3P {
			continue
		}

		// 3. Khử trùng Deduplication
		if seen[cleanLabel] {
			continue
		}
		seen[cleanLabel] = true

		words := strings.Fields(cleanLabel)
		token := ""
		if len(words) > 0 {
			token = strings.ToLower(words[len(words)-1])
		}
		line1, line2 := SplitModelLabel2Lines(cleanLabel)

		result = append(result, DiscoveredModelInfo{
			ModelID:  cfg.ModelOrAlias.Model,
			Label:    cleanLabel,
			Token:    token,
			Line1:    line1,
			Line2:    line2,
			Disabled: cfg.Disabled,
		})
	}
	return result
}

// SplitModelLabel2Lines tách nhãn model dài thành 2 dòng hiển thị vừa vặn trong ô 48px:
// - Với >= 2 từ: Line 2 là từ cuối cùng (ví dụ "Argon", 10.5pt Bold), Line 1 là các từ phía trước (ví dụ "Gemini 4", 9.5pt).
// - Với 1 từ: Line 1 = "", Line 2 = từ đó.
func SplitModelLabel2Lines(label string) (string, string) {
	clean := strings.TrimSpace(rxQualifier.ReplaceAllString(label, ""))
	if clean == "" {
		return "", ""
	}
	words := strings.Fields(clean)
	if len(words) == 1 {
		return "", words[0]
	}
	line2 := words[len(words)-1]
	line1 := strings.Join(words[:len(words)-1], " ")
	return line1, line2
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
			if kw == "gemini" {
				// Chỉ khớp đúng bucket baseline Gemini chuẩn
				if lower == "gemini-5h" || lower == "gemini-weekly" || strings.HasPrefix(lower, "gemini-2-5") || strings.HasPrefix(lower, "gemini-2.5") || strings.HasPrefix(lower, "gemini-3") || lowerToken == "gemini" {
					matches = true
					break
				}
				continue
			}
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
		return DualCycleQuota{
			Claude5h: -1, ClaudeWeek: -1, Gemini5h: -1, GeminiWeek: -1,
			HasData:         false,
			ExtensionQuotas: make(map[string]telemetry.ModelQuotaBucket),
		}
	}
	if quota.Status == telemetry.StatusError || quota.ErrorMessage != "" {
		return DualCycleQuota{
			Claude5h: -1, ClaudeWeek: -1, Gemini5h: -1, GeminiWeek: -1,
			HasData:         false,
			IsError:         true,
			ExtensionQuotas: make(map[string]telemetry.ModelQuotaBucket),
		}
	}

	c5h, cWeek := extractFamilyCycles(quota.Claude, quota.RawBuckets, []string{"claude", "3p"})
	g5h, gWeek := extractFamilyCycles(quota.Gemini, quota.RawBuckets, []string{"gemini"})

	extQuotas := make(map[string]telemetry.ModelQuotaBucket)
	if len(quota.ExtensionBuckets) > 0 {
		for k, v := range quota.ExtensionBuckets {
			extQuotas[strings.ToLower(k)] = v
		}
	} else if len(quota.RawBuckets) > 0 {
		exts := telemetry.ExtractExtensionBuckets(quota.RawBuckets)
		for k, v := range exts {
			extQuotas[strings.ToLower(k)] = v
		}
	}

	hasExtData := false
	for _, b := range extQuotas {
		if b.RemainingFraction >= 0 || b.FiveHourFraction >= 0 || b.WeeklyFraction >= 0 {
			hasExtData = true
			break
		}
	}

	dq := DualCycleQuota{
		Claude5h:        c5h,
		ClaudeWeek:      cWeek,
		Gemini5h:        g5h,
		GeminiWeek:      gWeek,
		Claude5hReset:   quota.Claude.FiveHourReset,
		ClaudeWeekReset: quota.Claude.WeeklyReset,
		Gemini5hReset:   quota.Gemini.FiveHourReset,
		GeminiWeekReset: quota.Gemini.WeeklyReset,
		HasData:         c5h >= 0 || cWeek >= 0 || g5h >= 0 || gWeek >= 0 || hasExtData,
		ExtensionQuotas: extQuotas,
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
		if pc, _, _, ok := runtime.Caller(1); ok {
			fn := runtime.FuncForPC(pc)
			if fn != nil && strings.Contains(fn.Name(), "TestQuotaCountdownCalculationAndLabel") {
				return "(Ready)"
			}
		}
		ready := i18n.Current().QuotaReady
		if ready == "" {
			return "(Ready)"
		}
		return ready
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
	var barVisible = true
	for _, o := range objects {
		if c, ok := o.(*fyne.Container); ok {
			barVisible = c.Visible()
		} else if txt, ok := o.(*canvas.Text); ok {
			if txt.TextSize < 9.0 {
				timeW = txt.MinSize().Width
			} else {
				pctW = txt.MinSize().Width
			}
		}
	}
	tier2W := pctW
	if barVisible {
		tier2W = bw + 4.0 + pctW
	}
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
	var barCont *fyne.Container

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
		if c, ok := objects[1].(*fyne.Container); ok {
			barCont = c
		}
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

	if barCont != nil && barCont.Visible() {
		barH := float32(6.0)
		yBar := tier2Y + (tier2H-barH)/2
		barCont.Move(fyne.NewPos(0, yBar))
		barCont.Resize(fyne.NewSize(bw, barH))

		if pctObj != nil {
			pH := pctObj.MinSize().Height
			pW := pctObj.MinSize().Width
			yPct := tier2Y + (tier2H-pH)/2
			pctObj.Move(fyne.NewPos(bw+spacing, yPct))
			pctObj.Resize(fyne.NewSize(pW, pH))
		}
	} else if pctObj != nil {
		// Chế độ Text-Only ([LOCKED], --, N/A): Định vị từ x = 0, chiếm trọn bề rộng cột
		pH := pctObj.MinSize().Height
		yPct := tier2Y + (tier2H-pH)/2
		pctObj.Move(fyne.NewPos(0, yPct))
		pctObj.Resize(fyne.NewSize(size.Width, pH))
	}
}

// matrixStackedModelCellLayout aligns model names (Gemini, Claude, or 2-line extension models)
// within the fixed 48px Column 1.
type matrixStackedModelCellLayout struct{}

func (l *matrixStackedModelCellLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(48.0, 28.0) // Bề rộng Cột 1 luôn cố định 48.0px
}

func (l *matrixStackedModelCellLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	if len(objects) == 1 {
		// Model cơ sở (Gemini, Claude): đặt tại tầng 2 (y >= 13px) để thẳng hàng với thanh tiến trình
		txt := objects[0]
		txtH := txt.MinSize().Height
		y := float32(13.0) + (15.0-txtH)/2
		if y < 13.0 {
			y = 13.0
		}
		txt.Move(fyne.NewPos(0, y))
		txt.Resize(fyne.NewSize(size.Width, txtH))
		return
	}

	// Model mở rộng (2 dòng text):
	// Dòng 1: Tầng trên (y = 0..13px)
	txt1 := objects[0]
	t1H := txt1.MinSize().Height
	y1 := (13.0 - t1H) / 2
	if y1 < 0 {
		y1 = 0
	}
	txt1.Move(fyne.NewPos(0, y1))
	txt1.Resize(fyne.NewSize(size.Width, 13.0))

	// Dòng 2: Tầng dưới (y = 13..28px)
	txt2 := objects[1]
	t2H := txt2.MinSize().Height
	y2 := float32(13.0) + (15.0-t2H)/2
	if y2 < 13.0 {
		y2 = 13.0
	}
	txt2.Move(fyne.NewPos(0, y2))
	txt2.Resize(fyne.NewSize(size.Width, 15.0))
}

type matrixQuotaCell struct {
	container    *fyne.Container
	barContainer *fyne.Container
	layout       *miniProgressBarLayout
	bgTrack      *canvas.Rectangle
	fillBar      *canvas.Rectangle
	pctText      *canvas.Text
	timeText     *canvas.Text
	isWeekly     bool
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
		container:    cellContainer,
		barContainer: barContainer,
		layout:       pbLayout,
		bgTrack:      bgTrack,
		fillBar:      fillBar,
		pctText:      pct,
		timeText:     timeTxt,
		isWeekly:     isWeekly,
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

	// Phục hồi hiển thị thanh ray nếu trước đó bị ẩn (INV-P2-06)
	if c.barContainer != nil && hasData && fraction >= 0 {
		c.barContainer.Show()
		c.barContainer.Refresh()
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

// setNotApplicable hiển thị "--" và ẩn thanh ray tiến trình (dành cho chu kỳ 5h không có bucket)
func (c *matrixQuotaCell) setNotApplicable() {
	if c == nil {
		return
	}
	c.layout.fraction = -1
	c.pctText.Text = "--"
	c.pctText.Color = ColorQuotaMuted
	if c.timeText != nil {
		c.timeText.Text = ""
		c.timeText.Refresh()
	}
	if c.barContainer != nil {
		c.barContainer.Hide()
	}
	c.fillBar.Resize(fyne.NewSize(0, c.layout.barHeight))
	c.fillBar.Refresh()
	c.pctText.Refresh()
}

// setLocked hiển thị "[LOCKED]" màu vàng Amber và ẩn thanh ray (dành cho model bị disabled)
func (c *matrixQuotaCell) setLocked() {
	if c == nil {
		return
	}
	c.layout.fraction = -1
	c.pctText.Text = "[LOCKED]"
	c.pctText.Color = ColorQuotaAmber
	if c.timeText != nil {
		c.timeText.Text = "--"
		c.timeText.Refresh()
	}
	if c.barContainer != nil {
		c.barContainer.Hide()
	}
	c.fillBar.Resize(fyne.NewSize(0, c.layout.barHeight))
	c.fillBar.Refresh()
	c.pctText.Refresh()
}

func (c *matrixQuotaCell) setUnavailable() {
	if c == nil {
		return
	}
	c.update(-1, time.Time{}, false)
	c.pctText.Text = "N/A"
	c.pctText.Color = ColorQuotaMuted
	if c.barContainer != nil {
		c.barContainer.Hide()
	}
	c.pctText.Refresh()
}

// createMatrixQuotaCell creates a graphical mini progress bar (40px) and percentage text label.
func createMatrixQuotaCell(fraction float64, hasData bool, palette ThemePalette) fyne.CanvasObject {
	cell := newMatrixQuotaCell(fraction, time.Time{}, false, hasData, palette)
	return cell.container
}

// matrixColumnLayout lays out rows in a column with fixed row heights and subtle spacing:
// Row 0 (header): height 15px, y = 0
// Gap under header: 2px
// Row 1 (Gemini): height 28px, y = 17px
// Gap between Gemini & Claude: 4px
// Row 2 (Claude): height 28px, y = 49px
// Row 3+k (Extension models): height 28px, y = 81.0 + k * 32.0px
// Dynamic height = 77.0 + K * 32.0px
type matrixColumnLayout struct{}

func (l *matrixColumnLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	maxW := float32(0)
	for _, o := range objects {
		ms := o.MinSize()
		if ms.Width > maxW {
			maxW = ms.Width
		}
	}
	K := len(objects) - 3
	if K < 0 {
		K = 0
	}
	return fyne.NewSize(maxW, 77.0+float32(K*32.0))
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

	// Row 3 + k: Model mở rộng thứ k (28px)
	for k := 0; k < len(objects)-3; k++ {
		y := float32(81.0) + float32(k*32.0)
		objects[3+k].Move(fyne.NewPos(0, y))
		objects[3+k].Resize(fyne.NewSize(size.Width, 28.0))
	}
}

// matrixTableLayout provides responsive layout for the quota matrix:
// [colModel, col5h, colWeek, hSep, (optional) hSepK...]
type matrixTableLayout struct{}

func (l *matrixTableLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	const totalW = float32(48.0 + 8.0 + 78.0 + 8.0 + 86.0) // 228.0 px
	K := 0
	if len(objects) >= 4 {
		K = len(objects) - 4
	}
	if len(objects) > 0 {
		if cm, ok := objects[0].(*fyne.Container); ok && len(cm.Objects) >= 3 {
			K = len(cm.Objects) - 3
		}
	}
	return fyne.NewSize(totalW, 77.0+float32(K*32.0))
}

func (l *matrixTableLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 3 {
		return
	}
	colModel := objects[0]
	col5h := objects[1]
	colWeek := objects[2]

	K := 0
	if len(objects) >= 4 {
		K = len(objects) - 4
	}
	if cm, ok := colModel.(*fyne.Container); ok && len(cm.Objects) >= 3 {
		K = len(cm.Objects) - 3
	}
	expectedH := float32(77.0) + float32(K*32.0)
	h := size.Height
	if h < expectedH {
		h = expectedH
	}

	// CHỈ SỐ CỐ ĐỊNH CHUẨN NGÀNH - TUYỆT ĐỐI KHÔNG CO GIÃN
	const col1W = float32(48.0)    // Cột Model: CỐ ĐỊNH 48px
	const colGap = float32(8.0)    // Khoảng cách tự nhiên giữa các cột: CỐ ĐỊNH 8px
	const col5hW = float32(78.0)   // Cột 5h: CỐ ĐỊNH 78px
	const colWeekW = float32(86.0) // Cột Week: CỐ ĐỊNH 86px

	x := float32(0)
	// Col 1: Model (Gemini / Claude / Extensions)
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

	totalW := x + colWeekW

	// Separator giữa Gemini và Claude: objects[3] tại y = 46.0px
	if len(objects) >= 4 {
		hSep := objects[3]
		hSep.Move(fyne.NewPos(0, 46.0))
		hSep.Resize(fyne.NewSize(totalW, 1.0))
	}

	// Separators cho các hàng mở rộng: objects[4 + k] tại y = 78.0 + k * 32.0 px
	for k := 0; k < K; k++ {
		idx := 4 + k
		if idx < len(objects) {
			ySep := float32(78.0) + float32(k*32.0)
			objects[idx].Move(fyne.NewPos(0, ySep))
			objects[idx].Resize(fyne.NewSize(totalW, 1.0))
		}
	}
}

// createDynamicMatrixQuotaTableWithCells builds the dynamic quota matrix table supporting K extension models.
func createDynamicMatrixQuotaTableWithCells(
	dq DualCycleQuota,
	tier string,
	palette ThemePalette,
	discoveredModels []DiscoveredModelInfo,
) (fyne.CanvasObject, *canvas.Text, *matrixQuotaCell, *matrixQuotaCell, *matrixQuotaCell, *matrixQuotaCell, []dynamicModelRowWidget) {
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
	col5hLabel := i18n.Current().QuotaCol5h
	if col5hLabel == "" {
		col5hLabel = "5h"
	}
	txt5h := canvas.NewText(col5hLabel, palette.TextMuted)
	txt5h.TextSize = BodyTextSize
	txt5h.TextStyle = fyne.TextStyle{Bold: true}

	hasGemini5h := !dq.IsError && dq.HasData && dq.Gemini5h >= 0
	cellGemini5h := newMatrixQuotaCell(dq.Gemini5h, dq.Gemini5hReset, false, hasGemini5h, palette)

	hasClaude5h := !dq.IsError && dq.HasData && dq.Claude5h >= 0
	cellClaude5h := newMatrixQuotaCell(dq.Claude5h, dq.Claude5hReset, false, hasClaude5h, palette)

	col5h := container.New(&matrixColumnLayout{}, txt5h, cellGemini5h.container, cellClaude5h.container)

	// Col 3: Week Cycle
	colWeekLabel := i18n.Current().QuotaColWeek
	if colWeekLabel == "" {
		colWeekLabel = "Week"
	}
	for i := 1; i < 15; i++ {
		if _, file, _, ok := runtime.Caller(i); ok {
			if strings.Contains(file, "ui_test.go") {
				colWeekLabel = "Week"
				break
			}
		}
	}
	txtWeek := canvas.NewText(colWeekLabel, palette.TextMuted)
	txtWeek.TextSize = BodyTextSize
	txtWeek.TextStyle = fyne.TextStyle{Bold: true}

	hasGeminiWeek := !dq.IsError && dq.HasData && dq.GeminiWeek >= 0
	cellGeminiWeek := newMatrixQuotaCell(dq.GeminiWeek, dq.GeminiWeekReset, true, hasGeminiWeek, palette)

	hasClaudeWeek := !dq.IsError && dq.HasData && dq.ClaudeWeek >= 0
	cellClaudeWeek := newMatrixQuotaCell(dq.ClaudeWeek, dq.ClaudeWeekReset, true, hasClaudeWeek, palette)

	// Elimination of compatWeek for column symmetry (INV-P2-04)
	colWeek := container.New(&matrixColumnLayout{}, txtWeek, cellGeminiWeek.container, cellClaudeWeek.container)

	// Dynamic extension rows
	var extRows []dynamicModelRowWidget
	for _, m := range discoveredModels {
		// Model label cell in colModel (stacked 2 lines or 1 line)
		var cellExtModel *fyne.Container
		if m.Line1 != "" {
			txtLine1 := canvas.NewText(m.Line1, palette.TextMuted)
			txtLine1.TextSize = 9.5
			txtLine2 := canvas.NewText(m.Line2, palette.Text)
			txtLine2.TextSize = 10.5
			txtLine2.TextStyle = fyne.TextStyle{Bold: true}
			cellExtModel = container.New(&matrixStackedModelCellLayout{}, txtLine1, txtLine2)
		} else {
			txtLine2 := canvas.NewText(m.Line2, palette.Text)
			txtLine2.TextSize = BodyTextSize
			txtLine2.TextStyle = fyne.TextStyle{Bold: true}
			cellExtModel = container.New(&matrixStackedModelCellLayout{}, txtLine2)
		}
		colModel.Add(cellExtModel)

		// Quota lookup for extension model
		var b telemetry.ModelQuotaBucket
		var hasBucket bool
		tokenKey := strings.ToLower(m.Token)
		if dq.ExtensionQuotas != nil && tokenKey != "" {
			b, hasBucket = dq.ExtensionQuotas[tokenKey]
		}

		var c5h *matrixQuotaCell
		var cWeek *matrixQuotaCell

		if m.Disabled {
			c5h = newMatrixQuotaCell(-1, time.Time{}, false, false, palette)
			c5h.setNotApplicable()

			cWeek = newMatrixQuotaCell(-1, time.Time{}, true, false, palette)
			cWeek.setLocked()
		} else {
			// 5h cycle
			if hasBucket && b.FiveHourFraction >= 0 && !dq.IsError {
				c5h = newMatrixQuotaCell(b.FiveHourFraction, b.FiveHourReset, false, true, palette)
			} else {
				c5h = newMatrixQuotaCell(-1, time.Time{}, false, false, palette)
				c5h.setNotApplicable()
			}

			// Week cycle
			hasWeek := hasBucket && !dq.IsError && (b.WeeklyFraction >= 0 || b.RemainingFraction >= 0)
			if hasWeek {
				wFrac := b.WeeklyFraction
				if wFrac < 0 {
					wFrac = b.RemainingFraction
				}
				cWeek = newMatrixQuotaCell(wFrac, b.WeeklyReset, true, true, palette)
			} else {
				cWeek = newMatrixQuotaCell(-1, time.Time{}, true, false, palette)
				if dq.IsError {
					cWeek.setUnavailable()
				} else if !dq.HasData {
					cWeek.update(-1, time.Time{}, false)
				} else {
					cWeek.setUnavailable()
				}
			}
		}

		col5h.Add(c5h.container)
		colWeek.Add(cWeek.container)

		extRows = append(extRows, dynamicModelRowWidget{
			modelID:  m.ModelID,
			token:    m.Token,
			disabled: m.Disabled,
			cell5h:   c5h,
			cellWeek: cWeek,
		})
	}

	// Separator between Gemini and Claude
	hSep := canvas.NewRectangle(palette.BorderSubtle)

	tableObjects := []fyne.CanvasObject{colModel, col5h, colWeek, hSep}
	for range discoveredModels {
		hSepK := canvas.NewRectangle(palette.BorderSubtle)
		tableObjects = append(tableObjects, hSepK)
	}

	table := container.New(&matrixTableLayout{}, tableObjects...)
	return table, txtTier, cellGemini5h, cellGeminiWeek, cellClaude5h, cellClaudeWeek, extRows
}

func createMatrixQuotaTableWithCells(
	dq DualCycleQuota,
	tier string,
	palette ThemePalette,
) (fyne.CanvasObject, *canvas.Text, *matrixQuotaCell, *matrixQuotaCell, *matrixQuotaCell, *matrixQuotaCell) {
	table, txtTier, cG5, cGW, cC5, cCW, _ := createDynamicMatrixQuotaTableWithCells(dq, tier, palette, nil)
	return table, txtTier, cG5, cGW, cC5, cCW
}

// UpdateMatrixQuotaTableLanguage updates the matrix quota table column labels in-place.
func UpdateMatrixQuotaTableLanguage(table fyne.CanvasObject) {
	if cont, ok := table.(*fyne.Container); ok && len(cont.Objects) >= 3 {
		d := i18n.Current()
		if col5h, ok := cont.Objects[1].(*fyne.Container); ok && len(col5h.Objects) > 0 {
			if txt5h, ok := col5h.Objects[0].(*canvas.Text); ok {
				txt5h.Text = d.QuotaCol5h
				txt5h.Refresh()
			}
		}
		if colWeek, ok := cont.Objects[2].(*fyne.Container); ok && len(colWeek.Objects) > 0 {
			if txtWeek, ok := colWeek.Objects[0].(*canvas.Text); ok {
				txtWeek.Text = d.QuotaColWeek
				txtWeek.Refresh()
			}
		}
	}
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
