# Supervisor Report: Chuẩn Hóa i18n Popup 'Thêm Tài Khoản' & Phân Định Độc Lập Trạng Thái Ứng Viên

Verdict: PASS

## Metadata
- Report file: reports/supervisor/rp_supervisor_261026_add_account_modal_i18n_candidate_badges_separation.md
- Review time: 2026-10-26 11:05:00 +0700
- Reviewer: antigravity-supervisor
- Repo/project: antigravity-switch (g:\antigravity_auto_switch)
- Scope reviewed: Commits 9e54faa -> c1becab -> 3d2b03b; production source files in `internal/i18n/` and `internal/ui/`; test suites in `internal/i18n/`, `internal/ui/`, `reports/QA/modal-i18n-candidate/`; Windows binary `bin/antigravity-switch.exe`
- Claim reviewed: Toàn bộ giải pháp phân định độc lập trạng thái ứng viên popup "Thêm tài khoản" với thẻ màn hình chính, chuẩn hóa đa ngôn ngữ thuần Việt và chuẩn Anh thương mại, khai tử triệt để mã hack call stack runtime.Caller/isLegacyModalTest, tuân thủ 100% Touch-Map và 5 bất biến kiến trúc, đạt chuẩn nghiệm thu Zero-Trust.
- Authority used: 
  - `reports/system-architect/rp_system-architect_261026_101500_by_antigravity_add_account_modal_i18n_candidate_badges_separation.md` (SSOT Blueprint)
  - `reports/coder/rp_coder_261026_add_account_modal_i18n_candidate_badges_separation.md`
  - `reports/qa/rp_qa_261026_add_account_modal_i18n_candidate_badges_separation.md`
  - `g:\antigravity_auto_switch\.agents\skills\supervisor\SKILL.md`
- Related artifacts:
  - Binary: `bin/antigravity-switch.exe` (27.92 MB, PE version 1.0.3.0, 32x32 icon)
  - Visual Evidence: `reports/QA/modal-i18n-candidate/screenshots/01_modal_candidates_vi.png`
  - Visual Evidence: `reports/QA/modal-i18n-candidate/screenshots/02_modal_candidates_en.png`
  - Visual Evidence: `reports/QA/modal-i18n-candidate/screenshots/03_modal_mutation_inplace_vi.png`
  - Visual Evidence: `reports/QA/modal-i18n-candidate/screenshots/04_main_ui_vs_modal_distinction.png`

---

## Executive Summary
- Problem: Người dùng phát hiện popup "Thêm tài khoản" hiển thị nhãn sai lệch `[✔ Active]` / `[✔ Đang dùng]` cho các ứng viên đã có token trong kho. Kiểm tra kiến trúc cho thấy lập trình viên trước đã mượn nhãn Main UI `d.BadgeActive` và dùng thủ thuật `runtime.Caller` / `isLegacyModalTest` để lừa test runner trong khi chạy ứng dụng thật thì lộ lỗi; kèm theo việc thiếu các trường từ điển riêng cho modal trong i18n và chuỗi tiếng Việt in hoa không dấu (`"THEM"`).
- Decision: PASS. Thẩm định Zero-Trust độc lập xác nhận:
  1. Touch-Map tuân thủ 100% (chính xác 7 tệp được phê duyệt; các tệp cấm can thiệp giữ nguyên vẹn).
  2. Triệt tiêu hoàn toàn DEAD CODE và mã hack Call Stack (`runtime.Caller`, `isLegacyModalTest` và import `runtime` sạch 100% khỏi `add_modal.go`).
  3. Cả 5 bất biến kiến trúc cốt tử (Candidate Autonomy, Zero-Reflection, Pure Dual-JSON Parity, In-Place Reactive Modal Mutation, Session Reloader 8s Invariant) đều được khép kín và kiểm chứng thực nghiệm.
  4. Toàn bộ test suites của kho mã nguồn (i18n, ui, QA render test) PASS 100%. Build script `scripts/build_windows.ps1` tạo thành công binary sạch `bin/antigravity-switch.exe`.
- Required outcome: Accepted. Sẵn sàng bàn giao cho Human Owner & CEO Agent để hợp nhất vào nhánh chính.

---

## Source-Level Clearance Notes

### 1. `internal/i18n/dictionary.go` (CLEAR)
- Lines 73-77: Bổ sung chính xác 5 trường modal candidate badge và button (`ModalBadgeOAuthDone`, `ModalBadgeAdded`, `ModalBadgeNeedOAuth`, `ModalBadgeNotFound`, `ModalBtnAddItem`) với tag json đồng bộ.
- Không có trường thừa, không có dead code.

### 2. `internal/i18n/locales/vi.json` (CLEAR)
- Lines 50-54: Thêm 5 cặp key-value tiếng Việt chuẩn có dấu:
  - `"modal_badge_oauth_done": "[✔ Đã OAuth]"`
  - `"modal_badge_added": "[Đã thêm]"`
  - `"modal_badge_need_oauth": "[Cần OAuth]"`
  - `"modal_badge_not_found": "[Tài khoản không tồn tại]"`
  - `"modal_btn_add_item": "Thêm"` (Đã chuẩn hóa từ chuỗi `"THEM"`).
