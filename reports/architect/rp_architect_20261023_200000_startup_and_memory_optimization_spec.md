# BÁO CÁO ĐẶC TẢ KIẾN TRÚC (ARCHITECTURE SPECIFICATION REPORT)
## TỐI ƯU HÓA TỐC ĐỘ KHỞI ĐỘNG VÀ MỨC TIÊU THỤ BỘ NHỚ RAM ỨNG DỤNG SWITCHER

- **Mã tài liệu:** `rp_architect_20261023_200000_startup_and_memory_optimization_spec`
- **Tác giả:** System Architect Lane
- **Ngày lập:** 2026-10-23
- **Thẩm quyền:** Human Owner
- **Đối tượng thụ hưởng:** Coder Lane, QA Lane
- **Trạng thái:** **`VERDICT: PASS / READY_FOR_CODER`**

---

## 1. BỐI CẢNH & MỤC TIÊU (CONTEXT & OBJECTIVES)

### 1.1. Hiện trạng đo đạc thực tế (Baseline Analysis)
Khảo sát trực tiếp trên mã nguồn và tiến trình thực tế (`antigravity-switch.exe` - PID 16020):
1. **Tốc độ khởi động:** Trễ từ **1.5s - 2.5s** trước khi giao diện sẵn sàng thao tác.
   - *Nguyên nhân cốt lõi:* Hàm `QueryLanguageServerProcesses` trong `internal/connector/discovery.go` đang gọi `powershell.exe -Command Get-CimInstance Win32_Process ...`. Trên Windows, mỗi lần spawn PowerShell tốn **500ms - 1200ms**. Trong chu kỳ `GetAccountStates()`, do thiếu bộ nhớ đệm, PowerShell bị gọi **2 lần liên tiếp** gây nghẽn nghiêm trọng.
   - Thêm vào đó, sau khi mở cửa sổ Fyne, có goroutine chạy nền `time.Sleep(50ms)` rồi gọi `SyncVisualScaleAndTextures()` ép re-theme và re-render lại toàn bộ cây widget lần thứ 2.
2. **Bộ nhớ RAM:** Working Set đo được trong Task Manager dao động **~134 MB – 182.4 MB** (Private Memory ~259.5 MB).
   - *Nguyên nhân cốt lõi:* Fyne render qua OpenGL (`nvoglv64.dll`, `OPENGL32.dll`), driver card đồ họa nạp vào tiến trình giữ cố định **~70MB – 90MB** (framebuffers, shader compilers, texture glyph cache).
   - Go runtime (`mheap`) giữ lại các trang nhớ sau khi giải mã JSON/config khởi động (`GOGC=100`), không tự động trả về cho OS.

### 1.2. Ba Mục tiêu Tối ưu hóa (Triple Targets)
1. **Mở giao diện tức thì (Startup Latency < 300ms):** Render khung giao diện và dữ liệu cache ngay trong frame đầu tiên; loại bỏ 100% việc gọi subprocess `powershell.exe`.
2. **Giảm bộ nhớ Go giữ lại (Go Heap Retention):** Thu hồi trang nhớ nhàn rỗi sau khi hoàn tất khởi động; thiết lập ngưỡng bộ nhớ linh hoạt qua `GOMEMLIMIT`.
3. **Giảm RAM Resident khi chạy nền (Background Working Set Trim):** Khi ứng dụng ẩn vào System Tray (`win.Hide()`), giải phóng các trang bộ nhớ vật lý nhàn rỗi mà không làm ảnh hưởng đến luồng giám sát nền (AutoSwitch & Telemetry).

---

## 2. NGUYÊN TẮC BẤT BIẾN KIẾN TRÚC (ARCHITECTURAL INVARIANTS)

1. **`Invariant-Zero-Subprocess-Discovery`:**
   Tuyệt đối KHÔNG gọi `powershell.exe`, `cmd.exe` hay bất kỳ tiến trình con bên ngoài nào để truy vấn tiến trình hoặc mạng. Toàn bộ thao tác giám sát hệ thống phải sử dụng Win32 / NT API thuần trong Go (`windows.CreateToolhelp32Snapshot`, `ntdll.dll::NtQueryInformationProcess`, `iphlpapi.dll`).
2. **`Invariant-Lifecycle-Safe-Caching`:**
   Bộ nhớ đệm Discovery phải gắn liền với cặp định danh `(PID, ProcessCreationTime)`. Không bao giờ tin tưởng riêng rẽ một mình `PID` để ngăn chặn lỗi tái sử dụng PID (PID recycling) của Windows. Khi thay thế cache, bắt buộc phải đóng Handle cũ.
