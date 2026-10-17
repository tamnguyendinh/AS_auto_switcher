# Phase 6A: Multi-Account Quota Telemetry Engine, Live Dashboard & Multi-Theme System Evidence Ledger

## Metadata

- Date: `2026-10-17`
- Plan: `docs/plans/2026-10-17-phase-6a-quota-telemetry-dashboard/2026-10-17-phase-6a-quota-telemetry-dashboard-plan.md`
- Rules: `docs/plans/2026-10-17-phase-6a-quota-telemetry-dashboard/2026-10-17-phase-6a-quota-telemetry-dashboard-plan.md`
- Evidence: `docs/plans/2026-10-17-phase-6a-quota-telemetry-dashboard/2026-10-17-phase-6a-quota-telemetry-dashboard-evidence.md`
- Benchmark: `docs/plans/2026-10-17-phase-6a-quota-telemetry-dashboard/2026-10-17-phase-6a-quota-telemetry-dashboard-benchmark.md`
- Actual status: `docs/plans/2026-10-17-phase-6a-quota-telemetry-dashboard/2026-10-17-phase-6a-quota-telemetry-dashboard-actual-status.md`

## Evidence Rules

The evidence file explains why the work is known to be correct.
It contains:
- Metadata and companion file links;
- Stable evidence IDs formatted as `E<phase>-<item>-<kind><n>`;
- Codebase facts, inspection commands, pass/fail results, and empirical benchmarks;
- Impact and blast-radius evidence for each implementation slice;
- Detailed proof targets for downstream Coder, QA, and Supervisor lanes.

### Evidence ID Naming Convention
- `E0-P0A-SPEC<n>`: Architectural specification and requirements evidence.
- `E0-P0A-SRC<n>`: Source code baseline inspections.
- `E0-P0A-STORE<n>`: Credential and storage inspections.
- `E0-P0A-TEST<n>`: Baseline test suite verification.
- `E1-P1A-CODE<n>`: Slice P6A-1 implementation evidence (Telemetry Engine & 60s Loop).
- `E1-P1A-TEST<n>`: Slice P6A-1 unit test and mock server results.
- `E2-P2A-CODE<n>`: Slice P6A-2 implementation evidence (UI Dashboard & Multi-Theme).
- `E2-P2A-TEST<n>`: Slice P6A-2 UI unit tests and layout verification.
- `E2-P2A-QA<n>`: Slice P6A-2 UI Traceability Matrix (UI-TR-01..08) verification.
- `E3-P3A-CODE<n>`: Slice P6A-3 CLI `--quota` implementation evidence.
- `E3-P3A-TEST<n>`: Slice P6A-3 terminal execution and dual-build verification.
- `E4-P4A-SUP<n>`: Final supervisor acceptance and git commit verification.

---

## E0 - P0 Baseline Codebase & Spec Inspection

### E0-P0A-SPEC1: Architectural Specification & Feasibility Discovery
- **Document:** `reports/architect/2026-10-17-phase-6a-quota-telemetry-and-multi-theme-dashboard-architecture.md`
- **Fact:** Discovered Google Cloud Code internal quota endpoint `https://daily-cloudcode-pa.googleapis.com/v1internal:retrieveUserQuota`.
- **Empirical Validation:** Tested live across 7 stored accounts with 100% success rate (7/7 accounts responding HTTP 200 with bucket data), completing in 1.69 seconds total parallel duration.
- **Quota Buckets:** Authoritatively identifies remaining fractions and UTC reset times for Claude 4.6 (Opus/Sonnet), Gemini 2.5 Pro/Flash, and GPT-OSS.

### E0-P0A-SRC1: Telemetry Package Baseline
- **Inspection Command:** `Get-ChildItem -Path "internal/telemetry"`
- **Result:** Directory does not exist.
- **Classification:** `missing`. Package `internal/telemetry` must be newly created in Slice P6A-1.

### E0-P0A-SRC2: OAuth Refresher Capability
- **Inspection Command:** `Get-Content -Path "internal/connector/oauth_refresher.go"`
- **Result:** `GoogleOAuthRefresher` is implemented with client ID `1071006060591-tmhssin2h21lcre235vtolojh4g403ep.apps.googleusercontent.com` and endpoint `https://oauth2.googleapis.com/token`.
- **Classification:** `correct`. Provides full capability to refresh expired tokens for quota telemetry calls.

### E0-P0A-SRC3: Current Fyne UI Baseline
- **Inspection Command:** `Get-Content -Path "internal/ui/window.go", "internal/ui/widgets.go", "internal/ui/app.go"`
- **Result:**
  - GUI currently has only a dark theme (`SwitcherTheme`). Light and Warm themes are missing.
  - Window toolbar has `🔄 Làm mới Chrome`, `💾 Lưu phiên`, and `📌 Ghim`. It lacks `[📊 Làm mới Quota Tất cả]`, auto-refresh indicator, and theme selector.
  - Account cards in `widgets.go` display email and status dot, but lack mini quota progress badges and per-row `[🔄]` refresh button.
  - `app.go` lacks background 60s ticker loop and quota data integration.
- **Classification:** `partial`. Must be expanded in Slice P6A-2.

### E0-P0A-SRC4: Current CLI Baseline
- **Inspection Command:** `Get-Content -Path "cmd/switcher/main.go"`
- **Result:** Supported flags: `-cli`, `-gui`, `-switch`, `-status`, `-list`, `-capture`. Missing `-quota` flag.
- **Classification:** `partial`. Must be added in Slice P6A-3.

