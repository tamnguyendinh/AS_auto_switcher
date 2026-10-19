# Báo Cáo Đặc Tả Kỹ Thuật Chuẩn Ngành: Gỡ Bỏ Toàn Bộ Thành Phần Co Giãn, Khóa Cứng Chỉ Số Cố Định (Zero Resizing - Fixed Metric Layout)

- **Ngày ban hành:** 2026-10-19
- **Vai trò:** System Architect
- **Thẩm quyền:** Lệnh tối cao trực tiếp từ Human Owner
- **Mã văn kiện:** `SPEC-ARCH-20261019-INDUSTRY-STANDARD-FIXED-UI`
- **Trạng thái:** APPROVED BY OWNER — CHỈ THỊ THI CÔNG TUYỆT ĐỐI (CODER THỰC HIỆN 100% THEO MÃ MẪU, CẤM SUY DIỄN)

---

## 1. Triết Lý & Nguyên Tắc Tối Thượng (Supreme Iron Rules)

1. **Chuẩn ngành Desktop (Antigravity IDE / VS Code):**
   - Toàn bộ các phần tử giao diện (từ thanh bar, bảng, cột, thẻ Card) đều tuân theo **hệ thống số đo cố định tuyệt đối (Fixed Metric System)**.
   - Khi dữ liệu Quota làm mới tự động (chu kỳ 60s hay bấm nút): **TUYỆT ĐỐI KHÔNG CÓ BẤT CỨ THỨ GÌ CO GIÃN HOẶC THAY ĐỔI DÀI RỘNG**.
   - Cửa sổ là khung nhìn độc lập (Viewport): Người dùng có toàn quyền kéo chuột chỉnh to nhỏ theo ý muốn, hệ thống không được tự ý can thiệp.
2. **Hành vi DUY NHẤT khi Quota làm mới:**
   - **CHỈ CÓ vệt màu tiến trình `fillBar` bên trong ray 40px thay đổi chiều dài theo % quota (tụt dần khi tiêu hao):**  
     $$\text{fillWidth} = 40.0\text{ px} \times \text{fraction}$$
   - Nhãn `%` cập nhật text (ví dụ `100.0%`, `50.0%`, `--%`).
   - **TẤT CẢ NHỮNG THỨ KHÁC ĐỨNG IM 100%, KHÔNG CO GIÃN THEO.**
3. **DANH SÁCH GỠ BỎ TOÀN BỘ (XÓA SẠCH SẼ):**
   - Xóa bỏ struct `minSizeLayout` và hook `OnWindowResized`.
   - Xóa bỏ toàn bộ biến timer resize (`resizeTimer`, `resizeMu`, `isUIReady`) trong `App`.
   - Xóa bỏ toàn bộ thuật toán tính toán co giãn cột động `remainW / 2.0` trong `matrixTableLayout`.
   - Xóa bỏ lệnh `c.container.Refresh()` trong `matrixQuotaCell.update`.
   - Xóa bỏ lớp bọc `NewVBox` bên trong `scrollList`.
   - Xóa bỏ `WindowWidth`, `WindowHeight` trong `UIPreferences`.

---

## 2. Chỉ Thị Thi Công Chi Tiết Từng File Cho Coder (Zero Speculation Code)

---

### TỆP 1: `internal/ui/quota_badge.go`

#### 1.1. Sửa hàm `matrixTableLayout.Layout` (Khóa cứng kích thước cột):
**Xóa sạch** đoạn tính `remainW` và `remainW / 2.0`. Thay bằng mã cố định 100%:

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
	// CHỈ SỐ CỐ ĐỊNH CHUẨN NGÀNH - TUYỆT ĐỐI KHÔNG CO GIÃN
	const col1W = float32(50.0)     // Cột Model: CỐ ĐỊNH 50px
	const sepW = float32(1.0)      // Vách ngăn đứng: CỐ ĐỊNH 1px
	const colGap = float32(4.0)    // Khoảng cách giữa các cột: CỐ ĐỊNH 4px
	const colCycleW = float32(88.0) // Cột 5h và Week: CỐ ĐỊNH 88px

	x := float32(0)
	// Col 1: Model (Gemini / Claude)
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

#### 1.2. Sửa hàm `matrixTableLayout.MinSize`:
Trả về bề rộng cố định `244 px` và chiều cao `49 px`:

```go
func (l *matrixTableLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	const totalW = float32(50.0 + 1.0 + 88.0 + 1.0 + 88.0 + 16.0) // 244.0 px
	return fyne.NewSize(totalW, 49.0)
}
```

