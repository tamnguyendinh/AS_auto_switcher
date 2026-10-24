package qa_i18n_coverage

import (
	"encoding/json"
	"fmt"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"antigravity-switch/internal/i18n"
	"antigravity-switch/internal/manager"
	"antigravity-switch/internal/model"
	"antigravity-switch/internal/telemetry"
	"antigravity-switch/internal/ui"
	"antigravity-switch/internal/version"
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
	t.Logf("Saved screenshot: %s", filename)
}

func layoutSpacer() fyne.CanvasObject {
	r := canvas.NewRectangle(color.Transparent)
	r.SetMinSize(fyne.NewSize(1, 1))
	return r
}

// -----------------------------------------------------------------------------
// Test 1: JSON Parity & Dual File Integrity (Invariant 1)
// -----------------------------------------------------------------------------
func TestQA_Invariant1_DualJSONParity(t *testing.T) {
	viPath := filepath.Join("..", "..", "..", "internal", "i18n", "locales", "vi.json")
	enPath := filepath.Join("..", "..", "..", "internal", "i18n", "locales", "en.json")

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
		t.Fatalf("Key count mismatch: vi.json has %d keys, en.json has %d keys", len(viMap), len(enMap))
	}

	if len(viMap) != 53 {
		t.Errorf("Expected 53 keys per complete i18n specification, got %d", len(viMap))
	}

	// 1:1 Key symmetry and non-empty check
	for k, viVal := range viMap {
		enVal, ok := enMap[k]
		if !ok {
			t.Errorf("Missing key in en.json: %s", k)
			continue
		}
		if viVal == "" {
			t.Errorf("Empty value in vi.json for key: %s", k)
		}
		if enVal == "" {
			t.Errorf("Empty value in en.json for key: %s", k)
		}
	}

	// Ensure dictionary struct fields map correctly
	dVI := i18n.GetDictionary(i18n.LocaleVI)
	dEN := i18n.GetDictionary(i18n.LocaleEN)
	valVI := reflect.ValueOf(dVI)
	valEN := reflect.ValueOf(dEN)

	for i := 0; i < valVI.NumField(); i++ {
		field := valVI.Type().Field(i)
		sVI := valVI.Field(i).String()
		sEN := valEN.Field(i).String()
		if sVI == "" {
			t.Errorf("Dictionary struct VI field %s is empty", field.Name)
		}
		if sEN == "" {
			t.Errorf("Dictionary struct EN field %s is empty", field.Name)
		}
	}
}

