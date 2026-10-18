# Supervisor R2: nghiệm thu sửa F1–F5 của quy trình hotswap 4 bước

**Verdict: REJECT**

## Metadata và thẩm quyền

- Ngày kiểm định: 2026-10-18, evidence runtime hoàn thành khoảng 09:27, Asia/Bangkok (UTC+07:00).
- Vai trò: Supervisor độc lập; địa chỉ nhận handoff CEO: `01a14378-4a08-77c1-88d5-666ad8c3592d`.
- Commit sản phẩm: `1c9e560e1bc004da7fa6eabbc33870d63f12c0b2`; HEAD lúc bắt đầu kiểm định trùng commit này, cây làm việc ban đầu sạch.
- Claim được kiểm định: các invariant F1–F5 từ báo cáo Supervisor commit `35138bb` đã được khép; đặc biệt mọi trường hợp không có client RAM sau recycle phải trả thất bại, không continuation; project được đọc đúng qua SQLite, UUID được lọc, wrapper được tách cấu hình và không còn dead code.
- Authority: hợp đồng R2 của CEO, báo cáo Supervisor R1, đặc tả kỹ thuật đọc trực tiếp tại commit `1ec2e6a`. Báo cáo Coder hiện tại là đầu vào kiểm chứng, không phải bằng chứng tự đủ.
- Đã đọc skill supervisor tại đường dẫn repo theo hợp đồng, các reference về claim/evidence/invariant/reporting, working-rules, debugging và databases, cùng quy ước Go tests. Không tìm thấy `AGENTS.md` trong checkout hoặc `G:\AGENTS.md`; tiếp tục theo hợp đồng và đặc tả đã xác định rõ phạm vi. Debugging dùng bản khả dụng trong `C:\Users\TAM NGUYEN\.agents\skills\debugging\SKILL.md`.
- Không dùng Anvien, không mở subagent, không sửa source/test hiện hữu hoặc dependency files. Output thuộc lane R2 chỉ nằm trong `reports/supervisor/`.

## Quyết định nghiệm thu

Không thể chấp nhận claim sửa xong F1–F5. **F1 còn một nhánh trả thành công dựa vào WinCred khi `newClient == nil`; F5 còn dead code trong chính package reloader**, ngoài một diagnostic CLI đã có từ R1.

F2 đã sửa đúng lỗi gán nhầm project từ preview: truy vấn SQLite chọn đúng record/cột, được kiểm chứng bằng database thật và committed WAL. F3 đã loại `index.db` và file không phải UUID; F4 không còn dùng `StoragePath` làm thư mục conversations. Chờ 8 giây, xác nhận RAM trên đường client hợp lệ, đọc lại đúng snapshot disk và continuation tuần tự tiếp tục sau lỗi thread đầu đều có bằng chứng mới đạt.

Lệnh bắt buộc `go test -v -count=1 ./...` **exit 0 trên đủ 10 package: 144 PASS events, 0 FAIL, 0 SKIP**. Một reproduction độc lập vẫn thất bại ở nhánh nil client chưa được suite bao phủ; vì vậy suite xanh không đóng được invariant F1.

## Đối soát diff `git show 1c9e560` và source thực thi

Đã kiểm tra toàn bộ diff sản phẩm/test của commit, gồm `account_manager.go`, `account_manager_test.go`, `manager_test.go`, `active_threads.go`, `active_threads_test.go`, `session_reloader.go`, `go.mod` và `go.sum`. Đã đọc source các caller/callee cùng invariant, không chỉ dựa vào tên test hoặc thông điệp commit.

