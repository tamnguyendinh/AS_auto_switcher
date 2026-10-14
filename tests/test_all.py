"""
Unit tests for Antigravity Multi-Account Switcher (JSON Storage Architecture).
"""

import unittest
import base64
import os
import shutil
from core.jwt_helper import decode_jwt_payload, extract_email_from_token_data
from core.account_store import AccountStore, sanitize_filename
from scanner.chrome_scanner import is_blacklisted, get_chrome_signed_in_accounts
from core.credential_manager import read_antigravity_token
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

    def test_live_credential_read(self):
        token_data = read_antigravity_token()
        self.assertIsNotNone(token_data)
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


if __name__ == "__main__":
    unittest.main()
