# Báo Cáo Thi Công: Lưu Trữ Bền Vững Trạng Thái Nút Ghim Cửa Sổ (Always-on-Top / Pin Window)
(Window Always-on-Top Pin Persistence Implementation Report)

- **Ngày thực hiện:** 2026-10-21
- **Vai trò:** Coder
- **Thẩm quyền kiến trúc (Authority / SSOT):** `reports/architect/2026-10-21-always-on-top-persistence-spec.md` (`SPEC-ARCH-20261021-ALWAYS-ON-TOP-PERSISTENCE`, commit `b909f06`)
- **Trạng thái:** HOÀN THÀNH (READY_FOR_SUPERVISOR)

---

## I. TỔNG QUAN THI CÔNG & INVARIANT FAMILY MAP

### 1. Invariant Family Map
- **Family Name:** `Window Always-on-Top Persistence & 4-Way Cross-Preservation`
- **Authority / SSOT:** `reports/architect/2026-10-21-always-on-top-persistence-spec.md`
- **Sibling Surfaces Inspected & Closed:**
  1. `UIPreferences` schema tại `internal/ui/themes.go`: Bổ sung trường con trỏ boolean `AlwaysOnTop *bool \`json:"always_on_top,omitempty"\``.
  2. Hàm `LoadUIPreferences` tại `internal/ui/themes.go`: Đảm bảo giá trị chuẩn mực mặc định (*Canonical Default*) là `true` khi tệp chưa tồn tại, file rỗng, parse lỗi hoặc trường `always_on_top` bị khuyết trong các tệp JSON cũ; không bao giờ bị gán zero-value `false`.
  3. Hàm `SaveUIPreferences` tại `internal/ui/themes.go`: Khi struct truyền vào là `nil`, khởi tạo mặc định với `AlwaysOnTop: &defaultTop (true)`.
  4. Nạp trạng thái lúc khởi động tại `internal/ui/app.go` (`NewAppWithFyne`): Đọc `prefs.AlwaysOnTop` gán vào `a.alwaysOnTop`.
  5. Gọi API Win32 thực tế tại `internal/ui/app.go` (`Init` và `ShowAndRun`): Cập nhật gọi `setWindowAlwaysOnTop(windowTitle, a.alwaysOnTop)` để phản ánh chính xác trạng thái cửa sổ thực tế kể cả khi `alwaysOnTop == false`.
  6. Khởi tạo mức độ ưu tiên nút bấm tại `internal/ui/window.go`: Khởi tạo đúng `widget.HighImportance` khi `alwaysOnTop == true` và `widget.MediumImportance` khi `alwaysOnTop == false`.
  7. Ghi đĩa non-blocking tại `internal/ui/app.go` (`ToggleAlwaysOnTop` và `SetAlwaysOnTop`): Khởi chạy goroutine ghi đĩa bất đồng bộ mà không gây giật lag giao diện người dùng (< 16ms).
  8. Bảo toàn chéo 4 chiều (Mutual 4-Way Cross Preservation) & Loại bỏ Race Condition tại `internal/ui/app.go`:
     - Bổ sung `prefSaveMu sync.Mutex` trên `App` để tuần tự hóa các thao tác đọc-sửa-ghi file `ui_preferences.json`.
     - Trong `SetTheme`, `SetMaskAccounts`, `SetAutoSwitchEnabled`, `ToggleAlwaysOnTop`, và `SetAlwaysOnTop`, đồng bộ đầy đủ cả 4 trường (`Theme`, `MaskAccounts`, `AutoSwitch`, `AlwaysOnTop`) từ bộ nhớ RAM xuống đĩa, triệt tiêu hoàn toàn rủi ro ghi đè hoặc làm mất dữ liệu chéo.
- **Legacy Fallback Status:** Tương thích ngược 100% với các tệp cấu hình cũ chưa có `always_on_top`. `logs/ui_preferences.json` là nguồn chân lý duy nhất quản lý cả 4 tùy chọn UI (`theme`, `mask_accounts`, `auto_switch`, `always_on_top`).
- **Stale Tests/Helpers:** Đã bổ sung 4 bài test tự động chính thức theo yêu cầu (kèm 4 bài test alias theo đúng Mục IV của SPEC) trong `internal/ui/themes_test.go`.
- **Residual Unverified Surfaces:** `none`.

