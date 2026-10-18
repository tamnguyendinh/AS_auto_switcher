# Báo Cáo Đặc Tả Kiến Trúc: Thanh Đo Tiến Trình Đồ Họa Mini Rút Dần Theo % Cho Quota Thẻ Tài Khoản

- **Tài liệu đặc tả:** `reports/architect/2026-10-18-mini-graphical-progress-bar-spec.md`
- **Ngày lập:** 2026-10-18
- **Làn thực hiện:** Senior System Architect
- **Cơ quan chỉ đạo:** CEO Agent (Thread ID: `01a14378-4a08-77c1-88d5-666ad8c3592d`)
- **Căn cứ tối cao:** Chỉ thị trực tiếp từ Human Owner lựa chọn **Cách 2** (Thanh đồ họa mini bo góc rút dần theo thời gian thực) và **quy định điều phối tuần tự**: *Chỉ triển khai mục này sau khi Coder hoàn tất 100% mục thu nhỏ kích thước UI 70%*.
- **Trạng thái:** ARCHITECT SPECIFICATION APPROVED — QUEUED FOR PHASE 2 IMPLEMENTATION

---

## 1. Tóm Tắt Mục Tiêu & Yêu Cầu Của Human Owner

1. **Trực quan hóa hạn ngạch Quota:** Thay vì chỉ hiển thị chấm tròn trạng thái (Status Dot) và chữ số, tích hợp **Thanh đo tiến trình đồ họa mini (Mini Graphical Progress Bar)** bo góc hiện đại (tương tự vạch pin iPhone/MacBook) cho cả Claude và Gemini trên từng chu kỳ thời gian (5h và tuần).
2. **Cơ chế rút ngắn động theo thời gian thực (%-Proportional Depletion):** Thanh đo tự động co ngắn lại mượt mà chính xác theo số phần trăm bên cạnh (ví dụ `100%` đầy thanh, `91.6%` rút nhẹ, `46%` còn một nửa, `12%` còn một đoạn ngắn).
3. **Màu sắc ngữ nghĩa 3 cấp:** Vệt sáng tự động đổi màu theo 3 cấp ngưỡng:
   - **Xanh Emerald (`#22c55e`):** Hạn ngạch > 50%.
   - **Vàng Amber (`#f59e0b`):** Hạn ngạch 15% – 50%.
   - **Đỏ Crimson (`#ef4444`):** Hạn ngạch < 15%.
4. **Quy định điều phối nghiêm ngặt:** Coder **chỉ được phép bắt đầu triển khai tính năng này sau khi đã hoàn thành và nghiệm thu 100% việc thu nhỏ UI 70%**.

---

## 2. Thiết Kế Chi Tiết Widget `MiniProgressBar` Trong Fyne

### 2.1 Thông Số Hình Học (Geometry)
- **Chiều rộng (Width):** `50 px` (cố định, siêu nhỏ gọn).
- **Chiều cao (Height):** `6 px`.
- **Bo góc (CornerRadius):** `3 px` (viên thuốc / capsule tròn 2 đầu).
- **Thanh nền (Background Track):**
  - Màu sắc: `palette.BorderSubtle` hoặc xám mờ bán trong suốt `color.NRGBA{R: 0x44, G: 0x44, B: 0x55, A: 0x88}`.
  - Kích thước: `Width = 50px, Height = 6px`, CornerRadius = 3.
- **Thanh tiến trình (Progress Fill Bar):**
  - Chiều rộng động: `fillWidth = 50.0 * float32(fraction)`.
  - Giới hạn (Clamp): nếu `fraction < 0` $\rightarrow 0$; nếu `fraction > 1.0` $\rightarrow 50$.
  - Màu sắc: Lấy theo `GetQuotaTierColor(fraction)`.
  - CornerRadius = 3.

### 2.2 Cấu Trúc Hiển Thị Trực Quan (Wireframe)
```text
Dòng 2: 5h:   Claude [██████████] 100%    Gemini [████████░░] 91.6%
              |<- 50px ->|               |<- 50px ->|

Dòng 3: week: Claude [██████████] 100%    Gemini [██████████] 98.0%
              |<- 50px ->|               |<- 50px ->|
```

*(Lưu ý: Ký tự minh họa `███` đại diện cho thanh đồ họa Canvas mượt mà 6px bo góc, không phải ký tự text thô).*

---

## 3. Thẩm Định Kích Thước & Khả Năng Hiển Thị Trên Thẻ 70%

Trên giao diện đã thu nhỏ 70%, chiều rộng nội dung khả dụng trong mỗi thẻ là **320 px** (cửa sổ 1008px) và **264 px** (cửa sổ tối thiểu 840px).

