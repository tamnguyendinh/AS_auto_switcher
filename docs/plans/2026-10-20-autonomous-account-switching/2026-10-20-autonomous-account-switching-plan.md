# Autonomous Account Switching Plan

## Metadata

- Date: `2026-10-20`
- Status: `planned — tài liệu đã sửa R1; chưa triển khai sản phẩm`
- Revision: `R1 — sửa theo review F1–F8 và yêu cầu Owner; chỉ thay đổi tài liệu`
- Source baseline: `2e4c8939037871f7247f6030d03a69a0c90c89ab`
- Review đầu vào: `reports/supervisor/rp_supervisor_261020_144149_by_gpt-6_autonomous_plan_review.md`
- Plan: `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-plan.md`
- Evidence: `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-evidence.md`
- Benchmark: `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-benchmark.md`
- Actual status: `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-actual-status.md`

## Goal

Nối automation vào telemetry, model discovery, GUI toggle và engine chuyển tài khoản hiện có. Tự chuyển khi quota của model đang dùng đạt 5h <= 2% hoặc tuần = 0%; khi gặp lỗi giới hạn thì khôi phục tối đa ba lần theo kết quả thực thi của Antigravity trước khi chuyển. Chọn tài khoản theo tuần tăng dần, ưu tiên có 5h khi tuần bằng nhau, sau đó 5h giảm dần và thứ tự cố định. Giữ nguyên `SwitchAccount`, khoảng chờ 8 giây và continuation hai giờ đã nằm trong engine.

Bản R1 sửa các điểm nối còn thiếu trong kế hoạch; không mở đợt audit, refactor hoặc sửa lỗi engine. Quyết định Owner trong cuộc thảo luận và R1 này được ưu tiên hơn các mô tả trái nhau trong báo cáo kiến trúc nguồn. Không sửa SPEC nguồn hoặc bộ Phase 7 cũ trong lượt sửa tài liệu này.

## Rules

- Complete P0 actual status before implementation work.
- Update each checklist item immediately when it is completed.
- Record evidence as work completes.
- Record benchmarkable counts or measurements when they are taken.
- Update later phase status assumptions, next actions, and work steps when actual-status evidence changes the repo state.
- After completing a phase or implementation slice and refreshing `actual-status.md`, update the next affected phase's work steps as needed to match the latest repo reality, while preserving that phase's original goal, scope, acceptance criteria, and major phase order.
- Với thay đổi GUI, build và chạy cửa sổ Fyne native để kiểm tra binding thực. Headless component test chỉ là kiểm tra bổ sung. Docker và browser Playwright không áp dụng cho Fyne.
- Các lệnh build/test trong kế hoạch chỉ thực hiện khi triển khai các slice code. Sửa tài liệu R1 chỉ kiểm tra nội dung, links và diff, không chạy build/test sản phẩm.
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
  1. Code: Trong implementation slice chỉ sửa production; P2-A là slice test-only đã xác định riêng. Mỗi leaf task có Action, Inputs, Allowed Edit Scope, Outputs, Verification Condition và micro WIP checkpoint, không phải slice completion commit.
  2. Code Inspection & Lint: Review diff, syntax, linter, typecheck, and ensure edits remain strictly within assigned boundary.
  3. Update Tests: Following Master Rule 6 (Code first, test second), add or update unit/integration tests to verify the newly implemented code behavior.
  4. Build: Run the repository's full build to ensure type contracts, package bundles, and binaries compile cleanly.
  5. QA (Mini QA): Verify user-visible runtime behavior (desktop app GUI, API, CLI, or simulated test harness evidence).
  6. Slice Acceptance & Official Milestone Commit: Verify slice acceptance criteria, record evidence in `evidence.md`, refresh `actual-status.md`, and create the official slice completion commit (e.g. `feat(<slice_id>): <slice outcome>`).
- Hidden fallback is forbidden. Prefer a visible failure over a fallback that hides a broken primary path.
- When touching DB-backed content, verify the full loop when applicable: UI input -> submit action -> DB write -> DB read after reload/new request -> correct UI render or omission. If there is no UI, replace UI steps with the real caller/consumer flow.
- Tests must prove product behavior. Delete or replace tests that only assert implementation details, helper output, static DOM existence, or mocked plumbing without proving trigger -> process -> observable result.
- Go tests đặt cạnh source dưới tên `_test.go`, fixtures ở `testdata/`; không đưa mock vào binary sản phẩm. Test tự động dùng fake RPC/executor, store và lock riêng; không taskkill, đổi credential hoặc gửi prompt vào IDE thật.
- Evidence chỉ ghi kết quả đã quan sát. Các mục E1/E2 chưa thực hiện phải ghi `PLANNED / NOT RUN`, không ghi trước PASS hoặc số đo.
- P0 hoàn tất về khảo sát/tài liệu không có nghĩa P1/P2 đã thực hiện. Giữ nguyên hành vi đóng ứng dụng, layout/theme và continuation hiện tại.

## Problem

Trong phiên bản Antigravity v2.24.0, người dùng lập trình gặp phải sự cố gián đoạn luồng làm việc do cạn kiệt Quota:
1. **Chưa có tự động điều phối:** Khi quota cạn hoặc gặp lỗi giới hạn, người dùng vẫn phải bấm chuyển. Engine hiện có đã tự gửi continuation sau khi chuyển; phần thiếu là tự quyết định và kích hoạt luồng này.
2. **Yêu cầu kỹ thuật thực tế trên Antigravity v2.24.0:**
   - Engine hiện tại dùng thứ tự ToS -> WinCred -> restart Language Server -> chờ 8 giây -> xác nhận email -> continuation. Giữ nguyên cơ chế đó; không thiết kế ReAuth khác hoặc suy ra tổng thời gian switch bằng 8 giây.
   - Quá trình giám sát cần tận dụng gói `internal/telemetry` có sẵn (gọi Google API mỗi 60 giây và theo dõi phần trăm quota tường minh) thay vì quét log văn bản thụ động.
   - Sắp xếp ứng viên tài khoản phải có cơ chế liên kết với Model đang chạy trong phiên làm việc hiện tại (Gemini vs Claude/GPT-OSS), tránh chuyển đổi tài khoản mù quáng hoặc không tối ưu.
   - Hệ thống cần tích hợp khay điều khiển và công tắc chuyển đổi trực tiếp trên Fyne GUI có sẵn.
3. **Retry nguyên bản chưa được nối vào Switcher:** Nút Try again gửi `SendUserCascadeMessage` với `items: []` tới hội thoại lỗi. Automation cần đọc kết quả lượt chạy từ stream, giới hạn ba recovery actions; HTTP chấp nhận lệnh không phải kết quả khôi phục. Không đặt cam kết phục hồi dưới 500ms.
4. **Binding còn thiếu:** GUI đã có toggle Tự động; cache quota hiện chưa giữ cờ `disabled`; callback telemetry và các caller chuyển tay cần được nối vào controller, không tạo lại chức năng nền.

## Scope

