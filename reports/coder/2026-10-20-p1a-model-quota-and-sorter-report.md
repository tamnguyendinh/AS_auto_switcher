# Coder Implementation Report: Triển Khai Slice P1-A (Model Quota Resolver & Burn-down Sorter)

- **Date:** 2026-10-20
- **Role:** Coder
- **Authority / SSOT:**
  - `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-plan.md` (Mục Slice P1-A)
  - `reports/system-architect/rp_system-architect_261020_141500_by_gemini-2-5-pro_autonomous_account_switching_architecture_specification.md` (Mục 3.1, 3.2, 6.1, 6.2)
  - `AGENTS.md` & `working-rules`
- **Scope (Touch-Map Khóa Cứng theo Plan):**
  - `internal/automation/model_quota.go` (tạo mới: types QuotaCategory, QuotaState, ModelQuotaEvaluation, ResolveActiveModelCategory, ClassifyQuotaState, EvaluateAccountQuota)
  - `internal/automation/sorter.go` (tạo mới: struct CandidateAccount, FilterEligibleCandidates với 5 tiêu chí, SortCandidatesBurnDown với thuật toán 4 tầng ưu tiên)
  - `internal/connector/automation_rpc.go` (tạo mới: struct CascadeModelConfigData, ClientModelConfig, GetCascadeModelConfig, ResolveActiveModel)
  - `internal/telemetry/client.go` (chỉ thêm metadata disabled pass-through vào ModelQuotaBucket, rawQuotaBucket, rawBucketInfo; giữ nguyên logic fetch/cache/loop)
  - Tests:
    - `internal/automation/model_quota_test.go`
    - `internal/automation/sorter_test.go`
    - `internal/connector/automation_rpc_test.go`
    - `internal/telemetry/quota_test.go`
  - Báo cáo thi công: `reports/coder/2026-10-20-p1a-model-quota-and-sorter-report.md`

---

## 1. Mục Tiêu & Tuân Thủ Kỷ Luật Sắt (Iron Rules Compliance)

1. **Tuân thủ đúng ranh giới Slice P1-A:**
   - Hoàn thành đầy đủ các module core cho Model Quota Resolver và Burn-down Sorter.
   - Tuyệt đối không can thiệp vào `internal/automation/controller.go`, `retry.go`, `internal/ui/*` (các file thuộc P1-B và P1-C).
   - Tuyệt đối bảo tồn nguyên vẹn `internal/manager/account_manager.go`, `internal/reloader/*`, các build scripts.
2. **Không dùng anvien:**
   - Toàn bộ quá trình khảo sát, đối chiếu và thi công đều sử dụng công cụ chuẩn của hệ thống, tuyệt đối cấm và không gọi công cụ anvien.
3. **Cấm spawn subagents:**
   - Thực thi cục bộ theo lệnh ủy thác trực tiếp từ CEO Session.

---

## 2. Invariant Family Map & Chi Tiết Thi Công

### Invariant Family: `Model Quota Resolution & Candidate Burn-down Ordering`

| Thành Phần / Bề Mặt | Target File | Chi Tiết Kỹ Thuật |
|---|---|---|
| **Active Model Category & Tam Phân Thái Quota** | `internal/automation/model_quota.go` | - Khai báo `QuotaCategory` (`CategoryGemini`, `CategoryClaudeGPT`).<br>- Khai báo Tam Phân Thái `QuotaState`: `ApplicableValid` (đầy đủ 5h & tuần), `NotApplicable` (chỉ có tuần hoặc 5h disabled: true), `UnknownUncertain` (mất mạng/timeout/khuyết tuần).<br>- Khai báo các ngưỡng: `Threshold5hLow = 0.02` (2%), `ThresholdWeeklyDepleted = 0.0` (0%).<br>- Hàm `ResolveActiveModelCategory(modelID, modelLabel)` phân loại Gemini vs Claude/GPT linh hoạt theo ID và Label.<br>- Hàm `ClassifyQuotaState` đánh giá tài khoản: với active account kiểm tra điều kiện chuyển chủ động (`5h <= 2%` hoặc `tuần = 0%`); với candidate kiểm tra tiêu chí 4 (`5h > 2%` và `tuần > 0%` hoặc chỉ `tuần > 0%`).<br>- Đảm bảo bất biến cách ly: Claude cạn kiệt tuyệt đối không kích hoạt chuyển tài khoản nếu phiên đang chạy Gemini. |
| **Bộ Lọc Sơ Loại 5 Tiêu Chí & Burn-down Sorter 4 Tầng** | `internal/automation/sorter.go` | - Khai báo `struct CandidateAccount` và `struct CandidateExclusion`.<br>- Hàm `FilterEligibleCandidates`: áp dụng 5 tiêu chí loại trừ: (1) Khác active account; (2) Token hợp lệ; (3) Chưa bị đánh dấu thất bại trong đợt; (4) Hạn mức nhóm model active thỏa mãn; (5) Dữ liệu quota tin cậy.<br>- Hàm `SortCandidatesBurnDown`: dùng `sort.SliceStable` với bộ so sánh 4 tầng: Tầng 1: Tuần thấp nhất trước -> Tầng 2: Ưu tiên có 5h -> Tầng 3: 5h cao nhất trước -> Tầng 4: Thứ tự danh bạ cố định. |
| **Model Config RPC Adapter** | `internal/connector/automation_rpc.go` | - Structs `CascadeModelConfigData`, `ClientModelConfig`, `ModelOrAlias`.<br>- Phương thức `GetCascadeModelConfig`: gọi Connect-RPC `EndpointCascadeModelConfigData` (`/exa.language_server_pb.LanguageServerService/GetCascadeModelConfigData`).<br>- Phương thức `ResolveActiveModel`: gọi `DiscoverActiveAgentModel` rồi join với cấu hình model động từ config API để lấy tên hiển thị và trạng thái disabled; fallback an toàn khi server lỗi. |
| **Disabled Metadata Pass-Through** | `internal/telemetry/client.go` | - Bổ sung trường `Disabled`, `FiveHourDisabled`, `WeeklyDisabled` vào `ModelQuotaBucket`, `rawQuotaBucket`, `rawBucketInfo`.<br>- Bổ sung giải mã trong `ModelQuotaBucket.UnmarshalJSON`.<br>- Truyền cờ disabled trong `ParseQuotaSummaryWithBuckets` và `toModelQuotaBucket` qua `RawBuckets` và các bucket tổng hợp `gemini`, `claude`, `gpt` mà không làm thay đổi các phân số fraction hiển thị. |

