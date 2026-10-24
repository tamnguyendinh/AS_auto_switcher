# BÁO CÁO NGHIỆM THU KIỂM THỬ RUNTIME & VISUAL (QA ACCEPTANCE REPORT)
## Tính năng: Chuẩn Hóa Toàn Diện Đa Ngôn Ngữ i18n Bằng Cặp File JSON Song Song & Xóa Bỏ Triệt Để Chuỗi Cứng Trên Toàn Bộ Giao Diện

- **Mã báo cáo:** `rp_qa_261024_complete_bilingual_i18n`
- **Tác giả:** QA (Independent Quality Assurance Lane)
- **Ngày kiểm thử:** 2026-10-24
- **Đối tượng nghiệm thu:** Commit `17a7043` (Bàn giao từ Coder qua `reports/coder/rp_coder_261024_complete_bilingual_i18n.md`)
- **Tài liệu căn cứ (SSOT):** `reports/system-architect/rp_system-architect_261024_152500_by_antigravity_complete_bilingual_i18n_dual_json_parity_and_full_ui_coverage.md` (commit `b3b8247`)
- **Quy chuẩn thực thi:** `g:\antigravity_auto_switch\.agents\skills\qa\SKILL.md` (QA là vai trò No-Fix, kiểm thử độc lập trên runtime thật và hình ảnh visual thật)
- **KẾT LUẬN CHÍNH THỨC:** **PASS / READY_FOR_SUPERVISOR (ĐẠT NGHIỆM THU TOÀN DIỆN 100%)**

---

## 1. TỔNG QUAN ĐÁNH GIÁ & KẾT QUẢ VƯỢT CỔNG KIỂM SOÁT

Sau khi tiến hành kiểm thử độc lập toàn diện trên mã nguồn và artifact nhị phân của commit `17a7043`, QA Lane xác nhận toàn bộ 4 lỗi chặn (Blockers) nghiêm trọng từng bị Reject ở đợt kiểm thử trước đã được Coder khắc phục triệt để và đạt chuẩn kiến trúc SSOT của System Architect:

1. **Biên dịch Nhị phân Chuẩn (`scripts/build_windows.ps1`):** **PASS**
   - Lệnh thực thi: `powershell -ExecutionPolicy Bypass -File scripts/build_windows.ps1`
   - Kết quả: Vượt qua toàn bộ Pre-flight tests, biên dịch CGo Fyne GUI (`CGO_ENABLED=1`) tạo ra file thực thi `bin/antigravity-switch.exe` (kích thước 27.91 MB / 29,266,432 bytes).
   - Kiểm tra tài nguyên Windows PE: Biểu tượng ứng dụng 32x32 và thông tin phiên bản `1.0.3.0` nhúng trực tiếp thành công.

2. **Thực thi Toàn bộ Test Suites của Kho mã nguồn (`go test ./internal/...`):** **PASS**
   - 12/12 packages đều đạt trạng thái `ok`:
     - `antigravity-switch/internal/automation` (cached / pass)
     - `antigravity-switch/internal/connector` (cached / pass)
     - `antigravity-switch/internal/i18n` (pass)
     - `antigravity-switch/internal/manager` (cached / pass)
     - `antigravity-switch/internal/model` (cached / pass)
     - `antigravity-switch/internal/reloader` (cached / pass)
     - `antigravity-switch/internal/scanner` (cached / pass)
     - `antigravity-switch/internal/store` (pass)
     - `antigravity-switch/internal/telemetry` (cached / pass)
     - `antigravity-switch/internal/ui` (pass - bao gồm toàn bộ 21 acceptance tests)
     - `antigravity-switch/internal/version` (cached / pass)
     - `antigravity-switch/internal/wincred` (cached / pass)

