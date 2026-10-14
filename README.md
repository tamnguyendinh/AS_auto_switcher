# ⚡ Antigravity Multi-Account Switcher

Công cụ quản lý và chuyển đổi qua lại giữa nhiều tài khoản Google trên Antigravity IDE một cách mượt mà, độc lập, không làm mất session/thread làm việc, và không cần mở trình duyệt Chrome thủ công mỗi khi hết hạn mức (quota limit).

---

## 🌟 Tính Năng Nổi Bật

1. **Đồng bộ tự động từ Google Chrome (Chrome Auto-Discovery):**
   - Tự động nhận diện tất cả các tài khoản Google đang đăng nhập trên trình duyệt Chrome.
   - Khi tài khoản bị đăng xuất khỏi Chrome, nó sẽ tự động biến mất khỏi danh sách của Tool.
2. **Máy trạng thái trực quan (Status State Machine):**
   - 🟢 **Xanh lá [ĐANG DÙNG]:** Tài khoản hiện tại đang được kết nối với Antigravity.
   - 🟢 **Xanh lá [⚡ Chuyển sang]:** Tài khoản đã có token trong Kho bảo mật (Vault), sẵn sàng chuyển đổi ngầm trong 1 giây.
   - 🟡 **Vàng [🔑 Kích hoạt]:** Tài khoản có trên Chrome nhưng chưa cấp quyền cho Antigravity.
3. **Bảo mật chuẩn Windows DPAPI:**
   - Mọi token trong Kho (Vault) đều được mã hóa bằng Windows DPAPI (`CryptProtectData`), chỉ có tài khoản người dùng Windows hiện tại mới giải mã được.
4. **Không làm mất lịch sử làm việc:**
   - Phiên làm việc, thread chat và ngữ cảnh công việc của Antigravity được bảo tồn nguyên vẹn khi chuyển đổi tài khoản.
5. **Giao diện tối giản hiện đại (Dark Mode):**
   - Mini Window gọn gàng, hỗ trợ tính năng "Always on Top" để dễ dàng thao tác song song với IDE.

---

## 🚀 Hướng Dẫn Sử Dụng

### Yêu Cầu
- Windows 10 / 11
- Python 3.10+ (Đã tích hợp sẵn standard library, không cần cài đặt thêm thư viện ngoài)

### Khởi Chạy Ứng Dụng
```bash
python main.py
```

### Cách Kích Hoạt Tài Khoản Thứ Hai (Chỉ làm 1 lần duy nhất)
1. Bật Tool lên (`python main.py`). Bạn sẽ thấy tài khoản đầu tiên có chấm 🟢 **Xanh lá [ĐANG DÙNG]**.
2. Tài khoản thứ hai trong Chrome sẽ có chấm 🟡 **Vàng [🔑 Kích hoạt]**.
3. Bạn đăng nhập tài khoản thứ hai trên Antigravity (bằng Chrome một lần).
4. Tool sẽ tự động bắt Token của tài khoản thứ hai lưu vào Kho bảo mật và chuyển trạng thái của nó thành 🟢 **Xanh lá [⚡ Chuyển sang]**.
5. Kể từ lúc này, cả hai tài khoản đều có chấm 🟢 **Xanh lá**. Bạn chỉ việc click **[⚡ Chuyển sang]** bất cứ khi nào hết limit mà không cần mở lại trình duyệt!

---

## 🧪 Kiểm Thử (Unit Tests)

Chạy bộ kiểm thử tự động toàn diện:
```bash
python -m unittest tests/test_all.py
```
