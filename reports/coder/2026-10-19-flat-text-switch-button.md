# Coder Implementation Report: Chuyển Nút "Chuyển" Sang Dạng Text In Đậm Màu Xanh Dương ([Chuyển]) Đồng Bộ Tuyệt Đối Với [✔ Dùng]

- **Date:** 2026-10-19
- **Role:** Coder
- **Authority / SSOT:**
  - `reports/architect/2026-10-19-flat-text-switch-button-spec.md` (Commit `d449e5d`, Phê duyệt chính thức bởi Human Owner)
  - Mã văn kiện đặc tả: `SPEC-ARCH-20261019-FLAT-TEXT-SWITCH-BUTTON`
  - AGENTS.md & working-rules
- **Scope (Touch-Map Khóa Cứng):**
  - `internal/ui/widgets.go`
  - `internal/ui/ui_test.go`
  - `reports/coder/2026-10-19-flat-text-switch-button.md`

---

## 1. Mục Tiêu & Tuân Thủ Kỷ Luật Sắt (Iron Rules Compliance)

1. **Loại bỏ hoàn toàn khối hộp nền xanh đặc dày cộm (No Solid Block Button):**
   - Loại bỏ `widget.NewButton("Chuyển", ...)` có `Importance = widget.HighImportance` cùng việc bọc bởi `compactButton` trong thẻ Inactive.
   - Thay thế bằng custom widget `switchTextButton` kế thừa `widget.BaseWidget`, hiển thị text phẳng `"[Chuyển]"` với `TextSize: 10`, `TextStyle: fyne.TextStyle{Bold: true}`, màu sắc thương hiệu `palette.Primary`.
   - Nền mặc định hoàn toàn trong suốt (`color.Transparent`), không còn khối hộp thô ráp, tối ưu không gian thị giác và mang lại diện mạo sang trọng, tối giản.
2. **Đồng bộ phong cách tuyệt đối với nhãn `[✔ Dùng]`:**
   - Thẻ Đang dùng (Active): Hiển thị nhãn `[✔ Dùng]` (10pt Bold).
   - Thẻ Chưa dùng (Inactive): Hiển thị nút chữ `[Chuyển]` (10pt Bold, màu xanh `palette.Primary`).
   - Cả hai phần tử dùng chung cấu trúc padding và căn lề, có chiều cao ~20-22px đồng bộ thẳng hàng hoàn hảo với nhãn email và nút icon `[🔄]` (được bọc bởi `compactButton`).
3. **Trải nghiệm tương tác cao cấp (Affordance & Interaction):**
   - Implement `desktop.Cursorable`: `Cursor() desktop.Cursor` trả về `desktop.PointerCursor` (con trỏ hình bàn tay chỉ trỏ) khi chuột di chuyển vào vùng `[Chuyển]`.
   - Implement `desktop.Hoverable`: `MouseIn` và `MouseOut` phản hồi thị giác:
     + Chữ `"[Chuyển]"` chuyển sang màu sáng hơn (`lightenColor(palette.Primary, 35)`).
     + Nền xuất hiện lớp phủ bo góc siêu mờ (`CornerRadius: 3`, alpha `0x24`) tạo cảm giác xúc giác tinh tế.
     + Trả về trạng thái trong suốt ban đầu khi chuột rời đi.
   - Implement `fyne.Tappable`: Phương thức `Tapped(*fyne.PointEvent)` và trường `OnTapped func()` kích hoạt chính xác `onSwitch(item.Email)` 100% như ban đầu.
4. **Bảo tồn 100% thông số hình học & cơ chế in-place:**
   - Kích thước thẻ Card: Giữ nguyên tuyệt đối **`280 x 130 px`** trong `GridWrap`.
   - Cửa sổ ứng dụng: Giữ nguyên tuyệt đối **`960 x 630 px`**.
   - Bảng Quota 2 tầng: Giữ nguyên kích thước ma trận **`228 x 77 px`**.
   - Cơ chế in-place reactive update (Zero Jank / Zero Resizing) không phá hủy hay tái tạo Canvas tree.
5. **Kỷ luật sắt đối với build script:**
   - Script `scripts/build_windows.ps1` nguyên vẹn 100%, không bị sửa đổi bất kỳ ký tự nào.
   - Chạy full build bằng `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all` đạt 100% test PASS, exit code 0.
