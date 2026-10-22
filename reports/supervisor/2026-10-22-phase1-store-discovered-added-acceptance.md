# Báo Cáo Nghiệm Thu Độc Lập (Zero-Trust): Phase 1 - Tầng Lưu Trữ Danh Bạ (Discovered Store)
(Independent Zero-Trust Acceptance Report: Discovered Store Added Flag Persistence & Queries - Phase 1)

Verdict: PASS

## Metadata
- Report file: `reports/supervisor/2026-10-22-phase1-store-discovered-added-acceptance.md`
- Review time: 2026-10-22 18:03:00 +07:00
- Reviewer: Supervisor (Zero-Trust Independent Auditor)
- Repo/project: Antigravity Multi-Account Switcher (`antigravity-switch`)
- Scope reviewed: Phase 1 (Tầng Lưu trữ Danh bạ - Discovered Store)
- Commit reviewed: `15a0879803493b2d894d7be18e2c4e29c7c97dfc`
- Spec base commit: `54617e3`
- Authority used: 
  - `reports/architect/2026-10-22-add-account-modal-and-discovered-filtering-spec.md` (commit `54617e3`)
  - `reports/coder/2026-10-22-phase1-store-discovered-added-report.md`
  - Supervisor Skill (`g:\antigravity_auto_switch\.agents\skills\supervisor\SKILL.md`)
  - Quy tắc ranh giới bất biến & kỷ luật Zero-Trust repo
- Related artifacts:
  - Binary verified: `bin/antigravity-switch.exe` (27.78 MB, PE icon verified)

---

## Executive Summary
- **Vấn đề / Mục tiêu:** Thực hiện kiểm định độc lập Zero-Trust đối với việc triển khai BƯỚC 1 (PHASE 1) - Tầng Lưu trữ Danh bạ (Discovered Store) nhằm hỗ trợ cờ `Added` bền vững, chuẩn bị cho việc lọc thẻ danh bạ và modal thêm tài khoản theo đúng Đặc tả Kiến trúc tại commit `54617e3`.
- **Quyết định (Verdict):** **PASS**
- **Lý do quyết định:** Toàn bộ 5 hạng mục yêu cầu kỹ thuật của Phase 1 đã được kiểm chứng đối soát trực tiếp trên mã nguồn thực tế tại commit `15a0879`, tất cả test suite (`go test -v`, `go test -race`) và quy trình build chuẩn của repo (`pwsh -File .\scripts\build_windows.ps1 -Clean`) đều vượt qua 100% với Exit Code 0, ranh giới file được bảo toàn tuyệt đối (0 lines diff ngoài scope).
- **Hành động tiếp theo:** Chuyển giao nghiệm thu cho CEO để kích hoạt Phase 2 (Tách rời Account Manager).

---

## Source-Level Clearance Notes

### 1. `internal/store/discovered_store.go`: CLEAR
- **Trường `Added` trong struct `DiscoveredAccountEnvelope` (Dòng 16-20):**
  - Khai báo: `Added bool json:"added,omitempty"`.
  - Đối soát: Tuân thủ chính xác đặc tả dòng 196-200 của tài liệu kiến trúc. Tag `omitempty` đảm bảo khi unmarshal tệp JSON cũ không có trường `added`, giá trị mặc định trong Go sẽ là `false`, không gây lỗi định dạng và bảo toàn tương thích ngược 100%.
- **Bảo toàn cờ `Added` trong `SaveDiscovered(email string)` (Dòng 41-85):**
  - Chuẩn hóa email với `strings.ToLower(strings.TrimSpace(email))`.
  - Kiểm tra an toàn path traversal bằng `SafeAccountPath(s.dir, email)`.
  - Đoạn mã dòng 53-60 đọc tệp hiện có trên đĩa trước khi lưu; nếu tìm thấy bản ghi cũ hợp lệ, cờ `added` được trích xuất và bảo toàn nguyên vẹn (`added = existingEnv.Added`).
  - Ghi tệp nguyên tử bằng `os.CreateTemp` + `tmp.Sync()` + `tmp.Close()` + `os.Rename`. Trong trường hợp rename thất bại, bản ghi cũ không bị xóa hoặc hỏng.
