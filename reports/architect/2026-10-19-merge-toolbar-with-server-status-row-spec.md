# Đặc Tả Kiến Trúc & Kỹ Thuật: Gộp 3 Nút Toolbar Lên Cùng Hàng Với Thông Tin Server Status (Header 2 Hàng Tinh Gọn)

- **Ngày lập:** 2026-10-19
- **Vai trò:** System Architect
- **Thẩm quyền:** Chỉ đạo trực tiếp từ Human Owner qua hình ảnh minh chứng `reports/problem/images/screenshot_1792417807.png`: *"đưa 3 nút lên cùng hàng với Port 62220 - veit dah ta gui CEO"*
- **Mã văn kiện:** `SPEC-ARCH-20261019-MERGE-TOOLBAR-WITH-SERVER-STATUS-ROW`
- **Trạng thái:** CHÍNH THỨC ĐÃ ĐƯỢC HUMAN OWNER PHÊ DUYỆT (OFFICIAL APPROVED)

---

## ⚠️ ĐIỀU KHOẢN KỶ LUẬT SẮT (MANDATORY IRON RULES)

> **CHỈ THỊ TRỰC TIẾP TỪ HUMAN OWNER:**
> 1. **GỘP HÀNG 2 VÀ HÀNG 3 THÀNH 1 HÀNG DUY NHẤT:**
>    - Đưa 3 nút Toolbar (`[ Danh sách tài khoản ]`, `[ Làm mới Quota ]`, `[ 📌 ]`) lên nằm cùng hàng ngang với nhãn trạng thái Server (`Hoạt động | PID ... | Port ...`).
>    - Xóa bỏ hoàn toàn Hàng 3 (`row3Toolbar`) riêng biệt trong `topSection`.
>    - `topSection` chỉ còn lại: Hàng 1 (Header/Title bar), Hàng 2 (Status + 3 Nút Toolbar + Nút đổi Theme), và Đường kẻ ngang phân cách (`sep`).
> 2. **BẢO TỒN NGUYÊN VẸN CÁC THÀNH PHẦN & CHỨC NĂNG:**
>    - Giữ nguyên 100% logic sự kiện và chức năng của `a.refreshBtn`, `a.quotaRefreshAllBtn`, `a.alwaysOnTopBtn`, `a.serverStatusLabel`, `a.themeSelect`.
>    - Duy trì cơ chế `compactButton` cho 3 nút trên hàng này để chiều cao nút đồng bộ, thanh mảnh.
> 3. **BẢO TỒN NGUYÊN VẸN HÌNH HỌC ĐÃ KHÓA CỨNG:**
>    - Cửa sổ ứng dụng: CỐ ĐỊNH **`920 x 630 px`** (`DefaultWindowWidth = 920.0`, `DefaultWindowHeight = 630.0`).
>    - Thẻ Card: CỐ ĐỊNH **`280 x 130 px`** trong `container.NewGridWrap(fyne.NewSize(280, 130))`.
>    - Bảng Quota: CỐ ĐỊNH **`228 x 77 px`**.
>    - In-place reactive update (Zero Jank / Zero Resizing) 100%.

---

## 1. PHÂN TÍCH THỊ GIÁC & HÌNH HỌC (VISUAL SPECIFICATION)

### 1.1. Bản Vẽ So Sánh Bố Cục Header

#### TRƯỚC (3 Hàng riêng biệt - Lãng phí chiều cao, nhiều khoảng trống chết):
```text
┌────────────────────────────────────────────────────────────────────────────────────────┐
│ [AS] Antigravity Multi-Account Switcher                                        _ |_| X │ ◄── Hàng 1
│ Hoạt động | PID 18660 | Port 62220                                            [ ☀️ ▼ ] │ ◄── Hàng 2 (Thừa 650px trống)
│ [ Danh sách tài khoản ]  [ Làm mới Quota ]  [ 📌 ]                                      │ ◄── Hàng 3 (Thừa 600px trống)
├────────────────────────────────────────────────────────────────────────────────────────┤
```

