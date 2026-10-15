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
VK_RETURN = 0x0D
VK_L = 0x4C
VK_V = 0x56
VK_R = 0x52
KEYEVENTF_KEYUP = 0x0002


import os
import json
import tempfile
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


# Backward compatibility alias
reload_session = reload_antigravity_session


def get_app_storage_path() -> str:
    """Returns the absolute path to Antigravity's app_storage.json."""
    app_data = os.environ.get("APPDATA", "")
    return os.path.join(app_data, "Antigravity", "app_storage.json")


def pre_patch_tos_acceptance(
    target_email: str,
    storage_path: Optional[str] = None,
) -> bool:
    """
    Layer 1 Defense: Pre-patches app_storage.json to mark GCP ToS as accepted
    for target_email, preventing the modal dialog from appearing.
    Sets jetski.onboarding.lastLoginUsername to target_email and
    jetski.onboarding.lastLoginIsGcpTos to literal string "false".
    Uses atomic replace to prevent corrupting storage during concurrent IDE access.
    """
    if not target_email or not isinstance(target_email, str):
        return False

    if storage_path is not None and not str(storage_path).strip():
        return False

    path = storage_path or get_app_storage_path()
    if not path:
        return False

    # If default path is used and does not exist on disk, return False safely
    if storage_path is None and not os.path.exists(path):
        return False

    try:
        data = {}
        if os.path.exists(path):
            try:
                with open(path, "r", encoding="utf-8") as f:
                    data = json.load(f)
            except Exception:
                data = {}

        data["jetski.onboarding.lastLoginUsername"] = target_email
        data["jetski.onboarding.lastLoginIsGcpTos"] = "false"  # Must be string literal "false" per schema

        dir_name = os.path.dirname(path)
        if dir_name:
            os.makedirs(dir_name, exist_ok=True)

        with tempfile.NamedTemporaryFile("w", dir=dir_name if dir_name else None, delete=False, encoding="utf-8") as tf:
            json.dump(data, tf, indent=2, ensure_ascii=False)
            temp_path = tf.name

        os.replace(temp_path, path)
        return True
    except Exception as exc:
        logger.debug("pre_patch_tos_acceptance error: %s", exc)
        return False


def ensure_antigravity_foreground(
    window_finder: Optional[Callable[[], List[wintypes.HWND]]] = None,
    foreground_setter: Optional[Callable[[wintypes.HWND], bool]] = None,
    sleep_fn: Optional[Callable[[float], None]] = None,
) -> bool:
    """
    Ensures the Antigravity main window is active and in foreground.
    """
    finder = window_finder or find_antigravity_windows
    setter = foreground_setter or SetForegroundWindow
    sleeper = sleep_fn or time.sleep

    try:
        hwnds = finder()
        if not hwnds:
            return False
        setter(hwnds[0])
        sleeper(0.1)
        return True
    except Exception as exc:
        logger.debug("ensure_antigravity_foreground error: %s", exc)
        return False


def recycle_language_server(
    wait_seconds: float = 5.0,
    process_killer: Optional[Callable[..., Any]] = None,
    timeout: float = 5.0,
    sleep_fn: Optional[Callable[[float], None]] = None,
) -> bool:
    """
    Step 2: Terminates language_server.exe cleanly and waits full wait_seconds (default 5.0s).
    Do NOT press any keys during this wait window.
    Antigravity's monitorLsCrashInternal automatically respawns a fresh instance within 2000ms,
    loads credentials via CredReadW into RAM, and allows the Electron DOM ToS modal to completely appear on screen.
    """
    sleeper = sleep_fn or time.sleep
    try:
        if process_killer is not None:
            res = process_killer()
            if wait_seconds > 0:
                sleeper(wait_seconds)
            return bool(res) if res is not None else True

        creationflags = 0
        if hasattr(subprocess, "CREATE_NO_WINDOW"):
            creationflags = subprocess.CREATE_NO_WINDOW

        subprocess.run(
            ["taskkill", "/F", "/IM", "language_server.exe"],
            capture_output=True,
            timeout=timeout,
            creationflags=creationflags,
            check=False,
        )
        if wait_seconds > 0:
            sleeper(wait_seconds)
        return True
    except Exception as exc:
        logger.debug("recycle_language_server error: %s", exc)
        return False


def dismiss_tos_modal(
    key_sender: Optional[Callable[[], None]] = None,
    sleep_fn: Optional[Callable[[float], None]] = None,
) -> bool:
    """
    Step 3: Dispatches VK_RETURN (Enter) key event to accept/dismiss the rendered Google ToS modal dialog.
    Since Google ToS is an Electron DOM modal inside Antigravity's main window,
    dispatching Enter cleanly accepts and dismisses the modal without brittle window title polling.
    """
    if key_sender is not None:
        try:
            key_sender()
            return True
        except Exception as exc:
            logger.debug("key_sender error: %s", exc)
            return False

    sleeper = sleep_fn or time.sleep
    try:
        keybd_event(VK_RETURN, 0, 0, None)
        sleeper(0.05)
        keybd_event(VK_RETURN, 0, KEYEVENTF_KEYUP, None)
        return True
    except Exception as exc:
        logger.debug("dismiss_tos_modal error: %s", exc)
        return False


