# Báo Cáo Đặc Tả Kỹ Thuật Chi Tiết: Cửa Sổ Frameless Không Viền Tự Vẽ (CSD) & Tinh Giản TopSection

- **Tài liệu đặc tả:** `reports/architect/2026-10-19-frameless-window-csd-and-topsection-refactor-spec.md`
- **Ngày lập:** 2026-10-19
- **Làn thực hiện:** Senior System Architect
- **Cơ quan chỉ đạo:** Human Owner trực tiếp thảo luận và phê duyệt thiết kế
- **Mục tiêu:** Cung cấp mã nguồn cụ thể 100% từng tệp, từng dòng code để Coder thi công chính xác tuyệt đối, không phải phỏng đoán hay suy diễn.

---

## ⚠️ ĐIỀU KHOẢN KỶ LUẬT SẮT BẮT BUỘC DÀNH CHO CODER (MANDATORY IRON RULE)

> **CHỈ THỊ TRỰC TIẾP TỪ HUMAN OWNER:**  
> **"KHI CODER CODE XONG, CODER PHẢI CHẠY FULL BUILD BẰNG SCRIPT CỦA REPO, CODER KHÔNG ĐƯỢC SỬA SCRIPT FULL BUILD."**  
>
> - Script full build bắt buộc: `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all`
> - Coder **TUYỆT ĐỐI KHÔNG ĐƯỢC PHÉP CHỈNH SỬA BẤT KỲ DÒNG NÀO** trong tệp script `scripts/build_windows.ps1`.
> - Quá trình build phải biên dịch sạch cả CLI lẫn GUI, 100% test suite PASS, kết thúc với Exit Code 0.

---

## 1. Mô Hình Khung Giao Diện Chuẩn Hóa (Master Layout)

```lua
+-----------------------------------------------------------------------------------------+
| [AS] Antigravity Multi-Account Switcher     [Vùng Kéo Cửa Sổ]                   _  |_|  X |  <-- Hàng 1: Titlebar tự vẽ 100% (Frameless), 3 nút cửa sổ tùy biến
| Hoạt động  | PID 21484 | Port 65144                                            [ ☀️ ▼ ] |  <-- Hàng 2: Trạng thái đưa lên đầu + Dropdown Theme mini
| [ Danh sách tài khoản ]   [ Làm mới Quota ]   [ 📌 Ghim : Bật ]                         |  <-- Hàng 3: 3 nút Toolbar phẳng, dọn sạch chữ timestamp thừa
+-----------------------------------------------------------------------------------------+
| -------------------------------------- (Đường kẻ phân cách) --------------------------- |
| [ Bảng Ma Trận 3x3 danh sách tài khoản viền đậm 1.8px nổi bật ]                         |
+-----------------------------------------------------------------------------------------+
```

---

## 2. Hướng Dẫn Thi Công Chi Tiết Từng Tệp (Code Cụ Thể 100%)

### TỆP 1 (TẠO MỚI): `internal/ui/frameless_windows.go`
*Chứa toàn bộ logic Win32 API loại bỏ thanh tiêu đề xám của Windows và xử lý kéo thả cửa sổ.*

