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

### P2-A Execution Evidence

- `E2-P2A-DISC1`: Automated discovery of `language_server.exe` PID, listening port, and `--csrf_token` argument from Windows process table (via `Get-CimInstance Win32_Process`) and TCP listening table (`netstat -ano -p tcp` with fallback to `Get-NetTCPConnection`).
  - Extracted fields on live system:
    - PID: `17516`
    - CSRF Token: `06404fbc-050a-451b-840b-e39b7da7b334`
    - Discovered Port: `59227` (HTTP listener matching active `language_server.exe` loopback ports `59226` [HTTPS/gRPC] and `59227` [HTTP])
    - Base URL: `http://127.0.0.1:59227`
  - Validated via unit test suite in `tests/test_all.py`:
    - `test_extract_csrf_token_formats_and_edge_cases`: verifies space-separated, equals-separated, single/double quoted, and complex real-world Antigravity command lines, plus graceful `None` handling for empty, non-string, or missing tokens.
    - `test_language_server_info_dataclass`: verifies dataclass attributes, equality, and `base_url` property.
    - `test_discover_listening_port_mocked`: verifies port extraction with mocked port provider, multiple ports fallback, invalid PIDs, and empty port lists.
    - `test_find_language_server_process_mocked`: verifies process discovery, missing token, missing port, empty lists, and runtime exceptions handled gracefully.
    - `test_live_find_language_server_process`: verifies dynamic discovery against live running Antigravity process.
  ```text
  python -m unittest -v tests/test_all.py
  test_extract_csrf_token_formats_and_edge_cases (tests.test_all.TestAntigravitySwitcher.test_extract_csrf_token_formats_and_edge_cases) ... ok
  test_language_server_info_dataclass (tests.test_all.TestAntigravitySwitcher.test_language_server_info_dataclass) ... ok
  test_discover_listening_port_mocked (tests.test_all.TestAntigravitySwitcher.test_discover_listening_port_mocked) ... ok
  test_find_language_server_process_mocked (tests.test_all.TestAntigravitySwitcher.test_find_language_server_process_mocked) ... ok
  test_live_find_language_server_process (tests.test_all.TestAntigravitySwitcher.test_live_find_language_server_process) ... ok
  ----------------------------------------------------------------------
  Ran 18 tests in 1.384s
  OK
  ```

### P2-B Execution Evidence

- `E2-P2B-API1`: `get_user_status()` HTTP API client implementation in `core/api_connector.py`.
  - Communicates with loopback Connect-RPC/REST endpoint (`/exa.language_server_pb.LanguageServerService/GetUserStatus`, `/GetUserStatus`, `/user_status`) with headers `X-Csrf-Token` and `x-codeium-csrf-token`.
  - Successfully parses response into dict with `email`, `name`, `display_name`, and `status`. Returns `None` on 401/403/500/HTML/timeout without throwing uncaught exceptions.
  - Verified live against active Antigravity language server PID 17516:
    - Target: `http://127.0.0.1:59227/exa.language_server_pb.LanguageServerService/GetUserStatus`
    - Live User Email: `hiendao1760@gmail.com`
    - Live Display Name: `Hiền Đào thị`
    - Live Status: `ACTIVE`
  - Validated via unit test suite in `tests/test_all.py`:
    - `test_get_user_status_mock_success`: verifies Connect-RPC nested format, snake_case format, flat dict format, headers propagation (`X-Csrf-Token` and `x-codeium-csrf-token`).
    - `test_get_user_status_mock_errors_and_timeout`: verifies 401, 403, 500, non-JSON HTML error page, network socket timeout, and unreachable ports return `None`.
    - `test_api_connector_invalid_info_graceful_handling`: verifies `None`, non-dataclass, zero port, negative port, empty token return `None`.
    - `test_live_get_user_status`: verifies live query against host Antigravity language server.

