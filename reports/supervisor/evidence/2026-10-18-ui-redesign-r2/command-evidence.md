# Command evidence — Fyne GUI Supervisor R2

Review target: `6b5cfe3037b114bf92782f313658008b034f45af`. Review date: 2026-10-18, Asia/Bangkok. Production source remained unchanged throughout the review. Official verdict and required fixes are in [the R2 report](../../2026-10-18-ui-redesign-r2-supervisor-report.md).

| ID | Evidence | Result / purpose |
|---|---|---|
| E00 | [Reviewed diff](E00-reviewed-diff.patch) | Fresh `git show` of target production and test changes; read before validation. |
| E01 | [Toolchain](E01-toolchain.log) | Go 1.26.3 Windows/amd64, CGO=1, gcc; staticcheck 2026.2.2 (0.8.2). |
| E02 | [Full build](E02-build.log) | `go build ./...`, exit 0. |
| E03 | [Native GUI build](E03-native-gui-build.log) | `-tags gui -ldflags '-H=windowsgui'`, exit 0. Disposable binary removed after validation. |
| E04 | [Verbose tests](E04-go-test-verbose.log), [summary](E04-go-test-summary.json) | `go test -v -count=1 ./...`; 10 packages, 153 PASS, 0 FAIL, 0 SKIP, exit 0. |
| E05 | [Staticcheck](E05-staticcheck.log) | `staticcheck -tags gui ./...`, exit 0, zero diagnostics. |
| E06 | [Runtime JSON](E06-runtime-probe.json), [command](E06-runtime-command.log), [stderr](E06-runtime-stderr.log), [Dark](E06-headless-dark.png), [Light](E06-headless-light.png) | Independent probe output. Happy path and F3/F5/F6 pass; F1 missing/zero/legacy and F2 type checks expose remaining defects. Software-canvas images visually inspected; not native OS screenshots. |
| E07 | [Actual RPC pool/cache probe](E07-rpc-pipeline.log) | Additional overlay test calls product HTTP/RPC/pool code against loopback and reloads persisted quota; PASS. |
| E08 | [Reachability scan](E08-deadcode.log) | `go run golang.org/x/tools/cmd/deadcode@v0.52.0 -tags gui -filter antigravity-switch ./cmd/switcher`; exit 0 with 62 candidates. Scan exit is not a zero-candidate claim; report classifies in-scope retired UI separately from seams and outside-scope inventory. |
| E09 | [Legacy UI runtime JSON](E09-legacy-ui-probe.json), [stderr](E09-legacy-ui-stderr.log) | Four legacy objects initialized but unattached to render tree. Probe exit 0. |
| E10 | [Source map](E10-source-map.txt) | Source locations for quota/GPT, refresh, monogram, scroll and legacy UI owners. |

## Reproduction

From repository root:

```powershell
go run reports/supervisor/evidence/2026-10-18-ui-redesign-r2/runtime_probe.go
& reports/supervisor/evidence/2026-10-18-ui-redesign-r2/run_rpc_probe.ps1
go run reports/supervisor/evidence/2026-10-18-ui-redesign-r2/legacy_ui_probe.go
```

`runtime_probe.go` and `legacy_ui_probe.go` have `//go:build ignore`; they are evidence programs and do not join product builds or the ten-package suite. [RPC test source](rpc_pipeline_probe_test.go.txt) is loaded as a package-local Go `_test.go` only through the [overlay runner](run_rpc_probe.ps1). No production file or permanent test under `internal/` is modified. Temporary accounts, preferences, cache and overlay state stay under repository `.tmp`; cleanup checks its absolute parent before recursive removal. All new synthetic HTTP fixtures use loopback and no provider credentials.

The full suite is the repository's required command and includes its pre-existing live discovery/read tests. New probe fixtures do not call provider services, real Chrome profiles or credential writers. Probe exit 0 means observation completed; inspect expected/actual and the report for acceptance. No repeated post-completion validation or commit/hash self-audit was performed.
