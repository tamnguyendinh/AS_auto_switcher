# Đặc Tả Kiến Trúc & Kỹ Thuật: Bố Cục Quota 2 Tầng (Stacked Layout), Thẻ Card 300x130px & Cửa Sổ 1008x630px

- **Ngày lập:** 2026-10-19
- **Vai trò:** System Architect
- **Thẩm quyền:** Chỉ đạo trực tiếp và thảo luận cùng Human Owner
- **Mã văn kiện:** `SPEC-ARCH-20261019-QUOTA-STACKED-CARD-300X130`
- **Trạng thái:** CHÍNH THỨC ĐÃ ĐƯỢC HUMAN OWNER PHÊ DUYỆT (OFFICIAL APPROVED)

---

## ⚠️ ĐIỀU KHOẢN KỶ LUẬT SẮT (MANDATORY IRON RULES)

> **CHỈ THỊ TRỰC TIẾP TỪ HUMAN OWNER:**
> 1. **BẢO TỒN CỠ CHỮ CHUẨN NGÀNH (TYPOGRAPHY SCALE):**
>    - Toàn bộ text hiện tại của UI (Tiêu đề App, Status bar PID/Port, Email, Nút bấm, Tiêu đề Quota/5h/Week, Tên Model Gemini/Claude, Số phần trăm `%`) **GIỮ NGUYÊN 100% (Chuẩn 10pt hiện hành)**.
>    - **DUY NHẤT** phần tử nhãn thời gian đếm lùi là có cỡ chữ **`8.5pt`** (`palette.TextMuted`).
> 2. **BẢO LƯU 100% CƠ CHẾ IN-PLACE REACTIVE UPDATE (ZERO JANK / ZERO RESIZING):**
>    - Tuyệt đối không hủy bỏ hoặc khởi tạo lại đối tượng Canvas/Widget trong chu kỳ cập nhật dữ liệu Quota.
>    - Mọi cập nhật đều thực hiện trực tiếp tại chỗ trên các con trỏ `c.pctText.Text`, `c.timeText.Text`, `c.fillBar.Resize()` và gọi `Refresh()` cục bộ.
> 3. **DUY NHẤT 1 CƠ CHẾ BÓC TÁCH DỮ LIỆU THỜI GIAN:**
>    - Chỉ lấy trực tiếp từ `bucket.FiveHourReset` và `bucket.WeeklyReset` do `ParseQuotaSummaryWithBuckets` bóc tách từ Google telemetry. Không dùng cơ chế fallback 2 tầng.
> 4. **KHÓA CỨNG HÌNH HỌC (FIXED GEOMETRY INVARIANTS):**
>    - Kích thước thẻ Card trong `GridWrap`: **`300 x 130 px`**.
>    - Kích thước cửa sổ mặc định: **`1008 x 630 px`** (`DefaultWindowWidth = 1008.0`, `DefaultWindowHeight = 630.0`).
>    - Độ dài ray tiến trình MiniBar: **`40.0 px`**, chiều cao **`6.0 px`**, `CornerRadius = 3.0 px`.

---

## 1. BẢN VẼ THỊ GIÁC & QUY CÁCH HÌNH HỌC (VISUAL SPECIFICATION)

### 1.1. Bản Vẽ Tổng Thể Cửa Sổ 1008 x 630 px
```text
┌────────────────────────────────────────────────────────────────────────────────────────┐
│ [AS] Antigravity Multi-Account Switcher (10pt Bold)                            _ |_| X │
│ Hoạt động | PID 21484 | Port 65144 (10pt)                                     [ ☀️ ▼ ] │
│ [ Danh sách tài khoản ]  [ Làm mới Quota ]  [ 📌]                                      │
├────────────────────────────────────────────────────────────────────────────────────────┤
│ ┌──────────────────────────────┐ ┌──────────────────────────────┐ ┌──────────────────┐ │
│ │ ngau2hung@gmail.com [✔][🔄]  │ │ account2@gmail.com [Chuyển][🔄]│ │ ...              │ │
│ │ ──────────────────────────── │ │ ──────────────────────────── │ │                  │ │
│ │ Quota   │   5h   │   Week    │ │ Quota   │   5h   │   Week    │ │                  │ │
│ │         │(04h32m)│(6d 21h59m)│ │         │(01h15m)│(5d 08h12m)│ │                  │ │
│ │ Gemini  │[████]100%[██░░]43% │ │ Gemini  │[████]100%[████]100%│ │                  │ │
│ │ ────────┼────────┼────────── │ │ ────────┼────────┼────────── │ │                  │ │
│ │         │(02h12m)│(2d 07h08m)│ │         │(Ready) │(1d 14h20m)│ │                  │ │
│ │ Claude  │[████]100%[████]100%│ │ Claude  │[████]100%[████]100%│ │                  │ │
│ └──────────────────────────────┘ └──────────────────────────────┘ └──────────────────┘ │
│ (3 Cột Thẻ 300x130px Thẳng Tắp - Không Cuộn Ngang - Co Gọn Thoáng Đãng Trên Laptop 14")│
└────────────────────────────────────────────────────────────────────────────────────────┘
```

