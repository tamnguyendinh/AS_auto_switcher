# Báo Cáo Thi Công Coder: Gỡ Bỏ Toàn Bộ Các Hàm Tạo Ra Sự Co Giãn Của UI & Khóa Chết Kích Thước Cố Định (Zero Resizing Purge)

- **Ngày thi công:** 2026-10-19
- **Vai trò:** Coder
- **Thẩm quyền:** Lệnh tối cao trực tiếp từ Human Owner qua CEO Agent
- **Tài liệu tham chiếu / SSOT:**
  - `reports/architect/2026-10-19-purge-all-resizing-functions-spec.md` (Commit c1d0315)
  - `SPEC-ARCH-20261019-PURGE-ALL-RESIZING-FUNCTIONS`
  - `AGENTS.md` & `working-rules`
- **Mục tiêu hoàn thành:**
  1. Gỡ bỏ triệt để toàn bộ các hàm, phương thức, style và lệnh tạo ra sự co giãn của UI (cấm sửa chắp vá).
  2. Thiết lập tỷ lệ và kích thước cố định chuẩn ngành cho toàn bộ giao diện từ ngoài vào trong: Cửa sổ chuẩn `1008 x 630 px`, `FixedSize == true`.
  3. Khung danh sách tài khoản cố định bằng `container.NewGridWrap(fyne.NewSize(318, 110))`, thẻ Card cố định `318 x 110 px`.
  4. Bảng Quota Matrix khóa cứng: Model `50.0 px`, Separator `1.0 px`, Gap `4.0 px`, Cột 5h `88.0 px`, Cột Week `88.0 px`, Ray tiến trình nền `40.0 px`.
  5. Cập nhật dữ liệu tại chỗ: Duy nhất vệt màu `fillBar` co ngắn lại (tụt dần) theo % quota trong ray 40px, cấm mọi thành phần khác co giãn.
  6. Thi công chính xác 100% theo đúng mã mẫu do Kiến trúc sư cung cấp trong văn kiện, TUYỆT ĐỐI CẤM SUY DIỄN.

---

## 1. Invariant Family Map

