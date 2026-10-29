# Phase 2 Codex Benchmark Ledger

## Metadata
Date: 2026-10-29. Companions: same-prefix plan, evidence and actual-status.
## Benchmark Rules
Measured counts only; validation outcomes in evidence.
## B0 - P0 Benchmarks
| Metric | Baseline | Target | Evidence |
|---|---|---|---|
| Antigravity Go files | 53 | preserve 53 | E0-P0A-SRC1 |
| Codex Go files | 0 | 53 | E0-P0A-SRC1 |
| Engine subpackages | 9 | mirror 9 | E0-P0A-SRC1 |
| Sidebar tabs | 1 | 2 | E0-P0A-SRC1 |
## B1 - P1 Benchmarks
| Metric | Final | Evidence |
|---|---|---|
| Codex Go files | 53 | E1-P1A-TEST1 |
| Codex subpackages | 9 | E1-P1A-BUILD1 |
| Sidebar tabs | 2 | E1-P1A-TEST1 |
| GUI binary bytes | 29313024 | E1-P1A-BUILD1 |
| GUI compilation milliseconds | 7324 | E1-P1A-BUILD1 |
| UI package test seconds | 35.442 | E1-P1A-BUILD1 |
## Non-Benchmarkable Notes
No performance target or backend implementation in Phase 2.
