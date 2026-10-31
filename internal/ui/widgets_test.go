package ui

import (
	"fmt"
	"image/color"
	"sync"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"as-auto-switcher/internal/engines/antigravity/manager"
	"as-auto-switcher/internal/engines/antigravity/telemetry"
)

func TestStatusTextButton_RenderingAndHover(t *testing.T) {
	palette := DarkPalette
	tapped := false
	btn := newStatusTextButton("on", func() {
		tapped = true
	}, palette)

	// Initial State: "on"
	if btn.Status != "on" {
		t.Fatalf("expected status 'on', got %q", btn.Status)
	}
	if btn.LabelText() != "[ON]" {
		t.Fatalf("expected label '[ON]', got %q", btn.LabelText())
	}
	if btn.Cursor() != desktop.PointerCursor {
		t.Fatalf("expected pointer cursor, got %v", btn.Cursor())
	}
	if btn.text.Color != palette.Success {
		t.Fatalf("expected text color %v, got %v", palette.Success, btn.text.Color)
	}

	// Hover: MouseIn
	btn.MouseIn(&desktop.MouseEvent{})
	if !btn.hovered {
		t.Fatalf("expected hovered true")
	}
	expectedHoverColor := lightenColor(palette.Success, 35)
	if btn.text.Color != expectedHoverColor {
		t.Fatalf("expected hover text color %v, got %v", expectedHoverColor, btn.text.Color)
	}
	expectedHoverBg := color.NRGBA{
		R: palette.Success.R,
		G: palette.Success.G,
		B: palette.Success.B,
		A: 0x24,
	}
	if btn.bg.FillColor != expectedHoverBg {
		t.Fatalf("expected hover bg color %v, got %v", expectedHoverBg, btn.bg.FillColor)
	}

	// Unhover: MouseOut
	btn.MouseOut()
	if btn.hovered {
		t.Fatalf("expected hovered false")
	}
	if btn.text.Color != palette.Success {
		t.Fatalf("expected text color restored to %v, got %v", palette.Success, btn.text.Color)
	}
	if btn.bg.FillColor != color.Transparent {
		t.Fatalf("expected transparent bg, got %v", btn.bg.FillColor)
	}

	// Click / Tap
	btn.Tapped(&fyne.PointEvent{})
	if !tapped {
		t.Fatalf("expected tapped callback to be called")
	}

	// Transition to "off"
	btn.SetStatus("off")
	if btn.Status != "off" {
		t.Fatalf("expected status 'off', got %q", btn.Status)
	}
	if btn.LabelText() != "[OFF]" {
		t.Fatalf("expected label '[OFF]', got %q", btn.LabelText())
	}
	if btn.text.Color != palette.TextMuted {
		t.Fatalf("expected text color %v for OFF, got %v", palette.TextMuted, btn.text.Color)
	}

	// Hover when "off"
	btn.MouseIn(&desktop.MouseEvent{})
	expectedOffHoverBg := color.NRGBA{
		R: palette.TextMuted.R,
		G: palette.TextMuted.G,
		B: palette.TextMuted.B,
		A: 0x24,
	}
	if btn.bg.FillColor != expectedOffHoverBg {
		t.Fatalf("expected hover bg %v for OFF, got %v", expectedOffHoverBg, btn.bg.FillColor)
	}
	btn.MouseOut()
}