| Finding / yêu cầu | Thay đổi và bằng chứng mới | Trạng thái |
|---|---|---|
| F1: RAM gate trước continuation/thành công | `account_manager.go:557-578` gọi GetUserStatus và normalize/so sánh email; target đúng, stale, HTTP error và empty email đã chạy độc lập | Đã đóng các đường có client hợp lệ; chưa đóng nhánh nil client bên dưới |
| F1: `newClient == nil` luôn thất bại | `account_manager.go:519-553` có exception `:531-542` trả `Success=true` qua WinCred khi finder trả `(nil,nil)` | **Chưa đạt**, blocking R2-F1 |
| F2: đúng `workspace_uris` theo `conversation_id`, bỏ quét binary 2.048 byte | `active_threads.go:185-205` dùng `database/sql`, driver modernc, DSN `mode=ro`, parameterized `SELECT workspace_uris ... WHERE conversation_id = ? LIMIT 1` | Lỗi gốc đã đóng; preview độc hại, hai project, IDLE, committed WAL và absent ID đã kiểm chứng |
| F3: regex UUID 36 ký tự cho DB/WAL | `active_threads.go:36,100-113` kiểm tra cùng một `reUUID` sau khi tách suffix | Đã đóng lỗi gốc; file `index.db` bị loại; suite còn kiểm tra backup WAL không phải UUID |
| F4: wrapper độc lập với ToS StoragePath | `session_reloader.go:48-54,401-427` dùng ConversationsDir/SummariesDbPath mới; default scanner `active_threads.go:73-80` resolve đúng profile | Đã đóng lỗi gốc; reproduction R1 với chỉ StoragePath và conversations mặc định nay đạt |
| F5: xóa `isFileViewTab` | Diff xóa helper; tra cứu source không còn declaration/caller | Đã đóng việc xóa helper này |
| F5: **0 dead code** | Production-only U1000 báo 6 declarations trong reloader; full-workspace U1000 báo `hasGUI` | **Chưa đạt**, blocking R2-F5 |
| Giữ MAX mtime và ngưỡng inclusive -2h | Scanner `active_threads.go:135-162`; reproduction R1 chạy lại trên commit mới | Đạt: đúng -2h được chọn; DB -3h được WAL mới cứu; -2h-1s bị loại; sort/main pane đúng |
| Giữ chờ 8s → discovery → RAM → đúng snapshot disk | `account_manager.go:488-578,584-599`; trace độc lập | Đạt trên đường thành công; mock sleeper nhận đúng `[8s]` |
| Giữ RPC, model, pacing và lỗi từng thread | `continuation.go:84-121`, không thay đổi trong commit; reproduction default HTTP runner chạy mới | Đạt: SmartFocus → LoadTrajectory → Send; hai sleep 3s; active model đúng; thread đầu lỗi không dừng thread 2/3; summary 3/2/1 |
| 10 package test không SKIP | Full suite uncached đã chạy và đọc toàn log | Đạt: 144 PASS events, 0 FAIL/SKIP, exit 0 |

`go.mod`/`go.sum` bổ sung driver SQLite cùng dependencies; build/test sử dụng được driver đăng ký mới. Không sửa schema/database của Antigravity để lấy kết quả. Việc đọc read-only nguồn do ứng dụng khác sở hữu không yêu cầu đổi PRAGMA/schema của nguồn đó trong lượt review này.

## Blocking findings còn lại

### R2-F1 — [P1] Nil client vẫn có thể trả thành công bằng credential trên disk

**Source:** `internal/manager/account_manager.go:519-542`, đặc biệt `:531-536`.

Code xử lý `newClient == nil` bằng hai đường khác nhau:

- Finder trả error: trả failure, đúng yêu cầu.
- Finder không trả error: đọc lại `m.wincredReader`; nếu credential có target email thì trả `Success=true`, `err=nil`, trước cả GetUserStatus và bước đọc lại snapshot.

Comment gọi đây là trường hợp mock/test harness, nhưng nhánh nằm trong source sản phẩm, không có build guard dành riêng cho test. `SetClientFinder` và `SetWincredHandlers` là seams có thể gọi của manager. Hợp đồng R2 không có ngoại lệ cho `(nil,nil)`: **nếu `newClient == nil` phải `Success=false`**. Credential đã ghi thành công không chứng minh identity của Language Server RAM.

