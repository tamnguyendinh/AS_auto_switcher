# Antigravity Multi-Account Switcher Actual Status

Title: Antigravity Multi-Account Switcher Actual Status
Date: 2026-10-14
Status: P0 Complete
Companion plan: `docs/plans/2026-10-14-antigravity-multi-account-switcher/2026-10-14-antigravity-multi-account-switcher-plan.md`
Companion rules: `docs/plans/2026-10-14-antigravity-multi-account-switcher/2026-10-14-antigravity-multi-account-switcher-plan.md#rules`
Companion evidence: `docs/plans/2026-10-14-antigravity-multi-account-switcher/2026-10-14-antigravity-multi-account-switcher-evidence.md`
Companion benchmark: `docs/plans/2026-10-14-antigravity-multi-account-switcher/2026-10-14-antigravity-multi-account-switcher-benchmark.md`

## Purpose

This file records the real current state before implementation.

Implementation must not start until the target scope has a completed status row, evidence IDs, and a downstream plan decision.

This file does not replace `evidence.md`. It classifies current state from evidence.

Use exact evidence IDs from `evidence.md`, such as `E0-P0A-SRC1`, not broad section IDs such as `E0` or `E1`.

## Freshness / Refresh Rules

This actual-status file is a living current-state record, not a one-time P0 snapshot.

P0 records the baseline before implementation. After implementation begins, keep the Current Status Matrix updated so the next agent can trust it as the latest repo reality.

Update this file:

- after each completed implementation slice;
- before starting the next phase if repo state changed;
- whenever evidence changes a current-state classification;
- whenever the next phase's status assumptions, next action, or work steps need updating because reality differs from the previous status.

When refreshing status:

- update only the rows affected by the completed work or new evidence;
- use explicit transitions such as `missing -> correct`, `partial -> correct`, `fake-or-stub -> removed`, or `unbound -> bound-correct`;
- append a Status Refresh Log row instead of deleting history;
- keep detailed proof in `evidence.md`; store only classifications, evidence IDs, touch mode, and plan consequences here.

## Scope

Target scope:

- Complete Antigravity Multi-Account Switcher with persistent JSON account storage (`accounts_store/<email>.json`), automated Chrome discovery, and the Owner's mandated 4-step closed-loop internal API switching architecture (`GetUserStatus`, `AuthLogout`, Windows Credential injection, and Re-Auth / Reload into RAM).
- Core modules: `core/account_store.py`, `core/credential_manager.py`, `core/session_reloader.py`, `core/api_connector.py`, `core/state_resolver.py`, `scanner/chrome_scanner.py`, `ui/app_window.py`, `tests/test_all.py`.

Out of scope:

- Modifying Antigravity IDE binaries, Electron runtime, or language_server.exe executable on disk.
- Web/Production OAuth server or cloud auth brokering (Prototype Auth only).
- Bypassing or modifying Chrome browser user profiles on disk other than read-only discovery.
- Use of anvien.

## Relationship / Impact Evidence

| Unit / File / Surface | File Detail Evidence | Related File Count | Relationship Summary | Impact Note |
|-----------------------|----------------------|--------------------|----------------------|-------------|
| `core/account_store.py` | `E0-P0A-SRC1` | 3 | Imported by `state_resolver.py` and `tests/test_all.py` | low |
| `core/credential_manager.py` | `E0-P0A-SRC2`, `E0-P0A-CRED1` | 3 | Imported by `state_resolver.py` and `tests/test_all.py` | low (preserves Windows DPAPI credential format) |
| `core/session_reloader.py` | `E0-P0A-SRC3` | 2 | Imported by `state_resolver.py` | low |
| `core/api_connector.py` | `E0-P0A-PROC1`, `E0-P0A-PROC2`, `E0-P0A-LOG1` | 3 | New component; will be imported by `state_resolver.py` and tested by `tests/test_all.py` | medium (central to 4-step switching cycle) |
| `scanner/chrome_scanner.py` | `E0-P0A-SRC5`, `E0-P0A-SEC1` | 2 | Imported by `state_resolver.py` and `tests/test_all.py` | low |
| `core/state_resolver.py` | `E0-P0A-SRC4` | 4 | Orchestrates store, credential, scanner, reloader, imported by `ui/app_window.py` | medium |
| `ui/app_window.py` | `E0-P0A-SRC6` | 2 | Imported by `main.py` | medium |
| `tests/test_all.py` | `E0-P0A-SRC7`, `E0-P0A-TEST1` | 6 | Tests all core modules and scanner | low |

## Status Rules

