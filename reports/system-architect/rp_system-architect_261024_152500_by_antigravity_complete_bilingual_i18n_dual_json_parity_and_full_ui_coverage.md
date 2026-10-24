# BÁO CÁO ĐẶC TẢ KIẾN TRÚC CHI TIẾT (SYSTEM ARCHITECTURE SPECIFICATION)
## Tính năng: Chuẩn Hóa Toàn Diện Đa Ngôn Ngữ i18n Bằng Cặp File JSON Song Song & Xóa Bỏ Triệt Để Chuỗi Cứng Trên Toàn Bộ Giao Diện

- **Mã tài liệu:** `rp_system-architect_261024_152500_by_antigravity_complete_bilingual_i18n_dual_json_parity_and_full_ui_coverage`
- **Tác giả:** System Architect (Strategic Lane)
- **Ngày lập:** 2026-10-24
- **Thẩm quyền phê duyệt:** Human Owner
- **Trạng thái:** PRODUCTION-READY BLUEPRINT (Bản thiết kế chìa khóa trao tay cho Coder)
- **Mục tiêu tối thượng:** 
  1. Loại bỏ 100% tình trạng hardcode chuỗi hiển thị rải rác trong `internal/ui/app.go`, `internal/ui/window.go`, `internal/ui/widgets.go`, `internal/ui/add_modal.go`, `internal/ui/quota_badge.go`, `internal/ui/tray.go`.
  2. Xóa bỏ hoàn toàn hiện tượng lai căng ngôn ngữ (nửa Anh nửa Việt như `[Đã add]`, `Week`, `(Ready)`, `ADD`), thực thi nguyên tắc bất biến: **"Tiếng Anh ra tiếng Anh, tiếng Việt ra tiếng Việt"**.
  3. Chuyển đổi toàn bộ từ điển sang **cặp file JSON đối soát song song (`vi.json` và `en.json`)** nhúng trực tiếp qua `//go:embed`, có bài kiểm thử tự động kiểm tra đối xứng 1:1 (Key-Parity Test).
  4. Thiết lập quy trình thi công theo cơ chế cổng kiểm soát tuần tự (Gating System): Bước nào pass 100% kiểm thử thì Coder mới được phép thi công bước tiếp theo.

---

# PHẦN 1: BÁO CÁO HIỆN TRẠNG & DANH SÁCH VẤN ĐỀ TỒN ĐỌNG (PROBLEM LIST)

Qua khảo sát toàn diện mã nguồn Fyne GUI của `antigravity-switch`, Kiến trúc sư phát hiện 9 nhóm vấn đề vi phạm nguyên tắc thiết kế đa ngôn ngữ và tiêu chuẩn phần mềm:

### Vấn đề 1: Hardcode trong các trạng thái tải (Loading/Reset) của Toolbar
- **Vị trí:** `internal/ui/app.go:771, 781, 818, 928, 937`.
- **Hiện tượng:** Khi người dùng bấm "Làm mới danh sách", code gán cứng `a.refreshBtn.SetText("⏳ Đang làm mới...")` và khôi phục `a.refreshBtn.SetText("Làm mới danh sách")`. Khi bấm "Làm mới Quota", code gán cứng `"⏳ Đang quét..."` và khôi phục `"Làm mới Quota"`.
- **Hệ quả:** Dù người dùng đang ở giao diện tiếng Anh (`EN`), chỉ cần bấm nút thì text lập tức bị biến thành tiếng Việt, phá vỡ trải nghiệm i18n.

### Vấn đề 2: Ngôn ngữ lai căng trong Lưới Quota Matrix (3x3) & Bộ đếm thời gian
- **Vị trí:** `internal/ui/quota_badge.go:275, 837`.
- **Hiện tượng:** Cột chu kỳ tuần bị gán cứng `"Week"` ngay cả khi đang hiển thị tiếng Việt. Trạng thái quota còn đầy (remaining = 100%) bị gán cứng `"(Ready)"`.
- **Hệ quả:** Giao diện tiếng Việt bị lai tiếng Anh (`Week`, `(Ready)`). Không đạt tiêu chuẩn "Tiếng Việt ra tiếng Việt".