- Đối xứng 100% với `en.json`.

### 3. `internal/i18n/locales/en.json` (CLEAR)
- Lines 50-54: Thêm 5 cặp key-value tiếng Anh chuẩn thương mại:
  - `"modal_badge_oauth_done": "[✔ OAuth Done]"`
  - `"modal_badge_added": "[Added]"`
  - `"modal_badge_need_oauth": "[Need OAuth]"`
  - `"modal_badge_not_found": "[Account Not Found]"`
  - `"modal_btn_add_item": "Add"`
- Khớp 1:1 với struct `Dictionary`.

### 4. `internal/ui/add_modal.go` (CLEAR)
- Lines 3-18: Gỡ bỏ import `runtime`.
- Xóa sạch biến cờ `isLegacyModalTest` và vòng lặp duyệt call stack `runtime.Caller`.
- Lines 84-138: Triển khai tường minh 4 nhánh trạng thái ứng viên kết nối trực tiếp với từ điển `i18n.Current()`:
  - Case 1 (`candItem.HasToken`): Badge `d.ModalBadgeOAuthDone` (màu Success) + Nút Xóa. Tuyệt đối không dùng `d.BadgeActive`.
  - Case 2 (`candItem.Added`): Badge `d.ModalBadgeAdded` (màu TextMuted) + Nút Xóa.
  - Case 3 (`!candItem.Added && !candItem.HasToken`): Badge `d.ModalBadgeNeedOAuth` (màu Warning) + Nút `d.ModalBtnAddItem` + Nút Xóa. Bấm nút Thêm thực hiện đột biến tại chỗ: nhãn chuyển thành `d.ModalBadgeAdded`, nút Thêm biến mất, nút Xóa giữ nguyên, refresh overview ngầm không gây giật lag.
  - Case 4 (`!candItem.FileExists`): Badge `d.ModalBadgeNotFound` (màu TextMuted) + Nút Xóa.
- Lines 188-231: Giữ nguyên vẹn logic xác nhận xóa candidate.

### 5. `internal/ui/add_modal_test.go` & `internal/ui/add_modal_acceptance_test.go` (CLEAR)
- Nâng cấp `inspectModalActionSlot` và `inspectAcceptanceActionSlot` để nhận diện nút Thêm qua icon `theme.ContentAddIcon()` hoặc text `"Thêm"`, `"Add"`, `"ADD"`.
- Cập nhật các câu lệnh assert từ `[Đã add]` sang `[Đã thêm]`.
- Giữ vững toàn bộ assertion logic, không làm suy yếu kiểm thử.

### 6. `internal/ui/i18n_acceptance_test.go` (CLEAR)
- Lines 285-473: Bổ sung ca kiểm thử chấp nhận toàn diện `TestAcceptance_i18n_AddModal_BilingualCandidateBadges`.
- Thiết lập đầy đủ 4 trường hợp ứng viên (OAuth, Added, Need OAuth, Ghost account).
- Xác minh tính chính xác của cả 4 nhãn trên cả hai locale VI và EN.
- Khẳng định tường minh `badgeOAuthVI != "[✔ Active]"` và `badgeOAuthVI != "[✔ Đang dùng]"`.
- Kiểm tra cơ chế đột biến phản ứng tại chỗ (In-Place Reactive Mutation) trên canvas thật.

### 7. Vùng cấm can thiệp (NON-GOALS PROTECTION) (CLEAR)
- `internal/reloader/session_reloader.go`: 0 byte thay đổi.
- `internal/manager/account_manager.go`: 0 byte thay đổi (bất biến `waitDuration := 8 * time.Second` nguyên vẹn).
- `internal/automation/*`: 0 byte thay đổi.
- `internal/ui/widgets.go`: 0 byte thay đổi (nhãn thẻ Main UI nguyên vẹn).

---

## Evidence Checked

### Passed:
1. **Touch-Map Audit:**
   - Lệnh: `git diff --stat 9e54faa..c1becab`
   - Bằng chứng: Chỉ có đúng 7 tệp được chỉ định trong Touch-Map bị thay đổi.
   - Verification freshness: Fresh (đối soát trực tiếp trên git log và git diff).
2. **Dead Code & Zero-Hack Static Sweep:**
   - Lệnh: `git grep "runtime.Caller" internal/ui/add_modal.go` & `git grep "isLegacyModalTest" internal/ui/add_modal.go`
   - Bằng chứng: 0 kết quả phát hiện trong `add_modal.go`.
   - Lệnh: `go vet ./internal/i18n/... ./internal/ui/...`
   - Bằng chứng: Exit code 0, không có biến thừa, hàm rác hay import lỗi thời.
   - Verification freshness: Fresh.
3. **i18n Unit Test Suite:**
   - Lệnh: `go test -v -count=1 ./internal/i18n/...`
   - Bằng chứng: 4/4 tests PASS (0.801s) gồm `TestI18n_DictionaryCompleteness`, `TestI18n_JSONKeyParity`, `TestI18n_LocaleSwitching`, `TestI18n_ThreadSafety`.
   - Verification freshness: Fresh.