6. **Không sử dụng Anvien (NO ANVIEN):**
   - Tuân thủ tuyệt đối lệnh cấm Anvien đối với Coder.

---

## 2. Invariant Family Map & Chi Tiết Thi Công

| Invariant / Surface | Target File | Chi Tiết Thi Công Cụ Thể |
|---|---|---|
| **Flat Text Switch Button Widget (`switchTextButton`)** | `internal/ui/widgets.go` | - Khai báo struct `switchTextButton` nhúng `widget.BaseWidget`, quản lý `Text`, `OnTapped`, `hovered`, `palette`, `bg *canvas.Rectangle`, `text *canvas.Text`.<br>- Triển khai đầy đủ interface: `fyne.Widget`, `fyne.Tappable`, `desktop.Hoverable`, `desktop.Cursorable`.<br>- `CreateRenderer()` sử dụng `container.NewStack(b.bg, container.NewPadded(b.text))` bọc trong `widget.NewSimpleRenderer`.<br>- Hàm `Refresh()` cập nhật màu chữ (`lightenColor`) và màu nền (`0x24` alpha) khi `hovered`.<br>- `Cursor()` trả về `desktop.PointerCursor`. |
| **Card Action Cluster Alignment** | `internal/ui/widgets.go` | - Trong `NewAccountCardWidgetWithQuota`, nhánh `item.IsActive == false` khởi tạo `switchBtn := newSwitchTextButton(func() { onSwitch(item.Email) }, palette)`.<br>- Cụm `actionCluster` trở thành `container.NewHBox(switchBtn, compactRefresh)` đồng nhất với Active `container.NewHBox(activeBadge, compactRefresh)`. |
| **Test Suite Alignment & Conformance Verification** | `internal/ui/ui_test.go` | - Cập nhật `findSwitch` trong `TestOwnerRedesign_AccountCardThreeRows` tìm `*switchTextButton` và kích hoạt `OnTapped()`.<br>- Cập nhật `findSwitchButton` tìm `*switchTextButton` và kiểm tra sự kiện chuyển đổi trong `TestArchitectSpec_CompactSwitchButtonHeight70Pct_FullConformance`.<br>- Bổ sung test suite toàn diện `TestArchitectSpec_FlatTextSwitchButton_FullConformance` kiểm tra: AC-1 (Text style, Bold, 10pt, `palette.Primary`), AC-2 (`PointerCursor`, hover highlight & restore), AC-3 (Tapped & `OnTapped`), AC-4 (Kích thước thẻ <= 280x130px), AC-5 (Đa theme: Dark, Light, Warm). |

---

## 3. Bảng Đối Chiếu Tiêu Chí Nghiệm Thu (Acceptance Criteria)

| Tiêu Chí Nghiệm Thu | Kết Quả Thực Tế | Trạng Thái |
|---|---|---|
| **AC-1 (Flat Text Style)** | Nút `[Chuyển]` không còn nền khối hộp xanh đặc. Hiển thị nhãn `"[Chuyển]"` phẳng phiu, in đậm 10pt, màu `palette.Primary`, nền trong suốt lúc bình thường, đồng bộ phong cách nhãn `[✔ Dùng]`. | **PASS** |
| **AC-2 (Pointer Cursor & Hover)** | Khi hover: `Cursor()` trả về `desktop.PointerCursor`, nền nhận hiệu ứng phủ mờ bo góc (`0x24` alpha), chữ `"[Chuyển]"` sáng màu hơn (`lightenColor`), trở lại bình thường khi rời chuột. | **PASS** |
| **AC-3 (Switch Functionality)** | Cả hai cơ chế gọi `Tapped(&fyne.PointEvent{})` và `OnTapped()` đều kích hoạt hàm callback `onSwitch(email)` chính xác 100%. | **PASS** |
| **AC-4 (Dimensions Stability)** | Giữ nguyên thẻ Card `280 x 130 px`, cửa sổ `960 x 630 px`, bảng Quota `228 x 77 px`. Dòng TopRow căn chỉnh ngang bằng phẳng hoàn hảo. | **PASS** |
| **AC-5 (Test Suite)** | `go test -v ./internal/ui` đạt 100% PASS trên tất cả các ca kiểm thử (bao gồm unit tests, reactive in-place updates, theme tests). | **PASS** |