### Vấn đề 3: Hardcode chuỗi hiển thị kết nối Antigravity Server trên Thanh Đáy
- **Vị trí:** `internal/ui/window.go:139`, `internal/ui/app.go:1564, 1566`.
- **Hiện tượng:** Nhãn `serverStatusLabel` khởi tạo gán cứng `"Antigravity: Hoạt động  | Đang kết nối..."`. Khi có Language Server gán cứng `"Antigravity: Hoạt động  | PID %d | Port %d"`, khi mất kết nối gán cứng `"Antigravity: Không hoạt động  | Dự phòng WinCred"`.
- **Hệ quả:** Thanh đáy luôn hiển thị tiếng Việt bất kể người dùng chọn tiếng Anh hay tiếng Việt.

### Vấn đề 4: 14+ dòng thông báo phản hồi (Status Bar) bị hardcode hoàn toàn
- **Vị trí:** `internal/ui/app.go:670, 957, 981, 1008, 1061, 1120, 1214, 1276, 1299, 1538, 1540`.
- **Hiện tượng:** Tất cả các thông báo lỗi, thông báo thành công (chuyển tài khoản, xóa tài khoản, ẩn danh tài khoản, không có ứng viên) đều dùng chuỗi tiếng Việt cố định truyền vào `showStatusBar(...)`.
- **Hệ quả:** Người dùng quốc tế không thể hiểu được các thông báo nghiệp vụ quan trọng.

### Vấn đề 5: Modal "Thêm tài khoản" (`add_modal.go`) bị hardcode 100%
- **Vị trí:** `internal/ui/add_modal.go:55, 84, 92, 100, 108, 113, 116, 144, 146, 151, 159, 181`.
- **Hiện tượng:** Tiêu đề `"Thêm tài khoản"`, nút `"Quét Chrome"`, trạng thái `"⏳ Đang quét..."`, thông báo rỗng `"Không có tài khoản nào trong danh bạ."`, các badge `"[✔ Đã OAuth]"`, `"[Cần OAuth]"`, `"[Tài khoản không tồn tại]"`, và đặc biệt là từ lai `"[Đã add]"` đều bị viết cứng.
- **Hệ quả:** Toàn bộ trải nghiệm nạp tài khoản bị cô lập trong tiếng Việt lai tiếng Anh.

### Vấn đề 6: Toàn bộ Hộp thoại Popup Xác nhận Xóa bị hardcode
- **Vị trí:** `internal/ui/add_modal.go:197-224` (`showDeleteCandidateConfirm`) và `internal/ui/app.go:1235-1258` (`ShowRemoveAccountConfirmation`).
- **Hiện tượng:** Tiêu đề `"Xác nhận xóa tài khoản"`, câu cảnh báo xóa tài khoản, nút `"Xóa"`, nút `"Hủy"` đều là chuỗi tiếng Việt cứng trong hàm tạo dialog của Fyne.
- **Hệ quả:** Người dùng chọn EN vẫn gặp popup tiếng Việt.

### Vấn đề 7: Menu Khay Hệ Thống (System Tray) bị hardcode
- **Vị trí:** `internal/ui/tray.go:18, 27, 31`.
- **Hiện tượng:** `"Hiển thị Switcher"`, `"Làm mới Chrome"`, `"Thoát"` được truyền cứng vào `fyne.NewMenuItem`, không liên kết với `i18n.Current()`.
- **Hệ quả:** Tray Menu không bao giờ đổi ngôn ngữ theo cài đặt.

### Vấn đề 8: Kiến trúc i18n hiện tại thiếu file JSON song song
- **Vị trí:** `internal/i18n/dictionary.go`.
- **Hiện tượng:** Toàn bộ từ điển đang viết trực tiếp bằng mã nguồn Go (`viDictionary`, `enDictionary` struct literals).
- **Hệ quả:** Không có công cụ đối soát song song trực quan ngoài code, khó tra soát lỗi chính tả hoặc thiếu key, không đúng chuẩn ngành phần mềm i18n.

