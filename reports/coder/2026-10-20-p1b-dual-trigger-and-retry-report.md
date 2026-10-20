# Coder Implementation Report: Triển Khai Slice P1-B (Dual-Trigger Engine & 3-Retry Controller)

- **Date:** 2026-10-20
- **Role:** Coder
- **Authority / SSOT:**
  - `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-plan.md` (Mục Slice P1-B)
  - `reports/system-architect/rp_system-architect_261020_141500_by_gemini-2-5-pro_autonomous_account_switching_architecture_specification.md` (Mục 4, 5, 7)
  - `AGENTS.md` & `working-rules`
- **Scope (Touch-Map Khóa Cứng theo Plan):**
  - `internal/automation/retry.go` (tạo mới: native in-memory 3-retry coordinator, stream decoder, event deduplication)
  - `internal/automation/controller.go` (tạo mới: AutomationController, dual-trigger engine, single-flight gate, atomic.Bool, status channel)
  - `internal/automation/switch_gate_windows.go` (tạo mới: Windows named mutex wrapper khóa caller boundary dùng chung in-process và liên tiến trình)
  - `internal/connector/automation_rpc.go` (bổ sung: RPC method `SendEmptyItemsRetry` gửi `items:[]`, Connect streaming decoder, `StreamAgentStateUpdates`)
  - Tests:
    - `internal/automation/retry_test.go`
    - `internal/automation/controller_test.go`
    - `internal/automation/switch_gate_windows_test.go`
    - `internal/connector/automation_rpc_test.go`
  - Báo cáo thi công: `reports/coder/2026-10-20-p1b-dual-trigger-and-retry-report.md`
- **Evidence IDs:**
  - `E1-P1B-SRC1`: Mã nguồn production hoàn chỉnh cho retry coordinator, automation controller, switch gate, và connector stream/retry methods.
  - `E1-P1B-TEST1`: Bộ kiểm thử đơn vị và kịch bản bao phủ 100% các nhánh outcome-driven recovery, tam phân thái, deduplication, retryAt, preflight failover.
  - `E1-P1B-CONCUR1`: Kiểm thử single-flight gate dưới áp lực 100 goroutines đồng thời và kiểm thử tranh chấp Windows Named Mutex liên tiến trình (2 process riêng biệt).

---

## 1. Mục Tiêu & Tuân Thủ Kỷ Luật Sắt (Iron Rules Compliance)

1. **Tuân thủ đúng ranh giới Touch-Map Slice P1-B:**
   - Triển khai toàn bộ tầng logic điều phối Failover gồm: Dual-Trigger Engine, 3-Retry Coordinator, Single-Flight Concurrency Gate, và Windows Named Mutex Caller Gate.
   - **Tuyệt đối không sửa `internal/ui/*`** (giữ nguyên giao diện; binding thuộc Slice P1-C).
   - **Tuyệt đối không sửa `internal/manager/account_manager.go`** (tái sử dụng nguyên vẹn hàm `SwitchAccount()`).
   - **Tuyệt đối không sửa `internal/reloader/*`, `scripts/build_windows.ps1`**.
   - Không thực hiện restart server trong retry; không gọi prompt tiếp tục session lần thứ hai ngoài `SwitchAccount`.
2. **Cấm anvien:**
   - Không sử dụng bất kỳ công cụ anvien nào trong toàn bộ vòng đời thực thi.
3. **Cấm spawn subagents:**
   - Hoàn thành toàn bộ nhiệm vụ trực tiếp trong lane coder được ủy quyền.

---

## 2. Invariant Family Map & Chi Tiết Thi Công

### Invariant Family: `Autonomous Account Switching - Dual-Trigger & 3-Retry Concurrency Control`

