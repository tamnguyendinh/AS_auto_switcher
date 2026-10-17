# Root Cause Analysis (RCA) Báo Cáo Chuyên Sâu: Sự Cố Mất Cơ Chế Gửi Prompt Tiếp Tục Phiên Vào Thread Khi Đổi Tài Khoản

**Tài liệu tham chiếu:** `reports/architect/2026-10-17-session-continuation-missing-rca.md`  
**Ngày thực hiện:** 2026-10-17  
**Làn đảm nhiệm:** Senior System Architect (Kiến Trúc Sư Hệ Thống)  
**Cơ quan chỉ đạo:** CEO Agent (Thread ID: `01a14378-4a08-77c1-88d5-666ad8c3592d`)  
**Thẩm quyền & Phê duyệt:** Chỉ đạo trực tiếp từ **Human Owner** (Báo cáo sự cố: *"Khi thực hiện đổi tài khoản, quá trình đổi diễn ra rất nhanh nhưng hệ thống hoàn toàn KHÔNG gửi tin nhắn tiếp tục phiên: 'Continue if you still have unfinished work in progress. If all tasks are already completed, reply with: Currently there is no work to do.' vào các thread gần nhất đang hoạt động như thiết kế ban đầu (P0-04)"*).  
**Phạm vi:** Thẩm tra toàn diện codebase `internal/manager/`, `internal/reloader/`, `internal/connector/`, đối soát 100% với Python SSOT (`core/state_resolver.py`, `core/session_reloader.py`, `core/api_connector.py`), đối soát runtime thực tế Antigravity v2.24.0 (PID 11116) và lịch sử snapshot tại `accounts_store/active_threads_history/`.  
**Trạng thái thẩm tra:** HOÀN TẤT & ĐỐI SOÁT THỰC NGHIỆM 100% — XÁC ĐỊNH CHÍNH XÁC NGUYÊN NHÂN GỐC RỄ & PHƯƠNG ÁN KHẮC PHỤC DỨT ĐIỂM.

---

## 1. Tóm tắt sự cố & Hiện tượng quan sát (Incident Summary)

### 1.1 Hiện tượng quan sát từ Human Owner
- Khi thực hiện thao tác đổi tài khoản Antigravity (qua GUI hoặc CLI), quá trình đổi tài khoản diễn ra rất nhanh chóng và thành công (tài khoản đích đã được ghi vào Windows Credential Manager, Antigravity Language Server nhận diện đúng email mới).
- **Tuy nhiên**, hệ thống hoàn toàn **KHÔNG** gửi tin nhắn tiếp tục phiên (continuation prompt):
  > `"Continue if you still have unfinished work in progress. If all tasks are already completed, reply with: Currently there is no work to do."`
  vào các thread hội thoại gần nhất đang hoạt động, trái ngược với thiết kế hợp đồng P0-04.
- Kết quả là các tab hội thoại của người dùng đang chạy dở bị dừng lại sau khi đổi tài khoản, không tự động tiếp tục công việc như kỳ vọng.

---

## 2. Bằng chứng đối soát Thực nghiệm & Runtime (Empirical Evidence)

### 2.1 Bằng chứng 1: Tệp Snapshot Rỗng Trong Lịch Sử Chuyển Đổi Thực Tế
Kiểm tra thư mục `accounts_store/active_threads_history/` ngay tại thời điểm xảy ra sự cố hôm nay (2026-10-17 19:22:50 đến 19:22:54):
```text
Tên tệp                                  Kích thước (Bytes)
2026-10-17-19-22-50_active_threads.json  65
2026-10-17-19-22-51_active_threads.json  65
2026-10-17-19-22-52_active_threads.json  65
2026-10-17-19-22-53_active_threads.json  65
2026-10-17-19-22-54_active_threads.json  65
```
Nội dung bên trong toàn bộ các tệp 65 bytes này:
```json
{
  "timestamp": "2026-10-17 19:22:50",
  "active_threads": []
}
```
👉 **Phát hiện chí tử 1:** Tại thời điểm chuyển tài khoản, hàm `CaptureActiveThreadsSnapshot` đã chụp ra danh sách thread **hoàn toàn rỗng (`active_threads: []`)**!

