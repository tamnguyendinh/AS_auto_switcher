# Coder Implementation Report: Quota Stacked Layout (2 Tầng), Thẻ Card 300x130px & Cửa Sổ 1008x630px

- **Date:** 2026-10-19
- **Role:** Coder
- **Authority / SSOT:**
  - `reports/architect/2026-10-19-quota-stacked-layout-card-300x130-spec.md` (Commit `5c98eaa`, Approved by Human Owner)
  - `SPEC-ARCH-20261019-QUOTA-STACKED-CARD-300X130`
  - `AGENTS.md` & `working-rules`
- **Scope (Touch-Map Khóa Cứng):**
  - `internal/ui/window.go`
  - `internal/ui/quota_badge.go`
  - `internal/ui/widgets.go`
  - `internal/ui/ui_test.go`
  - `reports/coder/2026-10-19-quota-stacked-layout-card-300x130.md`

---

## 1. Mục Tiêu & Nguyên Tắc Kỷ Luật Sắt (Iron Rules Compliance)

1. **Bảo tồn 100% Text UI hiện hành (Typography Scale Invariant):**
   - Giữ nguyên cỡ chữ 10pt (`BodyTextSize`) cho toàn bộ text của UI: Tiêu đề App, Status PID/Port, Email, Nút bấm, Tiêu đề Quota/5h/Week, Tên Model Gemini/Claude, số phần trăm `%`.
   - **DUY NHẤT** nhãn thời gian đếm lùi mới có cỡ chữ **`8.5pt`** với màu **`palette.TextMuted`**.
2. **Duy nhất 1 cơ chế bóc tách thời gian (Single Telemetry Mechanism):**
   - Lấy trực tiếp từ các trường của bucket: `bucket.FiveHourReset` và `bucket.WeeklyReset` do `ParseQuotaSummaryWithBuckets` cung cấp. Tuyệt đối không dùng cơ chế fallback 2 tầng.
3. **Khóa cứng hình học (Geometric Invariants):**
   - Kích thước thẻ Card trong `GridWrap`: cố định chính xác **`300 x 130 px`**.
   - Kích thước cửa sổ mặc định: khôi phục về **`1008 x 630 px`** (`DefaultWindowWidth = 1008.0`, `DefaultWindowHeight = 630.0`).
   - Độ dài thanh ray tiến trình MiniBar: cố định **`40 px`**, chiều cao **`6.0 px`**, `CornerRadius = 3.0 px`.
   - Bảng Quota Matrix: kích thước cố định **`230 x 77 px`** (Cột Model: `48 px`, Vách ngăn đứng: `1 px`, Khoảng cách cột: `4 px`, Cột 5h: `78 px`, Cột Week: `86 px`).
4. **Bảo lưu 100% cơ chế in-place reactive update (Zero Jank / Zero Canvas Churn):**
   - Quota update cập nhật trực tiếp tại chỗ trên các con trỏ `c.pctText`, `c.timeText`, `c.fillBar.Resize()` và gọi `Refresh()` cục bộ. Tuyệt đối không hủy bỏ hoặc khởi tạo lại đối tượng Canvas/Widget trong chu kỳ cập nhật Quota.
5. **Full build bằng script repo nguyên bản:**
   - Chạy `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all` đạt 100% test PASS, exit code 0.
   - Script build `scripts/build_windows.ps1` nguyên vẹn 100%, không bị chỉnh sửa.

---

## 2. Invariant Family Map & Chi Tiết Thi Công

