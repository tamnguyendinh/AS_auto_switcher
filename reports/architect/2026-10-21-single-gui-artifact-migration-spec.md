# Đặc Tả Kiến Trúc & Quy Trình Chuyển Đổi Sang Đơn Artifact GUI Duy Nhất
(Single-Artifact GUI Migration Specification & Execution Blueprint)

- **Ngày lập:** 2026-10-21
- **Vai trò:** System Architect (Kiến trúc sư Hệ thống)
- **Thẩm quyền:** Chỉ đạo trực tiếp và phê duyệt từ Human Owner
- **Mã văn kiện:** `SPEC-ARCH-20261021-SINGLE-GUI-ARTIFACT-MIGRATION`
- **Trạng thái:** BẢN CHÍNH THỨC BAN HÀNH (OFFICIAL RELEASED)

---

## ⚠️ ĐIỀU KHOẢN KỶ LUẬT SẮT (MANDATORY INVARIANTS)

1. **QUY TRÌNH 2 BƯỚC ĐỘC LẬP BẮT BUỘC (STRICT 2-STEP GATE):**
   - **Tuyệt đối cấm gộp 2 bước vào làm một lúc.** Việc vừa xóa CLI vừa đổi tên file binary cùng lúc bị nghiêm cấm vì sẽ gây tranh chấp file binary cũ (`antigravity-switch.exe`), kẹt tiến trình Windows và lẫn lộn bộ test.
   - **Bước 1 (Phase 1):** Triệt tiêu hoàn toàn CLI, dọn sạch code và cờ CLI trong `main.go`, xóa file `app_cli.go`, đồng bộ `scripts/build_windows.ps1`. Chỉ xuất ra DUY NHẤT 1 file artifact: `bin/antigravity-switch-gui.exe`.
   - **Bước 2 (Phase 2):** Chỉ được phép thực hiện khi Bước 1 đã được kiểm thử thực tế và nghiệm thu PASS 100%. Bước 2 chỉ đơn thuần đổi tên file artifact xuất ra từ `antigravity-switch-gui.exe` $\rightarrow$ `antigravity-switch.exe` và cập nhật tài liệu `README.md`.
2. **BẢO TỒN 100% LÕI NGHIỆP VỤ HỆ THỐNG (`internal/` PRESERVATION):**
   - Toàn bộ các package nghiệp vụ tại `internal/` (`manager`, `store`, `scanner`, `automation`, `telemetry`, `wincred`, `ui`) giữ nguyên vẹn 100%. Không làm thay đổi hay suy giảm bất kỳ logic chuyển đổi tài khoản, quét Chrome hay cache quota nào.
3. **TRẢI NGHIỆM ZERO-CONSOLE:**
   - Ứng dụng phải luôn được biên dịch với cờ `-ldflags="-H windowsgui -s -w"` kèm resource PE (`rsrc_windows_amd64.syso`) để khi người dùng click đúp, ứng dụng mở ngay lập tức mà không có màn hình đen console (CMD) nhấp nháy.

---

## PHẦN I: BƯỚC 1 (PHASE 1) — TRIỆT TIÊU CLI, CHỈ XUẤT ARTIFACT GUI

### 1.1 Mục Tiêu Cốt Lõi
- Xóa bỏ hoàn toàn khả năng biên dịch và sinh ra binary CLI.
- Dọn dẹp sạch sẽ mã nguồn tại `cmd/switcher`, loại bỏ hoàn toàn các cờ CLI, các build tags `//go:build !gui` và `//go:build gui`.
- Cập nhật script `scripts/build_windows.ps1` để toàn bộ quy trình kiểm thử và biên dịch chạy đồng nhất ở chế độ GUI với `CGO_ENABLED=1`.
- Kết quả đầu ra trong `bin/` CHỈ CÓ DUY NHẤT 1 file: `antigravity-switch-gui.exe`.

---

### 1.2 Chỉ Dẫn Chi Tiết Cho Từng File Cần Thay Đổi (Step-by-Step Recipe)

#### File 1: `cmd/switcher/app_cli.go`
- **Hành động:** **XÓA BỎ HOÀN TOÀN TỆP NÀY (DELETE FILE)**.
- **Lý do:** Không còn nhu cầu hỗ trợ chế độ pure Go không có giao diện (`!gui`).