**Reproduction độc lập:** `TestR2RAMGateEveryOutcome/nil-client-without-error`:

1. Trước switch, HTTP server trả RAM `old@r2.invalid`, credential cũng là old, nên không đi vào no-op.
2. Có snapshot một thread hợp lệ để continuation không bị bỏ qua do danh sách trống.
3. Writer mô phỏng ghi token target và cập nhật credential reader; killer và discoverer thành công; sleeper nhận đúng `8s`.
4. Sau recycle, finder trả `(nil,nil)`.

Trace thực tế:

```text
scenario=nil-client-without-error
success=true
error=<nil>
credential_email=target@r2.invalid
status_after_kill=0
continued=0
finder_after_kill=1
waits=[8s]
```

Đã bảo vệ continuation trong trường hợp này, nhưng vẫn báo đổi account thành công khi RAM chưa xác nhận và bỏ qua bước 3/4. Lỗi F1 do đó chỉ được sửa một phần. Không suy diễn rằng default finder luôn trả `(nil,nil)`; reproduction chứng minh chính đường được hợp đồng yêu cầu và được source hỗ trợ vẫn vi phạm.

Test Coder `manager_test.go:317-357` chỉ dùng `return nil, connector.ErrLanguageServerNotFound` (`:348`). Test đó không thể đi qua nhánh `(nil,nil)` với credential reader đã đổi sang target. Test cũ mang tên `ReAuthFallback_ClientNil` hiện cũng dựng client HTTP, nên tên test không bổ sung được bằng chứng cho nhánh này.

**Hướng sửa để khép invariant:** mọi nil client sau recycle phải trả `Success=false` và lỗi, bất kể credential hiện đang chứa email nào. Nếu test UI/CLI cần thành công thì mock một client có GetUserStatus hợp lệ; không thêm ngoại lệ sản phẩm để giả lập RAM bằng WinCred. Bổ sung regression `(nil,nil)` với credential đã là target và giữ các đường mismatch/status error/empty/nil-with-error đang đạt.

### R2-F5 — [P2] Parser cũ vẫn chỉ được test gọi; 0 dead code chưa đạt

**Source:** `internal/reloader/active_threads.go:38-46,284-325`; caller duy nhất tại `internal/reloader/active_threads_test.go:292`. `isFileViewTab` đã bị xóa đúng, nhưng `extractProjectFromText` và năm regex không còn consumer sản phẩm sau khi resolver chuyển sang SQLite.

Phân biệt cấu hình phân tích là cần thiết: mặc định staticcheck tính cả test, nên test cũ giữ các symbol này có vẻ còn được dùng. Phân tích đúng mã production:

```text
go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tests=false -checks=U1000 ./internal/reloader/... ./internal/manager/...
internal\reloader\active_threads.go:38:2: var reActiveWorkspaces is unused (U1000)
internal\reloader\active_threads.go:41:2: var reCommandCwd is unused (U1000)
internal\reloader\active_threads.go:43:2: var reJSONCwd is unused (U1000)
internal\reloader\active_threads.go:44:2: var reJSONProj is unused (U1000)
internal\reloader\active_threads.go:45:2: var reFileURI is unused (U1000)
internal\reloader\active_threads.go:284:6: func extractProjectFromText is unused (U1000)
exit status 1
```

Source sweep bằng `rg` xác nhận helper chỉ được test gọi; năm regex chỉ phục vụ helper này. Đây là dead production code trong chính scope sửa F5, không phải lỗi lint format hoặc yêu cầu mở rộng sang subsystem khác.

Phân tích toàn workspace với test mặc định cũng vẫn không sạch:

```text
go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -checks=U1000 ./...
cmd\switcher\app_cli.go:12:7: const hasGUI is unused (U1000)
exit status 1
```

`hasGUI` không thuộc diff sửa lần này và đã được ghi nhận từ R1; ghi rõ để không quy nhầm nó là regression mới. Các gói reloader/manager không có diagnostic trong cấu hình **bao gồm tests**, nhưng điều đó không chứng minh 0 dead code của binary sản phẩm.

