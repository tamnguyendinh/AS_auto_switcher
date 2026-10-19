# Coder Implementation Report: Căn Đều Khoảng Đệm Lề Trái và Lề Phải Lưới Thẻ Tài Khoản (Symmetric Balanced Margins)

- **Date:** 2026-10-19
- **Role:** Coder
- **Authority / SSOT:**
  - `reports/architect/2026-10-19-symmetric-grid-wrap-margins-spec.md` (Commit `cbd8257`, Phê duyệt chính thức bởi Human Owner)
  - Mã văn kiện đặc tả: `SPEC-ARCH-20261019-SYMMETRIC-GRID-WRAP-MARGINS`
  - Chỉ đạo từ Human Owner: *"khoang cach 2 ben phải đối xung vơi nhau khi chứa các thẻ, nhin nhu teh se rõ ràng và chueyen nghiệp hơn - hợp lý, viết đặc tả gửi CEO"*
  - `AGENTS.md` & `working-rules`
- **Scope (Touch-Map Khóa Cứng):**
  - `internal/ui/window.go`
  - `internal/ui/ui_test.go`
  - `reports/coder/2026-10-19-symmetric-grid-wrap-margins.md`

---

## 1. Mục Tiêu & Tuân Thủ Kỷ Luật Sắt (Iron Rules Compliance)

1. **Căn lề đối xứng tuyệt đối 2 bên lề (Symmetric Margins):**
   - Loại bỏ thuật toán mặc định của Fyne `container.NewGridWrap` (vốn ghim $x=0$ dồn hết khoảng trống thừa sang lề phải tạo ra khoảng hở lệch thô kệch ~56px).
   - Thay thế bằng custom layout **`symmetricGridWrapLayout`** trong `internal/ui/window.go`:
     - Tự động tính số cột: `cols = floor((width + padding) / (cellSize.Width + padding))`.
     - Tính tổng bề rộng thẻ: `totalCardsW = cols * cellSize.Width`.
     - Khi `width > totalCardsW`: tính `gapX = (width - totalCardsW) / (cols + 1)`.
     - Nếu `gapX >= padding`: `startX = gapX` (Lề trái == Khe giữa == Lề phải).
     - Nếu `gapX < padding`: `gapX = padding`, `startX = (width - (totalCardsW + (cols - 1) * gapX)) / 2.0` (Lề trái == Lề phải cân xứng tuyệt đối).
     - Đảm bảo lề trái và lề phải bằng nhau chính xác (sai lệch $\le 1.0\text{ px}$).
2. **Gióng hàng dọc hoàn hảo giữa các hàng (Aligned Follower Rows):**
   - Toàn bộ các thẻ ở hàng 2, hàng 3 có tọa độ $X$ trùng khớp 100% với cột tương ứng ở hàng 1 (`(i+1)%cols == 0` thì quay về `startX`, `y += cellSize.Height + gapY`).
3. **Bảo tồn nguyên vẹn kích thước & cơ chế đã khóa cứng:**
   - Cửa sổ ứng dụng: CỐ ĐỊNH **`920 x 630 px`** (`DefaultWindowWidth = 920.0`, `DefaultWindowHeight = 630.0`).
   - Thẻ Card: CỐ ĐỊNH **`280 x 130 px`** (`CellSize = fyne.NewSize(280, 130)`).
   - Bảng Quota: CỐ ĐỊNH **`228 x 77 px`**.
   - Cơ chế in-place reactive update (Zero Jank / Zero Canvas Churn) bảo toàn 100%.
4. **Bổ sung Conformance Test:**
   - Thêm ca kiểm thử độc lập `TestSymmetricGridWrapLayout_ExactSymmetry` trong `internal/ui/ui_test.go` xác minh toàn diện cả 4 tiêu chí nghiệm thu AC-1 đến AC-4.
   - Toàn bộ suite test `internal/ui` đạt 100% PASS.
