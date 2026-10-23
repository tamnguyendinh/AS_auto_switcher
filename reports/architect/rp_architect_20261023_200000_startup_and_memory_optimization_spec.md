# BÁO CÁO ĐẶC TẢ KIẾN TRÚC (ARCHITECTURE SPECIFICATION REPORT)
## TỐI ƯU HÓA HIỆU NĂNG TẦNG DƯỚI & BẢO TOÀN TRỌN VẸN TRẢI NGHIỆM NGƯỜI DÙNG (TRANSPARENT OPTIMIZATION SPECIFICATION)

- **Mã tài liệu:** `rp_architect_20261023_200000_startup_and_memory_optimization_spec`
- **Tác giả:** System Architect Lane
- **Ngày lập:** 2026-10-23 (Phiên bản R3 — Tối ưu hóa trong suốt, Không xáo trộn UX)
- **Thẩm quyền:** Human Owner
- **Đối tượng thụ hưởng:** Coder Lane, QA Lane
- **Trạng thái:** **`VERDICT: PASS / READY_FOR_CODER`**

---

## 1. NGUYÊN TẮC THIẾT KẾ CỐT LÕI (CORE PHILOSOPHY)

> **"Sự cẩn thận thực sự là bảo vệ trải nghiệm đang có. Tối ưu hoàn toàn ở tầng dưới (under-the-hood); giữ nguyên 100% giao diện, cách bấm và điều kiện thao tác hiện tại của người dùng. Không được vì 'an toàn' phòng ngừa mà đẻ thêm trạng thái khóa toàn cục, skeleton hay menu rườm rà làm ứng dụng khó dùng hơn."**

### Ba Mục tiêu Trọng tâm:
1. **Khởi động tức thì (< 300ms):** Cửa sổ hiện ngay với dữ liệu có sẵn từ cache đĩa; loại bỏ hoàn toàn các tiến trình phụ (`powershell.exe`, `netstat.exe`).
2. **Không xáo trộn thao tác khi đang dùng:** Khi người dùng bấm "Làm mới danh sách", toàn bộ nội dung và các nút bấm hợp lệ vẫn giữ nguyên khả năng tương tác bình thường; dữ liệu mới âm thầm cập nhật khi có kết quả.
3. **Bộ nhớ nhẹ nhàng:** Giảm lượng heap mà Go giữ lại thông qua dọn dẹp nhàn rỗi và khảo sát `GOMEMLIMIT`, không làm gián đoạn độ mượt của giao diện.

---

## 2. NGUYÊN TẮC BẤT BIẾN BẮT BUỘC (ARCHITECTURAL INVARIANTS)

1. **`Invariant-Preserve-Existing-UX` (Bảo toàn 100% Giao diện & Thao tác):**
   - Giữ nguyên toàn bộ layout, màu sắc, vị trí các nút bấm, dialog xác nhận xóa và cách tương tác hiện có.
   - Tuyệt đối KHÔNG đưa toàn bộ app vào trạng thái khóa toàn cục (`VERIFYING`) mỗi khi làm mới.
   - Tuyệt đối KHÔNG thêm các thành phần giao diện thừa thãi như skeleton loader, badge lạ hay các nút thử lại gây rối mắt.
2. **`Invariant-Trash-Button-Untouched` (Bảo toàn luồng chạy Nút Xóa):**
   - Nút Xóa `[ 🗑 ]` trên mọi thẻ không hoạt động (Inactive: bao gồm Status ON, Status OFF, Cần OAuth) **LÚC NÀO CŨNG BẬT VÀ HIỂN THỊ (`Show()` và `Enable()`)**.
   - Nút Xóa trên thẻ đang hoạt động (`IsActive == true`): **ẨN HOÀN TOÀN (`Hide()`)**.
   - Đợt tối ưu này tuyệt đối KHÔNG can thiệp, không vô hiệu hóa hay làm thay đổi bất kỳ hành vi nào của nút Xóa (tuân thủ tuyệt đối `SPEC-ARCH-20261023-ACCOUNT-DELETION-AND-TRASH-BUTTON`).
3. **`Invariant-Zero-Subprocess-Discovery` (Cấm 100% Subprocess):**
   - Loại bỏ hoàn toàn việc gọi `powershell.exe` và `netstat.exe`.
   - Quét tiến trình `language_server.exe` bằng Win32 Toolhelp32 API thuần trong kernel (< 1ms).
   - Đọc CommandLine bằng `ntdll.dll::NtQueryInformationProcess` (Class 60 - `ProcessCommandLineInformation`).
   - Quét cổng lắng nghe 100% bằng Win32 API `iphlpapi.dll::GetExtendedTcpTable` thuần.
