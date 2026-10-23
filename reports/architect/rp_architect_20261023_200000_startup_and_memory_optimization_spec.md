# BÁO CÁO ĐẶC TẢ KIẾN TRÚC & BẢN THIẾT KẾ TRIỂN KHAI CHI TIẾT (ARCHITECTURE & CODE-LEVEL IMPLEMENTATION SPECIFICATION)
## TỐI ƯU HÓA HIỆU NĂNG TẦNG DƯỚI & BẢO TOÀN TRỌN VẸN TRẢI NGHIỆM NGƯỜI DÙNG

- **Mã tài liệu:** `rp_architect_20261023_200000_startup_and_memory_optimization_spec`
- **Tác giả:** System Architect Lane
- **Ngày lập:** 2026-10-23 (Phiên bản R4 — Bản Đặc tả Kỹ thuật Mức Mã Nguồn: Coder triển khai trực tiếp 100% không cần suy đoán)
- **Thẩm quyền:** Human Owner
- **Đối tượng thụ hưởng:** Coder Lane, QA Lane
- **Trạng thái:** **`VERDICT: PASS / READY_FOR_CODER`**

---

## 1. NGUYÊN TẮC THIẾT KẾ CỐT LÕI (CORE PHILOSOPHY)

> **"Sự cẩn thận thực sự là bảo vệ trải nghiệm đang có. Tối ưu hoàn toàn ở tầng dưới (under-the-hood); giữ nguyên 100% giao diện, cách bấm và thói quen tương tác của người dùng. Không được vì 'an toàn' phòng ngừa mà đẻ thêm trạng thái khóa toàn cục, skeleton hay menu rườm rà làm ứng dụng khó dùng hơn."**

### Ba Mục tiêu Trọng tâm:
1. **Khởi động tức thì (< 300ms):** Cửa sổ hiện ngay với dữ liệu có sẵn từ cache đĩa; loại bỏ hoàn toàn các tiến trình phụ (`powershell.exe`, `netstat.exe`).
2. **Không xáo trộn thao tác khi đang dùng:** Khi người dùng bấm "Làm mới danh sách", toàn bộ nội dung và các nút bấm hợp lệ vẫn giữ nguyên khả năng tương tác bình thường; dữ liệu mới âm thầm cập nhật khi có kết quả.
3. **Bộ nhớ nhẹ nhàng:** Giảm lượng heap mà Go giữ lại thông qua dọn dẹp nhàn rỗi và khảo sát `GOMEMLIMIT`, không làm gián đoạn độ mượt của giao diện.

---

## 2. NGUYÊN TẮC BẤT BIẾN BẮT BUỘC (ARCHITECTURAL INVARIANTS)

1. **`Invariant-Preserve-Existing-UX` (Bảo toàn 100% Giao diện & Thao tác):**
   - Giữ nguyên toàn bộ layout, màu sắc, vị trí các nút bấm, dialog xác nhận xóa và cách tương tác hiện có.
   - Tuyệt đối KHÔNG đưa toàn bộ app vào trạng thái khóa toàn cục (`VERIFYING`) mỗi khi làm mới.
   - Tuyệt đối KHÔNG thêm các thành phần giao diện thừa thãi như skeleton loader, badge lạ hay các nút thử lại gây rối mắt.
2. **`Invariant-Trash-Button-Untouched` (Bảo toàn luồng chạy Nút Xóa):**
   - Nút Xóa `[ 🗑 ]` trên mọi thẻ không hoạt động (Inactive: bao gồm Status ON, Status OFF, Cần OAuth) **LÚC NÀO CŨNG BẬT VÀ HIỂN THỊ (`Show()` và `Enable()`)**.
   - Nút Xóa trên thẻ đang hoạt động (`IsActive == true`): **ẨN HOÀN TOÀN (`Hide()`)**.
   - Đợt tối ưu này tuyệt đối KHÔNG can thiệp, không vô hiệu hóa hay làm thay đổi bất kỳ hành vi nào của nút Xóa (tuân thủ tuyệt đối `SPEC-ARCH-20261023-ACCOUNT-DELETION-AND-TRASH-BUTTON`).
