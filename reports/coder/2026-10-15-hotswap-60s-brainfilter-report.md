# Engineering Report: Seamless Hot-Swap 60s DOM Delay & Brain/FileView Filter

- **Author**: Coder Specialist Lane
- **Date**: 2026-10-15
- **Task Contract**: Increase `dom_settle_delay` fallback to 60.0s, filter non-conversation `fileView` tabs, exclude thread IDs without corresponding brain directories, and update unit test suites.

---

## 1. Executive Summary

All objectives of the assigned contract have been implemented and verified:
1. **DOM Settle Delay Increased to 60.0s**:
   - In `core/state_resolver.py`, the fallback stabilization delay in `AccountManager.switch_account` was changed from `15.0s` to `60.0s` (`settle = kwargs.get("dom_settle_delay", 60.0)`).
2. **Conversation Tab & Brain Directory Filtering**:
   - In `core/session_reloader.py`, `capture_active_threads_snapshot` now rejects non-conversation tabs (`fileView` tabs with `content.type == "fileView"` or `id` starting with `"file__"`).
   - In addition, thread IDs are strictly validated against `brain_dir`: threads without a corresponding directory on disk (`if b_dir and os.path.isdir(b_dir) and not os.path.isdir(os.path.join(b_dir, thread_id)): continue`) are excluded.
3. **Unit Test Updates & Zero-Regression Verification**:
   - `tests/test_hotswap_continuation.py` and `tests/test_all.py` were updated to create mock thread subdirectories in temporary brain directories.
   - Settle delay unit tests were updated to assert the `60.0s` default fallback delay using mocked `sleep_fn`.
   - Dedicated tests were added to assert rejection of `fileView` tabs (`id.startswith("file__")` and `content.type == "fileView"`) and missing brain directories.
   - Rapid test execution is maintained using mocked sleep providers.

---

## 2. Modified Slices & Implementation Details

### 2.1. `core/state_resolver.py`
- Line 627: Updated `settle = kwargs.get("dom_settle_delay", 60.0)`.

### 2.2. `core/session_reloader.py`
- Added helper `_is_file_view_tab(tab)` checking:
  - `tab_id.startswith("file__")`
  - `content.get("type") == "fileView"` (or `tab.get("content.type") == "fileView"` / `tab.get("type") == "fileView"`)
- In `capture_active_threads_snapshot`:
  - Filtered out panes or tabs matching `_is_file_view_tab` or starting with `"file__"`.
  - Added filter: `if b_dir and os.path.isdir(b_dir) and not os.path.isdir(os.path.join(b_dir, thread_id)): continue` for both dict and list conversation pane representations.

### 2.3. Test Suites (`tests/test_hotswap_continuation.py` & `tests/test_all.py`)
- Mock setups create thread subdirectories in temp brain dirs.
- Asserted `60.0 in sleep_durations` for the default `dom_settle_delay`.
- Added `test_capture_active_threads_snapshot_filters_fileview_and_missing_brain_dirs` verifying filtering of `fileView` tabs and non-existent brain subdirectories.

---

## 3. Test & Verification Results

### Test Suite Execution
- **Command**: `pytest tests/test_hotswap_continuation.py tests/test_all.py`
- **Collected**: 125 items
- **Passed**: 125 passed in 75.28s
- **Pass Rate**: 100%
- **Status**: PASSED
