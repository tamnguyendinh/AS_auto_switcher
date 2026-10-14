"""
Session Reloader for Antigravity.
Sends reload signal to Antigravity window or prompts refresh.
"""

import ctypes
from ctypes import wintypes
import time
import logging
from typing import Optional, List, Callable
from core.api_connector import LanguageServerInfo, trigger_reauth

logger = logging.getLogger(__name__)

user32 = ctypes.WinDLL("user32.dll", use_last_error=True)

WNDENUMPROC = ctypes.WINFUNCTYPE(wintypes.BOOL, wintypes.HWND, wintypes.LPARAM)

EnumWindows = user32.EnumWindows
EnumWindows.argtypes = [WNDENUMPROC, wintypes.LPARAM]
EnumWindows.restype = wintypes.BOOL

GetWindowThreadProcessId = user32.GetWindowThreadProcessId
GetWindowThreadProcessId.argtypes = [wintypes.HWND, ctypes.POINTER(wintypes.DWORD)]
GetWindowThreadProcessId.restype = wintypes.DWORD

GetWindowTextW = user32.GetWindowTextW
GetWindowTextW.argtypes = [wintypes.HWND, wintypes.LPWSTR, ctypes.c_int]
GetWindowTextW.restype = ctypes.c_int

IsWindowVisible = user32.IsWindowVisible
IsWindowVisible.argtypes = [wintypes.HWND]
IsWindowVisible.restype = wintypes.BOOL

SetForegroundWindow = user32.SetForegroundWindow
SetForegroundWindow.argtypes = [wintypes.HWND]
SetForegroundWindow.restype = wintypes.BOOL

keybd_event = user32.keybd_event
keybd_event.argtypes = [wintypes.BYTE, wintypes.BYTE, wintypes.DWORD, ctypes.c_void_p]
keybd_event.restype = None

VK_CONTROL = 0x11
VK_R = 0x52
KEYEVENTF_KEYUP = 0x0002


def find_antigravity_windows() -> list[wintypes.HWND]:
    """Finds all visible window handles belonging to Antigravity."""
    hwnds: list[wintypes.HWND] = []

    def enum_proc(hwnd, lparam):
        if IsWindowVisible(hwnd):
            pid = wintypes.DWORD()
            GetWindowThreadProcessId(hwnd, ctypes.byref(pid))
            buf = ctypes.create_unicode_buffer(512)
            GetWindowTextW(hwnd, buf, 512)
            title = buf.value
            # Check window title or class
            if "antigravity" in title.lower():
                hwnds.append(hwnd)
        return True

    EnumWindows(WNDENUMPROC(enum_proc), 0)
    return hwnds


def reload_antigravity_window() -> bool:
    """
    Sends Ctrl+R to Antigravity window if found, to refresh the Electron window cleanly.
    """
    windows = find_antigravity_windows()
    if not windows:
        return False

    for hwnd in windows:
        SetForegroundWindow(hwnd)
        time.sleep(0.05)
        # Press Ctrl+R
        keybd_event(VK_CONTROL, 0, 0, None)
        keybd_event(VK_R, 0, 0, None)
        time.sleep(0.05)
        keybd_event(VK_R, 0, KEYEVENTF_KEYUP, None)
        keybd_event(VK_CONTROL, 0, KEYEVENTF_KEYUP, None)
        time.sleep(0.1)

    return True


def reload_antigravity_session(
    info: Optional[LanguageServerInfo] = None,
    reload_window: bool = True,
    window_reloader: Optional[Callable[[], bool]] = None,
    reauth_trigger: Optional[Callable[[LanguageServerInfo], bool]] = None,
) -> bool:
    """
    Coordinated reload bridge for Antigravity.
    1. If `info` is provided, attempts internal Re-Auth signal via `trigger_reauth`.
    2. If `reload_window` is True (or if `trigger_reauth` fails/endpoint not present), invokes window reload (Ctrl+R).
    Returns True if either reauth signal succeeded or window reload succeeded (or False if both failed).
    """
    reauth_func = reauth_trigger or trigger_reauth
    win_func = window_reloader or reload_antigravity_window

    reauth_ok = False
    if info is not None:
        try:
            reauth_ok = reauth_func(info)
        except Exception as exc:
            logger.debug("trigger_reauth failed: %s", exc)
            reauth_ok = False

    window_ok = False
    if reload_window or not reauth_ok:
        try:
            window_ok = win_func()
        except Exception as exc:
            logger.debug("window reload failed: %s", exc)
            window_ok = False

    return bool(reauth_ok or window_ok)