func TestSwitchTextButton_DisabledState(t *testing.T) {
	palette := DarkPalette

	// 1. Normal active switch button
	normalClicked := false
	normalBtn := newSwitchTextButton(func() {
		normalClicked = true
	}, palette)

	if normalBtn.IsDisabled() {
		t.Fatalf("expected normal button not to be disabled")
	}
	if normalBtn.Cursor() != desktop.PointerCursor {
		t.Fatalf("expected normal button cursor to be pointer")
	}
	normalBtn.Tapped(&fyne.PointEvent{})
	if !normalClicked {
		t.Fatalf("expected normal button click to invoke callback")
	}

	// 2. Disabled switch button
	disabledBtn := newDisabledSwitchTextButton(palette)
	if !disabledBtn.IsDisabled() {
		t.Fatalf("expected disabled button to have IsDisabled() == true")
	}
	if disabledBtn.Cursor() != desktop.DefaultCursor {
		t.Fatalf("expected default cursor on disabled button, got %v", disabledBtn.Cursor())
	}
	if disabledBtn.text.Color != palette.TextMuted {
		t.Fatalf("expected TextMuted color on disabled button, got %v", disabledBtn.text.Color)
	}

	// Tap disabled button -> must be no-op
	disabledClicked := false
	disabledBtn.OnTapped = func() { disabledClicked = true }
	disabledBtn.Tapped(&fyne.PointEvent{})
	if disabledClicked {
		t.Fatalf("expected disabled button click to be no-op")
	}

	// Hover disabled button -> must not change to hover state
	disabledBtn.MouseIn(&desktop.MouseEvent{})
	if disabledBtn.hovered {
		t.Fatalf("expected disabled button not to hover")
	}
	if disabledBtn.bg.FillColor != color.Transparent {
		t.Fatalf("expected transparent bg on disabled button hover")
	}
}

func TestAccountCardWidget_StatusButton_DefaultOnAndToggle(t *testing.T) {
	test.NewApp()
	palette := DarkPalette

	var switchedEmail, toggledEmail, refreshedEmail string
	item := manager.AccountDisplayItem{
		Email:    "test@example.com",
		HasToken: true,
		IsActive: false,
		Status:   "", // empty defaults to "on"
	}

	card := NewAccountCardWidgetWithQuota(
		item,
		nil,
		palette,
		func(email string) { switchedEmail = email },
		func(email string) { refreshedEmail = email },
		func(email string) { toggledEmail = email },
	)

	// Verify defaults
	if card.Status != "on" {
		t.Fatalf("expected default status 'on', got %q", card.Status)
	}
	if card.statusButton == nil {
		t.Fatalf("expected statusButton not to be nil")
	}
	if !card.statusButton.Visible() {
		t.Fatalf("expected statusButton to be visible on card with token")
	}
	if card.statusButton.LabelText() != "[ON]" {
		t.Fatalf("expected label '[ON]', got %q", card.statusButton.LabelText())
	}

	// Verify switch button is enabled and clickable
	switchBtn, ok := card.ActionButton().(*switchTextButton)
	if !ok || switchBtn == nil {
		t.Fatalf("expected action button to be *switchTextButton")
	}
	if switchBtn.IsDisabled() {
		t.Fatalf("expected switch button not to be disabled when status is 'on'")
	}
	if switchBtn.Cursor() != desktop.PointerCursor {
		t.Fatalf("expected pointer cursor on switch button")
	}

	switchBtn.Tapped(&fyne.PointEvent{})
	if switchedEmail != "test@example.com" {
		t.Fatalf("expected switchedEmail 'test@example.com', got %q", switchedEmail)
	}

	// Verify refresh button is enabled
	if card.refreshButton.Disabled() {
		t.Fatalf("expected refresh button to be enabled when status is 'on'")
	}
	card.refreshButton.Tapped(&fyne.PointEvent{})
	if refreshedEmail != "test@example.com" {
		t.Fatalf("expected refreshedEmail 'test@example.com', got %q", refreshedEmail)
	}

	// Verify status toggle click
	card.statusButton.Tapped(&fyne.PointEvent{})
	if toggledEmail != "test@example.com" {
		t.Fatalf("expected toggledEmail 'test@example.com', got %q", toggledEmail)
	}
}

