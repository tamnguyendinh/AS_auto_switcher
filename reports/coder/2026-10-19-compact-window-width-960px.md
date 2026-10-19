# Coder Implementation Report: Thu Gọn Bề Rộng Cửa Sổ Mặc Định Xuống 960px (Window 960x630px)

- **Date:** 2026-10-19
- **Role:** Coder
- **Authority / SSOT:**
  - `reports/architect/2026-10-19-compact-window-width-960px-spec.md` (Commit `b2c3e52`, Phê duyệt chính thức bởi Human Owner)
  - Mã văn kiện đặc tả: `SPEC-ARCH-20261019-COMPACT-WINDOW-WIDTH-960PX`
  - `AGENTS.md` & `working-rules`
- **Scope (Touch-Map Khóa Cứng):**
  - `internal/ui/window.go`
  - `internal/ui/ui_test.go`
  - `reports/coder/2026-10-19-compact-window-width-960px.md`

---

## 1. Mục Tiêu & Tuân Thủ Kỷ Luật Sắt (Iron Rules Compliance)

1. **Khóa cứng kích thước cửa sổ mặc định mới: `960 x 630 px`:**
   - Trong `internal/ui/window.go`, đã cập nhật hằng số `DefaultWindowWidth float32 = 960.0` (giảm từ `1008.0`).
   - Giữ nguyên `DefaultWindowHeight float32 = 630.0`, `MinWindowWidth float32 = 480.0`, `MinWindowHeight float32 = 360.0`.
2. **Bảo tồn kích thước thẻ Card & bảng Quota ma trận:**
   - Kích thước thẻ Card trong `GridWrap` giữ nguyên tuyệt đối **`280 x 130 px`** (`container.NewGridWrap(fyne.NewSize(280, 130))`).
   - Bảng Quota 2 tầng bên trong thẻ giữ nguyên kích thước toán học **`228 x 77 px`**.
3. **Bảo tồn nguyên vẹn các tiêu chuẩn đã phê duyệt:**
   - Bố cục danh sách luôn hiển thị **3 cột thẻ** thẳng tắp, không phát sinh thanh cuộn ngang.
   - Khoảng trống hình học: Bề rộng 960px trừ ~40px viền/thanh cuộn còn 920px khả dụng; 3 thẻ x 280px = 840px (khoảng cách tự nhiên ~20px giữa các cột), nằm trong khoảng $840 \le 920 < 1120$, đảm bảo Fyne GridWrap khóa cố định 3 cột thẻ.
   - Typography: Text chính 10pt Bold (`BodyTextSize`), text thời gian đếm lùi 8.5pt `TextMuted`.
   - In-place reactive update (Zero Jank / Zero Canvas Churn): Quota update tiếp tục cập nhật trực tiếp tại chỗ trên các con trỏ `c.pctText`, `c.timeText`, `c.fillBar.Resize()` và gọi `Refresh()` cục bộ.
4. **Cập nhật Conformance Tests:**
   - Toàn bộ các test assertions kiểm tra kích thước cửa sổ trong `internal/ui/ui_test.go` đã được cập nhật sang chuẩn `960x630`.
   - Đổi tên `TestIndustryStandard_WindowInit1008x630` -> `TestIndustryStandard_WindowInit960x630`.
   - Cập nhật các assertions trong `TestOwnerRedesign_WindowSizeAndLayout`, `TestOwnerRedesign_F5_CompactCardHeight`, `TestArchitectSpec_QuotaStackedLayoutCard280x130_FullConformance`, `TestArchitectSpec_RemoveVerticalSeparatorsQuotaMatrix_FullConformance`.
   - Bổ sung ca kiểm thử conformance độc lập: `TestArchitectSpec_CompactWindowWidth960px_FullConformance` kiểm tra toàn diện AC-1 đến AC-4 theo đúng văn kiện đặc tả kiến trúc.
5. **Full build bằng script repo nguyên bản:**
   - Chạy `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all` đạt 100% test PASS, biên dịch thành công cả CLI lẫn GUI, exit code 0.
   - Script build `scripts/build_windows.ps1` nguyên vẹn 100%, không bị chỉnh sửa.

---

## 2. Invariant Family Map & Chi Tiết Thi Công

| Invariant / Surface | Target File | Chi Tiết Thi Công Cụ Thể |
|---|---|---|
| **Window Default Width** | `internal/ui/window.go` | - Dòng 49: Cập nhật hằng số kích thước mặc định:<br>  `DefaultWindowWidth float32 = 960.0` (từ `1008.0`).<br>  Giữ nguyên `DefaultWindowHeight float32 = 630.0`. |
| **Unit Test Suite Conformance** | `internal/ui/ui_test.go` | - Cập nhật comment Slice P6A-3 sang `960x630`.<br>  - `TestOwnerRedesign_WindowSizeAndLayout`: Assert `sz.Width == 960 && sz.Height == 630`.<br>  - `TestOwnerRedesign_F5_CompactCardHeight`: Cập nhật `w.Resize(fyne.NewSize(960, 630))`.<br>  - Đổi tên `TestIndustryStandard_WindowInit1008x630` thành `TestIndustryStandard_WindowInit960x630` và assert `sz.Width == 960 && sz.Height == 630`.<br>  - `TestArchitectSpec_QuotaStackedLayoutCard280x130_FullConformance`: Assert `sz.Width == 960 && sz.Height == 630`.<br>  - `TestArchitectSpec_RemoveVerticalSeparatorsQuotaMatrix_FullConformance`: Assert `DefaultWindowWidth == 960.0 && DefaultWindowHeight == 630.0`.<br>  - Bổ sung `TestArchitectSpec_CompactWindowWidth960px_FullConformance` xác minh toàn diện DefaultWindowWidth=960.0, DefaultWindowHeight=630.0, Window Canvas 960x630 và lưới 3 cột thẻ 280x130px. |

