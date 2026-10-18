# Supervisor R4: nghiệm thu sửa R3-F1

**Verdict: PASS**

## Metadata và claim được nghiệm thu

- Ngày kiểm định: 2026-10-18; evidence được đọc hoàn tất trước 10:17:50, Asia/Bangkok (UTC+07:00).
- Vai trò: Supervisor độc lập; CEO nhận handoff: `01a14378-4a08-77c1-88d5-666ad8c3592d`.
- Commit sản phẩm: `93308c5e912d68c77843a2d35ed1780346206068`. HEAD lúc bắt đầu trùng commit này; cây làm việc ban đầu sạch.
- Phạm vi nghiệm thu: sửa R3-F1 theo hợp đồng R4 — xóa server/client xác thực giả và magic PID/port khỏi mã manager; đưa HTTP mock về UI test, đăng ký qua SetClientFinder; sau recycle, nil client phải thất bại, không continuation; giữ toàn bộ 15 scenario độc lập và full suite không SKIP; kiểm tra U1000.
- Authority: hợp đồng CEO R4, báo cáo Supervisor R3 commit `c17add0`, đặc tả đọc trực tiếp từ commit `1ec2e6a`. Báo cáo Coder chỉ là đầu vào đối chiếu; không dùng kết luận hoặc log của Coder làm proof.
- Đã đọc lại skill supervisor bắt buộc trong repo và các references, working-rules và debugging. Không tìm thấy `AGENTS.md` trong checkout hoặc `G:\AGENTS.md`; hợp đồng/đặc tả vẫn xác định rõ scope. Debugging dùng bản khả dụng tại `C:\Users\TAM NGUYEN\.agents\skills\debugging\SKILL.md`.
- Supervisor không sửa source/test hiện hữu, không dùng Anvien hoặc subagent. Output R4 chỉ ở `reports/supervisor/`.

## Kết quả nghiệm thu

R3-F1 đã được khép ở cả source và runtime boundary. Manager không còn tạo HTTP test server hoặc nhận diện PID/port fixture; cả hai nil-client scenarios từng dùng để đối chứng ở R3 nay trả `Success=false`, error khác nil, summary=nil và không gửi RPC. UI mock được sở hữu bởi `_test.go`, có cleanup và được cung cấp qua factory seam hiện hữu.

Tất cả gate được giao đều đạt mới trên commit `93308c5`:

- **15/15 leaf scenarios độc lập PASS**, 17 PASS events khi tính cả parent tests; 0 FAIL, 0 SKIP; exit 0.
- **Full suite đủ 10 package: 140 PASS events, 0 FAIL, 0 SKIP**; `go test -v -count=1 ./...` exit 0.
- `go build ./...` exit 0.
- U1000 toàn workspace **0 diagnostics**, exit 0 với tests và với `-tests=false`, trong cấu hình build mặc định.

Không có blocking finding hoặc yêu cầu sửa bắt buộc còn lại trong phạm vi R3-F1/R4.

## Đối soát diff `git show 93308c5` và source-level clearance

Đã đọc toàn diff của commit và source runtime liên quan. Commit thay đổi một tệp sản phẩm, một tệp test và báo cáo Coder; không có thay đổi dependency hoặc pipeline khác.

| Tệp / invariant | Bằng chứng source và diff | Clearance |
|---|---|---|
| `internal/manager/account_manager.go`: bỏ mock khỏi sản phẩm | Diff xóa toàn branch tạo httptest server/client theo PID 12345/port 54321. Imports `net/http`, `net/http/httptest`, `net/url`, `strconv` bị xóa. Source sweep không còn `httptest`, `net/url`, `strconv`, `12345`, `54321`, `test-csrf`, `testServer` trong file này | Clear |
| Nil client sau recycle phải fail | `account_manager.go:506-539` chỉ lấy client qua finder/discovery rồi đi thẳng vào nil guard; cả nhánh có clientErr và không có clientErr đều trả `Success=false` và error khác nil | Clear |
| Identity RAM trước continuation | `account_manager.go:543-575` gọi GetUserStatus, từ chối status error, status/email rỗng và email mismatch. Continuation ở `:581-597` nằm sau các gate | Clear |
| `internal/ui/ui_test.go`: HTTP mock đúng nơi sở hữu | `:80-87` tạo server trong createTestManager; response phản ánh activeCredToken của fixture, không còn là logic trong sản phẩm | Clear |
| Lifecycle HTTP mock | `ui_test.go:88` đăng ký `t.Cleanup(testServer.Close)` | Clear |
| Client injection và discovery của test | `ui_test.go:90-108` lấy port thật của test server, tạo testClient, cấu hình ServerDiscoverer cùng port và SetClientFinder trả client đó | Clear |
| Sibling boundaries | Đối chiếu nil guard, credential seams, snapshot/continuation caller và các fixtures độc lập được giữ nguyên; không còn consumer sản phẩm dựng success endpoint để thay client nil | Clear trong invariant R3-F1 |

