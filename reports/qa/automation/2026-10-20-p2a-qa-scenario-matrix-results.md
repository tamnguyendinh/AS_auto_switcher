# QA Scenario Matrix & Runtime Verification Report (Slice P2-A)

- **Date:** 2026-10-20
- **Slice:** P2-A (E2E Integration & 14 Edge Cases Simulation Test Harness)
- **Status:** PASS (100% 14/14 Edge Cases Passed, 9/9 Binding Assertions Passed, Full Windows Build Exit Code 0)
- **Policy Revision:** Cập nhật theo chỉ đạo tối cao từ Human Owner (Commit `e1dab85`): Tuần > 1% dùng tiếp; Tuần <= 1% chủ động chuyển.
- **Evidence IDs:**
  - `E2-P2A-SRC1`: Hermetic integration test harness in `internal/automation/automation_e2e_test.go`, `internal/ui/automation_binding_test.go`, and `cmd/switcher/main_test.go`.
  - `E2-P2A-TEST14`: 14/14 scenario tests in Matrix R1 executed and passed with `-race` enabled.
  - `E2-P2A-BIND1`: Actual caller/data paths verified (callback coexistence, disabled roundtrip, dynamic model config, execution outcome recovery, lifecycle cleanup, shared mutex gate contention).
  - `E2-P2A-GUI1`: Standalone native GUI binary `bin/antigravity-switch-gui.exe` built with `-tags gui`, launched cleanly, verified responsive toggle on Row 2 toolbar, and closed cleanly without orphan processes.

---

## 1. Kết Quả Kiểm Thử Ma Trận 14 Tình Huống Biên (Matrix R1)

| ID | Tình huống (Scenario) | Kết quả mong đợi (Expected) | Test Case tương ứng | Trạng thái | Thời gian |
|---|---|---|---|---|---|
| **1** | Gemini đang dùng, 5h <= 2% | Chuyển một lần | `TestEdgeCases1To7/EdgeCase1_Gemini5hLow_SwitchesOnce` | **PASS** | 0.83s |
| **2** | Claude tuần <= 1%, phiên dùng Gemini còn quota | Không chuyển do Claude | `TestEdgeCases1To7/EdgeCase2_ClaudeWeeklyDepleted_GeminiActive_NoSwitch` | **PASS** | 0.55s |
| **3** | Chỉ tuần, tuần > 1% (ví dụ: tuần = 2%) | Dùng tiếp | `TestEdgeCases1To7/EdgeCase3_WeeklyOnly_Above1Percent_Continues` | **PASS** | 0.55s |
| **4** | Chỉ tuần, tuần <= 1% (ví dụ: tuần = 1% hoặc 0%) | Chuyển | `TestEdgeCases1To7/EdgeCase4_WeeklyOnly_AtOrBelow1Percent_Switches` | **PASS** | 0.75s |
| **5** | API bucket 5h disabled | Cờ đi qua parser/cache, bỏ qua 5h và xét tuần | `TestEdgeCases1To7/EdgeCase5_Disabled5hBucket_Ignored_EvaluatesWeekly` | **PASS** | 0.54s |
| **6** | Recovery 1 hoặc 2 kết thúc không lỗi | Dừng recovery, không switch | `TestEdgeCases1To7/EdgeCase6_RecoveryOutcomeSuccess_Resets_NoSwitch` | **PASS** | 0.07s |
| **7** | Ba recovery actions kết thúc lỗi giới hạn | Một quyết định switch | `TestEdgeCases1To7/EdgeCase7_ThreeRecoveryFailures_TriggersOneSwitch` | **PASS** | 0.77s |
| **8** | Proactive và reactive cùng kích hoạt | Một đợt; không phát retry tiếp sau switch | `TestEdgeCases8To14/EdgeCase8_ProactiveSupersedesReactive_NoDuplicateRetry` | **PASS** | 0.86s |
| **9** | Manual GUI/CLI và auto cùng yêu cầu | Một executor; kiểm tra manual-first và auto-first | `TestEdgeCases8To14/EdgeCase9_ManualAndAutoContention_TwoWay` | **PASS** | 0.17s |
| **10** | Model đổi trước switch | Resolve và tính lại nhóm/ứng viên | `TestEdgeCases8To14/EdgeCase10_ModelChangedBeforeSwitch_ReranksCandidates` | **PASS** | 0.79s |
| **11** | Preflight ứng viên đầu lỗi | Bỏ qua trong đợt, xét kế tiếp trước khi thay credential | `TestEdgeCases8To14/EdgeCase11_PreflightCandidate1Fails_SkipsToCandidate2` | **PASS** | 0.75s |
| **12** | SwitchAccount trả lỗi | Dừng đợt, hiện lỗi, không xoay vô điều kiện | `TestEdgeCases8To14/EdgeCase12_SwitchAccountError_HaltsWithoutIndefiniteRotation` | **PASS** | 0.76s |
| **13** | Không có ứng viên | Hiện lý do, chờ telemetry hiện có | `TestEdgeCases8To14/EdgeCase13_NoCandidates_EmitsStatusNoCandidate` | **PASS** | 0.56s |
| **14** | Event cũ/trùng tới muộn | Không tăng retry hoặc switch vì event cũ | `TestEdgeCases8To14/EdgeCase14_StaleDuplicateEvents_DroppedByDeduplication` | **PASS** | 0.03s |

