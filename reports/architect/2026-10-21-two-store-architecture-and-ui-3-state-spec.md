# Kiến trúc hai kho dữ liệu và giao diện ba trạng thái tài khoản

(Two-Store Architecture & UI 3-State Specification — bản cập nhật sau đối chiếu codebase)

- **Ngày lập:** 2026-10-21.
- **Ngày cập nhật:** 2026-10-21, múi giờ Asia/Bangkok.
- **Phiên bản:** R2.
- **Mã văn kiện:** `SPEC-ARCH-20261021-TWO-STORE-AND-UI-3-STATE`.
- **Cơ sở đối chiếu source:** commit `4213e80`.
- **Trạng thái:** DRAFT — đã bổ sung kết quả review và quyết định của Owner; chưa triển khai thay đổi sản phẩm.
- **Quyết định Owner đã chốt:** bỏ chức năng “Lưu phiên hiện tại” khỏi Tray, bỏ thao tác lưu riêng ở GUI; tiếp tục dùng capture credential trong các luồng hiện có.
- **Phạm vi lần cập nhật này:** cập nhật trực tiếp báo cáo theo yêu cầu Owner. Không sửa source, không chạy build/test, không thao tác credential hoặc chuyển tài khoản.
- **Người đọc:** Owner và người triển khai sau khi phương án được chốt.

Bản R2 thay thế nội dung đề xuất R1 ngay trong file này theo yêu cầu Owner. Các nhận định cũ không còn đúng được ghi rõ ở phần XII; không sử dụng chúng làm chỉ dẫn triển khai. Việc cập nhật tài liệu không đồng nghĩa tính năng đã được thực hiện hoặc đã nghiệm thu.

## I. Mục tiêu và phạm vi

### 1. Mục tiêu

1. Hiển thị đúng khả năng sử dụng tài khoản: đang dùng, có credential đã lưu, hoặc cần đăng nhập/lưu credential.
2. Ngăn giao diện cung cấp nút chuyển cho tài khoản mà ứng dụng đã biết chưa có credential.
3. Lưu danh bạ tài khoản phát hiện từ Chrome bằng JSON đơn giản, không chứa token.
4. Giữ các tài khoản đã lưu credential trong danh sách dù chúng không còn được Chrome phát hiện.
5. Bỏ thao tác “Lưu phiên hiện tại” riêng, tận dụng capture đã có khi lấy danh sách và trước khi chuyển tài khoản.
6. Giữ dữ liệu hiện hữu khi chuẩn hóa đường dẫn và tránh đặt dữ liệu lâu dài vào thư mục bị clean build xóa.
7. Bảo tồn engine chuyển tài khoản, telemetry, automation và continuation đang hoạt động.

### 2. Nguyên tắc thực hiện

- Tận dụng manager, scanner, store, quota cache và cơ chế cập nhật Fyne hiện có.
- Không thêm daemon, service, cơ sở dữ liệu hoặc hệ thống OAuth mới.
- Ba trạng thái là cách trình bày trên card; không cần xây lại toàn bộ mô hình trạng thái nghiệp vụ.
- Không thêm vòng kiểm tra online chỉ để quyết định vẽ nút.
- Phân biệt dữ liệu danh bạ, credential và kết quả truy vấn quota.
- Mọi mô tả “hiện có” phải được phân biệt với “hành vi đích”.
- Các tên hàm/file ở phần bằng chứng và gợi ý tích hợp giúp tìm đúng code; không bắt buộc tạo thêm lớp/helper chỉ để khớp tên trong tài liệu.

## II. Những gì đã xác nhận trong codebase

### 1. Luồng hiện có

Khi người dùng bấm “Danh sách tài khoản”, ứng dụng gọi luồng lấy trạng thái tài khoản. Luồng đó thử capture credential hiện tại, xác định danh tính active, quét Chrome, đọc các tài khoản đã lưu và hợp nhất theo email.

~~~text
“Danh sách tài khoản”
    → Capture credential hiện tại từ Windows Credential Manager
    → Lưu credential nếu đọc được dữ liệu hợp lệ
    → Xác định tài khoản active bằng cơ chế hiện có
    → Quét tài khoản Chrome
    → Đọc tài khoản có credential trong accounts_store
    → Hợp nhất, phân loại và cập nhật UI
~~~

“Capture tự động” ở đây nghĩa là được thực hiện trong những luồng gọi hiện có. Code không có watcher riêng liên tục lưu mọi lần đăng nhập bên ngoài ứng dụng.

### 2. Bằng chứng và hệ quả

