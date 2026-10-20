# Coder Implementation Report: Nắn Thẳng Hàng Trục Dọc Cụm Công Tắc & Dropdown Theme (Header Row 2 Vertical Alignment)

- **Date:** 2026-10-20
- **Role:** Coder
- **Authority / SSOT:**
  - Chỉ đạo trực tiếp từ Human Owner qua hình ảnh minh chứng `reports/problem/images/screenshot_1792479866.png`: *"gạt on/off tại sao lại nhảy len trên nhu thế? lệch hàng. - viét đặc tả gui CEO"*
  - `reports/architect/2026-10-20-align-header-row2-vertical-baseline-spec.md` (`SPEC-ARCH-20261020-ALIGN-HEADER-ROW2-VERTICAL-BASELINE`)
  - Base Commit: `24f7087`
  - `AGENTS.md` & `working-rules`
- **Scope (Touch-Map Khóa Cứng):**
  - `internal/ui/window.go`:
    1. Chuẩn hóa `autoSwitchToggleRenderer.Layout` tính toán tọa độ dọc động $y = (\text{size.Height} - 12.0) / 2.0$ và $thumbY = y + 2.0$, triệt tiêu việc dính cứng ở $y = 0$.
    2. Đổi nhãn `"Tự động:"` (`lblAuto`) từ `*canvas.Text` sang `*widget.Label` với `fyne.TextAlignLeading` và `fyne.TextStyle{Bold: true}`, dùng chung hệ quy chiếu padding và đường chân chữ với `serverStatusLabel`.
    3. Triển khai custom layout `row2RightClusterLayout` căn giữa dọc toàn bộ các phần tử trong `rightCluster` (`lblAuto`, `a.autoSwitchToggle`, `themeSelectWrap`).
  - `internal/ui/ui_test.go`:
    1. Cập nhật `TestArchitectSpec_AutoSwitchToggle_CapsuleWidget_FullConformance` tương thích hợp đồng `widget.Label` mới.
    2. Bổ sung `TestHeaderRow2_VerticalAlignment` kiểm thử độ lệch tâm trục dọc giữa cụm bên trái (`serverStatusLabel`) và cụm bên phải (`lblAuto`, `autoSwitchToggle`, `themeSelectWrap`) $\le 2.0\text{ px}$.
  - `reports/coder/2026-10-20-align-header-row2-vertical-baseline-report.md`: Báo cáo nghiệm thu chi tiết.

---

## 1. Mục Tiêu & Tuân Thủ Kỷ Luật Sắt

1. **Triệt tiêu lỗi lệch hàng (Zero Vertical Misalignment - AC-1):**
   - Trước khi sửa: Cụm bên phải (`lblAuto`, `autoSwitchToggle`, `themeSelectWrap`) bị kéo dính sát nóc trần ($y = 0$), trong khi cụm bên trái (`serverStatusLabel` + 3 nút Toolbar) nằm ở tâm hàng ($y \approx 6-8\text{ px}$), gây ra tình trạng công tắc gạt nhảy chồm lên trên.
   - Sau khi sửa: Toàn bộ các phần tử trên Hàng 2 (`row2Combined`) nằm trên cùng một đường tâm ngang ($y_{center} = \text{rowHeight} / 2$). Độ lệch tâm dọc $|Y_{left} - Y_{right}| \le 2.0\text{ px}$ (thực tế $\approx 0.0\text{ px}$).
2. **Đồng nhất widget type cho text labels (AC-2):**
   - Nhãn `"Tự động:"` được chuyển từ `*canvas.Text` sang `*widget.Label` (`widget.NewLabelWithStyle("Tự động:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})`).
   - Nhãn tự động thừa hưởng cơ chế theme, padding và text baseline của Fyne widget, hoàn toàn tương thích và cân xứng tuyệt đối với `a.serverStatusLabel`.
3. **Căn giữa động cho AutoSwitchToggle (AC-3):**
   - Trong `autoSwitchToggleRenderer.Layout(size)`: Tọa độ $y$ của track được tính động:
     $$y = \max\left(0, \frac{\text{size.Height} - 12.0}{2.0}\right)$$
     $$thumbY = y + 2.0$$
   - Khi kích thước phân bổ cho widget lớn hơn chiều cao tối thiểu 12px, công tắc tự động căn giữa mà không bao giờ bị dính sát mép trên.
   - Trong `updateVisuals()`: Con trượt thumb cũng tính toán $thumbY = y + 2.0$, đảm bảo tính nhất quán tuyệt đối giữa trạng thái hiển thị và layout pass.
4. **Bảo tồn kích thước hệ thống đã khóa cứng (AC-4):**
   - Cửa sổ ứng dụng: CỐ ĐỊNH **`920 x 630 px`**.
   - Thẻ Card: CỐ ĐỊNH **`280 x 130 px`**.
   - Bảng Quota: CỐ ĐỊNH **`228 x 77 px`**.
5. **Cấm Anvien:**
   - Hoàn toàn không sử dụng công cụ anvien trong toàn bộ quá trình thi công.
6. **Full build bằng script repo nguyên bản (AC-5):**
   - `scripts/build_windows.ps1` nguyên vẹn 100%.
   - Chạy `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all` đạt Exit Code 0, build thành công cả CLI lẫn GUI.

---

## 2. Invariant Family Map & Chi Tiết Thi Công

