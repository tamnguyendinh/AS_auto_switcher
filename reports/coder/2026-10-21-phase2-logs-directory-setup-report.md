# Báo Cáo Thi Công: Thiết Lập Thư Mục logs/ & Di Dời 3 Tệp Tin Phi Chứng Chỉ (Phase 2)

- **Ngày thi công:** 2026-10-21
- **Vai trò thực hiện:** Coder
- **Thẩm quyền kiến trúc:** `reports/architect/2026-10-21-logs-directory-migration-and-renaming-spec.md` (Mã văn kiện: `SPEC-ARCH-20261021-LOGS-AND-DAILY-QUOTA-SPEC`, Mục II - Giai đoạn 2)
- **Git Commit Baseline:** `23d487f` (docs(supervisor): record Phase 1 telemetry test isolation acceptance report)
- **Trạng thái:** HOÀN THÀNH 100% TIÊU CHUẨN NGHIỆM THU PHASE 2 (PHASE 2 COMPLETE - READY FOR SUPERVISOR)

---

## 1. Bản Đồ Họ Bất Biến (Invariant Family Map)

- **Tên họ bất biến:** Centralized Logs Directory Setup & Non-Credential Files Migration (`invariant-logs-directory-setup-phase2`)
- **Nguồn thẩm quyền (SSOT):** `SPEC-ARCH-20261021-LOGS-AND-DAILY-QUOTA-SPEC` Mục II - Giai đoạn 2 (Phase 2).
- **Các bề mặt thực thi liên quan (Sibling Runtime Surfaces):**
  - Git repository boundary: `.gitignore` bổ sung rule `logs/` ngay dưới `accounts_store/` để ngăn ngừa rò rỉ telemetry / runtime preferences lên repository.
  - Telemetry cache subsystem: `internal/telemetry/cache.go` cập nhật hằng số `DefaultQuotaCachePath = "logs/.quota_cache.json"`.
  - UI preferences subsystem: `internal/ui/themes.go` cập nhật hằng số `DefaultUIPreferencesPath = "logs/.ui_preferences.json"`.
  - GUI runtime wiring: `internal/ui/app.go` hàm `NewAppWithFyne` cập nhật tính toán `logsDir` chuẩn hóa kế thừa từ store root hoặc `logs/`, trỏ đường dẫn cache và preferences trực tiếp vào thư mục `logs/`.
  - File system layout: Thư mục `logs/` được khởi tạo, 3 file phi chứng chỉ (`.quota_cache.json`, `.ui_preferences.json`, `switch_timeline.log`) được di dời an toàn từ `accounts_store/` sang `logs/`.
- **Đường dẫn/Cơ chế cũ bị loại bỏ (Forbidden Legacy Fallback):**
  - Không lưu trữ tệp nhật ký, telemetry cache hoặc cấu hình giao diện bên trong `accounts_store/`.
  - Không viết code fallback/migration phức tạp qua lại giữa các phase (tuân thủ nguyên tắc Zero Legacy Migration Bloat của kiến trúc).
- **Artifacts và Tests cũ đã cập nhật/dọn dẹp:**
  - Cập nhật `.gitignore` ngăn chặn commit runtime files của `logs/`.
  - Xác nhận các unit test trong `internal/telemetry/` và `internal/ui/` hoạt động đồng nhất với các hằng số và đường dẫn mới.
- **Ma trận xác minh (Verify Matrix):**
  - Primary path: `go test ./...` PASS 100%.
  - Package isolation path: `go test -count=1 ./internal/telemetry/... ./internal/ui/...` PASS 100%.
  - Production compile verification: `pwsh -File .\scripts\build_windows.ps1 -Clean` Exit Code 0.
  - File layout verification: `accounts_store/` không còn 3 file phi chứng chỉ; `logs/` lưu trữ đầy đủ 3 file.
  - Script discipline: `scripts/build_windows.ps1` giữ nguyên 100% (0 lines diff).
- **Residual unverified surfaces:** `none`.

---

## 2. Chi Tiết Các Tệp Thay Đổi (Files Changed)

### 2.1 Cập nhật `.gitignore`
- Bổ sung `logs/` ngay bên dưới `accounts_store/`:
  ```gitignore
  accounts_store/
  logs/
  discovered_accounts/
  ```

### 2.2 Di chuyển 3 tệp tin trên hệ thống tệp
- `accounts_store\.quota_cache.json` $\rightarrow$ `logs\.quota_cache.json`
- `accounts_store\.ui_preferences.json` $\rightarrow$ `logs\.ui_preferences.json`
- `accounts_store\switch_timeline.log` $\rightarrow$ `logs\switch_timeline.log`

### 2.3 Cập nhật mã nguồn hệ thống
1. **`internal/telemetry/cache.go` (dòng 16):**
   ```go
   const (
       // DefaultQuotaCachePath is the canonical disk cache file for quota telemetry.
       DefaultQuotaCachePath = "logs/.quota_cache.json"
   )
   ```
2. **`internal/ui/themes.go` (dòng 27):**
   ```go
   const (
       // DefaultUIPreferencesPath is the canonical disk path for UI settings.
       DefaultUIPreferencesPath = "logs/.ui_preferences.json"
   )
   ```