3. **`Invariant-UI-Two-Phase-Hydration`:**
   Giao diện người dùng phân tách thành 2 giai đoạn:
   - Giai đoạn 1 (`VERIFYING`): Hiện ngay email và quota từ cache cục bộ hôm nay để cửa sổ có nội dung hữu ích; tạm thời khóa các nút hành động nguy hiểm (Chuyển, Xóa).
   - Giai đoạn 2 (`READY` hoặc `LIMITED_CONNECTIVITY`): Mở khóa nút bấm khi xác minh hoàn tất, hoặc hiển thị nút "Thử lại" nếu kết nối lỗi/quá hạn; tuyệt đối không để giao diện kẹt vĩnh viễn ở `VERIFYING`.
4. **`Invariant-Stale-Response-Drop`:**
   Mọi phản hồi bất đồng bộ từ các lượt refresh cũ phải bị hủy bỏ nếu đã có lượt refresh mới hơn hoặc đã diễn ra thao tác chuyển tài khoản (kiểm soát qua `requestID / generationID uint64`).
5. **`Invariant-Backend-First-Safety`:**
   Trạng thái vô hiệu hóa (disabled) của nút bấm trên UI chỉ là lớp bảo vệ thị giác (UX). Lớp chốt chặn bảo vệ cuối cùng chống race condition và bảo toàn tài khoản bắt buộc nằm ở `SwitchGate` tại tầng backend.
6. **`Invariant-Background-Monitoring-Integrity`:**
   Tác vụ hoán chuyển bộ nhớ (`EmptyWorkingSet`) chỉ được kích hoạt khi cửa sổ **ẩn hoàn toàn vào System Tray**, KHÔNG kích hoạt khi minimize xuống Taskbar. Vòng lặp tự động chuyển tài khoản (AutoSwitch) và đo kiểm Quota (Telemetry 60s) phải tiếp tục hoạt động chính xác và không bị gián đoạn.

---

## 3. BẢN ĐỒ CAN THIỆP MÃ NGUỒN (TOUCH-MAP)

| Tệp mã nguồn | Trách nhiệm can thiệp | Phạm vi thay đổi |
|---|---|---|
| `internal/connector/discovery.go` | Thay thế PowerShell bằng Win32 API + Singleflight Cache | Viết lại `QueryLanguageServerProcesses`, thêm cache `(PID, CreationTime)`, thêm Singleflight |
| `internal/manager/account_manager.go` | Tích hợp cache discovery và hủy kết quả stale | Cập nhật `clientFinder`, tích hợp Generation ID kiểm tra trạng thái |
| `internal/ui/app.go` | UI State Machine + Bỏ Re-theme thừa + Generation ID | Thêm trạng thái `VERIFYING` -> `READY`/`LIMITED`, bỏ goroutine 50ms `SyncVisualScaleAndTextures`, thêm 1-shot `FreeOSMemory` |
| `internal/ui/window.go` | Khóa/Mở nút theo trạng thái an toàn | Cập nhật render các nút Chuyển/Xóa theo State |
| `internal/ui/tray.go` | Kích hoạt Working Set Trim khi Ẩn Tray | Thêm gọi Win32 `SetProcessWorkingSetSize` khi `win.Hide()` |

---

## 4. CHI TIẾT ĐẶC TẢ KỸ THUẬT (DETAILED SPECIFICATION)

### 4.1. Bước 1: Win32 API Discovery + Singleflight + Cache Vòng đời

#### A. Truy vấn tiến trình qua Win32 API (`internal/connector/discovery.go`)
Thay thế toàn bộ đoạn code gọi `powershell` trong `QueryLanguageServerProcesses`:
1. Dùng `windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)` để lấy snapshot toàn bộ tiến trình.
2. Dùng `windows.Process32First` và `windows.Process32Next` để duyệt danh sách, lọc các tiến trình có `ExeFile == "language_server.exe"`.
3. Với mỗi tiến trình tìm thấy:
   - Gọi `windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pe.ProcessID)` (hoặc `PROCESS_QUERY_INFORMATION | PROCESS_VM_READ`).
   - Lấy thời điểm khởi tạo tiến trình (`CreationTime`) qua `windows.GetProcessTimes(hProcess, &creationTime, &exitTime, &kernelTime, &userTime)`.
   - Lấy chuỗi dòng lệnh (CommandLine) bằng một trong hai cách:
     - **Cách ưu tiên:** Gọi hàm `ntdll.dll::NtQueryInformationProcess` với `ProcessInformationClass = 60` (`ProcessCommandLineInformation`). Kernel Windows sao chép trực tiếp struct `UNICODE_STRING` chứa CommandLine vào buffer an toàn, không lo crash nếu tiến trình thoát.
     - **Dự phòng:** Trích xuất qua PEB (`ProcessParameters->CommandLine`) có bọc kiểm tra con trỏ hợp lệ.
   - Nếu không thể mở tiến trình hoặc không đọc được command line (ví dụ tiến trình thoát giữa chừng), bỏ qua và không yêu cầu quyền Admin/Elevation.
   - Đóng `hProcess` sau khi đọc xong.