### In Scope
- Tạo mới package `internal/automation/`:
  - `internal/automation/model_quota.go`: Dùng `DiscoverActiveAgentModel` hiện có lấy ID, đối chiếu `GetCascadeModelConfigData` để chọn nhóm quota; không hardcode bảng placeholder. Phân loại dữ liệu từ pipeline telemetry hiện có.
  - `internal/automation/sorter.go`: Bộ lọc sơ loại ứng viên (5 tiêu chí) và thuật toán sắp xếp ưu tiên Burn-down Sorter (4 tầng ưu tiên: Tuần thấp nhất -> Có 5h -> 5h cao nhất -> Thứ tự cố định).
  - `internal/automation/retry.go`: Giám sát stream `StreamAgentStateUpdates`, bắt lỗi có cấu trúc (`429`, `RESOURCE_EXHAUSTED`, `RATE_LIMIT_EXCEEDED`, `QUOTA_EXHAUSTED`), thực hiện Native In-Memory 3-Retry qua Connect-RPC `SendUserCascadeMessage` với empty content list.
  - `internal/automation/controller.go`: Bộ điều phối kích hoạt kép (Dual-Trigger Engine: Van chủ động 60s telemetry + Van bị động lỗi 429), cổng kiểm soát đơn luồng (Single-Flight Concurrency Gate) tránh xung đột lệnh chuyển, công tắc nguyên tử `atomic.Bool` quản lý trạng thái Bật/Tắt.
- Cập nhật giao diện người dùng Fyne GUI tại `internal/ui/`:
  - `internal/ui/app.go`, `internal/ui/window.go`: Nối toggle Tự động có sẵn, vòng đời controller, callback telemetry và status bar. Giữ thiết kế/layout hiện tại, không tạo `AutomationView` hoặc toggle thứ hai.
- Các phần nối dữ liệu/caller được phép bổ sung:
  - `internal/telemetry/client.go`: Chỉ thêm metadata `disabled` vào DTO/parser/RawBuckets để truyền qua cache; giữ số liệu hiển thị, HTTP fetch, endpoint và vòng quét.
  - `internal/connector/automation_rpc.go` (mới): Adapter đọc model config, native empty-items Retry và Connect state stream; tái dùng discovery/client/CSRF. Không thay phương thức gửi prompt text hiện có.
  - `internal/automation/switch_gate_windows.go` (mới): Wrapper khóa thao tác ở caller boundary, dùng chung trong process và giữa GUI/CLI qua Windows named mutex theo user/credential scope; không phụ thuộc store directory và không dùng lại GUI single-instance mutex.
  - `cmd/switcher/main.go`: Chỉ nối nhánh `--switch` qua wrapper dùng chung; giữ flags và engine.
- Kiểm thử tự động hóa toàn diện:
  - `internal/automation/model_quota_test.go`, `internal/automation/sorter_test.go`, `internal/automation/retry_test.go`, `internal/automation/controller_test.go`.
  - `internal/automation/automation_e2e_test.go`: Test harness mô phỏng 14 tình huống biên và an toàn.

### Out of Scope / Preserve Only
- `internal/manager/account_manager.go`: Giữ nguyên và tái sử dụng hàm `SwitchAccount()`. Các caller đi qua shared wrapper rồi gọi nguyên hàm này, không sửa đổi engine.
- `internal/reloader/`: Giữ nguyên cơ chế quét SQLite và phục hồi active threads trong 2 giờ (`continuation.go`). Controller không can thiệp hay gọi tiếp tục lần thứ hai vì `SwitchAccount` đã bao hàm việc này.
- `internal/telemetry/loop.go`, `pool.go`, `cache.go`: Giữ nguyên client polling, cache và loop 60s. Ngoại lệ hẹp duy nhất của package là pass-through metadata trong `client.go` nêu trên.
- `internal/connector/api_client.go`, `discovery.go`, `endpoints.go`: Tái sử dụng, không viết lại resolver hoặc thay behavior gửi prompt continuation.
- `internal/ui/themes.go`, `tray.go`, widget rendering, preferences, store/scanner/WinCred và scripts build: preserve-only. Toggle khởi động mặc định tắt như hiện tại; không bổ sung persistence hoặc thay close-to-exit.

## Non-Goals

- Không triển khai daemon systray độc lập ngoài Fyne GUI.
- Không can thiệp vào mã nguồn nhị phân của Antigravity IDE hay Language Server.
- Không triển khai hoán đổi nóng RPC ReAuth bỏ qua restart tiến trình (bảo lưu bắt buộc: taskkill + 8s wait).
- Không chuyển đổi tài khoản khi model đang chạy khác nhóm model bị cạn kiệt (ví dụ: Claude hết tuần nhưng phiên đang chạy Gemini thì tuyệt đối không chuyển).
- Không đoán mò quota từ ký hiệu `--%` hay trạng thái mạng lỗi.

## Requirements

1. **Active Model Binding (R1):** Mọi quyết định kiểm tra ngưỡng quota và xếp hạng ứng viên phải dựa trên model của phiên làm việc hiện tại (`Gemini` -> Nhóm Gemini 5h & Tuần; `Claude/GPT-OSS` -> Nhóm Claude 5h & Tuần).
   - Adapter gọi resolver hiện có lấy ID rồi join với `clientModelConfigs[].modelOrAlias.model`/label từ API. Hàm classifier nhận cấu hình đã resolve, không đoán `M318` bằng bảng cố định. Model đổi thì tính lại, kiểm tra lại trước switch.
2. **Tam Phân Thái Quota (R2):** Dữ liệu quota từ Google API được chuẩn hóa thành 3 trạng thái:
   - `APPLICABLE_VALID`: Có bucket 5h đang mở, fraction hợp lệ ($0.0 \le \text{fraction} \le 1.0$), kiểm tra cả $5\text{h} \le 2\%$ và $\text{Tuần} = 0\%$.
   - `NOT_APPLICABLE`: Chỉ có bucket tuần hoặc bucket 5h có cờ `disabled: true`, bỏ qua điều kiện 5h, chỉ theo dõi $\text{Tuần} = 0\%$.
   - `UNKNOWN_UNCERTAIN`: Dữ liệu khuyết/lỗi không tạo trigger từ quota đó. Nhánh reactive vẫn hoạt động với lỗi thực thi đủ điều kiện và ứng viên dùng được; không chặn toàn bộ automation vì telemetry lỗi.
   - Luồng metadata: provider response -> `rawQuotaBucket.disabled` -> `ModelQuotaBucket.Disabled`/`RawBuckets` -> cache/callback -> classifier. Cache cũ vẫn đọc được; sau bật dùng `TriggerNow`/callback hoặc `PollAccount` trước quyết định, không tạo TTL/polling engine mới. Khi phản hồi thành công chỉ có tuần thì dùng tuần; khi disabled thì bỏ qua 5h trong lần đánh giá đó. So sánh fraction gốc, không dùng badge hoặc số làm tròn.
