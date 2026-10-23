# BÁO CÁO ĐẶC TẢ KIẾN TRÚC (ARCHITECTURE SPECIFICATION REPORT)
## TỐI ƯU HÓA TỐC ĐỘ KHỞI ĐỘNG VÀ BỘ NHỚ RAM ỨNG DỤNG SWITCHER (BẢN CHUẨN HÓA BẢO TOÀN NGHIỆP VỤ)

- **Mã tài liệu:** `rp_architect_20261023_200000_startup_and_memory_optimization_spec`
- **Tác giả:** System Architect Lane
- **Ngày lập:** 2026-10-23 (Cập nhật phiên bản R2 theo chỉ đạo của Owner)
- **Thẩm quyền:** Human Owner
- **Đối tượng thụ hưởng:** Coder Lane, QA Lane
- **Trạng thái:** **`VERDICT: PASS / READY_FOR_CODER`**

---

## 1. BỐI CẢNH & MỤC TIÊU (CONTEXT & OBJECTIVES)

### 1.1. Hiện trạng đo đạc thực tế (Baseline Analysis)
Khảo sát trực tiếp trên mã nguồn và tiến trình thực tế (`antigravity-switch.exe` - PID 16020):
1. **Tốc độ khởi động:** Trễ từ **1.5s - 2.5s** trước khi giao diện sẵn sàng thao tác.
   - *Nguyên nhân cốt lõi:* `QueryLanguageServerProcesses` trong `internal/connector/discovery.go` đang gọi `powershell.exe` thông qua `exec.Command`. Trên Windows, mỗi lần spawn PowerShell tốn **500ms - 1200ms**. Trong chu kỳ `GetAccountStates()`, do thiếu bộ nhớ đệm dùng chung, PowerShell bị gọi **2 lần liên tiếp**. Thêm vào đó, hàm fallback `discoverPortsViaNetstat` cũng đang gọi tiến trình con `netstat.exe`.
   - Sau khi mở cửa sổ Fyne, có goroutine chạy nền `time.Sleep(50ms)` gọi `SyncVisualScaleAndTextures()` ép Fyne re-theme và re-render lại toàn bộ cây widget lần thứ 2.
2. **Bộ nhớ RAM:** Working Set đo được trong Task Manager dao động **~134 MB – 182.4 MB** (Private Memory ~259.5 MB).
   - *Nguyên nhân cốt lõi:* Fyne render qua OpenGL (`nvoglv64.dll`, `OPENGL32.dll`), driver card đồ họa nạp vào tiến trình giữ cố định **~70MB – 90MB** (framebuffers, shader compilers, texture glyph cache).
   - Go runtime (`mheap`) giữ lại các trang nhớ sau khi giải mã JSON/config khởi động (`GOGC=100`), không tự động trả về cho OS.

### 1.2. Ba Mục tiêu Tối ưu hóa (Triple Targets)
1. **Mở giao diện tức thì (Startup Latency mục tiêu < 300ms):** Render khung giao diện và dữ liệu cache đĩa ngay trong frame đầu tiên; loại bỏ 100% subprocess bên ngoài.
2. **Giảm bộ nhớ Go giữ lại (Go Heap Retention):** Thu hồi trang nhớ nhàn rỗi sau khi hoàn tất khởi động; thử nghiệm các mức `GOMEMLIMIT`.
3. **Giảm RAM Resident khi chạy nền (Background Working Set Trim):** Khi ứng dụng ẩn vào System Tray, giải phóng các trang bộ nhớ vật lý nhàn rỗi mà không làm ảnh hưởng đến luồng giám sát nền (AutoSwitch & Telemetry).

---

## 2. NGUYÊN TẮC BẤT BIẾN KIẾN TRÚC (ARCHITECTURAL INVARIANTS)

