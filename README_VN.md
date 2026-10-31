# ⚡ AS Auto Switcher - **Version 1.0.3.3**

[🇻🇳 Tiếng Việt](README_VN.md) | [🇬🇧 English](README_EN.md)

Nếu bạn có 1 tài khoản cá nhân, 1 tài khoản công ty, 1 tài khoản để làm dự án bên ngoài; mỗi khi chuyển tài khoản để làm việc cứ phải sign in/out mất thời gian thì công cụ này chính là dành cho bạn.

### **Khi tài khoản được chuyển tự động, tất cả công việc đang làm vẫn tiếp tục bình thường, tuyệt đối không bị gián đoạn, không cần can thiệp thủ công.**

### **An toàn vì tuân thủ theo cơ chế chuyển đổi tài khoản được Google cho phép. Không can thiệp mã nguồn Antigravity, không dùng cơ chế bọc API**

Ứng dụng Windows viết bằng **Go**, giúp quản lý nhiều tài khoản Google, theo dõi quota và chuyển đổi tài khoản đang dùng trong Antigravity 2.x một cách linh hoạt, an toàn. Ứng dụng cung cấp giao diện desktop **Fyne** đồ họa trực quan, hiện đại, mang lại trải nghiệm Zero-Console mượt mà với duy nhất một file thực thi `AS_Auto_Switcher.exe`.

Ứng dụng tích hợp toàn diện hệ thống **Chuyển Đổi Tài Khoản Tự Động (Autonomous Multi-Account Switcher & Failover Daemon)** với đường ống 3 giai đoạn nâng cao (v1.0.3.2) tích hợp bộ lọc chống nhảy tài khoản lặp (Anti-Thrashing Guard). Người dùng có thể linh hoạt chọn giữa chuyển tài khoản thủ công (bằng nút **Chuyển** trên GUI) và chuyển đổi tự động hoàn toàn (AutoSwitch). Khi bật chế độ Tự động, hệ thống tự động giám sát quota, phát hiện cạn kiệt tài nguyên, tự phục hồi thử lại trong bộ nhớ, sàng lọc ứng viên sạch và tự động chọn tài khoản dự phòng tối ưu nhất theo thuật toán **Burn-down Sorter** mà không làm gián đoạn dòng suy nghĩ công việc của lập trình viên.

## ✨ Tính năng nổi bật (Trải nghiệm người dùng)

- **Khắc phục trùng lặp nút Thoát khay hệ thống & Chuẩn hóa đóng ứng dụng an toàn (v1.0.3.2):**
  - Chuẩn hóa menu ngữ cảnh System Tray với cờ `exitItem.IsQuit = true`, loại bỏ triệt để nút "Quit" thừa do cơ chế ngầm của Fyne framework.
  - Bảo toàn chuẩn hóa đa ngữ i18n (`i18n.Current().TrayExit`), hiển thị chính xác "Thoát" (Tiếng Việt) và "Exit" (Tiếng Anh).
  - Đồng bộ 100% quy trình dọn dẹp an toàn (Graceful Shutdown Parity: `a.Close()` -> `a.app.Quit()`), hủy context, dừng ticker telemetry và lưu cấu hình sạch sẽ trước khi thoát.
- **Đồng bộ Quota tức thì khi khởi động (Initial Sync tại t = 0):**
  - Xóa bỏ hoàn toàn độ trễ 60 giây khi mở ứng dụng, tự động kéo dữ liệu quota mới nhất từ Google ngay trong giây đầu tiên.
  - Hoạt động tức thì bất kể công tắc AutoSwitch đang BẬT hay TẮT, cập nhật ngay số liệu quota lên toàn bộ thẻ tài khoản.
  - Tự động kích hoạt chuyển đổi tài khoản ngay khi mở app nếu phát hiện tài khoản đang dùng đã cạn kiệt, đảm bảo chuyển đổi dự phòng tức thì mà không cần người dùng thao tác thủ công.
- **Thuật toán AutoSwitch 3 Giai đoạn Thông minh (v1.0.3.2):**
  - **Giai đoạn 1 (Phát hiện):** Giám sát liên tục phát hiện cạn kiệt quota thời gian thực qua mã lỗi 429 và đánh giá telemetry chu kỳ nền chủ động.
  - **Giai đoạn 2 (Lọc ứng viên & Chống nhảy lặp Thrashing):** Áp dụng ngưỡng hạn mức tuần an toàn nghiêm ngặt **> 2%** (loại bỏ hoàn toàn các tài khoản <= 2%) cùng điều kiện "VÀ" bắt buộc (**5h > 2% VÀ Tuần > 2%**), triệt tiêu triệt để hiện tượng nhảy tài khoản liên tục (Thrashing / Flapping Loop).
  - **Giai đoạn 3 (Sắp xếp Burn-down & Preflight an toàn):** Luân chuyển đến tài khoản dự phòng tối ưu nhất theo nguyên tắc Burn-down (ưu tiên tuần thấp nhất trước `w1 < w2`), thực hiện kiểm tra preflight thăm dò quota thực tế trước khi chốt chuyển đổi.