### Vấn đề 9: Thông báo trạng thái trống ở màn hình chính bị hardcode
- **Vị trí:** `internal/ui/app.go:1597`.
- **Hiện tượng:** `emptyLabel := widget.NewLabelWithStyle("Không tìm thấy tài khoản Google nào trên Chrome.", ...)` viết cứng tiếng Việt.

---

# PHẦN 2: NGUYÊN TẮC BẤT BIẾN & RÀNG BUỘC KIẾN TRÚC (INVARIANTS)

1. **Pure Language Invariant ("Tiếng Anh ra tiếng Anh, tiếng Việt ra tiếng Việt"):**
   - Không được tồn tại bất kỳ từ ngữ lai căng nào.
   - Ở chế độ `vi`: `[Đã thêm]`, `Tuần`, `(Sẵn sàng)`, `THÊM`, `Cổng %d`.
   - Ở chế độ `en`: `[Added]`, `Week`, `(Ready)`, `ADD`, `Port %d`.
2. **Dual JSON Parity Invariant (Đối soát Song song 1:1):**
   - Mỗi ngôn ngữ là 1 file JSON độc lập: `internal/i18n/locales/vi.json` và `internal/i18n/locales/en.json`.
   - Hai file JSON này bắt buộc phải có tập hợp Key giống hệt nhau 100%. Không thừa, không thiếu, không rỗng giá trị.
3. **Compile-time Embedding Invariant (Nhúng Binary An toàn):**
   - Dùng `//go:embed locales/*.json` trong package `internal/i18n`. Toàn bộ file JSON được nhúng thẳng vào file nhị phân `antigravity-switch.exe`.
   - Cấm đọc file qua đường dẫn tương đối từ ổ cứng lúc runtime để tránh lỗi thiếu file trong môi trường production.
4. **Zero-Restart & In-Place Update Invariant:**
   - Chuyển đổi ngôn ngữ cập nhật tức thì trên Canvas UI đang hoạt động bằng cách gọi `.SetText()` và `.Refresh()`. Cấm gọi `win.SetContent()` hay khởi tạo lại cửa sổ làm chớp giật màn hình.
5. **Backward Compatibility Invariant:**
   - Khi file `logs/ui_preferences.json` chưa có trường `language` hoặc mang giá trị bất thường, hệ thống tự động fallback về `"vi"`. Toàn bộ test suite Tiếng Việt hiện hành tiếp tục pass 100%.

---

# PHẦN 3: ĐẶC TẢ CHI TIẾT NỘI DUNG 2 FILE JSON ĐỐI SOÁT SONG SONG

Dưới đây là nội dung chuẩn mực của 2 file JSON. Coder phải đưa nguyên vẹn nội dung này vào thư mục `internal/i18n/locales/`:

### 3.1. File `internal/i18n/locales/vi.json`
```json
{
  "app_title": "Antigravity Multi-Account Switcher",
  "lang_dropdown_label": "VN",

  "btn_add": "Thêm",
  "btn_refresh_list": "Làm mới danh sách",
  "btn_refresh_list_loading": "⏳ Đang làm mới...",
  "btn_refresh_quota": "Làm mới Quota",
  "btn_refresh_quota_loading": "⏳ Đang quét...",
  "btn_always_on_top": "📌",
  "lbl_mask_accounts": "Ẩn tài khoản:",
  "lbl_auto_switch": "Tự động:",

  "main_empty_accounts": "Không tìm thấy tài khoản Google nào trên Chrome.",
  "badge_active": "[✔ Đang dùng]",
  "badge_switch": "[Chuyển]",
  "badge_need_oauth": "[Cần OAuth]",
  "badge_added": "[Đã thêm]",
  "badge_account_not_found": "[Tài khoản không tồn tại]",

  "quota_col_5h": "5h",
  "quota_col_week": "Tuần",
  "quota_ready": "(Sẵn sàng)",
  "quota_unavailable": "N/A",

  "server_connecting": "Antigravity: Hoạt động  | Đang kết nối...",
  "server_connected": "Antigravity: Hoạt động  | PID %d | Cổng %d",
  "server_stopped": "Antigravity: Không hoạt động  | Dự phòng WinCred",

  "status_no_candidate": "⚠️ Không có tài khoản để chuyển.",
  "status_fetch_error": "Không lấy được danh sách tài khoản: %s",
  "status_fetch_success": "Đã cập nhật danh sách tài khoản",
  "status_switching_to": "⏳ Đang chuyển sang %s...",
  "status_switched_success": "✅ Đã chuyển sang %s",
  "status_update_error": "❌ Lỗi cập nhật trạng thái: %v",
  "status_cannot_delete_active": "⚠️ Không thể xóa tài khoản đang hoạt động.",
  "status_delete_error": "❌ Lỗi xóa tài khoản: %v",
  "status_delete_success": "🗑 Đã xóa tài khoản %s khỏi hệ thống.",
  "status_mask_on": "🔒 Đã ẩn thông tin tài khoản",
  "status_mask_off": "🔓 Đã hiển thị đầy đủ tài khoản",
  "status_auto_on": "Tự động chuyển tài khoản: BẬT",
  "status_auto_off": "Tự động chuyển tài khoản: TẮT",

  "modal_add_title": "Thêm tài khoản",
  "modal_add_header": "Thêm tài khoản",
  "modal_add_close": "Đóng",
  "modal_scan_chrome": "Quét Chrome",
  "modal_scanning_chrome": "⏳ Đang quét...",
  "modal_empty_candidates": "Không có tài khoản nào trong danh bạ.",
  "modal_btn_add_item": "THÊM",

  "confirm_delete_title": "Xác nhận xóa tài khoản",
  "confirm_btn_delete": "Xóa",
  "confirm_btn_cancel": "Hủy",
  "confirm_msg_disc_exists": "Bạn có chắc chắn muốn xóa tài khoản %s khỏi danh bạ đã quét không?\n\n(Lưu ý: Thao tác này chỉ xóa dữ liệu trong danh bạ khám phá, không xóa tài khoản hay token trên màn hình chính).",
  "confirm_msg_disc_no_file": "Tài khoản %s không có tệp dữ liệu trên hệ thống.\nBạn có muốn gỡ bỏ tài khoản này khỏi danh sách hiển thị không?",
  "confirm_msg_main_no_token": "Bạn có chắc chắn muốn gỡ tài khoản %s khỏi màn hình chính không?",
  "confirm_msg_main_has_token": "Bạn có chắc chắn muốn xóa tài khoản %s khỏi hệ thống? Thao tác này sẽ xóa token đã lưu và bạn sẽ cần đăng nhập lại.",

  "tray_open": "Hiển thị Switcher",
  "tray_refresh": "Làm mới Chrome",
  "tray_exit": "Thoát"
}
```

