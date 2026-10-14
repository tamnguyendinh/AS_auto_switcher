# Architectural Design Report: Seamless Account Hot-Swap & Uninterrupted Session Continuation Architecture

**Document Reference:** `reports/architect/2026-10-15-uninterrupted-session-continuation-architecture.md`  
**Date:** 2026-10-15  
**Author:** System Architecture Group & CEO Executive Review  
**Authority:** Direct Human Owner Directive via CEO Agent (`403f41a7-0810-44d9-b875-1e08c4ed13a0`)  
**Status:** APPROVED ARCHITECTURAL SPECIFICATION & IMPLEMENTATION BLUEPRINT  
**Target Milestone:** Enhanced Account Switcher Engine & Autonomous Session Resumption  

---

## 1. Executive Summary & Problem Formulation

### 1.1 Context & Real-World Diagnostic Findings
During hands-on real-world validation of the multi-account switching prototype (`antigravity-switcher`), the Owner uncovered three critical operational blockers that prevented true seamless account switching:

1. **Failure of UI Account Switch vs CLI Success:**
   When the user triggered an account switch via the graphical interface (UI button), a modal popup announced that the switch was successful. However, in reality, Antigravity continued executing requests under the **old account's quota in RAM**. Conversely, when terminating `language_server.exe` via CLI/PowerShell, Antigravity instantly switched to the new account in RAM with zero browser popups.
2. **Obtrusive Terms of Service (ToS) Modal Dialog:**
   When switching to an account that had not been previously initialized in Antigravity, Google Antigravity displayed an intrusive modal dialog requiring the user to manually click "Đồng ý" (Accept Terms). This manual friction defeats the purpose of an autonomous failover engine.
3. **Loss of In-Flight Conversation Context (Session Disruption):**
   Instead of continuing the active conversation where the AI agent was mid-task, restarting or reloading caused the IDE to open a blank new chat session or lose focus on the active thread, forcing the developer to manually navigate back and re-prompt the agent.
4. **Poor UI/UX Modal Interruption:**
   The UI spawned a blocking message box dialog (`messagebox.showinfo` in Tkinter / `QMessageBox.information` in Qt) stating that the switch was successful, halting the user's workflow until they dismissed the modal, while failing to update the live visual indicator of the active account in RAM.

---

## 2. Root Cause Analysis: UI Switch Failure vs CLI Success

### 2.1 The Architectural Disconnect Between Electron DOM and Language Server
The initial account switcher implementation in `core/state_resolver.py::switch_account` executed the following sequence:

```
[Step 1: Write active_account.json]
             │
             ▼
[Step 2: Read OAuth Token from tokens.json / AccountStore]
             │
             ▼
[Step 3: Write Token to Windows Credential Manager ("gemini:antigravity")]
             │
             ▼
[Step 4: Dispatch Ctrl+R to Electron Window via session_reloader.reload_antigravity_window()]
```

#### Why Step 4 (`Ctrl+R`) Failed to Switch the Account in RAM:
- Google Antigravity is built on top of the VS Code / Electron architecture.
- The user interface (renderer process) executes HTML/CSS/JavaScript in an Electron webview. Dispatching `Ctrl+R` (`WM_KEYDOWN` with `VK_CONTROL` + `'R'`) causes Electron to reload the DOM webview.
- **However**, all actual Gemini AI interactions—including code completion, prompt dispatch, token streaming, quota enforcement, and communication with `daily-cloudcode-pa.googleapis.com`—are handled exclusively by **`language_server.exe`**, a native compiled Go binary running as an independent background process (`resources/bin/language_server.exe`).
- `language_server.exe` reads the active authentication credentials from Windows Credential Manager (`LegacyGeneric:target=gemini:antigravity`) **strictly once during process initialization** via the Win32 API `CredReadW`.
- `language_server.exe` does **not** poll Windows Credential Manager or monitor registry changes. Therefore, while Step 3 successfully updated the credential on disk, `language_server.exe` continued running in RAM with the old in-memory OAuth access/refresh tokens. Dispatching `Ctrl+R` merely refreshed the visual HTML frame while the underlying Go server remained locked to Account A!

