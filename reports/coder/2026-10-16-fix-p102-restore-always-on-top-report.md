# Coder Report: Restore Always-on-Top Feature & Remove Unapproved Auto-Failover Stubs (P1-02 Parity Fix)

## 1. Metadata
- **Date**: 2026-10-16
- **Task ID**: P1-02
- **Author**: Coder Lane
- **Target Invariant**: Khôi phục tính năng "Ghim trên cùng (Always on Top)" của Python prototype (`ui/app_window.py:86-90, 128-137, 230-236`) và loại bỏ hoàn toàn các logic/nút bấm "Auto-Failover" tự chế chưa được phê duyệt.
- **Authority / SSOT**:
  - `ui/app_window.py:86-90, 128-137, 230-236` (Tkinter `-topmost` attribute, default True, checkbox / toggle callback)
  - `reports/supervisor/2026-10-16-phase6-python-go-full-parity-zero-trust-review.md` (§3.6, §5.5)

---

## 2. Invariant Family Map
- **Family Name**: AlwaysOnTopAndCleanUIParity
- **SSOT**: `ui/app_window.py:86-90, 128-137, 230-236`, Supervisor Zero-Trust Review (§3.6).
- **Broken Surface (Identified)**:
  - Bản Go trước đó đã bỏ sót tính năng cốt lõi "Ghim trên cùng (Always on Top)" của Python (vốn giúp Tool nổi trên IDE Antigravity trong lúc lập trình viên làm việc).
  - Thay vào đó, Go tự ý đưa vào nút `⚡ Auto-Failover: BẬT` cùng các trường tự chế `autoFailover`, `failoverBtn`, `ToggleAutoFailover()` mà không có trong Python prototype gốc.
- **Sibling Runtime Surfaces Inspected & Fixed**:
  - `internal/ui/window.go`:
    + Thay thế hoàn toàn `failoverBtn` trong thanh điều khiển 3 cột bằng `alwaysOnTopBtn`:
      * Trạng thái BẬT: `📌 Ghim: BẬT` với độ ưu tiên cao (`widget.HighImportance`).
      * Trạng thái TẮT: `📌 Ghim: TẮT` với độ ưu tiên bình thường (`widget.MediumImportance`).
    + Định nghĩa interface `Window` mở rộng `fyne.Window` với phương thức `SetAlwaysOnTop(onTop bool)` và `IsAlwaysOnTop() bool`.
    + Triển khai struct `switcherWindow` bọc `fyne.Window`:
      * Hỗ trợ API Fyne Desktop: Gọi `desktop.Window.RequestAlwaysOnTop()`.
      * Tương thích sâu với Win32 OS: Gọi trực tiếp Win32 API `user32.dll::SetWindowPos` với `HWND_TOPMOST` (-1) khi bật và `HWND_NOTOPMOST` (-2) khi tắt, định vị qua `FindWindowW` theo tiêu đề cửa sổ `⚡ Antigravity Multi-Account Switcher`.
      * An toàn tuyệt đối trong môi trường kiểm thử không đầu (headless test driver), không crash khi không có HWND thực.
  - `internal/ui/app.go`:
    + Loại bỏ các trường/hàm tự chế: `autoFailover bool`, `failoverBtn *widget.Button`, `ToggleAutoFailover()`.
    + Bổ sung trường chuẩn: `alwaysOnTop bool` (mặc định `true` tương tự `tk.BooleanVar(value=True)` của Python), `alwaysOnTopBtn *widget.Button`.
    + Thêm các phương thức quản lý trạng thái: `ToggleAlwaysOnTop()`, `SetAlwaysOnTop(enabled bool)`, `IsAlwaysOnTop() bool`.
    + Khi kích hoạt, cập nhật nhãn nút trực quan, gọi `a.win.SetAlwaysOnTop(enabled)` và cập nhật thanh trạng thái `statusBarLabel`.
    + Trong `Init()`, bọc cửa sổ Fyne qua `newSwitcherWindow(rawWin)` và tự động áp dụng `SetAlwaysOnTop(true)`.
  - `internal/ui/ui_test.go`:
    + Cập nhật `TestNewAppAndInit` xác nhận nút điều khiển `alwaysOnTopBtn` được khởi tạo chuẩn xác.
    + Thay thế `TestToggleAutoFailover` bằng `TestToggleAlwaysOnTop`:
      * Kiểm tra trạng thái khởi tạo: `alwaysOnTop == true`, nhãn `📌 Ghim: BẬT`, importance `HighImportance`.
      * Kiểm tra toggle TẮT: `alwaysOnTop == false`, nhãn `📌 Ghim: TẮT`, importance `MediumImportance`.
      * Kiểm tra toggle BẬT: `alwaysOnTop == true`, nhãn `📌 Ghim: BẬT`, importance `HighImportance`.
      * Kiểm tra gọi trực tiếp `SetAlwaysOnTop(false)` và `SetAlwaysOnTop(true)`.
      * Kiểm tra gọi thông qua interface `Window.SetAlwaysOnTop` trên `app.Window()`.