3. **Dual-Trigger Architecture & Single-Flight Gate (R3):**
   - Van chủ động (Proactive): Telemetry quét định kỳ 60s, kích hoạt chuyển khi $5\text{h} \le 2\%$ hoặc $\text{Tuần} = 0\%$.
   - Van bị động (Reactive): Bắt lỗi thực thi `429`/`RESOURCE_EXHAUSTED`, kích hoạt 3 lần Native Retry. Thất bại cả 3 lần -> Kích hoạt chuyển khẩn cấp.
   - Single-Flight Gate: Hợp nhất các tín hiệu xảy ra đồng thời thành 1 đợt chuyển duy nhất.
4. **Native Retry theo execution outcome (R4):** Gửi đúng `items: []` tới conversation ID lỗi, đợi cập nhật thực thi mới rồi quyết định lần kế tiếp; tối đa ba recovery actions của automation. Không tính HTTP ACK hoặc số update lặp thành kết quả thành công/thất bại. Antigravity đang native retry thì đợi theo `attempt/maxAttempts/retryAt`, không thêm lịch gửi cách nhau cố định. Không restart trong retry. Nếu proactive đã đủ điều kiện thì ngừng phát retry và hợp nhất một đợt switch.
5. **Burn-down Sorter (R5):** Ứng viên phải vượt qua 5 tiêu chí lọc, sau đó sắp xếp theo: (1) Quota Tuần thấp nhất trước -> (2) Ưu tiên tài khoản có áp dụng 5h -> (3) Quota 5h cao nhất trước -> (4) Giữ nguyên thứ tự danh bạ cố định.
6. **Xử lý 14 Tình Huống Biên & An Toàn (R6):** Dùng ma trận ở cuối plan R1 (khác nhóm model, disabled 5h, deduplication, GUI/CLI contention và chờ ứng viên). Các mô tả mâu thuẫn của bản kiến trúc cũ không ghi đè R1.
7. **GUI Controls (R7):** Giao diện Fyne có Toggle công tắc Bật/Tắt rõ ràng và nhãn hiển thị trạng thái động của Controller.

### Integration contracts — dùng cơ chế có sẵn

- **Nguồn hội thoại/stream:** Khi bật, dùng `ScanActiveConversations2h` hiện có lấy IDs; cập nhật tập IDs cùng lượt telemetry và sau switch, không tạo lại snapshot chỉ để giám sát. Subscribe `StreamAgentStateUpdates` với `conversationId`, `subscriberId`, page bounds nhỏ và `disableRehydration: true`, đọc Connect streaming JSON bằng cổng/CSRF discovery hiện có. Theo dõi status, terminationReason, lỗi cuối và execution status. Không ghi nội dung chat/token vào evidence.
- **Retry outcome:** Native UI nhận lỗi khi status IDLE và terminationReason ERROR cùng lastStepError. Sau send, đợi lượt/execution mới; kết thúc không lỗi thì dừng recovery, kết thúc lỗi giới hạn mới tính thất bại kế tiếp. Đang chạy/native retry thì chờ. Lỗi stream/cancel không tự tính hết quota. Event trùng theo session/conversation/execution/step chỉ xử lý một lần; sau switch loại event cũ và reconnect bằng discovery mới.
- **Adapter mới, method cũ giữ nguyên:** Empty-items Retry cần phương thức riêng; method text hiện có tạo một item ngay cả khi chuỗi rỗng. Streaming dùng transport hiện có với context cancellation, không lấy timeout unary 3s làm hạn toàn subscription. Reconnect theo cơ chế native đã khảo sát, không thêm quota cooldown.
- **Shared switch gate:** GUI chuyển tay, CLI --switch và auto cùng wrapper acquire -> gọi nguyên `mgr.SwitchAccount(ctx, email, false)` -> trả kết quả -> release. Guard đồng bộ cả manual-first và auto-first, giữ qua continuation. Named mutex dùng user/credential scope, wait không xếp hàng lệnh cũ; bảo đảm OS-thread ownership của acquire/release trong Go, release/close bằng defer. Abandoned lock đọc lại session trước thao tác. Retry không gửi khi switch đang diễn ra; manual thay phiên invalidates recovery cũ.
- **App lifecycle:** App tạo controller từ manager/pool/cache hiện hữu. Trong callback duy nhất đang đặt ở `App.Init`, giữ quota UI refresh và chuyển cùng results tới controller theo đường không chặn. Không gọi SetUpdateCallback lần hai làm mất UI callback, không tạo ticker/pool mới. App.Close cancel stream/controller/status listener rồi dừng loop sẵn có.
- **Toggle/status:** Callback toggle hiện có và SetAutoSwitchEnabled dùng chung setter, tránh đệ quy. Bật gọi TriggerNow; tắt ngừng trigger/recovery mới nhưng không cắt engine đã bắt đầu thay phiên. Status cập nhật bằng fyne.Do vào status bar hiện có; sau auto switch cập nhật overview/card như manual. Không đổi layout/theme hoặc persist toggle.
- **Ứng viên:** Có 5h thì 5h > 2% và tuần > 0%; chỉ tuần thì tuần > 0%. Loại chính active account, credential không dùng được và ứng viên vừa thất bại. Dùng thứ tự store cố định làm tie-break. PollAccount ứng viên trước khi thay phiên; không ghi thử token vào WinCred. Không có ứng viên thì báo lý do và chờ vòng telemetry cũ; không suy ra tất cả hết quota.
- **Kết quả switch:** Lỗi từ engine kết thúc đợt và hiển thị lỗi, không tự xoay vô điều kiện sau thao tác đã thay credential. Continuation đã nằm trong engine và chỉ được gọi một lần. Event/ý định đã xử lý không được phát lại ngay khi mở gate.

Ví dụ sorter (mọi số liệu cùng nhóm model đang dùng):

| Tài khoản | Tuần | 5h | Kết quả mong đợi |
|---|---:|---:|---|
| A | 3% | 80% | 1 |
| B | 3% | 40% | 2 |
| C | 3% | Không áp dụng | 3 |
| D | 20% | 90% | 4 |
| E | 1% | 1% | Loại vì 5h <= 2% |
| F | 0% | 100% | Loại vì tuần = 0% |

Không thêm tầng Pristine riêng. Các hàng này là kỳ vọng nghiệp vụ, không phải kết quả test đã chạy.

## Acceptance Criteria

- [ ] Automation adapters/controller/gate được nối vào app thật theo Integration contracts; không chỉ tạo đủ file.
- [ ] 100% các bài kiểm thử đơn vị (`*_test.go`) cho từng module được triển khai độc lập, kiểm tra đầy đủ các nhánh logic, tam phân thái, cờ `disabled: true`, và 4 tầng ưu tiên Burn-down Sorter.
- [ ] Bộ điều phối `controller.go` bảo đảm tính đơn luồng (Single-Flight Gate), không bao giờ kích hoạt 2 tiến trình `SwitchAccount` đồng thời khi van chủ động và bị động cùng phát tín hiệu.
- [ ] Giao diện Fyne GUI hiển thị Toggle Automation và trạng thái hoạt động chính xác theo thời gian thực.
- [ ] Ma trận 14 tình huống biên được bao phủ đầy đủ bằng test kịch bản tự động (Integration Test Harness).
- [ ] GUI build với CGO/GCC và `-tags gui`, có native runtime check cho toggle/callback/lifecycle; CLI build riêng. Mock scenario tests được ghi tách khỏi runtime thật.
- [ ] Dữ liệu disabled đi qua production parser/cache; ID model qua config API; HTTP ACK rồi execution lỗi vẫn được xử lý đúng.
- [ ] Manual GUI/CLI và automation cùng gate, một callback quota phục vụ cả UI và automation, một toggle hiện hữu, engine manager/reloader giữ nguyên.

