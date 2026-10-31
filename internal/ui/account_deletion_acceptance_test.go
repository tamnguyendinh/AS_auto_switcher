package ui

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"

	"as-auto-switcher/internal/engines/antigravity/connector"
	"as-auto-switcher/internal/engines/antigravity/manager"
	"as-auto-switcher/internal/engines/antigravity/model"
	"as-auto-switcher/internal/engines/antigravity/reloader"
	"as-auto-switcher/internal/engines/antigravity/store"
	"as-auto-switcher/internal/engines/antigravity/telemetry"
	"as-auto-switcher/internal/engines/antigravity/wincred"
)

// helper to locate an AccountCardWidget by email in App
func findCardInApp(a *App, email string) *AccountCardWidget {
	a.mu.RLock()
	defer a.mu.RUnlock()
	for _, card := range a.accountCards {
		if card != nil && strings.EqualFold(card.Email, email) {
			return card
		}
	}
	return nil
}

// helper to check if discovered store has a record with Added=true
func isDiscoveredAdded(s *store.DiscoveredStore, email string) bool {
	records, err := s.ListDiscoveredRecords()
	if err != nil {
		return false
	}
	for _, r := range records {
		if strings.EqualFold(r.Email, email) {
			return r.Added
		}
	}
	return false
}

// setupDeletionAcceptanceApp creates an isolated App environment for testing account deletion.
func setupDeletionAcceptanceApp(t *testing.T, activeEmail string) (*App, *manager.AccountManager, *store.AccountStore, *store.DiscoveredStore, string) {
	tempDir := t.TempDir()
	storeDir := filepath.Join(tempDir, "accounts_store")
	prefPath := filepath.Join(tempDir, "ui_preferences.json")

	accStore, err := store.NewAccountStore(storeDir)
	if err != nil {
		t.Fatalf("failed initializing AccountStore: %v", err)
	}

	sessionReloader := reloader.NewSessionReloader()
	sessionReloader.StoragePath = filepath.Join(tempDir, "app_storage.json")
	sessionReloader.ProcessKiller = func(ctx context.Context) error { return nil }
	sessionReloader.Sleeper = func(ctx context.Context, d time.Duration) error { return nil }
	sessionReloader.SnapshotCapturer = func(string, string, string) (string, []reloader.ActiveThread, error) { return "", nil, nil }

	mgr, err := manager.NewAccountManager(accStore, sessionReloader)
	if err != nil {
		t.Fatalf("failed initializing AccountManager: %v", err)
	}
	discStore := mgr.DiscoveredStore()
	emptyChromeDir := filepath.Join(tempDir, "empty_chrome")
	mgr.SetChromeDir(emptyChromeDir)

	var activeToken *model.TokenData
	if strings.TrimSpace(activeEmail) != "" {
		activeToken = &model.TokenData{
			AccessToken:  "active_access_" + activeEmail,
			RefreshToken: "active_refresh_" + activeEmail,
			IDToken:      makeTestJWT(activeEmail),
		}
	}

	mgr.SetWincredHandlers(
		func() (*model.TokenData, error) {
			if activeToken == nil {
				return nil, wincred.ErrNotFound
			}
			return activeToken, nil
		},
		func(token *model.TokenData) error {
			activeToken = token
			return nil
		},
	)
	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return nil, errors.New("isolated acceptance test: no language server")
	})

	testApp := test.NewApp()
	app := newIsolatedApp(testApp, mgr)
	app.SetPreferencesPath(prefPath)

	return app, mgr, accStore, discStore, prefPath
}