1. **`Invariant-Zero-Subprocess-Discovery`:**
   Tuyệt đối KHÔNG gọi `powershell.exe`, `netstat.exe`, `cmd.exe` hay bất kỳ tiến trình con bên ngoài nào để truy vấn tiến trình hoặc mạng. Toàn bộ thao tác giám sát hệ thống phải sử dụng Win32 / NT API thuần trong Go (`windows.CreateToolhelp32Snapshot`, `ntdll.dll::NtQueryInformationProcess`, `iphlpapi.dll::GetExtendedTcpTable`).
2. **`Invariant-Lifecycle-Safe-Caching`:**
   Bộ nhớ đệm Discovery phải gắn liền với cặp định danh `(PID, ProcessCreationTime)`. Không bao giờ tin tưởng riêng rẽ một mình `PID` để ngăn chặn lỗi tái sử dụng PID (PID recycling) của Windows. Khi thay thế cache, bắt buộc phải đóng Handle cũ. TTL 10 giây phải có hiệu lực thực sự: quá hạn bắt buộc phải làm mới thông tin dù PID/CreationTime chưa đổi.
3. **`Invariant-Action-Permission-Integrity`:**
   Trạng thái `READY` tuyệt đối KHÔNG được hiểu là mở khóa mù quáng nút Chuyển tài khoản, đồng thời **TUYỆT ĐỐI BẢO TOÀN NGUYÊN VẸN LUỒNG CHẠY HIỆN CÓ CỦA NÚT XÓA `[ 🗑 ]`**:
   - **Quy tắc Bất biến của Nút Xóa (Trash Button):**
     - Nút Xóa trên mọi thẻ không hoạt động (Inactive: bao gồm Status ON, Status OFF và Cần OAuth) **LÚC NÀO CŨNG BẬT VÀ HIỂN THỊ (`Show()` và `Enable()`)**.
     - Nút Xóa trên thẻ đang hoạt động (`IsActive == true`): **ẨN HOÀN TOÀN (`Hide()`)**.
     - Toàn bộ quá trình tối ưu khởi động / bộ nhớ tuyệt đối KHÔNG can thiệp, không vô hiệu hóa hay làm phá vỡ luồng hoạt động này của nút Xóa.
   - **Quy tắc của Nút Chuyển tài khoản (Switch Button):**
     - Trong lúc đang xác minh (`VERIFYING`): Nút Chuyển tạm thời bị khóa (`Disable()`) để tránh race condition.
     - Khi xác minh hoàn tất (`READY`): Tính lại quyền cho nút Chuyển:
       - Tài khoản có trạng thái `status == "off"`: Khóa nút Chuyển (`Disable()`), hiển thị badge `[OFF]`.
       - Tài khoản thiếu token/credential (`StateUnauth`): Khóa nút Chuyển, hiển thị `[Cần OAuth]`.
       - Tài khoản đang hoạt động (`IsActive == true`): Khóa nút Chuyển, hiển thị badge `[✔ Dùng]`.
       - Chỉ tài khoản `StateReady` (Status == "on", có token hợp lệ, không active) mới được mở khóa nút Chuyển (`Enable()`).
4. **`Invariant-Clear-Limited-Connectivity`:**
   Định nghĩa rõ ràng trạng thái `LIMITED_CONNECTIVITY`: Khi Language Server không chạy hoặc lỗi kết nối, nhưng WinCred và AccountStore đọc thành công -> Hệ thống tự động chuyển sang chế độ **WinCred Fallback Mode**. Người dùng vẫn được phép thực hiện chuyển tài khoản qua WinCred. Chỉ chuyển sang trạng thái lỗi kết nối hạn chế (có nút "Thử lại") khi backend không thể đọc được cả WinCred lẫn AccountStore. Cảnh báo lỗi từng phần (như quota lỗi 1 tài khoản) không được phép khóa toàn bộ app.
5. **`Invariant-Generation-UI-Callback-Enforcement`:**
   Mọi thao tác làm mới (`Refresh`), chuyển tài khoản thủ công (`SwitchAccount`), và chuyển tự động (`AutomationController`) đều phải tăng `generationID` để làm mất hiệu lực toàn bộ các kết quả xác minh cũ. **Việc kiểm tra `generationID` bắt buộc phải diễn ra ngay bên trong callback cập nhật UI (`fyne.Do()`)**, ngay trước khi ghi trạng thái hoặc thay đổi nút. Callback lỗi và phần cleanup của lượt cũ tuyệt đối không được mở khóa nút hoặc xóa trạng thái của lượt mới.