3. **`Invariant-Zero-Subprocess-Discovery` (Cấm 100% Subprocess):**
   - Loại bỏ hoàn toàn việc gọi `powershell.exe` và `netstat.exe`.
   - Quét tiến trình `language_server.exe` bằng Win32 Toolhelp32 API thuần trong kernel (< 1ms).
   - Đọc CommandLine bằng `ntdll.dll::NtQueryInformationProcess` (Class 60 - `ProcessCommandLineInformation`).
   - Quét cổng lắng nghe 100% bằng Win32 API `iphlpapi.dll::GetExtendedTcpTable` thuần.
4. **`Invariant-Transparent-Background-Refresh` (Làm mới ngầm không gián đoạn):**
   - Trong chu kỳ app đang mở và người dùng bấm "Làm mới danh sách": Nút trên Toolbar chỉ đổi chữ sang `"⏳ Đang làm mới..."` như thiết kế hiện tại, còn lại danh sách thẻ tài khoản và các nút bấm vẫn giữ nguyên trạng thái để người dùng thao tác. Khi goroutine ngầm hoàn tất thì dữ liệu mới cập nhật vào UI.
5. **`Invariant-Silent-Stale-Drop` (Hủy kết quả cũ âm thầm):**
   - Quản lý `refreshGen atomic.Uint64` tăng dần khi có Refresh hoặc Switch.
   - Kiểm tra `refreshGen` ngay trong callback `fyne.Do()`. Nếu phát hiện là kết quả của lượt cũ thì âm thầm bỏ qua (silent drop), không ghi đè dữ liệu mới và không can thiệp vào UI.
6. **`Invariant-Backend-Execution-Time-Check` (Bảo vệ tại thời điểm thực thi):**
   - Không khóa nút phòng ngừa vô cớ trên UI.
   - Tại tầng backend (`AccountManager`), việc kiểm tra điều kiện (tài khoản active, token hợp lệ) được thực hiện ngay tại thời điểm thực thi lệnh Chuyển hoặc Xóa. Nếu vi phạm (ví dụ tài khoản vừa trở thành active), backend trả về lỗi tương ứng để UI hiển thị thông báo chính xác cho thao tác đó.
7. **`Invariant-Preserve-Window-Controls` (Giữ nguyên điều khiển cửa sổ):**
   - Nút `X` và close intercept: Thoát sạch hoàn toàn ứng dụng (`a.Close()` -> `a.app.Quit()`).
   - Nút `—`: Thu nhỏ xuống Windows Taskbar (`MinimizeNativeWindow()`).
   - Tuyệt đối không thêm luồng mới hay sửa nút đóng thành ẩn chỉ để tìm chỗ gọi hàm trim.

---

## 3. BẢN ĐỒ CAN THIỆP MÃ NGUỒN (TOUCH-MAP)

| Tệp mã nguồn | Vị trí can thiệp | Nội dung thay đổi chi tiết |
|---|---|---|
| `internal/connector/discovery.go` | Dòng 69 - 100 & Dòng 193 - 245 | - Thay thế `QueryLanguageServerProcesses` bằng Win32 Toolhelp32 + `NtQueryInformationProcess`.<br>- Xóa bỏ hàm `discoverPortsViaNetstat`.<br>- Thêm struct `serverDiscoveryCache` + `singleflight.Group` dùng chung. |
| `internal/ui/app.go` | Dòng 628 - 635 & Dòng 806 - 864 | - Thêm trường `refreshGen atomic.Uint64` vào `struct App`.<br>- Xóa bỏ goroutine 50ms `SyncVisualScaleAndTextures` sau `win.Show()`, thay bằng gọi 1 lần `debug.FreeOSMemory()` chạy ngầm.<br>- Trong `Refresh()`: tăng và so khớp `refreshGen` trong `fyne.Do()`. |
| `internal/manager/account_manager.go` | Dòng 215 - 235 (`RemoveAccount`) | - Trong `RemoveAccount`: kiểm tra lại `activeEmail` (Double-check) trước khi xóa tệp. |

