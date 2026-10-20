# Báo Cáo Thi Công: Chặng 2 - Chức Năng Ẩn Tài Khoản & Che Email (Account Privacy Functionality & Masking)

- **Ngày lập:** 2026-10-20
- **Vai trò:** Coder Lane
- **Thẩm quyền & Single Source of Truth (SSOT):**
  - Chỉ thị giao việc từ Human Owner và Kiến trúc sư Hệ thống (commit `7e85f3f`)
  - Đặc tả kiến trúc: `reports/architect/2026-10-20-account-privacy-masking-toolbar-spec.md`
  - Scope: `internal/ui/widgets.go`, `internal/ui/app.go`, `internal/ui/ui_test.go`
- **Mục tiêu hoàn thành:** Hoàn thành 100% Chặng 2: Triển khai thuật toán che email `MaskEmail(email string) string`, method in-place `SetMasked(masked bool)` trên `AccountCardWidget`, nối tương tác phản ứng tức thì không jank layout vào `App State`, bảo vệ che email trên thanh trạng thái (Status Bar), tận dụng 100% hover tooltip email thật có sẵn và bảo toàn tuyệt đối nút gạt "Tự động" cũ.

---

## 1. Invariant Family Map & Phạm Vi Thực Thi

| Trường Thông Tin | Chi Tiết Xác Lập |
|---|---|
| **Tên Family** | Account Privacy Masking & Status Bar Protection (Chặng 2) |
| **Tài liệu SSOT** | `reports/architect/2026-10-20-account-privacy-masking-toolbar-spec.md` (commit `7e85f3f`) |
| **Quy Tắc KISS Masking** | Giữ lại đúng 1 ký tự đầu của username + `***` + `@` + `tên miền` (e.g. `tamnguyen@gmail.com` $\rightarrow$ `t***@gmail.com`). UTF-8 rune safe. Chuỗi không hợp lệ hoặc không có `@` giữ nguyên bản. |
| **View-Only Presentation Filter** | Chế độ ẩn tài khoản chỉ là bộ lọc hiển thị tầng View. Kiểu dữ liệu `Email string` trong Model, Account Store, Quota Cache, Chrome Scanner, Telemetry và Hot-swap Switcher được bảo toàn 100% nguyên bản. |
| **Zero Layout Jank & Rebuild** | Việc gạt toggle không recreate thẻ canvas trong `accountListContainer`. Duyệt in-place danh sách thẻ trong RAM (`card.SetMasked(isOn)`). Con trỏ widget được giữ nguyên. |
| **Zero-Code Tooltip Reuse** | Tận dụng nguyên vẹn `emailLabelWithTooltip` sẵn có trong `internal/ui/widgets.go`. Khi `SetText(MaskEmail(email))` được gọi, Fyne chỉ đổi hiển thị màn hình `l.Text`, trường `l.fullText` vẫn giữ email thật. Khi rê chuột, popup tooltip tự động bung email thật mà không cần thêm code hover nào. |
| **Signature Compatibility** | Giữ nguyên 100% chữ ký của `NewAccountCardWidgetWithQuota` và `NewAccountCardWithQuota`, không làm gãy bất kỳ unit test nào trong codebase. |
| **Status Bar Masking** | Che email trên thanh trạng thái trong cả `SwitchAccount` (`⏳ Đang chuyển sang...`, `✅ Đã chuyển sang...`) và các sự kiện dispatch từ `handleAutomationStatus`. |
| **Cấm Tuyệt Đối (Hard Rules)** | - CẤM đụng vào nút gạt "Tự động" (`AutoSwitchToggle`).<br>- CẤM sửa script build `scripts/build_windows.ps1`.<br>- KHÔNG can thiệp backend/model/telemetry.<br>- NO ANVIEN. |
| **Verify Matrix** | - `TestMaskEmail_AlgorithmConformance`: Bảng 9 ca kiểm thử thuật toán bao gồm chuỗi hợp lệ, invalid, rỗng, alias, và UTF-8 rune.<br>- `TestArchitectSpec_MaskAccountsToggle_Behavior`: Đổi toggle che/hiện in-place, giữ nguyên tooltip fullText, phản hồi status bar.<br>- `TestArchitectSpec_MaskAccounts_StatusBarMasking`: Che email trên status bar khi chuyển tài khoản và nhận event automation.<br>- `TestArchitectSpec_MaskAccounts_NewCardInitialMask`: Khởi tạo card mới với trạng thái che ngay từ đầu khi `maskAccounts = true`.<br>- `TestArchitectSpec_MaskAccounts_Persistence`: Nạp và lưu cấu hình `.ui_preferences.json`.<br>- Regression guard: Build script `pwsh -File .\scripts\build_windows.ps1 -Mode all` Exit Code 0, sinh đầy đủ CLI & GUI binary. |

