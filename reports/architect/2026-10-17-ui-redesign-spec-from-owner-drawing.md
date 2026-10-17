# Báo Cáo Đặc Tả Kiến Trúc: Tái Cấu Trúc Toàn Diện Giao Diện Fyne GUI Theo Bản Vẽ & Chỉ Đạo Chuẩn Của Human Owner

**Tài liệu tham chiếu:** `reports/architect/2026-10-17-ui-redesign-spec-from-owner-drawing.md`  
**Ngày thực hiện:** 2026-10-17  
**Làn đảm nhiệm:** Senior System Architect (Kiến Trúc Sư Hệ Thống)  
**Cơ quan chỉ đạo:** CEO Agent (Thread ID: `01a14378-4a08-77c1-88d5-666ad8c3592d`)  
**Thẩm quyền & Căn cứ tối cao:** Chỉ thị trực tiếp từ **Human Owner**, bản vẽ phác thảo gốc tại `reports/problem/images/9180d29a.jpg`, và **mô hình bố cục thẻ 3 dòng chuẩn xác** do chính Human Owner phê duyệt và cung cấp trực tiếp:
```text
| | email1@gmail.com [✔ Dùng] [🔄]| | email2@gmail.com [Chuyển] [🔄]| | email3@gmail.com [Chuyển] [🔄]|
| | 5h: Claude 100% Gemini 88%    | | 5h: Claude 100% Gemini 23%    | | 5h: Claude 100% Gemini 95%    |
| | week: Claude 100% Gemini 90%  | | week: Claude 100% Gemini 40%  | | week: Claude 100% Gemini 98%  |
```
**Trạng thái tài liệu:** SPECIFICATION APPROVED BY ARCHITECT — READY FOR SUPERVISOR REVIEW & CODER IMPLEMENTATION

---

## 1. Tóm Tắt Điều Hành & Bối Cảnh Tái Thiết Kế (Executive Summary)

### 1.1 Yêu Cầu Trực Tiếp Từ Human Owner
Human Owner đã cung cấp một bản vẽ phác thảo tay chi tiết tại `reports/problem/images/9180d29a.jpg`, đồng thời trực tiếp chỉ định bố cục thẻ tài khoản chuẩn xác:
> **Chỉ đạo của Owner:** *"Chuyển" riêng biệt trên từng card <== nên có ngay cạnh email.*  
> **Chỉ đạo về làm mới:** *"Làm mới riêng lẻ [🔄]" trên từng card <== nên có.*

Các chỉ thị cốt lõi của Owner bao gồm:
1. **Bố cục thẻ 3 dòng chuẩn hóa 100%:**
   - **Hàng 1 (Header thẻ):** Địa chỉ Email + nút `[Chuyển]` riêng biệt (hoặc huy hiệu `[✔ Dùng]` nếu là tài khoản đang active) + nút làm mới riêng lẻ `[🔄]`.
   - **Hàng 2 (Hạn ngạch 5 giờ):** `5h: Claude <pct>% Gemini <pct>%`.
   - **Hàng 3 (Hạn ngạch tuần):** `week: Claude <pct>% Gemini <pct>%`.
2. **Quy chuẩn hiển thị Quota:**
   - Bắt buộc hiển thị theo **2 chu kỳ thời gian**: **5 giờ (5h)** và **Hàng tuần (week)**.
   - **CHỈ CẦN hiển thị hạn ngạch của Claude và Gemini; GỠ BỎ HOÀN TOÀN GPT**.
3. **Cấu trúc lưới đa cột:** Danh sách tài khoản hiển thị dạng **Lưới 3 cột (3-Column Grid)** trực quan, gọn gàng.
4. **Loại bỏ triệt để các thành phần thừa không có trong bản vẽ:** Gỡ bỏ nút Lưu phiên, nút Lưu trên thẻ, thanh chú thích Legend, cảnh báo đếm ngược dài dòng, thanh status bar dưới đáy cửa sổ.

---

## 2. Phân Tích Chi Tiết Cấu Trúc Giao Diện Chuẩn Hóa Của Owner

