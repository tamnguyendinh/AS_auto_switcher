# Báo cáo Supervisor: nghiệm thu độc lập quy trình hotswap 4 bước

**Verdict: REJECT**

## Metadata và phạm vi

- Thời điểm kiểm định: 2026-10-18, khoảng 08:44–08:52, Asia/Bangkok (UTC+07:00).
- Vai trò: Supervisor độc lập, thực hiện hợp đồng từ CEO `01a14378-4a08-77c1-88d5-666ad8c3592d`.
- Commit sản phẩm được kiểm định: `272170f74c47bdafd0f49be647f2c3387647d06e`.
- Đặc tả có thẩm quyền: `reports/architect/2026-10-18-coder-implementation-spec-4-step-hotswap.md`, đọc trực tiếp từ commit `1ec2e6a` bằng `git show`.
- Claim: sản phẩm thực hiện đầy đủ snapshot SQLite 2 giờ trên mọi project → đổi credential/ToS, foreground, recycle, chờ 8 giây và xác nhận RAM → đọc đúng snapshot → tiếp tục tuần tự với pacing 3 giây; toàn workspace không có test SKIP và không có dead code/import thừa.
- Đọc trực tiếp diff `git show --format= --no-ext-diff 272170f -- <paths>` của cả sáu tệp trong hợp đồng và `internal/manager/account_manager_test.go` được sửa thêm trong cùng commit. Đã đọc báo cáo Coder làm đầu vào, không dùng kết luận của báo cáo đó làm bằng chứng nghiệm thu.
- Source thực thi lúc bắt đầu kiểm định đúng commit trên; cây làm việc ban đầu sạch. Supervisor không sửa tệp sản phẩm hoặc test hiện hữu. Output thuộc lane chỉ nằm trong `reports/supervisor/`.
- Áp dụng skill supervisor trong `.agents/skills/supervisor/`, working-rules, debugging và quy ước test Go. Không có `AGENTS.md` trong checkout hoặc tại `G:\AGENTS.md` tại thời điểm tìm kiếm; hợp đồng CEO và đặc tả cung cấp phạm vi và tiêu chí đủ rõ để tiếp tục. Skill debugging tại đường dẫn repo không tồn tại nên đã đọc bản khả dụng `C:\Users\TAM NGUYEN\.agents\skills\debugging\SKILL.md`.
- Không dùng Anvien, không mở lane phụ, không sửa lỗi sản phẩm trong vai trò Supervisor.

## Quyết định

Không thể nghiệm thu claim khớp 100% đặc tả. Suite hiện hữu chạy mới hoàn toàn **PASS trên 10 package, 139 test/subtest PASS events, 0 FAIL, 0 SKIP**, nhưng bốn kiểm chứng độc lập tái hiện sai lệch về identity RAM, lọc UUID, ánh xạ project và cấu hình đường dẫn snapshot. Staticcheck còn phát hiện hàm chết trong chính tệp đã thay đổi.

Phần chờ 8 giây đã được xác nhận bằng source và mock sleeper theo đúng điều kiện hợp đồng. Phần continuation với thread hợp lệ cũng đạt thứ tự RPC, model tự động, pacing và khả năng tiếp tục sau lỗi thread đầu. Những kết quả đó không khép được các invariant đang hỏng ở bước 1 và bước 2.

## Đối soát 4 bước với source và kết quả thực thi

