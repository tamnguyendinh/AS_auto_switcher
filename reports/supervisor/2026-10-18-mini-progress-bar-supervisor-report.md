# Supervisor Report: Mini Graphical Progress Bar

**Verdict: REJECT**

## Metadata

- Review date: 2026-10-18, Asia/Bangkok. Fresh command timestamps are retained in the evidence JSON.
- Role: Independent Supervisor, using the repository `supervisor` skill and the mandatory `debugging` workflow for reproduction and root-cause tracing.
- Repository: `G:\antigravity_auto_switch`.
- Reviewed implementation: `4da49ab2073ee1920d3dce47085342ccc7901acf`.
- Authority: CEO's delegated acceptance contract and [architect specification](../architect/2026-10-18-mini-graphical-progress-bar-spec.md), commit `e551fa7`. The specification file has no differences between `e551fa7` and the reviewed implementation.
- Claim: mini quota bars replace status dots, use proportional 6px rounded fills and semantic colors, preserve the 70% window dimensions and previous invariants, and pass all system gates.
- Entry point: [Coder handoff](../coder/2026-10-18-mini-progress-bar-report.md). Its claims were independently checked against source and fresh mounted runtime.
- Rules: no `AGENTS.md` exists at the repository root, anywhere in the repository, or the `G:\` parent. The explicit CEO contract governs this review. `working-rules`, the repository supervisor skill and its four references, and `debugging` were read. Anvien was not used, as expressly prohibited by the contract. No secondary agents were opened.
- Changes owned by this lane: this report and reproducible evidence under `reports/supervisor/evidence/2026-10-18-mini-progress-bar/`. No production source or existing test was edited.

## Executive Summary

The normal-data widget works at **1008×630**: track **50×6px**, radius **3px**, fill lengths **50 / 45.8 / 23 / 6px** for **100 / 91.6 / 46 / 12%**, correct color thresholds, and account cards **328×81.96875px**. Quota rows fit within the **320px** inner card width. Refreshing the first card changes its fills to **23px / 6px** and preserves all **5/5 sibling objects**.

Acceptance is blocked by two reproduced product defects:

1. Fixed-width mini bars increase the populated content minimum width to **903.2031px**, breaking the required **840×525** minimum. A forced **840×525** software canvas clips the third column; the grid remains **895.2031px** wide instead of **824px**.
2. Connection-error rows still render the old **7×7 status dot** and contain **zero mini bars**, contrary to the specification's muted-gray bar requirement for errors.

All required system commands pass: **10/10 packages**, **173 PASS records** including **147 top-level tests**, **0 FAIL**, **0 SKIP**; native GUI build exit **0**; staticcheck exit **0**, **0 diagnostics**. These gates do not cover the two failing runtime invariants. Required next step: Coder repairs only these two invariants, adds suitable regression assertions, and returns for independent acceptance.

## Blocking Findings

### [P2] Fixed bar widths raise the populated window minimum above 840px

**Source:** `internal/ui/quota_badge.go:55`, `:227`, `:241`, `:251`; propagation through `internal/ui/widgets.go:149`, `internal/ui/window.go:171`, `:172`, `:55`.

Both normal and missing-data model items always request a **50px** bar. `miniProgressBarLayout.MinSize` advertises that fixed width. Each model item and cycle row uses a horizontal box; there is no compact-width adaptation. The longest mounted cycle row measured **287.73438px**, producing a card minimum width of **295.73438px**. Fyne's three-column grid uses the largest child minimum:

```text
card minimum       = 295.73438px
grid minimum       = 3 × 295.73438 + 2 × 4 = 895.2031px
content minimum    = 895.2031 + 8 = 903.2031px
required canvas    = 840px
expected grid      = 840 - 16 = 824px
```

Fyne v2.8.2 `layout/gridlayout.go:132` takes the maximum child minimum across grid cells; `internal/widget/scroller.go:560` propagates the content minimum width for a vertical-only scroller. Production `minSizeLayout.MinSize` correctly takes the maximum of its requested minimum and its child minimum, so the hard-coded `840` lower bound does not prevent this regression.

**Fresh evidence:** E06 JSON `default_geometry.content_min.Width = 903.2031`, `minimum_geometry.grid.Width = 895.2031`; failed checks `window/min-exact` and `layout/840x525/grid-width`. E07 visibly shows clipped rightmost percentages/actions at **840×525**. Default E06 has no clipping and is not a counterexample to the minimum-size failure.

**Why it blocks:** the CEO contract requires preservation of the **840×525** minimum, and architect §3.2 calls for compact behavior at that size. Merely changing the configured root constant cannot close this invariant.

**Fix direction:** adapt the quota row/bar width to the actual card width, retaining **50px at 1008×630**, the three-column grid, proportional fills and card height. Ensure the mounted content can actually fit **840×525**, including the longest normal percentage strings and missing-data labels. No unrelated layout or backend change is needed.

**Required re-review evidence:** populated root minimum **840×525**, natural grid **824px**, cards **272px** wide, complete visible third column and non-overlapping quota rows at minimum; repeat default rendering and refresh preservation checks.

### [P2] Error rows bypass mini bars and preserve the old status dot

**Source:** `internal/ui/quota_badge.go:255`–`:272`.

`CreateQuotaCycleRow` immediately returns `container.NewHBox(lbl, createErrorBadge())` when `isError` is true. `createErrorBadge` constructs a `canvas.Circle` inside a **7×7** wrapper. Neither model item nor `createMiniProgressBar` runs on that branch.

**Fresh evidence:** account06 is populated with `status: "Error"` and a synthetic error message. E06 records `error_card_bar_count = 0`; texts are `5h:`, `[!] Lỗi kết nối`, `week:`, `[!] Lỗi kết nối`. The check `error/four-muted-empty-mini-bars` fails. E06/E09 visually show dots and no gray mini tracks on the error card; the adjacent missing-data card does show gray mini tracks.

**Why it blocks:** architect §5.2 explicitly requires muted-gray bars when data is absent **or an error occurs**. The contract also asks to verify replacement of the old quota status-dot representation.

**Fix direction:** render muted mini tracks for the error state while retaining a truthful connection-error indication and never presenting a red `0%` as real quota. Close both cycle rows, not only the normal-data or missing-data paths.

**Required re-review evidence:** a mounted error card with gray mini bars for both cycles, no residual quota status dots, no fabricated percentage, and preserved compact geometry.

## Source-Level Clearance Notes

| Production surface | Independent result |
|---|---|
| `internal/ui/quota_badge.go:60`–`:107` | **Clear for geometry and tiers.** Track and fill are canvas rectangles, height 6 and radius 3. Width is `barWidth * float32(clamped)`, negative fractions become zero, values above 1 clamp to full width. Independent mounted matrix verifies 11 values, including 15% and 50% boundaries. Track `#44445566` is translucent muted gray; fill tiers are exact `#22c55e`, `#f59e0b`, `#ef4444`. |
| `internal/ui/quota_badge.go:225`–`:251` | **Clear at default size, blocked at minimum.** Both normal and missing-data items have `[model][mini bar][percentage]` order. Missing fractions render `--%` and an empty muted track. Fixed 50px minimum is the responsive defect above. |
| `internal/ui/quota_badge.go:255`–`:278` | **Blocked on error branch.** Normal cycle rows create both models. Error rows return the pre-existing dot/error badge and skip all bars. |
| `internal/ui/widgets.go:136`–`:177` | **Preserved.** Uses production extraction for independent 5h/week rows in a three-row card. Mounted default cards measure 81.96875px high with no overlap. No production edits in this file. |
| `internal/ui/app.go:201`, `:676`; `internal/ui/window.go:186` | Configured constants remain **1008×630**, **992** fallback and **840×525**. Mounted default and 992px grid pass; populated minimum grows through the fixed-width quota content. |
| `internal/ui/app.go:346`–`:407` | **Preserved.** Single-account refresh replaces only its target object. Fresh control activation changes the target bar widths and keeps five sibling pointers identical. |
| `internal/telemetry/client.go:238`, `:281`, `:313` | **Preserved F1/F2.** Explicit cycle fractions remain independent; missing families/cycles use unknown values. No GPT field in `AccountQuotaData`; legacy GPT input disappears from marshaled JSON. Fresh runtime reflection/serialization and existing telemetry regressions pass. |
| `internal/ui/app.go:164`–`:180`; `internal/ui/themes.go:75` | **Preserved F6.** Light theme updates the mounted AS background to **#2563eb**. Dark/light screenshots are current. |

