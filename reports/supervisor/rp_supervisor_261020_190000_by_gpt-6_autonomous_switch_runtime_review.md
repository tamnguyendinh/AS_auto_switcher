# Review automation chuyển tài khoản — lỗi trên đường chạy thực tế

Verdict: REJECT

## Phạm vi và căn cứ

- Ngày review: 2026-10-20, Asia/Bangkok; khảo sát runtime và chẩn đoán trong khoảng 18:52–19:00.
- Reviewer: GPT-6, debugging + supervisor.
- Source: `3d58dff39a73d1e0dbe4328d451e3f409bf5626a`.
- Yêu cầu: tìm nguyên nhân chức năng tự động chuyển tài khoản không hoạt động và xác định phạm vi sửa. Đây là review, chưa triển khai bản sửa.
- Căn cứ: source hiện tại; plan `docs/plans/2026-10-20-autonomous-account-switching/`; các báo cáo P1-B, P1-C, P2-A; giao tiếp đọc trạng thái với Language Server đang chạy; probe cô lập gọi code production.
- Policy hiện tại trong plan/code là 5h <= 2%, tuần <= 1%, theo thay đổi `e1dab85`/`3d58dff`. Review này không thay đổi hoặc mở lại quyết định ngưỡng.
- Không sửa manager, telemetry engine, reloader, GUI hoặc plan; không gửi Retry/prompt, thay credential hay restart Language Server thật.

## Kết luận

Hai nhánh kích hoạt đều có lỗi ở phần automation mới. Proactive dùng context bị hủy ngay khi giao việc chuyển sang goroutine. Reactive gửi request Connect sai header/body, bỏ qua lỗi ở cuối stream và đọc sai cấu trúc phản hồi. Chỉ sửa một nhánh sẽ chưa làm automation hoạt động đầy đủ.

Toggle và callback telemetry đã được nối vào controller; không phải trường hợp thiếu toàn bộ binding GUI. Chưa có căn cứ yêu cầu viết lại engine chuyển tài khoản hoặc continuation.

## Findings cần sửa

### F1 — [P1] Context của kiểm tra quota bị hủy trước khi tác vụ chuyển hoàn tất

Source: `internal/automation/controller.go:260–271`, `internal/automation/retry.go:256–266`, `internal/automation/switch_gate_windows.go:91–110`.

`OnTelemetryUpdate` tạo context timeout 15s và `defer cancel()`. `SupersedeByProactive` chạy `go rc.onSwitchNeeded(ctx, reason)` rồi trả về ngay. Goroutine cha kết thúc và hủy context đang được `TriggerSwitch` sử dụng. Đây không chỉ là timeout sau 15s: cancellation xảy ra ngay sau dispatch. Tùy lịch chạy, switch bị hủy lúc chờ khóa hoặc trong preflight/engine.

Probe gọi đúng `OnTelemetryUpdate`, giữ mutex riêng để quan sát: nhận hai event SWITCHING nhưng `IsSwitching=false` khi mutex vẫn đang được giữ. Chỉ có một lần đọc identity cho evaluation; chưa vào thân switch dưới gate. Kết quả được lưu trong `evidence/autonomous_switch_runtime_review_261020/diagnostic_results.json`.

Hướng sửa: cho operation chuyển sở hữu context đủ sống đến khi engine/continuation kết thúc; không chuyền context ngắn của evaluation sang worker rồi hủy nó. Bảo toàn quy tắc tắt automation ngăn việc mới nhưng không cắt engine đã bắt đầu thay phiên. Trả lỗi qua status thay vì bỏ `_` tại callback.

Kiểm tra lại: đi từ telemetry callback thật qua `OnTelemetryUpdate`, giữ rồi nhả gate, dùng executor cô lập xác nhận đúng một switch hoàn tất. Không chỉ gọi `TriggerSwitch(context.Background())` trực tiếp.

### F2 — [P1] Request stream không đúng Connect-RPC; lỗi server bị nuốt

Source: `internal/connector/automation_rpc.go:360–373`, `:225–226`; đối chiếu helper đã hoạt động `internal/connector/api_client.go:99–101`.

