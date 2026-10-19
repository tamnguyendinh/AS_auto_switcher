package ui

import (
	"image/color"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
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

const (
	DefaultWindowWidth  float32 = 1008.0
	DefaultWindowHeight float32 = 630.0
	MinWindowWidth      float32 = 480.0
	MinWindowHeight     float32 = 360.0
)

// setupLayout builds the standard 1008x630 desktop layout with 3-row TopSection and 3-column account grid.
func (a *App) setupLayout() {
	palette := GetThemePalette(a.currentTheme)

	// -------------------------------------------------------------------------
	// TopSection Row 1 (Header): Monogram [AS] + App Title + Drag Area + Window Controls (_ X)
	// -------------------------------------------------------------------------
	a.monogramBg = canvas.NewRectangle(palette.Primary)
	a.monogramBg.CornerRadius = 6
	monogramText := canvas.NewText("AS", color.White)
	monogramText.TextStyle = fyne.TextStyle{Bold: true}
	monogramText.TextSize = 13
	monogramText.Alignment = fyne.TextAlignCenter
	monogramStack := container.NewStack(a.monogramBg, container.NewCenter(monogramText))
	monogramWrap := container.NewGridWrap(fyne.NewSize(26, 26), monogramStack)

	titleLabel := widget.NewLabelWithStyle("Antigravity Multi-Account Switcher", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	titleLeft := container.NewHBox(monogramWrap, titleLabel)

	// Cụm nút điều khiển cửa sổ cố định (chỉ thu nhỏ Taskbar và đóng)
	btnMin := container.NewGridWrap(fyne.NewSize(32, 26), newCustomTitleBtn("—", func() {
		MinimizeNativeWindow()
	}))
	btnClose := container.NewGridWrap(fyne.NewSize(32, 26), newCustomCloseBtn(func() {
		a.Close()
		a.app.Quit()
	}))
	windowControls := container.NewHBox(btnMin, btnClose)

	dragArea := &windowDragArea{}
	dragArea.ExtendBaseWidget(dragArea)

	row1Header := container.NewBorder(nil, nil, titleLeft, windowControls, dragArea)

	// -------------------------------------------------------------------------
	// TopSection Row 2 (Sub-header): Status đưa lên đầu (trái) + Theme Select mini (phải)
	// -------------------------------------------------------------------------
	a.serverStatusLabel = widget.NewLabelWithStyle("Hoạt động  | Đang kết nối...", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Theme Select chỉ gồm 3 icon, thu hẹp bề ngang tối đa
	themeOptions := []string{"🌙", "☀️", "☕"}
	a.themeSelect = widget.NewSelect(themeOptions, func(selected string) {
		switch selected {
		case "🌙":
			a.SetTheme(ThemeDark)
		case "☀️":
			a.SetTheme(ThemeLight)
		case "☕":
			a.SetTheme(ThemeWarm)
		}
	})

	switch a.currentTheme {
	case ThemeLight:
		a.themeSelect.SetSelected("☀️")
	case ThemeWarm:
		a.themeSelect.SetSelected("☕")
	default:
		a.themeSelect.SetSelected("🌙")
	}

	themeSelectWrap := container.NewGridWrap(fyne.NewSize(58, 32), a.themeSelect)
	row2SubHeader := container.NewBorder(nil, nil, a.serverStatusLabel, themeSelectWrap)

	// -------------------------------------------------------------------------
	// TopSection Row 3 (Toolbar): Nút phẳng (trái) + Bỏ sạch timestamp
	// -------------------------------------------------------------------------
	a.refreshBtn = widget.NewButton("Danh sách tài khoản", func() {
		a.Refresh()
	})

	a.quotaRefreshAllBtn = widget.NewButton("Làm mới Quota", func() {
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

	toolbarLeft := container.NewHBox(a.refreshBtn, a.quotaRefreshAllBtn, a.alwaysOnTopBtn)
	row3Toolbar := toolbarLeft // Phía bên phải để trống hoàn toàn thoáng đãng

	// Separator chia phần điều khiển với lưới tài khoản
	sep := widget.NewSeparator()

	topSection := container.NewVBox(
		row1Header,
		row2SubHeader,
		row3Toolbar,
		sep,
	)

	// -------------------------------------------------------------------------
	// Center Section: 318x90 Fixed Card Grid inside Vertical Scroller
	// -------------------------------------------------------------------------
	a.accountListContainer = container.NewGridWrap(fyne.NewSize(318, 90))
	scrollList := container.NewVScroll(container.NewPadded(a.accountListContainer))
	a.accountScroll = scrollList

	// Main Layout: No footer status bar, no legend bar, no bulky header card
	mainLayout := container.NewBorder(
		topSection,
		nil,
		nil,
		nil,
		scrollList,
	)

	a.win.SetContent(mainLayout)
}

// customTitleBtn: Widget nút thu nhỏ/phóng to có hover sáng
type customTitleBtn struct {
	widget.BaseWidget
	bg       *canvas.Rectangle
	text     *canvas.Text
	onTapped func()
}

func newCustomTitleBtn(label string, onTapped func()) *customTitleBtn {
	b := &customTitleBtn{onTapped: onTapped}
	b.bg = canvas.NewRectangle(color.Transparent)
	b.bg.CornerRadius = 3
	b.text = canvas.NewText(label, color.NRGBA{R: 0xcd, G: 0xd6, B: 0xf4, A: 0xff})
	b.text.TextSize = 13
	b.text.Alignment = fyne.TextAlignCenter
	b.ExtendBaseWidget(b)
	return b
}

func (b *customTitleBtn) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewStack(b.bg, container.NewCenter(b.text))
	return widget.NewSimpleRenderer(c)
}

func (b *customTitleBtn) MouseIn(*desktop.MouseEvent) {
	b.bg.FillColor = color.NRGBA{R: 0x45, G: 0x47, B: 0x5a, A: 0xaa} // Hover sáng
	b.Refresh()
}

func (b *customTitleBtn) MouseOut() {
	b.bg.FillColor = color.Transparent
	b.Refresh()
}

func (b *customTitleBtn) MouseMoved(*desktop.MouseEvent) {}

func (b *customTitleBtn) Tapped(*fyne.PointEvent) {
	if b.onTapped != nil {
		b.onTapped()
	}
}

// customCloseBtn: Nút X có hover màu đỏ rực (#ef4444)
type customCloseBtn struct {
	widget.BaseWidget
	bg       *canvas.Rectangle
	text     *canvas.Text
	onTapped func()
}

func newCustomCloseBtn(onTapped func()) *customCloseBtn {
	b := &customCloseBtn{onTapped: onTapped}
	b.bg = canvas.NewRectangle(color.Transparent)
	b.bg.CornerRadius = 3
	b.text = canvas.NewText("✕", color.NRGBA{R: 0xcd, G: 0xd6, B: 0xf4, A: 0xff})
	b.text.TextSize = 13
	b.text.Alignment = fyne.TextAlignCenter
	b.ExtendBaseWidget(b)
	return b
}

func (b *customCloseBtn) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewStack(b.bg, container.NewCenter(b.text))
	return widget.NewSimpleRenderer(c)
}

func (b *customCloseBtn) MouseIn(*desktop.MouseEvent) {
	b.bg.FillColor = color.NRGBA{R: 0xef, G: 0x44, B: 0x44, A: 0xff} // Đỏ rực khi hover
	b.text.Color = color.White
	b.Refresh()
}

func (b *customCloseBtn) MouseOut() {
	b.bg.FillColor = color.Transparent
	b.text.Color = color.NRGBA{R: 0xcd, G: 0xd6, B: 0xf4, A: 0xff}
	b.Refresh()
}

func (b *customCloseBtn) MouseMoved(*desktop.MouseEvent) {}

func (b *customCloseBtn) Tapped(*fyne.PointEvent) {
	if b.onTapped != nil {
		b.onTapped()
	}
}

// windowDragArea: Vùng kéo thả cửa sổ native khi bấm giữ chuột
type windowDragArea struct {
	widget.BaseWidget
}

func (d *windowDragArea) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(canvas.NewRectangle(color.Transparent))
}

func (d *windowDragArea) Dragged(e *fyne.DragEvent) {
	DragNativeWindow()
}

func (d *windowDragArea) DragEnd() {}