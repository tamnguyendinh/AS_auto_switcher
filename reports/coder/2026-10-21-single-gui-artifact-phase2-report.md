# Báo Cáo Thi Công: Chuyển Đổi Sang Đơn Artifact GUI Duy Nhất (Bước 2 / Phase 2)

- **Ngày thi công:** 2026-10-21
- **Vai trò thực hiện:** Coder
- **Thẩm quyền kiến trúc:** `reports/architect/2026-10-21-single-gui-artifact-migration-spec.md` (Mã văn kiện: `SPEC-ARCH-20261021-SINGLE-GUI-ARTIFACT-MIGRATION`, Phần II)
- **Git Commit Baseline:** `a11c8c7` (docs(architect): approve phase 1 and authorize phase 2 execution)
- **Trạng thái:** HOÀN THÀNH 100% TIÊU CHUẨN NGHIỆM THU BƯỚC 2 (PHASE 2 COMPLETE - READY FOR SUPERVISOR / ARCHITECT)

---

## 1. Bản Đồ Họ Bất Biến (Invariant Family Map)

- **Tên họ bất biến:** Single GUI Artifact Pipeline & Standard Binary Naming (`invariant-single-gui-artifact`)
- **Nguồn thẩm quyền (SSOT):** `SPEC-ARCH-20261021-SINGLE-GUI-ARTIFACT-MIGRATION` Phần II (Mục 2.1, 2.2, 2.3).
- **Các bề mặt thực thi liên quan (Sibling Runtime Surfaces):**
  - Kịch bản build tự động: `scripts/build_windows.ps1` (hàm `Build-GUI` và khâu `Verification Gate`)
  - Tài liệu hướng dẫn sử dụng & kiến trúc: `README.md`
  - Thư mục artifact đầu ra: `bin/antigravity-switch.exe`
- **Đường dẫn/Danh xưng cũ bị loại bỏ (Forbidden Legacy Fallback):**
  - File trung gian `bin/antigravity-switch-gui.exe` (đã được thay thế hoàn toàn bằng `bin/antigravity-switch.exe`).
  - Toàn bộ danh mục lệnh CLI cũ (`--list`, `--status`, `--quota`, `--capture`, `--switch`).
  - Tham số kịch bản build cũ (`-Mode all`, `-Mode cli`, `-Mode gui`).
  - Lệnh build pure Go không CGO (`CGO_ENABLED=0`) và cờ `-tags gui`.
- **Ma trận xác minh:**
  - Đường dẫn chính (Primary Path): `pwsh -File .\scripts\build_windows.ps1 -Clean` biên dịch ra duy nhất `bin/antigravity-switch.exe` thành công (Exit Code 0).
  - Kiểm tra tài nguyên PE: Trích xuất biểu tượng icon `32x32` từ file binary chuẩn Windows PE.
  - Kiểm tra hệ thống con PE (PE Subsystem): Giá trị `2` (`IMAGE_SUBSYSTEM_WINDOWS_GUI`), bảo đảm không nhấp nháy màn hình console đen (Zero-Console).
  - Đơn bản sao (Single Instance Lock): `main.go` điều phối `ui.TryAcquireSingleInstance`.
  - Kiểm tra bộ test (Full Test Suite): `go test ./...` pass 100% với `CGO_ENABLED=1`.

---

## 2. Chi Tiết Các Tệp Thay Đổi (Files Changed)

### 2.1 Chuẩn hóa danh xưng binary trong `scripts/build_windows.ps1`
- Đổi tên tệp đầu ra trong hàm `Build-GUI`:
  ```powershell
  $outFile = Join-Path $TargetDir "antigravity-switch.exe"
  ```
- Đổi tên tệp đích trong khâu Verification Gate:
  ```powershell
  Verify-BinaryIcon -FilePath (Join-Path $OutDir "antigravity-switch.exe")
  Write-Host "  [INFO] GUI binary built at $(Join-Path $OutDir "antigravity-switch.exe")." -ForegroundColor Green
  ```

### 2.2 Cập nhật toàn diện tài liệu `README.md`
- **Gỡ bỏ danh mục CLI cũ:** Xóa bỏ hoàn toàn các hướng dẫn chạy lệnh CLI `--list`, `--status`, `--quota`, `--switch`, `--capture`.
- **Cập nhật mục Khởi chạy ứng dụng:** Hướng dẫn chạy file nhị phân chuẩn `.\bin\antigravity-switch.exe` hoặc click đúp trực tiếp từ Windows File Explorer.
- **Bổ sung mục Kiến trúc & Artifact:** Nêu rõ mô hình đơn artifact duy nhất `antigravity-switch.exe` với cờ `-ldflags="-H windowsgui -s -w"`, tích hợp icon PE COFF và cơ chế Single Instance.
- **Cập nhật hướng dẫn Build từ mã nguồn:** Hướng dẫn sử dụng `.\scripts\build_windows.ps1 -Clean` kèm các tham số mới (`-SkipTests`, `-OutDir`), gỡ bỏ hoàn toàn `-Mode` cũ.
- **Đồng bộ hóa lệnh kiểm thử:** Hướng dẫn chạy `go test ./...` và kiểm thử race condition `go test -v -race -count=1 ./internal/automation/...` đồng nhất với `CGO_ENABLED=1`.

