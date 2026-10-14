"""
Session Reloader for Antigravity.
Sends reload signal to Antigravity window or prompts refresh.
"""

import ctypes
from ctypes import wintypes
import time
import logging
from typing import Optional, List, Callable, Any
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


import subprocess
import csv
import io

def find_antigravity_pids(
    command_runner: Optional[Callable[..., Any]] = None,
) -> List[int]:
    """
    Discovers all running PIDs belonging to process Antigravity.exe (or Antigravity)
    using standard Windows tools (tasklist /FI "IMAGENAME eq Antigravity.exe" /FO CSV).
    Handles missing tools or errors gracefully.
    """
    pids: List[int] = []
    try:
        if command_runner is not None:
            output = command_runner(["tasklist", "/FI", "IMAGENAME eq Antigravity.exe", "/FO", "CSV"])
            if hasattr(output, "stdout"):
                raw_text = output.stdout
            elif isinstance(output, list):
                for item in output:
                    if isinstance(item, int) and item > 0 and item not in pids:
                        pids.append(item)
                    elif isinstance(item, str) and item.isdigit() and int(item) > 0 and int(item) not in pids:
                        pids.append(int(item))
                return pids
            else:
                raw_text = str(output or "")
        else:
            creationflags = 0
            if hasattr(subprocess, "CREATE_NO_WINDOW"):
                creationflags = subprocess.CREATE_NO_WINDOW
            res = subprocess.run(
                ["tasklist", "/FI", "IMAGENAME eq Antigravity.exe", "/FO", "CSV"],
                capture_output=True,
                text=True,
                timeout=5,
                creationflags=creationflags,
            )
            raw_text = res.stdout

        reader = csv.reader(io.StringIO(raw_text))
        for row in reader:
            if len(row) >= 2:
                img_name = row[0].strip().lower()
                pid_str = row[1].strip()
                if pid_str.isdigit() and ("antigravity" in img_name):
                    pid_val = int(pid_str)
                    if pid_val > 0 and pid_val not in pids:
                        pids.append(pid_val)
    except Exception as exc:
        logger.debug("find_antigravity_pids error: %s", exc)

    return pids


def find_antigravity_windows(
    pid_provider: Optional[Callable[[], List[int]]] = None,
    enum_windows_fn: Optional[Callable[[Any, int], bool]] = None,
) -> List[wintypes.HWND]:
    """
    Finds all visible window handles belonging to Antigravity.
    Matches visible window handles (HWND) whose GetWindowThreadProcessId matches any PID of Antigravity.exe.
    Retains fallback window title check (if "antigravity" in title.lower()) if PID discovery returns empty.
    Supports injecting pid_provider and enum_windows_fn for hermetic testing.
    """
    hwnds: List[wintypes.HWND] = []
    try:
        finder = pid_provider if pid_provider is not None else find_antigravity_pids
        target_pids = set(finder() or [])
    except Exception:
        target_pids = set()

    def enum_proc(hwnd, lparam):
        if IsWindowVisible(hwnd):
            pid = wintypes.DWORD()
            GetWindowThreadProcessId(hwnd, ctypes.byref(pid))
            curr_pid = int(pid.value)
            if target_pids:
                if curr_pid in target_pids:
                    hwnds.append(hwnd)
            else:
                buf = ctypes.create_unicode_buffer(512)
                GetWindowTextW(hwnd, buf, 512)
                title = buf.value
                if "antigravity" in title.lower():
                    hwnds.append(hwnd)
        return True

    enum_fn = enum_windows_fn or EnumWindows
    try:
        enum_fn(WNDENUMPROC(enum_proc), 0)
    except Exception as exc:
        logger.debug("find_antigravity_windows error: %s", exc)

    return hwnds


def reload_antigravity_window(
    windows: Optional[List[wintypes.HWND]] = None,
    windows_finder: Optional[Callable[[], List[wintypes.HWND]]] = None,
) -> bool:
    """
    Sends Ctrl+R to Antigravity window if found, to refresh the Electron window cleanly.
    """
    if windows is None:
        finder = windows_finder or find_antigravity_windows
        windows = finder()
    if not windows:
        return False

    for hwnd in windows:
        try:
            SetForegroundWindow(hwnd)
            time.sleep(0.05)
            # Press Ctrl+R
            keybd_event(VK_CONTROL, 0, 0, None)
            keybd_event(VK_R, 0, 0, None)
            time.sleep(0.05)
            keybd_event(VK_R, 0, KEYEVENTF_KEYUP, None)
            keybd_event(VK_CONTROL, 0, KEYEVENTF_KEYUP, None)
            time.sleep(0.1)
        except Exception as exc:
            logger.debug("reload_antigravity_window error on HWND %s: %s", hwnd, exc)

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
