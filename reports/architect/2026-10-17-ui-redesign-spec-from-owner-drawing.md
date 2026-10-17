# Báo Cáo Đặc Tả Kiến Trúc: Tái Cấu Trúc Toàn Diện Giao Diện Fyne GUI Theo Bản Vẽ Của Human Owner

**Tài liệu tham chiếu:** `reports/architect/2026-10-17-ui-redesign-spec-from-owner-drawing.md`  
**Ngày thực hiện:** 2026-10-17  
**Làn đảm nhiệm:** Senior System Architect (Kiến Trúc Sư Hệ Thống)  
**Cơ quan chỉ đạo:** CEO Agent (Thread ID: `01a14378-4a08-77c1-88d5-666ad8c3592d`)  
**Thẩm quyền & Căn cứ tối cao:** Chỉ thị trực tiếp từ **Human Owner**, bản vẽ phác thảo tay tại `reports/problem/images/9180d29a.jpg` (ảnh xoay chuẩn tại `reports/problem/images/drawing_upright.jpg`), và bằng chứng thực nghiệm đối soát trực tiếp từ Google Quota API runtime (`daily-cloudcode-pa.googleapis.com`) cùng Antigravity Language Server Connect-RPC (`RetrieveUserQuotaSummary`).  
**Trạng thái tài liệu:** SPECIFICATION APPROVED BY ARCHITECT — READY FOR SUPERVISOR REVIEW & CODER IMPLEMENTATION

---

## 1. Tóm Tắt Điều Hành & Bối Cảnh Tái Thiết Kế (Executive Summary)

### 1.1 Yêu Cầu Trực Tiếp Từ Human Owner
Human Owner đã cung cấp một bản vẽ phác thảo tay chi tiết tại `reports/problem/images/9180d29a.jpg`, chỉ đạo tái cấu trúc toàn diện giao diện Fyne GUI của ứng dụng **Antigravity Multi-Account Switcher**.

Các chỉ thị cốt lõi của Owner bao gồm:
1. **Tuyệt đối trung thực với bản vẽ:** Những thành phần nào **KHÔNG CÓ** trong bản vẽ của Owner thì phải **GỠ BỎ TRIỆT ĐỂ**. Những thành phần nào Owner **BỔ SUNG** trong bản vẽ thì phải **THÊM VÀO ĐẦY ĐỦ**.
2. **Quy chuẩn hiển thị Quota mới:**
   - Bắt buộc hiển thị theo **2 chu kỳ thời gian**: **5 giờ (5h)** và **Hàng tuần (Tuần / week)**.
   - **CHỈ CẦN hiển thị hạn ngạch của Claude và Gemini; GỠ BỎ HOÀN TOÀN GPT**.
3. **Cấu trúc lưới đa cột:** Danh sách tài khoản chuyển đổi từ danh sách dọc đơn lẻ (1 cột) sang **Lưới 3 cột (3-Column Grid)** trực quan, gọn gàng, thể hiện tối đa thông tin mà không chiếm dụng không gian màn hình.
4. **Tối giản hóa thanh công cụ và thẻ tài khoản:** Loại bỏ các nút bấm rườm rà (Lưu phiên, Lưu thẻ, Chuyển thẻ riêng biệt, Làm mới đơn lẻ trên thẻ, thanh chú thích Legend, cảnh báo đếm ngược dài dòng, thanh trạng thái status bar dưới đáy).

---

## 2. Phân Tích Chi Tiết Từng Khối Giao Diện Trong Bản Vẽ Của Owner

Căn cứ vào ảnh chụp phác thảo gốc tại `reports/problem/images/9180d29a.jpg` (được xoay đứng 90 độ tại `reports/problem/images/drawing_upright.jpg`), cấu trúc giao diện mới được chia thành 4 phân vùng chính từ trên xuống dưới:

```text
+----------------------------------------------------------------------------------------------------+
| [AS] Antigravity Multi-Account Switcher                                                  [ - ][口][ X ] |
| Language Server : PID 7768 | Port 52704 | Hoạt động                            Theme [ Sáng      ▼ ] |
| [ 🔄 Làm mới Chrome ]   [ 🔄 Làm mới Quota ]   [ 📌 Ghim : Bật ]                       Cập nhật : 19:24:37 |
+----------------------------------------------------------------------------------------------------+
| ------------------------------------- (Đường phân cách mảnh) -------------------------------------- |
| +-------------------------+ +-------------------------+ +-------------------------+                |
| | email@...com [✔Đang dùng]| | Tương tự                | | Tương tự                |                |
| | 5h: Claude 100% Gemini88%| |                         | |                         |                |
| | week:Gemini 90%Claude100%| |                         | |                         |                |
| +-------------------------+ +-------------------------+ +-------------------------+                |
| +-------------------------+ +-------------------------+ +-------------------------+                |
| | Tương tự                | | Tương tự                | | Tương tự                |                |
| +-------------------------+ +-------------------------+ +-------------------------+                |
| +-------------------------+ +-------------------------+ +-------------------------+                |
| | Tương tự                | | Tương tự                | | Tương tự                |                |
| +-------------------------+ +-------------------------+ +-------------------------+                |
+----------------------------------------------------------------------------------------------------+
```

