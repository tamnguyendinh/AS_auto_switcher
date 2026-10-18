# Báo Cáo Coder R6: Khắc Phục Triệt Để & Dứt Điểm Lỗi Tồn Đọng F1 (Positive Legacy Với Hai Cycle Zero), Bảo Toàn Tuyệt Đối F2, F3, F4, F5, F6

**Tệp tin:** `reports/coder/2026-10-18-ui-redesign-r6-fix-report.md`  
**Ngày:** 2026-10-18  
**Tác giả:** Coder Specialist Lane  
**Người nhận:** CEO Agent (`01a14378-4a08-77c1-88d5-666ad8c3592d`), Supervisor Lane  
**Authority:** Chỉ thị CEO; Báo cáo Supervisor R5 `reports/supervisor/2026-10-18-ui-redesign-r5-supervisor-report.md` (commit `cb529ca`); Touch-Map Addendum `reports/architect/2026-10-18-touchmap-addendum-gpt-removal.md`; Đặc tả kiến trúc `reports/architect/2026-10-17-ui-redesign-spec-from-owner-drawing.md`; Bản vẽ thiết kế Owner `reports/problem/images/9180d29a.jpg`.  
**Verdict:** `[VERDICT: READY_FOR_SUPERVISOR]`

---

## 1. Mục Tiêu & Phạm Vi Công Việc

Khắc phục lỗi tồn đọng F1 duy nhất còn lại của toàn bộ hệ thống được chỉ ra trong Báo cáo Supervisor R5:
1. **F1 [P2] Loại bỏ hoàn toàn nhánh đặc thù `RemainingFraction > 0` phủ nhận hai cycle zero tường minh:**
   - **Vấn đề gốc tại R5:** Khối `if` đặc thù tại `internal/ui/quota_badge.go:73-77`:
     ```go
     // Special case: if Go struct literal in tests only initialized RemainingFraction > 0 and left cycle fractions as 0
     if bucket.RemainingFraction > 0 && bucket.FiveHourFraction == 0 && bucket.WeeklyFraction == 0 && bucket.FiveHourReset.IsZero() && bucket.WeeklyReset.IsZero() {
         has5h = false
         hasWeek = false
     }
     ```
     Nhánh code này vốn được thêm vào để hỗ trợ các Go struct literal cũ trong test chỉ gán `RemainingFraction: 1.0` (khiến `FiveHourFraction` và `WeeklyFraction` nhận giá trị zero value `0.0` của Go). Tuy nhiên, trong môi trường sản phẩm thực tế khi cache lưu `remaining_fraction: 0.2`, `five_hour_fraction: 0.0`, `weekly_fraction: 0.0` mà không có raw bucket hoặc reset time, nhánh này vô tình phủ nhận hai cycle zero thật, làm rơi dữ liệu thành `(20%, --%)` thay vì hiển thị trung thực `(0%, 0%)`.
   - **Khắc phục tại R6:**
     - Loại bỏ hoàn toàn khối `if` đặc thù này khỏi `internal/ui/quota_badge.go`. Dữ liệu cycle tường minh (`FiveHourFraction >= 0` hoặc `WeeklyFraction >= 0`) **LUÔN LUÔN ĐƯỢC ƯU TIÊN** bất kể `RemainingFraction` mang giá trị nào.
     - Fallback legacy chỉ kích hoạt khi thực sự **KHÔNG CÓ** dữ liệu cycle-specific (`FiveHourFraction < 0 && WeeklyFraction < 0`).
     - Chuẩn hóa toàn bộ các test fixture cũ trong `internal/ui/ui_test.go` và `internal/telemetry/quota_test.go` có ý định kiểm thử legacy single-bucket fallback bằng cách thiết lập rõ ràng sentinel `FiveHourFraction: -1, WeeklyFraction: -1` thay vì để code sản phẩm đoán mò.
     - Bổ sung regression tests chặt chẽ trong cả `internal/telemetry/quota_test.go` (`TestF1_PositiveLegacyFraction_WithBothCyclesZero`) và `internal/ui/ui_test.go` (`TestF1_ExtractDualCycleQuota_PositiveLegacyFraction_WithBothCyclesZero`).
     - Bảo đảm 100% các trường hợp legacy-only single bucket, missing fraction (`-1`), weekly-only zero, raw bucket supplement, reset time và asymmetric zero vẫn chạy chính xác.
