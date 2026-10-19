# Đặc Tả Kiến Trúc & Kỹ Thuật: Tinh Giản Nút Ghim Cửa Sổ Thành Icon-Only [ 📌 ]

- **Ngày ban hành:** 2026-10-19
- **Vai trò:** System Architect
- **Thẩm quyền:** Chỉ đạo trực tiếp từ Human Owner
- **Mã văn kiện:** `SPEC-ARCH-20261019-PIN-BUTTON-ICON-ONLY`
- **Mục tiêu:** Tinh giản nút ghim cửa sổ (Always-on-Top) trên Toolbar Row 3 từ dạng nhãn text dài `[ 📌 Ghim : Bật/Tắt ]` thành dạng icon duy nhất `[ 📌 ]`. Giúp thanh công cụ cực kỳ gọn gàng, hiện đại và tiết kiệm không gian ngang.

---

## 1. BỐI CẢNH & YÊU CẦU CỦA HUMAN OWNER

- **Hiện trạng:**
  - Hàng 3 (Toolbar) hiện có 3 nút: `[ Danh sách tài khoản ]`, `[ Làm mới Quota ]`, `[ 📌 Ghim : Bật ]` (hoặc `[ 📌 Ghim : Tắt ]`).
  - Nút Ghim hiện đang hiển thị cả cụm chữ tiếng Việt, chiếm chiều ngang > 100px.
- **Yêu cầu từ Human Owner:**
  - Xóa bỏ hoàn toàn dòng text ` Ghim : Bật` / ` Ghim : Tắt`, chỉ giữ lại duy nhất icon `[ 📌 ]`.
  - Giữ nguyên cơ chế màu sáng / chìm đã có sẵn trong mã nguồn (`widget.HighImportance` khi đang bật và `widget.MediumImportance` khi đang tắt).

---

## 2. NGUYÊN TẮC THIẾT KẾ & BẢO TOÀN KIẾN TRÚC

1. **Bảo toàn chức năng (Zero Functional Regression):**
   - Click vào nút vẫn gọi `a.ToggleAlwaysOnTop()`.
   - Vẫn đồng bộ gọi `setWindowAlwaysOnTop(windowTitle, enabled)`.
2. **Bảo toàn trạng thái thị giác (Visual State Parity):**
   - **Khi BẬT (Always-on-Top ON):** Nhãn nút là `"📌"`, `Importance = widget.HighImportance` (nút sáng màu nhấn primary).
   - **Khi TẮT (Always-on-Top OFF):** Nhãn nút là `"📌"`, `Importance = widget.MediumImportance` (nút chìm/mờ).
3. **Phạm vi sửa đổi tối thiểu (Minimal Surgical Scope):**
   - Tuyệt đối không thay đổi layout cấu trúc của Toolbar.
   - Chỉ thay đổi chuỗi ký tự hiển thị và cập nhật kiểm thử tự động tương ứng.

---

## 3. TOUCH-MAP CHI TIẾT DÀNH CHO CODER

### 3.1. `internal/ui/window.go`
- **Vị trí:** Khối khởi tạo Toolbar Row 3 (hàm `setupLayout`, khoảng dòng 158-168).
- **Hành động:**
  - Xóa bỏ logic rẽ nhánh gán biến `pinLabel`:
    ```go
    // CŨ:
    pinLabel := "📌 Ghim : Bật"
    pinImportance := widget.HighImportance
    if !a.alwaysOnTop {
        pinLabel = "📌 Ghim : Tắt"
        pinImportance = widget.MediumImportance
    }
    a.alwaysOnTopBtn = widget.NewButton(pinLabel, func() {
        a.ToggleAlwaysOnTop()
    })
    a.alwaysOnTopBtn.Importance = pinImportance
    ```
  - Thay bằng khởi tạo nhãn cố định `"📌"`:
    ```go
    // MỚI:
    pinImportance := widget.HighImportance
    if !a.alwaysOnTop {
        pinImportance = widget.MediumImportance
    }
    a.alwaysOnTopBtn = widget.NewButton("📌", func() {
        a.ToggleAlwaysOnTop()
    })
    a.alwaysOnTopBtn.Importance = pinImportance
    ```