### 3.2. File `internal/i18n/locales/en.json`
```json
{
  "app_title": "Antigravity Multi-Account Switcher",
  "lang_dropdown_label": "EN",

  "btn_add": "Add",
  "btn_refresh_list": "Refresh List",
  "btn_refresh_list_loading": "⏳ Refreshing...",
  "btn_refresh_quota": "Refresh Quota",
  "btn_refresh_quota_loading": "⏳ Scanning...",
  "btn_always_on_top": "📌",
  "lbl_mask_accounts": "Mask Accounts:",
  "lbl_auto_switch": "Auto Switch:",

  "main_empty_accounts": "No Google accounts found in Chrome.",
  "badge_active": "[✔ Active]",
  "badge_switch": "[Switch]",
  "badge_need_oauth": "[Need OAuth]",
  "badge_added": "[Added]",
  "badge_account_not_found": "[Account Not Found]",

  "quota_col_5h": "5h",
  "quota_col_week": "Week",
  "quota_ready": "(Ready)",
  "quota_unavailable": "N/A",

  "server_connecting": "Antigravity: Active | Connecting...",
  "server_connected": "Antigravity: Active | PID %d | Port %d",
  "server_stopped": "Antigravity: Inactive | WinCred Fallback",

  "status_no_candidate": "⚠️ No accounts available to switch.",
  "status_fetch_error": "Failed to get account list: %s",
  "status_fetch_success": "Account list updated",
  "status_switching_to": "⏳ Switching to %s...",
  "status_switched_success": "✅ Switched to %s",
  "status_update_error": "❌ Failed to update status: %v",
  "status_cannot_delete_active": "⚠️ Cannot delete active account.",
  "status_delete_error": "❌ Failed to delete account: %v",
  "status_delete_success": "🗑 Removed account %s from system.",
  "status_mask_on": "🔒 Account information masked",
  "status_mask_off": "🔓 Full account information shown",
  "status_auto_on": "Auto-switch accounts: ON",
  "status_auto_off": "Auto-switch accounts: OFF",

  "modal_add_title": "Add Account",
  "modal_add_header": "Add Account",
  "modal_add_close": "Close",
  "modal_scan_chrome": "Scan Chrome",
  "modal_scanning_chrome": "⏳ Scanning...",
  "modal_empty_candidates": "No accounts found in catalog.",
  "modal_btn_add_item": "ADD",

  "confirm_delete_title": "Confirm Account Deletion",
  "confirm_btn_delete": "Delete",
  "confirm_btn_cancel": "Cancel",
  "confirm_msg_disc_exists": "Are you sure you want to remove account %s from the discovered catalog?\n\n(Note: This only deletes catalog entry, not main accounts or tokens).",
  "confirm_msg_disc_no_file": "Account %s has no data file on the system.\nDo you want to remove this account from the display list?",
  "confirm_msg_main_no_token": "Are you sure you want to remove account %s from the main screen?",
  "confirm_msg_main_has_token": "Are you sure you want to delete account %s from the system? This will delete the stored token and require logging in again.",

  "tray_open": "Show Switcher",
  "tray_refresh": "Refresh Chrome",
  "tray_exit": "Exit"
}
```

---

# PHẦN 4: ĐẶC TẢ TỪNG BƯỚC THI CÔNG TUẦN TỰ KÈM CỔNG KIỂM SOÁT (GATED EXECUTION SPECIFICATION)

> **QUY TẮC BẮT BUỘC DÀNH CHO CODER:**
> Tuyệt đối không được nhảy cóc. Thi công xong Bước nào thì phải chạy lệnh kiểm thử (Gate Command) của Bước đó. Chỉ khi kiểm thử đạt 100% **PASS**, Coder mới được phép sang Bước tiếp theo!

```text
 ┌────────────────────────────────────────────────────────┐
 │ BƯỚC 1: Tạo cặp file JSON song song (vi.json & en.json) │
 └───────────────────────────┬────────────────────────────┘
                             │ GATE 1: Parse JSON hợp lệ 100%
                             ▼
 ┌────────────────────────────────────────────────────────┐
 │ BƯỚC 2: Cập nhật dictionary.go với //go:embed & Test   │
 └───────────────────────────┬────────────────────────────┘
                             │ GATE 2: TestI18n_KeyParity PASS 100%
                             ▼
 ┌────────────────────────────────────────────────────────┐
 │ BƯỚC 3: Chuẩn hóa Quota Matrix & Countdown (quota_badge)│
 └───────────────────────────┬────────────────────────────┘
                             │ GATE 3: Test Quota Matrix PASS
                             ▼
 ┌────────────────────────────────────────────────────────┐
 │ BƯỚC 4: Chuẩn hóa Modals & Popups (add_modal.go, app.go)│
 └───────────────────────────┬────────────────────────────┘
                             │ GATE 4: Test Modal & Confirmations PASS
                             ▼
 ┌────────────────────────────────────────────────────────┐
 │ BƯỚC 5: Chuẩn hóa Thanh Đáy, Toolbar states & Tray Menu│
 └───────────────────────────┬────────────────────────────┘
                             │ GATE 5: Test Footer & Acceptance PASS
                             ▼
 ┌────────────────────────────────────────────────────────┐
 │ BƯỚC 6: Đồng bộ hóa toàn diện & Biên dịch nhị phân     │
 └────────────────────────────────────────────────────────┘
```

