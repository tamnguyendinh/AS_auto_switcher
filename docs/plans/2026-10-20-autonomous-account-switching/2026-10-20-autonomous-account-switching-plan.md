# Autonomous Account Switching Plan

## Metadata

- Date: `2026-10-20`
- Status: `draft`
- Plan: `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-plan.md`
- Evidence: `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-evidence.md`
- Benchmark: `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-benchmark.md`
- Actual status: `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-actual-status.md`

## Goal

Xây dựng hệ thống điều phối tự động hoàn toàn mới (**Autonomous Account Switching Controller**) trong Antigravity Multi-Account Switcher (Go Engine) nhằm tự động luân chuyển tài khoản khi tài khoản hiện tại chạm ngưỡng cạn kiệt Quota (5h <= 2% hoặc Tuần = 0%) hoặc gặp sự cố chặn hạn ngạch (429, RESOURCE_EXHAUSTED). Hệ thống bảo toàn 100% ngữ cảnh hội thoại trong vòng 2 giờ gần nhất mà không cần người dùng thao tác bấm nút thủ công, tuân thủ triệt để triết lý thiết kế KISS (Đơn giản là sức mạnh), chiến lược Burn-down Sorter (ưu tiên dùng cạn tài khoản tuần thấp trước, giữ tài nguyên mới/dự trữ sau), và tích hợp tính năng Native In-Memory 3-Retry thông qua Connect-RPC `SendUserCascadeMessage` với danh sách nội dung rỗng (`empty content list`) để phục hồi nhanh (< 500ms) trước khi kích hoạt chuyển đổi tài khoản.

## Rules

- Complete P0 actual status before implementation work.
- Update each checklist item immediately when it is completed.
- Record evidence as work completes.
- Record benchmarkable counts or measurements when they are taken.
- Update later phase status assumptions, next actions, and work steps when actual-status evidence changes the repo state.
- After completing a phase or implementation slice and refreshing `actual-status.md`, update the next affected phase's work steps as needed to match the latest repo reality, while preserving that phase's original goal, scope, acceptance criteria, and major phase order.
- Anvien only helps localize candidate boundaries; relationships and blast radius must be cross-checked against imports, call paths, and actual code before deciding scope.
- Run Anvien detect-changes before every implementation-slice commit when implementation work was performed.
- For public runtime or UI-facing changes, validate the real user-visible runtime with browser or Playwright evidence.
- For app/runtime validation, full build must include Docker image/container build. If Docker is missing or not run, full build is incomplete. (N/A for native desktop Go application: compile with `go build -v ./...` and execute GUI/headless test suite).
- Keep the standard planner structure. These detail rules only make phase checklist items concrete enough to implement safely.
- Every implementation phase must be decomposed into multiple implementation slices that are as small as practical. A phase is a grouping and ordering container; a slice is the executable implementation unit.
- Do not implement a phase directly. Work starts from a slice ID such as `P1-A`, `P1-B`, or `P1-C`.
- Prefer many narrow slices over one broad slice. A single-slice implementation phase is allowed only when the plan explicitly states why the phase cannot be split further without creating empty or non-executable slices.
- Each implementation slice must include:
  + Goal
  + Scope Boundary
  + Non-Goals when useful
  + Pre-flight Questions
  + Work Steps (must act as a strict "control plane" following the 6-step execution lifecycle: Code → Inspect → Update tests → Build → QA → Acceptance/Commit. Step 1 (Code) must be decomposed into leaf atomic tasks, where each leaf task defines: Action, Inputs, Allowed Edit Scope, Outputs, and Verification Condition).
  + Implementation Gate
  + Acceptance
  + Evidence Targets
  + Actual-status Update
  + Commit Boundary
- Every slice's Work Steps must strictly follow the 6-step execution lifecycle:
  1. Code: Execute leaf atomic tasks in core source files only. Each leaf atomic task must explicitly specify Action, Inputs, Allowed Edit Scope, Outputs, local Verification Condition, and a Task Checkpoint with explicit prefix (e.g. `wip(<slice_id>): task 1.x - <verified task outcome>`; micro WIP commit for granular task progress, NOT a slice completion commit). Do not touch test files during this step.
  2. Code Inspection & Lint: Review diff, syntax, linter, typecheck, and ensure edits remain strictly within assigned boundary.
  3. Update Tests: Following Master Rule 6 (Code first, test second), add or update unit/integration tests to verify the newly implemented code behavior.
  4. Build: Run the repository's full build to ensure type contracts, package bundles, and binaries compile cleanly.
  5. QA (Mini QA): Verify user-visible runtime behavior (desktop app GUI, API, CLI, or simulated test harness evidence).
  6. Slice Acceptance & Official Milestone Commit: Verify slice acceptance criteria, record evidence in `evidence.md`, refresh `actual-status.md`, and create the official slice completion commit (e.g. `feat(<slice_id>): <slice outcome>`).
- Hidden fallback is forbidden. Prefer a visible failure over a fallback that hides a broken primary path.
- When touching DB-backed content, verify the full loop when applicable: UI input -> submit action -> DB write -> DB read after reload/new request -> correct UI render or omission. If there is no UI, replace UI steps with the real caller/consumer flow.
- Tests must prove product behavior. Delete or replace tests that only assert implementation details, helper output, static DOM existence, or mocked plumbing without proving trigger -> process -> observable result.

## Problem