func TestAccountCardWidget_StatusOff_MutesSwitchAndDisablesRefresh(t *testing.T) {
	test.NewApp()
	palette := DarkPalette

	var switchedEmail, refreshedEmail string
	item := manager.AccountDisplayItem{
		Email:    "off_account@example.com",
		HasToken: true,
		IsActive: false,
		Status:   "off",
	}

	card := NewAccountCardWidgetWithQuota(
		item,
		nil,
		palette,
		func(email string) { switchedEmail = email },
		func(email string) { refreshedEmail = email },
	)

	if card.Status != "off" {
		t.Fatalf("expected status 'off', got %q", card.Status)
	}
	if !card.statusButton.Visible() {
		t.Fatalf("expected statusButton to be visible")
	}
	if card.statusButton.LabelText() != "[OFF]" {
		t.Fatalf("expected label '[OFF]', got %q", card.statusButton.LabelText())
	}

	// Verify switch button is muted/disabled
	switchBtn, ok := card.ActionButton().(*switchTextButton)
	if !ok || switchBtn == nil {
		t.Fatalf("expected action button to be *switchTextButton")
	}
	if !switchBtn.IsDisabled() {
		t.Fatalf("expected switch button to be disabled when status is 'off'")
	}
	if switchBtn.Cursor() != desktop.DefaultCursor {
		t.Fatalf("expected default cursor when switch button is muted/disabled, got %v", switchBtn.Cursor())
	}
	if switchBtn.text.Color != palette.TextMuted {
		t.Fatalf("expected TextMuted color on muted switch button")
	}

	// Click switch button -> must be no-op
	switchBtn.Tapped(&fyne.PointEvent{})
	if switchedEmail != "" {
		t.Fatalf("expected switchedEmail to remain empty, got %q", switchedEmail)
	}

	// Verify refresh button is disabled
	if !card.refreshButton.Disabled() {
		t.Fatalf("expected refresh button to be disabled when status is 'off'")
	}
	card.refreshButton.Tapped(&fyne.PointEvent{})
	if refreshedEmail != "" {
		t.Fatalf("expected refreshedEmail to remain empty, got %q", refreshedEmail)
	}
}

func TestAccountCardWidget_UnauthAccount_HidesStatusButton(t *testing.T) {
	test.NewApp()
	palette := DarkPalette

	item := manager.AccountDisplayItem{
		Email:    "unauth@example.com",
		HasToken: false,
		IsActive: false,
		Status:   "on",
	}

	card := NewAccountCardWidgetWithQuota(item, nil, palette, nil, nil)

	// Thẻ [Cần OAuth]: Tuyệt đối không hiển thị nút [ON/OFF]
	if card.statusButton.Visible() {
		t.Fatalf("expected statusButton to be hidden on unauthenticated card [Cần OAuth]")
	}
	if !card.refreshButton.Disabled() {
		t.Fatalf("expected refresh button to be disabled on unauthenticated card")
	}
}

func TestAccountCardWidget_ActiveAccount_ShowsStatusButton(t *testing.T) {
	test.NewApp()
	palette := DarkPalette

	item := manager.AccountDisplayItem{
		Email:    "active@example.com",
		HasToken: true,
		IsActive: true,
		Status:   "on",
	}

	card := NewAccountCardWidgetWithQuota(item, nil, palette, nil, nil)

	if !card.statusButton.Visible() {
		t.Fatalf("expected statusButton to be visible on active account")
	}
	if card.statusButton.LabelText() != "[ON]" {
		t.Fatalf("expected label '[ON]', got %q", card.statusButton.LabelText())
	}
	if card.refreshButton.Disabled() {
		t.Fatalf("expected refresh button to be enabled for active account with status 'on'")
	}

	// Update to OFF
	item.Status = "off"
	card.UpdateState(item, palette)

	if !card.statusButton.Visible() {
		t.Fatalf("expected statusButton to remain visible on active account even when OFF")
	}
	if card.statusButton.LabelText() != "[OFF]" {
		t.Fatalf("expected label '[OFF]', got %q", card.statusButton.LabelText())
	}
	if !card.refreshButton.Disabled() {
		t.Fatalf("expected refresh button to be disabled when active account is OFF")
	}
}

