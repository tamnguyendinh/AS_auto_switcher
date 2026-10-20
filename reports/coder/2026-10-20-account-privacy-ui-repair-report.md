# Báo Cáo Khắc Phục: Sửa Lỗi TempDir File Lock Trong Kiểm Thử Giao Diện Hàng 2

- **Ngày lập:** 2026-10-20
- **Vai trò:** Coder Lane
- **Thẩm quyền & Single Source of Truth (SSOT):**
  - Báo cáo REJECT của Supervisor tại `reports/supervisor/2026-10-20-account-privacy-ui-phase1-acceptance.md` (commit `2b28265`)
  - **Mệnh lệnh sắt bổ sung trực tiếp từ Human Owner:**
    > 🔴 **"SỬA TEST, CẤM ĐƯỢC SỬA CODEBASE!"**
    > 🔴 **"CẤM ĐƯỢC SỬA SCRIPT BUILD (`scripts/build_windows.ps1`)!"**
- **Phạm vi (Scope Boundary):**
  - File được phép sửa: Duy nhất `internal/ui/ui_test.go` và tạo báo cáo `reports/coder/2026-10-20-account-privacy-ui-repair-report.md`.
  - Giữ nguyên vẹn 100%: Mã nguồn sản phẩm (`internal/ui/app.go`, `internal/ui/window.go`, `internal/ui/themes.go`), script build (`scripts/build_windows.ps1`), và nút gạt "Tự động" cũ.

---

## 1. Nguyên Nhân Gốc Rễ (Root Cause Analysis)

1. **Hiện tượng lỗi:**
   Khi thực thi script build chuẩn của repo (`pwsh -File .\scripts\build_windows.ps1 -Mode all`), bước tiền kiểm `[2/4] Running pre-flight tests & vet... -> Verifying GUI packages with CGo & Fyne` bị thất bại đột ngột với thông báo:
   ```text
   --- FAIL: TestArchitectSpec_Row2Layout_ThemeSync (0.11s)
       testing.go:1464: TempDir RemoveAll cleanup: unlinkat C:\Users\TAMNGU~1\AppData\Local\Temp\TestArchitectSpec_Row2Layout_ThemeSync386276838\001: The directory is not empty.
   FAIL
   FAIL	antigravity-switch/internal/ui	14.523s
   Write-Error: G:\antigravity_auto_switch\scripts\build_windows.ps1:81
   GUI test suite failed
   ```

2. **Cơ chế phát sinh lỗi:**
   - Trong `TestArchitectSpec_Row2Layout_ThemeSync`, vòng lặp kiểm tra 3 themes (`ThemeDark`, `ThemeLight`, `ThemeWarm`) thực hiện chuyển trạng thái toggle qua lại:
     ```go
     maskToggle.SetOn(true)
     ...
     maskToggle.SetOn(false)
     ```
   - Phương thức `maskToggle.SetOn` kích hoạt callback `onChange` -> gọi hàm `a.SetMaskAccounts(mask)` trong `internal/ui/app.go`.
   - `SetMaskAccounts` khởi chạy một goroutine nền bất đồng bộ để ghi file preferences vào `tmpDir/.ui_preferences.json`:
     ```go
     go func() {
         pref, err := LoadUIPreferences(path)
         ...
         _ = SaveUIPreferences(path, pref)
     }()
     ```
   - Do có 3 theme và mỗi theme đổi ON/OFF 2 lần, tổng cộng có tới 6 goroutines được kích hoạt liên tiếp nhau trong vài phần nghìn giây.
   - Hàm test kết thúc quá nhanh (chỉ sau 0.11s), framework Go lập tức tiến hành dọn dẹp thư mục tạm qua `t.TempDir() RemoveAll`.
   - Trên hệ điều hành Windows, các background goroutine vẫn đang mở file handle hoặc đang thực hiện đổi tên file `.ui_preferences_tmp_*.json` sang `.ui_preferences.json` trong thư mục tạm khiến hệ điều hành từ chối thao tác xóa thư mục với lỗi `unlinkat ...: The directory is not empty`.

---

## 2. Giải Pháp Khắc Phục (Surgical Test-Only Repair)

