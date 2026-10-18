# Báo Cáo Coder: Khắc Phục Triệt Để 3 Lỗi Chặn F1, F2, F4 & Bảo Toàn Invariants F3, F5, F6

**Tác giả:** Coder Specialist Lane  
**Thời điểm:** 2026-10-18, Asia/Bangkok (UTC+07)  
**Địa chỉ bàn giao (Return Address):** CEO Session Thread ID `01a14378-4a08-77c1-88d5-666ad8c3592d`  
**Thẩm quyền & Căn cứ:**  
- Báo cáo Nghiệm thu Supervisor Vòng 2 `reports/supervisor/2026-10-18-ui-redesign-r2-supervisor-report.md` (commit `9b4edb2`)  
- Quyết định Kiến trúc & Touch-Map Addendum `reports/architect/2026-10-18-touchmap-addendum-gpt-removal.md` (commit `7baf312`)  
- Bản vẽ Owner `reports/problem/images/9180d29a.jpg` & Spec `reports/architect/2026-10-17-ui-redesign-spec-from-owner-drawing.md`  
**Trạng thái bàn giao:** `[VERDICT: READY_FOR_SUPERVISOR]`

---

## 1. Invariant Family Map & Phạm Vi Giải Quyết

| Invariant / Lỗi chặn | Nguyên nhân gốc (Root Cause) | Giải pháp thực hiện (Surgical Fix) | Tệp tin chỉnh sửa |
|---|---|---|---|
| **F1 [P1] Quota Presence & Truthful Fallback** | 1. `ParseQuotaSummaryWithBuckets` khởi tạo fraction mặc định 1.0 khi thiếu `RemainingFraction` hoặc khi nhóm `groups` rỗng.<br>2. REST client fallback sang `AggregateBuckets` khi `groups` rỗng, tự gán giá trị 1.0 Pristine.<br>3. `ExtractDualCycleQuota` kiểm tra `> 0` thay vì `>= 0`, khiến quota 0% bị bỏ qua và rơi vào fallback.<br>4. Priority 3 của `ExtractDualCycleQuota` tự sao chép `RemainingFraction` sang cả `5h` và `week`. | 1. Khởi tạo fraction mặc định `-1.0` (chưa biết). Khi `groups` rỗng, trả về `-1.0` và `HasData=false`.<br>2. Nhận diện payload summary rỗng ngay tại REST client, trả về struct không có data mà không fallback sang `AggregateBuckets`.<br>3. Kiểm tra `FiveHourFraction >= 0` để nhận diện chính xác quota 0% (depleted).<br>4. Custom `UnmarshalJSON` trên `ModelQuotaBucket` mặc định `FiveHourFraction = -1`, `WeeklyFraction = -1` khi đọc cache cũ.<br>5. Priority 3 chỉ gán `Claude5h` / `Gemini5h` và giữ `ClaudeWeek = -1`, `GeminiWeek = -1` (không tự suy diễn chu kỳ tuần). | `internal/telemetry/client.go`<br>`internal/telemetry/quota_test.go`<br>`internal/ui/quota_badge.go` |
| **F2 [P2] Eradicate GPT from Struct & Contract** | Struct `AccountQuotaData` vẫn giữ trường `GPT ModelQuotaBucket json:"-"`; reflection vẫn tìm thấy trường; REST client và parsers vẫn gán `GPT: gpt`; `cmd/switcher/main.go` và tests vẫn truy cập `.GPT`. | 1. Xóa hoàn toàn trường `GPT` khỏi `AccountQuotaData` trong `internal/telemetry/client.go`.<br>2. Xóa các phép gán `GPT: gpt` trong REST client `RetrieveUserQuota`.<br>3. Cập nhật `cmd/switcher/main.go` (loại bỏ `q.GPT`), `cmd/switcher/main_test.go`.<br>4. Cập nhật `internal/telemetry/telemetry_test.go` (xóa assertions trên `data.GPT`).<br>5. Bổ sung kiểm tra reflection trong `quota_test.go` xác nhận trường `GPT` không còn tồn tại trong Go struct. | `internal/telemetry/client.go`<br>`internal/telemetry/quota_test.go`<br>`internal/telemetry/telemetry_test.go`<br>`cmd/switcher/main.go`<br>`cmd/switcher/main_test.go` |
| **F4 [P2] Dead UI Objects & Stubs Cleanup** | 1. 4 widget cũ (`headerBg`, `statusBarLabel`, `activeStatusLabel`, `activeQuotaLabel`) được khởi tạo trên `App` nhưng không gắn vào render tree (`window.Content()`), vẫn nhận lệnh cập nhật/recolor ngầm.<br>2. `NewQuotaCountdownLabel` là stub luôn trả `nil`; `FormatRemainingCountdown` chỉ còn caller trong test. | 1. Xóa bỏ hoàn toàn 4 trường trên khỏi struct `App` trong `internal/ui/app.go`.<br>2. Xóa các lệnh khởi tạo trong `internal/ui/window.go`.<br>3. Xóa các lệnh cập nhật `SetText`, `FillColor`, `Refresh()` đối với 4 widget trong `SetTheme`, callbacks và `updateUI`.<br>4. Xóa hàm stub `NewQuotaCountdownLabel` và hàm `FormatRemainingCountdown` khỏi `internal/ui/quota_badge.go`.<br>5. Dọn dẹp các assertions/calls cũ trong `internal/ui/ui_test.go` và `internal/ui/themes_test.go`. | `internal/ui/app.go`<br>`internal/ui/window.go`<br>`internal/ui/quota_badge.go`<br>`internal/ui/ui_test.go`<br>`internal/ui/themes_test.go` |
| **F3 (Bảo toàn)** | Refresh riêng từng thẻ in-place | Giữ nguyên cơ chế thay riêng `grid.Objects[targetIdx]`, 5 sibling card không bị thay đổi con trỏ, giữ trạng thái disabled/focus. | `internal/ui/app.go` |
| **F5 (Bảo toàn)** | Thẻ compact ~82px, layout VScroll | Giữ nguyên cấu trúc `VScroll(Padded(VBox(Grid)))`, thẻ giữ nguyên chiều cao tự nhiên 81.97px trên mọi kích thước. | `internal/ui/window.go`<br>`internal/ui/app.go` |
| **F6 (Bảo toàn)** | Monogram `[AS]` đổi màu theo theme | Giữ nguyên cập nhật `a.monogramBg.FillColor = palette.Primary` và Refresh khi chuyển theme (Sáng: `#2563eb`, Ấm: `#d97706`, Tối: `#89b4fa`). | `internal/ui/app.go`<br>`internal/ui/window.go` |