### 2.1 Khối 1: Thanh Tiêu Đề Cửa Sổ (Window Header & Controls)
- **Vị trí:** Hàng 1, sát mép trên cửa sổ.
- **Bên trái:**
  - Biểu tượng Monogram/Logo `[AS]` (App Logo).
  - Tên ứng dụng: `Antigravity Multi-Account Switcher` (chữ in đậm).
- **Bên phải:**
  - Cụm điều khiển cửa sổ tiêu chuẩn: Thu nhỏ `[ - ]`, Phóng to `[ 口 ]`, Đóng `[ X ]`. Trong Fyne GUI, cụm này do khung cửa sổ Window Manager của OS hoặc Fyne Window Header điều khiển.

### 2.2 Khối 2: Trạng Thái Language Server & Bộ Chọn Giao Diện (Sub-Header Row)
- **Vị trí:** Hàng 2, ngay dưới tiêu đề.
- **Bên trái:**
  - Nhãn trạng thái Language Server theo cú pháp chính xác:
    ```text
    Language Server : PID <PID> | Port <PORT> | Hoạt động
    ```
    *(Ví dụ thực tế trên bản vẽ Owner: `Language Server : PID 7768 | Port 52704 | Hoạt động`)*.
    - Nếu Language Server đang chạy và kết nối tốt: Hiển thị PID, Port, và chữ `Hoạt động` (màu xanh lá/chữ rõ).
    - Nếu Language Server chưa bật: Hiển thị `Language Server : Không phát hiện (Chế độ dự phòng WinCred)`.
- **Bên phải:**
  - Nhãn `Theme` kèm dropdown select `[ Sáng ▼ ]`.
  - Cho phép người dùng chuyển đổi nhanh giữa 3 chủ đề giao diện:
    - `🌙 Tối` (Dark Theme)
    - `☀️ Sáng` (Light Theme)
    - `☕ Ấm` (Warm Theme)

### 2.3 Khối 3: Thanh Nút Tác Vụ & Thời Gian Cập Nhật (Action Toolbar & Timestamp Row)
- **Vị trí:** Hàng 3, đối xứng hai bên.
- **Bên trái (Cụm 3 nút bấm tác vụ):**
  1. `[ 🔄 Làm mới Chrome ]`: Quét và đồng bộ danh sách tài khoản từ các Profile Chrome.
  2. `[ 🔄 Làm mới Quota ]`: Quét song song và cập nhật hạn ngạch quota cho toàn bộ tài khoản (thay thế nút cồng kềnh `[📊 Làm mới Quota Tất cả]`).
  3. `[ 📌 Ghim : Bật ]` / `[ 📌 Ghim : Tắt ]`: Nút ghim cửa sổ luôn nổi lên trên (Always-on-Top toggle). Khi đang bật hiển thị `Ghim : Bật` (High Importance/màu sáng), khi tắt hiển thị `Ghim : Tắt` (Medium Importance).
- **Bên phải (Thời gian cập nhật):**
  - Nhãn text hiển thị thời điểm đồng bộ gần nhất:
    ```text
    Cập nhật : 19:24:37
    ```
    *(Được cập nhật tự động mỗi khi vòng lặp 60s quét xong hoặc người dùng nhấn Làm mới Quota)*.

### 2.4 Khối 4: Vạch Ngăn Cách (Horizontal Separator)
- Một đường kẻ ngang thanh mảnh (1px border/separator) tách biệt hoàn toàn cụm header điều khiển với khu vực hiển thị danh sách tài khoản.