- **Hàm `SetDiscoveredAdded(email string, added bool)` (Dòng 87-137):**
  - Kiểm tra tính hợp lệ của email: nếu rỗng sau khi trim trả về `ErrInvalidEmail` chuẩn hóa.
  - Kiểm tra path traversal an toàn.
  - Đoạn mã dòng 105-113 kiểm tra tệp cũ: nếu đã có trên đĩa và có `ScannedAt > 0`, giữ nguyên mốc thời gian quét gốc `existingEnv.ScannedAt`, chỉ cập nhật cờ `Added`. Nếu bản ghi mới, khởi tạo `scannedAt = float64(time.Now().UnixNano()) / 1e9`.
  - Cơ chế ghi tệp nguyên tử tương tự, thread-safe dưới khóa `s.mu.Lock()`.
- **Hàm `ListDiscoveredRecords() ([]DiscoveredAccountEnvelope, error)` (Dòng 174-215):**
  - Thực thi thread-safe dưới `s.mu.RLock()`.
  - Xử lý trường hợp thư mục chưa tồn tại bằng `os.IsNotExist(err)` -> trả về danh sách rỗng và nil error, không gây panic hay lỗi giả mạo.
  - Bỏ qua các thư mục con và tệp không có đuôi `.json`.
  - Kiểm tra tính toàn vẹn bản ghi: kiểm tra lỗi unmarshal JSON, email không rỗng, tên tệp khớp với `SanitizeFilename(email)`, và `ScannedAt > 0`. Các tệp lỗi được tích lũy vào danh sách `failures` qua `errors.Join`.
  - Khử trùng lặp email bằng bản đồ `seen[email]`.
  - Sắp xếp kết quả trả về theo thứ tự bảng chữ cái A-Z (`records[i].Email < records[j].Email`).
- **Hàm `ListDiscovered() ([]string, error)` (Dòng 140-172):**
  - Giữ nguyên vẹn 100% chữ ký và hành vi trả về `[]string` để đảm bảo tương thích ngược tuyệt đối với các module hiện hữu.

### 2. `internal/store/discovered_store_test.go`: CLEAR
- Toàn bộ 5 test case mới được cô lập 100% trong `t.TempDir()`, không tạo file rác trong workspace:
  - `TestDiscoveredStore_BackwardCompatibility_LegacyJSON`: Xác minh file JSON không có trường `added` được unmarshal ra `Added == false`.
  - `TestDiscoveredStore_SaveDiscovered_PreservesAddedFlag`: Xác minh sau khi gán `Added == true`, việc gọi lại `SaveDiscovered` (mô phỏng Chrome re-scan) vẫn giữ nguyên `Added == true`.
  - `TestDiscoveredStore_SetDiscoveredAdded_ToggleAndPersistence`: Xác minh việc bật/tắt cờ `Added`, kiểm tra format JSON trên đĩa và đọc lại sau khi khởi tạo lại store (`NewDiscoveredStore`).
  - `TestDiscoveredStore_ListDiscoveredRecords_SortingAndFiltering`: Xác minh khả năng lọc bỏ folder, non-json, file corrupt, filename mismatch, timestamp 0, và sắp xếp thứ tự A-Z.
  - `TestDiscoveredStore_SetDiscoveredAdded_ErrorAndEdgeCases`: Xác minh bắt lỗi `ErrInvalidEmail` với chuỗi rỗng/khoảng trắng, chuẩn hóa email hoa/thường, và xử lý I/O error khi thư mục bị chiếm dụng.

### 3. Verification of Boundary Isolation: CLEAR
- Lệnh kiểm tra: `git diff 54617e3..15a0879803493b2d894d7be18e2c4e29c7c97dfc`
- Chỉ có 3 tệp thay đổi:
  - `internal/store/discovered_store.go` (+113, -1)
  - `internal/store/discovered_store_test.go` (+304, -0)
  - `reports/coder/2026-10-22-phase1-store-discovered-added-report.md` (+176, -0)
- Các tệp nhạy cảm ngoài scope:
  - `scripts/build_windows.ps1`: 0 lines diff (giữ nguyên 100%)
  - `README.md`: 0 lines diff (giữ nguyên 100%)
  - `internal/manager/*`: 0 lines diff (chưa can thiệp trước Phase 2)
  - `internal/ui/*`: 0 lines diff (chưa can thiệp trước Phase 3, 4)

---

## Evidence Checked

