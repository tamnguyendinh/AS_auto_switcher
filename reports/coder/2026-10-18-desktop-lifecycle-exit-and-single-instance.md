# Báo Cáo Coder: Chuẩn Hóa Vòng Đời Ứng Dụng Desktop — Nút X Thoát Hẳn & Khóa Đơn Bản Sao (Single Instance)

- **Tệp tin:** `reports/coder/2026-10-18-desktop-lifecycle-exit-and-single-instance.md`
- **Ngày:** 2026-10-18
- **Làn thực hiện:** Coder Specialist Lane
- **Người nhận:** CEO Agent (`01a14fbf-84eb-7bd2-98be-127e84b09884`), Supervisor Lane
- **Tài liệu căn cứ (Authority):** Báo cáo đặc tả kiến trúc `reports/architect/2026-10-18-desktop-lifecycle-exit-and-single-instance-spec.md` đã được Human Owner trực tiếp phê duyệt
- **Verdict:** `[VERDICT: READY_FOR_SUPERVISOR]`

---

## 1. Mục Tiêu & Yêu Cầu Thi Công

Triển khai chuẩn hóa vòng đời ứng dụng Desktop theo đúng chỉ đạo trực tiếp của Human Owner và đặc tả kiến trúc của Senior System Architect:

1. **Chuẩn hóa nút `[X]` thoát sạch 100% ứng dụng:**
   - Xóa bỏ hành vi thu nhỏ/ẩn xuống khay hệ thống (`a.win.Hide()`) khi người dùng bấm nút `[X]`.
   - Bấm nút `[X]` thực hiện tuần tự:
     1. Gọi `a.Close()` dừng toàn bộ background loops, timer tự động cập nhật và giải phóng tài nguyên.
     2. Gọi `a.app.Quit()` kết thúc hoàn toàn tiến trình Fyne GUI và dọn dẹp System Tray icon, tắt sạch tiến trình khỏi Task Manager.
2. **Khóa đơn bản sao (Single Instance Win32 Mutex & Foreground Activation):**
   - Sử dụng Windows Named Mutex (`Global\AntigravitySwitcher_SingleInstance_Mutex`) qua API thuần `kernel32.dll` (`CreateMutexW`, `CloseHandle`) và `user32.dll` (`FindWindowW`, `ShowWindow`, `SetForegroundWindow`), không phụ thuộc thư viện ngoài.
   - Khi ứng dụng đã đang chạy, nếu người dùng click đúp file `.exe` lần nữa:
     - Không mở thêm bất kỳ cửa sổ nào, không sinh thêm icon ở khay hệ thống.
     - Tự động tìm cửa sổ cũ (`FindWindowW`), khôi phục nếu đang bị thu nhỏ (`ShowWindow` với `SW_RESTORE = 9`) và đưa cửa sổ cũ lên trên cùng (`SetForegroundWindow`).
     - Bản sao thứ hai lập tức thoát sạch ngay (`os.Exit(0)`).
   - Bản sao đầu tiên nắm giữ khóa an toàn trong suốt vòng đời và giải phóng (`Release()`) khi thoát.
   - Hỗ trợ biên dịch chéo an toàn trên nền tảng khác qua stub `//go:build !windows`.

---

## 2. Invariant Family Map