## Checklist

- [x] P0-A: Baseline & Actual Status Discovery — R1 corrected, documentation only
  - Goal: Rà soát hiện trạng thực tế, xác lập baseline trước khi thực thi.
  - Scope Boundary: sửa bốn file kế hoạch; inspect source/reports; preserve toàn bộ product và dữ liệu tài khoản.
  - Non-Goals: không chạy test/build, không triển khai automation, không gửi Retry/switch thật.
  - Pre-flight Questions: source/review là input; output chỉ tài liệu; không DB write hoặc runtime UI; Docker/Playwright N/A; không tạo test artifacts.
  - Work Steps: inspect source-of-truth files (`internal/manager/account_manager.go`, `internal/telemetry/`, `internal/reloader/`, `internal/ui/`), classify each surface, record missing pieces (`internal/automation/`), and update later phase status assumptions, next actions, and work steps from evidence.
  - Implementation Gate: no implementation or editing starts until `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-actual-status.md` has a final P0 decision.
  - Acceptance: actual status xác định toggle hiện có là unbound; metadata disabled là partial; automation là missing; các caller/callback được giao đúng file. P0 hoàn tất không đánh dấu code/test sản phẩm hoàn thành.
  - Evidence Targets: `E0-P0A-SRC1` đến `E0-P0A-SRC7`, `E0-P0A-REV1`, `E0-P0A-DOC1`.
  - Actual-status Update: R1 corrected; sản phẩm chưa triển khai.
  - Commit Boundary: tài liệu thuộc bộ plan; không commit source hoặc thay đổi ngoài phạm vi.

### P1: Autonomous Failover Engine Implementation

- Phase Goal: Bổ sung adapter/sorter/controller và nối vào model resolver, telemetry, engine và GUI toggle hiện hữu.
- Phase Boundary:
  - In scope: automation modules/gate, connector adapter, quota metadata pass-through, App/toggle/CLI binding và tests như Scope.
  - Out of scope: nghiệp vụ SwitchAccount/reloader, telemetry fetch/cache/loop implementation và UI design.
  - Dependencies: Types từ `internal/telemetry` và cấu trúc tài khoản từ `internal/manager`.
- Phase Implementation Rule: Do not implement P1 directly. Implement P1-A, verify it, record evidence, refresh actual-status, commit; then continue to P1-B, then P1-C.
- Ordered Slice List:
  - P1-A: Model Quota Resolver & Burn-down Sorter
  - P1-B: Dual-Trigger Engine & 3-Retry Controller
  - P1-C: Automation Coordinator & GUI Toggle

- [x] P1-A: Model Quota Resolver & Burn-down Sorter
  - Goal: Hiện thực hóa logic nhận diện Active Model, ánh xạ nhóm Quota, Tam Phân Thái hạn ngạch, bộ lọc sơ loại 5 tiêu chí, và thuật toán sắp xếp ứng viên 4 tầng ưu tiên Burn-down Sorter.
  - Scope Boundary:
    - Editable: `internal/automation/model_quota.go`, `sorter.go`, `internal/connector/automation_rpc.go`, `internal/telemetry/client.go` chỉ metadata; tests cạnh package ở bước 3.
    - Inspect-only: `internal/telemetry/cache.go`, `pool.go`, `internal/connector/api_client.go`, `discovery.go`, `internal/manager/account_manager.go`
    - Preserve-only: telemetry client fetch/cache/loop behavior, model resolver, manager và reloader
    - Out of scope: `internal/automation/controller.go`, `internal/ui/*`
  - Non-Goals: Sorter/classifier thuần không gọi mạng. Adapter model dùng connector hiện có; không viết lại model resolver, không tạo telemetry engine mới.
  - Pre-flight Questions:
    - Data source: Model info từ connector hiện có và Quota Snapshot từ `telemetry.QuotaCache`.
    - Display permission: N/A.
    - DB read flow: Đọc trạng thái tài khoản từ file/store.
    - DB write flow: N/A.
    - Render location: N/A.
    - UI behavior flow: N/A.
    - Docker runtime: N/A (Desktop Go).
    - Playwright target: N/A.
    - Behavior test: `internal/automation/model_quota_test.go`, `internal/automation/sorter_test.go`.
    - Cleanup/quarantine: Mock HTTP server đóng và temp cache/testdata riêng được dọn bằng test cleanup.
    - External side effects: Adapter production chỉ đọc model config; tests dùng mock HTTP/temp cache, không gọi tài khoản thật.
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
       - Task 1.2: Nối model discovery và quota metadata thật trước classifier.
         + Action: Thêm method đọc model config trong file connector mới; gọi DiscoverActiveAgentModel hiện có, join ID với config API, rồi phân nhóm. Trong client.go quota, truyền disabled qua raw/normalized bucket/RawBuckets để cache giữ được; không đổi các số fraction hiển thị. Classifier phân biệt thiếu 5h trong response thành công với lỗi truy vấn.
         + Inputs: Client/ServerInfo hiện có, model config response và quota JSON.
         + Allowed Edit Scope: `internal/connector/automation_rpc.go`, `internal/telemetry/client.go` metadata-only, `internal/automation/model_quota.go`.
         + Outputs: Model category động và quota metadata đủ cho ClassifyQuotaState; không hardcode M318.
         + Verification Condition: Inspect đường API -> cache -> classifier, `go vet ./internal/connector/... ./internal/telemetry/... ./internal/automation/...`; behavior tests ở bước 3.
         + Task Checkpoint: `wip(P1-A): task 1.2 - implement model category and quota classifier`
       - Task 1.3: Cài đặt bộ lọc sơ loại 5 tiêu chí `FilterEligibleCandidates` và thuật toán sắp xếp ưu tiên 4 tầng `SortCandidatesBurnDown` trong `internal/automation/sorter.go`.
         + Action: Hiện thực hóa logic lọc loại trừ ứng viên không đủ điều kiện và thuật toán `sort.SliceStable` với bộ so sánh 4 tầng (Tuần thấp nhất -> Có 5h -> 5h cao nhất -> Thứ tự cố định).
         + Inputs: Danh sách ứng viên thô và quota tương ứng.
         + Allowed Edit Scope: `internal/automation/sorter.go`.
         + Outputs: Struct `CandidateAccount`, hàm `FilterEligibleCandidates` và `SortCandidatesBurnDown`.
         + Verification Condition: Review bảng A-F/ngưỡng fraction; `go vet ./internal/automation/...`; behavior tests ở bước 3.
         + Task Checkpoint: `wip(P1-A): task 1.3 - implement candidate pre-filtering and burn-down sorter`
    2. Code Inspection & Lint:
       - Review diff đúng scope P1-A, telemetry chỉ thêm metadata, không đổi client fetch/loop/resolver/engine/UI.
    3. Update Tests (Code first, test second):
       - Tạo `internal/automation/model_quota_test.go`, `sorter_test.go`, `internal/connector/automation_rpc_test.go`; cập nhật `internal/telemetry/quota_test.go`/`telemetry_test.go` cho provider JSON -> parser -> cache round-trip -> classifier (disabled/missing bucket/error). Kiểm tra config API ánh xạ ID động, không chỉ fixture tên model; bảng A-F ở Evidence Targets vẫn là kết quả mong đợi, chưa phải PASS.
    4. Build:
       - Khi triển khai: `go test ./internal/connector/... ./internal/telemetry/... ./internal/automation/...`; race cùng các package với CGO/GCC bật; CLI compile `go build ./cmd/switcher`. GUI tag/runtime được kiểm tra ở P1-C.
    5. QA (Mini QA / Local Runtime Validation):
       - Chạy kiểm tra pipeline dữ liệu và regression quota bằng fixture cô lập; race với CGO/GCC sẵn có. Nếu đo sorter thì ghi số đo thực, không đặt ngưỡng tốc độ mới ngoài yêu cầu Owner.
       - Evidence target: Log kết quả kiểm thử pass toàn bộ test cases cho Model Quota Resolver và Burn-down Sorter.
    6. Slice Acceptance & Official Milestone Commit:
       - Kiểm tra tiêu chí nghiệm thu P1-A, ghi nhận evidence vào `evidence.md`, cập nhật `actual-status.md`, tạo commit chính thức `feat(P1-A): implement model quota resolver and burn-down sorter`.
  - Implementation Gate:
    - P0 actual-status đã hoàn tất và ghi nhận các module automation ở trạng thái `missing`.
  - Acceptance:
    - Source: model_quota/sorter, connector config adapter và metadata quota pass-through cùng tests được hoàn tất trong slice.
    - Runtime/UI: N/A.
    - DB/data: N/A.
    - Behavior test: Connector/config và provider-to-cache metadata paths đúng; bảng A-F đúng qua production classifier/sorter.
    - Cleanup/quarantine: HTTP server/temp cache thử được dọn; không đụng store thật.
    - Evidence IDs: `E1-P1A-SRC1`, `E1-P1A-TEST1`, `E1-P1A-BENCH1`.
    - Actual-status rows chỉ đổi sau acceptance: quota metadata, model adapter, model_quota/sorter; app/controller vẫn chưa nối.
  - Evidence Targets: `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-evidence.md#e1---p1-evidence`.
  - Actual-status Update: Cập nhật dòng `model_quota.go` và `sorter.go` sang `correct`.
  - Commit Boundary: Commit sau khi slice P1-A hoàn thành và pass toàn bộ acceptance.

