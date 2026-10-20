# Autonomous Account Switching Actual Status

Title: Autonomous Account Switching
Date: 2026-10-20
Status: P0 Complete
Companion plan: `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-plan.md`
Companion rules: `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-plan.md`
Companion evidence: `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-evidence.md`
Companion benchmark: `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-benchmark.md`

## Purpose

This file records the real current state before implementation.

Implementation must not start until the target scope has a completed status row, evidence IDs, and a downstream plan decision.

This file does not replace `evidence.md`. It classifies current state from evidence.

Use exact evidence IDs from `evidence.md`, such as `E0-P0A-SPEC1`, `E0-P0A-SRC1`, not broad section IDs such as `E0` or `E1`.

## Freshness / Refresh Rules

This actual-status file is a living current-state record, not a one-time P0 snapshot.

P0 records the baseline before implementation. After implementation begins, keep the Current Status Matrix updated so the next agent can trust it as the latest repo reality.

Update this file:

- after each completed implementation slice;
- before starting the next phase if repo state changed;
- whenever evidence changes a current-state classification;
- whenever the next phase's status assumptions, next action, or work steps need updating because reality differs from the previous status.

When refreshing status:

- update only the rows affected by the completed work or new evidence;
- use explicit transitions such as `missing -> correct`, `partial -> correct`, `fake-or-stub -> removed`, or `unbound -> bound-correct`;
- append a Status Refresh Log row instead of deleting history;
- keep detailed proof in `evidence.md`; store only classifications, evidence IDs, touch mode, and plan consequences here.

## Scope

Target scope:

- Thiết lập package mới `internal/automation/` bao gồm:
  + `internal/automation/model_quota.go`: Bộ phân giải model hoạt động & Tam phân thái hạn ngạch.
  + `internal/automation/sorter.go`: Bộ lọc sơ loại 5 tiêu chí & Thuật toán Burn-down Sorter 4 tầng ưu tiên.
  + `internal/automation/retry.go`: Điều khiển Native In-Memory 3-Retry qua Connect-RPC với empty content list.
  + `internal/automation/controller.go`: Bộ điều phối kích hoạt kép (Proactive 60s + Reactive 429) & Single-Flight Concurrency Gate.
- Mở rộng giao diện Fyne GUI tại `internal/ui/`:
  + `internal/ui/automation_view.go`: Toggle switch Bật/Tắt và nhãn hiển thị trạng thái Automation.
- Bộ kiểm thử tự động toàn diện:
  + `internal/automation/*_test.go`: Unit tests và E2E test harness mô phỏng 14 tình huống biên.

Out of scope:

- `internal/manager/account_manager.go`: Hàm `SwitchAccount()` đã hoàn thiện và ổn định; giữ nguyên và tái sử dụng, không sửa đổi core hoán đổi hotswap.
- `internal/reloader/`: Quét SQLite và khôi phục thread 2h (`continuation.go`) đã tích hợp hoàn hảo trong `SwitchAccount`; không gọi lại lần thứ hai.
- `internal/telemetry/`: Client polling Google API 60s và cache; giữ nguyên và chỉ đọc dữ liệu.

## Relationship / Impact Evidence

