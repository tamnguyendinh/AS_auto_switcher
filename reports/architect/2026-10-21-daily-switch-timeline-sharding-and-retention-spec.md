# Đặc Tả Kiến Trúc & Chỉ Dẫn Kỹ Thuật Chi Tiết Cho Coder: Phân Mảnh Nhật Ký Chuyển Tài Khoản Theo Ngày & Tự Động Dọn Dẹp 7 Ngày
(Daily Switch Timeline Sharding & 7-Day Rolling Retention Specification — Implementation Recipe)

- **Ngày lập:** 2026-10-21
- **Vai trò:** System Architect (Kiến trúc sư Hệ thống)
- **Thẩm quyền:** Phê chuẩn trực tiếp từ Human Owner
- **Mã văn kiện:** `SPEC-ARCH-20261021-DAILY-SWITCH-TIMELINE-SPEC`
- **Trạng thái:** DRAFT (Trình Human Owner phê duyệt)
- **Đối tượng đọc:** Coder, Supervisor, QA

---

## I. MỤC TIÊU VÀ NGUYÊN TẮC CỐT LÕI (OBJECTIVES & INVARIANTS)

Bản đặc tả này chuẩn hóa cơ chế ghi nhật ký tiến trình chuyển tài khoản (`switch_timeline`), loại bỏ file log đơn lẻ mồ côi `logs/switch_timeline.log` và thiết lập hệ thống ghi log theo ngày tương tự như `logs/quota_log/`.

### Các điều khoản kỷ luật sắt (Mandatory Invariants):
1. **QUY TỤ VÀO THƯ MỤC RIÊNG (ISOLATED TIMELINE DIRECTORY):**
   - File log đơn lẻ `logs/switch_timeline.log` phải được chuyển thành thư mục con: `logs/switch_timeline/`.
   - Tuyệt đối không để bất kỳ file log trôi nổi nào bên ngoài thư mục con trong `logs/`.
2. **PHÂN MẢNH THEO NGÀY CHUẨN ISO (DAILY LOG SHARDING):**
   - Nhật ký chuyển đổi từng ngày được lưu theo định dạng: `logs/switch_timeline/YYYY-MM-DD_switch_timeline.log` (ví dụ: `2026-10-21_switch_timeline.log`).
   - Sắp xếp giảm dần theo tên file luôn đồng nghĩa với thứ tự thời gian giảm dần (mới nhất $\rightarrow$ cũ nhất).
3. **GIỮ TỐI ĐA 7 FILE GẦN NHẤT (7-DAY ROLLING RETENTION):**
   - Tự động dọn dẹp các file cũ hơn 7 ngày tại `logs/switch_timeline/`, chỉ giữ lại đúng 7 file mới nhất tương ứng với 7 ngày gần nhất.
4. **KHỞI ĐỘNG NON-BLOCKING & FAIL-SAFE:**
   - Cơ chế dọn dẹp chạy ngầm sau khi khởi động app xong, nếu gặp lỗi I/O (file locked) thì bỏ qua an toàn, không bao giờ làm gián đoạn UI.
5. **GHI LOG NON-BLOCKING (ZERO IMPACT ON SWITCH CRITICAL PATH):**
   - Thao tác append log sự kiện vào file nhật ký diễn ra đồng thời với các bước trong `SwitchAccount`. Lỗi ghi file log (nếu có) chỉ bỏ qua an toàn, tuyệt đối không được làm fail hay đảo ngược kết quả chuyển tài khoản.

---

## II. HỆ THỐNG CẤU TRÚC THƯ MỤC ĐÍCH (TARGET DIRECTORY LAYOUT)

```text
logs/
├── quota_log/
│   ├── 2026-10-21_quota_log.json
│   └── ... (tối đa 7 ngày)
├── switch_timeline/
│   ├── 2026-10-21_switch_timeline.log
│   └── ... (tối đa 7 ngày)
├── active_threads_history/
│   ├── 2026-10-21-13-23-52_active_threads.json
│   └── ... (tối đa 2 file)
└── ui_preferences.json
```

---

## III. CHỈ DẪN TRIỂN KHAI CHI TIẾT TỪNG BƯỚC CHO CODER (EXHAUSTIVE STEP-BY-STEP RECIPE)

---

### BƯỚC 1: DI DỜI TỆP LOG CŨ TRÊN ĐĨA
File `logs/switch_timeline.log` hiện tại (file mồ côi từ ngày 17/10/2026) được chuyển vào thư mục con `logs/switch_timeline/` với tên chuẩn ngày:
```powershell
New-Item -ItemType Directory -Force -Path "logs\switch_timeline"
Move-Item -Path "logs\switch_timeline.log" -Destination "logs\switch_timeline\2026-10-17_switch_timeline.log"
```

