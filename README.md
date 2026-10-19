# ⚡ Antigravity Multi-Account Switcher

An toàn tuyệt đối vì tuân thủ theo cơ chế chuyển đổi tài khoản được Google cho phép.

Ứng dụng Windows viết bằng **Go**, giúp quản lý nhiều tài khoản Google, theo dõi quota và chuyển tài khoản đang dùng trong Antigravity IDE. Có giao diện desktop **Fyne** và bộ lệnh **CLI**.

Việc chuyển tài khoản do người dùng bấm **Chuyển** hoặc gọi lệnh CLI. Vòng lặp nền tự động làm mới quota, chưa tự chọn tài khoản khác khi hết quota.

## Tính năng hiện có

- **Danh sách tài khoản hợp nhất:** đọc thông tin từ các profile Chrome và kho `accounts_store/`, gộp theo email. Tài khoản đã lưu vẫn xuất hiện ngay cả khi không còn được tìm thấy trên Chrome.
- **Lưu phiên Antigravity:** lấy token của phiên đang đăng nhập từ Windows Credential Manager. Phiên hiện tại được tự thu thập khi làm mới danh sách; cũng có thể lưu qua menu khay hệ thống hoặc `--capture`.
- **Chuyển tài khoản đã lưu:** nạp token, làm mới OAuth khi token sắp hết hạn và có refresh token, đồng bộ thông tin đăng nhập, khởi động lại Language Server rồi xác nhận email đang chạy.
- **Hỗ trợ tiếp tục công việc:** ghi lại danh sách hội thoại có hoạt động trong 2 giờ gần nhất và gửi prompt tiếp tục sau khi xác nhận tài khoản mới.
- **Dashboard quota:** hiển thị phần trăm còn lại và thanh tiến trình cho **Gemini / Claude**, với hai chu kỳ **5h / Week** khi API có dữ liệu. Có nút làm mới từng tài khoản, làm mới toàn bộ và tự cập nhật mỗi **60 giây** khi GUI đang chạy.
- **Giao diện desktop:** thẻ tài khoản dạng lưới, đánh dấu tài khoản đang dùng bằng **[✔ Dùng]**, nút **Chuyển**, tooltip email đầy đủ, ba theme **Dark / Light / Warm** và nút ghim **Always on Top**.
- **Tích hợp Windows:** menu khay hệ thống, thanh tiêu đề riêng với nút thu nhỏ/phóng to/đóng và cơ chế chỉ mở một phiên GUI. Mở GUI lần nữa sẽ đưa cửa sổ đang chạy lên trước.
- **CLI:** xem danh sách tài khoản, trạng thái phiên/Language Server, bảng quota, lưu phiên và chuyển tài khoản.

## Chạy ứng dụng

### Yêu cầu sử dụng

- Windows 10 / 11, bản 64-bit.
- Antigravity IDE đã cài đặt; mở IDE và đăng nhập trước khi lưu hoặc chuyển tài khoản.
- Google Chrome nếu muốn tự phát hiện tài khoản từ các profile Chrome.
- Kết nối mạng để làm mới OAuth và lấy quota từ Google.

Với bản `.exe` đã build, chạy trực tiếp ứng dụng. Các lệnh PowerShell dưới đây thực hiện từ thư mục gốc của repo:

```powershell
.\bin\antigravity-switch-gui.exe
```

### Lưu thêm tài khoản

1. Đăng nhập tài khoản cần lưu trong **Antigravity IDE**. Chỉ đăng nhập trên Chrome chưa đủ để Switcher có token Antigravity.
2. Trong Switcher, bấm **Danh sách tài khoản** để quét lại và tự lưu phiên hiện tại. Hoặc chọn **Lưu phiên hiện tại** trong menu khay hệ thống.
3. Lặp lại với các tài khoản còn lại. Mỗi tài khoản được lưu thành một file JSON trong `accounts_store/`.
4. Khi cần đổi tài khoản, bấm **Chuyển** trên thẻ tài khoản đã lưu.

Cũng có thể lưu phiên hiện tại bằng CLI:

```powershell
.\bin\antigravity-switch.exe --capture
```

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

Kho mặc định là `accounts_store/` của repo. Để dùng thư mục khác, tạo thư mục trước rồi đặt biến môi trường **`ANTIGRAVITY_STORE_DIR`** trước khi chạy ứng dụng.

| Đường dẫn trong kho | Nội dung |
| --- | --- |
| `<email>.json` | Token và thông tin phiên của tài khoản. |
| `.quota_cache.json` | Cache dữ liệu quota. |
| `.ui_preferences.json` | Theme đã chọn. |
| `active_threads_history/` | Danh sách hội thoại ghi nhận trước mỗi lần chuyển. |

**Các file token trong kho được lưu dạng JSON văn bản thuần, chưa được mã hóa bằng DPAPI.** Credential của phiên IDE được đọc/ghi riêng qua Windows Credential Manager. Thư mục `accounts_store/` đã được đưa vào `.gitignore`; không chia sẻ các file token.

## Build từ mã nguồn

### Công cụ cần có

- **Go 1.26.3 trở lên**, theo `go.mod`.
- Với GUI: **CGO** và trình biên dịch C tương thích, ví dụ GCC trong **MSYS2 UCRT64 / MinGW-w64**, có trong `PATH`.
- Với script build Windows: **`windres.exe`** để nhúng icon từ `logo/app.ico`. Script tìm trong `PATH` hoặc `C:\msys64\ucrt64\bin\windres.exe`.

Build cả CLI và GUI bằng PowerShell:

```powershell
.\scripts\build_windows.ps1 -Mode all
```

Có thể dùng `-Mode cli` hoặc `-Mode gui` để build riêng. Script chạy kiểm tra trước khi build và tạo:

```text
bin/
  antigravity-switch.exe       # CLI
  antigravity-switch-gui.exe   # GUI
```

Nếu chỉ cần CLI, có thể build trực tiếp mà không cần CGO hoặc bộ công cụ đồ họa:

```powershell
$env:CGO_ENABLED = "0"
go build -o .\bin\antigravity-switch.exe .\cmd\switcher
```

Chạy GUI từ mã nguồn với GCC đã sẵn sàng:

```powershell
$env:CGO_ENABLED = "1"
go run -tags gui ./cmd/switcher
```

Build GUI cần **`-tags gui`**. Build entrypoint không có tag này sẽ dùng chế độ CLI.

## Kiểm thử

Kiểm thử engine và CLI không cần CGO:

```powershell
$env:CGO_ENABLED = "0"
go test ./internal/model ./internal/wincred ./internal/store ./internal/scanner ./internal/connector ./internal/reloader ./internal/manager ./internal/telemetry ./cmd/switcher
```

Kiểm thử toàn bộ, gồm giao diện Fyne, với GCC đã cài đặt:

```powershell
$env:CGO_ENABLED = "1"
go test -tags gui ./...
```
