# Báo Cáo Đặc Tả Kiến Trúc: Tinh Chỉnh Kích Thước UI Xuống ~80% (820x520) Cho Màn Hình PC Nhỏ & Bổ Sung Hiệu Ứng Nút Bấm Xúc Giác (Tactile Button Feedback)

- **Tài liệu bàn giao:** `reports/architect/2026-10-18-ui-scaling-80-percent-and-tactile-buttons-spec.md`
- **Ngày lập:** 2026-10-18
- **Làn thực hiện:** Senior System Architect
- **Cơ quan chỉ đạo:** CEO Agent (Thread ID: `01a14378-4a08-77c1-88d5-666ad8c3592d`)
- **Căn cứ & Chỉ đạo tối cao từ Human Owner:**
  1. *Chỉ đạo kích thước:* "Kích thước đã được thu nhỏ nhưng tôi muốn kích thước UI hiện tại nhỏ đi, còn khoảng 75% ~ 80% hiện tại thì sẽ hiển thị tốt hơn cho các loại màn hình PC nhỏ."
  2. *Chỉ đạo hiệu ứng nút:* "Tôi muốn các nút bấm trên UI có hiệu ứng nút bấm, chứ hiện tại nhấn vô chả có cảm giác gì."
  3. *Chỉ đạo điều phối tuần tự:* "Yêu cầu CEO sau khi Coder hoàn thành công việc hiện tại thì lập tức mở Coder mới triển khai theo báo cáo này."
- **Trạng thái:** ARCHITECT SPECIFICATION APPROVED BY OWNER — READY FOR IMMEDIATE CODER DISPATCH

---

## 1. Tóm Tắt Điều Hành (Executive Summary)

Sau khi gói thu nhỏ ban đầu (1008x630) được nghiệm thu trên thực tế, Human Owner đã trực tiếp đánh giá và đưa ra 2 chỉ đạo cải tiến mang tính đột phá về trải nghiệm người dùng (UX):
1. **Thu nhỏ tiếp kích thước UI xuống ~80% hiện tại (`820 x 520 px`):** Tối ưu hóa đặc biệt cho các loại màn hình laptop 13" – 14", màn hình độ phân giải HD (1366x768) hoặc Full HD có bật Windows Display Scaling 125% – 150%, biến ứng dụng thành một Mini Widget Utility thanh thoát, không choán diện tích làm việc.
2. **Bổ sung hiệu ứng nút bấm xúc giác & thị giác (Tactile & Visual Feedback):** Khắc phục triệt để hiện tượng "bấm vào không có cảm giác gì", bổ sung độ nảy, độ lún, highlight hover/press rõ nét và phản hồi trạng thái tức thì (`[⏳ Đang chuyển...]`, `[⏳ Đang làm mới...]`).

---

## 2. Vấn Đề 1: Tinh Chỉnh Kích Thước UI Xuống Mức ~80% Hiện Tại

### 2.1 Bảng Thông Số Kích Thước Mới
| Thông số hình học | Mức cũ (1008x630) | Mức mới ~80% (`820 x 520 px`) | Tỷ lệ giảm | Mục đích sử dụng |
|---|---|---|---|---|
| **Cửa sổ mặc định (Default)** | `1008 x 630 px` | **`820 x 520 px`** | ~81% chiều rộng, ~82% chiều cao | Chuẩn mở cửa sổ, vừa khít màn hình scale 150% |
| **Cửa sổ tối thiểu (MinSize)** | `840 x 525 px` | **`680 x 440 px`** | ~80% | Giới hạn nhỏ nhất khi người dùng co kéo |
| **Fallback Container Width** | `992 px` | **`804 px`** | `820 - 16 = 804` | Chiều rộng fallback an toàn cho container lưới |
| **Chiều rộng mỗi thẻ (Lưới 3 cột)** | `~328 px` | **`~260 px`** | `(804 - 8) / 3 = 265 px` | Thẻ thu gọn thanh lịch, không bị thừa khoảng trắng |
| **Chiều rộng nội dung thẻ khả dụng**| `~320 px` | **`~252 px`** | Trừ 8px padding thẻ | Không gian khả dụng cho 3 dòng nội dung |

