# Báo Cáo Kiến Trúc: Nút Xóa Tài Khoản (Trash Button) & Đồng Bộ Hai Kho Dữ Liệu
(System Architect Lane Completion Report — Account Deletion & Two-Store Synchronization Specification)

- **Thời gian lập:** 2026-10-23 09:05:00 (Asia/Bangkok)
- **Lane:** System Architect (Nhánh Chiến Lược)
- **Model:** gemini-2-5-pro
- **Thẩm quyền:** Trực tiếp thảo luận sâu, giải thích phản biện và được Human Owner chính thức chấp thuận toàn bộ phương án
- **Mã văn kiện đặc tả liên kết:** `SPEC-ARCH-20261023-ACCOUNT-DELETION-AND-TRASH-BUTTON`
- **Tệp đặc tả kỹ thuật chi tiết:** [`reports/architect/2026-10-23-account-deletion-and-trash-button-spec.md`](file:///g:/antigravity_auto_switch/reports/architect/2026-10-23-account-deletion-and-trash-button-spec.md)
- **Mã báo cáo:** `rp_system-architect_261023_090500_by_gemini-2-5-pro_account_deletion_and_trash_button_spec`

---

## I. PHẠM VI CÔNG VIỆC HOÀN THÀNH (SCOPE COMPLETED)

1. **Khảo sát & Phân tích Hiện trạng Codebase:**
   - Khảo sát mô hình Two-Store giữa `accounts_store/` (kho token) và `discovered_accounts/` (danh bạ Chrome).
   - Khảo sát widget thẻ tài khoản `AccountCardWidget` tại [`internal/ui/widgets.go`](file:///g:/antigravity_auto_switch/internal/ui/widgets.go) và cấu trúc `actionCluster`.
   - Khảo sát cơ chế tự động capture session của Windows Credential Manager (`wincred`) trong `GetAccountStates()`.
   - Phát hiện rủi ro nghiêm trọng: Nếu cho phép xóa tài khoản đang Active, hệ thống sẽ tự động bắt lại token từ `wincred` và ghi đè file trở lại vào `accounts_store/`.

2. **Thảo luận Sâu, Phản Biện & Thống Nhất cùng Human Owner:**
   - **Kịch bản xóa:** Thống nhất áp dụng **Kịch bản B** (Gỡ hoàn toàn thẻ khỏi màn hình chính): Xóa file trong `accounts_store/` VÀ đặt `added: false` trong `discovered_accounts/` để đưa tài khoản về lại danh sách ứng viên trong Modal `[+ Thêm]`.
   - **Bảo vệ tài khoản Active (Invariant 1):** Tuyệt đối không hiển thị nút Xóa trên thẻ đang Active (`[✔ Dùng]`). Cấm triệt để việc xóa tài khoản đang hoạt động.
   - **Thứ tự nút Action (Invariant 2):** Thống nhất thứ tự chuẩn:
     $$\text{[ 1. Action Slot ]} \quad \longrightarrow \quad \text{[ 2. Refresh Quota ]} \quad \longrightarrow \quad \text{[ 3. Status ON/OFF ]} \quad \longrightarrow \quad \text{[ 4. Trash Button 🗑 ]}$$
   - **Cơ chế an toàn (Invariant 3):** Bắt buộc hiển thị Pop-up xác nhận (`dialog.NewCustomConfirm`) với 2 nút: `[Hủy]` và `[Xóa]` (Danger đỏ). Phân cấp thông báo thông minh giữa thẻ có token và thẻ `[Cần OAuth]`.
   - **Hỗ trợ thẻ OFF (Invariant 5):** Thẻ mang `Status == "off"` vẫn cho phép click nút Xóa để dọn dẹp.
   - **Hỗ trợ thẻ `[Cần OAuth]`:** Cho phép click nút Xóa để gỡ thẻ chờ OAuth xuống khỏi màn hình chính.
   - **Dọn dẹp bộ nhớ & Auto-switch safety (Invariant 6):** Xóa quota cache trong RAM; nếu hết tài khoản hợp lệ, tự động ngắt auto-switch an toàn và báo `⚠️ Không có tài khoản để chuyển.`.

3. **Biên soạn Văn kiện Đặc tả Chi Tiết cho Coder:**
   - Đã biên soạn đầy đủ tài liệu tại [`reports/architect/2026-10-23-account-deletion-and-trash-button-spec.md`](file:///g:/antigravity_auto_switch/reports/architect/2026-10-23-account-deletion-and-trash-button-spec.md).
   - Tài liệu chứa đầy đủ Touch-Map 5 bước triển khai, code mẫu chi tiết từng hàm, struct và 8 kịch bản kiểm thử chấp nhận (TC-01 đến TC-08).

---

## II. DANH MỤC TỆP ĐẦU RA (OUTPUT ARTIFACTS)

1. [`reports/architect/2026-10-23-account-deletion-and-trash-button-spec.md`](file:///g:/antigravity_auto_switch/reports/architect/2026-10-23-account-deletion-and-trash-button-spec.md) (Văn kiện đặc tả kỹ thuật chi tiết)
2. [`reports/system-architect/rp_system-architect_261023_090500_by_gemini-2-5-pro_account_deletion_and_trash_button_spec.md`](file:///g:/antigravity_auto_switch/reports/system-architect/rp_system-architect_261023_090500_by_gemini-2-5-pro_account_deletion_and_trash_button_spec.md) (Báo cáo tổng kết của System Architect Lane)

---

## III. TOUCH-MAP & CÁC TỆP BỊ ẢNH HƯỞNG

1. `internal/telemetry/cache.go`: Thêm `Delete(email string)` vào `QuotaCache`.
2. `internal/manager/account_manager.go`: Thêm `ErrCannotDeleteActiveAccount` và method `RemoveAccount(ctx, email)`.
3. `internal/ui/widgets.go`: Thêm trường `onDelete`, `deleteButton`, bố trí vào vị trí thứ 4 của `actionCluster`, cập nhật `UpdateState`, bổ sung `DeleteButton()` và `TriggerDelete()`.
4. `internal/ui/app.go`: Thêm `ConfirmAndDeleteAccount(email)` (Confirmation Dialog với Danger button), `executeRemoveAccount(email, display)`, gắn callback vào các vị trí sinh card.
5. Unit & Acceptance Tests: Thêm các ca kiểm thử kiểm tra TC-01 đến TC-08.

---

## IV. TRẠNG THÁI & CHUYỂN GIAO TIẾP THEO

- **Trạng thái:** SPEC ĐÃ KHÓA & HOÀN TẤT (LOCKED & READY FOR CODER).
- **Handoff:** Tuân thủ quy định: Không tự ý gửi báo cáo cho CEO khi chưa có lệnh trực tiếp từ Human Owner. Sẵn sàng gửi tin nhắn chuyển giao khi Owner ra lệnh.
