# Supervisor Report: Zero-Trust Acceptance Review for GUI Startup Panic Fix

Verdict: PASS

## Metadata
- Report file: reports/supervisor/2026-10-16-fix-gui-startup-panic-supervisor-review.md
- Review time: 2026-10-16 13:40:00 Asia/Bangkok
- Reviewer: supervisor (gpt-6-astra)
- Repo/project: antigravity_auto_switch
- Scope reviewed: Commit 741410c2f06456e727d00481bec2853770b08463
  - internal/ui/app.go
  - internal/ui/window.go
  - internal/ui/tray.go
  - internal/ui/ui_test.go
  - bin/antigravity-switch-gui.exe
  - reports/coder/2026-10-16-fix-gui-startup-panic-report.md
- Claim reviewed: Coder completely eliminated the GUI startup panic (`panic: interface conversion: fyne.Window is *ui.switcherWindow, not *glfw.window`) by eradicating the custom `switcherWindow` wrapper struct and `Window` interface 100%, restoring `a.win` as the native, unadulterated `fyne.Window` (*glfw.window) returned by `a.app.NewWindow(...)`, passing `a.win` directly into `desk.SetSystemTrayWindow(a.win)`, and decoupling the Always on Top feature into a standalone Win32 API helper `setWindowAlwaysOnTop` via `user32.dll` (`FindWindowW` + `SetWindowPos`).
- Authority used:
  - CEO Task Contract directive for Zero-Trust independent acceptance review of commit 741410c
  - Fyne v2 Desktop Driver Contract: `internal/driver/glfw.(*gLDriver).SetSystemTrayWindow(w fyne.Window)` requiring `w` to be the concrete `*glfw.window`
  - Microsoft Win32 User32 API: `FindWindowW` and `SetWindowPos` (`HWND_TOPMOST` / `HWND_NOTOPMOST`, `SWP_NOMOVE | SWP_NOSIZE`)
  - Coder Report: `reports/coder/2026-10-16-fix-gui-startup-panic-report.md`
- Related artifacts:
  - Commit 741410c2f06456e727d00481bec2853770b08463
  - Build script: `scripts/build_windows.ps1`
  - Compiled GUI binary: `bin/antigravity-switch-gui.exe` (23.54 MB)

## Executive Summary
- Problem: During commit `787489a` / `a7265d0` (P1-02), `a.win` was wrapped in a custom struct `*switcherWindow` to attach `SetAlwaysOnTop(bool)`. At runtime on Windows, Fyne's GLFW desktop driver executed `desk.SetSystemTrayWindow(a.win)` which internally performs a type assertion to `*glfw.window`. Because `a.win` was `*ui.switcherWindow`, Go runtime threw a fatal panic `panic: interface conversion: fyne.Window is *ui.switcherWindow, not *glfw.window`, causing `antigravity-switch-gui.exe` to crash immediately with exit code 2 on startup.
- Decision: PASS. Independent zero-trust line-by-line inspection and real runtime execution confirm that:
  1. The custom `switcherWindow` struct and custom `Window` interface have been 100% eradicated from the codebase (`git grep "switcherWindow" -- "*.go"` returned 0 matches).
  2. `App.win` is restored to `fyne.Window`, directly instantiated as `a.win = a.app.NewWindow(windowTitle)` with zero wrapper or proxy.
  3. `desk.SetSystemTrayWindow(a.win)` receives the exact concrete `*glfw.window` instance, completely eliminating the interface conversion panic at the architectural root.
  4. Always on Top functionality is fully preserved and decoupled via `setWindowAlwaysOnTop(title string, isTop bool)` using `user32.dll`'s `FindWindowW` and `SetWindowPos` with `HWND_TOPMOST` (-1) / `HWND_NOTOPMOST` (-2), executing safely as a no-op if window is not found in headless test environments.
  5. Real runtime smoke test (`Start-Process bin\antigravity-switch-gui.exe -PassThru`) confirmed the GUI process runs normally and stays active (`ALIVE=True`, `HasExited=False`), with zero crashes or panics.
  6. Compiler, linter, and build pipeline checks are completely clean (`go vet ./internal/ui/...` 0 errors; `powershell -File scripts/build_windows.ps1 -Mode all` produced both CLI and GUI binaries with exit code 0).
- Required outcome: accepted (Ready for Owner GUI testing).

## Source-Level Clearance Notes
- `internal/ui/window.go`: CLEAR
  - Lines 15: Defined standard window title constant `const windowTitle = "⚡ Antigravity Multi-Account Switcher"`.
  - Lines 17-30: Declared Win32 User32 procedures (`FindWindowW`, `SetWindowPos`) and constants (`hwndTopMost`, `hwndNoTopMost`, `swpNoMove`, `swpNoSize`).
  - Lines 32-47: Implemented `setWindowAlwaysOnTop(title string, isTop bool)`: locates HWND via `StringToUTF16Ptr(title)` and `FindWindowW(0, ...)`. If found, invokes `SetWindowPos(hwnd, insertAfter, 0, 0, 0, 0, swpNoMove | swpNoSize)`. If HWND is 0 (e.g., in unit test headless runners), returns immediately without error or panic.
  - Custom `Window` interface, `switcherWindow` struct, and its delegation methods are completely removed.
