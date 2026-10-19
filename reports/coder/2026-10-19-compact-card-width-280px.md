# Coder Implementation Report: Thu Gọn Độ Rộng Thẻ Tài Khoản Xuống 280px (Card 280x130px)

- **Date:** 2026-10-19
- **Role:** Coder
- **Authority / SSOT:**
  - `reports/architect/2026-10-19-compact-card-width-280px-spec.md` (Commit `05fbc8b`, Phê duyệt chính thức bởi Human Owner)
  - Mã văn kiện đặc tả: `SPEC-ARCH-20261019-COMPACT-CARD-WIDTH-280PX`
  - `AGENTS.md` & `working-rules`
- **Scope (Touch-Map Khóa Cứng):**
  - `internal/ui/window.go`
  - `internal/ui/ui_test.go`
  - `reports/coder/2026-10-19-compact-card-width-280px.md`

---

## 1. Mục Tiêu & Tuân Thủ Kỷ Luật Sắt (Iron Rules Compliance)

1. **Khóa cứng kích thước thẻ Card mới: `280 x 130 px`:**
   - Trong `internal/ui/window.go` (`setupLayout`), đã cập nhật kích thước thẻ của `GridWrap` từ `fyne.NewSize(300, 130)` thành `fyne.NewSize(280, 130)`.
   - Chiều rộng của thẻ Card giảm từ **`300 px`** xuống cố định **`280 px`**. Chiều cao giữ nguyên cố định **`130 px`**.
2. **Bảo tồn kích thước cửa sổ & bảng Quota ma trận:**
   - Cửa sổ ứng dụng giữ nguyên tuyệt đối **`1008 x 630 px`** (`DefaultWindowWidth = 1008.0`, `DefaultWindowHeight = 630.0`).
   - Bảng Quota 2 tầng (không vách dọc, có vách ngang `hSep`) giữ nguyên kích thước toán học **`228 x 77 px`**.
   - Bảng Quota nằm lọt thỏm vừa vặn bên trong thẻ 280px với lề an toàn 36px (18px mỗi bên sau khi trừ padding viền 16px).
3. **Bảo tồn nguyên vẹn các tiêu chuẩn đã phê duyệt:**
   - Typography: Text chính 10pt Bold (`BodyTextSize`), text thời gian đếm lùi 8.5pt `TextMuted`.
   - In-place reactive update (Zero Jank / Zero Canvas Churn): Quota update tiếp tục cập nhật trực tiếp tại chỗ trên các con trỏ `c.pctText`, `c.timeText`, `c.fillBar.Resize()` và gọi `Refresh()` cục bộ.
4. **Cập nhật Conformance Tests:**
   - Toàn bộ các test assertions kiểm tra kích thước Card trong `internal/ui/ui_test.go` đã được cập nhật sang chuẩn `280x130`.
   - Ca kiểm thử conformance được cập nhật thành `TestArchitectSpec_QuotaStackedLayoutCard280x130_FullConformance`.
5. **Full build bằng script repo nguyên bản:**
   - Chạy `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all` đạt 100% test PASS, biên dịch thành công cả CLI lẫn GUI, exit code 0.
   - Script build `scripts/build_windows.ps1` nguyên vẹn 100%, không bị chỉnh sửa.

---

## 2. Invariant Family Map & Chi Tiết Thi Công

| Invariant / Surface | Target File | Chi Tiết Thi Công Cụ Thể |
|---|---|---|
| **Account Card GridWrap Layout** | `internal/ui/window.go` | - Dòng 185 - 187: Sửa kích thước thẻ trong `GridWrap`:<br>  `a.accountListContainer = container.NewGridWrap(fyne.NewSize(280, 130))` |
| **Unit Test Suite Conformance** | `internal/ui/ui_test.go` | - Cập nhật các test assertion:<br>  + `TestOwnerRedesign_F5_CompactCardHeight`: Cập nhật `card0Size.Width == 280` (thay vì 300).<br>  + `TestCreateMatrixQuotaTable_DetailedMatrixBehavior`: Cập nhật `minSize.Width <= 280.0`.<br>  + `TestIndustryStandard_FixedMatrixMetrics`: Bổ sung kiểm tra mốc `280.0 px` trong vòng lặp resize.<br>  + `TestPurgeAllResizing_AccountGridWrapFixedSize`: Cập nhật `cardSize.Width == 280`.<br>  + Đổi tên và nâng cấp test conformance: `TestArchitectSpec_QuotaStackedLayoutCard280x130_FullConformance` kiểm tra toàn diện AC-1 đến AC-4 với kích thước thẻ `280x130 px`. |

---

## 3. Ranh Giới Touch-Map & Non-Goals

- **Touch-Map Tuân Thủ 100%:** Chỉ chạm đúng 2 file mã nguồn và 1 file báo cáo được phê duyệt trong hợp đồng:
  - `internal/ui/window.go`
  - `internal/ui/ui_test.go`
  - `reports/coder/2026-10-19-compact-card-width-280px.md`
