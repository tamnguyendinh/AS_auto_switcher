# Supervisor Report: Acceptance Review of Frameless Window CSD & TopSection Refactor (Commit 80c2276)

Verdict: REJECT

## Metadata
- Report file: reports/supervisor/2026-10-19-frameless-window-csd-acceptance.md
- Review time: 2026-10-19 06:50:00 ICT (+0700)
- Reviewer: Supervisor Lane (Claude Sonnet 4.6 / gpt-6.1-sol)
- Repo/project: antigravity_auto_switch
- Scope reviewed: Commit `80c227696eb536e96ca9a775799c74f57add9515` ("feat(ui): implement custom frameless window csd and streamline topsection")
- Claim reviewed: Full implementation of Custom Frameless Window CSD (Issue 1), Row 2 Status & Mini Theme icon (Issue 2), and Row 3 Flat Toolbar & timestamp cleanup (Issue 3).
- Authority used:
  - Visual defect evidence from Human Owner: `reports/problem/images/screenshot_1792366727.png`
  - Architect Specification: `reports/architect/2026-10-19-frameless-window-csd-and-topsection-refactor-spec.md`
  - Coder Report: `reports/coder/2026-10-19-frameless-window-csd-and-topsection-refactor.md`
  - AGENTS.md & working-rules
- Related artifacts:
  - Defect screenshot: `reports/problem/images/screenshot_1792366727.png`
  - Binaries: `bin/antigravity-switch.exe`, `bin/antigravity-switch-gui.exe`
  - Build script: `scripts/build_windows.ps1`

---

## Executive Summary
- **Problem:** Human Owner detected and recorded indisputable visual evidence in `reports/problem/images/screenshot_1792366727.png` showing that Vấn đề 1 (Cửa sổ Frameless CSD) chưa hoàn tất: thanh tiêu đề mặc định của Windows (màu trắng, icon AS, tiêu đề "⚡ Antigravity Multi-Account Switcher", nút thu nhỏ/phóng to/đóng tiêu chuẩn OS) chưa bị gỡ bỏ, dẫn đến việc xuất hiện 2 thanh tiêu đề xếp chồng lên nhau (thanh tiêu đề Windows ở trên và thanh tiêu đề tự vẽ Hàng 1 ở dưới).
- **Decision:** **REJECT**. Bằng chứng thực nghiệm Zero-Trust 100% chứng minh rằng:
  1. Trong `internal/ui/app.go:204-207`, việc gỡ bỏ thanh tiêu đề được kích hoạt bằng một goroutine đơn lẻ với độ trễ tĩnh `time.Sleep(150 * time.Millisecond)`.
  2. Đo đạc thực tế vòng đời khởi động Fyne/GLFW trên Windows cho thấy: Tại 100ms, 150ms, 200ms và 300ms, `MainWindowHandle` hoàn toàn bằng 0; cửa sổ chỉ thực sự được hệ điều hành khởi tạo và hiển thị sau ~400-500ms.
  3. Tại thời điểm 150ms, `FindWindowW` trả về `hwnd == 0`. Hàm `MakeWindowFrameless` âm thầm return 0 mà không hề có cơ chế thử lại (retry/polling).
  4. Hệ quả: `WS_CAPTION` (0x00C00000) không bao giờ bị loại bỏ (`Style` duy trì `0x16CF0000`), biến toàn cục `mainHWND` bị kẹt ở `0`, khiến toàn bộ các tương tác native trên Hàng 1 (kéo thả cửa sổ qua `windowDragArea`, nút thu nhỏ `—`, nút phóng to `口`) hoàn toàn bị tê liệt (dead stubs), đồng thời làm biến dạng cấu trúc giao diện như Owner đã chụp lại.
- **Required outcome:** Coder phải sửa chữa cơ chế can thiệp HWND trong `internal/ui/app.go` và `internal/ui/frameless_windows.go` bằng vòng lặp thăm dò (polling/retry) tin cậy hoặc hook vòng đời hợp lệ để đảm bảo tước bỏ triệt để `WS_CAPTION`, lưu trữ chính xác `mainHWND`, và loại bỏ hoàn toàn thanh tiêu đề mặc định của Windows.

---

## Detailed Evaluation Across All 3 Improvement Items

### 1. Vấn đề 1: Cửa sổ Frameless CSD (Custom Titlebar & Window Controls) — [VERDICT: REJECT]
- **Tình trạng:** Thất bại nghiêm trọng tại runtime.
- **Bằng chứng đối soát trực quan:** Ảnh `reports/problem/images/screenshot_1792366727.png` phơi bày rõ ràng 2 thanh tiêu đề xếp chồng:
  - Thanh trên: Thanh tiêu đề native chuẩn của Windows 10/11 với nền trắng, icon AS, chuỗi tiêu đề `"⚡ Antigravity Multi-Account Switcher"`, và bộ nút `[_] [口] [X]`.
  - Thanh dưới: Hàng 1 CSD tự vẽ với badge `[AS]`, chuỗi tiêu đề `"Antigravity Multi-Account Switcher"`, và bộ 3 nút tự vẽ `[ — ] [ 口 ] [ ✕ ]`.
