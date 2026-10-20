# Autonomous Account Switching Evidence Ledger

## Metadata

- Date: 2026-10-20
- Revision: R1 — documentation correction only
- Plan: docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-plan.md
- Rules: mục Rules và Integration contracts của plan R1
- Evidence: docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-evidence.md
- Benchmark: docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-benchmark.md
- Actual status: docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-actual-status.md
- Source baseline: 2e4c8939037871f7247f6030d03a69a0c90c89ab

## Evidence Rules

- Chỉ OBSERVED khi đã đọc source/command/artifact tương ứng. PLANNED / NOT RUN là mục tiêu cần thu thập, không chứng minh code tồn tại hoặc tests pass.
- Mỗi evidence thực thi sau này phải có command, thời điểm/HEAD, kết quả, artifact và phạm vi được chứng minh. Mock khác native GUI khác live provider.
- Không ghi token, CSRF, credential hoặc nội dung chat vào report.
- Benchmark chỉ ghi số đo đã đo. Hằng số source 60s/8s/2h/3s được ghi như cấu hình bảo lưu, không phải benchmark.
- Giữ format ID E<phase>-<item>-<kind><n>. E0 = khảo sát/tài liệu; E1/E2 = implementation tương lai. Không dùng review markdown để nghiệm thu sản phẩm.
- Những câu ở draft cũ về “đã tạo file”, “100% PASS”, “Supervisor đã nghiệm thu” trong E1/E2 được R1 sửa thành targets chưa thực hiện.

## E0 - P0 Evidence

Matching plan item: P0-A.

### E0-P0A-SPEC1 — Tài liệu đầu vào lịch sử

Status: OBSERVED DOCUMENT, không phải bằng chứng runtime hoặc lệnh bắt buộc sửa SPEC.

Đã đọc báo cáo reports/system-architect/rp_system-architect_261020_141500_by_gemini-2-5-pro_autonomous_account_switching_architecture_specification.md. Giữ chính sách model quota/ngưỡng/thứ tự/continuation theo Owner. Các mô tả retry cố định/tốc độ, UI missing và phạm vi cấm DTO được R1 sửa trong plan sau review. Không sửa báo cáo gốc hoặc xóa Phase 7 cũ.

### E0-P0A-SRC1 — Manager engine

Status: OBSERVED SOURCE.

internal/manager/account_manager.go:378 SwitchAccount; :465 ToS, :471 WinCred, :488 chờ 8s, :543 GetUserStatus, :594 continuation. R1 gọi nguyên hàm qua shared caller gate. Không có yêu cầu fix hoặc refactor engine trong plan này.

### E0-P0A-SRC2 — Telemetry pipeline

Status: OBSERVED SOURCE.

- internal/telemetry/client.go:225 ModelQuotaBucket; :313 AccountQuotaData; :672 rawQuotaBucket; :731 normalized RawBuckets chưa có disabled.
- internal/telemetry/cache.go:70 Get và JSON persistence dùng parsed AccountQuotaData; RawBuckets không phải JSON response gốc.
- internal/telemetry/loop.go:13 interval 60s; :72 callback setter thay callback; :137–143 phát results; TriggerNow đã có.
- internal/telemetry/pool.go:293 PollAccount hiện có.
- Decision: giữ fetch/cache/loop; chỉ thêm metadata field propagation và App callback binding.

### E0-P0A-SRC3 — Reloader/continuation

Status: OBSERVED SOURCE.

internal/reloader/active_threads.go:58 ScanActiveConversations2h, source conversation IDs có sẵn. internal/reloader/continuation.go:29 và :95 có continuation/pacing 3s. R1 dùng scanner read-only cho subscription và giữ nguyên continuation được manager gọi.

### E0-P0A-SRC4 — Connector và model/native payload

Status: OBSERVED SOURCE.

internal/connector/api_client.go:307 DiscoverActiveAgentModel trả ID từ trạng thái/API, :419–424 text sender luôn tạo một item; discovery.go:312 cung cấp ServerInfo. Adapter mới phải join ID với config label và gửi native items rỗng bằng method riêng. Không thay behavior resolver/text sender.

### E0-P0A-SRC5 — Toggle/App thực tế

Status: OBSERVED SOURCE.