---

## 3. Kết Quả Kiểm Thử Toàn Diện (Verification Evidence)

### 3.1 Bảng Kiểm Thử Thực Tế Table A-F (Kiến Trúc Mục 6.2)
Đã chạy kiểm thử với ma trận tài khoản chuẩn từ tài liệu kiến trúc:

| Tài khoản | Quota Tuần | Quota 5h | Phân loại / Tiêu chí | Kết Quả Sorter Thực Tế | Trạng Thái |
|---|---:|---:|:---:|:---:|:---:|
| **A** | **3%** (0.03) | **80%** (0.80) | ApplicableValid (Eligible) | **Hạng 1** | PASS |
| **B** | **3%** (0.03) | **40%** (0.40) | ApplicableValid (Eligible) | **Hạng 2** | PASS |
| **C** | **3%** (0.03) | *Disabled / N/A* | NotApplicable (Eligible) | **Hạng 3** | PASS |
| **D** | **20%** (0.20) | **90%** (0.90) | ApplicableValid (Eligible) | **Hạng 4** | PASS |
| **E** | **1%** (0.01) | **1%** (0.01) | Bị loại ở Tiêu chí 4 ($5\text{h} \le 2\%$) | **Loại trừ** | PASS |
| **F** | **0%** (0.00) | **100%** (1.00) | Bị loại ở Tiêu chí 4 ($\text{Tuần} = 0\%$) | **Loại trừ** | PASS |

### 3.2 Kết Quả Chạy Tests Tự Động
- `internal/automation`:
  - `TestResolveActiveModelCategory`: 10/10 subtests PASS.
  - `TestClassifyQuotaState_TernaryAndSwitchConditions`: 10/10 subtests PASS (bao gồm các nhánh 5h <= 2%, tuần = 0%, 5h disabled, missing 5h, status error, nil data).
  - `TestActiveModelBinding_ModelIsolation`: PASS (chứng minh Claude cạn kiệt không làm switch Gemini, chuyển model sang Claude lập tức kích hoạt switch).
  - `TestFilterEligibleCandidates_5Criteria`: PASS (bao phủ 5 tiêu chí loại trừ).
  - `TestSortCandidatesBurnDown_TableAF_ArchitectureSpecification`: PASS (xếp hạng chính xác thứ tự A -> B -> C -> D).
  - `TestSortCandidatesBurnDown_Tier4TieBreak`: PASS (bảo tồn thứ tự store index khi các chỉ số quota bằng nhau).
- `internal/connector`:
  - `TestGetCascadeModelConfig_And_ResolveActiveModel`: PASS (kiểm tra đầy đủ parsing config, override priority, dynamic model ID join, label mapping, disabled flag và graceful fallback).
  - Toàn bộ các test cũ (`TestExtractCSRFToken`, `TestProbeHTTPPort`, `TestGetUserStatus`, `TestLiveLanguageServerDiscoveryAndStatus`, `TestGoogleOAuthRefresher_*`): PASS 100%.
- `internal/telemetry`:
  - `TestDisabledMetadataPassThrough_ParserCacheRoundTrip`: PASS (kiểm tra cờ disabled đi từ Connect-RPC JSON -> parser -> ModelQuotaBucket -> QuotaCache disk persistence -> reloaded cache).
  - Toàn bộ test suite telemetry hiện hữu: PASS 100%.
- Kiểm tra Race Detector (`-race`):
  - `go test -race ./internal/connector/... ./internal/telemetry/... ./internal/automation/...`: PASS 100% không phát sinh data race.
- CLI Compilation:
  - `go build ./cmd/switcher`: PASS (exit code 0).

---

## 4. Danh Mục Files Đã Thay Đổi & Tạo Mới

```text
internal/automation/model_quota.go          (new: 172 lines)
internal/automation/model_quota_test.go     (new: 250 lines)
internal/automation/sorter.go               (new: 135 lines)
internal/automation/sorter_test.go          (new: 155 lines)
internal/connector/automation_rpc.go        (new: 98 lines)
internal/connector/automation_rpc_test.go   (new: 154 lines)
internal/telemetry/client.go                (modified: metadata disabled pass-through only)
internal/telemetry/quota_test.go            (modified: added TestDisabledMetadataPassThrough_ParserCacheRoundTrip)
reports/coder/2026-10-20-p1a-model-quota-and-sorter-report.md
```

## 5. Kết Luận & Bàn Giao
Slice P1-A đã được thi công hoàn chỉnh, chuẩn xác 100% theo kế hoạch và đặc tả kiến trúc. Không còn bất kỳ bề mặt nào chưa được kiểm chứng trong phạm vi P1-A. Sẵn sàng bàn giao cho Supervisor Lane nghiệm thu.
