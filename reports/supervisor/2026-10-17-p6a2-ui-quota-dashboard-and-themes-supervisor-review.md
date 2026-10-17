# Supervisor Report: Zero-Trust Acceptance Review for Slice P6A-2 (Fyne UI Quota Dashboard, Refresh Actions & Multi-Theme System)

Verdict: REJECT

## Metadata
- **Report file:** `reports/supervisor/2026-10-17-p6a2-ui-quota-dashboard-and-themes-supervisor-review.md`
- **Review time:** 2026-10-17 16:00:00 ICT (+07:00)
- **Reviewer:** Supervisor Lane
- **Repo/project:** `antigravity_auto_switch`
- **Scope reviewed:** Commit `7bbd9ac06f4017ae38df003987a9b5cee4c1edd2`, package `internal/ui/` (`themes.go`, `quota_badge.go`, `widgets.go`, `window.go`, `app.go`, `ui_test.go`, `themes_test.go`), and Coder Report `reports/coder/2026-10-17-p6a2-ui-quota-dashboard-and-themes-report.md`
- **Claim reviewed:** Slice P6A-2 is 100% complete with 21/21 passing tests, zero regressions across the workspace, and full verification against UI Traceability Matrix UI-TR-01..08.
- **Authority used:** Direct Human Owner Mandate (4 Zero-Trust Pillars), AGENTS.md, Phase 6A Architecture Spec (`reports/architect/2026-10-17-phase-6a-quota-telemetry-and-multi-theme-dashboard-architecture.md`), Phase 6A Plan (`docs/plans/2026-10-17-phase-6a-quota-telemetry-dashboard/2026-10-17-phase-6a-quota-telemetry-dashboard-plan.md`), Coder Report (`reports/coder/2026-10-17-p6a2-ui-quota-dashboard-and-themes-report.md`).
- **Related artifacts:** Commit `7bbd9ac06f4017ae38df003987a9b5cee4c1edd2`.

---

## Executive Summary
- **Problem:** The Coder claimed completion of Slice P6A-2 with 21 passing test cases and zero regression across the repository. A Zero-Trust audit was mandated by the Human Owner to verify test authenticity, UI traceability, absence of dead code, and runtime invariants.
- **Decision:** **REJECT**. While the production UI architecture in `internal/ui/` is largely well-designed (clean theme palette, responsive layouts, non-blocking asynchronous dispatches), independent verification revealed:
  1. An active race condition causing `go test ./...` and `go test -count=10 ./internal/ui` to intermittently **FAIL** at `TestGlobalQuotaRefreshAndIndicator` (`app.quotaRefreshAllBtn.Text` stuck at `⏳ Đang quét...`, timestamp still `--:--:--`, status bar race).
  2. A "pass-by-default" fake test in `TestAccountCardWithQuotaAndRefresh`, where the test created a detached closure `refreshFn` and called it directly instead of testing the actual `[🔄]` button on the card.
  3. Superficial canvas assertions in `TestQuotaBadgeGroupAndColors` and `TestThemeInterfaceImplementations` where badges and colors are merely checked for `!= nil` without validating canvas elements, colors, or percentage text.
  4. Concurrent disk cache pollution during tests: `NewAppWithFyne` defaults to relative path `accounts_store/.quota_cache.json`, resulting in orphaned temp files (`.quota_cache_tmp_*.json`) inside `internal/ui/accounts_store` on Windows.
- **Required outcome:** Coder must fix the asynchronous synchronization in `waitForIdle` / `RefreshAllQuota`, make tests interact with real Canvas widgets, eliminate pass-by-default dummy closures, isolate test cache paths into `t.TempDir()`, and re-submit for review.

---

## Blocking Findings