func TestAccountCardWidget_TransitionFromOnToOffAndBack(t *testing.T) {
	test.NewApp()
	palette := DarkPalette

	switchedCount := 0
	item := manager.AccountDisplayItem{
		Email:    "toggle@example.com",
		HasToken: true,
		IsActive: false,
		Status:   "on",
	}

	card := NewAccountCardWidgetWithQuota(item, nil, palette, func(string) { switchedCount++ }, nil)

	// 1. Initial State "on"
	if card.Status != "on" || card.statusButton.LabelText() != "[ON]" {
		t.Fatalf("initial state should be 'on'")
	}
	btn1, _ := card.ActionButton().(*switchTextButton)
	btn1.Tapped(&fyne.PointEvent{})
	if switchedCount != 1 {
		t.Fatalf("expected switch click to count 1, got %d", switchedCount)
	}

	// 2. Transition to "off"
	item.Status = "off"
	card.UpdateState(item, palette)
	if card.Status != "off" || card.statusButton.LabelText() != "[OFF]" {
		t.Fatalf("state should update to 'off'")
	}
	btn2, _ := card.ActionButton().(*switchTextButton)
	if !btn2.IsDisabled() {
		t.Fatalf("switch button should be disabled when status is 'off'")
	}
	btn2.Tapped(&fyne.PointEvent{})
	if switchedCount != 1 {
		t.Fatalf("switch click should be no-op when status is 'off', count remained %d", switchedCount)
	}
	if !card.refreshButton.Disabled() {
		t.Fatalf("refresh button should be disabled when status is 'off'")
	}

	// 3. Transition back to "on"
	item.Status = "on"
	card.UpdateState(item, palette)
	if card.Status != "on" || card.statusButton.LabelText() != "[ON]" {
		t.Fatalf("state should update back to 'on'")
	}
	btn3, _ := card.ActionButton().(*switchTextButton)
	if btn3.IsDisabled() {
		t.Fatalf("switch button should be enabled again when status is 'on'")
	}
	btn3.Tapped(&fyne.PointEvent{})
	if switchedCount != 2 {
		t.Fatalf("expected switch click to count 2, got %d", switchedCount)
	}
	if card.refreshButton.Disabled() {
		t.Fatalf("refresh button should be enabled again when status is 'on'")
	}
}

func TestAccountCardWidget_DeleteButton_OrderAndHierarchy(t *testing.T) {
	test.NewApp()
	palette := DarkPalette

	item := manager.AccountDisplayItem{
		Email:    "test@example.com",
		HasToken: true,
		IsActive: false,
		Status:   "on",
	}

	deletedEmail := ""
	card := NewAccountCardWidgetWithQuota(item, nil, palette, nil, nil, nil, func(email string) {
		deletedEmail = email
	})

	cluster := card.ActionCluster()
	if cluster == nil {
		t.Fatalf("expected actionCluster to be accessible")
	}
	if len(cluster.Objects) != 4 {
		t.Fatalf("expected actionCluster to have exactly 4 items, got %d", len(cluster.Objects))
	}

	// Verify Invariant 2 strict order: [1. Action Slot] -> [2. Refresh Quota] -> [3. Status ON/OFF] -> [4. Trash Button 🗑]
	if cluster.Objects[0] != card.actionSlot {
		t.Errorf("expected slot 0 to be actionSlot")
	}
	if cluster.Objects[1] != card.actionCluster.Objects[1] {
		t.Errorf("expected slot 1 to be compact refresh button")
	}
	if cluster.Objects[2] != card.statusButton {
		t.Errorf("expected slot 2 to be statusButton")
	}
	if cluster.Objects[3] != card.deleteContainer {
		t.Errorf("expected slot 3 to be compact delete button container")
	}

	btn := card.DeleteButton()
	if btn == nil {
		t.Fatalf("expected DeleteButton() to return *widget.Button")
	}
	if btn.Importance != widget.LowImportance {
		t.Errorf("expected LowImportance, got %v", btn.Importance)
	}

	// Verify click triggering callback
	card.TriggerDelete()
	if deletedEmail != "test@example.com" {
		t.Fatalf("expected deletedEmail 'test@example.com', got %q", deletedEmail)
	}
}

