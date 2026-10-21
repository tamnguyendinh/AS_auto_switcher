# Báo Cáo Thi Công: Tích Hợp Nhật Ký Chuyển Tài Khoản Vào SwitchAccount & Kích Hoạt Dọn Dẹp Nền 7 Ngày (Phase 2)

- **Ngày thi công:** 2026-10-21
- **Vai trò thực hiện:** Coder
- **Thẩm quyền kiến trúc:** `reports/architect/2026-10-21-daily-switch-timeline-sharding-and-retention-spec.md` (Mã văn kiện: `SPEC-ARCH-20261021-DAILY-SWITCH-TIMELINE-SPEC`, Commit: `d57c6e2`)
- **Trạng thái:** HOÀN THÀNH 100% TIÊU CHUẨN NGHIỆM THU PHASE 2 (PHASE 2 COMPLETE - READY FOR SUPERVISOR)

---

## 1. Bản Đồ Họ Bất Biến (Invariant Family Map)

- **Tên họ bất biến:** Daily Switch Timeline Sharding & 7-Day Rolling Retention (`invariant-daily-switch-timeline-phase2`)
- **Nguồn thẩm quyền (SSOT):** `SPEC-ARCH-20261021-DAILY-SWITCH-TIMELINE-SPEC` Mục I (Invariants), Mục II (Layout), Mục III (Bước 3 & Bước 4), Mục IV (Test Matrix Mục 4).
- **Các bề mặt thực thi liên quan (Sibling Runtime Surfaces):**
  - **Quy trình hoán đổi tài khoản (`internal/manager/account_manager.go` - `SwitchAccount`):**
    - Xác định thư mục ghi timeline động dựa theo `m.store.GetStoreDir()`:
      ```go
      timelineDir := DefaultSwitchTimelineDir
      if m.store != nil && m.store.GetStoreDir() != "" {
          timelineDir = filepath.Join(filepath.Dir(m.store.GetStoreDir()), "logs", "switch_timeline")
      }
      ```
    - Ghi nhận đầy đủ các mốc sự kiện fail-safe với thời gian tương đối `time.Since(start)`:
      - `SWITCH_START`: Ngay khi bắt đầu chu trình hoán đổi tài khoản (bỏ qua nếu no-op).
      - `SNAPSHOT_DONE`: Sau khi chụp snapshot các active thread.
      - `WINCRED_WRITE`: Sau khi ghi token mới vào Windows Credential Manager.
      - `KILL_LS`: Sau khi khởi động lại Language Server (recycle).
      - `RAM_CHECK_DONE`: Sau khi xác thực danh tính Language Server mới qua RAM (`GetUserStatus`).
      - `DISPATCH_START`: Bắt đầu gửi prompt tiếp tục phiên vào các thread (khi có active threads).
      - `DISPATCH_DONE`: Gửi tiếp tục phiên hoàn tất.
      - `SWITCH_SUCCESS`: Trước khi trả về kết quả thành công hoàn tất.
      - `SWITCH_FAILED`: Bọc qua helper closure `failSwitch` đảm bảo 100% các nhánh lỗi (invalid target, token not found, wincred error, recycle error, client nil, RAM status error, RAM mismatch) đều được ghi nhận fail-safe.
  - **Goroutine dọn dẹp nền khi khởi động ứng dụng (`internal/ui/app.go` - `NewAppWithFyne`):**
    - Kích hoạt song song dọn dẹp log quota và log switch timeline cũ hơn 7 ngày trong goroutine nền nhẹ non-blocking:
      ```go
      go func() {
          quotaDir := filepath.Join(logsDir, "quota_log")
          _, _ = telemetry.PruneDailyQuotaLogs(quotaDir, telemetry.DefaultQuotaLogRetentionDays)

          timelineDir := filepath.Join(logsDir, "switch_timeline")
          _, _ = manager.PruneDailySwitchTimelineLogs(timelineDir, manager.DefaultSwitchTimelineRetentionDays)
      }()
      ```
  - **Bài kiểm thử tích hợp (`internal/manager/timeline_test.go` - `TestSwitchAccount_WritesDailyTimeline`):**
    - Khởi tạo môi trường kiểm thử hermetic hoàn chỉnh với `AccountStore`, `SessionReloader`, mock server language server, credential handler.
    - Chạy `SwitchAccount` thành công và xác thực tệp `logs/switch_timeline/YYYY-MM-DD_switch_timeline.log` sinh ra với đầy đủ các tag: `SWITCH_START`, `SNAPSHOT_DONE`, `WINCRED_WRITE`, `KILL_LS`, `RAM_CHECK_DONE`, `SWITCH_SUCCESS`.
    - Chạy `SwitchAccount` với tài khoản không tồn tại và xác thực tag `SWITCH_FAILED` được ghi nhận.
- **Đường dẫn/Cơ chế cũ bị loại bỏ (Forbidden Legacy Fallback):**
  - Tuyệt đối không còn bất kỳ dòng code nào ghi ra tệp log mồ côi `logs/switch_timeline.log`.
  - Mọi sự kiện đều phân mảnh theo ngày ISO `YYYY-MM-DD_switch_timeline.log` đặt trong `logs/switch_timeline/`.
- **Ranh giới cô lập (Boundary Isolation & Non-Goals):**
  - Không sửa đổi bất kỳ tệp nào ngoài slice được phân công (`internal/manager/account_manager.go`, `internal/ui/app.go`, `internal/manager/timeline_test.go`).
  - `scripts/build_windows.ps1` giữ nguyên 100% (0 lines diff).
  - Không dùng anvien, không spawn subagent.
- **Residual unverified surfaces:** `none`.

---

## 2. Chi Tiết Thay Đổi Mã Nguồn (Code Changes)