**Hướng sửa để khép invariant:** dọn helper parser/regex đã mất consumer sản phẩm và test chỉ phục vụ pipeline cũ; không thêm consumer giả để làm analyzer xanh. Xử lý `hasGUI` theo phạm vi 0 dead code toàn workspace trong hợp đồng, hoặc CEO phải thu hẹp rõ gate nếu không giao sửa nó. Giữ cả cấu hình `-tests=false` cho các package sản phẩm được sửa và kiểm tra workspace có phạm vi build nêu rõ.

## Các điểm bất thường khác đã ghi nhận

- **[P3, source và runtime] Thông báo lỗi empty RAM email bị format sai.** `account_manager.go:558-566` dùng `%w` với `statusErr == nil` khi status trả email rỗng. Reproduction `empty-email` trả failure đúng và không continuation, nhưng error là `failed verifying user status in RAM: %!w(<nil>)`. Cần tạo lỗi rõ cho status/email rỗng, chỉ wrap error khi error thực sự khác nil. Đây không phải nguyên nhân làm test gate chính thất bại.
- **[Source-only] Empty workspace array chưa được chuyển thành unknown.** Tại `active_threads.go:194-204`, JSON `[]` có len 0 rơi xuống `return rawURIs`, nên Project có thể là literal `[]` thay vì empty/unknown như fallback của đặc tả. Không có claim runtime riêng cho case này trong lượt R2; cần xử lý/kiểm chứng cùng boundary project resolver khi giao sửa tiếp. Lỗi gốc lấy project từ preview và ownership giữa các record đã được chứng minh đóng.
- **[Residual giữ từ R1] Snapshot capture/read errors vẫn bị bỏ qua.** `account_manager.go:420-423,584-589` còn fallback memory nếu không đọc được file hoặc disk list rỗng. F4 wrapper gốc đã sửa; đây không phải bằng chứng mọi failure path bảo đảm snapshot disk làm source of truth. Đường đọc thành công đã được kiểm chứng bằng memory/disk khác nhau.
- **[Residual cùng invariant] API ReAuth cũ còn trong connector.** `connector/api_client.go:216` và test `connector_test.go:248` vẫn tồn tại. SwitchAccount không có call ReAuth; R2 không nhận thêm sửa connector và không tuyên bố API ảo đã biến mất toàn repo.
- **[Test isolation]** Một số manager tests không cấu hình StoragePath; lệnh full-suite của Supervisor dùng profile APPDATA/USERPROFILE/store tạm. Foreground Win32 vẫn là hành vi thực của suite hiện hữu; ProcessKiller/credential writes trong manager checks được mock. Không biến các test live thành SKIP để lấy kết quả xanh.

## Source-level clearance notes

- **`internal/manager/account_manager.go`: chưa clear.** Đã có RAM gate tốt trên client hợp lệ, recycle failure nay trả thất bại đúng; chưa khép R2-F1 ở nil client. Đã đọc factory, GetCurrentActiveEmail, credential writer seams và đường gọi continuation, không dừng ở hunk thêm GetUserStatus.
- **`internal/reloader/active_threads.go`: F2/F3 gốc đã clear; F5 chưa clear.** Không còn bytes.Index/2.048-byte resolver. Query có parameter theo ID, đọc đúng URI đã percent-decode trong WAL fixture. Scanner giữ membership/threshold/sort/main pane. Parser production dư và unknown-array observation còn được ghi nhận ở trên.
- **`internal/reloader/session_reloader.go`: F4 gốc clear.** ConversationsDir và SummariesDbPath được resolve độc lập StoragePath/BrainDir; reproduction dùng conversations mặc định dưới profile và một StoragePath JSON tồn tại nay ghi snapshot thành công. Sleeper/recycle đã đọc để xác minh 8s trước discovery.
- **`internal/reloader/continuation.go` và connector callers:** được đọc như sibling boundaries, không nằm trong diff R2. Kiểm chứng HTTP default runner mới xác nhận nguyên vẹn model/ordering/pacing/per-thread errors và summary.
- **Các test sửa trong commit:** compile và full suite PASS. Tests mới bắt stale/status-error/nil-with-error, SQLite preview nhiễu và StoragePath explicit; còn thiếu `(nil,nil)` + credential target. Tests parser cũ là consumer duy nhất còn lại của parser đã rời pipeline sản phẩm.
- **`go.mod`/`go.sum`:** additions phù hợp registration driver mới và build/test đã compile; không tự chạy tidy hoặc thay đổi dependency trong lượt review.

