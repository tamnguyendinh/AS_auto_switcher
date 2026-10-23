package ui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"antigravity-switch/internal/connector"
	"antigravity-switch/internal/manager"
	"antigravity-switch/internal/model"
	"antigravity-switch/internal/store"
	"antigravity-switch/internal/wincred"
)

func TestShowAddAccountModal_NilSafety(t *testing.T) {
	// 1. nil App must not panic
	var nilApp *App
	nilApp.ShowAddAccountModal()

	// 2. App with nil window must not panic
	mgr := createTestManager(t)
	appWithoutWin := &App{mgr: mgr}
	appWithoutWin.ShowAddAccountModal()

	// 3. App with nil manager must not panic
	testApp := test.NewApp()
	testWin := testApp.NewWindow("Test")
	appWithoutMgr := &App{win: testWin}
	appWithoutMgr.ShowAddAccountModal()
}

func TestShowAddAccountModal_EmptyCandidates(t *testing.T) {
	mgr := createTestManager(t)
	testApp := test.NewApp()
	app := newIsolatedApp(testApp, mgr)
	app.Init()
	defer app.Close()

	contentBox, scanBtn, listContainer, _ := app.buildAddAccountModalContent()
	if contentBox == nil {
		t.Fatal("expected non-nil contentBox")
	}
	if scanBtn == nil {
		t.Fatal("expected non-nil scanBtn")
	}
	if scanBtn.Text != "Quét Chrome" {
		t.Errorf("expected scanBtn text 'Quét Chrome', got '%s'", scanBtn.Text)
	}

	if len(listContainer.Objects) != 1 {
		t.Fatalf("expected 1 item in empty listContainer, got %d", len(listContainer.Objects))
	}
	emptyLbl, ok := listContainer.Objects[0].(*widget.Label)
	if !ok {
		t.Fatalf("expected *widget.Label for empty state, got %T", listContainer.Objects[0])
	}
	if emptyLbl.Text != "Không có tài khoản nào trong danh bạ." {
		t.Errorf("unexpected empty state label text: %s", emptyLbl.Text)
	}
	if !emptyLbl.TextStyle.Italic {
		t.Errorf("expected empty label style to be Italic")
	}
}

