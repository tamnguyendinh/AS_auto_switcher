# Antigravity Multi-Account Switcher Evidence Ledger

## Metadata

- Date: `2026-10-14`
- Plan: `docs/plans/2026-10-14-antigravity-multi-account-switcher/2026-10-14-antigravity-multi-account-switcher-plan.md`
- Rules: `docs/plans/2026-10-14-antigravity-multi-account-switcher/2026-10-14-antigravity-multi-account-switcher-plan.md#rules`
- Evidence: `docs/plans/2026-10-14-antigravity-multi-account-switcher/2026-10-14-antigravity-multi-account-switcher-evidence.md`
- Benchmark: `docs/plans/2026-10-14-antigravity-multi-account-switcher/2026-10-14-antigravity-multi-account-switcher-benchmark.md`
- Actual status: `docs/plans/2026-10-14-antigravity-multi-account-switcher/2026-10-14-antigravity-multi-account-switcher-actual-status.md`

## Evidence Rules

The evidence file explains why the work is known to be correct.

It should contain:

- metadata and companion files;
- evidence rules or evidence template;
- evidence sections such as `E0`, `E1`, or sections by phase/task;
- user report or problem evidence;
- source inspection, codebase facts, and document facts;
- commands run and pass/fail result;
- impact or blast-radius evidence when code/graph behavior changes;
- implementation evidence: files changed and behavior changed;
- validation evidence: build, tests, e2e, screenshots, or traces;
- failures encountered and how they were handled;
- detect-changes before commit;
- commit hash and closure evidence.

Evidence can reference short metric traces, but long metric tables belong in the benchmark file.

### Evidence ID Naming

Use stable, phase-scoped evidence IDs so `plan.md`, `actual-status.md`, `benchmark.md`, and later agents can reference exact proof without ambiguity.

Format:

```text
E<phase>-<item>-<kind><n>
```

Rules:

- `E<phase>` matches the plan phase number: `E0` for `P0`, `E1` for `P1`, `E2` for `P2`, and so on.
- `<item>` matches the checklist item without the dash: `P0A`, `P1A`, `P2B`.
- `<kind>` is plan-local. Choose a short uppercase token that is meaningful for this repo and this plan:
  - `SRC`: Source inspection facts
  - `PROC`: OS process / runtime inspection facts
  - `LOG`: Log file inspection facts
  - `CRED`: Credential Manager inspection facts
  - `TEST`: Automated test run results
  - `SEC`: Security and filter policy facts
  - `API`: Internal API endpoint / protocol facts
- `<n>` is a 1-based sequence number within that phase item and kind.
- Keep the same `<kind>` meaning stable inside one plan.
- Do not reuse an evidence ID for different facts.
- Reference exact evidence IDs from `actual-status.md` and `benchmark.md`; avoid referencing only broad section IDs such as `E1`.
- Use ranges only for compact inventory summaries; use exact IDs when a specific status decision depends on a specific fact.

## E0 - P0 Evidence

Matching plan item(s): `P0-A`

### Source Inspection Facts

- `E0-P0A-SRC1`: `core/account_store.py` implements `AccountStore` handling JSON serialization/deserialization into `accounts_store/<email>.json`. Methods include `store_account`, `get_account_token`, `has_account`, `remove_account`, and `list_stored_emails`. Filename sanitation replaces unsafe characters via regex `[^a-zA-Z0-9_.@-]`.
- `E0-P0A-SRC2`: `core/credential_manager.py` wraps Windows Credential Manager via `ctypes` calling `Advapi32.dll` (`CredReadW`, `CredWriteW`, `CredDeleteW`, `CredFree`). Successfully targets `LegacyGeneric:target=gemini:antigravity` with user `antigravity`.
- `E0-P0A-SRC3`: `core/session_reloader.py` currently implements only window enumeration (`find_antigravity_windows`) and keystroke simulation (`Ctrl+R` via `keybd_event`). It does NOT discover the `language_server.exe` process, does not communicate via HTTP/RPC, and does not execute `GetUserStatus()` or `AuthLogout()`.
- `E0-P0A-SRC4`: `core/state_resolver.py` orchestrates Chrome discovery and AccountStore into `AccountInfo` records (`ACTIVE`, `READY`, `UNAUTH`). Method `switch_account` writes to Credential Manager and invokes `reload_antigravity_window()`, which relies purely on `Ctrl+R` without clearing backend RAM credentials or confirming re-auth.
- `E0-P0A-SRC5`: `scanner/chrome_scanner.py` reads `Local State` and profile `Preferences` files from `%LOCALAPPDATA%\Google\Chrome\User Data`. It enforces `BLACKLIST_KEYWORDS = ["anvien"]` via `is_blacklisted()`, ensuring blacklisted emails are excluded from discovery.
- `E0-P0A-SRC6`: `ui/app_window.py` implements a Tkinter dark-mode interface (`SwitcherApp`) with scrollable card list, status dots, `Always-on-Top` toggle, `[⚡ Chuyển sang]` button, `[💾 Lưu phiên hiện tại]` button, and `[🔄 Làm mới Chrome]` button. Currently lacks live RAM email sync and granular API-level switching status.
- `E0-P0A-SRC7`: `tests/test_all.py` includes 6 test cases (`test_jwt_payload_decode`, `test_blacklist_enforcement`, `test_account_store_json_files`, `test_live_credential_read`, `test_live_chrome_scanner`, `test_state_resolver_with_account_store`). All 6 pass against live system.
- `E0-P0A-SRC8`: `core/jwt_helper.py` provides standalone base64 URL decoding to extract email from `id_token`. `core/vault.py` provides Windows DPAPI token encryption as an alternative/legacy store.