- **Phân tích cơ chế kỹ thuật sâu (Root Cause Analysis):**
  - Trong `internal/ui/app.go`:
    ```go
    func (a *App) Init() {
        ...
        go func() {
            time.Sleep(150 * time.Millisecond)
            MakeWindowFrameless(windowTitle)
        }()
    ```
  - Trong `internal/ui/frameless_windows.go`:
    ```go
    func MakeWindowFrameless(windowTitle string) uintptr {
        titlePtr := windows.StringToUTF16Ptr(windowTitle)
        hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(titlePtr)))
        if hwnd == 0 {
            return 0
        }
        SetMainHWND(hwnd)
        style, _, _ := procGetWindowLongW.Call(hwnd, uintptr(gwlStyleVal))
        newStyle := (style &^ uintptr(WS_CAPTION)) | uintptr(WS_THICKFRAME)
        procSetWindowLongW.Call(hwnd, uintptr(gwlStyleVal), newStyle)
        procSetWindowPos.Call(hwnd, 0, 0, 0, 0, 0, uintptr(SWP_FRAMECHANGED|SWP_NOMOVE|SWP_NOSIZE|SWP_NOZORDER))
        return hwnd
    }
    ```
  - **Chứng cứ đo đạc thực nghiệm độc lập của Supervisor:**
    Chạy probe vòng đời tiến trình GUI tại các mốc thời gian:
    - Tại 100ms: `MainWindowHandle = 0`
    - Tại 150ms: `MainWindowHandle = 0` (Thời điểm goroutine chạy -> `FindWindowW` trả về 0 -> Thoát luôn!)
    - Tại 200ms: `MainWindowHandle = 0`
    - Tại 300ms: `MainWindowHandle = 0`
    - Tại 500ms: `MainWindowHandle = 12193928`
  - Khi ứng dụng chạy bình thường, kiểm tra thuộc tính Win32 lúc 2 giây:
    - `Style: 0x16CF0000` -> `Has WS_CAPTION: True` (Thanh tiêu đề Windows vẫn 100% nguyên vẹn).
  - Vì `hwnd == 0` lúc 150ms, hàm `SetMainHWND(hwnd)` không được gọi -> `mainHWND` = 0.
  - Các hàm điều khiển cửa sổ phụ thuộc `GetMainHWND()`:
    - `DragNativeWindow()`: Bị bỏ qua vì `if hwnd != 0`. Kéo thanh CSD không di chuyển được cửa sổ.
    - `MinimizeNativeWindow()`: Bị bỏ qua vì `if hwnd != 0`. Bấm nút `—` không hoạt động.
    - `ToggleMaximizeNativeWindow()`: Bị bỏ qua vì `if hwnd != 0`. Bấm nút `口` không hoạt động.

### 2. Vấn đề 2: Tinh giản Hàng 2 (Status & Mini Theme icon) — [VERDICT: ACCEPT]
- **Tình trạng:** Đạt yêu cầu thiết kế và đặc tả.
- **Đối soát mã nguồn & Giao diện:**
  - `internal/ui/window.go:121-149`:
    - `serverStatusLabel` được đưa sang trái, hiển thị trạng thái đưa lên đầu: `"Hoạt động  | Đang kết nối..."`.
    - Khi nhận telemetry trong `internal/ui/app.go:640-646`: Hiển thị `"Hoạt động  | PID %d | Port %d"` hoặc `"Không hoạt động  | Dự phòng WinCred"`.
    - Dropdown theme mini được bọc trong `container.NewGridWrap(fyne.NewSize(58, 32), a.themeSelect)`, chỉ chứa 3 biểu tượng ngắn gọn: `["🌙", "☀️", "☕"]`.
  - Trên ảnh chụp thực tế `screenshot_1792366727.png`, Hàng 2 hiển thị chính xác: `Hoạt động | PID 21484 | Port 65144` bên trái và icon `🌙 ▼` thu gọn ở góc phải, giải phóng không gian theo đúng ý đồ kiến trúc.