Căn cứ vào bản vẽ phác thảo gốc tại `reports/problem/images/9180d29a.jpg` và mô hình thẻ chuẩn hóa do Human Owner cung cấp, giao diện hoàn chỉnh được cấu trúc như sau:

```text
+----------------------------------------------------------------------------------------------------------------+
| [AS] Antigravity Multi-Account Switcher                                                              [ - ][口][ X ] |
| Language Server : PID 7768 | Port 52704 | Hoạt động                                        Theme [ Sáng      ▼ ] |
| [ 🔄 Làm mới Chrome ]   [ 🔄 Làm mới Quota ]   [ 📌 Ghim : Bật ]                                   Cập nhật : 19:24:37 |
+----------------------------------------------------------------------------------------------------------------+
| --------------------------------------------- (Đường phân cách mảnh) ------------------------------------------ |
| +----------------------------------+ +----------------------------------+ +----------------------------------+ |
| | email1@gmail.com   [✔ Dùng] [🔄] | | email2@gmail.com   [Chuyển] [🔄] | | email3@gmail.com   [Chuyển] [🔄] | |
| | 5h: Claude 100% Gemini 88%       | | 5h: Claude 100% Gemini 23%       | | 5h: Claude 100% Gemini 95%       | |
| | week: Claude 100% Gemini 90%     | | week: Claude 100% Gemini 40%     | | week: Claude 100% Gemini 98%     | |
| +----------------------------------+ +----------------------------------+ +----------------------------------+ |
| +----------------------------------+ +----------------------------------+ +----------------------------------+ |
| | email4@gmail.com   [Chuyển] [🔄] | | email5@gmail.com   [Chuyển] [🔄] | | email6@gmail.com   [Chuyển] [🔄] | |
| | 5h: Claude 100% Gemini 100%      | | 5h: Claude 100% Gemini 82%       | | 5h: Claude 100% Gemini 100%      | |
| | week: Claude 100% Gemini 100%    | | week: Claude 100% Gemini 75%     | | week: Claude 100% Gemini 100%    | |
| +----------------------------------+ +----------------------------------+ +----------------------------------+ |
| +----------------------------------+ +----------------------------------+ +----------------------------------+ |
| | email7@gmail.com   [Chuyển] [🔄] | | (Trống)                          | | (Trống)                          | |
| | 5h: Claude 100% Gemini 100%      | |                                  | |                                  | |
| | week: Claude 100% Gemini 100%    | |                                  | |                                  | |
| +----------------------------------+ +----------------------------------+ +----------------------------------+ |
+----------------------------------------------------------------------------------------------------------------+
```

### 2.1 Khối 1: Thanh Tiêu Đề Cửa Sổ (Window Header & Controls)
- **Vị trí:** Hàng 1, sát mép trên cửa sổ.
- **Bên trái:**
  - Biểu tượng Logo `[AS]` (Antigravity Switcher Logo).
  - Tên ứng dụng: `Antigravity Multi-Account Switcher` (chữ in đậm).
- **Bên phải:** Cụm điều khiển cửa sổ tiêu chuẩn: Thu nhỏ `[ - ]`, Phóng to `[ 口 ]`, Đóng `[ X ]` (do Fyne Window Manager quản lý).

### 2.2 Khối 2: Trạng Thái Language Server & Bộ Chọn Giao Diện (Sub-Header Row)
- **Vị trí:** Hàng 2, ngay dưới tiêu đề.
- **Bên trái:**
  - Nhãn trạng thái Language Server theo cú pháp chính xác:
    ```text
    Language Server : PID <PID> | Port <PORT> | Hoạt động
    ```
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
  2. `[ 🔄 Làm mới Quota ]`: Quét song song và cập nhật hạn ngạch quota cho toàn bộ tài khoản (thay thế nút cồng kềnh cũ `[📊 Làm mới Quota Tất cả]`).
  3. `[ 📌 Ghim : Bật ]` / `[ 📌 Ghim : Tắt ]`: Nút ghim cửa sổ luôn nổi lên trên (Always-on-Top toggle). Khi đang bật hiển thị `Ghim : Bật` (High Importance/màu sáng), khi tắt hiển thị `Ghim : Tắt` (Medium Importance).
