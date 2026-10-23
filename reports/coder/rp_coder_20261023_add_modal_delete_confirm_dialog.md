# BÁO CÁO THI CÔNG (CODER IMPLEMENTATION REPORT)

- **Mã báo cáo:** `rp_coder_20261023_add_modal_delete_confirm_dialog`
- **Tác giả:** Coder Lane
- **Ngày thi công:** 2026-10-23
- **Thẩm quyền kiến trúc (SSOT):** `reports/architect/rp_architect_20261023_131500_add_modal_delete_and_nonexistent_account_spec.md` (commit `3059dbb27958668ed81205c6bc28d13da9505b26`)
- **Trạng thái:** **`VERDICT: PASS / READY_FOR_QA`**

---

## 1. INVARIANT FAMILY MAP

- **Family Name:** Add Modal Delete Confirmation Dialog (Popup cảnh báo xác nhận xóa trên Modal Thêm tài khoản)
- **SSOT / Authority:** `reports/architect/rp_architect_20261023_131500_add_modal_delete_and_nonexistent_account_spec.md` (commit `3059dbb27958668ed81205c6bc28d13da9505b26`)
- **Core Invariants:**
  1. **`Invariant-Delete-Confirmation-Dialog` (Nguyên tắc bắt buộc Popup xác nhận xóa):**
     - Khi người dùng nhấn nút `[Xóa]` trên bất kỳ dòng tài khoản nào trong Modal Thêm tài khoản, hệ thống **TUYỆT ĐỐI KHÔNG** xóa ngay lập tức.
     - BẮT BUỘC phải mở hộp thoại Popup xác nhận (`dialog.NewConfirm`).
     - Chỉ khi người dùng bấm nút xác nhận ("Xóa" - `widget.DangerImportance`) trên popup thì thao tác xóa file và gỡ dòng mới được thực thi.
     - Nếu người dùng bấm "Hủy" hoặc đóng popup, hệ thống giữ nguyên 100% hiện trạng file trên đĩa và dòng hiển thị trên modal.
  2. **`Invariant-Store-Isolation` (Cô lập hai kho dữ liệu):**
     - Nút `[Xóa]` trong Modal Thêm tài khoản CHỈ thao tác trên thư mục `discovered_accounts/`.
     - Tuyệt đối không xóa hay sửa đổi `accounts_store/`, không gọi `RemoveAccount()`, và không làm biến mất thẻ tài khoản trên màn hình chính (`UI chính`).
  3. **`Invariant-Ghost-Account-State` (Xử lý tài khoản không tồn tại trên đĩa):**
     - Đối với dòng tài khoản mang trạng thái `[Tài khoản không tồn tại]` (`FileExists == false`), thông điệp popup xác nhận cảnh báo rõ ràng rằng tệp dữ liệu không tồn tại trên hệ thống và hỏi người dùng có muốn gỡ bỏ khỏi danh sách hiển thị không.
  4. **`Invariant-Modal-Removal` (Làm sạch danh sách Modal):**
     - Sau khi người dùng bấm xác nhận "Xóa" trên popup, dòng tài khoản tương ứng lập tức được gỡ bỏ khỏi danh sách hiển thị Modal (`reloadCandidates()`).

---

## 2. CHI TIẾT THI CÔNG (IMPLEMENTATION DETAILS)

### 2.1. Tầng Giao diện UI (`internal/ui/add_modal.go`)
1. **Cài đặt hàm `(a *App) showDeleteCandidateConfirm(cand manager.CandidateAccountItem, onConfirmed func()) *dialog.ConfirmDialog`:**
   - Kiểm tra an toàn `a == nil || a.win == nil` trả về `nil`.
   - Lấy email hiển thị có áp dụng che email nếu người dùng bật chế độ riêng tư (`a.IsMaskAccounts()` -> `MaskEmail(cand.Email)`).
   - Thiết lập tiêu đề: `"Xác nhận xóa tài khoản"`.
   - Phân luồng thông điệp theo 2 kịch bản:
     + **Kịch bản 1 (Tài khoản bình thường có file - `cand.FileExists == true`):**
       `"Bạn có chắc chắn muốn xóa tài khoản %s khỏi danh bạ đã quét không?\n\n(Lưu ý: Thao tác này chỉ xóa dữ liệu trong danh bạ khám phá, không xóa tài khoản hay token trên màn hình chính)."`
     + **Kịch bản 2 (Tài khoản không tồn tại trên hệ thống - `cand.FileExists == false`):**
       `"Tài khoản %s không có tệp dữ liệu trên hệ thống.\nBạn có muốn gỡ bỏ tài khoản này khỏi danh sách hiển thị không?"`
   - Khởi tạo hộp thoại `dialog.NewConfirm`:
     + Nút xác nhận: `confirmDlg.SetConfirmText("Xóa")` với màu cảnh báo `confirmDlg.SetConfirmImportance(widget.DangerImportance)`.
     + Nút hủy: `confirmDlg.SetDismissText("Hủy")`.
   - Xử lý callback của Dialog:
     + Nếu `!confirmed`: kết thúc ngay lập tức, không làm gì.
     + Nếu `confirmed`:
       * Nếu `cand.FileExists && a.mgr != nil`: thực thi xóa file `a.mgr.DeleteDiscoveredAccount(cand.Email)`.
       * Đưa việc cập nhật UI lên Fyne main thread qua `fyne.Do(func() { if onConfirmed != nil { onConfirmed() } })`.