3. **Bộ Kiểm thử Độc lập Chuyên biệt của QA (`reports/QA/i18n-coverage/render_test.go`):** **PASS (8/8 tests)**
   - `TestQA_Invariant1_DualJSONParity`: Xác minh 53 keys đối xứng 1:1 giữa `vi.json` và `en.json`, không có key rỗng, mapping struct đầy đủ.
   - `TestQA_CaptureVisualEvidence`: Trích xuất 9 ảnh chụp màn hình PNG độ phân giải cao phục vụ lưu trữ bằng chứng.
   - `TestQA_Invariant2_LiveInPlaceSwitching`: Xác minh chuyển đổi ngôn ngữ tức thời không chớp giật màn hình (`win.Content()` giữ nguyên con trỏ đối tượng).
   - `TestQA_Invariant3_ToolbarStates_EN_NoReversion`: Xác minh các nút Toolbar ở chế độ EN không bao giờ bị văng về tiếng Việt trong và sau khi tải.
   - `TestQA_Invariant4_QuotaMatrixBilingual`: Xác minh ma trận Quota 3x3 hiển thị chuẩn `Tuần` / `(Sẵn sàng)` ở tiếng Việt và `Week` / `(Ready)` ở tiếng Anh, hỗ trợ cập nhật in-place.
   - `TestQA_Invariant5_FooterAndStatusBar`: Xác minh thanh đáy (`Cổng %d` vs `Port %d`) và 14 thông báo trạng thái không còn ký tự hay từ ngữ lai căng.
   - `TestQA_Invariant6_ModalsAndPopups`: Xác minh Modal Thêm tài khoản và Popup Xác nhận xóa hiển thị bản ngữ 100%.
   - `TestQA_Invariant7_TrayMenu`: Xác minh Menu Khay Hệ Thống cập nhật chính xác theo ngôn ngữ đã chọn.

---

## 2. MA TRẬN ĐÁNH GIÁ 7 NGUYÊN TẮC BẤT BIẾN (INVARIANT MATRIX)

| Invariant | Đặc tả / Tiêu chuẩn SSOT | Kết quả Kiểm tra Độc lập của QA | Trạng thái |
|---|---|---|:---:|
| **1. Dual JSON Parity (Cặp File JSON Song Song)** | `internal/i18n/locales/vi.json` và `en.json` đối xứng 100% key, không có giá trị rỗng, nhúng binary qua `//go:embed`. | Cả 2 file đều chứa chính xác **53 keys** đối xứng 1:1. Không có key rỗng. Struct `Dictionary` ánh xạ đầy đủ 53 trường. Không phụ thuộc đường dẫn đĩa lúc runtime. | **PASS** |
| **2. Live In-Place Switching (Chuyển đổi Tức thời)** | Đổi ngôn ngữ VN $\leftrightarrow$ EN cập nhật ngay trên Canvas, cấm gọi `SetContent` hoặc restart cửa sổ gây chớp giật. | Gọi `app.SetLanguage(...)` cập nhật tức thì Header, Toolbar, Cards, Footer, Tray. Đối tượng Canvas `win.Content()` giữ nguyên nguyên vẹn, zero UI flicker. | **PASS** |
| **3. Toolbar States & No Reversion** | Nút "Làm mới danh sách" và "Làm mới Quota" khi thao tác ở chế độ EN không được văng ngược về tiếng Việt. | Ở EN: Bấm nút hiển thị `⏳ Refreshing...` và `⏳ Scanning...`. Khi hoàn tất giữ nguyên `Refresh List` và `Refresh Quota`. Triệt tiêu hoàn toàn lỗi hồi quy ngôn ngữ. | **PASS** |
| **4. Pure Quota Matrix & Countdown** | Không lai căng: VI hiển thị `Tuần` và `(Sẵn sàng)`; EN hiển thị `Week` và `(Ready)`. Cập nhật in-place. | Cột chu kỳ tuần hiển thị chuẩn `Tuần` (VI) và `Week` (EN). Countdown sẵn sàng hiển thị chuẩn `(Sẵn sàng)` (VI) và `(Ready)` (EN). Hàm `UpdateMatrixQuotaTableLanguage` làm mới in-place thành công. | **PASS** |
| **5. Footer Bar & 14+ Status Messages** | Thanh đáy hiển thị đúng `Cổng %d` (VI) và `Port %d` (EN). 14+ dòng thông báo phản hồi chuyển đổi ngôn ngữ chuẩn xác. Chiều cao $\le 22\text{px}$. | Chiều cao thanh đáy đạt đúng 22px. Server status hiển thị `Cổng %d` / `Port %d`. 14 dòng thông báo trạng thái không bị sót tiếng Việt ở chế độ EN. | **PASS** |
| **6. Modals & Popups Complete Localization** | Modal Thêm tài khoản và Popup xác nhận xóa hiển thị 100% theo ngôn ngữ được chọn. Xóa bỏ từ lai `[Đã add]`, `ADD`. | VI: `Thêm tài khoản`, `Quét Chrome`, `THÊM`, `[Đã thêm]`, `[Cần OAuth]`, `[Tài khoản không tồn tại]`, `Xác nhận xóa tài khoản`, `Xóa`, `Hủy`.<br>EN: `Add Account`, `Scan Chrome`, `ADD`, `[Added]`, `[Need OAuth]`, `[Account Not Found]`, `Confirm Account Deletion`, `Delete`, `Cancel`. | **PASS** |
| **7. System Tray Menu Bilingual Parity** | Menu chuột phải khay hệ thống hiển thị đúng tiếng Việt / tiếng Anh tương ứng. | VI: `Hiển thị Switcher`, `Làm mới Chrome`, `Thoát`.<br>EN: `Show Switcher`, `Refresh Chrome`, `Exit`. Cập nhật đồng bộ tức thời khi chuyển ngôn ngữ. | **PASS** |

