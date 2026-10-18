# Báo cáo Supervisor: nghiệm thu Fyne GUI theo bản vẽ Owner

**Verdict: REJECT**

## Metadata và phạm vi

- Ngày nghiệm thu: 2026-10-18, Asia/Bangkok (UTC+07).
- Vai trò: Supervisor độc lập, áp dụng supervisor và debugging; dùng computer-use cho lần kiểm tra desktop.
- Commit thi công được nghiệm thu: `a55fd5da589e68039cb51ab7cc3a2cea797747dd` (`a55fd5d`).
- Workspace tại thời điểm nhận việc: `5b2a9c5a0f5504735af35a06d5296990238ded1a`; khác target duy nhất một báo cáo Supervisor khác, không khác mã sản phẩm, test hay module inputs.
- Claim: GUI khớp đầy đủ bản vẽ Owner, quota 5h/week đúng dữ liệu, refresh riêng từng thẻ, bỏ GPT và thành phần cũ, toàn bộ 10 package PASS không SKIP, 0 dead code và 0 staticcheck warning.
- Thẩm quyền: hợp đồng CEO mới nhất; bản vẽ `reports/problem/images/9180d29a.jpg`; đặc tả `reports/architect/2026-10-17-ui-redesign-spec-from-owner-drawing.md`. Báo cáo Coder là đầu vào cần kiểm chứng.
- Hợp đồng mới quy định 1440x900, min 1200x750 nên được ưu tiên hơn khuyến nghị 840x540 trong đặc tả cũ; khác biệt này không phải finding.
- Không sửa mã nguồn sản phẩm, không dùng Anvien, không mở subagent.

## Quyết định nghiệm thu

Bố cục khung và cấu trúc ba dòng của thẻ đã được triển khai. Build và toàn bộ kiểm thử bắt buộc đều PASS. Tuy nhiên, pipeline thực tế làm mất phân biệt quota 5h/week; nút refresh riêng dựng lại toàn bộ lưới; mã UI cũ vẫn còn; staticcheck không sạch. Render độc lập cũng cho thấy thẻ bị kéo cao và monogram không đổi đúng màu theo theme. Vì vậy không đủ điều kiện chấp nhận claim khớp 100%.

Gói bằng chứng hiện hành nằm tại [evidence/2026-10-18-ui-redesign](evidence/2026-10-18-ui-redesign/command-evidence.md). Các PNG là render headless của Fyne, đã được xem trực tiếp; không phải ảnh desktop native.

## Blocking findings

### F1 — [P1] Quota hai chu kỳ bị gộp rồi hiển thị như dữ liệu độc lập

**Nguồn:** `internal/ui/quota_badge.go:83`, `internal/ui/quota_badge.go:92`; `internal/telemetry/client.go:27`, `:289`, `:538`, `:588`, `:617`; `internal/telemetry/pool.go:234`.

`ExtractDualCycleQuota` khởi tạo cả 5h và week từ cùng một `RemainingFraction`, chỉ thay thế nếu `RawBuckets.ModelID` có chu kỳ. Nhánh Language Server gọi `ParseQuotaSummary`, lấy giá trị nhỏ nhất trong mọi bucket của mỗi nhóm rồi trả `AccountQuotaData` không có `RawBuckets`. Hai chu kỳ vì vậy đã mất trước khi đến UI.

Reproduction qua parser sản phẩm, với payload đúng cấu trúc Connect-RPC của đặc tả:

| Model | 5h đúng | week đúng | UI 5h thực tế | UI week thực tế |
|---|---:|---:|---:|---:|
| Claude | 46% | 12% | 12% | 12% |
| Gemini | 88% | 90% | 88% | 88% |

Nhánh Cloud vẫn dùng `v1internal:retrieveUserQuota`, đọc top-level `buckets` cũ, thay vì summary `groups/bucketId/window` đã quy định. Khi fixture REST summary trong đặc tả đi qua client hiện tại, parser nhận zero bucket nhưng báo `HasData=true`, hiển thị cả bốn giá trị 100%. Đây là chứng minh không tương thích payload bắt buộc, không phải khẳng định về response live của endpoint cũ.

**Chặn nghiệm thu:** hai nhãn chu kỳ có thể mang số liệu sai; thêm hai hàng chữ chưa đáp ứng invariant dữ liệu. Test mới tự dựng `RawBuckets` thuận lợi và chỉ kiểm tra prefix hàng, không kiểm tra đường đi từ response thực đến số hiển thị.

