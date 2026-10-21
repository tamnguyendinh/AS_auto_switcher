# Báo Cáo Thi Công: Chuyển Đổi Sang Đơn Artifact GUI Duy Nhất (Bước 1 / Phase 1)

- **Ngày thi công:** 2026-10-21
- **Vai trò thực hiện:** Coder
- **Thẩm quyền kiến trúc:** `reports/architect/2026-10-21-single-gui-artifact-migration-spec.md` (Mã văn kiện: `SPEC-ARCH-20261021-SINGLE-GUI-ARTIFACT-MIGRATION`)
- **Git Commit Baseline:** `25da496` (docs(architect): add single-artifact GUI migration specification and 2-step blueprint)
- **Trạng thái:** HOÀN THÀNH 100% TIÊU CHUẨN NGHIỆM THU BƯỚC 1 (PHASE 1 COMPLETE - READY FOR SUPERVISOR / ARCHITECT)

---

## 1. Bản Đồ Họ Bất Biến (Invariant Family Map)

- **Tên họ bất biến:** Single GUI Artifact Pipeline & Zero-Console Entrypoint (`invariant-single-gui-artifact`)
- **Nguồn thẩm quyền (SSOT):** `SPEC-ARCH-20261021-SINGLE-GUI-ARTIFACT-MIGRATION` Mục 1.1, 1.2, 1.3.
- **Các bề mặt thực thi liên quan (Sibling Runtime Surfaces):**
  - Entrypoint ứng dụng: `cmd/switcher/main.go`
  - Đơn bản sao và Single Instance Lock: `ui.TryAcquireSingleInstance`
  - Cơ chế điều phối chuyển tài khoản (Switch Gate): `cmd/switcher/main_test.go`
  - Quy trình tự động hóa kiểm thử & đóng gói: `scripts/build_windows.ps1`
- **Đường dẫn/Cơ chế cũ bị loại bỏ (Forbidden Legacy Fallback):**
  - Khả năng build chế độ dòng lệnh pure Go (`!gui`, `CGO_ENABLED=0`, cờ `--cli`, `--status`, `--list`, `--switch`, `--capture`, `--quota`).
  - Hàm `runCLI`, `runQuotaCLI`, `renderQuotaTable`.
  - Phân nhánh build tags `//go:build !gui` và `//go:build gui`.
  - Cờ `-tags gui` trong lệnh build/test.
- **Artifacts và Tests cũ đã cập nhật/dọn dẹp:**
  - Xóa bỏ `TestFlagQuotaRegistration` và các test format CLI `TestRenderQuotaTable_*`.
  - Bảo tồn và chuẩn hóa các bài test điều phối `TestSwitch_WithGate`, `TestSwitch_GateContention`, `TestSwitch_TwoWayContention`.
- **Ma trận xác minh:**
  - Đường dẫn chính (Primary Path): `pwsh -File .\scripts\build_windows.ps1 -Clean` biên dịch ra duy nhất `bin/antigravity-switch-gui.exe` thành công.
  - Kiểm tra tài nguyên PE: `Verify-BinaryIcon` trích xuất thành công icon `32x32` từ file binary.
  - Kiểm tra đơn bản sao (Single Instance): `main.go` tích hợp `ui.TryAcquireSingleInstance`.
  - Kiểm tra bộ test (Full Test Suite): `go test ./cmd/switcher/... ./internal/...` pass 100%.

---

## 2. Chi Tiết Các Tệp Thay Đổi (Files Changed)

### 2.1 Xóa bỏ tệp dư thừa (Deleted Files)
1. `cmd/switcher/app_cli.go`: Xóa bỏ hoàn toàn chế độ `!gui`.
2. `cmd/switcher/app_gui.go`: Xóa bỏ hoàn toàn do logic khởi chạy GUI và single instance lock đã được sáp nhập trực tiếp vào `main.go`.

### 2.2 Viết lại entrypoint tinh gọn (`cmd/switcher/main.go`)
- Tinh giản từ 346 dòng xuống 45 dòng chuẩn mực.
- Loại bỏ toàn bộ `flag.*`, `tabwriter`, các cờ CLI `--status`, `--list`, `--switch`, `--quota`, `--capture`.
- Tích hợp trực tiếp:
  1. Kiểm tra đơn bản sao `ui.TryAcquireSingleInstance("⚡ Antigravity Multi-Account Switcher")`.
  2. Khởi tạo `store.NewAccountStore("")`, `reloader.NewSessionReloader()`, `manager.NewAccountManager()`.
  3. Khởi tạo và hiển thị GUI `ui.NewApp(mgr) -> appInstance.ShowAndRun()`.

