# Kế Hoạch & Thiết Kế Kiến Trúc: Antigravity Multi-Account Switcher (Giai Đoạn 1)

## 1. Mục Tiêu Dự Án
Xây dựng một công cụ hỗ trợ người dùng Antigravity quản lý và chuyển đổi qua lại giữa nhiều tài khoản Google một cách mượt mà, không làm gián đoạn luồng làm việc (flow), không mất lịch sử/thread chat, và không cần mở trình duyệt Chrome để thao tác thủ công.

---

## 2. Lộ Trình Triển Khai Tổng Thể (2 Bước)
- **Giai đoạn 1 (Hiện tại):** Xây dựng Tool có giao diện đồ họa (UI) phục vụ việc nhận diện tài khoản Chrome, quản lý kho token và chuyển đổi tài khoản thủ công (1-Click Switch).
- **Giai đoạn 2 (Sau khi Giai đoạn 1 ổn định 100%):** Tích hợp module tự động phát hiện giới hạn quota (5h limit hoặc weekly limit) để tự động gọi hàm chuyển đổi của Giai đoạn 1 (Auto-switch).

---

## 3. Kiến Trúc Hoạt Động Của Antigravity (Đã Khảo Sát & Xác Thực Thực Tế)
1. **Quản lý phiên (Session & State):**
   - Lịch sử chat, thread, workspace được lưu trữ độc lập tại SQLite (`conversation_summaries.db`) và các file `.pb` trong `C:\Users\<User>\.gemini\antigravity\conversations`.
   - Khi thay đổi tài khoản xác thực, dữ liệu làm việc hiện tại **không bị mất**.
2. **Kho lưu trữ Token (Auth Store):**
   - Lưu trữ tại **Windows Credential Manager**:
     - `Target: LegacyGeneric:target=gemini:antigravity`
     - `User: antigravity`
     - `CredentialBlob`: Chuỗi JSON chứa `access_token`, `refresh_token`, `id_token`, `expiry`, `token_type`.
   - Địa chỉ Email thực tế được mã hóa bên trong payload của `id_token` (chuẩn JWT).
3. **Tiến trình Backend:**
   - `language_server.exe` thực hiện toàn bộ request AI lên Google API (`daily-cloudcode-pa.googleapis.com`).

---

## 4. Đặc Tả Chi Tiết Giai Đoạn 1: Tool Chuyển Đổi Thủ Công (UI)

### 4.1. Khám Phá Tài Khoản Động Từ Google Chrome (Chrome Auto-Discovery)
- Tool tự động đọc cấu hình Chrome Profile tại:
  `%LOCALAPPDATA%\Google\Chrome\User Data\Profile 1\Preferences` (và các profile khác trong `User Data`).
- Quét danh sách các tài khoản Google đang đăng nhập phiên web.
- **Quy tắc sống động:**
  - Nếu tài khoản có trong Chrome -> Hiển thị trên danh sách của Tool.
  - Nếu người dùng đăng xuất tài khoản khỏi Chrome -> Tool tự động loại bỏ tài khoản đó khỏi danh sách (tự biến mất, không để lại rác).

### 4.2. Máy Trạng Thái Của Tài Khoản (State Machine)
Mỗi tài khoản hiển thị trên UI sẽ có một trong các trạng thái sau:

| Biểu Tượng | Trạng Thái | Điều Kiện Kỹ Thuật | Hành Vi Trên Giao Diện |
| :---: | :--- | :--- | :--- |
| 🟢 | **Đã Auth (Sẵn sàng)** | Tồn tại trên Chrome **VÀ** Đã có token trong Kho lưu trữ (Vault) của Tool | Hiển thị nút **[⚡ Chuyển sang]** |
| 🟢 | **Đang Hoạt Động** | Token đang nằm trong `LegacyGeneric:target=gemini:antigravity` | Hiển thị huy hiệu **[Đang hoạt động]** |
| 🟡 | **Chưa Auth** | Tồn tại trên Chrome **NHƯNG** chưa có token trong Kho lưu trữ của Tool | Hiển thị nút **[🔑 Kích hoạt]** |
| ❌ | **Đã Xóa Khỏi Chrome** | Không còn xuất hiện trong Preferences của Chrome | Tự động biến mất khỏi danh sách |

### 4.3. Quy Trình Nghiệp Vụ Của Tool

#### Quy trình 1: Kích hoạt tài khoản mới (Chuyển từ 🟡 sang 🟢)
1. Người dùng bấm **[🔑 Kích hoạt]** ở tài khoản có chấm vàng.
2. Tool khởi tạo luồng OAuth của Antigravity với tham số `login_hint=<email>`.
3. Chrome tự động chọn đúng tài khoản tương ứng mà không hiện bảng Account Chooser bắt người dùng click chuột.
4. Token trả về được Tool tự động trích xuất và lưu vào Kho bảo mật cục bộ (Vault).
5. Trạng thái của tài khoản trên UI đổi ngay lập tức thành 🟢 **Xanh lá**.

#### Quy trình 2: Chuyển đổi giữa các tài khoản 🟢 Xanh lá (Hot-Swap)
1. Người dùng bấm **[⚡ Chuyển sang]** tại một tài khoản xanh lá khác.
2. Tool lấy gói token tương ứng từ Kho (Vault) và ghi đè vào Windows Credential Manager thông qua Win32 API `CredWrite`.
3. Tool kích hoạt tín hiệu nạp lại session của Antigravity (qua Window Reload hoặc API nội bộ).
4. Antigravity chuyển sang sử dụng tài khoản mới. Quá trình hoàn tất trong ~1-2 giây, không mở trình duyệt.

---

## 5. Thiết Kế Giao Diện Người Dùng (UI Mockup)
Giao diện được thiết kế dạng cửa sổ nhỏ gọn (Mini Floating Window hoặc System Tray widget):

```text
┌────────────────────────────────────────────────────────┐
│  ⚡ Antigravity Multi-Account Switcher                 │
├────────────────────────────────────────────────────────┤
│  Tài khoản Google (Đồng bộ Chrome):                    │
│                                                        │
│  🟢 hiendao1760@gmail.com            [ĐANG HOẠT ĐỘNG]  │
│                                                        │
│  🟢 daohien204ccyersin@gmail.com      [ ⚡ Chuyển sang ]│
│                                                        │
│  🟡 ngau2hung@gmail.com               [ 🔑 Kích hoạt ] │
│  🟡 skey.miennam@gmail.com            [ 🔑 Kích hoạt ] │
│                                                        │
├────────────────────────────────────────────────────────┤
│  [🔄 Làm mới Chrome]                Trạng thái: Sẵn sàng│
└────────────────────────────────────────────────────────┘
```

---

## 6. Bảo Mật & Lưu Trữ Cục Bộ
- Toàn bộ token trong Kho lưu trữ (Vault) được mã hóa bằng **Windows Data Protection API (DPAPI - CryptProtectData)**.
- Chỉ người dùng hiện tại đang đăng nhập vào phiên Windows mới có quyền giải mã.
- Tuyệt đối không gửi token ra ngoài môi trường máy tính của người dùng.
