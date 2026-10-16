package ui

import (
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
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
	// 1. Header Card with Brand Logo & Specular Keyline Border
	logoImg := canvas.NewImageFromResource(AppLogoResource)
	logoImg.FillMode = canvas.ImageFillContain
	logoContainer := container.NewGridWrap(fyne.NewSize(34, 34), logoImg)

	titleLabel := widget.NewLabelWithStyle(windowTitle, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	titleRow := container.NewHBox(container.NewCenter(logoContainer), titleLabel)

	a.activeDotContainer = container.NewHBox(NewStatusDot(ColorYellow, 12))
	a.activeStatusLabel = widget.NewLabelWithStyle("● Đang kiểm tra Antigravity...", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	activeHBox := container.NewHBox(a.activeDotContainer, a.activeStatusLabel)

	a.serverStatusLabel = widget.NewLabelWithStyle("● Đang kết nối Language Server...", fyne.TextAlignLeading, fyne.TextStyle{Italic: true})

	headerBox := container.NewVBox(
		titleRow,
		activeHBox,
		a.serverStatusLabel,
	)

	headerBg := canvas.NewRectangle(ColorCard)
	headerBg.StrokeColor = ColorBorderSubtle
	headerBg.StrokeWidth = 1
	headerBg.CornerRadius = 8
	a.headerCard = container.NewStack(headerBg, container.NewPadded(headerBox))

	// 2. Action Controls Bar
	a.refreshBtn = widget.NewButtonWithIcon("🔄 Làm mới Chrome", theme.ViewRefreshIcon(), func() {
		a.Refresh()
	})
	a.saveBtn = widget.NewButtonWithIcon("💾 Lưu phiên", theme.DocumentSaveIcon(), func() {
		a.SaveSession()
	})

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

	ctrlGrid := container.NewGridWithColumns(3, a.refreshBtn, a.saveBtn, a.alwaysOnTopBtn)

	// 3. Legend / Status Info Bar with Specular Keyline Border
	legendLabel := widget.NewLabelWithStyle("🟢 Sẵn sàng chuyển đổi ngầm   |   🟡 Chưa có file JSON", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})
	legendBg := canvas.NewRectangle(ColorCard)
	legendBg.StrokeColor = ColorBorderSubtle
	legendBg.StrokeWidth = 1
	legendBg.CornerRadius = 6
	legendCard := container.NewStack(legendBg, container.NewPadded(legendLabel))

	topSection := container.NewVBox(
		container.NewPadded(a.headerCard),
		ctrlGrid,
		legendCard,
	)

	// 4. Scrollable Account List
	a.accountListContainer = container.NewVBox()
	scrollList := container.NewVScroll(a.accountListContainer)

	// 5. Status Bar Footer with Specular Keyline Border
	a.statusBarLabel = widget.NewLabelWithStyle("Sẵn sàng", fyne.TextAlignLeading, fyne.TextStyle{Italic: true})
	statusBg := canvas.NewRectangle(ColorCard)
	statusBg.StrokeColor = ColorBorderSubtle
	statusBg.StrokeWidth = 1
	statusBg.CornerRadius = 6
	bottomSection := container.NewStack(statusBg, container.NewPadded(a.statusBarLabel))

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
