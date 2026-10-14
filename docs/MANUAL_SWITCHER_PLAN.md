# Kế Hoạch Chi Tiết Triển Khai: Antigravity Multi-Account Switcher (Giai Đoạn 1)

> **Mục tiêu:** Xây dựng công cụ chuyển đổi tài khoản thủ công có giao diện đồ họa (UI) mượt mà, độc lập, đồng bộ tự động danh sách tài khoản từ Google Chrome, bảo mật token bằng Windows DPAPI, không làm mất session/thread làm việc của Antigravity.

---

## Danh Sách Nhiệm Vụ Chi Tiết (Implementation Checklist)

### PHẦN 1: TẦNG NỀN TẢNG HỆ THỐNG & WIN32 API (CORE ENGINE)
- [x] **Task 1.1: Xây dựng module tương tác Windows Credential Manager (`core/credential_manager.py`)**
  - [x] Gọi trực tiếp Win32 API (`Advapi32.dll` qua `ctypes`): `CredReadW`, `CredWriteW`, `CredDeleteW`.
  - [x] Đọc/Ghi chính xác target: `LegacyGeneric:target=gemini:antigravity` với Type = `Generic (1)`, Persist = `LocalMachine (2)`.
  - [x] Viết hàm parse cấu trúc JSON `CredentialBlob` lấy ra các trường: `access_token`, `refresh_token`, `id_token`, `expiry`, `token_type`.
- [x] **Task 1.2: Xây dựng module giải mã JWT định danh Email (`core/jwt_helper.py`)**
  - [x] Giải mã phần Payload của `id_token` bằng thuật toán Base64URL decode không cần thư viện ngoài.
  - [x] Bóc tách trường `email` và `sub` (Google User ID) để nhận diện chính xác danh tính tài khoản.
- [x] **Task 1.3: Xây dựng kho lưu trữ token bảo mật cục bộ (Vault) (`core/vault.py`)**
  - [x] Sử dụng **Windows DPAPI** (`Crypt32.dll` -> `CryptProtectData` & `CryptUnprotectData`) để mã hóa file dữ liệu `vault.enc`.
  - [x] Cấu trúc dữ liệu lưu trữ: `Dict[email, {tokens, last_used, added_at}]`.
  - [x] Đảm bảo chỉ đúng User Windows hiện tại mới có thể giải mã token.

---

### PHẦN 2: ENGINE QUÉT & ĐỒNG BỘ TỰ ĐỘNG CHROME (CHROME DISCOVERY)
- [x] **Task 2.1: Xây dựng module dò tìm thư mục Chrome (`scanner/chrome_scanner.py`)**
  - [x] Tự động định vị đường dẫn dữ liệu Chrome: `%LOCALAPPDATA%\Google\Chrome\User Data`.
  - [x] Quét tất cả các thư mục profile hiện có (`Default`, `Profile 1`, `Profile 2`, ...).
- [x] **Task 2.2: Trích xuất danh sách tài khoản Google đăng nhập (`scanner/profile_parser.py`)**
  - [x] Đọc file `Preferences` trong từng profile và `Local State`.
  - [x] Parse trường `account_tracker_service_last_update` và các key lưu cache gaia info.
  - [x] Trích xuất toàn bộ danh sách địa chỉ Gmail hợp lệ đang có phiên đăng nhập trên Chrome.
- [x] **Task 2.3: Bộ lọc và làm sạch dữ liệu (Filter & Sanitizer)**
  - [x] Chuẩn hóa chữ thường (lowercase) cho tất cả email.
  - [x] Loại bỏ các email trùng lặp giữa các profile.
  - [x] Áp dụng bộ lọc loại trừ các tài khoản không mong muốn.
- [x] **Task 2.4: Cơ chế đồng bộ động (Dynamic Sync & Purge)**
  - [x] Hàm kiểm tra đối chiếu: Nếu một email có trong Vault nhưng không còn tồn tại trong Chrome -> Tự động loại khỏi danh sách hiển thị trên UI.

---