#### B. Cơ chế Caching và Singleflight
Tạo cấu trúc quản lý Discovery dùng chung:
```go
type serverDiscoveryCache struct {
    mu           sync.Mutex
    info         *ServerInfo
    creationTime windows.Filetime
    cachedAt     time.Time
}

var (
    discCache serverDiscoveryCache
    discFlight singleflight.Group // gom các cuộc gọi đồng thời thành 1
)
```
* **Thời gian sống của cache (TTL):** 10 giây.
* **Quy tắc tái sử dụng cache:**
  1. Kiểm tra xem PID trong cache còn tồn tại không bằng `windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(cachedPID))`.
  2. Nếu mở được, kiểm tra `GetProcessTimes`. Nếu `CreationTime` trùng khớp 100% với cache -> Cache hợp lệ, trả về ngay lập tức (không chạy lại Win32 Snapshot).
  3. Nếu `CreationTime` khác hoặc tiến trình đã thoát -> Cache bị hủy, thực hiện quét lại.
* **Hủy cache khi có lỗi kết nối:** Khi các lệnh Connect-RPC nghiệp vụ nhận mã lỗi `Connection Refused` hoặc `401 Unauthorized`, gọi hàm `InvalidateDiscoveryCache()` để xóa cache, buộc lần gọi tiếp theo phải quét lại.
* **Không chèn HTTP Probe trước mọi API call:** Xóa bỏ bước thăm dò cổng HTTP nhân tạo nếu cổng đã được xác định trước đó; để request nghiệp vụ thật tự động xác thực cổng.

---

### 4.2. Bước 2: Chuẩn hóa Giao diện (UI Two-Phase Hydration & Stale Response Drop)

#### A. Generation ID chống Stale Response (`internal/ui/app.go`)
Trong `struct App`:
```go
type App struct {
    ...
    refreshGen atomic.Uint64
    uiState    atomic.Int32 // 0: StateVerifying, 1: StateReady, 2: StateLimited
}

const (
    UIStateVerifying = 0
    UIStateReady     = 1
    UIStateLimited   = 2
)
```
Khi kích hoạt `Refresh()`:
1. `gen := a.refreshGen.Add(1)`: Tăng generation counter.
2. Đặt `a.uiState.Store(UIStateVerifying)`.
3. Hiển thị ngay tức thì dữ liệu tài khoản có sẵn trong bộ nhớ/cache đĩa hôm nay, nhãn trạng thái gắn `[⏳ Đang kiểm tra...]`.
4. Vô hiệu hóa (`Disable()`) các nút "Chuyển tài khoản" và "Xóa" trên giao diện để ngăn người dùng thao tác trong lúc trạng thái đang được xác minh lại.
5. Khi goroutine nền lấy được kết quả từ backend:
   ```go
   if a.refreshGen.Load() != gen {
       // Đã có một lượt refresh mới hơn hoặc đã chuyển tài khoản -> BỎ QUA KẾT QUẢ CŨ
       return
   }
   ```
6. Nếu thành công: Cập nhật `UIStateReady`, mở khóa (`Enable()`) các nút bấm và cập nhật badge chuẩn.
7. Nếu thất bại/hết thời gian: Cập nhật `UIStateLimited`, hiển thị thông báo lỗi trên thanh Status Bar kèm nút **"Thử lại"**, không để UI bị treo vô hạn ở `VERIFYING`.

#### B. Loại bỏ Re-theme và Re-render thừa sau `Show()`
Trong hàm `ShowAndRun()` tại `internal/ui/app.go`:
* **Hiện tại:**
  ```go
  go func() {
      time.Sleep(50 * time.Millisecond)
      fyne.Do(func() {
          a.SyncVisualScaleAndTextures()
      })
  }()
  ```
