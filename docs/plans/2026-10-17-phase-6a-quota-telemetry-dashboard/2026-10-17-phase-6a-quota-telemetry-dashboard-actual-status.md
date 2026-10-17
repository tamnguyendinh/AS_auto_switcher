# Phase 6A: Multi-Account Quota Telemetry Engine, Live Dashboard & Multi-Theme System Actual Status

Title: Phase 6A: Multi-Account Quota Telemetry Engine, Live Dashboard & Multi-Theme System Actual Status
Date: 2026-10-17
Status: in_progress (Slice P6A-3 Complete -> Next Step: P4 Final Supervisor Acceptance)
Companion plan: `docs/plans/2026-10-17-phase-6a-quota-telemetry-dashboard/2026-10-17-phase-6a-quota-telemetry-dashboard-plan.md`
Companion rules: `docs/plans/2026-10-17-phase-6a-quota-telemetry-dashboard/2026-10-17-phase-6a-quota-telemetry-dashboard-plan.md`
Companion evidence: `docs/plans/2026-10-17-phase-6a-quota-telemetry-dashboard/2026-10-17-phase-6a-quota-telemetry-dashboard-evidence.md`
Companion benchmark: `docs/plans/2026-10-17-phase-6a-quota-telemetry-dashboard/2026-10-17-phase-6a-quota-telemetry-dashboard-benchmark.md`

## Purpose

This file records the real current state before implementation of Phase 6A.
Implementation must not start until the target scope has a completed status row, evidence IDs, and a downstream plan decision.
This file classifies current state from empirical evidence gathered from the codebase and architecture specification.

## Freshness / Refresh Rules

This actual-status file is a living current-state record, not a one-time P0 snapshot.
P0 records the baseline before implementation. After implementation begins, keep the Current Status Matrix updated so the next agent can trust it as the latest repo reality.

Update this file:
- After each completed implementation slice (P6A-1, P6A-2, P6A-3);
- Before starting the next phase if repo state changed;
- Whenever evidence changes a current-state classification;
- Whenever the next phase's status assumptions, next action, or work steps need updating because reality differs from the previous status.

When refreshing status:
- Update only the rows affected by the completed work or new evidence;
- Use explicit transitions such as `missing -> correct`, `partial -> correct`, `unbound -> bound-correct`;
- Append a Status Refresh Log row instead of deleting history;
- Keep detailed proof in `evidence.md`; store only classifications, evidence IDs, touch mode, and plan consequences here.

## Scope

Target scope:
- `internal/telemetry/`: New package for Google Cloud Code Quota Telemetry Client, Worker Pool, Two-Tier Cache, and 60s Auto-Telemetry Loop.
- `internal/ui/`: Fyne Native GUI extension with Quota Badges, Refresh All / Row Refresh controls, Multi-Theme Engine (Dark, Light, Warm), and Non-Blocking Event Dispatch.
- `cmd/switcher/`: CLI extension with `--quota` flag and tabwriter output.
- `accounts_store/`: Integration of `.quota_cache.json` and `.ui_preferences.json`.

Out of scope:
- `internal/reloader/`: Session reloader and thread preservation (Preserve-only).
- `internal/wincred/`: Windows Credential Manager integration (Preserve-only).
- `internal/scanner/`: Chrome profile scanner (Preserve-only).
- `internal/manager/`: Core account switching state machine (Preserve-only; inspect-only for account listing).
- Phase 7 Autonomous Failover Daemon: Strictly reserved for subsequent milestone.

## Current Status Matrix

