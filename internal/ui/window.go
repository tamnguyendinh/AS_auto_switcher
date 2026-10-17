package ui

import (
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"golang.org/x/sys/windows"
)

const windowTitle = "⚡ Antigravity Multi-Account Switcher"

var (
	modUser32        = windows.NewLazySystemDLL("user32.dll")
	procSetWindowPos = modUser32.NewProc("SetWindowPos")
	procFindWindowW  = modUser32.NewProc("FindWindowW")
)

const (
	hwndTopMost   = ^uintptr(0) // (HWND)-1
	hwndNoTopMost = ^uintptr(1) // (HWND)-2
	swpNoMove     = 0x0002
	swpNoSize     = 0x0001
)

func setWindowAlwaysOnTop(title string, isTop bool) {
	if procFindWindowW.Find() != nil || procSetWindowPos.Find() != nil {
		return
	}
	titlePtr := windows.StringToUTF16Ptr(title)
	hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(titlePtr)))
	if hwnd == 0 {
		return
	}
	var insertAfter uintptr = hwndNoTopMost
	if isTop {
		insertAfter = hwndTopMost
	}
	flags := uintptr(swpNoMove | swpNoSize)
	_, _, _ = procSetWindowPos.Call(hwnd, insertAfter, 0, 0, 0, 0, flags)
}

// setupLayout builds the main compact 440x560 window layout.
func (a *App) setupLayout() {
	palette := GetThemePalette(a.currentTheme)

	// 1. Header Card with Brand Logo & Specular Keyline Border
	logoImg := canvas.NewImageFromResource(AppLogoResource)
	logoImg.FillMode = canvas.ImageFillContain
	logoContainer := container.NewGridWrap(fyne.NewSize(32, 32), logoImg)

	titleLabel := widget.NewLabelWithStyle(windowTitle, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	titleHBox := container.NewHBox(container.NewCenter(logoContainer), titleLabel)

	// Multi-Theme Selector (UI-TR-06)
	themeOptions := []string{"🌙 Tối", "☀️ Sáng", "☕ Ấm"}
	a.themeSelect = widget.NewSelect(themeOptions, func(selected string) {
		switch selected {
		case "🌙 Tối":
			a.SetTheme(ThemeDark)
		case "☀️ Sáng":
			a.SetTheme(ThemeLight)
		case "☕ Ấm":
			a.SetTheme(ThemeWarm)
		}
	})

	switch a.currentTheme {
	case ThemeLight:
		a.themeSelect.SetSelected("☀️ Sáng")
	case ThemeWarm:
		a.themeSelect.SetSelected("☕ Ấm")
	default:
		a.themeSelect.SetSelected("🌙 Tối")
	}

	headerTopRow := container.NewBorder(nil, nil, titleHBox, a.themeSelect)

	// Active Account info
	a.activeDotContainer = container.NewHBox(NewStatusDot(palette.Warning, 12))
	a.activeStatusLabel = widget.NewLabelWithStyle("● Đang kiểm tra Antigravity...", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	activeHBox := container.NewHBox(a.activeDotContainer, a.activeStatusLabel)

	// Active Quota summary
	a.activeQuotaLabel = widget.NewLabelWithStyle("Hạn ngạch: Đang kiểm tra...", fyne.TextAlignLeading, fyne.TextStyle{Italic: true})

	a.serverStatusLabel = widget.NewLabelWithStyle("● Đang kết nối Language Server...", fyne.TextAlignLeading, fyne.TextStyle{Italic: true})

	headerBox := container.NewVBox(
		headerTopRow,
		activeHBox,
		a.activeQuotaLabel,
		a.serverStatusLabel,
	)

	a.headerBg = canvas.NewRectangle(palette.Card)
	a.headerBg.StrokeColor = palette.BorderSubtle
	a.headerBg.StrokeWidth = 1
	a.headerBg.CornerRadius = 8
	a.headerCard = container.NewStack(a.headerBg, container.NewPadded(headerBox))

	// 2. Action Controls Bar & Header Toolbar
	a.refreshBtn = widget.NewButtonWithIcon("🔄 Chrome", theme.ViewRefreshIcon(), func() {
		a.Refresh()
	})
	a.saveBtn = widget.NewButtonWithIcon("💾 Lưu phiên", theme.DocumentSaveIcon(), func() {
		a.SaveSession()
	})

	// Global Quota Refresh Button (UI-TR-01)
	a.quotaRefreshAllBtn = widget.NewButtonWithIcon("📊 Làm mới Quota Tất cả", theme.ViewRefreshIcon(), func() {
		a.RefreshAllQuota()
	})
	a.quotaRefreshAllBtn.Importance = widget.HighImportance

	pinLabel := "📌 Ghim: BẬT"
	pinImportance := widget.HighImportance
	if !a.alwaysOnTop {
		pinLabel = "📌 Ghim: TẮT"
		pinImportance = widget.MediumImportance
	}
	a.alwaysOnTopBtn = widget.NewButton(pinLabel, func() {
		a.ToggleAlwaysOnTop()
	})
	a.alwaysOnTopBtn.Importance = pinImportance

	// Auto-Refresh 60s Indicator (UI-TR-03)
	a.autoRefreshLabel = widget.NewLabelWithStyle("Tự động: 60s | Cập nhật: --:--:--", fyne.TextAlignTrailing, fyne.TextStyle{Italic: true})

	// Toolbar Grid: Row 1 = Chrome, Save, Quota All
	row1Grid := container.NewGridWithColumns(3, a.refreshBtn, a.saveBtn, a.quotaRefreshAllBtn)
	// Toolbar Row 2 = Pin button + Auto-Refresh indicator
	row2Box := container.NewBorder(nil, nil, a.alwaysOnTopBtn, nil, container.NewHBox(layout.NewSpacer(), a.autoRefreshLabel))

	ctrlContainer := container.NewVBox(row1Grid, row2Box)

	// 3. Legend Bar
	legendLabel := widget.NewLabelWithStyle("🟢 Sẵn sàng   |   🟡 Chưa có file JSON   |   🔴 Hạn ngạch < 15%", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})
	a.legendBg = canvas.NewRectangle(palette.Card)
	a.legendBg.StrokeColor = palette.BorderSubtle
	a.legendBg.StrokeWidth = 1
	a.legendBg.CornerRadius = 6
	legendCard := container.NewStack(a.legendBg, container.NewPadded(legendLabel))

	topSection := container.NewVBox(
		container.NewPadded(a.headerCard),
		ctrlContainer,
		legendCard,
	)

	// 4. Scrollable Account List
	a.accountListContainer = container.NewVBox()
	scrollList := container.NewVScroll(a.accountListContainer)

	// 5. Status Bar Footer with Specular Keyline Border
	a.statusBarLabel = widget.NewLabelWithStyle("Sẵn sàng", fyne.TextAlignLeading, fyne.TextStyle{Italic: true})
	a.statusBg = canvas.NewRectangle(palette.Card)
	a.statusBg.StrokeColor = palette.BorderSubtle
	a.statusBg.StrokeWidth = 1
	a.statusBg.CornerRadius = 6
	bottomSection := container.NewStack(a.statusBg, container.NewPadded(a.statusBarLabel))

	// Main Layout
	mainLayout := container.NewBorder(
		topSection,
		bottomSection,
		nil,
		nil,
		scrollList,
	)

	a.win.SetContent(mainLayout)
}