---

### 3.2. `internal/ui/app.go`
- **Vị trí:** Hàm `syncAlwaysOnTopBtn(enabled bool)` (khoảng dòng 609-620).
- **Hành động:**
  - Luôn đặt `SetText("📌")` cho nút ghim, chỉ cập nhật thuộc tính `Importance` theo trạng thái `enabled`:
    ```go
    // CŨ:
    func (a *App) syncAlwaysOnTopBtn(enabled bool) {
        if a.alwaysOnTopBtn == nil {
            return
        }
        if enabled {
            a.alwaysOnTopBtn.SetText("📌 Ghim : Bật")
            a.alwaysOnTopBtn.Importance = widget.HighImportance
        } else {
            a.alwaysOnTopBtn.SetText("📌 Ghim : Tắt")
            a.alwaysOnTopBtn.Importance = widget.MediumImportance
        }
        a.alwaysOnTopBtn.Refresh()
    }

    // MỚI:
    func (a *App) syncAlwaysOnTopBtn(enabled bool) {
        if a.alwaysOnTopBtn == nil {
            return
        }
        a.alwaysOnTopBtn.SetText("📌")
        if enabled {
            a.alwaysOnTopBtn.Importance = widget.HighImportance
        } else {
            a.alwaysOnTopBtn.Importance = widget.MediumImportance
        }
        a.alwaysOnTopBtn.Refresh()
    }
    ```

---

### 3.3. `internal/ui/ui_test.go`
- **Vị trí:** Hàm kiểm thử `TestAlwaysOnTop(t *testing.T)` (khoảng dòng 190-210).
- **Hành động:**
  - Cập nhật các câu lệnh assert kiểm tra text từ `"📌 Ghim : Tắt"` và `"📌 Ghim : Bật"` thành `"📌"`.
  - Bổ sung assert kiểm tra `Importance` tương ứng (`widget.MediumImportance` khi Tắt, `widget.HighImportance` khi Bật) để đảm bảo trạng thái visual được kiểm chứng chặt chẽ:
    ```go
    app.ToggleAlwaysOnTop()
    if app.IsAlwaysOnTop() {
        t.Errorf("expected alwaysOnTop to be false after toggle")
    }
    if app.alwaysOnTopBtn.Text != "📌" {
        t.Errorf("expected button text '📌', got: %s", app.alwaysOnTopBtn.Text)
    }
    if app.alwaysOnTopBtn.Importance != widget.MediumImportance {
        t.Errorf("expected button importance MediumImportance, got: %v", app.alwaysOnTopBtn.Importance)
    }

    app.SetAlwaysOnTop(true)
    if !app.IsAlwaysOnTop() {
        t.Errorf("expected alwaysOnTop to be true after SetAlwaysOnTop(true)")
    }
    if app.alwaysOnTopBtn.Text != "📌" {
        t.Errorf("expected button text '📌', got: %s", app.alwaysOnTopBtn.Text)
    }
    if app.alwaysOnTopBtn.Importance != widget.HighImportance {
        t.Errorf("expected button importance HighImportance, got: %v", app.alwaysOnTopBtn.Importance)
    }
    ```

---

## 4. TIÊU CHÍ NGHIỆM THU (ACCEPTANCE CRITERIA)

1. **AC-1 (UI String):** Trên giao diện, nút ghim chỉ hiển thị duy nhất icon `📌`, không còn bất kỳ chữ `Ghim`, `Bật`, `Tắt` nào.
2. **AC-2 (Toggle & Importance):** 
   - Khi ở trạng thái Ghim Bật: Nút có `Importance == widget.HighImportance`.
   - Khi ở trạng thái Ghim Tắt: Nút có `Importance == widget.MediumImportance`.
   - Chức năng ghim cửa sổ trên Windows vẫn hoạt động 100% chuẩn xác.
3. **AC-3 (Automated Tests):** Chạy lệnh kiểm thử sau phải vượt qua 100% không có lỗi:
   ```bash
   go test -v -run TestAlwaysOnTop ./internal/ui
   ```
