# Đặc Tả Kiến Trúc & Kỹ Thuật: Chuyển Nút "Chuyển" Sang Dạng Text In Đậm Màu Xanh Dương ([Chuyển]) Đồng Bộ Với [✔ Dùng]

- **Ngày lập:** 2026-10-19
- **Vai trò:** System Architect
- **Thẩm quyền:** Chỉ đạo trực tiếp từ Human Owner: *"nếu nút "chuyển" cung giống [Dùng] tức là nó như thế này [Chuyển] với Text in đậm màu xanh dương thay vì là 1 nút thô như hiện tại, thì nhìn sẽ sang hơn nhiều. viet dạ tả đi"*
- **Mã văn kiện:** `SPEC-ARCH-20261019-FLAT-TEXT-SWITCH-BUTTON`
- **Trạng thái:** CHÍNH THỨC ĐÃ ĐƯỢC HUMAN OWNER PHÊ DUYỆT (OFFICIAL APPROVED)

---

## ⚠️ ĐIỀU KHOẢN KỶ LUẬT SẮT (MANDATORY IRON RULES)

> **CHỈ THỊ TRỰC TIẾP TỪ HUMAN OWNER:**
> 1. **LOẠI BỎ KHỐI HỘP NỀN XANH ĐẶC (NO SOLID BLOCK BUTTON):**
>    - Xóa bỏ hoàn toàn dạng nút bấm khối hộp nền xanh đặc dày cộm của `widget.HighImportance`.
>    - Thay thế bằng **Clickable Text Badge/Button `[Chuyển]`**: Text in đậm (10pt Bold), màu xanh dương thương hiệu (`palette.Primary`), không có viền hộp hay nền đặc lúc bình thường.
> 2. **ĐỒNG BỘ PHONG CÁCH VỚI NHÃN `[✔ Dùng]`:**
>    - Thẻ Đang dùng (Active): Hiển thị nhãn `[✔ Dùng]` (10pt Bold, text muted/secondary).
>    - Thẻ Chưa dùng (Inactive): Hiển thị nút chữ `[Chuyển]` (10pt Bold, màu xanh `palette.Primary`).
>    - Cả hai trạng thái có cùng chiều cao, cùng độ dày chữ và cùng ngôn ngữ thiết kế Typography-driven phẳng phiu, thanh lịch.
> 3. **TRẢI NGHIỆM TƯƠNG TÁC (AFFORDANCE & INTERACTION):**
>    - Nút `[Chuyển]` phải hỗ trợ sự kiện chuột: Khi rê chuột (Hover), con trỏ chuyển thành **Bàn tay trỏ (`desktop.PointerCursor`)** và có phản hồi thị giác nhẹ.
>    - Khi bấm (Tapped / Clicked): Kích hoạt lập tức `onSwitch(item.Email)` 100% như cũ.
> 4. **BẢO TỒN NGUYÊN VẸN CÁC THÔNG SỐ ĐÃ KHÓA CỨNG:**
>    - Thẻ Card: CỐ ĐỊNH **`280 x 130 px`** trong `GridWrap`.
>    - Cửa sổ ứng dụng: CỐ ĐỊNH **`960 x 630 px`**.
>    - Bảng Quota 2 tầng: CỐ ĐỊNH **`228 x 77 px`**.
>    - In-place reactive update (Zero Jank / Zero Resizing) bảo toàn 100%.

---

## 1. PHÂN TÍCH THỊ GIÁC & BẢN VẼ GIAO DIỆN

### 1.1. Bản Vẽ So Sánh Dòng 1 (TopRow)

#### TRƯỚC (Nút khối hộp thô - Rối mắt và chiếm diện tích):
```text
┌──────────────────────────────────────────────────────────────────┐
│ ngau2hung@gmail.com                     [ Chuyển (Khối Xanh) ] [🔄]│
└──────────────────────────────────────────────────────────────────┘
```

#### SAU (Text phẳng in đậm màu xanh dương - Đẳng cấp, thanh thoát, sang trọng):
```text
┌──────────────────────────────────────────────────────────────────┐
│ quantritaikhoanemail@gmail.com                 [✔ Dùng]       [🔄]│  ◄── Thẻ Active
├──────────────────────────────────────────────────────────────────┤
│ ngau2hung@gmail.com                            [Chuyển]       [🔄]│  ◄── Thẻ Inactive (Chữ xanh dương in đậm)
└──────────────────────────────────────────────────────────────────┘
```

