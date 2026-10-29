package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"antigravity-switch/internal/engines/antigravity/connector"
	"antigravity-switch/internal/engines/antigravity/manager"
)

// TC-01: Header Row 2 Tinh Gọn (Clean Header)
// Tiêu chí PASS:
// - row2LeftCluster chỉ chứa cụm nút bấm, phần tử đầu tiên chính là nút Thêm (a.addAccountBtn).
// - Không còn chứa a.serverStatusLabel trong bất kỳ container nào của Top Section.
func TestAcceptance_TC01_HeaderRow2_CleanLayout(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	mgr := createTestManager(t)
	app := newIsolatedApp(testApp, mgr)
	defer app.Close()

	app.Init()
	waitForIdle(app, 1*time.Second)

	win := app.Window()
	if win == nil {
		t.Fatal("expected non-nil window after Init")
	}

	topSection := app.TopSection()
	if topSection == nil {
		t.Fatalf("expected topSection container with separator")
	}

	// 1. Kiểm tra không còn serverStatusLabel trong toàn bộ cây của topSection
	var containsServerStatus func(fyne.CanvasObject) bool
	containsServerStatus = func(obj fyne.CanvasObject) bool {
		if obj == app.serverStatusLabel {
			return true
		}
		if c, ok := obj.(*fyne.Container); ok {
			for _, child := range c.Objects {
				if containsServerStatus(child) {
					return true
				}
			}
		}
		return false
	}

	if containsServerStatus(topSection) {
		t.Fatal("TC-01 VIOLATION: serverStatusLabel must NOT be present in topSection")
	}

	// 2. row2Combined (Toolbar)
	row2Combined := app.Toolbar()
	if row2Combined == nil {
		t.Fatal("TC-01 VIOLATION: app.Toolbar() must be non-nil row2Combined container")
	}

	var foundLeftCluster *fyne.Container
	for _, obj := range row2Combined.Objects {
		if c, ok := obj.(*fyne.Container); ok {
			for _, inner := range c.Objects {
				if innerC, ok := inner.(*fyne.Container); ok {
					for _, deep := range innerC.Objects {
						if deep == app.addAccountBtn {
							foundLeftCluster = c
							break
						}
					}
				}
			}
		}
	}

	if foundLeftCluster == nil {
		t.Fatal("TC-01 VIOLATION: expected to find row2LeftCluster containing toolbar buttons")
	}

	// Phần tử đầu tiên của cụm nút bấm phải là nút Thêm
	if len(foundLeftCluster.Objects) < 1 {
		t.Fatalf("TC-01 VIOLATION: row2LeftCluster has no buttons")
	}
	btn0Wrap, ok := foundLeftCluster.Objects[0].(*fyne.Container)
	if !ok || len(btn0Wrap.Objects) == 0 || btn0Wrap.Objects[0] != app.addAccountBtn {
		t.Fatalf("TC-01 VIOLATION: first item in row2LeftCluster must wrap addAccountBtn")
	}
	if app.addAccountBtn.Text != "Thêm" {
		t.Errorf("TC-01 VIOLATION: expected first button text 'Thêm', got '%s'", app.addAccountBtn.Text)
	}
}