Adapter stream chỉ gửi `X-CSRF-Token`, thiếu `x-codeium-csrf-token`. Body gửi JSON thô dù khai báo `application/connect+json`, thiếu envelope 5 byte. Hàm `EncodeConnectFrame` đã có nhưng không được dùng ở request này.

Đã đối chiếu bốn request đọc cùng một conversation trên LS PID 21944, port 52205 tại thời điểm review (PID/port là tạm thời):

| Request | Phản hồi thực |
|---|---|
| Header/body giống source production | HTTP 200; frame flags=2; `unauthenticated: missing CSRF token` |
| Chỉ bổ sung header cần thiết | HTTP 200; frame flags=2; `invalid_argument: protocol error` do thiếu envelope |
| Bổ sung header và envelope | HTTP 200; frame flags=0; có `update` |
| Header/envelope và page bounds theo native UI | HTTP 200; frame flags=0; có `update` |

Decoder hiện gặp flags=2 là `return nil`, không đọc `error`. Vì vậy HTTP 200 chứa lỗi bị coi là kết thúc bình thường, subscriber rời đi mà không báo lỗi cho GUI.

Hướng sửa: tái dùng header auth hiện có; encode request bằng helper framing đã có; giải mã lỗi Connect trong end-of-stream và phát status có thể quan sát. Dùng các trường bounds native `initialStepsPageBounds`, `initialGeneratorMetadatasPageBounds`, `initialExecutorMetadatasPageBounds`; `pageBounds:10` hiện không phải các trường native đó (server hiện bỏ qua, không phải nguyên nhân chính bị từ chối).

Kiểm tra lại: request qua production adapter nhận được frame dữ liệu từ LS; fixture kiểm tra header, framing và lỗi trong HTTP 200/EOS, không chỉ status code.

### F3 — [P1] Decoder đọc sai envelope, vị trí trường và giá trị enum

Source: `internal/connector/automation_rpc.go:79–89`, `:117–144`, `:229–232`.

Phản hồi native là `{"update": {...}}`. Code unmarshal thẳng vào `AgentStateUpdate`, nên JSON decode có thể không báo lỗi nhưng status/ID/error đều rỗng. Probe chạy decoder production với projection đã loại bỏ dữ liệu riêng tư của frame native xác nhận `status=""`, không nhận completion.

Các khác biệt còn lại sau khi bỏ lớp `update`:

- Native status là `CASCADE_RUN_STATUS_IDLE`/`CASCADE_RUN_STATUS_RUNNING`; helpers chỉ so với `IDLE`/`RUNNING`.
- Native UI lấy termination reason từ `executorMetadata.terminationReason`, không phải root `terminationReason`.
- Dữ liệu trajectory đi qua `mainTrajectoryUpdate`; native UI hợp nhất nó rồi lấy `trajectorySlice.lastStepError`.
- Native `executionStatus` là cấu trúc trạng thái, có nhánh `modelRequest` với `attempt`, `maxAttempts`, `retryAt`; DTO hiện khai báo `ExecutionStatus string`, `Attempt`/`RetryAt` ở root và không có `maxAttempts`.

Hướng sửa: adapter chuẩn hóa frame native sang event nội bộ; giải mã đúng trường, enum và update cần hợp nhất. Giữ việc quyết định retry trong coordinator; không cần viết lại cơ chế tiếp tục thread.

Kiểm tra lại: fixture từ native schema gồm running, terminal error, completion và native retry waiting đi qua decoder production rồi mới tới coordinator. Frame malformed phải hiện lỗi, không im lặng bỏ qua.

### F4 — [P1] Subscription chỉ mở lúc khởi động, thiếu cập nhật và nối lại

Source: `internal/automation/retry.go:271–294`, `:335–389`; `internal/automation/controller.go:149–165`, `:256–271`, `:470–473`; `internal/ui/app.go:298–310`.

`Start` quét danh sách conversation một lần. Nếu lúc đó IDE chưa sẵn sàng, request stream lỗi, hoặc về sau mở thread mới, worker chỉ return và xóa entry. Không có reconnect. Call site production duy nhất của `RescanConversations` nằm sau auto switch thành công; không được gọi khi bật hoặc cùng lượt telemetry. Manual switch cũng không mở lại subscriptions sau LS restart.

