# Báo Cáo Thi Công: Phase 1 - Tầng Giao Diện Thẻ (Account Status UI Widgets)
(Account Status On/Off Toggle & Card UI Widgets Implementation Report - Phase 1)

- **Ngày thực hiện:** 2026-10-21
- **Vai trò:** Coder
- **Thẩm quyền kiến trúc (Authority / SSOT):** `reports/architect/2026-10-21-account-status-on-off-and-auto-stop-spec.md` (`SPEC-ARCH-20261021-ACCOUNT-STATUS-ON-OFF-AUTO-STOP`, commit `64a2a36`)
- **Trạng thái:** HOÀN THÀNH (READY_FOR_SUPERVISOR)

---

## I. TỔNG QUAN THI CÔNG & INVARIANT FAMILY MAP

### 1. Invariant Family Map
- **Family Name:** `Account Card Status Toggle & Auto-Stop Safety (Phase 1: UI Widgets)`
- **Authority / SSOT:** `reports/architect/2026-10-21-account-status-on-off-and-auto-stop-spec.md` (Mã `SPEC-ARCH-20261021-ACCOUNT-STATUS-ON-OFF-AUTO-STOP`, commit `64a2a36`)
- **Sibling Surfaces Inspected & Closed:**
  1. `AccountDisplayItem` tại `internal/manager/account_manager.go`:
     - Bổ sung trường `Status string \`json:"status,omitempty"\`` bảo đảm tính tương thích kiểu dữ liệu và serialization khi biên dịch, chuẩn bị cho Phase 2.
  2. `switchTextButton` tại `internal/ui/widgets.go`:
     - Mở rộng cờ `disabled bool`.
     - Bổ sung hàm dựng `newDisabledSwitchTextButton(palette ThemePalette) *switchTextButton`.
     - Khi `disabled == true`: nhãn mang màu `palette.TextMuted`, không có hiệu ứng hover, con trỏ chuột là `desktop.DefaultCursor`, click hoàn toàn no-op (`Tapped` return ngay).
  3. `statusTextButton` tại `internal/ui/widgets.go`:
     - Xây dựng widget phẳng độc lập: `Status string` ("on" hoặc "off"), `OnTapped func()`.
     - Nhãn `"[ON]"` hiển thị màu `palette.Success` (xanh lá), in đậm `Bold: true`, cỡ chữ 10, căn giữa.
     - Nhãn `"[OFF]"` hiển thị màu `palette.TextMuted` (xám mờ), in đậm `Bold: true`, cỡ chữ 10, căn giữa.
     - Hiệu ứng chuột (Hover): nền sáng nhẹ 14% (`A: 0x24`), văn bản sáng nhẹ qua hàm `lightenColor(..., 35)`.
     - Con trỏ chuột: `desktop.PointerCursor`.
  4. `AccountCardWidget` tại `internal/ui/widgets.go`:
     - Mở rộng các trường: `Status string`, `statusButton *statusTextButton`, `onToggleStatus func(email string)`.
     - Bổ sung các phương thức hỗ trợ kiểm thử và truy xuất: `StatusButton()`, `ActionButton()`, `RefreshButton()`.
  5. `NewAccountCardWidgetWithQuota` & `NewAccountCardWithQuota` tại `internal/ui/widgets.go`:
     - Hỗ trợ tham số variadic `onToggleStatus ...func(email string)` bảo đảm tương thích 100% với tất cả các vị trí gọi hiện hữu trong `internal/ui/app.go`, `two_store_test.go`, `ui_test.go`, `window_native_test.go` (0 lỗi biên dịch, 0 lines diff ở các file gọi cũ).
     - Bố cục cụm hành động `actionCluster`: `container.NewHBox(actionSlot, compactButton(refreshBtn), statusBtn)` — đặt nút `statusBtn` nằm ở bên phải nút làm mới `[🔄]`.
  6. `UpdateState(item, palette)` tại `internal/ui/widgets.go`:
     - Cập nhật `w.Status = item.Status` (chuẩn hóa mặc định là `"on"` nếu rỗng hoặc khác `"off"`).
     - **Thẻ `[Cần OAuth]` (`!item.HasToken && !item.IsActive`):** Gọi `w.statusButton.Hide()` — tuyệt đối không hiển thị nút `[ON/OFF]` trên thẻ chưa có credential.
     - **Thẻ có credential hoặc đang Active:** Gọi `w.statusButton.Show()`, hiển thị nhãn `[ON]` (xanh lá) hoặc `[OFF]` (xám) tương ứng.
     - **Khi `w.Status == "off"`:** Nút `[Chuyển]` được khởi tạo ở trạng thái làm mờ (`newDisabledSwitchTextButton`, màu chữ `palette.TextMuted`, con trỏ chuột `desktop.DefaultCursor`, click không có tác dụng), nút `[🔄]` bị vô hiệu hóa (`w.refreshButton.Disable()`).
     - **Khi `w.Status == "on"`:** Nút `[Chuyển]` hoạt động bình thường (`palette.Primary`, con trỏ `desktop.PointerCursor`, click gọi `w.onSwitch`), nút `[🔄]` được kích hoạt bình thường (`w.refreshButton.Enable()`).
  7. `internal/ui/widgets_test.go`:
     - Bổ sung 6 bài kiểm thử toàn diện cho các hành vi widget, hover, cursor, disable, chuyển đổi trạng thái thẻ và ẩn/hiện nút ON/OFF.
