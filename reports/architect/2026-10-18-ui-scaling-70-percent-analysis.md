# Báo Cáo Phân Tích Kiến Trúc: Thu Nhỏ Kích Thước Toàn Bộ Giao Diện Desktop GUI Xuống ~70% Dựa Trên Khảo Sát Trực Tiếp Codebase

- **Tài liệu bàn giao:** `reports/architect/2026-10-18-ui-scaling-70-percent-analysis.md`
- **Ngày lập:** 2026-10-18
- **Làn thực hiện:** Senior System Architect
- **Cơ quan chỉ đạo:** CEO Agent (Thread ID: `01a14378-4a08-77c1-88d5-666ad8c3592d`)
- **Căn cứ tối cao:** Chỉ thị trực tiếp từ Human Owner ("*UI hiện tại có kích thước hơi lớn. Nên giảm lại còn khoảng 70% kích thước hiện tại thì đẹp hơn. Gọi 1 lane kiến trúc sư ra phân tích vấn đề này*") và khảo sát thực nghiệm trực tiếp trên toàn bộ codebase `internal/ui`.
- **Trạng thái:** ARCHITECT SPECIFICATION APPROVED — READY FOR CODER IMPLEMENTATION

---

## 1. Tóm Tắt Điều Hành (Executive Summary)

Theo chỉ đạo trực tiếp của Human Owner, kích thước giao diện hiện tại của ứng dụng desktop Antigravity Switcher (1440x900 px) chiếm dụng không gian màn hình tương đối lớn. Mục tiêu cốt lõi là **thu nhỏ kích thước tổng thể toàn bộ giao diện xuống khoảng 70%** để cửa sổ gọn gàng, thanh thoát, tinh tế và dễ dàng đặt cạnh các cửa sổ IDE / trình duyệt mà vẫn:
1. **Bảo toàn 100% bố cục lưới 3 cột (3-Column Grid):** Danh sách tài khoản hiển thị cân đối thành 3 cột.
2. **Bảo toàn 100% cấu trúc 3 dòng của thẻ tài khoản (3-Row Cards):**
   - Dòng 1: `email` (với ellipsis chống tràn + tooltip) + Cụm thao tác (`[Chuyển]` / `[✔ Dùng]` + `[🔄]`).
   - Dòng 2: `5h: Claude <pct>% Gemini <pct>%` kèm dot màu trạng thái.
   - Dòng 3: `week: Claude <pct>% Gemini <pct>%` kèm dot màu trạng thái.
3. **Không làm vỡ, ép hay tràn chữ:** Mọi nhãn quota, email và các nút điều khiển trong TopSection (Header, Sub-header, Toolbar) đều có không gian hiển thị vừa vặn, không bị che khuất hay đè lên nhau.

---

## 2. Khảo Sát Thực Tế Kích Thước Hiện Tại Trên Codebase

Qua khảo sát trực tiếp mã nguồn trong gói `internal/ui`, các thông số kích thước hiện hành đang được cấu hình cứng như sau:

| Thành phần giao diện | Vị trí định nghĩa trong Codebase | Giá trị hiện tại | Nhận định kỹ thuật |
|---|---|---|---|
| **Kích thước cửa sổ mặc định (Default Window Size)** | `internal/ui/app.go:201` (`a.win.Resize(...)`) | `1440 x 900 px` | Khá lớn trên màn hình 1080p, chiếm gần như toàn màn hình nếu không phóng to. |
| **Kích thước cửa sổ tối thiểu (Min Window Size)** | `internal/ui/window.go:148` (`minSizeLayout{min: ...}`) | `1200 x 750 px` | Ngăn không cho người dùng thu nhỏ dưới 1200px, gây chiếm dụng diện tích làm việc. |
| **Hardcoded Width Fallback của Container** | `internal/ui/app.go:676` (`if w <= 0 { w = 1424 }`) | `1424 px` | Con số gán cứng dựa trên cửa sổ 1440px trừ lề 16px. |
| **Độ rộng trung bình mỗi thẻ trong lưới 3 cột** | `internal/ui/window.go:132` (`container.NewGridWithColumns(3)`) | `~445 px` (ở 1440px) / `~365 px` (ở 1200px) | Chiều rộng thẻ rất rộng, dư thừa nhiều khoảng trắng ở dòng 2 và dòng 3. |
| **Chiều cao thẻ tài khoản** | `internal/ui/widgets.go:154` (`container.NewStack(...)`) | `~80 - 84 px` | Chiều cao rất gọn gàng và chuẩn mực cho cấu trúc 3 dòng. |
| **Định dạng phông chữ Quota & Dot** | `internal/ui/quota_badge.go:139, 178` | TextSize = 11, Dot = 7x7 px | Rất tối ưu, dễ đọc và sắc nét. |

