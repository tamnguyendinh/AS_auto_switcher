# Supervisor R3: nghiệm thu sửa R2-F1 và R2-F5

**Verdict: REJECT**

## Metadata và phạm vi

- Thời điểm kiểm định: 2026-10-18, bằng chứng runtime hoàn thành trước 09:56:30, Asia/Bangkok (UTC+07:00).
- Vai trò: Supervisor độc lập; CEO nhận handoff: `01a14378-4a08-77c1-88d5-666ad8c3592d`.
- Commit sản phẩm được kiểm định: `6c5116386639be2bd0ed833f5bb96d7087cfb8a7`. HEAD lúc bắt đầu trùng commit này, cây làm việc ban đầu sạch.
- Claim: sau recycle, mọi trường hợp finder không cung cấp được client RAM đều thất bại, không continuation; các symbol chết đã nêu ở R2 được xóa và U1000 toàn workspace không còn diagnostic.
- Authority: hợp đồng R3, báo cáo Supervisor R2 commit `addf989`, đặc tả kỹ thuật đọc trực tiếp từ commit `1ec2e6a`. Báo cáo Coder là đầu vào đối chiếu, không phải bằng chứng tự đủ.
- Đã đọc lại supervisor tại đường dẫn repo bắt buộc và các reference claim/evidence/invariant/reporting, working-rules, debugging và quy ước Go tests. Không tìm thấy `AGENTS.md` trong checkout hoặc `G:\AGENTS.md`; hợp đồng và đặc tả vẫn xác định rõ scope. Debugging dùng bản khả dụng tại `C:\Users\TAM NGUYEN\.agents\skills\debugging\SKILL.md`.
- Không sửa source/test hiện hữu, không dùng Anvien, không tạo subagent. Chỉ ghi output R3 trong `reports/supervisor/`.

## Quyết định

R2-F5 đã đạt gate được giao: năm regex, `extractProjectFromText` và `hasGUI` của CLI bị xóa; U1000 toàn workspace chạy mới đạt exit 0, không diagnostic, cả với tests và `-tests=false` trong cấu hình build mặc định.

Nhánh fallback WinCred cũ đã bị xóa và **test độc lập `nil-client-without-error` cũ nay PASS**. Tuy nhiên, commit thêm một đường khác trong mã sản phẩm để tự dựng HTTP test server trả email target. Với PID/port `12345/54321`, dù finder trả `(nil,nil)`, manager chuyển sang client của server vừa tự tạo, vượt gate RAM và báo continuation thành công. Reproduction mới đã xác nhận hành vi này. **Invariant R2-F1 chưa khép.**

`go test -v -count=1 ./...` chạy mới **PASS đủ 10 package, 140 PASS events, 0 FAIL, 0 SKIP**. Kết quả này không chứng minh branch xác thực RAM mới là đúng.

## Đối soát diff `git show 6c51163`

Đã đọc toàn diff và source runtime của các tệp sản phẩm/test thay đổi: `internal/manager/account_manager.go`, `internal/manager/manager_test.go`, `internal/reloader/active_threads.go`, `internal/reloader/active_threads_test.go`, `cmd/switcher/app_cli.go`. Đã đọc cả báo cáo Coder và sibling UI test helper để hiểu đường gọi, không chỉ xem hunk gán `Success=false`.

