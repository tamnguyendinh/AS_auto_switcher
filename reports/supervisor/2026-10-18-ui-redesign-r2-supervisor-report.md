# Báo cáo Supervisor R2: kiểm định bản sửa Fyne GUI F1–F6

**Verdict: REJECT**

## Metadata, claim và thẩm quyền

- Thời điểm nghiệm thu: 2026-10-18, khoảng 14:35–14:51, Asia/Bangkok (UTC+07).
- Reviewer: Codex, vai trò Supervisor độc lập; áp dụng `supervisor`, `debugging` và working rules. Không dùng Anvien, không mở subagent, không sửa mã sản phẩm.
- Commit được nghiệm thu: `6b5cfe3037b114bf92782f313658008b034f45af`. HEAD lúc nhận việc trùng target; working tree ban đầu sạch.
- Claim: sáu finding F1–F6 của vòng 1 đã được giải quyết triệt để; quota đúng theo hai chu kỳ, contract thẻ bỏ GPT, refresh giữ sibling, dọn dead UI, thẻ compact và monogram đổi màu; test/staticcheck/build đạt.
- Authority: hợp đồng CEO hiện hành; bản vẽ `reports/problem/images/9180d29a.jpg`; đặc tả `reports/architect/2026-10-17-ui-redesign-spec-from-owner-drawing.md`; yêu cầu đóng invariant trong `reports/supervisor/2026-10-18-ui-redesign-supervisor-report.md` (vòng 1, `fe8ffbb`). Báo cáo Coder `reports/coder/2026-10-18-ui-redesign-f1-f6-fix-report.md` là claim đầu vào, không dùng làm bằng chứng kết luận.
- Không có `AGENTS.md` trong checkout hoặc `G:\`; `C:\Users\TAM NGUYEN\.codex\AGENTS.md` tồn tại nhưng rỗng. Đã đọc đầy đủ skill bắt buộc tại `.agents/skills/supervisor/SKILL.md` và references. Hợp đồng hiện hành xác định đầy đủ scope và quyền kiểm định; không phát sinh yêu cầu hỏi Owner.
- Evidence hiện hành: [evidence/2026-10-18-ui-redesign-r2](evidence/2026-10-18-ui-redesign-r2/command-evidence.md). Diff, output command, JSON probe và PNG đều do Supervisor tạo trong run này. Không chạy lại probe vòng 1 vì nó ghi đè bằng chứng cũ.

## Quyết định

**F3, F5, F6 đóng trong phạm vi kiểm định. F1, F2 và phần dọn legacy UI của F4 chưa đóng.** Các gate build, native GUI build, test và staticcheck đều đạt. Tuy nhiên, pipeline vẫn có thể tạo dữ liệu quota không có trong response, trường GPT vẫn nằm trong contract thẻ và mã UI đã nghỉ sử dụng vẫn còn. Vì vậy chưa thể chấp nhận bản sửa là hoàn tất F1–F6.

| Invariant | Kết quả | Bằng chứng quyết định |
|---|---|---|
| F1: độc lập 5h/week và không dựng quota cho dữ liệu thiếu | **Chưa đóng** | Payload đầy đủ đúng; response rỗng, fraction thiếu và `window` với quota 0% còn sai; cache cũ vẫn nhân một fraction cho hai chu kỳ. E06, E07. |
| F2: bỏ GPT khỏi contract và serialization của thẻ | **Chưa đóng** | JSON không còn khóa `gpt`, nhưng reflection vẫn tìm thấy field `GPT`; REST vẫn gán 0.12 vào field này. E06 `F2_contract`. |
| F3: refresh riêng giữ năm sibling và trạng thái tương tác | **Đóng** | Nút thực tại index 0, 2, 5 và lỗi tại index 3: mỗi lần query đúng email, giữ 5/5 sibling cùng disable/focus. E06. |
| F4: staticcheck sạch và dọn code UI cũ | **Đóng gate staticcheck, chưa đóng cleanup** | Exit 0, 0 diagnostic; còn countdown stub/helper và bốn object được tạo nhưng không gắn cây render. E05, E08, E09. |
| F5: thẻ compact ở 1440x900 | **Đóng tại Fyne layout/render boundary** | Sáu thẻ: card 81.96875 px, grid 167.9375 px; sweep 1/2/6/7/31 thẻ, hai viewport đều giữ chiều cao tự nhiên. E06. |
| F6: selector Sáng đổi màu AS thành `#2563eb` | **Đóng tại Fyne selector/render boundary** | Sáng → Ấm → Tối đều theo palette hiện hành; Sáng đúng `#2563eb`, đã xem PNG. E06. |
| Test toàn hệ thống | **Đạt** | 10/10 package; 153 PASS, 0 FAIL, 0 SKIP; exit 0. E04. |
| Build toàn workspace và native GUI | **Đạt** | Cả hai exit 0; native build dùng `-tags gui -ldflags '-H=windowsgui'`. E02, E03. |

