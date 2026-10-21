# Báo Cáo Thi Công: Đổi Tên Chuẩn Hóa Tệp Tin (Loại Bỏ Dấu Chấm Ẩn) Thành quota_log.json & ui_preferences.json (Phase 3)

- **Ngày thi công:** 2026-10-21
- **Vai trò thực hiện:** Coder
- **Thẩm quyền kiến trúc:** `reports/architect/2026-10-21-logs-directory-migration-and-renaming-spec.md` (Mã văn kiện: `SPEC-ARCH-20261021-LOGS-AND-DAILY-QUOTA-SPEC`, Mục II - Giai đoạn 3)
- **Git Commit Baseline:** `7543284` (docs(supervisor): record Phase 2 logs directory setup and migration acceptance report)
- **Trạng thái:** HOÀN THÀNH 100% TIÊU CHUẨN NGHIỆM THU PHASE 3 (PHASE 3 COMPLETE - READY FOR SUPERVISOR)

---

## 1. Bản Đồ Họ Bất Biến (Invariant Family Map)

- **Tên họ bất biến:** Standardized Non-Hidden Log and Preference Filenames (`invariant-standardize-filenames-phase3`)
- **Nguồn thẩm quyền (SSOT):** `SPEC-ARCH-20261021-LOGS-AND-DAILY-QUOTA-SPEC` Mục II - Giai đoạn 3 (Phase 3).
- **Các bề mặt thực thi liên quan (Sibling Runtime Surfaces):**
  - Hệ thống tệp trên đĩa (`logs/` directory):
    - Đổi tên `logs/.ui_preferences.json` thành `logs/ui_preferences.json`.
    - Đổi tên `logs/.quota_cache.json` thành `logs/quota_log.json`.
  - Hằng số Telemetry Cache Subsystem (`internal/telemetry/cache.go`):
    - Cập nhật hằng số `DefaultQuotaCachePath = "logs/quota_log.json"`.
  - Hằng số UI Preferences Subsystem (`internal/ui/themes.go`):
    - Cập nhật hằng số `DefaultUIPreferencesPath = "logs/ui_preferences.json"`.
  - Runtime Wiring Application GUI (`internal/ui/app.go`):
    - Cập nhật hàm `NewAppWithFyne`:
      - `cachePath := filepath.Join(logsDir, "quota_log.json")`
      - `a.preferencesPath = filepath.Join(logsDir, "ui_preferences.json")`
  - Automated Tests Suite:
    - `internal/telemetry/telemetry_test.go`: Bổ sung kiểm tra trực tiếp hằng số `DefaultQuotaCachePath == "logs/quota_log.json"`.
    - `internal/telemetry/quota_test.go`: Đồng bộ đường dẫn cache trong test thành `quota_log.json`.
    - `internal/ui/themes_test.go`: Bổ sung test `TestDefaultUIPreferencesPath` kiểm tra `DefaultUIPreferencesPath == "logs/ui_preferences.json"`, cập nhật đường dẫn test thành `ui_preferences.json` và `quota_log.json`.
    - `internal/ui/ui_test.go`: Cập nhật các đường dẫn mock preferences trong unit tests thành `ui_preferences.json`.
- **Đường dẫn/Cơ chế cũ bị loại bỏ (Forbidden Legacy Fallback):**
  - Không sử dụng tên tệp có dấu chấm ẩn (`.ui_preferences.json`, `.quota_cache.json`) trong môi trường Windows.
  - Cắt bỏ mọi mã fallback migration rườm rà (mã nguồn trỏ trực tiếp 100% vào tên chuẩn mới, tuân thủ nghiêm ngặt nguyên tắc Zero Legacy Migration Bloat).
- **Artifacts và Tests cũ đã cập nhật/dọn dẹp:**
  - Đồng bộ toàn bộ các unit test trong `internal/` sử dụng tên file chuẩn hóa không có dấu chấm ẩn.
