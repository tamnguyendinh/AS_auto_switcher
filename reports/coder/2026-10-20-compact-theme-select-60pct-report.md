# Coder Implementation Report: Thu Nhỏ Dropdown Chọn Theme Xuống 60% (Compact Theme Selector)

- **Date:** 2026-10-20
- **Role:** Coder
- **Authority / SSOT:**
  - `reports/architect/2026-10-20-compact-theme-select-60pct-spec.md` (Phê duyệt chính thức bởi Human Owner)
  - Mã văn kiện đặc tả: `SPEC-ARCH-20261020-COMPACT-THEME-SELECT-60PCT`
  - Base Commit: `c8c8b05`
  - `AGENTS.md` & `working-rules`
- **Scope (Touch-Map Khóa Cứng):**
  - `internal/ui/window.go` (thu nhỏ kích thước `themeSelectWrap` GridWrap từ 58x32 px xuống 36x20 px)
  - `internal/ui/ui_test.go` (cập nhật assertion kích thước `themeSelectWrap` và bổ sung test case `TestArchitectSpec_CompactThemeSelect60Pct_FullConformance`)
  - `reports/coder/2026-10-20-compact-theme-select-60pct-report.md` (báo cáo thi công của coder)

---

## 1. Mục Tiêu & Tuân Thủ Kỷ Luật Sắt (Iron Rules Compliance)

1. **Thu nhỏ cả độ cao và độ rộng của dropdown theme về ~60%:**
   - Kích thước cũ: `58 x 32 px` (quá to bè so với cỡ chữ 10pt của các nhãn xung quanh trên Header Row 2).
   - Kích thước mới: **`36.0 x 20.0 px`** trong `container.NewGridWrap(fyne.NewSize(36, 20), a.themeSelect)`.
   - Chiều rộng: $36 / 58 \approx 62\%$.
   - Chiều cao: $20 / 32 \approx 62.5\%$.
   - Đạt chuẩn xác chỉ đạo của Owner: *"dọ cao + độ rọng của dopdow theme qua lon so với kich thucoj của text xong quanh, can nho lại con khoảng 60%"*.
2. **Đồng bộ chiều cao với cụm Hàng 2 (Row 2 Vertical Harmony):**
   - Chiều cao `20.0 px` ngang bằng hoàn hảo với công tắc gạt tự động `autoSwitchToggle` (`38 x 20 px`), tạo nên một trục ngang phẳng phiu, thanh thoát và liền mạch trên cụm góc phải (`rightCluster`).
3. **Bảo tồn nguyên vẹn các tùy chọn & chức năng:**
   - Giữ nguyên 3 tùy chọn theme: `🌙`, `☀️`, `☕`.
   - Giữ nguyên 100% logic chuyển đổi theme khi chọn: `a.SetTheme(...)`.
4. **Bảo tồn nguyên vẹn các kích thước đã khóa cứng:**
   - Cửa sổ ứng dụng: CỐ ĐỊNH **`920 x 630 px`** (`DefaultWindowWidth = 920.0`, `DefaultWindowHeight = 630.0`).
   - Thẻ Card: CỐ ĐỊNH **`280 x 130 px`** trong `newSymmetricGridWrap`.
   - Bảng Quota: CỐ ĐỊNH **`228 x 77 px`**.
5. **Cấm tuyệt đối sử dụng Anvien:**
   - Không gọi hoặc sử dụng bất kỳ công cụ nào thuộc plugin anvien.
6. **Full build bằng script repo nguyên bản:**
   - `scripts/build_windows.ps1` nguyên vẹn 100%, không bị sửa đổi.
   - Chạy `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all` đạt 100% test PASS, biên dịch thành công cả CLI (`bin/antigravity-switch.exe`) lẫn GUI (`bin/antigravity-switch-gui.exe`), exit code 0.

---

## 2. Invariant Family Map & Chi Tiết Thi Công

| Invariant / Surface | Target File | Chi Tiết Thi Công Cụ Thể |
|---|---|---|
| **Theme Selector Geometry** | `internal/ui/window.go` | Thay thế dòng 174:<br>`themeSelectWrap := container.NewGridWrap(fyne.NewSize(58, 32), a.themeSelect)`<br>thành:<br>`themeSelectWrap := container.NewGridWrap(fyne.NewSize(36, 20), a.themeSelect)`.<br>Kích thước bọc được thu nhỏ còn 36x20 px, đồng bộ chiều cao 20px với `autoSwitchToggle`. |
| **Unit Test Suite Conformance** | `internal/ui/ui_test.go` | 1. Trong `TestArchitectSpec_MergeToolbarWithServerStatusRow_FullConformance`:<br>Bổ sung kiểm tra kích thước `themeSelectWrap` có chiều rộng $\le 40.0\text{ px}$ và chiều cao $\le 22.0\text{ px}$.<br>2. Bổ sung test case toàn diện `TestArchitectSpec_CompactThemeSelect60Pct_FullConformance`:<br>- Xác định vị trí `themeSelectWrap` trong `row2Combined`.<br>- Kiểm tra MinSize đạt `36x20 px` (AC-1).<br>- Kiểm tra chiều cao `20.0 px` khớp với `autoSwitchToggle.MinSize().Height` (AC-2).<br>- Kiểm tra chuyển đổi theme mượt mà giữa `☀️` (Light), `☕` (Warm), `🌙` (Dark) (AC-3).<br>- Kiểm tra bảo toàn kích thước cửa sổ 920x630 px và card 280x130 px (AC-4). |

---

## 3. Ranh Giới Touch-Map & Non-Goals

- **Chỉ sửa đúng 2 file mã nguồn theo chỉ định của CEO:**
  - `internal/ui/window.go`
  - `internal/ui/ui_test.go`
