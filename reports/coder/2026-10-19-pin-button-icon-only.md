# Báo Cáo Thi Công Coder: Tinh Giản Nút Ghim Cửa Sổ Thành Icon-Only [ 📌 ]

- **Ngày thi công:** 2026-10-19
- **Vai trò:** Coder
- **Thẩm quyền:** Chỉ đạo trực tiếp từ Human Owner qua CEO Agent
- **Tài liệu tham chiếu / SSOT:**
  - `reports/architect/2026-10-19-pin-button-icon-only-spec.md` (Commit 372c0f7)
  - `SPEC-ARCH-20261019-PIN-BUTTON-ICON-ONLY`
  - `AGENTS.md` & `working-rules`
- **Mục tiêu hoàn thành:**
  1. Tinh giản nút ghim cửa sổ (Always-on-Top) trên Toolbar Row 3 thành icon duy nhất `[ 📌 ]`, xóa bỏ hoàn toàn chuỗi text tiếng Việt ` Ghim : Bật` / ` Ghim : Tắt`.
  2. Bảo toàn nguyên vẹn cơ chế đổi màu `Importance`: `widget.HighImportance` khi đang bật (sáng màu primary) và `widget.MediumImportance` khi đang tắt (chìm/mờ).
  3. Cập nhật bộ test tự động trong `internal/ui/ui_test.go`: assert nhãn nút luôn là `"📌"`, kiểm tra chặt chẽ `Importance` ở cả trạng thái Bật và Tắt.
  4. Thi công chính xác 100% theo các đoạn code mẫu trong tài liệu đặc tả của Kiến trúc sư, TUYỆT ĐỐI CẤM SUY DIỄN.

---

## 1. Invariant Family Map

| Invariant | Tệp thực hiện | Chi tiết thi công kỹ thuật |
|---|---|---|
| **INV-01 (Fixed Icon-Only Label at Init)** | `internal/ui/window.go` | Trong `setupLayout()`: Xóa bỏ rẽ nhánh gán nhãn `pinLabel` (`"📌 Ghim : Bật"` / `"📌 Ghim : Tắt"`). Khởi tạo `alwaysOnTopBtn` cố định nhãn text là `"📌"`. Gán `pinImportance` theo trạng thái `a.alwaysOnTop` (`HighImportance` khi bật, `MediumImportance` khi tắt). |
| **INV-02 (Parity Sync & Importance Toggle)** | `internal/ui/app.go` | Trong `syncAlwaysOnTopBtn(enabled bool)`: Luôn gọi `a.alwaysOnTopBtn.SetText("📌")` và chỉ cập nhật `a.alwaysOnTopBtn.Importance` theo trạng thái `enabled` (`widget.HighImportance` khi bật, `widget.MediumImportance` khi tắt), sau đó gọi `Refresh()`. Bảo toàn 100% chức năng OS `setWindowAlwaysOnTop`. |
| **INV-03 (Automated Verification & Visual State Assertions)** | `internal/ui/ui_test.go` | Đổi tên/cập nhật `TestAlwaysOnTop` (và alias `TestToggleAlwaysOnTop`): kiểm tra nhãn nút luôn là `"📌"`, kiểm tra `Importance == widget.MediumImportance` sau khi toggle tắt, và `Importance == widget.HighImportance` sau khi set bật lại. |

---

## 2. Touch-Map Compliance Verification

Tuân thủ nghiêm ngặt ranh giới Touch-Map khóa cứng được giao, tuyệt đối không chỉnh sửa bất kỳ file nào ngoài phạm vi:

| STT | File Path | Trạng thái | Tóm tắt kiểm tra |
|---|---|---|---|
| 1 | `internal/ui/window.go` | Modified | Sửa khối khởi tạo Toolbar Row 3, dùng nhãn cố định `"📌"`, loại bỏ biến `pinLabel`. |
| 2 | `internal/ui/app.go` | Modified | Trong `syncAlwaysOnTopBtn`: luôn đặt `SetText("📌")`, chỉ toggle `Importance`. |
| 3 | `internal/ui/ui_test.go` | Modified | Cập nhật `TestAlwaysOnTop` và `TestToggleAlwaysOnTop` assert `"📌"` và kiểm tra `Importance`. |
| 4 | `reports/coder/2026-10-19-pin-button-icon-only.md` | Created | Báo cáo thi công chính thức lưu vết kỹ thuật. |