### Passed:
1. **Unit Test Suite của gói Store:**
   - Command: `go test -v ./internal/store/...`
   - Freshness: Fresh (thực thi độc lập bởi Supervisor).
   - Kết quả: 22/22 unit tests PASS (Thời gian chạy: 5.327s).
2. **Kiểm tra Race Condition:**
   - Command: `go test -race ./internal/store/...`
   - Freshness: Fresh (thực thi độc lập bởi Supervisor).
   - Kết quả: PASS (14.081s, không phát hiện data race trong bất kỳ thao tác RWMutex nào).
3. **Kiểm tra Toàn Bộ Test Suite Repo:**
   - Command: `go test ./...` (được chạy tự động trong build script).
   - Freshness: Fresh.
   - Kết quả: PASS 100% trên toàn bộ các package (`cmd/switcher`, `internal/automation`, `internal/connector`, `internal/manager`, `internal/model`, `internal/reloader`, `internal/scanner`, `internal/store`, `internal/telemetry`, `internal/ui`, `internal/wincred`).
4. **Kiểm thử Pipeline Biên Dịch Repo Chính Thức:**
   - Command: `pwsh -File .\scripts\build_windows.ps1 -Clean`
   - Freshness: Fresh (thực thi độc lập bởi Supervisor).
   - Kết quả: Exit Code 0, biên dịch PE resource sạch sẽ, tạo thành công file thực thi `bin\antigravity-switch.exe` (27.78 MB, kích thước 29,131,264 bytes, icon 32x32 verified).
5. **Đối soát Thẩm quyền Kiến trúc (SSOT Alignment):**
   - Tài liệu: `reports/architect/2026-10-22-add-account-modal-and-discovered-filtering-spec.md` (commit `54617e3`), Phần IV.1.
   - Kết quả: Mã nguồn thực tế khớp 100% từng dòng lệnh và logic atomic temp file + sync + rename được đặc tả.

### Failed:
- Không có (None).

### Not run:
- Các bài kiểm thử UI tổng hợp (TC-01 đến TC-08) thuộc phạm vi kiểm định end-to-end của các Phase sau (Phase 2 -> Phase 5), sẽ được kiểm chứng khi UI và Manager hoàn tất tích hợp.

---

## Invariant Closure
- **Affected Invariants:**
  - `Invariant-Discovered-Added-Persistence`: Đã đóng và được kiểm chứng qua tệp tin trên đĩa.
  - `Invariant-Backward-Compatibility-Default-Unadded`: Đã đóng qua struct tag `omitempty` và unit test `TestDiscoveredStore_BackwardCompatibility_LegacyJSON`.
  - `Invariant-Scan-Added-Preservation`: Đã đóng qua logic đọc bảo tồn cờ `Added` trong `SaveDiscovered` và test case `TestDiscoveredStore_SaveDiscovered_PreservesAddedFlag`.
  - `Invariant-ScannedAt-Preservation-On-Add`: Đã đóng qua logic giữ nguyên `ScannedAt` trong `SetDiscoveredAdded`.
  - `Invariant-Sorted-Discovered-Records`: Đã đóng qua thuật toán sort và filter trong `ListDiscoveredRecords`.
- **Sibling surfaces checked:**
  - `internal/store/discovered_store.go`
  - `internal/store/discovered_store_test.go`
- **Residual unverified same-invariant surfaces:**
  - `none` (Toàn bộ các bề mặt của Tầng Store Phase 1 đã được nghiệm thu đầy đủ).

---

## Overall Evaluation
Việc triển khai Phase 1 của Coder đạt chất lượng xuất sắc:
1. Bám sát 100% bản đặc tả kiến trúc tại commit `54617e3`.
2. Kỷ luật ranh giới nghiêm ngặt: không sửa đổi bất kỳ tệp nào ngoài phạm vi Phase 1, giữ nguyên build script của repo.
3. Bộ test case bổ sung phong phú, bao quát toàn diện các trường hợp tương thích ngược, tranh chấp, file rác/hỏng, và các edge case về chuỗi rỗng/khoảng trắng.
4. Đạt chuẩn Zero-Trust: toàn bộ kiểm tra độc lập (`go test`, `go test -race`, `build_windows.ps1 -Clean`) đều vượt qua tuyệt đối.

**Cấp chứng chỉ nghiệm thu: PASS cho PHASE 1.**
Sẵn sàng tiến sang **PHASE 2: Tách rời Account Manager (account_manager.go)**.
