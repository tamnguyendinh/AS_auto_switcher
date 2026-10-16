# Antigravity Multi-Account Switcher Actual Status

Title: Antigravity Multi-Account Switcher Actual Status
Date: 2026-10-14 (Updated: 2026-10-15)
Status: in_progress
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

- Python Validation Prototype (Phases P0–P5): complete multi-account switcher with persistent JSON account storage (`accounts_store/<email>.json`), automated Chrome discovery, and 4-step closed-loop internal API switching architecture.
- Phase P6 (Go Engine Port, Fyne Native GUI & Standalone Binary): complete port to compiled Go (Golang), pure Go Fyne native GUI (< 20MB RAM, < 50ms startup), Win32 `Advapi32.dll` integration via `golang.org/x/sys/windows`, compiled into single standalone executable (`antigravity-switch.exe`, 10MB–15MB).
- Phase P7 (Go-Native Autonomous Quota Failover Daemon): non-blocking background goroutine tailer on `language_server.log` detecting 429/RESOURCE_EXHAUSTED in $\le 20\text{ms}$, managing thread-safe `AccountPool` with exponential backoff cooldowns, 15-second anti-flapping hysteresis gate, sub-second ($< 1.0\text{s}$) hot-swap, native Windows System Tray HUD with dynamic status dots, discrete Windows Toast alerts, and 100% IDE context preservation.
- Target Go units: `internal/model/`, `internal/wincred/`, `internal/store/`, `internal/scanner/`, `internal/connector/`, `internal/reloader/`, `internal/manager/`, `internal/ui/`, `cmd/switcher/`, `antigravity-switch.exe`, `internal/sensor/`, `internal/controller/`, `internal/tray/`, `tests/e2e/`.

Out of scope:

- Modifying Antigravity IDE binaries, Electron runtime, or language_server.exe executable on disk.
- Web/Production OAuth server or cloud auth brokering (Prototype Auth only).
- Bypassing or modifying Chrome browser user profiles on disk other than read-only discovery.
- Electron or WebView2 embedded runtimes (pure Go native Fyne GUI only).
- Use of anvien.

## Relationship / Impact Evidence