### 1.2. Lợi Ích Vượt Trội:
- **Thẩm mỹ Minimalist cao cấp:** Loại bỏ được "rừng nút xanh" khi hiển thị 6-10 tài khoản. Giao diện trở nên tinh giản, hướng sự tập trung của mắt người dùng vào thông tin quan trọng nhất: Email và Quota.
- **Tối ưu chiều cao:** Dòng 1 co về chiều cao tự nhiên của text (~16px), tạo ra khoảng thở rộng rãi hơn cho thẻ 130px.

---

## 2. THIẾT KẾ THÀNH PHẦN KỸ THUẬT (COMPONENT SPECIFICATION)

### 2.1. Cấu Trúc Widget `switchTextButton`
Tạo một custom widget nhẹ (hoặc button phẳng) đại diện cho `[Chuyển]`:
```go
type switchTextButton struct {
    widget.BaseWidget
    Text     string
    OnTapped func()
    hovered  bool
    palette  ThemePalette
}
```
- **Renderer / Visual State:**
  - Bình thường: Vẽ text `"[Chuyển]"` với `TextSize: 10`, `TextStyle: fyne.TextStyle{Bold: true}`, màu sắc: `palette.Primary`. Nền trong suốt hoàn toàn.
  - Khi Hover (`Cursor() desktop.Cursor` trả về `desktop.PointerCursor`): Chữ `"[Chuyển]"` có màu sáng hơn nhẹ (`palette.PrimaryHover` hoặc giảm alpha) hoặc có nền bo góc siêu mờ để người dùng cảm nhận được phản hồi xúc giác.
- **Tương tác:**
  - Implement `fyne.Tappable`: `Tapped(*fyne.PointEvent)` gọi `OnTapped()`.
  - Implement `desktop.Hoverable`: `MouseIn`, `MouseOut` để cập nhật trạng thái `hovered` và `Refresh()`.
  - Implement `desktop.Cursorable`: `Cursor() desktop.Cursor` trả về `desktop.PointerCursor`.

---

## 3. TOUCH-MAP KHÓA CỨNG DÀNH CHO CODER (EXACT FILE TOUCHES)

### 3.1. File `internal/ui/widgets.go`
- **Xây dựng `newSwitchTextButton(onSwitch func()) fyne.CanvasObject`:**
  - Khởi tạo nút text `"[Chuyển]"` in đậm màu `palette.Primary`.
  - Hỗ trợ đầy đủ `Tapped`, `Hovered`, `PointerCursor`.
- **Trong `NewAccountCardWidgetWithQuota`:**
  - Thay thế khối tạo `switchBtn := widget.NewButton("Chuyển", ...)` bằng `newSwitchTextButton`.
  - Cụm `actionCluster` cho thẻ Inactive trở thành:
    ```go
    switchBtn := newSwitchTextButton(func() {
        if onSwitch != nil {
            onSwitch(item.Email)
        }
    }, palette)
    actionCluster = container.NewHBox(switchBtn, refreshBtn)
    ```

### 3.2. File `internal/ui/ui_test.go`
- **Cập nhật hàm tìm nút chuyển đổi `findSwitch` trong test suite:**
  - Mở rộng hàm tìm kiếm để phát hiện cả `switchTextButton` hoặc kiểm tra đối tượng có text `"Chuyển"` / `"[Chuyển]"` và hỗ trợ `Tappable`.
  - Kích hoạt `OnTapped()` (hoặc `Tapped(&fyne.PointEvent{})`) và kiểm tra biến `switchedEmail` nhận đúng email tài khoản.
- Chạy `go test -v ./internal/ui` đạt 100% PASS.

---

## 4. TIÊU CHÍ NGHIỆM THU (ACCEPTANCE CRITERIA)

- [ ] **AC-1 (Flat Text Style):** Nút `[Chuyển]` không còn nền khối hộp màu xanh đặc, hiển thị dưới dạng chữ `[Chuyển]` in đậm màu xanh dương (`palette.Primary`), phẳng phiu tương đồng với `[✔ Dùng]`.
- [ ] **AC-2 (Pointer Cursor & Hover):** Khi rê chuột vào `[Chuyển]`, con trỏ chuột đổi thành hình bàn tay chỉ trỏ (`desktop.PointerCursor`) và có phản hồi thị giác nhẹ.
- [ ] **AC-3 (Switch Functionality):** Click chuột vào `[Chuyển]` kích hoạt chuyển tài khoản `onSwitch(email)` mượt mà, chính xác 100%.
- [ ] **AC-4 (Dimensions Stability):** Giữ nguyên thẻ Card `280 x 130 px`, cửa sổ `960 x 630 px`, bảng Quota `228 x 77 px`.
- [ ] **AC-5 (Test Suite):** Chạy `go test -v ./internal/ui` đạt 100% PASS.
