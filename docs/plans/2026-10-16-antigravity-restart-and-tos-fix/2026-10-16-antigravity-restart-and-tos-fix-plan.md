# Antigravity Restart & Google Policy Modal Fix Plan

## Metadata

- Date: `2026-10-16`
- Status: `ready_for_review`
- Plan: `docs/plans/2026-10-16-antigravity-restart-and-tos-fix/2026-10-16-antigravity-restart-and-tos-fix-plan.md`
- Evidence: `docs/plans/2026-10-16-antigravity-restart-and-tos-fix/2026-10-16-antigravity-restart-and-tos-fix-evidence.md`
- Benchmark: `docs/plans/2026-10-16-antigravity-restart-and-tos-fix/2026-10-16-antigravity-restart-and-tos-fix-benchmark.md`
- Actual status: `docs/plans/2026-10-16-antigravity-restart-and-tos-fix/2026-10-16-antigravity-restart-and-tos-fix-actual-status.md`

## Goal

Chuyển hóa phương án khắc phục dứt điểm sự cố Antigravity Restart & Google Policy Modal (theo 4 trụ cột trong Báo cáo RCA `reports/architect/2026-10-16-antigravity-restart-and-tos-modal-rca.md`) thành một bản Kế hoạch Hành động chi tiết, phân rã thành 4 Slice thực thi độc lập:
1. **Slice 1 (P1-A):** Cách ly tuyệt đối Test Suite khỏi môi trường thực (Mock process killer trong `internal/reloader/reloader_test.go`, triệt tiêu hoàn toàn lệnh `taskkill` thật bắn hạ IDE).
2. **Slice 2 (P1-B):** Cơ chế chủ động Refresh Google OAuth Token trước khi ghi vào Windows Credential Manager và `accounts_store/<email>.json` khi token hết hạn hoặc sắp hết hạn (< 5 phút), ngăn ngừa nạp token chết vào RAM gây crash `language_server.exe`.
3. **Slice 3 (P1-C):** Đồng bộ hóa nguyên tử `%APPDATA%\Antigravity\app_storage.json` (`jetski.onboarding.lastLoginUsername` và `jetski.onboarding.lastLoginIsGcpTos`) trước khi reload để triệt tiêu hoàn toàn modal Google Policy / ToS.
4. **Slice 4 (P1-D):** Nâng cấp cơ chế Re-auth / Hot-swap văn minh qua RPC endpoint `/exa.language_server_pb.LanguageServerService/ReAuth` thay vì thô bạo `taskkill /F`.

## Rules

- Hoàn thành P0 actual status trước khi tiến hành thực thi code.
- Mỗi phase/slice phải tuân thủ nghiêm ngặt chu kỳ thực thi 6 bước (`Code → Inspect → Update tests → Build → QA → Acceptance/Commit`).
- Bước 1 (Code) phải được chia nhỏ thành các leaf atomic task có Input, Edit Scope, Output, Điều kiện kiểm chứng và Task Checkpoint rõ ràng.
- Tuyệt đối không sửa mã nguồn ngoài Touch-Map đã khóa cho từng Slice.
- Không tự tiện sáng tác thêm tính năng mới ngoài 4 trụ cột kiến trúc.
- Không phá vỡ định dạng payload Windows Credential Manager (`LegacyGeneric:target=gemini:antigravity`).
- Cấm lạm dụng fallback ẩn (hidden fallback). Lỗi từ các bước quan trọng phải được ghi nhận rõ ràng.

## Problem

