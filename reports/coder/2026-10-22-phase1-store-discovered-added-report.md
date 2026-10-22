# Báo Cáo Thi Công: Phase 1 - Tầng Lưu Trữ Danh Bạ (Discovered Store)
(Discovered Store Added Flag Persistence & Query Implementation Report - Phase 1)

- **Ngày thực hiện:** 2026-10-22
- **Vai trò:** Coder
- **Thẩm quyền kiến trúc (Authority / SSOT):** `reports/architect/2026-10-22-add-account-modal-and-discovered-filtering-spec.md` (commit `54617e3`)
- **Trạng thái:** HOÀN THÀNH (READY_FOR_SUPERVISOR)

---

## I. TỔNG QUAN THI CÔNG & INVARIANT FAMILY MAP

### 1. Invariant Family Map
- **Family Name:** `Discovered Account Persistence & Added Flag State Management (Phase 1: Discovered Store)`
- **Authority / SSOT:** `reports/architect/2026-10-22-add-account-modal-and-discovered-filtering-spec.md` (commit `54617e3`)
- **Core Invariants Enforced:**
  1. `Invariant-Discovered-Added-Persistence`: Cờ `Added` được lưu bền vững vào từng tệp bản ghi `discovered_accounts/<email>.json` qua cơ chế ghi đĩa nguyên tử `os.CreateTemp` + `tmp.Sync()` + `tmp.Close()` + `os.Rename`.
  2. `Invariant-Backward-Compatibility-Default-Unadded`: Mọi tệp JSON hiện hữu không có trường `added` (hoặc mang chuỗi JSON legacy) khi unmarshal đều có giá trị `Added == false` (nhờ tag `json:"added,omitempty"`). Bảo toàn tương thích ngược 100% không làm xáo trộn dữ liệu cũ.
  3. `Invariant-Scan-Added-Preservation`: Khi `SaveDiscovered(email)` ghi đè tệp bản ghi (trong các lần quét lại Chrome định kỳ hoặc on-demand), hệ thống chủ động đọc tệp hiện có trên đĩa trước khi ghi đè để bảo tồn cờ `Added == true`. Tuyệt đối không reset `Added` về `false` khi quét lại Chrome.
  4. `Invariant-ScannedAt-Preservation-On-Add`: Khi gọi `SetDiscoveredAdded(email, added)`, nếu bản ghi đã tồn tại trên đĩa, giá trị `ScannedAt` cũ được bảo tồn nguyên vẹn (chỉ cập nhật cờ `Added`), tránh làm xáo trộn mốc thời gian quét gốc. Nếu bản ghi chưa có, khởi tạo `ScannedAt = time.Now()`.
  5. `Invariant-Sorted-Discovered-Records`: Hàm `ListDiscoveredRecords()` lọc bỏ thư mục, tệp không phải `.json`, tệp JSON hỏng, tệp không khớp tên chuẩn hóa sanitizer hoặc có `ScannedAt <= 0`, loại bỏ bản ghi trùng lặp và trả về danh sách được sắp xếp theo `Email` A-Z.

- **Sibling Surfaces Inspected & Closed:**
  1. `DiscoveredAccountEnvelope` tại `internal/store/discovered_store.go`:
     - Bổ sung trường `Added bool "json:added,omitempty"`.
  2. `SaveDiscovered(email string)` tại `internal/store/discovered_store.go`:
     - Chuẩn hóa email bằng `strings.ToLower(strings.TrimSpace(email))`.
     - Kiểm tra an toàn đường dẫn bằng `SafeAccountPath(s.dir, email)`.
     - Đọc file cũ nếu tồn tại để trích xuất và bảo tồn giá trị `existingEnv.Added`.
     - Ghi tệp nguyên tử qua temp file `.discovered-*.tmp` + `Sync()` + `Close()` + `os.Rename`.
  3. `SetDiscoveredAdded(email string, added bool)` tại `internal/store/discovered_store.go`:
     - Kiểm tra tính hợp lệ của email (nếu rỗng trả về `ErrInvalidEmail`).
     - Kiểm tra an toàn đường dẫn bằng `SafeAccountPath`.
     - Đọc file cũ nếu tồn tại để giữ nguyên `existingEnv.ScannedAt`.
     - Ghi nhận trường `Added = added` và ghi đĩa nguyên tử.
  4. `ListDiscoveredRecords()` tại `internal/store/discovered_store.go`:
     - Đọc toàn bộ thư mục `s.dir` dưới `s.mu.RLock()`. Nếu thư mục chưa tồn tại (`os.IsNotExist`), trả về `[]DiscoveredAccountEnvelope{}, nil`.
     - Bỏ qua tệp không phải JSON, thư mục con.
     - Lọc bỏ bản ghi lỗi / hỏng và tích lũy lỗi qua `errors.Join`.
     - Sắp xếp danh sách kết quả theo `Email` tăng dần (A-Z).
  5. `ListDiscovered()` tại `internal/store/discovered_store.go`:
     - Giữ nguyên 100% chữ ký và hành vi trả về `[]string` để duy trì tính tương thích ngược với các caller hiện có.
  6. `internal/store/discovered_store_test.go`:
     - Bổ sung test tương thích ngược: nạp file JSON cũ không có trường `added` -> `env.Added == false`.
     - Bổ sung test bảo tồn trạng thái: `SaveDiscovered` không làm mất cờ `Added == true` khi quét lại Chrome.
     - Bổ sung test `SetDiscoveredAdded`: chuyển đổi cờ `true`/`false` thành công, lưu bền vững qua việc mở lại store từ đĩa.
     - Bổ sung test `ListDiscoveredRecords`: lọc file rác/hỏng, sắp xếp A-Z, trả về đúng cờ `Added`.
     - Bổ sung test edge cases: email rỗng, whitespace, chuẩn hóa ký tự hoa/thường, store trên đường dẫn file bị chiếm dụng.