### 3.1 Bảng Tính Chiều Rộng Chi Tiết Cho Dòng 2 & 3
| Thành phần trên dòng | Chiều rộng (px) | Ghi chú |
|---|---|---|
| **Nhãn chu kỳ (`5h:` / `week:`)** | `~22 - 32 px` | TextSize 11, Bold |
| **Mục Claude:** | | |
| - Chữ `"Claude"` | `~40 px` | TextSize 11, Bold |
| - Khoảng cách đệm | `4 px` | Spacing |
| - Thanh Mini Bar | `50 px` | Bo góc 6px cao |
| - Khoảng cách đệm | `4 px` | Spacing |
| - Số % (`100%` / `--%`) | `~32 px` | TextSize 11, Bold |
| *Tổng mục Claude* | **130 px** | |
| **Khoảng cách giữa Claude & Gemini** | `8 px` | Spacing giữa 2 mô hình |
| **Mục Gemini:** | | |
| - Chữ `"Gemini"` | `~42 px` | TextSize 11, Bold |
| - Khoảng cách đệm | `4 px` | Spacing |
| - Thanh Mini Bar | `50 px` | Bo góc 6px cao |
| - Khoảng cách đệm | `4 px` | Spacing |
| - Số % (`91.6%`) | `~36 px` | TextSize 11, Bold |
| *Tổng mục Gemini* | **136 px** | |
| **TỔNG CỘNG CẢ DÒNG 2** | `22 + 130 + 8 + 136 =` **296 px** | **< 320 px (Dư 24 px)** |
| **TỔNG CỘNG CẢ DÒNG 3** | `32 + 130 + 8 + 136 =` **306 px** | **< 320 px (Dư 14 px)** |

### 3.2 Tối Ưu Tự Động Co Giãn Cho Mức Cửa Sổ Nhỏ Nhất (840px)
Nếu người dùng kéo cửa sổ co về mức tối thiểu `840 px` (thẻ rộng `264 px`):
- Để thanh bar vẫn hiển thị hoàn hảo mà không tràn, Coder Lane chỉ cần định nghĩa chiều rộng thanh bar là `40 px` (hoặc co giãn động theo tỷ lệ), khi đó tổng cả dòng chỉ còn:
  `22 + 120 + 8 + 126 =` **276 px** (hoặc ẩn bớt nhãn chữ nếu cần). Tuy nhiên ở kích thước 1008px chuẩn của Owner thì thanh 50px là tỷ lệ vàng cực kỳ cân đối.

---

## 4. Kế Hoạch Triển Khai Cho Coder (Phase 2 Touch-Map)

> **CẢNH BÁO TIẾN ĐỘ (EXECUTION GATE):** Coder Lane CHỈ ĐƯỢC PHÉP thao tác các nội dung dưới đây sau khi Giai đoạn 1 (Thu nhỏ UI 70%) đã hoàn thành và vượt qua 100% test.

### 4.1 Tệp Cần Chỉnh Sửa
1. **`internal/ui/quota_badge.go`:**
   - Xây dựng hàm `createMiniProgressBar(fraction float64, barWidth float32) fyne.CanvasObject`:
     - Tạo `bgRect := canvas.NewRectangle(...)` với kích thước `Size{Width: barWidth, Height: 6}`, `CornerRadius: 3`.
     - Tạo `fillRect := canvas.NewRectangle(...)` với kích thước `Size{Width: barWidth * float32(fraction), Height: 6}`, `CornerRadius: 3`.
     - Đặt vào container dạng Stack / Custom Layout để thanh fill đè lên thanh nền.
   - Cập nhật hàm `createModelQuotaItem(modelName string, fraction float64, hasData bool)`:
     - Thay thế `StatusDot` (chấm tròn 7x7) bằng `createMiniProgressBar(fraction, 50)`.
     - Kết cấu HBox: `[Tên Model] [MiniProgressBar] [Số %]`.
2. **`internal/ui/ui_test.go`:**
   - Cập nhật kiểm thử `TestOwnerRedesign_F1_DualCycleIndependentFractionsAndMissing`:
     - Kiểm tra sự hiện diện của `MiniProgressBar` trên các thẻ tài khoản thay cho dot cũ.
   - Kiểm tra chiều cao thẻ `TestOwnerRedesign_F5_CompactCardHeight` vẫn bảo đảm `<= 100px` (do thanh bar chỉ cao 6px, hoàn toàn không làm đội chiều cao thẻ).

---

## 5. Tiêu Chuẩn Nghiệm Thu Của Phase 2 (Acceptance Criteria)

1. **Hiển thị đồ họa mượt mà:** Thanh bar có viền bo tròn 3px, chiều cao 6px mảnh mai, nằm thẳng hàng với chữ.
2. **Tự động co rút theo %:**
   - `1.0` (100%): Vạch đầy 50px màu Xanh.
   - `0.916` (91.6%): Vạch rút còn 45.8px màu Xanh.
   - `0.46` (46%): Vạch rút còn 23px màu Vàng.
   - `0.12` (12%): Vạch rút còn 6px màu Đỏ.
   - Khi không có data hoặc lỗi: Vạch mờ xám.
3. **Thẩm mỹ:** Bố cục 3 dòng của thẻ tài khoản vẫn gọn gàng, sắc nét, không bị đội chiều cao hay tràn lề.
4. **Zero Test Regressions:** `go test ./...` 100% PASS.

---

## 6. Lệnh Điều Phối Gửi CEO Agent

Kiến trúc sư kính chuyển bản đặc tả này tới CEO Agent với chỉ thị đóng băng tiến độ:
- **LƯU TRỮ VÀO HÀNG ĐỢI (QUEUED):** Giữ bản đặc tả Phase 2 này.
- **ĐIỀU KIỆN KÍCH HOẠT:** Đợi Coder Lane thi công xong gói 1 (Thu nhỏ UI 70% theo commit `eecb4ad`) $\rightarrow$ Sau khi nghiệm thu PASS mới giải phóng lệnh cho Coder triển khai tiếp Phase 2 này.
