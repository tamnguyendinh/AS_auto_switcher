# BÁO CÁO THI CÔNG (CODER IMPLEMENTATION REPORT)

- **Mã báo cáo:** `rp_coder_20261023_add_modal_delete_and_nonexistent_account`
- **Tác giả:** Coder Lane
- **Ngày thi công:** 2026-10-23
- **Thẩm quyền kiến trúc (SSOT):** `reports/architect/rp_architect_20261023_131500_add_modal_delete_and_nonexistent_account_spec.md` (commit `ff7bbbd1626c60f628b25225e769c92f3e36b81f`)
- **Trạng thái:** **`VERDICT: PASS / READY_FOR_QA`**

---

## 1. INVARIANT FAMILY MAP

- **Family Name:** Add Modal Account Deletion and Non-Existent Account State
- **SSOT / Authority:** `reports/architect/rp_architect_20261023_131500_add_modal_delete_and_nonexistent_account_spec.md`
- **Core Invariants:**
  1. `Invariant-Store-Isolation`: Nút [Xóa] trong Add Account Modal CHỈ thao tác xóa file JSON trong `discovered_accounts/`. Tuyệt đối không xóa trong `accounts_store/`, không gọi `RemoveAccount()`, và không làm biến mất thẻ tài khoản trên màn hình chính (`UI chính`).
  2. `Invariant-Idempotent-Delete`: Hàm `DeleteDiscovered(email)` xóa an toàn và idempotent: nếu file không tồn tại trên đĩa (`os.IsNotExist`), trả về `nil` thành công, không ném lỗi giả hay crash.
  3. `Invariant-Ghost-Account-State`: Khi nhấn nút `[Quét Chrome]` (hoặc tải danh sách modal), bất kỳ tài khoản nào nằm trong theo dõi modal (hoặc thẻ `StateUnauth` / cờ `Added` từ UI chính) mà kiểm tra trên đĩa cứng không có file JSON trong `discovered_accounts/` (`FileExists == false`) -> BẮT BUỘC hiển thị trạng thái `[Tài khoản không tồn tại]` (palette.TextMuted, Bold), ẩn nút `ADD`, hiển thị nút `[Xóa]`.
  4. `Invariant-Modal-Removal`: Bấm nút `[Xóa]` trên bất kỳ dòng tài khoản nào trong modal (kể cả dòng `[Tài khoản không tồn tại]`) lập tức loại bỏ dòng đó khỏi danh sách hiển thị Modal (`reloadCandidates()`).

---

## 2. CHI TIẾT THI CÔNG THEO TỪNG TẦNG (IMPLEMENTATION DETAILS)

### 2.1. Tầng Store (`internal/store/discovered_store.go`)
- Cài đặt `DeleteDiscovered(email string) error`:
  - Chuẩn hóa email `strings.ToLower(strings.TrimSpace(email))`.
  - Validate email rỗng trả về `ErrInvalidEmail`.
  - Khóa mutex `s.mu.Lock()`, xóa file `os.Remove(path)`. Nếu lỗi `os.IsNotExist(err)` trả về `nil` (idempotent).
- Cài đặt `HasDiscovered(email string) bool`:
  - Khóa đọc `s.mu.RLock()`, kiểm tra `os.Stat(path)` tồn tại và không phải directory.

### 2.2. Tầng Manager (`internal/manager/account_manager.go`)
- Cập nhật struct `CandidateAccountItem`:
  ```go
  type CandidateAccountItem struct {
      Email      string
      HasToken   bool
      Added      bool
      FileExists bool // true: file JSON tồn tại; false: không tồn tại
  }
  ```
- Thêm trường `trackedAccounts map[string]bool` trong `AccountManager` để quản lý các tài khoản đã được nạp/thêm vào UI.
- Thêm các hàm quản lý tài khoản theo dõi:
  - `TrackAccount(email string)` và `UntrackAccount(email string)` thread-safe.
  - Cập nhật `AddDiscoveredAccount(email string)` ghi nhận tài khoản vào `trackedAccounts`.
  - Cập nhật `GetAccountStates()` ghi nhận các tài khoản `StateUnauth` hoặc chưa có token vào `trackedAccounts`.