### 2.5 Khối 5: Lưới Thẻ Tài Khoản 3 Cột (3-Column Account Card Grid)
- **Vị trí:** Thân chính của cửa sổ, đặt bên trong container cuộn dọc (`container.NewVScroll`).
- **Bố cục lưới:** Chia thành **3 cột đều nhau** (`container.NewGridWithColumns(3, ...)`). Trong bản vẽ, Owner vẽ minh họa 9 thẻ tài khoản (3 hàng x 3 cột).
- **Cấu trúc chi tiết của từng Thẻ Tài Khoản (quy chuẩn theo Thẻ 1):**
  - **Dòng 1 (Card Header):**
    - Bên trái: Địa chỉ Email (ví dụ: `email@...com`, hoặc địa chỉ thực tế `daohien204ccyersin@gmail.com`). Chữ đậm rõ nét.
    - Bên phải: Huy hiệu trạng thái tài khoản đang kích hoạt: `[ ✔ Đang dùng ]` (Badge nền sáng/chữ xanh nổi bật). Nếu không phải tài khoản active thì để trống khu vực này.
  - **Dòng 2 (Hạn ngạch chu kỳ 5 giờ):**
    ```text
    5h : Claude <pct>%   Gemini <pct>%
    ```
    *(Ví dụ trong bản vẽ của Owner: `5h : Claude 100%  Gemini 88%`)*.
  - **Dòng 3 (Hạn ngạch chu kỳ Tuần):**
    ```text
    week : Claude <pct>%   Gemini <pct>%
    ```
    *(Ví dụ trong bản vẽ của Owner: `week : Gemini 90%  Claude 100%` — trên UI thống nhất chuẩn hóa hiển thị Claude trước, Gemini sau)*.
  - **Màu sắc theo ngưỡng hạn ngạch:**
    - Xanh lá (`#22c55e`): Quota $> 50%$.
    - Vàng hổ phách (`#f59e0b`): Quota từ $15%$ đến $50%$.
    - Đỏ cảnh báo (`#ef4444`): Quota $< 15%$.
    - Xám nhạt (`#888899`): Chưa có dữ liệu / Lỗi kết nối `[!]`.
  - **Cơ chế chuyển đổi tài khoản (Touch & Click Interaction):**
    - Trên thẻ **KHÔNG CÒN** nút bấm riêng lẻ `[⚡ Chuyển]` hay `[💾 Lưu]`.
    - Người dùng **click trực tiếp vào bất kỳ điểm nào trên Thẻ Tài Khoản** để thực hiện hot-swap tức thời sang tài khoản đó.
    - Thẻ đang dùng (`IsActive`) có viền sáng nổi bật (Active Border) và không kích hoạt hot-swap lại chính nó khi click.

---

## 3. Ma Trận Đối Soát: Thành Phần Gỡ Bỏ Triệt Để & Thành Phần Thêm Mới

Tuân thủ nguyên tắc sắt đá: *"Những phần nào KHÔNG CÓ trong bản vẽ của Owner thì GỠ BỎ TRIỆT ĐỂ. Những phần nào Owner bổ sung trong bản vẽ thì THÊM VÀO ĐẦY ĐỦ"*.

### 3.1 Bảng Danh Sách Các Thành Phần Phải GỠ BỎ TRIỆT ĐỂ

| STT | Thành phần cũ cần gỡ bỏ | Vị trí cũ trong mã nguồn | Lý do gỡ bỏ |
|---|---|---|---|
| 1 | **Toàn bộ hiển thị Quota GPT** | `quota_badge.go`, `widgets.go`, `window.go` | **Chỉ thị trực tiếp của Owner**: Bỏ hoàn toàn GPT, chỉ giữ Claude & Gemini. |
| 2 | **Khối Active Header Card cồng kềnh** | `window.go` (`a.headerCard`, `a.activeStatusLabel`, `a.activeQuotaLabel`) | Không có trong bản vẽ. Thông tin tài khoản đang dùng đã được tích hợp ngay trong thẻ tài khoản (`[ ✔ Đang dùng ]`). |
| 3 | **Nút `[💾 Lưu phiên]` trên toolbar** | `window.go` (`a.saveBtn`) | Không có trong bản vẽ. |
| 4 | **Nút `[💾 Lưu]` và badge `[Chưa lưu]` trên từng thẻ** | `widgets.go` (`saveBtn`, `unauthBadge`) | Không có trong bản vẽ. Việc lưu trữ diễn ra tự động ngầm. |
| 5 | **Nút làm mới đơn lẻ `[🔄]` trên từng thẻ** | `widgets.go` (`refreshBtn`) | Không có trong bản vẽ. Người dùng sử dụng nút `[ Làm mới Quota ]` chung ở toolbar. |
| 6 | **Nút bấm riêng lẻ `[⚡ Chuyển]` trên thẻ** | `widgets.go` (`switchBtn`) | Không có trong bản vẽ. Thay thế bằng cơ chế click trực tiếp toàn thẻ (Card Click-to-Switch). |
| 7 | **Thanh chú thích màu (Legend Card)** | `window.go` (`legendCard`: 🟢 Sẵn sàng \| 🟡 Chưa có JSON \| 🔴 <15%) | Không có trong bản vẽ của Owner. |
| 8 | **Thanh trạng thái đáy cửa sổ (Footer Status Bar)** | `window.go` (`statusBarLabel`, `bottomSection`: "Sẵn sàng", "Đồng bộ lúc...") | Không có trong bản vẽ của Owner. Nhãn thời gian cập nhật đã được đưa lên góc phải hàng 3 (`Cập nhật : 19:24:37`). |
| 9 | **Dòng cảnh báo đếm ngược hồi phục quota** | `quota_badge.go` (`NewQuotaCountdownLabel`: "⚠️ ... sắp cạn lúc HH:mm UTC") | Không có trong bản vẽ của Owner. Làm thẻ bị phình to, phá vỡ bố cục lưới gọn gàng. |
| 10 | **Bố cục cuộn dọc 1 cột (Vertical VBox)** | `window.go` (`a.accountListContainer = container.NewVBox()`) | Thay thế triệt để bằng Grid 3 cột. |

