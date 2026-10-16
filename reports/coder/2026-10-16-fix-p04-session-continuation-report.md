# Coder Report: Active Threads Snapshot & Session Continuation Implementation (P0-04 Parity Fix)

## 1. Metadata
- **Date**: 2026-10-16
- **Task ID**: P0-04
- **Author**: Coder Lane
- **Target Invariant**: Inherit Python's active threads snapshot (`capture_active_threads_snapshot`) and in-flight session continuation (`SmartFocusConversation` -> `LoadTrajectory` -> `SendUserCascadeMessage`) on language server hot-swap.
- **Authority / SSOT**:
  - `core/session_reloader.py:818-990` (`capture_active_threads_snapshot`, `get_default_threads_history_dir`, helpers)
  - `core/state_resolver.py:560-640` (Pure native Connect-RPC Continuation: re-probe new LS -> SmartFocus -> LoadTrajectory -> SendUserCascadeMessage)
  - `reports/supervisor/2026-10-16-phase6-python-go-full-parity-zero-trust-review.md` (§3.3, §5.4)

---

## 2. Invariant Family Map
- **Family Name**: SessionContinuationAndActiveThreadsSnapshotParity
- **SSOT**: `core/session_reloader.py:818-990`, `core/state_resolver.py:560-640`, Supervisor Zero-Trust Review (§3.3, §5.4).
- **Broken Surface (Identified)**:
  - In Go, `internal/connector/api_client.go` had implemented Connect-RPC endpoints (`SmartFocusConversation`, `LoadTrajectory`, `SendUserCascadeMessage`), but they were never called during account hot-swap (`internal/manager/account_manager.go`), rendering them dead code.
  - Go completely lacked the active threads snapshot mechanism to read `aux-pane-v2-session.conversationPanes` from `app_storage.json` and persist snapshots to `accounts_store/active_threads_history/<timestamp>_active_threads.json`.
  - Switching accounts resulted in abandoned developer cascade sessions, lost focus, and interrupted in-flight tasks.
- **Sibling Runtime Surfaces Inspected & Fixed**:
  - `internal/reloader/active_threads.go`:
    + Implemented `CaptureActiveThreadsSnapshot(storagePath, brainDir, outputDir string) (string, []ActiveThread, error)`.
    + Implemented `GetDefaultThreadsHistoryDir() string` using `store.GetDefaultStoreDir()` ensuring paths are anchored to repository root `accounts_store/active_threads_history`.
    + Implemented `defaultCaptureActiveThreadsSnapshot` with full Python parity:
      * Parses `aux-pane-v2-session.conversationPanes` (supporting map, list, and string-encoded JSON formats).
      * Excludes `file__` prefixes and `fileView` tabs (`isFileViewTab`).
      * Resolves workspace project via `resolveThreadProject` and `extractProjectFromText` (parsing metadata, transcripts, URI unescaping, Active Workspaces, and Cwd keys).
      * Identifies `is_main_pane` and verifies thread existence against `brainDir` when configured.
      * Sorts active threads with `is_main_pane == true` first, followed by alphabetical `thread_id`.
      * Writes timestamped JSON atomically via `.tmp` file and rename.
  - `internal/reloader/continuation.go`:
    + Implemented `PerformSessionContinuation` and `RestoreActiveSessions` on `*SessionReloader` and package level.
    + Uses `DefaultContinuationPrompt` matching Python `core/state_resolver.py:42`: `"Continue if you still have unfinished work in progress. If all tasks are already completed, reply with: Currently there is no work to do."`.
    + Connects to newly respawned Language Server via Connect-RPC:
      1. `SmartFocusConversation(ctx, threadID)`
      2. `LoadTrajectory(ctx, threadID)`
      3. `SendUserCascadeMessage(ctx, threadID, prompt, "", nil)`
    + Implements 3-second pacing delay between consecutive threads to avoid overloading the language server.
    + Resilient error handling ensures RPC warnings on individual threads do not abort the hot-swap process.
  - `internal/reloader/session_reloader.go`:
    + Added `BrainDir`, `OutputDir`, `SnapshotCapturer`, and `ContinuationRunner` fields to `SessionReloader` for zero-side-effect testability.
    + Added `CaptureActiveThreadsSnapshot` methods.
  - `internal/manager/account_manager.go`:
    + Updated `SwitchResult` to include `ActiveThreads []reloader.ActiveThread`.
    + Added `SetReloader(rel *reloader.SessionReloader)`.
    + Integrated two-stage workflow into `SwitchAccount`:
      * **Step 2 (Before Switch)**: Capture active threads snapshot via `rel.CaptureActiveThreadsSnapshot("", "", "")`.
      * **Step 11 (After LS Recycle)**: Execute session continuation on fresh client via `rel.PerformSessionContinuation(ctx, newClient, activeThreads, "")`.
  - `internal/reloader/active_threads_test.go`:
    + Added `TestCaptureActiveThreadsSnapshot_DictFormat`: tests map-based panes, fileView filtering, main pane sorting, and JSON persistence.
    + Added `TestCaptureActiveThreadsSnapshot_ListFormat`: tests slice-based panes.
    + Added `TestCaptureActiveThreadsSnapshot_BrainDirFiltering`: tests filtering threads against brain directory.
    + Added `TestExtractProjectFromText`: tests all regex patterns (Active Workspaces, Command Working Directory, Cwd JSON, file URI).
  - `internal/reloader/continuation_test.go`:
    + Added `TestPerformSessionContinuation_HappyPath`: verifies SmartFocus, LoadTrajectory, SendUserCascadeMessage, and pacing sleep.
    + Added `TestPerformSessionContinuation_ResilienceToErrors`: verifies non-fatal behavior on RPC failures.
  - `internal/manager/account_manager_test.go`:
    + Added `TestSwitchAccount_WithSessionContinuation`: verifies end-to-end integration of pre-recycle snapshot and post-recycle continuation during `SwitchAccount`.