internal/ui/window.go:176–187 đã có label Tự động và AutoSwitchToggle. internal/ui/app.go:650–666 có getter/setter; :79 tạo dependencies; :234 gán callback autoLoop; :257 Close dừng loop. Callback toggle hiện chỉ cập nhật bool. Decision: unbound cần nối, không missing widget.

### E0-P0A-SRC6 — Manual callers và khóa

Status: OBSERVED SOURCE.

internal/ui/app.go:506–530 manual GUI gọi manager với isBusy riêng; cmd/switcher/main.go:96 --switch gọi manager trực tiếp; cmd/switcher/app_gui.go:15 chỉ giữ single-instance GUI. Decision: giao shared gate ngoài engine và binding cả hai entrypoints, không tuyên bố atomic.Bool bảo vệ cross-process.

### E0-P0A-SRC7 — Native build

Status: OBSERVED SOURCE, BUILD NOT RUN.

cmd/switcher/app_gui.go dùng gui tag, app_cli.go dùng !gui. scripts/build_windows.ps1 có cấu hình CGO/GCC và build -tags gui; danh sách test cũ cố định. P1-C/P2-A cần explicit automation/telemetry tests và native GUI build/run. Không chạy các lệnh đó khi sửa markdown.

### E0-P0A-DIR1 — Inventory

Status: OBSERVED FILESYSTEM.

Tại baseline chưa có internal/automation/, internal/connector/automation_rpc.go hoặc automation binding tests. Không có internal/ui/main_window.go hoặc internal/telemetry/quota_models.go. Đường dẫn thật là ui/window.go và telemetry/client.go. GUI toggle đã tồn tại như SRC5.

### E0-P0A-RPC1 — Native mechanism read-only đã khảo sát

Status: OBSERVED READ-ONLY trong cuộc thảo luận và review, không thực hiện recovery/switch.

- Đã đọc model/config/quota và StreamAgentStateUpdates của Antigravity đang chạy trong lượt khảo sát trước. M318 được config trả về là Gemini 3.8 Flash (High); quota summary chia Gemini và Claude/GPT, có bucket disabled.
- Review đọc lại /main.js của Language Server: V2b -> H4 với items:[], media:[], allowEmpty:true; l1b dùng IDLE + ERROR + lastStepError; QEb hiển thị modelRequest attempt/maxAttempts/retryAt.
- Chỉ chứng minh nguồn và cách giao tiếp có sẵn. Không có phép đo recovery dưới 500ms hoặc kết quả gửi Retry thật.
- Source/command scope được lưu tại báo cáo review đầu vào E0-P0A-REV1; không cần yêu cầu Owner chứng minh lại cơ chế đã khảo sát.

### E0-P0A-REV1 — Review sửa kế hoạch

Status: OBSERVED REPORT.

reports/supervisor/rp_supervisor_261020_144149_by_gpt-6_autonomous_plan_review.md: REJECT readiness của draft, F1–F8. Owner đã yêu cầu sửa bộ plan, không yêu cầu triển khai sản phẩm.

| Finding | R1 sửa tại |
|---|---|
| F1 | Scope metadata-only, R2/Integration contracts và P1-A task 1.2 |
| F2 | Integration contracts App lifecycle, P1-B stream/lifecycle và P1-C tasks 1.1–1.2 |
| F3 | Actual-status unbound toggle, P1-C bind control hiện có |
| F4 | Shared gate P1-B task 1.3; GUI/CLI P1-C task 1.3 |
| F5 | R4/outcome contract, adapter items:[], P1-B task 1.1 và execution-result tests |
| F6 | R1, model config adapter P1-A task 1.2 |
| F7 | E1/E2 PLANNED / NOT RUN; benchmark tách constants/measurements |
| F8 | P1-C/P2-A native gui tag build/run và actual caller binding checks |

### E0-P0A-DEC1 — Baseline decision

Status: DOCUMENT DECISION.

P0 đã khảo sát và sửa hướng thi công R1. Slice code đầu tiên là P1-A; mọi slice sản phẩm còn chưa bắt đầu. Quyết định này không tuyên bố modules/tests đã tồn tại.

### E0-P0A-DOC1 — Kiểm tra tài liệu R1

Status: OBSERVED DOCUMENT CHECK — 2026-10-20, chỉ kiểm tra markdown.

