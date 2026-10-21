# Báo Cáo Thi Công: Phase 4 - Tầng Tự Động Chuyển Đổi & Telemetry Engine (Automation Sorter, Controller & Telemetry Silence)
(Account Status On/Off Automation Controller & Telemetry Silence Report - Phase 4)

- **Ngày thực hiện:** 2026-10-21
- **Vai trò:** Coder
- **Thẩm quyền kiến trúc (Authority / SSOT):** `reports/architect/2026-10-21-account-status-on-off-and-auto-stop-spec.md` (`SPEC-ARCH-20261021-ACCOUNT-STATUS-ON-OFF-AUTO-STOP`, commit `966f955`)
- **Trạng thái:** HOÀN THÀNH (READY_FOR_SUPERVISOR)

---

## I. TỔNG QUAN THI CÔNG & INVARIANT FAMILY MAP

### 1. Invariant Family Map
- **Family Name:** `Account Status Automation & Telemetry Silence (Phase 4)`
- **Authority / SSOT:** `reports/architect/2026-10-21-account-status-on-off-and-auto-stop-spec.md` (Mã `SPEC-ARCH-20261021-ACCOUNT-STATUS-ON-OFF-AUTO-STOP`, commit `966f955`)
- **Core Invariants Enforced:**
  1. `Invariant-Disabled-Switch-Lockout`:
     - Tầng Bộ điều khiển (Controller): Bị bộ lọc ứng viên `FilterEligibleCandidates` loại bỏ ngay từ vòng duyệt đầu tiên thông qua Criterion 6 (`Status == "off"`).
     - Tài khoản mang trạng thái `Status == "off"` được thêm vào danh sách loại trừ `CandidateExclusion` với `Criterion: 6` và lý do chuẩn hóa `"tài khoản đang ở chế độ vô hiệu hóa (OFF)"`.
  2. `Invariant-No-Candidate-Auto-Halt`:
     - Khi `len(eligible) == 0` (hết ứng viên hợp lệ do tất cả tài khoản dự phòng đều mang trạng thái `off`, hết quota, hoặc không hợp lệ):
       - Lập tức kích hoạt cơ chế Dừng an toàn (Safety Stop): `c.enabled.Store(false)`.
       - Đồng bộ trạng thái retry coordinator: `c.syncMonitoring()`.
       - Phát sự kiện trạng thái chuẩn hóa: `c.EmitStatus(StatusNoCandidate, "Không có tài khoản để chuyển.", exclusions)`.
       - Trả về lỗi `fmt.Errorf("%w: %+v", ErrNoCandidates, exclusions)`.
     - Tương tự khi tất cả các ứng viên đều thất bại trong bước preflight:
       - `c.enabled.Store(false)`.
       - `c.syncMonitoring()`.
       - `c.EmitStatus(StatusNoCandidate, "Không có tài khoản để chuyển.", nil)`.
       - Trả về lỗi `ErrNoCandidates`.
  3. `Invariant-Disabled-Telemetry-Silence`:
     - Không thực hiện quét quota định kỳ (Telemetry loop) hoặc gửi request mạng tới Google API cho bất kỳ tài khoản nào đang ở trạng thái `status == "off"`.
     - Trong vòng lặp polling toàn bộ tài khoản `PollAll(ctx)`: Lọc bỏ các tài khoản có `status == "off"`, không khởi chạy worker goroutine và tuyệt đối không gửi request HTTP tới Google Cloud Code API hay Connect-RPC.
     - Trong hàm `PollAccount(ctx, email)`: Ngăn chặn gửi request và trả về lỗi nếu tài khoản mang trạng thái `status == "off"`.
  4. `Invariant-Backward-Compatibility-Default-On`:
     - Trường `Status` trên `CandidateAccount` và `AccountTokenInfo` mặc định nhận giá trị `"on"` nếu rỗng, đảm bảo tương thích ngược 100% với toàn bộ dữ liệu hiện hữu và các mock/fixture kiểm thử cũ.

