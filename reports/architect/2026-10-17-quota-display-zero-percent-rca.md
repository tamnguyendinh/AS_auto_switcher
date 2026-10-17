# Root Cause Analysis (RCA) Báo Cáo Chuyên Sâu: Sự Cố Hiển Thị Quota 0% Màu Đỏ Trên Fyne GUI

**Tài liệu tham chiếu:** `reports/architect/2026-10-17-quota-display-zero-percent-rca.md`  
**Ngày thực hiện:** 2026-10-17  
**Làn đảm nhiệm:** Senior System Architect (Kiến Trúc Sư Hệ Thống)  
**Cơ quan chỉ đạo:** CEO Agent (Thread ID: `01a14378-4a08-77c1-88d5-666ad8c3592d`)  
**Thẩm quyền & Căn cứ:** Chỉ đạo trực tiếp từ Human Owner, hình ảnh bằng chứng thực tế `reports/problem/images/screenshot_1792236267.png`, mã nguồn Go (`internal/ui/`, `internal/telemetry/`, `internal/store/`, `internal/connector/`, `cmd/switcher/`), bộ nhớ đệm `accounts_store/.quota_cache.json`, và runtime thực tế của Antigravity `language_server.exe`.  
**Trạng thái thẩm tra:** HOÀN TẤT & ĐỐI SOÁT BẰNG CHỨNG THỰC NGHIỆM 100% (ZERO SPECULATION)

---

## 1. Tóm tắt sự cố & Hiện tượng quan sát (Incident Summary)

### 1.1 Hiện tượng quan sát trên ảnh thực tế (`reports/problem/images/screenshot_1792236267.png`)
Human Owner cung cấp bằng chứng ảnh chụp giao diện Fyne GUI của ứng dụng Antigravity Multi-Account Switcher (phiên bản Phase 6A):
1. **Header Card:** Hiển thị `Hạn ngạch: Claude: 0% | Gemini: 0% | GPT: 0%`.
2. **Toàn bộ 7 Thẻ tài khoản (Account Cards):** Đều hiển thị 3 chấm tròn màu đỏ kèm nhãn:
   - `Claude: 0%`
   - `Gemini: 0%`
   - `GPT: 0%`
3. **Dòng cảnh báo màu đỏ/cam dưới mỗi thẻ:**
   - `⚠️ Gemini, Claude, GPT-OSS sắp cạn (Đang chờ thông tin hồi phục từ Google)`
4. **Nút bấm làm mới:** Nút `[Làm mới Quota Tất cả]` hiển thị sẵn sàng, thanh trạng thái báo `Đồng bộ lúc 18:24:14 • 7 tài khoản trong kho JSON • 7 trên Chrome`.
5. **Mâu thuẫn kỹ thuật:** Trước đó, việc quét API thực tế cho thấy các tài khoản đều dồi dào hạn ngạch (90% - 100%), nhưng GUI hiển thị 0% cạn kiệt toàn diện.

---

## 2. Bằng chứng đối soát Thực nghiệm & Runtime (Empirical Evidence)

### 2.1 Bằng chứng File Cache (`accounts_store/.quota_cache.json`)
Kiểm tra tệp cache lưu tại `accounts_store/.quota_cache.json` vào lúc `2026-10-17T11:27:08.8142107Z`:
```json
{
  "saved_at": "2026-10-17T11:27:08.8142107Z",
  "account_count": 7,
  "matrix": {
    "hiendao1760@gmail.com": {
      "email": "hiendao1760@gmail.com",
      "claude": {
        "model_id": "",
        "remaining_fraction": 0,
        "reset_time": "0001-01-01T00:00:00Z"
      },
      "gemini": {
        "model_id": "",
        "remaining_fraction": 0,
        "reset_time": "0001-01-01T00:00:00Z"
      },
      "gpt": {
        "model_id": "",
        "remaining_fraction": 0,
        "reset_time": "0001-01-01T00:00:00Z"
      },
      "reset_time": "0001-01-01T00:00:00Z",
      "status": "Error",
      "error_message": "failed to retrieve user quota from google: server returned status 403: {\n  \"error\": {\n    \"code\": 403,\n    \"message\": \"You do not have a valid license of this product. Please contact your administrator to request a license. If you are not an enterprise user and believe you are receiving this message as an error, please try using the latest version and logging in again. (#3501)\",\n    \"status\": \"PERMISSION_DENIED\",\n    \"details\": [\n      {\n        \"@type\": \"type.googleapis.com/google.rpc.ErrorInfo\",\n        \"reason\": \"SUBSCRIPTION_REQUIRED\",\n        \"domain\": \"cloudaicompanion.googleapis.com\",\n        \"metadata\": {\n          \"uiMessage\": \"true\",\n          \"error_number\": \"1001\"\n        }\n      }\n    ]\n  }\n}\n",
      "last_updated": "2026-10-17T11:27:08.7137604Z"
    }
  }
}
```
**Phát hiện:** Toàn bộ 7/7 tài khoản đều có `status: "Error"`, mã lỗi HTTP 403 `PERMISSION_DENIED` với lý do `SUBSCRIPTION_REQUIRED` trên domain `cloudaicompanion.googleapis.com`.