#### 1.3. Sửa hàm `matrixQuotaCell.update` (Xóa bỏ `c.container.Refresh()`):
Chỉ cập nhật text và vệt fill màu của thanh tiến trình (tụt dần), **XÓA BỎ `c.container.Refresh()`**:

```go
func (c *matrixQuotaCell) update(fraction float64, hasData bool) {
	if c == nil {
		return
	}
	const barWidth = float32(MatrixMiniBarWidth) // Ray tiến trình CỐ ĐỊNH 40.0 px

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

		// CHỈ DUY NHẤT fillBar THAY ĐỔI CHIỀU DÀI THEO % QUOTA (TỤT DẦN)
		fw := barWidth * float32(clamped)
		c.fillBar.FillColor = tierColor
		c.fillBar.Resize(fyne.NewSize(fw, c.layout.barHeight))
	}

	c.fillBar.Refresh()
	c.pctText.Refresh()
	// TUYỆT ĐỐI KHÔNG GỌI c.container.Refresh()
}
```

---

### TỆP 2: `internal/ui/window.go`

#### 2.1. Xóa bỏ hoàn toàn struct `minSizeLayout` và các hàm của nó:
Xóa sạch từ dòng định nghĩa `type minSizeLayout struct` cho tới hết hàm `Layout()` của nó.

#### 2.2. Sửa hàm `setupLayout()`:
1. Gỡ bỏ lớp bọc `NewVBox` gây cascade chiều cao trong `scrollList`:
   ```go
   // Thay vì NewVScroll(NewPadded(NewVBox(a.accountListContainer)))
   // Chuẩn ngành: Gán trực tiếp lưới tài khoản vào VScroll có đệm:
   a.accountListContainer = container.NewGridWithColumns(3)
   scrollList := container.NewVScroll(container.NewPadded(a.accountListContainer))
   a.accountScroll = scrollList
   ```
2. Gán thẳng `mainLayout` vào Window Content (không bọc qua `minSizeLayout`):
   ```go
   a.win.SetContent(mainLayout)
   ```

---

### TỆP 3: `internal/ui/app.go`

#### 3.1. Dọn sạch struct `App`:
Xóa bỏ các trường thừa: `windowWidth`, `windowHeight`, `resizeTimer`, `resizeMu`, `isUIReady`.  
Chỉ giữ lại con trỏ `accountCards []*AccountCardWidget` để phục vụ in-place binding.

#### 3.2. Xóa bỏ các hàm resize thừa:
Xóa hoàn toàn các hàm:
- `WindowSize()`
- `OnWindowResized()`
- `saveWindowGeometry()`
- `SaveWindowGeometryNow()`

#### 3.3. Sửa hàm `Init()`:
Mở cửa sổ theo tỷ lệ chuẩn `1008 x 630 px`. Cho phép người dùng tự do kéo chuột chỉnh to nhỏ (`SetFixedSize(false)`), code tuyệt đối không can thiệp:

```go
func (a *App) Init() {
	a.app.Settings().SetTheme(NewThemeInstance(a.currentTheme))
	a.app.SetIcon(AppLogoResource)

	a.win = a.app.NewWindow(windowTitle)
	a.win.SetIcon(AppLogoResource)
	
	// Khởi tạo kích thước chuẩn 1008x630 px
	a.win.Resize(fyne.NewSize(1008, 630))
	a.win.SetFixedSize(false) // User được tự do chỉnh kích thước cửa sổ nếu muốn

	a.setupLayout()
	a.setupTray()

	if a.alwaysOnTop {
		setWindowAlwaysOnTop(windowTitle, true)
	}

	// 60s background auto-telemetry loop
	if a.autoLoop != nil {
		a.autoLoop.SetUpdateCallback(func(results []*telemetry.AccountQuotaData) {
			fyne.Do(func() {
				a.mu.Lock()
				a.lastQuotaRefresh = time.Now()
				a.mu.Unlock()

				a.mu.RLock()
				overview := a.overview
				a.mu.RUnlock()
				if overview != nil {
					a.updateUI(overview)
				}

				nowStr := time.Now().Format("15:04:05")
				if a.lastUpdatedLabel != nil {
					a.lastUpdatedLabel.SetText(fmt.Sprintf("Cập nhật : %s", nowStr))
				}
				if a.autoRefreshLabel != nil {
					a.autoRefreshLabel.SetText(fmt.Sprintf("Cập nhật : %s", nowStr))
				}
			})
		})
		_ = a.autoLoop.Start(context.Background())
	}

	a.Refresh()
}
```