Theo điều tra chuyên sâu từ RCA `reports/architect/2026-10-16-antigravity-restart-and-tos-modal-rca.md`:
1. **Hiện tượng Restart liên tục (~40-60s interval):**
   - Cơ chế hot-swap cũ lạm dụng lệnh `taskkill /F /IM language_server.exe`. Electron Antigravity coi việc tiến trình biến mất bất thường là unhandled crash (`_intentionalTermination === false`), kích hoạt bộ đếm `monitorLsCrashInternal` sau 2 giây, gọi `onPortChanged` và thực thi `window.loadURL(newUrl)` cưỡng chế reload toàn bộ cửa sổ IDE.
   - Nghiêm trọng hơn, file `internal/reloader/reloader_test.go::TestDefaultKillLanguageServer` gọi trực tiếp `DefaultKillLanguageServer(ctx)` bắn hạ IDE thật của lập trình viên mỗi khi test suite chạy.
   - Đồng thời, khi chuyển sang tài khoản có token đã hết hạn (`accounts_store/daohien204ccyersin@gmail.com.json` hết hạn lúc 14:30), `language_server.exe` không thể xác thực với Google API -> gọi `os.Exit(1)` -> kích hoạt crash loop 40-50s cho đến khi đạt giới hạn 3 lần và IDE đóng hẳn.
2. **Hiện tượng Popup Modal Google Policy / Terms of Service (ToS modal):**
   - Trong React bundle (`/main.js`), component `ACb` so sánh email của phiên hiện tại với `jetski.onboarding.lastLoginUsername` trong `app_storage.json`.
   - Do `app_storage.json` bị lệch pha (đang chứa test email `switch-continuation@example.com` hoặc chưa được flush xuống đĩa), điều kiện so khớp thất bại (`stored.email !== currentEmail`), kích hoạt modal `zCb` với lý do `"account_changed"`. Người dùng bị chặn hoàn toàn, buộc phải bấm "Accept and continue" mới tiếp tục được.

## Scope

### In-Scope (Touch-Map cho 4 Slices)
- Slice 1: `internal/reloader/reloader_test.go`, `internal/reloader/session_reloader.go`.
- Slice 2: `internal/connector/oauth_refresher.go` (tạo mới), `internal/connector/oauth_refresher_test.go` (tạo mới), `internal/model/types.go`, `internal/store/account_store.go`, `internal/manager/account_manager.go`.
- Slice 3: `internal/reloader/session_reloader.go`, `internal/reloader/reloader_test.go`, `internal/manager/account_manager.go`.
- Slice 4: `internal/manager/account_manager.go`, `internal/manager/account_manager_test.go`, `internal/connector/api_client.go`.

### Out-of-Scope (Non-Goals)
- CẤM sửa bất kỳ mã nguồn nào trong giao diện đồ họa UI window/tray (`internal/ui/`).
- CẤM sửa scanner phát hiện profile Chrome (`internal/scanner/`).
- CẤM thay đổi cấu trúc dữ liệu WinCred (`LegacyGeneric:target=gemini:antigravity`).
- CẤM gọi `taskkill` trên OS thực trong bất kỳ unit/integration test nào.
- CẤM thêm thư viện bên ngoài không cần thiết.

## Technical Invariants

1. **Test Isolation Invariant:** Mọi unit/integration test trong `internal/reloader/` và toàn bộ repo phải chạy 100% trong môi trường cô lập, sử dụng mock process killer. Tuyệt đối không có bất kỳ lệnh taskkill nào được gửi tới OS thật.
2. **Token Freshness Invariant:** Mọi token được ghi vào Windows Credential Manager hoặc nạp vào Language Server RAM phải có thời hạn sử dụng còn lại tối thiểu 5 phút (`time.Until(expiry) >= 5 * time.Minute`). Nếu nhỏ hơn, bắt buộc phải refresh qua Google OAuth endpoint `https://oauth2.googleapis.com/token` trước khi bàn giao.
3. **Payload Serialization Fidelity Invariant:** Khi cập nhật `AccessToken` và `Expiry`, `TokenData.RawPayload` phải được xóa hoặc đồng bộ để `MarshalJSON()` không xuất dữ liệu cũ đã hết hạn.
4. **Atomic Storage Invariant:** `%APPDATA%\Antigravity\app_storage.json` phải được ghi theo cơ chế atomic (file tạm + fsync + atomic rename) và hoàn tất trước khi phát tín hiệu ReAuth hoặc reload, bảo đảm Electron đọc được giá trị `lastLoginUsername` và `lastLoginIsGcpTos` chính xác 100%.
5. **Civilized ReAuth Invariant:** Ưu tiên số 1 khi đổi tài khoản là gửi Connect-RPC POST tới `/exa.language_server_pb.LanguageServerService/ReAuth`. Chỉ fallback sang recycle tiến trình nếu Language Server không phản hồi hoặc cổng RPC không khả dụng.

