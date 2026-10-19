# Coder Implementation Report: Gỡ Bỏ 2 Vách Ngăn Dọc Trong Bảng Quota Ma Trận Thẻ Tài Khoản

- **Date:** 2026-10-19
- **Role:** Coder
- **Authority / SSOT:**
  - `reports/architect/2026-10-19-remove-vertical-separators-quota-matrix-spec.md` (Commit `05d86ee`, Phê duyệt chính thức bởi Human Owner)
  - Mã văn kiện đặc tả: `SPEC-ARCH-20261019-REMOVE-VERTICAL-SEPARATORS-QUOTA-MATRIX`
  - `AGENTS.md` & `working-rules`
- **Scope (Touch-Map Khóa Cứng):**
  - `internal/ui/quota_badge.go`
  - `internal/ui/ui_test.go`
  - `reports/coder/2026-10-19-remove-vertical-separators-quota-matrix.md`

---

## 1. Mục Tiêu & Tuân Thủ Kỷ Luật Sắt (Iron Rules Compliance)

1. **Loại bỏ hoàn toàn 2 vách ngăn dọc (`sep1`, `sep2`):**
   - Đã gỡ bỏ 2 vách ngăn kẻ dọc giữa Cột Model ↔ Cột 5h và giữa Cột 5h ↔ Cột Week trong `createMatrixQuotaTableWithCells`.
   - Bảng Quota chuyển sang sử dụng khoảng đệm tự nhiên (whitespace `colGap = 8.0 px`) tạo cảm giác thanh thoát, khoáng đạt, hiện đại và triệt tiêu visual noise.
2. **Bảo tồn nguyên vẹn vách ngăn ngang (`hSep`):**
   - Vách kẻ ngang mờ (`hSep`, màu `palette.BorderSubtle`, 1px) giữa hàng Gemini và hàng Claude được duy trì tuyệt đối tại vị trí `y = 46.0 px`, chiều cao `1.0 px`, chiều rộng bao phủ đủ 3 cột (`228.0 px`).
3. **Bảo tồn các kích thước và định dạng đã nghiệm thu:**
   - Thẻ Card: CỐ ĐỊNH **`300 x 130 px`** trong `GridWrap`.
   - Cửa sổ ứng dụng: CỐ ĐỊNH **`1008 x 630 px`** (`DefaultWindowWidth = 1008.0`, `DefaultWindowHeight = 630.0`).
   - Typography: Text chính 10pt Bold (`BodyTextSize`), text thời gian đếm lùi 8.5pt `TextMuted`.
   - Bảng Quota Matrix: MinSize cập nhật chuẩn xác **`228.0 x 77.0 px`** (`colModel 48px + colGap 8px + col5h 78px + colGap 8px + colWeek 86px = 228.0 px`).
   - Ray tiến trình MiniBar: CỐ ĐỊNH **`40.0 px`**, `CornerRadius = 3.0 px`.
4. **Bảo toàn 100% cơ chế in-place reactive update (Zero Jank / Zero Canvas Churn):**
   - Quota update tiếp tục cập nhật trực tiếp tại chỗ trên các con trỏ `c.pctText`, `c.timeText`, `c.fillBar.Resize()` và gọi `Refresh()` cục bộ. Tuyệt đối không xóa Canvas hay tái tạo Widget khi cập nhật quota.
5. **Full build bằng script repo nguyên bản:**
   - Chạy `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all` đạt 100% test PASS, biên dịch thành công cả CLI lẫn GUI, exit code 0.
   - Script build `scripts/build_windows.ps1` nguyên vẹn 100%, không bị chỉnh sửa.

---

## 2. Invariant Family Map & Chi Tiết Thi Công

| Invariant / Surface | Target File | Chi Tiết Thi Công Cụ Thể |
|---|---|---|
| **Matrix Table Layout & Spacing** | `internal/ui/quota_badge.go` | - Cập nhật `matrixTableLayout`:<br>  + Nhận 3 cột chính (`colModel`, `col5h`, `colWeek`) và 1 đối tượng tùy chọn `hSep`.<br>  + `MinSize`: Trả về `fyne.NewSize(228.0, 77.0)` dựa trên toán học hình học: `48.0 + 8.0 + 78.0 + 8.0 + 86.0 = 228.0 px`.<br>  + `Layout`: Cột 1 `colModel` (48px) tại `x = 0`; Cột 2 `col5h` (78px) tại `x = 56.0` (`48 + 8`); Cột 3 `colWeek` (86px) tại `x = 142.0` (`56 + 78 + 8`).<br>  + `hSep`: Đặt tại `y = 46.0`, kích thước `(228.0, 1.0)`. |
| **Matrix Quota Table Construction** | `internal/ui/quota_badge.go` | - Trong `createMatrixQuotaTableWithCells`:<br>  + Gỡ bỏ việc khởi tạo `sep1` và `sep2` (`canvas.NewRectangle(palette.BorderSubtle)`).<br>  + Khởi tạo `table := container.New(&matrixTableLayout{}, colModel, col5h, colWeek, hSep)`. |
| **Unit Test Suite Conformance** | `internal/ui/ui_test.go` | - Cập nhật các test assertion:<br>  + `TestOwnerRedesign_AccountCardThreeRows`: Kiểm tra bảng Quota có 4 objects (3 cột + `hSep`), kiểm tra màu `BorderSubtle` trên `hSep`.<br>  + `TestCreateMatrixQuotaTable_DetailedMatrixBehavior`: Cập nhật kiểm tra 4 objects và `hSep`, kiểm tra MinSize `228.0 x 77.0 px`.<br>  + `TestIndustryStandard_FixedMatrixMetrics`: Cập nhật cấu trúc container (3 cột, `hSep`), kiểm tra MinSize `228.0 x 77.0 px`.<br>  + `TestArchitectSpec_QuotaStackedLayoutCard300x130_FullConformance`: Cập nhật MinSize `228.0 x 77.0 px`.<br>  + Bổ sung ca kiểm thử `TestArchitectSpec_RemoveVerticalSeparatorsQuotaMatrix_FullConformance` nghiệm thu toàn diện AC-1 đến AC-5. |