| Status | Meaning | Allowed next action |
|--------|---------|---------------------|
| `correct` | Already behaves as required. | Preserve. Add evidence or tests only if needed. |
| `partial` | Some required behavior exists, but gaps remain. | Change only the missing parts. Preserve correct parts. |
| `wrong` | Current behavior, source, or contract is incorrect. | Replace with required behavior. Record the exact reason. |
| `missing` | Required behavior, source, or contract does not exist. | Implement the missing piece only. |
| `unbound` | Surface exists but is not wired to the real source, flow, or contract. | Bind to the real source only. Preserve approved surface. |
| `fake-or-stub` | Prototype, demo, mock, fallback, or placeholder data is being used as real behavior. | Remove fake behavior or replace it with an approved truthful state. |
| `blocked` | Source, authority, contract, or required evidence is unclear. | Stop. Do not implement until resolved. |

## Current Status Matrix

| Unit | Current State | Required State | Status | Relationship Count | Evidence | Next Plan Decision |
|------|---------------|----------------|--------|--------------------|----------|--------------------|
| `core/account_store.py` | Stores accounts in individual JSON files under `accounts_store/<email>.json`. | Robust JSON persistence with filename sanitation and list capability. | `correct` | 3 related files | `E0-P0A-SRC1` | Preserve in P1-A, verify in test suite |
| `core/credential_manager.py` | Advapi32 ctypes wrapper for `LegacyGeneric:target=gemini:antigravity`. | Reliable read/write of raw and JSON tokens to Windows Credential Manager. | `correct` | 3 related files | `E0-P0A-SRC2`, `E0-P0A-CRED1` | Preserve in P1-B, verify in test suite |
| `core/session_reloader.py` | Window title search, keystroke dispatch, and coordinated window/session reload implemented. | Keystroke reload wrapper coordinated with API re-auth trigger. | `correct` | 2 related files | `E0-P0A-SRC3`, `E2-P2C-REAUTH1` | Preserve in Phase P3 / wire into state resolver |
| `core/api_connector.py` | Process discovery, PID, port, `--csrf_token` extraction, and internal API clients (`GetUserStatus`, `AuthLogout`, `trigger_reauth`) implemented. | Discovers `language_server.exe` PID, port, `--csrf_token`; calls `GetUserStatus()`, `AuthLogout()`, and `trigger_reauth()`. | `correct` | 3 related files | `E2-P2A-DISC1`, `E2-P2B-API1`, `E2-P2B-API2`, `E2-P2C-REAUTH1` | Preserve in Phase P3 / wire into state resolver |
| `scanner/chrome_scanner.py` | Reads Chrome profile preferences and Local State with `anvien` blacklist filtering. | Robust extraction of signed-in Chrome emails excluding blacklisted terms. | `correct` | 2 related files | `E0-P0A-SRC5`, `E0-P0A-SEC1` | Preserve in P3-A, wire into state resolver |
| `core/state_resolver.py` | Resolves accounts to ACTIVE, READY, UNAUTH; `switch_account` only writes credential and sends `Ctrl+R`. | Resolves accounts using RAM status from `GetUserStatus()`; coordinates full 4-step hot-swap loop. | `partial` | 4 related files | `E0-P0A-SRC4` | Update in P3-B, P3-C |
| `ui/app_window.py` | Tkinter UI with account list, status dots, and action buttons; lacks live RAM email sync and API feedback. | Displays live RAM account status, real-time 4-step switching progress, error feedback, and manual sync. | `partial` | 2 related files | `E0-P0A-SRC6` | Update in P4-A, P4-B, P4-C |
| `tests/test_all.py` | 6 unit tests passing for store, jwt, blacklist, and live credential read. | Comprehensive test suite covering API connector, port discovery, 4-step switching loop, and edge cases. | `partial` | 6 related files | `E0-P0A-SRC7`, `E0-P0A-TEST1` | Expand in P5-A, P5-B |

## Status Refresh Log