## Evidence mới của R2

Tất cả lệnh chạy trên source commit `1c9e560` trước khi tạo báo cáo hoàn tất. Kết quả R1 được dùng để thiết kế reproduction, không thay thế proof R2.

| Kiểm chứng | Kết quả | Artifact |
|---|---|---|
| `go build ./...` | Exit 0, Go 1.26.3 Windows/amd64, CGO=1; không lỗi compiler/import | [go-build-result.json](evidence/2026-10-18-4-step-hotswap-r2/go-build-result.json) |
| `go test -v -count=1 ./...` | **10/10 package, 144 PASS events, 0 FAIL, 0 SKIP**, exit 0 | [log đầy đủ](evidence/2026-10-18-4-step-hotswap-r2/go-test-v-all.log), [JSON](evidence/2026-10-18-4-step-hotswap-r2/go-test-v-all-result.json) |
| Independent acceptance Go overlay | 13 leaf scenarios: 12 đạt, **1 lỗi `(nil,nil)`**; exit 1 | [trace đầy đủ](evidence/2026-10-18-4-step-hotswap-r2/independent-acceptance.log), [JSON](evidence/2026-10-18-4-step-hotswap-r2/independent-acceptance-result.json) |
| Full-workspace U1000, tests included | Exit 1, `hasGUI` diagnostic | [log](evidence/2026-10-18-4-step-hotswap-r2/staticcheck-u1000.log), [JSON](evidence/2026-10-18-4-step-hotswap-r2/staticcheck-u1000-result.json) |
| Production-only U1000, reloader/manager | Exit 1, 6 unused declarations trong active_threads.go | [log](evidence/2026-10-18-4-step-hotswap-r2/staticcheck-production-u1000.log), [JSON](evidence/2026-10-18-4-step-hotswap-r2/staticcheck-production-u1000-result.json) |

Independent log có **12 PASS events và 2 FAIL events** vì Go ghi FAIL cho cả nil-client leaf và parent table test. Chỉ có **một failing leaf**, không phải hai lỗi runtime độc lập. Có 8 top-level tests; không có SKIP hoặc lỗi bootstrap trong acceptance run.

Full-suite process dùng profile tạm tại `.tmp/supervisor-2026-10-18-4-step-r2/suite-profile/` qua APPDATA, USERPROFILE, ANTIGRAVITY_STORE_DIR; giữ Go module/build caches gốc. Live Language Server status, Chrome discovery và đọc WinCred mặc định đều thực sự PASS. Không đổi account thật hoặc gửi prompt vào conversation thật để tái hiện. Test WinCred round-trip của suite dùng target riêng; manager/acceptance credential writer và taskkill là mock.

### Tái hiện acceptance

```powershell
& .\reports\supervisor\evidence\2026-10-18-4-step-hotswap-r2\reproduce.ps1
```