Hướng sửa: gắn rescan/reconnect vào vòng telemetry hiện có, lúc bật và sau thay session; sử dụng discovery lại khi kết nối chết. Không thêm daemon hay ticker quota thứ hai. Bật toggle gọi `TriggerNow` hiện có để lấy quota mới như plan; hiện `SetEnabled` chỉ đọc cache.

Kiểm tra lại: mở GUI trước IDE; mở thread mới sau GUI; stream EOF; manual/auto restart; tất cả phải được theo dõi lại mà không phải đóng/mở Switcher.

### F5 — [P1] OFF và proactive supersede chưa ngăn gửi Retry

Source: `internal/automation/controller.go:149–165`, `:214–215`; `internal/automation/retry.go:153–156`, `:229–245`, `:256–266`.

App khởi động retry coordinator dù automation mặc định OFF. `HandleAgentStateUpdate` chỉ kiểm tra `running`, không kiểm tra enabled hoặc đang switch. OFF gọi `Reset`, không dừng coordinator hay worker chờ gửi. `SupersedeByProactive` cũng chỉ reset biến, không hủy Retry đã dispatch; goroutine sau sleep chỉ kiểm tra `running`.

Probe cô lập: controller `enabled=false`, nhận một event giới hạn hợp lệ, vẫn gửi **1 Retry** tới HTTP server giả. Không gửi gì tới IDE thật. Hiện F2/F3 che bớt lỗi này; sau khi sửa stream, lỗi điều khiển OFF sẽ lộ rõ hơn.

Hướng sửa: lifecycle/gate gửi Retry phải tuân thủ enabled và operation hiện hành; invalidation khi OFF, proactive hoặc manual thay session phải chặn worker cũ trước send. Giữ engine đang thay phiên chạy hết. Không chỉ reset counter.

Kiểm tra lại: OFF ngay từ đầu, OFF trong thời gian chờ, proactive khi Retry đang chờ, manual switch; không có Retry mới ngoài operation được phép.

### F6 — [P1] Ngân sách ba lần recovery chưa gắn với kết quả của đúng recovery

Source: `internal/automation/retry.go:30–36`, `:160–173`, `:184–219`, `:240–245`; `internal/connector/automation_rpc.go:133–134`.

`recoveryAttempts` là một counter chung cho mọi conversation. Mỗi event giới hạn có key mới tăng counter, còn completion của bất kỳ conversation nào reset counter. Ví dụ A đang fail, B hoàn tất thì ngân sách của A về 0; nhiều thread fail đồng thời cũng cộng dồn thành ngân sách chung dù chưa có ba recovery actions thất bại của cùng đợt. Code không gắn terminal outcome với lệnh Retry vừa gửi; lỗi send bị bỏ qua nhưng event đã được đánh dấu seen, có thể kẹt recovery nếu không có execution mới.

`HasRetryAt` chỉ kiểm tra chuỗi khác rỗng, không dựa vào native modelRequest/attempt/maxAttempts. Đây cũng là phần phải sửa cùng adapter F3.

Hướng sửa: giữ context tối thiểu của đợt recovery (session/conversation/execution đang chờ), chỉ dùng outcome tương ứng để tăng hoặc reset số lần; xử lý lỗi send thành trạng thái rõ ràng. Không cần thiết kế bộ chọn thread continuation mới. Continuation hai giờ giữ nguyên.

Kiểm tra lại: A fail trong khi B thành công; A/B cùng báo lỗi; duplicate event; event phiên cũ sau switch; Retry transport failure; native retry đang chờ.

### F7 — [P1] Ứng viên lỗi một lần bị loại qua mọi đợt tiếp theo

Source: `internal/automation/controller.go:388–393`, `:420–438`, `:443–445`, `:458`, `:466–468`.

`failedInRound` là field sống theo controller. Mỗi trigger copy lại danh sách cũ, nhưng chỉ xóa sau switch thành công. Nếu mọi ứng viên từng lỗi preflight, lần sau chúng bị lọc hết trước cả PollAccount, dù telemetry đã báo quota/token dùng được trở lại. Không có switch thành công để xóa blacklist: ứng dụng mắc kẹt “không có ứng viên” cho đến khi controller được tạo lại.

