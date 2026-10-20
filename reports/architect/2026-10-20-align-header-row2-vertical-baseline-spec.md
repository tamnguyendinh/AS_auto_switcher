# Đặc Tả Kiến Trúc & Kỹ Thuật: Nắn Thẳng Hàng Trục Dọc Cụm Công Tắc & Dropdown Theme (Header Row 2 Vertical Alignment)

- **Ngày lập:** 2026-10-20
- **Vai trò:** System Architect
- **Thẩm quyền:** Chỉ đạo trực tiếp từ Human Owner qua hình ảnh minh chứng `reports/problem/images/screenshot_1792479866.png`: *"gạt on/off tại sao lại nhảy len trên nhu thế? lệch hàng. - viét đặc tả gui CEO"*
- **Mã văn kiện:** `SPEC-ARCH-20261020-ALIGN-HEADER-ROW2-VERTICAL-BASELINE`
- **Trạng thái:** CHÍNH THỨC ĐÃ ĐƯỢC HUMAN OWNER PHÊ DUYỆT (OFFICIAL APPROVED)

---

## ⚠️ ĐIỀU KHOẢN KỶ LUẬT SẮT (MANDATORY IRON RULES)

> **CHỈ THỊ TRỰC TIẾP TỪ HUMAN OWNER:**
> 1. **TRIỆT TIÊU LỖI LỆCH HÀNG (ZERO VERTICAL MISALIGNMENT):**
>    - Toàn bộ các phần tử trên Hàng 2 (`row2Combined`): Cụm bên trái (`serverStatusLabel` + 3 nút Toolbar) và Cụm bên phải (`lblAuto` + `autoSwitchToggle` + `themeSelectWrap`) phải **nằm trên cùng một đường tâm ngang (Vertical Centerline Baseline)**.
>    - Tuyệt đối cấm để công tắc hoặc dropdown theme bị kéo dính sát nóc trần ($y = 0$) trong khi cụm bên trái nằm ở tâm hàng.
> 2. **ĐỒNG NHẤT WIDGET TYPE CHO TEXT LABELS:**
>    - Đổi nhãn `"Tự động:"` từ `*canvas.Text` sang `*widget.Label` (`widget.NewLabelWithStyle("Tự động:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})`) để sử dụng cùng hệ quy chiếu padding và đường chân chữ (text baseline) với `a.serverStatusLabel`.
> 3. **BẢO TỒN NGUYÊN VẸN CÁC THÔNG SỐ ĐÃ KHÓA CỨNG:**
>    - Cửa sổ ứng dụng: CỐ ĐIH **`920 x 630 px`**.
>    - Thẻ Card: CỐ ĐỊNH **`280 x 130 px`** trong `newSymmetricGridWrap`.
>    - Bảng Quota: CỐ ĐỊNH **`228 x 77 px`**.
>    - Logic hoạt động toggle On/Off và Theme Select giữ nguyên 100%.

---

## 1. PHÂN TÍCH NGUYÊN NHÂN GỐC RỄ & BẢN VẼ HÌNH HỌC

### 1.1. Bản Vẽ So Sánh

#### TRƯỚC (Cụm phải bị treo lơ lửng sát trần, lệch hàng nghiêm trọng):
```text
┌────────────────────────────────────────────────────────────────────────────────────────┐
│                                                                        Tự động: [ ○ ] [☀️▼] (Bị hút lên y=0)
│ Hoạt động | PID 5284 | Port 49952   [Danh sách...] [Làm mới...] [📌]                   (Nằm ở tâm y=8)
├────────────────────────────────────────────────────────────────────────────────────────┤
```

#### SAU (Tất cả thành phần thẳng tắp theo trục tâm hàng - Cân đối tuyệt đối):
```text
┌────────────────────────────────────────────────────────────────────────────────────────┐
│ Hoạt động | PID 5284 | Port 49952   [Danh sách...] [Làm mới...] [📌]   Tự động: [ ○ ] [☀️▼]   │ ◄── Tất cả cùng nằm
├────────────────────────────────────────────────────────────────────────────────────────┤     ở tâm trục dọc y_center
```

### 1.2. Nguyên Nhân Gốc Rễ Đã Được Xác Minh
1. Trong `autoSwitchToggleRenderer.Layout(size)`: Coder đã hardcode `r.toggle.track.Move(fyne.NewPos(0, 0))`. Khi chiều cao hàng là ~28px - 32px, công tắc vẫn bị ép dính cứng ở $y = 0$.
2. `lblAuto` dùng `canvas.Text` trần không có padding của theme, bị đặt ở $y = 0$.
3. `HBox` mặc định trong Fyne đặt các phần tử con bắt đầu từ $y = 0$, không tự động căn giữa dọc khi các phần tử có chiều cao khác nhau.

---

## 2. THIẾT KẾ KỸ THUẬT & GIẢI PHÁP NẮN THẲNG TRỤC DỌC

