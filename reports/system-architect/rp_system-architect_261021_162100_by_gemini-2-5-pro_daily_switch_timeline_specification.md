# Báo Cáo Kiến Trúc: Phân Mảnh Nhật Ký Chuyển Tài Khoản Theo Ngày & Dọn Dẹp 7 Ngày
(System Architect Lane Completion Report — Daily Switch Timeline Sharding & 7-Day Retention Specification)

- **Thời gian lập:** 2026-10-21 16:21:00 (Asia/Bangkok)
- **Lane:** System Architect
- **Model:** gemini-2-5-pro
- **Thẩm quyền:** Chỉ đạo trực tiếp và phê duyệt từ Human Owner
- **Mã văn kiện đặc tả liên kết:** `SPEC-ARCH-20261021-DAILY-SWITCH-TIMELINE-SPEC`
- **Mã báo cáo:** `rp_system-architect_261021_162100_by_gemini-2-5-pro_daily_switch_timeline_specification`

---

## I. PHẠM VI CÔNG VIỆC HOÀN THÀNH (SCOPE COMPLETED)

1. **Khảo sát và phân tích hiện trạng:**
   - File `logs/switch_timeline.log` hiện là file mồ côi tĩnh lưu lại từ ngày 17/10/2026, chưa có cơ chế ghi tự động hàng ngày.
   - Thư mục `logs/` đã chuẩn hóa các thành phần khác (`quota_log/`, `active_threads_history/`, `ui_preferences.json`), việc để `switch_timeline.log` trôi nổi ở ngoài gốc làm mất tính đồng bộ kiến trúc.
2. **Thảo luận và thống nhất giải pháp cùng Human Owner:**
   - Thống nhất áp dụng mô hình phân mảnh theo ngày và dọn dẹp rolling 7 ngày cho `switch_timeline` tương tự như `quota_log`.
   - Di dời file log cũ vào thư mục con `logs/switch_timeline/2026-10-17_switch_timeline.log`.
   - Bổ sung ghi nhận các mốc sự kiện trong luồng `SwitchAccount` để phục vụ audit và điều tra sự cố.
3. **Biên soạn văn kiện đặc tả kỹ thuật chi tiết:**
   - Hoàn thành tài liệu [2026-10-21-daily-switch-timeline-sharding-and-retention-spec.md](file:///g:/antigravity_auto_switch/reports/architect/2026-10-21-daily-switch-timeline-sharding-and-retention-spec.md) (commit `462836a`).
   - Cung cấp sẵn mã nguồn 100% cho `internal/manager/timeline.go` (`LogSwitchEvent`, `PruneDailySwitchTimelineLogs`), chỉ dẫn tích hợp trong `SwitchAccount`, dọn dẹp khi app khởi động tại `internal/ui/app.go`, và 4 bài test tự động.
4. **Phê chuẩn từ Owner:**
   - Human Owner đã trực tiếp kiểm tra và ban hành lệnh: `"gui CEO"`.

---

## II. DANH MỤC TỆP ĐẦU RA (OUTPUT ARTIFACTS)

1. `reports/architect/2026-10-21-daily-switch-timeline-sharding-and-retention-spec.md` (Đặc tả kỹ thuật, commit `462836a`)
2. `reports/system-architect/rp_system-architect_261021_162100_by_gemini-2-5-pro_daily_switch_timeline_specification.md` (Báo cáo hoàn thành lane)

---

## III. QUYẾT ĐỊNH KIẾN TRÚC ĐÃ CHỐT (ARCHITECTURAL DECISIONS)

1. **Cấu trúc lưu trữ:** Phân mảnh nhật ký chuyển đổi tại `logs/switch_timeline/YYYY-MM-DD_switch_timeline.log`.
2. **Di dời file cũ:** Chuyển `logs/switch_timeline.log` cũ thành `logs/switch_timeline/2026-10-17_switch_timeline.log`. Thư mục `logs/` sạch bóng 100%, không còn file log mồ côi.
3. **Ghi log sự kiện:** Tự động append các mốc chuyển tài khoản (`SWITCH_START`, `SNAPSHOT_DONE`, `WINCRED_WRITE`, `KILL_LS`, `RAM_CHECK_DONE`, `DISPATCH_START`, `DISPATCH_DONE`, `SWITCH_SUCCESS`) vào file ngày hiện tại. Xử lý fail-safe, không ảnh hưởng critical path của switch.
4. **Dọn dẹp tự động:** Giữ tối đa 7 file gần nhất (tương ứng 7 ngày), tự động chạy trong goroutine nền nhẹ khi app khởi động xong.

---

## IV. VẤN ĐỀ CÒN TỒN ĐỌNG (RESIDUAL OPEN QUESTIONS)

- **Không có (None).** Văn kiện đặc tả đã được Owner phê chuẩn toàn bộ.

---

## V. ĐỀ XUẤT BÀN GIAO (HANDOFF RECOMMENDATION)

- **Verdict:** `READY_FOR_CODER`
- **Handoff Target:** Coder Lane nhận bản đặc tả `reports/architect/2026-10-21-daily-switch-timeline-sharding-and-retention-spec.md` để triển khai mã nguồn và bộ test tự động tại `internal/manager/`.