| Invariant / Surface | Target File | Chi Tiết Thi Công Cụ Thể |
|---|---|---|
| **Window & Grid Dimensions** | `internal/ui/window.go` | - Khôi phục `DefaultWindowWidth = 1008.0` (giữ nguyên `DefaultWindowHeight = 630.0`).<br>- Cập nhật `a.accountListContainer = container.NewGridWrap(fyne.NewSize(300, 130))`. |
| **2-Tier Stacked Cell Layout** | `internal/ui/quota_badge.go` | - Triển khai struct `matrixStackedCellLayout`:<br>  + Tầng 1 (trên): `timeText` (8.5pt, `palette.TextMuted`) cao `13px`, căn lề trái thẳng hàng với ray bar tại `x = 0`.<br>  + Tầng 2 (dưới): `[barContainer 40px] [gap 4px] [pctText 10pt Bold]` cao `15px`, bắt đầu tại `y = 13px`.<br>  + Tổng chiều cao cell = `28.0 px`. `MinSize` trả về `fyne.NewSize(maxW, 28.0)`. |
| **Model Name Visual Baseline** | `internal/ui/quota_badge.go` | - Triển khai struct `matrixStackedModelCellLayout`:<br>  + Tầng trên (0..13px) để trống tạo spacer.<br>  + Tầng dưới (13..28px) đặt Text Model (`Gemini`, `Claude` 10pt Bold) căn giữa theo chiều cao 15px tại `y = 13px`.<br>  + Chữ Model đứng sóng đôi cân xứng hoàn hảo cùng hàng ngang với thanh Bar và số `%`. |
| **Column & Matrix Table Layout** | `internal/ui/quota_badge.go` | - Struct `matrixColumnLayout`: Row 0 Header `15px`, gap `2px`, Row 1 Gemini `28px`, gap `4px`, Row 2 Claude `28px`. Tổng cao `77px`.<br>- Struct `matrixTableLayout`:<br>  + Cột Model: CỐ ĐỊNH `48px`.<br>  + Vách ngăn đứng 1 & 2: CỐ ĐỊNH `1px` (`palette.BorderSubtle`).<br>  + Khoảng cách cột: CỐ ĐỊNH `4px`.<br>  + Cột 5h: CỐ ĐỊNH `78px`.<br>  + Cột Week: CỐ ĐỊNH `86px`.<br>  + Vách ngăn ngang mờ (1px, `palette.BorderSubtle`) tại `y = 46.5px` giữa hàng Gemini và Claude.<br>  + Tổng kích thước: `230.0 x 77.0 px`. |
| **In-Place Reactive Update** | `internal/ui/quota_badge.go` | - Giữ nguyên phương thức `(c *matrixQuotaCell) update(fraction float64, resetTime time.Time, hasData bool)`:<br>  + Cập nhật `c.pctText.Text`, `c.pctText.Color`.<br>  + Cập nhật `c.fillBar.Resize()` và `c.fillBar.FillColor`.<br>  + Cập nhật `c.timeText.Text = formatCountdown(...)`.<br>  + Gọi `Refresh()` cục bộ trên từng con trỏ; không gọi `c.container.Refresh()` hay resize cửa sổ. |
| **Card Stack Padded Layout** | `internal/ui/widgets.go` | - Thẻ Card trong `NewAccountCardWidgetWithQuota` bao bọc trong `container.NewStack(cardBg, container.NewPadded(cardVBox))` khớp hoàn hảo với khung `300 x 130 px` của `GridWrap`. |
| **Unit Test Conformance** | `internal/ui/ui_test.go` | - Cập nhật các test assertion:<br>  + Kích thước cửa sổ mặc định: `1008 x 630 px`.<br>  + Kích thước Card trong GridWrap: `300 x 130 px`.<br>  + Kích thước Table: `230 x 77 px`, các cột `48px`, `78px`, `86px`.<br>  + Bổ sung ca kiểm thử `TestArchitectSpec_QuotaStackedLayoutCard300x130_FullConformance` nghiệm thu toàn diện AC-1 đến AC-6. |

---

## 3. Ranh Giới Touch-Map & Non-Goals

- **Touch-Map Tuân Thủ 100%:** Chỉ chạm đúng 4 file mã nguồn và 1 file báo cáo được phê duyệt trong hợp đồng:
  - `internal/ui/window.go`
  - `internal/ui/quota_badge.go`
  - `internal/ui/widgets.go`
  - `internal/ui/ui_test.go`
  - `reports/coder/2026-10-19-quota-stacked-layout-card-300x130.md`
- **NO ANVIEN:** Hoàn toàn không sử dụng Anvien.
- **Script Build Không Bị Sửa:** File `scripts/build_windows.ps1` nguyên vẹn 100%.
- **Typography Không Bị Lệch:** Toàn bộ text 10pt cũ giữ nguyên 100%, chỉ nhãn countdown mới là 8.5pt.

---

## 4. Bằng Chứng Nghiệm Thu (Verification Evidence)

### 4.1. Unit Test Package `internal/ui` (`go test -v ./internal/ui`)
```text
=== RUN   TestArchitectSpec_QuotaStackedLayoutCard300x130_FullConformance
--- PASS: TestArchitectSpec_QuotaStackedLayoutCard300x130_FullConformance (0.35s)
=== RUN   TestIndustryStandard_WindowInit1008x630
--- PASS: TestIndustryStandard_WindowInit1008x630 (0.30s)
=== RUN   TestPurgeAllResizing_AccountGridWrapFixedSize
--- PASS: TestPurgeAllResizing_AccountGridWrapFixedSize (0.14s)
=== RUN   TestIndustryStandard_FixedMatrixMetrics
--- PASS: TestIndustryStandard_FixedMatrixMetrics (0.00s)
=== RUN   TestIndustryStandard_OnlyMinibarDecreasesOnQuotaUpdate
--- PASS: TestIndustryStandard_OnlyMinibarDecreasesOnQuotaUpdate (0.00s)
=== RUN   TestArchitectSpec_INV01_InPlaceReactiveQuotaUpdatesZeroCanvasChurn
--- PASS: TestArchitectSpec_INV01_InPlaceReactiveQuotaUpdatesZeroCanvasChurn (0.50s)
=== RUN   TestOwnerRedesign_AccountCardThreeRows
--- PASS: TestOwnerRedesign_AccountCardThreeRows (0.00s)
=== RUN   TestOwnerRedesign_ThreeColumnGrid
--- PASS: TestOwnerRedesign_ThreeColumnGrid (0.23s)
=== RUN   TestOwnerRedesign_F5_CompactCardHeight
--- PASS: TestOwnerRedesign_F5_CompactCardHeight (0.21s)
PASS
ok  	antigravity-switch/internal/ui	13.086s
```

