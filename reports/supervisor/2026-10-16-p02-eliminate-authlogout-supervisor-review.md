# Supervisor Report: Zero-Trust Acceptance Review for P0-02 Eliminate AuthLogout Fix

Verdict: PASS

## Metadata
- Report file: `reports/supervisor/2026-10-16-p02-eliminate-authlogout-supervisor-review.md`
- Review time: `2026-10-16 12:00:00 +0700`
- Reviewer: `gpt-6.1-sol (Supervisor Agent)`
- Repo/project: `antigravity-switch` (G:\antigravity_auto_switch)
- Scope reviewed: Commit `1ed7e6fe3c3eb2bb86c083431dbbf5fa6e3cf2b7`, `internal/manager/account_manager.go`, `cmd/switcher/main.go`, `internal/ui/app.go`, `internal/manager/account_manager_test.go`, `reports/coder/2026-10-16-fix-p02-eliminate-authlogout-report.md`
- Claim reviewed: P0-02 Parity Fix - Eliminate `AuthLogout` in switch flow to prevent browser OAuth popups, matching Python `core/state_resolver.py:380-425` where `should_logout = False` by default.
- Authority used:
  - Python baseline: `core/state_resolver.py:380-425`
  - Gap audit: `reports/supervisor/2026-10-16-phase6-python-go-full-parity-zero-trust-review.md` (Finding P0-02)
  - Coder report: `reports/coder/2026-10-16-fix-p02-eliminate-authlogout-report.md`
  - Supervisor Skill: `g:\antigravity_auto_switch\.agents\skills\supervisor\SKILL.md`
- Related artifacts:
  - Commit `1ed7e6fe3c3eb2bb86c083431dbbf5fa6e3cf2b7`

## Executive Summary
- Problem: In the previous implementation of Go `AccountManager.SwitchAccount`, inverted logic was used (`if !skipLogout && client != nil { _ = client.AuthLogout(ctx) }` with callers passing `skipLogout = false`), causing an RPC `AuthLogout` request to be sent to Language Server on every account hot-swap. This caused Language Server to alert Electron IDE that the user logged out, opening Chrome to the Google OAuth consent page. In Python baseline (`core/state_resolver.py:380-425`), hot-swap explicitly defaults `should_logout = False` to prevent any browser popups.
- Decision: PASS. Independent line-by-line inspection confirms that `client.AuthLogout` is no longer invoked during normal hot-swap. The parameter signature of `SwitchAccount` was changed to variadic `forceLogout ...bool` with default `shouldLogout = false`. Callers in both CLI (`cmd/switcher/main.go:91`) and GUI (`internal/ui/app.go:174`) explicitly supply `false` (and default zero-argument calls evaluate to `false`). Full static code verification confirmed no residual active `AuthLogout` triggers exist across the switch pipeline. Go vet and Windows build pipeline (`scripts/build_windows.ps1 -Mode all`) passed with zero errors and exit code 0.
- Required outcome: Accepted. Ready for P0-03 (`dismiss_tos_modal` / pre-patch ToS alignment).

## Source-Level Clearance Notes
- `internal/manager/account_manager.go:354, 361-364, 408`: CLEAR.
  - Line 354: Function signature changed to `func (m *AccountManager) SwitchAccount(ctx context.Context, targetEmail string, forceLogout ...bool) (*SwitchResult, error)`.
  - Lines 361-364: `var shouldLogout bool; if len(forceLogout) > 0 { shouldLogout = forceLogout[0] }` ensures `shouldLogout` defaults to `false` when omitted or when `false` is passed.
  - Line 408: `if shouldLogout && client != nil { _ = client.AuthLogout(ctx) }` guards `AuthLogout` so that it is never invoked during normal hot-swap, matching Python `core/state_resolver.py:440-445`.
- `cmd/switcher/main.go:91`: CLEAR.
  - `res, err := mgr.SwitchAccount(ctx, switchTarget, false)` explicitly passes `false` (`forceLogout = false`), preserving smooth zero-chrome hot-swap in CLI mode.
- `internal/ui/app.go:174`: CLEAR.
  - `res, err := a.mgr.SwitchAccount(ctx, targetEmail, false)` explicitly passes `false` (`forceLogout = false`), preserving smooth zero-chrome hot-swap in GUI mode.
- `internal/manager/account_manager_test.go:265-330`: CLEAR.
  - Mock Language Server HTTP server explicitly records `AuthLogout` hit count (`authLogoutCount`).
  - Unit tests verify `authLogoutCount == 0` for `SwitchAccount(ctx, targetEmail, false)`.
  - Unit tests verify `authLogoutCount == 0` for default variadic call `SwitchAccount(ctx, anotherEmail)`.
  - Unit tests verify `authLogoutCount == 1` only when `forceLogout = true` is explicitly passed.

## Evidence Checked
Passed:
- Source diff inspection (`git show 1ed7e6f`): Confirmed exact removal of inverted `!skipLogout` logic and introduction of `forceLogout ...bool` defaulting to `false`.
- Repository static code search (`rg "AuthLogout" -g "*.go"`): Confirmed that within `internal/manager/`, the only reference to `AuthLogout` is inside the `if shouldLogout && client != nil` block and in tests.
- Call site audit (`rg "SwitchAccount" -g "*.go"`): Confirmed all caller sites (`cmd/switcher/main.go`, `internal/ui/app.go`) pass `false`.
- Reloader audit (`rg "AuthLogout|logout" internal/reloader`): Confirmed zero calls to logout in `internal/reloader`.
- Dead code & compiler check (`go vet ./internal/manager/...`): Returned exit code 0, 0 warnings/errors.
- Repository-wide compiler check (`go vet ./...`): Returned exit code 0, 0 warnings/errors.
- Full Windows build pipeline (`powershell -File scripts/build_windows.ps1 -Mode all`):
  - Pure Go preflight: PASS
  - GUI CGO preflight: PASS
  - Win32 resource compilation (`windres.exe`): SUCCESS (`rsrc_windows_amd64.syso`)
  - CLI compilation: SUCCESS (`bin/antigravity-switch.exe`, 6.69 MB)
  - GUI compilation: SUCCESS (`bin/antigravity-switch-gui.exe`, 23.5 MB)
  - Embedded PE icon verification: PASS
  - Runtime smoke (`bin/antigravity-switch.exe --status`): Exit code 0
- Unit test verification (`go test -v -count=1 ./internal/manager/...`): PASS in 1.666s with all assertions satisfied.

## Invariant Closure
- Affected invariant: Zero-Chrome Hot-Swap Invariant — the account switcher must never invoke `AuthLogout` against the Language Server during normal account switching, preventing Electron from detecting a logout event and opening Google OAuth in Chrome.
- Sibling surfaces checked:
  - `internal/manager/account_manager.go` (core hot-swap logic)
  - `cmd/switcher/main.go` (CLI entry point)
  - `internal/ui/app.go` (GUI entry point)
  - `internal/reloader/session_reloader.go` (session reloading and UI orchestration)
  - `internal/connector/api_client.go` (API client definition)
- Residual unverified same-invariant surfaces: none.

## Overall Evaluation
The Coder fix at commit `1ed7e6fe3c3eb2bb86c083431dbbf5fa6e3cf2b7` completely closes gap P0-02. The inverted logic has been eliminated, the default behavior aligns 100% with Python `core/state_resolver.py:380-425`, and all call sites in CLI and GUI explicitly ensure `forceLogout = false`. The codebase compiles cleanly, passes `go vet` with 0 issues, and generates functional Windows binaries.
