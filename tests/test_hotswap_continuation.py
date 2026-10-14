"""
Unit tests for Seamless Account Hot-Swap, Two-Layer ToS Bypass,
and In-Flight Session Continuation Protocol.
"""

import unittest
import os
import json
import tempfile
import shutil
from unittest.mock import MagicMock, patch

from core.session_reloader import (
    get_app_storage_path,
    pre_patch_tos_acceptance,
    recycle_language_server,
    ensure_antigravity_foreground,
    is_tos_modal_present,
    dismiss_tos_modal,
    accept_tos_modal,
    dismiss_tos_modal_hardware_fallback,
    wait_and_dismiss_tos_modal,
    inject_continuation_prompt,
    reload_session,
    reload_antigravity_session,
)
from core.state_resolver import AccountManager, SwitchResult
from core.account_store import AccountStore
from core.api_connector import LanguageServerInfo
from ui.app_window import AppWindow


class TestHotSwapAndContinuation(unittest.TestCase):

    def setUp(self):
        self.test_dir = tempfile.mkdtemp()
        self.store_dir = os.path.join(self.test_dir, "accounts_store")
        os.makedirs(self.store_dir, exist_ok=True)
        self.store = AccountStore(self.store_dir)
        self.storage_file = os.path.join(self.test_dir, "app_storage.json")

        self.sleep_patcher = patch("time.sleep", return_value=None)
        self.mock_sleep = self.sleep_patcher.start()

    def tearDown(self):
        self.sleep_patcher.stop()
        if os.path.exists(self.test_dir):
            shutil.rmtree(self.test_dir, ignore_errors=True)

    # -------------------------------------------------------------------------
    # 1. Backward Compatibility
    # -------------------------------------------------------------------------
    def test_reload_session_alias_compatibility(self):
        """Verifies reload_session is backward-compatible alias of reload_antigravity_session."""
        self.assertIs(reload_session, reload_antigravity_session)

    # -------------------------------------------------------------------------
    # 2. Layer 1 ToS Pre-Patching (pre_patch_tos_acceptance)
    # -------------------------------------------------------------------------
    def test_pre_patch_tos_acceptance_creates_and_updates_storage(self):
        """Verifies pre_patch_tos_acceptance atomically updates app_storage.json with string literal 'true'."""
        target_email = "target.developer@gmail.com"

        # Initially write existing JSON with false ToS
        initial_data = {
            "jetski.onboarding.lastLoginUsername": "old.developer@gmail.com",
            "jetski.onboarding.lastLoginIsGcpTos": "false",
            "other_setting": 123
        }
        with open(self.storage_file, "w", encoding="utf-8") as f:
            json.dump(initial_data, f)

        res = pre_patch_tos_acceptance(target_email, storage_path=self.storage_file)
        self.assertTrue(res)

        with open(self.storage_file, "r", encoding="utf-8") as f:
            updated = json.load(f)

        self.assertEqual(updated["jetski.onboarding.lastLoginUsername"], target_email)
        # CRITICAL INVARIANT: must be string literal "true", NOT boolean True
        self.assertEqual(updated["jetski.onboarding.lastLoginIsGcpTos"], "true")
        self.assertIsInstance(updated["jetski.onboarding.lastLoginIsGcpTos"], str)
        self.assertEqual(updated["other_setting"], 123)

    def test_pre_patch_tos_acceptance_new_file(self):
        """Verifies pre_patch_tos_acceptance handles new/empty storage path."""
        target_email = "new.user@gmail.com"
        non_existent_path = os.path.join(self.test_dir, "nested", "app_storage.json")

        res = pre_patch_tos_acceptance(target_email, storage_path=non_existent_path)
        self.assertTrue(res)
        self.assertTrue(os.path.exists(non_existent_path))

        with open(non_existent_path, "r", encoding="utf-8") as f:
            data = json.load(f)

        self.assertEqual(data["jetski.onboarding.lastLoginUsername"], target_email)
        self.assertEqual(data["jetski.onboarding.lastLoginIsGcpTos"], "true")

    def test_pre_patch_tos_acceptance_invalid_inputs(self):
        """Verifies invalid email inputs return False safely."""
        self.assertFalse(pre_patch_tos_acceptance(""))
        self.assertFalse(pre_patch_tos_acceptance(None))
        self.assertFalse(pre_patch_tos_acceptance("valid@gmail.com", storage_path=""))

    def test_get_app_storage_path(self):
        """Verifies get_app_storage_path returns path ending with Antigravity\\app_storage.json."""
        path = get_app_storage_path()
        self.assertTrue(path.endswith(os.path.join("Antigravity", "app_storage.json")))

    # -------------------------------------------------------------------------
    # 3. Language Server Recycling (recycle_language_server)
    # -------------------------------------------------------------------------
    def test_recycle_language_server_with_process_killer_mock(self):
        """Verifies recycle_language_server terminates language_server and respects wait_seconds."""
        killer_called = []

        def mock_killer():
            killer_called.append(True)
            return True

        res = recycle_language_server(wait_seconds=0, process_killer=mock_killer)
        self.assertTrue(res)
        self.assertEqual(len(killer_killer := killer_called), 1)

    def test_recycle_language_server_default_wait_buffer_5s(self):
        """Verifies recycle_language_server defaults to 5.0s wait buffer."""
        killer_called = []

        def mock_killer():
            killer_called.append(True)
            return True

        res = recycle_language_server(process_killer=mock_killer)
        self.assertTrue(res)
        self.assertEqual(len(killer_called), 1)
        self.mock_sleep.assert_called_with(5.0)

    def test_recycle_language_server_subprocess_run(self):
        """Verifies recycle_language_server invokes taskkill when no custom killer provided."""
        with patch("core.session_reloader.subprocess.run") as mock_run:
            mock_run.return_value = MagicMock(returncode=0)
            res = recycle_language_server(wait_seconds=0)
            self.assertTrue(res)
            mock_run.assert_called_once()
            args = mock_run.call_args[0][0]
            self.assertIn("taskkill", args)
            self.assertIn("language_server.exe", args)

    def test_recycle_language_server_exception_resilience(self):
        """Verifies recycle_language_server handles exceptions gracefully returning False."""
        def blow_up():
            raise RuntimeError("Taskkill permission denied")

        res = recycle_language_server(wait_seconds=0, process_killer=blow_up)
        self.assertFalse(res)

    # -------------------------------------------------------------------------
    # 4. Layer 2 Hardware Fallback (dismiss_tos_modal_hardware_fallback)
    # -------------------------------------------------------------------------
    def test_dismiss_tos_modal_hardware_fallback_with_key_sender(self):
        """Verifies dismiss_tos_modal_hardware_fallback invokes custom key_sender."""
        invocations = []
        dismiss_tos_modal_hardware_fallback(key_sender=lambda: invocations.append("RETURN"))
        self.assertEqual(invocations, ["RETURN"])

    def test_dismiss_tos_modal_hardware_fallback_direct(self):
        """Verifies dismiss_tos_modal_hardware_fallback executes keybd_event safely."""
        with patch("core.session_reloader.keybd_event") as mock_keybd, \
             patch("core.session_reloader.time.sleep"):
            dismiss_tos_modal_hardware_fallback()
            # Expects key down (0) and key up (0x0002)
            self.assertEqual(mock_keybd.call_count, 2)
            self.assertEqual(mock_keybd.call_args_list[0][0][0], 0x0D) # VK_RETURN
            self.assertEqual(mock_keybd.call_args_list[1][0][0], 0x0D)

    def test_is_tos_modal_present(self):
        """Verifies is_tos_modal_present handles custom modal_finder."""
        self.assertTrue(is_tos_modal_present(modal_finder=lambda: True))
        self.assertFalse(is_tos_modal_present(modal_finder=lambda: False))

    def test_wait_and_dismiss_tos_modal_adaptive_polling_detected_early(self):
        """Verifies wait_and_dismiss_tos_modal detects modal on 2nd iteration and exits early."""
        responses = [False, True]

        def mock_finder():
            return responses.pop(0) if responses else True

        key_sender_called = []
        def mock_key_sender():
            key_sender_called.append("ENTER")

        sleep_calls = []
        def mock_sleeper(duration):
            sleep_calls.append(duration)

        res = wait_and_dismiss_tos_modal(
            timeout_seconds=4.5,
            poll_interval=0.25,
            modal_finder=mock_finder,
            key_sender=mock_key_sender,
            sleep_fn=mock_sleeper,
        )

        self.assertTrue(res)
        self.assertEqual(key_sender_called, ["ENTER"])
        self.assertEqual(sleep_calls, [0.25])  # Exited after 1 sleep (2nd poll detected)

    def test_wait_and_dismiss_tos_modal_timeout_not_detected(self):
        """Verifies wait_and_dismiss_tos_modal polls over full timeout without detecting modal."""
        key_sender_called = []
        sleep_calls = []

        res = wait_and_dismiss_tos_modal(
            timeout_seconds=4.5,
            poll_interval=0.25,
            modal_finder=lambda: False,
            key_sender=lambda: key_sender_called.append(True),
            sleep_fn=lambda d: sleep_calls.append(d),
            fallback_at_peak=False,
        )

        self.assertFalse(res)
        self.assertEqual(len(key_sender_called), 0)
        self.assertEqual(len(sleep_calls), 18)  # 4.5 / 0.25 = 18 polls
        self.assertTrue(all(d == 0.25 for d in sleep_calls))

    def test_wait_and_dismiss_tos_modal_fallback_at_peak(self):
        """Verifies wait_and_dismiss_tos_modal triggers fallback VK_RETURN at peak render time if not detected."""
        key_sender_called = []
        res = wait_and_dismiss_tos_modal(
            timeout_seconds=1.0,
            poll_interval=0.25,
            modal_finder=lambda: False,
            key_sender=lambda: key_sender_called.append("PEAK_FALLBACK"),
            sleep_fn=lambda d: None,
            fallback_at_peak=True,
        )

        self.assertTrue(res)
        self.assertEqual(key_sender_called, ["PEAK_FALLBACK"])

    # -------------------------------------------------------------------------
    # 5. Prompt Injection Protocol (inject_continuation_prompt)
    # -------------------------------------------------------------------------
    def test_inject_continuation_prompt_with_key_sender(self):
        """Verifies inject_continuation_prompt passes default prompt text 'continue' to key_sender."""
        received = []
        res = inject_continuation_prompt(key_sender=lambda p: received.append(p) or True)
        self.assertTrue(res)
        self.assertEqual(received, ["continue"])

    def test_inject_continuation_prompt_custom_text(self):
        """Verifies inject_continuation_prompt supports custom prompt text."""
        received = []
        res = inject_continuation_prompt(
            prompt_text="proceed with deployment",
            key_sender=lambda p: received.append(p) or True
        )
        self.assertTrue(res)
        self.assertEqual(received, ["proceed with deployment"])

    def test_inject_continuation_prompt_no_windows_found(self):
        """Verifies inject_continuation_prompt returns False safely when no window is found."""
        res = inject_continuation_prompt(window_finder=lambda: [])
        self.assertFalse(res)

    def test_inject_continuation_prompt_full_keystroke_sequence(self):
        """Verifies Ctrl+L -> Clipboard Paste -> VK_RETURN sequence when window is present."""
        with patch("core.session_reloader.keybd_event") as mock_keybd, \
             patch("core.session_reloader._set_clipboard_text") as mock_clip, \
             patch("core.session_reloader.time.sleep"):
            mock_hwnd = 12345
            res = inject_continuation_prompt(
                prompt_text="continue",
                window_finder=lambda: [mock_hwnd],
                foreground_setter=lambda hwnd: True,
            )
            self.assertTrue(res)
            mock_clip.assert_called_once_with("continue")
            # Verify synthesized key sequences (Ctrl+L, Ctrl+V, Return)
            self.assertTrue(mock_keybd.call_count >= 6)

    def test_inject_continuation_prompt_timing_parameters(self):
        """Verifies inject_continuation_prompt ensures SetForegroundWindow, 0.5s focus settling, and 0.25s pre-Enter delay."""
        mock_hwnd = 54321
        foreground_hwnds = []
        sleep_durations = []

        def mock_setter(hwnd):
            foreground_hwnds.append(hwnd)
            return True

        def mock_sleeper(duration):
            sleep_durations.append(duration)

        with patch("core.session_reloader.keybd_event"), \
             patch("core.session_reloader._set_clipboard_text") as mock_clip:
            res = inject_continuation_prompt(
                prompt_text="continue",
                window_finder=lambda: [mock_hwnd],
                foreground_setter=mock_setter,
                sleep_fn=mock_sleeper,
            )

            self.assertTrue(res)
            self.assertEqual(foreground_hwnds, [mock_hwnd])
            mock_clip.assert_called_once_with("continue")
            # Verify required timing delays: 0.1s foreground, 0.5s focus settling, 0.25s settling before Enter
            self.assertIn(0.1, sleep_durations)
            self.assertIn(0.5, sleep_durations)
            self.assertIn(0.25, sleep_durations)

    # -------------------------------------------------------------------------
    # 6. Full Hot-Swap Coordination Pipeline in AccountManager.switch_account
    # -------------------------------------------------------------------------
    def test_switch_account_full_seamless_hotswap_pipeline(self):
        """
        Tests complete seamless account hot-swap orchestration:
        Step 1: Write target token to Credential Manager ("gemini:antigravity")
        Step 2: Recycle language_server.exe (5.0s wait, no premature keys)
        Step 3: Dispatch VK_RETURN to accept/dismiss rendered ToS modal
        Step 4: Wait 3.0s stabilization delay for modal to close
        Step 5: Focus active chat (Ctrl+L), wait 0.5s, inject 'continue', wait 0.25s, Enter
        """
        active_email = "old.user@gmail.com"
        target_email = "new.user@gmail.com"
        target_token = {"access_token": "at_new", "id_token": "id_new"}
        self.store.store_account(target_email, target_token)

        pipeline_trace = []

        mock_info = LanguageServerInfo(pid=9999, csrf_token="csrf-9999", port=55555)

        def mock_finder():
            pipeline_trace.append("1_finder")
            return mock_info

        def mock_getter(info):
            pipeline_trace.append("2_getter")
            return {"email": active_email}

        def mock_writer(tok):
            pipeline_trace.append(("3_token_writer", tok.get("access_token")))
            return True

        def mock_patcher(em):
            pipeline_trace.append(("4_tos_pre_patcher", em))
            return True

        def mock_recycler(info=None):
            pipeline_trace.append("5_recycle_language_server")
            return True

        def mock_dismisser():
            pipeline_trace.append("6_tos_dismissal")

        def mock_prompt_injector(prompt_text="continue"):
            pipeline_trace.append(("7_prompt_injection", prompt_text))
            return True

        manager = AccountManager(
            store=self.store,
            process_finder=mock_finder,
            status_getter=mock_getter,
            token_writer=mock_writer,
            tos_pre_patcher=mock_patcher,
            language_server_recycler=mock_recycler,
            tos_dismisser=mock_dismisser,
            prompt_injector=mock_prompt_injector,
            auto_continue=True,
        )

        res = manager.switch_account(target_email)
        self.assertTrue(res)
        self.assertTrue(res.success)
        self.assertIn(target_email, res.message)

        # Validate exact Owner-verified linear execution order
        self.assertEqual(pipeline_trace[0], "1_finder")
        self.assertEqual(pipeline_trace[1], "2_getter")
        self.assertEqual(pipeline_trace[2], ("3_token_writer", "at_new"))
        self.assertEqual(pipeline_trace[3], ("4_tos_pre_patcher", target_email))
        self.assertEqual(pipeline_trace[4], "5_recycle_language_server")
        self.assertEqual(pipeline_trace[5], "6_tos_dismissal")
        self.assertEqual(pipeline_trace[6], ("7_prompt_injection", "continue"))

    def test_switch_account_auto_continue_false(self):
        """Verifies auto_continue=False suppresses prompt continuation injection."""
        target_email = "nocontinue@gmail.com"
        self.store.store_account(target_email, {"access_token": "at"})

        pipeline_trace = []
        manager = AccountManager(
            store=self.store,
            process_finder=lambda: None,
            token_writer=lambda tok: True,
            tos_pre_patcher=lambda em: True,
            language_server_recycler=lambda: True,
            tos_dismisser=lambda: None,
            prompt_injector=lambda prompt_text: pipeline_trace.append("injected"),
            auto_continue=True,
        )

        res = manager.switch_account(target_email, auto_continue=False)
        self.assertTrue(res)
        self.assertNotIn("injected", pipeline_trace)

    def test_switch_account_custom_prompt_text(self):
        """Verifies switch_account passes custom prompt_text to injector."""
        target_email = "custom_prompt@gmail.com"
        self.store.store_account(target_email, {"access_token": "at"})

        prompts_injected = []
        manager = AccountManager(
            store=self.store,
            process_finder=lambda: None,
            token_writer=lambda tok: True,
            tos_pre_patcher=lambda em: True,
            language_server_recycler=lambda: True,
            tos_dismisser=lambda: None,
            prompt_injector=lambda prompt_text: prompts_injected.append(prompt_text),
        )

        res = manager.switch_account(target_email, auto_continue=True, prompt_text="resume task 42")
        self.assertTrue(res)
        self.assertEqual(prompts_injected, ["resume task 42"])

    def test_switch_account_deterministic_tos_dismissal_without_polling(self):
        """Verifies ToS dismissal is dispatched deterministically without brittle window title polling."""
        target_email = "clean_tos@gmail.com"
        self.store.store_account(target_email, {"access_token": "at"})

        dismisser_called = []
        manager = AccountManager(
            store=self.store,
            process_finder=lambda: None,
            token_writer=lambda tok: True,
            tos_pre_patcher=lambda em: True,
            language_server_recycler=lambda: True,
            tos_dismisser=lambda: dismisser_called.append("DISMISSED"),
            prompt_injector=lambda prompt_text: True,
        )

        res = manager.switch_account(target_email)
        self.assertTrue(res)
        self.assertEqual(dismisser_called, ["DISMISSED"])

    def test_switch_account_modal_checker_false_bypasses_dismisser(self):
        """Alias verifying deterministic dismissal is invoked cleanly."""
        self.test_switch_account_deterministic_tos_dismissal_without_polling()

    def test_switch_account_owner_verified_sequencing_and_timing(self):
        """
        Verifies switch_account executes the Owner-verified sequential timing:
        Step 1: Write target token to Credential Manager
        Step 2: Recycle language_server.exe (5.0s buffer)
        Step 3: Dispatch VK_RETURN (Enter) to dismiss Google ToS modal dialog
        Step 4: Wait 3.0s stabilization delay for modal to disappear and session to settle
        Step 5: Focus active chat (Ctrl+L), wait 0.5s settling, inject 'continue', wait 0.25s, Enter
        """
        target_email = "ownersequence@gmail.com"
        target_token = {"access_token": "at_stages", "id_token": "id_stages"}
        self.store.store_account(target_email, target_token)

        stage_log = []

        def mock_writer(tok):
            stage_log.append("step_1_writer")
            return True

        def mock_recycler(info=None, wait_seconds=5.0, sleep_fn=None):
            stage_log.append(("step_2_recycle", wait_seconds))
            if sleep_fn:
                sleep_fn(wait_seconds)
            return True

        def mock_dismisser(sleep_fn=None):
            stage_log.append("step_3_tos_dismissal")
            return True

        def mock_sleep_fn(duration):
            stage_log.append(("sleep", duration))

        def mock_injector(prompt_text="continue", sleep_fn=None):
            stage_log.append(("step_5_inject", prompt_text))
            return True

        manager = AccountManager(
            store=self.store,
            process_finder=lambda: None,
            token_writer=mock_writer,
            tos_pre_patcher=lambda em: True,
            language_server_recycler=mock_recycler,
            tos_dismisser=mock_dismisser,
            prompt_injector=mock_injector,
            sleep_fn=mock_sleep_fn,
            auto_continue=True,
            dom_settle_delay=3.0,
        )

        res = manager.switch_account(target_email)
        self.assertTrue(res.success)

        # Verify Step 1 -> Step 2 (sleep 5.0) -> Step 3 -> Step 4 (sleep 3.0) -> Step 5
        self.assertEqual(stage_log[0], "step_1_writer")
        self.assertEqual(stage_log[1], ("step_2_recycle", 5.0))
        self.assertEqual(stage_log[2], ("sleep", 5.0))
        self.assertEqual(stage_log[3], "step_3_tos_dismissal")
        self.assertEqual(stage_log[4], ("sleep", 3.0))
        self.assertEqual(stage_log[5], ("step_5_inject", "continue"))

    def test_switch_account_4_stages_sequencing_and_timing(self):
        """Backward-compatible test name alias for Owner sequence test."""
        self.test_switch_account_owner_verified_sequencing_and_timing()

    # -------------------------------------------------------------------------
    # 7. Non-Blocking Reactive UI Feedback in AppWindow
    # -------------------------------------------------------------------------
    def test_app_window_switch_purges_blocking_messagebox(self):
        """Verifies messagebox.showinfo is NEVER called upon successful account switch."""
        target_email = "target.ui@gmail.com"
        self.store.store_account(target_email, {"access_token": "tok"})

        mock_manager = AccountManager(
            store=self.store,
            user_status_provider=lambda: target_email,
        )
        mock_manager.switch_account = lambda target: SwitchResult(True, f"Đã chuyển đổi thành công sang: {target}")

        app = AppWindow(manager=mock_manager, headless=True, show_dialogs=True)

        with patch("ui.app_window.messagebox.showinfo") as mock_info:
            res = app._on_switch_click(target_email)
            self.assertTrue(res.success)
            # CRITICAL VERIFICATION: Zero blocking popups!
            mock_info.assert_not_called()
            self.assertIn("Language Server re-bound in RAM", app.status_bar_text)
            self.assertIn(target_email, app.status_bar_text)


if __name__ == "__main__":
    unittest.main()
