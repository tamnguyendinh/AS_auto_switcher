# Báo cáo sửa chức năng tự động chuyển tài khoản

Ngày: 2026-10-20, Asia/Bangkok.

Trạng thái: Đã thực hiện bản sửa F1–F8, build CLI/GUI và kiểm tra các package liên quan. Người dùng phản hồi app hoạt động tốt. Chưa tạo commit cho đợt sửa này.

## 1. Phạm vi và căn cứ

- Repository: `G:/antigravity_auto_switch`.
- Git baseline: `3d58dff39a73d1e0dbe4328d451e3f409bf5626a`; bản sửa nằm trong working tree trên baseline này.
- Plan: `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-plan.md`, bổ sung repair slice P2-R.
- Review đầu vào: `reports/supervisor/rp_supervisor_261020_190000_by_gpt-6_autonomous_switch_runtime_review.md`.
- Owner giao sửa các lỗi khiến automation không hoạt động, giữ cơ chế hiện hữu; cho phép thử trên `daohien204ccyersin@gmail.com` và dùng chuyển tay/continuation khi cần.
- Owner yêu cầu thực hiện theo thứ tự **fix trước, test sau**. Production được sửa trước, sau đó build và chạy kiểm tra.

Invariant family: bật tự động → nhận quota hoặc trạng thái thực thi → quyết định retry/chuyển → gọi engine với vòng đời hợp lệ → tiếp tục theo dõi. Tắt tự động phải ngăn công việc mới.

## 2. Các lỗi đã sửa

### F1. Quota thấp nhưng tác vụ chuyển bị hủy ngay sau khi kích hoạt

**Trước:** `OnTelemetryUpdate` tạo context kiểm tra quota, truyền context đó cho goroutine chuyển tài khoản rồi hủy khi hàm cha kết thúc. Vì vậy có thể xuất hiện “đang chuyển” nhưng thao tác không hoàn tất.

**Đã sửa:**

- Tách context kiểm tra quota khỏi vòng đời operation chuyển.
- Gọi operation từ worker callback bằng context của controller, không bằng deadline ngắn của evaluation.
- Khi engine đã bắt đầu thay credential, dùng `context.WithoutCancel` để thao tác OFF hoặc đóng stream không cắt ngang restart, xác nhận identity và continuation.
- Ngăn các lượt evaluation chồng nhau; lỗi operation được đưa ra status.
- Kiểm tra lại identity/model trước khi dùng quyết định proactive; bỏ quyết định cũ nếu tài khoản đã đổi.

Source: `internal/automation/controller.go:302`, `:418`, `:550`.

Regression: `TestQuotaCallbackCompletesEngineAfterEvaluationAndToggleOff`, đi qua `OnTelemetryUpdate` thật đến engine với side effects giả lập, thay vì gọi tắt `TriggerSwitch`.

### F2. Request stream sai header và định dạng Connect-RPC

**Trước:** thiếu `x-codeium-csrf-token`; gửi JSON thô dù khai báo `application/connect+json`. Server có thể trả HTTP 200 nhưng chứa lỗi ở frame kết thúc, và code bỏ qua lỗi đó.

**Đã sửa:**

- Thêm header auth đúng như connector đang hoạt động.
- Dùng `EncodeConnectFrame` đóng gói body trước khi gửi.
- Gửi các page bounds native: `initialStepsPageBounds`, `initialGeneratorMetadatasPageBounds`, `initialExecutorMetadatasPageBounds`.
- Giải mã lỗi trong end-of-stream; chuyển lỗi ra caller/status.
- Decoder hỗ trợ cancellation khi chờ đưa event cho consumer; giới hạn kích thước frame để không cấp phát theo length bất thường.

Source: `internal/connector/automation_rpc.go:339`, `:347`, `:355`; `internal/connector/agent_state_stream.go:31`.

Regression: kiểm tra header, request framing, bounds, HTTP 200 chứa lỗi Connect và cancellation của decoder.

### F3. Decoder đọc sai cấu trúc dữ liệu và tên trạng thái Antigravity

**Trước:** đọc thẳng root JSON vào event nội bộ, trong khi native response bọc trong `update`; các field lỗi, execution và retry nằm ở cấu trúc con. Enum native cũng khác chuỗi được so sánh trong code.

**Đã sửa:**