6. **`Invariant-Backend-Switch-Delete-Mutual-Exclusion`:**
   Trạng thái nút bấm trên UI chỉ là lớp phủ giao diện. Tại tầng backend, `AccountManager` bắt buộc phải điều phối đồng bộ (mutual exclusion) giữa luồng Chuyển tài khoản và luồng Xóa tài khoản (`RemoveAccount`). Tại thời điểm bắt đầu xóa tệp, `RemoveAccount` phải kiểm tra lại `activeEmail` trong vùng khóa bảo vệ để đảm bảo tài khoản không thể trở thành active giữa hai bước kiểm tra và xóa.
7. **`Invariant-Preserve-Window-Lifecycle`:**
   Giữ nguyên 100% hành vi hiện có của nút `X` (thoát ứng dụng hoàn toàn qua `a.Close()` + `a.app.Quit()`) và nút thu nhỏ `—` (thu nhỏ xuống Windows Taskbar). Lệnh Trim Working Set chỉ được gắn vào thao tác **"Ẩn vào khay hệ thống"** được xác định rõ ràng (qua menu Tray); tuyệt đối cấm tự ý đổi nút đóng `X` thành ẩn vào khay. Trim thất bại không được cản trở việc ẩn/hiện cửa sổ.

---

## 3. BẢN ĐỒ CAN THIỆP MÃ NGUỒN (TOUCH-MAP)

| Tệp mã nguồn | Trách nhiệm can thiệp | Phạm vi thay đổi chi tiết |
|---|---|---|
| `internal/connector/discovery.go` | Loại bỏ 100% subprocess + Singleflight Cache | Viết lại `QueryLanguageServerProcesses` bằng Win32 Toolhelp32, đọc CommandLine qua `NtQueryInformationProcess`, bỏ `discoverPortsViaNetstat`, thêm cache vòng đời (PID, CreationTime, TTL 10s, singleflight, context cancellation) |
| `internal/manager/account_manager.go` | Điều phối đồng bộ Switch-Delete + Chống Stale Discovery | Bổ sung khóa điều phối giữa `SwitchAccount` và `RemoveAccount`, double-check active trước khi xóa tệp; dùng chung discovery cache chuẩn |
| `internal/ui/app.go` | UI State Machine + Generation ID trong `fyne.Do()` + Bỏ Re-theme thừa | Quản lý `generationID`, kiểm tra generation ngay trong `fyne.Do()`, tính ma trận quyền chi tiết tại `READY`, bỏ goroutine 50ms `SyncVisualScaleAndTextures`, 1-shot `FreeOSMemory` |
| `internal/ui/window.go` | Phản ánh đúng quyền nút theo từng tài khoản | Cập nhật logic render nút Chuyển/Xóa theo điều kiện nghiệp vụ thực tế |
| `internal/ui/tray.go` & `single_instance_windows.go` | Thêm lệnh "Ẩn vào khay" & Trim Working Set an toàn | Thêm mục menu Tray "Ẩn giao diện vào khay", gắn `TrimProcessWorkingSet()`, bảo đảm không ảnh hưởng nút X và nút minimize |

---

## 4. CHI TIẾT ĐẶC TẢ KỸ THUẬT (DETAILED SPECIFICATION)

### 4.1. Bước 1: Win32 API Discovery, Singleflight, Cache Vòng đời & Xóa bỏ Netstat

