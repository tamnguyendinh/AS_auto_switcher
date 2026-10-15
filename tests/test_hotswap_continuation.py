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
    discover_most_recent_cascade_id,
    capture_active_threads_snapshot,
)
from core.state_resolver import AccountManager, SwitchResult, DEFAULT_CONTINUATION_PROMPT
from core.account_store import AccountStore
from core.api_connector import (
    LanguageServerInfo,
    load_trajectory,
    smart_focus_conversation,
    send_user_cascade_message,
)
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
        """Verifies pre_patch_tos_acceptance atomically updates app_storage.json with string literal 'false'."""
        target_email = "target.developer@gmail.com"

        # Initially write existing JSON with other value
        initial_data = {
            "jetski.onboarding.lastLoginUsername": "old.developer@gmail.com",
            "jetski.onboarding.lastLoginIsGcpTos": "unaccepted",
            "other_setting": 123
        }
        with open(self.storage_file, "w", encoding="utf-8") as f:
            json.dump(initial_data, f)

        res = pre_patch_tos_acceptance(target_email, storage_path=self.storage_file)
        self.assertTrue(res)

        with open(self.storage_file, "r", encoding="utf-8") as f:
            updated = json.load(f)

        self.assertEqual(updated["jetski.onboarding.lastLoginUsername"], target_email)
        # CRITICAL INVARIANT: must be string literal "false", NOT boolean False or "true"
        self.assertEqual(updated["jetski.onboarding.lastLoginIsGcpTos"], "false")
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
        self.assertEqual(data["jetski.onboarding.lastLoginIsGcpTos"], "false")

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
        Step 1: Discover language server and check current active email
        Step 2: Snapshot active conversation threads across workspaces/projects
        Step 3: Write target token to Credential Manager ("gemini:antigravity")
        Step 4: Pre-patch ToS in app_storage.json (lastLoginIsGcpTos = "false")
        Step 5: Recycle language_server.exe (5.0s wait, no premature keys)
        Step 6: Bypass ToS dismissal Enter key press by default
        Step 7: Re-query finder() for fresh LanguageServerInfo (PID, port, CSRF token)
        Step 8: Pure native RPC continuation (smart_focus -> load_trajectory -> send_cascade_message)
                with ZERO Win32 keyboard emulation!
        """
        active_email = "old.user@gmail.com"
        target_email = "new.user@gmail.com"
        target_token = {"access_token": "at_new", "id_token": "id_new"}
        self.store.store_account(target_email, target_token)

        pipeline_trace = []

        old_info = LanguageServerInfo(pid=1111, csrf_token="csrf-1111", port=51111)
        fresh_info = LanguageServerInfo(pid=2222, csrf_token="csrf-2222", port=52222)
        finder_call_count = [0]

        def mock_finder():
            finder_call_count[0] += 1
            if finder_call_count[0] == 1:
                pipeline_trace.append(("1_finder_initial", old_info.pid))
                return old_info
            pipeline_trace.append(("6_finder_fresh_after_recycle", fresh_info.pid))
            return fresh_info

        def mock_getter(info):
            pipeline_trace.append(("2_getter", info.pid))
            return {"email": active_email}

        def mock_capturer():
            pipeline_trace.append("2_snapshot_threads")
            return ("/mock/path/snapshot.json", [
                {"thread_id": "thread-alpha", "project": "g:/projA", "is_main_pane": True}
            ])

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

        def mock_focuser(info, cid):
            pipeline_trace.append(("7_rpc_focus", info.pid, cid))
            return True

        def mock_loader(info, cid):
            pipeline_trace.append(("7_rpc_load", info.pid, cid))
            return True

        def mock_sender(info, cid, prompt_text="continue"):
            pipeline_trace.append(("7_rpc_send", info.pid, cid, prompt_text))
            return True

        mock_prompt_injector_called = []
        def mock_prompt_injector(prompt_text="continue"):
            mock_prompt_injector_called.append(prompt_text)
            return True

        manager = AccountManager(
            store=self.store,
            process_finder=mock_finder,
            status_getter=mock_getter,
            threads_snapshot_capturer=mock_capturer,
            token_writer=mock_writer,
            tos_pre_patcher=mock_patcher,
            language_server_recycler=mock_recycler,
            tos_dismisser=mock_dismisser,
            smart_focuser=mock_focuser,
            trajectory_loader=mock_loader,
            cascade_sender=mock_sender,
            prompt_injector=mock_prompt_injector,
            auto_continue=True,
            dom_settle_delay=0.0,
        )

        res = manager.switch_account(target_email)
        self.assertTrue(res)
        self.assertTrue(res.success)
        self.assertIn(target_email, res.message)

        # Validate exact Owner-verified linear execution order:
        self.assertEqual(pipeline_trace[0], ("1_finder_initial", 1111))
        self.assertEqual(pipeline_trace[1], ("2_getter", 1111))
        self.assertEqual(pipeline_trace[2], "2_snapshot_threads")
        self.assertEqual(pipeline_trace[3], ("3_token_writer", "at_new"))
        self.assertEqual(pipeline_trace[4], ("4_tos_pre_patcher", target_email))
        self.assertEqual(pipeline_trace[5], "5_recycle_language_server")
        self.assertEqual(pipeline_trace[6], ("6_finder_fresh_after_recycle", 2222))
        self.assertEqual(pipeline_trace[7], ("7_rpc_focus", 2222, "thread-alpha"))
        self.assertEqual(pipeline_trace[8], ("7_rpc_load", 2222, "thread-alpha"))
        self.assertEqual(pipeline_trace[9], ("7_rpc_send", 2222, "thread-alpha", DEFAULT_CONTINUATION_PROMPT))
        self.assertNotIn("6_tos_dismissal", pipeline_trace)

        # CRITICAL INVARIANT: Win32 keyboard emulation was strictly bypassed! Zero keyboard events!
        self.assertEqual(mock_prompt_injector_called, [])

        # Also verify that when dismiss_tos=True is explicitly requested, dismisser IS called
        pipeline_trace.clear()
        target_email2 = "another.user@gmail.com"
        self.store.store_account(target_email2, {"access_token": "at2"})
        res2 = manager.switch_account(target_email2, dismiss_tos=True)
        self.assertTrue(res2.success)
        self.assertIn("6_tos_dismissal", pipeline_trace)

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
            cascade_sender=lambda info, cid, prompt_text="continue": pipeline_trace.append("injected"),
            auto_continue=True,
        )

        res = manager.switch_account(target_email, auto_continue=False)
        self.assertTrue(res)
        self.assertNotIn("injected", pipeline_trace)

    def test_switch_account_custom_prompt_text(self):
        """Verifies switch_account passes custom prompt_text to RPC sender."""
        target_email = "custom_prompt@gmail.com"
        self.store.store_account(target_email, {"access_token": "at"})

        prompts_injected = []
        mock_info = LanguageServerInfo(pid=8888, csrf_token="token", port=50000)
        manager = AccountManager(
            store=self.store,
            process_finder=lambda: mock_info,
            token_writer=lambda tok: True,
            tos_pre_patcher=lambda em: True,
            language_server_recycler=lambda: True,
            tos_dismisser=lambda: None,
            threads_snapshot_capturer=lambda: ("/path", [{"thread_id": "c-custom", "project": "", "is_main_pane": True}]),
            smart_focuser=lambda info, cid: True,
            trajectory_loader=lambda info, cid: True,
            cascade_sender=lambda info, cid, prompt_text="continue": prompts_injected.append(prompt_text) or True,
            dom_settle_delay=0.0,
        )

        res = manager.switch_account(target_email, auto_continue=True, prompt_text="resume task 42")
        self.assertTrue(res)
        self.assertEqual(prompts_injected, ["resume task 42"])

    def test_switch_account_deterministic_tos_dismissal_without_polling(self):
        """Verifies ToS dismissal is bypassed by default and invoked cleanly when dismiss_tos=True."""
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

        # In standard switch: ToS modal is eliminated by pre-patching, so dismisser is NOT called
        res = manager.switch_account(target_email)
        self.assertTrue(res)
        self.assertEqual(dismisser_called, [])

        # In fallback mode (dismiss_tos=True): dismisser is called
        res_fallback = manager.switch_account(target_email, dismiss_tos=True)
        self.assertTrue(res_fallback)
        self.assertEqual(dismisser_called, ["DISMISSED"])

    def test_switch_account_modal_checker_false_bypasses_dismisser(self):
        """Alias verifying deterministic dismissal is bypassed in standard flow."""
        self.test_switch_account_deterministic_tos_dismissal_without_polling()

    def test_switch_account_owner_verified_sequencing_and_timing(self):
        """
        Verifies switch_account executes the Owner-verified sequential timing:
        Giai đoạn 1: Snapshot active threads before token swap
        Giai đoạn 2: Token swap and ToS pre-patching (isGcpTos = "false"), recycle language_server.exe (5.0s buffer), 3.0s stabilization delay
        Giai đoạn 3: Re-query fresh language server info and execute Connect-RPC continuation with zero keyboard events.
        """
        target_email = "ownersequence@gmail.com"
        target_token = {"access_token": "at_stages", "id_token": "id_stages"}
        self.store.store_account(target_email, target_token)

        stage_log = []

        def mock_capturer():
            stage_log.append("giai_doan_1_snapshot")
            return ("/path/history.json", [{"thread_id": "c-owner", "project": "proj-a", "is_main_pane": True}])

        def mock_writer(tok):
            stage_log.append("giai_doan_2_writer")
            return True

        def mock_recycler(info=None, wait_seconds=5.0, sleep_fn=None):
            stage_log.append(("giai_doan_2_recycle", wait_seconds))
            if sleep_fn:
                sleep_fn(wait_seconds)
            return True

        def mock_dismisser(sleep_fn=None):
            stage_log.append("tos_dismissal")
            return True

        def mock_sleep_fn(duration):
            stage_log.append(("sleep", duration))

        fresh_info = LanguageServerInfo(pid=9999, csrf_token="fresh_token", port=55555)

        def mock_focuser(info, cid):
            stage_log.append(("giai_doan_3_smart_focus", info.pid, cid))
            return True

        def mock_loader(info, cid):
            stage_log.append(("giai_doan_3_load_trajectory", info.pid, cid))
            return True

        def mock_sender(info, cid, prompt_text="continue"):
            stage_log.append(("giai_doan_3_send_message", info.pid, cid, prompt_text))
            return True

        keyboard_injector_called = []
        def mock_injector(prompt_text="continue", sleep_fn=None):
            keyboard_injector_called.append(prompt_text)
            return True

        manager = AccountManager(
            store=self.store,
            process_finder=lambda: fresh_info,
            token_writer=mock_writer,
            tos_pre_patcher=lambda em: True,
            language_server_recycler=mock_recycler,
            tos_dismisser=mock_dismisser,
            threads_snapshot_capturer=mock_capturer,
            smart_focuser=mock_focuser,
            trajectory_loader=mock_loader,
            cascade_sender=mock_sender,
            prompt_injector=mock_injector,
            sleep_fn=mock_sleep_fn,
            auto_continue=True,
            dom_settle_delay=3.0,
        )

        res = manager.switch_account(target_email)
        self.assertTrue(res.success)

        # Verify exact sequence across all 3 phases
        self.assertEqual(stage_log[0], "giai_doan_1_snapshot")
        self.assertEqual(stage_log[1], "giai_doan_2_writer")
        self.assertEqual(stage_log[2], ("giai_doan_2_recycle", 5.0))
        self.assertEqual(stage_log[3], ("sleep", 5.0))
        self.assertEqual(stage_log[4], ("sleep", 3.0))
        self.assertEqual(stage_log[5], ("giai_doan_3_smart_focus", 9999, "c-owner"))
        self.assertEqual(stage_log[6], ("giai_doan_3_load_trajectory", 9999, "c-owner"))
        self.assertEqual(stage_log[7], ("giai_doan_3_send_message", 9999, "c-owner", DEFAULT_CONTINUATION_PROMPT))

        # Invariant: ToS dismissal bypassed by default, keyboard emulation strictly purged (zero events)
        self.assertNotIn("tos_dismissal", stage_log)
        self.assertEqual(keyboard_injector_called, [])

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

    # -------------------------------------------------------------------------
    # 8. Thread ID Native Continuation Pipeline & Discovery Tests
    # -------------------------------------------------------------------------
    def test_discover_most_recent_cascade_id_nonexistent_and_empty(self):
        """Verifies discover_most_recent_cascade_id handles non-existent or empty brain directories."""
        self.assertIsNone(discover_most_recent_cascade_id(brain_dir=os.path.join(self.test_dir, "nonexistent")))

        empty_brain = os.path.join(self.test_dir, "empty_brain")
        os.makedirs(empty_brain, exist_ok=True)
        self.assertIsNone(discover_most_recent_cascade_id(brain_dir=empty_brain))

        # Subdirectory without transcript.jsonl should be ignored
        no_transcript_dir = os.path.join(empty_brain, "dummy-cascade-id")
        os.makedirs(no_transcript_dir, exist_ok=True)
        self.assertIsNone(discover_most_recent_cascade_id(brain_dir=empty_brain))

    def test_discover_most_recent_cascade_id_mtime_resolution(self):
        """Verifies mtime resolution selects the most recent transcript across root and .system_generated/logs."""
        brain_dir = os.path.join(self.test_dir, "mock_brain_1")
        c1 = os.path.join(brain_dir, "uuid-alpha")
        c2 = os.path.join(brain_dir, "uuid-beta", ".system_generated", "logs")
        c3 = os.path.join(brain_dir, "uuid-gamma")
        os.makedirs(c1, exist_ok=True)
        os.makedirs(c2, exist_ok=True)
        os.makedirs(c3, exist_ok=True)

        t1 = os.path.join(c1, "transcript.jsonl")
        t2 = os.path.join(c2, "transcript.jsonl")
        t3 = os.path.join(c3, "transcript.jsonl")

        with open(t1, "w", encoding="utf-8") as f:
            f.write("{}\n")
        with open(t2, "w", encoding="utf-8") as f:
            f.write("{}\n")
        with open(t3, "w", encoding="utf-8") as f:
            f.write("{}\n")

        os.utime(t1, (1000, 1000))
        os.utime(t2, (3000, 3000))  # Most recent
        os.utime(t3, (2000, 2000))

        discovered = discover_most_recent_cascade_id(brain_dir=brain_dir, storage_path=os.path.join(self.test_dir, "absent.json"))
        self.assertEqual(discovered, "uuid-beta")

    def test_discover_most_recent_cascade_id_cross_check_conversation_panes(self):
        """Verifies cross-checking prioritizes cascade IDs listed in aux-pane-v2-session conversationPanes."""
        brain_dir = os.path.join(self.test_dir, "mock_brain_2")
        c_subagent = os.path.join(brain_dir, "subagent-child-uuid")
        c_main_old = os.path.join(brain_dir, "main-thread-old")
        c_main_active = os.path.join(brain_dir, "main-thread-active")

        os.makedirs(c_subagent, exist_ok=True)
        os.makedirs(c_main_old, exist_ok=True)
        os.makedirs(c_main_active, exist_ok=True)

        t_sub = os.path.join(c_subagent, "transcript.jsonl")
        t_old = os.path.join(c_main_old, "transcript.jsonl")
        t_act = os.path.join(c_main_active, "transcript.jsonl")

        with open(t_sub, "w", encoding="utf-8") as f:
            f.write("{}\n")
        with open(t_old, "w", encoding="utf-8") as f:
            f.write("{}\n")
        with open(t_act, "w", encoding="utf-8") as f:
            f.write("{}\n")

        # Subagent was touched most recently, but is NOT a main conversation pane
        os.utime(t_old, (1000, 1000))
        os.utime(t_act, (2000, 2000))
        os.utime(t_sub, (3000, 3000))

        # Storage contains conversationPanes with main-thread-active and main-thread-old
        storage_file = os.path.join(self.test_dir, "app_storage_panes.json")
        storage_content = {
            "aux-pane-v2-session": json.dumps({
                "version": 1,
                "conversationPanes": {
                    "main-thread-old": {"tabs": []},
                    "main-thread-active": {"tabs": [{"childCascadeId": "subagent-child-uuid"}]}
                }
            })
        }
        with open(storage_file, "w", encoding="utf-8") as f:
            json.dump(storage_content, f)

        discovered = discover_most_recent_cascade_id(brain_dir=brain_dir, storage_path=storage_file)
        # MUST select the active main conversation pane (main-thread-active), not the subagent
        self.assertEqual(discovered, "main-thread-active")

    def test_discover_most_recent_cascade_id_cross_check_layout_keys_priority(self):
        """Verifies layout keys are prioritized if aux-pane-v2-session is absent or non-matching."""
        brain_dir = os.path.join(self.test_dir, "mock_brain_3")
        c1 = os.path.join(brain_dir, "layout-cand-1")
        c2 = os.path.join(brain_dir, "layout-cand-2")
        c3 = os.path.join(brain_dir, "other-cand")

        for d in [c1, c2, c3]:
            os.makedirs(d, exist_ok=True)
            with open(os.path.join(d, "transcript.jsonl"), "w", encoding="utf-8") as f:
                f.write("{}\n")

        os.utime(os.path.join(c1, "transcript.jsonl"), (100, 100))
        os.utime(os.path.join(c2, "transcript.jsonl"), (200, 200))
        os.utime(os.path.join(c3, "transcript.jsonl"), (300, 300))

        storage_file = os.path.join(self.test_dir, "app_storage_layout.json")
        storage_content = {
            "antigravity-multi-conversation-layout-v3-layout-cand-1": {},
            "antigravity-multi-conversation-layout-v3-layout-cand-2+_new": {},
            "antigravity-multi-conversation-layout-v3-index": {}
        }
        with open(storage_file, "w", encoding="utf-8") as f:
            json.dump(storage_content, f)

        discovered = discover_most_recent_cascade_id(brain_dir=brain_dir, storage_path=storage_file)
        self.assertEqual(discovered, "layout-cand-2")

    def test_load_trajectory_mock_success_and_error(self):
        """Verifies load_trajectory Connect-RPC request headers, payload, and response handling."""
        mock_info = LanguageServerInfo(pid=1234, csrf_token="csrf-lt-1", port=55100)

        with patch("urllib.request.urlopen") as mock_urlopen:
            mock_resp = MagicMock()
            mock_resp.status = 200
            mock_resp.__enter__.return_value = mock_resp
            mock_urlopen.return_value = mock_resp

            res = load_trajectory(mock_info, "test-cascade-id-123")
            self.assertTrue(res)

            req = mock_urlopen.call_args[0][0]
            self.assertEqual(req.full_url, "http://127.0.0.1:55100/exa.language_server_pb.LanguageServerService/LoadTrajectory")
            self.assertEqual(req.headers["X-csrf-token"], "csrf-lt-1")
            self.assertEqual(req.headers["X-codeium-csrf-token"], "csrf-lt-1")
            payload = json.loads(req.data.decode("utf-8"))
            self.assertEqual(payload, {"cascadeId": "test-cascade-id-123"})

        # Error cases
        with patch("urllib.request.urlopen", side_effect=RuntimeError("connection refused")):
            self.assertFalse(load_trajectory(mock_info, "test-cascade-id-123"))

        self.assertFalse(load_trajectory(None, "test-cascade-id-123"))
        self.assertFalse(load_trajectory(mock_info, ""))
        self.assertFalse(load_trajectory(LanguageServerInfo(pid=0, csrf_token="", port=0), "test-cascade-id-123"))

    def test_smart_focus_conversation_mock_success_and_resilience(self):
        """Verifies smart_focus_conversation Connect-RPC invocation and graceful exception handling."""
        mock_info = LanguageServerInfo(pid=1234, csrf_token="csrf-sfc-1", port=55100)

        with patch("urllib.request.urlopen") as mock_urlopen:
            mock_resp = MagicMock()
            mock_resp.status = 200
            mock_resp.__enter__.return_value = mock_resp
            mock_urlopen.return_value = mock_resp

            res = smart_focus_conversation(mock_info, "cascade-sfc-target")
            self.assertTrue(res)

            req = mock_urlopen.call_args[0][0]
            self.assertEqual(req.full_url, "http://127.0.0.1:55100/exa.language_server_pb.LanguageServerService/SmartFocusConversation")
            payload = json.loads(req.data.decode("utf-8"))
            self.assertEqual(payload, {"cascadeId": "cascade-sfc-target"})

        # Safe exception handling without crashing
        with patch("urllib.request.urlopen", side_effect=Exception("Internal gRPC exception")):
            self.assertFalse(smart_focus_conversation(mock_info, "cascade-sfc-target"))

        self.assertFalse(smart_focus_conversation(None, "cascade-sfc-target"))
        self.assertFalse(smart_focus_conversation(mock_info, "   "))

    def test_send_user_cascade_message_mock_success_and_custom_prompt(self):
        """Verifies send_user_cascade_message Connect-RPC payload with default and custom prompts."""
        mock_info = LanguageServerInfo(pid=1234, csrf_token="csrf-msg-1", port=55100)

        with patch("urllib.request.urlopen") as mock_urlopen:
            mock_resp = MagicMock()
            mock_resp.status = 200
            mock_resp.__enter__.return_value = mock_resp
            mock_urlopen.return_value = mock_resp

            # Default prompt
            res1 = send_user_cascade_message(mock_info, "cascade-msg-target")
            self.assertTrue(res1)
            req1 = mock_urlopen.call_args[0][0]
            payload1 = json.loads(req1.data.decode("utf-8"))
            self.assertEqual(payload1, {"cascadeId": "cascade-msg-target", "items": [{"text": "continue"}]})

            # Custom prompt
            res2 = send_user_cascade_message(mock_info, "cascade-msg-target", prompt_text="execute test step")
            self.assertTrue(res2)
            req2 = mock_urlopen.call_args[0][0]
            payload2 = json.loads(req2.data.decode("utf-8"))
            self.assertEqual(payload2, {"cascadeId": "cascade-msg-target", "items": [{"text": "execute test step"}]})

        # Error handling
        with patch("urllib.request.urlopen", side_effect=Exception("Timeout")):
            self.assertFalse(send_user_cascade_message(mock_info, "cascade-msg-target"))
        self.assertFalse(send_user_cascade_message(None, "cascade-msg-target"))

    def test_switch_account_native_rpc_continuation_pipeline_success(self):
        """
        Verifies switch_account executes 3-step native continuation pipeline in order:
        1. smart_focus_conversation(info, cascade_id)
        2. load_trajectory(info, cascade_id)
        3. send_user_cascade_message(info, cascade_id, prompt_text)
        and strictly bypasses Win32 keyboard prompt_injector when RPC succeeds.
        """
        target_email = "rpc_native@gmail.com"
        self.store.store_account(target_email, {"access_token": "at_native"})

        mock_info = LanguageServerInfo(pid=8888, csrf_token="csrf-native", port=59000)
        target_cascade_id = "cascade-native-thread-uuid"
        rpc_call_log = []

        def mock_focuser(info, cid):
            rpc_call_log.append(("smart_focus", info.pid, cid))
            return True

        def mock_loader(info, cid):
            rpc_call_log.append(("load_trajectory", info.pid, cid))
            return True

        def mock_sender(info, cid, prompt_text="continue"):
            rpc_call_log.append(("send_cascade_message", info.pid, cid, prompt_text))
            return True

        keyboard_injector_called = []
        def mock_injector(prompt_text="continue", sleep_fn=None):
            keyboard_injector_called.append(prompt_text)
            return True

        manager = AccountManager(
            store=self.store,
            process_finder=lambda: mock_info,
            status_getter=lambda info: {"email": "old_user@gmail.com"},
            token_writer=lambda tok: True,
            tos_pre_patcher=lambda em: True,
            language_server_recycler=lambda: True,
            cascade_discoverer=lambda: target_cascade_id,
            smart_focuser=mock_focuser,
            trajectory_loader=mock_loader,
            cascade_sender=mock_sender,
            prompt_injector=mock_injector,
            auto_continue=True,
            dom_settle_delay=0.0,
        )

        res = manager.switch_account(target_email, prompt_text="continue pipeline")
        self.assertTrue(res.success)

        # 1. Verify 3-step native RPC pipeline called in exact order
        self.assertEqual(len(rpc_call_log), 3)
        self.assertEqual(rpc_call_log[0], ("smart_focus", 8888, target_cascade_id))
        self.assertEqual(rpc_call_log[1], ("load_trajectory", 8888, target_cascade_id))
        self.assertEqual(rpc_call_log[2], ("send_cascade_message", 8888, target_cascade_id, "continue pipeline"))

        # 2. CRITICAL INVARIANT: Win32 keyboard emulation was strictly bypassed!
        self.assertEqual(keyboard_injector_called, [])

    def test_switch_account_native_rpc_fallback_when_rpc_fails(self):
        """Verifies switch_account never invokes Win32 keyboard prompt_injector even if send_user_cascade_message fails."""
        target_email = "rpc_fail@gmail.com"
        self.store.store_account(target_email, {"access_token": "at_fail"})
        mock_info = LanguageServerInfo(pid=8888, csrf_token="csrf-fail", port=59000)

        keyboard_injector_called = []
        def mock_injector(prompt_text="continue", sleep_fn=None):
            keyboard_injector_called.append(prompt_text)
            return True

        manager = AccountManager(
            store=self.store,
            process_finder=lambda: mock_info,
            status_getter=lambda info: {"email": "old_user@gmail.com"},
            token_writer=lambda tok: True,
            tos_pre_patcher=lambda em: True,
            language_server_recycler=lambda: True,
            cascade_discoverer=lambda: "failed-cascade-uuid",
            smart_focuser=lambda info, cid: True,
            trajectory_loader=lambda info, cid: True,
            cascade_sender=lambda info, cid, prompt_text="continue": False,  # RPC failed
            prompt_injector=mock_injector,
            auto_continue=True,
            dom_settle_delay=0.0,
        )

        res = manager.switch_account(target_email, prompt_text="fallback-continue")
        self.assertTrue(res.success)
        # Keyboard emulation strictly purged: zero keyboard events dispatched
        self.assertEqual(keyboard_injector_called, [])

    def test_switch_account_native_rpc_fallback_when_no_cascade_id(self):
        """Verifies switch_account never invokes Win32 keyboard prompt_injector if no cascade_id is discoverable."""
        target_email = "no_cascade@gmail.com"
        self.store.store_account(target_email, {"access_token": "at_nocascade"})
        mock_info = LanguageServerInfo(pid=8888, csrf_token="csrf-nocascade", port=59000)

        keyboard_injector_called = []
        def mock_injector(prompt_text="continue", sleep_fn=None):
            keyboard_injector_called.append(prompt_text)
            return True

        manager = AccountManager(
            store=self.store,
            process_finder=lambda: mock_info,
            status_getter=lambda info: {"email": "old_user@gmail.com"},
            token_writer=lambda tok: True,
            tos_pre_patcher=lambda em: True,
            language_server_recycler=lambda: True,
            cascade_discoverer=lambda: None,  # No cascade discovered
            prompt_injector=mock_injector,
            auto_continue=True,
            dom_settle_delay=0.0,
        )

        res = manager.switch_account(target_email, prompt_text="fallback-nocascade")
        self.assertTrue(res.success)
        # Keyboard emulation strictly purged: zero keyboard events dispatched
        self.assertEqual(keyboard_injector_called, [])

    def test_capture_active_threads_snapshot_schema_and_timestamp(self):
        """
        Verifies capture_active_threads_snapshot creates a timestamped JSON file
        in accounts_store/active_threads_history/<YYYY-MM-DD-HH-MM-SS>_active_threads.json
        matching the exact schema:
        {"timestamp": "YYYY-MM-DD HH:MM:SS", "active_threads": [{"thread_id": "...", "project": "...", "is_main_pane": bool}]}
        """
        import re
        temp_storage = os.path.join(self.test_dir, "test_app_storage.json")
        temp_history_dir = os.path.join(self.test_dir, "history_out")

        mock_storage_data = {
            "aux-pane-v2-session.conversationPanes": [
                {
                    "paneId": "main_cascade_pane",
                    "tabs": [
                        {
                            "tabId": "tab-1",
                            "metadata": {
                                "cascadeId": "thread-main-alpha",
                                "workingDirectory": "C:/Projects/ProjectAlpha"
                            }
                        }
                    ]
                },
                {
                    "paneId": "secondary_pane",
                    "tabs": [
                        {
                            "tabId": "tab-2",
                            "metadata": {
                                "cascadeId": "thread-sec-beta",
                                "workspaceUri": "file:///C:/Projects/ProjectBeta"
                            }
                        }
                    ]
                }
            ]
        }
        with open(temp_storage, "w", encoding="utf-8") as f:
            json.dump(mock_storage_data, f)

        snapshot_file, active_threads = capture_active_threads_snapshot(
            storage_path=temp_storage,
            brain_dir=self.test_dir,
            output_dir=temp_history_dir
        )

        self.assertIsNotNone(snapshot_file)
        self.assertTrue(os.path.exists(snapshot_file))
        basename = os.path.basename(snapshot_file)
        self.assertTrue(
            bool(re.match(r"^\d{4}-\d{2}-\d{2}-\d{2}-\d{2}-\d{2}_active_threads\.json$", basename)),
            f"Filename {basename} does not match timestamp format"
        )

        with open(snapshot_file, "r", encoding="utf-8") as f:
            data = json.load(f)

        self.assertIn("timestamp", data)
        self.assertIn("active_threads", data)
        self.assertTrue(
            bool(re.match(r"^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$", data["timestamp"])),
            f"Timestamp {data['timestamp']} does not match format"
        )

        threads = data["active_threads"]
        self.assertEqual(len(threads), 2)
        self.assertEqual(threads[0]["thread_id"], "thread-main-alpha")
        self.assertTrue(threads[0]["is_main_pane"])
        self.assertEqual(threads[0]["project"], "C:/Projects/ProjectAlpha")

        self.assertEqual(threads[1]["thread_id"], "thread-sec-beta")
        self.assertFalse(threads[1]["is_main_pane"])
        self.assertEqual(threads[1]["project"], "C:/Projects/ProjectBeta")

    def test_capture_active_threads_snapshot_layout_keys_and_fallback(self):
        """Verifies capture_active_threads_snapshot strictly captures active conversationPanes and ignores stale layout keys."""
        temp_storage = os.path.join(self.test_dir, "layout_storage.json")
        temp_brain = os.path.join(self.test_dir, "brain")
        temp_history_dir = os.path.join(self.test_dir, "layout_history")

        active_thread_id = "thread-active-in-pane"
        stale_thread_1 = "thread-stale-historical-1"
        stale_thread_2 = "thread-stale-historical-2"

        thread_brain_dir = os.path.join(temp_brain, active_thread_id)
        os.makedirs(thread_brain_dir, exist_ok=True)
        with open(os.path.join(thread_brain_dir, "metadata.json"), "w", encoding="utf-8") as f:
            json.dump({"active_workspace": "D:/workspace/gamma_proj"}, f)

        mock_storage_data = {
            "aux-pane-v2-session": json.dumps({
                "conversationPanes": {
                    "pane-main": {
                        "paneId": "pane-main",
                        "tabs": [
                            {
                                "cascadeId": active_thread_id,
                                "metadata": {
                                    "workingDirectory": "D:/workspace/gamma_proj"
                                }
                            }
                        ]
                    }
                }
            }),
            f"antigravity-multi-conversation-layout-v3-{stale_thread_1}": json.dumps({"activeThreadId": stale_thread_1}),
            f"antigravity-multi-conversation-layout-v3-{stale_thread_2}+_new": json.dumps({"activeThreadId": stale_thread_2}),
            "workbench.panel.cascade": {
                "activeThreadId": "stale-workbench-thread",
            }
        }
        with open(temp_storage, "w", encoding="utf-8") as f:
            json.dump(mock_storage_data, f)

        snapshot_file, active_threads = capture_active_threads_snapshot(
            storage_path=temp_storage,
            brain_dir=temp_brain,
            output_dir=temp_history_dir
        )

        # Strictly captures ONLY active conversationPanes, ignoring stale layout keys
        self.assertEqual(len(active_threads), 1)
        self.assertEqual(active_threads[0]["thread_id"], active_thread_id)
        self.assertEqual(active_threads[0]["project"], "D:/workspace/gamma_proj")
        self.assertTrue(active_threads[0]["is_main_pane"])

        captured_ids = [t["thread_id"] for t in active_threads]
        self.assertNotIn(stale_thread_1, captured_ids)
        self.assertNotIn(stale_thread_2, captured_ids)
        self.assertNotIn("stale-workbench-thread", captured_ids)

    def test_switch_account_fresh_language_server_info_rediscovery(self):
        """
        Verifies switch_account re-queries process_finder() post-recycle to retrieve fresh LanguageServerInfo
        (new PID, new Port, new CSRF token), never using stale info from before recycling.
        """
        target_email = "freshls@gmail.com"
        self.store.store_account(target_email, {"access_token": "at_fresh"})

        old_info = LanguageServerInfo(pid=1111, csrf_token="old_csrf", port=41111)
        fresh_info = LanguageServerInfo(pid=2222, csrf_token="fresh_csrf", port=52222)

        finder_calls = 0
        def mock_finder():
            nonlocal finder_calls
            finder_calls += 1
            if finder_calls == 1:
                return old_info
            return fresh_info

        rpc_infos_received = []
        def mock_focuser(info, cid):
            rpc_infos_received.append(("focus", info.pid, info.port, info.csrf_token, cid))
            return True

        def mock_loader(info, cid):
            rpc_infos_received.append(("load", info.pid, info.port, info.csrf_token, cid))
            return True

        def mock_sender(info, cid, prompt_text="continue"):
            rpc_infos_received.append(("send", info.pid, info.port, info.csrf_token, cid, prompt_text))
            return True

        manager = AccountManager(
            store=self.store,
            process_finder=mock_finder,
            status_getter=lambda info: {"email": "old_user@gmail.com"},
            token_writer=lambda tok: True,
            tos_pre_patcher=lambda em: True,
            language_server_recycler=lambda: True,
            cascade_discoverer=lambda: "target-cascade-id",
            smart_focuser=mock_focuser,
            trajectory_loader=mock_loader,
            cascade_sender=mock_sender,
            auto_continue=True,
            dom_settle_delay=0.0,
        )

        res = manager.switch_account(target_email, prompt_text="continue task")
        self.assertTrue(res.success)
        self.assertGreaterEqual(finder_calls, 2)

        # Asserts ALL RPC continuation calls received the FRESH LanguageServerInfo (PID 2222, port 52222)
        self.assertEqual(len(rpc_infos_received), 3)
        self.assertEqual(rpc_infos_received[0], ("focus", 2222, 52222, "fresh_csrf", "target-cascade-id"))
        self.assertEqual(rpc_infos_received[1], ("load", 2222, 52222, "fresh_csrf", "target-cascade-id"))
        self.assertEqual(rpc_infos_received[2], ("send", 2222, 52222, "fresh_csrf", "target-cascade-id", "continue task"))

    def test_switch_account_multi_project_pure_native_rpc_continuation(self):
        """
        Verifies switch_account iterates over all active threads across multiple projects
        from snapshot, executing native RPC continuation for each thread in order with zero keyboard events.
        """
        target_email = "multiproject@gmail.com"
        self.store.store_account(target_email, {"access_token": "at_multi"})
        fresh_info = LanguageServerInfo(pid=3333, csrf_token="multi_csrf", port=53333)

        mock_threads_snapshot = [
            {"thread_id": "thread-alpha", "project": "C:/Projects/Alpha", "is_main_pane": True},
            {"thread_id": "thread-beta", "project": "C:/Projects/Beta", "is_main_pane": False},
            {"thread_id": "thread-gamma", "project": "C:/Projects/Gamma", "is_main_pane": False},
        ]

        rpc_sequence = []
        def mock_focuser(info, cid):
            rpc_sequence.append(("focus", cid))
            return True

        def mock_loader(info, cid):
            rpc_sequence.append(("load", cid))
            return True

        def mock_sender(info, cid, prompt_text="continue"):
            rpc_sequence.append(("send", cid, prompt_text))
            return True

        keyboard_calls = []
        def mock_injector(prompt_text="continue", sleep_fn=None):
            keyboard_calls.append(prompt_text)
            return True

        sleep_log = []
        def mock_sleeper(duration):
            sleep_log.append(duration)

        manager = AccountManager(
            store=self.store,
            process_finder=lambda: fresh_info,
            status_getter=lambda info: {"email": "old_user@gmail.com"},
            token_writer=lambda tok: True,
            tos_pre_patcher=lambda em: True,
            language_server_recycler=lambda: True,
            threads_snapshot_capturer=lambda: ("/path/snapshot.json", mock_threads_snapshot),
            smart_focuser=mock_focuser,
            trajectory_loader=mock_loader,
            cascade_sender=mock_sender,
            prompt_injector=mock_injector,
            sleep_fn=mock_sleeper,
            auto_continue=True,
            dom_settle_delay=0.0,
        )

        res = manager.switch_account(target_email, prompt_text="continue working")
        self.assertTrue(res.success)

        # 3 threads * 3 RPCs = 9 calls in exact sequence
        self.assertEqual(len(rpc_sequence), 9)
        self.assertEqual(rpc_sequence[0:3], [("focus", "thread-alpha"), ("load", "thread-alpha"), ("send", "thread-alpha", "continue working")])
        self.assertEqual(rpc_sequence[3:6], [("focus", "thread-beta"), ("load", "thread-beta"), ("send", "thread-beta", "continue working")])
        self.assertEqual(rpc_sequence[6:9], [("focus", "thread-gamma"), ("load", "thread-gamma"), ("send", "thread-gamma", "continue working")])

        # Assert 3.0s pacing delay between threads (2 delays for 3 threads)
        pacing_delays = [d for d in sleep_log if d == 3.0]
        self.assertEqual(len(pacing_delays), 2)

        # Zero keyboard events
        self.assertEqual(keyboard_calls, [])

    def test_switch_account_default_dom_settle_delay_15s(self):
        """Verifies switch_account dom_settle_delay defaults to 15.0s when not specified."""
        target_email = "settle15@gmail.com"
        self.store.store_account(target_email, {"access_token": "at_15"})
        sleep_durations = []

        fresh_info = LanguageServerInfo(pid=5555, csrf_token="tok15", port=55551)
        manager = AccountManager(
            store=self.store,
            process_finder=lambda: fresh_info,
            token_writer=lambda tok: True,
            tos_pre_patcher=lambda em: True,
            language_server_recycler=lambda **kw: None,
            threads_snapshot_capturer=lambda: ("/path", [{"thread_id": "c15", "project": "", "is_main_pane": True}]),
            smart_focuser=lambda info, cid: True,
            trajectory_loader=lambda info, cid: True,
            cascade_sender=lambda info, cid, prompt_text="": True,
            sleep_fn=lambda d: sleep_durations.append(d),
            auto_continue=True,
            # No dom_settle_delay passed -> MUST default to 15.0s
        )

        res = manager.switch_account(target_email)
        self.assertTrue(res.success)
        self.assertIn(15.0, sleep_durations)

    def test_switch_account_default_continuation_prompt(self):
        """Verifies default continuation prompt matches the English directive."""
        self.assertEqual(
            DEFAULT_CONTINUATION_PROMPT,
            "Continue if you still have unfinished work in progress. If all tasks are already completed, reply with: Currently there is no work to do."
        )
        manager = AccountManager(store=self.store)
        self.assertEqual(manager.prompt_text, DEFAULT_CONTINUATION_PROMPT)


if __name__ == "__main__":
    unittest.main()