### 3.2 Bảng Danh Sách Các Thành Phần THÊM MỚI ĐẦY ĐỦ

| STT | Thành phần mới cần thêm | Vị trí thiết kế | Quy chuẩn kiến trúc & Hành vi |
|---|---|---|---|
| 1 | **Hàng Sub-Header hợp nhất** | Dòng 2 dưới Header | Bên trái: `Language Server : PID <PID> \| Port <PORT> \| Hoạt động`. Bên phải: `Theme [ Sáng ▼ ]`. |
| 2 | **Thanh Toolbar tinh gọn 3 nút + Timestamp** | Dòng 3 | Trái: `[ 🔄 Làm mới Chrome ]` `[ 🔄 Làm mới Quota ]` `[ 📌 Ghim : Bật ]`. Phải: `Cập nhật : <HH:MM:SS>`. |
| 3 | **Lưới tài khoản 3 cột (3-Column Grid)** | Khung cuộn nội dung chính | `container.NewGridWithColumns(3, ...)` tự động dàn đều các thẻ tài khoản sang 3 cột. |
| 4 | **Hiển thị Quota 2 chu kỳ (Dual-Cycle Quota)** | Dòng 2 & 3 của mỗi thẻ | • Dòng 2: `5h : Claude <pct>%   Gemini <pct>%`<br>• Dòng 3: `week : Claude <pct>%   Gemini <pct>%` (hoặc `Tuần : ...`). |
| 5 | **Huy hiệu active tinh tế `[ ✔ Đang dùng ]`** | Góc trên phải của thẻ tài khoản | Hiển thị duy nhất trên thẻ của tài khoản đang active trong Antigravity. |
| 6 | **Tương tác Click-To-Switch trên toàn bộ thẻ** | Toàn bộ diện tích thẻ tài khoản | Bọc thẻ bằng custom canvas tap hoặc Clickable Container, click là kích hoạt chuyển đổi tức thì. |

---

## 4. Khảo Sát Kỹ Thuật: Cơ Chế Bóc Tách Hạn Ngạch Dual-Cycle (5h & Tuần) Từ Google API

### 4.1 Bằng Chứng Thực Nghiệm Trực Tiếp Từ Runtime Google API

Trong quá trình điều tra kiến trúc, Kiến trúc sư đã trực tiếp gửi HTTP request kèm access token tươi tới các endpoint của Google và Antigravity Language Server. Kết quả đối soát nhị phân thu được như sau:

#### A. Endpoint Google Cloud Code REST API (`v1internal:retrieveUserQuotaSummary`)
Gửi POST tới `https://daily-cloudcode-pa.googleapis.com/v1internal:retrieveUserQuotaSummary` với header:
- `Authorization: Bearer <access_token>`
- `User-Agent: antigravity/2.24.0`
- `X-Goog-Api-Client: gl-go/1.22.0 gdcl/20261017`

