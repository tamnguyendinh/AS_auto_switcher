# Supervisor Report: Phase 3 Add Account Modal UI Widget Acceptance Review

Verdict: PASS

## Metadata
- Report file: `reports/supervisor/2026-10-22-phase3-add-account-modal-acceptance.md`
- Review time: 2026-10-22 18:42:00 +07:00
- Reviewer: supervisor
- Repo/project: antigravity-switch
- Scope reviewed: Commit `7d2eb40` (`internal/ui/add_modal.go`, `internal/ui/add_modal_test.go`, `reports/coder/2026-10-22-phase3-add-account-modal-report.md`)
- Claim reviewed: Phase 3 (Add Account Modal UI Widget) correctly constructs the dialog modal (460x340) with scrollable candidate list (min 420x260), enforces full nil-safety on `ShowAddAccountModal()`, displays italic empty label when candidate list is empty, handles 3 distinct candidate states (`[✔ Đã OAuth]`, `[Cần OAuth]` + ADD button, `[Đã add]`), executes in-place reaction upon ADD click and reactively refreshes main UI overview, supports on-demand Chrome profile scan with busy indicator ("⏳ Đang quét..."), respects email privacy masking via `MaskEmail` and `IsMaskAccounts()`, preserves 0 diff on `scripts/build_windows.ps1`, `README.md`, and `internal/ui/window.go`, passes all unit tests (100%), and cleanly compiles with `pwsh -File .\scripts\build_windows.ps1 -Clean` (Exit Code 0).
- Authority used: `reports/architect/2026-10-22-add-account-modal-and-discovered-filtering-spec.md` (SPEC-ARCH-20261022-ADD-ACCOUNT-MODAL-AND-DISCOVERED-FILTERING-SPEC, commit a77efb9 § Phần IV.3, V Bước 3), CEO mandate, supervisor skill, AGENTS.md.
- Related artifacts:
  - Architecture Spec: `reports/architect/2026-10-22-add-account-modal-and-discovered-filtering-spec.md`
  - Coder Report: `reports/coder/2026-10-22-phase3-add-account-modal-report.md`
  - Commits: `a77efb9`, `15a0879`, `0cb9d01`, `880f01d`, `cee71b5`, `7d2eb40`

## Executive Summary
- Problem: As specified in the architecture blueprint, candidate accounts discovered from Chrome profiles must be managed through an explicit modal dialog ("Thêm tài khoản") rather than being automatically loaded onto the main dashboard. Phase 3 implements this modal widget, providing Chrome scanning on-demand, candidate display with 3-tier status hierarchy, email masking compliance, and immediate reactive addition of candidates to the primary UI dashboard upon clicking ADD.
- Decision: PASS. Zero-trust source code inspection, test suite execution, clean Windows build script execution, and git diff analysis confirm strict adherence to all architectural invariants and UX requirements.
- Required outcome: Accepted. Ready for Phase 4 (Toolbar integration in `internal/ui/window.go` and `app.go`).

## Source-Level Clearance Notes

- `internal/ui/add_modal.go` (`ShowAddAccountModal` & `buildAddAccountModalContent`): **CLEAR**
  - **Nil Safety (Lines 126-129):** Guards against nil receiver, nil window, or nil manager (`if a == nil || a.win == nil || a.mgr == nil { return }`), preventing any runtime panics.
  - **Modal Sizing & Structure (Lines 16-20, 131-135):** Uses `dialog.NewCustom("Thêm tài khoản", "Đóng", contentBox, a.win)` with explicit dimensions `460x340` and scrollable content viewport minimum `420x260`.
  - **Empty State (Lines 28-33):** When candidate pool is empty, renders center-aligned italic label `"Không có tài khoản nào trong danh bạ."`.
  - **3-Tier Candidate State Hierarchy (Lines 44-88):**
    1. `candItem.HasToken`: Renders `[✔ Đã OAuth]` badge (`palette.Success`, bold, 11pt, centered in `actionSlot`). No ADD button.
    2. `!candItem.HasToken && !candItem.Added`: Renders `[Cần OAuth]` badge (`palette.Warning`, bold, 11pt) alongside `ADD` button (`HighImportance`, `theme.ContentAddIcon()`, wrapped in `compactButton`).
       - On click: invokes `a.mgr.AddDiscoveredAccount(candItem.Email)`.
       - In-place mutation: immediately mutates `actionSlot` to display `[Đã add]` (`palette.TextMuted`, bold, 11pt) and eliminates the ADD button without disrupting viewport scroll position.
       - Reactive UI update: launches goroutine to query `a.mgr.GetAccountStates(context.Background())`, updates `a.overview` under lock, and calls `fyne.Do(func() { a.updateUI(overview) })` to render the newly added account card onto the main dashboard.
    3. `!candItem.HasToken && candItem.Added`: Renders `[Đã add]` badge (`palette.TextMuted`, bold, 11pt, centered in `actionSlot`). No ADD button.
  - **On-Demand Chrome Profile Scan (Lines 96-107):**
    - Renders `"Quét Chrome"` button (`theme.ViewRefreshIcon()`, wrapped in `compactButton`).
    - On click: disables button immediately, changes text to `"⏳ Đang quét..."`, executes `a.mgr.ScanChromeAccounts()` asynchronously in a background goroutine.
    - Upon completion: dispatches via `fyne.Do` to reload candidates (`reloadCandidates()`), restores button label to `"Quét Chrome"`, and re-enables button.
  - **Privacy Masking (Lines 37-42):** Checks `a.IsMaskAccounts()`, applying `MaskEmail(candItem.Email)` if enabled, and preserves `TextTruncateEllipsis` for overflow protection.
  - **Theme Adaptability (Line 16):** Reads active theme palette via `GetThemePalette(a.currentTheme)` ensuring full compliance across Dark, Light, and Warm themes.

