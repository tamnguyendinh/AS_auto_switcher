# Đặc Tả Kiến Trúc & Chỉ Dẫn Kỹ Thuật Chi Tiết Cho Coder: Chuẩn Hóa Thư Mục Logs, Phân Mảnh Quota Theo Ngày & Tự Động Dọn Dẹp 7 Ngày
(Logs Directory Standardization, Daily Quota Sharding & 7-Day Rolling Retention — Complete Implementation Recipe)

- **Ngày lập:** 2026-10-21
- **Vai trò:** System Architect (Kiến trúc sư Hệ thống)
- **Thẩm quyền:** Phê chuẩn trực tiếp từ Human Owner
- **Mã văn kiện:** `SPEC-ARCH-20261021-LOGS-AND-DAILY-QUOTA-SPEC`
- **Trạng thái:** BẢN CHÍNH THỨC BAN HÀNH (OFFICIAL RELEASE)
- **Đối tượng đọc:** Coder, Supervisor, QA

---

## I. TỔNG QUAN VÀ NGUYÊN TẮC CỐT LÕI (OBJECTIVES & INVARIANTS)

Bản đặc tả này cung cấp **công thức triển khai chính xác đến từng dòng mã (step-by-step recipe)**, đảm bảo Coder đọc vào có thể code được ngay mà không cần suy nghĩ hay phán đoán thêm.

### Các điều khoản kỷ luật sắt (Mandatory Invariants):
1. **QUY TRÌNH 5 GIAI ĐOẠN ĐỘC LẬP (STRICT 5-PHASE GATE):** Coder triển khai lần lượt từ Phase 1 đến Phase 5. Không được nhảy cóc hoặc làm xáo trộn thứ tự.
2. **KHO TÀI KHOẢN THUẦN KHIẾT (PURE ACCOUNT STORE):** Sau khi hoàn thành Phase 2, `accounts_store/` CHỈ CHỨA các file credential tài khoản (`<email>.json`) và thư mục snapshot lịch sử (`active_threads_history/`). Tuyệt đối không còn file log, cache hay preference nào.
3. **CẤU TRÚC PHÂN MẢNH THEO NGÀY (DAILY SHARDING):** Nhật ký quota được lưu tại `logs/quota_log/YYYY-MM-DD_quota_log.json`. Mỗi ngày lưu đúng một file theo định dạng chuẩn ISO.
4. **GIỮ TỐI ĐA 7 FILE GẦN NHẤT (7-DAY ROLLING RETENTION):** Tự động dọn dẹp các file cũ hơn 7 ngày tại `logs/quota_log/`, chỉ giữ lại đúng 7 file mới nhất tương ứng với 7 ngày gần nhất.
5. **KHỞI ĐỘNG NON-BLOCKING & FAIL-SAFE:** Cơ chế dọn dẹp chạy ngầm sau khi khởi động app xong, nếu gặp lỗi I/O (file locked) thì bỏ qua an toàn, không bao giờ làm gián đoạn UI.
6. **KHÔNG VIẾT CODE CHUYỂN TIẾP RƯỜM RÀ (ZERO LEGACY MIGRATION BLOAT):** Tuyệt đối không viết thêm logic copy/convert dữ liệu cũ qua lại giữa các phase để tránh làm phình to codebase.

---

## II. CHỈ DẪN TRIỂN KHAI CHI TIẾT TỪNG BƯỚC CHO CODER (EXHAUSTIVE STEP-BY-STEP RECIPE)

---

### GIAI ĐOẠN 1 (PHASE 1): SỬA UNIT TEST & XÓA RÁC CÂY MÃ NGUỒN

#### 1. Xóa thư mục rác
Chạy lệnh xóa thư mục rác đã bị rò rỉ:
```powershell
Remove-Item -Recurse -Force "internal\telemetry\accounts_store"
```

#### 2. Sửa mã nguồn test tại `internal/telemetry/telemetry_test.go`
Tìm đến hàm `TestTelemetryPool_AdditionalCoverage` (khoảng dòng 943–956):
- **Code hiện tại (gây rò rỉ):**
  ```go
  // Dòng 945:
  cache := NewQuotaCache("")
  // Dòng 950:
  cacheLoad := NewQuotaCache("")
  ```
- **Sửa thành (cô lập an toàn vào `tempDir`):**
  ```go
  testCachePath := filepath.Join(tempDir, "test_quota_cache.json")
  cache := NewQuotaCache(testCachePath)
  ...
  cacheLoad := NewQuotaCache(testCachePath)
  ```