// TC-02: Thanh Đáy Chuẩn Mực & Tiền Tố Antigravity (Footer Placement & Prefix)
// Tiêu chí PASS:
// - a.serverStatusLabel nằm ở mép trái của footerContent.
// - Khi chưa kết nối: chuỗi hiển thị bắt đầu bằng "Antigravity: Hoạt động  | Đang kết nối...".
// - Khi kết nối thành công: chuỗi hiển thị có định dạng "Antigravity: Hoạt động  | PID %d | Port %d".
// - Khi không có IDE: chuỗi hiển thị là "Antigravity: Không hoạt động  | Dự phòng WinCred".
func TestAcceptance_TC02_FooterPlacementAndPrefix(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	mgr := createTestManager(t)
	app := newIsolatedApp(testApp, mgr)
	defer app.Close()

	app.Init()
	waitForIdle(app, 1*time.Second)

	// Kiểm tra vị trí footer: tìm bottomSection trong content
	win := app.Window()
	content := win.Content().(*fyne.Container)

	var bottomSection *fyne.Container
	var footerContent *fyne.Container

	for _, obj := range content.Objects {
		if c, ok := obj.(*fyne.Container); ok {
			for _, inner := range c.Objects {
				if innerPadded, ok := inner.(*fyne.Container); ok {
					for _, deep := range innerPadded.Objects {
						if deepC, ok := deep.(*fyne.Container); ok {
							for _, footerObj := range deepC.Objects {
								if footerContains(footerObj, app.serverStatusLabel) {
									bottomSection = c
									footerContent = deepC
									break
								}
							}
						}
					}
				}
			}
		}
	}

	if bottomSection == nil {
		t.Fatalf("TC-02 VIOLATION: failed to locate bottomSection containing serverStatusLabel")
	}
	if footerContent == nil {
		t.Fatalf("TC-02 VIOLATION: failed to locate footerContent container")
	}

	// bottomSection chứa: [0] Separator, [1] Padded(footerContent)
	if len(bottomSection.Objects) < 2 {
		t.Fatalf("expected bottomSection to contain separator and padded footerContent, got %d", len(bottomSection.Objects))
	}
	if _, ok := bottomSection.Objects[0].(*widget.Separator); !ok {
		t.Errorf("expected bottomSection.Objects[0] to be *widget.Separator")
	}

	// footerContent chứa: serverStatusLabel (Left), versionLabel (Right), statusBar (Center)
	var foundServerStatus, foundVersion, foundStatusBar bool
	for _, obj := range footerContent.Objects {
		if footerContains(obj, app.serverStatusLabel) {
			foundServerStatus = true
		}
		if footerContains(obj, app.versionLabel) {
			foundVersion = true
		}
		if obj == app.statusBar {
			foundStatusBar = true
		}
	}
	if !foundServerStatus {
		t.Fatal("TC-02 VIOLATION: serverStatusLabel must be in footerContent")
	}
	if !foundVersion {
		t.Fatal("TC-02 VIOLATION: versionLabel must be in footerContent")
	}
	if !foundStatusBar {
		t.Fatal("TC-02 VIOLATION: statusBar must be in footerContent")
	}

	// 1. Trạng thái kết nối thành công: "Antigravity: Hoạt động  | PID 19224 | Port 57868"
	overviewConnected := &manager.AccountOverview{
		LanguageServerRunning: true,
		ServerInfo:            &connector.ServerInfo{PID: 19224, Port: 57868},
	}
	app.updateUI(overviewConnected)
	expectedConnected := fmt.Sprintf("Antigravity: Hoạt động  | PID %d | Port %d", 19224, 57868)
	if app.serverStatusLabel.Text != expectedConnected {
		t.Errorf("TC-02 VIOLATION: expected '%s', got '%s'", expectedConnected, app.serverStatusLabel.Text)
	}

	// 2. Trạng thái không có Language Server: "Antigravity: Không hoạt động  | Dự phòng WinCred"
	overviewDisconnected := &manager.AccountOverview{
		LanguageServerRunning: false,
	}
	app.updateUI(overviewDisconnected)
	expectedDisconnected := "Antigravity: Không hoạt động  | Dự phòng WinCred"
	if app.serverStatusLabel.Text != expectedDisconnected {
		t.Errorf("TC-02 VIOLATION: expected '%s', got '%s'", expectedDisconnected, app.serverStatusLabel.Text)
	}
}

