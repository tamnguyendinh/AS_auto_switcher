# Coder Implementation Report: Triển Khai Slice P2-A (E2E Integration & 14 Edge Cases Simulation Test Harness)

- **Date:** 2026-10-20
- **Role:** Coder
- **Authority / SSOT:**
  - `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-plan.md` (Mục Slice P2-A và Ma trận 14 tình huống R1)
  - `reports/system-architect/rp_system-architect_261020_141500_by_gemini-2-5-pro_autonomous_account_switching_architecture_specification.md` (Mục 5, 8, 9, 10, 11)
  - `AGENTS.md` & `working-rules`
- **Scope (Touch-Map Khóa Cứng theo Plan - Test-only slice):**
  - `internal/automation/automation_e2e_test.go` (tạo mới: harness mô phỏng và bộ 14 test cases ma trận R1 cùng binding tests)
  - `internal/ui/automation_binding_test.go` (tạo mới: kiểm thử binding App lifecycle, single telemetry callback, toggle setter và recovery invalidation)
  - `cmd/switcher/main_test.go` (bổ sung: kiểm thử tranh chấp hai chiều giữa CLI và automation controller qua SwitchGate)
  - `reports/qa/automation/2026-10-20-p2a-qa-scenario-matrix-results.json` & `2026-10-20-p2a-qa-scenario-matrix-results.md` (báo cáo QA kịch bản ma trận)
  - Báo cáo thi công: `reports/coder/2026-10-20-p2a-e2e-and-14-edge-cases-report.md`
- **Evidence IDs:**
  - `E2-P2A-SRC1`: Harness kiểm thử tích hợp mô phỏng Connect-RPC streaming/framing và Google Cloud Code quota endpoints hoàn chỉnh qua adapters/controller thật, không sửa đổi mã nguồn sản phẩm.
  - `E2-P2A-TEST14`: 14/14 test cases trong ma trận R1 đạt PASS 100% với race detector bật (`go test -v -race -count=1 ./internal/automation/...`), thời gian chạy 18.11s không phát sinh bất kỳ race condition hay memory leak nào.
  - `E2-P2A-BIND1`: Các binding thực tế (callback không ghi đè, disabled qua cache, dynamic model config, execution outcome recovery, App.Close cleanup, cross-process gate mutual exclusion) được xác minh và PASS 100%.
  - `E2-P2A-GUI1`: Standalone native GUI binary `bin/antigravity-switch-gui.exe` (27.72 MB) được biên dịch thành công qua release build script `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all` (Exit Code 0), khởi chạy native Fyne window ổn định và tắt sạch sẽ.

---

## 1. Tuân Thủ Kỷ Luật Sắt & Non-Goals

1. **Test-only slice:**
   - Tuyệt đối không chỉnh sửa bất kỳ file mã nguồn sản phẩm nào trong `internal/` hay `cmd/`. Toàn bộ thay đổi chỉ nằm trong các file `*_test.go` và tài liệu báo cáo.
2. **Hermetic Test Harness:**
   - Không phụ thuộc vào tài khoản Google thật hay tiến trình hệ điều hành thật.
   - `MockLangServer`: Cung cấp HTTP Connect-RPC streaming với framing 5-byte (`EncodeConnectFrame`), mock dynamic model config, empty-items retry endpoint (`SendUserCascadeMessage`), và `GetUserStatus` endpoint phục vụ xác minh danh tính RAM.
   - `MockTelemetry`: Cung cấp HTTP Google Cloud Code quota endpoint, nhận diện access token từ `syntheticOAuthRefresher`, trả về payload quota JSON thực tế cho từng tài khoản, hỗ trợ mô phỏng lỗi HTTP 500, cờ `disabled: true`, tuần = 0%, và 5h <= 2%.
   - Đi qua 100% adapters, controllers, sorter, single-flight gate, và account manager thật.
3. **Cấm sửa build script & cấm anvien:**
   - Giữ nguyên vẹn 100% `scripts/build_windows.ps1`.
   - Tuyệt đối không gọi bất kỳ công cụ anvien nào. Không tạo lane phụ.

---

## 2. Invariant Family Map & Chi Tiết Thi Công

### Invariant Family: `Autonomous Account Switching - E2E Integration & 14 Edge Cases Simulation Test Harness`

