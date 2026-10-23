# Supervisor Report: Nghiệm thu Nâng cấp Modal Thêm Tài Khoản (Nút Xóa & Trạng thái Tài khoản không tồn tại)

Verdict: PASS

## Metadata
- **Report file:** `reports/supervisor/rp_supervisor_20261023_add_modal_delete_and_nonexistent_account_acceptance.md`
- **Review time:** 2026-10-23 13:55:00 +07:00
- **Reviewer:** Supervisor Lane
- **Repo/project:** `antigravity-switch`
- **Scope reviewed:** Commit `49d18abeffbf48dd616d3c980bfbc2ac698db47e`, báo cáo `reports/coder/rp_coder_20261023_add_modal_delete_and_nonexistent_account.md`, Touch-Map 7 tệp mã nguồn và kiểm thử.
- **Claim reviewed:** Thi công hoàn tất 100% tính năng Nâng cấp Modal Thêm Tài Khoản (Nút Xóa [ 🗑 ] phổ quát trên mọi dòng và Trạng thái mới `[Tài khoản không tồn tại]`) chuẩn xác theo Đặc tả Kiến trúc, bảo toàn 4 Supreme Invariants, vượt qua toàn bộ test suite độc lập và build sạch Windows PE binary.
- **Authority used:** `reports/architect/rp_architect_20261023_131500_add_modal_delete_and_nonexistent_account_spec.md`, `supervisor` skill, `Edge-Case` skill.
- **Related artifacts:** 
  - `reports/architect/rp_architect_20261023_131500_add_modal_delete_and_nonexistent_account_spec.md`
  - `reports/coder/rp_coder_20261023_add_modal_delete_and_nonexistent_account.md`

---

## Executive Summary
- **Problem:** Thẩm định Zero-Trust toàn diện kết quả triển khai commit `49d18abeffbf48dd616d3c980bfbc2ac698db47e` của Coder Lane đối với Modal "Thêm tài khoản", giải quyết hai thiếu sót lịch sử:
  1. Thiếu nút Xóa dòng tài khoản phát hiện trong `discovered_accounts/`.
  2. Không nhận diện được các tài khoản hiển thị trên UI nhưng file JSON trong `discovered_accounts/` đã bị xóa hoặc mất trên đĩa cứng (tài khoản ma).
- **Decision:** **PASS**. Toàn bộ mã nguồn triển khai chuẩn xác theo từng hàm, từng struct được đặc tả trong Spec kiến trúc. Kiểm thử độc lập tươi mới (`-count=1`) trên các gói `internal/store`, `internal/manager`, `internal/ui` và toàn bộ repository đều đạt 100% PASS. Bản build Windows PE bằng script chuẩn `build_windows.ps1 -Clean` biên dịch thành công nhị phân `bin/antigravity-switch.exe` hoàn chỉnh.
- **Required outcome:** Chấp thuận (Accepted) và sẵn sàng tích hợp sản xuất.

---

## Source-Level Clearance Notes

### 1. Tầng Store: `internal/store/discovered_store.go` [CLEARED]
- `DeleteDiscovered(email string) error`:
  - Chuẩn hóa email với `strings.ToLower(strings.TrimSpace(email))`.
  - Kiểm tra rỗng trả về `ErrInvalidEmail`.
  - Sử dụng `SafeAccountPath(s.dir, email)` chặn triệt để lỗ hổng path traversal.
  - Sử dụng mutex lock `s.mu.Lock()` đảm bảo thread-safe.
  - Xử lý `if os.IsNotExist(err) { return nil }` đảm bảo tính `Idempotent` tuyệt đối: nếu file không tồn tại, thao tác xóa coi như thành công và trả về `nil`.
- `HasDiscovered(email string) bool`:
  - Khóa đọc `s.mu.RLock()` / `defer s.mu.RUnlock()`.
  - Sử dụng `os.Stat(path)` và kiểm tra `!info.IsDir()` đảm bảo đường dẫn phải là file thực sự, không nhận nhầm thư mục.
- Kiểm tra `go vet ./internal/store/...`: Không có cảnh báo hoặc lỗi.

### 2. Tầng Manager: `internal/manager/account_manager.go` [CLEARED]
- Model `CandidateAccountItem`:
  - Đã bổ sung trường `FileExists bool` phản ánh chính xác trạng thái tồn tại file JSON trong `discovered_accounts/`.
- Quản lý bộ nhớ theo dõi tài khoản:
  - Đã bổ sung `trackedAccounts map[string]bool` với các hàm quản lý an toàn đa luồng: `TrackAccount()`, `UntrackAccount()`.
  - `GetAccountStates()` tự động ghi nhận các tài khoản `StateUnauth` hoặc chưa có token vào danh bạ theo dõi.
- `DeleteDiscoveredAccount(email string) error`:
  - Thu hồi tài khoản khỏi `trackedAccounts`.
  - Gọi `discoveredStore.DeleteDiscovered(normEmail)`.
  - **Invariant-Store-Isolation:** Tuyệt đối không chạm vào `accounts_store/` hoặc gọi `RemoveAccount()`.