func TestAccountCardWidget_DeleteButton_VisibilityStates(t *testing.T) {
	test.NewApp()
	palette := DarkPalette

	// Case 1: Inactive ON
	card1 := NewAccountCardWidgetWithQuota(manager.AccountDisplayItem{
		Email:    "inactive.on@example.com",
		HasToken: true,
		IsActive: false,
		Status:   "on",
	}, nil, palette, nil, nil)
	if !card1.DeleteButton().Visible() {
		t.Errorf("expected DeleteButton to be visible for Inactive ON")
	}

	// Case 2: Inactive OFF
	card2 := NewAccountCardWidgetWithQuota(manager.AccountDisplayItem{
		Email:    "inactive.off@example.com",
		HasToken: true,
		IsActive: false,
		Status:   "off",
	}, nil, palette, nil, nil)
	if !card2.DeleteButton().Visible() {
		t.Errorf("expected DeleteButton to be visible for Inactive OFF")
	}

	// Case 3: Need OAuth
	card3 := NewAccountCardWidgetWithQuota(manager.AccountDisplayItem{
		Email:    "need.oauth@example.com",
		HasToken: false,
		IsActive: false,
		Status:   "on",
	}, nil, palette, nil, nil)
	if !card3.DeleteButton().Visible() {
		t.Errorf("expected DeleteButton to be visible for Need OAuth")
	}

	// Case 4: Active Account (Invariant 1: Tuyệt đối KHÔNG hiển thị nút Xóa)
	card4 := NewAccountCardWidgetWithQuota(manager.AccountDisplayItem{
		Email:    "active@example.com",
		HasToken: true,
		IsActive: true,
		Status:   "on",
	}, nil, palette, nil, nil)
	if card4.DeleteButton().Visible() {
		t.Errorf("Invariant 1 Violated: DeleteButton MUST be hidden for Active Account")
	}

	// Case 5: Dynamic transition: Active -> Inactive -> Active
	item := manager.AccountDisplayItem{
		Email:    "switchable@example.com",
		HasToken: true,
		IsActive: true,
		Status:   "on",
	}
	dynCard := NewAccountCardWidgetWithQuota(item, nil, palette, nil, nil)
	if dynCard.DeleteButton().Visible() {
		t.Errorf("expected DeleteButton hidden for Active account")
	}

	item.IsActive = false
	dynCard.UpdateState(item, palette)
	if !dynCard.DeleteButton().Visible() {
		t.Errorf("expected DeleteButton visible after becoming Inactive")
	}

	item.IsActive = true
	dynCard.UpdateState(item, palette)
	if dynCard.DeleteButton().Visible() {
		t.Errorf("expected DeleteButton hidden after becoming Active again")
	}
}

func TestAccountCardWidget_TriggerDelete_And_SetOnDelete(t *testing.T) {
	test.NewApp()
	palette := DarkPalette

	deletedEmail := ""
	card := NewAccountCardWidgetWithQuota(manager.AccountDisplayItem{
		Email:    "test@example.com",
		HasToken: true,
		IsActive: false,
		Status:   "on",
	}, nil, palette, nil, nil)

	card.SetOnDelete(func(email string) {
		deletedEmail = email
	})

	card.TriggerDelete()
	if deletedEmail != "test@example.com" {
		t.Errorf("expected deletedEmail 'test@example.com', got %q", deletedEmail)
	}

	// Active card must never trigger onDelete
	activeDeletedEmail := ""
	activeCard := NewAccountCardWidgetWithQuota(manager.AccountDisplayItem{
		Email:    "active@example.com",
		HasToken: true,
		IsActive: true,
		Status:   "on",
	}, nil, palette, nil, nil)
	activeCard.SetOnDelete(func(email string) {
		activeDeletedEmail = email
	})
	activeCard.TriggerDelete()
	if activeDeletedEmail != "" {
		t.Errorf("active card must NOT trigger delete callback")
	}
}