| Thuộc tính Invariant | Chi tiết đặc tả (SSOT) | Bề mặt mã nguồn (Touch-Map) | Hành vi chuẩn hóa thi công |
|---|---|---|---|
| **Clean Exit on `[X]`** | Architect Spec §1.1, §2.1 | `internal/ui/tray.go:50-54` | `SetCloseIntercept` thực thi `a.Close()` và `a.app.Quit()`, xóa bỏ triệt để `a.win.Hide()`. |
| **Win32 Named Mutex Lock** | Architect Spec §1.2, §2.2 | `internal/ui/single_instance_windows.go` | Tạo Mutex `Global\AntigravitySwitcher_SingleInstance_Mutex`. Trả về `(*SingleInstanceLock, bool)`. Nắm giữ handle Win32. |
| **Foreground Activation on Duplicate** | Architect Spec §1.2, §2.2 | `internal/ui/single_instance_windows.go` | Nếu `ERROR_ALREADY_EXISTS` hoặc `hMutex == 0`, tìm `hwnd` theo tiêu đề cửa sổ, gọi `ShowWindow(hwnd, SW_RESTORE)` và `SetForegroundWindow(hwnd)`. Đóng `hMutex` và trả về `(nil, false)`. |
| **Cross-Platform Stub** | Architect Spec §2.2 | `internal/ui/single_instance_other.go` | Cung cấp kiểu và hàm stub cho các nền tảng `!windows` đảm bảo tính tương thích biên dịch. |
| **Startup Single Instance Gate** | Architect Spec §2.2 | `cmd/switcher/app_gui.go:14-20` | Kiểm tra `TryAcquireSingleInstance` trước khi gọi `ui.NewApp(mgr)`. Nếu `!isFirst`, gọi `os.Exit(0)` ngay lập tức. |

---

## 3. Khóa Touch-Map Thực Thi

Thực thi chuẩn xác, khóa cứng đúng 4 tệp tin code + 1 tệp tin báo cáo:

1. **`internal/ui/tray.go`**:
   - Thay thế khối `SetCloseIntercept`:
     ```go
     // Close window intercept: Thoát sạch hoàn toàn ứng dụng
     a.win.SetCloseIntercept(func() {
         a.Close()    // 1. Dừng mọi background loop, timer, dọn dẹp tài nguyên
         a.app.Quit() // 2. Thoát hoàn toàn tiến trình Fyne GUI và System Tray
     })
     ```
2. **`internal/ui/single_instance_windows.go` (Tạo mới)**:
   - Triển khai `SingleInstanceLock` và `TryAcquireSingleInstance(windowTitle string) (*SingleInstanceLock, bool)`.
   - Kết nối Win32 API: `CreateMutexW`, `CloseHandle`, `FindWindowW`, `ShowWindow`, `SetForegroundWindow`.
3. **`internal/ui/single_instance_other.go` (Tạo mới)**:
   - Stub cho các hệ điều hành không phải Windows (`//go:build !windows`).
4. **`cmd/switcher/app_gui.go`**:
   - Chèn kiểm tra đơn bản sao tại đầu hàm `launchGUI`:
     ```go
     lock, isFirst := ui.TryAcquireSingleInstance("⚡ Antigravity Multi-Account Switcher")
     if !isFirst {
         os.Exit(0)
         return nil
     }
     defer lock.Release()
     ```
5. **`reports/coder/2026-10-18-desktop-lifecycle-exit-and-single-instance.md`**:
   - Báo cáo thi công chính thức.

---

## 4. Kết Quả Xác Thực Kỷ Luật Sắt (Iron Rule Validation)

### 4.1 Tuân Thủ Tuyệt Đối Kỷ Luật Sắt Từ Human Owner
- Script `scripts/build_windows.ps1` **HOÀN TOÀN KHÔNG BỊ CHỈNH SỬA BẤT KỲ DÒNG NÀO** (`git status` và `git diff` xác nhận).
- Quá trình chạy full build bắt buộc bằng lệnh:
  ```powershell
  powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all
  ```

