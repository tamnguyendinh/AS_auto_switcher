package ui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"antigravity-switch/internal/engines/antigravity/automation"
	"antigravity-switch/internal/engines/antigravity/connector"
	"antigravity-switch/internal/engines/antigravity/manager"
	"antigravity-switch/internal/engines/antigravity/model"
	"antigravity-switch/internal/engines/antigravity/reloader"
	"antigravity-switch/internal/engines/antigravity/store"
	"antigravity-switch/internal/engines/antigravity/telemetry"
	"antigravity-switch/internal/engines/antigravity/wincred"
)

// Helper to create an isolated test AccountManager within an explicit base directory.
func createAcceptanceManagerInDir(t *testing.T, baseDir string) (*manager.AccountManager, *store.AccountStore, *store.DiscoveredStore) {
	t.Helper()
	storeDir := filepath.Join(baseDir, "accounts_store")
	accStore, err := store.NewAccountStore(storeDir)
	if err != nil {
		t.Fatalf("failed initializing AccountStore: %v", err)
	}
	discStore, err := store.NewDiscoveredStore(filepath.Join(baseDir, "discovered_accounts"))
	if err != nil {
		t.Fatalf("failed initializing DiscoveredStore: %v", err)
	}

	sessionReloader := reloader.NewSessionReloader()
	sessionReloader.ProcessKiller = func(ctx context.Context) error { return nil }
	sessionReloader.Sleeper = func(ctx context.Context, d time.Duration) error { return nil }
	sessionReloader.StoragePath = filepath.Join(baseDir, "app_storage.json")
	sessionReloader.SnapshotCapturer = func(string, string, string) (string, []reloader.ActiveThread, error) { return "", nil, nil }

	mgr, err := manager.NewAccountManager(accStore, sessionReloader)
	if err != nil {
		t.Fatalf("failed initializing manager: %v", err)
	}

	// Isolate Chrome path to empty directory so host machine Chrome profiles never pollute tests
	emptyChromeDir := filepath.Join(baseDir, "isolated_chrome_empty")
	_ = os.MkdirAll(emptyChromeDir, 0755)
	mgr.SetChromeDir(emptyChromeDir)

	var activeCredToken *model.TokenData
	mgr.SetWincredHandlers(
		func() (*model.TokenData, error) {
			if activeCredToken == nil {
				return nil, wincred.ErrNotFound
			}
			return activeCredToken, nil
		},
		func(token *model.TokenData) error {
			activeCredToken = token
			return nil
		},
	)
	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return nil, errors.New("no language server")
	})
	return mgr, accStore, discStore
}

// Helper to extract email string and action slot container from a modal row object.
func extractModalRow(t *testing.T, obj fyne.CanvasObject) (string, *fyne.Container) {
	t.Helper()
	rowContainer, ok := obj.(*fyne.Container)
	if !ok {
		t.Fatalf("expected modal row to be *fyne.Container, got %T", obj)
	}
	var email string
	var actionSlot *fyne.Container
	for _, child := range rowContainer.Objects {
		if lbl, ok := child.(*widget.Label); ok {
			email = lbl.Text
		} else if c, ok := child.(*fyne.Container); ok {
			actionSlot = c
		}
	}
	return email, actionSlot
}

// Helper to extract badge text, add button, and delete button from an actionSlot container
func inspectAcceptanceActionSlot(t *testing.T, slot *fyne.Container) (badgeText string, addBtn *widget.Button, deleteBtn *widget.Button) {
	t.Helper()
	if slot == nil {
		t.Fatal("actionSlot is nil")
	}
	for _, obj := range slot.Objects {
		if c, ok := obj.(*fyne.Container); ok {
			for _, inner := range c.Objects {
				if txt, ok := inner.(*canvas.Text); ok {
					badgeText = txt.Text
				} else if btn, ok := inner.(*widget.Button); ok {
					if btn.Icon == theme.DeleteIcon() || btn.Text == "" {
						deleteBtn = btn
					} else if btn.Icon == theme.ContentAddIcon() || btn.Text == "Thêm" || btn.Text == "Add" || btn.Text == "ADD" {
						addBtn = btn
					}
				}
			}
		}
	}
	return
}