---

## 2. Chi Tiết Thi Công Mã Nguồn

### 2.1. Khai Báo Thuật Toán `MaskEmail` & Trợ Giúp `maskMessageEmails` (`internal/ui/widgets.go`)
- **`MaskEmail(email string) string`**:
  ```go
  func MaskEmail(email string) string {
      trimmed := strings.TrimSpace(email)
      atIdx := strings.Index(trimmed, "@")
      if atIdx <= 0 {
          return trimmed
      }
      username := trimmed[:atIdx]
      domain := trimmed[atIdx:]
      runes := []rune(username)
      if len(runes) == 0 {
          return trimmed
      }
      return string(runes[0]) + "***" + domain
  }
  ```
- **`maskMessageEmails(msg string) string`**: Sử dụng regexp `[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}` quét và thay thế tất cả email trong thông điệp status bar bằng chuỗi đã che, trả về nguyên bản nếu không chứa ký tự `@`.

### 2.2. Bổ Sung Phương Thức In-Place Trên `AccountCardWidget` (`internal/ui/widgets.go`)
- **`SetMasked(masked bool)`**:
  - Khi `masked == true`: `w.emailLabel.SetText(MaskEmail(w.Email))`.
  - Khi `masked == false`: `w.emailLabel.SetText(w.Email)`.
  - Không động chạm trường `w.emailLabel.fullText`, đảm bảo popup tooltip hover luôn bung email gốc nguyên bản.
- **Exported helper methods cho kiểm thử**:
  - `EmailLabelText() string`: Trả về text hiển thị hiện hành.
  - `EmailFullText() string`: Trả về fullText nguyên bản phục vụ kiểm chứng tooltip.

### 2.3. Nối Trạng Thái Toggle Vào App State (`internal/ui/app.go`)
- **`SetMaskAccounts(mask bool)`**:
  - Kiểm tra trạng thái hiện hành để tránh đệ quy và xung đột file lock.
  - Cập nhật `a.maskAccounts = mask` dưới mutex bảo vệ thread-safe.
  - Đồng bộ `a.maskAccountsToggle.SetOn(mask)`.
  - Duyệt in-place toàn bộ `a.accountCards` trong RAM gọi `card.SetMasked(mask)` (Zero Layout Jank).
  - Khởi chạy goroutine non-blocking ghi cấu hình vào `.ui_preferences.json`.
  - Phản hồi thanh trạng thái tức thì: `"🔒 Đã ẩn thông tin tài khoản"` khi bật, `"🔓 Đã hiển thị đầy đủ tài khoản"` khi tắt, tự ẩn sau 2 giây.
- **`ToggleMaskAccounts()`**: Đảo trạng thái qua `SetMaskAccounts(!IsMaskAccounts())`.
- **Khởi tạo ban đầu (`NewAppWithFyne`)**: Đã tích hợp nạp `a.maskAccounts = prefs.MaskAccounts` từ trước.
- **Render Card mới (`updateUI`, `UpdateSingleAccountQuota`)**:
  - Khi khởi tạo `NewAccountCardWidgetWithQuota`: kiểm tra `if a.maskAccounts { cardWidget.SetMasked(true) }` để card mới lập tức che email nếu chế độ ẩn đang bật.