### 3. Vấn đề 3: Tinh giản Hàng 3 (Toolbar phẳng & gỡ nhãn thời gian) — [VERDICT: ACCEPT]
- **Tình trạng:** Đạt yêu cầu thiết kế và đặc tả.
- **Đối soát mã nguồn & Giao diện:**
  - `internal/ui/window.go:153-172`:
    - Đã chuyển các nút thành dạng nút chữ phẳng, bỏ icon thừa: `[ Danh sách tài khoản ]`, `[ Làm mới Quota ]`, `[ 📌 Ghim : Bật ]`.
    - Đã gỡ bỏ toàn bộ nhãn thời gian cập nhật `lastUpdatedLabel` / `autoRefreshLabel` khỏi bố cục `row3Toolbar`.
    - Bên phải Hàng 3 để trống hoàn toàn, không còn hiện chữ `"Cập nhật : HH:MM:SS"` gây rối mắt.
  - Trên ảnh chụp thực tế `screenshot_1792366727.png`, Hàng 3 hiển thị gọn gàng, chuẩn xác 3 nút phẳng bên trái và không gian mở rộng sang bên phải.

---

## Blocking Findings

### [P0/BLOCKER] Double Title Bar & Silent HWND Acquisition Failure at Startup
- **File:** `internal/ui/app.go:204-207`, `internal/ui/frameless_windows.go:56-70`
- **Issue:** 
  Cơ chế kích hoạt `MakeWindowFrameless` bằng goroutine đơn lẻ với độ trễ cứng `150ms` xảy ra race condition nghiêm trọng với vòng đời Fyne GLFW window initialization (vốn cần ~400-500ms). Tại 150ms, HWND chưa tồn tại khiến `FindWindowW` trả về 0. Hàm thoát sớm không retry, không tước bỏ `WS_CAPTION`, không lưu `mainHWND`, gây ra lỗi 2 thanh tiêu đề xếp chồng và vô hiệu hóa toàn bộ nút điều khiển CSD.
- **Evidence:**
  1. Ảnh chụp thực tế của Owner: `reports/problem/images/screenshot_1792366727.png`.
  2. Thực nghiệm đo đạc `MainWindowHandle` qua thời gian: HWND = 0 ở các mốc 100ms, 150ms, 200ms, 300ms.
  3. Giá trị Win32 `GWL_STYLE` thực tế trên tiến trình đang chạy: `0x16CF0000` (chứa cờ `WS_CAPTION = 0x00C00000`).
  4. Thực nghiệm kiểm chứng can thiệp trực tiếp Win32 API trên HWND sống: Khi áp dụng công thức tước `WS_CAPTION` sau khi có HWND, `Style` chuyển thành công từ `0x16CF0000` sang `0x160F0000` (`Has WS_CAPTION: False`).
- **Why this blocks acceptance:** 
  Vi phạm trực tiếp yêu cầu cốt lõi của Vấn đề 1, tạo ra giao diện biến dạng, lỗi thẩm mỹ nặng và hỏng tính năng cơ bản của các nút điều khiển cửa sổ.
- **Fix direction:**
  Trong `internal/ui/frameless_windows.go` (hoặc `internal/ui/app.go`), thay thế việc sleep 150ms cố định bằng cơ chế thăm dò tích cực (polling loop):
  - Chạy goroutine thăm dò định kỳ (ví dụ mỗi 30-50ms) trong tối đa 3-5 giây cho đến khi `procFindWindowW` tìm thấy `hwnd != 0`.
  - Khi tìm thấy `hwnd`: Lưu `SetMainHWND(hwnd)`, áp dụng `SetWindowLongW` gỡ `WS_CAPTION`, thêm `WS_THICKFRAME`, và gọi `SetWindowPos` với cờ `SWP_FRAMECHANGED | SWP_NOMOVE | SWP_NOSIZE | SWP_NOZORDER`.
  - Có thể áp dụng thêm một lần re-assert sau đó để đảm bảo Fyne không ghi đè lại style khi map window.
- **Re-review evidence required:**
  - Ảnh chụp thực tế hoặc kiểm tra runtime xác nhận cửa sổ mở lên hoàn toàn không còn thanh tiêu đề xám/trắng của Windows.
  - Probe kiểm tra `GetWindowLongW(hwnd, -16)` xác nhận `(style & WS_CAPTION) == 0`.
  - Xác nhận bấm `—` cửa sổ thu nhỏ, bấm `口` cửa sổ phóng to/khôi phục, bấm giữ kéo Hàng 1 cửa sổ di chuyển native mượt mà.

---

## Source-Level Clearance Notes
- `internal/ui/frameless_windows.go`: **BLOCKED** - Cần bổ sung polling/retry loop khi tìm HWND để đảm bảo xử lý được độ trễ khởi động của Fyne GUI.
- `internal/ui/frameless_other.go`: **CLEAR** - No-op stubs biên dịch an toàn cho các nền tảng `!windows`.
- `internal/ui/window.go`: **CLEAR** - Bố cục CSD Hàng 1, Hàng 2 mini theme, Hàng 3 toolbar phẳng và xóa timestamp được code đúng cấu trúc.
- `internal/ui/app.go`: **BLOCKED** - Dòng 204-207 sử dụng `time.Sleep(150 * time.Millisecond)` gây race condition; cần chuyển sang polling có retry hoặc ủy quyền cho `MakeWindowFrameless` tự poll.
- `internal/ui/ui_test.go`: **CLEAR** - Unit test suite được cập nhật phù hợp với định dạng mới của TopSection.

