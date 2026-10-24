# BÁO CÁO NGHIỆM THU KIỂM THỬ RUNTIME & VISUAL (QA ACCEPTANCE REPORT)
## Tính năng: Dropdown Chuyển đổi Ngôn ngữ i18n (VN/EN) & Thu gọn 50% Độ cao Thanh Đáy (Footer Bar)

- **Mã báo cáo:** `rp_qa_261024_i18n_footer_reduction`
- **Tác giả:** QA (Independent Quality Assurance Lane)
- **Ngày kiểm thử:** 2026-10-24
- **Đối tượng nghiệm thu:** Commit `1e55b77` (Bàn giao từ Coder qua `rp_coder_261024_i18n_footer_reduction.md`)
- **Tài liệu căn cứ (SSOT):** `reports/system-architect/rp_system-architect_261024_130500_by_antigravity_i18n_dropdown_language_switcher_and_preferences_persistence.md`
- **Chỉ thị trực tiếp từ Human Owner:**
  1. *"tất cả cac nút, các dong trạng thai, cac thogn báo, ben trong cac tab các nut, ben ngaoig UI tông chua dịch haonf taonf day du Anh-Viet. rà saot kỹ và đanh reject đi"*
  2. *"ra saot ben trong từng tab, ben trign tung nut, rà soat rung modael, tung pop up, rà saot gun dong thogn bao, ra soat tung dong hein thị tren UI tông, liẹt ke taon bọi nhugn cho chau dịch du i18n"*
  3. *"yeu cầu moi ngon ngũ pahi co 1 file json để tra soat song song, nhu teh dịch moi chính xac và dung chẩun i18n đa ngon ngũ"*
- **KẾT LUẬN CHÍNH THỨC:** **REJECT (KHÔNG ĐẠT NGHIỆM THU - YÊU CẦU TÁI THIẾT KẾ & BỔ SUNG TOÀN DIỆN)**

---

## 1. TỔNG QUAN ĐÁNH GIÁ & CÁC LỖI CHẶN NGHIỆM THU (BLOCKERS)

Mặc dù giải pháp của Coder đã đạt được một số tiêu chí về mặt hình học (dropdown 54x22px đặt cạnh nút thu nhỏ, chiều cao thanh đáy đạt 22px), đợt kiểm thử độc lập của QA cùng chỉ thị rà soát của Owner phát hiện **4 lỗi nghiêm trọng (Critical Invariant Violations)** khiến phiên bản bàn giao không thể nghiệm thu:

1. **Thất bại trong quy trình biên dịch chuẩn (`scripts/build_windows.ps1`):**
   - Chạy `powershell -ExecutionPolicy Bypass -File scripts/build_windows.ps1` bị **FAIL** ở bước Pre-flight Test do lỗi race condition bất đồng bộ trong `TestAcceptance_i18n_LiveInPlaceSwitching`:
     ```text
     --- FAIL: TestAcceptance_i18n_LiveInPlaceSwitching (0.79s)
         i18n_acceptance_test.go:156: expected persisted language 'en', got 'vi'
         i18n_acceptance_test.go:186: expected persisted language 'vi', got 'en'
     ```
   - **Nguyên nhân cốt lõi:** Hàm `SetLanguage()` trong `internal/ui/app.go` ghi đĩa qua một goroutine ngầm không có cơ chế đồng bộ / flush (`go func() { ... SaveUIPreferences(...) }`). Dưới tải kiểm thử đầy đủ của repo, thao tác ghi chưa hoàn tất trước khi test đọc lại, dẫn đến tình trạng flaky test làm gãy build pipeline.

2. **Dịch thuật dở dang, hàng loạt bề mặt UI bị bỏ sót 100% (Hardcoded Vietnamese Strings):**
   - Tầng `Dictionary` chỉ khai báo 26 trường tĩnh, bỏ quên toàn bộ các dòng thông báo trạng thái (`showStatusBar`), hộp thoại xác nhận xóa tài khoản (`confirmDlg`), danh bạ tài khoản khám phá (`add_modal.go`), và nhãn trạng thái server thanh đáy.

3. **Lỗi hồi quy nhãn nút bấm sau khi thao tác (Button Label Reversion Defect):**
   - Khi đang ở ngôn ngữ Tiếng Anh (`EN`), nếu người dùng bấm nút **"Refresh Quota"** hoặc **"Refresh List"**, các hàm xử lý trong `app.go` (`RefreshAllQuota`, `Refresh`) đã gán cứng nhãn tiếng Việt trong các pha loading và kết thúc (`"⏳ Đang quét..."`, `"Làm mới Quota"`, `"⏳ Đang làm mới..."`, `"Làm mới danh sách"`). Nút bấm bị biến thành Tiếng Việt vĩnh viễn dù app đang ở chế độ Tiếng Anh!

