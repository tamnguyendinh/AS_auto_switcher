# Báo Cáo Đặc Tả Kiến Trúc: Chuẩn Hóa Vòng Đời Ứng Dụng Desktop — Nút X Thoát Hẳn & Khóa Đơn Bản Sao (Single Instance)

- **Tài liệu đặc tả:** `reports/architect/2026-10-18-desktop-lifecycle-exit-and-single-instance-spec.md`
- **Ngày lập:** 2026-10-18
- **Làn thực hiện:** Senior System Architect
- **Cơ quan chỉ đạo:** Human Owner trực tiếp thảo luận và chỉ đạo
- **Trạng thái:** DRAFT COMPLETED — CHỜ HUMAN OWNER XEM LẠI & PHÊ DUYỆT (CHƯA GỬI CEO)

---

## 1. Tóm Tắt Mục Tiêu & Yêu Cầu Của Human Owner

Trong quá trình sử dụng thực tế, Human Owner đã phát hiện 2 điểm bất cập nghiêm trọng về trải nghiệm người dùng (UX) và vòng đời tiến trình desktop của ứng dụng:

1. **Vấn đề 1 — Nút `[X]` không đóng hẳn ứng dụng mà lại giấu xuống khay hệ thống (Tray):**
   - *Hiện trạng:* Người dùng bấm nút `X` màu đỏ ở góc cửa sổ với mục đích tắt hoàn toàn tool, nhưng app lại tự động ẩn xuống System Tray và tiếp tục chạy ngầm.
   - *Yêu cầu:* Chuẩn hóa đúng chuẩn UX hệ điều hành: **Bấm nút `[X]` là phải THOÁT HẲN, ĐÓNG SẠCH 100% TIẾN TRÌNH**, giải phóng toàn bộ tài nguyên (RAM/CPU/goroutines) và biến mất hoàn toàn khỏi Taskbar lẫn System Tray.
2. **Vấn đề 2 — Cho phép mở cùng lúc nhiều cửa sổ tool khi bấm file `.exe` nhiều lần:**
   - *Hiện trạng:* Mỗi lần người dùng khởi chạy file `.exe` là một cửa sổ mới lại được tạo ra, sinh ra hàng loạt icon trùng lặp ở khay hệ thống, gây xung đột tranh chấp đọc/ghi file token `accounts_store/accounts.json` và xung đột kết nối Language Server.
   - *Yêu cầu:* Áp dụng chuẩn **Single Instance Application**:
     - Nếu tool **đang chạy sẵn**: Bấm file `.exe` tiếp lần nữa **tuyệt đối không được mở thêm cửa sổ thứ 2**.
     - Tiến trình mới tự động tìm cửa sổ cũ, **phục hồi và đưa cửa sổ cũ lên trên cùng màn hình (Bring to Front / Focus)** để người dùng thấy ngay.
     - Tiến trình mới tự động thoát ngay lập tức trong tích tắc (0.1ms).

---

## 2. Thiết Kế Kỹ Thuật Chi Tiết — Vấn Đề 1: Nút `[X]` Thoát Hẳn Ứng Dụng

### 2.1 Điểm Lỗi Trong Codebase Hiện Tại
Trong tệp `internal/ui/tray.go` (dòng 50–54):
```go
if a.win != nil {
    desk.SetSystemTrayWindow(a.win)
    // Minimize to tray on close  <-- ĐÂY CHÍNH LÀ NGUYÊN NHÂN GÂY LỖI UX
    a.win.SetCloseIntercept(func() {
        a.win.Hide()
    })
}
```
Hàm `SetCloseIntercept` đang can thiệp vào sự kiện đóng cửa sổ và chuyển hướng thành `a.win.Hide()`.

### 2.2 Giải Pháp Kiến Trúc Mới
Sửa đổi hành vi chặn đóng cửa sổ trong `internal/ui/tray.go`:

```go
if a.win != nil {
    desk.SetSystemTrayWindow(a.win)
    // Close Window Intercept: Thoát sạch hoàn toàn ứng dụng theo đúng chuẩn UX
    a.win.SetCloseIntercept(func() {
        a.Close()    // 1. Dọn dẹp sạch background autoLoop timer, hủy telemetry goroutine
        a.app.Quit() // 2. Thoát hoàn toàn tiến trình Fyne GUI và System Tray
    })
}
```