Trong phiên bản Antigravity v2.24.0, người dùng lập trình gặp phải sự cố gián đoạn luồng làm việc do cạn kiệt Quota:
1. **Gián đoạn công việc do cạn kiệt Quota:** Khi một tài khoản chạm ngưỡng hạn ngạch 5h (<= 2%) hoặc hạn ngạch tuần (0%), hoặc gặp lỗi `429 Too Many Requests` / `RESOURCE_EXHAUSTED`, lập trình viên phải dừng công việc, mở ứng dụng quản lý tài khoản, bấm chuyển tài khoản thủ công, rồi quay lại IDE tìm các đoạn hội thoại bị gián đoạn để tiếp tục.
2. **Yêu cầu kỹ thuật thực tế trên Antigravity v2.24.0:**
   - Antigravity v2.24.0 không hỗ trợ ReAuth nóng trong bộ nhớ RAM; quy trình chuyển đổi chuẩn bắt buộc phải ghi WinCred, đồng bộ ToS, `taskkill /F /IM language_server.exe` và **chờ đủ 8 giây** để `monitorLsCrashInternal` tái sinh tiến trình an toàn.
   - Quá trình giám sát cần tận dụng gói `internal/telemetry` có sẵn (gọi Google API mỗi 60 giây và theo dõi phần trăm quota tường minh) thay vì quét log văn bản thụ động.
   - Sắp xếp ứng viên tài khoản phải có cơ chế liên kết với Model đang chạy trong phiên làm việc hiện tại (Gemini vs Claude/GPT-OSS), tránh chuyển đổi tài khoản mù quáng hoặc không tối ưu.
   - Hệ thống cần tích hợp khay điều khiển và công tắc chuyển đổi trực tiếp trên Fyne GUI có sẵn.
3. **Cơ chế phục hồi siêu tốc chưa được tự động hóa:** Nút "Try again" trong IDE thực chất gửi Connect-RPC `SendUserCascadeMessage` với danh sách nội dung rỗng (`empty content list`) tới conversation ID lỗi, giúp Language Server thử lại mà không cần restart tiến trình (< 500ms). Cần một bộ điều phối tự động kích hoạt tính năng này tối đa 3 lần trước khi kích hoạt quy trình hoán đổi tài khoản.

## Scope

### In Scope
- Tạo mới package `internal/automation/`:
  - `internal/automation/model_quota.go`: Xác định Active Model từ phiên làm việc (`GetCascadeModelConfigData`), ánh xạ nhóm quota tương ứng (Gemini vs Claude/GPT), phân loại trạng thái Quota thành Tam Phân Thái (`APPLICABLE_VALID`, `NOT_APPLICABLE`, `UNKNOWN_UNCERTAIN`), xử lý cờ `disabled: true` trên bucket 5h.
  - `internal/automation/sorter.go`: Bộ lọc sơ loại ứng viên (5 tiêu chí) và thuật toán sắp xếp ưu tiên Burn-down Sorter (4 tầng ưu tiên: Tuần thấp nhất -> Có 5h -> 5h cao nhất -> Thứ tự cố định).
  - `internal/automation/retry.go`: Giám sát stream `StreamAgentStateUpdates`, bắt lỗi có cấu trúc (`429`, `RESOURCE_EXHAUSTED`, `RATE_LIMIT_EXCEEDED`, `QUOTA_EXHAUSTED`), thực hiện Native In-Memory 3-Retry qua Connect-RPC `SendUserCascadeMessage` với empty content list.
  - `internal/automation/controller.go`: Bộ điều phối kích hoạt kép (Dual-Trigger Engine: Van chủ động 60s telemetry + Van bị động lỗi 429), cổng kiểm soát đơn luồng (Single-Flight Concurrency Gate) tránh xung đột lệnh chuyển, công tắc nguyên tử `atomic.Bool` quản lý trạng thái Bật/Tắt.
- Cập nhật giao diện người dùng Fyne GUI tại `internal/ui/`:
  - `internal/ui/automation_view.go`: Thêm Toggle Switch Bật/Tắt chế độ tự động chuyển đổi và nhãn hiển thị trạng thái Automation động.
- Kiểm thử tự động hóa toàn diện:
  - `internal/automation/model_quota_test.go`, `internal/automation/sorter_test.go`, `internal/automation/retry_test.go`, `internal/automation/controller_test.go`.
  - `internal/automation/automation_e2e_test.go`: Test harness mô phỏng 14 tình huống biên và an toàn.

### Out of Scope / Preserve Only
- `internal/manager/account_manager.go`: Giữ nguyên và tái sử dụng hàm `SwitchAccount()`. Bộ điều phối tự động gọi trực tiếp hàm này, không sửa đổi core chuyển đổi hotswap đã hoàn thiện.
- `internal/reloader/`: Giữ nguyên cơ chế quét SQLite và phục hồi active threads trong 2 giờ (`continuation.go`). Controller không can thiệp hay gọi tiếp tục lần thứ hai vì `SwitchAccount` đã bao hàm việc này.
- `internal/telemetry/`: Giữ nguyên client polling, cache và loop 60s. Controller chỉ đọc dữ liệu quota đã được cache và cập nhật.

## Non-Goals

- Không triển khai daemon systray độc lập ngoài Fyne GUI.
- Không can thiệp vào mã nguồn nhị phân của Antigravity IDE hay Language Server.
- Không triển khai hoán đổi nóng RPC ReAuth bỏ qua restart tiến trình (bảo lưu bắt buộc: taskkill + 8s wait).
- Không chuyển đổi tài khoản khi model đang chạy khác nhóm model bị cạn kiệt (ví dụ: Claude hết tuần nhưng phiên đang chạy Gemini thì tuyệt đối không chuyển).
- Không đoán mò quota từ ký hiệu `--%` hay trạng thái mạng lỗi.

## Requirements

1. **Active Model Binding (R1):** Mọi quyết định kiểm tra ngưỡng quota và xếp hạng ứng viên phải dựa trên model của phiên làm việc hiện tại (`Gemini` -> Nhóm Gemini 5h & Tuần; `Claude/GPT-OSS` -> Nhóm Claude 5h & Tuần).
2. **Tam Phân Thái Quota (R2):** Dữ liệu quota từ Google API được chuẩn hóa thành 3 trạng thái:
   - `APPLICABLE_VALID`: Có bucket 5h đang mở, fraction hợp lệ ($0.0 \le \text{fraction} \le 1.0$), kiểm tra cả $5\text{h} \le 2\%$ và $\text{Tuần} = 0\%$.
   - `NOT_APPLICABLE`: Chỉ có bucket tuần hoặc bucket 5h có cờ `disabled: true`, bỏ qua điều kiện 5h, chỉ theo dõi $\text{Tuần} = 0\%$.
   - `UNKNOWN_UNCERTAIN`: Dữ liệu khuyết, timeout, mất mạng. Tuyệt đối không chuyển, yêu cầu làm mới.