#### 3.4. Sửa hàm `Close()`:
Chỉ dừng vòng lặp nền `autoLoop`:
```go
func (a *App) Close() {
	if a.autoLoop != nil {
		a.autoLoop.Stop()
	}
}
```

#### 3.5. Hàm `updateUI()`:
Giữ nguyên logic In-Place Binding: khi danh sách tài khoản không đổi, chỉ gọi `a.accountCards[i].UpdateQuota(...)`. **Tuyệt đối không đụng vào kích thước cửa sổ.**

---

### TỆP 4: `internal/ui/themes.go` & `accounts_store/.ui_preferences.json`

#### 4.1. Trong `internal/ui/themes.go`:
Gỡ bỏ các trường kích thước trong struct:
```go
type UIPreferences struct {
	Theme string `json:"theme"`
}
```

#### 4.2. File `accounts_store/.ui_preferences.json`:
Làm sạch file cấu hình:
```json
{
  "theme": "dark"
}
```

---

### TỆP 5: `internal/ui/ui_test.go`

1. Xóa các test case cũ gọi `OnWindowResized`, `WindowSize`, `SaveWindowGeometryNow`, `isUIReady`.
2. Bổ sung/cập nhật test case khẳng định chuẩn ngành:
   - `TestIndustryStandard_FixedMatrixMetrics`: Đo đạc cột Model luôn đúng `50px`, cột 5h đúng `88px`, cột Week đúng `88px`, ray bar đúng `40px` dù kích thước cha thay đổi.
   - `TestIndustryStandard_OnlyMinibarDecreasesOnQuotaUpdate`: Chứng minh khi quota tụt từ 100% về 50% rồi về 0%, toàn bộ thẻ Card và các cột đứng yên 100%, chỉ duy nhất `fillBar.Size().Width` thay đổi từ 40px -> 20px -> 0px.
   - `TestIndustryStandard_WindowInit1008x630`: Cửa sổ mở ra đạt chuẩn kích thước `1008 x 630 px`.

---

## 3. Khóa Phạm Vi Chỉnh Sửa (Touch-Map Khóa Cứng)

Coder TUYỆT ĐỐI chỉ được phép chỉnh sửa các file sau:

| STT | File | Hành động thi công |
|---|---|---|
| 1 | `internal/ui/quota_badge.go` | Hardcode fixed metrics `50px`, `88px`, `88px`, ray `40px`; xóa bỏ `remainW/2.0` và `c.container.Refresh()` |
| 2 | `internal/ui/window.go` | Xóa bỏ struct `minSizeLayout`; bỏ bọc `NewVBox` trong `scrollList`; gán thẳng `mainLayout` |
| 3 | `internal/ui/app.go` | Xóa bỏ timer, biến resize và các hàm resize thừa; khởi tạo chuẩn `1008x630` |
| 4 | `internal/ui/themes.go` | Trả `UIPreferences` về nguyên bản chỉ lưu `Theme` |
| 5 | `accounts_store/.ui_preferences.json` | Reset sạch sẽ `{"theme": "dark"}` |
| 6 | `internal/ui/ui_test.go` | Cập nhật assertions khớp với Fixed Metrics và kiểm thử tụt minibar |

---

## 4. Tiêu Chuẩn Nghiệm Thu Kỹ Thuật (Acceptance Criteria)

1. **AC-01 (Fixed Column Metrics 100%):** Bảng Quota có các chỉ số cột cố định bất biến: Model = 50px, 5h = 88px, Week = 88px, ray = 40px. Không có bất kỳ dòng mã co giãn động nào.
2. **AC-02 (Only Minibar Decreases):** Khi Quota refresh tự động 60s hay bấm nút thủ công, **CHỈ CÓ vệt màu `fillBar` co ngắn lại theo % Quota** (tụt dần khi dùng). Tất cả chữ, thẻ, cột, khoảng cách, bảng giữ nguyên 100%.
3. **AC-03 (UI Tổng Bất Biến Khi Refresh):** Kích thước toàn bộ cửa sổ đứng im tuyệt đối, không co giãn một pixel nào khi Quota refresh.
4. **AC-04 (Build Discipline):**
   - Chạy `go test -tags gui ./internal/ui/...` đạt **100% PASS**.
   - Chạy `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all` đạt **Exit code = 0**.