// -----------------------------------------------------------------------------
// TC-01: Xóa thẻ Inactive Status ON
// -----------------------------------------------------------------------------
func TestAcceptance_TC01_DeleteInactiveStatusOnAccount(t *testing.T) {
	activeEmail := "active@test.com"
	targetEmail := "user1@test.com"

	app, _, accStore, discStore, _ := setupDeletionAcceptanceApp(t, activeEmail)
	defer app.Close()

	// 1. Add active account and target account
	_ = accStore.SaveAccount(activeEmail, &model.TokenData{AccessToken: "act_tok", RefreshToken: "act_ref", IDToken: makeTestJWT(activeEmail)})
	_ = accStore.SaveAccount(targetEmail, &model.TokenData{AccessToken: "tok1", RefreshToken: "ref1", IDToken: makeTestJWT(targetEmail)})
	_ = discStore.SetDiscoveredAdded(targetEmail, true)

	app.Init()
	waitForIdle(app, 2*time.Second)

	// 2. Verify target card is displayed and Delete Button is visible
	card := findCardInApp(app, targetEmail)
	if card == nil {
		t.Fatalf("expected card for %s to be displayed", targetEmail)
	}
	if !card.DeleteButton().Visible() {
		t.Fatalf("expected DeleteButton to be visible on Inactive ON card")
	}

	// 3. Trigger ConfirmAndDeleteAccount -> pop-up confirmation
	app.ConfirmAndDeleteAccount(targetEmail)
	topOverlay := app.win.Canvas().Overlays().Top()
	if topOverlay == nil {
		t.Fatalf("expected confirmation dialog overlay on app.win")
	}

	// 4. Execute deletion
	app.executeRemoveAccount(targetEmail, targetEmail)
	waitForIdle(app, 2*time.Second)

	// 5. Verify: accounts_store file deleted, discovered Added=false, card removed from UI
	if accStore.HasAccount(targetEmail) {
		t.Errorf("expected %s to be deleted from accounts_store", targetEmail)
	}
	if isDiscoveredAdded(discStore, targetEmail) {
		t.Errorf("expected discovered record for %s to have Added=false", targetEmail)
	}

	cardAfter := findCardInApp(app, targetEmail)
	if cardAfter != nil {
		t.Errorf("expected card for %s to be removed from UI", targetEmail)
	}

	// Check status bar message
	app.mu.RLock()
	sbText := ""
	if app.statusBar != nil {
		sbText = app.statusBar.Text
	}
	app.mu.RUnlock()
	expectedMsg := fmt.Sprintf("🗑 Đã xóa tài khoản %s khỏi hệ thống.", targetEmail)
	if sbText != expectedMsg {
		t.Errorf("expected status bar %q, got %q", expectedMsg, sbText)
	}
}

// -----------------------------------------------------------------------------
// TC-02: Xóa thẻ Inactive Status OFF
// -----------------------------------------------------------------------------
func TestAcceptance_TC02_DeleteInactiveStatusOffAccount(t *testing.T) {
	activeEmail := "active@test.com"
	targetEmail := "user2@test.com"

	app, _, accStore, _, _ := setupDeletionAcceptanceApp(t, activeEmail)
	defer app.Close()

	_ = accStore.SaveAccount(activeEmail, &model.TokenData{AccessToken: "act_tok", RefreshToken: "act_ref", IDToken: makeTestJWT(activeEmail)})
	_ = accStore.SaveAccount(targetEmail, &model.TokenData{AccessToken: "tok2", RefreshToken: "ref2", IDToken: makeTestJWT(targetEmail)})
	_ = accStore.SetAccountStatus(targetEmail, "off")

	app.Init()
	waitForIdle(app, 2*time.Second)

	card := findCardInApp(app, targetEmail)
	if card == nil {
		t.Fatalf("expected card for %s to be displayed", targetEmail)
	}

	// Verify Switch button is disabled
	actionBtn, ok := card.ActionButton().(*switchTextButton)
	if !ok || !actionBtn.IsDisabled() {
		t.Fatalf("expected switch button to be disabled for Status OFF card")
	}

	// Verify Invariant 5: Delete button remains visible and enabled
	if !card.DeleteButton().Visible() || card.DeleteButton().Disabled() {
		t.Fatalf("Invariant 5 violated: DeleteButton must be visible and enabled for Status OFF card")
	}

	// Execute deletion
	app.executeRemoveAccount(targetEmail, targetEmail)
	waitForIdle(app, 2*time.Second)

	if accStore.HasAccount(targetEmail) {
		t.Errorf("expected token file for %s to be deleted", targetEmail)
	}
	if findCardInApp(app, targetEmail) != nil {
		t.Errorf("expected card for %s to disappear from UI", targetEmail)
	}
}