### 2.2 Thẩm Định Pixel 3 Dòng Của Thẻ Trên Chiều Rộng 252 px
1. **Dòng 1: Email + Cụm Nút Thao Tác (Action Cluster):**
   - Cụm nút bên phải:
     - Nút `[Chuyển]` (hoặc badge `[✔ Dùng]`): ~60–66 px.
     - Nút làm mới thẻ `[🔄]`: ~34 px.
     - Khoảng cách đệm: 4 px.
     - Tổng cụm Action bên phải: **~104 px**.
   - Phần dành cho nhãn Email bên trái: `252 px - 104 px =` **~148 px**.
   - Với font 12pt (~7.5px/ký tự): Hiển thị trọn vẹn **~20 ký tự email** (ví dụ `user2026@gmail.com`).
   - Nếu email dài hơn 20 ký tự: `emailLabelWithTooltip` tự động thêm `...` ở đuôi, đồng thời hiển thị Tooltip toàn bộ email khi rê chuột vào. **Tuyệt đối không đè nút hay vỡ khung.**
2. **Dòng 2: Hạn Ngạch 5 Giờ (`5h: Claude ... Gemini ...`):**
   - Nhãn `"5h:"` (22px) + Claude (Dot 7px + Chữ 70px = 81px) + Gemini (Dot 7px + Chữ 70px = 81px) + Spacing (8px).
   - Tổng chiều rộng chiếm dụng: **192 px**.
   - So với không gian 252 px khả dụng: **Dư tới 60 px!** Hoàn toàn vừa vặn trên 1 dòng duy nhất.
3. **Dòng 3: Hạn Ngạch Tuần (`week: Claude ... Gemini ...`):**
   - Nhãn `"week:"` (32px) + Claude (81px) + Gemini (81px) + Spacing (8px).
   - Tổng chiều rộng chiếm dụng: **202 px**.
   - So với không gian 252 px khả dụng: **Dư tới 50 px!** Nằm gọn gàng trên 1 dòng.
4. **TopSection (Header, Sub-header, Toolbar):**
   - Chiều rộng chiếm dụng tối đa cả 2 đầu: **~518 px**.
   - So với chiều rộng cửa sổ mới 820 px: **Còn dư hơn 300 px** khoảng trống ở giữa, hoàn toàn thoáng đãng, không xảy ra chồng đè.

---

## 3. Vấn Đề 2: Thiết Kế Hiệu Ứng Nút Bấm Xúc Giác & Thị Giác (Tactile & Visual Feedback)

Để khắc phục hiện tượng "bấm vào không có cảm giác gì", kiến trúc đưa ra giải pháp 3 lớp:

### 3.1 Lớp 1: Bổ Sung Màu Tương Tác Trong Bộ Theme (`internal/ui/themes.go`)
Hiện tại `DarkTheme`, `LightTheme`, `WarmTheme` chỉ khai báo `Button` mà bỏ trống `ColorNameHover` và `ColorNamePressed`, dẫn đến việc Fyne dùng lớp phủ mặc định có độ mờ < 5% (mắt người không thấy).

**Đặc tả mã hóa màu tương tác:**
Trong phương thức `Color(name fyne.ThemeColorName, variant fyne.ThemeVariant)` của cả 3 theme:
- Khi `name == theme.ColorNameHover`:
  - `DarkTheme`: Trả về màu sáng hơn 25% $\rightarrow$ `#45475a` (Surface1 Catppuccin) hoặc viền sáng.
  - `LightTheme`: Trả về màu sẫm nhẹ $\rightarrow$ `#cbd5e1`.
  - `WarmTheme`: Trả về màu nâu sáng $\rightarrow$ `#544635`.
- Khi `name == theme.ColorNamePressed`:
  - `DarkTheme`: Trả về màu lún/chìm đậm rõ rệt $\rightarrow$ `#252538` hoặc màu Primary Accent `#89b4fa` với độ alpha 0.4.
  - `LightTheme`: Trả về `#94a3b8` / Primary `#2563eb`.
  - `WarmTheme`: Trả về `#33281d` / Primary `#d97706`.
- Khi `name == theme.ColorNameFocus`: Trả về màu viền Accent tương ứng của theme.

> **Hiệu quả thị giác:** Ngay khi con trỏ chuột chạm vào nút, nút sáng bừng lên xác nhận. Khi bấm chuột xuống, nút đổi màu đậm lún xuống tức thì.

### 3.2 Lớp 2: Cải Tiến Nút `[🔄]` Trên Thẻ Tài Khoản (`internal/ui/widgets.go`)
- **Vấn đề cũ:** `refreshBtn.Importance = widget.LowImportance` làm nút có nền trong suốt, bấm vào không thấy phản ứng.
- **Giải pháp mới:**
  - Đổi sang `Importance = widget.MediumImportance` (hoặc bọc trong container có bo góc nhẹ và nền mờ tiệp màu thẻ).
  - Khi rê chuột (Hover): Viền tròn sáng lên rõ rệt.
  - Khi bấm (Pressed): Có phản hồi chớp màu xác nhận.