---

## 3. Phân Tích Phương Án Thu Nhỏ Xuống ~70%

### 3.1 Tính Toán Kích Thước Cửa Sổ Mới
Áp dụng tỷ lệ ~70% (0.7x) trực tiếp vào các thông số kích thước cửa sổ:

- **Kích thước cửa sổ mặc định mới:**
  - Chiều rộng: `1440 px * 0.7 = 1008 px` (hoặc làm tròn thẩm mỹ **`1000 px`** hoặc chuẩn tỷ lệ 16:10 là **`1024 x 640 px`**). Đề xuất chuẩn kiến trúc: **`1008 x 630 px`** (chính xác 70.0% so với 1440x900).
- **Kích thước cửa sổ tối thiểu mới (Min Size):**
  - Chiều rộng: `1200 px * 0.7 = 840 px`.
  - Chiều cao: `750 px * 0.7 = 525 px`.
  - Đề xuất chuẩn kiến trúc: **`840 x 525 px`**.

### 3.2 Tính Toán Không Gian Lưới 3 Cột (Grid & Card Width)
Trong `internal/ui/window.go`:
```go
a.accountListContainer = container.NewGridWithColumns(3)
scrollList := container.NewVScroll(container.NewPadded(container.NewVBox(a.accountListContainer)))
```
Fyne sử dụng:
- Lề đệm container (`container.NewPadded`): 4px mỗi bên = 8px.
- VScroll thanh cuộn và insets viền cửa sổ: ~8px.
- Tổng lề trừ đi: ~16px.
- Khoảng cách giữa các cột trong `GridWithColumns(3)`: 2 khoảng cách x 4px (`theme.Padding()`) = 8px.

**Bảng tính toán chiều rộng thẻ thực tế:**

| Trạng thái kích thước cửa sổ | Chiều rộng cửa sổ | Chiều rộng khả dụng container | Chiều rộng mỗi thẻ (Card Width) | Chiều rộng nội dung trong thẻ (trừ 8px padding thẻ) |
|---|---|---|---|---|
| **Mặc định (~70%)** | **1008 px** | `1008 - 16 = 992 px` | `(992 - 8) / 3 =` **328 px** | **320 px** |
| **Tối thiểu (MinSize 70%)** | **840 px** | `840 - 16 = 824 px` | `(824 - 8) / 3 =` **272 px** | **264 px** |
| *Cũ (100%)* | *1440 px* | *1424 px* | *(1424 - 8) / 3 = 472 px* | *464 px* |

---

## 4. Thẩm Định Chi Tiết Khả Năng Hiển Thị Trên Thẻ (Pixel-Level Inspection)

Mỗi thẻ tài khoản gồm đúng 3 dòng nội dung. Chúng ta tiến hành thẩm định chi tiết kích thước của từng thành phần trên chiều rộng khả dụng **320 px** (mặc định) và **264 px** (tối thiểu).

### 4.1 Dòng 1: Email + Cụm Nút Thao Tác (Action Cluster)
Đoạn mã hiện tại trong `internal/ui/widgets.go`:
```go
emailLabel := newEmailLabelWithTooltip(item.Email, item.IsActive)
refreshBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), ...) // Nút [🔄]
if item.IsActive {
    activeBadge := widget.NewLabelWithStyle("[✔ Dùng]", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true})
    actionCluster = container.NewHBox(activeBadge, refreshBtn)
} else {
    switchBtn := widget.NewButton("Chuyển", ...)
    actionCluster = container.NewHBox(switchBtn, refreshBtn)
}
topRow := container.NewBorder(nil, nil, nil, actionCluster, emailLabel)
```

**Đo lường kích thước pixel:**
1. **Nút Làm Mới Thẻ `[🔄]`:**
   - Icon kích thước chuẩn: 16-20 px.
   - Nút không nhãn chữ với LowImportance: chiều rộng khoảng **~34 px**.
2. **Nút `[Chuyển]`:**
   - Nhãn "Chuyển" (7 ký tự, font 12pt) + padding nút = **~66 px**.
3. **Huy hiệu `[✔ Dùng]`:**
   - Nhãn "[✔ Dùng]" (8 ký tự, font 12pt bold) = **~60 px**.
