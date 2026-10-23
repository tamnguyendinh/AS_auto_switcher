# R5 command and runtime evidence

Reviewed source: `28a7e3e4594550c394885b8502293d52a765d167`, Windows/amd64. Date: 2026-10-18, Asia/Bangkok. Initial HEAD matched target and working tree was clean.

| ID | Command / artifact | Result |
|---|---|---|
| E00 | `git show --format=fuller 28a7e3e4594550c394885b8502293d52a765d167 -- internal/ui/quota_badge.go internal/telemetry/quota_test.go internal/ui/ui_test.go` | `E00-reviewed-diff.patch`; production deletion and two added regression tests inspected. |
| E01 | `go version`, `go env GOOS GOARCH CGO_ENABLED CC`, `gcc --version`, `staticcheck -version` | `E01-toolchain.log`; Go 1.26.3, Windows/amd64, CGO=1, GCC 16.1.0, staticcheck 2026.2.2. |
| E02 | `go build ./...` | Exit 0; 5569 ms. |
| E03 | `go build -tags gui -ldflags -H=windowsgui -o G:\antigravity_auto_switch\.tmp\supervisor-ui-r5-gui.exe ./cmd/switcher` | Exit 0; 8755 ms; temporary binary removed by runner. |
| E04 | `go test -v -count=1 ./...` | Exit 0; 29218 ms; 10 packages, 163 PASS (144 top-level + 19 subcase), 0 FAIL, 0 SKIP. Full log and summary JSON retained. |
| E05 | `staticcheck -tags gui ./...` | Exit 0; 2254 ms; 0 diagnostic. |
| E06 | `go run reports/supervisor/evidence/2026-10-18-ui-redesign-r5/runtime_probe.go reports/supervisor/evidence/2026-10-18-ui-redesign-r5/quota_matrix.go reports/supervisor/evidence/2026-10-18-ui-redesign-r5/explicit_cycles.go` | Exit 1; 13397 ms; final run started 17:24:21 +07. 238/256 assertions pass; 18 F1 failures, no F2–F6 failure. Full JSON, stderr, command metadata, summary and PNGs retained. |
| E07 | Production card observations from E06 | 20 explicit-cycle fixtures, 60 observations; 51 match, 9 fail. JSON/Markdown table and card PNG retained. All failures are the three legacy-positive fixtures at normalized input and two disk reload stages. |
| E08 | `rg` searches of retired helper declarations/calls, retired fields and GPT field consumers | No matches; search exit 1 is expected for absent symbols. Patterns, roots and exclusions recorded in `E08-source-searches.json`. Runtime AST/reflection provide additional evidence. |
| E09 | `rg -n` source anchors in `internal/ui` and `internal/telemetry/client.go` | `E09-source-map.txt`; exact extractor, fallback, card, contract, refresh, layout and theme locations. |

Gate runner: `run_gates.ps1`. Runtime runner: `run_runtime_probe.ps1`. Readable observation export: `summarize_probe.ps1`. Standalone Go probes use `//go:build ignore`; no product source/test file was changed. Historical Supervisor evidence is unchanged.

The R4 Supervisor harness was read and rerun as new R5 evidence, with an additional independent zero-cycle/card matrix; Coder probe output was not used. The inherited matrix passes 79/79 observations. Three legacy-positive fixtures fail 9 observations: Claude stored `(0,0)` renders `(0.2,-1)`, Gemini stored `(0,0)` renders `(0.8,-1)`, and both fail together. Positive controls with a reset/raw bucket retain zero, isolating the UI heuristic.

PNG captures use Fyne's software canvas, not native OS capture. E07 card order is described in `E07-explicit-cycle-observations.md`. The final E07 and Light PNGs were visually inspected. An earlier owned E07 capture had mismatched test app theme/palette; the harness theme was aligned and only the final run output is retained. Full build/test/staticcheck were not repeated after this evidence-only correction.

Temporary account/cache/preferences data stayed in a validated direct child of workspace `.tmp` and were cleaned by the probe. Synthetic fetcher delay was 50 ms; measured completion times are not live provider latency. The report records the remaining F1 defect and acceptance verdict.
