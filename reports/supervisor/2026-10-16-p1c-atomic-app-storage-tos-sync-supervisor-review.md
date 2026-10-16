# Supervisor Report: Slice 3 (P1-C) Atomic app_storage.json & Session Synchronization Zero-Trust Acceptance Review

Verdict: PASS

## Metadata
- Report file: reports/supervisor/2026-10-16-p1c-atomic-app-storage-tos-sync-supervisor-review.md
- Review time: 2026-10-16 19:43:00 Asia/Bangkok
- Reviewer: supervisor (gpt-6-astra)
- Repo/project: antigravity_auto_switch
- Scope reviewed:
  - `internal/reloader/session_reloader.go` (`SyncAppStorageToS`, atomic file write, `tmpFile.Sync()`, Windows sharing violation retry loop, JSON key preservation)
  - `internal/manager/account_manager.go` (Pre-WinCred sync step 5.1, dynamic `isGcpTos` derivation)
  - `internal/reloader/reloader_test.go` (6 unit tests for atomic sync, key preservation, both ToS states, error recovery, and temp cleanup)
  - `internal/manager/account_manager_test.go` (`TestSwitchAccount_AtomicAppStorageSync` verifying pre-WinCred write synchronization)
  - `reports/coder/2026-10-16-p1c-atomic-app-storage-tos-sync-report.md`
  - Commit `265b69f`
- Claim reviewed: Slice 3 (P1-C) eliminates Google Cloud ToS modal prompts across all account switches (Metric B-M04 = 0) by atomically synchronizing `%APPDATA%\Antigravity\app_storage.json` (`jetski.onboarding.lastLoginUsername` and `jetski.onboarding.lastLoginIsGcpTos`) strictly BEFORE writing tokens to Windows Credential Manager and BEFORE language server reload, preserving 100% of other user configuration keys, handling Windows sharing violations with retry backoff, and passing all unit test suites and Windows build automation cleanly.
- Authority used:
  - Implementation Plan: `docs/plans/2026-10-16-antigravity-restart-and-tos-fix/2026-10-16-antigravity-restart-and-tos-fix-plan.md` (§Slice 3 P1-C)
  - Root Cause Analysis: `reports/architect/2026-10-16-antigravity-restart-and-tos-modal-rca.md` (§3.5, §7 Trụ cột 3)
  - Coder Report: `reports/coder/2026-10-16-p1c-atomic-app-storage-tos-sync-report.md`
  - Commit `265b69f`
  - Supervisor Skill: `g:\antigravity_auto_switch\.agents\skills\supervisor\SKILL.md`
- Related artifacts:
  - Plan: `docs/plans/2026-10-16-antigravity-restart-and-tos-fix/2026-10-16-antigravity-restart-and-tos-fix-plan.md`
  - Evidence: `docs/plans/2026-10-16-antigravity-restart-and-tos-fix/2026-10-16-antigravity-restart-and-tos-fix-evidence.md` (E3-P1C)

## Executive Summary
- Problem: During account switching, Antigravity's Electron/React renderer reads `%APPDATA%\Antigravity\app_storage.json` to determine onboarding state and ToS acceptance. If this file was updated after credentials were written or lacked the proper `jetski.onboarding.lastLoginIsGcpTos` value matching the account's authentication method, a Google Cloud Terms of Service modal prompt was triggered.
- Decision: PASS. Zero-trust code inspection, fresh test execution, and full repository verification confirm that commit `265b69f` implements robust atomic write semantics with `tmpFile.Sync()` (fsync), implements Windows sharing violation retry backoff, dynamically derives `isGcpTos` from `targetToken.AuthMethod == "gcp"`, preserves 100% of other JSON settings, and enforces pre-WinCred write execution order.
- Required outcome: Accepted. Ready for Mechanical Planner to proceed to next milestones.

## Source-Level Clearance Notes

- `internal/reloader/session_reloader.go`: CLEAR
  - Lines 292-373: Implements `SyncAppStorageToS(targetEmail string, isGcpTos bool, storagePath string) error`.
  - Input validation: Trims email and returns `ErrInvalidEmail` on empty strings.
  - Path resolution: Resolves `storagePath` or falls back to `GetDefaultAppStoragePath()`.
  - Self-healing recovery: Reads existing file; if missing, empty, or corrupted JSON, initializes a fresh map instead of failing.
  - Key fidelity: Explicitly updates:
    - `data["jetski.onboarding.lastLoginUsername"] = normEmail`
    - `data["jetski.onboarding.lastLoginIsGcpTos"] = strconv.FormatBool(isGcpTos)`
    Preserves all other configuration keys (layout, zoomLevel, conversations, custom settings).
  - Atomic write & fsync: Creates temp file `app_storage_tmp_*.json` in the same directory, writes indented JSON, calls `tmpFile.Sync()` to flush buffers to disk, and closes the descriptor before renaming.
  - Concurrency & sharing violation resilience: Retries `os.Rename` up to 5 times with 50ms backoff on `ERROR_SHARING_VIOLATION`, with a final fallback removal before renaming. Temporary files are cleaned up on error.
  - Method delegation: `(r *SessionReloader) SyncAppStorageToS(targetEmail string, isGcpTos bool)` correctly wraps package function using `r.StoragePath`.
  - Backwards compatibility: `PrepatchToS` and `PrePatchToSAcceptance` delegate cleanly to `SyncAppStorageToS`.