## Acceptance Criteria (Zero-Trust Definition of Done)

- **AC1 (Test Isolation):** `go test -v ./internal/reloader/...` chạy qua 100% mà không làm gián đoạn bất kỳ tiến trình `language_server.exe` hay `Antigravity.exe` nào trên máy.
- **AC2 (Token Refresh):** Kiểm tra hạn token tự động phát hiện token hết hạn (như `daohien204ccyersin@gmail.com`), tự động gọi `oauth_refresher` lấy token mới có hạn 3599s, cập nhật `accounts_store/daohien204ccyersin@gmail.com.json` và ghi vào WinCred an toàn.
- **AC3 (Zero ToS Modal):** Trước khi switch tài khoản, `app_storage.json` được cập nhật đồng bộ cả email mục tiêu và `lastLoginIsGcpTos`. Mở IDE hoặc reload không xuất hiện modal Google Policy / ToS popup.
- **AC4 (Zero Window Reload):** Khi `language_server.exe` đang chạy, `SwitchAccount` kích hoạt RPC `TriggerReauth` thành công, `GetUserStatus` phản hồi email mới, tiến trình không bị kill, Electron không phát hiện crash, IDE không bị chớp giật hay reload toàn bộ cửa sổ.

---

## Phase Checklist

### P0: Baseline Actual Status & Environment Verification

- [x] P0-A: Complete actual status before implementation work.
  - Goal: Xác lập hiện trạng thực tế mã nguồn, đối chiếu với các phát hiện trong báo cáo RCA.
  - Work Steps:
    1. Đọc và phân tích `reports/architect/2026-10-16-antigravity-restart-and-tos-modal-rca.md`.
    2. Thẩm tra `internal/reloader/reloader_test.go:183` phát hiện lệnh `DefaultKillLanguageServer` không qua mock.
    3. Thẩm tra `internal/model/types.go` phát hiện `RawPayload` ưu tiên ghi đè dữ liệu cũ.
    4. Thẩm tra endpoint Google OAuth `https://oauth2.googleapis.com/token` với Client ID `1071006060591-tmhssin2h21lcre235vtolojh4g403ep.apps.googleusercontent.com` và Secret `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`, xác nhận trả về `200 OK`, `expires_in: 3599`.
    5. Kiểm tra `%APPDATA%\Antigravity\app_storage.json` phát hiện giá trị `jetski.onboarding.lastLoginUsername` bị gán test email `switch-continuation@example.com`.
    6. Tạo bộ tài liệu chuẩn: `actual-status.md`, `evidence.md`, `benchmark.md`.
  - Implementation Gate: `actual-status.md` hoàn thành phân loại baseline đầy đủ; chứng cứ E0 xác lập 100%.
  - Acceptance: Toàn bộ hiện trạng mã nguồn được phân loại chính xác, sẵn sàng cho P1.

---

### P1: Implementation Slices

#### Slice 1 (P1-A): Test Suite Isolation & Mock Process Killer

- **Goal:** Cách ly 100% test suite khỏi hệ điều hành thực tế. Triệt tiêu hoàn toàn lệnh `taskkill` thật trong `TestDefaultKillLanguageServer` và toàn bộ unit test của `internal/reloader/`.
- **Scope Boundary:**
  - Editable: `internal/reloader/reloader_test.go`, `internal/reloader/session_reloader.go`.
  - Inspect-only: `internal/manager/account_manager.go`.
  - Out of scope: Mọi file ngoài package `reloader`.