### 2.3 Cơ Chế Dọn Dẹp An Toàn Khi Thoát (`a.Close()`)
Trong `internal/ui/app.go`, hàm `Close()` hiện đã có sẵn cơ chế dừng an toàn:
```go
func (a *App) Close() {
    if a.autoLoop != nil {
        a.autoLoop.Stop()
    }
}
```
Khi `a.app.Quit()` được gọi sau `a.Close()`, vòng lặp sự kiện của Fyne kết thúc, toàn bộ cửa sổ, widget, icon khay hệ thống và tiến trình hệ điều hành Windows biến mất hoàn toàn.

---

## 3. Thiết Kế Kỹ Thuật Chi Tiết — Vấn Đề 2: Khóa Đơn Bản Sao (Single Instance Mutex & Foreground Activation)

### 3.1 Cơ Chế Windows Named Mutex
Để đảm bảo ứng dụng chỉ có duy nhất 1 bản sao hoạt động đồng thời trên toàn hệ điều hành, giải pháp tối ưu và nhẹ nhất là sử dụng **Named Mutex** chuẩn của Windows (`kernel32.dll`):

1. **Khởi tạo Mutex với định danh duy nhất:**
   - Tên Mutex toàn cục: `"Global\\AntigravitySwitcher_SingleInstance_Mutex_v1"`
   - Gọi hàm Win32 API: `CreateMutexW(nil, false, mutexNamePtr)`.
2. **Kiểm tra trạng thái tồn tại:**
   - Kiểm tra mã lỗi trả về: `err == windows.ERROR_ALREADY_EXISTS` (mã lỗi `183`).
   - **Trường hợp 1 (Chưa có ai chạy):** Tiến trình hiện tại là instance duy nhất. Lưu giữ con trỏ handle của Mutex trong suốt vòng đời ứng dụng. Khi ứng dụng thoát, hệ điều hành tự động giải phóng Mutex.
   - **Trường hợp 2 (Đã có instance chạy sẵn):**
     1. Gọi Win32 API `FindWindowW(nil, "⚡ Antigravity Multi-Account Switcher")` để tìm handle của cửa sổ đang chạy.
     2. Nếu tìm thấy `hwnd != 0`:
        - Gọi `ShowWindow(hwnd, SW_RESTORE)` (mã `9`): Phục hồi cửa sổ nếu đang bị thu nhỏ dưới Taskbar hoặc ẩn.
        - Gọi `SetForegroundWindow(hwnd)`: Bật sáng và đưa cửa sổ lên trên cùng màn hình ngay trước mắt người dùng.
     3. Đóng handle vừa tạo (`CloseHandle(hMutex)`).
     4. Gọi `os.Exit(0)` thoát ngay lập tức tiến trình trùng lặp này.

### 3.2 Cấu Trúc Mã Nguồn Đề Xuất
Tạo tệp chuyên biệt `internal/ui/single_instance_windows.go` (đóng gói sạch sẽ, cô lập hệ điều hành):

```go
//go:build windows

package ui

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modKernel32      = windows.NewLazySystemDLL("kernel32.dll")
	procCreateMutexW = modKernel32.NewProc("CreateMutexW")
	procCloseHandle  = modKernel32.NewProc("CloseHandle")

	procShowWindow          = modUser32.NewProc("ShowWindow")
	procSetForegroundWindow = modUser32.NewProc("SetForegroundWindow")
)

const (
	swRestore = 9
	singleInstanceMutexName = "Global\\AntigravitySwitcher_SingleInstance_Mutex"
)

// SingleInstanceLock nắm giữ handle của Windows Mutex.
type SingleInstanceLock struct {
	handle uintptr
}

// TryAcquireSingleInstance kiểm tra xem đã có bản sao nào đang chạy chưa.
// Nếu đã chạy: kích hoạt cửa sổ cũ lên đầu và trả về false.
// Nếu là bản đầu tiên: giữ khóa và trả về true.
func TryAcquireSingleInstance(windowTitle string) (*SingleInstanceLock, bool) {
	namePtr := windows.StringToUTF16Ptr(singleInstanceMutexName)
	hMutex, _, err := procCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(namePtr)))

	if err == windows.ERROR_ALREADY_EXISTS || hMutex == 0 {
		// Đã có instance khác đang chạy -> Kích hoạt cửa sổ cũ lên
		titlePtr := windows.StringToUTF16Ptr(windowTitle)
		hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(titlePtr)))
		if hwnd != 0 {
			procShowWindow.Call(hwnd, uintptr(swRestore))
			procSetForegroundWindow.Call(hwnd)
		}
		if hMutex != 0 {
			procCloseHandle.Call(hMutex)
		}
		return nil, false
	}

	return &SingleInstanceLock{handle: hMutex}, true
}

func (l *SingleInstanceLock) Release() {
	if l != nil && l.handle != 0 {
		procCloseHandle.Call(l.handle)
		l.handle = 0
	}
}
```

