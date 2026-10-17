# Phase 6A: Multi-Account Quota Telemetry Engine, Live Dashboard & Multi-Theme System Benchmark Ledger

## Metadata

- Date: `2026-10-17`
- Plan: `docs/plans/2026-10-17-phase-6a-quota-telemetry-dashboard/2026-10-17-phase-6a-quota-telemetry-dashboard-plan.md`
- Rules: `docs/plans/2026-10-17-phase-6a-quota-telemetry-dashboard/2026-10-17-phase-6a-quota-telemetry-dashboard-plan.md`
- Evidence: `docs/plans/2026-10-17-phase-6a-quota-telemetry-dashboard/2026-10-17-phase-6a-quota-telemetry-dashboard-evidence.md`
- Benchmark: `docs/plans/2026-10-17-phase-6a-quota-telemetry-dashboard/2026-10-17-phase-6a-quota-telemetry-dashboard-benchmark.md`
- Actual status: `docs/plans/2026-10-17-phase-6a-quota-telemetry-dashboard/2026-10-17-phase-6a-quota-telemetry-dashboard-actual-status.md`

## Benchmark Rules

The benchmark file records quantitative measurements.
It contains:
- Baseline metrics established prior to implementation (B0);
- Target metrics for each implementation slice (B1, B2, B3);
- Exact units, targets, latest recorded values, and deltas;
- Strictly measured numbers; design narrative and bug reports belong in `evidence.md`.

---

## B0 - P0 Baseline Measurements

| Phase | Metric | Unit | Baseline | Latest | Final | Target | Delta | Evidence |
|---|---|:---:|:---:|:---:|:---:|:---:|:---:|---|
| P0 | Stored Accounts Inventory | accounts | 7 | 7 | - | 7 | 0 | E0-P0A-STORE1 |
| P0 | Test Suite Package Pass Rate | packages | 8/8 (100%) | 8/8 (100%) | - | 8/8 (100%) | 0 | E0-P0A-TEST1 |
| P0 | Language Server Discovery Latency | ms | 10 | 10 | - | <= 20 | 0 | E0-P0A-TEST1 |
| P0 | Standalone CLI Cold Startup | ms | ~25 | ~25 | - | <= 50 | 0 | E0-P0A-TEST1 |
| P0 | Empirical Parallel Quota Query Latency | s | 1.69 | 1.69 | - | < 2.0 | 0 | E0-P0A-SPEC1 |
| P0 | Empirical Quota Fetch Success Rate | % | 100% | 100% | - | 100% | 0 | E0-P0A-SPEC1 |

---

## B1 - P1 (Slice P6A-1) Telemetry Engine & Loop Benchmarks

| Phase | Metric | Unit | Baseline | Latest | Final | Target | Delta | Evidence |
|---|---|:---:|:---:|:---:|:---:|:---:|:---:|---|
| P1 | Parallel Worker Pool Latency (7 accounts) | s | 1.69 | 0.056 | 0.056 | < 2.0 | -1.944s | E1-P1A-TEST3 |
| P1 | Google Quota Fetch Success Rate | % | 100% | 100% | 100% | 100% | 0 | E1-P1A-TEST1 |
| P1 | Auto-Telemetry Loop Interval | s | - | 60s | 60s | 60 ± 1 | 0 | E1-P1A-CODE4 |
| P1 | Single-Flight Overlap Drop Rate | % | - | 100% | 100% | 100% | 0 | E1-P1A-CODE4 |
| P1 | Unit Test Coverage (`internal/telemetry`) | % | 0% | 92.8% | 92.8% | >= 90% | +2.8% | E1-P1A-TEST1 |
| P1 | Data Race Free (`go test -race`) | issues | 0 | 0 | 0 | 0 | 0 | E1-P1A-TEST2 |

---

## B2 - P2 (Slice P6A-2) Fyne UI Dashboard & Multi-Theme Benchmarks

| Phase | Metric | Unit | Baseline | Latest | Final | Target | Delta | Evidence |
|---|---|:---:|:---:|:---:|:---:|:---:|:---:|---|
| P2 | GUI Frame Rate during Background Polling | FPS | 60 | - | - | 60 | - | E2-P2A-QA1 |
| P2 | Theme Switching Latency (without restart) | ms | - | - | - | <= 50 | - | E2-P2A-TEST1 |
| P2 | Supported Themes Count | count | 1 (Dark) | - | - | 3 (Dark/Light/Warm) | - | E2-P2A-CODE1 |
| P2 | UI Traceability Matrix Pass Rate | items | 0/8 | - | - | 8/8 (100%) | - | E2-P2A-QA1 |

---

## B3 - P3 (Slice P6A-3) CLI Integration & Binary Profile Benchmarks

| Phase | Metric | Unit | Baseline | Latest | Final | Target | Delta | Evidence |
|---|---|:---:|:---:|:---:|:---:|:---:|:---:|---|
| P3 | CLI `--quota` Execution Latency (Live Fetch) | s | - | - | - | <= 2.5 | - | E3-P3A-TEST1 |
| P3 | CLI `--quota` Execution Latency (From Cache) | ms | - | - | - | <= 50 | - | E3-P3A-TEST1 |
| P3 | Compiled Binary RAM Footprint (CLI / GUI) | MB | ~18 | - | - | <= 20 | - | E3-P3A-BENCH1 |
| P3 | Dual-Build Compliance (`cli` & `gui` tags) | builds | 2/2 | - | - | 2/2 (100%) | - | E3-P3A-CODE2 |

---

## Non-Benchmarkable Notes
- Zero-regression requirement: hot-swap switching latency and session reloader behavior must not be degraded by the telemetry engine or UI theme additions.
- The 60-second loop utilizes `time.Ticker` and runs in background goroutines with less than 0.02% CPU consumption.


