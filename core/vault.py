"""
Secure local token storage encrypted with Windows DPAPI (CryptProtectData / CryptUnprotectData).
Stores account tokens by email in a local encrypted vault file.
Only the current logged-in Windows user can decrypt this file.
"""

import os
import json
import ctypes
from ctypes import wintypes
from typing import Dict, Any, Optional

crypt32 = ctypes.WinDLL("Crypt32.dll", use_last_error=True)
kernel32 = ctypes.WinDLL("Kernel32.dll", use_last_error=True)

class DATA_BLOB(ctypes.Structure):
    _fields_ = [
        ("cbData", wintypes.DWORD),
        ("pbData", ctypes.c_void_p),
    ]

CryptProtectData = crypt32.CryptProtectData
CryptProtectData.argtypes = [
    ctypes.POINTER(DATA_BLOB),
    wintypes.LPCWSTR,
    ctypes.POINTER(DATA_BLOB),
    ctypes.c_void_p,
    ctypes.c_void_p,
    wintypes.DWORD,
    ctypes.POINTER(DATA_BLOB)
]
CryptProtectData.restype = wintypes.BOOL

CryptUnprotectData = crypt32.CryptUnprotectData
CryptUnprotectData.argtypes = [
    ctypes.POINTER(DATA_BLOB),
    ctypes.POINTER(wintypes.LPWSTR),
    ctypes.POINTER(DATA_BLOB),
    ctypes.c_void_p,
    ctypes.c_void_p,
    wintypes.DWORD,
    ctypes.POINTER(DATA_BLOB)
]
CryptUnprotectData.restype = wintypes.BOOL

LocalFree = kernel32.LocalFree
LocalFree.argtypes = [ctypes.c_void_p]
LocalFree.restype = ctypes.c_void_p

CRYPTPROTECT_UI_FORBIDDEN = 0x01


def dpapi_encrypt(data: bytes, description: str = "Antigravity Token Vault") -> Optional[bytes]:
    """Encrypts bytes using Windows DPAPI."""
    if not data:
        return b""

    in_size = len(data)
    in_buf = (ctypes.c_byte * in_size)(*data)
    blob_in = DATA_BLOB(in_size, ctypes.cast(in_buf, ctypes.c_void_p))
    blob_out = DATA_BLOB()

    success = CryptProtectData(
        ctypes.byref(blob_in),
        description,
        None,
        None,
        None,
        CRYPTPROTECT_UI_FORBIDDEN,
        ctypes.byref(blob_out)
    )

    if not success:
        return None

    try:
        out_buf = (ctypes.c_byte * blob_out.cbData).from_address(blob_out.pbData)
        return bytes(out_buf)
    finally:
        LocalFree(blob_out.pbData)


def dpapi_decrypt(encrypted_data: bytes) -> Optional[bytes]:
    """Decrypts DPAPI-encrypted bytes."""
    if not encrypted_data:
        return b""

    in_size = len(encrypted_data)
    in_buf = (ctypes.c_byte * in_size)(*encrypted_data)
    blob_in = DATA_BLOB(in_size, ctypes.cast(in_buf, ctypes.c_void_p))
    blob_out = DATA_BLOB()

    success = CryptUnprotectData(
        ctypes.byref(blob_in),
        None,
        None,
        None,
        None,
        CRYPTPROTECT_UI_FORBIDDEN,
        ctypes.byref(blob_out)
    )

    if not success:
        return None

    try:
        out_buf = (ctypes.c_byte * blob_out.cbData).from_address(blob_out.pbData)
        return bytes(out_buf)
    finally:
        LocalFree(blob_out.pbData)


class TokenVault:
    """Manages the encrypted local storage of account tokens."""

    def __init__(self, vault_path: Optional[str] = None):
        if not vault_path:
            app_dir = os.path.join(os.environ.get("LOCALAPPDATA", "."), "antigravity_switcher")
            os.makedirs(app_dir, exist_ok=True)
            vault_path = os.path.join(app_dir, "vault.enc")
        self.vault_path = vault_path
        self._accounts: Dict[str, Dict[str, Any]] = {}
        self.load()

    def load(self) -> bool:
        """Loads and decrypts the vault file."""
        if not os.path.exists(self.vault_path):
            self._accounts = {}
            return True

        try:
            with open(self.vault_path, "rb") as f:
                encrypted = f.read()
            if not encrypted:
                self._accounts = {}
                return True

            decrypted = dpapi_decrypt(encrypted)
            if decrypted is None:
                return False

            self._accounts = json.loads(decrypted.decode("utf-8"))
            return True
        except Exception:
            self._accounts = {}
            return False

    def save(self) -> bool:
        """Encrypts and writes the vault file."""
        try:
            json_text = json.dumps(self._accounts, indent=2)
            decrypted_bytes = json_text.encode("utf-8")
            encrypted = dpapi_encrypt(decrypted_bytes)
            if encrypted is None:
                return False

            with open(self.vault_path, "wb") as f:
                f.write(encrypted)
            return True
        except Exception:
            return False

    def store_account(self, email: str, token_data: Dict[str, Any]) -> bool:
        """Stores or updates token data for a specific email."""
        norm_email = email.strip().lower()
        self._accounts[norm_email] = {
            "email": norm_email,
            "token_data": token_data,
            "updated_at": os.path.getmtime(self.vault_path) if os.path.exists(self.vault_path) else 0
        }
        return self.save()

    def get_account_token(self, email: str) -> Optional[Dict[str, Any]]:
        """Retrieves token data for a specific email."""
        norm_email = email.strip().lower()
        acc = self._accounts.get(norm_email)
        if acc:
            return acc.get("token_data")
        return None

    def has_account(self, email: str) -> bool:
        """Checks if a valid token exists for this email."""
        return email.strip().lower() in self._accounts

    def remove_account(self, email: str) -> bool:
        """Removes an account from the vault."""
        norm_email = email.strip().lower()
        if norm_email in self._accounts:
            del self._accounts[norm_email]
            return self.save()
        return False

    def list_stored_emails(self) -> list[str]:
        """Returns all stored emails in the vault."""
        return list(self._accounts.keys())
