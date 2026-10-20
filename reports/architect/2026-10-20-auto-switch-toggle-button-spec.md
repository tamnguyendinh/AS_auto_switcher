# Đặc Tả Kiến Trúc & Kỹ Thuật: Nút Công Tắc Gạt "Tự Động Đổi Tài Khoản: On/Off" Trên UI Tổng (Capsule Toggle Switch)

- **Ngày lập:** 2026-10-20
- **Vai trò:** System Architect
- **Thẩm quyền:** Chỉ đạo trực tiếp từ Human Owner: *"phong cach 1 , viet dạ tả gui CEO đi., day là code cai nut tren UI chu k opahi code chcu angn nhé. chuc angnw ben duoi hạot dong the nào thì vàn còn dang thao luận"*
- **Mã văn kiện:** `SPEC-ARCH-20261020-AUTO-SWITCH-TOGGLE-BUTTON`
- **Trạng thái:** CHÍNH THỨC ĐÃ ĐƯỢC HUMAN OWNER PHÊ DUYỆT (OFFICIAL APPROVED)

---

## ⚠️ ĐIỀU KHOẢN KỶ LUẬT SẮT (MANDATORY IRON RULES)

> **CHỈ THỊ TRỰC TIẾP TỪ HUMAN OWNER:**
> 1. **GIỚI HẠN TUYỆT ĐỐI Ở PHẠM VI GIAO DIỆN (UI ONLY - NO BACKEND LOGIC):**
>    - Nhiệm vụ này **CHỈ TẠO PHẦN TỬ NÚT BẤM TRÊN GIAO DIỆN (WIDGET)** và xử lý chuyển đổi trạng thái hiển thị (State Toggle & Visual Feedback khi click).
>    - **TUYỆT ĐỐI CẤM** tự ý lập trình bất kỳ logic nghiệp vụ tự động đổi tài khoản ngầm nào ở backend (chức năng nghiệp vụ sẽ được thảo luận và thiết kế riêng sau).
> 2. **PHONG CÁCH THIẾT KẾ: CÔNG TẮC GẠT DẠNG VIÊN THUỐC (PHONG CÁCH 1 - CAPSULE TOGGLE SWITCH):**
>    - Thanh trượt hình viên thuốc bo tròn (Pill Capsule), có nút tròn trượt qua lại:
>      - **Trạng thái OFF (Tắt):** Nền ray màu xám mờ (`palette.BorderSubtle` / `TextMuted`), nút tròn nằm bên trái.
>      - **Trạng thái ON (Bật):** Nền ray màu xanh dương thương hiệu (`palette.Primary`), nút tròn nằm bên phải.
>    - Kèm nhãn chữ: `"Tự động:"` hoặc `"Tự động đổi"` (10pt).
> 3. **VỊ TRÍ ĐẶT TRÊN HEADER:**
>    - Đặt ở Hàng 2 gộp (`row2Combined`) bên góc phải, nằm ngay cạnh trước nút đổi Theme (`themeSelectWrap`).
> 4. **BẢO TỒN NGUYÊN VẸN CÁC THÔNG SỐ ĐÃ KHÓA CỨNG:**
>    - Cửa sổ ứng dụng: CỐ ĐỊNH **`920 x 630 px`**.
>    - Thẻ Card: CỐ ĐỊNH **`280 x 130 px`** trong `newSymmetricGridWrap`.
>    - Bảng Quota: CỐ ĐỊNH **`228 x 77 px`**.
>    - In-place reactive update (Zero Jank / Zero Resizing) bảo toàn 100%.

---

## 1. BẢN VẼ THỊ GIÁC & QUY CÁCH HÌNH HỌC (VISUAL SPECIFICATION)

### 1.1. Bản Vẽ Bố Cục Hàng 2 Tổng Thể
```text
┌────────────────────────────────────────────────────────────────────────────────────────┐
│ [AS] Antigravity Multi-Account Switcher (10pt Bold)                            _ |_| X │ ◄── Hàng 1
│ Hoạt động | PID 18660 | Port 62220  [Danh sách...] [Làm mới...] [📌]   Tự động [ ○ ] [ ☀️ ▼ ] │ ◄── Hàng 2
├────────────────────────────────────────────────────────────────────────────────────────┤
  ◄─────────────────── Cụm bên trái (~512px) ─────────────────►          ◄─ Cụm phải (~170px) ─►
```

### 1.2. Phóng To Nút Công Tắc Gạt (Pill Capsule Switch)

#### Trạng thái OFF (Mặc định khi mở):
```text
   Tự động: [ ○       ]
            ◄── 38px ──►
   - Nền ray: Màu xám mờ (BorderSubtle / TextMuted)
   - Nút tròn: Màu trắng/xám sáng nằm sát lề trái
```

#### Trạng thái ON (Khi click vào):
```text
   Tự động: [       ● ]
            ◄── 38px ──►
   - Nền ray: Màu xanh dương (palette.Primary)
   - Nút tròn: Màu trắng nổi bật nằm sát lề phải
```

---

## 2. QUY CÁCH HÌNH HỌC TOÁN HỌC (GEOMETRIC SPECIFICATION)

1. **Kích thước công tắc gạt (`capsuleToggle`):**
   - Chiều rộng ray: **`38.0 px`**.
   - Chiều cao ray: **`20.0 px`**.
   - Bo góc ray: **`CornerRadius = 10.0 px`** (bo tròn dạng viên thuốc hoàn hảo).
   - Nút tròn bên trong (Thumb knob): Đường kính **`14.0 px`**, `CornerRadius = 7.0 px`.
   - Vị trí nút tròn:
     - Khi `OFF`: Tọa độ $x = 3.0\text{ px}$, $y = 3.0\text{ px}$.
     - Khi `ON`: Tọa độ $x = 21.0\text{ px}$, $y = 3.0\text{ px}$.
