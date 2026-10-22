# ĐẶC TẢ KIẾN TRÚC & KỸ THUẬT: THANH ĐÁY (FOOTER STATUS BAR), TỰ ĐỘNG ĐỒNG BỘ VERSION & CƠ CHẾ ẨN/HIỆN THÔNG BÁO

- **Tài liệu:** Đặc tả Kỹ thuật Kiến trúc & Hướng dẫn Thi công Chi tiết (Software Architecture & Implementation Blueprint)
- **Mã định danh:** `SPEC-ARCH-20261022-FOOTER-STATUS-BAR-AND-VERSION-SYNC-SPEC`
- **Tác giả:** System Architect (Strategic Lane)
- **Ngày phê duyệt:** 2026-10-22 (Đã thảo luận và thống nhất toàn diện với Human Owner)
- **Trạng thái:** APPROVED / READY FOR PLANNER & CODER
- **Đối tượng đọc:** Strategic Planner, Coder Lane, Supervisor Lane, CEO Agent

---

# PHẦN I: BỐI CẢNH, NGUYÊN TẮC THIẾT KẾ & TỔNG QUAN GIẢI PHÁP

## 1. Hiện trạng Codebase cũ

Hiện tại trên thanh công cụ dòng 2 của cửa sổ ứng dụng ([`internal/ui/window.go:121-156`](file:///g:/antigravity_auto_switch/internal/ui/window.go#L121-L156)):
```text
[Hoạt động  | PID 19224 | Port 57868]  [ [+ Thêm] [Danh sách tài khoản] [Làm mới Quota] [📌] ] ...
```
- Cụm text trạng thái tiến trình `a.serverStatusLabel` đứng ngay đầu góc trái, làm xô lệch và chiếm dụng không gian của 4 nút điều khiển chính.
- Text chỉ ghi cụt ngủn *"Hoạt động"* mà không nêu rõ là dịch vụ hay tiến trình nào đang hoạt động (gây mơ hồ ngữ nghĩa).
- Ứng dụng hoàn toàn thiếu khu vực hiển thị số phiên bản (`Version`), gây khó khăn cho người dùng và kỹ thuật viên khi kiểm tra phiên bản binary đang chạy.
- Thanh thông báo (`a.statusBar`) nằm ở đáy nhưng chưa có khung viền phân cách, khi xuất hiện thông báo thì không gian bị hạn chế bởi kích thước cửa sổ.

---

## 2. Nguyên Tắc Thiết Kế Trực Quan (Visual & UX Blueprint)

Theo thống nhất với Human Owner, hệ thống được cấu trúc lại theo chuẩn mực Desktop:
1. **Header Row 2 (Thanh công cụ trên cùng):**
   - Giải phóng hoàn toàn góc trái: Bắt đầu ngay lập tức bằng cụm 4 nút điều khiển nghiệp vụ:
     `[+ Thêm]  [Danh sách tài khoản]  [Làm mới Quota]  📌`
   - Giao diện phía trên thoáng đãng, cân đối, không bị chèn ép khi co giãn cửa sổ.

2. **Footer Status Bar (Thanh đáy mới):**
   - Được phân tách với vùng lưới thẻ tài khoản bằng một đường kẻ phân cách mảnh (`widget.NewSeparator()`).
   - Có 2 trạng thái hiển thị luân phiên cực kỳ thông minh:

### **Trạng thái 1: Bình thường (Idle / Telemetry Mode)**
```text
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ AS  Antigravity Multi-Account Switcher                                                  —  □  ✕  │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│ [+ Thêm]  [Danh sách tài khoản]  [Làm mới Quota]  📌          Ẩn tài khoản: [○]  Tự động: [●]  🌙 │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                                  │
│   [Thẻ tài khoản 1]        [Thẻ tài khoản 2]        [Thẻ tài khoản 3]                           │
│   [Thẻ tài khoản 4]        [Thẻ tài khoản 5]        [Thẻ tài khoản 6]                           │
│                                (Vùng cuộn lưới thẻ tài khoản)                                    │
│                                                                                                  │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│ ──────────────────────────────── (Đường kẻ mảnh Separator) ──────────────────────────────────── │
│ Antigravity: Hoạt động | PID 19224 | Port 57868                Version: 1.0.2.0 (rev a77efb9)    │
└──────────────────────────────────────────────────────────────────────────────────────────────────┘
  ▲ (Mép trái: Rõ tiến trình Antigravity)                          ▲ (Mép phải: Version kèm commit)
```

### **Trạng thái 2: Khi có thông báo (Full-Width Notification Mode)**
```text
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│ ──────────────────────────────── (Đường kẻ mảnh Separator) ──────────────────────────────────── │
│             [ 🔒 Đã ẩn thông tin tài khoản (bảo vệ quyền riêng tư người dùng) ]                  │
└──────────────────────────────────────────────────────────────────────────────────────────────────┘
  ▲ (Text bên trái và Version bên phải TẠM ẨN. Dòng thông báo CHIẾM TRỌN 100% CHIỀU NGANG thanh đáy)
```
- Khi thông báo hết thời gian hiển thị (khoảng 3–4 giây), thông báo tự tắt và thanh đáy tự động trả lại **Trạng thái 1** (hiện lại text bên trái và Version bên phải).

---

# PHẦN II: LỘ TRÌNH THI CÔNG CHI TIẾT THEO 4 GIAI ĐOẠN (PHASED ROADMAP)

Theo chỉ đạo trực tiếp của Human Owner, thứ tự thi công bắt buộc tuân theo:
- **Phase 1:** Dời `serverStatusLabel` xuống thanh đáy, thêm tiền tố `Antigravity:`, giải phóng Header Row 2.
- **Phase 2:** Xây dựng hệ thống Versioning (Single Source of Truth, Build `-ldflags`, đồng bộ Windows PE Resource `app.rc`) và hiển thị `Version: xxxxx` ở mép phải thanh đáy.
- **Phase 3:** Xây dựng cơ chế ẩn text trái/phải khi có thông báo chạy (Full-Width Notification takeover).
- **Phase 4:** Bộ Kiểm thử Chấp nhận Toàn diện (TC-01 đến TC-06) & Verification Gate.

---

## GIAI ĐOẠN 1 (PHASE 1): DỜI SERVER STATUS XUỐNG ĐÁY & TINH GỌN HEADER ROW 2

### 1. Mục tiêu
- Chuyển `a.serverStatusLabel` từ Header Row 2 xuống mép trái của vùng đáy cửa sổ.
- Bổ sung tiền tố định danh rõ ràng: **`Antigravity:`**.
- Tinh gọn cụm nút bấm bên trái Header Row 2.
- Bổ sung đường kẻ mảnh `widget.NewSeparator()` phía trên thanh đáy.

### 2. Các tệp tin ảnh hưởng
- [`internal/ui/window.go`](file:///g:/antigravity_auto_switch/internal/ui/window.go)
- [`internal/ui/app.go`](file:///g:/antigravity_auto_switch/internal/ui/app.go)
- [`internal/ui/ui_test.go`](file:///g:/antigravity_auto_switch/internal/ui/ui_test.go)

### 3. Quy cách code chi tiết

#### A. Cập nhật Định dạng Text trong [`internal/ui/app.go:1293-1299`](file:///g:/antigravity_auto_switch/internal/ui/app.go#L1293-L1299)
Cập nhật tiền tố `Antigravity:` cho toàn bộ các trạng thái:
```go
if a.serverStatusLabel != nil {
	if overview.LanguageServerRunning && overview.ServerInfo != nil {
		a.serverStatusLabel.SetText(fmt.Sprintf("Antigravity: Hoạt động  | PID %d | Port %d", overview.ServerInfo.PID, overview.ServerInfo.Port))
	} else {
		a.serverStatusLabel.SetText("Antigravity: Không hoạt động  | Dự phòng WinCred")
	}
}
```

Trong hàm khởi tạo ban đầu tại [`internal/ui/window.go:121`](file:///g:/antigravity_auto_switch/internal/ui/window.go#L121):
```go
a.serverStatusLabel = widget.NewLabelWithStyle("Antigravity: Hoạt động  | Đang kết nối...", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
```

#### B. Tinh gọn Header Row 2 trong [`internal/ui/window.go:145-156`](file:///g:/antigravity_auto_switch/internal/ui/window.go#L145-L156)
Gỡ bỏ `a.serverStatusLabel` ra khỏi `row2LeftCluster`:
```go
toolbarLeft := container.NewHBox(
	compactButton(a.addAccountBtn),
	compactButton(a.refreshBtn),
	compactButton(a.quotaRefreshAllBtn),
	compactButton(a.alwaysOnTopBtn),
)

// row2LeftCluster chỉ còn chứa toolbarLeft, đặt ngay đầu góc trái
row2LeftCluster := toolbarLeft
```

#### C. Tái cấu trúc Vùng Đáy trong [`internal/ui/window.go:240-256`](file:///g:/antigravity_auto_switch/internal/ui/window.go#L240-L256)
```go
// Status Bar: transient feedback for loading/success/error
a.statusBar = widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})
a.statusBar.Truncation = fyne.TextTruncateEllipsis
a.statusBar.Hide()

// Khung chứa nội dung thanh đáy: Left (Server Status), Center (Status Bar)
footerContent := container.NewBorder(
	nil, nil,
	a.serverStatusLabel, // Mép trái thanh đáy
	nil,                 // Mép phải (sẽ gắn versionLabel ở Phase 2)
	a.statusBar,         // Trung tâm
)

bottomSection := container.NewVBox(
	widget.NewSeparator(),
	container.NewPadded(footerContent),
)

// Main Layout
mainLayout := container.NewBorder(
	topSection,
	bottomSection,
	nil,
	nil,
	scrollList,
)
```

#### D. Đồng bộ hóa Kiểm thử Hiện hữu (`internal/ui/ui_test.go`)
Trong các bài test cũ có kiểm tra chuỗi `app.serverStatusLabel.Text`:
- Cập nhật assertion từ `"Không hoạt động"` thành `"Antigravity: Không hoạt động"`.
- Cập nhật assertion từ `"Hoạt động  | PID 7768 | Port 52704"` thành `"Antigravity: Hoạt động  | PID 7768 | Port 52704"`.

---

## GIAI ĐOẠN 2 (PHASE 2): HỆ THỐNG VERSIONING & HIỂN THỊ PHIÊN BẢN MÉP PHẢI

### 1. Mục tiêu
- Xây dựng Single Source of Truth cho Version tại `internal/version/version.go` (giá trị chuẩn: `"1.0.2.0"`).
- Nâng cấp `scripts/build_windows.ps1` inject version và git short commit hash qua `-ldflags`.
- Tự động đồng bộ khối tài nguyên `VERSIONINFO` trong `cmd/switcher/app.rc` để file `.exe` trên Windows hiển thị đúng File Version & Product Version là `1.0.2.0`.
- Bổ sung `a.versionLabel` vào mép phải thanh đáy (`fyne.TextAlignTrailing`).

### 2. Các tệp tin tạo mới và cập nhật
- **Tạo mới:** [`internal/version/version.go`](file:///g:/antigravity_auto_switch/internal/version/version.go)
- **Tạo mới:** [`internal/version/version_test.go`](file:///g:/antigravity_auto_switch/internal/version/version_test.go)
- **Cập nhật:** [`cmd/switcher/app.rc`](file:///g:/antigravity_auto_switch/cmd/switcher/app.rc)
- **Cập nhật:** [`scripts/build_windows.ps1`](file:///g:/antigravity_auto_switch/scripts/build_windows.ps1)
- **Cập nhật:** [`internal/ui/app.go`](file:///g:/antigravity_auto_switch/internal/ui/app.go)
- **Cập nhật:** [`internal/ui/window.go`](file:///g:/antigravity_auto_switch/internal/ui/window.go)

### 3. Quy cách code chi tiết

#### A. Mã nguồn `internal/version/version.go`
```go
package version

import "fmt"

var (
	// Version là phiên bản chính thức của ứng dụng (Single Source of Truth).
	// Giá trị fallback khi chạy dev nội bộ hoặc "go test ./...".
	Version = "1.0.2.0"

	// GitCommit là short commit hash của Git (ví dụ: "a77efb9"), được inject qua -ldflags lúc build.
	GitCommit = ""
)

// GetVersion trả về chuỗi phiên bản thuần túy (ví dụ: "1.0.2.0").
func GetVersion() string {
	return Version
}

// GetFullVersion trả về chuỗi phiên bản định dạng đầy đủ hiển thị trên UI.
// Nếu có GitCommit: "Version: 1.0.2.0 (rev a77efb9)"
// Nếu không có GitCommit: "Version: 1.0.2.0"
func GetFullVersion() string {
	if GitCommit != "" {
		return fmt.Sprintf("Version: %s (rev %s)", Version, GitCommit)
	}
	return fmt.Sprintf("Version: %s", Version)
}
```

#### B. Mã nguồn `internal/version/version_test.go`
```go
package version

import (
	"strings"
	"testing"
)

func TestVersion_Default(t *testing.T) {
	if GetVersion() != "1.0.2.0" {
		t.Errorf("expected default Version '1.0.2.0', got: %s", GetVersion())
	}
	full := GetFullVersion()
	if !strings.HasPrefix(full, "Version: 1.0.2.0") {
		t.Errorf("expected full version to start with 'Version: 1.0.2.0', got: %s", full)
	}
}

func TestVersion_WithGitCommit(t *testing.T) {
	oldCommit := GitCommit
	defer func() { GitCommit = oldCommit }()

	GitCommit = "a77efb9"
	expected := "Version: 1.0.2.0 (rev a77efb9)"
	if GetFullVersion() != expected {
		t.Errorf("expected '%s', got '%s'", expected, GetFullVersion())
	}
}
```

#### C. Cập nhật Windows PE Resource `cmd/switcher/app.rc`
Thêm cấu trúc `VERSIONINFO` để nhúng vào PE Header của Windows:
```rc
// Resource script for Antigravity Multi-Account Switcher
// Embeds Windows application icon and Version metadata into compiled PE header

1 ICON "logo/app.ico"

1 VERSIONINFO
FILEVERSION 1,0,2,0
PRODUCTVERSION 1,0,2,0
FILEFLAGSMASK 0x3fL
FILEFLAGS 0x0L
FILEOS 0x40004L
FILETYPE 0x1L
FILESUBTYPE 0x0L
BEGIN
    BLOCK "StringFileInfo"
    BEGIN
        BLOCK "040904b0"
        BEGIN
            VALUE "CompanyName", "Antigravity Open Source"
            VALUE "FileDescription", "Antigravity Multi-Account Switcher"
            VALUE "FileVersion", "1.0.2.0"
            VALUE "InternalName", "antigravity-switch"
            VALUE "LegalCopyright", "Copyright (C) 2026"
            VALUE "OriginalFilename", "antigravity-switch.exe"
            VALUE "ProductName", "Antigravity Multi-Account Switcher"
            VALUE "ProductVersion", "1.0.2.0"
        END
    END
    BLOCK "VarFileInfo"
    BEGIN
        VALUE "Translation", 0x409, 1200
    END
END
```

#### D. Nâng cấp `scripts/build_windows.ps1`
1. Đọc short commit hash từ Git:
```powershell
$AppVersion = "1.0.2.0"
$GitCommit = ""
try {
    $GitCommit = (git rev-parse --short HEAD 2>$null).Trim()
} catch {
    $GitCommit = ""
}
```
2. Inject vào `Build-GUI`:
```powershell
$ldflags = "-H windowsgui -s -w -X 'antigravity-switch/internal/version.Version=$AppVersion' -X 'antigravity-switch/internal/version.GitCommit=$GitCommit'"
go build -ldflags="$ldflags" -o $outFile .\cmd\switcher
```

#### E. Tích hợp `versionLabel` vào Giao diện UI
1. Trong [`internal/ui/app.go`](file:///g:/antigravity_auto_switch/internal/ui/app.go), thêm trường vào struct `App`:
```go
type App struct {
    ...
    versionLabel *widget.Label
    ...
}
```
2. Trong [`internal/ui/window.go`](file:///g:/antigravity_auto_switch/internal/ui/window.go):
```go
a.versionLabel = widget.NewLabelWithStyle(version.GetFullVersion(), fyne.TextAlignTrailing, fyne.TextStyle{})

footerContent := container.NewBorder(
	nil, nil,
	a.serverStatusLabel, // Mép trái
	a.versionLabel,      // Mép phải
	a.statusBar,         // Trung tâm
)
```

---

## GIAI ĐOẠN 3 (PHASE 3): CƠ CHẾ ẨN/HIỆN THÔNG BÁO TOÀN CHIỀU NGANG (FULL-WIDTH TAKEOVER)

### 1. Mục tiêu
- Khi ứng dụng kích hoạt một thông báo (`showStatusBar(msg)`):
  - Lập tức ẩn text bên trái (`a.serverStatusLabel.Hide()`).
  - Lập tức ẩn text bên phải (`a.versionLabel.Hide()`).
  - `a.statusBar` chiếm trọn 100% bề ngang thanh đáy, chữ căn giữa (`fyne.TextAlignCenter`), nổi bật và không bị chèn ép.
- Khi thông báo kết thúc (`clearStatusBar()`):
  - Ẩn dòng thông báo (`a.statusBar.Hide()`).
  - Tự động hiển thị lại cả hai nhãn: `a.serverStatusLabel.Show()` và `a.versionLabel.Show()`.

### 2. Các tệp tin ảnh hưởng
- [`internal/ui/app.go`](file:///g:/antigravity_auto_switch/internal/ui/app.go)
- [`internal/ui/window.go`](file:///g:/antigravity_auto_switch/internal/ui/window.go)

### 3. Quy cách code chi tiết

Trong [`internal/ui/app.go:1353-1385`](file:///g:/antigravity_auto_switch/internal/ui/app.go#L1353-L1385):

```go
// showStatusBar displays a transient status message taking over the entire bottom bar.
func (a *App) showStatusBar(msg string) {
	if a.statusBar == nil {
		return
	}
	if a.statusBarHideTimer != nil {
		a.statusBarHideTimer.Stop()
		a.statusBarHideTimer = nil
	}

	// 1. Tạm ẩn text hai bên mép để nhường 100% không gian cho thông báo
	if a.serverStatusLabel != nil {
		a.serverStatusLabel.Hide()
	}
	if a.versionLabel != nil {
		a.versionLabel.Hide()
	}

	// 2. Cài đặt nội dung và hiển thị thông báo
	a.statusBar.SetText(msg)
	a.statusBar.Show()
}

// clearStatusBar hides the status bar and restores normal left/right footer labels.
func (a *App) clearStatusBar() {
	if a.statusBar == nil {
		return
	}
	a.statusBar.SetText("")
	a.statusBar.Hide()

	// Khôi phục lại trạng thái bình thường cho thanh đáy
	if a.serverStatusLabel != nil {
		a.serverStatusLabel.Show()
	}
	if a.versionLabel != nil {
		a.versionLabel.Show()
	}
}
```

Trong hàm `autoHideStatusBar(d time.Duration)`:
- Thời gian hiển thị chuẩn: `3 * time.Second` đến `4 * time.Second`.
- Khi hết thời gian, hàm gọi `fyne.Do(func() { a.clearStatusBar() })`, tự động trả lại `a.serverStatusLabel` và `a.versionLabel` trên luồng UI chính mà không gây giật lag.

---

## GIAI ĐOẠN 4 (PHASE 4): BỘ KIỂM THỬ CHẤP NHẬN TOÀN DIỆN (TC-01 ĐẾN TC-06)

### 1. Mục tiêu
Thiết lập bộ test case độc lập trong `internal/ui/footer_bar_acceptance_test.go` và `internal/version/version_test.go` kiểm chứng nghiêm ngặt từng tiêu chí nghiệm thu.

### 2. Danh mục Kịch bản Kiểm thử Chấp nhận

#### **TC-01: Header Row 2 Tinh Gọn (Clean Header)**
- **Mô tả:** Kiểm tra bố cục thanh công cụ dòng 2.
- **Tiêu chí PASS:**
  - `row2LeftCluster` chỉ chứa cụm nút bấm, phần tử đầu tiên chính là nút `Thêm` (`a.addAccountBtn`).
  - Không còn chứa `a.serverStatusLabel` trong bất kỳ container nào của Top Section.

#### **TC-02: Thanh Đáy Chuẩn Mực & Tiền Tố Antigravity (Footer Placement & Prefix)**
- **Mô tả:** Kiểm tra vị trí và tiền tố của `serverStatusLabel`.
- **Tiêu chí PASS:**
  - `a.serverStatusLabel` nằm ở mép trái của `footerContent`.
  - Khi chưa kết nối: chuỗi hiển thị bắt đầu bằng `"Antigravity: Hoạt động  | Đang kết nối..."`.
  - Khi kết nối thành công: chuỗi hiển thị có định dạng `"Antigravity: Hoạt động  | PID %d | Port %d"`.
  - Khi không có IDE: chuỗi hiển thị là `"Antigravity: Không hoạt động  | Dự phòng WinCred"`.

#### **TC-03: Nhãn Phiên Bản Mép Phải (Version Display & Formatting)**
- **Mô tả:** Kiểm tra nhãn hiển thị phiên bản ở mép phải.
- **Tiêu chí PASS:**
  - `a.versionLabel` nằm ở mép phải của `footerContent`, căn lề phải (`TextAlignTrailing`).
  - Chuỗi hiển thị bắt đầu bằng `"Version: 1.0.2.0"`.

#### **TC-04: Cơ Chế Ẩn/Hiện Chiếm Trọn Chiều Ngang (Full-Width Notification Takeover)**
- **Mô tả:** Mô phỏng chu trình phát thông báo và hoàn tất thông báo.
- **Tiêu chí PASS:**
  - Ban đầu (Idle): `a.serverStatusLabel.Visible() == true`, `a.versionLabel.Visible() == true`, `a.statusBar.Visible() == false`.
  - Khi gọi `a.showStatusBar("🔒 Đã ẩn thông tin tài khoản")`:
    * `a.serverStatusLabel.Visible() == false`.
    * `a.versionLabel.Visible() == false`.
    * `a.statusBar.Visible() == true`.
    * `a.statusBar.Text == "🔒 Đã ẩn thông tin tài khoản"`.
  - Khi gọi `a.clearStatusBar()`:
    * `a.statusBar.Visible() == false`.
    * `a.serverStatusLabel.Visible() == true`.
    * `a.versionLabel.Visible() == true`.

#### **TC-05: Single Source of Truth & Package Version**
- **Mô tả:** Kiểm tra package `internal/version`.
- **Tiêu chí PASS:**
  - `version.GetVersion()` trả về `"1.0.2.0"`.
  - `version.GetFullVersion()` định dạng chính xác khi có hoặc không có `GitCommit`.

#### **TC-06: Xác Thực Toàn Diện Bản Dựng Windows (Build & Verification Gate)**
- **Mô tả:** Chạy kịch bản biên dịch đầy đủ của repository.
- **Tiêu chí PASS:**
  - Lệnh `pwsh -File .\scripts\build_windows.ps1 -Clean` thực thi thành công, Exit Code = 0.
  - File PE nhị phân `bin\antigravity-switch.exe` được tạo thành công, chứa icon PE hợp lệ và khối `VERSIONINFO` phản ánh `1.0.2.0`.
  - Toàn bộ suite test `go test ./...` đạt PASS 100% (bao gồm cả 8 bài test TC-01 đến TC-08 của tính năng Add Modal).

---

# PHẦN III: QUY TẮC BẤT BIẾN KIẾN TRÚC (INVARIANTS & BOUNDARIES)

1. **Zero Breaking Changes on Existing API:**
   - Các trường `serverStatusLabel`, `statusBar` giữ nguyên 100% tên trường và kiểu dữ liệu trên struct `App`.
   - Các luồng nghiệp vụ gọi `a.showStatusBar()` và `a.clearStatusBar()` trên toàn bộ codebase tiếp tục hoạt động trong suốt mà không cần sửa đổi call-site.
2. **Build Isolation & Zero Dependency Creep:**
   - Package `internal/version` là package độc lập ở tầng core, tuyệt đối không phụ thuộc ngược vào `internal/ui` hay các package giao diện (tránh import cycle).
3. **Responsive Stability:**
   - Việc ẩn `serverStatusLabel` và `versionLabel` khi thông báo xuất hiện không được làm giật hoặc sập chiều cao của cửa sổ. `container.NewBorder` duy trì chiều cao tự nhiên theo chiều cao của font chữ và padding định sẵn.

---

*Đặc tả kiến trúc hoàn chỉnh được chấp bút bởi System Architect Lane — Sẵn sàng chuyển giao cho Strategic Planner lập Execution Plan và CEO điều phối Coder triển khai.*