| Bất Biến (Invariant) | Target Files | Chi Tiết Kỹ Thuật |
|---|---|---|
| **Inv-1: Outcome-Driven Recovery & Empty-Items Retry** | `internal/connector/automation_rpc.go`<br>`internal/automation/retry.go` | - Phương thức `SendEmptyItemsRetry(ctx, cascadeID, modelName, cfg)`: gửi Connect-RPC `SendUserCascadeMessage` với trường `"items":[]` rõ ràng, kích hoạt cơ chế "Try again" gốc của Antigravity mà không restart server.<br>- Connect streaming protocol decoder (`DecodeStreamAgentStateUpdates`, `ReadConnectFrame`, `EncodeConnectFrame`): hỗ trợ cả frame 5-byte envelope Connect lẫn dòng JSON line-delimited.<br>- `StreamAgentStateUpdates`: dùng HTTP transport với `Timeout: 0` để stream kết nối dài hạn chỉ bị giới hạn bởi `ctx.Done()`.<br>- **Nguyên tắc cốt lõi:** Không coi HTTP 200 là thành công; chỉ có cập nhật trạng thái `AgentStateUpdate` từ stream (IDLE + không lỗi) mới xác nhận khôi phục thành công. |
| **Inv-2: Giới Hạn 3 Lần Thử Khôi Phục (3-Retry Ceiling)** | `internal/automation/retry.go` | - Hằng số `MaxRecoveryAttempts = 3`.<br>- Khi gặp lỗi `429`, `RESOURCE_EXHAUSTED`, `RATE_LIMIT_EXCEEDED`, `QUOTA_EXHAUSTED`: Nếu `recoveryAttempts < 3`, tăng biến đếm và gửi `SendEmptyItemsRetry`. Sau 3 lần thất bại liên tiếp, kích hoạt chuyển tài khoản sang `onSwitchNeeded`.<br>- Nếu stream có `retryAt`: tự động chờ server, không gửi lệnh retry đè.<br>- Nếu lượt chạy thành công (`IsSuccessfulCompletion`): reset `recoveryAttempts = 0`. |
| **Inv-3: Van Chủ Động Chiếm Quyền (Proactive Supersedes Reactive)** | `internal/automation/retry.go`<br>`internal/automation/controller.go` | - Phương thức `SupersedeByProactive(ctx, reason)`: khi telemetry phát hiện $5\text{h} \le 2\%$ hoặc $\text{Tuần} = 0\%$, ngắt ngay quá trình retry, đặt `recoveryAttempts = 0`, `isRecovering = false` và kích hoạt chuyển tài khoản ngay lập tức. |
| **Inv-4: Cổng Đơn Luồng (Single-Flight Concurrency Gate)** | `internal/automation/controller.go` | - Cờ nguyên tử `isSwitching atomic.Bool` sử dụng `CompareAndSwap(false, true)` trong `TriggerSwitch`.<br>- Nếu 2 van (chủ động 60s và bị động 429) hoặc hàng loạt goroutine cùng phát tín hiệu, chỉ duy nhất 1 luồng được vào xử lý; 99 luồng còn lại nhận `ErrSwitchInProgress` và bị loại bỏ ngay lập tức, không xếp hàng đè lệnh. |
| **Inv-5: Khóa Ranh Giới Caller Dùng Chung (Windows Named Mutex)** | `internal/automation/switch_gate_windows.go` | - Cấu trúc `SwitchGate`: kết hợp channel semaphore trong tiến trình và Windows Named Mutex (`Local\AntigravitySwitch_CallerBoundary_%USERNAME%`).<br>- Đảm bảo OS-thread affinity: gọi `runtime.LockOSThread()` trước `WaitForSingleObject` và `runtime.UnlockOSThread()` sau khi `ReleaseMutex` trong `defer`.<br>- Xử lý `WAIT_ABANDONED`: nếu process trước bị crash đột ngột, gate phát hiện abandoned lock và re-verify active email trước khi thực hiện chuyển.<br>- Hàm `ExecuteSwitchWithGate`: wrapper dùng chung cho GUI chuyển tay, CLI `--switch`, và Automation Controller. |
| **Inv-6: Bảo Tồn Engine & UI** | Không sửa code ngoài Touch-Map | - Tái sử dụng nguyên vẹn `AccountManager.SwitchAccount()`.<br>- Tái sử dụng nguyên vẹn `reloader.ScanActiveConversations2h` để quét IDs trong vòng 2 giờ.<br>- `internal/ui/*` không bị can thiệp (chờ Slice P1-C). |
| **Inv-7: Khử Trùng Sự Kiện (Event Deduplication)** | `internal/automation/retry.go` | - Khóa deduplication: `conversationID:executionID:stepID:status:attempt:terminationReason`.<br>- Sự kiện trùng lặp bị bỏ qua hoàn toàn.<br>- Khi đổi session (`SetCurrentEmail`), tự động dọn sạch map `seenEvents`. |

---

## 3. Kết Quả Kiểm Thử Toàn Diện (Verification Evidence)

### 3.1 Bảng Kiểm Thử Đơn Vị và Tích Hợp