- **Sibling Surfaces Inspected & Closed:**
  1. `internal/automation/sorter.go`:
     - Mở rộng struct `CandidateAccount`: Thêm trường `Status string json:"status"`.
     - Cập nhật struct `CandidateExclusion`: Cập nhật chú thích `Criterion int json:"criterion"` (`// 1 to 6`).
     - Bổ sung Criterion 6 trong `FilterEligibleCandidates`: Bỏ qua các tài khoản có `strings.EqualFold(cand.Status, "off")`.
     - Cập nhật `BuildCandidateAccount`: Khởi tạo mặc định `Status: "on"`.
  2. `internal/automation/controller.go`:
     - Trong `triggerSwitch`: Gán `cand.Status = c.mgr.GetAccountStatus(email)`.
     - Khi `len(eligible) == 0`: Gọi `c.enabled.Store(false)`, `c.syncMonitoring()`, `c.EmitStatus(StatusNoCandidate, "Không có tài khoản để chuyển.", exclusions)`.
     - Khi tất cả ứng viên thất bại preflight: Gọi `c.enabled.Store(false)`, `c.syncMonitoring()`, `c.EmitStatus(StatusNoCandidate, "Không có tài khoản để chuyển.", nil)`.
  3. `internal/telemetry/pool.go`:
     - Mở rộng `AccountTokenInfo`: Thêm trường `Status string`.
     - Cập nhật `StoreAccountSource.ListAccounts()`: Đọc `s.store.GetAccountStatus(clean)` và gán vào `AccountTokenInfo.Status` (mặc định `"on"`).
     - Bổ sung phương thức `GetAccountStatus(email string) (string, error)` cho `StoreAccountSource`.
     - Cập nhật `PollAll(ctx)`: Lọc bỏ tài khoản có `strings.EqualFold(st, "off")`, bảo đảm hoàn toàn không phát sinh request mạng tới Google API.
     - Cập nhật `PollAccount(ctx, email)`: Kiểm tra trạng thái tài khoản trước khi truy vấn, từ chối thực thi nếu mang trạng thái `"off"`.
  4. `internal/automation/sorter_test.go`:
     - Bổ sung ca kiểm thử `TestFilterEligibleCandidates_ExcludesOffAccount`: Kiểm tra Criterion 6 và lý do chuẩn hóa.
  5. `internal/automation/controller_test.go`:
     - Bổ sung kiểm tra `ctrl.IsEnabled() == false` vào `TestAutomationController_NoCandidates`.
     - Bổ sung ca kiểm thử `TestController_SafetyStop_WhenAllCandidatesOff`: Giả lập tất cả tài khoản phụ đều mang `Status = "off"`, kiểm chứng `enabled = false` và sự kiện `StatusNoCandidate` với thông báo `"Không có tài khoản để chuyển."`.
     - Bổ sung ca kiểm thử `TestController_SafetyStop_WhenAllCandidatesFailPreflight`: Giả lập tất cả ứng viên thất bại preflight, kiểm chứng Safety Stop tự động ngắt `enabled = false` và phát thông báo chuẩn.
  6. `internal/automation/runtime_regression_test.go`:
     - Cập nhật `TestCandidatePreflightFailureExpiresWithOperation` theo đúng bất biến Safety Stop mới: Sau khi preflight thất bại khiến controller tự động dừng, kích hoạt lại `SetEnabled(true)` trước khi thực thi vòng kiểm thử tiếp theo.
  7. `internal/telemetry/telemetry_test.go`:
     - Bổ sung ca kiểm thử `TestTelemetryPool_PollAll_SkipsOffAccounts`: Kiểm chứng `PollAll` bỏ qua tài khoản `off`, chỉ gửi đúng số lượng request cho tài khoản `on` và không đưa tài khoản `off` vào cache.
     - Bổ sung ca kiểm thử `TestTelemetryPool_WithStore_SkipsOffAccounts`: Kiểm chứng tích hợp với `AccountStore`, đảm bảo `PollAll` và `PollAccount` tuân thủ tuyệt đối `Invariant-Disabled-Telemetry-Silence`.
- **Legacy Fallback Status:** Tương thích 100%, không phá vỡ bất kỳ hành vi hay cấu trúc nào hiện có.
- **Residual Unverified Surfaces:** `none`.

---

## II. CHI TIẾT CÁC TỆP SỬA ĐỔI (FILES CHANGED)

