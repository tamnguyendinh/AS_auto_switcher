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


if __name__ == "__main__":
    unittest.main()