4. **UI Acceptance Test Suite:**
   - Lệnh: `go test -v -count=1 ./internal/ui/ -run TestAcceptance_i18n_AddModal_BilingualCandidateBadges`
   - Bằng chứng: PASS (0.47s) xác nhận độc lập cả 4 nhãn VI, 4 nhãn EN và mutation tại chỗ.
   - Lệnh: `go test -count=1 ./internal/ui/...`
   - Bằng chứng: PASS toàn bộ 40+ test suites trong `internal/ui` (22.797s).
   - Verification freshness: Fresh.
5. **QA Dedicated Invariant Suite:**
   - Lệnh: `go test -v -count=1 ./reports/QA/modal-i18n-candidate/...`
   - Bằng chứng: 6/6 tests PASS (1.460s), bao gồm Invariant 1 (Candidate Autonomy), Invariant 2 (Zero Reflection), Invariant 3 (Dual-JSON Parity), Invariant 4 (In-Place Mutation), Invariant 5 (Session Reloader Protection) và xuất 4 file PNG visual.
   - Verification freshness: Fresh.
6. **Windows Release Build:**
   - Lệnh: `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1`
   - Bằng chứng: Biên dịch thành công `bin/antigravity-switch.exe` (27.92 MB - 29,274,112 bytes), icon PE 32x32 verified, ProductVersion 1.0.3.0, 12/12 packages pre-flight PASS.
   - Verification freshness: Fresh.
7. **Visual Evidence Inspection:**
   - Tập tin: `reports/QA/modal-i18n-candidate/screenshots/01_modal_candidates_vi.png` (27,155 B)
   - Tập tin: `reports/QA/modal-i18n-candidate/screenshots/02_modal_candidates_en.png` (26,366 B)
   - Tập tin: `reports/QA/modal-i18n-candidate/screenshots/03_modal_mutation_inplace_vi.png` (32,756 B)
   - Tập tin: `reports/QA/modal-i18n-candidate/screenshots/04_main_ui_vs_modal_distinction.png` (54,247 B)
   - Bằng chứng: Hình ảnh kết xuất rõ ràng, không có artifacts đồ họa, phản ánh chính xác ranh giới độc lập giữa 2 vùng giao diện.
   - Verification freshness: Fresh.

### Failed:
- None.

### Not run:
- None. (Mọi khía cạnh kiểm tra tĩnh, kiểm tra động, build và visual đều đã được thực thi độc lập).

---

## Invariant Closure

- **Affected Invariant:**
  1. *Candidate Autonomy Invariant*: Độc lập tuyệt đối giữa danh bạ hồ sơ ứng viên Chrome (`ModalBadgeOAuthDone`) và phiên thực thi Antigravity IDE của Main UI (`BadgeActive`).
  2. *Zero-Reflection Invariant*: Cấm triệt để việc dùng `runtime.Caller` rẽ nhánh "lừa test".
  3. *Pure Dual-JSON Parity Invariant*: Tính đối xứng 1:1 tuyệt đối giữa `vi.json` và `en.json`.
  4. *In-Place Reactive Mutation Invariant*: Đột biến tại chỗ `[Cần OAuth]` -> `[Đã thêm]`, ẩn nút Thêm, giữ nút Xóa, nạp thẻ Main UI ngầm không giật popup.
  5. *Session Reloader 8-Second Protection*: Bất biến chờ 8s và logic switch tài khoản không bị xáo trộn.
- **Sibling surfaces checked:**
  - `internal/ui/widgets.go`: Đã xác minh nhãn Main UI giữ nguyên vẹn `d.BadgeActive`, `d.BadgeSwitch`, `d.BadgeNeedOAuth`.
  - `internal/manager/account_manager.go`: Đã kiểm tra `waitDuration := 8 * time.Second` ở dòng 787 còn nguyên vẹn.
  - `internal/reloader/session_reloader.go`: Đã xác minh `sleeper(ctx, waitDuration)` nguyên vẹn.
- **Residual unverified same-invariant surfaces:** None.

---

## Overall Evaluation

Bản triển khai kỹ thuật của Coder (commit `c1becab`) cùng hệ thống kiểm thử thực nghiệm và bằng chứng trực quan của QA (commit `3d2b03b`) đạt độ hoàn thiện cao nhất theo chuẩn Zero-Trust:
1. **Giải quyết đúng và triệt để nguyên nhân gốc rễ:** Thay vì "vá víu" bề mặt, Coder đã dũng cảm bóc tách hoàn toàn đoạn mã hack call stack `runtime.Caller` và định nghĩa riêng 5 khóa từ điển độc lập cho Candidate Modal.
2. **Chất lượng mã nguồn:** Sạch sẽ, tường minh, không dead code, tuân thủ chặt chẽ 100% Touch-Map.
3. **Chất lượng kiểm thử:** Các test suite kiểm chứng đa tầng từ Unit test (dictionary, parity) đến Acceptance test (bilingual, in-place mutation) và QA visual canvas rendering.
4. **Phán quyết:** Phê duyệt chính thức **PASS**. Kính chuyển hồ sơ nghiệm thu lên CEO Agent và Human Owner.