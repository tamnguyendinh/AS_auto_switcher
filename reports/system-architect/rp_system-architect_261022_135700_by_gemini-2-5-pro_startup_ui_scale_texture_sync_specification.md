# Báo Cáo Kiến Trúc Hệ Thống: Đặc Tả Đồng Bộ Tỷ Lệ Hiển Thị & Thanh Lọc Texture Cache Khi Khởi Động (Startup UI Scale & Texture Synchronization)

- **Mã báo cáo:** `rp_system-architect_261022_135700_by_gemini-2-5-pro_startup_ui_scale_texture_sync_specification`
- **Ngày lập:** 2026-10-22 13:57:00 +07:00
- **Lĩnh vực:** Kiến trúc Giao diện Đồ họa (GUI Architecture & OpenGL/GLFW Lifecycle Synchronization)
- **Tác giả:** System Architect (pair-consulting cùng Human Owner)
- **Văn kiện đặc tả:** `reports/architect/2026-10-22-startup-ui-scale-texture-sync-spec.md`

---

## 1. Phạm Vi Công Việc (Scope)
- Khảo sát sâu toàn bộ codebase, cấu trúc runtime Fyne v2.8.2, driver OpenGL/GLFW, và chu trình quản lý Theme / Texture Cache trên môi trường Windows High-DPI (120 DPI / 125% scale, Fyne scale = 1.3).
- Giải mã và xác định nguyên nhân kỹ thuật gốc rễ của hiện tượng: UI ban đầu to và hơi nhòe nét, nhưng khi đổi theme tại runtime thì co lại nhỏ gọn, sắc nét và tinh tế.
- Thiết kế giải pháp kiến trúc **"Post-Show Visual Synchronization"** nhằm đồng bộ tỷ lệ hiển thị và thanh lọc texture cache tự động ngay khi mở app.
- Soạn thảo tài liệu đặc tả kiến trúc (SPEC) hoàn chỉnh, chi tiết từng bước cho Coder tại `reports/architect/2026-10-22-startup-ui-scale-texture-sync-spec.md`.

---

## 2. Danh Mục Tệp Tin Đã Tạo (Output Files Created)
- `reports/architect/2026-10-22-startup-ui-scale-texture-sync-spec.md` (Đặc tả kiến trúc & hướng dẫn triển khai chi tiết cho Coder).
- `reports/system-architect/rp_system-architect_261022_135700_by_gemini-2-5-pro_startup_ui_scale_texture_sync_specification.md` (Báo cáo tổng kết lane Kiến trúc sư).

---

## 3. Các Quyết Định Kiến Trúc Đã Thống Nhất (Decisions Made)
1. **Thiết lập chu trình Post-Show Visual Sync:**
   - Thay vì can thiệp vào mã nguồn nội bộ của Fyne hoặc thay đổi cách khởi tạo GLFW, hệ thống bổ sung phương thức an toàn `SyncVisualScaleAndTextures()` trong `internal/ui/app.go`.
   - Phương thức này được kích hoạt tự động qua goroutine nền ngay sau khi `a.win.Show()` xuất hiện và GLFW event loop bắt đầu, dispatch vào main thread qua `fyne.Do()`.
2. **Không ghi đè hoặc thay đổi tệp Preferences của người dùng:**
   - `SyncVisualScaleAndTextures()` chỉ làm sạch cache texture (`DeleteTextTexturesFor`) và rescale context, tuyệt đối không gọi `SaveUIPreferences()` gây ghi đĩa thừa.
3. **Bảo toàn Invariants & Thread-Safety:**
   - Giữ nguyên cấu trúc lưới thẻ tài khoản `280x130` (`newSymmetricGridWrap`).
   - Xử lý trọn vẹn cả trường hợp `overview` có trước hoặc sau chu trình đồng bộ.
   - Giữ nguyên 100% kịch bản kiểm thử và tính tương thích ngược.

---

## 4. Các Câu Hỏi Còn Tồn Đọng (Residual Open Questions)
- Không còn câu hỏi tồn đọng nào. Giải pháp đã được kiểm nghiệm thực tế qua các script kiểm tra độc lập và đồng thuận cùng Human Owner.

---

## 5. Tham Chiếu Git Commit (Commit Reference)
- Đang chờ commit các văn kiện báo cáo thuộc quyền sở hữu của lane Kiến trúc sư (`reports/architect/*`, `reports/system-architect/*`).
- Chỉ gửi báo cáo bàn giao lên CEO khi Human Owner ra lệnh bằng đúng nguyên văn `"gửi CEO"` (hoặc `"gui ceo"`).