| ID | Phát hiện từ source | Hệ quả đối với phương án |
|---|---|---|
| F01 | [Manager đã hợp nhất Chrome và store, có thông tin active/token](G:/antigravity_auto_switch/internal/manager/account_manager.go:251) | Không cần tạo lại bộ phân loại tài khoản từ đầu. |
| F02 | [Card hiện vẽ nút chuyển cho mọi tài khoản không active](G:/antigravity_auto_switch/internal/ui/widgets.go:349) | Lỗi cho phép bấm chuyển tài khoản thiếu credential là có thật. |
| F03 | [Nhánh giữ card chỉ so email và active, sau đó cập nhật quota](G:/antigravity_auto_switch/internal/ui/app.go:974) | Chỉ sửa lúc tạo card chưa đủ; thay đổi tình trạng token cũng phải cập nhật cụm hành động. |
| F04 | [Store kiểm tra token trước khi lưu](G:/antigravity_auto_switch/internal/store/account_store.go:69) và [lọc dữ liệu khi liệt kê](G:/antigravity_auto_switch/internal/store/account_store.go:201) | Kho credential hiện không chủ đích lưu danh bạ Chrome rỗng token; lỗi UI không chứng minh hai loại dữ liệu đã bị trộn trong kho. |
| F05 | [Token hợp lệ về cấu trúc khi có access token hoặc refresh token](G:/antigravity_auto_switch/internal/model/types.go:161) | Có credential đọc được không bảo đảm Google vẫn chấp nhận token tại mọi thời điểm. |
| F06 | [Menu Tray đang gọi thao tác lưu phiên](G:/antigravity_auto_switch/internal/ui/tray.go:30) | Đây là thao tác còn đường gọi. Việc bỏ nó là quyết định giản lược sản phẩm của Owner, không chỉ là xóa dead code. |
| F07 | [Capture hiện có tại manager](G:/antigravity_auto_switch/internal/manager/account_manager.go:192), được gọi khi lấy danh sách và trước khi chuyển | Giữ capture dù bỏ thao tác lưu riêng trong Tray/GUI. |
| F08 | [Refresh có cờ bận, bỏ qua yêu cầu khi đang bận](G:/antigravity_auto_switch/internal/ui/app.go:624) | Bấm dồn không xếp hàng nhiều lượt refresh; lượt sau khi hoàn tất cập nhật cùng dữ liệu. |
| F09 | [Resolver đã hỗ trợ biến môi trường và tìm kho hiện tại](G:/antigravity_auto_switch/internal/store/sanitizer.go:113) | Có thể chuẩn hóa trên cơ chế sẵn có, không cần thay bằng quy tắc luôn cạnh exe. |
| F10 | [Build mặc định ra bin và Clean xóa thư mục đầu ra](G:/antigravity_auto_switch/scripts/build_windows.ps1:5) | Đặt dữ liệu dưới bin có thể làm mất dữ liệu khi clean build. |
| F11 | [Cache và preferences nằm trong accounts_store](G:/antigravity_auto_switch/internal/ui/app.go:101); [lịch sử thread cũng đặt dưới kho này](G:/antigravity_auto_switch/internal/reloader/active_threads.go:38) | Không được dọn sạch các file phụ với lý do “không có token_data”. |
| F12 | [Engine đặt thời gian chờ restart Language Server là 8 giây](G:/antigravity_auto_switch/internal/manager/account_manager.go:488) | Không thể cam kết chuyển tài khoản thật hoàn tất dưới 1,5 giây với engine hiện tại. |
| F13 | [Cửa sổ khởi tạo ở kích thước mặc định và cho phép resize](G:/antigravity_auto_switch/internal/ui/app.go:279) | 920 × 630 là kích thước khởi tạo, không phải yêu cầu khóa cửa sổ. |
| F14 | [Telemetry lấy danh sách từ credential store](G:/antigravity_auto_switch/internal/telemetry/pool.go:53); [automation cũng lấy ứng viên từ store](G:/antigravity_auto_switch/internal/automation/controller.go:484) | Danh bạ Chrome mới không trở thành nguồn ứng viên chuyển tài khoản. |

Các bằng chứng trên là đối chiếu source. Chúng không phải kết quả chạy E2E cho phương án hai kho chưa được triển khai.

## III. Hợp đồng hai kho dữ liệu

### 1. Vai trò và quan hệ

| Nội dung | Danh bạ discovered_accounts | Kho accounts_store |
|---|---|---|
| Dữ liệu chính | Email đã được phát hiện từ Chrome và thời điểm quét thấy | Credential Antigravity đã lưu |
| Nguồn cập nhật | Kết quả scanner hiện có | Capture credential hiện có và những luồng cập nhật credential đang có |
| Có token hay không | Tuyệt đối không chứa token | File tài khoản chứa credential theo schema hiện tại |
| Vai trò với UI | Bổ sung tài khoản đã được nhận diện | Xác định ứng dụng đã có credential của tài khoản |
| Vai trò với automation | Không cung cấp ứng viên có quyền sử dụng | Tiếp tục là nguồn ứng viên |
| Vai trò với telemetry hàng loạt | Không mở rộng danh sách polling sang tài khoản thiếu credential | Tiếp tục là nguồn tài khoản cần polling |

**Cùng một email được phép xuất hiện ở cả hai kho.** “Được Chrome phát hiện” không đồng nghĩa “chưa OAuth”. Hai kho phân tách loại dữ liệu, không phân chia hai nhóm tài khoản loại trừ nhau.

Không cần di chuyển hay xóa bản ghi danh bạ khi lưu được credential. UI hợp nhất theo email và suy ra trạng thái ở thời điểm cập nhật.

Credential của một tài khoản vẫn được chấp nhận theo cơ chế hiện có khi tài khoản đó không còn được scanner tìm thấy. Không dùng việc có/không có profile Chrome làm điều kiện bắt buộc để dùng credential đã lưu.

### 2. Schema danh bạ

Mỗi file tài khoản trong danh bạ chỉ có hai trường:

~~~json
{
  "email": "developer@gmail.com",
  "scanned_at": 1792500000
}
~~~

- `email`: email chuẩn hóa bằng cách bỏ khoảng trắng đầu/cuối và thống nhất chữ thường theo cách hiện có.
- `scanned_at`: Unix timestamp theo giây của lần gần nhất scanner trả về email đó; có thể dùng phần thập phân như quy ước timestamp của kho hiện tại.
- Không ghi token, quota, trạng thái active hoặc cờ có token vào danh bạ.
- Không ghi `name`, `profile_dir`, `avatar_url` vào JSON danh bạ.
- Việc giản lược JSON danh bạ không yêu cầu xóa metadata khỏi scanner hoặc khỏi những kiểu dữ liệu đang dùng cho mục đích khác.

Tên file và đường dẫn phải sử dụng cơ chế chuẩn hóa an toàn hiện có. Cùng một email chuẩn hóa luôn cập nhật cùng một bản ghi, không tạo file mới sau mỗi lần refresh.

### 3. Vòng đời danh bạ

