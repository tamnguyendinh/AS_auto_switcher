package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"as-auto-switcher/internal/core/i18n"
	"as-auto-switcher/internal/engines/antigravity/manager"
	"as-auto-switcher/internal/engines/antigravity/model"
	"fyne.io/fyne/v2"
)

const identityPollInterval = 5 * time.Second

// requestSessionSync coalesces requests without coupling polling to toolbar UX.
// Manual requests joining a background read receive its result and feedback.
func (a *App) requestSessionSync(manual, periodic bool) {
	a.mu.Lock()
	if a.closed.Load() || a.mgr == nil {
		a.mu.Unlock()
		return
	}
	if a.sessionSyncRunning {
		if a.mgr.IsMutating() || (a.isBusy && !a.sessionSyncManual) {
			if !periodic {
				a.sessionSyncPending = true
				a.sessionSyncPendingManual = a.sessionSyncPendingManual || manual
			}
			a.mu.Unlock()
			return
		}
		if manual {
			a.sessionSyncManual = true
			a.isBusy = true
		}
		if !periodic && !manual {
			a.sessionSyncPending = true
		}
		a.mu.Unlock()
		if manual {
			fyne.Do(func() {
				if !a.closed.Load() && a.refreshBtn != nil {
					a.refreshBtn.SetText(i18n.Current().BtnRefreshListLoading)
				}
			})
		}
		return
	}
	if a.isBusy || a.mgr.IsMutating() {
		if !periodic {
			a.sessionSyncPending = true
			a.sessionSyncPendingManual = a.sessionSyncPendingManual || manual
		}
		a.mu.Unlock()
		return
	}
	manual = manual || a.sessionSyncPendingManual
	a.sessionSyncPending, a.sessionSyncPendingManual = false, false
	a.sessionSyncRunning, a.sessionSyncManual = true, manual
	if manual {
		a.isBusy = true
	}
	a.lastIdentityPoll = time.Now()
	gen := a.refreshGen.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	a.sessionSyncCancel = cancel
	mgr := a.mgr
	a.mu.Unlock()
	if manual {
		fyne.Do(func() {
			if !a.closed.Load() && a.refreshBtn != nil {
				a.refreshBtn.SetText(i18n.Current().BtnRefreshListLoading)
			}
		})
	}
	go func() {
		overview, err := mgr.GetAccountStates(ctx)
		cancel()
		if a.closed.Load() {
			a.mu.Lock()
			a.sessionSyncRunning = false
			if a.sessionSyncManual {
				a.isBusy = false
			}
			a.mu.Unlock()
			return
		}
		fyne.Do(func() { a.finishSessionSync(gen, overview, err) })
	}()
}

func (a *App) finishSessionSync(gen uint64, overview *manager.AccountOverview, err error) {
	a.mu.Lock()
	manual := a.sessionSyncManual
	valid := !a.closed.Load() && gen == a.refreshGen.Load() && (!a.isBusy || manual) && a.mgr != nil && !a.mgr.IsMutating()
	if overview != nil && overview.Versioned && overview.Revision != a.mgr.Revision() {
		valid = false
	}
	if (!valid || errors.Is(err, context.Canceled)) && !a.closed.Load() {
		a.sessionSyncPending = true
		a.sessionSyncPendingManual = a.sessionSyncPendingManual || manual
	}
	a.mu.Unlock()
	// Keep the single-flight slot until the UI result and feedback are applied.
	defer func() {
		a.mu.Lock()
		a.sessionSyncRunning, a.sessionSyncManual = false, false
		a.sessionSyncCancel = nil
		if manual && gen == a.refreshGen.Load() {
			a.isBusy = false
		}
		pending := a.sessionSyncPending
		a.mu.Unlock()
		if !a.closed.Load() && pending {
			a.requestSessionSync(false, false)
		}
	}()
	if a.closed.Load() {
		return
	}
	if manual && a.refreshBtn != nil {
		a.refreshBtn.SetText(i18n.Current().BtnRefreshList)
	}
	if !valid {
		return
	}
	if err == nil && overview != nil {
		a.updateUI(overview)
		a.showSessionWarnings(overview, manual)
	} else if err != nil && !errors.Is(err, context.Canceled) {
		a.mu.RLock()
		previous := a.overview
		a.mu.RUnlock()
		if previous != nil {
			unknown := *previous
			unknown.ActiveEmail, unknown.IdentitySource = "", "unknown"
			unknown.Revision = a.mgr.Revision()
			unknown.Accounts = append([]manager.AccountDisplayItem(nil), previous.Accounts...)
			for i := range unknown.Accounts {
				item := &unknown.Accounts[i]
				item.IsActive = false
				if item.HasToken {
					item.State, item.DisplayBadge = model.StateReady, "[Chuyển]"
				} else {
					item.State, item.DisplayBadge = model.StateUnauth, "[Cần OAuth]"
				}
			}
			a.updateUI(&unknown)
		}
		a.showSessionWarnings(&manager.AccountOverview{Warnings: []string{fmt.Sprintf(i18n.Current().StatusFetchError, err)}, WarningCodes: []string{"session_read"}}, manual)
	}
}

