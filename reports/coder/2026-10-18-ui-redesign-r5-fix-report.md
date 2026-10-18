# Báo Cáo Coder R5: Khắc Phục Triệt Để Lỗi Tồn Đọng F1 (Weekly Zero), Bảo Toàn Tuyệt Đối F2, F3, F4, F5, F6

**Tệp tin:** `reports/coder/2026-10-18-ui-redesign-r5-fix-report.md`  
**Ngày:** 2026-10-18  
**Tác giả:** Coder Specialist Lane  
**Người nhận:** CEO Agent (`01a14378-4a08-77c1-88d5-666ad8c3592d`), Supervisor Lane  
**Authority:** Chỉ thị CEO; Báo cáo Supervisor R4 `reports/supervisor/2026-10-18-ui-redesign-r4-supervisor-report.md` (commit `c316687`); Touch-Map Addendum `reports/architect/2026-10-18-touchmap-addendum-gpt-removal.md`; Đặc tả kiến trúc `reports/architect/2026-10-17-ui-redesign-spec-from-owner-drawing.md`; Bản vẽ thiết kế Owner `reports/problem/images/9180d29a.jpg`.  
**Verdict:** `[VERDICT: READY_FOR_SUPERVISOR]`

---

## 1. Mục Tiêu & Phạm Vi Công Việc

Khắc phục lỗi tồn đọng duy nhất của toàn bộ hệ thống được chỉ ra trong Báo cáo Supervisor R4:
1. **F1 [P2] Khắc phục triệt để mất dữ liệu chu kỳ tuần (Weekly Zero) khi cả 2 cycle đều bằng 0 tường minh:**
   - **Vấn đề gốc tại R4:** Nhánh code `if bucket.RemainingFraction == 0 && bucket.FiveHourFraction == 0 && bucket.WeeklyFraction == 0 && bucket.FiveHourReset.IsZero() && bucket.WeeklyReset.IsZero() && len(rawBuckets) == 0 { return 0.0, -1.0 }` tại `internal/ui/quota_badge.go:79-81` đã cố tình ép `WeeklyFraction` thành `-1.0` (biến thành `--%` trên card giao diện) trong trường hợp cả 2 chu kỳ đều có giá trị zero tường minh và không có raw bucket hoặc reset time.
   - **Khắc phục tại R5:** 
     - Loại bỏ hoàn toàn khối code suy diễn sai lệch tại `internal/ui/quota_badge.go:79-81`.
     - Giữ nguyên vẹn mọi cycle fraction tường minh `>= 0`, kể cả khi cả hai đều bằng `0.0` và không có reset/raw bucket. Khi `bucket.FiveHourFraction == 0.0` và `bucket.WeeklyFraction == 0.0` (hoặc `WeeklyFraction >= 0`), trả về chính xác tuple `(0.0, 0.0)`.
     - Sự vắng mặt của cycle đã được biểu diễn chuẩn hóa bằng sentinel `-1.0` từ tầng dữ liệu (`telemetry.ModelQuotaBucket` và unmarshaler); tầng UI tôn trọng contract dữ liệu chuẩn hóa, không phủ nhận giá trị zero của Weekly.
     - Bổ sung regression tests chặt chẽ trong cả `internal/telemetry/quota_test.go` và `internal/ui/ui_test.go` cho cả Claude và Gemini với trường hợp cả 2 cycle đều bằng 0 không có raw/reset, đồng thời bảo toàn hành vi legacy single-bucket / missing fraction (`-1.0` -> `"--%"`).
