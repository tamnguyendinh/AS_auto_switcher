# Autonomous Account Switching Actual Status

Title: Autonomous Account Switching
Date: 2026-10-20
Status: P0 Complete — R1 corrected; implementation not started
Companion plan: docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-plan.md
Companion rules: mục Rules và Integration contracts trong companion plan R1
Companion evidence: docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-evidence.md
Companion benchmark: docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-benchmark.md
Source baseline: 2e4c8939037871f7247f6030d03a69a0c90c89ab

## Purpose

Ghi đúng phần có sẵn để tái dùng và phần cần nối cho automation. R1 sửa kết luận P0 cũ theo review F1–F8; không audit hoặc sửa engine. Các module sản phẩm mới vẫn chưa tồn tại. Tài liệu được sửa không phải evidence triển khai.

## Freshness / Refresh Rules

- Sau mỗi slice đã nghiệm thu, cập nhật đúng hàng source/binding thay đổi và thêm Status Refresh Log.
- Trước slice kế tiếp, đối chiếu source hiện tại; không tạo lại chức năng đã được lane khác triển khai.
- Phần preserve-only được giữ nguyên; ngoại lệ hẹp phải nằm trong touch map của plan.
- Evidence chứa nguồn/command/kết quả; bảng này chứa phân loại và hành động tiếp theo. Không ghi trước PASS.
- Relationship counts bên dưới là số đường dẫn trực tiếp được liệt kê trong bảng, không phải số đo graph hoặc toàn bộ blast radius.

## Scope

Target: automation adapters/controller/gate; metadata quota pass-through; binding toggle/callback/lifecycle App; bọc caller GUI/CLI.

Preserve: AccountManager.SwitchAccount, reloader và continuation hai giờ, text message sender/model resolver hiện có, telemetry fetch/cache/loop, GUI styling/theme/close behavior, store/scanner/WinCred.

Không tạo automation_view.go hoặc toggle thứ hai. Không thêm persistence của toggle hay sửa Phase 7 cũ.

## Relationship / Impact Evidence

| Surface | Direct relationships inspected/planned | Count | Evidence | Impact |
|---|---|---:|---|---|
| telemetry/client.go metadata | telemetry/pool.go; telemetry/cache.go; ui/quota_badge.go; automation/model_quota.go (planned consumer) | 4 | E0-P0A-SRC2 | Chỉ thêm disabled vào dữ liệu; không đổi số liệu UI/fetch. |
| connector/automation_rpc.go (planned) | connector/api_client.go; connector/discovery.go; automation/model_quota.go; automation/retry.go | 4 | E0-P0A-SRC4 | Tái dùng client/discovery, method text giữ nguyên. |
| automation/controller.go (planned) | automation/model_quota.go; sorter.go; retry.go; switch_gate_windows.go; manager/account_manager.go; telemetry/pool.go; ui/app.go | 7 | E0-P0A-SRC1, E0-P0A-SRC2, E0-P0A-SRC5 | Controller nhận dữ liệu, không sở hữu engine mới. |
| automation/switch_gate_windows.go (planned) | manager/account_manager.go; automation/controller.go; ui/app.go; cmd/switcher/main.go | 4 | E0-P0A-SRC1, E0-P0A-SRC6 | Shared caller boundary, không sửa engine. |
| ui/app.go binding | ui/window.go; telemetry/loop.go; telemetry/pool.go; automation/controller.go; automation/switch_gate_windows.go | 5 | E0-P0A-SRC5, E0-P0A-SRC6 | Một callback/toggle, bảo toàn UI refresh. |

## Status Rules

| Status | Meaning | Next action |
|---|---|---|
| correct / bound-correct | Chức năng đang có trong source thuộc scope được bảo lưu | Reuse; không mở audit engine. |
| partial | Pipeline hiện có thiếu metadata cần cho automation mới | Bổ sung đúng dữ liệu được giao. |
| unbound | UI/caller đã có nhưng chưa nối controller/gate | Bind đúng điểm gọi hiện hữu. |
| missing | File/chức năng automation mới chưa có | Chỉ triển khai slice sở hữu. |

## Current Status Matrix

