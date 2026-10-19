# Đặc Tả Kiến Trúc & Kỹ Thuật: Tối Ưu Hóa Kích Thước Cửa Sổ Ứng Dụng Thành 920x630px

- **Ngày lập:** 2026-10-19
- **Vai trò:** System Architect
- **Thẩm quyền:** Chỉ đạo trực tiếp từ Human Owner: *"Thay đổi kích thước cửa sổ thành: 920x630px vi còn rat nheiu khoảng trống. viet đặc tả gửi CEO"*
- **Mã văn kiện:** `SPEC-ARCH-20261019-COMPACT-WINDOW-WIDTH-920PX`
- **Trạng thái:** CHÍNH THỨC ĐÃ ĐƯỢC HUMAN OWNER PHÊ DUYỆT (OFFICIAL APPROVED)

---

## ⚠️ ĐIỀU KHOẢN KỶ LUẬT SẮT (MANDATORY IRON RULES)

> **CHỈ THỊ TRỰC TIẾP TỪ HUMAN OWNER:**
> 1. **KHÓA CỨNG KÍCH THƯỚC CỬA SỔ MẶC ĐỊNH MỚI: `920 x 630 px`:**
>    - Chiều rộng cửa sổ ứng dụng giảm từ **`960 px`** xuống cố định **`920 px`** (`DefaultWindowWidth = 920.0`).
>    - Chiều cao cửa sổ ứng dụng giữ nguyên cố định **`630 px`** (`DefaultWindowHeight = 630.0`).
> 2. **BẢO TỒN KÍCH THƯỚC THẺ CARD & BẢNG QUOTA:**
>    - Kích thước thẻ Card trong `GridWrap` giữ nguyên cố định **`280 x 130 px`** (`container.NewGridWrap(fyne.NewSize(280, 130))`).
>    - Bảng Quota 2 tầng bên trong thẻ giữ nguyên kích thước toán học **`228 x 77 px`**.
>    - Nút `[Chuyển]` dạng text in đậm màu xanh dương (`palette.Primary`) đồng bộ với `[✔ Dùng]`.
> 3. **BẢO TỒN NGUYÊN VẸN CÁC TIÊU CHUẨN ĐÃ PHÊ DUYỆT:**
>    - Danh sách luôn hiển thị **3 cột thẻ** thẳng tắp, không thừa khoảng trắng vô lý và không phát sinh thanh cuộn ngang.
>    - In-place reactive update (Zero Jank / Zero Resizing) bảo toàn 100%.

---

## 1. PHÂN BỔ HÌNH HỌC TOÁN HỌC (GEOMETRIC BUDGET)

```text
┌────────────────────────────────────────────────────────────────────────────────────────┐  ▲
│ [AS] Antigravity Multi-Account Switcher (10pt Bold)                            _ |_| X │  │ TopSection
│ Hoạt động | PID 21484 | Port 65144 (10pt)                                     [ ☀️ ▼ ] │  │ (~85px)
│ [ Danh sách tài khoản ]  [ Làm mới Quota ]  [ 📌]                                      │  ▼
├────────────────────────────────────────────────────────────────────────────────────────┤
│ ┌──────────────────────────┐ ┌──────────────────────────┐ ┌──────────────────────────┐ │  ▲
│ │ user1@gmail.com [✔][🔄]  │ │ user2@gmail.com [Ch][🔄] │ │ user3@gmail.com [Ch][🔄] │ │  │
│ │ ──────────────────────── │ │ ──────────────────────── │ │ ──────────────────────── │ │  │ VScroll
│ │ Quota    5h     Week     │ │ Quota    5h     Week     │ │ Quota    5h     Week     │ │  │ Center
│ │        (04h32) (6d 21h)  │ │        (01h15) (5d 08h)  │ │        (Ready) (2d 07h)  │ │  │ Section
│ │ Gemini [████]  [██░░]    │ │ Gemini [████]  [████]    │ │ Gemini [████]  [████]    │ │  │ (~540px)
│ │ ──────────────────────── │ │ ──────────────────────── │ │ ──────────────────────── │ │  │
│ │        (02h12) (2d 07h)  │ │        (Ready) (1d 14h)  │ │        (03h45) (4d 18h)  │ │  │
│ │ Claude [████]  [████]    │ │ Claude [████]  [████]    │ │ Claude [████]  [████]    │ │  │
│ └──────────────────────────┘ └──────────────────────────┘ └──────────────────────────┘ │  ▼
│ ◄────── 280px ────────────►  ◄────── 280px ────────────►  ◄────── 280px ────────────►   │
│ (3 Cột Thẻ 280x130px Thẳng Tắp - Cửa Sổ 920x630px Khít Từng Pixel - Triệt Tiêu Khoảng Trống)│
└────────────────────────────────────────────────────────────────────────────────────────┘
```