4. **Vi phạm chuẩn kiến trúc i18n do Owner yêu cầu (Missing Parallel JSON Locale Files):**
   - Ứng dụng hiện đang hardcode từ điển trong struct mã nguồn Go (`internal/i18n/dictionary.go`).
   - Theo đúng yêu cầu kiến trúc của Owner: **"mỗi ngôn ngữ phải có 1 file JSON để tra soát song song"** (ví dụ: `locales/vi.json` và `locales/en.json`). Kiến trúc này cho phép đối soát ngữ nghĩa 1:1, kiểm tra độ phủ tự động, và không làm phình mã nguồn Go khi mở rộng ngôn ngữ.

---

## 2. MA TRẬN ĐÁNH GIÁ CÁC NGUYÊN TẮC BẤT BIẾN (INVARIANT MATRIX)

| Invariant | Đặc tả / Tiêu chuẩn | Kết quả QA | Trạng thái |
|---|---|---|:---:|
| **1. Header Layout & Geometry** | Dropdown VN/EN nằm tại `windowControls` bên trái `btnMin`, kích thước 54x22px, không vỡ baseline. | Khớp hoàn toàn: `GridWrap(54, 22)`, thứ tự `[Dropdown] -> [—] -> [□] -> [✕]`. | **PASS** |
| **2. Footer Height 50% Reduction** | Chiều cao thanh đáy $\le 25\text{px}$ (thực tế 22px), đệm ngang 8px hai bên, cấu trúc Separator + Content nguyên vẹn. | Khớp hoàn toàn: `MinSize().Height = 22.0px`, giữ đúng hợp đồng cây `TestAcceptance_TC02`. | **PASS** |
| **3. Live In-Place Switching** | Chuyển đổi ngôn ngữ tức thì không chớp giật canvas, cập nhật toàn bộ widget và cards. | **FAIL**: Hàng loạt widget không cập nhật (Server Status, Modal, Thông báo), nút bấm bị revert về Tiếng Việt khi click. | **FAIL** |
| **4. Preferences Persistence** | Lưu vào `ui_preferences.json`, nạp lại chính xác khi khởi động lại. | **FAIL**: Thao tác ghi bất đồng bộ không an toàn luồng kiểm thử, gây trượt test trong build script. | **FAIL** |
| **5. Default Fallback** | Tự động fallback về `"vi"` khi cấu hình rỗng hoặc không hợp lệ. | Nạp fallback chính xác về `"vi"`. | **PASS** |
| **6. Parallel JSON Locale Architecture** | Mỗi ngôn ngữ có 1 file JSON độc lập để tra soát song song (`vi.json`, `en.json`). | **FAIL**: Hoàn toàn chưa có tệp JSON nào, vẫn dùng struct cứng Go. | **FAIL** |

---

## 3. BẢNG TRA SOÁT FORENSIC CHI TIẾT TOÀN BỘ CÁC BỀ MẶT CHƯA ĐƯỢC DỊCH

Thực hiện chỉ thị rà soát kỹ từng tab, từng nút, từng modal, từng popup và từng dòng thông báo:

### 3.1. Bề mặt Nút Bấm & Hiện tượng Hồi quy (Buttons & Transient State Defects)
| File & Dòng | Thành phần UI | Chuỗi cứng hiện tại (Hardcoded VN) | Trạng thái khi chọn EN | Chuỗi chuẩn Tiếng Anh cần có |
|---|---|---|---|---|
| `internal/ui/app.go:771` | Nút Làm mới Quota (đang chạy) | `"⏳ Đang quét..."` | Bị đổi sang Tiếng Việt | `"⏳ Scanning..."` |
| `internal/ui/app.go:781` | Nút Làm mới Quota (hoàn tất) | `"Làm mới Quota"` | Bị đổi vĩnh viễn về VN | `"Refresh Quota"` |
| `internal/ui/app.go:818` | Nút Làm mới Quota (lỗi/timeout) | `"Làm mới Quota"` | Bị đổi vĩnh viễn về VN | `"Refresh Quota"` |
| `internal/ui/app.go:928` | Nút Làm mới danh sách (đang chạy) | `"⏳ Đang làm mới..."` | Bị đổi sang Tiếng Việt | `"⏳ Refreshing..."` |
| `internal/ui/app.go:937` | Nút Làm mới danh sách (hoàn tất) | `"Làm mới danh sách"` | Bị đổi vĩnh viễn về VN | `"Refresh List"` |
| `internal/ui/add_modal.go:113` | Nút thêm tài khoản ứng viên | `"ADD"` (chỉ có EN) | Không có bản Tiếng Việt | VN: `"Thêm"`, EN: `"Add"` |
| `internal/ui/add_modal.go:144` | Nút Quét Chrome trong Modal | `"Quét Chrome"` | Không dịch | `"Scan Chrome"` |
| `internal/ui/add_modal.go:146` | Nút Quét Chrome (đang quét) | `"⏳ Đang quét..."` | Không dịch | `"⏳ Scanning..."` |
| `internal/ui/add_modal.go:151` | Nút Quét Chrome (hoàn tất) | `"Quét Chrome"` | Không dịch | `"Scan Chrome"` |