#### A. Truy vấn tiến trình qua Win32 API (`internal/connector/discovery.go`)
1. **Duyệt tiến trình:** Dùng `windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)` kết hợp `Process32First` / `Process32Next` để tìm `language_server.exe` trong kernel bộ nhớ (thời gian chạy < 1ms).
2. **Đọc CommandLine an toàn:**
   - Mở tiến trình bằng `windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pe.ProcessID)`.
   - Lấy `CreationTime` qua `windows.GetProcessTimes(hProcess, &creationTime, &exitTime, &kernelTime, &userTime)`.
   - Gọi `ntdll.dll::NtQueryInformationProcess` với `ProcessInformationClass = 60` (`ProcessCommandLineInformation`). Kernel Windows sao chép chuỗi CommandLine vào struct `UNICODE_STRING` của caller an toàn, không rủi ro crash nếu tiến trình thoát. Nếu không gọi được class 60, fallback sang đọc PEB có kiểm tra con trỏ hợp lệ.
   - Nếu `OpenProcess` thất bại hoặc đọc thất bại (tiến trình thoát, quyền truy cập bị hạn chế), trả về trạng thái chưa kết nối và thử lại ở chu kỳ sau; **tuyệt đối không yêu cầu quyền Admin/UAC Elevation**.
   - Luôn đóng Handle tiến trình sau khi đọc.
3. **Loại bỏ hoàn toàn `discoverPortsViaNetstat`:**
   - Xóa bỏ toàn bộ hàm gọi `exec.Command("netstat", ...)`. Việc phát hiện cổng listening loopback phải 100% sử dụng Win32 API `discoverPortsViaWin32` (`iphlpapi.dll::GetExtendedTcpTable`) đã có sẵn.

#### B. Cơ chế Caching, Singleflight và Bảo vệ Invalidation
```go
type serverDiscoveryCache struct {
    mu           sync.Mutex
    info         *ServerInfo
    creationTime windows.Filetime
    cachedAt     time.Time
}

var (
    discCache  serverDiscoveryCache
    discFlight singleflight.Group
)
```
1. **Hiệu lực TTL 10 giây:**
   - Khi có yêu cầu discovery, kiểm tra: Nếu `time.Since(cachedAt) > 10*time.Second`, cache bị coi là hết hạn và bắt buộc phải quét lại, bất kể PID và CreationTime có thay đổi hay không.
2. **Kiểm tra vòng đời khi còn TTL:**
   - Nếu trong vòng 10s: Kiểm tra nhanh `windows.OpenProcess` và đối chiếu `GetProcessTimes`. Nếu `CreationTime` khớp 100% thì trả về cache ngay lập tức. Nếu tiến trình đã thoát hoặc CreationTime khác, hủy cache và quét lại.
3. **Phạm vi dùng chung thống nhất:**
   - `AccountManager`, `telemetry.TelemetryPool`, và `automation.AutomationController` bắt buộc phải gọi chung qua `connector.DiscoverLanguageServer()`. Tuyệt đối không lưu cache endpoint cục bộ riêng rẽ.
4. **Hỗ trợ `context.Context` (Deadline / Cancellation):**
   - Lời gọi gom luồng qua `singleflight` phải lắng nghe `ctx.Done()`. Nếu context của caller bị timeout hoặc cancel, goroutine caller phải lập tức thoát ra, không bị treo chờ discovery.
5. **Bảo vệ chống Invalidate nhầm (Stale Invalidation Protection):**
   - Khi một request RPC thất bại, hàm `InvalidateDiscoveryCache(failedPort int, failedCSRF string)` chỉ được xóa cache NẾU endpoint bị lỗi trùng khớp với endpoint đang lưu trong cache. Nếu cache đã được làm mới trước đó bởi một lượt discovery khác, không được phép xóa cache mới.
   - Kết quả discovery chạy chậm (stale result) không được phép ghi đè vào cache nếu đã có lệnh invalidate xảy ra sau thời điểm bắt đầu quét.
6. **Không tự ý gửi lại RPC có tác dụng phụ:** Lệnh invalidate chỉ làm mới thông tin kết nối cho các lần gọi sau, tuyệt đối không tự động retry các RPC có tác dụng phụ (như switch hoặc logout).
7. **Không chèn HTTP Probe trước mọi API call:** Bỏ bước HTTP probe nhân tạo lặp lại; để request nghiệp vụ thật tự động xác thực kết nối.