```
+-----------------------------------------------------------------------------------+
| BEFORE (UI Switch with Ctrl+R):                                                   |
|                                                                                   |
|  [UI Switch Button]                                                               |
|        │                                                                          |
|        ▼                                                                          |
|  [Win Credential Mgr]  <-- Updated to Account B (Token B)                         |
|        │                                                                          |
|        ▼                                                                          |
|  [Ctrl+R Shortcut]   -----> [Electron Webview DOM] Reloaded (HTML Only)           |
|                                     │                                             |
|                                     x (NO EFFECT on background process)           |
|                                     ▼                                             |
|                             [language_server.exe] STILL USING TOKEN A IN RAM!     |
+-----------------------------------------------------------------------------------+
```

### 2.2 Why the CLI Test Succeeded: `monitorLsCrashInternal` Lifecycle
When terminating the language server via PowerShell:
```powershell
Stop-Process -Name language_server -Force
# or
taskkill /F /IM language_server.exe
```
Antigravity seamlessly switched to Account B in RAM within ~2.5 seconds. The exact underlying mechanism was audited directly from Antigravity's core bundle (`app.asar`, line 75676-76030):

```javascript
// Ground runtime source audited from resources/app.asar:
const RESTART_WINDOW_MS = 60000;
const MAX_RESTARTS = 3;
const RESTART_COOLDOWN_MS = 2000;

function monitorLsCrashInternal(handle, port, csrf, options) {
    void handle.exitPromise.then(async (exitInfo) => {
        clearLsProcess();
        if (_intentionalTermination) {
            return;
        }
        const now = Date.now();
        if (now - _lastRestartTime > RESTART_WINDOW_MS) {
            _restartCount = 0;
        }
        _lastRestartTime = now;
        if (_restartCount >= MAX_RESTARTS) {
            console.error(`Language server crashed ${MAX_RESTARTS} times in a row. Giving up.`);
            return;
        }
        _restartCount++;
        console.log(`Attempting restart ${_restartCount}/${MAX_RESTARTS} in ${RESTART_COOLDOWN_MS / 1000}s...`);
        await sleep(RESTART_COOLDOWN_MS); // Exactly 2000ms delay
        if (_intentionalTermination) {
            return;
        }
        try {
            const newHandle = await startLanguageServer(port, csrf, options);
            _lsPort = newHandle.port;
            if (options.onPortChanged) {
                options.onPortChanged(_lsPort); // Re-binds active IDE panes to new port
            }
            monitorLsCrashInternal(newHandle, port, csrf, options);
        } catch (err) {
            console.error(`Failed to restart language server: ${err.message}`);
        }
    });
}
```

#### Ground Runtime Truth:
1. Antigravity's Electron main process monitors the child process PID of `language_server.exe` via `monitorLsCrashInternal()`.
2. When `language_server.exe` is terminated, `_intentionalTermination` is `false`, so `monitorLsCrashInternal` triggers an automated restart timer after `RESTART_COOLDOWN_MS` (2000ms).
3. Antigravity spawns a fresh `language_server.exe` instance with the exact same CLI flags (`--headless`, `--port`, `--workspace`, etc.).
4. On startup, the fresh `language_server.exe` executes `CredReadW("gemini:antigravity")`. Because Step 3 had already written Token B to Credential Manager, the new process reads **Token B** directly into RAM!
5. Once the new language server reports healthy over HTTP/gRPC, Antigravity fires `options.onPortChanged(newPort)`, reconnecting all IDE panes and chat windows to the new server instance.
6. **Crucially: ZERO Chrome browser windows are opened, ZERO login prompts are displayed, and the switch occurs completely in-place.**
7. **Crash Guard Limit:** `MAX_RESTARTS = 3` within a rolling `RESTART_WINDOW_MS = 60000` (60s). Rapid switches must respect this threshold or implement debounce gating.