func TestShowAddAccountModal_ThreeCandidateStates(t *testing.T) {
	mgr := createTestManager(t)
	testApp := test.NewApp()
	app := newIsolatedApp(testApp, mgr)
	app.Init()
	defer app.Close()

	discStorePath := filepath.Join(filepath.Dir(mgr.Store().GetStoreDir()), "discovered_accounts")
	discStore, err := store.NewDiscoveredStore(discStorePath)
	if err != nil {
		t.Fatalf("failed creating discovered store: %v", err)
	}

	// 1. Candidate 1: Has token -> [✔ Đã OAuth]
	email1 := "user1@company.com"
	if err := discStore.SaveDiscovered(email1); err != nil {
		t.Fatalf("SaveDiscovered error: %v", err)
	}
	if err := mgr.Store().SaveAccount(email1, &model.TokenData{
		AccessToken: "test_token_1",
		IDToken:     makeTestJWT(email1),
	}); err != nil {
		t.Fatalf("SaveAccount error: %v", err)
	}

	// 2. Candidate 2: Discovered, !HasToken && !Added -> [Cần OAuth] + ADD
	email2 := "user2@gmail.com"
	if err := discStore.SaveDiscovered(email2); err != nil {
		t.Fatalf("SaveDiscovered error: %v", err)
	}

	// 3. Candidate 3: Discovered, !HasToken && Added -> [Đã add]
	email3 := "user3@school.edu"
	if err := discStore.SaveDiscovered(email3); err != nil {
		t.Fatalf("SaveDiscovered error: %v", err)
	}
	if err := discStore.SetDiscoveredAdded(email3, true); err != nil {
		t.Fatalf("SetDiscoveredAdded error: %v", err)
	}

	_, _, listContainer, _ := app.buildAddAccountModalContent()

	if len(listContainer.Objects) != 3 {
		t.Fatalf("expected 3 items in listContainer, got %d", len(listContainer.Objects))
	}

	// Helper to extract email and actionSlot from a row container
	extractRow := func(obj fyne.CanvasObject) (string, *fyne.Container) {
		row, ok := obj.(*fyne.Container)
		if !ok {
			t.Fatalf("expected row to be *fyne.Container, got %T", obj)
		}
		var email string
		var actionSlot *fyne.Container
		for _, child := range row.Objects {
			if lbl, ok := child.(*widget.Label); ok {
				email = lbl.Text
			} else if c, ok := child.(*fyne.Container); ok {
				actionSlot = c
			}
		}
		return email, actionSlot
	}

	rowsByEmail := make(map[string]*fyne.Container)
	for _, obj := range listContainer.Objects {
		em, slot := extractRow(obj)
		rowsByEmail[em] = slot
	}

	// Verify State 1: user1@company.com -> [✔ Đã OAuth] + Nút Xóa
	slot1, ok := rowsByEmail[email1]
	if !ok || slot1 == nil {
		t.Fatalf("row for %s not found", email1)
	}
	badge1, add1, del1 := inspectModalActionSlot(t, slot1)
	if badge1 != "[✔ Đã OAuth]" {
		t.Errorf("expected [✔ Đã OAuth], got '%s'", badge1)
	}
	if add1 != nil {
		t.Errorf("expected no ADD button for state 1")
	}
	if del1 == nil || del1.Icon != theme.DeleteIcon() {
		t.Errorf("expected delete button with DeleteIcon for state 1")
	}

	// Verify State 3: user3@school.edu -> [Đã add] + Nút Xóa
	slot3, ok := rowsByEmail[email3]
	if !ok || slot3 == nil {
		t.Fatalf("row for %s not found", email3)
	}
	badge3, add3, del3 := inspectModalActionSlot(t, slot3)
	if badge3 != "[Đã add]" {
		t.Errorf("expected [Đã add], got '%s'", badge3)
	}
	if add3 != nil {
		t.Errorf("expected no ADD button for state 3")
	}
	if del3 == nil || del3.Icon != theme.DeleteIcon() {
		t.Errorf("expected delete button with DeleteIcon for state 3")
	}

	// Verify State 2: user2@gmail.com -> [Cần OAuth] + ADD button + Nút Xóa
	slot2, ok := rowsByEmail[email2]
	if !ok || slot2 == nil {
		t.Fatalf("row for %s not found", email2)
	}
	badge2, add2, del2 := inspectModalActionSlot(t, slot2)
	if badge2 != "[Cần OAuth]" {
		t.Errorf("expected [Cần OAuth], got '%s'", badge2)
	}
	if add2 == nil || add2.Text != "ADD" || add2.Importance != widget.HighImportance {
		t.Errorf("expected ADD button with HighImportance for state 2")
	}
	if del2 == nil || del2.Icon != theme.DeleteIcon() {
		t.Errorf("expected delete button with DeleteIcon for state 2")
	}
}

func inspectModalActionSlot(t *testing.T, slot *fyne.Container) (badgeText string, addBtn *widget.Button, deleteBtn *widget.Button) {
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
					} else if btn.Text == "ADD" {
						addBtn = btn
					}
				}
			}
		}
	}
	return
}