// TC-03: Nhãn Phiên Bản Mép Phải (Version Display & Formatting)
// Tiêu chí PASS:
// - a.versionLabel nằm ở mép phải của footerContent, căn lề phải (TextAlignTrailing).
// - Chuỗi hiển thị chính xác là "Version: 1.0.2.0" (chuẩn thương mại sạch, không chứa commit hash).
// - Tuyệt đối không chứa chuỗi "rev" hoặc dấu ngoặc "(".
func TestAcceptance_TC03_VersionDisplayAndFormatting(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	mgr := createTestManager(t)
	app := newIsolatedApp(testApp, mgr)
	defer app.Close()

	app.Init()
	waitForIdle(app, 1*time.Second)

	if app.versionLabel == nil {
		t.Fatal("TC-03 VIOLATION: versionLabel must not be nil")
	}

	if app.versionLabel.Alignment != fyne.TextAlignTrailing {
		t.Errorf("TC-03 VIOLATION: expected versionLabel.Alignment == fyne.TextAlignTrailing, got %v", app.versionLabel.Alignment)
	}

	expectedVersion := "Version: 1.0.3.0"
	if app.versionLabel.Text != expectedVersion {
		t.Errorf("TC-03 VIOLATION: expected versionLabel text '%s', got '%s'", expectedVersion, app.versionLabel.Text)
	}

	if strings.Contains(app.versionLabel.Text, "rev") {
		t.Errorf("TC-03 VIOLATION: versionLabel must NOT contain 'rev', got '%s'", app.versionLabel.Text)
	}

	if strings.Contains(app.versionLabel.Text, "(") {
		t.Errorf("TC-03 VIOLATION: versionLabel must NOT contain '(', got '%s'", app.versionLabel.Text)
	}
}

// TC-04: Cơ Chế Ẩn/Hiện Chiếm Trọn Chiều Ngang (Full-Width Notification Takeover)
// Tiêu chí PASS:
// - Ban đầu (Idle): a.serverStatusLabel.Visible() == true, a.versionLabel.Visible() == true, a.statusBar.Visible() == false.
// - Khi gọi a.showStatusBar("🔒 Đã ẩn thông tin tài khoản"):
//   - a.serverStatusLabel.Visible() == false.
//   - a.versionLabel.Visible() == false.
//   - a.statusBar.Visible() == true.
//   - a.statusBar.Text == "🔒 Đã ẩn thông tin tài khoản".
//
// - Khi gọi a.clearStatusBar():
//   - a.statusBar.Visible() == false.
//   - a.serverStatusLabel.Visible() == true.
//   - a.versionLabel.Visible() == true.
func TestAcceptance_TC04_FullWidthNotificationTakeover(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	mgr := createTestManager(t)
	app := newIsolatedApp(testApp, mgr)
	defer app.Close()

	app.Init()
	waitForIdle(app, 1*time.Second)

	// Đảm bảo kết thúc thông báo khởi động và chuyển về trạng thái Idle thuần túy
	app.clearStatusBar()

	// 1. Trạng thái ban đầu (Idle)
	if !app.serverStatusLabel.Visible() {
		t.Fatal("TC-04 VIOLATION: serverStatusLabel must be visible in Idle state")
	}
	if !app.versionLabel.Visible() {
		t.Fatal("TC-04 VIOLATION: versionLabel must be visible in Idle state")
	}
	if app.statusBar.Visible() {
		t.Fatal("TC-04 VIOLATION: statusBar must be hidden in Idle state")
	}

	// 2. Kích hoạt thông báo: Chiếm trọn 100% bề ngang thanh đáy
	msg := "🔒 Đã ẩn thông tin tài khoản (bảo vệ quyền riêng tư người dùng)"
	app.showStatusBar(msg)

	if app.serverStatusLabel.Visible() {
		t.Errorf("TC-04 VIOLATION: serverStatusLabel must be hidden when statusBar is active")
	}
	if app.versionLabel.Visible() {
		t.Errorf("TC-04 VIOLATION: versionLabel must be hidden when statusBar is active")
	}
	if !app.statusBar.Visible() {
		t.Errorf("TC-04 VIOLATION: statusBar must be visible when showStatusBar is called")
	}
	if app.statusBar.Text != msg {
		t.Errorf("TC-04 VIOLATION: expected statusBar text '%s', got '%s'", msg, app.statusBar.Text)
	}

	// 3. Xóa thông báo: Khôi phục lại trạng thái bình thường cho thanh đáy
	app.clearStatusBar()

	if app.statusBar.Visible() {
		t.Errorf("TC-04 VIOLATION: statusBar must be hidden after clearStatusBar")
	}
	if !app.serverStatusLabel.Visible() {
		t.Errorf("TC-04 VIOLATION: serverStatusLabel must be restored (visible) after clearStatusBar")
	}
	if !app.versionLabel.Visible() {
		t.Errorf("TC-04 VIOLATION: versionLabel must be restored (visible) after clearStatusBar")
	}

	// 4. Kiểm tra autoHideStatusBar: tự động khôi phục sau thời gian quy định
	app.showStatusBar("Thông báo tạm thời")
	if !app.statusBar.Visible() || app.serverStatusLabel.Visible() {
		t.Fatal("TC-04 VIOLATION: statusBar not active during showStatusBar")
	}
	app.autoHideStatusBar(50 * time.Millisecond)
	time.Sleep(100 * time.Millisecond)

	if app.statusBar.Visible() {
		t.Errorf("TC-04 VIOLATION: statusBar should be hidden after autoHideStatusBar timeout")
	}
	if !app.serverStatusLabel.Visible() {
		t.Errorf("TC-04 VIOLATION: serverStatusLabel should be restored after autoHideStatusBar timeout")
	}
	if !app.versionLabel.Visible() {
		t.Errorf("TC-04 VIOLATION: versionLabel should be restored after autoHideStatusBar timeout")
	}
}