```go
//go:build windows

package ui

import (
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modUser32Frameless     = windows.NewLazySystemDLL("user32.dll")
	procFindWindowW        = modUser32Frameless.NewProc("FindWindowW")
	procGetWindowLongW     = modUser32Frameless.NewProc("GetWindowLongW")
	procSetWindowLongW     = modUser32Frameless.NewProc("SetWindowLongW")
	procSetWindowPos       = modUser32Frameless.NewProc("SetWindowPos")
	procShowWindow         = modUser32Frameless.NewProc("ShowWindow")
	procIsZoomed           = modUser32Frameless.NewProc("IsZoomed")
	procReleaseCapture     = modUser32Frameless.NewProc("ReleaseCapture")
	procSendMessageW       = modUser32Frameless.NewProc("SendMessageW")

	gwlStyleVal int32 = -16 // GWL_STYLE = -16
)

const (
	WS_CAPTION       = 0x00C00000
	WS_THICKFRAME    = 0x00040000
	SWP_FRAMECHANGED = 0x0020
	SWP_NOMOVE       = 0x0002
	SWP_NOSIZE       = 0x0001
	SWP_NOZORDER     = 0x0004

	SW_MINIMIZE = 6
	SW_MAXIMIZE = 3
	SW_RESTORE  = 9

	WM_NCLBUTTONDOWN = 0x00A1
	HTCAPTION        = 0x0002
)

var (
	mainHWNDLock sync.RWMutex
	mainHWND     uintptr
)

func SetMainHWND(hwnd uintptr) {
	mainHWNDLock.Lock()
	defer mainHWNDLock.Unlock()
	mainHWND = hwnd
}

func GetMainHWND() uintptr {
	mainHWNDLock.RLock()
	defer mainHWNDLock.RUnlock()
	return mainHWND
}

// MakeWindowFrameless loại bỏ thanh tiêu đề xám của Windows nhưng giữ WS_THICKFRAME để co giãn cửa sổ.
func MakeWindowFrameless(windowTitle string) uintptr {
	titlePtr := windows.StringToUTF16Ptr(windowTitle)
	hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(titlePtr)))
	if hwnd == 0 {
		return 0
	}
	SetMainHWND(hwnd)

	style, _, _ := procGetWindowLongW.Call(hwnd, uintptr(gwlStyleVal))
	newStyle := (style &^ uintptr(WS_CAPTION)) | uintptr(WS_THICKFRAME)
	procSetWindowLongW.Call(hwnd, uintptr(gwlStyleVal), newStyle)
	procSetWindowPos.Call(hwnd, 0, 0, 0, 0, 0, uintptr(SWP_FRAMECHANGED|SWP_NOMOVE|SWP_NOSIZE|SWP_NOZORDER))
	return hwnd
}

// DragNativeWindow cho phép bấm giữ chuột vào thanh tiêu đề tự vẽ để kéo cửa sổ native.
func DragNativeWindow() {
	hwnd := GetMainHWND()
	if hwnd != 0 {
		procReleaseCapture.Call()
		procSendMessageW.Call(hwnd, uintptr(WM_NCLBUTTONDOWN), uintptr(HTCAPTION), 0)
	}
}

// MinimizeNativeWindow thu nhỏ cửa sổ xuống Taskbar.
func MinimizeNativeWindow() {
	hwnd := GetMainHWND()
	if hwnd != 0 {
		procShowWindow.Call(hwnd, uintptr(SW_MINIMIZE))
	}
}

// ToggleMaximizeNativeWindow phóng to hoặc khôi phục kích thước cửa sổ.
func ToggleMaximizeNativeWindow() {
	hwnd := GetMainHWND()
	if hwnd != 0 {
		zoomed, _, _ := procIsZoomed.Call(hwnd)
		if zoomed != 0 {
			procShowWindow.Call(hwnd, uintptr(SW_RESTORE))
		} else {
			procShowWindow.Call(hwnd, uintptr(SW_MAXIMIZE))
		}
	}
}
```

---

### TỆP 2 (TẠO MỚI): `internal/ui/frameless_other.go`
*Stub rỗng để đảm bảo cross-platform không bị lỗi compile.*

```go
//go:build !windows

package ui

func SetMainHWND(hwnd uintptr) {}
func GetMainHWND() uintptr { return 0 }
func MakeWindowFrameless(windowTitle string) uintptr { return 0 }
func DragNativeWindow() {}
func MinimizeNativeWindow() {}
func ToggleMaximizeNativeWindow() {}
```

---

### TỆP 3: `internal/ui/window.go`
*Sửa đổi và bổ sung các widget tiêu đề tự vẽ, nút đóng mở và sắp xếp lại 3 hàng TopSection.*

#### 3.1 Bổ sung Widget Nút Cửa Sổ Tùy Biến (Custom Window Buttons)
Thêm vào cuối tệp `window.go`:

```go
// customTitleBtn: Widget nút thu nhỏ/phóng to có hover sáng
type customTitleBtn struct {
	widget.BaseWidget
	bg       *canvas.Rectangle
	text     *canvas.Text
	onTapped func()
}

func newCustomTitleBtn(label string, onTapped func()) *customTitleBtn {
	b := &customTitleBtn{onTapped: onTapped}
	b.bg = canvas.NewRectangle(color.Transparent)
	b.bg.CornerRadius = 3
	b.text = canvas.NewText(label, color.NRGBA{R: 0xcd, G: 0xd6, B: 0xf4, A: 0xff})
	b.text.TextSize = 13
	b.text.Alignment = fyne.TextAlignCenter
	b.ExtendBaseWidget(b)
	return b
}

func (b *customTitleBtn) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewStack(b.bg, container.NewCenter(b.text))
	return widget.NewSimpleRenderer(c)
}

func (b *customTitleBtn) MouseIn(*desktop.MouseEvent) {
	b.bg.FillColor = color.NRGBA{R: 0x45, G: 0x47, B: 0x5a, A: 0xaa} // Hover sáng
	b.Refresh()
}

func (b *customTitleBtn) MouseOut() {
	b.bg.FillColor = color.Transparent
	b.Refresh()
}

func (b *customTitleBtn) MouseMoved(*desktop.MouseEvent) {}

func (b *customTitleBtn) Tapped(*fyne.PointEvent) {
	if b.onTapped != nil {
		b.onTapped()
	}
}

// customCloseBtn: Nút X có hover màu đỏ rực (#ef4444)
type customCloseBtn struct {
	widget.BaseWidget
	bg       *canvas.Rectangle
	text     *canvas.Text
	onTapped func()
}

func newCustomCloseBtn(onTapped func()) *customCloseBtn {
	b := &customCloseBtn{onTapped: onTapped}
	b.bg = canvas.NewRectangle(color.Transparent)
	b.bg.CornerRadius = 3
	b.text = canvas.NewText("✕", color.NRGBA{R: 0xcd, G: 0xd6, B: 0xf4, A: 0xff})
	b.text.TextSize = 13
	b.text.Alignment = fyne.TextAlignCenter
	b.ExtendBaseWidget(b)
	return b
}

func (b *customCloseBtn) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewStack(b.bg, container.NewCenter(b.text))
	return widget.NewSimpleRenderer(c)
}

func (b *customCloseBtn) MouseIn(*desktop.MouseEvent) {
	b.bg.FillColor = color.NRGBA{R: 0xef, G: 0x44, B: 0x44, A: 0xff} // Đỏ rực khi hover
	b.text.Color = color.White
	b.Refresh()
}

func (b *customCloseBtn) MouseOut() {
	b.bg.FillColor = color.Transparent
	b.text.Color = color.NRGBA{R: 0xcd, G: 0xd6, B: 0xf4, A: 0xff}
	b.Refresh()
}

func (b *customCloseBtn) MouseMoved(*desktop.MouseEvent) {}

func (b *customCloseBtn) Tapped(*fyne.PointEvent) {
	if b.onTapped != nil {
		b.onTapped()
	}
}

// windowDragArea: Vùng kéo thả cửa sổ native khi bấm giữ chuột
type windowDragArea struct {
	widget.BaseWidget
}

func (d *windowDragArea) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(canvas.NewRectangle(color.Transparent))
}

func (d *windowDragArea) Dragged(e *fyne.DragEvent) {
	DragNativeWindow()
}

func (d *windowDragArea) DragEnd() {}
```

#### 3.2 Tái cấu trúc hàm `setupLayout()` trong `window.go`

Thay thế toàn bộ đoạn khởi tạo TopSection cũ bằng:

```go
// -------------------------------------------------------------------------
// TopSection Row 1 (Header): Monogram [AS] + App Title + Drag Area + Window Controls (_ |_| X)
// -------------------------------------------------------------------------
a.monogramBg = canvas.NewRectangle(palette.Primary)
a.monogramBg.CornerRadius = 6
monogramText := canvas.NewText("AS", color.White)
monogramText.TextStyle = fyne.TextStyle{Bold: true}
monogramText.TextSize = 13
monogramText.Alignment = fyne.TextAlignCenter
monogramStack := container.NewStack(a.monogramBg, container.NewCenter(monogramText))
monogramWrap := container.NewGridWrap(fyne.NewSize(26, 26), monogramStack)

titleLabel := widget.NewLabelWithStyle("Antigravity Multi-Account Switcher", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
titleLeft := container.NewHBox(monogramWrap, titleLabel)

// Bộ 3 nút điều khiển cửa sổ
btnMin := container.NewGridWrap(fyne.NewSize(32, 26), newCustomTitleBtn("—", func() {
	MinimizeNativeWindow()
}))
btnMax := container.NewGridWrap(fyne.NewSize(32, 26), newCustomTitleBtn("口", func() {
	ToggleMaximizeNativeWindow()
}))
btnClose := container.NewGridWrap(fyne.NewSize(32, 26), newCustomCloseBtn(func() {
	a.Close()
	a.app.Quit()
}))
windowControls := container.NewHBox(btnMin, btnMax, btnClose)

dragArea := &windowDragArea{}
dragArea.ExtendBaseWidget(dragArea)

row1Header := container.NewBorder(nil, nil, titleLeft, windowControls, dragArea)

// -------------------------------------------------------------------------
// TopSection Row 2 (Sub-header): Status đưa lên đầu (trái) + Theme Select mini (phải)
// -------------------------------------------------------------------------
a.serverStatusLabel = widget.NewLabelWithStyle("Hoạt động  | Đang kết nối...", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

// Theme Select chỉ gồm 3 icon, thu hẹp bề ngang tối đa
themeOptions := []string{"🌙", "☀️", "☕"}
a.themeSelect = widget.NewSelect(themeOptions, func(selected string) {
	switch selected {
	case "🌙":
		a.SetTheme(ThemeDark)
	case "☀️":
		a.SetTheme(ThemeLight)
	case "☕":
		a.SetTheme(ThemeWarm)
	}
})

switch a.currentTheme {
case ThemeLight:
	a.themeSelect.SetSelected("☀️")
case ThemeWarm:
	a.themeSelect.SetSelected("☕")
default:
	a.themeSelect.SetSelected("🌙")
}

themeSelectWrap := container.NewGridWrap(fyne.NewSize(58, 32), a.themeSelect)
row2SubHeader := container.NewBorder(nil, nil, a.serverStatusLabel, themeSelectWrap)

// -------------------------------------------------------------------------
// TopSection Row 3 (Toolbar): Nút phẳng (trái) + Bỏ sạch timestamp
// -------------------------------------------------------------------------
a.refreshBtn = widget.NewButton("Danh sách tài khoản", func() {
	a.Refresh()
})

a.quotaRefreshAllBtn = widget.NewButton("Làm mới Quota", func() {
	a.RefreshAllQuota()
})
a.quotaRefreshAllBtn.Importance = widget.HighImportance

pinLabel := "📌 Ghim : Bật"
pinImportance := widget.HighImportance
if !a.alwaysOnTop {
	pinLabel = "📌 Ghim : Tắt"
	pinImportance = widget.MediumImportance
}
a.alwaysOnTopBtn = widget.NewButton(pinLabel, func() {
	a.ToggleAlwaysOnTop()
})
a.alwaysOnTopBtn.Importance = pinImportance

toolbarLeft := container.NewHBox(a.refreshBtn, a.quotaRefreshAllBtn, a.alwaysOnTopBtn)
row3Toolbar := toolbarLeft // Phía bên phải để trống hoàn toàn thoáng đãng

// Separator chia phần điều khiển với lưới tài khoản
sep := widget.NewSeparator()

topSection := container.NewVBox(
	row1Header,
	row2SubHeader,
	row3Toolbar,
	sep,
)
```