3. **Dual-Trigger Architecture & Single-Flight Gate (R3):**
   - Van chủ động (Proactive): Telemetry quét định kỳ 60s, kích hoạt chuyển khi $5\text{h} \le 2\%$ hoặc $\text{Tuần} = 0\%$.
   - Van bị động (Reactive): Bắt lỗi thực thi `429`/`RESOURCE_EXHAUSTED`, kích hoạt 3 lần Native Retry. Thất bại cả 3 lần -> Kích hoạt chuyển khẩn cấp.
   - Single-Flight Gate: Hợp nhất các tín hiệu xảy ra đồng thời thành 1 đợt chuyển duy nhất.
4. **Native In-Memory 3-Retry (R4):** Khi gặp lỗi giới hạn, gửi Connect-RPC `SendUserCascadeMessage` với `empty content list` tới conversation ID bị lỗi, lặp lại tối đa 3 lần (cách nhau 2-3s). Nếu thành công dừng retry; nếu cả 3 lần thất bại mới chuyển tài khoản.
5. **Burn-down Sorter (R5):** Ứng viên phải vượt qua 5 tiêu chí lọc, sau đó sắp xếp theo: (1) Quota Tuần thấp nhất trước -> (2) Ưu tiên tài khoản có áp dụng 5h -> (3) Quota 5h cao nhất trước -> (4) Giữ nguyên thứ tự danh bạ cố định.
6. **Xử lý 14 Tình Huống Biên & An Toàn (R6):** Tuân thủ tuyệt đối ma trận 14 tình huống biên quy định trong Báo cáo Kiến trúc (bao gồm: không chuyển khi khác nhóm model, cờ disabled 5h, single-flight deduplication, mutex bảo vệ khi người dùng bấm tay trên GUI, xử lý khi kho hết tài khoản hợp lệ).
7. **GUI Controls (R7):** Giao diện Fyne có Toggle công tắc Bật/Tắt rõ ràng và nhãn hiển thị trạng thái động của Controller.

## Acceptance Criteria

- [ ] Cấu trúc package `internal/automation/` được thiết lập hoàn chỉnh với 4 module cốt lõi: `model_quota.go`, `sorter.go`, `retry.go`, `controller.go`.
- [ ] 100% các bài kiểm thử đơn vị (`*_test.go`) cho từng module được triển khai độc lập, kiểm tra đầy đủ các nhánh logic, tam phân thái, cờ `disabled: true`, và 4 tầng ưu tiên Burn-down Sorter.
- [ ] Bộ điều phối `controller.go` bảo đảm tính đơn luồng (Single-Flight Gate), không bao giờ kích hoạt 2 tiến trình `SwitchAccount` đồng thời khi van chủ động và bị động cùng phát tín hiệu.
- [ ] Giao diện Fyne GUI hiển thị Toggle Automation và trạng thái hoạt động chính xác theo thời gian thực.
- [ ] Ma trận 14 tình huống biên được bao phủ đầy đủ bằng test kịch bản tự động (Integration Test Harness).
- [ ] Lệnh biên dịch toàn bộ repository `go build -v ./...` hoàn thành thành công 100% không có lỗi cảnh báo.

## Checklist

- [ ] P0-A: Baseline & Actual Status Discovery
  - Goal: Rà soát hiện trạng thực tế, xác lập baseline trước khi thực thi.
  - Work Steps: inspect source-of-truth files (`internal/manager/account_manager.go`, `internal/telemetry/`, `internal/reloader/`, `internal/ui/`), classify each surface, record missing pieces (`internal/automation/`), and update later phase status assumptions, next actions, and work steps from evidence.
  - Implementation Gate: no implementation or editing starts until `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-actual-status.md` has a final P0 decision.
  - Acceptance: actual status identifies correct, partial, missing/unbound, fake-or-stub, and blocked surfaces for this scope.

### P1: Autonomous Failover Engine Implementation

- Phase Goal: Xây dựng và triển khai toàn bộ các thành phần lõi của động cơ tự động chuyển đổi tài khoản (Model Resolver, Burn-down Sorter, Native Retry, Dual-Trigger Controller, và Fyne GUI Toggle).
- Phase Boundary:
  - In scope: `internal/automation/model_quota.go`, `internal/automation/sorter.go`, `internal/automation/retry.go`, `internal/automation/controller.go`, `internal/ui/automation_view.go`, và các unit tests đi kèm.
  - Out of scope: Core `SwitchAccount` trong `internal/manager/`, `internal/telemetry/`, và `internal/reloader/`.
  - Dependencies: Types từ `internal/telemetry` và cấu trúc tài khoản từ `internal/manager`.
- Phase Implementation Rule: Do not implement P1 directly. Implement P1-A, verify it, record evidence, refresh actual-status, commit; then continue to P1-B, then P1-C.
- Ordered Slice List:
  - P1-A: Model Quota Resolver & Burn-down Sorter
  - P1-B: Dual-Trigger Engine & 3-Retry Controller
  - P1-C: Automation Coordinator & GUI Toggle