| Unit | Current state | Required state | Status | Evidence | Next decision |
|---|---|---|---|---|---|
| manager/account_manager.go | SwitchAccount có flow đổi phiên, wait 8s, xác nhận email, continuation | Gọi nguyên engine qua wrapper caller | correct / preserve-only | E0-P0A-SRC1 | Không sửa |
| reloader/ | Scanner hai giờ, snapshot, continuation pacing 3s | Dùng scanner IDs và giữ continuation bên trong manager | correct / preserve-only | E0-P0A-SRC3 | Không sửa |
| telemetry/client.go | rawQuotaBucket và ModelQuotaBucket chưa có disabled; RawBuckets là dữ liệu parsed | Truyền disabled từ API qua RawBuckets/cache tới classifier | partial for new automation metadata | E0-P0A-SRC2 | Edit hẹp P1-A |
| telemetry/loop.go, pool.go, cache.go | Có 60s loop, một callback, PollAccount, TriggerNow và cache | Reuse nguyên cơ chế; App mở rộng callback | correct / preserve-only | E0-P0A-SRC2 | Binding ở P1-C, không viết loop mới |
| connector/api_client.go | DiscoverActiveAgentModel trả ID; text sender luôn tạo một item | Reuse model resolver/text sender; method mới bổ sung trong file riêng | correct / preserve-only | E0-P0A-SRC4 | Không sửa behavior hiện có |
| connector/automation_rpc.go | Đã triển khai model config reader (P1-A), empty-items retry sender và Connect-RPC stream frame adapter (P1-B) | Tái dùng cho coordinator P1-C | correct | E1-P1A-SRC1, E1-P1B-SRC1 | Hoàn tất P1-B |
| automation/model_quota.go, sorter.go | Đã triển khai model quota mapping, ternary classifier, 5 tiêu chí lọc, 4 tầng Burn-down sorter | Tái dùng cho controller P1-B và coordinator P1-C | correct | E1-P1A-SRC1, E1-P1A-TEST1, E1-P1A-BENCH1 | Hoàn tất P1-A; bảo lưu cho P1-B |
| automation/retry.go, controller.go, switch_gate_windows.go | Đã triển khai Single-Flight Gate, RetryCoordinator (tối đa 3 lần theo outcome), AutomationController (dual-trigger + preflight fallback), Windows Named Mutex caller boundary | Tái dùng cho coordinator và binding P1-C | correct | E1-P1B-SRC1, E1-P1B-TEST1, E1-P1B-CONCUR1 | Hoàn tất P1-B; chuyển sang P1-C |
| ui/window.go AutoSwitchToggle | Đã có toggle và label Tự động; callback gọi SetAutoSwitchEnabled | Giữ widget và nối common setter/controller | correct | E1-P1C-SRC1, E1-P1C-UI1 | Hoàn tất P1-C |
| ui/app.go | Có Set/IsAutoSwitchEnabled, App.Init quota callback, App.Close, AutomationController lifecycle | Khởi tạo controller bằng dependencies sẵn có, mở rộng callback, cancel lifecycle | correct | E1-P1C-SRC1, E1-P1C-UI1 | Hoàn tất P1-C |
| GUI/CLI switch callers | App.SwitchAccount và --switch bọc qua ExecuteSwitchWithGate | Cùng wrapper mutex ngoài manager, bảo vệ in-process và cross-process | correct | E1-P1C-SRC1, E1-P1C-CALL1 | Hoàn tất P1-C |
| Automation scenario/binding tests | Chưa có | Test provider/caller paths, fake side effects và native GUI check | missing | E0-P0A-DIR1, E0-P0A-SRC7 | Theo slice/P2-A |

## Status Refresh Log

| Refresh | Date | Repo basis | Change | Evidence | Next phase |
|---|---|---|---|---|---|
| R0 | 2026-10-20 | Bản draft trước review | Đã ghi automation missing nhưng nhầm toggle missing và thiếu metadata/binding surfaces. Kết luận proceed unchanged của R0 bị R1 thay thế. | E0-P0A-SPEC1 | Không thực hiện nguyên draft |
| R1 | 2026-10-20 | HEAD 2e4c893, review F1–F8 và yêu cầu sửa plan | Toggle -> unbound; quota metadata -> partial; thêm RPC/gate/App/CLI paths; giữ engine; E1/E2 -> planned/not run | E0-P0A-REV1, E0-P0A-SRC1..7, E0-P0A-DOC1 | Slice đầu tiên là P1-A; chưa bắt đầu |
| R2 | 2026-10-20 | Commit c01b16f, nghiệm thu 9633470 | Hoàn thành Slice P1-A: Model Quota Resolver & Burn-down Sorter. model_quota.go, sorter.go chuyển sang correct. | E1-P1A-SRC1, E1-P1A-TEST1, E1-P1A-BENCH1 | Chuyển sang Slice P1-B |
| R3 | 2026-10-20 | Commit 2c837b1, nghiệm thu e0ac3b2 | Hoàn thành Slice P1-B: Dual-Trigger Engine & 3-Retry Controller. retry.go, controller.go, switch_gate_windows.go và connector/automation_rpc.go chuyển sang correct. | E1-P1B-SRC1, E1-P1B-TEST1, E1-P1B-CONCUR1 | Chuyển sang Slice P1-C |
| R4 | 2026-10-20 | Commit 5aa9d92, nghiệm thu d8803e0 | Hoàn thành Slice P1-C: Automation Coordinator & GUI Toggle. ui/app.go, ui/window.go và cmd/switcher/main.go chuyển sang correct. Kết thúc toàn bộ Phase 1. | E1-P1C-SRC1, E1-P1C-BUILD1, E1-P1C-UI1, E1-P1C-CALL1 | Chuyển sang Phase 2 (P2-A) |

