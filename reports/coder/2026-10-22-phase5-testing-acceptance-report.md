# BÁO CÁO THI CÔNG: PHASE 5 - BỘ KIỂM THỬ CHẤP NHẬN TOÀN DIỆN (COMPREHENSIVE ACCEPTANCE TEST SUITE TC-01 -> TC-08)

- **Người thực hiện:** Coder
- **Ngày thực hiện:** 2026-10-22
- **Trạng thái:** COMPLETED / READY FOR SUPERVISOR REVIEW
- **Tài liệu tham chiếu tối cao (SSOT):** `reports/architect/2026-10-22-add-account-modal-and-discovered-filtering-spec.md` (Mã SPEC-ARCH-20261022-ADD-ACCOUNT-MODAL-AND-DISCOVERED-FILTERING-SPEC, commit a77efb9, Phần V Bước 5, Phần VI Bộ tiêu chuẩn kiểm thử chấp nhận)
- **Cam kết kỷ luật:**
  - Không sửa bất kỳ file nào ngoài scope Phase 5 (0 lines diff trên `scripts/build_windows.ps1`, `README.md`, và toàn bộ file mã nguồn production).
  - Không dùng lệnh Git diện rộng. Chỉ `git add` từng file trong scope được giao.
  - Coder không tự cấp PASS; bàn giao `[READY_FOR_SUPERVISOR]`.

---

## 1. BẢN ĐỒ BẤT BIẾN KIẾN TRÚC (INVARIANT FAMILY MAP)

| Thuộc tính | Chi tiết xác thực |
|---|---|
| **Family Name** | AddAccountModalAndDiscoveredFilteringAcceptance Invariant |
| **SSOT / Authority Source** | `reports/architect/2026-10-22-add-account-modal-and-discovered-filtering-spec.md` § Phần V Bước 5, Phần VI Bộ tiêu chuẩn kiểm thử chấp nhận |
| **Sibling Runtime Surfaces** | `internal/store/discovered_store.go` (`DiscoveredAccountEnvelope.Added`, `SetDiscoveredAdded`, `ListDiscoveredRecords`), `internal/manager/account_manager.go` (`GetAccountStates` decoupling, `ScanChromeAccounts`, `GetCandidateAccounts`, `AddDiscoveredAccount`), `internal/ui/add_modal.go` (`buildAddAccountModalContent`, `ShowAddAccountModal`), `internal/ui/window.go` (`addAccountBtn` in `toolbarLeft`), `internal/ui/app.go` (`Init`, `overview`, `updateUI`), `internal/automation/controller.go` & `sorter.go` (`FilterEligibleCandidates`), `internal/telemetry/pool.go` (`PollAll`) |
| **Forbidden Legacy Fallback** | 1. Quét Chrome tự động ngầm khi mở ứng dụng, chuyển tài khoản hoặc failover.<br/>2. Các tài khoản từ Chrome tự ý xuất hiện thành thẻ trên UI tổng mà không được người dùng bấm `[ + ADD ]`.<br/>3. Tài khoản `[Cần OAuth]` bị Burn-down Sorter chọn để chuyển tự động.<br/>4. Tài khoản `[Cần OAuth]` kích hoạt gọi API đo hạn ngạch Google ngầm.<br/>5. Mất cờ `Added: true` hoặc thẻ `[Cần OAuth]` biến mất khi khởi động lại ứng dụng. |
| **Stale Tests / Helpers Updated** | Đã bao phủ toàn diện 8 kịch bản chấp nhận TC-01 -> TC-08 trong file độc lập `internal/ui/add_modal_acceptance_test.go` mà không làm thay đổi các test hiện hữu. |
| **Verify Matrix** | 1. **TC-01 (Khởi động sạch):** `discovered_accounts/` có 5 email (`Added: false`), `accounts_store/` có 2 email -> UI tổng hiển thị đúng 2 thẻ từ `accounts_store/`. Tuyệt đối không xuất hiện 5 thẻ từ Chrome.<br/>2. **TC-02 (Mở Modal [+ Thêm]):** 1 email có token, 1 email đã add, 3 email chưa add -> Modal hiển thị đúng 3 trạng thái: 1 dòng `[✔ Đã OAuth]`, 1 dòng `[Đã add]`, 3 dòng `[Cần OAuth]` + `[ + ADD ]`.<br/>3. **TC-03 (Nhấn nút [ + ADD ]):** Nhấn `[ + ADD ]` tại `test@gmail.com` -> Nhãn đổi thành `[Đã add]`, nút ADD biến mất, thẻ xuất hiện ngoài UI tổng với badge `[Cần OAuth]` và Quota `N/A`.<br/>4. **TC-04 (Tính bền vững - Restart App):** Tắt app sau TC-03 rồi mở lại -> Thẻ `test@gmail.com` vẫn hiển thị nguyên vẹn trên UI tổng ở trạng thái `[Cần OAuth]`.<br/>5. **TC-05 (Quét Chrome On-Demand):** Bấm `[🔄 Quét Chrome]` -> Quét profiles Chrome, ghi nhận email mới vào `discovered_accounts/` (`Added: false`). Modal nạp thêm dòng mới. UI tổng không bị thêm thẻ tự động.<br/>6. **TC-06 (Hoàn tất OAuth sau khi add):** Giả lập đăng nhập lưu token vào `accounts_store/` -> Thẻ UI tổng đổi thành `[Chuyển]` / `[✔ Dùng]`. Mở Modal thấy chuyển thành `[✔ Đã OAuth]`.<br/>7. **TC-07 (An toàn Auto-Switch & Telemetry):** `FilterEligibleCandidates` tự động loại bỏ thẻ `[Cần OAuth]` qua Criterion 2. Vòng lặp telemetry chỉ poll các tài khoản có token trong `accounts_store/`, không bao giờ gọi API cho tài khoản `[Cần OAuth]`. Không panic, không crash.<br/>8. **TC-08 (Hiệu năng Khởi động):** Đo thời gian chạy `GetAccountStates()` nạp UI từ đĩa -> Dưới 50ms (thực tế đo được trung bình 4.90ms), không bị nghẽn bởi Chrome profile I/O. |

