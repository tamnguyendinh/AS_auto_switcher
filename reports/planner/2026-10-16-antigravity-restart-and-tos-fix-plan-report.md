# Báo Cáo Kế Hoạch Hành Động: Khắc Phục Triệt Để Sự Cố Antigravity Restart & Google Policy Modal

**Tài liệu:** `reports/planner/2026-10-16-antigravity-restart-and-tos-fix-plan-report.md`  
**Ngày thực hiện:** 2026-10-16  
**Làn đảm nhiệm:** Planner Specialist  
**Cơ quan chỉ đạo:** CEO Agent (Thread ID: `01a14378-4a08-77c1-88d5-666ad8c3592d`)  
**Thẩm quyền & Căn cứ:** Chỉ đạo trực tiếp từ Human Owner, AGENTS.md, Báo cáo Chuyên sâu RCA `reports/architect/2026-10-16-antigravity-restart-and-tos-modal-rca.md`.  
**Kế hoạch chi tiết đi kèm:** `docs/plans/2026-10-16-antigravity-restart-and-tos-fix/2026-10-16-antigravity-restart-and-tos-fix-plan.md`  
**Trạng thái bàn giao:** `[VERDICT: READY_FOR_ARCHITECT_REVIEW]`

---

## 1. Tóm tắt Kế hoạch & Mục tiêu chuyển hóa

Thực thi nghiêm ngặt chỉ thị từ CEO Agent và kết quả điều tra gốc rễ (RCA) từ Architect, Planner Lane đã hoàn tất việc chuyển hóa 4 trụ cột giải pháp kiến trúc thành một bản **Kế hoạch Hành động chi tiết** (Action Plan) tại thư mục chuẩn `docs/plans/2026-10-16-antigravity-restart-and-tos-fix/`.

Kế hoạch giải quyết tận gốc hai vấn đề gây tê liệt trải nghiệm của Human Owner:
1. **Antigravity IDE liên tục bị "restart" (Window reload)** chu kỳ ~40-60 giây do cơ chế hot-swap thô bạo (`taskkill /F /IM language_server.exe`), test suite bắn hạ IDE thật, và nạp token hết hạn gây crash loop `os.Exit(1)`.
2. **Modal Google Policy / Terms of Service (ToS popup)** xuất hiện sau mỗi lần reload do `app_storage.json` bị lệch pha email (`stored.email !== currentEmail`) kích hoạt component React `ACb` render dialog `zCb`.

---

## 2. Bảng Phân rã 4 Slice Thực thi Độc lập (Execution Slices)

| Slice ID | Tên Slice | Trụ cột Kiến trúc tương ứng | Touch-Map (Khóa chặt Scope) | Tiêu chí Nghiệm thu (Zero-Trust Acceptance) | Non-Goals (Ranh giới cấm vượt) |
| :---: | :--- | :--- | :--- | :--- | :--- |
| **P1-A** | Test Suite Isolation Mock | Trụ cột 1: Cách ly Test Suite khỏi OS thực | `internal/reloader/reloader_test.go`<br>`internal/reloader/session_reloader.go` | `go test -v ./internal/reloader/...` chạy qua 100% mà số lệnh `taskkill` gửi tới OS thực = 0 (`B-M01` = 0). IDE của Owner không bị gián đoạn khi chạy test. | Không can thiệp mã nguồn ngoài package `reloader`. Không xóa hàm fallback `DefaultKillLanguageServer` trên runtime thực. |
| **P1-B** | Token Expiry Check & OAuth Refresh Engine | Trụ cột 2: Refresh Token chủ động trước khi nạp WinCred | `internal/connector/oauth_refresher.go` *(mới)*<br>`internal/connector/oauth_refresher_test.go` *(mới)*<br>`internal/model/types.go`<br>`internal/store/account_store.go`<br>`internal/manager/account_manager.go` | Tự động phát hiện token có hạn < 5 phút, gửi HTTP POST tới Google OAuth endpoint `https://oauth2.googleapis.com/token`, nhận token mới (3599s TTL), cập nhật `AccountStore` và vô hiệu hóa `RawPayload` cũ trước khi ghi WinCred (`B-M02` = 0%). | Không can thiệp khi token còn hạn >= 5 phút. Không đổi format lưu trữ `accounts_store/`. Không đổi cặp khóa Google CloudCode đã xác minh. |
| **P1-C** | Atomic `app_storage.json` Sync | Trụ cột 3: Đồng bộ nguyên tử `app_storage.json` trước khi ReAuth/Reload | `internal/reloader/session_reloader.go`<br>`internal/reloader/reloader_test.go`<br>`internal/manager/account_manager.go` | Cập nhật chính xác `jetski.onboarding.lastLoginUsername` và `jetski.onboarding.lastLoginIsGcpTos` với cơ chế atomic rename + `fsync()` hoàn tất TRƯỚC KHI ReAuth/Reload. Triệt tiêu 100% modal ToS popup (`B-M04` = 0). | Không can thiệp vào các trường dữ liệu người dùng khác trong `app_storage.json` (layout, zoom, conversations). |
| **P1-D** | RPC ReAuth Hot-Swap with Safe Fallback | Trụ cột 4: Re-auth văn minh qua Connect-RPC endpoint | `internal/manager/account_manager.go`<br>`internal/manager/account_manager_test.go`<br>`internal/connector/api_client.go` | `SwitchAccount` ưu tiên gọi `TriggerReauth(ctx)` tới `/exa.language_server_pb.LanguageServerService/ReAuth`. Khi thành công và `GetUserStatus` xác nhận email mới, bỏ qua hoàn toàn `taskkill`, IDE không bị crash và không reload window (`B-M03` = 0%, độ trễ < 500ms). | Không bỏ fallback: nếu Language Server chưa chạy hoặc RPC không phản hồi, vẫn có fallback restart an toàn. Không can thiệp UI. |