### 1.2. Bản Vẽ Phóng To Thẻ Card (300 x 130 px) — Phương Án A (Stacked Countdown Top)
```text
┌──────────────────────────────────────────────────────────────────────────┐  ▲
│ ngau2hung@gmail.com (10pt)                             [✔ Dùng]  [🔄]   │  │ 26px (Dòng 1: Email + Nút)
│ ──────────────────────────────────────────────────────────────────────── │  │  8px (Khoảng cách đệm)
│ Quota (10pt)  │         5h (10pt)       │          Week (10pt)           │  │ 15px (Hàng 0: Header Cột)
│               │    (04h 32m) (8.5pt)    │       (6d 21h 59m) (8.5pt)     │  │ 13px (Hàng 1a: Thời gian đếm lùi)
│ Gemini (10pt) │  [██████████]  100%     │       [████░░░░░░]  43.8%      │  │ 15px (Hàng 1b: Bar 40px + % 10pt)
│ ──────────────┼─────────────────────────┼────────────────────────────────┤  │  4px (Đường vách ngăn mờ)
│               │    (02h 12m) (8.5pt)    │       (2d 07h 08m) (8.5pt)     │  │ 13px (Hàng 2a: Thời gian đếm lùi)
│ Claude (10pt) │  [██████████]  100%     │       [██████████]  100%       │  │ 15px (Hàng 2b: Bar 40px + % 10pt)
└──────────────────────────────────────────────────────────────────────────┘  ▼ Padding trên dưới 20px -> Tổng 130px
  ◄── 48px ───►◄9►◄────── 78px ─────────►◄9►◄────────── 86px ─────────────►
  (Tổng bảng Quota = 48 + 9 + 78 + 9 + 86 = 230 px | Lọt thỏm trong thẻ 300px)
```

---

## 2. QUY CÁCH PHÂN CHIA HÌNH HỌC TOÁN HỌC (GEOMETRIC BUDGET)

### 2.1. Phân Bổ Bề Rộng (Horizontal Budget): Thẻ 300 px
- **Bề rộng thẻ Card:** `300.0 px`.
- **Padding nội bộ của Card (Stack Padded):** `16.0 px` (trái `8px`, phải `8px`).
- **Bề rộng khả dụng tối đa:** `300 - 16 = 284.0 px`.
- **Bảng Quota chiếm:** `230.0 px`, phân bổ chi tiết:
  1. **Cột 1 (Model Name):** CỐ ĐỊNH **`48.0 px`**. Chứa "Quota", "Gemini", "Claude" (10pt Bold).
  2. **Vách ngăn 1 (Sep 1):** CỐ ĐỊNH **`1.0 px`** (`palette.BorderSubtle`) + 2 khoảng cách lề `4.0 px` = **`9.0 px`**.
  3. **Cột 2 (Chu kỳ 5h):** CỐ ĐỊNH **`78.0 px`**. 
     - Dòng 1: Text đếm lùi `(04h 32m)` (10 ký tự ở 8.5pt chiếm ~55px) $\rightarrow$ Dư 23px.
     - Dòng 2: `[MiniBar 40px] + [Gap 4px] + [100% ~28px] = 72px` $\rightarrow$ Vừa khít chuẩn xác.
  4. **Vách ngăn 2 (Sep 2):** CỐ ĐỊNH **`1.0 px`** (`palette.BorderSubtle`) + 2 khoảng cách lề `4.0 px` = **`9.0 px`**.
  5. **Cột 3 (Chu kỳ Tuần Week):** CỐ ĐỊNH **`86.0 px`**.
     - Dòng 1: Text đếm lùi `(6d 21h 59m)` (13 ký tự ở 8.5pt chiếm ~76px) $\rightarrow$ Dư 10px.
     - Dòng 2: `[MiniBar 40px] + [Gap 4px] + [43.8% ~32px] = 76px` $\rightarrow$ Vừa khít chuẩn xác.
- **Biên an toàn (Safety Margin):** `284 - 230 = 54.0 px` đệm lề tự do, đảm bảo 100% không bao giờ xảy ra hiện tượng tràn chữ, ép chữ hay vỡ khung.

