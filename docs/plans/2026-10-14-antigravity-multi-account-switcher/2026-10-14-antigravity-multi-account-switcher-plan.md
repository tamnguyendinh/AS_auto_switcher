# Antigravity Multi-Account Switcher Plan

## Metadata

- Date: `2026-10-14` (Updated: `2026-10-15`)
- Status: `in_progress`
- Plan: `docs/plans/2026-10-14-antigravity-multi-account-switcher/2026-10-14-antigravity-multi-account-switcher-plan.md`
- Evidence: `docs/plans/2026-10-14-antigravity-multi-account-switcher/2026-10-14-antigravity-multi-account-switcher-evidence.md`
- Benchmark: `docs/plans/2026-10-14-antigravity-multi-account-switcher/2026-10-14-antigravity-multi-account-switcher-benchmark.md`
- Actual status: `docs/plans/2026-10-14-antigravity-multi-account-switcher/2026-10-14-antigravity-multi-account-switcher-actual-status.md`

## Goal

Re-architect and implement the Antigravity Multi-Account Switcher to provide seamless, persistent multi-account switching for Google Antigravity on Windows. The tool functions as the permanent external memory for Antigravity accounts using individual JSON token files (`accounts_store/<email>.json`), dynamically discovers accounts from Google Chrome while enforcing strict blacklist filtering, and executes the Owner's mandated 4-step closed-loop internal API switching architecture:
1. `GetUserStatus()` to synchronize live email in `language_server.exe` RAM;
2. `AuthLogout()` to cleanly purge active tokens from RAM and terminate the session of Account A (bypassed in smooth zero-chrome hot-swap);
3. Inject Account B's token into Windows Credential Manager (`LegacyGeneric:target=gemini:antigravity`);
4. Trigger Re-Auth / Reload to load Token B into RAM ("Auth succeeded: Tài khoản B").

Following the completion of the Python prototype (Phases P0–P5), the project advances into two high-performance production milestones directly derived from the approved Architect Report (`reports/architect/2026-10-14-autonomous-quota-failover-architecture.md`):
- **Phase P6 (Go Engine Port, Fyne Native GUI & Standalone Binary):** Complete port to compiled Go (Golang), featuring a 100% pure Go native GUI using Fyne (< 20MB RAM, < 50ms startup), native Win32 `Advapi32.dll` integration via `golang.org/x/sys/windows`, compiled into a single standalone Windows executable (`antigravity-switch.exe`, 10MB–15MB) with zero Python, pip, or external C++ runtime dependencies.
- **Phase P7 (Go-Native Autonomous Quota Failover Daemon):** Non-blocking background goroutine tailer on `language_server.log` detecting HTTP 429 and `RESOURCE_EXHAUSTED` in $\le 20\text{ms}$, managing an in-memory `AccountPool` with exponential backoff cooldowns, 15-second anti-flapping hysteresis gate, sub-second ($< 1.0\text{s}$) hot-swap execution, native Windows System Tray HUD with dynamic status dots, discrete Windows Toast alerts, and 100% IDE context preservation (tabs, dirty buffers, terminals unaffected).

## Rules

- Complete P0 actual status before implementation work.
- Update each checklist item immediately when it is completed.
- Record evidence as work completes.
- Record benchmarkable counts or measurements when they are taken.
- Update later phase status assumptions, next actions, and work steps when actual-status evidence changes the repo state.
- After completing a phase or implementation slice and refreshing `actual-status.md`, update the next affected phase's work steps as needed to match the latest repo reality, while preserving that phase's original goal, scope, acceptance criteria, and major phase order.
- Every implementation phase must be decomposed into multiple implementation slices that are as small as practical. A phase is a grouping and ordering container; a slice is the executable implementation unit.
- Do not implement a phase directly. Work starts from a slice ID such as `P1-A`, `P1-B`, or `P2-A`.
- Each implementation slice must include:
  + Goal
  + Scope Boundary (Editable, Inspect-only, Preserve-only, Out-of-scope)
  + Non-Goals when useful
  + Pre-flight Questions
  + Work Steps strictly following the 6-step execution lifecycle:
    1. Code: Execute leaf atomic tasks in core source files only. Each leaf atomic task must explicitly specify Action, Inputs, Allowed Edit Scope, Outputs, local Verification Condition, and a Task Checkpoint (e.g. `wip(<slice_id>): task 1.x - <verified task outcome>`; micro WIP commit for granular task progress, NOT a slice completion commit). Do not touch test files during this step.
    2. Code Inspection & Lint: Review diff, syntax, linter, typecheck, and ensure edits remain strictly within assigned boundary.
    3. Update Tests: Following Master Rule 6 (Code first, test second), add or update unit/integration tests to verify the newly implemented code behavior.
    4. Build: Run the repository's build/compilation check.
    5. QA (Mini QA): Verify user-visible runtime behavior (desktop app GUI, API response, process signals).
    6. Slice Acceptance & Official Milestone Commit: Verify slice acceptance criteria, record evidence in `evidence.md`, refresh `actual-status.md`, and create official milestone commit (e.g. `feat(<slice_id>): <slice outcome>`).
  + Implementation Gate
  + Acceptance
  + Evidence Targets
  + Actual-status Update
  + Commit Boundary
- Hidden fallback is forbidden. Prefer a visible failure over a fallback that hides a broken primary path.
- Tests must prove product behavior. Assert trigger -> process -> observable result.
- Strictly PROHIBIT coding Web/Production Auth; Prototype Auth only.
- Strictly NO ANVIEN for implementation lanes; only Planner (Strategic) and Architect lanes are authorized.
- Strictly DO NOT modify Antigravity binary executables on disk or Chrome browser SQLite databases directly.

## Problem

1. **Antigravity Credential Overwrite:** Antigravity stores only 1 active account in Windows Credential Manager under `LegacyGeneric:target=gemini:antigravity`. Whenever a user logs into a new account, Antigravity overwrites the credential, permanently discarding the previous account's OAuth tokens.
2. **RAM / Process Desynchronization:** Simply writing a new token to Windows Credential Manager does not change the running session in `language_server.exe`. The backend process keeps the old account's credentials in memory (RAM). Subsequent prompts continue to be signed with the old account or fail with quota exhaustion.
3. **Loss of Session Context:** Force-closing and restarting Antigravity destroys open chat threads, unsaved context, and IDE responsiveness.
4. **UI Inconsistency:** Without real-time synchronization with the language server's internal API, switcher tools can report an account as active when Antigravity's RAM is still bound to a different identity.
5. **Runtime Footprint & Dependencies of Interpreted Prototypes:** Python runtimes consume 40MB–80MB RAM, require a local virtual environment/Python interpreter, and impose cold start delays.
6. **Quota Bottleneck in Agentic Workflows:** Multi-turn autonomous agent sessions hit HTTP 429 / gRPC `RESOURCE_EXHAUSTED` caps frequently, shattering developer focus and requiring manual account switching.

## Scope

- **Editable Surfaces:**
  - *Python Baseline Surfaces (Phases P0–P5, preserved as completed):* `core/account_store.py`, `core/credential_manager.py`, `core/api_connector.py`, `core/session_reloader.py`, `core/state_resolver.py`, `scanner/chrome_scanner.py`, `ui/app_window.py`, `tests/test_all.py`, `main.py`.
  - *Phase P6 Go Engine Surfaces:* `go.mod`, `go.sum`, `cmd/switcher/main.go`, `internal/wincred/cred.go`, `internal/wincred/types.go`, `internal/model/types.go`, `internal/store/account_store.go`, `internal/store/sanitizer.go`, `internal/scanner/chrome_scanner.go`, `internal/connector/discovery.go`, `internal/connector/api_client.go`, `internal/connector/endpoints.go`, `internal/reloader/session_reloader.go`, `internal/manager/account_manager.go`, `internal/ui/app.go`, `internal/ui/window.go`, `internal/ui/widgets.go`, `internal/ui/tray.go`, `scripts/build_windows.ps1`, `antigravity-switch.exe`.
  - *Phase P7 Autonomous Failover Surfaces:* `internal/sensor/log_tailer.go`, `internal/sensor/signatures.go`, `internal/sensor/event_bus.go`, `internal/controller/pool_manager.go`, `internal/controller/scheduler.go`, `internal/controller/cooldown.go`, `internal/controller/hysteresis.go`, `internal/tray/tray_daemon.go`, `internal/tray/notifications.go`, `internal/tray/hotkeys.go`, `tests/e2e/quota_storm_test.go`, `tests/e2e/e2e_test.go`.
- **Inspect-Only Surfaces:**
  - `%LOCALAPPDATA%\Google\Chrome\User Data`: Chrome user profile preferences and Local State.
  - `language_server.exe` process table & TCP listening ports (`127.0.0.1:<ephemeral_port>`).
  - `%APPDATA%\Antigravity\logs\language_server.log`: Diagnostic logs confirming auth transitions and quota exhaustion events.
- **Preserve-Only Surfaces:**
  - `LegacyGeneric:target=gemini:antigravity` target naming convention.
  - Dark mode color palette and practical developer UI ergonomics.
  - Strict blacklist keyword filtering (`["anvien"]`).
  - Python prototype codebase in `core/`, `scanner/`, `ui/`, `tests/` as baseline reference.
- **Out of Scope:**
  - Antigravity IDE core binaries and language server code modifications.
  - Production OAuth server or web-based token broker.
  - Electron or WebView2 embedded runtimes (pure Go native Fyne GUI only).

## Non-Goals

- No cloud servers, web authentication flows, or external proxy servers.
- No modifying Chrome profile data or browser configurations directly.
- No patching of `language_server.exe` binary.
- No Electron, Node.js, npm, or WebView2 asset toolchains (Fyne pure Go native GUI selected via ADR-P6-05).
- No CGo requirements on Windows (pure Go via `golang.org/x/sys/windows`).
- No anvien usage during implementation.

## Requirements

1. **Persistent Multi-Account Storage:** Each discovered/authenticated account is persisted as an independent file `accounts_store/<email>.json` containing full token payloads (`access_token`, `refresh_token`, `id_token`, `expiry`, `token_type`).
2. **Chrome Discovery with Blacklist:** Discovers all Google accounts logged into Chrome profiles; automatically filters out emails matching blacklist keywords (`anvien`); automatically removes accounts logged out of Chrome.
3. **4-Step Closed-Loop Internal API Switching:**
   - Step 1: Query `GetUserStatus()` from Antigravity `language_server.exe` to retrieve the true active email in RAM.
   - Step 2: Issue `AuthLogout()` to language server API to flush active tokens from RAM and cancel Account A's session (or bypass in smooth zero-chrome hot-swap).
   - Step 3: Write Account B's JSON token to Windows Credential Manager `LegacyGeneric:target=gemini:antigravity`.
   - Step 4: Trigger Re-Auth / Reload to prompt Antigravity to load Token B into RAM, verified by `Auth succeeded`.
4. **State Machine:**
   - 🟢 `ACTIVE [ĐANG DÙNG]`: verified matching true RAM identity.
   - 🟢 `READY [⚡ Chuyển sang]`: present in Chrome AND saved in `accounts_store/<email>.json`.
   - 🟡 `UNAUTH [Chưa lưu vào kho]`: present in Chrome but missing from `accounts_store/`.
   - 🔴 `QUOTA_EXHAUSTED`: rate-limited account cooling down until `CooldownUntil`.
   - ⚠️ `ERROR`: corrupted token or rejected refresh token.
5. **Modern Desktop UI:** Fyne 100% Pure Go Native Dark Mode Window ($440 \times 560\text{ px}$) matching IDE aesthetics, active account banner with live RAM status dot, 1-Click `[⚡ Chuyển]`, `[💾 Lưu phiên]`, `[🔄 Làm mới Chrome]`, and Auto-Failover toggle.
6. **Standalone Windows Binary (Phase P6):**
   - Single standalone Windows binary `antigravity-switch.exe` (10MB–15MB).
   - Instant startup ($\le 10\text{ms}$ CLI / $\le 50\text{ms}$ GUI), ultra-low memory ($\le 12\text{MB}$ RAM daemon / $\le 20\text{MB}$ RAM GUI), CPU $< 0.05\%$.
   - Direct Win32 `Advapi32.dll` integration using `golang.org/x/sys/windows` with zero CGo.
   - Zero external runtime dependencies (no Python, no pip, no MSVC runtimes).
7. **Go-Native Autonomous Quota Failover Daemon (Phase P7):**
   - Non-blocking goroutine log tailer on `language_server.log` detecting HTTP 429 and `RESOURCE_EXHAUSTED` in $\le 20\text{ms}$.
   - Thread-safe `AccountPool` (`sync.RWMutex`) with Round-Robin / LRU scheduler and exponential backoff cooldowns (15m, 30m, 60m capped at 2h).
   - Hysteresis anti-flapping engine (15s minimum dwell time) and sliding-window circuit breaker (max 4 switches per 120s).
   - Native Windows System Tray HUD with dynamic status dots (🟢 Healthy, 🟡 Active/Single, 🔴 All Exhausted with countdown tooltip) and global hotkey (`Win+Alt+S`).
   - Discrete Windows 10/11 Toast Notification bridge alerting developer of automated switches without stealing keyboard focus.
   - Total automated failover latency $< 1.0\text{ second}$.
   - 100% IDE context preservation guarantee: open Monaco editor buffers, terminal PTYs, and chat history unaffected.

## Acceptance Criteria

1. `accounts_store/` stores individual valid JSON files per email without corruption.
2. Chrome scanner discovers all local Google accounts while strictly excluding blacklisted emails.
3. Language server connector dynamically identifies `language_server.exe` PID, port, and `--csrf_token` argument without hardcoding.
4. Calling `GetUserStatus()` returns the exact email running in RAM.
5. Calling `AuthLogout()` cleanly purges the session from language server RAM.
6. Hot-swap execution transitions Antigravity from Account A to Account B within < 1 second.
7. Subsequent prompt activity uses Account B's quota.
8. 100% of unit and integration tests pass cleanly.
9. `antigravity-switch.exe` compiles into a single static Windows executable ($\le 15\text{MB}$) with zero Python or MSVC runtime dependencies.
10. Fyne native GUI launches in $< 50\text{ms}$, consumes $< 20\text{MB}$ RAM, and executes 1-click hot-swap cleanly.
11. Goroutine log tailer detects HTTP 429 and `RESOURCE_EXHAUSTED` in $\le 20\text{ms}$ with $< 0.05\%$ idle CPU.
12. Failover controller performs complete hot-swap cycle in $< 1.0\text{s}$ while preserving 100% IDE editor tabs and unsaved context.
13. 15-second hysteresis gate debounces quota bursts, and all-exhausted sleep mode activates gracefully when the pool is depleted.

## Checklist

- [x] P0-A: Complete actual status before implementation work.
  - Goal: establish the real current state and classify all repository surfaces.
  - Work Steps: inspect source files, verify `language_server.exe` runtime, run baseline tests, and author `actual-status.md`, `evidence.md`, `benchmark.md`.
  - Implementation Gate: `actual-status.md` completed with R0 baseline and final P0 decision.
  - Acceptance: actual status classifies all 8 target units, records exact evidence IDs (`E0-P0A-SRC1..SRC8`), and defines concrete downstream phase decisions.

### P1: Account Store & Credential Management (Persistent JSON Storage Layer)

- Phase Goal: Ensure robust, isolated JSON storage for accounts in `accounts_store/<email>.json` and reliable Windows Credential Manager integration.
- Phase Boundary:
  - In scope: `core/account_store.py`, `core/credential_manager.py`, `accounts_store/`.
  - Out of scope: Language server API calls, UI rendering.
  - Dependencies: P0 baseline verification.
- Phase Implementation Rule: Implement `P1-A`, verify it, record evidence, refresh actual-status, commit, then proceed to `P1-B`.
- Ordered Slice List:
  - P1-A: Account Store Directory & File I/O Management
  - P1-B: Active Session Capture & Credential Synchronization

