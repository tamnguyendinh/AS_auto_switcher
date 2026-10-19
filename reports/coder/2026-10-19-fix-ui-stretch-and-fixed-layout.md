# Coder Implementation Report: Khóa Cứng UI Tổng, Cố Định Bố Cục Thẻ & Chỉ Cho Phép Thanh Minibar Co Giãn

- **Date:** 2026-10-19
- **Role:** Coder
- **Authority / SSOT:**
  - `reports/architect/2026-10-19-fix-ui-stretch-and-fixed-layout-spec.md` (Approved directly by Human Owner)
  - `SPEC-ARCH-20261019-FIX-UI-STRETCH-AND-FIXED-LAYOUT`
  - `AGENTS.md` & `working-rules`
- **Mục tiêu:** Thi công khóa cứng toàn bộ UI tổng và cấu trúc thẻ tài khoản theo đặc tả kiến trúc:
  1. **UI tổng (kích thước cửa sổ Windows & cấu trúc khung app):** Tuyệt đối KHÔNG ĐƯỢC THAY ĐỔI khi Quota refresh tự động (mỗi 60s) hoặc làm mới thủ công. Kích thước cửa sổ chỉ thay đổi khi người dùng tự tay dùng chuột kéo mép cửa sổ. Chặn đứng lỗi lưu đè kích thước giả mạo (`490x387 px` / `491x406 px`) lúc khởi động.
  2. **Cấu trúc bên trong thẻ Card khi Quota refresh:** CHỈ DUY NHẤT THANH MINIBAR (`fillBar`) ĐƯỢC PHÉP CO GIÃN ĐỘ DÀI THEO %; nhãn % cập nhật chuỗi số; TẤT CẢ các thành phần khác KHÓA CỨNG 100% (cột 5h cố định `88.0px`, cột Week cố định `88.0px`, cột Model cố định `50.0px`, ray nền `bgTrack` cố định `40.0px`), bảng ma trận và toàn bộ thẻ không xê dịch, không co giãn, không giật khung hình.
  3. **Reset file cấu hình:** Reset `accounts_store/.ui_preferences.json` về kích thước chuẩn `1008x630 px`.
  4. **Kiểm thử & Build:** Cập nhật test suite `internal/ui/ui_test.go` và chạy full build verification bằng `scripts/build_windows.ps1 -Mode all` (Exit code 0).

---

## 1. Invariant Family Map

| Invariant | Target File | Implementation Details |
|---|---|---|
| **INV-01 (Fixed Matrix Layout)** | `internal/ui/quota_badge.go` | - Khóa cứng 100% kích thước các cột trong `matrixTableLayout.Layout`: cột Model = `50.0px`, vách ngăn = `1.0px`, gap = `4.0px`, hai cột 5h và Week khóa cứng `colCycleW = 88.0px` tuyệt đối, xóa bỏ hoàn toàn cơ chế tính `remainW` đàn hồi.<br>- Khóa cứng `matrixTableLayout.MinSize` trả về kích thước cố định `244.0 x 49.0 px` (`50 + 1 + 88 + 1 + 88 + 16 = 244px`). |
| **INV-02 (Only Minibar Elastic)** | `internal/ui/quota_badge.go` | - Trong `matrixQuotaCell.update()`: chỉ tính toán và co giãn `fillWidth = barWidth * float32(clamped)` (với `barWidth = 40.0px`), cập nhật màu theo tier và nội dung `pctText`.<br>- Loại bỏ hoàn toàn lệnh `c.container.Refresh()` để ngăn chặn Fyne layout cascade rung giật toàn bộ thẻ Card khi Quota thay đổi. |
| **INV-03 (UI Ready Guard & Startup Geometry Stability)** | `internal/ui/app.go`<br>`internal/ui/window.go` | - Bổ sung cờ đồng bộ `isUIReady atomic.Bool` vào `App`.<br>- Trong `ShowAndRun()`: chỉ đánh dấu `isUIReady = true` sau khi `a.win.Show()`, `MakeWindowFrameless()` hoàn tất và delay 300ms để bỏ qua các xung resize khởi tạo của OS.<br>- Trong `minSizeLayout.Layout()`: chỉ gọi `a.app.OnWindowResized()` khi `m.app.isUIReady.Load() == true`.<br>- Trong `OnWindowResized()`: chặn ngay lập tức nếu `!a.isUIReady.Load()`. Tăng debounce lên 400ms.<br>- Trong `Init()` & `NewAppWithFyne()`: nếu preferences chưa có kích thước hoặc kích thước nhỏ hơn `800x500` (kích thước lỗi bị đè trước đây), lập tức khôi phục về kích thước chuẩn mặc định `1008x630 px`. |
| **INV-04 (Preferences Reset)** | `accounts_store/.ui_preferences.json` | - Reset tệp cấu hình về kích thước chuẩn `1008x630 px` với `theme: "dark"`. |
| **INV-05 (Verification Suite)** | `internal/ui/ui_test.go` | - Cập nhật `TestArchitectSpec_INV03_WindowGeometryPersistence` kiểm tra kích thước hợp lệ `>= 800x500` và kiểm tra tự động reset kích thước lỗi `< 800x500` về `1008x630`.<br>- Cập nhật `TestArchitectSpec_INV04_FixedMatrixQuotaTableLayout` kiểm chứng các cột Model (`50px`), 5h (`88px`), Week (`88px`) cố định 100% tại mọi kích thước thẻ (236px, 300px, 400px).<br>- Thêm `TestArchitectSpec_INV05_IsUIReadyProtectsWindowResize` kiểm chứng cơ chế chặn ghi đè khi chưa ready.<br>- Thêm `TestArchitectSpec_INV06_OnlyMinibarExpandsOnQuotaUpdate` kiểm chứng chỉ thanh minibar thay đổi độ dài, ray nền cố định 40px. |