2. **Bảo toàn tuyệt đối các Invariant đã đạt:**
   - **F2 (Zero GPT):** Duy trì 0 field `GPT` trong Go struct `AccountQuotaData` và 0 khóa `"gpt"` trong JSON serialization.
   - **F3 (Single card refresh):** Giữ nguyên cơ chế refresh single card in-place bảo toàn 5 sibling card, focus và disable state.
   - **F4 (Retired helpers cleanup):** Bảo toàn trạng thái đã dọn sạch 5 helper retired; staticcheck 0 diagnostic.
   - **F5 (Compact card height):** Giữ nguyên chiều cao thẻ compact ~82px (3 dòng: topRow, row5h, rowWeek).
   - **F6 (Monogram AS theme):** Giữ nguyên monogram [AS] đổi màu theo theme (Sáng #2563eb, Ấm #d97706, Tối #89b4fa).

---

## 2. Invariant Family Map

| Thành phần | SSOT / Quy định kỹ thuật | Bề mặt mã nguồn (Touch-Map) | Hành vi chuẩn hóa |
|---|---|---|---|
| **F1: Explicit Zero Cycles Preservation** | Spec Owner, Báo cáo Supervisor R4 | `internal/ui/quota_badge.go:70-90` | `extractFamilyCycles` tôn trọng `FiveHourFraction >= 0` và `WeeklyFraction >= 0`. Khi cả hai bằng `0.0`, trả về đúng `(0.0, 0.0)`. Không tự ép weekly thành unknown `-1.0`. |
| **F1: Data-Layer Missing Sentinel** | Spec Owner, Báo cáo Supervisor R3 & R4 | `internal/telemetry/client.go:240, 430, 474, 700`, `internal/telemetry/cache.go` | Mọi field cycle vắng mặt đều được khởi tạo sentinel `-1.0` từ tầng parser/unmarshaler. Tầng UI không cần phỏng đoán sự vắng mặt. |
| **F1: UI Truthful Rendering** | Spec Owner, Báo cáo Supervisor R4 | `internal/ui/quota_badge.go:194`, `internal/ui/widgets.go:100` | Render `0%` cho cycle có fraction `0.0`. Render `"--%"` chỉ khi fraction `< 0` (sentinel `-1.0`). |
| **F2: Zero GPT Contract** | ADR-20261018-GPT-REMOVAL, Addendum Architect | `internal/telemetry/client.go` | 0 field `GPT` trong struct `AccountQuotaData`, 0 khóa `"gpt"` trong JSON serialization. |
| **F3: Single Card Refresh** | Spec Owner | `internal/ui/app.go:407` | Refresh in-place chỉ thay thế `Objects[targetIdx]`, bảo toàn 5/5 sibling cards và trạng thái giao diện. |
| **F4: Retired Helpers Cleanup** | Báo cáo Supervisor R3 & R4 | `internal/ui/quota_badge.go`, `internal/ui/widgets.go`, `internal/ui/app.go` | 5 helper retired đã bị loại bỏ hoàn toàn, test gọi trực tiếp production API. |
| **F5: Compact Layout** | Bản vẽ Owner, Spec Architect | `internal/ui/widgets.go` | Thẻ compact 3 dòng (~82px). |
| **F6: Monogram AS Theme** | Spec Architect | `internal/ui/window.go`, `internal/ui/app.go` | Background monogram [AS] đổi màu động theo palette: Sáng #2563eb, Ấm #d97706, Tối #89b4fa. |

---

## 3. Chi Tiết Thay Đổi Mã Nguồn Trong Touch-Map

### 3.1 `internal/ui/quota_badge.go`
- **Xóa bỏ nhánh suy diễn sai:**
  Loại bỏ khối code:
  ```go
  // Special case: if Go struct literal in tests initialized RemainingFraction == 0 and left cycle fractions as 0 without any raw buckets
  if bucket.RemainingFraction == 0 && bucket.FiveHourFraction == 0 && bucket.WeeklyFraction == 0 && bucket.FiveHourReset.IsZero() && bucket.WeeklyReset.IsZero() && len(rawBuckets) == 0 {
      return 0.0, -1.0
  }
  ```
- **Hành vi sau chỉnh sửa:**
  - `has5h := bucket.FiveHourFraction >= 0` -> `true` khi `FiveHourFraction == 0.0`.
  - `hasWeek := bucket.WeeklyFraction >= 0` -> `true` khi `WeeklyFraction == 0.0`.
  - Nhánh test struct literal `bucket.RemainingFraction > 0` không kích hoạt (do `0.0 > 0` là `false`).
  - Gán trực tiếp `h5 = bucket.FiveHourFraction` (`0.0`) và `week = bucket.WeeklyFraction` (`0.0`).
  - Kết quả trả về: `(0.0, 0.0)` đầy đủ, trung thực, phản ánh chính xác dữ liệu từ server/cache.

### 3.2 `internal/telemetry/quota_test.go`
- Bổ sung unit test `TestF1_BothCyclesZero_ClaudeAndGemini_NoRaw_NoReset`:
  - Kiểm tra `ParseQuotaSummary` với summary Connect-RPC có bucket 5h và weekly đều là `remainingFraction: 0.0` cho cả Claude và Gemini, không có reset time, không có raw buckets -> trích xuất chính xác `(0.0, 0.0)` cho cả 2 family.
  - Kiểm tra vòng lặp ghi đĩa và đọc lại qua `QuotaCache` (`SaveToDisk` -> `LoadFromDisk`) bảo toàn tuyệt đối cả 4 cycle fraction `0.0` với `RawBuckets` rỗng.

### 3.3 `internal/ui/ui_test.go`
- Bổ sung unit test `TestF1_ExtractDualCycleQuota_BothCyclesZero_NoRaw_NoReset`:
  - Case 1: Cả Claude và Gemini đều có cả 2 cycle bằng `0.0` không có raw buckets và reset time -> `ExtractDualCycleQuota` trả về `Claude5h=0.0, ClaudeWeek=0.0, Gemini5h=0.0, GeminiWeek=0.0, HasData=true`. Tạo `NewAccountCardWithQuota` và duyệt toàn bộ text trên card, đảm bảo hiển thị đúng 4 nhãn `"0%"`, tuyệt đối không có nhãn `"--%"`.
  - Case 2: Claude có 2 cycle zero `(0.0, 0.0)`, Gemini missing `(-1.0, -1.0)` -> Claude giữ `(0.0, 0.0)`, Gemini giữ `(-1.0, -1.0)`.
  - Case 3: Gemini có 2 cycle zero `(0.0, 0.0)`, Claude missing `(-1.0, -1.0)` -> Gemini giữ `(0.0, 0.0)`, Claude giữ `(-1.0, -1.0)`.

### 3.4 `reports/coder/evidence/2026-10-18-ui-redesign-f1-f6/coder_probe_r5.go`
- Tạo probe kiểm chứng độc lập R5 gồm 84 assertions toàn diện kiểm tra tất cả các ca F1 (F1.1 đến F1.12), F2, F4, F5, F6.

---

## 4. Bằng Chứng Kiểm Định Thực Nghiệm (Verification Evidence)

### 4.1 Coder Independent Acceptance Probe R5 (`coder_probe_r5.go`)
- **Lệnh thực thi:** `go run reports/coder/evidence/2026-10-18-ui-redesign-f1-f6/coder_probe_r5.go`
- **Kết quả:** **84/84 assertions PASSED (Exit Code: 0)**
  - F1.1: REST Amount camelCase without fraction -> `-1`, `HasData=false`: **PASS**
  - F1.2: REST Amount snake_case without fraction -> `-1`, `HasData=false`: **PASS**
  - F1.3: Missing fraction & amount -> `(-1,-1,-1,-1)`, `HasData=false`: **PASS**
  - F1.4: Claude fraction 0 -> Claude 5h 0%, week `-1`, Gemini `-1`: **PASS**
  - F1.5: Connect-RPC summary weekly 0% -> mapped to week row, not 5h: **PASS**
  - F1.6: Disk cache save & reload `[-1, 0, -1, -1]` -> exact preservation: **PASS**
  - F1.7: Legacy cache missing Gemini -> Claude 20% preserved, Gemini `-1`: **PASS**
  - F1.8: Legacy cache Claude object without fraction -> `-1`, `HasData=false`: **PASS**
  - **F1.9 (Mới R5):** Claude both-cycles zero without raw/reset -> `[0, 0, -1, -1]`, card text Claude 5h 0%, week 0%, không có Claude `--%`: **PASS**
  - **F1.10 (Mới R5):** Gemini both-cycles zero without raw/reset -> `[-1, -1, 0, 0]`, card text Gemini 5h 0%, week 0%, không có Gemini `--%`: **PASS**
  - **F1.11 (Mới R5):** Both Claude & Gemini both-cycles zero without raw/reset -> `[0, 0, 0, 0]`, card text hiển thị đủ 4x `0%`, 0x `--%`: **PASS**
  - **F1.12 (Mới R5):** Cache round-trip both-cycles zero -> `[0, 0, 0, 0]` bảo toàn nguyên vẹn sau khi nạp lại từ đĩa, 0 raw buckets: **PASS**
  - F2: Zero GPT struct field & zero GPT in JSON serialization: **PASS**
  - F4: 4 retired UI fields absent from `ui.App` struct: **PASS**
  - F5: Card layout compact có đúng 3 rows (topRow, row5h, rowWeek): **PASS**
  - F6: Theme palette colors match Sáng #2563eb, Ấm #d97706, Tối #89b4fa: **PASS**

### 4.2 Supervisor R4 Probe Verification
- **Lệnh thực thi:** `go run reports/supervisor/evidence/2026-10-18-ui-redesign-r4/zero_cycle_card_probe.go` (đã đối soát và hoàn trả file PNG nguyên vẹn)
- **Kết quả:**
  - `claude-both-zero-no-raw`: `badge_cycles: [0, 0, -1, -1]`, `matches: true`
  - `claude-both-zero-with-raw`: `badge_cycles: [0, 0, -1, -1]`, `matches: true`
  - `claude-weekly-zero-no-raw`: `badge_cycles: [-1, 0, -1, -1]`, `matches: true`
  - `gemini-both-zero-no-raw`: `badge_cycles: [-1, -1, 0, 0]`, `matches: true`
  - **Failures: 0 (Trước đây tại R4: 2 failures)**

### 4.3 Full Workspace Test Suite
- **Lệnh thực thi:** `go test -v -count=1 ./...`
- **Kết quả:** **10/10 packages PASSED (163 tests, 0 FAIL, 0 SKIP, Exit Code: 0)**
  - `cmd/switcher`: **PASS** (3.167s)
  - `internal/connector`: **PASS** (3.925s)
  - `internal/manager`: **PASS** (11.985s)
  - `internal/model`: **PASS** (1.251s)
  - `internal/reloader`: **PASS** (1.758s)
  - `internal/scanner`: **PASS** (1.378s)
  - `internal/store`: **PASS** (1.298s)
  - `internal/telemetry`: **PASS** (15.176s)
  - `internal/ui`: **PASS** (12.245s)
  - `internal/wincred`: **PASS** (1.185s)

### 4.4 Staticcheck
- **Lệnh thực thi:** `staticcheck -tags gui ./...`
- **Kết quả:** **Exit Code: 0, 0 diagnostic**

### 4.5 Native GUI Build
- **Lệnh thực thi:** `go build -tags gui -o .tmp/test_build.exe ./cmd/switcher`
- **Kết quả:** **Exit Code: 0 (Biên dịch thành công)**

---

## 5. Danh Sách Tệp Tin Chỉnh Sửa Trong Touch-Map

1. `internal/ui/quota_badge.go` (Loại bỏ override sai làm mất WeeklyFraction zero khi cả 2 cycle đều bằng 0 không có raw/reset)
2. `internal/telemetry/quota_test.go` (Thêm `TestF1_BothCyclesZero_ClaudeAndGemini_NoRaw_NoReset`)
3. `internal/ui/ui_test.go` (Thêm `TestF1_ExtractDualCycleQuota_BothCyclesZero_NoRaw_NoReset`)
4. `reports/coder/evidence/2026-10-18-ui-redesign-f1-f6/coder_probe_r5.go` (Probe kiểm chứng độc lập R5 gồm 84 assertions)
5. `reports/coder/2026-10-18-ui-redesign-r5-fix-report.md` (Báo cáo chính thức R5 này)

---

## 6. Kết Luận & Handoff

Lỗi tồn đọng duy nhất của toàn bộ hệ thống (F1: Weekly zero bị chuyển thành -1 khi cả hai cycle đều là 0 tường minh không có raw/reset) đã được khắc phục hoàn toàn tại gốc. Toàn bộ các Invariant F1, F2, F3, F4, F5, F6 đều được bảo toàn 100%. Không còn bất kỳ khiếm khuyết nào tồn đọng.

Coder tự thực hiện commit toàn bộ mã nguồn sạch sẽ và bàn giao kết quả cho CEO Agent để chuyển sang Supervisor nghiệm thu.
