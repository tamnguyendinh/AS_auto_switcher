# Antigravity Restart & Google Policy Modal Fix Evidence Ledger

## Metadata

- Date: `2026-10-16`
- Status: `P0 Complete / Baseline Verified`
- Plan: `docs/plans/2026-10-16-antigravity-restart-and-tos-fix/2026-10-16-antigravity-restart-and-tos-fix-plan.md`
- Actual status: `docs/plans/2026-10-16-antigravity-restart-and-tos-fix/2026-10-16-antigravity-restart-and-tos-fix-actual-status.md`
- Benchmark: `docs/plans/2026-10-16-antigravity-restart-and-tos-fix/2026-10-16-antigravity-restart-and-tos-fix-benchmark.md`

## Evidence Rules

- Evidence IDs must be exact and stable: `E<phase>-<item>-<kind><n>`.
- Phase 0 uses `E0-P0A-...` for baseline facts.
- Phase 1 slices use `E1-P1A-...`, `E1-P1B-...`, `E1-P1C-...`, `E1-P1D-...`.
- No speculative assertions; all entries must be grounded in real command outputs, code lines, or log excerpts.

## E0: Baseline Reality & Problem Evidence

### E0-P0A-RCA1: Root Cause Analysis Báo Cáo Chuyên Sâu
- Source: `reports/architect/2026-10-16-antigravity-restart-and-tos-modal-rca.md`.
- Summary: System Architect đã xác định 4 nguyên nhân gốc rễ dẫn đến hiện tượng Antigravity liên tục reload và popup Google Policy / ToS modal:
  1. Multi-Account Switcher và Test Suite lạm dụng `taskkill /F /IM language_server.exe` thô bạo khiến Electron coi là crash, kích hoạt `monitorLsCrashInternal` -> `onPortChanged` -> `window.loadURL` reload toàn bộ IDE.
  2. Token hết hạn nạp vào Windows Credential Manager khiến `language_server.exe` crash `os.Exit(1)` sau 40-50s, tạo vòng lặp crash loop.
  3. Lệch pha `jetski.onboarding.lastLoginUsername` trong `%APPDATA%\Antigravity\app_storage.json` kích hoạt component `ACb` render modal `zCb` (ToS dialog) với lý do `"account_changed"`.
  4. Test suite `internal/reloader/reloader_test.go::TestDefaultKillLanguageServer` chạy `taskkill` thật tiêu diệt IDE trên máy thật mỗi khi chạy test.

### E0-P0A-LOG1: Electron Log Crash & Window Reload Evidence
- Source: `C:\Users\TAM NGUYEN\AppData\Roaming\Antigravity\logs\main.log`.
- Log Lines:
  ```log
  [2026-10-16 17:34:26.934] [error] Language server crashed: exited with code 1
  [2026-10-16 17:34:26.935] [info]  Attempting restart 1/3 in 2s...
  [2026-10-16 17:34:29.280] [info]  [Auto-Restart] Port changed! Reloading all windows with URL: https://127.0.0.1:50487/
  [2026-10-16 17:35:17.541] [error] Language server crashed: exited with code 1
  [2026-10-16 17:35:17.542] [info]  Attempting restart 2/3 in 2s...
  [2026-10-16 17:35:19.717] [info]  [Auto-Restart] Port changed! Reloading all windows with URL: https://127.0.0.1:50587/
  ```
- Fact: Chứng minh mỗi khi `language_server.exe` bị kill hoặc crash, Electron sẽ tự động reload toàn bộ các cửa sổ Antigravity (`window.loadURL`).

### E0-P0A-SRC1: Real Taskkill Invocation in Test Suite
- Source: `internal/reloader/reloader_test.go:183-193`.
- Code snippet:
  ```go
  func TestDefaultKillLanguageServer(t *testing.T) {
      ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
      defer cancel()
      err := DefaultKillLanguageServer(ctx)
      if err != nil {
          t.Fatalf("DefaultKillLanguageServer failed: %v", err)
      }
  }
  ```
