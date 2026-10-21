# Báo Cáo Thi Công: Khởi Tạo Module Timeline & Chuẩn Hóa Phân Mảnh Nhật Ký Chuyển Tài Khoản (Phase 1)

- **Ngày thi công:** 2026-10-21
- **Vai trò thực hiện:** Coder
- **Thẩm quyền kiến trúc:** `reports/architect/2026-10-21-daily-switch-timeline-sharding-and-retention-spec.md` (Mã văn kiện: `SPEC-ARCH-20261021-DAILY-SWITCH-TIMELINE-SPEC`, Commit: `d57c6e2`)
- **Trạng thái:** HOÀN THÀNH 100% TIÊU CHUẨN NGHIỆM THU PHASE 1 (PHASE 1 COMPLETE - READY FOR SUPERVISOR)

---

## 1. Bản Đồ Họ Bất Biến (Invariant Family Map)

- **Tên họ bất biến:** Daily Switch Timeline Sharding & 7-Day Pruning (`invariant-daily-switch-timeline-phase1`)
- **Nguồn thẩm quyền (SSOT):** `SPEC-ARCH-20261021-DAILY-SWITCH-TIMELINE-SPEC` Mục I (Invariants), Mục II (Layout), Mục III (Bước 1 & Bước 2), Mục IV (Test Matrix).
- **Các bề mặt thực thi liên quan (Sibling Runtime Surfaces):**
  - **Cấu trúc thư mục nhật ký đĩa (`logs/switch_timeline/`):**
    - Di dời tệp mồ côi `logs/switch_timeline.log` vào `logs/switch_timeline/2026-10-17_switch_timeline.log`.
    - Đảm bảo gốc `logs/` chỉ chứa các thư mục con phân loại (`quota_log/`, `switch_timeline/`, `active_threads_history/`) và tệp cấu hình giao diện `ui_preferences.json`.
  - **Module nghiệp vụ timeline độc lập (`internal/manager/timeline.go`):**
    - `DefaultSwitchTimelineDir = "logs/switch_timeline"`.
    - `DefaultSwitchTimelineRetentionDays = 7`.
    - `GetTodaySwitchTimelinePath(baseDir ...string) string`: Tạo đường dẫn chuẩn theo ngày `YYYY-MM-DD_switch_timeline.log`.
    - `LogSwitchEvent(timelineDir string, eventTag string, message string, elapsed ...time.Duration)`: Non-blocking, fail-safe ghi log sự kiện kèm timestamp `15:04:05.000` và thời gian thực thi `[+ 1234ms]`.
    - `PruneDailySwitchTimelineLogs(timelineDir string, keepCount int) (int, error)`: Quét thư mục, lọc hậu tố `_switch_timeline.log`, sắp xếp giảm dần theo tên (mới nhất đứng đầu), giữ lại tối đa `keepCount` (mặc định 7) và xóa các tệp quá hạn một cách an toàn.
  - **Bộ kiểm thử đơn vị (`internal/manager/timeline_test.go`):**
    - `TestPruneDailySwitchTimelineLogs_UnderLimit`: Kiểm thử khi số lượng tệp <= 7 (không tệp nào bị xóa).
    - `TestPruneDailySwitchTimelineLogs_OverLimit`: Kiểm thử khi có 10 tệp (xóa đúng 3 tệp cũ nhất `2026-10-01` -> `2026-10-03`, giữ nguyên 7 tệp mới nhất).
    - `TestLogSwitchEvent_CreatesDailyFile`: Kiểm thử tạo tệp ngày hôm nay và ghi đúng format `[TEST_EVENT]`, elapsed, message.
    - `TestPruneDailySwitchTimelineLogs_DefaultRetentionAndUnrelatedFiles`: Kiểm thử fallback khi `keepCount <= 0`, bảo vệ tệp không liên quan và thư mục con.
    - `TestPruneDailySwitchTimelineLogs_NonExistentDir`: Kiểm thử thư mục không tồn tại trả về `(0, nil)`.
    - `TestGetTodaySwitchTimelinePath`: Kiểm thử tạo đường dẫn tùy biến và đường dẫn mặc định.
- **Đường dẫn/Cơ chế cũ bị loại bỏ (Forbidden Legacy Fallback):**
  - Không còn tệp log mồ côi `logs/switch_timeline.log` nằm ở gốc thư mục `logs/`.
  - Không cho phép tích tụ log timeline vô hạn ngày mà không có cơ chế cắt tỉa.
- **Phân định ranh giới (Boundary Isolation & Non-Goals):**
  - Giữ nguyên 100% `internal/manager/account_manager.go` và `internal/ui/app.go` trong phiên này (sẽ tích hợp tại Phase 2 sau khi Phase 1 được Supervisor nghiệm thu).
  - Giữ nguyên 100% `scripts/build_windows.ps1` (0 lines diff).
  - Không sử dụng anvien, không kích hoạt subagent.
