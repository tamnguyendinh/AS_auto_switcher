# BÁO CÁO ĐẶC TẢ KIẾN TRÚC (ARCHITECTURE SPECIFICATION REPORT)
## CHUẨN HÓA NGỮ NGHĨA UX: NÚT "LÀM MỚI DANH SÁCH" (UI CHÍNH) VÀ NÚT "QUÉT CHROME" (MODAL)

- **Mã tài liệu:** `rp_architect_20261023_160500_toolbar_refresh_clarification_spec`
- **Tác giả:** System Architect Lane (Lane Sạch)
- **Ngày lập:** 2026-10-23
- **Thẩm quyền:** Human Owner
- **Đối tượng thụ hưởng:** Coder Lane, QA Lane
- **Trạng thái:** **`VERDICT: PASS / READY_FOR_CODER`**

---

## 1. BỐI CẢNH & MỤC TIÊU (CONTEXT & OBJECTIVES)

### 1.1. Bối cảnh
Trước đây, giao diện tồn tại sự mâu thuẫn về mặt thuật ngữ và trạng thái chờ (loading state) gây hiểu lầm nghiêm trọng cho người dùng:
- Nút trên thanh Toolbar của màn hình chính có tên là `"Danh sách tài khoản"`, nhưng khi người dùng bấm vào, text của nút lại đổi thành `"⏳ Đang quét..."`.
- Sự xuất hiện của chữ `"⏳ Đang quét..."` khiến người dùng tưởng rằng hệ thống đang đi quét trình duyệt Chrome trên máy tính, trong khi bản chất thật sự của hàm xử lý `Refresh()` chỉ là nạp lại dữ liệu từ các file cục bộ (`accounts_store/` và `discovered_accounts/`).
- Trong khi đó, nút `"Quét Chrome"` trong Modal Thêm tài khoản mới thực sự là nơi quét sâu vào các profile của Google Chrome trên ổ đĩa.

### 1.2. Mục tiêu Đặc tả
1. **Chuẩn hóa nút trên Toolbar UI chính:**
   - Đổi tên nút từ `"Danh sách tài khoản"` thành **`"Làm mới danh sách"`** (Refresh) để thể hiện rõ động từ hành động.
   - Đổi text trạng thái chờ khi đang thực thi từ `"⏳ Đang quét..."` thành **`"⏳ Đang làm mới..."`** để phản ánh đúng bản chất kỹ thuật (tải lại dữ liệu cục bộ, không quét Chrome).
2. **Bảo toàn nút "Quét Chrome" trong Modal Thêm tài khoản:**
   - Giữ nguyên 100% nhãn `"Quét Chrome"` và trạng thái `"⏳ Đang quét..."`. Đây là điểm duy nhất trên toàn hệ thống thực hiện tác vụ quét I/O vào trình duyệt Chrome.

---

## 2. NGUYÊN TẮC BẤT BIẾN BẮT BUỘC (ARCHITECTURAL INVARIANTS)

1. **`Invariant-Toolbar-Refresh-Semantics` (Chuẩn hóa ngữ nghĩa nút Toolbar):**
   - Nút trên Toolbar chính phải luôn hiển thị nhãn: `"Làm mới danh sách"`.
   - Trong suốt thời gian goroutine xử lý bất đồng bộ, nhãn nút phải là: `"⏳ Đang làm mới..."`.
   - Sau khi hoàn thành hoặc gặp lỗi, nhãn nút phải trở về: `"Làm mới danh sách"`.
   - Tuyệt đối CẤM hiển thị chữ `"Đang quét"` trên nút Toolbar này.
2. **`Invariant-Decoupled-Chrome-Scan` (Không quét Chrome khi làm mới):**
   - Hàm `Refresh()` trên UI chính bảo toàn triệt để nguyên lý: **Chỉ đọc local storage và Windows Credential Manager, tuyệt đối KHÔNG quét Chrome**.
3. **`Invariant-Modal-Scan-Preservation` (Bảo toàn chức năng Quét Chrome):**
   - Nút `"Quét Chrome"` trong Modal Thêm tài khoản giữ nguyên toàn bộ nhãn, icon và chức năng quét profiles Chrome.

---

## 3. CHI TIẾT ĐẶC TẢ KỸ THUẬT (DETAILED IMPLEMENTATION SPECIFICATION)

### 3.1. Tệp Giao diện Thanh công cụ: `internal/ui/window.go`