| Surface / File / Component | Expected / Required Capability | Current Status | Evidence IDs | Touch Mode | Downstream Plan Decision & Next Action |
|---|---|:---:|---|:---:|---|
| `internal/telemetry/` | Quota telemetry client for Google `v1internal:retrieveUserQuota`, concurrent worker pool, two-tier cache, and 60s loop | `correct` | E1-P1A-CODE1..CODE4, E1-P1A-TEST1..TEST3 | Preserve-only / Reusable | Quota telemetry engine, two-tier cache, parallel worker pool, and 60s loop fully operational and accepted by Supervisor (commit `c87939b`). |
| `internal/connector/oauth_refresher.go` | Google CloudCode OAuth2 token refresher with verified client credentials | `correct` | E0-P0A-SRC2 | Preserve-only / Reusable | Reuse existing `GoogleOAuthRefresher` in `internal/telemetry/client.go` to obtain fresh access tokens for quota queries. |
| `accounts_store/*.json` | 7 stored accounts with valid refresh tokens | `correct` | E0-P0A-STORE1 | Preserve-only / Read-only | Use as credential source for worker pool parallel quota queries; do not corrupt existing credentials. |
| `internal/ui/themes.go` | Multi-Theme System: Dark (Catppuccin Mocha), Light (Paper Slate), Warm (Amber/Sepia) satisfying `fyne.Theme` | `correct` | E2-P2A-CODE1, E2-P2A-TEST1, E2-P2A-QA1 | Preserve-only / Reusable | 3 themes (Dark, Light, Warm) satisfy `fyne.Theme`, atomic JSON persistence in `accounts_store/.ui_preferences.json`, accepted by Supervisor (commit `dd86d77`). |
| `internal/ui/quota_badge.go` | Visual mini quota progress bars / badges with 3-tier color coding and countdown text | `correct` | E2-P2A-CODE2, E2-P2A-TEST1, E2-P2A-QA1 | Preserve-only / Reusable | Quota badges and countdown widget implemented with exact color coding (>50% Green, 15-50% Amber, <15% Red) and recovery timestamp formatting, accepted by Supervisor (commit `dd86d77`). |
| `internal/ui/widgets.go` | Account row card embedding mini quota badges and per-row `[🔄]` single refresh button | `correct` | E2-P2A-CODE3, E2-P2A-TEST1, E2-P2A-QA1 | Preserve-only / Reusable | Account card updated with quota badges and row-level refresh button callback, accepted by Supervisor (commit `dd86d77`). |
| `internal/ui/window.go` | Header Toolbar with `[📊 Làm mới Quota Tất cả]`, auto-refresh 60s indicator, and theme selector | `correct` | E2-P2A-CODE4, E2-P2A-TEST1, E2-P2A-QA1 | Preserve-only / Reusable | Header Toolbar updated with global refresh button, 60s auto-refresh indicator, theme dropdown, and active summary card, accepted by Supervisor (commit `dd86d77`). |
| `internal/ui/app.go` | Application controller wiring telemetry loop, handling async updates, maintaining 60 FPS non-blocking UI | `correct` | E2-P2A-CODE5, E2-P2A-TEST1, E2-P2A-QA1 | Preserve-only / Reusable | Telemetry loop and async event dispatch via `fyne.Do()` wired with single-flight concurrency, maintaining 60 FPS non-blocking UI, accepted by Supervisor (commit `dd86d77`). |
| `cmd/switcher/main.go` | CLI flag `--quota` rendering tabular quota distribution across all accounts | `correct` | E3-P3A-CODE1, E3-P3A-TEST1, E3-P3A-BENCH1 | Preserve-only / Reusable | Implemented `--quota` flag, `runQuotaCLI`, and `renderQuotaTable` with tabwriter formatting in Slice P6A-3. Fully verified. |
| `cmd/switcher/app_cli.go` & `app_gui.go` | Dual-build support for pure CLI and native Fyne GUI | `correct` | E3-P3A-CODE2, E3-P3A-TEST1 | Preserve-only / Reusable | Synchronized `launchGUI` signatures across pure Go CLI (`CGO_ENABLED=0`) and Fyne GUI (`-tags gui`) build targets. |
| `internal/reloader/` | Language Server hot-swap and session reload | `correct` | E0-P0A-SRC5 | Preserve-only | Do not touch; ensure quota operations do not trigger false reloads or process termination. |
| `internal/wincred/` | Win32 Credential Manager persistence | `correct` | E0-P0A-SRC6 | Preserve-only | Do not alter WinCred format or keys. |

## Status Refresh Log

| Refresh Event | Date | Triggering Slice | Status Transitions | Evidence IDs | Next Step / Plan Impact |
|---|---|---|---|---|---|
| P0 Baseline Audit | 2026-10-17 | P0 Baseline | Established initial matrix (telemetry: missing, ui: partial, cli: partial, connector/store/reloader: correct) | E0-P0A-SRC1..SRC6, E0-P0A-STORE1, E0-P0A-TEST1 | Proceed to Slice P6A-1 implementation without modifying preserve-only components. |
| Slice P6A-1 Completion | 2026-10-17 | Slice P6A-1 | internal/telemetry/: missing -> correct | E1-P1A-CODE1..CODE4, E1-P1A-TEST1..TEST3 | Quota telemetry engine, two-tier cache, parallel worker pool, and 60s loop fully operational. Proceed to Slice P6A-2 (UI Dashboard & Multi-Theme). |
| Slice P6A-2 Completion | 2026-10-17 | Slice P6A-2 | internal/ui/ (themes, quota_badge, widgets, window, app): missing/partial -> correct | E2-P2A-CODE1..CODE5, E2-P2A-TEST1, E2-P2A-QA1 | Fyne UI quota dashboard, refresh actions, and multi-theme system verified and accepted by Supervisor (commit `dd86d77`). Proceed to Slice P6A-3. |
| Slice P6A-3 Completion | 2026-10-17 | Slice P6A-3 | cmd/switcher/main.go: partial -> correct; app_cli.go & app_gui.go: correct | E3-P3A-CODE1..CODE2, E3-P3A-TEST1, E3-P3A-BENCH1 | CLI `--quota` flag, dual-build targets, automated Windows binaries build, and UI Traceability Matrix verified 100%. Ready for P4 Final Supervisor Acceptance. |