- **Residual unverified surfaces:** `none`.

---

## 2. Chi Tiết Thực Thi (Implementation Details)

### 2.1 Di dời tệp log trên đĩa
Đã di dời thành công:
```powershell
New-Item -ItemType Directory -Force -Path "logs\switch_timeline"
Move-Item -Path "logs\switch_timeline.log" -Destination "logs\switch_timeline\2026-10-17_switch_timeline.log"
```

Hiện trạng `logs/`:
```text
logs/
├── active_threads_history/
├── quota_log/
├── switch_timeline/
│   └── 2026-10-17_switch_timeline.log
└── ui_preferences.json
```
Không còn bất kỳ tệp log mồ côi nào trôi nổi ở ngoài gốc `logs/`.

### 2.2 Tạo mới `internal/manager/timeline.go`
Mã nguồn cài đặt đầy đủ theo đặc tả kiến trúc:
- Hằng số cấu hình:
  - `DefaultSwitchTimelineDir = "logs/switch_timeline"`
  - `DefaultSwitchTimelineRetentionDays = 7`
- Các hàm tác vụ:
  - `GetTodaySwitchTimelinePath(baseDir ...string) string`
  - `LogSwitchEvent(timelineDir string, eventTag string, message string, elapsed ...time.Duration)`
  - `PruneDailySwitchTimelineLogs(timelineDir string, keepCount int) (int, error)`

### 2.3 Tạo mới `internal/manager/timeline_test.go`
Bao phủ 100% kịch bản trong `t.TempDir()` độc lập:
- `TestPruneDailySwitchTimelineLogs_UnderLimit`
- `TestPruneDailySwitchTimelineLogs_OverLimit`
- `TestLogSwitchEvent_CreatesDailyFile`
- `TestPruneDailySwitchTimelineLogs_DefaultRetentionAndUnrelatedFiles`
- `TestPruneDailySwitchTimelineLogs_NonExistentDir`
- `TestGetTodaySwitchTimelinePath`

---

## 3. Kết Quả Xác Minh (Verification Evidence)

### 3.1 Kiểm thử đơn vị cô lập (`internal/manager`)
Lệnh thực thi:
```powershell
go test -v ./internal/manager
```
Kết quả:
```text
=== RUN   TestExtractEmailFromToken
--- PASS: TestExtractEmailFromToken (0.00s)
=== RUN   TestCaptureCurrentSession
--- PASS: TestCaptureCurrentSession (0.02s)
=== RUN   TestGetCurrentActiveEmail
--- PASS: TestGetCurrentActiveEmail (0.01s)
=== RUN   TestGetAccountStates
--- PASS: TestGetAccountStates (0.23s)
=== RUN   TestSwitchAccount
--- PASS: TestSwitchAccount (0.34s)
=== RUN   TestSwitchAccount_WithSessionContinuation
--- PASS: TestSwitchAccount_WithSessionContinuation (0.04s)
=== RUN   TestSwitchAccount_ProactiveTokenRefresh
--- PASS: TestSwitchAccount_ProactiveTokenRefresh (0.26s)
=== RUN   TestSwitchAccount_AtomicAppStorageSync
--- PASS: TestSwitchAccount_AtomicAppStorageSync (0.17s)
=== RUN   TestSwitchAccount_CivilizedReAuth_HotSwapped
--- PASS: TestSwitchAccount_CivilizedReAuth_HotSwapped (0.07s)
=== RUN   TestSwitchAccount_ReAuthFallback_TriggerReauthError
--- PASS: TestSwitchAccount_ReAuthFallback_TriggerReauthError (0.07s)
=== RUN   TestSwitchAccount_ReAuthFallback_EmailMismatch
--- PASS: TestSwitchAccount_ReAuthFallback_EmailMismatch (0.07s)
=== RUN   TestSwitchAccount_ReAuthFallback_ClientNil
--- PASS: TestSwitchAccount_ReAuthFallback_ClientNil (0.07s)
=== RUN   TestSwitchAccount_8SecondWaitAnd4StepWorkflow
--- PASS: TestSwitchAccount_8SecondWaitAnd4StepWorkflow (0.05s)
=== RUN   TestSwitchAccount_F1_RAMConfirmation_Mismatch
--- PASS: TestSwitchAccount_F1_RAMConfirmation_Mismatch (0.03s)
=== RUN   TestSwitchAccount_F1_RAMConfirmation_GetUserStatusError
--- PASS: TestSwitchAccount_F1_RAMConfirmation_GetUserStatusError (0.04s)
=== RUN   TestSwitchAccount_F1_RAMConfirmation_ClientNil
--- PASS: TestSwitchAccount_F1_RAMConfirmation_ClientNil (0.05s)
=== RUN   TestSwitchAccount_F4_StoragePathIntegration
--- PASS: TestSwitchAccount_F4_StoragePathIntegration (0.06s)
=== RUN   TestSwitchAccount_F1_RAMConfirmation_ClientNilWithoutError
--- PASS: TestSwitchAccount_F1_RAMConfirmation_ClientNilWithoutError (0.05s)
=== RUN   TestSwitchAccount_PrunesHistoryToMax2
--- PASS: TestSwitchAccount_PrunesHistoryToMax2 (0.06s)
=== RUN   TestSwitchAccount_UsesLogsActiveThreads
--- PASS: TestSwitchAccount_UsesLogsActiveThreads (0.06s)
=== RUN   TestPruneDailySwitchTimelineLogs_UnderLimit
--- PASS: TestPruneDailySwitchTimelineLogs_UnderLimit (0.02s)
=== RUN   TestPruneDailySwitchTimelineLogs_OverLimit
--- PASS: TestPruneDailySwitchTimelineLogs_OverLimit (0.01s)
=== RUN   TestLogSwitchEvent_CreatesDailyFile
--- PASS: TestLogSwitchEvent_CreatesDailyFile (0.02s)
=== RUN   TestPruneDailySwitchTimelineLogs_DefaultRetentionAndUnrelatedFiles
--- PASS: TestPruneDailySwitchTimelineLogs_DefaultRetentionAndUnrelatedFiles (0.01s)
=== RUN   TestPruneDailySwitchTimelineLogs_NonExistentDir
--- PASS: TestPruneDailySwitchTimelineLogs_NonExistentDir (0.00s)
=== RUN   TestGetTodaySwitchTimelinePath
--- PASS: TestGetTodaySwitchTimelinePath (0.00s)
=== RUN   TestTwoStoreCaptureMergeRepeatAndRemember
--- PASS: TestTwoStoreCaptureMergeRepeatAndRemember (0.24s)
=== RUN   TestTwoStorePartialFailuresAndMissingSession
--- PASS: TestTwoStorePartialFailuresAndMissingSession (0.04s)
=== RUN   TestTwoStoreCaptureWriteFailureKeepsLiveIdentity
--- PASS: TestTwoStoreCaptureWriteFailureKeepsLiveIdentity (0.02s)
PASS
ok  	antigravity-switch/internal/manager	4.809s
```

