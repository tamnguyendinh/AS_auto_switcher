package ui

import (
	"fmt"
	"image/color"
	"time"
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

// minSizeLayout enforces a minimum size (1200x750) while allowing responsive expansion.
type minSizeLayout struct {
	min fyne.Size
}

func (m *minSizeLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	var maxW, maxH float32 = m.min.Width, m.min.Height
	for _, o := range objects {
		if !o.Visible() {
			continue
		}
		s := o.MinSize()
		if s.Width > maxW {
			maxW = s.Width
		}
		if s.Height > maxH {
			maxH = s.Height
		}
	}
	return fyne.NewSize(maxW, maxH)
}

func (m *minSizeLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objects {
		o.Resize(size)
		o.Move(fyne.NewPos(0, 0))
	}
}

// setupLayout builds the standard 1440x900 desktop layout with 3-row TopSection and 3-column account grid.
func (a *App) setupLayout() {
	palette := GetThemePalette(a.currentTheme)

	// -------------------------------------------------------------------------
	// TopSection Row 1 (Header): Monogram [AS] + App Title
	// -------------------------------------------------------------------------
	monogramBg := canvas.NewRectangle(palette.Primary)
	monogramBg.CornerRadius = 6
	monogramText := canvas.NewText("AS", color.White)
	monogramText.TextStyle = fyne.TextStyle{Bold: true}
	monogramText.TextSize = 13
	monogramText.Alignment = fyne.TextAlignCenter
	monogramStack := container.NewStack(monogramBg, container.NewCenter(monogramText))
	monogramWrap := container.NewGridWrap(fyne.NewSize(28, 28), monogramStack)

	titleLabel := widget.NewLabelWithStyle("Antigravity Multi-Account Switcher", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	row1Header := container.NewHBox(monogramWrap, titleLabel)

	// -------------------------------------------------------------------------
	// TopSection Row 2 (Sub-header): Language Server info (left) + Theme Select (right)
	// -------------------------------------------------------------------------
	a.serverStatusLabel = widget.NewLabelWithStyle("Language Server : Đang kiểm tra...", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

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

	themeLabel := widget.NewLabel("Theme")
	themeBox := container.NewHBox(themeLabel, a.themeSelect)
	row2SubHeader := container.NewBorder(nil, nil, a.serverStatusLabel, themeBox)

	// -------------------------------------------------------------------------
	// TopSection Row 3 (Toolbar): Action buttons (left) + Timestamp (right)
	// -------------------------------------------------------------------------
	a.refreshBtn = widget.NewButtonWithIcon("🔄 Làm mới Chrome", theme.ViewRefreshIcon(), func() {
		a.Refresh()
	})

	a.quotaRefreshAllBtn = widget.NewButtonWithIcon("🔄 Làm mới Quota", theme.ViewRefreshIcon(), func() {
		a.RefreshAllQuota()
	})
	a.quotaRefreshAllBtn.Importance = widget.HighImportance

	pinLabel := "📌 Ghim : Bật"
	pinImportance := widget.HighImportance
	if !a.alwaysOnTop {
		pinLabel = "📌 Ghim : Tắt"
		pinImportance = widget.MediumImportance
	}
	a.alwaysOnTopBtn = widget.NewButton(pinLabel, func() {
		a.ToggleAlwaysOnTop()
	})
	a.alwaysOnTopBtn.Importance = pinImportance

	nowStr := time.Now().Format("15:04:05")
	a.lastUpdatedLabel = widget.NewLabelWithStyle(fmt.Sprintf("Cập nhật : %s", nowStr), fyne.TextAlignTrailing, fyne.TextStyle{Italic: true})
	a.autoRefreshLabel = a.lastUpdatedLabel

	toolbarLeft := container.NewHBox(a.refreshBtn, a.quotaRefreshAllBtn, a.alwaysOnTopBtn)
	row3Toolbar := container.NewBorder(nil, nil, toolbarLeft, a.lastUpdatedLabel)

	// Separator dividing header controls from accounts grid
	sep := widget.NewSeparator()

	topSection := container.NewVBox(
		row1Header,
		row2SubHeader,
		row3Toolbar,
		sep,
	)

	// -------------------------------------------------------------------------
	// Center Section: 3-Column Account Grid inside Vertical Scroller
	// -------------------------------------------------------------------------
	a.accountListContainer = container.NewGridWithColumns(3)
	scrollList := container.NewVScroll(container.NewPadded(a.accountListContainer))

	// In-memory labels for backward-compatibility with tests & background workers
		a.headerBg = canvas.NewRectangle(palette.Card)
	a.statusBg = canvas.NewRectangle(palette.Card)
	a.legendBg = canvas.NewRectangle(palette.Card)
	a.statusBarLabel = widget.NewLabelWithStyle("Sẵn sàng", fyne.TextAlignLeading, fyne.TextStyle{Italic: true})
	a.activeStatusLabel = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	a.activeQuotaLabel = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Italic: true})

	// Main Layout: No footer status bar, no legend bar, no bulky header card
	mainLayout := container.NewBorder(
		topSection,
		nil,
		nil,
		nil,
		scrollList,
	)

	// Root wrapped with minSizeLayout to ensure window min size is at least 1200x750
	rootContainer := container.New(&minSizeLayout{min: fyne.NewSize(1200, 750)}, mainLayout)
	a.win.SetContent(rootContainer)
}