- **Non-goals tuân thủ tuyệt đối:**
  - Không làm xáo trộn bất kỳ widget nào khác trên Hàng 2 (`serverStatusLabel`, `toolbarLeft`, `autoSwitchToggle`).
  - Không thay đổi bất kỳ logic chuyển đổi theme nào của Fyne.
  - Không chỉnh sửa file kịch bản build `scripts/build_windows.ps1`.

---

## 4. Đối Chiếu Tiêu Chí Nghiệm Thu (Acceptance Criteria Matrix)

| Tiêu Chí | Nội Dung Đặc Tả Kiến Trúc | Trạng Thái | Minh Chứng Kiểm Thử |
|---|---|---|---|
| **AC-1** | Dropdown theme được thu nhỏ xuống 36x20 px (~60% kích thước cũ 58x32 px) | **PASS** | `themeWrapMin.Width == 36.0`, `themeWrapMin.Height == 20.0` trong `TestArchitectSpec_CompactThemeSelect60Pct_FullConformance`. |
| **AC-2** | Chiều cao của theme selector bằng 20.0 px, đồng bộ với `autoSwitchToggle` (20px) | **PASS** | `themeWrapMin.Height == 20.0` và `themeWrapMin.Height == toggleMin.Height` pass. |
| **AC-3** | Bấm chọn theme `🌙`, `☀️`, `☕` thay đổi theme mượt mà 100% | **PASS** | Chuyển đổi qua lại giữa `ThemeLight`, `ThemeWarm`, `ThemeDark` pass trong test. |
| **AC-4** | Kích thước cửa sổ giữ nguyên 920x630 px, card giữ nguyên 280x130 px | **PASS** | `DefaultWindowWidth == 920.0`, `DefaultWindowHeight == 630.0`, `cardMin <= 280x130` pass. |
| **AC-5** | Toàn bộ unit test suite `internal/ui` pass 100% | **PASS** | `go test ./internal/ui` pass 100% không có lỗi. |

---

## 5. Bằng Chứng Kiểm Thử & Biên Dịch (Verification Evidence)

### 5.1. Unit Test Chuyên Biệt
```text
=== RUN   TestArchitectSpec_MergeToolbarWithServerStatusRow_FullConformance
--- PASS: TestArchitectSpec_MergeToolbarWithServerStatusRow_FullConformance (0.60s)
=== RUN   TestArchitectSpec_CompactThemeSelect60Pct_FullConformance
--- PASS: TestArchitectSpec_CompactThemeSelect60Pct_FullConformance (0.38s)
PASS
ok  	antigravity-switch/internal/ui	2.366s
```

### 5.2. Toàn Bộ Package Unit Tests
```text
ok  	antigravity-switch/internal/ui	11.644s
```

### 5.3. Full Windows Build Automation Script (`scripts/build_windows.ps1 -Mode all`)
```text
================================================================================
⚡ Antigravity Multi-Account Switcher - Windows Build System
   Mode: all | OutDir: bin | SkipTests: False | Clean: False
================================================================================
[2/4] Running pre-flight tests & vet...
  -> Verifying pure Go packages (CGO_ENABLED=0)...
ok  	antigravity-switch/internal/model	(cached)
ok  	antigravity-switch/internal/wincred	(cached)
ok  	antigravity-switch/internal/store	4.434s
ok  	antigravity-switch/internal/scanner	(cached)
ok  	antigravity-switch/internal/connector	(cached)
ok  	antigravity-switch/internal/reloader	(cached)
ok  	antigravity-switch/internal/manager	6.028s
ok  	antigravity-switch/cmd/switcher	(cached)
  [PASS] Pure Go verification succeeded.
  -> Verifying GUI packages with CGo & Fyne (CGO_ENABLED=1)...
ok  	antigravity-switch/cmd/switcher	1.552s
ok  	antigravity-switch/internal/ui	15.179s
  [PASS] GUI verification succeeded.
[3/4] Compiling target binaries...
  -> Checking Windows application icon and resource...
  -> Compiling resource with windres: cmd\switcher\app.rc -> cmd\switcher\rsrc_windows_amd64.syso
  [SUCCESS] Compiled PE resource object: cmd\switcher\rsrc_windows_amd64.syso (12552 bytes)
  -> Building Standalone CLI Binary (Pure Go, CGO_ENABLED=0)...
  [SUCCESS] Created: bin\antigravity-switch.exe
  Compile Duration: 2967 ms | Size: 10.51 MB (11023872 bytes)
  -> Building Standalone GUI Binary (Fyne, CGO_ENABLED=1)...
  [SUCCESS] Created: bin\antigravity-switch-gui.exe
  Compile Duration: 6179 ms | Size: 27.62 MB (28957184 bytes)
[4/4] Executing quick verification...
  [PASS] Embedded PE application icon verified on bin\antigravity-switch.exe (32x32)
  [PASS] Embedded PE application icon verified on bin\antigravity-switch-gui.exe (32x32)
  -> Running 'bin\antigravity-switch.exe --status'...
================================================================================
⚡ ANTIGRAVITY SESSION STATUS
================================================================================
Active Email:    quantritaikhoanemail@gmail.com
Language Server: Running (PID: 5284 | Port: 49952 | CSRF: Present)
Win32 Credential: Present (260 bytes access token)
================================================================================
================================================================================
⚡ BUILD SUCCESSFUL
================================================================================
```

---

## 6. Kết Luận & Handoff Checklist

- **Kết luận:** Hoàn thành xuất sắc và chính xác 100% mục tiêu thu nhỏ Theme Select xuống ~60% (36x20 px) theo đúng thiết kế kiến trúc đã được phê duyệt.
- **Trạng thái:** `[VERDICT: PASS / READY_FOR_SUPERVISOR]`