1. Khởi tạo danh bạ cùng quá trình khởi tạo các phụ thuộc của manager, trước khi sử dụng.
2. Mỗi lần quét thành công, thêm hoặc cập nhật các email được phát hiện.
3. Đọc danh bạ đã lưu để tham gia dựng danh sách; không chỉ ghi ra đĩa rồi bỏ qua đường đọc.
4. Một lần quét không thấy email không tự động xóa bản ghi cũ.
5. Khi chỉ đọc lại bản ghi cũ, không tự cập nhật thời điểm “quét thấy” của email đó.
6. Chưa bổ sung cơ chế xóa/prune danh bạ tự động trong phạm vi này.
7. Lỗi đọc/ghi danh bạ không được xóa credential hoặc làm mất những tài khoản còn đọc được từ nguồn khác.
8. Phản hồi lỗi lưu cần thiết qua cơ chế trạng thái hiện có; không báo lưu thành công nếu thao tác ghi thất bại.

Ghi file theo cơ chế thay thế file an toàn phù hợp Windows, tận dụng cách làm hiện có; tránh ghi trực tiếp làm cụt file đích. Không tuyên bố cơ chế ghi trên đĩa “không thể lỗi” trong mọi tình huống.

### 4. Bảo tồn schema credential và dữ liệu phụ

File tài khoản trong `accounts_store` tiếp tục dùng envelope và khả năng đọc/ghi credential hiện tại. Không viết lại token bằng một mẫu JSON rút gọn trong tài liệu.

Giữ khả năng bảo toàn payload gốc, định dạng token lồng nhau và các trường như auth method, token type, expiry theo [TokenData hiện có](G:/antigravity_auto_switch/internal/model/types.go:120).

Yêu cầu “có token_data” áp dụng cho file tài khoản. Thư mục kho hiện còn có:

- `.quota_cache.json`;
- `.ui_preferences.json`;
- `active_threads_history/`.

Giữ những dữ liệu phụ này và đường liên kết của các thành phần sử dụng chúng. Không tái cấu trúc toàn bộ bố cục lưu trữ trong phạm vi chuẩn hóa card và thêm danh bạ.

### 5. Giới hạn bảo vệ dữ liệu

Giữ `accounts_store/` và bổ sung `discovered_accounts/` vào `.gitignore`. Đây là biện pháp tránh đưa dữ liệu người dùng vào Git, không phải cơ chế mã hóa.

Kho hiện lưu token bằng JSON văn bản thuần. Báo cáo không cam kết “bảo mật 100%” và không mở thêm một dự án mã hóa credential trong phạm vi này. Không đưa credential thật vào tài liệu, log kiểm tra hoặc fixture.

## IV. Luồng lấy danh sách và hành vi khi bấm nhiều lần

### 1. Luồng đích cho một lần lấy danh sách

~~~text
Người dùng bấm “Danh sách tài khoản”
    → Áp dụng cơ chế chống chạy chồng hiện có
    → Thử capture credential hiện tại
    → Xác định danh tính active bằng cơ chế hiện có
    → Quét Chrome
    → Lưu/cập nhật các email vừa phát hiện vào danh bạ
    → Đọc danh bạ đã lưu và các tài khoản có credential
    → Hợp nhất với kết quả quét hiện tại, loại trùng theo email
    → Xác định trạng thái tài khoản
    → Cập nhật card và phản hồi cần thiết
    → Kết thúc lượt làm mới
~~~

Danh sách hợp nhất sử dụng kết quả quét hiện tại kể cả khi việc lưu danh bạ thất bại. Việc lưu thất bại phải được phản hồi phù hợp, không được khiến kết quả quét biến mất khỏi UI trong lượt đó.

Giữ cơ chế nhận diện active hiện có: ưu tiên trạng thái Language Server, có fallback credential như code hiện tại. Phương án này không thay đổi ý nghĩa nguồn active bằng một cơ chế xác minh mới.

Giữ cách sắp xếp hiện có: active trước, tiếp theo tài khoản có credential, sau đó các tài khoản còn lại; thứ tự email ổn định trong từng nhóm. Không dùng thời điểm quét danh bạ để thay thứ tự xếp hạng ứng viên automation.

### 2. Bấm nút nhiều lần có gây vấn đề không?

**Có thể bấm lại.** Hành vi cần giữ và hoàn thiện như sau:

| Cách bấm | Hành vi |
|---|---|
| Bấm khi lượt refresh trước chưa hoàn tất | Cờ bận hiện có bỏ qua lần bấm thêm; không xếp hàng thêm lượt quét |
| Bấm khi ứng dụng đang bận thao tác thủ công dùng cùng cờ bận | Giữ cách bỏ qua yêu cầu hiện có |
| Bấm lại sau khi lượt trước hoàn tất | Thực hiện một lượt quét/capture/cập nhật mới |
| Cùng email được quét hoặc capture nhiều lần | Cập nhật cùng bản ghi; không tăng số tài khoản hoặc số card vì trùng lặp |
| Lượt mới không có thay đổi danh tính/credential | Giữ trạng thái hiển thị phù hợp, cập nhật thông tin cần thiết |
| Credential mới được lưu hoặc không còn đọc được | Cập nhật cụm hành động của card trong lượt làm mới |
| Bấm nhiều lần sau mỗi lần hoàn tất | Có thêm chi phí đọc/ghi và truy vấn nhận diện; không cần thao tác như vậy khi dữ liệu không đổi |

Tính chất lặp lại an toàn ở đây là không tạo bản sao tài khoản, không thêm hàng đợi và không khởi tạo thao tác chuyển tài khoản từ nút lấy danh sách. Timestamp có thể được cập nhật và file credential có thể được ghi lại theo dữ liệu mới, nên không có nghĩa mỗi lần bấm hoàn toàn không ghi dữ liệu.