### [HIGH] FINDING-01: Asynchronous Timing Race Condition Causing Flaky Test & Workspace Test Failure
- **File:** `internal/ui/ui_test.go:727-740`, `internal/ui/app.go:263-326, 332-377`
- **Issue:** `go test ./...` and `go test -run TestGlobalQuotaRefreshAndIndicator -count=10 ./internal/ui` fail intermittently.
  ```
  --- FAIL: TestGlobalQuotaRefreshAndIndicator (0.70s)
      ui_test.go:727: expected button restored to standard label, got: ⏳ Đang quét...
      ui_test.go:732: expected timestamp to be updated, got: Tự động: 60s | Cập nhật: --:--:--
      ui_test.go:740: expected status bar text to reflect single refresh, got: Đang quét hạn ngạch toàn bộ tài khoản...
  FAIL
  FAIL    antigravity-switch/internal/ui  6.522s
  FAIL
  ```
- **Evidence:**
  In `internal/ui/ui_test.go`, `waitForIdle(app, 3*time.Second)` only checks `app.isBusy`:
  ```go
  func waitForIdle(app *App, timeout time.Duration) {
      deadline := time.Now().Add(timeout)
      for time.Now().Before(deadline) {
          app.mu.RLock()
          busy := app.isBusy
          app.mu.RUnlock()
          if !busy {
              time.Sleep(30 * time.Millisecond)
              return
          }
          time.Sleep(20 * time.Millisecond)
      }
  }
  ```
  However, `RefreshAllQuota()` and `RefreshSingleQuota()` do NOT set `app.isBusy`—they use `a.isQuotaRefreshing.CompareAndSwap(false, true)`. Consequently, `waitForIdle` sees `app.isBusy == false` immediately, sleeps only 30ms, and returns before the background goroutine running `telemetryPool.PollAll(ctx)` and `fyne.Do()` has completed. When CPU load increases (e.g. during `go test ./...`), 30ms is insufficient, and the assertions evaluate while `a.quotaRefreshAllBtn` is still `"⏳ Đang quét..."` and `a.autoRefreshLabel` is still `"--:--:--"`.
  Additionally, `app.RefreshSingleQuota("acc1@gmail.com")` immediately attempts to run, but `a.isQuotaRefreshing` is still locked by the prior refresh, causing single refresh rejection and status bar mismatch.
- **Why this blocks acceptance:** A flaky test suite directly breaks CI/workspace validation (`go test ./...` exits with code 1), violating Zero-Trust Pillar 1 and Pillar 4.
- **Fix direction:**
  1. Update `waitForIdle` (or add `waitForQuotaIdle`) to poll both `app.isBusy` AND `app.isQuotaRefreshing.Load()`.
  2. Ensure `isQuotaRefreshing` is reset only AFTER `fyne.Do()` completes UI updates on the canvas.
- **Re-review evidence required:** 10 consecutive passes of `go test -count=10 -run TestGlobalQuotaRefreshAndIndicator ./internal/ui` and a clean `go test ./...` across the entire workspace.

---

### [MEDIUM] FINDING-02: Pass-By-Default Test in `TestAccountCardWithQuotaAndRefresh`
- **File:** `internal/ui/ui_test.go:677-681`
- **Issue:** The test asserts callback execution via a disconnected local variable instead of interacting with the card widget:
  ```go
  card := NewAccountCardWithQuota(
      item,
      quota,
      DarkPalette,
      func(email string) {},
      func() {},
      func(email string) {
          singleRefreshed = email
      },
  )

  if card == nil {
      t.Fatalf("expected non-nil card with quota")
  }

  refreshFn := func(email string) { singleRefreshed = email }
  refreshFn(item.Email)
  if singleRefreshed != "test@example.com" {
      t.Errorf("expected singleRefreshed to be test@example.com, got %s", singleRefreshed)
  }
  ```