- Đọc envelope `update`.
- Chuẩn hóa `CASCADE_RUN_STATUS_*` và `EXECUTOR_TERMINATION_REASON_*`.
- Lấy execution ID/termination reason từ `executorMetadata`.
- Lấy lỗi cuối và số bước từ `mainTrajectoryUpdate`.
- Đọc `executionStatus.modelRequest` với `attempt`, `maxAttempts`, `retryAt`.
- Hợp nhất các bản cập nhật từng phần; xóa lỗi cũ khi execution thay đổi hoặc có update xóa lỗi.
- Chỉ giữ dữ liệu trạng thái cần cho automation, không lưu toàn bộ nội dung trajectory/chat.

Source: file mới `internal/connector/agent_state_stream.go`; helpers trong `internal/connector/automation_rpc.go`.

Regression: `TestNativeStreamMergesExecutionPatches`, `TestNativeStreamErrorsAndCancellation`, `TestNativeStreamThroughCoordinatorAndRescan`.

### F4. Mất stream hoặc có thread mới thì không theo dõi tiếp

**Trước:** quét conversation lúc khởi động; worker lỗi thì thoát, không có đường quét lại trong lượt telemetry thông thường. Bật toggle chỉ dùng cache cũ.

**Đã sửa:**

- Nối rescan vào callback telemetry hiện có; các subscription đã kết thúc được tạo lại ở lần quét sau.
- Scanner nhận thêm conversation mới và bỏ subscription không còn trong tập đang theo dõi.
- Bật automation gọi `TriggerNow` của telemetry loop hiện hữu.
- Sau thay session, vô hiệu hóa worker cũ và nối lại bằng discovery mới.
- Các luồng refresh quota toàn bộ/từng tài khoản trong GUI cũng chuyển dữ liệu đến controller.

Source: `internal/automation/retry.go:334`, `:363`; `internal/automation/controller.go:177`, `:198`; `internal/ui/app.go:281`, `:484`, `:534`.

Không tạo daemon hoặc vòng polling quota thứ hai. Cadence reconnect thông thường dựa trên vòng telemetry sẵn có.

### F5. OFF hoặc bắt đầu switch vẫn có thể phát Retry cũ

**Trước:** chỉ reset counter/flag, không hủy worker đang chờ gửi. Coordinator khởi động kể cả khi toggle OFF.

**Đã sửa:**

- Chỉ chạy monitoring/recovery khi controller được bật và không đang chuyển.
- OFF dừng coordinator, hủy context/subscription/recovery đang chờ.
- Worker kiểm tra session generation và recovery hiện hành trước khi gửi.
- Dùng cùng switch gate để ngăn Retry gửi trong lúc auto/manual/CLI đang thay phiên.
- GUI chuyển tay gọi `BeginManualSwitch`/`EndManualSwitch`; kết thúc chuyển tay invalidates session cũ và yêu cầu quota mới.
- Tác vụ engine đã bắt đầu được phép chạy hết; OFF ngăn việc mới.

Source: `internal/automation/controller.go:120`, `:198`, `:212`; `internal/automation/retry.go:144`, `:170`; `internal/ui/app.go:682`.

Regression: `TestControllerOffDoesNotSubscribeOrRetry`, `TestRetryCoordinator_StoppedSupersededAndSessionChangedCancelPendingSend` và callback/OFF test của F1.

### F6. Counter retry dùng chung giữa các conversation

**Trước:** lỗi của nhiều thread cộng dồn vào một counter; completion của thread khác có thể reset recovery. Lỗi gửi Retry bị bỏ qua.

**Đã sửa:**

- Theo dõi recovery theo conversation, cùng execution/step đang chờ.
- Chờ chuyển trạng thái thực thi sau lệnh Retry; không coi HTTP ACK là recovery thành công hoặc thất bại.
- Giữ giới hạn ba recovery actions; duplicate event không tiêu tốn ngân sách.
- Completion của conversation B không reset recovery của A.
- Lỗi gửi/transport được hiện rõ và không tự diễn giải thành hết quota.
- Chờ native retry còn hiệu lực; bỏ lịch delay cố định hai giây của automation.
- Sau đổi session, bỏ worker cũ và không tự coi terminal state được rehydrate là một request lỗi mới.

Source: `internal/automation/retry.go`; `internal/connector/automation_rpc.go`.