- `internal/manager/account_manager.go`: CLEAR
  - Lines 460-470: Step 5.1 in `SwitchAccount` invokes `reloader.SyncAppStorageToS(normTarget, isGcpTos, storagePath)`.
  - Correct execution order: Executed strictly BEFORE Step 6 (`writer(targetToken)`).
  - Dynamic GCP ToS derivation: `isGcpTos := (targetToken != nil && strings.EqualFold(targetToken.AuthMethod, "gcp"))`.
  - Redundant legacy Step 7 removed cleanly without dead code.
- `internal/reloader/reloader_test.go`: CLEAR
  - 6 dedicated unit test functions cover:
    1. Key preservation (`TestSyncAppStorageToS_PreservesOtherKeys`).
    2. Both ToS states (`TestSyncAppStorageToS_BothGcpTosStates`: `true` for GCP, `false` for consumer).
    3. Error recovery (`TestSyncAppStorageToS_RecoveryFromMissingAndCorrupted`: missing file, corrupted JSON, empty file).
    4. Atomic temp file cleanup (`TestSyncAppStorageToS_AtomicWriteAndCleanup`: 0 leftover temp files).
    5. Input validation (`TestSyncAppStorageToS_InvalidEmail`).
    6. Instance method delegation (`TestSessionReloader_SyncAppStorageToS_Method`).
- `internal/manager/account_manager_test.go`: CLEAR
  - `TestSwitchAccount_AtomicAppStorageSync` intercepts disk state during `wincredWriter`, verifying that `app_storage.json` was already flushed and updated before credentials were written, for both GCP and Consumer profiles.

## Evidence Checked

### Passed:
1. Package Unit Tests:
   - Command: `go test -v ./internal/reloader/... ./internal/manager/...`
   - Result: 100% PASS across all 35 test cases (reloader ok, manager 10.726s).
   - Verification Freshness: Fresh.
2. Full Repository Regression Suite:
   - Command: `go test ./...`
   - Result: 100% PASS across all packages (`connector`, `manager`, `model`, `reloader`, `scanner`, `store`, `ui`, `wincred`).
   - Verification Freshness: Fresh.
3. Windows Build Automation:
   - Command: `powershell -File scripts/build_windows.ps1`
   - Result:
     - Pure Go verification: PASS
     - Resource compilation (`windres`): SUCCESS (`rsrc_windows_amd64.syso`, 12552 bytes)
     - Standalone CLI Binary compilation: SUCCESS (`bin/antigravity-switch.exe`, 6.75 MB)
     - PE Icon verification: PASS on `bin/antigravity-switch.exe` and `bin/antigravity-switch-gui.exe`
     - Quick verification: `bin/antigravity-switch.exe --status` executed cleanly and reported Language Server status (PID: 5904 | Port: 50915).
   - Verification Freshness: Fresh.
4. Dead Code & Sibling Surface Sweep:
   - Confirmed 0 dead functions, no leftover debug prints, and complete closure of the ToS synchronization invariant.

### Failed:
- None.

### Not Run:
- None.

## Invariant Closure
- Affected Invariant: `AtomicAppStorageSynchronizationAndZeroToSModal`
- Sibling Surfaces Checked:
  - `internal/reloader/session_reloader.go`: Implemented with atomic rename and fsync.
  - `internal/manager/account_manager.go`: Enforces pre-WinCred write ordering.
  - `internal/ui/app.go`: Inherits `SwitchAccount` logic transparently.
- Residual Unverified Same-Invariant Surfaces: None.

## Overall Evaluation
The implementation delivered in commit `265b69f` represents a complete, robust, and elegant resolution of the Google ToS modal defect. The atomic write mechanism incorporates industry-standard file-safety practices (same-directory temp file, fsync before close, retry backoff for Windows sharing violations), guarantees 100% preservation of unrelated user configurations, and strictly enforces the pre-WinCred execution order required by Antigravity's Electron runtime.

Verdict: **PASS**
Ready for Mechanical Planner to proceed to next milestones.
