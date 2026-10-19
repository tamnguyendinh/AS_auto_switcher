# Đặc Tả Kiến Trúc & Kỹ Thuật: Loại Bỏ 2 Vách Ngăn Dọc Trong Bảng Quota Thẻ Tài Khoản

- **Ngày lập:** 2026-10-19
- **Vai trò:** System Architect
- **Thẩm quyền:** Chỉ đạo trực tiếp từ Human Owner qua hình ảnh minh chứng `reports/problem/images/screenshot_1792410245.png`
- **Mã văn kiện:** `SPEC-ARCH-20261019-REMOVE-VERTICAL-SEPARATORS-QUOTA-MATRIX`
- **Trạng thái:** CHÍNH THỨC ĐÃ ĐƯỢC HUMAN OWNER PHÊ DUYỆT (OFFICIAL APPROVED)

---

## ⚠️ ĐIỀU KHOẢN KỶ LUẬT SẮT (MANDATORY IRON RULES)

> **CHỈ THỊ TRỰC TIẾP TỪ HUMAN OWNER:**
> 1. **LOẠI BỎ 2 VÁCH NGĂN DỌC (`sep1`, `sep2`):**
>    - Xóa bỏ hoàn toàn 2 vách ngăn kẻ dọc giữa Cột Model ↔ Cột 5h và giữa Cột 5h ↔ Cột Week trong bảng Quota.
>    - Thay thế bằng khoảng cách đệm tự nhiên (whitespace `colGap = 8.0 px`) để giao diện thoáng đãng, thanh thoát và hiện đại hơn.
> 2. **GIỮ NGUYÊN VÁCH NGĂN NGANG (`hSep`):**
>    - Giữ nguyên đường vách kẻ ngang mờ (`hSep`, màu `palette.BorderSubtle`, 1px) giữa hàng Gemini và hàng Claude để phân định rõ ràng 2 model.
> 3. **BẢO TỒN NGUYÊN VẸN CÁC THÔNG SỐ ĐÃ ĐƯỢC PHÊ DUYỆT:**
>    - Thẻ Card: CỐ ĐỊNH **`300 x 130 px`** trong `GridWrap`.
>    - Cửa sổ ứng dụng: CỐ ĐỊNH **`1008 x 630 px`**.
>    - Typography: Text chính 10pt Bold, text thời gian đếm lùi 8.5pt `TextMuted`.
>    - In-place reactive update (Zero Jank / Zero Resizing) bảo toàn 100%.

---

## 1. PHÂN TÍCH TỪ MINH CHỨNG THỰC TẾ (`screenshot_1792410245.png`)

Từ ảnh chụp màn hình thực tế do Human Owner cung cấp tại `reports/problem/images/screenshot_1792410245.png`:
- 2 mũi tên đỏ chỉ thẳng vào 2 đường kẻ dọc màu xám (`BorderSubtle`) ngăn cách 3 cột trong bảng Quota.
- Lời chỉ đạo: *"2 đường kẻ dọc này nên xoá bỏ thì nhìn sẽ đẹp hơn."*
- **Đánh giá thẩm mỹ UI:** Khi có quá nhiều đường kẻ dọc trong một diện tích nhỏ (300px), các phần tử bị cảm giác gò bó (caged in). Khi bỏ 2 đường kẻ dọc và dùng khoảng trắng tự nhiên (whitespace) để phân tách cột, bảng Quota trở nên khoáng đạt, sang trọng và giảm độ nhiễu thị giác (visual noise).

---

## 2. BẢN VẼ GIAO DIỆN & QUY CÁCH HÌNH HỌC (VISUAL SPECIFICATION)

### 2.1. Bản Vẽ So Sánh Trước & Sau

#### TRƯỚC (Có 2 vách ngăn dọc - Rối mắt):
```text
┌──────────────────────────────────────────────────────────────────────────┐
│ ngau2hung@gmail.com                                    [✔ Dùng]  [🔄]   │
│ ──────────────────────────────────────────────────────────────────────── │
│ Quota         │         5h              │          Week                  │
│               │    (04h 32m)            │       (6d 21h 59m)             │
│ Gemini        │  [██████████]  100%     │       [████░░░░░░]  43.8%      │
│ ──────────────┼─────────────────────────┼────────────────────────────────┤
│               │    (02h 12m)            │       (2d 07h 08m)             │
│ Claude        │  [██████████]  100%     │       [██████████]  100%       │
└──────────────────────────────────────────────────────────────────────────┘
```

#### SAU (Loại bỏ 2 vách dọc - Dùng khoảng cách tự nhiên 8px, Giữ vách ngang giữa 2 model):
```text
┌──────────────────────────────────────────────────────────────────────────┐
│ ngau2hung@gmail.com                                    [✔ Dùng]  [🔄]   │
│ ──────────────────────────────────────────────────────────────────────── │
│ Quota                  5h                          Week                  │
│                    (04h 32m)                   (6d 21h 59m)              │
│ Gemini           [██████████]  100%            [████░░░░░░]  43.8%       │
│ ──────────────────────────────────────────────────────────────────────── │
│                    (02h 12m)                   (2d 07h 08m)              │
│ Claude           [██████████]  100%            [██████████]  100%        │
└──────────────────────────────────────────────────────────────────────────┘
  ◄── 48px ───►  8px  ◄────── 78px ─────────► 8px ◄────────── 86px ────────►
  (Tổng bảng Quota = 48 + 8 + 78 + 8 + 86 = 228 px | Cực kỳ thoáng đãng)
```

---

## 3. PHÂN BỔ HÌNH HỌC TOÁN HỌC (GEOMETRIC BUDGET)

