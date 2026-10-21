# Báo Cáo Thi Công: Phase 2 - Tầng Lưu Trữ & Quản Lý Dữ Liệu (Account Store & Account Manager)
(Account Status On/Off Storage & Manager Entity Implementation Report - Phase 2)

- **Ngày thực hiện:** 2026-10-21
- **Vai trò:** Coder
- **Thẩm quyền kiến trúc (Authority / SSOT):** `reports/architect/2026-10-21-account-status-on-off-and-auto-stop-spec.md` (`SPEC-ARCH-20261021-ACCOUNT-STATUS-ON-OFF-AUTO-STOP`, commit `47ae804`)
- **Trạng thái:** HOÀN THÀNH (READY_FOR_SUPERVISOR)

---

## I. TỔNG QUAN THI CÔNG & INVARIANT FAMILY MAP

### 1. Invariant Family Map
- **Family Name:** `Account Status Persistence & Invariant Preservation (Phase 2: Store & Manager)`
- **Authority / SSOT:** `reports/architect/2026-10-21-account-status-on-off-and-auto-stop-spec.md` (Mã `SPEC-ARCH-20261021-ACCOUNT-STATUS-ON-OFF-AUTO-STOP`, commit `47ae804`)
- **Core Invariants Enforced:**
  1. `Invariant-Account-Status-Persistence`: Trạng thái Bật/Tắt ("on"/"off") được lưu bền vững vào chính tệp `accounts_store/<email>.json` thông qua atomic temp file + `os.Rename`. Đọc trạng thái trực tiếp từ đĩa.
  2. `Invariant-Backward-Compatibility-Default-On`: Mọi tệp JSON hiện hữu không có trường `status` hoặc mang chuỗi rỗng đều được nạp với giá trị mặc định là `"on"`. Không yêu cầu migration script, bảo toàn 100% tài khoản cũ.
  3. `Invariant-Token-Update-Preservation`: Khi `SaveAccount` ghi đè tệp tài khoản (do refresh token OAuth hoặc capture session từ Windows Credential Manager), hệ thống đọc tệp hiện tại trên đĩa và bảo toàn nguyên vẹn giá trị `status == "off"`. Tuyệt đối không reset về `"on"`.
  4. `Invariant-Unauth-Card-Isolation`: Thẻ chưa có token (`[Cần OAuth]`) không đọc status từ store và mang giá trị mặc định chuẩn là `"on"`.