### 2.2 Bằng chứng 2: Tệp Snapshot Không Rỗng Lại Chứa Toàn Thread Cũ Từ Quá Khứ
Kiểm tra tệp snapshot lúc 19:36:20 (`2026-10-17-19-36-20_active_threads.json`):
```json
{
  "timestamp": "2026-10-17 19:36:20",
  "active_threads": [
    {
      "thread_id": "9dded9d3-6ccf-4889-b888-37586aa2577c",
      "project": "...",
      "is_main_pane": true
    },
    {
      "thread_id": "cf87f55e-144b-4a6d-b6f3-2d652e9e90bb",
      "project": "...",
      "is_main_pane": false
    },
    {
      "thread_id": "e7886c1c-6c90-4a79-bc28-5119be04f7d7",
      "project": "...",
      "is_main_pane": false
    }
  ]
}
```
Thực nghiệm kiểm tra thời gian ghi cuối cùng (`LastWriteTime`) của 3 thư mục thread này trong `~/.gemini/antigravity/brain/`:
- `9dded9d3-6ccf-4889-b888-37586aa2577c`: **10/16/2026 8:27 AM** (Hôm qua)
- `cf87f55e-144b-4a6d-b6f3-2d652e9e90bb`: **10/16/2026 8:26 AM** (Hôm qua)
- `e7886c1c-6c90-4a79-bc28-5119be04f7d7`: **09/13/2026 11:57 PM** (1 tháng trước!)

Trong khi đó, thread mà Human Owner đang thực sự mở và làm việc trong phiên hiện tại hôm nay (10/17):
- `c355eae2-8aab-4b38-92c9-57617be13bd2`: **10/17/2026 7:51 PM** (`transcript.jsonl` kích thước 164 KB, `transcript_full.jsonl` kích thước 253 KB).
- `e0e27623-c2c1-48b4-9846-3a339f2aeb5a`: **10/17/2026 7:31 PM**
- `a5cafb33-a76a-4fff-9d8f-f36a6ee80472`: **10/17/2026 7:28 PM**

👉 **Phát hiện chí tử 2:** Thread in-flight đang hoạt động hôm nay hoàn toàn **KHÔNG ĐƯỢC CHỤP**. Nếu có gửi prompt thì hệ thống lại gửi nhầm vào các thread cũ từ hôm qua hoặc 1 tháng trước, khiến giao diện đang mở của người dùng không hề nhận được bất kỳ tin nhắn nào!

---

## 3. Phân tích Chi tiết Từng Bước Mã Nguồn (Deep-Dive Code Analysis)

### 3.1 Thẩm tra Bước Chụp Snapshot trong `internal/manager/account_manager.go`
Xem xét dòng 413-419 của `internal/manager/account_manager.go`:
```go
// 2. Snapshot active threads BEFORE recycling to preserve in-flight conversation state (P0-04)
var activeThreads []reloader.ActiveThread
if rel != nil {
    _, activeThreads, _ = rel.CaptureActiveThreadsSnapshot("", "", "")
} else {
    _, activeThreads, _ = reloader.CaptureActiveThreadsSnapshot("", "", "")
}
```

