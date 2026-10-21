# Đặc tả kiến trúc: Trạng thái Bật/Tắt tài khoản và Cơ chế Dừng an toàn khi cạn ứng viên

(Account On/Off Status & Auto-Stop Safety Specification)

- **Ngày lập:** 2026-10-21.
- **Múi giờ:** Asia/Bangkok.
- **Phiên bản:** 1.0 (FINAL SPEC).
- **Mã văn kiện:** `SPEC-ARCH-20261021-ACCOUNT-STATUS-ON-OFF-AUTO-STOP`.
- **Trạng thái:** APPROVED BY OWNER — Đã thống nhất toàn diện với Human Owner; sẵn sàng cho các lane lập kế hoạch và triển khai.
- **Phạm vi tài liệu:** Đặc tả kiến trúc, ranh giới dữ liệu, hợp đồng giao diện và bất biến vận hành cho tính năng vô hiệu hóa tài khoản và cơ chế dừng tự động an toàn. Tuyệt đối không can thiệp mã nguồn sản phẩm trong bước này.
- **Đối tượng áp dụng:** Human Owner, CEO, Planner, Coder, QA, Supervisor.

---

## I. Bối cảnh và Mục tiêu kiến trúc

### 1. Bối cảnh vận hành
Trong quá trình vận hành luân chuyển tài khoản tự động hoặc thủ công của `antigravity_auto_switch`, người dùng có nhu cầu thực tế:
1. Chủ động loại trừ tạm thời một hoặc nhiều tài khoản (ví dụ: tài khoản cá nhân, tài khoản đang bảo trì hoặc tài khoản dự phòng khẩn cấp) khỏi chu trình chuyển đổi mà không cần phải xóa credential.
2. Ngăn ngừa bấm nhầm chuyển đổi thủ công vào tài khoản đã bị vô hiệu hóa.
3. Tiết kiệm tài nguyên mạng và tránh gửi request không cần thiết tới Google API đối với các tài khoản đang bị vô hiệu hóa.
4. Đảm bảo tính nhất quán của cơ chế Tự động (Auto-Switch): Khi cạn kiệt tài khoản chuyển đổi hợp lệ, hệ thống phải dừng lại an toàn và thông báo rõ ràng cho người dùng, thay vì tiếp tục chạy vòng lặp vô tận.

### 2. Mục tiêu kiến trúc
1. **Quản lý trạng thái thực thể độc lập:** Bổ sung trường trạng thái trực tiếp vào tệp lưu trữ của từng tài khoản trong kho credential, bảo đảm tính gắn kết dữ liệu (high cohesion) và tách bạch hoàn toàn với cấu hình giao diện tổng thể.
2. **Trực quan và dễ truy vết:** Trạng thái lưu trữ bằng chuỗi văn bản tường minh, khớp trực tiếp 1:1 với hiển thị nút bấm trên giao diện người dùng.
3. **Tương thích ngược tuyệt đối (Zero Migration):** Các tài khoản hiện hữu chưa có trường trạng thái tự động được nhận diện là trạng thái hoạt động bình thường, không yêu cầu kịch bản chuyển đổi dữ liệu.
4. **Bảo tồn trạng thái qua vòng đời:** Mọi luồng tự động cập nhật token (capture credential, làm mới OAuth) phải bảo toàn trạng thái Bật/Tắt đã được người dùng thiết lập.
5. **Khóa chốt 2 tầng (Defense-in-Depth):** Khóa cả giao diện người dùng (nút chuyển bị làm mờ, chặn click) lẫn tầng điều khiển chuyển đổi (bộ lọc ứng viên tự động loại trừ tài khoản bị tắt).
6. **Dừng tự động an toàn (Fail-Safe Auto-Stop):** Khi không còn tài khoản hợp lệ để chuyển đổi, hệ thống phải tự động ngắt cờ Tự động, lưu bền vững vào cấu hình và phát thông báo chuẩn hóa.

---

## II. Ranh giới sở hữu và Hợp đồng dữ liệu

### 1. Phân định ranh giới lưu trữ

Hệ thống tuân thủ chặt chẽ nguyên tắc phân tách trách nhiệm giữa hai tầng dữ liệu:

| Tầng dữ liệu | Vị trí lưu trữ | Trách nhiệm sở hữu |
|---|---|---|
| **Cấu hình toàn cục ứng dụng** | `logs/ui_preferences.json` | Quản lý các tùy chọn hiển thị và chế độ hoạt động chung của ứng dụng: Theme giao diện, Ẩn tài khoản, Ghim cửa sổ, và Chế độ Tự động (`auto_switch`). |
| **Thực thể tài khoản & Credential** | `accounts_store/<email>.json` | Quản lý thông tin định danh, thời điểm lưu, dữ liệu token OAuth và trạng thái Bật/Tắt (`status`) của chính tài khoản đó. |
| **Danh bạ phát hiện từ Chrome** | `discovered_accounts/<email>.json` | Chỉ lưu email và thời điểm quét phát hiện; tuyệt đối không chứa token, không chứa trạng thái Bật/Tắt. |

### 2. Hợp đồng Schema tệp tài khoản (`accounts_store/<email>.json`)

Tệp tài khoản trong kho credential mở rộng thêm trường trạng thái với cấu trúc như sau:

```json
{
  "email": "developer@gmail.com",
  "status": "on",
  "saved_at": 1792502031.42,
  "token_data": {
    "token": {
      "access_token": "ya29...",
      "token_type": "Bearer",
      "refresh_token": "1//...",
      "expiry": "2026-10-21T21:00:00+07:00"
    },
    "auth_method": "consumer"
  }
}
```

#### Quy tắc trường `status`:
- **Kiểu dữ liệu:** Chuỗi văn bản (`string`).
- **Tập giá trị hợp lệ:**
  - `"on"`: Tài khoản đang ở trạng thái Hoạt động (Enabled).
  - `"off"`: Tài khoản đang ở trạng thái Vô hiệu hóa (Disabled).
- **Quy tắc giá trị mặc định:** Nếu trường `status` vắng mặt trong tệp JSON hoặc mang giá trị rỗng/không xác định, hệ thống bắt buộc suy diễn giá trị mặc định là `"on"`.
- **Định dạng ghi ra đĩa:** Khi ghi tệp JSON, trường `status` phải luôn được ghi rõ ràng dưới dạng `"status": "on"` hoặc `"status": "off"`.

### 3. Bất biến bảo tồn trạng thái (State Preservation Invariant)
Khi hệ thống thực hiện ghi đè hoặc cập nhật tệp tài khoản (ví dụ: tự động capture credential từ hệ điều hành, làm mới token trước khi chuyển đổi):
- Hệ thống bắt buộc phải đọc trước giá trị `status` hiện hữu trên đĩa của tài khoản đó.
- Trường `status` mới ghi xuống tệp phải bảo lưu chính xác giá trị đã có. Tuyệt đối không được gán đè về giá trị mặc định làm mất trạng thái `"off"` mà người dùng đã thiết lập.

---

## III. Đặc tả Giao diện Thẻ tài khoản (Card UI Specification)

### 1. Bố cục hàng trên của Thẻ (Card Top Row Layout)
Kích thước thẻ tài khoản giữ nguyên quy chuẩn cố định $280 \times 130$ pixel. Hàng trên cùng của thẻ được bố trí như sau:

```text
+-------------------------------------------------------------------+
|  [Email (Ellipsis + Tooltip)]       [Cụm Trạng thái/Chuyển] [🔄] [ON] |
+-------------------------------------------------------------------+
```

- **Phía bên trái:** Nhãn Email hỗ trợ thu gọn ba chấm và tooltip khi rê chuột (kèm chế độ ẩn thông tin riêng tư).
- **Phía bên phải:** Cụm điều khiển hành động gồm 3 thành phần theo thứ tự từ trái qua phải:
  1. `[Cụm Trạng thái / Chuyển]`: Nút `[Chuyển]` hoặc nhãn `[✔ Dùng]` hoặc nhãn `[Cần OAuth]`.
  2. `[Nút Làm mới Quota]`: Nút biểu tượng `[🔄]` kích thước nhỏ gọn.
  3. `[Nút Trạng thái Bật/Tắt]`: Nút Text hiển thị `[ON]` hoặc `[OFF]`.

### 2. Quy tắc hiển thị và tương tác của Nút `[ON/OFF]`