---

## 2. NỘI DUNG MÃ NGUỒN KIỂM THỬ ĐÃ THI CÔNG

Tệp tin mới: [`internal/ui/add_modal_acceptance_test.go`](file:///g:/antigravity_auto_switch/internal/ui/add_modal_acceptance_test.go)

### 2.1. Helper Khởi Tạo Môi Trường Cô Lập
- `createAcceptanceManagerInDir(t *testing.T, baseDir string) (*manager.AccountManager, *store.AccountStore, *store.DiscoveredStore)`:
  - Khởi tạo `AccountStore` và `DiscoveredStore` trong thư mục tạm riêng biệt `baseDir`.
  - Giả lập `reloader.SessionReloader` an toàn không tác động tiến trình hệ thống.
  - Thiết lập thư mục Chrome giả lập rỗng `isolated_chrome_empty` đảm bảo không bao giờ chạm vào Chrome thật trên máy host.
  - Thiết lập mock handler Windows Credential Manager: trả về `wincred.ErrNotFound` khi chưa gán token, giúp cách ly 100% khỏi credential của máy người phát triển.
- `extractModalRow(t *testing.T, obj fyne.CanvasObject) (string, *fyne.Container)`:
  - Trích xuất nhãn email và container `actionSlot` từ mỗi hàng trong danh sách modal ứng viên.

### 2.2. Chi Tiết Triển Khai 8 Kịch Bản Chấp Nhận (TC-01 -> TC-08)

#### TC-01: `TestAcceptance_TC01_CleanStartup`
- Nạp 5 email Chrome vào `discovered_accounts/` (`Added: false`) và 2 email vào `accounts_store/`.
- Khởi động app (`app.Init()`), chờ xử lý hoàn tất (`waitForIdle`).
- Xác thực:
  - `app.overview.Accounts` có chính xác 2 tài khoản.
  - Cả 2 tài khoản đều từ `accounts_store/` và có `HasToken: true`.
  - Không có bất kỳ email nào trong 5 tài khoản Chrome xuất hiện trong danh sách.
  - Lưới giao diện `app.accountListContainer.Objects` hiển thị chính xác 2 widget thẻ.

#### TC-02: `TestAcceptance_TC02_OpenAddModal_ThreeStates`
- Tạo 5 tài khoản trong `discovered_accounts/`:
  - 1 tài khoản đã có token: `user1_oauth@company.com`.
  - 1 tài khoản đã add: `user2_added@school.edu` (`Added: true`).
  - 3 tài khoản chưa add: `user3_raw@gmail.com`, `user4_raw@gmail.com`, `user5_raw@gmail.com` (`Added: false`).
- Khởi tạo app và gọi `app.buildAddAccountModalContent()`.
- Xác thực:
  - Danh sách modal `listContainer.Objects` có chính xác 5 dòng.
  - `user1_oauth@company.com`: hiển thị duy nhất nhãn `[✔ Đã OAuth]`, không có nút.
  - `user2_added@school.edu`: hiển thị duy nhất nhãn `[Đã add]`, không có nút.
  - 3 tài khoản còn lại: hiển thị nhãn cảnh báo `[Cần OAuth]` kèm nút bấm `ADD` (`widget.HighImportance`).

#### TC-03: `TestAcceptance_TC03_TapAddButton_InPlaceMutationAndOverviewCard`
- Tạo tài khoản `test@gmail.com` trong `discovered_accounts/` (`Added: false`).
- Mở modal, định vị dòng của `test@gmail.com`.
- Thực hiện bấm nút `addBtn.OnTapped()`.
- Xác thực:
  - Phản ứng tại chỗ (In-Place Mutation): `actionSlot` ngay lập tức chuyển thành duy nhất nhãn `[Đã add]`, nút `ADD` biến mất.
  - Độ bền vững trên đĩa: file `discovered_accounts/test@gmail.com.json` được cập nhật `Added: true`.
  - Phản ứng trên UI tổng: goroutine nền nạp ngay thẻ `test@gmail.com` lên `app.overview` với badge `[Cần OAuth]`, `State == model.StateUnauth`, `HasToken == false`.
  - Widget thẻ ngoài UI tổng hiển thị hạn ngạch Quota `N/A` trên cả hai chu kỳ `gemini5hCell` và `geminiWeekCell`.

#### TC-04: `TestAcceptance_TC04_PersistenceAcrossAppRestart`
- Mô phỏng trạng thái sau TC-03: `test@gmail.com` có `Added: true` trên đĩa cùng 1 tài khoản đã có token `persisted_user@company.com`.
- Khởi chạy Phiên 1 (`app1.Init()`), xác thực hiển thị 2 tài khoản, sau đó đóng app (`app1.Close()`).
- Tái khởi động: Khởi chạy Phiên 2 (`app2.Init()`) từ cùng cấu trúc thư mục trên đĩa.
- Xác thực:
  - `app2.overview.Accounts` nạp đầy đủ 2 tài khoản.
  - Thẻ `test@gmail.com` vẫn hiển thị nguyên vẹn với badge `[Cần OAuth]`, `State == model.StateUnauth`.
  - Giao diện `app2.accountListContainer.Objects` render đầy đủ 2 thẻ.

#### TC-05: `TestAcceptance_TC05_OnDemandChromeScan`
- App đang chạy với 1 tài khoản ban đầu trong `accounts_store/`.
- Chuẩn bị thư mục profile Chrome giả lập với 2 tài khoản mới (`scanned_chrome1@gmail.com`, `scanned_chrome2@gmail.com`).
- Thiết lập đường dẫn Chrome qua `mgr.SetChromeDir(mockChromeDir)`.
- Mở modal và bấm nút `scanBtn.OnTapped()`.
- Chờ tiến trình nền hoàn tất (`scanBtn` re-enable với nhãn `"Quét Chrome"`).
- Xác thực:
  - `discovered_accounts/` ghi nhận 2 bản ghi mới với `Added: false`.
  - Modal danh bạ nạp thêm 2 dòng ứng viên mới.
  - UI tổng giữ nguyên số lượng thẻ ban đầu (`afterCount == initCount`), tuyệt đối không tự ý thêm thẻ.

#### TC-06: `TestAcceptance_TC06_CompleteOAuthAfterAdd`
- Tài khoản `test@gmail.com` đã được add (`Added: true`), hiển thị `[Cần OAuth]` ngoài UI tổng.
- Giả lập người dùng đăng nhập qua Antigravity IDE, lưu token vào `accounts_store/test@gmail.com.json`.
- Kích hoạt làm mới `app.Refresh()`, `waitForIdle`.
- Xác thực:
  - Thẻ trên UI tổng chuyển đổi thành `[Chuyển]` (hoặc `[✔ Dùng]`), `HasToken: true`.
  - Modal danh sách ứng viên chuyển đổi dòng `test@gmail.com` thành `[✔ Đã OAuth]`.

#### TC-07: `TestAcceptance_TC07_AutoSwitchAndTelemetrySafety`
- Thiết lập hệ thống gồm: 1 tài khoản active (`active@company.com`), 1 tài khoản ready (`ready@company.com`, quota dồi dào), và 1 tài khoản `[Cần OAuth]` (`unauth@company.com`, `Added: true`, không có token).
- Tích hợp mock quota client theo dõi danh sách email bị truy vấn.
- Xác thực 1 (Auto-Switch Burn-down Sorter):
  - Chạy `automation.FilterEligibleCandidates`.
  - Thẻ `unauth@company.com` bị loại trừ hoàn toàn với Criterion 2 ("token invalid or refresh token missing").
  - Danh sách eligible chỉ còn duy nhất `ready@company.com`.
- Xác thực 2 (Telemetry Pool Safety):
  - Chạy `app.telemetryPool.PollAll(context.Background())`.
  - Quota API client chỉ được gọi cho tài khoản có token, tuyệt đối không bị gọi cho `unauth@company.com`.
  - Toàn bộ quá trình diễn ra an toàn, không có panic, không có crash.

#### TC-08: `TestAcceptance_TC08_StartupPerformanceBenchmark`
- Nạp 15 tài khoản vào danh bạ (5 đã add, 10 chưa add) và 5 tài khoản vào `accounts_store/`.
- Trỏ đường dẫn Chrome sang thư mục không tồn tại để chứng minh tách rời hoàn toàn khỏi Chrome I/O.
- Thực hiện đo đạc benchmark `mgr.GetAccountStates(context.Background())` qua 10 lượt chạy liên tiếp.
- Xác thực:
  - Kết quả trả về chính xác 10 tài khoản (5 tài khoản có token + 5 tài khoản có cờ `Added: true`).
  - Thời gian đo được: **trung bình 4.90ms**, **lớn nhất 12.55ms** (vượt xa tiêu chuẩn yêu cầu < 50ms).

---

## 3. BẰNG CHỨNG KIỂM CHỨNG CHẤT LƯỢNG (E2E VERIFICATION)

### 3.1. Kết Quả Chạy Trực Tiếp Bộ 8 Kiểm Thử Chấp Nhận TC-01 -> TC-08
- **Lệnh thực thi:** `go test -v -run "TestAcceptance_TC" ./internal/ui`
- **Kết quả:** `PASS` 100% (8/8 tests pass, 3.739s)
  ```text
  === RUN   TestAcceptance_TC01_CleanStartup
  --- PASS: TestAcceptance_TC01_CleanStartup (0.25s)
  === RUN   TestAcceptance_TC02_OpenAddModal_ThreeStates
  --- PASS: TestAcceptance_TC02_OpenAddModal_ThreeStates (0.10s)
  === RUN   TestAcceptance_TC03_TapAddButton_InPlaceMutationAndOverviewCard
  --- PASS: TestAcceptance_TC03_TapAddButton_InPlaceMutationAndOverviewCard (0.08s)
  === RUN   TestAcceptance_TC04_PersistenceAcrossAppRestart
  --- PASS: TestAcceptance_TC04_PersistenceAcrossAppRestart (0.25s)
  === RUN   TestAcceptance_TC05_OnDemandChromeScan
  --- PASS: TestAcceptance_TC05_OnDemandChromeScan (0.17s)
  === RUN   TestAcceptance_TC06_CompleteOAuthAfterAdd
  --- PASS: TestAcceptance_TC06_CompleteOAuthAfterAdd (0.22s)
  === RUN   TestAcceptance_TC07_AutoSwitchAndTelemetrySafety
  --- PASS: TestAcceptance_TC07_AutoSwitchAndTelemetrySafety (0.69s)
  === RUN   TestAcceptance_TC08_StartupPerformanceBenchmark
      add_modal_acceptance_test.go:843: TC-08 Startup Benchmark: 10 runs, avg=4.90211ms, max=12.552ms (tiêu chuẩn: < 50ms)
  --- PASS: TestAcceptance_TC08_StartupPerformanceBenchmark (0.32s)
  PASS
  ok  	antigravity-switch/internal/ui	3.739s
  ```

### 3.2. Toàn Bộ Suite Test Package `internal/ui`
- **Lệnh thực thi:** `go test ./internal/ui/...`
- **Kết quả:** `PASS` 100% (24.367s)
  ```text
  ok  	antigravity-switch/internal/ui	24.367s
  ```

### 3.3. Toàn Bộ Test Toàn Repository
- **Lệnh thực thi:** `go test ./...`
- **Kết quả:** `PASS` 100%
  ```text
  ok  	antigravity-switch/cmd/switcher	(cached)
  ok  	antigravity-switch/internal/automation	(cached)
  ok  	antigravity-switch/internal/connector	(cached)
  ok  	antigravity-switch/internal/manager	(cached)
  ok  	antigravity-switch/internal/model	(cached)
  ok  	antigravity-switch/internal/reloader	(cached)
  ok  	antigravity-switch/internal/scanner	(cached)
  ok  	antigravity-switch/internal/store	(cached)
  ok  	antigravity-switch/internal/telemetry	(cached)
  ok  	antigravity-switch/internal/ui	41.548s
  ok  	antigravity-switch/internal/wincred	(cached)
  ?   	antigravity-switch/reports/supervisor/evidence/autonomous_switch_runtime_review_261020	[no test files]
  ```

### 3.4. Biên Dịch Toàn Diện Hệ Thống Bằng Script của Repo
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
  ok  	antigravity-switch/cmd/switcher	(cached)
  ok  	antigravity-switch/internal/automation	(cached)
  ok  	antigravity-switch/internal/connector	(cached)
  ok  	antigravity-switch/internal/manager	(cached)
  ok  	antigravity-switch/internal/model	(cached)
  ok  	antigravity-switch/internal/reloader	(cached)
  ok  	antigravity-switch/internal/scanner	(cached)
  ok  	antigravity-switch/internal/store	5.630s
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
    Compile Duration: 7278 ms | Size: 27.8 MB (29153792 bytes)
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

| Tệp tin | Trạng thái | Mô tả thay đổi |
|---|---|---|
| `internal/ui/add_modal_acceptance_test.go` | Created | Tạo mới bộ kiểm thử chấp nhận toàn diện bao phủ 8 kịch bản TC-01 -> TC-08 |
| `reports/coder/2026-10-22-phase5-testing-acceptance-report.md` | Created | Báo cáo thi công chính thức Phase 5 kèm bằng chứng kiểm chứng |
| `scripts/build_windows.ps1` | Unchanged | Giữ nguyên 100% (0 lines diff) |
| `README.md` | Unchanged | Giữ nguyên 100% (0 lines diff) |

---

## 5. KẾT LUẬN & TRẠNG THÁI BÀN GIAO

- **Tiến độ:** Hoàn thành 100% Phase 5 (Bộ Kiểm thử Chấp nhận Toàn diện TC-01 -> TC-08) theo đúng Đặc tả Kiến trúc chính thức `reports/architect/2026-10-22-add-account-modal-and-discovered-filtering-spec.md`.
- **Chất lượng:** Tất cả 8 kịch bản TC-01 -> TC-08 vượt qua 100%. Toàn bộ test suite repository pass 100%. Script build `scripts/build_windows.ps1 -Clean` hoàn thành thành công Exit Code 0.
- **Trạng thái:** `[READY_FOR_SUPERVISOR]`