### 2.2. Phân Bổ Chiều Cao (Vertical Budget): Thẻ 130 px
- **Padding trên & dưới (Card Padding):** `20.0 px` (trên `10px`, dưới `10px`).
- **Dòng 1 (TopRow: Email & Action Cluster):** `26.0 px`.
- **Khoảng đệm giữa TopRow và Bảng Quota:** `8.0 px`.
- **Bảng Quota chiếm tổng cộng:** `76.0 px`:
  - Hàng 0 (Header `Quota | 5h | Week`): `15.0 px`.
  - Khoảng đệm dưới header: `2.0 px`.
  - Khối Gemini (2 tầng): `28.0 px`:
    - Tầng trên (Countdown `(04h 32m)`): `13.0 px` (căn lề trái thẳng hàng ray Bar).
    - Tầng dưới (Model `Gemini` + `[Bar 40px] 100%`): `15.0 px`.
  - Vách phân cách ngang mờ (`BorderSubtle`): `1.0 px` (cộng khoảng đệm 2px trên dưới = `3.0 px`).
  - Khối Claude (2 tầng): `28.0 px`:
    - Tầng trên (Countdown `(02h 12m)`): `13.0 px`.
    - Tầng dưới (Model `Claude` + `[Bar 40px] 100%`): `15.0 px`.
- **Tổng cộng chiều cao:** `20 + 26 + 8 + 76 = 130.0 px` (Ăn khớp 100% với `GridWrap fyne.NewSize(300, 130)`).

---

## 3. THIẾT KẾ KIẾN TRÚC THÀNH PHẦN (COMPONENT ARCHITECTURE)

### 3.1. Cấu Trúc Cell 2 Tầng: `matrixStackedQuotaCell`
Mỗi ô Quota cho một Model và một Chu kỳ (Gemini 5h, Gemini Week, Claude 5h, Claude Week) quản lý:
- `timeText *canvas.Text`: Kích thước `8.5pt`, màu `palette.TextMuted`, nằm ở tầng trên.
- `barContainer *fyne.Container`: Chứa `bgTrack` (40x6px) và `fillBar` (nội suy tỉ lệ), nằm ở tầng dưới.
- `pctText *canvas.Text`: Kích thước `10pt Bold`, màu theo Tier (Xanh/Vàng/Đỏ), nằm ở tầng dưới kế bên `barContainer`.

```text
┌───────────────────────────┐
│ (04h 32m)                 │  ◄── timeText (8.5pt TextMuted) - Tầng Trên
│ [████████████]  100%      │  ◄── [barContainer 40px] + [pctText 10pt Bold] - Tầng Dưới
└───────────────────────────┘
```

### 3.2. Cấu Trúc Cột Model Cân Xứng (Visual Baseline Alignment)
Tại Cột 1:
- Header: Text `"Quota"` (10pt Bold, TextMuted) ở hàng 0.
- Khối Gemini: Tầng trên để trống (hoặc spacer `13px`), tầng dưới đặt Text `"Gemini"` (10pt Bold) ở `y = 13px`. Nhờ đó, chữ `"Gemini"` đứng sóng đôi ngay ngắn cùng hàng với thanh Bar `[██████████]` và số `100%`!
- Khối Claude: Tương tự, chữ `"Claude"` (10pt Bold) đứng sóng đôi cùng hàng với thanh Bar của Claude.

### 3.3. Bảo Toàn Cơ Chế In-place Reactive Update (Zero Jank)
Phương thức `update(fraction float64, resetTime time.Time, hasData bool)` của cell:
```go
func (c *matrixQuotaCell) update(fraction float64, resetTime time.Time, hasData bool) {
    if c == nil {
        return
    }
    const barWidth = float32(MatrixMiniBarWidth) // 40.0 px

    if !hasData || fraction < 0 {
        c.pctText.Text = "--%"
        c.pctText.Color = ColorQuotaMuted
        c.fillBar.FillColor = ColorQuotaMuted
        c.fillBar.Resize(fyne.NewSize(0, c.layout.barHeight))
    } else {
        clamped := fraction
        if clamped > 1.0 { clamped = 1.0 } else if clamped < 0 { clamped = 0 }
        c.pctText.Text = formatFraction(fraction)
        c.pctText.Color = GetQuotaTierColor(fraction)
        c.fillBar.FillColor = GetQuotaTierColor(fraction)
        c.fillBar.Resize(fyne.NewSize(barWidth*float32(clamped), c.layout.barHeight))
    }

    if c.timeText != nil {
        if !hasData {
            c.timeText.Text = "--"
        } else {
            c.timeText.Text = formatCountdown(resetTime, c.isWeekly)
        }
        c.timeText.Refresh()
    }
    c.fillBar.Refresh()
    c.pctText.Refresh()
    // TUYỆT ĐỐI CẤM GỌI c.container.Refresh() HOẶC WINDOW RESIZE
}
```

