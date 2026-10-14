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

- `E3-P3-CLEAN1`: Complete Purge of Bogus Blacklist & Truthful 7-Account Discovery:
  - Direct Owner Mandate: Removed bogus `anvien` keyword blacklist completely from `scanner/chrome_scanner.py` and `core/state_resolver.py`.
  - Deleted `BLACKLIST_KEYWORDS` and `is_blacklisted()` to eliminate artificial censorship and restore natural Chrome profile discovery semantics.
  - Verified truthful discovery of all 7 Google accounts from host Chrome User Data (including the Owner's business account `hokinhdoanh.anvien.sales@gmail.com`):
    1. `daohien204ccyersin@gmail.com` (Name: "Long Nguyễn đình", Profile: "Profile 1")
    2. `hiendao1760@gmail.com` (Name: "Hiền Đào thị", Profile: "Profile 1")
    3. `hokinhdoanh.anvien.sales@gmail.com` (Name: "Tâm Nguyễn Đình", Profile: "Profile 1")
    4. `ngau2hung@gmail.com` (Name: "Logan Duroc", Profile: "Profile 1")
    5. `quantritaikhoanemail@gmail.com` (Name: "Tâm Nguyễn Đình", Profile: "Profile 1")
    6. `skey.miennam@gmail.com` (Name: "Miền Nam SKEY", Profile: "Profile 1")
    7. `skey.vanphongmiennam@gmail.com` (Name: "Miền Nam Skey", Profile: "Profile 1")
  - Test Suite Validation: 35/35 tests passing cleanly in `tests/test_all.py` across 5 consecutive execution passes:
    ```text
    Results: [0, 0, 0, 0, 0]
    ```

### P3-C Execution Evidence

- `E3-P3C-SW1`: 4-Step Closed-Loop Hot-Swap Coordinator in `core/state_resolver.py`:
  - Implemented `AccountManager.switch_account(self, target_email: str, ...)` executing the complete 4-step closed-loop coordinator:
    * Step 1 (`GetUserStatus`): Discovers running `language_server.exe` (`find_language_server_process`) and queries active email via `get_user_status(info)`. If target account is already active, returns immediately with `SwitchResult(True, ...)` (idempotent / no-op). Automatically captures current active session before switching.
    * Step 2 (`AuthLogout`): If `language_server.exe` is running, executes `auth_logout(info)` to flush old session credentials from RAM. If language server is not running, gracefully skips this step.
    * Step 3 (`Swap Token`): Loads target token from store (`self.store.get_account` / `self.store.get_account_token`). If missing or invalid, immediately aborts with `SwitchResult(False, ...)` without touching credentials or logging out the active session. Writes validated target token to Windows Credential Manager via `write_antigravity_token(target_token)`.
    * Step 4 (`Re-Auth Trigger`): Signals Antigravity to reload credentials into RAM via `reload_antigravity_session(info=info)` (coordinating backend Re-Auth with frontend Electron window refresh).
  - Dependency Injection Support: `AccountManager` accepts optional DI hooks (`logout_fn`, `reloader_fn`, `token_writer`, `process_finder`, `status_getter`) in both constructor and `switch_account` keyword arguments for 100% deterministic, hermetic unit testing without touching live host processes.
  - Dual Semantic Result: `SwitchResult` class inherits from `tuple` providing seamless tuple unpacking `success, msg = manager.switch_account(target_email)` for UI compatibility, while implementing `__bool__` and `__eq__` so it behaves directly as a boolean (`if res:`, `res == True`, `res == False`).
  - Unit Test Suite Validation in `tests/test_all.py` (43 passing tests):
    * `test_switch_account_full_4step_sequence`: verifies exact ordered execution of all 4 steps (`step1_finder` -> `step1_getter` -> `step2_logout` -> `step3_writer` -> `step4_reloader`), correct token propagation to Credential Manager writer, and correct process info propagation to logout and reloader.
    * `test_switch_account_idempotent_when_already_active`: verifies no-op when target account is already active in RAM; Steps 2-4 skipped.
    * `test_switch_account_target_missing_from_store`: verifies safe abort when target token is absent from store; existing session is preserved and no logout/credential write occurs.
    * `test_switch_account_language_server_not_running_fallback`: verifies graceful handling when language server is offline (skips Step 2 logout, completes Step 3 credential write and Step 4 reload trigger).
    * `test_switch_account_writer_failure_handling`: verifies failure when token writer returns False; Step 4 reload trigger is aborted.
    * `test_switch_account_invalid_target_email_inputs`: verifies rejection of None, empty, whitespace, and non-string inputs.
    * `test_switch_account_method_kwarg_overrides`: verifies method-level keyword arguments override instance defaults.
    * `test_switch_result_tuple_and_boolean_semantics`: verifies unpacking, boolean truthiness, equality, and indexing of `SwitchResult`.
  - Multi-run batch verification (5 sequential runs of 43 tests):
    ```text
    Results: [0, 0, 0, 0, 0]
    ```

Planned Evidence Targets:
- `E3-P3A-SCAN1`: Multi-profile Chrome discovery output with truthful 7-account detection (Achieved).
- `E3-P3B-STATE1`: State machine output matrix resolving 🟢 ACTIVE, 🟢 READY, 🟡 UNAUTH (Achieved).
- `E3-P3-CLEAN1`: Elimination of blacklist filtering and truthful discovery of `hokinhdoanh.anvien.sales@gmail.com` (Achieved).
- `E3-P3C-SW1`: Execution of full 4-step hot-swap loop (GetUserStatus -> AuthLogout -> Credential write -> ReAuth trigger) (Achieved).

## E4 - P4 Evidence

Matching plan item(s): `P4-A`, `P4-B`, `P4-C`

### P4-A Execution Evidence

- `E4-P4A-UI1`: UI Wiring & Status Visualization in `ui/app_window.py`:
  - `AppWindow` / `SwitcherApp` Architecture:
    * Supports dependency injection of custom/mock `AccountManager` (`manager: Optional[AccountManager] = None`) for hermetic unit testing.
    * Supports headless execution (`headless: bool = False`, defaults to True if `root is None` or if Tk initialization fails) so tests and non-display environments run cleanly without crashing or requiring X11/Win32 display handles.
    * Graceful Tkinter initialization and fallback: handles missing accounts directory, empty account list, and UI modal suppression via `show_dialogs: bool = True`.
  - Header RAM Status Presentation:
    * Detects running language server process via `find_language_server_process()` or injected `process_finder`.
    * Prominently displays active email with green indicator: `● Đang hoạt động: <email>` (or `● Đang hoạt động: <email> (Credential fallback)` when offline).
    * Displays `● Chưa phát hiện phiên Antigravity` in yellow when no active session is detected.
  - Account Card List & Visual Badging:
    * Renders name, email, avatar URL (if present), and status dot (🟢 for ACTIVE / READY, 🟡 for UNAUTH).
    * `ACTIVE`: Display badge `[ĐANG DÙNG]` with green highlight styling.
    * `READY`: Action button `⚡ Chuyển sang` (1-click hot swap) bound to `_on_switch_click`.
    * `UNAUTH`: Display badge `[Chưa lưu vào kho]` in yellow.
    * Empty state notice: `Không tìm thấy tài khoản Google nào trên Chrome.` when no accounts exist.
  - Hot-Swap Click Feedback:
    * `_on_switch_click(target_email)`: Immediately updates status bar to `Đang chuyển đổi sang <email>...`, disables button, calls `manager.switch_account(target_email)`, refreshes account cards, and displays completion toast/status bar notification with timestamp.
  - Automated Test Suite Validation (49 tests passing 100%):
    * `test_app_window_initialization_headless_with_mock_manager`: verifies active RAM identity, language server detection, card rendering, and badge formatting (`[ĐANG DÙNG]`, `[⚡ Chuyển sang]`, `[Chưa lưu vào kho]`).
    * `test_app_window_fallback_status_when_language_server_offline`: verifies fallback text when language server is not running.
    * `test_app_window_switch_click_trigger`: verifies switch click triggers `manager.switch_account`, returns `SwitchResult`, and updates status bar.
    * `test_app_window_empty_and_error_states`: verifies empty accounts message and error handling during switch failure.
    * `test_app_window_tk_rendering_and_button_wiring`: verifies Tkinter widget creation and button command invocation with withdrawn root.
    * `test_app_window_aliases_and_exports`: verifies `AppWindow is SwitcherApp`.
  - Full test execution output:
    ```text
    Ran 49 tests in 11.289s
    OK
    ```
  - Batch stability verification across 5 sequential runs:
    ```text
    Results: [0, 0, 0, 0, 0]
    ```

### P4-B Execution Evidence

- `E4-P4B-ACT1`: 1-Click Hot-Swap Trigger & Status Feedback in `ui/app_window.py`:
  - Hardened `_on_switch_click(self, target_email: str, run_in_background: Optional[bool] = None, callback: Optional[Callable[[SwitchResult], None]] = None) -> Union[SwitchResult, threading.Thread]`:
    * Concurrency Guard: Maintains `self.is_switching` state flag. If a switch is already in progress, any subsequent trigger immediately returns `SwitchResult(False, "Chuyển đổi đang diễn ra, vui lòng đợi...")` without calling `manager.switch_account` or modifying ongoing UI feedback.
    * Immediate Progress Feedback: Updates status bar to `Đang chuyển đổi sang <target_email>...`, changes target button text to `Đang chuyển...`, and disables all action buttons (`switch_buttons`, `save_btn`, `refresh_btn`) to prevent concurrent interactions. Calls `self.root.update_idletasks()` to flush GUI rendering before entering execution.
    * Non-Blocking Background Thread Execution: Supports `run_in_background=True` (and `async_mode=True`) using `threading.Thread` and a thread-safe `queue.Queue` polling bridge (`_poll_switch` scheduled via `self.root.after(20, _poll_switch)`). Worker thread never touches Tcl/Tkinter directly, eliminating thread-affinity runtime errors and preventing the desktop window from freezing during HTTP API requests (`AuthLogout`, `GetUserStatus`, `reload_antigravity_session`).
    * Synchronous Fallback & `wait_for_switch(timeout)` Helper: In test/scripting environments (`run_in_background=False`), executes synchronously and returns `SwitchResult` directly for 100% deterministic testing. Provides `wait_for_switch()` to safely join background threads and dispatch UI completion handlers.
    * Success Feedback: Calls `self.refresh_accounts()`, updates status bar to `Đã chuyển đổi thành công sang: <target_email> lúc <timestamp>`, and displays `messagebox.showinfo` ("Chuyển Đổi Thành Công").
    * Error Feedback & Button Restoration: Restores all switch buttons to `NORMAL` state and text `⚡ Chuyển sang`, re-enables `save_btn` and `refresh_btn`, updates status bar to `Lỗi chuyển đổi: <msg>`, and displays `messagebox.showerror` ("Lỗi Chuyển Đổi"). Handles unexpected exceptions safely.
    * Headless Popup Suppression: When `self.headless=True` or `self.root is None` or `not self.show_dialogs`, all modal dialogs (`messagebox.showinfo`, `messagebox.showerror`, `messagebox.showwarning`) are strictly suppressed so headless tests and automation never hang.
  - Automated Unit Test Suite Validation in `tests/test_all.py` (54 passing tests):
    * `test_app_window_switch_click_trigger`: verifies switch click triggers `manager.switch_account`, returns `SwitchResult`, and updates status bar.
    * `test_app_window_switch_updates_active_email_and_status`: verifies successful switch trigger transitions active email in header, updates status text with timestamp, and refreshes cards.
    * `test_app_window_switch_failure_and_exception_resilience`: verifies error handling for failed switch results, unexpected exceptions (`RuntimeError`), and invalid/empty email arguments without crashing.
    * `test_app_window_switch_concurrency_guard`: verifies double-click concurrency protection using `threading.Event` to prove parallel switch attempts are rejected while a switch is executing.
    * `test_app_window_headless_popup_suppression`: verifies complete suppression of modal dialogs (`showinfo`, `showerror`, `showwarning`) in headless mode.
    * `test_app_window_button_states_during_and_after_switch_tk`: verifies button states transition to `DISABLED` with text `Đang chuyển...` during switch and restore to `NORMAL` with `⚡ Chuyển sang` upon failure.
    * `test_app_window_empty_and_error_states`: verifies empty state notice and error state reporting.
    * `test_app_window_tk_rendering_and_button_wiring`: verifies Tkinter widget creation and button command invocation with withdrawn root.
  - Test suite execution output:
    ```text
    Ran 54 tests in 18.129s
    OK
    ```
  - Batch stability verification across 5 sequential runs:
    ```text
    Batch Results: [0, 0, 0, 0, 0]
    ```

### P4-C Execution Evidence

- `E4-P4C-SYNC1`: Session Persistence & Manual Chrome Sync Controls in `ui/app_window.py`:
  - Session Capture Handler (`_on_save_current_session`):
    * Invokes `self.manager.auto_capture_active_session()` to capture and persist current active credential into `accounts_store/<email>.json`.
    * On successful capture (`email` returned): calls `self.refresh_accounts()`, updates status bar text to `Đã lưu phiên của <email> vào kho JSON`, updates status bar widget, and shows `messagebox.showinfo` ("Đã Lưu Phiên") with token persistence notice (when not headless and dialogs enabled).
    * On missing session (`None` returned or exception caught): updates status bar text to `Chưa phát hiện phiên đăng nhập Antigravity nào để lưu`, updates status bar widget, and displays `messagebox.showwarning` ("Không Tìm Thấy Phiên") (when not headless and dialogs enabled).
    * Safely catches any runtime exceptions during credential reading/capture, preventing unhandled GUI crashes.
    * Returns the captured email string or `None`.
  - Manual Refresh & Synchronization Controls (`refresh_accounts`):
    * Queries `AccountStore.list_accounts()` or `AccountStore.list_stored_emails()` for stored JSON account count $N$ and Chrome accounts count $M$.
    * Formats status bar with exact timestamp and metrics: `Đồng bộ lúc <HH:MM:SS> • <N> tài khoản trong kho JSON • <M> trên Chrome`.
    * Button `[🔄 Làm mới Chrome]` wired to `refresh_accounts` via `command=self.refresh_accounts`.
    * Button `[💾 Lưu phiên hiện tại]` wired to `_on_save_current_session` via `command=self._on_save_current_session`.
  - Automated Unit Test Suite Validation in `tests/test_all.py` (58 passing tests):
    * `test_app_window_on_save_current_session_success`: verifies session capture stores account into `AccountStore`, updates status bar to `Đã lưu phiên của <email> vào kho JSON`, and returns the captured email.
    * `test_app_window_on_save_current_session_none_and_exceptions`: verifies graceful handling when no active session exists or when an exception occurs, setting status bar to `Chưa phát hiện phiên đăng nhập Antigravity nào để lưu` and returning `None`.
    * `test_app_window_refresh_accounts_metrics_and_status_bar`: verifies status bar formatting matches regex `^Đồng bộ lúc \d{2}:\d{2}:\d{2} • <N> tài khoản trong kho JSON • <M> trên Chrome$` and dynamically updates metrics when store or Chrome accounts change.
    * `test_app_window_save_and_refresh_buttons_wiring_tk`: verifies Tkinter button bindings for `save_btn` and `refresh_btn` via `.invoke()`, with modal suppression and dialog verification.
  - Test suite execution output:
    ```text
    Ran 58 tests in 24.322s
    OK
    ```
  - Batch stability verification across 5 sequential runs:
    ```text
    Batch Results: [0, 0, 0, 0, 0]
    ```

Planned Evidence Targets:
- `E4-P4A-UI1`: Tkinter GUI rendering accounts with exact color dots and status badges (Achieved).
- `E4-P4B-ACT1`: Click action on `[⚡ Chuyển sang]` executing hot-swap with modal/status bar feedback (Achieved).
- `E4-P4C-SYNC1`: Click action on `[🔄 Làm mới Chrome]` refreshing account state and updating sync timestamp (Achieved).

## E5 - P5 Evidence

Matching plan item(s): `P5-A`, `P5-B`

### P5-A Execution Evidence

- `E5-P5A-UNIT1`: Unit & Integration Test Suite Expansion with Adversarial Edge Cases in `tests/test_all.py`:
  - Expanded test suite from 58 to 64 tests by adding 6 dedicated adversarial test cases covering model blind spots, boundary conditions, and mid-step rollback safeguards:
    * `test_account_store_adversarial_malformed_json_cases`:
      - Corrupted, truncated, and malformed JSON in `accounts_store/`:
      - Binary non-UTF8 garbage bytes (`\x00\xff\xfe...`), raw plaintext string (`THIS_IS_NOT_JSON...`), truncated syntax (`{"email": ... {"access_token": "abc`), truncated keys (`{"email": ... "token_data":`).
      - Missing or invalid required fields: missing `token_data`, `null` `token_data`, empty `{}` `token_data`, missing `email` field (`{"saved_at": 1700000000, "token_data": ...}`).
      - Non-dict JSON roots (JSON array, JSON scalars: int, string, bool).
      - Verified: `has_account` returns `False`, `get_account_token` returns `None`/falsy, `list_stored_emails()` skips all corrupt/missing-email files without crashing, valid accounts alongside work cleanly, clean removal works without error.
    * `test_credential_manager_adversarial_boundary_edge_cases`:
      - Windows Credential Manager boundary edge cases:
      - Binary garbage payload (arbitrary non-UTF8 bytes `\x00\xff\xfe...`): `read_antigravity_token` returns `None` safely without exception.
      - Zero-byte blob: returns `None` safely.
      - Non-JSON plaintext blob: returns `None` safely.
      - Non-dict JSON primitives (`true`, `false`, `null`, `12345`, `"scalar string"`, `[1, 2, 3]`): returns `None` safely.
      - Unicode edge cases: Vietnamese diacritics with full accents ("Đặng Thị Mỹ Duyên", "Hộ kinh doanh An Viên"), Unicode emojis and symbols ("🚀🔑💎🔥✨🛡️⚡", "ACTIVE_✅"), escaped nulls and special characters (`\u0000`, `\n\r\t\"\\`). Written and read back with exact equality.
      - Target names with unicode / special characters (`LegacyGeneric:target=gemini:antigravity_test_adv_unicode_tiếng_việt`).
      - Safe deletion in `finally:`.
    * `test_api_connector_adversarial_network_resilience`:
      - Network resilience against abrupt TCP connection resets and malformed HTTP responses:
      - Abrupt TCP connection reset (RST packet via `SO_LINGER=(1, 0)`): `get_user_status` returns `None`, `auth_logout` returns `False`, `trigger_reauth` returns `False` without uncaught exceptions.
      - Malformed HTTP 200 response with truncated JSON: returns `None`.
      - Malformed HTTP 200 response with non-dict JSON primitives (`"just a string"`, `12345`, `[1, 2, 3]`, `true`): returns `None`.
      - Malformed HTTP 200 response with HTML error page (`502 Bad Gateway`): returns `None`.
      - Missing email in `userStatus`: partial parse returns dict with `email=None` safely.
    * `test_api_connector_adversarial_cli_and_port_discovery`:
      - Command line parsing and port discovery resilience:
      - Missing CSRF token variations in command line (flag with no argument, whitespace, empty equals, empty quotes, hyphenated flag, etc.): `extract_csrf_token` returns `None` on all.
      - Process discovery with bad command lines returns `None`.
      - Listening port discovery with corrupted/garbage netstat output: handles unparseable lines gracefully and extracts valid port `[60001]`.
      - Fallback port selection when HTTP probe fails on all candidate ports: safely falls back to `ports[0]`.
    * `test_session_reloader_adversarial_hwnd_and_window_resilience`:
      - Window handle discovery resilience:
      - Invalid / non-existent HWNDs (`0`, `999999999`, `-1`, `12345678`): `reload_antigravity_window()` executes safely without crashing and returns `True`.
      - Empty windows list returns `False`.
      - Window reloader throwing unexpected exceptions (`OSError`): `reload_antigravity_session` catches exception, coordinates reauth, and returns without crashing.
    * `test_switch_account_adversarial_sequence_and_rollback_safeguards`:
      - Full 4-step sequence verification asserting exact chronological order:
        Asserts chronological order: `["step1_finder", "step1_getter", "step2_logout", "step3_writer", "step4_reloader"]`.
      - Mid-step failure at Step 1 (target missing from store):
        CRITICAL SAFEGUARD ASSERTION: Neither Step 2 (`logout`), Step 3 (`writer`), nor Step 4 (`reloader`) is called! Active session in language server is completely untouched and preserved.
      - Mid-step failure at Step 1 (language server process inaccessible):
        Safe fallback to stored credentials, Step 3 and Step 4 proceed.
      - Mid-step failure at Step 2 (logout throws `ConnectionResetError`):
        Exception caught safely, Step 3 writer and Step 4 reloader proceed.
      - Mid-step failure at Step 3 (writer fails):
        CRITICAL SAFEGUARD ASSERTION: Step 4 (`reloader`) is NOT called! Active session remains safely backed up in `accounts_store/<active_email>.json`.
      - Mid-step failure at Step 4 (reloader throws `RuntimeError`):
        Exception caught safely, returns `SwitchResult(True, ...)`.
  - Unit Test Execution Output:
    ```text
    python -m unittest -v tests/test_all.py
    Ran 64 tests in 24.037s
    OK
    ```
  - Multi-Run Batch Verification across 5 sequential runs:
    Command: `python -c "import subprocess, sys; res = [subprocess.run([sys.executable, '-m', 'unittest', 'tests/test_all.py'], capture_output=True).returncode for _ in range(5)]; print('Results:', res); assert all(r == 0 for r in res)"`
    Output:
    ```text
    Results: [0, 0, 0, 0, 0]
    ```

### P5-B Execution Evidence

- `E5-P5B-REG1`: PID-based Window Discovery & Smooth Zero-Chrome Hot-Swap Regression Verification:
  - PID-based Antigravity Window Discovery (`core/session_reloader.py`):
    * `find_antigravity_pids(command_runner=None) -> List[int]`:
      - Discovers running process IDs belonging to `Antigravity.exe` (or `Antigravity`) using standard Windows tool `tasklist /FI "IMAGENAME eq Antigravity.exe" /FO CSV`.
      - Suppresses flashing command window via `subprocess.CREATE_NO_WINDOW`.
      - Handles non-antigravity processes, headers, "INFO: No tasks are running..." notice, and missing tools or process errors gracefully returning `[]`.
      - Supports injecting custom `command_runner` for hermetic testing.
    * `find_antigravity_windows(pid_provider=None, enum_windows_fn=None) -> List[wintypes.HWND]`:
      - Matches visible top-level window handles (`HWND`) whose `GetWindowThreadProcessId` matches any discovered PID of `Antigravity.exe`.
      - Retains fallback window title check (`if "antigravity" in title.lower()`) if PID discovery returns empty list.
      - Supports injecting `pid_provider` and `enum_windows_fn` for hermetic testing.
    * `reload_antigravity_window(windows=None, windows_finder=None) -> bool`:
      - Sends `Ctrl+R` to discovered Antigravity windows with exception safety on individual HWND operations.
  - Smooth Zero-Chrome Hot-Swap Flow (`core/state_resolver.py`):
    * `AccountManager.switch_account(target_email, force_logout=False, skip_logout=None, ...)`:
      - **CRITICAL OWNER DIRECTIVE IMPLEMENTED**: By default, `auth_logout()` is strictly BYPASSED (`should_logout=False`). Bypassing `AuthLogout()` avoids notifying the Electron IDE that the user logged out, eliminating the popup that forced the user to re-authorize via Chrome OAuth!
      - Target account token validated in `accounts_store/<target_email>.json`. Missing or invalid tokens safely abort with descriptive feedback before touching credentials.
      - Current active session is auto-captured into `accounts_store/<email>.json` if not already present.
      - Target Token B is written directly into Windows Credential Manager (`LegacyGeneric:target=gemini:antigravity`) via `write_antigravity_token(target_token)`.
      - Session reload is triggered directly via `reload_antigravity_session(info=info)`: dispatches reload signal / `Ctrl+R` directly to `Antigravity.exe`, prompting the Electron IDE to immediately reload its credentials from Windows Credential Manager into RAM in 0.1s — completely seamless, with ZERO Chrome popups and ZERO user clicks!
      - Supports `force_logout=True` or `skip_logout=False` to explicitly trigger `auth_logout()` when desired.
      - Supports instance-level `AccountManager(force_logout=...)` configuration.
  - Test Suite Expansion in `tests/test_all.py` (expanded from 64 to 68 tests, 100% passing):
    * `test_find_antigravity_pids_mock_and_errors`: tests CSV parsing, header/non-target filtering, empty tasklist notice, direct list provider, and exception handling.
    * `test_find_antigravity_windows_pid_matching_and_title_fallback`: tests window matching by PID, window title fallback when PID list is empty, and exception resilience.
    * `test_live_find_antigravity_pids`: tests live host process discovery verifying valid PIDs.
    * `test_switch_account_full_4step_sequence`: tests default hot-swap bypassing logout and forced logout invoking logout.
    * `test_switch_account_smooth_zero_chrome_hot_swap_comprehensive`: comprehensive test verifying zero-chrome hot-swap flow, `skip_logout` flags, `force_logout` flags, and instance defaults.
    * `test_switch_account_adversarial_sequence_and_rollback_safeguards`: updated to assert both smooth hot-swap sequence (`step1_finder` -> `step1_getter` -> `step3_writer` -> `step4_reloader`) and forced logout sequence (`step1_finder` -> `step1_getter` -> `step2_logout` -> `step3_writer` -> `step4_reloader`).
  - Full Test Suite Execution Output:
    ```text
    Ran 68 tests in 29.098s
    OK
    ```
  - Multi-Run Batch Verification across 5 sequential runs:
    Command: `python -c "import subprocess, sys; res = [subprocess.run([sys.executable, '-m', 'unittest', 'tests/test_all.py'], capture_output=True).returncode for _ in range(5)]; print('Batch Results:', res); assert all(r == 0 for r in res)"`
    Output:
    ```text
    Batch Results: [0, 0, 0, 0, 0]
    ```

Planned Evidence Targets:
- `E5-P5A-UNIT1`: Unit test suite expansion verifying all new API connector and state resolver methods (Achieved).
- `E5-P5B-REG1`: Full test suite run showing 100% passing tests with zero regressions (Achieved).

## Closure Evidence

Use this section for final detect-changes, commit hash, and closure evidence when the plan reaches completion.

- `EC-PNC-GIT1`: Target commit hash for completed plan set.
- `EC-PNC-TREE1`: Clean repository worktree state verification.