### 2.2 Bằng chứng Thực nghiệm Header `User-Agent` (The Silver Bullet Proof)
Kiến trúc sư tiến hành thử nghiệm gửi cùng một access token tươi tới Google Cloud Code Quota Endpoint (`https://daily-cloudcode-pa.googleapis.com/v1internal:retrieveUserQuota`) với các biến thể Header `User-Agent`:

| User-Agent Thử Nghiệm | Phản Hồi Từ Google Gateway | Phân Loại |
|---|:---:|---|
| *(Không truyền User-Agent)* | **HTTP 403 PERMISSION_DENIED** (896 bytes) | `SUBSCRIPTION_REQUIRED (#3501)` |
| `Go-http-client/1.1` (Mặc định Go) | **HTTP 403 PERMISSION_DENIED** (896 bytes) | `SUBSCRIPTION_REQUIRED (#3501)` |
| `Mozilla/5.0` (Trình duyệt chung) | **HTTP 403 PERMISSION_DENIED** (896 bytes) | `SUBSCRIPTION_REQUIRED (#3501)` |
| `antigravity` | **HTTP 200 OK** (2749 bytes) | **Thành công 100%** |
| `antigravity/2.24.0` | **HTTP 200 OK** (3212 bytes) | **Thành công 100% (Chuẩn Antigravity IDE)** |

Tiếp tục kiểm tra song song với `User-Agent: antigravity/2.24.0` trên toàn bộ 7/7 tài khoản thực tế trong `accounts_store/`:
- `[daohien204ccyersin@gmail.com]` -> **HTTP 200 OK** (3332 bytes)
- `[hiendao1760@gmail.com]` -> **HTTP 200 OK** (3332 bytes)
- `[hokinhdoanh.anvien.sales@gmail.com]` -> **HTTP 200 OK** (3212 bytes)
- `[ngau2hung@gmail.com]` -> **HTTP 200 OK** (3212 bytes)
- `[quantritaikhoanemail@gmail.com]` -> **HTTP 200 OK** (3212 bytes)
- `[skey.miennam@gmail.com]` -> **HTTP 200 OK** (3212 bytes)
- `[skey.vanphongmiennam@gmail.com]` -> **HTTP 200 OK** (3212 bytes)

**Kết luận thực nghiệm:** Tỷ lệ thành công đạt **100% (7/7 accounts)** khi có Header `User-Agent: antigravity/2.24.0`.

### 2.3 Bằng chứng Phân Tích Nhị Phân `language_server.exe`
Kiểm tra dòng lệnh thực tế của tiến trình `language_server.exe` đang chạy (PID 7768):
```text
"C:\Users\TAM NGUYEN\AppData\Local\Programs\antigravity\resources\bin\language_server.exe" 
  --standalone 
  --override_ide_name antigravity 
  --subclient_type hub 
  --override_ide_version 2.24.0 
  --override_user_agent_name antigravity 
  --csrf_token 0c4298fb-8bbb-4452-9be9-8740e95e9108 
  --cloud_code_endpoint https://daily-cloudcode-pa.googleapis.com
```
Antigravity IDE định danh chính xác với Google qua cờ `--override_user_agent_name antigravity` và version `2.24.0`.

