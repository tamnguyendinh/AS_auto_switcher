# Supervisor Report: Nghiệm Thu Độc Lập Phase 2 - Tầng Lưu Trữ & Quản Lý Dữ Liệu (Account Store & Account Manager)

Verdict: PASS

## Metadata
- Report file: `reports/supervisor/2026-10-21-phase2-account-status-store-manager-acceptance.md`
- Review time: 2026-10-21 22:23:00 +07:00 (Asia/Bangkok)
- Reviewer: Supervisor Lane
- Repo/project: antigravity-auto-switch
- Scope reviewed: Triển khai BƯỚC 2 (PHASE 2) - Tầng Lưu trữ & Thực thể Dữ liệu (Store & Manager) tại commit `a11a378`
- Claim reviewed: Phase 2 triển khai đầy đủ, chính xác theo Đặc tả Kiến trúc `SPEC-ARCH-20261021-ACCOUNT-STATUS-ON-OFF-AUTO-STOP`, đảm bảo 100% các bất biến hệ thống (`Invariant-Account-Status-Persistence`, `Invariant-Backward-Compatibility-Default-On`, `Invariant-Token-Update-Preservation`, `Invariant-Unauth-Card-Isolation`), `scripts/build_windows.ps1` giữ nguyên 0 lines diff, 100% test suite và clean build Windows đạt Exit Code 0.
- Authority used: 
  - `reports/architect/2026-10-21-account-status-on-off-and-auto-stop-spec.md` (`SPEC-ARCH-20261021-ACCOUNT-STATUS-ON-OFF-AUTO-STOP`, commit `47ae804`)
  - Git commit `a11a378` (`feat(store,manager): implement phase 2 account status storage and manager layer`)
  - Supervisor Skill (`g:\antigravity_auto_switch\.agents\skills\supervisor\SKILL.md`)
  - Báo cáo thi công Coder: `reports/coder/2026-10-21-phase2-account-status-store-manager-report.md`
- Related artifacts: None

---

## Executive Summary
- **Problem:** Kiểm định độc lập Zero-Trust đối với việc thực thi Phase 2: Mở rộng `StoredAccountEnvelope`, bổ sung `SetAccountStatus` và `GetAccountStatus` ở Store và Manager, bảo tồn trạng thái `"off"` khi ghi đè token credential, và ánh xạ trường `Status` trong `AccountDisplayItem` tại `GetAccountStates`.
- **Decision:** **PASS**. Toàn bộ mã nguồn triển khai đúng hợp đồng kiến trúc, xử lý đầy đủ các điều kiện biên, thread-safe qua mutex/RWMutex, bảo tồn atomic write, vượt qua 100% test suite độc lập (`go test ./internal/store/...`, `go test ./internal/manager/...`, `go test ./...`) và script build repo `pwsh -File .\scripts\build_windows.ps1 -Clean` đạt Exit Code 0.
- **Required outcome:** Nghiệm thu Phase 2 và sẵn sàng bàn giao cho CEO / Planner / Coder để triển khai Phase 3 (Tầng Điều phối Giao diện & Xử lý Sự kiện `internal/ui/app.go`).

---

## Source-Level Clearance Notes

### 1. `internal/store/account_store.go` (CLEAR)
- **Struct `StoredAccountEnvelope`:**
  - File/Line: `internal/store/account_store.go:27-33`
  - Bổ sung trường `Status string json:"status,omitempty"` với chú thích chuẩn `"on" or "off", defaults to "on" if empty`. Khi parse JSON legacy không có trường `status`, trường nhận giá trị `""` và được nạp mặc định là `"on"`.
- **Hàm `SaveAccount(email string, token *model.TokenData) error`:**
  - File/Line: `internal/store/account_store.go:73-112`
  - Đảm bảo `Invariant-Token-Update-Preservation`: Đọc `targetPath` trên đĩa trước khi ghi đè. Nếu tệp tồn tại và `existingEnv.Status` là `"off"` (so sánh case-insensitive `strings.EqualFold`), giữ nguyên `status = "off"`. Ngược lại gán `"on"`.
  - Giữ nguyên cơ chế atomic temp file + `Sync()` + `Close()` + Windows retry rename.