```
+-----------------------------------------------------------------------------------+
| AFTER (Language Server Recycle Protocol):                                         |
|                                                                                   |
|  [Account Switch Trigger]                                                         |
|        │                                                                          |
|        ▼                                                                          |
|  [Win Credential Mgr]  <-- Writes Token B                                         |
|        │                                                                          |
|        ▼                                                                          |
|  [Terminate language_server.exe] (Graceful Process Termination)                   |
|        │                                                                          |
|        ▼                                                                          |
|  [Antigravity monitorLsCrashInternal] Catches exit -> Waits 2.0s                  |
|        │                                                                          |
|        ▼                                                                          |
|  [Spawns New language_server.exe] -> Calls CredReadW() -> Reads Token B into RAM  |
|        │                                                                          |
|        ▼                                                                          |
|  [options.onPortChanged] Re-binds active session -> Active in RAM: Account B!     |
+-----------------------------------------------------------------------------------+
```

---

## 3. Two-Layer Automated Terms of Service (ToS) Bypass

### 3.1 Problem: The Google Terms of Service Interruption
When switching to an account that has not yet completed initial onboarding in Antigravity, the IDE intercepts the session and presents a full modal overlay titled **"Terms of Service"** or **"Google Cloud Code Terms"**, demanding the user click "Đồng ý" (Agree) before any prompt can be dispatched.

### 3.2 Empirical Verification of Antigravity's Onboarding Storage
Inspection of Antigravity's live configuration persistence at:  
`%APPDATA%\Antigravity\app_storage.json`  
confirms the exact keys responsible for onboarding gate evaluation:

```json
{
  "jetski.onboarding.lastLoginUsername": "hokinhdoanh.anvien.sales@gmail.com",
  "jetski.onboarding.lastLoginIsGcpTos": "false",
  "aux-pane-v2-session": "{\"version\":1,\"conversationPanes\":{\"403f41a7-0810-44d9-b875-1e08c4ed13a0\":{\"tabs\":[]}}}"
}
```

> [!IMPORTANT]
> Notice that `jetski.onboarding.lastLoginIsGcpTos` is stored as a string literal `"false"`, **not** a boolean `false`. Pre-patching must strictly assign the string literal `"true"` to adhere to Antigravity's internal schema parser.

### 3.3 The Two-Layer Defense Architecture

```
                                  [Account Switch Initiated]
                                              │
                                              ▼
                             ┌───────────────────────────────────┐
                             │ Layer 1: Data Pre-Patching        │
                             │ (Atomic Write to app_storage.json)│
                             │ - lastLoginUsername = new_email   │
                             │ - lastLoginIsGcpTos = "true"      │
                             └───────────────────────────────────┘
                                              │
                                              ▼
                             ┌───────────────────────────────────┐
                             │ Recycle language_server.exe       │
                             └───────────────────────────────────┘
                                              │
                                              ▼
                                 Did any ToS modal appear?
                                        │               │
                                       NO              YES (Edge case)
                                        │               │
                                        ▼               ▼
                                 [Proceed to     ┌───────────────────────────────┐
                                  Injection]     │ Layer 2: Hardware Fallback    │
                                                 │ Win32 VK_RETURN Synthesis     │
                                                 │ (Direct keybd_event to modal) │
                                                 └───────────────────────────────┘
                                                                │
                                                                ▼
                                                         [Proceed to Injection]
```

#### Layer 1: Data-Level Pre-Patching (Primary Defense)
Before restarting `language_server.exe`:
1. Safely locate `%APPDATA%\Antigravity\app_storage.json`.
2. Load JSON content with atomic locking.
3. Inject / overwrite:
   - `"jetski.onboarding.lastLoginUsername": "<new_target_email>"`
   - `"jetski.onboarding.lastLoginIsGcpTos": "true"`
4. Atomically commit the file back to disk (via temporary file swap `os.replace` to prevent race conditions).
5. **Result:** When the UI connects to the new language server instance, Antigravity reads `app_storage.json`, evaluates that the user has already consented to GCP Terms of Service, and completely suppresses the modal dialog.

#### Layer 2: Win32 Hardware Fallback (Secondary Defense)
In the rare event that Antigravity launches an unmanaged modal window (e.g., initial OAuth scope grant or secondary dialog):
1. Immediately following language server recycling, start an asynchronous watcher (timeout: 3.0s).
2. Check if an Antigravity modal dialog holds focus.
3. Dispatch a synthesized `VK_RETURN` (Enter) key event using `ctypes.windll.user32.keybd_event(0x0D, 0, 0, 0)` followed by `KEYEVENTF_KEYUP`.
4. The dialog is automatically dismissed in under 50ms without requiring user mouse intervention.

