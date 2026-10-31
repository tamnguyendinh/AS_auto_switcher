package ui

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"as-auto-switcher/internal/core/i18n"
	"as-auto-switcher/internal/engines/codex/connector"
	"as-auto-switcher/internal/engines/codex/manager"
)

// CodexAddModal implements the Phase 5 OAuth addition and re-authentication modal
// with installed browser dropdown, link copy to clipboard, in-flight status,
// and safe 3-second auto-close upon successful login.
type CodexAddModal struct {
	window        fyne.Window
	mgr           manager.OAuthManager
	browserMgr    connector.BrowserLauncher
	dialog        dialog.Dialog
	browserSel    *widget.Select
	openBtn       *widget.Button
	copyBtn       *widget.Button
	statusLabel   *widget.Label
	cancelBtn     *widget.Button
	isClosed      atomic.Bool
	closeTimer    *time.Timer
	cooldownTimer *time.Timer
	retryBtn      *widget.Button
	unsubscribe   func()
	sequence      uint64
	titleLabel    *widget.Label
	authURL       string

	mu                sync.Mutex
	cancelCtx         context.CancelFunc
	attempt           *manager.OAuthAttempt
	mode              string // "add" or "reauth"
	expectedEmail     string
	expectedAccountID string
	browsers          []connector.BrowserInfo
	copiedContent     atomic.Value
	authReady         atomic.Bool
	successHandled    atomic.Bool
	successStarted    atomic.Bool
	onSuccess         func(email string)
	onClosed          func()
}