// INV-P3-01 & INV-P3-07: In-place Extension Rows & Public API Signature Preservation
func TestAccountCardWidget_DynamicModels_PreservationAndExtensionRows(t *testing.T) {
	test.NewApp()
	palette := DarkPalette

	item := manager.AccountDisplayItem{
		Email:    "base@example.com",
		HasToken: true,
		IsActive: true,
		Status:   "on",
		Tier:     "PRO",
	}

	// 1. Backward-compatible call with NewAccountCardWidgetWithQuota (K=0)
	cardBase := NewAccountCardWidgetWithQuota(item, nil, palette, nil, nil)
	if cardBase == nil {
		t.Fatal("expected non-nil cardBase")
	}
	if len(cardBase.ExtensionRows()) != 0 {
		t.Errorf("expected 0 extension rows for baseline card, got %d", len(cardBase.ExtensionRows()))
	}
	if cardBase.gemini5hCell == nil || cardBase.geminiWeekCell == nil || cardBase.claude5hCell == nil || cardBase.claudeWeekCell == nil {
		t.Fatal("expected all base matrixQuotaCell pointers to be non-nil")
	}
	if cardBase.gemini5hPct == nil || cardBase.gemini5hBar == nil || cardBase.claude5hPct == nil || cardBase.claude5hBar == nil {
		t.Fatal("expected all base canvas text/bar pointers to be non-nil")
	}

	// 2. NewAccountCardWidgetWithDynamicModels with K=1 (Argon)
	models := []DiscoveredModelInfo{
		{
			ModelID:  "gemini-4-argon",
			Label:    "Gemini 4 Argon",
			Token:    "argon",
			Line1:    "Gemini 4",
			Line2:    "Argon",
			Disabled: false,
		},
	}

	cardDyn := NewAccountCardWidgetWithDynamicModels(item, nil, palette, models, nil, nil)
	if cardDyn == nil {
		t.Fatal("expected non-nil cardDyn")
	}
	extRows := cardDyn.ExtensionRows()
	if len(extRows) != 1 {
		t.Fatalf("expected 1 extension row, got %d", len(extRows))
	}
	if extRows[0].token != "argon" {
		t.Errorf("expected token 'argon', got %q", extRows[0].token)
	}
	if extRows[0].cell5h == nil || extRows[0].cellWeek == nil {
		t.Fatal("expected extension row cells to be non-nil")
	}

	// 3. Wrapper NewAccountCardWithDynamicModels returns CanvasObject
	rootObj := NewAccountCardWithDynamicModels(item, nil, palette, models, nil, nil)
	if rootObj == nil {
		t.Fatal("expected non-nil root CanvasObject")
	}
}