### Runtime & Process Facts

- `E0-P0A-PROC1`: Antigravity language server process running under PID 17516. Executable: `C:\Users\TAM NGUYEN\AppData\Local\Programs\antigravity\resources\bin\language_server.exe`. CLI arguments explicitly contain:
  - `--https_server_port 0` (binds to ephemeral port)
  - `--csrf_token 06404fbc-050a-451b-840b-e39b7da7b334` (random UUID per session)
  - `--app_data_dir antigravity`
  - `--override_ide_name antigravity`
- `E0-P0A-PROC2`: `Get-NetTCPConnection -OwningProcess 17516 -State Listen` shows `language_server.exe` listening on local loopback ports `127.0.0.1:59226` and `127.0.0.1:59227`.
- `E0-P0A-LOG1`: Log file `C:\Users\TAM NGUYEN\AppData\Roaming\Antigravity\logs\language_server.log` confirms `AuthProvider` lifecycle:
  - `auth_provider.go:1323] [AuthProvider] SetEnableBusinessLogin called with enable: true`
  - `server.go:3178] Auth succeeded, refreshing features and managers`
  - Demonstrates that Antigravity maintains its authenticated identity in server RAM, which must be purged via `AuthLogout` and re-established via Re-Auth for full session handoff.
- `E0-P0A-CRED1`: Direct test read of Windows Credential `LegacyGeneric:target=gemini:antigravity` succeeds and extracts active JSON payload containing valid `id_token`, `refresh_token`, and user metadata.
- `E0-P0A-TEST1`: Baseline test execution output:
  ```text
  python -m unittest tests/test_all.py
  ......
  ----------------------------------------------------------------------
  Ran 6 tests in 0.174s
  OK
  ```
- `E0-P0A-SEC1`: Blacklist filter verification confirms `hokinhdoanh.anvien.sales@gmail.com` and `anvien@gmail.com` return `True` for `is_blacklisted()` and are strictly blocked from Chrome account discovery.

## E1 - P1 Evidence

Matching plan item(s): `P1-A`, `P1-B`

### P1-A Execution Evidence

- `E1-P1A-STORE1`: JSON file creation in `accounts_store/<email>.json` with complete token structure (`access_token`, `refresh_token`, `id_token`, `expiry`, `token_type`) and atomic writing via `.tmp` file and `os.replace`. `saved_at` timestamp recorded using `time.time()`. Corrupted and empty JSON files handled gracefully without crashing `has_account`, `get_account_token`, or `list_stored_emails`. Verified via `test_account_store_atomic_write_and_payload`, `test_account_store_corrupted_json`, and `test_account_store_json_files` in `tests/test_all.py`:
  ```text
  python -m unittest -v tests/test_all.py
  test_account_store_atomic_write_and_payload (tests.test_all.TestAntigravitySwitcher.test_account_store_atomic_write_and_payload) ... ok
  test_account_store_corrupted_json (tests.test_all.TestAntigravitySwitcher.test_account_store_corrupted_json) ... ok
  test_account_store_json_files (tests.test_all.TestAntigravitySwitcher.test_account_store_json_files) ... ok
  ```
- `E1-P1A-STORE2`: Filename sanitation verification with various email formats, domain dots, case normalization, plus aliases, special characters, and path traversal prevention. Verified via `test_sanitize_filename_edge_cases` in `tests/test_all.py`:
  - `User.Test@Gmail.COM` -> `user.test@gmail.com.json`
  - `  developer@domain.com.  ` -> `developer@domain.com.json`
  - `dev+test@domain.com` -> `dev+test@domain.com.json`
  - `user#special!name@domain.com` -> `user_special_name@domain.com.json`
  - `user..name@domain.com` -> `user.name@domain.com.json`
  - `../../evil@domain.com` -> `_._evil@domain.com.json` (no directory traversal)
  - `""` and `None` -> `unknown_account.json`
  ```text
  test_sanitize_filename_edge_cases (tests.test_all.TestAntigravitySwitcher.test_sanitize_filename_edge_cases) ... ok
  ```

