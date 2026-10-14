"""
JSON-based Account Store for Antigravity Switcher.
Stores each account's complete OAuth token in an individual JSON file:
  accounts_store/<email>.json
Acts as the permanent multi-account memory on behalf of Antigravity.
"""

import os
import json
import re
from typing import Dict, Any, Optional, List


def get_default_store_dir() -> str:
    """Returns the accounts_store directory located within the project workspace."""
    base_dir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    store_dir = os.path.join(base_dir, "accounts_store")
    os.makedirs(store_dir, exist_ok=True)
    return store_dir


def sanitize_filename(email: str) -> str:
    """Converts email into a safe filename."""
    norm = email.strip().lower()
    # Replace any unsafe characters
    safe = re.sub(r'[^a-zA-Z0-9_.@-]', '_', norm)
    return f"{safe}.json"


class AccountStore:
    """Manages reading and writing individual account JSON files in accounts_store/."""

    def __init__(self, store_dir: Optional[str] = None):
        self.store_dir = store_dir or get_default_store_dir()
        os.makedirs(self.store_dir, exist_ok=True)

    def _get_account_path(self, email: str) -> str:
        filename = sanitize_filename(email)
        return os.path.join(self.store_dir, filename)

    def store_account(self, email: str, token_data: Dict[str, Any]) -> bool:
        """Stores or updates the JSON file for a given email."""
        if not email or not token_data:
            return False

        path = self._get_account_path(email)
        try:
            payload = {
                "email": email.strip().lower(),
                "saved_at": os.path.getmtime(path) if os.path.exists(path) else 0,
                "token_data": token_data
            }
            with open(path, "w", encoding="utf-8") as f:
                json.dump(payload, f, indent=2, ensure_ascii=False)
            return True
        except Exception:
            return False

    def get_account_token(self, email: str) -> Optional[Dict[str, Any]]:
        """Reads and returns the token_data dictionary for an email."""
        path = self._get_account_path(email)
        if not os.path.exists(path):
            return None

        try:
            with open(path, "r", encoding="utf-8") as f:
                data = json.load(f)
            return data.get("token_data")
        except Exception:
            return None

    def has_account(self, email: str) -> bool:
        """Checks if a valid JSON file exists for the given email."""
        path = self._get_account_path(email)
        if not os.path.exists(path):
            return False
        try:
            with open(path, "r", encoding="utf-8") as f:
                data = json.load(f)
            return bool(data.get("token_data"))
        except Exception:
            return False

    def remove_account(self, email: str) -> bool:
        """Removes the JSON file for an email."""
        path = self._get_account_path(email)
        if os.path.exists(path):
            try:
                os.remove(path)
                return True
            except Exception:
                return False
        return False

    def list_stored_emails(self) -> List[str]:
        """Returns a list of all emails currently saved in accounts_store/."""
        if not os.path.exists(self.store_dir):
            return []

        emails: List[str] = []
        try:
            for file in os.listdir(self.store_dir):
                if file.endswith(".json"):
                    file_path = os.path.join(self.store_dir, file)
                    try:
                        with open(file_path, "r", encoding="utf-8") as f:
                            data = json.load(f)
                        em = data.get("email")
                        if em:
                            emails.append(em)
                    except Exception:
                        pass
        except Exception:
            pass

        return sorted(emails)
