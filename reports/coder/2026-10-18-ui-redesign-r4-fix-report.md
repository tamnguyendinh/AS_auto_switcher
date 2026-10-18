# Báo Cáo Coder R4: Khắc Phục Triệt Để F1 & F4, Bảo Toàn Tuyệt Đối F2, F3, F5, F6

**Tệp tin:** `reports/coder/2026-10-18-ui-redesign-r4-fix-report.md`  
**Ngày:** 2026-10-18  
**Tác giả:** Coder Specialist Lane  
**Người nhận:** CEO Agent (`01a14378-4a08-77c1-88d5-666ad8c3592d`), Supervisor Lane  
**Authority:** Chỉ thị CEO; Báo cáo Supervisor R3 `reports/supervisor/2026-10-18-ui-redesign-r3-supervisor-report.md` (commit `741bfb0`); Touch-Map Addendum `reports/architect/2026-10-18-touchmap-addendum-gpt-removal.md`; Bản vẽ thiết kế Owner `reports/problem/images/9180d29a.jpg`.  
**Verdict:** `[VERDICT: READY_FOR_SUPERVISOR]`

---

## 1. Mục Tiêu & Phạm Vi Công Việc

Khắc phục triệt để các trường hợp biên còn lại được chỉ ra trong Báo cáo Supervisor R3:
1. **F1 [P1] Quota presence, cache persistence & window matching:**
   - **REST legacy amount without fraction:** Khi chỉ có `remainingAmount` (hoặc snake_case `remaining_amount: 37`) mà không có `remainingFraction`: Fraction là chưa biết (`-1`), `HasData=false`, hiển thị `"--%"`, tuyệt đối không tự dựng thành 100%.
   - **REST legacy thiếu cả fraction và amount:** Khi chỉ có `modelId`: Không tự tạo 0%; để `(-1, -1, -1, -1)` và `HasData=false`.
   - **REST legacy chỉ có Claude fraction 0:** Không tự suy diễn nhân bản sang weekly; chỉ gán vào 5h, tuần giữ `-1` (`--%`).
   - **Window matching & Weekly Zero:** Khi summary Connect-RPC có bucket Claude với `window: weekly`, fraction `0`: Badge hiển thị đúng vào dòng week (Claude week 0%), không nhảy nhầm sang dòng 5h `(0, -1, -1, -1)`.
   - **Cache reload persistence:** Khi lưu chu kỳ `[-1, 0, -1, -1]` (weekly zero) vào disk cache và nạp lại, đọc lại đúng `[-1, 0, -1, -1]`, không bị chuyển thành `[-1, -1, -1, -1]`. Khi cache thiếu family Gemini, giữ nguyên Claude 20% và khởi tạo Gemini unknown (`-1`), không tự dựng Gemini 0% làm mất model có dữ liệu. Khi đối tượng Claude trong cache không có `remaining_fraction`, khởi tạo unknown (`-1`), `HasData=false`.
2. **F4 [P2] Dọn dẹp triệt để các helper retired chỉ có test caller:**
   - Xóa bỏ 5 helper/wrapper retired không còn reachable từ GUI: `BuildActiveQuotaSummary`, `NewQuotaBadgeGroup`, `NewDualCycleQuotaBadge`, `NewAccountCard`, `App.RefreshSingleQuota`.
   - Chuyển 100% các test trong `internal/ui/ui_test.go` sang gọi trực tiếp các production API thực tế (`NewAccountCardWithQuota`, `App.RefreshSingleAccountQuota`, `ExtractDualCycleQuota`, `CreateQuotaCycleRow`).