**Payload JSON thực tế trả về:**
```json
{
  "groups": [
    {
      "displayName": "Gemini Models",
      "description": "Models within this group: Gemini Flash, Gemini Pro",
      "buckets": [
        {
          "bucketId": "gemini-weekly",
          "displayName": "Weekly Limit Remaining",
          "window": "weekly",
          "resetTime": "2026-10-24T06:29:41Z",
          "remainingFraction": 0.6574791
        },
        {
          "bucketId": "gemini-5h",
          "displayName": "Five Hour Limit Remaining",
          "window": "5h",
          "resetTime": "2026-10-17T16:29:41Z",
          "remainingFraction": 0.67917466
        }
      ]
    },
    {
      "displayName": "Claude and GPT models",
      "description": "Models within this group: Claude Opus, Claude Sonnet, GPT-OSS",
      "buckets": [
        {
          "bucketId": "3p-weekly",
          "displayName": "Weekly Limit Remaining",
          "window": "weekly",
          "resetTime": "2026-10-24T12:45:17Z",
          "remainingFraction": 1.0
        },
        {
          "bucketId": "3p-5h",
          "displayName": "Five Hour Limit Remaining",
          "window": "5h",
          "resetTime": "2026-10-17T17:45:17Z",
          "remainingFraction": 1.0
        }
      ]
    }
  ],
  "description": "Within each group, models share a weekly limit and a 5-hour limit. Quota is consumed proportionally to the cost of the tokens..."
}
```

#### B. Endpoint Local Connect-RPC (`RetrieveUserQuotaSummary`)
Gửi POST tới `http://127.0.0.1:<PORT>/exa.language_server_pb.LanguageServerService/RetrieveUserQuotaSummary` với CSRF token:
**Payload JSON thực tế trả về:** Cùng cấu trúc dữ liệu chính xác, được bọc trong trường gốc `{"response": { "groups": [...] }}`.

### 4.2 Nguyên Lý Bóc Tách Bucket Của Google
Google tổ chức hạn ngạch trí tuệ nhân tạo thành 2 nhóm lớn (`groups`):
1. **Nhóm 1: `Gemini Models`** (Chứa Gemini Flash, Gemini Pro).
   - Bucket chu kỳ 5 giờ: `bucketId == "gemini-5h"` hoặc `window == "5h"`.
   - Bucket chu kỳ tuần: `bucketId == "gemini-weekly"` hoặc `window == "weekly"`.
2. **Nhóm 2: `Claude and GPT models` (Third-Party / 3P)** (Chứa Claude Opus, Claude Sonnet, GPT-OSS).
   - Trong hạ tầng của Google, toàn bộ hạn ngạch Claude được tính chung trong nhóm `3p`.
   - Bucket chu kỳ 5 giờ: `bucketId == "3p-5h"` hoặc `window == "5h"`.
   - Bucket chu kỳ tuần: `bucketId == "3p-weekly"` hoặc `window == "weekly"`.
3. **Quy tắc gỡ bỏ GPT:**
   - Do Claude và GPT dùng chung hạn ngạch nhóm `3p`, việc hiển thị Claude chính là phản ánh chính xác 100% hạn ngạch của nhóm `3p`.
   - Giao diện và cấu trúc dữ liệu **loại bỏ hoàn toàn trường GPT**, chỉ hiển thị nhãn `Claude` cho nhóm này.

### 4.3 Định Nghĩa Cấu Trúc Dữ Liệu Quota Mới (`ModelDualQuota` & `AccountQuotaData`)
Trong `internal/telemetry/client.go`, cấu trúc dữ liệu được tinh chỉnh như sau:

```go
// ModelDualQuota lưu trữ đầy đủ hạn ngạch 2 chu kỳ: 5 giờ và Hàng tuần cho một dòng model.
type ModelDualQuota struct {
    FiveHourFraction float64   `json:"five_hour_fraction"` // Tỷ lệ còn lại chu kỳ 5h (0.0 đến 1.0)
    WeeklyFraction   float64   `json:"weekly_fraction"`    // Tỷ lệ còn lại chu kỳ Tuần (0.0 đến 1.0)
    FiveHourReset    time.Time `json:"five_hour_reset"`    // Thời điểm reset chu kỳ 5h (UTC)
    WeeklyReset      time.Time `json:"weekly_reset"`       // Thời điểm reset chu kỳ Tuần (UTC)
}

// AccountQuotaData lưu trữ hạn ngạch Claude và Gemini theo thiết kế mới của Owner (GPT bị xóa bỏ hoàn toàn).
type AccountQuotaData struct {
    Email        string         `json:"email"`
    Claude       ModelDualQuota `json:"claude"`
    Gemini       ModelDualQuota `json:"gemini"`
    ResetTime    time.Time      `json:"reset_time"`    // Mốc reset sớm nhất để tính toán trạng thái
    Status       string         `json:"status"`        // Pristine, Plentiful, Low, Depleted, Error
    ErrorMessage string         `json:"error_message,omitempty"`
    LastUpdated  time.Time      `json:"last_updated"`
}
```