| Invariant | Tệp thực hiện | Chi tiết thi công kỹ thuật |
|---|---|---|
| **INV-01 (Purge OS Window Resize)** | `internal/ui/frameless_windows.go` | - Gỡ bỏ cờ `WS_THICKFRAME` trong `MakeWindowFrameless`: sửa thành `newStyle := style &^ uintptr(WS_CAPTION \| WS_THICKFRAME)` để loại bỏ hoàn toàn viền kéo co giãn của Windows ở cấp độ Win32 API.<br>- Vô hiệu hóa hàm `ToggleMaximizeNativeWindow()` thành no-op; dọn sạch các hằng số/biến Win32 không còn sử dụng (`procIsZoomed`, `SW_MAXIMIZE`, `SW_RESTORE`). |
| **INV-02 (Purge Maximize Button & Dynamic Grid)** | `internal/ui/window.go` | - Trong `setupLayout()`: Xóa bỏ nút phóng to `btnMax`. Cụm nút điều khiển cửa sổ chỉ còn đúng 2 nút: Thu nhỏ Taskbar (`btnMin`) và Đóng (`btnClose`): `windowControls := container.NewHBox(btnMin, btnClose)`.<br>- Xóa bỏ lưới tự động co dãn `container.NewGridWithColumns(3)`. Thay thế bằng `container.NewGridWrap(fyne.NewSize(318, 110))` để khóa cứng kích thước từng thẻ Card là `318px x 110px`. |
| **INV-03 (Lock Window Fixed Size & In-Place Quota Update)** | `internal/ui/app.go` | - Trong `Init()`: Gỡ bỏ `a.win.SetFixedSize(false)`, thay bằng `a.win.SetFixedSize(true)` để Fyne GLFW driver khóa chết kích thước cửa sổ ở mức `1008 x 630 px`, vô hiệu hóa cơ chế tự động bung nội dung `fitContent()`.<br>- Trong `updateUI()`: Khi quota cập nhật, cơ chế `canUpdateInPlace` gọi trực tiếp `a.accountCards[i].UpdateQuota(quotaData, palette)`, giữ nguyên 100% thẻ Card đã có, tuyệt đối không tạo mới Canvas objects hay xóa `Objects = nil`. |
| **INV-04 (Fixed Matrix Quota & Decreasing Fillbar Only)** | `internal/ui/quota_badge.go` | - Toàn bộ các cột khóa cứng: Model `50.0px`, Sep `1.0px`, Gap `4.0px`, 5h `88.0px`, Week `88.0px`, Ray nền `40.0px`. Tổng bảng quota `244.0 x 49.0 px`. Xóa bỏ hoàn toàn biến `remainW` và hàm `math.Min(remainW/2.0, 125.0)`.<br>- Trong `matrixQuotaCell.update()`: Duy nhất vệt màu `fillBar` co ngắn lại theo công thức `fw = barWidth * float32(clamped)` (với `barWidth = 40.0px`), cập nhật `pctText`. Xóa bỏ hoàn toàn `c.container.Refresh()` để ngăn layout cascade. |
| **INV-05 (Preferences SSOT & Theme Only)** | `internal/ui/themes.go`<br>`accounts_store/.ui_preferences.json` | - `UIPreferences` chỉ lưu duy nhất trường `Theme string json:"theme"`.<br>- `accounts_store/.ui_preferences.json` lưu nguyên bản `{"theme": "dark"}`. |
| **INV-06 (Automated Regression & Fixed Metrics Assertions)** | `internal/ui/ui_test.go` | - Cập nhật `TestOwnerRedesign_WindowSizeAndLayout` và `TestIndustryStandard_WindowInit1008x630`: xác nhận `win.FixedSize() == true` và Canvas size `1008 x 630 px`.<br>- Thêm `TestPurgeAllResizing_WindowControlsTwoButtons`: xác nhận `windowControls` chỉ chứa đúng 2 nút (`btnMin` và `btnClose`), không có nút phóng to `btnMax`.<br>- Thêm `TestPurgeAllResizing_AccountGridWrapFixedSize`: xác nhận `accountListContainer` sử dụng `GridWrap` cố định kích thước thẻ `318 x 110 px`. |

---

## 2. Touch-Map Compliance Verification

Tuân thủ nghiêm ngặt ranh giới Touch-Map được giao, tuyệt đối không chỉnh sửa bất kỳ file nào ngoài phạm vi:

| STT | File Path | Trạng thái | Tóm tắt kiểm tra |
|---|---|---|---|
| 1 | `internal/ui/frameless_windows.go` | Modified | Gỡ bỏ `WS_THICKFRAME`, vô hiệu hóa `ToggleMaximizeNativeWindow`. |
| 2 | `internal/ui/window.go` | Modified | Xóa bỏ `btnMax`, chỉ giữ `btnMin` và `btnClose`; thay `GridWithColumns(3)` bằng `GridWrap(318, 110)`. |
| 3 | `internal/ui/app.go` | Modified | `SetFixedSize(true)` khóa cứng cửa sổ 1008x630; cập nhật quota in-place. |
| 4 | `internal/ui/quota_badge.go` | Checked/Aligned | Cột cố định 50px, 88px, 88px, ray 40px; chỉ duy nhất `fillBar` co ngắn lại; không gọi `c.container.Refresh()`. |
| 5 | `internal/ui/themes.go` | Checked/Aligned | `UIPreferences` chỉ lưu `Theme`. |
| 6 | `accounts_store/.ui_preferences.json` | Checked/Aligned | Cấu hình mặc định `{"theme": "dark"}`. |
| 7 | `internal/ui/ui_test.go` | Modified | Cập nhật assertions `FixedSize() == true`, thêm test 2 nút điều khiển và test `GridWrap(318, 110)`. |
| 8 | `reports/coder/2026-10-19-purge-all-resizing-functions.md` | Created | Báo cáo thi công chính thức của Coder. |