---

## 3. Bằng Chứng Thực Thi & Xác Minh (Evidence & Benchmarks)

### 3.1 Kết quả chạy kịch bản build chuẩn (`scripts/build_windows.ps1 -Clean`)
```text
================================================================================
⚡ Antigravity Multi-Account Switcher - Windows Build System
   OutDir: bin | SkipTests: False | Clean: True
================================================================================
[1/4] Cleaning output artifacts...
  Removed directory: bin
  Clean complete.
[2/4] Running pre-flight tests & vet...
ok  	antigravity-switch/cmd/switcher	1.749s
ok  	antigravity-switch/internal/automation	(cached)
ok  	antigravity-switch/internal/connector	(cached)
ok  	antigravity-switch/internal/manager	4.526s
ok  	antigravity-switch/internal/model	(cached)
ok  	antigravity-switch/internal/reloader	(cached)
ok  	antigravity-switch/internal/scanner	(cached)
ok  	antigravity-switch/internal/store	13.742s
ok  	antigravity-switch/internal/telemetry	(cached)
ok  	antigravity-switch/internal/ui	(cached)
ok  	antigravity-switch/internal/wincred	(cached)
  [PASS] Verification succeeded.
[3/4] Compiling target binaries...
  -> Checking Windows application icon and resource...
  -> Compiling resource with windres: cmd\switcher\app.rc -> cmd\switcher\rsrc_windows_amd64.syso
  [SUCCESS] Compiled PE resource object: cmd\switcher\rsrc_windows_amd64.syso (12552 bytes)
  -> Building Standalone GUI Binary (Fyne, CGO_ENABLED=1)...
  [SUCCESS] Created: bin\antigravity-switch.exe
  Compile Duration: 5814 ms | Size: 27.71 MB (29056512 bytes)
[4/4] Executing quick verification...
  [PASS] Embedded PE application icon verified on bin\antigravity-switch.exe (32x32)
  [INFO] GUI binary built at bin\antigravity-switch.exe.
================================================================================
⚡ BUILD SUCCESSFUL
================================================================================
```

### 3.2 Kiểm tra thư mục đầu ra `bin/` (Đơn artifact duy nhất)
Lệnh kiểm tra:
```powershell
Get-ChildItem -Path bin
```
Kết quả:
```text
    Directory: G:\antigravity_auto_switch\bin

Mode                 LastWriteTime         Length Name
----                 -------------         ------ ----
-a---          10/21/2026  8:59 AM       29056512 antigravity-switch.exe
```
**Xác nhận tuyệt đối:** Thư mục `bin/` chỉ có DUY NHẤT 1 file thực thi `antigravity-switch.exe`. Không còn file `antigravity-switch-gui.exe` hay bất kỳ file exe nào khác.

### 3.3 Xác thực thuộc tính Windows PE & Zero-Console
- **PE Application Icon:**
  ```powershell
  Width  : 32
  Height : 32
  ```
- **PE Subsystem Header Check:**
  ```powershell
  Subsystem: 2 (IMAGE_SUBSYSTEM_WINDOWS_GUI)
  ```
  Xác nhận ứng dụng mở thẳng giao diện đồ họa, triệt tiêu 100% cửa sổ console đen nhấp nháy.

### 3.4 Kiểm thử đơn vị toàn diện (`go test ./...`)
```text
ok  	antigravity-switch/cmd/switcher	1.586s
ok  	antigravity-switch/internal/automation	(cached)
ok  	antigravity-switch/internal/connector	(cached)
ok  	antigravity-switch/internal/manager	4.410s
ok  	antigravity-switch/internal/model	(cached)
ok  	antigravity-switch/internal/reloader	(cached)
ok  	antigravity-switch/internal/scanner	(cached)
ok  	antigravity-switch/internal/store	17.480s
ok  	antigravity-switch/internal/telemetry	(cached)
ok  	antigravity-switch/internal/ui	(cached)
ok  	antigravity-switch/internal/wincred	(cached)
```
Kiểm thử `cmd/switcher` không cache (`go test -count=1 ./cmd/switcher/...`):
```text
ok  	antigravity-switch/cmd/switcher	1.307s
```

---

## 4. Đối Chiếu Danh Mục Tiêu Chuẩn Nghiệm Thu Bước 2 (Checklist Alignment)

- [x] Chạy `.\scripts\build_windows.ps1 -Clean` tạo ra đúng 1 file duy nhất: `bin/antigravity-switch.exe` (Exit Code 0).
- [x] File `bin/antigravity-switch.exe` có kích thước chuẩn (29,056,512 bytes, ~27.71 MB), icon PE 32x32 và PE Subsystem 2 (GUI).
- [x] Trải nghiệm Zero-Console hoàn hảo, không có màn hình đen nhấp nháy.
- [x] Bảo tồn 100% cơ chế Single Instance qua Windows Mutex và lõi nghiệp vụ `internal/`.
- [x] Tài liệu `README.md` được cập nhật đồng bộ, gỡ bỏ toàn bộ dấu vết các lệnh CLI cũ, hướng dẫn sử dụng đơn artifact chuẩn mực.
- [x] Toàn bộ test suite pass 100%.
