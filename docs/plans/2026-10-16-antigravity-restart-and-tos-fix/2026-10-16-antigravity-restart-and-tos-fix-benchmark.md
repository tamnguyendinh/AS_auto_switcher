# Antigravity Restart & Google Policy Modal Fix Benchmark Ledger

## Metadata

- Date: `2026-10-16`
- Status: `P0 Baseline Measured`
- Plan: `docs/plans/2026-10-16-antigravity-restart-and-tos-fix/2026-10-16-antigravity-restart-and-tos-fix-plan.md`
- Actual status: `docs/plans/2026-10-16-antigravity-restart-and-tos-fix/2026-10-16-antigravity-restart-and-tos-fix-actual-status.md`
- Evidence: `docs/plans/2026-10-16-antigravity-restart-and-tos-fix/2026-10-16-antigravity-restart-and-tos-fix-evidence.md`

## Benchmark Rules

- Record measured numbers only. Do not put command logs or narrative here.
- Baseline captures reality before implementation.
- Targets define strict acceptance bars.

## B0: Baseline Metric Matrix

| Metric ID | Metric Description | Unit | Baseline (Current) | Target | Current Status | Delta |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| `B-M01` | Real OS `taskkill` calls during test suite execution (`go test ./...`) | calls | 1 | 0 | Baseline | -1 (100% reduction) |
| `B-M02` | Expired tokens written to Windows Credential Manager | % of expired | 100% | 0% | Baseline | -100% (zero dead tokens) |
| `B-M03` | Window reloads (`window.loadURL`) triggered during account switch (when LS running) | % of switches | 100% | 0% | Baseline | -100% (zero window reload) |
| `B-M04` | Google Policy / ToS Modal dialog appearances during switch | popups | 1 per reload | 0 | Baseline | -100% (zero popup) |
| `B-M05` | Account switch execution latency (end-to-end) | ms | 3150 ms | < 500 ms | Baseline | > 80% faster |
| `B-M06` | Language Server crash-loop incident frequency | crashes/hour | 3 (cap) | 0 | Baseline | -100% (zero crashes) |

## B1: Target Measurements for Implementation Slices

### Slice 1 (P1-A): Test Suite Isolation
- Verification target: `B-M01` = 0 calls to `taskkill` when running `go test -v ./internal/reloader/...`.

### Slice 2 (P1-B): Proactive Token Refresh
- Verification target: `B-M02` = 0% expired tokens written; 100% of tokens with expiry < 5 minutes refreshed before WinCred persistence.

### Slice 3 (P1-C): Atomic app_storage.json Sync
- Verification target: `B-M04` = 0 ToS modal popups across repeated switch operations.

### Slice 4 (P1-D): Civilized RPC ReAuth
- Verification target: `B-M03` = 0% window reload when Language Server is responsive; `B-M05` < 500 ms switch latency.

