# Coder Report: Fix GUI Account Switching Flow UX (Zero Modal Dialogs) & Console Flashing Elimination (CREATE_NO_WINDOW)

## 1. Metadata
- **Date**: 2026-10-16
- **Task ID**: FIX-UX-MODAL-AND-CONSOLE-FLASHING
- **Author**: Coder Lane (gpt-6.1-sol)
- **Target Invariant**: Khắc phục triệt để 02 lỗi UX bị Supervisor REJECT theo báo cáo `reports/supervisor/2026-10-16-gui-switch-flow-and-ux-supervisor-review.md`:
  1. Loại bỏ 100% modal popup thành công `dialog.ShowInformation` khi chuyển đổi tài khoản và lưu phiên trong Fyne GUI, đồng bộ hóa tuyệt đối với chuẩn Python SSOT (`ui/app_window.py:508-509`), đảm bảo trải nghiệm người dùng liền mạch không bị chặn (uninterrupted flow).
  2. Triệt tiêu hoàn toàn hiện tượng cửa sổ console màu đen nhấp nháy trên Windows bằng cách thiết lập cờ `SysProcAttr` với `CreationFlags = windows.CREATE_NO_WINDOW` (`0x08000000`) và `HideWindow = true` cho toàn bộ các lệnh `exec.Command` / `exec.CommandContext` (`powershell`, `netstat`, `taskkill`).
  3. Biên dịch lại toàn bộ binary Windows (`scripts/build_windows.ps1 -Mode all`) và kiểm tra tính toàn vẹn của ứng dụng.
- **Authority / SSOT**:
  - Supervisor Rejection Report: `reports/supervisor/2026-10-16-gui-switch-flow-and-ux-supervisor-review.md`.
  - Python SSOT: `ui/app_window.py:508-509, 617-624` (Purged `messagebox.showinfo`, zero blocking modal popups; silent status bar feedback).
  - Python SSOT: `core/session_reloader.py:86-88, 325-327` (`subprocess.CREATE_NO_WINDOW`).
  - Win32 Process Creation Flags: `CREATE_NO_WINDOW = 0x08000000`, `SysProcAttr.HideWindow = true`.

---

## 2. Invariant Family Map
- **Family Name**: `SilentBackgroundExecutionAndUninterruptedUX`
- **SSOT**: Python prototype `ui/app_window.py` + `core/session_reloader.py` + Microsoft Win32 Process Creation Subsystem.
- **Root Cause Analysis**:
  1. **UX Blocking Modals**: `internal/ui/app.go` (trước đây tại dòng 197 và 235) gọi `dialog.ShowInformation("Chuyển thành công", ...)` và `dialog.ShowInformation("Lưu phiên", ...)`. Modal này hiển thị lớp phủ overlay chặn người dùng và bắt buộc phải ấn OK mới tiếp tục được, vi phạm nguyên lý Zero Blocking Modal của Python SSOT (`ui/app_window.py:508-509`).
  2. **Console Flashing**: Windows console subsystem tự động phân bổ và hiển thị cửa sổ console đen khi các tiến trình console (`powershell.exe`, `netstat.exe`, `taskkill.exe`) được sinh ra mà không có thuộc tính `SysProcAttr` chỉ định `CreationFlags: CREATE_NO_WINDOW` và `HideWindow: true`.
- **Surgical Changes Applied**:
  - `internal/ui/app.go`:
    + Xóa bỏ hoàn toàn 2 khối lệnh `dialog.ShowInformation` tại luồng thành công của `SwitchAccount()` và `SaveSession()`.
    + Giữ nguyên `dialog.ShowError(err, a.win)` cho các trường hợp lỗi thực sự theo đúng chỉ đạo của Supervisor.
    + Tái cấu trúc luồng cập nhật UI sau khi chuyển đổi / lưu phiên thành công: gọi `a.mgr.GetAccountStates(ctx)` để cập nhật RAM state, gọi `a.updateUI(overview)` để refresh visual card badges/dots, sau đó đặt thông điệp thành công chi tiết trực tiếp lên `a.statusBarLabel` (ví dụ: `Đã chuyển sang <email> (<duration>)` hoặc `Đã lưu phiên của <email> vào kho JSON`). Điều này ngăn ngừa hiện tượng status bar bị chu kỳ `Refresh()` mặc định ghi đè thành `Đồng bộ lúc...`.
  - `internal/connector/discovery.go`:
    + Bổ sung cấu hình `cmd.SysProcAttr = &windows.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}` cho lệnh `powershell.exe` trong `QueryLanguageServerProcesses()`.
    + Bổ sung cấu hình `cmd.SysProcAttr = &windows.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}` cho lệnh `netstat.exe` trong `discoverPortsViaNetstat()`.
  - `internal/reloader/session_reloader.go`:
    + Bổ sung thuộc tính `HideWindow: true` cùng với `CreationFlags: windows.CREATE_NO_WINDOW` cho lệnh `taskkill.exe` trong `DefaultKillLanguageServer()`.
  - `internal/ui/ui_test.go`:
    + Bổ sung 2 automated regression unit tests: `TestSwitchAccount_NoModalPopup` và `TestSaveSession_NoModalPopup`.
    + Kiểm tra trực tiếp xác nhận: status bar cập nhật chính xác thông điệp thành công, và `app.win.Canvas().Overlays().Top() == nil` (chứng minh 0 modal popup xuất hiện trên UI canvas).
    + Cập nhật helper `waitForIdle()` để đồng bộ hóa an toàn vòng đời Fyne goroutine trong môi trường test Windows, ngăn ngừa file lock trên thư mục tạm.
  - `internal/reloader/reloader_test.go`:
    + Bổ sung test `TestDefaultKillLanguageServer` kiểm tra việc thực thi `taskkill` ngầm với `SysProcAttr` an toàn không phát sinh lỗi.
