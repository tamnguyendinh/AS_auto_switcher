# Autonomous Account Switching Benchmark Ledger

## Metadata

- Date: 2026-10-20
- Revision: R1 — implementation not started
- Plan: docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-plan.md
- Rules: mục Rules của companion plan
- Evidence: docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-evidence.md
- Benchmark: docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-benchmark.md
- Actual status: docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-actual-status.md

## Benchmark Rules

- Latest/Final chỉ chứa số đã đo bằng command/runtime được tham chiếu.
- Chưa đo ghi NOT MEASURED, không lấy 0 hoặc target thay baseline.
- Test/build pass-fail nằm trong evidence; không điền exit code 0 trước khi chạy.
- Hằng số source, policy thresholds và số lần tối đa là cấu hình, không phải benchmark latency.
- Tách thời gian gửi/ACK RPC khỏi kết quả hồi phục lượt chạy. Không có cam kết recovery dưới 500ms, model mapping dưới 50µs hoặc GUI dưới 50ms trong phương án Owner đã chốt.
- Không chạy benchmark chỉ để hoàn thành việc sửa tài liệu.

## B0 - P0 Benchmarks

Không có phép đo hiệu năng mới. Inventory quan sát source:

| Metric | Unit | Baseline observed | Latest | Final | Evidence |
|---|---|---:|---:|---|---|
| Production files trong internal/automation/ | files | 0 | 0 | Chưa triển khai | E0-P0A-DIR1 |
| Công tắc Tự động hiện có trên header | controls | 1 | 1 | Giữ nguyên | E0-P0A-SRC5 |

## B1 - P1 Benchmarks

Các mục dưới đây là vị trí dành cho số đo nếu thu thập trong implementation; không phải kết quả hiện tại.

| Item | Metric | Unit | Baseline | Latest | Final | Target | Evidence target |
|---|---|---|---|---|---|---|---|
| P1-A | Sorter trên 50 ứng viên | ms | NOT MEASURED | NOT MEASURED | NOT MEASURED | Ghi thực tế, không SLA mới | E1-P1A-BENCH1 |
| P1-B | Số recovery actions trong scenario ba lần thất bại | count | NOT MEASURED | NOT MEASURED | NOT MEASURED | 3 theo policy | E1-P1B-TEST1 |
| P1-B | Số SwitchAccount đồng thời trong thử contention | count | NOT MEASURED | NOT MEASURED | NOT MEASURED | Tối đa 1 | E1-P1B-CONCUR1 |
| P1-C | Số toggle automation sau binding | controls | 1 | NOT MEASURED | NOT MEASURED | 1 | E1-P1C-UI1 |

## B2 - P2 Benchmarks

| Item | Metric | Unit | Baseline | Latest | Final | Target | Evidence target |
|---|---|---|---|---|---|---|---|
| P2-A | Scenario có kết quả đúng | count | NOT MEASURED | NOT MEASURED | NOT MEASURED | 14/14 và binding checks riêng | E2-P2A-TEST14, E2-P2A-BIND1 |

Không thêm coverage percentage hoặc runtime timing chưa được yêu cầu/đo. Scenario counts chỉ điền sau khi suite thực sự chạy.

## Non-Benchmarkable Notes

Các giá trị sau là source configuration/policy được giữ, không phải thời gian vận hành đã đo:

| Giá trị | Ý nghĩa | Nguồn |
|---|---|---|
| 60 giây | Chu kỳ telemetry hiện có | E0-P0A-SRC2 |
| 8 giây | Khoảng chờ trong engine sau restart, không phải tổng switch latency | E0-P0A-SRC1 |
| 2 giờ | Lookback conversation scanner/continuation | E0-P0A-SRC3 |
| 3 giây | Pacing giữa các thread continuation, không phải lịch Retry của automation | E0-P0A-SRC3 |
| 5h <= 2%; tuần = 0% | Điều kiện proactive Owner chốt | Plan R2/R3 |
| Tối đa 3 recovery actions | Ngân sách khôi phục của automation; đợi execution outcome | Plan R4 |

Build/tests sản phẩm và phép đo hiệu năng: NOT RUN trong lượt sửa markdown R1.