- **Legacy Fallback Status:** Tương thích ngược 100% với các thẻ hiện hữu; nếu chưa có giá trị `Status` hoặc để rỗng thì mặc định luôn là `"on"`.
- **Residual Unverified Surfaces:** `none`.

---

## II. CHI TIẾT CÁC TỆP SỬA ĐỔI (FILES CHANGED)

1. `internal/manager/account_manager.go`:
   - Bổ sung trường `Status string \`json:"status,omitempty"\`` vào struct `AccountDisplayItem`.
2. `internal/ui/widgets.go`:
   - Mở rộng struct `switchTextButton` với trường `disabled bool`, bổ sung `newDisabledSwitchTextButton`, cập nhật xử lý `Refresh`, `Tapped`, `MouseIn`, `Cursor`, `IsDisabled`.
   - Hiện thực struct `statusTextButton` kèm các phương thức `newStatusTextButton`, `CreateRenderer`, `Refresh`, `SetStatus`, `SetPalette`, `Tapped`, `MouseIn`, `MouseOut`, `Cursor`, `LabelText`.
   - Mở rộng struct `AccountCardWidget` với `Status`, `statusButton`, `onToggleStatus`, và các getter `StatusButton()`, `ActionButton()`, `RefreshButton()`.
   - Cập nhật hàm `UpdateState` xử lý ẩn/hiện `statusButton`, làm mờ nút `[Chuyển]` và vô hiệu hóa nút `[🔄]` khi `w.Status == "off"`.
   - Cập nhật `NewAccountCardWidgetWithQuota` và `NewAccountCardWithQuota` nhận variadic `onToggleStatus ...func(email string)` và bố trí `statusBtn` bên phải nút `refreshBtn`.
3. `internal/ui/widgets_test.go` (Tệp mới):
   - `TestStatusTextButton_RenderingAndHover`: Kiểm tra hiển thị `[ON]`, `[OFF]`, màu sắc, hover 14%, con trỏ pointer và callback click.
   - `TestSwitchTextButton_DisabledState`: Kiểm tra trạng thái disabled của nút chuyển, màu `TextMuted`, con trỏ mặc định, click no-op, không kích hoạt hover.
   - `TestAccountCardWidget_StatusButton_DefaultOnAndToggle`: Kiểm tra khởi tạo mặc định `"on"`, hiển thị `[ON]`, nút `[Chuyển]` và `[🔄]` hoạt động, click toggle status.
   - `TestAccountCardWidget_StatusOff_MutesSwitchAndDisablesRefresh`: Kiểm tra thẻ mang `Status: "off"`, nhãn `[OFF]`, nút `[Chuyển]` bị mờ và click no-op, nút `[🔄]` bị vô hiệu hóa.
   - `TestAccountCardWidget_UnauthAccount_HidesStatusButton`: Kiểm tra thẻ `[Cần OAuth]` ẩn hoàn toàn nút `[ON/OFF]` (`Visible() == false`).
   - `TestAccountCardWidget_ActiveAccount_ShowsStatusButton`: Kiểm tra tài khoản Active luôn hiển thị nút `[ON/OFF]`, khi tắt thì nút làm mới bị vô hiệu hóa.
   - `TestAccountCardWidget_TransitionFromOnToOffAndBack`: Kiểm tra chu trình chuyển đổi mượt mà `on -> off -> on` mà không gây lỗi hoặc rò rỉ trạng thái.

---

## III. KẾT QUẢ KIỂM CHỨNG & BẰNG CHỨNG THỰC NGHIỆM (E2E VERIFICATION)

### 1. Kiểm thử tự động (Unit Tests)
- **Lệnh thực thi kiểm thử widget:** `go test -v ./internal/ui/... -run TestAccountCardWidget`
  - `TestAccountCardWidget_StatusButton_DefaultOnAndToggle` -> **PASS (0.01s)**
  - `TestAccountCardWidget_StatusOff_MutesSwitchAndDisablesRefresh` -> **PASS (0.01s)**
  - `TestAccountCardWidget_UnauthAccount_HidesStatusButton` -> **PASS (0.00s)**
  - `TestAccountCardWidget_ActiveAccount_ShowsStatusButton` -> **PASS (0.01s)**
  - `TestAccountCardWidget_TransitionFromOnToOffAndBack` -> **PASS (0.01s)**
  - Kết quả package: `ok antigravity-switch/internal/ui 1.566s` -> **PASS (100%)**