---

## 4. TOUCH-MAP KHÓA CỨNG DÀNH CHO CODER (EXACT FILE TOUCHES)

### 4.1. `internal/ui/window.go`
- **Cửa sổ mặc định:**
  ```go
  const (
      DefaultWindowWidth  float32 = 1008.0 // Đưa từ 1090.0 quay về chuẩn ban đầu 1008.0
      DefaultWindowHeight float32 = 630.0  // Giữ nguyên 630.0
      MinWindowWidth      float32 = 480.0
      MinWindowHeight     float32 = 360.0
  )
  ```
- **Kích thước Card GridWrap:**
  ```go
  // Center Section: 300x130 Fixed Card Grid inside Vertical Scroller
  a.accountListContainer = container.NewGridWrap(fyne.NewSize(300, 130))
  ```

### 4.2. `internal/ui/quota_badge.go`
- **Hằng số kích thước:**
  - `MatrixMiniBarWidth = 40.0 px` (giữ nguyên).
- **Struct `matrixStackedCellLayout` (Layout 2 tầng):**
  - Quản lý 2 tầng: Dòng 1 chứa `timeText` (chiều cao `13px`), Dòng 2 chứa `[barContainer 40px] [gap 4px] [pctText]` (chiều cao `15px`).
  - `MinSize`: Trả về `fyne.NewSize(width, 28.0)`.
- **Struct `matrixTableLayout`:**
  - Khóa cứng các chỉ số độ rộng cột:
    ```go
    const col1W = float32(48.0)     // Cột Model: CỐ ĐỊNH 48px
    const sepW = float32(1.0)       // Vách ngăn đứng: CỐ ĐỊNH 1px
    const colGap = float32(4.0)     // Khoảng cách giữa các cột: CỐ ĐỊNH 4px
    const col5hW = float32(78.0)    // Cột 5h: CỐ ĐỊNH 78px
    const colWeekW = float32(86.0)  // Cột Week: CỐ ĐỊNH 86px
    // Tổng chiều rộng MinSize = 48 + 1 + 78 + 1 + 86 + 16 (gaps) = 230.0 px
    // Tổng chiều cao MinSize = 77.0 px
    ```
  - Bố trí vách ngăn ngang mờ giữa hàng Gemini và Claude.

### 4.3. `internal/ui/widgets.go`
- Đảm bảo `NewAccountCardWidgetWithQuota` bao bọc trong `container.NewStack(cardBg, container.NewPadded(cardVBox))` khớp với kích thước `300 x 130 px`.

### 4.4. `internal/ui/ui_test.go`
- Cập nhật các test assertion:
  - `sz.Width == 1008 && sz.Height == 630`
  - `cardSize.Width == 300 && cardSize.Height == 130`
  - `matrixTableLayout MinSize.Width <= 300` (giá trị thực tế ~230px)
  - `matrixTableLayout MinSize.Height <= 80` (giá trị thực tế ~77px)
- Đảm bảo 100% test pass.

---

## 5. TIÊU CHÍ NGHIỆM THU (ACCEPTANCE CRITERIA)

- [ ] **AC-1 (Stacked Layout):** Bố cục Quota 2 tầng (Phương án A): Dòng trên hiển thị thời gian đếm lùi `(04h 32m)` / `(6d 21h 59m)` (8.5pt `TextMuted`), dòng dưới hiển thị thanh MiniBar (40px) và số `%` (10pt Bold).
- [ ] **AC-2 (Model Baseline):** Tên Model (`Gemini`, `Claude` 10pt Bold) đứng ngang hàng cân xứng với thanh MiniBar ở dòng dưới.
- [ ] **AC-3 (Thẻ Card 300x130):** Thẻ Card có kích thước khóa cứng chính xác `300 x 130 px` trong `GridWrap`.
- [ ] **AC-4 (Cửa Sổ 1008x630):** Cửa sổ ứng dụng có kích thước mặc định `1008 x 630 px`, hiển thị 3 cột thẻ thẳng hàng, thoáng đãng, không có thanh cuộn ngang.
- [ ] **AC-5 (Zero Jank / In-place):** Reload Quota tự động hoặc thủ công chỉ cập nhật dữ liệu tại chỗ, không giật màn hình, không resize cửa sổ.
- [ ] **AC-6 (Test Suite):** Chạy `go test -v ./internal/ui` đạt 100% PASS.