4. **`Invariant-Transparent-Background-Refresh` (Làm mới ngầm không gián đoạn):**
   - Trong chu kỳ app đang mở và người dùng bấm "Làm mới danh sách": Nút trên Toolbar chỉ đổi chữ sang `"⏳ Đang làm mới..."` như thiết kế hiện tại, còn lại danh sách thẻ tài khoản và các nút bấm vẫn giữ nguyên trạng thái để người dùng thao tác. Khi goroutine ngầm hoàn tất thì dữ liệu mới cập nhật vào UI.
5. **`Invariant-Silent-Stale-Drop` (Hủy kết quả cũ âm thầm):**
   - Quản lý `generationID uint64` tăng dần khi có Refresh hoặc Switch.
   - Kiểm tra `generationID` ngay trong callback `fyne.Do()`. Nếu phát hiện là kết quả của lượt cũ thì âm thầm bỏ qua (silent drop), không ghi đè dữ liệu mới và không can thiệp vào UI.
6. **`Invariant-Backend-Execution-Time-Check` (Bảo vệ tại thời điểm thực thi):**
   - Không khóa nút phòng ngừa vô cớ trên UI.
   - Tại tầng backend (`AccountManager`), việc kiểm tra điều kiện (tài khoản active, token hợp lệ) được thực hiện ngay tại thời điểm thực thi lệnh Chuyển hoặc Xóa. Nếu vi phạm (ví dụ tài khoản vừa trở thành active), backend trả về lỗi tương ứng để UI hiển thị thông báo chính xác cho thao tác đó.
7. **`Invariant-Preserve-Window-Controls` (Giữ nguyên điều khiển cửa sổ):**
   - Nút `X` và close intercept: Thoát sạch hoàn toàn ứng dụng (`a.Close()` -> `a.app.Quit()`).
   - Nút `—`: Thu nhỏ xuống Windows Taskbar (`MinimizeNativeWindow()`).
   - Tuyệt đối không thêm luồng mới hay sửa nút đóng thành ẩn chỉ để tìm chỗ gọi hàm trim.

---

## 3. BẢN ĐỒ CAN THIỆP MÃ NGUỒN (TOUCH-MAP)

| Tệp mã nguồn | Trách nhiệm can thiệp | Nội dung thay đổi (Thuần tầng dưới) |
|---|---|---|
| `internal/connector/discovery.go` | Thay thế PowerShell & Netstat bằng Win32 API thuần + Singleflight | - Viết lại `QueryLanguageServerProcesses` dùng Toolhelp32 Snapshot.<br>- Đọc CommandLine bằng `NtQueryInformationProcess`.<br>- Xóa bỏ hàm `discoverPortsViaNetstat`.<br>- Bổ sung Cache discovery dùng chung: TTL 10s, cặp `(PID, CreationTime)`, Singleflight gom luồng, tôn trọng `ctx.Done()`. |
| `internal/manager/account_manager.go` | Double-check active khi xóa + Tích hợp discovery cache | - Dùng chung hàm discovery của `connector`.<br>- Trong `RemoveAccount`: kiểm tra lại `activeEmail` ngay trước khi xóa tệp (Double-check validation). |
| `internal/ui/app.go` | Tải dữ liệu đĩa sớm + Bỏ Re-theme thừa + Generation ngầm | - Khi mở app: Đọc ngay store/cache hôm nay để hiển thị ngay frame đầu.<br>- Xóa bỏ goroutine 50ms `SyncVisualScaleAndTextures` sau `win.Show()`.<br>- Quản lý `refreshGen` ngầm để bỏ qua phản hồi stale trong `fyne.Do()`.<br>- Thêm 1 lần `debug.FreeOSMemory()` chạy ngầm sau khi nạp xong. |

---

## 4. CHI TIẾT ĐẶC TẢ KỸ THUẬT (DETAILED SPECIFICATION)

### 4.1. Tối ưu Discovery Tầng Dưới (`internal/connector/discovery.go`)

1. **Quét tiến trình Win32 API trực tiếp:**
   - Dùng `windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)` duyệt tiến trình `language_server.exe` trong kernel.
   - Mở tiến trình: `windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)`.
   - Lấy thời điểm tạo tiến trình: `windows.GetProcessTimes(hProcess, &creationTime, &exitTime, &kernelTime, &userTime)`.
   - Đọc chuỗi CommandLine: Gọi `ntdll.dll::NtQueryInformationProcess` với `ProcessInformationClass = 60` (`ProcessCommandLineInformation`). Nếu không đọc được (tiến trình thoát giữa chừng), bỏ qua an toàn; **không đòi hỏi quyền Admin/UAC**.
   - Luôn đóng Handle tiến trình sau khi đọc.