- `internal/ui/add_modal_test.go`: **CLEAR**
  - Implements 7 targeted, deterministic unit tests verifying the full behavior contract:
    - `TestShowAddAccountModal_NilSafety`: Verifies nil receiver, nil window, and nil manager cases without panic.
    - `TestShowAddAccountModal_EmptyCandidates`: Verifies empty state italic text.
    - `TestShowAddAccountModal_ThreeCandidateStates`: Verifies simultaneous presence of all 3 status tiers (`[✔ Đã OAuth]`, `[Đã add]`, and `[Cần OAuth]` + ADD button with `HighImportance`).
    - `TestShowAddAccountModal_AddActionInPlaceMutationAndOverviewUpdate`: Verifies in-place replacement of action slot, store persistence (`Added: true`), and async appearance on main UI overview.
    - `TestShowAddAccountModal_EmailMasking`: Verifies masked output when `SetMaskAccounts(true)` and raw email when disabled.
    - `TestShowAddAccountModal_ScanChromeButtonBehavior`: Verifies disable/busy text transition and async re-enable/restore.
    - `TestShowAddAccountModal_DialogDisplayAndOverlay`: Verifies Fyne canvas dialog overlay creation and removal.

- Scope Discipline & Preservation: **CLEAR**
  - `scripts/build_windows.ps1`: 0 lines diff.
  - `README.md`: 0 lines diff.
  - `internal/ui/window.go`: 0 lines diff (properly deferred to Phase 4).

## Evidence Checked

Passed:
- `git show --stat 7d2eb40`:
  - 3 files changed, 704 insertions(+) (`internal/ui/add_modal.go`, `internal/ui/add_modal_test.go`, and coder report).
  - Scope strictly confined to Phase 3 slice.
  - Verification freshness: fresh (executed on current commit `7d2eb40`).
- `git diff cee71b5 HEAD -- scripts/build_windows.ps1 README.md internal/ui/window.go`:
  - Exactly 0 lines diff across all three files.
  - Verification freshness: fresh.
- `go test -v -count=1 ./internal/ui -run TestShowAddAccountModal`:
  - 7/7 tests passed in 2.187s without caching.
  - Verification freshness: fresh.
- `go test -count=1 ./internal/ui/...`:
  - Entire UI package passed in 20.891s (background task-36).
  - Verification freshness: fresh.
- `go test -p 1 ./...`:
  - All packages PASS across entire repository (`cmd/switcher`, `internal/automation`, `internal/connector`, `internal/manager`, `internal/model`, `internal/reloader`, `internal/scanner`, `internal/store`, `internal/telemetry`, `internal/ui`, `internal/wincred`).
  - Verification freshness: fresh (background task-54).
- `pwsh -File .\scripts\build_windows.ps1 -Clean`:
  - Output: `[PASS] Verification succeeded`, `[SUCCESS] Created: bin\antigravity-switch.exe` (27.77 MB), `[PASS] Embedded PE application icon verified (32x32)`, `BUILD SUCCESSFUL`.
  - Exit code: 0 (background task-50).
  - Verification freshness: fresh.
- `git status`:
  - Working tree clean.

Failed:
- None.

Not run:
- None.

## Invariant Closure
- Affected invariant: `Add Account Modal and Candidate Pool UI Widget Invariant`
  - Invariant-Auto-Switch-Immunity: Preserved. Accounts in candidate pool remain isolated until explicitly added; adding them marks `HasToken: false`, preventing unwanted rotation.
  - Invariant-Telemetry-Loop-Safety: Preserved. Accounts displayed in modal or unauthenticated cards do not invoke Google API telemetry.
  - Invariant-Hot-Swap-Integrity: Preserved. Switch workflow remains untouched.
  - Invariant-Grid-Layout-Stability & Card-Signature-Compatibility: Preserved. UI grid scroller and card signatures remain untouched.
- Sibling surfaces checked:
  - `internal/ui/add_modal.go`
  - `internal/ui/add_modal_test.go`
  - `internal/ui/app.go`
  - `internal/manager/account_manager.go`
  - `scripts/build_windows.ps1`
  - `README.md`
- Residual unverified same-invariant surfaces: none.
