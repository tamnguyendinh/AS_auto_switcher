# Báo Cáo Đặc Tả Kỹ Thuật: Đơn Giản Hóa Triệt Để — Hardcode Chỉ Số Cố Định, Gỡ Bỏ Toàn Bộ Thành Phần Co Giãn & Chỉ User Được Phép Đổi Kích Thước UI

- **Ngày ban hành:** 2026-10-19
- **Vai trò:** System Architect
- **Thẩm quyền:** Lệnh trực tiếp từ Human Owner
- **Mã văn kiện:** `SPEC-ARCH-20261019-HARDCODE-FIXED-RATIO-SPEC`
- **Nguyên tắc tối thượng:** 
  1. KHÔNG bổ sung cờ, timer hay logic phức tạp.
  2. GỠ BỎ toàn bộ các thành phần gây co giãn động; HARDCODE toàn bộ tỷ lệ, kích thước các cột, ô, ray thành CHỈ SỐ CỐ ĐỊNH.
  3. CHỈ CÓ USER mới được phép dùng chuột thay đổi kích thước cửa sổ UI. Kích thước này được lưu lại khi đóng ứng dụng.
  4. Khi Quota tự động làm mới: CHỈ CÓ vệt màu tiến trình `fillBar` tụt dần theo % quota bên trong ray cố định 40px, TOÀN BỘ NHỮNG THỨ KHÁC ĐỨNG YÊN 100%.

---

## 1. Bản Chất Vấn Đề & Phương Châm Đơn Giản Hóa

- Trước đây các agent tự làm phức tạp hóa vấn đề: thêm timer debounce, gắn hàm lưu đĩa vào hàm `Layout()` chạy liên tục mọi frame, tính toán bề rộng cột động theo `remainW / 2.0`... làm cho các thành phần đánh nhau và gây giật, giãn nở giao diện mỗi khi Quota refresh.
- **Giải pháp đúng đắn theo chỉ đạo của Owner:**
  - Bảng Quota: **Hardcode chỉ số cố định**. Cột Model = 50px, Cột 5h = 88px, Cột Week = 88px, Ray bar = 40px. Không có công thức co giãn nào hết $\rightarrow$ Đã cố định thì không bao giờ có thể bị giãn!
  - Quota thay đổi: Chỉ gán lại chữ % và độ dài thanh tiến trình `fillBar = 40px * fraction` (tụt dần).
  - Cửa sổ: User được tự do kéo chuột chỉnh to/nhỏ. Kích thước cửa sổ chỉ lưu 1 lần duy nhất khi ứng dụng đóng (`Close()`), hoặc khi thoát app. Tuyệt đối không can thiệp vào kích thước cửa sổ khi Quota refresh.

---

## 2. Hướng Dẫn Thi Công Chính Xác 100% Cho Coder (Zero Speculation)

### TỆP 1: `internal/ui/quota_badge.go`
*Mục đích: Hardcode toàn bộ chỉ số cột cố định, gỡ bỏ sạch sẽ mọi phép tính co giãn động.*

1. **Hàm `matrixTableLayout.Layout`:**
   Hardcode các chỉ số cố định, KHÔNG TÍNH TOÁN ĐỘNG:
   ```go
   func (l *matrixTableLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
       if len(objects) < 5 {
           return
       }
       colModel := objects[0]
       sep1 := objects[1]
       col5h := objects[2]
       sep2 := objects[3]
       colWeek := objects[4]
   
       h := size.Height
       // CHỈ SỐ CỐ ĐỊNH 100% - KHÔNG CO GIÃN
       col1W := float32(50.0)     // Cột Model: CỐ ĐỊNH 50px
       sepW := float32(1.0)      // Vách ngăn: CỐ ĐỊNH 1px
       colGap := float32(4.0)    // Khoảng cách: CỐ ĐỊNH 4px
       colCycleW := float32(88.0) // Cột 5h và Week: KHÓA CỨNG CỐ ĐỊNH 88px
   
       x := float32(0)
       colModel.Move(fyne.NewPos(x, 0))
       colModel.Resize(fyne.NewSize(col1W, h))
       x += col1W + colGap
   
       sep1.Move(fyne.NewPos(x, 1))
       sep1.Resize(fyne.NewSize(sepW, h-2))
       x += sepW + colGap
   
       col5h.Move(fyne.NewPos(x, 0))
       col5h.Resize(fyne.NewSize(colCycleW, h))
       x += colCycleW + colGap
   
       sep2.Move(fyne.NewPos(x, 1))
       sep2.Resize(fyne.NewSize(sepW, h-2))
       x += sepW + colGap
   
       colWeek.Move(fyne.NewPos(x, 0))
       colWeek.Resize(fyne.NewSize(colCycleW, h))
   }
   ```

