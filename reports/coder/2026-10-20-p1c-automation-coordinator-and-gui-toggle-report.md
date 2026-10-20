# Coder Implementation Report: Triển Khai Slice P1-C (Automation Coordinator & GUI Toggle)

- **Date:** 2026-10-20
- **Role:** Coder
- **Authority / SSOT:**
  - `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-plan.md` (Mục Slice P1-C)
  - `reports/architect/2026-10-20-p1c-fast-block-resolution.md` (Văn kiện quyết định kiến trúc Addendum Touch-Map, Commit `bb14c66`)
  - `reports/system-architect/rp_system-architect_261020_141500_by_gemini-2-5-pro_autonomous_account_switching_architecture_specification.md` (Mục 5, 8, 9, 10)
  - `AGENTS.md` & `working-rules`
- **Scope (Touch-Map Khóa Cứng Tích Lũy kèm Addendum Kiến Trúc):**
  - `internal/ui/app.go` (khởi tạo controller lifecycle trong Init/Close, mở rộng callback duy nhất của autoLoop, bọc SwitchAccount qua SwitchGate và invalidate recovery)
  - `internal/ui/window.go` (nối toggle callback từ Row 2 Toolbar vào common setter SetAutoSwitchEnabled, đồng bộ thread-safe qua fyne.Do)
  - `cmd/switcher/main.go` (bọc nhánh CLI `--switch` qua wrapper dùng chung `automation.ExecuteSwitchWithGate`)
  - `internal/ui/ui_test.go` (bổ sung bộ test bao phủ: lifecycle, toggle binding, telemetry callback integration, status listener & status bar, manual switch gate & invalidation, gate mutual exclusion)
  - `cmd/switcher/main_test.go` (bổ sung unit tests cho CLI `--switch` qua SwitchGate và kiểm thử tranh chấp gate)
  - `internal/manager/account_manager_test.go` (Addendum kiến trúc: tiêm mock `rel.ServerDiscoverer` hermetic vào 3 hàm test: `TestSwitchAccount`, `TestSwitchAccount_AtomicAppStorageSync`, `TestSwitchAccount_CivilizedReAuth_HotSwapped`)
  - Báo cáo thi công: `reports/coder/2026-10-20-p1c-automation-coordinator-and-gui-toggle-report.md`
- **Evidence IDs:**
  - `E1-P1C-SRC1`: Mã nguồn production và test hoàn chỉnh cho App lifecycle, toggle binding, telemetry callback, caller gate wrapper và test hermetic mock.
  - `E1-P1C-BUILD1`: Toàn bộ hệ thống biên dịch thành công qua release build script `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all` (Exit Code 0), tạo đầy đủ `bin/antigravity-switch.exe` (10.53 MB) và `bin/antigravity-switch-gui.exe` (27.72 MB).
  - `E1-P1C-UI1`: Bộ kiểm thử UI pass 100% (`go test -v -tags gui ./internal/ui/...`), xác minh toggle hiện hữu trên Row 2 điều khiển controller, status bar hiển thị tiến trình, quota UI refresh liên tục không gián đoạn.
  - `E1-P1C-CALL1`: Bộ kiểm thử CLI và caller gate pass 100% (`go test -v ./cmd/switcher/...`), bảo đảm mutual exclusion giữa auto, GUI manual và CLI.

---

## 1. Mục Tiêu & Tuân Thủ Kỷ Luật Sắt (Iron Rules Compliance)

1. **Tuân thủ đúng ranh giới Touch-Map:**
   - Kết nối thành công `AutomationController` vào vòng đời `App.Init` và `App.Close`.
   - Mở rộng callback duy nhất của `AutoTelemetryLoop`: vừa cấp quota results cho `controller.OnTelemetryUpdate` (bất đồng bộ, non-blocking), vừa cập nhật giao diện `updateUI` thông qua `fyne.Do`.
   - Kết nối widget công tắc capsule `autoSwitchToggle` có sẵn trên Row 2 Toolbar vào `SetAutoSwitchEnabled` mà không tạo thêm bất kỳ widget/view thứ hai nào.
   - Bọc toàn bộ các entrypoint đổi tài khoản chuyển tay (GUI `App.SwitchAccount` và CLI `main.go --switch`) qua `automation.ExecuteSwitchWithGate` dùng chung Windows Named Mutex, triệt tiêu race condition và vô hiệu hóa (`Reset()`) recovery state khi người dùng đổi session chủ động.
   - Tuyệt đối không can thiệp vào `internal/manager/account_manager.go`, `internal/reloader/*`, `internal/ui/themes.go`, `internal/ui/tray.go`.