### 3.2. Bề mặt Modal Khám phá & Danh bạ Tài khoản (`internal/ui/add_modal.go`)
| File & Dòng | Thành phần UI | Chuỗi cứng hiện tại (Hardcoded VN) | Chuỗi chuẩn Tiếng Anh (`EN`) |
|---|---|---|---|
| `add_modal.go:55` | Thông báo danh bạ trống | `"Không có tài khoản nào trong danh bạ."` | `"No accounts found in catalog."` |
| `add_modal.go:84` | Badge tài khoản không tồn tại | `"[Tài khoản không tồn tại]"` | `"[Account Not Found]"` |
| `add_modal.go:92` | Badge tài khoản đã có OAuth | `"[✔ Đã OAuth]"` | `"[✔ OAuth Ready]"` |
| `add_modal.go:100` | Badge tài khoản đã thêm vào UI | `"[Đã add]"` | `"[Added]"` |
| `add_modal.go:108` | Badge tài khoản chưa có token | `"[Cần OAuth]"` | `"[Need OAuth]"` |
| `add_modal.go:116` | Badge sau khi bấm ADD | `"[Đã add]"` | `"[Added]"` |
| `add_modal.go:159` | Tiêu đề Header trong Modal | `"Thêm tài khoản"` | `"Add Account"` |
| `add_modal.go:181` | Tiêu đề Dialog Custom | `"Thêm tài khoản"` | `"Add Account"` |
| `add_modal.go:181` | Nút đóng Modal Dialog | `"Đóng"` | `"Close"` |

### 3.3. Hộp thoại Popup Xác nhận Xóa Tài khoản (`Confirm Dialogs`)
| File & Dòng | Thành phần UI | Chuỗi cứng hiện tại (Hardcoded VN) | Chuỗi chuẩn Tiếng Anh (`EN`) |
|---|---|---|---|
| `add_modal.go:197` | Tiêu đề Popup xóa ứng viên | `"Xác nhận xóa tài khoản"` | `"Confirm Account Deletion"` |
| `add_modal.go:198` | Nội dung Popup xóa ứng viên | `"Bạn có chắc chắn muốn xóa tài khoản %s khỏi danh bạ đã quét không?..."` | `"Are you sure you want to remove account %s from discovered catalog?..."` |
| `add_modal.go:201` | Nội dung Popup khi file mất | `"Tài khoản %s không có tệp dữ liệu trên hệ thống..."` | `"Account %s data file is missing. Remove from list?"` |
| `add_modal.go:223` | Nút xác nhận xóa trong Modal | `"Xóa"` | `"Delete"` |
| `add_modal.go:224` | Nút hủy xóa trong Modal | `"Hủy"` | `"Cancel"` |
| `app.go:1235` | Tiêu đề Popup xóa thẻ chính | `"Xác nhận xóa tài khoản"` | `"Confirm Account Deletion"` |
| `app.go:1236` | Nội dung gỡ thẻ chưa có token | `"Bạn có chắc chắn muốn gỡ tài khoản %s khỏi màn hình chính không?"` | `"Are you sure you want to remove account %s from main window?"` |
| `app.go:1238` | Nội dung xóa tài khoản có token | `"Bạn có chắc chắn muốn xóa tài khoản %s khỏi hệ thống? Thao tác này sẽ xóa token đã lưu..."` | `"Are you sure you want to delete account %s from system? Saved tokens will be permanently removed."` |
| `app.go:1246` | Nút xác nhận xóa thẻ chính | `"Xóa"` | `"Delete"` |
| `app.go:1247` | Nút hủy xóa thẻ chính | `"Hủy"` | `"Cancel"` |

