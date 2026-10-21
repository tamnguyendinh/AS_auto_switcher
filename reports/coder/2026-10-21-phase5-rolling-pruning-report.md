# Báo Cáo Thi Công: Tự Động Dọn Dẹp 7 Ngày Gần Nhất (Rolling 7-Day Pruning) (Phase 5)

- **Ngày thi công:** 2026-10-21
- **Vai trò thực hiện:** Coder
- **Thẩm quyền kiến trúc:** `reports/architect/2026-10-21-logs-directory-migration-and-renaming-spec.md` (Mã văn kiện: `SPEC-ARCH-20261021-LOGS-AND-DAILY-QUOTA-SPEC`, Mục II - Giai đoạn 5 & Mục III.1)
- **Git Commit Baseline:** `57f51a9` (docs(supervisor): record Phase 4 daily quota sharding acceptance report)
- **Trạng thái:** HOÀN THÀNH 100% TIÊU CHUẨN NGHIỆM THU PHASE 5 (PHASE 5 COMPLETE - READY FOR SUPERVISOR)

---

## 1. Bản Đồ Họ Bất Biến (Invariant Family Map)

- **Tên họ bất biến:** Rolling 7-Day Quota Log Pruning (`invariant-rolling-7day-pruning-phase5`)
- **Nguồn thẩm quyền (SSOT):** `SPEC-ARCH-20261021-LOGS-AND-DAILY-QUOTA-SPEC` Mục II - Giai đoạn 5 (Phase 5) & Mục III.1 (Test Matrix).
- **Các bề mặt thực thi liên quan (Sibling Runtime Surfaces):**
  - **Hằng số & Hàm dọn dẹp nhật ký quota (`internal/telemetry/cache.go`):**
    - Hằng số `DefaultQuotaLogRetentionDays = 7`.
    - Hàm `PruneDailyQuotaLogs(quotaLogDir string, keepCount int) (int, error)`:
      - Tự động fallback về `DefaultQuotaLogRetentionDays` (7) nếu `keepCount <= 0`.
      - Tự động fallback về `DefaultQuotaLogDir` (`logs/quota_log`) nếu `quotaLogDir` rỗng/whitespace.
      - Trả về `(0, nil)` nếu thư mục không tồn tại (`os.IsNotExist`).
      - Lọc chỉ các file có đuôi `_quota_log.json` (bỏ qua thư mục con và các file định dạng khác).
      - Sắp xếp giảm dần theo tên file (ngày mới nhất đứng đầu).
      - Xóa an toàn các file vượt quá `keepCount`, bỏ qua lỗi I/O nếu file đang bị khóa (fail-safe).
  - **Khởi động ứng dụng GUI (`internal/ui/app.go`):**
    - Trong `NewAppWithFyne`:
      - Kích hoạt goroutine nền nhẹ tự động gọi `telemetry.PruneDailyQuotaLogs(quotaDir, telemetry.DefaultQuotaLogRetentionDays)`.
      - Đảm bảo non-blocking hoàn toàn, không làm gián đoạn hay trễ UI khi khởi động.
  - **Bộ kiểm thử tự động (Automated Test Suite):**
    - `internal/telemetry/cache_test.go`: Bổ sung 4 unit tests mới bao phủ toàn bộ các kịch bản under limit, over limit, fallback keepCount, ignore unrelated files/dirs, và non-existent directory.
- **Đường dẫn/Cơ chế cũ bị loại bỏ (Forbidden Legacy Fallback):**
  - Không cho phép tích tụ file log quota vô hạn ngày trong thư mục `logs/quota_log/`.
  - Không chạy tiến trình dọn dẹp dạng blocking trên UI thread.
  - Không để lỗi I/O của file log làm crash ứng dụng.
- **Artifacts và Tests cũ đã cập nhật/dọn dẹp:**
  - `internal/telemetry/cache_test.go`: Bổ sung 4 test cases cho Phase 5, nâng tổng số test cases của suite lên 11 test cases pass 100%.
  - Toàn bộ test suite `internal/telemetry/...`, `internal/ui/...` và repo `go test ./...` pass 100%.