- **Kiểm chứng:** Chạy `go test -v ./internal/telemetry/...` $\rightarrow$ Test PASS 100% và thư mục `internal/telemetry/accounts_store/` không bao giờ xuất hiện lại.

---

### GIAI ĐOẠN 2 (PHASE 2): THIẾT LẬP THƯ MỤC `logs/` & DI DỜI 3 FILE

#### 1. Cập nhật `.gitignore`
Mở file `.gitignore`, bổ sung dòng `logs/` ngay bên dưới `accounts_store/`:
```gitignore
accounts_store/
logs/
discovered_accounts/
```

#### 2. Tạo thư mục `logs/` và di dời 3 file hiện hữu
Thực hiện di chuyển 3 file runtime ra khỏi `accounts_store/` vào `logs/`:
```powershell
New-Item -ItemType Directory -Force -Path "logs"
Move-Item -Path "accounts_store\.quota_cache.json" -Destination "logs\.quota_cache.json"
Move-Item -Path "accounts_store\.ui_preferences.json" -Destination "logs\.ui_preferences.json"
Move-Item -Path "accounts_store\switch_timeline.log" -Destination "logs\switch_timeline.log"
```

#### 3. Cập nhật mã nguồn hệ thống
1. **Trong `internal/telemetry/cache.go`:**
   Sửa hằng số:
   ```go
   const (
       DefaultQuotaCachePath = "logs/.quota_cache.json"
   )
   ```
2. **Trong `internal/ui/app.go`:**
   - Sửa hằng số:
     ```go
     const (
         DefaultUIPreferencesPath = "logs/.ui_preferences.json"
     )
     ```
   - Trong hàm `NewAppWithFyne`:
     ```go
     cachePath := filepath.Join("logs", ".quota_cache.json")
     a.preferencesPath = filepath.Join("logs", ".ui_preferences.json")
     ```
     *(Bỏ logic `filepath.Join(mgr.Store().GetStoreDir(), ...)` cũ để không còn gắn vào `accounts_store`)*.

---

### GIAI ĐOẠN 3 (PHASE 3): ĐỔI TÊN CHUẨN HÓA (BỎ DẤU CHẤM ẨN)

#### 1. Đổi tên tệp trên đĩa
```powershell
Rename-Item -Path "logs\.ui_preferences.json" -NewName "ui_preferences.json"
Rename-Item -Path "logs\.quota_cache.json" -NewName "quota_log.json"
```

#### 2. Cập nhật mã nguồn hệ thống
1. **Trong `internal/telemetry/cache.go`:**
   ```go
   const (
       DefaultQuotaCachePath = "logs/quota_log.json"
   )
   ```
2. **Trong `internal/ui/app.go`:**
   ```go
   const (
       DefaultUIPreferencesPath = "logs/ui_preferences.json"
   )
   ```
   Và trong `NewAppWithFyne`:
   ```go
   cachePath := filepath.Join("logs", "quota_log.json")
   a.preferencesPath = filepath.Join("logs", "ui_preferences.json")
   ```
   *(Trỏ trực tiếp vào tên mới, không viết code fallback chuyển tiếp rườm rà).*

---

### GIAI ĐOẠN 4 (PHASE 4): PHÂN MẢNH QUOTA THEO NGÀY (`logs/quota_log/YYYY-MM-DD_quota_log.json`)

#### 1. Cấu trúc lưu trữ
`logs/quota_log.json` được chuyển đổi thành thư mục:
```text
logs/
├── quota_log/
│   ├── 2026-10-21_quota_log.json
│   └── 2026-10-22_quota_log.json
├── ui_preferences.json
└── switch_timeline.log
```

#### 2. Cập nhật `internal/telemetry/cache.go`
1. Bổ sung các hằng số và hàm sinh đường dẫn ngày:
   ```go
   const (
       DefaultQuotaLogDir = "logs/quota_log"
   )

   // GetTodayQuotaLogPath returns the canonical path for today's quota log file.
   func GetTodayQuotaLogPath(baseDir ...string) string {
       dir := DefaultQuotaLogDir
       if len(baseDir) > 0 && strings.TrimSpace(baseDir[0]) != "" {
           dir = baseDir[0]
       }
       today := time.Now().Format("2006-01-02")
       return filepath.Join(dir, fmt.Sprintf("%s_quota_log.json", today))
   }
   ```