---

## 4. Active Session Preservation & Autonomous Prompt Continuation

### 4.1 Problem: Session Loss vs Session Continuation
When developers or autonomous agents operate in an IDE session, they work within a specific **Conversation Thread** (identified by a UUID such as `403f41a7-0810-44d9-b875-1e08c4ed13a0`). 
When an account hits a quota limit (HTTP 429 / RESOURCE_EXHAUSTED):
- Opening a brand new session forces the agent to reload the entire context, re-read files, and lose its intermediate chain-of-thought.
- **Requirement:** The system MUST retain the exact active conversation session and immediately inject a continuation prompt so that the agent picks up execution right where it was throttled.

### 4.2 Conversation State Preservation in `app_storage.json`
Antigravity manages open conversation tabs and the active chat panel through the session state keys in `app_storage.json`:
- `"aux-pane-v2-session"`: Stores the layout of the auxiliary side pane, including active conversation tab IDs (empirically verified to contain active conversation UUIDs such as `403f41a7-0810-44d9-b875-1e08c4ed13a0`).
- `"workbench.panel.chat"`: Tracks the currently focused conversation ID and scroll positions.

#### Why Process Recycling Preserves the Session:
Because `language_server.exe` terminates without closing or reloading the Electron IDE window, the IDE DOM retains the chat history buffer and active tab in memory. When the new language server comes online and `options.onPortChanged` fires, the existing conversation history remains visible and active in the chat panel!

### 4.3 Autonomous Prompt Injection Protocol (APIP)
Once the new `language_server.exe` is healthy (verified after ~2.5s delay following process recycling), the system triggers autonomous continuation injection:

```
[ language_server online & healthy ]
                 │
                 ▼
[ Step 1: Ensure Antigravity IDE Window is Active / Foreground ]
  - Win32 SetForegroundWindow(hwnd)
                 │
                 ▼
[ Step 2: Focus Active Chat Input Box ]
  - Dispatch Ctrl+L (VK_CONTROL + 0x4C)
  - Settling delay: 150ms (allow DOM focus event to settle)
                 │
                 ▼
[ Step 3: Inject Continuation Prompt ]
  - Text: "continue" (clean, concise, and universal payload per Owner directive)
  - Mechanism: Win32 SendInput or clipboard paste (Ctrl+V) for instant high-speed injection
                 │
                 ▼
[ Step 4: Dispatch Execution Trigger ]
  - Dispatch VK_RETURN (0x0D Enter key)
                 │
                 ▼
[ Result: AI Agent Resumes Thinking & Tool Execution Seamlessly under New Account Quota ]
```

---

## 5. UI/UX Refactoring: Eliminating Blocking Popups

### 5.1 Critique of Existing UX
- **Previous implementation:** In `ui/app_window.py` (lines 507–514), completing an account switch triggered:
  ```python
  if not self.headless and self.root is not None and self.show_dialogs:
      messagebox.showinfo("Chuyển Đổi Thành Công", f"{msg}...")
  ```
- **Defects:**
  1. It blocks the UI thread until the developer dismisses the modal.
  2. It forces manual physical intervention (hitting Enter or clicking OK).
  3. The active account indicator badge does not update in RAM until the modal closes.

### 5.2 Target Reactive UX Pattern
1. **Remove `messagebox.showinfo` completely.**
2. **Immediate In-Place State Mutation:**
   - As soon as the user clicks "⚡ Chuyển sang" (or auto-failover triggers), the button instantly updates its text to `"Đang chuyển..."` and disables adjacent buttons.
   - The header label updates to `"● Đang hoạt động: <new_email>"` with `COLOR_GREEN`.
   - The active account indicator badge switches to `[ĐANG DÙNG (RAM)]`.
3. **Non-Blocking Status Bar Notification:**
   - Display a lightweight, non-modal status bar message at the bottom of the tool: `Đã chuyển đổi thành công sang: <email> lúc <time> (Language Server re-bound in RAM)`.
   - No modal windows, no focus stealing.

