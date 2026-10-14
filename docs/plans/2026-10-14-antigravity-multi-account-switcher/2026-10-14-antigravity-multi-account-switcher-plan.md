# Antigravity Multi-Account Switcher Plan

## Metadata

- Date: `2026-10-14`
- Status: `draft`
- Plan: `docs/plans/2026-10-14-antigravity-multi-account-switcher/2026-10-14-antigravity-multi-account-switcher-plan.md`
- Evidence: `docs/plans/2026-10-14-antigravity-multi-account-switcher/2026-10-14-antigravity-multi-account-switcher-evidence.md`
- Benchmark: `docs/plans/2026-10-14-antigravity-multi-account-switcher/2026-10-14-antigravity-multi-account-switcher-benchmark.md`
- Actual status: `docs/plans/2026-10-14-antigravity-multi-account-switcher/2026-10-14-antigravity-multi-account-switcher-actual-status.md`

## Goal

Re-architect and implement the Antigravity Multi-Account Switcher to provide seamless, persistent multi-account switching for Google Antigravity on Windows. The tool functions as the permanent external memory for Antigravity accounts using individual JSON token files (`accounts_store/<email>.json`), dynamically discovers accounts from Google Chrome while enforcing strict blacklist filtering, and executes the Owner's mandated 4-step closed-loop internal API switching architecture:
1. `GetUserStatus()` to synchronize live email in `language_server.exe` RAM;
2. `AuthLogout()` to cleanly purge active tokens from RAM and terminate the session of Account A;
3. Inject Account B's token into Windows Credential Manager (`LegacyGeneric:target=gemini:antigravity`);
4. Trigger Re-Auth / Reload to load Token B into RAM ("Auth succeeded: Tài khoản B").

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

## Scope

- **Editable Surfaces:**
  - `core/account_store.py`: Individual JSON file management for accounts.
  - `core/credential_manager.py`: Windows Credential Manager API ctypes interface.
  - `core/api_connector.py`: New module for `language_server.exe` discovery (port & CSRF token), `GetUserStatus()`, `AuthLogout()`, and Re-Auth trigger.
  - `core/session_reloader.py`: Window focus and reload dispatch coordinated with API re-auth.
  - `core/state_resolver.py`: Multi-account state machine (ACTIVE, READY, UNAUTH) and 4-step hot-swap loop.
  - `scanner/chrome_scanner.py`: Chrome profile account discovery with keyword blacklist.
  - `ui/app_window.py`: Tkinter dark-theme GUI with live RAM status and switching controls.
  - `tests/test_all.py`: Comprehensive test suite.
  - `main.py`: Application launcher.
- **Inspect-Only Surfaces:**
  - `%LOCALAPPDATA%\Google\Chrome\User Data`: Chrome user profile preferences and Local State.
  - `language_server.exe` process table & TCP listening ports (`127.0.0.1:59226/59227`).
  - `%APPDATA%\Antigravity\logs\language_server.log`: Diagnostic logs confirming auth transitions.
- **Preserve-Only Surfaces:**
  - `LegacyGeneric:target=gemini:antigravity` target naming convention.
  - Dark mode color palette and UI ergonomics.
  - Strict blacklist keyword filtering (`["anvien"]`).
- **Out of Scope:**
  - Antigravity IDE core binaries and language server code modifications.
  - Production OAuth server or web-based token broker.

## Non-Goals

- No cloud servers, web authentication flows, or external proxy servers.
- No modifying Chrome profile data or browser configurations.
- No patching of `language_server.exe` binary.
- No anvien usage during implementation.

## Requirements

1. **Persistent Multi-Account Storage:** Each discovered/authenticated account is persisted as an independent file `accounts_store/<email>.json` containing full token payloads (`access_token`, `refresh_token`, `id_token`, `expiry`, `token_type`).
2. **Chrome Discovery with Blacklist:** Discovers all Google accounts logged into Chrome profiles; automatically filters out emails matching blacklist keywords (`anvien`); automatically removes accounts logged out of Chrome.
3. **4-Step Closed-Loop Internal API Switching:**
   - Step 1: Query `GetUserStatus()` from Antigravity `language_server.exe` to retrieve the true active email in RAM.
   - Step 2: Issue `AuthLogout()` to language server API to flush active tokens from RAM and cancel Account A's session.
   - Step 3: Write Account B's JSON token to Windows Credential Manager `LegacyGeneric:target=gemini:antigravity`.
   - Step 4: Trigger Re-Auth / Reload to prompt Antigravity to load Token B into RAM, verified by `Auth succeeded`.
4. **State Machine:**
   - 🟢 `ACTIVE [ĐANG DÙNG]`: verified matching true RAM identity.
   - 🟢 `READY [⚡ Chuyển sang]`: present in Chrome AND saved in `accounts_store/<email>.json`.
   - 🟡 `UNAUTH [Chưa lưu vào kho]`: present in Chrome but missing from `accounts_store/`.