- **Non-Goals:**
  - Không xóa bỏ chức năng `DefaultKillLanguageServer` (vẫn cần cho trường hợp khẩn cấp trên runtime thực), chỉ thay đổi cách kiểm thử và cho phép dependency injection hoàn toàn.
- **Pre-flight Questions:**
  - Data source: Test context và function pointers trong `SessionReloader`.
  - Runtime targets: Go test runner (`go test ./internal/reloader/...`).
  - Cleanup/quarantine: Không để lại tiến trình zombie, không tác động process thực.
- **Work Steps (6-Step Lifecycle):**
  1. **Code:**
     - *Leaf Task 1.1:* Trong `internal/reloader/session_reloader.go`, bổ sung biến function pointer cấp package `var osTaskKill = defaultOsTaskKill` để cho phép test override lệnh thực thi shell `taskkill`.
       - Inputs: `session_reloader.go:70-78`.
       - Allowed Edit Scope: `internal/reloader/session_reloader.go`.
       - Outputs: Package-level hook `osTaskKill func(ctx context.Context) error`.
       - Verification Condition: `DefaultKillLanguageServer` gọi qua `osTaskKill`.
       - Checkpoint: `wip(P1-A): task 1.1 - introduce injectable osTaskKill hook in session_reloader.go`.
     - *Leaf Task 1.2:* Trong `internal/reloader/reloader_test.go`, tái cấu trúc `TestDefaultKillLanguageServer` để ghi đè `osTaskKill` bằng một hàm mock ghi nhận lệnh gọi (recording mock) thay vì chạy lệnh `taskkill` thật vào OS.
       - Inputs: `internal/reloader/reloader_test.go:183-193`.
       - Allowed Edit Scope: `internal/reloader/reloader_test.go`.
       - Outputs: Safe `TestDefaultKillLanguageServer` assert mock invocation cleanly.
       - Verification Condition: Chạy test không sinh tiến trình con `taskkill.exe`.
       - Checkpoint: `wip(P1-A): task 1.2 - mock osTaskKill in TestDefaultKillLanguageServer`.
  2. **Code Inspection & Lint:**
     - Review `git diff internal/reloader/` đảm bảo không có hardcoded shell command nào bị lọt.
  3. **Update Tests:**
     - Bổ sung test case kiểm tra hành vi context timeout và cancelled context trên mock killer.
  4. **Build:**
     - Chạy `go test -c ./internal/reloader/...` bảo đảm compile nhị phân test thành công.
  5. **QA (Mini QA):**
     - Chạy `go test -v -run TestDefaultKillLanguageServer ./internal/reloader/...` và kiểm tra PID của `language_server.exe` thực tế trên máy không hề thay đổi.
  6. **Slice Acceptance & Official Milestone Commit:**
     - Verify tiêu chí nghiệm thu AC1 (`B-M01` = 0).
     - Commit: `feat(P1-A): isolate test suite from real OS process killer`.

---

#### Slice 2 (P1-B): Proactive Token Expiry Verification & Google OAuth Refresh Engine

- **Goal:** Xây dựng cơ chế chủ động kiểm tra thời hạn token và refresh token qua Google OAuth endpoint `https://oauth2.googleapis.com/token` trước khi ghi vào Windows Credential Manager hoặc nạp vào Language Server RAM. Triệt tiêu hoàn toàn sự cố nạp token chết gây crash loop.
- **Scope Boundary:**
  - Editable:
    - `internal/connector/oauth_refresher.go` (file mới: module refresh OAuth token).
    - `internal/connector/oauth_refresher_test.go` (file mới: test cho module refresh).
    - `internal/model/types.go` (bổ sung hàm cập nhật TokenData và vô hiệu hóa RawPayload).
    - `internal/store/account_store.go` (cập nhật token mới vào storage đĩa).
    - `internal/manager/account_manager.go` (tích hợp pre-switch expiry check & refresh).
  - Out of scope: Thay đổi cấu trúc file lưu trữ trong `accounts_store/`.