### 3.4. Dòng Trạng thái Server & Thanh Thông báo Đáy (`Footer Bar & Status Bar`)
| File & Dòng | Thành phần UI | Chuỗi cứng hiện tại (Hardcoded VN) | Chuỗi chuẩn Tiếng Anh (`EN`) |
|---|---|---|---|
| `window.go:139` | Nhãn Server khởi tạo | `"Antigravity: Hoạt động  | Đang kết nối..."` | `"Antigravity: Active | Connecting..."` |
| `app.go:1564` | Nhãn Server khi đang chạy | `fmt.Sprintf("Antigravity: Hoạt động  | PID %d | Port %d", ...)` | `fmt.Sprintf("Antigravity: Active | PID %d | Port %d", ...)` |
| `app.go:1566` | Nhãn Server khi dừng | `"Antigravity: Không hoạt động  | Dự phòng WinCred"` | `"Antigravity: Inactive | WinCred Fallback"` |
| `app.go:670` | Thông báo dừng an toàn (Safety Stop) | `"⚠️ Không có tài khoản để chuyển."` | `"⚠️ No accounts available to switch."` |
| `app.go:957` | Thông báo lỗi nạp danh sách | `"Không lấy được danh sách tài khoản: %v"` | `"Failed to retrieve account list: %v"` |
| `app.go:981` | Thông báo làm mới danh sách xong | `"Đã cập nhật danh sách tài khoản"` | `"Account list updated"` |
| `app.go:1008` | Thông báo đang chuyển tài khoản | `"⏳ Đang chuyển sang %s..."` | `"⏳ Switching to %s..."` |
| `app.go:1061` | Thông báo đã chuyển thành công | `"✅ Đã chuyển sang %s"` | `"✅ Switched to %s"` |
| `app.go:1120` | Thông báo lỗi cập nhật toggle | `"❌ Lỗi cập nhật trạng thái: %v"` | `"❌ Failed to update status: %v"` |
| `app.go:1214` | Thông báo chặn xóa Active | `"⚠️ Không thể xóa tài khoản đang hoạt động."` | `"⚠️ Cannot delete active account."` |
| `app.go:1276` | Thông báo lỗi xóa tài khoản | `"❌ Lỗi xóa tài khoản: %v"` | `"❌ Error deleting account: %v"` |
| `app.go:1299` | Thông báo đã xóa tài khoản | `"🗑 Đã xóa tài khoản %s khỏi hệ thống."` | `"🗑 Removed account %s from system."` |
| `app.go:1538` | Thông báo bật ẩn email | `"🔒 Đã ẩn thông tin tài khoản"` | `"🔒 Account privacy masking enabled"` |
| `app.go:1540` | Thông báo tắt ẩn email | `"🔓 Đã hiển thị đầy đủ tài khoản"` | `"🔓 Account privacy masking disabled"` |
| `app.go:1597` | Lưới tài khoản chính khi rỗng | `"Không tìm thấy tài khoản Google nào trên Chrome."` | `"No Google accounts found in Chrome."` |

### 3.5. Menu Khay Hệ thống (`System Tray Menu`)
| File & Dòng | Thành phần UI | Chuỗi cứng hiện tại (Hardcoded VN) | Chuỗi chuẩn Tiếng Anh (`EN`) |
|---|---|---|---|
| `internal/ui/tray.go:18` | Mục menu Hiển thị Switcher | `"Hiển thị Switcher"` | `"Open Switcher"` |
| `internal/ui/tray.go:27` | Mục menu Làm mới Chrome | `"Làm mới Chrome"` | `"Refresh Chrome"` |
| `internal/ui/tray.go:31` | Mục menu Thoát | `"Thoát"` | `"Exit"` |

*(Lưu ý: Khay hệ thống hiện chỉ tạo 1 lần khi khởi động, chưa có hàm lắng nghe để cập nhật lại nhãn menu khi chuyển ngôn ngữ).*

### 3.6. Thông điệp Tự động Chuyển đổi (`Automation Status Events`)
| File & Dòng | Thành phần UI | Chuỗi cứng hiện tại (Hardcoded VN) | Chuỗi chuẩn Tiếng Anh (`EN`) |
|---|---|---|---|
| `automation/controller.go:169` | Bật auto-switch | `"Tự động chuyển tài khoản: BẬT"` | `"Auto-switch accounts: ON"` |
| `automation/controller.go:172` | Tắt auto-switch | `"Tự động chuyển tài khoản: TẮT"` | `"Auto-switch accounts: OFF"` |
| `automation/controller.go:571` | Tự động chuyển thành công | `"Đã tự động chuyển thành công sang %s"` | `"Successfully auto-switched to %s"` |
| `automation/retry.go:247` | Thông báo Native Retry | `"Khôi phục bằng Native Retry lần %d/%d..."` | `"Recovering with Native Retry attempt %d/%d..."` |

