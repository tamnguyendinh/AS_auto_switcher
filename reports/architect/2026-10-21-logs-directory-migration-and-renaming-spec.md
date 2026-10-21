# Đặc Tả Kiến Trúc: Chuẩn Hóa Thư Mục Logs, Phân Mảnh Quota Theo Ngày & Tự Động Dọn Dẹp 7 Ngày
(Logs Directory Standardization, Daily Quota Sharding & 7-Day Rolling Retention Specification)

- **Ngày lập:** 2026-10-21
- **Vai trò:** System Architect (Kiến trúc sư Hệ thống)
- **Thẩm quyền:** Phê chuẩn trực tiếp từ Human Owner
- **Mã văn kiện:** `SPEC-ARCH-20261021-LOGS-AND-DAILY-QUOTA-SPEC`
- **Trạng thái:** DRAFT (Trình Human Owner phê duyệt)
- **Đối tượng đọc:** Coder, Supervisor, QA

---

## I. TỔNG QUAN VÀ MỤC TIÊU CỐT LÕI (OBJECTIVES)

Văn kiện này đặc tả toàn diện lộ trình chuyển đổi và chuẩn hóa cấu trúc lưu trữ runtime của dự án `antigravity_auto_switch` thành 5 giai đoạn tuần tự, dứt khoát:
1. **Cô lập Unit Test (Test Isolation):** Bắt buộc test sinh file vào `.tmp/`, xóa sổ rác rò rỉ vào cây mã nguồn.
2. **Quy tụ vào `logs/`:** Gom toàn bộ dữ liệu phụ trợ và telemetry từ `accounts_store/` về thư mục tập trung `logs/` và bảo vệ trên Git qua `.gitignore`.
3. **Đổi tên chuẩn hóa (Explicit Naming):** Xóa bỏ dấu chấm ẩn, định danh rõ ràng `ui_preferences.json` và `quota_log.json`. Không viết code chuyển tiếp dữ liệu cũ để tránh phức tạp hóa codebase.
4. **Phân mảnh Quota theo ngày (Daily Sharding):** Chuyển đổi `quota_log.json` thành thư mục `logs/quota_log/YYYY-MM-DD_quota_log.json` (mỗi ngày lưu một file độc lập).
5. **Dọn dẹp nhật ký 7 ngày (Rolling 7-Day Pruning):** Tự động quét và xóa các file log cũ khi ứng dụng khởi động xong, chỉ giữ lại đúng 7 file gần nhất tương ứng với 7 ngày gần nhất.

---

## II. ĐIỀU KHOẢN KỶ LUẬT SẮT (MANDATORY INVARIANTS)

1. **QUY TRÌNH 5 GIAI ĐOẠN TUẦN TỰ (STRICT 5-PHASE GATE):**
   - Coder phải triển khai theo đúng thứ tự từ Giai đoạn 1 đến Giai đoạn 5. Giai đoạn sau kế thừa kết quả của giai đoạn trước.
2. **KHO TÀI KHOẢN THUẦN KHIẾT (PURE ACCOUNT STORE INVARIANT):**
   - Sau khi hoàn thành Giai đoạn 2, thư mục `accounts_store/` CHỈ ĐƯỢC CHỨA các file credential tài khoản (`<email>.json`) và thư mục snapshot phiên (`active_threads_history/`). Tuyệt đối không chứa bất kỳ file log, cache hay preference nào.
3. **ĐỊNH DẠNG TÊN NHẬT KÝ THEO NGÀY (DAILY SHARDING INVARIANT):**
   - Tên file log quota hàng ngày phải tuân thủ chuẩn ISO: `YYYY-MM-DD_quota_log.json` (ví dụ: `2026-10-21_quota_log.json`).
   - Sắp xếp tên file giảm dần luôn đồng nghĩa với thứ tự thời gian giảm dần (mới nhất $\rightarrow$ cũ nhất).
4. **GIỮ ĐÚNG 7 FILE GẦN NHẤT (MAX-7 ROLLING RETENTION INVARIANT):**
   - Cơ chế dọn dẹp tại `logs/quota_log/` chỉ giữ lại tối đa **7 file mới nhất** (tương ứng 7 ngày gần nhất). Toàn bộ file cũ từ file thứ 8 trở đi phải bị xóa bỏ.