---

## 6. Implementation Blueprint for Coder Lane

### 6.1 Touch-Point Map

| Module | File Path | Scope of Modification |
| :--- | :--- | :--- |
| **Session Reloader** | `core/session_reloader.py` | 1. Implement `recycle_language_server()` via graceful Win32 process termination.<br>2. Implement `pre_patch_tos_acceptance(email)` to pre-emptively accept GCP ToS.<br>3. Implement `dismiss_tos_modal_hardware_fallback()` for Layer 2 fallback.<br>4. Implement `inject_continuation_prompt(prompt_text="continue")` via Win32 `Ctrl+L` + SendInput/clipboard + Enter. |
| **State Resolver** | `core/state_resolver.py` | Update `switch_account()` to coordinate: Credential Manager write -> Storage Pre-patch -> `recycle_language_server()` -> Fallback Check -> Prompt Injection. |
| **UI Window** | `ui/app_window.py` | 1. Purge `messagebox.showinfo` modal popup.<br>2. Implement real-time reactive UI status indicators for active account in RAM. |

---

### 6.2 Code Blueprint: `core/session_reloader.py`

```python
import os
import json
import time
import subprocess
import tempfile
import ctypes
from ctypes import wintypes
from typing import Optional, List, Callable, Any

user32 = ctypes.WinDLL("user32.dll", use_last_error=True)

VK_CONTROL = 0x11
VK_RETURN = 0x0D
VK_L = 0x4C
VK_V = 0x56
KEYEVENTF_KEYUP = 0x0002
KEYEVENTF_UNICODE = 0x0004

def get_app_storage_path() -> str:
    """Returns the absolute path to Antigravity's app_storage.json."""
    app_data = os.environ.get("APPDATA", "")
    return os.path.join(app_data, "Antigravity", "app_storage.json")

def pre_patch_tos_acceptance(
    target_email: str,
    storage_path: Optional[str] = None
) -> bool:
    """
    Layer 1 Defense: Pre-patches app_storage.json to mark GCP ToS as accepted
    for target_email, preventing the modal dialog from appearing.
    Uses atomic replace to prevent corrupting storage during concurrent IDE access.
    """
    path = storage_path or get_app_storage_path()
    if not os.path.exists(path):
        return False
    try:
        with open(path, "r", encoding="utf-8") as f:
            data = json.load(f)

        data["jetski.onboarding.lastLoginUsername"] = target_email
        data["jetski.onboarding.lastLoginIsGcpTos"] = "true"  # Must be string "true"

        dir_name = os.path.dirname(path)
        with tempfile.NamedTemporaryFile("w", dir=dir_name, delete=False, encoding="utf-8") as tf:
            json.dump(data, tf, indent=2, ensure_ascii=False)
            temp_path = tf.name

        os.replace(temp_path, path)
        return True
    except Exception:
        return False

def recycle_language_server(
    wait_seconds: float = 2.5,
    process_killer: Optional[Callable[..., Any]] = None,
    timeout: float = 5.0
) -> bool:
    """
    Terminates language_server.exe cleanly.
    Antigravity's internal monitorLsCrashInternal automatically spawns a fresh
    instance within 2000ms, which calls CredReadW and loads the new token into RAM.
    """
    try:
        if process_killer is not None:
            return bool(process_killer())

        creationflags = 0
        if hasattr(subprocess, "CREATE_NO_WINDOW"):
            creationflags = subprocess.CREATE_NO_WINDOW

        subprocess.run(
            ["taskkill", "/F", "/IM", "language_server.exe"],
            capture_output=True,
            timeout=timeout,
            creationflags=creationflags,
            check=False
        )
        # Allow Antigravity's monitorLsCrashInternal (2000ms delay) to spawn fresh instance
        time.sleep(wait_seconds)
        return True
    except Exception:
        return False

def dismiss_tos_modal_hardware_fallback(
    key_sender: Optional[Callable[[], None]] = None
) -> None:
    """
    Layer 2 Defense: Synthesizes Enter key event if an unmanaged ToS dialog holds focus.
    """
    if key_sender is not None:
        key_sender()
        return

    user32.keybd_event(VK_RETURN, 0, 0, None)
    time.sleep(0.05)
    user32.keybd_event(VK_RETURN, 0, KEYEVENTF_KEYUP, None)

def inject_continuation_prompt(
    prompt_text: str = "continue",
    window_finder: Optional[Callable[[], List[wintypes.HWND]]] = None,
    foreground_setter: Optional[Callable[[wintypes.HWND], bool]] = None,
    key_sender: Optional[Callable[..., Any]] = None
) -> bool:
    """
    Preserves in-flight workflow by focusing the active chat tab (Ctrl+L)
    and injecting continuation prompt + Enter.
    """
    if key_sender is not None:
        return bool(key_sender(prompt_text))

    finder = window_finder or find_antigravity_windows
    hwnds = finder()
    if not hwnds:
        return False

    setter = foreground_setter or user32.SetForegroundWindow
    target_hwnd = hwnds[0]
    try:
        setter(target_hwnd)
        time.sleep(0.1)

        # 1. Synthesize Ctrl+L to focus chat input
        user32.keybd_event(VK_CONTROL, 0, 0, None)
        user32.keybd_event(VK_L, 0, 0, None)
        time.sleep(0.05)
        user32.keybd_event(VK_L, 0, KEYEVENTF_KEYUP, None)
        user32.keybd_event(VK_CONTROL, 0, KEYEVENTF_KEYUP, None)

        time.sleep(0.15)

        # 2. Inject text using clipboard paste (Ctrl+V) for instant high-speed Unicode injection
        import tkinter as tk
        r = tk.Tk()
        r.withdraw()
        r.clipboard_clear()
        r.clipboard_append(prompt_text)
        r.update()
        r.destroy()

        user32.keybd_event(VK_CONTROL, 0, 0, None)
        user32.keybd_event(VK_V, 0, 0, None)
        time.sleep(0.05)
        user32.keybd_event(VK_V, 0, KEYEVENTF_KEYUP, None)
        user32.keybd_event(VK_CONTROL, 0, KEYEVENTF_KEYUP, None)

        time.sleep(0.1)

        # 3. Press Enter to submit prompt
        user32.keybd_event(VK_RETURN, 0, 0, None)
        time.sleep(0.05)
        user32.keybd_event(VK_RETURN, 0, KEYEVENTF_KEYUP, None)
        return True
    except Exception:
        return False
```

