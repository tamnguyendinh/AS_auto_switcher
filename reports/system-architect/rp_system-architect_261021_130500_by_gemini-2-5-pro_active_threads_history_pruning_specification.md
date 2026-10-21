# Báo Cáo Kiến Trúc: Đặc Tả Dọn Dẹp Lịch Sử Active Threads
(System Architect Lane Completion Report — Active Threads History Pruning)

- **Thời gian lập:** 2026-10-21 13:05:00 (Asia/Bangkok)
- **Lane:** System Architect
- **Model:** gemini-2-5-pro
- **Thẩm quyền:** Chỉ đạo trực tiếp và phê duyệt từ Human Owner
- **Mã văn kiện đặc tả liên kết:** `SPEC-ARCH-20261021-ACTIVE-THREADS-HISTORY-PRUNING`
- **Mã báo cáo:** `rp_system-architect_261021_130500_by_gemini-2-5-pro_active_threads_history_pruning_specification`

---

## I. PHẠM VI CÔNG VIỆC HOÀN THÀNH (SCOPE COMPLETED)

1. **Khảo sát hiện trạng codebase:**
   - Điều tra thư mục `accounts_store\active_threads_history\`, ghi nhận thực trạng tích tụ **949 file JSON** rác không bao giờ được dọn dẹp sau các lần chuyển tài khoản.
   - Rà soát luồng `SwitchAccount`, `CaptureActiveThreadsSnapshot`, `ReadActiveThreadsSnapshot`, và `PerformSessionContinuation`.
2. **Thảo luận và thống nhất giải pháp cùng Human Owner:**
   - Đề xuất các phương án kích hoạt và chính sách lưu trữ.
   - Thống nhất theo quyết định dứt khoát của Owner: Loại bỏ các chốt rườm rà, chỉ áp dụng **1 chốt duy nhất sau khi hoàn tất chuyển tài khoản (Post-Switch Completion)**, giữ tối đa **đúng 2 file mới nhất**, xóa toàn bộ các file còn lại.
3. **Biên soạn văn kiện đặc tả kiến trúc chính thức:**
   - Hoàn thành tài liệu [2026-10-21-active-threads-history-pruning-spec.md](file:///g:/antigravity_auto_switch/reports/architect/2026-10-21-active-threads-history-pruning-spec.md) với đầy đủ điều khoản kỷ luật sắt, thuật toán cắt tỉa, chỉ dẫn chi tiết từng bước cho Coder (step-by-step recipe) và ma trận kiểm thử tự động.
4. **Phê chuẩn từ Owner:**
   - Owner đã trực tiếp kiểm tra và ra lệnh chuẩn xác: `"gửi CEO"`.

---

## II. DANH MỤC TỆP ĐẦU RA (OUTPUT ARTIFACTS)

1. `reports/architect/2026-10-21-active-threads-history-pruning-spec.md` (Đặc tả kiến trúc chi tiết, commit `aa72f36`)
2. `reports/system-architect/rp_system-architect_261021_130500_by_gemini-2-5-pro_active_threads_history_pruning_specification.md` (Báo cáo hoàn thành lane)

---

## III. QUYẾT ĐỊNH KIẾN TRÚC ĐÃ CHỐT (ARCHITECTURAL DECISIONS)

1. **Giới hạn lưu trữ:** Giữ tối đa 2 file snapshot mới nhất tại `accounts_store\active_threads_history\`.
2. **Điểm kích hoạt:** Đặt ở cuối hàm `SwitchAccount` sau khi Bước 4 (tiếp tục phiên - Continuation) đã chạy xong xuôi và file snapshot đã được đọc xong.
3. **Tính bất biến & An toàn:** Xử lý lỗi mềm dẻo (fail-safe), không để lỗi xóa file làm fail thao tác chuyển tài khoản.
4. **Giải phóng bãi rác:** Ngay ở lần switch kế tiếp, toàn bộ 949 file cũ sẽ được tự động xóa sạch, thư mục duy trì $\le 2$ file vĩnh viễn.

---

## IV. VẤN ĐỀ CÒN TỒN ĐỌNG (RESIDUAL OPEN QUESTIONS)

- **Không có (None).** Văn kiện đặc tả đã được Owner duyệt toàn bộ.

---

## V. ĐỀ XUẤT BÀN GIAO (HANDOFF RECOMMENDATION)

- **Verdict:** `READY_FOR_CODER`
- **Handoff Target:** Coder Lane nhận bản đặc tả `reports/architect/2026-10-21-active-threads-history-pruning-spec.md` để triển khai mã nguồn và bổ sung bộ test tự động tại `internal/reloader/` và `internal/manager/`.