Regression: `TestRetryCoordinator_ThreeActionsWaitForExecution`, `TestRetryCoordinator_ConversationOutcomesAreIsolated`, `TestRetryCoordinator_NativeRetryAndNonQuotaErrors`, `TestRetryCoordinator_SendFailureDoesNotBecomeQuotaFailure`.

### F7. Ứng viên lỗi tạm thời bị loại qua mọi đợt

**Trước:** `failedInRound` sống theo controller, chỉ xóa sau switch thành công. Nếu mọi ứng viên từng lỗi, chúng có thể bị loại vĩnh viễn trong phiên app.

**Đã sửa:** danh sách thất bại chỉ tồn tại trong operation hiện tại. Lượt sau với quota mới được xem xét lại.

Source: `internal/automation/controller.go:489`.

Regression: `TestCandidatePreflightFailureExpiresWithOperation`: ứng viên lỗi preflight, sau đó phục hồi, được chọn trong lượt tiếp theo.

### F8. Test giả lập bỏ qua các điểm nối đang hỏng

**Đã sửa:**

- Mock stream phát envelope/schema native, kiểm tra auth/header/request framing.
- Bổ sung test đi qua `OnTelemetryUpdate` thật.
- Bổ sung trường hợp OFF, cancellation, multi-conversation outcomes, send failure, EOF/rescan và candidate phục hồi.
- Test callback GUI chờ điều kiện `lastQuotaRefresh` được callback cập nhật, không tự gán giá trị để làm test pass.
- Thay kiểm tra chỉ gọi trực tiếp `Reset()` bằng coverage qua caller manual hiện có.
- Cô lập GUI test khỏi Language Server của người dùng: scanner/client và quota source giả; snapshot/storage của switch tests dùng đường dẫn thử riêng.
- Điều chỉnh test contention để mong đợi `ErrGateBusy` ngay, thay vì timeout do xếp hàng thao tác cũ.

Source: `internal/automation/runtime_regression_test.go`, `retry_test.go`, `automation_e2e_test.go`, `controller_test.go`; `internal/connector/automation_rpc_test.go`; `internal/ui/automation_binding_test.go`, `ui_test.go` và các caller của helper test.

## 3. Điều chỉnh liên quan trong cùng phạm vi

- **Model chưa resolve được:** thêm `CategoryUnknown`; adapter trả lỗi khi không đối chiếu được model config. Không âm thầm dùng Gemini cho một placeholder chưa xác định.
- **Contention:** wrapper auto/manual/CLI dùng `TryExecute`, trả busy ngay để tránh xếp hàng lệnh chuyển đã cũ. Vẫn dùng named mutex và OS-thread ownership hiện có.
- **Status bar:** dừng timer ẩn cũ khi có trạng thái mới, tránh thông báo “đang chuyển” bị timer trước đó che mất.

## 4. Các phần giữ nguyên

- `AccountManager.SwitchAccount`.
- Thứ tự engine ToS → WinCred → restart Language Server → chờ tám giây → kiểm tra identity → continuation.
- Quét và tiếp tục các thread trong hai giờ; pacing giữa thread hiện hữu.
- Telemetry client/pool/cache/loop, ngoại trừ consumer/binding bên App; không viết lại bộ lấy quota.
- Policy hiện hành: quota 5h <= 2%, quota tuần <= 1%; thứ tự tuần tăng dần → ưu tiên có 5h khi tuần bằng nhau → 5h giảm dần → thứ tự ổn định.
- Layout, theme, tray và công tắc GUI có sẵn.

Đã kiểm tra diff: không sửa source production trong `internal/manager`, `internal/reloader`, `internal/telemetry` hoặc `cmd/switcher` ở đợt repair này.

## 5. Kết quả build và kiểm tra

Artifacts/log chính thức: `reports/qa/automation/2026-10-20-runtime-fix/`.

| Kiểm tra | Kết quả quan sát |
|---|---|
| CLI: `CGO_ENABLED=0; go build -o bin/antigravity-switch.exe ./cmd/switcher` | PASS |
| GUI: `CGO_ENABLED=1; go build -tags gui -ldflags '-H windowsgui' -o bin/antigravity-switch-gui.exe ./cmd/switcher` | PASS |
| `go test -tags gui ./... -count=1 -timeout 120s` | Tất cả package ngoài UI PASS; UI có một test callback chờ cố định 200ms bị FAIL |
| Sau sửa test chờ điều kiện: `go test -tags gui ./internal/ui -count=1 -timeout 120s` | PASS, 17.113s |
| `go test -race ./internal/automation ./internal/connector -count=1 -timeout 120s` | PASS; automation 13.456s, connector 2.235s; không có race được báo trong phạm vi này |