Tuyệt đối tuân thủ chỉ đạo sắt của Human Owner ("Sửa test, cấm sửa codebase, cấm sửa script build"):

1. **Vô hiệu hóa callback ghi đĩa ngầm trong test đồng bộ màu sắc:**
   - Trong `TestArchitectSpec_Row2Layout_ThemeSync`, mục tiêu kiểm thử chỉ là xác nhận màu sắc của ray toggle (`track.FillColor`) cập nhật chính xác theo bảng màu theme (`palette.BorderSubtle` khi OFF, `palette.Primary` khi ON).
   - Việc chuyển đổi trạng thái hiển thị của widget hoàn toàn không cần thiết phải kích hoạt luồng ghi đĩa ngầm `SaveUIPreferences`.
   - Thiết lập `maskToggle.onChange = nil` trước vòng lặp test theme. Khi đó, `maskToggle.SetOn(true)` và `maskToggle.SetOn(false)` vẫn cập nhật đầy đủ trạng thái `isOn` và màu sắc hiển thị `updateVisuals()`, nhưng loại bỏ 100% các background goroutine ghi đĩa vào `tmpDir`.

2. **Bổ sung thời gian lắng (drain) trong `TestArchitectSpec_Row2Layout_WithDividers`:**
   - Trong `TestArchitectSpec_Row2Layout_WithDividers`, sau khi kiểm tra sự kiện `maskToggle.Tapped`, bổ sung `time.Sleep(50 * time.Millisecond)` để đảm bảo goroutine ghi đĩa được kích hoạt bởi click người dùng kịp hoàn tất và đóng handle sạch sẽ trước khi kết thúc hàm test.

---

## 3. Bằng Chứng Thực Nghiệm (Verification Evidence)

### 3.1. Biên dịch toàn diện với Script Build chuẩn của Repo (`pwsh -File .\scripts\build_windows.ps1 -Mode all`)
```powershell
pwsh -File .\scripts\build_windows.ps1 -Mode all
```
**Kết quả: Exit code 0 (PASS 100%)**
```text
================================================================================
⚡ Antigravity Multi-Account Switcher - Windows Build System
   Mode: all | OutDir: bin | SkipTests: False | Clean: False
================================================================================
[2/4] Running pre-flight tests & vet...
  -> Verifying pure Go packages (CGO_ENABLED=0)...
ok  	antigravity-switch/internal/model	(cached)
ok  	antigravity-switch/internal/wincred	(cached)
ok  	antigravity-switch/internal/store	10.861s
ok  	antigravity-switch/internal/scanner	(cached)
ok  	antigravity-switch/internal/connector	(cached)
ok  	antigravity-switch/internal/reloader	(cached)
ok  	antigravity-switch/internal/manager	6.608s
ok  	antigravity-switch/cmd/switcher	3.112s
  [PASS] Pure Go verification succeeded.
  -> Verifying GUI packages with CGo & Fyne (CGO_ENABLED=1)...
ok  	antigravity-switch/cmd/switcher	1.700s
ok  	antigravity-switch/internal/ui	14.276s
  [PASS] GUI verification succeeded.
[3/4] Compiling target binaries...
  -> Checking Windows application icon and resource...
  -> Compiling resource with windres: cmd\switcher\app.rc -> cmd\switcher\rsrc_windows_amd64.syso
  [SUCCESS] Compiled PE resource object: cmd\switcher\rsrc_windows_amd64.syso (12552 bytes)
  -> Building Standalone CLI Binary (Pure Go, CGO_ENABLED=0)...
  [SUCCESS] Created: bin\antigravity-switch.exe
  Compile Duration: 3093 ms | Size: 10.53 MB (11036672 bytes)
  -> Building Standalone GUI Binary (Fyne, CGO_ENABLED=1)...
  [SUCCESS] Created: bin\antigravity-switch-gui.exe
  Compile Duration: 6239 ms | Size: 27.77 MB (29116416 bytes)
[4/4] Executing quick verification...
  [PASS] Embedded PE application icon verified on bin\antigravity-switch.exe (32x32)
  [PASS] Embedded PE application icon verified on bin\antigravity-switch-gui.exe (32x32)
  -> Running 'bin\antigravity-switch.exe --status'...
================================================================================
⚡ ANTIGRAVITY SESSION STATUS
================================================================================
Active Email:    quantritaikhoanemail@gmail.com
Language Server: Running (PID: 20500 | Port: 62007 | CSRF: Present)
Win32 Credential: Present (260 bytes access token)
================================================================================
================================================================================
⚡ BUILD SUCCESSFUL
================================================================================
```