// INV-P3-02 & INV-P3-03: UpdateQuota In-Place for Extension Rows, Error Branch Coverage, and Progress Bar Restoration
func TestAccountCardWidget_UpdateQuota_ExtensionRows_InPlace(t *testing.T) {
	test.NewApp()
	palette := DarkPalette

	item := manager.AccountDisplayItem{
		Email:    "test@example.com",
		HasToken: true,
		IsActive: true,
		Status:   "on",
		Tier:     "ULTRA",
	}

	models := []DiscoveredModelInfo{
		{
			ModelID:  "gemini-4-argon",
			Label:    "Gemini 4 Argon",
			Token:    "argon",
			Line1:    "Gemini 4",
			Line2:    "Argon",
			Disabled: false,
		},
		{
			ModelID:  "gemini-4-disabled",
			Label:    "Gemini 4 Disabled",
			Token:    "disabled",
			Line1:    "Gemini 4",
			Line2:    "Disabled",
			Disabled: true,
		},
	}

	card := NewAccountCardWidgetWithDynamicModels(item, nil, palette, models, nil, nil)
	extRows := card.ExtensionRows()
	if len(extRows) != 2 {
		t.Fatalf("expected 2 extension rows, got %d", len(extRows))
	}
	rowArgon := extRows[0]
	rowDisabled := extRows[1]

	// Step 1: Normal update with quota data (Argon has 85% 5h and 40% Week)
	quota := &telemetry.AccountQuotaData{
		Email: "test@example.com",
		Tier:  "ULTRA",
		RawBuckets: []telemetry.ModelQuotaBucket{
			{
				ModelID:          "gemini-4-argon",
				TokenType:        "argon",
				FiveHourFraction: 0.85,
				WeeklyFraction:   0.40,
			},
		},
	}
	card.UpdateQuota(quota, palette)

	// Verify Argon updated in-place
	if rowArgon.cell5h.pctText.Text != "85%" {
		t.Errorf("expected Argon 5h '85%%', got %q", rowArgon.cell5h.pctText.Text)
	}
	if rowArgon.cellWeek.pctText.Text != "40%" {
		t.Errorf("expected Argon Week '40%%', got %q", rowArgon.cellWeek.pctText.Text)
	}
	// Verify disabled model has "--" and "[LOCKED]"
	if rowDisabled.cell5h.pctText.Text != "--" {
		t.Errorf("expected disabled 5h '--', got %q", rowDisabled.cell5h.pctText.Text)
	}
	if rowDisabled.cellWeek.pctText.Text != "[LOCKED]" {
		t.Errorf("expected disabled Week '[LOCKED]', got %q", rowDisabled.cellWeek.pctText.Text)
	}

	// Step 2: Error branch coverage - Quota error (INV-P3-02 / Bẫy 9)
	errQuota := &telemetry.AccountQuotaData{
		Email:        "test@example.com",
		Status:       "error",
		ErrorMessage: "network failure",
	}
	card.UpdateQuota(errQuota, palette)

	// All base cells must be N/A
	if card.gemini5hPct.Text != "N/A" || card.claude5hPct.Text != "N/A" {
		t.Errorf("expected base cells to be 'N/A' on error, got %q, %q", card.gemini5hPct.Text, card.claude5hPct.Text)
	}
	// All extension rows must be setUnavailable() -> N/A
	if rowArgon.cell5h.pctText.Text != "N/A" {
		t.Errorf("expected Argon 5h 'N/A' on error, got %q", rowArgon.cell5h.pctText.Text)
	}
	if rowArgon.cellWeek.pctText.Text != "N/A" {
		t.Errorf("expected Argon Week 'N/A' on error, got %q", rowArgon.cellWeek.pctText.Text)
	}
	if rowArgon.cell5h.barContainer.Visible() {
		t.Errorf("expected barContainer hidden when setUnavailable")
	}

	// Step 3: Auto-restore progress bar when valid quota returns (INV-P3-03 / Bẫy 8)
	card.UpdateQuota(quota, palette)
	if rowArgon.cell5h.pctText.Text != "85%" {
		t.Errorf("expected Argon 5h restored to '85%%', got %q", rowArgon.cell5h.pctText.Text)
	}
	if !rowArgon.cell5h.barContainer.Visible() {
		t.Errorf("expected barContainer to be automatically restored to Visible")
	}

	// Step 4: Unauthenticated / inactive branch coverage (INV-P3-02)
	card.IsActive = false
	card.HasToken = false
	card.UpdateQuota(quota, palette)
	if rowArgon.cell5h.pctText.Text != "N/A" || rowArgon.cellWeek.pctText.Text != "N/A" {
		t.Errorf("expected extension rows 'N/A' when !IsActive && !HasToken, got %q, %q",
			rowArgon.cell5h.pctText.Text, rowArgon.cellWeek.pctText.Text)
	}
}

