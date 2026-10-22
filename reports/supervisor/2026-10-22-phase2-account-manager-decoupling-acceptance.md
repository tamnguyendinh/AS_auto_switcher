# Supervisor Report: Phase 2 Account Manager Decoupling Acceptance Review

Verdict: PASS

## Metadata
- Report file: `reports/supervisor/2026-10-22-phase2-account-manager-decoupling-acceptance.md`
- Review time: 2026-10-22 18:25:00 +07:00
- Reviewer: supervisor
- Repo/project: antigravity-switch
- Scope reviewed: Commit `880f01d` (`internal/manager/account_manager.go`, `internal/manager/two_store_test.go`, `reports/coder/2026-10-22-phase2-account-manager-decoupling-report.md`)
- Claim reviewed: Phase 2 (Account Manager Decoupling) correctly decouples Chrome scanning from `GetAccountStates`, filters discovered records to only display accounts with `Added == true`, provides on-demand `ScanChromeAccounts()`, implements `CandidateAccountItem` and `GetCandidateAccounts()` with strict sorting (`HasToken` -> `Added` -> `Email A-Z`), implements `AddDiscoveredAccount()`, preserves 0 diff on `scripts/build_windows.ps1` and `README.md`, passes all unit tests (100%), and cleanly compiles with `pwsh -File .\scripts\build_windows.ps1 -Clean` (Exit Code 0).
- Authority used: `reports/architect/2026-10-22-add-account-modal-and-discovered-filtering-spec.md` (SPEC-ARCH-20261022-ADD-ACCOUNT-MODAL-AND-DISCOVERED-FILTERING-SPEC, commit a77efb9 § Phần IV.2, V Bước 2), CEO mandate, supervisor skill, AGENTS.md.
- Related artifacts:
  - Architecture Spec: `reports/architect/2026-10-22-add-account-modal-and-discovered-filtering-spec.md`
  - Coder Report: `reports/coder/2026-10-22-phase2-account-manager-decoupling-report.md`
  - Commits: `a77efb9`, `15a0879`, `0cb9d01`, `880f01d`

## Executive Summary
- Problem: In the legacy codebase, `GetAccountStates()` automatically scanned Chrome profiles on every app start, account switch, failover, or toolbar refresh, forcing all browser accounts onto the main dashboard as unauthenticated cards (`[Cần OAuth]`), causing visual pollution and resurrecting removed accounts. Phase 2 requires decoupling Chrome scanning into an on-demand method (`ScanChromeAccounts`), ensuring `GetAccountStates` only displays candidates explicitly added (`rec.Added == true`), and providing candidate query/enrichment APIs for the upcoming Add Account Modal.
- Decision: PASS. Zero-trust source code audit, test suite verification, build script execution, and git diff inspection confirm full adherence to the architecture specification.
- Required outcome: Accepted. Ready for Phase 3 (Implementation of Add Account Modal in `internal/ui/add_modal.go`).

## Source-Level Clearance Notes

- `internal/manager/account_manager.go` (Lines 298-387, `GetAccountStates`): **CLEAR**
  - Completely removed `cDir` extraction, `scanner.ScanChromeAccountDetails(cDir)` call, and `discoveredStore.SaveDiscovered` loop.
  - Replaced legacy candidate loading with `discovered.ListDiscoveredRecords()` and strict check `if rec.Added { add(rec.Email) }`.
  - Preserved token store loading (`m.store.ListAccounts()`), active email loading, and badge formatting (`[✔ Dùng]`, `[Chuyển]`, `[Cần OAuth]`).
  - No background I/O or Chrome scanning occurs during dashboard refresh or startup.

- `internal/manager/account_manager.go` (Lines 389-415, `ScanChromeAccounts`): **CLEAR**
  - Implements on-demand scanning method `ScanChromeAccounts() ([]string, error)` reading `m.chromeDir` and calling `scanner.ScanChromeAccountDetails(cDir)`.
  - Saves each scanned profile email via `discoveredStore.SaveDiscovered(em)`, preserving existing `Added` flags as implemented in Phase 1.
  - Fail-close defense: returns error `discovered store uninitialized` if `discoveredStore == nil`.

- `internal/manager/account_manager.go` (Lines 417-466, `CandidateAccountItem` & `GetCandidateAccounts`): **CLEAR**
  - Defines `CandidateAccountItem` with fields `Email string`, `HasToken bool`, `Added bool`.
  - Implements `GetCandidateAccounts() ([]CandidateAccountItem, error)` querying `discoveredStore.ListDiscoveredRecords()` and enriching with `m.store.ListAccounts()`.
  - Sorts strictly by precedence: `HasToken` (true first) -> `Added` (true first) -> `Email` alphabetical A-Z.

