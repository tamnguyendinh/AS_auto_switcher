# Báo Cáo Đặc Tả Kiến Trúc: Chuẩn Hóa Vòng Đời Ứng Dụng Desktop — Nút X Thoát Hẳn & Khóa Đơn Bản Sao (Single Instance)

- **Tài liệu đặc tả:** `reports/architect/2026-10-18-desktop-lifecycle-exit-and-single-instance-spec.md`
- **Ngày lập:** 2026-10-18
- **Làn thực hiện:** Senior System Architect
- **Cơ quan chỉ đạo:** Human Owner trực tiếp thảo luận và chỉ đạo
- **Trạng thái:** DRAFT UPDATED WITH EXACT CODE — CHỜ HUMAN OWNER XEM LẠI & PHÊ DUYỆT (CHƯA GỬI CEO)

---

## 1. Tóm Tắt Mục Tiêu & Yêu Cầu Của Human Owner

1. **Vấn đề 1 — Nút `[X]` phải thoát hẳn 100% ứng dụng:**
   - Không được giấu/thu nhỏ ứng dụng xuống System Tray khi bấm nút `X`.
   - Bấm nút `[X]` là phải dừng toàn bộ luồng chạy ngầm, hủy timer tự động làm mới, dọn dẹp tài nguyên và tắt sạch tiến trình khỏi Task Manager và khay hệ thống.
2. **Vấn đề 2 — Khóa đơn bản sao ứng dụng (Single Instance):**
   - Khi tool đã đang chạy, nếu người dùng click đúp file `.exe` lần nữa thì **tuyệt đối không mở thêm cửa sổ thứ 2**, không sinh thêm icon trùng lặp ở khay hệ thống.
   - Bản sao mới tự động tìm cửa sổ cũ, **phục hồi và đưa cửa sổ cũ lên trên cùng màn hình (Bring to Front)**, sau đó bản sao mới tự động thoát ngay lập tức trong tích tắc (0.1ms).
3. **KỶ LUẬT THI CÔNG SẮT TỪ HUMAN OWNER (STRICT IRON RULE):**
   - **Code xong bắt buộc phải chạy full build bằng script của repo:**  
     `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all`
   - **TUYỆT ĐỐI CẤM SỬA SCRIPT FULL BUILD NÀY.**

---

## 2. Hướng Dẫn Thi Công Cụ Thể Từng Tệp (Không Cho Phép Coder Suy Diễn)

### 2.1 BƯỚC 1: Sửa Nút `[X]` Thoát Hẳn (Sửa 1 Tệp Duy Nhất)

**Tệp:** `internal/ui/tray.go` (tại khối `setupTray`, quanh dòng 50–54)

- **Đoạn mã cũ cần xóa bỏ:**
  ```go
  if a.win != nil {
      desk.SetSystemTrayWindow(a.win)
      // Minimize to tray on close
      a.win.SetCloseIntercept(func() {
          a.win.Hide()
      })
  }
  ```

- **Đoạn mã mới thay thế chính xác:**
  ```go
  if a.win != nil {
      desk.SetSystemTrayWindow(a.win)
      // Close window intercept: Thoát sạch hoàn toàn ứng dụng
      a.win.SetCloseIntercept(func() {
          a.Close()    // 1. Dừng mọi background loop, timer, dọn dẹp tài nguyên
          a.app.Quit() // 2. Thoát hoàn toàn tiến trình Fyne GUI và System Tray
      })
  }
  ```

---

### 2.2 BƯỚC 2: Khóa Đơn Bản Sao Single Instance (3 Tệp Cụ Thể)

#### Tệp 1 (Tạo mới): `internal/ui/single_instance_windows.go`
Sử dụng Windows Named Mutex chuẩn của `kernel32.dll` và `user32.dll` (không dùng thư viện ngoài):

