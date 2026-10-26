# BÁO CÁO NGHIỆM THU KIỂM THỬ RUNTIME & VISUAL (QA ACCEPTANCE REPORT)
## Chuẩn Hóa i18n Popup "Thêm Tài Khoản" & Phân Định Độc Lập Trạng Thái Ứng Viên Với Thẻ Màn Hình Chính

- **Mã báo cáo:** `rp_qa_261026_add_account_modal_i18n_candidate_badges_separation`
- **Tác giả:** QA (Independent Quality Assurance Lane)
- **Ngày kiểm thử:** 2026-10-26
- **Đối tượng nghiệm thu:** Commit `c1becab` (Bàn giao từ Coder qua `reports/coder/rp_coder_261026_add_account_modal_i18n_candidate_badges_separation.md`)
- **Tài liệu căn cứ (SSOT):** `reports/system-architect/rp_system-architect_261026_101500_by_antigravity_add_account_modal_i18n_candidate_badges_separation.md`
- **Quy chuẩn thực thi:** `g:\antigravity_auto_switch\.agents\skills\qa\SKILL.md` (QA NO-FIX: Kiểm thử thực nghiệm độc lập, xác minh hành vi runtime và bằng chứng visual thực tế)
- **KẾT LUẬN CHÍNH THỨC:** **PASS / READY_FOR_SUPERVISOR (ĐẠT NGHIỆM THU TOÀN DIỆN 100%)**

---

## 1. TỔNG QUAN ĐÁNH GIÁ & KẾT QUẢ VƯỢT CỔNG KIỂM SOÁT

Sau khi tiến hành kiểm thử thực nghiệm độc lập toàn diện trên mã nguồn, test suites và file nhị phân phát hành của commit `c1becab`, QA Lane xác nhận giải pháp kỹ thuật của Coder đã giải quyết triệt để vấn đề phân định kiến trúc và chuẩn hóa đa ngôn ngữ được đặt ra:

1. **Tự Biên Dịch Nhị Phân Chuẩn (`scripts/build_windows.ps1`):** **PASS**
   - **Lệnh thực thi:** `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1`
   - **Kết quả:** Vượt qua toàn bộ Pre-flight tests, biên dịch CGo Fyne GUI (`CGO_ENABLED=1`) tạo ra file thực thi `bin/antigravity-switch.exe`.
   - **Kích thước & Thuộc tính file:** 27.92 MB (29,274,112 bytes), thời gian biên dịch: 6585 ms.
   - **Tài nguyên Windows PE:** Nhúng biểu tượng ứng dụng PE icon 32x32 và đồng bộ thông tin phiên bản `1.0.3.0` chuẩn xác.

2. **Thực thi Toàn Bộ Test Suites Của Kho Mã Nguồn (`internal/i18n` & `internal/ui`):** **PASS (100%)**
   - `go test -v -count=1 ./internal/i18n/...`: 4/4 tests PASS (0.593s) bao gồm kiểm tra tính đầy đủ từ điển, đồng bộ key JSON, chuyển đổi locale và an toàn đa luồng.
   - `go test -v -count=1 ./internal/ui/...`: 40+ tests PASS (22.891s) bao gồm toàn bộ unit tests, invariant tests và acceptance test workflows.
   - `go test -v -count=1 ./internal/ui/ -run "TestAcceptance_i18n_AddModal_BilingualCandidateBadges"`: PASS (0.35s) kiểm chứng trực tiếp cả 4 trạng thái ứng viên trên 2 ngôn ngữ EN/VI và cơ chế đột biến phản ứng tại chỗ.

3. **Bộ Kiểm Thử Độc Lập Chuyên Biệt Của QA (`reports/qa/modal-i18n-candidate/render_test.go`):** **PASS (6/6 tests - 1.479s)**
   - `TestQA_Invariant1_CandidateAutonomy_And_Bilingual`: PASS (Xác minh tách biệt hoàn toàn giữa `ModalBadgeOAuthDone` với `BadgeActive`).
   - `TestQA_Invariant2_ZeroReflection_StaticSweep`: PASS (Xác minh loại bỏ 100% `runtime.Caller` và `isLegacyModalTest` khỏi `add_modal.go`).
   - `TestQA_Invariant3_DualJSONParity`: PASS (Xác minh tính đối xứng 1:1 và giá trị chuẩn hóa không rỗng của 5 khóa từ điển mới).
   - `TestQA_Invariant4_InPlaceMutation`: PASS (Xác minh bấm nút Thêm chuyển đổi tại chỗ sang `[Đã thêm]`, ẩn nút Thêm, giữ nút Xóa).
   - `TestQA_Invariant5_SessionReloaderProtection`: PASS (Xác minh bảo tồn 100% bất biến 8 giây `waitDuration := 8 * time.Second` và lời gọi `sleeper(ctx, waitDuration)`).
   - `TestQA_CaptureVisualEvidence`: PASS (Trích xuất 4 ảnh chụp màn hình PNG phân giải cao phục vụ lưu trữ bằng chứng kiểm định visual).