| Yêu cầu | Source được kiểm tra | Bằng chứng mới | Kết quả kỹ thuật |
|---|---|---|---|
| Quét tất cả `.db`/`.db-wal`, không giới hạn project hoặc loại IDLE | `active_threads.go:81-159` duyệt toàn bộ entries/candidates; không lọc project/status | Suite hiện hữu; `TestSupervisorThresholdInclusiveAndWAL` | Logic duyệt toàn bộ và ngưỡng đạt; đầu vào UUID chưa được bảo vệ, xem F3 |
| `MAX(mtime(.db), mtime(.db-wal)) >= start - 2h`, sort mới nhất, main pane đầu tiên | `active_threads.go:132-159` | Fixture độc lập: đúng -2h được chọn; DB -3h + WAL -1m được chọn và lên đầu; -2h-1s bị loại | Đạt trên các trường hợp đã kiểm chứng |
| Chỉ nhận file thread UUID | `active_threads.go:98-109` | `TestSupervisorRejectNonUUIDDatabase`: `index.db` thành thread `index`, còn được chọn làm main pane | Không đạt, F3 |
| Project lấy đúng từ `conversation_summaries.db.workspace_uris` hoặc fallback an toàn | `active_threads.go:165-208,306-346` | SQLite thật do Python tạo: `workspace_uris=C:/ActualProject`, kết quả `D:/UnrelatedPreview` | Không đạt, F2 |
| Ghi một snapshot JSON và đọc lại đầy đủ trường | `active_threads.go:214-272`; wrapper `session_reloader.go:398-419` | Round-trip hiện hữu PASS; test độc lập cho đường dẫn wrapper thất bại | Hàm ghi/đọc trực tiếp đạt; đường cấu hình wrapper không đạt, F4 |
| ToS trước WinCred, dùng target Windows chuẩn | `account_manager.go:459-479`; `session_reloader.go:292-372`; `wincred/cred.go:171-189`; `wincred/types.go:15` định nghĩa đúng `LegacyGeneric:target=gemini:antigravity` | Test độc lập đọc file ToS ngay trong writer, thấy username target và chuỗi `"false"` cho consumer; round-trip WinCred test target riêng PASS | Đạt đối với consumer được kiểm chứng; sample code đặc tả giữ lựa chọn ToS theo AuthMethod |
| Foreground rồi taskkill recycle | `account_manager.go:482-493`; `session_reloader.go:73-88` | Source gọi `EnsureAntigravityForeground` trước recycle; `TestEnsureAntigravityForeground` PASS; process-killer mock được gọi | Wiring đạt; không kill LS thật trong kiểm chứng switch |
| **Chờ đủ 8 giây sau kill, trước discovery** | `account_manager.go:488-493`; `session_reloader.go:137-151`, default sleeper `:98-104` | `TestSupervisorDiskSnapshotAndEightSecondOrder`: `snapshot → write-credential → kill → wait:8s → discover → new-client → continue`; waits chính xác `[8s]` | **Đạt trên đường thành công với context còn hiệu lực; mock sleeper được hợp đồng cho phép** |
| Bỏ nhánh ReAuth khỏi `SwitchAccount` | Diff `account_manager.go`, đường mới `:485-514` | Không còn call `TriggerReauth` trong `SwitchAccount`; test hiện hữu xác nhận recycle và không gọi ReAuth | Nhánh switch đã bỏ; connector API/tables/test ReAuth vẫn còn trong repo, ghi rõ tại F5 và phần residual |
| Sau recycle, xác nhận tài khoản trong RAM qua `GetUserStatus` trước continuation | `account_manager.go:506-553`; `session_reloader.go:150-159` chỉ kiểm PID/port | `TestSupervisorSwitchRequiresRAMConfirmation`: `status_after_kill=0`, `sends=1`, `Success=true`, RAM fixture luôn là tài khoản cũ | Không đạt, F1 |
| Bước 3 đọc chính `snapshotPath` từ bước 1 | `account_manager.go:417-423,519-523` | `TestSupervisorDiskSnapshotAndEightSecondOrder` cố ý cho memory list khác disk list; kết quả continuation nhận đúng hai thread trên disk, giữ Project/IsMainPane/Model | Đạt trên đường đọc thành công; lỗi đọc/empty list còn fallback memory theo pseudocode đặc tả |
| SmartFocus → LoadTrajectory → Send tuần tự, pacing 3s, model tự phát hiện, lỗi từng thread, summary | `continuation.go:77-121`; connector `api_client.go:249-437` | `TestSupervisorFirstThreadFailureStillPacesAndSendsRemainingThreads`: thread 1 Send HTTP 500; threads 2/3 vẫn gửi; đúng hai `sleep:3s`; cả ba payload dùng model fixture được discovery; summary 3/2/1 | Đạt cho danh sách thread hợp lệ và mock sleeper thành công |
| Toàn workspace test PASS không SKIP | `go test -v -count=1 ./...` | 10 package; 139 PASS events; 0 FAIL; 0 SKIP; exit 0 | Đạt tiêu chí suite hiện hữu, không đủ để phủ nhận các reproduction thất bại |
| 0 dead code/import thừa | `go build ./...`; staticcheck U1000 | Build exit 0; U1000 báo `isFileViewTab` và `hasGUI` | Không có lỗi import từ compiler; **0 dead code không đạt**, F5 |

