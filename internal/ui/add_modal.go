package ui

import (
	"context"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"antigravity-switch/internal/manager"
	"antigravity-switch/internal/model"
)

// buildAddAccountModalContent creates and binds the internal UI layout and candidate reloader.
func (a *App) buildAddAccountModalContent() (contentBox *fyne.Container, scanBtn *widget.Button, listContainer *fyne.Container, reloadCandidates func()) {
	palette := GetThemePalette(a.currentTheme)
	dismissedAccounts := make(map[string]bool)

	listContainer = container.NewVBox()
	scrollContent := container.NewVScroll(container.NewPadded(listContainer))
	scrollContent.SetMinSize(fyne.NewSize(420, 260))

	reloadCandidates = func() {
		var known []string
		a.mu.RLock()
		if a.overview != nil {
			for _, acc := range a.overview.Accounts {
				if acc.State == model.StateUnauth || !acc.HasToken {
					known = append(known, acc.Email)
				}
			}
		}
		a.mu.RUnlock()

		candidates, err := a.mgr.GetCandidateAccounts(known...)
		if err != nil {
			return
		}
		listContainer.Objects = nil

		var activeCandidates []manager.CandidateAccountItem
		for _, cand := range candidates {
			norm := strings.ToLower(strings.TrimSpace(cand.Email))
			if !dismissedAccounts[norm] {
				activeCandidates = append(activeCandidates, cand)
			}
		}

		if len(activeCandidates) == 0 {
			emptyLbl := widget.NewLabelWithStyle("Không có tài khoản nào trong danh bạ.", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})
			listContainer.Add(emptyLbl)
			listContainer.Refresh()
			return
		}

		for _, cand := range activeCandidates {
			candItem := cand // capture
			emailText := candItem.Email
			if a.IsMaskAccounts() {
				emailText = MaskEmail(candItem.Email)
			}
			emailLbl := widget.NewLabel(emailText)
			emailLbl.Truncation = fyne.TextTruncateEllipsis

			deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
				a.showDeleteCandidateConfirm(candItem, func() {
					norm := strings.ToLower(strings.TrimSpace(candItem.Email))
					dismissedAccounts[norm] = true
					reloadCandidates()
				})
			})
			deleteBtn.Importance = widget.LowImportance
			compactDel := compactButton(deleteBtn)

			var actionSlot *fyne.Container

			if !candItem.FileExists {
				// Trường hợp 4: [Tài khoản không tồn tại] + Nút [Xóa] (Ẩn nút ADD)
				statusBadge := canvas.NewText("[Tài khoản không tồn tại]", palette.TextMuted)
				statusBadge.TextSize = 11
				statusBadge.TextStyle = fyne.TextStyle{Bold: true}

				actionSlot = container.NewHBox(container.NewCenter(statusBadge), compactDel)

			} else if candItem.HasToken {
				// Trường hợp 1: [✔ Đã OAuth] + Nút [Xóa]
				statusBadge := canvas.NewText("[✔ Đã OAuth]", palette.Success)
				statusBadge.TextSize = 11
				statusBadge.TextStyle = fyne.TextStyle{Bold: true}

				actionSlot = container.NewHBox(container.NewCenter(statusBadge), compactDel)

			} else if candItem.Added {
				// Trường hợp 2: [Đã add] + Nút [Xóa]
				statusBadge := canvas.NewText("[Đã add]", palette.TextMuted)
				statusBadge.TextSize = 11
				statusBadge.TextStyle = fyne.TextStyle{Bold: true}

				actionSlot = container.NewHBox(container.NewCenter(statusBadge), compactDel)

			} else {
				// Trường hợp 3: [Cần OAuth] + Nút [ + ADD ] + Nút [Xóa]
				statusBadge := canvas.NewText("[Cần OAuth]", palette.Warning)
				statusBadge.TextSize = 11
				statusBadge.TextStyle = fyne.TextStyle{Bold: true}

				var addBtn *widget.Button
				addBtn = widget.NewButtonWithIcon("ADD", theme.ContentAddIcon(), func() {
					if err := a.mgr.AddDiscoveredAccount(candItem.Email); err == nil {
						// Chuyển đổi tại chỗ: [Cần OAuth] đổi thành [Đã add] và nút ADD biến mất, giữ nguyên nút Xóa
						doneBadge := canvas.NewText("[Đã add]", palette.TextMuted)
						doneBadge.TextSize = 11
						doneBadge.TextStyle = fyne.TextStyle{Bold: true}
						actionSlot.Objects = []fyne.CanvasObject{container.NewCenter(doneBadge), compactDel}
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
				actionSlot = container.NewHBox(container.NewCenter(statusBadge), compactButton(addBtn), compactDel)
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

// showDeleteCandidateConfirm displays a confirmation dialog before deleting a candidate account.
func (a *App) showDeleteCandidateConfirm(cand manager.CandidateAccountItem, onConfirmed func()) *dialog.ConfirmDialog {
	if a == nil || a.win == nil {
		return nil
	}

	displayEmail := cand.Email
	if a.IsMaskAccounts() {
		displayEmail = MaskEmail(cand.Email)
	}

	title := "Xác nhận xóa tài khoản"
	msg := fmt.Sprintf("Bạn có chắc chắn muốn xóa tài khoản %s khỏi danh bạ đã quét không?\n\n(Lưu ý: Thao tác này chỉ xóa dữ liệu trong danh bạ khám phá, không xóa tài khoản hay token trên màn hình chính).", displayEmail)

	if !cand.FileExists {
		msg = fmt.Sprintf("Tài khoản %s không có tệp dữ liệu trên hệ thống.\nBạn có muốn gỡ bỏ tài khoản này khỏi danh sách hiển thị không?", displayEmail)
	}

	confirmDlg := dialog.NewConfirm(
		title,
		msg,
		func(confirmed bool) {
			if !confirmed {
				return
			}
			if cand.FileExists && a.mgr != nil {
				_ = a.mgr.DeleteDiscoveredAccount(cand.Email)
			}
			fyne.Do(func() {
				if onConfirmed != nil {
					onConfirmed()
				}
			})
		},
		a.win,
	)
	confirmDlg.SetConfirmImportance(widget.DangerImportance)
	confirmDlg.SetConfirmText("Xóa")
	confirmDlg.SetDismissText("Hủy")
	confirmDlg.Show()
	return confirmDlg
}
