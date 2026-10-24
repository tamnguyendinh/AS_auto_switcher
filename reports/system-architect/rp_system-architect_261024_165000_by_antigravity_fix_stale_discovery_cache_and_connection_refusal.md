# BÁO CÁO ĐẶC TẢ KIẾN TRÚC CHI TIẾT (SYSTEM ARCHITECTURE SPECIFICATION)
## Khắc Phục Triệt Để Lỗi Kết Nối Cổng Cũ Đã Chết (Stale Discovery Cache & Connection Refusal) Khi Chuyển Tài Khoản

- **Mã tài liệu:** `rp_system-architect_261024_165000_by_antigravity_fix_stale_discovery_cache_and_connection_refusal`
- **Tác giả:** System Architect (Strategic Lane)
- **Ngày lập:** 2026-10-24
- **Thẩm quyền phê duyệt:** Human Owner
- **Trạng thái:** PRODUCTION-READY BLUEPRINT (Bản thiết kế chìa khóa trao tay cho Coder)
- **Mục tiêu tối thượng:**
  1. Khắc phục triệt để lỗi `failed verifying user status in RAM: no candidate endpoint responded successfully: Post "http://127.0.0.1:<old_port>/user_status": dial tcp 127.0.0.1:<old_port>: connectex: No connection could be made because the target machine actively refused it`.
  2. Triệt tiêu hoàn toàn hiện tượng lấy nhầm cổng chết do bộ nhớ đệm Discovery quá hạn (`discCache` stale) sau khi tiến trình `language_server.exe` bị kill.
  3. **BẢO TỒN NGUYÊN VẸN 100% QUY TRÌNH CHỜ ĐỦ 8 GIÂY** (`waitDuration = 8 * time.Second` và `TestSwitchAccount_8SecondWaitAnd4StepWorkflow`), tuyệt đối không được cắt giảm hay thay thế khoảng đệm an toàn này.
  4. Thiết lập cơ chế phòng thủ 3 tầng: Tự động xóa cache khi kill tiến trình (Proactive Invalidation), Kiểm tra tính sống sót thực sự của tiến trình trên Windows (Liveness Verification), và Tự động giải phóng cache khi gặp lỗi kết nối cấp mạng (Reactive Eviction).

---

# PHẦN 1: BÁO CÁO HIỆN TRẠNG & BẰNG CHỨNG ĐỐI SOÁT FORENSIC