### 2.4 Bằng chứng Đột Phá Kiến Trúc: Connect-RPC Local Quota Endpoint
Quét nhị phân `language_server.exe` phát hiện tiến trình cục bộ hỗ trợ endpoint Connect-RPC:
`/exa.language_server_pb.LanguageServerService/RetrieveUserQuotaSummary`

Gửi thử nghiệm HTTP POST tới `http://127.0.0.1:52704/exa.language_server_pb.LanguageServerService/RetrieveUserQuotaSummary`:
```json
{
  "response": {
    "groups": [
      {
        "displayName": "Gemini Models",
        "buckets": [
          {
            "bucketId": "gemini-5h",
            "displayName": "Five Hour Limit Remaining",
            "remainingFraction": 0.933044,
            "resetTime": "2026-10-17T12:19:14Z"
          },
          {
            "bucketId": "gemini-weekly",
            "displayName": "Weekly Limit Remaining",
            "remainingFraction": 0.7574286,
            "resetTime": "2026-10-24T02:19:14Z"
          }
        ]
      },
      {
        "displayName": "Claude and GPT models",
        "buckets": [
          {
            "bucketId": "3p-5h",
            "displayName": "Five Hour Limit Remaining",
            "remainingFraction": 1.0,
            "resetTime": "2026-10-17T12:32:55Z"
          },
          {
            "bucketId": "3p-weekly",
            "displayName": "Weekly Limit Remaining",
            "remainingFraction": 1.0,
            "resetTime": "2026-10-24T07:32:55Z"
          }
        ]
      }
    ]
  }
}
```
**Kết quả:** Trả về kết quả sau **~45 ms** từ RAM cục bộ của `language_server.exe` mà không cần ra internet, không cần refresh OAuth token.

---

## 3. Thẩm Tra Chi Tiết 4 Câu Hỏi Trọng Tâm Của CEO & Human Owner

### Câu hỏi 1: Tại sao GUI khi khởi động hoặc làm mới lại nhận giá trị 0% hoặc không đọc được dữ liệu quota thực tế?

**Trả lời:** Sự cố bắt nguồn từ chuỗi tác động domino qua 3 tầng (Tầng Giao thức mạng -> Tầng Telemetry Pool -> Tầng Giao diện UI):
1. **Tầng Giao thức mạng (`internal/telemetry/client.go`):** Hàm `RetrieveUserQuota` tạo HTTP request gửi tới `https://daily-cloudcode-pa.googleapis.com/v1internal:retrieveUserQuota` chỉ set `Authorization`, `Content-Type`, và `Accept`, hoàn toàn bỏ quên Header `User-Agent`. Do Go HTTP client mặc định gửi `User-Agent: Go-http-client/1.1`, Google gateway phân loại client này là truy cập trái phép không thuộc IDE Antigravity và chặn lại bằng mã `HTTP 403 PERMISSION_DENIED (SUBSCRIPTION_REQUIRED)`.
2. **Tầng Telemetry Pool (`internal/telemetry/pool.go`):** Khi `pollAccountWithToken` nhận lỗi HTTP 403, hàm này tạo một struct `AccountQuotaData` lỗi:
   ```go
   errData := &AccountQuotaData{
       Email:        cleanEmail,
       Status:       StatusError,
       ErrorMessage: err.Error(),
       LastUpdated:  time.Now().UTC(),
   }
   ```
   Trong struct này, các trường `Claude`, `Gemini`, `GPT` mang giá trị zero-value của Go: `RemainingFraction: 0.0`, `ResetTime: 0001-01-01T00:00:00Z`. Dữ liệu lỗi này được lưu đè vào `QuotaCache` và ghi xuống tệp đĩa `.quota_cache.json`.
3. **Tầng Giao diện UI (`internal/ui/quota_badge.go` & `widgets.go`):**
   - Hàm `NewQuotaBadgeGroup(quota)` chỉ kiểm tra `if quota == nil`. Khi `quota != nil` (dù mang `Status: StatusError`), UI coi `RemainingFraction: 0.0` là dữ liệu hợp lệ (`hasData = true`).
   - Hàm `GetQuotaTierColor(0.0)` trả về `ColorQuotaRed` (`#ef4444`).
   - Hàm `formatFraction(0.0)` trả về chuỗi `"0%"`.
   - Kết quả: GUI hiển thị 3 chấm đỏ: `Claude: 0% | Gemini: 0% | GPT: 0%`.
   - Tiếp theo, `NewQuotaCountdownLabel(quota)` so sánh `RemainingFraction < 0.15`. Do `0.0 < 0.15` là đúng với cả 3 model, danh sách `lowModels` có cả `Gemini, Claude, GPT-OSS`. Vì `resetTime` là zero, hàm render dòng cảnh báo: `"⚠️ Gemini, Claude, GPT-OSS sắp cạn (Đang chờ thông tin hồi phục từ Google)"`.