- Cài đặt `DeleteDiscoveredAccount(email string) error`:
  - Xóa tài khoản khỏi `trackedAccounts`.
  - Gọi `discoveredStore.DeleteDiscovered(email)`.
  - Tuyệt đối không can thiệp vào `accounts_store/`.
- Nâng cấp `GetCandidateAccounts(knownEmails ...string) ([]CandidateAccountItem, error)`:
  - Đọc toàn bộ danh bạ từ `discoveredStore.ListDiscoveredRecords()`, đánh dấu `FileExists: true`.
  - Duyệt qua các tài khoản từ `trackedAccounts` và `knownEmails`: nếu tài khoản chưa có trong records và `!discovered.HasDiscovered(email)` -> bổ sung ứng viên với `FileExists: false`, `Added: true`.
  - Giữ nguyên thuật toán sắp xếp ưu tiên: `HasToken -> Added -> Email (A-Z)`.

### 2.3. Tầng Giao diện UI (`internal/ui/add_modal.go`)
- Thêm bộ nhớ theo dõi tạm thời `dismissedAccounts := make(map[string]bool)` trong phạm vi mở modal.
- Bổ sung nút Xóa phổ quát `widget.NewButtonWithIcon("", theme.DeleteIcon(), ...)` (`compactButton`, `widget.LowImportance`) trên TẤT CẢ các dòng tài khoản.
- Render đầy đủ 4 trạng thái theo đúng State Matrix của Spec:
  1. `!candItem.FileExists`: Hiển thị badge `[Tài khoản không tồn tại]` (palette.TextMuted, Bold), ẩn nút ADD, hiển thị nút Xóa.
  2. `candItem.HasToken`: Hiển thị badge `[✔ Đã OAuth]` (palette.Success, Bold), hiển thị nút Xóa.
  3. `candItem.Added`: Hiển thị badge `[Đã add]` (palette.TextMuted, Bold), hiển thị nút Xóa.
  4. `!candItem.Added && !candItem.HasToken`: Hiển thị badge `[Cần OAuth]` (palette.Warning, Bold), nút `[ADD]`, nút Xóa.
- Hành vi khi nhấn nút Xóa:
  - Đánh dấu `dismissedAccounts[candItem.Email] = true`.
  - Nếu `candItem.FileExists == true`: gọi `a.mgr.DeleteDiscoveredAccount(candItem.Email)`.
  - Gọi `reloadCandidates()` để gỡ ngay dòng tài khoản khỏi modal.
- Nút `[Quét Chrome]`: scan tài khoản Chrome và gọi `reloadCandidates()`. Trong quá trình reload, các tài khoản trên UI mất file JSON sẽ tự động chuyển sang `[Tài khoản không tồn tại]`.

---

## 3. DANH SÁCH FILE THAY ĐỔI (TOUCH-MAP)

| Đường dẫn file | Nội dung thay đổi |
|:---|:---|
| `internal/store/discovered_store.go` | Cài đặt `DeleteDiscovered(email)` (idempotent) & `HasDiscovered(email)`. |
| `internal/store/discovered_store_test.go` | Thêm 3 unit tests: `TestDiscoveredStore_DeleteDiscovered_Success`, `TestDiscoveredStore_DeleteDiscovered_Idempotent`, `TestDiscoveredStore_HasDiscovered`. |
| `internal/manager/account_manager.go` | Bổ sung `FileExists` vào `CandidateAccountItem`, `DeleteDiscoveredAccount`, `TrackAccount`, nâng cấp `GetCandidateAccounts`. |
| `internal/manager/two_store_test.go` | Thêm 2 unit tests: `TestDeleteDiscoveredAccount_Isolation`, `TestGetCandidateAccounts_NonExistentDetection`. |
| `internal/ui/add_modal.go` | Hiển thị nút Xóa trên 4 trạng thái, xử lý `[Tài khoản không tồn tại]`, ẩn nút ADD, gỡ dòng khi xóa. |
| `internal/ui/add_modal_test.go` | Cập nhật cấu trúc `actionSlot` và thêm 4 unit tests: `TestModal_Delete_Button_Rendered`, `TestModal_Delete_Click_Action`, `TestModal_ScanChrome_NonExistent_Status`, `TestModal_Delete_NonExistent`. |
| `internal/ui/add_modal_acceptance_test.go` | Cập nhật các helper kiểm tra `actionSlot` tương thích với nút Xóa mới trên TC-02, TC-03, TC-06. |