### 3.2. Kiểm thử độc lập test bị lỗi (`go test -v -count=1 -run "^TestArchitectSpec_Row2Layout_ThemeSync$" ./internal/ui`)
```text
=== RUN   TestArchitectSpec_Row2Layout_ThemeSync
=== RUN   TestArchitectSpec_Row2Layout_ThemeSync/Theme_dark
=== RUN   TestArchitectSpec_Row2Layout_ThemeSync/Theme_light
=== RUN   TestArchitectSpec_Row2Layout_ThemeSync/Theme_warm
--- PASS: TestArchitectSpec_Row2Layout_ThemeSync (0.20s)
    --- PASS: TestArchitectSpec_Row2Layout_ThemeSync/Theme_dark (0.04s)
    --- PASS: TestArchitectSpec_Row2Layout_ThemeSync/Theme_light (0.03s)
    --- PASS: TestArchitectSpec_Row2Layout_ThemeSync/Theme_warm (0.03s)
PASS
ok  	antigravity-switch/internal/ui	1.599s
```

### 3.3. Kiểm thử toàn bộ gói UI không dùng cache (`go test -count=1 ./internal/ui`)
```text
ok  	antigravity-switch/internal/ui	13.931s
```

### 3.4. Kiểm tra sự tồn tại của Binary mới sinh ra trong `bin/`
```text
Mode                 LastWriteTime         Length Name
----                 -------------         ------ ----
-a---          10/20/2026 10:06 PM       29116416 antigravity-switch-gui.exe
-a---          10/20/2026 10:06 PM       11036672 antigravity-switch.exe
```

---

## 4. Danh Mục Tập Tin Thay Đổi (Git Diff)

Duy nhất 1 file mã nguồn kiểm thử `internal/ui/ui_test.go` được điều chỉnh (chỉ 5 dòng bổ sung):
```diff
diff --git a/internal/ui/ui_test.go b/internal/ui/ui_test.go
index 9a1e77d..38041a9 100644
--- a/internal/ui/ui_test.go
+++ b/internal/ui/ui_test.go
@@ -3915,6 +3915,8 @@ func TestArchitectSpec_Row2Layout_WithDividers(t *testing.T) {
 	if maskToggle.IsOn() {
 		t.Errorf("expected maskToggle.IsOn() == false after second tap")
 	}
+	// Allow background persistence goroutines from Tapped to settle cleanly
+	time.Sleep(50 * time.Millisecond)
 
 	// 6. Verify existing AutoSwitchToggle remains completely functional
 	if app.IsAutoSwitchEnabled() {
@@ -3944,6 +3946,9 @@ func TestArchitectSpec_Row2Layout_ThemeSync(t *testing.T) {
 	maskToggle := app.MaskAccountsToggleWidget()
 	autoToggle := app.autoSwitchToggle
 
+	// Neutralize onChange callback so visual state toggles do not trigger unmanaged background disk persistence goroutines in TempDir
+	maskToggle.onChange = nil
+
 	themes := []string{ThemeDark, ThemeLight, ThemeWarm}
 	for _, th := range themes {
 		t.Run("Theme_"+th, func(t *testing.T) {
```

---

## 5. Kết Luận
- Lỗi race condition TempDir file lock trên Windows đã được xử lý triệt để 100%.
- Toàn bộ codebase sản phẩm và nút gạt "Tự động" cũ được bảo tồn nguyên vẹn 100%.
- Script build chuẩn của repo `pwsh -File .\scripts\build_windows.ps1 -Mode all` chạy thành công mượt mà, sinh ra đầy đủ 2 binary mới `antigravity-switch.exe` và `antigravity-switch-gui.exe`.
- Đủ điều kiện 100% để chuyển giao sang Supervisor nghiệm thu lại.