| Unit / File / Surface | File Detail Evidence | Related File Count | Relationship Summary | Impact Note |
|-----------------------|----------------------|--------------------|----------------------|-------------|
| `internal/automation/model_quota.go` | `E0-P0A-SPEC1` | 3 | Phụ thuộc vào `internal/telemetry` (models, cache); là nền tảng cho `sorter.go` và `controller.go`. | Thấp; file mới độc lập |
| `internal/automation/sorter.go` | `E0-P0A-SPEC1` | 4 | Đọc `model_quota.go`, danh sách tài khoản từ `internal/manager`; cung cấp ứng viên cho `controller.go`. | Thấp; file mới độc lập |
| `internal/automation/retry.go` | `E0-P0A-SPEC1` | 3 | Lắng nghe stream lỗi từ Language Server, gửi RPC `SendUserCascadeMessage`, báo tín hiệu về `controller.go`. | Trung bình; giao tiếp RPC nội bộ |
| `internal/automation/controller.go` | `E0-P0A-SPEC1` | 6 | Trung tâm điều phối: kết nối telemetry, error stream, retry, sorter, và gọi `accountManager.SwitchAccount()`. | Cao; điều phối vòng đời tự động |
| `internal/ui/automation_view.go` | `E0-P0A-SPEC1` | 3 | Nhúng widget Toggle và Label vào cửa sổ chính `internal/ui/`, kết nối với `controller.go`. | Thấp; bổ sung UI không phá vỡ layout |
| `internal/manager/account_manager.go` | `E0-P0A-SRC1` | 8 | Core switch account (hotswap 4 bước, taskkill, delay 8s, continuation 2h). | Bảo lưu tuyệt đối (Preserve-only) |
| `internal/telemetry/` | `E0-P0A-SRC2` | 5 | Polling 60s và lưu cache Quota Snapshot từ Google API. | Chỉ đọc (Inspect-only) |
| `internal/reloader/` | `E0-P0A-SRC3` | 4 | Snapshot và phục hồi các cuộc hội thoại trong vòng 2h. | Bảo lưu tuyệt đối (Preserve-only) |

## Status Rules

| Status | Meaning | Allowed next action |
|--------|---------|---------------------|
| `correct` | Already behaves as required. | Preserve. Add evidence or tests only if needed. |
| `partial` | Some required behavior exists, but gaps remain. | Change only the missing parts. Preserve correct parts. |
| `wrong` | Current behavior, source, or contract is incorrect. | Replace with required behavior. Record the exact reason. |
| `missing` | Required behavior, source, or contract does not exist. | Implement the missing piece only. |
| `unbound` | Surface exists but is not wired to the real source, flow, or contract. | Bind to the real source only. Preserve approved surface. |
| `fake-or-stub` | Prototype, demo, mock, fallback, or placeholder data is being used as real behavior. | Remove fake behavior or replace it with an approved truthful state. |
| `blocked` | Source, authority, contract, or required evidence is unclear. | Stop. Do not implement until resolved. |

## Current Status Matrix

| Unit | Current State | Required State | Status | Relationship Count | Evidence | Next Plan Decision |
|------|---------------|----------------|--------|--------------------|----------|--------------------|
| `internal/automation/model_quota.go` | Chưa tồn tại mã nguồn package `internal/automation`. | File mã nguồn triển khai nhận diện model, tam phân thái hạn ngạch, xử lý cờ `disabled: true`. | `missing` | 3 related files | `E0-P0A-SPEC1` | edit P1-A |
| `internal/automation/sorter.go` | Chưa tồn tại mã nguồn bộ lọc và thuật toán sắp xếp. | File mã nguồn triển khai 5 tiêu chí lọc sơ loại và thuật toán Burn-down Sorter 4 tầng ưu tiên. | `missing` | 4 related files | `E0-P0A-SPEC1` | edit P1-A |
| `internal/automation/retry.go` | Chưa tồn tại mã nguồn Native In-Memory 3-Retry. | File mã nguồn bắt lỗi 429/ResourceExhausted và thực hiện retry Connect-RPC nội dung rỗng tối đa 3 lần. | `missing` | 3 related files | `E0-P0A-SPEC1` | edit P1-B |
| `internal/automation/controller.go` | Chưa tồn tại bộ điều phối kích hoạt kép và single-flight. | File mã nguồn quản lý vòng đời automation, van chủ động 60s, van bị động 429, Single-Flight Gate. | `missing` | 6 related files | `E0-P0A-SPEC1` | edit P1-B |
| `internal/ui/automation_view.go` | Giao diện Fyne hiện tại chỉ hỗ trợ chuyển đổi thủ công. | Bổ sung widget Toggle công tắc Bật/Tắt và nhãn hiển thị trạng thái động của Automation. | `missing` | 3 related files | `E0-P0A-SPEC1` | edit P1-C |
| `internal/manager/account_manager.go` | Đã có hàm `SwitchAccount()` hoàn chỉnh thực hiện hotswap an toàn với 8s cooldown. | Giữ nguyên vẹn, tái sử dụng như đích gọi duy nhất của automation khi cần chuyển tài khoản. | `correct` | 8 related files | `E0-P0A-SRC1` | preserve-only |
| `internal/telemetry/` | Đã có cơ chế polling định kỳ 60s và cache quota từ Google API. | Giữ nguyên vẹn, cung cấp Quota Snapshot cho van chủ động của automation. | `correct` | 5 related files | `E0-P0A-SRC2` | inspect-only |
| `internal/reloader/` | Đã có module snapshot SQLite và tiếp tục các phiên hội thoại trong 2h. | Giữ nguyên vẹn, tự động được kích hoạt bởi `SwitchAccount()`. | `correct` | 4 related files | `E0-P0A-SRC3` | preserve-only |