2. Cập nhật hàm `LoadFromDisk`:
   - Nếu `c.filePath` trỏ đến thư mục hoặc đường dẫn mặc định:
     1. Kiểm tra file ngày hôm nay (`GetTodayQuotaLogPath()`). Nếu tồn tại, nạp file này.
     2. Nếu file ngày hôm nay chưa tồn tại: Quét thư mục `DefaultQuotaLogDir`, lọc các file có đuôi `_quota_log.json`, sắp xếp giảm dần theo tên file, nạp file của ngày gần nhất trước đó.
     3. Nếu thư mục rỗng, bắt đầu với ma trận rỗng (chờ auto-poll nạp dữ liệu).
3. Cập nhật hàm `SaveToDisk`:
   - Ghi dữ liệu trực tiếp vào file ngày hôm nay: `GetTodayQuotaLogPath()`.
   - Đảm bảo tự động tạo thư mục `os.MkdirAll(filepath.Dir(targetPath), 0755)`.

#### 3. Cập nhật `internal/ui/app.go`
Trong `NewAppWithFyne`:
```go
cachePath := telemetry.GetTodayQuotaLogPath()
a.quotaCache = telemetry.NewQuotaCache(cachePath)
_ = a.quotaCache.LoadFromDisk()
```

---

### GIAI ĐOẠN 5 (PHASE 5): TỰ ĐỘNG DỌN DẸP 7 NGÀY GẦN NHẤT (ROLLING 7-DAY PRUNING)

#### 1. Bổ sung hàm cắt tỉa tại `internal/telemetry/cache.go`
```go
const DefaultQuotaLogRetentionDays = 7

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

#### 2. Kích hoạt tự động khi App khởi động xong tại `internal/ui/app.go`
Trong hàm `NewAppWithFyne` (ngay cuối hàm trước khi `return a`):
```go
// Tự động dọn dẹp các log quota cũ hơn 7 ngày trong goroutine nền nhẹ
go func() {
    _, _ = telemetry.PruneDailyQuotaLogs(telemetry.DefaultQuotaLogDir, telemetry.DefaultQuotaLogRetentionDays)
}()
```

---

## III. MA TRẬN KIỂM THỬ TỰ ĐỘNG BẮT BUỘC CHO CODER (TEST MATRIX)

### 1. Unit Tests tại `internal/telemetry/cache_test.go`
1. **`TestPruneDailyQuotaLogs_UnderLimit`:**
   - Tạo thư mục tạm với 3 file log: `2026-10-19_quota_log.json`, `2026-10-20_quota_log.json`, `2026-10-21_quota_log.json`.
   - Gọi `PruneDailyQuotaLogs(dir, 7)`.
   - Kết quả: `deletedCount == 0`, cả 3 file vẫn còn nguyên.
2. **`TestPruneDailyQuotaLogs_OverLimit`:**
   - Tạo thư mục tạm với 10 file log từ `2026-10-01_...` đến `2026-10-10_...`.
   - Gọi `PruneDailyQuotaLogs(dir, 7)`.
   - Kết quả:
     - `deletedCount == 3`.
     - 7 file mới nhất (`2026-10-04` đến `2026-10-10`) tồn tại.
     - 3 file cũ nhất (`2026-10-01`, `2026-10-02`, `2026-10-03`) bị xóa.
3. **`TestDailyQuotaLog_LoadTodayOrFallback`:**
   - Tạo file hôm qua `2026-10-20_quota_log.json` có dữ liệu mock.
   - Gọi `LoadFromDisk` khi chưa có file hôm nay `2026-10-21`.
   - Kết quả: Nạp thành công dữ liệu từ file hôm qua.
   - Gọi `SaveToDisk` $\rightarrow$ Kết quả: Sinh ra file đúng ngày hôm nay `2026-10-21_quota_log.json`.

---

## IV. TIÊU CHÍ NGHIỆM THU CUỐI CÙNG (FINAL ACCEPTANCE CRITERIA)

1. `go test ./...` chạy PASS 100% không có lỗi.
2. Thư mục `accounts_store/` tuyệt đối không còn file `.quota_cache.json`, `.ui_preferences.json` hay `switch_timeline.log`.
3. Thư mục `logs/` tồn tại, được ignore trên Git, bên trong có thư mục `logs/quota_log/` chứa log ngày và các file `ui_preferences.json`, `switch_timeline.log`.
4. App khởi động mượt mà, hiển thị đầy đủ thông số quota và theme của người dùng.