- **Bên phải (Thời gian cập nhật):**
  - Nhãn text hiển thị thời điểm đồng bộ gần nhất:
    ```text
    Cập nhật : 19:24:37
    ```

### 2.4 Khối 4: Vạch Ngăn Cách (Horizontal Separator)
- Đường kẻ ngang thanh mảnh (1px border/separator) tách biệt cụm header điều khiển với khu vực hiển thị danh sách tài khoản.

### 2.5 Khối 5: Lưới Thẻ Tài Khoản 3 Cột Chuẩn Hóa Của Owner (3-Column Account Card Grid)
- **Vị trí:** Thân chính của cửa sổ, đặt bên trong container cuộn dọc (`container.NewVScroll`).
- **Bố cục lưới:** Chia thành **3 cột đều nhau** (`container.NewGridWithColumns(3, ...)`).
- **Cấu trúc chi tiết 3 dòng của từng Thẻ Tài Khoản:**
  - **Dòng 1 (Card Header Row):**
    - Bên trái: Địa chỉ Email (ví dụ: `email1@gmail.com`).
    - Bên phải (Cụm tác vụ & trạng thái):
      - Đối với **Tài khoản đang dùng (Active):** Hiển thị badge `[✔ Dùng]` kèm nút `[🔄]`.
      - Đối với **Tài khoản sẵn sàng (Inactive/Ready):** Hiển thị nút bấm `[Chuyển]` kèm nút `[🔄]`.
      - Nút làm mới riêng lẻ `[🔄]`: Nút bấm icon nhỏ gọn (`theme.ViewRefreshIcon()`, `widget.LowImportance`), click vào sẽ quét riêng hạn ngạch của tài khoản này (~50ms) trong background goroutine.
      - Nút chuyển đổi `[Chuyển]`: Nút bấm `widget.HighImportance`, click vào kích hoạt hot-swap mượt sang tài khoản này.
  - **Dòng 2 (Hạn ngạch chu kỳ 5 giờ):**
    ```text
    5h: Claude <pct>% Gemini <pct>%
    ```
    *(Ví dụ chuẩn của Owner: `5h: Claude 100% Gemini 88%`)*.
  - **Dòng 3 (Hạn ngạch chu kỳ Tuần):**
    ```text
    week: Claude <pct>% Gemini <pct>%
    ```
    *(Ví dụ chuẩn của Owner: `week: Claude 100% Gemini 90%`)*.
  - **Màu sắc theo ngưỡng hạn ngạch:**
    - Xanh lá (`#22c55e`): Quota $> 50%$.
    - Vàng hổ phách (`#f59e0b`): Quota từ $15%$ đến $50%$.
    - Đỏ cảnh báo (`#ef4444`): Quota $< 15%$.
    - Xám nhạt (`#888899`): Chưa có dữ liệu / Lỗi kết nối `[!]`.

---

## 3. Ma Trận Đối Soát: Thành Phần Gỡ Bỏ Triệt Để & Thành Phần Giữ Lại / Thêm Mới

### 3.1 Bảng Danh Sách Các Thành Phần Phải GỠ BỎ TRIỆT ĐỂ