## Status Refresh Log

| Refresh | Date | Repo Basis | Changed Scope | Status Changes | Evidence | Next Phase Update |
|---------|------|------------|----------------|----------------|----------|-------------------|
| R0 | 2026-10-20 | baseline before P0 | Toàn bộ phạm vi Autonomous Account Switching | Initial baseline classification: automation modules marked `missing`, existing infrastructure marked `correct`. | `E0-P0A-SPEC1`, `E0-P0A-SRC1..3` | Proceed with P1-A implementation |

## Phase Touch Map

| Unit / File / Surface | Plan-Relevant Relationship File | Relationship to Target | Plan Item | Touch Mode | Evidence | Constraint |
|-----------------------|---------------------------------|------------------------|-----------|------------|----------|------------|
| `internal/automation/model_quota.go` | `internal/telemetry/quota_models.go` | dependency / consumer | P1-A | edit | `E0-P0A-SPEC1` | Chỉ tạo mới trong `internal/automation/`, không sửa `internal/telemetry/` |
| `internal/automation/sorter.go` | `internal/automation/model_quota.go` | consumer | P1-A | edit | `E0-P0A-SPEC1` | Tuân thủ 100% logic Burn-down Sorter trong spec |
| `internal/automation/retry.go` | Language Server Connect-RPC | consumer | P1-B | edit | `E0-P0A-SPEC1` | Gửi empty content list, tối đa 3 lần, không restart tiến trình |
| `internal/automation/controller.go` | `internal/manager/account_manager.go` | caller | P1-B | edit | `E0-P0A-SPEC1` | Bắt buộc kiểm soát Single-Flight Gate qua `atomic.Bool` |
| `internal/ui/automation_view.go` | `internal/ui/main_window.go` | consumer / embed | P1-C | edit | `E0-P0A-SPEC1` | Thread-safe trên UI thread Fyne, không làm xáo trộn layout có sẵn |
| `internal/manager/account_manager.go` | `internal/automation/controller.go` | dependency | P1-B | preserve-only | `E0-P0A-SRC1` | Tuyệt đối không thay đổi mã nguồn `SwitchAccount()` |
| `internal/telemetry/` | `internal/automation/controller.go` | data provider | P1-B | inspect-only | `E0-P0A-SRC2` | Chỉ đọc cache, không can thiệp vào vòng lặp 60s |
| `internal/reloader/` | `internal/manager/account_manager.go` | downstream consumer | P1-B | preserve-only | `E0-P0A-SRC3` | Không gọi thêm continuation lần 2 bên ngoài |

## Detailed Findings

### `internal/automation/model_quota.go`

Current state:
Chưa tồn tại thư mục `internal/automation/` cũng như file `model_quota.go`. Chưa có logic tự động ánh xạ Active Model sang Quota Group (Gemini vs Claude/GPT), chưa có tam phân thái hạn ngạch (`APPLICABLE_VALID`, `NOT_APPLICABLE`, `UNKNOWN_UNCERTAIN`), và chưa xử lý cờ `disabled: true` trên bucket 5h.

Required state:
```text
Cung cấp types QuotaCategory, QuotaState, struct ModelQuotaEvaluation và các hàm ResolveModelCategory, ClassifyQuotaState xử lý chuẩn xác 100% các quy định tại Mục 3 của Báo cáo Kiến trúc.
```

