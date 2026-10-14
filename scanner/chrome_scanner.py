"""
Chrome Profile Discovery Engine.
Scans Google Chrome User Data to dynamically detect signed-in Google accounts.
Enforces exclusion filter (e.g. strict blacklist).
"""

import os
import json
import re
from dataclasses import dataclass
from typing import List, Set, Optional, Dict, Any


# Strictly filtered words (case-insensitive)
BLACKLIST_KEYWORDS = ["anvien"]

# Regular expression matching valid email address format
EMAIL_REGEX = re.compile(r'^[a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\.[a-zA-Z0-9-.]+$')


@dataclass
class ChromeAccountInfo:
    """Rich metadata for a signed-in Chrome account."""
    email: str
    name: str = ""
    profile_dir: str = ""
    avatar_url: Optional[str] = None


def is_blacklisted(email: Optional[str]) -> bool:
    """
    Checks if an email matches any blacklisted keywords (case-insensitive).
    Returns True if blacklisted, False otherwise.
    """
    if not email or not isinstance(email, str):
        return False
    lower = email.lower().strip()
    for kw in BLACKLIST_KEYWORDS:
        if kw in lower:
            return True
    return False


def is_valid_email(email: Optional[str]) -> bool:
    """
    Checks if a string is a syntactically valid email and not an obvious false positive
    or blacklisted address.
    """
    if not email or not isinstance(email, str):
        return False
    norm = email.strip().lower()
    if is_blacklisted(norm):
        return False
    # Filter out obvious false positives like file extensions or internal schemas
    if norm.endswith((".png", ".jpg", ".jpeg", ".js", ".css", ".json", ".googleapis.com", ".chromium.org")):
        return False
    return bool(EMAIL_REGEX.match(norm))


def get_chrome_user_data_path() -> str:
    """Returns the default Chrome User Data directory path on Windows."""
    local_app_data = os.environ.get("LOCALAPPDATA", "")
    return os.path.join(local_app_data, "Google", "Chrome", "User Data")


def extract_emails_from_preferences(pref_path: str) -> Set[str]:
    """
    Extracts valid Google account emails from a Chrome profile Preferences file.
    Handles missing files, locked files, permission errors, and corrupted JSON gracefully.
    """
    emails: Set[str] = set()
    if not pref_path or not os.path.exists(pref_path):
        return emails

    try:
        with open(pref_path, "r", encoding="utf-8", errors="ignore") as f:
            content = f.read()

        # Regex search for Gmail / Google Workspace email pattern
        found = re.findall(r'[a-zA-Z0-9_.+-]+@gmail\.com|[a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\.[a-zA-Z0-9-.]+', content)
        for em in found:
            norm = em.strip().lower()
            if is_valid_email(norm):
                emails.add(norm)

        # Parse structured json if possible for high precision
        try:
            data = json.loads(content)
            if isinstance(data, dict):
                # 1. Modern Chrome: account_info list of dicts
                acc_info_list = data.get("account_info")
                if isinstance(acc_info_list, list):
                    for acc in acc_info_list:
                        if isinstance(acc, dict) and "email" in acc:
                            em = str(acc["email"]).strip().lower()
                            if is_valid_email(em):
                                emails.add(em)

                # 2. Older Chrome: account_tracker_service_last_update dict
                tracker = data.get("account_tracker_service_last_update")
                if isinstance(tracker, dict):
                    for acc in tracker.values():
                        if isinstance(acc, dict) and "email" in acc:
                            em = str(acc["email"]).strip().lower()
                            if is_valid_email(em):
                                emails.add(em)
        except Exception:
            pass

    except (PermissionError, FileNotFoundError, OSError, UnicodeDecodeError):
        pass
    except Exception:
        pass

    return emails