---

### BƯỚC 1: Tạo thư mục và cặp file JSON song song
- **Mục tiêu:** Tạo `internal/i18n/locales/vi.json` và `internal/i18n/locales/en.json` với chính xác 45 keys như đã định nghĩa ở Phần 3.
- **Tập tin thao tác:**
  1. `internal/i18n/locales/vi.json` (Tạo mới)
  2. `internal/i18n/locales/en.json` (Tạo mới)
- **CỔNG KIỂM SOÁT BƯỚC 1 (GATE 1):**
  Chạy lệnh kiểm tra cú pháp JSON hợp lệ trong PowerShell:
  ```powershell
  Get-Content internal/i18n/locales/vi.json | ConvertFrom-Json
  Get-Content internal/i18n/locales/en.json | ConvertFrom-Json
  ```
  *Điều kiện vượt cổng:* Cả 2 lệnh trả về đối tượng JSON thành công không có lỗi cú pháp.

---

### BƯỚC 2: Nâng cấp `internal/i18n/dictionary.go` nạp từ `//go:embed` & Viết Key-Parity Test
- **Mục tiêu:** 
  1. Khai báo struct `Dictionary` đầy đủ 45 trường kiểu `string` tương ứng chính xác với 45 key JSON (sử dụng tag `json:"<key_name>"`).
  2. Dùng `//go:embed locales/*.json` để nạp nhị phân.
  3. Hàm `init()` unmarshal dữ liệu JSON vào biến `viDictionary` và `enDictionary`. Nếu unmarshal thất bại, gọi panic để chặn đứng lỗi lúc khởi động.
  4. Trong `internal/i18n/i18n_test.go`: Bổ sung bài test `TestI18n_JSONKeyParity`: Duyệt đọc trực tiếp cả 2 file JSON, kiểm tra:
     - Số lượng key ở `vi.json` bằng số lượng key ở `en.json`.
     - Từng key trong `vi.json` đều phải có mặt trong `en.json` và ngược lại.
     - Không có bất kỳ value nào bị rỗng `""`.
- **Tập tin thao tác:**
  1. `internal/i18n/dictionary.go` (Sửa đổi)
  2. `internal/i18n/i18n_test.go` (Sửa đổi)
- **CỔNG KIỂM SOÁT BƯỚC 2 (GATE 2):**
  Chạy lệnh test i18n package:
  ```powershell
  go test -v ./internal/i18n/...
  ```
  *Điều kiện vượt cổng:* 100% các bài test trong package `internal/i18n` đều `PASS`.

---

### BƯỚC 3: Chuẩn hóa Quota Matrix & Bộ đếm thời gian trong `internal/ui/quota_badge.go`
- **Mục tiêu:**
  1. Thay thế hardcode `"Week"` bằng `i18n.Current().QuotaColWeek`.
  2. Thay thế hardcode `"(Ready)"` trong `formatCountdown()` bằng `i18n.Current().QuotaReady`.
  3. Bổ sung phương thức `UpdateLanguage(palette ThemePalette)` cho bảng Quota Table để cập nhật nhãn cột `Week/Tuần` khi chuyển đổi ngôn ngữ.
- **Tập tin thao tác:**
  1. `internal/ui/quota_badge.go` (Sửa đổi)
- **CỔNG KIỂM SOÁT BƯỚC 3 (GATE 3):**
  Chạy kiểm thử hiển thị quota:
  ```powershell
  go test -v ./internal/ui -run TestCreateMatrixQuotaTable
  ```
  *Điều kiện vượt cổng:* Toàn bộ các test liên quan tới Quota Matrix đều `PASS`.

---

