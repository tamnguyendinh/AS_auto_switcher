# Supervisor Report: Nghiệm Thu Chặng 1 - Giao Diện Cụm Điều Khiển Hàng 2 (Row 2 Right Cluster UI)

Verdict: REJECT

## Metadata
- Report file: `reports/supervisor/2026-10-20-account-privacy-ui-phase1-acceptance.md`
- Review time: 2026-10-20 21:58:00 +07:00
- Reviewer: Supervisor Lane
- Repo/project: `antigravity_auto_switch`
- Scope reviewed: Commit `33d1b3227dcc5c6223d7f160e0e75e8ade3b682e` (diff trên `internal/ui/app.go`, `internal/ui/themes.go`, `internal/ui/window.go`, `internal/ui/ui_test.go`, `reports/coder/2026-10-20-account-privacy-ui-phase1-report.md`)
- Claim reviewed: "Hoàn thành 100% Chặng 1 - Giao diện thanh công cụ cụm bên phải Hàng 2 với 7 phần tử Canvas, bảo toàn nguyên vẹn nút gạt Tự động có sẵn, toàn bộ test suite và build pass 100%."
- Authority used:
  - `reports/architect/2026-10-20-account-privacy-masking-toolbar-spec.md` (commit `7e85f3f`)
  - Mệnh lệnh tối cao từ Human Owner:
    1. Nút gạt "Tự động" cũ ĐƯỢC BẢO TOÀN NGUYÊN VẸN 100%, không bị sửa đổi hay phá vỡ.
    2. Bố cục cụm 7 phần tử Canvas Hàng 2 đúng thứ tự: `[Ẩn tài khoản: ○] │ [Tự động: ●] │ [ ☀️ ▼ ]`.
    3. Hai thanh Divider 1x14 px bọc `container.NewGridWrap(fyne.NewSize(1, 14), divRect)`, không bị teo nhỏ về (0, 0).
    4. Nút gạt Ẩn tài khoản đạt chuẩn `CapsuleToggle` 23x12 px.
    5. Đồng bộ màu Palette Theme (`BorderSubtle` / `Primary`) cho Divider và Toggle.
    6. Toàn bộ test suite UI (`go test -v ./internal/ui`) và build (`pwsh -File .\scripts\build_windows.ps1 -Mode all`) chạy đạt PASS độc lập. Nghiêm cấm chạy `go build` trần.
- Related artifacts:
  - `reports/architect/2026-10-20-account-privacy-masking-toolbar-spec.md`
  - `reports/coder/2026-10-20-account-privacy-ui-phase1-report.md`

---

## Executive Summary
- **Problem:** Kiểm định Zero-Trust độc lập việc thi công Chặng 1 (Giao diện Cụm điều khiển Hàng 2) tại commit `33d1b3227dcc5c6223d7f160e0e75e8ade3b682e` theo 6 tiêu chí chỉ đạo của Owner và kiến trúc đã phê duyệt.
- **Decision:** **REJECT**. Dù phần hình học, bố cục 7 Canvas objects và bảo toàn nút gạt Tự động đạt chuẩn 100%, quy trình nghiệm thu độc lập phát hiện **Test suite và Script Build chuẩn của repo (`pwsh -File .\scripts\build_windows.ps1 -Mode all`) BỊ THẤT BẠI (FAIL)** do lỗi chạy bất đồng bộ không kiểm soát gây race condition rác file tạm tại `TestArchitectSpec_Row2Layout_ThemeSync` (`internal/ui/ui_test.go:3933-3977`).
- **Required outcome:** Trả về Coder Lane để khắc phục dứt điểm rò rỉ goroutine / race condition trong kiểm thử đồng bộ Theme/Toggle persistence, đảm bảo `pwsh -File .\scripts\build_windows.ps1 -Mode all` và `go test -count=1 ./internal/ui` PASS 100% trước khi nghiệm thu lại.

---

## Blocking Findings

### [CRITICAL] Build Script Repo Thất Bại Tại Giai Đoạn Pre-flight GUI Test Do Lỗi TempDir Cleanup Trong `TestArchitectSpec_Row2Layout_ThemeSync`
- **File:** `scripts/build_windows.ps1:81`, `internal/ui/ui_test.go:3933-3977`, `internal/ui/app.go:876-904`
- **Issue:** Khi thực thi script build chuẩn repo theo đúng lệnh Owner:
  ```powershell
  pwsh -File .\scripts\build_windows.ps1 -Mode all
  ```
  Quá trình dừng đột ngột và báo lỗi tại bước `[2/4] Running pre-flight tests & vet... -> Verifying GUI packages with CGo & Fyne`:
  ```text
  --- FAIL: TestArchitectSpec_Row2Layout_ThemeSync (0.26s)
      testing.go:1464: TempDir RemoveAll cleanup: unlinkat C:\Users\TAMNGU~1\AppData\Local\Temp\TestArchitectSpec_Row2Layout_ThemeSync140738810\001: The directory is not empty.
  FAIL
  FAIL	antigravity-switch/internal/ui	20.554s
  FAIL
  Write-Error: G:\antigravity_auto_switch\scripts\build_windows.ps1:81
  GUI test suite failed
  ```
