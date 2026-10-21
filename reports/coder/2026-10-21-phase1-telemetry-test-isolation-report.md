# Báo Cáo Thi Công: Cô Lập Test Cache & Dọn Dẹp Rò Rỉ Unit Test Telemetry (Phase 1)

- **Ngày thi công:** 2026-10-21
- **Vai trò thực hiện:** Coder
- **Thẩm quyền kiến trúc:** `reports/architect/2026-10-21-logs-directory-migration-and-renaming-spec.md` (Mã văn kiện: `SPEC-ARCH-20261021-LOGS-AND-DAILY-QUOTA-SPEC`)
- **Git Commit Baseline:** `1a119ce` (docs(system-architect): record lane completion report for logs directory standardization and daily quota)
- **Trạng thái:** HOÀN THÀNH 100% TIÊU CHUẨN NGHIỆM THU PHASE 1 (PHASE 1 COMPLETE - READY FOR SUPERVISOR)

---

## 1. Bản Đồ Họ Bất Biến (Invariant Family Map)

- **Tên họ bất biến:** Telemetry Test Cache Isolation & Artifact Purge (`invariant-telemetry-test-isolation`)
- **Nguồn thẩm quyền (SSOT):** `SPEC-ARCH-20261021-LOGS-AND-DAILY-QUOTA-SPEC` Mục II - Giai đoạn 1 (Phase 1).
- **Các bề mặt thực thi liên quan (Sibling Runtime Surfaces):**
  - Unit test telemetry: `internal/telemetry/telemetry_test.go` (hàm `TestCoverageBooster`).
  - Cấu trúc thư mục mã nguồn: `internal/telemetry/` tuyệt đối không bị ô nhiễm bởi các thư mục sinh ra trong lúc chạy kiểm thử.
  - Sibling surfaces checked:
    - `defCache` trong `telemetry_test.go:689` chỉ kiểm tra getter/setter đường dẫn và cập nhật bộ nhớ, không gọi `SaveToDisk` trên đường dẫn mặc định.
    - Toàn bộ các test suite khác trong `internal/telemetry/` (`quota_test.go`, các case khác trong `telemetry_test.go`) đều sử dụng `t.TempDir()` độc lập.
- **Đường dẫn/Cơ chế cũ bị loại bỏ (Forbidden Legacy Fallback):**
  - Khởi tạo `NewQuotaCache("")` bên trong unit test khiến `c.filePath` nhận giá trị mặc định tương đối `accounts_store/.quota_cache.json`, dẫn đến việc tự ý tạo thư mục rác `internal/telemetry/accounts_store/` khi `SaveToDisk()` được kích hoạt trong `failPool.PollAccount`.
- **Artifacts và Tests cũ đã cập nhật/dọn dẹp:**
  - Xóa bỏ triệt để 100% thư mục rác `internal/telemetry/accounts_store/`.
  - Cập nhật `internal/telemetry/telemetry_test.go` (dòng 944-955) để sử dụng `testCachePath := filepath.Join(tempDir, "test_quota_cache.json")`.
- **Ma trận xác minh (Verify Matrix):**
  - Primary path: `go test -v ./internal/telemetry/...` PASS 100%.
  - Leak isolation path: `Test-Path 'internal\telemetry\accounts_store'` -> `False` sau khi chạy toàn bộ test suite.
  - Full suite regression: `go test ./...` PASS 100%.
  - Production compile verification: `pwsh -File .\scripts\build_windows.ps1 -Clean` Exit Code 0.
  - Store integrity verification: Thư mục `accounts_store/` tại root không bị biến đổi, giữ nguyên toàn bộ credential và history.
  - Build script discipline: `scripts/build_windows.ps1` giữ nguyên 100% (0 lines diff).
- **Residual unverified surfaces:** `none`.

---

## 2. Chi Tiết Các Tệp Thay Đổi (Files Changed)

### 2.1 Xóa bỏ thư mục rác (Purged Directory)
- Đã xóa hoàn toàn: `internal/telemetry/accounts_store/` (chứa `.quota_cache.json` rác sinh ra từ test trước đó).

### 2.2 Sửa đổi unit test (`internal/telemetry/telemetry_test.go`)
- Vị trí: Hàm `TestCoverageBooster` (dòng 944–955).
- Khởi tạo biến `testCachePath := filepath.Join(tempDir, "test_quota_cache.json")`.
- Thay thế `NewQuotaCache("")` bằng `NewQuotaCache(testCachePath)` cho cả hai đối tượng `cache` và `cacheLoad`.
- Đảm bảo khi `failPool.PollAccount(context.Background(), "user@fail.com")` thực hiện `SaveToDisk()`, file cache được ghi an toàn vào `tempDir` và tự động được dọn dẹp bởi Go test runtime khi kết thúc test.

---

## 3. Bằng Chứng Thực Thi & Xác Minh (Evidence & Benchmarks)

### 3.1 Kiểm tra cô lập thư mục rác (Isolation Check)
```powershell
pwsh -Command "Test-Path 'internal\telemetry\accounts_store'"
# Kết quả: False
```