- [ ] P1-A: Model Quota Resolver & Burn-down Sorter
  - Goal: Hiện thực hóa logic nhận diện Active Model, ánh xạ nhóm Quota, Tam Phân Thái hạn ngạch, bộ lọc sơ loại 5 tiêu chí, và thuật toán sắp xếp ứng viên 4 tầng ưu tiên Burn-down Sorter.
  - Scope Boundary:
    - Editable: `internal/automation/model_quota.go`, `internal/automation/sorter.go`
    - Inspect-only: `internal/telemetry/quota_models.go`, `internal/telemetry/cache.go`, `internal/manager/account_manager.go`
    - Preserve-only: `internal/telemetry/*`, `internal/manager/*`
    - Out of scope: `internal/automation/controller.go`, `internal/ui/*`
  - Non-Goals: Không thực hiện gọi HTTP ra ngoài trong module này; module chỉ phân giải trên dữ liệu Quota Snapshot đã có và model config data.
  - Pre-flight Questions:
    - Data source: Model info từ Connect-RPC/Language Server và Quota Snapshot từ `telemetry.Cache`.
    - Display permission: N/A.
    - DB read flow: Đọc trạng thái tài khoản từ file/store.
    - DB write flow: N/A.
    - Render location: N/A.
    - UI behavior flow: N/A.
    - Docker runtime: N/A (Desktop Go).
    - Playwright target: N/A.
    - Behavior test: `internal/automation/model_quota_test.go`, `internal/automation/sorter_test.go`.
    - Cleanup/quarantine: N/A.
    - External side effects: None. Pure functional transformation and classification.
    - N/A notes: None.
  - Work Steps:
    1. Code (Execute leaf atomic tasks in core source files only):
       - Task 1.1: Định nghĩa types cho QuotaCategory (`CategoryGemini`, `CategoryClaudeGPT`), QuotaState (`ApplicableValid`, `NotApplicable`, `UnknownUncertain`), và struct `ModelQuotaEvaluation`.
         + Action: Viết khai báo type và hằng số trong `internal/automation/model_quota.go`.
         + Inputs: Khái niệm phân nhóm model trong Báo cáo Kiến trúc Mục 3.1 & 3.2.
         + Allowed Edit Scope: `internal/automation/model_quota.go`.
         + Outputs: Enums và cấu trúc dữ liệu cho model quota mapping.
         + Verification Condition: `go vet ./internal/automation/...` biên dịch được types.
         + Task Checkpoint: `wip(P1-A): task 1.1 - define model quota types and states`
       - Task 1.2: Cài đặt hàm `ResolveModelCategory(modelNameOrID string) QuotaCategory` ánh xạ các mã placeholder (`MODEL_PLACEHOLDER_M318`, `gemini-2.5`, `claude-4-6`, ...) về nhóm Quota chuẩn, và hàm `ClassifyQuotaState` phân loại Tam Phân Thái cùng cờ `disabled: true`.
         + Action: Viết hàm ánh xạ chuỗi tên/id model thành `QuotaCategory` và phân loại trạng thái Quota.
         + Inputs: Chuỗi ký tự model ID / placeholder, Quota Snapshot.
         + Allowed Edit Scope: `internal/automation/model_quota.go`.
         + Outputs: Hàm `ResolveModelCategory` và `ClassifyQuotaState`.
         + Verification Condition: `go test -run TestModelQuota ./internal/automation/...`
         + Task Checkpoint: `wip(P1-A): task 1.2 - implement model category and quota classifier`
       - Task 1.3: Cài đặt bộ lọc sơ loại 5 tiêu chí `FilterEligibleCandidates` và thuật toán sắp xếp ưu tiên 4 tầng `SortCandidatesBurnDown` trong `internal/automation/sorter.go`.
         + Action: Hiện thực hóa logic lọc loại trừ ứng viên không đủ điều kiện và thuật toán `sort.SliceStable` với bộ so sánh 4 tầng (Tuần thấp nhất -> Có 5h -> 5h cao nhất -> Thứ tự cố định).
         + Inputs: Danh sách ứng viên thô và quota tương ứng.
         + Allowed Edit Scope: `internal/automation/sorter.go`.
         + Outputs: Struct `CandidateAccount`, hàm `FilterEligibleCandidates` và `SortCandidatesBurnDown`.
         + Verification Condition: `go test -run TestSorter ./internal/automation/...`
         + Task Checkpoint: `wip(P1-A): task 1.3 - implement candidate pre-filtering and burn-down sorter`
    2. Code Inspection & Lint:
       - Review code diff, syntax, linter, typecheck, xác nhận không chạm ngoài `internal/automation/model_quota.go` và `sorter.go`.
    3. Update Tests (Code first, test second):
       - Tạo `internal/automation/model_quota_test.go` và `internal/automation/sorter_test.go` kiểm thử đầy đủ các ca (Gemini, Claude, placeholder, disabled 5h, và tái hiện chính xác bảng 6 tài khoản mẫu A-F).
    4. Build:
       - Chạy `go test -v ./internal/automation/...` và `go build ./...`.
    5. QA (Mini QA / Local Runtime Validation):
       - Chạy unit test suite với cờ `-race` và benchmark đo thời gian thực thi sorter < 1ms cho 50 tài khoản.
       - Evidence target: Log kết quả kiểm thử pass toàn bộ test cases cho Model Quota Resolver và Burn-down Sorter.
    6. Slice Acceptance & Official Milestone Commit:
       - Kiểm tra tiêu chí nghiệm thu P1-A, ghi nhận evidence vào `evidence.md`, cập nhật `actual-status.md`, tạo commit chính thức `feat(P1-A): implement model quota resolver and burn-down sorter`.
  - Implementation Gate:
    - P0 actual-status đã hoàn tất và ghi nhận các module automation ở trạng thái `missing`.
  - Acceptance:
    - Source: `internal/automation/model_quota.go`, `sorter.go` và test files hoàn chỉnh.
    - Runtime/UI: N/A.
    - DB/data: N/A.
    - Behavior test: `go test -v -race ./internal/automation/...` đạt PASS 100%. Bảng mẫu A-F cho kết quả chính xác 100%.
    - Cleanup/quarantine: N/A.
    - Evidence IDs: `E1-P1A-SRC1`, `E1-P1A-TEST1`, `E1-P1A-BENCH1`.
    - Actual-status rows refreshed: `internal/automation/model_quota.go` -> `correct`, `internal/automation/sorter.go` -> `correct`.
  - Evidence Targets: `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-evidence.md#e1---p1-evidence`.
  - Actual-status Update: Cập nhật dòng `model_quota.go` và `sorter.go` sang `correct`.
  - Commit Boundary: Commit sau khi slice P1-A hoàn thành và pass toàn bộ acceptance.