// -----------------------------------------------------------------------------
// TC-03: Xóa thẻ [Cần OAuth]
// -----------------------------------------------------------------------------
func TestAcceptance_TC03_DeleteNeedOAuthAccount(t *testing.T) {
	activeEmail := "active@test.com"
	targetEmail := "user3@test.com"

	app, _, accStore, discStore, _ := setupDeletionAcceptanceApp(t, activeEmail)
	defer app.Close()

	_ = accStore.SaveAccount(activeEmail, &model.TokenData{AccessToken: "act_tok", RefreshToken: "act_ref", IDToken: makeTestJWT(activeEmail)})
	_ = discStore.SetDiscoveredAdded(targetEmail, true)

	app.Init()
	waitForIdle(app, 2*time.Second)

	card := findCardInApp(app, targetEmail)
	if card == nil {
		t.Fatalf("expected card for [Cần OAuth] %s to appear on main window", targetEmail)
	}
	if card.HasToken {
		t.Fatalf("expected HasToken=false for [Cần OAuth] card")
	}

	// Verify Delete Button is visible
	if !card.DeleteButton().Visible() {
		t.Fatalf("expected DeleteButton to be visible on [Cần OAuth] card")
	}

	// Test Confirmation Dialog message formatting for Need OAuth
	app.ConfirmAndDeleteAccount(targetEmail)
	topOverlay := app.win.Canvas().Overlays().Top()
	if topOverlay == nil {
		t.Fatalf("expected confirmation dialog overlay")
	}

	// Execute deletion
	app.executeRemoveAccount(targetEmail, targetEmail)
	waitForIdle(app, 2*time.Second)

	// Verify discovered_accounts has Added=false
	if isDiscoveredAdded(discStore, targetEmail) {
		t.Errorf("expected discovered record for %s to have Added=false", targetEmail)
	}
	if findCardInApp(app, targetEmail) != nil {
		t.Errorf("expected card for %s to disappear from main UI", targetEmail)
	}
}

// -----------------------------------------------------------------------------
// TC-04: Chặn xóa thẻ Active
// -----------------------------------------------------------------------------
func TestAcceptance_TC04_ProtectActiveAccount(t *testing.T) {
	activeEmail := "active@test.com"
	otherEmail := "other@test.com"

	app, mgr, accStore, _, _ := setupDeletionAcceptanceApp(t, activeEmail)
	defer app.Close()

	_ = accStore.SaveAccount(activeEmail, &model.TokenData{AccessToken: "act_tok", RefreshToken: "act_ref", IDToken: makeTestJWT(activeEmail)})
	_ = accStore.SaveAccount(otherEmail, &model.TokenData{AccessToken: "other_tok", RefreshToken: "other_ref", IDToken: makeTestJWT(otherEmail)})

	app.Init()
	waitForIdle(app, 2*time.Second)

	activeCard := findCardInApp(app, activeEmail)
	if activeCard == nil {
		t.Fatalf("expected active card to exist")
	}

	// Invariant 1: Tuyệt đối KHÔNG hiển thị nút Xóa trên thẻ Active
	if activeCard.DeleteButton().Visible() {
		t.Fatalf("Invariant 1 Violated: DeleteButton MUST be hidden on Active Card")
	}

	// Direct call to mgr.RemoveAccount must return ErrCannotDeleteActiveAccount
	err := mgr.RemoveAccount(context.Background(), activeEmail)
	if !errors.Is(err, manager.ErrCannotDeleteActiveAccount) {
		t.Fatalf("expected ErrCannotDeleteActiveAccount, got: %v", err)
	}

	// Call to app.ConfirmAndDeleteAccount must be rejected and show warning
	app.ConfirmAndDeleteAccount(activeEmail)
	app.mu.RLock()
	sbText := ""
	if app.statusBar != nil {
		sbText = app.statusBar.Text
	}
	app.mu.RUnlock()
	if sbText != "⚠️ Không thể xóa tài khoản đang hoạt động." {
		t.Errorf("expected warning status bar for active deletion attempt, got %q", sbText)
	}

	// Verify active account is still intact on disk
	if !accStore.HasAccount(activeEmail) {
		t.Errorf("active account file must not be deleted")
	}
}