`internal/ui/ui_test.go` was reviewed as part of the implementation diff. The new geometry test checks objects immediately after creation; it does not measure a populated minimum window. `TestOwnerRedesign_WindowSizeAndLayout` checks only that minimum size is **at least** 840×525, so it accepts an oversized minimum. `TestOwnerRedesign_F5_CompactCardHeight` checks height only. `TestUIRendering_QuotaErrorBadgesTruthful` checks error text/muted colors and absence of fake `0%`, but does not require bars. These coverage gaps explain the green suite alongside the two product failures.

F4 preservation: production searches find none of `buildActiveQuotaSummary`, `BuildActiveQuotaSummary`, `NewQuotaBadgeGroup`, `NewDualCycleQuotaBadge`, the retired `NewAccountCard` constructor, or `RefreshSingleQuota`. Searches for production `.GPT` access and a `json:"gpt` tag return no matches. New mini-bar helpers are called by the quota-row path/layout; staticcheck reports no unused-code diagnostics. No claim is made to a new repository-wide dead-code audit beyond this invariant family.

## Fresh Command Evidence

| ID | Command | Result |
|---|---|---|
| E01 | `git show 4da49ab2073ee1920d3dce47085342ccc7901acf -- internal/ui/quota_badge.go internal/ui/ui_test.go` | Full reviewed diff retained in `E01-reviewed-diff.patch`. Production change is limited to quota_badge.go; ui_test.go contains its test updates. |
| E02 | `go build ./...` | Exit **0**, **5623ms**. |
| E03 | `go build -tags gui ./cmd/switcher` | Exit **0**, **12107ms**, native binary **53,520,282 bytes**. |
| E04 | `go test -v -count=1 ./...` | Exit **0**, **34333ms**, **10/10 packages**. **173 PASS records = 147 top-level + 26 subtests; 0 FAIL; 0 SKIP.** |
| E05 | `staticcheck -tags gui ./...` | Exit **0**, **1892ms**, **0 diagnostics**. |
| E06 | `go run reports/supervisor/evidence/2026-10-18-mini-progress-bar/runtime_probe.go` | Exit **1** from acceptance assertions: **160 checks**, **157 pass**, **3 fail**, grouped into the two defects above. Probe completed its screenshots, refresh and theme checks; stderr contains only `exit status 1`. |