### 4.2. Toàn Bộ Test Suite Repo (`go test ./...`)
```text
ok  	antigravity-switch/cmd/switcher	(cached)
ok  	antigravity-switch/internal/connector	2.893s
ok  	antigravity-switch/internal/manager	8.590s
ok  	antigravity-switch/internal/model	(cached)
ok  	antigravity-switch/internal/reloader	(cached)
ok  	antigravity-switch/internal/scanner	(cached)
ok  	antigravity-switch/internal/store	17.253s
ok  	antigravity-switch/internal/telemetry	12.312s
ok  	antigravity-switch/internal/ui	13.887s
ok  	antigravity-switch/internal/wincred	(cached)
```

### 4.3. Kết Quả Chạy Full Build Bắt Buộc (`powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all`)
```text
================================================================================
⚡ Antigravity Multi-Account Switcher - Windows Build System
   Mode: all | OutDir: bin | SkipTests: False | Clean: False
================================================================================
[2/4] Running pre-flight tests & vet...
  -> Verifying pure Go packages (CGO_ENABLED=0)...
ok  	antigravity-switch/internal/model	(cached)
ok  	antigravity-switch/internal/wincred	(cached)
ok  	antigravity-switch/internal/store	13.033s
ok  	antigravity-switch/internal/scanner	(cached)
ok  	antigravity-switch/internal/connector	1.978s
ok  	antigravity-switch/internal/reloader	(cached)
ok  	antigravity-switch/internal/manager	7.578s
ok  	antigravity-switch/cmd/switcher	(cached)
  [PASS] Pure Go verification succeeded.
  -> Verifying GUI packages with CGo & Fyne (CGO_ENABLED=1)...
ok  	antigravity-switch/cmd/switcher	1.645s
ok  	antigravity-switch/internal/ui	14.703s
  [PASS] GUI verification succeeded.
[3/4] Compiling target binaries...
  -> Checking Windows application icon and resource...
  -> Compiling resource with windres: cmd\switcher\app.rc -> cmd\switcher\rsrc_windows_amd64.syso
  [SUCCESS] Compiled PE resource object: cmd\switcher\rsrc_windows_amd64.syso (12552 bytes)
  -> Building Standalone CLI Binary (Pure Go, CGO_ENABLED=0)...
  [SUCCESS] Created: bin\antigravity-switch.exe
  Compile Duration: 3772 ms | Size: 10.51 MB (11023872 bytes)
  -> Building Standalone GUI Binary (Fyne, CGO_ENABLED=1)...
  [SUCCESS] Created: bin\antigravity-switch-gui.exe
  Compile Duration: 6115 ms | Size: 27.58 MB (28918272 bytes)
[4/4] Executing quick verification...
  [PASS] Embedded PE application icon verified on bin\antigravity-switch.exe (32x32)
  [PASS] Embedded PE application icon verified on bin\antigravity-switch-gui.exe (32x32)
  -> Running 'bin\antigravity-switch.exe --status'...
================================================================================
⚡ ANTIGRAVITY SESSION STATUS
================================================================================
Active Email:    quantritaikhoanemail@gmail.com
Language Server: Running (PID: 14880 | Port: 50222 | CSRF: Present)
Win32 Credential: Present (260 bytes access token)
================================================================================
================================================================================
⚡ BUILD SUCCESSFUL
================================================================================
```

---

## 5. Kết Luận & Handoff

Hạng mục thi công chuẩn hóa bố cục Quota 2 tầng (Stacked Layout), kích thước thẻ tài khoản 300x130 px và cửa sổ 1008x630 px theo đúng đặc tả kiến trúc `reports/architect/2026-10-19-quota-stacked-layout-card-300x130-spec.md` đã hoàn thành 100%. Toàn bộ tiêu chí nghiệm thu AC-1 đến AC-6 đều đạt chuẩn. Code đã sẵn sàng bàn giao cho Supervisor Lane nghiệm thu độc lập Zero-Trust.