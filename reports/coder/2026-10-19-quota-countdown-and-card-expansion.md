# Coder Implementation Report: Quota Countdown Timer & Account Card Expansion to 350px

- **Date:** 2026-10-19
- **Role:** Coder
- **Authority / SSOT:**
  - `reports/architect/2026-10-19-quota-countdown-and-card-expansion-spec.md` (Commit `07c988f`, Approved by Human Owner)
  - `SPEC-ARCH-20261019-QUOTA-COUNTDOWN-CARD-350PX`
  - `AGENTS.md` & `working-rules`
- **Scope (Touch-Map Khóa Cứng):**
  - `internal/ui/window.go`
  - `internal/ui/quota_badge.go`
  - `internal/ui/widgets.go`
  - `internal/ui/ui_test.go`
  - `reports/coder/2026-10-19-quota-countdown-and-card-expansion.md`

---

## 1. Mục Tiêu & Nguyên Tắc Kỷ Luật Sắt (Iron Rules)

1. **Bảo toàn 100% Text UI hiện hành:**
   - Giữ nguyên cỡ chữ 10pt (`BodyTextSize`) cho tất cả các phần tử UI hiện hữu: Tiêu đề App, Status PID/Port, Email, Nút bấm, Quota/5h/Week, Tên Model Gemini/Claude, số phần trăm `%`.
   - **DUY NHẤT** nhãn thời gian đếm ngược mới được thêm vào có kích thước **`8.5pt`** với màu **`palette.TextMuted`**.
2. **Duy nhất 1 cơ chế bóc tách thời gian:**
   - Lấy trực tiếp từ các trường của bucket: `quota.Claude.FiveHourReset`, `quota.Claude.WeeklyReset`, `quota.Gemini.FiveHourReset`, `quota.Gemini.WeeklyReset` do `ParseQuotaSummaryWithBuckets` cung cấp.
   - Tuyệt đối không dùng cơ chế fallback 2 tầng hay quét raw buckets.
3. **Mở rộng kích thước thẻ và cửa sổ (Khóa cứng Zero-Jank):**
   - Thẻ Card trong `GridWrap` cố định **`350 x 100 px`** (mở rộng từ 318px).
   - Kích thước cửa sổ mặc định cố định **`1090 x 630 px`** (chứa vừa khít 3 cột thẻ thẳng hàng, không clipping).
   - Chiều dài mini progress bar giữ nguyên cố định: **`40 px`**.
4. **Cập nhật Reactive tại chỗ (In-Place):**
   - Tích hợp đếm ngược vào hàm `(w *AccountCardWidget) UpdateQuota` và `(c *matrixQuotaCell) update`, cập nhật trực tiếp `c.timeText` mà không hủy/tạo lại cây Canvas, bảo toàn nguyên lý Zero Canvas Churn.

---

## 2. Invariant Family Map & Chi Tiết Thi Công

| Invariant / Surface | Target File | Chi Tiết Thi Công Cụ Thể |
|---|---|---|
| **Window & Grid Metrics** | `internal/ui/window.go` | - Cập nhật `DefaultWindowWidth = 1090.0` (giữ nguyên `DefaultWindowHeight = 630.0`).<br>- Cập nhật `a.accountListContainer = container.NewGridWrap(fyne.NewSize(350, 100))`. |
| **DualCycle Struct & Extraction** | `internal/ui/quota_badge.go` | - Mở rộng `DualCycleQuota` thêm 4 trường: `Claude5hReset`, `ClaudeWeekReset`, `Gemini5hReset`, `GeminiWeekReset` (`time.Time`).<br>- Trong `ExtractDualCycleQuota`: gán trực tiếp 4 trường reset time từ `quota.Claude` và `quota.Gemini`. |
| **Countdown Formatting** | `internal/ui/quota_badge.go` | - Viết hàm `formatCountdown(resetTime time.Time, isWeekly bool) string`:<br>  + `resetTime.IsZero()` -> `"--"`<br>  + `time.Until(resetTime) <= 0` -> `"(Ready)"`<br>  + 5h (`!isWeekly`): `fmt.Sprintf("(%02dh %02dm)", h, m)` (ví dụ: `"(04h 32m)"`)<br>  + Week (`isWeekly`): `fmt.Sprintf("(%dd %02dh %02dm)", d, h, m)` (ví dụ: `"(6d 21h 59m)"`) |
| **Cell Layout & 8.5pt Label** | `internal/ui/quota_badge.go` | - Cập nhật `matrixCellLayout`: xếp ngang 3 phần tử `[miniBar 40px] [pct 10pt] [time 8.5pt]`.<br>- Mở rộng `matrixQuotaCell` thêm `timeText *canvas.Text` (cỡ 8.5, màu `palette.TextMuted`) và `isWeekly bool`.<br>- Cập nhật `(c *matrixQuotaCell) update(fraction float64, resetTime time.Time, hasData bool)`: cập nhật `c.timeText.Text = formatCountdown(...)` và gọi `c.timeText.Refresh()`. |
| **Matrix Column Dimensions** | `internal/ui/quota_badge.go` | - Khóa cứng tỷ lệ 3 cột trong `matrixTableLayout`:<br>  + Cột Model: `48 px`<br>  + Cột 5h: `128 px`<br>  + Cột Week: `146 px`<br>  + Vách ngăn và khoảng cách: `18 px` (2 vách ngăn 1px + 4 gaps 4px)<br>  + Tổng chiều rộng bảng Quota: `340 px` (vừa khít trong thẻ 350px). |
| **In-Place Reactive Wiring** | `internal/ui/widgets.go` | - Mở rộng `AccountCardWidget` lưu trữ trực tiếp con trỏ `gemini5hTime`, `geminiWeekTime`, `claude5hTime`, `claudeWeekTime`.<br>- Trong `UpdateQuota`: truyền `dq.Gemini5hReset`, `dq.GeminiWeekReset`, `dq.Claude5hReset`, `dq.ClaudeWeekReset` vào phương thức `update()` của từng cell tương ứng. |
| **Unit Test Coverage** | `internal/ui/ui_test.go` | - Cập nhật `TestQuotaCountdownCalculationAndLabel` kiểm thử toàn diện các ca: `ZeroTime` (`"--"`), quá hạn (`"(Ready)"`), 5h format, Week format, kiểm tra cỡ chữ 8.5pt và màu `TextMuted`.<br>- Bổ sung kiểm thử `TestExtractDualCycleQuota_Contract` cho 4 trường ResetTime.<br>- Đồng bộ các ca kiểm tra kích thước cửa sổ 1090x630 và kích thước thẻ 350x100 trong `TestIndustryStandard_WindowInit1090x630`, `TestPurgeAllResizing_AccountGridWrapFixedSize`, `TestIndustryStandard_FixedMatrixMetrics`. |

