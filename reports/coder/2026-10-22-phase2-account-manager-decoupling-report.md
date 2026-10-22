# BÁO CÁO THI CÔNG: PHASE 2 - TÁCH RỜI ĐIỀU PHỐI QUẢN LÝ TÀI KHOẢN (ACCOUNT MANAGER DECOUPLING)

- **Người thực hiện:** Coder
- **Ngày thực hiện:** 2026-10-22
- **Trạng thái:** COMPLETED / READY FOR SUPERVISOR REVIEW
- **Tài liệu tham chiếu tối cao (SSOT):** `reports/architect/2026-10-22-add-account-modal-and-discovered-filtering-spec.md` (Phần IV.2, V Bước 2)
- **Cam kết kỷ luật:**
  - Không sửa bất kỳ file nào ngoài scope Phase 2 (0 lines diff trên `scripts/build_windows.ps1`, `internal/ui/`, `README.md`).
  - Không dùng lệnh Git diện rộng. Chỉ `git add` từng file trong scope.
  - Coder không tự cấp PASS; bàn giao `[READY_FOR_SUPERVISOR]`.

---

## 1. BẢN ĐỒ BẤT BIẾN KIẾN TRÚC (INVARIANT FAMILY MAP)

| Thuộc tính | Chi tiết xác thực |
|---|---|
| **Family Name** | Account Manager Chrome-Decoupling & Discovered Filtering Invariant |
| **SSOT / Authority Source** | `reports/architect/2026-10-22-add-account-modal-and-discovered-filtering-spec.md` § Phần IV.2 |
| **Sibling Runtime Surfaces** | `internal/manager/account_manager.go` (`GetAccountStates`, `ScanChromeAccounts`, `CandidateAccountItem`, `GetCandidateAccounts`, `AddDiscoveredAccount`), `internal/store/discovered_store.go` (Envelope.Added, SetDiscoveredAdded, ListDiscoveredRecords - Phase 1) |
| **Forbidden Legacy Fallback** | `GetAccountStates` tự động quét Chrome ngầm; tự động lưu profile Chrome vào `discoveredStore`; nạp tài khoản chưa được add (`Added == false`) lên `view.Accounts` |
| **Stale Tests / Helpers Updated** | `TestTwoStoreCaptureMergeRepeatAndRemember` (chuyển sang gọi `ScanChromeAccounts` + `AddDiscoveredAccount`), `TestTwoStorePartialFailuresAndMissingSession` (căn chỉnh theo mô hình không quét Chrome ngầm) |
| **Verify Matrix** | 1. Quét Chrome on-demand độc lập (`ScanChromeAccounts`).<br/>2. `GetAccountStates` không quét ngầm Chrome và chỉ hiển thị tài khoản đã `Added == true` từ danh bạ ứng viên.<br/>3. `GetCandidateAccounts` sắp xếp phân cấp theo `HasToken` -> `Added` -> `Email` A-Z.<br/>4. `AddDiscoveredAccount` bật cờ bền vững `Added = true`.<br/>5. Build Windows script `pwsh -File .\scripts\build_windows.ps1 -Clean` Exit Code 0. |

---

## 2. NỘI DUNG MÃ NGUỒN ĐÃ THI CÔNG

### 2.1. `internal/manager/account_manager.go`
1. **Tái cấu trúc `GetAccountStates(ctx context.Context) (*AccountOverview, error)`:**
   - Cắt bỏ hoàn toàn biến `cDir` và lệnh gọi `scanner.ScanChromeAccountDetails(cDir)`.
   - Cắt bỏ hoàn toàn vòng lặp lưu Chrome vào `discoveredStore.SaveDiscovered`.
   - Đọc danh sách từ `discoveredStore.ListDiscoveredRecords()` và CHỈ nạp vào `itemsMap` các bản ghi có `rec.Added == true`.
   - Nạp các tài khoản từ `m.store.ListAccounts()` và `activeEmail`.
   - Giữ nguyên 100% phân cấp trạng thái `StateActive` (`[✔ Dùng]`), `StateReady` (`[Chuyển]`), `StateUnauth` (`[Cần OAuth]`).
2. **Bổ sung `ScanChromeAccounts() ([]string, error)`:**
   - Quét profile Chrome theo yêu cầu (on-demand) qua `scanner.ScanChromeAccountDetails(cDir)`.
   - Đồng bộ email quét được vào `discoveredStore.SaveDiscovered` (bảo toàn cờ `Added`).
   - Trả về danh sách email tìm thấy.
3. **Bổ sung struct `CandidateAccountItem`:**
   ```go
   type CandidateAccountItem struct {
       Email    string
       HasToken bool
       Added    bool
   }
   ```
4. **Bổ sung `GetCandidateAccounts() ([]CandidateAccountItem, error)`:**
   - Ghép nối dữ liệu từ `discoveredStore.ListDiscoveredRecords()` với `storedSet` từ `m.store.ListAccounts()`.
   - Sắp xếp chuẩn xác theo thứ tự ưu tiên: `HasToken` (true trước) -> `Added` (true trước) -> `Email` A-Z.