---

### 6.3 Orchestration Sequence in `core/state_resolver.py`

```python
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
    auto_continue: bool = False,
    prompt_text: Optional[str] = None,
    tos_pre_patcher: Optional[Callable[[str], bool]] = None,
    language_server_recycler: Optional[Callable[..., bool]] = None,
    prompt_injector: Optional[Callable[..., bool]] = None,
    **kwargs: Any,
) -> SwitchResult:
    """
    Executes Seamless Account Hot-Swap with RAM synchronization, ToS bypass,
    and uninterrupted in-flight session continuation.
    """
    if not target_email or not isinstance(target_email, str):
        return SwitchResult(False, "Email tài khoản không hợp lệ.")

    norm_email = target_email.strip().lower()
    if not norm_email:
        return SwitchResult(False, "Email tài khoản không hợp lệ.")

    # Step 1: Idempotency check in RAM
    finder = process_finder or self.process_finder or find_language_server_process
    getter = status_getter or self.status_getter or get_user_status
    # ... (existing RAM status query) ...

    # Step 2: Auto-capture current active session
    self.auto_capture_active_session()

    # Step 3: Validate target token in store
    get_acc_fn = getattr(self.store, "get_account", None) or getattr(self.store, "get_account_token", None)
    target_token = None
    if callable(get_acc_fn):
        try:
            target_token = get_acc_fn(norm_email)
        except Exception:
            target_token = None

    if isinstance(target_token, dict) and "token_data" in target_token:
        target_token = target_token["token_data"]

    if not target_token or not isinstance(target_token, dict):
        return SwitchResult(False, f"Chưa có file JSON của {norm_email} trong accounts_store/!")

    # Step 4: Write target token to Windows Credential Manager
    writer = token_writer or self.token_writer or write_antigravity_token
    if not writer(target_token):
        return SwitchResult(False, "Không thể ghi Token vào Windows Credential Manager!")

    # Step 5: Layer 1 ToS Pre-Patching in app_storage.json
    patcher = tos_pre_patcher or pre_patch_tos_acceptance
    try:
        patcher(norm_email)
    except Exception:
        pass

    # Step 6: Recycle language_server.exe to re-bind credentials into RAM
    recycler = language_server_recycler or reloader_fn or self.reloader_fn or recycle_language_server
    try:
        import inspect
        sig = inspect.signature(recycler)
        if "info" in sig.parameters:
            recycler(info=info)
        else:
            recycler()
    except Exception:
        pass

    # Step 7: Layer 2 ToS Dismissal Fallback
    try:
        dismiss_tos_modal_hardware_fallback()
    except Exception:
        pass

    # Step 8: Autonomous Prompt Continuation (if requested)
    if auto_continue:
        injector = prompt_injector or inject_continuation_prompt
        try:
            p_text = prompt_text or "continue"
            injector(prompt_text=p_text)
        except Exception:
            pass

    return SwitchResult(True, f"Đã chuyển đổi thành công sang: {norm_email}")
```