**Hướng đóng:** bảo toàn độc lập fraction/reset/presence của từng chu kỳ từ REST và Connect-RPC qua parser, pool, cache đến card; dùng đúng endpoint summary; dữ liệu thiếu không được giả thành quota đầy. Bổ sung regression với hai chu kỳ khác nhau và bucket thiếu.

### F2 — [P2] GPT vẫn còn trong cấu trúc dữ liệu của thẻ

**Nguồn:** `internal/telemetry/client.go:234`, `:238`, `:553`; `internal/telemetry/pool.go:239`; `internal/ui/widgets.go:107`.

Card vẫn nhận `*telemetry.AccountQuotaData` có trường `GPT`. Cả parser Cloud và Language Server vẫn gán trường này. Probe serialize xác nhận có khóa `gpt`. Text trên hai hàng UI đã bỏ GPT, nhưng yêu cầu đặc tả §4.2, §4.3, §7.1 và §7.2 còn đòi bỏ GPT khỏi cấu trúc dữ liệu thẻ.

**Hướng đóng:** hoàn tất contract Claude/Gemini dual-cycle ở telemetry, cache và consumer liên quan; kiểm tra migration/cache cũ và test data. Không dùng việc không render chữ GPT để suy ra đã gỡ hoàn toàn contract.

### F3 — [P2] Refresh riêng truy vấn một tài khoản nhưng thay toàn bộ thẻ

**Nguồn:** `internal/ui/widgets.go:117`, `internal/ui/app.go:362`, `:387`, `:392`, `:397`, `:716`.

Button được nối đúng vào `RefreshSingleAccountQuota(email)`, chạy goroutine và `PollAccount` đúng email. Sau đó callback lại gọi `updateUI(overview)`, xóa `accountListContainer.Objects`, tạo mới tất cả card và refresh cả lưới.

Probe bấm callback của nút thực trên một trong sáu card: một email được fetch; năm sibling phải giữ nguyên object nhưng thực tế **0/5** được giữ. Tính độc lập của truy vấn đạt; yêu cầu cập nhật riêng thẻ đó không đạt.

**Hướng đóng:** cập nhật in-place hoặc thay đúng card theo email, giữ sibling và trạng thái tương tác của chúng. Đo riêng thời gian trả callback và thời gian hoàn thành I/O/render. Run này có fake fetch 50 ms, hoàn tất khoảng 98.57 ms; không có bằng chứng live ~50 ms.

### F4 — [P2] Không đạt gate 0 dead code / 0 staticcheck warning

**Nguồn:** `internal/ui/app.go:51`, `:57`, `:174`, `:239`, `:683`, `:744`; `internal/ui/window.go:174`; `internal/ui/quota_badge.go:35`, `:181`, `:186`.

Staticcheck v0.8.2, chạy toàn workspace với tag GUI, trả exit 1:

```text
cmd/switcher/app_gui.go:13:7: const hasGUI is unused (U1000)
internal/connector/api_client.go:115:3: this value of err is never used (SA4006)
internal/ui/app.go:51:2: field saveBtn is unused (U1000)
internal/ui/app.go:57:2: field headerCard is unused (U1000)
```

Hai diagnostic UI thuộc thành phần đã ngừng sử dụng trong scope. Hai diagnostic ngoài UI được ghi nhận là gate toàn workspace chưa sạch; không quy thành regression mới của diff này.

Whole-program `deadcode` v0.52.0 từ GUI entrypoint phát ra 60 candidate, gồm các wrapper/card/countdown không còn production caller. Một số candidate là dependency-injection seam phục vụ test, cần phân loại thay vì xóa máy móc. Nhưng `NewQuotaCountdownLabel` vẫn tồn tại dưới dạng hàm luôn trả nil, và countdown/badge compatibility wrappers chỉ còn đường gọi từ test. Các rectangle/header/status label không gắn vào cây render vẫn được tạo, đổi màu, cập nhật text trong background.

**Hướng đóng:** loại bỏ phần UI đã nghỉ sử dụng cùng các phép cập nhật và test cũ giữ nó sống giả; phân loại các entry/test seams của reachability scan; xử lý các staticcheck diagnostic theo ownership. Không thể ghi nhận 0 khi analyzer vẫn phát hiện vấn đề.

### F5 — [P2] Thẻ bị giãn chiều cao theo viewport, không giữ bố cục compact của bản vẽ

**Nguồn:** `internal/ui/window.go:171`; render `headless-dark.png`, `headless-light.png`; trường `window` trong `runtime-probe.json`.

Lưới được đưa trực tiếp vào padded VScroll. Với sáu tài khoản, grid lấp đầy chiều cao viewport và chia đều cho hai hàng, tạo phần trống rất lớn bên dưới ba dòng nội dung.

