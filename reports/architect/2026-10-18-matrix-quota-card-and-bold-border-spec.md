# Báo Cáo Đặc Tả Kiến Trúc: Bố Cục Thẻ Ma Trận 3x3 & Viền Thẻ Đậm Nổi Bật (High-Contrast Bold Border)

- **Tài liệu đặc tả:** `reports/architect/2026-10-18-matrix-quota-card-and-bold-border-spec.md`
- **Ngày lập:** 2026-10-18
- **Làn thực hiện:** Senior System Architect
- **Cơ quan chỉ đạo:** Human Owner trực tiếp thảo luận và phê duyệt phác thảo
- **Căn cứ tối cao:**
  1. Bản vẽ tay phác thảo của Human Owner tại `reports/problem/images/179234194450.jpg`.
  2. Bố cục chuẩn ma trận 3x3 do Owner trực tiếp định dạng:
     ```less
     +-----------------------------------------------------------------------+
     |  email@gmail.com                                     [Chuyển]   [🔄]  |
     +-----------------------------------------------------------------------+
     |  Quota     |            5h             |            Week              |
     |  Gemini    |  [██████████]  100%       |   [██████████]  100%         |
     |  Claude    |  [██████████]  100%       |   [██████████]  100%         |
     +-----------------------------------------------------------------------+
     ```
  3. Chỉ đạo nâng cấp viền thẻ đậm nét để thẻ nổi bật, không bị chìm trên mọi theme (Sáng / Tối / Ấm).
- **Trạng thái:** SPECIFICATION APPROVED BY OWNER — READY FOR CODER IMPLEMENTATION

---

## ⚠️ ĐIỀU KHOẢN KỶ LUẬT SẮT BẮT BUỘC DÀNH CHO CODER (MANDATORY IRON RULE)

> **CHỈ THỊ TRỰC TIẾP TỪ HUMAN OWNER:**  
> **"KHI CODER CODE XONG, CODER PHẢI CHẠY FULL BUILD BẰNG SCRIPT CỦA REPO, CODER KHÔNG ĐƯỢC SỬA SCRIPT FULL BUILD."**  
>
> - Script full build bắt buộc của repo: `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all`
> - Coder **TUYỆT ĐỐI KHÔNG ĐƯỢC PHÉP CHỈNH SỬA BẤT KỲ DÒNG NÀO** trong tệp script `scripts/build_windows.ps1`.
> - Toàn bộ quá trình build phải biên dịch sạch cả CLI lẫn GUI, chạy 100% test suite đạt PASS, và kết thúc với Exit Code 0.

---

## 1. Tóm Tắt Mục Tiêu & Yêu Cầu Thiết Kế

1. **Tái cấu trúc Quota thành Bảng Ma Trận 3 Cột x 3 Hàng (3x3 Matrix Table):**
   - Khắc phục triệt để nhược điểm của bố cục cũ (bị dàn ngang dài dòng, lặp lại tên model Claude/Gemini 2 lần, khó so sánh chéo).
   - Thiết lập bảng ma trận trực quan:
     - **Cột 1 (Model):** Tiêu đề `Quota` $\rightarrow$ Hàng 1: `Gemini` $\rightarrow$ Hàng 2: `Claude`.
     - **Cột 2 (Chu kỳ 5 giờ):** Tiêu đề `5h` $\rightarrow$ Hàng 1: `[bar] %` (Gemini) $\rightarrow$ Hàng 2: `[bar] %` (Claude).
     - **Cột 3 (Chu kỳ Tuần):** Tiêu đề `Week` $\rightarrow$ Hàng 1: `[bar] %` (Gemini) $\rightarrow$ Hàng 2: `[bar] %` (Claude).
   - Có đường phân cách mảnh giữa các cột để phân tách rõ ràng.
2. **Viền Thẻ Đậm Nét Nổi Bật (High-Contrast Bold Border):**
   - Khắc phục hiện tượng thẻ bị chìm vào màu nền canvas.
   - Tăng độ dày viền `StrokeWidth` từ `1.0px` lên **`1.8px` (hoặc `2.0px`)**.
   - Nâng độ tương phản màu viền (`BorderSubtle`) trên cả 3 theme để khối thẻ tách bạch rõ ràng dù ở chế độ Sáng, Tối hay Ấm.

---

## 2. Thiết Kế Chi Tiết Bố Cục Thẻ Ma Trận 3x3

### 2.1 Cấu Trúc Khối Giao Diện (Layout Hierarchy)
Thẻ tài khoản `NewAccountCardWithQuota` gồm 2 phần chính theo chiều dọc:

1. **Phần Đầu Thẻ (Top Row Header):**
   - Bên trái: `emailLabel` với cơ chế `TruncateEllipsis` và Tooltip hover.
   - Bên phải: Cụm thao tác `[✔ Dùng]` / `[Chuyển]` + `[🔄]`.
   - Kết cấu: `container.NewBorder(nil, nil, nil, actionCluster, emailLabel)`.

2. **Phần Bảng Ma Trận Quota (Matrix Table):**
   Gồm 3 cột dọc ngăn cách bởi vạch phân cách mảnh `|`:
   - **Cột 1 (Model Labels - Căn trái):**
     - Hàng tiêu đề: Nhãn `"Quota"` (Màu TextMuted, TextSize 11, Bold).
     - Hàng Gemini: Nhãn `"Gemini"` (Màu Text, TextSize 11, Bold).
     - Hàng Claude: Nhãn `"Claude"` (Màu Text, TextSize 11, Bold).
     - Kích thước chiều rộng: Cố định `~48 - 52 px`.
   - **Vạch ngăn cách 1:** `canvas.NewRectangle(palette.BorderSubtle)` kích thước rộng 1px, cao vừa với bảng (hoặc `widget.NewSeparator()`).
   - **Cột 2 (5h Cycle - Cân đối giữa):**
     - Hàng tiêu đề: Nhãn `"5h"` (Màu TextMuted, TextSize 11, Bold, căn giữa/căn lề đều).
     - Hàng Gemini: `container.NewHBox(miniBarGemini5h, pctGemini5h)`.
     - Hàng Claude: `container.NewHBox(miniBarClaude5h, pctClaude5h)`.
     - Thanh mini bar: Chiều rộng **`40 px`**, chiều cao **`6 px`**, bo góc 3px.
     - Số `%`: Chiều rộng `~32 - 36 px` (TextSize 11, Bold).
     - Kích thước chiều rộng cột 2: `~85 - 90 px`.
   - **Vạch ngăn cách 2:** Rộng 1px.
   - **Cột 3 (Week Cycle - Cân đối phải):**
     - Hàng tiêu đề: Nhãn `"Week"` (Màu TextMuted, TextSize 11, Bold).
     - Hàng Gemini: `container.NewHBox(miniBarGeminiWeek, pctGeminiWeek)`.
     - Hàng Claude: `container.NewHBox(miniBarClaudeWeek, pctClaudeWeek)`.
     - Kích thước chiều rộng cột 3: `~85 - 90 px`.

### 2.2 Bảng Thẩm Định Kích Thước Pixel Trên Thẻ
- Chiều rộng khả dụng bên trong thẻ: **`252 px`** (ở mức cửa sổ thu nhỏ 820px).
- Tổng chiều rộng chiếm dụng của bảng ma trận:
  $$\text{Width} = 50\text{px (Col 1)} + 1\text{px (Vạch 1)} + 88\text{px (Col 2)} + 1\text{px (Vạch 2)} + 88\text{px (Col 3)} + 8\text{px (Lề đệm)} = \mathbf{236\text{ px}}$$
- **Kết luận:** $236\text{ px} < 252\text{ px}$ (Dư 16 px khoảng đệm an toàn). Thẻ cực kỳ vừa vặn, không bị tràn hay ép dòng!

---

## 3. Thiết Kế Nâng Cấp Viền Đậm Nổi Bật (High-Contrast Bold Border)

### 3.1 Cập Nhật Palette Màu Viền Trong `internal/ui/themes.go`
Thay thế mã màu viền mờ nhạt cũ bằng mã màu có độ tương phản cao:

```go
// 1. DarkPalette (Theme Tối):
// Cũ: BorderSubtle: #313244 (chìm vào nền đen #11111b)
// Mới: Sử dụng Surface2 / Overlay0 để tách rõ ranh giới khối thẻ
BorderSubtle: color.NRGBA{R: 0x58, G: 0x5b, B: 0x70, A: 0xff}, // #585b70 (High-Contrast Border)
BorderActive: color.NRGBA{R: 0xa6, G: 0xe3, B: 0xa1, A: 0xff}, // #a6e3a1 (Green Accent sáng rõ)

// 2. LightPalette (Theme Sáng):
// Cũ: BorderSubtle: #e2e8f0 (xám rất nhạt, bị chóa trên nền #f8fafc)
// Mới: Sử dụng Slate-400 sắc nét
BorderSubtle: color.NRGBA{R: 0x94, G: 0xa3, B: 0xb8, A: 0xff}, // #94a3b8 (Slate-400 sắc sảo)
BorderActive: color.NRGBA{R: 0x25, G: 0x63, B: 0xeb, A: 0xff}, // #2563eb (Royal Blue)

// 3. WarmPalette (Theme Ấm):
// Cũ: BorderSubtle: #4a3d2f (chìm vào nền nâu #241e17)
// Mới: Nâu cà phê đậm rõ nét
BorderSubtle: color.NRGBA{R: 0x7d, G: 0x66, B: 0x4d, A: 0xff}, // #7d664d (Coffee Border)
BorderActive: color.NRGBA{R: 0xd9, G: 0x77, B: 0x06, A: 0xff}, // #d97706 (Amber Gold)
```