---

## 3. Ranh Giới Touch-Map & Non-Goals

- **Touch-Map Tuân Thủ 100%:** Chỉ chạm đúng 4 file mã nguồn và 1 file báo cáo được phê duyệt.
- **NO ANVIEN:** Hoàn toàn không sử dụng Anvien.
- **Script Build Không Bị Sửa:** File `scripts/build_windows.ps1` nguyên vẹn 100%.
- **Typography Không Bị Lệch:** Toàn bộ text 10pt cũ giữ nguyên 100%, chỉ nhãn countdown mới là 8.5pt.

---

## 4. Bằng Chứng Nghiệm Thu (Verification Evidence)

### 4.1. Unit Test Package `internal/ui` (`go test -v -count=1 ./internal/ui`)
Tất cả 36 tests PASS 100% trong 8.409s:
- `TestQuotaCountdownCalculationAndLabel`: PASS
- `TestExtractDualCycleQuota_Contract`: PASS
- `TestIndustryStandard_FixedMatrixMetrics`: PASS
- `TestIndustryStandard_OnlyMinibarDecreasesOnQuotaUpdate`: PASS
- `TestIndustryStandard_WindowInit1090x630`: PASS
- `TestPurgeAllResizing_AccountGridWrapFixedSize`: PASS
- `TestArchitectSpec_INV01_InPlaceReactiveQuotaUpdatesZeroCanvasChurn`: PASS

### 4.2. Full Build Repository Script (`powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all`)
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
ok  	antigravity-switch/internal/manager	6.013s
ok  	antigravity-switch/cmd/switcher	(cached)
  [PASS] Pure Go verification succeeded.
  -> Verifying GUI packages with CGo & Fyne (CGO_ENABLED=1)...
ok  	antigravity-switch/cmd/switcher	1.348s
ok  	antigravity-switch/internal/ui	8.670s
  [PASS] GUI verification succeeded.
[3/4] Compiling target binaries...
  -> Checking Windows application icon and resource...
  -> Compiling resource with windres: cmd\switcher\app.rc -> cmd\switcher\rsrc_windows_amd64.syso
  [SUCCESS] Compiled PE resource object: cmd\switcher\rsrc_windows_amd64.syso (12552 bytes)
  -> Building Standalone CLI Binary (Pure Go, CGO_ENABLED=0)...
  [SUCCESS] Created: bin\antigravity-switch.exe
  Compile Duration: 3180 ms | Size: 10.51 MB (11023872 bytes)
  -> Building Standalone GUI Binary (Fyne, CGO_ENABLED=1)...
  [SUCCESS] Created: bin\antigravity-switch-gui.exe
  Compile Duration: 5806 ms | Size: 27.58 MB (28915200 bytes)
[4/4] Executing quick verification...
  [PASS] Embedded PE application icon verified on bin\antigravity-switch.exe (32x32)
  [PASS] Embedded PE application icon verified on bin\antigravity-switch-gui.exe (32x32)
  -> Running 'bin\antigravity-switch.exe --status'...
================================================================================
⚡ ANTIGRAVITY SESSION STATUS
================================================================================
Active Email:    quantritaikhoanemail@gmail.com
Language Server: Running (PID: 8712 | Port: 59114 | CSRF: Present)
Win32 Credential: Present (260 bytes access token)
================================================================================
================================================================================
⚡ BUILD SUCCESSFUL (Exit Code 0)
================================================================================
```

---

## 5. Kết Luận & Handoff

Hạng mục thi công bổ sung Quota Countdown Timer và mở rộng thẻ tài khoản lên 350x100 px đã hoàn thành 100%, thỏa mãn toàn bộ tiêu chí nghiệm thu AC-1 đến AC-5 trong văn kiện kiến trúc của Human Owner.
Sẵn sàng bàn giao cho Supervisor Lane tiến hành thẩm định độc lập Zero-Trust.
