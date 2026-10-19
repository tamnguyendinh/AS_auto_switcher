# Coder Implementation Report: UI Window Resize Stability, In-Place Reactive Quota Updates & Window Geometry Persistence

- **Date:** 2026-10-19
- **Role:** Coder
- **Authority / SSOT:**
  - `reports/architect/2026-10-19-ui-window-resize-stability-and-inplace-quota-binding-spec.md` (Approved by Human Owner)
  - `AGENTS.md` & `working-rules`
- **Goal:** Thi công khắc phục triệt để lỗi tự động co giãn giao diện, cập nhật Quota tại chỗ (In-Place Quota Binding) và lưu vết kích thước cửa sổ (Window Geometry Persistence) theo đúng báo cáo đặc tả kiến trúc:
  1. **INV-01 (In-Place Reactive Quota Updates):** Tuyệt đối không xóa cây Canvas (`Objects = nil`) trong chu kỳ 60s của `updateUI` khi danh sách tài khoản không thay đổi. Cập nhật trực tiếp số liệu % và thanh mini progress bar tại chỗ mà không gây re-layout giật khung hình.
  2. **INV-02 (Freedom of Window Resizing):** Hạ ngưỡng kích thước tối thiểu `MinWindowWidth` x `MinWindowHeight` từ `840x525` xuống `480x360` px. Đảm bảo người dùng kéo thu nhỏ tự do mà Fyne GLFW driver không tự ý ép cửa sổ bung to trở lại.
  3. **INV-03 (Window Geometry Persistence):** Lưu trữ kích thước cửa sổ (`WindowWidth`, `WindowHeight`) vào `accounts_store/.ui_preferences.json` (debounced 300ms) và khôi phục chính xác khi mở lại ứng dụng.
  4. **INV-04 (Elastic Matrix Quota Table):** Bảng Quota 3x3 co giãn đàn hồi hai cột 5h và Week theo tỷ lệ bề rộng thực tế của thẻ Card (`clamp(88.0, (cardWidth - fixedWidth)/2, 125.0)`).

---

## 1. Invariant Family Map

| Invariant | Target File | Implementation Details |
|---|---|---|
| **INV-01** | `internal/ui/widgets.go`<br>`internal/ui/quota_badge.go`<br>`internal/ui/app.go` | - Khai báo cấu trúc `AccountCardWidget` giữ tham chiếu trực tiếp đến `matrixQuotaCell`, `canvas.Text`, `canvas.Rectangle`, `emailLabel`, `cardBg`.<br>- Bổ sung phương thức `UpdateQuota(quota, palette)` cập nhật trực tiếp `pctText`, `fillBar.Resize()`, `fillBar.FillColor` và gọi `Refresh()` cục bộ trên cell con.<br>- Tái cấu trúc `updateUI()`: khi danh sách tài khoản không đổi (`canUpdateInPlace == true`), duyệt qua `a.accountCards[i].UpdateQuota(...)` mà hoàn toàn KHÔNG chạm vào `a.accountListContainer.Objects`, không gọi `a.accountListContainer.Refresh()`, không reset scroll view. |
| **INV-02** | `internal/ui/window.go` | - Định nghĩa các hằng số kích thước: `DefaultWindowWidth = 1008.0`, `DefaultWindowHeight = 630.0`, `MinWindowWidth = 480.0`, `MinWindowHeight = 360.0`.<br>- Cập nhật `minSizeLayout` trả về MinSize `480x360`. Khi cửa sổ bị thu nhỏ dưới ngưỡng cũ (840x525), GLFW driver nhận thấy kích thước vẫn >= 480x360 nên hoàn toàn không kích hoạt cờ `shouldExpand`, cửa sổ đứng yên tuyệt đối. |
| **INV-03** | `internal/ui/themes.go`<br>`internal/ui/app.go`<br>`internal/ui/window.go` | - Mở rộng struct `UIPreferences` với `WindowWidth float32` và `WindowHeight float32` (`json:",omitempty"`).<br>- Trong `minSizeLayout.Layout()`, đo kích thước canvas cửa sổ và chuyển tiếp sự kiện resize tới `a.app.OnWindowResized(w, h)`.<br>- Cơ chế Debounce 300ms trong `app.go`: ghi nhận kích thước và lưu trữ an toàn vào `.ui_preferences.json` khi người dùng kết thúc kéo chuột.<br>- Khôi phục kích thước tại `Init()`: đọc `WindowWidth` và `WindowHeight` đã lưu và áp dụng vào `a.win.Resize(fyne.NewSize(initW, initH))`. |
| **INV-04** | `internal/ui/quota_badge.go` | - Trong `matrixTableLayout.Layout()`, tính toán đàn hồi `colCycleW`: khi `remainW > 176.0`, phân bổ đều `colCycleW = math.Min(remainW / 2.0, 125.0)`.<br>- Hai cột 5h và Week tự động nở rộng cân đối khi kéo dãn cửa sổ mà vẫn bảo toàn typography tối đa 125px. |

---

## 2. Touch-Map Compliance Verification

Tuân thủ nghiêm ngặt ranh giới Touch-Map được giao, tuyệt đối không chỉnh sửa bất kỳ file nào ngoài phạm vi:

| STT | File Path | Status | Verification Summary |
|---|---|---|---|
| 1 | `internal/ui/themes.go` | Modified | Thêm `WindowWidth`, `WindowHeight` vào `UIPreferences` |
| 2 | `internal/ui/quota_badge.go` | Modified | Thêm `math`, cấu trúc `matrixQuotaCell`, hàm `createMatrixQuotaTableWithCells`, tính toán đàn hồi `colCycleW` |
| 3 | `internal/ui/widgets.go` | Modified | Cấu trúc `AccountCardWidget`, hàm `NewAccountCardWidgetWithQuota`, phương thức `UpdateQuota` in-place, giữ nguyên `NewAccountCardWithQuota` tương thích ngược |
| 4 | `internal/ui/window.go` | Modified | Khai báo hằng số `MinWindowWidth = 480`, `MinWindowHeight = 360`, gắn hook `m.app.OnWindowResized` |
| 5 | `internal/ui/app.go` | Modified | Quản lý `accountCards`, nạp/lưu debounced window geometry, `updateUI` in-place zero canvas churn, `RefreshSingleAccountQuota` in-place |
| 6 | `internal/ui/ui_test.go` | Modified | Cập nhật assertions cho 480x360 MinSize, bổ sung 4 test cases cho 4 Invariants |
| 7 | `reports/coder/2026-10-19-ui-window-resize-stability-and-inplace-quota.md` | Created | Báo cáo thi công chính thức |

**Non-goals Verified:**
- NO Anvien utilized.
- File `scripts/build_windows.ps1` giữ nguyên trạng 100% không bị can thiệp.
- Logic nghiệp vụ của `telemetryPool`, `QuotaClient` và backend switcher hoàn toàn không thay đổi.

---

## 3. Verification Evidence

### 3.1. Go Test Suite (`go test -tags gui -count=1 ./...`)
100% pass trên toàn bộ 10 packages của repository:
```text
ok  	antigravity-switch/cmd/switcher	2.693s
ok  	antigravity-switch/internal/connector	3.242s
ok  	antigravity-switch/internal/manager	10.033s
ok  	antigravity-switch/internal/model	0.927s
ok  	antigravity-switch/internal/reloader	3.588s
ok  	antigravity-switch/internal/scanner	1.047s
ok  	antigravity-switch/internal/store	7.668s
ok  	antigravity-switch/internal/telemetry	12.610s
ok  	antigravity-switch/internal/ui	12.581s
ok  	antigravity-switch/internal/wincred	0.832s
```

### 3.2. Dedicated Invariant Unit Tests
- `TestArchitectSpec_INV01_InPlaceReactiveQuotaUpdatesZeroCanvasChurn`: **PASS** (chứng minh pointer mọi phần tử trong `accountListContainer.Objects` giữ nguyên 100%, dữ liệu text và bar cập nhật đúng).
- `TestArchitectSpec_INV02_FreedomOfWindowResizing`: **PASS** (MinSize nội dung đạt đúng 480x360, cửa sổ co giãn tự do, `FixedSize == false`).
- `TestArchitectSpec_INV03_WindowGeometryPersistence`: **PASS** (lưu kích thước tùy chỉnh 768x490 vào `.ui_preferences.json`, mở lại app khôi phục đúng 768x490).
- `TestArchitectSpec_INV04_ElasticMatrixQuotaTableLayout`: **PASS** (đo đạc kích thước thực tế: 236px -> 88px, 300px -> 116px, 400px -> 125px).

### 3.3. Mandatory Full Build Script Evidence
Chạy lệnh bắt buộc của repo: `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all`
- Pre-flight pure Go tests: **PASS**
- GUI package tests (CGO_ENABLED=1, Fyne): **PASS**
- PE Resource windres compilation: **SUCCESS** (`cmd\switcher\rsrc_windows_amd64.syso`)
- CLI Binary compilation: **SUCCESS** (`bin\antigravity-switch.exe`, 10.51 MB)
- GUI Binary compilation: **SUCCESS** (`bin\antigravity-switch-gui.exe`, 27.6 MB)
- Embedded 32x32 PE icon verification: **PASS** on both binaries
- Quick runtime verification (`bin\antigravity-switch.exe --status`): **PASS** (Exit code 0)
- **BUILD STATUS: SUCCESSFUL (Exit Code 0)**

---

## 4. E2E Verification Flow

1. **Trigger:** Vòng lặp `autoLoop` chạy ngầm 60s quét Quota từ Google API, hoặc người dùng bấm nút [🔄] trên thẻ card.
2. **Process:**
   - Dữ liệu Quota mới được nạp vào RAM và ghi vào `QuotaCache`.
   - `updateUI` so sánh danh sách `overview.Accounts` với `accountCards`.
   - Nhận diện `canUpdateInPlace == true`, gọi trực tiếp `w.UpdateQuota(quotaData, palette)`.
   - Các Canvas primitives con (`pctText`, `fillBar`) cập nhật text và width, gọi `Refresh()` cục bộ.
   - Hoàn toàn không gọi `accountListContainer.Objects = nil`, không gọi `accountListContainer.Refresh()`.
3. **Observable Result:**
   - Giao diện cập nhật tức thì phần trăm và thanh bar Quota mà không chớp nháy, không giật khung hình.
   - Vị trí cuộn của người dùng được giữ nguyên 100%.
   - Khi người dùng kéo thu nhỏ cửa sổ tới 480x360 px, chu kỳ Quota 60s trôi qua cửa sổ vẫn đứng yên hoàn toàn, không bị Fyne GLFW driver phóng to lại.
   - Kích thước cửa sổ được lưu bền vững vào file preferences và khôi phục chuẩn xác khi khởi động lại.

---

## 5. Conclusion & Handoff Recommendation

- **Verdict:** READY_FOR_OWNER_REVIEW
- Toàn bộ 4 Invariants (`INV-01`, `INV-02`, `INV-03`, `INV-04`) đã được thi công hoàn chỉnh và kiểm thử tự động đạt 100% PASS.
- Sẵn sàng bàn giao cho Human Owner thực nghiệm runtime trên môi trường Windows.