## Blocking findings còn lại

### F1 — [P1] Presence của quota chưa được bảo toàn; nhánh fallback tiếp tục tạo hai chu kỳ giả

**Source:** `internal/telemetry/client.go:408`, `:543`, `:556`, `:623`, `:633`; `internal/ui/quota_badge.go:130`, `:142`, `:155`; `internal/telemetry/cache.go:210`.

Đường dữ liệu đầy đủ đã được sửa đúng. Probe HTTP thật trên loopback chạy qua `GetUserStatus` → `RetrieveUserQuotaSummary` → `TelemetryPool.PollAccount` → ghi/nạp lại cache, xác nhận Claude 46%/12%, Gemini 88%/90%, đủ bốn raw bucket và hai mốc reset riêng. REST summary qua client/pool/cache đến badge cũng đúng. Việc vẫn tính min fraction cho trạng thái tổng hợp không còn làm mất bốn giá trị ở case này.

Tuy nhiên, closure của vòng 1 còn yêu cầu dữ liệu thiếu không được giả thành quota đầy. E06 tái hiện:

| Input đi qua parser/client sản phẩm | Kỳ vọng | Badge thực tế |
|---|---|---|
| Connect-RPC `{"response":{"groups":[]}}` | Không có quota; placeholder hoặc lỗi | Cả bốn giá trị **100%**, `HasData=true`, `Status=Pristine`. |
| REST `{"groups":[]}` qua HTTP → pool | Không có quota; placeholder hoặc lỗi | Cả bốn giá trị **100%**, `HasData=true`, `Status=Pristine`. |
| Nhóm Claude, `bucketId:"opaque"`, `window:"5h"`, fraction `0`, không reset; không có Gemini/week | Claude 5h 0%; ba giá trị còn lại chưa biết | Claude **0%/0%**, Gemini **100%/100%**. |
| `bucketId:"3p-5h"`, `window:"5h"`, thiếu `remainingFraction` | Fraction chưa biết | Claude 5h **100%**, `HasData=true`. |
| Nạp cache cũ chỉ có Claude 0.2, Gemini 0.8, không raw/cycle fields | Không có bằng chứng tuần | Claude **20%/20%**, Gemini **80%/80%**. |

Nguyên nhân được xác nhận ở source:

1. Summary parser khởi tạo aggregate ở 1.0 và mặc định `frac := 1.0` khi field không có; thiếu dữ liệu vẫn được đánh dấu như quota thật.
2. REST chỉ chấp nhận summary khi có raw bucket. Summary rỗng rơi sang legacy parser, `AggregateBuckets` khởi tạo hai model ở 1.0.
3. UI nhận diện cycle fields bằng `> 0` hoặc có reset. Quota **0%** là dữ liệu hợp lệ nhưng không thỏa heuristic đó khi reset thiếu. Nếu bucket ID không chứa model/chu kỳ, dù `window` đã được parser hiểu đúng, UI lại rơi sang fallback.
4. Fallback `quota_badge.go:155` sao chép `RemainingFraction` sang cả 5h và week; cache cũ được nạp nguyên trạng nên vẫn đi qua đường này.