3. **`internal/ui/app.go` (dòng 100–105):**
   ```go
   logsDir := "logs"
   if mgr != nil && mgr.Store() != nil && mgr.Store().GetStoreDir() != "" {
       logsDir = filepath.Join(filepath.Dir(mgr.Store().GetStoreDir()), "logs")
   }
   cachePath := filepath.Join(logsDir, ".quota_cache.json")
   a.preferencesPath = filepath.Join(logsDir, ".ui_preferences.json")
   ```

---

## 3. Bằng Chứng Thực Thi & Xác Minh (Evidence & Benchmarks)

### 3.1 Kiểm tra trạng thái thư mục `logs/` và `accounts_store/`
```powershell
Get-ChildItem logs -Force
# Kết quả:
# - .quota_cache.json (16599 bytes)
# - .ui_preferences.json (47 bytes)
# - switch_timeline.log (2780 bytes)

Get-ChildItem accounts_store -Force
# Kết quả:
# - active_threads_history (thư mục con, chuyển tiếp ở Phase 6)
# - daohien204ccyersin@gmail.com.json
# - hiendao1760@gmail.com.json
# - hokinhdoanh.anvien.sales@gmail.com.json
# - ngau2hung@gmail.com.json
# - quantritaikhoanemail@gmail.com.json
# - skey.miennam@gmail.com.json
# - skey.vanphongmiennam@gmail.com.json
```

### 3.2 Kiểm tra `git check-ignore` đối với thư mục `logs/`
```powershell
git check-ignore -v logs/test.json
# Kết quả: .gitignore:6:logs/    logs/test.json
```

### 3.3 Kết quả chạy kiểm thử tươi (`go test -count=1 ./internal/telemetry/... ./internal/ui/...`)
```text
ok  	antigravity-switch/internal/telemetry	8.147s
ok  	antigravity-switch/internal/ui	27.766s
```

### 3.4 Kết quả chạy toàn bộ test suite (`go test ./...`)
```text
ok  	antigravity-switch/cmd/switcher	(cached)
ok  	antigravity-switch/internal/automation	(cached)
ok  	antigravity-switch/internal/connector	(cached)
ok  	antigravity-switch/internal/manager	(cached)
ok  	antigravity-switch/internal/model	(cached)
ok  	antigravity-switch/internal/reloader	(cached)
ok  	antigravity-switch/internal/scanner	(cached)
ok  	antigravity-switch/internal/store	(cached)
ok  	antigravity-switch/internal/telemetry	(cached)
ok  	antigravity-switch/internal/ui	(cached)
ok  	antigravity-switch/internal/wincred	(cached)
```

### 3.5 Kết quả biên dịch kiểm chứng `pwsh -File .\scripts\build_windows.ps1 -Clean`
```text
================================================================================
⚡ Antigravity Multi-Account Switcher - Windows Build System
   OutDir: bin | SkipTests: False | Clean: True
================================================================================
[1/4] Cleaning output artifacts...
  Removed directory: bin
  Clean complete.
[2/4] Running pre-flight tests & vet...
ok  	antigravity-switch/cmd/switcher	2.071s
ok  	antigravity-switch/internal/automation	(cached)
ok  	antigravity-switch/internal/connector	(cached)
ok  	antigravity-switch/internal/manager	(cached)
ok  	antigravity-switch/internal/model	(cached)
ok  	antigravity-switch/internal/reloader	(cached)
ok  	antigravity-switch/internal/scanner	(cached)
ok  	antigravity-switch/internal/store	14.421s
ok  	antigravity-switch/internal/telemetry	(cached)
ok  	antigravity-switch/internal/ui	27.565s
ok  	antigravity-switch/internal/wincred	(cached)
  [PASS] Verification succeeded.
[3/4] Compiling target binaries...
  -> Checking Windows application icon and resource...
  -> Compiling resource with windres: cmd\switcher\app.rc -> cmd\switcher\rsrc_windows_amd64.syso
  [SUCCESS] Compiled PE resource object: cmd\switcher\rsrc_windows_amd64.syso (12552 bytes)
  -> Building Standalone GUI Binary (Fyne, CGO_ENABLED=1)...
  [SUCCESS] Created: bin\antigravity-switch.exe
  Compile Duration: 5948 ms | Size: 27.73 MB (29073408 bytes)
[4/4] Executing quick verification...
  [PASS] Embedded PE application icon verified on bin\antigravity-switch.exe (32x32)
  [INFO] GUI binary built at bin\antigravity-switch.exe.
================================================================================
⚡ BUILD SUCCESSFUL
================================================================================
```

### 3.6 Kiểm tra toàn vẹn kịch bản build (`scripts/build_windows.ps1`)
```powershell
git diff scripts/build_windows.ps1
# Kết quả: 0 lines diff (rỗng)
```

---

## 4. Kết Luận & Bàn Giao (Conclusion & Handoff Checklist)

- [x] Đã cập nhật `.gitignore` với dòng `logs/` ngay dưới `accounts_store/`.
- [x] Đã tạo thư mục `logs/` và di chuyển 3 file phi chứng chỉ khỏi `accounts_store/`.
- [x] Đã cập nhật `internal/telemetry/cache.go`, `internal/ui/themes.go`, `internal/ui/app.go` theo đúng thiết kế Phase 2.
- [x] `go test ./...` đạt 100% PASS.
- [x] `pwsh -File .\scripts\build_windows.ps1 -Clean` thành công Exit Code 0, tạo `bin\antigravity-switch.exe`.
- [x] Tuyệt đối không làm trước Phase 3, 4, 5, 6.
- [x] Sẵn sàng chuyển giao cho Supervisor nghiệm thu.
