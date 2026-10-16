# Antigravity Restart & Google Policy Modal Fix Actual Status

Title: Antigravity Restart & Google Policy Modal Fix Actual Status
Date: 2026-10-16
Status: in_progress (Slice 1, 2 & 3 complete ➔ Next: Slice 4 P1-D)
Companion plan: `docs/plans/2026-10-16-antigravity-restart-and-tos-fix/2026-10-16-antigravity-restart-and-tos-fix-plan.md`
Companion rules: `g:\antigravity_auto_switch\.agents\skills\planner\templates\plan.template.md`
Companion evidence: `docs/plans/2026-10-16-antigravity-restart-and-tos-fix/2026-10-16-antigravity-restart-and-tos-fix-evidence.md`
Companion benchmark: `docs/plans/2026-10-16-antigravity-restart-and-tos-fix/2026-10-16-antigravity-restart-and-tos-fix-benchmark.md`

## Purpose

This file records the real current state before implementation.

Implementation must not start until the target scope has a completed status row, evidence IDs, and a downstream plan decision.

This file does not replace `evidence.md`. It classifies current state from evidence.

Use exact evidence IDs from `evidence.md`, such as `E0-P0A-RCA1`, not broad section IDs such as `E0` or `E1`.

## Freshness / Refresh Rules

This actual-status file is a living current-state record, not a one-time P0 snapshot.

P0 records the baseline before implementation. After implementation begins, keep the Current Status Matrix updated so the next agent can trust it as the latest repo reality.

Update this file:

- after each completed implementation slice;
- before starting the next phase if repo state changed;
- whenever evidence changes a current-state classification;
- whenever the next phase's status assumptions, next action, or work steps need updating because reality differs from the previous status.

When refreshing status:

- update only the rows affected by the completed work or new evidence;
- use explicit transitions such as `missing -> correct`, `partial -> correct`, `fake-or-stub -> removed`, or `unbound -> bound-correct`;
- append a Status Refresh Log row instead of deleting history;
- keep detailed proof in `evidence.md`; store only classifications, evidence IDs, touch mode, and plan consequences here.

## Scope

Target scope:

- Trụ cột 1: `internal/reloader/reloader_test.go` (Cách ly Test Suite khỏi OS thực).
- Trụ cột 2: `internal/model/types.go`, `internal/connector/oauth_refresher.go` (mới), `internal/store/account_store.go`, `internal/manager/account_manager.go` (Cơ chế refresh OAuth token trước khi ghi WinCred).
- Trụ cột 3: `internal/reloader/session_reloader.go`, `internal/manager/account_manager.go` (Đồng bộ nguyên tử `app_storage.json`).
- Trụ cột 4: `internal/connector/api_client.go`, `internal/manager/account_manager.go`, `internal/reloader/session_reloader.go` (RPC ReAuth endpoint `/exa.language_server_pb.LanguageServerService/ReAuth` thay cho `taskkill /F`).

Out of scope:

- Mã nguồn UI window/tray (`internal/ui/`).
- Scanner Chrome (`internal/scanner/`).
- WinCred format low-level API (`internal/wincred/cred.go`).
- Python core CLI cũ nếu không nằm trong danh mục bàn giao.

## Slice Execution Status

- [x] Slice 1 (P1-A): Test Suite Isolation & Mock Process Killer (PASS in commit `8289c28`)
- [x] Slice 2 (P1-B): Proactive Token Expiry Verification & Google OAuth Refresh Engine (PASS in commit 9492736)
- [x] Slice 3 (P1-C): Atomic app_storage.json & Session Synchronization (PASS in commit b3cd1f8)
- [ ] Slice 4 (P1-D): Civilized RPC ReAuth Hot-Swap with Safe Fallback (Active Next)

## Current Status Matrix