- **Ma trận xác minh (Verify Matrix):**
  - Primary path: `go test ./... -count=1` PASS 100%.
  - Clean build verification: `pwsh -File .\scripts\build_windows.ps1 -Clean` Exit Code 0.
  - File layout verification: `logs/` chỉ chứa 3 file chuẩn hóa (`quota_log.json`, `switch_timeline.log`, `ui_preferences.json`), hoàn toàn không còn file có dấu chấm ẩn.
  - Storage pureness: `accounts_store/` giữ nguyên cấu trúc từ Phase 2, không bị ảnh hưởng.
  - Script discipline: `scripts/build_windows.ps1` giữ nguyên 100% (0 lines diff).
- **Residual unverified surfaces:** `none`.

---

## 2. Chi Tiết Các Thay Đổi (Changes Implemented)

### 2.1 Đổi tên tệp trên đĩa trong `logs/`
Thực thi lệnh PowerShell theo đúng SPEC:
```powershell
Rename-Item -Path "logs\.ui_preferences.json" -NewName "ui_preferences.json"
Rename-Item -Path "logs\.quota_cache.json" -NewName "quota_log.json"
```

### 2.2 Cập nhật mã nguồn hệ thống

1. **`internal/telemetry/cache.go` (dòng 16):**
   ```go
   const (
       // DefaultQuotaCachePath is the canonical disk cache file for quota telemetry.
       DefaultQuotaCachePath = "logs/quota_log.json"
   )
   ```

2. **`internal/ui/themes.go` (dòng 27):**
   ```go
   const (
       // DefaultUIPreferencesPath is the canonical disk path for UI settings.
       DefaultUIPreferencesPath = "logs/ui_preferences.json"
   )
   ```

3. **`internal/ui/app.go` (dòng 104–105):**
   ```go
   cachePath := filepath.Join(logsDir, "quota_log.json")
   a.preferencesPath = filepath.Join(logsDir, "ui_preferences.json")
   ```

### 2.3 Cập nhật Unit Tests để đồng bộ và khóa cứng bất biến mới
1. **`internal/telemetry/telemetry_test.go`:** Khóa cứng kiểm tra `DefaultQuotaCachePath == "logs/quota_log.json"`.
2. **`internal/telemetry/quota_test.go`:** Cập nhật mock file name thành `quota_log.json`.
3. **`internal/ui/themes_test.go`:** Thêm `TestDefaultUIPreferencesPath`, đổi tên mock file thành `ui_preferences.json` và `quota_log.json`.
4. **`internal/ui/ui_test.go`:** Cập nhật mock path trong `TestGlobalQuotaRefreshAndIndicator`, `TestUITraceabilityChecklist`, `TestArchitectSpec_Row2Layout_ThemeSync`, `TestArchitectSpec_MaskAccounts_Persistence`.

---

## 3. Bằng Chứng Thực Thi & Xác Minh (Evidence & Benchmarks)

### 3.1 Kiểm tra trạng thái thư mục `logs/` và `accounts_store/`
```powershell
Get-ChildItem -Path logs -Force
```
**Kết quả thực tế:**
```text
Directory: G:\antigravity_auto_switch\logs

Mode                 LastWriteTime         Length Name
----                 -------------         ------ ----
-a---          10/21/2026  2:38 PM          16599 quota_log.json
-a---          10/17/2026 10:57 PM           2780 switch_timeline.log
-a---          10/21/2026  1:23 PM             47 ui_preferences.json
```
*(Xác nhận: Không còn bất kỳ tệp ẩn có dấu chấm nào trong thư mục `logs/`).*

```powershell
Get-ChildItem -Path accounts_store -Force
```
**Kết quả thực tế:**
```text
Directory: G:\antigravity_auto_switch\accounts_store

Mode                 LastWriteTime         Length Name
----                 -------------         ------ ----
d----          10/21/2026  1:24 PM                active_threads_history
-a---          10/20/2026  8:13 PM           1708 daohien204ccyersin@gmail.com.json
-a---          10/18/2026 10:30 AM           1834 hiendao1760@gmail.com.json
-a---          10/18/2026 11:40 PM           1613 hokinhdoanh.anvien.sales@gmail.com.json
-a---          10/19/2026  1:29 PM           1580 ngau2hung@gmail.com.json
-a---          10/21/2026 10:47 AM           1605 quantritaikhoanemail@gmail.com.json
-a---          10/21/2026  1:24 PM           1587 skey.miennam@gmail.com.json
-a---          10/21/2026  1:23 PM           1605 skey.vanphongmiennam@gmail.com.json
```
*(Xác nhận: Kho tài khoản giữ nguyên 100% tính toàn vẹn).*