| Unit / File / Surface | File Detail Evidence | Related File Count | Relationship Summary | Impact Note |
|-----------------------|----------------------|--------------------|----------------------|-------------|
| `core/account_store.py` | `E0-P0A-SRC1` | 3 | Imported by `state_resolver.py` and `tests/test_all.py` | low |
| `core/credential_manager.py` | `E0-P0A-SRC2`, `E0-P0A-CRED1` | 3 | Imported by `state_resolver.py` and `tests/test_all.py` | low (preserves Windows DPAPI credential format) |
| `core/session_reloader.py` | `E0-P0A-SRC3` | 2 | Imported by `state_resolver.py` | low |
| `core/api_connector.py` | `E0-P0A-PROC1`, `E0-P0A-PROC2`, `E0-P0A-LOG1` | 3 | Discovers `language_server.exe` PID, port, CSRF token; internal API clients | medium (central to 4-step switching cycle) |
| `scanner/chrome_scanner.py` | `E0-P0A-SRC5`, `E0-P0A-SEC1` | 2 | Imported by `state_resolver.py` and `tests/test_all.py` | low |
| `core/state_resolver.py` | `E0-P0A-SRC4` | 4 | Orchestrates store, credential, scanner, reloader, imported by `ui/app_window.py` | medium |
| `ui/app_window.py` | `E0-P0A-SRC6` | 2 | Imported by `main.py` | medium |
| `tests/test_all.py` | `E0-P0A-SRC7`, `E0-P0A-TEST1` | 6 | Tests all core Python modules and scanner | low |
| `internal/model/` | `E6-P6A-MOD1` | 5 | Core data structures: `TokenData`, `AccountRecord`, `AccountPool` | low |
| `internal/wincred/` | `E6-P6A-WIN1` | 3 | Win32 `Advapi32.dll` wrapper via `golang.org/x/sys/windows` | medium (native OS credential bridge) |
| `internal/store/` | `E6-P6A-STORE1` | 3 | Atomic JSON file persistence in `accounts_store/<email>.json` | low |
| `internal/scanner/` | `E6-P6B-SCAN1` | 2 | Chrome profile scanner with strict `anvien` blacklist filtering | low |
| `internal/connector/` | `E6-P6B-CONN1` | 3 | `net/http` loopback client for `GetUserStatus`, `AuthLogout`, `TriggerReauth` | medium |
| `internal/reloader/` | `E6-P6C-REL1` | 2 | PID-based HWND discovery and Win32 `keybd_event` reload dispatch | low |
| `internal/manager/` | `E6-P6C-MGR1` | 5 | Orchestrates store, creds, scanner, connector, reloader (4-step hot-swap) | medium |
| `internal/ui/` | `E6-P6C-UI1` | 2 | Fyne 100% pure Go native GUI dashboard (< 20MB RAM, < 50ms startup) | medium |
| `cmd/switcher/` | `E6-P6C-CLI1` | 4 | Dual CLI/GUI entrypoint with command flags | low |
| `antigravity-switch.exe` | `E6-P6D-BIN1` | 1 | Final standalone static Windows binary ($\le 15\text{MB}$) | high |
| `internal/sensor/` | `E7-P7A-TAIL1` | 2 | Non-blocking goroutine log tailer on `language_server.log` (429 detection) | medium |
| `internal/controller/` | `E7-P7B-POOL1` | 4 | Thread-safe `AccountPool` FSM, scheduler, and 15s hysteresis gate | medium |
| `internal/tray/` | `E7-P7C-TRAY1` | 3 | Native Windows systray, Action Center Toast bridge, and global hotkeys | low |
| `tests/e2e/` | `E7-P7D-STORM1` | 5 | E2E simulation test harness and Quota Storm stress testing suite | low |

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
| `scanner/chrome_scanner.py` | Multi-profile extraction and strict `anvien` blacklist filtering hardened. | Robust extraction of signed-in Chrome emails excluding blacklisted terms. | `correct` | 2 related files | `E0-P0A-SRC5`, `E0-P0A-SEC1`, `E3-P3A-SCAN1` | Preserve in P3-B, wire into state resolver |
| `core/state_resolver.py` | Live RAM identity binding, auto-capture, and state resolution (ACTIVE, READY, UNAUTH) implemented. | Resolves accounts using RAM status from `GetUserStatus()`; coordinates full 4-step hot-swap loop. | `correct` | 4 related files | `E0-P0A-SRC4`, `E3-P3B-STATE1` | Preserve in P3-C / wire 4-step switching loop coordinator |
| `ui/app_window.py` | Live RAM identity display, 1-click hot-swap with visual feedback, session persistence, and manual sync controls implemented. | Displays live RAM account status, real-time 4-step switching progress, error feedback, and manual sync. | `correct` | 2 related files | `E0-P0A-SRC6`, `E4-P4A-UI1`, `E4-P4B-ACT1`, `E4-P4C-SYNC1` | Preserve in Phase P5 / verify in test suite |
| `tests/test_all.py` | 68 unit & regression tests passing, 100% pass rate. | Comprehensive test suite covering API connector, port discovery, smooth hot-swap, PID discovery, and edge cases. | `correct` | 6 related files | `E5-P5A-UNIT1`, `E5-P5B-REG1` | Verified in P5-B |
| `internal/model/` | Strongly typed Go models (`TokenData`, `AccountRecord`, `AccountState`, `AccountPool`, `FailoverConfig`) implemented and verified. | Strongly typed Go models (`TokenData`, `AccountRecord`, `AccountState`). | `correct` | 5 related files | `E6-P6A-MOD1` | Implemented in Slice P6-A (Commit: `a7f999b`, Supervisor Commit: `39abfde`) |
| `internal/wincred/` | Pure Go Win32 `Advapi32.dll` wrapper via `golang.org/x/sys/windows` implemented and verified (CredReadW, CredWriteW, CredDeleteW, CredFree). | Pure Go Win32 `Advapi32.dll` wrapper using `golang.org/x/sys/windows`. | `correct` | 3 related files | `E6-P6A-WIN1` | Implemented in Slice P6-A (Commit: `a7f999b`, Supervisor Commit: `39abfde`) |
| `internal/store/` | Atomic JSON file persistence in `accounts_store/<email>.json` with path sanitizer implemented and verified. | Atomic JSON file persistence in `accounts_store/<email>.json` with path sanitizer. | `correct` | 3 related files | `E6-P6A-STORE1` | Implemented in Slice P6-A (Commit: `a7f999b`, Supervisor Commit: `39abfde`) |
| `internal/scanner/` | Chrome profile scanner in Go with strict `anvien` blacklist filtering implemented and verified (Local State, Preferences, regex fallback, metadata enrichment). | Chrome profile scanner in Go with strict `anvien` blacklist filtering. | `correct` | 2 related files | `E6-P6B-SCAN1` | Implemented in Slice P6-B (Commit: `68440b1`, Supervisor Commit: `ca6c76b`) |
| `internal/connector/` | Language server process discovery (PID, ephemeral port via Win32 GetExtendedTcpTable / netstat, --csrf_token) and loopback Connect-RPC client implemented and verified. | Language server PID/port discovery and loopback HTTP Connect-RPC client. | `correct` | 3 related files | `E6-P6B-CONN1` | Implemented in Slice P6-B (Commit: `68440b1`, Supervisor Commit: `ca6c76b`) |
| `internal/reloader/` | PID-based HWND discovery, window enumeration, Ctrl+R keybd_event reload dispatch, and atomic ToS pre-patching implemented and verified without CGo. | PID-based HWND discovery and Win32 `keybd_event` Ctrl+R reload bridge. | `correct` | 2 related files | `E6-P6C-REL1`, `E6-P6C-MGR1` | Implemented in Slice P6-C (Commit: `363307e`, Supervisor Commit: `5dd7e8a`) |
| `internal/manager/` | Multi-account orchestrator, active session auto-capture, live RAM identity resolution, deterministic account sorting, and closed-loop 4-step hot-swap implemented and verified. | Multi-account orchestrator and 4-step hot-swap engine in Go. | `correct` | 5 related files | `E6-P6C-MGR1` | Implemented in Slice P6-C (Commit: `363307e`, Supervisor Commit: `5dd7e8a`) |
| `internal/ui/` | Catppuccin Mocha-inspired Fyne native GUI dashboard, async non-blocking operations, custom dynamic tray icon, and minimize-to-tray implemented and verified. | Fyne 100% pure Go native GUI dashboard (< 20MB RAM, < 50ms startup). | `correct` | 2 related files | `E6-P6C-UI1` | Implemented in Slice P6-C (Commit: `363307e`, Supervisor Commit: `5dd7e8a`) |
| `cmd/switcher/` | Dual CLI/GUI entrypoint with full CLI actions (--cli, --status, --list, --switch, --capture) and Fyne GUI default boot implemented and verified. | Dual CLI/GUI entrypoint with command flags (`--cli`, `--gui`, `--switch`). | `correct` | 4 related files | `E6-P6C-CLI1`, `E6-P6C-UI1` | Implemented in Slice P6-C (Commit: `363307e`, Supervisor Commit: `5dd7e8a`) |
| `antigravity-switch.exe` | Not compiled yet. | Standalone compiled Windows binary ($\le 15\text{MB}$, zero external dependencies). | `missing` | 1 related file | `E6-P6D-BIN1` | Compile & benchmark in Slice P6-D |
| `internal/sensor/` | Not created yet in repository. | Non-blocking goroutine log tailer on `language_server.log` (429 detection). | `missing` | 2 related files | `E7-P7A-TAIL1` | Implement in Slice P7-A |
| `internal/controller/` | Not created yet in repository. | Thread-safe `AccountPool` FSM, scheduler, and 15s hysteresis gate. | `missing` | 4 related files | `E7-P7B-POOL1` | Implement in Slice P7-B |
| `internal/tray/` | Not created yet in repository. | Native Windows systray, Action Center Toast bridge, and global hotkeys. | `missing` | 3 related files | `E7-P7C-TRAY1` | Implement in Slice P7-C |
| `tests/e2e/` | Not created yet in repository. | E2E simulation test harness and Quota Storm stress testing suite. | `missing` | 5 related files | `E7-P7D-STORM1` | Implement in Slice P7-D |