- **Root Cause Analysis:**
  1. Trong `internal/ui/app.go`, hàm `SetMaskAccounts(mask bool)` khởi chạy một background goroutine unmanaged:
     ```go
     go func() {
         pref, err := LoadUIPreferences(path)
         if err != nil || pref == nil {
             pref = &UIPreferences{Theme: theme}
         }
         pref.MaskAccounts = mask
         _ = SaveUIPreferences(path, pref)
     }()
     ```
  2. Trong `TestArchitectSpec_Row2Layout_ThemeSync` (`internal/ui/ui_test.go`), vòng lặp kiểm tra 3 themes (`ThemeDark`, `ThemeLight`, `ThemeWarm`) thực hiện:
     ```go
     maskToggle.SetOn(true)
     ...
     maskToggle.SetOn(false)
     ```
  3. `maskToggle.SetOn` kích hoạt `onChange` $\rightarrow$ gọi `a.SetMaskAccounts` $\rightarrow$ sinh ra hàng loạt background goroutines ghi file `.ui_preferences_tmp_*.json` và rename đè vào `tmpDir/.ui_preferences.json`.
  4. Test không chờ các goroutine này hoàn tất trước khi kết thúc hàm test. Khi test kết thúc, Go framework tự động dọn dẹp thư mục tạm qua `t.TempDir() RemoveAll`.
  5. Trên hệ điều hành Windows, việc xóa thư mục đang có tiến trình con / luồng ngầm mở handle ghi file hoặc đổi tên file lập tức bị hệ điều hành từ chối với mã lỗi `The directory is not empty`, dẫn đến việc test bị đánh dấu FAIL và toàn bộ build pipeline của repo bị chặn đứng.
- **Why this blocks acceptance:** Vi phạm trực tiếp Tiêu chí 6 của Hợp đồng và Chỉ đạo sắt của Owner ("Toàn bộ test suite UI và build bằng script repo phải PASS độc lập").
- **Fix direction:**
  - Trong `TestArchitectSpec_Row2Layout_ThemeSync`, việc kiểm tra màu `FillColor` của Toggle track theo Palette Theme không cần thiết phải kích hoạt luồng ghi đĩa ngầm (hoặc cần đợi goroutine hoàn tất / không gọi hàm `SetOn` kích hoạt `onChange` ghi đĩa vào temp dir đã đóng).
  - Hoặc trong `App`, cơ chế lưu preferences bất đồng bộ cần có cơ chế tracking/drain khi đóng app (`app.Close()`).
- **Re-review evidence required:**
  - Output thực tế của `pwsh -File .\scripts\build_windows.ps1 -Mode all` đạt mã thoát 0 (Exit code 0), build thành công cả 2 binary CLI và GUI vào thư mục `bin/`.
  - Output của `go test -count=1 -v -run "^TestArchitectSpec_Row2Layout_ThemeSync$" ./internal/ui` PASS hoàn toàn không còn lỗi `TempDir RemoveAll cleanup`.

---

## Source-Level Clearance Notes

| File / Component | Trạng Thái | Đánh Giá Chi Tiết |
|---|---|---|
| `internal/ui/window.go` (Bảo toàn nút Tự động) | **PASS** | `AutoSwitchToggle` được bảo toàn 100% nguyên vẹn (lines 485–603), không bị sửa đổi hay thay đổi logic. Coder tuân thủ nghiêm ngặt chỉ đạo của Human Owner. |
| `internal/ui/window.go` (Bố cục 7 phần tử Hàng 2) | **PASS** | `rightCluster` xếp đúng 7 Canvas objects theo đúng thứ tự: `[lblMask, a.maskAccountsToggle, div1Wrap, lblAuto, a.autoSwitchToggle, div2Wrap, themeSelectWrap]`. |
| `internal/ui/window.go` (Divider 1x14 GridWrap) | **PASS** | Cả 2 Divider `row2Divider1` và `row2Divider2` đều được bọc trong `container.NewGridWrap(fyne.NewSize(1, 14), ...)`, loại bỏ hoàn toàn bẫy MinSize (0, 0). |
| `internal/ui/window.go` (CapsuleToggle 23x12) | **PASS** | `MaskAccountsToggle` có MinSize chuẩn 23x12 px, thumb 8x8 px, tọa độ switch 2px và 13px chuẩn xác. |
| `internal/ui/themes.go` | **PASS** | `UIPreferences` bổ sung `MaskAccounts bool` `json:"mask_accounts"` chính xác, tương thích ngược với JSON cũ. |
| `internal/ui/app.go` (Theme Palette Sync) | **PASS** | Trong `SetTheme`, cả 2 Divider và cả 2 Toggle đều được cập nhật màu `BorderSubtle` và `Primary` tương ứng theo Theme. |
| `internal/ui/app.go` (Async Persistence) | **BLOCKED** | Hàm `SetMaskAccounts` tạo goroutine không theo dõi gây lỗi rác file tạm khi chạy test / build script. |
| `internal/ui/ui_test.go` | **BLOCKED** | `TestArchitectSpec_Row2Layout_ThemeSync` thất bại do lỗi rác file tạm trên Windows (`The directory is not empty`). |