- Fact: Lệnh `DefaultKillLanguageServer` chạy trực tiếp `taskkill /F /IM language_server.exe` trên hệ điều hành của lập trình viên mà không hề được mock.

### E0-P0A-SRC2: Taskkill in SwitchAccount Workflow
- Source: `internal/manager/account_manager.go:460-465`.
- Code snippet:
  ```go
  if rel != nil {
      recycleErr = rel.RecycleLanguageServer(ctx, 2*time.Second)
  } else {
      recycleErr = reloader.RecycleLanguageServer(ctx, 2*time.Second)
  }
  ```
- Fact: `RecycleLanguageServer` mặc định gọi `killer = DefaultKillLanguageServer` (`taskkill /F /IM language_server.exe`), cưỡng ép crash language server thay vì dùng RPC ReAuth.

### E0-P0A-SRC3: RawPayload Masking in TokenData Serialization
- Source: `internal/model/types.go:121-125`.
- Code snippet:
  ```go
  func (t *TokenData) MarshalJSON() ([]byte, error) {
      if len(t.RawPayload) > 0 {
          return t.RawPayload, nil
      }
      ...
  }
  ```
- Fact: Nếu chỉ gán giá trị mới cho `t.AccessToken` và `t.Expiry` mà không xóa hoặc cập nhật `t.RawPayload`, `MarshalJSON()` sẽ trả về `RawPayload` cũ với token đã hết hạn, dẫn đến ghi token chết vào WinCred.

### E0-P0A-EXP1: Stored Account with Expired Token
- Source: `accounts_store/daohien204ccyersin@gmail.com.json`.
- Payload verification:
  - `expiry: 2026-10-16T14:30:50.1769557+07:00`.
  - Lúc 17:34 (sau 3 tiếng hết hạn), token này được nạp vào Windows Credential Manager, gây crash `language_server.exe` và crash loop 40-50s.

### E0-P0A-API1: Google OAuth Refresh API Parameter Verification
- Test Command & Parameters:
  - Endpoint: `https://oauth2.googleapis.com/token` (Method: POST)
  - `client_id`: `1071006060591-tmhssin2h21lcre235vtolojh4g403ep.apps.googleusercontent.com`
  - `client_secret`: `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`
  - `grant_type`: `refresh_token`
  - `refresh_token`: (Lấy từ `accounts_store/daohien204ccyersin@gmail.com.json`)
- Output / Response:
  - HTTP Status: `200 OK`
  - `expires_in`: `3599` (giây = ~1 giờ)
  - `token_type`: `"Bearer"`
  - `access_token`: Cấp phát access_token mới tinh, hợp lệ 100%.
- Fact: Đã kiểm chứng thực tế không giả lập: Endpoint và thông số client OAuth của Antigravity CloudCode hoàn toàn hoạt động và cấp token mới thành công!

### E0-P0A-TOS1: Current app_storage.json Stale Email Triggering ToS Modal
- Source: `%APPDATA%\Antigravity\app_storage.json`.
- Inspected content:
  ```json
  "jetski.onboarding.lastLoginIsGcpTos": "false",
  "jetski.onboarding.lastLoginUsername": "switch-continuation@example.com",
  ```
- Fact: Giá trị `lastLoginUsername` bị gán test email `switch-continuation@example.com`. Khi Antigravity nạp phiên người dùng thực (`hiendao1760@gmail.com`), so khớp `stored.email !== currentEmail` thất bại, hàm `xCb` trả về `"account_changed"`, kích hoạt modal `zCb` (ToS dialog) trên màn hình!

### E0-P0A-RPC1: Pre-existing TriggerReauth Client Method
- Source: `internal/connector/api_client.go:215-245`.
- Verification: Method `TriggerReauth(ctx context.Context) error` gửi Connect-RPC POST `b"{}"` tới `DefaultReauthEndpoints` (`/exa.language_server_pb.LanguageServerService/ReAuth` và `/Reload`) với headers `X-Csrf-Token` và `x-codeium-csrf-token`. Đã có sẵn trong codebase nhưng chưa được `AccountManager` gọi.

---

## Planned Evidence Targets for Execution