## Status Refresh Log

| Refresh | Date | Repo Basis | Changed Scope | Status Changes | Evidence | Next Phase Update |
|---------|------|------------|----------------|----------------|----------|-------------------|
| R0 | 2026-10-14 | baseline before P0 | Entire repository | Initial classification: 3 correct, 4 partial, 1 missing | `E0-P0A-SRC1..SRC8`, `E0-P0A-PROC1..PROC2`, `E0-P0A-TEST1` | Proceed to implementation slices P1 through P5 |
| R1 | 2026-10-14 | commit 2088e77 (Slice P1-A) | `core/account_store.py`, `tests/test_all.py` | P1-A complete: atomic JSON file operations, filename sanitation, schema validation verified | `E1-P1A-STORE1`, `E1-P1A-STORE2` | Proceed to Slice P1-B (Active Session Capture & Credential Synchronization) |
| R2 | 2026-10-14 | commit 50149b2 (Slice P1-B) | `core/credential_manager.py`, `tests/test_all.py` | P1-B complete: Windows Credential Manager integration hardened (validation, error resilience, safe delete) and verified | `E1-P1B-CRED1`, `E1-P1B-AUTO1` | Proceed to Phase P2 (Language Server API Connector) |
| R3 | 2026-10-14 | commit 04bdf95 (Slice P2-A) | `core/api_connector.py`, `tests/test_all.py` | P2-A complete: language server discovery (PID, port, CSRF token extraction) implemented and verified | `E2-P2A-DISC1` | Proceed to Slice P2-B (Internal Auth API Client: GetUserStatus & AuthLogout) |
| R4 | 2026-10-14 | commit b7d550b (Slice P2-B) | `core/api_connector.py`, `tests/test_all.py` | P2-B complete: GetUserStatus and AuthLogout HTTP API clients implemented and verified | `E2-P2B-API1`, `E2-P2B-API2` | Proceed to Slice P2-C (Re-Auth Trigger & RAM Session Reload) |
| R5 | 2026-10-14 | commit c8d3423 (Slice P2-C) | `core/api_connector.py`, `core/session_reloader.py`, `tests/test_all.py` | P2-C complete: Re-Auth trigger and RAM session reloader implemented and verified | `E2-P2C-REAUTH1` | Proceed to Phase P3 (Account Discovery & State Management) |
| R6 | 2026-10-14 | commit 38d5e0d (Slice P3-A) | `scanner/chrome_scanner.py`, `tests/test_all.py` | P3-A complete: Chrome profile extraction and strict `anvien` blacklist filtering hardened and verified | `E3-P3A-SCAN1` | Proceed to Slice P3-B (Multi-Account State Machine Resolution) |
| R7 | 2026-10-14 | commit 53a456b (Slice P3-B) | `core/state_resolver.py`, `tests/test_all.py` | P3-B complete: live RAM identity binding, state machine resolution (ACTIVE, READY, UNAUTH), and auto-capture implemented and verified | `E3-P3B-STATE1` | Proceed to Slice P3-C (4-Step Closed-Loop Hot-Swap Coordinator) |
| R8 | 2026-10-14 | commit cf952ea (Slice P3-C) | `core/state_resolver.py`, `tests/test_all.py` | P3-C complete: 4-step closed-loop hot-swap coordinator implemented and verified | `E3-P3C-SW1` | Proceed to Phase P4 (User Interface Upgrades) |
| R9 | 2026-10-14 | commit ddd180e (Slice P4-A) | `ui/app_window.py`, `tests/test_all.py` | P4-A complete: UI wired to live RAM state and hot-swap coordinator with visual badging and status bar feedback | `E4-P4A-UI1` | Proceed to Slice P4-B (1-Click Hot-Swap Trigger & Status Feedback) |
| R10 | 2026-10-14 | commit 3460f02 (Slice P4-B) | `ui/app_window.py`, `tests/test_all.py` | P4-B complete: 1-click hot-swap trigger connected with visual progress feedback, error handling, and post-swap state refresh | `E4-P4B-ACT1` | Proceed to Slice P4-C (Session Persistence & Manual Chrome Sync Controls) |
| R11 | 2026-10-14 | commit 14b567b (Slice P4-C) | `ui/app_window.py`, `tests/test_all.py` | P4-C complete: session persistence and manual Chrome sync controls implemented and verified | `E4-P4C-SYNC1` | Proceed to Phase P5 (Verification & Regression QA) |
| R12 | 2026-10-14 | commit 4673f40 (Slice P5-A) | `tests/test_all.py` | P5-A complete: expanded comprehensive unit and integration test suite (API connector, 4-step switching flow, store, credential manager) verified | `E5-P5A-UNIT1` | Proceed to Slice P5-B (End-to-End Switching Cycle & Robustness Verification) |
| R13 | 2026-10-14 | Slice P5-B | `core/session_reloader.py`, `core/state_resolver.py`, `tests/test_all.py` | P5-B complete: PID-based window discovery and smooth zero-chrome hot-swap implemented; 68/68 tests passing across 5 consecutive runs | `E5-P5B-REG1` | Proceed to Supervisor loop (Pn-A) |
| R14 | 2026-10-14 | commit a34a6f8 (Slice Pn-A) | `docs/plans/` | Supervisor verification loop PASS on commit a34a6f8 | `E-PNA-VERIFY1` | Proceed to Slice Pn-B (Remove dead work created during this plan) |
| R15 | 2026-10-14 | commit 9c9d862 (Slice Pn-B) | `core/`, `scanner/`, `tests/` | Dead code purge verified by Supervisor PASS on commit 9c9d862 | `E-PNB-CLEAN1` | Proceed to Slice Pn-C (Close the plan) |
| R16 | 2026-10-15 | Slice Pn-C | `docs/plans/`, `reports/planner/` | Official plan closure and ledgers synchronization; all 5 phases, Pn-A, Pn-B passed, test suite 68/68 passed | `E-PNC-CLOSE1` | Plan completed and ready for Owner live testing |
| R17 | 2026-10-15 | Architect Report approval | Docs & Plan Set | Phase 6 (Go Engine, Fyne GUI, Standalone Binary) & Phase 7 (Autonomous Failover Daemon) incorporated; 14 new Go units classified as `missing` | `reports/architect/2026-10-14-autonomous-quota-failover-architecture.md` | Proceed to implementation of Slice P6-A |
| R18 | 2026-10-15 | commit a7f999b, supervisor 39abfde (Slice P6-A) | `internal/model/`, `internal/wincred/`, `internal/store/`, `go.mod` | P6-A complete: Go primitives (TokenData, AccountRecord, AccountState), pure Go Win32 Advapi32 credential wrapper, atomic JSON store with path sanitizer implemented and supervisor verified; 10/10 tests pass | `E6-P6A-WIN1`, `E6-P6A-STORE1`, `E6-P6A-MOD1` | Proceed to Slice P6-B (Chrome Discovery Scanner & Language Server Connector) |
| R19 | 2026-10-15 | commit 68440b1, supervisor ca6c76b (Slice P6-B) | `internal/scanner/`, `internal/connector/` | P6-B complete: Chrome profile discovery scanner with strict `anvien` blacklist filtering, language server PID/port discovery (Win32 GetExtendedTcpTable / netstat), and loopback Connect-RPC client implemented; 13/13 tests pass (100%), verified against live language_server.exe | `E6-P6B-SCAN1`, `E6-P6B-CONN1` | Proceed to Slice P6-C (Core Orchestrator & Fyne Native GUI) |
| R20 | 2026-10-15 | commit 363307e, supervisor 5dd7e8a (Slice P6-C) | `cmd/switcher/`, `internal/manager/`, `internal/reloader/`, `internal/ui/` | P6-C complete: Core multi-account orchestrator, session reloader (Win32 toolhelp, Ctrl+R, ToS patch without CGo), Catppuccin Mocha-inspired Fyne native GUI dashboard, system tray, and dual CLI/GUI entrypoint implemented; 35/35 Go tests and 133/133 Python regression tests pass (100%) | `E6-P6C-MGR1`, `E6-P6C-UI1`, `E6-P6C-REL1`, `E6-P6C-CLI1` | Proceed to Slice P6-D (Standalone Windows Binary Compilation & Benchmark) |

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
| `internal/model/` | `internal/store/`, `internal/manager/` | dependency | P6-A | edit (new) | `E6-P6A-MOD1` | Pure Go models matching Architect spec |
| `internal/wincred/` | `internal/manager/` | dependency | P6-A | edit (new) | `E6-P6A-WIN1` | Pure Go Win32 `Advapi32.dll` via `golang.org/x/sys/windows` |
| `internal/store/` | `internal/manager/` | dependency | P6-A | edit (new) | `E6-P6A-STORE1` | Atomic JSON rename write pattern |
| `internal/scanner/` | `internal/manager/` | dependency | P6-B | edit (new) | `E6-P6B-SCAN1` | Read-only Chrome preference parsing; strict `anvien` filter |
| `internal/connector/` | `internal/manager/` | dependency | P6-B | edit (new) | `E6-P6B-CONN1` | Loopback `127.0.0.1:<port>` only |
| `internal/reloader/` | `internal/manager/` | dependency | P6-C | edit (new) | `E6-P6C-REL1` | Target `Antigravity.exe` PID window; Win32 keystroke reload |
| `internal/manager/` | `internal/ui/`, `cmd/switcher/` | orchestrator | P6-C | edit (new) | `E6-P6C-MGR1` | 4-step hot-swap coordination in Go |
| `internal/ui/` | `cmd/switcher/` | interface | P6-C | edit (new) | `E6-P6C-UI1` | Fyne pure Go native GUI; dark theme; < 20MB RAM |
| `cmd/switcher/` | `antigravity-switch.exe` | entrypoint | P6-C | edit (new) | `E6-P6C-CLI1` | CLI/GUI flag parsing; daemon loop |
| `antigravity-switch.exe` | Windows OS | artifact | P6-D | build (new) | `E6-P6D-BIN1` | Single static binary $\le 15\text{MB}$; zero DLL dependencies |
| `internal/sensor/` | `internal/controller/` | event source | P7-A | edit (new) | `E7-P7A-TAIL1` | Non-blocking goroutine tailing `language_server.log` |
| `internal/controller/` | `internal/manager/` | decision engine | P7-B | edit (new) | `E7-P7B-POOL1` | Thread-safe `AccountPool` FSM; 15s hysteresis gate |
| `internal/tray/` | `cmd/switcher/` | background UI | P7-C | edit (new) | `E7-P7C-TRAY1` | Fyne `desktop.App` / native systray; Windows toast alerts |
| `tests/e2e/` | Go codebase | test target | P7-D | edit (new) | `E7-P7D-STORM1` | Quota Storm stress testing; benchmark $< 1.0\text{s}$ failover |

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
Runs 68 unit & regression tests covering API connector discovery, GetUserStatus(), AuthLogout(), Re-Auth trigger, smooth hot-swap, PID discovery, and state resolver mocking, passing 100%.

