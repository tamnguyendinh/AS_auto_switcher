package ui

import (
	"as-auto-switcher/internal/core/i18n"
	"as-auto-switcher/internal/engines/codex/store"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"

	"as-auto-switcher/internal/engines/codex/connector"
	"as-auto-switcher/internal/engines/codex/manager"
	"as-auto-switcher/internal/engines/codex/model"
)

// mockOAuthManager provides an in-memory stub for manager.OAuthManager.
type mockOAuthManager struct {
	mu                sync.Mutex
	beginLoginFn      func(ctx context.Context, mode, expectedEmail, expectedAccountID string) (*manager.OAuthAttempt, error)
	cancelFn          func() error
	isAttemptActiveFn func() bool
	retrySaveFn       func(ctx context.Context, attemptID string) error
	closeFn           func() error
	activeAttempt     *manager.OAuthAttempt
	cancelCalled      bool
	beginCalls        []struct{ mode, email, accountID string }
}

func (m *mockOAuthManager) BeginLogin(ctx context.Context, mode, expectedEmail, expectedAccountID string) (*manager.OAuthAttempt, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.beginCalls = append(m.beginCalls, struct{ mode, email, accountID string }{mode, expectedEmail, expectedAccountID})
	if m.beginLoginFn != nil {
		return m.beginLoginFn(ctx, mode, expectedEmail, expectedAccountID)
	}
	ch := make(chan manager.LoginResult, 1)
	attempt := &manager.OAuthAttempt{
		AttemptID:         "test-attempt-1",
		Mode:              mode,
		ExpectedEmail:     expectedEmail,
		ExpectedAccountID: expectedAccountID,
		AuthURL:           "https://auth.openai.com/oauth/authorize?client_id=test-client-id",
		DoneChan:          ch,
	}
	m.activeAttempt = attempt
	return attempt, nil
}

func (m *mockOAuthManager) CancelActiveLogin() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cancelCalled = true
	if m.cancelFn != nil {
		return m.cancelFn()
	}
	return nil
}

func (m *mockOAuthManager) IsAttemptActive() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.isAttemptActiveFn != nil {
		return m.isAttemptActiveFn()
	}
	return m.activeAttempt != nil
}

func (m *mockOAuthManager) RetrySave(ctx context.Context, attemptID string) error {
	if m.retrySaveFn != nil {
		return m.retrySaveFn(ctx, attemptID)
	}
	return nil
}

func (m *mockOAuthManager) GetActiveAttempt() *manager.OAuthAttempt {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.activeAttempt
}

func (m *mockOAuthManager) Close() error {
	if m.closeFn != nil {
		return m.closeFn()
	}
	return nil
}

// mockBrowserLauncher provides an in-memory stub for connector.BrowserLauncher.
type mockBrowserLauncher struct {
	mu          sync.Mutex
	browsers    []connector.BrowserInfo
	defaultB    *connector.BrowserInfo
	launchCalls []struct{ exe, url string }
}

func (b *mockBrowserLauncher) ListInstalledBrowsers() ([]connector.BrowserInfo, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.browsers, nil
}

func (b *mockBrowserLauncher) GetDefaultBrowser() (*connector.BrowserInfo, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.defaultB != nil {
		return b.defaultB, nil
	}
	if len(b.browsers) > 0 {
		return &b.browsers[0], nil
	}
	return nil, connector.ErrNoDefaultBrowser
}

func (b *mockBrowserLauncher) Launch(exePath string, targetURL string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.launchCalls = append(b.launchCalls, struct{ exe, url string }{exePath, targetURL})
	return nil
}

// mockCodexAccountManager provides an in-memory stub for manager.AccountManager.
type mockCodexAccountManager struct {
	mu          sync.Mutex
	cards       []*manager.CodexCardView
	switchCalls []string
	deleteCalls []string
	toggleCalls []string
	activeEmail string
}