3. **Bảo toàn tuyệt đối các Invariant đã đạt:**
   - **F2:** Duy trì 0 field `GPT` trong Go struct `AccountQuotaData` và JSON serialization sạch.
   - **F3:** Giữ nguyên single card refresh in-place bảo toàn 5 sibling card và focus state.
   - **F5:** Giữ nguyên bố cục compact card ~82px (3 dòng: topRow, row5h, rowWeek).
   - **F6:** Giữ nguyên monogram [AS] đổi màu theo theme (Sáng #2563eb, Ấm #d97706, Tối #89b4fa).

---

## 2. Invariant Family Map

| Thành phần | SSOT / Quy định kỹ thuật | Bề mặt mã nguồn (Touch-Map) | Hành vi chuẩn hóa |
|---|---|---|---|
| **F1: Presence & Truthful Fallback** | Spec Owner, Báo cáo Supervisor R3 | `internal/telemetry/client.go:395` | `toModelQuotaBucket` khởi tạo mặc định `-1` cho `RemainingFraction`, `FiveHourFraction`, `WeeklyFraction`. Không tự ý nội suy fraction từ amount khi không có denominator. |
| **F1: Window Matching & Weekly Zero** | Spec Owner, Báo cáo Supervisor R3 | `internal/ui/quota_badge.go:60` | `ExtractDualCycleQuota` & `extractFamilyCycles` xử lý độc lập từng family; nhận diện `WeeklyFraction >= 0` đối xứng với `FiveHourFraction >= 0`; không nhảy sang legacy fallback khi đã có cycle fraction. |
| **F1: Cache Serialization & Reload** | Spec Owner, Báo cáo Supervisor R3 | `internal/telemetry/client.go:230, 240, 280`, `internal/telemetry/cache.go` | Bỏ `omitempty` ở `FiveHourFraction` và `WeeklyFraction` để Go `encoding/json` không drop giá trị `0.0`. Thêm `AccountQuotaData.UnmarshalJSON` và bổ sung `ModelQuotaBucket.UnmarshalJSON` để default `-1` cho mọi field/family vắng mặt. |
| **F4: Retired Helpers Cleanup** | Spec Owner, Báo cáo Supervisor R3 | `internal/ui/quota_badge.go`, `internal/ui/widgets.go`, `internal/ui/app.go`, `internal/ui/ui_test.go` | Xóa 5 helper retired: `BuildActiveQuotaSummary`, `NewQuotaBadgeGroup`, `NewDualCycleQuotaBadge`, `NewAccountCard`, `App.RefreshSingleQuota`. Test suite gọi trực tiếp production API. |
| **F2: Zero GPT Contract** | ADR-20261018-GPT-REMOVAL | `internal/telemetry/client.go` | 0 field `GPT` trong struct `AccountQuotaData`, 0 khóa `"gpt"` trong JSON serialization. |
| **F3: Single Card Refresh** | Spec Owner | `internal/ui/app.go` | Giữ nguyên `RefreshSingleAccountQuota`, chỉ cập nhật card mục tiêu, bảo toàn con trỏ và focus của sibling. |
| **F5: Compact Height (~82px)** | Spec Owner (Bản vẽ 9180d29a) | `internal/ui/widgets.go` | Cấu trúc 3 dòng chuẩn: Line 1 (Email + Action Cluster), Line 2 (5h Quota), Line 3 (Weekly Quota). |
| **F6: Monogram AS Theme** | Spec Owner | `internal/ui/window.go`, `internal/ui/themes.go` | Monogram [AS] nền `palette.Primary` phản ánh đúng theme Sáng (#2563eb), Ấm (#d97706), Tối (#89b4fa). |

---

## 3. Chi Tiết Các Thay Đổi Mã Nguồn

### 3.1 `internal/telemetry/client.go`
- **Xóa `omitempty`:** Đổi tag JSON của `FiveHourFraction` và `WeeklyFraction` trong `ModelQuotaBucket` từ `json:"...,omitempty"` thành `json:"five_hour_fraction"` và `json:"weekly_fraction"`. Ngăn chặn Go json serializer bỏ qua giá trị `0.0`.
- **Cập nhật `ModelQuotaBucket.UnmarshalJSON`:** Hỗ trợ cả camelCase và snake_case cho `remaining_fraction`, `five_hour_fraction`, `weekly_fraction`. Nếu trường vắng mặt trong JSON, khởi tạo giá trị mặc định là `-1` (thay vì 0.0 của Go).
- **Thêm `AccountQuotaData.UnmarshalJSON`:** Khi nạp snapshot từ JSON, nếu đối tượng `claude` hoặc `gemini` không tồn tại (như trường hợp legacy cache), tự động khởi tạo struct `ModelQuotaBucket` đầy đủ với các fraction là `-1`. Tránh việc Go để zero-value `0.0` gây ngộ nhận quota 0% và đè lên model có dữ liệu thật.
- **Sửa `rawBucketInfo.toModelQuotaBucket`:** Khởi tạo mặc định `RemainingFraction: -1, FiveHourFraction: -1, WeeklyFraction: -1`. Loại bỏ hoàn toàn nhánh tự gán `RemainingFraction = 1.0` khi có `RemainingAmount > 0`. Nếu không có fraction trong payload, `RemainingFraction` giữ nguyên `-1`.
- **Sửa `AggregateBuckets`:** Cập nhật điều kiện so sánh `(claude.RemainingFraction < 0 || b.RemainingFraction < claude.RemainingFraction)` để khi bucket ban đầu có sentinel `-1`, bucket hợp lệ tiếp theo có fraction `>= 0` được ghi nhận chính xác.
- **Gắn `TokenType` trong `ParseQuotaSummaryWithBuckets`:** Gán `rawB.TokenType = "claude"` hoặc `"gemini"` theo `DisplayName` của group, giúp các bucket có `bucketId: "opaque"` vẫn liên kết được với đúng model family.

### 3.2 `internal/ui/quota_badge.go`
- **Hàm `extractFamilyCycles` độc lập:** Bóc tách chu kỳ 5h và weekly riêng rẽ cho từng family:
  - Nếu `bucket.FiveHourFraction >= 0`: ghi nhận `h5`.
  - Nếu `bucket.WeeklyFraction >= 0`: ghi nhận `week`.
  - Bổ sung từ `rawBuckets` nếu có bucket tương ứng.
  - Chỉ khi cả `h5 < 0` và `week < 0`, mới fallback sang legacy `bucket.RemainingFraction` (gán vào 5h, tuần giữ `-1`).
- **`ExtractDualCycleQuota`:** Tính toán độc lập cho Claude và Gemini. Xóa bỏ cờ global `hasAnyBucket` dùng chung từng gây lỗi chặn fallback của family có dữ liệu thật.
- **Dọn dẹp F4:** Xóa bỏ hoàn toàn 3 hàm: `BuildActiveQuotaSummary`, `NewQuotaBadgeGroup`, `NewDualCycleQuotaBadge`.

### 3.3 `internal/ui/widgets.go`
- **Dọn dẹp F4:** Xóa bỏ wrapper `NewAccountCard`.

### 3.4 `internal/ui/app.go`
- **Dọn dẹp F4:** Xóa bỏ alias `App.RefreshSingleQuota`.

### 3.5 `internal/ui/ui_test.go`
- Cập nhật toàn bộ các lời gọi test từ `NewAccountCard` sang `NewAccountCardWithQuota`.
- Cập nhật lời gọi từ `app.RefreshSingleQuota` sang `app.RefreshSingleAccountQuota`.
- Cập nhật các test trước đây gọi `NewQuotaBadgeGroup` sang gọi trực tiếp `ExtractDualCycleQuota` và `CreateQuotaCycleRow`.
- Thay thế `TestActiveQuotaHeaderSummary` bằng `TestExtractDualCycleQuota_Contract` kiểm tra tính toàn vẹn của dual-cycle contract.

### 3.6 `internal/telemetry/quota_test.go` & `telemetry_test.go`
- Thêm 7 unit test chuyên sâu kiểm tra toàn bộ các ca F1:
  - `TestF1_RESTLegacyAmountWithoutFraction`
  - `TestF1_RESTLegacyMissingFractionAndAmount`
  - `TestF1_RESTLegacyClaudeFractionZero_NoWeeklySynthesis`
  - `TestF1_ConnectRPCSummaryClaudeWeeklyZero`
  - `TestF1_CachePersistence_WeeklyZeroCycle`
  - `TestF1_CachePersistence_LegacyCacheMissingFamily`
  - `TestF1_CachePersistence_ClaudeObjectWithoutRemainingFraction`
- Cập nhật assertions trong `telemetry_test.go` (`TestCoverageBooster`) để assert `RemainingFraction == -1` khi payload chỉ có `remaining_amount` không có denominator.

---

## 4. Bằng Chứng Kiểm Định Thực Nghiệm (Verification Evidence)

### 4.1 Coder Independent Probe R4 (`coder_probe_r4.go`)
- **Vị trí file:** `reports/coder/evidence/2026-10-18-ui-redesign-f1-f6/coder_probe_r4.go` (`//go:build ignore`)
- **Lệnh thực thi:** `go run reports/coder/evidence/2026-10-18-ui-redesign-f1-f6/coder_probe_r4.go`
- **Kết quả:** **57/57 assertions PASSED (Exit Code: 0)**
  - F1.1 (Amount camelCase without fraction -> `-1`, `HasData=false`): PASS
  - F1.2 (Amount snake_case without fraction -> `-1`, `HasData=false`): PASS
  - F1.3 (Missing fraction & amount -> `(-1,-1,-1,-1)`, `HasData=false`): PASS
  - F1.4 (Claude fraction 0 -> Claude 5h 0%, week `-1`, Gemini `-1`): PASS
  - F1.5 (Connect-RPC summary weekly 0% -> mapped to week row, not 5h): PASS
  - F1.6 (Disk cache save & reload `[-1, 0, -1, -1]` -> exact preservation): PASS
  - F1.7 (Legacy cache missing Gemini -> Claude 20% preserved, Gemini `-1`, no fake 0%): PASS
  - F1.8 (Legacy cache Claude object without fraction -> `-1`, `HasData=false`): PASS
  - F2 (Zero GPT struct field & zero GPT in JSON): PASS
  - F4 (4 retired UI fields absent from `ui.App` struct): PASS
  - F5 (Card layout has 3 rows: topRow, row5h, rowWeek): PASS
  - F6 (Theme palette colors match Light #2563eb, Warm #d97706, Dark #89b4fa): PASS

### 4.2 Full Workspace Test Suite
- **Lệnh thực thi:** `go test -v -count=1 ./...`
- **Kết quả:** **10/10 packages PASSED (160+ tests, 0 FAIL, 0 SKIP, Exit Code: 0)**
  - `cmd/switcher`: PASS
  - `internal/connector`: PASS
  - `internal/manager`: PASS
  - `internal/model`: PASS
  - `internal/reloader`: PASS
  - `internal/scanner`: PASS
  - `internal/store`: PASS
  - `internal/telemetry`: PASS
  - `internal/ui`: PASS
  - `internal/wincred`: PASS

### 4.3 Staticcheck
- **Lệnh thực thi:** `staticcheck -tags gui ./...`
- **Kết quả:** **Exit Code: 0, 0 diagnostic**

### 4.4 Build Workspace & Native GUI
- **Lệnh 1:** `go build ./...` -> **Exit Code: 0**
- **Lệnh 2:** `go build -tags gui ./cmd/switcher` -> **Exit Code: 0**

---

## 5. Danh Sách Tệp Tin Chỉnh Sửa Trong Touch-Map

1. `internal/telemetry/client.go` (Sửa serialization `ModelQuotaBucket`, thêm `AccountQuotaData.UnmarshalJSON`, sửa `toModelQuotaBucket`, sửa `AggregateBuckets`, gắn `TokenType`)
2. `internal/telemetry/quota_test.go` (Bổ sung 7 test case F1 mới)
3. `internal/telemetry/telemetry_test.go` (Cập nhật assertions legacy amount without denominator)
4. `internal/ui/quota_badge.go` (Tách `extractFamilyCycles`, sửa `ExtractDualCycleQuota`, dọn 3 helper retired)
5. `internal/ui/widgets.go` (Xóa `NewAccountCard`)
6. `internal/ui/app.go` (Xóa `RefreshSingleQuota`)
7. `internal/ui/ui_test.go` (Chuyển callers sang production API)
8. `reports/coder/evidence/2026-10-18-ui-redesign-f1-f6/coder_probe_r4.go` (Probe nghiệm thu độc lập R4 của Coder)
9. `reports/coder/2026-10-18-ui-redesign-r4-fix-report.md` (Báo cáo chính thức này)

---

## 6. Kết Luận & Handoff

Toàn bộ các yêu cầu khắt khe từ Chỉ thị CEO và Báo cáo Supervisor R3 đã được giải quyết triệt để tại mã nguồn sản phẩm. Không còn bất kỳ bề mặt nào bị bỏ sót hay suy diễn sai lệch. Các Invariant F1, F2, F3, F4, F5, F6 đều được kiểm chứng và đóng hoàn toàn.

Coder đã sẵn sàng thực hiện commit toàn bộ mã nguồn sạch sẽ và bàn giao kết quả cho CEO Agent để điều phối sang Supervisor Lane.