### 3.2 Cập Nhật Độ Dày Viền Trong `internal/ui/widgets.go`
Trong hàm `NewAccountCardWithQuota`:

```go
// Tăng độ dày viền thẻ để tạo hình khối chắc chắn, nổi bật
cardBg := canvas.NewRectangle(palette.Card)
cardBg.CornerRadius = 8
if item.IsActive {
    cardBg.FillColor = palette.CardActive
    cardBg.StrokeColor = palette.BorderActive
    cardBg.StrokeWidth = 2.0 // Thẻ đang dùng viền sáng đậm 2.0px
} else {
    cardBg.FillColor = palette.Card
    cardBg.StrokeColor = palette.BorderSubtle
    cardBg.StrokeWidth = 1.8 // Thẻ thường viền đậm 1.8px (thay vì 1.0px cũ)
}
```

---

## 4. Hướng Dẫn Thi Công Cụ Thể (Touch-Map Cho Coder)

### Tệp 1: `internal/ui/themes.go`
- Cập nhật giá trị trường `BorderSubtle` trong `DarkPalette`, `LightPalette`, `WarmPalette` sang mã màu tương phản cao đã định nghĩa ở mục 3.1.

### Tệp 2: `internal/ui/quota_badge.go`
- Bổ sung hàm xây dựng bảng ma trận:
  `func CreateMatrixQuotaTable(dq DualCycleQuota, palette ThemePalette) fyne.CanvasObject`
  - Render 3 cột theo chiều dọc (`colModel`, `col5h`, `colWeek`) kèm 2 vạch phân cách.
  - Hàng 1: `Gemini` kèm 2 thanh mini bar (5h & Week) rút dần theo %.
  - Hàng 2: `Claude` kèm 2 thanh mini bar (5h & Week) rút dần theo %.
  - Vạch mini progress bar dài `40 px`, cao `6 px`, bo tròn 3px.

### Tệp 3: `internal/ui/widgets.go`
- Trong `NewAccountCardWithQuota`:
  - Thay thế 2 dòng `row5h` và `rowWeek` cũ bằng `matrixTable := CreateMatrixQuotaTable(dq, palette)`.
  - Cập nhật `cardBg.StrokeWidth = 1.8` cho thẻ thường và `2.0` cho thẻ active.

### Tệp 4: `internal/ui/ui_test.go`
- Cập nhật các test case kiểm tra cấu trúc thẻ để xác nhận:
  - Có đủ các nhãn: `"Quota"`, `"Gemini"`, `"Claude"`, `"5h"`, `"Week"`.
  - Thanh mini progress bar hiển thị chính xác theo từng mô hình.
  - Chiều cao thẻ `TestOwnerRedesign_F5_CompactCardHeight` vẫn bảo đảm `<= 105 px` (vừa vặn, đầm chắc).

---

## 5. Quy Trình Nghiệm Thu (Acceptance Criteria)

1. **Khớp 100% Bản Vẽ & Mô Hình Chuẩn Của Owner:**
   - Hàng 1: Email + Nút `[Chuyển]` / `[✔ Dùng]` + Nút `[🔄]`.
   - Hàng 2: `Quota` | `5h` | `Week`.
   - Hàng 3: `Gemini` | `[MiniBar] %` | `[MiniBar] %`.
   - Hàng 4: `Claude` | `[MiniBar] %` | `[MiniBar] %`.
2. **Viền Thẻ Nổi Bật:**
   - Chuyển đổi giữa 3 theme Sáng, Tối, Ấm: Các thẻ đều có viền đậm 1.8px sắc nét, tách bạch rõ ràng khỏi nền canvas. Thẻ active có viền sáng 2.0px rực rỡ.
3. **Thanh Bar Co Rút Theo %:**
   - Các thanh mini progress bar rút ngắn mượt mà và đổi màu Xanh / Vàng / Đỏ đúng theo tỷ lệ % thực tế.
4. **Kỷ Luật Full Build 100% PASS:**
   - Chạy thành công lệnh:  
     `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all`
   - Exit code = 0, không có lỗi vet, build thành công cả CLI lẫn GUI.