2. **Hàm `matrixQuotaCell.update`:**
   Ray tiến trình cố định 40px x 6px. Duy nhất chỉ có vệt fill bar thay đổi chiều dài theo % quota (tụt dần):
   ```go
   func (c *matrixQuotaCell) update(fraction float64, hasData bool) {
       if c == nil {
           return
       }
       const barWidth float32 = 40.0 // Ray cố định 40px
   
       if !hasData || fraction < 0 {
           c.pctText.Text = "--%"
           c.pctText.Color = ColorQuotaMuted
           c.fillBar.FillColor = ColorQuotaMuted
           c.fillBar.Resize(fyne.NewSize(0, c.layout.barHeight))
       } else {
           clamped := fraction
           if clamped > 1.0 {
               clamped = 1.0
           } else if clamped < 0 {
               clamped = 0
           }
           tierColor := GetQuotaTierColor(fraction)
           c.pctText.Text = formatFraction(fraction)
           c.pctText.Color = tierColor
           
           // CHỈ CÓ fillBar THAY ĐỔI ĐỘ DÀI THEO TỶ LỆ QUOTA (TỤT DẦN)
           fw := barWidth * float32(clamped)
           c.fillBar.FillColor = tierColor
           c.fillBar.Resize(fyne.NewSize(fw, c.layout.barHeight))
       }
   
       c.fillBar.Refresh()
       c.pctText.Refresh()
   }
   ```

---

### TỆP 2: `internal/ui/window.go`
*Mục đích: Gỡ bỏ việc lưu đĩa trong Layout(), gỡ bỏ phức tạp.*

1. **Gỡ bỏ code gọi lưu đĩa trong `minSizeLayout.Layout`:**
   `Layout()` chỉ làm duy nhất việc định vị widget, TUYỆT ĐỐI KHÔNG gọi `OnWindowResized` hay lưu preferences trong hàm này:
   ```go
   func (m *minSizeLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
       for _, o := range objects {
           o.Resize(size)
           o.Move(fyne.NewPos(0, 0))
       }
   }
   ```
2. **MinSize tối thiểu an toàn:**
   `minSizeLayout` giữ ngưỡng tối thiểu nhỏ gọn `480 x 360 px` để người dùng thu nhỏ tự do mà không bị cản trở:
   ```go
   func (m *minSizeLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
       return m.min // 480x360 px
   }
   ```

---

### TỆP 3: `internal/ui/app.go`
*Mục đích: Chỉ User mới được thay đổi kích thước cửa sổ. Lưu kích thước khi đóng app, nạp khi mở app.*

1. **Gỡ bỏ các biến phức tạp trong struct `App`:**
   - Xóa bỏ: `resizeTimer`, `resizeMu`, `isUIReady`.
   - Giữ lại đơn giản: `windowWidth float32`, `windowHeight float32` để ghi nhận kích thước.
2. **Khởi tạo kích thước trong `Init()`:**
   - Đọc kích thước đã lưu từ preferences. Nếu có giá trị hợp lệ (`>= 480x360`), mở đúng kích thước đó. Nếu chưa có hoặc < 480x360, mở mặc định `1008 x 630 px`.
   - Cho phép người dùng tự do kéo chuột thay đổi kích thước:
     ```go
     a.win.Resize(fyne.NewSize(initW, initH))
     a.win.SetFixedSize(false) // User được phép tùy biến kích thước
     ```