Có 3 vấn đề kỹ thuật nghiêm trọng tại đây:
1. **Lỗi truyền tham số rỗng:** `rel.CaptureActiveThreadsSnapshot("", "", "")` truyền các chuỗi rỗng thay vì truyền `rel.StoragePath` và `rel.BrainDir`. Khi chạy trong môi trường kiểm thử hoặc cấu hình đường dẫn tùy biến, cấu hình của `SessionReloader` bị vô hiệu hóa hoàn toàn.
2. **Cơ chế đọc chỉ dựa vào `app_storage.json`:**
   - Trong `defaultCaptureActiveThreadsSnapshot` (`internal/reloader/active_threads.go`), code chỉ đọc `aux-pane-v2-session.conversationPanes` hoặc `aux-pane-session`.
   - Nhưng tệp `app_storage.json` của Antigravity/VS Code **không được ghi tức thời (buffered/unflushed)** trong lúc phiên làm việc đang chạy, mà chỉ được ghi định kỳ hoặc khi đóng cửa sổ.
   - Các thread mới mở hôm nay (`c355eae2...`, `a5cafb33...`) **chưa hề tồn tại trong `app_storage.json`** (đã đối soát thực tế bằng regex trong tệp `app_storage.json` hiện tại).
3. **Lỗi Parser không nhận diện cấu trúc Subagent Tab:**
   - Kiểm tra `app_storage.json` thực tế:
     ```json
     "tabs": [
       {
         "id": "subagent_convo__1a36346e-f1a4-4bd7-969b-27f83f15dd71",
         "content": {
           "type": "subagentView",
           "subagentView": {
             "childCascadeId": "1a36346e-f1a4-4bd7-969b-27f83f15dd71",
             "title": "Supervisor"
           }
         }
       }
     ]
     ```
   - Trong `active_threads.go:153-163`:
     ```go
     meta, _ := tab["metadata"].(map[string]interface{})
     tID := getMapString(meta, "cascadeId")
     if tID == "" { tID = getMapString(tab, "cascadeId") }
     if tID == "" { tID = getMapString(tab, "childCascadeId") }
     ```
     Code chỉ tìm ở tầng gốc `tab["childCascadeId"]` hoặc `tab["metadata"]`, hoàn toàn không parse `tab["content"]["subagentView"]["childCascadeId"]` hay tiền tố `subagent_convo__`.
   - Kết quả: `threadIDsInPane` bị rỗng. Khi rỗng, code lấy `cid = kID` (ID của Pane, ví dụ `403f41a7-0810-44d9-b875-1e08c4ed13a0`).
   - Sau đó kiểm tra `isDir(filepath.Join(bDir, cid))`: Vì `cid` là ID của Pane giao diện chứ không phải thư mục trong `brain`, hàm trả về false và loại bỏ!
   - Kết quả cuối cùng: `activeThreads` bị rỗng hoàn toàn (`len == 0`).

---

### 3.2 Thẩm tra Điều Kiện Gọi `PerformSessionContinuation` Trong `account_manager.go`
Xem xét dòng 551-557 của `internal/manager/account_manager.go`:
```go
// 9. In-Flight Session Continuation (Pure Native Connect-RPC: SmartFocus -> LoadTrajectory -> SendUserCascadeMessage) (P0-04)
if newClient != nil && len(activeThreads) > 0 {
    if rel != nil {
        _ = rel.PerformSessionContinuation(ctx, newClient, activeThreads, "")
    } else {
        _ = reloader.PerformSessionContinuation(ctx, newClient, activeThreads, "")
    }
}
```

- **Mắt xích bị đứt:** Điều kiện tiên quyết để hàm này được gọi là `len(activeThreads) > 0`.
- Do bước snapshot trước đó tạo ra `activeThreads` rỗng (`len == 0`), điều kiện `len(activeThreads) > 0` bị đánh giá là **`false`**.
- Do đó, **toàn bộ khối lệnh gửi Session Continuation bị bỏ qua 100%**! Hệ thống nhảy thẳng sang bước trả kết quả `SwitchResult`.

---

