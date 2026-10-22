package ui

import (
	"path/filepath"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"antigravity-switch/internal/model"
	"antigravity-switch/internal/store"
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

	// Verify State 1: user1@company.com -> [✔ Đã OAuth]
	slot1, ok := rowsByEmail[email1]
	if !ok || slot1 == nil {
		t.Fatalf("row for %s not found", email1)
	}
	// Center container -> 1 object: *canvas.Text
	if len(slot1.Objects) != 1 {
		t.Fatalf("expected 1 object in slot1, got %d", len(slot1.Objects))
	}
	txt1, ok := slot1.Objects[0].(*canvas.Text)
	if !ok {
		t.Fatalf("expected *canvas.Text in slot1, got %T", slot1.Objects[0])
	}
	if txt1.Text != "[✔ Đã OAuth]" {
		t.Errorf("expected [✔ Đã OAuth], got '%s'", txt1.Text)
	}
	if txt1.TextSize != 11 || !txt1.TextStyle.Bold {
		t.Errorf("expected size 11 bold for badge 1")
	}

	// Verify State 3: user3@school.edu -> [Đã add]
	slot3, ok := rowsByEmail[email3]
	if !ok || slot3 == nil {
		t.Fatalf("row for %s not found", email3)
	}
	if len(slot3.Objects) != 1 {
		t.Fatalf("expected 1 object in slot3, got %d", len(slot3.Objects))
	}
	txt3, ok := slot3.Objects[0].(*canvas.Text)
	if !ok {
		t.Fatalf("expected *canvas.Text in slot3, got %T", slot3.Objects[0])
	}
	if txt3.Text != "[Đã add]" {
		t.Errorf("expected [Đã add], got '%s'", txt3.Text)
	}
	if txt3.TextSize != 11 || !txt3.TextStyle.Bold {
		t.Errorf("expected size 11 bold for badge 3")
	}

	// Verify State 2: user2@gmail.com -> [Cần OAuth] + ADD button
	slot2, ok := rowsByEmail[email2]
	if !ok || slot2 == nil {
		t.Fatalf("row for %s not found", email2)
	}
	if len(slot2.Objects) != 2 {
		t.Fatalf("expected 2 objects in slot2 (badge container + compactButton), got %d", len(slot2.Objects))
	}
	centerBadge, ok := slot2.Objects[0].(*fyne.Container)
	if !ok || len(centerBadge.Objects) < 1 {
		t.Fatalf("expected center container for badge in slot2")
	}
	txt2, ok := centerBadge.Objects[0].(*canvas.Text)
	if !ok || txt2.Text != "[Cần OAuth]" {
		t.Errorf("expected [Cần OAuth], got '%v'", txt2)
	}

	compactBtn, ok := slot2.Objects[1].(*fyne.Container)
	if !ok || len(compactBtn.Objects) < 1 {
		t.Fatalf("expected compactButton container in slot2")
	}
	addBtn, ok := compactBtn.Objects[0].(*widget.Button)
	if !ok {
		t.Fatalf("expected *widget.Button in compact container, got %T", compactBtn.Objects[0])
	}
	if addBtn.Text != "ADD" {
		t.Errorf("expected ADD button text, got '%s'", addBtn.Text)
	}
	if addBtn.Importance != widget.HighImportance {
		t.Errorf("expected HighImportance for ADD button")
	}
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

	compactBtn := actionSlot.Objects[1].(*fyne.Container)
	addBtn := compactBtn.Objects[0].(*widget.Button)

	// Tap the ADD button
	addBtn.OnTapped()

	// 1. Verify in-place reaction: actionSlot now has only doneBadge [Đã add]
	if len(actionSlot.Objects) != 1 {
		t.Fatalf("expected 1 object in actionSlot after ADD, got %d", len(actionSlot.Objects))
	}
	doneBadge, ok := actionSlot.Objects[0].(*canvas.Text)
	if !ok {
		t.Fatalf("expected *canvas.Text after ADD, got %T", actionSlot.Objects[0])
	}
	if doneBadge.Text != "[Đã add]" {
		t.Errorf("expected '[Đã add]', got '%s'", doneBadge.Text)
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