Số PID trong test fixture không điều khiển hành vi sản phẩm nữa. Việc test giữ PID giả không phải điều kiện ngoại lệ trong manager; runtime reproduction xác nhận cả ordinary và legacy discovery identity cùng fail khi finder nil.

## Bằng chứng đóng failure mode R3

Bộ test cũ được chạy lại với nguyên assertions và **không override ContinuationRunner** trong hai subcase R3. Trước switch có account old, writer cập nhật credential target, snapshot có một thread hợp lệ, killer/discoverer thành công và sleeper nhận `[8s]`. Finder sau recycle trả `(nil,nil)`.

| Discovery identity | Kết quả R4 | RPC và summary |
|---|---|---|
| PID 9999 / port 23456 | `Success=false`, error `không tìm thấy language server mới sau khi khởi động lại` | genuine_status_after_kill=0, genuine_sends=0, summary=nil |
| PID 12345 / port 54321 | **`Success=false`, cùng error khác nil** | genuine_status_after_kill=0, genuine_sends=0, summary=nil |

Cả hai có `finds=1`, `discovery_calls=1`, `waits=[8s]`. Case legacy từng báo 1/1 continuation từ server tự dựng ở R3 nay kết thúc trước RAM/continuation, không tạo summary. Source không còn đường dựng endpoint, nên kết quả được chứng minh từ cả control flow và thực thi, không chỉ từ việc test đổi tên hoặc đổi kỳ vọng.

`TestR2RAMGateEveryOutcome/nil-client-without-error` cũng PASS nguyên trạng với credential đã là target, `status_after_kill=0`, `continued=0`, error khác nil. Các outcomes target, stale, HTTP error, empty email và nil-with-error đều PASS; chỉ target RAM đúng được phép continuation.

## Regression của phương án 4 bước

Toàn bộ 15 leaf scenarios R3 được tái sử dụng, không giảm assertion:

- Inclusive -2h, MAX mtime DB/WAL, sort và main pane giữ đúng; file không phải UUID bị loại.
- Mapping SQLite đúng workspace_uris, không bị preview đường dẫn khác đánh lừa; committed WAL 16.512 byte, hai project, IDLE/RUNNING và percent-encoded URI đều được đọc đúng.
- StoragePath JSON không thay thế conversations directory trong default wrapper.
- Trace nominal: `snapshot → write-credential → kill → wait:8s → discover → new-client → ram-confirm → continue`.
- List trên disk cố ý khác list capture trong memory; continuation nhận đúng disk snapshot, giữ fields và summary/message 2/2.
- SmartFocus → LoadTrajectory → Send đúng thứ tự; active model đúng payload; hai sleep 3s giữa ba threads; thread đầu Send HTTP 500 vẫn tiếp tục hai threads sau và summary 3 total / 2 success / 1 failed đúng.
- Sáu outcomes RAM R2 và hai discovery identities R3 đều đạt.

UI suite, gồm `TestSwitchAccount_NoModalPopup`, PASS trong full-suite run mới với mock được chuyển về `_test.go`.

## Evidence mới và cách tái hiện

Environment: Go `go1.26.3`, Windows/amd64, CGO=1. Mọi kết quả dưới đây được chạy mới trên source `93308c5`, trước khi ghi báo cáo hoàn tất.

