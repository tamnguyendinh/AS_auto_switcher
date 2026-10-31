package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"as-auto-switcher/internal/core/i18n"
)

// SidebarWidth định nghĩa độ rộng chuẩn của thanh Sidebar (thu gọn xuống 80% còn 136px theo Phase 1A)
const SidebarWidth float32 = 136.0

// NavTab định danh các tab điều hướng trên Sidebar
type NavTab int

const (
	NavTabAntigravity NavTab = iota
	NavTabCodex
)

var (
	_ fyne.Widget        = (*sidebarItem)(nil)
	_ fyne.Tappable      = (*sidebarItem)(nil)
	_ desktop.Hoverable  = (*sidebarItem)(nil)
	_ desktop.Cursorable = (*sidebarItem)(nil)
)

// sidebarItem đại diện cho một mục tab trên Sidebar
type sidebarItem struct {
	widget.BaseWidget
	tabID        NavTab
	iconResource fyne.Resource
	label        string
	isActive     bool
	onSelected   func(NavTab)

	// Các thành phần canvas đồ họa
	bg        *canvas.Rectangle
	activeBar *canvas.Rectangle
	iconImg   *canvas.Image
	textLabel *canvas.Text
	palette   ThemePalette
	isHovered bool
}

func newSidebarItem(tabID NavTab, icon fyne.Resource, label string, isActive bool, palette ThemePalette, onSelected func(NavTab)) *sidebarItem {
	item := &sidebarItem{
		tabID:        tabID,
		iconResource: icon,
		label:        label,
		isActive:     isActive,
		palette:      palette,
		onSelected:   onSelected,
	}
	item.initCanvas()
	item.ExtendBaseWidget(item)
	return item
}

func (s *sidebarItem) initCanvas() {
	s.bg = canvas.NewRectangle(color.Transparent)
	s.bg.CornerRadius = 6

	s.activeBar = canvas.NewRectangle(color.Transparent)
	s.activeBar.CornerRadius = 2
	s.activeBar.SetMinSize(fyne.NewSize(3, 20))

	s.iconImg = canvas.NewImageFromResource(s.iconResource)
	s.iconImg.FillMode = canvas.ImageFillContain
	s.iconImg.SetMinSize(fyne.NewSize(20, 20))

	s.textLabel = canvas.NewText(s.label, s.palette.Text)
	s.textLabel.TextSize = 13
	s.textLabel.TextStyle = fyne.TextStyle{Bold: s.isActive}

	s.applyTheme(s.palette)
}

func (s *sidebarItem) applyTheme(palette ThemePalette) {
	s.palette = palette
	if s.isActive {
		// Active: Nền highlight nhẹ, vạch mép màu Primary, chữ đậm màu chính
		s.bg.FillColor = palette.CardActive
		s.activeBar.FillColor = palette.Primary
		s.textLabel.Color = palette.Text
		s.textLabel.TextStyle = fyne.TextStyle{Bold: true}
	} else if s.isHovered {
		// Hover: Nền mờ nhẹ, vạch trong suốt, chữ màu chính
		s.bg.FillColor = color.NRGBA{R: palette.BorderSubtle.R, G: palette.BorderSubtle.G, B: palette.BorderSubtle.B, A: 0x40}
		s.activeBar.FillColor = color.Transparent
		s.textLabel.Color = palette.Text
		s.textLabel.TextStyle = fyne.TextStyle{Bold: false}
	} else {
		// Normal: Nền trong suốt, chữ màu phụ (muted)
		s.bg.FillColor = color.Transparent
		s.activeBar.FillColor = color.Transparent
		s.textLabel.Color = palette.TextMuted
		s.textLabel.TextStyle = fyne.TextStyle{Bold: false}
	}
	s.bg.Refresh()
	s.activeBar.Refresh()
	s.textLabel.Refresh()
}

func (s *sidebarItem) SetActive(active bool) {
	s.isActive = active
	s.applyTheme(s.palette)
}