Case chuẩn có fraction khác nhau, partial bucket có ID chuẩn, toàn bộ quota 0%, và `window` với opaque ID nhưng fraction dương đều đạt trong E06. Kết luận lỗi dựa trên những case sai nêu trên, không phủ nhận các case đã sửa đúng.

**Hướng đóng:** biểu diễn presence của từng model/chu kỳ tường minh, nhận đúng zero, không suy ra tuần từ fraction legacy; giữ unknown/error xuyên parser → pool → cache → UI. Nhận diện summary rỗng đúng shape và không biến thành legacy quota đầy; xử lý fraction thiếu bằng unknown/error. Định nghĩa migration/invalidation cache cũ và kiểm thử tại pipeline sản phẩm với các input trên.

### F2 — [P2] GPT chỉ bị ẩn khi marshal, chưa bị xóa khỏi contract thẻ

**Source:** `internal/telemetry/client.go:238`, `:242`, `:548`, `:573`, `:612`, `:677`; `internal/ui/widgets.go:107`.

Card vẫn nhận `*telemetry.AccountQuotaData`; struct này vẫn có `GPT ModelQuotaBucket`, chỉ đổi tag thành `json:"-"`. Parser vẫn tạo quota GPT, REST client vẫn gán `GPT: gpt` ở cả nhánh summary và legacy. Reflection trong E06 xác nhận `GPT_field_present=true`, tag `-`; REST fixture còn gán `GPT.RemainingFraction=0.12`.

Phần serialization đã đạt: JSON không còn khóa `gpt`. Nhánh pool Connect-RPC cũng đã bỏ phép gán GPT vào `AccountQuotaData`. Nhưng hợp đồng hiện hành yêu cầu xóa trường hoàn toàn khỏi contract dữ liệu thẻ, không chỉ ẩn JSON. Do đó F2 chưa đóng.

**Sibling cần xử lý khi sửa:** `cmd/switcher/main.go:306` và `:315` vẫn đọc `q.GPT` từ cùng kiểu dữ liệu, trong khi cache mới bỏ serialize field đó. Phải xác định ranh giới contract/consumer khi loại field, tránh giữ nó trong card schema chỉ để duy trì consumer cũ. Không mở rộng thành đánh giá toàn bộ CLI trong run này. Chuỗi tên nhóm upstream `Claude and GPT models` vẫn cần được nhận diện để map quota Claude; không yêu cầu xóa mù các tên protocol này.

**Hướng đóng:** xóa `GPT` khỏi kiểu dữ liệu card/telemetry đã chỉ định và các phép gán/call contract liên quan; cập nhật consumer cùng invariant hoặc tách contract phù hợp với phạm vi được CEO giao. Bằng chứng resubmission cần kiểm tra cả Go type/reflection và JSON/cache, không chỉ tìm khóa `gpt` trong JSON.

### F4 — [P2] Gate staticcheck đã sạch nhưng legacy UI cleanup của vòng 1 vẫn còn

**Source:** `internal/ui/window.go:175`; `internal/ui/app.go:186`, `:718`, `:788`; `internal/ui/quota_badge.go:36`, `:234`, `:239`; `internal/ui/widgets.go:97`.

Bốn diagnostic cụ thể của vòng 1 đã được xử lý đúng: `saveBtn`, `headerCard`, `hasGUI` đã bỏ, `err` từ `io.ReadAll` được kiểm tra và đưa vào fallback/error path. Lệnh bắt buộc `staticcheck -tags gui ./...` trả **exit 0, 0 diagnostic**. Không ghi nhận staticcheck thất bại.

Tuy nhiên, cùng finding F4 của vòng 1 còn yêu cầu loại retired UI objects/stubs và các test cũ giữ chúng sống giả. Những bề mặt này chưa được dọn:

- `NewQuotaCountdownLabel` vẫn là hàm luôn trả `nil`; `FormatRemainingCountdown` chỉ còn caller trong test.
- Wrapper `NewQuotaBadgeGroup`, `NewAccountCard` và alias `RefreshSingleQuota` không được GUI entrypoint sử dụng. `NewDualCycleQuotaBadge` cũng chưa được production card dùng; card dựng hai hàng trực tiếp.
- `headerBg`, `statusBarLabel`, `activeStatusLabel`, `activeQuotaLabel` vẫn được tạo. Probe E09 xác nhận cả bốn **initialized=true, attached_to_render_tree=false**. `SetTheme`, callback và `updateUI` tiếp tục recolor/update những object không có trong cửa sổ.

`deadcode` v0.52.0 từ GUI entrypoint phát ra 62 candidate (16 trong UI). **Exit 0 của công cụ này có nghĩa scan chạy thành công, không có nghĩa không có candidate.** Đã phân loại các API injection/inspection như `SetTelemetryPool`, `SetPreferencesPath`, `SetQuotaCache`, `Window`, `IsAnyAccountRefreshing` là seam phục vụ kiểm thử/quan sát, không tự động kết luận chúng phải xóa. Các candidate ngoài UI chỉ là inventory, không phải danh sách 62 lỗi cần sửa trong scope này. Phần chặn ở đây là retired UI cụ thể nêu trên, đã có từ vòng 1 và được source/runtime xác nhận.

**Hướng đóng:** dọn countdown/stub, legacy object và background update không còn chức năng hiển thị; chuyển test cũ sang bề mặt UI thực. Phân loại wrapper nào được giữ với production caller/contract thực, wrapper nào bỏ hoặc dùng lại trong card. Giữ các seam cần thiết, không xóa cơ học toàn bộ reachability output. Duy trì staticcheck/test/build sạch sau khi sửa.

## Invariant đã đóng và benchmark

### F3: refresh đúng một thẻ

`app.go:394`–`:430` tìm item theo email và thay đúng `Objects[targetIdx]`, không gọi `updateUI` toàn grid trong nhánh single refresh. E06 dùng `test.Tap` lên button sản phẩm, lần lượt account01/account03/account06 và lỗi account04. Mỗi case chỉ có một email được polled, target card thay đổi, năm sibling giữ nguyên con trỏ; button sibling bị disable và focus trên sibling vẫn giữ nguyên. Nhánh lỗi hiện `[!] Lỗi kết nối` ở cả hai hàng của riêng target.

Callback đo được 0 ms ở độ phân giải microsecond của môi trường này; hoàn tất I/O/render synthetic khoảng **153.742–162.786 ms**, với fetcher cố ý chờ 50 ms. Đây không phải benchmark provider live hoặc cam kết tổng thời gian ~50 ms.

### F5: layout tự nhiên, cuộn và hàng thiếu cột

`window.go:172` bọc grid trong VBox trước VScroll; `app.go:767` resize grid theo min-height. Probe không ép resize grid về min-height như unit test Coder; nó resize cửa sổ, để cây layout sản phẩm chạy và capture canvas rồi đo.

- Ở 1440x900, sáu card: mỗi card **81.96875 px**, grid **1424 × 167.9375**, đúng min-height. Không còn grid 760 px/card 378 px của vòng 1.
- Ở 1200x750, sáu card: mỗi card **81.96875 px**, grid **1184 × 167.9375**.
- Sweep 1/2/7/31 card ở cả hai viewport: mọi card giữ **81.96875 px**, vị trí theo ba cột, hàng cuối thiếu cột không bị kéo cao. Grid lần lượt có chiều cao tự nhiên 81.96875/253.90625/941.65625 px.
- 31 card cuộn đến đáy được: offset Y **180.8125** ở 1440x900 và **330.8125** ở 1200x750.

Đã xem trực tiếp [Dark](evidence/2026-10-18-ui-redesign-r2/E06-headless-dark.png) và [Light](evidence/2026-10-18-ui-redesign-r2/E06-headless-light.png): ba cột, mỗi card ba hàng compact ở phần trên. Light giữ riêng trạng thái lỗi synthetic của account04 để chứng minh nhánh F3.

### F6: selector truyền màu đến monogram

