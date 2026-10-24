package qa_i18n_footer

import (
	"image/color"
	"image/png"
	"os"
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

// buildMockUIWindow builds a faithful replica of the app window for screenshot capture
func buildMockUIWindow(locale i18n.Locale, showModal bool) (fyne.Window, fyne.App) {
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
		Text: "Antigravity Multi-Account Switcher",
		Style: widget.RichTextStyle{
			SizeName:  theme.SizeNameHeadingText,
			TextStyle: fyne.TextStyle{Bold: true},
		},
	})
	titleLeft := container.NewHBox(monogramWrap, titleLabel)

	// Language dropdown
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

	// 2. Header Row 2
	addAccountBtn := widget.NewButtonWithIcon(d.BtnAddAccount, theme.ContentAddIcon(), nil)
	refreshBtn := widget.NewButton(d.BtnRefreshList, nil)
	quotaRefreshAllBtn := widget.NewButton(d.BtnRefreshQuota, nil)
	quotaRefreshAllBtn.Importance = widget.HighImportance
	alwaysOnTopBtn := widget.NewButton("📌", nil)

	toolbarLeft := container.NewHBox(addAccountBtn, refreshBtn, quotaRefreshAllBtn, alwaysOnTopBtn)

	lblMask := widget.NewLabelWithStyle(d.LblMaskAccounts, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	lblAuto := widget.NewLabelWithStyle(d.LblAutoSwitch, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	themeSelect := widget.NewSelect([]string{"🌙", "☀️", "☕"}, nil)
	themeSelect.SetSelected("🌙")
	themeSelectWrap := container.NewGridWrap(fyne.NewSize(58, 20), themeSelect)

	rightCluster := container.NewHBox(lblMask, widget.NewCheck("", nil), lblAuto, widget.NewCheck("", nil), themeSelectWrap)
	row2Combined := container.NewBorder(nil, nil, toolbarLeft, rightCluster)

	topSection := container.NewVBox(row1Header, row2Combined, widget.NewSeparator())

	// 3. Center Cards
	card1Item := manager.AccountDisplayItem{
		Email:    "active.user@gmail.com",
		Name:     "Active Personal",
		Tier:     "ULTRA",
		State:    model.StateActive,
		IsActive: true,
		HasToken: true,
		Status:   "on",
	}
	card2Item := manager.AccountDisplayItem{
		Email:    "team.developer@antigravity.io",
		Name:     "Work Pro",
		Tier:     "PRO",
		State:    model.StateReady,
		IsActive: false,
		HasToken: true,
		Status:   "on",
	}
	card3Item := manager.AccountDisplayItem{
		Email:    "external.client@corp.org",
		Name:     "Unauthenticated",
		Tier:     "--",
		State:    model.StateUnauth,
		IsActive: false,
		HasToken: false,
		Status:   "on",
	}

	quotaData := &telemetry.AccountQuotaData{
		Email: "active.user@gmail.com",
		Tier:  "ULTRA",
		Gemini: telemetry.ModelQuotaBucket{
			RemainingFraction: 0.90,
			FiveHourFraction:  0.90,
			WeeklyFraction:    0.95,
			FiveHourReset:     time.Now().Add(4 * time.Hour),
			WeeklyReset:       time.Now().Add(48 * time.Hour),
		},
		Claude: telemetry.ModelQuotaBucket{
			RemainingFraction: 0.75,
			FiveHourFraction:  0.75,
			WeeklyFraction:    0.60,
			FiveHourReset:     time.Now().Add(2 * time.Hour),
			WeeklyReset:       time.Now().Add(72 * time.Hour),
		},
		Status: "ACTIVE",
	}

	card1 := ui.NewAccountCardWidgetWithQuota(card1Item, quotaData, palette, nil, nil)
	card2 := ui.NewAccountCardWidgetWithQuota(card2Item, quotaData, palette, nil, nil)
	card3 := ui.NewAccountCardWidgetWithQuota(card3Item, nil, palette, nil, nil)

	cardsContainer := container.NewGridWithColumns(3, card1.Root, card2.Root, card3.Root)

	// 4. Footer Bar (Compact 22px)
	// NOTICE: Server status label in EN mode still displays Vietnamese string if hardcoded!
	serverStatusText := "Antigravity: Hoạt động  | Đang kết nối..."
	if locale == i18n.LocaleEN {
		// Demonstrating the defect where it wasn't connected to i18n
		serverStatusText = "Antigravity: Hoạt động  | Đang kết nối... [UNTRANSLATED]"
	}
	serverStatusLabel := widget.NewLabelWithStyle(serverStatusText, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	versionLabel := widget.NewLabelWithStyle(version.GetFullVersion(), fyne.TextAlignTrailing, fyne.TextStyle{})
	statusBar := widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})

	footerContent := container.NewBorder(nil, nil, serverStatusLabel, versionLabel, statusBar)
	padLeft := canvas.NewRectangle(color.Transparent)
	padLeft.SetMinSize(fyne.NewSize(8, 1))
	padRight := canvas.NewRectangle(color.Transparent)
	padRight.SetMinSize(fyne.NewSize(8, 1))
	paddedFooter := container.NewBorder(nil, nil, padLeft, padRight, footerContent)

	bottomSection := container.NewVBox(widget.NewSeparator(), paddedFooter)

	mainLayout := container.NewBorder(topSection, bottomSection, nil, nil, cardsContainer)

	if showModal {
		// Overlay Add Modal to demonstrate hardcoded Vietnamese modal in EN mode
		modalHeader := widget.NewLabelWithStyle("Thêm tài khoản", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		scanBtn := widget.NewButtonWithIcon("Quét Chrome", theme.ViewRefreshIcon(), nil)
		modalTop := container.NewBorder(nil, nil, modalHeader, scanBtn)

		cand1Badge := canvas.NewText("[✔ Đã OAuth]", palette.Success)
		cand1 := container.NewBorder(nil, nil, nil, container.NewHBox(cand1Badge, widget.NewButton("Xóa", nil)), widget.NewLabel("lead.designer@gmail.com"))

		cand2Badge := canvas.NewText("[Cần OAuth]", palette.Warning)
		cand2 := container.NewBorder(nil, nil, nil, container.NewHBox(cand2Badge, widget.NewButton("ADD", nil), widget.NewButton("Xóa", nil)), widget.NewLabel("new.candidate@gmail.com"))

		modalBox := container.NewVBox(
			modalTop,
			widget.NewSeparator(),
			cand1,
			cand2,
			widget.NewSeparator(),
			widget.NewButton("Đóng", nil),
		)
		modalPadded := container.NewPadded(modalBox)
		modalWrap := container.NewCenter(container.NewGridWrap(fyne.NewSize(460, 280), modalPadded))

		win.SetContent(container.NewStack(mainLayout, modalWrap))
	} else {
		win.SetContent(mainLayout)
	}

	return win, app
}

func layoutSpacer() fyne.CanvasObject {
	r := canvas.NewRectangle(color.Transparent)
	r.SetMinSize(fyne.NewSize(1, 1))
	return r
}

func TestRender_CaptureEvidence(t *testing.T) {
	// Evidence 1: VN UI (Default)
	winVN, appVN := buildMockUIWindow(i18n.LocaleVI, false)
	saveCanvas(t, winVN, "screenshots/01_ui_vietnamese_default.png")
	appVN.Quit()

	// Evidence 2: EN UI with un-localized server status and buttons
	winEN, appEN := buildMockUIWindow(i18n.LocaleEN, false)
	saveCanvas(t, winEN, "screenshots/02_ui_english_switched.png")
	appEN.Quit()

	// Evidence 3: Add Modal hardcoded in Vietnamese even under EN locale
	winModal, appModal := buildMockUIWindow(i18n.LocaleEN, true)
	saveCanvas(t, winModal, "screenshots/03_add_modal_hardcoded_vietnamese.png")
	appModal.Quit()
}
