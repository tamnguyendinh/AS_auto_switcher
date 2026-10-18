# Báo cáo Supervisor R3: nghiệm thu Fyne GUI tại a3c0cdb

**Verdict: REJECT**

## Metadata, claim và authority

- Thời điểm kiểm định: 2026-10-18, khoảng 15:56–16:11, Asia/Bangkok (UTC+07).
- Reviewer: Codex, Supervisor độc lập. Áp dụng `working-rules`, `supervisor`, `debugging` và quy tắc cô lập probe Go. Không dùng Anvien, không mở subagent, không sửa mã nguồn hoặc test sản phẩm.
- Target: `a3c0cdb1748bb833f7d0be54d9148da66d452f94`; checkout lúc nhận việc trùng target, working tree sạch.
- Claim: F1/F2/F4 đã được đóng triệt để và F3/F5/F6 không hồi quy; full suite, native GUI build và staticcheck đạt.
- Authority: hợp đồng CEO hiện hành; bản vẽ `reports/problem/images/9180d29a.jpg`; đặc tả `reports/architect/2026-10-17-ui-redesign-spec-from-owner-drawing.md`; yêu cầu closure R2 tại `reports/supervisor/2026-10-18-ui-redesign-r2-supervisor-report.md`; Touch-Map Addendum `reports/architect/2026-10-18-touchmap-addendum-gpt-removal.md`.
- Đã đọc báo cáo Coder `reports/coder/2026-10-18-ui-redesign-f1-f2-f4-fix-report.md` làm đầu vào; kết luận dựa trên source và thực nghiệm R3. Không đánh giá cách diễn đạt hoặc thống kê trong báo cáo lane trước.
- Không tìm thấy `AGENTS.md` trong repository hoặc `G:\`; file `C:\Users\TAM NGUYEN\.codex\AGENTS.md` rỗng. Hợp đồng hiện hành xác định đầy đủ quyền kiểm định, scope và đầu ra.
- Bằng chứng mới: [command-evidence.md](evidence/2026-10-18-ui-redesign-r3/command-evidence.md). Toàn bộ probe và output R3 ở thư mục riêng; không chỉnh sửa bằng chứng lịch sử.

## Quyết định và ma trận nghiệm thu

**F2 đã đóng. F3/F5/F6 được bảo toàn tại ranh giới runtime Fyne đã được nghiệm thu ở R2. F1 và phần helper không dùng của F4 chưa đóng.** Các gate hệ thống bắt buộc đều đạt, nhưng không chứng minh được claim đóng toàn bộ ba finding.

| Invariant / gate | Kết quả R3 | Bằng chứng quyết định |
|---|---|---|
| F1: empty summary và fraction thiếu trong summary | Đạt phần này | Connect-RPC/REST `groups:[]`, fraction thiếu/null: bốn `--%`, `HasData=false`, giữ đúng qua pool/cache. E06/E07. |
| F1: presence và chu kỳ trên toàn pipeline/fallback | **Chưa đóng** | REST legacy vẫn dựng 100% hoặc 0%; weekly 0% theo `window` bị chuyển sang 5h; mất zero cycle sau cache; cache thiếu family còn tạo dữ liệu giả. E06/E07/E10. |
| F2: xóa GPT khỏi Go contract/callers | Đóng | Reflection không thấy field GPT; JSON không có khóa GPT; không còn `.GPT`/field literal trong `cmd`, `internal`. E06/E09. |
| F3: refresh riêng và sibling state | Bảo toàn | Bốn lần tap nút sản phẩm, kể cả nhánh lỗi: mỗi lần đúng một email, 5/5 sibling cùng con trỏ, focus và disabled state được giữ. E06. |
| F4: bốn object và countdown | Đạt phần này | Reflection: 0/4 field cũ; AST: không còn `NewQuotaCountdownLabel` hoặc `FormatRemainingCountdown`. E06, diff E00. |
| F4: helper retired không còn chức năng sản phẩm | **Chưa đóng** | `BuildActiveQuotaSummary` và các wrapper đã nêu ở R2 vẫn chỉ phục vụ test hoặc gọi lẫn nhau, không reachable từ GUI entrypoint. E08. |
| F5: compact ở hai viewport | Bảo toàn | Card **81.96875 px** ở 1440×900 và 1200×750; sweep 1/2/6/7/31 card, chiều cao tự nhiên và scroll đúng. E06. |
| F6: AS theo selector Sáng | Bảo toàn | Đi qua `widget.Select`: Sáng **#2563eb**, Ấm #d97706, Tối #89b4fa; đã xem PNG. E06. |
| Full suite | Đạt | **10/10 package**, **154 PASS = 135 top-level + 19 subcase**, **0 FAIL, 0 SKIP**, exit 0. E04. |
| Build workspace + native GUI | Đạt | Cả hai exit 0. E02/E03. |
| `staticcheck -tags gui ./...` | Đạt | **Exit 0, 0 diagnostic**. E05. |

## Blocking findings

### F1 — [P1] Presence vẫn bị suy diễn ở fallback; quota zero có thể sai chu kỳ hoặc mất qua cache

**Source:** `internal/telemetry/client.go:231`, `:239`, `:265`, `:381`, `:447`, `:691`; `internal/ui/quota_badge.go:109`, `:122`, `:136`; pipeline persistence tại `internal/telemetry/cache.go:128`, `:208`.

Các sửa đổi có tác dụng thực: summary parser dùng sentinel -1, REST nhận diện summary rỗng thay vì rơi sang legacy aggregate, UI không nhân fraction legacy nonzero thành weekly. Các case đầy đủ Claude 46%/12%, Gemini 88%/90%, empty summary, summary fraction thiếu/null, partial standard ID, zero 5h và cache legacy có đủ hai family đều đạt trong run này.

Tuy nhiên, cùng invariant vẫn có các đường thực thi sản phẩm sai sau đây. Các fixture đều synthetic, chạy HTTP loopback; cache được lưu/nạp bằng API sản phẩm. Ký hiệu tuple trong bảng: `(Claude5h, ClaudeWeek, Gemini5h, GeminiWeek)`; `-1` được badge format thành `--%`.

| Input / stage | Kỳ vọng | Thực tế R3 |
|---|---|---|
| REST legacy `{"buckets":[{"modelId":"claude-4-6","remainingAmount":37}]}` | Fraction chưa biết; `HasData=false` | **Claude 100% ở 5h**, `HasData=true`; giữ nguyên sau reload. E10. |
| REST legacy snake case `model_id:gemini-2-5`, `remaining_amount:37`, không fraction | Fraction chưa biết | **Gemini 100% ở 5h**, `HasData=true`; giữ nguyên sau reload. E10. |
| REST legacy `{"buckets":[{"modelId":"claude-4-6"}]}` | `(-1,-1,-1,-1)`, không data | RAM: **(0,0,-1,-1)**, `HasData=true`; sau reload vẫn giả Claude 5h 0%. E06. |
| REST legacy chỉ có Claude fraction 0, không cycle | Không suy ra weekly | RAM hiển thị **Claude 0% ở cả 5h và week**; sau reload weekly mới thành `--%`. E06. |
| Summary Claude `bucketId:opaque`, `window:weekly`, fraction 0, không reset | `(-1,0,-1,-1)` | Parser aggregate đúng weekly 0, nhưng badge trước và sau cache lại **(0,-1,-1,-1)**: quota bị chuyển từ week sang 5h. E06. |
| Connect-RPC thực qua `GetUserStatus` → `RetrieveUserQuotaSummary` → `PollAccount` → cache với fixture weekly zero trên | Cycle storage giữ `[-1,0,-1,-1]` | Trước lưu đúng; sau reload thành **[-1,-1,-1,-1]**. E07 test độc lập FAIL. |
| Legacy cache chỉ có `claude.remaining_fraction:0.2`, thiếu toàn bộ Gemini | Giữ Claude 20%, Gemini unknown | Badge mất Claude 20% và dựng **Gemini 0%/0%**, `HasData=true`. E06. |
| Legacy cache có Claude object nhưng không `remaining_fraction` | Không data | Claude bị gán **5h 0%**, `HasData=true`. E06. |

**Root cause đã đối soát:**

1. `rawBucketInfo.toModelQuotaBucket` khởi tạo `ModelQuotaBucket{}`. Khi thiếu fraction và amount, zero mặc định trở thành quota thật; khi có amount dương, code gán thẳng `RemainingFraction=1.0` dù không có tổng hạn ngạch để tính tỷ lệ (`client.go:382`, `:400`–`:412`). `AggregateBuckets` sao chép nguyên bucket, đưa cycle zero mặc định trở lại aggregate (`:447`, `:452`).
2. Heuristic presence trong `ExtractDualCycleQuota` chỉ có nhánh zero đặc biệt cho **FiveHourFraction == 0**, không đối xứng với WeeklyFraction == 0 (`quota_badge.go:109`–`:123`). Raw bucket không giữ trường `window`, nên opaque ID không cứu được weekly zero. Fallback lấy aggregate fraction và đặt vào 5h (`:136`–`:144`). Đây là sai dữ liệu tại badge, không chỉ khác storage format.
3. `FiveHourFraction`/`WeeklyFraction` vẫn có `omitempty`, nên zero hợp lệ bị bỏ khi marshal (`client.go:231`–`:232`); custom unmarshal mới đổi field vắng thành -1 (`:251`–`:259`). Probe RPC chứng minh sự mất presence này bằng dữ liệu trước/sau cache. Với ID chuẩn, UI còn cứu được bằng raw bucket; vì vậy không kết luận mọi quota zero đều hỏng.
4. Unmarshal mới chỉ default hai cycle field nếu **bucket object tồn tại**; chưa default `RemainingFraction` thiếu, và không khởi tạo unknown cho family vắng hoàn toàn trong `AccountQuotaData`. UI lại dùng một cờ `hasAnyBucket` chung cho cả hai family, nên zero giả của family vắng có thể chặn cả fallback của family có dữ liệu thật.

**Vì sao chặn:** hợp đồng yêu cầu fraction thiếu phải là unknown, `HasData` phải phản ánh dữ liệu thật và không được dựng chu kỳ thiếu. Các trường hợp trên vẫn tạo 100%/0%, tuần giả hoặc chuyển dữ liệu sang sai chu kỳ. Đây là các nhánh cùng F1 đang tồn tại trong code, không mở rộng sang miền khác.

**Hướng đóng:** bảo toàn presence từng family/cycle xuyên parser, fallback và cache; không tính phần trăm từ một amount không có denominator; nhận zero đối xứng; serialize zero có mặt tường minh; giữ cycle metadata cần thiết; phân biệt family/field vắng với giá trị 0 thật. Thay global fallback gate bằng xử lý không làm mất model có dữ liệu. Giữ các case summary/legacy đã đạt.

**Bằng chứng re-review:** tái chạy E06/E07/E10 hoặc regression tương đương ở HTTP → pool → cache reload → badge, có assertions cho missing/null/zero và model vắng. E06 matrix hiện có **41 quan sát, 33 khớp, 8 sai**; E10 bổ sung hai fixture amount, bốn quan sát sai. Các số này là inventory quan sát, không thay thế verdict theo invariant.

### F4 — [P2] Đã bỏ object/countdown, còn helper của UI retired và wrapper chỉ dùng bởi test

**Source:** `internal/ui/quota_badge.go:207`, `:215`, `:220`; `internal/ui/widgets.go:97`; `internal/ui/app.go:422`; các caller test trong E08.

Phần sửa object là đúng: `headerBg`, `statusBarLabel`, `activeStatusLabel`, `activeQuotaLabel` không còn trong `App`, không còn khởi tạo ngoài render tree hoặc cập nhật/recolor ngầm; hai hàm countdown đã bị xóa. Staticcheck bắt buộc đạt exit 0, 0 diagnostic. Không ghi nhận lại những lỗi đã được giải quyết này.

Phần helper cùng finding R2 vẫn chưa đóng:

| Hàm | Caller/ownership thực tế R3 |
|---|---|
| `BuildActiveQuotaSummary` | Helper cho active header đã nghỉ sử dụng; sau xóa `activeQuotaLabel`, chỉ còn caller trong `ui_test.go`. Không dùng để render card sản phẩm. |
| `NewQuotaBadgeGroup` | Comment trong source nêu rõ tương thích với test; chỉ có caller test, gọi `NewDualCycleQuotaBadge`. |
| `NewDualCycleQuotaBadge` | Chỉ reachable qua wrapper trên trong code sản phẩm; card thực gọi `ExtractDualCycleQuota` + `CreateQuotaCycleRow` trực tiếp. |
| `NewAccountCard` | Wrapper default palette/không quota chỉ có caller test; app dùng `NewAccountCardWithQuota`. |
| `App.RefreshSingleQuota` | Alias chỉ có caller test; callback trên card dùng `RefreshSingleAccountQuota`. |

E08 gồm tìm caller hiện hành và `deadcode` v0.52.0 từ `-tags gui ./cmd/switcher`; cả năm hàm đều được xác nhận không reachable từ GUI. **Exit 0 của deadcode chỉ xác nhận scan chạy được**, không có nghĩa không có candidate. Không dùng inventory ngoài UI làm finding, không yêu cầu xóa các seam injection/inspection như `SetTelemetryPool`, `SetPreferencesPath`, `Window` hoặc `IsAnyAccountRefreshing`.

Đây là các wrapper/helper cùng retired UI đã được R2 yêu cầu phân loại theo production contract, không phải yêu cầu dọn toàn repo. Đặc biệt `BuildActiveQuotaSummary` mất caller sản phẩm sau chính lần dọn object này. Test gọi helper cũ không chứng minh nó cần tồn tại trong GUI. `TestQuotaCountdownCalculationAndLabel` tại `ui_test.go:625` còn là một test không assertion; PASS của test này không được dùng làm bằng chứng countdown hoạt động hay cleanup đầy đủ.

**Vì sao chặn:** yêu cầu F4 gồm dọn stub và hàm không dùng, cùng yêu cầu closure R2 đối với wrapper retired. Việc dọn object đã đạt nhưng còn các helper không có chức năng runtime trong cùng phạm vi.

**Hướng đóng:** xóa các helper/alias retired và chuyển test sang API/render surface hiện hành; hoặc gắn helper cần giữ vào caller sản phẩm thực với ownership rõ ràng, bảo toàn bố cục ba dòng và invariant F3/F5/F6. Không xóa cơ học toàn bộ output deadcode. Chạy lại staticcheck/build/suite sau sửa.

## Các invariant đã đóng / bảo toàn

### F2: Go type và consumer sạch GPT

Reflection trên `AccountQuotaData` thực trong runtime: `GPT_field_present=false`. Marshal dữ liệu REST thực không có khóa `gpt`. Source `client.go:265` chỉ chứa Claude và Gemini trong contract family; phép gán REST đã bỏ GPT. Sweep `cmd`/`internal` cho `.GPT`, field declaration và literal `GPT:` có **0 match**, bao gồm caller CLI và test. Test reflection chính thức cũng chạy PASS.

CLI vẫn giữ cột GPT với giá trị mặc định N/A theo Addendum; không coi đây là hồi quy. Tên nhóm protocol upstream `Claude and GPT models`, return tuple parser legacy và `StatusLowGPT` không phải field trong contract thẻ; không yêu cầu xóa mù các chuỗi này. F2 đóng trong phạm vi được giao.

### F3: refresh một thẻ giữ sibling

Probe đi qua `test.Tap` trên nút refresh sản phẩm ở index 0, 2, 5 và nhánh lỗi index 3. Mỗi case chỉ poll đúng email target, thay đúng target card, giữ nguyên **5/5** sibling và trạng thái button disable/focus. Nhánh lỗi render `[!] Lỗi kết nối` ở hai hàng của riêng target. Source `app.go:399`–`:408` vẫn thay đúng `Objects[targetIdx]`, không dựng lại toàn grid cho single refresh.

Thời gian synthetic: callback 0 ms ở độ phân giải microsecond của môi trường này; hoàn tất **94.302–108.116 ms**, gồm fetcher cố ý đợi 50 ms. Không diễn giải thành độ trễ provider live.

### F5/F6: kích thước tự nhiên và theme

- Cả 1440×900 và 1200×750: mọi card giữ **81.96875 px** khi sweep 1/2/6/7/31 account. Grid sáu card lần lượt **1424×167.9375** và **1184×167.9375**. Probe resize cửa sổ và để layout sản phẩm chạy, không ép grid về min-height từ test.
- 31 account: grid cao **941.65625 px**, scroll đến đáy được; offset Y **180.8125** và **330.8125** tương ứng hai viewport.
- Selector Sáng → Ấm → Tối cập nhật monogram thật trong cây render; Sáng đúng **#2563eb**. Source giữ `monogramBg` và refresh palette tại `app.go:178`–`:180`.
- Đã xem trực tiếp [Dark](evidence/2026-10-18-ui-redesign-r3/E06-headless-dark.png) và [Light](evidence/2026-10-18-ui-redesign-r3/E06-headless-light.png): ba cột, ba hàng mỗi thẻ, card compact phía trên; Light có lỗi synthetic riêng account04 để ghi nhận nhánh F3.

## Source-level clearance notes

- **`cmd/switcher/main.go`: clear cho F2.** Bỏ đọc `q.GPT`, giữ format CLI như Addendum. `main_test.go` bỏ field literals đã bị xóa; package PASS.
- **`internal/telemetry/client.go`: F2 clear, F1 blocked.** Struct thực bỏ GPT; summary empty/missing đúng. Custom unmarshal, legacy raw conversion, zero serialization và family presence còn hở như F1.
- **`internal/ui/app.go`: F3/F5/F6 clear; F4 partial.** Đã xóa bốn field và background update; giữ target replacement, sizing và monogram. Còn alias/test-only helper inventory cùng F4.
- **`internal/ui/window.go`: clear cho thay đổi R3.** Bốn khởi tạo ngoài cây render đã xóa. `VScroll(Padded(VBox(grid)))` còn giữ card compact.
- **`internal/ui/quota_badge.go`: F1/F4 blocked.** Countdown đã bỏ, fallback legacy nonzero không còn nhân tuần. Heuristic weekly zero/global fallback và helper retired vẫn tồn tại.
- **Siblings `internal/ui/widgets.go`, `internal/telemetry/pool.go`, `internal/telemetry/cache.go`: đã trace.** Card dùng extractor/row thực; REST và Connect-RPC qua pool/cache sản phẩm; lỗi F1 không được che bằng mock parser hoặc skip persistence. Không sửa các file này.
- **Các test được sửa trong target:** suite chạy đầy đủ. Coverage hiện hành chưa chứng minh các nhánh F1 mới tái hiện; test gọi retired API không được dùng để chứng nhận production reachability. Không sửa test để thay đổi kết quả review.

## Evidence checked và freshness

**Passed — fresh trên target:**

- E00: đối soát `git show --no-ext-diff --format=fuller a3c0cdb`, đọc touched production sources và cùng-invariant consumer/cache; đã xem bản vẽ Owner.
- E01: Go **1.26.3**, Windows/amd64, CGO=1, gcc 16.1.0; staticcheck **2026.2.2 (0.8.2)**.
- E02: `go build ./...`, exit 0, **4854 ms**.
- E03: `go build -tags gui -ldflags '-H=windowsgui' -o .tmp/supervisor-ui-r3-gui.exe ./cmd/switcher`, exit 0, **8713 ms**. Binary validation tạm đã được xóa.
- E04: `go test -v -count=1 ./...`, exit 0, **30047 ms**, 10/10 package, **154 PASS**, 0 FAIL, 0 SKIP. Đã đọc log đầy đủ. Các package: cmd/switcher; connector, manager, model, reloader, scanner, store, telemetry, ui, wincred. Một số test có live discovery/read theo suite sẵn có, không có SKIP.
- E05: `staticcheck -tags gui ./...`, exit 0, **0 diagnostic**, **1982 ms**. Exit/diagnostic count được ghi trong `command-results.json`; stdout thành công rỗng.
- E06: F2 reflection/serialization, bốn field và hai hàm retired đã mất; F3/F5/F6 đạt, PNG được xem trực tiếp. Các case F1 đạt được phân biệt rõ với case sai.
- E07: actual Connect-RPC identity/header/summary/pool/cache: empty, missing fraction, full nonzero đều PASS.

**Failed — fresh:**

- E06 runtime probe exit **1**, **14718 ms**: hoàn tất toàn bộ quan sát và ghi JSON/PNG; exit 1 là các mismatch F1, không phải panic hoặc lỗi compile. Stderr `exit status 1` do `go run` chuyển tiếp trạng thái probe.
- E07 RPC overlay exit **1**: subtest `window_weekly_zero` chứng minh zero cycle mất sau cache; ba subtest khác đạt.
- E10 targeted fallback exit **1**: amount không có fraction vẫn hiển thị 100% qua HTTP → pool → disk reload → badge ở cả camel/snake case.
- F4 closure: source/caller inventory và E08 reachability còn năm helper/wrapper retired. Staticcheck vẫn PASS; hai kết quả này không mâu thuẫn.

**Not run / giới hạn:**

- Không launch/capture native desktop hoặc kiểm thử OS hover/minimize/resize trong run này. PNG là **software canvas Fyne test driver**, không phải screenshot Windows. Native GUI **build** đã đạt. F3/F5/F6 được bảo toàn tại cùng ranh giới Fyne đã nêu trong R2; không chứng nhận thêm phạm vi native OS.
- Probe RPC seed discovery cache của pool synthetic để trỏ loopback, còn method/CSRF/GetUserStatus/summary/parser/cache là code sản phẩm. Không dùng probe này để chứng nhận discovery Language Server live.
- Không đổi tài khoản Owner, không đo quota-provider live latency. Không chạy lại full suite/build sau khi chỉ viết thêm evidence build-excluded và báo cáo; không có sửa đổi sản phẩm cần regression lần nữa.

## Invariant closure và resubmission

Đã sweep REST/Connect-RPC → raw/summary parser → pool → cache save/reload → badge/card; Go contract → REST construction → CLI/test consumers; single refresh → sibling object/state; root → VScroll/grid/card; selector → palette/monogram; retired fields/helpers → production callers → GUI reachability.

Residual trong scope: **F1 presence/fallback/zero/cache và F4 helper retired**. F2/F3/F5/F6 không còn residual trong ranh giới kiểm định đã mô tả.

1. Đóng F1 ở toàn bộ sibling path đã tái hiện, bao gồm legacy REST amount/missing fraction, weekly zero theo `window`, lưu/nạp zero, cache thiếu field/family; giữ các case empty/full/legacy đã đạt.
2. Dọn hoặc dùng thực các helper retired còn lại của F4 và chuyển test sang surface sản phẩm; không xóa các test seam quan sát/injection cần thiết.
3. Sau sửa, chạy full suite 10/10 không SKIP, native GUI build, staticcheck 0 diagnostic và probe closure mới; hồi quy F3/F5/F6 tại các boundary bị ảnh hưởng. Giữ nguyên bằng chứng R3 này.

**Đánh giá tổng thể:** bản sửa đã đóng F2, xử lý đúng summary empty/missing và dọn các object/countdown cụ thể. F1 vẫn tạo dữ liệu không có trong input hoặc đặt quota sai chu kỳ; F4 còn code retired được test giữ sống. Vì vậy **REJECT** claim hoàn tất F1/F2/F4; Supervisor bàn giao về CEO, không sửa sản phẩm.