2. **Xóa bỏ `discoverPortsViaNetstat`:**
   - Xóa bỏ hoàn toàn hàm gọi `exec.Command("netstat", ...)`. Việc dò cổng chỉ dùng `discoverPortsViaWin32` (`iphlpapi.dll::GetExtendedTcpTable`).
3. **Bộ nhớ đệm Discovery & Singleflight:**
   - Dùng cấu trúc cache chia sẻ chung:
     ```go
     type serverDiscoveryCache struct {
         mu           sync.Mutex
         info         *ServerInfo
         creationTime windows.Filetime
         cachedAt     time.Time
     }
     ```
   - **TTL 10 giây:** Nếu `time.Since(cachedAt) <= 10*time.Second`, kiểm tra nhanh `PID` và `CreationTime` còn sống thì trả về ngay (độ trễ = 0). Quá 10s hoặc tiến trình đổi -> quét lại.
   - **Singleflight:** Gom các yêu cầu đồng thời từ `manager`, `telemetry`, và `automation` thành 1 lượt quét duy nhất.
   - **Hỗ trợ `ctx.Done()`:** Nếu context của caller bị timeout/cancel, caller thoát ngay mà không bị block.
   - **Không thăm dò HTTP giả lập trước mỗi request:** Bỏ probe nhân tạo lặp lại; request Connect-RPC thật sẽ tự fail-fast nếu cổng lỗi.

---

### 4.2. Khởi động Nhanh & Làm mới Trong suốt (`internal/ui/app.go`)

1. **Lần mở đầu (Cold Start):**
   - Nạp ngay danh sách tài khoản từ file cục bộ `accounts_store/` và cache quota từ `logs/quota_log/`.
   - Cửa sổ Fyne hiện lên lập tức (< 300ms) với đầy đủ danh sách tài khoản và số liệu quota gần nhất đã lưu.
   - Language Server và token capture được kích hoạt kiểm tra ngầm dưới background; khi có kết quả mới nhất sẽ âm thầm cập nhật Footer status và badge trạng thái qua `fyne.Do()`.
2. **Làm mới khi đang dùng (Warm Refresh):**
   - Khi bấm `"Làm mới danh sách"`: Nút Toolbar đổi chữ `"⏳ Đang làm mới..."`.
   - **Toàn bộ danh sách thẻ, nút Chuyển và nút Xóa vẫn giữ nguyên khả năng tương tác** với dữ liệu hiện tại, không khóa toàn cục, không làm gián đoạn người dùng.
   - Khi goroutine đọc xong: Cập nhật giao diện mượt mà và nút Toolbar trở lại `"Làm mới danh sách"`.
3. **Chống kết quả cũ âm thầm (Silent Stale Drop):**
   - Quản lý `refreshGen atomic.Uint64`. Tăng khi Refresh hoặc Switch.
   - Trong `fyne.Do()`: So khớp `if a.refreshGen.Load() != savedGen { return }`. Nếu kết quả thuộc về lượt cũ thì âm thầm bỏ qua, không làm xáo trộn dữ liệu mới.
4. **Loại bỏ Re-theme và Re-render thừa sau `Show()`:**
   - **XÓA BỎ** goroutine `time.Sleep(50ms)` và lời gọi `SyncVisualScaleAndTextures()` sau `win.Show()`.
   - Theme và phông chữ được Fyne khởi tạo chuẩn xác một lần duy nhất trước `Show()`.

---

### 4.3. Kiểm tra Điều kiện Thực thi Backend (`internal/manager/account_manager.go`)

1. **Trong `RemoveAccount` (`account_manager.go:215`):**
   - Giữ nguyên luồng gọi và giao diện.
   - Tại tầng backend, trước khi thực hiện xóa tệp tin, thực hiện kiểm tra lại `activeEmail` (Double-checked verification) để bảo đảm tài khoản mục tiêu không bị active ở thời điểm xóa.
   - Nếu tài khoản đang active, trả về lỗi `ErrCannotDeleteActiveAccount` để UI hiển thị pop-up cảnh báo thông thường, không xóa nhầm.
2. **Trong `SwitchAccount`:**
   - Giữ nguyên cơ chế chuyển tài khoản hiện có qua `SwitchGate`.

---

### 4.4. Tối ưu Go Runtime Memory (`internal/ui/app.go`)

1. **Thu hồi heap nhàn rỗi một lần:**
   - Sau khi ứng dụng nạp xong dữ liệu lần đầu tiên và đạt trạng thái ổn định, một goroutine chạy ngầm với `sync.Once` gọi `time.Sleep(200ms)` rồi gọi `debug.FreeOSMemory()`.
   - Thao tác này chạy hoàn toàn dưới nền, âm thầm trả các trang nhớ giải mã JSON ban đầu về cho OS mà không làm giật lag giao diện.
