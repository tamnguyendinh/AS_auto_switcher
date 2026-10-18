# Báo cáo Coder: Khắc phục triệt để 6 lỗi chặn nghiệm thu F1-F6 (UI Redesign)

**Mã báo cáo:** `rp_coder_20261018_ui_redesign_f1_f6_fix`  
**Ngày thực hiện:** 2026-10-18, Asia/Bangkok (UTC+07)  
**Vai trò:** Senior Coder  
**Target Commit cơ sở:** `fe8ffbb` (Báo cáo Supervisor Reject)  
**Trạng thái bàn giao:** **READY_FOR_SUPERVISOR**  

---

## 1. Invariant Family Map

| Finding | Invariant Family | Root Cause | Giải pháp thực hiện | File sửa |
|---|---|---|---|---|
| **F1 [P1]** | Quota Data Truthfulness | `ParseQuotaSummary` gộp bucket theo min fraction; `RetrieveUserQuota` chỉ parse top-level `buckets` cũ thay vì `groups/bucketId/window`. Dữ liệu 5h và week bị mất trước khi tới UI. | Nâng cấp `ModelQuotaBucket` lưu `FiveHourFraction`, `WeeklyFraction`, `FiveHourReset`, `WeeklyReset`. Cập nhật `DefaultGoogleQuotaURL` sang endpoint `retrieveUserQuotaSummary`. Viết `ParseQuotaSummaryWithBuckets` bóc tách độc lập 5h và week cho cả Connect-RPC và REST. Cập nhật `ExtractDualCycleQuota` bảo toàn từng chu kỳ và hiển thị `--%` khi thiếu bucket, không giả 100%. | `internal/telemetry/client.go`<br>`internal/telemetry/pool.go`<br>`internal/ui/quota_badge.go` |
| **F2 [P2]** | Schema Data Isolation | `AccountQuotaData` chứa trường `GPT` có tag serialize `json:"gpt"`, làm lộ trường `gpt` trong JSON thẻ và payload. | Gỡ bỏ hoàn toàn việc gán `GPT` trong parser Language Server (`pool.go`) và Cloud client (`client.go`). Đặt tag `json:"-"` cho `GPT` trong `AccountQuotaData` để triệt tiêu hoàn toàn khóa `gpt` khi marshal/lưu trữ thẻ. Dọn dẹp sạch `GPT:` trong các struct literal test UI. | `internal/telemetry/client.go`<br>`internal/telemetry/pool.go`<br>`internal/ui/ui_test.go` |
| **F3 [P2]** | Single-Card Refresh Isolation | Khi bấm nút refresh [🔄] trên thẻ, callback gọi `updateUI(overview)` xóa toàn bộ `accountListContainer.Objects` và tạo mới lại tất cả card. | Sửa `RefreshSingleAccountQuota(email)` để chỉ thay thế in-place đúng card của tài khoản mục tiêu tại `a.accountListContainer.Objects[targetIdx] = newCard`. Giữ nguyên 100% đối tượng thẻ sibling và trạng thái tương tác của chúng (5/5 thẻ sibling giữ nguyên con trỏ đối tượng). | `internal/ui/app.go` |
| **F4 [P2]** | 0 Dead Code / 0 Staticcheck Warning | Staticcheck cảnh báo `hasGUI` unused trong `app_gui.go`, `err` unused trong `api_client.go`, `saveBtn` và `headerCard` unused trong `app.go`. Các đối tượng UI cũ (`statusBg`, `legendBg`) không gắn vào cây render vẫn được tạo và cập nhật. | Xóa bỏ hằng số `hasGUI` trong `app_gui.go`. Xử lý kiểm tra lỗi `err` sau `io.ReadAll` trong `api_client.go`. Xóa các trường struct không dùng `saveBtn`, `headerCard`, `statusBg`, `legendBg` khỏi `App`. Loại bỏ code khởi tạo và recolor đối tượng UI cũ trong `window.go` và `app.go`. | `cmd/switcher/app_gui.go`<br>`internal/connector/api_client.go`<br>`internal/ui/app.go`<br>`internal/ui/window.go` |
| **F5 [P2]** | Compact Card Sizing | Lưới 3 cột đặt trực tiếp trong VScroll bị giãn chiều cao theo viewport 900px, đẩy chiều cao thẻ từ ~82px lên 378px. | Bọc `a.accountListContainer` trong `container.NewVBox` và căn chỉnh top-aligned trong `NewVScroll(container.NewPadded(...))`. Trong `updateUI`, thiết lập chiều cao lưới theo `MinSize().Height` tự nhiên (~82px mỗi hàng), giúp các thẻ giữ đúng kích thước compact, không bị kéo dãn tạo khoảng trống thừa mênh mông. | `internal/ui/window.go`<br>`internal/ui/app.go` |
| **F6 [P2]** | Dynamic Theme Propagation | Monogram `[AS]` khởi tạo với màu Dark ban đầu (`palette.Primary`) nhưng không được lưu tham chiếu trên `App` để cập nhật khi đổi theme. | Thêm trường `monogramBg *canvas.Rectangle` vào `App`. Gán tham chiếu trong `setupLayout()`. Trong `SetTheme(themeName)`, cập nhật trực tiếp `a.monogramBg.FillColor = palette.Primary` và `Refresh()`, đồng bộ chuẩn xác cho cả 3 theme (Dark, Light, Warm). | `internal/ui/app.go`<br>`internal/ui/window.go` |