---

## Evidence Checked

### Passed:
- **Build Script:** `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all`
  - Pure Go tests: PASS
  - GUI tests with CGO: PASS
  - Binary compilation:
    - `bin/antigravity-switch.exe` (10.51 MB) created successfully.
    - `bin/antigravity-switch-gui.exe` (27.56 MB) created successfully.
  - Script exit code: 0 (Kỷ luật sắt: script build không bị sửa đổi).
- **Hạng mục 2 (Row 2 Status & Mini Theme):** Xác thực mã nguồn và hiển thị thực tế trên screenshot trùng khớp đặc tả.
- **Hạng mục 3 (Row 3 Toolbar & Timestamp cleanup):** Xác thực mã nguồn và hiển thị thực tế trên screenshot trùng khớp đặc tả.

### Failed:
- **Hạng mục 1 (Frameless CSD & gỡ thanh tiêu đề Windows):**
  - Defect screenshot: `reports/problem/images/screenshot_1792366727.png` xác nhận 2 thanh tiêu đề xếp chồng.
  - Runtime HWND probe: HWND = 0 tại 150ms -> `MakeWindowFrameless` bị hụt -> `WS_CAPTION` còn nguyên `0x16CF0000`.

---

## Invariant Closure
- **Affected invariant:** Native window frame suppression contract (Windows OS decoration must be stripped cleanly without leaving residual titlebars or breaking CSD controls).
- **Sibling surfaces checked:** Single-instance activation (`internal/ui/single_instance_windows.go`), `setWindowAlwaysOnTop`, tray integration, CSD window controls (`customTitleBtn`, `customCloseBtn`, `windowDragArea`).
- **Residual unverified same-invariant surfaces:** Cần kiểm tra độ mượt của việc kéo thả cửa sổ (`DragNativeWindow`) và các nút thu nhỏ/phóng to một khi HWND được bắt thành công.

---

## Required Fix List For Resubmission

1. **Sửa đổi cơ chế HWND Resolution:**
   - Trong `MakeWindowFrameless` hoặc goroutine khởi tạo tại `internal/ui/app.go`, thay thế `time.Sleep(150 * time.Millisecond)` bằng vòng lặp thăm dò (polling loop) kiểm tra `FindWindowW(0, titlePtr)` mỗi 30-50ms trong tối đa 3-5 giây.
2. **Đảm bảo tính trọn vẹn của Frameless Hook:**
   - Khi tìm thấy HWND (`hwnd != 0`), gọi `SetMainHWND(hwnd)` ngay lập tức.
   - Thực thi `SetWindowLongW` loại bỏ `WS_CAPTION`, thêm `WS_THICKFRAME`.
   - Thực thi `SetWindowPos` với cờ `SWP_FRAMECHANGED | SWP_NOMOVE | SWP_NOSIZE | SWP_NOZORDER` để ép hệ điều hành tính toán lại non-client area.
3. **Chạy lại Full Build Script:**
   - Chạy `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all` đảm bảo exit code 0.
4. **Cung cấp Bằng chứng Runtime:**
   - Cung cấp log probe hoặc ảnh chụp nghiệm thu chứng minh cửa sổ chỉ còn duy nhất 1 thanh tiêu đề tự vẽ CSD của ứng dụng, không còn thanh tiêu đề xám/trắng của Windows.

---

## Overall Evaluation
Commit `80c2276` đã thực hiện rất tốt việc tinh giản Hàng 2 (Status-First + Mini Theme icon) và Hàng 3 (Toolbar phẳng + loại bỏ timestamp). Tuy nhiên, đối với hạng mục cốt lõi nhất là Cửa sổ Frameless CSD (Vấn đề 1), việc giả định thời gian khởi tạo HWND cố định 150ms đã dẫn đến lỗi race condition tại runtime, khiến thanh tiêu đề mặc định của Windows không bị loại bỏ như phản ánh chính xác của Human Owner.

Vì vậy, theo nguyên tắc Zero-Trust và kỷ luật nghiệm thu độc lập, Supervisor ban hành quyết định **[VERDICT: REJECT]** đối với Vấn đề 1 và toàn bộ gói nghiệm thu của commit `80c2276`, yêu cầu Coder tiến hành sửa chữa theo các chỉ dẫn kỹ thuật nêu trên.
