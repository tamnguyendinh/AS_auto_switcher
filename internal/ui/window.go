package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// setupLayout builds the main compact 440x560 window layout.
func (a *App) setupLayout() {
	// 1. Header Card
	titleLabel := widget.NewLabelWithStyle("⚡ Antigravity Multi-Account Switcher", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	a.activeDotContainer = container.NewHBox(NewStatusDot(ColorYellow, 12))
	a.activeStatusLabel = widget.NewLabelWithStyle("● Đang kiểm tra Antigravity...", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	activeHBox := container.NewHBox(a.activeDotContainer, a.activeStatusLabel)

	a.serverStatusLabel = widget.NewLabelWithStyle("● Đang kết nối Language Server...", fyne.TextAlignLeading, fyne.TextStyle{Italic: true})

	headerBox := container.NewVBox(
		titleLabel,
		activeHBox,
		a.serverStatusLabel,
	)

	headerBg := canvas.NewRectangle(ColorCard)
	headerBg.CornerRadius = 8
	a.headerCard = container.NewStack(headerBg, container.NewPadded(headerBox))

	// 2. Action Controls Bar
	a.refreshBtn = widget.NewButtonWithIcon("🔄 Làm mới Chrome", theme.ViewRefreshIcon(), func() {
		a.Refresh()
	})
	a.saveBtn = widget.NewButtonWithIcon("💾 Lưu phiên", theme.DocumentSaveIcon(), func() {
		a.SaveSession()
	})
	a.failoverBtn = widget.NewButton("⚡ Auto-Failover: BẬT", func() {
		a.ToggleAutoFailover()
	})
	a.failoverBtn.Importance = widget.HighImportance

	ctrlGrid := container.NewGridWithColumns(3, a.refreshBtn, a.saveBtn, a.failoverBtn)

	// 3. Legend / Status Info Bar
	legendLabel := widget.NewLabelWithStyle("🟢 Sẵn sàng chuyển đổi ngầm   |   🟡 Chưa có file JSON", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})
	legendBg := canvas.NewRectangle(ColorCard)
	legendBg.CornerRadius = 4
	legendCard := container.NewStack(legendBg, container.NewPadded(legendLabel))

	topSection := container.NewVBox(
		container.NewPadded(a.headerCard),
		ctrlGrid,
		legendCard,
	)

	// 4. Scrollable Account List
	a.accountListContainer = container.NewVBox()
	scrollList := container.NewVScroll(a.accountListContainer)

	// 5. Status Bar Footer
	a.statusBarLabel = widget.NewLabelWithStyle("Sẵn sàng", fyne.TextAlignLeading, fyne.TextStyle{Italic: true})
	statusBg := canvas.NewRectangle(ColorCard)
	statusBg.CornerRadius = 4
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