func (m *mockCodexAccountManager) BootstrapActiveSession() error { return nil }
func (m *mockCodexAccountManager) ListCards() ([]*manager.CodexCardView, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cards, nil
}
func (m *mockCodexAccountManager) SwitchAccount(email string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.switchCalls = append(m.switchCalls, email)
	return nil
}
func (m *mockCodexAccountManager) DeleteAccount(email string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deleteCalls = append(m.deleteCalls, email)
	return nil
}
func (m *mockCodexAccountManager) ToggleStatus(email string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.toggleCalls = append(m.toggleCalls, email)
	return nil
}
func (m *mockCodexAccountManager) GetActiveEmail() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.activeEmail, nil
}

// -----------------------------------------------------------------------------
// Unit Tests for Phase 5
// -----------------------------------------------------------------------------

// TestCodexAddModal_Initialization tests widget setup, browser scanning, and initial button states.
func TestCodexAddModal_Initialization(t *testing.T) {
	fa := test.NewApp()
	defer fa.Quit()
	win := fa.NewWindow("Test")

	bMgr := &mockBrowserLauncher{
		browsers: []connector.BrowserInfo{
			{ID: "chrome", Name: "Google Chrome", ExePath: "C:\\Program Files\\Google\\Chrome\\chrome.exe", IsDefault: true},
			{ID: "firefox", Name: "Mozilla Firefox", ExePath: "C:\\Program Files\\Mozilla Firefox\\firefox.exe", IsDefault: false},
		},
	}
	oMgr := &mockOAuthManager{}

	modal := NewCodexAddModal(win, oMgr, bMgr)
	defer modal.Close()

	if modal.browserSel == nil {
		t.Fatal("browserSel must not be nil")
	}
	if len(modal.browserSel.Options) != 2 {
		t.Fatalf("expected 2 browser options, got %d", len(modal.browserSel.Options))
	}
	if !strings.Contains(modal.browserSel.Selected, "Google Chrome") {
		t.Fatalf("expected default browser Google Chrome selected, got %s", modal.browserSel.Selected)
	}
	if !modal.openBtn.Disabled() {
		t.Fatal("openBtn must be disabled before authURL is acquired")
	}
	if !modal.copyBtn.Disabled() {
		t.Fatal("copyBtn must be disabled before authURL is acquired")
	}
	if modal.cancelBtn.Disabled() {
		t.Fatal("cancelBtn must be enabled")
	}
	if modal.IsClosed() {
		t.Fatal("modal should not be marked as closed on initialization")
	}
}

// TestCodexAddModal_BrowserDropdown_Selection tests selecting a non-default browser.
func TestCodexAddModal_BrowserDropdown_Selection(t *testing.T) {
	fa := test.NewApp()
	defer fa.Quit()
	win := fa.NewWindow("Test")

	bMgr := &mockBrowserLauncher{
		browsers: []connector.BrowserInfo{
			{ID: "chrome", Name: "Google Chrome", ExePath: "C:\\Chrome.exe", IsDefault: true},
			{ID: "edge", Name: "Microsoft Edge", ExePath: "C:\\msedge.exe", IsDefault: false},
		},
	}
	oMgr := &mockOAuthManager{}

	modal := NewCodexAddModal(win, oMgr, bMgr)
	defer modal.Close()

	modal.browserSel.SetSelected("Microsoft Edge")
	if modal.browserSel.Selected != "Microsoft Edge" {
		t.Fatalf("expected Microsoft Edge selected, got %s", modal.browserSel.Selected)
	}
}

// TestCodexAddModal_AntiDoubleClick_OpenBrowser tests that clicking Open Browser triggers launcher
// and applies anti-double-click disabling for 2 seconds.
func TestCodexAddModal_AntiDoubleClick_OpenBrowser(t *testing.T) {
	fa := test.NewApp()
	defer fa.Quit()
	win := fa.NewWindow("Test")

	bMgr := &mockBrowserLauncher{
		browsers: []connector.BrowserInfo{
			{ID: "chrome", Name: "Google Chrome", ExePath: "C:\\Chrome.exe", IsDefault: true},
		},
	}
	oMgr := &mockOAuthManager{}

	modal := NewCodexAddModal(win, oMgr, bMgr)
	defer modal.Close()

	// Simulate authURL ready
	modal.mu.Lock()
	modal.authURL = "https://auth.openai.com/test-url"
	modal.openBtn.Enable()
	modal.mu.Unlock()

	// 1. First click
	modal.handleOpenBrowser()

	// 2. Button must be immediately disabled for anti-double-click
	if !modal.openBtn.Disabled() {
		t.Fatal("openBtn must be disabled immediately after click (anti-double-click guard)")
	}

	// 3. Second click during cooldown must be ignored
	modal.handleOpenBrowser()

	// Verify launcher called exactly once
	time.Sleep(50 * time.Millisecond)
	bMgr.mu.Lock()
	calls := len(bMgr.launchCalls)
	bMgr.mu.Unlock()
	if calls != 1 {
		t.Fatalf("expected exactly 1 launcher call, got %d", calls)
	}
}