Required state:
```text
Full Python baseline regression suite passing 100% across all 68 tests.
```

Evidence:
- `E5-P5A-UNIT1`: Source inspection and test suite expansion.
- `E5-P5B-REG1`: Regression pass (68 passed in 28.174s, 5/5 batch runs).

Relationship and impact:
- Related file count: 6
- Relationship summary: Test harness for entire Python codebase.
- Impact note: Low blast radius.

Classification:
`correct`

Allowed next action:
Preserve as historical regression baseline.

Forbidden next action:
Do not alter existing Python tests during Go conversion.

---

### `internal/model/`

Current state:
Not yet implemented in repository.

Required state:
```text
Strongly typed Go model definitions: TokenData, AccountRecord, AccountState (ACTIVE, READY, QUOTA_EXHAUSTED, UNAUTH, ERROR), and FailoverConfig matching Architect specification.
```

Evidence:
- `E6-P6A-MOD1`: Architect specification Section 4.3.

Relationship and impact:
- Related file count: 5
- Relationship summary: Core domain model imported across store, manager, controller, and UI.
- Impact note: Low blast radius; foundational data structs.

Classification:
`missing`

Allowed next action:
Implement in Slice P6-A (`internal/model/types.go`).

Forbidden next action:
Do not introduce external CGo dependencies or conflicting state enums.