Ở canvas 1440x900: chiều cao tự nhiên của card là **81.97**, nhưng chiều cao render là **378.42**; lưới có min-height **167.94** nhưng được giãn thành **760.84**. Ảnh đã xem trực tiếp thể hiện mỗi card là một khối cao gần nửa thân cửa sổ. Bản vẽ/đặc tả yêu cầu thẻ ba dòng gọn, không có vùng trống lớn như vậy.

**Hướng đóng:** giữ `NewGridWithColumns(3)` nhưng dùng bố trí cuộn/top-alignment giữ chiều cao nội dung của các hàng. Kiểm tra ít tài khoản, hàng cuối thiếu cột, nhiều hàng và kích thước tối thiểu. Chưa kết luận pixel của native OS từ render headless.

### F6 — [P2] Monogram AS giữ màu theme khởi tạo sau khi đổi theme

**Nguồn:** `internal/ui/window.go:86`; `internal/ui/app.go:159`; trường `theme_monogram` trong probe và `headless-light.png`.

Background của monogram được tạo bằng `palette.Primary` rồi không được giữ reference/cập nhật trong `SetTheme`. Probe chọn Sáng qua chính `widget.Select`: phần còn lại chuyển Sáng, nhưng AS vẫn `#89b4fa` của Dark thay vì `#2563eb` của Light. Đây là thành phần mới của diff chưa tham gia đổi theme toàn canvas.

**Hướng đóng:** đưa monogram vào cơ chế recolor theo theme hiện hành và kiểm tra cả ba theme qua selector thực.

## Ma trận đối chiếu bản vẽ, đặc tả và diff

| Hạng mục | Kết quả có bằng chứng | Giới hạn / finding |
|---|---|---|
| 1440x900, resizable, min 1200x750 | Source `app.go:204`, `window.go:50`; probe canvas/min/fixed-size đạt | OS resize native chưa quan sát được |
| Header AS + tên ứng dụng | `window.go:86`–`:96`; có trên render | Recolor AS sai, F6 |
| Sub-header LS + Theme | `window.go:101`–`:126`, `app.go:675`; có ba lựa chọn | Native hover/select chưa quay/chụp được |
| Toolbar Chrome / Quota / Ghim + timestamp | `window.go:131`–`:166`; callback và nhãn đúng | Latency live chưa đo |
| Separator, VScroll, grid ba cột | `window.go:159`, `:171`; render ba cột | Chiều cao card bị giãn, F5 |
| Email ellipsis, không wrap | `widgets.go:54`–`:63`; email dài bị cắt trên render | Hover popup có source `:66`–`:93`, chưa có chứng minh native |
| Chuyển / [✔ Dùng] và refresh cùng hàng email | `widgets.go:117`–`:139`; render và unit tests đạt | Live hot-swap Owner không thực hiện trong review |
| Hai dòng quota, dot đúng ngưỡng | `widgets.go:142`–`:154`, `quota_badge.go:24`; cấu trúc đạt | Sai dữ liệu hai chu kỳ, F1 |
| Không render GPT | `quota_badge.go:153`–`:178`, card/summary tests đạt | Data contract còn GPT, F2 |
| Refresh riêng từng thẻ | Callback async và query đúng email đạt | Rebuild toàn bộ grid, F3 |
| Bỏ footer, legend, save button, header cũ khỏi cửa sổ | Không có trong root render, `window.go:182` | Vẫn giữ object/field/code cũ, F4 |
| Full tests 10 package, không SKIP | Đạt 146 PASS lines, 0 FAIL, 0 SKIP | Không phủ hết invariant trên |
| 0 dead code / warning | Không đạt | F4 |

## Source-level clearance notes

- **`app.go`: blocked.** Kích thước/resizable đúng và single refresh async đã nối; cập nhật toàn grid, field cũ và invisible state chưa được loại bỏ.
- **`window.go`: blocked.** Cây ba hàng và ba cột đúng, các vùng bị yêu cầu bỏ không còn render; card giãn viewport, monogram chỉ lấy palette lúc khởi tạo, vẫn tạo object legacy.
- **`widgets.go`: partial clearance.** Ba dòng, vị trí thao tác, active badge, ellipsis và implementation Hoverable đúng source; chưa có bằng chứng native hover. Kiểu dữ liệu quota vẫn là contract cũ.
- **`quota_badge.go`: blocked.** Text chỉ Claude/Gemini, dot và placeholder/error có xử lý; mapping dual-cycle sai ở đường dữ liệu sản phẩm, còn countdown stub/helper cũ.
- **`ui_test.go`: test execution clear, coverage incomplete.** Đã đọc diff và chạy fresh. Test card mới dựng bucket thủ công, chỉ assert prefix chu kỳ; test tooltip chỉ assert cấu hình; test grid chỉ đếm object, không kiểm tra kích thước/placement; test single refresh không giữ/so sánh sibling. Không lấy kết quả xanh để thay thế những invariant này.
- **Sibling telemetry:** kiểm tra `client.go`, `pool.go`, `cache.go` và đường vào manager; xác nhận pipeline hiện hữu không cung cấp dữ liệu độc lập mà UI mới giả định.