---

### 4.2. Bước 2: UI Hydration, Generation Enforcement & Đồng bộ Switch-Delete

#### A. Generation ID: Điểm tăng và Điểm kiểm tra chuẩn xác (`internal/ui/app.go`)
```go
type App struct {
    ...
    refreshGen atomic.Uint64
    uiState    atomic.Int32 // 0: StateVerifying, 1: StateReady, 2: StateLimited
}
```
* **Các điểm bắt buộc tăng `refreshGen`:**
  1. Khi người dùng bấm "Làm mới danh sách" (`Refresh()`).
  2. Khi bắt đầu thao tác chuyển tài khoản thủ công (`SwitchAccount()`).
  3. Khi `AutomationController` kích hoạt tự động chuyển tài khoản (`AutoSwitch`).
* **Điểm kiểm tra bắt buộc:**
  * BẮT BUỘC kiểm tra `a.refreshGen.Load() == currentGen` **ngay bên trong thân hàm `fyne.Do(func() { ... })`**, ngay trước khi cập nhật `a.overview`, ghi đè state hay cập nhật widget.
  * Nếu `refreshGen` đã khác: Lập tức `return`, hủy bỏ toàn bộ kết quả cũ.
  * Trong khối `defer / cleanup` của goroutine xác minh cũ: Phải kiểm tra `refreshGen`. Nếu đã là lượt cũ, tuyệt đối KHÔNG được mở khóa nút bấm hoặc xóa trạng thái/thông báo của lượt mới.

#### B. Quy tắc phân quyền nút bấm chi tiết tại `READY`
Khi xác minh hoàn tất (`UIStateReady`), hệ thống tính toán trạng thái nút theo đúng ma trận nghiệp vụ và **bảo toàn nguyên vẹn luồng chạy của nút Xóa**:

1. **Bảo toàn 100% Hành vi Nút Xóa `[ 🗑 ]` (Trash Button):**
   - Trên mọi thẻ Inactive (bao gồm `Status == "on"`, `Status == "off"`, hoặc `StateUnauth` [Cần OAuth]): Nút Xóa **LÚC NÀO CŨNG BẬT VÀ HIỂN THỊ (`w.deleteButton.Show()`, `w.deleteButton.Enable()`)**, cho phép người dùng click mở dialog xác nhận xóa bất cứ lúc nào.
   - Trên thẻ Active (`IsActive == true`): Nút Xóa **ẨN HOÀN TOÀN (`w.deleteButton.Hide()`)**, cấm xóa tài khoản đang hoạt động.
   - Tuyệt đối KHÔNG được khóa, vô hiệu hóa hay làm thay đổi luồng hoạt động sẵn có này của nút Xóa.

2. **Quy tắc Nút Chuyển tài khoản `[Chuyển]` (Switch Button):**
   - Trong thời gian `UIStateVerifying`: Nút Chuyển tạm thời bị khóa (`Disable()`) để chống race condition.
   - Khi chuyển sang `UIStateReady`:
     - `acc.Status == "off"`: Nút Chuyển bị khóa (`Disable()`), hiển thị nhãn `[OFF]`.
     - `acc.State == model.StateUnauth`: Nút Chuyển bị khóa (`Disable()`), hiển thị nhãn `[Cần OAuth]`.
     - `acc.IsActive == true`: Nút Chuyển bị khóa (`Disable()`), hiển thị nhãn `[✔ Dùng]`.
     - `acc.State == model.StateReady` (Status == "on", có token hợp lệ, không active): Nút Chuyển được mở khóa (`Enable()`), hiển thị `[Chuyển]`.

