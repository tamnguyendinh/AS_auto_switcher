# Báo Cáo Đặc Tả Kỹ Thuật: Gỡ Bỏ Toàn Bộ Các Hàm Tạo Ra Sự Co Giãn Của UI & Khóa Chết Kích Thước Cố Định

- **Ngày ban hành:** 2026-10-19
- **Vai trò:** System Architect
- **Thẩm quyền:** Lệnh trực tiếp từ Human Owner
- **Mã văn kiện:** `SPEC-ARCH-20261019-PURGE-ALL-RESIZING-FUNCTIONS`
- **Nguyên lý thực hiện:** 
  1. UI TUYỆT ĐỐI KHÔNG ĐƯỢC PHÉP THAY ĐỔI KÍCH THƯỚC VÌ BẤT CỨ LÝ DO GÌ.
  2. BẮT ĐẦU BẰNG VIỆC GỠ BỎ TOÀN BỘ CÁC HÀM TẠO RA SỰ CO GIÃN CỦA UI.
  3. KHÓA CỨNG TOÀN BỘ KÍCH THƯỚC TỪ NGOÀI VÀO TRONG.
  4. CẬP NHẬT DỮ LIỆU TẠI CHỖ — DUY NHẤT VỆT MÀU `fillBar` TỤT DẦN THEO % QUOTA TRONG RAY 40PX.
  5. CẤM SỬA CHẮP VÁ, CẤM SUY DIỄN.

---

## PHẦN 1: GỠ BỎ TOÀN BỘ CÁC HÀM VÀ LỆNH TẠO RA SỰ CO GIÃN CỦA UI (DELETION PURGE)

Coder phải thực hiện GỠ BỎ dứt điểm toàn bộ các hàm, phương thức, thuộc tính và lệnh gây co giãn sau:

### 1.1. Trong tệp `internal/ui/frameless_windows.go`:
- ❌ **GỠ BỎ CỜ `WS_THICKFRAME`:**  
  Trong hàm `MakeWindowFrameless(windowTitle string)`:  
  Xóa hoàn toàn `| uintptr(WS_THICKFRAME)`.  
  Sửa thành:
  ```go
  newStyle := style &^ uintptr(WS_CAPTION | WS_THICKFRAME)
  procSetWindowLongW.Call(hwnd, uintptr(gwlStyleVal), newStyle)
  ```
  *(Gỡ bỏ viền kéo co giãn của Windows, khóa cứng cửa sổ ở mức hệ điều hành).*
- ❌ **GỠ BỎ HÀM `ToggleMaximizeNativeWindow()`:**  
  Xóa bỏ hoặc vô hiệu hóa hàm này. Cửa sổ cố định không có chức năng phóng to cực đại.

---

### 1.2. Trong tệp `internal/ui/window.go`:
- ❌ **GỠ BỎ NÚT PHÓNG TO `btnMax`:**  
  Trong hàm `setupLayout()`: Xóa bỏ nút `btnMax`. Cụm nút điều khiển chỉ còn 2 nút: Thu nhỏ Taskbar (`btnMin`) và Đóng (`btnClose`):
  ```go
  windowControls := container.NewHBox(btnMin, btnClose)
  ```
- ❌ **GỠ BỎ LỚP BỌC `NewVBox`:**  
  Trong hàm `setupLayout()`: Xóa `container.NewVBox(a.accountListContainer)`.  
  Gán trực tiếp:
  ```go
  scrollList := container.NewVScroll(container.NewPadded(a.accountListContainer))
  ```
- ❌ **GỠ BỎ LƯỚI TỰ ĐỘNG CO DÃN `container.NewGridWithColumns(3)`:**  
  Không dùng `GridWithColumns(3)` vì nó tự động kéo dãn bề rộng thẻ theo cửa sổ.  
  Thay bằng **`container.NewGridWrap(fyne.NewSize(318, 78))`** để cố định cứng kích thước mỗi thẻ Card là **318px $\times$ 78px**.

---

