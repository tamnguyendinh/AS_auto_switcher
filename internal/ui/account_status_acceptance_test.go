package ui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"antigravity-switch/internal/automation"
	"antigravity-switch/internal/connector"
	"antigravity-switch/internal/manager"
	"antigravity-switch/internal/model"
	"antigravity-switch/internal/reloader"
	"antigravity-switch/internal/store"
	"antigravity-switch/internal/telemetry"
)

// acceptanceQuotaClient provides configurable quota responses for acceptance testing.
type acceptanceQuotaClient struct {
	quotaFn func(email string) (*telemetry.AccountQuotaData, error)
}

func (c *acceptanceQuotaClient) RetrieveUserQuota(ctx context.Context, refreshToken string) (*telemetry.AccountQuotaData, error) {
	email := strings.TrimPrefix(refreshToken, "ref_")
	if c.quotaFn != nil {
		return c.quotaFn(email)
	}
	return &telemetry.AccountQuotaData{
		Email:  email,
		Status: telemetry.StatusPlentiful,
		Gemini: telemetry.ModelQuotaBucket{
			ModelID:          "gemini-2.5-pro",
			FiveHourFraction: 0.85,
			WeeklyFraction:   0.60,
		},
		LastUpdated: time.Now(),
	}, nil
}

type statusAcceptanceOAuthRefresher struct{}

func (statusAcceptanceOAuthRefresher) RefreshToken(ctx context.Context, refreshToken string) (*connector.TokenRefreshResult, error) {
	return &connector.TokenRefreshResult{
		AccessToken:  "refreshed_access_" + refreshToken,
		RefreshToken: refreshToken,
		IDToken:      makeTestJWT("refreshed@example.com"),
		ExpiresIn:    3600,
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(1 * time.Hour),
	}, nil
}

