# BÁO CÁO THI CÔNG: PHASE 3 - XÂY DỰNG GIAO DIỆN HỘP THOẠI THÊM TÀI KHOẢN (ADD ACCOUNT MODAL UI WIDGET)

- **Người thực hiện:** Coder
- **Ngày thực hiện:** 2026-10-22
- **Trạng thái:** COMPLETED / READY FOR SUPERVISOR REVIEW
- **Tài liệu tham chiếu tối cao (SSOT):** `reports/architect/2026-10-22-add-account-modal-and-discovered-filtering-spec.md` (Mã SPEC-ARCH-20261022-ADD-ACCOUNT-MODAL-AND-DISCOVERED-FILTERING-SPEC, commit a77efb9, Phần IV.3, V Bước 3)
- **Cam kết kỷ luật:**
  - Không sửa bất kỳ file nào ngoài scope Phase 3 (0 lines diff trên `scripts/build_windows.ps1`, `README.md`, `internal/ui/window.go`).
  - Không dùng lệnh Git diện rộng. Chỉ `git add` từng file trong scope.
  - Coder không tự cấp PASS; bàn giao `[READY_FOR_SUPERVISOR]`.

---

## 1. BẢN ĐỒ BẤT BIẾN KIẾN TRÚC (INVARIANT FAMILY MAP)

| Thuộc tính | Chi tiết xác thực |
|---|---|
| **Family Name** | Add Account Modal and Candidate Pool UI Widget Invariant |
| **SSOT / Authority Source** | `reports/architect/2026-10-22-add-account-modal-and-discovered-filtering-spec.md` § Phần IV.3, V Bước 3 |
| **Sibling Runtime Surfaces** | `internal/ui/add_modal.go` (`ShowAddAccountModal`, `buildAddAccountModalContent`), `internal/ui/add_modal_test.go`, `internal/manager/account_manager.go` (`GetCandidateAccounts`, `ScanChromeAccounts`, `AddDiscoveredAccount` - Phase 2), `internal/store/discovered_store.go` (`ListDiscoveredRecords`, `SetDiscoveredAdded` - Phase 1) |
| **Forbidden Legacy Fallback** | Quét Chrome tự động ngầm khi mở modal; tự ý đưa tài khoản ứng viên lên UI tổng trước khi người dùng bấm ADD; sửa trước toolbar `internal/ui/window.go` (thuộc Phase 4) |
| **Stale Tests / Helpers Updated** | Không có file test cũ nào bị stale trong slice Phase 3. Thêm mới 7 kịch bản kiểm thử độc lập trong `internal/ui/add_modal_test.go`. |
| **Verify Matrix** | 1. `ShowAddAccountModal` an toàn tuyệt đối khi `a == nil`, `a.win == nil`, `a.mgr == nil`.<br/>2. Hiển thị nhãn in nghiêng `"Không có tài khoản nào trong danh bạ."` khi danh sách ứng viên rỗng.<br/>3. Phân cấp 3 trạng thái ứng viên: `[✔ Đã OAuth]` (đã có token), `[Cần OAuth]` + nút `ADD` (chưa có token & chưa add), `[Đã add]` (chưa có token & đã add).<br/>4. Nút `ADD` phản ứng tại chỗ (in-place mutation): đổi thành badge `[Đã add]`, ẩn nút `ADD`, ghi nhận đĩa và nạp thẻ mới lên UI tổng.<br/>5. Nút `Quét Chrome`: vô hiệu hóa, đổi text `"⏳ Đang quét..."`, quét on-demand, khôi phục `"Quét Chrome"` và nạp lại danh sách.<br/>6. Email privacy masking: tuân thủ `a.IsMaskAccounts()` qua hàm `MaskEmail`.<br/>7. Modal dialog: kích thước `460x340`, `scrollContent` tối thiểu `420x260`, hiển thị overlay chuẩn trên cửa sổ Fyne.<br/>8. Build script Windows `pwsh -File .\scripts\build_windows.ps1 -Clean` Exit Code 0 và `go test ./...` pass 100%. |

---

## 2. NỘI DUNG MÃ NGUỒN ĐÃ THI CÔNG

### 2.1. `internal/ui/add_modal.go` (Tạo mới)
Triển khai widget Modal thêm tài khoản theo đúng thiết kế kiến trúc Phần IV.3:
1. **Phương thức công khai `ShowAddAccountModal()`:**
   - Kiểm tra nil-safety: nếu `a == nil || a.win == nil || a.mgr == nil` thì return an toàn, không panic.
   - Khởi tạo dialog modal kích thước `460x340` (với `scrollContent` tối thiểu `420x260`) qua `dialog.NewCustom("Thêm tài khoản", "Đóng", contentBox, a.win)`.
