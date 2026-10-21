# Đặc Tả Kiến Trúc: Tự Động Dọn Dẹp Lịch Sử Active Threads
(Active Threads History Pruning Specification & Implementation Blueprint)

- **Ngày lập:** 2026-10-21
- **Vai trò:** System Architect (Kiến trúc sư Hệ thống)
- **Thẩm quyền:** Phê chuẩn trực tiếp từ Human Owner
- **Mã văn kiện:** `SPEC-ARCH-20261021-ACTIVE-THREADS-HISTORY-PRUNING`
- **Trạng thái:** DRAFT (Trình Owner duyệt trước khi ban hành chính thức)
- **Đối tượng đọc:** Coder, Supervisor, QA

---

## I. BỐI CẢNH VÀ MỤC TIÊU (CONTEXT & OBJECTIVES)

### 1. Hiện trạng
- Mỗi lần ứng dụng thực hiện chuyển tài khoản (`SwitchAccount`), hệ thống chụp nhanh danh sách các hội thoại đang mở trong 2 giờ qua và lưu vào `accounts_store\active_threads_history\` dưới dạng file JSON `<timestamp>_active_threads.json`.
- Do chưa có cơ chế dọn dẹp, toàn bộ các file này bị dồn ứ vô hạn (hiện tại đã tích tụ **949 file rác** trong thư mục).
- Các snapshot cũ từ các phiên làm việc trước hoàn toàn không còn giá trị sử dụng cho cơ chế khôi phục phiên (*Session Continuation*).

### 2. Mục tiêu kiến trúc
1. **Tinh gọn vĩnh viễn:** Tự động cắt tỉa thư mục `accounts_store\active_threads_history\`, giữ lại tối đa **đúng 2 file mới nhất**.
2. **Không rườm rà (Single Trigger):** Kích hoạt duy nhất 1 lần tại thời điểm **hoàn tất chuyển tài khoản (Post-Switch Completion)**.
3. **An toàn tuyệt đối (Zero Critical Path Impact):** Quá trình dọn dẹp diễn ra sau khi phiên làm việc và prompt tiếp tục phiên đã được gửi thành công; không làm trễ hay ảnh hưởng đến kết quả chuyển tài khoản.
4. **Tự giải phóng bãi rác hiện tại:** Ngay trong lần chuyển tài khoản tiếp theo, toàn bộ 949 file rác tồn đọng sẽ được quét và xóa sạch chỉ còn lại đúng 2 file.

---

## II. ĐIỀU KHOẢN KỶ LUẬT SẮT (MANDATORY INVARIANTS)

1. **QUY TẮC GIỮ TỐI ĐA 2 FILE (MAX-2 RETENTION INVARIANT):**
   - Thư mục `accounts_store\active_threads_history\` chỉ được phép lưu tối đa **2 file snapshot mới nhất** (1 file của lần chuyển hiện tại + 1 file của lần chuyển liền kề trước đó làm dự phòng).
   - Mọi file thứ 3 trở đi theo thứ tự thời gian phải bị xóa bỏ hoàn toàn.
2. **RANH GIỚI THỜI ĐIỂM KÍCH HOẠT (TRIGGER BOUNDARY INVARIANT):**
   - Thao tác dọn dẹp CHỈ ĐƯỢC CHẠY sau khi Bước 4 (gửi prompt tiếp tục phiên - `PerformSessionContinuation`) đã hoàn thành và file snapshot của lần chuyển hiện tại đã được đọc/sử dụng xong.
   - Tuyệt đối KHÔNG chạy dọn dẹp ở giữa Bước 1 và Bước 2 nhằm tránh mọi rủi ro tranh chấp I/O trên critical path.
3. **KHÔNG GÂY TREO HAY PHÁ HỎNG KẾT QUẢ SWITCH (FAIL-SAFE INVARIANT):**
   - Lỗi phát sinh trong quá trình dọn dẹp (nếu có, ví dụ file bị tiến trình khác giữ khóa) chỉ ghi log cảnh báo, tuyệt đối KHÔNG được phép làm hàm chuyển tài khoản trả về `error` hoặc làm đảo ngược kết quả chuyển thành công.
4. **BẢO TỒN CẤU TRÚC CODEBASE (`internal/` PRESERVATION):**
   - Chỉ bổ sung logic dọn dẹp vào package `reloader` và gắn lệnh gọi tại cuối `SwitchAccount` trong package `manager`.
   - Không được sửa đổi schema của `ActiveThreadsSnapshot` hay định dạng tên file snapshot `YYYY-MM-DD-HH-mm-ss_active_threads.json`.

---

## III. THUẬT TOÁN CẮT TỈA (PRUNING ALGORITHM)

### 1. Tiêu chí sắp xếp và lọc file
- **Bộ lọc:** Chỉ xử lý các file thông thường (không phải thư mục) nằm trực tiếp trong thư mục `active_threads_history/`. Bỏ qua các file không phải snapshot nếu có.
- **Tiêu chí độ mới:**
  - Định dạng tên file chuẩn là `YYYY-MM-DD-HH-mm-ss_active_threads.json`.
  - Định dạng này có đặc tính tự nhiên là **sắp xếp theo tên giảm dần (alphabetical descending)** trùng khớp tuyệt đối với **thứ tự thời gian giảm dần (newest first)**.
  - Sắp xếp danh sách tên file theo thứ tự giảm dần: Phần tử index 0 là file mới nhất, index 1 là file mới nhì,...

### 2. Quy tắc cắt tỉa
- Gọi `N` là tổng số file snapshot hợp lệ tìm thấy:
  - Nếu `N <= 2`: Không thực hiện thao tác xóa nào, trả về kết quả thành công ngay.
  - Nếu `N > 2`:
    - Giữ lại `files[0]` và `files[1]`.
    - Lặp qua từ `files[2]` đến `files[N-1]` và gọi hàm xóa file của hệ thống (`os.Remove`).
    - Bỏ qua lỗi từng file riêng lẻ nếu xóa thất bại để tiếp tục xóa các file còn lại.

---

## IV. ĐẶC TẢ GIAO DIỆN & TÍCH HỢP (CONTRACTS & INTEGRATION)

### 1. Tầng `internal/reloader` (Nơi sở hữu logic dọn dẹp)
- **Vị trí file:** `internal/reloader/active_threads.go`
- **Quy chuẩn hợp đồng:**
  - Cung cấp hàm thực hiện dọn dẹp thư mục lịch sử:
    - Nhận vào: đường dẫn thư mục lịch sử (`historyDir string`) và số lượng file tối đa cần giữ (`keepCount int`, mặc định chuẩn là `2`).
    - Trả về: số lượng file đã xóa (`deletedCount int`) và lỗi tổng thể (`error`).
    - Nếu `historyDir` không tồn tại: Trả về `nil` (bỏ qua an toàn).
    - Nếu `keepCount <= 0`: Mặc định gán `keepCount = 2`.
  - Bổ sung phương thức wrapper trên `SessionReloader` (`internal/reloader/session_reloader.go`) để gọi tiện lợi khi có instance:
    - Phương thức sử dụng `OutputDir` sẵn có của struct, gọi hàm xử lý cốt lõi với `keepCount = 2`.

### 2. Tầng `internal/manager` (Điểm kích hoạt)
- **Vị trí file:** `internal/manager/account_manager.go`
- **Vị trí gọi trong hàm `SwitchAccount`:**
  - Ngay sau khối `BƯỚC 4: Gửi tuần tự toàn bộ danh sách (Continuation)` và trước khi trả về `&SwitchResult{ Success: true, ... }`.
  - Thực hiện gọi hàm dọn dẹp thư mục `active_threads_history`.
  - Thư mục đích lấy từ `m.reloader.OutputDir` (hoặc `reloader.GetDefaultThreadsHistoryDir()` nếu reloader nil).
  - Không chặn luồng nếu dọn dẹp gặp lỗi.

---

## V. HƯỚNG DẪN CHI TIẾT DÀNH CHO CODER (STEP-BY-STEP RECIPE)

> [!NOTE]
> Mục này là chỉ dẫn thực thi cụ thể để Coder đọc vào có thể code được ngay mà không cần suy nghĩ hay phán đoán thêm.

### Bước 1: Thêm logic dọn dẹp vào `internal/reloader/active_threads.go`
1. Khai báo hằng số mặc định:
   ```go
   const DefaultHistoryKeepCount = 2
   ```
2. Thêm hàm dọn dẹp (ví dụ `PruneActiveThreadsHistory`):
   ```go
   func PruneActiveThreadsHistory(historyDir string, keepCount int) (int, error)
   ```
   - Kiểm tra `keepCount <= 0` thì gán `keepCount = DefaultHistoryKeepCount`.
   - Nếu `historyDir == ""` thì lấy `GetDefaultThreadsHistoryDir()`.
   - Đọc danh mục: `entries, err := os.ReadDir(historyDir)`. Nếu thư mục không tồn tại (`os.IsNotExist(err)`), return `0, nil`. Nếu lỗi khác, return `0, err`.
   - Lọc danh sách: Chỉ lấy các file có `!entry.IsDir()` và kết thúc bằng `_active_threads.json`.
   - Sắp xếp: Dùng `sort.Slice` so sánh tên file giảm dần (`entries[i].Name() > entries[j].Name()`).
   - Nếu số lượng `<= keepCount`: return `0, nil`.
   - Lặp từ vị trí `keepCount` đến hết:
     - Tạo đường dẫn: `filepath.Join(historyDir, entries[k].Name())`.
     - Gọi `os.Remove(filePath)`.
     - Nếu thành công, tăng biến đếm `deleted++`.
   - Trả về `deleted, nil`.

### Bước 2: Bổ sung method trên `SessionReloader` tại `internal/reloader/session_reloader.go`
1. Thêm method:
   ```go
   func (r *SessionReloader) PruneActiveThreadsHistory(keepCount int) (int, error) {
       dir := r.OutputDir
       if dir == "" {
           dir = GetDefaultThreadsHistoryDir()
       }
       return PruneActiveThreadsHistory(dir, keepCount)
   }
   ```
2. Thêm package-level helper nếu cần:
   ```go
   func PruneActiveThreadsHistory(historyDir string, keepCount int) (int, error) {
       return defaultReloader.PruneActiveThreadsHistory(...) // hoặc gọi trực tiếp reloader.PruneActiveThreadsHistory
   }
   ```

### Bước 3: Tích hợp vào `SwitchAccount` trong `internal/manager/account_manager.go`
1. Tìm đến cuối hàm `SwitchAccount`, ngay sau Bước 4 (dòng ~602):
   ```go
   // =========================================================================
   // BƯỚC 5: Tự động dọn dẹp lịch sử snapshot (Chỉ giữ đúng 2 file mới nhất)
   // =========================================================================
   historyDir := ""
   if rel != nil && rel.OutputDir != "" {
       historyDir = rel.OutputDir
   } else {
       historyDir = filepath.Join(m.store.GetStoreDir(), "active_threads_history")
   }
   _, _ = reloader.PruneActiveThreadsHistory(historyDir, 2)
   ```
2. Đặt trước câu lệnh `return &SwitchResult{ Success: true, ... }, nil`.

---

## VI. MA TRẬN KIỂM THỬ & NGHIỆM THU (TEST MATRIX & ACCEPTANCE CRITERIA)

Coder và QA phải đảm bảo các bài test tự động sau vượt qua 100%:

### 1. Unit Tests tại `internal/reloader/active_threads_test.go`
- **TestPruneActiveThreadsHistory_UnderLimit:**
  - Tạo thư mục tạm với 0, 1 hoặc 2 file snapshot.
  - Gọi hàm dọn dẹp với `keepCount = 2`.
  - Kết quả: Không file nào bị xóa, trả về `deletedCount = 0`, không lỗi.
- **TestPruneActiveThreadsHistory_OverLimit:**
  - Tạo thư mục tạm chứa 10 file snapshot với các timestamp giả lập từ cũ đến mới (ví dụ `2026-10-18-...` đến `2026-10-21-...`).
  - Gọi hàm dọn dẹp với `keepCount = 2`.
  - Kết quả:
    - Đúng 8 file cũ bị xóa (`deletedCount = 8`).
    - Đúng 2 file có timestamp mới nhất vẫn tồn tại nguyên vẹn.
- **TestPruneActiveThreadsHistory_NonExistentDir:**
  - Truyền đường dẫn thư mục không tồn tại.
  - Kết quả: Trả về `deletedCount = 0, err = nil` mà không bị panic.

### 2. Integration Test tại `internal/manager/manager_test.go`
- **TestSwitchAccount_PrunesHistoryToMax2:**
  - Mock hoặc tạo sẵn 5 file snapshot cũ trong `active_threads_history` của test environment.
  - Thực hiện gọi `SwitchAccount`.
  - Sau khi `SwitchAccount` hoàn tất thành công:
    - Đếm số lượng file trong thư mục `active_threads_history`.
    - Kết quả: Số lượng file trong thư mục phải `<= 2`.

---

## VII. ĐỐI CHIẾU KIẾN TRÚC & PHÂN ĐỊNH (RULES VS SUGGESTIONS)

| Hạng mục | Bản chất | Ghi chú |
|---|---|---|
| **Giữ đúng tối đa 2 file snapshot** | **Architecture Rule (Bắt buộc)** | Vi phạm = Tích rác, không đạt mục tiêu |
| **Kích hoạt sau khi Bước 4 hoàn tất** | **Architecture Rule (Bắt buộc)** | Vi phạm = Chèn I/O vào critical path |
| **Không làm fail Switch khi dọn gặp lỗi** | **Architecture Rule (Bắt buộc)** | Vi phạm = Làm hỏng trải nghiệm người dùng |
| **Tên hàm `PruneActiveThreadsHistory`** | *Implementation Suggestion* | Coder có thể đặt tên phù hợp phong cách codebase |
| **Hằng số `DefaultHistoryKeepCount = 2`** | *Implementation Suggestion* | Giúp code sạch, tránh magic number |
