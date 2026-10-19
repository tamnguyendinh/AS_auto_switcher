# Báo Cáo Đặc Tả Kỹ Thuật: Khóa Cứng UI Tổng, Cố Định Bố Cục Thẻ & Chỉ Cho Phép Thanh Minibar Co Giãn

- **Ngày ban hành:** 2026-10-19
- **Vai trò:** System Architect
- **Thẩm quyền:** Lệnh trực tiếp từ Human Owner
- **Mã văn kiện:** `SPEC-ARCH-20261019-FIX-UI-STRETCH-AND-FIXED-LAYOUT`
- **Mục tiêu:** Khóa cứng toàn bộ UI tổng và cấu trúc thẻ; khi Quota refresh, CHỈ DUY NHẤT thanh minibar (fill bar) được co giãn theo %, cấm mọi thành phần khác co giãn. Coder thi công chính xác 100% theo mã mẫu, TUYỆT ĐỐI KHÔNG SUY DIỄN.

---

## 1. Yêu Cầu Tối Cao Của Human Owner (Core Directives)

1. **UI tổng (kích thước cửa sổ Windows và cấu trúc khung app):**
   - Tuyệt đối KHÔNG ĐƯỢC THAY ĐỔI khi Quota refresh tự động (mỗi 60s) hoặc làm mới thủ công.
   - Kích thước cửa sổ chỉ thay đổi khi người dùng tự tay dùng chuột kéo mép cửa sổ. Khi người dùng thay đổi, hệ thống phải lưu lại bền vững vào `accounts_store/.ui_preferences.json`.
   - CẤM lưu kích thước giả mạo lúc khởi động ứng dụng (loại bỏ hoàn toàn lỗi lưu đè `490 x 387 px`).
2. **Hành vi bên trong thẻ Card khi Quota refresh:**
   - **CHỈ CÓ THANH MINIBAR (`fillBar`) ĐƯỢC PHÉP CO GIÃN ĐỘ DÀI THEO % QUOTA.**
   - Nhãn phần trăm (`pctText`) cập nhật chuỗi số (ví dụ: `100.0%`, `50.0%`, `--%`).
   - **TẤT CẢ CÁC THỨ KHÁC PHẢI CỐ ĐỊNH 100%, KHÔNG ĐƯỢC THAY ĐỔI THEO:**
     + Bề rộng cột `5h` KHÓA CỨNG `88.0 px`.
     + Bề rộng cột `Week` KHÓA CỨNG `88.0 px`.
     + Bề rộng cột Model (Gemini/Claude) KHÓA CỨNG `50.0 px`.
     + Ray nền thanh bar (`bgTrack`) KHÓA CỨNG `40.0 px`.
     + Bảng ma trận Quota và toàn bộ thẻ Card không được xê dịch, không co giãn, không giật khung hình.

---

## 2. Hướng Dẫn Thi Công Chi Tiết Từng File Cho Coder (Zero Speculation Code)

### TỆP 1: `internal/ui/quota_badge.go`
*Mục đích: Khóa cứng 100% kích thước các cột ma trận Quota, chỉ cho phép `fillBar` co giãn theo %.*

