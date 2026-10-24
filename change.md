# 📋 Lịch Sử Thay Đổi (Changelog) - Antigravity Multi-Account Switcher

Tài liệu ghi nhận toàn bộ các thay đổi, tính năng mới, cải tiến hiệu năng và bản sửa lỗi qua từng phiên bản phát hành của **Antigravity Multi-Account Switcher**.

---

## [Version 1.0.3.0] - 2026-10-24

### ✨ Tính Năng Mới & Nâng Cấp UX
- **Tự động nhận diện Tên Gói (Tier Badge):** Thay thế chữ `"Quota"` tĩnh ở ô góc trên bên trái của bảng ma trận Quota bằng Tên gói thực tế (`PRO`, `ULTRA`, `FREE`, `BIZ`, `--`) được phát hiện tự động 100%.
- **Phân cấp Thị giác (Visual Hierarchy):** Định danh màu sắc trực quan cho từng gói:
  - `ULTRA`: Màu Vàng Amber (`#f59e0b`) nổi bật rực rỡ cho tài khoản Google AI Ultra cao cấp nhất.
  - `PRO`: Màu Xanh Primary (`#38bdf8`) cho gói Google One AI Pro / Premium làm việc chính.
  - `FREE`: Màu Xám mờ (`#888899`) nhẹ nhàng, tinh tế cho tài khoản cá nhân thông thường.
  - `BIZ`: Màu Xanh lục (`#10b981`) cho tài khoản Google Workspace / Doanh nghiệp.
  - `--`: Màu Muted cho tài khoản chưa có token hoặc đang nạp dữ liệu.

### 🛠️ Cải Tiến Kỹ Thuật & Kiến Trúc
- **Chữa lành Schema Drift (`GetUserStatus`):** Khắc phục triệt để lỗi ép kiểu do Antigravity 2.x nâng cấp `planStatus` thành đối tượng JSON map lồng nhau (`planInfo.planName`, `planInfo.teamsTier`). Hỗ trợ song song cả chuỗi phẳng cũ lẫn map mới.
- **Mở rộng Telemetry 2 Nguồn (Dual-Source):**
  - Tích hợp endpoint Google Cloud Code `/v1internal:loadCodeAssist` để xác định chính xác Tier cho toàn bộ danh bạ tài khoản (kể cả Inactive).
  - Kết nối local RPC `/GetUserStatus` để lấy thông tin tức thì khi tài khoản đang Active trong IDE.
- **Cập nhật In-place (Zero Flicker):** Bổ sung method `UpdateTier` phản ứng tức thì trên Canvas cây đối tượng của Fyne, giữ nguyên 100% độ rộng cột `col1W = 48px`, không gây giật lag giao diện.
- **Khởi tạo tài liệu theo dõi phiên bản:** Bổ sung file `change.md` vào gốc kho mã nguồn.

---

## [Version 1.0.2.0] - 2026-10-23

### ✨ Tính Năng Mới & Tài Liệu
- **Hỗ trợ Đa ngôn ngữ (Multilingual Documentation):** Bổ sung tài liệu song ngữ toàn diện gồm `README_VN.md` (Tiếng Việt) và `README_EN.md` (Tiếng Anh) cùng điều hướng chuyển ngữ linh hoạt.
- **Cơ chế Đồng bộ Phiên bản Tự Động (SSOT Version Auto-Sync):**
  - Thiết lập `internal/version/version.go` làm Single Source of Truth (SSOT).
  - Tự động đồng bộ hóa phiên bản sang `cmd/switcher/app.rc` (Win32 resource metadata) và tiêu đề của toàn bộ các file `README.md`.
  - Hỗ trợ switch `-SyncOnly` trong `scripts/build_windows.ps1` giúp đồng bộ tức thì dưới 200ms.
- **Contract Test Suite:** Bổ sung kiểm thử `TestVersion_ReadmeSync` phát hiện và ngăn chặn độ trôi dạt phiên bản (version drift).

---

## [Version 1.0.1.0] - 2026-10-21

### ⚡ Tối Ưu Hóa Hiệu Năng (Startup & Memory Optimization)
- **Tăng tốc khởi động (Cold Startup Boost):** Rút ngắn thời gian khởi động ứng dụng xuống dưới 500ms bằng cách tối ưu hóa chu trình nạp cấu hình và kho tài khoản.
- **Giảm chiếm dụng bộ nhớ RAM:** Cắt giảm hơn 40% RAM lúc chạy nền thông qua cơ chế giải phóng tài nguyên đồ họa không sử dụng.
- **Win32 Discovery & Singleflight:** Sử dụng Windows API cấp thấp để phát hiện tiến trình Antigravity Language Server nhanh chóng và tin cậy; loại bỏ tình trạng gọi trùng lặp telemetry.
- **Double-check Active Protection:** Cơ chế kiểm tra an toàn 2 lớp trước khi thao tác xóa hoặc chuyển tài khoản, chống xóa nhầm tài khoản đang hoạt động.

---

## [Version 1.0.0.0] - 2026-10-18

### 🚀 Phát Hành Đầu Tiên (Initial Release)
- **Kiến trúc Zero-Console độc lập:** Đóng gói toàn bộ ứng dụng trong một file thực thi duy nhất `bin/antigravity-switch.exe`, không bật popup terminal đen khi chạy.
- **Giao diện Desktop hiện đại (Fyne GUI):** Quản lý danh sách tài khoản theo dạng thẻ trực quan với đầy đủ trạng thái `[✔ Dùng]`, `[Chuyển]`, `[Cần OAuth]`.
- **Hệ thống Giám sát Hạn mức Kép (Dual-Cycle Quota Matrix):**
  - Theo dõi đồng thời cả chu kỳ 5 giờ (5h) và chu kỳ Tuần (Weekly) cho Gemini và Claude.
  - Thanh tiến trình Mini ProgressBar tự động đổi màu theo mức độ khả dụng (Xanh > 50%, Vàng 15-50%, Đỏ < 15%).
- **Tiến trình Chuyển đổi Tự động (Autonomous AutoSwitch Daemon):**
  - Tự động phát hiện khi tài khoản cạn kiệt hạn mức.
  - Áp dụng thuật toán sắp xếp thông minh **Burn-down Sorter** để chọn tài khoản dự phòng tối ưu nhất.
  - Cơ chế Re-try Backoff lũy thừa chống nghẽn và duy trì dòng chảy lập trình liên tục.
- **Tiện ích hệ thống:** Hỗ trợ Ghim trên cùng (Always-on-Top), Thu nhỏ xuống khay hệ thống (System Tray), và 3 bộ giao diện Đậm/Sáng/Ấm (Dark/Light/Warm Themes).