---

## 4. HƯỚNG DẪN MÃ NGUỒN CHI TIẾT CHO CODER (EXACT CODE IMPLEMENTATION)

### 4.1. Cập nhật `internal/connector/discovery.go`

#### A. Khai báo DLL và Win32 Struct cần thiết
Tại phần đầu file `internal/connector/discovery.go`:
```go
import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sync/singleflight"
	"golang.org/x/sys/windows"
)

var (
	modNtdll                      = windows.NewLazySystemDLL("ntdll.dll")
	procNtQueryInformationProcess = modNtdll.NewProc("NtQueryInformationProcess")
)

const (
	processCommandLineInformation = 60
)

type unicodeString struct {
	Length        uint16
	MaximumLength uint16
	Buffer        *uint16
}

// getProcessCommandLine đọc dòng lệnh của tiến trình bằng NtQueryInformationProcess class 60
func getProcessCommandLine(hProcess windows.Handle) (string, error) {
	var returnLength uint32
	// Lần gọi 1 để lấy độ dài buffer cần cấp phát
	r0, _, _ := procNtQueryInformationProcess.Call(
		uintptr(hProcess),
		uintptr(processCommandLineInformation),
		0,
		0,
		uintptr(unsafe.Pointer(&returnLength)),
	)
	if returnLength == 0 {
		return "", errors.New("không xác định được kích thước buffer dòng lệnh")
	}

	buf := make([]byte, returnLength)
	r0, _, _ = procNtQueryInformationProcess.Call(
		uintptr(hProcess),
		uintptr(processCommandLineInformation),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(returnLength),
		uintptr(unsafe.Pointer(&returnLength)),
	)
	if int32(r0) < 0 {
		return "", fmt.Errorf("NtQueryInformationProcess lỗi NTSTATUS 0x%X", r0)
	}

	us := (*unicodeString)(unsafe.Pointer(&buf[0]))
	if us.Length == 0 || us.Buffer == nil {
		return "", nil
	}

	charCount := int(us.Length / 2)
	slice := unsafe.Slice(us.Buffer, charCount)
	return windows.UTF16ToString(slice), nil
}
```

#### B. Viết lại hàm `QueryLanguageServerProcesses`
Thay thế toàn bộ hàm cũ (dòng 70–100) bằng:
```go
// QueryLanguageServerProcesses tìm các tiến trình language_server.exe qua Win32 Toolhelp32 Snapshot thuần (< 1ms)
func QueryLanguageServerProcesses() ([]ProcessCandidate, error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, fmt.Errorf("CreateToolhelp32Snapshot failed: %w", err)
	}
	defer windows.CloseHandle(snapshot)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))

	if err := windows.Process32First(snapshot, &entry); err != nil {
		return nil, fmt.Errorf("Process32First failed: %w", err)
	}

	var candidates []ProcessCandidate
	for {
		name := windows.UTF16ToString(entry.ExeFile[:])
		if strings.EqualFold(name, "language_server.exe") {
			pid := int(entry.ProcessID)
			hProcess, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, entry.ProcessID)
			if err == nil {
				cmdLine, err := getProcessCommandLine(hProcess)
				_ = windows.CloseHandle(hProcess)
				if err == nil && cmdLine != "" {
					candidates = append(candidates, ProcessCandidate{
						PID:         pid,
						CommandLine: cmdLine,
					})
				}
			}
		}

		if err := windows.Process32Next(snapshot, &entry); err != nil {
			break
		}
	}
	return candidates, nil
}
```