Hướng sửa: failed set thuộc một đợt lựa chọn/chuyển, không thuộc toàn vòng đời app; đợt sau với telemetry mới được đánh giá lại. Không cần thêm cooldown hoặc hệ thống chặn tài khoản mới.

Kiểm tra lại: ứng viên duy nhất lỗi preflight tạm thời, sau đó telemetry tốt; trigger tiếp theo phải thử lại được.

### F8 — [P1 đối với nghiệm thu] Test hiện tại đi vòng qua chính các điểm bị lỗi

Source: `internal/automation/automation_e2e_test.go:80–123`, `:595–627`; `internal/connector/automation_rpc_test.go:239–283`; `internal/ui/automation_binding_test.go:52–94`, `:132–167`.

- Case quota thấp gọi `EvaluateProactiveTrigger` rồi trực tiếp `TriggerSwitch(ctx)` với context sống lâu, không qua callback `OnTelemetryUpdate` đang bị F1.
- Mock stream phát struct nội bộ phẳng, không phải envelope/schema native; không kiểm tra header CSRF hay envelope request. Vì vậy F2/F3 không thể bị bắt bởi những fixtures này.
- Test dual callback đọc `lastQuotaRefresh` rồi bỏ biến bằng `_ = lastRefresh`, không assert UI cập nhật.
- Test manual recovery tự gọi `rc.Reset()`, không chứng minh đường `App.SwitchAccount` đã làm việc đó.

Hướng sửa: sửa đúng các điểm vào/fixture kể trên, dùng fake cho side effects nhưng vẫn đi qua callback/adapter/caller production. Có thể giữ test helper thuần hữu ích; không dùng số 14/14 làm bằng chứng toàn luồng đã chạy với Antigravity thật.

Kiểm tra lại: regression phải thất bại trên source cũ ở F1–F7 và pass sau sửa; quan sát GUI native tách khỏi mock. Các báo cáo PASS P1-B/P1-C/P2-A hiện không đủ để nghiệm thu chức năng thực tế.

## Lưu ý bổ sung, không phải nguyên nhân chính đã tái hiện

- `ResolveActiveModel` (`automation_rpc.go:273–275`) trả nil error khi đọc config lỗi; classifier (`model_quota.go:73`) mặc định Gemini nếu không nhận diện được label. Khi model là placeholder Claude mà config thất bại, automation có thể dùng nhầm nhóm quota. Giữ resolver cũ nhưng adapter/controller phải báo chưa resolve được và không tự coi Gemini là kết quả xác nhận.
- Shared gate có mutual exclusion, nhưng `Execute` chờ tuần tự; auto/manual có thể xếp hàng lệnh dựa trên quyết định cũ. Khi sửa operation, cần bỏ hoặc kiểm tra lại trigger sau contention theo plan; không chỉ chứng minh hai engine không chạy đồng thời.
- Actual-status có bảng đánh dấu P1/P2-A hoàn tất nhưng header/next-action vẫn ghi implementation chưa bắt đầu; evidence E2 còn PLANNED, benchmark còn ngưỡng tuần cũ. Cần đồng bộ sau sửa, không lấy ledger cũ làm bằng chứng runtime.

## Source-level clearance và phạm vi giữ nguyên

| Nhóm source | Kết quả review |
|---|---|
| `ui/window.go`, `ui/app.go` | Toggle -> setter -> controller và callback telemetry cùng UI refresh đã có. Cần sửa lifecycle/TriggerNow/status theo F1/F4/F5; không dựng UI khác. |
| `automation/controller.go` | Blocked F1/F4/F7; đã có chọn ứng viên và gọi manager. |
| `automation/retry.go` | Blocked F4/F5/F6; reset biến chưa đồng nghĩa hủy tác vụ. |
| `connector/automation_rpc.go` | Blocked F2/F3; model config reader hoạt động với response hiện tại trong probe đọc. |
| `automation/model_quota.go`, `sorter.go` | Có ngưỡng hiện tại, nhánh weekly-only/disabled và thứ tự tuần tăng -> có 5h -> 5h giảm. Không có căn cứ viết lại sorter; xử lý unknown model ở adapter/caller. |
| `telemetry/client.go`, `pool.go`, `cache.go`, `loop.go` | Metadata disabled có đường truyền; PollAccount/callback/TriggerNow/loop sẵn có để tái dùng. Không thêm pipeline fetch. |
| `switch_gate_windows.go`, GUI/CLI callers | Có shared mutex, OS thread affinity và wrapper. Cần kiểm tra operation sau chờ; không thay bằng khóa riêng rời rạc. |
| `manager/account_manager.go`, `reloader/` | Bảo lưu engine, wait 8s và continuation hai giờ. Không có thay đổi sản phẩm trong review. |
| Tests automation/connector/UI | Blocked F8; test mô phỏng không chứng minh wire contract hiện tại. |