---

### `internal/wincred/`

Current state:
Not yet implemented in repository.

Required state:
```text
Pure Go native wrapper for Windows Advapi32.dll (CredReadW, CredWriteW, CredDeleteW, CredFree) using golang.org/x/sys/windows targeting LegacyGeneric:target=gemini:antigravity with zero CGo.
```

Evidence:
- `E6-P6A-WIN1`: Architect specification Section 2 & ADR-P6-02.

Relationship and impact:
- Related file count: 3
- Relationship summary: Windows Credential Manager bridge imported by `manager` and `cmd/switcher`.
- Impact note: Medium blast radius; OS credential security boundary.

Classification:
`missing`

Allowed next action:
Implement in Slice P6-A (`internal/wincred/cred.go`, `internal/wincred/types.go`).

Forbidden next action:
Do not use CGo or external C compilers.

---

### `internal/store/`

Current state:
Not yet implemented in repository.

Required state:
```text
Atomic JSON file storage engine in accounts_store/<email>.json using os.CreateTemp + os.Rename transactional pattern, with email path sanitizer preventing directory traversal.
```

Evidence:
- `E6-P6A-STORE1`: Architect specification Section 2 & Section 6.

Relationship and impact:
- Related file count: 3
- Relationship summary: Multi-account JSON external memory imported by `manager` and `controller`.
- Impact note: Low blast radius; safe isolated file I/O.