#### C. Định nghĩa minh bạch `LIMITED_CONNECTIVITY`
* **Trường hợp WinCred Fallback:**
  - Nếu `LanguageServer` không hoạt động hoặc không tìm thấy cổng, nhưng `store.ListAccounts()` và `wincred` đọc được:
  - Đây là hoạt động bình thường ở chế độ dự phòng. `overview.LanguageServerRunning = false`.
  - Footer bar hiển thị: `"Antigravity: Không hoạt động | Dự phòng WinCred"`.
  - Các nút "Chuyển tài khoản" vẫn hoạt động bình thường qua cơ chế nạp WinCred trực tiếp.
* **Trường hợp lỗi kết nối thực sự (`UIStateLimited`):**
  - Chỉ chuyển sang `UIStateLimited` khi không thể đọc được cả kho tài khoản lẫn WinCred (lỗi I/O ổ đĩa nghiêm trọng hoặc timeout toàn diện).
  - Lúc này hiển thị trên Status Bar: `"⚠️ Không thể kết nối hoặc nạp danh sách tài khoản"` kèm nút **"Thử lại"**.

#### D. Loại bỏ Re-theme và Re-render thừa sau `Show()`
* **XÓA BỎ** hoàn toàn goroutine `time.Sleep(50ms)` và lời gọi `SyncVisualScaleAndTextures()` sau `win.Show()` trong file `internal/ui/app.go`.
* Palette, Theme và Window size được cấu hình đầy đủ một lần duy nhất trong `Init()`.
* Giữ nguyên hàm `SyncVisualScaleAndTextures()` để phục vụ người dùng đổi Theme Sáng/Tối chủ động.

---

### 4.3. Bước 3: Điều phối Đồng bộ Backend Chuyển - Xóa Tài khoản (`internal/manager/account_manager.go`)

Tại `AccountManager`, bổ sung cơ chế khóa điều phối để loại bỏ triệt để race condition:
```go
type AccountManager struct {
    ...
    opMu sync.Mutex // Khóa điều phối độc quyền giữa Switch và Delete
}
```
1. **Trong `SwitchAccount`:** Bắt buộc nắm giữ `m.opMu.Lock()` (hoặc thông qua `SwitchGate`) trong toàn bộ quá trình hot-swap tài khoản.
2. **Trong `RemoveAccount` (`account_manager.go:215`):**
   ```go
   func (m *AccountManager) RemoveAccount(ctx context.Context, email string) error {
       normEmail := strings.ToLower(strings.TrimSpace(email))
       if normEmail == "" {
           return store.ErrInvalidEmail
       }

       // Khóa điều phối: Chặn mọi thao tác chuyển tài khoản diễn ra đồng thời
       m.opMu.Lock()
       defer m.opMu.Unlock()

       // Double-check: Kiểm tra lại activeEmail ngay trong vùng khóa bảo vệ
       activeEmail, _ := m.GetCurrentActiveEmail(ctx)
       if activeEmail != "" && strings.EqualFold(activeEmail, normEmail) {
           return ErrCannotDeleteActiveAccount
       }

       // Tiến hành xóa tệp tài khoản an toàn
       ...
   }
   ```
* Quy tắc này đảm bảo một tài khoản không thể nào chuyển sang trạng thái Active ở giữa hai bước kiểm tra và xóa tệp!

---

### 4.4. Bước 4: Quản lý Go Heap (A/B Test `FreeOSMemory` & `GOMEMLIMIT`)

1. **Khảo sát `FreeOSMemory()` 1 lần duy nhất:**
   * Trong `internal/ui/app.go`, dùng `sync.Once`:
     ```go
     var steadyStateOnce sync.Once
     ```
   * Khi ứng dụng đạt trạng thái `UIStateReady` lần đầu tiên sau khi khởi động:
     ```go
     steadyStateOnce.Do(func() {
         go func() {
             time.Sleep(200 * time.Millisecond)
             debug.FreeOSMemory()
         }()
     })
     ```
   * Đo đạc A/B so sánh với bản baseline không gọi, xác nhận mức RAM giảm và đảm bảo không gây khựng giao diện.
2. **Khảo sát `GOMEMLIMIT` qua biến môi trường:**
   * Thử nghiệm khởi chạy với các mức: `GOMEMLIMIT=64MiB`, `96MiB`, `128MiB`.
   * Ghi nhận số liệu GC pause time và độ mượt của UI, không thêm cài đặt tùy biến vào GUI.