### 3.3 Thẩm tra Nhánh `hotSwapped == true` và Hiện Tượng "/ReAuth" Ảo
Xem xét dòng 497-520 của `account_manager.go`:
```go
var hotSwapped bool
var newClient *connector.Client

if client != nil {
    reauthCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
    reauthErr := client.TriggerReauth(reauthCtx)
    cancel()

    if reauthErr == nil {
        // Short polling (up to 5 attempts, 50ms intervals) to confirm new token loaded into RAM
        for attempt := 0; attempt < 5; attempt++ {
            time.Sleep(50 * time.Millisecond)
            status, statusErr := client.GetUserStatus(ctx)
            if statusErr == nil && status != nil && strings.EqualFold(strings.TrimSpace(status.Email), normTarget) {
                hotSwapped = true
                newClient = client
                break
            }
        }
    }
}
```

Thực nghiệm kiểm tra trực tiếp trên Language Server v2.24.0 đang chạy (PID 11116, Port 59678):
1. **Endpoint Connect-RPC `/ReAuth` không hề tồn tại:**
   ```bash
   POST http://127.0.0.1:59678/exa.language_server_pb.LanguageServerService/ReAuth
   -> HTTP 404 Not Found (404 page not found)
   ```
2. **Endpoint `/reauth` trả về HTTP 200 giả mạo (False Positive):**
   ```bash
   POST http://127.0.0.1:59678/reauth
   -> HTTP 200 OK (Content-Type: text/html - Trả về mã nguồn HTML của Webview SPA!)
   ```
   - Trong `internal/connector/api_client.go:232`:
     `if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent { return nil }`
     Hàm `TriggerReauth` thấy HTTP 200 thì tưởng là RPC thành công, nên trả về `reauthErr = nil`!
3. **Vòng lặp Polling 250ms thất bại:**
   - Sau đó, vòng lặp 5 lần kiểm tra `GetUserStatus` (tổng thời gian 250ms).
   - Vì Language Server không hề có cơ chế RPC ReAuth nên RAM của nó vẫn chứa email cũ.
   - Vòng lặp kết thúc mà `hotSwapped` vẫn là `false`, `newClient` vẫn là `nil`.
4. **Nhánh Fallback `!hotSwapped` kích hoạt:**
   - Hệ thống chuyển sang `rel.RecycleLanguageServer(ctx, 2*time.Second)`.
   - `RecycleLanguageServer` dùng `taskkill` diệt `language_server.exe` và đợi 2 giây để Antigravity crash monitor tự động respawn tiến trình mới.
   - Tiến trình mới khởi động lại, nạp token mới từ Windows Credential Manager.
   - `newClient` được phát hiện lại thành công (có PID mới, Port mới, CSRF mới).
   - Nhưng khi đến bước 9: `if newClient != nil && len(activeThreads) > 0`, do `activeThreads` đã rỗng từ bước 2, **Session Continuation vẫn không được kích hoạt**!

---

### 3.4 Thẩm tra Cơ Chế Nuốt Lỗi (Error Swallowing) Trong `continuation.go`
Xem xét dòng 48-65 của `internal/reloader/continuation.go`:
```go
threadsSent := 0
for _, thread := range activeThreads {
    tID := strings.TrimSpace(thread.ThreadID)
    if tID == "" {
        continue
    }

    if threadsSent > 0 {
        _ = sleeper(ctx, 3*time.Second)
    }

    _ = client.SmartFocusConversation(ctx, tID)
    _ = client.LoadTrajectory(ctx, tID)
    _ = client.SendUserCascadeMessage(ctx, tID, pText, "", nil)

    threadsSent++
}
```

Kiểm tra thực nghiệm phản hồi của Language Server với 3 hàm trên:
1. `SmartFocusConversation`:
   ```bash
   POST /exa.language_server_pb.LanguageServerService/SmartFocusConversation
   -> HTTP 500 Internal Server Error: {"code":"unknown","message":"extension server client not available"}
   ```
   Endpoint này đòi hỏi extension server bridge đang kết nối; nếu gọi ngay sau khi server vừa respawn thì thường xuyên trả về HTTP 500.