Qua đối soát trực tiếp giữa nhật ký thao tác người dùng tại [`logs/switch_timeline/2026-10-24_switch_timeline.log`](file:///g:/antigravity_auto_switch/logs/switch_timeline/2026-10-24_switch_timeline.log) và nhật ký Antigravity IDE tại `%APPDATA%\Antigravity\logs\main.log`, Kiến trúc sư phát hiện bằng chứng xác thực nguyên nhân gốc rễ:

### 1.1. Dòng thời gian đối soát lúc 15:54:40 (Trùng khớp ảnh chụp màn hình của Owner)
- **15:27:03**: Antigravity IDE khởi tạo `language_server.exe` chạy tài khoản `daohien204ccyersin@gmail.com` trên cặp cổng: HTTPS **54315** và Connect-RPC HTTP **54316**.
- **15:54:40.211**: Người dùng bấm nút **[Chuyển]** sang tài khoản `hokinhdoanh.anvien.sales@gmail.com`.
- **15:54:40.420**: Switcher ghi token mới vào Windows Credential Manager (`WINCRED_WRITE`).
- **15:54:40.699**: Switcher gọi lệnh `taskkill /F /IM language_server.exe`. Antigravity ghi nhận:
  `[error] Language server crashed: exited with code 1. Attempting restart 1/3 in 2s...`
  -> **Cổng 54316 đã bị đóng và giải phóng ngay tại thời điểm này**.
- **15:54:43.299**: Antigravity khởi động thành công tiến trình `language_server.exe` **MỚI** với cấu hình:
  `[Auto-Restart] Port changed! Reloading all windows with URL: https://127.0.0.1:55762/`
  -> **Tiến trình mới mở cặp cổng: HTTPS 55762 và Connect-RPC HTTP 55763**.
- **15:54:48.628**: Switcher hoàn tất chờ 8 giây (`waitDuration = 8s`), tiến hành gọi `DiscoverLanguageServer()` để lấy thông tin kết nối và kiểm tra RAM.
- **15:54:48.631**: **SỰ CỐ XẢY RA**: `DiscoverLanguageServer()` trả về cổng cũ **54316** thay vì cổng mới **55763**. Switcher gửi HTTP POST tới `http://127.0.0.1:54316/user_status` và lập tức nhận phản hồi TCP RST (`connectex: No connection could be made because the target machine actively refused it`).

### 1.2. Tính chất tái diễn mang tính hệ thống
Lỗi này xuất hiện lặp đi lặp lại ở mọi lần chuyển tài khoản:
- Lần 1: Cổng chết bị gọi `50318` (cổng của phiên 14:25:09).
- Lần 2: Cổng chết bị gọi `54316` (cổng của phiên 15:27:03).
- Lần 3: Cổng chết bị gọi `55763` (cổng của phiên 15:54:43).
- Lần 4: Cổng chết bị gọi `55848` (cổng của phiên 15:54:54).
- Lần 5: Cổng chết bị gọi `55945` (cổng của phiên 15:56:05).

---

# PHẦN 2: NGUYÊN TẮC BẤT BIẾN & RÀNG BUỘC KIẾN TRÚC (INVARIANTS)

Mọi thay đổi mã nguồn của Coder bắt buộc phải tuân thủ nghiêm ngặt 5 nguyên tắc bất biến sau:

1. **8-Second Recovery Stabilization Invariant (BẤT BIẾN CỐT TỬ):**
   - Hằng số `waitDuration := 8 * time.Second` trong [`internal/manager/account_manager.go`](file:///g:/antigravity_auto_switch/internal/manager/account_manager.go#L787) và bước ngủ `sleeper(ctx, waitDuration)` trong [`internal/reloader/session_reloader.go`](file:///g:/antigravity_auto_switch/internal/reloader/session_reloader.go#L148) **PHẢI ĐƯỢC GIỮ NGUYÊN 100%**.
   - Cấm giảm thời gian chờ hoặc xóa bỏ bước này.
   - Test regression [`TestSwitchAccount_8SecondWaitAnd4StepWorkflow`](file:///g:/antigravity_auto_switch/internal/manager/manager_test.go#L23) phải tiếp tục pass 100%.

2. **Proactive Invalidation on Process Kill Invariant:**
   - Bất cứ khi nào tiến trình `language_server.exe` bị yêu cầu dừng hoặc kill qua `killer(ctx)`, hệ thống bắt buộc phải chủ động vô hiệu hóa toàn bộ bộ nhớ đệm Discovery ngay lập tức. Cấm giữ lại cache của tiến trình đã bị khai tử.

3. **Dead Process Rejection Invariant (Liveness Check):**
   - Cơ chế kiểm tra cache của Windows `DiscoverLanguageServer()` không được phép tin tưởng việc `OpenProcess` và `GetProcessTimes` trả về `ct == cachedCreation` là tiến trình còn sống.
   - Bắt buộc phải xác minh tiến trình chưa thoát (`exitCode == STILL_ACTIVE`) và/hoặc thăm dò cổng TCP (port probing) trước khi trả về cache hit.

4. **Reactive Cache Eviction Invariant:**
   - Nếu một kết nối API (`GetUserStatus`, `AuthLogout`) gặp lỗi kết nối cấp mạng (`net.OpError`, `connectex`, `connection refused`), hệ thống phải lập tức vô hiệu hóa cache đối với cổng đó để ngăn chặn việc tái sử dụng cổng lỗi ở các chu kỳ tiếp theo.

5. **Pure Win32 & Zero External CLI Overhead Invariant:**
   - Toàn bộ cơ chế kiểm tra liveness và dọn cache phải chạy bằng Win32 API thuần (`golang.org/x/sys/windows`) hoặc package tiêu chuẩn Go. Tuyệt đối không spawn thêm subprocess (`cmd.exe`, `powershell.exe`) gây nghẽn CPU.

---

# PHẦN 3: THIẾT KẾ KIẾN TRÚC CƠ CHẾ 3 TẦNG BẢO VỆ (3-TIER RESILIENCE)

Kiến trúc sư thiết kế giải pháp phân tầng khép kín đảm bảo loại bỏ hoàn toàn khả năng lấy nhầm cổng chết:

```mermaid
flowchart TD
    subgraph TANG_1["TẦNG 1: Chủ Động Xóa Cache Khi Kill (Proactive Invalidation)"]
        A["RecycleLanguageServer() thực thi killer(ctx)"] --> B["Gọi ngay connector.InvalidateDiscoveryCache(0, '')"]
        B --> C["BƯỚC CHỜ ĐỦ 8 GIÂY (waitDuration = 8s)"]
    end

    subgraph TANG_2["TẦNG 2: Kiểm Tra Liveness Thực Sự Khi Đọc Cache (Liveness Gate)"]
        C --> D["Hết 8s: Gọi DiscoverLanguageServer()"]
        D --> E{"Kiểm tra Cache còn hạn (TTL 10s)?"}
        E -->|Có| F["OpenProcess + GetExitCodeProcess(hProcess, &exitCode)"]
        F --> G{"exitCode == STILL_ACTIVE (259)?"}
        G -->|Đã chết| H["Vô hiệu hóa cache ngay lập tức"]
        G -->|Đang chạy| I{"Thăm dò cổng: ProbeHTTPPort(port, 300ms)?"}
        I -->|Thất bại| H
        I -->|Thành công| J["Trả về cache an toàn"]
        E -->|Không| K["Quét Toolhelp32Snapshot tìm tiến trình MỚI"]
        H --> K
        K --> L["Lấy PID mới + Dò cổng mới + Cập nhật Cache mới"]
    end

    subgraph TANG_3["TẦNG 3: Tự Phục Hồi Khi Gặp Lỗi Mạng (Reactive Eviction)"]
        J --> M["GetUserStatus(ctx) qua Client"]
        L --> M
        M --> N{"HTTP Call thành công?"}
        N -->|Thành công| O["RAM Check Thành Công (PASS F1)"]
        N -->|Lỗi Mạng (connectex)| P["Tự động InvalidateDiscoveryCache(c.info.Port, ...)"]
        P --> Q["Báo lỗi chuẩn xác, cache sạch cho lần bấm tiếp theo"]
    end
```

---

# PHẦN 4: ĐẶC TẢ CHI TIẾT THAY ĐỔI & HƯỚNG DẪN THI CÔNG TỪNG FILE

Coder thi công theo đúng 3 touchpoint kỹ thuật sau:

---

### Touchpoint 1: Xóa Cache Ngay Khi Kill Tiến Trình Trong Session Reloader
- **File:** [`internal/reloader/session_reloader.go`](file:///g:/antigravity_auto_switch/internal/reloader/session_reloader.go)
- **Vị trí:** Trong hàm `RecycleLanguageServer(ctx context.Context, waitDuration time.Duration) error` (khoảng dòng 139 - 144).
- **Quy tắc Kiến trúc (SPEC Rule):** Ngay sau khi lệnh `killer(ctx)` được thực thi (bất kể `killer` trả về lỗi hay thành công), bắt buộc gọi `connector.InvalidateDiscoveryCache(0, "")` trước khi bước vào hàm `sleeper(ctx, waitDuration)`.
- **Hướng dẫn thi công cụ thể (Implementation Suggestion):**
  ```go
  // 1. Terminate language_server.exe
  if err := killer(ctx); err != nil {
      return fmt.Errorf("failed executing taskkill for language_server.exe: %w", err)
  }

  // BẢO VỆ BỘ NHỚ ĐỆM: Xóa sạch cache discovery ngay khi tiêu diệt tiến trình
  connector.InvalidateDiscoveryCache(0, "")

  // 2. Wait for Antigravity parent (monitorLsCrashInternal) to respawn fresh instance
  if waitDuration <= 0 {
      waitDuration = 2 * time.Second
  }
  if err := sleeper(ctx, waitDuration); err != nil {
      return err
  }
  ```

---

### Touchpoint 2: Nâng Cấp Logic Kiểm Tra Trạng Thái Sống Của Tiến Trình Trong Discovery Cache
- **File:** [`internal/connector/discovery.go`](file:///g:/antigravity_auto_switch/internal/connector/discovery.go)
- **Vị trí:** Trong hàm `DiscoverLanguageServer()` (khoảng dòng 337 - 370).
- **Quy tắc Kiến trúc (SPEC Rule):**
  1. Khi kiểm tra cache hit với PID cũ, không được phép chỉ đối chiếu `ct == cachedCreation`.
  2. Bắt buộc phải kiểm tra mã thoát của tiến trình bằng `windows.GetExitCodeProcess(hProcess, &exitCode)`:
     - Giá trị của `exitCode` khi tiến trình còn đang chạy là `259` (tương đương hằng số Win32 `STILL_ACTIVE`).
     - Nếu `exitCode != 259`, hoặc hàm `GetExitCodeProcess` thất bại, tiến trình đã chết -> Xóa cache ngay lập tức (`discCache.info = nil`).
  3. Bắt buộc kiểm tra `et` (ExitTime) trong `GetProcessTimes`: Nếu `et.LowDateTime != 0 || et.HighDateTime != 0`, tiến trình đã có thời gian thoát -> Xóa cache.
  4. Trước khi trả về cache, thực hiện một lần probe nhanh `ProbeHTTPPort(cached.Port, 300*time.Millisecond)`: Nếu cổng từ chối kết nối, xóa cache và buộc quét mới.
  5. Trong khối `discFlight.Do`: Xóa bỏ đoạn code trả về cache rỗng mà không kiểm tra liveness (dòng 363-369).
- **Hướng dẫn thi công cụ thể (Implementation Suggestion):**
  Trong `DiscoverLanguageServer()`:
  ```go
  // 1. Kiểm tra cache còn hạn (TTL 10s)
  discCache.mu.Lock()
  cached := discCache.info
  cachedTime := discCache.cachedAt
  cachedCreation := discCache.creationTime
  if cached != nil && time.Since(cachedTime) <= 10*time.Second {
      isAlive := false
      if hProcess, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(cached.PID)); err == nil {
          var ct, et, kt, ut windows.Filetime
          var exitCode uint32
          if err := windows.GetProcessTimes(hProcess, &ct, &et, &kt, &ut); err == nil {
              if errExit := windows.GetExitCodeProcess(hProcess, &exitCode); errExit == nil {
                  // STILL_ACTIVE = 259 trong Win32 API
                  if exitCode == 259 && et.LowDateTime == 0 && et.HighDateTime == 0 && ct == cachedCreation {
                      isAlive = true
                  }
              }
          }
          _ = windows.CloseHandle(hProcess)
      }

      if isAlive && ProbeHTTPPort(cached.Port, 300*time.Millisecond) {
          discCache.mu.Unlock()
          return cached, nil
      }

      // Tiến trình đã chết hoặc cổng không còn phản hồi -> dọn dẹp cache
      discCache.info = nil
      discCache.cachedAt = time.Time{}
  }
  discCache.mu.Unlock()
  ```
  *(Đồng thời bên trong closure `discFlight.Do`, loại bỏ việc return cache thô mà không kiểm tra liveness, đảm bảo luồng bay qua `QueryLanguageServerProcesses` khi cache không hợp lệ)*.

---

### Touchpoint 3: Tự Động Giải Phóng Cache Khi Nhận Lỗi Kết Nối Cấp Mạng
- **File:** [`internal/connector/api_client.go`](file:///g:/antigravity_auto_switch/internal/connector/api_client.go)
- **Vị trí:** Trong vòng lặp gửi request của `GetUserStatus(ctx context.Context)` (khoảng dòng 109 - 113) và `AuthLogout(ctx context.Context)` (khoảng dòng 218 - 222).
- **Quy tắc Kiến trúc (SPEC Rule):** Khi hàm `postJSON` trả về lỗi không thể kết nối (network error / dial refused), lập tức gọi `InvalidateDiscoveryCache(c.info.Port, c.info.CSRFToken)` để cổng chết này không bị tái sử dụng.
- **Hướng dẫn thi công cụ thể (Implementation Suggestion):**
  ```go
  for _, endpoint := range DefaultUserStatusEndpoints {
      resp, err := c.postJSON(ctx, endpoint, emptyPayload)
      if err != nil {
          lastErr = err
          // Nếu không kết nối được tới cổng, lập tức xóa cache cổng này
          InvalidateDiscoveryCache(c.info.Port, c.info.CSRFToken)
          continue
      }
      ...
  ```

---

# PHẦN 5: KỊCH BẢN KIỂM THỬ CHẤP NHẬN (ACCEPTANCE TEST SUITE)

Coder phải xây dựng/cập nhật bộ unit test và acceptance test tự động để chứng minh tính đúng đắn:

### Test Case 1: `TestRecycleLanguageServer_InvalidatesDiscoveryCache`
- **Mục đích:** Xác minh `RecycleLanguageServer` xóa sạch cache ngay khi gọi `killer`.
- **Các bước:**
  1. Ghi một dữ liệu giả lập vào `discCache` (`info = &ServerInfo{PID: 9999, Port: 54316}`).
  2. Gọi `rel.RecycleLanguageServer(ctx, 10*time.Millisecond)`.
  3. Khẳng định `discCache.info == nil` sau khi hàm thực thi.

### Test Case 2: `TestDiscoverLanguageServer_RejectsDeadProcessInCache`
- **Mục đích:** Xác minh `DiscoverLanguageServer` từ chối trả về cache nếu tiến trình đã thoát (`exitCode != 259` hoặc `OpenProcess` thất bại).
- **Các bước:**
  1. Spawn một tiến trình ngắn hạn (như `cmd.exe /c exit 0`) để lấy PID và `CreationTime`. Chờ tiến trình kết thúc.
  2. Nạp PID này vào `discCache` kèm `cachedAt = time.Now()`.
  3. Gọi `DiscoverLanguageServer()`.
  4. Khẳng định hàm không trả về thông tin của tiến trình đã chết đó mà tiến hành quét tiến trình thực tế.

### Test Case 3: `TestSwitchAccount_8SecondWaitPreserved`
- **Mục đích:** Kiểm tra quy trình 8 giây không bị vi phạm hồi quy (Non-regression).
- **Chạy lệnh:**
  ```powershell
  go test -v -run TestSwitchAccount_8SecondWaitAnd4StepWorkflow ./internal/manager/...
  ```
- **Tiêu chí Đạt:** PASS với thời gian ngủ đúng `8 * time.Second`.

---

# PHẦN 6: KẾ HOẠCH BÀN GIAO & PHÂN CÔNG THI CÔNG

- **Báo cáo này:** Thuộc sở hữu của `System Architect (Strategic Lane)`.
- **Trạng thái:** Đã thống nhất với Human Owner.
- **Quy trình tiếp theo:**
  1. Kiến trúc sư bàn giao Đặc tả này để Human Owner xem xét toàn bộ lần cuối.
  2. Khi Human Owner chính thức phê duyệt và phát lệnh gửi báo cáo, Kiến trúc sư sẽ gửi gói bàn giao về CEO Session (`9ec46f35-a50c-4580-86d0-b92f3725848e`).
  3. CEO sẽ phân công Lane `Coder` tiến hành thi công code theo đúng từng bước của bản đặc tả này.