Đối với các hệ điều hành khác (Linux/macOS), tạo tệp stub `internal/ui/single_instance_other.go`:
```go
//go:build !windows

package ui

type SingleInstanceLock struct{}

func TryAcquireSingleInstance(windowTitle string) (*SingleInstanceLock, bool) {
	return &SingleInstanceLock{}, true
}

func (l *SingleInstanceLock) Release() {}
```

### 3.3 Tích Hợp Vào Điểm Khởi Chạy (`cmd/switcher/app_gui.go`)
Trong `cmd/switcher/app_gui.go`:
```go
func launchGUI(mgr *manager.AccountManager, showStatus, showList, doCapture bool, switchTarget string, showQuota bool) error {
    // 1. Kiểm tra Single Instance
    lock, isFirstInstance := ui.TryAcquireSingleInstance("⚡ Antigravity Multi-Account Switcher")
    if !isFirstInstance {
        // Đã có app chạy sẵn -> Cửa sổ cũ đã được đưa lên đầu màn hình -> Thoát sạch bản sao mới
        os.Exit(0)
        return nil
    }
    defer lock.Release()

    // 2. Khởi tạo và hiển thị app bình thường
    appInstance, err := ui.NewApp(mgr)
    ...
    appInstance.ShowAndRun()
    return nil
}
```

---

## 4. Danh Sách Tệp Thi Công Khóa Chặt Cho Coder (Touch-Map)

Coder Lane sẽ chỉ thao tác trên các tệp sau:
1. **`internal/ui/tray.go`:**
   - Sửa dòng 51: Trong `SetCloseIntercept`, gọi `a.Close()` và `a.app.Quit()` thay vì `a.win.Hide()`.
2. **`internal/ui/single_instance_windows.go` (Tệp mới):**
   - Viết cơ chế `TryAcquireSingleInstance` dùng `CreateMutexW`, `FindWindowW`, `ShowWindow`, `SetForegroundWindow`.
3. **`internal/ui/single_instance_other.go` (Tệp mới):**
   - Stub hỗ trợ biên dịch cross-platform.
4. **`cmd/switcher/app_gui.go`:**
   - Gọi `TryAcquireSingleInstance` trước khi khởi tạo giao diện.
5. **`internal/ui/ui_test.go`:**
   - Thêm bài test kiểm tra hàm `SetCloseIntercept` gọi `Close()` và `Quit()` chuẩn xác.

---

## 5. Tiêu Chuẩn Nghiệm Thu (Acceptance Criteria)

1. **Nút `[X]` hoạt động chuẩn 100%:**
   - Bấm vào nút `[X]` trên cửa sổ: Ứng dụng lập tức đóng ngay, biến mất khỏi màn hình, biến mất khỏi System Tray.
   - Kiểm tra `Task Manager` (hoặc lệnh `Get-Process switcher`): Tiến trình `switcher.exe` biến mất hoàn toàn, không còn sót lại tiến trình chạy ngầm.
2. **Chống mở nhiều cửa sổ (Single Instance) hoạt động chuẩn 100%:**
   - Khi app đang mở: Chạy tiếp lệnh `./switcher.exe` hoặc click đúp file exe lần thứ 2 $\rightarrow$ Không có cửa sổ mới nào hiện lên, không sinh thêm icon nào ở tray.
   - Cửa sổ đang chạy lập tức được nhấc lên trên cùng màn hình (Active / Focused).
   - Lệnh gọi thứ 2 tự động kết thúc với exit code `0`.
3. **100% Test PASS:** `go test ./...` và `go build -tags gui ./cmd/switcher` biên dịch thành công 100%.

---

## 6. Trạng Thái Bàn Giao

Tài liệu này được lập hoàn chỉnh, lưu tại `reports/architect/2026-10-18-desktop-lifecycle-exit-and-single-instance-spec.md` và **ĐANG CHỜ HUMAN OWNER XEM LẠI, CHƯA GỬI CHO CEO THEO ĐÚNG CHỈ ĐẠO**.