// =============================================================================
// TC-01: Khởi động ứng dụng sạch (Clean Startup)
// Điều kiện: discovered_accounts/ có 5 email (Added: false), accounts_store/ có 2 email.
// Kỳ vọng: UI tổng hiển thị đúng 2 thẻ từ accounts_store/. Tuyệt đối không xuất hiện 5 thẻ từ Chrome.
// =============================================================================
func TestAcceptance_TC01_CleanStartup(t *testing.T) {
	tempDir := t.TempDir()
	mgr, accStore, discStore := createAcceptanceManagerInDir(t, tempDir)

	// 1. discovered_accounts/ có 5 email (Added: false)
	discoveredEmails := []string{
		"chrome1@gmail.com",
		"chrome2@gmail.com",
		"chrome3@gmail.com",
		"chrome4@gmail.com",
		"chrome5@gmail.com",
	}
	for _, em := range discoveredEmails {
		if err := discStore.SaveDiscovered(em); err != nil {
			t.Fatalf("failed saving discovered email %s: %v", em, err)
		}
	}

	// 2. accounts_store/ có 2 email
	storedEmails := []string{
		"stored1@company.com",
		"stored2@company.com",
	}
	for _, em := range storedEmails {
		tok := &model.TokenData{
			AccessToken:  "tok_" + em,
			RefreshToken: "ref_" + em,
			IDToken:      makeTestJWT(em),
		}
		if err := accStore.SaveAccount(em, tok); err != nil {
			t.Fatalf("failed saving stored account %s: %v", em, err)
		}
	}

	testApp := test.NewApp()
	app := newIsolatedApp(testApp, mgr)
	app.Init()
	defer app.Close()

	waitForIdle(app, 2*time.Second)

	// UI tổng hiển thị đúng 2 thẻ từ accounts_store/
	app.mu.RLock()
	ov := app.overview
	app.mu.RUnlock()

	if ov == nil {
		t.Fatal("overview is nil")
	}
	if len(ov.Accounts) != 2 {
		t.Fatalf("expected exactly 2 accounts on main UI overview, got %d", len(ov.Accounts))
	}

	for _, acc := range ov.Accounts {
		if !acc.HasToken {
			t.Errorf("expected account %s on main UI to have HasToken=true", acc.Email)
		}
		for _, disc := range discoveredEmails {
			if strings.EqualFold(acc.Email, disc) {
				t.Fatalf("discovered Chrome email %s must NOT appear on main UI cards", disc)
			}
		}
	}

	// Tuyệt đối không xuất hiện 5 thẻ từ Chrome: kiểm tra widgets thực tế trên UI
	if len(app.accountListContainer.Objects) != 2 {
		t.Fatalf("expected exactly 2 card widgets rendered in accountListContainer, got %d", len(app.accountListContainer.Objects))
	}
}

// =============================================================================
// TC-02: Mở Modal [+ Thêm] (Open Add Modal)
// Điều kiện: 1 email có token, 1 email đã add, 3 email chưa add.
// Kỳ vọng: Modal hiển thị đúng 3 trạng thái: 1 dòng [✔ Đã OAuth], 1 dòng [Đã thêm], 3 dòng [Cần OAuth] + [ + Thêm ].
// =============================================================================
func TestAcceptance_TC02_OpenAddModal_ThreeStates(t *testing.T) {
	tempDir := t.TempDir()
	mgr, accStore, discStore := createAcceptanceManagerInDir(t, tempDir)

	// 1 email có token trong accounts_store
	oauthEmail := "user1_oauth@company.com"
	if err := discStore.SaveDiscovered(oauthEmail); err != nil {
		t.Fatal(err)
	}
	if err := accStore.SaveAccount(oauthEmail, &model.TokenData{
		AccessToken:  "token_oauth",
		RefreshToken: "ref_oauth",
		IDToken:      makeTestJWT(oauthEmail),
	}); err != nil {
		t.Fatal(err)
	}

	// 1 email đã add (Added: true)
	addedEmail := "user2_added@school.edu"
	if err := discStore.SaveDiscovered(addedEmail); err != nil {
		t.Fatal(err)
	}
	if err := discStore.SetDiscoveredAdded(addedEmail, true); err != nil {
		t.Fatal(err)
	}

	// 3 email chưa add (Added: false)
	rawEmails := []string{
		"user3_raw@gmail.com",
		"user4_raw@gmail.com",
		"user5_raw@gmail.com",
	}
	for _, em := range rawEmails {
		if err := discStore.SaveDiscovered(em); err != nil {
			t.Fatal(err)
		}
	}

	testApp := test.NewApp()
	app := newIsolatedApp(testApp, mgr)
	app.Init()
	defer app.Close()

	// Mở Modal [+ Thêm]
	contentBox, scanBtn, listContainer, _ := app.buildAddAccountModalContent()
	if contentBox == nil || scanBtn == nil || listContainer == nil {
		t.Fatal("modal components must not be nil")
	}

	// Modal hiển thị đúng 5 dòng ứng viên
	if len(listContainer.Objects) != 5 {
		t.Fatalf("expected 5 candidate rows in modal listContainer, got %d", len(listContainer.Objects))
	}

	rowsMap := make(map[string]*fyne.Container)
	for _, obj := range listContainer.Objects {
		em, slot := extractModalRow(t, obj)
		rowsMap[em] = slot
	}

	// 1. Dòng [✔ Đã OAuth] + Nút Xóa
	slotOAuth := rowsMap[oauthEmail]
	badgeOAuth, addOAuth, delOAuth := inspectAcceptanceActionSlot(t, slotOAuth)
	if badgeOAuth != "[✔ Đã OAuth]" {
		t.Fatalf("expected badge '[✔ Đã OAuth]' for %s, got '%s'", oauthEmail, badgeOAuth)
	}
	if addOAuth != nil {
		t.Fatalf("expected no ADD button for %s", oauthEmail)
	}
	if delOAuth == nil || delOAuth.Icon != theme.DeleteIcon() {
		t.Fatalf("expected delete button for %s", oauthEmail)
	}

	// 2. Dòng [Đã thêm] + Nút Xóa
	slotAdded := rowsMap[addedEmail]
	badgeAdded, addAdded, delAdded := inspectAcceptanceActionSlot(t, slotAdded)
	if badgeAdded != "[Đã thêm]" {
		t.Fatalf("expected badge '[Đã thêm]' for %s, got '%s'", addedEmail, badgeAdded)
	}
	if addAdded != nil {
		t.Fatalf("expected no ADD button for %s", addedEmail)
	}
	if delAdded == nil || delAdded.Icon != theme.DeleteIcon() {
		t.Fatalf("expected delete button for %s", addedEmail)
	}

	// 3. 3 dòng [Cần OAuth] + [ + ADD ] + Nút Xóa
	for _, rawEm := range rawEmails {
		slotRaw := rowsMap[rawEm]
		badgeRaw, addRaw, delRaw := inspectAcceptanceActionSlot(t, slotRaw)
		if badgeRaw != "[Cần OAuth]" {
			t.Fatalf("expected '[Cần OAuth]' for %s, got '%s'", rawEm, badgeRaw)
		}
		if addRaw == nil || (addRaw.Text != "Thêm" && addRaw.Text != "Add" && addRaw.Text != "ADD") {
			t.Fatalf("expected ADD button for %s", rawEm)
		}
		if delRaw == nil || delRaw.Icon != theme.DeleteIcon() {
			t.Fatalf("expected delete button for %s", rawEm)
		}
	}
}