- **Bảo vệ Thanh Trạng Thái (Status Bar Protection)**:
  - Trong `SwitchAccount(targetEmail)`: Nếu `a.IsMaskAccounts()`, chuyển đổi email hiển thị thành `MaskEmail(targetEmail)` cho cả thông báo đang chuyển (`⏳`) và thông báo chuyển thành công (`✅`).
  - Trong `handleAutomationStatus(evt)`: Nếu `a.IsMaskAccounts()`, tự động che email trong `evt.Message` qua `maskMessageEmails(msg)`.

---

## 3. Bằng Chứng Kiểm Thử & Nghiệm Thu (Evidence)

### 3.1. Chạy Bộ Kiểm Thử Chuyên Biệt Chặng 2 (`go test -v -run "TestMaskEmail|TestArchitectSpec_MaskAccounts" ./internal/ui`)
```text
=== RUN   TestArchitectSpec_MaskAccounts_Persistence
--- PASS: TestArchitectSpec_MaskAccounts_Persistence (0.33s)
=== RUN   TestMaskEmail_AlgorithmConformance
=== RUN   TestMaskEmail_AlgorithmConformance/Standard_gmail
=== RUN   TestMaskEmail_AlgorithmConformance/Dotted_username_and_io_domain
=== RUN   TestMaskEmail_AlgorithmConformance/Single_character_username
=== RUN   TestMaskEmail_AlgorithmConformance/Plus_tag_and_multi-level_domain
=== RUN   TestMaskEmail_AlgorithmConformance/Invalid_format_without_@
=== RUN   TestMaskEmail_AlgorithmConformance/Empty_string
=== RUN   TestMaskEmail_AlgorithmConformance/No_username_before_@
=== RUN   TestMaskEmail_AlgorithmConformance/Whitespace_trimmed
=== RUN   TestMaskEmail_AlgorithmConformance/UTF-8_rune_safe
--- PASS: TestMaskEmail_AlgorithmConformance (0.00s)
    --- PASS: TestMaskEmail_AlgorithmConformance/Standard_gmail (0.00s)
    --- PASS: TestMaskEmail_AlgorithmConformance/Dotted_username_and_io_domain (0.00s)
    --- PASS: TestMaskEmail_AlgorithmConformance/Single_character_username (0.00s)
    --- PASS: TestMaskEmail_AlgorithmConformance/Plus_tag_and_multi-level_domain (0.00s)
    --- PASS: TestMaskEmail_AlgorithmConformance/Invalid_format_without_@ (0.00s)
    --- PASS: TestMaskEmail_AlgorithmConformance/Empty_string (0.00s)
    --- PASS: TestMaskEmail_AlgorithmConformance/No_username_before_@ (0.00s)
    --- PASS: TestMaskEmail_AlgorithmConformance/Whitespace_trimmed (0.00s)
    --- PASS: TestMaskEmail_AlgorithmConformance/UTF-8_rune_safe (0.00s)
=== RUN   TestArchitectSpec_MaskAccountsToggle_Behavior
--- PASS: TestArchitectSpec_MaskAccountsToggle_Behavior (0.08s)
=== RUN   TestArchitectSpec_MaskAccounts_StatusBarMasking
--- PASS: TestArchitectSpec_MaskAccounts_StatusBarMasking (0.41s)
=== RUN   TestArchitectSpec_MaskAccounts_NewCardInitialMask
--- PASS: TestArchitectSpec_MaskAccounts_NewCardInitialMask (0.07s)
PASS
ok  	antigravity-switch/internal/ui	2.476s
```

### 3.2. Toàn Bộ Test Suite Giao Diện (`go test -count=1 ./internal/ui`)
```text
ok  	antigravity-switch/internal/ui	14.302s
```
Tất cả 65+ unit tests trong gói `internal/ui` PASS 100%.