2. `LoadTrajectory`:
   ```bash
   POST /exa.language_server_pb.LanguageServerService/LoadTrajectory
   -> HTTP 200 OK: {}
   ```
3. `SendUserCascadeMessage`:
   Nếu gọi với Cascade ID không tồn tại hoặc model không hợp lệ:
   ```bash
   POST /exa.language_server_pb.LanguageServerService/SendUserCascadeMessage
   -> HTTP 500 Internal Server Error: {"code":"unknown","message":"trajectory not found: ..."}
   ```

🚨 **Vấn đề cực kỳ nghiêm trọng:**
Cả 3 lệnh gọi RPC đều dùng `_ = ...` để **nuốt lỗi hoàn toàn**. Ngay cả ở tầng `account_manager.go:553`:
```go
_ = rel.PerformSessionContinuation(ctx, newClient, activeThreads, "")
```
cũng nuốt lỗi bằng `_ = ...`!
Hệ thống không ghi bất kỳ một dòng log cảnh báo nào, không trả error, không có telemetry, biến tầng Continuation thành một "hố đen thông tin" (silent failure), che giấu mọi lỗi thất bại.

---

## 4. Đối soát Chi tiết với Python SSOT (Matrix Comparison vs Python SSOT)

Đối soát với `core/state_resolver.py`, `core/session_reloader.py`, và `core/api_connector.py`:

| Tiêu chí kỹ thuật | Python SSOT (`core/`) | Hiện trạng Go (`internal/`) | Mức độ nghiêm trọng |
|---|---|---|:---:|
| **Fallback khi Snapshot rỗng** | **CÓ:** `core/state_resolver.py:520-545`<br>`discoverer = discover_most_recent_cascade_id`<br>Nếu snapshot rỗng, tự động quét `brain` lấy thread gần nhất theo `mtime` của `transcript.jsonl`. | **KHÔNG CÓ:** Hoàn toàn bỏ sót hàm `DiscoverMostRecentCascadeID`. Khi snapshot rỗng, `activeThreads` rỗng luôn, bỏ qua gửi prompt. | **P0 (BLOCKER GỐC RỄ)** |
| **Quét Não theo Mtime (`brain_dir`)** | **CÓ:** `core/session_reloader.py:557-650`<br>Quét toàn bộ thư mục con trong `~/.gemini/antigravity/brain/`, sắp xếp theo `mtime` của `transcript.jsonl` và `transcript_full.jsonl`, đối soát layout keys. | **KHÔNG CÓ:** Chỉ đọc `app_storage.json`, không hề có logic quét thư mục não theo thời gian sửa đổi gần nhất. | **P0 (BLOCKER GỐC RỄ)** |
| **Nhận diện Tab Subagent / Background Task** | Có regex trích xuất linh hoạt, nhận diện các cấu trúc tab phong phú. | Chỉ đọc `tab["metadata"]` và `tab["cascadeId"]`, bỏ sót `content.subagentView.childCascadeId` và `subagent_convo__<uuid>`. | **P1 (NGHIÊM TRỌNG)** |
| **Truyền Tham Số Reloader** | Truyền đầy đủ `s_path`, `b_dir`, `out_dir` lấy từ instance cấu hình. | Hardcode `rel.CaptureActiveThreadsSnapshot("", "", "")`, bỏ qua `StoragePath` và `BrainDir`. | **P1 (NGHIÊM TRỌNG)** |
| **Stabilization Settle Delay** | **CÓ:** `core/state_resolver.py:633-642`<br>Chờ `dom_settle_delay` (mặc định 5.0 giây) để Language Server ổn định trước khi gọi RPC. | **KHÔNG CÓ:** Sau khi recycle xong, lập tức gọi continuation mà không có độ trễ ổn định, dễ gây lỗi 500 extension server. | **P1 (NGHIÊM TRỌNG)** |
| **Logging & Báo cáo Lỗi** | **CÓ:** `logger.debug(...)` chi tiết từng lỗi của `smart_focus`, `load_trajectory`, `send_user_cascade_message`. | **KHÔNG CÓ:** Nuốt lỗi 100% bằng `_ = ...` ở mọi tầng. | **P1 (NGHIÊM TRỌNG)** |

