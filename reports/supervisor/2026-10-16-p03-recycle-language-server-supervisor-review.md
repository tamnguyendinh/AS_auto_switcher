# Supervisor Report: Zero-Trust Acceptance Review for P0-03 Language Server Recycling Fix

Verdict: PASS

## Metadata
- Report file: reports/supervisor/2026-10-16-p03-recycle-language-server-supervisor-review.md
- Review time: 2026-10-16 12:45:00 Asia/Bangkok
- Reviewer: supervisor (gpt-6-astra)
- Repo/project: antigravity_auto_switch
- Scope reviewed: Commit 75664d8c275f4c038f388434b9fcd880a46bbaa5 (internal/reloader/session_reloader.go, internal/manager/account_manager.go, internal/reloader/reloader_test.go, internal/manager/account_manager_test.go, eports/coder/2026-10-16-fix-p03-recycle-language-server-report.md)
- Claim reviewed: Coder fixed P0-03 by completely purging virtual key Ctrl+R (keybd_event VK_CONTROL + VK_R) and implementing RecycleLanguageServer via 	askkill /F /IM language_server.exe with process respawn wait and re-probing, achieving 100% parity with Python core/session_reloader.py:304-335.
- Authority used:
  - CEO Task Contract directive for P0-03 review
  - Python SSOT: core/session_reloader.py:304-335
  - Python hot-swap integration: core/state_resolver.py:525-580
  - Prior Phase 6 audit: eports/supervisor/2026-10-16-phase6-python-go-full-parity-zero-trust-review.md (§3.2, §5.2, §5.3)
- Related artifacts:
  - Commit 75664d8
  - Coder Report: eports/coder/2026-10-16-fix-p03-recycle-language-server-report.md

## Executive Summary
- Problem: The Go reloader previously simulated virtual Ctrl+R keystrokes (keybd_event VK_CONTROL + VK_R) to reload the Electron window. Because language_server.exe only reads credentials from Windows Credential Manager on boot via CredReadW, virtual keystrokes did not cause language_server.exe to pick up the new token in RAM. Python solved this cleanly by killing language_server.exe with 	askkill /F, allowing the parent Antigravity.exe (monitorLsCrashInternal) to respawn a fresh child process that re-reads WinCred.
- Decision: PASS. Verified line-by-line that virtual key simulation (procKeybdEvent, VK_CONTROL, VK_R, ReloadAntigravityWindow) has been entirely deleted from Go production code. RecycleLanguageServer was cleanly implemented with parity flags (CREATE_NO_WINDOW, check=False equivalent _ = cmd.Run()), configurable wait duration (default 2s), and verification of PID/port discovery. SwitchAccount calls RecycleLanguageServer after updating WinCred and re-probes active identity. Static codebase scan confirms 0 occurrences of virtual keys or keybd_event in the Go codebase. Both compiler checks (go vet and full Windows build script) passed cleanly with exit code 0.
- Required outcome: accepted (Ready for P0-04).

## Source-Level Clearance Notes
- internal/reloader/session_reloader.go: CLEAR - Line 25-30 eliminated procKeybdEvent. Removed constants VK_CONTROL, VK_R, VK_RETURN, KEYEVENTF_KEYUP. Added DefaultKillLanguageServer (lines 63-69) executing 	askkill /F /IM language_server.exe with windows.CREATE_NO_WINDOW. Added RecycleLanguageServer (lines 79-130) coordinating termination, respawn delay, and new PID/port discovery. Purged ReloadAntigravityWindow. Added EnsureAntigravityForeground and PrepatchToS. ReloadAntigravitySession preserved as compatibility shim delegating to RecycleLanguageServer.
- internal/manager/account_manager.go: CLEAR - Line 429 invokes eloader.PrepatchToS(normTarget). Lines 434-446 invoke el.RecycleLanguageServer(ctx, 2*time.Second) with graceful warning fallback preserving WinCred write. Lines 449-467 re-probe the fresh Language Server via connector.DiscoverLanguageServer() and verify identity with client.GetUserStatus(ctx).
- internal/reloader/reloader_test.go: CLEAR - Removed all legacy keystroke tests. Added unit tests for discovery, recycling success, discovery failure handling, and context cancellation. All passed.
- internal/manager/account_manager_test.go: CLEAR - Added assertions in TestSwitchAccount verifying language server recycling is executed on hot-swap, and verifying warning tolerance if recycling discovery fails. All passed.

## Evidence Checked
Passed:
- Source code inspection: Confirmed complete elimination of procKeybdEvent, VK_CONTROL, VK_R, and ReloadAntigravityWindow in internal/reloader/session_reloader.go.
- Parity check against Python: Verified DefaultKillLanguageServer calls 	askkill /F /IM language_server.exe with windows.CREATE_NO_WINDOW and safely ignores non-zero exit codes when process is already terminated, matching Python core/session_reloader.py:322-329.
- Static Code Verification: Scanned entire Go codebase (internal/, cmd/) for keybd_event, VK_, Ctrl+R. Result: 0 occurrences found.
- Static analysis: go vet ./internal/reloader/... ./internal/manager/... finished with exit code 0 and 0 warnings. Freshness: fresh.
- Unit test suite: go test -count=1 ./internal/reloader/... ./internal/manager/... passed with 100% success in 4.7s. Freshness: fresh.
- Windows Build Automation: Executed powershell -File scripts/build_windows.ps1 -Mode all. Both in/antigravity-switch.exe (Pure Go CLI) and in/antigravity-switch-gui.exe (Fyne GUI) compiled cleanly, passed resource validation, and executed in/antigravity-switch.exe --status verifying live Language Server detection. Exit code 0. Freshness: fresh.

Failed:
- none

Not run:
- none

## Invariant Closure
- affected invariant: LanguageServerRecycleHotSwapParity (P0-03)
- sibling surfaces checked:
  - internal/reloader/session_reloader.go (lifecycle, termination, wait, respawn discovery)
  - internal/manager/account_manager.go (account switch integration, post-swap identity probe)
  - internal/reloader/reloader_test.go (isolated unit verification)
  - internal/manager/account_manager_test.go (orchestrator hot-swap verification)
  - cmd/switcher/* and whole Go codebase (static scan for forbidden virtual keystrokes)
- residual unverified same-invariant surfaces: none

## Overall Evaluation
The implementation by Coder at commit 75664d8 is exceptionally clean, robust, and completely adheres to the architectural requirements. It eliminates the fragile GUI-dependent virtual keystroke mechanism, replaces it with deterministic process recycling via 	askkill /F /IM language_server.exe, and verifies the newly respawned child process before completing the account switch. The codebase compiles cleanly, passes all vet and test checks, and achieves 100% functional parity with the Python reference implementation.