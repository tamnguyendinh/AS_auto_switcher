# R4 command and runtime evidence

Target: `5996891479f8271bfdbe5822f57dbd81b7161d0f`. Fresh review on 2026-10-18, Asia/Bangkok (UTC+07). Product source and product tests were not edited. Historical evidence was retained unchanged.

| ID | Command / evidence | Result |
|---|---|---|
| E00 | `git show --no-ext-diff --format=fuller 5996891479f8271bfdbe5822f57dbd81b7161d0f` | Full reviewed patch in `E00-reviewed-diff.patch`; initial HEAD equalled target and working tree was clean. |
| E01 | `go version`; `go env GOOS GOARCH CGO_ENABLED`; `gcc --version`; `staticcheck -version` | Go 1.26.3, windows/amd64, CGO=1; gcc 16.1.0; staticcheck 2026.2.2 (0.8.2). |
| E02 | `go build ./...` | Exit 0; 4398 ms. |
| E03 | `go build -tags gui -ldflags -H=windowsgui -o .tmp/supervisor-ui-r4-gui.exe ./cmd/switcher` | Exit 0; 10687 ms. Temporary binary removed by runner. |
| E04 | `go test -v -count=1 ./...` | Exit 0; 34808 ms; 10/10 package PASS, 161 PASS (142 top-level + 19 subcases), 0 FAIL, 0 SKIP. Full output: `E04-go-test-verbose.log`; counts: `E04-go-test-summary.json`. |
| E05 | `staticcheck -tags gui ./...` | Exit 0, 0 diagnostics; 3212 ms. E02–E05 exit/timing data: `command-results.json`. |
| E06 | `go run reports/supervisor/evidence/2026-10-18-ui-redesign-r4/runtime_probe.go reports/supervisor/evidence/2026-10-18-ui-redesign-r4/quota_matrix.go` | Exit 1; 16982 ms. Completed 33 F1 fixtures / 79 observations: 75 matched, 4 mismatched. F2/F3/F4/F5/F6 assertions had no failure. |
| E07 | `run_rpc_probe.ps1` → `go test -v -count=1 -overlay .tmp/supervisor-r4-rpc-overlay.json ./internal/telemetry -run ^TestSupervisorR4ActualRPCPoolCache$` | Exit 0. Seven subcases PASS, no skip: empty, missing, full, Claude weekly zero, Gemini weekly zero, both-zero without reset, null. Real GetUserStatus → RetrieveUserQuotaSummary → PollAccount → disk reload, with loopback HTTP. |
| E08 | Scoped `rg` searches in `internal/ui`, `cmd`, `internal` | Both searches exit 1 with 0 matches: retired helper calls/declarations and GPT field consumers/literals. This is the normal `rg` result for absence. Full patterns and outputs: `E08-source-searches.json`. AST/reflection is also in E06. |
| E09 | Source map from current UI and telemetry | `E09-source-map.txt`, plus direct source inspection at target. |
| E10 | `go run reports/supervisor/evidence/2026-10-18-ui-redesign-r4/zero_cycle_card_probe.go` | Exit 1 from two genuine mismatches; two positive controls matched. Public parser → typed cache update/save/load → actual production account card. Output: `E10-zero-cycle-cards.json`; viewed image: `E10-zero-cycle-cards.png`. |

E06 raw observations and badge/storage values are in `E06-runtime-probe.json` and `E06-quota-observations.json`; a compact complete table is in `E06-quota-observations.md`. The four mismatches are Claude/Gemini explicit both-cycle zero with no raw buckets, on initial disk load and save/reload. Disk cycle fields remain correct; UI drops the weekly zero. This is distinct from the repaired weekly-only zero case.

Viewed images: `E06-headless-dark.png`, `E06-headless-light.png`, `E10-zero-cycle-cards.png`. These are Fyne software canvas captures, not native Windows screenshots. Main probe uses actual application layout, theme selector, refresh buttons, pool and cache with isolated synthetic accounts. RPC probe only seeds discovery-cache data in the synthetic pool; identity, CSRF, HTTP requests, parser and persistence are production code.

Reproduction runners: `run_gates.ps1`, `run_runtime_probe.ps1`, `run_rpc_probe.ps1`. The Go main probes have `//go:build ignore`; the RPC source is a `.go.txt` file injected only by a temporary Go overlay. No probe is part of the normal product build or suite. No Anvien or additional agent was used.

No repeat of green gates after recording documentation/evidence: product state did not change. Probe exit 1 denotes the recorded F1 mismatches, not a panic, build error or external blocker.