Evidence:
- `E0-P0A-SPEC1`: Báo cáo kiến trúc Mục 3 & Mục 9.
- `E0-P0A-DIR1`: Thư mục `internal/automation/` chưa tồn tại trong repo.

Classification: `missing`
Allowed next action: Implement trong slice P1-A.
Forbidden next action: Không can thiệp sửa đổi các struct trong `internal/telemetry/`.

---

### `internal/automation/sorter.go`

Current state:
Chưa có module nào lọc và sắp xếp ứng viên tài khoản theo hạn mức khả dụng và mô hình hoạt động.

Required state:
```text
Cung cấp hàm FilterEligibleCandidates (5 tiêu chí loại trừ) và SortCandidatesBurnDown (4 tầng ưu tiên: Tuần thấp nhất -> Có 5h -> 5h cao nhất -> Danh bạ cố định) tái hiện 100% bảng phân xử mẫu A-F.
```

Evidence:
- `E0-P0A-SPEC1`: Báo cáo kiến trúc Mục 6.

Classification: `missing`
Allowed next action: Implement trong slice P1-A.
Forbidden next action: Không kích hoạt chuyển tài khoản từ module sorter.

---

### `internal/automation/retry.go`

Current state:
Khi Antigravity gặp lỗi cạn kiệt Quota hoặc Rate Limit (`429`), người dùng phải bấm "Try again" thủ công trên giao diện IDE. Chưa có cơ chế bắt stream `StreamAgentStateUpdates` và gửi Connect-RPC với empty content list tự động.

Required state:
```text
Module RetryCoordinator tự động gửi Connect-RPC SendUserCascadeMessage với danh sách nội dung rỗng tối đa 3 lần (< 500ms mỗi lần). Nếu khôi phục thành công -> tiếp tục phiên; nếu thất bại cả 3 lần -> báo ErrRetryExhausted.
```

Evidence:
- `E0-P0A-SPEC1`: Báo cáo kiến trúc Mục 2 (Phát hiện 1 & 2) & Mục 5.

Classification: `missing`
Allowed next action: Implement trong slice P1-B.
Forbidden next action: Không thực hiện `taskkill` hay restart server trong module retry.

---

### `internal/automation/controller.go`

Current state:
Chưa có bộ điều phối kích hoạt kép kết hợp tín hiệu chủ động 60s và tín hiệu bị động 429 với Single-Flight Concurrency Gate.

Required state:
```text
AutomationController liên kết telemetry loop, error stream, retry coordinator, sorter, và gọi SwitchAccount an toàn với khóa atomic.Bool và sync.Mutex, bảo đảm không bao giờ xảy ra chạy đè hoặc lặp vô tận.
```

Evidence:
- `E0-P0A-SPEC1`: Báo cáo kiến trúc Mục 4 & Mục 7.

Classification: `missing`
Allowed next action: Implement trong slice P1-B.
Forbidden next action: Không gọi hàm tiếp tục phiên lần thứ hai bên ngoài `SwitchAccount`.

---

### `internal/ui/automation_view.go`

Current state:
Giao diện Fyne GUI của switcher chỉ có danh sách tài khoản, trạng thái quota, và các nút chuyển đổi thủ công. Chưa có Toggle công tắc bật/tắt automation hay nhãn hiển thị trạng thái automation.

Required state:
```text
Component AutomationView nhúng vào Fyne GUI với switch/check toggle và nhãn hiển thị trạng thái động (Đang giám sát, Đang thử lại, Đang chuyển đổi, Chờ an toàn).
```

Evidence:
- `E0-P0A-SPEC1`: Báo cáo kiến trúc Mục 9 & Mục 10.

Classification: `missing`
Allowed next action: Implement trong slice P1-C.
Forbidden next action: Không tạo process systray riêng biệt gây xung đột với Fyne.

---

### `internal/manager/account_manager.go`

Current state:
Hàm `AccountManager.SwitchAccount()` đã hoàn thiện đầy đủ 4 bước: ghi WinCred, đồng bộ ToS, `taskkill` language_server.exe, chờ đủ 8 giây an toàn, và tự động gọi continuation cho các phiên trong vòng 2h qua `internal/reloader`.