### 3.2 Kiểm thử toàn bộ hệ thống (`go test ./...`)
Lệnh thực thi:
```powershell
go test ./...
```
Kết quả:
```text
ok  	antigravity-switch/cmd/switcher	2.302s
ok  	antigravity-switch/internal/automation	14.895s
ok  	antigravity-switch/internal/connector	(cached)
ok  	antigravity-switch/internal/manager	6.871s
ok  	antigravity-switch/internal/model	(cached)
ok  	antigravity-switch/internal/reloader	(cached)
ok  	antigravity-switch/internal/scanner	(cached)
ok  	antigravity-switch/internal/store	8.342s
ok  	antigravity-switch/internal/telemetry	(cached)
ok  	antigravity-switch/internal/ui	21.624s
ok  	antigravity-switch/internal/wincred	(cached)
?   	antigravity-switch/reports/supervisor/evidence/autonomous_switch_runtime_review_261020	[no test files]
```
Toàn bộ test suites của repo đạt PASS 100%.

### 3.3 Biên dịch kiểm chứng Windows sạch (`build_windows.ps1 -Clean`)
Lệnh thực thi:
```powershell
pwsh -File .\scripts\build_windows.ps1 -Clean
```
Kết quả:
- Exit Code: `0`
- Target binary: `bin\antigravity-switch.exe`
- Dung lượng binary: `27.73 MB (29081088 bytes)`
- Trạng thái icon PE: `[PASS] Embedded PE application icon verified on bin\antigravity-switch.exe (32x32)`
- Script build diff: `0 lines diff` (`scripts/build_windows.ps1` nguyên vẹn 100%).

---

## 4. Danh Mục Tệp Thay Đổi (Files Changed)

1. `logs/switch_timeline/2026-10-17_switch_timeline.log` (di chuyển từ `logs/switch_timeline.log` trên đĩa)
2. `internal/manager/timeline.go` (tạo mới)
3. `internal/manager/timeline_test.go` (tạo mới)
4. `reports/coder/2026-10-21-daily-switch-timeline-phase1-report.md` (tạo mới)

---

## 5. Kết Luận & Bàn Giao (Conclusion & Next Steps)

Phase 1 đã hoàn thành trọn vẹn mọi yêu cầu kỹ thuật và kỷ luật kiến trúc:
- File log mồ côi đã được thu gom vào thư mục con chuẩn hóa.
- Module `timeline.go` được khởi tạo độc lập, kiểm thử unit 100% trong `t.TempDir()`.
- Toàn bộ suite `go test ./...` và tiến trình build sạch `build_windows.ps1 -Clean` đạt Exit Code 0.
- Sẵn sàng bàn giao cho Supervisor thẩm định nghiệm thu Phase 1 trước khi tiến hành Phase 2.
