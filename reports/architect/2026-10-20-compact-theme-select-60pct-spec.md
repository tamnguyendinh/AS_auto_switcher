# Đặc Tả Kiến Trúc & Kỹ Thuật: Thu Nhỏ Dropdown Chọn Theme Xuống 60% (Compact Theme Selector)

- **Ngày lập:** 2026-10-20
- **Vai trò:** System Architect
- **Thẩm quyền:** Chỉ đạo trực tiếp từ Human Owner: *"dọ cao + độ rọng của dopdow theme qua lon so với kich thucoj của text xong quanh, can nho lại con khoảng 60%"*
- **Mã văn kiện:** `SPEC-ARCH-20261020-COMPACT-THEME-SELECT-60PCT`
- **Trạng thái:** CHÍNH THỨC ĐÃ ĐƯỢC HUMAN OWNER PHÊ DUYỆT (OFFICIAL APPROVED)

---

## ⚠️ ĐIỀU KHOẢN KỶ LUẬT SẮT (MANDATORY IRON RULES)

> **CHỈ THỊ TRỰC TIẾP TỪ HUMAN OWNER:**
> 1. **THU NHỎ CẢ ĐỘ CAO VÀ ĐỘ RỘNG CỦA DROPDOWN THEME VỀ ~60%:**
>    - Kích thước hiện tại của `themeSelectWrap` (`58 x 32 px`) quá to bè so với cỡ chữ 10pt của các nhãn xung quanh.
>    - Thu nhỏ cả 2 chiều về khoảng **60%**:
>      - **Chiều rộng (Width):** Giảm từ `58 px` xuống **`36.0 px`** (hoặc `38.0 px`).
>      - **Chiều cao (Height):** Giảm từ `32 px` xuống **`20.0 px`** ($32 \times 0.6 = 19.2 \approx 20\text{ px}$).
> 2. **ĐỒNG BỘ CHIỀU CAO VỚI CỤM HÀNG 2 (ROW 2 VERTICAL HARMONY):**
>    - Chiều cao `20.0 px` khớp hoàn hảo với chiều cao của công tắc gạt tự động `autoSwitchToggle` (`38 x 20 px`) và tương thích cao độ với các nút `compactButton` (`22 - 24 px`), tạo nên một trục ngang phẳng phiu, thanh thoát.
> 3. **BẢO TỒN NGUYÊN VẸN CÁC TÙY CHỌN & CHỨC NĂNG:**
>    - Giữ nguyên 3 tùy chọn theme: `🌙`, `☀️`, `☕`.
>    - Giữ nguyên 100% logic chuyển đổi theme khi chọn: `a.SetTheme(...)`.
> 4. **BẢO TỒN NGUYÊN VẸN HÌNH HỌC ĐÃ KHÓA CỨNG:**
>    - Cửa sổ ứng dụng: CỐ ĐỊNH **`920 x 630 px`**.
>    - Thẻ Card: CỐ ĐỊNH **`280 x 130 px`**.
>    - Bảng Quota: CỐ ĐỊNH **`228 x 77 px`**.

---

## 1. PHÂN TÍCH THỊ GIÁC & HÌNH HỌC (VISUAL SPECIFICATION)

### 1.1. Bản Vẽ So Sánh Cụm Góc Phải Hàng 2

#### TRƯỚC (Dropdown Theme 58x32px quá to, lấn át text 10pt xung quanh):
```text
┌──────────────────────────────────────────────────────────────────┐
│ ... Port 62220      Tự động: [ ○  ] (20px)      [ ☀️ ▼ ] (32px) │  ◄── Cọc cạch, nút theme quá cao
└──────────────────────────────────────────────────────────────────┘
```

#### SAU (Dropdown Theme 36x20px thu nhỏ 60%, bằng phẳng tuyệt đối):
```text
┌──────────────────────────────────────────────────────────────────┐
│ ... Port 62220      Tự động: [ ○  ] (20px)      [☀️▼] (20px)     │  ◄── Bằng phẳng, thanh tú, đồng bộ
└──────────────────────────────────────────────────────────────────┘
```

### 1.2. Phân Bổ Toán Học (Geometric Budget)
- **Kích thước `themeSelectWrap` mới:**
  `themeSelectWrap := container.NewGridWrap(fyne.NewSize(36, 20), a.themeSelect)` (hoặc `fyne.NewSize(38, 20)`).
- **So sánh tỷ lệ:**
  - Chiều rộng: $36 / 58 \approx 62\%$.
  - Chiều cao: $20 / 32 \approx 62.5\%$.
  - Đúng chuẩn yêu cầu thu nhỏ còn khoảng 60% của Owner.
- **Tổng bề rộng cụm góc phải (`rightCluster`):**
  - Text `"Tự động:"` (~65px) + Công tắc gạt (38px) + Khoảng cách (8px) + Theme Select (36px) = **`~147.0 px`** (giảm hơn 25px so với trước, cực kỳ tinh tế).

---

## 2. TOUCH-MAP KHÓA CỨNG DÀNH CHO CODER (EXACT FILE TOUCHES)

### 2.1. File `internal/ui/window.go`
- **Sửa kích thước bọc của `themeSelectWrap`:**
  ```go
  // Thay thế dòng 174:
  themeSelectWrap := container.NewGridWrap(fyne.NewSize(36, 20), a.themeSelect)
  ```
  *(Nếu cần thêm khoảng trống cho icon và mũi tên tam giác của Fyne Select trên một số màn hình DPI cao, có thể dùng `fyne.NewSize(38, 20)`).*

### 2.2. File `internal/ui/ui_test.go`
- Cập nhật test case kiểm tra `themeSelectWrap`:
  - Assert kích thước của `themeSelectWrap` có chiều cao $\le 22.0\text{ px}$ và chiều rộng $\le 40.0\text{ px}$ (thay vì 58x32).
  - Chạy `go test -v ./internal/ui` đạt 100% PASS.

---

## 3. TIÊU CHÍ NGHIỆM THU (ACCEPTANCE CRITERIA)

- [ ] **AC-1 (60% Size Reduction):** Dropdown chọn theme `themeSelectWrap` được thu nhỏ xuống kích thước `36x20 px` (hoặc `38x20 px`), giảm khoảng 60% cả bề rộng và bề cao so với kích thước cũ `58x32 px`.
- [ ] **AC-2 (Uniform Height):** Chiều cao của nút chọn theme bằng `20.0 px`, ngang bằng và hài hòa với công tắc `autoSwitchToggle` và text 10pt xung quanh.
- [ ] **AC-3 (Functional Continuity):** Bấm chọn theme `🌙`, `☀️`, `☕` vẫn thay đổi theme ứng dụng mượt mà, chính xác 100%.
- [ ] **AC-4 (Dimensions Stability):** Cửa sổ giữ nguyên `920 x 630 px`, thẻ Card giữ nguyên `280 x 130 px`.
- [ ] **AC-5 (Test Suite):** Chạy `go test -v ./internal/ui` đạt 100% PASS.