#### File 2: `cmd/switcher/app_gui.go`
- **Hành động:** **XÓA BỎ HOÀN TOÀN TỆP NÀY (DELETE FILE)** (hoặc tích hợp logic vào `main.go`).
- **Lý do:** Logic kiểm tra đơn bản sao và gọi `ui.NewApp` sẽ được đặt trực tiếp bên trong `main.go`. Việc xóa file này giúp xóa sổ hoàn toàn build tag `//go:build gui`, biến `cmd/switcher` thành một package Go chuẩn mực.

#### File 3: `cmd/switcher/main.go`
- **Hành động:** Viết lại toàn bộ tệp để tinh giản tối đa (từ 346 dòng xuống ~45 dòng).
- **Quy cách cụ thể:**
  - Loại bỏ import `flag`, `text/tabwriter`, `context`, `time`, và các package không còn dùng ở entrypoint.
  - Xóa bỏ toàn bộ các biến flag: `flagCLI`, `flagGUI`, `flagSwitch`, `flagStatus`, `flagList`, `flagCapture`, `flagQuota`.
  - Xóa bỏ hoàn toàn các hàm: `runCLI`, `runQuotaCLI`, `renderQuotaTable`.
  - Cấu trúc tệp chuẩn mực thay thế:
    ```go
    package main

    import (
    	"fmt"
    	"os"

    	"antigravity-switch/internal/manager"
    	"antigravity-switch/internal/reloader"
    	"antigravity-switch/internal/store"
    	"antigravity-switch/internal/ui"
    )

    func main() {
    	// 1. Kiểm tra đơn bản sao (Single Instance Lock)
    	lock, isFirst := ui.TryAcquireSingleInstance("⚡ Antigravity Multi-Account Switcher")
    	if !isFirst {
    		// Ứng dụng đã đang chạy, cửa sổ cũ đã được đưa lên trên màn hình -> Thoát bản sao mới ngay
    		os.Exit(0)
    	}
    	defer lock.Release()

    	// 2. Khởi tạo AccountStore & Reloader & Manager
    	accStore, err := store.NewAccountStore("")
    	if err != nil {
    		fmt.Fprintf(os.Stderr, "Error initializing account store: %v\n", err)
    		os.Exit(1)
    	}

    	sessionReloader := reloader.NewSessionReloader()
    	mgr, err := manager.NewAccountManager(accStore, sessionReloader)
    	if err != nil {
    		fmt.Fprintf(os.Stderr, "Error initializing account manager: %v\n", err)
    		os.Exit(1)
    	}

    	// 3. Khởi tạo và khởi chạy giao diện GUI Fyne
    	appInstance, err := ui.NewApp(mgr)
    	if err != nil {
    		fmt.Fprintf(os.Stderr, "Failed launching GUI: %v\n", err)
    		os.Exit(1)
    	}
    	appInstance.ShowAndRun()
    }
    ```

#### File 4: `cmd/switcher/main_test.go`
- **Hành động:** Dọn dẹp các bài test CLI format và bảo tồn các bài test nghiệp vụ Gate.
- **Quy cách cụ thể:**
  - Xóa bỏ `TestFlagQuotaRegistration` (do cờ quota đã bị xóa).
  - Xóa bỏ các bài test `TestRenderQuotaTable_EmptyCache`, `TestRenderQuotaTable_PopulatedAndStatuses`, `TestRenderQuotaTable_ActiveLowQuotaStatus` (do hàm `renderQuotaTable` dạng text table đã bị xóa).
  - Giữ lại và cập nhật các bài test:
    - `TestCLI_Switch_WithGate` (có thể đổi tên thành `TestSwitch_WithGate`).
    - `TestCLI_Switch_GateContention` (đổi tên thành `TestSwitch_GateContention`).
    - `TestCLI_Switch_TwoWayContention` (đổi tên thành `TestSwitch_TwoWayContention`).
    - Các bài test này trực tiếp kiểm tra `automation.ExecuteSwitchWithGate` và `automation.NewSwitchGate`, bảo đảm không suy giảm độ phủ kiểm thử của nghiệp vụ Gate.

