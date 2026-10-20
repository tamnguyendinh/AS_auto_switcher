# Coder Implementation Report: Thi Công Nút Công Tắc Gạt "Tự Động Đổi Tài Khoản: On/Off" Trên UI Tổng (Capsule Toggle Switch)

- **Date:** 2026-10-20
- **Role:** Coder
- **Authority / SSOT:**
  - `reports/architect/2026-10-20-auto-switch-toggle-button-spec.md` (Phê duyệt chính thức bởi Human Owner)
  - Mã văn kiện đặc tả: `SPEC-ARCH-20261020-AUTO-SWITCH-TOGGLE-BUTTON`
  - Base Commit: `5cf296c`
  - `AGENTS.md` & `working-rules`
- **Scope (Touch-Map Khóa Cứng):**
  - `internal/ui/window.go` (thi công AutoSwitchToggle capsule widget cạnh themeSelectWrap trên Row 2)
  - `internal/ui/app.go` (bổ sung autoSwitchEnabled bool, con trỏ widget, thread-safe accessor methods, cập nhật theme reactive)
  - `internal/ui/ui_test.go` (visual toggle state and interaction tests, cập nhật test suite conformance)
  - `reports/coder/2026-10-20-auto-switch-toggle-button-report.md` (báo cáo nghiệm thu của coder)

---

## 1. Mục Tiêu & Tuân Thủ Kỷ Luật Sắt (Iron Rules Compliance)

1. **Giới hạn tuyệt đối ở phạm vi giao diện (UI ONLY - NO BACKEND LOGIC):**
   - Nhiệm vụ chỉ tạo phần tử nút bấm widget trên giao diện và xử lý chuyển đổi trạng thái hiển thị (State Toggle & Visual Feedback khi click).
   - Tuyệt đối không lập trình bất kỳ logic nghiệp vụ tự động đổi tài khoản ngầm nào ở backend, không cài đặt daemon hooks.
2. **Phong cách thiết kế: Công tắc gạt dạng viên thuốc (Phong cách 1 - Capsule Toggle Switch):**
   - Thanh trượt hình viên thuốc bo tròn (Pill Capsule): Chiều rộng 38.0 px, chiều cao 20.0 px, `CornerRadius = 10.0 px`.
   - Nút tròn con trượt bên trong (Thumb knob): Đường kính 14.0 px, `CornerRadius = 7.0 px` (`canvas.NewCircle(color.White)`).
   - **Trạng thái OFF (Tắt):** Nền ray màu `palette.BorderSubtle`, nút tròn nằm bên trái tại tọa độ $(x=3.0, y=3.0)$.
   - **Trạng thái ON (Bật):** Nền ray màu xanh dương thương hiệu `palette.Primary`, nút tròn nằm bên phải tại tọa độ $(x=21.0, y=3.0)$.
   - Nhãn chữ đi kèm: `"Tự động:"` (10pt Bold, màu `palette.Text`), bọc cùng công tắc trong `autoCluster := container.NewHBox(lblAuto, a.autoSwitchToggle)`.
   - Con trỏ chuột: Implement `desktop.Cursorable`, trả về `desktop.PointerCursor` (hình bàn tay).
3. **Vị trí đặt trên Header Row 2:**
   - Đặt ở Hàng 2 gộp (`row2Combined`) bên cụm góc phải, nằm ngay cạnh trước nút đổi Theme (`themeSelectWrap`):
     ```go
     rightCluster := container.NewHBox(autoCluster, themeSelectWrap)
     row2Combined := container.NewBorder(nil, nil, row2LeftCluster, rightCluster)
     ```
4. **Bảo tồn nguyên vẹn các thông số đã khóa cứng:**
   - Cửa sổ ứng dụng: CỐ ĐỊNH **`920 x 630 px`** (`DefaultWindowWidth = 920.0`, `DefaultWindowHeight = 630.0`).
   - Thẻ Card: CỐ ĐỊNH **`280 x 130 px`** trong `newSymmetricGridWrap`.
   - Bảng Quota: CỐ ĐỊNH **`228 x 77 px`**.
   - In-place reactive update (Zero Jank / Zero Resizing) bảo toàn 100%.
5. **Cấm Anvien:**
   - Tuyệt đối không dùng công cụ anvien trong toàn bộ quá trình thi công.
6. **Full build bằng script repo nguyên bản:**
   - Chạy `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all` đạt 100% test PASS, biên dịch thành công cả CLI (`bin/antigravity-switch.exe`) lẫn GUI (`bin/antigravity-switch-gui.exe`), exit code 0.
   - Script build `scripts/build_windows.ps1` nguyên vẹn 100%, tuyệt đối không bị chỉnh sửa.

---

## 2. Invariant Family Map & Chi Tiết Thi Công