- **Lệnh thực thi toàn bộ package UI:** `go test ./internal/ui/...`
  - Kết quả: `ok antigravity-switch/internal/ui 18.470s` -> **PASS (100%)**

- **Lệnh thực thi toàn bộ test repository:** `go test ./...`
  - `cmd/switcher` -> **PASS**
  - `internal/automation` -> **PASS**
  - `internal/connector` -> **PASS**
  - `internal/manager` -> **PASS**
  - `internal/model` -> **PASS**
  - `internal/reloader` -> **PASS**
  - `internal/scanner` -> **PASS**
  - `internal/store` -> **PASS**
  - `internal/telemetry` -> **PASS**
  - `internal/ui` -> **PASS**
  - `internal/wincred` -> **PASS**
  - Kết quả: **PASS (100%)**

### 2. Biên dịch hệ thống bằng script của repo (Windows Build Verification)
- **Lệnh thực thi:** `pwsh -File .\scripts\build_windows.ps1 -Clean`
- **Kết quả:**
  ```text
  ================================================================================
  ⚡ Antigravity Multi-Account Switcher - Windows Build System
     OutDir: bin | SkipTests: False | Clean: True
  ================================================================================
  [1/4] Cleaning output artifacts...
    Removed directory: bin
    Clean complete.
  [2/4] Running pre-flight tests & vet...
    [PASS] Verification succeeded.
  [3/4] Compiling target binaries...
    -> Checking Windows application icon and resource...
    -> Compiling resource with windres: cmd\switcher\app.rc -> cmd\switcher\rsrc_windows_amd64.syso
    [SUCCESS] Compiled PE resource object: cmd\switcher\rsrc_windows_amd64.syso (12552 bytes)
    -> Building Standalone GUI Binary (Fyne, CGO_ENABLED=1)...
    [SUCCESS] Created: bin\antigravity-switch.exe
    Compile Duration: 6058 ms | Size: 27.76 MB (29106176 bytes)
  [4/4] Executing quick verification...
    [PASS] Embedded PE application icon verified on bin\antigravity-switch.exe (32x32)
    [INFO] GUI binary built at bin\antigravity-switch.exe.
  ================================================================================
  ⚡ BUILD SUCCESSFUL
  ================================================================================
  ```
- **Exit Code:** `0`
- **Diff của build script:** `scripts/build_windows.ps1` giữ nguyên 100% (**0 lines diff**).

---

## IV. ĐỐI SOÁT TIÊU CHÍ NGHIỆM THU THEO ĐẶC TẢ KIẾN TRÚC

| STT | Tiêu chí nghiệm thu Phase 1 | Kết quả | Bằng chứng đối soát |
|:---:|---|:---:|---|
| **1** | Bổ sung trường `Status string` vào `AccountDisplayItem` | **ĐẠT** | `internal/manager/account_manager.go` line 38, biên dịch toàn repo sạch sẽ. |
| **2** | Hiện thực `statusTextButton` phẳng, màu Success khi `[ON]`, TextMuted khi `[OFF]`, hover 14%, cursor Pointer | **ĐẠT** | `internal/ui/widgets.go`, kiểm chứng qua `TestStatusTextButton_RenderingAndHover`. |
| **3** | Bố trí nút `statusBtn` bên phải nút `[🔄]` trong `actionCluster` | **ĐẠT** | `container.NewHBox(actionSlot, compactButton(refreshBtn), statusBtn)`. |
| **4** | Thẻ `[Cần OAuth]` ẩn hoàn toàn nút `[ON/OFF]` | **ĐẠT** | `w.statusButton.Hide()` khi `!w.HasToken && !w.IsActive`, kiểm chứng qua `TestAccountCardWidget_UnauthAccount_HidesStatusButton`. |
| **5** | Khi `Status == "off"`, nút `[Chuyển]` mờ đi, cursor Default, click no-op; nút `[🔄]` bị vô hiệu hóa | **ĐẠT** | Kiểm chứng qua `TestAccountCardWidget_StatusOff_MutesSwitchAndDisablesRefresh`. |
| **6** | Khi `Status == "on"`, nút `[Chuyển]` và nút `[🔄]` hoạt động bình thường | **ĐẠT** | Kiểm chứng qua `TestAccountCardWidget_StatusButton_DefaultOnAndToggle`. |
| **7** | `go test ./...` đạt 100% pass | **ĐẠT** | Tất cả packages pass với Exit Code 0. |
| **8** | Biên dịch `pwsh -File .\scripts\build_windows.ps1 -Clean` thành công Exit Code 0 | **ĐẠT** | Build pass, tạo file `bin\antigravity-switch.exe` thành công. |
| **9** | Không xâm lấn các phase sau (Phase 2-5), không sửa script build | **ĐẠT** | Chỉ chạm đúng 3 files thuộc scope Phase 1, script build 0 lines diff. |