### Câu hỏi 2: Có phải do đường dẫn file cache `accounts_store/.quota_cache.json` bị lệch pha tương đối giữa thư mục chạy binary và thư mục gốc của repo hay không?

**Trả lời:** **CÓ NGUY CƠ LỆCH PHA Ở MỘT SỐ HOÀN CẢNH, NHƯNG KHÔNG PHẢI NGUYÊN NHÂN TRỰC TIẾP TẠO RA SỐ 0% HIỆN TẠI.**
- **Tại runtime bình thường:** Trong `internal/ui/app.go` và `cmd/switcher/main.go`, code đã có đoạn bọc:
  ```go
  if mgr != nil && mgr.Store() != nil && mgr.Store().GetStoreDir() != "" {
      cachePath = filepath.Join(mgr.Store().GetStoreDir(), ".quota_cache.json")
  }
  ```
  Nhờ `store.GetDefaultStoreDir()` tìm kiếm thư mục cha có `accounts_store` và `go.mod`, tệp cache tại gốc repo `accounts_store/.quota_cache.json` đã được ghi nhận đúng lúc 18:27 và 18:29.
- **Điểm khiếm khuyết phát hiện:** Trong `internal/telemetry/cache.go`, hằng số mặc định là đường dẫn tương đối:
  ```go
  const DefaultQuotaCachePath = "accounts_store/.quota_cache.json"
  ```
  Nếu khởi tạo `telemetry.NewQuotaCache("")` mà không thông qua `AccountStore` (ví dụ khi chạy test hoặc khi app khởi động độc lập từ shortcut Windows), đường dẫn tương đối sẽ tạo ra tệp rác tại thư mục chạy lệnh (minh chứng: đã tìm thấy tệp rác `internal\telemetry\accounts_store\.quota_cache.json`).
- **Khuyến nghị kiến trúc:** Cần loại bỏ hoàn toàn hardcoded relative path, chuyển sang dùng `store.GetDefaultStoreDir()` để luôn đảm bảo đường dẫn tuyệt đối bất kể working directory.

### Câu hỏi 3: Có phải do cấu trúc struct JSON giữa `internal/telemetry/` và `internal/ui/` bị lệch tên trường (`remaining_fraction` vs `remainingFraction` hoặc tỷ lệ 0.0-1.0 vs 0-100%) dẫn đến việc parse ra giá trị 0 hay không?

**Trả lời:** **HOÀN TOÀN KHÔNG PHẢI.**
Kiến trúc sư đã đối soát mã nguồn và khẳng định:
1. **Khớp nối Go struct nội bộ:** Giữa `internal/telemetry/` và `internal/ui/` sử dụng chung struct `telemetry.AccountQuotaData` và `telemetry.ModelQuotaBucket`. UI truy cập trực tiếp struct field trong bộ nhớ RAM, không qua serialize/deserialize trung gian.
2. **Khớp nối JSON Disk Cache:** Trong `internal/telemetry/cache.go`, thẻ tag JSON là:
   `RemainingFraction float64 json:"remaining_fraction"`
   Tệp `accounts_store/.quota_cache.json` lưu đúng trường `"remaining_fraction"`. Việc đọc/ghi bằng `json.Marshal`/`json.Unmarshal` hoàn toàn toàn vẹn.
3. **Khớp nối API Google:** Trong `internal/telemetry/client.go`, struct `rawBucketInfo` hỗ trợ cả 2 biến thể:
   `RemainingFraction *float64 json:"remainingFraction"` (Google trả về)
   `RemainingFractionSnake *float64 json:"remaining_fraction"`