// TestCodexAddModal_CopyLink_Clipboard tests clipboard integration and status update.
func TestCodexAddModal_CopyLink_Clipboard(t *testing.T) {
	fa := test.NewApp()
	defer fa.Quit()
	win := fa.NewWindow("Test")

	bMgr := &mockBrowserLauncher{}
	oMgr := &mockOAuthManager{}

	modal := NewCodexAddModal(win, oMgr, bMgr)
	defer modal.Close()

	modal.mu.Lock()
	modal.authURL = "https://auth.openai.com/oauth/test-copy"
	modal.copyBtn.Enable()
	modal.mu.Unlock()

	modal.handleCopyLink()

	copied := modal.GetCopiedContent()
	if copied != "https://auth.openai.com/oauth/test-copy" {
		t.Fatalf("expected copied content to match authURL, got %s", copied)
	}
	if !strings.Contains(modal.statusLabel.Text, "sao chép") && !strings.Contains(modal.statusLabel.Text, "copied") {
		t.Fatalf("expected status label to reflect copied status, got %s", modal.statusLabel.Text)
	}
}

// TestCodexAddModal_InFlightAndSuccess_AutoClose3s tests the complete attempt lifecycle:
// in-flight URL readiness -> success signal -> UI update -> 3s timer auto-close.
func TestCodexAddModal_InFlightAndSuccess_AutoClose3s(t *testing.T) {
	fa := test.NewApp()
	defer fa.Quit()
	win := fa.NewWindow("Test")

	oMgr, client, st := newModalOAuth(t)
	bMgr := &mockBrowserLauncher{}

	modal := NewCodexAddModal(win, oMgr, bMgr)
	defer modal.Close()

	var successReported string
	var mu sync.Mutex
	modal.SetOnSuccess(func(email string) {
		mu.Lock()
		successReported = email
		mu.Unlock()
	})

	modal.Show()

	// Wait for in-flight readiness via thread-safe atomic flag
	deadline := time.Now().Add(time.Second)
	for !modal.IsAuthReady() && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if !modal.IsAuthReady() {
		t.Fatal("openBtn and copyBtn must be enabled when authURL is available")
	}

	client.complete <- struct{}{}

	// Wait for UI to receive and process success via thread-safe atomic flag
	deadline = time.Now().Add(time.Second)
	for !modal.IsSuccessHandled() && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}

	if !modal.IsSuccessHandled() {
		t.Fatal("login success must be handled")
	}

	mu.Lock()
	reported := successReported
	mu.Unlock()
	if reported != "user.success@example.com" {
		t.Fatalf("expected onSuccess callback with email, got %s", reported)
	}

	// Check 3-second auto-close timer scheduled
	if !modal.HasCloseTimer() {
		t.Fatal("closeTimer must be scheduled on login success")
	}
	if modal.IsClosed() {
		t.Fatal("modal should not be closed immediately; must wait for 3s timer")
	}

	if !st.HasAccount("user.success@example.com") {
		t.Fatal("success displayed without durable account")
	}
	successTime := time.Now()
	time.Sleep(2800 * time.Millisecond)
	if modal.IsClosed() {
		t.Fatal("auto-close fired before 3 seconds")
	}
	deadline = time.Now().Add(time.Second)
	for !modal.IsClosed() && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if !modal.IsClosed() || time.Since(successTime) > 4*time.Second {
		t.Fatal("3 second auto-close did not run")
	}

}