// =============================================================================
// TC-03: Nhấn nút [ + ADD ] (Tap ADD button)
// Điều kiện: Nhấn [ + ADD ] tại email test@gmail.com.
// Kỳ vọng: Nhãn [Cần OAuth] đổi thành [Đã thêm], nút [ + Thêm ] biến mất.
// Thẻ test@gmail.com xuất hiện ngay ngoài UI tổng với badge [Cần OAuth] và Quota N/A.
// =============================================================================
func TestAcceptance_TC03_TapAddButton_InPlaceMutationAndOverviewCard(t *testing.T) {
	tempDir := t.TempDir()
	mgr, _, discStore := createAcceptanceManagerInDir(t, tempDir)

	targetEmail := "test@gmail.com"
	if err := discStore.SaveDiscovered(targetEmail); err != nil {
		t.Fatal(err)
	}

	testApp := test.NewApp()
	app := newIsolatedApp(testApp, mgr)
	app.Init()
	defer app.Close()

	_, _, listContainer, _ := app.buildAddAccountModalContent()
	if len(listContainer.Objects) != 1 {
		t.Fatalf("expected 1 item in modal listContainer, got %d", len(listContainer.Objects))
	}

	_, actionSlot := extractModalRow(t, listContainer.Objects[0])
	_, addBtn, delBtn := inspectAcceptanceActionSlot(t, actionSlot)
	if addBtn == nil {
		t.Fatal("ADD button not found")
	}
	if delBtn == nil {
		t.Fatal("delete button not found")
	}

	// Nhấn nút [ + ADD ] tại email test@gmail.com
	addBtn.OnTapped()

	// 1. Nhãn [Cần OAuth] đổi thành [Đã thêm], nút [ + Thêm ] biến mất, nút Xóa vẫn còn
	badgeAfter, addAfter, delAfter := inspectAcceptanceActionSlot(t, actionSlot)
	if badgeAfter != "[Đã thêm]" {
		t.Fatalf("expected badge '[Đã thêm]', got '%s'", badgeAfter)
	}
	if addAfter != nil {
		t.Fatalf("expected ADD button to disappear after click")
	}
	if delAfter == nil {
		t.Fatalf("expected delete button to remain after ADD clicked")
	}

	// 2. File discovered_accounts/ ghi nhận Added: true
	records, err := discStore.ListDiscoveredRecords()
	if err != nil || len(records) != 1 || !records[0].Added {
		t.Fatalf("expected record Added: true in store, got %+v (err: %v)", records, err)
	}

	// 3. Thẻ test@gmail.com xuất hiện ngay ngoài UI tổng với badge [Cần OAuth] và Quota N/A
	deadline := time.Now().Add(3 * time.Second)
	var foundCard *AccountCardWidget
	for time.Now().Before(deadline) {
		app.mu.RLock()
		ov := app.overview
		cards := app.accountCards
		app.mu.RUnlock()

		if ov != nil {
			for i, acc := range ov.Accounts {
				if acc.Email == targetEmail && acc.DisplayBadge == "[Cần OAuth]" && acc.State == model.StateUnauth {
					if i < len(cards) {
						foundCard = cards[i]
					}
					break
				}
			}
		}
		if foundCard != nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if foundCard == nil {
		t.Fatalf("card for %s with [Cần OAuth] not found on main UI", targetEmail)
	}

	// Quota N/A
	if foundCard.gemini5hCell == nil || foundCard.gemini5hCell.pctText.Text != "N/A" {
		t.Errorf("expected gemini5hCell to display 'N/A', got '%s'", foundCard.gemini5hCell.pctText.Text)
	}
	if foundCard.geminiWeekCell == nil || foundCard.geminiWeekCell.pctText.Text != "N/A" {
		t.Errorf("expected geminiWeekCell to display 'N/A', got '%s'", foundCard.geminiWeekCell.pctText.Text)
	}
}

// =============================================================================
// TC-04: Kiểm tra tính bền vững - Restart App (Persistence across App Restart)
// Điều kiện: Tắt app sau khi hoàn thành TC-03 rồi mở lại.
// Kỳ vọng: Thẻ test@gmail.com vẫn hiển thị nguyên vẹn trên UI tổng ở trạng thái [Cần OAuth].
// =============================================================================
func TestAcceptance_TC04_PersistenceAcrossAppRestart(t *testing.T) {
	tempDir := t.TempDir()
	storeDir := filepath.Join(tempDir, "accounts_store")
	discoveredDir := filepath.Join(tempDir, "discovered_accounts")

	accStore, err := store.NewAccountStore(storeDir)
	if err != nil {
		t.Fatal(err)
	}
	discStore, err := store.NewDiscoveredStore(discoveredDir)
	if err != nil {
		t.Fatal(err)
	}

	// Đã hoàn thành TC-03: test@gmail.com có Added: true
	targetEmail := "test@gmail.com"
	if err := discStore.SaveDiscovered(targetEmail); err != nil {
		t.Fatal(err)
	}
	if err := discStore.SetDiscoveredAdded(targetEmail, true); err != nil {
		t.Fatal(err)
	}

	// Thêm 1 tài khoản thường có token
	authEmail := "persisted_user@company.com"
	if err := accStore.SaveAccount(authEmail, &model.TokenData{
		AccessToken:  "token_persisted",
		RefreshToken: "ref_persisted",
		IDToken:      makeTestJWT(authEmail),
	}); err != nil {
		t.Fatal(err)
	}

	// Khởi chạy Phiên App thứ nhất
	{
		mgr1, _, _ := createAcceptanceManagerInDir(t, tempDir)
		testApp1 := test.NewApp()
		app1 := newIsolatedApp(testApp1, mgr1)
		app1.Init()
		waitForIdle(app1, 2*time.Second)

		// Xác nhận trên app1
		app1.mu.RLock()
		ov1 := app1.overview
		app1.mu.RUnlock()
		if len(ov1.Accounts) != 2 {
			t.Fatalf("app1 expected 2 accounts, got %d", len(ov1.Accounts))
		}

		// Tắt app1 (Close)
		app1.Close()
	}

	// RESTART APP: Khởi chạy Phiên App thứ hai đọc cùng dữ liệu từ đĩa
	{
		mgr2, _, _ := createAcceptanceManagerInDir(t, tempDir)
		testApp2 := test.NewApp()
		app2 := newIsolatedApp(testApp2, mgr2)
		app2.Init()
		defer app2.Close()
		waitForIdle(app2, 2*time.Second)

		app2.mu.RLock()
		ov2 := app2.overview
		app2.mu.RUnlock()

		if ov2 == nil {
			t.Fatal("app2 overview is nil after restart")
		}
		if len(ov2.Accounts) != 2 {
			t.Fatalf("app2 expected 2 accounts after restart, got %d", len(ov2.Accounts))
		}

		var foundTarget bool
		for _, acc := range ov2.Accounts {
			if acc.Email == targetEmail {
				foundTarget = true
				if acc.DisplayBadge != "[Cần OAuth]" {
					t.Errorf("expected [Cần OAuth] after restart, got '%s'", acc.DisplayBadge)
				}
				if acc.State != model.StateUnauth {
					t.Errorf("expected StateUnauth, got %v", acc.State)
				}
				if acc.HasToken {
					t.Errorf("expected HasToken=false")
				}
			}
		}
		if !foundTarget {
			t.Fatalf("test@gmail.com not found on app2 after restart")
		}

		// Thẻ test@gmail.com vẫn hiển thị nguyên vẹn trên UI tổng
		if len(app2.accountListContainer.Objects) != 2 {
			t.Fatalf("expected 2 card widgets rendered on app2, got %d", len(app2.accountListContainer.Objects))
		}
	}
}

// =============================================================================
// TC-05: Quét Chrome On-Demand (On-Demand Chrome Scan)
// Điều kiện: Mở Modal, bấm nút [🔄 Quét Chrome].
// Kỳ vọng: Quét các profile Chrome, ghi nhận email mới vào discovered_accounts/ (Added: false).
// Danh sách Modal nạp thêm dòng mới. UI tổng không bị thêm thẻ tự động.
// =============================================================================
func TestAcceptance_TC05_OnDemandChromeScan(t *testing.T) {
	tempDir := t.TempDir()
	mgr, accStore, discStore := createAcceptanceManagerInDir(t, tempDir)

	// Tài khoản ban đầu trong accounts_store
	initialAuth := "initial@company.com"
	if err := accStore.SaveAccount(initialAuth, &model.TokenData{
		AccessToken:  "tok_init",
		RefreshToken: "ref_init",
		IDToken:      makeTestJWT(initialAuth),
	}); err != nil {
		t.Fatal(err)
	}

	// Chuẩn bị thư mục Chrome giả lập với 2 email mới
	mockChromeDir := filepath.Join(tempDir, "mock_chrome")
	profileDir := filepath.Join(mockChromeDir, "Profile 1")
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		t.Fatal(err)
	}
	prefJSON := `{
		"account_info": [
			{"email": "scanned_chrome1@gmail.com"},
			{"email": "scanned_chrome2@gmail.com"}
		]
	}`
	if err := os.WriteFile(filepath.Join(profileDir, "Preferences"), []byte(prefJSON), 0600); err != nil {
		t.Fatal(err)
	}
	mgr.SetChromeDir(mockChromeDir)

	testApp := test.NewApp()
	app := newIsolatedApp(testApp, mgr)
	app.Init()
	defer app.Close()
	waitForIdle(app, 2*time.Second)

	// UI tổng ban đầu có đúng 1 tài khoản
	app.mu.RLock()
	initCount := len(app.overview.Accounts)
	app.mu.RUnlock()
	if initCount != 1 {
		t.Fatalf("expected initial 1 account on main UI, got %d", initCount)
	}

	// Mở Modal [+ Thêm]
	_, scanBtn, listContainer, reloadCandidates := app.buildAddAccountModalContent()

	// Bấm nút [🔄 Quét Chrome]
	scanBtn.OnTapped()

	// Chờ scan hoàn tất
	deadline := time.Now().Add(3 * time.Second)
	scanDone := false
	for time.Now().Before(deadline) {
		if !scanBtn.Disabled() && scanBtn.Text == "Quét Chrome" {
			scanDone = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !scanDone {
		t.Fatal("scan chrome button did not restore within timeout")
	}

	// Ghi nhận email mới vào discovered_accounts/ với Added == false
	records, err := discStore.ListDiscoveredRecords()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 discovered records, got %d", len(records))
	}
	for _, r := range records {
		if r.Added {
			t.Errorf("expected newly scanned account %s to have Added: false", r.Email)
		}
	}

	// Danh sách Modal nạp thêm dòng mới
	reloadCandidates()
	if len(listContainer.Objects) != 2 {
		t.Fatalf("expected 2 candidates in modal after scan, got %d", len(listContainer.Objects))
	}

	// UI tổng không bị thêm thẻ tự động
	app.mu.RLock()
	afterCount := len(app.overview.Accounts)
	app.mu.RUnlock()
	if afterCount != initCount {
		t.Fatalf("UI tổng must NOT automatically add cards: before=%d, after=%d", initCount, afterCount)
	}
}

// =============================================================================
// TC-06: Hoàn tất OAuth sau khi add (Complete OAuth after add)
// Điều kiện: Giả lập đăng nhập lưu token vào accounts_store/.
// Kỳ vọng: Thẻ UI tổng đổi thành [Chuyển] / [✔ Dùng]. Mở Modal thấy chuyển thành [✔ Đã OAuth].
// =============================================================================
func TestAcceptance_TC06_CompleteOAuthAfterAdd(t *testing.T) {
	tempDir := t.TempDir()
	mgr, accStore, discStore := createAcceptanceManagerInDir(t, tempDir)

	targetEmail := "test@gmail.com"
	if err := discStore.SaveDiscovered(targetEmail); err != nil {
		t.Fatal(err)
	}
	if err := discStore.SetDiscoveredAdded(targetEmail, true); err != nil {
		t.Fatal(err)
	}

	testApp := test.NewApp()
	app := newIsolatedApp(testApp, mgr)
	app.Init()
	defer app.Close()
	waitForIdle(app, 2*time.Second)

	// Ban đầu: thẻ trên UI tổng mang trạng thái [Cần OAuth]
	app.mu.RLock()
	ov := app.overview
	app.mu.RUnlock()
	if len(ov.Accounts) != 1 || ov.Accounts[0].DisplayBadge != "[Cần OAuth]" {
		t.Fatalf("expected 1 account with [Cần OAuth], got %+v", ov.Accounts)
	}

	// Giả lập người dùng đăng nhập qua Antigravity IDE lưu token vào accounts_store/
	token := &model.TokenData{
		AccessToken:  "oauth_access_token",
		RefreshToken: "oauth_refresh_token",
		IDToken:      makeTestJWT(targetEmail),
	}
	if err := accStore.SaveAccount(targetEmail, token); err != nil {
		t.Fatalf("failed saving account token: %v", err)
	}

	// Kích hoạt làm mới UI
	app.Refresh()
	waitForIdle(app, 2*time.Second)

	// Thẻ UI tổng đổi thành [Chuyển] (hoặc [✔ Dùng] nếu active)
	app.mu.RLock()
	ovAfter := app.overview
	app.mu.RUnlock()
	if len(ovAfter.Accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(ovAfter.Accounts))
	}
	accAfter := ovAfter.Accounts[0]
	if !accAfter.HasToken {
		t.Errorf("expected HasToken=true after OAuth save")
	}
	if accAfter.DisplayBadge != "[Chuyển]" && accAfter.DisplayBadge != "[✔ Dùng]" {
		t.Errorf("expected badge '[Chuyển]' or '[✔ Dùng]', got '%s'", accAfter.DisplayBadge)
	}

	// Mở Modal thấy chuyển thành [✔ Đã OAuth]
	_, _, listContainer, _ := app.buildAddAccountModalContent()
	if len(listContainer.Objects) != 1 {
		t.Fatalf("expected 1 candidate in modal, got %d", len(listContainer.Objects))
	}
	_, actionSlot := extractModalRow(t, listContainer.Objects[0])
	badge, addOAuth, delOAuth := inspectAcceptanceActionSlot(t, actionSlot)
	if badge != "[✔ Đã OAuth]" {
		t.Fatalf("expected badge '[✔ Đã OAuth]' in modal, got '%s'", badge)
	}
	if addOAuth != nil {
		t.Fatalf("expected no ADD button for OAuth account in modal")
	}
	if delOAuth == nil || delOAuth.Icon != theme.DeleteIcon() {
		t.Fatalf("expected delete button for OAuth account in modal")
	}
}

