package ui

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"as-auto-switcher/internal/core/i18n"
	"as-auto-switcher/internal/engines/antigravity/manager"
	"as-auto-switcher/internal/engines/antigravity/model"
)

// buildAddAccountModalContent creates and binds the internal UI layout and candidate reloader.
func (a *App) buildAddAccountModalContent() (contentBox *fyne.Container, scanBtn *widget.Button, listContainer *fyne.Container, reloadCandidates func()) {
	d := i18n.Current()
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
			emptyLbl := widget.NewLabelWithStyle(d.ModalEmptyCandidates, fyne.TextAlignCenter, fyne.TextStyle{Italic: true})
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
				// Trường hợp 4: [Tài khoản không tồn tại] / [Account Not Found] + Nút [Xóa] (Ẩn nút ADD)
				statusBadge := canvas.NewText(d.ModalBadgeNotFound, palette.TextMuted)
				statusBadge.TextSize = 11
				statusBadge.TextStyle = fyne.TextStyle{Bold: true}

				actionSlot = container.NewHBox(container.NewCenter(statusBadge), compactDel)

			} else if candItem.HasToken {
				// Trường hợp 1: [✔ Đã OAuth] / [✔ OAuth Done] + Nút [Xóa]
				statusBadge := canvas.NewText(d.ModalBadgeOAuthDone, palette.Success)
				statusBadge.TextSize = 11
				statusBadge.TextStyle = fyne.TextStyle{Bold: true}

				actionSlot = container.NewHBox(container.NewCenter(statusBadge), compactDel)

			} else if candItem.Added {
				// Trường hợp 2: [Đã thêm] / [Added] + Nút [Xóa]
				statusBadge := canvas.NewText(d.ModalBadgeAdded, palette.TextMuted)
				statusBadge.TextSize = 11
				statusBadge.TextStyle = fyne.TextStyle{Bold: true}

				actionSlot = container.NewHBox(container.NewCenter(statusBadge), compactDel)

			} else {
				// Trường hợp 3: [Cần OAuth] / [Need OAuth] + Nút [Thêm / Add] + Nút [Xóa]
				statusBadge := canvas.NewText(d.ModalBadgeNeedOAuth, palette.Warning)
				statusBadge.TextSize = 11
				statusBadge.TextStyle = fyne.TextStyle{Bold: true}

				var addBtn *widget.Button
				addBtn = widget.NewButtonWithIcon(d.ModalBtnAddItem, theme.ContentAddIcon(), func() {
					addBtn.Disable()
					go func() {
						err := a.mgr.AddDiscoveredAccount(candItem.Email)
						fyne.Do(func() {
							if a.closed.Load() {
								return
							}
							if err != nil {
								addBtn.Enable()
								a.showStatusBar(fmt.Sprintf(i18n.Current().StatusUpdateError, err))
								a.autoHideStatusBar(3 * time.Second)
								return
							}
							addBtn.Hide()
							doneBadge := canvas.NewText(i18n.Current().ModalBadgeAdded, palette.TextMuted)
							doneBadge.TextSize = 11
							doneBadge.TextStyle = fyne.TextStyle{Bold: true}
							actionSlot.Objects = []fyne.CanvasObject{container.NewCenter(doneBadge), compactDel}
							actionSlot.Refresh()
							a.requestSessionSync(false, false)
						})
					}()
				})
				addBtn.Importance = widget.HighImportance
				actionSlot = container.NewHBox(container.NewCenter(statusBadge), compactButton(addBtn), compactDel)
			}

			row := container.NewBorder(nil, nil, nil, actionSlot, emailLbl)
			listContainer.Add(row)
		}
		listContainer.Refresh()
	}

	scanBtn = widget.NewButtonWithIcon(d.ModalScanChrome, theme.ViewRefreshIcon(), func() {
		scanBtn.Disable()
		scanBtn.SetText(d.ModalScanningChrome)
		go func() {
			_, _ = a.mgr.ScanChromeAccounts()
			fyne.Do(func() {
				if a.closed.Load() {
					return
				}
				reloadCandidates()
				scanBtn.SetText(d.ModalScanChrome)
				scanBtn.Enable()
			})
		}()
	})

	headerRow := container.NewBorder(
		nil, nil,
		widget.NewLabelWithStyle(d.ModalAddHeader, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
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

	d := i18n.Current()
	dlg := dialog.NewCustom(d.ModalAddTitle, d.ModalAddClose, contentBox, a.win)
	dlg.Resize(fyne.NewSize(460, 340))
	dlg.Show()
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

	d := i18n.Current()
	title := d.ConfirmDeleteTitle
	msg := fmt.Sprintf(d.ConfirmMsgDiscExists, displayEmail)

	if !cand.FileExists {
		msg = fmt.Sprintf(d.ConfirmMsgDiscNoFile, displayEmail)
	}

	confirmDlg := dialog.NewConfirm(
		title,
		msg,
		func(confirmed bool) {
			if !confirmed {
				return
			}
			go func() {
				if cand.FileExists && a.mgr != nil {
					if err := a.mgr.DeleteDiscoveredAccount(cand.Email); err != nil {
						fyne.Do(func() {
							if !a.closed.Load() {
								a.showStatusBar(fmt.Sprintf(i18n.Current().StatusDeleteError, err))
								a.autoHideStatusBar(3 * time.Second)
							}
						})
						return
					}
				}
				fyne.Do(func() {
					if a.closed.Load() {
						return
					}
					if onConfirmed != nil {
						onConfirmed()
					}
					a.requestSessionSync(false, false)
				})
			}()
		},
		a.win,
	)
	confirmDlg.SetConfirmImportance(widget.DangerImportance)
	confirmDlg.SetConfirmText(d.ConfirmBtnDelete)
	confirmDlg.SetDismissText(d.ConfirmBtnCancel)
	confirmDlg.Show()
	return confirmDlg
}
