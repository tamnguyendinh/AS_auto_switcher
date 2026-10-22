# ⚡ Antigravity Multi-Account Switcher - **Version 1.0.1.9**

Nếu bạn có 1 tài khoản cá nhân, 1 tài khoản công ty, 1 tài khoản để làm dự án bên ngoài; mỗi khi chuyển tài khoản để làm việc cứ phải sign in/out mất thời gian thì công cụ này chính là dành cho bạn.

### **Khi tài khoản được chuyển tự động, tất cả công việc đang làm vẫn tiếp tục bình thường, tuyệt đối không bị gián đoạn, không cần can thiệp thủ công.**

### **An toàn tuyệt đối vì tuân thủ theo cơ chế chuyển đổi tài khoản được Google cho phép. Không can thiệp mã nguồn Antigravity, không dùng cơ chế bọc API**

Ứng dụng Windows viết bằng **Go**, giúp quản lý nhiều tài khoản Google, theo dõi quota và chuyển đổi tài khoản đang dùng trong Antigravity 2.x một cách linh hoạt, an toàn. Ứng dụng cung cấp giao diện desktop **Fyne** đồ họa trực quan, hiện đại, mang lại trải nghiệm Zero-Console mượt mà với duy nhất một file thực thi `antigravity-switch.exe`.

Ứng dụng tích hợp toàn diện hệ thống **Chuyển Đổi Tài Khoản Tự Động (Autonomous Multi-Account Switcher & Failover Daemon)**. Người dùng có thể linh hoạt chọn giữa chuyển tài khoản thủ công (bằng nút **Chuyển** trên GUI) và chuyển đổi tự động hoàn toàn (AutoSwitch). Khi bật chế độ Tự động, hệ thống tự động giám sát quota, phát hiện cạn kiệt tài nguyên, tự phục hồi thử lại trong bộ nhớ, và tự động chọn tài khoản dự phòng tối ưu nhất theo thuật toán **Burn-down Sorter** mà không làm gián đoạn dòng suy nghĩ công việc của lập trình viên.

## Tính năng hiện có

- **Chuyển đổi tài khoản tự động (Autonomous Failover Daemon):** Tự động phát hiện cạn kiệt quota hoặc lỗi giới hạn tài nguyên để hoán đổi tài khoản dự phòng tối ưu nhất mà không cần can thiệp thủ công. 
- **Van kích hoạt kép (Dual-Trigger Engine):** Kết hợp kích hoạt chủ động (chu kỳ telemetry 60 giây khi quota 5h <= 2% hoặc tuần <= 1%) và kích hoạt bị động tức thì (lắng nghe stream Connect-RPC khi gặp lỗi 429 / `RESOURCE_EXHAUSTED`).
- **Phục hồi Native In-Memory 3-Retry:** Tự động thử lại tối đa 3 lần qua Connect-RPC `SendUserCascadeMessage` với danh sách nội dung rỗng (`empty content list`) để xác thực trạng thái tài nguyên thực tế trước khi quyết định chuyển đổi.
- **Thuật toán thông minh Burn-down Sorter:** Lọc qua 5 tiêu chí sơ loại nghiêm ngặt và sắp xếp ứng viên theo 4 tầng ưu tiên nhằm tối đa hóa việc tận dụng hạn mức trước chu kỳ gia hạn.
- **Bảo vệ liên tiến trình Single-Flight Concurrency Gate:** Điều phối độc quyền qua Windows Named Mutex, triệt tiêu hoàn toàn xung đột dữ liệu và race condition giữa chuyển tự động và thao tác chuyển thủ công trên GUI.
- **Tích hợp Fyne GUI & Trạng thái thời gian thực:** Công tắc gạt con nhộng **AutoSwitchToggle** tại Header Row 2 và thanh trạng thái thời gian thực (`IDLE`, `EVALUATING`, `RETRYING`, `SWITCHING`, `SWITCHED`, `NO_CANDIDATE`, `ERROR`).
- **Danh sách tài khoản hợp nhất:** Gộp tài khoản Chrome, danh bạ `discovered_accounts/` và kho credential `accounts_store/` theo email. Danh bạ chỉ lưu email và thời điểm quét; tài khoản đã phát hiện hoặc đã lưu credential vẫn xuất hiện khi Chrome không còn tìm thấy chúng.
- **Lưu phiên Antigravity:** Tự lấy credential từ Windows Credential Manager khi làm mới danh sách và trước khi chuyển tài khoản. Tray không còn thao tác “Lưu phiên hiện tại” riêng. Card hiển thị `[✔ Dùng]`, `[Chuyển]` hoặc `[Cần OAuth]`; tài khoản thiếu credential không có nút chuyển và hiện quota `N/A`.
- **Chuyển tài khoản đã lưu:** Nạp token, làm mới OAuth khi token sắp hết hạn và có refresh token, đồng bộ thông tin đăng nhập, khởi động lại Language Server rồi xác nhận email đang chạy.
- **Hỗ trợ tiếp tục công việc:** Ghi lại danh sách hội thoại có hoạt động trong 2 giờ gần nhất và gửi prompt tiếp tục sau khi xác nhận tài khoản mới.
- **Dashboard quota:** Hiển thị phần trăm còn lại và thanh tiến trình cho **Gemini / Claude**, với hai chu kỳ **5h / Week** khi API có dữ liệu. Có nút làm mới từng tài khoản, làm mới toàn bộ và tự cập nhật mỗi **60 giây** khi GUI đang chạy.
- **Giao diện desktop hiện đại:** Thẻ tài khoản dạng lưới, đánh dấu tài khoản đang dùng bằng **[✔ Dùng]**, nút **Chuyển**, tooltip email đầy đủ, ba theme **Dark / Light / Warm** và nút ghim **Always on Top**.
- **Tích hợp Windows toàn diện:** Menu khay hệ thống (System Tray), thanh tiêu đề riêng với nút thu nhỏ/phóng to/đóng và cơ chế chỉ mở một phiên GUI duy nhất (Single Instance).
- **Đơn Artifact GUI duy nhất:** Toàn bộ tính năng quản lý, theo dõi quota, chuyển đổi thủ công và tự động được tích hợp trọn vẹn trong một file thực thi duy nhất `antigravity-switch.exe` không cửa sổ console.

