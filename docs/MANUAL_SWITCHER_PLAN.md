# Kế Hoạch Chi Tiết Triển Khai: Antigravity Multi-Account Switcher (Giai Đoạn 1)

> **Mục tiêu:** Xây dựng công cụ chuyển đổi tài khoản thủ công có giao diện đồ họa (UI) mượt mà, độc lập, đồng bộ tự động danh sách tài khoản từ Google Chrome, bảo mật token bằng Windows DPAPI, không làm mất session/thread làm việc của Antigravity.

---

## Danh Sách Nhiệm Vụ Chi Tiết (Implementation Checklist)

### PHẦN 1: TẦNG NỀN TẢNG HỆ THỐNG & WIN32 API (CORE ENGINE)
- [ ] **Task 1.1: Xây dựng module tương tác Windows Credential Manager (`core/credential_manager.py`)**
  - [ ] Gọi trực tiếp Win32 API (`Advapi32.dll` qua `ctypes`): `CredReadW`, `CredWriteW`, `CredDeleteW`.
  - [ ] Đọc/Ghi chính xác target: `LegacyGeneric:target=gemini:antigravity` với Type = `Generic (1)`, Persist = `LocalMachine (2)`.
  - [ ] Viết hàm parse cấu trúc JSON `CredentialBlob` lấy ra các trường: `access_token`, `refresh_token`, `id_token`, `expiry`, `token_type`.
- [ ] **Task 1.2: Xây dựng module giải mã JWT định danh Email (`core/jwt_helper.py`)**
  - [ ] Giải mã phần Payload của `id_token` bằng thuật toán Base64URL decode không cần thư viện ngoài.
  - [ ] Bóc tách trường `email` và `sub` (Google User ID) để nhận diện chính xác danh tính tài khoản.
- [ ] **Task 1.3: Xây dựng kho lưu trữ token bảo mật cục bộ (Vault) (`core/vault.py`)**
  - [ ] Sử dụng **Windows DPAPI** (`Crypt32.dll` -> `CryptProtectData` & `CryptUnprotectData`) để mã hóa file dữ liệu `vault.enc`.
  - [ ] Cấu trúc dữ liệu lưu trữ: `Dict[email, {tokens, last_used, added_at}]`.
  - [ ] Đảm bảo chỉ đúng User Windows hiện tại mới có thể giải mã token.

---

### PHẦN 2: ENGINE QUÉT & ĐỒNG BỘ TỰ ĐỘNG CHROME (CHROME DISCOVERY)
- [ ] **Task 2.1: Xây dựng module dò tìm thư mục Chrome (`scanner/chrome_scanner.py`)**
  - [ ] Tự động định vị đường dẫn dữ liệu Chrome: `%LOCALAPPDATA%\Google\Chrome\User Data`.
  - [ ] Quét tất cả các thư mục profile hiện có (`Default`, `Profile 1`, `Profile 2`, ...).
- [ ] **Task 2.2: Trích xuất danh sách tài khoản Google đăng nhập (`scanner/profile_parser.py`)**
  - [ ] Đọc file `Preferences` trong từng profile và `Local State`.
  - [ ] Parse trường `account_tracker_service_last_update` và các key lưu cache gaia info.
  - [ ] Trích xuất toàn bộ danh sách địa chỉ Gmail hợp lệ đang có phiên đăng nhập trên Chrome.
- [ ] **Task 2.3: Bộ lọc và làm sạch dữ liệu (Filter & Sanitizer)**
  - [ ] Chuẩn hóa chữ thường (lowercase) cho tất cả email.
  - [ ] Loại bỏ các email trùng lặp giữa các profile.
  - [ ] Áp dụng bộ lọc loại trừ các tài khoản không mong muốn.
- [ ] **Task 2.4: Cơ chế đồng bộ động (Dynamic Sync & Purge)**
  - [ ] Hàm kiểm tra đối chiếu: Nếu một email có trong Vault nhưng không còn tồn tại trong Chrome -> Tự động loại khỏi danh sách hiển thị trên UI.

---

### PHẦN 3: BỘ QUẢN LÝ TRẠNG THÁI (STATE MACHINE RESOLVER)
- [ ] **Task 3.1: Xây dựng State Resolver (`core/state_resolver.py`)**
  - [ ] Kết hợp 3 nguồn dữ liệu:
    1. Danh sách email từ Chrome (Live).
    2. Danh sách token trong Vault (Stored).
    3. Token hiện tại trong Windows Credential Manager (Active).
  - [ ] Phân loại chính xác từng tài khoản thành 3 trạng thái:
    - 🟢 **ACTIVE:** Email trùng với token đang nạp trong Windows Credential.
    - 🟢 **READY (Xanh lá):** Email có trong Chrome VÀ đã có token hợp lệ trong Vault.
    - 🟡 **UNAUTHENTICATED (Vàng):** Email có trong Chrome NHƯNG chưa có token trong Vault.