- **Ma trận xác minh (Verify Matrix):**
  - Primary path: `go test -v ./internal/telemetry/...` PASS 100%.
  - UI integration path: `go test -v ./internal/ui/...` PASS 100%.
  - Full repo regression: `go test ./...` PASS 100%.
  - Windows clean build verification: `pwsh -File .\scripts\build_windows.ps1 -Clean` Exit Code 0.
  - Script discipline: `scripts/build_windows.ps1` giữ nguyên 100% (0 lines diff).
- **Residual unverified surfaces:** `none`.

---

## 2. Chi Tiết Các Thay Đổi (Changes Implemented)

### 2.1 Cập nhật `internal/telemetry/cache.go`
1. Bổ sung hằng số `DefaultQuotaLogRetentionDays = 7`:
   ```go
   const (
       // DefaultQuotaCachePath is the canonical disk cache file for quota telemetry (Phase 3).
       DefaultQuotaCachePath = "logs/quota_log.json"
       // DefaultQuotaLogDir is the canonical directory for daily quota logs (Phase 4).
       DefaultQuotaLogDir = "logs/quota_log"
       // DefaultQuotaLogRetentionDays is the rolling retention window for daily quota logs (Phase 5).
       DefaultQuotaLogRetentionDays = 7
   )
   ```

2. Cài đặt hàm `PruneDailyQuotaLogs`:
   ```go
   // PruneDailyQuotaLogs keeps only the most recent keepCount daily quota log files in quotaLogDir.
   func PruneDailyQuotaLogs(quotaLogDir string, keepCount int) (int, error) {
       if keepCount <= 0 {
           keepCount = DefaultQuotaLogRetentionDays
       }
       dir := quotaLogDir
       if strings.TrimSpace(dir) == "" {
           dir = DefaultQuotaLogDir
       }

       entries, err := os.ReadDir(dir)
       if err != nil {
           if os.IsNotExist(err) {
               return 0, nil
           }
           return 0, err
       }

       var logFiles []os.DirEntry
       for _, entry := range entries {
           if !entry.IsDir() && strings.HasSuffix(entry.Name(), "_quota_log.json") {
               logFiles = append(logFiles, entry)
           }
       }

       // Sắp xếp giảm dần theo tên: file mới nhất (ngày lớn nhất) đứng đầu
       sort.Slice(logFiles, func(i, j int) bool {
           return logFiles[i].Name() > logFiles[j].Name()
       })

       if len(logFiles) <= keepCount {
           return 0, nil
       }

       deleted := 0
       for i := keepCount; i < len(logFiles); i++ {
           targetFile := filepath.Join(dir, logFiles[i].Name())
           if err := os.Remove(targetFile); err == nil {
               deleted++
           }
       }

       return deleted, nil
   }
   ```

### 2.2 Cập nhật `internal/ui/app.go`
Trong hàm `NewAppWithFyne`:
```go
	// 4. Initialize Automation Controller
	if mgr != nil {
		a.controller = automation.NewAutomationController(mgr, a.telemetryPool, a.quotaCache, nil)
	}

	// Tự động dọn dẹp các log quota cũ hơn 7 ngày trong goroutine nền nhẹ
	go func() {
		quotaDir := filepath.Join(logsDir, "quota_log")
		_, _ = telemetry.PruneDailyQuotaLogs(quotaDir, telemetry.DefaultQuotaLogRetentionDays)
	}()

	return a
```