- [x] P1-B: Dual-Trigger Engine & 3-Retry Controller
  - Goal: Nối native Retry/state stream với controller, tối đa ba recovery actions theo outcome; cả hai trigger gọi lại engine qua shared switch gate. Không triển khai app binding trong slice này.
  - Scope Boundary:
    - Editable: `internal/automation/retry.go`, `controller.go`, `switch_gate_windows.go`, `internal/connector/automation_rpc.go`; tests tương ứng ở bước 3.
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
    - Cleanup/quarantine: Cancel stream/context, đóng HTTP server và mutex handles; helper-process tests dùng lock name riêng, không gọi engine thật.
    - External side effects: Production gửi native Retry/call SwitchAccount; test inject fake RPC/executor và isolated locks.
    - N/A notes: None.
  - Work Steps:
    1. Code (Execute leaf atomic tasks in core source files only):
       - Task 1.1: Adapter native empty-items Retry và stream execution outcomes.
         + Action: Thêm RPC method riêng gửi items:[] với config hiện có; thêm Connect stream decoder/cancellation. RetryCoordinator dùng scanner hai giờ lấy IDs, theo dõi status/error/execution và native retry, đợi outcome mới; ba terminal failures của ba recovery actions mới chuyển. Không gọi text sender với prompt rỗng.
         + Inputs: Context, ServerInfo/CSRF discovery, config, conversation IDs từ ScanActiveConversations2h và native stream contract đã khảo sát.
         + Allowed Edit Scope: `internal/connector/automation_rpc.go`, `internal/automation/retry.go`.
         + Outputs: State subscription, native sender và RetryCoordinator không timer gửi cố định; reconnect sau server đổi, cancel khi stop.
         + Verification Condition: Inspect framing/payload và state transitions theo Integration contracts; `go vet ./internal/connector/... ./internal/automation/...`.
         + Task Checkpoint: `wip(P1-B): task 1.1 - implement native in-memory 3-retry via connect-rpc`
       - Task 1.2: Định nghĩa cấu trúc `AutomationController` trong `internal/automation/controller.go` với cờ nguyên tử `enabled`, `isSwitching`, kênh trạng thái `statusChan`, và liên kết tới AccountManager.
         + Action: Khai báo struct và constructor `NewAutomationController`.
         + Inputs: R1–R7 và Integration contracts trong plan R1 này.
         + Allowed Edit Scope: `internal/automation/controller.go`.
         + Outputs: Struct `AutomationController`.
         + Verification Condition: `go vet ./internal/automation/...`
         + Task Checkpoint: `wip(P1-B): task 1.2 - define automation controller lifecycle struct`
       - Task 1.3: Hiện thực hóa EvaluateProactiveTrigger/TriggerSwitch và shared wrapper cho mọi caller.
         + Action: Hợp nhất triggers, PollAccount ứng viên, recheck model/session, deduplicate sự kiện cũ, xử lý chờ/error; tạo named mutex theo user/credential scope và guard in-process ở wrapper gọi engine. Giữ đúng OS-thread ownership acquire/release và defer cleanup. Tái dùng pool/cache, không ticker mới.
         + Inputs: Active model, Quota cache, danh sách ứng viên.
         + Allowed Edit Scope: `internal/automation/controller.go`, `internal/automation/switch_gate_windows.go`.
         + Outputs: EvaluateProactiveTrigger/TriggerSwitch, shared switch wrapper, Start/Stop/SetEnabled và non-blocking status delivery; GUI/CLI binding được giao P1-C.
         + Verification Condition: Trace mọi caller đi qua wrapper đã định nghĩa, không sửa manager; `go vet ./internal/automation/...`.
         + Task Checkpoint: `wip(P1-B): task 1.3 - implement dual-trigger and single-flight gate`
    2. Code Inspection & Lint:
       - Rà soát kỹ lưỡng khóa đồng bộ Mutex và `atomic.Bool` bảo đảm không xảy ra Deadlock hay Data Race.
    3. Update Tests (Code first, test second):
       - Tạo retry/controller/switch_gate_windows tests và bổ sung connector automation_rpc tests: HTTP ACK rồi runtime lỗi, native retry đang chạy, kết thúc thành công, ba recovery failures, proactive supersedes recovery, duplicate/stale updates, cancel/reconnect và 100 concurrent triggers. Hai helper processes tranh named mutex test riêng, không taskkill/WinCred/prompt thật.
    4. Build:
       - Khi triển khai: `go test ./internal/connector/... ./internal/automation/...`; CGO=1/GCC: `go test -race ./internal/connector/... ./internal/automation/...`; CLI compile `go build ./cmd/switcher`.
    5. QA (Mini QA / Runtime Validation):
       - Kiểm tra native-format Connect frames qua adapter production tới controller/fake executor, lifecycle cleanup và cross-process gate. Không dùng HTTP ACK làm kết quả khôi phục hoặc gọi tests là live-provider E2E.
       - Evidence target: Test log ghi nhận retry và controller concurrent test đạt PASS.
    6. Slice Acceptance & Official Milestone Commit:
       - Ghi nhận evidence, cập nhật `actual-status.md`, tạo commit `feat(P1-B): implement dual-trigger engine and 3-retry controller`.
  - Implementation Gate:
    - Slice P1-A đã hoàn tất.
  - Acceptance:
    - Source: RPC stream/retry adapter, retry/controller/shared gate và tests hoàn chỉnh; manager/reloader không đổi.
    - Runtime/UI: N/A.
    - DB/data: N/A.
    - Behavior test: Outcome-driven recovery và mutex process isolation đạt acceptance; các caller UI/CLI chưa được tuyên bố bound trước P1-C.
    - Cleanup/quarantine: Contexts, stream responses và lock handles được đóng trong test/runtime stop.
    - Evidence IDs: `E1-P1B-SRC1`, `E1-P1B-TEST1`, `E1-P1B-CONCUR1`.
    - Actual-status rows refreshed: `internal/automation/retry.go` -> `correct`, `internal/automation/controller.go` -> `correct`.
  - Evidence Targets: `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-evidence.md#e1---p1-evidence`.
  - Actual-status Update: Dòng `retry.go` và `controller.go` chuyển sang `correct`.
  - Commit Boundary: Commit sau khi slice P1-B hoàn thành.