Đã đọc lại nội dung bốn file, đối chiếu policy/scope/tasks, evidence IDs và source-path references. Kiểm tra whitespace bằng git diff --check giới hạn thư mục kế hoạch, không có lỗi. Chỉ bốn file plan được sửa bởi lượt này; thay đổi .gitignore xuất hiện đồng thời từ ngoài lượt được giữ nguyên. Không chạy test/build sản phẩm. Báo cáo rà lại tài liệu: reports/supervisor/rp_supervisor_261020_151400_by_gpt-6_autonomous_plan_r1.md.

## E1 - P1 Evidence

Matching items: P1-A, P1-B, P1-C. Tất cả các hàng sau đều là mục tiêu, chưa có artifact thực thi.

| ID | Status | Evidence cần thu thập khi triển khai |
|---|---|---|
| E1-P1A-SRC1 | PLANNED / NOT RUN | Diff metadata pass-through, model config adapter, classifier/sorter. |
| E1-P1A-TEST1 | PLANNED / NOT RUN | Provider JSON -> production parser -> cache -> evaluation; dynamic model ID; error/weekly-only; bảng A-F. |
| E1-P1A-BENCH1 | PLANNED / NOT RUN | Timing sorter nếu thực sự đo, không có SLA tự đặt. |
| E1-P1B-SRC1 | PLANNED / NOT RUN | Native RPC/stream adapter, retry/controller/shared gate diff. |
| E1-P1B-TEST1 | PLANNED / NOT RUN | items:[]; HTTP ACK rồi runtime lỗi; terminal success; ba failed actions; native retry waiting/cancel/stale events. |
| E1-P1B-CONCUR1 | PLANNED / NOT RUN | Concurrent triggers và cross-process named-mutex test isolated; fake executor count. |
| E1-P1C-SRC1 | PLANNED / NOT RUN | Existing toggle/lifecycle/callback/GUI-CLI caller binding diff. |
| E1-P1C-BUILD1 | PLANNED / NOT RUN | Native GUI với gui tag/CGO và CLI build; không thay GUI build bằng default package build. |
| E1-P1C-UI1 | PLANNED / NOT RUN | Existing toggle thực, status, quota refresh và close lifecycle; native screenshots/observations. |
| E1-P1C-CALL1 | PLANNED / NOT RUN | Actual App/CLI call paths: manual-first, auto-first, process contention, no duplicate continuation. |

## E2 - P2 Evidence

Matching items: P2-A, P2-B, P2-C, P2-D.

| ID | Status | Evidence cần thu thập khi triển khai |
|---|---|---|
| E2-P2A-SRC1 | PLANNED / NOT RUN | Integration harness qua real adapters/callers, fake external side effects. |
| E2-P2A-TEST14 | PLANNED / NOT RUN | Kết quả từng hàng ma trận 14 tình huống trong plan; ghi cả thất bại nếu có. |
| E2-P2A-BIND1 | PLANNED / NOT RUN | Callback coexistence, disabled round-trip, dynamic model, outcome retry, lifecycle và shared callers. |
| E2-P2A-GUI1 | PLANNED / NOT RUN | Final native GUI build/run, screenshots và kết quả tách khỏi mocks. |
| E2-P2B-SUP1 | PLANNED / NOT RUN | Supervisor nghiệm thu code/runtime sản phẩm; không dùng verdict tài liệu thay thế. |
| E2-P2C-CLEAN1 | PLANNED / NOT RUN | Inventory/cleanup artifact do plan tạo, giữ evidence hợp lệ. |
| E2-P2D-BUILD1 | PLANNED / NOT RUN | Final validation áp dụng đúng code final, GUI tag và CLI target. |
| E2-P2D-COMMIT1 | PLANNED / NOT RUN | Commit đóng implementation sau các gates. |

Ma trận scenario và acceptance nằm trong plan; bảng này không lặp lại chúng như kết quả PASS. Báo cáo/screenshot runtime mới lưu tại reports/qa/automation/ và được tham chiếu bằng ID tương ứng.

## Closure Evidence

- Baseline commit chứa draft: 2e4c8939037871f7247f6030d03a69a0c90c89ab.
- R1: sửa tài liệu theo yêu cầu Owner; chưa triển khai P1/P2.
- Product build/tests/native switch/retry: NOT RUN trong lượt sửa tài liệu.
- Implementation final commit: NOT CREATED.