| Thuộc tính | Trạng thái Bật (`"status": "on"`) | Trạng thái Tắt (`"status": "off"`) |
|---|---|---|
| **Văn bản hiển thị** | `[ON]` | `[OFF]` |
| **Màu sắc nhãn** | Màu xanh lá thành công (`Success`) | Màu xám chữ mờ (`TextMuted`) |
| **Hành vi khi bấm** | Chuyển trạng thái sang `"off"`, cập nhật tức thì tệp tài khoản trên đĩa và cập nhật giao diện tại chỗ. | Chuyển trạng thái sang `"on"`, cập nhật tức thì tệp tài khoản trên đĩa và cập nhật giao diện tại chỗ. |
| **Tác động tới các thành phần khác trên cùng thẻ** | Nút `[Chuyển]` và nút `[🔄]` hoạt động bình thường theo tình trạng token và active. | - Nút `[Chuyển]` bị làm mờ, không thể bấm.<br>- Nút `[🔄]` bị vô hiệu hóa.<br>- Khung thẻ có thể giảm độ sáng viền để nhận diện trực quan. |

### 3. Ranh giới nghiêm ngặt với tài khoản `[Cần OAuth]`
- Tài khoản mới chỉ được phát hiện từ trình duyệt mà chưa có credential lưu trong kho (`has_token == false` và `is_active == false`) hiển thị nhãn `[Cần OAuth]`.
- Do tài khoản này chưa thể sử dụng trong ứng dụng, **tuyệt đối KHÔNG hiển thị nút `[ON/OFF]` trên thẻ của tài khoản `[Cần OAuth]`**.
- Chỉ các thẻ tài khoản đã có credential (`has_token == true`) hoặc đang là tài khoản active mới hiển thị nút `[ON/OFF]`.

### 4. Hành vi của nút `[Chuyển]` khi tài khoản ở trạng thái `[OFF]`
Khi một tài khoản có credential hợp lệ nhưng đang ở trạng thái `[OFF]`:
- Nút `[Chuyển]` vẫn giữ nguyên vị trí để không làm xáo trộn bố cục layout của thẻ.
- **Hình thức hiển thị:** Màu chữ chuyển sang màu xám mờ (`TextMuted`), nền không đổi màu khi rê chuột (không có hover highlight).
- **Con trỏ chuột:** Giữ nguyên con trỏ mặc định (mũi tên tiêu chuẩn), tuyệt đối không chuyển thành con trỏ bàn tay/chỉ tay (`PointerCursor`).
- **Xử lý sự kiện:** Bấm vào nút hoàn toàn không kích hoạt bất kỳ hành động nào (no-op).

### 5. Hành vi của nút `[🔄]` và Quota khi tài khoản ở trạng thái `[OFF]`
- Nút làm mới đơn lẻ `[🔄]` trên thẻ bị vô hiệu hóa hoàn toàn (`disabled`), người dùng không thể kích hoạt truy vấn quota đơn lẻ cho tài khoản đã tắt.
- Toàn bộ các vòng lặp định kỳ ngầm truy vấn quota (Telemetry) bắt buộc **bỏ qua**, tuyệt đối không gửi request truy vấn quota đối với các tài khoản đang mang trạng thái `"off"`.

---

## IV. Đặc tả Cơ chế Tự động và Dừng an toàn (Automation & Safety Stop)

### 1. Tiêu chí loại trừ ứng viên chuyển đổi tự động
Trong quy trình đánh giá và phân loại ứng viên chuyển đổi tự động, hệ thống bổ sung thêm tiêu chí loại trừ:
- **Tiêu chí loại trừ trạng thái:** Bất kỳ tài khoản nào mang trạng thái `status == "off"` đều bị loại bỏ ngay từ bước lọc ứng viên ban đầu.
- Lý do loại trừ được ghi nhận rõ ràng vào báo cáo loại trừ: `"tài khoản đang ở chế độ vô hiệu hóa (OFF)"`.