func TestShowAddAccountModal_AddActionInPlaceMutationAndOverviewUpdate(t *testing.T) {
	mgr := createTestManager(t)
	testApp := test.NewApp()
	app := newIsolatedApp(testApp, mgr)
	app.Init()
	defer app.Close()

	discStorePath := filepath.Join(filepath.Dir(mgr.Store().GetStoreDir()), "discovered_accounts")
	discStore, err := store.NewDiscoveredStore(discStorePath)
	if err != nil {
		t.Fatalf("failed creating discovered store: %v", err)
	}

	targetEmail := "newaccount@gmail.com"
	if err := discStore.SaveDiscovered(targetEmail); err != nil {
		t.Fatalf("SaveDiscovered error: %v", err)
	}

	_, _, listContainer, _ := app.buildAddAccountModalContent()

	if len(listContainer.Objects) != 1 {
		t.Fatalf("expected 1 item, got %d", len(listContainer.Objects))
	}

	row := listContainer.Objects[0].(*fyne.Container)
	var actionSlot *fyne.Container
	for _, child := range row.Objects {
		if c, ok := child.(*fyne.Container); ok {
			actionSlot = c
		}
	}
	if actionSlot == nil {
		t.Fatal("actionSlot not found")
	}

	_, addBtn, delBtn := inspectModalActionSlot(t, actionSlot)
	if addBtn == nil {
		t.Fatal("addBtn not found")
	}
	if delBtn == nil {
		t.Fatal("delBtn not found")
	}

	// Tap the ADD button
	addBtn.OnTapped()

	// 1. Verify in-place reaction: badge now says [Đã add], ADD button is gone, delete button remains
	badgeAfter, addAfter, delAfter := inspectModalActionSlot(t, actionSlot)
	if badgeAfter != "[Đã add]" {
		t.Errorf("expected '[Đã add]', got '%s'", badgeAfter)
	}
	if addAfter != nil {
		t.Errorf("expected add button to disappear after click, got %+v", addAfter)
	}
	if delAfter == nil {
		t.Errorf("expected delete button to remain present after ADD clicked")
	}

	// 2. Verify store persistence: record marked as Added: true
	records, err := discStore.ListDiscoveredRecords()
	if err != nil {
		t.Fatalf("ListDiscoveredRecords error: %v", err)
	}
	if len(records) != 1 || !records[0].Added {
		t.Fatalf("expected 1 record with Added: true, got %+v", records)
	}

	// 3. Wait for async goroutine to refresh overview and call updateUI
	deadline := time.Now().Add(3 * time.Second)
	foundOnUI := false
	for time.Now().Before(deadline) {
		app.mu.RLock()
		ov := app.overview
		app.mu.RUnlock()
		if ov != nil {
			for _, acc := range ov.Accounts {
				if acc.Email == targetEmail && acc.DisplayBadge == "[Cần OAuth]" && acc.State == model.StateUnauth {
					foundOnUI = true
					break
				}
			}
		}
		if foundOnUI {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if !foundOnUI {
		t.Errorf("expected targetEmail %s with badge [Cần OAuth] to appear on main UI overview", targetEmail)
	}
}

func TestShowAddAccountModal_EmailMasking(t *testing.T) {
	mgr := createTestManager(t)
	testApp := test.NewApp()
	app := newIsolatedApp(testApp, mgr)
	app.Init()
	defer app.Close()

	discStorePath := filepath.Join(filepath.Dir(mgr.Store().GetStoreDir()), "discovered_accounts")
	discStore, err := store.NewDiscoveredStore(discStorePath)
	if err != nil {
		t.Fatalf("failed creating discovered store: %v", err)
	}

	email := "privacyuser@example.com"
	if err := discStore.SaveDiscovered(email); err != nil {
		t.Fatalf("SaveDiscovered error: %v", err)
	}

	// Case A: Privacy masking is ON
	app.SetMaskAccounts(true)
	_, _, listContainer1, _ := app.buildAddAccountModalContent()
	row1 := listContainer1.Objects[0].(*fyne.Container)
	var lbl1 *widget.Label
	for _, child := range row1.Objects {
		if l, ok := child.(*widget.Label); ok {
			lbl1 = l
		}
	}
	if lbl1 == nil {
		t.Fatal("email label not found")
	}
	expectedMasked := MaskEmail(email)
	if lbl1.Text != expectedMasked {
		t.Errorf("expected masked email '%s', got '%s'", expectedMasked, lbl1.Text)
	}

	// Case B: Privacy masking is OFF
	app.SetMaskAccounts(false)
	_, _, listContainer2, _ := app.buildAddAccountModalContent()
	row2 := listContainer2.Objects[0].(*fyne.Container)
	var lbl2 *widget.Label
	for _, child := range row2.Objects {
		if l, ok := child.(*widget.Label); ok {
			lbl2 = l
		}
	}
	if lbl2 == nil {
		t.Fatal("email label not found")
	}
	if lbl2.Text != email {
		t.Errorf("expected raw email '%s', got '%s'", email, lbl2.Text)
	}
}

func TestShowAddAccountModal_ScanChromeButtonBehavior(t *testing.T) {
	mgr := createTestManager(t)
	testApp := test.NewApp()
	app := newIsolatedApp(testApp, mgr)
	app.Init()
	defer app.Close()

	_, scanBtn, _, _ := app.buildAddAccountModalContent()
	if scanBtn.Text != "Quét Chrome" {
		t.Errorf("expected button text 'Quét Chrome', got '%s'", scanBtn.Text)
	}
	if scanBtn.Disabled() {
		t.Errorf("expected scanBtn to be enabled initially")
	}

	// Tap the button
	scanBtn.OnTapped()

	// Immediately, button is disabled and text set to "⏳ Đang quét..."
	if scanBtn.Text != "⏳ Đang quét..." {
		t.Errorf("expected scanBtn text '⏳ Đang quét...', got '%s'", scanBtn.Text)
	}
	if !scanBtn.Disabled() {
		t.Errorf("expected scanBtn to be disabled during scan")
	}

	// Wait for background scan goroutine and fyne.Do to complete
	deadline := time.Now().Add(3 * time.Second)
	restored := false
	for time.Now().Before(deadline) {
		if scanBtn.Text == "Quét Chrome" && !scanBtn.Disabled() {
			restored = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if !restored {
		t.Errorf("expected scanBtn to be re-enabled with text 'Quét Chrome' after scan completes")
	}
}

func TestShowAddAccountModal_DialogDisplayAndOverlay(t *testing.T) {
	mgr := createTestManager(t)
	testApp := test.NewApp()
	app := newIsolatedApp(testApp, mgr)
	app.Init()
	defer app.Close()

	if app.win.Canvas().Overlays().Top() != nil {
		t.Fatalf("expected no overlays initially")
	}

	app.ShowAddAccountModal()

	topOverlay := app.win.Canvas().Overlays().Top()
	if topOverlay == nil {
		t.Fatalf("expected active dialog overlay after ShowAddAccountModal()")
	}

	// Hide the dialog overlay
	app.win.Canvas().Overlays().Remove(topOverlay)
	if app.win.Canvas().Overlays().Top() != nil {
		t.Errorf("expected overlays cleared")
	}
}

// TC-Modal-Delete-Button-Rendered: Kiểm tra cả 4 trạng thái dòng tài khoản đều render nút Xóa (theme.DeleteIcon()).
func TestModal_Delete_Button_Rendered(t *testing.T) {
	mgr := createTestManager(t)
	_ = mgr.Store().DeleteAccount("current@gmail.com")
	mgr.SetWincredHandlers(func() (*model.TokenData, error) { return nil, wincred.ErrNotFound }, nil)
	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return nil, errors.New("no language server")
	})
	mgr.SetChromeDir(filepath.Join(t.TempDir(), "empty_chrome"))
	testApp := test.NewApp()
	app := newIsolatedApp(testApp, mgr)
	app.Init()
	defer app.Close()

	discStorePath := filepath.Join(filepath.Dir(mgr.Store().GetStoreDir()), "discovered_accounts")
	discStore, err := store.NewDiscoveredStore(discStorePath)
	if err != nil {
		t.Fatalf("failed creating discovered store: %v", err)
	}

	// 1. HasToken == true, FileExists == true
	email1 := "state1@example.com"
	_ = discStore.SaveDiscovered(email1)
	_ = mgr.Store().SaveAccount(email1, &model.TokenData{AccessToken: "tok", IDToken: makeTestJWT(email1)})

	// 2. Added == true, FileExists == true
	email2 := "state2@example.com"
	_ = discStore.SaveDiscovered(email2)
	_ = discStore.SetDiscoveredAdded(email2, true)

	// 3. !Added && !HasToken, FileExists == true
	email3 := "state3@example.com"
	_ = discStore.SaveDiscovered(email3)

	// 4. FileExists == false (tài khoản không tồn tại trên đĩa)
	email4 := "state4_ghost@example.com"
	mgr.TrackAccount(email4)

	_, _, listContainer, _ := app.buildAddAccountModalContent()

	if len(listContainer.Objects) != 4 {
		t.Fatalf("expected 4 candidate rows, got %d", len(listContainer.Objects))
	}

	rowsByEmail := make(map[string]*fyne.Container)
	for _, obj := range listContainer.Objects {
		row := obj.(*fyne.Container)
		var email string
		var slot *fyne.Container
		for _, child := range row.Objects {
			if lbl, ok := child.(*widget.Label); ok {
				email = lbl.Text
			} else if c, ok := child.(*fyne.Container); ok {
				slot = c
			}
		}
		rowsByEmail[email] = slot
	}

	// Kiểm tra State 1: [✔ Đã OAuth] + Nút Xóa
	badge1, add1, del1 := inspectModalActionSlot(t, rowsByEmail[email1])
	if badge1 != "[✔ Đã OAuth]" || add1 != nil || del1 == nil || del1.Icon != theme.DeleteIcon() {
		t.Errorf("state 1 mismatch: badge=%s, add=%v, del=%v", badge1, add1, del1)
	}

	// Kiểm tra State 2: [Đã add] + Nút Xóa
	badge2, add2, del2 := inspectModalActionSlot(t, rowsByEmail[email2])
	if badge2 != "[Đã add]" || add2 != nil || del2 == nil || del2.Icon != theme.DeleteIcon() {
		t.Errorf("state 2 mismatch: badge=%s, add=%v, del=%v", badge2, add2, del2)
	}

	// Kiểm tra State 3: [Cần OAuth] + Nút ADD + Nút Xóa
	badge3, add3, del3 := inspectModalActionSlot(t, rowsByEmail[email3])
	if badge3 != "[Cần OAuth]" || add3 == nil || del3 == nil || del3.Icon != theme.DeleteIcon() {
		t.Errorf("state 3 mismatch: badge=%s, add=%v, del=%v", badge3, add3, del3)
	}

	// Kiểm tra State 4: [Tài khoản không tồn tại] + Nút Xóa (Ẩn nút ADD)
	badge4, add4, del4 := inspectModalActionSlot(t, rowsByEmail[email4])
	if badge4 != "[Tài khoản không tồn tại]" {
		t.Errorf("state 4 badge mismatch: expected '[Tài khoản không tồn tại]', got '%s'", badge4)
	}
	if add4 != nil {
		t.Errorf("state 4 must NOT have ADD button")
	}
	if del4 == nil || del4.Icon != theme.DeleteIcon() {
		t.Errorf("state 4 must have delete button with DeleteIcon")
	}
}

// TC-Modal-Delete-Click-Action: Nhấn nút Xóa trên dòng tài khoản có file -> xác nhận file bị xóa khỏi discovered_accounts/, dòng biến mất khỏi modal, nhưng thẻ ngoài UI chính không bị mất.
func TestModal_Delete_Click_Action(t *testing.T) {
	mgr := createTestManager(t)
	_ = mgr.Store().DeleteAccount("current@gmail.com")
	mgr.SetWincredHandlers(func() (*model.TokenData, error) { return nil, wincred.ErrNotFound }, nil)
	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return nil, errors.New("no language server")
	})
	mgr.SetChromeDir(filepath.Join(t.TempDir(), "empty_chrome"))
	testApp := test.NewApp()
	app := newIsolatedApp(testApp, mgr)
	app.Init()
	defer app.Close()

	discStorePath := filepath.Join(filepath.Dir(mgr.Store().GetStoreDir()), "discovered_accounts")
	discStore, err := store.NewDiscoveredStore(discStorePath)
	if err != nil {
		t.Fatalf("failed creating discovered store: %v", err)
	}

	email := "delete.target@company.com"
	if err := discStore.SaveDiscovered(email); err != nil {
		t.Fatal(err)
	}
	// Cũng lưu vào accounts_store để có thẻ trên UI chính
	if err := mgr.Store().SaveAccount(email, &model.TokenData{
		AccessToken: "tok_del",
		IDToken:     makeTestJWT(email),
	}); err != nil {
		t.Fatal(err)
	}

	// Cập nhật UI chính để có thẻ
	ov, _ := mgr.GetAccountStates(context.Background())
	app.mu.Lock()
	app.overview = ov
	app.mu.Unlock()
	app.updateUI(ov)

	if len(app.overview.Accounts) != 1 {
		t.Fatalf("expected 1 account on main UI, got %d", len(app.overview.Accounts))
	}

	_, _, listContainer, reloadCandidates := app.buildAddAccountModalContent()
	if len(listContainer.Objects) != 1 {
		t.Fatalf("expected 1 item in modal, got %d", len(listContainer.Objects))
	}

	row := listContainer.Objects[0].(*fyne.Container)
	var actionSlot *fyne.Container
	for _, child := range row.Objects {
		if c, ok := child.(*fyne.Container); ok {
			actionSlot = c
		}
	}
	_, _, delBtn := inspectModalActionSlot(t, actionSlot)
	if delBtn == nil {
		t.Fatal("delete button not found")
	}

	// Nhấn nút Xóa
	delBtn.OnTapped()

	// 1. File bị xóa khỏi discovered_accounts/
	if discStore.HasDiscovered(email) {
		t.Errorf("file in discovered_accounts/ must be deleted")
	}

	// 2. Dòng biến mất khỏi modal (danh sách modal rỗng)
	reloadCandidates()
	if len(listContainer.Objects) != 1 {
		t.Fatalf("expected 1 object (empty label) in listContainer, got %d", len(listContainer.Objects))
	}
	emptyLbl, ok := listContainer.Objects[0].(*widget.Label)
	if !ok || emptyLbl.Text != "Không có tài khoản nào trong danh bạ." {
		t.Errorf("expected empty label in modal, got %v", listContainer.Objects[0])
	}

	// 3. Thẻ ngoài UI chính và file trong accounts_store TUYỆT ĐỐI không bị mất!
	if !mgr.Store().HasAccount(email) {
		t.Errorf("accounts_store credentials must NOT be deleted by modal delete button!")
	}
	app.mu.RLock()
	mainCount := len(app.overview.Accounts)
	app.mu.RUnlock()
	if mainCount != 1 {
		t.Errorf("main UI card must be preserved, got %d accounts", mainCount)
	}
}