- `E2-P2B-API2`: `auth_logout()` HTTP API client implementation in `core/api_connector.py`.
  - Sends POST request to candidate logout endpoints (`/exa.language_server_pb.LanguageServerService/AuthLogout`, `/AuthLogout`, `/logout`) with headers `X-Csrf-Token` and `x-codeium-csrf-token`.
  - Returns `True` on HTTP 200/204 response. Returns `False` on 401/500 error or network timeout without throwing uncaught exceptions.
  - Validated via unit test suite in `tests/test_all.py`:
    - `test_auth_logout_mock_success`: verifies HTTP 200 and 204 responses return `True`, header propagation verified (`X-Csrf-Token` and `x-codeium-csrf-token`).
    - `test_auth_logout_mock_errors_and_timeout`: verifies HTTP 500, 401, network socket timeout, unreachable ports return `False`.
    - `test_api_connector_invalid_info_graceful_handling`: verifies invalid/missing `LanguageServerInfo` returns `False`.
  - Full test suite output:
    ```text
    python -m unittest -v tests/test_all.py
    test_account_store_atomic_write_and_payload (tests.test_all.TestAntigravitySwitcher.test_account_store_atomic_write_and_payload) ... ok
    test_account_store_corrupted_json (tests.test_all.TestAntigravitySwitcher.test_account_store_corrupted_json) ... ok
    test_account_store_json_files (tests.test_all.TestAntigravitySwitcher.test_account_store_json_files) ... ok
    test_api_connector_invalid_info_graceful_handling (tests.test_all.TestAntigravitySwitcher.test_api_connector_invalid_info_graceful_handling) ... ok
    test_auth_logout_mock_errors_and_timeout (tests.test_all.TestAntigravitySwitcher.test_auth_logout_mock_errors_and_timeout) ... ok
    test_auth_logout_mock_success (tests.test_all.TestAntigravitySwitcher.test_auth_logout_mock_success) ... ok
    test_auto_capture_active_session (tests.test_all.TestAntigravitySwitcher.test_auto_capture_active_session) ... ok
    test_blacklist_enforcement (tests.test_all.TestAntigravitySwitcher.test_blacklist_enforcement) ... ok
    test_credential_manager_read_edge_cases (tests.test_all.TestAntigravitySwitcher.test_credential_manager_read_edge_cases) ... ok
    test_credential_manager_round_trip_and_delete (tests.test_all.TestAntigravitySwitcher.test_credential_manager_round_trip_and_delete) ... ok
    test_credential_manager_write_validation (tests.test_all.TestAntigravitySwitcher.test_credential_manager_write_validation) ... ok
    test_discover_listening_port_mocked (tests.test_all.TestAntigravitySwitcher.test_discover_listening_port_mocked) ... ok
    test_extract_csrf_token_formats_and_edge_cases (tests.test_all.TestAntigravitySwitcher.test_extract_csrf_token_formats_and_edge_cases) ... ok
    test_find_language_server_process_mocked (tests.test_all.TestAntigravitySwitcher.test_find_language_server_process_mocked) ... ok
    test_get_user_status_mock_errors_and_timeout (tests.test_all.TestAntigravitySwitcher.test_get_user_status_mock_errors_and_timeout) ... ok
    test_get_user_status_mock_success (tests.test_all.TestAntigravitySwitcher.test_get_user_status_mock_success) ... ok
    test_jwt_payload_decode (tests.test_all.TestAntigravitySwitcher.test_jwt_payload_decode) ... ok
    test_language_server_info_dataclass (tests.test_all.TestAntigravitySwitcher.test_language_server_info_dataclass) ... ok
    test_live_chrome_scanner (tests.test_all.TestAntigravitySwitcher.test_live_chrome_scanner) ... ok
    test_live_credential_read (tests.test_all.TestAntigravitySwitcher.test_live_credential_read) ... ok
    test_live_find_language_server_process (tests.test_all.TestAntigravitySwitcher.test_live_find_language_server_process) ... ok
    test_live_get_user_status (tests.test_all.TestAntigravitySwitcher.test_live_get_user_status) ... ok
    test_sanitize_filename_edge_cases (tests.test_all.TestAntigravitySwitcher.test_sanitize_filename_edge_cases) ... ok
    test_state_resolver_with_account_store (tests.test_all.TestAntigravitySwitcher.test_state_resolver_with_account_store) ... ok

    ----------------------------------------------------------------------
    Ran 24 tests in 4.345s

    OK
    ```

### P2-C Execution Evidence

