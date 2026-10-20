# Coder Implementation Report: Điều Chỉnh Giao Diện Header Row 2 Theo Chỉ Đạo Trực Tiếp Của Owner

- **Date:** 2026-10-20
- **Role:** Coder
- **Authority / SSOT:**
  - Chỉ đạo trực tiếp từ Human Owner: *"tang dọ rong cảu dropdown them vè nhu cũ. thu nhỏ nut gạt on/off cau "tự dọng:" còn lại 60%"*
  - `reports/architect/2026-10-20-compact-theme-select-60pct-spec.md` (`SPEC-ARCH-20261020-COMPACT-THEME-SELECT-60PCT`)
  - `reports/architect/2026-10-20-auto-switch-toggle-button-spec.md` (`SPEC-ARCH-20261020-AUTO-SWITCH-TOGGLE-BUTTON`)
  - Base Commit: `2b7ac7f`
  - `AGENTS.md` & `working-rules`
- **Scope (Touch-Map Khóa Cứng):**
  - `internal/ui/window.go`:
    1. Tăng độ rộng của `themeSelectWrap` về kích thước cũ `58px` trong khi vẫn giữ chiều cao gọn gàng `20px` (`58 x 20 px`).
    2. Thu nhỏ công tắc gạt `autoSwitchToggle` ("Tự động:") xuống 60% từ `38 x 20 px` xuống `23 x 12 px`, nút tròn thumb từ `14 x 14 px` xuống `8 x 8 px`, CornerRadius `6.0 px`.
  - `internal/ui/ui_test.go`: Cập nhật toàn bộ các test assertions về kích thước và tọa độ con trượt thumb cho trạng thái OFF/ON của toggle và theme select.
  - `reports/coder/2026-10-20-compact-theme-select-60pct-report.md`: Báo cáo nghiệm thu chi tiết.

---

## 1. Mục Tiêu & Tuân Thủ Kỷ Luật Sắt

1. **Tăng độ rộng dropdown theme về như cũ (Width = 58px):**
   - Độ rộng được phục hồi từ 36px về **58.0 px** theo đúng kích thước ban đầu, giúp biểu tượng theme (`🌙`, `☀️`, `☕`) và mũi tên dropdown tam giác có đủ không gian hiển thị thoáng đãng, không bị chật chội hay tràn viền.
   - Chiều cao tiếp tục duy trì **20.0 px** (thanh thoát, không bị to bè 32px như trước).
2. **Thu nhỏ nút công tắc gạt "Tự động:" về 60%:**
   - Kích thước ray trượt (Track): Thu nhỏ từ `38 x 20 px` xuống **`23.0 x 12.0 px`** ($38 \times 0.6 = 22.8 \approx 23$, $20 \times 0.6 = 12.0$).
   - Bo góc ray trượt: `CornerRadius = 6.0 px` (khớp hoàn hảo với nửa chiều cao 12px tạo dáng capsule viên thuốc).
   - Nút tròn con trượt (Thumb): Thu nhỏ đường kính từ `14 x 14 px` xuống **`8.0 x 8.0 px`** ($14 \times 0.6 = 8.4 \approx 8$).
   - Tọa độ trạng thái OFF: $x = 2.0\text{ px}, y = 2.0\text{ px}$ (đệm đều 2px xung quanh).
   - Tọa độ trạng thái ON: $x = 13.0\text{ px}, y = 2.0\text{ px}$ ($23 - 8 - 2 = 13.0\text{ px}$).
   - Chiều cao 12px của công tắc gạt tương thích hoàn hảo với cỡ chữ 10pt của nhãn `"Tự động:"` bên cạnh, tạo tổng thể vô cùng tinh tế và đồng bộ.
3. **Bảo tồn kích thước hệ thống đã khóa cứng:**
   - Cửa sổ ứng dụng: CỐ ĐỊNH **`920 x 630 px`**.
   - Thẻ Card: CỐ ĐỊNH **`280 x 130 px`**.
   - Bảng Quota: CỐ ĐỊNH **`228 x 77 px`**.
4. **Cấm Anvien:**
   - Tuyệt đối không dùng công cụ anvien.
5. **Full build bằng script repo nguyên bản:**
   - `scripts/build_windows.ps1` nguyên vẹn 100%.
   - Chạy `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all` đạt Exit Code 0, build thành công cả CLI lẫn GUI.

---

## 2. Invariant Family Map & Chi Tiết Thi Công