- **Forbidden Legacy Fallback**: Không còn nút `Auto-Failover`, không còn biến/hàm `autoFailover`.
- **Residual Unverified Surfaces**: `none`

---

## 3. Verification Evidence

### 3.1. Unit Test Verification (`internal/ui`)
Lệnh thực thi:
```powershell
go test -tags gui -v ./internal/ui/...
```
Kết quả:
```text
=== RUN   TestNewAppAndInit
--- PASS: TestNewAppAndInit (0.10s)
=== RUN   TestToggleAlwaysOnTop
--- PASS: TestToggleAlwaysOnTop (0.03s)
=== RUN   TestUpdateUI_WithAccounts
--- PASS: TestUpdateUI_WithAccounts (0.03s)
=== RUN   TestUpdateUI_Empty
--- PASS: TestUpdateUI_Empty (0.03s)
=== RUN   TestNewAccountCard_Callbacks
--- PASS: TestNewAccountCard_Callbacks (0.00s)
=== RUN   TestStatusDot
--- PASS: TestStatusDot (0.00s)
=== RUN   TestAppLogoResourceAndWindowIcon
--- PASS: TestAppLogoResourceAndWindowIcon (0.02s)
=== RUN   TestElevationTokensAndBorders
--- PASS: TestElevationTokensAndBorders (0.00s)
PASS
ok  	antigravity-switch/internal/ui	1.429s
```

### 3.2. Toàn Bộ Test Suite Repo (`go test ./...`)
Lệnh thực thi:
```powershell
go test ./...
```
Kết quả:
```text
?   	antigravity-switch/cmd/switcher	[no test files]
ok  	antigravity-switch/internal/connector	(cached)
ok  	antigravity-switch/internal/manager	4.859s
?   	antigravity-switch/internal/model	[no test files]
ok  	antigravity-switch/internal/reloader	(cached)
ok  	antigravity-switch/internal/scanner	(cached)
ok  	antigravity-switch/internal/store	16.384s
ok  	antigravity-switch/internal/ui	1.789s
ok  	antigravity-switch/internal/wincred	(cached)
```

### 3.3. Windows Build Pipeline (`scripts/build_windows.ps1 -Mode all`)
Lệnh thực thi:
```powershell
powershell -File scripts/build_windows.ps1 -Mode all
```
Kết quả:
```text
================================================================================
⚡ Antigravity Multi-Account Switcher - Windows Build System
   Mode: all | OutDir: bin | SkipTests: False | Clean: False
================================================================================
[2/4] Running pre-flight tests & vet...
  -> Verifying pure Go packages (CGO_ENABLED=0)...
  [PASS] Pure Go verification succeeded.
  -> Verifying GUI packages with CGo & Fyne (CGO_ENABLED=1)...
  [PASS] GUI verification succeeded.
[3/4] Compiling target binaries...
  -> Checking Windows application icon and resource...
  -> Compiling resource with windres: cmd\switcher\app.rc -> cmd\switcher\rsrc_windows_amd64.syso
  [SUCCESS] Compiled PE resource object: cmd\switcher\rsrc_windows_amd64.syso (12552 bytes)
  -> Building Standalone CLI Binary (Pure Go, CGO_ENABLED=0)...
  [SUCCESS] Created: bin\antigravity-switch.exe
  Compile Duration: 2279 ms | Size: 6.73 MB (7058944 bytes)
  -> Building Standalone GUI Binary (Fyne, CGO_ENABLED=1)...
  [SUCCESS] Created: bin\antigravity-switch-gui.exe
  Compile Duration: 5116 ms | Size: 23.56 MB (24704512 bytes)
[4/4] Executing quick verification...
  [PASS] Embedded PE application icon verified on bin\antigravity-switch.exe (32x32)
  [PASS] Embedded PE application icon verified on bin\antigravity-switch-gui.exe (32x32)
  -> Running 'bin\antigravity-switch.exe --status'...
================================================================================
⚡ ANTIGRAVITY SESSION STATUS
================================================================================
Active Email:    (No active session detected)
Language Server: Running (PID: 28580 | Port: 51569 | CSRF: Present)
Win32 Credential: None
================================================================================
================================================================================
⚡ BUILD SUCCESSFUL
================================================================================
```

---

## 4. Kết Luận & Bàn Giao
- Vấn đề 5 (P1-02) đã được giải quyết triệt để, khôi phục 100% tính năng Always on Top của Python prototype và dọn dẹp sạch sẽ các thành phần tự chế không được duyệt.
- Cả hai phiên bản thực thi (CLI binary `antigravity-switch.exe` và GUI binary `antigravity-switch-gui.exe`) đều biên dịch thành công, test pass 100%.
- Sẵn sàng bàn giao cho Supervisor Lane nghiệm thu độc lập.