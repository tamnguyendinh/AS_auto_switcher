# Báo Cáo Đặc Tả Kiến Trúc: Cửa Sổ Không Viền Tự Vẽ (Custom Frameless Window) & Tinh Giản Toàn Diện TopSection

- **Tài liệu đặc tả:** `reports/architect/2026-10-19-frameless-window-csd-and-topsection-refactor-spec.md`
- **Ngày lập:** 2026-10-19
- **Làn thực hiện:** Senior System Architect
- **Cơ quan chỉ đạo:** Human Owner trực tiếp thảo luận và phê duyệt thiết kế
- **Căn cứ tối cao:** Chỉ đạo trực tiếp từ Human Owner:
  1. *Xóa sổ hoàn toàn thanh xám mặc định của Windows* và mang 3 nút điều khiển `_  |_|  X` vào vẽ trực tiếp bên trong giao diện ứng dụng (Row 1).
  2. Tinh giản Row 2: Đưa trạng thái `Hoạt động` lên đầu (`Hoạt động | PID ... | Port ...`), bỏ chữ "Language Server :", bộ chọn Theme thu nhỏ thành dropdown mini chỉ chứa 3 icon `[ ☀️ ▼ ]` không chữ.
  3. Tinh giản Row 3 Toolbar: Nút phẳng `[ Danh sách tài khoản ]`, `[ Làm mới Quota ]`, `[ 📌 Ghim : Bật ]`, xóa sạch timestamp `[Cập nhật : HH:MM:SS]`.
- **Trạng thái:** SPECIFICATION APPROVED BY OWNER — READY FOR CODER IMPLEMENTATION

---

## ⚠️ ĐIỀU KHOẢN KỶ LUẬT SẮT BẮT BUỘC DÀNH CHO CODER (MANDATORY IRON RULE)

> **CHỈ THỊ TRỰC TIẾP TỪ HUMAN OWNER:**  
> **"KHI CODER CODE XONG, CODER PHẢI CHẠY FULL BUILD BẰNG SCRIPT CỦA REPO, CODER KHÔNG ĐƯỢC SỬA SCRIPT FULL BUILD."**  
>
> - Script full build bắt buộc: `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all`
> - Coder **TUYỆT ĐỐI KHÔNG ĐƯỢC PHÉP CHỈNH SỬA BẤT KỲ DÒNG NÀO** trong tệp script `scripts/build_windows.ps1`.
> - Quá trình build phải biên dịch sạch cả CLI lẫn GUI, 100% test suite PASS, kết thúc với Exit Code 0.

---

## 1. Bản Vẽ Định Hình Giao Diện Hoàn Chỉnh (Master Wireframe)

```lua
+-----------------------------------------------------------------------------------------+
| [AS] Antigravity Multi-Account Switcher                                         _  |_|  X |  <-- Hàng 1: Titlebar tự vẽ 100% (Frameless), 3 nút cửa sổ tùy biến
| Hoạt động  | PID 21484 | Port 65144                                            [ ☀️ ▼ ] |  <-- Hàng 2: Trạng thái đưa lên đầu + Dropdown Theme mini
| [ Danh sách tài khoản ]   [ Làm mới Quota ]   [ 📌 Ghim : Bật ]                         |  <-- Hàng 3: 3 nút Toolbar phẳng, dọn sạch chữ timestamp thừa
+-----------------------------------------------------------------------------------------+
| -------------------------------------- (Đường kẻ phân cách) --------------------------- |
| [ Bảng Ma Trận 3x3 danh sách tài khoản viền đậm 1.8px nổi bật ]                         |
+-----------------------------------------------------------------------------------------+
```

---

## 2. Thiết Kế Kỹ Thuật Chi Tiết — Cửa Sổ Không Viền (Custom Frameless Window)

### 2.1 Cơ Chế Gỡ Bỏ Thanh Tiêu Đề Windows (`WS_CAPTION`)
Trong `internal/ui/window.go`:
Sử dụng Win32 API để gỡ bỏ cờ `WS_CAPTION` khỏi thuộc tính `GWL_STYLE` của cửa sổ sau khi hiển thị:

```go
var (
	modUser32          = windows.NewLazySystemDLL("user32.dll")
	procFindWindowW    = modUser32.NewProc("FindWindowW")
	procGetWindowLongW = modUser32.NewProc("GetWindowLongW")
	procSetWindowLongW = modUser32.NewProc("SetWindowLongW")
	procSetWindowPos   = modUser32.NewProc("SetWindowPos")
	procShowWindow     = modUser32.NewProc("ShowWindow")
	procIsZoomed       = modUser32.NewProc("IsZoomed")
	procReleaseCapture = modUser32.NewProc("ReleaseCapture")
	procSendMessageW   = modUser32.NewProc("SendMessageW")

	gwlStyle int32 = -16
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

// makeWindowFrameless loại bỏ thanh tiêu đề xám của Windows nhưng giữ viền co giãn WS_THICKFRAME.
func makeWindowFrameless(title string) uintptr {
	titlePtr := windows.StringToUTF16Ptr(title)
	hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(titlePtr)))
	if hwnd == 0 {
		return 0
	}
	style, _, _ := procGetWindowLongW.Call(hwnd, uintptr(gwlStyle))
	newStyle := (style &^ uintptr(WS_CAPTION)) | uintptr(WS_THICKFRAME)
	procSetWindowLongW.Call(hwnd, uintptr(gwlStyle), newStyle)
	procSetWindowPos.Call(hwnd, 0, 0, 0, 0, 0, uintptr(SWP_FRAMECHANGED|SWP_NOMOVE|SWP_NOSIZE|SWP_NOZORDER))
	return hwnd
}

// dragNativeWindow cho phép bấm giữ chuột vào thanh tiêu đề tự vẽ để kéo di chuyển cửa sổ.
func dragNativeWindow(hwnd uintptr) {
	if hwnd != 0 {
		procReleaseCapture.Call()
		procSendMessageW.Call(hwnd, uintptr(WM_NCLBUTTONDOWN), uintptr(HTCAPTION), 0)
	}
}
```

### 2.2 Xây Dựng Bộ 3 Nút Cửa Sổ Tự Vẽ (`_`, `|_|`, `X`) Trên Hàng 1
Trong `internal/ui/window.go`:
1. **Nút Thu Nhỏ `[ — ]` (Minimize):**
   - Click: Gọi `procShowWindow.Call(hwnd, uintptr(SW_MINIMIZE))` $\rightarrow$ Cửa sổ thu nhỏ mượt mà xuống Taskbar.
2. **Nút Phóng To / Khôi Phục `[ 口 ]` (Maximize/Restore):**
   - Click:
     ```go
     zoomed, _, _ := procIsZoomed.Call(hwnd)
     if zoomed != 0 {
         procShowWindow.Call(hwnd, uintptr(SW_RESTORE))
     } else {
         procShowWindow.Call(hwnd, uintptr(SW_MAXIMIZE))
     }
     ```
3. **Nút Đóng Ứng Dụng `[ ✕ ]` (Close/Exit):**
   - Hover: Nút đổi sang màu đỏ rực `#ef4444` với chữ trắng (chuẩn phong cách Windows Close Button).
   - Click: Gọi `a.Close()` (dừng sạch background loop) và `a.app.Quit()` (thoát hẳn tiến trình 100%).

---

## 3. Thiết Kế Tinh Giản Toàn Diện Hàng 2 & Hàng 3

### 3.1 Hàng 2 (Sub-header) — Tư Duy "Status-First":
- **Góc trái:**
  - Bỏ tiền tố `"Language Server :"`.
  - Đưa trạng thái vận hành lên đầu tiên:
    - Khi server chạy: `Hoạt động  | PID <pid> | Port <port>` (chữ `Hoạt động` in đậm màu xanh hoặc text nổi bật).
    - Khi không có server: `Không hoạt động  | Dự phòng WinCred`.