- `internal/ui/app.go`: CLEAR
  - Line 22: Field `win` reverted to `fyne.Window` (pure standard Fyne interface).
  - Lines 69-71: In `Init()`, `a.win` is assigned directly via `a.app.NewWindow(windowTitle)`.
  - Lines 76-78: Initial Always-on-Top state applied via standalone helper `setWindowAlwaysOnTop(windowTitle, true)`.
  - Lines 89-91: In `ShowAndRun()`, ensures `setWindowAlwaysOnTop(windowTitle, true)` before running event loop.
  - Lines 98-102: `Window() fyne.Window` getter returns native `fyne.Window`.
  - Lines 258, 286: In `ToggleAlwaysOnTop()` and `SetAlwaysOnTop(enabled bool)`, calls `setWindowAlwaysOnTop(windowTitle, enabled)` without mutating or wrapping `a.win`.
- `internal/ui/tray.go`: CLEAR
  - Lines 52-57: In Tray menu "Hiển thị Switcher", calls `a.win.Show()`, `a.win.RequestFocus()`, and if `a.IsAlwaysOnTop()` calls `setWindowAlwaysOnTop(windowTitle, true)` to restore topmost z-order.
  - Lines 78-84: Passes `a.win` directly to `desk.SetSystemTrayWindow(a.win)`, perfectly matching the GLFW driver contract.
- `internal/ui/ui_test.go`: CLEAR
  - Lines 162-175: Updated `TestToggleAlwaysOnTop` to verify `app.Window()` is non-nil with title `windowTitle`, and verifies `setWindowAlwaysOnTop` executes safely on both valid and non-existent window titles without throwing any panics.

## Evidence Checked
Passed:
- Source inspection of `internal/ui/app.go`, `internal/ui/window.go`, `internal/ui/tray.go`, `internal/ui/ui_test.go`:
  - `git grep "switcherWindow" -- "*.go"` returned 0 matches.
  - `git grep "type Window interface" -- "*.go"` returned 0 matches.
  - Verification freshness: fresh (inspected on working tree and commit 741410c).
- Static analysis & linting:
  - `go vet ./internal/ui/...` returned exit code 0, 0 warnings/errors.
  - Verification freshness: fresh.
- Unit testing:
  - `go test -v ./internal/ui/...` executed 8 test functions with 100% PASS in 2.319s.
  - Verification freshness: fresh.
- Full Windows build pipeline:
  - `powershell -File scripts/build_windows.ps1 -Mode all` completed with exit code 0 (`BUILD SUCCESSFUL`).
  - Generated `bin\antigravity-switch.exe` (6.73 MB) and `bin\antigravity-switch-gui.exe` (23.54 MB).
  - Embedded PE icons (32x32) verified on both executables.
  - Verification freshness: fresh.
- Real Runtime Execution (Smoke Test):
  - PowerShell execution:
    `$p = Start-Process -FilePath "G:\antigravity_auto_switch\bin\antigravity-switch-gui.exe" -PassThru; Start-Sleep -Seconds 4; $alive = -not $p.HasExited; Write-Output "ALIVE=$alive PID=$($p.Id) HANDLE=$($p.Handle)"; Stop-Process -Id $p.Id -Force`
  - Output: `ALIVE=True PID=18960 HANDLE=2680`.
  - Process did NOT crash, did NOT exit with code 2, and did NOT emit panic.
  - Verification freshness: fresh.

Failed:
- None.

Not run:
- None.

## Invariant Closure
- Affected invariant: `UIWindowLifecycleAndSystemTrayIntegration`
- Sibling surfaces checked:
  - `cmd/switcher/app_gui.go`: Invokes `app.Init()` and `app.ShowAndRun()` cleanly.
  - `cmd/switcher/main.go`: Delegates to `runGUI` seamlessly.
- Residual unverified same-invariant surfaces: none.

## Overall Evaluation
The fix implemented by Coder at commit `741410c` correctly solves the root cause rather than patching over symptoms. By eliminating the unneeded struct wrapper `switcherWindow` and keeping `a.win` as pure `fyne.Window`, the internal GLFW desktop driver's type assertion in `desk.SetSystemTrayWindow(a.win)` succeeds without issue. Win32 Always on Top functionality is cleanly decoupled into an independent native helper. Real runtime execution conclusively proves that `bin/antigravity-switch-gui.exe` launches smoothly and remains active without crashing.

Acceptance is granted with zero reservations.