---

### 4.5. Bước 5: Thao tác "Ẩn vào Tray" & Trim Working Set An Toàn

1. **Bảo toàn nguyên vẹn hành vi cửa sổ hiện tại:**
   * **Nút X và Close Intercept:** Giữ nguyên 100% chức năng đóng và thoát app (`a.Close()` -> `a.app.Quit()`). Tuyệt đối KHÔNG sửa nút X thành ẩn cửa sổ.
   * **Nút thu nhỏ `—`:** Giữ nguyên 100% chức năng thu nhỏ xuống Taskbar (`MinimizeNativeWindow()`).
2. **Thao tác "Ẩn vào khay hệ thống" riêng biệt:**
   * Trong menu Tray (`internal/ui/tray.go`), bổ sung rõ ràng mục menu:
     ```go
     fyne.NewMenuItem("Ẩn giao diện vào khay", func() {
         if a.win != nil {
             a.win.Hide()
             TrimProcessWorkingSet()
         }
     })
     ```
   * Hàm `TrimProcessWorkingSet()` trong `single_instance_windows.go`:
     ```go
     func TrimProcessWorkingSet() {
         hProcess := uintptr(windows.CurrentProcess())
         _, _, _ = procSetProcessWorkingSetSize.Call(hProcess, ^uintptr(0), ^uintptr(0))
     }
     ```
3. **Tính chịu lỗi (Fault-Tolerance):**
   * Nếu `procSetProcessWorkingSetSize` thất bại, chỉ ghi log debug nhẹ, tuyệt đối không được cản trở việc ẩn hay hiện cửa sổ.
4. **Tiêu chuẩn chấp nhận luồng nền:**
   * Luồng `AutoTelemetryLoop` 60s và `AutomationController` vẫn phải chạy chuẩn 100% sau khi trim.
   * Khi người dùng click menu Tray `"Hiển thị Switcher"`, cửa sổ phải hiện lại mượt mà trong **< 100ms**.

---

## 5. THỨ TỰ THỰC HIỆN VÀ THỬ NGHIỆM ĐỘC LẬP CHO CODER

Coder bắt buộc thực hiện theo đúng 5 bước sau:

```
                  [ĐO ĐẠC BASELINE GỐC]
                            │
                            ▼
┌────────────────────────────────────────────────────────┐
│ BƯỚC 1: Win32 API Discovery, Singleflight & Xóa Netstat│
│ • Sửa internal/connector/discovery.go                  │
│ • Bỏ netstat, gắn TTL 10s, PID+CreationTime, context   │
│ • Đo thời gian mở app, xác nhận không có subprocess    │
└───────────────────────────┬────────────────────────────┘
                            │
                            ▼
┌────────────────────────────────────────────────────────┐
│ BƯỚC 2: UI Hydration, Generation ID & Bỏ Re-theme      │
│ • Sửa internal/ui/app.go & window.go                   │
│ • Kiểm tra generationID trong fyne.Do()                │
│ • Tính lại quyền nút chi tiết tại READY                │
│ • Xóa goroutine 50ms, kiểm tra độ nét DPI              │
└───────────────────────────┬────────────────────────────┘
                            │
                            ▼
┌────────────────────────────────────────────────────────┐
│ BƯỚC 3: Đồng bộ An toàn Backend Switch - Delete        │
│ • Sửa internal/manager/account_manager.go              │
│ • Thêm khóa opMu điều phối giữa Switch và Remove       │
│ • Double-check active trong RemoveAccount              │
└───────────────────────────┬────────────────────────────┘
                            │
                            ▼
┌────────────────────────────────────────────────────────┐
│ BƯỚC 4: Khảo sát Go Heap (FreeOSMemory & GOMEMLIMIT)   │
│ • Test A/B 1 lần FreeOSMemory khi vào Steady State     │
│ • Thử nghiệm GOMEMLIMIT=64MiB, 96MiB, 128MiB           │
└───────────────────────────┬────────────────────────────┘
                            │
                            ▼
┌────────────────────────────────────────────────────────┐
│ BƯỚC 5: Thao tác "Ẩn vào Tray" & Trim Working Set      │
│ • Thêm mục menu "Ẩn giao diện vào khay" trong Tray     │
│ • Giữ nguyên nút X và nút minimize                     │
│ • Đo RAM trong Task Manager khi ẩn xuống Tray          │
│ • Xác nhận Telemetry 60s và AutoSwitch chạy đều đặn    │
└────────────────────────────────────────────────────────┘
```