- **Forbidden Legacy Fallback**: Zero keyboard emulation, zero stale language server info reuse.
- **Residual Unverified Surfaces**: `none`

---

## 3. Verification Evidence

### 3.1. Unit Test Verification (`internal/reloader` & `internal/manager`)
```powershell
go test -v ./internal/reloader/... ./internal/manager/...
```
Output:
```
=== RUN   TestCaptureActiveThreadsSnapshot_DictFormat
--- PASS: TestCaptureActiveThreadsSnapshot_DictFormat (0.03s)
=== RUN   TestCaptureActiveThreadsSnapshot_ListFormat
--- PASS: TestCaptureActiveThreadsSnapshot_ListFormat (0.03s)
=== RUN   TestCaptureActiveThreadsSnapshot_BrainDirFiltering
--- PASS: TestCaptureActiveThreadsSnapshot_BrainDirFiltering (0.06s)
=== RUN   TestExtractProjectFromText
=== RUN   TestExtractProjectFromText/Active_Workspaces_pattern
=== RUN   TestExtractProjectFromText/Command_Working_Directory_pattern
=== RUN   TestExtractProjectFromText/JSON_Cwd_pattern
=== RUN   TestExtractProjectFromText/File_URI_pattern
--- PASS: TestExtractProjectFromText (0.00s)
    --- PASS: TestExtractProjectFromText/Active_Workspaces_pattern (0.00s)
    --- PASS: TestExtractProjectFromText/Command_Working_Directory_pattern (0.00s)
    --- PASS: TestExtractProjectFromText/JSON_Cwd_pattern (0.00s)
    --- PASS: TestExtractProjectFromText/File_URI_pattern (0.00s)
=== RUN   TestPerformSessionContinuation_HappyPath
--- PASS: TestPerformSessionContinuation_HappyPath (0.00s)
=== RUN   TestPerformSessionContinuation_ResilienceToErrors
--- PASS: TestPerformSessionContinuation_ResilienceToErrors (0.00s)
=== RUN   TestFindAntigravityPIDs
--- PASS: TestFindAntigravityPIDs (0.01s)
=== RUN   TestFindAntigravityWindows
--- PASS: TestFindAntigravityWindows (0.01s)
=== RUN   TestEnsureAntigravityForeground
--- PASS: TestEnsureAntigravityForeground (0.01s)
=== RUN   TestPrePatchToSAcceptance
--- PASS: TestPrePatchToSAcceptance (0.03s)
=== RUN   TestPrepatchToS
--- PASS: TestPrepatchToS (0.00s)
=== RUN   TestRecycleLanguageServer_Success
--- PASS: TestRecycleLanguageServer_Success (0.00s)
=== RUN   TestRecycleLanguageServer_DiscovererFails
--- PASS: TestRecycleLanguageServer_DiscovererFails (0.00s)
=== RUN   TestRecycleLanguageServer_ContextCancelled
--- PASS: TestRecycleLanguageServer_ContextCancelled (0.00s)
PASS
ok  	antigravity-switch/internal/reloader	1.890s
=== RUN   TestExtractEmailFromToken
--- PASS: TestExtractEmailFromToken (0.00s)
=== RUN   TestCaptureCurrentSession
--- PASS: TestCaptureCurrentSession (0.02s)
=== RUN   TestGetCurrentActiveEmail
--- PASS: TestGetCurrentActiveEmail (0.00s)
=== RUN   TestGetAccountStates
--- PASS: TestGetAccountStates (0.04s)
=== RUN   TestSwitchAccount
--- PASS: TestSwitchAccount (1.91s)
=== RUN   TestSwitchAccount_WithSessionContinuation
--- PASS: TestSwitchAccount_WithSessionContinuation (0.04s)
PASS
ok  	antigravity-switch/internal/manager	3.682s
```