| STT | Thành phần cũ cần gỡ bỏ | Vị trí cũ trong mã nguồn | Lý do gỡ bỏ |
|---|---|---|---|
| 1 | **Toàn bộ hiển thị Quota GPT** | `quota_badge.go`, `widgets.go`, `window.go` | **Chỉ thị trực tiếp của Owner**: Bỏ hoàn toàn GPT, chỉ giữ Claude & Gemini. |
| 2 | **Khối Active Header Card cồng kềnh** | `window.go` (`a.headerCard`, `a.activeStatusLabel`, `a.activeQuotaLabel`) | Không có trong bản vẽ. Thông tin tài khoản đang dùng đã được tích hợp ngay trong thẻ tài khoản (`[✔ Dùng]`). |
| 3 | **Nút `[💾 Lưu phiên]` trên toolbar** | `window.go` (`a.saveBtn`) | Không có trong bản vẽ. |
| 4 | **Nút `[💾 Lưu]` và badge `[Chưa lưu]` trên từng thẻ** | `widgets.go` (`saveBtn`, `unauthBadge`) | Không có trong bản vẽ. Việc lưu trữ diễn ra tự động ngầm. |
| 5 | **Thanh chú thích màu (Legend Card)** | `window.go` (`legendCard`: 🟢 Sẵn sàng \| 🟡 Chưa có JSON \| 🔴 <15%) | Không có trong bản vẽ của Owner. |
| 6 | **Thanh trạng thái đáy cửa sổ (Footer Status Bar)** | `window.go` (`statusBarLabel`, `bottomSection`: "Sẵn sàng", "Đồng bộ lúc...") | Không có trong bản vẽ của Owner. Nhãn thời gian cập nhật đã được đưa lên góc phải hàng 3 (`Cập nhật : 19:24:37`). |
| 7 | **Dòng cảnh báo đếm ngược hồi phục quota** | `quota_badge.go` (`NewQuotaCountdownLabel`: "⚠️ ... sắp cạn lúc HH:mm UTC") | Không có trong bản vẽ của Owner. Làm thẻ bị phình to, phá vỡ bố cục lưới gọn gàng. |
| 8 | **Bố cục cuộn dọc 1 cột (Vertical VBox)** | `window.go` (`a.accountListContainer = container.NewVBox()`) | Thay thế triệt để bằng Grid 3 cột. |

### 3.2 Bảng Danh Sách Các Thành Phần THÊM MỚI & GIỮ LẠI THEO CHỈ THỊ OWNER

| STT | Thành phần | Trạng thái | Quy chuẩn kiến trúc & Hành vi |
|---|---|:---:|---|
| 1 | **Nút bấm riêng biệt `[Chuyển]` ngay cạnh email** | **GIỮ LẠI & ĐẶT CẠNH EMAIL** | **Chỉ đạo trực tiếp từ Owner**: *"Chuyển" riêng biệt trên từng card <== nên có ngay cạnh email.* Nút bấm `[Chuyển]` hiển thị trên mọi thẻ chưa active, bấm vào chuyển đổi ngay sang tài khoản đó. |
| 2 | **Huy hiệu `[✔ Dùng]` ngay cạnh email** | **THÊM MỚI** | Hiển thị tại vị trí nút Chuyển đối với tài khoản đang kích hoạt trong Antigravity. |
| 3 | **Nút làm mới riêng lẻ `[🔄]` trên thẻ** | **GIỮ LẠI & TINH CHỈNH** | **Chỉ đạo trực tiếp từ Owner**: *"Làm mới riêng lẻ [🔄]" trên từng card <== nên có.* Đặt ngay cạnh nút `[Chuyển]` hoặc badge `[✔ Dùng]`, quét riêng tài khoản đó (~50ms). |
| 4 | **Hàng Sub-Header hợp nhất** | **THÊM MỚI** | Bên trái: `Language Server : PID <PID> \| Port <PORT> \| Hoạt động`. Bên phải: `Theme [ Sáng ▼ ]`. |
| 5 | **Thanh Toolbar tinh gọn 3 nút + Timestamp** | **THÊM MỚI** | Trái: `[ 🔄 Làm mới Chrome ]` `[ 🔄 Làm mới Quota ]` `[ 📌 Ghim : Bật ]`. Phải: `Cập nhật : <HH:MM:SS>`. |
| 6 | **Lưới tài khoản 3 cột (3-Column Grid)** | **THÊM MỚI** | `container.NewGridWithColumns(3, ...)` tự động dàn đều các thẻ tài khoản sang 3 cột. |
| 7 | **Hiển thị Quota 2 chu kỳ (5h & week)** | **THÊM MỚI** | • Dòng 2: `5h: Claude <pct>% Gemini <pct>%`<br>• Dòng 3: `week: Claude <pct>% Gemini <pct>%`. |

---

## 4. Khảo Sát Kỹ Thuật: Cơ Chế Bóc Tách Hạn Ngạch Dual-Cycle (5h & Tuần) Từ Google API