4. **Tổng chiều rộng cụm Action bên phải:**
   - Khi tài khoản Inactive: `66px (Chuyển) + 34px (🔄) + 4px (spacing) =` **~104 px**.
   - Khi tài khoản Active: `60px (✔ Dùng) + 34px (🔄) + 4px (spacing) =` **~98 px**.
5. **Không gian còn lại cho nhãn Email:**
   - Ở kích thước mặc định (nội dung 320 px): `320 px - 104 px =` **216 px**.
     - Một ký tự email ở font 12pt chiếm trung bình ~7.5 px.
     - 216 px hiển thị trọn vẹn **~28 ký tự** (ví dụ: `hoangtam.nguyen2026@gmail.com` dài 27 ký tự hiển thị đầy đủ 100% không bị cắt!).
     - Nếu email dài hơn 28 ký tự: widget `emailLabelWithTooltip` tự động cắt đuôi dạng `...` nhờ `Truncation = fyne.TextTruncateEllipsis`, và khi rê chuột (hover) sẽ bung tooltip hiển thị toàn bộ email.
   - Ở kích thước tối thiểu (nội dung 264 px): `264 px - 104 px =` **160 px**.
     - Hiển thị trọn vẹn **~21 ký tự** trước khi có dấu `...`.
6. **Kết luận Dòng 1:** Cụm nút bấm bên phải được cố định kích thước, nhãn email co giãn linh hoạt ở giữa qua `container.NewBorder`. **Tuyệt đối KHÔNG có hiện tượng nút bị đẩy văng hay đè chữ.**

### 4.2 Dòng 2: Hạn Ngạch 5 Giờ (`5h: Claude <pct>% Gemini <pct>%`)
Đoạn mã hiện tại trong `internal/ui/quota_badge.go`:
```go
// CreateQuotaCycleRow("5h:", dq.Claude5h, dq.Gemini5h, ...)
// HBox: [lbl "5h:"] [claudeItem: dot + "Claude 100%"] [geminiItem: dot + "Gemini 100%"]
```
**Đo lường kích thước pixel:**
- Nhãn `"5h:"` (TextSize 11, Bold, 3 ký tự): **~22 px**.
- Mục Claude (`createModelQuotaItem("Claude", fraction)`):
  - Dot tròn: 7x7 px trong GridWrap = **7 px**.
  - Khoảng cách giữa Dot và chữ: **4 px**.
  - Nhãn chữ `"Claude 100%"` (11 ký tự, TextSize 11, Bold): **~70 px**.
  - Tổng mục Claude: `7 + 4 + 70 =` **81 px**.
- Mục Gemini (`createModelQuotaItem("Gemini", fraction)`):
  - Dot tròn (7px) + Spacing (4px) + Nhãn `"Gemini 100%"` (70px): **81 px**.
- Khoảng cách giữa 3 thành phần trong HBox: 2 khoảng cách x 4px = **8 px**.
- **Tổng chiều rộng hiển thị tối đa của Dòng 2:**
  $$\text{Width}_{\text{Row 2}} = 22 + 81 + 81 + 8 = \mathbf{192\text{ px}}$$

**Đối chiếu với không gian thẻ:**
- Chiều rộng khả dụng mặc định: **320 px** $\rightarrow$ Dư dả **128 px** ($192 < 320$).
- Chiều rộng khả dụng tối thiểu: **264 px** $\rightarrow$ Dư dả **72 px** ($192 < 264$).
- **Kết luận Dòng 2:** Dòng 2 hiển thị cực kỳ thoải mái trên một dòng duy nhất, **100% KHÔNG BAO GIỜ bị tràn hay ép xuống dòng**.

### 4.3 Dòng 3: Hạn Ngạch Tuần (`week: Claude <pct>% Gemini <pct>%`)
- Nhãn `"week:"` (TextSize 11, Bold, 5 ký tự): **~32 px**.
- Mục Claude: **81 px**.
- Mục Gemini: **81 px**.
- Khoảng cách HBox: **8 px**.
- **Tổng chiều rộng hiển thị tối đa của Dòng 3:**
  $$\text{Width}_{\text{Row 3}} = 32 + 81 + 81 + 8 = \mathbf{202\text{ px}}$$

**Đối chiếu với không gian thẻ:**
- Chiều rộng khả dụng mặc định: **320 px** $\rightarrow$ Dư dả **118 px** ($202 < 320$).
- Chiều rộng khả dụng tối thiểu: **264 px** $\rightarrow$ Dư dả **62 px** ($202 < 264$).
- **Kết luận Dòng 3:** Dòng 3 vừa vặn hoàn hảo, thẳng hàng tuyệt đối với Dòng 2.

---