2. **Nhãn chữ đi kèm (`toggleLabel`):**
   - Text: `"Tự động:"` (10pt Bold, màu `palette.Text`).
3. **Tổng cụm góc phải (`rightCluster`):**
   - `rightCluster := container.NewHBox(autoSwitchToggle, themeSelectWrap)`
   - Bề rộng: Nhãn (~65px) + Công tắc (38px) + Khoảng cách (8px) + Theme Select (58px) $\approx \mathbf{170.0\text{ px}}$.
4. **Phân bổ trên cửa sổ 920px:**
   - Cụm trái: $512.0\text{ px}$.
   - Cụm phải: $170.0\text{ px}$.
   - Khoảng trống tự nhiên ở giữa: $920.0 - 512.0 - 170.0 = \mathbf{238.0\text{ px}}$ $\rightarrow$ Vô cùng rộng rãi, thoáng đãng và sang trọng.

---

## 3. THIẾT KẾ THÀNH PHẦN KỸ THUẬT (COMPONENT SPECIFICATION)

### 3.1. Struct Widget `AutoSwitchToggle`
Trong file `internal/ui/window.go` (hoặc `auto_switch_toggle.go`):
```go
type AutoSwitchToggle struct {
    widget.BaseWidget
    isOn     bool
    onChange func(bool)
    palette  ThemePalette

    track *canvas.Rectangle
    thumb *canvas.Circle // hoặc canvas.Rectangle với CornerRadius lớn
}
```

- **Hành vi tương tác:**
  - Implement `fyne.Tappable`:
    ```go
    func (t *AutoSwitchToggle) Tapped(_ *fyne.PointEvent) {
        t.isOn = !t.isOn
        t.updateVisuals()
        if t.onChange != nil {
            t.onChange(t.isOn)
        }
    }
    ```
  - Implement `desktop.Cursorable`: `Cursor() desktop.Cursor` trả về `desktop.PointerCursor` (hình bàn tay).
- **Hàm `updateVisuals()`:**
  - Khi `isOn == true`: `t.track.FillColor = t.palette.Primary`, `t.thumb.Move(fyne.NewPos(21, 3))`.
  - Khi `isOn == false`: `t.track.FillColor = t.palette.BorderSubtle`, `t.thumb.Move(fyne.NewPos(3, 3))`.
  - Gọi `Refresh()` cục bộ.

### 3.2. Quản Lý Trạng Thái Trong `App`
- Thêm trường `autoSwitchEnabled bool` vào `struct App` (`internal/ui/app.go`).
- Mặc định khởi tạo `false`.
- Khi người dùng click: Chỉ cập nhật biến `a.autoSwitchEnabled = isOn` (chưa gọi bất kỳ tác vụ chuyển tài khoản nào).

---

## 4. TOUCH-MAP KHÓA CỨNG DÀNH CHO CODER (EXACT FILE TOUCHES)

### 4.1. File `internal/ui/window.go`
1. Xây dựng struct và hàm khởi tạo `newAutoSwitchToggle(initial bool, onChange func(bool), palette ThemePalette)`.
2. Tạo nhãn `"Tự động:"` và bọc cùng công tắc:
   ```go
   lblAuto := canvas.NewText("Tự động:", palette.Text)
   lblAuto.TextSize = BodyTextSize
   lblAuto.TextStyle = fyne.TextStyle{Bold: true}
   autoCluster := container.NewHBox(lblAuto, a.autoSwitchToggle)
   ```
3. Ghép vào cụm góc phải của Hàng 2:
   ```go
   rightCluster := container.NewHBox(autoCluster, themeSelectWrap)
   row2Combined := container.NewBorder(nil, nil, row2LeftCluster, rightCluster)
   ```

### 4.2. File `internal/ui/app.go`
- Bổ sung trường `autoSwitchEnabled bool` và con trỏ `autoSwitchToggle fyne.CanvasObject` vào `struct App`.

### 4.3. File `internal/ui/ui_test.go`
- Viết Unit Test kiểm tra:
  - Công tắc `autoSwitchToggle` hiển thị trên UI.
  - Khi click (tap) vào công tắc: Trạng thái `isOn` đổi từ `false` sang `true` và ngược lại.
  - Vị trí con trượt `thumb` và màu nền `track` cập nhật tương ứng.
  - Toàn bộ test suite `go test -v ./internal/ui` đạt 100% PASS.

---

## 5. TIÊU CHÍ NGHIỆM THU (ACCEPTANCE CRITERIA)

- [ ] **AC-1 (UI Toggle Only):** Công tắc gạt dạng viên thuốc (Capsule Pill Toggle) xuất hiện bên cạnh nút đổi Theme với nhãn `"Tự động:"`.
- [ ] **AC-2 (Visual Toggle Interaction):** Click vào công tắc làm trượt nút tròn qua lại (OFF $\leftrightarrow$ ON), đổi màu nền ray (Xám $\leftrightarrow$ Xanh `Primary`).
- [ ] **AC-3 (Pointer Cursor):** Con trỏ chuột chuyển thành hình bàn tay (`desktop.PointerCursor`) khi rê chuột vào công tắc.
- [ ] **AC-4 (No Backend Side Effects):** Bấm công tắc không kích hoạt bất kỳ luồng chuyển tài khoản nào.
- [ ] **AC-5 (Geometry & Test Suite):** Cửa sổ giữ nguyên `920 x 630 px`, thẻ Card `280 x 130 px`; `go test -v ./internal/ui` đạt 100% PASS.
