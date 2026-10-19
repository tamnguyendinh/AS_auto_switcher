# Báo Cáo Đặc Tả Kỹ Thuật Tổng Thể: Gỡ Bỏ Toàn Bộ Các Hàm Tạo Ra Sự Co Giãn Của UI & Thiết Lập Tỷ Lệ Cố Định Chuẩn Ngành

- **Ngày ban hành:** 2026-10-19
- **Vai trò:** System Architect
- **Thẩm quyền:** Lệnh trực tiếp từ Human Owner
- **Mã văn kiện:** `SPEC-ARCH-20261019-MASTER-ELIMINATE-ALL-UI-RESIZING`
- **Trạng thái:** APPROVED BY OWNER — CHỈ THỊ THI CÔNG BẮT BUỘC (CODER THỰC HIỆN ĐÚNG 3 GIAI ĐOẠN, TUYỆT ĐỐI CẤM SỬA CHẮP VÁ)

---

## 1. Nguyên Lý Chỉ Đạo Tối Cao Của Human Owner

1. **Bất biến tối cao:** UI TUYỆT ĐỐI KHÔNG ĐƯỢC PHÉP THAY ĐỔI KÍCH THƯỚC VÌ BẤT CỨ LÝ DO GÌ.
2. **Phương pháp thực hiện bắt buộc:**
   - **GIAI ĐOẠN 1:** GỠ BỎ TOÀN BỘ CÁC HÀM, PHƯƠNG THỨC, STYLE VÀ LỆNH TẠO RA SỰ CO GIÃN CỦA UI.
   - **GIAI ĐOẠN 2:** THIẾT LẬP TỶ LỆ VÀ KÍCH THƯỚC CỐ ĐỊNH CHUẨN NGÀNH CHO TOÀN BỘ GIAO DIỆN (TỪ NGOÀI VÀO TRONG).
   - **GIAI ĐOẠN 3:** CẬP NHẬT DỮ LIỆU TẠI CHỖ — DUY NHẤT VỆT MÀU `fillBar` TỤT DẦN THEO % QUOTA TRONG RAY 40PX, CẤM MỌI THỨ KHÁC CO GIÃN.
3. **Cấm tuyệt đối:** Cấm sửa chắp vá "chỗ này một chút, chỗ kia một chút". Phải thanh trừng triệt để toàn bộ các hàm gây co giãn trước tiên.

---

## GIAI ĐOẠN 1: GỠ BỎ TOÀN BỘ CÁC HÀM & CƠ CHẾ TẠO RA SỰ CO GIÃN CỦA UI (DELETION PURGE)

Coder phải thực hiện xóa bỏ toàn bộ các thành phần gây co giãn sau đây:

### 1.1. Trong `internal/ui/app.go`:
- ❌ **GỠ BỎ LỆNH CHO PHÉP CO GIÃN CỬA SỔ:**  
  Thay thế `a.win.SetFixedSize(false)` bằng **`a.win.SetFixedSize(true)`**.  
  *Lý do:* Khi `SetFixedSize(true)`, Fyne kích hoạt `view.SetSizeLimits(1008, 630, 1008, 630)`, hệ điều hành Windows KHÓA CHẾT kích thước cửa sổ ở đúng `1008 x 630 px`, triệt tiêu vĩnh viễn việc GLFW tự động bung cửa sổ (`w.fitContent()`).
- ❌ **GỠ BỎ TOÀN BỘ BIẾN & HÀM LIÊN QUAN ĐẾN RESIZE:**  
  Xóa hoàn toàn nếu còn tồn tại: `windowWidth`, `windowHeight`, `resizeTimer`, `resizeMu`, `isUIReady`, `OnWindowResized()`, `saveWindowGeometry()`, `SaveWindowGeometryNow()`.
- ❌ **GỠ BỎ LỆNH XÓA TRẮNG CANVAS:**  
  Trong `updateUI`: Tuyệt đối cấm gọi `a.accountListContainer.Objects = nil`. Khi danh sách tài khoản đã có, 100% các lần Quota refresh đều đi vào luồng In-Place update.