**Non-goals Verified:**
- NO Anvien utilized.
- File `scripts/build_windows.ps1` tuyệt đối giữ nguyên trạng không bị can thiệp.
- Không spawn subagent.
- Đúng 100% mã mẫu của Kiến trúc sư, không suy diễn thêm bớt.

---

## 3. Mandatory Evidence: Test Suite Verification

### 3.1. GUI Unit Tests:
```powershell
go test -tags gui -v -run "TestPurgeAllResizing|TestOwnerRedesign_WindowSizeAndLayout|TestIndustryStandard_WindowInit1008x630" ./internal/ui/...
```
Output:
```text
=== RUN   TestOwnerRedesign_WindowSizeAndLayout
--- PASS: TestOwnerRedesign_WindowSizeAndLayout (0.35s)
=== RUN   TestIndustryStandard_WindowInit1008x630
--- PASS: TestIndustryStandard_WindowInit1008x630 (0.12s)
=== RUN   TestPurgeAllResizing_WindowControlsTwoButtons
--- PASS: TestPurgeAllResizing_WindowControlsTwoButtons (0.15s)
=== RUN   TestPurgeAllResizing_AccountGridWrapFixedSize
--- PASS: TestPurgeAllResizing_AccountGridWrapFixedSize (0.12s)
PASS
ok  	antigravity-switch/internal/ui	1.985s
```

Full GUI Package test:
```powershell
go test -tags gui ./internal/ui/...
```
Output:
```text
ok  	antigravity-switch/internal/ui	11.415s
```
**Kết quả:** 100% PASS.

### 3.2. Toàn Bộ Packages của Repository:
```powershell
go test ./...
```
Output:
```text
ok  	antigravity-switch/cmd/switcher	(cached)
ok  	antigravity-switch/internal/connector	(cached)
ok  	antigravity-switch/internal/manager	11.130s
ok  	antigravity-switch/internal/model	(cached)
ok  	antigravity-switch/internal/reloader	(cached)
ok  	antigravity-switch/internal/scanner	(cached)
ok  	antigravity-switch/internal/store	12.124s
ok  	antigravity-switch/internal/telemetry	(cached)
ok  	antigravity-switch/internal/ui	13.256s
ok  	antigravity-switch/internal/wincred	(cached)
```
**Kết quả:** 100% PASS trên tất cả các package.

---

## 4. Mandatory Evidence: Full Build Verification

Thực thi lệnh kiểm tra bắt buộc của repo:
```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all
```

**Nhật ký thực thi:**
```text
================================================================================
⚡ Antigravity Multi-Account Switcher - Windows Build System
   Mode: all | OutDir: bin | SkipTests: False | Clean: False
================================================================================
[2/4] Running pre-flight tests & vet...
  -> Verifying pure Go packages (CGO_ENABLED=0)...
ok  	antigravity-switch/internal/model	(cached)
ok  	antigravity-switch/internal/wincred	(cached)
ok  	antigravity-switch/internal/store	5.614s
ok  	antigravity-switch/internal/scanner	(cached)
ok  	antigravity-switch/internal/connector	(cached)
ok  	antigravity-switch/internal/reloader	(cached)
ok  	antigravity-switch/internal/manager	8.834s
ok  	antigravity-switch/cmd/switcher	(cached)
  [PASS] Pure Go verification succeeded.
  -> Verifying GUI packages with CGo & Fyne (CGO_ENABLED=1)...
ok  	antigravity-switch/cmd/switcher	1.596s
ok  	antigravity-switch/internal/ui	10.120s
  [PASS] GUI verification succeeded.
[3/4] Compiling target binaries...
  -> Checking Windows application icon and resource...
  -> Compiling resource with windres: cmd\switcher\app.rc -> cmd\switcher\rsrc_windows_amd64.syso
  [SUCCESS] Compiled PE resource object: cmd\switcher\rsrc_windows_amd64.syso (12552 bytes)
  -> Building Standalone CLI Binary (Pure Go, CGO_ENABLED=0)...
  [SUCCESS] Created: bin\antigravity-switch.exe
  Compile Duration: 3488 ms | Size: 10.51 MB (11023872 bytes)
  -> Building Standalone GUI Binary (Fyne, CGO_ENABLED=1)...
  [SUCCESS] Created: bin\antigravity-switch-gui.exe
  Compile Duration: 6928 ms | Size: 27.57 MB (28904448 bytes)
[4/4] Executing quick verification...
  [PASS] Embedded PE application icon verified on bin\antigravity-switch.exe (32x32)
  [PASS] Embedded PE application icon verified on bin\antigravity-switch-gui.exe (32x32)
  -> Running 'bin\antigravity-switch.exe --status'...
================================================================================
⚡ ANTIGRAVITY SESSION STATUS
================================================================================
Active Email:    ngau2hung@gmail.com
Language Server: Running (PID: 21484 | Port: 65144 | CSRF: Present)
Win32 Credential: Present (260 bytes access token)
================================================================================
================================================================================
⚡ BUILD SUCCESSFUL
================================================================================
```
**Trạng thái kết thúc:** Exit code = 0.