Tại vị trí khởi tạo nút Toolbar (khoảng dòng 129):
- **Hiện tại:**
  ```go
  a.refreshBtn = widget.NewButton("Danh sách tài khoản", func() {
      a.Refresh()
  })
  ```
- **Chuẩn hóa mới:**
  ```go
  a.refreshBtn = widget.NewButton("Làm mới danh sách", func() {
      a.Refresh()
  })
  ```

---

### 3.2. Tệp Xử lý Ứng dụng: `internal/ui/app.go`

Tại hàm `Refresh()` (khoảng dòng 806 - 830):
- **Hiện tại:**
  ```go
  if a.refreshBtn != nil {
      fyne.Do(func() {
          a.refreshBtn.SetText("⏳ Đang quét...")
      })
  }

  go func() {
      defer func() {
          fyne.Do(func() {
              if a.refreshBtn != nil {
                  a.refreshBtn.SetText("Danh sách tài khoản")
              }
              a.mu.Lock()
              a.isBusy = false
              a.mu.Unlock()
          })
      }()
      ...
  ```
- **Chuẩn hóa mới:**
  ```go
  if a.refreshBtn != nil {
      fyne.Do(func() {
          a.refreshBtn.SetText("⏳ Đang làm mới...")
      })
  }

  go func() {
      defer func() {
          fyne.Do(func() {
              if a.refreshBtn != nil {
                  a.refreshBtn.SetText("Làm mới danh sách")
              }
              a.mu.Lock()
              a.isBusy = false
              a.mu.Unlock()
          })
      }()
      ...
  ```

---

### 3.3. Tệp Kiểm thử Giao diện: `internal/ui/ui_test.go`

Cập nhật ca kiểm thử AC-1 tại khoảng dòng 2914:
- **Hiện tại:**
  ```go
  if app.refreshBtn.Text != "Danh sách tài khoản" {
      t.Errorf("expected refreshBtn text 'Danh sách tài khoản', got '%s'", app.refreshBtn.Text)
  }
  ```
- **Chuẩn hóa mới:**
  ```go
  if app.refreshBtn.Text != "Làm mới danh sách" {
      t.Errorf("expected refreshBtn text 'Làm mới danh sách', got '%s'", app.refreshBtn.Text)
  }
  ```

---

## 4. MA TRẬN ĐỐI CHIẾU SAU CHUẨN HÓA (SEMANTIC MATRIX)

| Vị trí | Tên nút mặc định | Text khi đang xử lý | Hành vi kỹ thuật thực tế |
| :--- | :---: | :---: | :--- |
| **UI chính (Toolbar)** | **`Làm mới danh sách`** | **`⏳ Đang làm mới...`** | Đọc lại `accounts_store/` + `discovered_accounts/`, cập nhật lại toàn bộ Card ngoài màn hình chính. |
| **Modal Thêm tài khoản** | **`Quét Chrome`** | **`⏳ Đang quét...`** | Quét các Profile Chrome thực tế trên ổ cứng, ghi file vào `discovered_accounts/`, cập nhật danh sách ứng viên trong modal. |

---

## 5. DANH MỤC KIỂM THỬ BẮT BUỘC (TESTING MATRIX)

Coder và QA phải đảm bảo các test sau chạy PASS 100%:

1. **`TestToolbar_RefreshButton_DefaultLabel`:**
   - Khởi tạo UI chính $\rightarrow$ kiểm tra `app.refreshBtn.Text == "Làm mới danh sách"`.
2. **`TestToolbar_RefreshButton_LoadingAndResetState`:**
   - Kích hoạt `app.Refresh()` $\rightarrow$ kiểm tra ngay lập tức `app.refreshBtn.Text == "⏳ Đang làm mới..."`.
   - Chờ hoàn tất tác vụ $\rightarrow$ kiểm tra text trả về `"Làm mới danh sách"`.
3. **`TestAddModal_ScanChromeButton_Preserved`:**
   - Mở Modal Thêm tài khoản $\rightarrow$ kiểm tra `scanBtn.Text == "Quét Chrome"`.
   - Nhấn nút $\rightarrow$ kiểm tra `scanBtn.Text == "⏳ Đang quét..."`.
   - Hoàn tất $\rightarrow$ trả về `"Quét Chrome"`.

---

## 6. KẾT LUẬN

Bản đặc tả này chuẩn hóa dứt điểm sự mâu thuẫn UX, mang lại sự trong sáng, minh bạch và chính xác tuyệt đối giữa hai tác vụ: **Làm mới dữ liệu giao diện** và **Quét dữ liệu từ trình duyệt Chrome**.