- [ ] P1-B: Dual-Trigger Engine & 3-Retry Controller
  - Goal: Xây dựng cơ chế khôi phục nhanh Native In-Memory 3-Retry qua Connect-RPC (`SendUserCascadeMessage` với empty content list) và bộ điều phối trung tâm `Controller` kết nối Van 1 (Proactive: telemetry 60s) và Van 2 (Reactive: bắt lỗi 429), quản lý qua cổng đơn luồng `SingleFlightGate` (`atomic.Bool`).
  - Scope Boundary:
    - Editable: `internal/automation/retry.go`, `internal/automation/controller.go`
    - Inspect-only: `internal/automation/model_quota.go`, `internal/automation/sorter.go`, `internal/manager/account_manager.go`, `internal/telemetry/`
    - Preserve-only: `internal/manager/account_manager.go` (tái sử dụng nguyên vẹn `SwitchAccount`)
    - Out of scope: `internal/ui/*`
  - Non-Goals: Không thực hiện restart server trong retry; không gọi prompt tiếp tục session lần thứ hai ngoài `SwitchAccount`.
  - Pre-flight Questions:
    - Data source: Telemetry cache (60s loop) và Language Server error channel / stream.
    - Display permission: N/A.
    - DB read flow: Danh sách tài khoản từ file/store.
    - DB write flow: Chuyển tài khoản thông qua `SwitchAccount`.
    - Render location: Gửi thông điệp cập nhật trạng thái ra event channel.
    - UI behavior flow: Phát sinh event cập nhật nhãn trạng thái giao diện.
    - Docker runtime: N/A.
    - Playwright target: N/A.
    - Behavior test: `internal/automation/retry_test.go`, `internal/automation/controller_test.go`.
    - Cleanup/quarantine: N/A.
    - External side effects: Kích hoạt `SwitchAccount` khi đủ điều kiện.
    - N/A notes: None.
  - Work Steps:
    1. Code (Execute leaf atomic tasks in core source files only):
       - Task 1.1: Hiện thực hóa cơ chế Native In-Memory 3-Retry trong `internal/automation/retry.go` gửi Connect-RPC `SendUserCascadeMessage` với empty content list tới conversation ID bị lỗi, tối đa 3 lần cách nhau 2-3s, tôn trọng cờ `retryAt` của Language Server.
         + Action: Viết client gửi Connect-RPC và bộ điều phối `RetryCoordinator`.
         + Inputs: Context, conversation ID, cổng RPC nội bộ.
         + Allowed Edit Scope: `internal/automation/retry.go`.
         + Outputs: Struct `RetryCoordinator` và phương thức `ExecuteRetryLoop`.
         + Verification Condition: `go test -run TestRetry ./internal/automation/...`
         + Task Checkpoint: `wip(P1-B): task 1.1 - implement native in-memory 3-retry via connect-rpc`
       - Task 1.2: Định nghĩa cấu trúc `AutomationController` trong `internal/automation/controller.go` với cờ nguyên tử `enabled`, `isSwitching`, kênh trạng thái `statusChan`, và liên kết tới AccountManager.
         + Action: Khai báo struct và constructor `NewAutomationController`.
         + Inputs: Thiết kế Mục 4 & 9.
         + Allowed Edit Scope: `internal/automation/controller.go`.
         + Outputs: Struct `AutomationController`.
         + Verification Condition: `go vet ./internal/automation/...`
         + Task Checkpoint: `wip(P1-B): task 1.2 - define automation controller lifecycle struct`
       - Task 1.3: Hiện thực hóa van chủ động `EvaluateProactiveTrigger` và cổng đơn luồng `TriggerSwitch` với Single-Flight Gate (kiểm tra `CompareAndSwap` trên `isSwitching`, deduplicate các tín hiệu đồng thời, xử lý khi kho hết tài khoản, kiểm tra ứng viên qua `PollAccount`, và gọi `SwitchAccount`).
         + Action: Viết logic điều phối chuyển đổi tài khoản an toàn.
         + Inputs: Active model, Quota cache, danh sách ứng viên.
         + Allowed Edit Scope: `internal/automation/controller.go`.
         + Outputs: Hàm `EvaluateProactiveTrigger` và `TriggerSwitch`.
         + Verification Condition: `go test -run TestController ./internal/automation/...`
         + Task Checkpoint: `wip(P1-B): task 1.3 - implement dual-trigger and single-flight gate`
    2. Code Inspection & Lint:
       - Rà soát kỹ lưỡng khóa đồng bộ Mutex và `atomic.Bool` bảo đảm không xảy ra Deadlock hay Data Race.
    3. Update Tests (Code first, test second):
       - Tạo `internal/automation/retry_test.go` và `internal/automation/controller_test.go` kiểm thử đầy đủ các nhánh retry (lần 1/2 thành công, lần 3 thất bại), và stress test 100 concurrent triggers với Single-Flight Gate.
    4. Build:
       - Chạy `go test -v -race ./internal/automation/...` và `go build ./...`.
    5. QA (Mini QA / Runtime Validation):
       - Kiểm tra mô phỏng thời gian retry và single-flight deduplication.
       - Evidence target: Test log ghi nhận retry và controller concurrent test đạt PASS.
    6. Slice Acceptance & Official Milestone Commit:
       - Ghi nhận evidence, cập nhật `actual-status.md`, tạo commit `feat(P1-B): implement dual-trigger engine and 3-retry controller`.
  - Implementation Gate:
    - Slice P1-A đã hoàn tất.
  - Acceptance:
    - Source: `internal/automation/retry.go`, `internal/automation/controller.go` và test files hoàn chỉnh.
    - Runtime/UI: N/A.
    - DB/data: N/A.
    - Behavior test: Concurrent single-flight test PASS 100%.
    - Cleanup/quarantine: N/A.
    - Evidence IDs: `E1-P1B-SRC1`, `E1-P1B-TEST1`, `E1-P1B-CONCUR1`.
    - Actual-status rows refreshed: `internal/automation/retry.go` -> `correct`, `internal/automation/controller.go` -> `correct`.
  - Evidence Targets: `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-evidence.md#e1---p1-evidence`.
  - Actual-status Update: Dòng `retry.go` và `controller.go` chuyển sang `correct`.
  - Commit Boundary: Commit sau khi slice P1-B hoàn thành.