---

## 4. KẾT QUẢ KIỂM THỬ (VERIFICATION EVIDENCE)

### 4.1. Unit Tests Tầng Store (`internal/store`)
```text
=== RUN   TestDiscoveredStore_DeleteDiscovered_Success
--- PASS: TestDiscoveredStore_DeleteDiscovered_Success (0.01s)
=== RUN   TestDiscoveredStore_DeleteDiscovered_Idempotent
--- PASS: TestDiscoveredStore_DeleteDiscovered_Idempotent (0.01s)
=== RUN   TestDiscoveredStore_HasDiscovered
--- PASS: TestDiscoveredStore_HasDiscovered (0.02s)
PASS
ok  	antigravity-switch/internal/store	(5.064s)
```

### 4.2. Unit Tests Tầng Manager (`internal/manager`)
```text
=== RUN   TestDeleteDiscoveredAccount_Isolation
--- PASS: TestDeleteDiscoveredAccount_Isolation (0.03s)
=== RUN   TestGetCandidateAccounts_NonExistentDetection
--- PASS: TestGetCandidateAccounts_NonExistentDetection (0.03s)
PASS
ok  	antigravity-switch/internal/manager	(5.826s)
```

### 4.3. Unit & Acceptance Tests Tầng UI (`internal/ui`)
```text
=== RUN   TestModal_Delete_Button_Rendered
--- PASS: TestModal_Delete_Button_Rendered (0.19s)
=== RUN   TestModal_Delete_Click_Action
--- PASS: TestModal_Delete_Click_Action (0.05s)
=== RUN   TestModal_ScanChrome_NonExistent_Status
--- PASS: TestModal_ScanChrome_NonExistent_Status (0.08s)
=== RUN   TestModal_Delete_NonExistent
--- PASS: TestModal_Delete_NonExistent (0.03s)
=== RUN   TestShowAddAccountModal_NilSafety
--- PASS: TestShowAddAccountModal_NilSafety (0.00s)
=== RUN   TestShowAddAccountModal_EmptyCandidates
--- PASS: TestShowAddAccountModal_EmptyCandidates (0.17s)
=== RUN   TestShowAddAccountModal_ThreeCandidateStates
--- PASS: TestShowAddAccountModal_ThreeCandidateStates (0.09s)
=== RUN   TestShowAddAccountModal_AddActionInPlaceMutationAndOverviewUpdate
--- PASS: TestShowAddAccountModal_AddActionInPlaceMutationAndOverviewUpdate (0.10s)
=== RUN   TestShowAddAccountModal_EmailMasking
--- PASS: TestShowAddAccountModal_EmailMasking (0.06s)
=== RUN   TestShowAddAccountModal_ScanChromeButtonBehavior
--- PASS: TestShowAddAccountModal_ScanChromeButtonBehavior (0.09s)
=== RUN   TestShowAddAccountModal_DialogDisplayAndOverlay
--- PASS: TestShowAddAccountModal_DialogDisplayAndOverlay (0.06s)
=== RUN   TestAcceptance_TC01_CleanStartup
--- PASS: TestAcceptance_TC01_CleanStartup (0.17s)
=== RUN   TestAcceptance_TC02_OpenAddModal_ThreeStates
--- PASS: TestAcceptance_TC02_OpenAddModal_ThreeStates (0.10s)
=== RUN   TestAcceptance_TC03_TapAddButton_InPlaceMutationAndOverviewCard
--- PASS: TestAcceptance_TC03_TapAddButton_InPlaceMutationAndOverviewCard (0.08s)
=== RUN   TestAcceptance_TC04_PersistenceAcrossAppRestart
--- PASS: TestAcceptance_TC04_PersistenceAcrossAppRestart (0.24s)
=== RUN   TestAcceptance_TC05_OnDemandChromeScan
--- PASS: TestAcceptance_TC05_OnDemandChromeScan (0.23s)
=== RUN   TestAcceptance_TC06_CompleteOAuthAfterAdd
--- PASS: TestAcceptance_TC06_CompleteOAuthAfterAdd (0.22s)
=== RUN   TestAcceptance_TC07_AutoSwitchAndTelemetrySafety
--- PASS: TestAcceptance_TC07_AutoSwitchAndTelemetrySafety (0.62s)
=== RUN   TestAcceptance_TC08_StartupPerformanceBenchmark
--- PASS: TestAcceptance_TC08_StartupPerformanceBenchmark (0.30s)
PASS
ok  	antigravity-switch/internal/ui	(26.277s)
```

