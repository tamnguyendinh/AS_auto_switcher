# Báo Cáo Đặc Tả Kỹ Thuật: Gỡ Bỏ Toàn Bộ Cơ Chế Co Giãn, Khóa Chết UI Cố Định 100% Theo Tỷ Lệ Chuẩn (Pure Fixed UI - Zero Resizing)

- **Ngày ban hành:** 2026-10-19
- **Vai trò:** System Architect
- **Thẩm quyền:** Lệnh tối cao trực tiếp từ Human Owner
- **Mã văn kiện:** `SPEC-ARCH-20261019-PURE-FIXED-UI-ZERO-RESIZING`
- **Nguyên lý bất di bất dịch:** UI LÀ CỐ ĐỊNH 100%. ĐÃ THIẾT KẾ VỚI TỶ LỆ CHUẨN (1008 x 630 px) THÌ KHÔNG ĐƯỢC PHÉP CÓ BẤT CỨ THỨ GÌ CO GIÃN HOẶC THAY ĐỔI DÀI RỘNG. TẤT CẢ CÁC CƠ CHẾ GÂY CO GIÃN ĐỀU PHẢI GỠ BỎ HOÀN TOÀN.

---

## 1. Tinh Thần Chỉ Đạo Của Human Owner

1. **Bản chất của Tool:**
   - Ứng dụng đã được thiết kế với **tỷ lệ chuẩn tối ưu (1008 x 630 px)**.
   - Giao diện là **CỐ ĐỊNH HOÀN TOÀN (Fixed UI)**, không phải là trang web co giãn responsive.
   - Tuyệt đối KHÔNG ĐƯỢC PHÉP có bất cứ thứ gì co giãn hoặc thay đổi kích thước dài rộng (cả cửa sổ lẫn bên trong thẻ).
2. **Những thứ gây co giãn làm hỏng giao diện PHẢI GỠ BỎ TRIỆT ĐỂ:**
   - Gỡ bỏ `SetFixedSize(false)` $\rightarrow$ Chuyển thành `SetFixedSize(true)` (khóa cứng cửa sổ hệ điều hành ở mức 1008 x 630 px).
   - Gỡ bỏ cờ `WS_THICKFRAME` trong Win32 API (cấm người dùng dùng chuột kéo mép cửa sổ).
   - Gỡ bỏ nút phóng to/thu nhỏ `btnMax` trên Title bar (chỉ giữ nút thu nhỏ Taskbar `btnMin` và nút Đóng `btnClose`).
   - Gỡ bỏ toàn bộ struct `minSizeLayout`, hàm `OnWindowResized`, `saveWindowGeometry`, timer debounce và các biến `windowWidth`/`windowHeight`.
   - Gỡ bỏ `WindowWidth`, `WindowHeight` trong `UIPreferences`.
   - Gỡ bỏ toàn bộ công thức `remainW` tự động kéo dãn cột bảng Quota.
3. **Thanh Quota:**
   - Bảng Quota cố định 100%: Cột Model = 50px, Cột 5h = 88px, Cột Week = 88px.
   - Ray nền progress bar cố định 40px x 6px.
   - Khi Quota refresh: **Chỉ có vệt màu fill bên trong ray là tự động tụt dần/tăng dần theo % quota của Google API (`40px * fraction`)**, bản chất widget đứng im cố định, không kéo dãn bất cứ thứ gì xung quanh.

---

## 2. Danh Sách Gỡ Bỏ & Khóa Cứng Chi Tiết Cho Coder (Zero Speculation)

### TỆP 1: `internal/ui/app.go`
*Mục đích: Khóa cứng kích thước cửa sổ 1008x630 px; gỡ bỏ toàn bộ code đo đạc, lưu vết, timer resize.*

1. **Gỡ bỏ các trường thừa trong struct `App`:**
   - Xóa bỏ: `windowWidth`, `windowHeight`, `resizeTimer`, `resizeMu`, `isUIReady`.
