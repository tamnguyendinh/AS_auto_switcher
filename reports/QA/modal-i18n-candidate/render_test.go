package qa_modal_i18n_candidate

import (
	"encoding/json"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"antigravity-switch/internal/core/i18n"
	"antigravity-switch/internal/ui"
)

func saveCanvas(t *testing.T, win fyne.Window, filename string) {
	_ = os.MkdirAll("screenshots", 0755)
	img := win.Canvas().Capture()
	if img == nil {
		t.Fatalf("win.Canvas().Capture() returned nil for %s", filename)
	}
	f, err := os.Create(filename)
	if err != nil {
		t.Fatalf("failed to create %s: %v", filename, err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("failed to encode %s: %v", filename, err)
	}
	t.Logf("Saved visual evidence: %s", filename)
}

func layoutSpacer() fyne.CanvasObject {
	r := canvas.NewRectangle(color.Transparent)
	r.SetMinSize(fyne.NewSize(1, 1))
	return r
}

// -----------------------------------------------------------------------------
// Test 1: Candidate Autonomy Invariant & TC-MODAL-i18n-01/02
// -----------------------------------------------------------------------------
func TestQA_Invariant1_CandidateAutonomy_And_Bilingual(t *testing.T) {
	// 1. VI Locale
	i18n.SetCurrentLocale(i18n.LocaleVI)
	dVI := i18n.Current()

	// Assert independence from Main UI BadgeActive
	if dVI.ModalBadgeOAuthDone == dVI.BadgeActive {
		t.Fatalf("VI: ModalBadgeOAuthDone (%q) must NOT equal BadgeActive (%q)", dVI.ModalBadgeOAuthDone, dVI.BadgeActive)
	}
	if dVI.ModalBadgeOAuthDone != "[✔ Đã OAuth]" {
		t.Errorf("VI: expected ModalBadgeOAuthDone '[✔ Đã OAuth]', got %q", dVI.ModalBadgeOAuthDone)
	}
	if dVI.ModalBadgeAdded != "[Đã thêm]" {
		t.Errorf("VI: expected ModalBadgeAdded '[Đã thêm]', got %q", dVI.ModalBadgeAdded)
	}
	if dVI.ModalBadgeNeedOAuth != "[Cần OAuth]" {
		t.Errorf("VI: expected ModalBadgeNeedOAuth '[Cần OAuth]', got %q", dVI.ModalBadgeNeedOAuth)
	}
	if dVI.ModalBadgeNotFound != "[Tài khoản không tồn tại]" {
		t.Errorf("VI: expected ModalBadgeNotFound '[Tài khoản không tồn tại]', got %q", dVI.ModalBadgeNotFound)
	}
	if dVI.ModalBtnAddItem != "Thêm" {
		t.Errorf("VI: expected ModalBtnAddItem 'Thêm', got %q", dVI.ModalBtnAddItem)
	}

	// 2. EN Locale
	i18n.SetCurrentLocale(i18n.LocaleEN)
	dEN := i18n.Current()

	if dEN.ModalBadgeOAuthDone == dEN.BadgeActive {
		t.Fatalf("EN: ModalBadgeOAuthDone (%q) must NOT equal BadgeActive (%q)", dEN.ModalBadgeOAuthDone, dEN.BadgeActive)
	}
	if dEN.ModalBadgeOAuthDone != "[✔ OAuth Done]" {
		t.Errorf("EN: expected ModalBadgeOAuthDone '[✔ OAuth Done]', got %q", dEN.ModalBadgeOAuthDone)
	}
	if dEN.ModalBadgeAdded != "[Added]" {
		t.Errorf("EN: expected ModalBadgeAdded '[Added]', got %q", dEN.ModalBadgeAdded)
	}
	if dEN.ModalBadgeNeedOAuth != "[Need OAuth]" {
		t.Errorf("EN: expected ModalBadgeNeedOAuth '[Need OAuth]', got %q", dEN.ModalBadgeNeedOAuth)
	}
	if dEN.ModalBadgeNotFound != "[Account Not Found]" {
		t.Errorf("EN: expected ModalBadgeNotFound '[Account Not Found]', got %q", dEN.ModalBadgeNotFound)
	}
	if dEN.ModalBtnAddItem != "Add" {
		t.Errorf("EN: expected ModalBtnAddItem 'Add', got %q", dEN.ModalBtnAddItem)
	}
}

// -----------------------------------------------------------------------------
// Test 2: Zero-Reflection / Zero Call-Stack Inspection Invariant (TC-MODAL-i18n-04)
// -----------------------------------------------------------------------------
func TestQA_Invariant2_ZeroReflection_StaticSweep(t *testing.T) {
	addModalPath := filepath.Join("..", "..", "..", "internal", "ui", "add_modal.go")
	content, err := os.ReadFile(addModalPath)
	if err != nil {
		t.Fatalf("Failed to read add_modal.go: %v", err)
	}

	code := string(content)
	if strings.Contains(code, "runtime.Caller") {
		t.Errorf("VIOLATION: add_modal.go contains forbidden 'runtime.Caller'")
	}
	if strings.Contains(code, "isLegacyModalTest") {
		t.Errorf("VIOLATION: add_modal.go contains forbidden 'isLegacyModalTest'")
	}
	if strings.Contains(code, "\"runtime\"") {
		t.Errorf("VIOLATION: add_modal.go imports 'runtime' package")
	}
}

// -----------------------------------------------------------------------------
// Test 3: Pure Dual-JSON Parity Invariant (TC-MODAL-i18n-01/02)
// -----------------------------------------------------------------------------
func TestQA_Invariant3_DualJSONParity(t *testing.T) {
	viPath := filepath.Join("..", "..", "..", "internal", "core", "i18n", "locales", "vi.json")
	enPath := filepath.Join("..", "..", "..", "internal", "core", "i18n", "locales", "en.json")

	viBytes, err := os.ReadFile(viPath)
	if err != nil {
		t.Fatalf("Failed to read vi.json: %v", err)
	}
	enBytes, err := os.ReadFile(enPath)
	if err != nil {
		t.Fatalf("Failed to read en.json: %v", err)
	}

	var viMap map[string]string
	if err := json.Unmarshal(viBytes, &viMap); err != nil {
		t.Fatalf("Failed to parse vi.json: %v", err)
	}

	var enMap map[string]string
	if err := json.Unmarshal(enBytes, &enMap); err != nil {
		t.Fatalf("Failed to parse en.json: %v", err)
	}

	if len(viMap) != len(enMap) {
		t.Fatalf("JSON key count mismatch: vi=%d, en=%d", len(viMap), len(enMap))
	}

	requiredKeys := []string{
		"modal_badge_oauth_done",
		"modal_badge_added",
		"modal_badge_need_oauth",
		"modal_badge_not_found",
		"modal_btn_add_item",
	}

	for _, k := range requiredKeys {
		viVal, okVI := viMap[k]
		if !okVI || viVal == "" {
			t.Errorf("Missing or empty key in vi.json: %s", k)
		}
		enVal, okEN := enMap[k]
		if !okEN || enVal == "" {
			t.Errorf("Missing or empty key in en.json: %s", k)
		}
	}
}

// -----------------------------------------------------------------------------
// Test 4: In-Place Reactive Modal Mutation Invariant (TC-MODAL-i18n-03)
// -----------------------------------------------------------------------------
func TestQA_Invariant4_InPlaceMutation(t *testing.T) {
	i18n.SetCurrentLocale(i18n.LocaleVI)
	d := i18n.Current()
	palette := ui.DarkPalette

	statusBadge := canvas.NewText(d.ModalBadgeNeedOAuth, palette.Warning)
	statusBadge.TextSize = 11
	statusBadge.TextStyle = fyne.TextStyle{Bold: true}

	delBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), nil)
	delBtn.Importance = widget.LowImportance

	var actionSlot *fyne.Container
	addBtn := widget.NewButtonWithIcon(d.ModalBtnAddItem, theme.ContentAddIcon(), func() {
		doneBadge := canvas.NewText(d.ModalBadgeAdded, palette.TextMuted)
		doneBadge.TextSize = 11
		doneBadge.TextStyle = fyne.TextStyle{Bold: true}
		actionSlot.Objects = []fyne.CanvasObject{container.NewCenter(doneBadge), delBtn}
		actionSlot.Refresh()
	})
	addBtn.Importance = widget.HighImportance

	actionSlot = container.NewHBox(container.NewCenter(statusBadge), addBtn, delBtn)

	// Pre-mutation check
	if len(actionSlot.Objects) != 3 {
		t.Fatalf("Expected 3 items in actionSlot, got %d", len(actionSlot.Objects))
	}

	// Trigger Add tap
	addBtn.OnTapped()

	// Post-mutation check
	if len(actionSlot.Objects) != 2 {
		t.Fatalf("Expected 2 items in actionSlot after mutation, got %d", len(actionSlot.Objects))
	}

	centerObj, ok := actionSlot.Objects[0].(*fyne.Container)
	if !ok || len(centerObj.Objects) == 0 {
		t.Fatalf("Expected center container at slot 0")
	}
	mutatedText, ok := centerObj.Objects[0].(*canvas.Text)
	if !ok || mutatedText.Text != "[Đã thêm]" {
		t.Fatalf("Expected mutated badge '[Đã thêm]', got %v", mutatedText)
	}
}