### 2.3 Bổ sung Unit Tests tại `internal/telemetry/cache_test.go`
Bổ sung 4 unit tests mới:
1. `TestPruneDailyQuotaLogs_UnderLimit`: Kiểm tra khi có 3 file log (< 7), gọi `PruneDailyQuotaLogs(dir, 7)` trả về 0 deleted và giữ nguyên cả 3 file.
2. `TestPruneDailyQuotaLogs_OverLimit`: Kiểm tra khi có 10 file log (`2026-10-01` đến `2026-10-10`), gọi `PruneDailyQuotaLogs(dir, 7)` trả về 3 deleted, xóa đúng 3 file cũ nhất (`01`, `02`, `03`) và giữ nguyên 7 file mới nhất (`04` đến `10`).
3. `TestPruneDailyQuotaLogs_DefaultRetentionAndUnrelatedFiles`: Kiểm tra khi truyền `keepCount = 0`, hàm tự động áp dụng `DefaultQuotaLogRetentionDays = 7`, đồng thời bỏ qua các file không phải `_quota_log.json` và các thư mục con.
4. `TestPruneDailyQuotaLogs_NonExistentDir`: Kiểm tra thư mục không tồn tại trả về `(0, nil)` mà không gây lỗi hay crash.

---

## 3. Bằng Chứng Xác Minh Thực Nghiệm (Verification Evidence)