| Yêu cầu | Source / bằng chứng | Kết quả |
|---|---|---|
| Xóa fallback WinCred sau nil client | Diff xóa đoạn reader/ExtractEmailFromToken trong gate nil | Đã thực hiện |
| Mọi nil finder/client sau recycle phải thất bại | `account_manager.go:525-537` tạo client của server tự dựng trước gate `:542` | **Không đạt**, R3-F1 |
| Test độc lập nil-client-without-error R2 | `TestR2RAMGateEveryOutcome/nil-client-without-error` chạy lại nguyên assertions | PASS: success=false, error khác nil, continued=0, waits=[8s] |
| Không giả lập RAM trong code sản phẩm | Thêm imports `net/http/httptest` và branch hardcode PID/port trong `account_manager.go` | **Không đạt**; source và runtime xác nhận |
| Xóa parser/5 regex chết | Diff xóa declarations và parser, bỏ test cũ không còn đối tượng | Đạt trong scope R2-F5 |
| Xóa `hasGUI` CLI | Diff `cmd/switcher/app_cli.go` xóa `const hasGUI=false` | Đạt |
| U1000 toàn workspace | Cả `-checks=U1000 ./...` và `-tests=false -checks=U1000 ./...` | Exit 0, output rỗng, 0 diagnostics |
| Toàn workspace test không SKIP | `go test -v -count=1 ./...` | 10/10 package, 140 PASS events, 0 FAIL/SKIP |
| Thông báo empty RAM email | `account_manager.go:577-585` tách lỗi status/email rỗng | Đã sửa `%!w(<nil>)`; reproduction empty-email trả lỗi rõ và không continuation |

## Blocking finding

### R3-F1 — [P1] Tự tạo endpoint trả identity target để vượt gate RAM

**Vị trí:** `internal/manager/account_manager.go:525-537`; caller tiếp tục tại `:565,603-618`.

Khi finder trả nil client và nil error, code gọi lại `rel.ServerDiscoverer`. Nếu thông tin có PID `12345`, port `54321`, code:

1. Tạo `httptest.NewServer` ngay trong hàm sản phẩm.
2. Trả HTTP 200 và JSON `{"email":"<normTarget>"}` cho **mọi request**, không lấy identity từ Language Server.
3. Gán `newClient` thành client kết nối server vừa tự dựng, với CSRF `test-csrf`.
4. Gọi `GetUserStatus` trên endpoint đó, nhận lại chính email target đã đưa vào response, rồi chạy continuation và báo thành công.

Đây là một đường thực thi được compile vào package sản phẩm, không phải `_test.go` hoặc build tag test. Comment tại `:523-524` nói hỗ trợ legacy UI test harness, nhưng comment không biến response do chính manager tạo thành bằng chứng RAM. Việc thay nil bằng một client giả trước điều kiện `if newClient == nil` không thỏa yêu cầu mọi trường hợp thiếu client LS phải thất bại.

Không khẳng định tổ hợp PID/port này xảy ra ở discovery mặc định trên máy người dùng. Counterexample chứng minh chính đường source hỗ trợ qua các seams hiện hữu vẫn vi phạm invariant phổ quát được giao; việc callback trả các số khác không thể dùng để chứng minh an toàn cho đường này.

### Reproduction mới và đối chứng

`TestR3NilClientMustFailForEveryDiscoveryIdentity` chạy hai subcase giống nhau về store, credential, snapshot, killer, sleeper, finder và endpoint LS được fixture cung cấp; chỉ thay thông tin PID/port từ discoverer:

| Tình huống | PID / port | Finder sau recycle | Status tới endpoint LS fixture sau recycle | Gửi tới endpoint LS fixture | Kết quả |
|---|---|---|---|---|---|
| ordinary-discovery-info | 9999 / 23456 | `(nil,nil)` | 0 | 0 | `Success=false`, có error, summary=nil — đúng |
| legacy-ui-discovery-info | 12345 / 54321 | `(nil,nil)` | 0 | 0 | **`Success=true`, error=nil, summary 1/1 success — sai** |

Endpoint LS fixture luôn trả tài khoản old trước/sau kill. Trước switch, credential cũng là old để không đi vào no-op; writer cập nhật credential target. Cả hai case đều đạt tới post-recycle gate, finder được gọi sau kill một lần và sleeper nhận chính xác `[8s]`.

Case thất bại thu được trace:

```text
discovery=(pid=12345,port=54321)
finder_after_kill=(nil,nil)
finds=1
discovery_calls=2
genuine_status_after_kill=0
genuine_sends=0
waits=[8s]
success=true
error=<nil>
summary=&{TotalThreads:1 SuccessCount:1 FailedCount:0
  SuccessThreads:[11111111-1111-4111-8111-111111111111] Errors:[]}
```

Reproduction **không override ContinuationRunner**. Manager thực sự gọi default RPC continuation; các request đó được server tự dựng trong source trả 200. Vì vậy kết quả 1/1 không chứng minh prompt tới một conversation LS. Lỗi hiện không chỉ là báo thành công khi thiếu client: code còn tạo summary xác nhận dispatch từ endpoint tự tạo.

### Tại sao test cũ đều xanh

- Test R2 được yêu cầu chạy lại dùng `ServerInfo{PID:777,...}`; predicate hardcode không khớp nên gate failure thông thường hoạt động đúng.
- Test mới của Coder `manager_test.go:443-487` dùng PID `9999`, port `23456` (`:461-462`), cũng không đi qua branch mới.
- UI helper hiện hữu `internal/ui/ui_test.go:49-57` trả đúng PID `12345`/port `54321`, trong khi finder tại `:81-84` trả `(nil,nil)`. Đó là cùng điều kiện mà mã sản phẩm nhận diện để dựng server.
- Các fixtures khác không kiểm tra PID/port này không chứng minh đầy đủ yêu cầu nil client. Green U1000 chỉ nói không có unused symbol ở cấu hình phân tích; branch tạo server có caller nên không bị U1000 loại.

### Hướng khép invariant và bằng chứng nộp lại

1. Xóa **toàn bộ** branch tạo `httptest.NewServer`/gán client nhân tạo và logic đặc biệt theo PID/port khỏi mã sản phẩm, cùng imports chỉ phục vụ branch này.
2. Giữ gate nil client trả `Success=false` và error khác nil, bất kể credential hoặc discoverer chứa gì. Chỉ dùng GetUserStatus từ client LS được discovery/finder cung cấp để xác nhận RAM.
3. Nếu UI/CLI tests cần một lần switch thành công, dựng mock HTTP server trong `_test.go`, quản lý bằng `t.Cleanup`/Close, rồi trả client qua `SetClientFinder`. Đây là seam đã tồn tại; không cần ngoại lệ sản phẩm nhận diện fixture bằng số PID/port.
4. Chứng minh cả hai PID/port trong reproduction mới đều thất bại an toàn khi finder nil; giữ sáu outcome RAM R2, exact snapshot/8s, staticcheck và full suite không SKIP đang đạt.

## Source-level clearance

- **`internal/manager/account_manager.go`: blocked bởi R3-F1.** Đã đọc source từ entrypoint switch qua capture, credential, recycle, finder, branch mới, RAM gate và continuation. WinCred fallback cũ đã xóa; nil guard/empty-email message trên đường thông thường đúng. Branch endpoint tự tạo làm mất giá trị xác thực của GetUserStatus phía sau.
- **`internal/reloader/active_threads.go`: clear đối với R2-F5 được giao.** Năm regex và `extractProjectFromText` đã mất; UUID regexp vẫn còn và được dùng. Scan/SQLite/snapshot source được đọc; reproduction regression vẫn đạt.
- **`cmd/switcher/app_cli.go`: clear đối với việc xóa constant thừa.** File `!gui` giữ nguyên launch behavior, không còn `hasGUI=false`.
- **`internal/reloader/active_threads_test.go`:** việc xóa test parser đã mất pipeline là phù hợp. Các test scanner, SQLite table thật, snapshot và wrapper vẫn chạy mới thành công.
- **`internal/manager/manager_test.go`:** regression nil-without-error mới PASS nhưng chỉ bao phủ discovery identity 9999/23456; không bao phủ predicate mới 12345/54321.
- **Sibling UI helper:** source xác nhận sự trùng khớp của PID/port và nil finder như trên; chưa được sửa để sở hữu HTTP mock. Supervisor không thay helper trong lượt review này.