---

## 6. MA TRẬN KIỂM THỬ & TIÊU CHUẨN NGHIỆM THU (TEST & ACCEPTANCE MATRIX)

| ID Kiểm thử | Hạng mục kiểm thử | Kịch bản kiểm thử | Tiêu chuẩn Nghiệm thu bắt buộc |
|---|---|---|---|
| **`TC-OPT-01`** | Win32 Process Discovery | Bật app khi Language Server đang chạy | Tìm thấy PID và CSRF Token trong **< 5ms**; Task Manager không hề xuất hiện `powershell.exe` hay `netstat.exe`. |
| **`TC-OPT-02`** | Singleflight & Context | 5 goroutine cùng gọi Discovery, 1 goroutine bị cancel context | Chỉ chạy 1 lượt Win32 snapshot; luồng bị cancel thoát ngay; 4 luồng còn lại nhận kết quả chuẩn xác. |
| **`TC-OPT-03`** | TTL 10s & Restart Handling | Chờ 11 giây hoặc khởi động lại Language Server | Quá 10s buộc phải quét lại; server đổi PID/CreationTime tự động làm mới cache và kết nối thành công. |
| **`TC-OPT-04`** | Authorization Matrix tại READY | Kiểm tra tài khoản OFF, Unauth, Active, Ready | Nút Chuyển chỉ mở trên thẻ Ready; nút Xóa lúc nào cũng bật trên toàn bộ thẻ Inactive (ON, OFF, Unauth) và ẩn hoàn toàn trên thẻ Active. |
| **`TC-OPT-05`** | Stale Response Drop trong `fyne.Do` | Bấm "Làm mới" rồi bấm "Chuyển tài khoản" ngay | Generation tăng; callback cũ khi vào `fyne.Do` phát hiện lệch gen và hủy ngay; không ghi đè trạng thái mới. |
| **`TC-OPT-06`** | Switch-Delete Concurrency | Gọi song song SwitchAccount và RemoveAccount cùng email | Luồng được điều phối tuần tự qua khóa backend; tài khoản không thể bị xóa khi đang là active target. |
| **`TC-OPT-07`** | Theme & DPI Sharpness | Mở app ở các tỷ lệ màn hình 100%, 125%, 150% DPI | Chữ hiển thị sắc nét 100%, không giật màn hình sau 50ms, không re-render cây widget lần 2. |
| **`TC-OPT-08`** | Preserved Window Controls | Bấm nút thu nhỏ `—` và nút đóng `X` | Nút `—` thu nhỏ xuống Taskbar; nút `X` thoát app sạch hoàn toàn; không bị biến tướng thành ẩn tray. |
| **`TC-OPT-09`** | Tray Trim & Background Loop | Chọn "Ẩn giao diện vào khay" trong 15 phút | RAM Task Manager tụt sâu (< 40MB); Telemetry 60s và AutoSwitch vẫn ghi log và chuyển tài khoản bình thường; click hiện lại < 100ms. |

---

## 7. KẾT LUẬN

Bản đặc tả R2 đã bổ sung đầy đủ và chặt chẽ toàn bộ các quy tắc hành vi, loại trừ 100% việc Coder phải tự suy đoán hay vô tình làm thay đổi tính năng sản phẩm. Coder Lane có thể triển khai tuần tự theo 5 bước với độ chính xác tuyệt đối.