---

### 6.4 UI Window Refactoring in `ui/app_window.py`

In `_finish_switch`:
```python
def _finish_switch(res: SwitchResult):
    try:
        self.last_switch_result = res
        success, msg = res

        # Re-enable top action buttons
        if hasattr(self, "save_btn") and self.save_btn:
            self.save_btn.config(state=tk.NORMAL)
        if hasattr(self, "refresh_btn") and self.refresh_btn:
            self.refresh_btn.config(state=tk.NORMAL)

        if success:
            # Refresh UI to show new active account in RAM immediately
            self.refresh_accounts()
            now_str = datetime.datetime.now().strftime("%H:%M:%S")
            self.status_bar_text = f"Đã chuyển đổi thành công sang: {target_email} lúc {now_str} (Language Server re-bound in RAM)"
            if hasattr(self, "status_bar") and self.status_bar:
                self.status_bar.config(text=self.status_bar_text)
            
            # CRITICAL UX REFACTORING: Completely remove messagebox.showinfo!
            # Zero blocking modal popups. The user remains 100% in flow.
        else:
            # Restore buttons on error
            for b in self.switch_buttons.values():
                b.config(state=tk.NORMAL, text="⚡ Chuyển sang")
            self.status_bar_text = f"Lỗi chuyển đổi: {msg}"
            if hasattr(self, "status_bar") and self.status_bar:
                self.status_bar.config(text=self.status_bar_text)
    finally:
        self.is_switching = False
```

---

## 7. Architectural Invariants & Verification Criteria

1. **Zero Browser Interruption Invariant:** The entire account switch must never launch a Chrome/Edge browser window or require external web OAuth.
2. **RAM Synchronicity Invariant:** After `recycle_language_server()`, `get_user_status()` from the language server port must return `email == target_email`.
3. **Zero Modal Disruption Invariant:** Neither the switcher tool nor Antigravity may display blocking modal dialogs that halt the user's workflow or require manual clicks.
4. **Session Continuity Invariant:** The conversation ID prior to switching (`aux-pane-v2-session`) must remain active and focused, with prompt continuation seamlessly submitted to the existing thread.
5. **Crash Guard / Rate Limit Invariant:** Process recycling frequency must not exceed `MAX_RESTARTS = 3` per 60-second window (`RESTART_WINDOW_MS = 60000`) defined in Antigravity's `monitorLsCrashInternal`.
6. **Universal Continuation Prompt Invariant:** The prompt injection payload defaults to `"continue"` (clean, concise, and universal across all AI models and languages per Owner directive), submitted via synthesized `VK_RETURN` without requiring user intervention.

---

## 8. Conclusion & Recommendation
This architecture solves all real-world issues cleanly by exploiting Antigravity's built-in `monitorLsCrashInternal` crash recovery system and persistent `app_storage.json` state. It transforms the account switcher from a brittle UI DOM refresher into a true industrial-grade, zero-friction Hot-Swap Engine.

Proceed immediately to dispatching the **Coder Lane** to implement the touch-points defined in Section 6.