- [ ] **Task 3.2: Cơ chế tráo đổi token (Hot-Swap Execution)**
  - [ ] Khi chọn một tài khoản 🟢: Lấy token từ Vault -> Ghi đè vào Windows Credential.
  - [ ] Cập nhật timestamp `last_used`.

---

### PHẦN 4: CƠ CHẾ NẠP LẠI PHIÊN ANTIGRAVITY (SESSION RELOADER)
- [ ] **Task 4.1: Tích hợp trigger nạp lại phiên Antigravity (`core/session_reloader.py`)**
  - [ ] Phương án 1 (Ưu tiên): Gửi tín hiệu phím tắt / lệnh reload window của Electron vào tiến trình Antigravity đang chạy.
  - [ ] Phương án 2: Restart tiến trình backend `language_server.exe` ngầm (Antigravity tự động reconnect trong 1 giây mà không tắt giao diện).
  - [ ] Đảm bảo 100% giữ nguyên thread chat và nội dung đang làm việc.

---

### PHẦN 5: CƠ CHẾ KÍCH HOẠT OAUTH 1 LẦN (YELLOW ➔ GREEN)
- [ ] **Task 5.1: Xây dựng OAuth Helper (`auth/oauth_helper.py`)**
  - [ ] Khởi tạo luồng OAuth với tham số `login_hint=<target_email>`.
  - [ ] Khi Chrome mở URL xác thực, tham số `login_hint` sẽ yêu cầu Chrome chọn thẳng tài khoản đó mà không hiển thị Account Chooser.
  - [ ] Lắng nghe token phản hồi từ cổng callback loopback của Antigravity.
  - [ ] Lưu token thu được vào Vault và lập tức cập nhật trạng thái sang 🟢.

---

### PHẦN 6: GIAO DIỆN NGƯỜI DÙNG (MODERN UI & SYSTEM TRAY)
- [ ] **Task 6.1: Xây dựng khung giao diện chính (`ui/app_window.py`)**
  - [ ] Sử dụng UI nhẹ, hiện đại, hỗ trợ Dark Mode.
  - [ ] Thiết kế dạng Mini Window có tùy chọn "Always on Top" (Ghim trên cùng) để tiện thao tác khi đang code.
- [ ] **Task 6.2: Hiển thị danh sách tài khoản theo đèn trạng thái**
  - [ ] Hiển thị chấm 🟢 (Active), 🟢 (Ready), 🟡 (Unauthenticated) rõ ràng theo từng dòng.
  - [ ] Nút bấm hành động tương ứng:
    - Với 🟢: Nút **[⚡ Chuyển sang]**.
    - Với 🟡: Nút **[🔑 Kích hoạt]**.
  - [ ] Nút **[🔄 Làm mới Chrome]** để quét lại ngay lập tức khi người dùng vừa thêm/bớt tài khoản trên trình duyệt.
- [ ] **Task 6.3: Tích hợp Khay hệ thống (System Tray) (`ui/tray_icon.py`)**
  - [ ] Thu nhỏ xuống góc đồng hồ Windows khi bấm nút đóng (X).
  - [ ] Menu chuột phải ở khay hệ thống cho phép chuyển nhanh tài khoản mà không cần mở giao diện chính.

---

### PHẦN 7: KIỂM THỬ TỪNG PHẦN & TOÀN DIỆN (TESTING & VERIFICATION)
- [ ] **Task 7.1: Kiểm thử Unit Test các module nền tảng**
  - [ ] Test đọc/ghi Windows Credential Manager với dữ liệu mẫu an toàn.
  - [ ] Test giải mã JWT payload và trích xuất email.
  - [ ] Test mã hóa/giải mã DPAPI.
- [ ] **Task 7.2: Kiểm thử quét Chrome Profile thực tế**
  - [ ] Xác nhận đọc được danh sách email chuẩn từ Chrome trên máy.
- [ ] **Task 7.3: Kiểm thử luồng chuyển đổi tài khoản thực tế với Antigravity**
  - [ ] Thực hiện chuyển từ tài khoản hiện tại sang tài khoản thứ hai.
  - [ ] Kiểm tra xem Antigravity có nhận tài khoản mới hay không.
  - [ ] Kiểm tra thread chat cũ có tiếp tục hoạt động bình thường không.

---

### PHẦN 8: ĐÓNG GÓI & TÀI LIỆU
- [ ] **Task 8.1: Đóng gói thành file chạy duy nhất (`antigravity-switcher.exe`)**
  - [ ] Tạo file thực thi độc lập bằng PyInstaller / Nuitka, không yêu cầu cài đặt Python hay môi trường ngoài.
- [ ] **Task 8.2: Viết hướng dẫn sử dụng chi tiết (`README.md`)**