- **Góc phải:**
  - Dropdown theme mini `a.themeSelect`:
    - Danh sách lựa chọn chỉ gồm 3 icon: `[]string{"🌙", "☀️", "☕"}`.
    - Xóa bỏ nhãn text `"Theme"` đứng trước.
    - Dropdown co lại siêu nhỏ gọn (~48px rộng), nằm ngay dưới cụm nút đóng mở cửa sổ.

### 3.2 Hàng 3 (Toolbar) — Tối Giản & Phẳng (De-clutter):
- **Góc trái:**
  - Nút 1: `[ Danh sách tài khoản ]` (Đổi từ `"🔄 Làm mới Chrome"`, bỏ icon unicode rườm rà).
  - Nút 2: `[ Làm mới Quota ]` (Bỏ icon unicode lặp lại).
  - Nút 3: `[ 📌 Ghim : Bật ]` (Giữ icon ghim để hiển thị trạng thái toggle ghim cửa sổ).
- **Góc phải:**
  - **Xóa bỏ hoàn toàn nhãn `a.lastUpdatedLabel` ("Cập nhật : HH:MM:SS")** khỏi thanh toolbar để trả lại không gian thoáng đãng sạch sẽ.

---

## 4. Danh Sách Tệp Thi Công Khóa Chặt Cho Coder (Touch-Map)

Coder Lane sẽ thao tác trên các tệp sau:
1. **`internal/ui/window.go`:**
   - Thêm các hàm Win32 Frameless (`makeWindowFrameless`, `dragNativeWindow`).
   - Xây dựng Row 1 Header: Tiêu đề bên trái + Bộ 3 nút điều khiển cửa sổ `_`, `|_|`, `X` bên phải.
   - Sửa Row 2: Đưa `Hoạt động` lên đầu; rút gọn `themeSelect` chỉ còn 3 icon `["🌙", "☀️", "☕"]`, bỏ `themeLabel`.
   - Sửa Row 3: Đổi tên nút thành `[ Danh sách tài khoản ]`, `[ Làm mới Quota ]`; bỏ `lastUpdatedLabel` khỏi bố cục `row3Toolbar`.
2. **`internal/ui/app.go`:**
   - Cập nhật định dạng nhãn trạng thái server: `Hoạt động  | PID %d | Port %d`.
   - Cập nhật tên nút trong logic xử lý: `"Danh sách tài khoản"`.
   - Gọi `makeWindowFrameless` sau khi cửa sổ được khởi tạo.
3. **`internal/ui/ui_test.go`:**
   - Cập nhật assertion kiểm tra định dạng nhãn `serverStatusLabel` bắt đầu bằng `"Hoạt động"` hoặc `"Không hoạt động"`.
   - Cập nhật kiểm tra `themeSelect.Options` gồm 3 icon.

---

## 5. Tiêu Chuẩn Nghiệm Thu (Acceptance Criteria)

1. **Cửa Sổ Frameless Liền Khối 100%:**
   - Mở ứng dụng: Không còn thanh xám mặc định của Windows trên đỉnh. Toàn bộ cửa sổ là một khối đồng nhất màu sắc theo theme.
   - Bấm giữ vùng tiêu đề Row 1 kéo chuột: Cửa sổ di chuyển mượt mà trên màn hình.
   - Nút `_`: Thu nhỏ cửa sổ xuống Taskbar.
   - Nút `|_|`: Phóng to / khôi phục kích thước cửa sổ.
   - Nút `X`: Nút chuyển nền đỏ khi hover, click là thoát sạch 100% ứng dụng.
2. **Header & Toolbar Đạt Chuẩn Tinh Gọn:**
   - Row 2 hiển thị rõ nét: `Hoạt động  | PID ... | Port ...` ở bên trái, bên phải là dropdown mini `[ ☀️ ▼ ]`.
   - Row 3 Toolbar chỉ có 3 nút phẳng: `[ Danh sách tài khoản ]`, `[ Làm mới Quota ]`, `[ 📌 Ghim : Bật ]`. Không còn chữ cập nhật giờ phút giây.
3. **Kỷ Luật Full Build 100% PASS:**
   - Chạy lệnh `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all` thành công với Exit Code 0, cấm sửa script build.