---

## Evidence Checked

### Passed:
1. `go test -v -run "^TestArchitectSpec_Row2Layout_WithDividers$" ./internal/ui`: PASS (0.61s) - Xác nhận 7 phần tử Canvas, kích thước Divider 1x14, kích thước Toggle 23x12.
2. `go test -v -run "^TestArchitectSpec_AutoSwitchToggle" ./internal/ui`: PASS (0.82s) - Xác nhận nút Tự động cũ hoạt động nguyên vẹn 100%.
3. `go test -v -run "^TestHeaderRow2_VerticalAlignment$" ./internal/ui`: PASS (0.14s) - Độ lệch trục dọc giữa các phần tử đạt chuẩn.
4. `go test -v -run "^TestArchitectSpec_MaskAccounts_Persistence$" ./internal/ui`: PASS (0.25s) - Lưu nạp preferences hoạt động chuẩn khi có sleep điều phối.

### Failed:
1. `pwsh -File .\scripts\build_windows.ps1 -Mode all`:
   - Command: `pwsh -File .\scripts\build_windows.ps1 -Mode all`
   - Result: Exit code 1.
   - Evidence log:
     ```text
     --- FAIL: TestArchitectSpec_Row2Layout_ThemeSync (0.26s)
         testing.go:1464: TempDir RemoveAll cleanup: unlinkat C:\Users\TAMNGU~1\AppData\Local\Temp\TestArchitectSpec_Row2Layout_ThemeSync140738810\001: The directory is not empty.
     FAIL
     FAIL	antigravity-switch/internal/ui	20.554s
     FAIL
     Write-Error: G:\antigravity_auto_switch\scripts\build_windows.ps1:81
     GUI test suite failed
     ```
2. `go test -count=1 ./internal/ui`:
   - Command: `go test -count=1 ./internal/ui`
   - Result: Exit code 1.
   - Evidence log:
     ```text
     --- FAIL: TestArchitectSpec_Row2Layout_ThemeSync (0.11s)
         testing.go:1464: TempDir RemoveAll cleanup: unlinkat C:\Users\TAMNGU~1\AppData\Local\Temp\TestArchitectSpec_Row2Layout_ThemeSync2173433748\001: The directory is not empty.
     FAIL
     FAIL	antigravity-switch/internal/ui	14.212s
     FAIL
     ```

---

## Invariant Closure
- **Affected Invariant:** Cụm thanh công cụ Hàng 2 (Row 2 Right Cluster UI) và tính ổn định của quy trình build & test tự động của repository.
- **Sibling surfaces checked:** `AutoSwitchToggle`, `themeSelectWrap`, `UIPreferences`, `build_windows.ps1`.
- **Residual unverified same-invariant surfaces:** Tính ổn định khi chạy đồng thời hoặc lặp lại nhiều lần của suite test UI trên môi trường Windows do rò rỉ goroutine ghi file tạm.

---

## Required Fix List For Resubmission

1. **Khắc phục lỗi TempDir Cleanup trong `TestArchitectSpec_Row2Layout_ThemeSync`:**
   - Điều chỉnh việc kiểm tra trạng thái visual của Toggle (track `FillColor`) sao cho không kích hoạt chuỗi ghi đĩa async không kiểm soát vào temp dir đang trong vòng đời dọn dẹp, HOẶC bổ sung cơ chế đồng bộ (drain / sync) trước khi kết thúc test.
2. **Xác nhận qua Build Script Chuẩn Của Repo:**
   - Chạy lệnh `pwsh -File .\scripts\build_windows.ps1 -Mode all` thành công 100%, ghi lại toàn bộ output không có lỗi để nộp bằng chứng.
3. **Cập nhật Báo cáo Coder:**
   - Cập nhật báo cáo thi công ghi nhận bằng chứng build từ script repo và kết quả test uncached (`-count=1`).

---

## Overall Evaluation
Coder lane đã làm rất tốt phần nghiệp vụ UI cốt lõi: bảo tồn tuyệt đối 100% nút gạt Tự động cũ theo đúng mệnh lệnh của Human Owner, dựng đúng chuẩn 7 phần tử Canvas, bao bọc Divider trong GridWrap 1x14 px và thiết kế Toggle 23x12 px rất chuẩn mực. Tuy nhiên, việc để lọt lỗi goroutine race condition trong test khiến script build repo chính thức bị thất bại tại cổng kiểm định Zero-Trust. Báo cáo nghiệm thu ban hành kết luận **REJECT** để Coder hoàn thiện triệt để trước khi chuyển giao sang Chặng 2.