The E04 package list is complete: `cmd/switcher`, `internal/connector`, `internal/manager`, `internal/model`, `internal/reloader`, `internal/scanner`, `internal/store`, `internal/telemetry`, `internal/ui`, `internal/wincred`. Full verbose output is retained; no test was skipped or selectively excluded.

## Visual and Runtime Evidence

Evidence root: [evidence/2026-10-18-mini-progress-bar](evidence/2026-10-18-mini-progress-bar/).

- [E00 review context](evidence/2026-10-18-mini-progress-bar/E00-review-context.json): reviewed state, specification comparison, tool versions and rule context.
- [E01 exact reviewed diff](evidence/2026-10-18-mini-progress-bar/E01-reviewed-diff.patch).
- [E02–E05 command records](evidence/2026-10-18-mini-progress-bar/command-results.json), [full test output](evidence/2026-10-18-mini-progress-bar/E04-go-test-verbose.log), [test summary](evidence/2026-10-18-mini-progress-bar/E04-go-test-summary.json), [staticcheck output](evidence/2026-10-18-mini-progress-bar/E05-staticcheck.log).
- [E06 full runtime observations](evidence/2026-10-18-mini-progress-bar/E06-runtime-probe.json), [command result](evidence/2026-10-18-mini-progress-bar/E06-runtime-command.json), [stderr](evidence/2026-10-18-mini-progress-bar/E06-runtime-stderr.log).
- [E06 Dark, 1008×630](evidence/2026-10-18-mini-progress-bar/E06-headless-dark-1008x630.png): full/proportional/zero/missing bars and the error-card dot. All three columns fit.
- [E07 Dark, 840×525](evidence/2026-10-18-mini-progress-bar/E07-headless-dark-840x525.png): reproduced clipping of column three.
- [E08 after single refresh](evidence/2026-10-18-mini-progress-bar/E08-headless-after-single-refresh.png): first card changes from 100%/91.6% to 46%/12%; five siblings remain identical objects.
- [E09 Light, 1008×630](evidence/2026-10-18-mini-progress-bar/E09-headless-light-1008x630.png): proportional bars and AS background #2563eb.

All four screenshots were visually inspected. They are fresh **Fyne software-canvas captures of the production App**, scale 1, with six synthetic offline accounts. The probe injects temporary storage under repository `.tmp`, an offline client/source and synthetic WinCred handlers; it removes its own temporary directory on exit. It does not call real quota endpoints or write real credentials. This is the requested headless canvas evidence, not a native desktop screenshot.

Reproduction from the repository root:

```powershell
& .\reports\supervisor\evidence\2026-10-18-mini-progress-bar\run_gates.ps1
& .\reports\supervisor\evidence\2026-10-18-mini-progress-bar\run_runtime_probe.ps1
```

## Invariant Closure and Required Resubmission

| Invariant | Status |
|---|---|
| Mini bar geometry, depletion and color thresholds | Verified for normal/missing data, including clamp and boundary cases. |
| Default 1008×630, fallback 992px, compact cards | Preserved; 328×81.96875px cards, no quota overflow at default. |
| Minimum 840×525 | **Open: populated minimum is 903.2031×525; third column clips when forced smaller.** |
| Error-state muted bars and status-dot replacement | **Open: error rows contain zero bars and retain 7×7 dots.** |
| F1 independent cycles / missing / zero | Preserved by current source, mounted extraction and fresh full regression suite. |
| F2 zero GPT struct/JSON | Preserved by reflection, serialization and fresh telemetry regressions. |
| F3 in-place single-card refresh | Preserved; target fill changes; 5/5 siblings identical; one provider call. |
| F4 retired helpers / new wiring | Preserved within the reviewed helper family; staticcheck clean. |
| F6 AS theme recolor | Preserved; mounted Light background #2563eb. |
| Full tests / native GUI build / staticcheck | All required gates pass. |

Required fixes are limited to (1) fitting complete mini-bar quota rows and all three columns at the required minimum, while retaining 50px bars at default size, and (2) adding muted mini bars to the connection-error path without losing truthful error indication. Add regression checks at these real boundaries and supply fresh default/minimum/error/refresh evidence. Supervisor made no product repair. **Commit 4da49ab is not accepted until these two same-scope invariants are closed.**
