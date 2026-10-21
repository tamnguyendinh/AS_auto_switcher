# Báo Cáo Kiến Trúc: Chuẩn Hóa Thư Mục Logs, Phân Mảnh Quota Theo Ngày & Di Dời Active Threads History
(System Architect Lane Completion Report — Logs Directory Standardization & Daily Quota Specification)

- **Thời gian lập:** 2026-10-21 14:23:00 (Asia/Bangkok)
- **Lane:** System Architect
- **Model:** gemini-2-5-pro
- **Thẩm quyền:** Chỉ đạo trực tiếp và phê duyệt từ Human Owner
- **Mã văn kiện đặc tả liên kết:** `SPEC-ARCH-20261021-LOGS-AND-DAILY-QUOTA-SPEC`
- **Mã báo cáo:** `rp_system-architect_261021_142300_by_gemini-2-5-pro_logs_directory_standardization_and_daily_quota_specification`

---

## I. PHẠM VI CÔNG VIỆC HOÀN THÀNH (SCOPE COMPLETED)

1. **Điều tra sâu nguyên nhân rò rỉ rác:**
   - Phát hiện thư mục `internal\telemetry\accounts_store\.quota_cache.json` là rác phát sinh do unit test `telemetry_test.go:945` gọi `NewQuotaCache("")` làm rơi vào đường dẫn tương đối khi chạy `go test`.
2. **Thảo luận và thống nhất giải pháp cùng Human Owner:**
   - Thống nhất nguyên lý dứt khoát (không dùng "hoặc", không chia nhánh).
   - Xác định `.quota_cache.json` bản chất là telemetry status log, gom toàn bộ 3 file phụ từ `accounts_store/` về thư mục tập trung `logs/`.
   - Cắt bỏ hoàn toàn mọi logic legacy migration bloat để giữ codebase tinh gọn.
   - Thống nhất mở rộng thêm Giai đoạn 4 (phân mảnh quota theo ngày `logs/quota_log/YYYY-MM-DD_quota_log.json`), Giai đoạn 5 (dọn dẹp rolling 7 ngày gần nhất khi khởi động xong) và Giai đoạn 6 (di dời `active_threads_history/` sang `logs/`).
3. **Đối soát 100% codebase thực tế (Zero-Trust Cross-Verification):**
   - Đã phát hiện và sửa 4 điểm kỹ thuật trong bản thảo cũ:
     - Vị trí `DefaultUIPreferencesPath` tại `internal/ui/themes.go:27`.
     - Logic đường dẫn `logsDir` trong `internal/ui/app.go:100-104`.
     - Quy chuẩn đường dẫn `sessionReloader.OutputDir` đồng bộ với `discovered_accounts` tại `internal/manager/account_manager.go:91, 96, 609-612`.
     - Đồng bộ câu lệnh assert tại `internal/manager/two_store_test.go:70`.
4. **Biên soạn văn kiện đặc tả chính thức:**
   - Hoàn tất tài liệu [2026-10-21-logs-directory-migration-and-renaming-spec.md](file:///g:/antigravity_auto_switch/reports/architect/2026-10-21-logs-directory-migration-and-renaming-spec.md) (commit `31c2285`) với đầy đủ step-by-step recipe cho từng file.
5. **Phê chuẩn từ Owner:**
   - Human Owner đã trực tiếp kiểm tra và ban hành lệnh: `"gửi CEO"`.

---

## II. DANH MỤC TỆP ĐẦU RA (OUTPUT ARTIFACTS)

1. `reports/architect/2026-10-21-logs-directory-migration-and-renaming-spec.md` (Đặc tả kiến trúc 6 giai đoạn, commit `31c2285`)
2. `reports/system-architect/rp_system-architect_261021_142300_by_gemini-2-5-pro_logs_directory_standardization_and_daily_quota_specification.md` (Báo cáo hoàn thành lane)

---

## III. QUYẾT ĐỊNH KIẾN TRÚC ĐÃ CHỐT (ARCHITECTURAL DECISIONS)

1. **Phase 1 (Cô lập test):** Sửa `telemetry_test.go` trỏ vào `tempDir` (`.tmp/`), xóa thư mục `internal/telemetry/accounts_store/`.
2. **Phase 2 (Thiết lập `logs/`):** Thêm `logs/` vào `.gitignore`, di dời 3 file phụ từ `accounts_store/` sang `logs/`.
3. **Phase 3 (Đổi tên chuẩn hóa):** Đổi thành `ui_preferences.json` và `quota_log.json` (bỏ dấu chấm ẩn). Không viết code chuyển tiếp thừa.
4. **Phase 4 (Phân mảnh theo ngày):** Lưu tại `logs/quota_log/YYYY-MM-DD_quota_log.json`. Nạp file hôm nay hoặc file ngày gần nhất.
5. **Phase 5 (Dọn dẹp 7 ngày):** Tự động dọn dẹp `logs/quota_log/` khi app khởi động xong, chỉ giữ lại đúng 7 file gần nhất.
6. **Phase 6 (Thuần khiết hóa `accounts_store/`):** Di dời `active_threads_history/` sang `logs/active_threads_history/`. `accounts_store/` trở thành kho 100% thuần khiết chỉ chứa duy nhất các file `<email>.json`.

---

## IV. VẤN ĐỀ CÒN TỒN ĐỌNG (RESIDUAL OPEN QUESTIONS)

- **Không có (None).** Văn kiện đặc tả đã được Owner phê chuẩn toàn bộ.

---

## V. ĐỀ XUẤT BÀN GIAO (HANDOFF RECOMMENDATION)

- **Verdict:** `READY_FOR_CODER`
- **Handoff Target:** Coder Lane nhận bản đặc tả `reports/architect/2026-10-21-logs-directory-migration-and-renaming-spec.md` để triển khai tuần tự 6 Phase và chạy bộ test tự động.