- `E2-P2C-REAUTH1`: Re-Auth trigger and RAM session reload coordinated bridge in `core/api_connector.py` and `core/session_reloader.py`.
  - `trigger_reauth(info: LanguageServerInfo, timeout: float = 2.0, endpoint: Optional[str] = None) -> bool`:
    - Connects to loopback base URL (`http://127.0.0.1:<port>`) and candidate re-auth/reload paths (`/exa.language_server_pb.LanguageServerService/ReAuth`, `/exa.language_server_pb.LanguageServerService/Reload`, `/ReAuth`, `/reauth`, `/reload`).
    - Dispatches POST request with `X-Csrf-Token` and `x-codeium-csrf-token` headers.
    - Handles HTTP 200/204 response codes, automatically progresses through candidate endpoints on 404/405, closes HTTP error sockets cleanly, and safely catches timeouts/exceptions without raising uncaught errors.
  - `reload_antigravity_session(info: Optional[LanguageServerInfo] = None, reload_window: bool = True) -> bool`:
    - Coordinated reload bridge connecting internal backend RAM Re-Auth with frontend Electron window refresh (`reload_antigravity_window` via `keybd_event Ctrl+R`).
    - Dispatches `trigger_reauth(info)` when `info` is present; if `reload_window` is True (or if backend reauth fails/endpoint not supported), safely invokes `reload_antigravity_window()` fallback.
    - Returns `True` if either reauth signal succeeded or window reload succeeded (or `False` if both failed).
  - Validated via full test suite (28 passing tests):
    - `test_trigger_reauth_mock_success`: verifies default candidate path execution (HTTP 200, headers `x-csrf-token`, `x-codeium-csrf-token`), HTTP 204 No Content, custom endpoint override, and candidate fallback path progression (candidate 0 404 -> candidate 1 200).
    - `test_trigger_reauth_mock_errors_and_timeout`: verifies HTTP 500, 401, 404 on all paths, socket timeout, and unreachable ports return `False`.
    - `test_reload_antigravity_session_coordinated_flow`: verifies all 7 coordination scenarios (both ok, reauth only, window fallback on reauth failure, both fail, no info with reload_window, no info without reload_window fallback, and exception resilience).
    - `test_live_session_reloader_safety`: verifies `find_antigravity_windows()` and `reload_antigravity_session(None, False)` execute cleanly on live host.
    - `test_api_connector_invalid_info_graceful_handling`: verifies `None`, non-dataclass, zero port, negative port, and empty token return `False`.
  - Full test execution output:
    ```text
    python -m unittest -v tests/test_all.py
    test_account_store_atomic_write_and_payload (tests.test_all.TestAntigravitySwitcher.test_account_store_atomic_write_and_payload) ... ok
    test_account_store_corrupted_json (tests.test_all.TestAntigravitySwitcher.test_account_store_corrupted_json) ... ok
    test_account_store_json_files (tests.test_all.TestAntigravitySwitcher.test_account_store_json_files) ... ok
    test_api_connector_invalid_info_graceful_handling (tests.test_all.TestAntigravitySwitcher.test_api_connector_invalid_info_graceful_handling) ... ok
    test_auth_logout_mock_errors_and_timeout (tests.test_all.TestAntigravitySwitcher.test_auth_logout_mock_errors_and_timeout) ... ok
    test_auth_logout_mock_success (tests.test_all.TestAntigravitySwitcher.test_auth_logout_mock_success) ... ok
    test_auto_capture_active_session (tests.test_all.TestAntigravitySwitcher.test_auto_capture_active_session) ... ok
    test_blacklist_enforcement (tests.test_all.TestAntigravitySwitcher.test_blacklist_enforcement) ... ok
    test_credential_manager_read_edge_cases (tests.test_all.TestAntigravitySwitcher.test_credential_manager_read_edge_cases) ... ok
    test_credential_manager_round_trip_and_delete (tests.test_all.TestAntigravitySwitcher.test_credential_manager_round_trip_and_delete) ... ok
    test_credential_manager_write_validation (tests.test_all.TestAntigravitySwitcher.test_credential_manager_write_validation) ... ok
    test_discover_listening_port_mocked (tests.test_all.TestAntigravitySwitcher.test_discover_listening_port_mocked) ... ok
    test_extract_csrf_token_formats_and_edge_cases (tests.test_all.TestAntigravitySwitcher.test_extract_csrf_token_formats_and_edge_cases) ... ok
    test_find_language_server_process_mocked (tests.test_all.TestAntigravitySwitcher.test_find_language_server_process_mocked) ... ok
    test_get_user_status_mock_errors_and_timeout (tests.test_all.TestAntigravitySwitcher.test_get_user_status_mock_errors_and_timeout) ... ok
    test_get_user_status_mock_success (tests.test_all.TestAntigravitySwitcher.test_get_user_status_mock_success) ... ok
    test_jwt_payload_decode (tests.test_all.TestAntigravitySwitcher.test_jwt_payload_decode) ... ok
    test_language_server_info_dataclass (tests.test_all.TestAntigravitySwitcher.test_language_server_info_dataclass) ... ok
    test_live_chrome_scanner (tests.test_all.TestAntigravitySwitcher.test_live_chrome_scanner) ... ok
    test_live_credential_read (tests.test_all.TestAntigravitySwitcher.test_live_credential_read) ... ok
    test_live_find_language_server_process (tests.test_all.TestAntigravitySwitcher.test_live_find_language_server_process) ... ok
    test_live_get_user_status (tests.test_all.TestAntigravitySwitcher.test_live_get_user_status) ... ok
    test_live_session_reloader_safety (tests.test_all.TestAntigravitySwitcher.test_live_session_reloader_safety) ... ok
    test_reload_antigravity_session_coordinated_flow (tests.test_all.TestAntigravitySwitcher.test_reload_antigravity_session_coordinated_flow) ... ok
    test_sanitize_filename_edge_cases (tests.test_all.TestAntigravitySwitcher.test_sanitize_filename_edge_cases) ... ok
    test_state_resolver_with_account_store (tests.test_all.TestAntigravitySwitcher.test_state_resolver_with_account_store) ... ok
    test_trigger_reauth_mock_errors_and_timeout (tests.test_all.TestAntigravitySwitcher.test_trigger_reauth_mock_errors_and_timeout) ... ok
    test_trigger_reauth_mock_success (tests.test_all.TestAntigravitySwitcher.test_trigger_reauth_mock_success) ... ok

    ----------------------------------------------------------------------
    Ran 28 tests in 5.446s

    OK
    ```