## 5. Thẩm Định Khả Năng Hiển Thị Của TopSection (3 Hàng Header)

Ở kích thước cửa sổ thu nhỏ còn **1008 px** (tối thiểu **840 px**), chúng ta thẩm định 3 hàng của `TopSection` (`internal/ui/window.go`):

1. **Hàng 1 (Header):**
   - Monogram `[AS]` (28x28 px) + Tiêu đề `"Antigravity Multi-Account Switcher"` (~250 px) = **~285 px**.
   - Cửa sổ rộng 1008 px $\rightarrow$ Dư > 700 px. Cực kỳ thoáng đãng.
2. **Hàng 2 (Sub-header):**
   - Bên trái: `serverStatusLabel` (`"Language Server : PID 7768 | Port 52704 | Hoạt động"`) = **~320 px**.
   - Bên phải: `themeBox` (Nhãn `"Theme"` + Dropdown `[🌙 Tối ▼]`) = **~155 px**.
   - Tổng chiều rộng chiếm dụng cả 2 bên: `320 + 155 =` **475 px**.
   - Cửa sổ rộng 1008 px $\rightarrow$ Dư > 500 px ở khoảng giữa. Kể cả ở mức tối thiểu 840 px vẫn dư > 360 px. Tuyệt đối không va chạm.
3. **Hàng 3 (Toolbar):**
   - Cụm nút bên trái:
     - `[🔄 Làm mới Chrome]`: **~140 px**.
     - `[🔄 Làm mới Quota]`: **~135 px**.
     - `[📌 Ghim : Bật]`: **~105 px**.
     - Spacing: 2 x 4px = **8 px**.
     - Tổng bên trái: `140 + 135 + 105 + 8 =` **~388 px**.
   - Bên phải: `lastUpdatedLabel` (`"Cập nhật : 15:04:05"`) = **~130 px**.
   - Tổng chiều rộng chiếm dụng cả 2 đầu: `388 + 130 =` **518 px**.
   - Đối chiếu với cửa sổ:
     - Tại kích thước mặc định (1008 px): Còn dư **490 px** khoảng trống ở giữa.
     - Tại kích thước tối thiểu (840 px): Còn dư **322 px** khoảng trống ở giữa.
   - **Kết luận TopSection:** 3 hàng điều khiển trên cùng hoạt động hoàn hảo, cân đối, không có bất kỳ xung đột layout nào.

---

## 6. So Sánh & Lựa Chọn Giải Pháp Kỹ Thuật Trong Fyne

Có 2 cách tiếp cận kỹ thuật khả thi khi thu nhỏ giao diện trong Fyne:

| Tiêu chí đánh giá | Phương án A: Tinh chỉnh Kích thước Hình học & Lưới (Geometric Sizing - KHUYÊN DÙNG) | Phương án B: Fyne Settings Scale Factor (`SetScale(0.7)`) |
|---|---|---|
| **Cơ chế hoạt động** | Giảm kích thước cửa sổ (`Resize(1008, 630)`), giảm `minSizeLayout(840, 525)` và cập nhật container width fallback thành `992px`. | Gọi `app.Settings().SetScale(0.7)` để Fyne tự co toàn bộ canvas xuống 70%. |
| **Độ sắc nét của Font chữ** | **100% sắc nét tuyệt đối**. Phông chữ được render ở độ phân giải pixel gốc của màn hình. | **Nguy cơ mờ chữ cao**. Co tỷ lệ 0.7x khiến font 11pt biến thành ~7.7pt với hiện tượng nhòe sub-pixel trên màn hình 1080p. |
| **Chất lượng hiển thị Viền & Icon** | Đường viền thẻ 1.0px / 1.5px và bo góc 8px giữ nguyên độ sắc nét chuẩn từng pixel. Icon vector của theme không bị răng cưa. | Viền 1.0px nhân 0.7 thành 0.7px gây hiện tượng mất nét viền hoặc viền mờ đục cục bộ. |
| **Tương thích với Windows DPI Scaling** | Tương thích hoàn hảo với cài đặt hiển thị Windows (100%, 125%, 150%). | Dễ bị nhân đôi tỷ lệ scale (ví dụ: Windows 125% x Fyne 70%), gây sai lệch toạ độ chuột hoặc popup tooltip. |
| **Độ ổn định mã nguồn** | Thuần khiết và an toàn, chỉ cần cập nhật các hằng số kích thước cửa sổ và container. | Ghi đè vào file preferences toàn cục của Fyne trên máy người dùng, dễ gây tác dụng phụ. |