- **Evidence:** `refreshFn` is defined locally and invoked directly: `refreshFn(item.Email)`. The `[🔄]` button inside `card` (`refreshBtn` at `widgets.go:88`) is never tapped, tested, or inspected. If the button was omitted from `card` entirely, this test would still pass!
- **Why this blocks acceptance:** Directly violates Human Owner Pillar 1 ("TUYỆT ĐỐI KHÔNG TIN TEST XANH MÙ QUÁNG... xác minh kiểm thử nút bấm và render badge có thực sự kiểm tra thuộc tính đồ họa trên Canvas hay không").
- **Fix direction:** Extract or traverse `card`'s canvas hierarchy, locate the `refreshBtn` (`*widget.Button`) inside the card container, and trigger `refreshBtn.OnTapped()` (or verify its properties directly).
- **Re-review evidence required:** Direct invocation of `refreshBtn.OnTapped()` from the card widget causing `singleRefreshed` to be populated with `test@example.com`.

---

### [MEDIUM] FINDING-03: Missing Canvas Graphic Attribute Assertions for Quota Badges & Themes
- **File:** `internal/ui/ui_test.go:535-548`, `internal/ui/themes_test.go:60-95`
- **Issue:** Tests assert only non-nil status rather than checking rendered visual properties:
  1. `TestQuotaBadgeGroupAndColors`: Calls `badges := NewQuotaBadgeGroup(quota)` and checks only `if badges == nil { t.Fatalf(...) }`. It never verifies whether the container has 3 columns, whether circles have colors `#22c55e` (Claude), `#f59e0b` (Gemini), `#ef4444` (GPT), or whether percentage texts ("100%", "23.5%", "10%") are present.
  2. `TestThemeInterfaceImplementations`: Calls `th.Color(theme.ColorNameBackground, ...)` and checks only `if bg == nil`. It does not compare against `DarkPalette.Background`, `LightPalette.Background`, or `WarmPalette.Background`.
  3. `TestAppSetThemeRuntime`: Only asserts `app.CurrentTheme() != ThemeLight` (string struct field); it does not verify whether `app.headerBg.FillColor`, `app.statusBg.FillColor`, or `app.legendBg.FillColor` updated to the new theme's card color.
- **Why this blocks acceptance:** Fails Zero-Trust Pillar 1 verification of graphical properties on Canvas.
- **Fix direction:**
  - Inspect `badges` container children to verify dot circle colors and text labels.
  - Assert that `th.Color(theme.ColorNameBackground, ...)` matches the palette's expected RGBA values for each theme.
  - Assert that `headerBg.FillColor` updates upon `app.SetTheme(...)`.
- **Re-review evidence required:** Concrete assertions on Canvas objects in `ui_test.go` and `themes_test.go`.

---

### [LOW] FINDING-04: Test Disk Pollution and Windows File Lock Artifacts in `internal/ui/accounts_store`
- **File:** `internal/ui/app.go:94-96`, `internal/ui/ui_test.go:370-500`
- **Issue:** When tests run inside `internal/ui`, `NewAppWithFyne` initializes `telemetry.NewQuotaCache(telemetry.DefaultQuotaCachePath)`. Because `DefaultQuotaCachePath` is relative (`accounts_store/.quota_cache.json`), tests create an un-isolated `internal/ui/accounts_store` directory containing 17+ orphaned temporary files (`.quota_cache_tmp_*.json`).
- **Evidence:**
  Directory `G:\antigravity_auto_switch\internal\ui\accounts_store` contains:
  - 17 temporary files such as `.quota_cache_tmp_1001753418.json`
  - `.quota_cache.json`
  - `.ui_preferences.json`
  Concurrent tests writing to the same relative file on Windows trigger atomic rename collisions, leaving orphaned temp files.
- **Why this blocks acceptance:** Pollutes the repository with untracked/ignored disk clutter and risks file contention during test execution.
- **Fix direction:** Ensure test cases inject an isolated `QuotaCache` backed by `t.TempDir()`, and remove the accumulated test debris from `internal/ui/accounts_store`.
- **Re-review evidence required:** Running tests does not leave orphaned files in `internal/ui/accounts_store`.