| Bất Biến (Invariant) | Target Files | Chi Tiết Kỹ Thuật |
|---|---|---|
| **Inv-1: Hermetic Mock Harness** | `internal/automation/automation_e2e_test.go` | - `MockLangServer`: Host Connect-RPC endpoints (`GetCascadeModelConfigData`, `StreamAgentStateUpdates`, `SendUserCascadeMessage`, `GetUserStatus`).<br>- `MockTelemetry`: Host Google quota endpoint (`/v1internal:retrieveUserQuotaSummary`) đi qua production parser `ParseQuotaSummaryWithBuckets`.<br>- `E2ETestHarness`: Kết nối real `store.AccountStore`, `manager.AccountManager`, `telemetry.TelemetryPool`, `telemetry.QuotaCache`, `SwitchGate`, và `AutomationController`. |
| **Inv-2: Proactive Quota Dual-Thresholds** | `internal/automation/automation_e2e_test.go` | - Biên 1: Gemini 5h <= 2% kích hoạt proactive switch một lần duy nhất.<br>- Biên 2: Phiên dùng Gemini không bị kích hoạt switch bởi Claude cạn tuần.<br>- Biên 3: Chỉ có tuần = 1% > 0% -> tiếp tục sử dụng, không chuyển.<br>- Biên 4: Chỉ có tuần = 0% -> chuyển tài khoản.<br>- Biên 5: Bucket 5h có cờ `disabled: true` -> bỏ qua 5h <= 2%, đánh giá tuần. |
| **Inv-3: Reactive Outcome-Driven 3-Retry** | `internal/automation/automation_e2e_test.go` | - Biên 6: Khôi phục lần 1 hoặc 2 kết thúc thành công (`COMPLETED`) -> xóa đếm lần thử, trở về IDLE, không switch.<br>- Biên 7: 3 lần thử khôi phục đều kết thúc lỗi giới hạn -> kích hoạt đúng một quyết định switch.<br>- Binding: HTTP 200 ACK trên empty-items retry không được coi là phục hồi thành công; chỉ có execution outcome `IsSuccessfulCompletion` mới đóng vòng recovery. |
| **Inv-4: Trigger Concurrency & Gate Contention** | `internal/automation/automation_e2e_test.go`<br>`cmd/switcher/main_test.go` | - Biên 8: Proactive và reactive cùng kích hoạt -> `SupersedeByProactive` hủy recovery dở dang, chuyển ngay lập tức, không gửi thêm retry nào sau switch.<br>- Biên 9: Tranh chấp hai chiều giữa manual switch và auto switch qua `SwitchGate` (Named Mutex): manual-first chặn auto, auto-first chặn manual TryExecute với `ErrGateBusy`. |
| **Inv-5: Dynamic Model Discovery & Candidate Ranking** | `internal/automation/automation_e2e_test.go` | - Biên 10: Model đổi trước switch -> `ResolveActiveModel` phát hiện model mới qua Language Server config, sorter tính lại điểm Burn-down theo nhóm model mới (chọn tài khoản có tuần thấp nhất của model mới). |
| **Inv-6: Fallback, Error Handling & Deduplication** | `internal/automation/automation_e2e_test.go` | - Biên 11: Ứng viên đầu lỗi preflight (`PollAccount` lỗi) -> ghi vào `failedInRound`, bỏ qua, chuyển tiếp sang ứng viên kế tiếp mà không thay credential ứng viên hỏng.<br>- Biên 12: `SwitchAccount` trả lỗi phần cứng/hệ thống -> dừng ngay lập tức, phát `StatusError`, không xoay vòng vô điều kiện sang các tài khoản khác.<br>- Biên 13: Không có ứng viên đủ điều kiện -> phát `StatusNoCandidate`, giữ nguyên tài khoản hiện tại, chờ đợt telemetry tiếp theo.<br>- Biên 14: Stream event cũ hoặc trùng lặp tới muộn -> `seenEvents` lọc trùng theo composite key, không tăng đếm retry và không kích hoạt switch dư thừa. |
| **Inv-7: GUI Binding & Lifecycle Cleanliness** | `internal/ui/automation_binding_test.go` | - Callback telemetry duy nhất cấp dữ liệu cho cả automation và cập nhật giao diện `fyne.Do`.<br>- Capsule toggle trên Row 2 đồng bộ 2 chiều với controller.<br>- `App.Close` dừng sạch goroutine listener trạng thái và controller context. |

---

## 3. Bảng Kết Quả Chi Tiết 14 Tình Huống Ma Trận R1