5. **Modern Desktop UI:** Tkinter Dark Mode Mini Window with `Always on Top`, 1-Click `[⚡ Chuyển sang]`, `[💾 Lưu phiên hiện tại]`, and `[🔄 Làm mới Chrome]`.

## Acceptance Criteria

1. `accounts_store/` stores individual valid JSON files per email without corruption.
2. Chrome scanner discovers all local Google accounts while strictly excluding blacklisted emails.
3. `api_connector.py` dynamically identifies `language_server.exe` PID, port, and `--csrf_token` argument without hardcoding.
4. Calling `GetUserStatus()` returns the exact email running in RAM.
5. Calling `AuthLogout()` cleanly purges the session from language server RAM.
6. Hot-swap execution transitions Antigravity from Account A to Account B within < 1 second.
7. Subsequent prompt activity uses Account B's quota.
8. 100% of unit and integration tests pass cleanly.

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

- [ ] P3-B: Multi-Account State Machine Resolution (ACTIVE, READY, UNAUTH)
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

- [ ] P3-C: 4-Step Closed-Loop Hot-Swap Coordinator
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

- [ ] P4-A: Real-time Account List & Visual State Badging
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

- [ ] P4-B: 1-Click Hot-Swap Trigger & Status Feedback
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

- [ ] P4-C: Session Persistence & Manual Chrome Sync Controls
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

- [ ] P5-A: Comprehensive Unit & Integration Test Suite
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

- [ ] P5-B: End-to-End Switching Cycle & Robustness Verification
  - Goal: Run full end-to-end regression validation, verifying 100% pass rate, zero regressions, and accurate benchmark records.
  - Scope Boundary:
    - Editable: `tests/test_all.py`, `docs/plans/2026-10-14-antigravity-multi-account-switcher/`.
    - Inspect-only: Entire repository.
    - Preserve-only: N/A.
    - Out of scope: Source modifications.
  - Non-Goals: N/A.
  - Pre-flight Questions:
    - Data source: Test suites and benchmarks.
    - Display permission: N/A.
    - DB read flow: Full suite.
    - DB write flow: N/A.
    - Render location: Console.
    - UI behavior flow: N/A.
    - Docker runtime: N/A.
    - Playwright target: N/A.
    - Behavior test: Full suite regression.
    - Cleanup/quarantine: All temp files removed.
    - External side effects: N/A.
    - N/A notes: Docker/Playwright N/A.
  - Work Steps:
    1. Code (Execute leaf atomic tasks in test harness only):
       - Task 1.1: Regression Suite Execution
         + Action: Execute full automated test suite `python -m unittest tests/test_all.py`.
         + Inputs: Complete codebase.
         + Allowed Edit Scope: `tests/test_all.py`.
         + Outputs: 100% pass output.
         + Verification Condition: Zero errors, zero failures.
         + Task Checkpoint: `wip(P5-B): task 1.1 - full regression suite verified`
    2. Code Inspection & Lint:
       - Review test execution log.
    3. Update Tests (Code first, test second):
       - Confirm all test targets green.
    4. Build:
       - Compile all project files: `python -m compileall core scanner ui tests main.py`.
    5. QA (Mini QA / Runtime Validation):
       - Verify test timings and pass counts.
       - Evidence target: `E5-P5B-REG1`.
    6. Slice Acceptance & Official Milestone Commit:
       - Verify slice acceptance criteria, record evidence in `evidence.md`, refresh `actual-status.md`, commit with `test(P5-B): verify complete switching regression suite`.
  - Implementation Gate: `P5-A` completed.
  - Acceptance:
    - Source: All modules compile cleanly.
    - Runtime/UI: Zero regressions.
    - DB/data: Consistent.
    - Behavior test: 100% test pass rate.
    - Cleanup/quarantine: Clean worktree.
    - Evidence IDs: `E5-P5B-REG1`.
    - Actual-status rows refreshed: `tests/test_all.py` -> `correct`.
  - Evidence Targets: `E5-P5B-REG1`.
  - Actual-status Update: Transition `tests/test_all.py` to `correct`.
  - Commit Boundary: Commit milestone `test(P5-B): verify complete switching regression suite`.

- [ ] Pn-A: Call supervisor for the implemented-plan acceptance loop.
  - Goal: Verify completed plan work against the accepted plan, actual-status decisions, evidence, benchmark, changed files, and validation results before closure.
  - Work Steps:
    1. Call the supervisor skill/agent to review the completed plan work.
    2. If supervisor fails the work, return to the responsible implementation lane for the failed scope only.
    3. Re-run supervisor review after fixes.
    4. Repeat until supervisor passes or records a blocker.
  - Implementation Gate: All planned implementation phases (P1 through P5) must be completed or explicitly blocked.
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
