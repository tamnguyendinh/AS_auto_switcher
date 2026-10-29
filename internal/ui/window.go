package ui

import (
	"image/color"
	"math"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"antigravity-switch/internal/core/i18n"
	"antigravity-switch/internal/core/version"

	"golang.org/x/sys/windows"
)

const windowTitle = "⚡ Auto Switcher"

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
	DefaultWindowWidth  float32 = 1100.0 // Mở rộng từ 980.0 lên 1100.0 theo Phase 1A (Khai phóng lưới 3 cột card)
	DefaultWindowHeight float32 = 630.0
	MinWindowWidth      float32 = 640.0
	MinWindowHeight     float32 = 360.0
)

// newMainWindow lets the desktop driver own the borderless state so its client
// size, OpenGL viewport, and pointer coordinates stay in the same coordinate space.
func newMainWindow(app fyne.App) fyne.Window {
	if driver, ok := app.Driver().(desktop.Driver); ok {
		win := driver.CreateSplashWindow()
		win.SetTitle(windowTitle)
		win.SetPadded(true) // Preserve the normal window's content padding.
		return win
	}
	// The headless Fyne test driver has no native window decorations.
	return app.NewWindow(windowTitle)
}

// setupLayout builds the compact desktop layout with a reflowing fixed-card grid.
func (a *App) setupLayout() {
	palette := GetThemePalette(a.currentTheme)

	// -------------------------------------------------------------------------
	// TopSection Row 1 (Header): Monogram [AS] + App Title + Drag Area + Window Controls (_ □ X)
	// -------------------------------------------------------------------------
	a.monogramBg = canvas.NewRectangle(palette.Primary)
	a.monogramBg.CornerRadius = 6
	monogramText := canvas.NewText("AS", color.White)
	monogramText.TextStyle = fyne.TextStyle{Bold: true}
	monogramText.TextSize = 13
	monogramText.Alignment = fyne.TextAlignCenter
	monogramStack := container.NewStack(a.monogramBg, container.NewCenter(monogramText))
	monogramWrap := container.NewGridWrap(fyne.NewSize(26, 26), monogramStack)

	a.titleLabel = widget.NewRichText(&widget.TextSegment{
		Text: i18n.Current().AppTitle,
		Style: widget.RichTextStyle{
			SizeName:  titleTextSizeName,
			TextStyle: fyne.TextStyle{Bold: true},
		},
	})
	titleLeft := container.NewHBox(monogramWrap, a.titleLabel)

	// Standard window controls: [Dropdown VN/EN] -> [— Thu nhỏ] -> [□ Phóng to] -> [✕ Đóng]
	a.langSelect = widget.NewSelect([]string{"VN", "EN"}, func(selected string) {
		targetLocale := "vi"
		if selected == "EN" {
			targetLocale = "en"
		}
		a.SetLanguage(targetLocale)
	})
	if a.currentLocale == "en" {
		a.langSelect.SetSelected("EN")
	} else {
		a.langSelect.SetSelected("VN")
	}
	langSelectWrap := container.NewGridWrap(fyne.NewSize(54, 22), a.langSelect)

	btnMin := container.NewGridWrap(fyne.NewSize(32, 26), newCustomTitleBtn("—", func() {
		MinimizeNativeWindow()
	}))
	maximizeButton := newCustomTitleBtn("□", ToggleMaximizeNativeWindow)
	maximizeButton.stateLabel = func() string {
		if IsNativeWindowMaximized() {
			return "❐"
		}
		return "□"
	}
	btnMax := container.NewGridWrap(fyne.NewSize(32, 26), maximizeButton)
	btnClose := container.NewGridWrap(fyne.NewSize(32, 26), newCustomCloseBtn(func() {
		a.Close()
		a.app.Quit()
	}))
	windowControls := container.NewHBox(langSelectWrap, btnMin, btnMax, btnClose)

	dragArea := &windowDragArea{}
	dragArea.ExtendBaseWidget(dragArea)

	row1Header := container.NewBorder(nil, nil, titleLeft, windowControls, dragArea)

	// -------------------------------------------------------------------------
	// TopSection Row 2 (Combined): Toolbar Buttons (trái) + Theme Select (phải)
	// -------------------------------------------------------------------------
	d := i18n.Current()
	a.serverStatusLabel = widget.NewLabelWithStyle(d.ServerConnecting, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	a.addAccountBtn = widget.NewButtonWithIcon(d.BtnAdd, theme.ContentAddIcon(), func() {
		a.ShowAddAccountModal()
	})

	a.refreshBtn = widget.NewButton(d.BtnRefreshList, func() {
		a.Refresh()
	})

	a.quotaRefreshAllBtn = widget.NewButton(d.BtnRefreshQuota, func() {
		a.RefreshAllQuota()
	})
	a.quotaRefreshAllBtn.Importance = widget.HighImportance

	pinImportance := widget.HighImportance
	if !a.alwaysOnTop {
		pinImportance = widget.MediumImportance
	}
	a.alwaysOnTopBtn = widget.NewButton("📌", func() {
		a.ToggleAlwaysOnTop()
	})
	a.alwaysOnTopBtn.Importance = pinImportance

	toolbarLeft := container.NewHBox(
		compactButton(a.addAccountBtn),
		compactButton(a.refreshBtn),
		compactButton(a.quotaRefreshAllBtn),
		compactButton(a.alwaysOnTopBtn),
	)

	// row2LeftCluster chỉ còn chứa toolbarLeft, đặt ngay đầu góc trái
	row2LeftCluster := toolbarLeft

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

	themeSelectWrap := container.NewGridWrap(fyne.NewSize(58, 20), a.themeSelect)

	// -------------------------------------------------------------------------
	// Khối 1: Label & Capsule Toggle Ẩn tài khoản (SPEC-ARCH-20261020-ACCOUNT-PRIVACY-MASKING-TOOLBAR)
	// -------------------------------------------------------------------------
	lblMask := widget.NewLabelWithStyle(d.LblMaskAccounts, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	headerRow2MaskLabel = lblMask

	a.maskAccountsToggle = newCapsuleToggle(a.maskAccounts, func(isOn bool) {
		a.SetMaskAccounts(isOn)
	}, palette)

	// Divider 1 (Bắt buộc bọc GridWrap 1x14 px để không bị MinSize 0x0)
	a.row2Divider1 = canvas.NewRectangle(palette.BorderSubtle)
	div1Wrap := container.NewGridWrap(fyne.NewSize(1, 14), a.row2Divider1)

	// -------------------------------------------------------------------------
	// Khối 2: Label & Auto Switch Toggle Capsule Widget (SPEC-ARCH-20261020-AUTO-SWITCH-TOGGLE-BUTTON)
	// SPEC-ARCH-20261020-ALIGN-HEADER-ROW2-VERTICAL-BASELINE: widget.Label with Bold style
	// ĐÃ CÓ - GIỮ NGUYÊN HOÀN TOÀN KHÔNG ĐỤNG VÀO NÚT GẠT TỰ ĐỘNG
	// -------------------------------------------------------------------------
	lblAuto := widget.NewLabelWithStyle(d.LblAutoSwitch, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	headerRow2AutoLabel = lblAuto

	a.autoSwitchToggle = newAutoSwitchToggle(a.autoSwitchEnabled, func(isOn bool) {
		a.SetAutoSwitchEnabled(isOn)
	}, palette)

	// Divider 2 (Bắt buộc bọc GridWrap 1x14 px để không bị MinSize 0x0)
	a.row2Divider2 = canvas.NewRectangle(palette.BorderSubtle)
	div2Wrap := container.NewGridWrap(fyne.NewSize(1, 14), a.row2Divider2)

	// Right Cluster: Bố cục 7 phần tử Canvas
	// [Ẩn tài khoản: ○] │ [Tự động: ●] │ [ ☀️ ▼ ]
	rightCluster := container.New(row2RightClusterLayout{},
		lblMask,
		a.maskAccountsToggle,
		div1Wrap,
		lblAuto,
		a.autoSwitchToggle,
		div2Wrap,
		themeSelectWrap,
	)
	row2Combined := container.NewBorder(nil, nil, row2LeftCluster, rightCluster)
	a.toolbar = row2Combined

	// Separator chia phần Top Bar với thân app
	sep := widget.NewSeparator()
	topSection := container.NewVBox(row1Header, sep)
	a.topSection = topSection

	// -------------------------------------------------------------------------
	// Sidebar (136px - 80%) bên trái (Giai đoạn 1A: Duy nhất tab Antigravity)
	// -------------------------------------------------------------------------
	a.sidebar = newSidebarContainer(palette, func(selectedTab NavTab) {
		a.SwitchTab(selectedTab)
	})

	// -------------------------------------------------------------------------
	// Center Section: 280x130 Fixed Card Grid inside Vertical Scroller
	// -------------------------------------------------------------------------
	a.accountListContainer = newSymmetricGridWrap(fyne.NewSize(280, 130))
	scrollList := container.NewVScroll(container.NewPadded(a.accountListContainer))
	a.accountScroll = scrollList

	// Khung làm việc Content Area của Antigravity (Toolbar hàng 2 + Lưới Card cuộn dọc)
	a.antigravityContent = container.NewBorder(
		row2Combined,
		nil,
		nil,
		nil,
		scrollList,
	)
	a.codex = newCodexView(palette, a.ToggleAlwaysOnTop)
	a.codex.content.Hide()
	a.syncAlwaysOnTopBtn(a.alwaysOnTop)
	contentStack := container.NewStack(a.antigravityContent, a.codex.content)

	// Thân app: Sidebar bên trái + Content Area bên phải
	bodyLayout := container.NewBorder(
		nil,
		nil,
		a.sidebar,
		nil,
		contentStack,
	)

	// Status Bar: transient feedback for loading/success/error
	a.statusBar = widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})
	// Long capture/path errors must not increase the window's minimum width.
	a.statusBar.Truncation = fyne.TextTruncateEllipsis
	a.statusBar.Hide()

	// Version Label: Mép phải thanh đáy
	a.versionLabel = widget.NewLabelWithStyle(version.GetFullVersion(), fyne.TextAlignTrailing, fyne.TextStyle{})
	a.codexStatusLabel = widget.NewLabelWithStyle(d.CodexStatus, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	a.codexVersionLabel = widget.NewLabelWithStyle(d.CodexVersion, fyne.TextAlignTrailing, fyne.TextStyle{})
	a.leftFooter = container.NewStack(a.serverStatusLabel, a.codexStatusLabel)
	a.rightFooter = container.NewStack(a.versionLabel, a.codexVersionLabel)
	a.renderFooter()

	// Khung chứa nội dung thanh đáy: Left (Server Status), Center (Status Bar), Right (Version Label)
	footerContent := container.NewBorder(
		nil, nil,
		a.leftFooter,  // Contextual left edge
		a.rightFooter, // Contextual right edge
		a.statusBar,   // Trung tâm
	)

	// SPEC-ARCH-20261024-FOOTER-HEIGHT-REDUCTION: Compact Footer Bar (giảm 50% độ cao xuống chuẩn ~21-22px)
	padLeft := canvas.NewRectangle(color.Transparent)
	padLeft.SetMinSize(fyne.NewSize(8, 1))
	padRight := canvas.NewRectangle(color.Transparent)
	padRight.SetMinSize(fyne.NewSize(8, 1))

	paddedFooter := container.NewBorder(nil, nil, padLeft, padRight, footerContent)

	bottomSection := container.New(
		&compactFooterLayout{},
		widget.NewSeparator(),
		paddedFooter,
	)

	// Main Layout: Top Bar (Row 1 Header + Sep) | Bottom Section (Footer) | Center (Body Layout)
	mainLayout := container.NewBorder(
		topSection,
		bottomSection,
		nil,
		nil,
		bodyLayout,
	)

	addWindowResizeFrame(a.win, mainLayout)
	a.win.SetContent(mainLayout)
}

// customTitleBtn: Widget nút thu nhỏ/phóng to có hover sáng
type customTitleBtn struct {
	widget.BaseWidget
	bg         *canvas.Rectangle
	text       *canvas.Text
	onTapped   func()
	stateLabel func() string
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

func (b *customTitleBtn) Refresh() {
	if b.stateLabel != nil {
		b.text.Text = b.stateLabel()
	}
	b.BaseWidget.Refresh()
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
	b.Refresh()
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

// -----------------------------------------------------------------------------
// Symmetric GridWrap Layout (SPEC-ARCH-20261019-SYMMETRIC-GRID-WRAP-MARGINS)
// -----------------------------------------------------------------------------

var _ fyne.Layout = (*symmetricGridWrapLayout)(nil)

type symmetricGridWrapLayout struct {
	CellSize fyne.Size
	colCount int
	rowCount int
}

func newSymmetricGridWrap(cellSize fyne.Size, objects ...fyne.CanvasObject) *fyne.Container {
	return container.New(&symmetricGridWrapLayout{CellSize: cellSize, colCount: 1, rowCount: 1}, objects...)
}

func (g *symmetricGridWrapLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	padding := theme.Padding()
	g.colCount = 1
	g.rowCount = 0

	if size.Width > g.CellSize.Width {
		g.colCount = int(math.Floor(float64(size.Width+padding) / float64(g.CellSize.Width+padding)))
	}
	if g.colCount < 1 {
		g.colCount = 1
	}

	totalCardsW := float32(g.colCount) * g.CellSize.Width
	var startX, gapX float32

	if size.Width > totalCardsW && g.colCount > 0 {
		extra := size.Width - totalCardsW
		gapX = extra / float32(g.colCount+1)
		if gapX < padding {
			gapX = padding
			startX = (size.Width - (totalCardsW + float32(g.colCount-1)*gapX)) / 2.0
		} else {
			startX = gapX
		}
	} else {
		gapX = padding
		startX = 0
	}

	gapY := padding
	i, x, y := 0, startX, float32(0)
	for _, child := range objects {
		if !child.Visible() {
			continue
		}

		if i%g.colCount == 0 {
			g.rowCount++
		}

		child.Move(fyne.NewPos(x, y))
		child.Resize(g.CellSize)

		if (i+1)%g.colCount == 0 {
			x = startX
			y += g.CellSize.Height + gapY
		} else {
			x += g.CellSize.Width + gapX
		}
		i++
	}
}

func (g *symmetricGridWrapLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	rows := g.rowCount
	if rows < 1 {
		rows = 1
	}
	return fyne.NewSize(g.CellSize.Width,
		(g.CellSize.Height*float32(rows))+(float32(rows-1)*theme.Padding()))
}

// compactFooterLayout delivers a 50% reduced compact height footer bar (~22px)
// with precise pixel placement and zero excessive vertical padding.
type compactFooterLayout struct{}

var _ fyne.Layout = (*compactFooterLayout)(nil)

func (l *compactFooterLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	var maxW float32
	for _, o := range objects {
		if o.Visible() {
			if w := o.MinSize().Width; w > maxW {
				maxW = w
			}
		}
	}
	// Height 22px: Separator 1px + Content 21px (50% reduction from ~48px)
	return fyne.NewSize(maxW, 22)
}

func (l *compactFooterLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 2 {
		return
	}
	sep := objects[0]
	footer := objects[1]

	sepHeight := float32(1)
	sep.Move(fyne.NewPos(0, 0))
	sep.Resize(fyne.NewSize(size.Width, sepHeight))

	footerHeight := size.Height - sepHeight
	if footerHeight < 0 {
		footerHeight = 0
	}
	footer.Move(fyne.NewPos(0, sepHeight))
	footer.Resize(fyne.NewSize(size.Width, footerHeight))
}

// -----------------------------------------------------------------------------
// Auto Switch Toggle Widget (SPEC-ARCH-20261020-AUTO-SWITCH-TOGGLE-BUTTON)
// -----------------------------------------------------------------------------

// AutoSwitchToggle represents a capsule pill toggle switch.
type AutoSwitchToggle struct {
	widget.BaseWidget
	isOn     bool
	onChange func(bool)
	palette  ThemePalette

	track *canvas.Rectangle
	thumb *canvas.Circle
}

var (
	_ fyne.Widget        = (*AutoSwitchToggle)(nil)
	_ fyne.Tappable      = (*AutoSwitchToggle)(nil)
	_ desktop.Hoverable  = (*AutoSwitchToggle)(nil)
	_ desktop.Cursorable = (*AutoSwitchToggle)(nil)
)

func newAutoSwitchToggle(initial bool, onChange func(bool), palette ThemePalette) *AutoSwitchToggle {
	t := &AutoSwitchToggle{
		isOn:     initial,
		onChange: onChange,
		palette:  palette,
	}

	t.track = canvas.NewRectangle(palette.BorderSubtle)
	t.track.CornerRadius = 6.0
	t.track.Resize(fyne.NewSize(23, 12))
	t.track.Move(fyne.NewPos(0, 0))

	t.thumb = canvas.NewCircle(color.White)
	t.thumb.Resize(fyne.NewSize(8, 8))

	if initial {
		t.track.FillColor = palette.Primary
		t.thumb.Move(fyne.NewPos(13, 2))
	} else {
		t.track.FillColor = palette.BorderSubtle
		t.thumb.Move(fyne.NewPos(2, 2))
	}

	t.ExtendBaseWidget(t)
	return t
}

func (t *AutoSwitchToggle) CreateRenderer() fyne.WidgetRenderer {
	t.ExtendBaseWidget(t)
	return &autoSwitchToggleRenderer{toggle: t}
}

func (t *AutoSwitchToggle) MinSize() fyne.Size {
	t.ExtendBaseWidget(t)
	return fyne.NewSize(23, 12)
}

func (t *AutoSwitchToggle) Tapped(_ *fyne.PointEvent) {
	t.isOn = !t.isOn
	t.updateVisuals()
	if t.onChange != nil {
		t.onChange(t.isOn)
	}
}

func (t *AutoSwitchToggle) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

func (t *AutoSwitchToggle) MouseIn(*desktop.MouseEvent)    {}
func (t *AutoSwitchToggle) MouseOut()                      {}
func (t *AutoSwitchToggle) MouseMoved(*desktop.MouseEvent) {}

func (t *AutoSwitchToggle) IsOn() bool {
	return t.isOn
}

func (t *AutoSwitchToggle) SetOn(on bool) {
	if t.isOn == on {
		return
	}
	t.isOn = on
	t.updateVisuals()
	if t.onChange != nil {
		t.onChange(t.isOn)
	}
}

func (t *AutoSwitchToggle) SetPalette(palette ThemePalette) {
	t.palette = palette
	t.updateVisuals()
}

func (t *AutoSwitchToggle) updateVisuals() {
	if t.isOn {
		t.track.FillColor = t.palette.Primary
	} else {
		t.track.FillColor = t.palette.BorderSubtle
	}
	y := (t.Size().Height - 12.0) / 2.0
	if y < 0 {
		y = 0
	}
	thumbY := y + 2.0
	if t.isOn {
		t.thumb.Move(fyne.NewPos(13, thumbY))
	} else {
		t.thumb.Move(fyne.NewPos(2, thumbY))
	}
	t.track.Refresh()
	t.thumb.Refresh()
	t.Refresh()
}

type autoSwitchToggleRenderer struct {
	toggle *AutoSwitchToggle
}

func (r *autoSwitchToggleRenderer) MinSize() fyne.Size {
	return fyne.NewSize(23, 12)
}

func (r *autoSwitchToggleRenderer) Layout(size fyne.Size) {
	trackW := float32(23.0)
	trackH := float32(12.0)

	// Căn giữa theo chiều cao thực tế được phân bổ cho widget
	y := (size.Height - trackH) / 2.0
	if y < 0 {
		y = 0
	}

	r.toggle.track.Move(fyne.NewPos(0, y))
	r.toggle.track.Resize(fyne.NewSize(trackW, trackH))

	thumbY := y + 2.0
	if r.toggle.isOn {
		r.toggle.thumb.Move(fyne.NewPos(13, thumbY))
	} else {
		r.toggle.thumb.Move(fyne.NewPos(2, thumbY))
	}
	r.toggle.thumb.Resize(fyne.NewSize(8, 8))
}

func (r *autoSwitchToggleRenderer) Refresh() {
	if r.toggle.isOn {
		r.toggle.track.FillColor = r.toggle.palette.Primary
	} else {
		r.toggle.track.FillColor = r.toggle.palette.BorderSubtle
	}
	r.toggle.track.Refresh()
	r.toggle.thumb.Refresh()
	r.Layout(r.toggle.Size())
}

func (r *autoSwitchToggleRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.toggle.track, r.toggle.thumb}
}

func (r *autoSwitchToggleRenderer) Destroy() {}

// -----------------------------------------------------------------------------
// Row 2 Right Cluster Layout (SPEC-ARCH-20261020-ALIGN-HEADER-ROW2-VERTICAL-BASELINE)
// -----------------------------------------------------------------------------

var headerRow2AutoLabel *widget.Label

// AutoSwitchLabelWidget returns the header row 2 auto switch label widget.
func (a *App) AutoSwitchLabelWidget() *widget.Label {
	return headerRow2AutoLabel
}

type row2RightClusterLayout struct{}

func (row2RightClusterLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	w, h := float32(0), float32(0)
	padding := theme.Padding()
	for i, o := range objects {
		if !o.Visible() {
			continue
		}
		ms := o.MinSize()
		w += ms.Width
		if i > 0 {
			w += padding
		}
		if ms.Height > h {
			h = ms.Height
		}
	}
	return fyne.NewSize(w, h)
}

func (row2RightClusterLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	x := float32(0)
	padding := theme.Padding()
	for _, o := range objects {
		if !o.Visible() {
			continue
		}
		ms := o.MinSize()
		y := (size.Height - ms.Height) / 2.0
		if y < 0 {
			y = 0
		}
		o.Move(fyne.NewPos(x, y))
		o.Resize(ms)
		x += ms.Width + padding
	}
}

// -----------------------------------------------------------------------------
// Mask Accounts Toggle Widget (SPEC-ARCH-20261020-ACCOUNT-PRIVACY-MASKING-TOOLBAR)
// -----------------------------------------------------------------------------

// MaskAccountsToggle represents an independent 23x12 capsule pill toggle switch for account privacy masking.
type MaskAccountsToggle struct {
	widget.BaseWidget
	isOn     bool
	onChange func(bool)
	palette  ThemePalette

	track *canvas.Rectangle
	thumb *canvas.Circle
}

// CapsuleToggle is an alias to MaskAccountsToggle for the standardized capsule toggle specification.
type CapsuleToggle = MaskAccountsToggle

var (
	_ fyne.Widget        = (*MaskAccountsToggle)(nil)
	_ fyne.Tappable      = (*MaskAccountsToggle)(nil)
	_ desktop.Hoverable  = (*MaskAccountsToggle)(nil)
	_ desktop.Cursorable = (*MaskAccountsToggle)(nil)
)

func newMaskAccountsToggle(initial bool, onChange func(bool), palette ThemePalette) *MaskAccountsToggle {
	t := &MaskAccountsToggle{
		isOn:     initial,
		onChange: onChange,
		palette:  palette,
	}

	t.track = canvas.NewRectangle(palette.BorderSubtle)
	t.track.CornerRadius = 6.0
	t.track.Resize(fyne.NewSize(23, 12))
	t.track.Move(fyne.NewPos(0, 0))

	t.thumb = canvas.NewCircle(color.White)
	t.thumb.Resize(fyne.NewSize(8, 8))

	if initial {
		t.track.FillColor = palette.Primary
		t.thumb.Move(fyne.NewPos(13, 2))
	} else {
		t.track.FillColor = palette.BorderSubtle
		t.thumb.Move(fyne.NewPos(2, 2))
	}

	t.ExtendBaseWidget(t)
	return t
}

func newCapsuleToggle(initial bool, onChange func(bool), palette ThemePalette) *CapsuleToggle {
	return newMaskAccountsToggle(initial, onChange, palette)
}

func (t *MaskAccountsToggle) CreateRenderer() fyne.WidgetRenderer {
	t.ExtendBaseWidget(t)
	return &maskAccountsToggleRenderer{toggle: t}
}

func (t *MaskAccountsToggle) MinSize() fyne.Size {
	t.ExtendBaseWidget(t)
	return fyne.NewSize(23, 12)
}

func (t *MaskAccountsToggle) Tapped(_ *fyne.PointEvent) {
	t.isOn = !t.isOn
	t.updateVisuals()
	if t.onChange != nil {
		t.onChange(t.isOn)
	}
}

func (t *MaskAccountsToggle) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

func (t *MaskAccountsToggle) MouseIn(*desktop.MouseEvent)    {}
func (t *MaskAccountsToggle) MouseOut()                      {}
func (t *MaskAccountsToggle) MouseMoved(*desktop.MouseEvent) {}

func (t *MaskAccountsToggle) IsOn() bool {
	return t.isOn
}

func (t *MaskAccountsToggle) SetOn(on bool) {
	if t.isOn == on {
		return
	}
	t.isOn = on
	t.updateVisuals()
	if t.onChange != nil {
		t.onChange(t.isOn)
	}
}

func (t *MaskAccountsToggle) SetPalette(palette ThemePalette) {
	t.palette = palette
	t.updateVisuals()
}

func (t *MaskAccountsToggle) updateVisuals() {
	if t.isOn {
		t.track.FillColor = t.palette.Primary
	} else {
		t.track.FillColor = t.palette.BorderSubtle
	}
	y := (t.Size().Height - 12.0) / 2.0
	if y < 0 {
		y = 0
	}
	thumbY := y + 2.0
	if t.isOn {
		t.thumb.Move(fyne.NewPos(13, thumbY))
	} else {
		t.thumb.Move(fyne.NewPos(2, thumbY))
	}
	t.track.Refresh()
	t.thumb.Refresh()
	t.Refresh()
}

type maskAccountsToggleRenderer struct {
	toggle *MaskAccountsToggle
}

func (r *maskAccountsToggleRenderer) MinSize() fyne.Size {
	return fyne.NewSize(23, 12)
}

func (r *maskAccountsToggleRenderer) Layout(size fyne.Size) {
	trackW := float32(23.0)
	trackH := float32(12.0)

	y := (size.Height - trackH) / 2.0
	if y < 0 {
		y = 0
	}

	r.toggle.track.Move(fyne.NewPos(0, y))
	r.toggle.track.Resize(fyne.NewSize(trackW, trackH))

	thumbY := y + 2.0
	if r.toggle.isOn {
		r.toggle.thumb.Move(fyne.NewPos(13, thumbY))
	} else {
		r.toggle.thumb.Move(fyne.NewPos(2, thumbY))
	}
	r.toggle.thumb.Resize(fyne.NewSize(8, 8))
}

func (r *maskAccountsToggleRenderer) Refresh() {
	if r.toggle.isOn {
		r.toggle.track.FillColor = r.toggle.palette.Primary
	} else {
		r.toggle.track.FillColor = r.toggle.palette.BorderSubtle
	}
	r.toggle.track.Refresh()
	r.toggle.thumb.Refresh()
	r.Layout(r.toggle.Size())
}

func (r *maskAccountsToggleRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.toggle.track, r.toggle.thumb}
}

func (r *maskAccountsToggleRenderer) Destroy() {}

var headerRow2MaskLabel *widget.Label

// MaskAccountsLabelWidget returns the header row 2 mask accounts label widget.
func (a *App) MaskAccountsLabelWidget() *widget.Label {
	return headerRow2MaskLabel
}

// MaskAccountsToggleWidget returns the header row 2 mask accounts capsule toggle.
func (a *App) MaskAccountsToggleWidget() *CapsuleToggle {
	return a.maskAccountsToggle
}

// Row2Dividers returns the two vertical divider canvas rectangles.
func (a *App) Row2Dividers() (*canvas.Rectangle, *canvas.Rectangle) {
	return a.row2Divider1, a.row2Divider2
}
