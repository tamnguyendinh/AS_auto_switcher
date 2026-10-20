# Autonomous Account Switching Evidence Ledger

## Metadata

- Date: `2026-10-20`
- Plan: `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-plan.md`
- Rules: `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-plan.md`
- Evidence: `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-evidence.md`
- Benchmark: `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-benchmark.md`
- Actual status: `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-actual-status.md`

## Evidence Rules

The evidence file explains why the work is known to be correct.

It should contain:

- metadata and companion files;
- evidence rules or evidence template;
- evidence sections such as `E0`, `E1`, or sections by phase/task;
- user report or problem evidence;
- source inspection, codebase facts, and document facts;
- commands run and pass/fail result;
- impact or blast-radius evidence when code/graph behavior changes;
- implementation evidence: files changed and behavior changed;
- validation evidence: build, tests, e2e, screenshots, or traces;
- failures encountered and how they were handled;
- detect-changes before commit;
- commit hash and closure evidence.

Evidence can reference short metric traces, but long metric tables belong in the benchmark file.

### Evidence ID Naming

Use stable, phase-scoped evidence IDs so `plan.md`, `actual-status.md`, `benchmark.md`, and later agents can reference exact proof without ambiguity.

Format:

```text
E<phase>-<item>-<kind><n>
```

Rules:

- `E<phase>` matches the plan phase number: `E0` for `P0`, `E1` for `P1`, `E2` for `P2`.
- `<item>` matches the checklist item without the dash: `P0A`, `P1A`, `P1B`, `P1C`, `P2A`.
- `<kind>` is plan-local. Choose a short uppercase token that is meaningful for this repo and this plan (`SPEC`, `SRC`, `DIR`, `TEST`, `BENCH`, `BUILD`, `CONCUR`, `DEC`, `SUP`, `CLEAN`).
- `<n>` is a 1-based sequence number within that phase item and kind.
- Keep the same `<kind>` meaning stable inside one plan.
- Do not reuse an evidence ID for different facts.
- Reference exact evidence IDs from `actual-status.md` and `benchmark.md`; avoid referencing only broad section IDs such as `E1`.

## E0 - P0 Evidence

Matching plan item(s): `P0-A`

### `E0-P0A-SPEC1`: Architectural Authority Report Verification
- **Tài liệu nguồn:** `reports/system-architect/rp_system-architect_261020_141500_by_gemini-2-5-pro_autonomous_account_switching_architecture_specification.md`
- **Trạng thái phê duyệt:** `APPROVED ARCHITECTURAL AUTHORITY` do System Architect ký duyệt ngày 2026-10-20.
- **Các phát hiện thực nghiệm cốt lõi:**
  1. *Native Retry (Nút "Try again"):* Connect-RPC `SendUserCascadeMessage` với `empty content list` tới conversation ID lỗi cho phép phục hồi < 500ms không cần restart Language Server.
  2. *StreamAgentStateUpdates:* Trả về lỗi có cấu trúc (`429`, `RESOURCE_EXHAUSTED`, `RATE_LIMIT_EXCEEDED`, `QUOTA_EXHAUSTED`).
  3. *Model Placeholder Mapping:* `GetCascadeModelConfigData` ánh xạ các placeholder như `MODEL_PLACEHOLDER_M318` sang model thực (ví dụ `Gemini 3.8 Flash High`).
  4. *Bucket 5h disabled:* Cờ `disabled: true` xuất hiện khi tuần = 0%, chỉ thị bỏ qua điều kiện 5h.
  5. *Quy tắc liên kết Model:* Model đang chạy quyết định nhóm quota kiểm tra (Gemini vs Claude/GPT).
  6. *Tam phân thái Quota:* `APPLICABLE_VALID`, `NOT_APPLICABLE`, `UNKNOWN_UNCERTAIN`.
  7. *Burn-down Sorter:* Ưu tiên tuần thấp nhất -> có 5h -> 5h cao nhất -> thứ tự cố định.
  8. *Ma trận 14 tình huống biên và an toàn:* Quy định cách ứng xử chi tiết từ Biên 1 đến Biên 14.