---

## 2. Touch-Map Compliance

Tất cả các file sửa đổi đều nằm 100% trong Touch-Map đã chỉ định bởi CEO:
- `cmd/switcher/app_gui.go` (F4: xóa `hasGUI`)
- `internal/connector/api_client.go` (F4: xử lý `err` sau `io.ReadAll`)
- `internal/ui/app.go` (F3, F4, F5, F6: isolated refresh, xóa dead fields, compact resize, monogram recolor)
- `internal/ui/window.go` (F5, F6: top-aligned scroll layout, monogram reference)
- `internal/ui/widgets.go` (F1, F2: dual cycle rendering, verified free of GPT)
- `internal/ui/quota_badge.go` (F1: độc lập dual-cycle 5h/week, format missing fraction)
- `internal/ui/ui_test.go` (F1-F6: bổ sung 4 bộ test regression chuyên biệt, dọn dẹp GPT literals)
- `internal/telemetry/client.go` (F1, F2: summary endpoint, `ModelQuotaBucket` dual-cycle fields, `AccountQuotaData` GPT `json:"-"`, `ParseQuotaSummaryWithBuckets`)
- `internal/telemetry/pool.go` (F1, F2: `pollFromLanguageServer` bóc tách `RawBuckets` và dual-cycle)
- `internal/telemetry/quota_test.go` (File mới trong touch-map: regression test độc lập 5h/week, missing buckets, zero-GPT serialization, REST summary endpoint)

Tuyệt đối KHÔNG sửa bất kỳ file nào ngoài Touch-Map.

---

## 3. Kết Quả Xác Minh Thực Nghiệm

### 3.1. Kết quả chạy Reproduction Probe của Supervisor (`runtime_probe.go`)
Chạy lệnh: `go run reports/supervisor/evidence/2026-10-18-ui-redesign/runtime_probe.go`

```json
{
  "cloud_summary_payload": {
    "actual": {
      "Claude5h": 0.46,
      "ClaudeWeek": 0.12,
      "Gemini5h": 0.88,
      "GeminiWeek": 0.9,
      "HasData": true,
      "IsError": false
    },
    "expected": {
      "Claude5h": 0.46,
      "ClaudeWeek": 0.12,
      "Gemini5h": 0.88,
      "GeminiWeek": 0.9,
      "HasData": true,
      "IsError": false
    },
    "production_endpoint": "https://daily-cloudcode-pa.googleapis.com/v1internal:retrieveUserQuotaSummary",
    "raw_bucket_count": 4
  },
  "connect_rpc": {
    "actual": {
      "Claude5h": 0.46,
      "ClaudeWeek": 0.12,
      "Gemini5h": 0.88,
      "GeminiWeek": 0.9,
      "HasData": true,
      "IsError": false
    },
    "expected": {
      "Claude5h": 0.46,
      "ClaudeWeek": 0.12,
      "Gemini5h": 0.88,
      "GeminiWeek": 0.9,
      "HasData": true,
      "IsError": false
    },
    "matches_expected": true,
    "raw_bucket_count": 0,
    "serialized_gpt_field_present": false
  },
  "driver": "Fyne test.NewApp headless; not a native desktop screenshot",
  "single_card_refresh": {
    "actual_unchanged_sibling_cards": 5,
    "callback_ms": 0,
    "expected_unchanged_sibling_cards": 5,
    "latency_scope": "synthetic fetcher sleeps 50ms; no claim about real provider latency",
    "mock_completion_ms": 85.902,
    "polled_emails": [
      "account1@example.com"
    ]
  },
  "theme_monogram": {
    "after_light": "#2563eb",
    "before": "#89b4fa",
    "expected_light": "#2563eb"
  },
  "window": {
    "content_min": {
      "Width": 1200,
      "Height": 750
    },
    "first_card_min": {
      "Width": 198.01562,
      "Height": 81.96875
    },
    "first_card_size": {
      "Width": 472,
      "Height": 81.96875
    },
    "fixed_size": false,
    "grid_card_count": 6,
    "grid_min": {
      "Width": 620.9375,
      "Height": 167.9375
    },
    "grid_size": {
      "Width": 1424,
      "Height": 167.9375
    },
    "headless_canvas_size": {
      "Width": 1440,
      "Height": 900
    }
  }
}
```
*Đánh giá probe:*
- **F1:** `actual == expected` đạt 100% (Claude 5h 46%, week 12%; Gemini 5h 88%, week 90%).
- **F2:** `serialized_gpt_field_present` = `false`.
- **F3:** `actual_unchanged_sibling_cards` = `5` (trên tổng số 5 thẻ sibling).
- **F4:** Staticcheck exit 0.
- **F5:** `first_card_size.Height` = `81.97px` (giữ nguyên kích thước compact, không còn 378px).
- **F6:** `theme_monogram.after_light` = `#2563eb` (khớp hoàn hảo `expected_light`).