Bản thân nút lấy danh sách không ra lệnh chuyển tài khoản. Khi automation đang bật, automation vẫn có thể chuyển theo điều kiện riêng; không mô tả nút refresh như một công tắc tạm dừng automation.

Không thêm bộ đếm retry, delay hoặc hàng đợi riêng cho nút này. Tận dụng cờ bận hiện có; hoàn tất cả nhánh thành công và lỗi đều phải trả UI về trạng thái có thể thao tác lại.

### 3. Capture và phản hồi lỗi

| Tình huống | Xử lý đích |
|---|---|
| Có credential hợp lệ và lưu thành công | Dùng kết quả store để cập nhật tình trạng tài khoản |
| Chưa có phiên credential để capture | Tiếp tục hiển thị danh sách; đây có thể là trạng thái bình thường |
| Đọc credential gặp lỗi thực sự hoặc ghi file thất bại | Có phản hồi ngắn trên thanh trạng thái hiện có; vẫn hiển thị phần danh sách lấy được |
| Không đọc được một nguồn danh bạ | Không xóa nguồn còn lại hoặc credential đã lưu |
| Không nhận diện được tài khoản active | Không tự chọn một tài khoản bất kỳ làm active |

Không chuyển toàn bộ việc lấy danh sách thành thất bại chỉ vì chưa có phiên đăng nhập. Đồng thời không nuốt lỗi ghi credential rồi khiến người dùng hiểu rằng tài khoản đã được lưu.

## V. Quyết định Owner: bỏ “Lưu phiên hiện tại” trong Tray

### 1. Phạm vi đã chốt

| Thành phần | Quyết định |
|---|---|
| Menu “Lưu phiên hiện tại” trong Tray | Xóa |
| Thao tác lưu phiên riêng của GUI | Xóa |
| Tham số/callback lưu phiên trong các hàm tạo card | Xóa |
| Những nơi truyền callback lưu phiên không còn dùng | Xóa |
| Capture credential trong manager | Giữ |
| Capture khi lấy danh sách | Giữ |
| Capture trước khi chuyển tài khoản | Giữ |

Ở baseline, thao tác lưu riêng vẫn được Tray gọi. Owner quyết định bỏ do không cần sử dụng thao tác này. Sau khi bỏ cả menu và các callback, mới loại bỏ hàm GUI tương ứng.

Không xóa cả khay hệ thống. Các menu khác, icon, hành vi cửa sổ và cơ chế đóng ứng dụng nằm ngoài thay đổi này.

### 2. Quy trình dành cho người dùng

~~~text
Đăng nhập tài khoản trong Antigravity
    → Bấm “Danh sách tài khoản” trong Switcher
    → Ứng dụng capture và lưu credential bằng cơ chế hiện có
    → Card phản ánh trạng thái hiện tại
~~~

Tài khoản vừa được capture mà vẫn đang active hiển thị `[✔ Dùng]`. Khi chuyển sang tài khoản khác, tài khoản đã lưu có thể hiển thị `[Chuyển]`.

Menu “Làm mới Chrome” còn lại trong Tray đang dùng cùng luồng refresh và tiếp tục dùng luồng đó.

Không bổ sung watcher OAuth mới để thay menu đã xóa. Cũng không biến icon làm mới quota trên card thành một nút lưu phiên hoặc đăng nhập khác.

### 3. Phản hồi sau khi bỏ thao tác lưu riêng

Di chuyển phản hồi capture cần thiết vào luồng lấy danh sách bằng thanh trạng thái hiện có. Giữ thông tin thành công/thất bại đúng thực tế, áp dụng quy tắc ở IV.3 và cơ chế ẩn email hiện có khi hiển thị thông báo chứa email.

Không yêu cầu người dùng tìm một nút “Lưu phiên” khác sau khi chức năng này đã được bỏ.

## VI. Giao diện ba trạng thái và cập nhật tại chỗ

### 1. Bảng trạng thái

| Ưu tiên | Điều kiện | Hiển thị | Hành vi |
|---|---|---|---|
| 1 | Được cơ chế nhận diện hiện có xác định là tài khoản active | `[✔ Dùng]` | Không hiển thị nút chuyển tới chính tài khoản đó |
| 2 | Không active, có credential đọc được trong kho | `[Chuyển]` | Một lần bấm yêu cầu engine hiện có chuyển tài khoản |
| 3 | Không active, chưa có credential đọc được trong kho | `[Cần OAuth]` | Không có nút chuyển |

Trạng thái active có ưu tiên hiển thị cao nhất. Trạng thái active và việc có bản credential lưu trên đĩa là hai thông tin độc lập; nếu active nhưng capture gặp lỗi, vẫn phản ánh active theo nguồn hiện có và phản hồi lỗi lưu riêng.

`[Cần OAuth]` được giữ làm nhãn sản phẩm. Ý nghĩa là ứng dụng chưa có credential đọc được để dùng lại, không phải một khẳng định rằng tài khoản chưa từng OAuth trong quá khứ.

Có credential không đồng nghĩa quota luôn đủ hoặc token không thể bị thu hồi. Giữ cơ chế refresh, preflight và phản hồi lỗi hiện có; không thêm trạng thái card thứ tư chỉ để xác minh online mỗi lần vẽ.

### 2. Sửa cả lúc tạo và lúc cập nhật

Hành vi đúng phải áp dụng khi:

- mở ứng dụng và dựng card lần đầu;
- lấy lại danh sách;
- capture thành công;
- phát hiện credential không còn đọc được;
- hoàn tất chuyển thủ công hoặc tự động;
- đổi theme hoặc dựng lại card vì thay đổi danh sách.

Nếu email và thứ tự card không đổi nhưng tình trạng credential đổi, cụm hành động vẫn phải cập nhật. Nhánh chỉ cập nhật quota không được giữ lại nút chuyển hoặc nhãn OAuth đã lỗi thời.