### 3.2 Kiểm tra kiểm thử tự động toàn diện (`go test ./... -count=1`)
```powershell
go test ./... -count=1
```
**Kết quả thực tế:**
```text
ok  	antigravity-switch/cmd/switcher	4.500s
ok  	antigravity-switch/internal/automation	23.310s
ok  	antigravity-switch/internal/connector	4.471s
ok  	antigravity-switch/internal/manager	7.460s
ok  	antigravity-switch/internal/model	0.885s
ok  	antigravity-switch/internal/reloader	4.352s
ok  	antigravity-switch/internal/scanner	1.201s
ok  	antigravity-switch/internal/store	9.776s
ok  	antigravity-switch/internal/telemetry	18.321s
ok  	antigravity-switch/internal/ui	25.907s
ok  	antigravity-switch/internal/wincred	1.099s
?   	antigravity-switch/reports/supervisor/evidence/autonomous_switch_runtime_review_261020	[no test files]
```
*(Xác nhận: 100% packages PASS).*

### 3.3 Kiểm tra biên dịch hệ thống (`pwsh -File .\scripts\build_windows.ps1 -Clean`)
```powershell
pwsh -File .\scripts\build_windows.ps1 -Clean
```
**Kết quả thực tế:**
```text
================================================================================
⚡ Antigravity Multi-Account Switcher - Windows Build System
   OutDir: bin | SkipTests: False | Clean: True
================================================================================
[1/4] Cleaning output artifacts...
  Removed directory: bin
  Clean complete.
[2/4] Running pre-flight tests & vet...
ok  	antigravity-switch/cmd/switcher	2.425s
ok  	antigravity-switch/internal/automation	16.888s
ok  	antigravity-switch/internal/connector	(cached)
ok  	antigravity-switch/internal/manager	(cached)
ok  	antigravity-switch/internal/model	(cached)
ok  	antigravity-switch/internal/reloader	(cached)
ok  	antigravity-switch/internal/scanner	(cached)
ok  	antigravity-switch/internal/store	8.249s
ok  	antigravity-switch/internal/telemetry	11.573s
ok  	antigravity-switch/internal/ui	22.533s
ok  	antigravity-switch/internal/wincred	(cached)
  [PASS] Verification succeeded.
[3/4] Compiling target binaries...
  -> Checking Windows application icon and resource...
  -> Compiling resource with windres: cmd\switcher\app.rc -> cmd\switcher\rsrc_windows_amd64.syso
  [SUCCESS] Compiled PE resource object: cmd\switcher\rsrc_windows_amd64.syso (12552 bytes)
  -> Building Standalone GUI Binary (Fyne, CGO_ENABLED=1)...
  [SUCCESS] Created: bin\antigravity-switch.exe
  Compile Duration: 6633 ms | Size: 27.73 MB (29073408 bytes)
[4/4] Executing quick verification...
  [PASS] Embedded PE application icon verified on bin\antigravity-switch.exe (32x32)
  [INFO] GUI binary built at bin\antigravity-switch.exe.
================================================================================
⚡ BUILD SUCCESSFUL
================================================================================
```
*(Xác nhận: Exit Code 0, binary độc lập hợp lệ).*

### 3.4 Kiểm tra kỷ luật bảo toàn script build (`git diff scripts/build_windows.ps1`)
```powershell
git diff scripts/build_windows.ps1
```
**Kết quả:** Rỗng (0 lines diff).

---

## 4. Kết Luận & Bàn Giao (Handoff Verdict)

- **Đánh giá:** Hoàn thành 100% mục tiêu Giai đoạn 3 (Phase 3) theo đúng văn kiện kiến trúc `SPEC-ARCH-20261021-LOGS-AND-DAILY-QUOTA-SPEC`.
- **Verdict:** `READY_FOR_SUPERVISOR` (PASS Phase 3).
- **Handoff:** Chuyển giao hồ sơ và kết quả cho Supervisor Lane nghiệm thu Phase 3 độc lập trước khi chuyển sang Phase 4.