| Surface / Path | Current Classification | Reality Summary | Evidence IDs | Touch Mode | Plan Consequence |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `internal/reloader/reloader_test.go::TestDefaultKillLanguageServer` | `correct` | Test được cách ly hoàn toàn qua mock process killer và TestMain safety barrier; không lệnh taskkill thật nào được gọi vào OS khi chạy test. | `E1-P1A-MOCK1`, `E1-P1A-TEST1` | Preserve | Slice 1 (P1-A) hoàn thành (commit `8289c28`); bảo vệ 100% IDE khỏi lệnh taskkill trong mọi test. |
| `internal/reloader/session_reloader.go::DefaultKillLanguageServer` | `partial` | Hiện tại thực thi `taskkill /F /IM language_server.exe`. Đây là cơ chế thô bạo khiến Electron xem là crash và reload all windows. | `E0-P0A-SRC2`, `E0-P0A-LOG1` | Edit | Đổi thành fallback cuối cùng khi RPC ReAuth thất bại; không dùng làm phương thức hot-swap mặc định. |
| `internal/model/types.go::TokenData` | `correct` | `UpdateTokens` cập nhật các trường token và xóa `RawPayload = nil`, đảm bảo `MarshalJSON()` xuất ra token mới chính xác. | `E1-P1B-OAUTH1`, `E1-P1B-TEST1` | Preserve | Slice 2 (P1-B) hoàn thành; kiểm thử unit test TestTokenData_UpdateTokens PASS. |
| `internal/connector/oauth_refresher.go` | `correct` | Module `GoogleOAuthRefresher` triển khai interface `OAuthRefresher`, hỗ trợ custom HTTP client và xử lý refresh token chuẩn Google OAuth2. | `E1-P1B-OAUTH2`, `E1-P1B-TEST2` | Preserve | Slice 2 (P1-B) hoàn thành; kiểm thử TestGoogleOAuthRefresher_* PASS 100%. |
| `internal/manager/account_manager.go::SwitchAccount` | `partial` | Đã tích hợp proactive token refresh (< 5 phút), đồng bộ nguyên tử app_storage.json trước WinCred write. Còn chờ Slice 4 tích hợp Connect-RPC ReAuth. | `E1-P1B-STORE1`, `E3-P1C-SYNC2` | Edit | Bước refresh token và đồng bộ ToS hoàn tất; sẵn sàng cho Slice 4 (P1-D: Civilized RPC ReAuth Hot-Swap with Safe Fallback). |
| `internal/reloader/session_reloader.go::SyncAppStorageToS` | `correct` | Đã triển khai SyncAppStorageToS ghi nguyên tử (temp file + fsync + os.Rename), xử lý retry backoff cho Windows sharing violation, bảo toàn 100% key khác và hỗ trợ dynamic isGcpTos. | `E3-P1C-SYNC1`, `E3-P1C-TEST1` | Preserve | Slice 3 (P1-C) hoàn thành (commit `b3cd1f8`). Triệt tiêu Google Cloud ToS modal prompt (B-M04 = 0). |
| `internal/connector/api_client.go::TriggerReauth` | `bound-correct` | Đã triển khai sẵn hàm `TriggerReauth(ctx)` gọi danh sách `DefaultReauthEndpoints` (`/exa.language_server_pb.LanguageServerService/ReAuth`), nhưng chưa được `AccountManager.SwitchAccount` tích hợp. | `E0-P0A-RPC1` | Preserve / Integrate | Giữ nguyên implementation chuẩn, tích hợp vào `AccountManager.SwitchAccount` làm First-Class ReAuth mechanism. |

## Status Refresh Log

| Refresh ID | Trigger / Slice | Rows Updated | Old Status -> New Status | Evidence IDs | Plan Consequence |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `SRL-P0` | Baseline audit for Antigravity Restart & ToS Fix | All rows | Baseline audit -> P0 Complete | `E0-P0A-RCA1` .. `E0-P0A-RPC1` | Baseline established. Ready for Slice 1 (P1-A). |
| `SRL-P1A` | Slice 1 (P1-A: Test Suite Isolation & Mock Process Killer) completion (commit `8289c28`) | `internal/reloader/reloader_test.go::TestDefaultKillLanguageServer` | `wrong` -> `correct` | `E1-P1A-MOCK1`, `E1-P1A-TEST1` | Slice 1 (P1-A) PASS. Proceed to Slice 2 (P1-B: Proactive Token Expiry Verification & Google OAuth Refresh Engine). |

| `SRL-P1B` | Slice 2 (P1-B: Proactive Token Expiry Verification & Google OAuth Refresh Engine) completion | 3 rows | partial/missing -> correct | `E1-P1B-OAUTH1`, `E1-P1B-TEST3` | Slice 2 (P1-B) PASS. Proceed to Slice 3 (P1-C: Atomic app_storage.json & Session Synchronization). |
| SRL-P1B-PASS | Slice 2 (P1-B) Supervisor PASS (commit 9492736) | Current Status Matrix | verified correct | E1-P1B-OAUTH1..E1-P1B-MGR1 | Slice 2 PASS. Proceed to Slice 3 (P1-C). |
| SRL-P1C-PASS | Slice 3 (P1-C) Supervisor PASS (commit b3cd1f8) | Current Status Matrix | verified correct | E3-P1C | Slice 3 PASS. Proceed to Slice 4 (P1-D). |