### 3.1 Unit Test Package Telemetry (`go test -v ./internal/telemetry/...`)
```text
=== RUN   TestGetTodayQuotaLogPath
--- PASS: TestGetTodayQuotaLogPath (0.00s)
=== RUN   TestDailyQuotaLog_LoadTodayOrFallback
--- PASS: TestDailyQuotaLog_LoadTodayOrFallback (0.03s)
=== RUN   TestDailyQuotaLog_TodayExists
--- PASS: TestDailyQuotaLog_TodayExists (0.01s)
=== RUN   TestDailyQuotaLog_MultiplePreviousDays
--- PASS: TestDailyQuotaLog_MultiplePreviousDays (0.01s)
=== RUN   TestDailyQuotaLog_EmptyDir
--- PASS: TestDailyQuotaLog_EmptyDir (0.00s)
=== RUN   TestDailyQuotaLog_SaveCreatesDirectory
--- PASS: TestDailyQuotaLog_SaveCreatesDirectory (0.02s)
=== RUN   TestDailyQuotaLog_DirectoryPathTarget
--- PASS: TestDailyQuotaLog_DirectoryPathTarget (0.02s)
=== RUN   TestPruneDailyQuotaLogs_UnderLimit
--- PASS: TestPruneDailyQuotaLogs_UnderLimit (0.00s)
=== RUN   TestPruneDailyQuotaLogs_OverLimit
--- PASS: TestPruneDailyQuotaLogs_OverLimit (0.01s)
=== RUN   TestPruneDailyQuotaLogs_DefaultRetentionAndUnrelatedFiles
--- PASS: TestPruneDailyQuotaLogs_DefaultRetentionAndUnrelatedFiles (0.01s)
=== RUN   TestPruneDailyQuotaLogs_NonExistentDir
--- PASS: TestPruneDailyQuotaLogs_NonExistentDir (0.00s)
=== RUN   TestDualCycleQuota_IndependentFractionsAndMissingBuckets
--- PASS: TestDualCycleQuota_IndependentFractionsAndMissingBuckets (0.00s)
=== RUN   TestAccountQuotaData_ZeroGPTSerialization
--- PASS: TestAccountQuotaData_ZeroGPTSerialization (0.00s)
=== RUN   TestParseQuotaSummaryWithBuckets_F1EdgeCases
--- PASS: TestParseQuotaSummaryWithBuckets_F1EdgeCases (0.00s)
=== RUN   TestRetrieveUserQuotaSummary_RESTGroups
--- PASS: TestRetrieveUserQuotaSummary_RESTGroups (0.47s)
=== RUN   TestF1_RESTLegacyAmountWithoutFraction
--- PASS: TestF1_RESTLegacyAmountWithoutFraction (0.00s)
=== RUN   TestF1_RESTLegacyMissingFractionAndAmount
--- PASS: TestF1_RESTLegacyMissingFractionAndAmount (0.00s)
=== RUN   TestF1_RESTLegacyClaudeFractionZero_NoWeeklySynthesis
--- PASS: TestF1_RESTLegacyClaudeFractionZero_NoWeeklySynthesis (0.00s)
=== RUN   TestF1_ConnectRPCSummaryClaudeWeeklyZero
--- PASS: TestF1_ConnectRPCSummaryClaudeWeeklyZero (0.00s)
=== RUN   TestF1_CachePersistence_WeeklyZeroCycle
--- PASS: TestF1_CachePersistence_WeeklyZeroCycle (0.02s)
=== RUN   TestF1_CachePersistence_LegacyCacheMissingFamily
--- PASS: TestF1_CachePersistence_LegacyCacheMissingFamily (0.00s)
=== RUN   TestF1_CachePersistence_ClaudeObjectWithoutRemainingFraction
--- PASS: TestF1_CachePersistence_ClaudeObjectWithoutRemainingFraction (0.00s)
=== RUN   TestF1_BothCyclesZero_ClaudeAndGemini_NoRaw_NoReset
--- PASS: TestF1_BothCyclesZero_ClaudeAndGemini_NoRaw_NoReset (0.02s)
=== RUN   TestF1_PositiveLegacyFraction_WithBothCyclesZero
--- PASS: TestF1_PositiveLegacyFraction_WithBothCyclesZero (0.02s)
=== RUN   TestDisabledMetadataPassThrough_ParserCacheRoundTrip
--- PASS: TestDisabledMetadataPassThrough_ParserCacheRoundTrip (0.02s)
=== RUN   TestQuotaClient_RetrieveUserQuota_Success
--- PASS: TestQuotaClient_RetrieveUserQuota_Success (0.00s)
=== RUN   TestQuotaClient_RetrieveUserQuota_SnakeCaseAndProtobuf
--- PASS: TestQuotaClient_RetrieveUserQuota_SnakeCaseAndProtobuf (0.00s)
=== RUN   TestQuotaClient_RetrieveUserQuota_StatusCategories
--- PASS: TestQuotaClient_RetrieveUserQuota_StatusCategories (0.00s)
=== RUN   TestQuotaClient_RetrieveUserQuota_Errors
--- PASS: TestQuotaClient_RetrieveUserQuota_Errors (0.00s)
=== RUN   TestQuotaCache_Operations
--- PASS: TestQuotaCache_Operations (0.02s)
=== RUN   TestTelemetryPool_PollAccountAndPollAll
--- PASS: TestTelemetryPool_PollAccountAndPollAll (0.51s)
=== RUN   TestTelemetryPool_PartialFailureHandling
--- PASS: TestTelemetryPool_PartialFailureHandling (0.41s)
=== RUN   TestAutoTelemetryLoop_LifecycleAndSingleFlight
--- PASS: TestAutoTelemetryLoop_LifecycleAndSingleFlight (0.48s)
=== RUN   TestTelemetry_ConcurrentRace
--- PASS: TestTelemetry_ConcurrentRace (1.94s)
=== RUN   TestQuotaCache_EdgeCasesAndFlatMap
--- PASS: TestQuotaCache_EdgeCasesAndFlatMap (0.02s)
=== RUN   TestTelemetryPool_StoreAccountSourceIntegration
--- PASS: TestTelemetryPool_StoreAccountSourceIntegration (0.48s)
=== RUN   TestAutoTelemetryLoop_EdgeCases
--- PASS: TestAutoTelemetryLoop_EdgeCases (0.02s)
=== RUN   TestCoverageBooster
--- PASS: TestCoverageBooster (0.45s)
=== RUN   TestDiscoverAntigravityUserAgent_Priorities
--- PASS: TestDiscoverAntigravityUserAgent_Priorities (0.41s)
=== RUN   TestQuotaClient_UserAgentAndGoogApiClientHeaders
--- PASS: TestQuotaClient_UserAgentAndGoogApiClientHeaders (0.00s)
=== RUN   TestParseQuotaSummary_ConnectRPC
--- PASS: TestParseQuotaSummary_ConnectRPC (0.00s)
=== RUN   TestHybridTelemetry_ActiveAccountConnectRPC
--- PASS: TestHybridTelemetry_ActiveAccountConnectRPC (0.03s)
PASS
ok  	antigravity-switch/internal/telemetry	7.038s
```