### 1.2. Trong `internal/ui/frameless_windows.go`:
- ❌ **GỠ BỎ CỜ CO GIÃN VIỀN CỬA SỔ `WS_THICKFRAME` CỦA WINDOWS:**  
  Trong hàm `MakeWindowFrameless`:  
  Thay vì: `newStyle := (style &^ uintptr(WS_CAPTION)) | uintptr(WS_THICKFRAME)`  
  Sửa thành: **`newStyle := style &^ uintptr(WS_CAPTION | WS_THICKFRAME)`**  
  *Lý do:* `WS_THICKFRAME` là style Win32 tạo viền kéo co giãn cửa sổ. Gỡ bỏ `WS_THICKFRAME` thì Windows không cho phép kéo mép cửa sổ nữa, khóa chết kích thước.
- ❌ **GỠ BỎ HÀM PHÓNG TO `ToggleMaximizeNativeWindow()`**: Cửa sổ cố định không cho phép phóng to toàn màn hình.

### 1.3. Trong `internal/ui/window.go`:
- ❌ **GỠ BỎ NÚT PHÓNG TO `btnMax` ("口") TRÊN TITLE BAR:**  
  Cụm điều khiển cửa sổ chỉ gồm 2 nút: Thu nhỏ Taskbar (`btnMin`) và Đóng (`btnClose`). Xóa hoàn toàn `btnMax`.
- ❌ **GỠ BỎ HOÀN TOÀN struct `minSizeLayout` VÀ CÁC PHƯƠNG THỨC CỦA NÓ:**  
  Xóa sạch struct này. Cửa sổ gán trực tiếp: `a.win.SetContent(mainLayout)`.
- ❌ **GỠ BỎ LỚP BỌC `NewVBox` GÂY CASCADE CHIỀU CAO:**  
  Trong `setupLayout()`: Xóa `container.NewVBox(a.accountListContainer)`.  
  Gán trực tiếp: `scrollList := container.NewVScroll(container.NewPadded(a.accountListContainer))`.

### 1.4. Trong `internal/ui/quota_badge.go`:
- ❌ **GỠ BỎ TOÀN BỘ CÔNG THỨC CO GIÃN CỘT `remainW / 2.0`:**  
  Trong `matrixTableLayout.Layout`: Xóa sạch biến `remainW` và hàm `math.Min(remainW/2.0, 125.0)`.
- ❌ **GỠ BỎ LỆNH `c.container.Refresh()`:**  
  Trong `matrixQuotaCell.update`: Xóa bỏ lệnh `c.container.Refresh()` để không kích hoạt Fyne đo lại kích thước container cha.

---

## GIAI ĐOẠN 2: THIẾT LẬP TỶ LỆ & KÍCH THƯỚC CỐ ĐỊNH CHUẨN NGÀNH CHO TOÀN BỘ GIAO DIỆN

Sau khi đã gỡ bỏ sạch sẽ toàn bộ các hàm gây co giãn ở Giai đoạn 1, Coder tiến hành khóa chết các thông số cố định theo chuẩn ngành:

### 2.1. Khóa Cứng Cửa Sổ Tổng (`internal/ui/app.go` & `frameless_windows.go`):
- Kích thước cửa sổ chuẩn cố định duy nhất: **`1008 x 630 px`** (Tỷ lệ vàng 16:10).
- `a.win.Resize(fyne.NewSize(1008, 630))`
- `a.win.SetFixedSize(true)`
- Kiểu dáng Windows: Frameless (không thanh xám), không viền kéo mép (`WS_THICKFRAME = false`).

### 2.2. Khóa Cứng Cấu Trúc Bảng Quota (`internal/ui/quota_badge.go`):
Toàn bộ bảng ma trận Quota có kích thước bất biến:
- **Chiều rộng tổng bảng:** CỐ ĐỊNH **`244 px`**.
- **Chiều cao tổng bảng:** CỐ ĐỊNH **`49 px`**.
- **Cột Model (Gemini / Claude):** CỐ ĐỊNH **`50 px`**.
- **Cột 5h:** CỐ ĐỊNH **`88 px`**.
- **Cột Week:** CỐ ĐỊNH **`88 px`**.
- **Vách ngăn 1px:** CỐ ĐỊNH **`1 px`**.
- **Khoảng cách (gap):** CỐ ĐỊNH **`4 px`**.
- **Ray tiến trình (`bgTrack`):** CỐ ĐỊNH **`40 px` $\times$ `6 px`**, bo góc 3px.

