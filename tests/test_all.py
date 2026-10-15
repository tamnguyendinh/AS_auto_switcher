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
from scanner.chrome_scanner import (
    get_chrome_signed_in_accounts,
    get_chrome_account_details,
    ChromeAccountInfo,
)
from core.credential_manager import (
    read_antigravity_token,
    write_antigravity_token,
    delete_antigravity_token,
    write_raw_credential,
)
from core.state_resolver import AccountManager, AccountInfo, SwitchResult, DEFAULT_CONTINUATION_PROMPT
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
    load_trajectory,
    smart_focus_conversation,
    send_user_cascade_message,
    discover_active_agent_model,
)
from core.session_reloader import (
    find_antigravity_pids,
    find_antigravity_windows,
    reload_antigravity_window,
    reload_antigravity_session,
    pre_patch_tos_acceptance,
    discover_most_recent_cascade_id,
    capture_active_threads_snapshot,
    get_default_threads_history_dir,
)
from ui.app_window import AppWindow, SwitcherApp
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

    def test_chrome_scanner_truthful_account_discovery(self):
        # Direct verification: Chrome scanner truthfully detects all 7 accounts
        # including the owner's business account (hokinhdoanh.anvien.sales@gmail.com)
        accounts = get_chrome_signed_in_accounts()
        self.assertEqual(len(accounts), 7)
        self.assertIn("hokinhdoanh.anvien.sales@gmail.com", accounts)
        self.assertIn("hiendao1760@gmail.com", accounts)
        self.assertIn("daohien204ccyersin@gmail.com", accounts)
        self.assertIn("ngau2hung@gmail.com", accounts)
        self.assertIn("quantritaikhoanemail@gmail.com", accounts)
        self.assertIn("skey.miennam@gmail.com", accounts)
        self.assertIn("skey.vanphongmiennam@gmail.com", accounts)

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
        self.assertEqual(len(accounts), 7)
        self.assertIn("hokinhdoanh.anvien.sales@gmail.com", accounts)

    def test_live_chrome_account_details(self):
        details = get_chrome_account_details()
        self.assertIsInstance(details, list)
        self.assertEqual(len(details), 7)
        emails = [acc.email for acc in details]
        self.assertIn("hokinhdoanh.anvien.sales@gmail.com", emails)
        for acc in details:
            self.assertIsInstance(acc, ChromeAccountInfo)
            self.assertIsInstance(acc.email, str)
            self.assertTrue("@" in acc.email)
            self.assertIsInstance(acc.name, str)
            self.assertIsInstance(acc.profile_dir, str)

    def test_chrome_scanner_mock_profiles(self):
        mock_user_data = os.path.join(self.test_store_dir, "mock_chrome_user_data")
        os.makedirs(mock_user_data, exist_ok=True)

        # 1. Mock Local State with multiple profiles and one blacklisted
        local_state = {
            "profile": {
                "info_cache": {
                    "Default": {
                        "user_name": "alice@gmail.com",
                        "name": "Alice Work",
                        "gaia_name": "Alice Wonderland",
                        "last_downloaded_gaia_picture_url_with_size": "https://avatar.example.com/alice.png"
                    },
                    "Profile 1": {
                        "user_name": "bob@example.com",
                        "name": "Bob Dev",
                        "gaia_name": "Bob Builder",
                        "picture_url": "https://avatar.example.com/bob.png"
                    },
                    "Profile 2": {
                        "user_name": "blocked.anvien@company.com",
                        "name": "Blocked Anvien",
                        "gaia_name": "Anvien User"
                    }
                }
            }
        }
        with open(os.path.join(mock_user_data, "Local State"), "w", encoding="utf-8") as f:
            json.dump(local_state, f)

        # 2. Mock Default profile Preferences with modern account_info list
        default_dir = os.path.join(mock_user_data, "Default")
        os.makedirs(default_dir, exist_ok=True)
        default_pref = {
            "account_info": [
                {
                    "email": "alice@gmail.com",
                    "full_name": "Alice Wonderland",
                    "given_name": "Alice",
                    "picture_url": "https://avatar.example.com/alice.png"
                },
                {
                    "email": "secondary@gmail.com",
                    "full_name": "Secondary User",
                    "picture_url": "https://avatar.example.com/sec.png"
                },
                {
                    "email": "nested.anvien.user@gmail.com",
                    "full_name": "Nested Anvien"
                }
            ]
        }
        with open(os.path.join(default_dir, "Preferences"), "w", encoding="utf-8") as f:
            json.dump(default_pref, f)

        # 3. Mock Profile 1 Preferences with legacy account_tracker_service_last_update dict
        prof1_dir = os.path.join(mock_user_data, "Profile 1")
        os.makedirs(prof1_dir, exist_ok=True)
        prof1_pref = {
            "account_tracker_service_last_update": {
                "acc_bob": {
                    "email": "bob@example.com",
                    "full_name": "Bob Builder",
                    "picture_url": "https://avatar.example.com/bob.png"
                },
                "acc_legacy": {
                    "email": "legacy.user@example.org",
                    "full_name": "Legacy User",
                    "picture_url": None
                },
                "acc_blacklisted": {
                    "email": "tracker.anvien@example.com",
                    "full_name": "Tracker Anvien"
                }
            }
        }
        with open(os.path.join(prof1_dir, "Preferences"), "w", encoding="utf-8") as f:
            json.dump(prof1_pref, f)

        # 4. Mock Profile 3 with regex unstructured email discovery
        prof3_dir = os.path.join(mock_user_data, "Profile 3")
        os.makedirs(prof3_dir, exist_ok=True)
        with open(os.path.join(prof3_dir, "Preferences"), "w", encoding="utf-8") as f:
            f.write('{"unstructured": "contact regex.user@domain.com or anvien.sub@evil.org for help"}')

        # Test get_chrome_signed_in_accounts with user_data_path
        accounts = get_chrome_signed_in_accounts(user_data_path=mock_user_data)
        expected_accounts = [
            "alice@gmail.com",
            "anvien.sub@evil.org",
            "blocked.anvien@company.com",
            "bob@example.com",
            "legacy.user@example.org",
            "nested.anvien.user@gmail.com",
            "regex.user@domain.com",
            "secondary@gmail.com",
            "tracker.anvien@example.com",
        ]
        self.assertEqual(accounts, expected_accounts)

        # Test get_chrome_account_details with user_data_path
        details = get_chrome_account_details(user_data_path=mock_user_data)
        self.assertEqual(len(details), len(expected_accounts))
        details_by_email = {d.email: d for d in details}

        # Verify rich metadata on alice
        alice_info = details_by_email.get("alice@gmail.com")
        self.assertIsNotNone(alice_info)
        self.assertEqual(alice_info.name, "Alice Wonderland")
        self.assertEqual(alice_info.profile_dir, "Default")
        self.assertEqual(alice_info.avatar_url, "https://avatar.example.com/alice.png")

        # Verify rich metadata on bob
        bob_info = details_by_email.get("bob@example.com")
        self.assertIsNotNone(bob_info)
        self.assertEqual(bob_info.name, "Bob Builder")
        self.assertEqual(bob_info.profile_dir, "Profile 1")
        self.assertEqual(bob_info.avatar_url, "https://avatar.example.com/bob.png")

        # Verify secondary user
        sec_info = details_by_email.get("secondary@gmail.com")
        self.assertIsNotNone(sec_info)
        self.assertEqual(sec_info.name, "Secondary User")
        self.assertEqual(sec_info.profile_dir, "Default")

        # Verify regex user fallback
        regex_info = details_by_email.get("regex.user@domain.com")
        self.assertIsNotNone(regex_info)
        self.assertEqual(regex_info.profile_dir, "Profile 3")

        # Verify anvien accounts are truthfully retained
        anvien_info = details_by_email.get("blocked.anvien@company.com")
        self.assertIsNotNone(anvien_info)
        self.assertEqual(anvien_info.name, "Anvien User")
        self.assertEqual(anvien_info.profile_dir, "Profile 2")

    def test_chrome_scanner_corrupted_and_empty_dirs(self):
        # 1. Non-existent path returns empty list
        non_existent = os.path.join(self.test_store_dir, "does_not_exist_chrome_dir")
        self.assertEqual(get_chrome_signed_in_accounts(user_data_path=non_existent), [])
        self.assertEqual(get_chrome_account_details(user_data_path=non_existent), [])

        # 2. Empty directory returns empty list
        empty_dir = os.path.join(self.test_store_dir, "empty_chrome_dir")
        os.makedirs(empty_dir, exist_ok=True)
        self.assertEqual(get_chrome_signed_in_accounts(user_data_path=empty_dir), [])
        self.assertEqual(get_chrome_account_details(user_data_path=empty_dir), [])

        # 3. Corrupted Local State (syntax error)
        corrupted_dir = os.path.join(self.test_store_dir, "corrupted_chrome_dir")
        os.makedirs(corrupted_dir, exist_ok=True)
        with open(os.path.join(corrupted_dir, "Local State"), "w", encoding="utf-8") as f:
            f.write("{ invalid json corrupted content ...")

        # Profile with empty / corrupted Preferences
        prof_dir = os.path.join(corrupted_dir, "Default")
        os.makedirs(prof_dir, exist_ok=True)
        with open(os.path.join(prof_dir, "Preferences"), "w", encoding="utf-8") as f:
            f.write("")

        # Should execute safely without raising exceptions and return empty
        self.assertEqual(get_chrome_signed_in_accounts(user_data_path=corrupted_dir), [])
        self.assertEqual(get_chrome_account_details(user_data_path=corrupted_dir), [])

        # 4. Non-dict JSON in Local State (e.g. array)
        with open(os.path.join(corrupted_dir, "Local State"), "w", encoding="utf-8") as f:
            f.write('["not", "a", "dict"]')
        self.assertEqual(get_chrome_signed_in_accounts(user_data_path=corrupted_dir), [])
        self.assertEqual(get_chrome_account_details(user_data_path=corrupted_dir), [])

    def test_state_resolver_with_account_store(self):
        manager = AccountManager(self.store)
        active_email, accounts = manager.resolve_accounts()
        self.assertIsNotNone(active_email)
        self.assertGreater(len(accounts), 0)

        # Verify active account or captured credential session is saved in store as JSON
        stored_emails = self.store.list_stored_emails()
        self.assertGreater(len(stored_emails), 0)

        active_found = False
        for acc in accounts:
            if acc.email == active_email:
                self.assertTrue(acc.is_active)
                self.assertEqual(acc.status_code, "ACTIVE")
                self.assertEqual(acc.status_color, "green")
                active_found = True

        if any(acc.email == active_email for acc in accounts):
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

    def test_find_antigravity_pids_mock_and_errors(self):
        # 1. Successful CSV parsing with multiple Antigravity PIDs and non-Antigravity processes
        sample_csv = (
            '"Antigravity.exe","14408","Console","1","156,788 K"\n'
            '"Antigravity.exe","2668","Console","1","279,664 K"\n'
            '"chrome.exe","9999","Console","1","10,000 K"\n'
            '"Antigravity","13716","Console","1","58,504 K"\n'
        )
        pids = find_antigravity_pids(command_runner=lambda cmd: sample_csv)
        self.assertEqual(pids, [14408, 2668, 13716])

        # 2. tasklist output when no matching tasks exist
        info_no_tasks = "INFO: No tasks are running which match the specified criteria.\n"
        pids_empty = find_antigravity_pids(command_runner=lambda cmd: info_no_tasks)
        self.assertEqual(pids_empty, [])

        # 3. Custom runner returning direct list of PIDs
        direct_pids = find_antigravity_pids(command_runner=lambda cmd: [5492, "9680", 0, -5])
        self.assertEqual(direct_pids, [5492, 9680])

        # 4. Runner throwing exception handled gracefully returning []
        def throwing_runner(cmd):
            raise OSError("tasklist executable not found")

        self.assertEqual(find_antigravity_pids(command_runner=throwing_runner), [])

    def test_find_antigravity_windows_pid_matching_and_title_fallback(self):
        from unittest.mock import patch

        # Mock windows data: (hwnd, pid, title, visible)
        mock_windows = [
            (101, 14408, "Project Alpha - Google Antigravity", True),
            (102, 99999, "Chrome Browser", True),
            (103, 2668, "Sub-window", True),
            (104, 14408, "Hidden", False),
        ]

        def fake_enum_windows(enum_proc, lparam):
            for hwnd, pid_val, title, visible in mock_windows:
                def make_set_pid(p):
                    return lambda h, ref: setattr(ref._obj, "value", p)
                def make_set_text(t):
                    def _setter(h, buf, maxlen):
                        buf.value = t
                        return len(t)
                    return _setter

                with patch("core.session_reloader.IsWindowVisible", return_value=visible), \
                     patch("core.session_reloader.GetWindowThreadProcessId", side_effect=make_set_pid(pid_val)), \
                     patch("core.session_reloader.GetWindowTextW", side_effect=make_set_text(title)):
                    enum_proc(hwnd, lparam)
            return True

        # Case 1: PID discovery succeeds -> matches window 101 and 103 (visible, matching PIDs)
        matched_hwnds = find_antigravity_windows(
            pid_provider=lambda: [14408, 2668],
            enum_windows_fn=fake_enum_windows,
        )
        self.assertEqual(matched_hwnds, [101, 103])

        # Case 2: PID discovery returns empty -> fallback to title check containing "antigravity"
        matched_fallback = find_antigravity_windows(
            pid_provider=lambda: [],
            enum_windows_fn=fake_enum_windows,
        )
        self.assertEqual(matched_fallback, [101])

        # Case 3: PID provider raises exception -> falls back to title check safely
        def broken_pid_provider():
            raise RuntimeError("Process query failed")

        matched_exc = find_antigravity_windows(
            pid_provider=broken_pid_provider,
            enum_windows_fn=fake_enum_windows,
        )
        self.assertEqual(matched_exc, [101])

    def test_live_find_antigravity_pids(self):
        pids = find_antigravity_pids()
        self.assertIsInstance(pids, list)
        for pid in pids:
            self.assertIsInstance(pid, int)
            self.assertGreater(pid, 0)

    def test_live_get_user_status(self):
        info = find_language_server_process()
        if info is not None:
            status = get_user_status(info)
            self.assertIsNotNone(status)
            self.assertIn("email", status)
            self.assertTrue("@" in status["email"])
            self.assertIsNotNone(status.get("status"))

    def test_account_manager_get_active_email_priority(self):
        # 1. RAM identity provider takes priority over credential manager
        mock_cred = {"email": "cred_user@example.com"}
        manager1 = AccountManager(
            self.store,
            user_status_provider=lambda: "ram_user@example.com",
            credential_reader=lambda: mock_cred
        )
        self.assertEqual(manager1.get_current_active_email(), "ram_user@example.com")

        # 2. RAM identity provider returning dict format
        manager_dict = AccountManager(
            self.store,
            user_status_provider=lambda: {"email": "dict_ram@example.com"},
            credential_reader=lambda: mock_cred
        )
        self.assertEqual(manager_dict.get_current_active_email(), "dict_ram@example.com")

        # 3. Method argument overrides instance user_status_provider
        self.assertEqual(
            manager1.get_current_active_email(user_status_provider=lambda: "arg_override@example.com"),
            "arg_override@example.com"
        )

        # 4. Fallback to Credential Manager when RAM provider returns None
        header = base64.b64encode(b'{"alg":"RS256"}').decode("utf-8").rstrip("=")
        payload = base64.b64encode(b'{"email":"cred_fallback@example.com"}').decode("utf-8").rstrip("=")
        mock_jwt = f"{header}.{payload}.sig"
        manager2 = AccountManager(
            self.store,
            user_status_provider=lambda: None,
            credential_reader=lambda: {"id_token": mock_jwt}
        )
        self.assertEqual(manager2.get_current_active_email(), "cred_fallback@example.com")

        # 5. Both RAM and Credential Manager return None
        manager3 = AccountManager(
            self.store,
            user_status_provider=lambda: None,
            credential_reader=lambda: None
        )
        self.assertIsNone(manager3.get_current_active_email())

        # 6. Exception resilience in providers
        def broken_ram():
            raise RuntimeError("RPC connection failed")

        def broken_cred():
            raise RuntimeError("OS credential read failed")

        manager4 = AccountManager(
            self.store,
            user_status_provider=broken_ram,
            credential_reader=broken_cred
        )
        self.assertIsNone(manager4.get_current_active_email())

    def test_account_manager_resolve_accounts_state_machine(self):
        # Setup: Store has ready_user token saved on disk
        self.store.store_account("ready_user@example.com", {"access_token": "mock_ready_token"})

        chrome_mock = [
            ChromeAccountInfo(
                email="active_user@example.com",
                name="Active User",
                profile_dir="Default",
                avatar_url="https://avatar.example.com/active.png"
            ),
            ChromeAccountInfo(
                email="ready_user@example.com",
                name="Ready User",
                profile_dir="Profile 1",
                avatar_url=None
            ),
            ChromeAccountInfo(
                email="unauth_user@example.com",
                name="Unauth User",
                profile_dir="Profile 2",
                avatar_url=None
            ),
            ChromeAccountInfo(
                email="hokinhdoanh.anvien.sales@gmail.com",
                name="Tâm Nguyễn Đình",
                profile_dir="Profile 3",
                avatar_url=None
            )
        ]

        manager = AccountManager(
            self.store,
            user_status_provider=lambda: "active_user@example.com",
            chrome_accounts_provider=lambda: chrome_mock,
            credential_reader=lambda: None
        )

        active_email, accounts = manager.resolve_accounts()

        self.assertEqual(active_email, "active_user@example.com")
        # All 4 accounts recognized truthfully
        self.assertEqual(len(accounts), 4)

        acc_map = {acc.email: acc for acc in accounts}

        # 1. Verify ACTIVE account
        active_acc = acc_map.get("active_user@example.com")
        self.assertIsNotNone(active_acc)
        self.assertTrue(active_acc.is_active)
        self.assertTrue(active_acc.has_token)
        self.assertEqual(active_acc.status_code, "ACTIVE")
        self.assertEqual(active_acc.status_color, "green")
        self.assertEqual(active_acc.display_badge, "[ĐANG DÙNG]")
        self.assertEqual(active_acc.name, "Active User")
        self.assertEqual(active_acc.avatar_url, "https://avatar.example.com/active.png")

        # 2. Verify READY account
        ready_acc = acc_map.get("ready_user@example.com")
        self.assertIsNotNone(ready_acc)
        self.assertFalse(ready_acc.is_active)
        self.assertTrue(ready_acc.has_token)
        self.assertEqual(ready_acc.status_code, "READY")
        self.assertEqual(ready_acc.status_color, "green")
        self.assertEqual(ready_acc.display_badge, "[⚡ Chuyển sang]")
        self.assertEqual(ready_acc.name, "Ready User")

        # 3. Verify UNAUTH account
        unauth_acc = acc_map.get("unauth_user@example.com")
        self.assertIsNotNone(unauth_acc)
        self.assertFalse(unauth_acc.is_active)
        self.assertFalse(unauth_acc.has_token)
        self.assertEqual(unauth_acc.status_code, "UNAUTH")
        self.assertEqual(unauth_acc.status_color, "yellow")
        self.assertEqual(unauth_acc.display_badge, "[Chưa lưu vào kho]")
        self.assertEqual(unauth_acc.name, "Unauth User")

        # 4. Verify anvien business account truthfully present with UNAUTH status
        anvien_acc = acc_map.get("hokinhdoanh.anvien.sales@gmail.com")
        self.assertIsNotNone(anvien_acc)
        self.assertFalse(anvien_acc.is_active)
        self.assertFalse(anvien_acc.has_token)
        self.assertEqual(anvien_acc.status_code, "UNAUTH")
        self.assertEqual(anvien_acc.status_color, "yellow")
        self.assertEqual(anvien_acc.display_badge, "[Chưa lưu vào kho]")
        self.assertEqual(anvien_acc.name, "Tâm Nguyễn Đình")

    def test_account_manager_auto_capture_and_deduplication(self):
        # Auto-capture active session from Windows Credential Manager
        header = base64.b64encode(b'{"alg":"RS256"}').decode("utf-8").rstrip("=")
        payload = base64.b64encode(b'{"email":"autocapture@example.com"}').decode("utf-8").rstrip("=")
        mock_jwt = f"{header}.{payload}.sig"
        mock_token = {"id_token": mock_jwt, "access_token": "mock_at"}

        # Custom Chrome provider returning string list with duplicate entries (different cases)
        mock_chrome_strings = [
            "autocapture@example.com",
            "AUTOCAPTURE@EXAMPLE.COM",
            "other@example.com",
            "hokinhdoanh.anvien.sales@gmail.com"
        ]

        manager = AccountManager(
            self.store,
            user_status_provider=lambda: None,  # triggers fallback to credential
            chrome_accounts_provider=lambda: mock_chrome_strings,
            credential_reader=lambda: mock_token
        )

        active_email, accounts = manager.resolve_accounts()

        # Active email resolved via credential fallback and auto-captured into store
        self.assertEqual(active_email, "autocapture@example.com")
        self.assertTrue(self.store.has_account("autocapture@example.com"))

        # Deduplication check: "autocapture@example.com" should only appear once
        # All valid accounts are retained without censorship
        self.assertEqual(len(accounts), 3)
        emails = [a.email for a in accounts]
        self.assertEqual(emails, ["autocapture@example.com", "other@example.com", "hokinhdoanh.anvien.sales@gmail.com"])

        # Auto-captured active account has status ACTIVE
        self.assertTrue(accounts[0].is_active)
        self.assertEqual(accounts[0].status_code, "ACTIVE")
        self.assertEqual(accounts[0].status_color, "green")
        self.assertEqual(accounts[0].display_badge, "[ĐANG DÙNG]")

        # Other account is UNAUTH
        self.assertFalse(accounts[1].is_active)
        self.assertEqual(accounts[1].status_code, "UNAUTH")
        self.assertEqual(accounts[1].status_color, "yellow")
        self.assertEqual(accounts[1].display_badge, "[Chưa lưu vào kho]")

    def test_account_manager_account_info_metadata(self):
        # Verify AccountInfo dataclass instantiation with default and custom metadata
        info_default = AccountInfo(
            email="user@example.com",
            is_active=True,
            has_token=True,
            status_code="ACTIVE",
            status_color="green",
            display_badge="[ĐANG DÙNG]"
        )
        self.assertEqual(info_default.name, "")
        self.assertIsNone(info_default.avatar_url)

        info_meta = AccountInfo(
            email="user@example.com",
            is_active=False,
            has_token=True,
            status_code="READY",
            status_color="green",
            display_badge="[⚡ Chuyển sang]",
            name="John Doe",
            avatar_url="https://avatar.example.com/john.png"
        )
        self.assertEqual(info_meta.name, "John Doe")
        self.assertEqual(info_meta.avatar_url, "https://avatar.example.com/john.png")

    def test_switch_account_full_4step_sequence(self):
        # Setup test accounts in store
        active_email = "account_a@gmail.com"
        target_email = "account_b@gmail.com"
        active_token = {"access_token": "at_a", "id_token": "mock_id_token_a"}
        target_token = {"access_token": "at_b", "id_token": "mock_id_token_b"}

        self.store.store_account(active_email, active_token)
        self.store.store_account(target_email, target_token)

        step_trace = []
        mock_info = LanguageServerInfo(pid=8888, csrf_token="csrf-uuid-test", port=59999)

        def mock_finder():
            step_trace.append("step1_finder")
            return mock_info

        def mock_getter(info):
            step_trace.append(("step1_getter", info.pid))
            return {"email": active_email, "name": "Account A"}

        def mock_logout(info):
            step_trace.append(("step2_logout", info.pid))
            return True

        def mock_writer(token_data):
            step_trace.append(("step3_writer", token_data.get("id_token")))
            return True

        def mock_reloader(info=None):
            step_trace.append(("step4_reloader", info.pid if info else None))
            return True

        manager = AccountManager(
            store=self.store,
            process_finder=mock_finder,
            status_getter=mock_getter,
            logout_fn=mock_logout,
            token_writer=mock_writer,
            reloader_fn=mock_reloader,
        )

        # 1. Default switch: smooth zero-chrome hot-swap (auth_logout bypassed to prevent OAuth popups)
        res = manager.switch_account(target_email)

        self.assertTrue(res)
        self.assertEqual(res, True)
        self.assertTrue(res.success)
        self.assertIn(target_email, res.message)

        # Verify tuple unpacking
        success, msg = res
        self.assertTrue(success)
        self.assertEqual(msg, res.message)

        # Verify default smooth sequence: logout is bypassed!
        self.assertEqual(len(step_trace), 4)
        self.assertEqual(step_trace[0], "step1_finder")
        self.assertEqual(step_trace[1], ("step1_getter", 8888))
        self.assertEqual(step_trace[2], ("step3_writer", "mock_id_token_b"))
        self.assertEqual(step_trace[3], ("step4_reloader", 8888))
        self.assertNotIn(("step2_logout", 8888), step_trace)

        # 2. Forced logout switch: force_logout=True explicitly invokes auth_logout
        step_trace.clear()
        target_email_c = "account_c@gmail.com"
        self.store.store_account(target_email_c, {"access_token": "at_c", "id_token": "mock_id_token_c"})
        res_forced = manager.switch_account(target_email_c, force_logout=True)
        self.assertTrue(res_forced)
        self.assertEqual(len(step_trace), 5)
        self.assertEqual(step_trace[0], "step1_finder")
        self.assertEqual(step_trace[1], ("step1_getter", 8888))
        self.assertEqual(step_trace[2], ("step2_logout", 8888))
        self.assertEqual(step_trace[3], ("step3_writer", "mock_id_token_c"))
        self.assertEqual(step_trace[4], ("step4_reloader", 8888))

    def test_switch_account_idempotent_when_already_active(self):
        active_email = "current_user@gmail.com"
        mock_info = LanguageServerInfo(pid=7777, csrf_token="csrf-uuid-idem", port=59998)
        step_trace = []

        def mock_finder():
            step_trace.append("finder")
            return mock_info

        def mock_getter(info):
            step_trace.append("getter")
            return {"email": active_email, "name": "Current Active"}

        def mock_logout(info):
            step_trace.append("logout")
            return True

        def mock_writer(token_data):
            step_trace.append("writer")
            return True

        def mock_reloader(info=None):
            step_trace.append("reloader")
            return True

        manager = AccountManager(
            store=self.store,
            process_finder=mock_finder,
            status_getter=mock_getter,
            logout_fn=mock_logout,
            token_writer=mock_writer,
            reloader_fn=mock_reloader,
        )

        # Switch to the same account with varied case and surrounding whitespace
        res = manager.switch_account("  CURRENT_USER@GMAIL.COM  ")

        self.assertTrue(res)
        self.assertEqual(res, True)
        self.assertTrue(res.success)
        self.assertIn("hiện đã đang hoạt động", res.message)

        # Finder and getter were called in Step 1, but Steps 2-4 were skipped
        self.assertEqual(step_trace, ["finder", "getter"])

    def test_switch_account_target_missing_from_store(self):
        active_email = "running_user@gmail.com"
        mock_info = LanguageServerInfo(pid=6666, csrf_token="csrf-uuid-miss", port=59997)
        step_trace = []

        def mock_finder():
            step_trace.append("finder")
            return mock_info

        def mock_getter(info):
            step_trace.append("getter")
            return {"email": active_email, "name": "Running User"}

        def mock_logout(info):
            step_trace.append("logout")
            return True

        def mock_writer(token_data):
            step_trace.append("writer")
            return True

        def mock_reloader(info=None):
            step_trace.append("reloader")
            return True

        manager = AccountManager(
            store=self.store,
            process_finder=mock_finder,
            status_getter=mock_getter,
            logout_fn=mock_logout,
            token_writer=mock_writer,
            reloader_fn=mock_reloader,
        )

        # Target email does not exist in store
        res = manager.switch_account("absent_user@gmail.com")

        self.assertFalse(res)
        self.assertEqual(res, False)
        self.assertFalse(res.success)
        self.assertIn("Chưa có file JSON của absent_user@gmail.com trong accounts_store/!", res.message)

        # Safeguard verification: logout, writer, and reloader were NOT called
        self.assertNotIn("logout", step_trace)
        self.assertNotIn("writer", step_trace)
        self.assertNotIn("reloader", step_trace)

    def test_switch_account_language_server_not_running_fallback(self):
        target_email = "offline_target@gmail.com"
        target_token = {"access_token": "offline_at", "id_token": "offline_id"}
        self.store.store_account(target_email, target_token)

        step_trace = []

        def mock_finder():
            step_trace.append("finder")
            return None  # Language server is not running

        def mock_getter(info):
            step_trace.append("getter")
            return None

        def mock_logout(info):
            step_trace.append("logout")
            return True

        def mock_writer(token_data):
            step_trace.append(("writer", token_data.get("id_token")))
            return True

        def mock_reloader(info=None):
            step_trace.append(("reloader", info))
            return True

        manager = AccountManager(
            store=self.store,
            process_finder=mock_finder,
            status_getter=mock_getter,
            logout_fn=mock_logout,
            token_writer=mock_writer,
            reloader_fn=mock_reloader,
        )

        res = manager.switch_account(target_email)

        self.assertTrue(res)
        self.assertEqual(res, True)
        self.assertTrue(res.success)
        self.assertIn(target_email, res.message)

        # Step 2 logout was skipped because language_server is not running
        self.assertNotIn("logout", step_trace)
        self.assertNotIn("getter", step_trace)
        # Step 3 writer was executed with target token
        self.assertIn(("writer", "offline_id"), step_trace)
        # Step 4 reloader was executed with info=None
        self.assertIn(("reloader", None), step_trace)

    def test_switch_account_writer_failure_handling(self):
        target_email = "write_fail_target@gmail.com"
        target_token = {"access_token": "fail_at", "id_token": "fail_id"}
        self.store.store_account(target_email, target_token)

        step_trace = []
        mock_info = LanguageServerInfo(pid=5555, csrf_token="csrf-uuid-wfail", port=59996)

        def mock_finder():
            return mock_info

        def mock_getter(info):
            return {"email": "other_user@gmail.com"}

        def mock_logout(info):
            step_trace.append("logout")
            return True

        def mock_writer(token_data):
            step_trace.append("writer_failed")
            return False  # Write failed

        def mock_reloader(info=None):
            step_trace.append("reloader")
            return True

        manager = AccountManager(
            store=self.store,
            process_finder=mock_finder,
            status_getter=mock_getter,
            logout_fn=mock_logout,
            token_writer=mock_writer,
            reloader_fn=mock_reloader,
        )

        res = manager.switch_account(target_email)

        self.assertFalse(res)
        self.assertEqual(res, False)
        self.assertFalse(res.success)
        self.assertIn("Không thể ghi Token vào Windows Credential Manager!", res.message)

        # Reloader was NOT triggered because token write failed
        self.assertIn("writer_failed", step_trace)
        self.assertNotIn("reloader", step_trace)

    def test_switch_account_invalid_target_email_inputs(self):
        manager = AccountManager(store=self.store)

        for invalid_input in [None, "", "   ", 123, [], {}]:
            res = manager.switch_account(invalid_input)
            self.assertFalse(res)
            self.assertEqual(res, False)
            self.assertFalse(res.success)
            self.assertIn("Email tài khoản không hợp lệ", res.message)

    def test_switch_account_method_kwarg_overrides(self):
        target_email = "override_target@gmail.com"
        target_token = {"access_token": "ovr_at", "id_token": "ovr_id"}
        self.store.store_account(target_email, target_token)

        invoked_calls = []

        def default_finder():
            invoked_calls.append("default_finder")
            return None

        def custom_finder():
            invoked_calls.append("custom_finder")
            return LanguageServerInfo(pid=4444, csrf_token="csrf-ovr", port=59995)

        def custom_getter(info):
            invoked_calls.append("custom_getter")
            return {"email": "prev@gmail.com"}

        def custom_logout(info):
            invoked_calls.append("custom_logout")
            return True

        def custom_writer(token_data):
            invoked_calls.append("custom_writer")
            return True

        def custom_reloader(info=None):
            invoked_calls.append("custom_reloader")
            return True

        manager = AccountManager(
            store=self.store,
            process_finder=default_finder,
        )

        # Pass custom mocks as keyword arguments directly to switch_account (with force_logout=True)
        res = manager.switch_account(
            target_email,
            force_logout=True,
            logout_fn=custom_logout,
            reloader_fn=custom_reloader,
            token_writer=custom_writer,
            process_finder=custom_finder,
            status_getter=custom_getter,
        )

        self.assertTrue(res)
        self.assertNotIn("default_finder", invoked_calls)
        self.assertIn("custom_finder", invoked_calls)
        self.assertIn("custom_getter", invoked_calls)
        self.assertIn("custom_logout", invoked_calls)
        self.assertIn("custom_writer", invoked_calls)
        self.assertIn("custom_reloader", invoked_calls)

    def test_switch_account_smooth_zero_chrome_hot_swap_comprehensive(self):
        """
        Comprehensive test verifying Owner's smooth zero-chrome hot-swap flow:
        - By default, auth_logout is strictly NOT called (eliminating Chrome OAuth login tab)
        - Token is written directly to Windows Credential Manager
        - Session reload (Ctrl+R / Re-Auth) dispatched to IDE
        - Supports skip_logout=True (bypasses logout) and skip_logout=False (invokes logout)
        - Supports force_logout=False (bypasses logout) and force_logout=True (invokes logout)
        - Supports AccountManager instance level force_logout setting
        """
        email_b = "smooth_target_b@gmail.com"
        email_c = "smooth_target_c@gmail.com"
        email_d = "smooth_target_d@gmail.com"
        email_e = "smooth_target_e@gmail.com"
        token_data = {"access_token": "smooth_at", "id_token": "smooth_id"}

        for em in [email_b, email_c, email_d, email_e]:
            self.store.store_account(em, token_data)

        actions = []
        mock_info = LanguageServerInfo(pid=1111, csrf_token="smooth-csrf", port=59999)

        def mock_finder():
            actions.append("finder")
            return mock_info

        def mock_getter(info):
            actions.append("getter")
            return {"email": "active_user@gmail.com"}

        def mock_logout(info):
            actions.append("logout")
            return True

        def mock_writer(tok):
            actions.append("writer")
            return True

        def mock_reloader(info=None):
            actions.append("reloader")
            return True

        # 1. Default switch: zero-chrome hot-swap (no logout)
        mgr = AccountManager(
            store=self.store,
            process_finder=mock_finder,
            status_getter=mock_getter,
            logout_fn=mock_logout,
            token_writer=mock_writer,
            reloader_fn=mock_reloader,
        )
        res1 = mgr.switch_account(email_b)
        self.assertTrue(res1)
        self.assertEqual(actions, ["finder", "getter", "writer", "reloader"])
        self.assertNotIn("logout", actions)

        # 2. skip_logout=True explicitly
        actions.clear()
        res2 = mgr.switch_account(email_c, skip_logout=True)
        self.assertTrue(res2)
        self.assertEqual(actions, ["finder", "getter", "writer", "reloader"])
        self.assertNotIn("logout", actions)

        # 3. skip_logout=False -> logout called
        actions.clear()
        res3 = mgr.switch_account(email_d, skip_logout=False)
        self.assertTrue(res3)
        self.assertEqual(actions, ["finder", "getter", "logout", "writer", "reloader"])

        # 4. force_logout=True -> logout called
        actions.clear()
        res4 = mgr.switch_account(email_e, force_logout=True)
        self.assertTrue(res4)
        self.assertEqual(actions, ["finder", "getter", "logout", "writer", "reloader"])

        # 5. AccountManager instance with force_logout=True
        actions.clear()
        mgr_forced = AccountManager(
            store=self.store,
            process_finder=mock_finder,
            status_getter=mock_getter,
            logout_fn=mock_logout,
            token_writer=mock_writer,
            reloader_fn=mock_reloader,
            force_logout=True,
        )
        res5 = mgr_forced.switch_account(email_b)
        self.assertTrue(res5)
        self.assertEqual(actions, ["finder", "getter", "logout", "writer", "reloader"])

    def test_switch_result_tuple_and_boolean_semantics(self):
        # Truthy SwitchResult
        res_ok = SwitchResult(True, "All good")
        self.assertTrue(isinstance(res_ok, tuple))
        self.assertTrue(bool(res_ok))
        self.assertEqual(res_ok, True)
        self.assertNotEqual(res_ok, False)
        self.assertEqual(res_ok, (True, "All good"))
        self.assertEqual(res_ok[0], True)
        self.assertEqual(res_ok[1], "All good")
        self.assertEqual(res_ok.success, True)
        self.assertEqual(res_ok.message, "All good")
        self.assertIn("SwitchResult", repr(res_ok))

        # Falsy SwitchResult
        res_err = SwitchResult(False, "Something went wrong")
        self.assertTrue(isinstance(res_err, tuple))
        self.assertFalse(bool(res_err))
        self.assertEqual(res_err, False)
        self.assertNotEqual(res_err, True)
        self.assertEqual(res_err, (False, "Something went wrong"))
        self.assertEqual(res_err[0], False)
        self.assertEqual(res_err[1], "Something went wrong")
        self.assertEqual(res_err.success, False)
        self.assertEqual(res_err.message, "Something went wrong")

        # Unpacking
        s, m = res_ok
        self.assertIs(s, True)
        self.assertEqual(m, "All good")

        s0, m0 = res_err
        self.assertIs(s0, False)
        self.assertEqual(m0, "Something went wrong")

    def test_app_window_initialization_headless_with_mock_manager(self):
        # 1. Mock AccountManager with 3 accounts (ACTIVE, READY, UNAUTH)
        self.store.store_account("ready@example.com", {"access_token": "mock_ready"})
        acc_active = AccountInfo(
            email="active@example.com",
            is_active=True,
            has_token=True,
            status_code="ACTIVE",
            status_color="green",
            display_badge="[ĐANG DÙNG]",
            name="Active Developer",
            avatar_url="https://avatar.example.com/active.png"
        )
        acc_ready = AccountInfo(
            email="ready@example.com",
            is_active=False,
            has_token=True,
            status_code="READY",
            status_color="green",
            display_badge="[⚡ Chuyển sang]",
            name="Ready Developer",
            avatar_url=None
        )
        acc_unauth = AccountInfo(
            email="unauth@example.com",
            is_active=False,
            has_token=False,
            status_code="UNAUTH",
            status_color="yellow",
            display_badge="[Chưa lưu vào kho]",
            name="Unsaved Developer",
            avatar_url=None
        )

        mock_manager = AccountManager(
            store=self.store,
            user_status_provider=lambda: "active@example.com",
            chrome_accounts_provider=lambda: [acc_active, acc_ready, acc_unauth],
            process_finder=lambda: LanguageServerInfo(pid=12345, csrf_token="mock-csrf", port=59999),
        )

        # 2. Initialize in headless mode
        app = AppWindow(manager=mock_manager, headless=True)

        # 3. Assert active RAM identity and Language Server detection
        self.assertEqual(app.active_email, "active@example.com")
        self.assertTrue(app.is_language_server_detected)
        self.assertEqual(app.active_status_text, "● Đang hoạt động: active@example.com")
        self.assertEqual(app.active_status_color, "#a6e3a1")

        # 4. Assert accounts list and badges
        self.assertEqual(len(app.rendered_accounts), 3)
        self.assertEqual(app.account_badges["active@example.com"], "[ĐANG DÙNG]")
        self.assertEqual(app.account_badges["ready@example.com"], "[⚡ Chuyển sang]")
        self.assertEqual(app.account_badges["unauth@example.com"], "[Chưa lưu vào kho]")

    def test_app_window_fallback_status_when_language_server_offline(self):
        acc_fallback = AccountInfo(
            email="fallback@example.com",
            is_active=True,
            has_token=True,
            status_code="ACTIVE",
            status_color="green",
            display_badge="[ĐANG DÙNG]"
        )

        mock_manager = AccountManager(
            store=self.store,
            user_status_provider=lambda: "fallback@example.com",
            chrome_accounts_provider=lambda: [acc_fallback],
            process_finder=lambda: None,  # Language server offline
        )

        app = AppWindow(manager=mock_manager, headless=True)

        self.assertEqual(app.active_email, "fallback@example.com")
        self.assertFalse(app.is_language_server_detected)
        self.assertIn("● Đang hoạt động: fallback@example.com (Credential fallback)", app.active_status_text)
        self.assertEqual(app.active_status_color, "#a6e3a1")

    def test_app_window_switch_click_trigger(self):
        target_email = "ready@example.com"
        self.store.store_account(target_email, {"access_token": "mock_ready"})
        switched_calls = []

        def mock_switch(target):
            switched_calls.append(target)
            return SwitchResult(True, f"Đã chuyển đổi thành công sang: {target}")

        mock_manager = AccountManager(
            store=self.store,
            user_status_provider=lambda: "active@example.com",
            chrome_accounts_provider=lambda: [
                AccountInfo(
                    email="ready@example.com",
                    is_active=False,
                    has_token=True,
                    status_code="READY",
                    status_color="green",
                    display_badge="[⚡ Chuyển sang]"
                )
            ]
        )
        mock_manager.switch_account = mock_switch

        app = AppWindow(manager=mock_manager, headless=True, show_dialogs=False)

        res = app._on_switch_click(target_email)

        self.assertTrue(res)
        self.assertEqual(switched_calls, [target_email])
        self.assertIsNotNone(app.last_switch_result)
        self.assertTrue(app.last_switch_result.success)
        self.assertIn(f"Đã chuyển đổi thành công sang: {target_email}", app.status_bar_text)

    def test_app_window_empty_and_error_states(self):
        # 1. Empty accounts state
        mock_empty_manager = AccountManager(
            store=self.store,
            user_status_provider=lambda: None,
            chrome_accounts_provider=lambda: [],
            credential_reader=lambda: None
        )
        app_empty = AppWindow(manager=mock_empty_manager, headless=True)
        self.assertIsNone(app_empty.active_email)
        self.assertEqual(app_empty.active_status_text, "● Chưa phát hiện phiên Antigravity")
        self.assertEqual(app_empty.empty_message, "Không tìm thấy tài khoản Google nào trên Chrome.")
        self.assertEqual(len(app_empty.rendered_accounts), 0)

        # 2. Switch error handling
        mock_err_manager = AccountManager(
            store=self.store,
            user_status_provider=lambda: "active@example.com",
            chrome_accounts_provider=lambda: [],
            credential_reader=lambda: None
        )
        mock_err_manager.switch_account = lambda target: SwitchResult(False, "Lỗi ghi token!")
        app_err = AppWindow(manager=mock_err_manager, headless=True, show_dialogs=False)

        res = app_err._on_switch_click("failed@example.com")
        self.assertFalse(res)
        self.assertFalse(app_err.last_switch_result.success)
        self.assertIn("Lỗi chuyển đổi: Lỗi ghi token!", app_err.status_bar_text)

    def test_app_window_tk_rendering_and_button_wiring(self):
        # Full widget rendering test using Tkinter with a withdrawn root
        import tkinter as tk
        try:
            root = tk.Tk()
            root.withdraw()
        except Exception:
            self.skipTest("Tkinter display not available on this environment")

        try:
            self.store.store_account("ready@example.com", {"access_token": "mock_ready"})
            acc_active = AccountInfo(
                email="active@example.com",
                is_active=True,
                has_token=True,
                status_code="ACTIVE",
                status_color="green",
                display_badge="[ĐANG DÙNG]",
                name="Active User"
            )
            acc_ready = AccountInfo(
                email="ready@example.com",
                is_active=False,
                has_token=True,
                status_code="READY",
                status_color="green",
                display_badge="[⚡ Chuyển sang]",
                name="Ready User"
            )
            mock_manager = AccountManager(
                store=self.store,
                user_status_provider=lambda: "active@example.com",
                chrome_accounts_provider=lambda: [acc_active, acc_ready],
                process_finder=lambda: LanguageServerInfo(pid=9999, csrf_token="csrf-tk", port=59998)
            )
            mock_manager.switch_account = lambda target: SwitchResult(True, f"Switched to {target}")

            app = AppWindow(root=root, manager=mock_manager, show_dialogs=False)

            # Check header widget
            self.assertEqual(app.active_status_label.cget("text"), "● Đang hoạt động: active@example.com")

            # Check switch button rendered for ready account
            self.assertIn("ready@example.com", app.switch_buttons)
            btn = app.switch_buttons["ready@example.com"]
            self.assertIsInstance(btn, tk.Button)
            self.assertEqual(btn.cget("text"), "⚡ Chuyển sang")

            # Invoke button
            btn.invoke()
            self.assertIsNotNone(app.last_switch_result)
            self.assertTrue(app.last_switch_result.success)
        finally:
            try:
                if 'app' in locals() and hasattr(app, "always_on_top_var"):
                    app.always_on_top_var = None
                root.destroy()
            except Exception:
                pass

    def test_app_window_aliases_and_exports(self):
        self.assertIs(AppWindow, SwitcherApp)

    def test_app_window_switch_updates_active_email_and_status(self):
        target_email = "target_success@example.com"
        self.store.store_account(target_email, {"access_token": "mock_target_tok"})

        current_active = ["initial@example.com"]

        def mock_switch(target):
            current_active[0] = target
            return SwitchResult(True, f"Đã chuyển đổi thành công sang: {target}")

        mock_manager = AccountManager(
            store=self.store,
            user_status_provider=lambda: current_active[0],
            chrome_accounts_provider=lambda: [
                AccountInfo(
                    email="initial@example.com",
                    is_active=(current_active[0] == "initial@example.com"),
                    has_token=True,
                    status_code="ACTIVE" if current_active[0] == "initial@example.com" else "READY",
                    status_color="green",
                    display_badge="[ĐANG DÙNG]" if current_active[0] == "initial@example.com" else "[⚡ Chuyển sang]"
                ),
                AccountInfo(
                    email="target_success@example.com",
                    is_active=(current_active[0] == "target_success@example.com"),
                    has_token=True,
                    status_code="ACTIVE" if current_active[0] == "target_success@example.com" else "READY",
                    status_color="green",
                    display_badge="[ĐANG DÙNG]" if current_active[0] == "target_success@example.com" else "[⚡ Chuyển sang]"
                ),
            ],
            process_finder=lambda: LanguageServerInfo(pid=1234, csrf_token="token", port=50000)
        )
        mock_manager.switch_account = mock_switch

        app = AppWindow(manager=mock_manager, headless=True, show_dialogs=False)
        self.assertEqual(app.active_email, "initial@example.com")

        # Execute switch
        res = app._on_switch_click(target_email)
        self.assertTrue(res)
        self.assertTrue(res.success)
        self.assertEqual(app.active_email, target_email)
        self.assertIn(f"Đã chuyển đổi thành công sang: {target_email}", app.status_bar_text)
        self.assertIn(f"● Đang hoạt động: {target_email}", app.active_status_text)
        self.assertFalse(app.is_switching)

    def test_app_window_switch_failure_and_exception_resilience(self):
        # 1. Normal failure message handling
        mock_manager = AccountManager(store=self.store)
        mock_manager.switch_account = lambda target: SwitchResult(False, "Tài khoản không tồn tại!")
        app = AppWindow(manager=mock_manager, headless=True, show_dialogs=False)

        res = app._on_switch_click("nonexistent@example.com")
        self.assertFalse(res)
        self.assertFalse(res.success)
        self.assertIn("Lỗi chuyển đổi: Tài khoản không tồn tại!", app.status_bar_text)
        self.assertFalse(app.is_switching)

        # 2. Unexpected exception handling without crashing
        def blow_up(target):
            raise RuntimeError("Unexpected language server socket crash")

        mock_manager.switch_account = blow_up
        res_ex = app._on_switch_click("crash@example.com")
        self.assertFalse(res_ex)
        self.assertFalse(res_ex.success)
        self.assertIn("Lỗi ngoại lệ: Unexpected language server socket crash", app.status_bar_text)
        self.assertFalse(app.is_switching)

        # 3. Invalid email inputs
        self.assertFalse(app._on_switch_click(None).success)
        self.assertFalse(app._on_switch_click("").success)
        self.assertFalse(app.is_switching)

    def test_app_window_switch_concurrency_guard(self):
        # Concurrency guard: double click does not trigger multiple parallel switches
        switch_started = threading.Event()
        switch_continue = threading.Event()
        calls = []

        def blocking_switch(target):
            calls.append(target)
            switch_started.set()
            switch_continue.wait(timeout=2.0)
            return SwitchResult(True, f"Switched to {target}")

        mock_manager = AccountManager(store=self.store)
        mock_manager.switch_account = blocking_switch
        app = AppWindow(manager=mock_manager, headless=True, show_dialogs=False)

        # Trigger first switch in background thread
        th = app._on_switch_click("account1@example.com", run_in_background=True)
        self.assertTrue(switch_started.wait(timeout=1.0))
        self.assertTrue(app.is_switching)

        # Concurrently attempt second switch while first is running
        res_rejected = app._on_switch_click("account2@example.com")
        self.assertFalse(res_rejected)
        self.assertFalse(res_rejected.success)
        self.assertIn("Chuyển đổi đang diễn ra", res_rejected.message)
        # Verify manager.switch_account was NOT called a second time
        self.assertEqual(calls, ["account1@example.com"])

        # Release first switch and join
        switch_continue.set()
        th.join(timeout=2.0)

        # Verify is_switching reset to False
        self.assertFalse(app.is_switching)
        self.assertIn("Đã chuyển đổi thành công sang: account1@example.com", app.status_bar_text)

        # Subsequent switch can now execute cleanly
        switch_started.clear()
        switch_continue.set()
        res_next = app._on_switch_click("account3@example.com")
        self.assertTrue(res_next)
        self.assertEqual(calls, ["account1@example.com", "account3@example.com"])

    def test_app_window_headless_popup_suppression(self):
        # In headless mode (headless=True or root=None), modal dialogs are strictly suppressed
        from unittest.mock import patch

        mock_manager = AccountManager(store=self.store)
        mock_manager.switch_account = lambda target: SwitchResult(True, "Success")
        mock_manager.auto_capture_active_session = lambda: "captured@example.com"

        app = AppWindow(manager=mock_manager, headless=True, show_dialogs=True)
        self.assertTrue(app.headless)

        with patch("ui.app_window.messagebox.showinfo") as mock_info, \
             patch("ui.app_window.messagebox.showerror") as mock_err, \
             patch("ui.app_window.messagebox.showwarning") as mock_warn:

            # 1. Successful switch -> no popup
            res_ok = app._on_switch_click("target@example.com")
            self.assertTrue(res_ok.success)
            mock_info.assert_not_called()
            mock_err.assert_not_called()

            # 2. Failed switch -> no popup
            mock_manager.switch_account = lambda target: SwitchResult(False, "Failed switch")
            res_err = app._on_switch_click("target@example.com")
            self.assertFalse(res_err.success)
            mock_info.assert_not_called()
            mock_err.assert_not_called()

            # 3. Save session -> no popup
            app._on_save_current_session()
            mock_info.assert_not_called()
            mock_warn.assert_not_called()

            mock_manager.auto_capture_active_session = lambda: None
            app._on_save_current_session()
            mock_warn.assert_not_called()

    def test_app_window_button_states_during_and_after_switch_tk(self):
        import tkinter as tk
        try:
            root = tk.Tk()
            root.withdraw()
        except Exception:
            self.skipTest("Tkinter display not available on this environment")

        try:
            self.store.store_account("ready@example.com", {"access_token": "mock_tok"})
            acc_active = AccountInfo(
                email="active@example.com",
                is_active=True,
                has_token=True,
                status_code="ACTIVE",
                status_color="green",
                display_badge="[ĐANG DÙNG]"
            )
            acc_ready = AccountInfo(
                email="ready@example.com",
                is_active=False,
                has_token=True,
                status_code="READY",
                status_color="green",
                display_badge="[⚡ Chuyển sang]"
            )

            switch_hold = threading.Event()
            switch_finish = threading.Event()

            def delayed_switch(target):
                switch_hold.set()
                switch_finish.wait(timeout=2.0)
                return SwitchResult(False, "Simulated rollback error")

            mock_manager = AccountManager(
                store=self.store,
                user_status_provider=lambda: "active@example.com",
                chrome_accounts_provider=lambda: [acc_active, acc_ready]
            )
            mock_manager.switch_account = delayed_switch

            app = AppWindow(root=root, manager=mock_manager, show_dialogs=False)

            self.assertIn("ready@example.com", app.switch_buttons)
            btn = app.switch_buttons["ready@example.com"]
            self.assertEqual(str(btn.cget("state")), tk.NORMAL)

            # Start switch in background thread
            th = app._on_switch_click("ready@example.com", run_in_background=True)
            self.assertTrue(switch_hold.wait(timeout=1.0))

            # While switch is executing, button should be DISABLED and display "Đang chuyển..."
            self.assertEqual(str(btn.cget("state")), tk.DISABLED)
            self.assertEqual(str(btn.cget("text")), "Đang chuyển...")
            if app.save_btn:
                self.assertEqual(str(app.save_btn.cget("state")), tk.DISABLED)
            if app.refresh_btn:
                self.assertEqual(str(app.refresh_btn.cget("state")), tk.DISABLED)

            # Let the switch complete with error
            switch_finish.set()
            app.wait_for_switch(timeout=2.0)
            try:
                root.update()
            except Exception:
                pass

            # After failure, buttons must be restored to NORMAL state
            self.assertEqual(str(btn.cget("state")), tk.NORMAL)
            self.assertEqual(str(btn.cget("text")), "⚡ Chuyển sang")
            if app.save_btn:
                self.assertEqual(str(app.save_btn.cget("state")), tk.NORMAL)
            if app.refresh_btn:
                self.assertEqual(str(app.refresh_btn.cget("state")), tk.NORMAL)
            self.assertIn("Lỗi chuyển đổi: Simulated rollback error", app.status_bar_text)
            self.assertFalse(app.is_switching)
        finally:
            try:
                if 'app' in locals() and hasattr(app, "always_on_top_var"):
                    app.always_on_top_var = None
                root.destroy()
            except Exception:
                pass

    def test_app_window_on_save_current_session_success(self):
        saved_email = "session_user@example.com"
        token_payload = {"access_token": "captured_token_xyz"}

        def mock_capture():
            self.store.store_account(saved_email, token_payload)
            return saved_email

        mock_manager = AccountManager(
            store=self.store,
            user_status_provider=lambda: saved_email,
            chrome_accounts_provider=lambda: [saved_email]
        )
        mock_manager.auto_capture_active_session = mock_capture

        app = AppWindow(manager=mock_manager, headless=True, show_dialogs=False)

        # Execute save session
        res = app._on_save_current_session()
        self.assertEqual(res, saved_email)
        self.assertEqual(app.status_bar_text, f"Đã lưu phiên của {saved_email} vào kho JSON")
        self.assertTrue(self.store.has_account(saved_email))
        self.assertEqual(self.store.get_account_token(saved_email), token_payload)

    def test_app_window_on_save_current_session_none_and_exceptions(self):
        # 1. auto_capture_active_session returns None
        mock_manager = AccountManager(store=self.store)
        mock_manager.auto_capture_active_session = lambda: None
        app = AppWindow(manager=mock_manager, headless=True, show_dialogs=False)

        res = app._on_save_current_session()
        self.assertIsNone(res)
        self.assertEqual(app.status_bar_text, "Chưa phát hiện phiên đăng nhập Antigravity nào để lưu")

        # 2. auto_capture_active_session raises an exception
        def blow_up():
            raise OSError("Credential read failed")

        mock_manager.auto_capture_active_session = blow_up
        res_exc = app._on_save_current_session()
        self.assertIsNone(res_exc)
        self.assertEqual(app.status_bar_text, "Chưa phát hiện phiên đăng nhập Antigravity nào để lưu")

    def test_app_window_refresh_accounts_metrics_and_status_bar(self):
        # Prepopulate store with 3 accounts
        for i in range(1, 4):
            self.store.store_account(f"user{i}@example.com", {"access_token": f"token_{i}"})

        chrome_accs = [f"chrome_{i}@example.com" for i in range(1, 6)]  # 5 accounts

        mock_manager = AccountManager(
            store=self.store,
            credential_reader=lambda: None,
            user_status_provider=lambda: None,
            chrome_accounts_provider=lambda: chrome_accs
        )

        app = AppWindow(manager=mock_manager, headless=True, show_dialogs=False)

        # Initial refresh on __init__ sets format:
        # "Đồng bộ lúc <HH:MM:SS> • <N> tài khoản trong kho JSON • <M> trên Chrome"
        expected_pattern = r"^Đồng bộ lúc \d{2}:\d{2}:\d{2} • 3 tài khoản trong kho JSON • 5 trên Chrome$"
        self.assertRegex(app.status_bar_text, expected_pattern)

        # Dynamically change accounts: add 1 to store (N=4), remove 2 from Chrome (M=3)
        self.store.store_account("user4@example.com", {"access_token": "token_4"})
        chrome_accs.pop()
        chrome_accs.pop()

        app.refresh_accounts()
        expected_pattern_updated = r"^Đồng bộ lúc \d{2}:\d{2}:\d{2} • 4 tài khoản trong kho JSON • 3 trên Chrome$"
        self.assertRegex(app.status_bar_text, expected_pattern_updated)

    def test_app_window_save_and_refresh_buttons_wiring_tk(self):
        import tkinter as tk
        from unittest.mock import patch

        try:
            root = tk.Tk()
            root.withdraw()
        except Exception:
            self.skipTest("Tkinter display not available on this environment")

        try:
            captured_email = "gui_save@example.com"
            self.store.store_account("existing@example.com", {"access_token": "tok"})

            mock_manager = AccountManager(
                store=self.store,
                user_status_provider=lambda: captured_email,
                chrome_accounts_provider=lambda: ["existing@example.com", captured_email]
            )
            mock_manager.auto_capture_active_session = lambda: captured_email

            app = AppWindow(root=root, manager=mock_manager, show_dialogs=True)

            self.assertIsNotNone(app.save_btn)
            self.assertIsNotNone(app.refresh_btn)
            self.assertIsNotNone(app.status_bar)

            # 1. Click [🔄 Làm mới Chrome] via invoke()
            app.refresh_btn.invoke()
            self.assertIn("tài khoản trong kho JSON", app.status_bar.cget("text"))
            self.assertIn("trên Chrome", app.status_bar.cget("text"))

            # 2. Click [💾 Lưu phiên hiện tại] via invoke() with dialog mocked
            with patch("ui.app_window.messagebox.showinfo") as mock_info:
                app.save_btn.invoke()
                mock_info.assert_called_once()
                self.assertEqual(app.status_bar.cget("text"), f"Đã lưu phiên của {captured_email} vào kho JSON")
                self.assertEqual(app.status_bar_text, f"Đã lưu phiên của {captured_email} vào kho JSON")

            # 3. Click [💾 Lưu phiên hiện tại] when no session detected
            mock_manager.auto_capture_active_session = lambda: None
            with patch("ui.app_window.messagebox.showwarning") as mock_warn:
                app.save_btn.invoke()
                mock_warn.assert_called_once()
                self.assertEqual(app.status_bar.cget("text"), "Chưa phát hiện phiên đăng nhập Antigravity nào để lưu")
                self.assertEqual(app.status_bar_text, "Chưa phát hiện phiên đăng nhập Antigravity nào để lưu")

        finally:
            try:
                if 'app' in locals() and hasattr(app, "always_on_top_var"):
                    app.always_on_top_var = None
                root.destroy()
            except Exception:
                pass

    def test_account_store_adversarial_malformed_json_cases(self):
        """
        Adversarial test covering corrupted, truncated, and malformed JSON in accounts_store/:
        - Binary garbage payload (non-UTF8 bytes)
        - Non-JSON plaintext string
        - Truncated JSON structures
        - Missing or invalid required fields (missing token_data, null token_data, empty dict, missing email)
        - Non-dict JSON roots (arrays, scalars)
        Asserts has_account returns False, get_account_token returns None/falsy,
        and list_stored_emails skips them without crashing.
        """
        cases = {
            "binary_garbage@example.com": b"\x00\xff\xfe\xfd\x80\x90\xaa\xbb\xcc\xdd\xee\xff",
            "plain_garbage@example.com": b"THIS_IS_NOT_JSON_<<<>>>_!@#$%^&*()",
            "truncated_syntax@example.com": b'{"email": "truncated_syntax@example.com", "token_data": {"access_token": "abc',
            "truncated_key@example.com": b'{"email": "truncated_key@example.com", "token_data":',
            "missing_token_field@example.com": b'{"email": "missing_token_field@example.com", "saved_at": 1700000000}',
            "null_token_field@example.com": b'{"email": "null_token_field@example.com", "token_data": null}',
            "empty_token_dict@example.com": b'{"email": "empty_token_dict@example.com", "token_data": {}}',
            "json_array_root@example.com": b'[{"email": "json_array_root@example.com", "token_data": {"access_token": "mock"}}]',
            "json_scalar_int@example.com": b'987654321',
            "json_scalar_str@example.com": b'"just a string scalar"',
            "json_scalar_bool@example.com": b'false',
        }

        for email, raw_bytes in cases.items():
            file_path = os.path.join(self.test_store_dir, f"{email}.json")
            with open(file_path, "wb") as f:
                f.write(raw_bytes)

        # File with missing email field
        missing_email_path = os.path.join(self.test_store_dir, "missing_email_field@example.com.json")
        with open(missing_email_path, "wb") as f:
            f.write(b'{"saved_at": 1700000000, "token_data": {"access_token": "mock_tok"}}')

        # 1. Verify has_account returns False for every adversarial case
        for email in cases:
            self.assertFalse(
                self.store.has_account(email),
                f"has_account({email}) should be False for adversarial payload"
            )

        # 2. Verify get_account_token returns None for corrupt / missing / empty / non-dict tokens
        for email in cases:
            token = self.store.get_account_token(email)
            if email == "empty_token_dict@example.com":
                self.assertFalse(bool(token))
            else:
                self.assertIsNone(token, f"get_account_token({email}) should return None for malformed JSON")

        # 3. Verify list_stored_emails skips all corrupted / invalid files without crashing
        stored_emails = self.store.list_stored_emails()
        self.assertIsInstance(stored_emails, list)
        self.assertNotIn("missing_email_field@example.com", stored_emails)
        for email in [
            "binary_garbage@example.com",
            "plain_garbage@example.com",
            "truncated_syntax@example.com",
            "truncated_key@example.com",
            "json_array_root@example.com",
            "json_scalar_int@example.com",
            "json_scalar_str@example.com",
            "json_scalar_bool@example.com",
        ]:
            self.assertNotIn(email, stored_emails)

        # 4. Adding a valid account alongside the corrupted ones works cleanly
        valid_email = "valid_alongside_corrupt@example.com"
        valid_token = {"access_token": "valid_tok_123", "id_token": "valid_id_456"}
        self.assertTrue(self.store.store_account(valid_email, valid_token))
        self.assertTrue(self.store.has_account(valid_email))
        self.assertEqual(self.store.get_account_token(valid_email), valid_token)
        self.assertIn(valid_email, self.store.list_stored_emails())

        # 5. Clean removal of corrupt files works without error
        for email in cases:
            self.assertTrue(self.store.remove_account(email))
            self.assertFalse(os.path.exists(os.path.join(self.test_store_dir, f"{email}.json")))
        if os.path.exists(missing_email_path):
            os.remove(missing_email_path)

    def test_credential_manager_adversarial_boundary_edge_cases(self):
        """
        Adversarial test covering Windows Credential Manager boundary edge cases:
        - Binary garbage payload (arbitrary non-UTF8 bytes)
        - Zero-byte blob
        - Non-JSON plaintext blob
        - Non-dict JSON primitives (scalar, array, boolean)
        - Unicode edge cases (Vietnamese diacritics, emoji, escaped nulls)
        - Target names with unicode / special characters
        """
        test_target = "LegacyGeneric:target=gemini:antigravity_test_adv_boundary"
        unicode_target = "LegacyGeneric:target=gemini:antigravity_test_adv_unicode_tiếng_việt"

        try:
            # 1. Binary garbage payload (non-UTF-8 bytes)
            write_raw_credential(b"\x00\xff\xfe\xfd\x80\x90\xaa\xbb\xcc\xdd\xee\xff", target=test_target)
            self.assertIsNone(read_antigravity_token(target=test_target))

            # 2. Zero-byte blob
            write_raw_credential(b"", target=test_target)
            self.assertIsNone(read_antigravity_token(target=test_target))

            # 3. Non-JSON plaintext blob
            write_raw_credential(b"PLAIN_TEXT_PASSWORD_OR_STRING_NOT_JSON", target=test_target)
            self.assertIsNone(read_antigravity_token(target=test_target))

            # 4. Non-dict JSON primitives
            primitives = [b"true", b"false", b"null", b"12345", b'"scalar string"', b'[1, 2, 3]']
            for prim in primitives:
                write_raw_credential(prim, target=test_target)
                self.assertIsNone(
                    read_antigravity_token(target=test_target),
                    f"read_antigravity_token should return None for primitive: {prim}"
                )

            # 5. Unicode edge cases: Vietnamese diacritics, emojis, escaped nulls
            unicode_payloads = [
                {
                    "name": "Đặng Thị Mỹ Duyên",
                    "city": "Thành phố Hồ Chí Minh",
                    "org": "Hộ kinh doanh An Viên",
                    "email": "duyen.dang@example.com",
                    "notes": "Kiểm thử tiếng Việt có dấu đầy đủ: ă, â, đ, ê, ô, ơ, ư"
                },
                {
                    "icon": "🚀🔑💎🔥✨🛡️⚡",
                    "badge": "[ĐANG DÙNG 🟢]",
                    "status": "ACTIVE_✅"
                },
                {
                    "escaped_null": "\\u0000",
                    "special_chars": "\\n\\r\\t\\\"\\\\"
                }
            ]

            for payload in unicode_payloads:
                self.assertTrue(write_antigravity_token(payload, target=test_target))
                read_back = read_antigravity_token(target=test_target)
                self.assertIsNotNone(read_back)
                self.assertEqual(read_back, payload)

            # 6. Target names with unicode characters
            vn_payload = {"email": "unicode_target@example.com", "token": "test_tok"}
            self.assertTrue(write_antigravity_token(vn_payload, target=unicode_target))
            vn_read = read_antigravity_token(target=unicode_target)
            self.assertIsNotNone(vn_read)
            self.assertEqual(vn_read, vn_payload)
            self.assertTrue(delete_antigravity_token(target=unicode_target))

        finally:
            delete_antigravity_token(test_target)
            delete_antigravity_token(unicode_target)

    def test_api_connector_adversarial_network_resilience(self):
        """
        Adversarial test covering api_connector.py network resilience:
        - Abrupt TCP connection resets (RST packet / immediate close)
        - Malformed HTTP responses (truncated JSON, non-dict JSON, HTML proxy error)
        - Missing email field in userStatus
        Asserts get_user_status, auth_logout, and trigger_reauth handle all cases safely.
        """
        server = MockHttpServerHelper()
        server.start()
        try:
            info = LanguageServerInfo(pid=1234, csrf_token="test-adv-uuid-001", port=server.port)

            # 1. Abrupt TCP connection reset
            server.reset_connection = True
            self.assertIsNone(get_user_status(info, timeout=0.5))
            self.assertFalse(auth_logout(info, timeout=0.5))
            self.assertFalse(trigger_reauth(info, timeout=0.5))
            server.reset_connection = False

            # 2. Malformed HTTP response: Truncated JSON
            server.response_status = 200
            server.response_body = b'{"userStatus": {"email": "broken@example.com", "name":'
            self.assertIsNone(get_user_status(info))

            # 3. Malformed HTTP response: Non-dict JSON primitives
            for non_dict in [b'"just a string"', b'12345', b'[1, 2, 3]', b'true']:
                server.response_body = non_dict
                self.assertIsNone(get_user_status(info))

            # 4. Malformed HTTP response: HTML error page with 200 OK
            server.response_body = b"<!DOCTYPE html><html><body>502 Bad Gateway</body></html>"
            self.assertIsNone(get_user_status(info))

            # 5. Missing email in userStatus (graceful partial parse)
            server.response_body = json.dumps({
                "userStatus": {
                    "name": "Anonymous User",
                    "planStatus": "PRO"
                }
            }).encode("utf-8")
            status = get_user_status(info)
            self.assertIsNotNone(status)
            self.assertIsNone(status.get("email"))
            self.assertEqual(status.get("name"), "Anonymous User")
            self.assertEqual(status.get("status"), "PRO")

        finally:
            server.stop()

    def test_api_connector_adversarial_cli_and_port_discovery(self):
        """
        Adversarial test covering command line parsing and port discovery resilience:
        - Missing CSRF token in command line (flag with no argument, whitespace, hyphenated flag)
        - Port scanning failures (garbage netstat output, unparseable lines, timeouts)
        - Fallback port selection when HTTP probing fails
        """
        # 1. Missing CSRF token variations
        malformed_cmdlines = [
            "language_server.exe --csrf_token",
            "language_server.exe --csrf_token ",
            "language_server.exe --csrf_token=",
            "language_server.exe --csrf_token=\"\"",
            "language_server.exe --csrf_token=''",
            "language_server.exe --csrf-token 12345",
            "language_server.exe --token 12345",
            "language_server.exe --standalone --https_server_port 0",
            "language_server.exe",
        ]
        for cmd in malformed_cmdlines:
            self.assertIsNone(extract_csrf_token(cmd))

        # 2. find_language_server_process with malformed command lines returns None
        mock_bad_procs = [
            {"ProcessId": 1234, "CommandLine": "language_server.exe --csrf_token"},
            {"ProcessId": 5678, "CommandLine": "language_server.exe --csrf-token=abc"},
        ]
        self.assertIsNone(find_language_server_process(
            process_provider=lambda: mock_bad_procs,
            port_provider=lambda pid: 50000
        ))

        # 3. discover_listening_ports with corrupted / garbage netstat output
        from unittest.mock import patch, MagicMock
        mock_run = MagicMock()
        mock_run.return_value.stdout = (
            "Garbage header text\n"
            "TCP    127.0.0.1:invalid    0.0.0.0:0    LISTENING    9999\n"
            "UDP    127.0.0.1:54321      *:*                       9999\n"
            "TCP    broken_addr          0.0.0.0:0    LISTENING    9999\n"
            "TCP    127.0.0.1:60001      0.0.0.0:0    LISTENING    9999\n"
        )
        with patch("subprocess.run", mock_run):
            ports = discover_listening_ports(9999)
            self.assertEqual(ports, [60001])

        # 4. discover_listening_port fallback when HTTP probe fails on all ports
        with patch("core.api_connector.probe_http_port", return_value=False):
            chosen = discover_listening_port(9999, ports_provider=lambda pid: [59226, 59227])
            self.assertEqual(chosen, 59226)

    def test_session_reloader_adversarial_hwnd_and_window_resilience(self):
        """
        Adversarial test covering session_reloader.py window handle resilience:
        - Calling reload_antigravity_window with invalid / non-existent HWNDs
        - Calling reload_antigravity_window when no windows are found
        - Exception resilience during window reload and reauth trigger coordination
        """
        from unittest.mock import patch

        # 1. Non-existent and invalid HWNDs (e.g. 0, 999999999, -1)
        invalid_hwnds = [0, 999999999, -1, 12345678]
        with patch("core.session_reloader.find_antigravity_windows", return_value=invalid_hwnds):
            # Must execute without raising any exception and return True
            res = reload_antigravity_window()
            self.assertTrue(res)

        # 2. Empty windows list returns False
        with patch("core.session_reloader.find_antigravity_windows", return_value=[]):
            self.assertFalse(reload_antigravity_window())

        # 3. reload_antigravity_session when window_reloader raises unexpected exceptions
        def throwing_window_reloader():
            raise OSError("Window station disconnected")

        info = LanguageServerInfo(pid=9999, csrf_token="token", port=50000)
        # When reauth succeeds, window_reloader exception doesn't prevent success if reload_window=False
        res_ok = reload_antigravity_session(
            info=info,
            reload_window=False,
            window_reloader=throwing_window_reloader,
            reauth_trigger=lambda inf: True
        )
        self.assertTrue(res_ok)

        # When reauth fails and window_reloader raises, returns False without crashing
        res_fail = reload_antigravity_session(
            info=info,
            reload_window=False,
            window_reloader=throwing_window_reloader,
            reauth_trigger=lambda inf: False
        )
        self.assertFalse(res_fail)

    def test_pre_patch_tos_acceptance_sets_false_literal(self):
        """Verifies pre_patch_tos_acceptance atomically writes lastLoginIsGcpTos as string literal 'false'."""
        storage_file = os.path.join(self.test_store_dir, "test_app_storage.json")
        res = pre_patch_tos_acceptance("pilot.user@gmail.com", storage_path=storage_file)
        self.assertTrue(res)
        with open(storage_file, "r", encoding="utf-8") as f:
            data = json.load(f)
        self.assertEqual(data.get("jetski.onboarding.lastLoginUsername"), "pilot.user@gmail.com")
        self.assertEqual(data.get("jetski.onboarding.lastLoginIsGcpTos"), "false")
        self.assertIsInstance(data.get("jetski.onboarding.lastLoginIsGcpTos"), str)

    def test_switch_account_adversarial_sequence_and_rollback_safeguards(self):
        """
        Adversarial test covering full 4-step sequence verification and mid-step rollback safeguards:
        - Exact chronological order assertion (Step 1 -> Step 2 -> Step 3 -> Step 4)
        - Mid-step failure at Step 1 (target missing from store): aborts before Step 2, active session preserved
        - Mid-step failure at Step 1 (language server error): falls back safely
        - Mid-step failure at Step 2 (logout fails/throws): proceeds safely to Step 3 and 4
        - Mid-step failure at Step 3 (credential write fails): Step 4 aborted, active session safely preserved in store
        - Mid-step failure at Step 4 (reloader throws): returns SwitchResult safely
        """
        active_email = "active_current@example.com"
        target_email = "target_new@example.com"
        active_token = {"access_token": "act_tok", "id_token": "act_id"}
        target_token = {"access_token": "tar_tok", "id_token": "tar_id"}

        self.store.store_account(active_email, active_token)
        self.store.store_account(target_email, target_token)

        mock_info = LanguageServerInfo(pid=8888, csrf_token="test-csrf", port=50000)

        # 1. Exact chronological order assertion
        chronological_log = []

        def tracked_finder():
            chronological_log.append("step1_finder")
            return mock_info

        def tracked_getter(inf):
            chronological_log.append("step1_getter")
            return {"email": active_email}

        def tracked_logout(inf):
            chronological_log.append("step2_logout")
            return True

        def tracked_writer(tok):
            chronological_log.append("step3_writer")
            return True

        def tracked_reloader(info=None):
            chronological_log.append("step4_reloader")
            return True

        manager = AccountManager(
            store=self.store,
            process_finder=tracked_finder,
            status_getter=tracked_getter,
            logout_fn=tracked_logout,
            token_writer=tracked_writer,
            reloader_fn=tracked_reloader,
        )

        # 1. Exact chronological order assertion with force_logout=True
        res = manager.switch_account(target_email, force_logout=True)
        self.assertTrue(res)
        self.assertEqual(
            chronological_log,
            ["step1_finder", "step1_getter", "step2_logout", "step3_writer", "step4_reloader"]
        )

        # 1b. Smooth hot-swap by default: auth_logout is bypassed
        chronological_log.clear()
        res_smooth = manager.switch_account(target_email)
        self.assertTrue(res_smooth)
        self.assertEqual(
            chronological_log,
            ["step1_finder", "step1_getter", "step3_writer", "step4_reloader"]
        )

        # 2. Mid-step failure at Step 1 (target missing from store)
        # CRITICAL SAFEGUARD: Neither Step 2 logout, Step 3 write, nor Step 4 reload must be called!
        chronological_log.clear()
        res_missing = manager.switch_account("absent_target@example.com")
        self.assertFalse(res_missing)
        self.assertIn("Chưa có file JSON", res_missing.message)
        self.assertNotIn("step2_logout", chronological_log)
        self.assertNotIn("step3_writer", chronological_log)
        self.assertNotIn("step4_reloader", chronological_log)

        # 3. Mid-step failure at Step 1 (language server throws exception)
        def broken_finder():
            raise RuntimeError("Language server process inaccessible")

        chronological_log.clear()
        manager_broken_ls = AccountManager(
            store=self.store,
            process_finder=broken_finder,
            logout_fn=tracked_logout,
            token_writer=tracked_writer,
            reloader_fn=tracked_reloader,
        )
        res_ls_fail = manager_broken_ls.switch_account(target_email)
        self.assertTrue(res_ls_fail)
        self.assertIn("step3_writer", chronological_log)
        self.assertIn("step4_reloader", chronological_log)

        # 4. Mid-step failure at Step 2 (logout raises exception when force_logout=True)
        def broken_logout(inf):
            raise ConnectionResetError("TCP connection reset during logout")

        chronological_log.clear()
        manager_broken_logout = AccountManager(
            store=self.store,
            process_finder=tracked_finder,
            status_getter=tracked_getter,
            logout_fn=broken_logout,
            token_writer=tracked_writer,
            reloader_fn=tracked_reloader,
        )
        res_log_fail = manager_broken_logout.switch_account(target_email, force_logout=True)
        self.assertTrue(res_log_fail)
        self.assertIn("step3_writer", chronological_log)
        self.assertIn("step4_reloader", chronological_log)

        # 5. Mid-step failure at Step 3 (writer fails)
        # CRITICAL SAFEGUARD: Step 4 reload MUST NOT be called!
        def failing_writer(tok):
            chronological_log.append("step3_writer_fail")
            return False

        chronological_log.clear()
        manager_writer_fail = AccountManager(
            store=self.store,
            process_finder=tracked_finder,
            status_getter=tracked_getter,
            logout_fn=tracked_logout,
            token_writer=failing_writer,
            reloader_fn=tracked_reloader,
        )
        res_wfail = manager_writer_fail.switch_account(target_email, force_logout=True)
        self.assertFalse(res_wfail)
        self.assertIn("Không thể ghi Token", res_wfail.message)
        self.assertIn("step3_writer_fail", chronological_log)
        self.assertNotIn("step4_reloader", chronological_log)
        # Verify active session is preserved in store
        self.assertTrue(self.store.has_account(active_email))

        # 6. Mid-step failure at Step 4 (reloader raises exception)
        def broken_reloader(info=None):
            chronological_log.append("step4_reloader_fail")
            raise RuntimeError("Window reload keystroke failed")

        chronological_log.clear()
        manager_reloader_fail = AccountManager(
            store=self.store,
            process_finder=tracked_finder,
            status_getter=tracked_getter,
            logout_fn=tracked_logout,
            token_writer=tracked_writer,
            reloader_fn=broken_reloader,
        )
        res_rfail = manager_reloader_fail.switch_account(target_email, force_logout=True)
        self.assertTrue(res_rfail)
        self.assertIn("step4_reloader_fail", chronological_log)

    def test_load_trajectory_mock_http_server(self):
        server = MockHttpServerHelper()
        server.start()
        try:
            info = LanguageServerInfo(pid=4321, csrf_token="csrf-load-traj", port=server.port)
            server.response_status = 200
            server.response_body = b"{}"

            self.assertTrue(load_trajectory(info, "mock-cascade-uuid-1"))
            self.assertEqual(server.last_path, "/exa.language_server_pb.LanguageServerService/LoadTrajectory")
            self.assertEqual(server.last_headers.get("x-csrf-token"), info.csrf_token)
            self.assertEqual(server.last_headers.get("x-codeium-csrf-token"), info.csrf_token)

            server.response_status = 500
            self.assertFalse(load_trajectory(info, "mock-cascade-uuid-1"))
        finally:
            server.stop()

    def test_smart_focus_conversation_mock_http_server(self):
        server = MockHttpServerHelper()
        server.start()
        try:
            info = LanguageServerInfo(pid=4321, csrf_token="csrf-smart-focus", port=server.port)
            server.response_status = 200
            server.response_body = b"{}"

            self.assertTrue(smart_focus_conversation(info, "mock-cascade-uuid-2"))
            self.assertEqual(server.last_path, "/exa.language_server_pb.LanguageServerService/SmartFocusConversation")
            self.assertEqual(server.last_headers.get("x-csrf-token"), info.csrf_token)

            server.response_status = 404
            self.assertFalse(smart_focus_conversation(info, "mock-cascade-uuid-2"))
        finally:
            server.stop()

    def test_send_user_cascade_message_mock_http_server(self):
        server = MockHttpServerHelper()
        server.start()
        try:
            info = LanguageServerInfo(pid=4321, csrf_token="csrf-send-cascade", port=server.port)
            server.response_status = 200
            server.response_body = b"{}"

            self.assertTrue(send_user_cascade_message(info, "mock-cascade-uuid-3", prompt_text="custom prompt test"))
            self.assertEqual(server.last_path, "/exa.language_server_pb.LanguageServerService/SendUserCascadeMessage")
            self.assertEqual(server.last_headers.get("x-csrf-token"), info.csrf_token)
            sent_data = json.loads(server.last_body.decode("utf-8"))
            self.assertIn("cascadeConfig", sent_data)
            self.assertIn("plannerConfig", sent_data["cascadeConfig"])
            self.assertIn("planModel", sent_data["cascadeConfig"]["plannerConfig"])
            self.assertIn("requestedModel", sent_data["cascadeConfig"]["plannerConfig"])
            self.assertEqual(
                sent_data["cascadeConfig"]["plannerConfig"]["planModel"],
                sent_data["cascadeConfig"]["plannerConfig"]["requestedModel"]["model"]
            )

            # Test explicit model override
            self.assertTrue(send_user_cascade_message(
                info,
                "mock-cascade-uuid-3",
                prompt_text="explicit model",
                model_name="MODEL_EXPLICIT_TEST"
            ))
            sent_data_2 = json.loads(server.last_body.decode("utf-8"))
            self.assertEqual(
                sent_data_2["cascadeConfig"]["plannerConfig"]["planModel"],
                "MODEL_EXPLICIT_TEST"
            )

            server.response_status = 503
            self.assertFalse(send_user_cascade_message(info, "mock-cascade-uuid-3"))
        finally:
            server.stop()

    def test_discover_active_agent_model_pbtxt_and_fallback(self):
        temp_pbtxt = os.path.join(self.test_store_dir, "test_agent_state.pbtxt")
        with open(temp_pbtxt, "w", encoding="utf-8") as f:
            f.write("last_selected_agent_model: MODEL_PLACEHOLDER_M421\n")

        # Discovers from pbtxt
        self.assertEqual(discover_active_agent_model(pbtxt_path=temp_pbtxt), "MODEL_PLACEHOLDER_M421")

        # Fallback when file doesn't exist
        nonexistent = os.path.join(self.test_store_dir, "nonexistent.pbtxt")
        self.assertEqual(discover_active_agent_model(info=None, pbtxt_path=nonexistent), "MODEL_PLACEHOLDER_M318")

    def test_discover_most_recent_cascade_id_isolated_env(self):
        test_brain = os.path.join(self.test_store_dir, "test_brain_isolated")
        os.makedirs(os.path.join(test_brain, "casc-1"), exist_ok=True)
        os.makedirs(os.path.join(test_brain, "casc-2", ".system_generated", "logs"), exist_ok=True)

        t1 = os.path.join(test_brain, "casc-1", "transcript.jsonl")
        t2 = os.path.join(test_brain, "casc-2", ".system_generated", "logs", "transcript.jsonl")

        with open(t1, "w", encoding="utf-8") as f:
            f.write("{}\n")
        with open(t2, "w", encoding="utf-8") as f:
            f.write("{}\n")

        os.utime(t1, (5000, 5000))
        os.utime(t2, (6000, 6000))

        # Default fallback to most recent mtime
        self.assertEqual(discover_most_recent_cascade_id(brain_dir=test_brain, storage_path="absent.json"), "casc-2")

        # Storage cross-check prioritizes conversationPanes
        storage_path = os.path.join(self.test_store_dir, "test_storage.json")
        with open(storage_path, "w", encoding="utf-8") as f:
            json.dump({
                "aux-pane-v2-session": json.dumps({
                    "conversationPanes": {
                        "casc-1": {"tabs": []}
                    }
                })
            }, f)

        self.assertEqual(discover_most_recent_cascade_id(brain_dir=test_brain, storage_path=storage_path), "casc-1")

    def test_switch_account_native_rpc_continuation_flow(self):
        target_email = "rpc_flow@gmail.com"
        self.store.store_account(target_email, {"access_token": "flow_at"})

        mock_info = LanguageServerInfo(pid=9999, csrf_token="csrf-flow", port=59001)
        calls = []

        def mock_focuser(info, cid):
            calls.append(("focus", cid))
            return True

        def mock_loader(info, cid):
            calls.append(("load", cid))
            return True

        def mock_sender(info, cid, prompt_text="continue"):
            calls.append(("send", cid, prompt_text))
            return True

        manager = AccountManager(
            store=self.store,
            process_finder=lambda: mock_info,
            status_getter=lambda info: {"email": "old_flow@gmail.com"},
            token_writer=lambda tok: True,
            tos_pre_patcher=lambda em: True,
            language_server_recycler=lambda: True,
            cascade_discoverer=lambda: "discovered-flow-cascade",
            smart_focuser=mock_focuser,
            trajectory_loader=mock_loader,
            cascade_sender=mock_sender,
            auto_continue=True,
            dom_settle_delay=0.0,
        )

        res = manager.switch_account(target_email)
        self.assertTrue(res.success)
        self.assertEqual(calls, [
            ("focus", "discovered-flow-cascade"),
            ("load", "discovered-flow-cascade"),
            ("send", "discovered-flow-cascade", DEFAULT_CONTINUATION_PROMPT),
        ])

    def test_capture_active_threads_snapshot_schema_and_timestamp(self):
        """Verifies capture_active_threads_snapshot generates valid timestamped JSON matching exact schema."""
        import re
        temp_storage = os.path.join(self.test_store_dir, "test_app_storage.json")
        temp_history = os.path.join(self.test_store_dir, "history_snapshot")

        mock_storage = {
            "aux-pane-v2-session.conversationPanes": [
                {
                    "paneId": "main_cascade",
                    "tabs": [
                        {
                            "tabId": "tab-1",
                            "metadata": {
                                "cascadeId": "c-alpha-1",
                                "workingDirectory": "C:/Projects/ProjectAlpha"
                            }
                        }
                    ]
                },
                {
                    "paneId": "sub_cascade",
                    "tabs": [
                        {
                            "tabId": "tab-2",
                            "metadata": {
                                "cascadeId": "c-beta-2",
                                "workingDirectory": "C:/Projects/ProjectBeta"
                            }
                        }
                    ]
                }
            ]
        }
        with open(temp_storage, "w", encoding="utf-8") as f:
            json.dump(mock_storage, f)

        # Create thread subdirectories in temp brain dir
        os.makedirs(os.path.join(self.test_store_dir, "c-alpha-1"), exist_ok=True)
        os.makedirs(os.path.join(self.test_store_dir, "c-beta-2"), exist_ok=True)

        snapshot_file, active_threads = capture_active_threads_snapshot(
            storage_path=temp_storage,
            brain_dir=self.test_store_dir,
            output_dir=temp_history,
        )

        self.assertIsNotNone(snapshot_file)
        self.assertTrue(os.path.exists(snapshot_file))
        basename = os.path.basename(snapshot_file)
        self.assertTrue(bool(re.match(r"^\d{4}-\d{2}-\d{2}-\d{2}-\d{2}-\d{2}_active_threads\.json$", basename)))

        with open(snapshot_file, "r", encoding="utf-8") as f:
            data = json.load(f)

        self.assertIn("timestamp", data)
        self.assertIn("active_threads", data)
        self.assertEqual(len(data["active_threads"]), 2)
        self.assertEqual(data["active_threads"][0]["thread_id"], "c-alpha-1")
        self.assertTrue(data["active_threads"][0]["is_main_pane"])
        self.assertEqual(data["active_threads"][0]["project"], "C:/Projects/ProjectAlpha")
        self.assertEqual(data["active_threads"][1]["thread_id"], "c-beta-2")
        self.assertFalse(data["active_threads"][1]["is_main_pane"])
        self.assertEqual(data["active_threads"][1]["project"], "C:/Projects/ProjectBeta")

    def test_switch_account_multi_project_pure_native_rpc_continuation(self):
        """Verifies multi-project continuation loop sends RPCs to all threads with fresh LanguageServerInfo and zero keyboard events."""
        target_email = "multi_flow@gmail.com"
        self.store.store_account(target_email, {"access_token": "multi_at"})

        fresh_info = LanguageServerInfo(pid=7777, csrf_token="csrf-multi", port=57777)
        rpc_calls = []

        threads_snapshot = [
            {"thread_id": "c-proj1", "project": "C:/Work/Proj1", "is_main_pane": True},
            {"thread_id": "c-proj2", "project": "C:/Work/Proj2", "is_main_pane": False},
        ]

        def mock_focuser(info, cid):
            rpc_calls.append(("focus", info.pid, cid))
            return True

        def mock_loader(info, cid):
            rpc_calls.append(("load", info.pid, cid))
            return True

        def mock_sender(info, cid, prompt_text="continue"):
            rpc_calls.append(("send", info.pid, cid, prompt_text))
            return True

        keyboard_events = []
        def mock_injector(prompt_text="continue", sleep_fn=None):
            keyboard_events.append(prompt_text)
            return True

        sleep_calls = []
        def mock_sleep_fn(duration):
            sleep_calls.append(duration)

        manager = AccountManager(
            store=self.store,
            process_finder=lambda: fresh_info,
            status_getter=lambda info: {"email": "old_multi@gmail.com"},
            token_writer=lambda tok: True,
            tos_pre_patcher=lambda em: True,
            language_server_recycler=lambda: True,
            threads_snapshot_capturer=lambda: ("/path", threads_snapshot),
            smart_focuser=mock_focuser,
            trajectory_loader=mock_loader,
            cascade_sender=mock_sender,
            prompt_injector=mock_injector,
            sleep_fn=mock_sleep_fn,
            auto_continue=True,
            dom_settle_delay=0.0,
        )

        res = manager.switch_account(target_email, prompt_text="resume task")
        self.assertTrue(res.success)
        self.assertEqual(len(rpc_calls), 6)
        self.assertEqual(rpc_calls[0], ("focus", 7777, "c-proj1"))
        self.assertEqual(rpc_calls[1], ("load", 7777, "c-proj1"))
        self.assertEqual(rpc_calls[2], ("send", 7777, "c-proj1", "resume task"))
        self.assertEqual(rpc_calls[3], ("focus", 7777, "c-proj2"))
        self.assertEqual(rpc_calls[4], ("load", 7777, "c-proj2"))
        self.assertEqual(rpc_calls[5], ("send", 7777, "c-proj2", "resume task"))
        self.assertEqual(keyboard_events, [])

        # Assert 3.0s pacing delay called between threads
        pacing_delays = [d for d in sleep_calls if d == 3.0]
        self.assertEqual(len(pacing_delays), 1)

    def test_switch_account_default_dom_settle_delay_60s(self):
        """Verifies switch_account dom_settle_delay defaults to 60.0s when not specified."""
        target_email = "settle60_all@gmail.com"
        self.store.store_account(target_email, {"access_token": "at_60"})
        sleep_durations = []

        fresh_info = LanguageServerInfo(pid=5555, csrf_token="tok60", port=55551)
        manager = AccountManager(
            store=self.store,
            process_finder=lambda: fresh_info,
            token_writer=lambda tok: True,
            tos_pre_patcher=lambda em: True,
            language_server_recycler=lambda **kw: None,
            threads_snapshot_capturer=lambda: ("/path", [{"thread_id": "c60", "project": "", "is_main_pane": True}]),
            smart_focuser=lambda info, cid: True,
            trajectory_loader=lambda info, cid: True,
            cascade_sender=lambda info, cid, prompt_text="": True,
            sleep_fn=lambda d: sleep_durations.append(d),
            auto_continue=True,
            # No dom_settle_delay passed -> MUST default to 60.0s
        )

        res = manager.switch_account(target_email)
        self.assertTrue(res.success)
        self.assertIn(60.0, sleep_durations)

    def test_switch_account_default_continuation_prompt(self):
        """Verifies default continuation prompt matches the English directive."""
        self.assertEqual(
            DEFAULT_CONTINUATION_PROMPT,
            "Continue if you still have unfinished work in progress. If all tasks are already completed, reply with: Currently there is no work to do."
        )
        manager = AccountManager(store=self.store)
        self.assertEqual(manager.prompt_text, DEFAULT_CONTINUATION_PROMPT)

    def test_capture_active_threads_snapshot_ignores_stale_layout_keys(self):
        """Verifies capture_active_threads_snapshot only captures active conversationPanes and ignores stale layout keys."""
        temp_storage = os.path.join(self.test_store_dir, "stale_test_app_storage.json")
        temp_history = os.path.join(self.test_store_dir, "history_stale")

        active_thread = "active-pane-thread-99"
        stale_thread = "stale-layout-thread-42"

        mock_storage = {
            "aux-pane-v2-session": json.dumps({
                "conversationPanes": {
                    "main": {
                        "tabs": [
                            {
                                "cascadeId": active_thread,
                                "metadata": {"workingDirectory": "C:/Projects/ActiveApp"}
                            }
                        ]
                    }
                }
            }),
            f"antigravity-multi-conversation-layout-v3-{stale_thread}": json.dumps({"activeThreadId": stale_thread}),
            "workbench.panel.cascade": {"activeThreadId": "stale-workbench"}
        }
        with open(temp_storage, "w", encoding="utf-8") as f:
            json.dump(mock_storage, f)

        # Create thread directory in brain_dir for active_thread
        os.makedirs(os.path.join(self.test_store_dir, active_thread), exist_ok=True)

        snapshot_file, active_threads = capture_active_threads_snapshot(
            storage_path=temp_storage,
            brain_dir=self.test_store_dir,
            output_dir=temp_history,
        )

        self.assertEqual(len(active_threads), 1)
        self.assertEqual(active_threads[0]["thread_id"], active_thread)
        self.assertEqual(active_threads[0]["project"], "C:/Projects/ActiveApp")
        self.assertTrue(active_threads[0]["is_main_pane"])
        self.assertNotIn(stale_thread, [t["thread_id"] for t in active_threads])

    def test_capture_active_threads_snapshot_filters_fileview_and_missing_brain_dirs(self):
        """
        Verifies capture_active_threads_snapshot:
        1. Filters out non-conversation tabs (content.type == 'fileView' or id starting with 'file__')
        2. Excludes thread IDs that do not have a corresponding directory in brain/
        """
        temp_storage = os.path.join(self.test_store_dir, "filter_app_storage_all.json")
        temp_brain = os.path.join(self.test_store_dir, "filter_brain_all")
        temp_history_dir = os.path.join(self.test_store_dir, "filter_history_all")
        os.makedirs(temp_brain, exist_ok=True)

        valid_thread = "valid-thread-with-brain-dir-all"
        missing_brain_thread = "missing-brain-thread-all"
        fileview_thread = "fileview-thread-all"

        # Create brain dir only for valid_thread
        os.makedirs(os.path.join(temp_brain, valid_thread), exist_ok=True)

        mock_storage = {
            "aux-pane-v2-session.conversationPanes": [
                {
                    "paneId": "main_cascade_pane",
                    "tabs": [
                        {
                            "id": "tab-valid",
                            "metadata": {
                                "cascadeId": valid_thread,
                                "workingDirectory": "C:/Projects/Valid"
                            }
                        },
                        {
                            "id": "tab-missing-brain",
                            "metadata": {
                                "cascadeId": missing_brain_thread,
                                "workingDirectory": "C:/Projects/Missing"
                            }
                        },
                        {
                            "id": "tab-fileview-content",
                            "content": {"type": "fileView"},
                            "metadata": {
                                "cascadeId": fileview_thread,
                                "workingDirectory": "C:/Projects/File1"
                            }
                        },
                        {
                            "id": "file__readme_md",
                            "metadata": {
                                "cascadeId": "file-thread-id",
                                "workingDirectory": "C:/Projects/File2"
                            }
                        }
                    ]
                }
            ]
        }
        with open(temp_storage, "w", encoding="utf-8") as f:
            json.dump(mock_storage, f)

        snapshot_file, active_threads = capture_active_threads_snapshot(
            storage_path=temp_storage,
            brain_dir=temp_brain,
            output_dir=temp_history_dir,
        )

        # Only valid_thread should be captured
        self.assertEqual(len(active_threads), 1)
        self.assertEqual(active_threads[0]["thread_id"], valid_thread)
        self.assertEqual(active_threads[0]["project"], "C:/Projects/Valid")

        thread_ids = [t["thread_id"] for t in active_threads]
        self.assertNotIn(missing_brain_thread, thread_ids)
        self.assertNotIn(fileview_thread, thread_ids)
        self.assertNotIn("file-thread-id", thread_ids)