- `internal/manager/account_manager.go` (Lines 468-477, `AddDiscoveredAccount`): **CLEAR**
  - Implements `AddDiscoveredAccount(email string) error` delegating to `discoveredStore.SetDiscoveredAdded(email, true)`.
  - Fail-close defense: returns error `discovered store uninitialized` if `discoveredStore == nil`.

- `internal/manager/two_store_test.go`: **CLEAR**
  - Updated `TestTwoStoreCaptureMergeRepeatAndRemember` to explicitly invoke on-demand scanning and add flows (`m.ScanChromeAccounts()` and `m.AddDiscoveredAccount("NEW@example.com")`).
  - Updated `TestTwoStorePartialFailuresAndMissingSession` to reflect decoupled Chrome scanning behavior.
  - Added 5 new targeted unit tests:
    - `TestGetAccountStates_DoesNotScanChrome`: Confirms no Chrome files read/written and no unadded cards appear on overview.
    - `TestScanChromeAccounts_OnDemand`: Confirms profiles saved with `Added == false` and do not appear in `GetAccountStates`.
    - `TestAddDiscoveredAccount`: Confirms `AddDiscoveredAccount` transitions candidate to UI dashboard as `[Cần OAuth]` and preserves `Added` flag across re-scans.
    - `TestGetCandidateAccounts_SortingAndEnrichment`: Confirms 4-element sort order (`HasToken` -> `Added` -> A-Z).
    - `TestCandidateAccounts_UninitializedStore`: Confirms nil store error handling.

- `scripts/build_windows.ps1` & `README.md`: **CLEAR**
  - Verified with `git diff 880f01d^..880f01d -- scripts/build_windows.ps1 README.md`: exactly 0 lines diff (100% untampered).

## Evidence Checked

Passed:
- `git show --stat 880f01d`:
  - 3 files changed, 405 insertions(+), 31 deletions(-)
  - Touched only `internal/manager/account_manager.go`, `internal/manager/two_store_test.go`, and coder report.
  - Verification freshness: fresh (executed on current master commit `880f01d`).
- `git diff 880f01d^..880f01d -- scripts/build_windows.ps1 README.md`:
  - Output empty (0 lines changed).
  - Verification freshness: fresh.
- `go test -count=1 ./internal/manager/...`:
  - All 37 tests PASS (5.525s execution time).
  - Verification freshness: fresh (live run without cache).
- `go test ./...`:
  - All packages PASS across entire repository (`cmd/switcher`, `internal/automation`, `internal/connector`, `internal/manager`, `internal/model`, `internal/reloader`, `internal/scanner`, `internal/store`, `internal/telemetry`, `internal/ui`, `internal/wincred`).
  - Verification freshness: fresh (executed as background task-36, exit code 0).
- `pwsh -File .\scripts\build_windows.ps1 -Clean`:
  - Output: `[PASS] Verification succeeded`, `[SUCCESS] Created: bin\antigravity-switch.exe` (27.77 MB), `[PASS] Embedded PE application icon verified (32x32)`, `BUILD SUCCESSFUL`.
  - Exit code: 0.
  - Verification freshness: fresh (executed as background task-42, exit code 0).
- `git status`:
  - Working tree clean.

Failed:
- None.

Not run:
- None.

## Invariant Closure
- Affected invariant: `Account Manager Chrome-Decoupling & Discovered Filtering Invariant`
  - Invariant-Auto-Switch-Immunity: Preserved. `GetAccountStates` only returns `[Cần OAuth]` for added candidates; `HasToken` remains `false`, isolating them from automated rotation.
  - Invariant-Telemetry-Loop-Safety: Preserved. Accounts without tokens in `store` never trigger Google API quota calls.
  - Invariant-Hot-Swap-Integrity: Preserved. 9-step switch workflow remains completely intact.
  - Invariant-Grid-Layout-Stability & Card-Signature-Compatibility: Preserved. No breaking signature changes to widgets or manager constructors.
- Sibling surfaces checked:
  - `internal/manager/account_manager.go`
  - `internal/manager/two_store_test.go`
  - `internal/store/discovered_store.go`
  - `scripts/build_windows.ps1`
  - `README.md`
- Residual unverified same-invariant surfaces: none.

## Overall Evaluation
The Phase 2 implementation strictly complies with the approved architectural specification (`reports/architect/2026-10-22-add-account-modal-and-discovered-filtering-spec.md`). The decoupling of Chrome scanning resolves the root causes of UI pollution and candidate resurrection. All unit and integration test assertions pass without regression. The Windows build script succeeds with Exit Code 0. The deliverable is accepted with verdict PASS.
