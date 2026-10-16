# Supervisor Report: GUI Account Switching Flow & Silent Process Execution Review

Verdict: REJECT

## Metadata
- Report file: reports/supervisor/2026-10-16-gui-switch-flow-and-ux-supervisor-review.md
- Review time: 2026-10-16 14:05:00 Asia/Bangkok
- Reviewer: supervisor (gpt-6-astra)
- Repo/project: antigravity_auto_switch
- Scope reviewed:
  - internal/ui/app.go (Account switch & Save session UI flow)
  - internal/connector/discovery.go (Language server & netstat process execution)
  - internal/reloader/session_reloader.go (Process termination & taskkill execution)
  - ui/app_window.py (Python reference UX SSOT: zero modal popups)
  - core/session_reloader.py (Python reference Stealth Execution SSOT: CREATE_NO_WINDOW)
- Claim reviewed: GUI Account Switching maintains seamless user flow without blocking modal dialogs (Python parity), and all underlying system commands execute completely silently in background without flashing console/terminal windows on Windows.
- Authority used:
  - Human Owner direct review directive on GUI runtime behavior
  - Python SSOT: ui/app_window.py:508-509 (purged messagebox.showinfo, zero blocking popups)
  - Python SSOT: core/session_reloader.py:86-88, 325-327 (subprocess.CREATE_NO_WINDOW)
  - Microsoft Win32 Process Creation Flags: CREATE_NO_WINDOW = 0x08000000, SysProcAttr.HideWindow = true
- Related artifacts:
  - Commit 741410c
  - Report: reports/supervisor/2026-10-16-fix-gui-startup-panic-supervisor-review.md

## Executive Summary
- Problem: Manual GUI testing by Human Owner revealed two critical UX regressions in the account switching flow:
  1. Success modal popup (dialog.ShowInformation) appears after switching account or saving session, interrupting user flow.
  2. Multiple black terminal/console windows flash on screen during account switching due to visible execution of powershell.exe, netstat.exe, and taskkill.exe.
- Decision: REJECT. Independent code and SSOT inspection confirms both regressions exist in the Go codebase and violate parity with the Python prototype.
- Required outcome: Remediation required by Coder Lane.

## Blocking Findings

### [P1] Blocking Modal Popup On Account Switch & Save Session (UX Violation)
File: internal/ui/app.go:197, 235
Issue: When account switching succeeds, Go UI displays dialog.ShowInformation("Chuyển thành công", res.Message, a.win). When saving session succeeds, it displays dialog.ShowInformation("Lưu phiên", ...). This creates a blocking modal dialog requiring the user to click "OK", breaking uninterrupted workflow.
Evidence:
- Python SSOT ui/app_window.py:508-509:
  ```python
  # CRITICAL UX REFACTORING: Completely purged messagebox.showinfo!
  # Zero blocking modal popups. The user remains 100% in uninterrupted flow.
  ```
  Python prototype updates only self.status_bar_text and refreshes account cards in RAM.
- Go implementation internal/ui/app.go:197:
  ```go
  dialog.ShowInformation("Chuyển thành công", res.Message, a.win)
  ```
Why this blocks acceptance: Violates Python SSOT parity and UX responsiveness invariant.
Fix direction: Remove dialog.ShowInformation at internal/ui/app.go:197 and 235. Ensure success feedback is delivered exclusively via statusBarLabel.SetText(...) and dynamic card state updates.
Re-review evidence required: Diff of internal/ui/app.go showing removal of dialog.ShowInformation, and manual/unit verification that switching accounts does not produce modal dialogs.

### [P1] Flashing Console Windows During Account Switch (Stealth Execution Violation)
File: internal/connector/discovery.go:72, 195 and internal/reloader/session_reloader.go:71
Issue: Executing powershell.exe, netstat.exe, and taskkill.exe without Windows process creation attributes (CREATE_NO_WINDOW) causes the Windows OS console subsystem to allocate and flash black console/terminal windows on screen.
Evidence:
- Python SSOT core/session_reloader.py:86-88, 325-327:
  ```python
  creationflags = 0
  if hasattr(subprocess, "CREATE_NO_WINDOW"):
      creationflags = subprocess.CREATE_NO_WINDOW
  res = subprocess.run(..., creationflags=creationflags)
  ```
- Go implementation:
  - internal/connector/discovery.go:72: exec.Command("powershell", ...) lacks SysProcAttr.
  - internal/connector/discovery.go:195: exec.Command("netstat", ...) lacks SysProcAttr.
  - internal/reloader/session_reloader.go:71: exec.CommandContext(ctx, "taskkill", ...) lacks SysProcAttr.
Why this blocks acceptance: Violates silent background execution requirement on Windows desktop.
Fix direction: In Go on Windows, configure SysProcAttr on all exec.Cmd instances invoking console binaries:
  ```go
  cmd.SysProcAttr = &syscall.SysProcAttr{
      HideWindow:    true,
      CreationFlags: 0x08000000, // CREATE_NO_WINDOW
  }
  ```
Re-review evidence required: Diff across all exec.Command callers and runtime proof that account switching runs with 0 console windows visible.

## Source-Level Clearance Notes
- internal/ui/app.go: BLOCKED - contains dialog.ShowInformation at lines 197 and 235.
- internal/connector/discovery.go: BLOCKED - exec.Command calls at lines 72 and 195 lack CREATE_NO_WINDOW SysProcAttr.
- internal/reloader/session_reloader.go: BLOCKED - exec.CommandContext call at line 71 lacks CREATE_NO_WINDOW SysProcAttr.
- internal/ui/window.go: CLEAR - switcherWindow wrapper remains eliminated; setWindowAlwaysOnTop helper is sound.
- internal/ui/tray.go: CLEAR - desk.SetSystemTrayWindow(a.win) receives native fyne.Window.

## Evidence Checked
Passed:
- GUI Startup without crash/panic (bin/antigravity-switch-gui.exe ALIVE=True).
- Always on Top Win32 integration (internal/ui/window.go).

Failed:
- Python parity on account switch UX: internal/ui/app.go:197 invokes blocking modal dialog.
- Silent execution on Windows: internal/connector/discovery.go:72, 195 and internal/reloader/session_reloader.go:71 spawn unhidden console processes.

## Invariant Closure
- Affected invariant: SilentBackgroundExecutionAndUninterruptedUX
- Sibling surfaces checked: All exec.Command invocations in codebase (3 locations found); all dialog.Show* calls in internal/ui/ (4 locations found, 2 success modals must be removed).
- Residual unverified same-invariant surfaces: none.

## Required Fix List For Resubmission
1. In internal/ui/app.go:
   - Delete dialog.ShowInformation("Chuyển thành công", ...) at line 197.
   - Delete dialog.ShowInformation("Lưu phiên", ...) at line 235.
   - Retain error dialogs (dialog.ShowError) and status bar messages.
2. In internal/connector/discovery.go and internal/reloader/session_reloader.go:
   - Set cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} on all exec.Command / exec.CommandContext instances on Windows.
3. Rebuild with powershell -File scripts/build_windows.ps1 -Mode all.
4. Runtime test: Perform account switch in GUI and verify:
   - No success modal pops up.
   - No terminal/console windows flash.
