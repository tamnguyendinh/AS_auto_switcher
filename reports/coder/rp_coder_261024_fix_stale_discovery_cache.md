# BÁO CÁO THI CÔNG CHI TIẾT (CODER IMPLEMENTATION REPORT)
## Khắc Phục Triệt Để Lỗi Stale Discovery Cache & Cổng Chết Khi Chuyển Tài Khoản

- **Tác giả:** Coder Lane
- **Ngày thi công:** 2026-10-24
- **Thẩm quyền kiến trúc (SSOT Authority):** [`reports/system-architect/rp_system-architect_261024_165000_by_antigravity_fix_stale_discovery_cache_and_connection_refusal.md`](file:///g:/antigravity_auto_switch/reports/system-architect/rp_system-architect_261024_165000_by_antigravity_fix_stale_discovery_cache_and_connection_refusal.md) (Commit `48da555`)
- **Tình trạng:** HOÀN TẤT THI CÔNG 100% (READY FOR QA)

---

## 1. BẢN ĐỒ DÒNG BẤT BIẾN (INVARIANT FAMILY MAP)

- **Tên họ bất biến (Family Name):** `Stale Discovery Cache & Connection Refusal Protection`
- **Tài liệu nguồn chân lý tối cao (SSOT Authority):** `rp_system-architect_261024_165000_by_antigravity_fix_stale_discovery_cache_and_connection_refusal.md`
- **Các bề mặt runtime đồng sở hữu bất biến (Sibling Runtime Surfaces):**
  1. `internal/reloader/session_reloader.go`: `RecycleLanguageServer` quản lý vòng đời tiêu diệt `language_server.exe` và tái sinh tiến trình.
  2. `internal/connector/discovery.go`: `DiscoverLanguageServer`, `InvalidateDiscoveryCache`, `GetCachedServerInfo`, `SetDiscoveryCacheForTesting`.
  3. `internal/connector/api_client.go`: `GetUserStatus`, `AuthLogout`, `TriggerReauth` tương tác trực tiếp với API của Language Server qua loopback HTTP.
- **Đường dẫn/Cơ chế cũ bị cấm (Forbidden Legacy Fallback):**
  - Cấm tin tưởng mù quáng vào `ct == cachedCreation` mà không xác minh tiến trình còn sống (`exitCode == STILL_ACTIVE (259)`).
  - Cấm giữ cache cũ sau khi lệnh kill tiến trình (`killer(ctx)`) đã được phát ra.
  - Cấm tái sử dụng cổng khi kết nối mạng đã bị từ chối (`connectex / connection refused`).
  - Cấm cắt giảm hay sửa đổi hằng số thời gian chờ 8 giây (`waitDuration := 8 * time.Second`).
- **Hiện trạng kiểm thử & kế hoạch cũ (Stale Tests / Helpers Updated):**
  - Mở rộng `internal/reloader/reloader_test.go` với test case kiểm chứng chủ động xóa cache ngay khi kill.
  - Mở rộng `internal/connector/discovery_test.go` với test case kiểm tra từ chối tiến trình đã chết trong cache và kiểm tra reactive eviction khi cổng bị từ chối kết nối.
  - Chạy toàn bộ suite regression `internal/manager/manager_test.go` chứng minh bất biến 8 giây được bảo tồn nguyên vẹn 100%.
- **Bề mặt chưa được kiểm chứng (Residual Unverified Surfaces):** `none`

---

## 2. CHI TIẾT THI CÔNG THEO 3 TOUCHPOINT KỸ THUẬT

### Touchpoint 1: Chủ Động Xóa Cache Khi Kill Tiến Trình (`session_reloader.go`)
- **Vị trí:** [`internal/reloader/session_reloader.go`](file:///g:/antigravity_auto_switch/internal/reloader/session_reloader.go#L137-L149)
- **Thực thi:**
  - Thực hiện lệnh `killErr := killer(ctx)`.
  - Ngay lập tức gọi `connector.InvalidateDiscoveryCache(0, "")` để dọn sạch toàn bộ bộ nhớ đệm Discovery, triệt tiêu khả năng trả về cổng cũ của tiến trình đã chết.
  - Kiểm tra `killErr` và bước vào `sleeper(ctx, waitDuration)`.
  - Bảo tồn nguyên vẹn 100% bước chờ an toàn `waitDuration := 8 * time.Second`.

### Touchpoint 2: Kiểm Tra Liveness Win32 STILL_ACTIVE & Thăm Dò Cổng (`discovery.go`)
- **Vị trí:** [`internal/connector/discovery.go`](file:///g:/antigravity_auto_switch/internal/connector/discovery.go#L330-L380)
- **Thực thi:**
  - Thêm `GetCachedServerInfo() *ServerInfo` và `SetDiscoveryCacheForTesting(...)` hỗ trợ kiểm thử và quan sát trạng thái cache an toàn với đa luồng (`discCache.mu`).
  - Trong `DiscoverLanguageServer()`:
    1. Lấy thông tin cache dưới khóa `discCache.mu.Lock()`.
    2. Nếu cache còn hạn (< 10s), mở tiến trình với `windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(cached.PID))`.
    3. Xác minh tính toàn vẹn 4 tầng:
       - `windows.GetProcessTimes`: Kiểm tra `ct == cachedCreation` và thời gian thoát `et.LowDateTime == 0 && et.HighDateTime == 0`.
       - `windows.GetExitCodeProcess`: Kiểm tra mã thoát `exitCode == 259` (`STILL_ACTIVE`).
       - `ProbeHTTPPort(cached.Port, 300*time.Millisecond)`: Thăm dò loopback TCP HTTP cổng phản hồi 200.
    4. Nếu bất kỳ điều kiện nào không thỏa mãn (tiến trình đã chết hoặc cổng chết): Lập tức dọn dẹp cache `discCache.info = nil; discCache.cachedAt = time.Time{}`.
    5. Loại bỏ đoạn code trả về cache thô không kiểm tra liveness bên trong `discFlight.Do`, đảm bảo khi cache bị vô hiệu hóa luồng luôn quét tiến trình mới qua `QueryLanguageServerProcesses()`.

### Touchpoint 3: Tự Động Giải Phóng Cache Khi Gặp Lỗi Kết Nối Mạng (`api_client.go`)
- **Vị trí:** [`internal/connector/api_client.go`](file:///g:/antigravity_auto_switch/internal/connector/api_client.go#L108-L115, #L218-L225, #L250-L257)
- **Thực thi:**
  - Tại các vòng lặp gửi HTTP POST trong `GetUserStatus`, `AuthLogout`, và `TriggerReauth`:
  - Khi `c.postJSON` trả về lỗi kết nối cấp mạng (`connectex`, dial refused), lập tức gọi `InvalidateDiscoveryCache(c.info.Port, c.info.CSRFToken)`.
  - Giúp hệ thống tự phục hồi ngay tức thì, giải phóng cổng chết để lần quét tiếp theo lấy đúng cổng mới.

---

## 3. KẾT QUẢ KIỂM THỬ XÁC MINH (VERIFICATION EVIDENCE)

### 3.1. Unit Test & Acceptance Tests
Chạy kiểm thử các package thuộc Touch-Map:
```powershell
go test -v ./internal/reloader/...
go test -v ./internal/connector/...
go test -v -run TestSwitchAccount_8SecondWaitAnd4StepWorkflow ./internal/manager/...
```

**Bằng chứng chi tiết:**
- `TestRecycleLanguageServer_InvalidatesDiscoveryCache`: **PASS** (Chứng minh discovery cache bị xóa sạch ngay trước khi bước vào sleeper).
- `TestDiscoverLanguageServer_RejectsDeadProcessInCache`: **PASS** (Spawn tiến trình cmd chết, nạp vào cache, gọi `DiscoverLanguageServer()` -> tiến trình chết bị từ chối và trục xuất khỏi cache).
- `TestApiClient_ReactiveCacheEvictionOnConnectionRefused`: **PASS** (Client trỏ tới cổng đóng, gọi `GetUserStatus` và `AuthLogout` -> nhận lỗi và cache cổng đó bị xóa sạch tự động).
- `TestSwitchAccount_8SecondWaitAnd4StepWorkflow`: **PASS** (Thời gian chờ 8 giây và 4 bước chuyển tài khoản hoàn toàn bảo tồn nguyên vẹn 100%).

### 3.2. Repo-wide Build & Pre-flight Verification
Chạy lệnh đóng gói chuẩn repository:
```powershell
powershell -ExecutionPolicy Bypass -File scripts\build_windows.ps1
```
**Kết quả:**
- `[2/4] Running pre-flight tests & vet...`: **PASS** (Tất cả test suite `cmd/switcher`, `internal/automation`, `internal/connector`, `internal/manager`, `internal/reloader`, `internal/store`, `internal/telemetry`, `internal/ui` đều đạt chuẩn).
- `[3/4] Compiling target binaries...`: Compiled PE resource `rsrc_windows_amd64.syso` (13490 bytes) & Binary GUI `bin/antigravity-switch.exe` (27.91 MB, 6138 ms).
- `[4/4] Executing quick verification...`: Embedded PE application icon verified (32x32).
- `⚡ BUILD SUCCESSFUL` (Exit Code 0).

---

## 4. DANH MỤC FILE THAY ĐỔI (TOUCH-MAP CONFORMANCE)

Tất cả các thay đổi tuân thủ nghiêm ngặt 100% Touch-Map đã khóa, không chạm bất kỳ file nào ngoài phạm vi:
1. `internal/reloader/session_reloader.go` (+7, -2 dòng): Gọi `InvalidateDiscoveryCache(0, "")` khi kill.
2. `internal/connector/discovery.go` (+39, -22 dòng): Liveness gate với Win32 `GetExitCodeProcess` 259, `et`, `ct`, port probe, helper functions.
3. `internal/connector/api_client.go` (+6, -0 dòng): Reactive eviction khi dial refused trong `GetUserStatus`, `AuthLogout`, `TriggerReauth`.
4. `internal/reloader/reloader_test.go` (+61, -0 dòng): Mở rộng test case `TestRecycleLanguageServer_InvalidatesDiscoveryCache`.
5. `internal/connector/discovery_test.go` (+128, -0 dòng): Mở rộng test cases `TestDiscoverLanguageServer_RejectsDeadProcessInCache` và `TestApiClient_ReactiveCacheEvictionOnConnectionRefused`.
6. `reports/coder/rp_coder_261024_fix_stale_discovery_cache.md`: Báo cáo thi công chính thức.

---

## 5. KẾT LUẬN & SẴN SÀNG BÀN GIAO (VERDICT)

- **Đánh giá:** **[VERDICT: READY_FOR_QA]**
- Lỗi stale discovery cache và kết nối cổng cũ đã chết được triệt tiêu hoàn toàn bởi hệ thống phòng thủ 3 tầng khép kín.
- Bất biến 8 giây được giữ nguyên vẹn 100%.
- Toàn bộ kết quả đã sẵn sàng commit và bàn giao sang Supervisor / QA.
