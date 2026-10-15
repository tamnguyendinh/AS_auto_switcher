# Engineering Report: Send Cascade Model Configuration Protocol

- **Author**: Coder Specialist Lane
- **Date**: 2026-10-15
- **Task Contract**: Implement `discover_active_agent_model` and attach `cascadeConfig` (with `planModel` and `requestedModel`) in `send_user_cascade_message` to eliminate the executor error `plan model not specified; you must specify a valid model`; integrate with `AccountManager.switch_account`; add comprehensive unit tests; and verify 100% test pass rate.

---

## 1. Executive Summary

All contract deliverables have been implemented and validated:
1. **Model Discovery (`discover_active_agent_model`)**:
   - Implemented in [core/api_connector.py](file:///g:/antigravity_auto_switch/core/api_connector.py).
   - Priority 1: Reads `last_selected_agent_model` from `pbtxt_path` (or default `~/.gemini/antigravity/antigravity_state.pbtxt`).
   - Priority 2: Queries `/exa.language_server_pb.LanguageServerService/GetCascadeModelConfigData` Connect-RPC endpoint on `LanguageServerInfo` if available.
   - Priority 3: Fallback default to `"MODEL_PLACEHOLDER_M318"`.
2. **Cascade Config Payload (`send_user_cascade_message`)**:
   - In [core/api_connector.py](file:///g:/antigravity_auto_switch/core/api_connector.py), `send_user_cascade_message` now accepts `model_name` and `cascade_config`.
   - When `cascade_config` is None, it constructs:
     ```json
     {
       "plannerConfig": {
         "planModel": model,
         "requestedModel": { "model": model }
       }
     }
     ```
   - Dispatches `"cascadeConfig": cascade_config` in the Connect-RPC POST body, eliminating the `failed to construct executor: plan model not specified; you must specify a valid model` error.
3. **State Resolver Integration (`AccountManager.switch_account`)**:
   - In [core/state_resolver.py](file:///g:/antigravity_auto_switch/core/state_resolver.py), `AccountManager` accepts optional `model_name` and `cascade_config`.
   - `switch_account` passes `model_name` and `cascade_config` dynamically via signature inspection or relies on `send_user_cascade_message`'s automatic active model discovery, ensuring backwards compatibility with custom mock senders.
4. **Comprehensive Unit Testing & Zero Regression**:
   - Added unit tests in [tests/test_hotswap_continuation.py](file:///g:/antigravity_auto_switch/tests/test_hotswap_continuation.py) and [tests/test_all.py](file:///g:/antigravity_auto_switch/tests/test_all.py).
   - 133 / 133 tests passed (100% pass rate).

---

## 2. Modified Slices & Implementation Details

### 2.1. `core/api_connector.py`
- Implemented `DEFAULT_ACTIVE_AGENT_MODEL = "MODEL_PLACEHOLDER_M318"`.
- Implemented `discover_active_agent_model(info: Optional[LanguageServerInfo] = None, pbtxt_path: Optional[str] = None) -> str`:
  - Parses `last_selected_agent_model: ...` from pbtxt (handling quoted and unquoted enum strings).
  - Queries `GetCascadeModelConfigData` via Connect-RPC with CSRF headers if info provided.
  - Falls back to `DEFAULT_ACTIVE_AGENT_MODEL`.
- Updated `send_user_cascade_message(info, cascade_id, prompt_text="continue", timeout=5.0, model_name=None, cascade_config=None) -> bool`:
  - Resolves model via `model_name` or `discover_active_agent_model(info=info)`.
  - Injects `cascadeConfig` payload with `plannerConfig.planModel` and `plannerConfig.requestedModel`.

### 2.2. `core/state_resolver.py`
- Imported `discover_active_agent_model` from `core.api_connector`.
- Added `model_name` and `cascade_config` options to `AccountManager.__init__` and `AccountManager.switch_account`.
- Inspects `sender` parameter signature using `inspect.signature` to safely forward `model_name` / `cascade_config` if supported, with automatic fallback for older mock signatures.

### 2.3. Test Suites
- In `tests/test_hotswap_continuation.py`:
  - `test_discover_active_agent_model_from_pbtxt`: Verified unquoted pbtxt parsing.
  - `test_discover_active_agent_model_from_pbtxt_with_quotes`: Verified quoted pbtxt parsing.
  - `test_discover_active_agent_model_from_rpc`: Verified `GetCascadeModelConfigData` RPC query and override model extraction.
  - `test_discover_active_agent_model_fallback_default`: Verified fallback to `MODEL_PLACEHOLDER_M318`.
  - `test_send_user_cascade_message_attaches_cascade_config`: Verified Connect-RPC payload contains `plannerConfig`, `planModel`, and `requestedModel`.
  - `test_send_user_cascade_message_custom_cascade_config`: Verified explicit `cascade_config` passing.
  - `test_switch_account_passes_model_name_and_cascade_config_to_sender`: Verified end-to-end `switch_account` configuration forwarding.
  - Updated existing `test_send_user_cascade_message_mock_success_and_custom_prompt` to assert presence of `cascadeConfig`.
- In `tests/test_all.py`:
  - Enhanced `MockHttpServerHelper` to record `last_body`.
  - Updated `test_send_user_cascade_message_mock_http_server` to verify `cascadeConfig` in raw loopback wire traffic.
  - Added `test_discover_active_agent_model_pbtxt_and_fallback`.

---

## 3. Test & Verification Results

### Test Suite Execution
- **Command**: `pytest tests/test_hotswap_continuation.py tests/test_all.py`
- **Collected**: 133 items
- **Passed**: 133 passed in 69.96s
- **Pass Rate**: 100%
- **Status**: PASSED