- **Sibling Surfaces Inspected & Closed:**
  1. `StoredAccountEnvelope` tại `internal/store/account_store.go`:
     - Bổ sung trường `Status string json:"status,omitempty"` ("on" hoặc "off", mặc định là "on" nếu rỗng).
  2. `SaveAccount(email, token)` tại `internal/store/account_store.go`:
     - Trước khi ghi đè, kiểm tra sự tồn tại của `targetPath` trên đĩa.
     - Nếu tệp đã có và `existingEnv.Status == "off"`, bắt buộc giữ nguyên `envelope.Status = "off"`. Ngược lại gán `"on"`.
     - Ghi tệp nguyên tử bằng temp file + `Sync()` + `Close()` + atomic rename (kèm fallback remove+rename trên Windows).
  3. `SetAccountStatus(email, status)` tại `internal/store/account_store.go`:
     - Kiểm tra `email` hợp lệ (nếu rỗng trả về `ErrInvalidEmail`).
     - Chuẩn hóa status: chuỗi chữ thường, loại bỏ khoảng trắng, nếu khác `"off"` thì gán `"on"`.
     - Kiểm tra tệp tồn tại (nếu thiếu trả về `ErrAccountNotFound`).
     - Giải mã an toàn envelope hiện hữu (nếu JSON lỗi trả về `ErrCorruptedStoreFile`).
     - Cập nhật trường `env.Status = normStatus`.
     - Ghi tệp nguyên tử bằng pattern `acc_tmp_*.json` + `os.Rename`.
  4. `GetAccountStatus(email)` tại `internal/store/account_store.go`:
     - Kiểm tra `email` hợp lệ (nếu rỗng trả về `"on", ErrInvalidEmail`).
     - Kiểm tra tệp tồn tại (nếu thiếu trả về `"on", ErrAccountNotFound`).
     - Giải mã envelope: nếu JSON hỏng trả về `"on", ErrCorruptedStoreFile`.
     - Nếu `env.Status` rỗng hoặc khác `"off"` $\rightarrow$ trả về `"on", nil` (chuẩn hóa backward compatibility).
     - Nếu `env.Status == "off"` $\rightarrow$ trả về `"off", nil`.
  5. `AccountDisplayItem` tại `internal/manager/account_manager.go`:
     - Chuẩn hóa trường `Status string json:"status"` ("on" hoặc "off").
  6. `SetAccountStatus(email, status)` tại `internal/manager/account_manager.go`:
     - Bổ sung hàm vào `AccountManager`, ủy quyền an toàn cho `m.store.SetAccountStatus`. Kiểm tra `m.store != nil`.
  7. `GetAccountStatus(email)` tại `internal/manager/account_manager.go`:
     - Bổ sung hàm vào `AccountManager`, ủy quyền an toàn cho `m.store.GetAccountStatus`, trả về status hợp lệ hoặc `"on"` nếu có lỗi.
  8. `AccountStore()` tại `internal/manager/account_manager.go`:
     - Bổ sung hàm getter (alias cho `Store()`) tương thích với đặc tả kiến trúc.
  9. `GetAccountStates(ctx)` tại `internal/manager/account_manager.go`:
     - Trong vòng lặp tổng hợp danh sách tài khoản, khởi tạo `item.Status = "on"`.
     - Nếu `item.HasToken == true`, gọi `m.GetAccountStatus(item.Email)` để lấy giá trị bền vững từ store.
  10. `internal/store/account_store_test.go`:
     - Thêm `TestAccountStore_Status_Persistence_And_Preservation`: kiểm tra lưu mặc định "on", đổi sang "off", ghi đè token bằng `SaveAccount` bảo toàn "off", đổi lại "on", và nạp file legacy cũ không có trường status mặc định là "on".
     - Thêm `TestAccountStore_Status_ErrorAndEdgeCases`: kiểm tra tài khoản không tồn tại, email rỗng, chuẩn hóa hoa/thường/khoảng trắng, giá trị không hợp lệ, và tệp JSON bị hỏng.
  11. `internal/manager/account_manager_test.go`:
     - Thêm `TestAccountManager_SetAndGetAccountStatus`: kiểm tra ủy quyền store, nạp status mặc định, chuyển đổi "on"/"off", xử lý store nil, và phương thức `AccountStore()`.
     - Thêm `TestAccountManager_GetAccountStates_WithStatus`: kiểm tra tích hợp `GetAccountStates` ánh xạ chính xác status "on", "off", legacy, và tài khoản unauth.
- **Legacy Fallback Status:** Tương thích ngược 100% với toàn bộ file tài khoản cũ trong `accounts_store/`.
- **Residual Unverified Surfaces:** `none`.

---

## II. CHI TIẾT CÁC TỆP SỬA ĐỔI (FILES CHANGED)

1. `internal/store/account_store.go`:
   - Mở rộng struct `StoredAccountEnvelope`: thêm `Status string json:"status,omitempty"`.
   - Cập nhật hàm `SaveAccount`: đọc tệp hiện có trên đĩa để bảo tồn `status == "off"`.
   - Bổ sung hàm `SetAccountStatus(email string, status string) error`.
   - Bổ sung hàm `GetAccountStatus(email string) (string, error)`.
2. `internal/manager/account_manager.go`:
   - Cập nhật struct `AccountDisplayItem`: `Status string json:"status"`.
   - Bổ sung `SetAccountStatus(email string, status string) error`.
   - Bổ sung `GetAccountStatus(email string) string`.
   - Bổ sung `AccountStore() *store.AccountStore`.
   - Cập nhật `GetAccountStates`: nạp trạng thái status từ store cho từng tài khoản có credential.
3. `internal/store/account_store_test.go`:
   - Thêm `TestAccountStore_Status_Persistence_And_Preservation`.
   - Thêm `TestAccountStore_Status_ErrorAndEdgeCases`.