### 2. Quy trình xử lý khi người dùng TẮT (`[OFF]`) tài khoản đang Active (`[✔ Dùng]`)
Khi người dùng bấm chuyển nút `[ON]` sang `[OFF]` trên chính thẻ của tài khoản đang hoạt động:
1. Hệ thống cập nhật trạng thái của tài khoản đó thành `"status": "off"` trong tệp lưu trữ tương ứng.
2. Nút trên thẻ active lập tức chuyển sang hiển thị `[OFF]`.
3. **Nếu chế độ Tự động đang BẬT (`auto_switch == true`):**
   - Hệ thống lập tức kích hoạt chu trình tìm kiếm tài khoản thay thế.
   - Quá trình tìm kiếm chỉ xét các tài khoản có credential hợp lệ và mang trạng thái `"status": "on"`.
   - **Trường hợp tìm thấy ứng viên hợp lệ:** Thực hiện chuyển đổi tài khoản theo quy trình chuẩn.
   - **Trường hợp KHÔNG tìm thấy bất kỳ ứng viên hợp lệ nào:** Kích hoạt ngay **Cơ chế Dừng an toàn (Safety Stop)** (xem mục IV.3).
4. **Nếu chế độ Tự động đang TẮT (`auto_switch == false`):**
   - Tài khoản hiện tại tiếp tục duy trì phiên làm việc cho đến khi người dùng chuyển đổi thủ công sang tài khoản khác.
   - Sau khi chuyển đi, tài khoản này sẽ ở trạng thái `[OFF]` và nút `[Chuyển]` bị mờ, không thể chọn lại nếu chưa bật sang `[ON]`.