#### 1.1. Sửa hàm `matrixTableLayout.Layout` (dòng ~609):
**XÓA BỎ HOÀN TOÀN** đoạn code tính `remainW` và `colCycleW = math.Min(remainW/2.0, 125.0)`.  
Thay thế bằng code khóa cứng cố định:

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
	col1W := float32(50.0)    // Cột Model: CỐ ĐỊNH 50px
	sepW := float32(1.0)     // Vách ngăn: CỐ ĐỊNH 1px
	colGap := float32(4.0)   // Khoảng cách: CỐ ĐỊNH 4px
	colCycleW := float32(88.0) // Cột 5h và Week: KHÓA CỨNG 88px TUYỆT ĐỐI, CẤM CO GIÃN ĐỘNG!

	x := float32(0)
	// Col 1: Model
	colModel.Move(fyne.NewPos(x, 0))
	colModel.Resize(fyne.NewSize(col1W, h))
	x += col1W + colGap

	// Sep 1
	sep1.Move(fyne.NewPos(x, 1))
	sep1.Resize(fyne.NewSize(sepW, h-2))
	x += sepW + colGap

	// Col 2: 5h
	col5h.Move(fyne.NewPos(x, 0))
	col5h.Resize(fyne.NewSize(colCycleW, h))
	x += colCycleW + colGap

	// Sep 2
	sep2.Move(fyne.NewPos(x, 1))
	sep2.Resize(fyne.NewSize(sepW, h-2))
	x += sepW + colGap

	// Col 3: Week
	colWeek.Move(fyne.NewPos(x, 0))
	colWeek.Resize(fyne.NewSize(colCycleW, h))
}
```

#### 1.2. Sửa hàm `matrixQuotaCell.update` (dòng ~500):
Đảm bảo **CHỈ CÓ `fillBar` co giãn chiều dài**, không gọi `Refresh()` lên toàn bộ container làm rung giật layout:

```go
func (c *matrixQuotaCell) update(fraction float64, hasData bool) {
	if c == nil {
		return
	}
	barWidth := float32(MatrixMiniBarWidth) // 40.0 px cố định
	if barWidth <= 0 {
		barWidth = 40.0
	}

	if !hasData || fraction < 0 {
		c.pctText.Text = "--%"
		c.pctText.Color = ColorQuotaMuted
		c.layout.fraction = -1
		c.layout.fillWidth = 0
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
		c.layout.fraction = fraction
		
		// DUY NHẤT fillWidth CO GIÃN THEO FRACTION
		fw := barWidth * float32(clamped)
		c.layout.fillWidth = fw
		c.fillBar.FillColor = tierColor
		c.fillBar.Resize(fyne.NewSize(fw, c.layout.barHeight))
	}

	c.fillBar.Refresh()
	c.pctText.Refresh()
	// KHÔNG GỌI c.container.Refresh() để tránh kích hoạt layout cascade của Fyne
}
```

---

### TỆP 2: `internal/ui/app.go`
*Mục đích: Ngăn chặn ghi đè kích thước giả mạo lúc khởi động; đảm bảo UI tổng không bị thay đổi.*

#### 2.1. Thêm cờ `isUIReady atomic.Bool` vào struct `App`:
```go
type App struct {
    ...
    isUIReady    atomic.Bool // Chỉ = true sau khi cửa sổ đã Show() xong hoàn toàn
    windowWidth  float32
    windowHeight float32
    resizeTimer  *time.Timer
    resizeMu     sync.Mutex
    accountCards []*AccountCardWidget
}
```

#### 2.2. Kiểm tra cờ `isUIReady` trong `OnWindowResized`:
```go
func (a *App) OnWindowResized(w, h float32) {
	// CẤM GHI KÍCH THƯỚC KHI APP ĐANG KHỞI TẠO HOẶC CHƯA READY
	if !a.isUIReady.Load() {
		return
	}
	if w < MinWindowWidth || h < MinWindowHeight {
		return
	}

	a.resizeMu.Lock()
	defer a.resizeMu.Unlock()

	if a.windowWidth == w && a.windowHeight == h {
		return
	}
	a.windowWidth = w
	a.windowHeight = h

	if a.resizeTimer != nil {
		a.resizeTimer.Stop()
	}

	a.resizeTimer = time.AfterFunc(400*time.Millisecond, func() {
		a.saveWindowGeometry(w, h)
	})
}
```

#### 2.3. Bật cờ `isUIReady = true` trong `ShowAndRun()`:
```go
func (a *App) ShowAndRun() {
	if a.win == nil {
		a.Init()
	}
	defer a.Close()

	// 1. Ép GLFW tạo HWND và hiển thị cửa sổ lên Windows
	a.win.Show()

	// 2. Gỡ bỏ thanh Windows đồng bộ
	MakeWindowFrameless(windowTitle)

	if a.alwaysOnTop {
		setWindowAlwaysOnTop(windowTitle, true)
	}

	// 3. Đánh dấu UI đã sẵn sàng sau khi mọi pha layout khởi tạo hoàn tất
	// Chờ 300ms để bỏ qua các xung resize ban đầu của OS
	time.AfterFunc(300*time.Millisecond, func() {
		a.isUIReady.Store(true)
	})

	// 4. Chạy vòng lặp nhận sự kiện của Fyne
	a.app.Run()
}
```

#### 2.4. Khôi phục kích thước chuẩn mặc định trong `Init()`:
Nếu file preferences chưa có kích thước hoặc kích thước nhỏ hơn `800x500` (do bị ghi đè lỗi trước đó), khôi phục ngay kích thước chuẩn mặc định `1008 x 630`:
```go
initW := DefaultWindowWidth  // 1008.0
initH := DefaultWindowHeight // 630.0