| Refresh | Date | Repo Basis | Changed Scope | Status Changes | Evidence | Next Phase Update |
|---------|------|------------|----------------|----------------|----------|-------------------|
| R0 | 2026-10-14 | baseline before P0 | Entire repository | Initial classification: 3 correct, 4 partial, 1 missing | `E0-P0A-SRC1..SRC8`, `E0-P0A-PROC1..PROC2`, `E0-P0A-TEST1` | Proceed to implementation slices P1 through P5 |
| R1 | 2026-10-14 | commit 2088e77 (Slice P1-A) | `core/account_store.py`, `tests/test_all.py` | P1-A complete: atomic JSON file operations, filename sanitation, schema validation verified | `E1-P1A-STORE1`, `E1-P1A-STORE2` | Proceed to Slice P1-B (Active Session Capture & Credential Synchronization) |
| R2 | 2026-10-14 | commit 50149b2 (Slice P1-B) | `core/credential_manager.py`, `tests/test_all.py` | P1-B complete: Windows Credential Manager integration hardened (validation, error resilience, safe delete) and verified | `E1-P1B-CRED1`, `E1-P1B-AUTO1` | Proceed to Phase P2 (Language Server API Connector) |
| R3 | 2026-10-14 | commit 04bdf95 (Slice P2-A) | `core/api_connector.py`, `tests/test_all.py` | P2-A complete: language server discovery (PID, port, CSRF token extraction) implemented and verified | `E2-P2A-DISC1` | Proceed to Slice P2-B (Internal Auth API Client: GetUserStatus & AuthLogout) |
| R4 | 2026-10-14 | commit b7d550b (Slice P2-B) | `core/api_connector.py`, `tests/test_all.py` | P2-B complete: GetUserStatus and AuthLogout HTTP API clients implemented and verified | `E2-P2B-API1`, `E2-P2B-API2` | Proceed to Slice P2-C (Re-Auth Trigger & RAM Session Reload) |
| R5 | 2026-10-14 | commit c8d3423 (Slice P2-C) | `core/api_connector.py`, `core/session_reloader.py`, `tests/test_all.py` | P2-C complete: Re-Auth trigger and RAM session reloader implemented and verified | `E2-P2C-REAUTH1` | Proceed to Phase P3 (Account Discovery & State Management) |

## Phase Touch Map

| Unit / File / Surface | Plan-Relevant Relationship File | Relationship to Target | Plan Item | Touch Mode | Evidence | Constraint |
|-----------------------|---------------------------------|------------------------|-----------|------------|----------|------------|
| `core/account_store.py` | `core/state_resolver.py` | consumer | P1-A | preserve-only | `E0-P0A-SRC1` | Keep existing JSON schema intact |
| `core/credential_manager.py` | `core/state_resolver.py` | consumer | P1-B | preserve-only | `E0-P0A-SRC2` | Maintain Windows target `LegacyGeneric:target=gemini:antigravity` |
| `core/api_connector.py` | `core/state_resolver.py` | consumer | P2-A, P2-B | edit (new) | `E0-P0A-PROC1` | Strictly use process args / loopback connection; no network broadcast |
| `core/session_reloader.py` | `core/api_connector.py` | companion | P2-C | edit | `E0-P0A-SRC3` | Preserve existing window focus logic |
| `scanner/chrome_scanner.py` | `core/state_resolver.py` | consumer | P3-A | preserve-only | `E0-P0A-SRC5` | Strict blacklist enforcement must be preserved |
| `core/state_resolver.py` | `ui/app_window.py` | consumer | P3-B, P3-C | edit | `E0-P0A-SRC4` | Wire 4-step loop seamlessly |
| `ui/app_window.py` | `main.py` | entry point | P4-A, P4-B | edit | `E0-P0A-SRC6` | Preserve Tkinter dark theme aesthetics |
| `tests/test_all.py` | `core/`, `scanner/` | test target | P5-A, P5-B | edit | `E0-P0A-SRC7` | All existing 6 tests must continue to pass |

## Detailed Findings

### `core/account_store.py`

Current state:
Provides class `AccountStore` reading and writing files under `accounts_store/<email>.json`. Contains `store_account`, `get_account_token`, `has_account`, `remove_account`, `list_stored_emails`. Filename sanitization uses safe regex.

Required state:
```text
Clean JSON multi-account persistence layer ensuring isolated JSON files per email without overwriting or leaking tokens.
```

Evidence:
- `E0-P0A-SRC1`: Source code inspection of `core/account_store.py`.
- `E0-P0A-TEST1`: Unit test `test_account_store_json_files` passes cleanly.

Relationship and impact:
- Related file count: 3
- Relationship summary: Consumed by `core/state_resolver.py`, tested by `tests/test_all.py`.
- Impact note: Low blast radius.

Classification:
`correct`

Allowed next action:
Preserve file structure and contracts in P1-A.

Forbidden next action:
Do not alter JSON token schema or remove file-based persistence.

---

### `core/credential_manager.py`

Current state:
Direct `Advapi32.dll` wrapper using `ctypes` implementing `CredReadW`, `CredWriteW`, and `CredDeleteW`. Targets `LegacyGeneric:target=gemini:antigravity` with user `antigravity`.

Required state:
```text
Accurate read/write bridge to Windows Credential Manager for the active Antigravity OAuth token.
```