## Cơ chế Chuyển Đổi Tự Động (Autonomous Failover Engine)

### 1. Van kích hoạt kép (Dual-Trigger Engine)

Hệ thống kết hợp hai cơ chế kích hoạt bổ trợ nhau:

- **Kích hoạt chủ động (Proactive Trigger):**
  - Chạy đồng bộ cùng vòng lặp đo lường telemetry nền mỗi **60 giây**.
  - Sau mỗi đợt làm mới dữ liệu quota từ Google Cloud Code API, hệ thống phân giải model đang hoạt động trong Language Server (Gemini hoặc Claude) sang nhóm hạn mức tương ứng.
  - Ngưỡng kích hoạt chủ động:
    + **Tài khoản có chu kỳ 5h:** Tự động kích hoạt chuyển đổi khi **Quota 5h <= 2%** (`Threshold5hLow = 0.02`) hoặc **Quota Tuần <= 1%** (`ThresholdWeeklyDepleted = 0.01`).
    + **Tài khoản chỉ có chu kỳ Tuần:** Tự động kích hoạt chuyển đổi khi **Quota Tuần <= 1%** (`ThresholdWeeklyDepleted = 0.01`).
  - *Nguyên tắc fail-safe:* Khi dữ liệu telemetry trả về rỗng, lỗi mạng hoặc trạng thái không chắc chắn (`UNKNOWN_UNCERTAIN`), hệ thống **tuyệt đối không chuyển đổi mù quáng**, bảo vệ an toàn cho tài nguyên tài khoản.

- **Kích hoạt bị động tức thì (Reactive Trigger):**
  - Lắng nghe trực tiếp luồng Connect-RPC streaming giữa Antigravity IDE và Language Server.
  - Ngay khi Language Server phản hồi lỗi cạn kiệt tài nguyên tức thời như mã HTTP `429 Too Many Requests` hoặc mã lỗi gRPC/Connect-RPC `RESOURCE_EXHAUSTED`, van bị động lập tức can thiệp.
  - Hệ thống thẩm định kết quả thực thi trên luồng streaming thật (execution outcome), không chỉ dừng lại ở HTTP ACK sơ bộ, giúp triệt tiêu hoàn toàn các cảnh báo giả.

### 2. Cơ chế phục hồi Native In-Memory 3-Retry

Để loại trừ các trường hợp quá tải vi mô nhất thời (transient rate-limit spike) mà không cần hoán đổi tài khoản:

- **Thử lại nội bộ trước khi chuyển:** Hệ thống gửi yêu cầu `SendUserCascadeMessage` với danh sách nội dung rỗng (`empty content list`) qua Connect-RPC đến Language Server.
- **Ngân sách phục hồi:** Tối đa **3 lần thử lại** (`MaxRecoveryAttempts = 3`) độc lập trên từng luồng hội thoại.
- **Xử lý kết quả:**
  - Nếu 1 trong 3 lần thử thành công: Dừng phục hồi, giữ nguyên tài khoản hiện tại để tiếp tục công việc.
  - Nếu cả 3 lần thử đều xác nhận thất bại do lỗi cạn kiệt tài nguyên (`RESOURCE_EXHAUSTED`): Bộ điều phối (Retry Coordinator) mới chính thức kích hoạt quy trình chuyển đổi sang tài khoản mới.

### 3. Thuật toán Burn-down Sorter

Khi có lệnh chuyển tài khoản, hệ thống áp dụng thuật toán Burn-down Sorter để chọn ra tài khoản kế thừa tối ưu nhất từ kho lưu trữ `accounts_store/`:

#### A. 5 Tiêu chí sơ loại (Pre-filtering Criteria)
Một tài khoản chỉ trở thành ứng viên hợp lệ khi thỏa mãn đồng thời cả 5 tiêu chí:
1. **Khác tài khoản đang dùng:** Không chọn lại tài khoản đang kích hoạt trong Language Server (`Criterion 1`).
2. **Token hợp lệ:** Có file JSON hợp lệ trong `accounts_store/`, nạp thành công vào Windows Credential Manager và refresh token còn hiệu lực (`Criterion 2`).
3. **Không nằm trong danh sách lỗi:** Bỏ qua các tài khoản đã gặp sự cố trong phiên chuyển đổi hiện tại (`Criterion 3`).
4. **Hạn mức trên ngưỡng chuyển:** Quota của nhóm model đang hoạt động phải vượt ngưỡng kích hoạt: Với tài khoản có 5h thì 5h > 2% và Tuần > 1%; với tài khoản chỉ có tuần thì Tuần > 1% (`Criterion 4`).
5. **Dữ liệu quota tin cậy:** Trạng thái quota phải hợp lệ và cập nhật mới, loại bỏ hoàn toàn trạng thái `UNKNOWN_UNCERTAIN` hoặc lỗi kết nối (`Criterion 5`).

#### B. 4 Tầng ưu tiên sắp xếp (Burn-down Sorter Tiers)
Các ứng viên đạt chuẩn sơ loại sẽ được sắp xếp theo 4 tầng ưu tiên:
- **Tầng 1 (Tuần thấp nhất trước - Lowest Remaining Weekly Quota):** Ưu tiên dùng các tài khoản có quota tuần thấp nhất còn khả dụng (ví dụ: 3% -> 10% -> 50%...). Cơ chế này giúp tối đa hóa hiệu suất sử dụng tài nguyên ("burn-down" sạch quota) trước khi bước sang chu kỳ reset tuần mới.
- **Tầng 2 (Ưu tiên tài khoản có chu kỳ 5h):** Giữa các tài khoản có cùng tỷ lệ quota tuần, ưu tiên chọn tài khoản có chu kỳ 5h so với tài khoản chỉ có chu kỳ tuần.
- **Tầng 3 (5h cao nhất trước - Highest Remaining 5h Quota):** Giữa các tài khoản cùng có chu kỳ 5h, ưu tiên chọn tài khoản có quota 5h dồi dào nhất (ví dụ: 80% > 40%) để đảm bảo phiên làm việc kéo dài liên tục.
- **Tầng 4 (Thứ tự danh bạ cố định - Stable Directory Order):** Nếu tất cả các chỉ số trên tương đương, giữ nguyên thứ tự sắp xếp cố định theo index lưu trữ ban đầu để đảm bảo tính tất định (deterministic).

### 4. Cơ chế bảo vệ liên tiến trình Single-Flight Concurrency Gate