2. **Tuân thủ Addendum Kiến Trúc:**
   - Khi phát hiện lỗi môi trường ngoại vi trong `internal/manager/account_manager_test.go` làm nghẽn `build_windows.ps1`, Coder đã phát lệnh FAST BLOCK kịp thời về CEO Thread.
   - Sau khi System Architect ban hành văn kiện `reports/architect/2026-10-20-p1c-fast-block-resolution.md`, Coder chỉ can thiệp phẫu thuật đúng 3 vị trí tiêm mock `rel.ServerDiscoverer` trong `account_manager_test.go`, khôi phục tính hermetic cho test harness.
3. **Cấm anvien & cấm spawn subagents:**
   - Tuyệt đối không sử dụng công cụ anvien và không mở thêm lane phụ.

---

## 2. Invariant Family Map & Chi Tiết Thi Công

### Invariant Family: `Autonomous Account Switching - Automation Coordinator & UI/CLI Gate Binding`

| Bất Biến (Invariant) | Target Files | Chi Tiết Kỹ Thuật |
|---|---|---|
| **Inv-1: Controller Lifecycle Ownership** | `internal/ui/app.go` | - `NewAppWithFyne`: Khởi tạo `AutomationController` chia sẻ chung `AccountManager`, `TelemetryPool`, và `QuotaCache`.<br>- `App.Init`: Kích hoạt `controller.Start(statusCtx)` với context hủy theo vòng đời ứng dụng; khởi động goroutine lắng nghe `controller.StatusChannel()` không chặn.<br>- `App.Close`: Hủy `statusCancel`, gọi `controller.Stop()`, dừng `autoLoop`, giải phóng timers. |
| **Inv-2: Single Telemetry Callback Extension** | `internal/ui/app.go` | - Không tạo `TelemetryPool` hoặc ticker thứ hai.<br>- Callback duy nhất trong `autoLoop.SetUpdateCallback` được mở rộng:<br>  1. Cấp dữ liệu quota cho automation: `go c.OnTelemetryUpdate(results)` (chạy nền, không chặn chu kỳ quét).<br>  2. Cập nhật giao diện: gọi `fyne.Do` cập nhật `lastQuotaRefresh` và `updateUI(overview)`. |
| **Inv-3: Common Toggle Setter & Non-Recursive Sync** | `internal/ui/app.go`<br>`internal/ui/window.go` | - `SetAutoSwitchEnabled(enabled bool)` đóng vai trò là setter duy nhất quản trị trạng thái automation.<br>- Khi bật (ON): gọi `controller.SetEnabled(true)` kích hoạt kiểm tra quota tức thì từ cache.<br>- Khi tắt (OFF): gọi `controller.SetEnabled(false)` chuyển về IDLE mà không ngắt switch đang chạy dở.<br>- Cơ chế chống đệ quy: kiểm tra `a.autoSwitchEnabled == enabled` và `toggle.IsOn() != enabled` trước khi cập nhật widget/controller.<br>- `window.go` callback của `autoSwitchToggle` gọi thẳng `a.SetAutoSwitchEnabled(isOn)`. |
| **Inv-4: Real-time Status Feedback & Overview Auto-Refresh** | `internal/ui/app.go` | - Phương thức `handleAutomationStatus(evt)` phân loại trạng thái và hiển thị lên `statusBar`:<br>  - `StatusSwitching`: hiện `⏳ Đang chuyển đổi...`<br>  - `StatusRetrying`: hiện `🔄 Đang thử lại...`<br>  - `StatusEvaluating`: hiện `🔍 Đang đánh giá...`<br>  - `StatusError`: hiện `❌ Lỗi...` (tự ẩn sau 5s)<br>  - `StatusNoCandidate`: hiện `⚠️ Không có ứng viên...` (tự ẩn sau 4s)<br>  - `StatusSwitched`: hiện `⚡ Đã chuyển thành công...` và tự động kích hoạt goroutine chạy ngầm `mgr.GetAccountStates()` -> `fyne.Do(updateUI)` để làm mới toàn bộ thẻ tài khoản trong RAM. |
| **Inv-5: Shared Caller Boundary Guard (GUI & CLI)** | `internal/ui/app.go`<br>`cmd/switcher/main.go` | - `App.SwitchAccount`: gọi `automation.ExecuteSwitchWithGate(ctx, gate, mgr, targetEmail, false)`.<br>- `cmd/switcher/main.go` nhánh `--switch`: bọc qua `automation.ExecuteSwitchWithGate`.<br>- Triệt tiêu hoàn toàn đường gọi trực tiếp `mgr.SwitchAccount` từ callers bên ngoài.<br>- Invalidation: Khi manual switch thành công, gọi `controller.RetryCoordinator().Reset()` và `SetCurrentEmail(targetEmail)` để vô hiệu hóa retry cũ của session trước. |
| **Inv-6: Test Hermeticity Restoration** | `internal/manager/account_manager_test.go` | - Tiêm mock `rel.ServerDiscoverer` vào 3 test hàm: `TestSwitchAccount`, `TestSwitchAccount_AtomicAppStorageSync`, `TestSwitchAccount_CivilizedReAuth_HotSwapped`.<br>- Trả về loopback port của `httptest.NewServer` nội bộ, loại bỏ hoàn toàn sự phụ thuộc vào tiến trình OS bên ngoài. |