2. **Bảo toàn tuyệt đối các Invariant đã đạt:**
   - **F2 (Zero GPT):** 0 field `GPT` trong Go struct `AccountQuotaData`, 0 khóa `"gpt"` trong JSON serialization.
   - **F3 (Single card refresh):** Refresh in-place giữ nguyên 5 sibling cards, focus và disable state.
   - **F4 (Retired helpers cleanup):** Dọn sạch 5 helper retired, staticcheck 0 diagnostic.
   - **F5 (Compact card height):** Chiều cao thẻ compact ~82px (3 dòng: topRow, row5h, rowWeek).
   - **F6 (Monogram AS theme):** Monogram [AS] đổi màu theo theme (Sáng #2563eb, Ấm #d97706, Tối #89b4fa).

---

## 2. Invariant Family Map

| Thành phần | SSOT / Quy định kỹ thuật | Bề mặt mã nguồn (Touch-Map) | Hành vi chuẩn hóa R6 |
|---|---|---|---|
| **F1: Explicit Cycle Fractions Priority** | Spec Owner, Báo cáo Supervisor R5 | `internal/ui/quota_badge.go:68-76` | Mọi giá trị cycle tường minh (`>= 0`) được gán trực tiếp cho `h5` và `week`. Tuyệt đối không phỏng đoán hay phủ nhận cycle zero khi `RemainingFraction > 0`. |
| **F1: Strict Legacy Fallback Boundary** | Spec Owner, Báo cáo Supervisor R5 | `internal/ui/quota_badge.go:114-118` | Fallback sang `RemainingFraction` chỉ xảy ra khi và chỉ khi `h5 < 0 && week < 0 && bucket.RemainingFraction >= 0`. Không bao giờ ghi đè khi đã có cycle `>= 0`. |
| **F1: Test Fixtures Standardization** | Chỉ thị CEO R6 | `internal/ui/ui_test.go`, `internal/telemetry/quota_test.go` | Mọi struct literal mô phỏng legacy single-bucket phải khai báo tường minh `FiveHourFraction: -1, WeeklyFraction: -1`. |
| **F1: Truthful Dual-Cycle Rendering** | Spec Owner, Báo cáo Supervisor R5 | `internal/ui/widgets.go:137-143`, `internal/ui/quota_badge.go:142-180` | Khi cache có `remaining_fraction: 0.2` kèm `five_hour_fraction: 0`, `weekly_fraction: 0`, giao diện render trung thực `0%` cho 5h và `0%` cho Weekly (màu đỏ #ef4444), không rớt thành `20%` hay `--%`. |
| **F2: Zero GPT Contract** | ADR-20261018-GPT-REMOVAL, Addendum Architect | `internal/telemetry/client.go` | 0 field `GPT` trong struct `AccountQuotaData`, 0 khóa `"gpt"` trong JSON serialization. |
| **F3: Single Card Refresh** | Spec Owner | `internal/ui/app.go:407` | Refresh in-place chỉ thay thế `Objects[targetIdx]`, bảo toàn 5/5 sibling cards và trạng thái giao diện. |
| **F4: Retired Helpers Cleanup** | Báo cáo Supervisor R3, R4, R5 | `internal/ui/quota_badge.go`, `internal/ui/widgets.go`, `internal/ui/app.go` | 5 helper retired và 4 field cũ trong `ui.App` đã bị loại bỏ hoàn toàn; staticcheck 0 diagnostic. |
| **F5: Compact Card Height** | Bản vẽ Owner, Spec Owner | `internal/ui/widgets.go:88-154` | Chiều cao thẻ ~82px (đúng 3 dòng: topRow, row5h, rowWeek). |
| **F6: Monogram AS Theme Palette** | Bản vẽ Owner, Addendum Architect | `internal/ui/theme.go`, `internal/ui/app.go:273` | Monogram [AS] nền đổi màu động theo palette: Sáng #2563eb, Ấm #d97706, Tối #89b4fa. |

---

## 3. Chi Tiết Các Thay Đổi Trong Touch-Map

### 3.1 `internal/ui/quota_badge.go`
- **Xóa bỏ hoàn toàn khối `if` đặc thù tại dòng 73-77:**
  ```go
  // ĐÃ LOẠI BỎ:
  // Special case: if Go struct literal in tests only initialized RemainingFraction > 0 and left cycle fractions as 0
  if bucket.RemainingFraction > 0 && bucket.FiveHourFraction == 0 && bucket.WeeklyFraction == 0 && bucket.FiveHourReset.IsZero() && bucket.WeeklyReset.IsZero() {
      has5h = false
      hasWeek = false
  }
  ```
- **Chuẩn hóa hàm `extractFamilyCycles`:**
  ```go
  // Explicit cycle fractions (>= 0) always take precedence regardless of RemainingFraction.
  if bucket.FiveHourFraction >= 0 {
      h5 = bucket.FiveHourFraction
  }
  if bucket.WeeklyFraction >= 0 {
      week = bucket.WeeklyFraction
  }
  ```
  Nhờ đó, khi `FiveHourFraction == 0.0` và `WeeklyFraction == 0.0`, cả hai đều `>= 0` nên `h5 = 0.0` và `week = 0.0` được bảo toàn nguyên vẹn. Raw buckets không ghi đè vì `h5 < 0` và `week < 0` đều sai. Legacy fallback không kích hoạt vì `h5 < 0 && week < 0` sai. Kết quả trả về chính xác `(0.0, 0.0)`.

### 3.2 `internal/ui/ui_test.go`
- **Chuẩn hóa test fixtures cũ:** Thêm `FiveHourFraction: -1, WeeklyFraction: -1` vào các struct literal biểu diễn quota đơn bucket cũ (dòng 574, 653, 690, 798, 879, 941, 955, 1017, 1227, 1383).
- **Bổ sung test regression `TestF1_ExtractDualCycleQuota_PositiveLegacyFraction_WithBothCyclesZero`:**
  - Case 1: Cả Claude và Gemini có `RemainingFraction > 0` (0.2 và 0.8) kèm `FiveHourFraction: 0.0, WeeklyFraction: 0.0` không có raw/reset -> `ExtractDualCycleQuota` trả về `(0.0, 0.0)` cho cả hai family; thẻ giao diện render đúng bốn nhãn `0%`, không xuất hiện `20%`, `80%` hay `--%`.
  - Case 2: Asymmetric Claude positive legacy (0.25) kèm hai cycle zero, Gemini missing (`-1`) -> Claude `(0.0, 0.0)`, Gemini `(-1.0, -1.0)`.
  - Case 3: Asymmetric Gemini positive legacy (0.75) kèm hai cycle zero, Claude missing (`-1`) -> Claude `(-1.0, -1.0)`, Gemini `(0.0, 0.0)`.
  - Case 4: True legacy fallback (`FiveHourFraction: -1, WeeklyFraction: -1`) với `RemainingFraction` dương -> trả về đúng `(RemainingFraction, -1.0)`.

### 3.3 `internal/telemetry/quota_test.go`
- **Chuẩn hóa test fixture dòng 112:** Thêm `FiveHourFraction: -1, WeeklyFraction: -1`.
- **Bổ sung test regression `TestF1_PositiveLegacyFraction_WithBothCyclesZero`:**
  - Case 1: JSON unmarshaling từ cache với `remaining_fraction: 0.2` (hoặc `0.8`) và `five_hour_fraction: 0.0, weekly_fraction: 0.0` bảo toàn đầy đủ các giá trị cycle fractions `0.0`.
  - Case 2: Chu trình ghi đĩa và đọc lại từ đĩa (`SaveToDisk` và `LoadFromDisk`) bảo toàn nguyên vẹn `RemainingFraction` dương và các cycle fractions `0.0`, không có raw buckets.

### 3.4 `reports/coder/evidence/2026-10-18-ui-redesign-f1-f6/coder_probe_r6.go`
- Nâng cấp probe độc lập của Coder từ 84 assertions (R5) lên **114 assertions (R6)**, bổ sung:
  - F1.13: Case quan trọng nhất của R6 — `RemainingFraction` dương (0.2 / 0.8) kèm hai cycle zero `0.0`: kiểm tra `ExtractDualCycleQuota`, kiểm tra widget canvas card text (4x `0%`, 0x `20%`, 0x `80%`, 0x `--%`), kiểm tra cache persistence round-trip.
  - F1.14: Asymmetric positive legacy scalar kèm hai cycle zero cho từng family độc lập.
  - F1.15: True legacy fallback khi cycle fractions bằng `-1`.

---

## 4. Bằng Chứng Kiểm Định Thực Nghiệm (Verification Evidence)

### 4.1 Coder Independent Acceptance Probe R6 (`coder_probe_r6.go`)
- **Lệnh thực thi:** `go run reports/coder/evidence/2026-10-18-ui-redesign-f1-f6/coder_probe_r6.go`
- **Kết quả:** **114/114 assertions PASSED (Exit Code: 0)**
  - F1.1 đến F1.12: Toàn bộ 12 test case F1 lịch sử (REST amounts, missing fractions, summary weekly zero, cache persistence, asymmetric zero) đều **PASS 100%**.
  - **F1.13 (Mới R6 — Trọng tâm):**
    - `F1.13_PosLegacyZero_HasDataTrue`: **PASS**
    - `F1.13_PosLegacyZero_Claude5hZero` & `ClaudeWeekZero`: **PASS** (trả về đúng `(0.0, 0.0)`)
    - `F1.13_PosLegacyZero_Gemini5hZero` & `GeminiWeekZero`: **PASS** (trả về đúng `(0.0, 0.0)`)
    - `F1.13_PosLegacyZero_Card_Claude5h0` & `ClaudeWeek0`: **PASS**
    - `F1.13_PosLegacyZero_Card_Gemini5h0` & `GeminiWeek0`: **PASS**
    - `F1.13_PosLegacyZero_NoLegacyFractionsInCard`: **PASS** (không có `20%` hay `80%`)
    - `F1.13_PosLegacyZero_NoUnknownPlaceholdersInCard`: **PASS** (không có `--%`)
    - `F1.13_CacheRoundTrip_ReloadedTupleExact`: **PASS** (nạp lại từ đĩa bảo toàn đúng `(0.0, 0.0, 0.0, 0.0)`)
  - **F1.14 (Mới R6):** Asymmetric positive legacy với zero cycles: **PASS**
  - **F1.15 (Mới R6):** True legacy fallback khi cycle fractions `-1`: **PASS**
  - F2: Zero GPT field trong struct & zero `"gpt"` trong JSON serialization: **PASS**
  - F4: 4 retired UI fields absent khỏi `ui.App`: **PASS**
  - F5: Card layout compact có đúng 3 rows: **PASS**
  - F6: Theme palette colors khớp Sáng #2563eb, Ấm #d97706, Tối #89b4fa: **PASS**

### 4.2 Full Workspace Test Suite
- **Lệnh thực thi:** `go test -v -count=1 ./...`
- **Kết quả:** **10/10 packages PASSED (165 tests, 0 FAIL, 0 SKIP, Exit Code: 0)**
  - `cmd/switcher`: **PASS** (4.059s)
  - `internal/connector`: **PASS** (3.452s)
  - `internal/manager`: **PASS** (13.872s)
  - `internal/model`: **PASS** (1.152s)
  - `internal/reloader`: **PASS** (4.783s)
  - `internal/scanner`: **PASS** (1.355s)
  - `internal/store`: **PASS** (17.491s)
  - `internal/telemetry`: **PASS** (16.825s, thêm `TestF1_PositiveLegacyFraction_WithBothCyclesZero`)
  - `internal/ui`: **PASS** (14.434s, thêm `TestF1_ExtractDualCycleQuota_PositiveLegacyFraction_WithBothCyclesZero`)
  - `internal/wincred`: **PASS** (0.948s)

### 4.3 Staticcheck
- **Lệnh thực thi:** `staticcheck -tags gui ./...`
- **Kết quả:** **Exit Code: 0, 0 diagnostic**

### 4.4 Native GUI Build
- **Lệnh thực thi:** `go build -tags gui ./cmd/switcher`
- **Kết quả:** **Exit Code: 0 (Biên dịch thành công không lỗi)**

---

## 5. Danh Sách Tệp Tin Chỉnh Sửa Trong Touch-Map

1. `internal/ui/quota_badge.go` (Xóa bỏ nhánh đặc thù `RemainingFraction > 0` phủ nhận hai cycle zero tường minh; ưu tiên cycle fractions `>= 0` trên mọi trường hợp)
2. `internal/ui/ui_test.go` (Chuẩn hóa struct literal cũ với `-1`; thêm test regression `TestF1_ExtractDualCycleQuota_PositiveLegacyFraction_WithBothCyclesZero`)
3. `internal/telemetry/quota_test.go` (Chuẩn hóa struct literal cũ với `-1`; thêm test regression `TestF1_PositiveLegacyFraction_WithBothCyclesZero`)
4. `reports/coder/evidence/2026-10-18-ui-redesign-f1-f6/coder_probe_r6.go` (Probe kiểm chứng độc lập R6 gồm 114 assertions)
5. `reports/coder/2026-10-18-ui-redesign-r6-fix-report.md` (Báo cáo chính thức R6 này)

---

## 6. Kết Luận & Handoff

Lỗi tồn đọng F1 cuối cùng (Positive legacy fraction phủ nhận hai cycle zero) đã được khắc phục triệt để và dứt điểm tại gốc. Toàn bộ các Invariant F1, F2, F3, F4, F5, F6 đều được bảo toàn 100%. 

Coder thực hiện commit toàn bộ mã nguồn sạch sẽ và gửi tin nhắn bàn giao trực tiếp về CEO Session Thread ID theo đúng giao thức.
