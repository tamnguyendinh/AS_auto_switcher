"""
Chrome Profile Discovery Engine.
Scans Google Chrome User Data to dynamically detect signed-in Google accounts.
Enforces exclusion filter (e.g. strict blacklist).
"""

import os
import json
import re
from typing import List, Set


# Strictly filtered words (case-insensitive)
BLACKLIST_KEYWORDS = ["anvien"]


def is_blacklisted(email: str) -> bool:
    """Checks if an email matches any blacklisted keywords."""
    lower = email.lower()
    for kw in BLACKLIST_KEYWORDS:
        if kw in lower:
            return True
    return False


def get_chrome_user_data_path() -> str:
    """Returns the default Chrome User Data directory path on Windows."""
    local_app_data = os.environ.get("LOCALAPPDATA", "")
    return os.path.join(local_app_data, "Google", "Chrome", "User Data")


def extract_emails_from_preferences(pref_path: str) -> Set[str]:
    """Extracts valid Google account emails from a Chrome profile Preferences file."""
    emails: Set[str] = set()
    if not os.path.exists(pref_path):
        return emails

    try:
        with open(pref_path, "r", encoding="utf-8", errors="ignore") as f:
            content = f.read()

        # Regex search for Gmail / Google Workspace email pattern
        found = re.findall(r'[a-zA-Z0-9_.+-]+@gmail\.com|[a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\.[a-zA-Z0-9-.]+', content)
        for em in found:
            norm = em.strip().lower()
            if not is_blacklisted(norm):
                # Filter out obvious false positives like file extensions or schema names
                if not norm.endswith((".png", ".jpg", ".js", ".css", ".json", ".googleapis.com", ".chromium.org")):
                    if "@gmail.com" in norm or ("@" in norm and "." in norm.split("@")[1]):
                        emails.add(norm)

        # Parse structured json if possible for high precision
        try:
            data = json.loads(content)
            # Check account_tracker_service
            tracker = data.get("account_tracker_service_last_update")
            if isinstance(tracker, dict):
                for acc_info in tracker.values():
                    if isinstance(acc_info, dict) and "email" in acc_info:
                        em = acc_info["email"].strip().lower()
                        if not is_blacklisted(em):
                            emails.add(em)
        except Exception:
            pass

    except Exception:
        pass

    return emails


def get_chrome_signed_in_accounts() -> List[str]:
    """
    Scans all Chrome profiles and returns a unique, sorted list of signed-in Google accounts.
    Accounts removed from Chrome will not be present in this list.
    """
    user_data = get_chrome_user_data_path()
    if not os.path.exists(user_data):
        return []

    discovered_emails: Set[str] = set()

    # 1. Check Local State
    local_state_path = os.path.join(user_data, "Local State")
    if os.path.exists(local_state_path):
        try:
            with open(local_state_path, "r", encoding="utf-8", errors="ignore") as f:
                ls_data = json.loads(f.read())
            info_cache = ls_data.get("profile", {}).get("info_cache", {})
            for profile_data in info_cache.values():
                user_name = profile_data.get("user_name", "")
                if user_name and "@" in user_name:
                    norm = user_name.strip().lower()
                    if not is_blacklisted(norm):
                        discovered_emails.add(norm)
        except Exception:
            pass

    # 2. Check Default and Profile * directories
    try:
        entries = os.listdir(user_data)
        for entry in entries:
            full_dir = os.path.join(user_data, entry)
            if os.path.isdir(full_dir):
                if entry == "Default" or entry.startswith("Profile"):
                    pref_path = os.path.join(full_dir, "Preferences")
                    emails = extract_emails_from_preferences(pref_path)
                    discovered_emails.update(emails)
    except Exception:
        pass

    # Filter out non-email formats or blacklisted items
    final_emails = [
        e for e in discovered_emails
        if re.match(r'^[a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\.[a-zA-Z0-9-.]+$', e)
        and not is_blacklisted(e)
    ]

    return sorted(final_emails)