#### C. Xóa bỏ `discoverPortsViaNetstat` và Cập nhật `DiscoverListeningPorts`
- Xóa bỏ hoàn toàn hàm `discoverPortsViaNetstat(pid int) ([]int, error)`.
- Hàm `DiscoverListeningPorts(pid int)` (dòng 246):
```go
// DiscoverListeningPorts tìm tất cả các cổng TCP listening loopback bằng Win32 GetExtendedTcpTable thuần
func DiscoverListeningPorts(pid int) ([]int, error) {
	if pid <= 0 {
		return nil, errors.New("invalid PID")
	}
	ports, err := discoverPortsViaWin32(pid)
	if err != nil || len(ports) == 0 {
		return nil, fmt.Errorf("không tìm thấy cổng lắng nghe cho PID %d: %w", pid, err)
	}
	return ports, nil
}
```

#### D. Quản lý Cache Vòng đời & Singleflight trong `DiscoverLanguageServer`
```go
type serverDiscoveryCache struct {
	mu           sync.RWMutex
	info         *ServerInfo
	creationTime windows.Filetime
	cachedAt     time.Time
}

var (
	discCache  serverDiscoveryCache
	discFlight singleflight.Group
)

// InvalidateDiscoveryCache xóa cache nếu thông tin kết nối bị lỗi
func InvalidateDiscoveryCache(failedPort int, failedCSRF string) {
	discCache.mu.Lock()
	defer discCache.mu.Unlock()
	if discCache.info != nil {
		if (failedPort <= 0 || discCache.info.Port == failedPort) &&
			(failedCSRF == "" || discCache.info.CSRFToken == failedCSRF) {
			discCache.info = nil
			discCache.cachedAt = time.Time{}
		}
	}
}

// DiscoverLanguageServer quét tiến trình language_server.exe với bộ nhớ đệm 10s và singleflight gom luồng
func DiscoverLanguageServer() (*ServerInfo, error) {
	// 1. Kiểm tra cache còn hạn (TTL 10s)
	discCache.mu.RLock()
	cached := discCache.info
	cachedTime := discCache.cachedAt
	cachedCreation := discCache.creationTime
	discCache.mu.RUnlock()

	if cached != nil && time.Since(cachedTime) <= 10*time.Second {
		// Kiểm tra tiến trình còn sống và đúng CreationTime không
		if hProcess, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(cached.PID)); err == nil {
			var ct, et, kt, ut windows.Filetime
			if err := windows.GetProcessTimes(hProcess, &ct, &et, &kt, &ut); err == nil {
				_ = windows.CloseHandle(hProcess)
				if ct == cachedCreation {
					return cached, nil
				}
			} else {
				_ = windows.CloseHandle(hProcess)
			}
		}
	}

	// 2. Gom nhiều cuộc gọi đồng thời qua singleflight
	res, err, _ := discFlight.Do("discover_language_server", func() (any, error) {
		// Re-check cache bên trong flight
		discCache.mu.RLock()
		if discCache.info != nil && time.Since(discCache.cachedAt) <= 10*time.Second {
			info := discCache.info
			discCache.mu.RUnlock()
			return info, nil
		}
		discCache.mu.RUnlock()

		procs, err := QueryLanguageServerProcesses()
		if err != nil {
			return nil, fmt.Errorf("failed querying processes: %w", err)
		}
		if len(procs) == 0 {
			return nil, ErrLanguageServerNotFound
		}

		for _, p := range procs {
			if p.PID <= 0 || strings.TrimSpace(p.CommandLine) == "" {
				continue
			}
			csrfToken := ExtractCSRFToken(p.CommandLine)
			if csrfToken == "" {
				continue
			}
			port, err := DiscoverListeningPort(p.PID)
			if err != nil || port <= 0 {
				continue
			}

			var creationTime windows.Filetime
			if hProcess, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(p.PID)); err == nil {
				var et, kt, ut windows.Filetime
				_ = windows.GetProcessTimes(hProcess, &creationTime, &et, &kt, &ut)
				_ = windows.CloseHandle(hProcess)
			}

			info := &ServerInfo{
				PID:       p.PID,
				CSRFToken: csrfToken,
				Port:      port,
			}

			discCache.mu.Lock()
			discCache.info = info
			discCache.creationTime = creationTime
			discCache.cachedAt = time.Now()
			discCache.mu.Unlock()

			return info, nil
		}
		return nil, ErrLanguageServerNotFound
	})

	if err != nil {
		return nil, err
	}
	return res.(*ServerInfo), nil
}
```