Để đảm bảo an toàn tuyệt đối khi vận hành đa luồng và đa tiến trình:
- **Windows Named Mutex:** Hệ thống tạo một Named Mutex trên Windows được cô lập theo Windows Session của từng người dùng (`AntigravitySwitch_SwitchGate_<USERNAME>`), không xung đột với các phiên làm việc khác và tách biệt với GUI Single-Instance Mutex.
- **Đồng bộ toàn diện:**
  - Áp dụng thống nhất cho các luồng chuyển: Chuyển tự động từ Failover Daemon và nút bấm **Chuyển** thủ công trên giao diện Fyne GUI.
  - Khi một thao tác chuyển tài khoản đang diễn ra, mọi lệnh chuyển khác phát sinh từ bất kỳ luồng nào đều bị chặn và từ chối an toàn (`Single-Flight Dropped` hoặc `ErrGateBusy`).
  - Triệt tiêu hoàn toàn xung đột ghi đè Windows Credential Manager, file lock trong `accounts_store/`, hoặc tình trạng khởi động lại Language Server đồng thời.

### 5. Tích hợp Fyne GUI & Trạng thái động thời gian thực

- **Công tắc AutoSwitchToggle:**
  - Nằm ở góc phải của thanh công cụ **Header Row 2** (ngang hàng với nhãn **Tự động** và bộ chọn **Theme**).
  - Thiết kế dạng viên thuốc (capsule pill toggle) hiện đại, hỗ trợ hiệu ứng hover và phản hồi màu sắc theo theme chủ đạo.
  - Bật công tắc để kích hoạt Failover Daemon (ngay lập tức chạy kiểm tra hạn mức chủ động); tắt công tắc để dừng giám sát tự động mà không làm ảnh hưởng đến tác vụ chuyển đang chạy dở dang.
- **Nhãn trạng thái thời gian thực (Status Bar & Server Status):**
  - Cung cấp phản hồi trực quan về chu trình hoạt động của Failover Engine:
    + `IDLE`: Hệ thống sẵn sàng, giám sát nền bình thường.
    + `EVALUATING`: Đang đánh giá hạn mức quota chủ động hoặc thụ động.
    + `RETRYING`: Đang thực hiện 3-Retry phục hồi in-memory với Language Server.
    + `SWITCHING`: Đang thực hiện quy trình chuyển tài khoản, nạp credential và khởi động lại Language Server.
    + `SWITCHED`: Chuyển đổi thành công, đã khôi phục các luồng hội thoại.
    + `NO_CANDIDATE`: Không có tài khoản dự phòng nào đạt đủ 5 tiêu chí sơ loại.
    + `ERROR`: Báo lỗi chi tiết nếu có sự cố trong quá trình hoán đổi.

## Chạy ứng dụng

### Yêu cầu sử dụng

- Windows 10 / 11, bản 64-bit.
- Antigravity IDE đã cài đặt; mở IDE và đăng nhập trước khi lưu hoặc chuyển tài khoản.
- Google Chrome nếu muốn tự phát hiện tài khoản từ các profile Chrome.
- Kết nối mạng để làm mới OAuth và lấy quota từ Google.

### Khởi Chạy Ứng Dụng

Khởi chạy ứng dụng đồ họa trực quan:

```powershell
.\bin\antigravity-switch.exe
```

Hoặc click đúp trực tiếp vào file `bin\antigravity-switch.exe` từ Windows File Explorer. Ứng dụng khởi động ngay tức thì với giao diện Fyne GUI hiện đại, không nhấp nháy cửa sổ console đen (Zero-Console experience).

### Lưu thêm tài khoản

1. Đăng nhập tài khoản cần lưu trong **Antigravity IDE**. Chỉ đăng nhập trên Chrome chưa đủ để Switcher có token Antigravity.
2. Trong Switcher, bấm **Danh sách tài khoản** để quét lại và tự lưu phiên hiện tại. Bấm thêm khi đang quét sẽ được bỏ qua; bấm lại sau khi hoàn tất cập nhật cùng bản ghi, không tạo tài khoản trùng.
3. Lặp lại với các tài khoản còn lại. Mỗi tài khoản được lưu thành một file JSON trong `accounts_store/`.
4. Khi cần đổi tài khoản thủ công, bấm **Chuyển** trên thẻ tài khoản đã lưu.

Nếu chưa có phiên hợp lệ của tài khoản đích trong kho, cần đăng nhập tài khoản đó trong IDE rồi lưu lại trước khi chuyển.

### Khi chuyển tài khoản

Switcher lưu danh sách hội thoại gần đây, cập nhật credential, đưa Antigravity lên trước và khởi động lại `language_server.exe`. Sau thời gian chờ **8 giây**, ứng dụng kiểm tra email trong Language Server mới rồi mới tiếp tục các hội thoại đã ghi nhận.

