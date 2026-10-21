package ui

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"

	"antigravity-switch/internal/manager"
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
