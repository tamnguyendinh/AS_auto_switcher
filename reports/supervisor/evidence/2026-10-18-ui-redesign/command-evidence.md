# Command evidence — Supervisor review of a55fd5d

Date: 2026-10-18, Asia/Bangkok (UTC+07). These are recorded results of this review, not Coder's earlier logs.

## Reviewed source state

- Target: `a55fd5da589e68039cb51ab7cc3a2cea797747dd`.
- Workspace HEAD at intake: `5b2a9c5a0f5504735af35a06d5296990238ded1a`.
- Initial worktree status: clean.
- `git diff --name-only a55fd5d HEAD` returned only `reports/supervisor/2026-10-18-purge-ui-timeout-supervisor-report.md`. Thus production code, test code, and module inputs used here match the target commit.
- Inspected `git show a55fd5d` for all five touched Go files and followed the affected telemetry pipeline in `internal/telemetry/{client,pool,cache}.go`.
- Root `AGENTS.md` and ancestor `G:/AGENTS.md` were absent; discovery found no `AGENTS.md`. The explicit lane contract and supplied source authorities govern this review.

## Build

Environment: `go version go1.26.3 windows/amd64`, `CGO_ENABLED=1`, `CC=gcc`.

```text
go build ./...
exit 0, no diagnostics

go build -tags gui -o .tmp/supervisor-ui-redesign-gui.exe ./cmd/switcher
exit 0, no diagnostics
```

No repository build script was invoked because that script rebuilds release artifacts and has broader side effects. Both native build entry paths above were built directly.

## Required full workspace test command

```text
go test -v -count=1 ./...
ok  antigravity-switch/cmd/switcher          4.030s
ok  antigravity-switch/internal/connector   3.796s
ok  antigravity-switch/internal/manager    14.491s
ok  antigravity-switch/internal/model       0.901s
ok  antigravity-switch/internal/reloader    3.816s
ok  antigravity-switch/internal/scanner     1.119s
ok  antigravity-switch/internal/store       9.162s
ok  antigravity-switch/internal/telemetry   15.960s
ok  antigravity-switch/internal/ui         11.455s
ok  antigravity-switch/internal/wincred      0.596s
GO_TEST_EXIT=0
```

Full original command output: `go-test-verbose.log`. Result: 10 packages, 127 top-level cases plus 19 subcases = 146 PASS lines; 0 FAIL, 0 SKIP. Live Language Server discovery/status and live WinCred read both executed and passed. The five `TestOwnerRedesign_*` tests passed; their assertions do not establish the complete acceptance claim.

## Staticcheck — failed gate

Resolved tool version: `honnef.co/go/tools v0.8.2`.

```text
go run honnef.co/go/tools/cmd/staticcheck@latest -tags gui ./...
cmd\switcher\app_gui.go:13:7: const hasGUI is unused (U1000)
internal\connector\api_client.go:115:3: this value of err is never used (SA4006)
internal\ui\app.go:51:2: field saveBtn is unused (U1000)
internal\ui\app.go:57:2: field headerCard is unused (U1000)
exit status 1
```

The last two diagnostics concern retired UI members in the reviewed scope. The first two are outside the UI diff and are reported as workspace gate failures, not newly introduced UI regressions. Reproduction can pin `@v0.8.2`.

## Whole-program reachability — zero dead code not established

```text
go run golang.org/x/tools/cmd/deadcode@latest -tags gui -filter antigravity-switch ./cmd/switcher
```

Resolved tool version: `golang.org/x/tools v0.52.0`. Exit 0 means the analyzer completed; it does not mean no unreachable functions. The run emitted 60 unreachable-function candidates across the module. This is the UI subset of the actual output:

```text
internal\ui\app.go:120:15: unreachable func: App.SetPreferencesPath
internal\ui\app.go:127:15: unreachable func: App.SetTelemetryPool
internal\ui\app.go:141:15: unreachable func: App.SetQuotaCache
internal\ui\app.go:148:15: unreachable func: App.CurrentTheme
internal\ui\app.go:155:15: unreachable func: App.IsQuotaRefreshing
internal\ui\app.go:272:15: unreachable func: App.Window
internal\ui\app.go:279:15: unreachable func: App.Manager
internal\ui\app.go:419:15: unreachable func: App.RefreshSingleQuota
internal\ui\app.go:640:15: unreachable func: App.SetAlwaysOnTop
internal\ui\app.go:757:15: unreachable func: App.Logo
internal\ui\quota_badge.go:36:6: unreachable func: FormatRemainingCountdown
internal\ui\quota_badge.go:174:6: unreachable func: NewDualCycleQuotaBadge
internal\ui\quota_badge.go:182:6: unreachable func: NewQuotaBadgeGroup
internal\ui\quota_badge.go:187:6: unreachable func: NewQuotaCountdownLabel
internal\ui\widgets.go:97:6: unreachable func: NewAccountCard
```

Exported injection seams used by tests are not automatically defects. However, the retired countdown implementation/stub and compatibility badge/refresh wrappers have no production callers. Source inspection also confirmed invisible legacy labels/background rectangles still receive writes. A follow-up `deadcode -test -tags gui ... ./cmd/switcher` also emitted findings; testing only the command package does not add all dependency-package tests as roots. No zero-result claim is made for either run.

## Reproduction and rendering

```text
go run reports/supervisor/evidence/2026-10-18-ui-redesign/runtime_probe.go
exit 0
```

Final durable results: `runtime-probe.json`, `headless-dark.png`, `headless-light.png`. Probe uses unchanged production functions, synthetic accounts, loopback HTTP, isolated `.tmp` state, and Fyne's software test driver. The source is build-excluded so it does not add an eleventh package to normal builds/tests. Both final PNGs were visually inspected. Debug attempts were superseded by this successful final run; debug stderr contained no diagnostics on that run.

Observed results:

- Connect-RPC input: Claude 5h/week 46%/12%, Gemini 88%/90%. Actual extracted UI values: Claude 12%/12%, Gemini 88%/88%.
- The specification's REST summary payload sent by a loopback fixture to current `QuotaClient` produced 100%/100% for both models. This demonstrates parser incompatibility with the required payload, not an assertion that the live legacy endpoint returned that fixture.
- Serialized `AccountQuotaData` still contains `gpt`.
- Card refresh fetched only `account1@example.com`, but all five sibling card objects were replaced (expected five preserved, actual zero preserved).
- Refresh callback returned in under the millisecond resolution recorded; total synthetic operation took 98.57 ms with a fake 50 ms fetch. This is not a measurement or guarantee of live provider latency.
- Canvas 1440x900; content minimum 1200x750; fixed size false.
- Six cards in three columns; grid 1424x760.84375 against minimum 620.9375x167.9375. First card height 378.42188 against natural minimum 81.96875.
- Selecting Light through the actual theme select widget left the AS monogram `#89b4fa`; expected current Light primary is `#2563eb`.

## Native desktop evidence limit

The freshly built GUI was launched with the Windows Computer Use skill. `sky.list_windows()` returned exactly one matching target window, titled `⚡ Antigravity Multi-Account Switcher`. State capture failed first with `FrameArrived timed out: timed out waiting on channel`, then failed on one prescribed recovery attempt with `window capture timed out: timed out waiting on channel`. Further native UI input was stopped. The review's own GUI process was terminated afterward.

No native screenshot, OS resize/minimum-size interaction, hover demonstration, or live provider ~50 ms refresh was established. Headless PNGs are explicitly not native screenshots. Live account switching was not exercised against Owner credentials; existing isolated switch tests passed in the required suite.