---

### TỆP 4: `internal/ui/app.go`
*Kích hoạt tính năng Frameless và cập nhật định dạng hiển thị trạng thái server.*

1. **Trong hàm `Init()`:**
   Sau khi `a.win.Resize(...)`, thêm goroutine kích hoạt frameless:
   ```go
   go func() {
       time.Sleep(150 * time.Millisecond)
       MakeWindowFrameless(windowTitle)
   }()
   ```

2. **Trong logic cập nhật `serverStatusLabel` (khoảng dòng 640):**
   - Đổi từ:
     `fmt.Sprintf("Language Server : PID %d | Port %d | Hoạt động", ...)`
   - Sang định dạng Status-First mới:
     ```go
     if overview.LanguageServerRunning && overview.ServerInfo != nil {
         a.serverStatusLabel.SetText(fmt.Sprintf("Hoạt động  | PID %d | Port %d", overview.ServerInfo.PID, overview.ServerInfo.Port))
     } else {
         a.serverStatusLabel.SetText("Không hoạt động  | Dự phòng WinCred")
     }
     ```

3. **Cập nhật nút làm mới danh sách tài khoản:**
   Đổi nhãn khi kết thúc refresh: `a.refreshBtn.SetText("Danh sách tài khoản")`.

---

### TỆP 5: `internal/ui/ui_test.go`
*Cập nhật các ca kiểm thử để đảm bảo 100% test PASS.*

1. Sửa assertion kiểm tra trạng thái server:
   Kiểm tra `strings.Contains(app.serverStatusLabel.Text, "Hoạt động")` hoặc `"Không hoạt động"`.
2. Sửa kiểm tra `themeSelect.Options`: Đảm bảo mảng chứa `["🌙", "☀️", "☕"]`.

---

## 3. Quy Trình Xác Thực Nghiệm Thu Bắt Buộc

1. **Kỷ Luật Full Build:**
   ```powershell
   powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all
   ```
   Bắt buộc exit code 0, không được chỉnh sửa script build.
2. **Nghiệm Thu Bằng Mắt:**
   - Cửa sổ không còn thanh xám Windows.
   - Bấm giữ vùng trống Row 1 kéo chuột: Cửa sổ di chuyển mượt mà.
   - Nút `—`: Thu nhỏ cửa sổ xuống Taskbar.
   - Nút `口`: Phóng to / khôi phục cửa sổ.
   - Nút `✕`: Nền đổi sang màu đỏ rực khi rê chuột, bấm vào là app tắt sạch ngay.
   - Hàng 2: Hiện `Hoạt động | PID ... | Port ...` bên trái, `[ ☀️ ▼ ]` bên phải.
   - Hàng 3: Hiện `[ Danh sách tài khoản ]`, `[ Làm mới Quota ]`, `[ 📌 Ghim : Bật ]`. Không còn chữ cập nhật giờ phút giây.