// TC-Modal-ScanChrome-NonExistent-Status: Cho 1 tài khoản có trên UI nhưng cố tình xóa file JSON trong discovered_accounts/. Nhấn nút [Quét Chrome] -> kiểm tra modal hiển thị nhãn [Tài khoản không tồn tại], nút ADD bị ẩn, nút Xóa hiển thị.
func TestModal_ScanChrome_NonExistent_Status(t *testing.T) {
	mgr := createTestManager(t)
	_ = mgr.Store().DeleteAccount("current@gmail.com")
	mgr.SetWincredHandlers(func() (*model.TokenData, error) { return nil, wincred.ErrNotFound }, nil)
	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return nil, errors.New("no language server")
	})
	mgr.SetChromeDir(filepath.Join(t.TempDir(), "empty_chrome"))
	testApp := test.NewApp()
	app := newIsolatedApp(testApp, mgr)
	app.Init()
	defer app.Close()

	discStorePath := filepath.Join(filepath.Dir(mgr.Store().GetStoreDir()), "discovered_accounts")
	discStore, err := store.NewDiscoveredStore(discStorePath)
	if err != nil {
		t.Fatalf("failed creating discovered store: %v", err)
	}

	ghostEmail := "ghost.account@gmail.com"
	if err := discStore.SaveDiscovered(ghostEmail); err != nil {
		t.Fatal(err)
	}
	if err := discStore.SetDiscoveredAdded(ghostEmail, true); err != nil {
		t.Fatal(err)
	}

	// Nạp lên UI chính
	ov, _ := mgr.GetAccountStates(context.Background())
	app.mu.Lock()
	app.overview = ov
	app.mu.Unlock()
	app.updateUI(ov)

	// Cố tình xóa file JSON trong discovered_accounts/ để biến thành tài khoản ma
	jsonFile := filepath.Join(discStorePath, ghostEmail+".json")
	if err := os.Remove(jsonFile); err != nil {
		t.Fatalf("failed removing discovered json file: %v", err)
	}
	if discStore.HasDiscovered(ghostEmail) {
		t.Fatal("file should not exist in discoveredStore")
	}

	// Mở Modal và nhấn nút [Quét Chrome]
	_, scanBtn, listContainer, _ := app.buildAddAccountModalContent()

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

	// Kiểm tra modal hiển thị nhãn [Tài khoản không tồn tại], nút ADD bị ẩn, nút Xóa hiển thị
	if len(listContainer.Objects) != 1 {
		t.Fatalf("expected 1 item in listContainer, got %d", len(listContainer.Objects))
	}
	row := listContainer.Objects[0].(*fyne.Container)
	var actionSlot *fyne.Container
	for _, child := range row.Objects {
		if c, ok := child.(*fyne.Container); ok {
			actionSlot = c
		}
	}
	badge, addBtn, delBtn := inspectModalActionSlot(t, actionSlot)
	if badge != "[Tài khoản không tồn tại]" {
		t.Errorf("expected '[Tài khoản không tồn tại]', got '%s'", badge)
	}
	if addBtn != nil {
		t.Errorf("ADD button must be hidden for non-existent account")
	}
	if delBtn == nil || delBtn.Icon != theme.DeleteIcon() {
		t.Errorf("delete button must be rendered for non-existent account")
	}
}