# Semantic and backward-compatible aliases
dismiss_tos_modal_hardware_fallback = dismiss_tos_modal
accept_tos_modal = dismiss_tos_modal


def is_tos_modal_present(
    modal_finder: Optional[Callable[[], bool]] = None,
) -> bool:
    """
    Checks if an Antigravity Terms of Service modal dialog is visible.
    Note: In Antigravity, Google ToS is an Electron DOM modal inside the main window,
    NOT a separate Win32 window.
    """
    if modal_finder is not None:
        try:
            return bool(modal_finder())
        except Exception:
            return False

    try:
        hwnd = user32.GetForegroundWindow()
        if not hwnd:
            return False
        buf = ctypes.create_unicode_buffer(512)
        GetWindowTextW(hwnd, buf, 512)
        title = buf.value.lower()
        if any(w in title for w in ["terms of service", "google cloud code", "điều khoản", "terms"]):
            return True
        return False
    except Exception:
        return False


def wait_and_dismiss_tos_modal(
    timeout_seconds: float = 4.5,
    poll_interval: float = 0.25,
    modal_finder: Optional[Callable[[], bool]] = None,
    key_sender: Optional[Callable[[], None]] = None,
    fallback_dismisser: Optional[Callable[[], None]] = None,
    sleep_fn: Optional[Callable[[float], None]] = None,
    clock_fn: Optional[Callable[[], float]] = None,
    fallback_at_peak: bool = False,
) -> bool:
    """
    Dismissal helper for Antigravity ToS modal.
    If a custom modal_finder is provided (e.g. In unit tests), polls the finder.
    Otherwise dispatches deterministic dismissal via VK_RETURN without brittle window title polling.
    """
    dismisser_func = fallback_dismisser or (lambda: dismiss_tos_modal(key_sender=key_sender, sleep_fn=sleep_fn))

    if modal_finder is not None:
        sleeper = sleep_fn or time.sleep
        clock = clock_fn or time.time
        start_time = clock()
        max_polls = int(timeout_seconds / poll_interval) if poll_interval > 0 else 1
        polls = 0

        while (clock() - start_time) < timeout_seconds and polls < max_polls:
            if is_tos_modal_present(modal_finder=modal_finder):
                try:
                    dismisser_func()
                except Exception as exc:
                    logger.debug("Error in ToS modal dismissal: %s", exc)
                return True
            polls += 1
            sleeper(poll_interval)

        if fallback_at_peak:
            try:
                dismisser_func()
                return True
            except Exception as exc:
                logger.debug("Error in peak fallback ToS modal dismissal: %s", exc)

        return False

    # Pure deterministic dismissal when no synthetic modal finder is supplied
    try:
        dismisser_func()
        return True
    except Exception as exc:
        logger.debug("Error in direct ToS modal dismissal: %s", exc)
        return False


def _set_clipboard_text(text: str) -> bool:
    """Sets text into Windows clipboard using Win32 API, with Tkinter fallback."""
    try:
        kernel32 = ctypes.WinDLL("kernel32", use_last_error=True)
        if user32.OpenClipboard(None):
            try:
                user32.EmptyClipboard()
                encoded = (text + "\0").encode("utf-16le")
                # GMEM_MOVEABLE = 0x0002
                h_mem = kernel32.GlobalAlloc(0x0002, len(encoded))
                if h_mem:
                    p_mem = kernel32.GlobalLock(h_mem)
                    if p_mem:
                        ctypes.memmove(p_mem, encoded, len(encoded))
                        kernel32.GlobalUnlock(h_mem)
                        # CF_UNICODETEXT = 13
                        user32.SetClipboardData(13, h_mem)
                return True
            finally:
                user32.CloseClipboard()
    except Exception:
        pass

    try:
        import tkinter as tk
        r = tk.Tk()
        r.withdraw()
        r.clipboard_clear()
        r.clipboard_append(text)
        r.update()
        r.destroy()
        return True
    except Exception:
        return False


