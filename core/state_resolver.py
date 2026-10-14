"""
State Resolver & Account Switch Manager.
Orchestrates Chrome discovery, AccountStore (accounts_store/<email>.json),
Antigravity Language Server RAM API, and Windows Credential Manager.
Resolves each account into:
  🟢 ACTIVE: Currently loaded in Antigravity language server RAM (or credential fallback)
  🟢 READY: In Chrome AND saved in accounts_store/
  🟡 UNAUTH: In Chrome but not yet saved in accounts_store/
"""

from dataclasses import dataclass
from typing import List, Optional, Tuple, Set, Callable, Union, Dict, Any
from scanner.chrome_scanner import (
    get_chrome_signed_in_accounts,
    get_chrome_account_details,
    ChromeAccountInfo,
    is_blacklisted,
)
from core.credential_manager import read_antigravity_token, write_antigravity_token
from core.jwt_helper import extract_email_from_token_data
from core.account_store import AccountStore
from core.session_reloader import reload_antigravity_window
from core.api_connector import find_language_server_process, get_user_status


@dataclass
class AccountInfo:
    email: str
    is_active: bool
    has_token: bool
    status_code: str  # "ACTIVE", "READY", "UNAUTH"
    status_color: str # "green", "yellow"
    display_badge: str
    name: str = ""
    avatar_url: Optional[str] = None


class AccountManager:
    """Core state manager for Antigravity multi-account switching."""

    def __init__(
        self,
        store: Optional[AccountStore] = None,
        user_status_provider: Optional[Callable[[], Optional[str]]] = None,
        chrome_accounts_provider: Optional[Callable[[], Union[List[str], List[ChromeAccountInfo]]]] = None,
        credential_reader: Optional[Callable[[], Optional[Dict[str, Any]]]] = None,
    ):
        self.store = store or AccountStore()
        self.user_status_provider = user_status_provider
        self.chrome_accounts_provider = chrome_accounts_provider
        self.credential_reader = credential_reader

    def get_current_active_email(
        self,
        user_status_provider: Optional[Callable[[], Optional[str]]] = None
    ) -> Optional[str]:
        """
        Reads and extracts the email currently active in Antigravity.
        Prioritizes live RAM identity from language_server.exe (GetUserStatus),
        falling back to Windows Credential Manager if language server is not running or returns None.
        """
        provider = user_status_provider or self.user_status_provider
        ram_email: Optional[str] = None

        if provider is not None:
            try:
                res = provider()
                if isinstance(res, dict):
                    ram_email = res.get("email")
                elif isinstance(res, str):
                    ram_email = res
            except Exception:
                ram_email = None
        else:
            try:
                ls_info = find_language_server_process()
                if ls_info is not None:
                    status_dict = get_user_status(ls_info)
                    if status_dict and isinstance(status_dict, dict):
                        ram_email = status_dict.get("email")
            except Exception:
                ram_email = None

        if ram_email and isinstance(ram_email, str) and ram_email.strip():
            return ram_email.strip().lower()

        # Fallback to Windows Credential Manager
        try:
            reader = self.credential_reader or read_antigravity_token
            token_data = reader()
            if token_data and isinstance(token_data, dict):
                cred_email = extract_email_from_token_data(token_data)
                if cred_email and isinstance(cred_email, str) and cred_email.strip():
                    return cred_email.strip().lower()
        except Exception:
            pass

        return None

    def auto_capture_active_session(self) -> Optional[str]:
        """Automatically captures current active Antigravity session into accounts_store/."""
        try:
            reader = self.credential_reader or read_antigravity_token
            token_data = reader()
            if not token_data or not isinstance(token_data, dict):
                return None
            email = extract_email_from_token_data(token_data)
            if email and isinstance(email, str) and email.strip():
                norm_email = email.strip().lower()
                self.store.store_account(norm_email, token_data)
                return norm_email
        except Exception:
            pass
        return None

    def resolve_accounts(
        self,
        chrome_accounts_provider: Optional[Callable[[], Union[List[str], List[ChromeAccountInfo]]]] = None,
        user_status_provider: Optional[Callable[[], Optional[str]]] = None,
    ) -> Tuple[Optional[str], List[AccountInfo]]:
        """
        Scans Chrome and accounts_store/ to resolve the live account list and their states.
        Accounts removed from Chrome will automatically not appear in this list.
        Categorizes accounts into:
          🟢 ACTIVE: Currently active in language server RAM (or credential fallback)
          🟢 READY: Saved in accounts_store/<email>.json
          🟡 UNAUTH: In Chrome but not yet saved in accounts_store/
        """
        # Ensure active session is saved into accounts_store if credential present
        self.auto_capture_active_session()
        active_email = self.get_current_active_email(user_status_provider=user_status_provider)

        # Dynamic discovery from Chrome
        chrome_provider = chrome_accounts_provider or self.chrome_accounts_provider
        if chrome_provider is not None:
            try:
                raw_accounts = chrome_provider()
            except Exception:
                raw_accounts = []
        else:
            raw_accounts = get_chrome_account_details()

        account_list: List[AccountInfo] = []
        seen_emails: Set[str] = set()

        for item in raw_accounts:
            if isinstance(item, str):
                raw_email = item
                name = ""
                avatar_url = None
            elif isinstance(item, ChromeAccountInfo):
                raw_email = item.email
                name = item.name
                avatar_url = item.avatar_url
            elif isinstance(item, dict):
                raw_email = item.get("email", "")
                name = item.get("name", "")
                avatar_url = item.get("avatar_url")
            else:
                raw_email = getattr(item, "email", str(item))
                name = getattr(item, "name", "")
                avatar_url = getattr(item, "avatar_url", None)

            if not raw_email or not isinstance(raw_email, str):
                continue

            norm_email = raw_email.strip().lower()
            if not norm_email or is_blacklisted(norm_email):
                continue

            if norm_email in seen_emails:
                continue
            seen_emails.add(norm_email)

            is_active = (active_email is not None and norm_email == active_email.strip().lower())
            has_token = self.store.has_account(norm_email) or is_active

            if is_active:
                status_code = "ACTIVE"
                status_color = "green"
                display_badge = "[ĐANG DÙNG]"
            elif has_token:
                status_code = "READY"
                status_color = "green"
                display_badge = "[⚡ Chuyển sang]"
            else:
                status_code = "UNAUTH"
                status_color = "yellow"
                display_badge = "[Chưa lưu vào kho]"

            account_list.append(AccountInfo(
                email=norm_email,
                is_active=is_active,
                has_token=has_token,
                status_code=status_code,
                status_color=status_color,
                display_badge=display_badge,
                name=name,
                avatar_url=avatar_url,
            ))

        return active_email, account_list

    def switch_account(self, target_email: str) -> Tuple[bool, str]:
        """
        Switches Antigravity session to the target email.
        Writes token from accounts_store/<email>.json to Windows Credential Manager and triggers reload.
        """
        norm_email = target_email.strip().lower()

        # Save current session first before switching
        self.auto_capture_active_session()

        target_token = self.store.get_account_token(norm_email)
        if not target_token:
            return False, f"Chưa có file JSON của {norm_email} trong accounts_store/!"

        success = write_antigravity_token(target_token)
        if not success:
            return False, "Không thể ghi Token vào Windows Credential Manager!"

        # Trigger soft reload
        reload_antigravity_window()

        return True, f"Đã chuyển đổi thành công sang: {norm_email}"