### E0-P0A-SRC5: Reloader & Session Preservation
- **Inspection Command:** `Get-ChildItem -Path "internal/reloader"`
- **Result:** Complete mock-isolated test suite and session reloader implemented in Phase 6.
- **Classification:** `correct` (Preserve-only). Must remain untouched.

### E0-P0A-SRC6: WinCred Credential Format
- **Inspection Command:** `Get-Content -Path "internal/wincred/cred.go"`
- **Result:** Credential target `gemini:antigravity` read/write routines stable and isolated.
- **Classification:** `correct` (Preserve-only). Must remain untouched.

### E0-P0A-STORE1: Stored Accounts Inventory
- **Inspection Command:** `Get-ChildItem -Path "accounts_store"`
- **Result:** 7 account JSON files present:
  1. `daohien204ccyersin@gmail.com.json`
  2. `hiendao1760@gmail.com.json`
  3. `hokinhdoanh.anvien.sales@gmail.com.json`
  4. `ngau2hung@gmail.com.json`
  5. `quantritaikhoanemail@gmail.com.json`
  6. `skey.miennam@gmail.com.json`
  7. `skey.vanphongmiennam@gmail.com.json`
- **Classification:** `correct`.

### E0-P0A-TEST1: Baseline Test Suite Execution
- **Command:** `go test ./...`
- **Result:** 8/8 packages passed:
  - `antigravity-switch/internal/connector`: ok (5.258s)
  - `antigravity-switch/internal/manager`: ok (14.184s)
  - `antigravity-switch/internal/model`: ok (cached)
  - `antigravity-switch/internal/reloader`: ok (cached)
  - `antigravity-switch/internal/scanner`: ok (cached)
  - `antigravity-switch/internal/store`: ok (7.145s)
  - `antigravity-switch/internal/ui`: ok (2.090s)
  - `antigravity-switch/internal/wincred`: ok (cached)
- **Status:** PASS (100% clean baseline).

---

## E1 - P1 (Slice P6A-1) Telemetry Engine & 60s Loop Evidence Targets

- `E1-P1A-CODE1`: Implementation of `internal/telemetry/client.go` (HTTP client for Google Cloud Code `v1internal:retrieveUserQuota`, model bucket parser, UTC reset time parsing).
- `E1-P1A-CODE2`: Implementation of `internal/telemetry/cache.go` (RAM `QuotaMatrix` with `sync.RWMutex` + JSON snapshot `accounts_store/.quota_cache.json`).
- `E1-P1A-CODE3`: Implementation of `internal/telemetry/pool.go` (concurrent worker pool with `context.WithTimeout`, latency < 2.0s for $N$ accounts).
- `E1-P1A-CODE4`: Implementation of `internal/telemetry/loop.go` (60s ticker loop, `atomic.Bool` single-flight concurrency gate, manual refresh reset, clean shutdown).
- `E1-P1A-TEST1`: Unit tests with mock Google Cloud Code HTTP server verifying all bucket types, 100% / 0% quota, reset times, 401/403/500 errors, and network timeouts.
- `E1-P1A-TEST2`: Concurrency race testing (`go test -race ./internal/telemetry/...`) showing 0 data races.
- `E1-P1A-TEST3`: Benchmark test verifying parallel query latency < 2.0s.

---

## E2 - P2 (Slice P6A-2) Fyne UI Dashboard & Multi-Theme System Evidence Targets

- `E2-P2A-CODE1`: Implementation of `internal/ui/themes.go` (`DarkTheme`, `LightTheme`, `WarmTheme` satisfying `fyne.Theme`, theme switcher function, persistence in `accounts_store/.ui_preferences.json`).
- `E2-P2A-CODE2`: Implementation of `internal/ui/quota_badge.go` (mini progress bars for Claude, Gemini, GPT with Green >50%, Amber 15-50%, Red <15%, and recovery countdown label).
- `E2-P2A-CODE3`: Update to `internal/ui/widgets.go` (`NewAccountCard` embedding quota badges and per-row `[🔄]` single refresh button).
- `E2-P2A-CODE4`: Update to `internal/ui/window.go` (Header Toolbar with `[📊 Làm mới Quota Tất cả]`, auto-refresh 60s indicator with timestamp, and theme selector).
- `E2-P2A-CODE5`: Update to `internal/ui/app.go` (telemetry loop lifecycle, async non-blocking event dispatch via `fyne.Do()`, 60 FPS responsive UI).
- `E2-P2A-TEST1`: Unit tests in `internal/ui/` verifying theme switching without crash and badge color logic.
- `E2-P2A-QA1`: Verification of UI Traceability Matrix items `UI-TR-01` through `UI-TR-08`.

---

## E3 - P3 (Slice P6A-3) CLI Status Extension & Traceability QA Evidence Targets

- `E3-P3A-CODE1`: Update to `cmd/switcher/main.go` adding `--quota` flag and tabwriter output.
- `E3-P3A-CODE2`: Verification of dual-build configuration (`app_cli.go` and `app_gui.go`).
- `E3-P3A-TEST1`: Execution output of `antigravity-switch.exe --quota` showing tabular quota breakdown.
- `E3-P3A-BENCH1`: Verification of startup latency (<= 50ms CLI) and memory footprint (<= 20MB RAM).

---

## E4 - P4 Final Supervisor & Commit Closure Evidence Targets

- `E4-P4A-SUP1`: Supervisor validation report accepting all 3 slices against DoD and UI Traceability Matrix.
- `E4-P4A-GIT1`: Final Git commit hash for Phase 6A.