5. **KHỞI ĐỘNG NON-BLOCKING & FAIL-SAFE (STARTUP RESILIENCE):**
   - Lệnh dọn dẹp 7 ngày chạy sau khi app khởi động xong. Nếu gặp lỗi I/O (file bị khóa), hệ thống bỏ qua an toàn, tuyệt đối không làm treo hay gián đoạn trải nghiệm giao diện người dùng.
6. **NGUYÊN TẮC TINH GỌN (ZERO LEGACY MIGRATION BLOAT):**
   - Tuyệt đối không viết thêm các hàm/logic chuyển tiếp dữ liệu cũ (legacy migration) giữa các giai đoạn để tránh gây phình to và phức tạp hóa codebase. Mã nguồn chỉ trỏ trực tiếp vào cấu trúc đích.

---

## III. CHI TIẾT 5 GIAI ĐOẠN TRIỂN KHAI (5-PHASE ROADMAP)

### GIAI ĐOẠN 1: SỬA UNIT TEST & XÓA RÁC CÂY MÃ NGUỒN
- **Vị trí file:** `internal/telemetry/telemetry_test.go`
- **Thao tác:**
  1. Xóa bỏ hoàn toàn thư mục rác `internal/telemetry/accounts_store/`.
  2. Sửa toàn bộ các lệnh khởi tạo cache trong `telemetry_test.go` (đặc biệt dòng 945, 950):
     - Không truyền chuỗi rỗng `""`.
     - Truyền đường dẫn file nằm trong thư mục `.tmp/` (hoặc `t.TempDir()`).
- **Nghiệm thu:** Chạy `go test ./internal/telemetry/...` vượt qua 100%, không tái sinh thư mục `internal/telemetry/accounts_store/`.

---

### GIAI ĐOẠN 2: THIẾT LẬP THƯ MỤC `logs/` & DI DỜI 3 FILE
- **Thao tác:**
  1. **Cập nhật `.gitignore`:** Thêm dòng `logs/` vào file `.gitignore`.
  2. **Tạo thư mục:** Tạo thư mục `logs/` ở thư mục gốc của dự án.
  3. **Di dời tệp từ `accounts_store/` sang `logs/`:**
     - `accounts_store\.quota_cache.json` $\rightarrow$ `logs\.quota_cache.json`
     - `accounts_store\.ui_preferences.json` $\rightarrow$ `logs\.ui_preferences.json`
     - `accounts_store\switch_timeline.log` $\rightarrow$ `logs\switch_timeline.log`
  4. **Cập nhật mã nguồn trỏ sang `logs/`:**
     - Trong `internal/telemetry/cache.go`: Đổi `DefaultQuotaCachePath = "logs/.quota_cache.json"`.
     - Trong `internal/ui/app.go`: Đổi đường dẫn cache quota và UI preferences trỏ vào thư mục `logs/`.
- **Nghiệm thu:** `accounts_store/` không còn 3 file trên. Ứng dụng chạy đọc đúng dữ liệu từ `logs/`.

---

### GIAI ĐOẠN 3: ĐỔI TÊN CHUẨN HÓA (EXPLICIT RENAMING)
- **Thao tác:**
  1. **Đổi tên tệp trên đĩa:**
     - `logs\.ui_preferences.json` $\rightarrow$ `logs\ui_preferences.json`
     - `logs\.quota_cache.json` $\rightarrow$ `logs\quota_log.json`
  2. **Cập nhật mã nguồn:**
     - `internal/ui/app.go`: Trỏ đường dẫn preferences vào `logs/ui_preferences.json`.
     - `internal/telemetry/cache.go`: Trỏ `DefaultQuotaCachePath` vào `logs/quota_log.json`.
     - Trỏ trực tiếp vào tên file mới, không viết code chuyển tiếp dữ liệu cũ làm phức tạp codebase.
- **Nghiệm thu:** Tên file hiển thị minh bạch không có dấu chấm ẩn, giao diện nạp đúng cấu hình và quota.

---