## Blocking findings

### F1 — [P1] Bỏ mất bước xác nhận RAM, vẫn gửi tiếp và báo đổi tài khoản thành công

**Vị trí:** `internal/manager/account_manager.go:506-553`; cùng invariant tại `internal/reloader/session_reloader.go:150-159`.

Sau recycle, manager lấy `newClient` rồi đi thẳng sang đọc snapshot và continuation. Không có `newClient.GetUserStatus` hoặc so sánh email với `normTarget`. Trong diff commit, đoạn re-verification cũ bị xóa mà không có đoạn thay thế. Constructor finder (`account_manager.go:94-100`) chỉ discovery rồi tạo client; recycle chỉ kiểm tra PID/port hợp lệ, không xác nhận identity.

**Reproduction:** `TestSupervisorSwitchRequiresRAMConfirmation` dùng HTTP server luôn trả `old@supervisor.invalid`; mock killer/discoverer thành công; writer chỉ mô phỏng ghi credential; sleeper nhận `[8s]`. Kết quả:

```text
status_after_kill=0
sends=1
Success=true
Message=... Đã gửi prompt tiếp tục phiên vào 1/1 thread.
error=<nil>
```

**Tác động:** hệ thống có thể tiếp tục công việc bằng tài khoản RAM cũ và khẳng định target đã hoạt động. Chờ 8 giây và có PID/port không chứng minh token mới đã vào RAM. Vi phạm bước 2, mục 1 của đặc tả và mục kiểm định 3 trong hợp đồng. Pseudocode manager trong mục 3.3 không tự cung cấp gate này; tiêu chuẩn ở mục Target Invariants và hợp đồng vẫn yêu cầu xác nhận RAM.

**Hướng khép invariant:** kiểm tra `GetUserStatus` trên client mới, so sánh email đã normalize với target trước mọi continuation hoặc thông báo thành công. Lỗi, nil client, nil/empty status và mismatch phải có kết quả trung thực, không gửi prompt trên identity chưa được xác nhận. Bổ sung regression cho target đúng, stale account, status lỗi và không tìm thấy client.

### F2 — [P2] Project có thể bị lấy từ preview bất kỳ, không phải `workspace_uris`

**Vị trí:** `internal/reloader/active_threads.go:180-191,306-342`.

Resolver đọc nguyên binary SQLite, tìm lần xuất hiện đầu tiên của thread ID bằng `bytes.Index`, rồi chạy text regex trên 2.048 byte kế tiếp. Đây không phải đọc record/cột SQLite. Không có ràng buộc vào `conversation_id` và `workspace_uris`; regex `Command Working Directory` có ưu tiên cao hơn URI. Không đọc WAL của summaries, cũng không có bảo đảm record không tràn sang vùng dữ liệu khác.

**Reproduction:** Python chuẩn `sqlite3` tạo database thật với bảng `conversation_summaries`, các cột có tên/thứ tự tương ứng phần liên quan của schema runtime, thread UUID hợp lệ, `status=IDLE`:

```text
preview = Command Working Directory: D:/UnrelatedPreview\n
workspace_uris = ["file:///C:/ActualProject"]
expected Project = C:/ActualProject
actual Project = D:/UnrelatedPreview
```

`TestSupervisorSQLiteWorkspaceColumn` thất bại đúng giá trị trên. Trước đó đã đọc schema runtime bằng kết nối SQLite `mode=ro` và `PRAGMA table_info(conversation_summaries)`; không dùng dữ liệu cá nhân làm fixture.

**Tác động:** snapshot ghi project sai trong khi nguồn chuẩn có mapping đúng. Đặc tả cho phép unknown/empty khi không resolve được; không cho phép gán nhầm project từ một chuỗi preview không liên quan.

**Hướng khép invariant:** đọc đúng `workspace_uris` của record theo `conversation_id` bằng cơ chế SQLite read-only có hiểu record/WAL, hoặc dùng fallback có liên kết chính xác thread và trả unknown khi không xác định được. Regression cần database SQLite thật với preview chứa đường dẫn gây nhiễu, nhiều project/record và dữ liệu summaries trong WAL. Không mở rộng thành sửa database của người dùng.

