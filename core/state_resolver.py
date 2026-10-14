"""
State Resolver & Account Switch Manager.
Orchestrates Chrome discovery, AccountStore (accounts_store/<email>.json), and Windows Credential Manager.
Resolves each account into:
  🟢 ACTIVE: Currently loaded in Antigravity
  🟢 READY: In Chrome AND saved in accounts_store/
  🟡 UNAUTH: In Chrome but not yet saved in accounts_store/
"""

from dataclasses import dataclass
from typing import List, Optional, Tuple
from scanner.chrome_scanner import get_chrome_signed_in_accounts
from core.credential_manager import read_antigravity_token, write_antigravity_token
from core.jwt_helper import extract_email_from_token_data
from core.account_store import AccountStore
from core.session_reloader import reload_antigravity_window


@dataclass
class AccountInfo:
    email: str
    is_active: bool
    has_token: bool
    status_code: str  # "ACTIVE", "READY", "UNAUTH"
    status_color: str # "green", "yellow"
    display_badge: str


class AccountManager:
    """Core state manager for Antigravity multi-account switching."""

    def __init__(self, store: Optional[AccountStore] = None):
        self.store = store or AccountStore()

    def get_current_active_email(self) -> Optional[str]:
        """Reads and extracts the email currently active in Antigravity."""
        token_data = read_antigravity_token()
        if not token_data:
            return None
        return extract_email_from_token_data(token_data)

    def auto_capture_active_session(self) -> Optional[str]:
        """Automatically captures current active Antigravity session into accounts_store/."""
        token_data = read_antigravity_token()
        if not token_data:
            return None
        email = extract_email_from_token_data(token_data)
        if email:
            self.store.store_account(email, token_data)
            return email
        return None

    def resolve_accounts(self) -> Tuple[Optional[str], List[AccountInfo]]:
        """
        Scans Chrome and accounts_store/ to resolve the live account list and their states.
        Accounts removed from Chrome will automatically not appear in this list.
        """
        # Ensure active session is saved into accounts_store
        active_email = self.auto_capture_active_session()
        if not active_email:
            active_email = self.get_current_active_email()

        # Dynamic discovery from Chrome
        chrome_emails = get_chrome_signed_in_accounts()

        account_list: List[AccountInfo] = []

        for email in chrome_emails:
            norm_email = email.strip().lower()
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
                display_badge=display_badge
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