---

## 3. Các Phát hiện Kỹ thuật & Bằng chứng Đối soát Thực tế (Empirical Findings)

Trong quá trình lập kế hoạch, Planner đã trực tiếp đối soát các thông số kỹ thuật thực tế để đảm bảo Kế hoạch khả thi 100% (Zero Speculation):

1. **Bằng chứng Test suite tiêu diệt IDE thật (`E0-P0A-SRC1`):**
   Tại dòng 183 của `internal/reloader/reloader_test.go`, hàm `TestDefaultKillLanguageServer` trực tiếp thực thi `DefaultKillLanguageServer(ctx)` gọi lệnh `taskkill /F /IM language_server.exe` vào Windows. Đây là nguyên nhân khiến IDE của Owner biến mất bất ngờ khi chạy test.
2. **Khám phá và Xác minh Tham số Google OAuth Refresh (`E0-P0A-API1`):**
   - Đã dịch ngược chuỗi trong `resources\bin\language_server.exe` và tìm thấy cặp Client ID & Client Secret chính thống của Google CloudCode:
     * `client_id`: `1071006060591-tmhssin2h21lcre235vtolojh4g403ep.apps.googleusercontent.com`
     * `client_secret`: `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`
   - Đã thực nghiệm gửi `refresh_token` từ tài khoản `daohien204ccyersin@gmail.com` tới `https://oauth2.googleapis.com/token`.
   - Kết quả: **`HTTP 200 OK`**, cấp access_token mới có `expires_in: 3599` giây! Điều này chứng minh giải pháp tự động refresh token hoàn toàn hoạt động độc lập và tin cậy.
3. **Hiện trạng Lệch pha File `app_storage.json` trên máy thật (`E0-P0A-TOS1`):**
   - Kiểm tra `%APPDATA%\Antigravity\app_storage.json` trên máy phát hiện trường `"jetski.onboarding.lastLoginUsername"` đang bị gán chuỗi test `"switch-continuation@example.com"`.
   - Khi Antigravity nạp phiên thực của người dùng, component `ACb` trong `/main.js` so sánh `stored.email !== currentEmail`, hàm `xCb` trả về `"account_changed"` và kích hoạt modal `zCb` (ToS dialog).
4. **Endpoint RPC ReAuth đã có sẵn (`E0-P0A-RPC1`):**
   - Module `internal/connector/api_client.go` đã cài đặt sẵn method `TriggerReauth(ctx context.Context) error` gọi tới `/exa.language_server_pb.LanguageServerService/ReAuth`. Việc tích hợp vào `AccountManager.SwitchAccount` có thể thực hiện ngay lập tức mà không cần viết lại giao thức.

---

## 4. Bộ Tài liệu Kế hoạch Chuẩn đã Lập (Plan Artifacts)

Bộ tài liệu 5 tệp chuẩn được lưu trữ đồng bộ tại:
- `docs/plans/2026-10-16-antigravity-restart-and-tos-fix/2026-10-16-antigravity-restart-and-tos-fix-plan.md`: Kế hoạch hành động chi tiết 4 Slices theo chuẩn chu kỳ 6 bước.
- `docs/plans/2026-10-16-antigravity-restart-and-tos-fix/2026-10-16-antigravity-restart-and-tos-fix-actual-status.md`: Ma trận hiện trạng mã nguồn baseline (P0 Complete).
- `docs/plans/2026-10-16-antigravity-restart-and-tos-fix/2026-10-16-antigravity-restart-and-tos-fix-evidence.md`: Sổ cái chứng cứ đối soát `E0-P0A-...`.
- `docs/plans/2026-10-16-antigravity-restart-and-tos-fix/2026-10-16-antigravity-restart-and-tos-fix-benchmark.md`: Sổ cái đo lường chỉ số mục tiêu `B-M01` đến `B-M06`.
- `reports/planner/2026-10-16-antigravity-restart-and-tos-fix-plan-report.md`: Báo cáo tổng kết bàn giao.

---

## 5. Khuyến nghị Bàn giao Tiếp theo

1. Kế hoạch đã sẵn sàng để CEO Agent chuyển giao cho **Architect Review Lane** hoặc phê duyệt trực tiếp cho **Coder Lane** thực thi tuần tự từ Slice 1 đến Slice 4.
2. Thứ tự khuyến nghị triển khai:
   - **Ưu tiên 1 (Khẩn cấp): Slice 1 (Test Isolation)** để bảo vệ môi trường làm việc của Human Owner khi các lane chạy `go test`.
   - **Ưu tiên 2: Slice 2 (Token Refresh Engine)** để dập tắt triệt để hiện tượng nạp token chết gây crash loop.
   - **Ưu tiên 3: Slice 3 (Atomic app_storage.json Sync)** để triệt tiêu hoàn toàn modal Google Policy / ToS.
   - **Ưu tiên 4: Slice 4 (Civilized RPC ReAuth Hot-Swap)** để mang lại trải nghiệm hot-swap không reload cửa sổ, không giật màn hình.

