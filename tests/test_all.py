"""
Unit tests for Antigravity Multi-Account Switcher.
Verifies JWT decoding, DPAPI encryption, Credential Manager, Chrome Scanner, and State Resolver.
"""

import unittest
import base64
import json
from core.jwt_helper import decode_jwt_payload, extract_email_from_token_data
from core.vault import dpapi_encrypt, dpapi_decrypt, TokenVault
from scanner.chrome_scanner import is_blacklisted, get_chrome_signed_in_accounts
from core.credential_manager import read_antigravity_token
from core.state_resolver import AccountManager


class TestAntigravitySwitcher(unittest.TestCase):

    def test_jwt_payload_decode(self):
        # Create a mock JWT
        header = base64.b64encode(b'{"alg":"RS256"}').decode("utf-8").rstrip("=")
        payload = base64.b64encode(b'{"email":"testuser@gmail.com","sub":"12345"}').decode("utf-8").rstrip("=")
        sig = "invalidsignature"
        jwt_mock = f"{header}.{payload}.{sig}"

        decoded = decode_jwt_payload(jwt_mock)
        self.assertIsNotNone(decoded)
        self.assertEqual(decoded.get("email"), "testuser@gmail.com")

        email = extract_email_from_token_data({"id_token": jwt_mock})
        self.assertEqual(email, "testuser@gmail.com")

    def test_blacklist_enforcement(self):
        # Verify forbidden keywords are strictly filtered
        self.assertTrue(is_blacklisted("hokinhdoanh.anvien.sales@gmail.com"))
        self.assertTrue(is_blacklisted("anvien@gmail.com"))
        self.assertTrue(is_blacklisted("TEST_ANVIEN_TEST@gmail.com"))
        self.assertFalse(is_blacklisted("hiendao1760@gmail.com"))
        self.assertFalse(is_blacklisted("mydeveloper@gmail.com"))

    def test_dpapi_encryption_roundtrip(self):
        original = b"Secret Antigravity OAuth Token 123456"
        encrypted = dpapi_encrypt(original)
        self.assertIsNotNone(encrypted)
        self.assertNotEqual(original, encrypted)

        decrypted = dpapi_decrypt(encrypted)
        self.assertEqual(original, decrypted)

    def test_token_vault_memory(self):
        vault = TokenVault()
        test_email = "sample_test_acc@gmail.com"
        mock_token = {"access_token": "ya29.mock", "id_token": "mock.jwt.sig"}

        self.assertTrue(vault.store_account(test_email, mock_token))
        self.assertTrue(vault.has_account(test_email))

        retrieved = vault.get_account_token(test_email)
        self.assertEqual(retrieved, mock_token)

        vault.remove_account(test_email)
        self.assertFalse(vault.has_account(test_email))

    def test_live_credential_read(self):
        token_data = read_antigravity_token()
        self.assertIsNotNone(token_data)
        self.assertIn("id_token", token_data)

    def test_live_chrome_scanner(self):
        accounts = get_chrome_signed_in_accounts()
        self.assertIsInstance(accounts, list)
        self.assertGreater(len(accounts), 0)
        for acc in accounts:
            self.assertFalse(is_blacklisted(acc), f"Blacklisted account found: {acc}")

    def test_state_resolver(self):
        manager = AccountManager()
        active_email, accounts = manager.resolve_accounts()
        self.assertIsNotNone(active_email)
        self.assertGreater(len(accounts), 0)

        # Active account must have is_active == True and status_code == ACTIVE
        active_found = False
        for acc in accounts:
            if acc.email == active_email:
                self.assertTrue(acc.is_active)
                self.assertEqual(acc.status_code, "ACTIVE")
                self.assertEqual(acc.status_color, "green")
                active_found = True

        self.assertTrue(active_found, "Active account was not found in resolved list")


if __name__ == "__main__":
    unittest.main()