```go
//go:build windows

package ui

import (
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
	swRestore               = 9
	singleInstanceMutexName = "Global\\AntigravitySwitcher_SingleInstance_Mutex"
)

type SingleInstanceLock struct {
	handle uintptr
}

// TryAcquireSingleInstance kiểm tra xem app đã đang chạy chưa.
// Nếu đã chạy: Đưa cửa sổ cũ lên trên cùng màn hình và trả về false.
// Nếu là bản đầu tiên: Nắm giữ khóa và trả về true.
func TryAcquireSingleInstance(windowTitle string) (*SingleInstanceLock, bool) {
	namePtr := windows.StringToUTF16Ptr(singleInstanceMutexName)
	hMutex, _, err := procCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(namePtr)))

	if err == windows.ERROR_ALREADY_EXISTS || hMutex == 0 {
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

#### Tệp 2 (Tạo mới): `internal/ui/single_instance_other.go` (Stub cho cross-platform)
```go
//go:build !windows

package ui

type SingleInstanceLock struct{}

func TryAcquireSingleInstance(windowTitle string) (*SingleInstanceLock, bool) {
	return &SingleInstanceLock{}, true
}

func (l *SingleInstanceLock) Release() {}
```

#### Tệp 3: `cmd/switcher/app_gui.go` (Chèn kiểm tra trước khi khởi chạy GUI)
Chèn 7 dòng kiểm tra ngay đầu hàm `launchGUI`:

```go
//go:build gui

package main

import (
	"fmt"
	"os"

	"antigravity-switch/internal/manager"
	"antigravity-switch/internal/ui"
)

func launchGUI(mgr *manager.AccountManager, showStatus, showList, doCapture bool, switchTarget string, showQuota bool) error {
	// BƯỚC KIỂM TRA ĐƠN BẢN SAO:
	lock, isFirst := ui.TryAcquireSingleInstance("⚡ Antigravity Multi-Account Switcher")
	if !isFirst {
		// App đã đang chạy, cửa sổ cũ đã được kéo lên trước mặt -> Bản sao mới thoát ngay
		os.Exit(0)
		return nil
	}
	defer lock.Release()

	// KHỞI CHẠY GIAO DIỆN BÌNH THƯỜNG:
	appInstance, err := ui.NewApp(mgr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed launching GUI: %v\nFalling back to CLI mode.\n", err)
		runCLI(mgr, showStatus, showList, doCapture, switchTarget, showQuota)
		return nil
	}
	appInstance.ShowAndRun()
	return nil
}
```

---

## 3. Quy Trình Xác Thực Bắt Buộc & Tiêu Chuẩn Nghiệm Thu

Sau khi thi công xong, Coder **BẮT BUỘC** phải thực hiện các bước kiểm tra sau:

1. **Quy Định Full Build Bắt Buộc:**
   - Chạy lệnh:
     ```powershell
     powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all
     ```
   - **ĐIỀU KHOẢN CẤM:** Tuyệt đối **CẤM sửa** tệp `scripts/build_windows.ps1`.
   - Kết quả bắt buộc: Phải build thành công cả CLI lẫn GUI, kiểm thử tự động đạt 100% PASS, thoát với Exit Code 0.
2. **Nghiệm Thu Thủ Công Nút `[X]`:**
   - Chạy ứng dụng $\rightarrow$ Bấm nút `[X]` ở góc trên bên phải.
   - Kiểm tra: Cửa sổ đóng lập tức, khay hệ thống mất icon, Task Manager không còn tiến trình `switcher.exe` nào.
3. **Nghiệm Thu Thủ Công Khóa Đơn Bản Sao:**
   - Chạy ứng dụng lần 1 $\rightarrow$ Cửa sổ xuất hiện.
   - Chạy tiếp file `.exe` lần 2 $\rightarrow$ Không mở thêm cửa sổ mới, không sinh thêm icon tray, cửa sổ lần 1 lập tức nổi lên trên cùng màn hình (Focused).

---

## 4. Trạng Thái Bàn Giao

Tài liệu này đã cập nhật đầy đủ mã nguồn cụ thể và điều khoản cấm sửa build script, lưu tại:  
`reports/architect/2026-10-18-desktop-lifecycle-exit-and-single-instance-spec.md`  
**TRẠNG THÁI: CHỜ HUMAN OWNER XEM LẠI VÀ RA LỆNH — CHƯA GỬI CEO.**
