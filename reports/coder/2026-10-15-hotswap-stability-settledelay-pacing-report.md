# Hot-Swap Stability, Active Thread Filtering, Settle Delay & Pacing Report

- **Date:** 2026-10-15
- **Lane:** Coder Specialist
- **Authority:** Human Owner Directives & CEO Directive
- **Target Files:**
  - [`core/session_reloader.py`](file:///g:/antigravity_auto_switch/core/session_reloader.py)
  - [`core/state_resolver.py`](file:///g:/antigravity_auto_switch/core/state_resolver.py)
  - [`tests/test_hotswap_continuation.py`](file:///g:/antigravity_auto_switch/tests/test_hotswap_continuation.py)
  - [`tests/test_all.py`](file:///g:/antigravity_auto_switch/tests/test_all.py)

---

## 1. Summary of Changes

### A. Active Thread Filtering in `core/session_reloader.py`
- **Root Cause:** `capture_active_threads_snapshot` previously iterated over all keys in `sdata` matching `antigravity-multi-conversation-layout-v3-*` and `workbench.panel.cascade`. In long-running Antigravity instances, `app_storage.json` accumulates dead/closed conversation keys permanently, causing 39 historical sessions to be erroneously captured and continued upon hot-swap.
- **Refactoring:**
  - Strictly restricted active thread discovery to `conversationPanes` in `aux-pane-v2-session` (with fallback `aux-pane-session` and `conversationPanes`).
  - Completely purged stale historical layout key parsing (`antigravity-multi-conversation-layout-v3-*`) and generic panel keys.
  - Correctly extracts active panes and child tabs with `is_main_pane` resolution and project directory resolution.
  - Preserved atomic JSON snapshot writing to `accounts_store/active_threads_history/<YYYY-MM-DD-HH-MM-SS>_active_threads.json`.

### B. DOM Settle Delay (15.0s) & Thread Pacing (3.0s) in `core/state_resolver.py`
- **Default DOM Settle Delay:** Increased default stabilization wait from 3.0s to 15.0s (`settle = kwargs.get("dom_settle_delay", 15.0)`). This ensures the respawned `language_server.exe` process, CredReadW RAM authentication, and Electron DOM have completely settled before issuing RPC continuation commands.
- **Legacy Reloader Protection:** Maintained `dom_settle_delay=0.0` for legacy `reloader_fn` tests to keep them instantaneous.
- **Inter-Thread Pacing Delay:** In `AccountManager.switch_account` continuation loop over `active_threads_list`, added a deterministic 3.0s sleep delay (`s_fn(3.0)`) between subsequent threads (`threads_sent_count > 0`). This eliminates concurrency bottlenecks and prevents overloading the Language Server when sending messages across multiple project panes.

### C. English Continuation Prompt Directive
- Defined constant in `core/state_resolver.py`:
  ```python
  DEFAULT_CONTINUATION_PROMPT = (
      "Continue if you still have unfinished work in progress. "
      "If all tasks are already completed, reply with: Currently there is no work to do."
  )
  ```
- Configured `AccountManager.__init__` and `switch_account` to default to `DEFAULT_CONTINUATION_PROMPT`.

### D. Unit & Integration Test Suite Verification
- **`tests/test_hotswap_continuation.py`:**
  - Added test `test_capture_active_threads_snapshot_ignores_stale_layout_keys` asserting that historical layout keys (`antigravity-multi-conversation-layout-v3-*`, `workbench.panel.cascade`) are ignored and only active `conversationPanes` are captured.
  - Added test `test_switch_account_default_dom_settle_delay_15s` asserting `dom_settle_delay` defaults to 15.0s.
  - Added test `test_switch_account_default_continuation_prompt` verifying prompt constant and default attribute.
  - Updated `test_switch_account_multi_project_pure_native_rpc_continuation` asserting 3.0s pacing sleep delay is called between threads.
  - Updated existing continuation tests to assert `DEFAULT_CONTINUATION_PROMPT`.
- **`tests/test_all.py`:**
  - Added matching tests for 15.0s settle delay default, 3.0s inter-thread pacing delay, stale layout keys filtering, and `DEFAULT_CONTINUATION_PROMPT`.

---

## 2. Invariants Maintained

| Invariant | Status | Verification |
| :--- | :--- | :--- |
| Zero synthetic keyboard events in RPC continuation | PASS | Verified in all tests (`prompt_injector` never invoked) |
| Layer 1 GCP ToS bypass (`isGcpTos = "false"`) | PASS | Preserved and validated across all hot-swap tests |
| Atomic JSON thread snapshot persistence | PASS | Temporary file written then replaced (`os.replace`) |
| No anvien usage | PASS | Strict adherence to role boundaries |
| Test suite execution | PASS | 100% tests passing (123/123) |

---

## 3. Test Execution Evidence

```
============================= test session starts =============================
platform win32 -- Python 3.14.6, pytest-9.0.3, pluggy-1.6.0
rootdir: G:\antigravity_auto_switch
plugins: anyio-4.14.0
collected 123 items

tests\test_all.py ...................................................... [ 43%]
.........................                                                [ 64%]
tests\test_hotswap_continuation.py ..................................... [ 94%]
.......                                                                  [100%]

======================= 123 passed in 65.12s (0:01:05) ========================
```