| Lệnh / kiểm chứng | Kết quả | Artifact |
|---|---|---|
| `go build ./...` | Exit 0, không lỗi compiler/import | [JSON](evidence/2026-10-18-4-step-hotswap-r4/go-build-result.json) |
| `go test -v -count=1 ./...` | **10 package, 140 PASS events, 0 FAIL/SKIP**, exit 0 | [full log](evidence/2026-10-18-4-step-hotswap-r4/go-test-v-all.log), [JSON](evidence/2026-10-18-4-step-hotswap-r4/go-test-v-all-result.json) |
| Independent acceptance overlay | **15/15 leaf scenarios, 17 PASS events, 0 FAIL/SKIP**, exit 0 | [log](evidence/2026-10-18-4-step-hotswap-r4/independent-acceptance.log), [JSON](evidence/2026-10-18-4-step-hotswap-r4/independent-acceptance-result.json) |
| `go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -checks=U1000 ./...` | Exit 0, 0 diagnostics | [results](evidence/2026-10-18-4-step-hotswap-r4/staticcheck-results.json), [raw log](evidence/2026-10-18-4-step-hotswap-r4/staticcheck-u1000.log) |
| `go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tests=false -checks=U1000 ./...` | Exit 0, 0 diagnostics | [results](evidence/2026-10-18-4-step-hotswap-r4/staticcheck-results.json), [raw log](evidence/2026-10-18-4-step-hotswap-r4/staticcheck-production-u1000.log) |

Acceptance có 9 top-level tests và 8 subtests; 17 Go PASS events tương ứng 15 leaf scenarios, không phải 17 leaf cases.

Chạy lại bằng PowerShell:

```powershell
& .\reports\supervisor\evidence\2026-10-18-4-step-hotswap-r4\reproduce.ps1
```

[Runner R4](evidence/2026-10-18-4-step-hotswap-r4/reproduce.ps1) chỉ tạo overlay/fixture trong `.tmp`, tham chiếu nguyên nguồn test R1/R2/R3 đã có, và lưu log vào thư mục R4 riêng. Không ghi đè log lịch sử hoặc thêm file test vào source sản phẩm. Các virtual `_test.go` vẫn nằm cạnh đúng Go package khi chạy.

Full-suite process dùng APPDATA/USERPROFILE/ANTIGRAVITY_STORE_DIR tạm tại `.tmp/supervisor-2026-10-18-4-step-r4/suite-profile/`, giữ Go caches gốc. Đã đọc toàn log: live LS discovery/status, Chrome discovery và đọc credential mặc định thực sự PASS, không bị chuyển thành SKIP. Manager credential writer/process killer và sleeper trong acceptance là mock; WinCred round-trip của suite dùng target test riêng. Foreground Win32 vẫn là hành vi thực của suite hiện hữu.

## Invariant closure và giới hạn

- **Đã khép:** post-recycle missing-client failure, không còn RAM/dispatch success tự sinh trong product, HTTP mock chỉ do UI test sở hữu và được cleanup. Không còn same-invariant surface chưa kiểm chứng trong phạm vi R3-F1 được giao: nil guard, RAM gate, continuation caller, UI factory, ordinary/legacy discovery fixtures đều đã đối soát.
- **Dead-code gate:** không có diagnostic U1000 trong hai cấu hình mặc định đã chạy, bao gồm production-only. Đây là bằng chứng theo gate được dùng trong các vòng review; không mở rộng thành claim mọi build tag hoặc exported API đều có consumer runtime.
- **Không chạy:** separate GUI tag/package PE build, race suite, real account switch/taskkill/continuation trên conversation thật. Không dùng báo cáo Coder về các bước này thay proof độc lập. Chúng không phải gate bổ sung của hợp đồng R4.
- **Ngoài phạm vi sửa R3-F1:** các residual đã ghi ở R2/R3 về snapshot error fallback, empty workspace array và connector ReAuth legacy không được tuyên bố đã sửa bởi commit này. PASS này áp dụng cho scope sửa chữa R3-F1 và các regression/gate được giao, không thay thế một acceptance toàn bộ failure paths hoặc mọi build configuration.
- Không có bất thường mới trong diff/runtime của scope R4 cần sửa trước khi nghiệm thu. Scratch `.tmp` là working data tái sinh được; output chính thức nằm trong `reports/supervisor/`.
- Không sửa source trong lane Supervisor; không chạy vòng kiểm tra sau khi báo cáo hoàn tất hoặc tự audit Git log/hash sau commit. Handoff lấy SHA từ commit output.

## Đánh giá cuối

**PASS — chấp nhận sửa R3-F1 tại `93308c5`.** Lỗi server tự sinh trong sản phẩm đã được loại bỏ đúng owner boundary, UI mock được chuyển về test có cleanup, và failure mode legacy PID/port đã được chứng minh đóng bằng chính assertions từng thất bại. Đủ 15 scenario độc lập, full suite không SKIP và hai gate U1000 đều đạt; không còn hành động sửa bắt buộc trong phạm vi hợp đồng R4.