2. **Gỡ bỏ các hàm resize thừa:**
   - Xóa bỏ hoàn toàn: `WindowSize()`, `OnWindowResized()`, `saveWindowGeometry()`, `SaveWindowGeometryNow()`.
3. **Sửa hàm `Init()`:**
   Khóa chết kích thước cửa sổ chuẩn `1008 x 630 px`:
   ```go
   a.win = a.app.NewWindow(windowTitle)
   a.win.SetIcon(AppLogoResource)
   
   // KHÓA CỨNG KÍCH THƯỚC CHUẨN 1008x630 PX
   a.win.Resize(fyne.NewSize(1008, 630))
   a.win.SetFixedSize(true) // BẬT CỜ FIXED SIZE KHÓA CHẾT CỬA SỔ
   
   a.setupLayout()
   a.setupTray()
   ```
4. **Sửa hàm `ShowAndRun()`:**
   Chạy đơn giản, sạch sẽ:
   ```go
   func (a *App) ShowAndRun() {
       if a.win == nil {
           a.Init()
       }
       defer a.Close()
   
       a.win.Show()
       MakeWindowFrameless(windowTitle)
   
       if a.alwaysOnTop {
           setWindowAlwaysOnTop(windowTitle, true)
       }
   
       a.app.Run()
   }
   ```
5. **Sửa hàm `Close()`:**
   Chỉ đóng `autoLoop`, không còn timer nào cần dọn dẹp.

---

### TỆP 2: `internal/ui/frameless_windows.go`
*Mục đích: Loại bỏ style co giãn viền cửa sổ của Windows (`WS_THICKFRAME`).*

1. **Trong `MakeWindowFrameless(windowTitle string)`:**
   Thay vì thêm `WS_THICKFRAME` (cho phép kéo mép resize), chúng ta **KHÔNG THÊM `WS_THICKFRAME`**:
   ```go
   style, _, _ := procGetWindowLongW.Call(hwnd, uintptr(gwlStyleVal))
   // Chỉ gỡ bỏ WS_CAPTION, TUYỆT ĐỐI KHÔNG thêm WS_THICKFRAME -> Cửa sổ cố định, không có viền co giãn
   newStyle := style &^ uintptr(WS_CAPTION)
   procSetWindowLongW.Call(hwnd, uintptr(gwlStyleVal), newStyle)
   procSetWindowPos.Call(hwnd, 0, 0, 0, 0, 0, uintptr(SWP_FRAMECHANGED|SWP_NOMOVE|SWP_NOSIZE|SWP_NOZORDER))
   ```
2. **Xóa hoặc vô hiệu hóa `ToggleMaximizeNativeWindow()`**: Cửa sổ cố định không cần phóng to toàn màn hình.

---

### TỆP 3: `internal/ui/window.go`
*Mục đích: Gỡ bỏ `minSizeLayout`, gỡ bỏ nút Phóng to `btnMax`, khóa cứng layout.*

1. **Xóa bỏ hoàn toàn struct `minSizeLayout`**:
   Không cần bất kỳ custom layout nào ép MinSize.
2. **Trong `setupLayout()`:**
   - **Gỡ bỏ nút Phóng to `btnMax`**: Cụm điều khiển cửa sổ chỉ gồm 2 nút: Thu nhỏ Taskbar (`btnMin`) và Đóng (`btnClose`):
     ```go
     btnMin := container.NewGridWrap(fyne.NewSize(32, 26), newCustomTitleBtn("—", func() {
         MinimizeNativeWindow()
     }))
     btnClose := container.NewGridWrap(fyne.NewSize(32, 26), newCustomCloseBtn(func() {
         a.Close()
         a.app.Quit()
     }))
     windowControls := container.NewHBox(btnMin, btnClose)
     ```
   - **Gán trực tiếp `mainLayout` vào Window Content** (không bọc qua `minSizeLayout`):
     ```go
     a.win.SetContent(mainLayout)
     ```