---

## 3. Chi Tiết Thay Đổi Code (Diff Walkthrough)

### 3.1. `internal/ui/window.go`
```go
@@ -155,13 +155,11 @@ func (a *App) setupLayout() {
 	})
 	a.quotaRefreshAllBtn.Importance = widget.HighImportance
 
-	pinLabel := "📌 Ghim : Bật"
 	pinImportance := widget.HighImportance
 	if !a.alwaysOnTop {
-		pinLabel = "📌 Ghim : Tắt"
 		pinImportance = widget.MediumImportance
 	}
-	a.alwaysOnTopBtn = widget.NewButton(pinLabel, func() {
+	a.alwaysOnTopBtn = widget.NewButton("📌", func() {
 		a.ToggleAlwaysOnTop()
 	})
 	a.alwaysOnTopBtn.Importance = pinImportance
```

### 3.2. `internal/ui/app.go`
```go
@@ -609,11 +609,10 @@ func (a *App) syncAlwaysOnTopBtn(enabled bool) {
 	if a.alwaysOnTopBtn == nil {
 		return
 	}
+	a.alwaysOnTopBtn.SetText("📌")
 	if enabled {
-		a.alwaysOnTopBtn.SetText("📌 Ghim : Bật")
 		a.alwaysOnTopBtn.Importance = widget.HighImportance
 	} else {
-		a.alwaysOnTopBtn.SetText("📌 Ghim : Tắt")
 		a.alwaysOnTopBtn.Importance = widget.MediumImportance
 	}
 	a.alwaysOnTopBtn.Refresh()
```

### 3.3. `internal/ui/ui_test.go`
```go
@@ -179,7 +179,7 @@ func TestNewAppAndInit(t *testing.T) {
 	waitForIdle(app, 3*time.Second)
 }
 
-func TestToggleAlwaysOnTop(t *testing.T) {
+func TestAlwaysOnTop(t *testing.T) {
 	mgr := createTestManager(t)
 	testApp := test.NewApp()
 	app := NewAppWithFyne(testApp, mgr)
@@ -194,21 +194,30 @@ func TestToggleAlwaysOnTop(t *testing.T) {
 	if app.IsAlwaysOnTop() {
 		t.Errorf("expected alwaysOnTop to be false after toggle")
 	}
-	if app.alwaysOnTopBtn.Text != "📌 Ghim : Tắt" {
-		t.Errorf("expected button text '📌 Ghim : Tắt', got: %s", app.alwaysOnTopBtn.Text)
+	if app.alwaysOnTopBtn.Text != "📌" {
+		t.Errorf("expected button text '📌', got: %s", app.alwaysOnTopBtn.Text)
+	}
+	if app.alwaysOnTopBtn.Importance != widget.MediumImportance {
+		t.Errorf("expected button importance MediumImportance, got: %v", app.alwaysOnTopBtn.Importance)
 	}
 
 	app.SetAlwaysOnTop(true)
 	if !app.IsAlwaysOnTop() {
 		t.Errorf("expected alwaysOnTop to be true after SetAlwaysOnTop(true)")
 	}
-	if app.alwaysOnTopBtn.Text != "📌 Ghim : Bật" {
-		t.Errorf("expected button text '📌 Ghim : Bật', got: %s", app.alwaysOnTopBtn.Text)
+	if app.alwaysOnTopBtn.Text != "📌" {
+		t.Errorf("expected button text '📌', got: %s", app.alwaysOnTopBtn.Text)
+	}
+	if app.alwaysOnTopBtn.Importance != widget.HighImportance {
+		t.Errorf("expected button importance HighImportance, got: %v", app.alwaysOnTopBtn.Importance)
 	}
 
 	waitForIdle(app, 3*time.Second)
 }
 
+func TestToggleAlwaysOnTop(t *testing.T) {
+	TestAlwaysOnTop(t)
+}
```