- **Hàm `SetAccountStatus(email string, status string) error`:**
  - File/Line: `internal/store/account_store.go:155-225`
  - Kiểm tra `email` hợp lệ (`ErrInvalidEmail`). Chuẩn hóa `normStatus`: nếu khác `"off"` thì đưa về `"on"`.
  - Sử dụng khóa đồng bộ `s.mu.Lock()`, đọc an toàn `SafeAccountPath`.
  - Xử lý lỗi không tìm thấy tài khoản (`ErrAccountNotFound`) và JSON hỏng (`ErrCorruptedStoreFile`).
  - Ghi tệp an toàn bằng file tạm `acc_tmp_*.json`, đồng bộ ổ đĩa `Sync()` và rename nguyên tử.
- **Hàm `GetAccountStatus(email string) (string, error)`:**
  - File/Line: `internal/store/account_store.go:227-260`
  - Sử dụng khóa đọc `s.mu.RLock()`.
  - Kiểm tra email rỗng (`"on", ErrInvalidEmail`), tệp không tồn tại (`"on", ErrAccountNotFound`), JSON hỏng (`"on", ErrCorruptedStoreFile`).
  - Đảm bảo `Invariant-Backward-Compatibility-Default-On`: Nếu `env.Status` rỗng hoặc khác `"off"` $\rightarrow$ trả về `"on", nil`. Chỉ khi `env.Status == "off"` mới trả về `"off", nil`.

### 2. `internal/manager/account_manager.go` (CLEAR)
- **Struct `AccountDisplayItem`:**
  - File/Line: `internal/manager/account_manager.go:38`
  - Chuẩn hóa trường `Status string json:"status"` ("on" hoặc "off").
- **Hàm `SetAccountStatus(email string, status string) error`:**
  - File/Line: `internal/manager/account_manager.go:168-177`
  - Đọc khóa `m.mu.RLock()`, kiểm tra `m.store != nil`, ủy quyền gọi `st.SetAccountStatus(email, status)`.
- **Hàm `GetAccountStatus(email string) string`:**
  - File/Line: `internal/manager/account_manager.go:179-192`
  - Trả về status hoặc fallback an toàn `"on"` khi store nil hoặc có lỗi.
- **Hàm `AccountStore() *store.AccountStore`:**
  - File/Line: `internal/manager/account_manager.go:163-166`
  - Cung cấp getter chuẩn khớp với đặc tả kiến trúc (alias cho `Store()`).
- **Hàm `GetAccountStates(ctx context.Context)`:**
  - File/Line: `internal/manager/account_manager.go:370-373`
  - Duyệt danh sách tài khoản, khởi tạo `item.Status = "on"`.
  - Nếu `item.HasToken` (tài khoản đã có credential), nạp `item.Status = m.GetAccountStatus(item.Email)`.
  - Đảm bảo `Invariant-Unauth-Card-Isolation`: tài khoản unauth (`!item.HasToken`) giữ nguyên `Status = "on"` và không đọc store (UI Phase 1 ẩn nút `[ON/OFF]`).

### 3. `internal/store/account_store_test.go` & `internal/manager/account_manager_test.go` (CLEAR)
- File/Line: `internal/store/account_store_test.go:478-665`
  - `TestAccountStore_Status_Persistence_And_Preservation`: kiểm tra chu trình lưu ban đầu ("on"), chuyển sang "off", ghi đè token giữ nguyên "off", chuyển lại "on", và đọc tệp legacy không có trường status mặc định "on".
  - `TestAccountStore_Status_ErrorAndEdgeCases`: kiểm tra tài khoản không tồn tại, email rỗng, chuẩn hóa hoa/thường, chuỗi bất kỳ khác "off", và xử lý tệp JSON bị hỏng.
- File/Line: `internal/manager/account_manager_test.go:1139-1284`
  - `TestAccountManager_SetAndGetAccountStatus`: kiểm tra ủy quyền quản lý status qua AccountManager và fallback khi store nil.
  - `TestAccountManager_GetAccountStates_WithStatus`: kiểm tra tích hợp `GetAccountStates` ánh xạ chính xác status "on", "off", legacy file, và unauth.

### 4. `scripts/build_windows.ps1` (CLEAR)
- Kiểm tra diff `git diff 47ae804..a11a378 -- scripts/build_windows.ps1`:
  - Kết quả: **0 lines diff** (giữ nguyên 100%, không bị sửa đổi ngoài phạm vi).

---

## Evidence Checked

### Passed:
1. **Kiểm thử Store Package độc lập:**
   - Command: `go test -v ./internal/store/...`
   - Verification Freshness: Fresh (thực thi lúc 22:21:43)
   - Kết quả: 15/15 test passed, bao gồm `TestAccountStore_Status_Persistence_And_Preservation` và `TestAccountStore_Status_ErrorAndEdgeCases` (4.879s).