Prompt tiếp tục được gửi tuần tự, cách nhau **3 giây** giữa các hội thoại. Vì vậy thời gian chuyển còn phụ thuộc vào OAuth, Language Server và số hội thoại; không cố định ở một giây. Từng hội thoại có thể tiếp tục thành công hoặc thất bại riêng, dù tài khoản đã chuyển thành công.

### Đọc quota

- **5h / Week:** quota còn lại theo chu kỳ 5 giờ và tuần mà API trả về.
- **Xanh:** trên 50%; **vàng:** từ 15% đến 50%; **đỏ:** dưới 15%.
- **`--%`:** chưa có dữ liệu, thiếu dữ liệu cho chu kỳ đó hoặc truy vấn bị lỗi; khác với **`0%`** là đã hết quota.
- **Làm mới Quota:** quét các tài khoản đã lưu. Nút làm mới trên từng thẻ chỉ truy vấn tài khoản đó.

Tài khoản đang dùng ưu tiên lấy quota qua Language Server; tài khoản khác dùng refresh token đã lưu để truy vấn API Google. Dữ liệu quota được cache để dùng lại giữa các lần chạy.

## Dữ liệu lưu trên máy

Không dùng server bên thứ 3, tất cả nằm trên PC của bạn.

**Các file token trong kho được lưu dạng JSON văn bản thuần, chưa được mã hóa bằng DPAPI.** Credential của phiên IDE được đọc/ghi riêng qua Windows Credential Manager. Thư mục `accounts_store/` đã được đưa vào `.gitignore`; không chia sẻ các file token.

## Kiến trúc & Artifact

Ứng dụng được thiết kế theo mô hình **Đơn Artifact GUI Duy Nhất**:
- Toàn bộ giao diện desktop Fyne, quản lý tài khoản, đồng bộ credential, kiểm soát quota và hệ thống chuyển đổi tự động (Failover Daemon) được đóng gói bên trong duy nhất một tệp thực thi: `bin/antigravity-switch.exe`.
- Ứng dụng được biên dịch với cờ `-ldflags="-H windowsgui -s -w"` kèm resource PE (`rsrc_windows_amd64.syso`), bảo đảm trải nghiệm **Zero-Console** tuyệt đối (không nhấp nháy cửa sổ đen khi khởi động) và hiển thị biểu tượng icon sắc nét trên thanh tác vụ (Taskbar).
- Cơ chế **Single Instance Lock** bảo vệ ứng dụng chỉ mở duy nhất một phiên làm việc; khi người dùng mở lại file, cửa sổ hiện tại sẽ tự động được đưa lên trên cùng màn hình.

## Build từ mã nguồn

### Công cụ cần có

- **Go 1.26.3 trở lên**.
- **CGO** và trình biên dịch C tương thích: GCC trong **MSYS2 UCRT64 / MinGW-w64**, có trong `PATH`.
- Trình biên dịch tài nguyên Windows: **`windres.exe`** để nhúng icon từ `logo/app.ico`. Script tự tìm trong `PATH` hoặc tại `C:\msys64\ucrt64\bin\windres.exe`.

### Biên dịch ứng dụng bằng PowerShell

Quy trình biên dịch tự động hóa chuẩn cho môi trường Windows:

```powershell
.\scripts\build_windows.ps1 -Clean
```

Các tùy chọn tham số:
- `-Clean`: Dọn dẹp sạch thư mục đầu ra `bin/`, tắt tiến trình cũ trước khi biên dịch mới.
- `-SkipTests`: Bỏ qua khâu pre-flight test để tăng tốc độ đóng gói.
- `-OutDir <thư mục>`: Tùy chỉnh thư mục xuất file nhị phân (mặc định: `bin`).

Kết quả biên dịch xuất ra **DUY NHẤT 1 file artifact**:

```text
bin/
  antigravity-switch.exe   # Ứng dụng Desktop Fyne GUI Zero-Console (kèm PE icon)
```

### Khởi chạy từ mã nguồn

Chạy trực tiếp từ mã nguồn với CGO:

```powershell
$env:CGO_ENABLED = "1"
go run ./cmd/switcher
```

*(Lưu ý: Yêu cầu môi trường GCC/MinGW-w64 trong PATH và CGO_ENABLED=1. Không cần cờ `-tags gui` do mã nguồn đã được hợp nhất hoàn toàn).*