### F3 — [P2] Bộ quét chấp nhận mọi basename `.db`/`.db-wal`, chưa lọc UUID

**Vị trí:** `internal/reloader/active_threads.go:98-109`.

Code chỉ bỏ thread ID rỗng và tên `conversation_summaries`. Mọi basename khác đều đi vào danh sách, dù không phải UUID.

**Reproduction:** thư mục có một UUID `.db` hợp lệ và một `index.db` mới hơn. `TestSupervisorRejectNonUUIDDatabase` thu được:

```text
[{ThreadID:index IsMainPane:true} {ThreadID:11111111-1111-4111-8111-111111111111 IsMainPane:false}]
```

**Tác động:** file không phải conversation được đưa vào snapshot, chiếm main pane và được dùng làm `cascadeId`, thêm RPC/pacing lỗi. Vi phạm mục 3.1.B.4 của đặc tả: bỏ file không phải UUID.

**Hướng khép invariant:** validate thread UUID sau khi bỏ đúng suffix, trước khi tạo candidate; áp dụng nhất quán cả DB và WAL. Regression cần UUID hợp lệ, UUID sai, file index/backup/summaries và WAL liên quan. Fixture hiện hữu dùng các ID `thread-a`, `thread-step4-1` nên chưa kiểm chứng được contract UUID.

### F4 — [P2] Wrapper vẫn truyền `StoragePath`/`BrainDir` cũ vào signature conversations/summaries mới

**Vị trí thay đổi:** `internal/reloader/active_threads.go:214-216`; caller cùng invariant chưa cập nhật tại `internal/reloader/session_reloader.go:398-419`; lỗi bị manager bỏ qua ở `internal/manager/account_manager.go:420-423`.

Default capture mới hiểu hai tham số đầu là thư mục conversations và file summaries SQLite. Wrapper vẫn lấy `r.StoragePath` (đường dẫn `app_storage.json` dùng cho ToS) làm tham số đầu, `r.BrainDir` làm tham số thứ hai. Khi cấu hình đường dẫn ToS hợp lệ, scanner gọi `os.ReadDir` trên tệp JSON.

**Reproduction:** `TestSupervisorStoragePathDoesNotReplaceConversationsDirectory` tạo profile tạm có một conversation UUID trong đúng thư mục mặc định, một tệp `app_storage.json` tồn tại, cấu hình `rel.StoragePath` trỏ tệp này. Capture trả path rỗng, không có threads và lỗi `failed reading conversations dir: ...app_storage.json...`. Không cần taskkill hoặc credential để tái hiện.

**Tác động:** cấu hình ToS khiến bước snapshot thất bại; manager bỏ lỗi rồi vẫn đổi tài khoản, nên không có snapshot để thực hiện bước 3 và không tiếp tục được danh sách cần bảo toàn. Test 4 bước mới dùng `SnapshotCapturer` closure bỏ qua hai tham số được wrapper truyền nên che khuất lỗi tích hợp này.

**Hướng khép invariant:** tách rõ cấu hình conversations/summaries khỏi StoragePath/BrainDir cũ và cập nhật wrapper/defaults/caller; không dùng tệp ToS làm thư mục scan. Test cần đi qua wrapper thật với `StoragePath` được cấu hình, không thay toàn bộ capture bằng closure. Xử lý lỗi snapshot phải phản ánh việc quy trình 4 bước chưa hoàn tất.

### F5 — [P2] Tiêu chí 0 dead code không đạt

**Vị trí:** `internal/reloader/active_threads.go:276-296`; kết quả thêm ngoài diff tại `cmd/switcher/app_cli.go:12`.

Chạy độc lập:

```text
go run honnef.co/go/tools/cmd/staticcheck@latest -checks=U1000 ./...
cmd\switcher\app_cli.go:12:7: const hasGUI is unused (U1000)
internal\reloader\active_threads.go:276:6: func isFileViewTab is unused (U1000)
exit status 1
```

Module tool được resolver tải ở phiên này là `honnef.co/go/tools v0.8.2`. Lệnh tương đương có thể pin `@v0.8.2` khi tái hiện. `rg -n 'isFileViewTab' internal cmd --glob '*.go'` chỉ còn declaration. Diff commit xóa các caller của helper layout cũ nhưng giữ lại helper. `hasGUI` thuộc tệp không được sửa trong commit nên được ghi nhận riêng, không quy là regression do hotswap.