2. **Kiểm thử Manager Package độc lập:**
   - Command: `go test -v ./internal/manager/...`
   - Verification Freshness: Fresh (thực thi lúc 22:21:55)
   - Kết quả: 31/31 test passed, bao gồm `TestAccountManager_SetAndGetAccountStatus` và `TestAccountManager_GetAccountStates_WithStatus` (5.066s).
3. **Kiểm thử toàn bộ Repository:**
   - Command: `go test ./...`
   - Verification Freshness: Fresh (thực thi lúc 22:22:10)
   - Kết quả: 100% tất cả các package pass.
4. **Biên dịch độc lập bằng script chuẩn của Repo:**
   - Command: `pwsh -File .\scripts\build_windows.ps1 -Clean`
   - Verification Freshness: Fresh (thực thi lúc 22:22:38)
   - Kết quả:
     - `[1/4] Cleaning output artifacts... Clean complete.`
     - `[2/4] Running pre-flight tests & vet... [PASS] Verification succeeded.`
     - `[3/4] Compiling target binaries... [SUCCESS] Compiled PE resource object... Created: bin\antigravity-switch.exe (27.77 MB)`
     - `[4/4] Executing quick verification... [PASS] Embedded PE application icon verified on bin\antigravity-switch.exe (32x32)`
     - `⚡ BUILD SUCCESSFUL` - Exit Code: `0`.

### Failed:
- None.

### Not run:
- None.

---

## Invariant Closure

| Mã Invariant | Trạng Thái | Minh Chứng Mã Nguồn & Kiểm Thử |
|---|---|---|
| `Invariant-Account-Status-Persistence` | **ĐÃ ĐÓNG (CLOSED)** | `SetAccountStatus` ghi nguyên tử vào `accounts_store/<email>.json`, `GetAccountStatus` đọc chính xác trạng thái từ đĩa. Đã kiểm chứng qua `TestAccountStore_Status_Persistence_And_Preservation`. |
| `Invariant-Backward-Compatibility-Default-On` | **ĐÃ ĐÓNG (CLOSED)** | Tệp JSON legacy thiếu trường `status` hoặc mang chuỗi rỗng luôn trả về `"on"`, không yêu cầu migration. Đã kiểm chứng qua `TestAccountStore_Status_Persistence_And_Preservation` và `TestAccountManager_GetAccountStates_WithStatus`. |
| `Invariant-Token-Update-Preservation` | **ĐÃ ĐÓNG (CLOSED)** | `SaveAccount` kiểm tra `targetPath` trước khi ghi, đọc trạng thái hiện tại và bảo tồn tuyệt đối `status == "off"`. Đã kiểm chứng qua step 3 trong `TestAccountStore_Status_Persistence_And_Preservation`. |
| `Invariant-Unauth-Card-Isolation` | **ĐÃ ĐÓNG (CLOSED)** | `GetAccountStates` gán mặc định `"on"` cho thẻ chưa có token và không truy vấn store. Kết hợp với Phase 1 widget (`widgets.go:564-569`), nút `[ON/OFF]` bị ẩn hoàn toàn trên thẻ unauth. |

- Sibling surfaces checked: `internal/store/account_store.go`, `internal/store/account_store_test.go`, `internal/manager/account_manager.go`, `internal/manager/account_manager_test.go`, `scripts/build_windows.ps1`, `internal/ui/widgets.go`.
- Residual unverified same-invariant surfaces: `none`.

---

## Overall Evaluation
Mã nguồn triển khai của Phase 2 đạt tiêu chuẩn kỹ thuật cao, đáp ứng 100% đặc tả kiến trúc `SPEC-ARCH-20261021-ACCOUNT-STATUS-ON-OFF-AUTO-STOP` và các nguyên tắc Zero-Trust:
- Không xuất hiện sai lệch kiến trúc hoặc phá vỡ cấu trúc dữ liệu hiện hành.
- Khả năng bảo tồn cờ "off" trong chu trình ghi đè token hoạt động tin cậy.
- Không chỉnh sửa các file ngoài phạm vi được giao, script build Windows được bảo toàn tuyệt đối.
- Toàn bộ suite test và build script đều hoàn thành xuất sắc với Exit Code 0.

Nghiệm thu chính thức: **PASS**. Sẵn sàng chuyển tiếp sang Phase 3.