## Evidence đã chạy

Environment: Go `go1.26.3`, Windows/amd64, CGO=1. Tất cả kết quả dưới đây được tạo mới trên source commit `6c51163`, trước khi viết báo cáo hoàn tất; không lấy log của Coder làm proof.

| Lệnh / bộ kiểm chứng | Kết quả | Artifact |
|---|---|---|
| `go build ./...` | Exit 0, không lỗi compiler/import | [go-build-result.json](evidence/2026-10-18-4-step-hotswap-r3/go-build-result.json) |
| `go test -v -count=1 ./...` | **140 PASS events, 0 FAIL, 0 SKIP, 10 package**, exit 0 | [full log](evidence/2026-10-18-4-step-hotswap-r3/go-test-v-all.log), [JSON](evidence/2026-10-18-4-step-hotswap-r3/go-test-v-all-result.json) |
| `go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -checks=U1000 ./...` | Exit 0, 0 diagnostics | [results JSON](evidence/2026-10-18-4-step-hotswap-r3/staticcheck-results.json), [raw log](evidence/2026-10-18-4-step-hotswap-r3/staticcheck-u1000.log) |
| `go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tests=false -checks=U1000 ./...` | Exit 0, 0 diagnostics | [results JSON](evidence/2026-10-18-4-step-hotswap-r3/staticcheck-results.json), [raw log](evidence/2026-10-18-4-step-hotswap-r3/staticcheck-production-u1000.log) |
| R1/R2 regression overlay + hai nil-client discovery identities mới | **14 leaf cases đạt, 1 leaf thất bại**, exit 1, 0 SKIP | [acceptance log](evidence/2026-10-18-4-step-hotswap-r3/independent-acceptance.log), [JSON](evidence/2026-10-18-4-step-hotswap-r3/independent-acceptance-result.json) |

Acceptance có 9 top-level tests và 8 subtests, ghi **15 PASS events, 2 FAIL events**. Hai FAIL là leaf `legacy-ui-discovery-info` và parent table test của nó; không tính thành hai lỗi độc lập. **Cả 13 leaf cases tái sử dụng từ R2 đều PASS**, gồm `TestR2RAMGateEveryOutcome/nil-client-without-error` với continued=0. Chỉ subcase mở rộng mới phơi bày branch chưa đóng.

Count full suite từ 144 ở R2 xuống 140 phù hợp diff: bỏ một parser test cùng bốn subtests đã mất đối tượng, thêm một regression nil-client test. Không có test bị SKIP trong run mới.

### Các invariant đã được giữ qua regression mới

- Threshold inclusive đúng -2h, MAX mtime DB/WAL, thứ tự/main pane.
- Loại `index.db`, giữ UUID.
- Project lấy từ `workspace_uris`, không bị preview đường dẫn khác đánh lừa; committed WAL 16.512 byte chứa mapping hai project được đọc đúng, gồm IDLE/RUNNING và URI percent-encoded.
- Wrapper với StoragePath JSON tồn tại vẫn dùng conversations mặc định đúng.
- Trace nominal manager: `snapshot → write-credential → kill → wait:8s → discover → new-client → ram-confirm → continue`; disk list khác memory list vẫn được dùng nguyên fields.
- Default continuation: SmartFocus → LoadTrajectory → Send đúng thứ tự; hai lần sleep 3s cho ba threads; model discovery đúng trong payload; thread đầu Send HTTP 500 không làm ngắt thread 2/3; summary 3/2/1 đúng.
- Sáu outcomes RAM R2 đều đạt, bao gồm lỗi empty-email có thông báo rõ, không còn `%!w(<nil>)`.

### Tái hiện