### 3.2 Unit Test Package UI (`go test -v ./internal/ui/...`)
```text
PASS
ok  	antigravity-switch/internal/ui	27.422s
```

### 3.3 Full Test Suite (`go test ./...`)
```text
ok  	antigravity-switch/cmd/switcher	2.134s
ok  	antigravity-switch/internal/automation	14.851s
ok  	antigravity-switch/internal/connector	(cached)
ok  	antigravity-switch/internal/manager	(cached)
ok  	antigravity-switch/internal/model	(cached)
ok  	antigravity-switch/internal/reloader	(cached)
ok  	antigravity-switch/internal/scanner	(cached)
ok  	antigravity-switch/internal/store	6.884s
ok  	antigravity-switch/internal/telemetry	9.648s
ok  	antigravity-switch/internal/ui	21.191s
ok  	antigravity-switch/internal/wincred	(cached)
```

### 3.4 Windows Clean Build Verification (`pwsh -File .\scripts\build_windows.ps1 -Clean`)
```text
================================================================================
⚡ Antigravity Multi-Account Switcher - Windows Build System
   OutDir: bin | SkipTests: False | Clean: True
================================================================================
[1/4] Cleaning output artifacts...
  Removed directory: bin
  Clean complete.
[2/4] Running pre-flight tests & vet...
  [PASS] Verification succeeded.
[3/4] Compiling target binaries...
  -> Checking Windows application icon and resource...
  -> Compiling resource with windres: cmd\switcher\app.rc -> cmd\switcher\rsrc_windows_amd64.syso
  [SUCCESS] Compiled PE resource object: cmd\switcher\rsrc_windows_amd64.syso (12552 bytes)
  -> Building Standalone GUI Binary (Fyne, CGO_ENABLED=1)...
  [SUCCESS] Created: bin\antigravity-switch.exe
  Compile Duration: 6705 ms | Size: 27.73 MB (29081088 bytes)
[4/4] Executing quick verification...
  [PASS] Embedded PE application icon verified on bin\antigravity-switch.exe (32x32)
  [INFO] GUI binary built at bin\antigravity-switch.exe.
================================================================================
⚡ BUILD SUCCESSFUL
================================================================================
```

---

## 4. Tuân Thủ Kỷ Luật (Discipline & Constraints Checklist)

| Yêu cầu kỷ luật | Kết quả kiểm tra | Minh chứng |
|---|---|---|
| **Chỉ thi công Phase 5** | TUÂN THỦ 100% | Chỉ cập nhật hàm prune và kích hoạt trong goroutine nền |
| **Không làm trước Phase 6** | TUÂN THỦ 100% | Không đụng vào `active_threads_history` hay `reloader` |
| **CẤM SỬA SCRIPT BUILD** | TUÂN THỦ 100% | `scripts/build_windows.ps1` 0 lines diff |
| **Không dùng anvien** | TUÂN THỦ 100% | Hoàn toàn không triệu hồi anvien |
| **Code đến đâu build đến đó** | TUÂN THỦ 100% | Test từng package và build clean thành công |
| **Tránh rò rỉ rác test** | TUÂN THỦ 100% | Tất cả test đều dùng `t.TempDir()` độc lập |

---

## 5. Kết Luận & Bàn Giao

Giai đoạn 5 (PHASE 5: Tự động dọn dẹp 7 ngày gần nhất) đã hoàn thành xuất sắc, thỏa mãn 100% các tiêu chí nghiệm thu được giao phó trong bản đặc tả kiến trúc `SPEC-ARCH-20261021-LOGS-AND-DAILY-QUOTA-SPEC`. Toàn bộ mã nguồn và báo cáo sẵn sàng bàn giao cho Supervisor Lane nghiệm thu.