### 2.1. Chuẩn Hóa Căn Giữa Dọc Trong `AutoSwitchToggle`
Trong `internal/ui/window.go`, cập nhật `autoSwitchToggleRenderer`:
```go
func (r *autoSwitchToggleRenderer) Layout(size fyne.Size) {
    trackW := float32(23.0)
    trackH := float32(12.0)
    
    // Căn giữa theo chiều cao thực tế được phân bổ cho widget
    y := (size.Height - trackH) / 2.0
    if y < 0 {
        y = 0
    }
    
    r.toggle.track.Move(fyne.NewPos(0, y))
    r.toggle.track.Resize(fyne.NewSize(trackW, trackH))

    thumbY := y + 2.0
    if r.toggle.isOn {
        r.toggle.thumb.Move(fyne.NewPos(13, thumbY))
    } else {
        r.toggle.thumb.Move(fyne.NewPos(2, thumbY))
    }
    r.toggle.thumb.Resize(fyne.NewSize(8, 8))
}
```

### 2.2. Chuẩn Hóa Nhãn `"Tự động:"`
Trong `setupLayout()`:
```go
// Thay vì canvas.NewText:
lblAuto := widget.NewLabelWithStyle("Tự động:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
```

### 2.3. Tạo Layout Căn Giữa Dọc Cho Cụm Góc Phải (`row2RightClusterLayout`)
Xây dựng một custom layout nhẹ hoặc sử dụng container đảm bảo mọi phần tử trong `rightCluster` đều được định vị $y = (\text{rowHeight} - \text{itemHeight}) / 2$:
```go
type row2RightClusterLayout struct{}

func (row2RightClusterLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
    w, h := float32(0), float32(0)
    padding := theme.Padding()
    for i, o := range objects {
        if !o.Visible() { continue }
        ms := o.MinSize()
        w += ms.Width
        if i > 0 { w += padding }
        if ms.Height > h { h = ms.Height }
    }
    return fyne.NewSize(w, h)
}

func (row2RightClusterLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
    x := float32(0)
    padding := theme.Padding()
    for _, o := range objects {
        if !o.Visible() { continue }
        ms := o.MinSize()
        y := (size.Height - ms.Height) / 2.0
        if y < 0 { y = 0 }
        o.Move(fyne.NewPos(x, y))
        o.Resize(ms)
        x += ms.Width + padding
    }
}
```
Khởi tạo cụm phải:
```go
rightCluster := container.New(row2RightClusterLayout{}, lblAuto, a.autoSwitchToggle, themeSelectWrap)
```

---

## 3. TOUCH-MAP KHÓA CỨNG DÀNH CHO CODER (EXACT FILE TOUCHES)

### 3.1. File `internal/ui/window.go`
1. Thay đổi khởi tạo `lblAuto` từ `canvas.NewText` sang `widget.NewLabelWithStyle`.
2. Sửa `autoSwitchToggleRenderer.Layout` tính toán $y = (\text{size.Height} - 12.0) / 2.0$.
3. Khai báo `row2RightClusterLayout` và dùng để bọc `rightCluster := container.New(row2RightClusterLayout{}, lblAuto, a.autoSwitchToggle, themeSelectWrap)`.

### 3.2. File `internal/ui/ui_test.go`
- Viết Unit Test `TestHeaderRow2_VerticalAlignment`:
  - Lấy tâm trục dọc của `serverStatusLabel`: $Y_{\text{left}} = \text{pos.Y} + \text{height}/2$.
  - Lấy tâm trục dọc của `lblAuto` và `autoSwitchToggle`: $Y_{\text{right}} = \text{pos.Y} + \text{height}/2$.
  - Khẳng định độ lệch $|Y_{\text{left}} - Y_{\text{right}}| \le 2.0\text{ px}$.
- Chạy `go test -v ./internal/ui` đạt 100% PASS.

---

## 4. TIÊU CHÍ NGHIỆM THU (ACCEPTANCE CRITERIA)

- [ ] **AC-1 (Zero Misalignment):** Cụm `"Tự động:"`, công tắc gạt và nút Theme nằm thẳng hàng tâm trục dọc với cụm `Hoạt động | Port ...` và 3 nút Toolbar bên trái (độ lệch $\le 2.0\text{ px}$).
- [ ] **AC-2 (Label Theme Baseline):** Nhãn `"Tự động:"` sử dụng `widget.Label`, có cùng phong cách hiển thị và đường chân chữ với `serverStatusLabel`.
- [ ] **AC-3 (Dynamic Center):** Công tắc `AutoSwitchToggle` tự động định vị ở tâm chiều cao của hàng, không bao giờ bị dính sát mép trên $y = 0$.
- [ ] **AC-4 (Dimensions Stability):** Cửa sổ giữ nguyên `920 x 630 px`, thẻ Card giữ nguyên `280 x 130 px`.
- [ ] **AC-5 (Test Suite):** Chạy `go test -v ./internal/ui` đạt 100% PASS.