5. **Bổ sung `AddDiscoveredAccount(email string) error`:**
   - Ghi nhận cờ `Added = true` vào kho đệm qua `discoveredStore.SetDiscoveredAdded(email, true)`.

### 2.2. `internal/manager/two_store_test.go`
1. **Cập nhật `TestTwoStoreCaptureMergeRepeatAndRemember`:**
   - Gọi `m.ScanChromeAccounts()` và `m.AddDiscoveredAccount("NEW@example.com")` để xác nhận tài khoản ứng viên xuất hiện trên `overview.Accounts` đúng luồng mới.
2. **Cập nhật `TestTwoStorePartialFailuresAndMissingSession`:**
   - Giả lập lỗi hỏng bản ghi catalog để kiểm tra cảnh báo đọc danh bạ ứng viên và wincred độc lập, đảm bảo không quét Chrome ngầm và không làm mất credential trong store.
3. **Bổ sung 5 Unit Test mới:**
   - `TestGetAccountStates_DoesNotScanChrome`: Chứng minh `GetAccountStates` không đọc Chrome, không sinh file trong `discovered_accounts/`, không đưa email chưa add lên UI.
   - `TestScanChromeAccounts_OnDemand`: Chứng minh quét on-demand thành công, bản ghi lưu với `Added == false`, không làm xuất hiện thẻ trên UI tổng.
   - `TestAddDiscoveredAccount`: Chứng minh `AddDiscoveredAccount` kích hoạt `Added == true`, đưa thẻ `[Cần OAuth]` lên UI tổng và cờ `Added` được bảo toàn khi quét lại Chrome.
   - `TestGetCandidateAccounts_SortingAndEnrichment`: Chứng minh thứ tự sắp xếp và gắn cờ `HasToken` / `Added` chuẩn xác.
   - `TestCandidateAccounts_UninitializedStore`: Kiểm tra phòng vệ fail-close khi `discoveredStore == nil`.

---

## 3. BẰNG CHỨNG KIỂM CHỨNG CHẤT LƯỢNG (E2E VERIFICATION)

### 3.1. Unit Test Package `internal/manager`
- **Lệnh thực thi:** `go test -v ./internal/manager/...`
- **Kết quả:** `PASS` 100% (37/37 tests pass, 5.409s)
- **Danh sách test Phase 2:**
  - `TestGetAccountStates`: PASS
  - `TestTwoStoreCaptureMergeRepeatAndRemember`: PASS
  - `TestTwoStorePartialFailuresAndMissingSession`: PASS
  - `TestTwoStoreCaptureWriteFailureKeepsLiveIdentity`: PASS
  - `TestGetAccountStates_DoesNotScanChrome`: PASS
  - `TestScanChromeAccounts_OnDemand`: PASS
  - `TestAddDiscoveredAccount`: PASS
  - `TestGetCandidateAccounts_SortingAndEnrichment`: PASS
  - `TestCandidateAccounts_UninitializedStore`: PASS

### 3.2. Biên dịch Toàn diện Hệ thống bằng Script của Repo
- **Lệnh thực thi:** `pwsh -File .\scripts\build_windows.ps1 -Clean`
- **Kết quả:**
  ```text
  [1/4] Cleaning output artifacts...
    Clean complete.
  [2/4] Running pre-flight tests & vet...
    [PASS] Verification succeeded.
  [3/4] Compiling target binaries...
    [SUCCESS] Compiled PE resource object: cmd\switcher\rsrc_windows_amd64.syso (12552 bytes)
    [SUCCESS] Created: bin\antigravity-switch.exe
    Compile Duration: 6629 ms | Size: 27.77 MB (29113856 bytes)
  [4/4] Executing quick verification...
    [PASS] Embedded PE application icon verified on bin\antigravity-switch.exe (32x32)
    [INFO] GUI binary built at bin\antigravity-switch.exe.
  ================================================================================
  ⚡ BUILD SUCCESSFUL
  ================================================================================
  ```
- **Exit Code:** 0

---

## 4. BỀ MẶT THAY ĐỔI & RỦI RO (IMPACT ASSESSMENT)

- **Các tệp thay đổi:**
  - `internal/manager/account_manager.go` (+96 / -35 lines)
  - `internal/manager/two_store_test.go` (+182 / -10 lines)
  - `reports/coder/2026-10-22-phase2-account-manager-decoupling-report.md` (Tạo mới)
- **Tệp không bị ảnh hưởng:** Toàn bộ các package khác, UI, Scripts, README giữ nguyên 100%.
- **Residual unverified surfaces:** `none`
- **Kế hoạch tiếp theo:** Chuyển giao nghiệm thu Phase 2 cho Supervisor. Sau khi Supervisor cấp PASS, chuyển giao cho Coder thực hiện Phase 3 (Dựng Modal Thêm tài khoản `internal/ui/add_modal.go`).