4. `internal/manager/account_manager_test.go`:
   - Bổ sung import `"antigravity-switch/internal/wincred"`.
   - Thêm `TestAccountManager_SetAndGetAccountStatus`.
   - Thêm `TestAccountManager_GetAccountStates_WithStatus`.

---

## III. KẾT QUẢ KIỂM CHỨNG & BẰNG CHỨNG THỰC NGHIỆM (E2E VERIFICATION)

### 1. Kiểm thử tự động Store & Manager (Unit Tests)
- **Lệnh thực thi kiểm thử Store Status:** `go test -v ./internal/store/... -run TestAccountStore_Status`
  ```text
  === RUN   TestAccountStore_Status_Persistence_And_Preservation
  --- PASS: TestAccountStore_Status_Persistence_And_Preservation (0.11s)
  === RUN   TestAccountStore_Status_ErrorAndEdgeCases
  --- PASS: TestAccountStore_Status_ErrorAndEdgeCases (0.06s)
  PASS
  ok  	antigravity-switch/internal/store	0.905s
  ```

- **Lệnh thực thi kiểm thử Manager Status:** `go test -v ./internal/manager/... -run "TestAccountManager_SetAndGetAccountStatus|TestAccountManager_GetAccountStates_WithStatus"`
  ```text
  === RUN   TestAccountManager_SetAndGetAccountStatus
  --- PASS: TestAccountManager_SetAndGetAccountStatus (0.37s)
  === RUN   TestAccountManager_GetAccountStates_WithStatus
  --- PASS: TestAccountManager_GetAccountStates_WithStatus (1.12s)
  PASS
  ok  	antigravity-switch/internal/manager	3.566s
  ```

- **Lệnh thực thi toàn bộ Store & Manager packages:** `go test ./internal/store/... ./internal/manager/...`
  ```text
  ok  	antigravity-switch/internal/store	14.974s
  ok  	antigravity-switch/internal/manager	8.657s
  ```

- **Lệnh thực thi toàn bộ test repository:** `go test ./...`
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
  ok  	antigravity-switch/internal/ui	(cached)
  ok  	antigravity-switch/internal/wincred	(cached)
  PASS (100%)
  ```

### 2. Biên dịch hệ thống bằng script của repo (Windows Build Verification)
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
  ok  	antigravity-switch/cmd/switcher	1.931s
  ok  	antigravity-switch/internal/automation	(cached)
  ok  	antigravity-switch/internal/connector	(cached)
  ok  	antigravity-switch/internal/manager	(cached)
  ok  	antigravity-switch/internal/model	(cached)
  ok  	antigravity-switch/internal/reloader	(cached)
  ok  	antigravity-switch/internal/scanner	(cached)
  ok  	antigravity-switch/internal/store	4.729s
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
    Compile Duration: 6199 ms | Size: 27.77 MB (29113856 bytes)
  [4/4] Executing quick verification...
    [PASS] Embedded PE application icon verified on bin\antigravity-switch.exe (32x32)
    [INFO] GUI binary built at bin\antigravity-switch.exe.
  ================================================================================
  ⚡ BUILD SUCCESSFUL
  ================================================================================
  ```
- **Exit Code:** `0`

### 3. Kỷ luật mã nguồn & Script Build
- `scripts/build_windows.ps1`: Giữ nguyên 100% (`0 lines diff`).
- Phạm vi thay đổi: Đúng và đủ cho Phase 2 (`internal/store/`, `internal/manager/`, tests tương ứng, và báo cáo thi công). Không sửa đổi Phase 3, 4, 5. Không dùng anvien.

---

## IV. KẾT LUẬN & SẴN SÀNG BÀN GIAO
Toàn bộ yêu cầu của Phase 2 đã được hoàn thành độc lập, bảo đảm 100% các bất biến hệ thống:
1. `SetAccountStatus` và `GetAccountStatus` hoạt động chính xác, bền vững và tương thích ngược.
2. `SaveAccount` bảo toàn nguyên vẹn cờ `status == "off"`.
3. `AccountManager` liên kết thông suốt tầng Store và tầng hiển thị `AccountDisplayItem`.
4. Test suite và build script pass 100%.

Sẵn sàng bàn giao cho Supervisor Lane nghiệm thu và chuyển tiếp sang Phase 3.