// =============================================================================
// TC-07: An toàn Auto-Switch & Telemetry (Auto-Switch & Telemetry Safety)
// Điều kiện: Chạy FilterEligibleCandidates và vòng lặp telemetry khi có thẻ [Cần OAuth].
// Kỳ vọng: Burn-down Sorter tự động loại bỏ thẻ [Cần OAuth]. Không có panic, không có crash,
// không gọi API quota thừa cho tài khoản [Cần OAuth].
// =============================================================================
func TestAcceptance_TC07_AutoSwitchAndTelemetrySafety(t *testing.T) {
	tempDir := t.TempDir()
	mgr, accStore, discStore := createAcceptanceManagerInDir(t, tempDir)

	activeEmail := "active@company.com"
	readyEmail := "ready@company.com"
	unauthEmail := "unauth@company.com"

	// 1. activeEmail: in store with token, active in wincred
	tokActive := &model.TokenData{
		AccessToken:  "tok_active",
		RefreshToken: "ref_active",
		IDToken:      makeTestJWT(activeEmail),
	}
	_ = accStore.SaveAccount(activeEmail, tokActive)
	mgr.SetWincredHandlers(
		func() (*model.TokenData, error) { return tokActive, nil },
		func(tok *model.TokenData) error { return nil },
	)

	// 2. readyEmail: in store with token, plentiful quota
	tokReady := &model.TokenData{
		AccessToken:  "tok_ready",
		RefreshToken: "ref_ready",
		IDToken:      makeTestJWT(readyEmail),
	}
	_ = accStore.SaveAccount(readyEmail, tokReady)

	// 3. unauthEmail: added in discovered store, but NO token in accounts_store
	_ = discStore.SaveDiscovered(unauthEmail)
	_ = discStore.SetDiscoveredAdded(unauthEmail, true)

	testApp := test.NewApp()
	app := newIsolatedApp(testApp, mgr)

	// Wire telemetry with call tracker
	var mu sync.Mutex
	var queriedEmails []string
	quotaMock := &acceptanceQuotaClient{
		quotaFn: func(email string) (*telemetry.AccountQuotaData, error) {
			mu.Lock()
			queriedEmails = append(queriedEmails, email)
			mu.Unlock()
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
		},
	}
	pool, err := telemetry.NewTelemetryPool(quotaMock, app.quotaCache, accStore)
	if err != nil {
		t.Fatal(err)
	}
	pool.SetLocalQuotaFetcher(nil)
	app.SetTelemetryPool(pool)

	app.Init()
	defer app.Close()
	waitForIdle(app, 2*time.Second)

	// Overview contains 3 accounts
	app.mu.RLock()
	ov := app.overview
	app.mu.RUnlock()
	if len(ov.Accounts) != 3 {
		t.Fatalf("expected 3 accounts on overview, got %d", len(ov.Accounts))
	}

	// 1. Kiểm tra an toàn Auto-Switch: Burn-down Sorter loại trừ unauthEmail
	candidates := []automation.CandidateAccount{
		automation.BuildCandidateAccount(activeEmail, 0, true, tokActive.RefreshToken, nil, "gemini"),
		automation.BuildCandidateAccount(readyEmail, 1, true, tokReady.RefreshToken, &telemetry.AccountQuotaData{
			Email:  readyEmail,
			Status: telemetry.StatusPlentiful,
			Gemini: telemetry.ModelQuotaBucket{FiveHourFraction: 0.85, WeeklyFraction: 0.60},
		}, "gemini"),
		automation.BuildCandidateAccount(unauthEmail, 2, false, "", nil, "gemini"), // [Cần OAuth] card
	}
	candidates[0].Status = "on"
	candidates[1].Status = "on"
	candidates[2].Status = "on"

	eligible, exclusions := automation.FilterEligibleCandidates(candidates, activeEmail, nil)
	// Active account excluded by Criterion 1
	// Unauth account excluded by Criterion 2 (TokenValid == false)
	// Only readyEmail is eligible
	if len(eligible) != 1 || eligible[0].Email != readyEmail {
		t.Fatalf("expected only readyEmail to be eligible, got %+v", eligible)
	}
	var unauthExcluded bool
	for _, exc := range exclusions {
		if exc.Email == unauthEmail {
			unauthExcluded = true
			if exc.Criterion != 2 {
				t.Errorf("expected exclusion criterion 2 for unauth account, got %d (%s)", exc.Criterion, exc.Reason)
			}
		}
	}
	if !unauthExcluded {
		t.Errorf("unauth account was not excluded by FilterEligibleCandidates")
	}

	// 2. Kiểm tra an toàn Telemetry: poll chỉ chạy cho accounts có token
	_, _, _, pollErr := app.telemetryPool.PollAll(context.Background())
	if pollErr != nil {
		t.Fatalf("PollAll error: %v", pollErr)
	}

	mu.Lock()
	calls := append([]string(nil), queriedEmails...)
	mu.Unlock()

	for _, called := range calls {
		if strings.EqualFold(called, unauthEmail) {
			t.Fatalf("telemetry poll queried unauth email %s! Quota API must not be called for [Cần OAuth] accounts", called)
		}
	}
}