## Phase Touch Map

| Slice | Editable source | Preserve / inspect | Constraint |
|---|---|---|---|
| P1-A | telemetry/client.go; connector/automation_rpc.go; automation/model_quota.go; automation/sorter.go | api_client.go/discovery.go; pool/cache/loop; manager/reloader/UI | Metadata-only trong telemetry, dynamic config join, không hardcode IDs. |
| P1-B | connector/automation_rpc.go; automation/retry.go; controller.go; switch_gate_windows.go | connector cơ chế có sẵn; reloader scanner; manager | Empty items, execution outcomes, shared gate ngoài engine. |
| P1-C | ui/app.go; ui/window.go callbacks; cmd/switcher/main.go --switch | Layout/widget renderer/theme/preferences/tray/engine | Một toggle, một telemetry callback, shared callers. |
| P2-A | automation_e2e_test.go; ui/automation_binding_test.go; cmd/switcher/main_test.go; testdata/reports | Production code | Không fix production trong slice test. |
| P2-B/C/D | Owned report/evidence/cleanup/closure | Tài liệu và source ngoài scope | Không xóa Phase 7 cũ; không đóng plan vì sửa xong markdown. |

Paths trong bảng source tính tương đối từ internal/ trừ cmd/switcher; xem Scope trong plan cho đường dẫn đầy đủ. Tests cạnh source chỉ sửa ở bước 3 của từng implementation slice.

## Detailed Findings

### Quota metadata

Source: internal/telemetry/client.go:225, :313, :672, :731. Cờ disabled của provider bị bỏ khỏi DTO hiện tại; RawBuckets không chứa raw JSON. R1 giao rõ pass-through vào normalized bucket/cache. Không tạo một fetch pipeline khác hoặc chỉnh fractions UI. Evidence E0-P0A-SRC2.

### Model và native Retry

Source: internal/connector/api_client.go:307 trả enum/ID; :419 tạo items một phần tử. Nối config API để tìm model family, thêm method gửi items rỗng riêng, không sửa sender continuation. Stream/Retry native đã khảo sát trong cùng cuộc thảo luận; protocol read evidence ở E0-P0A-RPC1, không có kết quả thực hiện recovery thật. R1 đợi execution result thay vì HTTP ACK/timer. Evidence E0-P0A-SRC4.

### Toggle và vòng đời App

Source: internal/ui/window.go:176 và internal/ui/app.go:650 đã có toggle/state accessors. App.Init:234 gán callback duy nhất; loop.SetUpdateCallback thay callback cũ. R1 gắn controller vào constructor/Init/Close và cùng callback, giữ UI refresh. Không có nhu cầu tạo UI khác. Evidence E0-P0A-SRC5.

### Caller gate

Source: internal/ui/app.go:506 và cmd/switcher/main.go:96 gọi trực tiếp manager. Single-instance chỉ ở GUI launch. R1 giao wrapper/mutex theo credential scope và binding cả GUI/CLI, giữ manager/reloader nguyên vẹn. Evidence E0-P0A-SRC6.

### Build và evidence

GUI entrypoint dùng build tag gui. Script build có danh sách test cố định; các package mới phải được gọi rõ. Chưa chạy build/test sản phẩm ở lượt sửa markdown. Evidence E0-P0A-SRC7; mọi kết quả P1/P2 còn chờ thực thi.

## Next Phase Status Decisions

| Plan item | Finding | Decision |
|---|---|---|
| P1-A | Metadata disabled chưa truyền, resolver ID đã có | Bổ sung metadata và config adapter, reuse resolver; triển khai sorter. |
| P1-B | Native methods và controller/gate chưa có | Viết adapter/controller theo native contract, outcome-driven retry. |
| P1-C | Toggle có sẵn, app/callers chưa nối | Binding hiện hữu, không tạo UI trùng. |
| P2-A | Chưa có tests automation/binding | Tạo tests khi tới slice; chưa ghi PASS. |

## Implementation Gate

- [x] Đã đọc và đối chiếu source đúng HEAD baseline.
- [x] Đã xác định các phần đúng để bảo lưu và những phần cần nối.
- [x] Touch map nêu ngoại lệ metadata và các điểm gọi thực tế.
- [x] R1 xử lý hướng sửa F1–F8 trong tasks, requirements, evidence và benchmark.
- [x] P1/P2 còn chưa thực hiện; không có claim build/test/runtime acceptance sản phẩm.
- [x] Khi thực hiện P1-A, cập nhật trạng thái/ledgers sau acceptance của slice.
- [ ] Các slice tiếp theo chỉ bắt đầu theo gate đã ghi trong plan.

## Final P0 Decision

- [x] P0 complete; bản R1 đã cập nhật bước thực thi theo source hiện tại.
- [ ] Implementation complete.

Next action: triển khai P1-A khi bắt đầu công việc code theo kế hoạch đã sửa. Lượt hiện tại chỉ sửa tài liệu, không tự bắt đầu P1-A.