Tận dụng cập nhật tại chỗ: có thể thay phần nhãn/nút bên trong card mà giữ bảng quota, email và khung card. Không thêm cơ chế định kỳ dựng lại toàn bộ canvas chỉ để cập nhật một cờ trạng thái.

Khi trạng thái active đổi, cập nhật đồng bộ nhãn, nền/viền active và thông tin hành động. Khi danh sách thay đổi hoặc đổi thứ tự, cơ chế dựng lại hiện có vẫn có thể được dùng theo nhu cầu thực tế.

### 3. Bố cục và hình thức

- Giữ kích thước khởi tạo cửa sổ 920 × 630 và khả năng resize/maximize/restore hiện có.
- Giữ card 280 × 130 trong grid hiện tại.
- Giữ email ellipsis, tooltip, chức năng ẩn tài khoản, vị trí bảng quota và theme.
- Giữ kiểu nút chuyển hiện có, không tự đổi thành loại nút khác do diễn giải chữ “chữ nhật” của R1.
- Nhãn trạng thái dùng cỡ chữ nhỏ, đậm theo hệ thống hiện tại; màu active và nhãn cần OAuth phải lấy từ palette tương ứng.
- Nếu yêu cầu màu hổ phách/xám cho nhãn cần OAuth, phải áp dụng màu thực tế; chỉ đặt chữ đậm không tự tạo màu đó.
- Không mô tả thao tác chuyển là “click đúp”; hành vi nút hiện có là bấm để gọi thao tác.

### 4. Giới hạn của việc ngăn popup

Mục tiêu là ngăn UI cho bấm chuyển khi đã biết tài khoản thiếu credential. Không thể cam kết không còn bất kỳ lỗi nào sau khi bấm.

Credential có thể bị xóa sau khi UI được cập nhật, token có thể bị thu hồi hoặc restart có thể thất bại. Giữ kiểm tra và xử lý lỗi ở engine; không bỏ chúng vì UI đã kiểm tra tình trạng credential.

## VII. Quota trên card

### 1. Quy tắc hiển thị

| Tình huống | Hành vi đích |
|---|---|
| Tài khoản không active và chưa có credential đọc được | Hiển thị `N/A`; không giữ số quota cũ tạo cảm giác tài khoản đã sẵn sàng sử dụng |
| Có credential nhưng chưa có dữ liệu quota hoặc truy vấn lỗi | Giữ cách biểu diễn thiếu dữ liệu hiện có, chẳng hạn `--%`; không đổi thành 0% |
| Có dữ liệu quota hợp lệ | Hiển thị phần trăm, thanh quota và reset theo dữ liệu hiện có |
| Một chu kỳ quota không áp dụng | Giữ cơ chế phân loại chu kỳ hiện có; không suy từ việc thiếu credential |
| Tài khoản active nhưng capture chưa thành công | Giữ khả năng đọc quota active bằng cơ chế hiện có; không áp dụng máy móc quy tắc của tài khoản inactive thiếu credential |

N/A do chưa có credential là trạng thái của card. Nó không được ghi thành quota giả trong cache và không trở thành tín hiệu quota exhausted cho automation.

Không phát sinh lời hứa “có token thì quota hiển thị đầy đủ”. Những tài khoản khác nhau có thể áp dụng các chu kỳ quota khác nhau; truy vấn cũng có thể thất bại.

### 2. Đường cập nhật phải đồng nhất

Cả dựng mới, cập nhật quota tại chỗ, làm mới từng tài khoản và làm mới hàng loạt đều phải tôn trọng tình trạng credential của card.

Nếu chỉ đặt N/A lúc dựng mới mà cập nhật quota lại ghi đè thành `--%` hoặc số cũ thì yêu cầu chưa hoàn tất.

Giữ bố cục cụm làm mới quota. Không dùng thao tác làm mới quota để tạo credential, khởi chạy OAuth hoặc đưa tài khoản chỉ có danh bạ vào polling hàng loạt.

## VIII. Đường dẫn dữ liệu và tương thích kho hiện có

### 1. Rủi ro của quy tắc luôn cạnh exe

Ở cấu trúc build hiện tại, file thực thi nằm trong thư mục đầu ra `bin/`. Nếu luôn đặt kho cạnh exe, dữ liệu sẽ nằm dưới `bin/`.

Script clean build đang xóa toàn bộ thư mục đầu ra, nên cấu hình đó có thể xóa cả token, cache, preferences và danh bạ. Đổi sang kho mới dưới bin còn khiến ứng dụng không đọc kho cũ dù dữ liệu cũ chưa mất.

Do đó, R2 loại bỏ quy tắc “luôn ưu tiên thư mục exe bất kể môi trường”.

### 2. Quy tắc chọn vị trí

| Môi trường | Hướng xử lý |
|---|---|
| Có cấu hình `ANTIGRAVITY_STORE_DIR` | Tôn trọng kho được cấu hình; danh bạ nằm cạnh thư mục kho token đã chọn |
| Chạy trong repo, bao gồm exe đặt dưới bin | Tiếp tục xác định kho hiện có ở gốc repo theo cơ chế tìm root; không tạo mặc định dưới output build |
| Chạy bản portable độc lập đã có kho được resolver tìm thấy | Tiếp tục dùng kho đó, tránh âm thầm tạo kho mới |
| Bản portable mới, không có cấu hình hoặc kho trước đó | Có thể tạo hai kho cạnh exe khi đó là vị trí lưu dữ liệu lâu dài và ghi được |
| Vị trí được cấu hình/chọn không dùng được | Phản hồi lỗi; không âm thầm đổi sang CWD hoặc kho khác rồi ghi dữ liệu vào đó |