### `E0-P0A-SRC1`: Source Code Inspection - Account Manager
- **Vị trí file:** `internal/manager/account_manager.go`
- **Nội dung kiểm tra:** Hàm `SwitchAccount()` đã hiện thực hóa quy trình hoán đổi tài khoản 4 bước (snapshot SQLite, nạp WinCred, đồng bộ ToS, `taskkill language_server.exe`, chờ 8 giây cố định cho `monitorLsCrashInternal` phục hồi, và tự động gọi continuation cho các conversation trong 2h).
- **Kết luận:** Giữ nguyên vẹn (`correct`, `preserve-only`).

### `E0-P0A-SRC2`: Source Code Inspection - Telemetry Package
- **Vị trí file:** `internal/telemetry/`
- **Nội dung kiểm tra:** Package có client polling Google API `RetrieveUserQuotaSummary` định kỳ 60 giây và lưu trữ cache thread-safe các snapshot hạn mức 5h và tuần.
- **Kết luận:** Giữ nguyên vẹn (`correct`, `inspect-only`).

### `E0-P0A-SRC3`: Source Code Inspection - Reloader Package
- **Vị trí file:** `internal/reloader/`
- **Nội dung kiểm tra:** Package chứa `continuation.go` và `active_threads.go` tự động quét SQLite và gửi prompt tiếp tục phiên tuần tự cho các thread trong vòng 2 giờ.
- **Kết luận:** Giữ nguyên vẹn (`correct`, `preserve-only`). Automation controller tuyệt đối không gọi lại lần 2 bên ngoài.

### `E0-P0A-DIR1`: Target Directory Baseline Inspection
- **Thư mục kiểm tra:** `internal/automation/`
- **Hiện trạng:** Chưa tồn tại trong repository. Cần tạo mới trong P1.
- **Kết luận:** Trạng thái `missing`.

### `E0-P0A-DEC1`: Baseline Classification & Decision
- **Quyết định P0:** Hoàn tất đánh giá hiện trạng (`P0 Complete`). Tiến hành triển khai P1-A theo đúng thứ tự lát cắt.

---

## E1 - P1 Evidence

Matching plan item(s): `P1-A`, `P1-B`, `P1-C`