Ngoài ra, nhánh gọi ReAuth trong manager đã bỏ nhưng `Client.TriggerReauth` (`connector/api_client.go:215-245`), endpoint/table ReAuth (`connector/endpoints.go:7,31-37`) và test `TestAuthLogoutAndTriggerReauth` vẫn tồn tại. Tra cứu caller hiện tại chỉ thấy test gọi `TriggerReauth`, không thấy production caller. Đây là tồn dư cùng invariant; kết luận “API ảo đã bị xóa toàn repo” sẽ không đúng. U1000 không đủ để tự chứng minh toàn bộ exported API là dead code; report không dùng nó cho claim rộng hơn đó.

**Hướng khép invariant:** dọn helper đã mất caller trong slice; CEO giao phạm vi phù hợp để xử lý kết quả `hasGUI` nếu giữ tiêu chuẩn 0 dead code trên toàn workspace; xác định và dọn API/test ReAuth còn lại theo yêu cầu loại bỏ cơ chế ảo. Chạy lại static analysis có phạm vi build được ghi rõ. Compiler đã chứng minh không có lỗi import thừa trong cấu hình build/test đã chạy, không thay thế được reachability analysis.

## Source-level clearance notes

- `internal/reloader/active_threads.go`: **blocked** bởi F2/F3/F4/F5. Đã đối chiếu toàn diff và source cuối; MAX mtime, ngưỡng inclusive, sort, direct JSON round-trip được clearance trong phạm vi đã chạy.
- `internal/reloader/continuation.go`: **clear cho default runner với thread hợp lệ**. Source `:84-118` và trace độc lập xác nhận discovery model một lần, đúng thứ tự ba RPC từng thread, đúng hai yêu cầu nghỉ 3 giây cho ba threads và tiếp tục sau lỗi thread đầu. Summary `3 total / 2 success / 1 failed` cùng success IDs/errors đúng. ContinuationRunner injection cũng được cập nhật để trả summary; không dùng injection làm bằng chứng HTTP của default runner.
- `internal/manager/account_manager.go`: **blocked** bởi F1 và ảnh hưởng tích hợp F4. ToS/credential/kill/wait/discovery, đọc disk snapshot và summary/message trên đường nominal được clearance độc lập. Không gọi ReAuth trong switch.
- `internal/reloader/active_threads_test.go`: tests hiện hữu PASS nhưng dùng file byte/text giả cho summaries và ID không phải UUID, thiếu hai invariant F2/F3; direct round-trip đạt.
- `internal/reloader/continuation_test.go`: suite PASS; Supervisor bổ sung kiểm chứng first-thread error, trace toàn thứ tự RPC/sleeper và model thực trong payload, đều đạt.
- `internal/manager/manager_test.go`: suite PASS và đúng tham số wait 8 giây; handler có khả năng trả target sau kill nhưng test không bắt buộc `GetUserStatus` được gọi sau kill. Snapshot closure bypass wrapper, nên không bắt F1/F4. Các bộ đếm focus/load/send không được assert trong test mới.
- `internal/manager/account_manager_test.go`: hai hunk đổi kỳ vọng sang recycle và cấm ReAuth là phù hợp hướng mới; các test cũ về fallback/mismatch vẫn PASS nhưng chưa có gate bảo vệ post-recycle RAM như F1.

## Bằng chứng đã kiểm tra

Tất cả bằng chứng bên dưới được tạo mới trên source commit `272170f`, trước khi viết báo cáo này. Không dùng kết quả test cached của Coder làm proof.

