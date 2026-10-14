"""
Windows Credential Manager API wrapper using ctypes.
Directly interfaces with Advapi32.dll (CredReadW, CredWriteW, CredDeleteW).
Targets Antigravity's stored OAuth token: LegacyGeneric:target=gemini:antigravity
"""

import ctypes
from ctypes import wintypes
import json
from typing import Optional, Dict, Any

CRED_TYPE_GENERIC = 1
CRED_PERSIST_LOCAL_MACHINE = 2
ANTIGRAVITY_TARGET = "LegacyGeneric:target=gemini:antigravity"
ANTIGRAVITY_USER = "antigravity"

advapi32 = ctypes.WinDLL("Advapi32.dll", use_last_error=True)

class CREDENTIALW(ctypes.Structure):
    _fields_ = [
        ("Flags", wintypes.DWORD),
        ("Type", wintypes.DWORD),
        ("TargetName", wintypes.LPWSTR),
        ("Comment", wintypes.LPWSTR),
        ("LastWritten", wintypes.FILETIME),
        ("CredentialBlobSize", wintypes.DWORD),
        ("CredentialBlob", ctypes.c_void_p),
        ("Persist", wintypes.DWORD),
        ("AttributeCount", wintypes.DWORD),
        ("Attributes", ctypes.c_void_p),
        ("TargetAlias", wintypes.LPWSTR),
        ("UserName", wintypes.LPWSTR),
    ]

PCREDENTIALW = ctypes.POINTER(CREDENTIALW)

# Function signatures
CredReadW = advapi32.CredReadW
CredReadW.argtypes = [wintypes.LPCWSTR, wintypes.DWORD, wintypes.DWORD, ctypes.POINTER(PCREDENTIALW)]
CredReadW.restype = wintypes.BOOL

CredWriteW = advapi32.CredWriteW
CredWriteW.argtypes = [PCREDENTIALW, wintypes.DWORD]
CredWriteW.restype = wintypes.BOOL

CredFree = advapi32.CredFree
CredFree.argtypes = [ctypes.c_void_p]
CredFree.restype = None

CredDeleteW = advapi32.CredDeleteW
CredDeleteW.argtypes = [wintypes.LPCWSTR, wintypes.DWORD, wintypes.DWORD]
CredDeleteW.restype = wintypes.BOOL


def read_raw_credential(target: str = ANTIGRAVITY_TARGET) -> Optional[bytes]:
    """Reads raw CredentialBlob bytes from Windows Credential Manager."""
    p_cred = PCREDENTIALW()
    if not CredReadW(target, CRED_TYPE_GENERIC, 0, ctypes.byref(p_cred)):
        return None
    try:
        cred = p_cred.contents
        size = cred.CredentialBlobSize
        if not cred.CredentialBlob or size == 0:
            return b""
        buffer = (ctypes.c_byte * size).from_address(cred.CredentialBlob)
        return bytes(buffer)
    finally:
        CredFree(p_cred)


def write_raw_credential(
    blob: bytes,
    target: str = ANTIGRAVITY_TARGET,
    user_name: str = ANTIGRAVITY_USER
) -> bool:
    """Writes raw bytes into Windows Credential Manager."""
    blob_len = len(blob)
    blob_buf = (ctypes.c_byte * blob_len)(*blob)
    blob_ptr = ctypes.cast(blob_buf, ctypes.c_void_p)

    cred = CREDENTIALW()
    cred.Flags = 0
    cred.Type = CRED_TYPE_GENERIC
    cred.TargetName = target
    cred.Comment = None
    cred.CredentialBlobSize = blob_len
    cred.CredentialBlob = blob_ptr
    cred.Persist = CRED_PERSIST_LOCAL_MACHINE
    cred.AttributeCount = 0
    cred.Attributes = None
    cred.TargetAlias = None
    cred.UserName = user_name

    res = CredWriteW(ctypes.byref(cred), 0)
    return bool(res)


def read_antigravity_token() -> Optional[Dict[str, Any]]:
    """Reads and parses the active Antigravity JSON token dictionary."""
    raw = read_raw_credential(ANTIGRAVITY_TARGET)
    if not raw:
        return None
    try:
        text = raw.decode("utf-8")
        return json.loads(text)
    except Exception:
        return None


def write_antigravity_token(token_data: Dict[str, Any]) -> bool:
    """Serializes token dict to JSON and updates the active Antigravity credential."""
    try:
        text = json.dumps(token_data)
        blob = text.encode("utf-8")
        return write_raw_credential(blob, ANTIGRAVITY_TARGET, ANTIGRAVITY_USER)
    except Exception:
        return False