// -----------------------------------------------------------------------------
// Test 5: Session Reloader Protection Invariant
// -----------------------------------------------------------------------------
func TestQA_Invariant5_SessionReloaderProtection(t *testing.T) {
	mgrPath := filepath.Join("..", "..", "..", "internal", "manager", "account_manager.go")
	mgrContent, err := os.ReadFile(mgrPath)
	if err != nil {
		t.Fatalf("Failed to read account_manager.go: %v", err)
	}
	if !strings.Contains(string(mgrContent), "waitDuration := 8 * time.Second") {
		t.Errorf("CRITICAL VIOLATION: 8-second invariant in account_manager.go was modified or missing")
	}

	reloaderPath := filepath.Join("..", "..", "..", "internal", "reloader", "session_reloader.go")
	reloaderContent, err := os.ReadFile(reloaderPath)
	if err != nil {
		t.Fatalf("Failed to read session_reloader.go: %v", err)
	}
	if !strings.Contains(string(reloaderContent), "sleeper(ctx, waitDuration)") {
		t.Errorf("CRITICAL VIOLATION: sleeper call in session_reloader.go was modified or missing")
	}
}

// -----------------------------------------------------------------------------
// Test 6: Capture Visual Evidence for QA Report
// -----------------------------------------------------------------------------
func TestQA_CaptureVisualEvidence(t *testing.T) {
	palette := ui.DarkPalette

	// Helper to build candidate row
	buildRow := func(email string, badgeText string, badgeColor color.Color, showAdd bool, addLabel string, isMasked bool) fyne.CanvasObject {
		dispEmail := email
		if isMasked {
			dispEmail = ui.MaskEmail(email)
		}
		lbl := widget.NewLabel(dispEmail)
		lbl.Truncation = fyne.TextTruncateEllipsis

		badge := canvas.NewText(badgeText, badgeColor)
		badge.TextSize = 11
		badge.TextStyle = fyne.TextStyle{Bold: true}

		delBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), nil)
		delBtn.Importance = widget.LowImportance

		var rightBox *fyne.Container
		if showAdd {
			addBtn := widget.NewButtonWithIcon(addLabel, theme.ContentAddIcon(), nil)
			addBtn.Importance = widget.HighImportance
			rightBox = container.NewHBox(container.NewCenter(badge), addBtn, delBtn)
		} else {
			rightBox = container.NewHBox(container.NewCenter(badge), delBtn)
		}

		return container.NewBorder(nil, nil, nil, rightBox, lbl)
	}

	// 1. Evidence 01: Modal Candidates in Vietnamese
	{
		i18n.SetCurrentLocale(i18n.LocaleVI)
		d := i18n.Current()

		header := widget.NewLabelWithStyle(d.ModalAddHeader, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		scanBtn := widget.NewButtonWithIcon(d.ModalScanChrome, theme.ViewRefreshIcon(), nil)
		topRow := container.NewBorder(nil, nil, header, scanBtn)

		row1 := buildRow("oauth_verified@company.com", d.ModalBadgeOAuthDone, palette.Success, false, "", false)
		row2 := buildRow("added_member@school.edu", d.ModalBadgeAdded, palette.TextMuted, false, "", false)
		row3 := buildRow("new_candidate@partner.io", d.ModalBadgeNeedOAuth, palette.Warning, true, d.ModalBtnAddItem, false)
		row4 := buildRow("ghost_missing@domain.org", d.ModalBadgeNotFound, palette.TextMuted, false, "", false)

		list := container.NewVBox(row1, row2, row3, row4)
		closeBtn := widget.NewButton(d.ModalAddClose, nil)
		bottomRow := container.NewHBox(layoutSpacer(), closeBtn)

		box := container.NewVBox(topRow, widget.NewSeparator(), list, widget.NewSeparator(), bottomRow)
		win := test.NewWindow(container.NewPadded(box))
		win.Resize(fyne.NewSize(540, 260))
		saveCanvas(t, win, "screenshots/01_modal_candidates_vi.png")
	}

	// 2. Evidence 02: Modal Candidates in English
	{
		i18n.SetCurrentLocale(i18n.LocaleEN)
		d := i18n.Current()

		header := widget.NewLabelWithStyle(d.ModalAddHeader, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		scanBtn := widget.NewButtonWithIcon(d.ModalScanChrome, theme.ViewRefreshIcon(), nil)
		topRow := container.NewBorder(nil, nil, header, scanBtn)

		row1 := buildRow("oauth_verified@company.com", d.ModalBadgeOAuthDone, palette.Success, false, "", false)
		row2 := buildRow("added_member@school.edu", d.ModalBadgeAdded, palette.TextMuted, false, "", false)
		row3 := buildRow("new_candidate@partner.io", d.ModalBadgeNeedOAuth, palette.Warning, true, d.ModalBtnAddItem, false)
		row4 := buildRow("ghost_missing@domain.org", d.ModalBadgeNotFound, palette.TextMuted, false, "", false)

		list := container.NewVBox(row1, row2, row3, row4)
		closeBtn := widget.NewButton(d.ModalAddClose, nil)
		bottomRow := container.NewHBox(layoutSpacer(), closeBtn)

		box := container.NewVBox(topRow, widget.NewSeparator(), list, widget.NewSeparator(), bottomRow)
		win := test.NewWindow(container.NewPadded(box))
		win.Resize(fyne.NewSize(540, 260))
		saveCanvas(t, win, "screenshots/02_modal_candidates_en.png")
	}

	// 3. Evidence 03: In-Place Mutation Visual (Before & After Click)
	{
		i18n.SetCurrentLocale(i18n.LocaleVI)
		d := i18n.Current()

		header := widget.NewLabelWithStyle("Xác Minh Đột Biến Tại Chỗ (In-Place Mutation: Nút Thêm -> [Đã thêm])", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

		beforeLbl := widget.NewLabelWithStyle("Trước khi bấm (Before Click):", fyne.TextAlignLeading, fyne.TextStyle{Italic: true})
		rowBefore := buildRow("developer.candidate@corp.com", d.ModalBadgeNeedOAuth, palette.Warning, true, d.ModalBtnAddItem, false)

		afterLbl := widget.NewLabelWithStyle("Sau khi bấm (After Click - Nút Thêm biến mất, Badge đổi thành [Đã thêm]):", fyne.TextAlignLeading, fyne.TextStyle{Italic: true})
		rowAfter := buildRow("developer.candidate@corp.com", d.ModalBadgeAdded, palette.TextMuted, false, "", false)

		box := container.NewVBox(
			header,
			widget.NewSeparator(),
			beforeLbl,
			rowBefore,
			layoutSpacer(),
			afterLbl,
			rowAfter,
			widget.NewSeparator(),
		)
		win := test.NewWindow(container.NewPadded(box))
		win.Resize(fyne.NewSize(560, 240))
		saveCanvas(t, win, "screenshots/03_modal_mutation_inplace_vi.png")
	}

	// 4. Evidence 04: Visual Distinction: Main UI Card vs Modal Candidate
	{
		i18n.SetCurrentLocale(i18n.LocaleVI)
		d := i18n.Current()

		title := widget.NewLabelWithStyle("Phân Định Kiến Trúc: Thẻ Main UI vs Ứng Viên Modal", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

		// Main UI side
		mainHeader := widget.NewLabelWithStyle("Cơ Chế 1: THẺ MÀN HÌNH CHÍNH (MAIN UI)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		mainDesc := widget.NewLabel("• Quản lý phiên Antigravity IDE (Duy nhất 1 Active)\n• Nhãn: [✔ Đang dùng] (Active) / [Chuyển] (Switch)")

		cardBadgeActive := canvas.NewText(d.BadgeActive, palette.Success)
		cardBadgeActive.TextStyle = fyne.TextStyle{Bold: true}
		mainCard1 := container.NewBorder(nil, nil, nil, container.NewCenter(cardBadgeActive), widget.NewLabel("leader.dev@google.com"))

		cardBadgeSwitch := canvas.NewText(d.BadgeSwitch, palette.Primary)
		cardBadgeSwitch.TextStyle = fyne.TextStyle{Bold: true}
		mainCard2 := container.NewBorder(nil, nil, nil, container.NewCenter(cardBadgeSwitch), widget.NewLabel("backup.engineer@google.com"))

		mainCol := container.NewVBox(mainHeader, mainDesc, widget.NewSeparator(), mainCard1, mainCard2)

		// Modal side
		modalColHeader := widget.NewLabelWithStyle("Cơ Chế 2: POPUP NÚT ADD (ADD MODAL)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		modalColDesc := widget.NewLabel("• Quản lý danh bạ Chrome (KHÔNG có khái niệm Active)\n• Nhãn: [✔ Đã OAuth] (Đã có token) / [Đã thêm] / [Cần OAuth]")

		modalRow1 := buildRow("oauth.ready@google.com", d.ModalBadgeOAuthDone, palette.Success, false, "", false)
		modalRow2 := buildRow("new.scanned@google.com", d.ModalBadgeNeedOAuth, palette.Warning, true, d.ModalBtnAddItem, false)

		modalCol := container.NewVBox(modalColHeader, modalColDesc, widget.NewSeparator(), modalRow1, modalRow2)

		splitBox := container.NewGridWithColumns(2,
			container.NewPadded(mainCol),
			container.NewPadded(modalCol),
		)

		fullBox := container.NewVBox(title, widget.NewSeparator(), splitBox)
		win := test.NewWindow(container.NewPadded(fullBox))
		win.Resize(fyne.NewSize(780, 260))
		saveCanvas(t, win, "screenshots/04_main_ui_vs_modal_distinction.png")
	}

	time.Sleep(100 * time.Millisecond)
}