- **Non-Goals:**
  - Không can thiệp vào token khi token vẫn còn hạn sử dụng (>= 5 phút).
  - Không thay đổi `client_id` hay thông số client ngoài cặp khóa Google CloudCode đã kiểm chứng.
- **Pre-flight Questions:**
  - Google Client ID: `1071006060591-tmhssin2h21lcre235vtolojh4g403ep.apps.googleusercontent.com`.
  - Google Client Secret: `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`.
  - Expiry threshold: Cần refresh khi `time.Until(expiry) < 5 * time.Minute` hoặc `expiry.IsZero()`.
- **Work Steps (6-Step Lifecycle):**
  1. **Code:**
     - *Leaf Task 2.1:* Trong `internal/model/types.go`, bổ sung method `UpdateTokens(accessToken, refreshToken, idToken string, expiry time.Time)` cho `TokenData`. Method này phải cập nhật các trường tương ứng VÀ gán `t.RawPayload = nil` (hoặc tái tạo RawPayload) để `MarshalJSON()` xuất ra token mới.
       - Inputs: `internal/model/types.go:121`.
       - Allowed Edit Scope: `internal/model/types.go`.
       - Outputs: Method `UpdateTokens` bảo đảm serialization fidelity.
       - Verification Condition: Sau khi gọi `UpdateTokens`, `json.Marshal(t)` chứa access_token mới và expiry mới.
       - Checkpoint: `wip(P1-B): task 2.1 - implement UpdateTokens with RawPayload invalidation in TokenData`.
     - *Leaf Task 2.2:* Tạo file `internal/connector/oauth_refresher.go` định nghĩa `OAuthRefresher` interface và `GoogleOAuthRefresher` struct:
       - Phương thức `RefreshToken(ctx context.Context, refreshToken string) (*TokenRefreshResult, error)`.
       - Gửi HTTP POST tới `https://oauth2.googleapis.com/token` với `client_id`, `client_secret`, `refresh_token`, `grant_type="refresh_token"`.
       - Parse kết quả: `access_token`, `expires_in`, `token_type`.
       - Hỗ trợ custom HTTP client / transport cho unit test.
       - Inputs: Cấu hình OAuth Google CloudCode đã đối soát tại `E0-P0A-API1`.
       - Allowed Edit Scope: `internal/connector/oauth_refresher.go`.
       - Outputs: Module `GoogleOAuthRefresher`.
       - Verification Condition: Gọi `RefreshToken` thành công trả về token mới.
       - Checkpoint: `wip(P1-B): task 2.2 - implement GoogleOAuthRefresher`.
     - *Leaf Task 2.3:* Trong `internal/manager/account_manager.go`, tích hợp bước kiểm tra hạn token trong `SwitchAccount`:
       - Trước bước 6 (ghi WinCred), kiểm tra `record.Token.Expiry`.
       - Nếu `time.Until(record.Token.Expiry) < 5*time.Minute`:
         * Gọi `oauthRefresher.RefreshToken(ctx, record.Token.RefreshToken)`.
         * Cập nhật `record.Token.UpdateTokens(...)`.
         * Lưu vào `AccountStore` đĩa (`m.store.SaveAccount(record)`).
       - Inputs: `internal/manager/account_manager.go:420-447`.
       - Allowed Edit Scope: `internal/manager/account_manager.go`.
       - Outputs: Tích hợp refresh an toàn tự động.
       - Verification Condition: Token hết hạn được tự động làm mới trước khi ghi WinCred.
       - Checkpoint: `wip(P1-B): task 2.3 - integrate proactive token refresh into SwitchAccount`.
  2. **Code Inspection & Lint:**
     - Review code đảm bảo error handling chặt chẽ: nếu refresh thất bại (ví dụ không có mạng), ghi nhận cảnh báo rõ ràng chứ không để token hỏng làm sập app.
  3. **Update Tests:**
     - Tạo `internal/connector/oauth_refresher_test.go` kiểm thử:
       * Thành công khi server trả về HTTP 200 JSON với access_token và expires_in.
       * Xử lý lỗi khi HTTP 400/401 hoặc network timeout.
     - Cập nhật `internal/manager/account_manager_test.go` với test case tài khoản có token hết hạn được tự động refresh qua mock refresher.
  4. **Build:**
     - Chạy `go build ./internal/...` xác nhận không có compile error.
  5. **QA (Mini QA):**
     - Chạy test xác nhận tài khoản hết hạn được cấp token mới hợp lệ.
  6. **Slice Acceptance & Official Milestone Commit:**
     - Verify tiêu chí nghiệm thu AC2 (`B-M02` = 0%).
     - Commit: `feat(P1-B): proactive google oauth token refresh before wincred write`.

