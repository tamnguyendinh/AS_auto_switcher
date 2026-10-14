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
    get_chrome_account_details,
    ChromeAccountInfo,
)
from core.credential_manager import read_antigravity_token, write_antigravity_token
from core.jwt_helper import extract_email_from_token_data
from core.account_store import AccountStore
from core.session_reloader import (
    reload_antigravity_session,
    pre_patch_tos_acceptance,
    recycle_language_server,
    is_tos_modal_present,
    dismiss_tos_modal_hardware_fallback,
    inject_continuation_prompt,
)
from core.api_connector import (
    find_language_server_process,
    get_user_status,
    auth_logout,
    LanguageServerInfo,
)


class SwitchResult(tuple):
    """
    Result of an account switch operation.
    Acts simultaneously as a 2-tuple `(success: bool, message: str)` for unpacking,
    and as a boolean (`bool(res) == res.success`, `res == True/False`) for direct boolean checks.
    """
    def __new__(cls, success: bool, message: str = ""):
        return super().__new__(cls, (bool(success), str(message)))

    @property
    def success(self) -> bool:
        return self[0]

    @property
    def message(self) -> str:
        return self[1]

    def __bool__(self) -> bool:
        return self[0]

    def __eq__(self, other: Any) -> bool:
        if isinstance(other, bool):
            return self[0] == other
        return super().__eq__(other)

    def __ne__(self, other: Any) -> bool:
        if isinstance(other, bool):
            return self[0] != other
        return super().__ne__(other)

    def __repr__(self) -> str:
        return f"SwitchResult(success={self[0]}, message={self[1]!r})"


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
        logout_fn: Optional[Callable[[Any], bool]] = None,
        reloader_fn: Optional[Callable[..., bool]] = None,
        token_writer: Optional[Callable[[Dict[str, Any]], bool]] = None,
        process_finder: Optional[Callable[[], Any]] = None,
        status_getter: Optional[Callable[[Any], Optional[Dict[str, Any]]]] = None,
        force_logout: bool = False,
        auto_continue: bool = True,
        prompt_text: str = "continue",
        tos_pre_patcher: Optional[Callable[[str], bool]] = None,
        language_server_recycler: Optional[Callable[..., bool]] = None,
        prompt_injector: Optional[Callable[..., bool]] = None,
        tos_modal_checker: Optional[Callable[[], bool]] = None,
        fallback_dismisser: Optional[Callable[[], None]] = None,
    ):
        self.store = store or AccountStore()
        self.user_status_provider = user_status_provider
        self.chrome_accounts_provider = chrome_accounts_provider
        self.credential_reader = credential_reader
        self.logout_fn = logout_fn
        self.reloader_fn = reloader_fn
        self.token_writer = token_writer
        self.process_finder = process_finder
        self.status_getter = status_getter
        self.force_logout = force_logout
        self.auto_continue = auto_continue
        self.prompt_text = prompt_text
        self.tos_pre_patcher = tos_pre_patcher
        self.language_server_recycler = language_server_recycler
        self.prompt_injector = prompt_injector
        self.tos_modal_checker = tos_modal_checker
        self.fallback_dismisser = fallback_dismisser

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
                finder = self.process_finder or find_language_server_process
                ls_info = finder()
                if ls_info is not None:
                    getter = self.status_getter or get_user_status
                    status_dict = getter(ls_info)
                    if status_dict and isinstance(status_dict, dict):
                        ram_email = status_dict.get("email")
                    elif isinstance(status_dict, str):
                        ram_email = status_dict
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
            if not norm_email:
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

    def switch_account(
        self,
        target_email: str,
        force_logout: Optional[bool] = None,
        skip_logout: Optional[bool] = None,
        logout_fn: Optional[Callable[[Any], bool]] = None,
        reloader_fn: Optional[Callable[..., bool]] = None,
        token_writer: Optional[Callable[[Dict[str, Any]], bool]] = None,
        process_finder: Optional[Callable[[], Any]] = None,
        status_getter: Optional[Callable[[Any], Optional[Dict[str, Any]]]] = None,
        auto_continue: Optional[bool] = None,
        prompt_text: Optional[str] = None,
        tos_pre_patcher: Optional[Callable[[str], bool]] = None,
        language_server_recycler: Optional[Callable[..., bool]] = None,
        prompt_injector: Optional[Callable[..., bool]] = None,
        tos_modal_checker: Optional[Callable[[], bool]] = None,
        fallback_dismisser: Optional[Callable[[], None]] = None,
        **kwargs: Any,
    ) -> SwitchResult:
        """
        Executes Seamless Account Hot-Swap with RAM synchronization, ToS bypass,
        and uninterrupted in-flight session continuation:
          Step 1 (GetUserStatus): Discover running language_server.exe and query current active email in RAM.
                                  If already active on target_email, return True (idempotent / no-op).
          Step 2 (AuthLogout): By default, DO NOT call auth_logout()! Bypassing AuthLogout() avoids
                               notifying the Electron IDE that the user logged out, eliminating the Chrome OAuth popup.
                               Only called if force_logout=True (or skip_logout=False).
          Step 3 (Swap Token): Load target token from store. If missing or invalid, abort safely.
                               Write target Token B directly into Windows Credential Manager.
          Step 4 (Pre-patch ToS): Layer 1 Defense: atomize write to %APPDATA%\\Antigravity\\app_storage.json
                                  setting lastLoginUsername and lastLoginIsGcpTos = "true".
          Step 5 (Recycle Language Server): Terminate language_server.exe cleanly, letting monitorLsCrashInternal
                                            respawn it with fresh tokens in RAM without opening Chrome.
          Step 6 (Fallback Check): Layer 2 Defense: check if ToS modal appeared and dismiss via Win32 Enter.
          Step 7 (Prompt Continuation): Inject continuation prompt (default: "continue") to resume AI agent.
        """
        if not target_email or not isinstance(target_email, str):
            return SwitchResult(False, "Email tài khoản không hợp lệ.")

        norm_email = target_email.strip().lower()
        if not norm_email:
            return SwitchResult(False, "Email tài khoản không hợp lệ.")

        # Determine should_logout behavior:
        # Default is False (smooth zero-chrome hot-swap without OAuth popups)
        if callable(force_logout):
            logout_fn = force_logout
            force_logout = None

        if skip_logout is not None:
            should_logout = not bool(skip_logout)
        elif force_logout is not None:
            should_logout = bool(force_logout)
        else:
            should_logout = bool(getattr(self, "force_logout", False))

        # Resolve providers (method parameter override > instance attribute > default)
        finder = process_finder or self.process_finder or find_language_server_process
        getter = status_getter or self.status_getter or get_user_status

        # Step 1 (GetUserStatus): Discover running language_server.exe and query active email
        info: Optional[LanguageServerInfo] = None
        current_active_email: Optional[str] = None

        try:
            info = finder()
        except Exception:
            info = None

        if info is not None:
            try:
                status_res = getter(info)
                if isinstance(status_res, dict):
                    current_active_email = status_res.get("email")
                elif isinstance(status_res, str):
                    current_active_email = status_res
            except Exception:
                current_active_email = None

        # Fallback to user_status_provider if set and info query returned nothing
        if current_active_email is None and self.user_status_provider is not None:
            try:
                res = self.user_status_provider()
                if isinstance(res, dict):
                    current_active_email = res.get("email")
                elif isinstance(res, str):
                    current_active_email = res
            except Exception:
                pass

        if current_active_email and isinstance(current_active_email, str):
            if current_active_email.strip().lower() == norm_email:
                return SwitchResult(True, f"Tài khoản {norm_email} hiện đã đang hoạt động.")

        # Ensure current active session is captured into accounts_store before switching
        self.auto_capture_active_session()

        # Step 3 pre-check: Load target token from store.
        # If missing or invalid, abort safely before logging out or touching credentials.
        get_acc_fn = getattr(self.store, "get_account", None) or getattr(self.store, "get_account_token", None)
        target_token = None
        if callable(get_acc_fn):
            try:
                target_token = get_acc_fn(norm_email)
            except Exception:
                target_token = None

        if isinstance(target_token, dict) and "token_data" in target_token and isinstance(target_token["token_data"], dict):
            target_token = target_token["token_data"]

        if not target_token or not isinstance(target_token, dict):
            return SwitchResult(False, f"Chưa có file JSON của {norm_email} trong accounts_store/!")

        # Step 2 (AuthLogout): Only invoked if should_logout is True
        if should_logout and info is not None:
            logout = logout_fn or self.logout_fn or auth_logout
            try:
                logout(info)
            except Exception:
                pass

        # Step 3 (Swap Token): Write target Token B directly to Windows Credential Manager
        writer = token_writer or self.token_writer or write_antigravity_token
        write_ok = False
        try:
            write_ok = bool(writer(target_token))
        except Exception:
            write_ok = False

        if not write_ok:
            return SwitchResult(False, "Không thể ghi Token vào Windows Credential Manager!")

        # Step 4: Layer 1 ToS Pre-Patching in app_storage.json
        patcher = tos_pre_patcher or getattr(self, "tos_pre_patcher", None) or pre_patch_tos_acceptance
        try:
            patcher(norm_email)
        except Exception:
            pass

        # Step 5: Recycle language_server.exe to re-bind credentials into RAM
        recycler = (
            language_server_recycler
            or reloader_fn
            or getattr(self, "language_server_recycler", None)
            or getattr(self, "reloader_fn", None)
            or recycle_language_server
        )
        try:
            import inspect
            sig = inspect.signature(recycler)
            if "info" in sig.parameters or any(p.kind == inspect.Parameter.VAR_KEYWORD for p in sig.parameters.values()):
                recycler(info=info)
            elif len(sig.parameters) >= 1:
                recycler(info)
            else:
                recycler()
        except TypeError:
            try:
                recycler(info)
            except TypeError:
                recycler()
        except Exception:
            pass

        # Step 6: Layer 2 ToS Dismissal Fallback check
        checker = tos_modal_checker or getattr(self, "tos_modal_checker", None) or is_tos_modal_present
        dismisser = fallback_dismisser or getattr(self, "fallback_dismisser", None) or dismiss_tos_modal_hardware_fallback
        try:
            if checker():
                dismisser()
        except Exception:
            pass

        # Step 7: Autonomous Prompt Continuation
        should_auto_continue = self.auto_continue if auto_continue is None else bool(auto_continue)
        if should_auto_continue:
            injector = prompt_injector or getattr(self, "prompt_injector", None) or inject_continuation_prompt
            try:
                p_text = prompt_text or getattr(self, "prompt_text", "continue") or "continue"
                injector(prompt_text=p_text)
            except Exception:
                pass

        return SwitchResult(True, f"Đã chuyển đổi thành công sang: {norm_email}")