Với cấu hình tùy chọn, hai thư mục nằm cạnh nhau; không thêm một biến môi trường mới cho danh bạ nếu không cần thiết. Các thành phần dùng kho token tiếp tục dựa trên cùng kết quả xác định vị trí.

Giữ cơ chế tìm kho từ repo/executable hiện có nơi nó phù hợp. Bỏ fallback đường dẫn máy cá nhân hardcode. Không lấy thư mục khởi chạy ngẫu nhiên làm vị trí ghi mặc định khi đã xác định được vị trí dữ liệu.

Chạy phát triển qua công cụ tạo executable tạm vẫn phải tìm được kho repo phù hợp. Kiểm thử sau này sử dụng đường dẫn fixture được inject, tránh đụng kho credential thật.

### 3. Bố cục trong workspace hiện tại

Bố cục khuyến nghị sau triển khai:

~~~text
G:/antigravity_auto_switch/
    accounts_store/          # Kho đang dùng; giữ credential và dữ liệu phụ
    discovered_accounts/     # Danh bạ mới
    bin/
        antigravity-switch.exe
~~~

Đây là minh họa cho workspace, không phải đường dẫn được hardcode vào chương trình.

Không cần di chuyển kho hiện tại chỉ để thêm danh bạ. Nếu lựa chọn triển khai sau này đòi hỏi đổi vị trí dữ liệu, phải mô tả riêng cách giữ/đọc dữ liệu cũ trước khi thay đổi; không coi tạo một thư mục rỗng là migration thành công.

### 4. Bảo tồn các dữ liệu cùng kho

Cùng việc xác định vị trí credential, phải giữ đúng vị trí cache quota, preferences và lịch sử thread. Không để một thành phần dùng kho cũ còn thành phần khác ghi sang kho mới.

Clean build trong repo không được xóa dữ liệu tài khoản. Với bố cục dữ liệu ngoài output build, có thể đạt yêu cầu này mà không cần mở rộng thành sửa lại toàn bộ script build.

## IX. Bảo tồn chuyển tài khoản, telemetry và automation

### 1. Engine chuyển tài khoản

Giữ cơ chế đã có:

1. Nhận diện tài khoản hiện tại và xử lý trường hợp đã là tài khoản đích.
2. Ghi nhận danh sách thread phục vụ continuation.
3. Capture phiên hiện tại theo cơ chế đang dùng.
4. Đọc credential tài khoản đích và xử lý refresh hiện có.
5. Đồng bộ ToS và Windows Credential Manager.
6. Restart Language Server, chờ theo cơ chế hiện tại.
7. Kết nối lại và xác minh danh tính.
8. Tiếp tục các thread theo quy trình hiện có.

Không thay các bước này bằng ReAuth nóng hoặc một engine khác trong phạm vi hai kho/card.

### 2. Mô tả thời gian và kết quả

Mô tả sản phẩm phù hợp:

> Chuyển bằng credential đã lưu, không cần đăng nhập lại qua trình duyệt; ứng dụng thực hiện restart Language Server, xác minh tài khoản đích và continuation theo cơ chế hiện có.

Engine hiện đặt thời gian chờ restart là 8 giây. Tổng thời gian còn phụ thuộc các bước khác và số thread cần xử lý. Không cam kết chuyển thật dưới 1,5 giây; không đưa một số đo tổng thời gian chưa đo làm tiêu chí nghiệm thu.

Thành công/thất bại và tiến độ tiếp tục phản ánh kết quả từ engine. Tài liệu này không thay tiêu chí thành công của engine hoặc hứa rollback mới.

### 3. Nguồn ứng viên và cổng chuyển

- Telemetry hàng loạt tiếp tục lấy tài khoản từ kho credential.
- Automation tiếp tục lấy ứng viên từ kho credential và áp dụng bộ lọc, preflight, xếp hạng đang có.
- Không lấy danh bạ Chrome làm bằng chứng tài khoản đủ điều kiện chuyển.
- Giữ cổng khóa chung giữa chuyển thủ công và tự động, vòng đời monitoring/retry và continuation.
- Không thay ngưỡng quota, cách ánh xạ model, số lần khôi phục hoặc chính sách xếp hạng trong phạm vi này.
- Bỏ menu lưu phiên không được làm mất cơ chế capture trước khi chuyển.

## X. Gợi ý tích hợp vào code hiện có

Phần này là bản đồ vị trí triển khai, không phải yêu cầu tạo thêm kiến trúc. Cách chia helper có thể điều chỉnh miễn giữ các hợp đồng ở trên.

| Nhóm/file | Thay đổi dự kiến | Điều cần bảo tồn |
|---|---|---|
| `internal/store/discovered_store.go` — file dự kiến bổ sung | Lưu/đọc danh bạ hai trường; chuẩn hóa email và đường dẫn; trả lỗi đọc/ghi phù hợp | Không tái sử dụng envelope token để tạo bản ghi thiếu credential |
| [Store và resolver](G:/antigravity_auto_switch/internal/store/sanitizer.go:113) | Thống nhất vị trí hai kho, giữ kho cũ/override, bỏ hardcode máy cá nhân | Schema credential và dữ liệu phụ |
| [Manager](G:/antigravity_auto_switch/internal/manager/account_manager.go:75) | Khởi tạo danh bạ, nối cả đường ghi và đọc vào luồng lấy danh sách, hợp nhất email | Capture và nhận diện active hiện có; credential-only account vẫn xuất hiện |
| [Card](G:/antigravity_auto_switch/internal/ui/widgets.go:327) | Tạo ba trạng thái, cập nhật trạng thái tại chỗ, bỏ callback lưu riêng | Bố cục, email masking, theme và các hành vi card khác |
| [App](G:/antigravity_auto_switch/internal/ui/app.go:624) | Nối trạng thái mới vào refresh/updateUI, phản hồi capture cần thiết, bỏ thao tác lưu riêng | Cờ bận, Fyne main-thread updates, switch gate và monitoring |
| [Quota UI](G:/antigravity_auto_switch/internal/ui/quota_badge.go:632) | Áp dụng N/A theo trạng thái card trên cả đường dựng/cập nhật | Không biến thiếu dữ liệu thành 0% hoặc thay đổi phân loại quota |
| [Tray](G:/antigravity_auto_switch/internal/ui/tray.go:17) | Xóa menu “Lưu phiên hiện tại” | Menu khác, icon và hành vi cửa sổ |
| [.gitignore](G:/antigravity_auto_switch/.gitignore) | Bổ sung discovered_accounts, giữ accounts_store | Không tự đưa dữ liệu người dùng vào Git |
| [Test UI hiện có](G:/antigravity_auto_switch/internal/ui/ui_test.go:302) và test manager/store liên quan | Cập nhật hành vi và chữ ký liên quan sau khi sửa production code | Dùng fixture tách biệt, không thao tác credential thật |