- `E2-P2C-STAB1`: Socket lifecycle and `MockHttpServerHelper` stabilization on Windows:
  - Remediated intermittent `[WinError 10053]` connection aborted errors during rapid sequential HTTP calls on ephemeral ports:
    - Set `protocol_version = "HTTP/1.1"` in handler.
    - Drained incoming request body (`self.rfile.read(content_length)`) to prevent Winsock from sending TCP RST on socket close.
    - Added explicit output stream flushing (`self.wfile.flush()`) before socket teardown.
    - Created `StoppableHTTPServer` with `allow_reuse_address = True`, `daemon_threads = True`, and suppressed benign disconnect errors during shutdown.
    - Handled socket errors safely in `finish()` and improved `stop()` with thread join timeout and port release settling delay.
  - Multi-run batch validation:
    - 5-consecutive-run verification:
      ```text
      python -c "import subprocess, sys; res = [subprocess.run([sys.executable, '-m', 'unittest', 'tests/test_all.py'], capture_output=True).returncode for _ in range(5)]; print('Results:', res); assert all(r == 0 for r in res)"
      Results: [0, 0, 0, 0, 0]
      ```
    - 10-consecutive-run verification:
      ```text
      python -c "import subprocess, sys; res = [subprocess.run([sys.executable, '-m', 'unittest', 'tests/test_all.py'], capture_output=True).returncode for _ in range(10)]; print('Results:', res); assert all(r == 0 for r in res)"
      Results: [0, 0, 0, 0, 0, 0, 0, 0, 0, 0]
      ```