// INV-P3-03 / INV-P3-04: Dynamic Grid Cell Height Expansion & Anti-Clipping Invariant
func TestApp_UpdateCardHeight_DynamicGrid(t *testing.T) {
	test.NewApp()
	a := &App{}
	a.accountListContainer = newSymmetricGridWrap(fyne.NewSize(280, 130))
	grid := a.accountListContainer.Layout.(*symmetricGridWrapLayout)

	// Baseline K=0 -> 130px
	a.UpdateCardHeight(0)
	if grid.CellSize.Height != 130.0 {
		t.Errorf("expected height 130.0 for K=0, got %f", grid.CellSize.Height)
	}

	// K=1 -> 130 + 1*32 = 162px
	a.UpdateCardHeight(1)
	if grid.CellSize.Height != 162.0 {
		t.Errorf("expected height 162.0 for K=1, got %f", grid.CellSize.Height)
	}

	// K=2 -> 130 + 2*32 = 194px
	a.UpdateCardHeight(2)
	if grid.CellSize.Height != 194.0 {
		t.Errorf("expected height 194.0 for K=2, got %f", grid.CellSize.Height)
	}

	// Negative K -> clamped to 130px
	a.UpdateCardHeight(-5)
	if grid.CellSize.Height != 130.0 {
		t.Errorf("expected clamped height 130.0 for negative K, got %f", grid.CellSize.Height)
	}
}

// INV-P3-04 & INV-P3-05: Thread-Safe State for Discovered Models
func TestApp_DiscoveredModels_ThreadSafety(t *testing.T) {
	a := &App{}
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(2)
		idx := i
		// Writer goroutine
		go func() {
			defer wg.Done()
			a.SetDiscoveredModels([]DiscoveredModelInfo{
				{
					ModelID:  fmt.Sprintf("model-%d", idx),
					Label:    fmt.Sprintf("Model %d", idx),
					Token:    fmt.Sprintf("tok%d", idx),
					Disabled: idx%2 == 0,
				},
			})
		}()
		// Reader goroutine
		go func() {
			defer wg.Done()
			_ = a.DiscoveredModels()
		}()
	}
	wg.Wait()

	models := a.DiscoveredModels()
	if len(models) != 1 {
		t.Errorf("expected 1 model after concurrent writes, got %d", len(models))
	}
}

// INV-P3-06 & INV-P3-07: Graceful Fallback on IDE Disconnection (AC-11)
func TestApp_GracefulFallback_OnDisconnection(t *testing.T) {
	test.NewApp()
	palette := DarkPalette

	a := &App{}
	a.accountListContainer = newSymmetricGridWrap(fyne.NewSize(280, 130))

	models := []DiscoveredModelInfo{
		{
			ModelID:  "gemini-4-argon",
			Label:    "Gemini 4 Argon",
			Token:    "argon",
			Line1:    "Gemini 4",
			Line2:    "Argon",
			Disabled: false,
		},
	}
	a.SetDiscoveredModels(models)
	a.UpdateCardHeight(len(models))

	grid := a.accountListContainer.Layout.(*symmetricGridWrapLayout)
	if grid.CellSize.Height != 162.0 {
		t.Fatalf("expected 162px height, got %f", grid.CellSize.Height)
	}

	// Simulate IDE disconnection: Language Server stops (nextRunning == false).
	// Per INV-P3-07: retain layout and discovered models (prevent layout thrashing).
	retainedModels := a.DiscoveredModels()
	if len(retainedModels) != 1 {
		t.Errorf("expected retained models after disconnection, got %d", len(retainedModels))
	}
	if grid.CellSize.Height != 162.0 {
		t.Errorf("expected grid height preserved at 162px to avoid thrashing, got %f", grid.CellSize.Height)
	}

	// Card with inactive state transitions smoothly to N/A without crash
	item := manager.AccountDisplayItem{
		Email:    "fallback@example.com",
		HasToken: false,
		IsActive: false,
	}
	card := NewAccountCardWidgetWithDynamicModels(item, nil, palette, retainedModels, nil, nil)
	card.UpdateQuota(nil, palette)

	extRows := card.ExtensionRows()
	if len(extRows) != 1 {
		t.Fatalf("expected 1 extension row, got %d", len(extRows))
	}
	if extRows[0].cell5h.pctText.Text != "N/A" || extRows[0].cellWeek.pctText.Text != "N/A" {
		t.Errorf("expected fallback extension cells to display N/A, got %q, %q",
			extRows[0].cell5h.pctText.Text, extRows[0].cellWeek.pctText.Text)
	}
}