#### File 5: `scripts/build_windows.ps1`
- **Hành động:** Cập nhật đồng bộ toàn diện để kịch bản build chạy pass ngay lập tức.
- **Quy cách cụ thể:**
  1. **Khâu Tham số (Param):**
     - Đổi thành:
       ```powershell
       [CmdletBinding()]
       param(
           [switch]$SkipTests,
           [switch]$Clean,
           [string]$OutDir = "bin"
       )
       ```
     - Xóa bỏ tham số `[ValidateSet("cli", "gui", "all")]$Mode`.
  2. **Khâu Clean Stage:**
     - Giữ nguyên cơ chế kill process `antigravity-switch*`.
     - Lệnh xóa file: Dọn sạch mọi file `antigravity-switch*.exe` (xóa cả file CLI cũ nếu còn sót lại) để đảm bảo `bin/` sạch tuyệt đối.
  3. **Khâu Pre-Flight Tests:**
     - Xóa bỏ hoàn toàn khối `if ($Mode -eq "cli" ...)` và việc ép `$env:CGO_ENABLED = "0"`.
     - Thiết lập một khối test duy nhất:
       ```powershell
       if (-not $SkipTests) {
           Write-Host "[2/4] Running pre-flight tests & vet..." -ForegroundColor Yellow
           $env:CGO_ENABLED = "1"
           
           # go vet
           go vet ./cmd/switcher/... ./internal/...
           if ($LASTEXITCODE -ne 0) {
               Write-Error "go vet failed"
           }

           # go test
           go test ./cmd/switcher/... ./internal/...
           if ($LASTEXITCODE -ne 0) {
               Write-Error "Test suite failed"
           }
           Write-Host "  [PASS] Verification succeeded." -ForegroundColor Green
       } else {
           Write-Host "[2/4] Skipping tests (-SkipTests specified)." -ForegroundColor DarkGray
       }
       ```
  4. **Khâu Compilation Pipeline:**
     - Xóa bỏ hoàn toàn hàm `Build-CLI`.
     - Giữ hàm `Compile-WindowsResource` (biên dịch `app.rc` thành `rsrc_windows_amd64.syso`).
     - Hàm `Build-GUI`:
       - Biên dịch với lệnh:
         ```powershell
         go build -ldflags="-H windowsgui -s -w" -o $outFile .\cmd\switcher
         ```
         *(Chú ý: Bỏ cờ `-tags gui` vì mã nguồn không còn dùng build tag này)*.
       - Tệp đầu ra ở Bước 1: `$outFile = Join-Path $TargetDir "antigravity-switch-gui.exe"`.
     - Luồng thực thi: Gọi trực tiếp `Build-GUI -TargetDir $OutDir`.
  5. **Khâu Verification Gate:**
     - Xóa bỏ lệnh tìm kiếm và chạy thử `& $verifyExe --status`.
     - Chỉ gọi xác thực icon:
       ```powershell
       Verify-BinaryIcon -FilePath (Join-Path $OutDir "antigravity-switch-gui.exe")
       Write-Host "  [INFO] GUI binary built at $(Join-Path $OutDir "antigravity-switch-gui.exe")." -ForegroundColor Green
       ```

---

### 1.3 Tiêu Chuẩn Nghiệm Thu Bước 1 (Acceptance Checklist Phase 1)
Coder chỉ được phép tuyên bố hoàn thành Bước 1 khi thỏa mãn 100% các tiêu chí:
- [ ] Chạy `.\scripts\build_windows.ps1 -Clean` thành công mà không phát sinh bất kỳ lỗi nào.
- [ ] Kiểm tra thư mục `bin/`: **Chỉ có DUY NHẤT 1 file là `antigravity-switch-gui.exe`**. Hoàn toàn KHÔNG có file `antigravity-switch.exe`.
- [ ] Chạy `go test ./...` pass 100% không cần cờ `-tags gui`.
- [ ] Chạy file `bin/antigravity-switch-gui.exe`:
  - Mở thẳng giao diện GUI Fyne.
  - Không nhấp nháy cửa sổ console đen.
  - Đầy đủ các tính năng: danh sách tài khoản, quota card, nút gạt ẩn tài khoản, nút gạt tự động, menu khay hệ thống và ghim cửa sổ.
  - Icon ứng dụng hiển thị sắc nét, chuẩn xác trên thanh Taskbar và góc cửa sổ.