- **NO ANVIEN:** Hoàn toàn không sử dụng Anvien.
- **Không chỉnh sửa script build:** `scripts/build_windows.ps1` nguyên vẹn 100%.
- **Không thay đổi kích thước bảng Quota hay Cửa sổ:** Bảng Quota giữ nguyên `228 x 77 px`, cửa sổ giữ nguyên `1008 x 630 px`.

---

## 4. Bảng Đối Chiếu Tiêu Chí Nghiệm Thu (Acceptance Criteria)

| Tiêu Chí Nghiệm Thu | Kết Quả Thực Tế | Trạng Thái |
|---|---|---|
| **AC-1 (Card Width 280px)** | Thẻ Card có kích thước cố định khóa cứng **`280 x 130 px`** trong `GridWrap` (`fyne.NewSize(280, 130)`). | **PASS** |
| **AC-2 (Quota Table Fitting)** | Bảng Quota 228px nằm vừa vặn, cân đối bên trong thẻ 280px với khoảng đệm an toàn 18px mỗi bên, không bị tràn viền hay ép kích thước. | **PASS** |
| **AC-3 (Window Stability)** | Cửa sổ ứng dụng giữ nguyên kích thước mặc định **`1008 x 630 px`**, hiển thị 3 cột thẻ thẳng hàng, không có thanh cuộn ngang. | **PASS** |
| **AC-4 (Test Suite)** | Chạy `go test -v ./internal/ui` đạt 100% PASS (bao gồm `TestArchitectSpec_QuotaStackedLayoutCard280x130_FullConformance`). | **PASS** |

---

## 5. Minh Chứng Kiểm Thử & Biên Dịch (Mandatory Evidence)

### 5.1. Kết Quả Kiểm Thử Conformance Mới
```text
=== RUN   TestArchitectSpec_QuotaStackedLayoutCard280x130_FullConformance
--- PASS: TestArchitectSpec_QuotaStackedLayoutCard280x130_FullConformance (0.16s)
=== RUN   TestOwnerRedesign_F5_CompactCardHeight
--- PASS: TestOwnerRedesign_F5_CompactCardHeight (0.13s)
=== RUN   TestPurgeAllResizing_AccountGridWrapFixedSize
--- PASS: TestPurgeAllResizing_AccountGridWrapFixedSize (0.09s)
PASS
ok      antigravity-switch/internal/ui  9.752s
```

### 5.2. Kết Quả Full Build Bằng Script Repo (`.\scripts\build_windows.ps1 -Mode all`)
```text
================================================================================
⚡ Antigravity Multi-Account Switcher - Windows Build System
   Mode: all | OutDir: bin | SkipTests: False | Clean: False
================================================================================
[2/4] Running pre-flight tests & vet...
  -> Verifying pure Go packages (CGO_ENABLED=0)...
ok      antigravity-switch/internal/model       (cached)
ok      antigravity-switch/internal/wincred     (cached)
ok      antigravity-switch/internal/store       4.440s
ok      antigravity-switch/internal/scanner     (cached)
ok      antigravity-switch/internal/connector   (cached)
ok      antigravity-switch/internal/reloader    (cached)
ok      antigravity-switch/internal/manager     6.004s
ok      antigravity-switch/cmd/switcher         (cached)
  [PASS] Pure Go verification succeeded.
  -> Verifying GUI packages with CGo & Fyne (CGO_ENABLED=1)...
ok      antigravity-switch/cmd/switcher 1.277s
ok      antigravity-switch/internal/ui  15.716s
  [PASS] GUI verification succeeded.
[3/4] Compiling target binaries...
  -> Checking Windows application icon and resource...
  -> Compiling resource with windres: cmd\switcher\app.rc -> cmd\switcher\rsrc_windows_amd64.syso
  [SUCCESS] Compiled PE resource object: cmd\switcher\rsrc_windows_amd64.syso (12552 bytes)
  -> Building Standalone CLI Binary (Pure Go, CGO_ENABLED=0)...
  [SUCCESS] Created: bin\antigravity-switch.exe
  Compile Duration: 3324 ms | Size: 10.51 MB (11023872 bytes)
  -> Building Standalone GUI Binary (Fyne, CGO_ENABLED=1)...
  [SUCCESS] Created: bin\antigravity-switch-gui.exe
  Compile Duration: 5881 ms | Size: 27.58 MB (28917248 bytes)
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
⚡ BUILD SUCCESSFUL (Exit Code 0)
================================================================================
```

---

## 6. Kết Luận & Bàn Giao

- Công tác thi công thu gọn chiều rộng thẻ tài khoản xuống **`280px`** (kích thước thẻ cố định **`280 x 130 px`**) đã hoàn tất 100%, tuân thủ nghiêm ngặt văn kiện đặc tả kỹ thuật `reports/architect/2026-10-19-compact-card-width-280px-spec.md`.
- Toàn bộ Touch-Map bị khóa cứng đã được thực thi chính xác, không dư thừa, không vi phạm các điều khoản kỷ luật sắt.
- Sẵn sàng bàn giao cho CEO Agent và bộ phận Supervisor kiểm định độc lập Zero-Trust.