---

## Non-Blocking Notes & Dead Code Audit

### Note 01: Unused `generateTrayIconBytes()` in `internal/ui/tray.go`
- **File:** `internal/ui/tray.go:15-39`
- `generateTrayIconBytes()` is defined but never called anywhere in the codebase. Line 57 uses `AppLogoResource` directly:
  ```go
  if AppLogoResource != nil && len(AppLogoResource.Content()) > 0 {
      desk.SetSystemTrayIcon(AppLogoResource)
  }
  ```
- **Recommendation:** Remove `generateTrayIconBytes()` during the P6A-2 remediation pass to keep the codebase lean.

---

## UI Traceability Matrix Audit (UI-TR-01 through UI-TR-08)

| UI-TR ID | Requirement | Code Implementation | Verification Finding | Status |
|---|---|---|---|---|
| **UI-TR-01** | Global Quota Refresh Button labeled `[📊 Làm mới Quota Tất cả]` in Toolbar with async spinning indicator | `window.go:106`, `app.go:263` | Button exists with correct label and disables with `⏳ Đang quét...`. However, asynchronous timing race causes flakiness in unit testing. | **DEFECT (RACE)** |
| **UI-TR-02** | Row-Level Single Refresh Button `[🔄]` adjacent to account email | `widgets.go:88`, `app.go:332` | Button created with `theme.ViewRefreshIcon()` and wired to `onRefreshSingle`. However, test `TestAccountCardWithQuotaAndRefresh` used a fake closure instead of testing the button. | **DEFECT (FAKE TEST)** |
| **UI-TR-03** | Auto-Refresh 60s Indicator `Tự động: 60s | Cập nhật: HH:mm:ss` | `window.go:121`, `app.go:211` | Label properly initialized and updated in background loop. | **VERIFIED** |
| **UI-TR-04** | Mini Quota Progress Badges for Claude, Gemini, GPT with 3-tier colors (>50% Green, 15-50% Amber, <15% Red) | `quota_badge.go:19-29, 64-106` | Threshold colors `#22c55e`, `#f59e0b`, `#ef4444` correctly mapped. Unit test needs deep canvas assertions. | **VERIFIED (WEAK TEST)** |
| **UI-TR-05** | Quota Recovery Countdown Text `Google sẽ hồi phục lúc: HH:mm (còn Xh Yp)` for quota < 15% | `quota_badge.go:109-158` | Correctly triggers only when model quota < 15% with computed UTC countdown string. | **VERIFIED** |
| **UI-TR-06** | Multi-Theme Switcher Control (`🌙 Tối`, `☀️ Sáng`, `☕ Ấm`) with instant canvas refresh | `window.go:61`, `themes.go:183`, `app.go:127` | Implemented via `widget.NewSelect`. Switches themes instantly without app crash and persists to disk. | **VERIFIED** |
| **UI-TR-07** | Warm Mode Color Palette eliminating harsh blue light with warm sepia (#241e17) and amber (#d97706) | `themes.go:73-87` | All RGB color tokens verified in `TestThemeColorTokens`. | **VERIFIED** |
| **UI-TR-08** | Non-Blocking UI Invariant maintaining responsive 60 FPS event loop with zero GUI freezes | `app.go:263, 332` | Network calls run in goroutines with timeouts; UI mutations use `fyne.Do()`. However, test synchronization needs alignment with `isQuotaRefreshing`. | **VERIFIED (RUNTIME OK / TEST RACE)** |

---

## Detailed Audit of the 21 Test Cases

| # | Test Function | Target | Zero-Trust Finding |
|---|---|---|---|
| 1 | `TestThemeColorTokens` | `themes.go` | **PASS**: Strictly asserts RGB bytes for Dark, Light, Warm palettes. |
| 2 | `TestThemeInterfaceImplementations` | `themes.go` | **WEAK**: Only tests `color != nil`, not validating token fidelity. |
| 3 | `TestNewThemeInstance` | `themes.go` | **PASS**: Verifies concrete type conversion for all 3 themes. |
| 4 | `TestUIPreferencesPersistence` | `themes.go` | **PASS**: Tests save, load, fallback on missing/corrupt JSON. |
| 5 | `TestAppSetThemeRuntime` | `app.go`, `themes.go` | **INCOMPLETE**: Only tests `app.CurrentTheme()` string, not Canvas background updates. |
| 6 | `TestNewAppAndInit` | `app.go`, `window.go` | **PASS**: Checks window initialization and basic layout. |
| 7 | `TestToggleAlwaysOnTop` | `app.go`, `window.go` | **PASS**: Checks toggle state, button styling, user32 integration. |
| 8 | `TestUpdateUI_WithAccounts` | `app.go`, `widgets.go` | **PASS**: Checks rendering of account rows in container. |
| 9 | `TestUpdateUI_Empty` | `app.go` | **PASS**: Checks empty state message on zero accounts. |
| 10 | `TestNewAccountCard_Callbacks` | `widgets.go` | **PASS**: Verifies switch callback invocation. |
| 11 | `TestStatusDot` | `widgets.go` | **PASS**: Checks circle dimensions and color. |
| 12 | `TestAppLogoResourceAndWindowIcon` | `assets.go` | **PASS**: Checks logo byte resource integrity. |
| 13 | `TestElevationTokensAndBorders` | `themes.go` | **PASS**: Checks card background rectangle and borders. |
| 14 | `TestSwitchAccount_NoModalPopup` | `app.go` | **PASS**: Verifies non-blocking hot-swap execution without modal. |
| 15 | `TestSaveSession_NoModalPopup` | `app.go` | **PASS**: Verifies session capture without modal. |
| 16 | `TestQuotaBadgeGroupAndColors` | `quota_badge.go` | **WEAK**: Validates tier colors mathematically, but badge container only checked for `!= nil`. |
| 17 | `TestQuotaCountdownCalculationAndLabel` | `quota_badge.go` | **PASS**: Verifies countdown string format and `<15%` threshold trigger. |
| 18 | `TestAccountCardWithQuotaAndRefresh` | `widgets.go` | **FAIL (FAKE TEST)**: Tests a local closure instead of the card's `[🔄]` button. |
| 19 | `TestGlobalQuotaRefreshAndIndicator` | `app.go`, `window.go` | **FAIL (RACE)**: Premature exit in `waitForIdle` causes intermittent test failure. |
| 20 | `TestActiveQuotaHeaderSummary` | `quota_badge.go` | **PASS**: Validates nil and populated summary strings. |
| 21 | `TestUITraceabilityChecklist` | Multiple | **SURFACE**: Exercises methods without deep canvas assertions. |

---

## Source-Level Clearance Notes
- `internal/ui/themes.go`: **CLEAR WITH REFINEMENT** - Palette and theme structures are clean, robust, and correctly persist preferences.
- `internal/ui/quota_badge.go`: **CLEAR** - Math, formatting, and countdown calculations strictly meet requirements.
- `internal/ui/widgets.go`: **CLEAR IN CODE, DEFECT IN TEST** - `NewAccountCardWithQuota` correctly builds UI-TR-02 and UI-TR-04 components, but test coverage must be made genuine.
- `internal/ui/window.go`: **CLEAR** - Header Toolbar, Multi-Theme selector, and Status Bar layouts are well-organized and comply with 440x560 sizing.
- `internal/ui/app.go`: **BLOCKED (RACE CONDITION)** - Needs clean synchronization between `isQuotaRefreshing` and `waitForIdle`, plus `QuotaCache` isolation for tests.
- `internal/ui/ui_test.go`: **BLOCKED** - Contains race condition in `TestGlobalQuotaRefreshAndIndicator` and fake test in `TestAccountCardWithQuotaAndRefresh`.
- `internal/ui/themes_test.go`: **CLEAR WITH REFINEMENT** - Needs exact RGBA checks rather than `!= nil`.

---

## Evidence Checked
### Passed:
- `$env:CGO_ENABLED="1"; go test -v ./internal/ui/...`: Passed on isolated runs, but flaky under system load.
- `go build -tags gui -o bin/antigravity-switch.exe ./cmd/switcher`: PASS (exit code 0).
- `go build -o bin/antigravity-switch-cli.exe ./cmd/switcher`: PASS (exit code 0).
- `git status --porcelain`: 0 (worktree clean).
- Window size 440x560 preserved in `internal/ui/app.go:203`.

### Failed:
- `go test ./...`: **FAIL** with exit code 1 at `TestGlobalQuotaRefreshAndIndicator`.
- `go test -run TestGlobalQuotaRefreshAndIndicator -count=10 ./internal/ui`: **FAIL** (10-run iteration failure reproduced).
- Audit of `ui_test.go:677-681`: Fake closure in place of widget button testing.

---

## Invariant Closure
- **Affected invariant:** Zero-Trust Test Authenticity, CI Test Stability (`go test ./...`), and Canvas Graphics Verification.
- **Sibling surfaces checked:** `internal/ui/` entire package, workspace test suite `go test ./...`.
- **Residual unverified same-invariant surfaces:** Asynchronous UI synchronization in test harness and genuine card button tap testing.

---

## Required Fix List For Resubmission

1. **Fix Asynchronous Synchronization in `internal/ui` Tests (FINDING-01):**
   - In `internal/ui/app.go` or `ui_test.go`, ensure `waitForIdle` checks both `a.isBusy` AND `a.isQuotaRefreshing.Load()`.
   - Ensure `isQuotaRefreshing` is set to `false` only after `fyne.Do()` canvas updates finish.
   - Verify with `go test -count=10 -run TestGlobalQuotaRefreshAndIndicator ./internal/ui` and `go test ./...`.
2. **Make `TestAccountCardWithQuotaAndRefresh` Genuine (FINDING-02):**
   - Traverse the `card` container hierarchy to locate `refreshBtn` (`*widget.Button`) and call `refreshBtn.OnTapped()`.
   - Delete the disconnected `refreshFn := ...` local dummy closure.
3. **Enhance Canvas Property Assertions (FINDING-03):**
   - In `TestQuotaBadgeGroupAndColors`, inspect the 3 badge containers in `badges` to verify dot circle colors (`#22c55e`, `#f59e0b`, `#ef4444`) and text strings.
   - In `TestThemeInterfaceImplementations`, verify that `th.Color()` returns the specific palette color tokens, not just `!= nil`.
   - In `TestAppSetThemeRuntime`, assert that `app.headerBg.FillColor` changes to `LightPalette.Card`.
4. **Isolate Test QuotaCache Paths & Clean Up Temp Files (FINDING-04):**
   - In tests creating `NewAppWithFyne`, ensure `app.SetQuotaCache(...)` is initialized with a path inside `t.TempDir()`.
   - Remove accumulated `.quota_cache_tmp_*.json` debris from `internal/ui/accounts_store`.
5. **Clean Up Dead Code (Note 01):**
   - Remove unused function `generateTrayIconBytes()` from `internal/ui/tray.go`.

---

## Overall Evaluation
The production code implementation for Slice P6A-2 is structurally strong, elegant, and matches the architectural intent of Phase 6A. However, under the strict Zero-Trust Mandate of the Human Owner, test flakiness and fake/superficial assertions cannot be accepted into the codebase. Rejecting this slice now ensures that when Slice P6A-2 is merged, the test suite is rock-solid, CI is green 100% of the time, and all UI elements on Canvas are genuinely verified.