| Invariant / Surface | Target File | Chi Tiết Thi Công Cụ Thể |
|---|---|---|
| **App State & Lifecycle** | `internal/ui/app.go` | - Thêm trường `autoSwitchEnabled bool` vào `struct App` (mặc định khởi tạo `false`).<br>- Thêm con trỏ `autoSwitchToggle *AutoSwitchToggle` và `autoSwitchLabel *canvas.Text` để quản lý widget và cập nhật reactive khi đổi Theme.<br>- Bổ sung thread-safe accessors: `IsAutoSwitchEnabled() bool` và `SetAutoSwitchEnabled(enabled bool)`.<br>- Trong `SetTheme(themeName string)`: Cập nhật `autoSwitchToggle.SetPalette(palette)` và `autoSwitchLabel.Color = palette.Text` đảm bảo hiển thị đúng token màu trên cả 3 theme Dark, Light, Warm. |
| **Capsule Toggle Widget & Header Layout** | `internal/ui/window.go` | - Định nghĩa `struct AutoSwitchToggle` (implement `fyne.Widget`, `fyne.Tappable`, `desktop.Hoverable`, `desktop.Cursorable`).<br>- Hàm khởi tạo `newAutoSwitchToggle(initial bool, onChange func(bool), palette ThemePalette)` dựng track 38x20px (`CornerRadius = 10`) và thumb 14x14px (`canvas.NewCircle(color.White)`).<br>- Tọa độ toán học: OFF tại $(3, 3)$ với nền `BorderSubtle`, ON tại $(21, 3)$ với nền `Primary`.<br>- `autoSwitchToggleRenderer` cố định hình học chuẩn xác, render mượt mà không bị méo lệch.<br>- Trong `setupLayout()`: Tạo nhãn `lblAuto := canvas.NewText("Tự động:", palette.Text)` 10pt Bold, ghép vào `autoCluster := container.NewHBox(lblAuto, a.autoSwitchToggle)`, sau đó ghép vào cụm phải `rightCluster := container.NewHBox(autoCluster, themeSelectWrap)` và đưa vào `row2Combined`. |
| **Unit Test Suite Conformance** | `internal/ui/ui_test.go` | - Cập nhật `TestArchitectSpec_MergeToolbarWithServerStatusRow_FullConformance` để hỗ trợ tìm `themeSelectWrap` khi nằm trong `rightCluster`.<br>- Bổ sung test case toàn diện `TestArchitectSpec_AutoSwitchToggle_CapsuleWidget_FullConformance`:<br>  + Kiểm tra sự hiện diện của `autoSwitchToggle` và nhãn `Tự động:` 10pt Bold trong `rightCluster`.<br>  + Kiểm tra kích thước hình học chuẩn: MinSize 38x20px, track CornerRadius 10.0, track size 38x20px, thumb size 14x14px.<br>  + Kiểm tra trạng thái mặc định: OFF, tọa độ $(3, 3)$, màu nền `BorderSubtle`.<br>  + Kiểm tra con trỏ chuột: `desktop.PointerCursor`.<br>  + Kiểm tra tương tác Tap: chuyển ON, tọa độ $(21, 3)$, màu nền `Primary`.<br>  + Kiểm tra Tap lần 2: chuyển OFF, tọa độ $(3, 3)$, màu nền `BorderSubtle`.<br>  + Kiểm tra AC-4 (No Backend Side Effects): Manager active account không thay đổi khi bấm nút.<br>  + Kiểm tra điều khiển lập trình qua `SetAutoSwitchEnabled`.<br>  + Kiểm tra đổi Theme: ThemeLight, ThemeWarm, ThemeDark cập nhật màu track và label chính xác.<br>  + Kiểm tra bảo toàn kích thước cửa sổ 920x630px và thẻ card 280x130px. |

---

## 3. Ranh Giới Touch-Map & Non-Goals

- **Touch-Map Tuân Thủ 100%:** Chỉ chạm đúng 3 file mã nguồn và 1 file báo cáo:
  - `internal/ui/app.go`
  - `internal/ui/window.go`
  - `internal/ui/ui_test.go`
  - `reports/coder/2026-10-20-auto-switch-toggle-button-report.md`
- **NO ANVIEN:** Hoàn toàn không sử dụng Anvien.
- **Không backend side-effects:** Không gọi bất kỳ hàm switch account hay daemon hook nào khi click công tắc.
- **Không sửa script build:** `scripts/build_windows.ps1` nguyên vẹn 100%.

---

## 4. Bảng Đối Chiếu Tiêu Chí Nghiệm Thu (Acceptance Criteria)

