# Báo Cáo Thi Công: Tự Động Dọn Dẹp Lịch Sử Active Threads

- **Ngày thi công:** 2026-10-21
- **Vai trò thực hiện:** Coder
- **Thẩm quyền kiến trúc:** `reports/architect/2026-10-21-active-threads-history-pruning-spec.md` (Mã văn kiện: `SPEC-ARCH-20261021-ACTIVE-THREADS-HISTORY-PRUNING`)
- **Git Commit Baseline:** `5d6ef05` (docs(system-architect): record lane completion report for active threads history pruning)
- **Trạng thái:** HOÀN THÀNH 100% TIÊU CHUẨN NGHIỆM THU (READY FOR SUPERVISOR / ARCHITECT)

---

## 1. Bản Đồ Họ Bất Biến (Invariant Family Map)

- **Tên họ bất biến:** Active Threads History Retention & Auto-Pruning (`invariant-active-threads-history-pruning`)
- **Nguồn thẩm quyền (SSOT):** `SPEC-ARCH-20261021-ACTIVE-THREADS-HISTORY-PRUNING`
- **Các bề mặt thực thi liên quan (Sibling Runtime Surfaces):**
  - `internal/reloader/active_threads.go`: Khai báo hằng số `DefaultHistoryKeepCount = 2` và hàm `PruneActiveThreadsHistory(historyDir string, keepCount int) (int, error)`.
  - `internal/reloader/session_reloader.go`: Bổ sung method wrapper `(r *SessionReloader) PruneActiveThreadsHistory(keepCount int) (int, error)`.
  - `internal/manager/account_manager.go`: Tích hợp BƯỚC 5 vào cuối luồng `SwitchAccount` (ngay sau Bước 4 session continuation và trước khi trả về kết quả thành công).
  - `internal/reloader/active_threads_test.go`: Bổ sung 4 unit test toàn diện kiểm thử cắt tỉa snapshot (`TestPruneActiveThreadsHistory_UnderLimit`, `TestPruneActiveThreadsHistory_OverLimit`, `TestPruneActiveThreadsHistory_NonExistentDir`, `TestSessionReloader_PruneActiveThreadsHistory`).
  - `internal/manager/manager_test.go`: Bổ sung integration test `TestSwitchAccount_PrunesHistoryToMax2` kiểm thử toàn trình `SwitchAccount` tự động dọn dẹp thư mục lịch sử.