---

## 2. Bằng Chứng Kiểm Định Thực Nghiệm (Verification Evidence)

### 2.1 Full Test Suite 10/10 Packages Pass
Lệnh: `go test -v -count=1 ./...`  
Kết quả: **10/10 packages PASS**, **158 passed cases (138 top-level + 20 subtests), 0 FAIL, 0 SKIP**. Exit code 0.
- `cmd/switcher`: ok (4 test cases)
- `internal/connector`: ok (13 test cases)
- `internal/manager`: ok (18 test cases)
- `internal/model`: ok (1 test case)
- `internal/reloader`: ok (30 test cases)
- `internal/scanner`: ok (4 test cases)
- `internal/store`: ok (11 test cases)
- `internal/telemetry`: ok (23 test cases)
- `internal/ui`: ok (31 test cases)
- `internal/wincred`: ok (3 test cases)

### 2.2 Staticcheck Exit 0 (0 Diagnostics)
Lệnh: `staticcheck -tags gui ./...`  
Kết quả: **Exit code 0, 0 diagnostic**. Hoàn toàn sạch dead code và unused identifiers.

### 2.3 Native GUI Build Succeeded
Lệnh: `go build -tags gui -ldflags "-H=windowsgui" ./cmd/switcher`  
Kết quả: **Exit code 0**, biên dịch thành công nhị phân GUI Windows.

### 2.4 Coder Independent Acceptance Probe
Tệp probe: `reports/coder/evidence/2026-10-18-ui-redesign-f1-f6/coder_probe.go`  
Lệnh: `go run reports/coder/evidence/2026-10-18-ui-redesign-f1-f6/coder_probe.go`  
Kết quả:
```json
{
  "results": {
    "F1_cases": {
      "empty_connect_rpc": {
        "c5h": -1,
        "g5h": -1,
        "has_data": false,
        "pass": true
      },
      "empty_rest": {
        "has_data": false,
        "pass": true
      },
      "legacy_cache": {
        "c5h": 0.2,
        "cWeek": -1,
        "g5h": 0.8,
        "gWeek": -1,
        "has_data": true,
        "pass": true
      },
      "missing_fraction": {
        "has_data": false,
        "pass": true
      },
      "window_only_partial_zero": {
        "c5h": 0,
        "cWeek": -1,
        "g5h": -1,
        "has_data": true,
        "pass": true
      }
    },
    "F2_contract": {
      "GPT_field_present": false,
      "expected_field_present": false,
      "serialized_contains_gpt_key": false,
      "status": true
    },
    "F4_legacy_ui_fields": {
      "found_legacy_fields": null,
      "pass": true
    }
  },
  "status": "ALL_PROBES_PASS"
}
```

---

## 3. Tính Bất Biến Của Artifacts Supervisor
- Toàn bộ tệp lịch sử trong `reports/supervisor/evidence/*` được **bảo toàn nguyên vẹn 100%**, không có bất kỳ thao tác sửa đổi, xóa hoặc ghi đè nào.
- Mọi bằng chứng thực nghiệm của Coder được đặt cách ly tuyệt đối trong `reports/coder/evidence/2026-10-18-ui-redesign-f1-f6/`.

---

## 4. Kết Luận & Handoff
Mọi yêu cầu khắc phục lỗi F1, F2, F4 đã được giải quyết triệt để tại gốc rễ (Root Cause); các invariant F3, F5, F6 được bảo tồn hoàn hảo; toàn bộ hệ thống test suite, staticcheck và build đạt chuẩn production.