- [ ] P1-C: Automation Coordinator & GUI Toggle
  - Goal: Nối toggle Tự động đã có và status bar với controller, callback telemetry, lifecycle và các caller chuyển tay GUI/CLI.
  - Scope Boundary:
    - Editable: `internal/ui/app.go`, `internal/ui/window.go` chỉ callback, `cmd/switcher/main.go` chỉ --switch; `internal/ui/automation_binding_test.go`, `cmd/switcher/main_test.go` ở bước 3.
    - Inspect-only: `internal/automation/controller.go`
    - Preserve-only: Các layout danh sách tài khoản hiện hữu trên Fyne UI
    - Out of scope: Sửa đổi core backend switcher
  - Non-Goals: Không tạo widget/khối điều khiển mới, không đổi layout/theme, không persist toggle, không thay close-to-exit hoặc core engine.
  - Pre-flight Questions:
    - Data source: Kênh trạng thái `statusChan` từ `AutomationController`.
    - Display permission: Luôn hiển thị cho người dùng.
    - DB read flow: Manager/pool/cache đã nằm trong App; toggle giữ trạng thái trong phiên ứng dụng.
    - DB write flow: Không thêm file cấu hình. Side effects của switch do engine cũ thực hiện.
    - Render location: Toggle header và status bar sẵn có.
    - UI behavior flow: Toggle -> common setter -> controller; callback telemetry giữ UI refresh và cấp results cho automation; GUI/CLI chuyển qua shared gate đồng bộ; busy báo đang xử lý, không chỉ disable bằng status event tới muộn.
    - Docker runtime: N/A (Desktop Windows GUI).
    - Playwright target: N/A.
    - Behavior test: `internal/ui/automation_binding_test.go`, `cmd/switcher/main_test.go` kiểm tra call paths thực với injected dependencies.
    - Cleanup/quarantine: App.Close hủy controller/streams/status listener rồi dừng loop hiện có; temp store và locks của test riêng biệt.
    - External side effects: Cập nhật giao diện Fyne.
    - N/A notes: None.
  - Work Steps:
    1. Code (Execute leaf atomic tasks in core source files only):
       - Task 1.1: Khởi tạo và quản lý controller trong App hiện có.
         + Action: Constructor dùng cùng manager/pool/cache; App.Init start lifecycle, App.Close cancel controller/subscriptions/status listener. Không tạo pool/ticker thứ hai.
         + Inputs: App dependencies và controller P1-B.
         + Allowed Edit Scope: `internal/ui/app.go`.
         + Outputs: Controller được sở hữu và dừng theo App.
         + Verification Condition: `go vet -tags gui ./internal/ui/...`.
         + Task Checkpoint: `wip(P1-C): task 1.1 - bind controller to app lifecycle`
       - Task 1.2: Nối toggle, telemetry callback và status hiện hữu.
         + Action: Callback toggle và SetAutoSwitchEnabled dùng một setter; bật TriggerNow, tắt không cắt engine đang chạy. Mở rộng callback duy nhất của autoLoop, giữ quota UI refresh và cấp results không chặn cho controller; status qua fyne.Do, cập nhật overview sau auto switch.
         + Inputs: autoSwitchToggle, SetAutoSwitchEnabled, autoLoop.SetUpdateCallback, statusBar.
         + Allowed Edit Scope: `internal/ui/app.go`, `internal/ui/window.go` callback tại setupLayout.
         + Outputs: Toggle thực sự điều khiển automation và quota UI vẫn cập nhật; không có vòng callback đệ quy.
         + Verification Condition: Inspect callback/lifecycle và layout diff; `go vet -tags gui ./internal/ui/...`.
         + Task Checkpoint: `wip(P1-C): task 1.2 - wire status event listener to ui label`
       - Task 1.3: Nối cả manual GUI và CLI vào shared switch wrapper.
         + Action: Bọc đường App.SwitchAccount và nhánh --switch; giữ target/flags/result presentation, chặn manual-first/auto-first và CLI cạnh tranh; invalidate recovery khi manual thay session.
         + Inputs: Existing manager callers và gate P1-B.
         + Allowed Edit Scope: `internal/ui/app.go`, `cmd/switcher/main.go`.
         + Outputs: Mọi caller dùng chung guard mà không sửa engine.
         + Verification Condition: Inspect không còn direct manager switch bypass ở các entrypoints; `go vet -tags gui ./internal/ui/... ./cmd/switcher/...`.
         + Task Checkpoint: `wip(P1-C): task 1.3 - bind manual gui and cli to shared switch gate`
    2. Code Inspection & Lint:
       - Rà soát diff giao diện, bảo đảm việc cập nhật giao diện tuân thủ thread-safe của Fyne.
    3. Update Tests (Code first, test second):
       - Binding tests qua App.Init, toggle setter, telemetry callback, App.Close; quota UI vẫn refresh; kiểm tra cả manual-first/auto-first, CLI process contention, tắt lúc recovery/switch và không gọi continuation ngoài manager. Mock executor thay side effects, không viết lại sorter/controller trong test.
    4. Build:
       - CGO=1/GCC trong PATH: `go test -tags gui ./internal/ui/... ./cmd/switcher/... ./internal/automation/... ./internal/connector/... ./internal/telemetry/...`; `go build -tags gui -o bin/antigravity-switch-gui.exe ./cmd/switcher`. Kiểm tra CLI riêng với CGO=0: `go build -o bin/antigravity-switch.exe ./cmd/switcher`.
       - Nếu dùng script build hiện có, vẫn chạy rõ tests automation/telemetry vì script có danh sách package cố định chưa bao gồm tất cả phần mới.
    5. QA (Mini QA / Runtime Validation):
       - Chạy GUI native vừa build để quan sát toggle có sẵn/status/đóng app và quota refresh; runtime có side effect dùng phiên/tài khoản thử được chỉ định. Mock integration tests cách ly chứng minh đường call; không tuyên bố mock là thao tác live provider.
       - Evidence target: Build, binding tests và native GUI observations ghi riêng; screenshots/Markdown/JSON tại `reports/qa/automation/`. Không nhận headless-only làm native GUI acceptance.
    6. Slice Acceptance & Official Milestone Commit:
       - Ghi nhận evidence, cập nhật `actual-status.md`, tạo commit `feat(P1-C): integrate automation toggle and status indicators into fyne ui`.
  - Implementation Gate:
    - Slice P1-B đã hoàn tất.
  - Acceptance:
    - Source: App/window callbacks và CLI binding; không tạo lại toggle hoặc đổi layout.
    - Runtime/UI: Toggle hiện hữu điều khiển controller, callback quota phục vụ UI và automation, Close dừng work mới.
    - DB/data: N/A.
    - Behavior test: Existing toggle/App callback/lifecycle và GUI/CLI caller binding có kết quả đúng; mock tests và native GUI observations được ghi tách biệt.
    - Cleanup/quarantine: App.Close dừng work mới; test resources đóng, không để goroutine/handles thử tồn tại.
    - Evidence IDs: `E1-P1C-SRC1`, `E1-P1C-BUILD1`, `E1-P1C-UI1`, `E1-P1C-CALL1`.
    - Actual-status rows refreshed: toggle/lifecycle/callers từ unbound sang bound chỉ sau acceptance.
  - Evidence Targets: `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-evidence.md#e1---p1-evidence`.
  - Actual-status Update: Cập nhật App/GUI/CLI binding, không đánh dấu P2 đã hoàn tất.
  - Commit Boundary: Commit sau khi slice P1-C hoàn thành.