class MockHttpServerHelper:
    """Helper to spin up an in-process loopback HTTP server on an ephemeral port for testing."""

    def __init__(self):
        self.server = None
        self.thread = None
        self.port = 0
        self.last_headers = {}
        self.last_path = ""
        self.last_body = b""
        self.response_status = 200
        self.response_body = b"{}"
        self.response_headers = {"Content-Type": "application/json"}
        self.path_status_map = {}
        self.delay = 0.0
        self.reset_connection = False

    def start(self):
        helper = self

        class StoppableHTTPServer(HTTPServer):
            allow_reuse_address = True
            daemon_threads = True

            def handle_error(self, request, client_address):
                # Suppress benign client disconnects during teardown/rapid requests
                pass

        class Handler(BaseHTTPRequestHandler):
            protocol_version = "HTTP/1.1"

            def log_message(self, format, *args):
                pass  # suppress stderr log spam

            def do_POST(self):
                if helper.reset_connection:
                    import socket, struct
                    try:
                        self.connection.setsockopt(socket.SOL_SOCKET, socket.SO_LINGER, struct.pack('ii', 1, 0))
                    except Exception:
                        pass
                    try:
                        self.connection.close()
                    except Exception:
                        pass
                    self.close_connection = True
                    return

                if helper.delay > 0:
                    time.sleep(helper.delay)
                helper.last_path = self.path
                helper.last_headers = {k.lower(): v for k, v in self.headers.items()}

                # Safely drain incoming request body to avoid Windows TCP RST on socket close
                try:
                    content_length = int(self.headers.get("Content-Length", 0))
                except (ValueError, TypeError):
                    content_length = 0
                if content_length > 0:
                    try:
                        helper.last_body = self.rfile.read(content_length)
                    except (ConnectionResetError, ConnectionAbortedError, BrokenPipeError, OSError):
                        helper.last_body = b""
                else:
                    helper.last_body = b""

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
                    self.wfile.flush()
                except (ConnectionResetError, ConnectionAbortedError, BrokenPipeError, OSError):
                    pass

            def do_GET(self):
                self.do_POST()

            def finish(self):
                try:
                    super().finish()
                except (ConnectionResetError, ConnectionAbortedError, BrokenPipeError, OSError):
                    pass

        self.server = StoppableHTTPServer(("127.0.0.1", 0), Handler)
        self.port = self.server.server_port
        self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)
        self.thread.start()
        time.sleep(0.05)

    def stop(self):
        if self.server:
            try:
                self.server.shutdown()
            except Exception:
                pass
            try:
                self.server.server_close()
            except Exception:
                pass
            if self.thread and self.thread.is_alive():
                self.thread.join(timeout=1.5)
            time.sleep(0.02)


if __name__ == "__main__":
    unittest.main()