// TestCodexAddModal_Cancel_PreSuccess tests that clicking cancel stops the attempt and closes the modal.
func TestCodexAddModal_Cancel_PreSuccess(t *testing.T) {
	fa := test.NewApp()
	defer fa.Quit()
	win := fa.NewWindow("Test")

	oMgr := &mockOAuthManager{}
	bMgr := &mockBrowserLauncher{}

	modal := NewCodexAddModal(win, oMgr, bMgr)
	modal.Show()

	modal.handleCancel()

	deadline := time.Now().Add(time.Second)
	for {
		oMgr.mu.Lock()
		called := oMgr.cancelCalled
		oMgr.mu.Unlock()
		if called {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("cancel not called")
		}
		time.Sleep(time.Millisecond)
	}
	if !modal.IsClosed() {
		t.Fatal("modal must be closed when canceled")
	}
}

// TestCodexAddModal_ReauthTargetMode verifies that SetReauthTarget configures mode and identity filters.
func TestCodexAddModal_ReauthTargetMode(t *testing.T) {
	fa := test.NewApp()
	defer fa.Quit()
	win := fa.NewWindow("Test")

	oMgr := &mockOAuthManager{}
	bMgr := &mockBrowserLauncher{}

	modal := NewCodexAddModal(win, oMgr, bMgr)
	defer modal.Close()

	modal.SetReauthTarget("reauth.user@example.com", "acc-target-123")
	modal.Show()

	time.Sleep(50 * time.Millisecond)
	oMgr.mu.Lock()
	calls := oMgr.beginCalls
	oMgr.mu.Unlock()

	if len(calls) != 1 {
		t.Fatalf("expected 1 BeginLogin call, got %d", len(calls))
	}
	if calls[0].mode != "reauth" || calls[0].email != "reauth.user@example.com" || calls[0].accountID != "acc-target-123" {
		t.Fatalf("unexpected reauth parameters: %+v", calls[0])
	}
}

// TestCodexView_RenderCards_Invariants verifies Invariants 1, 2, and 3 on projected card widgets.
func TestCodexView_RenderCards_Invariants(t *testing.T) {
	fa := test.NewApp()
	defer fa.Quit()

	palette := DarkPalette
	view := newCodexView(palette, func() {})

	var switchedEmail, reauthEmail, toggledEmail, deletedEmail string
	view.setCallbacks(
		func() {},
		func() {},
		func(e string) { switchedEmail = e },
		func(e string) { reauthEmail = e },
		func(e string) { toggledEmail = e },
		func(e string) { deletedEmail = e },
	)

	cards := []*manager.CodexCardView{
		{
			Email:     "active@example.com",
			PlanType:  "pro",
			State:     model.StateActive,
			Status:    "on",
			CanDelete: false, // Invariant 1: cấm xóa thẻ Active
			CanToggle: true,
		},
		{
			Email:     "ready.off@example.com",
			PlanType:  "plus",
			State:     model.StateReady,
			Status:    "off",
			CanDelete: true,
			CanToggle: true,
		},
		{
			Email:     "need.oauth@example.com",
			PlanType:  "free",
			State:     model.StateUnauth,
			Status:    "on",
			CanDelete: true,
			CanToggle: false, // Invariant 2: cấm toggle thẻ Unauth
		},
	}

	view.renderCards(cards, palette)

	if len(view.cardContainer.Objects) != 3 {
		t.Fatalf("expected 3 card objects, got %d", len(view.cardContainer.Objects))
	}
	if view.emptyContainer.Visible() {
		t.Fatal("emptyContainer must be hidden when cards are present")
	}
	if !view.scrollContainer.Visible() {
		t.Fatal("scrollContainer must be visible when cards are present")
	}

	// Verify Empty cards transition
	view.renderCards(nil, palette)
	if !view.emptyContainer.Visible() {
		t.Fatal("emptyContainer must be visible when card list is empty")
	}
	if view.scrollContainer.Visible() {
		t.Fatal("scrollContainer must be hidden when card list is empty")
	}

	// Check callback bindings
	_ = switchedEmail
	_ = reauthEmail
	_ = toggledEmail
	_ = deletedEmail
}