### 3.2 Kết quả chạy kiểm thử Telemetry (`go test ./internal/telemetry/... -v`)
```text
=== RUN   TestDualCycleQuota_IndependentFractionsAndMissingBuckets
--- PASS: TestDualCycleQuota_IndependentFractionsAndMissingBuckets (0.00s)
=== RUN   TestAccountQuotaData_ZeroGPTSerialization
--- PASS: TestAccountQuotaData_ZeroGPTSerialization (0.00s)
=== RUN   TestParseQuotaSummaryWithBuckets_F1EdgeCases
--- PASS: TestParseQuotaSummaryWithBuckets_F1EdgeCases (0.00s)
=== RUN   TestRetrieveUserQuotaSummary_RESTGroups
--- PASS: TestRetrieveUserQuotaSummary_RESTGroups (0.46s)
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
--- PASS: TestF1_CachePersistence_LegacyCacheMissingFamily (0.01s)
=== RUN   TestF1_CachePersistence_ClaudeObjectWithoutRemainingFraction
--- PASS: TestF1_CachePersistence_ClaudeObjectWithoutRemainingFraction (0.00s)
=== RUN   TestF1_BothCyclesZero_ClaudeAndGemini_NoRaw_NoReset
--- PASS: TestF1_BothCyclesZero_ClaudeAndGemini_NoRaw_NoReset (0.01s)
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
--- PASS: TestTelemetryPool_PollAccountAndPollAll (0.50s)
=== RUN   TestTelemetryPool_PartialFailureHandling
--- PASS: TestTelemetryPool_PartialFailureHandling (0.43s)
=== RUN   TestAutoTelemetryLoop_LifecycleAndSingleFlight
--- PASS: TestAutoTelemetryLoop_LifecycleAndSingleFlight (0.51s)
=== RUN   TestTelemetry_ConcurrentRace
--- PASS: TestTelemetry_ConcurrentRace (1.82s)
=== RUN   TestQuotaCache_EdgeCasesAndFlatMap
--- PASS: TestQuotaCache_EdgeCasesAndFlatMap (0.02s)
=== RUN   TestTelemetryPool_StoreAccountSourceIntegration
--- PASS: TestTelemetryPool_StoreAccountSourceIntegration (0.47s)
=== RUN   TestAutoTelemetryLoop_EdgeCases
--- PASS: TestAutoTelemetryLoop_EdgeCases (0.02s)
=== RUN   TestCoverageBooster
--- PASS: TestCoverageBooster (0.46s)
=== RUN   TestDiscoverAntigravityUserAgent_Priorities
--- PASS: TestDiscoverAntigravityUserAgent_Priorities (0.41s)
=== RUN   TestQuotaClient_UserAgentAndGoogApiClientHeaders
--- PASS: TestQuotaClient_UserAgentAndGoogApiClientHeaders (0.00s)
=== RUN   TestParseQuotaSummary_ConnectRPC
--- PASS: TestParseQuotaSummary_ConnectRPC (0.00s)
=== RUN   TestHybridTelemetry_ActiveAccountConnectRPC
--- PASS: TestHybridTelemetry_ActiveAccountConnectRPC (0.03s)
PASS
ok  	antigravity-switch/internal/telemetry	6.843s
```

### 3.3 Toàn bộ Test Suite (`go test ./...`)
```text
ok  	antigravity-switch/cmd/switcher	(cached)
ok  	antigravity-switch/internal/automation	(cached)
ok  	antigravity-switch/internal/connector	(cached)
ok  	antigravity-switch/internal/manager	(cached)
ok  	antigravity-switch/internal/model	(cached)
ok  	antigravity-switch/internal/reloader	(cached)
ok  	antigravity-switch/internal/scanner	(cached)
ok  	antigravity-switch/internal/store	5.233s
ok  	antigravity-switch/internal/telemetry	7.205s
ok  	antigravity-switch/internal/ui	(cached)
ok  	antigravity-switch/internal/wincred	(cached)
```

### 3.4 Kết quả kiểm chứng biên dịch Windows (`pwsh -File .\scripts\build_windows.ps1 -Clean`)
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
ok  	antigravity-switch/internal/store	4.295s
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
  Compile Duration: 5814 ms | Size: 27.73 MB (29073408 bytes)
[4/4] Executing quick verification...
  [PASS] Embedded PE application icon verified on bin\antigravity-switch.exe (32x32)
  [INFO] GUI binary built at bin\antigravity-switch.exe.
================================================================================
⚡ BUILD SUCCESSFUL
================================================================================
```

---

## 4. Kết Luận & Bàn Giao

- Giai đoạn 1 (Phase 1) đã hoàn tất 100% mục tiêu:
  1. Unit test `TestCoverageBooster` trong `internal/telemetry/telemetry_test.go` được cô lập hoàn toàn vào thư mục tạm `tempDir`.
  2. Thư mục rác `internal/telemetry/accounts_store` bị xóa sạch và không tái xuất hiện.
  3. Thư mục `accounts_store/` ở root hoàn toàn nguyên vẹn.
  4. Toàn bộ test suite pass 100%.
  5. Script build `scripts/build_windows.ps1` nguyên vẹn 100% (0 lines diff).
  6. Biên dịch thành công với exit code 0.
- Sẵn sàng chuyển giao cho Supervisor nghiệm thu Phase 1 trước khi tiến hành Phase 2.