---

## 3. Ranh Giới Touch-Map & Non-Goals

- **Touch-Map Tuân Thủ 100%:** Chỉ chạm đúng 2 file mã nguồn và 1 file báo cáo được phê duyệt trong hợp đồng:
  - `internal/ui/window.go`
  - `internal/ui/ui_test.go`
  - `reports/coder/2026-10-19-compact-window-width-960px.md`
- **NO ANVIEN:** Hoàn toàn không sử dụng Anvien.
- **Không chỉnh sửa script build:** `scripts/build_windows.ps1` nguyên vẹn 100%.
- **Không thay đổi kích thước thẻ Card hay bảng Quota:** Thẻ Card giữ nguyên `280 x 130 px`, bảng Quota giữ nguyên `228 x 77 px`.

---

## 4. Bảng Đối Chiếu Tiêu Chí Nghiệm Thu (Acceptance Criteria)

| Tiêu Chí Nghiệm Thu | Kết Quả Thực Tế | Trạng Thái |
|---|---|---|
| **AC-1: Default Window Width 960px** | `DefaultWindowWidth = 960.0`, `DefaultWindowHeight = 630.0`. Cửa sổ khởi tạo đạt `960 x 630 px`. | **PASS** |
| **AC-2: 3-Column Card Layout** | GridWrap kích thước thẻ `280 x 130 px` hiển thị đúng 3 cột thẻ trên bề rộng 960px (content width 920px), không thanh cuộn ngang. | **PASS** |
| **AC-3: In-place Reactive Update** | Quota update in-place mượt mà, không tái tạo Canvas, không làm thay đổi kích thước cửa sổ. | **PASS** |
| **AC-4: Test Suite & Build** | Toàn bộ test suite `internal/ui` và các package Go đạt 100% PASS; script `build_windows.ps1 -Mode all` exit code 0. | **PASS** |

---

## 5. Bằng Chứng Thực Thi Full Build (Mandatory Verification Evidence)

### 5.1. Lệnh Thực Thi:
```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all
```

### 5.2. Nhật Ký Kết Quả Biên Dịch & Kiểm Thử:
```text
================================================================================
⚡ Antigravity Multi-Account Switcher - Windows Build System
   Mode: all | OutDir: bin | SkipTests: False | Clean: False
================================================================================
[2/4] Running pre-flight tests & vet...
  -> Verifying pure Go packages (CGO_ENABLED=0)...
ok  	antigravity-switch/internal/model	(cached)
ok  	antigravity-switch/internal/wincred	(cached)
ok  	antigravity-switch/internal/store	5.075s
ok  	antigravity-switch/internal/scanner	(cached)
ok  	antigravity-switch/internal/connector	(cached)
ok  	antigravity-switch/internal/reloader	(cached)
ok  	antigravity-switch/internal/manager	6.628s
ok  	antigravity-switch/cmd/switcher	(cached)
  [PASS] Pure Go verification succeeded.
  -> Verifying GUI packages with CGo & Fyne (CGO_ENABLED=1)...
ok  	antigravity-switch/cmd/switcher	1.532s
ok  	antigravity-switch/internal/ui	11.754s
  [PASS] GUI verification succeeded.
[3/4] Compiling target binaries...
  -> Checking Windows application icon and resource...
  -> Compiling resource with windres: cmd\switcher\app.rc -> cmd\switcher\rsrc_windows_amd64.syso
  [SUCCESS] Compiled PE resource object: cmd\switcher\rsrc_windows_amd64.syso (12552 bytes)
  -> Building Standalone CLI Binary (Pure Go, CGO_ENABLED=0)...
  [SUCCESS] Created: bin\antigravity-switch.exe
  Compile Duration: 2851 ms | Size: 10.51 MB (11023872 bytes)
  -> Building Standalone GUI Binary (Fyne, CGO_ENABLED=1)...
  [SUCCESS] Created: bin\antigravity-switch-gui.exe
  Compile Duration: 5774 ms | Size: 27.58 MB (28917248 bytes)
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

## 6. Kết Luận

Nhiệm vụ thi công thu gọn bề rộng cửa sổ mặc định từ 1008px xuống 960px (`960 x 630 px`) đã hoàn thành chính xác 100% theo đúng văn kiện đặc tả kỹ thuật `reports/architect/2026-10-19-compact-window-width-960px-spec.md` (Commit `b2c3e52`).

Sẵn sàng bàn giao cho Supervisor Lane nghiệm thu độc lập.