### E1 (Slice 1: Test Isolation Mock)
- `E1-P1A-MOCK1`: Verification that `TestDefaultKillLanguageServer` no longer executes OS taskkill.
- `E1-P1A-TEST1`: Unit test run `go test -v ./internal/reloader/...` passing without affecting running `language_server.exe`.

### E2 (Slice 2: Token Expiry Check & OAuth Refresh)
- `E2-P1B-REFRESH1`: Unit test verifying `oauth_refresher.go` exchanges refresh_token for a fresh access_token.
- `E2-P1B-MARSH1`: Unit test verifying `TokenData.UpdateToken` updates both fields and invalidates `RawPayload`.
- `E2-P1B-MGR1`: Integration test verifying `SwitchAccount` triggers refresh when token expiry is < 5 minutes.

### E3 (Slice 3: Atomic app_storage.json Sync)
- `E3-P1C-SYNC1`: Unit test verifying atomic write with fsync/flush and proper permission handling for `app_storage.json`.
- `E3-P1C-GCP1`: Unit test verifying correct `lastLoginIsGcpTos` value matching account's `auth_method`.

### E4 (Slice 4: RPC ReAuth Graceful Fallback)
- `E4-P1D-RPC1`: Unit test verifying `SwitchAccount` calls `TriggerReauth` first when language server is running.
- `E4-P1D-FALLBACK1`: Unit test verifying fallback to `RecycleLanguageServer` occurs only when `TriggerReauth` fails or LS is absent.
- `E4-P1D-VERIFY1`: Integration test confirming `SwitchResult` returns success without window reload when RPC ReAuth succeeds.


---

## Phase 1 Execution Evidence

### E1-P1B-OAUTH1: TokenData.UpdateTokens Invalidation of RawPayload
- Source: `internal/model/types.go:168-187` & `internal/model/types_test.go:TestTokenData_UpdateTokens`.
- Verification command: `go test -v ./internal/model/... -run TestTokenData_UpdateTokens` -> PASS (0.00s).
- Evidence: `UpdateTokens` correctly replaces AccessToken, RefreshToken, IDToken, Expiry and resets `RawPayload = nil`. Subsequent `json.Marshal` produces valid dual-envelope JSON containing the refreshed access token and does not emit old RawPayload bytes.

### E1-P1B-REFRESH1: Google OAuth Refresh Engine Implementation & Test Suite
- Source: `internal/connector/oauth_refresher.go` & `internal/connector/oauth_refresher_test.go`.
- Verification command: `go test -v ./internal/connector/... -run TestGoogleOAuthRefresher` -> PASS (5 sub-tests, 0.00s).
- Tests passed:
  - `TestGoogleOAuthRefresher_Success`: verifies HTTP POST form encoding with Google CloudCode client credentials and parsing of access_token, expires_in, token_type, and expiry.
  - `TestGoogleOAuthRefresher_HttpError`: handles HTTP 400 with invalid_grant properly.
  - `TestGoogleOAuthRefresher_NetworkFailure`: handles transport/network errors gracefully.
  - `TestGoogleOAuthRefresher_MalformedJSON`: handles unparseable JSON without panicking.
  - `TestGoogleOAuthRefresher_EmptyRefreshToken`: rejects empty refresh token with ErrEmptyRefreshToken.

### E1-P1B-MGR1: Proactive Expiry Check & Disk/WinCred Auto-Refresh in SwitchAccount
- Source: `internal/manager/account_manager.go:433-455` & `internal/manager/account_manager_test.go:TestSwitchAccount_ProactiveTokenRefresh`.
- Verification command: `go test -v ./internal/manager/... -run TestSwitchAccount_ProactiveTokenRefresh` -> PASS (5.13s).
- Evidence:
  - Account with expired token triggers proactive refresh call via OAuthRefresher.
  - Refreshed token is saved to account store on disk (`accounts_store/<email>.json`) with updated expiry.
  - Refreshed token is written to Windows Credential Manager.
  - Account with valid token (> 5 min left) does not trigger refresh.
  - Network failure on refresh does not crash or abort, but proceeds gracefully with existing token.