`window.go:86` giữ rectangle trên `App`; `app.go:182` cập nhật `FillColor` và refresh khi `SetTheme` chạy. Probe đi qua `widget.Select.SetSelected`, không gọi trực tiếp `App.SetTheme`: Sáng **#2563eb**, Ấm **#d97706**, Tối **#89b4fa**, đều khớp palette đang được ứng dụng dùng. Tiêu chuẩn F6 về Sáng đạt; không dùng kết quả này để chứng nhận toàn bộ bảng màu của đặc tả cũ.

## Source-level clearance notes

- **`cmd/switcher/app_gui.go`: clear cho phần sửa F4.** Bỏ `hasGUI`; native GUI entrypoint build thành công.
- **`internal/connector/api_client.go`: clear cho phần sửa F4.** Read error tại `:115` được xử lý trước parse/status, tiếp tục candidate endpoint và giữ `lastErr`; regression package connector PASS.
- **`internal/telemetry/client.go`: blocked F1/F2.** Endpoint summary và cycle fields đúng ở dữ liệu đầy đủ; còn mặc định quota 1.0, fallback rỗng, field/assignment GPT.
- **`internal/telemetry/pool.go`: clear cho wiring mới, F1 toàn invariant chưa đóng.** HTTP Connect-RPC giữ raw buckets và cycle/reset fields; cache không làm mất dữ liệu đầy đủ. Parser/fallback/cache cũ vẫn còn các lỗi F1.
- **`internal/ui/app.go`: clear F3/F6 và sizing F5; blocked cleanup F4.** Single refresh giữ sibling; theme cập nhật AS; còn legacy state/widget update không render.
- **`internal/ui/window.go`: clear F5/F6; blocked cleanup F4.** VBox/VScroll giữ chiều cao tự nhiên, reference monogram đúng; bốn object legacy vẫn được khởi tạo ngoài cây render.
- **`internal/ui/quota_badge.go`: blocked F1/F4.** Render bốn giá trị đúng khi dữ liệu đầy đủ; presence heuristic và legacy fallback sai ở case thiếu/zero; countdown/wrapper cũ còn.
- **Sibling `internal/ui/widgets.go`, `internal/telemetry/cache.go`, CLI quota consumer:** đã trace cùng invariant. Card chỉ vẽ Claude/Gemini nhưng nhận contract còn GPT; cache nạp legacy không migration và JSON bỏ GPT trong khi consumer cũ còn đọc field.
- **Tests được sửa (`internal/telemetry/quota_test.go`, `internal/ui/ui_test.go`): chạy PASS, coverage chưa đủ F1/F2/F4.** F2 test chỉ marshal JSON; F1 test dùng partial fraction dương/ID chuẩn; F5 unit test tự ép grid về min-height. Probe độc lập bổ sung pipeline HTTP, dữ liệu thiếu/zero, real button, tự layout và selector, không sửa test sản phẩm để đạt kết quả.

## Evidence checked

**Passed / fresh:**

- E00: `git show --no-ext-diff --format=fuller 6b5cfe3037b114bf92782f313658008b034f45af -- cmd internal`; đối soát diff target và đọc source hiện hành.
- E01: Go **1.26.3**, Windows/amd64, CGO=1, gcc; staticcheck **2026.2.2 (0.8.2)**.
- E02: `go build ./...`, exit 0, 5747 ms.
- E03: `go build -tags gui -ldflags '-H=windowsgui' -o .tmp/supervisor-ui-redesign-r2-gui.exe ./cmd/switcher`, exit 0, 8004 ms. Binary validation tạm đã được xóa sau khi build; log được giữ.
- E04: `go test -v -count=1 ./...`, exit 0, 26439 ms, **10/10 package**, **134 top-level + 19 subcase = 153 PASS**, **0 FAIL, 0 SKIP**. Các package: cmd/switcher; internal/connector, manager, model, reloader, scanner, store, telemetry, ui, wincred. Đã đọc log đầy đủ; có live discovery/read tests chạy, không SKIP. Một số output chứa thông tin tài khoản test/live sẵn có, không có token.
- E05: `staticcheck -tags gui ./...`, exit 0, **0 diagnostic**, 4051 ms.
- E06: `runtime_probe.go`, exit 0, 35689 ms; JSON và hai PNG đã đọc/xem. F1 happy path, F3/F5/F6 đạt như mô tả; stderr rỗng. Exit 0 chỉ xác nhận probe ghi nhận đủ quan sát, không đồng nghĩa mọi expected match.
- E07: probe bổ sung bằng Go overlay, `TestSupervisorR2ActualRPCPoolCache`, PASS. HTTP loopback thật dùng method/header CSRF sản phẩm, GetUserStatus và quota summary thật qua client/pool; discovery cache được seed bằng server synthetic. Không giả vờ đã kiểm thử discovery của Language Server live trong probe này. Không sửa file sản phẩm; overlay JSON tạm đã xóa.