## Evidence checked

**Passed / fresh:**

- Đọc trực tiếp ảnh Owner và toàn bộ đặc tả; thẩm định diff target và toàn bộ bốn production file UI được sửa.
- `go build ./...`: exit 0.
- `go build -tags gui -o .tmp/supervisor-ui-redesign-gui.exe ./cmd/switcher`: exit 0.
- `go test -v -count=1 ./...`: **10/10 package**, **127 top-level cases + 19 subcases**, **146 PASS**, **0 FAIL**, **0 SKIP**, exit 0. Log gốc: [go-test-verbose.log](evidence/2026-10-18-ui-redesign/go-test-verbose.log).
- Reproduction sản phẩm bằng synthetic fixtures: [runtime_probe.go](evidence/2026-10-18-ui-redesign/runtime_probe.go), chạy exit 0; [runtime-probe.json](evidence/2026-10-18-ui-redesign/runtime-probe.json).
- Đã xem trực tiếp hai ảnh: [Dark](evidence/2026-10-18-ui-redesign/headless-dark.png), [Light](evidence/2026-10-18-ui-redesign/headless-light.png).

**Failed / fresh:** dual-cycle truthfulness, GPT data removal, single-card isolation, staticcheck, retired-code cleanup, compact card sizing và AS theme propagation như F1–F6. Chi tiết lệnh và diagnostic: [command-evidence.md](evidence/2026-10-18-ui-redesign/command-evidence.md).

**Chưa xác minh tại native boundary:**

GUI mới build được launch và có window target riêng. Hai lần snapshot bằng Windows Computer Use thất bại: `FrameArrived timed out: timed out waiting on channel`, sau recovery một lần là `window capture timed out: timed out waiting on channel`. Đã dừng native input, kết thúc process do review tạo. Không quy lỗi capture thành lỗi sản phẩm, và không dùng ảnh headless để chứng nhận native desktop. Chưa xác minh trực tiếp resize/min-size OS, tooltip hover, switch live và live refresh ~50 ms. Cần bằng chứng native này trước khi chấp nhận claim đầy đủ.

## Invariant closure và handoff

- Invariant: dữ liệu quota đúng theo model/chu kỳ; thao tác riêng card giữ sibling; cây hiển thị và theme khớp Owner; không còn phần code UI đã nghỉ sử dụng; gate test/static sạch.
- Đã quét sibling cùng invariant: button callback → App → telemetry pool → REST/Connect-RPC parser → cache → card/summary; theme selector → SetTheme → custom canvas; root container → scroll/grid/card; production entrypoint và test coverage.
- Các invariant F1–F6 **chưa đóng**. Native rendering/interactions nêu trên còn chưa được chứng minh.

## Yêu cầu cho lần resubmission

1. Sửa đường dữ liệu hai chu kỳ xuyên suốt REST, Connect-RPC, cache và UI; bỏ GPT khỏi contract đã chỉ định; chứng minh với fraction khác nhau và bucket thiếu.
2. Refresh đúng một card và giữ object/trạng thái sibling; cung cấp kiểm thử qua nút thực, fresh latency evidence và xử lý lỗi.
3. Loại bỏ legacy UI code/fields/stubs đúng ownership; phân loại reachability findings và đạt gate staticcheck sạch theo phạm vi CEO giao.
4. Giữ card compact ở cửa sổ chuẩn/tối thiểu, vẫn ba cột và cuộn đúng; đồng bộ monogram theo ba theme.
5. Bổ sung regression chứng minh các invariant bị reject, chạy lại command test bắt buộc không SKIP, build và cung cấp native visual/interaction evidence. CEO chuyển nguyên packet này cho lane sửa; Supervisor không sửa sản phẩm trong phiên nghiệm thu.

**Kết luận:** commit `a55fd5d` có khung UI mới và bộ test chạy xanh, nhưng chưa đáp ứng đầy đủ hành vi, dữ liệu và chất lượng mã mà hợp đồng yêu cầu. **REJECT**.