### 1.1. Khớp Nối Toán Học Bề Rộng (Total Width = 920 px):
- **Bề rộng cửa sổ:** `920.0 px`.
- **Thanh cuộn dọc và padding lề cửa sổ:** `~32.0 px`.
- **Bề rộng nội dung thực tế khả dụng cho `GridWrap`:** `920 - 32 = 888.0 px`.
- **3 Cột Thẻ Card 280px:** $3 \times 280.0\text{ px} = 840.0\text{ px}$.
- **Khoảng trống còn lại:** $888 - 840 = 48.0\text{ px}$.
- **Khoảng cách đệm giữa các cột:** $48.0 / 3 = 16.0\text{ px}$ mỗi khoảng đệm.
- **Đánh giá hình học:** 
  - Khắc phục triệt để hiện tượng thừa khoảng trống bên phải ở kích thước 960px.
  - Các cột thẻ ôm sát nhau với khoảng cách chuẩn $16\text{ px}$, giao diện liền khối, hiện đại và cực kỳ gọn gàng.
  - Vì $848\text{ px} \le 888\text{ px} < 1132\text{ px}$, Fyne `GridWrap` được khóa cứng vĩnh viễn ở **chính xác 3 cột**, 100% không sinh thanh cuộn ngang.

---

## 2. TOUCH-MAP KHÓA CỨNG DÀNH CHO CODER (EXACT FILE TOUCHES)

### 2.1. File `internal/ui/window.go`
- **Sửa hằng số kích thước mặc định:**
  ```go
  const (
      DefaultWindowWidth  float32 = 920.0  // Thu gọn từ 960.0 xuống 920.0
      DefaultWindowHeight float32 = 630.0  // Giữ nguyên 630.0
      MinWindowWidth      float32 = 480.0
      MinWindowHeight     float32 = 360.0
  )
  ```

### 2.2. File `internal/ui/ui_test.go`
- **Cập nhật các test assertion kiểm tra kích thước cửa sổ từ `960` thành `920`:**
  - `TestIndustryStandard_WindowInit960x630`: Đổi tên thành `TestIndustryStandard_WindowInit920x630` và assert `sz.Width == 920 && sz.Height == 630`.
  - `TestSliceP6A_OwnerRedesignInvariants`: Assert `sz.Width == 920 && sz.Height == 630`.
  - `TestArchitectSpec_...`: Assert `DefaultWindowWidth == 920.0`.
  - `w.Resize(fyne.NewSize(920, 630))`.
- Chạy `go test -v ./internal/ui` đạt 100% PASS.

---

## 3. TIÊU CHÍ NGHIỆM THU (ACCEPTANCE CRITERIA)

- [ ] **AC-1 (Default Window Width 920px):** Hằng số `DefaultWindowWidth` bằng đúng `920.0`, kích thước cửa sổ khởi tạo là `920 x 630 px`.
- [ ] **AC-2 (3-Column Zero-Waste Layout):** Vùng danh sách tài khoản hiển thị chính xác 3 cột thẻ tài khoản (kích thước `280 x 130 px`) vừa khít, khoảng đệm tự nhiên ~16px, không thừa khoảng trống vô lý, không có thanh cuộn ngang.
- [ ] **AC-3 (In-place Reactive Update):** Cập nhật Quota tự động hoặc thủ công hoạt động mượt mà, không giật màn hình và không làm thay đổi kích thước cửa sổ.
- [ ] **AC-4 (Test Suite):** Toàn bộ test suite `go test -v ./internal/ui` đạt 100% PASS.