a.resizeMu.Lock()
if a.windowWidth >= 800.0 && a.windowHeight >= 500.0 {
    initW = a.windowWidth
    initH = a.windowHeight
} else {
    a.windowWidth = initW
    a.windowHeight = initH
}
a.resizeMu.Unlock()

a.win.Resize(fyne.NewSize(initW, initH))
```

---

### TỆP 3: `internal/ui/window.go`
*Mục đích: Đảm bảo layout gốc không kích hoạt lưu kích thước khi chưa ready.*

Trong `minSizeLayout.Layout`:
```go
func (m *minSizeLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objects {
		o.Resize(size)
		o.Move(fyne.NewPos(0, 0))
	}
	if m.app != nil && m.app.isUIReady.Load() {
		actualW := size.Width
		actualH := size.Height
		if m.app.win != nil && m.app.win.Canvas() != nil {
			cs := m.app.win.Canvas().Size()
			if cs.Width > 0 && cs.Height > 0 {
				actualW = cs.Width
				actualH = cs.Height
			}
		}
		if actualW >= MinWindowWidth && actualH >= MinWindowHeight {
			m.app.OnWindowResized(actualW, actualH)
		}
	}
}
```

---

### TỆP 4: `accounts_store/.ui_preferences.json`
*Mục đích: Reset file cấu hình bị ghi đè kích thước lỗi trước đó.*

Sửa lại nội dung file:
```json
{
  "theme": "dark",
  "window_width": 1008,
  "window_height": 630
}
```

---

## 3. Khóa Phạm Vi Chỉnh Sửa (Touch-Map Khóa Cứng)

Coder TUYỆT ĐỐI chỉ được phép chỉnh sửa các file sau:

| STT | File | Mục tiêu |
|---|---|---|
| 1 | `internal/ui/quota_badge.go` | Khóa cứng `colCycleW = 88.0`, chỉ co giãn `fillBar` |
| 2 | `internal/ui/app.go` | Bổ sung `isUIReady`, chặn lưu kích thước giả mạo, nạp chuẩn `1008x630` |
| 3 | `internal/ui/window.go` | Bọc điều kiện `m.app.isUIReady.Load()` trong `minSizeLayout.Layout` |
| 4 | `accounts_store/.ui_preferences.json` | Reset về kích thước mặc định `1008 x 630` |
| 5 | `internal/ui/ui_test.go` | Cập nhật assertions khớp với `colCycleW = 88.0` và cờ `isUIReady` |

---

## 4. Tiêu Chuẩn Nghiệm Thu Kỹ Thuật (Acceptance Criteria)

1. **AC-01 (Fixed Table Layout):** Bảng Quota có chiều rộng và vị trí cột cố định 100% (`col1 = 50px`, `col5h = 88px`, `colWeek = 88px`). Tuyệt đối không co giãn cột khi Quota thay đổi.
2. **AC-02 (Only Minibar Changes):** Khi Quota refresh, **CHỈ CÓ thanh `fillBar` bên trong ray 40px thay đổi chiều dài**. Không có bất kỳ phần tử nào khác giật hoặc giãn nở.
3. **AC-03 (UI Tổng Bất Biến):** Kích thước toàn bộ cửa sổ (UI tổng) đứng yên 100% trong mọi chu kỳ Quota refresh tự động 60s.
4. **AC-04 (True User Resize Persistence):** Chỉ khi người dùng thực sự kéo chuột trên màn hình thì kích thước mới được lưu vào `.ui_preferences.json`. Khi mở lại app, cửa sổ hiển thị đúng kích thước đã lưu.
5. **AC-05 (Build & Tests):**
   - `go test -tags gui ./internal/ui/...` đạt **100% PASS**.
   - `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all` đạt **Exit code = 0**.