Classification:
`missing`

Allowed next action:
Implement in Slice P6-A (`internal/store/account_store.go`, `internal/store/sanitizer.go`).

Forbidden next action:
Do not write non-atomic partial JSON files or permit path traversal characters.

---

### `internal/scanner/`

Current state:
Implemented in `internal/scanner/chrome_scanner.go` and verified in `chrome_scanner_test.go`. Discovers accounts from `%LOCALAPPDATA%\Google\Chrome\User Data\Local State` (`profile.info_cache`, `profile.profiles_attributes`) and profile `Preferences` files (`account_info`, `account_tracker_service_last_update`, and regex fallbacks). Enforces strict case-insensitive blacklist filtering (`anvien`) and eliminates false positives.

Required state:
```text
Fast Chrome profile preference reader in Go, extracting signed-in Google accounts from Local State and profile Preferences with strict case-insensitive blacklist filtering ("anvien").
```

Evidence:
- `E6-P6B-SCAN1`: 5/5 unit tests pass in `internal/scanner/chrome_scanner_test.go`, verified against live Chrome User Data directory.

Relationship and impact:
- Related file count: 2
- Relationship summary: Account discovery scanner imported by `manager`.
- Impact note: Low blast radius; read-only file parsing.

Classification:
`correct`

Allowed next action:
Preserve in Slice P6-C (`internal/scanner/chrome_scanner.go`).

Forbidden next action:
Do not modify Chrome browser profiles or SQLite databases on disk; strictly enforce anvien exclusion.

---

### `internal/connector/`

Current state:
Implemented in `internal/connector/discovery.go`, `internal/connector/api_client.go`, and `internal/connector/endpoints.go` and verified in `connector_test.go`. Discovers `language_server.exe` PID, extracts `--csrf_token` from command line, resolves ephemeral loopback listening port via Win32 `GetExtendedTcpTable` with fallback to `netstat -ano -p tcp`, probes HTTP port, and provides loopback Connect-RPC client for `GetUserStatus`, `AuthLogout`, `TriggerReauth`, `LoadTrajectory`, `SmartFocusConversation`, `DiscoverActiveAgentModel`, and `SendUserCascadeMessage`.

Required state:
```text
Language server process discovery (PID, ephemeral port via Windows TCP netstat, --csrf_token regex) and high-performance net/http loopback client for GetUserStatus(), AuthLogout(), and TriggerReauth().
```

Evidence:
- `E6-P6B-CONN1`: 8/8 unit tests pass in `internal/connector/connector_test.go`, verified live against running `language_server.exe` (PID 6396, Port 58644).

Relationship and impact:
- Related file count: 3
- Relationship summary: Antigravity language_server.exe internal API client imported by `manager` and `controller`.
- Impact note: Medium blast radius; essential for closed-loop hot-swap.

Classification:
`correct`

Allowed next action:
Preserve in Slice P6-C and wire into `internal/manager/account_manager.go`.

Forbidden next action:
Do not broadcast network requests; strictly bound to loopback 127.0.0.1.

---

### `internal/reloader/`

Current state:
Not yet implemented in repository.

Required state:
```text
Window targeting by PID and Win32 keybd_event reload keystroke dispatch (Ctrl+R) coordinated with Language Server API re-authentication.
```