---

## 4. Bằng Chứng Nghiệm Thu (Evidence & Test Results)

### 4.1. Unit Test Độc Lập
- Lệnh chạy: `go test -v -run TestAlwaysOnTop ./internal/ui`
```
=== RUN   TestAlwaysOnTop
--- PASS: TestAlwaysOnTop (0.46s)
PASS
ok  	antigravity-switch/internal/ui	1.777s
```

- Lệnh chạy kiểm thử alias: `go test -v -run TestToggleAlwaysOnTop ./internal/ui`
```
=== RUN   TestToggleAlwaysOnTop
--- PASS: TestToggleAlwaysOnTop (0.43s)
PASS
ok  	antigravity-switch/internal/ui	1.790s
```

### 4.2. Full Build Script Bắt Buộc Của Repo
- Lệnh thực thi: `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all`
- Kết quả: **PASS 100%, Exit Code 0**.
- Trích xuất nhật ký:
```
================================================================================
⚡ Antigravity Multi-Account Switcher - Windows Build System
   Mode: all | OutDir: bin | SkipTests: False | Clean: False
================================================================================
[2/4] Running pre-flight tests & vet...
  -> Verifying pure Go packages (CGO_ENABLED=0)...
ok  	antigravity-switch/internal/model	(cached)
ok  	antigravity-switch/internal/wincred	(cached)
ok  	antigravity-switch/internal/store	4.761s
ok  	antigravity-switch/internal/scanner	(cached)
ok  	antigravity-switch/internal/connector	(cached)
ok  	antigravity-switch/internal/reloader	(cached)
ok  	antigravity-switch/internal/manager	6.568s
ok  	antigravity-switch/cmd/switcher	(cached)
  [PASS] Pure Go verification succeeded.
  -> Verifying GUI packages with CGo & Fyne (CGO_ENABLED=1)...
ok  	antigravity-switch/cmd/switcher	1.362s
ok  	antigravity-switch/internal/ui	10.146s
  [PASS] GUI verification succeeded.
[3/4] Compiling target binaries...
  -> Checking Windows application icon and resource...
  -> Compiling resource with windres: cmd\switcher\app.rc -> cmd\switcher\rsrc_windows_amd64.syso
  [SUCCESS] Compiled PE resource object: cmd\switcher\rsrc_windows_amd64.syso (12552 bytes)
  -> Building Standalone CLI Binary (Pure Go, CGO_ENABLED=0)...
  [SUCCESS] Created: bin\antigravity-switch.exe
  Compile Duration: 2856 ms | Size: 10.51 MB (11023872 bytes)
  -> Building Standalone GUI Binary (Fyne, CGO_ENABLED=1)...
  [SUCCESS] Created: bin\antigravity-switch-gui.exe
  Compile Duration: 5705 ms | Size: 27.57 MB (28911104 bytes)
[4/4] Executing quick verification...
  [PASS] Embedded PE application icon verified on bin\antigravity-switch.exe (32x32)
  [PASS] Embedded PE application icon verified on bin\antigravity-switch-gui.exe (32x32)
  -> Running 'bin\antigravity-switch.exe --status'...
================================================================================
⚡ ANTIGRAVITY SESSION STATUS
================================================================================
Active Email:    skey.miennam@gmail.com
Language Server: Running (PID: 22216 | Port: 56078 | CSRF: Present)
Win32 Credential: Present (260 bytes access token)
================================================================================
================================================================================
⚡ BUILD SUCCESSFUL
================================================================================
```

---

## 5. Kết Luận
- Yêu cầu của Human Owner và tài liệu đặc tả của Kiến trúc sư đã được thi công chính xác 100%.
- Không có bất kỳ suy diễn nào ngoài phạm vi chỉ đạo.
- Không chỉnh sửa file nào ngoài Touch-Map khóa cứng.
- Đã sẵn sàng bàn giao cho Supervisor nghiệm thu.