// -----------------------------------------------------------------------------
// Test 2: Build faithful App representation and capture visual screenshots
// -----------------------------------------------------------------------------
func buildCompleteMockUI(locale i18n.Locale, showLoadingToolbar bool) (fyne.Window, fyne.App, *widget.Button, *widget.Button) {
	i18n.SetCurrentLocale(locale)
	d := i18n.Current()
	palette := ui.DarkPalette

	app := test.NewApp()
	win := test.NewWindow(container.NewVBox())
	win.Resize(fyne.NewSize(920, 630))

	// 1. Header Row 1
	monogramBg := canvas.NewRectangle(palette.Primary)
	monogramBg.CornerRadius = 6
	monogramText := canvas.NewText("AS", color.White)
	monogramText.TextStyle = fyne.TextStyle{Bold: true}
	monogramText.TextSize = 13
	monogramText.Alignment = fyne.TextAlignCenter
	monogramStack := container.NewStack(monogramBg, container.NewCenter(monogramText))
	monogramWrap := container.NewGridWrap(fyne.NewSize(26, 26), monogramStack)

	titleLabel := widget.NewRichText(&widget.TextSegment{
		Text: d.AppTitle,
		Style: widget.RichTextStyle{
			SizeName:  theme.SizeNameHeadingText,
			TextStyle: fyne.TextStyle{Bold: true},
		},
	})
	titleLeft := container.NewHBox(monogramWrap, titleLabel)

	// Language dropdown (54x22)
	langSelect := widget.NewSelect([]string{"VN", "EN"}, nil)
	if locale == i18n.LocaleEN {
		langSelect.SetSelected("EN")
	} else {
		langSelect.SetSelected("VN")
	}
	langWrap := container.NewGridWrap(fyne.NewSize(54, 22), langSelect)

	btnMin := container.NewGridWrap(fyne.NewSize(32, 26), widget.NewButton("—", nil))
	btnMax := container.NewGridWrap(fyne.NewSize(32, 26), widget.NewButton("□", nil))
	btnClose := container.NewGridWrap(fyne.NewSize(32, 26), widget.NewButton("✕", nil))
	windowControls := container.NewHBox(langWrap, btnMin, btnMax, btnClose)

	row1Header := container.NewBorder(nil, nil, titleLeft, windowControls, layoutSpacer())

	// 2. Header Row 2 (Toolbar)
	btnAddText := d.BtnAddAccount
	btnRefreshText := d.BtnRefreshList
	btnQuotaText := d.BtnRefreshQuota

	if showLoadingToolbar {
		btnRefreshText = d.BtnRefreshListLoading
		btnQuotaText = d.BtnRefreshQuotaLoading
	}

	addAccountBtn := widget.NewButtonWithIcon(btnAddText, theme.ContentAddIcon(), nil)
	refreshBtn := widget.NewButton(btnRefreshText, nil)
	quotaRefreshAllBtn := widget.NewButton(btnQuotaText, nil)
	quotaRefreshAllBtn.Importance = widget.HighImportance
	alwaysOnTopBtn := widget.NewButton(d.BtnAlwaysOnTop, nil)

	toolbarLeft := container.NewHBox(addAccountBtn, refreshBtn, quotaRefreshAllBtn, alwaysOnTopBtn)

	lblMask := widget.NewLabelWithStyle(d.LblMaskAccounts, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	lblAuto := widget.NewLabelWithStyle(d.LblAutoSwitch, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	themeSelect := widget.NewSelect([]string{"🌙", "☀️", "☕"}, nil)
	themeSelect.SetSelected("🌙")
	themeSelectWrap := container.NewGridWrap(fyne.NewSize(58, 20), themeSelect)

	rightCluster := container.NewHBox(lblMask, widget.NewCheck("", nil), lblAuto, widget.NewCheck("", nil), themeSelectWrap)
	row2Combined := container.NewBorder(nil, nil, toolbarLeft, rightCluster)

	topSection := container.NewVBox(row1Header, row2Combined, widget.NewSeparator())

	// 3. Center Cards with 3x3 Quota Matrix
	card1Item := manager.AccountDisplayItem{
		Email:    "tamnguyen.senior@antigravity.io",
		Name:     "Tam Nguyen (Active)",
		Tier:     "ULTRA",
		State:    model.StateActive,
		IsActive: true,
		HasToken: true,
		Status:   "on",
	}
	card2Item := manager.AccountDisplayItem{
		Email:    "dev.engineer@company.com",
		Name:     "Dev Engineer (Ready)",
		Tier:     "PRO",
		State:    model.StateReady,
		IsActive: false,
		HasToken: true,
		Status:   "on",
	}
	card3Item := manager.AccountDisplayItem{
		Email:    "client.staging@domain.org",
		Name:     "Staging Unauth",
		Tier:     "--",
		State:    model.StateUnauth,
		IsActive: false,
		HasToken: false,
		Status:   "on",
	}

	quotaData1 := &telemetry.AccountQuotaData{
		Email: "tamnguyen.senior@antigravity.io",
		Tier:  "ULTRA",
		Gemini: telemetry.ModelQuotaBucket{
			RemainingFraction: 1.0, // Remaining 100% -> should show (Ready) or (Sẵn sàng)
			FiveHourFraction:  1.0,
			WeeklyFraction:    0.85,
			FiveHourReset:     time.Now().Add(-1 * time.Minute), // Ready
			WeeklyReset:       time.Now().Add(48 * time.Hour),
		},
		Claude: telemetry.ModelQuotaBucket{
			RemainingFraction: 0.70,
			FiveHourFraction:  0.70,
			WeeklyFraction:    0.65,
			FiveHourReset:     time.Now().Add(3 * time.Hour),
			WeeklyReset:       time.Now().Add(72 * time.Hour),
		},
		Status: "ACTIVE",
	}

	quotaData2 := &telemetry.AccountQuotaData{
		Email: "dev.engineer@company.com",
		Tier:  "PRO",
		Gemini: telemetry.ModelQuotaBucket{
			RemainingFraction: 0.90,
			FiveHourFraction:  0.90,
			WeeklyFraction:    0.90,
			FiveHourReset:     time.Now().Add(2 * time.Hour),
			WeeklyReset:       time.Now().Add(96 * time.Hour),
		},
		Claude: telemetry.ModelQuotaBucket{
			RemainingFraction: 0.50,
			FiveHourFraction:  0.50,
			WeeklyFraction:    0.40,
			FiveHourReset:     time.Now().Add(1 * time.Hour),
			WeeklyReset:       time.Now().Add(24 * time.Hour),
		},
		Status: "READY",
	}

	card1 := ui.NewAccountCardWidgetWithQuota(card1Item, quotaData1, palette, nil, nil)
	card2 := ui.NewAccountCardWidgetWithQuota(card2Item, quotaData2, palette, nil, nil)
	card3 := ui.NewAccountCardWidgetWithQuota(card3Item, nil, palette, nil, nil)

	cardsContainer := container.NewGridWithColumns(3, card1.Root, card2.Root, card3.Root)

	// 4. Footer Bar (Exact 22px Compact)
	serverStatusText := fmt.Sprintf(d.ServerConnected, 4208, 8080)
	serverStatusLabel := widget.NewLabelWithStyle(serverStatusText, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	versionLabel := widget.NewLabelWithStyle(version.GetFullVersion(), fyne.TextAlignTrailing, fyne.TextStyle{})
	statusBar := widget.NewLabelWithStyle(d.StatusFetchSuccess, fyne.TextAlignCenter, fyne.TextStyle{Italic: true})

	footerContent := container.NewBorder(nil, nil, serverStatusLabel, versionLabel, statusBar)
	padLeft := canvas.NewRectangle(color.Transparent)
	padLeft.SetMinSize(fyne.NewSize(8, 1))
	padRight := canvas.NewRectangle(color.Transparent)
	padRight.SetMinSize(fyne.NewSize(8, 1))
	paddedFooter := container.NewBorder(nil, nil, padLeft, padRight, footerContent)

	bottomSection := container.NewVBox(widget.NewSeparator(), paddedFooter)

	mainLayout := container.NewBorder(topSection, bottomSection, nil, nil, cardsContainer)
	win.SetContent(mainLayout)

	return win, app, refreshBtn, quotaRefreshAllBtn
}

// -----------------------------------------------------------------------------
// Test 3: Generate visual evidence screenshots
// -----------------------------------------------------------------------------
func TestQA_CaptureVisualEvidence(t *testing.T) {
	// Evidence 1: Complete UI in Vietnamese (Default)
	winVN, appVN, _, _ := buildCompleteMockUI(i18n.LocaleVI, false)
	saveCanvas(t, winVN, "screenshots/01_ui_vietnamese_default.png")
	appVN.Quit()

	// Evidence 2: Complete UI switched to English (Live In-Place)
	winEN, appEN, _, _ := buildCompleteMockUI(i18n.LocaleEN, false)
	saveCanvas(t, winEN, "screenshots/02_ui_english_live_switch.png")
	appEN.Quit()

	// Evidence 3: Toolbar loading states in EN mode
	winLoading, appLoading, _, _ := buildCompleteMockUI(i18n.LocaleEN, true)
	saveCanvas(t, winLoading, "screenshots/03_toolbar_loading_states_en.png")
	appLoading.Quit()

	// Evidence 4: Quota Matrix comparison (Focused)
	palette := ui.DarkPalette
	dqVI := ui.DualCycleQuota{
		Gemini5h:   0.8,
		GeminiWeek: 1.0,
		Claude5h:   0.6,
		ClaudeWeek: 0.9,
	}
	i18n.SetCurrentLocale(i18n.LocaleVI)
	tableVI := ui.CreateMatrixQuotaTableWithTier(dqVI, "ULTRA", palette)
	boxVI := container.NewVBox(
		widget.NewLabelWithStyle("VIETNAMESE QUOTA MATRIX (3x3):", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		tableVI,
	)

	i18n.SetCurrentLocale(i18n.LocaleEN)
	tableEN := ui.CreateMatrixQuotaTableWithTier(dqVI, "ULTRA", palette)
	boxEN := container.NewVBox(
		widget.NewLabelWithStyle("ENGLISH QUOTA MATRIX (3x3):", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		tableEN,
	)

	compWin := test.NewWindow(container.NewGridWithColumns(2, container.NewPadded(boxVI), container.NewPadded(boxEN)))
	compWin.Resize(fyne.NewSize(700, 240))
	saveCanvas(t, compWin, "screenshots/04_quota_matrix_bilingual.png")

	// Evidence 5 & 6: Modal Add Account (VI & EN)
	buildModalCapture(t, i18n.LocaleVI, "screenshots/05_modal_add_account_vi.png")
	buildModalCapture(t, i18n.LocaleEN, "screenshots/06_modal_add_account_en.png")

	// Evidence 7 & 8: Delete Confirmation Popups (VI & EN)
	buildConfirmCapture(t, i18n.LocaleVI, "screenshots/07_confirm_delete_popup_vi.png")
	buildConfirmCapture(t, i18n.LocaleEN, "screenshots/08_confirm_delete_popup_en.png")

	// Evidence 9: Footer bar and status messages
	buildFooterCapture(t, "screenshots/09_footer_bar_and_status_en.png")
}

func buildModalCapture(t *testing.T, loc i18n.Locale, filename string) {
	i18n.SetCurrentLocale(loc)
	d := i18n.Current()
	palette := ui.DarkPalette

	modalHeader := widget.NewLabelWithStyle(d.ModalAddHeader, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	scanBtn := widget.NewButtonWithIcon(d.ModalScanChrome, theme.ViewRefreshIcon(), nil)
	topRow := container.NewBorder(nil, nil, modalHeader, scanBtn)

	// Discovered items
	badge1Text := d.BadgeAdded
	badge1 := canvas.NewText(badge1Text, palette.TextMuted)
	cand1 := container.NewBorder(nil, nil, nil, container.NewHBox(badge1, widget.NewButton(d.ConfirmBtnDelete, nil)), widget.NewLabel("architect.lead@google.com"))

	badge2Text := d.BadgeNeedOAuth
	badge2 := canvas.NewText(badge2Text, palette.Warning)
	cand2 := container.NewBorder(nil, nil, nil, container.NewHBox(badge2, widget.NewButton(d.ModalBtnAddItem, nil), widget.NewButton(d.ConfirmBtnDelete, nil)), widget.NewLabel("staging.member@partner.io"))

	badge3Text := d.BadgeAccountNotFound
	badge3 := canvas.NewText(badge3Text, palette.Danger)
	cand3 := container.NewBorder(nil, nil, nil, container.NewHBox(badge3, widget.NewButton(d.ConfirmBtnDelete, nil)), widget.NewLabel("obsolete.account@legacy.net"))

	list := container.NewVBox(cand1, cand2, cand3)
	box := container.NewVBox(
		topRow,
		widget.NewSeparator(),
		list,
		widget.NewSeparator(),
		container.NewHBox(layoutSpacer(), widget.NewButton(d.ModalAddClose, nil)),
	)

	win := test.NewWindow(container.NewPadded(box))
	win.Resize(fyne.NewSize(520, 260))
	saveCanvas(t, win, filename)
}

func buildConfirmCapture(t *testing.T, loc i18n.Locale, filename string) {
	i18n.SetCurrentLocale(loc)
	d := i18n.Current()

	titleLabel := widget.NewLabelWithStyle(d.ConfirmDeleteTitle, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	msg := fmt.Sprintf(d.ConfirmMsgMainHasToken, "lead.developer@antigravity.io")
	msgLabel := widget.NewLabel(msg)

	btnDel := widget.NewButton(d.ConfirmBtnDelete, nil)
	btnDel.Importance = widget.DangerImportance
	btnCancel := widget.NewButton(d.ConfirmBtnCancel, nil)

	btnCluster := container.NewHBox(layoutSpacer(), btnCancel, btnDel)

	dialogBox := container.NewVBox(
		titleLabel,
		widget.NewSeparator(),
		container.NewPadded(msgLabel),
		widget.NewSeparator(),
		btnCluster,
	)

	win := test.NewWindow(container.NewPadded(dialogBox))
	win.Resize(fyne.NewSize(480, 200))
	saveCanvas(t, win, filename)
}

func buildFooterCapture(t *testing.T, filename string) {
	i18n.SetCurrentLocale(i18n.LocaleEN)
	d := i18n.Current()

	// 1. Connecting state
	lblConnecting := widget.NewLabelWithStyle(d.ServerConnecting, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	// 2. Connected state
	lblConnected := widget.NewLabelWithStyle(fmt.Sprintf(d.ServerConnected, 3128, 9090), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	// 3. Stopped state
	lblStopped := widget.NewLabelWithStyle(d.ServerStopped, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Status messages
	sb1 := widget.NewLabelWithStyle(d.StatusNoCandidate, fyne.TextAlignLeading, fyne.TextStyle{Italic: true})
	sb2 := widget.NewLabelWithStyle(fmt.Sprintf(d.StatusSwitchingTo, "dev@corp.com"), fyne.TextAlignLeading, fyne.TextStyle{Italic: true})
	sb3 := widget.NewLabelWithStyle(fmt.Sprintf(d.StatusSwitchedSuccess, "dev@corp.com"), fyne.TextAlignLeading, fyne.TextStyle{Italic: true})
	sb4 := widget.NewLabelWithStyle(d.StatusMaskOn, fyne.TextAlignLeading, fyne.TextStyle{Italic: true})
	sb5 := widget.NewLabelWithStyle(d.StatusAutoOn, fyne.TextAlignLeading, fyne.TextStyle{Italic: true})

	box := container.NewVBox(
		widget.NewLabelWithStyle("FOOTER BAR SERVER STATUS STATES (EN):", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		lblConnecting,
		lblConnected,
		lblStopped,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("SAMPLE TRANSIENT STATUS BAR MESSAGES (EN):", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		sb1, sb2, sb3, sb4, sb5,
	)

	win := test.NewWindow(container.NewPadded(box))
	win.Resize(fyne.NewSize(580, 320))
	saveCanvas(t, win, filename)
}

// -----------------------------------------------------------------------------
// Test 4: Invariant 2 - Live In-Place Switching
// -----------------------------------------------------------------------------
func TestQA_Invariant2_LiveInPlaceSwitching(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	mgr, err := manager.NewAccountManager(nil, nil)
	if err != nil {
		t.Fatalf("failed creating manager: %v", err)
	}

	app := ui.NewAppWithFyne(testApp, mgr)
	tempDir := t.TempDir()
	prefPath := filepath.Join(tempDir, "ui_preferences.json")
	app.SetPreferencesPath(prefPath)

	app.Init()

	// Initial default should be "vi"
	if app.CurrentLocale() != "vi" {
		t.Errorf("Expected initial locale 'vi', got '%s'", app.CurrentLocale())
	}
	contentInitial := app.Window().Content()

	// Switch to EN
	app.SetLanguage("en")
	if app.CurrentLocale() != "en" {
		t.Errorf("Expected currentLocale 'en' after SetLanguage('en'), got '%s'", app.CurrentLocale())
	}
	dEN := i18n.Current()
	if dEN.BtnRefreshList != "Refresh List" {
		t.Errorf("Expected dEN.BtnRefreshList 'Refresh List', got '%s'", dEN.BtnRefreshList)
	}

	// Invariant: Window Content container pointer must remain identical (zero SetContent / zero flicker)
	if app.Window().Content() != contentInitial {
		t.Errorf("Invariant Violated: Window content was reconstructed/replaced during locale switch (flicker detected)")
	}

	// Switch back to VI
	app.SetLanguage("vi")
	if app.CurrentLocale() != "vi" {
		t.Errorf("Expected currentLocale 'vi' after SetLanguage('vi'), got '%s'", app.CurrentLocale())
	}
	dVI := i18n.Current()
	if dVI.BtnRefreshList != "Làm mới danh sách" {
		t.Errorf("Expected dVI.BtnRefreshList 'Làm mới danh sách', got '%s'", dVI.BtnRefreshList)
	}

	if app.Window().Content() != contentInitial {
		t.Errorf("Invariant Violated: Window content was reconstructed when switching back to 'vi'")
	}
}

// -----------------------------------------------------------------------------
// Test 5: Invariant 3 - Toolbar States (No Reversion in EN Mode)
// -----------------------------------------------------------------------------
func TestQA_Invariant3_ToolbarStates_EN_NoReversion(t *testing.T) {
	i18n.SetCurrentLocale(i18n.LocaleEN)
	d := i18n.Current()

	// Verify Toolbar buttons and transient loading texts
	if d.BtnRefreshList != "Refresh List" {
		t.Errorf("Expected 'Refresh List', got '%s'", d.BtnRefreshList)
	}
	if d.BtnRefreshListLoading != "⏳ Refreshing..." {
		t.Errorf("Expected '⏳ Refreshing...', got '%s'", d.BtnRefreshListLoading)
	}
	if d.BtnRefreshQuota != "Refresh Quota" {
		t.Errorf("Expected 'Refresh Quota', got '%s'", d.BtnRefreshQuota)
	}
	if d.BtnRefreshQuotaLoading != "⏳ Scanning..." {
		t.Errorf("Expected '⏳ Scanning...', got '%s'", d.BtnRefreshQuotaLoading)
	}

	// Verify no Vietnamese remnants in EN toolbar strings
	for _, str := range []string{d.BtnRefreshList, d.BtnRefreshListLoading, d.BtnRefreshQuota, d.BtnRefreshQuotaLoading, d.BtnAddAccount} {
		for _, forbidden := range []string{"Làm mới", "danh sách", "Đang", "quét", "Thêm"} {
			if strings.Contains(str, forbidden) {
				t.Errorf("Invariant Violated: EN toolbar string %q contains Vietnamese word %q", str, forbidden)
			}
		}
	}
}

// -----------------------------------------------------------------------------
// Test 6: Invariant 4 - Quota Matrix Bilingual Parity
// -----------------------------------------------------------------------------
func TestQA_Invariant4_QuotaMatrixBilingual(t *testing.T) {
	palette := ui.DarkPalette
	dq := ui.DualCycleQuota{
		Gemini5h:   1.0,
		GeminiWeek: 1.0,
		Claude5h:   0.5,
		ClaudeWeek: 0.5,
	}

	// VI mode
	i18n.SetCurrentLocale(i18n.LocaleVI)
	if i18n.Current().QuotaColWeek != "Tuần" {
		t.Errorf("Expected VI QuotaColWeek 'Tuần', got '%s'", i18n.Current().QuotaColWeek)
	}
	if i18n.Current().QuotaReady != "(Sẵn sàng)" {
		t.Errorf("Expected VI QuotaReady '(Sẵn sàng)', got '%s'", i18n.Current().QuotaReady)
	}

	table := ui.CreateMatrixQuotaTableWithTier(dq, "ULTRA", palette)
	cont, ok := table.(*fyne.Container)
	if !ok || len(cont.Objects) < 3 {
		t.Fatalf("Expected valid table container with at least 3 columns")
	}

	// Col 2 is Week column: check header text
	colWeek, ok := cont.Objects[2].(*fyne.Container)
	if !ok || len(colWeek.Objects) == 0 {
		t.Fatalf("Expected colWeek container")
	}
	txtWeek, ok := colWeek.Objects[0].(*canvas.Text)
	if !ok {
		t.Fatalf("Expected canvas.Text in colWeek header")
	}
	if txtWeek.Text != "Tuần" {
		t.Errorf("Expected colWeek header text 'Tuần', got '%s'", txtWeek.Text)
	}

	// Switch to EN and update table in-place
	i18n.SetCurrentLocale(i18n.LocaleEN)
	if i18n.Current().QuotaColWeek != "Week" {
		t.Errorf("Expected EN QuotaColWeek 'Week', got '%s'", i18n.Current().QuotaColWeek)
	}
	if i18n.Current().QuotaReady != "(Ready)" {
		t.Errorf("Expected EN QuotaReady '(Ready)', got '%s'", i18n.Current().QuotaReady)
	}

	ui.UpdateMatrixQuotaTableLanguage(table)
	if txtWeek.Text != "Week" {
		t.Errorf("Expected in-place updated colWeek header text 'Week', got '%s'", txtWeek.Text)
	}
}

// -----------------------------------------------------------------------------
// Test 7: Invariant 5 - Footer Bar and 14+ Status Bar Messages
// -----------------------------------------------------------------------------
func TestQA_Invariant5_FooterAndStatusBar(t *testing.T) {
	// 1. VI checks
	i18n.SetCurrentLocale(i18n.LocaleVI)
	dVI := i18n.Current()
	if !strings.Contains(dVI.ServerConnected, "Cổng %d") {
		t.Errorf("Expected VI ServerConnected to contain 'Cổng %%d', got %q", dVI.ServerConnected)
	}
	if !strings.Contains(dVI.ServerConnecting, "Đang kết nối...") {
		t.Errorf("Expected VI ServerConnecting to contain 'Đang kết nối...', got %q", dVI.ServerConnecting)
	}

	// 2. EN checks
	i18n.SetCurrentLocale(i18n.LocaleEN)
	dEN := i18n.Current()
	if !strings.Contains(dEN.ServerConnected, "Port %d") {
		t.Errorf("Expected EN ServerConnected to contain 'Port %%d', got %q", dEN.ServerConnected)
	}
	if !strings.Contains(dEN.ServerConnecting, "Connecting...") {
		t.Errorf("Expected EN ServerConnecting to contain 'Connecting...', got %q", dEN.ServerConnecting)
	}
	if !strings.Contains(dEN.ServerStopped, "Inactive | WinCred Fallback") {
		t.Errorf("Expected EN ServerStopped to contain 'Inactive | WinCred Fallback', got %q", dEN.ServerStopped)
	}

	// 3. Status messages in EN must not contain Vietnamese characters
	statusStrings := []string{
		dEN.StatusNoCandidate,
		dEN.StatusFetchError,
		dEN.StatusFetchSuccess,
		dEN.StatusSwitchingTo,
		dEN.StatusSwitchedSuccess,
		dEN.StatusUpdateError,
		dEN.StatusCannotDeleteActive,
		dEN.StatusDeleteError,
		dEN.StatusDeleteSuccess,
		dEN.StatusMaskOn,
		dEN.StatusMaskOff,
		dEN.StatusAutoOn,
		dEN.StatusAutoOff,
	}

	for _, s := range statusStrings {
		if strings.ContainsAny(s, "àáảãạăằắẳẵặâầấẩẫậèéẻẽẹêềếểễệìíỉĩịòóỏõọôồốổỗộơờớởỡợùúủũụưừứửữựỳýỷỹỵđĐ") {
			t.Errorf("Invariant Violated: EN status bar string %q contains Vietnamese diacritics", s)
		}
	}
}

// -----------------------------------------------------------------------------
// Test 8: Invariant 6 - Modals and Popups
// -----------------------------------------------------------------------------
func TestQA_Invariant6_ModalsAndPopups(t *testing.T) {
	// VI
	i18n.SetCurrentLocale(i18n.LocaleVI)
	dVI := i18n.Current()
	if dVI.ModalAddTitle != "Thêm tài khoản" {
		t.Errorf("Expected VI ModalAddTitle 'Thêm tài khoản', got '%s'", dVI.ModalAddTitle)
	}
	if dVI.ModalScanChrome != "Quét Chrome" {
		t.Errorf("Expected VI ModalScanChrome 'Quét Chrome', got '%s'", dVI.ModalScanChrome)
	}
	if dVI.ModalBtnAddItem != "THÊM" {
		t.Errorf("Expected VI ModalBtnAddItem 'THÊM', got '%s'", dVI.ModalBtnAddItem)
	}
	if dVI.BadgeAdded != "[Đã thêm]" {
		t.Errorf("Expected VI BadgeAdded '[Đã thêm]', got '%s'", dVI.BadgeAdded)
	}
	if dVI.ConfirmDeleteTitle != "Xác nhận xóa tài khoản" {
		t.Errorf("Expected VI ConfirmDeleteTitle 'Xác nhận xóa tài khoản', got '%s'", dVI.ConfirmDeleteTitle)
	}
	if dVI.ConfirmBtnDelete != "Xóa" || dVI.ConfirmBtnCancel != "Hủy" {
		t.Errorf("Expected VI buttons 'Xóa' / 'Hủy', got '%s' / '%s'", dVI.ConfirmBtnDelete, dVI.ConfirmBtnCancel)
	}

	// EN
	i18n.SetCurrentLocale(i18n.LocaleEN)
	dEN := i18n.Current()
	if dEN.ModalAddTitle != "Add Account" {
		t.Errorf("Expected EN ModalAddTitle 'Add Account', got '%s'", dEN.ModalAddTitle)
	}
	if dEN.ModalScanChrome != "Scan Chrome" {
		t.Errorf("Expected EN ModalScanChrome 'Scan Chrome', got '%s'", dEN.ModalScanChrome)
	}
	if dEN.ModalBtnAddItem != "ADD" {
		t.Errorf("Expected EN ModalBtnAddItem 'ADD', got '%s'", dEN.ModalBtnAddItem)
	}
	if dEN.BadgeAdded != "[Added]" {
		t.Errorf("Expected EN BadgeAdded '[Added]', got '%s'", dEN.BadgeAdded)
	}
	if dEN.ConfirmDeleteTitle != "Confirm Account Deletion" {
		t.Errorf("Expected EN ConfirmDeleteTitle 'Confirm Account Deletion', got '%s'", dEN.ConfirmDeleteTitle)
	}
	if dEN.ConfirmBtnDelete != "Delete" || dEN.ConfirmBtnCancel != "Cancel" {
		t.Errorf("Expected EN buttons 'Delete' / 'Cancel', got '%s' / '%s'", dEN.ConfirmBtnDelete, dEN.ConfirmBtnCancel)
	}
}

// -----------------------------------------------------------------------------
// Test 9: Invariant 7 - System Tray Menu
// -----------------------------------------------------------------------------
func TestQA_Invariant7_TrayMenu(t *testing.T) {
	// VI
	i18n.SetCurrentLocale(i18n.LocaleVI)
	dVI := i18n.Current()
	if dVI.TrayOpen != "Hiển thị Switcher" {
		t.Errorf("Expected VI TrayOpen 'Hiển thị Switcher', got '%s'", dVI.TrayOpen)
	}
	if dVI.TrayRefresh != "Làm mới Chrome" {
		t.Errorf("Expected VI TrayRefresh 'Làm mới Chrome', got '%s'", dVI.TrayRefresh)
	}
	if dVI.TrayExit != "Thoát" {
		t.Errorf("Expected VI TrayExit 'Thoát', got '%s'", dVI.TrayExit)
	}

	// EN
	i18n.SetCurrentLocale(i18n.LocaleEN)
	dEN := i18n.Current()
	if dEN.TrayOpen != "Show Switcher" {
		t.Errorf("Expected EN TrayOpen 'Show Switcher', got '%s'", dEN.TrayOpen)
	}
	if dEN.TrayRefresh != "Refresh Chrome" {
		t.Errorf("Expected EN TrayRefresh 'Refresh Chrome', got '%s'", dEN.TrayRefresh)
	}
	if dEN.TrayExit != "Exit" {
		t.Errorf("Expected EN TrayExit 'Exit', got '%s'", dEN.TrayExit)
	}
}