### BƯỚC 4: Chuẩn hóa Modal "Thêm tài khoản" & Toàn bộ Popups Xác nhận
- **Mục tiêu:**
  1. Trong `internal/ui/add_modal.go`:
     - Tiêu đề modal và header: lấy từ `d.ModalAddTitle` và `d.ModalAddHeader`.
     - Nút Quét Chrome: `d.ModalScanChrome`, lúc đang quét: `d.ModalScanningChrome`.
     - Thông báo rỗng: `d.ModalEmptyCandidates`.
     - Các badge trạng thái: `d.BadgeActive`, `d.BadgeAdded`, `d.BadgeNeedOAuth`, `d.BadgeAccountNotFound`.
     - Nút hành động thêm: `d.ModalBtnAddItem`.
     - Popup xác nhận `showDeleteCandidateConfirm`: Lấy tiêu đề từ `d.ConfirmDeleteTitle`, nút xác nhận `d.ConfirmBtnDelete`, nút hủy `d.ConfirmBtnCancel`, nội dung từ `d.ConfirmMsgDiscExists` và `d.ConfirmMsgDiscNoFile`.
  2. Trong `internal/ui/app.go` (`ShowRemoveAccountConfirmation`):
     - Lấy tiêu đề từ `d.ConfirmDeleteTitle`, nút xác nhận `d.ConfirmBtnDelete`, nút hủy `d.ConfirmBtnCancel`.
     - Lấy nội dung thông báo từ `d.ConfirmMsgMainNoToken` và `d.ConfirmMsgMainHasToken`.
- **Tập tin thao tác:**
  1. `internal/ui/add_modal.go` (Sửa đổi)
  2. `internal/ui/app.go` (Sửa đổi)
- **CỔNG KIỂM SOÁT BƯỚC 4 (GATE 4):**
  Chạy kiểm thử modal và xác nhận xóa:
  ```powershell
  go test -v ./internal/ui -run "TestAddModal|TestAccountDeletion"
  ```
  *Điều kiện vượt cổng:* 100% các bài test `PASS`.

---

### BƯỚC 5: Chuẩn hóa Thanh Đáy, Toolbar Loading States, Status Bar & Tray Menu
- **Mục tiêu:**
  1. Toolbar loading/reset trong `internal/ui/app.go`:
     - Bấm Quota: gán text `d.BtnRefreshQuotaLoading`, hoàn tất gán lại `d.BtnRefreshQuota`.
     - Bấm Làm mới danh sách: gán text `d.BtnRefreshListLoading`, hoàn tất gán lại `d.BtnRefreshList`.
  2. Thanh đáy trong `internal/ui/window.go` và `internal/ui/app.go`:
     - Khởi tạo `serverStatusLabel`: dùng `d.ServerConnecting`.
     - Khi chạy: dùng format `fmt.Sprintf(d.ServerConnected, pid, port)`.
     - Khi dừng: dùng `d.ServerStopped`.
     - Khi `overview` rỗng ở màn hình chính: dùng `d.MainEmptyAccounts`.
  3. Status Bar trong `internal/ui/app.go`:
     - Toàn bộ các câu `a.showStatusBar(...)` được thay bằng `d.Status*` tương ứng (ví dụ: `d.StatusNoCandidate`, `d.StatusCannotDeleteActive`, `fmt.Sprintf(d.StatusSwitchingTo, email)`...).
  4. Tray Menu trong `internal/ui/tray.go`:
     - Menu items lấy text từ `d.TrayOpen`, `d.TrayRefresh`, `d.TrayExit`.
     - Trong `App.SetLanguage()`, cập nhật lại menu khay hệ thống theo ngôn ngữ mới.
- **Tập tin thao tác:**
  1. `internal/ui/window.go` (Sửa đổi)
  2. `internal/ui/app.go` (Sửa đổi)
  3. `internal/ui/tray.go` (Sửa đổi)
- **CỔNG KIỂM SOÁT BƯỚC 5 (GATE 5):**
  Chạy kiểm thử thanh đáy và acceptance i18n:
  ```powershell
  go test -v ./internal/ui -run "TestAcceptance_i18n|TestAcceptance_TC"
  ```
  *Điều kiện vượt cổng:* 100% test `PASS`.