---

## PHẦN II: BƯỚC 2 (PHASE 2) — CHUẨN HÓA DANH XƯNG THÀNH `antigravity-switch.exe`

> ⚠️ **ĐIỀU KIỆN TIÊN QUYẾT:** Chỉ được bắt đầu Bước 2 sau khi Bước 1 đã vượt qua toàn bộ tiêu chuẩn nghiệm thu và được xác nhận PASS.

### 2.1 Mục Tiêu Cốt Lõi
- Đổi tên file nhị phân chính thức của ứng dụng từ `antigravity-switch-gui.exe` thành `antigravity-switch.exe`.
- Cập nhật tài liệu hướng dẫn để người dùng và nhà phát triển chỉ thao tác với duy nhất 1 file thực thi chuẩn mực.

---

### 2.2 Chỉ Dẫn Chi Tiết Cho Từng File Cần Thay Đổi

#### File 1: `scripts/build_windows.ps1`
- **Hành động:** Đổi tên file đích từ `antigravity-switch-gui.exe` sang `antigravity-switch.exe`.
- **Vị trí cụ thể:**
  - Trong hàm `Build-GUI`:
    ```powershell
    $outFile = Join-Path $TargetDir "antigravity-switch.exe"
    ```
  - Trong khâu Verification Gate:
    ```powershell
    Verify-BinaryIcon -FilePath (Join-Path $OutDir "antigravity-switch.exe")
    Write-Host "  [INFO] GUI binary built at $(Join-Path $OutDir "antigravity-switch.exe")." -ForegroundColor Green
    ```

#### File 2: `README.md`
- **Hành động:** Cập nhật lại hướng dẫn khởi chạy và sử dụng.
- **Quy cách cụ thể:**
  - Gỡ bỏ toàn bộ danh mục các câu lệnh CLI cũ (`.\bin\antigravity-switch.exe --list`, `--status`, `--quota`, `--switch`, `--capture`).
  - Cập nhật mục Build & Chạy ứng dụng:
    ```markdown
    ### Khởi Chạy Ứng Dụng
    Khởi chạy ứng dụng đồ họa trực quan:
    .\bin\antigravity-switch.exe
    ```
  - Cập nhật mục Kiến trúc / Artifact: Nêu rõ ứng dụng chỉ có 1 artifact duy nhất là `antigravity-switch.exe` (Fyne GUI với cờ `-H windowsgui`).

---

### 2.3 Tiêu Chuẩn Nghiệm Thu Bước 2 (Acceptance Checklist Phase 2)
- [ ] Chạy `.\scripts\build_windows.ps1 -Clean` tạo ra đúng 1 file duy nhất: `bin/antigravity-switch.exe`.
- [ ] File `bin/antigravity-switch.exe` có kích thước chuẩn (~25MB), icon PE chuẩn.
- [ ] Double-click mở thẳng GUI hoàn hảo, không có màn hình đen console.
- [ ] Kiểm tra cơ chế Single Instance: Khi ứng dụng đang chạy, click đúp file `antigravity-switch.exe` lần thứ 2 thì không sinh tiến trình mới, cửa sổ đang mở được tự động kéo lên phía trước màn hình.
- [ ] Tài liệu `README.md` được cập nhật đồng bộ, không còn dấu vết của các lệnh CLI cũ.

---

## PHẦN III: HƯỚNG DẪN BÀN GIAO CHO CODER (HANDOFF CONTRACT)

Khi CEO giao việc cho Coder, Coder phải tuân thủ nghiêm ngặt các mốc:
1. **Giai đoạn 1:** Chỉ thực hiện đúng các mục trong **Phần I (Bước 1)**. Không được tự ý đổi tên file sang `antigravity-switch.exe` ở giai đoạn này. Sau khi Bước 1 pass và được kiểm chứng, mới báo cáo hoàn tất Bước 1.
2. **Giai đoạn 2:** Nhận lệnh thực hiện **Phần II (Bước 2)**. Đổi tên file và cập nhật tài liệu. Nghiệm thu lần cuối.

Văn bản đặc tả này là kim chỉ nam tối cao cho việc tái cấu trúc artifact, mọi hành vi làm sai quy trình 2 bước trên đều bị coi là vi phạm kỷ luật kiến trúc hệ thống.