### P1-B Execution Evidence

- `E1-P1B-CRED1`: Robust Windows Credential Manager integration via `Advapi32.dll` (`CredReadW`, `CredWriteW`, `CredDeleteW`, `CredFree`) targeting `LegacyGeneric:target=gemini:antigravity`:
  - `read_antigravity_token`: Returns `None` gracefully without crashing when target does not exist, when raw content is corrupted or non-UTF8, when JSON is not a dictionary (e.g. array or primitive), or when blob is empty.
  - `write_antigravity_token`: Enforces strict payload validation before serialization; rejects non-dict inputs (`None`, strings, lists, integers), rejects empty dictionaries `{}`, and rejects non-serializable objects to prevent credential corruption.
  - `delete_antigravity_token`: Provides safe wrapper over `CredDeleteW` that returns boolean success and handles non-existent targets gracefully.
  - Verified via round-trip and edge-case unit tests in `tests/test_all.py`:
  ```text
  test_credential_manager_read_edge_cases (tests.test_all.TestAntigravitySwitcher.test_credential_manager_read_edge_cases) ... ok
  test_credential_manager_round_trip_and_delete (tests.test_all.TestAntigravitySwitcher.test_credential_manager_round_trip_and_delete) ... ok
  test_credential_manager_write_validation (tests.test_all.TestAntigravitySwitcher.test_credential_manager_write_validation) ... ok
  test_live_credential_read (tests.test_all.TestAntigravitySwitcher.test_live_credential_read) ... ok
  ```
- `E1-P1B-AUTO1`: Active session capture into `AccountStore` via `AccountManager.auto_capture_active_session()` and `AccountManager.resolve_accounts()`. Reads active credential from Windows Credential Manager, extracts email from `id_token` JWT payload, and persists complete token structure into `accounts_store/<email>.json`. Verified in `tests/test_all.py`:
  ```text
  test_auto_capture_active_session (tests.test_all.TestAntigravitySwitcher.test_auto_capture_active_session) ... ok
  test_state_resolver_with_account_store (tests.test_all.TestAntigravitySwitcher.test_state_resolver_with_account_store) ... ok
  ```

## E2 - P2 Evidence

Matching plan item(s): `P2-A`, `P2-B`, `P2-C`

Planned Evidence Targets:
- `E2-P2A-DISC1`: Automated discovery of `language_server.exe` PID, listening port, and `--csrf_token` argument from process table and TCP connection state.
- `E2-P2B-API1`: `GetUserStatus()` invocation returning active email and display name from language server RAM.
- `E2-P2B-API2`: `AuthLogout()` invocation returning HTTP 200 / success and clearing active session in language server RAM.
- `E2-P2C-REAUTH1`: Re-Auth trigger signaling language server to re-read Windows Credential and log `Auth succeeded` for Account B.

## E3 - P3 Evidence

Matching plan item(s): `P3-A`, `P3-B`, `P3-C`

Planned Evidence Targets:
- `E3-P3A-SCAN1`: Multi-profile Chrome discovery output with zero blacklisted items.
- `E3-P3B-STATE1`: State machine output matrix resolving 🟢 ACTIVE, 🟢 READY, 🟡 UNAUTH.
- `E3-P3C-SW1`: Execution of full 4-step hot-swap loop (GetUserStatus -> AuthLogout -> Credential write -> ReAuth trigger).

## E4 - P4 Evidence

Matching plan item(s): `P4-A`, `P4-B`, `P4-C`

Planned Evidence Targets:
- `E4-P4A-UI1`: Tkinter GUI rendering accounts with exact color dots and status badges.
- `E4-P4B-ACT1`: Click action on `[⚡ Chuyển sang]` executing hot-swap with modal/status bar feedback.
- `E4-P4C-SYNC1`: Click action on `[🔄 Làm mới Chrome]` refreshing account state and updating sync timestamp.

## E5 - P5 Evidence

Matching plan item(s): `P5-A`, `P5-B`

Planned Evidence Targets:
- `E5-P5A-UNIT1`: Unit test suite expansion verifying all new API connector and state resolver methods.
- `E5-P5B-REG1`: Full test suite run showing 100% passing tests with zero regressions.

## Closure Evidence

Use this section for final detect-changes, commit hash, and closure evidence when the plan reaches completion.

- `EC-PNC-GIT1`: Target commit hash for completed plan set.
- `EC-PNC-TREE1`: Clean repository worktree state verification.