### P2: System Integration, Stress Testing & Acceptance

- Phase Goal: Nghiệm thu 14 tình huống và các bindings thực qua adapters/callers, rồi cleanup artifact do plan tạo và đóng implementation sau Supervisor.
- Phase Boundary:
  - In scope: automation integration tests, UI/CLI binding tests, native GUI observations, plan-created cleanup và closure.
  - Out of scope: Tạo mới tính năng sản phẩm ngoài phạm vi đặc tả kiến trúc.
  - Dependencies: Hoàn tất toàn bộ Slices P1-A, P1-B, P1-C.
- Phase Implementation Rule: Do not implement P2 directly. Implement P2-A, verify it, record evidence, then proceed with P2-B, P2-C, P2-D.
- Ordered Slice List:
  - P2-A: E2E Integration & 14 Edge Cases Simulation Test Harness
  - P2-B: Call supervisor for the implemented-plan acceptance loop
  - P2-C: Remove dead work and deadcode created during this plan
  - P2-D: Close the plan

- [ ] P2-A: E2E Integration & 14 Edge Cases Simulation Test Harness
  - Goal: Kiểm tra 14 tình huống trong ma trận R1 và các binding thực tế; mock side effects đi qua production adapters/callers, native GUI check riêng.
  - Scope Boundary:
    - Editable: `internal/automation/automation_e2e_test.go`, `internal/ui/automation_binding_test.go`, `cmd/switcher/main_test.go` và fixtures/testdata tương ứng.
    - Inspect-only: `internal/automation/*`, `internal/manager/*`, `internal/telemetry/*`
    - Preserve-only: Core packages
    - Out of scope: Sửa đổi mã nguồn sản phẩm
  - Non-Goals: Scenario tests không phụ thuộc tài khoản Google thật; mock HTTP/executor nhưng đi qua parser/adapters/controller/caller production. Native GUI check riêng không bị thay bằng mock/headless. Không sửa production trong slice này.
  - Pre-flight Questions:
    - Data source: Fixtures mô phỏng 14 tình huống biên.
    - Display permission: N/A.
    - DB read flow: In-memory mock store.
    - DB write flow: In-memory mock switch target.
    - Render location: Test results và cửa sổ Fyne native cho binding/visual check.
    - UI behavior flow: App.Init -> toggle hiện hữu -> telemetry/stream -> controller -> shared wrapper -> status -> App.Close.
    - Docker runtime: N/A.
    - Playwright target: N/A.
    - Behavior test: `internal/automation/automation_e2e_test.go`.
    - Cleanup/quarantine: Dọn dẹp mock resources sau khi chạy test.
    - External side effects: None.
    - N/A notes: None.
  - Work Steps:
    1. Code (slice test-only; không sửa production):
       - Task 1.1: Thiết lập mock HTTP/Connect endpoints phát provider JSON và native-format frames qua adapters/controller thật; inject fake switch executor, không viết lại production logic trong harness.
         + Action: Viết mock harness trong `internal/automation/automation_e2e_test.go`.
         + Inputs: Ma trận 14 tình huống biên.
         + Allowed Edit Scope: `internal/automation/automation_e2e_test.go`.
         + Outputs: Structs `MockLangServer` và `MockTelemetry`.
         + Verification Condition: `go test -run TestMockHarnessInit ./internal/automation/...`
         + Task Checkpoint: `wip(P2-A): task 1.1 - build mock e2e harness for 14 edge cases`
       - Task 1.2: Viết test case mô phỏng lần lượt các biên 1 đến 7 (Gemini 5h <= 2%, Claude hết tuần khi đang chạy Gemini, Tuần 1% vs 0%, disabled bucket, retry thành công lần 1/2 vs thất bại lần 3).
         + Action: Cài đặt bài test cho Biên 1 - 7.
         + Inputs: Ma trận 14 tình huống ở cuối plan R1.
         + Allowed Edit Scope: `internal/automation/automation_e2e_test.go`.
         + Outputs: Hàm `TestEdgeCases1To7`.
         + Verification Condition: `go test -run TestEdgeCases1To7 ./internal/automation/...`
         + Task Checkpoint: `wip(P2-A): task 1.2 - verify edge cases 1 to 7`
       - Task 1.3: Viết các biên 8–14 qua điểm gọi thật (trùng triggers, GUI/CLI manual contention hai chiều, đổi model, ứng viên lỗi, engine fail, không có ứng viên, stream event phiên cũ). Bổ sung binding checks: disabled qua parser/cache, model ID qua config, HTTP ACK rồi execution lỗi, native retry waiting, App.Close cleanup và quota UI callback vẫn chạy.
         + Action: Cài đặt bài test cho Biên 8 - 14.
         + Inputs: Ma trận 14 tình huống ở cuối plan R1.
         + Allowed Edit Scope: `internal/automation/automation_e2e_test.go`, `internal/ui/automation_binding_test.go`, `cmd/switcher/main_test.go`, testdata tương ứng.
         + Outputs: Tests 8–14 và binding assertions tại App/CLI, không chỉ gọi TriggerSwitch hai lần.
         + Verification Condition: `go test -run TestEdgeCases8To14 ./internal/automation/...`
         + Task Checkpoint: `wip(P2-A): task 1.3 - verify edge cases 8 to 14`
    2. Code Inspection & Lint:
       - Rà soát test coverage, đảm bảo không có trường hợp rò rỉ goroutine hay timeout trong mock harness.
    3. Update Tests (Code first, test second):
       - Củng cố toàn bộ suite `automation_e2e_test.go` chạy độc lập, kích hoạt qua `go test -v -race ./internal/automation/...`.
    4. Build:
       - CGO=1/GCC: `go test -tags gui ./...`; `go test -race -tags gui ./internal/automation/... ./internal/connector/... ./internal/telemetry/... ./internal/ui/... ./cmd/switcher/...`; `go build -tags gui -o bin/antigravity-switch-gui.exe ./cmd/switcher`. Build CLI riêng với CGO=0 như P1-C.
    5. QA (Mini QA / Runtime Validation):
       - Chạy scenario/binding suite cô lập, ghi từng kết quả. Chạy final native GUI để quan sát toggle có sẵn, quota refresh/status và đóng app; lưu ảnh, Markdown/JSON tại `reports/qa/automation/`.
       - Evidence target: `E2-P2A-TEST14` cho scenarios, `E2-P2A-BIND1` cho actual caller/data paths, `E2-P2A-GUI1` cho native GUI. Không ghi kết quả thao tác live provider nếu chỉ chạy mocks; test tự động không chạm IDE thật.
    6. Slice Acceptance & Official Milestone Commit:
       - Ghi nhận evidence, cập nhật `actual-status.md`, tạo commit `feat(P2-A): add comprehensive e2e test harness verifying 14 edge cases`.
  - Implementation Gate:
    - Toàn bộ Slices P1-A, P1-B, P1-C đã hoàn tất.
  - Acceptance:
    - Source: `internal/automation/automation_e2e_test.go`.
    - Runtime/UI: Native GUI được build với gui tag và quan sát binding/lifecycle; không chỉ headless.
    - DB/data: Temporary fixtures/cache/locks riêng, cleanup đầy đủ.
    - Behavior test: 14/14 tình huống và binding checks đạt kết quả mong đợi qua adapters/callers production.
    - Cleanup/quarantine: Mock server tự đóng cổng sạch sẽ sau khi test xong.
    - Evidence IDs: `E2-P2A-SRC1`, `E2-P2A-TEST14`, `E2-P2A-BIND1`, `E2-P2A-GUI1`.
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
  - Goal: Dọn artifact trung gian do plan này tạo, bảo toàn evidence hợp lệ; không cleanup engine hoặc Phase 7 cũ.
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
    1. Xác nhận validation P1-C/P2-A áp dụng cho final code; chỉ chạy lại phần ảnh hưởng nếu có thay đổi mới. GUI build/run bắt buộc dùng `-tags gui` và toolchain native; CLI build riêng. Sửa tài liệu R1 không thực hiện bước này.
    2. Ghi nhận evidence cuối cùng, benchmark và commit hash vào `evidence.md` và `actual-status.md`.
    3. Cập nhật Status trong metadata của Plan thành `complete`.
    4. Tạo commit hoàn tất kế hoạch.
  - Implementation Gate: P2-B và P2-C phải hoàn tất đạt yêu cầu.
  - Acceptance: Final evidence được ghi nhận đầy đủ, commit hoàn tất tồn tại trên git tree.