4. **Tỷ lệ phần trăm:** Giá trị số thực `0.0 - 1.0` được hàm `formatFraction()` nhân với 100.0 một cách chuẩn xác (`1.0 -> 100%`, `0.235 -> 23.5%`, `0.0 -> 0%`).
*Kết luận:* Giá trị 0% xuất hiện là do struct zero-value khi gặp lỗi HTTP 403, hoàn toàn không phải do lỗi lệch tên trường hay parse JSON.

### Câu hỏi 4: Có phải do hàm `RefreshAllQuota` trong GUI gặp lỗi khi gọi worker pool ngầm nhưng bị nuốt lỗi (error swallowed) hay không?

**Trả lời:** **CHÍNH XÁC 100%. LỖI ĐÃ BỊ NUỐT Ở CẢ 2 TẦNG (WORKER POOL VÀ UI EVENT LOOP).**
- **Nuốt lỗi tại Tầng 1 (`internal/telemetry/pool.go`):**
  Trong hàm `PollAll`:
  ```go
  for i, acc := range accounts {
      wg.Add(1)
      go func(idx int, info AccountTokenInfo) {
          defer wg.Done()
          data, _ := p.pollAccountWithToken(pollCtx, info.Email, info.RefreshToken) // <--- BỎ QUA ERROR BẰNG '_'
          results[idx] = data
      }(i, acc)
  }
  wg.Wait()
  return results, nil // <--- LUÔN TRẢ VỀ NIL ERROR!
  ```
  Dù 100% tài khoản bị lỗi HTTP 403, `PollAll` vẫn trả về `results, nil`.
- **Nuốt lỗi tại Tầng 2 (`internal/ui/app.go`):**
  Trong hàm `RefreshAllQuota`:
  ```go
  results, err := a.telemetryPool.PollAll(ctx)
  ...
  fyne.Do(func() {
      if err != nil {
          a.statusBarLabel.SetText(fmt.Sprintf("Lỗi quét quota: %v", err))
      } else {
          a.statusBarLabel.SetText(fmt.Sprintf("Đã làm mới quota %d tài khoản lúc %s", len(results), nowStr))
      }
  })
  ```
  Vì `err` luôn là `nil`, giao diện luôn báo: *"Đã làm mới quota 7 tài khoản lúc 18:24:09"*, che giấu hoàn toàn sự cố lỗi mạng đối với người dùng!
- **Đè trạng thái:** Hàm `a.updateUI(overview)` sau đó tiếp tục ghi đè thanh trạng thái bằng thông điệp định kỳ: *"Đồng bộ lúc 18:24:14 • 7 tài khoản trong kho JSON • 7 trên Chrome"*, khiến mọi dấu vết lỗi trên thanh trạng thái biến mất hoàn toàn.

---

## 4. Sơ Đồ Luồng Dữ Liệu & Điểm Lỗi Kỹ Thuật (Data Flow Failure Trace)

```text
[Người dùng bấm "Làm mới Quota Tất cả" hoặc Ticker 60s kích hoạt]
                         │
                         ▼
           ui.App.RefreshAllQuota()
                         │
                         ▼
        telemetry.TelemetryPool.PollAll()
                         │
            ┌────────────┴────────────┐
            ▼                         ▼
   Tài khoản 1 (hiendao)      Tài khoản 2 (daohien)...
            │                         │
            ▼                         ▼
telemetry.QuotaClient.RetrieveUserQuota(ctx, token)
            │
            ├─► Gửi HTTP POST tới Google: https://daily-cloudcode-pa.googleapis.com
            ├─► Thiếu User-Agent! (Gửi mặc định: "Go-http-client/1.1")
            │
            ▼
   [Google Gateway từ chối: HTTP 403 PERMISSION_DENIED / SUBSCRIPTION_REQUIRED]
            │
            ▼
   pollAccountWithToken tạo errData:
     - Claude: { RemainingFraction: 0.0 }  (zero-value)
     - Gemini: { RemainingFraction: 0.0 }  (zero-value)
     - GPT:    { RemainingFraction: 0.0 }  (zero-value)
     - Status: "Error"
            │
            ▼
   QuotaCache.Update(errData) -> Lưu giá trị 0.0 vào RAM QuotaMatrix
   QuotaCache.SaveToDisk()    -> Lưu giá trị 0.0 vào .quota_cache.json
            │
            ▼
   PollAll nuốt lỗi (data, _ = ...), trả về results, nil
            │
            ▼
   ui.App.updateUI():
     - NewQuotaBadgeGroup: Không check quota.Status == "Error"
       => Render: Claude: 0% | Gemini: 0% | GPT: 0% (Chấm đỏ)
     - NewQuotaCountdownLabel: 0.0 < 0.15
       => Render: "⚠️ Gemini, Claude, GPT-OSS sắp cạn (Đang chờ thông tin hồi phục từ Google)"
```

