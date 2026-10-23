# BÁO CÁO THI CÔNG (CODER IMPLEMENTATION REPORT)
## CHUẨN HÓA NGỮ NGHĨA UX: NÚT "LÀM MỚI DANH SÁCH" (UI CHÍNH) VÀ BẢO TOÀN NÚT "QUÉT CHROME" (MODAL)

- **Mã báo cáo:** `rp_coder_20261023_toolbar_refresh_clarification`
- **Tác giả:** Coder Lane
- **Ngày thi công:** 2026-10-23
- **Thẩm quyền kiến trúc (SSOT):** `reports/architect/rp_architect_20261023_160500_toolbar_refresh_clarification_spec.md` (commit `8fc317942f6c75353fe4019d0ec88300c4fe439c`)
- **Trạng thái:** **`VERDICT: PASS / READY_FOR_QA`**

---

## 1. INVARIANT FAMILY MAP

- **Family Name:** Toolbar Refresh UX Clarification & Chrome Scan Decoupling
- **SSOT / Authority:** `reports/architect/rp_architect_20261023_160500_toolbar_refresh_clarification_spec.md` (commit `8fc317942f6c75353fe4019d0ec88300c4fe439c`)
- **Core Invariants:**
  1. **`Invariant-Toolbar-Refresh-Semantics` (Chuẩn hóa ngữ nghĩa nút Toolbar):**
     - Nút trên Toolbar chính phải luôn hiển thị nhãn mặc định: `"Làm mới danh sách"`.
     - Trong suốt thời gian goroutine xử lý bất đồng bộ của hàm `Refresh()`, nhãn nút phải là: `"⏳ Đang làm mới..."`.
     - Sau khi hoàn thành hoặc thoát khỏi hàm `Refresh()`, nhãn nút phải trở về: `"Làm mới danh sách"`.
     - Tuyệt đối CẤM hiển thị chữ `"Đang quét"` trên nút Toolbar này.
  2. **`Invariant-Decoupled-Chrome-Scan` (Không quét Chrome khi làm mới):**
     - Hàm `Refresh()` trên UI chính bảo toàn triệt để nguyên lý: Chỉ đọc local storage và Windows Credential Manager, tuyệt đối KHÔNG quét Chrome.
  3. **`Invariant-Modal-Scan-Preservation` (Bảo toàn chức năng Quét Chrome):**
     - Nút `"Quét Chrome"` trong Modal Thêm tài khoản giữ nguyên 100% toàn bộ nhãn `"Quét Chrome"`, icon và trạng thái chờ `"⏳ Đang quét..."` khi quét profile Chrome thực tế trên ổ đĩa.

---

## 2. CHI TIẾT THI CÔNG (IMPLEMENTATION DETAILS)

### 2.1. Tệp Khởi tạo Toolbar: `internal/ui/window.go`
- Tại dòng 129, đổi nhãn khởi tạo của `a.refreshBtn` từ `"Danh sách tài khoản"` thành `"Làm mới danh sách"`:
  ```go
  a.refreshBtn = widget.NewButton("Làm mới danh sách", func() {
      a.Refresh()
  })
  ```

