# Coder Report: Elimination of AuthLogout in Switch Flow (P0-02 Parity Fix)

## 1. Metadata
- **Date**: 2026-10-16
- **Task ID**: P0-02
- **Author**: Coder Lane
- **Target Invariant**: Eliminate `AuthLogout` in switch flow to prevent Electron IDE logout notification and browser OAuth popups.
- **Authority / SSOT**:
  - `core/state_resolver.py:380-425`
  - `reports/supervisor/2026-10-16-phase6-python-go-full-parity-zero-trust-review.md` (§3.1, §5.2)

---

## 2. Invariant Family Map
- **Family Name**: Zero-Chrome Hot-Swap Session Preservation
- **SSOT**: `core/state_resolver.py:380-425` (`should_logout = False` by default for smooth zero-chrome hot-swap without OAuth popups).
- **Broken Surface (Identified)**:
  - In `internal/manager/account_manager.go:398-403`, Go previously used inverted logic `if !skipLogout && client != nil { _ = client.AuthLogout(ctx) }` with callers passing `skipLogout = false`, causing `client.AuthLogout(ctx)` to be invoked on every hot-swap.
  - Calling `AuthLogout` purges the Language Server's in-memory session, alerting Electron IDE that the user logged out, which automatically opens Chrome to the Google OAuth consent page.
- **Sibling Runtime Surfaces Inspected**:
  - `internal/manager/account_manager.go`: Parameter changed to `forceLogout ...bool` with default `shouldLogout = false`. RPC `AuthLogout` is bypassed unless `forceLogout == true` is explicitly provided.
  - `cmd/switcher/main.go`: Call site explicitly passes `false` (`forceLogout = false`) for smooth zero-chrome hot-swap.
  - `internal/ui/app.go`: GUI call site explicitly passes `false` (`forceLogout = false`).
  - `internal/manager/account_manager_test.go`: Added mock Language Server client and explicit assertions verifying `authLogoutCount == 0` during hot-swap.
  - `internal/reloader/session_reloader.go`: Inspected; verified zero calls to `AuthLogout`.
  - `internal/connector/api_client.go`: `AuthLogout` RPC primitive remains available for explicit maintenance or testing, but is no longer triggered by switch flow.
- **Legacy Fallback Status**: Completely eliminated. Hot-swap never triggers `AuthLogout` by default.
- **Residual Unverified Surfaces**: `none`

---

## 3. Changes Implemented
1. **`internal/manager/account_manager.go`**:
   - Updated `SwitchAccount(ctx context.Context, targetEmail string, forceLogout ...bool)` signature.
   - Evaluated `shouldLogout` defaulting to `false`.
   - Conditioned `_ = client.AuthLogout(ctx)` strictly on `if shouldLogout && client != nil`.
2. **`cmd/switcher/main.go`**:
   - Updated `mgr.SwitchAccount(ctx, switchTarget, false)` to explicitly document `forceLogout = false` (smooth zero-chrome hot-swap).
3. **`internal/ui/app.go`**:
   - Updated `a.mgr.SwitchAccount(ctx, targetEmail, false)` to document `forceLogout = false`.
4. **`internal/manager/account_manager_test.go`**:
   - Renamed from `manager_test.go` for canonical naming.
   - Wired mock Language Server HTTP test server to track `AuthLogout` invocations.
   - Verified that `mgr.SwitchAccount(..., false)` results in `authLogoutCount == 0`.
   - Verified that `mgr.SwitchAccount(...)` (default variadic) results in `authLogoutCount == 0`.
   - Verified that `mgr.SwitchAccount(..., true)` results in `authLogoutCount == 1`.

---

## 4. Verification Evidence

### 4.1. Unit Test Verification
```powershell
go test -v ./internal/manager/...
```
Output:
```
=== RUN   TestExtractEmailFromToken
--- PASS: TestExtractEmailFromToken (0.00s)
=== RUN   TestCaptureCurrentSession
--- PASS: TestCaptureCurrentSession (0.02s)
=== RUN   TestGetCurrentActiveEmail
--- PASS: TestGetCurrentActiveEmail (0.01s)
=== RUN   TestGetAccountStates
--- PASS: TestGetAccountStates (0.03s)
=== RUN   TestSwitchAccount
--- PASS: TestSwitchAccount (0.08s)
PASS
ok  	antigravity-switch/internal/manager	1.707s
```

### 4.2. Repository-Wide Test Suite
```powershell
go test ./...
```
Output:
```
?   	antigravity-switch/cmd/switcher	[no test files]
ok  	antigravity-switch/internal/connector	(cached)
ok  	antigravity-switch/internal/manager	2.239s
?   	antigravity-switch/internal/model	[no test files]
ok  	antigravity-switch/internal/reloader	(cached)
ok  	antigravity-switch/internal/scanner	(cached)
ok  	antigravity-switch/internal/store	4.680s
ok  	antigravity-switch/internal/ui	1.549s
ok  	antigravity-switch/internal/wincred	(cached)
```

### 4.3. Full Windows Compilation Pipeline
```powershell
powershell -File scripts/build_windows.ps1 -Mode all
```
Output:
- Pure Go verification (CGO_ENABLED=0): PASS
- GUI verification (CGO_ENABLED=1): PASS
- Windows Resource Compilation (`windres.exe`): SUCCESS (`rsrc_windows_amd64.syso`, 12552 bytes)
- CLI Binary Built: `binntigravity-switch.exe` (6.69 MB)
- GUI Binary Built: `binntigravity-switch-gui.exe` (23.5 MB)
- Embedded PE Icon Verification: PASS (32x32)
- Runtime Smoke (`binntigravity-switch.exe --status`): Exit code 0, session probed cleanly.

---

## 5. E2E Report Summary
```text
E2E Verification:
  [PASS] Compiled: powershell -File scripts/build_windows.ps1 -Mode all -> Exit 0
  [PASS] Runtime: binntigravity-switch.exe --status -> Exit 0
  [PASS] Happy path: SwitchAccount with forceLogout=false -> authLogoutCount == 0, creds updated
  [PASS] Default path: SwitchAccount with default args -> authLogoutCount == 0
  [PASS] Edge case: SwitchAccount with forceLogout=true -> authLogoutCount == 1
```

## 6. Verdict
**[VERDICT: PASS / READY_FOR_SUPERVISOR]**
All acceptance criteria for P0-02 have been completely met with zero regressions.