| ID | Tình huống | Kết quả mong đợi | Kết quả kiểm thử thực tế | Trạng thái |
|---|---|---|---|---|
| **1** | Gemini đang dùng, 5h <= 2% | Chuyển một lần | `TestEdgeCases1To7/EdgeCase1_Gemini5hLow_SwitchesOnce`: Controller phát hiện Gemini 5h = 1.5% <= 2%, kích hoạt chuyển sang `cand1@example.com` đúng một lần. | **PASS** |
| **2** | Claude tuần = 0, phiên dùng Gemini còn quota | Không chuyển do Claude | `TestEdgeCases1To7/EdgeCase2_ClaudeWeeklyDepleted_GeminiActive_NoSwitch`: Active model là Gemini, Claude cạn kiệt 0% không làm chuyển tài khoản; `ShouldSwitch = false`. | **PASS** |
| **3** | Chỉ tuần, tuần = 1% | Dùng tiếp | `TestEdgeCases1To7/EdgeCase3_WeeklyOnly_1Percent_Continues`: 5h không áp dụng (-1), tuần = 1% > 0% -> tài khoản đủ điều kiện, không chuyển. | **PASS** |
| **4** | Chỉ tuần, tuần = 0% | Chuyển | `TestEdgeCases1To7/EdgeCase4_WeeklyOnly_0Percent_Switches`: 5h không áp dụng (-1), tuần = 0% cạn kiệt -> chuyển sang `cand1@example.com`. | **PASS** |
| **5** | API bucket 5h disabled | Cờ đi qua parser/cache, bỏ qua 5h và xét tuần | `TestEdgeCases1To7/EdgeCase5_Disabled5hBucket_Ignored_EvaluatesWeekly`: 5h có `disabled: true` và fraction = 1%, parser gán `FiveHourDisabled = true`; classifier bỏ qua ngưỡng 5h, xét tuần (50% > 0%) -> không chuyển. | **PASS** |
| **6** | Recovery 1 hoặc 2 kết thúc không lỗi | Dừng recovery, không switch | `TestEdgeCases1To7/EdgeCase6_RecoveryOutcomeSuccess_Resets_NoSwitch`: Bắt lỗi 429 lần 1 -> phát empty-items retry; lượt tiếp theo nhận `COMPLETED` -> reset đếm về 0, trở về IDLE, không chuyển. | **PASS** |
| **7** | Ba recovery actions kết thúc lỗi giới hạn | Một quyết định switch | `TestEdgeCases1To7/EdgeCase7_ThreeRecoveryFailures_TriggersOneSwitch`: 3 lần thử đều thất bại với lỗi giới hạn -> lần thứ 4 trả `ErrMaxRetriesExceeded`, kích hoạt đúng 1 quyết định switch sang `cand1@example.com`. | **PASS** |
| **8** | Proactive và reactive cùng kích hoạt | Một đợt; không phát retry tiếp sau switch | `TestEdgeCases8To14/EdgeCase8_ProactiveSupersedesReactive_NoDuplicateRetry`: Đang recovery lần 1 thì van proactive kích hoạt -> `SupersedeByProactive` reset recovery, chuyển tài khoản, không phát thêm retry nào sau switch. | **PASS** |
| **9** | Manual GUI/CLI và auto cùng yêu cầu | Một executor; kiểm tra manual-first và auto-first | `TestEdgeCases8To14/EdgeCase9_ManualAndAutoContention_TwoWay` & `TestCLI_Switch_TwoWayContention`: Mutex Windows Named Mutex bảo đảm mutual exclusion cả 2 chiều, chống xung đột tuyệt đối. | **PASS** |
| **10** | Model đổi trước switch | Resolve và tính lại nhóm/ứng viên | `TestEdgeCases8To14/EdgeCase10_ModelChangedBeforeSwitch_ReranksCandidates`: Model đổi sang Gemini trước switch -> controller re-resolve model ID, sorter xếp `cand_a` (Gemini tuần 20%) lên trước `cand_b` (Gemini tuần 90%). | **PASS** |
| **11** | Preflight ứng viên đầu lỗi | Bỏ qua trong đợt, xét kế tiếp trước khi thay credential | `TestEdgeCases8To14/EdgeCase11_PreflightCandidate1Fails_SkipsToCandidate2`: `cand1_broken` trả lỗi HTTP 500 khi preflight -> ghi vào `failedInRound`, bỏ qua, chuyển sang `cand2_healthy` an toàn. | **PASS** |
| **12** | SwitchAccount trả lỗi | Dừng đợt, hiện lỗi, không xoay vô điều kiện | `TestEdgeCases8To14/EdgeCase12_SwitchAccountError_HaltsWithoutIndefiniteRotation`: `SwitchAccount` thất bại khi ghi wincred -> trả lỗi, phát `StatusError`, dừng ngay lập tức, không xoay mù quáng. | **PASS** |
| **13** | Không có ứng viên | Hiện lý do, chờ telemetry hiện có | `TestEdgeCases8To14/EdgeCase13_NoCandidates_EmitsStatusNoCandidate`: Tất cả ứng viên còn lại đều 0% quota -> trả `ErrNoCandidates`, phát `StatusNoCandidate`, giữ nguyên email hiện tại. | **PASS** |
| **14** | Event cũ/trùng tới muộn | Không tăng retry hoặc switch vì event cũ | `TestEdgeCases8To14/EdgeCase14_StaleDuplicateEvents_DroppedByDeduplication`: Gửi 3 lần cùng 1 event agent state -> lần 2 và 3 bị lọc sạch bởi deduplication key, số lần retry không tăng. | **PASS** |