### 2.1 Tích hợp ghi nhật ký vào `SwitchAccount` (`internal/manager/account_manager.go`)
- Thêm cơ chế nhận diện `timelineDir` tự động trỏ về thư mục `logs/switch_timeline` tương ứng với thư mục gốc của kho dữ liệu `accounts_store`.
- Bổ sung helper closure `failSwitch` đóng gói việc ghi `SWITCH_FAILED` trước khi trả về lỗi.
- Đặt các lệnh gọi `LogSwitchEvent` tại đúng các mốc chuyển tiếp trọng yếu theo Mục III.Bước 3 của đặc tả.
- Đảm bảo tính chất fail-safe: lỗi ghi đĩa tuyệt đối không ảnh hưởng đến luồng chuyển tài khoản.

### 2.2 Kích hoạt dọn dẹp 7 ngày khi app khởi động (`internal/ui/app.go`)
- Trong `NewAppWithFyne`, cập nhật goroutine nền nhẹ để gọi `manager.PruneDailySwitchTimelineLogs(timelineDir, manager.DefaultSwitchTimelineRetentionDays)`.
- Chạy nền hoàn toàn, không chặn luồng khởi tạo giao diện người dùng.

### 2.3 Thêm bài kiểm thử tích hợp `TestSwitchAccount_WritesDailyTimeline` (`internal/manager/timeline_test.go`)
- Bổ sung bài test kiểm tra chu trình hoàn chỉnh của `SwitchAccount`:
  1. Kiểm tra thành công tạo tệp nhật ký ngày hôm nay.
  2. Xác minh sự hiện diện của các mốc sự kiện: `SWITCH_START`, `SNAPSHOT_DONE`, `WINCRED_WRITE`, `KILL_LS`, `RAM_CHECK_DONE`, `SWITCH_SUCCESS`.
  3. Kiểm tra trường hợp thất bại ghi nhận `SWITCH_FAILED`.

---

## 3. Kết Quả Xác Minh Thực Tế (Verification Evidence)

### 3.1 Kiểm thử tích hợp chuyên biệt (`TestSwitchAccount_WritesDailyTimeline`)
Lệnh thực thi:
```powershell
go test -v ./internal/manager -run TestSwitchAccount_WritesDailyTimeline
```
Kết quả:
```text
=== RUN   TestSwitchAccount_WritesDailyTimeline
--- PASS: TestSwitchAccount_WritesDailyTimeline (0.15s)
PASS
ok  	antigravity-switch/internal/manager	1.999s
```

### 3.2 Kiểm thử toàn bộ gói `internal/manager`
Lệnh thực thi:
```powershell
go test -v ./internal/manager
```
Kết quả: PASS 100% (28/28 tests passed).

### 3.3 Kiểm thử toàn bộ dự án (`go test ./...`)
Lệnh thực thi:
```powershell
go test ./...
```
Kết quả:
```text
ok  	antigravity-switch/cmd/switcher	1.813s
ok  	antigravity-switch/internal/automation	15.328s
ok  	antigravity-switch/internal/connector	(cached)
ok  	antigravity-switch/internal/manager	5.964s
ok  	antigravity-switch/internal/model	(cached)
ok  	antigravity-switch/internal/reloader	(cached)
ok  	antigravity-switch/internal/scanner	(cached)
ok  	antigravity-switch/internal/store	11.887s
ok  	antigravity-switch/internal/telemetry	(cached)
ok  	antigravity-switch/internal/ui	26.075s
ok  	antigravity-switch/internal/wincred	(cached)
```

### 3.4 Biên dịch kiểm chứng hệ thống (`build_windows.ps1`)
Lệnh thực thi:
```powershell
pwsh -File .\scripts\build_windows.ps1 -Clean
```
Kết quả:
```text
================================================================================
⚡ Antigravity Multi-Account Switcher - Windows Build System
   OutDir: bin | SkipTests: False | Clean: True
================================================================================
[1/4] Cleaning output artifacts...
  Removed directory: bin
  Clean complete.
[2/4] Running pre-flight tests & vet...
  [PASS] Verification succeeded.
[3/4] Compiling target binaries...
  -> Checking Windows application icon and resource...
  -> Compiling resource with windres: cmd\switcher\app.rc -> cmd\switcher\rsrc_windows_amd64.syso
  [SUCCESS] Compiled PE resource object: cmd\switcher\rsrc_windows_amd64.syso (12552 bytes)
  -> Building Standalone GUI Binary (Fyne, CGO_ENABLED=1)...
  [SUCCESS] Created: bin\antigravity-switch.exe
  Compile Duration: 5593 ms | Size: 27.74 MB (29088768 bytes)
[4/4] Executing quick verification...
  [PASS] Embedded PE application icon verified on bin\antigravity-switch.exe (32x32)
  [INFO] GUI binary built at bin\antigravity-switch.exe.
================================================================================
⚡ BUILD SUCCESSFUL
================================================================================
```
Exit Code: 0.

### 3.5 Kiểm tra tính nguyên vẹn của script build và bố cục thư mục `logs/`
- `scripts/build_windows.ps1`: Giữ nguyên 100% (0 lines diff).
- `logs/` directory layout:
  ```text
  logs/
  ├── active_threads_history/
  ├── quota_log/
  ├── switch_timeline/
  │   └── 2026-10-17_switch_timeline.log
  └── ui_preferences.json
  ```
  Tuyệt đối không còn tệp mồ côi ngoài gốc.

---

## 4. Kết Luận & Bàn Giao

Phase 2 đã hoàn thành trọn vẹn và đạt 100% các tiêu chí nghiệm thu của đặc tả `SPEC-ARCH-20261021-DAILY-SWITCH-TIMELINE-SPEC`. Mã nguồn sẵn sàng chuyển giao cho Supervisor nghiệm thu.