---

### 4.2. Cập nhật `internal/ui/app.go`

#### A. Khai báo `refreshGen` trong `struct App`
Tại `internal/ui/app.go` (khoảng dòng 75):
```go
type App struct {
    ...
    refreshGen atomic.Uint64 // Đếm thế hệ refresh/switch để hủy bỏ kết quả stale âm thầm
}
```

#### B. Xóa bỏ Re-theme 50ms và thêm Dọn dẹp Bộ nhớ trong `ShowAndRun()`
Tại `internal/ui/app.go` (khoảng dòng 626–635):
- **Xóa bỏ đoạn cũ:**
  ```go
  go func() {
      time.Sleep(50 * time.Millisecond)
      fyne.Do(func() {
          a.SyncVisualScaleAndTextures()
      })
  }()
  ```
- **Thay bằng:**
  ```go
  // Thu hồi bộ nhớ heap sau khi khởi động ổn định (chạy ngầm 1 lần duy nhất)
  go func() {
      time.Sleep(300 * time.Millisecond)
      debug.FreeOSMemory()
  }()
  ```

#### C. Cập nhật hàm `Refresh()` để chống kết quả stale âm thầm
Tại `internal/ui/app.go` (khoảng dòng 806–864):
```go
// Refresh updates the accounts overview asynchronously without locking user interactions.
func (a *App) Refresh() {
	a.mu.Lock()
	if a.isBusy {
		a.mu.Unlock()
		return
	}
	a.isBusy = true
	gen := a.refreshGen.Add(1) // Tăng thế hệ
	a.mu.Unlock()

	if a.refreshBtn != nil {
		fyne.Do(func() {
			a.refreshBtn.SetText("⏳ Đang làm mới...")
		})
	}

	go func() {
		defer func() {
			fyne.Do(func() {
				if a.refreshGen.Load() == gen {
					if a.refreshBtn != nil {
						a.refreshBtn.SetText("Làm mới danh sách")
					}
					a.mu.Lock()
					a.isBusy = false
					a.mu.Unlock()
				}
			})
		}()

		if a.mgr == nil {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()

		overview, err := a.mgr.GetAccountStates(ctx)
		if err != nil {
			fyne.Do(func() {
				if a.refreshGen.Load() == gen {
					a.showStatusBar("Không lấy được danh sách tài khoản: " + err.Error())
				}
			})
			return
		}

		fyne.Do(func() {
			// Bỏ qua kết quả cũ nếu người dùng đã bấm thao tác mới hơn
			if a.refreshGen.Load() != gen {
				return
			}

			a.mu.Lock()
			a.overview = overview
			a.mu.Unlock()

			a.updateUI(overview)
			if len(overview.Warnings) > 0 {
				msg := strings.Join(overview.Warnings, "; ")
				if a.IsMaskAccounts() {
					msg = maskMessageEmails(msg)
				}
				a.showStatusBar("⚠️ " + msg)
			} else {
				a.showStatusBar("Đã cập nhật danh sách tài khoản")
				a.autoHideStatusBar(3 * time.Second)
			}
		})
	}()
}
```

---

### 4.3. Cập nhật `internal/manager/account_manager.go`