| Lệnh/kiểm chứng | Kết quả | Artifact |
|---|---|---|
| `go build ./...` (Go 1.26.3 Windows/amd64, CGO=1) | exit 0, không có stdout/stderr hoặc lỗi compiler/import | [go-build-result.json](evidence/2026-10-18-4-step-hotswap/go-build-result.json) |
| `go test -v -count=1 ./...` | exit 0, 10 package, 139 PASS events, 0 FAIL, **0 SKIP** | [go-test-v-all.log](evidence/2026-10-18-4-step-hotswap/go-test-v-all.log), [kết quả JSON](evidence/2026-10-18-4-step-hotswap/go-test-v-all-result.json) |
| 7 kiểm chứng acceptance bằng Go overlay | 3 PASS, **4 FAIL**, 0 SKIP, exit 1 | [independent-acceptance.log](evidence/2026-10-18-4-step-hotswap/independent-acceptance.log), [kết quả JSON](evidence/2026-10-18-4-step-hotswap/independent-acceptance-result.json) |
| Staticcheck `-checks=U1000 ./...` | hai diagnostics unused, exit 1 | [staticcheck-u1000.log](evidence/2026-10-18-4-step-hotswap/staticcheck-u1000.log), [kết quả JSON](evidence/2026-10-18-4-step-hotswap/staticcheck-u1000-result.json) |
| Đọc schema summaries runtime | kết nối SQLite mode=ro; xác nhận cột `conversation_id`, `preview`, `workspace_uris`, `status` và thứ tự tương ứng | command output trong phiên kiểm định; fixture tái hiện độc lập bằng Python chuẩn |

Full suite chạy với `APPDATA`, `USERPROFILE` và `ANTIGRAVITY_STORE_DIR` của process trỏ profile tạm trong `.tmp/supervisor-2026-10-18-4-step/`, giữ Go cache/module cache ban đầu. Không sửa profile ToS thật để lấy kết quả PASS. Các test live discovery LS, Chrome discovery và đọc credential mặc định thực sự chạy và PASS; không bị đổi thành SKIP. Round-trip WinCred chỉ ghi/xóa target test riêng của suite. Source test đã được kiểm tra để xác nhận taskkill được mock; foreground Win32 là hành vi thực của suite hiện hữu.

Acceptance tests dùng mock credential/process/sleeper, HTTP loopback server và SQLite tổng hợp. Go overlay thêm `_test.go` ảo cạnh đúng package, không sửa source/test gốc. Đây là kiểm chứng runtime tại ranh giới scanner/SQLite, wrapper, manager và Connect-RPC client; không phải mô tả suy luận từ báo cáo Coder.

### Tái hiện các sai lệch

Chạy từ PowerShell:

```powershell
& .\reports\supervisor\evidence\2026-10-18-4-step-hotswap\reproduce.ps1
```

Script tạo fixture/overlay trong `.tmp`, chạy đúng bảy kiểm chứng `TestSupervisor*` và lưu log cùng JSON. Tại commit được kiểm định, exit 1 và bốn failing tests là kết quả tái hiện lỗi sản phẩm, không phải lỗi bootstrap. Nguồn fixture và hai test overlay được giữ tại:

- [make_sqlite_fixtures.py](evidence/2026-10-18-4-step-hotswap/make_sqlite_fixtures.py)
- [reloader_acceptance_test.go.txt](evidence/2026-10-18-4-step-hotswap/reloader_acceptance_test.go.txt)
- [manager_acceptance_test.go.txt](evidence/2026-10-18-4-step-hotswap/manager_acceptance_test.go.txt)

Không chạy lại suite hoặc kiểm tra vòng lặp sau khi tạo báo cáo hoàn tất. Không thực hiện đổi account thật, kill LS thật hoặc gửi prompt vào conversation thật để tạo reproduction. GUI build với tag `gui`, kiểm tra race và full staticcheck ngoài U1000 không được chạy trong lượt này; không dùng chúng làm bằng chứng chấp nhận.

## Invariant closure và các giới hạn đã ghi nhận

