# Supervisor: xóa UI context timeout và sửa discovery cổng

**Verdict: PASS**

## Phạm vi và thẩm quyền

- Commit sản phẩm đã kiểm định: `0289a73242d60039433273b3093f492cb955e76e`.
- Authority: hợp đồng CEO, đặc tả commit `5a71963`, RCA commit `df65988`, và xác nhận trực tiếp của Owner: “viết báo cáo… xác nhận pass, thông qua”.
- Kiểm chứng source/diff/runtime hoàn tất ngày 2026-10-18 trước 11:57:55, Asia/Bangkok. Báo cáo được chốt sau khi lượt ghi báo cáo trước bị gián đoạn; không chạy lại verification đã hoàn tất và không coi HEAD UI redesign hiện tại là source của các kết quả này.
- Vai trò Supervisor; không sửa mã sản phẩm, không dùng Anvien hoặc subagent. Không tìm thấy AGENTS.md trong checkout/G:\AGENTS.md; hợp đồng và các đặc tả xác định scope.

## Kết quả đối soát

| Yêu cầu | Source và bằng chứng | Kết quả |
|---|---|---|
| Bỏ timeout UI 15s, không thay bằng timeout tổng khác | `internal/ui/app.go:486` truyền `context.Background()` trực tiếp vào manager SwitchAccount; `:501` dùng Background mới cho GetAccountStates | Đạt |
| Hiển thị số phiên tiếp tục | `app.go:509-515` dùng SuccessCount/TotalThreads; UI runtime độc lập có một Send lỗi trên sáu threads hiển thị `• Đã tiếp tục 5/6 phiên` | Đạt |
| Probe 1s và fallback cổng cao nhất | `internal/connector/discovery.go:300,307`; hai nguồn ports được sort tăng dần tại `:189,240` | Đạt |
| Quy trình 4 bước giữ nguyên | 15 leaf scenarios hotswap cũ chạy lại nguyên assertions; trace snapshot → credential → kill → wait8s → discovery → RAM → exact disk snapshot → continuation | Đạt trên các boundary đã kiểm chứng |
| Không cắt context trước thread cuối | Test UI độc lập đi qua manager và default RPC runner với 6 threads, sample 7 context checkpoints không có deadline; refresh finder sau switch cũng không có deadline; sleeper nhận `[8s,3s,3s,3s,3s,3s]`, ngân sách mô phỏng 23 giây | Đạt |

Đã đọc toàn diff `git show 0289a73` của app.go/discovery.go/ui_test.go; source từng tệp thực thi và RCA. Source-level clearance: app.go khép context và summary UI; discovery.go khép probe budget/fallback; ui_test.go thêm test summary và reloader injection ở test scope, không thêm mock vào sản phẩm. Không có blocking finding trong phạm vi được giao.

## Bằng chứng chạy mới tại commit được kiểm định

- `go build ./...`: exit 0.
- `go test -v -count=1 ./...`: **10 package, 141 PASS events, 0 FAIL, 0 SKIP**, exit 0.
- Acceptance overlay: **18 leaf scenarios PASS**, 21 PASS events gồm parent tests, 0 FAIL/SKIP, exit 0. Gồm 15 regression hotswap, 2 scenario discovery và 1 UI context/partial summary.
- `staticcheck@v0.8.2 -checks=U1000 ./...` và `-tests=false -checks=U1000 ./...`: exit 0, **0 diagnostics** trên cấu hình mặc định.
- Discovery dùng Win32 TCP table thật với hai listener thuộc test PID: case HTTP trả sau 400ms hoàn thành sau ~410ms, không early cancellation; case hai probe đều trả non-200 chọn đúng listener cao nhất.
- UI Fyne canvas đã được render, capture và quan sát trực tiếp: status `Đã chuyển sang ui@t.invalid (57ms) • Đã tiếp tục 5/6 phiên` đọc được tại đáy cửa sổ.

Artifact tại [evidence/2026-10-18-purge-ui-timeout](evidence/2026-10-18-purge-ui-timeout/reproduce.ps1): full suite log/JSON, independent acceptance log/JSON, staticcheck log/JSON, nguồn overlay UI/connector, runner tái hiện và [ảnh UI](evidence/2026-10-18-purge-ui-timeout/ui-partial-continuation.png).

## Invariant closure và giới hạn

Context của switch/refresh sau switch, projection summary UI và listener selection đã được kiểm chứng từ source tới runtime; không còn same-invariant surface chưa đóng trong phạm vi sửa này. Các WithTimeout ở Refresh/telemetry riêng không phải context của SwitchAccount. Single-port return ports[0] là đường một phần tử, không phải fallback đa cổng.

Sleep 8s/3s dùng mock sleeper theo contract; không tuyên bố test đã đợi 23 giây thực. Test xác minh context không có deadline, toàn bộ sequence/pacing requests và xử lý đến thread cuối. Capture là Fyne test canvas, không phải screenshot desktop OS. Không đổi account thật/kill LS thật/gửi prompt vào thread thật. Live LS status, Chrome và credential-read tests thực sự PASS. Profile APPDATA/USERPROFILE/store của full-suite process được trỏ thư mục tạm để tránh sửa ToS thật.

Chưa chạy riêng GUI packaging/race hoặc real Antigravity switch; không dùng báo cáo Coder thay bằng chứng cho các bước này. “0 dead code” được ghi theo hai gate U1000 đã chạy, không mở rộng tới mọi build tag/exported API. Không phát hiện regression mới trong các boundary 4 bước đã kiểm chứng.

**PASS — chấp nhận commit `0289a73` trong phạm vi đặc tả purge UI timeout; Owner đã xác nhận thông qua.**