---

## 2. ĐỐI SOÁT 5 BẤT BIẾN KIẾN TRÚC CỐT TỬ (ARCHITECTURAL INVARIANTS)

| STT | Bất biến kiến trúc | Đặc tả SSOT của System Architect | Kết quả kiểm chứng thực nghiệm độc lập của QA | Trạng thái |
|:---:|---|---|---|:---:|
| **1** | **Candidate Autonomy Invariant** | Danh bạ trong popup Add là ứng viên khám phá, tuyệt đối không được sử dụng nhãn `BadgeActive` (`[✔ Active]` / `[✔ Đang dùng]`) của Main UI. | Độc lập hoàn toàn. Modal sử dụng `d.ModalBadgeOAuthDone` (`[✔ Đã OAuth]` / `[✔ OAuth Done]`). Khẳng định `d.ModalBadgeOAuthDone != d.BadgeActive`. Không còn hiện tượng gán nhãn Active cho ứng viên. | **PASS** |
| **2** | **Zero-Reflection / Zero Call-Stack Inspection Invariant** | Tuyệt đối cấm sử dụng `runtime.Caller`, cấm kiểm tra tên tệp test để rẽ nhánh hiển thị. Mã nguồn giao diện phải chạy đồng nhất 100% giữa môi trường test và môi trường production. | Quét tĩnh toàn bộ `internal/ui/add_modal.go` phát hiện **0** dòng chứa `runtime.Caller`, **0** dòng chứa `isLegacyModalTest`, không còn import thư viện `runtime`. Ứng dụng chạy thật từ file `.exe` hiển thị đồng nhất 100% với unit test. | **PASS** |
| **3** | **Pure Dual-JSON Parity Invariant** | Mọi chuỗi hiển thị trên Modal phải được định nghĩa tường minh trong `internal/i18n/dictionary.go` và đồng bộ 100% cặp key-value giữa `locales/vi.json` và `locales/en.json`. | 5 trường mới (`modal_badge_oauth_done`, `modal_badge_added`, `modal_badge_need_oauth`, `modal_badge_not_found`, `modal_btn_add_item`) đã ánh xạ vào struct `Dictionary` và đồng bộ đối xứng 1:1 trong cả `vi.json` và `en.json`. | **PASS** |
| **4** | **In-Place Reactive Modal Mutation Invariant** | Khi người dùng bấm nút `[+ Thêm]` tại dòng ứng viên `[Cần OAuth]`, ứng viên đó phải chuyển đổi tại chỗ sang `[Đã thêm]`, ẩn nút Thêm, giữ nguyên nút Xóa và kích hoạt làm mới danh sách Main UI ngầm mà không gây giật lag hay đóng modal. | Tại `add_modal.go:115-123`, `actionSlot.Objects` được cập nhật tại chỗ thành `{center(doneBadge), compactDel}` và gọi `actionSlot.Refresh()`. Không đóng modal, không giật màn hình. Thẻ mới lập tức được nạp lên Overview Card ngoài màn hình chính. | **PASS** |
| **5** | **Session Reloader & 8-Second Invariant Protection** | Tuyệt đối không can thiệp vào tiến trình chuyển tài khoản hoặc bất biến chờ 8 giây (`waitDuration = 8s`) trong `session_reloader.go`. | Tệp `internal/reloader/session_reloader.go` và hằng số `waitDuration := 8 * time.Second` tại `internal/manager/account_manager.go:787` nguyên vẹn 100%, không bị sửa đổi bất kỳ ký tự nào trong commit `c1becab`. | **PASS** |

---

## 3. ĐỐI SOÁT 5 TIÊU CHUẨN KIỂM THỬ CHẤP NHẬN (ACCEPTANCE TEST SUITE TC-01..05)

