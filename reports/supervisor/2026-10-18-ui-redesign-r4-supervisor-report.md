# Báo cáo Supervisor R4: nghiệm thu Fyne GUI tại 5996891

**Verdict: REJECT**

## Metadata, claim và authority

- Ngày kiểm định: **2026-10-18**, Asia/Bangkok (UTC+07); mốc chốt bằng chứng runtime **16:51:27**.
- Reviewer: Codex, Supervisor độc lập. Áp dụng `working-rules`, `supervisor`, `debugging` và quy tắc cô lập kiểm thử Go. Không sửa mã nguồn/test sản phẩm, không Anvien, không mở lane phụ.
- Target: **`5996891479f8271bfdbe5822f57dbd81b7161d0f`**. HEAD lúc nhận việc trùng target; working tree sạch.
- Claim kiểm định: đóng triệt để F1 quota presence/cache persistence/fallback/window matching và F4 helper retired; bảo toàn F2/F3/F5/F6; full suite 10/10 package, native GUI build và staticcheck đạt.
- Authority: hợp đồng CEO hiện hành; bản vẽ Owner `reports/problem/images/9180d29a.jpg`; đặc tả `reports/architect/2026-10-17-ui-redesign-spec-from-owner-drawing.md`; Addendum `reports/architect/2026-10-18-touchmap-addendum-gpt-removal.md`; các yêu cầu closure F1/F4 tại báo cáo Supervisor R3.
- Đầu vào: báo cáo R3 `reports/supervisor/2026-10-18-ui-redesign-r3-supervisor-report.md`, báo cáo Coder `reports/coder/2026-10-18-ui-redesign-r4-fix-report.md`. Kết luận dựa trên source và runtime R4; không nghiệm thu bằng tuyên bố của lane trước.
- Không có `AGENTS.md` trong repo hoặc `G:\`; `C:\Users\TAM NGUYEN\.codex\AGENTS.md` rỗng. Hợp đồng hiện hành xác định scope và quyền thực thi.
- Bằng chứng mới: [command-evidence.md](evidence/2026-10-18-ui-redesign-r4/command-evidence.md). Bằng chứng lịch sử được giữ nguyên.

## Quyết định

**Tất cả case lỗi cụ thể nêu trong handoff R3 đều đã được sửa. F4 đóng; F2/F3/F5/F6 được bảo toàn. Tuy nhiên F1 còn một trường hợp cùng invariant: hai cycle có giá trị zero tường minh trong cache bị UI đổi weekly thành unknown khi thiếu raw buckets và reset time. Vì vậy chưa thể chấp nhận claim đóng triệt để F1.**

| Invariant / gate | Kết quả R4 | Bằng chứng |
|---|---|---|
| F1: REST amount không có fraction, camel/snake | Đạt | Không dựng 100% hoặc 0%; tuple `(-1,-1,-1,-1)`, `HasData=false`, bốn `--%`, giữ đúng sau reload. E06. |
| F1: thiếu fraction và amount | Đạt | Cả bốn fraction unknown, `HasData=false`, không tạo 0%. E06. |
| F1: legacy Claude fraction 0 | Đạt | `(0,-1,-1,-1)`, không nhân sang weekly. E06. |
| F1: opaque ID, `window: weekly`, fraction 0 | Đạt | `(-1,0,-1,-1)`, đúng dòng week trước/sau cache; Connect-RPC thực cũng giữ đúng. E06/E07. |
| F1: cache weekly-only zero, thiếu field/family | Đạt | Weekly zero giữ nguyên; Claude 20% không mất khi thiếu Gemini; family/field thiếu là unknown. Kiểm tra cả family đối xứng và null. E06. |
| F1: zero tường minh trên cả hai cycle, không raw/reset | **Chưa đóng** | Storage `(0,0,-1,-1)` nhưng badge `(0,-1,-1,-1)`; tương tự Gemini. E06/E10. |
| F2: Zero GPT contract/serialization | Bảo toàn | Reflection không có field GPT, JSON không có khóa GPT, không còn field consumer/literal trong `cmd`/`internal`. E06/E08. |
| F3: single card refresh | Bảo toàn | Bốn lần tap thực, mỗi lần một email, giữ 5/5 sibling và focus/disable state, gồm nhánh lỗi. E06. |
| F4: năm helper retired | **Đóng** | AST/source: không còn năm helper; test dùng API production trực tiếp; bốn field cũ và hai hàm countdown vẫn vắng. E00/E06/E08. |
| F5: compact ở 1440×900 và 1200×750 | Bảo toàn | Card **81.96875 px**; sweep 1/2/6/7/31 account, không giãn theo viewport. E06. |
| F6: monogram AS theme Sáng | Bảo toàn | Selector đổi background object thực sang **#2563eb**; Warm/Dark cũng đúng; đã xem PNG. E06. |
| Full suite | Đạt | **10/10 package, 161 PASS, 0 FAIL, 0 SKIP**, exit 0. E04. |
| Workspace + GUI native build | Đạt | Cả hai exit 0. E02/E03. |
| `staticcheck -tags gui ./...` | Đạt | **Exit 0, 0 diagnostic**. E05. |

## Blocking finding

### F1 — [P2] Weekly zero tường minh bị loại khỏi badge khi cache có cả hai cycle zero nhưng không có raw/reset

**Source:** `internal/ui/quota_badge.go:79`–`:80`; caller cho hai family tại `:146`–`:147`; card production `NewAccountCardWithQuota` tại `internal/ui/widgets.go`. Contract cho raw bucket tùy chọn: `internal/telemetry/client.go:317`. Unmarshal giữ zero tường minh tại `client.go:262`–`:275`.

Input cache hợp lệ tối giản cho Claude:

```json
{"matrix":{"legacy@example.com":{"email":"legacy@example.com","claude":{"remaining_fraction":0,"five_hour_fraction":0,"weekly_fraction":0}}}}
```

`raw_buckets` và reset time không có. Cả hai fraction cycle đều **có mặt tường minh**, không phải zero mặc định do thiếu field. `LoadFromDisk` đọc đúng `(0,0)` vào Claude; serialize rồi reload tiếp vẫn giữ đúng trong storage. Nhưng `extractFamilyCycles` chạy nhánh được chú thích dành cho struct literal trong test:

```go
if bucket.RemainingFraction == 0 && bucket.FiveHourFraction == 0 && bucket.WeeklyFraction == 0 && bucket.FiveHourReset.IsZero() && bucket.WeeklyReset.IsZero() && len(rawBuckets) == 0 {
    return 0.0, -1.0
}
```

Nhánh trả về sớm trước khi dùng cycle data đã xác định, nên weekly 0% thật biến thành `--%`. Cùng hàm dùng cho Gemini nên lỗi đối xứng.

| Input / stage | Storage sau reload | Badge kỳ vọng | Badge thực tế |
|---|---|---|---|
| Claude hai cycle zero, không raw/reset | `(0,0,-1,-1)` | `(0,0,-1,-1)` | **`(0,-1,-1,-1)`** |
| Gemini hai cycle zero, không raw/reset | `(-1,-1,0,0)` | `(-1,-1,0,0)` | **`(-1,-1,0,-1)`** |
| Claude hai cycle zero, có raw buckets | `(0,0,-1,-1)` | `(0,0,-1,-1)` | Đúng — đối chứng dương |
| Claude weekly-only zero, không raw | `(-1,0,-1,-1)` | `(-1,0,-1,-1)` | Đúng — case R3 đã sửa |

Tuple có thứ tự `(Claude5h, ClaudeWeek, Gemini5h, GeminiWeek)`; `-1` là unknown.

**Bằng chứng trực tiếp:**

- E06 chạy 33 fixture/79 quan sát, **75 khớp, 4 sai**. Bốn quan sát sai là hai family ở lần load đầu và lần save/reload tiếp; JSON ghi riêng storage, extracted fractions và badge text. Các case R3 đã sửa được phân biệt rõ với lỗi mới này.
- E10 xác nhận bằng **API công khai và card production**, không chỉ JSON viết tay: `ParseQuotaSummary` → `AccountQuotaData` có cycle đã parse, không raw → `QuotaCache.Update` → `SaveToDisk` → instance mới `LoadFromDisk` → `NewAccountCardWithQuota`. Parser và disk storage đều đúng; card sai weekly ở hai case no-raw; hai đối chứng trong bảng đạt.
- Đã xem trực tiếp [E10-zero-cycle-cards.png](evidence/2026-10-18-ui-redesign-r4/E10-zero-cycle-cards.png): card Claude/Gemini no-raw hiện week `--%`; card giữ raw và card weekly-only hiện đúng 0%.
- Probe E06 và E10 exit 1 do mismatch được ghi nhận, không panic, compile error hoặc lỗi hạ tầng.

**Vì sao chặn:** F1 yêu cầu phân biệt unknown với zero thật trên toàn pipeline, gồm cache và fallback. Raw buckets là field tùy chọn trong contract; fraction tường minh không được mất presence chỉ vì thiếu metadata bổ sung. Lỗi này nằm trong chính extractor vừa sửa và cùng cache/presence invariant, không mở rộng sang miền khác.

**Phạm vi kết luận:** REST/Connect-RPC hiện hành có giữ raw và các case R3 cụ thể đều đạt. Không kết luận serializer còn làm mất weekly-only zero; serializer đã sửa đúng. Lỗi còn lại nằm ở diễn giải UI của dữ liệu hợp lệ khi không có raw/reset.

**Hướng đóng:** giữ mọi cycle fraction tường minh `>= 0`, kể cả khi cả hai bằng zero và không có reset/raw. Biểu diễn cycle vắng bằng sentinel `-1` tại data construction/legacy normalization; tránh nhánh production suy diễn absence để hỗ trợ literal của test. Giữ fallback legacy một cycle, empty/missing/null và missing-family đã đạt.

**Bằng chứng re-review cần có:** cache/card cho Claude và Gemini với hai zero, không raw/reset; đối chứng có raw và weekly-only zero; missing-cycle vẫn unknown, legacy zero không tạo tuần giả. Chạy lại regression và các gate bắt buộc sau sửa sản phẩm.

## Các invariant đã đóng / bảo toàn

### F1: các case R3 đã đóng

REST legacy amount 37 ở camel/snake, amount 0 không fraction, missing/null fraction đều unknown ở pool và disk reload. Fraction legacy 0 chỉ hiện 5h. Opaque weekly zero được giữ đúng vào week, kể cả không reset; RPC thực E07 xác minh chuỗi `GetUserStatus` → `RetrieveUserQuotaSummary` → `PollAccount` → cache với method/CSRF và đúng hai request. Cả bảy subcase RPC đạt; Claude và Gemini được kiểm tra đối xứng.

Cache thiếu Gemini giữ Claude 20%; thiếu Claude giữ Gemini 80%; missing/null family và fraction không tạo quota giả. Kiểm tra cả initial load và save/reload; camelCase weekly zero cũng giữ đúng. Aggregate có unknown đứng trước hoặc sau bucket hợp lệ đều giữ model có dữ liệu.

### F2/F4: contract và helper

`AccountQuotaData` thực không có GPT; JSON mới không có khóa GPT; source sweep không còn field consumer/literal. Chuỗi tên group upstream và cột CLI N/A được giữ theo Addendum, không thuộc field GPT của contract thẻ.

Năm helper `BuildActiveQuotaSummary`, `NewQuotaBadgeGroup`, `NewDualCycleQuotaBadge`, `NewAccountCard`, `RefreshSingleQuota` đã bị xóa; không còn declaration/call trong UI. Diff test chuyển sang `NewAccountCardWithQuota`, `RefreshSingleAccountQuota`, `ExtractDualCycleQuota`, `CreateQuotaCycleRow`. AST/reflection xác nhận bốn field cũ và countdown vẫn vắng. Staticcheck 0 diagnostic hỗ trợ gate; việc đóng helper còn dựa trên diff/source và inventory symbol, không suy ra chỉ từ staticcheck.

### F3/F5/F6: runtime Fyne

- Bấm nút refresh sản phẩm ở index 0, 2, 5 và nhánh lỗi index 3: đúng một email mỗi lần, chỉ target card đổi, giữ **5/5** sibling cùng object và focus/disable state. Nhánh lỗi chỉ target hiện `[!] Lỗi kết nối`. Hoàn tất synthetic **95.479–106.773 ms**, gồm fetcher chờ 50 ms; không diễn giải thành provider live latency.
- Layout tự nhiên tại hai viewport 1440×900 và 1200×750, sweep 1/2/6/7/31 account: chiều cao card **81.96875 px**, bằng min-size, không bị kéo giãn. Grid sáu card **1424×167.9375** và **1184×167.9375**; 31 card scroll được đến đáy.
- Qua `widget.Select`, monogram background thực đổi Sáng **#2563eb**, Ấm **#d97706**, Tối **#89b4fa**. Đã xem [Dark](evidence/2026-10-18-ui-redesign-r4/E06-headless-dark.png) và [Light](evidence/2026-10-18-ui-redesign-r4/E06-headless-light.png): lưới ba cột, ba dòng, card compact, không có quota GPT.

## Source-level clearance notes

- **`internal/telemetry/client.go`: clear cho thay đổi R4 được kiểm định.** Cycle zero được serialize tường minh; missing family/field default -1; legacy amount không dựng fraction; aggregate nhận valid sau unknown; raw summary giữ cycle/family metadata. Parser/storage của counterexample cũng đúng.
- **`internal/ui/quota_badge.go`: F4 clear; F1 blocked.** Ba helper retired tại file này đã bị xóa. Family xử lý độc lập và weekly-only zero đúng; nhánh tại `:79` làm mất weekly zero ở counterexample.
- **`internal/ui/app.go`: clear cho thay đổi R4 và bảo toàn F3/F6.** Chỉ alias retired bị xóa; single refresh vẫn thay `Objects[targetIdx]`, selector cập nhật monogram.
- **`internal/ui/widgets.go`: clear cho thay đổi R4.** Wrapper retired đã xóa; production card vẫn ba dòng và gọi extractor hiện hành. Card phản ánh lỗi F1 từ extractor, không có lỗi layout mới.
- **`internal/telemetry/quota_test.go`, `telemetry_test.go`, `internal/ui/ui_test.go`: đã đọc thay đổi và chạy fresh.** Các test cập nhật đạt; chưa che phủ trường hợp cả hai zero tường minh không raw/reset tại card. Probe độc lập phát hiện gap này.
- **Siblings `internal/telemetry/cache.go`, `pool.go`, `internal/ui/window.go`: đã trace.** Storage giữ đúng zero, pool/REST/RPC qua API sản phẩm; layout giữ compact, monogram có trong render tree. Không sửa các file này.

## Evidence checked, freshness và giới hạn

**Passed — fresh tại target:**

- E00/E01: đối soát diff, đọc source liên quan và bản vẽ; Go 1.26.3, Windows/amd64, CGO=1, gcc 16.1.0, staticcheck 2026.2.2.
- E02/E03: workspace build **4398 ms**, native GUI build **10687 ms**, exit 0. GUI build dùng `-tags gui`, linker Windows GUI và output tạm để không ghi đè binary người dùng.
- E04: `go test -v -count=1 ./...`, **34808 ms**, exit 0; **10/10 package, 161 PASS = 142 top-level + 19 subcase, 0 FAIL, 0 SKIP**. Đã đọc log đầy đủ. Các package: cmd/switcher; connector, manager, model, reloader, scanner, store, telemetry, ui, wincred.
- E05: `staticcheck -tags gui ./...`, **3212 ms**, exit 0, **0 diagnostic**.
- E06/E07/E08: các case R3 và các invariant đạt đã phân định ở trên; source search 0 match dùng exit 1 chuẩn của `rg`.

**Failed — fresh:**

- E06 runtime probe exit **1**, **16982 ms**: bốn quan sát F1 sai, không có failure assertion F2/F3/F4/F5/F6.
- E10 card probe exit **1**: hai counterexample sai và hai positive control đúng; storage đúng, render weekly sai như finding.

**Not run / giới hạn:**

- Không launch/capture desktop native hoặc kiểm thử OS minimize/hover trong run này. PNG là **Fyne software canvas**, không phải screenshot Windows. Native GUI **build** đã đạt; bảo toàn UI được xác nhận ở ranh giới Fyne như R3.
- RPC probe seed discovery cache của pool synthetic để trỏ loopback; không dùng làm chứng nhận live Language Server discovery. Suite chính thức có các test live discovery/read sẵn có, không SKIP.
- Không đo quota-provider live latency, không hot-swap tài khoản Owner trong probe. Không chạy lại gate đã xanh sau khi chỉ thêm báo cáo/evidence build-excluded; không sửa sản phẩm.

## Invariant closure và required fix list

Đã sweep REST/Connect-RPC → parser → pool → cache save/load → extractor → card; Go contract/serialization/callers; retired symbol/test migration; refresh/sibling state; layout/scroll; selector/monogram.

Residual trong scope: **F1 explicit both-cycle zero không raw/reset bị UI bỏ weekly**. F2/F3/F4/F5/F6 không còn residual trong ranh giới kiểm định đã mô tả.

1. Sửa F1 tại extractor/normalization để hai zero tường minh luôn được giữ cho cả Claude và Gemini; không hồi quy các case R3 đã đóng.
2. Thêm regression qua cache và card cho counterexample và đối chứng, dùng production APIs; không chỉ assert parser hoặc stored fields.
3. Sau sửa, cung cấp full suite 10/10 không FAIL/SKIP, native GUI build, staticcheck 0 diagnostic và bằng chứng bảo toàn F2/F3/F4/F5/F6.

**Đánh giá tổng thể:** bản R4 giải quyết các lỗi R3 được nêu cụ thể và dọn đủ helper retired; build/suite/staticcheck đều đạt. Một nhánh mới trong extractor vẫn làm mất dữ liệu zero hợp lệ tại card, nên claim đóng triệt để F1 bị **REJECT**. Supervisor chỉ ghi nhận và bàn giao về CEO, không sửa sản phẩm.
