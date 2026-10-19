# Đặc Tả Kiến Trúc & Kỹ Thuật: Thu Gọn Độ Rộng Cửa Sổ Ứng Dụng Xuống 960px (Window 960x630px)

- **Ngày lập:** 2026-10-19
- **Vai trò:** System Architect
- **Thẩm quyền:** Chỉ đạo trực tiếp từ Human Owner: *"gaim độ rộng UI tông xuong con 960px"*
- **Mã văn kiện:** `SPEC-ARCH-20261019-COMPACT-WINDOW-WIDTH-960PX`
- **Trạng thái:** CHÍNH THỨC ĐÃ ĐƯỢC HUMAN OWNER PHÊ DUYỆT (OFFICIAL APPROVED)

---

## ⚠️ ĐIỀU KHOẢN KỶ LUẬT SẮT (MANDATORY IRON RULES)

> **CHỈ THỊ TRỰC TIẾP TỪ HUMAN OWNER:**
> 1. **KHÓA CỨNG KÍCH THƯỚC CỬA SỔ MẶC ĐỊNH MỚI: `960 x 630 px`:**
>    - Chiều rộng cửa sổ chính giảm từ **`1008 px`** xuống cố định **`960 px`** (`DefaultWindowWidth = 960.0`).
>    - Chiều cao cửa sổ chính giữ nguyên cố định **`630 px`** (`DefaultWindowHeight = 630.0`).
> 2. **BẢO TỒN KÍCH THƯỚC THẺ CARD & BẢNG QUOTA:**
>    - Kích thước thẻ Card trong `GridWrap` giữ nguyên cố định **`280 x 130 px`** (`container.NewGridWrap(fyne.NewSize(280, 130))`).
>    - Bảng Quota 2 tầng bên trong thẻ giữ nguyên kích thước toán học **`228 x 77 px`**.
> 3. **BẢO TỒN NGUYÊN VẸN CÁC TIÊU CHUẨN ĐÃ PHÊ DUYỆT:**
>    - Bố cục danh sách luôn hiển thị **3 cột thẻ** thẳng tắp, không phát sinh thanh cuộn ngang.
>    - Typography: Text chính 10pt Bold, text thời gian đếm lùi 8.5pt `TextMuted`.
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
│ (3 Cột Thẻ 280x130px Thẳng Tắp - Bề Rộng Cửa Sổ 960px - Hoàn Hảo Cho Màn Hình Nhỏ)      │
└────────────────────────────────────────────────────────────────────────────────────────┘
```

### 1.1. Phân Bổ Bề Rộng Cửa Sổ (Total Width = 960 px):
- **Bề rộng cửa sổ:** `960.0 px`.
- **Thanh cuộn dọc & padding viền cửa sổ:** `~40.0 px`.
- **Bề rộng nội dung khả dụng cho GridWrap:** `960 - 40 = 920.0 px`.
- **3 Cột Thẻ Card 280px:** $3 \times 280.0\text{ px} = 840.0\text{ px}$.
- **Khoảng cách tự nhiên giữa các cột:** $(920 - 840) / 4 = 20.0\text{ px}$.
- **Độ ổn định của GridWrap:**
  - Vì $840\text{ px} \le 920\text{ px} < 1120\text{ px}$ ($4 \times 280\text{ px}$), cơ chế bố cục `GridWrap` của Fyne sẽ **luôn luôn hiển thị chính xác 3 cột thẻ**.
  - Tuyệt đối không bị rớt xuống 2 cột và không bao giờ bị tràn sang 4 cột.
  - Hoàn toàn không phát sinh thanh cuộn ngang (Horizontal scrollbar).

### 1.2. Tính Tương Thích Màn Hình:
- Kích thước `960 x 630 px` cực kỳ thân thiện với:
  - Màn hình laptop 13–14 inch có Windows DPI Scaling 125% - 150%.
  - Màn hình phụ độ phân giải HD (1366x768).
  - Không gian desktop khi chia đôi màn hình (Split screen / Snap layout).

---

## 2. TOUCH-MAP KHÓA CỨNG DÀNH CHO CODER (EXACT FILE TOUCHES)

### 2.1. File `internal/ui/window.go`
- **Sửa hằng số kích thước mặc định:**
  ```go
  const (
      DefaultWindowWidth  float32 = 960.0  // Thu gọn từ 1008.0 xuống 960.0
      DefaultWindowHeight float32 = 630.0  // Giữ nguyên 630.0
      MinWindowWidth      float32 = 480.0
      MinWindowHeight     float32 = 360.0
  )
  ```

### 2.2. File `internal/ui/ui_test.go`
- **Cập nhật các test assertion kiểm tra kích thước cửa sổ từ `1008` thành `960`:**
  - `TestIndustryStandard_WindowInit1008x630`: Đổi tên thành `TestIndustryStandard_WindowInit960x630` và assert `sz.Width == 960 && sz.Height == 630`.
  - `TestSliceP6A_OwnerRedesignInvariants`: Assert `sz.Width == 960 && sz.Height == 630`.
  - `TestArchitectSpec_...`: Assert `DefaultWindowWidth == 960.0`.
  - `w.Resize(fyne.NewSize(960, 630))`.
- Chạy `go test -v ./internal/ui` đạt 100% PASS.

---

## 3. TIÊU CHÍ NGHIỆM THU (ACCEPTANCE CRITERIA)

- [ ] **AC-1 (Default Window Width 960px):** Hằng số `DefaultWindowWidth` bằng đúng `960.0`, kích thước cửa sổ khởi tạo là `960 x 630 px`.
- [ ] **AC-2 (3-Column Card Layout):** Vùng danh sách tài khoản hiển thị chính xác 3 cột thẻ tài khoản (kích thước `280 x 130 px`) thẳng hàng, không có thanh cuộn ngang.
- [ ] **AC-3 (In-place Reactive Update):** Cập nhật Quota tự động hoặc thủ công hoạt động mượt mà, không giật màn hình và không làm thay đổi kích thước cửa sổ.
- [ ] **AC-4 (Test Suite):** Toàn bộ test suite `go test -v ./internal/ui` đạt 100% PASS.