---

## 3. Ranh Giới Touch-Map & Non-Goals

- **Touch-Map Tuân Thủ 100%:** Chỉ chạm đúng 2 file mã nguồn và 1 file báo cáo được phê duyệt trong hợp đồng:
  - `internal/ui/quota_badge.go`
  - `internal/ui/ui_test.go`
  - `reports/coder/2026-10-19-remove-vertical-separators-quota-matrix.md`
- **NO ANVIEN:** Hoàn toàn không sử dụng Anvien.
- **Không chỉnh sửa script build:** `scripts/build_windows.ps1` nguyên vẹn 100%.
- **Không thay đổi layout cố định:** Giữ nguyên Card `300 x 130 px` và Window `1008 x 630 px`.

---

## 4. Bảng Đối Chiếu Tiêu Chí Nghiệm Thu (Acceptance Criteria)

| Tiêu Chí Nghiệm Thu | Kết Quả Thực Tế | Trạng Thái |
|---|---|---|
| **AC-1: No Vertical Separators** | 2 vách ngăn dọc (`sep1`, `sep2`) đã được loại bỏ hoàn toàn. Bảng Quota container chỉ gồm đúng 4 CanvasObject (`colModel`, `col5h`, `colWeek`, `hSep`). Không còn đối tượng phân cách dọc nào giữa các cột. | **PASS** |
| **AC-2: Horizontal Separator Preserved** | Vách ngăn kẻ ngang `hSep` (1px, `palette.BorderSubtle`) giữa hàng Gemini và Claude được duy trì nguyên vẹn tại `y = 46.0 px`, chiều rộng `228.0 px`. | **PASS** |
| **AC-3: Natural Spacing (Whitespace 8px)** | Khoảng cách giữa các cột được quy định bằng `colGap = 8.0 px`. Kích thước tối thiểu của bảng Quota đạt chuẩn toán học `228.0 x 77.0 px` (`48 + 8 + 78 + 8 + 86 = 228 px`). Vị trí cột: `colModel` tại `x = 0`, `col5h` tại `x = 56.0`, `colWeek` tại `x = 142.0`. | **PASS** |
| **AC-4: Card & Window Dimensions Preserved** | Thẻ Card giữ nguyên `300 x 130 px` trong `GridWrap`. Cửa sổ ứng dụng giữ nguyên `1008 x 630 px` (`DefaultWindowWidth = 1008.0`, `DefaultWindowHeight = 630.0`). | **PASS** |
| **AC-5: In-Place Reactive Update & Test Suite** | Cơ chế cập nhật in-place qua `cell.update()` bảo toàn 100%, không hủy bỏ/tái tạo đối tượng Canvas, zero layout jank. 100% test suite chạy thành công (`go test -tags gui ./internal/ui/...` PASS). | **PASS** |

---

## 5. Minh Chứng Kiểm Thử & Biên Dịch (Mandatory Evidence)

### 5.1. Kết Quả Kiểm Thử Conformance Mới
```text
=== RUN   TestArchitectSpec_RemoveVerticalSeparatorsQuotaMatrix_FullConformance
--- PASS: TestArchitectSpec_RemoveVerticalSeparatorsQuotaMatrix_FullConformance (0.06s)
PASS
ok      antigravity-switch/internal/ui  1.375s
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
ok      antigravity-switch/internal/store       4.245s
ok      antigravity-switch/internal/scanner     (cached)
ok      antigravity-switch/internal/connector   (cached)
ok      antigravity-switch/internal/reloader    (cached)
ok      antigravity-switch/internal/manager     5.720s
ok      antigravity-switch/cmd/switcher         (cached)
  [PASS] Pure Go verification succeeded.
  -> Verifying GUI packages with CGo & Fyne (CGO_ENABLED=1)...
ok      antigravity-switch/cmd/switcher 1.327s
ok      antigravity-switch/internal/ui  8.843s
  [PASS] GUI verification succeeded.
[3/4] Compiling target binaries...
  -> Checking Windows application icon and resource...
  -> Compiling resource with windres: cmd\switcher\app.rc -> cmd\switcher\rsrc_windows_amd64.syso
  [SUCCESS] Compiled PE resource object: cmd\switcher\rsrc_windows_amd64.syso (12552 bytes)
  -> Building Standalone CLI Binary (Pure Go, CGO_ENABLED=0)...
  [SUCCESS] Created: bin\antigravity-switch.exe
  Compile Duration: 2792 ms | Size: 10.51 MB (11023872 bytes)
  -> Building Standalone GUI Binary (Fyne, CGO_ENABLED=1)...
  [SUCCESS] Created: bin\antigravity-switch-gui.exe
  Compile Duration: 5459 ms | Size: 27.58 MB (28917248 bytes)
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

- Toàn bộ nội dung công việc theo đặc tả `reports/architect/2026-10-19-remove-vertical-separators-quota-matrix-spec.md` đã được thi công chính xác 100%, bảo toàn các quy chuẩn hình học, giao diện thoáng đãng, hiện đại và không có bất kỳ hồi quy (regression) nào.
- Residual unverified surfaces: **none**.
- Trạng thái bàn giao: **VERDICT: PASS / READY_FOR_SUPERVISOR**.