---

## 4. Bằng Chứng Kiểm Thử & Biên Dịch (Verification Evidence)

### 4.1. Unit Test Package `internal/ui`
```text
=== RUN   TestArchitectSpec_FlatTextSwitchButton_FullConformance
--- PASS: TestArchitectSpec_FlatTextSwitchButton_FullConformance (0.03s)
=== RUN   TestArchitectSpec_CompactSwitchButtonHeight70Pct_FullConformance
--- PASS: TestArchitectSpec_CompactSwitchButtonHeight70Pct_FullConformance (0.02s)
=== RUN   TestOwnerRedesign_AccountCardThreeRows
--- PASS: TestOwnerRedesign_AccountCardThreeRows (0.00s)
PASS
ok  	antigravity-switch/internal/ui	9.997s
```

### 4.2. Full Build Bằng Script Của Repo (`scripts/build_windows.ps1 -Mode all`)
Lệnh thực thi: `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all`
Exit Code: `0`

```text
================================================================================
⚡ Antigravity Multi-Account Switcher - Windows Build System
   Mode: all | OutDir: bin | SkipTests: False | Clean: False
================================================================================
[2/4] Running pre-flight tests & vet...
  -> Verifying pure Go packages (CGO_ENABLED=0)...
ok  	antigravity-switch/internal/model	(cached)
ok  	antigravity-switch/internal/wincred	(cached)
ok  	antigravity-switch/internal/store	4.407s
ok  	antigravity-switch/internal/scanner	(cached)
ok  	antigravity-switch/internal/connector	(cached)
ok  	antigravity-switch/internal/reloader	(cached)
ok  	antigravity-switch/internal/manager	6.196s
ok  	antigravity-switch/cmd/switcher	(cached)
  [PASS] Pure Go verification succeeded.
  -> Verifying GUI packages with CGo & Fyne (CGO_ENABLED=1)...
ok  	antigravity-switch/cmd/switcher	1.334s
ok  	antigravity-switch/internal/ui	8.958s
  [PASS] GUI verification succeeded.
[3/4] Compiling target binaries...
  -> Checking Windows application icon and resource...
  -> Compiling resource with windres: cmd\switcher\app.rc -> cmd\switcher\rsrc_windows_amd64.syso
  [SUCCESS] Compiled PE resource object: cmd\switcher\rsrc_windows_amd64.syso (12552 bytes)
  -> Building Standalone CLI Binary (Pure Go, CGO_ENABLED=0)...
  [SUCCESS] Created: bin\antigravity-switch.exe
  Compile Duration: 3004 ms | Size: 10.51 MB (11023872 bytes)
  -> Building Standalone GUI Binary (Fyne, CGO_ENABLED=1)...
  [SUCCESS] Created: bin\antigravity-switch-gui.exe
  Compile Duration: 6023 ms | Size: 27.59 MB (28926464 bytes)
[4/4] Executing quick verification...
  [PASS] Embedded PE application icon verified on bin\antigravity-switch.exe (32x32)
  [PASS] Embedded PE application icon verified on bin\antigravity-switch-gui.exe (32x32)
  -> Running 'bin\antigravity-switch.exe --status'...
================================================================================
⚡ ANTIGRAVITY SESSION STATUS
================================================================================
Active Email:    quantritaikhoanemail@gmail.com
Language Server: Running (PID: 14880 | Port: 50222 | CSRF: Present)
Win32 Credential: Present (260 bytes access token)
================================================================================
================================================================================
⚡ BUILD SUCCESSFUL
================================================================================
```

---

## 5. Kết Luận
- **Trạng thái:** HOÀN THÀNH 100% (READY_FOR_SUPERVISOR).
- Toàn bộ thay đổi nằm chính xác trong Touch-Map đã khóa cứng (`internal/ui/widgets.go`, `internal/ui/ui_test.go`, `reports/coder/2026-10-19-flat-text-switch-button.md`).
- Script build `scripts/build_windows.ps1` nguyên vẹn 100%, không bị sửa đổi.
- Không phát sinh lỗi hồi quy, sẵn sàng bàn giao cho Supervisor nghiệm thu độc lập Zero-Trust.
