# Evidence R3 — a3c0cdb1748bb833f7d0be54d9148da66d452f94

Run: 2026-10-18, Asia/Bangkok (UTC+07). Owner: Supervisor lane. Không sửa product sources, không chạy Anvien, không mở subagent. Các file Go `.go` ở đây dùng `//go:build ignore`; RPC fixture `.go.txt` chỉ được nạp bằng overlay. Evidence R1/R2 được giữ nguyên.

| ID | Command / artifact | Kết quả |
|---|---|---|
| E00 | `git show --no-ext-diff --format=fuller a3c0cdb` → `E00-reviewed-diff.patch` | Đối soát commit, đọc touched production code trước validation. |
| E01 | `go version`, `go env GOOS GOARCH CGO_ENABLED`, `gcc --version`, `staticcheck -version` | Go 1.26.3, Windows/amd64, CGO=1, staticcheck 2026.2.2. |
| E02 | `go build ./...` | Exit 0, 4854 ms. |
| E03 | `go build -tags gui -ldflags '-H=windowsgui' -o .tmp/supervisor-ui-r3-gui.exe ./cmd/switcher` | Exit 0, 8713 ms; binary tạm được xóa. |
| E04 | `go test -v -count=1 ./...` | Exit 0, 30047 ms; 10/10 package, 135 top-level + 19 subcase = 154 PASS, 0 FAIL, 0 SKIP. Log đầy đủ và summary JSON được giữ. |
| E05 | `staticcheck -tags gui ./...` | Exit 0, 0 diagnostic, 1982 ms. Output thành công rỗng; exit/count ghi trong `command-results.json`. |
| E06 | `go run reports/supervisor/evidence/2026-10-18-ui-redesign-r3/runtime_probe.go reports/supervisor/evidence/2026-10-18-ui-redesign-r3/quota_matrix.go` | Exit 1, 14718 ms, hoàn tất JSON/PNG. F1 matrix 41 observations: 33 match, 8 mismatch; F2/F3/F5/F6 và named F4 removal được quan sát riêng. |
| E07 | `run_rpc_probe.ps1` → additional `_test.go` Go overlay, `go test -v -count=1 -overlay .tmp/supervisor-r3-rpc-overlay.json ./internal/telemetry -run '^TestSupervisorR3ActualRPCPoolCache$'` | Exit 1: empty/missing/full PASS, weekly zero mất cycle sau reload FAIL. Overlay tạm được xóa. |
| E08 | `go run golang.org/x/tools/cmd/deadcode@v0.52.0 -tags gui ./cmd/switcher`; scoped caller search `internal`, `cmd` | Scan exit 0, còn candidates. Chỉ dùng named UI retired helpers làm finding; không coi mọi test seam/candidate là lỗi. |
| E09 | Source map, contract consumer sweep `.GPT\b`, `GPT ModelQuotaBucket`, `GPT:` trong `cmd`, `internal` | 0 GPT field/access/literal match; source line references được giữ. |
| E10 | `go run reports/supervisor/evidence/2026-10-18-ui-redesign-r3/fallback_presence_probe.go` | Exit 1: REST amount dương nhưng thiếu fraction → 100% cả trước/sau cache, hai dạng camel/snake case. Chạy bổ sung để khép điều tra đường fallback F1 chưa đạt, không chạy lại các gate đã đạt. |

## Reproduction

Từ repository root, chạy `run_gates.ps1`, `run_runtime_probe.ps1`, `run_rpc_probe.ps1` trong thư mục evidence này; E10 chạy bằng command trong bảng. Các runner sẽ tạo lại output **trong chính thư mục R3**. Giữ bộ evidence lịch sử bất biến: khi re-review sau fix, sao chép harness sang thư mục vòng mới, đổi output paths rồi chạy; không ghi đè R3.

`runtime_probe.go` được phát triển từ scaffolding Supervisor R2 sang bản R3, bỏ tham chiếu field GPT đã bị xóa và bổ sung matrix độc lập ở `quota_matrix.go`; output hiện tại đều được tạo mới trên target. Không dùng output Coder hoặc output R2 làm bằng chứng PASS.

## Runtime boundaries

- E06: Fyne software canvas, synthetic accounts/cache/OAuth; REST HTTP loopback qua QuotaClient và TelemetryPool thật. Refresh đi qua button sản phẩm, layout qua window resize, theme qua Select.
- E07: discovery cache của pool được seed đến server loopback; `GetUserStatus`, POST/CSRF, summary parser và cache save/reload là sản phẩm. Không thay parser/cache bằng mock.
- E10: nhánh legacy REST amount được gọi qua client/pool/cache sản phẩm đến badge thật.
- Hai PNG E06 đã được xem trực tiếp; không phải capture Windows native. Native GUI gate trong scope là build.
- Probe và GUI binary chỉ dùng `.tmp` cô lập và cleanup target đã kiểm tra nằm trong repository. Không ghi credential thật hoặc chuyển tài khoản Owner.

Full suite E04 có một số log tài khoản/PID từ live discovery/read test sẵn có; output không có token. Các test/probe fail được giữ làm bằng chứng active của REJECT, không phải artifact retry bị bỏ.