---

#### Slice 3 (P1-C): Atomic app_storage.json & Session Synchronization

- **Goal:** Đồng bộ hóa nguyên tử `%APPDATA%\Antigravity\app_storage.json` (`jetski.onboarding.lastLoginUsername` và `jetski.onboarding.lastLoginIsGcpTos`) trước khi reload/reauth, triệt tiêu hoàn toàn modal Google Policy / ToS popup.
- **Scope Boundary:**
  - Editable:
    - `internal/reloader/session_reloader.go`.
    - `internal/reloader/reloader_test.go`.
    - `internal/manager/account_manager.go`.
  - Out of scope: Can thiệp vào file cấu hình khác ngoài `app_storage.json`.
- **Non-Goals:**
  - Không ghi đè các settings khác của người dùng trong `app_storage.json` (giữ nguyên layout, conversations, zoomLevel, v.v.).
- **Pre-flight Questions:**
  - Storage path: `%APPDATA%\Antigravity\app_storage.json`.
  - Keys to synchronize:
    - `"jetski.onboarding.lastLoginUsername"`: `<target_email>`.
    - `"jetski.onboarding.lastLoginIsGcpTos"`: `"false"` (hoặc `"true"` nếu auth_method là gcp).
  - Sync order: Phải hoàn tất ghi đĩa VÀ fsync TRƯỚC KHI Language Server nhận token mới.
- **Work Steps (6-Step Lifecycle):**
  1. **Code:**
     - *Leaf Task 3.1:* Trong `internal/reloader/session_reloader.go`, nâng cấp `PrePatchToSAcceptance` thành `SyncAppStorageToS(targetEmail string, isGcpTos bool, storagePath string) error`:
       - Đọc file JSON hiện có (nếu chưa có thì tạo map rỗng).
       - Bảo toàn 100% các trường cấu hình khác của người dùng.
       - Ghi đè chính xác:
         * `data["jetski.onboarding.lastLoginUsername"] = normEmail`
         * `data["jetski.onboarding.lastLoginIsGcpTos"] = strconv.FormatBool(isGcpTos)`
       - Ghi ra temp file trong cùng thư mục với `app_storage.json`, gọi `tmpFile.Sync()` để ép flush dữ liệu xuống đĩa vật lý, đóng file và thực hiện `os.Rename` nguyên tử.
       - Inputs: `internal/reloader/session_reloader.go:271-330`.
       - Allowed Edit Scope: `internal/reloader/session_reloader.go`.
       - Outputs: Hàm `SyncAppStorageToS` nguyên tử và tin cậy.
       - Verification Condition: Dữ liệu được ghi an toàn trên đĩa, không bị mất khi reload.
       - Checkpoint: `wip(P1-C): task 3.1 - implement atomic SyncAppStorageToS with fsync`.
     - *Leaf Task 3.2:* Trong `internal/manager/account_manager.go`:
       - Điều chỉnh thứ tự gọi: Đồng bộ hóa `SyncAppStorageToS` TRƯỚC KHI thực hiện ghi Windows Credential Manager và TRƯỚC KHI ReAuth/Reload.
       - Xác định `isGcpTos`: Nếu `record.Token.AuthMethod == "gcp"` -> `true`, ngược lại -> `false`.
       - Inputs: `internal/manager/account_manager.go:448-456`.
       - Allowed Edit Scope: `internal/manager/account_manager.go`.
       - Outputs: Đồng bộ thứ tự thực thi chuẩn xác.
       - Verification Condition: `app_storage.json` luôn chứa email mới trước khi Language Server nạp token.
       - Checkpoint: `wip(P1-C): task 3.2 - wire SyncAppStorageToS into SwitchAccount before reauth`.
  2. **Code Inspection & Lint:**
     - Kiểm tra khả năng tương thích Windows path (`%APPDATA%`), kiểm tra atomic rename khi file đang bị process khác mở read-only.
  3. **Update Tests:**
     - Cập nhật `internal/reloader/reloader_test.go` bổ sung test case:
       * Kiểm tra bảo toàn các trường dữ liệu khác trong `app_storage.json`.
       * Kiểm tra trường hợp `isGcpTos = true` và `isGcpTos = false`.
       * Kiểm tra phục hồi khi file JSON ban đầu bị hỏng hoặc rỗng.
  4. **Build:**
     - Chạy `go test ./internal/reloader/...` xác nhận compile và test passed.
  5. **QA (Mini QA):**
     - Thực hiện kiểm tra nội dung file `%APPDATA%\Antigravity\app_storage.json` trên đĩa sau khi gọi hàm, xác nhận email và flag ToS được ghi chuẩn xác.
  6. **Slice Acceptance & Official Milestone Commit:**
     - Verify tiêu chí nghiệm thu AC3 (`B-M04` = 0).
     - Commit: `feat(P1-C): atomic app_storage.json sync to eliminate google tos modal`.

