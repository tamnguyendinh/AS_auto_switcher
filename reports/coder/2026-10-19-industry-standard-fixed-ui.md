# Báo Cáo Thi Công Coder: Gỡ Bỏ Toàn Bộ Thành Phần Co Giãn, Khóa Cứng Chỉ Số Cố Định Chuẩn Ngành (Zero Resizing - Fixed Metric Layout)

- **Ngày thi công:** 2026-10-19
- **Vai trò:** Coder
- **Thẩm quyền:** Lệnh tối cao trực tiếp từ Human Owner
- **Tài liệu tham chiếu / SSOT:**
  - `reports/architect/2026-10-19-industry-standard-fixed-ui-spec.md` (APPROVED BY OWNER)
  - `SPEC-ARCH-20261019-INDUSTRY-STANDARD-FIXED-UI`
  - `AGENTS.md` & `working-rules`
- **Mục tiêu hoàn thành:** Thi công chuẩn ngành gỡ bỏ toàn bộ thành phần co giãn, khóa cứng chỉ số cố định (Zero Resizing - Fixed Metric Layout):
  1. **Khóa cứng chuẩn ngành Desktop:** Toàn bộ bảng, cột, thẻ Card tuân theo hệ thống số đo cố định tuyệt đối (Fixed Metric System). Cột 5h khóa cứng 88.0px, cột Week khóa cứng 88.0px, cột Model khóa cứng 50.0px, ray nền khóa cứng 40.0px.
  2. **Quota refresh ổn định tuyệt đối:** Khi Quota làm mới tự động (60s) hoặc làm mới thủ công, CHỈ CÓ vệt màu tiến trình `fillBar` bên trong ray 40px thay đổi chiều dài theo % quota (tụt dần khi dùng); nhãn % cập nhật chuỗi số; TẤT CẢ các thành phần khác đứng im 100%, không co giãn theo.
  3. **Gỡ bỏ toàn diện hạ tầng resize:** Xóa bỏ hoàn toàn struct `minSizeLayout` và hook `OnWindowResized`, các biến timer resize (`resizeTimer`, `resizeMu`, `isUIReady`) trong `App`, xóa bỏ `WindowWidth`/`WindowHeight` trong `UIPreferences`.
  4. **Thi công đúng 100% mã mẫu:** Không suy diễn, rà soát và kiểm thử đạt 100% PASS trên toàn bộ suite và chạy full build script thành công.

---

## 1. Invariant Family Map

| Invariant | Tệp thực hiện | Chi tiết thi công kỹ thuật |
|---|---|---|
| **INV-01 (Fixed Metric System 100%)** | `internal/ui/quota_badge.go` | - Khóa cứng 100% kích thước các cột trong `matrixTableLayout.Layout`: cột Model = `50.0px`, vách ngăn = `1.0px`, khoảng cách cột = `4.0px`, hai cột chu kỳ 5h và Week khóa cứng `colCycleW = 88.0px` tuyệt đối.<br>- Khóa cứng `matrixTableLayout.MinSize` trả về kích thước cố định `244.0 x 49.0 px` (`50 + 1 + 88 + 1 + 88 + 16 = 244px`). Không có bất kỳ dòng mã co giãn động nào. |
| **INV-02 (Only Minibar Decreases On Refresh)** | `internal/ui/quota_badge.go` | - Trong `matrixQuotaCell.update()`: chỉ tính toán chiều dài `fw = barWidth * float32(clamped)` (với `barWidth = 40.0px`), cập nhật màu tier và nội dung `pctText`.<br>- Xóa bỏ hoàn toàn lệnh `c.container.Refresh()` để ngăn chặn Fyne layout cascade làm rung giật thẻ và cửa sổ. |
| **INV-03 (Eliminate minSizeLayout & VBox Wrap)** | `internal/ui/window.go` | - Xóa sạch struct `minSizeLayout` và các phương thức `MinSize()`, `Layout()` của nó.<br>- Gỡ bỏ lớp bọc `container.NewVBox()` gây cascade chiều cao trong `scrollList`: gắn trực tiếp `accountListContainer` vào `container.NewVScroll(container.NewPadded(...))`.<br>- Gán trực tiếp `mainLayout` vào `a.win.SetContent(mainLayout)`. |
| **INV-04 (Clean App Resize Logic & Standard Init)** | `internal/ui/app.go` | - Dọn sạch struct `App`: xóa bỏ các trường `windowWidth`, `windowHeight`, `resizeTimer`, `resizeMu`, `isUIReady`.<br>- Xóa bỏ hoàn toàn các hàm: `WindowSize()`, `OnWindowResized()`, `saveWindowGeometry()`, `SaveWindowGeometryNow()`, `SetUIReady()`, `IsUIReady()`.<br>- Trong `Init()`: khởi tạo chuẩn tỷ lệ `1008 x 630 px`, cho phép người dùng tự do kéo chuột với `a.win.SetFixedSize(false)`.<br>- Trong `Close()`: chỉ dừng vòng lặp nền `autoLoop`.<br>- Trong `SetTheme()`: dọn sạch các truy cập `resizeMu` và chỉ lưu `Theme`. |
| **INV-05 (Clean UIPreferences)** | `internal/ui/themes.go`<br>`accounts_store/.ui_preferences.json` | - Trả `UIPreferences` về nguyên bản: `type UIPreferences struct { Theme string json:"theme" }`.<br>- File `accounts_store/.ui_preferences.json` reset sạch sẽ: `{"theme": "dark"}`. |
| **INV-06 (Industry Standard Test Assertions)** | `internal/ui/ui_test.go` | - Xóa các test case cũ gọi hàm resize thừa.<br>- Thêm `TestIndustryStandard_FixedMatrixMetrics`: kiểm tra Model 50px, 5h 88px, Week 88px, ray 40px tại mọi kích thước cha (236, 244, 300, 400, 600px).<br>- Thêm `TestIndustryStandard_OnlyMinibarDecreasesOnQuotaUpdate`: kiểm chứng khi quota tụt 100% -> 50% -> 0%, toàn bộ thẻ và ray nền 40px đứng im 100%, chỉ duy nhất `fillBar` co ngắn lại (40px -> 20px -> 0px).<br>- Thêm `TestIndustryStandard_WindowInit1008x630`: kiểm chứng cửa sổ mở ra đúng kích thước chuẩn 1008x630 px và resizable. |