### 4.1 Bằng Chứng Thực Nghiệm Trực Tiếp Từ Runtime Google API

Trong quá trình điều tra kiến trúc, Kiến trúc sư đã trực tiếp gửi HTTP request kèm access token tươi tới các endpoint của Google và Antigravity Language Server:

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
Cùng cấu trúc dữ liệu chính xác, được bọc trong trường gốc `{"response": { "groups": [...] }}`.

### 4.2 Nguyên Lý Bóc Tách Bucket Của Google
1. **Nhóm `Gemini Models`:**
   - Bucket chu kỳ 5 giờ: `bucketId == "gemini-5h"` hoặc `window == "5h"`.
   - Bucket chu kỳ tuần: `bucketId == "gemini-weekly"` hoặc `window == "weekly"`.
2. **Nhóm `Claude and GPT models` (3P):**
   - Trong Google Cloud Code, Claude dùng chung quota với nhóm 3P.
   - Bucket chu kỳ 5 giờ: `bucketId == "3p-5h"` hoặc `window == "5h"`.
   - Bucket chu kỳ tuần: `bucketId == "3p-weekly"` hoặc `window == "weekly"`.
3. **Quy tắc gỡ bỏ GPT:**
   - Cấu trúc dữ liệu và giao diện loại bỏ hoàn toàn trường GPT, chỉ giữ trường Claude và Gemini.

### 4.3 Định Nghĩa Cấu Trúc Dữ Liệu Quota Mới (`ModelDualQuota` & `AccountQuotaData`)
Trong `internal/telemetry/client.go`:

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

---

## 5. Cấu Trúc Cây Widget Fyne & Bản Đồ Tương Tác (Layout Tree & Touch-Map)

### 5.1 Cây Bố Cục Giao Diện (Fyne Widget Hierarchy Tree)

```text
Window (fyne.Window) - Default Size: 840x540 (Mở rộng thoải mái cho Lưới 3 Cột có nút Chuyển)
│
└── MainContainer (container.NewBorder)
    ├── Top: TopSection (container.NewVBox)
    │   ├── Row 1: HeaderRow (container.NewBorder)
    │   │   ├── Left: container.NewHBox(Logo [AS], Title "Antigravity Multi-Account Switcher")
    │   │   └── Right: WindowControls
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
            ├── AccountCard_Active (Stack: CardBackground + Padded VBox)
            │   ├── Line 1: container.NewBorder(nil, nil, EmailLabel, ActionHBox)
            │   │   └── ActionHBox: container.NewHBox(activeBadge "[✔ Dùng]", refreshSingleBtn "[🔄]")
            │   ├── Line 2: container.NewHBox("5h:", Claude5hBadge, Gemini5hBadge)
            │   └── Line 3: container.NewHBox("week:", ClaudeWeekBadge, GeminiWeekBadge)
            │
            ├── AccountCard_Inactive (Stack: CardBackground + Padded VBox)
            │   ├── Line 1: container.NewBorder(nil, nil, EmailLabel, ActionHBox)
            │   │   └── ActionHBox: container.NewHBox(switchBtn "[Chuyển]", refreshSingleBtn "[🔄]")
            │   ├── Line 2: container.NewHBox("5h:", Claude5hBadge, Gemini5hBadge)
            │   └── Line 3: container.NewHBox("week:", ClaudeWeekBadge, GeminiWeekBadge)
            │
            └── ... (Tự động xuống dòng theo cột lưới)
```

### 5.2 Kích Thước & Tỷ Lệ Cửa Sổ (Window Dimensions)
- **Kích thước khuyến nghị:** `840 x 540 px`.
  - Chiều rộng `840 px` chia đều 3 cột, mỗi thẻ tài khoản có bề rộng xấp xỉ `260 px` — đủ không gian hoàn hảo cho:
    - Địa chỉ email ở góc trái.
    - Cụm `[Chuyển] [🔄]` (hoặc `[✔ Dùng] [🔄]`) ở góc phải ngay cạnh email.
    - 2 dòng hạn ngạch `5h: Claude ... Gemini ...` và `week: Claude ... Gemini ...` bên dưới.
  - Chiều cao `540 px` hiển thị ngay 2 hàng (6 tài khoản) trọn vẹn không cần cuộn, thanh cuộn mượt mà kích hoạt khi danh sách có từ 7 tài khoản trở lên.