- **Snapshot membership/data ownership:** đã sweep scanner, summaries resolver, capture wrapper, snapshot serializer/reader và manager caller. Chưa khép do non-UUID, project sai và nhầm cấu hình. Raw byte extraction không có bảo đảm đọc committed rows/WAL; test preview đã chứng minh sai dữ liệu mà không cần giả định lỗi WAL.
- **Identity transition:** đã sweep WinCred writer target, ToS, foreground, kill/sleep/discoverer và client factory/GetUserStatus. Chưa khép do thiếu xác nhận identity sau recycle. Live GetUserStatus PASS của suite chỉ chứng minh API live hoạt động lúc đọc; không chứng minh một lần switch đã nạp target.
- **Disk snapshot as source of truth:** đường đọc file thành công được chứng minh bằng danh sách disk khác memory. Source `account_manager.go:519-523` vẫn bỏ read error và fallback memory khi file thiếu hoặc list rỗng; capture errors cũng bị bỏ. Đây là residual theo pseudocode đặc tả, không được trình bày thành bảo đảm mọi đường lỗi luôn đọc đúng file. Khi sửa F4 cần xử lý/đưa ra trạng thái trung thực cho đường snapshot thất bại.
- **Continuation:** thứ tự, model, pacing và lỗi thread đầu đã khép trong các ranh giới mock được kiểm chứng. Không chứng minh real Antigravity endpoint chấp nhận các cuộc gửi trên nhiều conversation sau một lần đổi account thật.
- **Failure reporting:** recycle error hiện trả `Success=true` với cảnh báo (`account_manager.go:495-503`); client finder error bị bỏ (`:509`). Đây là source behavior cần được xem cùng F1 khi định nghĩa thành công sau đổi account, không được coi là RAM đã xác nhận.
- **Code liveness/ReAuth:** có counterexample xác định cho claim 0 dead code. Không có bằng chứng để tuyên bố toàn workspace sạch dead code; exported API ReAuth còn tồn tại và không có production caller trong source đã sweep.
- **Test isolation:** test 4 bước mới không tự đặt `StoragePath`; vì vậy full-suite invocation của Supervisor chủ động dùng APPDATA/profile tạm. Test vẫn gọi foreground function toàn cục; `rel.WindowFinder` không điều khiển call đó. Các điểm này được ghi nhận để không coi mock window finder là bằng chứng cách ly Win32 hoàn toàn.
- **Temporary working data:** thao tác dọn đệ quy thư mục do lane tạo `.tmp/supervisor-2026-10-18-4-step/` bị automatic approval review chặn với thông báo `blocked by policy`. Lệnh bị chặn chưa thực thi; không tìm cách né chặn hoặc lặp lại xóa. Profile test/fixture/overlay tổng hợp được giữ tại đây; đây là dữ liệu làm việc tạm nằm trong `.gitignore`, không phải thay đổi sản phẩm. Nguồn tái hiện và bằng chứng chính thức vẫn được lưu, commit trong `reports/supervisor/`.

## Danh sách yêu cầu cho lần nộp lại

1. **F1:** thêm gate xác nhận target trong RAM sau recycle và trước continuation; test fresh/mismatch/error/nil-client và trạng thái thành công trung thực.
2. **F2:** bảo đảm project thuộc đúng record/cột `workspace_uris` hoặc fallback có liên kết chắc chắn; dùng fixture SQLite thật với preview nhiễu, nhiều project và WAL.
3. **F3:** lọc UUID nhất quán cho `.db`/`.db-wal`; cập nhật fixture test cho ID hợp lệ và giữ case từ chối file không phải thread.
4. **F4:** cập nhật cấu hình/wrapper/caller cho meanings conversationsDir/summariesDbPath mới; chứng minh cấu hình ToS không phá snapshot; không che lỗi bằng injection bypass wrapper.
5. **F5:** dọn dead helper cùng slice; CEO giao xử lý các tồn dư ngoài diff theo phạm vi 0 dead code toàn workspace và API ReAuth đã nêu. Không dùng “unused imports không lỗi” thay cho bằng chứng dead-code gate.
6. Giao lại commit sửa cùng fresh `go test -v -count=1 ./...` exit 0, không SKIP, các reproduction tương ứng PASS và static-analysis result nêu rõ phạm vi. Giữ regression chứng minh 8-second wait, exact disk snapshot, 3-second pacing, active model và lỗi từng thread đã đạt trong lượt này.

## Đánh giá cuối

**REJECT.** Implementation đã có các phần nominal đúng của phương án 4 bước và suite hiện hữu xanh toàn bộ. Tuy nhiên, bằng chứng source lẫn thực thi độc lập đều phủ định claim đáp ứng đầy đủ đặc tả: thiếu xác nhận identity RAM, snapshot có thể chứa ID/project sai hoặc không được tạo khi cấu hình StoragePath, và dead code vẫn còn. Supervisor chỉ ghi nhận, báo cáo và bàn giao các invariant cần sửa; không thi công sửa sản phẩm trong lane này.