Evidence:
- `E0-P0A-SRC2`: Source code inspection of `core/credential_manager.py`.
- `E0-P0A-CRED1`: Live credential read verified on Windows host.

Relationship and impact:
- Related file count: 3
- Relationship summary: Consumed by `state_resolver.py`, tested by `tests/test_all.py`.
- Impact note: Low blast radius; critical Windows API bridge.

Classification:
`correct`

Allowed next action:
Preserve in P1-B.

Forbidden next action:
Do not alter target name `LegacyGeneric:target=gemini:antigravity` or user name `antigravity`.

---

### `core/api_connector.py` (Antigravity Language Server Connector)

Current state:
Component does not exist. Currently, the tool only sends `Ctrl+R` to Antigravity window via `session_reloader.py`, without discovering the `language_server.exe` process, without reading port or `--csrf_token`, and without invoking `GetUserStatus()` or `AuthLogout()`.

Required state:
```text
Python module core/api_connector.py providing:
1. Automated discovery of language_server.exe PID, listening port (via Get-NetTCPConnection or netstat), and CSRF token (from process command-line arguments).
2. GetUserStatus() HTTP client to fetch true active email and display name from server RAM.
3. AuthLogout() HTTP client to flush active account credentials and cancel session in RAM.
4. Re-Auth trigger signaling language server to reload new token from Windows Credential Manager.
```

Evidence:
- `E0-P0A-SRC3`: `session_reloader.py` lacks internal API capabilities.
- `E0-P0A-PROC1`: Verified `language_server.exe` running with `--csrf_token` in CLI args.
- `E0-P0A-PROC2`: Verified listening loopback ports on `127.0.0.1:59226` and `59227`.
- `E0-P0A-LOG1`: `language_server.log` confirms `AuthProvider` and `Auth succeeded` RAM state lifecycle.

Relationship and impact:
- Related file count: 3
- Relationship summary: Consumed by `core/state_resolver.py`, tested by `tests/test_all.py`.
- Impact note: Medium blast radius; core engine enabling the 4-step switching cycle.

Classification:
`missing`

Allowed next action:
Implement in P2-A, P2-B, and P2-C.

Forbidden next action:
Do not hardcode port or CSRF token (must be dynamically discovered per process lifecycle).

---

### `core/state_resolver.py`

Current state:
Provides `AccountManager` which scans Chrome and AccountStore, returning list of `AccountInfo`. Its `switch_account` method writes token to Windows Credential Manager and calls `reload_antigravity_window()`, which only sends `Ctrl+R` keystroke.

Required state:
```text
AccountManager orchestrating the Owner's mandated 4-step closed-loop switching architecture:
Step 1: Synchronize true RAM state via GetUserStatus().
Step 2: Send AuthLogout() to wipe old token out of language_server RAM.
Step 3: Read target account JSON from accounts_store/ and write to Windows Credential Manager.
Step 4: Trigger Re-Auth to load new token into RAM and confirm 'Auth succeeded'.
```

Evidence:
- `E0-P0A-SRC4`: Source inspection of `core/state_resolver.py`.

Relationship and impact:
- Related file count: 4
- Relationship summary: Bridge between UI, store, scanner, credential manager, and API connector.
- Impact note: Medium blast radius.

Classification:
`partial`

Allowed next action:
Integrate `api_connector` into `AccountManager.switch_account` in P3-C.

Forbidden next action:
Do not remove Chrome discovery or AccountStore fallbacks.

---

### `scanner/chrome_scanner.py`

Current state:
Scans Chrome profile directories (`Default`, `Profile *`) and `Local State`. Uses regex and JSON parsing to extract Google emails. Enforces strict blacklist `BLACKLIST_KEYWORDS = ["anvien"]`.

Required state:
```text
Fast, accurate discovery of signed-in Chrome accounts with strict keyword exclusion.
```

Evidence:
- `E0-P0A-SRC5`: Source inspection of `scanner/chrome_scanner.py`.
- `E0-P0A-SEC1`: Unit test verifying blacklist exclusion.

Relationship and impact:
- Related file count: 2
- Relationship summary: Consumed by `core/state_resolver.py`.
- Impact note: Low blast radius.

Classification:
`correct`

Allowed next action:
Preserve in P3-A.

Forbidden next action:
Do not disable or weaken the blacklist filter.

---

### `ui/app_window.py`

Current state:
Tkinter desktop app (`SwitcherApp`) rendering dark theme cards for accounts, with `[⚡ Chuyển sang]`, `[💾 Lưu phiên]`, `[🔄 Làm mới Chrome]`. Displays status dots (🟢 Active, 🟢 Ready, 🟡 Unsaved).