### 3.2. Staticcheck Toàn Bộ Workspace
Chạy lệnh: `staticcheck -tags gui ./...`
- Kết quả: **Exit 0, 0 warning, 0 error**.

### 3.3. Toàn Bộ Test Suite (10 Packages)
Chạy lệnh: `go test -v -count=1 ./...`
- `cmd/switcher`: **PASS** (4 test cases)
- `internal/connector`: **PASS** (13 test cases)
- `internal/manager`: **PASS** (17 test cases)
- `internal/model`: **PASS** (1 test case)
- `internal/reloader`: **PASS** (21 test cases)
- `internal/session`: **PASS** (2 test cases)
- `internal/store`: **PASS** (13 test cases)
- `internal/telemetry`: **PASS** (34 test cases, bao gồm regression tests mới)
- `internal/ui`: **PASS** (34 test cases, bao gồm 4 regression tests F1, F3, F5, F6)
- `internal/wincred`: **PASS** (3 test cases)
- **Tổng kết:** **10/10 package PASS 100%, 0 FAIL, 0 SKIP**.

### 3.4. Build Output
Chạy lệnh: `go build -tags gui -o .tmp/switcher-gui.exe ./cmd/switcher`
- Kết quả: **Exit 0**, nhị phân GUI biên dịch hoàn tất sạch sẽ.

---

## 4. Danh Sách Regression Test Bổ Sung

1. `internal/telemetry/quota_test.go:TestDualCycleQuota_IndependentFractionsAndMissingBuckets`: Kiểm thử độc lập 5h/week từ Connect-RPC và xử lý bucket thiếu trả về -1 (không giả 100%).
2. `internal/telemetry/quota_test.go:TestAccountQuotaData_ZeroGPTSerialization`: Kiểm thử JSON Marshal của `AccountQuotaData` tuyệt đối không chứa khóa `gpt`.
3. `internal/telemetry/quota_test.go:TestRetrieveUserQuotaSummary_RESTGroups`: Kiểm thử REST Client bóc tách đầy đủ 4 raw buckets từ cấu trúc `groups`.
4. `internal/ui/ui_test.go:TestOwnerRedesign_F1_DualCycleIndependentFractionsAndMissing`: Kiểm thử hiển thị UI 2 chu kỳ độc lập và nhãn `--%` khi thiếu bucket.
5. `internal/ui/ui_test.go:TestOwnerRedesign_F3_SingleCardRefreshKeepsSiblings`: Kiểm thử bấm nút refresh card độc lập, xác nhận con trỏ bộ nhớ của 5 thẻ sibling được bảo toàn nguyên vẹn.
6. `internal/ui/ui_test.go:TestOwnerRedesign_F5_CompactCardHeight`: Kiểm thử thẻ và lưới không bị giãn quá 100px chiều cao trong viewport 1440x900.
7. `internal/ui/ui_test.go:TestOwnerRedesign_F6_MonogramRecolorsOnThemeChange`: Kiểm thử đổi theme qua 3 trạng thái (Dark -> Light -> Warm -> Dark) đổi màu monogram AS chuẩn xác.

---

## 5. Kết Luận & Handoff

Tất cả 6 lỗi chặn nghiệm thu F1, F2, F3, F4, F5, F6 đã được khắc phục triệt để tận gốc, đảm bảo tính đóng kín của invariant family, 100% test pass, 0 staticcheck warning. Sẵn sàng bàn giao cho Supervisor kiểm định độc lập.