5. **Full build bằng script repo nguyên bản:**
   - Chạy `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all` đạt 100% test PASS, biên dịch thành công cả CLI (`bin/antigravity-switch.exe`) lẫn GUI (`bin/antigravity-switch-gui.exe`), exit code 0.
   - Script build `scripts/build_windows.ps1` nguyên vẹn 100%, tuyệt đối không bị chỉnh sửa.

---

## 2. Invariant Family Map & Chi Tiết Thi Công

| Invariant / Surface | Target File | Chi Tiết Thi Công Cụ Thể |
|---|---|---|
| **Symmetric GridWrap Layout Definition** | `internal/ui/window.go` | - Khai báo struct `symmetricGridWrapLayout` triển khai interface `fyne.Layout` (`Layout` và `MinSize`).<br>- Hàm tạo `newSymmetricGridWrap(cellSize fyne.Size, objects ...fyne.CanvasObject) *fyne.Container`.<br>- Phân bổ đều khoảng đệm lề trái, lề phải và khe giữa các cột.<br>- Gióng thẳng hàng các thẻ ở các hàng tiếp theo. |
| **Grid Container Initialization** | `internal/ui/window.go` | - Thay thế dòng khởi tạo tại `setupLayout`:<br>`a.accountListContainer = newSymmetricGridWrap(fyne.NewSize(280, 130))`<br>thay cho `container.NewGridWrap(fyne.NewSize(280, 130))`. |
| **Unit Test Suite Conformance** | `internal/ui/ui_test.go` | - Thêm unit test `TestSymmetricGridWrapLayout_ExactSymmetry`:<br>  + Kiểm tra đối xứng lề trái và lề phải (`math.Abs(leftMargin - rightMargin) <= 1.0px`).<br>  + Kiểm tra tính đều nhau của khe giữa (`gap1 == gap2 == leftMargin`).<br>  + Kiểm tra gióng hàng dọc hoàn hảo giữa hàng 1 và hàng 2 (`objects[0].X == objects[3].X`, v.v.).<br>  + Kiểm tra branch bề rộng hẹp (`gapX < padding`) lề vẫn đối xứng tuyệt đối.<br>  + Kiểm tra kích thước thẻ cố định 280x130px không co giãn.<br>  + Kiểm tra `app.accountListContainer.Layout` có kiểu `*symmetricGridWrapLayout`. |

---

## 3. Ranh Giới Touch-Map & Non-Goals

- **Touch-Map Tuân Thủ 100%:** Chỉ chạm đúng 2 file mã nguồn và 1 file báo cáo được phê duyệt trong hợp đồng:
  - `internal/ui/window.go`
  - `internal/ui/ui_test.go`
  - `reports/coder/2026-10-19-symmetric-grid-wrap-margins.md`
- **NO ANVIEN:** Hoàn toàn không sử dụng Anvien theo đúng lệnh cấm.
- **Không chỉnh sửa script build:** `scripts/build_windows.ps1` nguyên vẹn 100%.
- **Không thay đổi kích thước cửa sổ hay thẻ:** Cửa sổ giữ nguyên 920 x 630 px, thẻ Card giữ nguyên 280 x 130 px, bảng Quota 228 x 77 px.

---

## 4. Bảng Đối Chiếu Tiêu Chí Nghiệm Thu (Acceptance Criteria)