2. **Cụm Header:**
   - Tiêu đề in đậm `"Thêm tài khoản"` (`widget.NewLabelWithStyle(..., TextStyle{Bold: true})`).
   - Nút `"Quét Chrome"` (`widget.NewButtonWithIcon` kèm icon `theme.ViewRefreshIcon()`, bọc qua `compactButton`).
   - Xử lý tương tác nút Quét Chrome: khi bấm, ngay lập tức `scanBtn.Disable()` và đổi text thành `"⏳ Đang quét..."`, chạy goroutine gọi `a.mgr.ScanChromeAccounts()`, sau đó gọi `fyne.Do()` để nạp lại danh sách (`reloadCandidates()`), đặt lại text `"Quét Chrome"` và `scanBtn.Enable()`.
3. **Danh sách cuộn ứng viên (`reloadCandidates`):**
   - Lấy danh sách từ `a.mgr.GetCandidateAccounts()`.
   - Nếu rỗng: hiển thị nhãn in nghiêng `"Không có tài khoản nào trong danh bạ."`.
   - Mỗi hàng hiển thị Email (hỗ trợ ellipsis text; tự động bọc qua `MaskEmail` nếu `a.IsMaskAccounts()` bật).
   - **Trạng thái 1 (`candItem.HasToken`):** Badge `[✔ Đã OAuth]` màu `palette.Success`, bold, size 11, bọc `container.NewCenter`.
   - **Trạng thái 2 (`!candItem.HasToken && !candItem.Added`):** Badge `[Cần OAuth]` màu `palette.Warning`, bold, size 11 + Nút `ADD` (`HighImportance`, icon `theme.ContentAddIcon()`, bọc `compactButton`).
     - Khi bấm `ADD`: gọi `a.mgr.AddDiscoveredAccount(candItem.Email)`.
     - Phản ứng tại chỗ (in-place): thay thế `actionSlot` bằng badge `[Đã add]` màu `palette.TextMuted`, nút `ADD` biến mất.
     - Nạp ngay thẻ mới lên UI tổng: chạy goroutine lấy `a.mgr.GetAccountStates(context.Background())` rồi gọi `fyne.Do(func() { a.updateUI(overview) })`.
   - **Trạng thái 3 (`!candItem.HasToken && candItem.Added`):** Badge `[Đã add]` màu `palette.TextMuted`, bold, size 11, bọc `container.NewCenter`, không có nút ADD.

### 2.2. `internal/ui/add_modal_test.go` (Tạo mới)
Bổ sung 7 bài kiểm thử đơn vị toàn diện:
1. `TestShowAddAccountModal_NilSafety`: Chứng minh an toàn tuyệt đối khi `a == nil`, `a.win == nil`, hoặc `a.mgr == nil`.
2. `TestShowAddAccountModal_EmptyCandidates`: Chứng minh trạng thái rỗng hiển thị đúng nhãn in nghiêng `"Không có tài khoản nào trong danh bạ."`.
3. `TestShowAddAccountModal_ThreeCandidateStates`: Chứng minh phân giải chuẩn xác 3 trạng thái ứng viên:
   - `user1@company.com` (có token): hiển thị `[✔ Đã OAuth]`.
   - `user3@school.edu` (chưa có token, đã add): hiển thị `[Đã add]`.
   - `user2@gmail.com` (chưa có token, chưa add): hiển thị `[Cần OAuth]` + nút `ADD` (`HighImportance`).
4. `TestShowAddAccountModal_AddActionInPlaceMutationAndOverviewUpdate`: Chứng minh khi bấm nút `ADD`:
   - Phản ứng tại chỗ: `actionSlot` chuyển đổi thành badge `[Đã add]`, nút `ADD` biến mất.
   - Kho đệm bền vững: `discStore.ListDiscoveredRecords()` ghi nhận `Added: true`.
   - UI tổng: thẻ mới xuất hiện trên `app.overview.Accounts` với trạng thái `[Cần OAuth]` và `StateUnauth`.
5. `TestShowAddAccountModal_EmailMasking`: Chứng minh khi bật `a.SetMaskAccounts(true)`, email ứng viên được che giấu (`u***@example.com`); khi tắt, email hiển thị nguyên bản.
6. `TestShowAddAccountModal_ScanChromeButtonBehavior`: Chứng minh nút Quét Chrome chuyển sang `"⏳ Đang quét..."`, bị vô hiệu hóa khi quét và tự phục hồi thành `"Quét Chrome"` (kèm re-enable) sau khi hoàn tất.
7. `TestShowAddAccountModal_DialogDisplayAndOverlay`: Chứng minh dialog modal hiển thị overlay trên canvas cửa sổ Fyne khi gọi `ShowAddAccountModal()`.

---

## 3. BẰNG CHỨNG KIỂM CHỨNG CHẤT LƯỢNG (E2E VERIFICATION)