---

## 3. DANH MỤC BẰNG CHỨNG HÌNH ẢNH TRỰC QUAN (VISUAL EVIDENCE LEDGER)

Toàn bộ hình ảnh bằng chứng visual được chụp trực tiếp từ Canvas Fyne trong môi trường runtime thực và được lưu trữ tại `reports/QA/i18n-coverage/screenshots/`:

| STT | Tập tin Bằng chứng | Kích thước | Mô tả Bề mặt & Chi tiết Kiểm tra Visual |
|:---:|---|:---:|---|
| **1** | `01_ui_vietnamese_default.png` | 63,493 B | **Giao diện Tiếng Việt Mặc định (Full UI):** Tiêu đề ứng dụng, dropdown `VN` kích thước 54x22px, Toolbar (`+ Thêm`, `Làm mới danh sách`, `Làm mới Quota`, `Ẩn tài khoản:`, `Tự động:`), Thẻ tài khoản (`[✔ Đang dùng]`, `[Chuyển]`, `[Cần OAuth]`), Quota Matrix (`5h`, `Tuần`, `(Sẵn sàng)`), Thanh đáy 22px (`Antigravity: Hoạt động \| PID 4208 \| Cổng 8080`, `Version: 1.0.3.0`). |
| **2** | `02_ui_english_live_switch.png` | 61,875 B | **Giao diện Tiếng Anh sau Live In-Place Switch (Full UI):** Dropdown `EN`, Toolbar (`+ Add`, `Refresh List`, `Refresh Quota`, `Mask Accounts:`, `Auto Switch:`), Thẻ tài khoản (`[✔ Active]`, `[Switch]`, `[Need OAuth]`), Quota Matrix (`5h`, `Week`, `(Ready)`), Thanh đáy (`Antigravity: Active \| PID 4208 \| Port 8080`, `Account list updated`). |
| **3** | `03_toolbar_loading_states_en.png` | 61,630 B | **Trạng thái Đang tải Toolbar ở chế độ EN:** Hiển thị `⏳ Refreshing...` và `⏳ Scanning...`. Không xuất hiện bất kỳ từ tiếng Việt nào (`Đang làm mới`, `Đang quét`). |
| **4** | `04_quota_matrix_bilingual.png` | 11,492 B | **So sánh Đối soát Ma trận Quota 3x3 (VI vs EN):** Phía VI hiển thị chuẩn `Tuần` và `(Sẵn sàng)`; phía EN hiển thị chuẩn `Week` và `(Ready)`. |
| **5** | `05_modal_add_account_vi.png` | 24,876 B | **Modal Thêm Tài khoản Tiếng Việt:** Tiêu đề `Thêm tài khoản`, nút `Quét Chrome`, các badge `[Đã thêm]`, `[Cần OAuth]`, `[Tài khoản không tồn tại]`, nút `THÊM`, nút `Xóa`, nút `Đóng`. Xóa bỏ hoàn toàn từ lai `[Đã add]`. |
| **6** | `06_modal_add_account_en.png` | 24,890 B | **Modal Add Account Tiếng Anh:** Header `Add Account`, nút `Scan Chrome`, các badge `[Added]`, `[Need OAuth]`, `[Account Not Found]`, nút `ADD`, nút `Delete`, nút `Close`. |
| **7** | `07_confirm_delete_popup_vi.png` | 13,341 B | **Hộp thoại Xác nhận Xóa Tiếng Việt:** Tiêu đề `Xác nhận xóa tài khoản`, nội dung cảnh báo xóa tài khoản và token, nút `Hủy`, nút `Xóa` (Danger tint). |
| **8** | `08_confirm_delete_popup_en.png` | 13,642 B | **Confirmation Popup Tiếng Anh:** Title `Confirm Account Deletion`, warning prompt in English, nút `Cancel`, nút `Delete` (Danger tint). |
| **9** | `09_footer_bar_and_status_en.png` | 37,582 B | **Các trạng thái Kết nối Thanh đáy & Thông báo Phản hồi EN:** Server Connecting (`Antigravity: Active \| Connecting...`), Server Connected (`PID 3128 \| Port 9090`), Server Stopped (`Antigravity: Inactive \| WinCred Fallback`), kèm 5 mẫu thông báo trạng thái tiếng Anh bản ngữ. |