---

## II. CHI TIẾT CÁC TỆP SỬA ĐỔI (FILES CHANGED)

1. `internal/ui/themes.go`:
   - Bổ sung trường `AlwaysOnTop *bool \`json:"always_on_top,omitempty"\`` vào struct `UIPreferences`.
   - Cập nhật hàm `LoadUIPreferences`: Tự động gán `defaultTop := true` khi file không tồn tại, file rỗng, deserialization thất bại hoặc khi `pref.AlwaysOnTop == nil`.
   - Cập nhật hàm `SaveUIPreferences`: Gán fallback `AlwaysOnTop: &defaultTop` khi `pref == nil`.

2. `internal/ui/app.go`:
   - Bổ sung trường `prefSaveMu sync.Mutex` vào struct `App` để serialize các tác vụ ghi đĩa preferences, triệt tiêu race condition giữa các goroutine non-blocking.
   - Trong `NewAppWithFyne`: Khôi phục `a.alwaysOnTop` từ `*prefs.AlwaysOnTop` khi có cấu hình lưu trên đĩa.
   - Trong `Init()` và `ShowAndRun()`: Gọi `setWindowAlwaysOnTop(windowTitle, a.alwaysOnTop)` cho cửa sổ Win32 thực tế.
   - Trong `ToggleAlwaysOnTop` và `SetAlwaysOnTop`: Bổ sung goroutine lưu bất đồng bộ `pref.AlwaysOnTop = &enabled` vào đĩa qua `SaveUIPreferences`, bảo toàn cả 4 trường UI.
   - Trong `SetTheme`, `SetMaskAccounts`, và `SetAutoSwitchEnabled`: Bảo toàn trường `AlwaysOnTop: &alwaysOnTop` và khóa `a.prefSaveMu` khi ghi đĩa, bảo đảm không bị mất trạng thái khi thay đổi cài đặt khác.

3. `internal/ui/themes_test.go`:
   - Bổ sung import `"fyne.io/fyne/v2/widget"`.
   - Bổ sung bài test 1: `TestUIPreferences_AlwaysOnTop_BackwardCompatibility` (và alias `TestUIPreferences_JSONSerialization_WithAlwaysOnTop`) kiểm tra Case A (`always_on_top: true`), Case B (`always_on_top: false`), và Case C (tương thích ngược với file JSON cũ thiếu trường `always_on_top`).
   - Bổ sung bài test 2: `TestApp_AlwaysOnTop_RoundTrip` (và alias `TestApp_AlwaysOnTopPersistence_RoundTrip`) kiểm tra chu trình toggle/set Always-on-Top, ghi đĩa non-blocking và đọc lại từ đĩa.
   - Bổ sung bài test 3: `TestApp_AlwaysOnTop_StartupRestoration` (và alias `TestApp_StartupRestoresAlwaysOnTopState`) kiểm tra khôi phục trạng thái `always_on_top: false` từ đĩa khi khởi động app, xác nhận nút bấm chuyển sang `MediumImportance`.
   - Bổ sung bài test 4: `TestApp_UIPreferences_4WayCrossPreservation` (và alias `TestApp_CrossSettingPreservation_4Way`) kiểm tra bảo toàn chéo 4 chiều giữa `Theme`, `MaskAccounts`, `AutoSwitch`, và `AlwaysOnTop`.

---

## III. KẾT QUẢ KIỂM CHỨNG & BẰNG CHỨNG THỰC NGHIỆM (E2E VERIFICATION)

