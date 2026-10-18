# R6 independent Supervisor evidence

Reviewed commit: `383f56b03ec88f720594ace8f03645695f568b81`.
Validation date: 2026-10-18, Asia/Bangkok (UTC+07).

All results were gathered fresh against the target source. Historical R5 evidence
and Coder evidence were not modified. The R6 harness retains the 256 R5 assertions,
adds 270 scalar/cycle assertions, and compares theme colors to fixed hex values.
The Go evidence programs use `//go:build ignore` and do not join production builds,
the normal test suite, or staticcheck.

| ID | Evidence | Result |
|---|---|---|
| E00 | `git show --format=fuller --stat --patch 383f56b03ec88f720594ace8f03645695f568b81` → `E00-reviewed-diff.patch` | Source and test changes inspected |
| E01 | `go version`, `go env GOOS GOARCH CGO_ENABLED CC`, `staticcheck -version`, `gcc --version` | Go 1.26.3, Windows/amd64, CGO=1, GCC 16.1.0, staticcheck 2026.2.2 |
| E02 | `go build ./...` | Exit 0, 4494 ms |
| E03 | `go build -tags gui -ldflags -H=windowsgui -o .tmp/supervisor-ui-r6-gui.exe ./cmd/switcher` | Exit 0, 7928 ms; temporary binary removed by runner |
| E04 | `go test -v -count=1 ./...` | Exit 0, 24599 ms; 10/10 packages, 165 PASS (146 top-level + 19 subtests), 0 FAIL, 0 SKIP |
| E05 | `staticcheck -tags gui ./...` | Exit 0, 1844 ms; 0 diagnostic |
| E06 | Independent Fyne/parser/cache/HTTP probe | Exit 0, 15638 ms; 526/526 assertions, no mismatches; stderr empty |
| E07 | Explicit-cycle JSON and Markdown observations plus PNG | 65 fixtures × 3 stages = 195/195 matching observations; all 18 former R5 failures pass |
| E08 | Source searches in production files | No positive override, GPT field consumers, retired functions or retired fields; `rg` exit 1 means no match |
| E09 | Numbered current source map | Production paths and changed tests supporting the reviewed invariants |
| E10 | `E10-compact-1200.png` | Visually inspected 1200×750 Fyne canvas, three-column compact cards |

`command-results.json` contains each build/test/staticcheck command, start time,
duration and exit code. `E06-runtime-command.json` records the independent probe
command and time (17:53:15.624 +07; 15.638 seconds).

Reproduce from the repository root in PowerShell:

```powershell
& reports/supervisor/evidence/2026-10-18-ui-redesign-r6/run_gates.ps1
& reports/supervisor/evidence/2026-10-18-ui-redesign-r6/run_runtime_probe.ps1
& reports/supervisor/evidence/2026-10-18-ui-redesign-r6/summarize_probe.ps1
```

The second script runs:

```powershell
go run reports/supervisor/evidence/2026-10-18-ui-redesign-r6/runtime_probe.go reports/supervisor/evidence/2026-10-18-ui-redesign-r6/quota_matrix.go reports/supervisor/evidence/2026-10-18-ui-redesign-r6/explicit_cycles.go
```

The explicit matrix retains all 20 R5 fixtures, adds five directed controls, then
crosses five legacy scalars (`-1, 0, 0.2, 0.8, 1`) with eight cycle tuples. Each
fixture is checked immediately after production normalization and after two saves
and reloads into fresh cache instances. Assertions compare both the extracted
tuple and the actual production card's 5h/week text rows to the contract oracle.
Directed controls cover null fractions, camelCase, conflicting raw buckets, and
raw supplementation of missing cycles while retaining explicit zero.

The separate inherited HTTP/cache matrix has 33 fixtures and 79/79 matching
observations. It covers empty/missing/null fractions, amount-only REST, legacy
single-cycle fallback, missing families, opaque bucket IDs, zero and nonzero cycles.

Images are Fyne software-canvas captures, not native Windows screenshots. All four
images were visually inspected. Accounts, tokens, stores and HTTP in the probe are
synthetic; HTTP is loopback. Temporary probe storage stays in the repository's
`.tmp` directory and is cleaned by the runner. Native Windows GUI compilation is
proven separately by E03. No claim is made about native OS hover/minimize behavior
or live-provider latency.