### 2.3 Chuẩn hóa kiểm thử (`cmd/switcher/main_test.go`)
- Gỡ bỏ các bài test gắn liền với cờ CLI đã bị xóa (`TestFlagQuotaRegistration`, `TestRenderQuotaTable_*`).
- Giữ lại 100% và bảo tồn độ bao phủ của logic Gate điều phối switch tài khoản:
  - `TestSwitch_WithGate`: Kiểm thử luồng hoán đổi tài khoản có SwitchGate.
  - `TestSwitch_GateContention`: Kiểm thử chống tranh chấp khi Gate đang bận.
  - `TestSwitch_TwoWayContention`: Kiểm thử tranh chấp 2 chiều giữa manual switch và auto switch.

### 2.4 Đồng bộ hóa kịch bản build (`scripts/build_windows.ps1`)
- Loại bỏ tham số `-Mode` (`cli`, `gui`, `all`).
- Khâu Clean: Kill mọi tiến trình `antigravity-switch*`, xóa sạch file exe và syso cũ trong thư mục đích.
- Khâu Pre-flight: Thiết lập đồng nhất `CGO_ENABLED=1`, chạy `go vet -unsafeptr=false` và `go test` trên toàn bộ các package `cmd/switcher` và `internal`.
- Khâu Compilation: Loại bỏ hàm `Build-CLI`, chỉ chạy `Compile-WindowsResource` (`windres.exe` -> `rsrc_windows_amd64.syso`) và `Build-GUI` tạo ra `bin/antigravity-switch-gui.exe` với cờ `-ldflags="-H windowsgui -s -w"`.
- Khâu Verification Gate: Xác thực trích xuất PE Icon `32x32`.

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
ok  	antigravity-switch/cmd/switcher	1.710s
ok  	antigravity-switch/internal/automation	(cached)
ok  	antigravity-switch/internal/connector	(cached)
ok  	antigravity-switch/internal/manager	2.738s
ok  	antigravity-switch/internal/model	(cached)
ok  	antigravity-switch/internal/reloader	(cached)
ok  	antigravity-switch/internal/scanner	(cached)
ok  	antigravity-switch/internal/store	4.577s
ok  	antigravity-switch/internal/telemetry	(cached)
ok  	antigravity-switch/internal/ui	(cached)
ok  	antigravity-switch/internal/wincred	(cached)
  [PASS] Verification succeeded.
[3/4] Compiling target binaries...
  -> Checking Windows application icon and resource...
  -> Compiling resource with windres: cmd\switcher\app.rc -> cmd\switcher\rsrc_windows_amd64.syso
  [SUCCESS] Compiled PE resource object: cmd\switcher\rsrc_windows_amd64.syso (12552 bytes)
  -> Building Standalone GUI Binary (Fyne, CGO_ENABLED=1)...
  [SUCCESS] Created: bin\antigravity-switch-gui.exe
  Compile Duration: 6086 ms | Size: 27.71 MB (29056512 bytes)
[4/4] Executing quick verification...
  [PASS] Embedded PE application icon verified on bin\antigravity-switch-gui.exe (32x32)
  [INFO] GUI binary built at bin\antigravity-switch-gui.exe.
================================================================================
⚡ BUILD SUCCESSFUL
================================================================================
```

### 3.2 Kiểm tra cấu trúc thư mục đầu ra `bin/`
```powershell
Get-ChildItem -Path bin
```
Kết quả:
```text
    Directory: G:\antigravity_auto_switch\bin

Mode                 LastWriteTime         Length Name
----                 -------------         ------ ----
-a---          10/21/2026  8:50 AM       29056512 antigravity-switch-gui.exe
```
**Xác nhận:** DUY NHẤT 1 file `antigravity-switch-gui.exe`. Tuyệt đối không có file `antigravity-switch.exe`.

### 3.3 Kiểm thử đơn vị không cache (`go test -count=1 ./cmd/switcher/...`)
```text
ok  	antigravity-switch/cmd/switcher	1.575s
```

---

## 4. Đối Chiếu Danh Mục Tiêu Chuẩn Nghiệm Thu Bước 1 (Checklist Alignment)

- [x] Chạy `.\scripts\build_windows.ps1 -Clean` thành công không phát sinh bất kỳ lỗi nào (Exit code 0).
- [x] Thư mục `bin/` chỉ có DUY NHẤT 1 file `antigravity-switch-gui.exe`. Hoàn toàn KHÔNG có file `antigravity-switch.exe`.
- [x] Chạy `go test ./...` pass 100% không cần cờ `-tags gui`.
- [x] Ứng dụng biên dịch với `-H windowsgui -s -w` cùng PE resource COFF object (`rsrc_windows_amd64.syso`) đảm bảo trải nghiệm Zero-Console và icon chuẩn Windows.
- [x] Tuân thủ kỷ luật Bước 1: KHÔNG đổi tên thành `antigravity-switch.exe` trước khi được phê duyệt.
- [x] Bảo tồn 100% lõi nghiệp vụ `internal/`, không chạm vào bất kỳ file nào trong `internal/`.
- [x] Báo cáo thi công chính thức được lập đầy đủ theo định dạng quy định.