---

#### Slice 4 (P1-D): Civilized RPC ReAuth Hot-Swap with Safe Fallback

- **Goal:** Nâng cấp cơ chế chuyển đổi tài khoản văn minh qua Connect-RPC endpoint `/exa.language_server_pb.LanguageServerService/ReAuth` thay cho hành vi tiêu diệt tiến trình thô bạo (`taskkill /F`). Giữ cho Antigravity IDE hoàn toàn ổn định, không giật màn hình và không reload cửa sổ.
- **Scope Boundary:**
  - Editable:
    - `internal/manager/account_manager.go`.
    - `internal/manager/account_manager_test.go`.
    - `internal/connector/api_client.go`.
  - Out of scope: Sửa đổi các RPC endpoints khác ngoài ReAuth và GetUserStatus.
- **Non-Goals:**
  - Không bỏ rơi cơ chế fallback: Nếu Language Server chưa chạy hoặc không phản hồi RPC, hệ thống vẫn có fallback khởi động lại an toàn.
- **Pre-flight Questions:**
  - Endpoint: `/exa.language_server_pb.LanguageServerService/ReAuth` (đã có trong `connector.DefaultReauthEndpoints`).
  - Verification endpoint: `/exa.language_server_pb.LanguageServerService/GetUserStatus` để xác nhận email mới đã nạp vào RAM.