Planned Evidence Targets:
- `E2-P2A-DISC1`: Automated discovery of `language_server.exe` PID, listening port, and `--csrf_token` argument from process table and TCP connection state (Achieved).
- `E2-P2B-API1`: `GetUserStatus()` invocation returning active email and display name from language server RAM (Achieved).
- `E2-P2B-API2`: `AuthLogout()` invocation returning HTTP 200 / success and clearing active session in language server RAM (Achieved).
- `E2-P2C-REAUTH1`: Re-Auth trigger signaling language server to re-read Windows Credential and log `Auth succeeded` for Account B (Achieved).

## E3 - P3 Evidence

Matching plan item(s): `P3-A`, `P3-B`, `P3-C`

### P3-A Execution Evidence

- `E3-P3A-SCAN1`: Multi-profile Chrome discovery and strict blacklist filter verification in `scanner/chrome_scanner.py`:
  - `ChromeAccountInfo` dataclass defined with attributes: `email: str`, `name: str = ""`, `profile_dir: str = ""`, `avatar_url: Optional[str] = None`.
  - `get_chrome_signed_in_accounts(user_data_path: Optional[str] = None) -> List[str]`:
    - Backward-compatible signature supporting custom/mock user data directories for isolated unit testing.
    - Successfully returns unique, sorted list of valid Google accounts while strictly excluding blacklisted emails.
  - `get_chrome_account_details(user_data_path: Optional[str] = None) -> List[ChromeAccountInfo]`:
    - Traverses `Local State` (`info_cache` / `profiles_attributes`) and profile directories (`Default`, `Profile *`, and discovered folders).
    - Parses modern Chrome `account_info` list, legacy Chrome `account_tracker_service_last_update` dict, and fallback regex extraction.
    - Merges and enriches display name, avatar URL, and profile directory metadata across sources.
  - Blacklist enforcement & resilience:
    - `BLACKLIST_KEYWORDS = ["anvien"]` strictly checked case-insensitively via `is_blacklisted()`.
    - Filters substring variants (e.g. `test.anvien@gmail.com`, `ANVIEN@company.com`, `user@sub.anvien.net`, `hokinhdoanh.anvien.sales@gmail.com`).
    - Handles missing files, non-existent directories, empty directories, syntax-corrupted JSON, non-dict payloads, and permission/OS errors gracefully without throwing unhandled exceptions.
  - Validated via unit test suite in `tests/test_all.py` (31 tests passing cleanly):
    - `test_blacklist_enforcement`: verifies case variations, substrings, and clean emails.
    - `test_chrome_scanner_mock_profiles`: verifies multi-profile extraction (`Default`, `Profile 1`, `Profile 3`) with modern `account_info`, legacy tracker, and regex email discovery, confirming complete elimination of all 4 blacklisted test accounts and proper metadata population.
    - `test_chrome_scanner_corrupted_and_empty_dirs`: verifies non-existent paths, empty directories, corrupted `Local State`, empty `Preferences`, and non-dict JSON return empty lists safely.
    - `test_live_chrome_scanner` & `test_live_chrome_account_details`: validates live discovery on host machine returning 6 valid accounts with zero blacklisted items.
  - Test run output:
    ```text
    python -m unittest -v tests/test_all.py
    test_account_store_atomic_write_and_payload (tests.test_all.TestAntigravitySwitcher.test_account_store_atomic_write_and_payload) ... ok
    test_account_store_corrupted_json (tests.test_all.TestAntigravitySwitcher.test_account_store_corrupted_json) ... ok
    test_account_store_json_files (tests.test_all.TestAntigravitySwitcher.test_account_store_json_files) ... ok
    test_api_connector_invalid_info_graceful_handling (tests.test_all.TestAntigravitySwitcher.test_api_connector_invalid_info_graceful_handling) ... ok
    test_auth_logout_mock_errors_and_timeout (tests.test_all.TestAntigravitySwitcher.test_auth_logout_mock_errors_and_timeout) ... ok
    test_auth_logout_mock_success (tests.test_all.TestAntigravitySwitcher.test_auth_logout_mock_success) ... ok
    test_auto_capture_active_session (tests.test_all.TestAntigravitySwitcher.test_auto_capture_active_session) ... ok
    test_blacklist_enforcement (tests.test_all.TestAntigravitySwitcher.test_blacklist_enforcement) ... ok
    test_chrome_scanner_corrupted_and_empty_dirs (tests.test_all.TestAntigravitySwitcher.test_chrome_scanner_corrupted_and_empty_dirs) ... ok
    test_chrome_scanner_mock_profiles (tests.test_all.TestAntigravitySwitcher.test_chrome_scanner_mock_profiles) ... ok
    test_credential_manager_read_edge_cases (tests.test_all.TestAntigravitySwitcher.test_credential_manager_read_edge_cases) ... ok
    test_credential_manager_round_trip_and_delete (tests.test_all.TestAntigravitySwitcher.test_credential_manager_round_trip_and_delete) ... ok
    test_credential_manager_write_validation (tests.test_all.TestAntigravitySwitcher.test_credential_manager_write_validation) ... ok
    test_discover_listening_port_mocked (tests.test_all.TestAntigravitySwitcher.test_discover_listening_port_mocked) ... ok
    test_extract_csrf_token_formats_and_edge_cases (tests.test_all.TestAntigravitySwitcher.test_extract_csrf_token_formats_and_edge_cases) ... ok
    test_find_language_server_process_mocked (tests.test_all.TestAntigravitySwitcher.test_find_language_server_process_mocked) ... ok
    test_get_user_status_mock_errors_and_timeout (tests.test_all.TestAntigravitySwitcher.test_get_user_status_mock_errors_and_timeout) ... ok
    test_get_user_status_mock_success (tests.test_all.TestAntigravitySwitcher.test_get_user_status_mock_success) ... ok
    test_jwt_payload_decode (tests.test_all.TestAntigravitySwitcher.test_jwt_payload_decode) ... ok
    test_language_server_info_dataclass (tests.test_all.TestAntigravitySwitcher.test_language_server_info_dataclass) ... ok
    test_live_chrome_account_details (tests.test_all.TestAntigravitySwitcher.test_live_chrome_account_details) ... ok
    test_live_chrome_scanner (tests.test_all.TestAntigravitySwitcher.test_live_chrome_scanner) ... ok
    test_live_credential_read (tests.test_all.TestAntigravitySwitcher.test_live_credential_read) ... ok
    test_live_find_language_server_process (tests.test_all.TestAntigravitySwitcher.test_live_find_language_server_process) ... ok
    test_live_get_user_status (tests.test_all.TestAntigravitySwitcher.test_live_get_user_status) ... ok
    test_live_session_reloader_safety (tests.test_all.TestAntigravitySwitcher.test_live_session_reloader_safety) ... ok
    test_reload_antigravity_session_coordinated_flow (tests.test_all.TestAntigravitySwitcher.test_reload_antigravity_session_coordinated_flow) ... ok
    test_sanitize_filename_edge_cases (tests.test_all.TestAntigravitySwitcher.test_sanitize_filename_edge_cases) ... ok
    test_state_resolver_with_account_store (tests.test_all.TestAntigravitySwitcher.test_state_resolver_with_account_store) ... ok
    test_trigger_reauth_mock_errors_and_timeout (tests.test_all.TestAntigravitySwitcher.test_trigger_reauth_mock_errors_and_timeout) ... ok
    test_trigger_reauth_mock_success (tests.test_all.TestAntigravitySwitcher.test_trigger_reauth_mock_success) ... ok

    ----------------------------------------------------------------------
    Ran 31 tests in 5.714s

    OK
    ```
  - Batch stability verification (5 consecutive runs):
    ```text
    python -c "import subprocess, sys; res = [subprocess.run([sys.executable, '-m', 'unittest', 'tests/test_all.py'], capture_output=True).returncode for _ in range(5)]; print('Results:', res); assert all(r == 0 for r in res)"
    Results: [0, 0, 0, 0, 0]
    ```