def inject_continuation_prompt(
    prompt_text: str = "continue",
    window_finder: Optional[Callable[[], List[wintypes.HWND]]] = None,
    foreground_setter: Optional[Callable[[wintypes.HWND], bool]] = None,
    key_sender: Optional[Callable[..., Any]] = None,
    sleep_fn: Optional[Callable[[float], None]] = None,
) -> bool:
    """
    Preserves in-flight workflow by focusing the active chat tab (Ctrl+L)
    and injecting continuation prompt + Enter.
    Ensures SetForegroundWindow is called, applies 0.5s focus settling delay after Ctrl+L,
    injects "continue", waits 0.25s, then triggers VK_RETURN.
    """
    sleeper = sleep_fn or time.sleep
    if key_sender is not None:
        try:
            return bool(key_sender(prompt_text))
        except Exception:
            return False

    finder = window_finder or find_antigravity_windows
    try:
        hwnds = finder()
    except Exception:
        hwnds = []

    if not hwnds:
        return False

    setter = foreground_setter or SetForegroundWindow
    target_hwnd = hwnds[0]
    try:
        setter(target_hwnd)
        sleeper(0.1)

        # 1. Synthesize Ctrl+L to focus chat input
        keybd_event(VK_CONTROL, 0, 0, None)
        keybd_event(VK_L, 0, 0, None)
        sleeper(0.05)
        keybd_event(VK_L, 0, KEYEVENTF_KEYUP, None)
        keybd_event(VK_CONTROL, 0, KEYEVENTF_KEYUP, None)

        sleeper(0.5)  # Focus settling delay 0.5s after Ctrl+L

        # 2. Inject text using clipboard paste (Ctrl+V) for instant high-speed Unicode injection
        _set_clipboard_text(prompt_text)

        keybd_event(VK_CONTROL, 0, 0, None)
        keybd_event(VK_V, 0, 0, None)
        sleeper(0.05)
        keybd_event(VK_V, 0, KEYEVENTF_KEYUP, None)
        keybd_event(VK_CONTROL, 0, KEYEVENTF_KEYUP, None)

        sleeper(0.25)  # Settling delay 0.25s before Enter

        # 3. Press Enter to submit prompt
        keybd_event(VK_RETURN, 0, 0, None)
        sleeper(0.05)
        keybd_event(VK_RETURN, 0, KEYEVENTF_KEYUP, None)
        return True
    except Exception as exc:
        logger.debug("inject_continuation_prompt error: %s", exc)
        return False


def discover_most_recent_cascade_id(
    brain_dir: Optional[str] = None,
    storage_path: Optional[str] = None,
) -> Optional[str]:
    """
    Discovers the most recent active main conversation ID (cascade_id).
    Scans %USERPROFILE%/.gemini/antigravity/brain/ for subdirectories with transcript.jsonl,
    reads mtime, and cross-checks with app_storage.json (aux-pane-v2-session / layout keys)
    to target the active main user session.
    """
    b_dir = brain_dir
    if b_dir is None:
        user_profile = os.environ.get("USERPROFILE") or os.path.expanduser("~")
        b_dir = os.path.join(user_profile, ".gemini", "antigravity", "brain")

    if not b_dir or not os.path.isdir(b_dir):
        return None

    # Step 1: Scan brain_dir for subdirectories containing transcript.jsonl
    candidates: List[tuple[str, float]] = []
    try:
        for entry in os.listdir(b_dir):
            entry_path = os.path.join(b_dir, entry)
            if not os.path.isdir(entry_path):
                continue

            t1 = os.path.join(entry_path, "transcript.jsonl")
            t2 = os.path.join(entry_path, ".system_generated", "logs", "transcript.jsonl")

            mtime: Optional[float] = None
            if os.path.isfile(t1):
                try:
                    mtime = os.path.getmtime(t1)
                except Exception:
                    pass
            elif os.path.isfile(t2):
                try:
                    mtime = os.path.getmtime(t2)
                except Exception:
                    pass

            if mtime is not None:
                candidates.append((entry, mtime))
    except Exception as exc:
        logger.debug("discover_most_recent_cascade_id error scanning brain: %s", exc)
        return None

    if not candidates:
        return None

    # Step 2: Extract active main conversation IDs from app_storage.json
    s_path = storage_path if storage_path is not None else get_app_storage_path()
    main_panes: List[str] = []
    layout_ids: List[str] = []

    if s_path and os.path.isfile(s_path):
        try:
            with open(s_path, "r", encoding="utf-8") as f:
                sdata = json.load(f)
            if isinstance(sdata, dict):
                # Check aux-pane-v2-session for main conversationPanes
                aux = sdata.get("aux-pane-v2-session")
                if isinstance(aux, str):
                    try:
                        aux = json.loads(aux)
                    except Exception:
                        aux = None
                if isinstance(aux, dict):
                    panes = aux.get("conversationPanes")
                    if isinstance(panes, dict):
                        main_panes = [p for p in panes.keys() if p and isinstance(p, str)]

                # Check layout keys
                for k in sdata.keys():
                    if k.startswith("antigravity-multi-conversation-layout-v3-"):
                        cid = k[len("antigravity-multi-conversation-layout-v3-"):]
                        if cid.endswith("+_new"):
                            cid = cid[:-5]
                        if cid and cid != "index":
                            layout_ids.append(cid)
        except Exception as exc:
            logger.debug("discover_most_recent_cascade_id error reading app_storage: %s", exc)

    # Step 3: Cross-check candidates with app_storage.json
    # Priority 1: Match against main conversationPanes
    if main_panes:
        matching_panes = [c for c in candidates if c[0] in main_panes]
        if matching_panes:
            matching_panes.sort(key=lambda x: x[1], reverse=True)
            return matching_panes[0][0]

    # Priority 2: Match against layout keys
    if layout_ids:
        matching_layouts = [c for c in candidates if c[0] in layout_ids]
        if matching_layouts:
            matching_layouts.sort(key=lambda x: x[1], reverse=True)
            return matching_layouts[0][0]

    # Priority 3: Fallback to most recent candidate in brain directory
    candidates.sort(key=lambda x: x[1], reverse=True)
    return candidates[0][0]