2. **Tích hợp vào nút `deleteBtn` trên mọi dòng tài khoản:**
   ```go
   deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
       a.showDeleteCandidateConfirm(candItem, func() {
           norm := strings.ToLower(strings.TrimSpace(candItem.Email))
           dismissedAccounts[norm] = true
           reloadCandidates()
       })
   })
   deleteBtn.Importance = widget.LowImportance
   compactDel := compactButton(deleteBtn)
   ```

---

## 3. DANH SÁCH FILE THAY ĐỔI (TOUCH-MAP)

| Đường dẫn file | Nội dung thay đổi |
|:---|:---|
| `internal/ui/add_modal.go` | Cài đặt `showDeleteCandidateConfirm` với 2 kịch bản thông điệp, nút Danger "Xóa", tích hợp vào `deleteBtn`. |
| `internal/ui/add_modal_test.go` | Cài đặt helpers `findButtonsInCanvas`, `findLabelsInCanvas`; viết `TestModal_Delete_Click_Shows_Confirm_Dialog`, `TestModal_Delete_Dialog_Cancel`, `TestModal_Delete_Dialog_Confirm`, `TestModal_Delete_NonExistent`, `TestShowDeleteCandidateConfirm_NilSafety`, `TestShowDeleteCandidateConfirm_EmailMasking`. |
| `internal/ui/add_modal_acceptance_test.go` | Thêm acceptance test `TestAcceptance_TC09_DeleteCandidateConfirmationFlow` kiểm thử toàn diện luồng: Click Xóa -> Popup -> Hủy -> Không mất dữ liệu -> Click Xóa -> Xác nhận -> Xóa file & gỡ dòng & bảo toàn UI chính; và Xóa dòng tài khoản không tồn tại. |
| `reports/coder/rp_coder_20261023_add_modal_delete_confirm_dialog.md` | Báo cáo thi công chính thức của Coder Lane. |

---

## 4. KẾT QUẢ KIỂM THỬ (VERIFICATION EVIDENCE)

### 4.1. Unit Tests chuyên biệt cho Popup Xác Nhận Xóa (`internal/ui/add_modal_test.go`)
```text
=== RUN   TestModal_Delete_Button_Rendered
--- PASS: TestModal_Delete_Button_Rendered (0.19s)
=== RUN   TestModal_Delete_Click_Shows_Confirm_Dialog
--- PASS: TestModal_Delete_Click_Shows_Confirm_Dialog (0.05s)
=== RUN   TestModal_Delete_Dialog_Cancel
--- PASS: TestModal_Delete_Dialog_Cancel (0.05s)
=== RUN   TestModal_Delete_Dialog_Confirm
--- PASS: TestModal_Delete_Dialog_Confirm (0.06s)
=== RUN   TestModal_Delete_NonExistent
--- PASS: TestModal_Delete_NonExistent (0.04s)
=== RUN   TestShowDeleteCandidateConfirm_NilSafety
--- PASS: TestShowDeleteCandidateConfirm_NilSafety (0.00s)
=== RUN   TestShowDeleteCandidateConfirm_EmailMasking
--- PASS: TestShowDeleteCandidateConfirm_EmailMasking (0.05s)
PASS
ok  	antigravity-switch/internal/ui	1.868s
```

### 4.2. Acceptance Test End-to-End (`internal/ui/add_modal_acceptance_test.go`)
```text
=== RUN   TestAcceptance_TC09_DeleteCandidateConfirmationFlow
--- PASS: TestAcceptance_TC09_DeleteCandidateConfirmationFlow (0.29s)
PASS
ok  	antigravity-switch/internal/ui	1.937s
```

### 4.3. Toàn bộ Test Suite Package UI (`internal/ui`)
```text
PASS
ok  	antigravity-switch/internal/ui	30.927s
```

### 4.4. Toàn bộ Test Suite Repository (`go test ./...`)
```text
ok  	antigravity-switch/cmd/switcher	1.487s
ok  	antigravity-switch/internal/automation	(cached)
ok  	antigravity-switch/internal/connector	(cached)
ok  	antigravity-switch/internal/manager	(cached)
ok  	antigravity-switch/internal/model	(cached)
ok  	antigravity-switch/internal/reloader	(cached)
ok  	antigravity-switch/internal/scanner	(cached)
ok  	antigravity-switch/internal/store	(cached)
ok  	antigravity-switch/internal/telemetry	(cached)
ok  	antigravity-switch/internal/ui	26.327s
ok  	antigravity-switch/internal/version	(cached)
ok  	antigravity-switch/internal/wincred	(cached)
```

---

## 5. KẾT LUẬN & SẴN SÀNG BÀN GIAO (CONCLUSION)

Toàn bộ yêu cầu chỉ đạo trực tiếp từ Human Owner và bản cập nhật Đặc tả Kiến trúc đã được triển khai chính xác 100%:
- Đóng kín nguyên tắc bất biến `Invariant-Delete-Confirmation-Dialog`: Không bao giờ xóa ngay lập tức khi click nút Xóa trên dòng.
- Bật popup hộp thoại xác nhận rõ ràng, màu Danger đỏ trên nút Xóa.
- Kiểm thử bao phủ đầy đủ kịch bản Hủy, kịch bản Xác nhận, kịch bản Tài khoản không tồn tại, và Che email.
- Toàn bộ test suite 100% PASS.

**Đánh giá của Coder:** **`VERDICT: PASS / READY_FOR_QA`**
