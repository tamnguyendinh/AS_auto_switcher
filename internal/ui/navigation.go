package ui

import "fyne.io/fyne/v2/widget"

func setVisible(label *widget.Label, visible bool) {
	if label == nil {
		return
	}
	if visible {
		label.Show()
	} else {
		label.Hide()
	}
}

// SwitchTab runs on the Fyne UI thread. Views stay mounted, and engine lifecycle
// and cached account state are deliberately independent of navigation.
func (a *App) SwitchTab(target NavTab) {
	if a.sidebar == nil || (target != NavTabAntigravity && target != NavTabCodex) {
		return
	}
	a.sidebar.SwitchTab(target)
	if a.win != nil {
		a.win.Canvas().Unfocus()
	}
	if a.antigravityContent != nil && a.codex != nil {
		if target == NavTabCodex {
			a.antigravityContent.Hide()
			a.codex.content.Show()
		} else {
			a.codex.content.Hide()
			a.antigravityContent.Show()
		}
	}
	a.renderFooter()
}

// renderFooter is the sole visibility owner for both contextual labels and
// Antigravity's transient messages, including completion from background work.
func (a *App) renderFooter() {
	codexSelected := a.sidebar != nil && a.sidebar.ActiveTab() == NavTabCodex
	showMessage := !codexSelected && a.statusBar != nil && a.statusBar.Text != ""
	setVisible(a.serverStatusLabel, !codexSelected && !showMessage)
	setVisible(a.versionLabel, !codexSelected && !showMessage)
	setVisible(a.codexStatusLabel, codexSelected)
	setVisible(a.codexVersionLabel, codexSelected)
	setVisible(a.statusBar, showMessage)
	if a.leftFooter != nil {
		if showMessage {
			a.leftFooter.Hide()
			a.rightFooter.Hide()
		} else {
			a.leftFooter.Show()
			a.rightFooter.Show()
		}
	}
}