- [reproduce.ps1](evidence/2026-10-18-4-step-hotswap-r2/reproduce.ps1) tạo Go overlay trong `.tmp`, không sửa source/test hiện hữu.
- Năm assertions reloader R1 được tái sử dụng trực tiếp từ artifact commit `35138bb`; chỉ đổi tên test để tránh trùng test mới của Coder, không thay assertions.
- [manager_acceptance_test.go.txt](evidence/2026-10-18-4-step-hotswap-r2/manager_acceptance_test.go.txt) kiểm tra sáu outcome RAM và exact disk snapshot/8s/RAM ordering.
- [reloader_extra_acceptance_test.go.txt](evidence/2026-10-18-4-step-hotswap-r2/reloader_extra_acceptance_test.go.txt) tạo SQLite thật, checkpoint schema trước, giữ kết nối writer mở với WAL autocheckpoint tắt, commit hai mapping vào WAL rồi gọi scanner. WAL có 16.512 byte lúc đọc; cả IDLE/RUNNING của hai project được trả đúng. Fixture là tổng hợp, không copy database live.

Trace nominal manager:

```text
snapshot → write-credential → kill → wait:8s → discover → new-client → ram-confirm → continue
```

Disk list cố ý khác list capture trả trong memory; continuation nhận đúng disk fields, kết quả summary/message 2/2. Trace default continuation riêng xác nhận hai `sleep:3s`, ba RPC đúng thứ tự từng thread, model được discovery và thread 2/3 thành công sau HTTP 500 của thread 1.

### Không chạy / giới hạn bằng chứng

- Không chạy real account switch, real taskkill hoặc continuation trên thread thật; không coi live GetUserStatus của suite là bằng chứng một lần switch thật đã hoàn tất.
- Không chạy riêng build tag `gui`, race suite hoặc full staticcheck ngoài U1000. Không dùng báo cáo Coder về Windows PE build thay proof độc lập cho các bước chưa chạy này.
- Không viết/sửa code sản phẩm để sửa findings; không chạy lại test/analysis sau khi báo cáo hoàn tất hoặc tự audit commit/hash sau commit output.
- Scratch profile/fixture/overlay ở `.tmp` là working data tái sinh được, không thuộc output source sản phẩm. Artifact chính thức và nguồn tái hiện được commit trong `reports/supervisor/`.

## Invariant closure và yêu cầu nộp lại

- **F1:** chưa khép. Các đường target/stale/status-error/empty/nil-with-error đã đạt; nil-without-error + target credential vẫn vi phạm. Cần loại bỏ toàn bộ post-recycle success dựa riêng vào WinCred, mock RAM thật ở test, bổ sung regression cho đúng reproduction trên.
- **F2:** lỗi gốc preview/record ownership đã khép với cả committed WAL và hai project; ghi nhận source-only empty-array boundary để xử lý và chứng minh unknown hợp lệ.
- **F3/F4:** các failure mode gốc đã khép ở source, direct scanner/default wrapper và runtime reproduction mới. Giữ các regression này khi sửa tiếp.
- **F5:** chưa khép. Xóa parser/regex không có production consumer; xử lý diagnostic `hasGUI` cho gate toàn workspace; tái kiểm cả U1000 production-only và workspace. Không chỉ dùng test cũ gọi helper làm bằng chứng helper còn sống trong sản phẩm.
- Sửa thông báo empty-email có `%!w(<nil>)` trong cùng F1 failure handling; trả lỗi mô tả đúng khi status không có email.
- Nộp commit sửa, fresh full suite không SKIP, acceptance reproduction nil-client-without-error đạt và static-analysis không còn diagnostics trong phạm vi gate đã giao. Không cần thay lại MAX mtime, UUID, SQLite query/WAL, wrapper, exact snapshot hay pacing đã được kiểm chứng đúng nếu không có regression mới.

## Đánh giá cuối

**REJECT.** R2 đã sửa được nhiều lỗi thực và suite hiện hữu xanh đầy đủ, nhưng hai yêu cầu bắt buộc vẫn bị source/runtime phủ định: F1 còn bypass báo thành công khi thiếu client RAM; F5 chưa đạt 0 dead production code. Báo cáo này ghi nhận đúng những phần đã đóng và handoff phần còn lại về CEO, không sửa sản phẩm trong vai trò Supervisor.