- `GetCandidateAccounts(knownEmails ...string)`:
  - Đọc danh bạ thực từ `discoveredStore.ListDiscoveredRecords()` và gán `FileExists: true`.
  - Quét bổ sung các tài khoản ma từ `trackedAccounts` và `knownEmails` nếu `!discovered.HasDiscovered(norm)`, thiết lập `FileExists: false`.
  - Sắp xếp ưu tiên: `HasToken` -> `Added` -> Tên email (A-Z).
- Kiểm tra `go vet ./internal/manager/...`: Hoàn toàn sạch lỗi.

### 3. Tầng Giao diện UI: `internal/ui/add_modal.go` [CLEARED]
- Vòng đời giao diện và bộ nhớ tạm:
  - Sử dụng `dismissedAccounts := make(map[string]bool)` trong scope của phiên mở Modal.
- Nút Xóa phổ quát:
  - Render nút Xóa `widget.NewButtonWithIcon("", theme.DeleteIcon(), ...)` bọc trong `compactButton` (chiều cao 24px) trên TẤT CẢ 4 trạng thái dòng tài khoản.
  - Khi bấm Xóa: đánh dấu `dismissedAccounts[norm] = true`, gọi `a.mgr.DeleteDiscoveredAccount(candItem.Email)` nếu có file, và gọi ngay `reloadCandidates()` để gỡ dòng khỏi UI Modal ngay lập tức.
- Ma trận 4 trạng thái hiển thị:
  - Trạng thái 1 (`candItem.HasToken`): Badge `[✔ Đã OAuth]` (Success, Bold) + Nút [Xóa].
  - Trạng thái 2 (`candItem.Added`): Badge `[Đã add]` (TextMuted, Bold) + Nút [Xóa].
  - Trạng thái 3 (`!candItem.Added && !candItem.HasToken`): Badge `[Cần OAuth]` (Warning, Bold) + Nút [ADD] (HighImportance) + Nút [Xóa]. Khi bấm [ADD], chuyển đổi tại chỗ sang `[Đã add]` và giữ nguyên nút [Xóa].
  - Trạng thái 4 (`!candItem.FileExists`): Badge `[Tài khoản không tồn tại]` (TextMuted, Bold) + Nút [Xóa] (ẩn nút ADD).
- Nút `[Quét Chrome]`: Vô hiệu hóa nút và hiện "⏳ Đang quét...", chạy goroutine scan Chrome và cập nhật lại danh sách ứng viên qua `fyne.Do(reloadCandidates)`.
- Kiểm tra `go vet ./internal/ui/...`: Hoàn toàn sạch lỗi.

### 4. Hệ thống Kiểm thử: `internal/store/discovered_store_test.go`, `internal/manager/two_store_test.go`, `internal/ui/add_modal_test.go`, `internal/ui/add_modal_acceptance_test.go` [CLEARED]
- Toàn bộ các ca kiểm thử theo yêu cầu của Spec mục 6 đã được triển khai đầy đủ và bám sát thực tế:
  - `TestDiscoveredStore_DeleteDiscovered_Success`
  - `TestDiscoveredStore_DeleteDiscovered_Idempotent`
  - `TestDiscoveredStore_HasDiscovered`
  - `TestDeleteDiscoveredAccount_Isolation`
  - `TestGetCandidateAccounts_NonExistentDetection`
  - `TestModal_Delete_Button_Rendered`
  - `TestModal_Delete_Click_Action`
  - `TestModal_ScanChrome_NonExistent_Status`
  - `TestModal_Delete_NonExistent`
- Toàn bộ suite kiểm thử chấp thuận UI hiện hữu (`add_modal_acceptance_test.go`) được cập nhật tương thích 100% với cấu trúc `actionSlot` mới.

---

## Evidence Checked

### Passed:
1. **Fresh Store Unit Tests (`go test -v -count=1 ./internal/store/...`):**
   - 24/24 unit tests PASS (4.964s).
   - Bao gồm đầy đủ `DeleteDiscovered` (thành công & idempotent) và `HasDiscovered`.
2. **Fresh Manager Unit Tests (`go test -v -count=1 ./internal/manager/...`):**
   - 39/39 unit tests PASS (5.197s).
   - Bao gồm `TestDeleteDiscoveredAccount_Isolation` và `TestGetCandidateAccounts_NonExistentDetection`.
3. **Fresh UI Unit & Acceptance Tests (`go test -v -count=1 ./internal/ui/...`):**
   - 100% test cases PASS (25.942s).
   - Bao gồm cả 4 test cases cho nút Xóa, trạng thái tài khoản không tồn tại, và benchmark khởi động TC-08 (4.52ms < tiêu chuẩn 50ms).
4. **Full Repository Test Suite (`go test -p 2 ./...`):**
   - 100% packages PASS (`cmd/switcher`, `internal/automation`, `internal/connector`, `internal/manager`, `internal/model`, `internal/reloader`, `internal/scanner`, `internal/store`, `internal/telemetry`, `internal/ui`, `internal/version`, `internal/wincred`).