// NewCodexAddModal constructs a new CodexAddModal instance bound to the provided window and services.
func NewCodexAddModal(win fyne.Window, mgr manager.OAuthManager, bMgr connector.BrowserLauncher) *CodexAddModal {
	d := i18n.Current()
	m := &CodexAddModal{
		window:     win,
		mgr:        mgr,
		browserMgr: bMgr,
		mode:       "add",
	}

	// 1. Scan installed browsers and determine default
	var options []string
	defIdx := -1
	if bMgr != nil {
		if browsers, err := bMgr.ListInstalledBrowsers(); err == nil {
			m.browsers = browsers
			for i, b := range browsers {
				name := b.Name
				if b.IsDefault {
					name += " (" + d.CodexDefaultBrowser + ")"
					defIdx = i
				}
				options = append(options, name)
			}
		}
	}
	if len(options) == 0 {
		options = []string{d.CodexDefaultBrowser}
		defIdx = 0
	}

	m.browserSel = widget.NewSelect(options, nil)
	if defIdx >= 0 && defIdx < len(options) {
		m.browserSel.SetSelected(options[defIdx])
	} else if len(options) > 0 {
		options = append([]string{d.CodexDefaultBrowser}, options...)
		m.browserSel.Options = options
		m.browserSel.SetSelected(options[0])
	}

	// 2. Buttons & widgets
	m.openBtn = widget.NewButtonWithIcon(d.CodexModalBtnOpenBrowser, theme.NavigateNextIcon(), m.handleOpenBrowser)
	m.openBtn.Importance = widget.HighImportance
	m.openBtn.Disable()

	m.copyBtn = widget.NewButtonWithIcon(d.CodexModalBtnCopyLink, theme.ContentCopyIcon(), m.handleCopyLink)
	m.copyBtn.Disable()

	m.cancelBtn = widget.NewButton(d.ConfirmBtnCancel, m.handleCancel)
	m.cancelBtn.Importance = widget.LowImportance

	m.statusLabel = widget.NewLabel(d.CodexModalStatusPreparing)
	m.statusLabel.Alignment = fyne.TextAlignCenter
	m.statusLabel.Wrapping = fyne.TextWrapWord

	// 3. Assemble content layout
	titleText := d.CodexAddAccount
	titleLabel := widget.NewLabelWithStyle(titleText, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	m.titleLabel = titleLabel
	m.retryBtn = widget.NewButton(d.CodexRetrySave, m.handleRetrySave)
	m.retryBtn.Hide()
	descLabel := widget.NewLabelWithStyle(d.CodexModalDesc, fyne.TextAlignCenter, fyne.TextStyle{})
	descLabel.Wrapping = fyne.TextWrapWord

	browserBox := container.NewVBox(
		widget.NewLabelWithStyle(d.CodexModalBrowserLabel, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		m.browserSel,
	)
	actionGrid := container.NewGridWithColumns(2, m.openBtn, m.copyBtn)
	statusBox := container.NewPadded(m.statusLabel)

	content := container.NewVBox(
		titleLabel,
		descLabel,
		widget.NewSeparator(),
		browserBox,
		actionGrid,
		statusBox,
		widget.NewSeparator(),
		container.NewCenter(container.NewHBox(m.retryBtn, m.cancelBtn)),
	)

	// 4. Create dialog when window is available
	if win != nil {
		m.dialog = dialog.NewCustomWithoutButtons(titleText, content, win)
		m.dialog.Resize(fyne.NewSize(450, 320))
	}

	return m
}

// SetReauthTarget configures the modal to perform re-authentication for a specific account.
func (m *CodexAddModal) SetReauthTarget(email, accountID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mode = "reauth"
	m.expectedEmail = email
	m.expectedAccountID = accountID
	m.titleLabel.SetText(i18n.Current().CodexModalReauthTitle)
}

// SetOnSuccess registers a callback invoked when login succeeds with the verified account email.
func (m *CodexAddModal) SetOnSuccess(fn func(email string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onSuccess = fn
}

// SetOnClosed registers a callback invoked when the modal is closed.
func (m *CodexAddModal) SetOnClosed(fn func()) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onClosed = fn
}

// Show displays the modal and initiates the OAuth attempt lifecycle.
func (m *CodexAddModal) Show() {
	m.isClosed.Store(false)
	m.authReady.Store(false)
	m.successHandled.Store(false)
	m.successStarted.Store(false)
	m.sequence = 0

	ctx, cancel := context.WithCancel(context.Background())
	m.mu.Lock()
	m.cancelCtx = cancel
	mode := m.mode
	expEmail := m.expectedEmail
	expID := m.expectedAccountID
	m.mu.Unlock()

	if m.dialog != nil {
		m.dialog.Show()
	}

	go func() {
		if m.mgr == nil {
			fyne.Do(func() {
				if !m.isClosed.Load() {
					m.statusLabel.SetText(i18n.Current().CodexFailed)
				}
			})
			return
		}

		attempt, err := m.mgr.BeginLogin(ctx, mode, expEmail, expID)
		if err != nil {
			fyne.Do(func() {
				if !m.isClosed.Load() {
					m.statusLabel.SetText(i18n.Current().CodexFailed)
				}
			})
			return
		}
		events, unsubscribe := attempt.Subscribe()
		m.mu.Lock()
		if m.isClosed.Load() {
			m.mu.Unlock()
			unsubscribe()
			return
		}
		m.attempt = attempt
		m.unsubscribe = unsubscribe
		m.mu.Unlock()
		defer unsubscribe()
		for {
			select {
			case snapshot := <-events:
				if snapshot.AttemptID != attempt.AttemptID {
					continue
				}
				fyne.Do(func() { m.applySnapshot(snapshot) })
			case <-ctx.Done():
				return
			}
		}

	}()
}

// handleOpenBrowser launches the selected browser with the auth URL.
// Includes anti-double-click disabling for 2 seconds.
func (m *CodexAddModal) handleOpenBrowser() {
	m.mu.Lock()
	authURL := m.authURL
	if authURL == "" || m.openBtn.Disabled() {
		m.mu.Unlock()
		return
	}
	m.openBtn.Disable()

	// Find exe path for selected browser
	selected := m.browserSel.Selected
	var exePath string
	for _, b := range m.browsers {
		if b.Name == selected || (b.IsDefault && b.Name+" ("+i18n.Current().CodexDefaultBrowser+")" == selected) {
			exePath = b.ExePath
			break
		}
	}
	bMgr := m.browserMgr
	m.mu.Unlock()

	// Cooldown timer: re-enable after 2 seconds if still open
	m.mu.Lock()
	if m.cooldownTimer != nil {
		m.cooldownTimer.Stop()
	}
	m.cooldownTimer = time.AfterFunc(2*time.Second, func() {
		fyne.Do(func() {
			if !m.isClosed.Load() && !m.successHandled.Load() && m.openBtn != nil {
				m.mu.Lock()
				hasURL := m.authURL != ""
				m.mu.Unlock()
				if hasURL {
					m.openBtn.Enable()
				}
			}
		})
	})
	m.mu.Unlock()

	if bMgr != nil {
		go func() {
			if err := bMgr.Launch(exePath, authURL); err != nil {
				fyne.Do(func() {
					if !m.isClosed.Load() && !m.successHandled.Load() {
						m.statusLabel.SetText(i18n.Current().CodexBrowserFailed)
					}
				})
			}
		}()
	}

	fyne.Do(func() {
		if !m.isClosed.Load() {
			m.statusLabel.SetText(i18n.Current().CodexModalStatusWaitingBrowser)
		}
	})
}

// handleCopyLink copies the auth URL to the clipboard and updates status.
func (m *CodexAddModal) handleCopyLink() {
	m.mu.Lock()
	authURL := m.authURL
	m.mu.Unlock()

	if authURL != "" {
		m.copiedContent.Store(authURL)
		if m.window != nil && m.window.Clipboard() != nil {
			m.window.Clipboard().SetContent(authURL)
		}

		fyne.Do(func() {
			if !m.isClosed.Load() {
				m.statusLabel.SetText(i18n.Current().CodexModalStatusCopied)
			}
		})
	}
}

// handleCancel cancels the active attempt and closes the modal.
func (m *CodexAddModal) handleCancel() {
	m.Close()
	if m.mgr != nil {
		go func() { _ = m.mgr.CancelActiveLogin() }()
	}
}

// onLoginSuccess displays the success status, disables actionable buttons,
// and schedules automatic closure after exactly 3 seconds.
func (m *CodexAddModal) onLoginSuccess(email string) {
	fyne.Do(func() {
		if m.isClosed.Load() || m.successStarted.Swap(true) {
			return
		}
		m.mu.Lock()
		m.authURL = ""
		if m.cooldownTimer != nil {
			m.cooldownTimer.Stop()
		}
		m.mu.Unlock()
		m.retryBtn.Hide()
		m.statusLabel.SetText(fmt.Sprintf("%s (%s)", i18n.Current().CodexModalStatusSuccess, email))
		m.openBtn.Disable()
		m.copyBtn.Disable()
		m.cancelBtn.Disable()

		m.mu.Lock()
		onSuccess := m.onSuccess
		m.mu.Unlock()

		if onSuccess != nil {
			onSuccess(email)
		}

		// Schedule safe auto-close after 3 seconds
		timer := time.AfterFunc(3*time.Second, func() {
			fyne.Do(func() {
				if !m.isClosed.Load() {
					m.Close()
				}
			})
		})
		m.mu.Lock()
		m.closeTimer = timer
		m.mu.Unlock()
		m.successHandled.Store(true)
	})
}

// Close safely shuts down the modal, stops the timer, and dismisses the dialog idempotently.
func (m *CodexAddModal) Close() {
	if m.isClosed.Swap(true) {
		return
	}

	m.mu.Lock()
	if m.closeTimer != nil {
		m.closeTimer.Stop()
	}
	if m.cooldownTimer != nil {
		m.cooldownTimer.Stop()
	}
	if m.unsubscribe != nil {
		m.unsubscribe()
		m.unsubscribe = nil
	}
	m.authURL = ""
	m.copiedContent.Store("")
	if m.cancelCtx != nil {
		m.cancelCtx()
	}
	onClosed := m.onClosed
	m.mu.Unlock()

	fyne.Do(func() {
		if m.dialog != nil {
			m.dialog.Hide()
		}
	})

	if onClosed != nil {
		onClosed()
	}
}

// IsClosed reports whether the modal has been closed.
func (m *CodexAddModal) IsClosed() bool {
	return m.isClosed.Load()
}

// IsAuthReady reports whether authURL has been acquired and buttons are ready.
func (m *CodexAddModal) IsAuthReady() bool {
	return m.authReady.Load()
}

// IsSuccessHandled reports whether login success has been applied.
func (m *CodexAddModal) IsSuccessHandled() bool {
	return m.successHandled.Load()
}

// HasCloseTimer reports whether the 3-second auto-close timer has been scheduled.
func (m *CodexAddModal) HasCloseTimer() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closeTimer != nil
}

// GetCopiedContent returns the last content copied by handleCopyLink.
func (m *CodexAddModal) GetCopiedContent() string {
	v := m.copiedContent.Load()
	if v == nil {
		return ""
	}
	return v.(string)
}

// applySnapshot runs only on the Fyne UI thread.
func (m *CodexAddModal) applySnapshot(s manager.AttemptSnapshot) {
	if m.isClosed.Load() || s.Sequence <= m.sequence {
		return
	}
	m.sequence = s.Sequence
	m.mu.Lock()
	m.authURL = s.AuthURL
	m.mu.Unlock()
	d := i18n.Current()
	m.retryBtn.Hide()
	switch s.Stage {
	case "preparing":
		m.statusLabel.SetText(d.CodexModalStatusPreparing)
	case "waiting_browser":
		m.openBtn.Enable()
		m.copyBtn.Enable()
		m.authReady.Store(true)
		m.statusLabel.SetText(d.CodexModalStatusWaitingBrowser)
	case "verifying":
		m.openBtn.Disable()
		m.copyBtn.Disable()
		m.statusLabel.SetText(d.CodexVerifying)
	case "save_failed":
		m.openBtn.Disable()
		m.copyBtn.Disable()
		m.retryBtn.Enable()
		m.retryBtn.Show()
		m.statusLabel.SetText(d.CodexSaveFailed)
	case "success":
		m.onLoginSuccess(s.Email)
	case "canceled":
		m.openBtn.Disable()
		m.copyBtn.Disable()
		m.statusLabel.SetText(d.CodexModalStatusCanceled)
	default:
		if s.Err != nil {
			m.openBtn.Disable()
			m.copyBtn.Disable()
			m.statusLabel.SetText(d.CodexFailed)
		}
	}
}
func (m *CodexAddModal) handleRetrySave() {
	m.mu.Lock()
	attempt := m.attempt
	m.mu.Unlock()
	if attempt == nil || m.mgr == nil {
		return
	}
	m.retryBtn.Disable()
	go func() { _ = m.mgr.RetrySave(context.Background(), attempt.AttemptID) }()
}