---

## 2. Touch-Map Compliance Verification

Tuân thủ nghiêm ngặt ranh giới Touch-Map được giao, tuyệt đối không chỉnh sửa bất kỳ file nào ngoài phạm vi:

| STT | File Path | Trạng thái | Tóm tắt kiểm tra |
|---|---|---|---|
| 1 | `internal/ui/quota_badge.go` | Modified | Hardcode cố định 50px, 88px, 88px, ray 40px; xóa bỏ `c.container.Refresh()`. |
| 2 | `internal/ui/window.go` | Modified | Xóa bỏ `minSizeLayout`, bỏ bọc `NewVBox` trong `scrollList`, gán thẳng `mainLayout`. |
| 3 | `internal/ui/app.go` | Modified | Dọn sạch struct `App`, xóa timer/mutex/cờ resize, xóa các hàm resize thừa, `Init()` mở `1008x630`. |
| 4 | `internal/ui/themes.go` | Modified | Gỡ bỏ `WindowWidth`, `WindowHeight` trong `UIPreferences`. |
| 5 | `accounts_store/.ui_preferences.json` | Reset | Đặt về `{"theme": "dark"}`. |
| 6 | `internal/ui/ui_test.go` | Modified | Cập nhật bộ test chuẩn ngành (Fixed Metrics, Minibar Decreases, Window Init). |
| 7 | `reports/coder/2026-10-19-industry-standard-fixed-ui.md` | Created | Báo cáo thi công chính thức của Coder. |

**Non-goals Verified:**
- NO Anvien utilized.
- File `scripts/build_windows.ps1` tuyệt đối giữ nguyên trạng không bị can thiệp.
- Không spawn subagent.
- Đúng 100% mã mẫu của Kiến trúc sư, không suy diễn thêm bớt.

---

## 3. Mandatory Evidence: Full Build Verification

Thực thi lệnh kiểm tra bắt buộc của repo:
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
ok  	antigravity-switch/internal/store	8.346s
ok  	antigravity-switch/internal/scanner	(cached)
ok  	antigravity-switch/internal/connector	(cached)
ok  	antigravity-switch/internal/reloader	(cached)
ok  	antigravity-switch/internal/manager	12.110s
ok  	antigravity-switch/cmd/switcher	(cached)
  [PASS] Pure Go verification succeeded.
  -> Verifying GUI packages with CGo & Fyne (CGO_ENABLED=1)...
ok  	antigravity-switch/cmd/switcher	2.198s
ok  	antigravity-switch/internal/ui	16.099s
  [PASS] GUI verification succeeded.
[3/4] Compiling target binaries...
  -> Checking Windows application icon and resource...
  -> Compiling resource with windres: cmd\switcher\app.rc -> cmd\switcher\rsrc_windows_amd64.syso
  [SUCCESS] Compiled PE resource object: cmd\switcher\rsrc_windows_amd64.syso (12552 bytes)
  -> Building Standalone CLI Binary (Pure Go, CGO_ENABLED=0)...
  [SUCCESS] Created: bin\antigravity-switch.exe
  Compile Duration: 3869 ms | Size: 10.51 MB (11023872 bytes)
  -> Building Standalone GUI Binary (Fyne, CGO_ENABLED=1)...
  [SUCCESS] Created: bin\antigravity-switch-gui.exe
  Compile Duration: 8374 ms | Size: 27.57 MB (28905984 bytes)
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

## 4. Kết Luận & Bàn Giao

Coder đã thi công hoàn tất 100% các yêu cầu trong tài liệu kiến trúc `reports/architect/2026-10-19-industry-standard-fixed-ui-spec.md`. Toàn bộ hệ thống giao diện đạt chuẩn Desktop cố định tuyệt đối, gỡ bỏ toàn bộ thành phần resize phức tạp gây rung lắc, và thanh tiến trình minibar tụt dần mượt mà theo % quota tiêu hao.

Trạng thái: **READY_FOR_OWNER_REVIEW**.