Evidence:
- `E6-P6C-REL1`: Architect specification Section 2 & Section 6.

Relationship and impact:
- Related file count: 2
- Relationship summary: Session reloader imported by `manager`.
- Impact note: Low blast radius; UI event dispatch.

Classification:
`missing`

Allowed next action:
Implement in Slice P6-C (`internal/reloader/session_reloader.go`).

Forbidden next action:
Do not kill or restart the Antigravity IDE process forcefully.

---

### `internal/manager/`

Current state:
Not yet implemented in repository.

Required state:
```text
Central Go orchestrator coordinating account store, Win32 credentials, Chrome scanner, API connector, and reloader to execute the 4-step closed-loop hot-swap.
```

Evidence:
- `E6-P6C-MGR1`: Architect specification Section 2 & Section 6.

Relationship and impact:
- Related file count: 5
- Relationship summary: Orchestrator imported by `internal/ui` and `cmd/switcher`.
- Impact note: Medium blast radius; core switching logic.

Classification:
`missing`

Allowed next action:
Implement in Slice P6-C (`internal/manager/account_manager.go`).

Forbidden next action:
Do not trigger unwanted Chrome OAuth re-login popups; execute smooth hot-swap.

---

### `internal/ui/`

Current state:
Not yet implemented in repository.

Required state:
```text
100% pure Go native GUI using Fyne: compact 440x560 px dark theme window, active account banner with live status dot, scrollable account cards, 1-click switch button [⚡ Chuyển], [💾 Lưu phiên], [🔄 Làm mới Chrome], and Auto-Failover toggle (< 20MB RAM, < 50ms startup).
```

Evidence:
- `E6-P6C-UI1`: Architect specification Section 1.2 & ADR-P6-05.

Relationship and impact:
- Related file count: 2
- Relationship summary: Desktop GUI dashboard imported by `cmd/switcher/main.go`.
- Impact note: Medium blast radius; primary user interface.

Classification:
`missing`

Allowed next action:
Implement in Slice P6-C (`internal/ui/app.go`, `internal/ui/window.go`, `internal/ui/widgets.go`, `internal/ui/tray.go`).

Forbidden next action:
Do not use Electron, Node.js, npm, or WebView2 runtimes.

---

### `cmd/switcher/`

Current state:
Not yet implemented in repository.

Required state:
```text
Dual CLI/GUI application entrypoint supporting flags (--cli, --gui, --switch <email>, --status, --list, --capture), defaulting to Fyne GUI on double-click.
```

Evidence:
- `E6-P6C-CLI1`: Architect specification Section 2 & Section 6.

Relationship and impact:
- Related file count: 4
- Relationship summary: Main application entrypoint compiling into `antigravity-switch.exe`.
- Impact note: Low blast radius.

Classification:
`missing`

Allowed next action:
Implement in Slice P6-C (`cmd/switcher/main.go`).

Forbidden next action:
Do not hardcode paths or configuration values.

---

### `antigravity-switch.exe`

Current state:
Not yet compiled.

Required state:
```text
Single standalone Windows executable (10MB–15MB) compiled with -ldflags="-H windowsgui -s -w", embedded Windows manifest and icon, startup <= 10ms, RAM <= 12MB (daemon/CLI) / <= 20MB (GUI), zero Python or MSVC runtime dependencies.
```

Evidence:
- `E6-P6D-BIN1`: Architect specification Section 1.3 & Section 6.

Relationship and impact:
- Related file count: 1
- Relationship summary: Production binary deliverable.
- Impact note: High blast radius; end product artifact.

Classification:
`missing`

Allowed next action:
Build and benchmark in Slice P6-D (`scripts/build_windows.ps1`).

Forbidden next action:
Do not bundle Python virtual environments or external runtime installers.

---

### `internal/sensor/`

Current state:
Not yet implemented in repository.

Required state:
```text
Non-blocking background goroutine tailer on language_server.log, detecting HTTP 429, gRPC RESOURCE_EXHAUSTED, and quota errors in <= 20ms with < 0.05% CPU, emitting QuotaExhaustedEvent over an unbuffered Go channel.
```

Evidence:
- `E7-P7A-TAIL1`: Architect specification Section 2 & ADR-P7-01.

Relationship and impact:
- Related file count: 2
- Relationship summary: Real-time sensor piping events to `internal/controller`.
- Impact note: Medium blast radius; autonomous detection core.

Classification:
`missing`

Allowed next action:
Implement in Slice P7-A (`internal/sensor/log_tailer.go`, `internal/sensor/signatures.go`, `internal/sensor/event_bus.go`).

Forbidden next action:
Do not use tight CPU polling loops with sleep; use incremental byte seek and shrink detection.

---

### `internal/controller/`

Current state:
Not yet implemented in repository.

Required state:
```text
Thread-safe AccountPool (sync.RWMutex), Round-Robin / LRU scheduler, exponential backoff cooldown tracker (15m, 30m, 60m capped at 2h), 15-second minimum dwell time hysteresis gate, and automated hot-swap invocation.
```