3. **Lưu kích thước duy nhất khi đóng App trong `Close()`:**
   Khi ứng dụng đóng, lấy kích thước Canvas hiện tại mà người dùng đã kéo và lưu lại:
   ```go
   func (a *App) Close() {
       if a.autoLoop != nil {
           a.autoLoop.Stop()
       }
       if a.win != nil && a.win.Canvas() != nil {
           sz := a.win.Canvas().Size()
           if sz.Width >= MinWindowWidth && sz.Height >= MinWindowHeight {
               prefs, err := LoadUIPreferences(a.preferencesPath)
               if err != nil || prefs == nil {
                   prefs = &UIPreferences{Theme: a.currentTheme}
               }
               prefs.WindowWidth = sz.Width
               prefs.WindowHeight = sz.Height
               _ = SaveUIPreferences(a.preferencesPath, prefs)
           }
       }
   }
   ```
4. **Trong `updateUI` (khi Quota refresh 60s):**
   - Chỉ gọi `UpdateQuota` cập nhật trực tiếp tại chỗ cho các thẻ tài khoản.
   - **TUYỆT ĐỐI KHÔNG ĐỤNG VÀO KÍCH THƯỚC CỬA SỔ**, không gọi bất cứ lệnh resize nào lên cửa sổ tổng.

---

### TỆP 4: `accounts_store/.ui_preferences.json`
*Mục đích: Đặt lại file cấu hình sạch sẽ.*

```json
{
  "theme": "dark",
  "window_width": 1008,
  "window_height": 630
}
```

---

## 3. Khóa Phạm Vi Chỉnh Sửa (Touch-Map Khóa Cứng)

Coder TUYỆT ĐỐI chỉ thao tác trên đúng 5 file:

| STT | File | Nội dung thi công |
|---|---|---|
| 1 | `internal/ui/quota_badge.go` | Hardcode `colCycleW = 88.0 px`, ray bar 40px, chỉ fillBar co ngắn lại khi quota tụt |
| 2 | `internal/ui/window.go` | Gỡ bỏ toàn bộ code gọi lưu kích thước trong `minSizeLayout.Layout` |
| 3 | `internal/ui/app.go` | Gỡ bỏ timer resize, chỉ lưu kích thước cửa sổ trong `Close()`, giữ nguyên khi refresh |
| 4 | `accounts_store/.ui_preferences.json` | Reset về `1008 x 630 px` |
| 5 | `internal/ui/ui_test.go` | Cập nhật assertions khớp với cấu trúc hardcode cố định |

---

## 4. Tiêu Chuẩn Nghiệm Thu Kỹ Thuật (Acceptance Criteria)

1. **AC-01 (Hardcoded Fixed Matrix):** Bảng Quota có các chỉ số cột cố định 100% (`50px`, `88px`, `88px`, ray `40px`). Không có phép tính co giãn động nào tồn tại.
2. **AC-02 (Only Quota Minibar Decreases):** Khi Quota refresh tự động hay bấm nút, **CHỈ CÓ vệt màu `fillBar` thay đổi độ dài theo tỷ lệ % Quota** (tụt dần khi dùng). Tất cả chữ, thẻ, cột, khoảng cách giữ nguyên 100%.
3. **AC-03 (UI Tổng Bất Biến Khi Refresh):** Kích thước toàn bộ cửa sổ đứng yên tuyệt đối, không co giãn một pixel nào khi Quota refresh.
4. **AC-04 (User Resize Persistence On Close):** Người dùng được toàn quyền kéo chuột thay đổi kích thước cửa sổ. Kích thước này được ghi nhớ khi đóng app và phục hồi khi mở lại.
5. **AC-05 (Build & Tests):**
   - Chạy `go test -tags gui ./internal/ui/...` đạt **100% PASS**.
   - Chạy `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all` đạt **Exit code = 0**.