---

### TỆP 4: `internal/ui/quota_badge.go`
*Mục đích: Khóa cứng tuyệt đối kích thước ma trận Quota và ray tiến trình; chỉ vệt màu fill là thay đổi độ dài.*

1. **Trong `matrixTableLayout.Layout`:**
   Gỡ bỏ 100% các phép tính `remainW` hay `colCycleW` động. Khóa cứng cố định:
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
       col1W := float32(50.0)     // Model cố định 50px
       sepW := float32(1.0)      // Vách ngăn cố định 1px
       colGap := float32(4.0)    // Khoảng cách cố định 4px
       colCycleW := float32(88.0) // 5h và Week CỐ ĐỊNH 88px TUYỆT ĐỐI
   
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
2. **Trong `matrixQuotaCell.update`:**
   - Khung ray `bgTrack` cố định 40px x 6px.
   - Khi Quota tụt dần: Chỉ tính `fw = 40.0 * float32(clamped)`.
   - `c.fillBar.Resize(fyne.NewSize(fw, 6.0))` và `c.fillBar.Refresh()`.
   - `c.pctText.Refresh()`.
   - Tuyệt đối không gọi `c.container.Refresh()` hay bất cứ lệnh nào làm thay đổi kích thước container.

---

### TỆP 5: `internal/ui/themes.go` & `accounts_store/.ui_preferences.json`
*Mục đích: Trả `UIPreferences` về nguyên bản chỉ lưu Theme.*

1. **Struct trong `internal/ui/themes.go`:**
   ```go
   type UIPreferences struct {
       Theme string `json:"theme"`
   }
   ```
2. **File `accounts_store/.ui_preferences.json`:**
   ```json
   {
     "theme": "dark"
   }
   ```

---

## 3. Khóa Phạm Vi Chỉnh Sửa (Touch-Map Khóa Cứng)

Coder TUYỆT ĐỐI chỉ được phép sửa đúng 6 mục sau:

| STT | File | Hành động |
|---|---|---|
| 1 | `internal/ui/app.go` | `SetFixedSize(true)`, xóa toàn bộ biến/hàm resize |
| 2 | `internal/ui/frameless_windows.go` | Bỏ `WS_THICKFRAME`, cửa sổ không có viền kéo chuột |
| 3 | `internal/ui/window.go` | Bỏ `btnMax`, bỏ `minSizeLayout`, gán thẳng `mainLayout` |
| 4 | `internal/ui/quota_badge.go` | Khóa chết `colCycleW = 88.0 px`, chỉ thay đổi `fillBar` |
| 5 | `internal/ui/themes.go` & `.ui_preferences.json` | Khôi phục chỉ lưu `Theme`, xóa width/height |
| 6 | `internal/ui/ui_test.go` | Cập nhật assertions khớp với `FixedSize = true` và `1008x630` |

---

## 4. Tiêu Chuẩn Nghiệm Thu Kỹ Thuật (Acceptance Criteria)

1. **AC-01 (Fixed Window 100%):** Cửa sổ mở ra đúng chuẩn `1008 x 630 px`, `FixedSize == true`. Người dùng không thể kéo mép cửa sổ, không có nút phóng to.
2. **AC-02 (Zero Resizing on Quota Refresh):** Khi Quota refresh tự động 60s hay bấm nút thủ công, toàn bộ giao diện (cửa sổ, thẻ, bảng, cột) đứng im 100%, không rung giật, không co giãn 1 pixel nào.
3. **AC-03 (Only Quota Decreases):** Duy nhất thanh vệt màu `fillBar` bên trong ray 40px co ngắn lại khi quota tụt dần theo dữ liệu Google API.
4. **AC-04 (Build Discipline):**
   - Chạy `go test -tags gui ./internal/ui/...` đạt **100% PASS**.
   - Chạy `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all` đạt **Exit code = 0**.
