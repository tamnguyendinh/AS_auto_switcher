# Autonomous Account Switching Benchmark Ledger

## Metadata

- Date: `2026-10-20`
- Plan: `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-plan.md`
- Rules: `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-plan.md`
- Evidence: `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-evidence.md`
- Benchmark: `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-benchmark.md`
- Actual status: `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-actual-status.md`

## Benchmark Rules

The benchmark file records measurements.

It should contain:

- metadata and companion files;
- benchmark rules;
- benchmark sections such as `B0`, `B1`, `B2`;
- metric tables with unit, baseline, latest, final, target, and delta when needed;
- inventory count;
- runtime or performance metric;
- graph, coverage, or accuracy metric;
- package, bundle, file size, or hash metric;
- before/after numbers;
- UI, layout, or browser metric when the plan involves UI;
- command-surface or generated-output inventory when the plan involves generated documentation.

Benchmark records measured numbers only. Do not put command logs, design decisions, or validation narrative here. Build/test/e2e pass-fail belongs in evidence unless the timing, count, or size is the measured target.

Benchmark sections must follow the plan phases:

- `B0` corresponds to `P0`.
- `B1` corresponds to `P1` (Slices P1-A, P1-B, P1-C).
- `B2` corresponds to `P2` (Slice P2-A).
- Use item-level IDs such as `B-P1-A` when a checklist item needs separate benchmark evidence.
- Create a benchmark section only when the matching phase has benchmarkable measurements.
- Do not invent fixed metric categories; record the measurements required by the matching plan phase.

## B0 - P0 Benchmarks

| Phase | Metric | Unit | Baseline | Latest | Final | Target | Delta | Evidence |
|-------|--------|------|----------|--------|-------|--------|-------|----------|
| P0 | File Count: Package `internal/automation` | files | 0 | 0 | TBD | 4 | +4 | `E0-P0A-DIR1` |
| P0 | Test Coverage: Automation Package | % | 0% | 0% | TBD | >= 90% | +90% | `E0-P0A-DIR1` |
| P0 | Baseline Switch Account Delay | seconds | 8.0s | 8.0s | 8.0s | 8.0s | 0s | `E0-P0A-SRC1` |
| P0 | Active Threads Continuation Window | hours | 2.0h | 2.0h | 2.0h | 2.0h | 0h | `E0-P0A-SRC3` |

## B1 - P1 Benchmarks

| Phase | Metric | Unit | Baseline | Latest | Final | Target | Delta | Evidence |
|-------|--------|------|----------|--------|-------|--------|-------|----------|
| P1-A | Model Category Mapping Latency | microseconds | N/A | TBD | TBD | < 50µs | N/A | `E1-P1A-TEST1` |
| P1-A | 50-Candidate Sorter Execution Duration | milliseconds | N/A | TBD | TBD | < 1.0ms | N/A | `E1-P1A-BENCH1` |
| P1-A | Burn-Down Sorter Accuracy vs Spec Table A-F | % | 0% | TBD | TBD | 100% | +100% | `E1-P1A-TEST1` |
| P1-A | Candidate Elimination Rate (5h<=2% & Week=0%) | % | 0% | TBD | TBD | 100% | +100% | `E1-P1A-TEST1` |
| P1-B | Native In-Memory Single Retry Latency | milliseconds | N/A | TBD | TBD | < 500ms | N/A | `E1-P1B-TEST1` |
| P1-B | Max Native Retry Attempts Limit | count | 0 | TBD | TBD | 3 | +3 | `E1-P1B-TEST1` |
| P1-B | Single-Flight Concurrency Leaks Under 100 Concurrent Triggers | count | N/A | TBD | TBD | 0 | 0 | `E1-P1B-CONCUR1` |
| P1-B | Telemetry Proactive Evaluation Interval | seconds | 60s | 60s | 60s | 60s | 0s | `E1-P1B-TEST1` |
| P1-C | UI Toggle Response / State Transition Latency | milliseconds | N/A | TBD | TBD | < 50ms | N/A | `E1-P1C-BUILD1` |

## B2 - P2 Benchmarks

| Phase | Metric | Unit | Baseline | Latest | Final | Target | Delta | Evidence |
|-------|--------|------|----------|--------|-------|--------|-------|----------|
| P2-A | 14 Edge Cases Simulation Pass Rate | % | 0% | TBD | TBD | 100% (14/14) | +100% | `E2-P2A-TEST14` |
| P2-A | Overall Package Test Coverage | % | 0% | TBD | TBD | >= 90% | +90% | `E2-P2A-TEST14` |
| P2-A | Full Build Compilation Status (`go build ./...`) | exit code | 0 | 0 | TBD | 0 | 0 | `E2-P2D-BUILD1` |

## Non-Benchmarkable Notes

- **Language Server Crash Recovery Cooldown:** 8 giây chờ sau khi `taskkill` là thông số cứng bắt buộc của hệ thống Antigravity (`monitorLsCrashInternal`), không thể tối ưu hóa thành < 8 giây nếu muốn tránh lỗi sập socket.
- **Model Quota Mapping:** Việc chuyển đổi chuỗi tên model sang category là thao tác ánh xạ bộ nhớ tức thời (O(1)), thời gian thực thi là không đáng kể.