| STT | Tệp tin | Trách nhiệm thực thi |
|:---:|---|---|
| 1 | `internal/automation/sorter.go` | Mở rộng `CandidateAccount.Status`, bổ sung Criterion 6 loại trừ tài khoản "off", mặc định "on" trong `BuildCandidateAccount`. |
| 2 | `internal/automation/controller.go` | Gán `cand.Status` từ manager, kích hoạt Dừng an toàn (`enabled = false`, `syncMonitoring()`, thông báo `"Không có tài khoản để chuyển."`) khi hết ứng viên hoặc tất cả preflight thất bại. |
| 3 | `internal/telemetry/pool.go` | Mở rộng `AccountTokenInfo.Status`, nạp status trong `StoreAccountSource`, bỏ qua polling Google API cho tài khoản "off" trong `PollAll` và `PollAccount`. |
| 4 | `internal/automation/sorter_test.go` | Thêm unit test `TestFilterEligibleCandidates_ExcludesOffAccount`. |
| 5 | `internal/automation/controller_test.go` | Thêm unit test `TestController_SafetyStop_WhenAllCandidatesOff` và `TestController_SafetyStop_WhenAllCandidatesFailPreflight`. |
| 6 | `internal/automation/runtime_regression_test.go` | Căn chỉnh `TestCandidatePreflightFailureExpiresWithOperation` phù hợp với hợp đồng Safety Stop mới. |
| 7 | `internal/telemetry/telemetry_test.go` | Thêm unit test `TestTelemetryPool_PollAll_SkipsOffAccounts` và `TestTelemetryPool_WithStore_SkipsOffAccounts`. |

---

## III. KẾT QUẢ KIỂM CHỨNG & BẰNG CHỨNG THỰC NGHIỆM (E2E VERIFICATION)

### 1. Kiểm thử Unit Test Automation (`internal/automation/...`)
- **Lệnh thực thi:** `go test -v ./internal/automation/... -run "TestController_SafetyStop|TestFilterEligibleCandidates_ExcludesOffAccount"`
  ```text
  === RUN   TestController_SafetyStop_WhenAllCandidatesOff
  --- PASS: TestController_SafetyStop_WhenAllCandidatesOff (0.54s)
  === RUN   TestController_SafetyStop_WhenAllCandidatesFailPreflight
  --- PASS: TestController_SafetyStop_WhenAllCandidatesFailPreflight (0.88s)
  === RUN   TestFilterEligibleCandidates_ExcludesOffAccount
  --- PASS: TestFilterEligibleCandidates_ExcludesOffAccount (0.00s)
  PASS
  ok  	antigravity-switch/internal/automation	3.297s
  ```
- **Kiểm thử toàn bộ gói automation:**
  - **Lệnh thực thi:** `go test ./internal/automation/...`
  - **Kết quả:** `ok antigravity-switch/internal/automation 12.186s` (PASS 100%).

### 2. Kiểm thử Unit Test Telemetry (`internal/telemetry/...`)
- **Lệnh thực thi:** `go test -v ./internal/telemetry/... -run "TestTelemetryPool_PollAll_SkipsOffAccounts|TestTelemetryPool_WithStore_SkipsOffAccounts"`
  ```text
  === RUN   TestTelemetryPool_PollAll_SkipsOffAccounts
  --- PASS: TestTelemetryPool_PollAll_SkipsOffAccounts (0.50s)
  === RUN   TestTelemetryPool_WithStore_SkipsOffAccounts
  --- PASS: TestTelemetryPool_WithStore_SkipsOffAccounts (0.45s)
  PASS
  ok  	antigravity-switch/internal/telemetry	2.470s
  ```
- **Kiểm thử toàn bộ gói telemetry:**
  - **Lệnh thực thi:** `go test ./internal/telemetry/...`
  - **Kết quả:** `ok antigravity-switch/internal/telemetry 8.476s` (PASS 100%).

### 3. Kiểm thử toàn bộ kho mã nguồn (`go test ./...`)
- **Lệnh thực thi:** `go test ./...`
- **Kết quả:**
  ```text
  ok  	antigravity-switch/cmd/switcher	1.972s
  ok  	antigravity-switch/internal/automation	(cached)
  ok  	antigravity-switch/internal/connector	(cached)
  ok  	antigravity-switch/internal/manager	(cached)
  ok  	antigravity-switch/internal/model	(cached)
  ok  	antigravity-switch/internal/reloader	(cached)
  ok  	antigravity-switch/internal/scanner	(cached)
  ok  	antigravity-switch/internal/store	(cached)
  ok  	antigravity-switch/internal/telemetry	(cached)
  ok  	antigravity-switch/internal/ui	34.930s
  ok  	antigravity-switch/internal/wincred	(cached)
  ```
  Toàn bộ test suites của mọi package đều PASS 100%.