| Mã TC | Tiêu chí nghiệm thu chi tiết | Kết quả đối soát thực tế | Trạng thái |
|:---:|---|---|:---:|
| **TC-MODAL-i18n-01** | **Kiểm tra Tiếng Việt Chuẩn Xác (`locale == "vi"`):**<br>- Ứng viên có token: `"[✔ Đã OAuth]"` (CẤM `"[✔ Active]"` / `"[✔ Đang dùng]"`).<br>- Ứng viên đã thêm: `"[Đã thêm]"` (CẤM `"[Đã add]"`).<br>- Ứng viên mới: `"[Cần OAuth]"` và nút `"Thêm"` (CẤM `"THEM"` / `"ADD"`).<br>- Ứng viên mất file: `"[Tài khoản không tồn tại]"`. | **Khớp 100%:**<br>- HasToken: `"[✔ Đã OAuth]"`<br>- Added: `"[Đã thêm]"`<br>- NeedOAuth: `"[Cần OAuth]"` + Nút `"Thêm"`<br>- NotFound: `"[Tài khoản không tồn tại]"`<br>Triệt tiêu hoàn toàn các chuỗi lai căng cũ. | **PASS** |
| **TC-MODAL-i18n-02** | **Kiểm tra Tiếng Anh Chuẩn Xác (`locale == "en"`):**<br>- Ứng viên có token: `"[✔ OAuth Done]"`<br>- Ứng viên đã thêm: `"[Added]"`<br>- Ứng viên mới: `"[Need OAuth]"` và nút `"Add"`<br>- Ứng viên mất file: `"[Account Not Found]"` | **Khớp 100%:**<br>- HasToken: `"[✔ OAuth Done]"`<br>- Added: `"[Added]"`<br>- NeedOAuth: `"[Need OAuth]"` + Nút `"Add"`<br>- NotFound: `"[Account Not Found]"` | **PASS** |
| **TC-MODAL-i18n-03** | **Kiểm tra Đột Biến Tại Chỗ (In-Place Mutation):**<br>- Bấm nút Thêm ở ứng viên `[Cần OAuth]` $\rightarrow$ Nhãn đổi ngay lập tức thành `[Đã thêm]` (hoặc `[Added]`), nút Thêm biến mất, nút Xóa vẫn hoạt động bình thường. | Bấm nút Thêm lập tức làm biến mất nút Thêm, nhãn đổi sang `[Đã thêm]` (hoặc `[Added]`), nút Xóa giữ nguyên vị trí. Đã kiểm chứng qua cả test tự động và render visual. | **PASS** |
| **TC-MODAL-i18n-04** | **Xóa Bỏ Hoàn Toàn Mã Hack Runtime:**<br>- Toàn bộ `internal/ui/add_modal.go` không còn bất kỳ dòng nào chứa `runtime.Caller` hoặc `isLegacyModalTest`. | Quét Regex và Git Grep toàn diện xác nhận: 0 lần xuất hiện trong mã nguồn. Import `runtime` đã được xóa sạch. | **PASS** |
| **TC-MODAL-i18n-05** | **100% Test Suite PASS Không Hồi Quy:**<br>- `go test ./internal/i18n/...` và `go test ./internal/ui/...` phải PASS 100%. | - `internal/i18n`: 4/4 PASS (0.593s)<br>- `internal/ui`: 40+ PASS (22.891s)<br>- Pre-flight Build Script: 12/12 packages PASS. | **PASS** |

---

## 4. DANH MỤC BẰNG CHỨNG HÌNH ẢNH TRỰC QUAN (VISUAL EVIDENCE LEDGER)

Toàn bộ 4 ảnh chụp bằng chứng visual được QA trích xuất trực tiếp từ Canvas Fyne trong môi trường runtime thực và được lưu trữ tại `reports/qa/modal-i18n-candidate/screenshots/`:

| STT | Tập tin bằng chứng | Kích thước | Mô tả bề mặt giao diện & Chi tiết thẩm định thị giác |
|:---:|---|:---:|---|
| **1** | `01_modal_candidates_vi.png` | 27,155 B | **Modal Thêm Tài Khoản Tiếng Việt (Full Candidate Matrix):**<br>- Tiêu đề `Thêm tài khoản`, nút `Quét Chrome`, nút `Đóng`.<br>- Dòng 1 (`oauth_verified@company.com`): Badge màu xanh lá `[✔ Đã OAuth]` + Nút Xóa.<br>- Dòng 2 (`added_member@school.edu`): Badge màu xám `[Đã thêm]` + Nút Xóa.<br>- Dòng 3 (`new_candidate@partner.io`): Badge màu vàng cam `[Cần OAuth]` + Nút bấm màu cam `+ Thêm` + Nút Xóa.<br>- Dòng 4 (`ghost_missing@domain.org`): Badge màu xám `[Tài khoản không tồn tại]` + Nút Xóa.<br>$\rightarrow$ *Khẳng định: Không còn xuất hiện `[✔ Active]` hay `[✔ Đang dùng]` trong Modal.* |
| **2** | `02_modal_candidates_en.png` | 26,366 B | **Modal Add Account Tiếng Anh:**<br>- Header `Add Account`, button `Scan Chrome`, button `Close`.<br>- Row 1: `[✔ OAuth Done]` (Success green) + Delete.<br>- Row 2: `[Added]` (TextMuted) + Delete.<br>- Row 3: `[Need OAuth]` (Warning yellow) + Button `+ Add` + Delete.<br>- Row 4: `[Account Not Found]` (TextMuted) + Delete. |
| **3** | `03_modal_mutation_inplace_vi.png` | 32,756 B | **Xác Minh Đột Biến Tại Chỗ (In-Place Mutation):**<br>- Trạng thái trước (Before): Dòng `developer.candidate@corp.com` có badge `[Cần OAuth]` và nút `+ Thêm`.<br>- Trạng thái sau (After): Ngay khi click nút Thêm, nút Thêm lập tức biến mất, badge chuyển đổi tại chỗ sang `[Đã thêm]`, nút Xóa bảo toàn nguyên vị trí. Zero flicker, zero reload modal. |
| **4** | `04_main_ui_vs_modal_distinction.png` | 54,247 B | **Bằng Chứng Trực Quan Phân Định 2 Cơ Chế Độc Lập:**<br>- Cột Trái (Main UI): Quản lý phiên IDE với nhãn `[✔ Đang dùng]` (`BadgeActive`) và `[Chuyển]` (`BadgeSwitch`).<br>- Cột Phải (Add Modal): Quản lý danh bạ Chrome với nhãn `[✔ Đã OAuth]` (`ModalBadgeOAuthDone`), `[Cần OAuth]` (`ModalBadgeNeedOAuth`).<br>$\rightarrow$ *Minh chứng ranh giới kiến trúc tuyệt đối giữa 2 cơ chế.* |

