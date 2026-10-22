package ui

import (
	"context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// buildAddAccountModalContent creates and binds the internal UI layout and candidate reloader.
func (a *App) buildAddAccountModalContent() (contentBox *fyne.Container, scanBtn *widget.Button, listContainer *fyne.Container, reloadCandidates func()) {
	palette := GetThemePalette(a.currentTheme)

	listContainer = container.NewVBox()
	scrollContent := container.NewVScroll(container.NewPadded(listContainer))
	scrollContent.SetMinSize(fyne.NewSize(420, 260))

	reloadCandidates = func() {
		candidates, err := a.mgr.GetCandidateAccounts()
		if err != nil {
			return
		}
		listContainer.Objects = nil
		if len(candidates) == 0 {
			emptyLbl := widget.NewLabelWithStyle("Không có tài khoản nào trong danh bạ.", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})
			listContainer.Add(emptyLbl)
			listContainer.Refresh()
			return
		}

		for _, cand := range candidates {
			candItem := cand // capture
			emailText := candItem.Email
			if a.IsMaskAccounts() {
				emailText = MaskEmail(candItem.Email)
			}
			emailLbl := widget.NewLabel(emailText)
			emailLbl.Truncation = fyne.TextTruncateEllipsis

			var actionSlot *fyne.Container

			if candItem.HasToken {
				// Trường hợp 1: [✔ Đã OAuth]
				statusBadge := canvas.NewText("[✔ Đã OAuth]", palette.Success)
				statusBadge.TextSize = 11
				statusBadge.TextStyle = fyne.TextStyle{Bold: true}
				actionSlot = container.NewCenter(statusBadge)
			} else if candItem.Added {
				// Trường hợp 3: [Đã add]
				statusBadge := canvas.NewText("[Đã add]", palette.TextMuted)
				statusBadge.TextSize = 11
				statusBadge.TextStyle = fyne.TextStyle{Bold: true}
				actionSlot = container.NewCenter(statusBadge)
			} else {
				// Trường hợp 2: [Cần OAuth] + Nút [ + ADD ]
				statusBadge := canvas.NewText("[Cần OAuth]", palette.Warning)
				statusBadge.TextSize = 11
				statusBadge.TextStyle = fyne.TextStyle{Bold: true}

				var addBtn *widget.Button
				addBtn = widget.NewButtonWithIcon("ADD", theme.ContentAddIcon(), func() {
					if err := a.mgr.AddDiscoveredAccount(candItem.Email); err == nil {
						// Chuyển đổi tại chỗ: [Cần OAuth] đổi thành [Đã add] và nút ADD biến mất
						doneBadge := canvas.NewText("[Đã add]", palette.TextMuted)
						doneBadge.TextSize = 11
						doneBadge.TextStyle = fyne.TextStyle{Bold: true}
						actionSlot.Objects = []fyne.CanvasObject{doneBadge}
						actionSlot.Refresh()

						// Nạp ngay thẻ mới lên UI tổng
						go func() {
							overview, _ := a.mgr.GetAccountStates(context.Background())
							a.mu.Lock()
							a.overview = overview
							a.mu.Unlock()
							fyne.Do(func() {
								a.updateUI(overview)
							})
						}()
					}
				})
				addBtn.Importance = widget.HighImportance
				actionSlot = container.NewHBox(container.NewCenter(statusBadge), compactButton(addBtn))
			}

			row := container.NewBorder(nil, nil, nil, actionSlot, emailLbl)
			listContainer.Add(row)
		}
		listContainer.Refresh()
	}

	scanBtn = widget.NewButtonWithIcon("Quét Chrome", theme.ViewRefreshIcon(), func() {
		scanBtn.Disable()
		scanBtn.SetText("⏳ Đang quét...")
		go func() {
			_, _ = a.mgr.ScanChromeAccounts()
			fyne.Do(func() {
				reloadCandidates()
				scanBtn.SetText("Quét Chrome")
				scanBtn.Enable()
			})
		}()
	})

	headerRow := container.NewBorder(
		nil, nil,
		widget.NewLabelWithStyle("Thêm tài khoản", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		compactButton(scanBtn),
	)

	contentBox = container.NewBorder(
		container.NewVBox(headerRow, widget.NewSeparator()),
		nil, nil, nil,
		scrollContent,
	)

	reloadCandidates()
	return contentBox, scanBtn, listContainer, reloadCandidates
}

// ShowAddAccountModal displays the Add Account dialog.
func (a *App) ShowAddAccountModal() {
	if a == nil || a.win == nil || a.mgr == nil {
		return
	}

	contentBox, _, _, _ := a.buildAddAccountModalContent()

	d := dialog.NewCustom("Thêm tài khoản", "Đóng", contentBox, a.win)
	d.Resize(fyne.NewSize(460, 340))
	d.Show()
}