// TC-Modal-Delete-NonExistent: Bấm nút Xóa trên dòng [Tài khoản không tồn tại] -> kiểm tra dòng đó bị xóa sạch khỏi danh sách modal.
func TestModal_Delete_NonExistent(t *testing.T) {
	mgr := createTestManager(t)
	_ = mgr.Store().DeleteAccount("current@gmail.com")
	mgr.SetWincredHandlers(func() (*model.TokenData, error) { return nil, wincred.ErrNotFound }, nil)
	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return nil, errors.New("no language server")
	})
	mgr.SetChromeDir(filepath.Join(t.TempDir(), "empty_chrome"))
	testApp := test.NewApp()
	app := newIsolatedApp(testApp, mgr)
	app.Init()
	defer app.Close()

	ghostEmail := "ghost.delete@gmail.com"
	// Giả lập tài khoản có trên UI nhưng không có file trong discoveredStore
	app.mu.Lock()
	app.overview = &manager.AccountOverview{
		Accounts: []manager.AccountDisplayItem{
			{Email: ghostEmail, State: model.StateUnauth, DisplayBadge: "[Cần OAuth]"},
		},
	}
	app.mu.Unlock()

	_, _, listContainer, _ := app.buildAddAccountModalContent()
	if len(listContainer.Objects) != 1 {
		t.Fatalf("expected 1 item in listContainer, got %d", len(listContainer.Objects))
	}

	row := listContainer.Objects[0].(*fyne.Container)
	var actionSlot *fyne.Container
	for _, child := range row.Objects {
		if c, ok := child.(*fyne.Container); ok {
			actionSlot = c
		}
	}
	badge, _, delBtn := inspectModalActionSlot(t, actionSlot)
	if badge != "[Tài khoản không tồn tại]" {
		t.Fatalf("expected '[Tài khoản không tồn tại]', got '%s'", badge)
	}
	if delBtn == nil {
		t.Fatal("delete button not found")
	}

	// Bấm nút Xóa trên dòng [Tài khoản không tồn tại]
	delBtn.OnTapped()

	// Kiểm tra dòng đó bị xóa sạch khỏi danh sách modal (hiển thị empty label)
	if len(listContainer.Objects) != 1 {
		t.Fatalf("expected 1 object (empty label), got %d", len(listContainer.Objects))
	}
	emptyLbl, ok := listContainer.Objects[0].(*widget.Label)
	if !ok || emptyLbl.Text != "Không có tài khoản nào trong danh bạ." {
		t.Errorf("expected empty label after deleting non-existent row, got %v", listContainer.Objects[0])
	}
}