// =============================================================================
// TC-08: Hiệu năng & Tốc độ Khởi động (Startup Performance Benchmark)
// Điều kiện: Đo thời gian chạy GetAccountStates() khi khởi động nạp UI từ đĩa.
// Kỳ vọng: Thời gian dưới 50ms, không bị nghẽn bởi Chrome profile I/O.
// =============================================================================
func TestAcceptance_TC08_StartupPerformanceBenchmark(t *testing.T) {
	tempDir := t.TempDir()
	mgr, accStore, discStore := createAcceptanceManagerInDir(t, tempDir)

	// Nạp 15 tài khoản danh bạ (5 đã add, 10 chưa add)
	for i := 1; i <= 15; i++ {
		email := fmt.Sprintf("disc%02d@example.com", i)
		if err := discStore.SaveDiscovered(email); err != nil {
			t.Fatal(err)
		}
		if i <= 5 {
			if err := discStore.SetDiscoveredAdded(email, true); err != nil {
				t.Fatal(err)
			}
		}
	}

	// Nạp 5 tài khoản vào accounts_store
	for i := 1; i <= 5; i++ {
		email := fmt.Sprintf("auth%02d@example.com", i)
		tok := &model.TokenData{
			AccessToken:  fmt.Sprintf("tok_%d", i),
			RefreshToken: fmt.Sprintf("ref_%d", i),
			IDToken:      makeTestJWT(email),
		}
		if err := accStore.SaveAccount(email, tok); err != nil {
			t.Fatal(err)
		}
	}

	// Benchmark GetAccountStates qua 10 lượt chạy
	const iterations = 10
	var totalDuration time.Duration
	var maxDuration time.Duration

	for i := 0; i < iterations; i++ {
		start := time.Now()
		ov, err := mgr.GetAccountStates(context.Background())
		dur := time.Since(start)

		if err != nil {
			t.Fatalf("iteration %d: GetAccountStates failed: %v", i, err)
		}
		if ov == nil {
			t.Fatalf("iteration %d: overview is nil", i)
		}
		// Kỳ vọng: 5 tài khoản từ store + 5 tài khoản có Added=true = 10 tài khoản
		if len(ov.Accounts) != 10 {
			t.Fatalf("iteration %d: expected 10 accounts, got %d", i, len(ov.Accounts))
		}

		totalDuration += dur
		if dur > maxDuration {
			maxDuration = dur
		}
	}

	avgDuration := totalDuration / iterations
	t.Logf("TC-08 Startup Benchmark: %d runs, avg=%v, max=%v (tiêu chuẩn: < 50ms)", iterations, avgDuration, maxDuration)

	// Tiêu chuẩn Pass: thời gian dưới 50ms
	if maxDuration >= 50*time.Millisecond {
		t.Errorf("GetAccountStates took %v, exceeded maximum allowed 50ms", maxDuration)
	}
	if avgDuration >= 50*time.Millisecond {
		t.Errorf("GetAccountStates average took %v, exceeded maximum allowed 50ms", avgDuration)
	}
}