### Kết luận lựa chọn kiến trúc:
**CHỌN PHƯƠNG ÁN A (Geometric Sizing)** làm giải pháp thi công chính thức. 
Phương án A đáp ứng chính xác 100% mong muốn của Human Owner: diện tích cửa sổ thu gọn lại còn ~70%, không gian làm việc rộng rãi hơn, trong khi câu chữ, icon và các thành phần giao diện giữ nguyên vẻ đẹp sắc sảo, tinh tế nhất.

---

## 7. Khóa Danh Sách Tệp Thi Công Cho Coder (Touch-Map)

Khi CEO giao việc triển khai, Coder Lane chỉ cần thao tác trên đúng 3 tệp sau:

### 1. `internal/ui/window.go`
- **Dòng 50-60 & 148:**
  - Cập nhật kích thước tối thiểu trong `minSizeLayout`:
    - Cũ: `min: fyne.NewSize(1200, 750)`
    - **Mới:** `min: fyne.NewSize(840, 525)`
  - Cập nhật các dòng chú thích tương ứng từ 1440x900/1200x750 sang 1008x630/840x525.

### 2. `internal/ui/app.go`
- **Dòng 200-202:**
  - Cập nhật lệnh khởi tạo kích thước cửa sổ:
    - Cũ: `a.win.Resize(fyne.NewSize(1440, 900))`
    - **Mới:** `a.win.Resize(fyne.NewSize(1008, 630))`
- **Dòng 675-677:**
  - Cập nhật fallback width khi container chưa được vẽ:
    - Cũ: `if w <= 0 { w = 1424 }`
    - **Mới:** `if w <= 0 { w = 992 }`

### 3. `internal/ui/ui_test.go`
- **Kiểm thử `TestOwnerRedesign_WindowSizeAndLayout` (dòng 995):**
  - Cập nhật assertion kiểm tra MinSize:
    - Cũ: `if minSize.Width < 1200 || minSize.Height < 750`
    - **Mới:** `if minSize.Width < 840 || minSize.Height < 525`
- **Kiểm thử `TestOwnerRedesign_F5_CompactCardHeight` (dòng 1445, 1455):**
  - Cập nhật kích thước resize trong test môi trường giả lập:
    - Cũ: `w.Resize(fyne.NewSize(1440, 900))` và `app.accountListContainer.Resize(fyne.NewSize(1424, minH))`
    - **Mới:** `w.Resize(fyne.NewSize(1008, 630))` và `app.accountListContainer.Resize(fyne.NewSize(992, minH))`

---

## 8. Tiêu Chuẩn Nghiệm Thu Kỹ Thuật (Acceptance Criteria)

1. **Khớp Tỷ Lệ Kích Thước ~70%:**
   - Cửa sổ khi mở ra có kích thước mặc định đúng `1008 x 630 px`.
   - Giới hạn kích thước tối thiểu cho phép co kéo là `840 x 525 px`.
2. **Bố Cục 3 Cột Giữ Nguyên:**
   - Danh sách tài khoản tự động chia đều làm 3 cột trên container chiều rộng ~992 px (mỗi thẻ rộng ~328 px).
3. **Cấu Trúc 3 Dòng Thẻ Hoàn Hảo:**
   - Dòng 1: Email căn chỉnh bên trái (hiển thị tốt đến 28 ký tự trước khi có ellipsis + tooltip), cụm nút `[Chuyển]` / `[✔ Dùng]` và `[🔄]` cố định bên phải với kích thước chuẩn, bấm nhạy.
   - Dòng 2 (`5h: Claude ... Gemini ...`): chiếm tối đa 192 px, nằm lọt trong không gian 320 px khả dụng, không tràn dòng.
   - Dòng 3 (`week: Claude ... Gemini ...`): chiếm tối đa 202 px, nằm lọt trong không gian 320 px khả dụng, không tràn dòng.
4. **TopSection Không Bị Ép:**
   - Monogram [AS] và Tiêu đề, Sub-header (Server Status + Theme Selector), và Toolbar (3 nút thao tác + Timestamp) hiển thị rõ ràng, không bị đè chữ hay tràn viền.
5. **Zero Test Regressions:**
   - Toàn bộ suite `go test ./...` đạt 100% PASS.

---

## 9. Phê Duyệt & Chuyển Giao (Architect Verdict)

Bản đặc tả phân tích kiến trúc đã khóa chặt chẽ phương án kỹ thuật và Touch-Map thi công, bảo đảm 100% chuẩn xác theo chỉ đạo trực tiếp của Human Owner.

**TRẠNG THÁI: [VERDICT: READY_FOR_CODER]**
