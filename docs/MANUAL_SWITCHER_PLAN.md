# Kế Hoạch Chi Tiết: Antigravity Multi-Account Switcher (Kiến Trúc JSON Storage)

> **Mục tiêu:** Xây dựng công cụ chuyển đổi tài khoản thủ công có giao diện đồ họa (UI) mượt mà, độc lập. Tool đóng vai trò là "Kho lưu trữ vĩnh viễn thay cho Antigravity" bằng các file JSON riêng biệt, tự động đồng bộ tài khoản từ Chrome, không làm mất session/thread chat của Antigravity.

---

## 1. Nguyên Lý Lưu Trữ Bền Vững (Persistent Multi-Account Store)
- **Hạn chế của Antigravity:** Chỉ lưu 1 tài khoản duy nhất tại `LegacyGeneric:target=gemini:antigravity`. Khi người dùng đổi tài khoản, Antigravity ghi đè và xóa mất token của tài khoản cũ.
- **Giải pháp của Tool:**
  - Tool tạo thư mục `accounts_store/` trực tiếp trong thư mục ứng dụng.
  - Mỗi tài khoản một khi đã được kích hoạt sẽ được lưu thành 1 file JSON độc lập:
    `accounts_store/<email>.json`
  - Bên trong chứa toàn bộ gói Token (`access_token`, `refresh_token`, `id_token`, `expiry`, `token_type`).
  - Khi cần đổi tài khoản: Tool chỉ việc đọc file JSON tương ứng và nạp thẳng vào Windows Credential Manager của Antigravity. Cho dù Antigravity có ghi đè bao nhiêu lần, kho file JSON của Tool vẫn còn nguyên vẹn 100%.

---

## 2. Máy Trạng Thái Tài Khoản (State Machine)
- 🟢 **ACTIVE [ĐANG DÙNG]:** Email trùng với tài khoản đang nạp trong Antigravity.
- 🟢 **READY [⚡ Chuyển sang]:** Tài khoản có trong Chrome VÀ ĐÃ CÓ file `accounts_store/<email>.json`.
- 🟡 **UNAUTH [Chưa lưu vào kho]:** Tài khoản có trong Chrome NHƯNG CHƯA CÓ file JSON trong `accounts_store/`.
- ❌ **Tự động biến mất:** Nếu tài khoản bị đăng xuất khỏi Chrome, tool tự động ẩn khỏi danh sách.

---

## 3. Danh Sách Nhiệm Vụ Triển Khai (Checklist)

### PHẦN 1: TẦNG LƯU TRỮ JSON FILE (`core/account_store.py`)
- [ ] **Task 1.1:** Tạo thư mục lưu trữ `accounts_store/`.
- [ ] **Task 1.2:** Viết module `AccountStore` quản lý từng file `accounts_store/<email>.json`.
- [ ] **Task 1.3:** Tự động nạp tài khoản đang hoạt động của Antigravity thành file JSON đầu tiên.
- [ ] **Task 1.4:** Thêm thư mục `accounts_store/` vào `.gitignore` để bảo vệ token cá nhân.

### PHẦN 2: ENGINE QUÉT CHROME & PHÂN GIẢI TRẠNG THÁI (`scanner/`, `core/state_resolver.py`)
- [ ] **Task 2.1:** Quét danh sách Gmail từ Chrome Profile (đảm bảo lọc sạch từ khóa cấm).
- [ ] **Task 2.2:** Đối chiếu với các file `.json` trong `accounts_store/` để gán nhãn 🟢 hoặc 🟡.
- [ ] **Task 2.3:** Cơ chế Hot-swap: Đọc `<email>.json` -> ghi vào Windows Credential -> gửi tín hiệu nạp lại Antigravity.

### PHẦN 3: GIAO DIỆN NGƯỜI DÙNG TỐI ƯU (`ui/app_window.py`)
- [ ] **Task 3.1:** Hiển thị trực quan danh sách tài khoản theo đèn 🟢 và 🟡.
- [ ] **Task 3.2:** Nút **[⚡ Chuyển sang]** cho các tài khoản 🟢: Bấm là đổi ngay trong 1 giây.
- [ ] **Task 3.3:** Nút **[💾 Lưu phiên này vào kho]**: Cho phép lưu tài khoản đang chạy thành file JSON bất cứ lúc nào.
- [ ] **Task 3.4:** Nút **[🔄 Làm mới Chrome]**: Quét lại trạng thái tức thì.

### PHẦN 4: KIỂM THỬ & ĐÓNG GÓI
- [ ] **Task 4.1:** Cập nhật Unit Tests kiểm thử việc đọc/ghi các file `accounts_store/<email>.json`.
- [ ] **Task 4.2:** Chạy kiểm thử tự động, xác nhận 100% test case vượt qua.
- [ ] **Task 4.3:** Cập nhật script khởi chạy `run.bat` và `main.py`.