## Evidence đã kiểm tra

Fresh, trong lượt review này:

- Đọc source và call sites; HEAD ghi ở metadata.
- Read-only native GetCascadeModelConfigData, GET `/main.js`, StreamAgentStateUpdates; không gọi SendUserCascadeMessage.
- `stream_probe_sanitized.json`: bốn biến thể request cùng kết quả, chỉ giữ dữ liệu kỹ thuật đã loại bỏ token/identity/chat.
- `native_frame_redacted.json`: projection của frame native, giữ shape/trạng thái để kiểm tra decoder; không phải bản chụp đầy đủ response.
- `probe.go` + `diagnostic_results.json`: gọi production decoder/controller với mock HTTP, store rỗng và mutex riêng. Xác nhận swallow EOS error, decode mất status, OFF vẫn Retry và proactive kết thúc khi chờ gate. Đây là diagnostic reproduction, không phải acceptance suite.
- `git diff -- internal cmd scripts`: không có sửa source sản phẩm trong lượt này.

Không dùng làm bằng chứng:

- `anvien analyze --force` thất bại với `db_load phase: copy relationships Package->Function: schema pair unsupported`. Review tiếp tục bằng source/call sites và runtime trực tiếp; không kết luận từ index lỗi. Đã dọn output index tạo trong lượt và bỏ block `.gitignore` do command này tự sinh.

Not run:

- Không chạy lại full build hoặc toàn bộ `go test`/race/benchmark.
- Không bật automation hoặc gửi native Retry/prompt để gây chuyển tài khoản thật; chưa nghiệm thu full GUI -> live switch -> continuation.

Lệnh tái lập probe cô lập từ repo root (chỉ chạy diagnostic, không đổi tài khoản):

```powershell
go run ./reports/supervisor/evidence/autonomous_switch_runtime_review_261020/probe.go ./reports/supervisor/evidence/autonomous_switch_runtime_review_261020/native_frame_redacted.json
```

## Thứ tự khắc phục đề nghị

1. Sửa F1 và báo lỗi operation để nhánh quota thấp thực sự gọi hết engine.
2. Sửa F2/F3 cùng nhau để reactive nhận đúng trạng thái thật.
3. Sửa F4/F5/F6: subscription, OFF/supersede/manual invalidation và recovery outcome.
4. Sửa F7 để tài khoản từng lỗi tạm thời được xem xét lại.
5. Sửa tests theo F8, kiểm tra GUI/native boundary tương ứng, cập nhật các ledgers rồi review lại.

Các thay đổi tập trung ở automation, adapter và binding mỏng hiện hữu. Không cần mở thiết kế automation mới, daemon mới, thay engine hoặc thay cách tiếp tục các thread hai giờ.

## Invariant closure

Invariant cần đóng: bật automation -> nhận đúng quota/model hoặc lỗi thực thi -> chọn/retry đúng -> gọi đúng một operation switch -> engine và continuation hoàn tất -> tiếp tục theo dõi; OFF ngăn tác vụ mới.

Đã đối chiếu hai trigger, wire protocol, decoder, coordinator, candidate filter, GUI/CLI caller và test fixtures. Invariant hiện chưa đóng do F1–F8; full live switching chưa chạy. Verdict REJECT áp dụng cho khả năng nghiệm thu automation hiện tại, không phủ nhận phần engine chuyển tay đang có.