func (s *sidebarItem) CreateRenderer() fyne.WidgetRenderer {
	iconWrap := container.NewGridWrap(fyne.NewSize(20, 20), s.iconImg)
	spacer := container.NewGridWrap(fyne.NewSize(4, 20), canvas.NewRectangle(color.Transparent))
	contentBox := container.NewHBox(
		container.NewGridWrap(fyne.NewSize(3, 20), s.activeBar),
		spacer,
		iconWrap,
		container.NewCenter(s.textLabel),
	)
	paddedContent := container.NewPadded(contentBox)
	stack := container.NewStack(s.bg, paddedContent)
	return widget.NewSimpleRenderer(stack)
}

func (s *sidebarItem) MouseIn(*desktop.MouseEvent) {
	s.isHovered = true
	if !s.isActive {
		s.applyTheme(s.palette)
	}
}

func (s *sidebarItem) MouseOut() {
	s.isHovered = false
	if !s.isActive {
		s.applyTheme(s.palette)
	}
}

func (s *sidebarItem) MouseMoved(*desktop.MouseEvent) {}

func (s *sidebarItem) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

func (s *sidebarItem) Tapped(*fyne.PointEvent) {
	if s.onSelected != nil {
		s.onSelected(s.tabID)
	}
}

// SidebarContainer chứa cấu trúc sidebar hoàn chỉnh
type SidebarContainer struct {
	widget.BaseWidget
	antigravityItem *sidebarItem
	codexItem       *sidebarItem
	divider         *canvas.Rectangle
	palette         ThemePalette
	container       *fyne.Container
	activeTab       NavTab
}

func newSidebarContainer(palette ThemePalette, onTabSelected func(NavTab)) *SidebarContainer {
	sb := &SidebarContainer{
		palette:   palette,
		activeTab: NavTabAntigravity,
	}
	d := i18n.Current()

	// Khởi tạo tab Antigravity (Active mặc định)
	sb.antigravityItem = newSidebarItem(
		NavTabAntigravity,
		AntigravityLogoResource,
		d.NavAntigravity,
		true, // Active
		palette,
		func(tab NavTab) {
			if onTabSelected != nil {
				onTabSelected(tab)
			} else {
				sb.SwitchTab(tab)
			}
		},
	)
	sb.codexItem = newSidebarItem(NavTabCodex, CodexLogoResource, d.NavCodex, false, palette, func(tab NavTab) {
		if onTabSelected != nil {
			onTabSelected(tab)
		} else {
			sb.SwitchTab(tab)
		}
	})

	// Đường viền ngăn cách bên phải (1px)
	sb.divider = canvas.NewRectangle(palette.BorderSubtle)
	sb.divider.SetMinSize(fyne.NewSize(1, 0))

	// Persistent navigation items share one active selection.
	itemsBox := container.NewVBox(
		container.NewPadded(sb.antigravityItem),
		container.NewPadded(sb.codexItem),
	)

	sb.container = container.NewBorder(
		nil, nil, nil, sb.divider,
		itemsBox,
	)

	sb.ExtendBaseWidget(sb)
	return sb
}

func (sb *SidebarContainer) ApplyTheme(palette ThemePalette) {
	sb.palette = palette
	sb.divider.FillColor = palette.BorderSubtle
	sb.divider.Refresh()
	if sb.antigravityItem != nil {
		sb.antigravityItem.applyTheme(palette)
	}
	if sb.codexItem != nil {
		sb.codexItem.applyTheme(palette)
	}
}

// SwitchTab changes selection only; application routing owns view visibility.
func (sb *SidebarContainer) SwitchTab(target NavTab) {
	if target != NavTabAntigravity && target != NavTabCodex {
		return
	}
	sb.activeTab = target
	sb.antigravityItem.SetActive(target == NavTabAntigravity)
	sb.codexItem.SetActive(target == NavTabCodex)
}

func (sb *SidebarContainer) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(sb.container)
}

func (sb *SidebarContainer) MinSize() fyne.Size {
	minH := float32(0)
	if sb.container != nil {
		minH = sb.container.MinSize().Height
	}
	return fyne.NewSize(SidebarWidth, minH)
}

// AntigravityItem trả về item của tab Antigravity
func (sb *SidebarContainer) AntigravityItem() *sidebarItem {
	return sb.antigravityItem
}

// ActiveTab trả về tab đang được kích hoạt
func (sb *SidebarContainer) ActiveTab() NavTab {
	return sb.activeTab
}