#### SAU (2 Hàng tinh gọn - Tiết kiệm ~35px chiều cao, giao diện liền mạch, hiện đại):
```text
┌────────────────────────────────────────────────────────────────────────────────────────┐
│ [AS] Antigravity Multi-Account Switcher (10pt Bold)                            _ |_| X │ ◄── Hàng 1
│ Hoạt động | PID 18660 | Port 62220   [ Danh sách tài khoản ]  [ Làm mới Quota ]  [ 📌 ]       [ ☀️ ▼ ] │ ◄── Hàng 2 (GỘP MỚI)
├────────────────────────────────────────────────────────────────────────────────────────┤
│ ┌──────────────────────────┐ ┌──────────────────────────┐ ┌──────────────────────────┐ │
│ │ user1@gmail.com [✔ Dùng] │ │ user2@gmail.com [Chuyển] │ │ user3@gmail.com [Chuyển] │ │ ◄── Đẩy khu vực thẻ
│ │ Quota    5h     Week     │ │ Quota    5h     Week     │ │ Quota    5h     Week     │ │     lên cao, hiển thị
│ │ Gemini [████]  [████]    │ │ Gemini [████]  [██░░]    │ │ Gemini [████]  [██░░]    │ │     trọn vẹn 2 hàng
│ │ Claude [████]  [██░░]    │ │ Claude [████]  [██░░]    │ │ Claude [████]  [██░░]    │ │     thẻ (6 tài khoản)!
│ └──────────────────────────┘ └──────────────────────────┘ └──────────────────────────┘ │
```

### 1.2. Phân Bổ Bề Ngang Hàng 2 Gộp Mới (Window Width = 920 px):
- **Bên trái (`leftCluster`):**
  - Nhãn `serverStatusLabel`: $\sim 200.0\text{ px}$.
  - Khoảng cách đệm: $12.0\text{ px}$.
  - 3 Nút Toolbar (`refreshBtn`, `quotaRefreshAllBtn`, `alwaysOnTopBtn`): $\sim 300.0\text{ px}$.
  - Tổng bề rộng cụm trái: $\sim 512.0\text{ px}$.
- **Bên phải (`themeSelectWrap`):**
  - Kích thước: `58.0 x 32.0 px`.
- **Khoảng trống tự nhiên ở giữa:**
  - $920.0 - 512.0 - 58.0 = \mathbf{350.0\text{ px}}$ đệm tự do, đảm bảo 100% không bao giờ xảy ra hiện tượng chạm mép, chồng đè hay xuống dòng.

---

## 2. TOUCH-MAP KHÓA CỨNG DÀNH CHO CODER (EXACT FILE TOUCHES)

### 2.1. File `internal/ui/window.go`

Trong hàm `setupLayout()`:
1. **Gộp `serverStatusLabel` và 3 nút Toolbar thành một khối bên trái:**
   ```go
   toolbarLeft := container.NewHBox(
       compactButton(a.refreshBtn),
       compactButton(a.quotaRefreshAllBtn),
       compactButton(a.alwaysOnTopBtn),
   )

   row2LeftCluster := container.NewHBox(
       a.serverStatusLabel,
       toolbarLeft,
   )
   ```
2. **Khởi tạo Hàng 2 gộp mới (`row2Combined`):**
   ```go
   row2Combined := container.NewBorder(nil, nil, row2LeftCluster, themeSelectWrap)
   ```
3. **Cập nhật `topSection` chỉ còn 2 hàng:**
   ```go
   topSection := container.NewVBox(
       row1Header,
       row2Combined,
       sep,
   )
   ```
4. Xóa bỏ biến `row2SubHeader` cũ và biến `row3Toolbar` cũ.

### 2.2. File `internal/ui/ui_test.go`
- Đảm bảo các test case kiểm thử các nút `a.refreshBtn`, `a.quotaRefreshAllBtn`, `a.alwaysOnTopBtn`, `a.serverStatusLabel`, `a.themeSelect` vẫn truy cập và hoạt động 100% chính xác.
- Chạy `go test -v ./internal/ui` đạt 100% PASS.

---

## 3. TIÊU CHÍ NGHIỆM THU (ACCEPTANCE CRITERIA)

- [ ] **AC-1 (Single Combined Row 2):** 3 nút Toolbar (`[ Danh sách tài khoản ]`, `[ Làm mới Quota ]`, `[ 📌 ]`) nằm ngay sau thông tin Server Status (`Port ...`) trên cùng một hàng ngang.
- [ ] **AC-2 (No Row 3):** Hàng 3 cũ bị xóa bỏ hoàn toàn, `topSection` chỉ gồm Hàng 1, Hàng 2 gộp và vách ngăn `sep`.
- [ ] **AC-3 (Vertical Space Gain):** Khu vực danh sách tài khoản được nâng lên thêm ~35px, 2 hàng thẻ đầu tiên (6 tài khoản) hiển thị trọn vẹn và thoáng đãng.
- [ ] **AC-4 (Dimensions Stability):** Cửa sổ giữ nguyên `920 x 630 px`, thẻ Card giữ nguyên `280 x 130 px`.
- [ ] **AC-5 (Test Suite):** Chạy `go test -v ./internal/ui` đạt 100% PASS.