Required state:
```text
Giữ nguyên vẹn làm điểm thực thi chuyển đổi đích duy nhất mà AutomationController gọi tới.
```

Evidence:
- `E0-P0A-SRC1`: Kiểm tra mã nguồn `internal/manager/account_manager.go`.

Classification: `correct`
Allowed next action: Preserve-only.
Forbidden next action: Tuyệt đối không sửa đổi hoặc tái tạo lại logic chuyển đổi hotswap đã ổn định.

---

### `internal/telemetry/` & `internal/reloader/`

Current state:
`internal/telemetry` hoạt động định kỳ 60s lấy dữ liệu quota từ Google API và lưu vào cache. `internal/reloader` quản lý snapshot SQLite và phục hồi active threads trong 2h.

Required state:
```text
Giữ nguyên vẹn, phục vụ như nhà cung cấp dữ liệu quota và cơ chế phục hồi ngữ cảnh ngầm cho switcher.
```

Evidence:
- `E0-P0A-SRC2`: Kiểm tra `internal/telemetry/`.
- `E0-P0A-SRC3`: Kiểm tra `internal/reloader/`.

Classification: `correct`
Allowed next action: Preserve-only và Inspect-only.
Forbidden next action: Không chỉnh sửa mã nguồn cốt lõi của hai package này.

## Next Phase Status Decisions

| Plan Item | Actual Status Finding | Required Status / Next-Action Update |
|-----------|-----------------------|--------------------------------------|
| P1-A | `internal/automation/model_quota.go` & `sorter.go` are `missing`. | Proceed with implementing P1-A leaf tasks. |
| P1-B | `internal/automation/retry.go` & `controller.go` are `missing`. | Proceed with implementing P1-B leaf tasks. |
| P1-C | `internal/ui/automation_view.go` is `missing`. | Proceed with implementing P1-C leaf tasks. |
| P2-A | E2E test harness for 14 edge cases is `missing`. | Proceed with implementing P2-A leaf tasks. |

## Implementation Gate

- [x] Target scope is listed in Current Status Matrix.
- [x] Each target unit has a status.
- [x] Each status has evidence IDs.
- [x] Each target file has relationship count evidence from architectural specification and source inspection.
- [x] Phase Touch Map lists plan-relevant relationship files that can affect the current phase/slice.
- [x] Phase Touch Map defines touch mode for every plan-relevant relationship unit that may be affected.
- [x] Correct parts are marked preserve-only.
- [x] Partial, missing, wrong, unbound, and fake-or-stub parts have exact next actions.
- [x] Blockers are recorded, if any (No blockers identified).
- [x] Next phase status assumptions, next action, and work steps have been updated from this status file when needed.
- [x] Status Refresh Log has an R0 baseline row.
- [ ] If implementation has started, affected Current Status Matrix rows have been refreshed from latest evidence (N/A for P0).
- [ ] If refreshed statuses changed next work, only the stale next-phase status assumptions, next action, or work steps have been updated before the next phase (N/A for P0).

## Final P0 Decision

- [x] P0 complete. Next phase can proceed unchanged.

Decision note:

Toàn bộ hiện trạng mã nguồn và kiến trúc đã được đối soát chi tiết dựa trên Báo cáo Kiến trúc `rp_system-architect_261020_141500_by_gemini-2-5-pro_autonomous_account_switching_architecture_specification.md` và mã nguồn hiện hữu trong repository. Nền tảng hạ tầng chuyển đổi tài khoản (`internal/manager`, `internal/telemetry`, `internal/reloader`) đã sẵn sàng ở trạng thái `correct` (preserve-only). Các thành phần tự động hóa mới (`internal/automation/`, `internal/ui/automation_view.go`) được phân loại rõ ràng là `missing` và đã được lập lịch chi tiết trong các Slices P1-A, P1-B, P1-C, P2-A. Không có rào cản kỹ thuật hay xung đột nào. Kế hoạch sẵn sàng chuyển sang thực thi.