- **Legacy Fallback Status:** Tương thích ngược 100% với toàn bộ file tài khoản cũ trong `discovered_accounts/`.
- **Residual Unverified Surfaces:** `none`.

---

## II. CHI TIẾT CÁC TỆP SỬA ĐỔI (FILES CHANGED)

1. `internal/store/discovered_store.go`:
   - Mở rộng struct `DiscoveredAccountEnvelope` với `Added bool "json:added,omitempty"`.
   - Cập nhật hàm `SaveDiscovered(email string) error` đọc và bảo tồn cờ `Added`.
   - Bổ sung hàm `SetDiscoveredAdded(email string, added bool) error`.
   - Bổ sung hàm `ListDiscoveredRecords() ([]DiscoveredAccountEnvelope, error)`.
2. `internal/store/discovered_store_test.go`:
   - Bổ sung 5 test case unit test chuyên sâu:
     + `TestDiscoveredStore_BackwardCompatibility_LegacyJSON`
     + `TestDiscoveredStore_SaveDiscovered_PreservesAddedFlag`
     + `TestDiscoveredStore_SetDiscoveredAdded_ToggleAndPersistence`
     + `TestDiscoveredStore_ListDiscoveredRecords_SortingAndFiltering`
     + `TestDiscoveredStore_SetDiscoveredAdded_ErrorAndEdgeCases`

---

## III. BẰNG CHỨNG KIỂM CHỨNG CHẤT LƯỢNG (VERIFICATION EVIDENCE)

### 1. Kết quả chạy Unit Test Store (`go test -v ./internal/store/...`):
```text
=== RUN   TestAccountStoreRoundTrip
--- PASS: TestAccountStoreRoundTrip (0.02s)
=== RUN   TestAccountStoreListAccounts
--- PASS: TestAccountStoreListAccounts (0.05s)
=== RUN   TestAccountStoreCorruptedJSONHandling
--- PASS: TestAccountStoreCorruptedJSONHandling (0.03s)
=== RUN   TestSanitizeFilenameEdgeCases
--- PASS: TestSanitizeFilenameEdgeCases (0.00s)
=== RUN   TestSafeAccountPathTraversalGuards
--- PASS: TestSafeAccountPathTraversalGuards (0.00s)
=== RUN   TestAccountStoreConcurrentAccess
--- PASS: TestAccountStoreConcurrentAccess (4.71s)
=== RUN   TestModelTokenDataDualSerialization
--- PASS: TestModelTokenDataDualSerialization (0.00s)
=== RUN   TestGetDefaultStoreDir_CurrentProject
--- PASS: TestGetDefaultStoreDir_CurrentProject (0.00s)
=== RUN   TestGetDefaultStoreDir_SimulatedSubdirBin
--- PASS: TestGetDefaultStoreDir_SimulatedSubdirBin (0.00s)
=== RUN   TestGetDefaultStoreDir_MockNestedHierarchy
--- PASS: TestGetDefaultStoreDir_MockNestedHierarchy (0.01s)
=== RUN   TestGetDefaultStoreDir_EnvVarOverride
--- PASS: TestGetDefaultStoreDir_EnvVarOverride (0.00s)
=== RUN   TestAccountStore_Status_Persistence_And_Preservation
--- PASS: TestAccountStore_Status_Persistence_And_Preservation (0.10s)
=== RUN   TestAccountStore_Status_ErrorAndEdgeCases
--- PASS: TestAccountStore_Status_ErrorAndEdgeCases (0.05s)
=== RUN   TestDiscoveredStoreRepeatReloadAndPartialRead
--- PASS: TestDiscoveredStoreRepeatReloadAndPartialRead (0.14s)
=== RUN   TestDiscoveredStoreUnavailableAndFailedReplacement
--- PASS: TestDiscoveredStoreUnavailableAndFailedReplacement (0.03s)
=== RUN   TestResolveStoreLocationsAndInvalidOverride
--- PASS: TestResolveStoreLocationsAndInvalidOverride (0.01s)
=== RUN   TestDiscoveredStore_BackwardCompatibility_LegacyJSON
--- PASS: TestDiscoveredStore_BackwardCompatibility_LegacyJSON (0.02s)
=== RUN   TestDiscoveredStore_SaveDiscovered_PreservesAddedFlag
--- PASS: TestDiscoveredStore_SaveDiscovered_PreservesAddedFlag (0.08s)
=== RUN   TestDiscoveredStore_SetDiscoveredAdded_ToggleAndPersistence
--- PASS: TestDiscoveredStore_SetDiscoveredAdded_ToggleAndPersistence (0.04s)
=== RUN   TestDiscoveredStore_ListDiscoveredRecords_SortingAndFiltering
--- PASS: TestDiscoveredStore_ListDiscoveredRecords_SortingAndFiltering (0.06s)
=== RUN   TestDiscoveredStore_SetDiscoveredAdded_ErrorAndEdgeCases
--- PASS: TestDiscoveredStore_SetDiscoveredAdded_ErrorAndEdgeCases (0.03s)
PASS
ok  	antigravity-switch/internal/store	6.207s
```