### 3.3 Lớp 3: Phản Hồi Trạng Thái Tức Thì (Instant Action Feedback)
1. **Nút `[Chuyển]`:**
   - Trong `internal/ui/widgets.go`:
     - Khi người dùng click `switchBtn`: Lập tức đổi nhãn nút sang **`"⏳ Đang chuyển..."`** và `switchBtn.Disable()` để ngăn chặn click liên hoàn (double/spam click).
     - Khi callback `onSwitch(item.Email)` chạy xong và giao diện được refresh, thẻ chuyển sang trạng thái Active sẽ tự động hiển thị nhãn **`[✔ Dùng]`**.
2. **Nút `[🔄]` làm mới thẻ đơn lẻ:**
   - Khi bấm: Tạm thời vô hiệu hóa nút trong 500ms và hiển thị icon đồng hồ cát `hourglass` hoặc chớp sáng để xác nhận lệnh làm mới đã được gửi đi.
3. **Các nút Toolbar trên cùng (`internal/ui/window.go`):**
   - Nút `[🔄 Làm mới Chrome]` và `[🔄 Làm mới Quota]`:
     - Khi bấm: Lập tức đổi nhãn thành **`"⏳ Đang làm mới..."`** trong 400ms, sau đó khôi phục lại nhãn ban đầu kèm cập nhật nhãn `Cập nhật : HH:MM:SS`.

---

## 4. Khóa Danh Sách Tệp Thi Công Cho Coder (Touch-Map)

Coder Lane tiếp theo sẽ thao tác trên 4 tệp mã nguồn:

### 1. `internal/ui/app.go`
- **Dòng 201:** `a.win.Resize(fyne.NewSize(820, 520))` (thay cho `1008, 630`).
- **Dòng 676:** Fallback container width `w = 804` (thay cho `992`).

### 2. `internal/ui/window.go`
- **Dòng 148:** `minSizeLayout{min: fyne.NewSize(680, 440)}` (thay cho `840, 525`).
- **TopSection Toolbar:** Thêm phản hồi tạm thời `"⏳ Đang làm mới..."` khi click `refreshBtn` và `quotaRefreshAllBtn`.

### 3. `internal/ui/themes.go`
- Thêm các nhánh `case theme.ColorNameHover:`, `case theme.ColorNamePressed:`, `case theme.ColorNameFocus:` vào phương thức `Color(...)` của `DarkTheme`, `LightTheme`, và `WarmTheme`.

### 4. `internal/ui/widgets.go`
- Nút `[Chuyển]`: Đổi nhãn `"⏳ Đang chuyển..."` và disable khi click.
- Nút `[🔄]`: Đổi từ `LowImportance` sang `MediumImportance` để có viền nền và hiệu ứng hover/press rõ nét.

### 5. `internal/ui/ui_test.go`
- Cập nhật các assertions về kích thước:
  - `TestOwnerRedesign_WindowSizeAndLayout`: MinSize >= 680x440.
  - `TestOwnerRedesign_F5_CompactCardHeight`: Resize window về 820x520 và container về 804.

---

## 5. Tiêu Chuẩn Nghiệm Thu (Acceptance Criteria)

1. **Kích thước cửa sổ chuẩn ~80%:** Mặc định mở ra đúng `820 x 520 px`, co nhỏ tối thiểu được đến `680 x 440 px`.
2. **Không vỡ layout:** Lưới 3 cột và 3 dòng của thẻ tài khoản hiển thị vừa khít, không tràn chữ, không đè nút.
3. **Hiệu ứng nút bấm rõ rệt:**
   - Rê chuột qua bất kỳ nút nào đều thấy nút sáng lên rõ ràng.
   - Nhấn chuột xuống thấy nút chìm/đổi màu đậm ngay tức thì.
   - Bấm nút `[Chuyển]` có nhãn `[⏳ Đang chuyển...]`.
   - Nút `[🔄]` trên thẻ có khung bo tròn và phản hồi khi nhấn, không còn tàng hình.
4. **100% Test PASS:** `go test ./...` đạt 100% PASS trên tất cả các gói.

---

## 6. Chỉ Đạo Gửi CEO Agent

Báo cáo đặc tả này được bàn giao trực tiếp cho **CEO Agent** kèm theo **chỉ thị tối cao từ Human Owner**:
> *"Yêu cầu CEO sau khi Coder hoàn thành công việc hiện tại thì lập tức mở Coder mới triển khai theo báo cáo này."*