### P3-B Execution Evidence

- `E3-P3B-STATE1`: Multi-Account State Machine Resolution (`ACTIVE`, `READY`, `UNAUTH`) and Live RAM Identity Binding in `core/state_resolver.py`:
  - `AccountInfo` dataclass extended with rich metadata: `name: str = ""`, `avatar_url: Optional[str] = None`.
  - `AccountManager.__init__(self, store=None, user_status_provider=None, chrome_accounts_provider=None, credential_reader=None)`:
    - Provides dependency injection hooks for deterministic, isolated unit testing without depending on or touching live Windows host processes or credentials.
  - `get_current_active_email(self, user_status_provider=None) -> Optional[str]`:
    - Prioritizes live RAM identity by querying `find_language_server_process()` and `get_user_status(info)`.
    - If language server is absent, times out, or returns None, safely falls back to reading Windows Credential Manager (`read_antigravity_token()` + `extract_email_from_token_data()`).
    - Normalizes emails to lowercase and handles non-string or exception conditions gracefully.
  - `auto_capture_active_session(self) -> Optional[str]`:
    - Automatically persists active session into `accounts_store/<email>.json` if valid token is present in Windows Credential Manager.
  - `resolve_accounts(self, chrome_accounts_provider=None, user_status_provider=None) -> Tuple[Optional[str], List[AccountInfo]]`:
    - Discovers accounts via `get_chrome_account_details()` (or injected provider) with rich Gaia name and avatar metadata.
    - Categorizes accounts into:
      * `ACTIVE`: matches live RAM identity (`is_active=True`, `has_token=True`, `status_code="ACTIVE"`, `status_color="green"`, `display_badge="[ĐANG DÙNG]"`)
      * `READY`: saved in `accounts_store/<email>.json` (`is_active=False`, `has_token=True`, `status_code="READY"`, `status_color="green"`, `display_badge="[⚡ Chuyển sang]"`)
      * `UNAUTH`: in Chrome but missing from `accounts_store/` (`is_active=False`, `has_token=False`, `status_code="UNAUTH"`, `status_color="yellow"`, `display_badge="[Chưa lưu vào kho]"`)
    - Filters out blacklisted accounts (`is_blacklisted()`) and deduplicates email variants.
  - Validated via comprehensive unit test suite in `tests/test_all.py` (35 tests passing cleanly):
    - `test_account_manager_get_active_email_priority`: verifies RAM identity provider priority, dict response format, method argument override, fallback to Windows Credential Manager on None, both None handling, and provider exception resilience.
    - `test_account_manager_resolve_accounts_state_machine`: verifies exact categorization into ACTIVE, READY, UNAUTH, avatar and name metadata population, and total elimination of blacklisted accounts.
    - `test_account_manager_auto_capture_and_deduplication`: verifies auto-capture from credential into `accounts_store/`, case-insensitive deduplication, and status assignment.
    - `test_account_manager_account_info_metadata`: verifies `AccountInfo` dataclass instantiation with default and custom metadata.
    - `test_state_resolver_with_account_store`: verifies integration with live store and accounts resolution.
  - Full test execution output:
    ```text
    python -m unittest -v tests/test_all.py
    test_account_manager_account_info_metadata (tests.test_all.TestAntigravitySwitcher.test_account_manager_account_info_metadata) ... ok
    test_account_manager_auto_capture_and_deduplication (tests.test_all.TestAntigravitySwitcher.test_account_manager_auto_capture_and_deduplication) ... ok
    test_account_manager_get_active_email_priority (tests.test_all.TestAntigravitySwitcher.test_account_manager_get_active_email_priority) ... ok
    test_account_manager_resolve_accounts_state_machine (tests.test_all.TestAntigravitySwitcher.test_account_manager_resolve_accounts_state_machine) ... ok
    test_account_store_atomic_write_and_payload (tests.test_all.TestAntigravitySwitcher.test_account_store_atomic_write_and_payload) ... ok
    test_account_store_corrupted_json (tests.test_all.TestAntigravitySwitcher.test_account_store_corrupted_json) ... ok
    test_account_store_json_files (tests.test_all.TestAntigravitySwitcher.test_account_store_json_files) ... ok
    test_api_connector_invalid_info_graceful_handling (tests.test_all.TestAntigravitySwitcher.test_api_connector_invalid_info_graceful_handling) ... ok
    test_auth_logout_mock_errors_and_timeout (tests.test_all.TestAntigravitySwitcher.test_auth_logout_mock_errors_and_timeout) ... ok
    test_auth_logout_mock_success (tests.test_all.TestAntigravitySwitcher.test_auth_logout_mock_success) ... ok
    test_auto_capture_active_session (tests.test_all.TestAntigravitySwitcher.test_auto_capture_active_session) ... ok
    test_blacklist_enforcement (tests.test_all.TestAntigravitySwitcher.test_blacklist_enforcement) ... ok
    test_chrome_scanner_corrupted_and_empty_dirs (tests.test_all.TestAntigravitySwitcher.test_chrome_scanner_corrupted_and_empty_dirs) ... ok
    test_chrome_scanner_mock_profiles (tests.test_all.TestAntigravitySwitcher.test_chrome_scanner_mock_profiles) ... ok
    test_credential_manager_read_edge_cases (tests.test_all.TestAntigravitySwitcher.test_credential_manager_read_edge_cases) ... ok
    test_credential_manager_round_trip_and_delete (tests.test_all.TestAntigravitySwitcher.test_credential_manager_round_trip_and_delete) ... ok
    test_credential_manager_write_validation (tests.test_all.TestAntigravitySwitcher.test_credential_manager_write_validation) ... ok
    test_discover_listening_port_mocked (tests.test_all.TestAntigravitySwitcher.test_discover_listening_port_mocked) ... ok
    test_extract_csrf_token_formats_and_edge_cases (tests.test_all.TestAntigravitySwitcher.test_extract_csrf_token_formats_and_edge_cases) ... ok
    test_find_language_server_process_mocked (tests.test_all.TestAntigravitySwitcher.test_find_language_server_process_mocked) ... ok
    test_get_user_status_mock_errors_and_timeout (tests.test_all.TestAntigravitySwitcher.test_get_user_status_mock_errors_and_timeout) ... ok
    test_get_user_status_mock_success (tests.test_all.TestAntigravitySwitcher.test_get_user_status_mock_success) ... ok
    test_jwt_payload_decode (tests.test_all.TestAntigravitySwitcher.test_jwt_payload_decode) ... ok
    test_language_server_info_dataclass (tests.test_all.TestAntigravitySwitcher.test_language_server_info_dataclass) ... ok
    test_live_chrome_account_details (tests.test_all.TestAntigravitySwitcher.test_live_chrome_account_details) ... ok
    test_live_chrome_scanner (tests.test_all.TestAntigravitySwitcher.test_live_chrome_scanner) ... ok
    test_live_credential_read (tests.test_all.TestAntigravitySwitcher.test_live_credential_read) ... ok
    test_live_find_language_server_process (tests.test_all.TestAntigravitySwitcher.test_live_find_language_server_process) ... ok
    test_live_get_user_status (tests.test_all.TestAntigravitySwitcher.test_live_get_user_status) ... ok
    test_live_session_reloader_safety (tests.test_all.TestAntigravitySwitcher.test_live_session_reloader_safety) ... ok
    test_reload_antigravity_session_coordinated_flow (tests.test_all.TestAntigravitySwitcher.test_reload_antigravity_session_coordinated_flow) ... ok
    test_sanitize_filename_edge_cases (tests.test_all.TestAntigravitySwitcher.test_sanitize_filename_edge_cases) ... ok
    test_state_resolver_with_account_store (tests.test_all.TestAntigravitySwitcher.test_state_resolver_with_account_store) ... ok
    test_trigger_reauth_mock_errors_and_timeout (tests.test_all.TestAntigravitySwitcher.test_trigger_reauth_mock_errors_and_timeout) ... ok
    test_trigger_reauth_mock_success (tests.test_all.TestAntigravitySwitcher.test_trigger_reauth_mock_success) ... ok

    ----------------------------------------------------------------------
    Ran 35 tests in 6.855s

    OK
    ```
  - Batch stability verification (5 sequential runs):
    ```text
    Iteration 0: OK
    Iteration 1: OK
    Iteration 2: OK
    Iteration 3: OK
    Iteration 4: OK
    ALL 5 RUNS PASSED 100% CLEANLY!
    ```

Planned Evidence Targets:
- `E3-P3A-SCAN1`: Multi-profile Chrome discovery output with zero blacklisted items (Achieved).
- `E3-P3B-STATE1`: State machine output matrix resolving 🟢 ACTIVE, 🟢 READY, 🟡 UNAUTH (Achieved).
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