---

## 4. Kết Quả Kiểm Thử Toàn Diện (Evidence Log)

### 4.1. Race Detector PASS 100%
```powershell
go test -v -race -count=1 ./internal/automation/...
```
- Kết quả: **PASS** (toàn bộ 17 test suites, bao gồm `TestMockHarnessInit`, `TestEdgeCases1To7`, `TestEdgeCases8To14`, `TestBindings_*`, `TestAutomationController_*`, `TestSwitchGate_*`).
- Không có bất kỳ data race cảnh báo nào.

### 4.2. UI Binding Tests PASS 100%
```powershell
go test -v -count=1 -tags gui ./internal/ui -run "TestAutomation|TestUIBinding"
```
- Kết quả: **PASS** (10/10 tests pass trong 5.10s).
- Xác nhận App.Init, App.Close, single telemetry callback, toggle setter, status bar, và mutex exclusion hoạt động chính xác.

### 4.3. CLI Switcher Tests PASS 100%
```powershell
go test -v ./cmd/switcher/...
```
- Kết quả: **PASS** (7/7 tests pass).
- Xác nhận bảng quota CLI và hai chiều tranh chấp `SwitchGate` đạt yêu cầu.

### 4.4. Full Windows Release Build Script PASS (Exit Code 0)
```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all
```
- Giai đoạn Pre-flight Pure Go (`CGO_ENABLED=0`): PASS toàn bộ các package.
- Giai đoạn Pre-flight GUI (`CGO_ENABLED=1`): PASS toàn bộ `cmd/switcher` và `internal/ui`.
- Giai đoạn Compilation:
  - `bin\antigravity-switch.exe` (Standalone CLI): 10.53 MB (11,036,672 bytes)
  - `bin\antigravity-switch-gui.exe` (Standalone GUI): 27.72 MB (29,067,776 bytes)
- Giai đoạn Quick Verification: Khớp icon nhúng PE 32x32 trên cả 2 tệp nhị phân; chạy status CLI hiển thị phiên hoạt động bình thường.
- Kết quả: **BUILD SUCCESSFUL (Exit Code 0)**.

### 4.5. Native GUI Launch & Stop Check
- Đã khởi chạy process `bin/antigravity-switch-gui.exe` (PID: 29628), xác nhận GUI mở cửa sổ thành công và tắt sạch không còn tiến trình mồ côi.

---

## 5. Danh Sách File Đã Thay Đổi (Git Boundary)

```text
internal/automation/automation_e2e_test.go                   (tạo mới: 1256 dòng mã test hermetic và 14 tình huống)
internal/ui/automation_binding_test.go                       (tạo mới: 165 dòng mã test binding App/UI)
cmd/switcher/main_test.go                                    (thêm test tranh chấp hai chiều CLI/auto)
reports/qa/automation/2026-10-20-p2a-qa-scenario-matrix-results.json (kết quả kiểm thử ma trận dạng JSON)
reports/qa/automation/2026-10-20-p2a-qa-scenario-matrix-results.md   (báo cáo QA kịch bản ma trận)
reports/coder/2026-10-20-p2a-e2e-and-14-edge-cases-report.md (báo cáo thi công chính thức của Coder)
```

---

## 6. Kết Luận & Bàn Giao

- Slice P2-A đã hoàn thành 100% mục tiêu: Toàn bộ 14 tình huống biên trong ma trận R1 cùng các binding thực tế đã được xây dựng, kiểm thử và chứng minh bằng evidence thực tế.
- Sẵn sàng bàn giao cho Supervisor kiểm định nghiệm thu.
- **Verdict:** READY_FOR_SUPERVISOR.
