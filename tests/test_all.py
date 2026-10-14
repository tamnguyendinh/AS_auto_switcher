"""
Unit tests for Antigravity Multi-Account Switcher (JSON Storage Architecture).
"""

import unittest
import base64
import os
import shutil
import json
import time
from core.jwt_helper import decode_jwt_payload, extract_email_from_token_data
from core.account_store import AccountStore, sanitize_filename
from scanner.chrome_scanner import is_blacklisted, get_chrome_signed_in_accounts
from core.credential_manager import (
    read_antigravity_token,
    write_antigravity_token,
    delete_antigravity_token,
    write_raw_credential,
)
from core.state_resolver import AccountManager
from core.api_connector import (
    LanguageServerInfo,
    extract_csrf_token,
    discover_listening_port,
    discover_listening_ports,
    find_language_server_process,
    get_user_status,
    auth_logout,
    trigger_reauth,
    DEFAULT_REAUTH_ENDPOINTS,
)
from core.session_reloader import (
    find_antigravity_windows,
    reload_antigravity_window,
    reload_antigravity_session,
)
import threading
from http.server import HTTPServer, BaseHTTPRequestHandler



class TestAntigravitySwitcher(unittest.TestCase):

    def setUp(self):
        self.test_store_dir = os.path.join(os.path.dirname(__file__), "test_accounts_store")
        os.makedirs(self.test_store_dir, exist_ok=True)
        self.store = AccountStore(self.test_store_dir)

    def tearDown(self):
        if os.path.exists(self.test_store_dir):
            shutil.rmtree(self.test_store_dir, ignore_errors=True)

    def test_jwt_payload_decode(self):
        header = base64.b64encode(b'{"alg":"RS256"}').decode("utf-8").rstrip("=")
        payload = base64.b64encode(b'{"email":"testuser@gmail.com","sub":"12345"}').decode("utf-8").rstrip("=")
        sig = "invalidsig"
        jwt_mock = f"{header}.{payload}.{sig}"

        decoded = decode_jwt_payload(jwt_mock)
        self.assertIsNotNone(decoded)
        self.assertEqual(decoded.get("email"), "testuser@gmail.com")

        email = extract_email_from_token_data({"id_token": jwt_mock})
        self.assertEqual(email, "testuser@gmail.com")

    def test_blacklist_enforcement(self):
        self.assertTrue(is_blacklisted("hokinhdoanh.anvien.sales@gmail.com"))
        self.assertTrue(is_blacklisted("anvien@gmail.com"))
        self.assertFalse(is_blacklisted("hiendao1760@gmail.com"))
        self.assertFalse(is_blacklisted("mydev@gmail.com"))

    def test_account_store_json_files(self):
        test_email = "developer_b@gmail.com"
        mock_token = {"access_token": "mock_at", "refresh_token": "mock_rt", "id_token": "mock_id"}

        # Store account
        self.assertTrue(self.store.store_account(test_email, mock_token))
        self.assertTrue(self.store.has_account(test_email))

        # Verify JSON file exists on disk
        expected_file = os.path.join(self.test_store_dir, f"{test_email}.json")
        self.assertTrue(os.path.exists(expected_file))

        # Retrieve and verify content
        retrieved = self.store.get_account_token(test_email)
        self.assertEqual(retrieved, mock_token)

        # List stored
        stored = self.store.list_stored_emails()
        self.assertIn(test_email, stored)

        # Remove
        self.assertTrue(self.store.remove_account(test_email))
        self.assertFalse(self.store.has_account(test_email))
        self.assertFalse(os.path.exists(expected_file))

    def test_sanitize_filename_edge_cases(self):
        # Case normalization
        self.assertEqual(sanitize_filename("User.Test@Gmail.COM"), "user.test@gmail.com.json")
        # Trailing dot and whitespace
        self.assertEqual(sanitize_filename("  developer@domain.com.  "), "developer@domain.com.json")
        # Plus alias
        self.assertEqual(sanitize_filename("dev+test@domain.com"), "dev+test@domain.com.json")
        # Special characters in local part
        self.assertEqual(sanitize_filename("user#special!name@domain.com"), "user_special_name@domain.com.json")
        # Consecutive dots
        self.assertEqual(sanitize_filename("user..name@domain.com"), "user.name@domain.com.json")
        # Path traversal prevention
        sanitized_traversal = sanitize_filename("../../evil@domain.com")
        self.assertNotIn("/", sanitized_traversal)
        self.assertNotIn("\\", sanitized_traversal)
        self.assertTrue(sanitized_traversal.endswith(".json"))
        # Empty or None handling
        self.assertEqual(sanitize_filename(""), "unknown_account.json")
        self.assertEqual(sanitize_filename(None), "unknown_account.json")

    def test_account_store_atomic_write_and_payload(self):
        test_email = "atomic_user@example.com"
        full_token = {
            "access_token": "mock_access_token_12345",
            "refresh_token": "mock_refresh_token_67890",
            "id_token": "mock_id_token_abcde",
            "expiry": "2026-10-15T12:00:00Z",
            "token_type": "Bearer"
        }

        # Verify initial write
        t_before = time.time()
        self.assertTrue(self.store.store_account(test_email, full_token))
        t_after = time.time()

        account_path = os.path.join(self.test_store_dir, f"{test_email}.json")
        self.assertTrue(os.path.exists(account_path))

        # Check no .tmp file left behind
        tmp_path = f"{account_path}.tmp"
        self.assertFalse(os.path.exists(tmp_path))

        # Inspect raw JSON file content
        with open(account_path, "r", encoding="utf-8") as f:
            data = json.load(f)

        self.assertEqual(data.get("email"), test_email)
        self.assertIn("saved_at", data)
        self.assertGreaterEqual(data["saved_at"], t_before - 1.0)
        self.assertLessEqual(data["saved_at"], t_after + 1.0)
        self.assertEqual(data.get("token_data"), full_token)

        # Overwrite account and check atomic update
        updated_token = dict(full_token)
        updated_token["access_token"] = "mock_access_token_updated"
        self.assertTrue(self.store.store_account(test_email, updated_token))
        self.assertFalse(os.path.exists(tmp_path))

        retrieved = self.store.get_account_token(test_email)
        self.assertEqual(retrieved.get("access_token"), "mock_access_token_updated")

    def test_account_store_corrupted_json(self):
        corrupt_email = "corrupt_user@example.com"
        corrupt_path = os.path.join(self.test_store_dir, f"{corrupt_email}.json")
        with open(corrupt_path, "w", encoding="utf-8") as f:
            f.write("{ invalid json content ...")

        empty_email = "empty_user@example.com"
        empty_path = os.path.join(self.test_store_dir, f"{empty_email}.json")
        with open(empty_path, "w", encoding="utf-8") as f:
            f.write("")

        # Corrupted / empty files should be handled gracefully
        self.assertFalse(self.store.has_account(corrupt_email))
        self.assertIsNone(self.store.get_account_token(corrupt_email))
        self.assertFalse(self.store.has_account(empty_email))
        self.assertIsNone(self.store.get_account_token(empty_email))

        # list_stored_emails should skip corrupt/empty files without crashing
        stored = self.store.list_stored_emails()
        self.assertNotIn(corrupt_email, stored)
        self.assertNotIn(empty_email, stored)

        # Clean removal of corrupted file
        self.assertTrue(self.store.remove_account(corrupt_email))
        self.assertFalse(os.path.exists(corrupt_path))

    def test_credential_manager_read_edge_cases(self):
        # 1. Non-existent target returns None gracefully
        self.assertIsNone(read_antigravity_token("NonExistent_Target_P1B_999"))

        # 2. Corrupt JSON raw bytes returns None gracefully
        corrupt_target = "LegacyGeneric:target=gemini:antigravity_test_corrupt"
        try:
            write_raw_credential(b"{ invalid json content ...", target=corrupt_target)
            self.assertIsNone(read_antigravity_token(corrupt_target))
        finally:
            delete_antigravity_token(corrupt_target)

        # 3. Non-dict JSON (e.g. array or primitive) returns None
        array_target = "LegacyGeneric:target=gemini:antigravity_test_nondict"
        try:
            write_raw_credential(b'["item1", "item2"]', target=array_target)
            self.assertIsNone(read_antigravity_token(array_target))
        finally:
            delete_antigravity_token(array_target)

        # 4. Empty string / empty bytes returns None
        empty_target = "LegacyGeneric:target=gemini:antigravity_test_empty"
        try:
            write_raw_credential(b"", target=empty_target)
            self.assertIsNone(read_antigravity_token(empty_target))
        finally:
            delete_antigravity_token(empty_target)

    def test_credential_manager_write_validation(self):
        test_target = "LegacyGeneric:target=gemini:antigravity_test_validation"
        try:
            # Non-dict inputs must be rejected
            self.assertFalse(write_antigravity_token(None, target=test_target)) # type: ignore
            self.assertFalse(write_antigravity_token("raw string", target=test_target)) # type: ignore
            self.assertFalse(write_antigravity_token([1, 2, 3], target=test_target)) # type: ignore
            self.assertFalse(write_antigravity_token(12345, target=test_target)) # type: ignore

            # Empty dict must be rejected
            self.assertFalse(write_antigravity_token({}, target=test_target))

            # Non-serializable dict must be rejected
            self.assertFalse(write_antigravity_token({"bad_data": {1, 2, 3}}, target=test_target))
        finally:
            delete_antigravity_token(test_target)

    def test_credential_manager_round_trip_and_delete(self):
        test_target = "LegacyGeneric:target=gemini:antigravity_test_roundtrip"
        test_payload = {
            "access_token": "mock_at_roundtrip",
            "refresh_token": "mock_rt_roundtrip",
            "id_token": "mock_id_roundtrip",
            "email": "roundtrip@example.com",
            "token_type": "Bearer"
        }

        try:
            # 1. Write valid token
            self.assertTrue(write_antigravity_token(test_payload, target=test_target))

            # 2. Read and verify content
            retrieved = read_antigravity_token(target=test_target)
            self.assertIsNotNone(retrieved)
            self.assertEqual(retrieved, test_payload)

            # 3. Delete credential
            self.assertTrue(delete_antigravity_token(target=test_target))

            # 4. Verify gone
            self.assertIsNone(read_antigravity_token(target=test_target))

            # 5. Delete non-existent credential returns False gracefully
            self.assertFalse(delete_antigravity_token(target=test_target))
        finally:
            delete_antigravity_token(test_target)

    def test_live_credential_read(self):
        token_data = read_antigravity_token()
        self.assertIsNotNone(token_data)
        self.assertIsInstance(token_data, dict)
        self.assertIn("id_token", token_data)

    def test_live_chrome_scanner(self):
        accounts = get_chrome_signed_in_accounts()
        self.assertIsInstance(accounts, list)
        self.assertGreater(len(accounts), 0)
        for acc in accounts:
            self.assertFalse(is_blacklisted(acc))

    def test_state_resolver_with_account_store(self):
        manager = AccountManager(self.store)
        active_email, accounts = manager.resolve_accounts()
        self.assertIsNotNone(active_email)
        self.assertGreater(len(accounts), 0)

        # Verify active account is now saved in store as JSON
        self.assertTrue(self.store.has_account(active_email))

        active_found = False
        for acc in accounts:
            if acc.email == active_email:
                self.assertTrue(acc.is_active)
                self.assertEqual(acc.status_code, "ACTIVE")
                self.assertEqual(acc.status_color, "green")
                active_found = True

        self.assertTrue(active_found)

    def test_auto_capture_active_session(self):
        manager = AccountManager(self.store)
        captured_email = manager.auto_capture_active_session()
        self.assertIsNotNone(captured_email)
        self.assertTrue(self.store.has_account(captured_email))
        stored_token = self.store.get_account_token(captured_email)
        self.assertIsNotNone(stored_token)
        self.assertIn("id_token", stored_token)

    def test_extract_csrf_token_formats_and_edge_cases(self):
        token_uuid = "06404fbc-050a-451b-840b-e39b7da7b334"

        # 1. Standard space-separated
        cmd1 = f"language_server.exe --standalone --csrf_token {token_uuid} --port 0"
        self.assertEqual(extract_csrf_token(cmd1), token_uuid)

        # 2. Equals-separated
        cmd2 = f"language_server.exe --standalone --csrf_token={token_uuid} --port 0"
        self.assertEqual(extract_csrf_token(cmd2), token_uuid)

        # 3. Double-quoted
        cmd3 = f'language_server.exe --csrf_token "{token_uuid}" --app_data_dir antigravity'
        self.assertEqual(extract_csrf_token(cmd3), token_uuid)

        # 4. Single-quoted
        cmd4 = f"language_server.exe --csrf_token='{token_uuid}'"
        self.assertEqual(extract_csrf_token(cmd4), token_uuid)

        # 5. Full realistic Antigravity command line
        full_cmd = (
            r'"C:\Users\TAM NGUYEN\AppData\Local\Programs\antigravity\resources\bin\language_server.exe" '
            r'--standalone --override_ide_name antigravity --subclient_type hub --override_ide_version 2.23.0 '
            r'--override_user_agent_name antigravity --https_server_port 0 '
            rf'--csrf_token {token_uuid} --app_data_dir antigravity '
            r'--api_server_url https://generativelanguage.googleapis.com'
        )
        self.assertEqual(extract_csrf_token(full_cmd), token_uuid)

        # 6. Edge cases: missing, empty, non-string, malformed
        self.assertIsNone(extract_csrf_token(None))
        self.assertIsNone(extract_csrf_token(""))
        self.assertIsNone(extract_csrf_token("   "))
        self.assertIsNone(extract_csrf_token(12345))  # type: ignore
        self.assertIsNone(extract_csrf_token("language_server.exe --standalone --https_server_port 0"))
        self.assertIsNone(extract_csrf_token("language_server.exe --csrf_token"))

    def test_language_server_info_dataclass(self):
        info = LanguageServerInfo(pid=17516, csrf_token="test-uuid-1234", port=59227)
        self.assertEqual(info.pid, 17516)
        self.assertEqual(info.csrf_token, "test-uuid-1234")
        self.assertEqual(info.port, 59227)
        self.assertEqual(info.base_url, "http://127.0.0.1:59227")

        info_dup = LanguageServerInfo(pid=17516, csrf_token="test-uuid-1234", port=59227)
        self.assertEqual(info, info_dup)

    def test_discover_listening_port_mocked(self):
        # 1. Invalid PID handling
        self.assertIsNone(discover_listening_port(0))
        self.assertIsNone(discover_listening_port(-1))

        # 2. Single port returned directly
        self.assertEqual(discover_listening_port(9999, ports_provider=lambda pid: [54321]), 54321)

        # 3. Empty ports provider returns None
        self.assertIsNone(discover_listening_port(9999, ports_provider=lambda pid: []))

        # 4. Multiple ports fallback
        self.assertIn(
            discover_listening_port(9999, ports_provider=lambda pid: [59226, 59227]),
            [59226, 59227]
        )

    def test_find_language_server_process_mocked(self):
        mock_uuid = "11112222-3333-4444-5555-666677778888"

        # 1. Successful discovery
        mock_procs = [
            {
                "ProcessId": 8888,
                "CommandLine": f"language_server.exe --csrf_token {mock_uuid}"
            }
        ]
        info = find_language_server_process(
            process_provider=lambda: mock_procs,
            port_provider=lambda pid: 50001
        )
        self.assertIsNotNone(info)
        self.assertEqual(info.pid, 8888)
        self.assertEqual(info.csrf_token, mock_uuid)
        self.assertEqual(info.port, 50001)

        # 2. Missing token
        mock_no_token = [{"ProcessId": 8888, "CommandLine": "language_server.exe --standalone"}]
        self.assertIsNone(find_language_server_process(
            process_provider=lambda: mock_no_token,
            port_provider=lambda pid: 50001
        ))

        # 3. Missing port
        self.assertIsNone(find_language_server_process(
            process_provider=lambda: mock_procs,
            port_provider=lambda pid: None
        ))

        # 4. Empty candidate list
        self.assertIsNone(find_language_server_process(
            process_provider=lambda: [],
            port_provider=lambda pid: 50001
        ))

        # 5. Process provider exception handled gracefully
        def broken_provider():
            raise RuntimeError("OS process query failed")

        self.assertIsNone(find_language_server_process(
            process_provider=broken_provider,
            port_provider=lambda pid: 50001
        ))

    def test_live_find_language_server_process(self):
        info = find_language_server_process()
        # On this live host, Antigravity language_server.exe is running under PID 17516
        if info is not None:
            self.assertGreater(info.pid, 0)
            self.assertIsInstance(info.csrf_token, str)
            self.assertGreater(len(info.csrf_token), 10)
            self.assertGreater(info.port, 1024)
            self.assertTrue(info.base_url.startswith("http://127.0.0.1:"))

    def test_get_user_status_mock_success(self):
        server = MockHttpServerHelper()
        server.start()
        try:
            info = LanguageServerInfo(pid=1234, csrf_token="test-token-uuid-111", port=server.port)

            # Case 1: Standard Connect-RPC response structure
            server.response_status = 200
            server.response_body = json.dumps({
                "userStatus": {
                    "email": "alice@example.com",
                    "name": "Alice Dev",
                    "planStatus": "PRO"
                }
            }).encode("utf-8")

            res1 = get_user_status(info)
            self.assertIsNotNone(res1)
            self.assertEqual(res1["email"], "alice@example.com")
            self.assertEqual(res1["name"], "Alice Dev")
            self.assertEqual(res1["display_name"], "Alice Dev")
            self.assertEqual(res1["status"], "PRO")
            self.assertEqual(server.last_headers.get("x-csrf-token"), info.csrf_token)
            self.assertEqual(server.last_headers.get("x-codeium-csrf-token"), info.csrf_token)
            self.assertIn("application/json", server.last_headers.get("content-type", ""))

            # Case 2: Snake_case response structure
            server.response_body = json.dumps({
                "user_status": {
                    "email": "bob@example.com",
                    "display_name": "Bob Builder",
                    "status": "ACTIVE"
                }
            }).encode("utf-8")

            res2 = get_user_status(info)
            self.assertIsNotNone(res2)
            self.assertEqual(res2["email"], "bob@example.com")
            self.assertEqual(res2["display_name"], "Bob Builder")
            self.assertEqual(res2["status"], "ACTIVE")

            # Case 3: Flat dictionary response structure
            server.response_body = json.dumps({
                "email": "carol@example.com",
                "name": "Carol Danvers",
                "status": "ONLINE"
            }).encode("utf-8")

            res3 = get_user_status(info)
            self.assertIsNotNone(res3)
            self.assertEqual(res3["email"], "carol@example.com")
            self.assertEqual(res3["name"], "Carol Danvers")
            self.assertEqual(res3["status"], "ONLINE")
        finally:
            server.stop()

    def test_get_user_status_mock_errors_and_timeout(self):
        server = MockHttpServerHelper()
        server.start()
        try:
            info = LanguageServerInfo(pid=1234, csrf_token="test-token-uuid-222", port=server.port)

            # 401 Unauthorized
            server.response_status = 401
            server.response_body = json.dumps({"error": "unauthenticated"}).encode("utf-8")
            self.assertIsNone(get_user_status(info))

            # 403 Forbidden
            server.response_status = 403
            server.response_body = json.dumps({"error": "forbidden"}).encode("utf-8")
            self.assertIsNone(get_user_status(info))

            # 500 Internal Server Error
            server.response_status = 500
            server.response_body = json.dumps({"error": "internal error"}).encode("utf-8")
            self.assertIsNone(get_user_status(info))

            # 200 with non-JSON HTML error page
            server.response_status = 200
            server.response_body = b"<html><head><title>502 Bad Gateway</title></head><body>Error</body></html>"
            self.assertIsNone(get_user_status(info))

            # Network timeout
            server.delay = 0.25
            self.assertIsNone(get_user_status(info, timeout=0.03))
            server.delay = 0.0

            # Unreachable closed port
            closed_info = LanguageServerInfo(pid=1234, csrf_token="test-token", port=65534)
            self.assertIsNone(get_user_status(closed_info, timeout=0.1))
        finally:
            server.stop()

    def test_auth_logout_mock_success(self):
        server = MockHttpServerHelper()
        server.start()
        try:
            info = LanguageServerInfo(pid=1234, csrf_token="test-token-uuid-333", port=server.port)

            # 200 OK
            server.response_status = 200
            server.response_body = b"{}"
            self.assertTrue(auth_logout(info))
            self.assertEqual(server.last_headers.get("x-csrf-token"), info.csrf_token)
            self.assertEqual(server.last_headers.get("x-codeium-csrf-token"), info.csrf_token)

            # 204 No Content
            server.response_status = 204
            server.response_body = b""
            self.assertTrue(auth_logout(info))
        finally:
            server.stop()

    def test_auth_logout_mock_errors_and_timeout(self):
        server = MockHttpServerHelper()
        server.start()
        try:
            info = LanguageServerInfo(pid=1234, csrf_token="test-token-uuid-444", port=server.port)

            # 500 Internal Server Error
            server.response_status = 500
            server.response_body = b"{\"error\": \"internal error\"}"
            self.assertFalse(auth_logout(info))

            # 401 Unauthorized
            server.response_status = 401
            self.assertFalse(auth_logout(info))

            # Network timeout
            server.delay = 0.25
            self.assertFalse(auth_logout(info, timeout=0.03))
            server.delay = 0.0

            # Unreachable closed port
            closed_info = LanguageServerInfo(pid=1234, csrf_token="test-token", port=65534)
            self.assertFalse(auth_logout(closed_info, timeout=0.1))
        finally:
            server.stop()

    def test_api_connector_invalid_info_graceful_handling(self):
        # get_user_status invalid arguments
        self.assertIsNone(get_user_status(None))
        self.assertIsNone(get_user_status("not-an-info-object"))  # type: ignore
        self.assertIsNone(get_user_status(LanguageServerInfo(pid=0, csrf_token="", port=0)))
        self.assertIsNone(get_user_status(LanguageServerInfo(pid=123, csrf_token="token", port=-1)))
        self.assertIsNone(get_user_status(LanguageServerInfo(pid=123, csrf_token="", port=8080)))

        # auth_logout invalid arguments
        self.assertFalse(auth_logout(None))
        self.assertFalse(auth_logout("not-an-info-object"))  # type: ignore
        self.assertFalse(auth_logout(LanguageServerInfo(pid=0, csrf_token="", port=0)))
        self.assertFalse(auth_logout(LanguageServerInfo(pid=123, csrf_token="token", port=-1)))
        self.assertFalse(auth_logout(LanguageServerInfo(pid=123, csrf_token="", port=8080)))

        # trigger_reauth invalid arguments
        self.assertFalse(trigger_reauth(None))
        self.assertFalse(trigger_reauth("not-an-info-object"))  # type: ignore
        self.assertFalse(trigger_reauth(LanguageServerInfo(pid=0, csrf_token="", port=0)))
        self.assertFalse(trigger_reauth(LanguageServerInfo(pid=123, csrf_token="token", port=-1)))
        self.assertFalse(trigger_reauth(LanguageServerInfo(pid=123, csrf_token="", port=8080)))

    def test_trigger_reauth_mock_success(self):
        server = MockHttpServerHelper()
        server.start()
        try:
            info = LanguageServerInfo(pid=1234, csrf_token="test-reauth-uuid-555", port=server.port)

            # 200 OK default endpoint
            server.response_status = 200
            server.response_body = b"{}"
            self.assertTrue(trigger_reauth(info))
            self.assertEqual(server.last_headers.get("x-csrf-token"), info.csrf_token)
            self.assertEqual(server.last_headers.get("x-codeium-csrf-token"), info.csrf_token)
            self.assertEqual(server.last_path, DEFAULT_REAUTH_ENDPOINTS[0])

            # 204 No Content
            server.response_status = 204
            server.response_body = b""
            self.assertTrue(trigger_reauth(info))

            # Custom endpoint override
            custom_ep = "/custom/reload/endpoint"
            server.response_status = 200
            server.response_body = b"{}"
            self.assertTrue(trigger_reauth(info, endpoint=custom_ep))
            self.assertEqual(server.last_path, custom_ep)

            # Fallback path progression: first candidate 404, second 200
            server.response_status = 404
            server.path_status_map = {
                DEFAULT_REAUTH_ENDPOINTS[1]: 200,
            }
            self.assertTrue(trigger_reauth(info))
            self.assertEqual(server.last_path, DEFAULT_REAUTH_ENDPOINTS[1])
            server.path_status_map = {}
            server.response_status = 200
        finally:
            server.stop()

    def test_trigger_reauth_mock_errors_and_timeout(self):
        server = MockHttpServerHelper()
        server.start()
        try:
            info = LanguageServerInfo(pid=1234, csrf_token="test-reauth-uuid-666", port=server.port)

            # 500 Internal Server Error
            server.response_status = 500
            server.response_body = b"{\"error\": \"internal error\"}"
            self.assertFalse(trigger_reauth(info))

            # 401 Unauthorized
            server.response_status = 401
            self.assertFalse(trigger_reauth(info))

            # 404 on all candidate paths
            server.response_status = 404
            self.assertFalse(trigger_reauth(info))

            # Network timeout
            server.delay = 0.25
            self.assertFalse(trigger_reauth(info, timeout=0.03))
            server.delay = 0.0

            # Unreachable closed port
            closed_info = LanguageServerInfo(pid=1234, csrf_token="test-token", port=65534)
            self.assertFalse(trigger_reauth(closed_info, timeout=0.1))
        finally:
            server.stop()

    def test_reload_antigravity_session_coordinated_flow(self):
        info = LanguageServerInfo(pid=1234, csrf_token="mock-token", port=59999)

        call_trace = []

        def mock_reauth_ok(inf):
            call_trace.append(("reauth", inf.pid))
            return True

        def mock_reauth_fail(inf):
            call_trace.append(("reauth_fail", inf.pid))
            return False

        def mock_window_ok():
            call_trace.append(("window_ok",))
            return True

        def mock_window_fail():
            call_trace.append(("window_fail",))
            return False

        # Case 1: Reauth ok and reload_window=True -> both called, returns True
        call_trace.clear()
        res = reload_antigravity_session(
            info=info,
            reload_window=True,
            window_reloader=mock_window_ok,
            reauth_trigger=mock_reauth_ok,
        )
        self.assertTrue(res)
        self.assertEqual(call_trace, [("reauth", 1234), ("window_ok",)])

        # Case 2: Reauth ok and reload_window=False -> only reauth called, window reload skipped, returns True
        call_trace.clear()
        res = reload_antigravity_session(
            info=info,
            reload_window=False,
            window_reloader=mock_window_ok,
            reauth_trigger=mock_reauth_ok,
        )
        self.assertTrue(res)
        self.assertEqual(call_trace, [("reauth", 1234)])

        # Case 3: Reauth fails and reload_window=False -> window reload invoked as fallback, returns True
        call_trace.clear()
        res = reload_antigravity_session(
            info=info,
            reload_window=False,
            window_reloader=mock_window_ok,
            reauth_trigger=mock_reauth_fail,
        )
        self.assertTrue(res)
        self.assertEqual(call_trace, [("reauth_fail", 1234), ("window_ok",)])

        # Case 4: Both reauth and window reload fail -> returns False
        call_trace.clear()
        res = reload_antigravity_session(
            info=info,
            reload_window=False,
            window_reloader=mock_window_fail,
            reauth_trigger=mock_reauth_fail,
        )
        self.assertFalse(res)
        self.assertEqual(call_trace, [("reauth_fail", 1234), ("window_fail",)])

        # Case 5: No info (info=None), reload_window=True -> window reload invoked, returns True
        call_trace.clear()
        res = reload_antigravity_session(
            info=None,
            reload_window=True,
            window_reloader=mock_window_ok,
            reauth_trigger=mock_reauth_ok,
        )
        self.assertTrue(res)
        self.assertEqual(call_trace, [("window_ok",)])

        # Case 6: No info (info=None), reload_window=False -> window fallback invoked
        call_trace.clear()
        res = reload_antigravity_session(
            info=None,
            reload_window=False,
            window_reloader=mock_window_fail,
            reauth_trigger=mock_reauth_ok,
        )
        self.assertFalse(res)
        self.assertEqual(call_trace, [("window_fail",)])

        # Case 7: Exception in reauth_trigger or window_reloader handled gracefully
        def mock_reauth_throw(inf):
            raise RuntimeError("unexpected socket explosion")

        def mock_window_throw():
            raise RuntimeError("win32 error")

        res = reload_antigravity_session(
            info=info,
            reload_window=False,
            window_reloader=mock_window_throw,
            reauth_trigger=mock_reauth_throw,
        )
        self.assertFalse(res)

    def test_live_session_reloader_safety(self):
        # find_antigravity_windows should execute and return list without raising
        windows = find_antigravity_windows()
        self.assertIsInstance(windows, list)

        # reload_antigravity_session with info=None and reload_window=False should execute safely
        # and not raise any exception regardless of whether windows exist
        res = reload_antigravity_session(info=None, reload_window=False)
        self.assertIsInstance(res, bool)

    def test_live_get_user_status(self):
        info = find_language_server_process()
        if info is not None:
            status = get_user_status(info)
            self.assertIsNotNone(status)
            self.assertIn("email", status)
            self.assertTrue("@" in status["email"])
            self.assertIsNotNone(status.get("status"))