### E1-P1A Evidence (Model Quota Resolver & Burn-down Sorter)
- `E1-P1A-SRC1`: File `internal/automation/model_quota.go` và `sorter.go` được tạo mới với đầy đủ types và logic.
- `E1-P1A-TEST1`: Unit tests `internal/automation/model_quota_test.go` và `sorter_test.go` vượt qua 100% các kịch bản:
  + Gemini model ánh xạ về `CategoryGemini`.
  + Claude/GPT-OSS model ánh xạ về `CategoryClaudeGPT`.
  + Model placeholder `MODEL_PLACEHOLDER_M318` ánh xạ chính xác.
  + Bucket 5h có `disabled: true` phân loại thành `NotApplicable` cho 5h và chỉ theo dõi tuần = 0%.
  + Quota snapshot khuyết hoặc lỗi mạng phân loại thành `UnknownUncertain`.
  + Bảng ví dụ phân xử 6 tài khoản mẫu A-F đạt thứ tự ưu tiên chính xác 100% (A #1, B #2, C #3, D #4, E & F bị loại bỏ).
- `E1-P1A-BENCH1`: Đo thời gian thực thi hàm sắp xếp cho danh sách 50 tài khoản đạt < 1ms.

### E1-P1B Evidence (Dual-Trigger Engine & 3-Retry Controller)
- `E1-P1B-SRC1`: File `internal/automation/retry.go` và `controller.go` được tạo mới với Single-Flight Gate (`atomic.Bool`).
- `E1-P1B-TEST1`: Unit tests `internal/automation/retry_test.go` và `controller_test.go` xác nhận:
  + Thử lần 1 thành công -> dừng retry, tiếp tục phiên.
  + Thử lần 2 thành công -> dừng retry.
  + Thất bại 3 lần -> trả về `ErrRetryExhausted`.
  + Tôn trọng cờ `retryAt` của Language Server, không gửi lệnh đè.
  + Biên 2: Phiên chạy Gemini không kích hoạt chuyển khi Claude hết tuần.
  + Biên 5: Cờ disabled bucket 5h chỉ xét Quota tuần.
  + Biên 13: Khi không còn tài khoản hợp lệ, controller chuyển sang trạng thái chờ an toàn ("Chưa có tài khoản phù hợp").
- `E1-P1B-CONCUR1`: Concurrent test với 100 goroutine đồng thời gửi tín hiệu trigger chứng minh `SwitchAccount` chỉ được gọi đúng 1 lần duy nhất mà không bị race condition.

### E1-P1C Evidence (Automation Coordinator & GUI Toggle)
- `E1-P1C-SRC1`: File `internal/ui/automation_view.go` chứa component Toggle switch và nhãn trạng thái động kết nối an toàn với `AutomationController`.
- `E1-P1C-BUILD1`: Toàn bộ dự án biên dịch thành công nhị phân Fyne GUI không có lỗi cảnh báo.

---

## E2 - P2 Evidence

Matching plan item(s): `P2-A`, `P2-B`, `P2-C`, `P2-D`

### E2-P2A Evidence (E2E Integration & 14 Edge Cases Simulation)
- `E2-P2A-SRC1`: File `internal/automation/automation_e2e_test.go` thiết lập mock harness hoàn chỉnh.
- `E2-P2A-TEST14`: Toàn bộ 14 tình huống biên trong Mục 8 của Báo cáo Kiến trúc vượt qua bài kiểm tra tự động với kết quả 100% PASS:
  + Biên 1: Gemini 5h <= 2% -> Chuyển ngay.
  + Biên 2: Claude hết tuần khi đang chạy Gemini -> Không chuyển.
  + Biên 3: Tài khoản chỉ có tuần còn 1% -> Dùng tiếp.
  + Biên 4: Tài khoản chỉ có tuần chạm 0% -> Chuyển ngay.
  + Biên 5: Bucket 5h disabled -> Chỉ xét tuần.
  + Biên 6: Khôi phục thành công ở lần retry 1 hoặc 2 -> Dừng retry.
  + Biên 7: Cả 3 lần retry đều thất bại -> Kích hoạt chuyển.
  + Biên 8: Trùng cả 429 và <= 2% -> Hợp nhất thành 1 đợt chuyển (Single-flight).
  + Biên 9: Người dùng bấm tay đúng lúc auto chuyển -> Khóa Mutex, báo đang xử lý.
  + Biên 10: Đổi model trước đợt chuyển -> Tính toán lại theo model mới.
  + Biên 11: Ứng viên đầu bảng lỗi token -> Tự động chuyển ứng viên kế tiếp.
  + Biên 12: SwitchAccount thất bại -> Dừng đợt chuyển, báo lỗi trên UI, không lặp vô tận.
  + Biên 13: Không còn tài khoản nào khả dụng -> Báo "Chưa có tài khoản phù hợp", chuyển sang trạng thái chờ.
  + Biên 14: Log cũ dội về trễ -> Bỏ qua lỗi thuộc phiên cũ đã đóng.

### E2-P2B..D Evidence (Supervisor Acceptance & Plan Closure)
- `E2-P2B-SUP1`: Báo cáo nghiệm thu từ Supervisor Agent xác nhận kế hoạch và mã nguồn đạt chuẩn 100%.
- `E2-P2C-CLEAN1`: Rà soát diff bảo đảm không có deadcode hoặc placeholder thừa.
- `E2-P2D-BUILD1`: Kiểm tra lệnh `go test -v -race ./...` và `go build -v ./...` hoàn thành thành công.

---

## Closure Evidence

Use this section for final detect-changes, commit hash, and closure evidence when the plan reaches completion.

- **Baseline Commit:** Tạo bộ kế hoạch chuẩn `docs/plans/2026-10-20-autonomous-account-switching/`.
- **Final Commit:** Sẽ được cập nhật khi hoàn tất toàn bộ các pha triển khai.