func (a *App) showSessionWarnings(view *manager.AccountOverview, manual bool) {
	current := make(map[string]bool)
	fresh := false
	for _, code := range view.WarningCodes {
		current[code] = true
		if !a.backgroundWarnings[code] {
			fresh = true
		}
	}
	// Legacy manager fixtures can still provide warnings without structured codes.
	if len(view.Warnings) > 0 && len(current) == 0 {
		current["session_warning"] = true
		fresh = !a.backgroundWarnings["session_warning"]
	}
	// A blocked warning remains eligible after the operation finishes.
	if len(view.Warnings) > 0 && !manual && a.statusBar != nil && a.statusBar.Text != "" {
		return
	}
	a.backgroundWarnings = current
	if len(view.Warnings) > 0 && (manual || fresh) {
		// Background diagnostics must not replace an operation's status message.
		if !manual && a.statusBar != nil && a.statusBar.Text != "" {
			return
		}
		messages := view.Warnings
		if len(view.WarningCodes) == len(view.Warnings) {
			messages = nil
			seen := make(map[string]bool)
			for _, code := range view.WarningCodes {
				if !seen[code] {
					messages = append(messages, sessionWarningText(code))
					seen[code] = true
				}
			}
		}
		msg := strings.Join(messages, "; ")
		if a.IsMaskAccounts() {
			msg = maskMessageEmails(msg)
		}
		a.showStatusBar("⚠️ " + msg)
		a.autoHideStatusBar(3 * time.Second)
	} else if manual && len(view.Warnings) == 0 {
		a.showStatusBar(i18n.Current().StatusFetchSuccess)
		a.autoHideStatusBar(3 * time.Second)
	}
}

// Called by the existing liveness worker; no second polling goroutine is needed.
func (a *App) pollSessionIdentity() {
	a.mu.RLock()
	pending := a.sessionSyncPending
	running := a.overview != nil && a.overview.LanguageServerRunning
	due := time.Since(a.lastIdentityPoll) >= identityPollInterval
	changed := a.mgr != nil && a.overview != nil && a.overview.Versioned && a.overview.Revision != a.mgr.Revision()
	a.mu.RUnlock()
	if pending || changed {
		a.requestSessionSync(false, false)
	} else if running && due {
		a.requestSessionSync(false, true)
	}
}

func sessionWarningText(code string) string {
	d := i18n.Current()
	switch code {
	case "credential_identity_missing":
		return d.SessionCredentialMissing
	case "credential_identity_conflict":
		return d.SessionCredentialConflict
	case "credential_read":
		return d.SessionCredentialRead
	case "credential_invalid":
		return d.SessionCredentialInvalid
	case "credential_write":
		return d.SessionCredentialWrite
	case "identity_unavailable":
		return d.SessionIdentityUnavailable
	case "credential_store_read":
		return d.SessionStoreRead
	case "catalog_read":
		return d.SessionCatalogRead
	default:
		return d.SessionReadFailed
	}
}