### 4.4. Full Repository Test Suite
```text
ok  	antigravity-switch/cmd/switcher	3.037s
ok  	antigravity-switch/internal/connector	2.692s
ok  	antigravity-switch/internal/model	0.924s
ok  	antigravity-switch/internal/reloader	3.870s
ok  	antigravity-switch/internal/scanner	1.662s
ok  	antigravity-switch/internal/version	0.978s
ok  	antigravity-switch/internal/wincred	1.629s
ok  	antigravity-switch/internal/store	15.148s
ok  	antigravity-switch/internal/manager	9.200s
ok  	antigravity-switch/internal/ui	37.431s
ok  	antigravity-switch/internal/automation	16.602s
```
**Tổng tỷ lệ:** PASS 100% toàn bộ repository.

---

## 5. E2E FLOW: TRIGGER -> PROCESS -> OBSERVABLE RESULT

```text
E2E Verification:
  [PASS] Compiled: go test -c ./internal/ui/... -> Thành công không lỗi
  [PASS] Runtime: Khởi chạy modal Thêm tài khoản qua App GUI Fyne test harness -> Hoạt động mượt mà
  [PASS] Happy path 1 (Xóa tài khoản có file): Bấm nút Xóa [ 🗑 ] trên dòng tài khoản -> file JSON bị xóa trong discovered_accounts/, dòng biến mất khỏi modal, thẻ UI chính vẫn giữ nguyên
  [PASS] Happy path 2 (Tài khoản không tồn tại): Xóa file JSON trên đĩa -> Bấm [Quét Chrome] -> dòng hiển thị [Tài khoản không tồn tại], nút ADD bị ẩn, nút Xóa hiển thị
  [PASS] Happy path 3 (Xóa tài khoản không tồn tại): Bấm nút Xóa [ 🗑 ] trên dòng [Tài khoản không tồn tại] -> dòng biến mất ngay lập tức khỏi modal
  [PASS] Edge case: Gọi DeleteDiscovered 2 lần trên file không tồn tại -> Idempotent, trả về nil không crash
```

---

## 6. THÔNG TIN BÀN GIAO CHO QA & SUPERVISOR

- **Sibling surfaces checked:** Đã kiểm tra UI chính, kho lưu trữ `accounts_store/`, và `discovered_accounts/`. Xác nhận phân lập 100% giữa hai nút Xóa.
- **Legacy fallback status:** Đã loại bỏ các giả định cũ về actionSlot không có nút Xóa; tất cả các test case đều đã được cập nhật đồng bộ.
- **Residual unverified surfaces:** `none`
- **Risks/open points:** `none`
- **Kết luận:** Đạt 100% tiêu chí chấp thuận theo đặc tả kiến trúc.