### 1.3. Trong tệp `internal/ui/app.go`:
- ❌ **GỠ BỎ LỆNH `a.win.SetFixedSize(false)`:**  
  Trong hàm `Init()`: Sửa thành **`a.win.SetFixedSize(true)`**.  
  *(Khi `SetFixedSize(true)`, Fyne GLFW driver khóa chết kích thước cửa sổ ở mức 1008x630, vô hiệu hóa hoàn toàn cơ chế tự động bung cửa sổ `w.fitContent()`)*.
- ❌ **GỠ BỎ LỆNH `a.accountListContainer.Objects = nil` TRONG `updateUI`:**  
  Trong hàm `updateUI`: Xóa bỏ việc hủy và tạo lại thẻ Card khi Quota refresh. Giữ nguyên 100% thẻ Card đã có.
- ❌ **GỠ BỎ TOÀN BỘ CÁC HÀM RESIZE ĐĨA VÀ TIMER:**  
  Xóa hoàn toàn nếu còn tồn tại: `resizeTimer`, `resizeMu`, `isUIReady`, `OnWindowResized()`, `saveWindowGeometry()`, `SaveWindowGeometryNow()`, `WindowSize()`.

---

### 1.4. Trong tệp `internal/ui/quota_badge.go`:
- ❌ **GỠ BỎ TOÀN BỘ PHÉP TÍNH CO GIÃN CỘT `remainW / 2.0`:**  
  Trong `matrixTableLayout.Layout`: Xóa bỏ hoàn toàn biến `remainW` và hàm `math.Min(remainW/2.0, 125.0)`.
- ❌ **GỠ BỎ LỆNH `c.container.Refresh()`:**  
  Trong hàm `matrixQuotaCell.update`: Xóa bỏ lệnh `c.container.Refresh()`.

---

## PHẦN 2: THIẾT LẬP KÍCH THƯỚC CỐ ĐỊNH TỪ NGOÀI VÀO TRONG (FIXED METRICS)

Sau khi đã gỡ bỏ sạch sẽ các hàm gây co giãn ở Phần 1, thiết lập kích thước cố định như sau:

### 2.1. Cửa Sổ Tổng (Window):
- Kích thước chuẩn cố định: **`1008 x 630 px`**.
- `a.win.Resize(fyne.NewSize(1008, 630))`
- `a.win.SetFixedSize(true)`

### 2.2. Khung Danh Sách Tài Khoản (`internal/ui/window.go`):
- Sử dụng `GridWrap` để khóa cứng kích thước từng thẻ Card:
  ```go
  // Mỗi thẻ Card CỐ ĐỊNH 318px rộng, 78px cao
  a.accountListContainer = container.NewGridWrap(fyne.NewSize(318, 78))
  scrollList := container.NewVScroll(container.NewPadded(a.accountListContainer))
  a.accountScroll = scrollList
  ```

### 2.3. Bảng Ma Trận Quota (`internal/ui/quota_badge.go`):
Toàn bộ bảng và các cột có số đo cố định:
- Cột Model: **`50.0 px`**
- Vách ngăn: **`1.0 px`**
- Khoảng cách gap: **`4.0 px`**
- Cột 5h: **`88.0 px`**
- Cột Week: **`88.0 px`**
- Ray tiến trình (`bgTrack`): **`40.0 px` $\times$ `6.0 px`**
- Bảng Quota: **`244.0 px` $\times$ `49.0 px`**

Mã nguồn trong `matrixTableLayout.Layout`:
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
	const col1W = float32(50.0)
	const sepW = float32(1.0)
	const colGap = float32(4.0)
	const colCycleW = float32(88.0)

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

---

## PHẦN 3: CẬP NHẬT DỮ LIỆU TẠI CHỖ — DUY NHẤT VỆT MÀU TỤT DẦN THEO % QUOTA

### 3.1. Cập nhật trong ô Quota (`matrixQuotaCell.update`):
Bản chất widget đứng im cố định. Duy nhất chiều dài của vệt màu `fillBar` co ngắn lại (tụt dần khi dùng):