class MockHttpServerHelper:
    """Helper to spin up an in-process loopback HTTP server on an ephemeral port for testing."""

    def __init__(self):
        self.server = None
        self.thread = None
        self.port = 0
        self.last_headers = {}
        self.last_path = ""
        self.response_status = 200
        self.response_body = b"{}"
        self.response_headers = {"Content-Type": "application/json"}
        self.path_status_map = {}
        self.delay = 0.0

    def start(self):
        helper = self

        class Handler(BaseHTTPRequestHandler):
            def log_message(self, format, *args):
                pass  # suppress stderr log spam

            def do_POST(self):
                if helper.delay > 0:
                    time.sleep(helper.delay)
                helper.last_path = self.path
                helper.last_headers = {k.lower(): v for k, v in self.headers.items()}
                status = helper.path_status_map.get(self.path, helper.response_status)
                self.send_response(status)
                for k, v in helper.response_headers.items():
                    self.send_header(k, v)
                self.send_header("Content-Length", str(len(helper.response_body)))
                self.send_header("Connection", "close")
                self.end_headers()
                self.close_connection = True
                try:
                    self.wfile.write(helper.response_body)
                except (ConnectionResetError, ConnectionAbortedError, BrokenPipeError, OSError):
                    pass

            def do_GET(self):
                self.do_POST()

        self.server = HTTPServer(("127.0.0.1", 0), Handler)
        self.port = self.server.server_port
        self.thread = threading.Thread(target=self.server.serve_forever)
        self.thread.daemon = True
        self.thread.start()
        time.sleep(0.05)

    def stop(self):
        if self.server:
            self.server.shutdown()
            self.server.server_close()
            self.thread.join(timeout=1.0)


if __name__ == "__main__":
    unittest.main()