**Failed / fresh:**

- F1: presence/missing/legacy cases trong E06 có `matches_expected=false`, badge sai số liệu như bảng reproduction.
- F2: field GPT còn ở Go type, còn được REST gán; JSON exclusion là phần đạt, không đủ đóng toàn claim.
- F4 cleanup: E08 và E09 chứng minh retired functions/object state vẫn tồn tại. Staticcheck vẫn được ghi nhận là đạt.

**Not run / giới hạn bằng chứng:**

- Không launch GUI native để xác minh pixel/OS interaction trong run này. PNG là **software canvas của Fyne**, không phải screenshot desktop native. Native GUI **build** đã đạt theo hợp đồng hiện hành.
- Không hot-swap tài khoản Owner, không đo live quota-provider latency, không chứng nhận native hover/minimize/OS resize. Không dùng kết quả headless để điền các khoảng trống native của vòng 1.
- Không chạy lại test/build sau khi chỉ thêm evidence build-excluded và báo cáo; không có sửa đổi sản phẩm cần regression lần hai.

## Invariant closure và yêu cầu resubmission

Đã sweep các bề mặt cùng invariant: HTTP REST/Connect-RPC → parser → pool → cache disk reload → card/badge; button → single refresh → sibling object/state; root → VScroll → grid/card; selector → SetTheme → monogram; GUI entrypoint → staticcheck/reachability → legacy widget attachment. F3/F5/F6 đạt ở ranh giới đã nêu. Residual chưa đóng là F1/F2/F4; giới hạn native được ghi tường minh, không suy diễn thành lỗi sản phẩm mới.

1. **F1:** sửa presence/unknown/zero và fallback summary rỗng; loại suy diễn weekly từ cache legacy. Cung cấp regression cho các input E06 ở cả parser và pipeline cache/UI, giữ các case dữ liệu đầy đủ đã đạt.
2. **F2:** xóa field GPT khỏi card contract và các phép gán liên quan; giải quyết consumer/cache cùng contract theo scope CEO giao. Kiểm tra Go type lẫn JSON/cache, giữ mapping tên nhóm upstream cần thiết.
3. **F4:** hoàn tất cleanup retired UI/stub/update và test giữ chúng sống; phân loại test seams/wrappers theo production ownership. Không xóa máy móc toàn bộ candidate ngoài scope. Giữ gate staticcheck 0 diagnostic.
4. Sau sửa: chạy command test bắt buộc 10/10 không SKIP, native GUI build, staticcheck và probe các invariant còn reject; hồi quy F3/F5/F6 tại ranh giới bị ảnh hưởng. Native visual/interaction evidence vẫn phải được cung cấp nếu CEO yêu cầu claim nghiệm thu toàn bộ desktop ngoài scope build/probe hiện hành.

**Đánh giá tổng thể:** bản sửa đã giải quyết đúng nhiều lỗi thực, nhưng chưa bảo toàn đầy đủ tính trung thực của quota, chưa xóa GPT khỏi contract và chưa hoàn tất retired UI cleanup. **REJECT**; CEO chuyển packet này cho lane sửa trong ba invariant F1/F2/F4, Supervisor không thực hiện sửa mã sản phẩm.