---

## 5. Kết luận Nguyên nhân Gốc rễ (Root Cause Synthesis)

Có **3 nguyên nhân gốc rễ tương hỗ** gây ra sự cố mất cơ chế gửi prompt tiếp tục phiên:

1. **Nguyên nhân Gốc rễ #1 (Primary Blocker - Lệch pha SSOT): Thiếu cơ chế Fallback `DiscoverMostRecentCascadeID`**
   - Khi chuyển đổi mã nguồn từ Python sang Go, kỹ sư đã bỏ sót hoàn toàn hàm `discover_most_recent_cascade_id` (vốn là trái tim của cơ chế P0-04 trong Python `core/session_reloader.py:557-650`).
   - Do `app_storage.json` chưa kịp flush các phiên làm việc hiện tại, hàm đọc snapshot chỉ dựa vào file tĩnh trên đĩa nên trả về danh sách rỗng (`[]`). Thiếu hàm quét mtime từ `brain_dir` khiến hệ thống mất hoàn toàn khả năng nhận diện thread đang mở của người dùng.

2. **Nguyên nhân Gốc rễ #2 (Secondary Blocker): Parser `CaptureActiveThreadsSnapshot` lỗi thời**
   - Phiên bản Antigravity hiện tại (2.24.0) tổ chức các tab hội thoại dưới dạng `subagentView` (`childCascadeId`) và `subagent_convo__<uuid>`.
   - Parser của Go chỉ tìm các trường cũ ở tầng root (`tab["cascadeId"]`, `tab["childCascadeId"]`), không giải nén object `subagentView`, dẫn đến việc quét các pane hợp lệ nhưng vẫn coi là không có thread.

3. **Nguyên nhân Gốc rễ #3 (Silent Failure): Nuốt lỗi hoàn toàn và thiếu thông tin phản hồi trên UI/CLI**
   - Toàn bộ các tầng từ `SendUserCascadeMessage`, `SmartFocusConversation`, `LoadTrajectory` đến `PerformSessionContinuation` đều sử dụng `_ = ...`.
   - Không có bất kỳ dòng log nào, không có trường nào trong `SwitchResult` ghi nhận số lượng prompt đã gửi thành công, khiến lập trình viên và người dùng không thể biết quá trình tiếp tục phiên có diễn ra hay không.

---

## 6. Đề xuất Phương án Khắc phục Dứt điểm (Remedy Action Plan)

Kiến trúc sư đề xuất phương án khắc phục chuẩn hóa theo 5 bước rõ ràng để bàn giao cho Làn Coder thực thi sau khi Owner phê duyệt:

### Bước 1: Cổng hàm `DiscoverMostRecentCascadeID` chuẩn Python SSOT sang Go
- Tạo hàm `DiscoverMostRecentCascadeID(brainDir, storagePath string) (string, error)` trong `internal/reloader/active_threads.go`:
  1. Quét toàn bộ thư mục con trong `brainDir` (mặc định `%USERPROFILE%/.gemini/antigravity/brain/`).
  2. Tìm tệp `transcript.jsonl` hoặc `.system_generated/logs/transcript.jsonl` hoặc `transcript_full.jsonl`.
  3. Lấy `ModTime()` của tệp transcript để tìm các thread có hoạt động gần đây nhất.
  4. Sắp xếp các ứng viên theo `ModTime` giảm dần.
  5. Đối soát chéo với các layout keys trong `app_storage.json` (`antigravity-multi-conversation-layout-v3-*`) để ưu tiên thread chính.
  6. Trả về Cascade ID của thread đang hoạt động gần nhất.