---

## 3. Kết Quả Kiểm Thử & Xác Minh Bắt Buộc (Verification Evidence)

### 3.1. Toàn Bộ Test Suites Đạt PASS 100%

1. **Unit Tests GUI (`internal/ui`):**
   ```powershell
   go test -v -tags gui ./internal/ui/...
   ```
   *Kết quả:* **PASS** (tất cả 36 tests pass, bao gồm 6 tests mới của P1-C trong `ui_test.go`).
   - `TestAutomation_AppLifecycleAndController`: PASS
   - `TestAutomation_ToggleAndSetterBinding`: PASS
   - `TestAutomation_TelemetryCallbackIntegration`: PASS
   - `TestAutomation_StatusEventListenerAndStatusBar`: PASS
   - `TestAutomation_ManualSwitch_GateAndInvalidation`: PASS
   - `TestAutomation_GateMutualExclusion`: PASS

2. **Unit Tests CLI (`cmd/switcher`):**
   ```powershell
   go test -v ./cmd/switcher/...
   ```
   *Kết quả:* **PASS** (tất cả 6 tests pass, bao gồm 2 tests mới của P1-C).
   - `TestCLI_Switch_WithGate`: PASS
   - `TestCLI_Switch_GateContention`: PASS

3. **Unit Tests Core Manager (`internal/manager`):**
   ```powershell
   go test -v ./internal/manager/...
   ```
   *Kết quả:* **PASS** (tất cả 18 tests pass sau khi tiêm hermetic mock `rel.ServerDiscoverer`).

4. **Unit Tests Automation Coordinator (`internal/automation`):**
   ```powershell
   go test -v ./internal/automation/...
   ```
   *Kết quả:* **PASS** (tất cả 20 tests pass 100%).

### 3.2. Full Build Script Verification (`scripts/build_windows.ps1 -Mode all`)

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all
```
*Kết quả:* **BUILD SUCCESSFUL (Exit Code 0)**.
- Giai đoạn Pre-flight Pure Go (`CGO_ENABLED=0`): PASS toàn bộ các package `model`, `wincred`, `store`, `scanner`, `connector`, `reloader`, `manager`, `cmd/switcher`.
- Giai đoạn Pre-flight GUI (`CGO_ENABLED=1`): PASS toàn bộ `cmd/switcher` và `internal/ui`.
- Giai đoạn Biên dịch (Compilation):
  - `bin\antigravity-switch.exe` (Standalone CLI): 10.53 MB (11,036,672 bytes)
  - `bin\antigravity-switch-gui.exe` (Standalone GUI): 27.72 MB (29,067,776 bytes)
- Giai đoạn Quick Verification:
  - Khớp icon nhúng PE 32x32 trên cả 2 tệp nhị phân.
  - Chạy `bin\antigravity-switch.exe --status` hiển thị trạng thái hoàn hảo.

---

## 4. Danh Mục Files Đã Thay Đổi (Touch-Map Inventory)

```text
cmd/switcher/main.go                    (bọc nhánh --switch qua ExecuteSwitchWithGate)
cmd/switcher/main_test.go               (thêm tests CLI SwitchGate và Contention)
internal/manager/account_manager_test.go (tiêm hermetic mock rel.ServerDiscoverer cho 3 tests)
internal/ui/app.go                      (lifecycle, status listener, telemetry callback, gate wrapper)
internal/ui/ui_test.go                  (thêm 6 tests P1-C, sửa ServerDiscoverer mock)
internal/ui/window.go                   (nối autoSwitchToggle callback vào SetAutoSwitchEnabled)
reports/coder/2026-10-20-p1c-automation-coordinator-and-gui-toggle-report.md (báo cáo thi công mới)
```

---

## 5. Kết Luận & Bàn Giao

Toàn bộ các yêu cầu của Slice P1-C đã được thi công chính xác, tuân thủ nghiêm ngặt văn kiện quyết định kiến trúc `reports/architect/2026-10-20-p1c-fast-block-resolution.md`. Mọi ranh giới kiểm thử từ cấp độ đơn vị đến bản build tổng thể Windows đều đạt tiêu chuẩn nghiệm thu 100%.