- **Cột 1 (Model Name):** CỐ ĐỊNH **`48.0 px`**. Chứa "Quota", "Gemini", "Claude" (10pt Bold).
- **Khoảng cách giữa Cột 1 và Cột 2 (`colGap`):** CỐ ĐỊNH **`8.0 px`** (không còn đường kẻ 1px).
- **Cột 2 (5h Cycle):** CỐ ĐỊNH **`78.0 px`**.
  - Dòng 1: `(04h 32m)` (8.5pt).
  - Dòng 2: `[MiniBar 40px] + [Gap 4px] + [100% ~28px] = 72px`.
- **Khoảng cách giữa Cột 2 và Cột 3 (`colGap`):** CỐ ĐỊNH **`8.0 px`** (không còn đường kẻ 1px).
- **Cột 3 (Week Cycle):** CỐ ĐỊNH **`86.0 px`**.
  - Dòng 1: `(6d 21h 59m)` (8.5pt).
  - Dòng 2: `[MiniBar 40px] + [Gap 4px] + [43.8% ~32px] = 76px`.
- **Tổng bề rộng bảng Quota:** `48.0 + 8.0 + 78.0 + 8.0 + 86.0 = 228.0 px`.
- **Chiều cao bảng Quota:** Giữ nguyên `77.0 px` (gồm Header 15px, Gemini 28px, Claude 28px, vách ngang 1px + padding).

---

## 4. TOUCH-MAP KHÓA CỨNG DÀNH CHO CODER (EXACT FILE TOUCHES)

### 4.1. File `internal/ui/quota_badge.go`

1. **Trong hàm `createMatrixQuotaTableWithCells`:**
   - Xóa bỏ việc khởi tạo `sep1` và `sep2`:
     ```go
     // XÓA: sep1 := canvas.NewRectangle(palette.BorderSubtle)
     // XÓA: sep2 := canvas.NewRectangle(palette.BorderSubtle)
     ```
   - Cập nhật hàm tạo container bảng Quota chỉ nhận 4 đối tượng:
     ```go
     table := container.New(&matrixTableLayout{}, colModel, col5h, colWeek, hSep)
     ```

2. **Trong struct `matrixTableLayout`:**
   - Cập nhật `MinSize(objects []fyne.CanvasObject) fyne.Size`:
     ```go
     func (l *matrixTableLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
         const totalW = float32(48.0 + 8.0 + 78.0 + 8.0 + 86.0) // 228.0 px
         return fyne.NewSize(totalW, 77.0)
     }
     ```
   - Cập nhật `Layout(objects []fyne.CanvasObject, size fyne.Size)`:
     ```go
     func (l *matrixTableLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
         if len(objects) < 3 {
             return
         }
         colModel := objects[0]
         col5h := objects[1]
         colWeek := objects[2]

         h := size.Height
         if h < 77.0 {
             h = 77.0
         }
         const col1W = float32(48.0)    // Cột Model: 48px
         const colGap = float32(8.0)    // Khoảng cách tự nhiên giữa các cột: 8px
         const col5hW = float32(78.0)   // Cột 5h: 78px
         const colWeekW = float32(86.0) // Cột Week: 86px

         x := float32(0)
         // Col 1: Model
         colModel.Move(fyne.NewPos(x, 0))
         colModel.Resize(fyne.NewSize(col1W, h))
         x += col1W + colGap

         // Col 2: 5h
         col5h.Move(fyne.NewPos(x, 0))
         col5h.Resize(fyne.NewSize(col5hW, h))
         x += col5hW + colGap

         // Col 3: Week
         colWeek.Move(fyne.NewPos(x, 0))
         colWeek.Resize(fyne.NewSize(colWeekW, h))

         // Vách ngăn ngang (hSep) nếu có
         if len(objects) >= 4 {
             hSep := objects[3]
             hSep.Move(fyne.NewPos(0, 46.0))
             hSep.Resize(fyne.NewSize(x+colWeekW, 1.0))
         }
     }
     ```

### 4.2. File `internal/ui/ui_test.go`
- Cập nhật các test case đang kiểm tra `cont.Objects[1]` và `cont.Objects[3]` làm `sep1`, `sep2`:
  - `TestThemeColorsAcrossPalettes`: Kiểm tra `table.Objects` có 4 phần tử `[colModel, col5h, colWeek, hSep]`, kiểm tra màu viền trên `hSep`.
  - Cập nhật assertion bề rộng MinSize $\le 300\text{ px}$ (thực tế 228.0 px).
  - Đảm bảo 100% test pass (`go test -v ./internal/ui`).

---

## 5. TIÊU CHÍ NGHIỆM THU (ACCEPTANCE CRITERIA)

- [ ] **AC-1 (No Vertical Separators):** 2 vách ngăn dọc (`sep1`, `sep2`) hoàn toàn biến mất khỏi bảng Quota trong thẻ tài khoản.
- [ ] **AC-2 (Horizontal Separator Preserved):** Vách ngăn kẻ ngang `hSep` (1px, `BorderSubtle`) giữa hàng Gemini và Claude vẫn được hiển thị nguyên vẹn.
- [ ] **AC-3 (Natural Spacing):** Khoảng cách giữa các cột là `8.0 px`, các cột thẳng hàng, thoáng đãng và không bị dính chữ.
- [ ] **AC-4 (Card & Window Dimensions):** Kích thước thẻ Card giữ nguyên `300 x 130 px`, kích thước cửa sổ giữ nguyên `1008 x 630 px`.
- [ ] **AC-5 (Test Suite):** Chạy `go test -v ./internal/ui` đạt 100% PASS.