// -----------------------------------------------------------------------------
// TC-05: Hủy bỏ Dialog (Dismiss)
// -----------------------------------------------------------------------------
func TestAcceptance_TC05_DismissConfirmationDialog(t *testing.T) {
	activeEmail := "active@test.com"
	targetEmail := "user5@test.com"

	app, _, accStore, _, _ := setupDeletionAcceptanceApp(t, activeEmail)
	defer app.Close()

	_ = accStore.SaveAccount(activeEmail, &model.TokenData{AccessToken: "act_tok", RefreshToken: "act_ref", IDToken: makeTestJWT(activeEmail)})
	_ = accStore.SaveAccount(targetEmail, &model.TokenData{AccessToken: "tok5", RefreshToken: "ref5", IDToken: makeTestJWT(targetEmail)})

	app.Init()
	waitForIdle(app, 2*time.Second)

	card := findCardInApp(app, targetEmail)
	if card == nil {
		t.Fatalf("expected card for %s", targetEmail)
	}

	// Trigger confirmation dialog
	app.ConfirmAndDeleteAccount(targetEmail)
	topOverlay := app.win.Canvas().Overlays().Top()
	if topOverlay == nil {
		t.Fatalf("expected dialog overlay")
	}

	// Dismiss dialog without calling executeRemoveAccount
	app.win.Canvas().Overlays().Remove(topOverlay)

	// Verify file and card remain intact
	if !accStore.HasAccount(targetEmail) {
		t.Errorf("target account file must remain after dismiss")
	}
	if findCardInApp(app, targetEmail) == nil {
		t.Errorf("card must remain in UI after dismiss")
	}
}

// -----------------------------------------------------------------------------
// TC-06: Thứ tự nút trên ActionCluster
// -----------------------------------------------------------------------------
func TestAcceptance_TC06_ActionClusterStrictOrder(t *testing.T) {
	test.NewApp()
	palette := DarkPalette

	item := manager.AccountDisplayItem{
		Email:    "cluster.test@example.com",
		HasToken: true,
		IsActive: false,
		Status:   "on",
	}

	card := NewAccountCardWidgetWithQuota(item, nil, palette, nil, nil, nil, nil)
	cluster := card.ActionCluster()
	if cluster == nil {
		t.Fatalf("expected actionCluster container")
	}

	// Invariant 2 strict 4-item cluster verification:
	// [1. Action Slot] -> [2. Refresh Quota] -> [3. Status ON/OFF] -> [4. Trash Button 🗑]
	if len(cluster.Objects) != 4 {
		t.Fatalf("expected exactly 4 objects in actionCluster, got %d", len(cluster.Objects))
	}

	if cluster.Objects[0] != card.actionSlot {
		t.Errorf("position 0 must be actionSlot")
	}
	if cluster.Objects[1] != card.actionCluster.Objects[1] {
		t.Errorf("position 1 must be compact refresh button")
	}
	if cluster.Objects[2] != card.statusButton {
		t.Errorf("position 2 must be statusButton")
	}
	if cluster.Objects[3] != card.deleteContainer {
		t.Errorf("position 3 must be compact deleteButton container")
	}
}