### 4.4 Thuật Toán Bóc Tách (Parser Algorithm)
1. **Duyệt qua danh sách `groups`:**
   - Nếu `strings.Contains(lowerGroupName, "gemini")`:
     - Duyệt qua các `buckets` bên trong:
       - Nếu `b.Window == "5h"` hoặc `strings.HasSuffix(b.BucketID, "-5h")`: gán cho `Gemini.FiveHourFraction` và `Gemini.FiveHourReset`.
       - Nếu `b.Window == "weekly"` hoặc `strings.HasSuffix(b.BucketID, "-weekly")`: gán cho `Gemini.WeeklyFraction` và `Gemini.WeeklyReset`.
   - Nếu `strings.Contains(lowerGroupName, "claude")` hoặc `strings.Contains(lowerGroupName, "3p")`:
     - Duyệt qua các `buckets` bên trong:
       - Nếu `b.Window == "5h"` hoặc `strings.HasSuffix(b.BucketID, "-5h")`: gán cho `Claude.FiveHourFraction` và `Claude.FiveHourReset`.
       - Nếu `b.Window == "weekly"` hoặc `strings.HasSuffix(b.BucketID, "-weekly")`: gán cho `Claude.WeeklyFraction` và `Claude.WeeklyReset`.
2. **Cơ chế Fallback (nếu gọi REST endpoint cũ chỉ có 1 chu kỳ):**
   - Nếu endpoint chỉ trả về chu kỳ 5h: gán `FiveHourFraction` bằng giá trị nhận được, `WeeklyFraction` mặc định 1.0 (hoặc đồng bộ bằng `FiveHourFraction` nếu không có dữ liệu tuần).

---

## 5. Cấu Trúc Cây Widget Fyne & Bản Đồ Tương Tác (Layout Tree & Touch-Map)

### 5.1 Cây Bố Cục Giao Diện (Fyne Widget Hierarchy Tree)

```text
Window (fyne.Window) - Default Size: 780x520 (Mở rộng chiều ngang cho Lưới 3 Cột)
│
└── MainContainer (container.NewBorder)
    ├── Top: TopSection (container.NewVBox)
    │   ├── Row 1: HeaderRow (container.NewBorder)
    │   │   ├── Left: container.NewHBox(Logo [AS], Title "Antigravity Multi-Account Switcher")
    │   │   └── Right: WindowControls (nếu dùng custom title) hoặc nil
    │   │
    │   ├── Row 2: SubHeaderRow (container.NewBorder)
    │   │   ├── Left: serverStatusLabel ("Language Server : PID ... | Port ... | Hoạt động")
    │   │   └── Right: container.NewHBox(themeLabel "Theme", a.themeSelect [Sáng ▼])
    │   │
    │   ├── Row 3: ToolbarRow (container.NewBorder)
    │   │   ├── Left: container.NewHBox(a.refreshChromeBtn, a.refreshQuotaBtn, a.alwaysOnTopBtn)
    │   │   └── Right: a.lastUpdatedLabel ("Cập nhật : 19:24:37")
    │   │
    │   └── Separator: canvas.NewLine(BorderSubtleColor)
    │
    ├── Bottom: nil (ĐÃ GỠ BỎ TOÀN BỘ FOOTER STATUS BAR)
    │
    └── Center: container.NewVScroll (Cuộn dọc cho toàn bộ danh sách)
        └── a.accountGridContainer (container.NewGridWithColumns(3, cards...))
            ├── AccountCard_1 (Stack: CardBackground + Padded VBox)
            │   ├── Line 1: container.NewBorder(nil, nil, EmailLabel, ActiveBadge "[✔ Đang dùng]")
            │   ├── Line 2: container.NewHBox("5h:", Claude5hBadge, Gemini5hBadge)
            │   └── Line 3: container.NewHBox("week:", ClaudeWeekBadge, GeminiWeekBadge)
            ├── AccountCard_2 (Tương tự)
            ├── AccountCard_3 (Tương tự)
            └── ... (Tự động xuống dòng theo cột lưới)
```