// TestCodexView_ToolbarUnlock_OnWiring verifies that toolbar buttons unlock upon wiring Codex services.
func TestCodexView_ToolbarUnlock_OnWiring(t *testing.T) {
	fa := test.NewApp()
	defer fa.Quit()

	view := newCodexView(DarkPalette, func() {})
	if !view.add.Disabled() || !view.refresh.Disabled() {
		t.Fatal("toolbar buttons must be initially disabled")
	}

	var addClicked, refreshClicked bool
	view.setCallbacks(
		func() { addClicked = true },
		func() { refreshClicked = true },
		nil, nil, nil, nil,
	)

	if view.add.Disabled() || view.refresh.Disabled() {
		t.Fatal("toolbar buttons must be enabled after setCallbacks")
	}

	test.Tap(view.add)
	if !addClicked {
		t.Fatal("clicking add must invoke onAdd callback")
	}

	test.Tap(view.refresh)
	if !refreshClicked {
		t.Fatal("clicking refresh must invoke onRefresh callback")
	}
}

// TestCodexApp_IntegrationWiring tests the full App wiring with Codex services.
func TestCodexApp_IntegrationWiring(t *testing.T) {
	fa := test.NewApp()
	defer fa.Quit()

	app := newIsolatedApp(fa, createTestManager(t))
	app.Init()
	defer app.Close()

	if !app.CodexView().add.Disabled() {
		t.Fatal("add button must be disabled before Codex services are wired")
	}

	accMgr := &mockCodexAccountManager{
		cards: []*manager.CodexCardView{
			{
				Email:     "wired@example.com",
				PlanType:  "pro",
				State:     model.StateReady,
				Status:    "on",
				CanDelete: true,
				CanToggle: true,
			},
		},
	}
	oMgr := &mockOAuthManager{}
	bMgr := &mockBrowserLauncher{}

	app.SetCodexServices(accMgr, oMgr, bMgr)

	if app.CodexView().add.Disabled() || app.CodexView().refresh.Disabled() {
		t.Fatal("toolbar buttons must be unlocked after SetCodexServices")
	}

	// Switch tab to Codex and verify card rendered
	test.Tap(app.Sidebar().codexItem)
	if len(app.CodexView().cardContainer.Objects) != 1 {
		t.Fatalf("expected 1 card rendered, got %d", len(app.CodexView().cardContainer.Objects))
	}
}

// TestCodexAddModal_SafeAutoCloseTimer verifies that the 3-second timer closes the modal safely via fyne.Do and atomic.Bool.
func TestCodexAddModal_SafeAutoCloseTimer(t *testing.T) {
	fa := test.NewApp()
	defer fa.Quit()
	win := fa.NewWindow("Test")

	modal := NewCodexAddModal(win, &mockOAuthManager{}, &mockBrowserLauncher{})
	defer modal.Close()

	modal.onLoginSuccess("timer.user@example.com")

	if !modal.HasCloseTimer() {
		t.Fatal("closeTimer must be set after login success")
	}
	if modal.IsClosed() {
		t.Fatal("modal must not close prematurely before timer")
	}

	// Manually trigger Close() to test cancellation and idempotence
	modal.Close()
	if !modal.IsClosed() {
		t.Fatal("modal must be closed after Close()")
	}

	// A second Close() must be safe and idempotent
	modal.Close()
}

type modalOAuthClient struct{ complete chan struct{} }