### 5.3 Bản Đồ Tương Tác Cảm Ứng & Sự Kiện (Touch-Map & Interaction Map)

| Khu vực tương tác | Widget Fyne | Sự kiện (Event) | Hành vi kích hoạt (Behavior) |
|---|---|---|---|
| **Bộ chọn Theme** | `widget.Select` (`a.themeSelect`) | `OnChanged` | Đổi ngay theme toàn ứng dụng (`ThemeDark`, `ThemeLight`, `ThemeWarm`), cập nhật bảng màu của toàn bộ canvas mà không đóng app. |
| **Nút Làm mới Chrome** | `widget.Button` (`a.refreshChromeBtn`) | `OnTapped` | Gọi `App.Refresh()`: quét Chrome User Data profiles, phát hiện tài khoản mới. |
| **Nút Làm mới Quota** | `widget.Button` (`a.refreshQuotaBtn`) | `OnTapped` | Gọi `App.RefreshAllQuota()`: quét song song hạn ngạch toàn bộ tài khoản, cập nhật nhãn `Cập nhật : HH:MM:SS`. |
| **Nút Ghim Bật/Tắt** | `widget.Button` (`a.alwaysOnTopBtn`) | `OnTapped` | Gọi `App.ToggleAlwaysOnTop()`: đảo trạng thái ghim OS, đổi nhãn giữa `📌 Ghim : Bật` và `📌 Ghim : Tắt`. |
| **Nút Chuyển riêng lẻ trên thẻ** | `widget.Button` (`switchBtn [Chuyển]`) | `OnTapped` | Gọi `App.SwitchAccount(email)`: hot-swap tức thì sang tài khoản này, đổi `[Chuyển]` thành `[✔ Dùng]`. |
| **Nút làm mới riêng lẻ trên thẻ** | `widget.Button` (`refreshSingleBtn [🔄]`) | `OnTapped` | Gọi `App.RefreshSingleAccountQuota(email)`: chỉ quét hạn ngạch riêng của tài khoản này (~50ms) trong goroutine nền, cập nhật riêng thẻ đó. |

---

## 6. Bảng Mã Màu & Quy Chuẩn Thẩm Mỹ Đa Chủ Đề (Themes Palette)

| Thành phần | Chủ đề Tối (Dark) | Chủ đề Sáng (Light) | Chủ đề Ấm (Warm) |
|---|:---:|:---:|:---:|
| **Nền Cửa Sổ (Background)** | `#121216` | `#f8f9fa` | `#fbf7ee` |
| **Nền Thẻ Thường (Card Bg)** | `#1c1d22` | `#ffffff` | `#f4ede0` |
| **Viền Thẻ Thường (Card Border)** | `#2c2d35` (1px) | `#e2e8f0` (1px) | `#e5dac8` (1px) |
| **Nền Thẻ Đang Dùng (Active Card Bg)**| `#1a2634` | `#eef6ff` | `#ede6d4` |
| **Viền Thẻ Đang Dùng (Active Border)** | `#3b82f6` (1.5px) | `#2563eb` (1.5px) | `#b8860b` (1.5px) |
| **Huy Hiệu `[✔ Dùng]`** | Nền `#2563eb` Chữ trắng | Nền `#dbeafe` Chữ `#1e40af` | Nền `#e2d5b5` Chữ `#7a5200` |
| **Nút `[Chuyển]`** | Primary Button (`#3b82f6`) | Primary Button (`#2563eb`) | Warm Primary (`#b8860b`) |
| **Nút Làm Mới Thẻ `[🔄]`** | Màu `#9ca3af` (Hover: Trắng) | Màu `#64748b` (Hover: Đen) | Màu `#786b5e` (Hover: Nâu đậm) |
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
   - Thiết kế widget mới `NewDualCycleQuotaBadge(quota *telemetry.AccountQuotaData)` kết xuất 2 dòng: dòng 5h và dòng tuần (`5h: Claude ... Gemini ...` và `week: Claude ... Gemini ...`).