---

### BƯỚC 6: Đồng bộ hóa toàn diện, Kiểm thử toàn hệ thống & Biên dịch nhị phân
- **Mục tiêu:**
  1. Chạy toàn bộ test suite của dự án (`go test ./internal/...`).
  2. Đảm bảo mọi bài test cũ kiểm tra tiếng Việt vẫn pass 100% nhờ cơ chế default locale `"vi"`.
  3. Biên dịch binary sạch sẽ qua script `scripts/build_windows.ps1`.
- **CỔNG KIỂM SOÁT BƯỚC 6 (GATE 6 - FINAL GATE):**
  Thực thi lệnh kiểm thử toàn diện:
  ```powershell
  go test ./internal/...
  powershell -ExecutionPolicy Bypass -File scripts/build_windows.ps1
  ```
  *Điều kiện vượt cổng:*
  - Toàn bộ các package `automation`, `connector`, `i18n`, `manager`, `model`, `reloader`, `scanner`, `store`, `telemetry`, `ui`, `version`, `wincred` đều trả về `ok`.
  - Binary `bin/antigravity-switch.exe` được biên dịch thành công (kèm PE resource icon 32x32 verified).

---

# PHẦN 5: BẢN ĐỒ VÙNG ẢNH HƯỞNG (TOUCH-MAP)

| Đường dẫn tập tin | Trách nhiệm thay đổi |
|---|---|
| `internal/i18n/locales/vi.json` | **Tạo mới:** Khai báo 45 key-value Tiếng Việt thuần túy, sạch sẽ |
| `internal/i18n/locales/en.json` | **Tạo mới:** Khai báo 45 key-value Tiếng Anh bản ngữ, chuẩn xác 100% |
| `internal/i18n/dictionary.go` | **Sửa đổi:** Nhúng JSON bằng `//go:embed`, unmarshal vào `Dictionary` struct |
| `internal/i18n/i18n_test.go` | **Sửa đổi:** Bổ sung Key-Parity Test đối soát 1:1 giữa `vi.json` và `en.json` |
| `internal/ui/quota_badge.go` | **Sửa đổi:** Dùng `QuotaColWeek` và `QuotaReady` thay cho chuỗi hardcode |
| `internal/ui/add_modal.go` | **Sửa đổi:** Chuyển 100% chuỗi modal, badge và popup sang `i18n.Current()` |
| `internal/ui/window.go` | **Sửa đổi:** Liên kết `serverStatusLabel` khởi tạo với `i18n.Current()` |
| `internal/ui/app.go` | **Sửa đổi:** Chuyển đổi toàn bộ thông báo StatusBar, Loading/Reset buttons sang `i18n` |
| `internal/ui/tray.go` | **Sửa đổi:** Kết nối các mục Tray Menu với từ điển i18n |
| `internal/ui/i18n_acceptance_test.go` | **Sửa đổi:** Mở rộng test kiểm tra in-place live switch trên cả modal và thanh đáy |

---

# PHẦN 6: KẾT LUẬN & ĐỀ XUẤT HÀNH ĐỘNG

Báo cáo đặc tả kiến trúc này đã thiết lập một hệ thống khép kín:
1. Đầy đủ dữ liệu chuỗi 2 ngôn ngữ sạch sẽ (Phần 3).
2. Quy trình kiểm soát chất lượng từng bước có cổng kiểm soát chặt chẽ (Phần 4).
3. Đảm bảo Coder nhận việc chỉ việc làm đúng theo từng bước, không phải tự phỏng đoán hay suy nghĩ thêm bất kỳ chi tiết nào.

Báo cáo này được lưu trữ độc quyền tại:
`reports/system-architect/rp_system-architect_261024_152500_by_antigravity_complete_bilingual_i18n_dual_json_parity_and_full_ui_coverage.md`

Kính trình **Human Owner** duyệt lại toàn bộ nội dung. Nếu Owner đồng ý hoặc có chỉ đạo thêm, xin cho ý kiến!