- [ ] P1-C: Automation Coordinator & GUI Toggle
  - Goal: Tích hợp widget Toggle công tắc Bật/Tắt và nhãn hiển thị trạng thái Automation động ("Đang giám sát", "Đang thử lại 1/3", "Đang chuyển đổi...", "Chờ an toàn") vào Fyne GUI tại `internal/ui/`.
  - Scope Boundary:
    - Editable: `internal/ui/automation_view.go`, tích hợp gọi từ layout container chính trong `internal/ui/`
    - Inspect-only: `internal/automation/controller.go`
    - Preserve-only: Các layout danh sách tài khoản hiện hữu trên Fyne UI
    - Out of scope: Sửa đổi core backend switcher
  - Non-Goals: Không làm thay đổi giao diện quản lý tài khoản thủ công; Toggle automation xuất hiện như một khối điều khiển mở rộng tiện ích.
  - Pre-flight Questions:
    - Data source: Kênh trạng thái `statusChan` từ `AutomationController`.
    - Display permission: Luôn hiển thị cho người dùng.
    - DB read flow: Đọc trạng thái cấu hình lưu trữ.
    - DB write flow: Lưu trạng thái Bật/Tắt vào cấu hình.
    - Render location: Thanh điều khiển hoặc thanh trạng thái của Fyne main window.
    - UI behavior flow: Khi người dùng bấm Toggle -> gọi `controller.SetEnabled(true/false)`. Khi hệ thống đang chuyển -> hiển thị nhãn "Đang xử lý chuyển đổi", vô hiệu hóa nút chuyển tay để tránh xung đột (Biên 9).
    - Docker runtime: N/A (Desktop Windows GUI).
    - Playwright target: N/A.
    - Behavior test: UI component unit test trong `internal/ui/automation_view_test.go`.
    - Cleanup/quarantine: N/A.
    - External side effects: Cập nhật giao diện Fyne.
    - N/A notes: None.
  - Work Steps:
    1. Code (Execute leaf atomic tasks in core source files only):
       - Task 1.1: Tạo widget `AutomationView` bao gồm switch/check toggle và nhãn trạng thái trong `internal/ui/automation_view.go`.
         + Action: Viết component `AutomationView` kết nối với `AutomationController`.
         + Inputs: Pointer tới `AutomationController`.
         + Allowed Edit Scope: `internal/ui/automation_view.go`.
         + Outputs: Component `AutomationView`.
         + Verification Condition: `go vet ./internal/ui/...`
         + Task Checkpoint: `wip(P1-C): task 1.1 - create fyne automation view component`
       - Task 1.2: Kết nối lắng nghe sự kiện từ `statusChan` để cập nhật nhãn trạng thái giao diện theo thời gian thực an toàn trên UI thread của Fyne.
         + Action: Viết goroutine lắng nghe sự kiện và cập nhật `widget.Label.SetText`.
         + Inputs: Kênh trạng thái của controller.
         + Allowed Edit Scope: `internal/ui/automation_view.go`.
         + Outputs: Hàm `StartStatusListener`.
         + Verification Condition: `go test -run TestAutomationView ./internal/ui/...`
         + Task Checkpoint: `wip(P1-C): task 1.2 - wire status event listener to ui label`
       - Task 1.3: Nhúng `AutomationView` vào layout container của cửa sổ chính Fyne GUI.
         + Action: Chèn `automationView` vào layout container của cửa sổ chính.
         + Inputs: Main window layout.
         + Allowed Edit Scope: `internal/ui/main_window.go` (hoặc file layout tương đương).
         + Outputs: Layout tích hợp hoàn chỉnh.
         + Verification Condition: `go build ./...`
         + Task Checkpoint: `wip(P1-C): task 1.3 - embed automation view into main window`
    2. Code Inspection & Lint:
       - Rà soát diff giao diện, bảo đảm việc cập nhật giao diện tuân thủ thread-safe của Fyne.
    3. Update Tests (Code first, test second):
       - Bổ sung unit test cho logic cập nhật nhãn trạng thái và toggle callback trong `internal/ui/automation_view_test.go`.
    4. Build:
       - Chạy `go build ./...` bảo đảm nhị phân ứng dụng biên dịch hoàn toàn sạch sẽ.
    5. QA (Mini QA / Runtime Validation):
       - Khởi chạy headless test hoặc kiểm tra render logic của Fyne component.
       - Evidence target: Biên dịch nhị phân thành công và test UI logic PASS.
    6. Slice Acceptance & Official Milestone Commit:
       - Ghi nhận evidence, cập nhật `actual-status.md`, tạo commit `feat(P1-C): integrate automation toggle and status indicators into fyne ui`.
  - Implementation Gate:
    - Slice P1-B đã hoàn tất.
  - Acceptance:
    - Source: `internal/ui/automation_view.go` và tích hợp trong UI.
    - Runtime/UI: Giao diện Fyne hiển thị toggle và nhãn trạng thái.
    - DB/data: N/A.
    - Behavior test: Unit test cho view đạt PASS.
    - Cleanup/quarantine: N/A.
    - Evidence IDs: `E1-P1C-SRC1`, `E1-P1C-BUILD1`.
    - Actual-status rows refreshed: `internal/ui/automation_view.go` -> `correct`.
  - Evidence Targets: `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-evidence.md#e1---p1-evidence`.
  - Actual-status Update: Dòng `internal/ui/automation_view.go` chuyển sang `correct`.
  - Commit Boundary: Commit sau khi slice P1-C hoàn thành.

