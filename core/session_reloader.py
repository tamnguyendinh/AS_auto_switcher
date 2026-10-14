"""
Session Reloader for Antigravity.
Sends reload signal to Antigravity window or prompts refresh.
"""

import ctypes
from ctypes import wintypes
import time

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