- **Work Steps (6-Step Lifecycle):**
  1. **Code:**
     - *Leaf Task 4.1:* Trong `internal/manager/account_manager.go`, tái cấu trúc bước 9 của `SwitchAccount`:
       - Kiểm tra nếu `client != nil` (Language Server đang hoạt động):
         * Gửi lệnh `client.TriggerReauth(ctx)`.
         * Nếu `TriggerReauth` thành công (HTTP 200/204):
           - Chờ ngắn (100-200ms) để Language Server nạp token từ WinCred vào RAM.
           - Gọi `client.GetUserStatus(ctx)` xác nhận `status.Email == normTarget`.
           - Nếu xác nhận trùng khớp: **BỎ QUA HOÀN TOÀN `RecycleLanguageServer`**! Quá trình chuyển đổi hoàn tất "seamlessly" mà không có bất kỳ lệnh taskkill hay window reload nào!
         * Nếu `TriggerReauth` thất bại hoặc email không đổi sau retry:
           - Ghi log cảnh báo và fallback sang `rel.RecycleLanguageServer(ctx, 2*time.Second)`.
       - Nếu `client == nil` (Language Server chưa chạy):
         * Khởi động Language Server bình thường.
       - Inputs: `internal/manager/account_manager.go:458-475`.
       - Allowed Edit Scope: `internal/manager/account_manager.go`.
       - Outputs: Luồng ReAuth văn minh không gây crash IDE.
       - Verification Condition: SwitchAccount hoàn tất mà không gọi `killer`.
       - Checkpoint: `wip(P1-D): task 4.1 - implement RPC ReAuth primary path with recycle fallback`.
     - *Leaf Task 4.2:* Cập nhật `SwitchResult` trong `internal/manager/account_manager.go`:
       - Bổ sung trường `HotSwapped bool json:"hot_swapped"` để caller biết tài khoản được nạp trực tiếp vào RAM qua RPC hay phải restart.
       - Inputs: `internal/manager/account_manager.go:50-58`.
       - Allowed Edit Scope: `internal/manager/account_manager.go`.
       - Outputs: Thông tin chi tiết trong `SwitchResult`.
       - Verification Condition: `result.HotSwapped == true` khi ReAuth thành công.
       - Checkpoint: `wip(P1-D): task 4.2 - expose HotSwapped indicator in SwitchResult`.
  2. **Code Inspection & Lint:**
     - Kiểm tra timeout của RPC: giới hạn 2 giây cho ReAuth để không làm treo giao diện nếu Language Server bị nghẽn mạng.
  3. **Update Tests:**
     - Trong `internal/manager/account_manager_test.go`:
       * Test case 1: Language Server đang chạy, `TriggerReauth` thành công -> `SwitchAccount` hoàn tất thành công với `HotSwapped = true`, verify killer KHÔNG bị gọi.
       * Test case 2: `TriggerReauth` trả lỗi -> `SwitchAccount` fallback sang `RecycleLanguageServer`, verify killer ĐƯỢC gọi an toàn.
  4. **Build:**
     - Chạy toàn bộ test suite `go test -v ./...` để bảo đảm toàn bộ hệ thống tích hợp mượt mà.
  5. **QA (Mini QA):**
     - Kiểm tra benchmark metric `B-M03` (Window reloads = 0) và `B-M05` (Switch latency < 500ms).
  6. **Slice Acceptance & Official Milestone Commit:**
     - Verify tiêu chí nghiệm thu AC4 (`B-M03` = 0%, `B-M05` < 500ms).
     - Commit: `feat(P1-D): civilized rpc reauth hot-swap with graceful fallback`.

---

## Risk Assessment & Mitigation Matrix

| Rủi ro kỹ thuật | Mức độ | Biện pháp giảm thiểu |
| :--- | :---: | :--- |
| Google OAuth refresh endpoint từ chối token (token revoked hoặc mất mạng) | Trung bình | Ghi nhận lỗi chi tiết; giữ nguyên token hiện có thay vì ghi đè giá trị rỗng; thông báo rõ ràng cho UI biết tài khoản cần đăng nhập lại từ trình duyệt. |
| Language Server phiên bản cũ không hỗ trợ endpoint `/ReAuth` | Thấp | Danh sách `DefaultReauthEndpoints` đã có sẵn các candidate `/Reload`, `/reauth`, `/reload`. Nếu toàn bộ endpoint trả về 404/405, hệ thống tự động fallback về `RecycleLanguageServer`. |
| Tranh chấp quyền truy cập file `app_storage.json` khi Electron đang mở | Thấp | Áp dụng cơ chế ghi temp file và atomic rename (`os.Rename`), kèm retry loop 3 lần với backoff 50ms nếu gặp `ERROR_SHARING_VIOLATION`. |
| Test suite bị flaky do phụ thuộc vào thời gian thực | Rất thấp | Tất cả các slice đều cung cấp mock interface hoàn chỉnh (Mock Killer, Mock OAuth Refresher, Mock Client). |