### 4.2 Kết Quả Full Build
```text
================================================================================
⚡ Antigravity Multi-Account Switcher - Windows Build System
   Mode: all | OutDir: bin | SkipTests: False | Clean: False
================================================================================
[2/4] Running pre-flight tests & vet...
  -> Verifying pure Go packages (CGO_ENABLED=0)...
ok  	antigravity-switch/internal/model	(cached)
ok  	antigravity-switch/internal/wincred	(cached)
ok  	antigravity-switch/internal/store	(cached)
ok  	antigravity-switch/internal/scanner	(cached)
ok  	antigravity-switch/internal/connector	(cached)
ok  	antigravity-switch/internal/reloader	(cached)
ok  	antigravity-switch/internal/manager	8.271s
ok  	antigravity-switch/cmd/switcher	(cached)
  [PASS] Pure Go verification succeeded.
  -> Verifying GUI packages with CGo & Fyne (CGO_ENABLED=1)...
ok  	antigravity-switch/cmd/switcher	(cached)
ok  	antigravity-switch/internal/ui	8.293s
  [PASS] GUI verification succeeded.
[3/4] Compiling target binaries...
  -> Checking Windows application icon and resource...
  -> Compiling resource with windres: cmd\switcher\app.rc -> cmd\switcher\rsrc_windows_amd64.syso
  [SUCCESS] Compiled PE resource object: cmd\switcher\rsrc_windows_amd64.syso (12552 bytes)
  -> Building Standalone CLI Binary (Pure Go, CGO_ENABLED=0)...
  [SUCCESS] Created: bin\antigravity-switch.exe
  Compile Duration: 3916 ms | Size: 10.51 MB (11023872 bytes)
  -> Building Standalone GUI Binary (Fyne, CGO_ENABLED=1)...
  [SUCCESS] Created: bin\antigravity-switch-gui.exe
  Compile Duration: 5781 ms | Size: 27.54 MB (28876800 bytes)
[4/4] Executing quick verification...
  [PASS] Embedded PE application icon verified on bin\antigravity-switch.exe (32x32)
  [PASS] Embedded PE application icon verified on bin\antigravity-switch-gui.exe (32x32)
  -> Running 'bin\antigravity-switch.exe --status'...
================================================================================
⚡ ANTIGRAVITY SESSION STATUS
================================================================================
Active Email:    hokinhdoanh.anvien.sales@gmail.com
Language Server: Running (PID: 33584 | Port: 64470 | CSRF: Present)
Win32 Credential: Present (260 bytes access token)
================================================================================
================================================================================
⚡ BUILD SUCCESSFUL
================================================================================
```
- **Exit Code:** `0`
- **Tất cả test suite:** `100% PASS` (Pure Go và CGo GUI Fyne).
- **Kích thước file thực thi:**
  - `bin\antigravity-switch.exe`: `10.51 MB` (đạt ngưỡng an toàn < 15MB).
  - `bin\antigravity-switch-gui.exe`: `27.54 MB`.

### 4.3 Kiểm Thử Chi Tiết Cơ Chế Single Instance Mutex
Đã thực hiện kịch bản kiểm thử trực tiếp hành vi Mutex:
1. **Lần khởi tạo thứ nhất:** Gọi `TryAcquireSingleInstance("Test Window")` -> Trả về `isFirst = true`, `lock != nil` (Nắm giữ Mutex thành công).
2. **Lần khởi tạo thứ hai (trùng lặp):** Gọi `TryAcquireSingleInstance("Test Window")` trong khi lock 1 đang giữ -> Trả về `isFirst = false`, `lock == nil`, kích hoạt tìm kiếm cửa sổ và đóng handle tạm thời.
3. **Giải phóng khóa:** Gọi `lock1.Release()` -> Đóng handle mutex Win32 an toàn.
4. **Tái cấp khóa sau khi giải phóng:** Gọi `TryAcquireSingleInstance("Test Window")` -> Trả về `isFirst = true`, `lock != nil` (Khởi tạo lại bình thường).

---

## 5. Kết Luận & Sẵn Sàng Bàn Giao

Toàn bộ yêu cầu của Human Owner và đặc tả của Architect đã được thi công hoàn tất 100%, bảo đảm không có bất kỳ sai lệch nào.
Tất cả các tệp tin trong Scope đã sẵn sàng commit Git và bàn giao cho Supervisor Lane nghiệm thu.

`[VERDICT: PASS / READY_FOR_SUPERVISOR]`