Không mô tả lượt `go test ./...` ban đầu thành PASS: log thất bại được giữ. Sau đó chỉ sửa test GUI và chạy lại package UI; production không đổi sau build. Không chạy race toàn repository hoặc benchmark hiệu năng mới.

E2E verification theo phạm vi evidence:

- [PASS] Compiled: CLI và GUI native.
- [PASS] Mock integration: quota callback → controller → engine với side effects cô lập; native-format stream → decoder → retry coordinator.
- [PASS] Edge cases: OFF/cancel, recovery outcomes, candidate phục hồi, stream lỗi/EOF và rescan, contention; test/race nêu trên.
- [OBSERVED] Native read-only: connector mới đọc đúng session/model/lỗi thật của Antigravity.
- [OWNER FEEDBACK] Người dùng phản hồi “tuyệt vời, ngon lành luôn” sau khi nhận bản sửa.
- [NOT CAPTURED BY ASSISTANT] Một chuỗi GUI bật tự động → đổi identity → continuation hoàn tất có ảnh/log đầy đủ do agent tự thực hiện.

## 6. Quan sát Antigravity thật và giới hạn evidence

Probe đọc bằng connector production sau sửa đã nhận được:

```text
identity=daohien204ccyersin@gmail.com
model=MODEL_PLACEHOLDER_M318
label=Gemini 3.8 Flash (High)
recent_threads=1
status=IDLE
termination=ERROR
error_code=429
has_execution_id=true
is_quota_error=true
```

Đây là bằng chứng adapter đã kết nối và giải mã lỗi quota thật; probe này không gửi Retry, prompt hoặc lệnh switch.

GUI bản build được mở. Công cụ Computer Use không chụp được cửa sổ Fyne và trả `FrameArrived timed out`/`window capture timed out`; accessibility chỉ trả cửa sổ, không có các control con. Vì vậy chưa có ảnh và trace full switch do agent tự thu thập. Phản hồi tích cực của người dùng được ghi riêng, không biến thành số đo hoặc log native E2E do agent chạy.

## 7. Invariant Family Map và các đường liên quan

| Đường xử lý | Thay đổi/kiểm tra | Evidence |
|---|---|---|
| Proactive quota | Context operation, OFF không cắt engine, recheck identity/model | Callback regression, automation tests |
| Reactive 429 | Wire contract native, decoder, ba actions theo execution | Connector tests, stream/coordinator regression, native read-only |
| Toggle/lifecycle | OFF dừng recovery, ON TriggerNow, đóng app cancel monitoring | UI tests, OFF tests |
| Manual GUI/CLI/auto | Shared gate, busy ngay; GUI pause/invalidate recovery | Caller/gate tests; không sửa manager |
| Candidate/preflight | Loại trừ theo từng operation | Recovery-after-preflight regression |
| Session/stream thay đổi | Generation, cancellation, rescan trong loop có sẵn | Cancel/rescan tests |
| Engine/continuation | Preserve-only | Diff không đổi; manager/reloader suite PASS |

Legacy fallback status: bỏ fallback ngầm về Gemini khi config chưa resolve. Không thêm đường ReAuth/daemon/continuation khác.

Residual unverified surfaces: agent chưa thu thập full native GUI auto-switch/continuation trace; race chỉ chạy automation/connector. Báo cáo này không tuyên bố toàn bộ các phần đó đã nghiệm thu.

## 8. Trạng thái bàn giao

- Bản sửa source và binary đã có trong workspace; các lỗi F1–F8 đã có thay đổi xử lý và regression liên quan.
- Người dùng đã phản hồi app hoạt động tốt.
- Báo cáo và companion ledgers ghi đúng phạm vi kiểm tra; chưa đánh dấu đóng toàn bộ plan hoặc có Supervisor PASS sản phẩm khi chưa có evidence tương ứng.
- Git reference: working tree trên `3d58dff`; **chưa commit, chưa push**.
- Lượt viết báo cáo này chỉ đọc diff/log và ghi tài liệu/evidence, không chạy thêm test hoặc đổi tài khoản.