### 3.1. Kiểm thử Đơn vị Unit Tests Phase 3
- **Lệnh thực thi:** `go test -v ./internal/ui -run TestShowAddAccountModal`
- **Kết quả:** `PASS` 100% (7/7 tests pass, 2.255s)
  ```text
  === RUN   TestShowAddAccountModal_NilSafety
  --- PASS: TestShowAddAccountModal_NilSafety (0.00s)
  === RUN   TestShowAddAccountModal_EmptyCandidates
  --- PASS: TestShowAddAccountModal_EmptyCandidates (0.14s)
  === RUN   TestShowAddAccountModal_ThreeCandidateStates
  --- PASS: TestShowAddAccountModal_ThreeCandidateStates (0.10s)
  === RUN   TestShowAddAccountModal_AddActionInPlaceMutationAndOverviewUpdate
  --- PASS: TestShowAddAccountModal_AddActionInPlaceMutationAndOverviewUpdate (0.08s)
  === RUN   TestShowAddAccountModal_EmailMasking
  --- PASS: TestShowAddAccountModal_EmailMasking (0.06s)
  === RUN   TestShowAddAccountModal_ScanChromeButtonBehavior
  --- PASS: TestShowAddAccountModal_ScanChromeButtonBehavior (0.15s)
  === RUN   TestShowAddAccountModal_DialogDisplayAndOverlay
  --- PASS: TestShowAddAccountModal_DialogDisplayAndOverlay (0.13s)
  PASS
  ok  	antigravity-switch/internal/ui	2.255s
  ```

### 3.2. Toàn bộ Suite Test Package `internal/ui`
- **Lệnh thực thi:** `go test ./internal/ui/...`
- **Kết quả:** `PASS` (33.640s)

### 3.3. Toàn bộ Test Toàn Repository
- **Lệnh thực thi:** `go test -p 1 ./...`
- **Kết quả:** `PASS` 100% trên toàn bộ các package (`cmd/switcher`, `internal/automation`, `internal/connector`, `internal/manager`, `internal/model`, `internal/reloader`, `internal/scanner`, `internal/store`, `internal/telemetry`, `internal/ui`, `internal/wincred`).

### 3.4. Biên dịch Toàn diện Hệ thống bằng Script của Repo
- **Lệnh thực thi:** `pwsh -File .\scripts\build_windows.ps1 -Clean`
- **Kết quả:**
  ```text
  ================================================================================
  ⚡ Antigravity Multi-Account Switcher - Windows Build System
     OutDir: bin | SkipTests: False | Clean: True
  ================================================================================
  [1/4] Cleaning output artifacts...
    Removed directory: bin
    Clean complete.
  [2/4] Running pre-flight tests & vet...
  ok  	antigravity-switch/cmd/switcher	2.159s
  ok  	antigravity-switch/internal/automation	(cached)
  ok  	antigravity-switch/internal/connector	(cached)
  ok  	antigravity-switch/internal/manager	(cached)
  ok  	antigravity-switch/internal/model	(cached)
  ok  	antigravity-switch/internal/reloader	(cached)
  ok  	antigravity-switch/internal/scanner	(cached)
  ok  	antigravity-switch/internal/store	8.636s
  ok  	antigravity-switch/internal/telemetry	(cached)
  ok  	antigravity-switch/internal/ui	(cached)
  ok  	antigravity-switch/internal/wincred	(cached)
    [PASS] Verification succeeded.
  [3/4] Compiling target binaries...
    -> Checking Windows application icon and resource...
    -> Compiling resource with windres: cmd\switcher\app.rc -> cmd\switcher\rsrc_windows_amd64.syso
    [SUCCESS] Compiled PE resource object: cmd\switcher\rsrc_windows_amd64.syso (12552 bytes)
    -> Building Standalone GUI Binary (Fyne, CGO_ENABLED=1)...
    [SUCCESS] Created: bin\antigravity-switch.exe
    Compile Duration: 7147 ms | Size: 27.77 MB (29113856 bytes)
  [4/4] Executing quick verification...
    [PASS] Embedded PE application icon verified on bin\antigravity-switch.exe (32x32)
    [INFO] GUI binary built at bin\antigravity-switch.exe.
  ================================================================================
  ⚡ BUILD SUCCESSFUL
  ================================================================================
  ```
- **Exit Code:** `0`

---

## 4. TỔNG KẾT THAY ĐỔI TỆP TIN (FILE INVENTORY)

| Tệp tin | Trạng thái | Mục đích |
|---|---|---|
| `internal/ui/add_modal.go` | Tạo mới | Triển khai `ShowAddAccountModal()` hiển thị modal thêm tài khoản, quét Chrome on-demand, phản ứng in-place mutation và nạp thẻ lên UI tổng. |
| `internal/ui/add_modal_test.go` | Tạo mới | Bộ unit test kiểm thử nil safety, empty state, 3 trạng thái phân cấp, in-place reaction, email masking, Chrome scan và dialog overlay. |
| `reports/coder/2026-10-22-phase3-add-account-modal-report.md` | Tạo mới | Báo cáo thi công chính thức Phase 3 kèm đầy đủ bằng chứng kiểm thử và bản đồ bất biến. |
| `scripts/build_windows.ps1` | Giữ nguyên 100% | 0 lines diff |
| `README.md` | Giữ nguyên 100% | 0 lines diff |
| `internal/ui/window.go` | Giữ nguyên 100% | 0 lines diff (nút toolbar thuộc Phase 4) |

Bàn giao trạng thái: **[READY_FOR_SUPERVISOR]**.