| Tiêu Chí Nghiệm Thu | Kết Quả Thực Tế | Trạng Thái |
|---|---|---|
| **AC-1 (UI Toggle Only):** Công tắc gạt dạng viên thuốc (Capsule Pill Toggle) xuất hiện bên cạnh nút đổi Theme với nhãn `"Tự động:"` | Widget `AutoSwitchToggle` và `lblAuto` xuất hiện trong `rightCluster` ngay bên trái `themeSelectWrap` trên Hàng 2. | **PASS** |
| **AC-2 (Visual Toggle Interaction):** Click vào công tắc làm trượt nút tròn qua lại (OFF $\leftrightarrow$ ON), đổi màu nền ray (Xám $\leftrightarrow$ Xanh `Primary`) | Click làm đảo trạng thái `isOn`, ray đổi giữa `BorderSubtle` và `Primary`, thumb trượt giữa $(3, 3)$ và $(21, 3)$. | **PASS** |
| **AC-3 (Pointer Cursor):** Con trỏ chuột chuyển thành hình bàn tay (`desktop.PointerCursor`) khi rê chuột vào công tắc | Implement `desktop.Cursorable` trả về `desktop.PointerCursor`. | **PASS** |
| **AC-4 (No Backend Side Effects):** Bấm công tắc không kích hoạt bất kỳ luồng chuyển tài khoản nào | Chỉ thay đổi `a.autoSwitchEnabled` trong bộ nhớ UI, active account và manager store hoàn toàn không bị ảnh hưởng. | **PASS** |
| **AC-5 (Geometry & Test Suite):** Cửa sổ giữ nguyên `920 x 630 px`, thẻ Card `280 x 130 px`; `go test -v ./internal/ui` đạt 100% PASS | Cửa sổ cố định 920x630px, thẻ card cố định 280x130px, toàn bộ test suite `internal/ui` đạt 100% PASS. | **PASS** |

---

## 5. Bằng Chứng Biên Dịch & Kiểm Thử (Full Build Evidence)

### 5.1. Unit Test Suite Execution
Lệnh chạy: `go test -v ./internal/ui -run "TestArchitectSpec_AutoSwitchToggle_CapsuleWidget_FullConformance|TestArchitectSpec_MergeToolbarWithServerStatusRow_FullConformance"`

```text
=== RUN   TestArchitectSpec_MergeToolbarWithServerStatusRow_FullConformance
--- PASS: TestArchitectSpec_MergeToolbarWithServerStatusRow_FullConformance (0.68s)
=== RUN   TestArchitectSpec_AutoSwitchToggle_CapsuleWidget_FullConformance
--- PASS: TestArchitectSpec_AutoSwitchToggle_CapsuleWidget_FullConformance (0.39s)
PASS
ok  	antigravity-switch/internal/ui	2.451s
```

Lệnh chạy toàn bộ suite `internal/ui`: `go test ./internal/ui`
```text
ok  	antigravity-switch/internal/ui	17.023s
```

### 5.2. Full Windows Build Pipeline Execution
Lệnh chạy: `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all`

```text
================================================================================
⚡ Antigravity Multi-Account Switcher - Windows Build System
   Mode: all | OutDir: bin | SkipTests: False | Clean: False
================================================================================
[2/4] Running pre-flight tests & vet...
  -> Verifying pure Go packages (CGO_ENABLED=0)...
ok  	antigravity-switch/internal/model	(cached)
ok  	antigravity-switch/internal/wincred	(cached)
ok  	antigravity-switch/internal/store	4.491s
ok  	antigravity-switch/internal/scanner	(cached)
ok  	antigravity-switch/internal/connector	1.669s
ok  	antigravity-switch/internal/reloader	(cached)
ok  	antigravity-switch/internal/manager	5.938s
ok  	antigravity-switch/cmd/switcher	(cached)
  [PASS] Pure Go verification succeeded.
  -> Verifying GUI packages with CGo & Fyne (CGO_ENABLED=1)...
ok  	antigravity-switch/cmd/switcher	1.416s
ok  	antigravity-switch/internal/ui	11.059s
  [PASS] GUI verification succeeded.
[3/4] Compiling target binaries...
  -> Checking Windows application icon and resource...
  -> Compiling resource with windres: cmd\switcher\app.rc -> cmd\switcher\rsrc_windows_amd64.syso
  [SUCCESS] Compiled PE resource object: cmd\switcher\rsrc_windows_amd64.syso (12552 bytes)
  -> Building Standalone CLI Binary (Pure Go, CGO_ENABLED=0)...
  [SUCCESS] Created: bin\antigravity-switch.exe
  Compile Duration: 2727 ms | Size: 10.51 MB (11023872 bytes)
  -> Building Standalone GUI Binary (Fyne, CGO_ENABLED=1)...
  [SUCCESS] Created: bin\antigravity-switch-gui.exe
  Compile Duration: 5412 ms | Size: 27.62 MB (28957184 bytes)
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

## 6. Kết Luận & Bàn Giao

Hạng mục thi công UI Auto Switch Toggle Capsule Widget kế bên Theme Selector trên Hàng 2 đã hoàn thành 100%, tuân thủ tuyệt đối đặc tả kiến trúc `reports/architect/2026-10-20-auto-switch-toggle-button-spec.md`. Toàn bộ 5 tiêu chí nghiệm thu đạt PASS, hệ thống đạt trạng thái sẵn sàng cho Supervisor nghiệm thu.
