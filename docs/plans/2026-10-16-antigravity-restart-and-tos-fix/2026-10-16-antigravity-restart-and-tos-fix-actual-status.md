# Antigravity Restart & Google Policy Modal Fix Actual Status

Title: Antigravity Restart & Google Policy Modal Fix Actual Status
Date: 2026-10-16
Status: P0 Complete
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

## Current Status Matrix

| Surface / Path | Current Classification | Reality Summary | Evidence IDs | Touch Mode | Plan Consequence |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `internal/reloader/reloader_test.go::TestDefaultKillLanguageServer` | `wrong` | Test gọi trực tiếp `DefaultKillLanguageServer(ctx)` bắn hạ `language_server.exe` trên máy thật của lập trình viên khi chạy `go test`. | `E0-P0A-SRC1` | Edit | Thay thế bằng mock process killer; bảo vệ 100% IDE khỏi lệnh taskkill trong mọi test. |
| `internal/reloader/session_reloader.go::DefaultKillLanguageServer` | `partial` | Hiện tại thực thi `taskkill /F /IM language_server.exe`. Đây là cơ chế thô bạo khiến Electron xem là crash và reload all windows. | `E0-P0A-SRC2`, `E0-P0A-LOG1` | Edit | Đổi thành fallback cuối cùng khi RPC ReAuth thất bại; không dùng làm phương thức hot-swap mặc định. |
| `internal/model/types.go::TokenData` | `partial` | `MarshalJSON()` ưu tiên ghi `RawPayload` nguyên bản, dẫn đến việc nếu cập nhật `AccessToken` và `Expiry` mới thì dữ liệu cũ vẫn bị ghi ra đĩa/WinCred. | `E0-P0A-SRC3` | Edit | Bổ sung hàm cập nhật hoặc xóa `RawPayload` khi refresh token để `MarshalJSON()` xuất ra token mới. |
| `internal/connector/oauth_refresher.go` | `missing` | Chưa có module chuyên biệt thực hiện HTTP POST `https://oauth2.googleapis.com/token` bằng Google CloudCode Client ID & Secret đã xác minh. | `E0-P0A-API1` | Create | Tạo module `oauth_refresher.go` với Client ID `1071006060591-tmhssin2h21lcre235vtolojh4g403ep.apps.googleusercontent.com` và Secret `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`. |
| `internal/manager/account_manager.go::SwitchAccount` | `partial` | `SwitchAccount` ghi token vào WinCred mà không kiểm tra hạn (expiry); tự động gọi `RecycleLanguageServer` thô bạo; gọi `PrepatchToS` nhưng không bảo đảm atomic sync trước khi reload/reauth. | `E0-P0A-SRC2`, `E0-P0A-EXP1` | Edit | Tái cấu trúc luồng: (1) Kiểm tra & Refresh Token nếu < 5 phút; (2) Lưu token mới vào AccountStore; (3) Đồng bộ nguyên tử `app_storage.json`; (4) Ghi WinCred; (5) Thử RPC `TriggerReauth`; chỉ fallback recycle nếu ReAuth thất bại. |
| `internal/reloader/session_reloader.go::PrePatchToSAcceptance` | `partial` | Ghi đè `jetski.onboarding.lastLoginUsername` và `jetski.onboarding.lastLoginIsGcpTos` nhưng chưa đồng bộ chặt chẽ với auth_method và thiếu guard file locking/atomic sync hoàn chỉnh. | `E0-P0A-TOS1` | Edit | Chuẩn hóa hàm ghi atomic sync, hỗ trợ tham số `isGcpTos` chuẩn theo `auth_method` của tài khoản, flush I/O triệt để. |
| `internal/connector/api_client.go::TriggerReauth` | `bound-correct` | Đã triển khai sẵn hàm `TriggerReauth(ctx)` gọi danh sách `DefaultReauthEndpoints` (`/exa.language_server_pb.LanguageServerService/ReAuth`), nhưng chưa được `AccountManager.SwitchAccount` tích hợp. | `E0-P0A-RPC1` | Preserve / Integrate | Giữ nguyên implementation chuẩn, tích hợp vào `AccountManager.SwitchAccount` làm First-Class ReAuth mechanism. |

## Status Refresh Log

| Refresh ID | Trigger / Slice | Rows Updated | Old Status -> New Status | Evidence IDs | Plan Consequence |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `SRL-P0` | Baseline audit for Antigravity Restart & ToS Fix | All rows | Baseline audit -> P0 Complete | `E0-P0A-RCA1` .. `E0-P0A-RPC1` | Baseline established. Ready for Slice 1 (P1-A). |