### 3.3. Script Build Windows Chuẩn Repo (`pwsh -File .\scripts\build_windows.ps1 -Mode all`)
```text
================================================================================
⚡ Antigravity Multi-Account Switcher - Windows Build System
   Mode: all | OutDir: bin | SkipTests: False | Clean: False
================================================================================
[2/4] Running pre-flight tests & vet...
  -> Verifying pure Go packages (CGO_ENABLED=0)...
ok  	antigravity-switch/internal/model	(cached)
ok  	antigravity-switch/internal/wincred	(cached)
ok  	antigravity-switch/internal/store	(cached)
ok  	antigravity-switch/internal/scanner	(cached)
ok  	antigravity-switch/internal/connector	(cached)
ok  	antigravity-switch/internal/reloader	(cached)
ok  	antigravity-switch/internal/manager	3.187s
ok  	antigravity-switch/cmd/switcher	2.240s
  [PASS] Pure Go verification succeeded.
  -> Verifying GUI packages with CGo & Fyne (CGO_ENABLED=1)...
ok  	antigravity-switch/cmd/switcher	1.746s
ok  	antigravity-switch/internal/ui	15.082s
  [PASS] GUI verification succeeded.
[3/4] Compiling target binaries...
  -> Checking Windows application icon and resource...
  -> Compiling resource with windres: cmd\switcher\app.rc -> cmd\switcher\rsrc_windows_amd64.syso
  [SUCCESS] Compiled PE resource object: cmd\switcher\rsrc_windows_amd64.syso (12552 bytes)
  -> Building Standalone CLI Binary (Pure Go, CGO_ENABLED=0)...
  [SUCCESS] Created: bin\antigravity-switch.exe
  Compile Duration: 3021 ms | Size: 10.53 MB (11036672 bytes)
  -> Building Standalone GUI Binary (Fyne, CGO_ENABLED=1)...
  [SUCCESS] Created: bin\antigravity-switch-gui.exe
  Compile Duration: 5840 ms | Size: 27.77 MB (29119488 bytes)
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

### 3.4. Kiểm Tra Thực Tế Binary Mới Tạo
```text
Mode                 LastWriteTime         Length Name
----                 -------------         ------ ----
-a---          10/20/2026 10:32 PM       29119488 antigravity-switch-gui.exe
-a---          10/20/2026 10:32 PM       11036672 antigravity-switch.exe
```

---

## 4. Danh Mục Tập Tin Thay Đổi (Touch-Map Khóa Cứng)

1. `internal/ui/widgets.go`:
   - Bổ sung `MaskEmail(email string) string` và `maskMessageEmails(msg string) string`.
   - Bổ sung `SetMasked(masked bool)` trên `AccountCardWidget`.
   - Bổ sung `EmailLabelText()` và `EmailFullText()` phục vụ xác thực giao diện.
2. `internal/ui/app.go`:
   - Hoàn thiện `SetMaskAccounts(mask bool)` với in-place card updates và status bar transient feedback.
   - Thêm logic che email trong `updateUI`, `UpdateSingleAccountQuota`, `SwitchAccount`, `handleAutomationStatus`.
3. `internal/ui/ui_test.go`:
   - Thêm `TestMaskEmail_AlgorithmConformance` (9 test cases).
   - Thêm `TestArchitectSpec_MaskAccountsToggle_Behavior`.
   - Thêm `TestArchitectSpec_MaskAccounts_StatusBarMasking`.
   - Thêm `TestArchitectSpec_MaskAccounts_NewCardInitialMask`.
   - Ổn định hóa tempdir file lock cho các ca test trên Windows.

---

## 5. Kết Luận
- **CHẶNG 2: CHỨC NĂNG (FUNCTIONALITY & MASKING)** đã hoàn thành 100% theo đúng đặc tả kiến trúc `SPEC-ARCH-20261020-ACCOUNT-PRIVACY-MASKING-TOOLBAR`.
- Nút gạt "Tự động" cũ được bảo toàn nguyên vẹn 100%.
- Không thay đổi model dữ liệu backend ngầm (View-only presentation filter).
- Tận dụng 100% widget hover tooltip sẵn có (zero code popup thừa).
- Build script `pwsh -File .\scripts\build_windows.ps1 -Mode all` PASS 100%, sinh binary GUI và CLI sẵn sàng nghiệm thu.