def get_chrome_account_details(user_data_path: Optional[str] = None) -> List[ChromeAccountInfo]:
    """
    Scans all Chrome profiles and returns a unique list of ChromeAccountInfo objects sorted by email.
    Extracts rich metadata (name, profile_dir, avatar_url) from Local State and Preferences.
    Strictly excludes any blacklisted accounts.
    """
    path = user_data_path if user_data_path is not None else get_chrome_user_data_path()
    if not path or not os.path.exists(path):
        return []

    accounts_map: Dict[str, ChromeAccountInfo] = {}
    known_profile_dirs: Set[str] = set()

    # 1. Check Local State (contains info_cache / profiles_attributes)
    local_state_path = os.path.join(path, "Local State")
    if os.path.exists(local_state_path):
        try:
            with open(local_state_path, "r", encoding="utf-8", errors="ignore") as f:
                ls_data = json.loads(f.read())
            if isinstance(ls_data, dict):
                profile_section = ls_data.get("profile", {})
                if isinstance(profile_section, dict):
                    info_cache = profile_section.get("info_cache") or profile_section.get("profiles_attributes") or {}
                    if isinstance(info_cache, dict):
                        for prof_dir, p_data in info_cache.items():
                            known_profile_dirs.add(str(prof_dir))
                            if not isinstance(p_data, dict):
                                continue
                            user_name = p_data.get("user_name", "")
                            if user_name and is_valid_email(str(user_name)):
                                em = str(user_name).strip().lower()
                                if not is_blacklisted(em):
                                    name = str(p_data.get("gaia_name") or p_data.get("name") or "").strip()
                                    avatar = p_data.get("last_downloaded_gaia_picture_url_with_size") or p_data.get("picture_url") or None
                                    if avatar and not isinstance(avatar, str):
                                        avatar = str(avatar)
                                    accounts_map[em] = ChromeAccountInfo(
                                        email=em,
                                        name=name,
                                        profile_dir=str(prof_dir),
                                        avatar_url=avatar,
                                    )
        except (PermissionError, FileNotFoundError, OSError, UnicodeDecodeError, json.JSONDecodeError, ValueError):
            pass
        except Exception:
            pass

    # 2. Check Default, Profile *, and any discovered profile directories
    try:
        entries = os.listdir(path)
    except (PermissionError, FileNotFoundError, OSError):
        entries = []
    except Exception:
        entries = []

    for entry in entries:
        full_dir = os.path.join(path, entry)
        try:
            if not os.path.isdir(full_dir):
                continue
        except (PermissionError, OSError):
            continue

        # Inspect if directory is a standard Chrome profile or listed in Local State
        if entry == "Default" or entry.startswith("Profile") or entry in known_profile_dirs:
            pref_path = os.path.join(full_dir, "Preferences")
            if not os.path.exists(pref_path):
                continue

            try:
                with open(pref_path, "r", encoding="utf-8", errors="ignore") as f:
                    pref_content = f.read()
            except (PermissionError, FileNotFoundError, OSError, UnicodeDecodeError):
                continue
            except Exception:
                continue

            # Parse structured JSON if possible
            pref_data = None
            try:
                pref_data = json.loads(pref_content)
            except Exception:
                pass

            if isinstance(pref_data, dict):
                # Modern Chrome: account_info list
                acc_list = pref_data.get("account_info")
                if isinstance(acc_list, list):
                    for item in acc_list:
                        if not isinstance(item, dict):
                            continue
                        raw_em = item.get("email")
                        if raw_em and is_valid_email(str(raw_em)):
                            em = str(raw_em).strip().lower()
                            if not is_blacklisted(em):
                                fn = str(item.get("full_name") or item.get("given_name") or "").strip()
                                pic = item.get("picture_url") or item.get("last_downloaded_image_url_with_size") or None
                                if pic and not isinstance(pic, str):
                                    pic = str(pic)
                                if em not in accounts_map:
                                    accounts_map[em] = ChromeAccountInfo(
                                        email=em,
                                        name=fn,
                                        profile_dir=entry,
                                        avatar_url=pic,
                                    )
                                else:
                                    existing = accounts_map[em]
                                    if not existing.name and fn:
                                        existing.name = fn
                                    if not existing.avatar_url and pic:
                                        existing.avatar_url = pic
                                    if not existing.profile_dir:
                                        existing.profile_dir = entry

                # Older Chrome: account_tracker_service_last_update dict
                tracker = pref_data.get("account_tracker_service_last_update")
                if isinstance(tracker, dict):
                    for item in tracker.values():
                        if not isinstance(item, dict):
                            continue
                        raw_em = item.get("email")
                        if raw_em and is_valid_email(str(raw_em)):
                            em = str(raw_em).strip().lower()
                            if not is_blacklisted(em):
                                fn = str(item.get("full_name") or item.get("given_name") or "").strip()
                                pic = item.get("picture_url") or None
                                if pic and not isinstance(pic, str):
                                    pic = str(pic)
                                if em not in accounts_map:
                                    accounts_map[em] = ChromeAccountInfo(
                                        email=em,
                                        name=fn,
                                        profile_dir=entry,
                                        avatar_url=pic,
                                    )
                                else:
                                    existing = accounts_map[em]
                                    if not existing.name and fn:
                                        existing.name = fn
                                    if not existing.avatar_url and pic:
                                        existing.avatar_url = pic
                                    if not existing.profile_dir:
                                        existing.profile_dir = entry

            # Fallback regex search on Preferences content
            regex_emails = extract_emails_from_preferences(pref_path)
            for em in regex_emails:
                if is_valid_email(em) and not is_blacklisted(em):
                    if em not in accounts_map:
                        accounts_map[em] = ChromeAccountInfo(
                            email=em,
                            name="",
                            profile_dir=entry,
                            avatar_url=None,
                        )
                    elif not accounts_map[em].profile_dir:
                        accounts_map[em].profile_dir = entry

    # Final filtering and sorting
    final_accounts = [
        acc for acc in accounts_map.values()
        if is_valid_email(acc.email) and not is_blacklisted(acc.email)
    ]
    final_accounts.sort(key=lambda x: x.email)
    return final_accounts


def get_chrome_signed_in_accounts(user_data_path: Optional[str] = None) -> List[str]:
    """
    Scans all Chrome profiles and returns a unique, sorted list of signed-in Google accounts.
    Accounts removed from Chrome or matching blacklist keywords are excluded.
    """
    details = get_chrome_account_details(user_data_path=user_data_path)
    return [acc.email for acc in details]