// setupStatusAcceptanceApp constructs a fully wired, isolated test app in t.TempDir().
func setupStatusAcceptanceApp(
	t *testing.T,
	initialActiveEmail string,
	accountsWithStatus map[string]string,
	mockQuotaFn func(email string) (*telemetry.AccountQuotaData, error),
) (*App, *manager.AccountManager, *store.AccountStore, string, *httptest.Server) {
	t.Helper()

	testApp := test.NewApp()
	tempDir := t.TempDir()
	storeDir := filepath.Join(tempDir, "accounts_store")
	prefPath := filepath.Join(tempDir, "ui_preferences.json")

	accStore, err := store.NewAccountStore(storeDir)
	if err != nil {
		t.Fatalf("failed initializing AccountStore: %v", err)
	}

	var activeMu sync.Mutex
	currentActiveEmail := initialActiveEmail
	currentCredToken := &model.TokenData{
		AccessToken:  "tok_" + initialActiveEmail,
		RefreshToken: "ref_" + initialActiveEmail,
		IDToken:      makeTestJWT(initialActiveEmail),
	}

	// 1. Mock HTTP Server handling both language server model config and identity reflection
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == connector.EndpointCascadeModelConfigData {
			resp := connector.CascadeModelConfigData{
				ClientModelConfigs: []connector.ClientModelConfig{
					{
						ModelOrAlias: connector.ModelOrAlias{Model: "gemini-2.5-pro", Alias: connector.DefaultActiveAgentModel},
						Label:        "Gemini 2.5 Pro",
						Disabled:     false,
						IsSelected:   true,
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		activeMu.Lock()
		email := currentActiveEmail
		activeMu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"email":"%s"}`, email)))
	}))
	t.Cleanup(testServer.Close)

	testAddress, _ := url.Parse(testServer.URL)
	testPort, _ := strconv.Atoi(testAddress.Port())
	testClient := connector.NewClientWithHTTP(&connector.ServerInfo{
		PID:       12345,
		Port:      testPort,
		CSRFToken: "test-csrf",
	}, testServer.Client())

	// 2. Mock Reloader
	sessionReloader := reloader.NewSessionReloader()
	sessionReloader.StoragePath = filepath.Join(tempDir, "app_storage.json")
	sessionReloader.ProcessKiller = func(ctx context.Context) error { return nil }
	sessionReloader.Sleeper = func(ctx context.Context, d time.Duration) error { return nil }
	sessionReloader.SnapshotCapturer = func(string, string, string) (string, []reloader.ActiveThread, error) { return "", nil, nil }
	sessionReloader.ServerDiscoverer = func() (*connector.ServerInfo, error) {
		return &connector.ServerInfo{PID: 12345, Port: testPort, CSRFToken: "test-csrf"}, nil
	}

	// 3. Populate accounts in store
	for email, st := range accountsWithStatus {
		addTestAccount(t, accStore, email, st)
	}

	// 4. Create AccountManager
	mgr, err := manager.NewAccountManager(accStore, sessionReloader)
	if err != nil {
		t.Fatalf("failed initializing AccountManager: %v", err)
	}
	mgr.SetOAuthRefresher(statusAcceptanceOAuthRefresher{})
	mgr.SetWincredHandlers(
		func() (*model.TokenData, error) {
			activeMu.Lock()
			defer activeMu.Unlock()
			return currentCredToken, nil
		},
		func(token *model.TokenData) error {
			activeMu.Lock()
			defer activeMu.Unlock()
			if token != nil {
				currentCredToken = token
				if email := manager.ExtractEmailFromToken(token); email != "" {
					currentActiveEmail = email
				}
			}
			return nil
		},
	)
	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return testClient, nil
	})

	// 5. Create App
	app := NewAppWithFyne(testApp, mgr)
	app.SetPreferencesPath(prefPath)

	// Configure TelemetryPool with acceptance client
	client := &acceptanceQuotaClient{quotaFn: mockQuotaFn}
	pool, pErr := telemetry.NewTelemetryPool(client, app.quotaCache, accStore)
	if pErr != nil {
		t.Fatalf("failed creating TelemetryPool: %v", pErr)
	}
	pool.SetLocalQuotaFetcher(nil)
	app.SetTelemetryPool(pool)
	app.autoLoop, _ = telemetry.NewAutoTelemetryLoop(pool, telemetry.DefaultLoopInterval)

	// Configure AutomationController
	gate := automation.NewSwitchGate(fmt.Sprintf("Local\\TestStatusAcceptance_%d", time.Now().UnixNano()))
	app.controller = automation.NewAutomationController(mgr, pool, app.quotaCache, gate)
	app.controller.SetClientFinder(func(context.Context) (*connector.Client, error) {
		return testClient, nil
	})
	app.controller.SetPbtxtPath(filepath.Join(tempDir, "missing.pbtxt"))
	app.controller.RetryCoordinator().SetConversationsScanner(func(context.Context) ([]string, error) { return nil, nil })

	// Seed quota cache with plentiful quota so candidate evaluations pass
	for email := range accountsWithStatus {
		app.quotaCache.Update(&telemetry.AccountQuotaData{
			Email:  email,
			Status: telemetry.StatusPlentiful,
			Gemini: telemetry.ModelQuotaBucket{
				ModelID:          "gemini-2.5-pro",
				FiveHourFraction: 0.85,
				WeeklyFraction:   0.60,
			},
			LastUpdated: time.Now(),
		})
	}

	// Initialize UI
	app.Init()
	waitForIdle(app, 2*time.Second)

	return app, mgr, accStore, prefPath, testServer
}

// -----------------------------------------------------------------------------
// SECTION 1: END-TO-END (E2E) INTEGRATION ACCEPTANCE TESTS
// -----------------------------------------------------------------------------

// TestE2E_AccountStatus_ActiveTurnedOff_AutoSwitchesToEligibleCandidate verifies:
// Active account is toggled to OFF on UI -> Auto-Switch is enabled ->
// Controller filters out OFF accounts -> Discovers eligible ON candidate ->
// Switches active account to ON candidate -> Auto-Switch remains enabled ->
// UI card for previous account updates to [OFF], new active account displays active badge.
func TestE2E_AccountStatus_ActiveTurnedOff_AutoSwitchesToEligibleCandidate(t *testing.T) {
	accounts := map[string]string{
		"active@gmail.com":   "on",
		"disabled@gmail.com": "off",
		"eligible@gmail.com": "on",
	}

	app, mgr, accStore, _, _ := setupStatusAcceptanceApp(t, "active@gmail.com", accounts, nil)
	defer app.Close()

	// Turn ON Auto-Switch
	app.SetAutoSwitchEnabled(true)
	if !app.IsAutoSwitchEnabled() {
		t.Fatalf("expected auto-switch to be enabled")
	}

	// Verify initial state: active@gmail.com is active
	activeEmail, err := mgr.GetCurrentActiveEmail(context.Background())
	if err != nil || activeEmail != "active@gmail.com" {
		t.Fatalf("expected initial active account active@gmail.com, got %s (err: %v)", activeEmail, err)
	}

	// TRIGGER: User clicks [ON/OFF] button on the active account card to turn it OFF
	app.ToggleAccountStatus("active@gmail.com")

	// Wait for the async switch cycle to complete (active email becomes eligible@gmail.com)
	deadline := time.Now().Add(4 * time.Second)
	switched := false
	for time.Now().Before(deadline) {
		active, _ := mgr.GetCurrentActiveEmail(context.Background())
		if active == "eligible@gmail.com" {
			switched = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !switched {
		t.Fatalf("timed out waiting for auto-switch to eligible@gmail.com")
	}
	waitForIdle(app, 1*time.Second)

	// 1. Verify previous active account is now OFF in store
	stActive, err := accStore.GetAccountStatus("active@gmail.com")
	if err != nil || stActive != "off" {
		t.Errorf("expected active@gmail.com status to become 'off', got %s (err: %v)", stActive, err)
	}

	// 2. Verify disabled standby remains OFF
	stDisabled, err := accStore.GetAccountStatus("disabled@gmail.com")
	if err != nil || stDisabled != "off" {
		t.Errorf("expected disabled@gmail.com status to remain 'off', got %s", stDisabled)
	}

	// 3. Verify eligible standby remains ON
	stEligible, err := accStore.GetAccountStatus("eligible@gmail.com")
	if err != nil || stEligible != "on" {
		t.Errorf("expected eligible@gmail.com status to be 'on', got %s", stEligible)
	}

	// 4. Verify system successfully switched active account to eligible@gmail.com
	currentActive, err := mgr.GetCurrentActiveEmail(context.Background())
	if err != nil || currentActive != "eligible@gmail.com" {
		t.Errorf("expected new active account 'eligible@gmail.com', got '%s' (err: %v)", currentActive, err)
	}

	// 5. Verify Auto-Switch remains ENABLED (safety stop must NOT be triggered)
	if !app.IsAutoSwitchEnabled() {
		t.Errorf("expected Auto-Switch to remain enabled after successful switch")
	}

	// 6. Verify in-place UI card updates (allow background updateUI to refresh cards)
	var activeCard, eligibleCard *AccountCardWidget
	cardDeadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(cardDeadline) {
		for _, c := range app.AccountCards() {
			if c == nil {
				continue
			}
			if c.Email == "active@gmail.com" {
				activeCard = c
			} else if c.Email == "eligible@gmail.com" && c.IsActive {
				eligibleCard = c
			}
		}
		if activeCard != nil && eligibleCard != nil && eligibleCard.IsActive {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if activeCard == nil {
		t.Fatalf("card for active@gmail.com not found")
	}
	if eligibleCard == nil {
		t.Fatalf("card for eligible@gmail.com with IsActive==true not found")
	}

	// activeCard must show [OFF] and disabled switch button
	if activeCard.Status != "off" {
		t.Errorf("expected activeCard Status 'off', got %s", activeCard.Status)
	}
	if activeCard.statusButton.LabelText() != "[OFF]" {
		t.Errorf("expected activeCard label '[OFF]', got %s", activeCard.statusButton.LabelText())
	}
	if switchBtn, ok := activeCard.ActionButton().(*switchTextButton); ok && !switchBtn.IsDisabled() {
		t.Errorf("expected switch button on activeCard to be disabled/muted")
	}

	// eligibleCard must be active
	if !eligibleCard.IsActive {
		t.Errorf("expected eligibleCard.IsActive == true")
	}
	if eligibleCard.Status != "on" {
		t.Errorf("expected eligibleCard Status 'on', got %s", eligibleCard.Status)
	}
}

// TestE2E_AccountStatus_ActiveTurnedOff_NoOtherCandidate_SafetyStop verifies:
// Active account is toggled to OFF on UI -> Auto-Switch is enabled ->
// All other standby accounts are OFF (no candidate available) ->
// Immediate Safety Stop cycle: autoSwitchEnabled=false, toggle turned OFF,
// preferences saved with auto_switch: false, status bar displays '⚠️ Không có tài khoản để chuyển.'.
func TestE2E_AccountStatus_ActiveTurnedOff_NoOtherCandidate_SafetyStop(t *testing.T) {
	accounts := map[string]string{
		"active@gmail.com":   "on",
		"disabled@gmail.com": "off",
	}

	app, _, accStore, prefPath, _ := setupStatusAcceptanceApp(t, "active@gmail.com", accounts, nil)
	defer app.Close()

	// Turn ON Auto-Switch and inject check widget
	app.SetAutoSwitchEnabled(true)
	checkWidget := widget.NewCheck("Auto Switch", nil)
	checkWidget.SetChecked(true)
	app.SetAutoSwitchCheck(checkWidget)

	// TRIGGER: User clicks [ON/OFF] to turn OFF the active account
	app.ToggleAccountStatus("active@gmail.com")

	// 1. Verify active account status persisted as 'off'
	st, err := accStore.GetAccountStatus("active@gmail.com")
	if err != nil || st != "off" {
		t.Fatalf("expected active@gmail.com status to become 'off', got %s", st)
	}

	// 2. Safety Stop: autoSwitchEnabled must be false
	if app.IsAutoSwitchEnabled() {
		t.Errorf("expected Safety Stop: autoSwitchEnabled == false")
	}

	// 3. Safety Stop: UI toggle and check must be unchecked/off
	if app.autoSwitchToggle != nil && app.autoSwitchToggle.IsOn() {
		t.Errorf("expected autoSwitchToggle.IsOn() == false")
	}
	if checkWidget.Checked {
		t.Errorf("expected checkWidget.Checked == false")
	}

	// 4. Safety Stop: Standardized status message
	if app.StatusBarText() != "⚠️ Không có tài khoản để chuyển." {
		t.Errorf("expected status bar '⚠️ Không có tài khoản để chuyển.', got '%s'", app.StatusBarText())
	}

	// 5. Safety Stop: Disk preferences persisted with auto_switch: false
	deadline := time.Now().Add(2 * time.Second)
	var persistedPref *UIPreferences
	for time.Now().Before(deadline) {
		p, loadErr := LoadUIPreferences(prefPath)
		if loadErr == nil && p != nil && !p.AutoSwitch {
			persistedPref = p
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if persistedPref == nil || persistedPref.AutoSwitch {
		t.Errorf("expected persisted auto_switch == false in %s, got %#v", prefPath, persistedPref)
	}
}

// TestE2E_AccountStatus_ActiveTurnedOff_PreflightFails_SafetyStop verifies:
// Standby account is ON, but fails preflight check during controller evaluation ->
// Controller halts with StatusNoCandidate -> UI triggers Safety Stop cycle.
func TestE2E_AccountStatus_ActiveTurnedOff_PreflightFails_SafetyStop(t *testing.T) {
	accounts := map[string]string{
		"active@gmail.com":  "on",
		"standby@gmail.com": "on",
	}

	// Standby account fails preflight
	mockQuotaFn := func(email string) (*telemetry.AccountQuotaData, error) {
		if email == "standby@gmail.com" {
			return nil, errors.New("simulated preflight token expired")
		}
		return &telemetry.AccountQuotaData{
			Email:  email,
			Status: telemetry.StatusPlentiful,
			Gemini: telemetry.ModelQuotaBucket{FiveHourFraction: 0.85, WeeklyFraction: 0.50},
		}, nil
	}

	app, _, accStore, prefPath, _ := setupStatusAcceptanceApp(t, "active@gmail.com", accounts, mockQuotaFn)
	defer app.Close()

	app.SetAutoSwitchEnabled(true)

	// TRIGGER: Turn OFF active account
	app.ToggleAccountStatus("active@gmail.com")

	// Wait for preflight failure, Safety Stop event dispatch, and preferences persistence
	deadline := time.Now().Add(3 * time.Second)
	stopped := false
	var persistedPref *UIPreferences
	for time.Now().Before(deadline) {
		if !app.IsAutoSwitchEnabled() && app.StatusBarText() == "⚠️ Không có tài khoản để chuyển." {
			p, loadErr := LoadUIPreferences(prefPath)
			if loadErr == nil && p != nil && !p.AutoSwitch {
				stopped = true
				persistedPref = p
				break
			}
		}
		time.Sleep(30 * time.Millisecond)
	}

	if !stopped {
		t.Fatalf("expected Safety Stop and auto_switch: false persisted after preflight failure, autoSwitchEnabled=%v, statusBar=%q, pref=%#v",
			app.IsAutoSwitchEnabled(), app.StatusBarText(), persistedPref)
	}

	// Verify persistence of 'off' status
	st, _ := accStore.GetAccountStatus("active@gmail.com")
	if st != "off" {
		t.Errorf("expected active account status 'off', got %s", st)
	}
}

// -----------------------------------------------------------------------------
// SECTION 2: 7 SYSTEM INVARIANTS EXPLICIT ACCEPTANCE TESTS
// -----------------------------------------------------------------------------

// Invariant 1: Invariant-Account-Status-Persistence
// Trạng thái Bật/Tắt của tài khoản phải được lưu bền vững vào chính tệp accounts_store/<email>.json
// dưới dạng trường "status": "on" hoặc "status": "off". Khởi động lại bắt buộc khôi phục 100%.
func TestInvariant_1_AccountStatus_Persistence(t *testing.T) {
	tempDir := t.TempDir()
	storeDir := filepath.Join(tempDir, "accounts_store")
	st, err := store.NewAccountStore(storeDir)
	if err != nil {
		t.Fatalf("failed creating store: %v", err)
	}

	email := "persist_test@example.com"
	tok := &model.TokenData{AccessToken: "tok", RefreshToken: "ref"}
	if err := st.SaveAccount(email, tok); err != nil {
		t.Fatalf("SaveAccount failed: %v", err)
	}

	// 1. Initial status is "on"
	val, err := st.GetAccountStatus(email)
	if err != nil || val != "on" {
		t.Fatalf("expected initial 'on', got %s", val)
	}

	// 2. Set to "off"
	if err := st.SetAccountStatus(email, "off"); err != nil {
		t.Fatalf("SetAccountStatus('off') failed: %v", err)
	}

	// 3. Inspect raw JSON file on disk
	filePath, _ := store.SafeAccountPath(storeDir, email)
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed reading file from disk: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if raw["status"] != "off" {
		t.Errorf("expected raw json status == 'off', got %v", raw["status"])
	}

	// 4. Simulate app restart with a fresh store instance pointing to same directory
	freshStore, err := store.NewAccountStore(storeDir)
	if err != nil {
		t.Fatalf("failed recreating store: %v", err)
	}
	restoredStatus, err := freshStore.GetAccountStatus(email)
	if err != nil || restoredStatus != "off" {
		t.Fatalf("expected restored status 'off' after restart, got %s", restoredStatus)
	}
}

// Invariant 2: Invariant-Backward-Compatibility-Default-On
// Mọi tệp tài khoản hiện hữu chưa có trường status phải luôn được nạp với giá trị mặc định là "on".
func TestInvariant_2_BackwardCompatibility_DefaultOn(t *testing.T) {
	tempDir := t.TempDir()
	storeDir := filepath.Join(tempDir, "accounts_store")
	if err := os.MkdirAll(storeDir, 0700); err != nil {
		t.Fatalf("failed creating store dir: %v", err)
	}

	legacyEmail := "legacy_account@example.com"
	legacyFile := filepath.Join(storeDir, legacyEmail+".json")
	legacyJSON := `{
		"email": "legacy_account@example.com",
		"saved_at": 1700000000.0,
		"token_data": {
			"access_token": "legacy_token",
			"refresh_token": "legacy_ref"
		}
	}`
	if err := os.WriteFile(legacyFile, []byte(legacyJSON), 0644); err != nil {
		t.Fatalf("failed writing legacy file: %v", err)
	}

	st, err := store.NewAccountStore(storeDir)
	if err != nil {
		t.Fatalf("failed creating store: %v", err)
	}

	// Store layer defaults to "on"
	status, err := st.GetAccountStatus(legacyEmail)
	if err != nil || status != "on" {
		t.Errorf("expected legacy account status 'on', got %s (err: %v)", status, err)
	}

	// UI Widget layer defaults to "on" when status is empty
	item := manager.AccountDisplayItem{
		Email:    legacyEmail,
		HasToken: true,
		Status:   "", // empty
	}
	card := NewAccountCardWidgetWithQuota(item, nil, DarkPalette, nil, nil)
	if card.Status != "on" {
		t.Errorf("expected card.Status 'on', got %s", card.Status)
	}
	if card.statusButton.LabelText() != "[ON]" {
		t.Errorf("expected statusButton '[ON]', got %s", card.statusButton.LabelText())
	}
}

// Invariant 3: Invariant-Token-Update-Preservation
// Cập nhật token OAuth / refresh token / capture credential từ wincred bắt buộc giữ nguyên "off".
func TestInvariant_3_TokenUpdate_Preservation(t *testing.T) {
	tempDir := t.TempDir()
	st, _ := store.NewAccountStore(tempDir)
	email := "user_preserve@example.com"

	// 1. Save initially
	_ = st.SaveAccount(email, &model.TokenData{AccessToken: "old_tok", RefreshToken: "old_ref"})

	// 2. Set status to "off"
	if err := st.SetAccountStatus(email, "off"); err != nil {
		t.Fatalf("SetAccountStatus failed: %v", err)
	}

	// 3. Overwrite token with fresh token data (simulating background token refresh)
	freshToken := &model.TokenData{AccessToken: "new_refreshed_access", RefreshToken: "new_refreshed_refresh"}
	if err := st.SaveAccount(email, freshToken); err != nil {
		t.Fatalf("SaveAccount failed: %v", err)
	}

	// 4. Status must remain "off"
	status, err := st.GetAccountStatus(email)
	if err != nil || status != "off" {
		t.Fatalf("Invariant-Token-Update-Preservation VIOLATED: expected 'off', got %s", status)
	}
}

// Invariant 4: Invariant-Unauth-Card-Isolation
// Thẻ tài khoản chưa có credential ([Cần OAuth]) tuyệt đối KHÔNG hiển thị nút [ON/OFF].
func TestInvariant_4_UnauthCard_Isolation(t *testing.T) {
	test.NewApp()
	item := manager.AccountDisplayItem{
		Email:    "unauth@gmail.com",
		HasToken: false,
		IsActive: false,
		Status:   "on",
	}

	card := NewAccountCardWidgetWithQuota(item, nil, DarkPalette, nil, nil)

	// statusButton must be strictly hidden
	if card.statusButton.Visible() {
		t.Errorf("Invariant-Unauth-Card-Isolation VIOLATED: statusButton is visible on [Cần OAuth] card")
	}

	// Refresh button must be disabled
	if !card.refreshButton.Disabled() {
		t.Errorf("expected refreshButton disabled on [Cần OAuth] card")
	}
}

// Invariant 5: Invariant-Disabled-Switch-Lockout
// Tài khoản mang status == "off" bị ngăn chặn chuyển đổi 100% ở cả 2 tầng:
// UI: Nút [Chuyển] mờ đi, DefaultCursor, click no-op.
// Controller: FilterEligibleCandidates loại bỏ ngay từ vòng đầu (Criterion 6).
func TestInvariant_5_DisabledSwitch_Lockout_And_Exclusion(t *testing.T) {
	test.NewApp()
	palette := DarkPalette

	// 1. Tầng UI: Nút [Chuyển] bị làm mờ, click no-op
	switched := false
	item := manager.AccountDisplayItem{
		Email:    "disabled@example.com",
		HasToken: true,
		IsActive: false,
		Status:   "off",
	}
	card := NewAccountCardWidgetWithQuota(item, nil, palette, func(string) { switched = true }, nil)

	switchBtn, ok := card.ActionButton().(*switchTextButton)
	if !ok || switchBtn == nil {
		t.Fatalf("expected action button to be *switchTextButton")
	}
	if !switchBtn.IsDisabled() {
		t.Errorf("expected switch button to be disabled when status is 'off'")
	}
	if switchBtn.Cursor() != desktop.DefaultCursor {
		t.Errorf("expected DefaultCursor on disabled switch button, got %v", switchBtn.Cursor())
	}
	if switchBtn.text.Color != palette.TextMuted {
		t.Errorf("expected TextMuted color, got %v", switchBtn.text.Color)
	}

	switchBtn.Tapped(&fyne.PointEvent{})
	if switched {
		t.Errorf("Invariant-Disabled-Switch-Lockout VIOLATED: switch callback invoked for OFF account")
	}

	// 2. Tầng Controller: FilterEligibleCandidates loại bỏ với Criterion 6
	candOff := automation.BuildCandidateAccount("disabled@example.com", 0, true, "ref", &telemetry.AccountQuotaData{
		Email:  "disabled@example.com",
		Status: telemetry.StatusPlentiful,
		Gemini: telemetry.ModelQuotaBucket{FiveHourFraction: 0.8, WeeklyFraction: 0.5},
	}, automation.CategoryGemini)
	candOff.Status = "off"

	candOn := automation.BuildCandidateAccount("enabled@example.com", 1, true, "ref", &telemetry.AccountQuotaData{
		Email:  "enabled@example.com",
		Status: telemetry.StatusPlentiful,
		Gemini: telemetry.ModelQuotaBucket{FiveHourFraction: 0.8, WeeklyFraction: 0.5},
	}, automation.CategoryGemini)
	candOn.Status = "on"

	candidates := []automation.CandidateAccount{candOff, candOn}
	eligible, exclusions := automation.FilterEligibleCandidates(candidates, "current@example.com", nil)
	if len(eligible) != 1 || eligible[0].Email != "enabled@example.com" {
		t.Errorf("expected only enabled candidate, got %+v", eligible)
	}
	if len(exclusions) != 1 || exclusions[0].Criterion != 6 {
		t.Errorf("expected exclusion with Criterion 6, got %+v", exclusions)
	}
}

// Invariant 6: Invariant-No-Candidate-Auto-Halt
// Không có ứng viên nào đủ điều kiện -> Hệ thống lập tức ngắt cờ Tự động, lưu bền vững
// auto_switch: false vào ui_preferences.json, gạt công tắc OFF, hiển thị '⚠️ Không có tài khoản để chuyển.'.
func TestInvariant_6_NoCandidate_AutoHalt(t *testing.T) {
	app, _, _, prefPath, _ := setupStatusAcceptanceApp(t, "current@gmail.com", map[string]string{
		"current@gmail.com": "on",
	}, nil)
	defer app.Close()

	app.SetAutoSwitchEnabled(true)

	// Dispatch StatusNoCandidate
	app.OnStatusChange(automation.StatusEvent{
		Code:    automation.StatusNoCandidate,
		Message: "Không có tài khoản để chuyển.",
	})

	if app.IsAutoSwitchEnabled() {
		t.Errorf("expected IsAutoSwitchEnabled() == false")
	}
	if app.StatusBarText() != "⚠️ Không có tài khoản để chuyển." {
		t.Errorf("expected '⚠️ Không có tài khoản để chuyển.', got '%s'", app.StatusBarText())
	}

	// Check disk persistence
	deadline := time.Now().Add(2 * time.Second)
	var persisted *UIPreferences
	for time.Now().Before(deadline) {
		p, _ := LoadUIPreferences(prefPath)
		if p != nil && !p.AutoSwitch {
			persisted = p
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if persisted == nil || persisted.AutoSwitch {
		t.Errorf("expected auto_switch: false on disk, got %#v", persisted)
	}
}

// Invariant 7: Invariant-Telemetry-Silence
// Không quét quota định kỳ hoặc gửi request mạng cho tài khoản status == "off".
// Nút làm mới [🔄] trên thẻ bị vô hiệu hóa (disabled).
func TestInvariant_7_TelemetrySilence(t *testing.T) {
	test.NewApp()
	palette := DarkPalette

	// 1. UI: Nút [🔄] bị disabled khi status == "off"
	item := manager.AccountDisplayItem{
		Email:    "silent@example.com",
		HasToken: true,
		Status:   "off",
	}
	card := NewAccountCardWidgetWithQuota(item, nil, palette, nil, nil)
	if !card.refreshButton.Disabled() {
		t.Errorf("expected card.refreshButton to be disabled for OFF account")
	}

	// 2. Telemetry Pool: Bỏ qua hoàn toàn trong PollAll và từ chối PollAccount trực tiếp
	tempDir := t.TempDir()
	st, _ := store.NewAccountStore(tempDir)
	_ = st.SaveAccount("silent@example.com", &model.TokenData{AccessToken: "tok", RefreshToken: "ref"})
	_ = st.SetAccountStatus("silent@example.com", "off")

	apiCallCount := 0
	mockClient := &acceptanceQuotaClient{
		quotaFn: func(email string) (*telemetry.AccountQuotaData, error) {
			apiCallCount++
			return nil, nil
		},
	}

	pool, _ := telemetry.NewTelemetryPool(mockClient, telemetry.NewQuotaCache(filepath.Join(tempDir, "cache.json")), st)
	results, _, _, _ := pool.PollAll(context.Background())

	if len(results) != 0 {
		t.Errorf("expected 0 polled results for OFF account, got %d", len(results))
	}
	if apiCallCount != 0 {
		t.Errorf("Invariant-Telemetry-Silence VIOLATED: API called %d times for OFF account", apiCallCount)
	}

	// Direct PollAccount must be rejected without API call
	_, err := pool.PollAccount(context.Background(), "silent@example.com")
	if err == nil {
		t.Errorf("expected PollAccount on OFF account to return error")
	}
	if apiCallCount != 0 {
		t.Errorf("Invariant-Telemetry-Silence VIOLATED on direct PollAccount: API called %d times", apiCallCount)
	}
}

// TestInvariant_UI_Card_Feedback_Styling validates UI visual invariants:
// [ON] in Success green and bold, [OFF] in TextMuted gray and bold,
// pointer cursor, and hover background color 0x24 opacity.
func TestInvariant_UI_Card_Feedback_Styling(t *testing.T) {
	palette := DarkPalette

	// ON state
	btn := newStatusTextButton("on", nil, palette)
	if btn.LabelText() != "[ON]" {
		t.Errorf("expected label '[ON]', got %s", btn.LabelText())
	}
	if btn.text.Color != palette.Success {
		t.Errorf("expected Success color for [ON]")
	}
	if btn.Cursor() != desktop.PointerCursor {
		t.Errorf("expected PointerCursor")
	}

	// Hover ON
	btn.MouseIn(&desktop.MouseEvent{})
	expectedOnBg := color.NRGBA{R: palette.Success.R, G: palette.Success.G, B: palette.Success.B, A: 0x24}
	if btn.bg.FillColor != expectedOnBg {
		t.Errorf("expected hover bg %v, got %v", expectedOnBg, btn.bg.FillColor)
	}
	btn.MouseOut()

	// OFF state
	btn.SetStatus("off")
	if btn.LabelText() != "[OFF]" {
		t.Errorf("expected label '[OFF]', got %s", btn.LabelText())
	}
	if btn.text.Color != palette.TextMuted {
		t.Errorf("expected TextMuted color for [OFF]")
	}
	if btn.Cursor() != desktop.PointerCursor {
		t.Errorf("expected PointerCursor")
	}

	// Hover OFF
	btn.MouseIn(&desktop.MouseEvent{})
	expectedOffBg := color.NRGBA{R: palette.TextMuted.R, G: palette.TextMuted.G, B: palette.TextMuted.B, A: 0x24}
	if btn.bg.FillColor != expectedOffBg {
		t.Errorf("expected hover bg %v, got %v", expectedOffBg, btn.bg.FillColor)
	}
	btn.MouseOut()
}