## Risk Notes

1. **Khoảng chờ của engine:** Giữ 8 giây hiện có rồi kiểm tra email. Đây là cấu hình source, không phải số đo tổng switch hay bằng chứng mọi socket đều chưa sẵn sàng trước 8s. Continuation còn có pacing 3s giữa threads.
2. **Native Retry:** Đợi kết quả thực thi, không coi HTTP ACK là phục hồi. Ba recovery actions thất bại hết ngân sách thử của đợt, không chứng minh mọi quota đã cạn. Telemetry xác nhận tuần = 0 hoặc 5h <= 2% thì dùng proactive, không chờ thêm ba lần.
3. **Mất kết nối mạng hoặc lỗi Google API:** Khi telemetry trả về dữ liệu rỗng hoặc timeout (`UNKNOWN_UNCERTAIN`), controller tuyệt đối không được phép suy diễn thành hết quota để chuyển tài khoản mù quáng, tránh làm cạn kiệt tài nguyên của người dùng.
4. **Xung đột callers:** Shared wrapper ở caller boundary bảo vệ GUI/CLI/auto. Cờ atomic trong controller hoặc disable nút qua UI event không tự bảo vệ process CLI. Giữ manager/reloader nguyên vẹn.

## Ma trận 14 tình huống và mục tiêu kiểm tra

| ID | Tình huống | Kết quả mong đợi — chưa phải kết quả đã chạy |
|---|---|---|
| 1 | Gemini đang dùng, 5h <= 2% | Chuyển một lần. |
| 2 | Claude tuần = 0, phiên dùng Gemini còn quota | Không chuyển do Claude. |
| 3 | Chỉ tuần, tuần = 1% | Dùng tiếp. |
| 4 | Chỉ tuần, tuần = 0% | Chuyển. |
| 5 | API bucket 5h disabled | Cờ đi qua parser/cache, bỏ qua 5h và xét tuần. |
| 6 | Recovery 1 hoặc 2 kết thúc không lỗi | Dừng recovery, không switch. |
| 7 | Ba recovery actions kết thúc lỗi giới hạn | Một quyết định switch. |
| 8 | Proactive và reactive cùng kích hoạt | Một đợt; không phát retry tiếp sau quyết định switch. |
| 9 | Manual GUI/CLI và auto cùng yêu cầu | Một executor; kiểm tra manual-first và auto-first. |
| 10 | Model đổi trước switch | Resolve và tính lại nhóm/ứng viên. |
| 11 | Preflight ứng viên đầu lỗi | Bỏ qua trong đợt, xét kế tiếp trước khi thay credential. |
| 12 | SwitchAccount trả lỗi | Dừng đợt, hiện lỗi, không xoay vô điều kiện. |
| 13 | Không có ứng viên | Hiện lý do, chờ telemetry hiện có. |
| 14 | Event cũ/trùng tới muộn | Không tăng retry hoặc switch vì event cũ. |

Ngoài 14 hàng, nghiệm thu phải kiểm tra các bindings đã giao ở P1-C: callback quota không bị ghi đè, toggle/lifecycle thật, dynamic model config, provider-to-cache disabled, HTTP acceptance khác execution outcome, cross-process gate và GUI build tag. Đây là kiểm tra tính năng mới, không mở audit engine cũ.