### 4. Kiểm chứng biên dịch Windows bằng đúng script của repo (`scripts/build_windows.ps1`)
- **Lệnh thực thi:** `pwsh -File .\scripts\build_windows.ps1 -Clean`
- **Kết quả:**
  ```text
  ================================================================================
  ⚡ Antigravity Multi-Account Switcher - Windows Build System
     OutDir: bin | SkipTests: False | Clean: True
  ================================================================================
  [1/4] Cleaning output artifacts...
    Removed directory: bin
    Clean complete.
  [2/4] Running pre-flight tests & vet...
  ok  	antigravity-switch/cmd/switcher	1.508s
  ok  	antigravity-switch/internal/automation	(cached)
  ok  	antigravity-switch/internal/connector	(cached)
  ok  	antigravity-switch/internal/manager	(cached)
  ok  	antigravity-switch/internal/model	(cached)
  ok  	antigravity-switch/internal/reloader	(cached)
  ok  	antigravity-switch/internal/scanner	(cached)
  ok  	antigravity-switch/internal/store	4.891s
  ok  	antigravity-switch/internal/telemetry	(cached)
  ok  	antigravity-switch/internal/ui	(cached)
  ok  	antigravity-switch/internal/wincred	(cached)
    [PASS] Verification succeeded.
  [3/4] Compiling target binaries...
    -> Checking Windows application icon and resource...
    -> Compiling resource with windres: cmd\switcher\app.rc -> cmd\switcher\rsrc_windows_amd64.syso
    [SUCCESS] Compiled PE resource object: cmd\switcher\rsrc_windows_amd64.syso (12552 bytes)
    -> Building Standalone GUI Binary (Fyne, CGO_ENABLED=1)...
    [SUCCESS] Created: bin\antigravity-switch.exe
    Compile Duration: 5555 ms | Size: 27.78 MB (29128192 bytes)
  [4/4] Executing quick verification...
    [PASS] Embedded PE application icon verified on bin\antigravity-switch.exe (32x32)
    [INFO] GUI binary built at bin\antigravity-switch.exe.
  ================================================================================
  ⚡ BUILD SUCCESSFUL
  ================================================================================
  ```
  - **Exit Code:** `0`
  - **Script Diff:** `scripts/build_windows.ps1` giữ nguyên 100% (0 lines diff).

---

## IV. ĐỐI CHIẾU TIÊU CHÍ BÀN GIAO (DEFINITION OF DONE)

| Tiêu chuẩn nghiệm thu | Trạng thái | Minh chứng kỹ thuật |
|---|:---:|---|
| Mở rộng `CandidateAccount` có `Status string json:"status"` | **ĐẠT** | Đã khai báo tại `internal/automation/sorter.go:15`. |
| `FilterEligibleCandidates` loại trừ tài khoản `status == "off"` | **ĐẠT** | Criterion 6, reason `"tài khoản đang ở chế độ vô hiệu hóa (OFF)"`. Unit test `TestFilterEligibleCandidates_ExcludesOffAccount` PASS. |
| `triggerSwitch` nạp `cand.Status` từ store/manager | **ĐẠT** | Đã gán `cand.Status = c.mgr.GetAccountStatus(email)` tại `internal/automation/controller.go:514`. |
| Safety Stop ngắt `c.enabled.Store(false)`, đồng bộ monitoring và phát thông báo chuẩn khi `len(eligible) == 0` | **ĐẠT** | Thực thi tại `internal/automation/controller.go:521-525`. Unit test `TestController_SafetyStop_WhenAllCandidatesOff` PASS. |
| Safety Stop ngắt `c.enabled.Store(false)`, đồng bộ monitoring và phát thông báo chuẩn khi toàn bộ preflight thất bại | **ĐẠT** | Thực thi tại `internal/automation/controller.go:575-579`. Unit test `TestController_SafetyStop_WhenAllCandidatesFailPreflight` PASS. |
| `internal/telemetry/pool.go` bỏ qua polling Google API cho tài khoản "off" | **ĐẠT** | Thực thi tại `PollAll` và `PollAccount`. Unit tests `TestTelemetryPool_PollAll_SkipsOffAccounts` và `TestTelemetryPool_WithStore_SkipsOffAccounts` PASS. |
| Build script `scripts/build_windows.ps1` nguyên vẹn 100% | **ĐẠT** | 0 lines diff. |
| Toàn bộ `go test ./...` PASS 100% | **ĐẠT** | Toàn bộ các package passed. |
| Biên dịch `pwsh -File .\scripts\build_windows.ps1 -Clean` thành công Exit Code 0 | **ĐẠT** | Sinh file `bin\antigravity-switch.exe` thành công. |

Phase 4 đã hoàn tất đầy đủ mọi yêu cầu, sẵn sàng chuyển tiếp sang Supervisor lane nghiệm thu độc lập.