- **Đường dẫn cũ bị loại bỏ (Forbidden Legacy Fallback):**
  - Không cho phép tích tụ snapshot vô hạn trong thư mục `accounts_store\active_threads_history\`.
  - Không gọi dọn dẹp tại critical path (giữa Bước 1 và Bước 2) để tránh xung đột I/O.
  - Tuyệt đối không cho phép lỗi dọn dẹp (nếu có) làm fail hoặc gián đoạn kết quả thành công của `SwitchAccount`.
- **Ma trận xác minh:**
  - `TestPruneActiveThreadsHistory_UnderLimit`: [PASS] 0, 1, 2 file snapshot không bị xóa (`deletedCount = 0`).
  - `TestPruneActiveThreadsHistory_OverLimit`: [PASS] 10 file snapshot thì đúng 8 file cũ bị xóa, đúng 2 file mới nhất giữ lại, file rác và subfolder không liên quan được bảo toàn.
  - `TestPruneActiveThreadsHistory_NonExistentDir`: [PASS] Thư mục không tồn tại trả về `0, nil` an toàn.
  - `TestSessionReloader_PruneActiveThreadsHistory`: [PASS] Method wrapper hoạt động chính xác với `OutputDir` và tự động default `keepCount = 2` khi `keepCount <= 0`.
  - `TestSwitchAccount_PrunesHistoryToMax2`: [PASS] Tích hợp thực tế trong `SwitchAccount` cắt tỉa thư mục lịch sử về tối đa 2 file sau khi chuyển tài khoản thành công.
  - `go test ./...`: [PASS] Toàn bộ test suite dự án vượt qua 100%.
  - `pwsh -File .\scripts\build_windows.ps1 -Clean`: [PASS] Biên dịch thành công binary chuẩn `bin\antigravity-switch.exe` (Exit Code 0).

---

## 2. Chi Tiết Các Tệp Thay Đổi (Files Changed)

### 2.1 `internal/reloader/active_threads.go`
- Bổ sung hằng số `DefaultHistoryKeepCount = 2`.
- Triển khai hàm `PruneActiveThreadsHistory(historyDir string, keepCount int) (int, error)`:
  - Tự động chuẩn hóa `keepCount <= 0` thành `DefaultHistoryKeepCount` (2).
  - Tự động fallback sang `GetDefaultThreadsHistoryDir()` nếu `historyDir == ""`.
  - Kiểm tra `os.IsNotExist(err)` trả về `0, nil` fail-safe.
  - Lọc chính xác các file thông thường có đuôi `_active_threads.json` (bỏ qua directory và file rác).
  - Sắp xếp giảm dần theo tên file (`sort.Slice` so sánh `>`): định dạng `YYYY-MM-DD-HH-mm-ss_active_threads.json` bảo đảm sắp xếp chuỗi giảm dần trùng khớp tuyệt đối với thứ tự thời gian mới nhất trước.
  - Giữ lại 2 file đầu tiên, xóa các file từ index 2 trở đi bằng `os.Remove`. Bỏ qua lỗi từng file riêng lẻ để dọn sạch tối đa.

### 2.2 `internal/reloader/session_reloader.go`
- Bổ sung method `(r *SessionReloader) PruneActiveThreadsHistory(keepCount int) (int, error)`:
  - Đọc an toàn `r.OutputDir` bằng `r.mu.RLock()`.
  - Chuyển tiếp tới hàm xử lý cốt lõi `PruneActiveThreadsHistory(dir, keepCount)`.

### 2.3 `internal/manager/account_manager.go`
- Bổ sung BƯỚC 5 tại cuối hàm `SwitchAccount`:
  ```go
  // =========================================================================
  // BƯỚC 5: Tự động dọn dẹp lịch sử snapshot (Chỉ giữ đúng 2 file mới nhất)
  // =========================================================================
  historyDir := ""
  if rel != nil && rel.OutputDir != "" {
      historyDir = rel.OutputDir
  } else if m.store != nil {
      historyDir = filepath.Join(m.store.GetStoreDir(), "active_threads_history")
  } else {
      historyDir = reloader.GetDefaultThreadsHistoryDir()
  }
  _, _ = reloader.PruneActiveThreadsHistory(historyDir, reloader.DefaultHistoryKeepCount)
  ```
- Hoàn toàn non-blocking đối với luồng chuyển tài khoản (fail-safe).

### 2.4 `internal/reloader/active_threads_test.go`
- Thêm 4 unit test bao phủ các nhánh:
  - `TestPruneActiveThreadsHistory_UnderLimit`
  - `TestPruneActiveThreadsHistory_OverLimit`
  - `TestPruneActiveThreadsHistory_NonExistentDir`
  - `TestSessionReloader_PruneActiveThreadsHistory`

### 2.5 `internal/manager/manager_test.go`
- Thêm integration test `TestSwitchAccount_PrunesHistoryToMax2` kiểm thử `SwitchAccount` tự động dọn dẹp 5 snapshot cũ xuống còn đúng 2 snapshot sau khi switch.

---

## 3. Bằng Chứng Thực Thi & Xác Minh (Evidence & Benchmarks)

### 3.1 Unit Test Package `internal/reloader`
```text
=== RUN   TestPruneActiveThreadsHistory_UnderLimit
--- PASS: TestPruneActiveThreadsHistory_UnderLimit (0.01s)
=== RUN   TestPruneActiveThreadsHistory_OverLimit
--- PASS: TestPruneActiveThreadsHistory_OverLimit (0.02s)
=== RUN   TestPruneActiveThreadsHistory_NonExistentDir
--- PASS: TestPruneActiveThreadsHistory_NonExistentDir (0.00s)
=== RUN   TestSessionReloader_PruneActiveThreadsHistory
--- PASS: TestSessionReloader_PruneActiveThreadsHistory (0.00s)
PASS
ok  	antigravity-switch/internal/reloader	1.995s
```

### 3.2 Integration Test Package `internal/manager`
```text
=== RUN   TestSwitchAccount_PrunesHistoryToMax2
--- PASS: TestSwitchAccount_PrunesHistoryToMax2 (0.05s)
PASS
ok  	antigravity-switch/internal/manager	2.114s
```

### 3.3 Toàn Bộ Kiểm Thử Dự Án (`go test ./...`)
```text
ok  	antigravity-switch/cmd/switcher	2.521s
ok  	antigravity-switch/internal/automation	15.757s
ok  	antigravity-switch/internal/connector	(cached)
ok  	antigravity-switch/internal/manager	4.821s
ok  	antigravity-switch/internal/model	(cached)
ok  	antigravity-switch/internal/reloader	2.995s
ok  	antigravity-switch/internal/scanner	(cached)
ok  	antigravity-switch/internal/store	6.951s
ok  	antigravity-switch/internal/telemetry	(cached)
ok  	antigravity-switch/internal/ui	19.782s
ok  	antigravity-switch/internal/wincred	(cached)
?   	antigravity-switch/reports/supervisor/evidence/autonomous_switch_runtime_review_261020	[no test files]
```

### 3.4 Kịch Bản Build Chuẩn (`pwsh -File .\scripts\build_windows.ps1 -Clean`)
```text
================================================================================
⚡ Antigravity Multi-Account Switcher - Windows Build System
   OutDir: bin | SkipTests: False | Clean: True
================================================================================
[1/4] Cleaning output artifacts...
  Removed directory: bin
  Clean complete.
[2/4] Running pre-flight tests & vet...
ok  	antigravity-switch/cmd/switcher	1.896s
ok  	antigravity-switch/internal/automation	(cached)
ok  	antigravity-switch/internal/connector	(cached)
ok  	antigravity-switch/internal/manager	(cached)
ok  	antigravity-switch/internal/model	(cached)
ok  	antigravity-switch/internal/reloader	(cached)
ok  	antigravity-switch/internal/scanner	(cached)
ok  	antigravity-switch/internal/store	4.882s
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
  Compile Duration: 5986 ms | Size: 27.73 MB (29073408 bytes)
[4/4] Executing quick verification...
  [PASS] Embedded PE application icon verified on bin\antigravity-switch.exe (32x32)
  [INFO] GUI binary built at bin\antigravity-switch.exe.
================================================================================
⚡ BUILD SUCCESSFUL
================================================================================
```

---

## 4. Kết Luận & Bàn Giao

- Tính năng tự động cắt tỉa snapshot active threads history đã được thi công và kiểm chứng hoàn chỉnh theo đúng 100% đặc tả tại `reports/architect/2026-10-21-active-threads-history-pruning-spec.md`.
- Sẵn sàng bàn giao cho Supervisor và QA nghiệm thu.