3. `internal/ui/widgets.go`:
   - Cập nhật `NewAccountCardWithQuota`:
     - Hàng 1: Email (trái) + Action cluster (phải):
       - Nếu active: Badge `[✔ Dùng]` + Button `[🔄]`.
       - Nếu inactive: Button `[Chuyển]` + Button `[🔄]`.
     - Hàng 2: `5h: Claude <pct>% Gemini <pct>%`.
     - Hàng 3: `week: Claude <pct>% Gemini <pct>%`.
     - Xóa bỏ nút Lưu phiên, badge Chưa lưu.
4. `internal/ui/window.go`:
   - Viết lại `setupLayout()`:
     - Dòng 2: Sub-header chứa Language Server info và Theme selector.
     - Dòng 3: Toolbar chứa 3 nút (`Làm mới Chrome`, `Làm mới Quota`, `Ghim Bật/Tắt`) và nhãn timestamp `Cập nhật : HH:MM:SS`.
     - Xóa bỏ `a.saveBtn`, `a.headerCard`, `legendCard`, `bottomSection` (status bar footer).
     - Thay đổi `a.accountListContainer` từ `NewVBox` sang `NewGridWithColumns(3)`.
   - Mở rộng kích thước cửa sổ mặc định lên `840 x 540 px`.
5. `internal/ui/app.go`:
   - Duy trì hàm `RefreshSingleAccount(email string)` để phục vụ nút `[🔄]` trên thẻ.
   - Kết nối sự kiện làm mới Quota và Chrome cập nhật lên giao diện lưới 3 cột.

### 7.2 Tiêu Chuẩn Nghiệm Thu (Acceptance Criteria for QA & Supervisor)
1. **Khớp 100% bản vẽ & mô hình bố cục chuẩn của Owner:**
   - Mỗi thẻ có đúng 3 dòng:
     - Dòng 1: `email [Chuyển] [🔄]` (hoặc `email [✔ Dùng] [🔄]`).
     - Dòng 2: `5h: Claude ... Gemini ...`.
     - Dòng 3: `week: Claude ... Gemini ...`.
2. **Zero GPT Artifacts:** Không còn bất kỳ chữ `GPT` nào xuất hiện trên giao diện và cấu trúc dữ liệu thẻ.
3. **Dual-Cycle Display:** Mọi tài khoản đều hiển thị 2 dòng: `5h: Claude ... Gemini ...` và `week: Claude ... Gemini ...`.
4. **3-Column Grid:** Các tài khoản tự động căn đều thành 3 cột cân xứng, cuộn mượt mà.
5. **Nút Làm Mới Thẻ `[🔄]`:** Bấm `[🔄]` trên bất kỳ thẻ nào chỉ cập nhật dữ liệu cho thẻ đó mà không giật lag giao diện.
6. **Nút `[Chuyển]` Hoạt Động Chuẩn Xác:** Bấm `[Chuyển]` trên thẻ tài khoản phụ là kích hoạt chuyển đổi tức thì, nhãn lập tức chuyển thành `[✔ Dùng]`.
7. **Zero Regression:** `go test ./...` đạt 100% PASS trên tất cả 9 packages.

---

## 8. Kết Luận & Handoff Của Kiến Trúc Sư (Architect Verdict)

Báo cáo đặc tả kiến trúc đã được cập nhật hoàn chỉnh, tích hợp chính xác 100% chỉ đạo và mô hình bố cục 3 dòng do Human Owner trực tiếp cung cấp: có nút bấm riêng biệt `[Chuyển]` (hoặc badge `[✔ Dùng]`) ngay cạnh email và nút làm mới `[🔄]`.

**TRẠNG THÁI: [VERDICT: READY_FOR_OWNER_REVIEW]**  
Kiến trúc sư bàn giao báo cáo này về CEO Session (`01a14378-4a08-77c1-88d5-666ad8c3592d`) để chuyển tiếp sang các làn tiếp theo thực thi.