2. **Khảo sát `GOMEMLIMIT`:**
   - Khảo sát qua biến môi trường (`GOMEMLIMIT=64MiB`, `96MiB`, `128MiB`) để chọn giá trị tối ưu, không can thiệp code vào GUI.

---

## 5. THỨ TỰ THỰC HIỆN ĐỘC LẬP CHO CODER

Coder triển khai theo đúng 4 bước kỹ thuật tầng dưới sau:

```
                  [ĐO ĐẠC BASELINE GỐC]
                            │
                            ▼
┌────────────────────────────────────────────────────────┐
│ BƯỚC 1: Win32 API Discovery, Singleflight & Xóa Netstat│
│ • Sửa internal/connector/discovery.go                  │
│ • Thay PowerShell bằng Toolhelp32 snapshot             │
│ • Đọc CommandLine qua NtQueryInformationProcess        │
│ • Xóa discoverPortsViaNetstat, dùng GetExtendedTcpTable│
│ • Thêm Cache TTL 10s, PID+CreationTime, Singleflight   │
└───────────────────────────┬────────────────────────────┘
                            │
                            ▼
┌────────────────────────────────────────────────────────┐
│ BƯỚC 2: Tải Dữ liệu Sớm & Bỏ Re-theme Thừa             │
│ • Sửa internal/ui/app.go                               │
│ • Hiện cửa sổ tức thì với cache đĩa hôm nay            │
│ • Xóa bỏ goroutine 50ms SyncVisualScaleAndTextures     │
│ • Quản lý refreshGen âm thầm drop kết quả cũ          │
│ • Giữ nguyên 100% giao diện và nút bấm khi refresh     │
└───────────────────────────┬────────────────────────────┘
                            │
                            ▼
┌────────────────────────────────────────────────────────┐
│ BƯỚC 3: Backend Verification tại Thời điểm Thực thi    │
│ • Sửa internal/manager/account_manager.go              │
│ • Double-check activeEmail trong RemoveAccount         │
└───────────────────────────┬────────────────────────────┘
                            │
                            ▼
┌────────────────────────────────────────────────────────┐
│ BƯỚC 4: Khảo sát Bộ nhớ Go (FreeOSMemory & GOMEMLIMIT) │
│ • 1 lần FreeOSMemory ngầm khi app đạt trạng thái ổn định│
│ • Khảo sát các mức GOMEMLIMIT qua biến môi trường      │
└────────────────────────────────────────────────────────┘
```

---

## 6. MA TRẬN KIỂM THỬ NGHIỆM THU (ACCEPTANCE CRITERIA)

| ID Kiểm thử | Hạng mục kiểm thử | Tiêu chuẩn Nghiệm thu bắt buộc |
|---|---|---|
| **`TC-OPT-01`** | Win32 Discovery | Bật app: tìm thấy Language Server trong **< 5ms**; Task Manager không hề xuất hiện `powershell.exe` hay `netstat.exe`. |
| **`TC-OPT-02`** | Singleflight Cache | Nhiều luồng cùng gọi Discovery: chỉ 1 lần quét Win32; kết quả chia sẻ chuẩn xác; hết hạn 10s tự động làm mới. |
| **`TC-OPT-03`** | Khởi động tức thì | App mở lên trong **< 300ms**, hiện ngay email và quota đã lưu từ trước. |
| **`TC-OPT-04`** | Warm Refresh Mượt mà | Bấm "Làm mới danh sách": các nút bấm và thẻ vẫn thao tác bình thường, không bị khóa, không bị gián đoạn. |
| **`TC-OPT-05`** | Bảo toàn Nút Xóa | Nút Xóa `[ 🗑 ]` luôn bật và bấm được trên mọi thẻ Inactive (ON, OFF, Cần OAuth); ẩn hoàn toàn trên thẻ Active. |
| **`TC-OPT-06`** | Backend Active Protection | Cố tình xóa tài khoản vừa active: backend chặn an toàn và trả lỗi `ErrCannotDeleteActiveAccount`. |
| **`TC-OPT-07`** | Phông chữ & DPI | Cửa sổ hiển thị sắc nét ở các mức DPI 100%, 125%, 150%, không bị nháy giật màn hình sau 50ms. |
| **`TC-OPT-08`** | Điều khiển cửa sổ | Nút `—` thu nhỏ xuống Taskbar; nút `X` thoát app sạch sẽ hoàn toàn; không bị thêm bớt menu lạ. |

---

## 7. KẾT LUẬN

Bản đặc tả R3 đã loại bỏ triệt để mọi chi tiết gây nặng nề UX, tập trung 100% vào việc tối ưu sâu bên dưới hạ tầng: **loại bỏ tiến trình phụ, gom luồng, cache thông minh và dọn dẹp bộ nhớ nhàn rỗi**. Coder có thể tiến hành lập trình ngay lập tức với độ an toàn cao nhất.