### 3. Cơ chế Dừng an toàn toàn cục khi cạn ứng viên (No-Candidate Safety Stop)
Khi xảy ra tình huống không có ứng viên nào đủ điều kiện chuyển đổi (bao gồm cả trường hợp tất cả các tài khoản dự phòng đều bị Tắt `[OFF]`, hết hạn mức, hoặc lỗi token):
1. **Dừng bộ điều khiển tự động:** Lập tức ngắt trạng thái hoạt động của bộ điều khiển tự động.
2. **Cập nhật giao diện:** Gạt công tắc "Tự động" trên thanh công cụ giao diện về vị trí `OFF`.
3. **Lưu cấu hình bền vững:** Cập nhật ngay lập tức `"auto_switch": false` vào tệp cấu hình [logs/ui_preferences.json](file:///g:/antigravity_auto_switch/logs/ui_preferences.json).
4. **Thông báo chuẩn hóa cho người dùng:** Hiển thị thông báo cảnh báo rõ ràng trên thanh trạng thái phía dưới giao diện với nội dung chuẩn hóa:
   ```text
   ⚠️ Không có tài khoản để chuyển.
   ```
   Thông báo tự động ẩn sau khoảng thời gian tiêu chuẩn để giữ giao diện thanh thoát.

---

## V. Ma trận trạng thái và Bất biến hệ thống

### 1. Ma trận hiển thị và tương tác thẻ tài khoản

| Tình trạng tài khoản | Trạng thái Token | Trạng thái `status` | Hiển thị Hành động | Nút `[🔄]` | Nút `[ON/OFF]` | Khả năng chuyển đổi |
|---|---|---|---|---|---|---|
| **Active** | Hợp lệ | `"on"` | `[✔ Dùng]` | Hoạt động | `[ON]` (Xanh) | Đang sử dụng |
| **Active** | Hợp lệ | `"off"` | `[✔ Dùng]` | Vô hiệu | `[OFF]` (Xám) | Tìm ứng viên mới ngay nếu Tự động đang bật |
| **Inactive** | Hợp lệ | `"on"` | `[Chuyển]` | Hoạt động | `[ON]` (Xanh) | Bấm để chuyển thủ công; được xét tự động |
| **Inactive** | Hợp lệ | `"off"` | `[Chuyển]` *(Mờ)* | Vô hiệu | `[OFF]` (Xám) | **Khóa hoàn toàn** (cả thủ công lẫn tự động) |
| **Inactive** | Chưa có | Không áp dụng | `[Cần OAuth]` | Vô hiệu | **Ẩn** | Không có nút chuyển |

### 2. Các bất biến hệ thống bắt buộc (System Invariants)

1. **`Invariant-Account-Status-Persistence`:** Trạng thái Bật/Tắt của tài khoản phải được lưu bền vững vào chính tệp `accounts_store/<email>.json` dưới dạng trường `"status": "on"` hoặc `"status": "off"`. Khởi động lại ứng dụng không làm mất trạng thái này.
2. **`Invariant-Backward-Compatibility-Default-On`:** Mọi tệp tài khoản hiện hữu chưa có trường `status` phải luôn được nạp với giá trị mặc định là `"on"`.
3. **`Invariant-Token-Update-Preservation`:** Các tác vụ tự động ghi tệp tài khoản do cập nhật token hoặc capture credential tuyệt đối không được ghi đè làm mất giá trị `status` đang có.
4. **`Invariant-Unauth-Card-Isolation`:** Thẻ tài khoản chưa có credential (`[Cần OAuth]`) tuyệt đối không hiển thị nút `[ON/OFF]`.
5. **`Invariant-Disabled-Switch-Lockout`:** Tài khoản mang trạng thái `status == "off"` bị ngăn chặn chuyển đổi 100% ở cả tầng UI (nút Chuyển mờ, click vô hiệu) lẫn tầng Controller (bị loại khỏi danh sách ứng viên tự động).
6. **`Invariant-No-Candidate-Auto-Halt`:** Khi phát hiện không có ứng viên nào đủ điều kiện chuyển đổi, hệ thống bắt buộc phải tự động tắt cờ Tự động, đồng bộ hóa lưu `"auto_switch": false` vào `logs/ui_preferences.json` và hiển thị thông báo `⚠️ Không có tài khoản để chuyển.`.
7. **`Invariant-Disabled-Telemetry-Silence`:** Không thực hiện quét quota định kỳ hoặc gửi request mạng cho bất kỳ tài khoản nào đang ở trạng thái `status == "off"`.

---

## VI. Kịch bản Kiểm thử và Nghiệm thu (Acceptance Criteria)

### 1. Kiểm thử lưu trữ bền vững
- [ ] Mở ứng dụng lần đầu với các tệp tài khoản cũ: Toàn bộ hiển thị `[ON]`, không có lỗi nạp tệp.
- [ ] Bấm nút `[ON]` trên thẻ tài khoản A: Nút chuyển sang `[OFF]`, tệp `accounts_store/<email_A>.json` chứa `"status": "off"`.
- [ ] Khởi động lại ứng dụng: Tài khoản A vẫn hiển thị `[OFF]`, nút `[Chuyển]` của A vẫn bị mờ.
- [ ] Bấm nút `[OFF]` trên thẻ tài khoản A: Nút chuyển sang `[ON]`, tệp `accounts_store/<email_A>.json` chứa `"status": "on"`.
- [ ] Giả lập capture credential mới cho tài khoản A khi A đang `"off"`: Tệp ghi lại access token mới nhưng trường `"status": "off"` vẫn được giữ nguyên.

### 2. Kiểm thử tương tác giao diện Thẻ
- [ ] Trên thẻ có trạng thái `[OFF]`: Nút `[Chuyển]` có màu xám mờ (`TextMuted`), rê chuột không đổi màu, click không kích hoạt chuyển tài khoản.
- [ ] Trên thẻ có trạng thái `[OFF]`: Nút `[🔄]` bị vô hiệu hóa (`disabled`), không thể bấm làm mới.
- [ ] Thẻ tài khoản `[Cần OAuth]`: Hoàn toàn không có nút `[ON/OFF]`.

### 3. Kiểm thử cơ chế Tự động và Dừng an toàn
- [ ] Khi bật Tự động (`auto_switch == true`), giả lập tài khoản active cạn quota: Hệ thống chỉ chọn chuyển sang các tài khoản có credential và mang trạng thái `[ON]`; bỏ qua mọi tài khoản mang `[OFF]`.
- [ ] Giả lập người dùng bấm `[OFF]` trên chính tài khoản đang Active khi Tự động đang Bật:
  - Nếu còn tài khoản `[ON]` khác khả dụng: Hệ thống tự động chuyển ngay sang tài khoản đó.
  - Nếu tất cả các tài khoản còn lại đều mang `[OFF]` (hoặc không đủ quota): Hệ thống tự động gạt nút "Tự động" về `OFF`, lưu `"auto_switch": false` vào `logs/ui_preferences.json`, và hiển thị thông báo `⚠️ Không có tài khoản để chuyển.`.
- [ ] Giả lập người dùng bấm `[OFF]` trên tài khoản đang Active khi Tự động đang Tắt: Thẻ chuyển sang `[OFF]`, phiên làm việc hiện tại của tài khoản active không bị gián đoạn.