### Bước 2: Tích hợp Fallback tự động vào `CaptureActiveThreadsSnapshot` và `SwitchAccount`
- Trong `defaultCaptureActiveThreadsSnapshot`:
  - Nếu sau khi duyệt qua toàn bộ các pane trong `app_storage.json` mà `len(threadsMap) == 0`:
  - Tự động kích hoạt `DiscoverMostRecentCascadeID(bDir, sPath)`.
  - Nếu tìm thấy `discoveredCID != ""`, tự động thêm vào danh sách:
    ```go
    threadsMap[discoveredCID] = ActiveThread{
        ThreadID:   discoveredCID,
        Project:    resolveThreadProject(discoveredCID, nil, bDir, sdata),
        IsMainPane: true,
    }
    ```
- Điều này đảm bảo **`activeThreads` KHÔNG BAO GIỜ BỊ RỖNG** khi người dùng đang có ít nhất một phiên Antigravity còn lưu vết trong `brain`.

### Bước 3: Nâng cấp Parser trích xuất Subagent Tabs trong `active_threads.go`
- Bổ sung logic trích xuất `childCascadeId`:
  ```go
  // 1. Kiểm tra subagentView bên trong content
  if content, ok := tab["content"].(map[string]interface{}); ok {
      if subView, ok := content["subagentView"].(map[string]interface{}); ok {
          if cid := getMapString(subView, "childCascadeId"); cid != "" {
              tID = cid
          }
      }
      if bgTask, ok := content["backgroundTaskView"].(map[string]interface{}); ok {
          if cid := getMapString(bgTask, "cascadeId"); cid != "" {
              tID = cid
          }
      }
  }
  // 2. Trích xuất từ tiền tố tab ID: subagent_convo__<UUID>
  if tID == "" {
      tabID := getMapString(tab, "id")
      if strings.HasPrefix(tabID, "subagent_convo__") {
          tID = strings.TrimPrefix(tabID, "subagent_convo__")
      }
  }
  ```

### Bước 4: Sửa lỗi truyền tham số trong `internal/manager/account_manager.go`
- Thay thế dòng gọi:
  ```go
  storagePath := ""
  brainDir := ""
  if rel != nil {
      storagePath = rel.StoragePath
      brainDir = rel.BrainDir
  }
  _, activeThreads, _ = reloader.CaptureActiveThreadsSnapshot(storagePath, brainDir, "")
  ```

### Bước 5: Loại bỏ Nuốt Lỗi & Minh Bạch Hóa Trạng Thái Gửi Prompt
- Trong `internal/reloader/continuation.go`:
  - Thêm độ trễ ổn định `settleDelay` (1.5s - 2.0s) trước khi gửi RPC đầu tiên.
  - Ghi nhận kết quả của từng thread: số thread gửi thành công, số thread thất bại kèm mã lỗi chi tiết.
  - Không nuốt lỗi âm thầm; trả về cấu trúc `ContinuationSummary`:
    ```go
    type ContinuationSummary struct {
        TotalThreads int
        SentSuccess  int
        SentThreads  []string
        Errors       []string
    }
    ```
- Trong `account_manager.go`:
  - Nhận `ContinuationSummary` từ `PerformSessionContinuation`.
  - Cập nhật thông điệp hoàn thành của `SwitchResult.Message`:
    `"Chuyển sang tài khoản <email> thành công! Đã gửi prompt tiếp tục phiên vào <N> thread (<IDs>)."`
  - Hiển thị thông điệp minh bạch này lên thanh trạng thái của GUI Fyne và output của CLI switcher.

---
**Báo cáo được ký xác thực bởi:** Senior System Architect  
**Cam kết:** 100% tuân thủ Iron Laws, không tự ý sửa code sản phẩm khi chưa có lệnh của Owner, mọi kết luận đều có chứng cứ thực nghiệm runtime xác đáng.