### P2: System Integration, Stress Testing & Acceptance

- Phase Goal: Nghiệm thu toàn diện 14 tình huống biên và an toàn thông qua bộ test harness E2E, thực hiện kiểm thử tải đơn luồng (Single-Flight), rà soát dọn dẹp deadcode và hoàn tất đóng kế hoạch với sự phê duyệt của Supervisor.
- Phase Boundary:
  - In scope: `internal/automation/automation_e2e_test.go`, nghiệm thu chất lượng hệ thống, dọn dẹp deadcode và đóng kế hoạch.
  - Out of scope: Tạo mới tính năng sản phẩm ngoài phạm vi đặc tả kiến trúc.
  - Dependencies: Hoàn tất toàn bộ Slices P1-A, P1-B, P1-C.
- Phase Implementation Rule: Do not implement P2 directly. Implement P2-A, verify it, record evidence, then proceed with P2-B, P2-C, P2-D.
- Ordered Slice List:
  - P2-A: E2E Integration & 14 Edge Cases Simulation Test Harness
  - P2-B: Call supervisor for the implemented-plan acceptance loop
  - P2-C: Remove dead work and deadcode created during this plan
  - P2-D: Close the plan

- [ ] P2-A: E2E Integration & 14 Edge Cases Simulation Test Harness
  - Goal: Xây dựng bộ test harness E2E mô phỏng toàn diện 14 tình huống biên và an toàn quy định trong Mục 8 của Báo cáo Kiến trúc, xác minh sự phối hợp giữa Model Resolver, Burn-down Sorter, 3-Retry, Controller và UI.
  - Scope Boundary:
    - Editable: `internal/automation/automation_e2e_test.go`
    - Inspect-only: `internal/automation/*`, `internal/manager/*`, `internal/telemetry/*`
    - Preserve-only: Core packages
    - Out of scope: Sửa đổi mã nguồn sản phẩm
  - Non-Goals: Không phụ thuộc vào tài khoản Google thật hay kết nối mạng thật; toàn bộ 14 biên được mô phỏng bằng mock harness cô lập 100%.
  - Pre-flight Questions:
    - Data source: Fixtures mô phỏng 14 tình huống biên.
    - Display permission: N/A.
    - DB read flow: In-memory mock store.
    - DB write flow: In-memory mock switch target.
    - Render location: Test output logs.
    - UI behavior flow: N/A.
    - Docker runtime: N/A.
    - Playwright target: N/A.
    - Behavior test: `internal/automation/automation_e2e_test.go`.
    - Cleanup/quarantine: Dọn dẹp mock resources sau khi chạy test.
    - External side effects: None.
    - N/A notes: None.
  - Work Steps:
    1. Code (Execute leaf atomic tasks in core source files only):
       - Task 1.1: Thiết lập Mock Language Server và Mock Telemetry Provider cho phép phát tín hiệu quota giả lập và mã lỗi có cấu trúc trong `internal/automation/automation_e2e_test.go`.
         + Action: Viết mock harness trong `internal/automation/automation_e2e_test.go`.
         + Inputs: Ma trận 14 tình huống biên.
         + Allowed Edit Scope: `internal/automation/automation_e2e_test.go`.
         + Outputs: Structs `MockLangServer` và `MockTelemetry`.
         + Verification Condition: `go test -run TestMockHarnessInit ./internal/automation/...`
         + Task Checkpoint: `wip(P2-A): task 1.1 - build mock e2e harness for 14 edge cases`
       - Task 1.2: Viết test case mô phỏng lần lượt các biên 1 đến 7 (Gemini 5h <= 2%, Claude hết tuần khi đang chạy Gemini, Tuần 1% vs 0%, disabled bucket, retry thành công lần 1/2 vs thất bại lần 3).
         + Action: Cài đặt bài test cho Biên 1 - 7.
         + Inputs: Bảng Mục 8.
         + Allowed Edit Scope: `internal/automation/automation_e2e_test.go`.
         + Outputs: Hàm `TestEdgeCases1To7`.
         + Verification Condition: `go test -run TestEdgeCases1To7 ./internal/automation/...`
         + Task Checkpoint: `wip(P2-A): task 1.2 - verify edge cases 1 to 7`
       - Task 1.3: Viết test case mô phỏng các biên 8 đến 14 (Single-flight khi trùng 429 và <= 2%, người dùng bấm tay đúng lúc auto chuyển, đổi model ngay trước đợt chuyển, ứng viên #1 lỗi token, SwitchAccount fail, hết sạch tài khoản hợp lệ, log phiên cũ dội về trễ).
         + Action: Cài đặt bài test cho Biên 8 - 14.
         + Inputs: Bảng Mục 8.
         + Allowed Edit Scope: `internal/automation/automation_e2e_test.go`.
         + Outputs: Hàm `TestEdgeCases8To14`.
         + Verification Condition: `go test -run TestEdgeCases8To14 ./internal/automation/...`
         + Task Checkpoint: `wip(P2-A): task 1.3 - verify edge cases 8 to 14`
    2. Code Inspection & Lint:
       - Rà soát test coverage, đảm bảo không có trường hợp rò rỉ goroutine hay timeout trong mock harness.
    3. Update Tests (Code first, test second):
       - Củng cố toàn bộ suite `automation_e2e_test.go` chạy độc lập, kích hoạt qua `go test -v -race ./internal/automation/...`.
    4. Build:
       - Chạy kiểm tra biên dịch toàn dự án `go test -v ./...` và `go build ./...`.
    5. QA (Mini QA / Runtime Validation):
       - Chạy toàn bộ test suite tự động với cờ `-race` và ghi lại thời gian chạy cũng như số lượng test cases vượt qua.
       - Evidence target: Báo cáo kết quả E2E test harness pass 14/14 tình huống biên.
    6. Slice Acceptance & Official Milestone Commit:
       - Ghi nhận evidence, cập nhật `actual-status.md`, tạo commit `feat(P2-A): add comprehensive e2e test harness verifying 14 edge cases`.
  - Implementation Gate:
    - Toàn bộ Slices P1-A, P1-B, P1-C đã hoàn tất.
  - Acceptance:
    - Source: `internal/automation/automation_e2e_test.go`.
    - Runtime/UI: N/A.
    - DB/data: N/A.
    - Behavior test: 14/14 tình huống biên đạt PASS 100%.
    - Cleanup/quarantine: Mock server tự đóng cổng sạch sẽ sau khi test xong.
    - Evidence IDs: `E2-P2A-SRC1`, `E2-P2A-TEST14`.
    - Actual-status rows refreshed: `internal/automation/` -> `correct`.
  - Evidence Targets: `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-evidence.md#e2---p2-evidence`.
  - Actual-status Update: Toàn bộ module automation chuyển sang `correct`.
  - Commit Boundary: Commit sau khi slice P2-A hoàn thành.

- [ ] P2-B: Call supervisor for the implemented-plan acceptance loop.
  - Goal: Xác minh toàn bộ kết quả công việc đã hoàn thành so với kế hoạch đã duyệt, quyết định actual-status, evidence, benchmark, các file đã thay đổi, và kết quả kiểm thử trước khi đóng kế hoạch.
  - Work Steps:
    1. Gọi supervisor skill để review toàn bộ kế hoạch và mã nguồn đã triển khai.
    2. Nếu supervisor yêu cầu chỉnh sửa, quay lại workflow/skill chịu trách nhiệm cho phạm vi lỗi tương ứng.
    3. Chạy lại supervisor review sau khi fix xong.
    4. Lặp lại cho đến khi supervisor pass hoặc ghi nhận blocker.
  - Implementation Gate: Toàn bộ các slice triển khai trước đó phải hoàn thành hoặc được ghi nhận blocker rõ ràng.
  - Acceptance: Supervisor review thông qua với VERDICT: PASS.

- [ ] P2-C: Remove dead work and deadcode created during this plan.
  - Goal: Bảo đảm diff cuối cùng chỉ chứa các artifact thực sự phục vụ cho kế hoạch đã duyệt.
  - Work Steps:
    1. Rà soát lại tất cả các file, đoạn code, test và tài liệu được tạo ra trong kế hoạch này.
    2. Loại bỏ hoặc viết lại bất kỳ artifact, deadcode hoặc placeholder lỗi thời nào.
    3. Xác nhận không còn phương án bị loại bỏ hay helper thừa thãi trong diff cuối cùng.
    4. Yêu cầu supervisor kiểm tra lại việc dọn dẹp.
  - Implementation Gate: Chỉ dọn dẹp các artifact được tạo ra bởi kế hoạch này.
  - Acceptance: `git diff/status` sạch sẽ, không có deadcode phát sinh từ plan.

- [ ] P2-D: Close the plan.
  - Goal: Hoàn tất kiểm thử cuối cùng, ghi nhận evidence, benchmark, commit chính thức và cập nhật trạng thái đóng kế hoạch.
  - Work Steps:
    1. Chạy bài kiểm tra toàn diện cuối cùng (`go test -v -race ./...` và `go build -v ./...`).
    2. Ghi nhận evidence cuối cùng, benchmark và commit hash vào `evidence.md` và `actual-status.md`.
    3. Cập nhật Status trong metadata của Plan thành `complete`.
    4. Tạo commit hoàn tất kế hoạch.
  - Implementation Gate: P2-B và P2-C phải hoàn tất đạt yêu cầu.
  - Acceptance: Final evidence được ghi nhận đầy đủ, commit hoàn tất tồn tại trên git tree.

## Risk Notes

1. **Độ trễ tái sinh tiến trình Language Server (8 giây):** Quá trình chuyển đổi tài khoản gọi `SwitchAccount` bao gồm lệnh `taskkill` và độ trễ chờ cố định 8 giây để `monitorLsCrashInternal` của Antigravity hồi sinh tiến trình an toàn. Bất kỳ nỗ lực kết nối lại hoặc tiếp tục session sớm hơn 8s đều sẽ thất bại do socket chưa mở.
2. **Khôi phục in-memory (Nút "Try again") chỉ xử lý lỗi tạm thời:** Cơ chế retry Connect-RPC chỉ hiệu quả khi gặp gián đoạn nhất thời (temporary rate limit, network hiccup). Nếu tài khoản cạn sạch hạn ngạch tuần ($0\%$), việc retry sẽ nhanh chóng cạn kiệt 3 lần và chuyển sang quy trình đổi tài khoản.
3. **Mất kết nối mạng hoặc lỗi Google API:** Khi telemetry trả về dữ liệu rỗng hoặc timeout (`UNKNOWN_UNCERTAIN`), controller tuyệt đối không được phép suy diễn thành hết quota để chuyển tài khoản mù quáng, tránh làm cạn kiệt tài nguyên của người dùng.
4. **Xung đột điều khiển giữa người dùng và máy:** Nếu người dùng bấm đổi tài khoản bằng tay trên GUI đúng thời điểm automation đang kích hoạt, Mutex bảo đảm yêu cầu thủ công nhận được thông báo "Đang xử lý chuyển đổi", không sinh ra hai luồng taskkill song song.