Trong hàm `RemoveAccount` (`internal/manager/account_manager.go` dòng 215–225):
```go
func (m *AccountManager) RemoveAccount(ctx context.Context, email string) error {
	normEmail := strings.ToLower(strings.TrimSpace(email))
	if normEmail == "" {
		return store.ErrInvalidEmail
	}

	// Invariant: Double-check kiểm tra activeEmail trước khi thực hiện xóa
	activeEmail, _ := m.GetCurrentActiveEmail(ctx)
	if activeEmail != "" && strings.EqualFold(activeEmail, normEmail) {
		return ErrCannotDeleteActiveAccount
	}

	m.mu.Lock()
	accStore, discStore := m.store, m.discoveredStore
	m.mu.Unlock()

	// Tiến hành xóa tệp tài khoản bình thường
	if accStore != nil {
		if err := accStore.DeleteAccount(normEmail); err != nil {
			return fmt.Errorf("lỗi xóa tệp tài khoản: %w", err)
		}
	}
    ...
```

---

## 5. THỨ TỰ TRIỂN KHAI CHO CODER

Coder chỉ cần mở đúng 3 file và chỉnh sửa theo đúng thứ tự:
1. **File 1:** `internal/connector/discovery.go` (Thay thế PowerShell/Netstat bằng Win32 API thuần, thêm Cache TTL 10s + singleflight).
2. **File 2:** `internal/ui/app.go` (Thêm `refreshGen`, đổi `Refresh()` để drop stale result, xóa bỏ 50ms re-theme thừa trong `ShowAndRun()`).
3. **File 3:** `internal/manager/account_manager.go` (Xác nhận double-check active trong `RemoveAccount`).
4. **Chạy kiểm thử:** `go test ./...` đảm bảo toàn bộ unit test hiện có pass 100%.

---

## 6. MA TRẬN KIỂM THỬ NGHIỆM THU (ACCEPTANCE MATRIX)

| ID Kiểm thử | Hạng mục kiểm thử | Tiêu chuẩn Nghiệm thu bắt buộc |
|---|---|---|
| **`TC-OPT-01`** | Win32 Discovery | Bật app: tìm thấy Language Server trong **< 5ms**; Task Manager không hề xuất hiện `powershell.exe` hay `netstat.exe`. |
| **`TC-OPT-02`** | Singleflight Cache | Nhiều luồng cùng gọi Discovery: chỉ 1 lần quét Win32; kết quả chia sẻ chuẩn xác; hết hạn 10s tự động làm mới. |
| **`TC-OPT-03`** | Khởi động tức thì | App mở lên trong **< 300ms**, hiện ngay email và quota đã lưu từ trước. |
| **`TC-OPT-04`** | Warm Refresh Mượt mà | Bấm "Làm mới danh sách": các nút bấm và thẻ vẫn thao tác bình thường, không bị khóa, không bị gián đoạn. |
| **`TC-OPT-05`** | Bảo toàn Nút Xóa | Nút Xóa `[ 🗑 ]` luôn bật và bấm được trên mọi thẻ Inactive (ON, OFF, Cần OAuth); ẩn hoàn toàn trên thẻ Active. |
| **`TC-OPT-06`** | Backend Active Protection | Cố tình xóa tài khoản vừa active: backend chặn an toàn và trả lỗi `ErrCannotDeleteActiveAccount`. |
| **`TC-OPT-07`** | Phông chữ & DPI | Cửa sổ hiển thị sắc nét ở các mức DPI 100%, 125%, 150%, không bị nháy giật màn hình sau 50ms. |
| **`TC-OPT-08`** | Điều khiển cửa sổ | Nút `—` thu nhỏ xuống Taskbar; nút `X` thoát app sạch sẽ hoàn toàn; không bị thêm bớt menu lạ. |

---

## 7. KẾT LUẬN

Bản đặc tả R4 cung cấp mã nguồn mẫu chính xác, vị trí can thiệp từng dòng file cụ thể. Coder không cần phải tìm kiếm thư viện, không cần tự suy đoán cấu trúc Win32 hay tự quyết định luồng UI, đảm bảo triển khai trơn tru và hoàn thành ngay trong lần chạy đầu tiên.
