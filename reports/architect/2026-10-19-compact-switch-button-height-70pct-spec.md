# Đặc Tả Kiến Trúc & Kỹ Thuật: Giảm Chiều Cao Nút "Chuyển" Xuống 70% (Compact Switch Button)

- **Ngày lập:** 2026-10-19
- **Vai trò:** System Architect
- **Thẩm quyền:** Chỉ đạo trực tiếp từ Human Owner: *"nut chuyển có chiều cao hoi lon so với thẻ. giam chieu cao của nút "chuyển" xuong con khoang 70%"*
- **Mã văn kiện:** `SPEC-ARCH-20261019-COMPACT-SWITCH-BUTTON-HEIGHT-70PCT`
- **Trạng thái:** CHÍNH THỨC ĐÃ ĐƯỢC HUMAN OWNER PHÊ DUYỆT (OFFICIAL APPROVED)

---

## ⚠️ ĐIỀU KHOẢN KỶ LUẬT SẮT (MANDATORY IRON RULES)

> **CHỈ THỊ TRỰC TIẾP TỪ HUMAN OWNER:**
> 1. **GIẢM CHIỀU CAO NÚT "CHUYỂN" CÒN KHOẢNG 70%:**
>    - Nút `[Chuyển]` (`switchBtn`) trong thẻ tài khoản phải được thu gọn chiều cao xuống còn khoảng **70%** (chiều cao thực tế khoảng **`22.0 px - 24.0 px`** thay vì chiều cao nút mặc định ~36px của theme Fyne).
>    - Đồng bộ chiều cao cụm `actionCluster` (bao gồm cả nút icon làm mới `[🔄]` `refreshBtn`) để hàng 1 (TopRow) có trục dọc cân đối, thanh thoát và hài hòa với nhãn Email và nhãn `[✔ Dùng]`.
> 2. **BẢO TỒN NGUYÊN VẸN HÀNH VI & TEXT:**
>    - Giữ nguyên text `"Chuyển"`, font chữ 10pt, màu sắc nổi bật `HighImportance` (`palette.Primary`).
>    - Giữ nguyên 100% logic xử lý `onSwitch(item.Email)` khi người dùng bấm nút.
> 3. **KHÓA CỨNG HÌNH HỌC TỔNG THỂ ĐÃ PHÊ DUYỆT:**
>    - Thẻ Card: CỐ ĐỊNH **`280 x 130 px`** trong `GridWrap`.
>    - Cửa sổ ứng dụng: CỐ ĐỊNH **`960 x 630 px`**.
>    - Bảng Quota: CỐ ĐỊNH **`228 x 77 px`**.
>    - In-place reactive update (Zero Jank / Zero Resizing) bảo toàn 100%.

---

## 1. PHÂN TÍCH THỊ GIÁC & HÌNH HỌC (VISUAL SPECIFICATION)

```text
┌──────────────────────────────────────────────────────────────────┐  ▲
│ ngau2hung@gmail.com (10pt)              [Chuyển (22px)] [🔄(22px)]│  │ 22-24px (Dòng 1: Thu gọn thanh thoát)
│ ──────────────────────────────────────────────────────────────── │  │  8px (Khoảng cách đệm)
│ Quota                  5h                          Week          │  │
│                    (04h 32m)                   (6d 21h 59m)      │  │
│ Gemini           [██████████]  100%            [████░░░░░░]  43% │  │ Bảng Quota
│ ──────────────────────────────────────────────────────────────── │  │ (77px)
│                    (02h 12m)                   (2d 07h 08m)      │  │
│ Claude           [██████████]  100%            [██████████] 100% │  │
└──────────────────────────────────────────────────────────────────┘  ▼ Tổng chiều cao thẻ 130px
```

- **Hiện trạng:** Nút bấm mặc định trong Fyne có padding trên/dưới lớn, đẩy chiều cao nút lên ~36px. Khi đặt cạnh dòng text Email 10pt (chiều cao chữ ~14px), nút `[Chuyển]` tạo cảm giác bị to bè, cục mịch và chiếm quá nhiều không gian theo chiều dọc của thẻ 130px.
- **Giải pháp:** Áp dụng cơ chế layout thu gọn chiều cao xuống **70%** (chiều cao mục tiêu `22px - 24px`), căn giữa theo chiều dọc với dòng Email. Cụm nút chuyển và nút làm mới sẽ trông như một pill-button thanh lịch, vừa vặn tuyệt đối với tỷ lệ thẻ 280x130px.

---

## 2. TOUCH-MAP KHÓA CỨNG DÀNH CHO CODER (EXACT FILE TOUCHES)

### 2.1. File `internal/ui/compact_button.go`
- Tinh chỉnh `compactButtonLayout` đảm bảo chiều cao tối đa của nút thu gọn là `22.0 px - 24.0 px` (hệ số `0.7` so với `MinSize().Height` tiêu chuẩn):
  ```go
  func (compactButtonLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
      size := objects[0].MinSize()
      size.Height = size.Height * 0.7
      if size.Height > 24.0 {
          size.Height = 24.0
      }
      return size
  }
  ```

### 2.2. File `internal/ui/widgets.go`
- Áp dụng `compactButton` cho cả `switchBtn` và `refreshBtn` trong `actionCluster` để hai nút bằng chiều cao nhau:
  ```go
  // Trong NewAccountCardWidgetWithQuota:
  compactRefresh := compactButton(refreshBtn)
  if item.IsActive {
      activeBadge := widget.NewLabelWithStyle("[✔ Dùng]", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true})
      actionCluster = container.NewHBox(activeBadge, compactRefresh)
  } else {
      switchBtn := widget.NewButton("Chuyển", func() {
          if onSwitch != nil {
              onSwitch(item.Email)
          }
      })
      switchBtn.Importance = widget.HighImportance
      actionCluster = container.NewHBox(compactButton(switchBtn), compactRefresh)
  }
  ```

### 2.3. File `internal/ui/ui_test.go`
- Đảm bảo các test case tìm nút `[Chuyển]` (`findSwitch`) và gọi `switchBtn.OnTapped()` vẫn hoạt động 100% chuẩn xác.
- Bổ sung test kiểm tra chiều cao của `compactButton(switchBtn)` không vượt quá `25.0 px`.
- Toàn bộ test suite `go test -v ./internal/ui` đạt 100% PASS.

---

## 3. TIÊU CHÍ NGHIỆM THU (ACCEPTANCE CRITERIA)

- [ ] **AC-1 (70% Height):** Nút `[Chuyển]` có chiều cao giảm xuống còn khoảng 70% (khoảng `22px - 24px`), không còn bị quá khổ so với thẻ.
- [ ] **AC-2 (Action Cluster Harmony):** Nút làm mới `[🔄]` được đồng bộ chiều cao tương ứng, hàng 1 của thẻ tài khoản phẳng phiu, cân đối.
- [ ] **AC-3 (Functional Integrity):** Bấm nút `[Chuyển]` vẫn kích hoạt chính xác `onSwitch(email)`.
- [ ] **AC-4 (Card & Window Dimensions):** Kích thước thẻ giữ nguyên `280 x 130 px`, kích thước cửa sổ giữ nguyên `960 x 630 px`.
- [ ] **AC-5 (Test Suite):** Chạy `go test -v ./internal/ui` đạt 100% PASS.