- **Giao diện lưới thẻ tài khoản trực quan:** Danh sách tài khoản hiển thị dạng lưới tiện quan sát với các trạng thái rõ ràng:
  - `[✔ Dùng]`: Tài khoản đang hoạt động trong Antigravity IDE.
  - `[Chuyển]`: Đổi sang tài khoản này nhanh chóng chỉ với 1 thao tác bấm.
  - `[Cần OAuth]`: Nhắc nhở tài khoản cần đăng nhập trong IDE để cấp quyền truy cập.
- **Nút xóa an toàn `[ 🗑 ]`:** Hỗ trợ dọn dẹp các tài khoản không còn dùng (Inactive). Tích hợp hộp thoại xác nhận Danger để bảo vệ tuyệt đối, tránh xóa nhầm tài khoản đang hoạt động.
- **Công tắc AutoSwitch tự động:** Gạt bật/tắt chế độ tự động chuyển đổi tài khoản ngay khi cạn kiệt quota, đảm bảo công việc code diễn ra liên tục, không bị đứt mạch suy nghĩ.
- **Thanh trạng thái thời gian thực:** Cập nhật sống động trạng thái hoạt động của hệ thống (`IDLE`, `EVALUATING`, `SWITCHING`, `SWITCHED`...).
- **Dashboard theo dõi Quota chi tiết:** Theo dõi trực quan phần trăm quota còn lại của Gemini và Claude theo cả chu kỳ 5h và chu kỳ Tuần; thanh tiến trình tự đổi màu theo mức độ khả dụng (xanh: dồi dào, vàng: trung bình, đỏ: sắp cạn).
- **Làm mới danh sách & quota tức thì:** Nút làm mới toàn bộ danh sách hoặc làm mới riêng biệt từng tài khoản chỉ trong tích tắc.
- **3 chủ đề giao diện hiện đại:** Tùy chọn chuyển đổi linh hoạt giữa giao diện **Dark**, **Light**, và **Warm** phù hợp với mắt nhìn.
- **Chế độ ghim Always-on-Top:** Giữ cửa sổ ứng dụng luôn nổi trên màn hình để tiện theo dõi hạn mức quota trong lúc làm việc.
- **Thu nhỏ khay hệ thống (System Tray):** Thu gọn ứng dụng xuống Taskbar/Tray góc màn hình khi không cần mở lớn, với menu ngữ cảnh tinh gọn, không trùng lặp nút thoát và hỗ trợ đóng an toàn.

## 🚀 Cài đặt & Khởi chạy

### 1. Chạy trực tiếp (Portable - Không cần cài đặt)

Ứng dụng được đóng gói hoàn chỉnh dưới dạng **duy nhất một file thực thi `AS_Auto_Switcher.exe`**, mang lại trải nghiệm Zero-Console mượt mà (không bật cửa sổ dòng lệnh đen):
- Tải file `AS_Auto_Switcher.exe` (hoặc lấy từ thư mục `bin\AS_Auto_Switcher.exe`).
- Click đúp vào file để mở giao diện và sử dụng ngay lập tức.

### 2. Hướng dẫn sử dụng nhanh

1. **Lưu tài khoản:** Đăng nhập tài khoản Google của bạn trong **Antigravity IDE**, sau đó mở Switcher và bấm **Danh sách tài khoản** để tự động lưu phiên. Lặp lại với các tài khoản khác khi cần lưu thêm.
2. **Chuyển đổi thủ công:** Bấm nút **Chuyển** trên thẻ tài khoản đã lưu để hoán đổi tài khoản đang làm việc.
3. **Chuyển đổi tự động:** Gạt bật công tắc **Tự động** (AutoSwitch). Ứng dụng sẽ tự động phát hiện khi tài khoản hiện tại hết hạn mức và tự động chuyển sang tài khoản kế tiếp tốt nhất mà không cần bạn can thiệp.

### 3. Biên dịch từ mã nguồn

Dành cho lập trình viên muốn tự build ứng dụng từ source code:

**Yêu cầu hệ thống:**
- **Go 1.22+** (khuyến nghị phiên bản mới nhất).
- **GCC (MinGW-w64 / MSYS2 UCRT64)** đã cài đặt và cấu hình trong biến môi trường `PATH` để hỗ trợ CGO.

**Lệnh biên dịch:**
Mở PowerShell tại thư mục gốc của dự án và chạy script đóng gói:
```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1
```

Sau khi hoàn tất, file nhị phân sẽ được tạo tại `bin\AS_Auto_Switcher.exe`.

## 🔒 An toàn & Bảo mật

- **100% Offline trên máy cục bộ:** Không sử dụng bất kỳ máy chủ bên thứ 3 nào. Tất cả thông tin tài khoản được lưu trữ an toàn trên chính máy tính của bạn.
- **Tuân thủ chuẩn mực:** Cơ chế chuyển đổi hoàn toàn tương thích với Antigravity IDE, không chỉnh sửa mã nguồn gốc của IDE và không can thiệp bọc API.