---

## 2. Touch-Map Compliance Verification

Tuân thủ nghiêm ngặt ranh giới Touch-Map được giao, tuyệt đối không chỉnh sửa bất kỳ file nào ngoài phạm vi:

| STT | File Path | Status | Verification Summary |
|---|---|---|---|
| 1 | `internal/ui/quota_badge.go` | Modified | Khóa cứng `colCycleW = 88.0px`, `colModel = 50.0px`, MinSize cố định 244x49px, chỉ co giãn `fillBar`, bỏ `container.Refresh()`. |
| 2 | `internal/ui/app.go` | Modified | Thêm `isUIReady atomic.Bool`, chặn lưu size lúc khởi động trong `OnWindowResized`, khôi phục chuẩn `1008x630` khi size < `800x500`. |
| 3 | `internal/ui/window.go` | Modified | Bọc điều kiện `m.app.isUIReady.Load()` trong `minSizeLayout.Layout`. |
| 4 | `accounts_store/.ui_preferences.json` | Reset | Reset về `1008x630 px`, `dark` theme. |
| 5 | `internal/ui/ui_test.go` | Modified | Cập nhật assertions khớp với `colCycleW = 88.0`, bổ sung test `isUIReady`, minibar resize, và reset geometry lỗi. |
| 6 | `reports/coder/2026-10-19-fix-ui-stretch-and-fixed-layout.md` | Created | Báo cáo thi công chính thức. |

**Non-goals Verified:**
- NO Anvien utilized.
- File `scripts/build_windows.ps1` giữ nguyên trạng 100% không bị can thiệp.
- Không spawn subagent.
- Tuyệt đối không thay đổi bất kỳ logic nào ngoài đặc tả của Kiến trúc sư.

---

## 3. Mandatory Evidence: Full Build Verification

Chạy lệnh bắt buộc của repo:
```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all
```

**Kết quả thực thi (Exit Code: 0):**
```text
================================================================================
⚡ Antigravity Multi-Account Switcher - Windows Build System
   Mode: all | OutDir: bin | SkipTests: False | Clean: False
================================================================================
[2/4] Running pre-flight tests & vet...
  -> Verifying pure Go packages (CGO_ENABLED=0)...
ok  	antigravity-switch/internal/model	(cached)
ok  	antigravity-switch/internal/wincred	(cached)
ok  	antigravity-switch/internal/store	5.613s
ok  	antigravity-switch/internal/scanner	(cached)
ok  	antigravity-switch/internal/connector	(cached)
ok  	antigravity-switch/internal/reloader	(cached)
ok  	antigravity-switch/internal/manager	8.544s
ok  	antigravity-switch/cmd/switcher	(cached)
  [PASS] Pure Go verification succeeded.
  -> Verifying GUI packages with CGo & Fyne (CGO_ENABLED=1)...
ok  	antigravity-switch/cmd/switcher	1.499s
ok  	antigravity-switch/internal/ui	11.126s
  [PASS] GUI verification succeeded.
[3/4] Compiling target binaries...
  -> Checking Windows application icon and resource...
  -> Compiling resource with windres: cmd\switcher\app.rc -> cmd\switcher\rsrc_windows_amd64.syso
  [SUCCESS] Compiled PE resource object: cmd\switcher\rsrc_windows_amd64.syso (12552 bytes)
  -> Building Standalone CLI Binary (Pure Go, CGO_ENABLED=0)...
  [SUCCESS] Created: bin\antigravity-switch.exe
  Compile Duration: 3998 ms | Size: 10.51 MB (11023872 bytes)
  -> Building Standalone GUI Binary (Fyne, CGO_ENABLED=1)...
  [SUCCESS] Created: bin\antigravity-switch-gui.exe
  Compile Duration: 7248 ms | Size: 27.6 MB (28937216 bytes)
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

---

## 4. Acceptance Criteria Verification

- [x] **AC-01 (Fixed Table Layout):** Bảng Quota có chiều rộng và vị trí cột cố định 100% (`col1 = 50px`, `col5h = 88px`, `colWeek = 88px`). Đã kiểm chứng qua test `TestArchitectSpec_INV04_FixedMatrixQuotaTableLayout` tại các kích thước 236px, 300px, 400px.
- [x] **AC-02 (Only Minibar Changes):** Khi Quota refresh, CHỈ CÓ thanh `fillBar` bên trong ray 40px thay đổi chiều dài theo tỷ lệ fraction. Ray `bgTrack` cố định 40px. Đã kiểm chứng qua test `TestArchitectSpec_INV06_OnlyMinibarExpandsOnQuotaUpdate`.
- [x] **AC-03 (UI Tổng Bất Biến):** Kích thước toàn bộ cửa sổ không bị Fyne layout hay Quota refresh thay đổi. Cờ `isUIReady` bảo vệ hoàn toàn trước các xung resize ban đầu.
- [x] **AC-04 (True User Resize Persistence):** Kích thước chỉ được lưu khi người dùng kéo chuột sau khi UI đã sẵn sàng (`isUIReady == true`). Các kích thước lỗi nhỏ hơn 800x500 được tự động khôi phục về 1008x630.
- [x] **AC-05 (Build & Tests):**
  - `go test -tags gui ./internal/ui/...` đạt **100% PASS**.
  - `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all` đạt **Exit code = 0**.

---

## 5. Kết Luận
Thi công hoàn tất 100% theo đúng đặc tả kỹ thuật của Kiến trúc sư `reports/architect/2026-10-19-fix-ui-stretch-and-fixed-layout-spec.md`. Sẵn sàng bàn giao cho Human Owner tiến hành nghiệm thu trực tiếp runtime.