### 5.2 Kích Thước & Tỷ Lệ Cửa Sổ (Window Dimensions)
- **Kích thước khuyến nghị:** `780 x 520 px` (thay cho kích thước hẹp cũ `440 x 560 px`).
  - Chiều rộng `780 px` cho phép chia đều 3 cột, mỗi thẻ tài khoản có bề rộng xấp xỉ `240 px` — đủ không gian rộng rãi để hiển thị địa chỉ email và 2 cặp % quota mà không bị tràn chữ (text truncation).
  - Chiều cao `520 px` hiển thị trọn vẹn 2 hàng thẻ (6 tài khoản) ngay lập tức mà không cần cuộn chuột; khi có 7 đến 9 tài khoản thì thanh cuộn mượt mà hỗ trợ xem các thẻ tiếp theo.

### 5.3 Bản Đồ Tương Tác Cảm Ứng & Sự Kiện (Touch-Map & Interaction Map)

| Khu vực tương tác | Widget Fyne | Sự kiện (Event) | Hành vi kích hoạt (Behavior) |
|---|---|---|---|
| **Bộ chọn Theme** | `widget.Select` (`a.themeSelect`) | `OnChanged` | Đổi ngay theme toàn ứng dụng (`ThemeDark`, `ThemeLight`, `ThemeWarm`), cập nhật bảng màu của toàn bộ canvas mà không đóng app. |
| **Nút Làm mới Chrome** | `widget.Button` (`a.refreshChromeBtn`) | `OnTapped` | Gọi `App.Refresh()`: quét Chrome User Data profiles, phát hiện tài khoản mới. |
| **Nút Làm mới Quota** | `widget.Button` (`a.refreshQuotaBtn`) | `OnTapped` | Gọi `App.RefreshAllQuota()`: quét song song hạn ngạch toàn bộ tài khoản, cập nhật nhãn `Cập nhật : HH:MM:SS`. |
| **Nút Ghim Bật/Tắt** | `widget.Button` (`a.alwaysOnTopBtn`) | `OnTapped` | Gọi `App.ToggleAlwaysOnTop()`: đảo trạng thái ghim OS, đổi nhãn giữa `📌 Ghim : Bật` và `📌 Ghim : Tắt`. |
| **Thẻ Tài Khoản (Card)** | `ClickableCard` / Custom Button | `OnTapped` | • Nếu tài khoản **đang dùng** (`IsActive == true`): Bỏ qua không làm gì.<br>• Nếu tài khoản **sẵn sàng** (`IsActive == false`): Kích hoạt `App.SwitchAccount(email)` để hot-swap mượt mà, sau đó cập nhật huy hiệu `[✔ Đang dùng]` sang thẻ mới. |

---

## 6. Bảng Mã Màu & Quy Chuẩn Thẩm Mỹ Đa Chủ Đề (Themes Palette)

Ứng dụng hỗ trợ 3 chủ đề: **Tối (Dark)**, **Sáng (Light)**, và **Ấm (Warm)**. Bảng màu cho từng thành phần giao diện mới được chuẩn hóa như sau:

| Thành phần | Chủ đề Tối (Dark) | Chủ đề Sáng (Light) | Chủ đề Ấm (Warm) |
|---|:---:|:---:|:---:|
| **Nền Cửa Sổ (Background)** | `#121216` | `#f8f9fa` | `#fbf7ee` |
| **Nền Thẻ Thường (Card Bg)** | `#1c1d22` | `#ffffff` | `#f4ede0` |
| **Viền Thẻ Thường (Card Border)** | `#2c2d35` (1px) | `#e2e8f0` (1px) | `#e5dac8` (1px) |
| **Nền Thẻ Đang Dùng (Active Card Bg)**| `#1a2634` | `#eef6ff` | `#ede6d4` |
| **Viền Thẻ Đang Dùng (Active Border)** | `#3b82f6` (1.5px) | `#2563eb` (1.5px) | `#b8860b` (1.5px) |
| **Huy Hiệu `[✔ Đang Dùng]`** | Nền `#2563eb` Chữ trắng | Nền `#dbeafe` Chữ `#1e40af` | Nền `#e2d5b5` Chữ `#7a5200` |
| **Chữ Tiêu Đề & Email** | `#f3f4f6` (Bold) | `#0f172a` (Bold) | `#2c251e` (Bold) |
| **Chữ Nhãn Chu Kỳ (`5h:`, `week:`)**| `#9ca3af` | `#64748b` | `#786b5e` |
| **Quota Xanh (> 50%)** | `#22c55e` (Emerald) | `#16a34a` (Green) | `#2e7d32` (Forest) |
| **Quota Vàng (15% - 50%)** | `#f59e0b` (Amber) | `#d97706` (Amber) | `#b45309` (Warm Amber) |
| **Quota Đỏ (< 15%)** | `#ef4444` (Rose) | `#dc2626` (Red) | `#c62828` (Crimson) |