### 3.2. Repository-Wide Test Suite
```powershell
go test ./...
```
Output:
```
ok  	antigravity-switch/internal/connector	(cached)
ok  	antigravity-switch/internal/manager	4.612s
ok  	antigravity-switch/internal/reloader	2.199s
ok  	antigravity-switch/internal/scanner	(cached)
ok  	antigravity-switch/internal/store	4.653s
ok  	antigravity-switch/internal/ui	1.192s
ok  	antigravity-switch/internal/wincred	(cached)
```

### 3.3. Full Windows Build Verification
```powershell
powershell -File scripts/build_windows.ps1 -Mode all
```
Output:
```
[2/4] Running pre-flight tests & vet...
  [PASS] Pure Go verification succeeded.
  [PASS] GUI verification succeeded.
[3/4] Compiling target binaries...
  [SUCCESS] Compiled PE resource object: cmd\switcher\rsrc_windows_amd64.syso (12552 bytes)
  [SUCCESS] Created: bin\antigravity-switch.exe
  Compile Duration: 2555 ms | Size: 6.73 MB (7058944 bytes)
  [SUCCESS] Created: bin\antigravity-switch-gui.exe
  Compile Duration: 5316 ms | Size: 23.54 MB (24683520 bytes)
[4/4] Executing quick verification...
  [PASS] Embedded PE application icon verified on bin\antigravity-switch.exe (32x32)
  [PASS] Embedded PE application icon verified on bin\antigravity-switch-gui.exe (32x32)
================================================================================
⚡ BUILD SUCCESSFUL (Exit Code: 0)
================================================================================
```

---

## 4. Definition of Done Checklist
- [x] Exact SSOT identified and mapped (`core/session_reloader.py:818-990`, `core/state_resolver.py:560-640`).
- [x] Invariant family closed across all sibling runtime surfaces (`reloader`, `manager`).
- [x] Pure Go unit tests implemented for every new surface (snapshot extraction, sorting, atomic JSON write, project resolution, continuation RPC dispatch, pacing delay, and error resilience).
- [x] Full build script `scripts/build_windows.ps1 -Mode all` succeeds with exit code 0.
- [x] All docs and Go files verified UTF-8 without BOM.
- [x] Residual unverified surfaces: `none`.