Evidence:
- `E7-P7B-POOL1`: Architect specification Section 4 & Section 5.

Relationship and impact:
- Related file count: 4
- Relationship summary: Autonomous decision engine coordinating failovers.
- Impact note: Medium blast radius; state machine core.

Classification:
`missing`

Allowed next action:
Implement in Slice P7-B (`internal/controller/pool_manager.go`, `internal/controller/scheduler.go`, `internal/controller/cooldown.go`, `internal/controller/hysteresis.go`).

Forbidden next action:
Do not allow flapping or rapid cascading exhaustion of accounts during backend outages.

---

### `internal/tray/`

Current state:
Not yet implemented in repository.

Required state:
```text
Native Windows System Tray resident daemon with dynamic status color dots (🟢 Healthy, 🟡 Active/Single, 🔴 All Exhausted with countdown tooltip), Windows 10/11 Action Center Toast bridge, and global hotkey listener (Win+Alt+S).
```

Evidence:
- `E7-P7C-TRAY1`: Architect specification Section 2 & ADR-P7-02.

Relationship and impact:
- Related file count: 3
- Relationship summary: Ambient desktop HUD alerting the developer without interrupting flow.
- Impact note: Low blast radius; notification interface.

Classification:
`missing`

Allowed next action:
Implement in Slice P7-C (`internal/tray/tray_daemon.go`, `internal/tray/notifications.go`, `internal/tray/hotkeys.go`).

Forbidden next action:
Do not steal keyboard focus or show intrusive modal popups during auto-switches.

---

### `tests/e2e/`

Current state:
Not yet implemented in repository.

Required state:
```text
Automated simulation test harness with mock language server, synthetic log injection, Quota Storm stress testing across 3+ accounts, verifying sub-second (< 1.0s) failover cycle and 100% IDE context preservation.
```

Evidence:
- `E7-P7D-STORM1`: Architect specification Section 3 & Section 6.

Relationship and impact:
- Related file count: 5
- Relationship summary: End-to-end integration and resilience test harness.
- Impact note: Low blast radius; validation suite.

Classification:
`missing`

Allowed next action:
Implement in Slice P7-D (`tests/e2e/quota_storm_test.go`, `tests/e2e/e2e_test.go`).

Forbidden next action:
Do not terminate actual developer processes during tests; use isolated mock servers.

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
| P5-B | End-to-end QA is `correct`. | Execute full test suite and verify 100% pass rate across 68 tests. |
| P6-A | Go core primitives, Win32 Advapi32 wrapper, atomic store are `missing`. | Implement `internal/model/`, `internal/wincred/`, `internal/store/` in Go. |
| P6-B | Go Chrome scanner and language server loopback connector are `correct`. | Preserve `internal/scanner/` and `internal/connector/`; proceed to Slice P6-C. |
| P6-C | Go session reloader, account manager orchestrator, Fyne GUI, and CLI entrypoint are `correct`. | Preserve `internal/reloader/`, `internal/manager/`, `internal/ui/`, `cmd/switcher/`; proceed to Slice P6-D. |
| P6-D | Standalone Windows binary `antigravity-switch.exe` is `missing`. | Author build script `scripts/build_windows.ps1`, compile static binary, benchmark footprint. |
| P7-A | Goroutine log tailer and quota signature detection pipeline are `missing`. | Implement `internal/sensor/` (log tailer, signatures, event bus) in Go. |
| P7-B | Go failover controller, AccountPool FSM, and hysteresis gate are `missing`. | Implement `internal/controller/` (pool, scheduler, cooldown, hysteresis) in Go. |
| P7-C | Native Windows system tray HUD, toast notification bridge, and hotkeys are `missing`. | Implement `internal/tray/` (tray daemon, toast notifications, global hotkeys) in Go. |
| P7-D | End-to-end simulation test harness and Quota Storm stress tests are `missing`. | Implement `tests/e2e/` simulation suite and benchmark failover latency $< 1.0\text{s}$. |

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

Baseline analysis confirms that `core/account_store.py`, `core/credential_manager.py`, and `scanner/chrome_scanner.py` are structurally sound and correct. The critical missing capability is `core/api_connector.py` for Antigravity `language_server.exe` discovery (port & CSRF token), `GetUserStatus()`, `AuthLogout()`, and Re-Auth triggering. This enables the Owner's mandated 4-step closed-loop internal API switching architecture to be fully integrated into `core/state_resolver.py` (P3), `ui/app_window.py` (P4), and verified with comprehensive unit/integration tests in `tests/test_all.py` (P5). All downstream phases are strictly sequenced accordingly. Following Phase P5 completion, Phases P6 and P7 have been incorporated into the plan based on the Architect Report, decomposing the Go engine conversion, Fyne native GUI, standalone binary packaging, and autonomous quota failover daemon into concrete atomic execution slices.
