# Đặc Tả Kiến Trúc & Kỹ Thuật: Hiển Thị Thời Gian Đếm Lùi Quota & Mở Rộng Card Lên 350px

- **Ngày ban hành:** 2026-10-19
- **Vai trò:** System Architect
- **Thẩm quyền:** Chỉ đạo và phê duyệt trực tiếp từ Human Owner
- **Mã văn kiện:** `SPEC-ARCH-20261019-QUOTA-COUNTDOWN-CARD-350PX`
- **Trạng thái:** CHÍNH THỨC ĐÃ ĐƯỢC HUMAN OWNER PHÊ DUYỆT

---

## ⚠️ ĐIỀU KHOẢN KỶ LUẬT SẮT (MANDATORY IRON RULES)

> **CHỈ THỊ TRỰC TIẾP TỪ HUMAN OWNER:**  
> 1. **TEXT CỦA UI HIỆN TẠI CẤM THAY ĐỔI:** Toàn bộ cỡ chữ hiện tại của UI (Tiêu đề App, Status bar PID/Port, Email, Nút bấm, Tiêu đề Quota/5h/Week, Tên Model Gemini/Claude, Số phần trăm `%`) **GIỮ NGUYÊN 100% (Chuẩn 10pt hiện hành)**. Tuyệt đối cấm chỉnh sửa!  
> 2. **DUY NHẤT** phần tử nhãn thời gian đếm lùi mới được thêm vào là có cỡ chữ **`8.5pt`** (`palette.TextMuted`).  
> 3. **DUY NHẤT 1 CƠ CHẾ BÓC TÁCH:** Tuyệt đối không dùng cơ chế 2 tầng hay fallback qua lại. Chỉ lấy trực tiếp từ `bucket.FiveHourReset` và `bucket.WeeklyReset` do `ParseQuotaSummaryWithBuckets` bóc tách từ Google.

---

## 1. BẢN VẼ GIAO DIỆN & QUY CÁCH KỸ THUẬT

```text
┌────────────────────────────────────────────────────────────────────────┐
│ [AS] Antigravity Multi-Account Switcher (10pt Bold)            _ |_| X │
│ Hoạt động | PID 21484 | Port 65144 (10pt)                   [ ☀️ ▼ ] │
│ [ Danh sách tài khoản ]  [ Làm mới Quota ]  [ 📌]                      │
├────────────────────────────────────────────────────────────────────────┤
│ ┌────────────────────────────────────────────────────────────────────┐ │
│ │ ngau2hung@gmail.com (10pt)                     [✔ Dùng]  [🔄]   │ │
│ │ ────────────────────────────────────────────────────────────────── │ │
│ │ Quota (10pt)│       5h (10pt)                               │         Week (10pt)     │ │
│ │ Gemini(10pt)│ [██████████]  100%  (04h 32m) (8.5pt)         │   [████░░░░░░]  43.8%   (6d 21h 59m) (8.5pt)│ │
│ │ Claude(10pt)│ [██████████]  100%  (02h 12m) (8.5pt)         │   [██████████]  100%    (2d 07h 08m) (8.5pt)│ │
│ └────────────────────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────────────────┘
```

1. **Hiển thị thời gian đếm lùi hồi phục Quota:**
   - **Chu kỳ 5h:** Định dạng `"(04h 32m)"`, `"(02h 12m)"` (cố định 10 ký tự). Nếu thời gian `<= 0`: hiển thị `"(Ready)"`.
   - **Chu kỳ Tuần:** Định dạng `"(6d 21h 59m)"`, `"(2d 07h 08m)"` (cố định 13 ký tự).
   - **Không có dữ liệu / Lỗi:** Hiển thị `"--"`.
   - **Thị giác:** Cỡ chữ **8.5pt**, màu xám mờ **`palette.TextMuted`** trong ngoặc đơn, cùng hàng ngang với thanh bar và con số `%`.
2. **Khóa kích thước cố định (Zero Resizing / Zero Jank):**
   - Giữ nguyên độ dài thanh mini progress bar: **`40 px`**.
   - Mở rộng kích thước Card trong `GridWrap`: cố định **`350 x 100 px`**.
   - Mở rộng chiều rộng cửa sổ: cố định **`1090 x 630 px`** (chứa vừa vặn 3 cột thẻ thẳng tắp).
3. **Cơ chế cập nhật dữ liệu:**
   - Dữ liệu `resetTime` bóc tách duy nhất từ `bucket.FiveHourReset` và `bucket.WeeklyReset` trong `ParseQuotaSummaryWithBuckets` (`internal/telemetry/client.go`).
   - Cập nhật thời gian đồng bộ tại chỗ trong hàm `UpdateQuota` mỗi khi có dữ liệu từ `AutoTelemetryLoop` (60s) hoặc khi bấm Làm mới thủ công. Không sinh thêm chu kỳ hay Ticker riêng.