---

## 4. BẰNG CHỨNG KIỂM THỬ THỊ GIÁC (VISUAL EVIDENCE ARTIFACTS)

Toàn bộ ảnh chụp visual và tệp log nghiệm thu đã được lưu trữ tập trung tại thư mục:
`reports/QA/i18n-footer/`

1. **`screenshots/01_ui_vietnamese_default.png`** (58.8 KB):
   - Minh chứng giao diện mặc định Tiếng Việt: Dropdown `VN`, Footer Bar 22px, Toolbar Tiếng Việt.
2. **`screenshots/02_ui_english_switched.png`** (60.6 KB):
   - Minh chứng giao diện sau khi chuyển sang Tiếng Anh (`EN`): Nút Toolbar chuyển sang Tiếng Anh, nhưng dòng trạng thái thanh đáy vẫn hiển thị chữ tiếng Việt `Hoạt động`.
3. **`screenshots/03_add_modal_hardcoded_vietnamese.png`** (78.7 KB):
   - Minh chứng Modal "Thêm tài khoản" mở ra khi đang ở chế độ Tiếng Anh: Toàn bộ tiêu đề, nút "Quét Chrome", các badge trạng thái ứng viên và nút xác nhận vẫn giữ 100% tiếng Việt chưa được dịch.
4. **`build.log`**:
   - Ghi lại vết đứt gãy của quy trình `scripts/build_windows.ps1` do lỗi race condition trong persistence test.
5. **`test-suites.log`**:
   - Ghi lại kết quả thực thi các bài kiểm thử chấp nhận.
6. **`results.json`**:
   - Bảng tổng hợp trạng thái nghiệm thu dạng máy đọc được.

---

## 5. YÊU CẦU KHẮC PHỤC DÀNH CHO CÁC LANE (REMEDIAL ACTION DIRECTIVES)

Để đạt nghiệm thu trong đợt tiếp theo, các lane cần thực hiện các yêu cầu kỹ thuật sau:

### 5.1. Dành cho System Architect (Thiết kế lại chuẩn i18n):
1. **Thiết lập kiến trúc Tệp JSON Song Song (Parallel JSON Architecture):**
   - Tạo thư mục `locales/` hoặc `internal/i18n/locales/` chứa hai tệp:
     - `vi.json`: Chứa 100% toàn bộ chuỗi Tiếng Việt của ứng dụng theo cấu trúc phân cấp (header, toolbar, cards, modals, status_bar, tray, automation).
     - `en.json`: Chứa 100% toàn bộ chuỗi Tiếng Anh tương ứng với các khóa đối xứng 1:1.
   - Viết test kiểm tra tính toàn vẹn (Key Symmetry Test): Tự động so sánh hai file JSON để đảm bảo không có khóa nào bị thiếu hoặc để rỗng.
2. **Thiết kế cơ chế Đồng bộ Flush Preferences:**
   - Cung cấp hàm `SaveUIPreferencesSync(path, pref)` hoặc bổ sung channel/waitgroup đảm bảo ghi đĩa hoàn tất, chấm dứt triệt để lỗi flaky test trong CI/build.
3. **Mở rộng cơ chế Re-render In-Place:**
   - Cập nhật cả `serverStatusLabel`, `trayMenu`, và các modal đang mở khi gọi `SetLanguage()`.

### 5.2. Dành cho Coder (Thi công khắc phục):
1. Đọc và nạp từ điển trực tiếp từ hai file JSON song song (sử dụng `//go:embed` để nhúng vào binary nhị phân duy nhất).
2. Thay thế toàn bộ 42 vị trí chuỗi cứng tiếng Việt đã được QA liệt kê ở Mục 3 thành các lệnh gọi tra từ điển theo locale hiện hành.
3. Chấm dứt việc gán cứng tiếng Việt trong các callback sự kiện của nút `RefreshAllQuota` và `Refresh`.
4. Đảm bảo chạy `powershell -ExecutionPolicy Bypass -File scripts/build_windows.ps1` vượt qua 100% thành công không còn lỗi flaky.

---

## 6. KẾT LUẬN & TRẠNG THÁI BÀN GIAO
- **Verdict:** **REJECT**
- **Next Step:** Bàn giao báo cáo chi tiết về CEO Thread ID để điều phối Architect và Coder thực hiện chu trình sửa chữa theo chuẩn JSON song song của Owner.