- [x] P1-A: Account Store Directory & File I/O Management
  - Goal: Provide safe, atomic JSON file reading, writing, listing, and removal in `accounts_store/` with email sanitization.
  - Scope Boundary:
    - Editable: `core/account_store.py`.
    - Inspect-only: `docs/MANUAL_SWITCHER_PLAN.md`.
    - Preserve-only: `accounts_store/` directory layout.
    - Out of scope: Windows Credential Manager API.
  - Non-Goals: Do not encrypt JSON files (plain JSON per Owner's architecture; OS directory permissions apply).
  - Pre-flight Questions:
    - Data source: Local JSON files in `accounts_store/<email>.json`.
    - Display permission: N/A.
    - DB read flow: `AccountStore.get_account_token(email)` loads and parses JSON.
    - DB write flow: `AccountStore.store_account(email, token_data)` atomic dump to file.
    - Render location: N/A.
    - UI behavior flow: N/A.
    - Docker runtime: N/A (Windows desktop environment).
    - Playwright target: N/A.
    - Behavior test: Unit test verifying store, retrieve, list, and removal.
    - Cleanup/quarantine: Temporary test directory in `tests/test_accounts_store`.
    - External side effects: Filesystem write in workspace `accounts_store/`.
    - N/A notes: Docker and Playwright not applicable to native Windows desktop script.
  - Work Steps:
    1. Code (Execute leaf atomic tasks in core source files only):
       - Task 1.1: Atomic JSON File Storage Enhancement
         + Action: Ensure `AccountStore.store_account` performs atomic write via temporary file replacement to prevent partial write on sudden process kill.
         + Inputs: `email: str`, `token_data: Dict[str, Any]`.
         + Allowed Edit Scope: `core/account_store.py::AccountStore.store_account`.
         + Outputs: Boolean indicating storage success.
         + Verification Condition: Verify file exists and contains valid JSON payload.
         + Task Checkpoint: `wip(P1-A): task 1.1 - atomic JSON store implemented and verified`
       - Task 1.2: Directory Auto-Creation and Sanitation Safety
         + Action: Verify `sanitize_filename` correctly handles unusual email formats, domain dots, and casing.
         + Inputs: Arbitrary email strings.
         + Allowed Edit Scope: `core/account_store.py::sanitize_filename`.
         + Outputs: Safe lowercase filename ending in `.json`.
         + Verification Condition: Pass unit assertions with special character emails.
         + Task Checkpoint: `wip(P1-A): task 1.2 - email filename sanitation hardened`
    2. Code Inspection & Lint:
       - Review `core/account_store.py` diff and run syntax check.
    3. Update Tests (Code first, test second):
       - Add test cases in `tests/test_all.py` testing atomic store edge cases and corrupt file handling.
    4. Build:
       - Run `python -m py_compile core/account_store.py`.
    5. QA (Mini QA / Runtime Validation):
       - Run test harness verifying file creation in isolated test folder.
       - Evidence target: `E1-P1A-STORE1`, `E1-P1A-STORE2`.
    6. Slice Acceptance & Official Milestone Commit:
       - Verify slice acceptance criteria, record evidence in `evidence.md`, refresh `actual-status.md`, commit with `feat(P1-A): ensure atomic JSON account storage`.
  - Implementation Gate: `P0-A` complete; target unit classified as `correct` in `actual-status.md`.
  - Acceptance:
    - Source: `core/account_store.py` atomic write and safe listing.
    - Runtime/UI: N/A.
    - DB/data: Files in `accounts_store/` parse cleanly as valid JSON.
    - Behavior test: `tests/test_all.py` store tests pass.
    - Cleanup/quarantine: Test store directory removed after test run.
    - Evidence IDs: `E1-P1A-STORE1`, `E1-P1A-STORE2`.
    - Actual-status rows refreshed: `core/account_store.py`.
  - Evidence Targets: `E1-P1A-STORE1`, `E1-P1A-STORE2`.
  - Actual-status Update: Confirm row `core/account_store.py` remains `correct`.
  - Commit Boundary: Commit milestone `feat(P1-A): ensure atomic JSON account storage`.

- [x] P1-B: Active Session Capture & Credential Synchronization
  - Goal: Enable reading and writing the active Antigravity OAuth credential via Windows Credential Manager (`LegacyGeneric:target=gemini:antigravity`).
  - Scope Boundary:
    - Editable: `core/credential_manager.py`.
    - Inspect-only: Windows Credential Manager via `Advapi32.dll`.
    - Preserve-only: Target name `LegacyGeneric:target=gemini:antigravity` and user `antigravity`.
    - Out of scope: Other Windows credentials.
  - Non-Goals: Modifying credentials belonging to other applications.
  - Pre-flight Questions:
    - Data source: Windows Credential Manager generic credential blob.
    - Display permission: N/A.
    - DB read flow: `CredReadW` parses `CredentialBlob` to UTF-8 JSON.
    - DB write flow: `CredWriteW` updates `CredentialBlob` with UTF-8 JSON.
    - Render location: N/A.
    - UI behavior flow: N/A.
    - Docker runtime: N/A (Windows Credential Manager is a host OS API).
    - Playwright target: N/A.
    - Behavior test: Read active credential and assert structure; mock write round-trip.
    - Cleanup/quarantine: Preserve original credential if test modifies it.
    - External side effects: Modifies Windows Credential Manager generic credential.
    - N/A notes: Docker and Playwright N/A.
  - Work Steps:
    1. Code (Execute leaf atomic tasks in core source files only):
       - Task 1.1: Error Handling & Resilience in Credential Read
         + Action: Ensure `read_antigravity_token` returns `None` gracefully without crashing when credential is missing or empty.
         + Inputs: Windows Credential target.
         + Allowed Edit Scope: `core/credential_manager.py::read_antigravity_token`.
         + Outputs: Optional token dictionary.
         + Verification Condition: Calling with non-existent target returns `None`.
         + Task Checkpoint: `wip(P1-B): task 1.1 - credential read resilience verified`
       - Task 1.2: Validated Serialization in Credential Write
         + Action: Validate token payload structure before serializing to prevent writing malformed JSON into Windows Credential Manager.
         + Inputs: `token_data: Dict[str, Any]`.
         + Allowed Edit Scope: `core/credential_manager.py::write_antigravity_token`.
         + Outputs: Boolean success flag.
         + Verification Condition: Invalid payload rejected; valid payload serializes and writes.
         + Task Checkpoint: `wip(P1-B): task 1.2 - credential write payload validation verified`
    2. Code Inspection & Lint:
       - Review `core/credential_manager.py` diff and ensure `Advapi32` signatures are intact.
    3. Update Tests (Code first, test second):
       - Ensure `test_live_credential_read` asserts expected token structure.
    4. Build:
       - Run `python -m py_compile core/credential_manager.py`.
    5. QA (Mini QA / Runtime Validation):
       - Run unittest verifying successful read of live credential.
       - Evidence target: `E1-P1B-CRED1`, `E1-P1B-AUTO1`.
    6. Slice Acceptance & Official Milestone Commit:
       - Verify slice acceptance criteria, record evidence in `evidence.md`, refresh `actual-status.md`, commit with `feat(P1-B): harden Windows Credential Manager integration`.
  - Implementation Gate: `P1-A` completed; `core/credential_manager.py` classified as `correct`.
  - Acceptance:
    - Source: `core/credential_manager.py` cleanly handles read/write errors.
    - Runtime/UI: N/A.
    - DB/data: Windows Credential Manager payload matches Antigravity expectations.
    - Behavior test: `tests/test_all.py` credential test passes.
    - Cleanup/quarantine: Existing credential preserved.
    - Evidence IDs: `E1-P1B-CRED1`, `E1-P1B-AUTO1`.
    - Actual-status rows refreshed: `core/credential_manager.py`.
  - Evidence Targets: `E1-P1B-CRED1`, `E1-P1B-AUTO1`.
  - Actual-status Update: Confirm row `core/credential_manager.py` remains `correct`.
  - Commit Boundary: Commit milestone `feat(P1-B): harden Windows Credential Manager integration`.

### P2: Antigravity Internal API Connector (RAM & Session Synchronization Layer)

- Phase Goal: Build the core internal API connector module (`core/api_connector.py`) to discover `language_server.exe`, query `GetUserStatus()`, execute `AuthLogout()`, and trigger Re-Auth into RAM.
- Phase Boundary:
  - In scope: `core/api_connector.py`, `core/session_reloader.py`.
  - Out of scope: Chrome scanning, UI layout.
  - Dependencies: P1 completed.
- Phase Implementation Rule: Implement `P2-A`, verify it, record evidence, refresh actual-status, commit, then continue to `P2-B` and `P2-C`.
- Ordered Slice List:
  - P2-A: Language Server Process Discovery (Port & CSRF Token Extraction)
  - P2-B: Internal Auth API Client (GetUserStatus & AuthLogout)
  - P2-C: Re-Auth Trigger & RAM Session Reload

- [x] P2-A: Language Server Process Discovery (Port & CSRF Token Extraction)
  - Goal: Automatically detect the running `language_server.exe` process, parse `--csrf_token` from its command line, and detect its listening loopback port.
  - Scope Boundary:
    - Editable: `core/api_connector.py` (new file).
    - Inspect-only: Windows process table (via `psutil` or `ctypes`/PowerShell/WMI/CIM).
    - Preserve-only: N/A.
    - Out of scope: Terminating or modifying processes.
  - Non-Goals: Hardcoding port numbers or tokens.
  - Pre-flight Questions:
    - Data source: OS process information (`Win32_Process`) and TCP listening table (`Get-NetTCPConnection` or `netstat`).
    - Display permission: N/A.
    - DB read flow: Read command-line args for `--csrf_token` and `--https_server_port`.
    - DB write flow: N/A.
    - Render location: N/A.
    - UI behavior flow: N/A.
    - Docker runtime: N/A (Windows host processes).
    - Playwright target: N/A.
    - Behavior test: Discover PID, port, and token when language server is running; return `None` when absent.
    - Cleanup/quarantine: Read-only process query; no cleanup required.
    - External side effects: Query OS process table.
    - N/A notes: Docker/Playwright N/A.
  - Work Steps:
    1. Code (Execute leaf atomic tasks in core source files only):
       - Task 1.1: Language Server Process Identification
         + Action: Implement `find_language_server_process() -> Optional[LanguageServerInfo]` scanning running processes for `language_server.exe`.
         + Inputs: Process list from OS.
         + Allowed Edit Scope: `core/api_connector.py::find_language_server_process`.
         + Outputs: Dataclass `LanguageServerInfo(pid, csrf_token, port)`.
         + Verification Condition: Returns valid PID when Antigravity is active.
         + Task Checkpoint: `wip(P2-A): task 1.1 - process identification implemented`
       - Task 1.2: CSRF Token Extraction
         + Action: Parse `--csrf_token <UUID>` from the process command-line arguments using regex.
         + Inputs: Process command line string.
         + Allowed Edit Scope: `core/api_connector.py::extract_csrf_token`.
         + Outputs: CSRF token string.
         + Verification Condition: Correctly extracts UUID matching `E0-P0A-PROC1`.
         + Task Checkpoint: `wip(P2-A): task 1.2 - CSRF token extraction verified`
       - Task 1.3: Listening Port Discovery
         + Action: Resolve loopback listening port owned by the `language_server.exe` PID (e.g. via `Get-NetTCPConnection` or TCP table).
         + Inputs: Process PID.
         + Allowed Edit Scope: `core/api_connector.py::discover_listening_port`.
         + Outputs: Integer port number.
         + Verification Condition: Discovered port matches active listener (`E0-P0A-PROC2`).
         + Task Checkpoint: `wip(P2-A): task 1.3 - listening port discovery verified`
    2. Code Inspection & Lint:
       - Verify syntax, typing, and exception handling for absent process.
    3. Update Tests (Code first, test second):
       - Add unit test in `tests/test_api_connector.py` (or `tests/test_all.py`) testing process discovery and regex extraction with mock CLI strings.
    4. Build:
       - Run `python -m py_compile core/api_connector.py`.
    5. QA (Mini QA / Runtime Validation):
       - Run script against live host to confirm extraction of PID 17516, port, and CSRF token.
       - Evidence target: `E2-P2A-DISC1`.
    6. Slice Acceptance & Official Milestone Commit:
       - Verify slice acceptance criteria, record evidence in `evidence.md`, refresh `actual-status.md`, commit with `feat(P2-A): implement language server discovery`.
  - Implementation Gate: `P1-B` completed; `core/api_connector.py` classified as `missing -> edit`.
  - Acceptance:
    - Source: `core/api_connector.py` exports discovery functions.
    - Runtime/UI: Dynamic discovery succeeds on live host.
    - DB/data: Extracted PID, port, and token populated.
    - Behavior test: Unit tests pass.
    - Cleanup/quarantine: N/A.
    - Evidence IDs: `E2-P2A-DISC1`.
    - Actual-status rows refreshed: `core/api_connector.py` (partial).
  - Evidence Targets: `E2-P2A-DISC1`.
  - Actual-status Update: Transition `core/api_connector.py` from `missing` to `partial`.
  - Commit Boundary: Commit milestone `feat(P2-A): implement language server discovery`.

- [x] P2-B: Internal Auth API Client (GetUserStatus & AuthLogout)
  - Goal: Implement Step 1 (`GetUserStatus`) and Step 2 (`AuthLogout`) HTTP clients communicating with `language_server.exe` using the discovered port and CSRF token.
  - Scope Boundary:
    - Editable: `core/api_connector.py`.
    - Inspect-only: `language_server.log`.
    - Preserve-only: N/A.
    - Out of scope: Modifying language server internals.
  - Non-Goals: Calling arbitrary or unauthorized RPC endpoints.
  - Pre-flight Questions:
    - Data source: Antigravity language server HTTP/JSON endpoints.
    - Display permission: N/A.
    - DB read flow: `GetUserStatus` request with CSRF header returns JSON containing active user email and name.
    - DB write flow: `AuthLogout` request clears session state in RAM.
    - Render location: N/A.
    - UI behavior flow: N/A.
    - Docker runtime: N/A.
    - Playwright target: N/A.
    - Behavior test: Execute mock HTTP calls and live validation against language server port.
    - Cleanup/quarantine: N/A.
    - External side effects: HTTP requests to loopback `127.0.0.1:<port>`.
    - N/A notes: Docker/Playwright N/A.
  - Work Steps:
    1. Code (Execute leaf atomic tasks in core source files only):
       - Task 1.1: `GetUserStatus()` Client Implementation (Step 1)
         + Action: Implement `get_user_status(info: LanguageServerInfo) -> Optional[Dict[str, Any]]` issuing GET/POST request with CSRF token to language server to query active user in RAM.
         + Inputs: `LanguageServerInfo` (host, port, csrf_token).
         + Allowed Edit Scope: `core/api_connector.py::get_user_status`.
         + Outputs: Dict containing `email`, `display_name`, `status`.
         + Verification Condition: Returns current account email matching active session in RAM.
         + Task Checkpoint: `wip(P2-B): task 1.1 - GetUserStatus API client verified`
       - Task 1.2: `AuthLogout()` Client Implementation (Step 2)
         + Action: Implement `auth_logout(info: LanguageServerInfo) -> bool` issuing logout request with CSRF token to language server to purge RAM token and destroy Account A session.
         + Inputs: `LanguageServerInfo`.
         + Allowed Edit Scope: `core/api_connector.py::auth_logout`.
         + Outputs: Boolean indicating logout success.
         + Verification Condition: API responds with success; server logs indicate session termination.
         + Task Checkpoint: `wip(P2-B): task 1.2 - AuthLogout API client verified`
    2. Code Inspection & Lint:
       - Verify HTTP timeout handling, standard library `urllib.request` usage (zero extra dependencies), and error handling.
    3. Update Tests (Code first, test second):
       - Add test cases in `tests/test_all.py` testing `get_user_status` and `auth_logout` with mock HTTP server.
    4. Build:
       - Run `python -m py_compile core/api_connector.py`.
    5. QA (Mini QA / Runtime Validation):
       - Verify HTTP request headers and response formatting.
       - Evidence target: `E2-P2B-API1`, `E2-P2B-API2`.
    6. Slice Acceptance & Official Milestone Commit:
       - Verify slice acceptance criteria, record evidence in `evidence.md`, refresh `actual-status.md`, commit with `feat(P2-B): implement GetUserStatus and AuthLogout API clients`.
  - Implementation Gate: `P2-A` completed; discovery verified.
  - Acceptance:
    - Source: `core/api_connector.py` contains `get_user_status` and `auth_logout`.
    - Runtime/UI: API calls return structured responses.
    - DB/data: Active user email retrieved accurately.
    - Behavior test: Unit tests pass.
    - Cleanup/quarantine: N/A.
    - Evidence IDs: `E2-P2B-API1`, `E2-P2B-API2`.
    - Actual-status rows refreshed: `core/api_connector.py` (partial).
  - Evidence Targets: `E2-P2B-API1`, `E2-P2B-API2`.
  - Actual-status Update: Refresh row `core/api_connector.py`.
  - Commit Boundary: Commit milestone `feat(P2-B): implement GetUserStatus and AuthLogout API clients`.

- [x] P2-C: Re-Auth Trigger & RAM Session Reload
  - Goal: Implement Step 4: trigger Re-Auth / session reload to prompt Antigravity to read Token B from Windows Credential Manager into RAM ("Auth succeeded: Tài khoản B").
  - Scope Boundary:
    - Editable: `core/api_connector.py`, `core/session_reloader.py`.
    - Inspect-only: Windows window handles.
    - Preserve-only: N/A.
    - Out of scope: N/A.
  - Non-Goals: Forcing process crash or kill.
  - Pre-flight Questions:
    - Data source: Window enumeration and language server reload endpoints.
    - Display permission: N/A.
    - DB read flow: N/A.
    - DB write flow: N/A.
    - Render location: Antigravity IDE UI window.
    - UI behavior flow: Antigravity UI updates header / user badge cleanly.
    - Docker runtime: N/A.
    - Playwright target: N/A.
    - Behavior test: Trigger re-auth and observe language server reload confirmation.
    - Cleanup/quarantine: N/A.
    - External side effects: Dispatches reload signal to Antigravity window / API.
    - N/A notes: Docker/Playwright N/A.
  - Work Steps:
    1. Code (Execute leaf atomic tasks in core source files only):
       - Task 1.1: Re-Auth API / Signal Dispatch
         + Action: Implement `trigger_reauth(info: LanguageServerInfo) -> bool` issuing re-auth command to language server or dispatching reload to IDE window.
         + Inputs: `LanguageServerInfo`.
         + Allowed Edit Scope: `core/api_connector.py::trigger_reauth`.
         + Outputs: Boolean indicating re-auth signal sent.
         + Verification Condition: Language server re-authenticates and logs `Auth succeeded`.
         + Task Checkpoint: `wip(P2-C): task 1.1 - re-auth signal dispatch implemented`
       - Task 1.2: Coordinated Session Reloader Bridge
         + Action: Update `core/session_reloader.py` to coordinate with `trigger_reauth` ensuring both backend RAM and frontend Electron window are refreshed smoothly without stealing user focus unnecessarily.
         + Inputs: Target window handle.
         + Allowed Edit Scope: `core/session_reloader.py::reload_antigravity_window`.
         + Outputs: Boolean success.
         + Verification Condition: Antigravity UI refreshes cleanly.
         + Task Checkpoint: `wip(P2-C): task 1.2 - coordinated session reloader bridge verified`
    2. Code Inspection & Lint:
       - Review `core/api_connector.py` and `core/session_reloader.py` diff.
    3. Update Tests (Code first, test second):
       - Add test verifying coordinated re-auth trigger flow.
    4. Build:
       - Run `python -m py_compile core/session_reloader.py core/api_connector.py`.
    5. QA (Mini QA / Runtime Validation):
       - Verify re-auth trigger dispatch.
       - Evidence target: `E2-P2C-REAUTH1`.
    6. Slice Acceptance & Official Milestone Commit:
       - Verify slice acceptance criteria, record evidence in `evidence.md`, refresh `actual-status.md`, commit with `feat(P2-C): implement Re-Auth trigger and session reload`.
  - Implementation Gate: `P2-B` completed.
  - Acceptance:
    - Source: `core/api_connector.py` and `core/session_reloader.py` fully wired.
    - Runtime/UI: Antigravity re-reads credential into RAM.
    - DB/data: Token B active in RAM.
    - Behavior test: Tests pass.
    - Cleanup/quarantine: N/A.
    - Evidence IDs: `E2-P2C-REAUTH1`.
    - Actual-status rows refreshed: `core/api_connector.py` -> `correct`, `core/session_reloader.py` -> `correct`.
  - Evidence Targets: `E2-P2C-REAUTH1`.
  - Actual-status Update: Transition `core/api_connector.py` to `correct` and `core/session_reloader.py` to `correct`.
  - Commit Boundary: Commit milestone `feat(P2-C): implement Re-Auth trigger and session reload`.

### P3: Chrome Discovery & Multi-Account State Resolver

- Phase Goal: Integrate Chrome account discovery with strict blacklist filtering, and update the state resolver to orchestrate the Owner's mandated 4-step closed-loop hot-swap flow.
- Phase Boundary:
  - In scope: `scanner/chrome_scanner.py`, `core/state_resolver.py`.
  - Out of scope: GUI Tkinter code.
  - Dependencies: P1 and P2 completed.
- Phase Implementation Rule: Implement `P3-A`, verify it, record evidence, refresh actual-status, commit, then proceed to `P3-B` and `P3-C`.
- Ordered Slice List:
  - P3-A: Chrome Profile Extraction & Blacklist Sanitization
  - P3-B: Multi-Account State Machine Resolution (ACTIVE, READY, UNAUTH)
  - P3-C: 4-Step Closed-Loop Hot-Swap Coordinator

- [x] P3-A: Chrome Profile Extraction & Blacklist Sanitization
  - Goal: Ensure robust extraction of signed-in Chrome accounts across all user profiles while strictly blocking blacklisted terms (`anvien`).
  - Scope Boundary:
    - Editable: `scanner/chrome_scanner.py`.
    - Inspect-only: Chrome preferences files.
    - Preserve-only: `BLACKLIST_KEYWORDS = ["anvien"]`.
    - Out of scope: Non-Chrome browsers.
  - Non-Goals: Modifying or writing to Chrome directories.
  - Pre-flight Questions:
    - Data source: `%LOCALAPPDATA%\Google\Chrome\User Data\Local State` and `Preferences`.
    - Display permission: N/A.
    - DB read flow: Read-only regex and JSON parsing.
    - DB write flow: N/A.
    - Render location: N/A.
    - UI behavior flow: N/A.
    - Docker runtime: N/A.
    - Playwright target: N/A.
    - Behavior test: Unit tests asserting blacklist rejection and email format validity.
    - Cleanup/quarantine: N/A.
    - External side effects: Read-only filesystem access.
    - N/A notes: Docker/Playwright N/A.
  - Work Steps:
    1. Code (Execute leaf atomic tasks in core source files only):
       - Task 1.1: Chrome Profile Scanner Hardening
         + Action: Ensure scanner handles missing profile directories, locked files, and edge-case UTF-8 encoded preference strings gracefully without throwing unhandled exceptions.
         + Inputs: Chrome User Data path.
         + Allowed Edit Scope: `scanner/chrome_scanner.py::extract_emails_from_preferences`.
         + Outputs: Set of discovered emails.
         + Verification Condition: Handles malformed files returning empty set safely.
         + Task Checkpoint: `wip(P3-A): task 1.1 - scanner preference parsing hardened`
       - Task 1.2: Blacklist Enforcement Verification
         + Action: Re-verify that all email normalization strips whitespace, lowercases, and checks substring against `BLACKLIST_KEYWORDS = ["anvien"]`.
         + Inputs: Sample emails.
         + Allowed Edit Scope: `scanner/chrome_scanner.py::is_blacklisted`.
         + Outputs: Boolean flag.
         + Verification Condition: Returns `True` for any variant containing `anvien`.
         + Task Checkpoint: `wip(P3-A): task 1.2 - blacklist keyword filter verified`
    2. Code Inspection & Lint:
       - Review `scanner/chrome_scanner.py` diff and verify zero third-party dependencies.
    3. Update Tests (Code first, test second):
       - Ensure `test_blacklist_enforcement` and `test_live_chrome_scanner` cover edge cases.
    4. Build:
       - Run `python -m py_compile scanner/chrome_scanner.py`.
    5. QA (Mini QA / Runtime Validation):
       - Run live scanner check.
       - Evidence target: `E3-P3A-SCAN1`.
    6. Slice Acceptance & Official Milestone Commit:
       - Verify slice acceptance criteria, record evidence in `evidence.md`, refresh `actual-status.md`, commit with `feat(P3-A): harden Chrome profile account discovery`.
  - Implementation Gate: `P2-C` completed; `scanner/chrome_scanner.py` classified as `correct`.
  - Acceptance:
    - Source: `scanner/chrome_scanner.py` error-safe.
    - Runtime/UI: Returns list of clean Google accounts.
    - DB/data: Zero blacklisted emails returned.
    - Behavior test: Tests pass.
    - Cleanup/quarantine: N/A.
    - Evidence IDs: `E3-P3A-SCAN1`.
    - Actual-status rows refreshed: `scanner/chrome_scanner.py`.
  - Evidence Targets: `E3-P3A-SCAN1`.
  - Actual-status Update: Confirm row `scanner/chrome_scanner.py` remains `correct`.
  - Commit Boundary: Commit milestone `feat(P3-A): harden Chrome profile account discovery`.

- [x] P3-B: Multi-Account State Machine Resolution (ACTIVE, READY, UNAUTH)
  - Goal: Resolve each account into `ACTIVE` (matching true email in RAM via `GetUserStatus()`), `READY` (in Chrome + saved in `accounts_store/`), or `UNAUTH` (in Chrome but not yet in `accounts_store/`).
  - Scope Boundary:
    - Editable: `core/state_resolver.py`.
    - Inspect-only: `core/account_store.py`, `core/api_connector.py`.
    - Preserve-only: AccountInfo data structure.
    - Out of scope: UI rendering.
  - Non-Goals: Inferring status from stale assumptions.
  - Pre-flight Questions:
    - Data source: `api_connector.get_user_status()`, `AccountStore`, `chrome_scanner`.
    - Display permission: N/A.
    - DB read flow: Query true RAM active email, check `store.has_account(email)`.
    - DB write flow: Auto-capture active session to `accounts_store/` if not present.
    - Render location: N/A.
    - UI behavior flow: N/A.
    - Docker runtime: N/A.
    - Playwright target: N/A.
    - Behavior test: Unit test verifying classification logic.
    - Cleanup/quarantine: N/A.
    - External side effects: Potential auto-capture write to `accounts_store/`.
    - N/A notes: Docker/Playwright N/A.
  - Work Steps:
    1. Code (Execute leaf atomic tasks in core source files only):
       - Task 1.1: Live RAM Identity Binding
         + Action: Update `AccountManager.get_current_active_email` to prioritize live RAM email from `api_connector.get_user_status()`, falling back to Windows Credential Manager if language server is not running.
         + Inputs: System state.
         + Allowed Edit Scope: `core/state_resolver.py::AccountManager.get_current_active_email`.
         + Outputs: Verified active email string or `None`.
         + Verification Condition: Returns RAM email when language server is running.
         + Task Checkpoint: `wip(P3-B): task 1.1 - live RAM identity binding implemented`
       - Task 1.2: State Machine Categorization
         + Action: Categorize discovered Chrome accounts into `ACTIVE` (matches RAM active email), `READY` (stored in `accounts_store/<email>.json`), or `UNAUTH` (missing from store).
         + Inputs: List of Chrome emails, active email, `AccountStore`.
         + Allowed Edit Scope: `core/state_resolver.py::AccountManager.resolve_accounts`.
         + Outputs: `Tuple[Optional[str], List[AccountInfo]]`.
         + Verification Condition: Correct badge and color assigned to each account.
         + Task Checkpoint: `wip(P3-B): task 1.2 - state machine categorization verified`
    2. Code Inspection & Lint:
       - Review `core/state_resolver.py` diff.
    3. Update Tests (Code first, test second):
       - Add test verifying state resolution with mock RAM email and mock account store.
    4. Build:
       - Run `python -m py_compile core/state_resolver.py`.
    5. QA (Mini QA / Runtime Validation):
       - Verify state machine output matrix.
       - Evidence target: `E3-P3B-STATE1`.
    6. Slice Acceptance & Official Milestone Commit:
       - Verify slice acceptance criteria, record evidence in `evidence.md`, refresh `actual-status.md`, commit with `feat(P3-B): bind state resolution to live RAM identity`.
  - Implementation Gate: `P3-A` completed; `core/api_connector.py` available.
  - Acceptance:
    - Source: `core/state_resolver.py` uses `api_connector`.
    - Runtime/UI: Accounts resolved with accurate 🟢 and 🟡 states.
    - DB/data: State matches filesystem and RAM reality.
    - Behavior test: Tests pass.
    - Cleanup/quarantine: N/A.
    - Evidence IDs: `E3-P3B-STATE1`.
    - Actual-status rows refreshed: `core/state_resolver.py` (partial).
  - Evidence Targets: `E3-P3B-STATE1`.
  - Actual-status Update: Refresh row `core/state_resolver.py`.
  - Commit Boundary: Commit milestone `feat(P3-B): bind state resolution to live RAM identity`.

- [x] P3-C: 4-Step Closed-Loop Hot-Swap Coordinator
  - Goal: Implement the complete 4-step closed-loop hot-swap execution in `AccountManager.switch_account`:
    1. Verify current RAM email via `GetUserStatus()`;
    2. Issue `AuthLogout()` to language server API;
    3. Write Account B token from `accounts_store/` to Windows Credential Manager;
    4. Trigger Re-Auth / Reload to load Token B into RAM.
  - Scope Boundary:
    - Editable: `core/state_resolver.py`.
    - Inspect-only: `core/api_connector.py`, `core/credential_manager.py`, `core/account_store.py`.
    - Preserve-only: N/A.
    - Out of scope: GUI code.
  - Non-Goals: Skipping any of the 4 steps.
  - Pre-flight Questions:
    - Data source: `accounts_store/<target_email>.json`.
    - Display permission: N/A.
    - DB read flow: Read target token from store.
    - DB write flow: Write target token to Windows Credential Manager.
    - Render location: N/A.
    - UI behavior flow: N/A.
    - Docker runtime: N/A.
    - Playwright target: N/A.
    - Behavior test: End-to-end hot swap simulation asserting exact sequence of 4 steps.
    - Cleanup/quarantine: Restore original credential after test.
    - External side effects: Modifies active Antigravity session.
    - N/A notes: Docker/Playwright N/A.
  - Work Steps:
    1. Code (Execute leaf atomic tasks in core source files only):
       - Task 1.1: 4-Step Hot-Swap Execution Flow
         + Action: Implement `AccountManager.switch_account(target_email: str) -> Tuple[bool, str]` executing:
           1. Step 1: Query language server to confirm active state and auto-capture current session if not saved.
           2. Step 2: Issue `api_connector.auth_logout()` to flush old token from RAM.
           3. Step 3: Read target token from `self.store.get_account_token(target_email)` and write to Windows Credential Manager via `write_antigravity_token(token)`.
           4. Step 4: Call `api_connector.trigger_reauth()` and `reload_antigravity_window()`.
         + Inputs: `target_email: str`.
         + Allowed Edit Scope: `core/state_resolver.py::AccountManager.switch_account`.
         + Outputs: `Tuple[bool, str]` indicating success status and descriptive message.
         + Verification Condition: Returns `(True, "Đã chuyển đổi thành công sang...")` and verifies all 4 steps executed in exact order.
         + Task Checkpoint: `wip(P3-C): task 1.1 - 4-step hot-swap execution flow implemented`
       - Task 1.2: Error Boundary & Rollback Safeguards
         + Action: If Step 2, Step 3, or Step 4 fails, abort safely with clear error messaging without corrupting store or leaving credential in an undefined state.
         + Inputs: Error conditions.
         + Allowed Edit Scope: `core/state_resolver.py::AccountManager.switch_account`.
         + Outputs: Error message tuple.
         + Verification Condition: Target missing from store returns informative error before touching credentials.
         + Task Checkpoint: `wip(P3-C): task 1.2 - hot-swap error safeguards verified`
    2. Code Inspection & Lint:
       - Review `core/state_resolver.py` diff and verify clean logging/status reporting.
    3. Update Tests (Code first, test second):
       - Add comprehensive test in `tests/test_all.py` asserting all 4 steps execute in exact sequence.
    4. Build:
       - Run `python -m py_compile core/state_resolver.py`.
    5. QA (Mini QA / Runtime Validation):
       - Verify hot-swap loop execution.
       - Evidence target: `E3-P3C-SW1`.
    6. Slice Acceptance & Official Milestone Commit:
       - Verify slice acceptance criteria, record evidence in `evidence.md`, refresh `actual-status.md`, commit with `feat(P3-C): implement 4-step closed-loop hot-swap coordinator`.
  - Implementation Gate: `P3-B` completed.
  - Acceptance:
    - Source: `core/state_resolver.py` implements 4-step loop.
    - Runtime/UI: Switching transitions session cleanly.
    - DB/data: Credential manager updated with target token.
    - Behavior test: Sequence tests pass.
    - Cleanup/quarantine: N/A.
    - Evidence IDs: `E3-P3C-SW1`.
    - Actual-status rows refreshed: `core/state_resolver.py` -> `correct`.
  - Evidence Targets: `E3-P3C-SW1`.
  - Actual-status Update: Transition `core/state_resolver.py` to `correct`.
  - Commit Boundary: Commit milestone `feat(P3-C): implement 4-step closed-loop hot-swap coordinator`.

### P4: UI Integration & Hot-Swap Trigger (Tkinter Dark Mode GUI)

- Phase Goal: Update the Tkinter desktop GUI in `ui/app_window.py` to reflect verified RAM status, provide responsive 1-click hot-swap controls with live feedback, and expose session capture and Chrome refresh.
- Phase Boundary:
  - In scope: `ui/app_window.py`.
  - Out of scope: `core/`, `scanner/`.
  - Dependencies: P1, P2, P3 completed.
- Phase Implementation Rule: Implement `P4-A`, verify it, record evidence, refresh actual-status, commit, then proceed to `P4-B` and `P4-C`.
- Ordered Slice List:
  - P4-A: Real-time Account List & Visual State Badging
  - P4-B: 1-Click Hot-Swap Trigger & Status Feedback
  - P4-C: Session Persistence & Manual Chrome Sync Controls

- [x] P4-A: Real-time Account List & Visual State Badging
  - Goal: Render accounts in the dark mode UI with accurate visual state badges (🟢 `[ĐANG DÙNG]`, 🟢 `[⚡ Chuyển sang]`, 🟡 `[Chưa lưu vào kho]`) and display active RAM email in the header.
  - Scope Boundary:
    - Editable: `ui/app_window.py`.
    - Inspect-only: `core/state_resolver.py`.
    - Preserve-only: Dark palette theme constants (`BG_DARK`, `BG_CARD`, `COLOR_GREEN`, `COLOR_YELLOW`, `COLOR_BLUE`).
    - Out of scope: External UI frameworks.
  - Non-Goals: Adding heavy external GUI libraries (stick strictly to standard library `tkinter`).
  - Pre-flight Questions:
    - Data source: `AccountManager.resolve_accounts()`.
    - Display permission: N/A.
    - DB read flow: Retrieve list of `AccountInfo`.
    - DB write flow: N/A.
    - Render location: Desktop window Tkinter widgets.
    - UI behavior flow: Header shows active RAM email; scrollable card list displays accounts.
    - Docker runtime: N/A (Tkinter native desktop app).
    - Playwright target: N/A.
    - Behavior test: Verify widget instantiation and row generation.
    - Cleanup/quarantine: Destroy root after test.
    - External side effects: Desktop window rendering.
    - N/A notes: Docker/Playwright N/A.
  - Work Steps:
    1. Code (Execute leaf atomic tasks in core source files only):
       - Task 1.1: Header RAM Status Presentation
         + Action: Update `SwitcherApp.refresh_accounts` to display live RAM identity in header status label with clear green dot indicator.
         + Inputs: Active email from `manager.resolve_accounts()`.
         + Allowed Edit Scope: `ui/app_window.py::SwitcherApp.refresh_accounts`.
         + Outputs: Updated label text: `● Đang hoạt động: <email>`.
         + Verification Condition: Label reflects active email.
         + Task Checkpoint: `wip(P4-A): task 1.1 - header RAM status display updated`
       - Task 1.2: Card Rendering & Visual Badging
         + Action: Ensure `_render_account_row` correctly styles active card with bold highlight, ready card with `[⚡ Chuyển sang]` button, and unsaved card with yellow badge.
         + Inputs: `AccountInfo`.
         + Allowed Edit Scope: `ui/app_window.py::SwitcherApp._render_account_row`.
         + Outputs: Populated Tkinter row widgets.
         + Verification Condition: Cards render matching account status attributes.
         + Task Checkpoint: `wip(P4-A): task 1.2 - card rendering and badging verified`
    2. Code Inspection & Lint:
       - Review `ui/app_window.py` diff and verify Tkinter layout consistency.
    3. Update Tests (Code first, test second):
       - Add test verifying UI initialization without error.
    4. Build:
       - Run `python -m py_compile ui/app_window.py`.
    5. QA (Mini QA / Runtime Validation):
       - Verify UI cards and header display.
       - Evidence target: `E4-P4A-UI1`.
    6. Slice Acceptance & Official Milestone Commit:
       - Verify slice acceptance criteria, record evidence in `evidence.md`, refresh `actual-status.md`, commit with `feat(P4-A): render live account list and RAM status in UI`.
  - Implementation Gate: `P3-C` completed; `ui/app_window.py` classified as `partial`.
  - Acceptance:
    - Source: `ui/app_window.py` renders live state.
    - Runtime/UI: Dark mode window displays accurate cards.
    - DB/data: Matches state resolver.
    - Behavior test: Tests pass.
    - Cleanup/quarantine: Root window destroyed.
    - Evidence IDs: `E4-P4A-UI1`.
    - Actual-status rows refreshed: `ui/app_window.py` (partial).
  - Evidence Targets: `E4-P4A-UI1`.
  - Actual-status Update: Refresh row `ui/app_window.py`.
  - Commit Boundary: Commit milestone `feat(P4-A): render live account list and RAM status in UI`.

- [x] P4-B: 1-Click Hot-Swap Trigger & Status Feedback
  - Goal: Wire the `[⚡ Chuyển sang]` button to execute the 4-step hot-swap flow with instant visual status feedback and user alerts.
  - Scope Boundary:
    - Editable: `ui/app_window.py`.
    - Inspect-only: `core/state_resolver.py`.
    - Preserve-only: N/A.
    - Out of scope: Backend logic.
  - Non-Goals: Blocking GUI thread during API calls.
  - Pre-flight Questions:
    - Data source: Click event on `[⚡ Chuyển sang]`.
    - Display permission: N/A.
    - DB read flow: Read target token.
    - DB write flow: Execute 4-step switch via `AccountManager`.
    - Render location: Status bar and message dialogs.
    - UI behavior flow: Click -> status bar shows "Đang chuyển đổi..." -> success dialog -> list refreshes -> status bar shows updated timestamp.
    - Docker runtime: N/A.
    - Playwright target: N/A.
    - Behavior test: Test click callback execution.
    - Cleanup/quarantine: N/A.
    - External side effects: Triggers account switch.
    - N/A notes: Docker/Playwright N/A.
  - Work Steps:
    1. Code (Execute leaf atomic tasks in core source files only):
       - Task 1.1: Hot-Swap Action Handler
         + Action: Implement `_on_switch_click(self, target_email: str)` updating status bar to "Đang chuyển đổi...", invoking `manager.switch_account(target_email)`, refreshing UI, and displaying completion alert.
         + Inputs: `target_email: str`.
         + Allowed Edit Scope: `ui/app_window.py::SwitcherApp._on_switch_click`.
         + Outputs: UI refresh and message dialog.
         + Verification Condition: Handles success and failure states gracefully.
         + Task Checkpoint: `wip(P4-B): task 1.1 - hot-swap UI action handler verified`
    2. Code Inspection & Lint:
       - Review callback error boundaries and UI responsiveness.
    3. Update Tests (Code first, test second):
       - Add test verifying callback invokes `switch_account`.
    4. Build:
       - Run `python -m py_compile ui/app_window.py`.
    5. QA (Mini QA / Runtime Validation):
       - Verify hot-swap trigger feedback.
       - Evidence target: `E4-P4B-ACT1`.
    6. Slice Acceptance & Official Milestone Commit:
       - Verify slice acceptance criteria, record evidence in `evidence.md`, refresh `actual-status.md`, commit with `feat(P4-B): connect hot-swap trigger with progress feedback`.
  - Implementation Gate: `P4-A` completed.
  - Acceptance:
    - Source: `ui/app_window.py` button handler wired.
    - Runtime/UI: 1-click switching provides immediate feedback.
    - DB/data: Account switch executes.
    - Behavior test: Tests pass.
    - Cleanup/quarantine: N/A.
    - Evidence IDs: `E4-P4B-ACT1`.
    - Actual-status rows refreshed: `ui/app_window.py` (partial).
  - Evidence Targets: `E4-P4B-ACT1`.
  - Actual-status Update: Refresh row `ui/app_window.py`.
  - Commit Boundary: Commit milestone `feat(P4-B): connect hot-swap trigger with progress feedback`.

- [x] P4-C: Session Persistence & Manual Chrome Sync Controls
  - Goal: Verify and harden the `[💾 Lưu phiên hiện tại]` button and `[🔄 Làm mới Chrome]` button, and ensure the status bar reports accurate counts and timestamps.
  - Scope Boundary:
    - Editable: `ui/app_window.py`.
    - Inspect-only: `core/account_store.py`.
    - Preserve-only: N/A.
    - Out of scope: N/A.
  - Non-Goals: N/A.
  - Pre-flight Questions:
    - Data source: User button click.
    - Display permission: N/A.
    - DB read flow: Query accounts and store counts.
    - DB write flow: Save active session to JSON file.
    - Render location: Desktop window.
    - UI behavior flow: Click save -> session captured -> list refreshed. Click refresh -> Chrome re-scanned -> list refreshed.
    - Docker runtime: N/A.
    - Playwright target: N/A.
    - Behavior test: Test session save and refresh handlers.
    - Cleanup/quarantine: N/A.
    - External side effects: Modifies `accounts_store/`.
    - N/A notes: Docker/Playwright N/A.
  - Work Steps:
    1. Code (Execute leaf atomic tasks in core source files only):
       - Task 1.1: Session Capture and Manual Refresh Handlers
         + Action: Verify `_on_save_current_session` calls `manager.auto_capture_active_session()`, refreshes account list, and alerts user; verify `refresh_accounts` updates status bar with exact counts.
         + Inputs: Button events.
         + Allowed Edit Scope: `ui/app_window.py::SwitcherApp._on_save_current_session`, `ui/app_window.py::SwitcherApp.refresh_accounts`.
         + Outputs: Status updates.
         + Verification Condition: Status bar displays `Đồng bộ lúc <time> • <N> tài khoản trong kho JSON • <M> trên Chrome`.
         + Task Checkpoint: `wip(P4-C): task 1.1 - session persistence and sync controls verified`
    2. Code Inspection & Lint:
       - Review `ui/app_window.py` diff.
    3. Update Tests (Code first, test second):
       - Add test verifying session save callback.
    4. Build:
       - Run `python -m py_compile ui/app_window.py`.
    5. QA (Mini QA / Runtime Validation):
       - Verify manual sync and status bar updates.
       - Evidence target: `E4-P4C-SYNC1`.
    6. Slice Acceptance & Official Milestone Commit:
       - Verify slice acceptance criteria, record evidence in `evidence.md`, refresh `actual-status.md`, commit with `feat(P4-C): finalize UI session persistence and refresh controls`.
  - Implementation Gate: `P4-B` completed.
  - Acceptance:
    - Source: `ui/app_window.py` complete.
    - Runtime/UI: All buttons and status bar operate reliably.
    - DB/data: Saved JSON files created upon request.
    - Behavior test: Tests pass.
    - Cleanup/quarantine: N/A.
    - Evidence IDs: `E4-P4C-SYNC1`.
    - Actual-status rows refreshed: `ui/app_window.py` -> `correct`.
  - Evidence Targets: `E4-P4C-SYNC1`.
  - Actual-status Update: Transition `ui/app_window.py` to `correct`.
  - Commit Boundary: Commit milestone `feat(P4-C): finalize UI session persistence and refresh controls`.

### P5: Verification & Regression QA

- Phase Goal: Author comprehensive unit, integration, and regression tests ensuring 100% test coverage and flawless switching without regressions.
- Phase Boundary:
  - In scope: `tests/test_all.py`, `tests/`.
  - Out of scope: Application source code modification (tests only).
  - Dependencies: P1, P2, P3, P4 completed.
- Phase Implementation Rule: Implement `P5-A`, verify it, record evidence, refresh actual-status, commit, then proceed to `P5-B`.
- Ordered Slice List:
  - P5-A: Comprehensive Unit & Integration Test Suite
  - P5-B: End-to-End Switching Cycle & Robustness Verification

- [x] P5-A: Comprehensive Unit & Integration Test Suite
  - Goal: Expand `tests/test_all.py` to cover `core/api_connector.py`, `core/session_reloader.py`, `core/account_store.py`, and `core/state_resolver.py`.
  - Scope Boundary:
    - Editable: `tests/test_all.py`.
    - Inspect-only: All `core/` and `scanner/` modules.
    - Preserve-only: All 6 baseline test cases.
    - Out of scope: Modifying core application files.
  - Non-Goals: Modifying production logic during test writing.
  - Pre-flight Questions:
    - Data source: Test fixtures and mocks.
    - Display permission: N/A.
    - DB read flow: Mock reads.
    - DB write flow: Mock writes in isolated test directory.
    - Render location: Console test runner.
    - UI behavior flow: N/A.
    - Docker runtime: N/A.
    - Playwright target: N/A.
    - Behavior test: Full unit test execution.
    - Cleanup/quarantine: Temporary test directories removed in `tearDown`.
    - External side effects: Zero side effects outside test harness.
    - N/A notes: Docker/Playwright N/A.
  - Work Steps:
    1. Code (Execute leaf atomic tasks in test source files only):
       - Task 1.1: API Connector Discovery Unit Tests
         + Action: Add test cases verifying PID detection, regex extraction of `--csrf_token`, and TCP port resolution.
         + Inputs: Sample CLI argument strings.
         + Allowed Edit Scope: `tests/test_all.py`.
         + Outputs: Passing test assertions.
         + Verification Condition: `python -m unittest tests/test_all.py` passes new test cases.
         + Task Checkpoint: `wip(P5-A): task 1.1 - API connector unit tests added`
       - Task 1.2: 4-Step Switching Flow Unit Tests
         + Action: Add mock integration test verifying `switch_account` executes Step 1 (`GetUserStatus`), Step 2 (`AuthLogout`), Step 3 (Credential write), and Step 4 (Re-Auth trigger) in exact sequence.
         + Inputs: Mock language server and mock store.
         + Allowed Edit Scope: `tests/test_all.py`.
         + Outputs: Passing test assertions.
         + Verification Condition: Sequence assertions pass.
         + Task Checkpoint: `wip(P5-A): task 1.2 - 4-step switching flow unit tests verified`
    2. Code Inspection & Lint:
       - Review `tests/test_all.py` diff and ensure test independence.
    3. Update Tests (Code first, test second):
       - Run full test suite.
    4. Build:
       - Run `python -m py_compile tests/test_all.py`.
    5. QA (Mini QA / Runtime Validation):
       - Run `python -m unittest tests/test_all.py`.
       - Evidence target: `E5-P5A-UNIT1`.
    6. Slice Acceptance & Official Milestone Commit:
       - Verify slice acceptance criteria, record evidence in `evidence.md`, refresh `actual-status.md`, commit with `test(P5-A): expand comprehensive test suite`.
  - Implementation Gate: `P4-C` completed.
  - Acceptance:
    - Source: `tests/test_all.py` updated.
    - Runtime/UI: Test runner reports all tests passed.
    - DB/data: Zero test artifact leakage.
    - Behavior test: >= 12 tests pass.
    - Cleanup/quarantine: Isolated test store removed.
    - Evidence IDs: `E5-P5A-UNIT1`.
    - Actual-status rows refreshed: `tests/test_all.py` (partial).
  - Evidence Targets: `E5-P5A-UNIT1`.
  - Actual-status Update: Refresh row `tests/test_all.py`.
  - Commit Boundary: Commit milestone `test(P5-A): expand comprehensive test suite`.

- [x] P5-B: End-to-End Switching Cycle & Robustness Verification
  - Goal: Implement PID-based 100% accurate Antigravity window discovery in `core/session_reloader.py`, refine `core/state_resolver.py::AccountManager.switch_account` to execute smooth hot-swap (bypassing `AuthLogout()` by default to avoid triggering Chrome browser OAuth re-login, writing Token B directly to Windows Credential Manager, and sending reload signal directly to `Antigravity.exe` window to load Token B into RAM), and run full regression & multi-run stability validation ensuring 100% test pass rate across the entire test suite.
  - Scope Boundary:
    - Editable: `core/session_reloader.py`, `core/state_resolver.py`, `tests/test_all.py`, `docs/plans/2026-10-14-antigravity-multi-account-switcher/`.
    - Inspect-only: Entire repository (`core/`, `scanner/`, `ui/`, `tests/`).
    - Preserve-only: Baseline test coverage and existing account store JSON formats.
    - Out of scope: Modifying Chrome scanner logic or UI layout.
  - Non-Goals: Invoking interactive browser login prompts or terminating Antigravity process forcefully.
  - Pre-flight Questions:
    - Data source: `Antigravity.exe` process IDs (via ctypes / Windows API / tasklist), window enumeration (`EnumWindows`), and `accounts_store/<target_email>.json`.
    - Display permission: N/A.
    - DB read flow: Read target Token B from `accounts_store/`.
    - DB write flow: Write target Token B directly to Windows Credential Manager (`LegacyGeneric:target=gemini:antigravity`).
    - Render location: Antigravity IDE UI window (receives reload event).
    - UI behavior flow: Antigravity session reloads smoothly without launching Chrome OAuth login tab.
    - Docker runtime: N/A.
    - Playwright target: N/A.
    - Behavior test: Validate PID-based window discovery, assert smooth hot-swap bypasses `AuthLogout()`, and execute full regression test suite.
    - Cleanup/quarantine: All temporary test accounts, tokens, and credentials cleaned up in test `tearDown`.
    - External side effects: Modifies active Antigravity session credential and dispatches UI reload event.
    - N/A notes: Docker/Playwright N/A.
  - Work Steps:
    1. Code (Execute leaf atomic tasks in core source files and test harness):
       - Task 1.1: PID-based Window Discovery in `core/session_reloader.py::find_antigravity_windows`
         + Action: Upgrade `find_antigravity_windows()` to discover all process IDs of `Antigravity.exe` using standard Windows commands/ctypes (`GetWindowThreadProcessId`, matching PID against running `Antigravity.exe` processes obtained via `psutil`/`tasklist`/ctypes), ensuring 100% accurate HWND identification regardless of window title mutations, while retaining window title matching as a robust fallback.
         + Inputs: System processes, visible window handles (`EnumWindows`).
         + Allowed Edit Scope: `core/session_reloader.py::find_antigravity_windows`.
         + Outputs: `List[wintypes.HWND]` belonging strictly to `Antigravity.exe` processes.
         + Verification Condition: Accurately identifies window handle of `Antigravity.exe` even when window title does not contain `"antigravity"`.
         + Task Checkpoint: `wip(P5-B): task 1.1 - PID-based window discovery implemented`
       - Task 1.2: Smooth Hot-Swap Execution in `core/state_resolver.py::AccountManager.switch_account`
         + Action: Refactor `AccountManager.switch_account` to execute the Owner's smooth hot-swap mechanism: bypass `AuthLogout()` by default (e.g., via `skip_logout=True`) to eliminate unwanted Chrome OAuth re-login popups, write Token B from `accounts_store/` directly into Windows Credential Manager via `token_writer`, and dispatch window/session reload directly to `Antigravity.exe` to immediately load Token B into RAM.
         + Inputs: `target_email: str`, `accounts_store/` token JSON, `skip_logout: bool = True`.
         + Allowed Edit Scope: `core/state_resolver.py::AccountManager.switch_account`.
         + Outputs: `SwitchResult` with success status and user feedback.
         + Verification Condition: Token B is committed to credential store, `AuthLogout()` is skipped by default, reload signal is dispatched, and no external browser login is triggered.
         + Task Checkpoint: `wip(P5-B): task 1.2 - smooth hot-swap execution bypassing logout implemented`
       - Task 1.3: Full Regression & Multi-Run Stability across Test Suite
         + Action: Expand `tests/test_all.py` with comprehensive test cases for PID-based window discovery and smooth hot-swap (asserting `auth_logout` is not invoked during default switch), and execute multi-run test suite across all modules (`core`, `scanner`, `ui`) verifying 100% pass rate with zero regressions.
         + Inputs: Updated `core/session_reloader.py`, `core/state_resolver.py`, and test harness.
         + Allowed Edit Scope: `tests/test_all.py`.
         + Outputs: 100% passing test assertions across all test cases.
         + Verification Condition: `python -m unittest tests/test_all.py` passes 100% with zero errors and zero failures.
         + Task Checkpoint: `wip(P5-B): task 1.3 - regression suite and smooth hot-swap tests verified`
    2. Code Inspection & Lint:
       - Review diff in `core/session_reloader.py`, `core/state_resolver.py`, and `tests/test_all.py` for clean error handling, ctypes resource management, and zero external dependency footprint.
    3. Update Tests (Code first, test second):
       - Confirm all unit and regression tests in `tests/test_all.py` cover PID window discovery, smooth hot-swap without logout, and credential write verification.
    4. Build:
       - Compile all project files: `python -m compileall core scanner ui tests main.py`.
    5. QA (Mini QA / Runtime Validation):
       - Run full test suite: `python -m unittest tests/test_all.py`.
       - Evidence target: `E5-P5B-REG1`, `E5-P5B-PID1`, `E5-P5B-SWOOTH1`.
    6. Slice Acceptance & Official Milestone Commit:
       - Verify slice acceptance criteria, record evidence in `evidence.md`, refresh `actual-status.md`, commit with `test(P5-B): verify complete switching regression suite with PID window discovery and smooth hot-swap`.
  - Implementation Gate: `P5-A` completed.
  - Acceptance:
    - Source: `core/session_reloader.py` implements PID-based window discovery; `core/state_resolver.py` executes smooth hot-swap bypassing `AuthLogout()`.
    - Runtime/UI: Antigravity window accurately targeted by PID; hot-swap reloads session without launching Chrome OAuth.
    - DB/data: Windows Credential Manager updated with target Token B.
    - Behavior test: 100% test pass rate across `tests/test_all.py`.
    - Cleanup/quarantine: Clean worktree; temporary test credentials purged.
    - Evidence IDs: `E5-P5B-REG1`, `E5-P5B-PID1`, `E5-P5B-SWOOTH1`.
    - Actual-status rows refreshed: `core/session_reloader.py` -> `correct`, `core/state_resolver.py` -> `correct`, `tests/test_all.py` -> `correct`.
  - Evidence Targets: `E5-P5B-REG1`, `E5-P5B-PID1`, `E5-P5B-SWOOTH1`.
  - Actual-status Update: Transition `core/session_reloader.py`, `core/state_resolver.py`, and `tests/test_all.py` to `correct`.
  - Commit Boundary: Commit milestone `test(P5-B): verify complete switching regression suite with PID window discovery and smooth hot-swap`.

- Phase Goal: Port the core switcher engine to compiled Go (Golang), implement direct Win32 `Advapi32.dll` credential management via `golang.org/x/sys/windows`, build a pure Go native Fyne GUI dashboard (< 20MB RAM, < 50ms startup), and compile into a standalone Windows binary `antigravity-switch.exe` ($\le 15\text{MB}$) with zero Python or MSVC runtime dependencies.
- Phase Boundary:
  - In scope: `go.mod`, `go.sum`, `cmd/switcher/main.go`, `internal/wincred/`, `internal/store/`, `internal/model/`, `internal/scanner/`, `internal/connector/`, `internal/reloader/`, `internal/manager/`, `internal/ui/`, `scripts/build_windows.ps1`, `antigravity-switch.exe`.
  - Out of scope: Interpreted Python runtime modifications, CGo bindings, background log tailer (Phase P7).
  - Dependencies: P0–P5 completed and verified.
- Phase Implementation Rule: Implement slices in strict order: `P6-A` -> `P6-B` -> `P6-C` -> `P6-D`.
- Ordered Slice List:
  - P6-A: Go Core Primitives & Native Win32 Credential Layer
  - P6-B: Chrome Discovery Scanner & Language Server Connector
  - P6-C: Core Orchestrator & Fyne Native GUI
  - P6-D: Standalone Windows Binary Compilation & Benchmark

- [x] P6-A: Go Core Primitives & Native Win32 Credential Layer
  - Goal: Initialize Go module `antigravity-switch`, implement pure Win32 `Advapi32.dll` credential management via `golang.org/x/sys/windows` targeting `LegacyGeneric:target=gemini:antigravity`, define shared model structs, and create atomic JSON persistence in `accounts_store/<email>.json` with path traversal guards.
  - Scope Boundary:
    - Editable: `go.mod`, `go.sum`, `internal/wincred/cred.go`, `internal/wincred/types.go`, `internal/model/types.go`, `internal/store/account_store.go`, `internal/store/sanitizer.go`, `internal/wincred/cred_test.go`, `internal/store/account_store_test.go`.
    - Inspect-only: `core/account_store.py`, `core/credential_manager.py`, `accounts_store/`.
    - Preserve-only: `LegacyGeneric:target=gemini:antigravity` target name and JSON schema format for tokens (`access_token`, `refresh_token`, `id_token`, `expiry`, `token_type`).
    - Out of scope: Network HTTP calls, GUI code, scanner logic.
  - Non-Goals: No CGo requirements; pure Go Win32 system calls only.
  - Pre-flight Questions:
    - Data source: Windows Credential Manager (`Advapi32.dll`) and `accounts_store/` directory.
    - Display permission: N/A.
    - DB read flow: `CredReadW` from Windows Credential Manager; `os.ReadFile` / `json.NewDecoder` from `accounts_store/<email>.json`.
    - DB write flow: `CredWriteW` into Windows Credential Manager; atomic file write via `os.CreateTemp` + `os.Rename` into `accounts_store/<email>.json`.
    - Render location: Headless CLI / unit test runner.
    - UI behavior flow: N/A.
    - Docker runtime: N/A.
    - Playwright target: N/A.
    - Behavior test: Test credential read/write round-trip; test atomic store persistence and corruption recovery.
    - Cleanup/quarantine: All test credentials and test files purged during test teardown.
    - External side effects: Modifies `LegacyGeneric:target=gemini:antigravity` during test, restores original state.
    - N/A notes: Docker/Playwright N/A.
  - Work Steps:
    1. Code (Execute leaf atomic tasks in core source files only):
       - Task 1.1: Go Module Initialization & Data Models in `go.mod` and `internal/model/types.go`
         + Action: Initialize `go.mod` for `antigravity-switch`, declare module dependencies (`golang.org/x/sys`), define `TokenData`, `AccountRecord`, and `AccountState` structs matching the Architect specification Section 4.3.
         + Inputs: Architect specification Section 4.3.
         + Allowed Edit Scope: `go.mod`, `internal/model/types.go`.
         + Outputs: Initialized Go module and strongly typed model definitions.
         + Verification Condition: `go vet ./internal/model/...` compiles without error.
         + Task Checkpoint: `wip(P6-A): task 1.1 - Go module and model structs initialized`
       - Task 1.2: Native Win32 Credential Wrapper in `internal/wincred/`
         + Action: Implement `Advapi32.dll` wrappers (`CredReadW`, `CredWriteW`, `CredDeleteW`, `CredFree`) using `golang.org/x/sys/windows` targeting `LegacyGeneric:target=gemini:antigravity` with UTF-16 conversions and memory safety.
         + Inputs: Windows API documentation and `core/credential_manager.py` ctypes logic.
         + Allowed Edit Scope: `internal/wincred/cred.go`, `internal/wincred/types.go`.
         + Outputs: Native Go functions `ReadToken()`, `WriteToken()`, `DeleteToken()`, `HasToken()`.
         + Verification Condition: Functions compile cleanly with `go build ./internal/wincred/...`.
         + Task Checkpoint: `wip(P6-A): task 1.2 - Win32 credential manager wrapper implemented`
       - Task 1.3: Atomic JSON Store & Path Sanitizer in `internal/store/`
         + Action: Implement `AccountStore` for `accounts_store/<email>.json` with atomic rename pattern (`os.CreateTemp`, `json.NewEncoder`, `Sync()`, `os.Rename()`), directory auto-creation, and email path sanitizer preventing traversal.
         + Inputs: `core/account_store.py` logic and JSON token schema.
         + Allowed Edit Scope: `internal/store/account_store.go`, `internal/store/sanitizer.go`.
         + Outputs: Thread-safe `AccountStore` with `SaveAccount()`, `GetAccount()`, `HasAccount()`, `DeleteAccount()`, `ListAccounts()`.
         + Verification Condition: `go build ./internal/store/...` succeeds.
         + Task Checkpoint: `wip(P6-A): task 1.3 - atomic JSON store and sanitizer implemented`
    2. Code Inspection & Lint:
       - Review diff, syntax, `go vet ./...` on `internal/wincred` and `internal/store`, ensuring strict boundary adherence and memory safety.
    3. Update Tests (Code first, test second):
       - Author `internal/wincred/cred_test.go` and `internal/store/account_store_test.go` covering round-trip storage, UTF-16 credential conversion, and path traversal guards.
    4. Build:
       - Run `go test -c ./internal/wincred/...` and `go test -c ./internal/store/...`.
    5. QA (Mini QA / Runtime Validation):
       - Run `go test -v ./internal/wincred/... ./internal/store/...`.
       - Evidence target: `E6-P6A-WIN1`, `E6-P6A-STORE1`.
    6. Slice Acceptance & Official Milestone Commit:
       - Verify slice acceptance criteria, record evidence in `evidence.md`, refresh `actual-status.md`, commit with `feat(P6-A): implement Go core primitives and native Win32 credential layer`.
  - Implementation Gate: `P5-B` and architect specification approved.
  - Acceptance:
    - Source: `go.mod`, `internal/model/types.go`, `internal/wincred/cred.go`, `internal/store/account_store.go` implemented.
    - Runtime/UI: Pure Go Win32 calls execute without CGo.
    - DB/data: `accounts_store/<email>.json` atomic file persistence confirmed.
    - Behavior test: 100% unit tests pass across `internal/wincred` and `internal/store`.
    - Cleanup/quarantine: Isolated temporary test files purged.
    - Evidence IDs: `E6-P6A-WIN1`, `E6-P6A-STORE1`.
    - Actual-status rows refreshed: `internal/wincred` -> `correct`, `internal/store` -> `correct`, `internal/model` -> `correct`.
  - Evidence Targets: `E6-P6A-WIN1`, `E6-P6A-STORE1`.
  - Actual-status Update: Transition `internal/wincred`, `internal/store`, `internal/model` to `correct`.
  - Commit Boundary: Commit milestone `feat(P6-A): implement Go core primitives and native Win32 credential layer`.

- [x] P6-B: Chrome Discovery Scanner & Language Server Connector
  - Goal: Port the Chrome profile discovery scanner with strict `anvien` keyword filtering and the loopback HTTP / Connect-RPC client for `language_server.exe` process discovery, `GetUserStatus()`, `AuthLogout()`, and `trigger_reauth()`.
  - Scope Boundary:
    - Editable: `internal/scanner/chrome_scanner.go`, `internal/scanner/chrome_scanner_test.go`, `internal/connector/discovery.go`, `internal/connector/api_client.go`, `internal/connector/endpoints.go`, `internal/connector/connector_test.go`.
    - Inspect-only: `scanner/chrome_scanner.py`, `core/api_connector.py`, running `language_server.exe` process.
    - Preserve-only: Blacklist keyword filtering (`["anvien"]`), loopback port discovery via `Get-NetTCPConnection` / Windows netstat, `X-Csrf-Token` header handshake.
    - Out of scope: GUI rendering, systray.
  - Non-Goals: No direct modification of Chrome user profiles; read-only extraction.
  - Pre-flight Questions:
    - Data source: `%LOCALAPPDATA%\Google\Chrome\User Data` (Preferences, Local State), Windows process table for `language_server.exe` command line and TCP port.
    - Display permission: N/A.
    - DB read flow: Read Chrome JSON preferences; query OS TCP table.
    - DB write flow: N/A.
    - Render location: Headless CLI / unit test runner.
    - UI behavior flow: N/A.
    - Docker runtime: N/A.
    - Playwright target: N/A.
    - Behavior test: Assert Chrome profiles extracted with blacklist filtering; assert mock HTTP server handles `GetUserStatus`, `AuthLogout`, and reauth.
    - Cleanup/quarantine: Mock HTTP servers closed after tests.
    - External side effects: Network calls strictly to `127.0.0.1:<port>`.
    - N/A notes: Docker/Playwright N/A.
  - Work Steps:
    1. Code (Execute leaf atomic tasks in core source files only):
       - Task 1.1: Chrome Profile Scanner with Blacklist in `internal/scanner/chrome_scanner.go`
         + Action: Port Chrome scanner to Go, reading `%LOCALAPPDATA%\Google\Chrome\User Data\Local State` and profile `Preferences` JSON files, extracting logged-in Google emails, and enforcing strict case-insensitive filtering against blacklist (`anvien`).
         + Inputs: `scanner/chrome_scanner.py`.
         + Allowed Edit Scope: `internal/scanner/chrome_scanner.go`.
         + Outputs: `ScanChromeProfiles(userDataDir string, blacklist []string) ([]string, error)`.
         + Verification Condition: Discovers signed-in Chrome emails and excludes any email containing `anvien`.
         + Task Checkpoint: `wip(P6-B): task 1.1 - Chrome profile scanner with blacklist implemented`
       - Task 1.2: Language Server Process & Port Discovery in `internal/connector/discovery.go`
         + Action: Port language server discovery to Go, enumerating processes for `language_server.exe`, parsing `--csrf_token` and command line args, and finding the ephemeral listening port via Windows TCP netstat / Win32 `GetExtendedTcpTable`.
         + Inputs: `core/api_connector.py::discover_language_server`.
         + Allowed Edit Scope: `internal/connector/discovery.go`.
         + Outputs: `DiscoverLanguageServer() (*ServerInfo, error)` returning PID, Port, and CSRF token.
         + Verification Condition: Discovers live `language_server.exe` PID and loopback port.
         + Task Checkpoint: `wip(P6-B): task 1.2 - language server discovery implemented`
       - Task 1.3: Loopback HTTP & Connect-RPC Client in `internal/connector/`
         + Action: Implement high-efficiency `net/http` client with timeout, connection reuse, and custom headers (`X-Csrf-Token`, `x-codeium-csrf-token`), and define RPC methods: `GetUserStatus()`, `AuthLogout()`, and `TriggerReauth()`.
         + Inputs: `core/api_connector.py` endpoints and Architect specification Section 2.
         + Allowed Edit Scope: `internal/connector/api_client.go`, `internal/connector/endpoints.go`.
         + Outputs: Methods `GetUserStatus(ctx) (*UserStatus, error)`, `AuthLogout(ctx) error`, `TriggerReauth(ctx) error`.
         + Verification Condition: Passes against mock HTTP server returning 200 OK and error payloads.
         + Task Checkpoint: `wip(P6-B): task 1.3 - loopback HTTP RPC client implemented`
    2. Code Inspection & Lint:
       - Review diff, syntax, `go vet ./internal/scanner/... ./internal/connector/...`.
    3. Update Tests (Code first, test second):
       - Author `internal/scanner/chrome_scanner_test.go` and `internal/connector/connector_test.go`.
    4. Build:
       - Run `go test -c ./internal/scanner/...` and `go test -c ./internal/connector/...`.
    5. QA (Mini QA / Runtime Validation):
       - Run `go test -v ./internal/scanner/... ./internal/connector/...`.
       - Evidence target: `E6-P6B-SCAN1`, `E6-P6B-CONN1`.
    6. Slice Acceptance & Official Milestone Commit:
       - Verify slice acceptance criteria, record evidence in `evidence.md`, refresh `actual-status.md`, commit with `feat(P6-B): implement Chrome discovery scanner and language server connector`.
  - Implementation Gate: `P6-A` completed.
  - Acceptance:
    - Source: `internal/scanner/chrome_scanner.go` and `internal/connector/` implemented.
    - Runtime/UI: Process discovery resolves PID, port, CSRF token; HTTP client handles RPCs.
    - DB/data: N/A.
    - Behavior test: 100% tests pass in scanner and connector packages.
    - Cleanup/quarantine: Mock servers cleanly stopped.
    - Evidence IDs: `E6-P6B-SCAN1`, `E6-P6B-CONN1`.
    - Actual-status rows refreshed: `internal/scanner` -> `correct`, `internal/connector` -> `correct`.
  - Evidence Targets: `E6-P6B-SCAN1`, `E6-P6B-CONN1`.
  - Actual-status Update: Transition `internal/scanner` and `internal/connector` to `correct`.
  - Commit Boundary: Commit milestone `feat(P6-B): implement Chrome discovery scanner and language server connector`.

- [x] P6-C: Core Orchestrator & Fyne Native GUI
  - Goal: Implement the coordinated session reloader, multi-account orchestrator (`AccountManager`), and the pure Go native Fyne GUI dashboard (`app.go`, `window.go`, `widgets.go`, `tray.go`), plus CLI/GUI entrypoint `cmd/switcher/main.go`.
  - Scope Boundary:
    - Editable: `internal/reloader/session_reloader.go`, `internal/reloader/reloader_test.go`, `internal/manager/account_manager.go`, `internal/manager/manager_test.go`, `internal/ui/app.go`, `internal/ui/window.go`, `internal/ui/widgets.go`, `internal/ui/tray.go`, `cmd/switcher/main.go`.
    - Inspect-only: `ui/app_window.py`, `core/state_resolver.py`, `core/session_reloader.py`.
    - Preserve-only: Compact window dimensions ($440 \times 560\text{ px}$), dark theme aesthetics, active account badge, 1-click switch button `[⚡ Chuyển]`, `[💾 Lưu phiên]`, `[🔄 Làm mới Chrome]`.
    - Out of scope: Background goroutine log tailer (belongs to Phase P7).
  - Non-Goals: No WebViews, no Electron, no browser popups during hot-swap.
  - Pre-flight Questions:
    - Data source: Discovered Chrome accounts, JSON store accounts, and live `GetUserStatus()` from Language Server.
    - Display permission: Native Windows desktop GUI display.
    - DB read flow: `AccountStore.ListAccounts()`, `Scanner.Scan()`, `Connector.GetUserStatus()`.
    - DB write flow: `AccountStore.SaveAccount()`, `WinCred.WriteToken()`.
    - Render location: Fyne desktop window and native system tray.
    - UI behavior flow: Launch window -> display active account card + pool list -> click `[⚡ Chuyển]` -> 4-step hot-swap -> live RAM status updates.
    - Docker runtime: N/A.
    - Playwright target: N/A.
    - Behavior test: Test hot-swap flow, test account state resolution (ACTIVE, READY, UNAUTH), verify GUI window construction.
    - Cleanup/quarantine: Close GUI window on exit; purge test artifacts.
    - External side effects: Modifies active Antigravity credential and reloads editor window.
    - N/A notes: Docker/Playwright N/A.
  - Work Steps:
    1. Code (Execute leaf atomic tasks in core source files only):
       - Task 1.1: Session Reloader & HWND Discovery in `internal/reloader/session_reloader.go`
         + Action: Implement PID-based window discovery via Win32 `EnumWindows` and `GetWindowThreadProcessId` targeting `Antigravity.exe`, dispatching `keybd_event` Ctrl+R reload keystroke coordinated with language server reauth.
         + Inputs: `core/session_reloader.py`.
         + Allowed Edit Scope: `internal/reloader/session_reloader.go`.
         + Outputs: `ReloadSession(pid int) error`.
         + Verification Condition: Discovers `Antigravity.exe` HWND and dispatches reload.
         + Task Checkpoint: `wip(P6-C): task 1.1 - session reloader and HWND targeting implemented`
       - Task 1.2: Account Manager Orchestrator in `internal/manager/account_manager.go`
         + Action: Implement `AccountManager` orchestrating `AccountStore`, `WinCred`, `ChromeScanner`, `Connector`, and `Reloader`, implementing the closed-loop 4-step hot-swap: probe live RAM -> clear RAM via `AuthLogout` (or smooth swap) -> write target token to Windows Credential Manager -> trigger reauth/reload.
         + Inputs: `core/state_resolver.py`.
         + Allowed Edit Scope: `internal/manager/account_manager.go`.
         + Outputs: Methods `GetAccountStates()`, `SwitchAccount(targetEmail string, skipLogout bool)`, `CaptureCurrentSession()`.
         + Verification Condition: Orchestrator passes simulated multi-account hot-swap test.
         + Task Checkpoint: `wip(P6-C): task 1.2 - account manager orchestrator implemented`
       - Task 1.3: Fyne Native GUI Dashboard in `internal/ui/`
         + Action: Implement 100% pure Go native GUI using Fyne: dark theme, compact $440 \times 560\text{ px}$ window, active account header with live status dot, scrollable account card list, 1-click `[⚡ Chuyển]` button, `[💾 Lưu phiên]`, `[🔄 Làm mới Chrome]`, and minimize-to-tray.
         + Inputs: Architect specification Section 7 (ADR-P6-05) and `ui/app_window.py`.
         + Allowed Edit Scope: `internal/ui/app.go`, `internal/ui/window.go`, `internal/ui/widgets.go`, `internal/ui/tray.go`.
         + Outputs: Functional Fyne GUI application running on native Windows desktop.
         + Verification Condition: GUI compiles and initializes cleanly without Webview/Electron dependencies.
         + Task Checkpoint: `wip(P6-C): task 1.3 - Fyne native GUI dashboard implemented`
       - Task 1.4: Dual CLI/GUI Entrypoint in `cmd/switcher/main.go`
         + Action: Create `main.go` supporting CLI flags (`--cli`, `--gui`, `--switch <email>`, `--status`, `--list`, `--capture`), defaulting to GUI mode when launched without flags.
         + Inputs: CLI design specifications.
         + Allowed Edit Scope: `cmd/switcher/main.go`.
         + Outputs: Compiled entrypoint `cmd/switcher/main.go`.
         + Verification Condition: CLI commands `--list` and `--status` output correct account state to console.
         + Task Checkpoint: `wip(P6-C): task 1.4 - dual CLI/GUI entrypoint implemented`
    2. Code Inspection & Lint:
       - Review diff, syntax, `go vet ./internal/manager/... ./internal/ui/... ./cmd/switcher/...`.
    3. Update Tests (Code first, test second):
       - Author `internal/reloader/reloader_test.go` and `internal/manager/manager_test.go`.
    4. Build:
       - Run `go build -o antigravity-switch-dev.exe ./cmd/switcher`.
    5. QA (Mini QA / Runtime Validation):
       - Run `go test -v ./internal/manager/... ./internal/reloader/...`.
       - Evidence target: `E6-P6C-MGR1`, `E6-P6C-UI1`.
    6. Slice Acceptance & Official Milestone Commit:
       - Verify slice acceptance criteria, record evidence in `evidence.md`, refresh `actual-status.md`, commit with `feat(P6-C): implement core orchestrator, Fyne native GUI, and CLI entrypoint`.
  - Implementation Gate: `P6-B` completed.
  - Acceptance:
    - Source: `internal/reloader/`, `internal/manager/`, `internal/ui/`, `cmd/switcher/` implemented.
    - Runtime/UI: Fyne GUI launches with dark theme; active account banner and account cards render.
    - DB/data: Token stored and written on switch.
    - Behavior test: 100% tests pass in manager and reloader.
    - Cleanup/quarantine: UI windows close cleanly.
    - Evidence IDs: `E6-P6C-MGR1`, `E6-P6C-UI1`.
    - Actual-status rows refreshed: `internal/reloader` -> `correct`, `internal/manager` -> `correct`, `internal/ui` -> `correct`, `cmd/switcher` -> `correct`.
  - Evidence Targets: `E6-P6C-MGR1`, `E6-P6C-UI1`.
  - Actual-status Update: Transition `internal/reloader`, `internal/manager`, `internal/ui`, `cmd/switcher` to `correct`.
  - Commit Boundary: Commit milestone `feat(P6-C): implement core orchestrator, Fyne native GUI, and CLI entrypoint`.

- [x] P6-D: Standalone Windows Binary Compilation & Benchmark
  - Goal: Package, compile, embed Windows resources, and benchmark the standalone `antigravity-switch.exe` binary, verifying size $\le 15\text{MB}$, startup $\le 10\text{ms}$, RAM $\le 12\text{MB}$ (CLI) / $\le 20\text{MB}$ (GUI), and zero external dependencies.
  - Scope Boundary:
    - Editable: `scripts/build_windows.ps1`, Windows resource manifest/icons, `cmd/switcher/main.go`, `docs/plans/2026-10-14-antigravity-multi-account-switcher/`.
    - Inspect-only: Entire Go project tree.
    - Preserve-only: Binary output name `antigravity-switch.exe`.
    - Out of scope: Python files.
  - Non-Goals: No bundling of Python interpreters or C++ runtime installers; static Go compilation.
  - Pre-flight Questions:
    - Data source: Go source tree.
    - Display permission: N/A.
    - DB read flow: N/A.
    - DB write flow: Outputs `antigravity-switch.exe`.
    - Render location: Windows file system.
    - UI behavior flow: Launch executable directly without console flash (`-H windowsgui`).
    - Docker runtime: N/A.
    - Playwright target: N/A.
    - Behavior test: Measure binary size, cold boot startup time, resident memory footprint, and CLI flag executions.
    - Cleanup/quarantine: Clean intermediate build artifacts.
    - External side effects: Produces binary `antigravity-switch.exe`.
    - N/A notes: Docker/Playwright N/A.
  - Work Steps:
    1. Code (Execute leaf atomic tasks in core build script only):
       - Task 1.1: Build Automation Script & Windows Resource Manifest
         + Action: Create `scripts/build_windows.ps1` configuring compiler flags (`-ldflags="-H windowsgui -s -w"`), stripping debug symbols, and embedding Windows executable application icon and metadata manifest.
         + Inputs: Architect specification Section 1.3 & Section 6.
         + Allowed Edit Scope: `scripts/build_windows.ps1`, resource manifest files.
         + Outputs: Automated PowerShell build script.
         + Verification Condition: Script builds `antigravity-switch.exe` with exit code 0.
         + Task Checkpoint: `wip(P6-D): task 1.1 - build automation script and resource embedding implemented`
       - Task 1.2: Binary Footprint & Dependency Verification
         + Action: Verify binary size is $\le 15\text{MB}$, check DLL imports via `dumpbin /dependents` or PowerShell script verifying only standard Windows OS DLLs (`Advapi32.dll`, `Kernel32.dll`, `User32.dll`, `Shell32.dll`) are linked, ensuring zero Python/MSVC dependencies.
         + Inputs: `antigravity-switch.exe`.
         + Allowed Edit Scope: Build output verification script.
         + Outputs: Binary dependency verification audit report.
         + Verification Condition: Binary size $\le 15\text{MB}$ and zero non-system DLL dependencies.
         + Task Checkpoint: `wip(P6-D): task 1.2 - standalone binary footprint and dependencies verified`
       - Task 1.3: Performance & Memory Benchmark Suite
         + Action: Execute benchmark testing measuring CLI startup latency ($\le 10\text{ms}$), GUI cold start ($\le 50\text{ms}$), daemon memory footprint ($\le 12\text{MB}$ RAM), and GUI memory footprint ($\le 20\text{MB}$ RAM).
         + Inputs: `antigravity-switch.exe`.
         + Allowed Edit Scope: Benchmark ledger and test runner.
         + Outputs: Recorded benchmark measurements in `benchmark.md`.
         + Verification Condition: All performance metrics meet or exceed Architect targets.
         + Task Checkpoint: `wip(P6-D): task 1.3 - performance and memory benchmarks verified`
    2. Code Inspection & Lint:
       - Inspect build script and resource embedding configs.
      3. Update Tests (Code first, test second):
         - Author automated integration test executing `--help`, `--version`, `--list` via the compiled binary.
    4. Build:
       - Run `powershell -File scripts/build_windows.ps1`.
    5. QA (Mini QA / Runtime Validation):
       - Run `.\antigravity-switch.exe --cli --list` and inspect Task Manager memory usage.
       - Evidence target: `E6-P6D-BIN1`, `E6-P6D-PERF1`.
    6. Slice Acceptance & Official Milestone Commit:
       - Verify slice acceptance criteria, record evidence in `evidence.md`, refresh `actual-status.md`, commit with `feat(P6-D): compile and benchmark standalone Windows binary antigravity-switch.exe`.
  - Implementation Gate: `P6-C` completed.
  - Acceptance:
    - Source: `scripts/build_windows.ps1` and `antigravity-switch.exe` generated.
    - Runtime/UI: Executable launches cleanly with `-H windowsgui`.
    - DB/data: N/A.
    - Behavior test: Binary size $\le 15\text{MB}$, RAM $\le 20\text{MB}$, startup $\le 10\text{ms}$.
    - Cleanup/quarantine: Intermediate object files cleaned up.
    - Evidence IDs: `E6-P6D-BIN1`, `E6-P6D-PERF1`.
    - Actual-status rows refreshed: `antigravity-switch.exe` -> `correct`.
  - Evidence Targets: `E6-P6D-BIN1`, `E6-P6D-PERF1`.
  - Actual-status Update: Transition `antigravity-switch.exe` to `correct`.
  - Commit Boundary: Commit milestone `feat(P6-D): compile and benchmark standalone Windows binary antigravity-switch.exe`.

- Phase Goal: Implement the Go-native autonomous quota failover daemon sniffing `language_server.log`, detecting HTTP 429 and `RESOURCE_EXHAUSTED` in $\le 20\text{ms}$, managing an in-memory `AccountPool` with exponential backoff cooldowns, 15s hysteresis gate, sub-second ($< 1.0\text{s}$) hot-swap execution, native Windows System Tray HUD with dynamic status dots, discrete Windows Toast alerts, and 100% IDE context preservation.
- Phase Boundary:
  - In scope: `internal/sensor/`, `internal/controller/`, `internal/tray/`, `tests/e2e/`.
  - Out of scope: Modifying Antigravity IDE Electron internals.
  - Dependencies: Phase P6 standalone Go engine completed.
- Phase Implementation Rule: Implement slices in strict order: `P7-A` -> `P7-B` -> `P7-C` -> `P7-D`.
- Ordered Slice List:
  - P7-A: Goroutine Log Tailer & Quota Sensor Pipeline
  - P7-B: Go-Native Failover Controller & Account Pool FSM
  - P7-C: Native Windows System Tray & Toast HUD
  - P7-D: End-to-End Simulation QA & Quota Storm Stress Testing

- [ ] P7-A: Goroutine Log Tailer & Quota Sensor Pipeline
  - Goal: Implement the non-blocking background goroutine log tailer for `%APPDATA%\Antigravity\logs\language_server.log`, high-performance regex matching for HTTP 429 and `RESOURCE_EXHAUSTED` errors, and the unbuffered Go event channel `chan QuotaExhaustedEvent`.
  - Scope Boundary:
    - Editable: `internal/sensor/log_tailer.go`, `internal/sensor/signatures.go`, `internal/sensor/event_bus.go`, `internal/sensor/sensor_test.go`.
    - Inspect-only: `%APPDATA%\Antigravity\logs\language_server.log`.
    - Preserve-only: Non-blocking I/O; zero CPU consumption when idle ($< 0.05\%$).
    - Out of scope: Account switching execution (handled by controller in P7-B).
  - Non-Goals: No active polling loops with sleep; incremental seek and file watcher semantics.
  - Pre-flight Questions:
    - Data source: `%APPDATA%\Antigravity\logs\language_server.log`.
    - Display permission: N/A.
    - DB read flow: Incremental byte offset read from log file.
    - DB write flow: N/A.
    - Render location: Headless goroutine / event channel.
    - UI behavior flow: N/A.
    - Docker runtime: N/A.
    - Playwright target: N/A.
    - Behavior test: Feed synthetic log lines with 429 and quota exhaustion; verify `QuotaExhaustedEvent` emitted in $\le 20\text{ms}$.
    - Cleanup/quarantine: Stop goroutines and close channels on context cancellation.
    - External side effects: Read-only handle on `language_server.log` (shared read mode).
    - N/A notes: Docker/Playwright N/A.
  - Work Steps:
    1. Code (Execute leaf atomic tasks in core source files only):
       - Task 1.1: Quota Error Signatures & Token Pattern Matcher in `internal/sensor/signatures.go`
         + Action: Implement compiled regex patterns and fast string matching for HTTP 429, gRPC `RESOURCE_EXHAUSTED` (code 8), `daily-cloudcode-pa.googleapis.com` quota limits, and `rate limit exceeded`.
         + Inputs: Architect specification Section 2 & Section 6.
         + Allowed Edit Scope: `internal/sensor/signatures.go`.
         + Outputs: `MatchQuotaExhaustion(line string) (*QuotaMatchResult, bool)`.
         + Verification Condition: Accurately identifies all quota error signatures and ignores regular log lines.
         + Task Checkpoint: `wip(P7-A): task 1.1 - quota signatures and matchers implemented`
       - Task 1.2: Non-blocking Goroutine Log Tailer in `internal/sensor/log_tailer.go`
         + Action: Implement `LogTailer` running as a background goroutine, tracking byte offset in `language_server.log`, detecting log rotation (file shrink), and reading new lines asynchronously without blocking.
         + Inputs: Windows file I/O APIs and log tailing best practices.
         + Allowed Edit Scope: `internal/sensor/log_tailer.go`.
         + Outputs: Struct `LogTailer` with `Start(ctx)`, `Stop()`, and offset tracking.
         + Verification Condition: Handles file appends, truncation, and rotation without panic or memory leaks.
         + Task Checkpoint: `wip(P7-A): task 1.2 - non-blocking goroutine log tailer implemented`
       - Task 1.3: Event Bus & Quota Channel Dispatcher in `internal/sensor/event_bus.go`
         + Action: Implement `EventBus` managing `chan QuotaExhaustedEvent`, delivering detected quota exhaustion events to registered controller listeners with zero allocation overhead.
         + Inputs: Architect specification Section 2 & Section 4.3.
         + Allowed Edit Scope: `internal/sensor/event_bus.go`.
         + Outputs: `type QuotaExhaustedEvent struct` and `EventBus` interface.
         + Verification Condition: Channel dispatches events with latency $\le 20\text{ms}$ under load.
         + Task Checkpoint: `wip(P7-A): task 1.3 - event bus and quota channel dispatcher implemented`
    2. Code Inspection & Lint:
       - Review diff, syntax, `go vet ./internal/sensor/...`.
    3. Update Tests (Code first, test second):
       - Author `internal/sensor/sensor_test.go` testing pattern matching, synthetic log tailing with 100,000 lines, and channel event emission.
    4. Build:
       - Run `go test -c ./internal/sensor/...`.
    5. QA (Mini QA / Runtime Validation):
       - Run `go test -v ./internal/sensor/...`.
       - Evidence target: `E7-P7A-TAIL1`, `E7-P7A-SIG1`.
    6. Slice Acceptance & Official Milestone Commit:
       - Verify slice acceptance criteria, record evidence in `evidence.md`, refresh `actual-status.md`, commit with `feat(P7-A): implement goroutine log tailer and quota sensor pipeline`.
  - Implementation Gate: `P6-D` completed.
  - Acceptance:
    - Source: `internal/sensor/log_tailer.go`, `internal/sensor/signatures.go`, `internal/sensor/event_bus.go` implemented.
    - Runtime/UI: Goroutine tails logs without blocking; detection latency $\le 20\text{ms}$; CPU $< 0.05\%$.
    - DB/data: N/A.
    - Behavior test: 100% test pass rate across pattern matching and synthetic tailing.
    - Cleanup/quarantine: Context cancellation stops tailer cleanly.
    - Evidence IDs: `E7-P7A-TAIL1`, `E7-P7A-SIG1`.
    - Actual-status rows refreshed: `internal/sensor` -> `correct`.
  - Evidence Targets: `E7-P7A-TAIL1`, `E7-P7A-SIG1`.
  - Actual-status Update: Transition `internal/sensor` to `correct`.
  - Commit Boundary: Commit milestone `feat(P7-A): implement goroutine log tailer and quota sensor pipeline`.

- [ ] P7-B: Go-Native Failover Controller & Account Pool FSM
  - Goal: Implement the thread-safe `AccountPool` (`sync.RWMutex`), state machine (`ACTIVE`, `READY`, `QUOTA_EXHAUSTED`, `UNAUTH`, `ERROR`), round-robin / LRU scheduling with exponential backoff cooldowns, 15s hysteresis gate, sliding-window circuit breaker, and automated trigger of Phase P6 hot-swap.
  - Scope Boundary:
    - Editable: `internal/controller/pool_manager.go`, `internal/controller/scheduler.go`, `internal/controller/cooldown.go`, `internal/controller/hysteresis.go`, `internal/controller/controller_test.go`.
    - Inspect-only: `internal/manager/account_manager.go`, `internal/sensor/event_bus.go`.
    - Preserve-only: 15-second minimum dwell time, circuit breaker (max 4 switches per 120s), exponential cooldown (15m, 30m, 60m capped at 2h).
    - Out of scope: GUI rendering, desktop toast notifications.
  - Non-Goals: No cascading switches or infinite flapping loops during backend outages.
  - Pre-flight Questions:
    - Data source: Discovered accounts, `accounts_store/`, and incoming `QuotaExhaustedEvent` from sensor.
    - Display permission: N/A.
    - DB read flow: Read account list and cooldown timestamps from memory.
    - DB write flow: Update account states and invoke hot-swap.
    - Render location: In-memory state machine.
    - UI behavior flow: N/A.
    - Docker runtime: N/A.
    - Playwright target: N/A.
    - Behavior test: Test single account exhaustion, multi-account round-robin, hysteresis debouncing, and all-exhausted sleep state.
    - Cleanup/quarantine: Stop timers and release pool locks during shutdown.
    - External side effects: Triggers hot-swap execution.
    - N/A notes: Docker/Playwright N/A.
  - Work Steps:
    1. Code (Execute leaf atomic tasks in core source files only):
       - Task 1.1: Thread-Safe Account Pool & FSM in `internal/controller/pool_manager.go`
         + Action: Implement `AccountPool` guarded by `sync.RWMutex` managing account records and state transitions (`ACTIVE`, `READY`, `QUOTA_EXHAUSTED`, `UNAUTH`, `ERROR`) strictly enforcing that exactly one account is `ACTIVE`.
         + Inputs: Architect specification Section 4.
         + Allowed Edit Scope: `internal/controller/pool_manager.go`.
         + Outputs: Methods `GetActiveAccount()`, `GetReadyAccounts()`, `MarkExhausted()`, `MarkActive()`.
         + Verification Condition: Concurrent state read/write operations pass with `-race` flag.
         + Task Checkpoint: `wip(P7-B): task 1.1 - thread-safe account pool and FSM implemented`
       - Task 1.2: Round-Robin / LRU Scheduler & Cooldown Tracker in `internal/controller/`
         + Action: Implement scheduling algorithms (Round-Robin, LRU) and exponential backoff cooldown tracker (`15m -> 30m -> 60m -> 120m`) with background `time.Ticker` recovering accounts whose cooldown has elapsed.
         + Inputs: Architect specification Section 4.3 & Section 5.
         + Allowed Edit Scope: `internal/controller/scheduler.go`, `internal/controller/cooldown.go`.
         + Outputs: `SelectNextAccount() (*AccountRecord, error)` and `StartCooldownTicker(ctx)`.
         + Verification Condition: Selects eligible non-exhausted account; automatically restores expired accounts to `READY`.
         + Task Checkpoint: `wip(P7-B): task 1.2 - scheduler and exponential cooldown tracker implemented`
       - Task 1.3: Hysteresis Debounce Gate & Failover Execution Engine in `internal/controller/hysteresis.go`
         + Action: Implement 15-second minimum dwell time gate, sliding-window circuit breaker (max 4 switches per 120s), all-exhausted sleep mode with countdown wake timer, and automated triggering of `AccountManager.SwitchAccount`.
         + Inputs: Architect specification Section 5.1 & Section 5.2.
         + Allowed Edit Scope: `internal/controller/hysteresis.go`.
         + Outputs: Controller loop receiving `QuotaExhaustedEvent` and executing safe hot-swap.
         + Verification Condition: Suppresses flapping quota bursts within 15s window and triggers switch when gate opens.
         + Task Checkpoint: `wip(P7-B): task 1.3 - hysteresis debounce gate and failover execution engine implemented`
    2. Code Inspection & Lint:
       - Review diff, syntax, `go vet ./internal/controller/...`.
    3. Update Tests (Code first, test second):
       - Author `internal/controller/controller_test.go` with unit tests for FSM transitions, scheduling order, cooldown recovery, and circuit breaker activation.
    4. Build:
       - Run `go test -c -race ./internal/controller/...`.
    5. QA (Mini QA / Runtime Validation):
       - Run `go test -v -race ./internal/controller/...`.
       - Evidence target: `E7-P7B-POOL1`, `E7-P7B-HYST1`.
    6. Slice Acceptance & Official Milestone Commit:
       - Verify slice acceptance criteria, record evidence in `evidence.md`, refresh `actual-status.md`, commit with `feat(P7-B): implement Go-native failover controller, account pool FSM, and hysteresis engine`.
  - Implementation Gate: `P7-A` completed.
  - Acceptance:
    - Source: `internal/controller/pool_manager.go`, `internal/controller/scheduler.go`, `internal/controller/cooldown.go`, `internal/controller/hysteresis.go` implemented.
    - Runtime/UI: Pool transitions states accurately; hysteresis suppresses rapid switches; cooldown recovers accounts.
    - DB/data: N/A.
    - Behavior test: 100% tests pass under `-race` flag without data races.
    - Cleanup/quarantine: Background tickers cleanly stopped.
    - Evidence IDs: `E7-P7B-POOL1`, `E7-P7B-HYST1`.
    - Actual-status rows refreshed: `internal/controller` -> `correct`.
  - Evidence Targets: `E7-P7B-POOL1`, `E7-P7B-HYST1`.
  - Actual-status Update: Transition `internal/controller` to `correct`.
  - Commit Boundary: Commit milestone `feat(P7-B): implement Go-native failover controller, account pool FSM, and hysteresis engine`.

- [ ] P7-C: Native Windows System Tray & Toast HUD
  - Goal: Implement the native Windows system tray resident daemon with dynamic status color icons (🟢 Healthy, 🟡 Active/Single, 🔴 All Exhausted), Windows 10/11 Toast Notification bridge, and global hotkey listener (`Win+Alt+S`).
  - Scope Boundary:
    - Editable: `internal/tray/tray_daemon.go`, `internal/tray/notifications.go`, `internal/tray/hotkeys.go`, `internal/tray/tray_test.go`, `cmd/switcher/main.go`.
    - Inspect-only: `internal/controller/`, `internal/ui/`.
    - Preserve-only: Discrete notifications; no focus stealing or modal alerts.
    - Out of scope: Modifying core switching algorithms.
  - Non-Goals: No heavy webview notifications; native Windows Toast API and Win32 message loop only.
  - Pre-flight Questions:
    - Data source: Controller account pool state and failover events.
    - Display permission: Native Windows System Tray and Action Center notifications.
    - DB read flow: Read active account email and pool counts.
    - DB write flow: N/A.
    - Render location: Windows Taskbar System Tray & Action Center Toasts.
    - UI behavior flow: Tray icon runs in background -> right click displays menu (Status, Switch Account, Pause/Resume, Open Dashboard, Exit) -> auto-switch triggers toast: "Auto-switched to userB in 0.8s" -> `Win+Alt+S` opens GUI.
    - Docker runtime: N/A.
    - Playwright target: N/A.
    - Behavior test: Verify tray menu creation, test toast notification dispatch without UI hang, test hotkey registration and unregistration.
    - Cleanup/quarantine: Unregister hotkeys (`UnregisterHotKey`) and remove tray icon on application exit.
    - External side effects: Registers Win32 global hotkey `Win+Alt+S`.
    - N/A notes: Docker/Playwright N/A.
  - Work Steps:
    1. Code (Execute leaf atomic tasks in core source files only):
       - Task 1.1: System Tray Resident Application in `internal/tray/tray_daemon.go`
         + Action: Implement system tray daemon using Fyne `desktop.App` / native Win32 `Shell_NotifyIconW`, supporting dynamic icons (🟢 Green: $\ge 2$ accounts ready; 🟡 Yellow: 1 account ready / failover in progress; 🔴 Red: all accounts exhausted with countdown tooltip), and context menu with pool inventory, pause/resume, and exit.
         + Inputs: Architect specification Section 2 & Section 6.
         + Allowed Edit Scope: `internal/tray/tray_daemon.go`.
         + Outputs: `TrayDaemon` struct and message loop.
         + Verification Condition: Tray icon appears cleanly in notification area and updates color on state change.
         + Task Checkpoint: `wip(P7-C): task 1.1 - system tray resident daemon implemented`
       - Task 1.2: Windows 10/11 Toast Notification Bridge in `internal/tray/notifications.go`
         + Action: Implement discrete Windows Toast notification dispatcher using WinRT PowerShell / Windows Toast API, alerting the developer on automated switch completion (e.g., "⚡ Antigravity Auto-Switched: Swapped to userB@gmail.com in 0.8s") without stealing keyboard focus or popping intrusive dialogs.
         + Inputs: Architect specification Section 2 & Section 5.3.
         + Allowed Edit Scope: `internal/tray/notifications.go`.
         + Outputs: `NotifyFailover(oldEmail, newEmail string, duration time.Duration) error`.
         + Verification Condition: Dispatches native toast notification cleanly within 50ms.
         + Task Checkpoint: `wip(P7-C): task 1.2 - Windows toast notification bridge implemented`
       - Task 1.3: Global Hotkey Listener (`Win+Alt+S`) in `internal/tray/hotkeys.go`
         + Action: Implement Win32 `RegisterHotKey` listener for `MOD_WIN | MOD_ALT` + `VK_S` (`Win+Alt+S`) to toggle switcher GUI visibility instantly from any editor window, with clean unregistration on exit.
         + Inputs: Win32 API documentation (`User32.dll`).
         + Allowed Edit Scope: `internal/tray/hotkeys.go`.
         + Outputs: `RegisterGlobalHotkey(callback func()) error` and `UnregisterGlobalHotkey()`.
         + Verification Condition: Pressing `Win+Alt+S` invokes callback; unregisters cleanly on shutdown.
         + Task Checkpoint: `wip(P7-C): task 1.3 - global hotkey listener implemented`
    2. Code Inspection & Lint:
       - Review diff, syntax, `go vet ./internal/tray/...`.
    3. Update Tests (Code first, test second):
       - Author `internal/tray/tray_test.go` verifying notification payloads and hotkey registration lifecycle.
    4. Build:
       - Run `go build -o antigravity-switch-tray.exe ./cmd/switcher`.
    5. QA (Mini QA / Runtime Validation):
       - Run `go test -v ./internal/tray/...`.
       - Evidence target: `E7-P7C-TRAY1`, `E7-P7C-TOAST1`.
    6. Slice Acceptance & Official Milestone Commit:
       - Verify slice acceptance criteria, record evidence in `evidence.md`, refresh `actual-status.md`, commit with `feat(P7-C): implement native Windows system tray daemon, toast notifications, and global hotkeys`.
  - Implementation Gate: `P7-B` completed.
  - Acceptance:
    - Source: `internal/tray/tray_daemon.go`, `internal/tray/notifications.go`, `internal/tray/hotkeys.go` implemented.
    - Runtime/UI: Tray icon resides in notification area with dynamic status dots; toast alerts dispatch cleanly; `Win+Alt+S` toggles GUI.
    - DB/data: N/A.
    - Behavior test: 100% tests pass in tray package.
    - Cleanup/quarantine: Tray icon removed and hotkeys unregistered on shutdown.
    - Evidence IDs: `E7-P7C-TRAY1`, `E7-P7C-TOAST1`.
    - Actual-status rows refreshed: `internal/tray` -> `correct`.
  - Evidence Targets: `E7-P7C-TRAY1`, `E7-P7C-TOAST1`.
  - Actual-status Update: Transition `internal/tray` to `correct`.
  - Commit Boundary: Commit milestone `feat(P7-C): implement native Windows system tray daemon, toast notifications, and global hotkeys`.

- [ ] P7-D: End-to-End Simulation QA & Quota Storm Stress Testing
  - Goal: Build end-to-end simulation test harness, execute Quota Storm stress testing, verify $< 1.0\text{s}$ failover latency benchmark, and confirm 100% IDE context preservation (open tabs, dirty buffers, terminals unaffected).
  - Scope Boundary:
    - Editable: `tests/e2e/quota_storm_test.go`, `tests/e2e/e2e_test.go`, `docs/plans/2026-10-14-antigravity-multi-account-switcher/`.
    - Inspect-only: Entire codebase (`cmd/`, `internal/`).
    - Preserve-only: Zero disruption to Antigravity IDE UI or language server state.
    - Out of scope: Modifying production application binaries.
  - Non-Goals: No intrusive process killing or memory tampering during test runs.
  - Pre-flight Questions:
    - Data source: Mock language server, synthetic log generator, and 3+ simulated test accounts.
    - Display permission: N/A.
    - DB read flow: Read synthetic log lines; read token store.
    - DB write flow: Write test credentials to isolated test environment.
    - Render location: Test harness console and benchmark ledger.
    - UI behavior flow: N/A.
    - Docker runtime: N/A.
    - Playwright target: N/A.
    - Behavior test: End-to-end failover cycle simulation under rapid 429 quota exhaustion bursts.
    - Cleanup/quarantine: All synthetic logs, mock servers, and test accounts purged after test suite completion.
    - External side effects: Zero side effects outside test harness.
    - N/A notes: Docker/Playwright N/A.
  - Work Steps:
    1. Code (Execute leaf atomic tasks in test harness only):
       - Task 1.1: End-to-End Integration Simulation Test Harness in `tests/e2e/`
         + Action: Build an automated simulation test harness spinning up a mock language server with `GetUserStatus`, `AuthLogout`, and reauth endpoints, generating synthetic `language_server.log` entries, and validating end-to-end hot-swap loop.
         + Inputs: Architect specification Section 3 & Section 6.
         + Allowed Edit Scope: `tests/e2e/e2e_test.go`.
         + Outputs: Complete automated E2E simulation harness.
         + Verification Condition: Executes end-to-end hot-swap loop without error.
         + Task Checkpoint: `wip(P7-D): task 1.1 - E2E simulation harness implemented`
       - Task 1.2: Quota Storm Stress Test & Hysteresis Verification
         + Action: Implement Quota Storm test injecting rapid bursts of HTTP 429 and `RESOURCE_EXHAUSTED` errors into synthetic logs, asserting clean rotation through 3 accounts, strict enforcement of 15s hysteresis gate, and transition to graceful sleep when all accounts are exhausted.
         + Inputs: Architect specification Section 5.1 & Section 5.2.
         + Allowed Edit Scope: `tests/e2e/quota_storm_test.go`.
         + Outputs: Comprehensive Quota Storm test suite.
         + Verification Condition: 100% pass rate across multi-account storm bursts; zero race conditions or cascading failures.
         + Task Checkpoint: `wip(P7-D): task 1.2 - quota storm stress test and hysteresis verified`
       - Task 1.3: Failover Latency & IDE Context Preservation Benchmark
         + Action: Measure total failover latency (detection -> decision -> closed-loop hot-swap -> verification), asserting total cycle $< 1.0\text{ second}$, and execute validation proving 100% editor context preservation (unsaved Monaco editor buffers and active terminals unaffected).
         + Inputs: Benchmark test runner.
         + Allowed Edit Scope: `docs/plans/2026-10-14-antigravity-multi-account-switcher/2026-10-14-antigravity-multi-account-switcher-benchmark.md`.
         + Outputs: Recorded failover benchmark metrics in `benchmark.md`.
         + Verification Condition: Total failover latency $< 1.0\text{s}$ confirmed.
         + Task Checkpoint: `wip(P7-D): task 1.3 - failover latency benchmark and context preservation verified`
    2. Code Inspection & Lint:
       - Review test harness code for clean resource cleanup and determinism.
    3. Update Tests (Code first, test second):
       - Ensure all E2E test suites run deterministically in CI/local runs.
    4. Build:
       - Run `go test -c ./tests/e2e/...`.
    5. QA (Mini QA / Runtime Validation):
       - Run `go test -v -race ./tests/e2e/...`.
       - Evidence target: `E7-P7D-STORM1`, `E7-P7D-LAT1`.
    6. Slice Acceptance & Official Milestone Commit:
       - Verify slice acceptance criteria, record evidence in `evidence.md`, refresh `actual-status.md`, commit with `feat(P7-D): verify end-to-end autonomous failover, quota storm stress tests, and context preservation`.
  - Implementation Gate: `P7-C` completed.
  - Acceptance:
    - Source: `tests/e2e/` simulation test suite implemented.
    - Runtime/UI: Quota Storm executes cleanly; 15s hysteresis gate holds; all-exhausted sleep mode verified.
    - DB/data: Isolated test accounts cycled and purged.
    - Behavior test: Total failover cycle $< 1.0\text{s}$; 100% IDE context preservation verified.
    - Cleanup/quarantine: All synthetic logs and mock servers cleaned up.
    - Evidence IDs: `E7-P7D-STORM1`, `E7-P7D-LAT1`.
    - Actual-status rows refreshed: `tests/e2e` -> `correct`.
  - Evidence Targets: `E7-P7D-STORM1`, `E7-P7D-LAT1`.
  - Actual-status Update: Transition `tests/e2e` to `correct`.
  - Commit Boundary: Commit milestone `feat(P7-D): verify end-to-end autonomous failover, quota storm stress tests, and context preservation`.

- [ ] Pn-A: Call supervisor for the implemented-plan acceptance loop.
  - Goal: Verify completed plan work against the accepted plan, actual-status decisions, evidence, benchmark, changed files, and validation results before closure.
  - Work Steps:
    1. Call the supervisor skill/agent to review the completed plan work.
    2. If supervisor fails the work, return to the responsible implementation lane for the failed scope only.
    3. Re-run supervisor review after fixes.
    4. Repeat until supervisor passes or records a blocker.
  - Implementation Gate: All planned implementation phases (P1 through P7) must be completed or explicitly blocked.
  - Acceptance: Supervisor review passes, or plan records a blocker with evidence.

- [ ] Pn-B: Remove dead work created during this plan.
  - Goal: Ensure final diff contains only artifacts that serve the accepted plan.
  - Work Steps:
    1. Review files, sections, temp files, and plan artifacts created or modified during this plan.
    2. Remove or rewrite any artifact made obsolete by actual-status findings or user corrections.
    3. Verify no rejected approach, stale placeholder, or dead helper artifact remains in the final diff.
    4. Call supervisor to review dead-work cleanup.
  - Implementation Gate: Only remove artifacts created by this plan unless broader cleanup is explicitly approved.
  - Acceptance: Final `git status` contains no dead plan-created artifacts, and supervisor passes cleanup.

- [ ] Pn-C: Close the plan.
  - Goal: Finish validation, evidence, benchmark, final commit, and final status.
  - Work Steps:
    1. Run required final validation for the accepted scope.
    2. Record final validation and benchmark metrics in companion ledgers.
    3. Record final commit evidence in `evidence.md`.
    4. Verify clean worktree state.
  - Implementation Gate: `Pn-A` and `Pn-B` must pass.
  - Acceptance: Final evidence recorded, required commits exist, and worktree state is known.

## Risk Notes

1. **Language Server Process Lifecycle:** If Antigravity is closed or restarted by the user, `language_server.exe` PID and loopback port will change. The implementation in `core/api_connector.py` dynamically resolves the process on each call rather than caching static values, preventing stale socket errors.
2. **Ephemeral Port Allocation:** Antigravity starts `language_server.exe` with `--https_server_port 0`, which lets the OS assign an ephemeral port. Discovering the listening port via OS TCP connection queries (`Get-NetTCPConnection` or `netstat`) ensures robust connection establishment regardless of port churn.
3. **Blacklist Keyword Protection:** Strict filtering in `scanner/chrome_scanner.py` guarantees that accounts associated with blacklisted domains or terms (`anvien`) are never displayed or processed.
4. **Credential Safety:** Writing to `LegacyGeneric:target=gemini:antigravity` is guarded by payload validation to prevent corrupting the active credential format.
5. **CGo-Free Windows Syscall Compatibility:** Direct Win32 `Advapi32.dll` system calls via `golang.org/x/sys/windows` avoid MSVC runtime dependencies and maintain standard cross-compilation capability without requiring a local C compiler.
6. **Fyne GUI OpenGL Hardware Acceleration:** In headless CI or minimal virtual machine environments lacking OpenGL drivers, Fyne software rendering flags (`-tags=mobile` or software rasterizer) ensure execution stability.
7. **Log Tailer Offset Resilience:** If Antigravity rotates or truncates `language_server.log`, the sensor detects the file size shrink and resets byte offset to 0 to prevent missed quota signatures or out-of-bounds seeks.
8. **Hotkey Conflicts & Cleanup:** Win32 `RegisterHotKey` for `Win+Alt+S` returns an error if already registered by another utility; the daemon handles this gracefully by logging an alert and falling back to System Tray click activation without crashing.