Required state:
```text
Tkinter GUI displaying live RAM email verified against GetUserStatus(), detailed progress status during 4-step switching, error alerts, and manual Chrome refresh.
```

Evidence:
- `E0-P0A-SRC6`: Source inspection of `ui/app_window.py`.

Relationship and impact:
- Related file count: 2
- Relationship summary: Invoked by `main.py`.
- Impact note: Medium blast radius.

Classification:
`partial`

Allowed next action:
Update in P4-A, P4-B, P4-C to surface RAM status and switching progress.

Forbidden next action:
Do not break existing dark palette or remove existing action controls.

---

### `tests/test_all.py`

Current state:
Runs 6 unit tests covering JWT decoding, blacklist enforcement, AccountStore JSON I/O, live credential read, Chrome discovery, and state resolver mocking.

Required state:
```text
Full test suite including tests for API connector discovery, GetUserStatus(), AuthLogout(), Re-Auth trigger, and end-to-end 4-step switching sequence.
```

Evidence:
- `E0-P0A-SRC7`: Source inspection of `tests/test_all.py`.
- `E0-P0A-TEST1`: Baseline execution result (6 passed in 0.174s).

Relationship and impact:
- Related file count: 6
- Relationship summary: Test harness for entire codebase.
- Impact note: Low blast radius.

Classification:
`partial`

Allowed next action:
Add comprehensive test coverage in P5-A and P5-B.

Forbidden next action:
Do not remove existing baseline test assertions.

## Next Phase Status Decisions

| Plan Item | Actual Status Finding | Required Status / Next-Action Update |
|-----------|-----------------------|--------------------------------------|
| P1-A | `core/account_store.py` is `correct`. | Preserve and verify isolated JSON store. |
| P1-B | `core/credential_manager.py` is `correct`. | Preserve and verify Windows Credential Manager integration. |
| P2-A | Process discovery is `missing`. | Implement `language_server.exe` PID, port, and CSRF token extraction. |
| P2-B | `GetUserStatus()` and `AuthLogout()` are `missing`. | Implement HTTP client for internal auth API. |
| P2-C | Re-Auth trigger is `partial`. | Implement reload signal coordinated with API authentication. |
| P3-A | `scanner/chrome_scanner.py` is `correct`. | Preserve Chrome discovery with blacklist enforcement. |
| P3-B | Multi-account state resolution is `partial`. | Bind state resolution to live RAM status from `GetUserStatus()`. |
| P3-C | 4-step hot-swap coordination is `partial`. | Implement closed-loop 4-step hot-swap execution in `AccountManager`. |
| P4-A | UI account listing is `partial`. | Update UI to display verified RAM active email. |
| P4-B | UI hot-swap trigger is `partial`. | Wire `[⚡ Chuyển sang]` to the 4-step hot-swap flow with progress indicator. |
| P4-C | UI persistence & sync controls are `correct`. | Preserve `[💾 Lưu phiên]` and `[🔄 Làm mới Chrome]`. |
| P5-A | Unit test suite is `partial`. | Add comprehensive test coverage for API connector and new flows. |
| P5-B | End-to-end QA is `partial`. | Execute full test suite and verify 100% pass rate. |

## Implementation Gate

- [x] Target scope is listed in Current Status Matrix.
- [x] Each target unit has a status.
- [x] Each status has evidence IDs.
- [x] Each target file has relationship count evidence.
- [x] Phase Touch Map lists plan-relevant relationship files that can affect the current phase/slice.
- [x] Phase Touch Map defines touch mode for every plan-relevant relationship unit that may be affected.
- [x] Correct parts are marked preserve-only.
- [x] Partial, missing, wrong, unbound, and fake-or-stub parts have exact next actions.
- [x] Blockers are recorded, if any.
- [x] Next phase status assumptions, next action, and work steps have been updated from this status file when needed.
- [x] Status Refresh Log has an R0 baseline row.

## Final P0 Decision

- [x] P0 complete. Next phase status, next action, or work steps must be updated before implementation.

Decision note:

Baseline analysis confirms that `core/account_store.py`, `core/credential_manager.py`, and `scanner/chrome_scanner.py` are structurally sound and correct. The critical missing capability is `core/api_connector.py` for Antigravity `language_server.exe` discovery (port & CSRF token), `GetUserStatus()`, `AuthLogout()`, and Re-Auth triggering. This enables the Owner's mandated 4-step closed-loop internal API switching architecture to be fully integrated into `core/state_resolver.py` (P3), `ui/app_window.py` (P4), and verified with comprehensive unit/integration tests in `tests/test_all.py` (P5). All downstream phases are strictly sequenced accordingly.
