# Đặc Tả Kiến Trúc & Kỹ Thuật: Thu Gọn Độ Rộng Thẻ Tài Khoản Xuống 280px (Card 280x130px)

- **Ngày lập:** 2026-10-19
- **Vai trò:** System Architect
- **Thẩm quyền:** Chỉ đạo trực tiếp từ Human Owner: *"giảm độ rộng thẻ còn 280px. veit dạc tả gui CEO"*
- **Mã văn kiện:** `SPEC-ARCH-20261019-COMPACT-CARD-WIDTH-280PX`
- **Trạng thái:** CHÍNH THỨC ĐÃ ĐƯỢC HUMAN OWNER PHÊ DUYỆT (OFFICIAL APPROVED)

---

## ⚠️ ĐIỀU KHOẢN KỶ LUẬT SẮT (MANDATORY IRON RULES)

> **CHỈ THỊ TRỰC TIẾP TỪ HUMAN OWNER:**
> 1. **KHÓA CỨNG KÍCH THƯỚC THẺ CARD MỚI: `280 x 130 px`:**
>    - Chiều rộng của thẻ Card trong `GridWrap` giảm từ **`300 px`** xuống cố định **`280 px`**.
>    - Chiều cao của thẻ Card giữ nguyên cố định **`130 px`** (`fyne.NewSize(280, 130)`).
> 2. **BẢO TỒN KÍCH THƯỚC CỬA SỔ & BẢNG QUOTA:**
>    - Cửa sổ ứng dụng giữ nguyên **`1008 x 630 px`** (`DefaultWindowWidth = 1008.0`, `DefaultWindowHeight = 630.0`).
>    - Bảng Quota 2 tầng (không vách dọc, có vách ngang `hSep`) giữ nguyên kích thước toán học **`228 x 77 px`**.
> 3. **BẢO TỒN NGUYÊN VẸN CÁC TIÊU CHUẨN ĐÃ PHÊ DUYỆT:**
>    - Typography: Text chính 10pt Bold, text thời gian đếm lùi 8.5pt `TextMuted`.
>    - In-place reactive update (Zero Jank / Zero Resizing) bảo toàn 100%.

---

## 1. PHÂN BỔ HÌNH HỌC TOÁN HỌC (GEOMETRIC BUDGET)

```text
┌──────────────────────────────────────────────────────────────────┐  ▲
│ ngau2hung@gmail.com (10pt)                     [✔ Dùng]  [🔄]   │  │ 26px (Dòng 1: Email + Nút)
│ ──────────────────────────────────────────────────────────────── │  │  8px (Khoảng cách đệm)
│ Quota                  5h                          Week          │  │ 15px (Hàng 0: Header Cột)
│                    (04h 32m)                   (6d 21h 59m)      │  │ 13px (Hàng 1a: Đếm lùi)
│ Gemini           [██████████]  100%            [████░░░░░░]  43% │  │ 15px (Hàng 1b: Bar + %)
│ ──────────────────────────────────────────────────────────────── │  │  4px (Vách ngang hSep)
│                    (02h 12m)                   (2d 07h 08m)      │  │ 13px (Hàng 2a: Đếm lùi)
│ Claude           [██████████]  100%            [██████████] 100% │  │ 15px (Hàng 2b: Bar + %)
└──────────────────────────────────────────────────────────────────┘  ▼ Padding viền 20px -> Tổng 130px
  ◄── 48px ───►  8px  ◄────── 78px ─────────► 8px ◄────── 86px ────►
  (Tổng bảng Quota = 228 px | Nằm lọt thỏm hoàn hảo trong lòng thẻ 280px)
```

### 1.1. Bề Rộng Thẻ Card (Width = 280 px):
- **Bề rộng tổng thể thẻ:** `280.0 px`.
- **Padding viền thẻ (`container.NewPadded`):** `16.0 px` (`8px` mỗi bên).
- **Bề rộng nội dung khả dụng:** `280 - 16 = 264.0 px`.
- **Bảng Quota chiếm:** `228.0 px` (`48px Model + 8px gap + 78px 5h + 8px gap + 86px Week`).
- **Biên đệm lề an toàn:** `264 - 228 = 36.0 px` (mỗi bên `18.0 px`), đảm bảo bảng Quota không bị chạm viền, hiển thị thanh thoát và cân đối.
- **Dòng Email & Cụm Action:** Chiều rộng 264px đủ để hiển thị nút `[✔ Dùng]` / `[Chuyển]` + `[🔄]` (~85px) và nhãn Email (~179px có tính năng anti-break truncation + hover tooltip).

### 1.2. Bố Trí Cửa Sổ 1008 px (3 Cột Thẻ 280px):
- **Không gian khả dụng trong khung cuộn danh sách:** `~968 px`.
- **3 cột thẻ 280px:** $3 \times 280 = 840\text{ px}$.
- **Khoảng cách tự nhiên giữa các cột:** $(968 - 840) / 4 \approx 32\text{ px}$.
- Hiển thị 3 cột thẻ thẳng tắp, thoáng đãng, tuyệt đối không sinh thanh cuộn ngang.

---

## 2. TOUCH-MAP KHÓA CỨNG DÀNH CHO CODER (EXACT FILE TOUCHES)

### 2.1. File `internal/ui/window.go`
- **Sửa kích thước Card trong `GridWrap`:**
  ```go
  // Thay đổi dòng 185 - 187:
  // Center Section: 280x130 Fixed Card Grid inside Vertical Scroller
  a.accountListContainer = container.NewGridWrap(fyne.NewSize(280, 130))
  ```

### 2.2. File `internal/ui/ui_test.go`
- **Cập nhật các test assertion kiểm tra kích thước Card từ `300` thành `280`:**
  - `card0Size.Width == 280` (trước đây là 300)
  - `cardSize.Width == 280 && cardSize.Height == 130` (trước đây là 300x130)
  - `minSize.Width <= 280.0` (bảng Quota 228px nằm trọn trong thẻ 280px)
  - `TestArchitectSpec_QuotaStackedLayoutCard300x130_FullConformance`: Đổi tên và cập nhật assertion thành `280x130`.
- Chạy `go test -v ./internal/ui` đạt 100% PASS.

---

## 3. TIÊU CHÍ NGHIỆM THU (ACCEPTANCE CRITERIA)

- [ ] **AC-1 (Card Width 280px):** Thẻ Card có kích thước cố định khóa cứng **`280 x 130 px`** trong `GridWrap`.
- [ ] **AC-2 (Quota Table Fitting):** Bảng Quota 228px nằm vừa vặn, cân đối bên trong thẻ 280px mà không bị tràn viền hay ép kích thước.
- [ ] **AC-3 (Window Stability):** Cửa sổ ứng dụng giữ nguyên kích thước mặc định **`1008 x 630 px`**, hiển thị 3 cột thẻ thẳng hàng, không có thanh cuộn ngang.
- [ ] **AC-4 (Test Suite):** Chạy `go test -v ./internal/ui` đạt 100% PASS.