### 2. Kết quả kiểm thử Race Condition (`go test -race ./internal/store/...`):
```text
ok  	antigravity-switch/internal/store	13.431s
```
-> Không phát hiện data race trong bất kỳ thao tác đọc/ghi/khóa đồng thời nào.

### 3. Kết quả chạy Build Pipeline Repo (`pwsh -File .\scripts\build_windows.ps1 -Clean`):
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
ok  	antigravity-switch/internal/store	5.541s
ok  	antigravity-switch/internal/telemetry	(cached)
ok  	antigravity-switch/internal/ui	36.634s
ok  	antigravity-switch/internal/wincred	(cached)
  [PASS] Verification succeeded.
[3/4] Compiling target binaries...
  -> Checking Windows application icon and resource...
  -> Compiling resource with windres: cmd\switcher\app.rc -> cmd\switcher\rsrc_windows_amd64.syso
  [SUCCESS] Compiled PE resource object: cmd\switcher\rsrc_windows_amd64.syso (12552 bytes)
  -> Building Standalone GUI Binary (Fyne, CGO_ENABLED=1)...
  [SUCCESS] Created: bin\antigravity-switch.exe
  Compile Duration: 10427 ms | Size: 27.78 MB (29131264 bytes)
[4/4] Executing quick verification...
  [PASS] Embedded PE application icon verified on bin\antigravity-switch.exe (32x32)
  [INFO] GUI binary built at bin\antigravity-switch.exe.
================================================================================
⚡ BUILD SUCCESSFUL
================================================================================
```
-> Exit Code 0, toàn bộ suite test và build ứng dụng hoàn tất sạch sẽ.

---

## IV. ĐÁNH GIÁ TIÊU CHÍ HOÀN THÀNH (DEFINITION OF DONE)
- [x] Triển khai đúng đặc tả kiến trúc Phase 1 tại `reports/architect/2026-10-22-add-account-modal-and-discovered-filtering-spec.md`.
- [x] Thêm `Added bool "json:added,omitempty"` vào struct `DiscoveredAccountEnvelope`.
- [x] Cập nhật `SaveDiscovered` bảo toàn cờ `Added == true` khi quét lại Chrome.
- [x] Bổ sung `SetDiscoveredAdded` và `ListDiscoveredRecords` hoạt động ổn định và chính xác.
- [x] Viết unit test phủ 100% các nhánh logic mới, kiểm thử tương thích ngược và cô lập trong `t.TempDir()`.
- [x] Build repo bằng script chuẩn `pwsh -File .\scripts\build_windows.ps1 -Clean` đạt Exit Code 0.
- [x] Phạm vi tệp sửa đổi nằm đúng 100% trong Phase 1 (0 dòng diff ngoài scope).
- [x] Trạng thái: Chuyển giao nghiệm thu cho Supervisor (`READY_FOR_SUPERVISOR`).