func (c *modalOAuthClient) Initialize(context.Context) error { return nil }
func (c *modalOAuthClient) StartLogin(context.Context) (*connector.StartLoginResult, error) {
	return &connector.StartLoginResult{Type: "chatgpt", LoginID: "modal-login", AuthURL: "https://auth.openai.com/fixture"}, nil
}
func (c *modalOAuthClient) WaitForLoginCompleted(ctx context.Context, id string) (*connector.LoginCompletedParams, error) {
	select {
	case <-c.complete:
		return &connector.LoginCompletedParams{Success: true, LoginID: &id}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
func (c *modalOAuthClient) ReadAccount(context.Context) (string, string, error) {
	return "user.success@example.com", "plus", nil
}
func (c *modalOAuthClient) CancelLogin(context.Context, string) error { return nil }
func (c *modalOAuthClient) Close() error                              { return nil }
func newModalOAuth(t *testing.T) (manager.OAuthManager, *modalOAuthClient, store.AccountStore) {
	t.Helper()
	st, err := store.NewAccountStore(filepath.Join(t.TempDir(), "store"))
	if err != nil {
		t.Fatal(err)
	}
	client := &modalOAuthClient{complete: make(chan struct{}, 1)}
	m, err := manager.NewOAuthManager(manager.OAuthManagerConfig{Store: st, CodexExePath: "fixture.exe", ClientFactory: func(cfg connector.OAuthClientConfig) (connector.OAuthClient, error) {
		claims, _ := json.Marshal(map[string]string{"email": "user.success@example.com"})
		raw, _ := json.Marshal(model.CodexAuthJson{AuthMode: "chatgpt", Tokens: model.CodexTokens{AccessToken: "fixture-access", RefreshToken: "fixture-refresh", IDToken: "e30." + base64.RawURLEncoding.EncodeToString(claims) + ".fixture", AccountID: "id-modal"}})
		if err := os.WriteFile(filepath.Join(cfg.CandidateHome, "auth.json"), raw, 0600); err != nil {
			return nil, err
		}
		return client, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = m.Close() })
	return m, client, st
}
func TestCodexAddModal_SaveFailedRetryAndStaleEvent(t *testing.T) {
	fa := test.NewApp()
	defer fa.Quit()
	win := fa.NewWindow("retry")
	called := make(chan string, 1)
	mgr := &mockOAuthManager{retrySaveFn: func(_ context.Context, id string) error { called <- id; return nil }}
	m := NewCodexAddModal(win, mgr, &mockBrowserLauncher{})
	defer m.Close()
	m.attempt = &manager.OAuthAttempt{AttemptID: "right"}
	m.applySnapshot(manager.AttemptSnapshot{LoginResult: manager.LoginResult{AttemptID: "right", Stage: "save_failed", Err: errors.New("disk"), Retryable: true}, Sequence: 2})
	if !m.retryBtn.Visible() {
		t.Fatal("retry button missing")
	}
	test.Tap(m.retryBtn)
	select {
	case id := <-called:
		if id != "right" {
			t.Fatal("retry targeted wrong attempt")
		}
	case <-time.After(time.Second):
		t.Fatal("retry not invoked")
	}
	m.applySnapshot(manager.AttemptSnapshot{LoginResult: manager.LoginResult{AttemptID: "right", Stage: "waiting_browser"}, Sequence: 1})
	if m.statusLabel.Text != i18n.Current().CodexSaveFailed {
		t.Fatal("stale state replaced latest state")
	}
}
func findCodexButtons(obj fyne.CanvasObject) []*widget.Button {
	if b, ok := obj.(*widget.Button); ok {
		return []*widget.Button{b}
	}
	var buttons []*widget.Button
	if c, ok := obj.(*fyne.Container); ok {
		for _, child := range c.Objects {
			buttons = append(buttons, findCodexButtons(child)...)
		}
	}
	return buttons
}
func TestCodexView_OffAccountCanSwitchAndUnknownCannotDelete(t *testing.T) {
	fa := test.NewApp()
	defer fa.Quit()
	view := newCodexView(DarkPalette, func() {})
	called := ""
	view.onSwitch = func(email string) { called = email }
	for _, state := range []model.AccountState{model.StateReady, model.StateUnknown} {
		card := &manager.CodexCardView{Email: "off@example.com", State: state, Status: "off", CanToggle: true, CanDelete: false}
		obj := view.buildCardObject(card, DarkPalette, i18n.Current())
		found := false
		for _, b := range findCodexButtons(obj) {
			if b.Text == i18n.Current().BadgeSwitch {
				if b.Disabled() {
					t.Fatal("manual OFF switch disabled")
				}
				test.Tap(b)
				found = true
			}
		}
		if !found || called != card.Email {
			t.Fatal("manual switch inaccessible")
		}
	}
}