- **Sibling Runtime Surfaces Checked Across Repo**:
  - Rà soát toàn bộ các lệnh `exec.Command` / `exec.CommandContext` trên toàn bộ repository: chính xác chỉ có 3 điểm gọi (`discovery.go` x2, `session_reloader.go` x1), tất cả đều đã được áp dụng `HideWindow: true` và `CreationFlags: windows.CREATE_NO_WINDOW`.
  - Rà soát toàn bộ các lệnh `dialog.Show*` trong `internal/ui/`: 2 dialog thông tin thành công đã bị xóa hoàn toàn, 2 dialog báo lỗi được giữ lại theo spec.
  - `internal/ui/window.go` và `internal/ui/tray.go`: Kiểm tra và xác nhận không có bất kỳ dialog nào.
- **Forbidden Legacy Fallback**: Không còn bất kỳ tiến trình console nào chạy mà không có cờ `CREATE_NO_WINDOW` / `HideWindow`.
- **Residual Unverified Surfaces**: `none`.

---

## 3. Verification Evidence

### 3.1. Go Test Suite Execution

#### Pure Go Suite (CGO_ENABLED=0):
```text
ok      antigravity-switch/internal/wincred     (cached)
ok      antigravity-switch/internal/store       (cached)
ok      antigravity-switch/internal/scanner     (cached)
ok      antigravity-switch/internal/connector   (cached)
ok      antigravity-switch/internal/reloader    2.245s
ok      antigravity-switch/internal/manager     3.826s
```

#### GUI Test Suite with CGo & Fyne (CGO_ENABLED=1):
```text
=== RUN   TestNewAppAndInit
--- PASS: TestNewAppAndInit (0.17s)
=== RUN   TestToggleAlwaysOnTop
--- PASS: TestToggleAlwaysOnTop (0.10s)
=== RUN   TestUpdateUI_WithAccounts
--- PASS: TestUpdateUI_WithAccounts (0.06s)
=== RUN   TestUpdateUI_Empty
--- PASS: TestUpdateUI_Empty (0.05s)
=== RUN   TestNewAccountCard_Callbacks
--- PASS: TestNewAccountCard_Callbacks (0.00s)
=== RUN   TestStatusDot
--- PASS: TestStatusDot (0.00s)
=== RUN   TestAppLogoResourceAndWindowIcon
--- PASS: TestAppLogoResourceAndWindowIcon (0.10s)
=== RUN   TestElevationTokensAndBorders
--- PASS: TestElevationTokensAndBorders (0.00s)
=== RUN   TestSwitchAccount_NoModalPopup
--- PASS: TestSwitchAccount_NoModalPopup (0.28s)
=== RUN   TestSaveSession_NoModalPopup
--- PASS: TestSaveSession_NoModalPopup (0.20s)
PASS
ok      antigravity-switch/internal/ui  2.132s
```

### 3.2. Windows Build Automation (`scripts/build_windows.ps1 -Mode all`)
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
  Compile Duration: 2170 ms | Size: 6.73 MB (7059456 bytes)
  -> Building Standalone GUI Binary (Fyne, CGO_ENABLED=1)...
  [SUCCESS] Created: bin\antigravity-switch-gui.exe
  Compile Duration: 5035 ms | Size: 23.54 MB (24686080 bytes)
[4/4] Executing quick verification...
  [PASS] Embedded PE application icon verified on bin\antigravity-switch.exe (32x32)
  [PASS] Embedded PE application icon verified on bin\antigravity-switch-gui.exe (32x32)
  -> Running 'bin\antigravity-switch.exe --status'...
================================================================================
⚡ ANTIGRAVITY SESSION STATUS
================================================================================
Active Email:    daohien204ccyersin@gmail.com
Language Server: Running (PID: 1284 | Port: 63274 | CSRF: Present)
Win32 Credential: Present (260 bytes access token)
================================================================================
================================================================================
⚡ BUILD SUCCESSFUL
================================================================================
```

### 3.3. GUI Process Runtime Integrity
```text
GUI Process ID: 17012 Alive: True
GUI process stopped successfully.
```

---

## 4. Acceptance Checklist vs Supervisor Directive
- [x] **Modal Popup Purged**: Xóa bỏ `dialog.ShowInformation` tại dòng 197 và 235 của `internal/ui/app.go`. Trải nghiệm chuyển đổi tài khoản và lưu phiên không còn bị modal chặn.
- [x] **Silent Windows Execution**: Áp dụng `HideWindow: true` và `CreationFlags: windows.CREATE_NO_WINDOW` cho tất cả các tiến trình `powershell`, `netstat`, `taskkill` trong `internal/connector/discovery.go` và `internal/reloader/session_reloader.go`. 0 cửa sổ console hiển thị khi chạy.
- [x] **Python SSOT Parity**: Cập nhật trạng thái card trong RAM và hiển thị thông điệp trên `statusBarLabel` nguyên vẹn theo chuẩn `ui/app_window.py:508-509`.
- [x] **Full Windows Rebuild**: Script `scripts/build_windows.ps1 -Mode all` vượt qua toàn bộ các bước vet, test, icon linking và build ra 2 file exe hoàn chỉnh.
- [x] **Test Coverage**: Bổ sung unit tests kiểm chứng canvas overlays không có modal và tiến trình thực thi êm thuận.

Verdict: **READY_FOR_SUPERVISOR**