* **Đặc tả mới:**
  * **XÓA BỎ** hoàn toàn khối goroutine `time.Sleep(50ms)` và lời gọi `SyncVisualScaleAndTextures()` sau `Show()`.
  * Khởi tạo Palette, Theme và Canvas scale chuẩn xác duy nhất 1 lần trong `Init()` trước khi gọi `win.Show()`.
  * Giữ nguyên hàm `SyncVisualScaleAndTextures()` cho mục đích duy nhất: khi người dùng chủ động bấm đổi Theme Sáng/Tối trong màn hình Settings.

---

### 4.3. Bước 3: Quản lý Bộ nhớ Go Runtime (A/B Test `FreeOSMemory` & `GOMEMLIMIT`)

1. **Khảo sát `FreeOSMemory()` 1 lần duy nhất:**
   * Trong `internal/ui/app.go`, tạo một `sync.Once`:
     ```go
     var steadyStateOnce sync.Once
     ```
   * Khi ứng dụng hoàn tất quá trình tải dữ liệu lần đầu tiên và chuyển sang `UIStateReady`:
     ```go
     steadyStateOnce.Do(func() {
         go func() {
             // Đợi UI render xong frame đầu tiên ổn định
             time.Sleep(200 * time.Millisecond)
             debug.FreeOSMemory()
         }()
     })
     ```
   * **Yêu cầu đo đạc A/B:** Coder/QA phải ghi nhận số liệu RAM trước và sau lệnh này so với bản baseline không gọi, xác nhận mức RAM giảm và đảm bảo CPU không bị khựng giật.
2. **Khảo sát `GOMEMLIMIT`:**
   * Không viết thêm code tùy biến vào UI.
   * Thử nghiệm khởi chạy ứng dụng với các biến môi trường:
     - Baseline: Mặc định (không gán `GOMEMLIMIT`).
     - Test A: `GOMEMLIMIT=64MiB`.
     - Test B: `GOMEMLIMIT=96MiB`.
     - Test C: `GOMEMLIMIT=128MiB`.
   * Ghi nhận tần suất GC và độ mượt của animation giao diện để tìm cấu hình tối ưu khuyến nghị.

---

### 4.4. Bước 4: Tối ưu Working Set khi Ẩn vào System Tray (`internal/ui/tray.go`)

1. **Cơ chế gọi Win32 Trim Working Set:**
   Tạo hàm tiện ích trong `internal/ui/single_instance_windows.go`:
   ```go
   var procSetProcessWorkingSetSize = modKernel32.NewProc("SetProcessWorkingSetSize")

   func TrimProcessWorkingSet() {
       hProcess := uintptr(windows.CurrentProcess())
       // Giá trị -1, -1 yêu cầu Windows hoán chuyển các trang nhớ vật lý nhàn rỗi về Standby/Pagefile
       procSetProcessWorkingSetSize.Call(hProcess, ^uintptr(0), ^uintptr(0))
   }
   ```
2. **Điểm gắn móc (Hook Point) chuẩn xác:**
   * **KHÔNG GỌI** khi người dùng minimize xuống Taskbar thông thường.
   * **CHỈ GỌI** tại sự kiện cửa sổ **ẩn hoàn toàn vào System Tray** (`win.Hide()`):
     ```go
     // Trong internal/ui/tray.go hoặc window close intercept
     a.win.Hide()
     TrimProcessWorkingSet()
     ```
3. **Bảo toàn hoạt động giám sát nền:**
   * Goroutine chạy ngầm của `AutoTelemetryLoop` (chu kỳ 60s) và `AutomationController` vẫn giữ nguyên hoạt động bình thường. 
   * Khi đến chu kỳ 60s, goroutine thức dậy, gửi HTTP request lấy quota, Windows sẽ tự động page-in các trang nhớ mạng/JSON cần thiết mà không cần kéo lại toàn bộ buffer đồ họa OpenGL.

---

## 5. THỨ TỰ TRIỂN KHAI VÀ THỬ NGHIỆM ĐỘC LẬP CHO CODER

Coder bắt buộc phải tuân thủ thứ tự 4 bước sau, mỗi bước đều phải đo đạc so với cùng một cấu hình Baseline gốc:

```
                  [ĐO ĐẠC BASELINE GỐC]
                            │
                            ▼
┌────────────────────────────────────────────────────────┐
│ BƯỚC 1: Thay PowerShell bằng Win32 API + Singleflight  │
│ • Sửa internal/connector/discovery.go                  │
│ • Đo thời gian mở app & CPU/RAM tiến trình phụ         │
└───────────────────────────┬────────────────────────────┘
                            │
                            ▼
┌────────────────────────────────────────────────────────┐
│ BƯỚC 2: Cắt bỏ Re-theme thừa sau Show() + UI State     │
│ • Sửa internal/ui/app.go & window.go                   │
│ • Đo độ nét phông chữ (100%, 125%, 150% DPI)           │
│ • Kiểm chứng chuyển đổi VERIFYING -> READY             │
└───────────────────────────┬────────────────────────────┘
                            │
                            ▼
┌────────────────────────────────────────────────────────┐
│ BƯỚC 3: Khảo sát Go Heap (FreeOSMemory & GOMEMLIMIT)   │
│ • Test A/B gọi 1 lần FreeOSMemory khi vào Steady State │
│ • Chạy thử GOMEMLIMIT=64MiB, 96MiB, 128MiB             │
│ • Đo GC pause latency và lượng heap giải phóng         │
└───────────────────────────┬────────────────────────────┘
                            │
                            ▼
┌────────────────────────────────────────────────────────┐
│ BƯỚC 4: Trim Working Set khi Ẩn vào Tray (win.Hide)    │
│ • Sửa internal/ui/tray.go                              │
│ • Đo RAM trong Task Manager khi ẩn xuống Tray          │
│ • Đo thời gian khôi phục (Restore latency) khi click   │
│ • Xác nhận AutoSwitch & Telemetry 60s chạy chuẩn 100%  │
└───────────────────────────┬────────────────────────────┘
                            │
                            ▼
            [TỔNG HỢP CẤU HÌNH TỐI ƯU TOÀN DIỆN]
```

---

## 6. MA TRẬN KIỂM THỬ & TIÊU CHUẨN NGHIỆM THU (TEST & ACCEPTANCE MATRIX)

| ID Kiểm thử | Hạng mục kiểm thử | Kịch bản kiểm thử | Tiêu chuẩn Nghiệm thu bắt buộc |
|---|---|---|---|
| **`TC-OPT-01`** | Win32 Process Discovery | Bật app khi Language Server đang chạy | Tìm thấy PID và CSRF Token trong **< 5ms**; Task Manager không hề xuất hiện tiến trình con `powershell.exe`. |
| **`TC-OPT-02`** | Singleflight Discovery | 5 goroutine cùng gọi `DiscoverLanguageServer()` đồng thời | Chỉ có đúng 1 lệnh snapshot Win32 được thực thi; 5 luồng cùng nhận kết quả chuẩn xác. |
| **`TC-OPT-03`** | Server Restart Handling | Tắt `language_server.exe` và bật lại với PID mới | App phát hiện kết nối cũ thất bại, tự động xóa cache, quét lại PID mới và kết nối thành công. |
| **`TC-OPT-04`** | UI Two-Phase Hydration | Mở app từ trạng thái đóng hoàn toàn | Cửa sổ hiện ngay tức thì (< 300ms) kèm email cache; nút Chuyển/Xóa bị khóa mờ; sau đó mở khóa khi xác minh xong. |
| **`TC-OPT-05`** | Stale Response Drop | Bấm "Làm mới" liên tiếp 2 lần thật nhanh | Kết quả của lần bấm 1 bị hủy bỏ hoàn toàn; chỉ kết quả của lần bấm 2 được render lên UI. |
| **`TC-OPT-06`** | Theme & Font Sharpness | Kiểm tra màn hình ở các tỷ lệ scale 100%, 125%, 150% | Chữ hiển thị sắc nét 100%, không bị vỡ texture; không có hiện tượng giật màn hình sau 50ms. |
| **`TC-OPT-07`** | Working Set Tray Trim | Click ẩn cửa sổ xuống System Tray | Mức Working Set trong Task Manager lập tức tụt sâu (dưới 40MB); icon Tray vẫn phản hồi tốt. |
| **`TC-OPT-08`** | Background Loop Invariant | Ẩn app xuống Tray trong 15 phút | Vòng lặp Telemetry 60s vẫn đều đặn ghi log quota; tính năng AutoSwitch vẫn phát hiện và tự động chuyển tài khoản khi hết hạn ngạch. |
| **`TC-OPT-09`** | Restore Latency | Click icon Tray để mở lại giao diện | Cửa sổ hiện lên tức thì (< 100ms), không bị đơ hoặc treo ứng dụng. |

---

## 7. KẾT LUẬN & KIẾN NGHỊ

Bản đặc tả kiến trúc này đã phân định minh bạch giữa **tối ưu luồng khởi động**, **quản lý heap của Go runtime**, và **hoán chuyển working set của Windows OS**. Tài liệu cung cấp đầy đủ touch-map, chữ ký hàm và thứ tự thử nghiệm độc lập, đảm bảo Coder Lane có thể triển khai chính xác 100% mà không cần phải phỏng đoán.
