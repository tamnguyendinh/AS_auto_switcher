# Báo cáo Supervisor R5: nghiệm thu Fyne GUI tại 28a7e3e

**Verdict: REJECT**

## Metadata, claim và authority

- Ngày kiểm định: **2026-10-18**, Asia/Bangkok (UTC+07). Probe cuối bắt đầu **17:24:21**, hoàn thành sau **13.397 giây**; chốt đọc bằng chứng lúc **17:25**.
- Reviewer: Codex, Supervisor độc lập. Áp dụng `working-rules`, `supervisor`, `debugging` và quy tắc cô lập kiểm thử Go.
- Target: **`28a7e3e4594550c394885b8502293d52a765d167`**. HEAD lúc nhận việc trùng target, working tree sạch.
- Claim: đóng triệt để F1 — giữ hai cycle zero tường minh khi không có raw/reset cho Claude và Gemini; bảo toàn F2–F6; full suite, native GUI build và staticcheck đạt.
- Authority: hợp đồng CEO vòng 5; bản vẽ Owner `reports/problem/images/9180d29a.jpg`; đặc tả `reports/architect/2026-10-17-ui-redesign-spec-from-owner-drawing.md`; Addendum `reports/architect/2026-10-18-touchmap-addendum-gpt-removal.md`; closure F1 tại báo cáo Supervisor R4 `reports/supervisor/2026-10-18-ui-redesign-r4-supervisor-report.md`.
- Đầu vào: báo cáo Coder `reports/coder/2026-10-18-ui-redesign-r5-fix-report.md`. Kết luận dựa trên source và lần chạy mới của Supervisor; không dùng kết quả probe Coder làm bằng chứng nghiệm thu.
- Không có `AGENTS.md` ở root repo hoặc `G:\`; file `C:\Users\TAM NGUYEN\.codex\AGENTS.md` không có nội dung. Scope và quyền thực thi theo hợp đồng hiện hành.
- Không sửa mã nguồn hoặc test sản phẩm; không Anvien; không mở lane phụ; giữ nguyên bằng chứng lịch sử. Output mới thuộc `reports/supervisor/`.

## Quyết định

**Counterexample cụ thể ở R4 với `RemainingFraction == 0` đã được sửa. Tuy nhiên cùng invariant F1 vẫn hỏng khi legacy `RemainingFraction > 0` tồn tại cùng hai field cycle zero tường minh. UI bỏ cả hai cycle, dùng legacy cho 5h và đổi weekly thành unknown. Vì vậy chưa thể chấp nhận claim đóng triệt để F1.**

F2/F3/F4/F5/F6 được bảo toàn. Full suite đạt **10/10 package, 163 PASS, 0 FAIL, 0 SKIP**; native GUI build thành công; staticcheck **0 diagnostic**.

| Invariant / gate | Kết quả R5 | Bằng chứng mới |
|---|---|---|
| F1: Claude/Gemini cả hai zero, legacy fraction 0, không raw/reset | Đạt; đóng counterexample R4 | Parser hoặc JSON → cache save/reload hai lần → extractor → card production giữ đúng zero. E06/E07. |
| F1: cả hai family zero; camelCase; không có legacy fraction | Đạt | Bốn nhãn `0%`; không tự biến zero thành `--%`. E07. |
| F1: unknown `-1`, thiếu/null family, legacy chỉ một cycle | Đạt | Unknown giữ bốn `--%`, `HasData=false`; legacy 0 chỉ hiện ở 5h, không tạo tuần giả. E06/E07. |
| F1: explicit cycle zero cùng legacy fraction dương, không raw/reset | **Chưa đóng** | Storage `(0,0)` nhưng card thành `(legacy,-1)` cho cả Claude và Gemini. Ba fixture, ba stage, **9 quan sát sai**. E06/E07. |
| F2: Zero GPT contract/serialization | Bảo toàn | Reflection không có field GPT; JSON không có khóa GPT; source consumer sạch. E06/E08. |
| F3: refresh một thẻ | Bảo toàn | Bốn lần tap; mỗi lần chỉ một email, 5/5 sibling giữ object, focus và disable state được giữ, kể cả nhánh lỗi. E06. |
| F4: retired helpers | Bảo toàn | AST/reflection/source không có năm helper retired, hai countdown helper hoặc bốn field cũ. Staticcheck 0 diagnostic. E05/E06/E08. |
| F5: card compact | Bảo toàn | **81.96875 px** tại 1440×900 và 1200×750; sweep 1/2/6/7/31 account. E06. |
| F6: monogram AS ở theme Sáng | Bảo toàn | Selector cập nhật background object thực sang **#2563eb**; Warm/Dark cũng khớp. E06 và ảnh Light. |
| Workspace build | Đạt | `go build ./...`, exit 0. E02. |
| Native GUI build | Đạt | `go build -tags gui ... ./cmd/switcher`, exit 0. E03. |
| Full test suite | Đạt | `go test -v -count=1 ./...`, 163 PASS = 144 top-level + 19 subcase. E04. |
| Staticcheck | Đạt | `staticcheck -tags gui ./...`, exit 0, 0 diagnostic. E05. |

## Blocking finding

### F1 — [P2] Nhánh legacy dương vẫn phủ nhận hai cycle zero tường minh

**Source:** `internal/ui/quota_badge.go:73`–`:77`; legacy fallback tại `:124`–`:128`; caller cho hai family tại `:142`–`:143`. Card production dùng kết quả này tại `internal/ui/widgets.go:137`–`:143`.

Input cache tối giản cho Claude:

```json
{
  "matrix": {
    "claude-legacy-positive@example.com": {
      "email": "claude-legacy-positive@example.com",
      "claude": {
        "remaining_fraction": 0.2,
        "five_hour_fraction": 0,
        "weekly_fraction": 0
      }
    }
  }
}
```

Cả hai field cycle **có mặt tường minh**, không có raw buckets hoặc reset time. `ModelQuotaBucket.UnmarshalJSON` tại `internal/telemetry/client.go:238` dùng pointer để phân biệt field vắng với zero; `QuotaCache.SaveToDisk`/`LoadFromDisk` giữ nguyên cả hai zero. Tuy nhiên nhánh sau vẫn tồn tại:

```go
if bucket.RemainingFraction > 0 && bucket.FiveHourFraction == 0 && bucket.WeeklyFraction == 0 && bucket.FiveHourReset.IsZero() && bucket.WeeklyReset.IsZero() {
    has5h = false
    hasWeek = false
}
```

Không còn cycle được công nhận và không có raw để bổ sung, extractor chạy fallback legacy: `h5 = bucket.RemainingFraction; week = -1`. Hậu quả là mất weekly zero và thay cả 5h zero bằng số dương.

Tuple trong bảng có thứ tự `(Claude5h, ClaudeWeek, Gemini5h, GeminiWeek)`; `-1` nghĩa là unknown.

| Fixture | Storage cycle trước/sau reload | Legacy C/G | Badge kỳ vọng | Badge thực tế |
|---|---|---|---|---|
| Claude hai zero, không raw/reset | `(0,0,-1,-1)` | `(0.2,-1)` | `(0,0,-1,-1)` | **`(0.2,-1,-1,-1)`** |
| Gemini hai zero, không raw/reset | `(-1,-1,0,0)` | `(-1,0.8)` | `(-1,-1,0,0)` | **`(-1,-1,0.8,-1)`** |
| Cả hai family hai zero | `(0,0,0,0)` | `(0.2,0.8)` | `(0,0,0,0)` | **`(0.2,-1,0.8,-1)`** |
| Claude cùng input nhưng có reset | `(0,0,-1,-1)` | `(0.2,-1)` | `(0,0,-1,-1)` | Đúng — đối chứng metadata |
| Claude cùng input nhưng có raw cycles | `(0,0,-1,-1)` | `(0.2,-1)` | `(0,0,-1,-1)` | Đúng — đối chứng raw |

**Bằng chứng thực nghiệm:**

- E07 có **20 fixture × 3 stage = 60 quan sát**: normalized input, save/reload instance mới lần 1, save/reload instance mới lần 2. **51 khớp, 9 sai**; ba fixture legacy dương sai ở cả ba stage. Mỗi quan sát kiểm tra extractor lẫn text theo đúng dòng 5h/week của `NewAccountCardWithQuota`.
- Các field cycle trong JSON được decode qua production unmarshaler; không dùng struct zero mặc định để giả lập presence. Storage giữ đúng zero ở cả hai lần reload; lỗi nằm ở diễn giải UI.
- E06 có **256 assertion, 238 đạt, 18 fail**. Toàn bộ 18 failure là hai assertion extractor/card cho chín quan sát trên. Không có failure F2/F3/F4/F5/F6. Probe exit **1** vì mismatch; stderr chỉ có `exit status 1`, không compile error hoặc panic.
- Đã xem [E07-explicit-cycle-cards.png](evidence/2026-10-18-ui-redesign-r5/E07-explicit-cycle-cards.png): hàng đầu là đối chứng zero đã sửa; hàng giữa là unknown/legacy-only/weekly-only; hàng cuối cho thấy 5h **20%/80%** và week **`--%`** ở ba fixture bị lỗi.

**Vì sao cùng scope và chặn acceptance:** F1 yêu cầu giữ cycle zero tường minh khi thiếu raw/reset; closure R4 còn yêu cầu tránh suy diễn absence trong production để hỗ trợ struct literal của test. JSON contract cho phép lưu legacy fraction cùng cycle fields, và production cache đọc/lưu nguyên trạng input này. Source comment của extractor cũng quy định explicit cycle `>= 0` được ưu tiên, fallback chỉ khi không có cycle-specific data. Nhánh `> 0` phá chính invariant đó. Đây là đường cache → extractor → card đang được nghiệm thu, không phải lỗi miền khác.

**Phạm vi kết luận:** counterexample dùng legacy scalar dương khác hai cycle zero. Không khẳng định parser summary hiện hành tự sinh legacy scalar mâu thuẫn này từ response provider nhất quán. Parser tính aggregate và các case R4 thông thường đã đạt; vấn đề là UI không tôn trọng cycle fields tường minh đã được cache/unmarshaler chấp nhận. Không có căn cứ trong contract hiện hành để dùng legacy scalar xóa presence của cycle.

**Hướng đóng:** bảo toàn cycle fields tường minh bất kể legacy scalar là âm, zero hay dương. Missing cycle tiếp tục biểu diễn bằng `-1` tại construction/unmarshal; fallback legacy chỉ chạy khi thật sự không có cycle-specific data. Nếu test cũ dựa vào zero mặc định, chuẩn hóa fixture thiếu cycle thay vì để production đoán absence từ giá trị 0. Supervisor chỉ nêu hướng sửa, không sửa sản phẩm.

## Source-level clearance notes

- **`internal/ui/quota_badge.go`: blocked cho F1.** Diff R5 xóa bốn dòng của nhánh `RemainingFraction == 0`; counterexample R4 đã đóng. Nhánh `RemainingFraction > 0` ở `:74` vẫn bỏ explicit zero như finding. Hai family dùng chung extractor nên cùng chịu ảnh hưởng.
- **`internal/telemetry/quota_test.go`, `internal/ui/ui_test.go`: đã đọc diff và chạy mới.** Hai test thêm ở R5 đạt; cover scalar 0, parser/cache/card và family bất đối xứng. Chưa cover legacy scalar dương cùng hai explicit cycle zero. Không thay đổi test sản phẩm khi review.
- **Siblings `internal/telemetry/client.go`, `cache.go`: clear cho storage/presence được kiểm định.** Unmarshal phân biệt missing/zero, thiếu family về `-1`; save/reload giữ đúng cycle fields, kể cả counterexample. REST amount không fraction vẫn unknown; không phát hiện hồi quy serializer.
- **`internal/ui/widgets.go`, `window.go`: clear cho F3/F5 và render wiring.** Card dùng extractor production, ba dòng; grid ba cột nằm trong VBox/VScroll, card giữ minimum height. Sai số hiển thị F1 bắt nguồn ở extractor.
- **`internal/ui/app.go`: clear cho F3/F6/F4 trong phạm vi kiểm định.** Single refresh chỉ thay `Objects[targetIdx]` tại `:407`; selector cập nhật `monogramBg.FillColor` tại `:179`; không hồi sinh helper retired.
- **F2/F4 source sweep:** không có field GPT consumer trong `cmd`/`internal`; AST/reflection không có năm helper `BuildActiveQuotaSummary`, `NewQuotaBadgeGroup`, `NewDualCycleQuotaBadge`, `NewAccountCard`, `RefreshSingleQuota`; hai countdown helper và bốn field cũ cũng vắng. Chuỗi tên group upstream và CLI N/A được giữ theo Addendum.

## Evidence checked, freshness và giới hạn

Mọi kết quả dưới đây là lần chạy mới trên source target, không dùng output lịch sử làm bằng chứng PASS. Xem [command-evidence.md](evidence/2026-10-18-ui-redesign-r5/command-evidence.md) và JSON/log đi kèm.

**Passed:**

- E00/E01: `git show 28a7e3e` đối soát diff sản phẩm/test; đã đọc source liên quan và bản vẽ. Toolchain Go **1.26.3**, Windows/amd64, CGO=1, GCC **16.1.0**, staticcheck **2026.2.2**.
- E02: workspace build, exit **0**, **5569 ms**.
- E03: native GUI build có `-tags gui`, `-ldflags -H=windowsgui`, output riêng trong `.tmp`, exit **0**, **8755 ms**. Binary kiểm định đã được runner dọn.
- E04: `go test -v -count=1 ./...`, exit **0**, **29218 ms**; **10/10 package, 163 PASS, 0 FAIL, 0 SKIP**. Đã đọc toàn bộ verbose log. Package gồm cmd/switcher; connector, manager, model, reloader, scanner, store, telemetry, ui, wincred.
- E05: `staticcheck -tags gui ./...`, exit **0**, **2254 ms**, **0 diagnostic**.
- E06: ma trận hồi quy Supervisor R4 được chạy lại trong không gian R5: **33 fixture, 79/79 quan sát đạt**. Các case legacy amount/missing/null, opaque weekly zero, legacy cache, missing family và hai cycle zero scalar 0 đều đạt.
- E06/F3: tap nút refresh thật ở index 0/2/5 và nhánh lỗi index 3. Mỗi lần chỉ một email, giữ 5/5 sibling, grid object, focus trong quá trình cập nhật và disable state. Completion synthetic **93.703–96.925 ms**, gồm fetcher delay 50 ms; không coi đây là live provider latency.
- E06/F5: 10 trạng thái count/viewport, card **81.96875 px** bằng minimum height; grid sáu card **1424×167.9375** và **1184×167.9375**. Với 31 account, scroll-to-bottom offset Y **180.8125/330.8125** ở hai viewport.
- E06/F6: qua `widget.Select`, Sáng **#2563eb**, Ấm **#d97706**, Tối **#89b4fa** trên background monogram thực. Đã xem ảnh Light cuối cùng và ảnh Dark của harness cùng source.
- E08/E09: source search không có retired declarations/consumers; `rg` exit **1** ở các search không match là kết quả vắng symbol, không phải lỗi tool.

**Failed:**

- E06/E07: probe cuối exit **1**, **13397 ms**, do residual F1 nêu trên. **Không đạt yêu cầu F1–F6 100%** dù suite chính thức và các gate build/staticcheck đều xanh.

**Not run / giới hạn:**

- Không launch/capture desktop native, không kiểm thử OS minimize/hover. PNG dùng **Fyne software canvas**, không phải screenshot Windows. Native GUI **build** đã được kiểm định; runtime invariants trong scope được kiểm tra tại ranh giới Fyne như vòng 4.
- HTTP là loopback với token/account synthetic. Không dùng probe để chứng nhận provider live latency hoặc hot-swap tài khoản Owner. Full suite có các test live discovery/read sẵn có và không SKIP.
- Harness có một lần chỉnh theme cho ảnh đối chứng: đồng bộ app theme Sáng với palette Sáng để email/icon đọc được, rồi chạy lại probe. Chỉ giữ output cuối; không sửa sản phẩm và không chạy lại các gate đã xanh sau thay đổi evidence build-excluded.

## Invariant closure và required fix list

Đã kiểm tra parser/REST → pool → cache save/load → extractor → card; đối chứng missing/zero/legacy; Go contract/serialization/callers; retired symbols; refresh/sibling state; layout/scroll; selector/monogram.

**Residual trong scope duy nhất:** F1 explicit both-cycle zero cùng legacy scalar dương bị UI thay bằng legacy 5h và unknown weekly khi không raw/reset. F2/F3/F4/F5/F6 không có residual trong ranh giới runtime đã nêu.

1. Coder xử lý nhánh `internal/ui/quota_badge.go:74` để mọi explicit cycle zero được ưu tiên, không suy ra missing từ zero để hỗ trợ fixture cũ.
2. Bổ sung regression JSON/cache/card cho Claude, Gemini và cả hai family với legacy scalar dương + hai cycle zero, không raw/reset. Giữ đối chứng legacy-only 0/dương, missing/null, weekly-only zero, có raw/reset và toàn bộ case R4 đã đạt.
3. Sau sửa, cung cấp test suite 10/10 không FAIL/SKIP, native GUI build, staticcheck 0 diagnostic và bằng chứng bảo toàn F2–F6. Supervisor nghiệm thu source/runtime mới độc lập.

**Đánh giá tổng thể:** R5 sửa đúng nhánh scalar-zero được nêu cụ thể ở R4 và bảo toàn F2–F6. Nhánh scalar-positive còn phủ nhận chính dữ liệu cycle zero tường minh trên cùng đường cache/card, nên claim giải quyết triệt để F1 bị **REJECT**. Báo cáo và toàn bộ bằng chứng mới được commit riêng rồi bàn giao CEO; lane dừng sau handoff.