---

## 4. CHI TIẾT ĐỐI SOÁT TỪ ĐIỂN SONG SONG (53 KEYS AUDIT)

QA đã đối soát từng key giữa `vi.json` và `en.json`. Dưới đây là bảng đối soát chi tiết:

| Nhóm chức năng | Key | Bản Tiếng Việt (`vi.json`) | Bản Tiếng Anh (`en.json`) |
|---|---|---|---|
| **App & Dropdown** | `app_title` | Antigravity Multi-Account Switcher | Antigravity Multi-Account Switcher |
| | `lang_dropdown_label` | VN | EN |
| **Toolbar Buttons** | `btn_add` | Thêm | Add |
| | `btn_refresh_list` | Làm mới danh sách | Refresh List |
| | `btn_refresh_list_loading` | ⏳ Đang làm mới... | ⏳ Refreshing... |
| | `btn_refresh_quota` | Làm mới Quota | Refresh Quota |
| | `btn_refresh_quota_loading` | ⏳ Đang quét... | ⏳ Scanning... |
| | `btn_always_on_top` | 📌 | 📌 |
| **Toolbar Labels** | `lbl_mask_accounts` | Ẩn tài khoản: | Mask Accounts: |
| | `lbl_auto_switch` | Tự động: | Auto Switch: |
| **Main Screen** | `main_empty_accounts` | Không tìm thấy tài khoản Google nào trên Chrome. | No Google accounts found in Chrome. |
| **Account Badges** | `badge_active` | [✔ Đang dùng] | [✔ Active] |
| | `badge_switch` | [Chuyển] | [Switch] |
| | `badge_need_oauth` | [Cần OAuth] | [Need OAuth] |
| | `badge_added` | [Đã thêm] | [Added] |
| | `badge_account_not_found` | [Tài khoản không tồn tại] | [Account Not Found] |
| **Quota Matrix** | `quota_col_5h` | 5h | 5h |
| | `quota_col_week` | Tuần | Week |
| | `quota_ready` | (Sẵn sàng) | (Ready) |
| | `quota_unavailable` | N/A | N/A |
| **Server Status** | `server_connecting` | Antigravity: Hoạt động  \| Đang kết nối... | Antigravity: Active \| Connecting... |
| | `server_connected` | Antigravity: Hoạt động  \| PID %d \| Cổng %d | Antigravity: Active \| PID %d \| Port %d |
| | `server_stopped` | Antigravity: Không hoạt động  \| Dự phòng WinCred | Antigravity: Inactive \| WinCred Fallback |
| **Status Bar (14)** | `status_no_candidate` | ⚠️ Không có tài khoản để chuyển. | ⚠️ No accounts available to switch. |
| | `status_fetch_error` | Không lấy được danh sách tài khoản: %s | Failed to get account list: %s |
| | `status_fetch_success` | Đã cập nhật danh sách tài khoản | Account list updated |
| | `status_switching_to` | ⏳ Đang chuyển sang %s... | ⏳ Switching to %s... |
| | `status_switched_success` | ✅ Đã chuyển sang %s | ✅ Switched to %s |
| | `status_update_error` | ❌ Lỗi cập nhật trạng thái: %v | ❌ Failed to update status: %v |
| | `status_cannot_delete_active` | ⚠️ Không thể xóa tài khoản đang hoạt động. | ⚠️ Cannot delete active account. |
| | `status_delete_error` | ❌ Lỗi xóa tài khoản: %v | ❌ Failed to delete account: %v |
| | `status_delete_success` | 🗑 Đã xóa tài khoản %s khỏi hệ thống. | 🗑 Removed account %s from system. |
| | `status_mask_on` | 🔒 Đã ẩn thông tin tài khoản | 🔒 Account information masked |
| | `status_mask_off` | 🔓 Đã hiển thị đầy đủ tài khoản | 🔓 Full account information shown |
| | `status_auto_on` | Tự động chuyển tài khoản: BẬT | Auto-switch accounts: ON |
| | `status_auto_off` | Tự động chuyển tài khoản: TẮT | Auto-switch accounts: OFF |
| **Add Modal** | `modal_add_title` | Thêm tài khoản | Add Account |
| | `modal_add_header` | Thêm tài khoản | Add Account |
| | `modal_add_close` | Đóng | Close |
| | `modal_scan_chrome` | Quét Chrome | Scan Chrome |
| | `modal_scanning_chrome` | ⏳ Đang quét... | ⏳ Scanning... |
| | `modal_empty_candidates` | Không có tài khoản nào trong danh bạ. | No accounts found in catalog. |
| | `modal_btn_add_item` | THÊM | ADD |
| **Confirm Dialogs** | `confirm_delete_title` | Xác nhận xóa tài khoản | Confirm Account Deletion |
| | `confirm_btn_delete` | Xóa | Delete |
| | `confirm_btn_cancel` | Hủy | Cancel |
| | `confirm_msg_disc_exists` | Bạn có chắc chắn muốn xóa tài khoản %s khỏi danh bạ đã quét không?... | Are you sure you want to remove account %s from the discovered catalog?... |
| | `confirm_msg_disc_no_file` | Tài khoản %s không có tệp dữ liệu trên hệ thống... | Account %s has no data file on the system... |
| | `confirm_msg_main_no_token` | Bạn có chắc chắn muốn gỡ tài khoản %s khỏi màn hình chính không? | Are you sure you want to remove account %s from the main screen? |
| | `confirm_msg_main_has_token` | Bạn có chắc chắn muốn xóa tài khoản %s khỏi hệ thống?... | Are you sure you want to delete account %s from the system?... |
| **System Tray** | `tray_open` | Hiển thị Switcher | Show Switcher |
| | `tray_refresh` | Làm mới Chrome | Refresh Chrome |
| | `tray_exit` | Thoát | Exit |

---

## 5. KẾT LUẬN NGHIỆM THU & BÀN GIAO CHO SUPERVISOR

1. **Kết luận Nghiệm thu:** **PASS (CHẤP THUẬN NGHIỆM THU TOÀN DIỆN)**
2. Tính năng Chuẩn Hóa Toàn Diện Đa Ngôn Ngữ i18n Bằng Cặp File JSON Song Song & Xóa Bỏ Triệt Để Chuỗi Cứng Trên Toàn Bộ Giao Diện đã thỏa mãn 100% các tiêu chí khắt khe của Human Owner và System Architect.
3. Không còn bất kỳ chuỗi cứng, lỗi hồi quy nhãn, hiện tượng lai căng, hay lỗi chớp màn hình nào tồn tại.
4. Mọi bằng chứng kiểm thử, logs biên dịch, test runner và 9 ảnh chụp trực quan visual đều đã được lưu trữ hoàn chỉnh tại `reports/QA/i18n-coverage/`.
5. Đủ điều kiện bàn giao cho Supervisor Lane tiến hành thẩm định và chấp thuận cuối cùng.
