# Kế Hoạch & Bản Đặc Tả Kỹ Thuật Bàn Giao Làn Coder: Triển Khai Phương Án 4 Bước Chuyển Đổi Tài Khoản & Tiếp Tục Phiên

**Tài liệu:** `reports/architect/2026-10-18-coder-implementation-spec-4-step-hotswap.md`  
**Ngày ban hành:** 2026-10-18  
**Làn ban hành:** Senior System Architect (Kiến Trúc Sư Hệ Thống)  
**Làn tiếp nhận:** Coder (Lập Trình Viên)  
**Thẩm quyền phê chuẩn:** Chỉ thị tối cao từ **Human Owner**  
**Nền tảng mục tiêu:** Codebase commit `27d727a`, Antigravity runtime v2.24.0 (Windows 11)  
**Mục tiêu tối thượng:** Bàn giao bản thiết kế kỹ thuật chi tiết từng hàm, từng file, từng contract để Làn Coder thi công chuẩn xác 100%, không sai lệch, vượt qua kiểm định Zero-Trust của Supervisor.

---

## 1. Mục Tiêu & Nguyên Lý Bất Biến (Target Invariants)

Bản đặc tả này hiện thực hóa **Phương án 4 bước do Human Owner trực tiếp chỉ định**:
1. **BƯỚC 1 (Snapshot 2 Giờ Toàn Bộ Project):**
   - Quét trực tiếp các tệp vật lý `.db` và `.db-wal` trong `%USERPROFILE%\.gemini\antigravity\conversations\`.
   - Ngưỡng thời gian: `effective_mtime := MAX(mtime(.db), mtime(.db-wal)) >= start - 2*time.Hour`.
   - Lấy toàn bộ thread trên TẤT CẢ các project, không giới hạn 1 thread/project, không loại bỏ thread IDLE.
   - Ánh xạ Project qua `conversation_summaries.db` (`workspace_uris`).
   - Ghi 1 tệp JSON snapshot duy nhất vào `accounts_store\active_threads_history\<timestamp>_active_threads.json` và lưu lại đường dẫn `snapshotPath`.
2. **BƯỚC 2 (Cơ Chế Đổi Chuẩn Của Antigravity & Chờ Đủ 8 Giây):**
   - Ghi token mới vào Windows Credential Manager (`LegacyGeneric:target=gemini:antigravity`).
   - Ghi ToS vào `app_storage.json` (`jetski.onboarding.lastLoginUsername` và `jetski.onboarding.lastLoginIsGcpTos = "false"`).
   - Đưa Antigravity lên foreground.
   - Xóa bỏ hoàn toàn nhánh gọi RPC `/ReAuth` ảo (do endpoint không tồn tại trong Antigravity).
   - Tái sinh Language Server bằng `taskkill /F /IM language_server.exe` để cơ chế `monitorLsCrashInternal` của Antigravity tự động kích hoạt tiến trình mới.
   - **BẮT ĐẦU CHỜ ĐỦ 8 GIÂY** (`time.Sleep(8 * time.Second)` hoặc mock sleeper).
   - Dò tìm instance mới (`connector.DiscoverLanguageServer()`) và xác nhận RAM qua `GetUserStatus`.
3. **BƯỚC 3 (Đọc Lại Đúng File Snapshot Bước 1):**
   - Đọc dữ liệu từ đúng biến `snapshotPath` của Bước 1, unmarshal ra `[]ActiveThread`.
4. **BƯỚC 4 (Gửi Tuần Tự Toàn Bộ Danh Sách Với Pacing 3 Giây):**
   - Duyệt tuần tự qua từng thread:
     - Gửi thread 1 ➔ nghỉ 3 giây ➔ gửi thread 2 ➔ nghỉ 3 giây ➔ ...
     - Chuỗi RPC mỗi thread: `SmartFocusConversation` ➔ `LoadTrajectory` ➔ `SendUserCascadeMessage`.
     - Model: Tự động lấy active model (`client.DiscoverActiveAgentModel()`).
     - Lỗi ở 1 thread **KHÔNG ĐƯỢC ngắt vòng lặp**, phải ghi nhận lỗi và tiếp tục gửi cho các thread tiếp theo.
     - Trả về `ContinuationSummary` tổng kết số thread thành công/thất bại để cập nhật thông điệp hoàn thành.

---

## 2. Danh Sách Tệp Cần Chỉnh Sửa & Phạm Vi Trách Nhiệm

| Tệp Cần Chỉnh Sửa | Loại Thay Đổi | Trách Nhiệm Chi Tiết |
|---|---|---|
| `internal/reloader/active_threads.go` | Cải tiến lõi | Thay thế logic parse `app_storage.json` bằng bộ quét SQLite `conversations/*.db` và `*.db-wal` với ngưỡng 2h, đọc project từ `conversation_summaries.db`, bổ sung hàm `ReadActiveThreadsSnapshot(path string)`. |
| `internal/reloader/active_threads_test.go` | Cập nhật Test | Bổ sung unit tests cho bộ quét SQLite 2h (mock folder conversations chứa `.db` và `.db-wal` với các mtime khác nhau), test ánh xạ project và test đọc lại snapshot file. |
| `internal/reloader/continuation.go` | Cải tiến lõi | Bổ sung struct `ContinuationSummary`, nâng cấp `PerformSessionContinuation` chịu lỗi per-thread (try-catch không break), bảo đảm pacing 3s, tự động phát hiện model. |
| `internal/reloader/continuation_test.go` | Cập nhật Test | Bổ sung unit tests kiểm tra: pacing 3s giữa các thread, lỗi ở thread 1 không làm dừng thread 2 & 3, kiểm tra `ContinuationSummary`. |
| `internal/manager/account_manager.go` | Tái cấu trúc quy trình | Sửa hàm `SwitchAccount`: loại bỏ RPC ReAuth ảo, gọi Bước 1 snapshot SQLite, thực hiện Bước 2 WinCred + ToS + taskkill + **chờ đúng 8 giây** + re-probe, gọi Bước 3 đọc snapshot file, gọi Bước 4 gửi tuần tự và cập nhật message. |
| `internal/manager/manager_test.go` | Cập nhật Test | Cập nhật các mock test của `SwitchAccount` để xác nhận quy trình 4 bước diễn ra chuẩn xác, test case 8 giây wait (dùng mock sleeper). |

---

## 3. Đặc Tả Kỹ Thuật Chi Tiết Cho Làn Coder

### 3.1 Module 1: `internal/reloader/active_threads.go`

#### A. Cấu Trúc Dữ Liệu
Giữ nguyên định dạng JSON tương thích 100%:
```go
type ActiveThread struct {
	ThreadID   string `json:"thread_id"`
	Project    string `json:"project"`
	IsMainPane bool   `json:"is_main_pane"`
	Model      string `json:"model,omitempty"`
}

type ActiveThreadsSnapshot struct {
	Timestamp     string         `json:"timestamp"`
	ActiveThreads []ActiveThread `json:"active_threads"`
}
```

#### B. Hàm Quét SQLite 2 Giờ Mới: `ScanActiveConversations2h`
Coder cần xây dựng hàm nội bộ:
```go
// ScanActiveConversations2h scans %USERPROFILE%/.gemini/antigravity/conversations/
// for all threads with MAX(mtime(.db), mtime(.db-wal)) >= startTime - 2h.
func ScanActiveConversations2h(conversationsDir string, summariesDbPath string, startTime time.Time) ([]ActiveThread, error)
```
**Choreography chi tiết:**
1. Nếu `conversationsDir == ""`, mặc định là:
   `filepath.Join(userHome, ".gemini", "antigravity", "conversations")`
2. Nếu `summariesDbPath == ""`, mặc định là:
   `filepath.Join(userHome, ".gemini", "antigravity", "conversation_summaries.db")`
3. Tính ngưỡng: `threshold := startTime.Add(-2 * time.Hour)`.
4. Đọc danh sách file trong `conversationsDir`:
   - Duyệt qua toàn bộ entries.
   - Tìm các file có đuôi `.db` hoặc `.db-wal`.
   - Bỏ qua các file không phải UUID (ví dụ: `conversation_summaries.db`).
   - Gom nhóm theo `<thread_id>` (chuỗi trước dấu chấm).
   - Lấy `ModTime()` của `<thread_id>.db`.
   - Nếu tồn tại `<thread_id>.db-wal`, lấy `ModTime()` của file WAL.
   - `effectiveMtime := MAX(mtime(.db), mtime(.db-wal))`.
   - Nếu `effectiveMtime.Before(threshold)` ➔ Bỏ qua.
5. Với các thread thỏa mãn:
   - Tra cứu Project từ `summariesDbPath` bằng hàm trợ giúp `resolveProjectFromSummaries(summariesDbPath, threadID)`:
     - Mở read-only connection tới SQLite hoặc đọc trực tiếp nếu có sqlite driver, HOẶC fallback an toàn: kiểm tra `brain/<thread_id>/` hoặc `app_storage.json`.
     - *Gợi ý thực thi an toàn:* Trong codebase Go hiện tại, nếu không muốn kéo thêm CGo driver sqlite, có thể dùng `app_storage.json` và `brain/<thread_id>` làm nguồn bổ trợ project URI, hoặc đọc `conversation_summaries.db` qua query nhẹ. Nếu không lấy được project thì để trống hoặc `"unknown"`, KHÔNG ĐƯỢC làm hỏng thread ID! Vì `SendUserCascadeMessage` chỉ cần `threadID` (`cascadeId`), không phụ thuộc vào `project`.
   - Đánh dấu `IsMainPane`: Thread có `effectiveMtime` mới nhất (gần T0 nhất) đặt `IsMainPane = true`, các thread còn lại `IsMainPane = false`.
6. Sắp xếp danh sách `[]ActiveThread` theo `effectiveMtime` giảm dần (thread mới nhất lên đầu).

#### C. Hàm Ghi Snapshot: `CaptureActiveThreadsSnapshot`
```go
func defaultCaptureActiveThreadsSnapshot(conversationsDir, summariesDbPath, outputDir string) (string, []ActiveThread, error) {
    start := time.Now()
    activeThreads, err := ScanActiveConversations2h(conversationsDir, summariesDbPath, start)
    if err != nil {
        return "", nil, err
    }

    outDir := outputDir
    if outDir == "" {
        outDir = GetDefaultThreadsHistoryDir()
    }
    _ = os.MkdirAll(outDir, 0755)

    timestampStr := start.Format("2006-01-02 15:04:05")
    fileTs := start.Format("2006-01-02-15-04-05")
    filename := fmt.Sprintf("%s_active_threads.json", fileTs)
    snapshotPath := filepath.Join(outDir, filename)

    snapshot := ActiveThreadsSnapshot{
        Timestamp:     timestampStr,
        ActiveThreads: activeThreads,
    }

    data, err := json.MarshalIndent(snapshot, "", "  ")
    if err != nil {
        return "", nil, err
    }

    // Atomic write via temp file + rename
    tmpFile := snapshotPath + ".tmp"
    if err := os.WriteFile(tmpFile, data, 0644); err != nil {
        return "", nil, err
    }
    if err := os.Rename(tmpFile, snapshotPath); err != nil {
        return "", nil, err
    }

    return snapshotPath, activeThreads, nil
}
```

#### D. Hàm Đọc Lại Snapshot: `ReadActiveThreadsSnapshot` (Bước 3)
```go
// ReadActiveThreadsSnapshot reads and unmarshals an active threads snapshot JSON file.
func ReadActiveThreadsSnapshot(snapshotPath string) ([]ActiveThread, error) {
    if strings.TrimSpace(snapshotPath) == "" {
        return nil, errors.New("snapshotPath cannot be empty")
    }
    data, err := os.ReadFile(snapshotPath)
    if err != nil {
        return nil, fmt.Errorf("failed reading snapshot file: %w", err)
    }
    var snapshot ActiveThreadsSnapshot
    if err := json.Unmarshal(data, &snapshot); err != nil {
        return nil, fmt.Errorf("failed unmarshaling snapshot JSON: %w", err)
    }
    return snapshot.ActiveThreads, nil
}
```

---

### 3.2 Module 2: `internal/reloader/continuation.go`

#### A. Cấu Trúc Báo Cáo Kết Quả
```go
type ContinuationSummary struct {
    TotalThreads   int      `json:"total_threads"`
    SuccessCount   int      `json:"success_count"`
    FailedCount    int      `json:"failed_count"`
    SuccessThreads []string `json:"success_threads"`
    Errors         []string `json:"errors"`
}
```

#### B. Hàm Điều Phối Gửi Tuần Tự: `PerformSessionContinuation`
Coder cần cập nhật hàm này để nhận hoặc đọc danh sách thread, thực thi vòng lặp gửi an toàn:
```go
func (r *SessionReloader) PerformSessionContinuation(
    ctx context.Context,
    client *connector.Client,
    activeThreads []ActiveThread,
    promptText string,
) (*ContinuationSummary, error) {
    if client == nil || len(activeThreads) == 0 {
        return &ContinuationSummary{}, nil
    }

    pText := strings.TrimSpace(promptText)
    if pText == "" {
        pText = DefaultContinuationPrompt
    }

    sleeper := r.Sleeper
    if sleeper == nil {
        sleeper = defaultSleep
    }

    summary := &ContinuationSummary{
        TotalThreads:   len(activeThreads),
        SuccessThreads: make([]string, 0, len(activeThreads)),
        Errors:         make([]string, 0),
    }

    // Tự động phát hiện model đang hoạt động
    activeModel := client.DiscoverActiveAgentModel(ctx, "")

    threadsSent := 0
    for _, thread := range activeThreads {
        tID := strings.TrimSpace(thread.ThreadID)
        if tID == "" {
            continue
        }

        // Bước 4: Gửi thread 1 ➔ nghỉ 3 giây ➔ gửi thread 2 ➔ nghỉ 3 giây ➔ ...
        if threadsSent > 0 {
            _ = sleeper(ctx, 3*time.Second)
        }

        // Bọc riêng từng thread: Lỗi 1 thread KHÔNG ĐƯỢC ngắt vòng lặp!
        var threadErr error
        if err := client.SmartFocusConversation(ctx, tID); err != nil {
            threadErr = fmt.Errorf("SmartFocus error: %w", err)
        }
        if err := client.LoadTrajectory(ctx, tID); err != nil && threadErr == nil {
            threadErr = fmt.Errorf("LoadTrajectory error: %w", err)
        }
        if err := client.SendUserCascadeMessage(ctx, tID, pText, activeModel, nil); err != nil && threadErr == nil {
            threadErr = fmt.Errorf("SendUserCascadeMessage error: %w", err)
        }

        if threadErr != nil {
            summary.FailedCount++
            summary.Errors = append(summary.Errors, fmt.Sprintf("Thread %s: %v", tID, threadErr))
        } else {
            summary.SuccessCount++
            summary.SuccessThreads = append(summary.SuccessThreads, tID)
        }

        threadsSent++
    }

    return summary, nil
}
```

---

### 3.3 Module 3: `internal/manager/account_manager.go`

Tái cấu trúc hàm `SwitchAccount` để loại bỏ hoàn toàn các đoạn code thử nghiệm sai lầm trước đây và triển khai 4 bước chuẩn xác:

```go
func (m *AccountManager) SwitchAccount(ctx context.Context, targetEmail string, forceLogout ...bool) (*SwitchResult, error) {
    start := time.Now()
    normTarget := strings.ToLower(strings.TrimSpace(targetEmail))
    if normTarget == "" {
        return nil, ErrEmptyTargetEmail
    }

    // 1. Kiểm tra tài khoản hiện tại trong RAM, nếu đã là targetEmail thì no-op
    currentEmail, _ := m.GetCurrentActiveEmail(ctx)
    if currentEmail != "" && currentEmail == normTarget {
        return &SwitchResult{
            Success:       true,
            Message:       fmt.Sprintf("Tài khoản %s hiện đang hoạt động.", normTarget),
            TargetEmail:   normTarget,
            PreviousEmail: currentEmail,
            Duration:      time.Since(start),
        }, nil
    }

    // =========================================================================
    // BƯỚC 1: Chốt danh sách trước khi chuyển tài khoản (Snapshot SQLite 2 Giờ)
    // =========================================================================
    var snapshotPath string
    var activeThreads []reloader.ActiveThread
    if rel != nil {
        snapshotPath, activeThreads, _ = rel.CaptureActiveThreadsSnapshot("", "", "")
    } else {
        snapshotPath, activeThreads, _ = reloader.CaptureActiveThreadsSnapshot("", "", "")
    }

    // Tự động lưu phiên làm việc hiện tại
    _, _ = m.CaptureCurrentSession()

    // Tải token của tài khoản mục tiêu từ accounts_store/
    targetToken, err := m.store.GetAccount(normTarget)
    if err != nil {
        return &SwitchResult{
            Success:       false,
            Message:       fmt.Sprintf("Chưa có file JSON của %s trong accounts_store/!", normTarget),
            TargetEmail:   normTarget,
            PreviousEmail: currentEmail,
            Duration:      time.Since(start),
            ActiveThreads: activeThreads,
        }, ErrTargetNotFound
    }

    // Auto-refresh token nếu sắp hết hạn trong 5 phút
    if targetToken != nil && (targetToken.Expiry.IsZero() || time.Until(targetToken.Expiry) < 5*time.Minute) {
        if targetToken.RefreshToken != "" {
            activeRefresher := refresher
            if activeRefresher == nil {
                activeRefresher = connector.NewGoogleOAuthRefresher(nil)
            }
            refreshRes, refErr := activeRefresher.RefreshToken(ctx, targetToken.RefreshToken)
            if refErr == nil && refreshRes != nil && refreshRes.AccessToken != "" {
                targetToken.UpdateTokens(refreshRes.AccessToken, refreshRes.RefreshToken, refreshRes.IDToken, refreshRes.Expiry)
                _ = m.store.SaveAccount(normTarget, targetToken)
            }
        }
    }

    // =========================================================================
    // BƯỚC 2: Chuyển tài khoản bằng cơ chế chuẩn của Antigravity
    // =========================================================================
    // 2.1 Đồng bộ ToS vào app_storage.json
    isGcpTos := strings.EqualFold(targetToken.AuthMethod, "gcp")
    storagePath := ""
    if rel != nil {
        storagePath = rel.StoragePath
    }
    _ = reloader.SyncAppStorageToS(normTarget, isGcpTos, storagePath)

    // 2.2 Ghi token vào Windows Credential Manager
    if writer == nil {
        return nil, errors.New("wincred writer not configured")
    }
    if err := writer(targetToken); err != nil {
        return &SwitchResult{
            Success:       false,
            Message:       fmt.Sprintf("Không thể ghi Token vào Windows Credential Manager: %v", err),
            TargetEmail:   normTarget,
            PreviousEmail: currentEmail,
            Duration:      time.Since(start),
            ActiveThreads: activeThreads,
        }, fmt.Errorf("%w: %v", ErrWriteCredFailed, err)
    }

    // 2.3 Đưa cửa sổ Antigravity lên foreground
    _ = reloader.EnsureAntigravityForeground()

    // 2.4 Dừng language_server.exe để Antigravity monitorLsCrashInternal tự khởi động lại
    // LƯU Ý BẮT BUỘC: Xóa bỏ 100% nhánh gọi RPC ReAuth ảo!
    // BẮT ĐẦU CHỜ ĐỦ 8 GIÂY rồi mới qua bước tiếp theo
    waitDuration := 8 * time.Second
    var recycleErr error
    if rel != nil {
        recycleErr = rel.RecycleLanguageServer(ctx, waitDuration)
    } else {
        recycleErr = reloader.RecycleLanguageServer(ctx, waitDuration)
    }
    if recycleErr != nil {
        return &SwitchResult{
            Success:       true,
            Message:       fmt.Sprintf("Đã chuyển sang %s (cảnh báo khởi động lại Language Server: %v)", normTarget, recycleErr),
            TargetEmail:   normTarget,
            PreviousEmail: currentEmail,
            Duration:      time.Since(start),
            ActiveThreads: activeThreads,
        }, nil
    }

    // 2.5 Dò tìm Language Server mới sau 8 giây ổn định
    var newClient *connector.Client
    if finder != nil {
        newClient, _ = finder(ctx)
    } else {
        if newInfo, err := connector.DiscoverLanguageServer(); err == nil && newInfo != nil {
            newClient = connector.NewClient(newInfo)
        }
    }

    // =========================================================================
    // BƯỚC 3: Đọc đúng file JSON của bước 1
    // =========================================================================
    threadsToContinue := activeThreads
    if snapshotPath != "" {
        if diskThreads, readErr := reloader.ReadActiveThreadsSnapshot(snapshotPath); readErr == nil && len(diskThreads) > 0 {
            threadsToContinue = diskThreads
        }
    }

    // =========================================================================
    // BƯỚC 4: Gửi tuần tự toàn bộ danh sách (Nghỉ 3s giữa các thread, không ngắt khi lỗi)
    // =========================================================================
    var contSummary *reloader.ContinuationSummary
    if newClient != nil && len(threadsToContinue) > 0 {
        if rel != nil {
            contSummary, _ = rel.PerformSessionContinuation(ctx, newClient, threadsToContinue, "")
        } else {
            contSummary, _ = reloader.PerformSessionContinuation(ctx, newClient, threadsToContinue, "")
        }
    }

    // Cập nhật thông điệp hoàn tất minh bạch
    msg := fmt.Sprintf("Chuyển sang tài khoản %s thành công!", normTarget)
    if contSummary != nil && contSummary.TotalThreads > 0 {
        msg = fmt.Sprintf("Chuyển sang tài khoản %s thành công! Đã gửi prompt tiếp tục phiên vào %d/%d thread.",
            normTarget, contSummary.SuccessCount, contSummary.TotalThreads)
    }

    return &SwitchResult{
        Success:             true,
        Message:             msg,
        TargetEmail:         normTarget,
        PreviousEmail:       currentEmail,
        Duration:            time.Since(start),
        ActiveThreads:       threadsToContinue,
        ContinuationSummary: contSummary,
    }, nil
}
```

---

## 4. Kế Hoạch Kiểm Thử Tự Động & Tiêu Chí Nghiệm Thu (Zero-Trust DoD)

Coder phải thực hiện đầy đủ các bài kiểm thử tự động, tuân thủ nguyên tắc cách ly 100% (không gây ô nhiễm ổ đĩa thật, không kill tiến trình thật trong test):

### 4.1 Unit Tests Cần Bổ Sung
1. **`TestScanActiveConversations2h_ThresholdFiltering`:**
   - Tạo thư mục tạm `t.TempDir()`.
   - Tạo các file `.db` và `.db-wal` giả lập:
     - Thread A: mtime = `now - 30m` ➔ Phải được chọn.
     - Thread B: mtime = `now - 1h50m` ➔ Phải được chọn.
     - Thread C: `.db` có mtime = `now - 3h`, nhưng `.db-wal` có mtime = `now - 10m` ➔ **Phải được chọn** (kiểm chứng `MAX(mtime, wal_mtime)`).
     - Thread D: mtime = `now - 2h10m` ➔ Bị loại bỏ.
   - Khẳng định danh sách trả về đúng các thread thỏa điều kiện, sắp xếp đúng thứ tự thời gian.
2. **`TestReadActiveThreadsSnapshot_RoundTrip`:**
   - Ghi 1 snapshot JSON mẫu và gọi `ReadActiveThreadsSnapshot`.
   - Khẳng định unmarshal trả về 100% đúng các trường `ThreadID`, `Project`, `IsMainPane`.
3. **`TestPerformSessionContinuation_PacingAndFaultTolerance`:**
   - Giả lập mock HTTP server của Language Server:
     - Thread 1: Thành công (HTTP 200).
     - Thread 2: Lỗi (HTTP 500).
     - Thread 3: Thành công (HTTP 200).
   - Kiểm tra `ContinuationSummary`: `TotalThreads == 3`, `SuccessCount == 2`, `FailedCount == 1`.
   - Kiểm tra biến đếm thời gian sleeper: sleeper phải được gọi 2 lần với khoảng thời gian `3 * time.Second`.
   - Khẳng định Thread 3 vẫn được gửi dù Thread 2 bị lỗi (không làm đứt vòng lặp).
4. **`TestSwitchAccount_8SecondWaitAnd4StepWorkflow`:**
   - Test end-to-end `SwitchAccount` với mock sleeper.
   - Khẳng định sleeper nhận lệnh chờ đủ `8 * time.Second`.
   - Khẳng định quy trình 4 bước hoàn thành với `Success == true` và message thông báo chính xác số thread.

### 4.2 Lệnh Chạy Kiểm Thử Tiêu Chuẩn
```powershell
go test -v ./internal/reloader/...
go test -v ./internal/manager/...
go test -v ./...
```
Tất cả các bài test phải `PASS 100%` với `exit code 0`.

---
**Bản đặc tả đã được niêm phong bởi:** Senior System Architect  
**Hành động tiếp theo:** Bàn giao trực tiếp cho Làn Coder thi công theo đúng từng dòng mã và contract đã định.