| Thành Phần | Kích Thước Cũ | Kích Thước Mới (Theo Lệnh Owner) | File Sửa Đổi |
|---|---|---|---|
| **Theme Select Wrap** | `36 x 20 px` | **`58 x 20 px`** (Độ rộng về như cũ 58px, chiều cao giữ 20px) | `internal/ui/window.go` |
| **AutoSwitchToggle Track** | `38 x 20 px` (CornerRadius 10.0) | **`23 x 12 px`** (CornerRadius 6.0) (Giảm 60%) | `internal/ui/window.go` |
| **AutoSwitchToggle Thumb** | `14 x 14 px` | **`8 x 8 px`** (Giảm 60%) | `internal/ui/window.go` |
| **Toggle Thumb OFF Pos** | $(3.0, 3.0)$ | **$(2.0, 2.0)$** | `internal/ui/window.go` |
| **Toggle Thumb ON Pos** | $(21.0, 3.0)$ | **$(13.0, 2.0)$** | `internal/ui/window.go` |
| **Unit Test Assertions** | Assert 38x20, 36x20 | Cập nhật assertions khớp 58x20 và 23x12 | `internal/ui/ui_test.go` |

---

## 3. Đối Chiếu Tiêu Chí Nghiệm Thu

| Tiêu Chí | Nội Dung Yêu Cầu Của Owner | Kết Quả | Minh Chứng Kiểm Thử |
|---|---|---|---|
| **REQ-1** | Tăng độ rộng dropdown theme về như cũ (58px) | **PASS** | `themeWrapMin.Width == 58.0` trong `TestArchitectSpec_CompactThemeSelect60Pct_FullConformance` |
| **REQ-2** | Thu nhỏ nút gạt on/off "Tự động:" còn lại 60% (23x12px) | **PASS** | `minSize == (23, 12)`, `track.CornerRadius == 6.0`, `thumb == 8x8` trong `TestArchitectSpec_AutoSwitchToggle_CapsuleWidget_FullConformance` |
| **REQ-3** | Con trượt toggle hoạt động chính xác khi click ON/OFF | **PASS** | Chuyển đổi tọa độ $(2, 2) \leftrightarrow (13, 2)$ và màu sắc `BorderSubtle` $\leftrightarrow$ `Primary` pass 100% |
| **REQ-4** | Chức năng chuyển đổi theme và giữ nguyên cửa sổ 920x630px | **PASS** | Đổi theme `☀️`, `☕`, `🌙` và `DefaultWindowWidth=920, DefaultWindowHeight=630` pass 100% |
| **REQ-5** | Toàn bộ unit tests và build script pass 100% | **PASS** | `go test ./internal/ui` pass và `build_windows.ps1 -Mode all` Exit Code 0 |

---

## 4. Bằng Chứng Kiểm Thử & Biên Dịch

### 4.1. Unit Tests Chuyên Biệt
```text
=== RUN   TestArchitectSpec_MergeToolbarWithServerStatusRow_FullConformance
--- PASS: TestArchitectSpec_MergeToolbarWithServerStatusRow_FullConformance (0.35s)
=== RUN   TestArchitectSpec_AutoSwitchToggle_CapsuleWidget_FullConformance
--- PASS: TestArchitectSpec_AutoSwitchToggle_CapsuleWidget_FullConformance (0.39s)
=== RUN   TestArchitectSpec_CompactThemeSelect60Pct_FullConformance
--- PASS: TestArchitectSpec_CompactThemeSelect60Pct_FullConformance (0.33s)
PASS
ok  	antigravity-switch/internal/ui	2.421s
```

### 4.2. Toàn Bộ Package Unit Tests
```text
ok  	antigravity-switch/internal/ui	16.776s
```

### 4.3. Full Windows Build Automation Script (`scripts/build_windows.ps1 -Mode all`)
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
ok  	antigravity-switch/internal/manager	6.078s
ok  	antigravity-switch/cmd/switcher	(cached)
  [PASS] Pure Go verification succeeded.
  -> Verifying GUI packages with CGo & Fyne (CGO_ENABLED=1)...
ok  	antigravity-switch/cmd/switcher	(cached)
ok  	antigravity-switch/internal/ui	10.444s
  [PASS] GUI verification succeeded.
[3/4] Compiling target binaries...
  -> Checking Windows application icon and resource...
  -> Compiling resource with windres: cmd\switcher\app.rc -> cmd\switcher\rsrc_windows_amd64.syso
  [SUCCESS] Compiled PE resource object: cmd\switcher\rsrc_windows_amd64.syso (12552 bytes)
  -> Building Standalone CLI Binary (Pure Go, CGO_ENABLED=0)...
  [SUCCESS] Created: bin\antigravity-switch.exe
  Compile Duration: 3285 ms | Size: 10.51 MB (11023872 bytes)
  -> Building Standalone GUI Binary (Fyne, CGO_ENABLED=1)...
  [SUCCESS] Created: bin\antigravity-switch-gui.exe
  Compile Duration: 5422 ms | Size: 27.62 MB (28957184 bytes)
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

## 5. Trạng Thái & Kết Luận
- **Trạng thái:** `[VERDICT: PASS / READY_FOR_SUPERVISOR]`
