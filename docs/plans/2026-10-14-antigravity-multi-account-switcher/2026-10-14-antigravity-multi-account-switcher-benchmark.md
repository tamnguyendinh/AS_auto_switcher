# Antigravity Multi-Account Switcher Benchmark Ledger

## Metadata

- Date: `2026-10-14`
- Plan: `docs/plans/2026-10-14-antigravity-multi-account-switcher/2026-10-14-antigravity-multi-account-switcher-plan.md`
- Rules: `docs/plans/2026-10-14-antigravity-multi-account-switcher/2026-10-14-antigravity-multi-account-switcher-plan.md#rules`
- Evidence: `docs/plans/2026-10-14-antigravity-multi-account-switcher/2026-10-14-antigravity-multi-account-switcher-evidence.md`
- Benchmark: `docs/plans/2026-10-14-antigravity-multi-account-switcher/2026-10-14-antigravity-multi-account-switcher-benchmark.md`
- Actual status: `docs/plans/2026-10-14-antigravity-multi-account-switcher/2026-10-14-antigravity-multi-account-switcher-actual-status.md`

## Benchmark Rules

The benchmark file records measurements.

It should contain:

- metadata and companion files;
- benchmark rules;
- benchmark sections such as `B0`, `B1`, or sections by phase/task;
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
- `B1` corresponds to `P1`.
- `B2` corresponds to `P2`.
- `B3` corresponds to `P3`.
- `B4` corresponds to `P4`.
- `B5` corresponds to `P5`.

## B0 - P0 Benchmarks

| Phase | Metric | Unit | Baseline | Latest | Final | Target | Delta | Evidence |
|-------|--------|------|----------|--------|-------|--------|-------|----------|
| P0 | Unit test pass count | tests | 6 | 6 | - | >= 6 | 0 | `E0-P0A-TEST1` |
| P0 | Unit test execution duration | seconds | 0.174 | 0.174 | - | < 1.000 | 0.000 | `E0-P0A-TEST1` |
| P0 | Python source files inventory | files | 10 | 10 | - | 11 | 0 | `E0-P0A-SRC1..SRC8` |
| P0 | Active language server instances | processes | 1 | 1 | - | 1 | 0 | `E0-P0A-PROC1` |
| P0 | Blacklist keyword rejection rate | % | 100% | 100% | - | 100% | 0% | `E0-P0A-SEC1` |
| P0 | Accounts in `accounts_store/` | files | 0 | 0 | - | >= 1 | 0 | `E0-P0A-SRC1` |

## B1 - P1 Benchmarks

| Phase | Metric | Unit | Baseline | Latest | Final | Target | Delta | Evidence |
|-------|--------|------|----------|--------|-------|--------|-------|----------|
| P1 | JSON account storage I/O latency | ms | - | - | - | < 20 | - | `E1-P1A-STORE1` |
| P1 | Credential write duration | ms | - | - | - | < 50 | - | `E1-P1B-CRED1` |
| P1 | Account store test coverage | tests | 1 | - | - | >= 3 | - | `E1-P1A-STORE2` |

## B2 - P2 Benchmarks

| Phase | Metric | Unit | Baseline | Latest | Final | Target | Delta | Evidence |
|-------|--------|------|----------|--------|-------|--------|-------|----------|
| P2 | Language server port/CSRF discovery latency | ms | - | - | - | < 300 | - | `E2-P2A-DISC1` |
| P2 | `GetUserStatus()` API round-trip | ms | - | - | - | < 150 | - | `E2-P2B-API1` |
| P2 | `AuthLogout()` API round-trip | ms | - | - | - | < 150 | - | `E2-P2B-API2` |
| P2 | Re-Auth trigger round-trip | ms | - | - | - | < 500 | - | `E2-P2C-REAUTH1` |

## B3 - P3 Benchmarks

| Phase | Metric | Unit | Baseline | Latest | Final | Target | Delta | Evidence |
|-------|--------|------|----------|--------|-------|--------|-------|----------|
| P3 | Chrome profile scan duration | ms | - | - | - | < 250 | - | `E3-P3A-SCAN1` |
| P3 | Full 4-step hot-swap execution latency | ms | - | - | - | < 1000 | - | `E3-P3C-SW1` |
| P3 | State machine accuracy | % | 100% | - | - | 100% | - | `E3-P3B-STATE1` |

## B4 - P4 Benchmarks

| Phase | Metric | Unit | Baseline | Latest | Final | Target | Delta | Evidence |
|-------|--------|------|----------|--------|-------|--------|-------|----------|
| P4 | UI render and refresh latency | ms | - | - | - | < 100 | - | `E4-P4A-UI1` |
| P4 | Hot-swap UI click response latency | ms | - | - | - | < 50 | - | `E4-P4B-ACT1` |

## B5 - P5 Benchmarks