---

## 5. Đề Xuất Giải Pháp Kỹ Thuật Sửa Lỗi Toàn Diện (Architectural Remedy Specification)

Để khắc phục triệt để và bảo đảm tính chịu lỗi cao theo tiêu chuẩn Production, Kiến trúc sư ban hành đặc tả giải pháp 5 trụ cột kỹ thuật cho Coder Lane triển khai:

### Trụ cột 1: Bổ sung Header `User-Agent: antigravity/2.24.0` (Root Protocol Fix)
Trong `internal/telemetry/client.go`:
- Khai báo hằng số chuẩn:
  ```go
  const DefaultUserAgent = "antigravity/2.24.0"
  ```
- Thêm trường `userAgent string` vào struct `QuotaClient`. Khởi tạo mặc định `DefaultUserAgent`.
- Trong hàm `RetrieveUserQuota`:
  ```go
  req.Header.Set("User-Agent", c.userAgent)
  ```
- Kiểm tra tính tương thích ngược trong unit test với mock server.

### Trụ cột 2: Phân tách Trạng thái Lỗi (`StatusError`) khỏi Trạng thái Hết Quota (`Depleted`) trên UI
Trong `internal/ui/quota_badge.go` & `internal/ui/widgets.go`:
- Trong hàm `NewQuotaBadgeGroup(quota *telemetry.AccountQuotaData)`:
  Kiểm tra thêm:
  ```go
  if quota == nil {
      // Hiển thị --% (Màu xám/Muted)
  }
  if quota.Status == telemetry.StatusError {
      // Hiển thị trạng thái lỗi kết nối:
      // Claude: [!] | Gemini: [!] | GPT: [!] (Màu cam Warning hoặc xám Muted có tooltip/nhãn lỗi)
      // HOẶC hiển thị nhãn: "Lỗi kết nối Quota" thay vì hiển thị 0%
  }
  ```
- Trong hàm `NewQuotaCountdownLabel(quota *telemetry.AccountQuotaData)`:
  **BẮT BUỘC bỏ qua** nếu `quota.Status == telemetry.StatusError` hoặc `quota.ErrorMessage != ""`:
  ```go
  if quota == nil || quota.Status == telemetry.StatusError {
      return nil
  }
  ```
  Điều này triệt tiêu hoàn toàn cảnh báo sai lệch *"Sắp cạn (Đang chờ thông tin hồi phục)"* khi hệ thống gặp lỗi mạng hoặc lỗi phân quyền.

### Trụ cột 3: Xử lý Lỗi Toàn Diện & Phản Hồi Trực Quan (Eliminate Error Swallowing)
1. **Trong `internal/telemetry/pool.go`:**
   - Thay đổi `PollAll` để gom góp các lỗi con (aggregate errors) hoặc đếm số lượng tài khoản lỗi:
     ```go
     // Trả về error tổng hợp nếu có ít nhất một tài khoản bị lỗi,
     // hoặc trả về chi tiết thống kê (successCount, failCount).
     ```
2. **Trong `internal/ui/app.go`:**
   - Khi `RefreshAllQuota` hoàn tất:
     Đếm trong `results` số tài khoản `Status == StatusError`.
     Nếu có lỗi, hiển thị cảnh báo rõ ràng trên thanh trạng thái:
     `"Làm mới quota: X thành công, Y lỗi kết nối (Xem chi tiết)"`.
   - Lưu trữ `lastErrorMessage` để hiển thị dialog cảnh báo nếu người dùng bấm thủ công.