### 2.2. Tệp Xử lý Ứng dụng: `internal/ui/app.go`
- Tại hàm `Refresh()`, cập nhật trạng thái chờ thành `"⏳ Đang làm mới..."` và trả về `"Làm mới danh sách"` trong khối `defer`:
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
  ```

### 2.3. Tệp Kiểm thử: `internal/ui/ui_test.go`
- Cập nhật kiểm thử `TestArchitectSpec_MergeToolbarWithServerStatusRow_FullConformance` tại dòng 2914 kiểm tra nhãn `"Làm mới danh sách"`.
- Bổ sung 3 unit tests trực tiếp bám sát 100% Ma trận Kiểm thử của Architect Spec:
  1. `TestToolbar_RefreshButton_DefaultLabel`: Xác thực nhãn mặc định của `app.refreshBtn` là `"Làm mới danh sách"`.
  2. `TestToolbar_RefreshButton_LoadingAndResetState`: Xác thực khi gọi `app.Refresh()`, text chuyển ngay sang `"⏳ Đang làm mới..."`, và sau khi hoàn tất tự động trả về `"Làm mới danh sách"`.
  3. `TestAddModal_ScanChromeButton_Preserved`: Xác thực nút trong Modal Thêm tài khoản bảo toàn 100% nhãn `"Quét Chrome"` ban đầu, chuyển `"⏳ Đang quét..."` khi nhấn và phục hồi `"Quét Chrome"` khi hoàn tất.

---

## 3. DANH SÁCH FILE THAY ĐỔI (TOUCH-MAP)

| Đường dẫn file | Tóm tắt thay đổi |
| :--- | :--- |
| `internal/ui/window.go` | Đổi nhãn khởi tạo `refreshBtn` thành `"Làm mới danh sách"`. |
| `internal/ui/app.go` | Đổi text loading thành `"⏳ Đang làm mới..."` và reset thành `"Làm mới danh sách"`. |
| `internal/ui/ui_test.go` | Cập nhật AC-1 trong `TestArchitectSpec_MergeToolbarWithServerStatusRow_FullConformance` và thêm 3 unit tests: `TestToolbar_RefreshButton_DefaultLabel`, `TestToolbar_RefreshButton_LoadingAndResetState`, `TestAddModal_ScanChromeButton_Preserved`. |
| `reports/coder/rp_coder_20261023_toolbar_refresh_clarification.md` | Báo cáo thi công chính thức của Coder Lane. |

---

## 4. KẾT QUẢ KIỂM THỬ (VERIFICATION EVIDENCE)

### 4.1. Unit Tests Chuyên biệt cho Toolbar và Modal Scan
```text
=== RUN   TestArchitectSpec_MergeToolbarWithServerStatusRow_FullConformance
--- PASS: TestArchitectSpec_MergeToolbarWithServerStatusRow_FullConformance (0.47s)
=== RUN   TestToolbar_RefreshButton_DefaultLabel
--- PASS: TestToolbar_RefreshButton_DefaultLabel (0.21s)
=== RUN   TestToolbar_RefreshButton_LoadingAndResetState
--- PASS: TestToolbar_RefreshButton_LoadingAndResetState (0.31s)
=== RUN   TestAddModal_ScanChromeButton_Preserved
--- PASS: TestAddModal_ScanChromeButton_Preserved (0.46s)
PASS
ok  	antigravity-switch/internal/ui	2.864s
```

### 4.2. Kiểm thử Toàn bộ Package `internal/ui/...`
```text
PASS
ok  	antigravity-switch/internal/ui	44.691s
```

### 4.3. Kiểm thử Toàn bộ Kho lưu trữ (`go test ./...`)
```text
ok  	antigravity-switch/cmd/switcher	1.823s
ok  	antigravity-switch/internal/automation	(cached)
ok  	antigravity-switch/internal/connector	(cached)
ok  	antigravity-switch/internal/manager	(cached)
ok  	antigravity-switch/internal/model	(cached)
ok  	antigravity-switch/internal/reloader	(cached)
ok  	antigravity-switch/internal/scanner	(cached)
ok  	antigravity-switch/internal/store	6.394s
ok  	antigravity-switch/internal/telemetry	(cached)
ok  	antigravity-switch/internal/ui	28.171s
ok  	antigravity-switch/internal/version	(cached)
ok  	antigravity-switch/internal/wincred	(cached)
```

---

## 5. KẾT LUẬN & BÀN GIAO (HANDOFF)

- Đã giải quyết triệt để 100% mâu thuẫn thuật ngữ UX trên Toolbar UI chính.
- Bảo toàn nguyên vẹn chức năng quét Chrome sâu trong Modal Thêm tài khoản.
- Không phát sinh hồi quy, toàn bộ test suite pass 100%. Sẵn sàng bàn giao QA Lane.