### GIAI ĐOẠN 4: PHÂN MẢNH QUOTA THEO NGÀY (`logs/quota_log/YYYY-MM-DD_quota_log.json`)
- **Khái niệm:** `quota_log.json` không còn là 1 file đơn lẻ, mà chuyển thành một thư mục con `logs/quota_log/`.
- **Cấu trúc lưu trữ:**
  ```text
  logs/
  ├── quota_log/
  │   ├── 2026-10-21_quota_log.json
  │   ├── 2026-10-20_quota_log.json
  │   └── ...
  ├── ui_preferences.json
  └── switch_timeline.log
  ```
- **Quy tắc đọc và ghi nhật ký (Thuần khiết, không migration):**
  1. **Quy tắc xác định file ngày hôm nay:**
     - Tên file hôm nay: `fmt.Sprintf("%s_quota_log.json", time.Now().Format("2006-01-02"))`.
     - Đường dẫn: `filepath.Join("logs", "quota_log", filename)`.
  2. **Quy tắc ghi (Save):**
     - Mọi thao tác lưu quota trong ngày đều ghi trực tiếp vào file của ngày hôm đó (`logs/quota_log/YYYY-MM-DD_quota_log.json`).
     - Tự động tạo thư mục `logs/quota_log/` nếu chưa có (`os.MkdirAll`).
  3. **Quy tắc nạp khi khởi động (Load on Startup):**
     - Ưu tiên nạp file của ngày hôm nay nếu đã tồn tại.
     - Nếu file hôm nay chưa tồn tại (lần đầu mở app trong ngày mới): Tự động nạp file của ngày gần nhất trước đó trong `logs/quota_log/` để UI hiển thị thông số quota tức thì.
     - **Không thực hiện copy hay chuyển tiếp từ bất kỳ file cũ nào** bên ngoài thư mục `logs/quota_log/`.

---

### GIAI ĐOẠN 5: TỰ ĐỘNG DỌN DẸP 7 NGÀY GẦN NHẤT (ROLLING 7-DAY PRUNING)
- **Thời điểm kích hoạt:** Chạy tự động **mỗi khi ứng dụng khởi động xong (After App Startup)**.
- **Phạm vi xử lý:** Thư mục `logs/quota_log/`.
- **Thuật toán dọn dẹp:**
  1. Quét danh sách các file trong `logs/quota_log/` có đuôi `_quota_log.json`.
  2. Sắp xếp danh sách file theo thứ tự tên giảm dần (alphabetical descending) $\rightarrow$ Tương đương thứ tự từ ngày mới nhất đến ngày cũ nhất.
  3. **Quy tắc cắt tỉa:**
     - Nếu tổng số file $\le 7$: Dừng lại, giữ nguyên toàn bộ.
     - Nếu tổng số file $> 7$:
       + Giữ nguyên **7 file đầu tiên** (tương ứng với 7 ngày gần nhất).
       + Duyệt từ file thứ 8 trở đi và gọi `os.Remove` để xóa bỏ hoàn toàn.
  4. **Tính bền vững (Resilience):**
     - Chạy trong goroutine nền nhẹ nhàng sau khi khởi động giao diện xong để đảm bảo 0% độ trễ cho người dùng.
     - Bỏ qua các lỗi xóa file cục bộ nếu file đang bị lock, không làm phát sinh cảnh báo lỗi cho người dùng.

---

## IV. MA TRẬN KIỂM THỬ VÀ NGHIỆM THU (TEST MATRIX)

| Giai đoạn | Thao tác kiểm thử | Tiêu chuẩn ĐẠT (PASS) |
|---|---|---|
| **Phase 1** | Chạy `go test ./internal/telemetry/...` | PASS 100%, không sinh thư mục `internal/telemetry/accounts_store/`. |
| **Phase 2** | Kiểm tra thư mục `logs/` và `accounts_store/` | 3 file phụ chuyển sang `logs/`. `accounts_store/` sạch sẽ. Git ignore `logs/`. |
| **Phase 3** | Khởi động app và đọc cấu hình | `logs/ui_preferences.json` và `logs/quota_log.json` hiển thị đúng, không lỗi. |
| **Phase 4** | Poll quota telemetry và kiểm tra đĩa | File `logs/quota_log/YYYY-MM-DD_quota_log.json` được tạo và cập nhật đúng định dạng. |
| **Phase 5** | Tạo giả lập 10 file log từ các ngày trước trong test, khởi động dọn dẹp | Chỉ còn đúng 7 file của 7 ngày mới nhất, 3 file cũ bị xóa sạch sẽ. |