---

### BƯỚC 2: BỔ SUNG MODULE GHI NHẬT KÝ VÀ CẮT TỈA TIMELINE

Tạo mới file `internal/manager/timeline.go` thuộc package `manager`:

```go
package manager

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	DefaultSwitchTimelineDir           = "logs/switch_timeline"
	DefaultSwitchTimelineRetentionDays = 7
)

// GetTodaySwitchTimelinePath returns the canonical path for today's switch timeline log file.
func GetTodaySwitchTimelinePath(baseDir ...string) string {
	dir := DefaultSwitchTimelineDir
	if len(baseDir) > 0 && strings.TrimSpace(baseDir[0]) != "" {
		dir = baseDir[0]
	}
	today := time.Now().Format("2006-01-02")
	return filepath.Join(dir, fmt.Sprintf("%s_switch_timeline.log", today))
}

// LogSwitchEvent appends a timestamped line to today's switch timeline log file.
// Format: [15:04:05.000] [+ 1234ms] [EVENT_TAG] Message
func LogSwitchEvent(timelineDir string, eventTag string, message string, elapsed ...time.Duration) {
	if strings.TrimSpace(timelineDir) == "" {
		timelineDir = DefaultSwitchTimelineDir
	}
	_ = os.MkdirAll(timelineDir, 0755)

	today := time.Now().Format("2006-01-02")
	filePath := filepath.Join(timelineDir, fmt.Sprintf("%s_switch_timeline.log", today))

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return // Fail-safe: ignore disk write errors
	}
	defer f.Close()

	nowStr := time.Now().Format("15:04:05.000")
	elapsedStr := ""
	if len(elapsed) > 0 {
		elapsedStr = fmt.Sprintf("[+ %6dms] ", elapsed[0].Milliseconds())
	}

	line := fmt.Sprintf("[%s] %s[%-16s] %s\n", nowStr, elapsedStr, eventTag, message)
	_, _ = f.WriteString(line)
}

// PruneDailySwitchTimelineLogs keeps only the most recent keepCount timeline log files.
func PruneDailySwitchTimelineLogs(timelineDir string, keepCount int) (int, error) {
	if keepCount <= 0 {
		keepCount = DefaultSwitchTimelineRetentionDays
	}
	dir := timelineDir
	if strings.TrimSpace(dir) == "" {
		dir = DefaultSwitchTimelineDir
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
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), "_switch_timeline.log") {
			logFiles = append(logFiles, entry)
		}
	}

	// Sắp xếp giảm dần theo tên: file mới nhất đứng đầu
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

---

### BƯỚC 3: TÍCH HỢP GHI NHẬT KÝ VÀO `SwitchAccount` TRONG `internal/manager/account_manager.go`

Trong hàm `SwitchAccount`:
1. Xác định thư mục timeline:
   ```go
   timelineDir := DefaultSwitchTimelineDir
   if m.store != nil && m.store.GetStoreDir() != "" {
       timelineDir = filepath.Join(filepath.Dir(m.store.GetStoreDir()), "logs", "switch_timeline")
   }
   ```
2. Ghi nhận các mốc sự kiện trong suốt quá trình switch:
   - **Đầu hàm:**
     ```go
     LogSwitchEvent(timelineDir, "SWITCH_START", fmt.Sprintf("Bắt đầu chuyển tài khoản từ %s sang %s", currentEmail, normTarget))
     ```
   - **Sau Bước 1 (Snapshot):**
     ```go
     LogSwitchEvent(timelineDir, "SNAPSHOT_DONE", fmt.Sprintf("Đã snapshot %d thread(s) đang hoạt động", len(activeThreads)), time.Since(start))
     ```
   - **Sau Bước 2 (Ghi wincred & Kill Language Server):**
     ```go
     LogSwitchEvent(timelineDir, "WINCRED_WRITE", fmt.Sprintf("Đã ghi token mới của %s vào WinCred", normTarget), time.Since(start))
     LogSwitchEvent(timelineDir, "KILL_LS", "Language Server đã được yêu cầu khởi động lại", time.Since(start))
     ```
   - **Sau RAM Check:**
     ```go
     LogSwitchEvent(timelineDir, "RAM_CHECK_DONE", fmt.Sprintf("Xác thực RAM thành công: %s", ramEmail), time.Since(start))
     ```
   - **Tại Bước 4 (Continuation):**
     ```go
     LogSwitchEvent(timelineDir, "DISPATCH_START", fmt.Sprintf("Bắt đầu gửi tin nhắn tiếp tục phiên vào %d thread", len(threadsToContinue)), time.Since(start))
     // Sau khi xong:
     LogSwitchEvent(timelineDir, "DISPATCH_DONE", fmt.Sprintf("Gửi tiếp tục phiên hoàn tất: %d/%d thành công", contSummary.SuccessCount, contSummary.TotalThreads), time.Since(start))
     ```
   - **Tại cuối hàm (Trước khi return kết quả thành công):**
     ```go
     LogSwitchEvent(timelineDir, "SWITCH_SUCCESS", fmt.Sprintf("Chuyển sang %s thành công hoàn tất", normTarget), time.Since(start))
     ```
   - **Khi gặp lỗi thất bại tại bất kỳ bước nào:**
     ```go
     LogSwitchEvent(timelineDir, "SWITCH_FAILED", fmt.Sprintf("Thất bại: %v", err), time.Since(start))
     ```

---

### BƯỚC 4: KÍCH HOẠT DỌN DẸP 7 NGÀY KHI APP KHỞI ĐỘNG XONG TRONG `internal/ui/app.go`

Trong hàm `NewAppWithFyne`, cập nhật goroutine dọn dẹp nền (đã có từ Phase 5) để dọn đồng thời cả `quota_log` và `switch_timeline`:

```go
// Tự động dọn dẹp các log cũ hơn 7 ngày trong goroutine nền nhẹ
go func() {
    quotaDir := filepath.Join(logsDir, "quota_log")
    _, _ = telemetry.PruneDailyQuotaLogs(quotaDir, telemetry.DefaultQuotaLogRetentionDays)

    timelineDir := filepath.Join(logsDir, "switch_timeline")
    _, _ = manager.PruneDailySwitchTimelineLogs(timelineDir, manager.DefaultSwitchTimelineRetentionDays)
}()
```

---

## IV. MA TRẬN KIỂM THỬ TỰ ĐỘNG BẮT BUỘC CHO CODER (TEST MATRIX)

Tạo file `internal/manager/timeline_test.go`:

1. **`TestPruneDailySwitchTimelineLogs_UnderLimit`:**
   - Tạo thư mục tạm với 3 file log: `2026-10-19_switch_timeline.log`, `2026-10-20_switch_timeline.log`, `2026-10-21_switch_timeline.log`.
   - Gọi `PruneDailySwitchTimelineLogs(dir, 7)`.
   - Kết quả: `deletedCount == 0`, cả 3 file vẫn còn nguyên.
2. **`TestPruneDailySwitchTimelineLogs_OverLimit`:**
   - Tạo thư mục tạm với 10 file log từ `2026-10-01_...` đến `2026-10-10_...`.
   - Gọi `PruneDailySwitchTimelineLogs(dir, 7)`.
   - Kết quả:
     - `deletedCount == 3`.
     - 7 file mới nhất (`2026-10-04` đến `2026-10-10`) tồn tại.
     - 3 file cũ nhất (`2026-10-01`, `2026-10-02`, `2026-10-03`) bị xóa.
3. **`TestLogSwitchEvent_CreatesDailyFile`:**
   - Gọi `LogSwitchEvent(tempDir, "TEST_EVENT", "Test message", 150*time.Millisecond)`.
   - Kiểm tra file `YYYY-MM-DD_switch_timeline.log` sinh ra trong `tempDir`, có chứa chuỗi `[TEST_EVENT]` và `Test message`.
4. **`TestSwitchAccount_WritesDailyTimeline`:**
   - Chạy `SwitchAccount` trong test environment.
   - Kiểm tra thư mục `logs/switch_timeline/` có sinh ra file log của ngày hôm nay với đầy đủ các mốc `SWITCH_START`, `SNAPSHOT_DONE`, `SWITCH_SUCCESS`.

---

## V. TIÊU CHÍ NGHIỆM THU CUỐI CÙNG (FINAL ACCEPTANCE CRITERIA)

1. `go test ./...` chạy PASS 100% không có bất kỳ lỗi nào.
2. Thư mục `logs/switch_timeline/` được tạo, file log cũ `2026-10-17_switch_timeline.log` nằm gọn trong thư mục này.
3. Khi thực hiện chuyển tài khoản, các mốc thời gian được ghi vào file `logs/switch_timeline/YYYY-MM-DD_switch_timeline.log`.
4. Thư mục `logs/` hoàn toàn không còn bất kỳ file log mồ côi nào trôi nổi ở ngoài gốc.