5. **Clean Windows PE Build (`powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Clean`):**
   - Bước [1/4] Dọn dẹp output artifacts sạch sẽ.
   - Bước [2/4] Kiểm tra pre-flight tests & vet PASS.
   - Bước [3/4] Biên dịch PE resource với windres (`rsrc_windows_amd64.syso`, 13490 bytes) và tạo binary GUI `bin/antigravity-switch.exe` (27.83 MB, CGO_ENABLED=1).
   - Bước [4/4] Xác minh embedded PE application icon 32x32 thành công.
6. **Code Vet (`go vet ./internal/store/... ./internal/manager/... ./internal/ui/...`):**
   - Không phát hiện bất kỳ lỗi lầm syntax, types hay dead code nào.

- **Verification freshness:** Tươi mới 100%, thực thi trực tiếp tại phiên nghiệm thu với cờ `-count=1` và `-Clean`.

### Failed:
- Không có (None).

### Not run:
- Không có (None).

---

## Invariant Closure

| Mã Invariant | Nội dung bất biến | Hiện trạng xác minh | Kết luận |
|:---|:---|:---|:---:|
| **`Invariant-Store-Isolation`** | Nút [Xóa] trong Modal Thêm tài khoản CHỈ xóa trong `discovered_accounts/`, không xóa trong `accounts_store/`, không làm mất thẻ tài khoản trên UI chính. | Đã đối soát code `DeleteDiscoveredAccount` và test `TestDeleteDiscoveredAccount_Isolation`: file `discovered_accounts/<email>.json` bị xóa, nhưng `accounts_store/<email>.json` và thẻ UI chính vẫn nguyên vẹn. | **CLOSED** |
| **`Invariant-Idempotent-Delete`** | Thao tác xóa trên file không tồn tại trả về `nil`, không crash, không báo lỗi giả. | Đã đối soát `DiscoveredStore.DeleteDiscovered` kiểm tra `os.IsNotExist(err)` trả về `nil`. Đã kiểm chứng qua `TestDiscoveredStore_DeleteDiscovered_Idempotent` (xóa file không tồn tại và xóa lặp 2 lần). | **CLOSED** |
| **`Invariant-Ghost-Account-State`** | Khi quét Chrome hoặc mở Modal, tài khoản có trên UI nhưng mất file JSON trong `discovered_accounts/` phải hiển thị badge `[Tài khoản không tồn tại]`, ẩn nút `[ADD]`, hiển thị nút `[Xóa]`. | Đã đối soát code `CandidateAccountItem.FileExists`, logic `checkGhost` trong `GetCandidateAccounts`, và giao diện trong `add_modal.go`. Đã kiểm chứng qua `TestModal_ScanChrome_NonExistent_Status`. | **CLOSED** |
| **`Invariant-Modal-Removal`** | Bấm nút [Xóa] trên bất kỳ dòng nào (kể cả dòng `[Tài khoản không tồn tại]`) phải lập tức loại bỏ dòng đó khỏi danh sách hiển thị Modal. | Đã đối soát `dismissedAccounts` kết hợp `reloadCandidates()`. Đã kiểm chứng qua `TestModal_Delete_Click_Action` và `TestModal_Delete_NonExistent`. | **CLOSED** |

- **Sibling surfaces checked:** Đã kiểm tra UI chính (`internal/ui/app.go`), quy trình xóa tài khoản active (`RemoveAccount`), kho lưu trữ `accounts_store/`, kho lưu trữ `discovered_accounts/`, và các cơ chế reload. Xác nhận hai chức năng xóa (Xóa tài khoản vận hành trên UI chính vs Xóa danh bạ khám phá trong Add Modal) hoàn toàn độc lập và không có xung đột.
- **Residual unverified same-invariant surfaces:** Không có (`none`).

---

## Overall Evaluation
Bản thi công tại commit `49d18abeffbf48dd616d3c980bfbc2ac698db47e` đạt chất lượng kỹ thuật xuất sắc:
1. **Tuân thủ kiến trúc:** Bám sát 100% bản đặc tả `rp_architect_20261023_131500_add_modal_delete_and_nonexistent_account_spec.md`, thực hiện đúng chữ ký hàm, cấu trúc dữ liệu, và logic giao diện.
2. **Chất lượng mã nguồn:** Thread-safe, idempotent, phòng thủ đường dẫn thư mục (`SafeAccountPath`), xử lý lỗi chặt chẽ, không có dead code.
3. **Độ tin cậy kiểm thử:** Các test case bao phủ toàn diện từ unit test tầng thấp đến acceptance test tầng giao diện, toàn bộ repo vượt qua 100% kiểm thử tươi mới.
4. **Sẵn sàng triển khai:** Bản build Windows PE sạch sẽ, nhị phân chạy độc lập đã sẵn sàng cho người dùng cuối.

**Phán quyết chính thức: `PASS`**