```go
func (c *matrixQuotaCell) update(fraction float64, hasData bool) {
	if c == nil {
		return
	}
	const barWidth = float32(MatrixMiniBarWidth) // 40.0 px cố định

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

		// DUY NHẤT fillBar THAY ĐỔI ĐỘ DÀI THEO FRACTION (TỤT DẦN):
		fw := barWidth * float32(clamped)
		c.fillBar.FillColor = tierColor
		c.fillBar.Resize(fyne.NewSize(fw, c.layout.barHeight))
	}

	c.fillBar.Refresh()
	c.pctText.Refresh()
}
```

### 3.2. Cập nhật trong `updateUI` (`internal/ui/app.go`):
Khi có Quota mới, gọi trực tiếp `UpdateQuota` cho từng thẻ Card, tuyệt đối không tạo mới đối tượng:
```go
if canUpdateInPlace {
	for i, acc := range overview.Accounts {
		var quotaData *telemetry.AccountQuotaData
		if a.quotaCache != nil {
			quotaData, _ = a.quotaCache.Get(acc.Email)
		}
		a.accountCards[i].UpdateQuota(quotaData, palette)
	}
}
```

---

## 4. Khóa Phạm Vi Chỉnh Sửa (Touch-Map Khóa Cứng)

Coder TUYỆT ĐỐI chỉ thao tác trên đúng các file sau:

| STT | File | Hành động thi công |
|---|---|---|
| 1 | `internal/ui/frameless_windows.go` | Gỡ bỏ `WS_THICKFRAME`, cửa sổ cố định không có viền kéo chuột; gỡ bỏ `ToggleMaximizeNativeWindow` |
| 2 | `internal/ui/window.go` | Gỡ bỏ `btnMax`, gỡ bỏ `minSizeLayout`, thay `GridWithColumns(3)` bằng `GridWrap(318, 78)` |
| 3 | `internal/ui/app.go` | `SetFixedSize(true)`, xóa toàn bộ biến/hàm resize thừa, kích thước chuẩn `1008x630` |
| 4 | `internal/ui/quota_badge.go` | Khóa cứng `colCycleW = 88.0 px`, ray 40px; xóa bỏ `remainW/2.0` và `c.container.Refresh()` |
| 5 | `internal/ui/themes.go` | Trả `UIPreferences` về nguyên bản chỉ lưu `Theme` |
| 6 | `accounts_store/.ui_preferences.json` | Reset về `{"theme": "dark"}` |
| 7 | `internal/ui/ui_test.go` | Cập nhật assertions kiểm thử Fixed Metrics và Fixed Window |

---

## 5. Tiêu Chuẩn Nghiệm Thu Kỹ Thuật (Acceptance Criteria)

1. **AC-01 (Purged All Resizing Functions):** Toàn bộ các hàm và cờ gây co giãn (`WS_THICKFRAME`, `SetFixedSize(false)`, `btnMax`, `remainW/2.0`, `minSizeLayout`, `Objects = nil`) bị gỡ bỏ hoàn toàn khỏi mã nguồn.
2. **AC-02 (Fixed Window 100%):** Cửa sổ mở ra ở kích thước chuẩn `1008 x 630 px`, `FixedSize == true`. Không có viền kéo co giãn của Windows, không có nút phóng to. Cửa sổ không bao giờ bị thay đổi kích thước vì bất cứ lý do gì.
3. **AC-03 (Fixed Internal Metrics):** Thẻ Card cố định `318 x 78 px`, Bảng Quota cố định `244 x 49 px`, các cột Model 50px, 5h 88px, Week 88px, ray 40px cố định 100%.
4. **AC-04 (Only Quota Decreases):** Khi Quota refresh, duy nhất vệt màu `fillBar` co ngắn lại theo % quota tiêu hao. Toàn bộ giao diện đứng im 100%.
5. **AC-05 (Build & Tests):**
   - `go test -tags gui ./internal/ui/...` đạt **100% PASS**.
   - `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all` đạt **Exit code = 0**.