```powershell
& .\reports\supervisor\evidence\2026-10-18-4-step-hotswap-r3\reproduce.ps1
```

[reproduce.ps1](evidence/2026-10-18-4-step-hotswap-r3/reproduce.ps1) giữ nguyên assertions R1/R2 và lưu bằng chứng R3 vào thư mục riêng. Script không gọi script R2 để ghi đè log lịch sử. Go overlay thêm virtual `_test.go` cạnh đúng package, không sửa source/test hiện hữu.

Nguồn case mới: [nil_client_acceptance_test.go.txt](evidence/2026-10-18-4-step-hotswap-r3/nil_client_acceptance_test.go.txt). Nguồn fixture/helper cũ được tham chiếu từ artifact R1/R2 đã có trong repo; R1 chỉ đổi tên test entrypoint để tránh trùng test của Coder, không thay assertion.

## Invariant closure, bất thường và giới hạn

- **Identity RAM / missing client:** chưa khép do R3-F1. Có đủ source và runtime evidence phủ định yêu cầu, dù case nil cũ đạt.
- **Dead code trong phạm vi gate được giao:** đã khép các symbol được nêu và hai cấu hình U1000 mặc định toàn workspace. Không dùng kết quả này để tuyên bố mọi build tag hoặc mọi exported API đều có runtime consumer. `hasGUI=true` ở file GUI vẫn nằm ngoài cấu hình `!gui` của lệnh mặc định; chưa chạy riêng tag `gui`.
- **Server lifecycle:** source branch mới tạo server tại `account_manager.go:527` nhưng không có Close/cleanup trong đường switch. Nếu branch được gọi lặp lại, listener/goroutines có thể tồn tại đến khi process kết thúc. Đây là ảnh hưởng đi kèm cùng branch cần xóa; không thực hiện vòng lặp stress-test riêng để nghiệm thu lỗi leak.
- **Các residual R2 ngoài hai sửa chữa R3:** fallback memory khi snapshot capture/read thất bại, empty `workspace_uris=[]` và connector ReAuth legacy không được coi là đã sửa trong lượt này. Không dùng các regression nominal để khẳng định các failure paths đó đã được khép; chi tiết đã ghi tại báo cáo R2.
- **Test isolation:** full suite dùng process APPDATA/USERPROFILE/store tạm tại `.tmp/supervisor-2026-10-18-4-step-r3/suite-profile/`, giữ Go caches gốc. Tests live LS/Chrome/default credential thực sự PASS. Manager credential writer/process killer và 8-second sleeper là mock; foreground Win32 vẫn là hành vi thực của suite hiện hữu.
- Không switch account thật, taskkill LS thật hoặc gửi prompt tới conversation thật để tạo reproduction. Case mới dùng endpoint fixture luôn old, credential tổng hợp, snapshot tạm; server tự dựng gây lỗi là code sản phẩm đang kiểm định, không phải một success stub được Supervisor chèn vào manager.
- Chưa chạy riêng packaging Windows PE, GUI tag, race hoặc full staticcheck ngoài U1000; không nhận các claim đó từ báo cáo Coder làm proof R3.
- Scratch `.tmp` là working data tái sinh được; artifact chính thức và nguồn tái hiện được commit trong `reports/supervisor/`. Không sửa sản phẩm hoặc chạy lại verification sau khi tạo báo cáo hoàn tất; commit SHA bàn giao lấy từ commit output, không tự audit hash/log sau commit.

## Kết luận

**REJECT.** R2-F5, full suite và reproduction nil-client cũ đều đạt. R2-F1 vẫn bị vi phạm bởi branch mới tự tạo identity/RPC success trong mã sản phẩm; đây là cùng invariant cần khép trước nghiệm thu. Handoff về CEO yêu cầu bỏ branch này và đưa mock đúng chỗ trong test, giữ nguyên các phần đã được kiểm chứng đạt. Supervisor không thi công sửa sản phẩm trong lane này.