| Tiêu Chí Nghiệm Thu | Kết Quả Thực Tế | Trạng Thái |
|---|---|---|
| **AC-1 (Symmetric Margins):** Khoảng cách từ mép trái đến thẻ cột 1 và từ thẻ cột 3 đến mép phải bằng nhau (độ lệch $\le 1.0\text{ px}$) | Tại bề rộng hiển thị 888px: `col0 = 12.0px`, `rightMargin = 12.0px`, độ lệch = 0.0px. Tại bề rộng 850px: `leftMargin = 1.0px`, `rightMargin = 1.0px`, độ lệch = 0.0px. | **PASS** |
| **AC-2 (Aligned Follower Rows):** Các hàng tiếp theo có các thẻ gióng thẳng hàng 100% với cột tương ứng ở hàng 1 | Thẻ hàng 2 cột 0 gióng với thẻ hàng 1 cột 0; cột 1 gióng cột 1; cột 2 gióng cột 2 (độ lệch 0.0px). | **PASS** |
| **AC-3 (Dimensions Stability):** Cửa sổ giữ nguyên `920 x 630 px`, thẻ Card giữ nguyên `280 x 130 px`, không phát sinh thanh cuộn ngang | `DefaultWindowWidth = 920.0`, `DefaultWindowHeight = 630.0`. Thẻ Card kích thước cố định 280 x 130 px. | **PASS** |
| **AC-4 (Test Suite):** Chạy `go test -v ./internal/ui` đạt 100% PASS | Toàn bộ 42+ unit test trong `internal/ui` bao gồm test `TestSymmetricGridWrapLayout_ExactSymmetry` mới đều PASS (0 failures). | **PASS** |

---

## 5. Bằng Chứng Biên Dịch & Kiểm Thử (Full Build Evidence)

Lệnh thực thi: `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all`

```text
================================================================================
⚡ Antigravity Multi-Account Switcher - Windows Build System
   Mode: all | OutDir: bin | SkipTests: False | Clean: False
================================================================================
[2/4] Running pre-flight tests & vet...
  -> Verifying pure Go packages (CGO_ENABLED=0)...
ok  	antigravity-switch/internal/model	(cached)
ok  	antigravity-switch/internal/wincred	(cached)
ok  	antigravity-switch/internal/store	5.438s
ok  	antigravity-switch/internal/scanner	(cached)
ok  	antigravity-switch/internal/connector	(cached)
ok  	antigravity-switch/internal/reloader	(cached)
ok  	antigravity-switch/internal/manager	7.792s
ok  	antigravity-switch/cmd/switcher	(cached)
  [PASS] Pure Go verification succeeded.
  -> Verifying GUI packages with CGo & Fyne (CGO_ENABLED=1)...
ok  	antigravity-switch/cmd/switcher	1.407s
ok  	antigravity-switch/internal/ui	9.440s
  [PASS] GUI verification succeeded.
[3/4] Compiling target binaries...
  -> Checking Windows application icon and resource...
  -> Compiling resource with windres: cmd\switcher\app.rc -> cmd\switcher\rsrc_windows_amd64.syso
  [SUCCESS] Compiled PE resource object: cmd\switcher\rsrc_windows_amd64.syso (12552 bytes)
  -> Building Standalone CLI Binary (Pure Go, CGO_ENABLED=0)...
  [SUCCESS] Created: bin\antigravity-switch.exe
  Compile Duration: 2795 ms | Size: 10.51 MB (11023872 bytes)
  -> Building Standalone GUI Binary (Fyne, CGO_ENABLED=1)...
  [SUCCESS] Created: bin\antigravity-switch-gui.exe
  Compile Duration: 5985 ms | Size: 27.59 MB (28929536 bytes)
[4/4] Executing quick verification...
  [PASS] Embedded PE application icon verified on bin\antigravity-switch.exe (32x32)
  [PASS] Embedded PE application icon verified on bin\antigravity-switch-gui.exe (32x32)
  -> Running 'bin\antigravity-switch.exe --status'...
================================================================================
⚡ ANTIGRAVITY SESSION STATUS
================================================================================
Active Email:    skey.vanphongmiennam@gmail.com
Language Server: Running (PID: 18660 | Port: 62220 | CSRF: Present)
Win32 Credential: Present (260 bytes access token)
================================================================================
================================================================================
⚡ BUILD SUCCESSFUL
================================================================================
```

---

## 6. Kết Luận & Bàn Giao

Mục tiêu thi công căn lề đối xứng 2 bên cho lưới thẻ tài khoản theo văn kiện đặc tả `SPEC-ARCH-20261019-SYMMETRIC-GRID-WRAP-MARGINS` đã hoàn thành 100% với chất lượng chuẩn công nghiệp. Sẵn sàng bàn giao cho Supervisor độc lập nghiệm thu.