Các điểm dễ bỏ sót khi triển khai:

- Bỏ `onSave` ở cả `NewAccountCardWidgetWithQuota` và hàm bọc `NewAccountCardWithQuota`, không chỉ ở một chữ ký.
- Cập nhật cả nơi dựng toàn bộ card và nơi dựng/cập nhật một card khi làm mới quota.
- Bỏ `App.SaveSession()` và lời gọi từ Tray; giữ `AccountManager.CaptureCurrentSession()`.
- Theo dõi thay đổi `HasToken` bên cạnh `IsActive`; không chỉ sửa `DisplayBadge` trong manager vì card hiện tự dựng cụm hành động.
- Khi đổi schema danh bạ, không sửa codec `TokenData` để khớp ví dụ rút gọn.
- Phản hồi capture phải đến được UI; tiếp tục bỏ qua mọi lỗi như baseline sẽ làm mất thông tin sau khi menu lưu riêng bị xóa.
- Các test giả định mọi tài khoản inactive đều có nút chuyển cần được cập nhật theo hành vi mới.

## XI. Các trường hợp và tiêu chí nghiệm thu sau triển khai

Các mục dưới đây là yêu cầu kiểm tra trong giai đoạn triển khai sau này. Chưa mục nào được tuyên bố PASS runtime chỉ bởi việc cập nhật báo cáo.

| ID | Tình huống | Kết quả yêu cầu |
|---|---|---|
| AC01 | Chrome phát hiện tài khoản chưa có credential | Danh bạ có bản ghi hai trường; UI hiện [Cần OAuth], không có nút chuyển |
| AC02 | Tài khoản có credential, không active | Hiện [Chuyển], bấm gọi engine hiện có |
| AC03 | Tài khoản active | Hiện [✔ Dùng], không có nút chuyển tới chính nó |
| AC04 | Cùng email có trong Chrome, danh bạ và credential store | Một card duy nhất, trạng thái suy từ active/credential |
| AC05 | Chỉ còn credential trong store, không có trong Chrome | Vẫn xuất hiện trong danh sách |
| AC06 | Chỉ còn danh bạ đã lưu, lần quét hiện tại không thấy | Tiếp tục xuất hiện theo ý nghĩa danh bạ “đã từng phát hiện”; không tự xóa |
| AC07 | Đăng nhập Antigravity rồi bấm Danh sách tài khoản | Capture bằng cơ chế hiện có, lưu credential nếu hợp lệ và cập nhật trạng thái |
| AC08 | Bấm dồn trong một lượt refresh | Không có nhiều lượt chạy chồng hoặc hàng đợi yêu cầu mới |
| AC09 | Bấm lại nhiều lần sau khi lượt trước hoàn tất | Cập nhật cùng bản ghi, không tạo thêm tài khoản/card trùng |
| AC10 | Credential được thêm hoặc không còn đọc được, email/active/thứ tự vẫn giữ nguyên | Card đang tồn tại đổi đúng nhãn/nút; không cần restart ứng dụng |
| AC11 | Chuyển thủ công/tự động thành công | Active badge, nền/viền và card liên quan cập nhật đúng |
| AC12 | Chưa có phiên để capture | Danh sách vẫn hiển thị; không coi đây là lỗi bắt buộc phải chặn UI |
| AC13 | Đọc credential hoặc ghi credential thất bại | Có phản hồi phù hợp, không báo lưu thành công giả; vẫn hiển thị dữ liệu lấy được |
| AC14 | Ghi danh bạ thất bại | Không xóa credential; kết quả quét hiện tại vẫn tham gia danh sách |
| AC15 | Quota lỗi truy vấn hoặc thiếu dữ liệu | Không bị hiểu thành 0% và không phát tín hiệu exhausted giả |
| AC16 | Card thiếu credential được cập nhật quota hoặc đổi theme | Vẫn giữ N/A đúng quy tắc; không hiện lại số quota cũ |
| AC17 | Chu kỳ 5h không áp dụng | Giữ cơ chế quota hiện có, không nhầm với thiếu credential |
| AC18 | Khởi động bản cập nhật trong repo từ bin hoặc CWD khác | Vẫn dùng kho dữ liệu hiện tại, không sinh kho rỗng dưới bin |
| AC19 | Dùng ANTIGRAVITY_STORE_DIR | Tôn trọng vị trí chọn; danh bạ và dữ liệu phụ có vị trí nhất quán |
| AC20 | Clean build trong repo | Dữ liệu tài khoản, danh bạ và dữ liệu phụ không nằm trong vùng bị xóa |
| AC21 | Portable mới hoặc portable đã có dữ liệu | Chọn đúng vị trí theo VIII.2; không âm thầm bỏ kho cũ |
| AC22 | Vị trí dữ liệu không ghi được | Có lỗi rõ ràng; không âm thầm ghi sang vị trí khác |
| AC23 | Menu lưu phiên đã bị bỏ | Không còn lời gọi GUI/callback dư; capture qua danh sách và trước khi chuyển vẫn còn |
| AC24 | Credential bị thu hồi hoặc biến mất sau khi UI đã cập nhật | Engine vẫn giữ kiểm tra/xử lý lỗi; không báo chuyển thành công chỉ từ trạng thái card |
| AC25 | Card có ba trạng thái trên các theme, ẩn email, resize/maximize | Giữ hình học card, masking, theme và hành vi cửa sổ hiện có |
| AC26 | Chuyển thủ công/tự động sau thay đổi | Giữ engine, cổng khóa, nguồn ứng viên, retry, quota policy và continuation hiện có |