// TC-05: Thu Gọn 50% Độ Cao Thanh Đáy (Footer Bar Height <= 25px)
// Tiêu chí PASS:
// - bottomSection có cấu trúc chuẩn: separator ở index 0, container chứa footerContent ở index 1.
// - Chiều cao MinSize của bottomSection <= 25.0px (giảm ~50% so với layout đệm cũ ~48px).
func TestAcceptance_TC05_FooterCompactHeight(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	mgr := createTestManager(t)
	app := newIsolatedApp(testApp, mgr)
	defer app.Close()

	app.Init()
	waitForIdle(app, 1*time.Second)

	win := app.Window()
	content := win.Content().(*fyne.Container)

	var bottomSection *fyne.Container
	for _, obj := range content.Objects {
		if c, ok := obj.(*fyne.Container); ok {
			for _, inner := range c.Objects {
				if innerWrapper, ok := inner.(*fyne.Container); ok {
					for _, deep := range innerWrapper.Objects {
						if deepC, ok := deep.(*fyne.Container); ok {
							for _, footerObj := range deepC.Objects {
								if footerContains(footerObj, app.serverStatusLabel) {
									bottomSection = c
									break
								}
							}
						}
					}
				}
			}
		}
	}

	if bottomSection == nil {
		t.Fatalf("TC-05 VIOLATION: failed to locate bottomSection containing serverStatusLabel")
	}

	// Đo lường MinSize của bottomSection
	h := bottomSection.MinSize().Height
	if h > 25.0 {
		t.Errorf("TC-05 VIOLATION: expected bottomSection height <= 25px (50%% reduced), got %f px", h)
	}
}

// Footer labels can be nested in contextual stacks; placement remains at the edges.
func footerContains(root, target fyne.CanvasObject) bool {
	if root == target {
		return true
	}
	if c, ok := root.(*fyne.Container); ok {
		for _, child := range c.Objects {
			if footerContains(child, target) {
				return true
			}
		}
	}
	return false
}