| Thành Phần | Trạng Thái Trước | Trạng Thái Mới (Theo SPEC-ARCH-20261020-ALIGN-HEADER-ROW2-VERTICAL-BASELINE) | File Sửa Đổi |
|---|---|---|---|
| **`lblAuto`** | `canvas.NewText("Tự động:", palette.Text)` (Không có widget padding) | `widget.NewLabelWithStyle("Tự động:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})` | `internal/ui/window.go` |
| **`autoSwitchToggleRenderer.Layout`** | Hardcode `track.Move(fyne.NewPos(0, 0))` | Tính động $y = (size.Height - 12.0) / 2.0$, $thumbY = y + 2.0$ | `internal/ui/window.go` |
| **`rightCluster` Layout** | `container.NewHBox(autoCluster, themeSelectWrap)` (Bắt đầu từ $y=0$, lệch baseline) | `container.New(row2RightClusterLayout{}, lblAuto, a.autoSwitchToggle, themeSelectWrap)` | `internal/ui/window.go` |
| **`row2RightClusterLayout`** | Chưa có | Custom layout căn giữa dọc: $y = (size.Height - ms.Height) / 2.0$ cho từng phần tử | `internal/ui/window.go` |
| **Test Suite** | Chưa kiểm tra vertical baseline alignment | Bổ sung `TestHeaderRow2_VerticalAlignment` (khẳng định $|Y_{left} - Y_{right}| \le 2.0\text{ px}$) | `internal/ui/ui_test.go` |

### Ghi Chú Kỹ Thuật (Architecture & Scope Integrity):
- `internal/ui/app.go` chứa trường private `autoSwitchLabel *canvas.Text`. Do `app.go` nằm ngoài Touch-Map khóa cứng, coder giữ nguyên vẹn `app.go` theo đúng kỷ luật sắt. Nhãn mới `lblAuto` được lưu trữ an toàn trong package `ui` và cung cấp qua accessor `(a *App) AutoSwitchLabelWidget() *widget.Label` trong `window.go`. Nhãn `widget.Label` tự động làm mới theo theme của Fyne mà không cần can thiệp thủ công.
- Không có bất kỳ file nào ngoài Touch-Map (`window.go`, `ui_test.go`) bị sửa đổi.

---

## 3. Kết Quả Kiểm Thử & Nghiệm Thu (Evidence)

### 3.1. Unit Test Suite (`go test -v ./internal/ui`)
```text
=== RUN   TestArchitectSpec_AutoSwitchToggle_CapsuleWidget_FullConformance
--- PASS: TestArchitectSpec_AutoSwitchToggle_CapsuleWidget_FullConformance (0.51s)
=== RUN   TestHeaderRow2_VerticalAlignment
--- PASS: TestHeaderRow2_VerticalAlignment (0.14s)
=== RUN   TestArchitectSpec_CompactThemeSelect60Pct_FullConformance
--- PASS: TestArchitectSpec_CompactThemeSelect60Pct_FullConformance (0.38s)
...
PASS
ok  	antigravity-switch/internal/ui	10.949s
```
Toàn bộ test suite của `internal/ui` đạt 100% PASS.

### 3.2. Full Build Windows Script (`build_windows.ps1 -Mode all`)
```text
================================================================================
⚡ Antigravity Multi-Account Switcher - Windows Build System
   Mode: all | OutDir: bin | SkipTests: False | Clean: False
================================================================================
[2/4] Running pre-flight tests & vet...
  -> Verifying pure Go packages (CGO_ENABLED=0)...
  [PASS] Pure Go verification succeeded.
  -> Verifying GUI packages with CGo & Fyne (CGO_ENABLED=1)...
ok  	antigravity-switch/cmd/switcher	(cached)
ok  	antigravity-switch/internal/ui	10.077s
  [PASS] GUI verification succeeded.
[3/4] Compiling target binaries...
  -> Checking Windows application icon and resource...
  -> Compiling resource with windres: cmd\switcher\app.rc -> cmd\switcher\rsrc_windows_amd64.syso
  [SUCCESS] Compiled PE resource object: cmd\switcher\rsrc_windows_amd64.syso (12552 bytes)
  -> Building Standalone CLI Binary (Pure Go, CGO_ENABLED=0)...
  [SUCCESS] Created: bin\antigravity-switch.exe
  Compile Duration: 2972 ms | Size: 10.51 MB (11023872 bytes)
  -> Building Standalone GUI Binary (Fyne, CGO_ENABLED=1)...
  [SUCCESS] Created: bin\antigravity-switch-gui.exe
  Compile Duration: 6007 ms | Size: 27.62 MB (28959232 bytes)
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
Exit Code: 0 (BUILD SUCCESSFUL).

---

## 4. Kết Luận & Bàn Giao
- Mọi tiêu chí nghiệm thu (AC-1 đến AC-5) trong văn kiện kiến trúc `SPEC-ARCH-20261020-ALIGN-HEADER-ROW2-VERTICAL-BASELINE` đã được hiện thực hóa và kiểm chứng đầy đủ.
- Lỗi lệch hàng trục dọc (công tắc nhảy lên trần) đã bị triệt tiêu 100%.
- Cụm điều khiển Toolbar Header Row 2 đạt độ hoàn thiện cao, cân đối, chính xác từng pixel và sẵn sàng chuyển giao cho Supervisor nghiệm thu.