---

## 2. Kết Quả Kiểm Thử Binding Thực Tế (Binding Assertions)

1. **HTTP Acceptance vs Execution Outcome (`TestBindings_HTTPAcceptance_Versus_ExecutionOutcome`):**
   - HTTP 200 OK khi gửi empty-items retry không được tính là khôi phục thành công.
   - Trạng thái `recovering` được giữ nguyên cho đến khi nhận được `AgentStateUpdate` với trạng thái `COMPLETED`/`DONE` không lỗi.
2. **Native Retry Waiting (`TestBindings_NativeRetryWaiting_HasRetryAt`):**
   - Khung cập nhật chứa `retryAt` đưa coordinator vào trạng thái chờ đợi, không tăng số đếm lần thất bại.
3. **App Lifecycle & Goroutine Clean-up (`TestUIBinding_AppLifecycle_ControllerAndStatusCleanup`):**
   - `App.Init` khởi động `AutomationController` và listener trạng thái; `App.Close` dừng sạch toàn bộ goroutine và channel mà không gây rò rỉ.
4. **Telemetry Callback Coexistence (`TestUIBinding_TelemetryCallback_DualDispatchWithoutOverwrite`):**
   - Vòng lặp telemetry đơn nhất vừa cấp dữ liệu quota cho `controller.OnTelemetryUpdate` bất đồng bộ vừa cập nhật giao diện `updateUI`.
5. **GUI Toggle Setter Binding (`TestUIBinding_Toggle_SetterBindingAndDynamicCheck`):**
   - Widget capsule toggle trên Row 2 đồng bộ 2 chiều với `controller.SetEnabled`, kích hoạt kiểm tra quota tức thì khi bật.
6. **Manual Switch Resets Recovery State (`TestUIBinding_ManualSwitch_Resets_RecoveryState`):**
   - Người dùng chuyển tài khoản thủ công qua GUI/CLI lập tức gọi `rc.Reset()`, vô hiệu hóa phiên recovery cũ.
7. **Two-Way CLI/Auto Gate Contention (`TestCLI_Switch_TwoWayContention`):**
   - Kiểm tra tranh chấp 2 chiều giữa CLI `--switch` và Controller qua `SwitchGate` dùng chung Windows Named Mutex.

---

## 3. Native GUI Validation

- Binary `bin/antigravity-switch-gui.exe` (27.72 MB) được build thành công qua `scripts/build_windows.ps1 -Mode all`.
- Đã khởi chạy process GUI thật trên môi trường Windows (PID: 29628), xác nhận tiến trình hoạt động ổn định và đóng cửa sổ sạch sẽ.