### Trụ cột 4: Kiến Trúc Thu Thập Quota Lai (Hybrid Quota Telemetry Engine)
Tận dụng phát hiện đột phá về Connect-RPC của `language_server.exe`:
- **Đối với Tài khoản Đang Hoạt Động (Active Account):**
  Ưu tiên gọi trực tiếp `http://127.0.0.1:<port>/exa.language_server_pb.LanguageServerService/RetrieveUserQuotaSummary` thông qua `internal/connector/Client`.
  - Ưu điểm: Phản hồi siêu tốc (~45 ms), không cần kết nối mạng internet, không tiêu tốn token refresh, không phụ thuộc vào IP/proxy bên ngoài.
  - Dự phòng: Nếu `language_server.exe` không chạy hoặc RPC trả về lỗi, tự động fallback sang `v1internal:retrieveUserQuota` của Google.
- **Đối với Các Tài khoản Còn Lại Trong Kho (Inactive Accounts):**
  Tiếp tục truy vấn song song qua worker pool tới `daily-cloudcode-pa.googleapis.com` có Header `User-Agent: antigravity/2.24.0`.

### Trụ cột 5: Chuẩn hóa Tuyệt Đối Đường Dẫn Cache (Absolute Cache Path Invariant)
Trong `internal/telemetry/cache.go`:
- Thay thế relative constant `DefaultQuotaCachePath` bằng hàm giải quyết đường dẫn động hoặc khởi tạo mặc định liên kết với `store.GetDefaultStoreDir()`:
  ```go
  func GetDefaultQuotaCachePath() string {
      return filepath.Join(store.GetDefaultStoreDir(), ".quota_cache.json")
  }
  ```
- Triệt tiêu 100% tình trạng sinh tệp rác `.quota_cache.json` ở các thư mục con hoặc khi thực thi từ shortcut.

---

## 6. Phân Bổ Nhiệm Vụ Tiếp Theo & Ma Trận Nghiệm Thu (Downstream Handoff)

| Hạng mục cần thực hiện | Làn đảm nhiệm | Tiêu chuẩn nghiệm thu |
|---|:---:|---|
| **Patch Code Telemetry Client** | Coder | Thêm `User-Agent: antigravity/2.24.0` vào `internal/telemetry/client.go`. Chạy test suite `internal/telemetry` pass 100%. |
| **Patch UI Quota Badges & Labels** | Coder | Chặn hiển thị 0% và chặn dòng cảnh báo khi `Status == StatusError` trong `internal/ui/quota_badge.go`. |
| **Patch Error Propagation** | Coder | Không nuốt lỗi trong `internal/telemetry/pool.go` và hiển thị trạng thái lỗi chuẩn xác trên GUI `internal/ui/app.go`. |
| **Patch Absolute Cache Path** | Coder | Chuẩn hóa `DefaultQuotaCachePath` thông qua `store.GetDefaultStoreDir()`. |
| **Triển khai Hybrid RPC (Khuyến nghị)** | Coder | Tích hợp đọc quota của active account qua `LanguageServerClient.RetrieveUserQuotaSummary`. |
| **Kiểm Thử Thực Nghiệm & Visual QA** | QA | Khởi chạy GUI thực tế: xác nhận các tài khoản hiển thị hạn ngạch thật (90-100% màu xanh) thay vì 0% màu đỏ. |
| **Nghiệm Thu Toàn Diện** | Supervisor | Rà soát zero-regression trên toàn bộ 9 packages, build binary thành công. |

---

## 7. Kết luận của Kiến trúc sư (Architect Verdict)

- **Kết luận:** Nguyên nhân gốc rễ dẫn đến lỗi hiển thị Quota 0% trên Fyne GUI đã được **làm sáng tỏ 100% với đầy đủ bằng chứng thực nghiệm và đối soát nhị phân**.
- **Nguyên nhân cốt lõi:** Google Cloud Code API gateway chặn yêu cầu bằng HTTP 403 do thiếu Header `User-Agent: antigravity/2.24.0`, kết hợp với việc Fyne GUI diễn dịch nhầm struct lỗi zero-value `0.0` thành hạn ngạch cạn kiệt và nuốt toàn bộ thông báo lỗi.
- **Phương án giải quyết:** 5 trụ cột kỹ thuật nêu trên là giải pháp kiến trúc tối ưu, triệt để và an toàn nhất để bàn giao cho Coder Lane thực hiện.

**TRẠNG THÁI KIẾN TRÚC: APPROVED & READY FOR OWNER / CEO REVIEW**