---

## 7. Kế Hoạch Triển Khai Cho Coder Lane & Tiêu Chuẩn Nghiệm Thu

### 7.1 Phân Vùng Chỉnh Sửa Mã Nguồn Sản Phẩm (Touch-List)
Khi nhận lệnh triển khai từ CEO, Coder Lane sẽ thao tác trên các tệp sau:
1. `internal/telemetry/client.go`:
   - Cập nhật struct `AccountQuotaData` sang định dạng `ModelDualQuota` (5h và week cho Claude & Gemini).
   - Loại bỏ hoàn toàn trường `GPT` khỏi telemetry struct.
   - Nâng cấp hàm `ParseQuotaSummary` bóc tách độc lập `gemini-5h`, `gemini-weekly`, `3p-5h`, `3p-weekly`.
   - Cập nhật hàm gọi Google Cloud Code endpoint sang `v1internal:retrieveUserQuotaSummary`.
2. `internal/ui/quota_badge.go`:
   - Xóa bỏ `NewQuotaCountdownLabel` (cảnh báo sụt quota).
   - Xóa bỏ huy hiệu GPT trong `NewQuotaBadgeGroup`.
   - Thiết kế widget mới `NewDualCycleQuotaBadge(quota *telemetry.AccountQuotaData)` kết xuất 2 dòng: dòng 5h và dòng tuần.
3. `internal/ui/widgets.go`:
   - Viết lại `NewAccountCardWithQuota`: loại bỏ nút Chuyển, nút Lưu, nút Làm mới nhỏ.
   - Thêm cơ chế bọc thẻ dạng bấm được (Card Click-to-Switch).
   - Bố trí Email và huy hiệu `[✔ Đang dùng]` trên dòng đầu, 2 dòng quota phía dưới.
4. `internal/ui/window.go`:
   - Viết lại `setupLayout()`:
     - Dòng 2: Sub-header chứa Language Server info và Theme selector.
     - Dòng 3: Toolbar chứa 3 nút (`Làm mới Chrome`, `Làm mới Quota`, `Ghim Bật/Tắt`) và nhãn timestamp `Cập nhật : HH:MM:SS`.
     - Xóa bỏ `a.saveBtn`, `a.headerCard`, `legendCard`, `bottomSection` (status bar footer).
     - Thay đổi `a.accountListContainer` từ `NewVBox` sang `NewGridWithColumns(3)`.
   - Mở rộng kích thước cửa sổ mặc định lên `780 x 520 px`.
5. `internal/ui/app.go`:
   - Kết nối sự kiện làm mới Quota và Chrome cập nhật lên giao diện lưới 3 cột.

### 7.2 Tiêu Chuẩn Nghiệm Thu (Acceptance Criteria for QA & Supervisor)
1. **Khớp 100% bản vẽ của Human Owner:** Đối soát trực quan giao diện thực tế với ảnh `reports/problem/images/drawing_upright.jpg`.
2. **Zero GPT Artifacts:** Không còn bất kỳ chữ `GPT` nào xuất hiện trên giao diện và cấu trúc dữ liệu thẻ.
3. **Dual-Cycle Display:** Mọi tài khoản đều hiển thị 2 dòng: `5h: Claude ... Gemini ...` và `week: Claude ... Gemini ...`.
4. **3-Column Grid:** Các tài khoản tự động căn đều thành 3 cột cân xứng, cuộn mượt mà.
5. **Card Click-to-Switch:** Click vào thẻ tài khoản phụ là kích hoạt chuyển đổi tức thì, huy hiệu `[✔ Đang dùng]` lập tức chuyển sang thẻ mới.
6. **Zero Regression:** `go test ./...` đạt 100% PASS trên tất cả 9 packages.

---

## 8. Kết Luận & Handoff Của Kiến Trúc Sư (Architect Verdict)

Báo cáo đặc tả kiến trúc đã hoàn tất với đầy đủ bằng chứng thực nghiệm từ Google API và đối soát tỉ mỉ từng chi tiết trên bản vẽ của Human Owner.

**TRẠNG THÁI: [VERDICT: READY_FOR_OWNER_REVIEW]**  
Kiến trúc sư bàn giao báo cáo này về CEO Session (`01a14378-4a08-77c1-88d5-666ad8c3592d`) để phê duyệt và điều phối các bước triển khai tiếp theo.