### PHẦN 3: BỘ QUẢN LÝ TRẠNG THÁI (STATE MACHINE RESOLVER)
- [x] **Task 3.1: Xây dựng State Resolver (`core/state_resolver.py`)**
  - [x] Kết hợp 3 nguồn dữ liệu:
    1. Danh sách email từ Chrome (Live).
    2. Danh sách token trong Vault (Stored).
    3. Token hiện tại trong Windows Credential Manager (Active).
  - [x] Phân loại chính xác từng tài khoản thành 3 trạng thái:
    - 🟢 **ACTIVE:** Email trùng với token đang nạp trong Windows Credential.
    - 🟢 **READY (Xanh lá):** Email có trong Chrome VÀ đã có token hợp lệ trong Vault.
    - 🟡 **UNAUTHENTICATED (Vàng):** Email có trong Chrome NHƯNG chưa có token trong Vault.
- [x] **Task 3.2: Cơ chế tráo đổi token (Hot-Swap Execution)**
  - [x] Khi chọn một tài khoản 🟢: Lấy token từ Vault -> Ghi đè vào Windows Credential.
  - [x] Cập nhật timestamp `last_used`.

---

### PHẦN 4: CƠ CHẾ NẠP LẠI PHIÊN ANTIGRAVITY (SESSION RELOADER)
- [x] **Task 4.1: Tích hợp trigger nạp lại phiên Antigravity (`core/session_reloader.py`)**
  - [x] Phương án 1 (Ưu tiên): Gửi tín hiệu phím tắt / lệnh reload window của Electron vào tiến trình Antigravity đang chạy.
  - [x] Phương án 2: Restart tiến trình backend `language_server.exe` ngầm (Antigravity tự động reconnect trong 1 giây mà không tắt giao diện).
  - [x] Đảm bảo 100% giữ nguyên thread chat và nội dung đang làm việc.

---

### PHẦN 5: CƠ CHẾ KÍCH HOẠT OAUTH 1 LẦN (YELLOW ➔ GREEN)
- [x] **Task 5.1: Xây dựng cơ chế Auto-Capture phiên đăng nhập**
  - [x] Khi Antigravity đăng nhập phiên mới, `AccountManager` tự động trích xuất token và email nạp vào Vault.
  - [x] Tự động chuyển trạng thái của tài khoản từ 🟡 sang 🟢 vĩnh viễn.

---

### PHẦN 6: GIAO DIỆN NGƯỜI DÙNG (MODERN UI & SYSTEM TRAY)
- [x] **Task 6.1: Xây dựng khung giao diện chính (`ui/app_window.py`)**
  - [x] Sử dụng UI nhẹ, hiện đại, hỗ trợ Dark Mode.
  - [x] Thiết kế dạng Mini Window có tùy chọn "Always on Top" (Ghim trên cùng) để tiện thao tác khi đang code.
- [x] **Task 6.2: Hiển thị danh sách tài khoản theo đèn trạng thái**
  - [x] Hiển thị chấm 🟢 (Active), 🟢 (Ready), 🟡 (Unauthenticated) rõ ràng theo từng dòng.
  - [x] Nút bấm hành động tương ứng:
    - Với 🟢: Nút **[⚡ Chuyển sang]**.
    - Với 🟡: Nút **[🔑 Kích hoạt]**.
  - [x] Nút **[🔄 Làm mới Chrome]** để quét lại ngay lập tức khi người dùng vừa thêm/bớt tài khoản trên trình duyệt.

---

### PHẦN 7: KIỂM THỬ TỪNG PHẦN & TOÀN DIỆN (TESTING & VERIFICATION)
- [x] **Task 7.1: Kiểm thử Unit Test các module nền tảng**
  - [x] Test đọc/ghi Windows Credential Manager với dữ liệu mẫu an toàn.
  - [x] Test giải mã JWT payload và trích xuất email.
  - [x] Test mã hóa/giải mã DPAPI.
- [x] **Task 7.2: Kiểm thử quét Chrome Profile thực tế**
  - [x] Xác nhận đọc được danh sách email chuẩn từ Chrome trên máy.
- [x] **Task 7.3: Kiểm thử luồng chuyển đổi tài khoản thực tế với Antigravity**
  - [x] Kiểm tra phân giải trạng thái 🟢 ACTIVE, 🟢 READY, 🟡 UNAUTH trên hệ thống thực tế.

---

### PHẦN 8: ĐÓNG GÓI & TÀI LIỆU
- [x] **Task 8.1: Tạo file khởi chạy chính (`main.py`)**
- [x] **Task 8.2: Viết hướng dẫn sử dụng chi tiết (`README.md`)**