### 1. Kết quả kiểm thử tự động (Test Matrix)
- `go test -v ./internal/ui -run "AlwaysOnTop|4Way"` -> **PASS (100%)**
  + `TestUIPreferences_AlwaysOnTop_BackwardCompatibility` -> **PASS (0.00s)**
  + `TestUIPreferences_JSONSerialization_WithAlwaysOnTop` -> **PASS (0.00s)**
  + `TestApp_AlwaysOnTop_RoundTrip` -> **PASS (0.21s)**
  + `TestApp_AlwaysOnTopPersistence_RoundTrip` -> **PASS (0.05s)**
  + `TestApp_AlwaysOnTop_StartupRestoration` -> **PASS (0.03s)**
  + `TestApp_StartupRestoresAlwaysOnTopState` -> **PASS (0.04s)**
  + `TestApp_UIPreferences_4WayCrossPreservation` -> **PASS (0.19s)**
  + `TestApp_CrossSettingPreservation_4Way` -> **PASS (0.15s)**
  + `TestAlwaysOnTop` -> **PASS (0.20s)**
  + `TestToggleAlwaysOnTop` -> **PASS (0.21s)**

- `go test ./...` trên toàn bộ repository -> **PASS (100%)**
  + `cmd/switcher` -> **PASS**
  + `internal/automation` -> **PASS**
  + `internal/connector` -> **PASS**
  + `internal/manager` -> **PASS**
  + `internal/model` -> **PASS**
  + `internal/reloader` -> **PASS**
  + `internal/scanner` -> **PASS**
  + `internal/store` -> **PASS**
  + `internal/telemetry` -> **PASS**
  + `internal/ui` -> **PASS**
  + `internal/wincred` -> **PASS**

### 2. Kết quả biên dịch hệ thống (Windows Build Verification)
- Lệnh thực thi: `pwsh -File .\scripts\build_windows.ps1 -Clean`
- Exit Code: **0 (BUILD SUCCESSFUL)**
- Artifact sinh ra: `bin\antigravity-switch.exe` (27.75 MB / 29,093,376 bytes)
- Embedded PE application icon: Verified (32x32)
- scripts/build_windows.ps1 diff: **0 lines diff** (giữ nguyên 100%)

---

## IV. ĐỐI SOÁT TIÊU CHÍ NGHIỆM THU (ACCEPTANCE CRITERIA CONFORMANCE)

| Tiêu chí nghiệm thu (SPEC Section V) | Trạng thái | Ghi chú bằng chứng |
| :--- | :---: | :--- |
| **AC-1:** `go test ./internal/ui/...` chạy PASS 100% | **ĐẠT** | Toàn bộ suite nội bộ và toàn repo pass 100% không phát sinh lỗi. |
| **AC-2:** Bấm nút ghim 📌 để unpin, tệp `logs/ui_preferences.json` được cập nhật `"always_on_top": false` | **ĐẠT** | Được chứng minh qua `TestApp_AlwaysOnTop_RoundTrip` (bước 1). |
| **AC-3:** Bấm nút ghim 📌 để pin, tệp `logs/ui_preferences.json` được cập nhật `"always_on_top": true` | **ĐẠT** | Được chứng minh qua `TestApp_AlwaysOnTop_RoundTrip` (bước 2). |
| **AC-4:** Khởi động lại ứng dụng, trạng thái ghim/bỏ ghim được khôi phục chính xác | **ĐẠT** | Được chứng minh qua `TestApp_AlwaysOnTop_StartupRestoration` (khôi phục app state và `alwaysOnTopBtn.Importance = MediumImportance`). |
| **AC-5:** Đổi Theme, gạt Ẩn tài khoản 🔒, hay gạt Tự động không làm mất trạng thái nút Ghim 📌 | **ĐẠT** | Được chứng minh qua `TestApp_UIPreferences_4WayCrossPreservation` (bảo toàn chéo 4 chiều, không race condition). |
| **AC-6:** Tương thích ngược với file cấu hình cũ | **ĐẠT** | Được chứng minh qua `TestUIPreferences_AlwaysOnTop_BackwardCompatibility` (tự gán `true` khi unmarshal file cũ). |
| **AC-7:** Giữ nguyên 100% build script, không dùng anvien | **ĐẠT** | `scripts/build_windows.ps1` 0 lines diff; chỉ thay đổi 3 file mã nguồn theo đúng hợp đồng giao việc. |