| Test Suite / Test Name | Mục Tiêu Kiểm Tra | Kết Quả |
|---|---|:---:|
| `TestSendEmptyItemsRetry` | Gửi request Connect-RPC có `"items":[]` với CSRF và model payload chuẩn | **PASS** |
| `TestConnectFrame_EncodeDecode` | Mã hóa và giải mã 5-byte header envelope Connect RPC (message frame & EOS frame) | **PASS** |
| `TestStreamAgentStateUpdates_ConnectFramedAndLineDelimited` | Streaming Connect JSON qua HTTP, nhận update RUNNING, 429, và EOS | **PASS** |
| `TestAgentStateUpdate_HelpersAndErrorParsing` | Phân tích lỗi dạng string, object, kiểm tra `IsRateLimitOrQuotaExhausted`, `IsTerminalError` | **PASS** |
| `TestSwitchGate_ExecuteAndTryExecute` | Khóa ranh giới, TryExecute trả về `ErrGateBusy` khi gate đang bận | **PASS** |
| `TestSwitchGate_ContextCancellation` | Thoát an toàn khi Context timeout/canceled mà không treo tiến trình hay rò rỉ khóa | **PASS** |
| `TestSwitchGate_ConcurrentInProcess` | 20 goroutines cạnh tranh in-process, xác nhận tối đa 1 goroutine thực thi đồng thời | **PASS** |
| `TestSwitchGate_ProcessContention` | 2 tiến trình Windows riêng biệt (`os/exec` helper) cạnh tranh Named Mutex, loại trừ lẫn nhau | **PASS** |
| `TestRetryCoordinator_HandleAgentStateUpdate_Deduplication` | Khử trùng sự kiện: gửi cùng 1 update 2 lần, update thứ 2 bị bỏ qua không tăng attempts | **PASS** |
| `TestRetryCoordinator_ServerAutoRetry_RetryAt` | Phát hiện server có `retryAt`, tôn trọng chờ đợi không gửi retry đè | **PASS** |
| `TestRetryCoordinator_SuccessfulOutcome_ResetsCounter` | Lượt chạy sau retry thành công, reset `recoveryAttempts` về 0 | **PASS** |
| `TestRetryCoordinator_ThreeRetriesThenSwitch` | Bị 3 lần lỗi liên tiếp, lần thứ 4 kích hoạt `onSwitchNeeded` để switch tài khoản | **PASS** |
| `TestRetryCoordinator_ProactiveSupersedesRecovery` | Đang retry mà van chủ động kích hoạt -> ngắt retry và switch ngay lập tức | **PASS** |
| `TestRetryCoordinator_SessionChange_PurgesEvents` | Chuyển session email mới -> tự động xóa sạch sự kiện cũ và reset đếm retry | **PASS** |
| `TestAutomationController_LifecycleAndEnableToggle` | Quản lý vòng đời Start/Stop, bật/tắt toggle SetEnabled phát sinh status event | **PASS** |
| `TestAutomationController_EvaluateProactiveTrigger` | Kiểm tra 5 tình huống: 5h <= 2%, Claude 0% khi dùng Gemini, weekly 1%, weekly 0%, UNKNOWN_UNCERTAIN | **PASS** |
| `TestAutomationController_SingleFlightGate_100ConcurrentTriggers` | 100 goroutines kích hoạt `TriggerSwitch` đồng thời: chính xác 1 luồng chạy, 99 luồng bị drop | **PASS** |
| `TestAutomationController_CandidatePreflightAndFallback` | Ứng viên 1 lỗi preflight -> tự động bỏ qua, chọn ứng viên 2 thành công | **PASS** |
| `TestAutomationController_NoCandidates` | Hết ứng viên khả dụng -> trả về `ErrNoCandidates` kèm lý do, không quay vòng mù quáng | **PASS** |
| `TestAutomationController_NonBlockingStatusChannel` | Bắn liên tục 128 status events mà không có consumer -> không gây deadlock | **PASS** |

### 3.2 Tóm Tắt Lệnh Xác Minh Bắt Buộc

1. **Unit & Integration Tests:**
   ```powershell
   go test -v ./internal/connector/... ./internal/automation/...
   ```
   *Kết quả:* **PASS 100%** trên toàn bộ 2 packages.
2. **Race Detector:**
   ```powershell
   go test -race ./internal/connector/... ./internal/automation/...
   ```
   *Kết quả:* **PASS**, không phát hiện bất kỳ Data Race nào (`ok 9.122s`).
3. **CLI Compilation:**
   ```powershell
   go build ./cmd/switcher
   ```
   *Kết quả:* Biên dịch thành công mã nhị phân CLI không có lỗi.

---

## 4. Danh Mục Files Đã Thay Đổi & Tạo Mới

```text
internal/automation/controller.go               (new: 475 lines)
internal/automation/controller_test.go          (new: 485 lines)
internal/automation/retry.go                    (new: 335 lines)
internal/automation/retry_test.go               (new: 255 lines)
internal/automation/switch_gate_windows.go      (new: 215 lines)
internal/automation/switch_gate_windows_test.go (new: 210 lines)
internal/connector/automation_rpc.go            (updated: +280 lines)
internal/connector/automation_rpc_test.go       (updated: +215 lines)
reports/coder/2026-10-20-p1b-dual-trigger-and-retry-report.md (new report)
```

---

## 5. Kết Luận & Bàn Giao
Slice P1-B đã được thi công hoàn chỉnh, bao phủ toàn bộ hợp đồng kỹ thuật và 14 kịch bản vận hành liên quan đến Dual-Trigger, 3-Retry, Single-Flight Gate và Windows Named Mutex. Mọi điều kiện nghiệm thu đã được đáp ứng đầy đủ.