// =============================================================================
// TC-09: Xóa ứng viên trong Modal với Popup Xác Nhận (Delete Confirmation Flow)
// Kiểm tra Invariant-Delete-Confirmation-Dialog, Invariant-Store-Isolation,
// và Invariant-Ghost-Account-State.
// =============================================================================
func TestAcceptance_TC09_DeleteCandidateConfirmationFlow(t *testing.T) {
	tempDir := t.TempDir()
	mgr, accStore, discStore := createAcceptanceManagerInDir(t, tempDir)

	activeEmail := "main.active@test.com"
	if err := accStore.SaveAccount(activeEmail, &model.TokenData{
		AccessToken: "active_tok",
		IDToken:     makeTestJWT(activeEmail),
	}); err != nil {
		t.Fatal(err)
	}

	candEmail := "cand.discovered@test.com"
	if err := discStore.SaveDiscovered(candEmail); err != nil {
		t.Fatal(err)
	}

	ghostEmail := "cand.ghost@test.com"
	mgr.TrackAccount(ghostEmail)

	testApp := test.NewApp()
	app := newIsolatedApp(testApp, mgr)
	app.Init()
	defer app.Close()

	// Cập nhật UI chính
	ov, _ := mgr.GetAccountStates(context.Background())
	app.mu.Lock()
	app.overview = ov
	app.mu.Unlock()
	app.updateUI(ov)

	_, _, listContainer, reloadCandidates := app.buildAddAccountModalContent()

	// Modal phải hiển thị 2 ứng viên: candEmail và ghostEmail
	if len(listContainer.Objects) != 2 {
		t.Fatalf("expected 2 items in modal, got %d", len(listContainer.Objects))
	}

	rowsMap := make(map[string]*fyne.Container)
	for _, obj := range listContainer.Objects {
		em, slot := extractModalRow(t, obj)
		rowsMap[em] = slot
	}

	// 1. Thao tác trên candEmail: bấm Xóa -> popup xuất hiện -> bấm Hủy -> dữ liệu giữ nguyên
	candSlot, ok := rowsMap[candEmail]
	if !ok {
		t.Fatalf("row for %s not found", candEmail)
	}
	_, _, delCand := inspectAcceptanceActionSlot(t, candSlot)
	if delCand == nil {
		t.Fatal("delete button for candEmail not found")
	}

	delCand.OnTapped()
	topOverlay := app.win.Canvas().Overlays().Top()
	if topOverlay == nil {
		t.Fatal("expected confirmation dialog overlay")
	}

	buttons := findButtonsInCanvas(topOverlay)
	var cancelBtn, confirmBtn *widget.Button
	for _, b := range buttons {
		if b.Text == "Hủy" {
			cancelBtn = b
		} else if b.Text == "Xóa" {
			confirmBtn = b
		}
	}
	if cancelBtn == nil || confirmBtn == nil {
		t.Fatal("dialog buttons 'Hủy' or 'Xóa' not found")
	}
	if confirmBtn.Importance != widget.DangerImportance {
		t.Errorf("confirm button must have DangerImportance")
	}

	// Bấm Hủy
	cancelBtn.OnTapped()
	if !discStore.HasDiscovered(candEmail) {
		t.Errorf("file for %s must remain after cancel", candEmail)
	}

	// 2. Bấm Xóa lần 2 trên candEmail -> popup xuất hiện -> bấm Xóa -> file bị xóa, dòng biến mất
	delCand.OnTapped()
	topOverlay2 := app.win.Canvas().Overlays().Top()
	if topOverlay2 == nil {
		t.Fatal("expected confirmation dialog overlay on second tap")
	}
	buttons2 := findButtonsInCanvas(topOverlay2)
	var confirmBtn2 *widget.Button
	for _, b := range buttons2 {
		if b.Text == "Xóa" {
			confirmBtn2 = b
			break
		}
	}
	if confirmBtn2 == nil {
		t.Fatal("confirm button 'Xóa' not found")
	}

	confirmBtn2.OnTapped()

	// Chờ UI modal cập nhật sau khi xác nhận xóa
	deadline := time.Now().Add(2 * time.Second)
	candRemoved := false
	for time.Now().Before(deadline) {
		if len(listContainer.Objects) == 1 {
			em, _ := extractModalRow(t, listContainer.Objects[0])
			if em == ghostEmail {
				candRemoved = true
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !candRemoved {
		reloadCandidates()
		if len(listContainer.Objects) != 1 {
			t.Fatalf("expected 1 remaining row in modal, got %d", len(listContainer.Objects))
		}
	}

	// File trong discovered_accounts bị xóa
	if discStore.HasDiscovered(candEmail) {
		t.Errorf("file for %s in discovered_accounts must be deleted", candEmail)
	}
	// Tài khoản trên UI chính và accounts_store TUYỆT ĐỐI không bị ảnh hưởng (Invariant-Store-Isolation)
	if !accStore.HasAccount(activeEmail) {
		t.Errorf("activeEmail in accounts_store must NOT be affected")
	}
	app.mu.RLock()
	mainCount := len(app.overview.Accounts)
	app.mu.RUnlock()
	if mainCount != 1 {
		t.Errorf("main UI card count must remain 1, got %d", mainCount)
	}

	// 3. Thao tác trên ghostEmail: [Tài khoản không tồn tại] -> bấm Xóa -> popup xuất hiện -> bấm Xóa -> modal rỗng
	ghostRow := listContainer.Objects[0].(*fyne.Container)
	var ghostSlot *fyne.Container
	for _, child := range ghostRow.Objects {
		if c, ok := child.(*fyne.Container); ok {
			ghostSlot = c
		}
	}
	ghostBadge, ghostAdd, ghostDel := inspectAcceptanceActionSlot(t, ghostSlot)
	if ghostBadge != "[Tài khoản không tồn tại]" {
		t.Errorf("expected badge '[Tài khoản không tồn tại]', got '%s'", ghostBadge)
	}
	if ghostAdd != nil {
		t.Errorf("ADD button must be hidden for ghost account")
	}
	if ghostDel == nil {
		t.Fatal("delete button must be present for ghost account")
	}

	ghostDel.OnTapped()
	topOverlay3 := app.win.Canvas().Overlays().Top()
	if topOverlay3 == nil {
		t.Fatal("expected confirmation dialog overlay for ghost account")
	}
	buttons3 := findButtonsInCanvas(topOverlay3)
	var confirmBtn3 *widget.Button
	for _, b := range buttons3 {
		if b.Text == "Xóa" {
			confirmBtn3 = b
			break
		}
	}
	if confirmBtn3 == nil {
		t.Fatal("confirm button 'Xóa' not found for ghost account")
	}

	confirmBtn3.OnTapped()

	// Chờ UI modal cập nhật về trạng thái rỗng
	deadline3 := time.Now().Add(2 * time.Second)
	allRemoved := false
	for time.Now().Before(deadline3) {
		if len(listContainer.Objects) == 1 {
			if emptyLbl, ok := listContainer.Objects[0].(*widget.Label); ok && emptyLbl.Text == "Không có tài khoản nào trong danh bạ." {
				allRemoved = true
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !allRemoved {
		reloadCandidates()
		if len(listContainer.Objects) != 1 {
			t.Fatalf("expected empty label in modal, got %d items", len(listContainer.Objects))
		}
	}
}