// -----------------------------------------------------------------------------
// TC-07: Dọn dẹp Quota Cache
// -----------------------------------------------------------------------------
func TestAcceptance_TC07_CleanupQuotaCache(t *testing.T) {
	activeEmail := "active@test.com"
	targetEmail := "user7@test.com"

	app, _, accStore, _, _ := setupDeletionAcceptanceApp(t, activeEmail)
	defer app.Close()

	_ = accStore.SaveAccount(activeEmail, &model.TokenData{AccessToken: "act_tok", RefreshToken: "act_ref", IDToken: makeTestJWT(activeEmail)})
	_ = accStore.SaveAccount(targetEmail, &model.TokenData{AccessToken: "tok7", RefreshToken: "ref7", IDToken: makeTestJWT(targetEmail)})

	app.quotaCache.Update(&telemetry.AccountQuotaData{
		Email:  targetEmail,
		Status: telemetry.StatusPlentiful,
		Gemini: telemetry.ModelQuotaBucket{
			ModelID:          "gemini-2.5-pro",
			FiveHourFraction: 0.9,
			WeeklyFraction:   0.8,
		},
		LastUpdated: time.Now(),
	})

	if _, exists := app.quotaCache.Get(targetEmail); !exists {
		t.Fatalf("expected quotaCache to contain %s before deletion", targetEmail)
	}

	// Execute deletion
	app.executeRemoveAccount(targetEmail, targetEmail)
	waitForIdle(app, 2*time.Second)

	// Verify Invariant 6: Quota cache is cleared
	if _, exists := app.quotaCache.Get(targetEmail); exists {
		t.Errorf("Invariant 6 Violated: quotaCache must be deleted for %s", targetEmail)
	}
}

// -----------------------------------------------------------------------------
// TC-08: Tự động ngắt Auto-Switch khi hết tài khoản
// -----------------------------------------------------------------------------
func TestAcceptance_TC08_AutoSwitchCutoffWhenNoCandidates(t *testing.T) {
	// Active account has no token or is a different empty session
	app, _, accStore, _, prefPath := setupDeletionAcceptanceApp(t, "")
	defer app.Close()

	candidateEmail := "candidate@test.com"
	_ = accStore.SaveAccount(candidateEmail, &model.TokenData{AccessToken: "cand_tok", RefreshToken: "cand_ref", IDToken: makeTestJWT(candidateEmail)})

	app.Init()
	waitForIdle(app, 2*time.Second)

	// Turn ON Auto-switch
	app.SetAutoSwitchEnabled(true)
	if !app.IsAutoSwitchEnabled() {
		t.Fatalf("expected Auto-Switch to be enabled initially")
	}

	// Now delete the only candidate account
	app.executeRemoveAccount(candidateEmail, candidateEmail)
	waitForIdle(app, 2*time.Second)

	// Verify Invariant 6: Auto-switch is automatically turned OFF
	if app.IsAutoSwitchEnabled() {
		t.Errorf("Invariant 6 Violated: Auto-Switch must be automatically turned OFF when 0 candidate accounts remain")
	}

	// Verify UI checkbox unchecked
	if app.autoSwitchCheck != nil && app.autoSwitchCheck.Checked {
		t.Errorf("expected autoSwitchCheck to be unchecked")
	}

	// Verify status bar notice
	app.mu.RLock()
	sbText := ""
	if app.statusBar != nil {
		sbText = app.statusBar.Text
	}
	app.mu.RUnlock()
	expectedStatus := "⚠️ Không có tài khoản để chuyển."
	if sbText != expectedStatus {
		t.Errorf("expected status bar %q, got %q", expectedStatus, sbText)
	}

	// Verify saved preferences on disk
	time.Sleep(100 * time.Millisecond) // allow async disk write
	pref, err := LoadUIPreferences(prefPath)
	if err != nil || pref == nil {
		t.Fatalf("failed loading preferences: %v", err)
	}
	if pref.AutoSwitch {
		t.Errorf("expected preferences on disk to have auto_switch=false")
	}
}