Mã nguồn chuẩn mẫu trong `matrixTableLayout.Layout`:
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

## GIAI ĐOẠN 3: CẬP NHẬT DỮ LIỆU TẠI CHỖ — DUY NHẤT VỆT MÀU `fillBar` TỤT DẦN THEO % QUOTA

### 3.1. Hành vi cập nhật trong ô Quota (`matrixQuotaCell.update`):
Khi dữ liệu Quota mới được nạp vào, bản chất ô Quota và bảng đứng im 100%. Duy nhất chiều dài của vệt màu `fillBar` co ngắn lại (tụt dần khi dùng):

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
	// CẤM GỌI c.container.Refresh()
}
```

### 3.2. Hành vi cập nhật trong `updateUI` (`internal/ui/app.go`):
Khi chu kỳ 60s hoàn thành hoặc bấm nút Quota:
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
Không đụng vào cửa sổ, không đụng vào `a.accountListContainer.Objects`, không gọi `Refresh()` lên toàn cây.

---

## 4. Khóa Phạm Vi Chỉnh Sửa (Touch-Map Khóa Cứng)

Coder TUYỆT ĐỐI chỉ thao tác trên đúng các file sau:

| STT | File | Nội dung thi công |
|---|---|---|
| 1 | `internal/ui/frameless_windows.go` | Gỡ bỏ `WS_THICKFRAME`, cửa sổ cố định không có viền kéo chuột; gỡ bỏ `ToggleMaximizeNativeWindow` |
| 2 | `internal/ui/window.go` | Xóa `btnMax`, xóa struct `minSizeLayout`, bỏ bọc `NewVBox`, gán thẳng `mainLayout` |
| 3 | `internal/ui/app.go` | `SetFixedSize(true)`, xóa toàn bộ biến/hàm resize thừa, kích thước chuẩn `1008x630` |
| 4 | `internal/ui/quota_badge.go` | Khóa cứng `colCycleW = 88.0 px`, ray 40px; xóa bỏ `remainW/2.0` và `c.container.Refresh()` |
| 5 | `internal/ui/themes.go` | Trả `UIPreferences` về nguyên bản chỉ lưu `Theme` |
| 6 | `accounts_store/.ui_preferences.json` | Reset về `{"theme": "dark"}` |
| 7 | `internal/ui/ui_test.go` | Cập nhật assertions kiểm thử Fixed Metrics và Fixed Window |

---

## 5. Tiêu Chuẩn Nghiệm Thu Kỹ Thuật (Acceptance Criteria)

1. **AC-01 (Fixed Window 100%):** Cửa sổ mở ra ở kích thước chuẩn `1008 x 630 px`, `FixedSize == true`. Không có viền kéo co giãn của Windows, không có nút phóng to. Cửa sổ không bao giờ bị thay đổi kích thước vì bất cứ lý do gì.
2. **AC-02 (Zero Dynamic Resizing):** Toàn bộ các hàm, phép tính co giãn động (`remainW`, `minSizeLayout`, `WS_THICKFRAME`) bị gỡ bỏ hoàn toàn khỏi codebase.
3. **AC-03 (Fixed Internal Metrics):** Bảng Quota có chiều rộng cố định 244px, các cột Model 50px, 5h 88px, Week 88px, ray 40px cố định 100%.
4. **AC-04 (Only Quota Decreases):** Khi Quota refresh, duy nhất vệt màu `fillBar` co ngắn lại theo % quota tiêu hao. Toàn bộ giao diện đứng im 100%.
5. **AC-05 (Full Build Discipline):**
   - `go test -tags gui ./internal/ui/...` đạt **100% PASS**.
   - `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all` đạt **Exit code = 0**.