---

## 5. Đánh Giá Tiêu Chuẩn Nghiệm Thu (Acceptance Criteria Evaluation)

1. **AC-01 (Purged All Resizing Functions) — ĐẠT:**  
   Cờ `WS_THICKFRAME` trong Win32 frameless style đã bị xóa bỏ. `ToggleMaximizeNativeWindow()` bị vô hiệu hóa. Nút phóng to `btnMax` đã bị gỡ khỏi layout. Lưới co giãn `GridWithColumns(3)` được thay bằng `GridWrap(318, 110)`. Lệnh `SetFixedSize(false)` bị xóa bỏ và thay bằng `SetFixedSize(true)`. Các phép tính co giãn động và lệnh `c.container.Refresh()` bị loại trừ hoàn toàn.

2. **AC-02 (Fixed Window 100%) — ĐẠT:**  
   Cửa sổ khởi tạo chuẩn kích thước `1008 x 630 px` với `a.win.SetFixedSize(true)`. Cửa sổ không có viền kéo chuột co giãn của Windows và không có nút phóng to. Cửa sổ đứng im tuyệt đối, không co giãn vì bất kỳ lý do gì.

3. **AC-03 (Fixed Internal Metrics) — ĐẠT:**  
   Tất cả các thành phần giao diện tuân thủ số đo cố định: Thẻ Card cố định `318 x 110 px`, Bảng Quota Matrix cố định `244 x 49 px`, Cột Model `50.0 px`, Cột 5h `88.0 px`, Cột Week `88.0 px`, Ray nền tiến trình `40.0 x 6.0 px`.

4. **AC-04 (Only Quota Decreases) — ĐẠT:**  
   Khi Quota cập nhật, `canUpdateInPlace` gọi trực tiếp `UpdateQuota` cho từng thẻ Card mà không phá hủy hay tái tạo Canvas objects. Duy nhất chiều dài của vệt màu `fillBar` co ngắn lại (tụt dần) theo % quota trong ray 40px. Mọi thành phần khác đứng im 100%.

5. **AC-05 (Build & Tests) — ĐẠT:**  
   - `go test -tags gui ./internal/ui/...` đạt **100% PASS**.
   - `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all` đạt **Exit code = 0**.

---

## 6. Kết Luận & Bàn Giao

Thi công hoàn tất 100% mục tiêu của văn kiện đặc tả kỹ thuật `reports/architect/2026-10-19-purge-all-resizing-functions-spec.md`. Toàn bộ giao diện đã được khóa chết kích thước cố định và gỡ bỏ hoàn toàn mọi cơ chế co giãn. Sẵn sàng bàn giao cho Supervisor độc lập nghiệm thu.