| Phase | Metric | Unit | Baseline | Latest | Final | Target | Delta | Evidence |
|-------|--------|------|----------|--------|-------|--------|-------|----------|
| P5 | Total test pass count | tests | 6 | 68 | 68 | >= 12 | +62 | `E5-P5A-UNIT1`, `E-PNC-CLOSE1` |
| P5 | Total test execution duration | seconds | 0.174 | 28.174 | 28.174 | < 35.000 | +28.000 | `E5-P5B-REG1`, `E-PNC-CLOSE1` |
| P5 | Test pass rate | % | 100% | 100% | 100% | 100% | 0% | `E5-P5B-REG1`, `E-PNC-CLOSE1` |
| P5 / Pn | Sequential batch pass stability (5 runs) | batches | - | 5/5 [0,0,0,0,0] | 5/5 [0,0,0,0,0] | 5/5 | 0 | `E-PNC-CLOSE1` |
| P5 / Pn | Test failures / flakes | failures | 0 | 0 | 0 | 0 | 0 | `E-PNC-CLOSE1` |
| Pn | Dead code / obsolete modules purged | count | - | 6 | 6 | >= 1 | 0 | `E-PNB-CLEAN1` |
| Pn | Worktree clean status | % | - | 100% | 100% | 100% | 0% | `EC-PNC-TREE1` |

## B6 - P6 Benchmarks

| Phase | Metric | Unit | Baseline | Latest | Final | Target | Delta | Evidence |
|-------|--------|------|----------|--------|-------|--------|-------|----------|
| P6-A | Win32 Credential round-trip latency | ms | 35.0 (Python ctypes) | 30.0 | - | < 50.0 | -5.0 | `E6-P6A-WIN1` |
| P6-A | Atomic JSON store write & sync latency | ms | 25.0 (Python) | 20.0 | - | < 30.0 | -5.0 | `E6-P6A-STORE1` |
| P6-A | Go primitives & Win32 test pass count | tests | 0 | 10 | 10 | >= 8 | +10 | `E6-P6A-WIN1`, `E6-P6A-STORE1` |
| P6-A | Go primitives test pass rate | % | - | 100% | 100% | 100% | 0% | `E6-P6A-WIN1`, `E6-P6A-STORE1` |
| P6-B | Chrome profile discovery scan latency (Go) | ms | 120.0 (Python) | 20.0 | - | < 50.0 | -100.0 | `E6-P6B-SCAN1` |
| P6-B | Language server discovery & GetUserStatus latency | ms | 850.0 (Python) | 470.0 | - | < 1000.0 | -380.0 | `E6-P6B-CONN1` |
| P6-B | Scanner & Connector test pass count | tests | 0 | 13 | 13 | >= 10 | +13 | `E6-P6B-SCAN1`, `E6-P6B-CONN1` |
| P6-B | Scanner & Connector test pass rate | % | - | 100% | 100% | 100% | 0% | `E6-P6B-SCAN1`, `E6-P6B-CONN1` |
| P6-C | Multi-Account Orchestrator & UI test pass count | tests | 0 | 17 | 17 | >= 10 | +17 | `E6-P6C-MGR1`, `E6-P6C-UI1` |
| P6-C | Multi-Account Orchestrator & UI test pass rate | % | - | 100% | 100% | 100% | 0% | `E6-P6C-MGR1`, `E6-P6C-UI1` |
| P6-C | Total Go test suite pass count (`internal/...`) | tests | 0 | 35 | 35 | >= 25 | +35 | `E6-P6C-MGR1`, `E6-P6C-UI1` |
| P6-C | Total Go test suite pass rate | % | - | 100% | 100% | 100% | 0% | `E6-P6C-MGR1`, `E6-P6C-UI1` |
| P6 | Standalone binary size (`antigravity-switch.exe`) | MB | 50.0 (Python env) | - | - | <= 15.0 | - | `E6-P6D-BIN1` |
| P6 | CLI cold startup latency | ms | 450 (Python) | - | - | <= 10 | - | `E6-P6D-PERF1` |
| P6 | GUI cold start latency | ms | 1200 (Tkinter) | - | - | <= 50 | - | `E6-P6C-UI1` |
| P6 | Resident memory footprint (daemon/CLI) | MB | 45.0 (Python) | - | - | <= 12.0 | - | `E6-P6D-PERF1` |
| P6 | Resident memory footprint (Fyne GUI) | MB | 65.0 (Tkinter) | - | - | <= 20.0 | - | `E6-P6C-UI1` |
| P6 | Idle CPU usage | % | 0.20% | - | - | < 0.05% | - | `E6-P6D-PERF1` |
| P6 | External runtime dependencies (Python/MSVC) | count | 2 | - | - | 0 | - | `E6-P6D-BIN1` |

## B7 - P7 Benchmarks

| Phase | Metric | Unit | Baseline | Latest | Final | Target | Delta | Evidence |
|-------|--------|------|----------|--------|-------|--------|-------|----------|
| P7 | Quota error detection latency (goroutine sensor) | ms | - | - | - | <= 20 | - | `E7-P7A-TAIL1` |
| P7 | Anti-flapping minimum dwell time gate | seconds | - | - | - | >= 15 | - | `E7-P7B-HYST1` |
| P7 | Quota Storm stress rotation pass rate (3+ accounts)| % | - | - | - | 100% | - | `E7-P7D-STORM1` |
| P7 | Total automated failover latency | seconds | - | - | - | < 1.0 | - | `E7-P7D-LAT1` |
| P7 | IDE editor context preservation (tabs, buffers, PTY)| % | - | - | - | 100% | - | `E7-P7D-LAT1` |

## Non-Benchmarkable Notes

- Windows Credential Manager storage capacity and DPAPI entropy are governed by OS policy and are not dynamically benchmarked.
- UI look and feel (Dark mode aesthetics, card margins, and font crispness) are evaluated via visual inspection rather than scalar numerical benchmarks.