---

## 2. TOUCH-MAP CHI TIẾT DÀNH CHO CODER

### 2.1. File `internal/ui/window.go`
- **Kích thước cửa sổ mặc định:** Đổi `DefaultWindowWidth` từ `1008.0` thành **`1090.0`**. Giữ nguyên `DefaultWindowHeight = 630.0`.
- **Kích thước Card trong GridWrap:** Đổi `fyne.NewSize(318, 100)` thành **`fyne.NewSize(350, 100)`**.

---

### 2.2. File `internal/ui/quota_badge.go`
- **Bổ sung 4 trường thời gian vào `DualCycleQuota`:**
  ```go
  Claude5hReset   time.Time
  ClaudeWeekReset time.Time
  Gemini5hReset   time.Time
  GeminiWeekReset time.Time
  ```
- **Hàm `ExtractDualCycleQuota` (DUY NHẤT 1 CƠ CHẾ):**
  Lấy trực tiếp từ các trường của bucket, không bóc tách 2 tầng hay quét rawBuckets:
  ```go
  Claude5hReset:   quota.Claude.FiveHourReset,
  ClaudeWeekReset: quota.Claude.WeeklyReset,
  Gemini5hReset:   quota.Gemini.FiveHourReset,
  GeminiWeekReset: quota.Gemini.WeeklyReset,
  ```
- **Hàm định dạng thời gian `formatCountdown(resetTime time.Time, isWeekly bool) string`:**
  ```go
  func formatCountdown(resetTime time.Time, isWeekly bool) string {
      if resetTime.IsZero() {
          return "--"
      }
      duration := time.Until(resetTime)
      if duration <= 0 {
          return "(Ready)"
      }
      if !isWeekly {
          h := int(duration.Hours())
          m := int(duration.Minutes()) % 60
          return fmt.Sprintf("(%02dh %02dm)", h, m)
      }
      d := int(duration.Hours()) / 24
      h := int(duration.Hours()) % 24
      m := int(duration.Minutes()) % 60
      return fmt.Sprintf("(%dd %02dh %02dm)", d, h, m)
  }
  ```
- **Struct `matrixQuotaCell` & layout ô:**
  - Thêm `timeText *canvas.Text` (cỡ chữ cố định **`8.5`**, màu `palette.TextMuted`) và `isWeekly bool`.
  - Cập nhật `matrixCellLayout`: Xếp ngang 3 phần tử `[miniBar 40px] [pctText (giữ nguyên cỡ cũ 10pt)] [timeText (8.5pt)]`.
- **Struct `matrixTableLayout`:**
  - Khóa cứng kích thước các cột trong bảng:
    - Cột Model: `48 px`
    - Cột 5h: `128 px`
    - Cột Week: `146 px`
    - Vách ngăn & khoảng cách: `18 px`
    - Tổng bảng Quota: `340 px` (vừa khít trong thẻ 350px).
- **Hàm `update` của `matrixQuotaCell`:**
  - Cập nhật tại chỗ: `c.timeText.Text = formatCountdown(resetTime, c.isWeekly)` và `c.timeText.Refresh()`.

---

### 2.3. File `internal/ui/widgets.go`
- Trong `NewAccountCardWidgetWithQuota`: Truyền 4 trường `ResetTime` vào `createMatrixQuotaTableWithCells`.
- Trong `(w *AccountCardWidget) UpdateQuota`: Cập nhật cả 4 cell tại chỗ với `resetTime` mới mỗi khi có dữ liệu Quota từ AutoTelemetryLoop (60s) hoặc manual refresh.

---

### 2.4. File `internal/ui/ui_test.go`
- Thêm Unit Test cho hàm `formatCountdown` với các ca kiểm thử: 5h, Week, `<= 0` (`(Ready)`), và `ZeroTime` (`"--"`).
- Cập nhật các mock data trong test suite để 100% test PASS.

---

## 3. TIÊU CHÍ NGHIỆM THU (ACCEPTANCE CRITERIA)

1. **AC-1:** Giao diện hiển thị đầy đủ thời gian `(04h 32m)` cho cột 5h và `(6d 21h 59m)` cho cột Week trên cùng 1 hàng.
2. **AC-2:** Text UI hiện tại giữ nguyên 100% kích thước cũ (10pt). Duy nhất nhãn thời gian mới có kích thước 8.5pt.
3. **AC-3:** Dữ liệu thời gian lấy trực tiếp và duy nhất từ `FiveHourReset` và `WeeklyReset`. Không dùng cơ chế 2 tầng.
4. **AC-4:** Thẻ Card có kích thước cố định `350x100px`, cửa sổ cố định `1090x630px`.
5. **AC-5:** Chạy `go test -v ./internal/ui` đạt 100% PASS.