### Phạm vi kiểm chứng khi được triển khai

- Sửa production code trước, sau đó mới cập nhật test phù hợp hành vi mới.
- Test có ý nghĩa cần bao gồm hợp nhất hai kho, lặp refresh, cập nhật card đang tồn tại và tương thích đường dẫn.
- Loại bỏ/cập nhật test riêng của thao tác lưu đã bỏ; không chỉ xóa assertion để suite xanh.
- Dùng fixture cho tài khoản, credential và thư mục dữ liệu; kiểm tra clean/path trên thư mục thử nghiệm, không xóa kho thật.
- Chạy build/test phù hợp cấu hình Windows GUI hiện tại, rồi kiểm tra giao diện và luồng sản phẩm cần thiết khi bước triển khai được giao.
- Ghi riêng kết quả source/unit/runtime. Một lệnh test thành công không tự chứng minh toàn bộ luồng chuyển tài khoản thật.
- Lần cập nhật Markdown này không yêu cầu chạy các kiểm tra sản phẩm trên.

## XII. Những nội dung R1 được thay thế trong R2

| Nội dung R1 | Nội dung có hiệu lực trong R2 |
|---|---|
| Mọi tài khoản trong discovered_accounts là “chưa OAuth” | Danh bạ chứa email phát hiện; một email có thể đồng thời có credential |
| Hai kho như hai bước chuyển tài khoản từ thư mục này sang thư mục khác | Hai loại dữ liệu độc lập, hợp nhất theo email, không cần di chuyển bản ghi |
| Chỉ ghi danh bạ sau quét | Khởi tạo, ghi và đọc danh bạ đều tham gia luồng |
| Thêm nhánh HasToken lúc tạo card là sửa xong | Cập nhật trạng thái card đang tồn tại và các đường quota cũng phải đồng nhất |
| SaveSession là dead code cần xóa | Owner đã quyết định bỏ thao tác lưu riêng; giữ capture trong manager |
| Tự động bắt mọi lần đăng nhập | Capture chạy trong các luồng hiện có; người dùng đăng nhập rồi lấy lại danh sách |
| Luôn đặt kho cạnh exe | Tôn trọng override/kho cũ, phân biệt repo và portable, tránh thư mục clean build |
| Chuyển tài khoản thật dưới 1,5 giây | Giữ engine chờ restart 8 giây và các bước còn lại; không đặt SLA tổng thời gian chưa đo |
| Có token thì quota hiển thị đầy đủ | Hiển thị theo dữ liệu nhận được và chu kỳ áp dụng |
| N/A đạt được bằng đổi nhãn hành động | Cần quy tắc ở cả dựng và cập nhật quota, không đưa N/A thành quota giả |
| accounts_store chỉ được chứa JSON token | Quy tắc token áp dụng cho file tài khoản; giữ cache/preferences/lịch sử |
| Cửa sổ khóa cứng 920 × 630 | Kích thước khởi tạo 920 × 630, giữ resize; card giữ 280 × 130 |
| .gitignore bảo mật 100% | .gitignore tránh đưa dữ liệu vào Git; không mã hóa credential |
| Click đúp để chuyển | Một lần bấm theo hành vi nút hiện có |
| Không có mô tả bấm nút nhiều lần | Cơ chế bận bỏ qua yêu cầu chồng; lượt sau cập nhật cùng dữ liệu, không tạo bản sao |

## XIII. Trạng thái bàn giao tài liệu

- Đã đưa quyết định bỏ chức năng lưu phiên trong Tray vào phương án.
- Đã phân biệt hiện trạng source với hành vi đích và tiêu chí nghiệm thu tương lai.
- Đã bổ sung dữ liệu hai kho, luồng đọc/ghi/hợp nhất, cập nhật card, quota, lỗi capture và bấm refresh nhiều lần.
- Đã sửa đề xuất đường dẫn có nguy cơ đưa dữ liệu vào thư mục bị clean build xóa.
- Không tạo kế hoạch thực thi mới, không thay đổi source hay dữ liệu người dùng trong lần cập nhật này.
- Phương án đang ở trạng thái DRAFT để Owner xem lại. Báo cáo không tự cấp phép triển khai hoặc xác nhận nghiệm thu tính năng.

### Kiểm tra bản cập nhật R2

- **Verdict: PASS trong phạm vi cập nhật tài liệu.** Nội dung đã bao phủ kết quả review, quyết định Tray và câu hỏi về bấm refresh nhiều lần.
- Đã đọc lại toàn bộ bản cập nhật, đối chiếu các dẫn chiếu source, kiểm tra liên kết file/số dòng và định dạng Markdown/UTF-8.
- Diff của lần cập nhật chỉ gồm báo cáo này; không sửa các nhóm source manager, store, UI, telemetry, automation hoặc reloader.
- Không chạy build, test hoặc kiểm chứng runtime. Các tiêu chí AC01–AC26 vẫn là yêu cầu cho giai đoạn triển khai, không phải kết quả đã đạt.