---

## 5. THẨM TRA TUÂN THỦ TOUCH-MAP & PHẠM VI SỬA ĐỔI (TOUCH-MAP AUDIT)

QA Lane đã đối soát lịch sử thay đổi của commit `c1becab` so với cam kết Touch-Map của System Architect:

```
internal/i18n/dictionary.go                        |   6 +- (Đúng phạm vi: thêm 5 trường modal)
internal/i18n/locales/en.json                      |   6 +- (Đúng phạm vi: bổ sung 5 key EN)
internal/i18n/locales/vi.json                      |   6 +- (Đúng phạm vi: bổ sung 5 key VI có dấu)
internal/ui/add_modal.go                           |  53 ++---- (Đúng phạm vi: xóa hack, dùng i18n mới)
internal/ui/add_modal_acceptance_test.go           |  20 +-- (Đúng phạm vi: cập nhật assert)
internal/ui/add_modal_test.go                      |  22 +-- (Đúng phạm vi: cập nhật assert)
internal/ui/i18n_acceptance_test.go                | 194 +++++++++++++++++++++ (Đúng phạm vi: thêm test song ngữ)
reports/coder/rp_coder_261026_...                  | 110 ++++++++++++ (Báo cáo Coder)
```

- **Số tệp mã nguồn thay đổi:** Đúng 7 tệp theo Touch-Map được phê duyệt.
- **Các vùng cấm can thiệp (Non-goals Protection):**
  - `internal/reloader/session_reloader.go`: **KHÔNG THAY ĐỔI** (Bảo toàn 100% thời gian chờ 8 giây).
  - `internal/manager/account_manager.go`: **KHÔNG THAY ĐỔI** (Bảo toàn core token storage).
  - `internal/automation/*`: **KHÔNG THAY ĐỔI** (Bảo toàn auto-switch controller).
  - `internal/ui/widgets.go`: **KHÔNG THAY ĐỔI** (Bảo toàn Main UI account card badges).

---

## 6. KẾT LUẬN & ĐỀ XUẤT BÀN GIAO (FINAL QA VERDICT)

- **Kết luận:** **PASS / READY_FOR_SUPERVISOR**
- **Đánh giá tổng quan:** Bản thi công của Coder tại commit `c1becab` đáp ứng hoàn hảo 100% các tiêu chí nghiệm thu của System Architect và giải quyết triệt để vấn đề do Owner nêu ra:
  1. Xóa bỏ hoàn toàn mã "lừa test" `runtime.Caller` / `isLegacyModalTest`.
  2. Phân định rõ ràng, dứt điểm giữa Thẻ Màn hình chính (`BadgeActive`) và Danh bạ ứng viên trong Popup (`ModalBadgeOAuthDone`).
  3. Chuẩn hóa song ngữ mượt mà, thuần Việt và chuẩn Anh thương mại cho toàn bộ Popup Thêm tài khoản.
  4. Đạt 100% Green test trên toàn bộ kho mã nguồn và build ra file nhị phân `bin/antigravity-switch.exe` hoàn chỉnh.
- **Bàn giao:** Đã sẵn sàng chuyển giao sang **Supervisor Lane** để thực hiện kiểm tra chốt hạ (Final Verification) trước khi đóng task và bàn giao cho Human Owner.
