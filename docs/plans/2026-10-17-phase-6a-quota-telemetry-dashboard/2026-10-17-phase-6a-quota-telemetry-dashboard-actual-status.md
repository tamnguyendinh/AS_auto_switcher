# Phase 6A: Multi-Account Quota Telemetry Engine, Live Dashboard & Multi-Theme System Actual Status

Title: Phase 6A: Multi-Account Quota Telemetry Engine, Live Dashboard & Multi-Theme System Actual Status
Date: 2026-10-17
Status: in_progress (Slice P6A-1 Complete -> Next Slice: P6A-2)
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
| `internal/telemetry/` | Quota telemetry client for Google `v1internal:retrieveUserQuota`, concurrent worker pool, two-tier cache, and 60s loop | `correct` | E1-P1A-CODE1..CODE4, E1-P1A-TEST1..TEST3 | Preserve-only / Reusable | Quota telemetry engine, two-tier cache, parallel worker pool, and 60s loop fully operational and accepted by Supervisor (commit `c87939b`). Ready for Slice P6A-2 (UI Dashboard & Multi-Theme). |
| `internal/connector/oauth_refresher.go` | Google CloudCode OAuth2 token refresher with verified client credentials | `correct` | E0-P0A-SRC2 | Preserve-only / Reusable | Reuse existing `GoogleOAuthRefresher` in `internal/telemetry/client.go` to obtain fresh access tokens for quota queries. |
| `accounts_store/*.json` | 7 stored accounts with valid refresh tokens | `correct` | E0-P0A-STORE1 | Preserve-only / Read-only | Use as credential source for worker pool parallel quota queries; do not corrupt existing credentials. |
| `internal/ui/themes.go` | Multi-Theme System: Dark (Catppuccin Mocha), Light (Paper Slate), Warm (Amber/Sepia) satisfying `fyne.Theme` | `missing` | E0-P0A-SRC3 | Editable | Implement in Slice P6A-2 with hot-switch capability and persistence in `accounts_store/.ui_preferences.json`. |
| `internal/ui/quota_badge.go` | Visual mini quota progress bars / badges with 3-tier color coding and countdown text | `missing` | E0-P0A-SRC3 | Editable | Implement in Slice P6A-2: badges for Claude 4.6, Gemini 2.5, GPT-OSS with Green (>50%), Amber (15-50%), Red (<15%). |
| `internal/ui/widgets.go` | Account row card embedding mini quota badges and per-row `[🔄]` single refresh button | `partial` | E0-P0A-SRC3 | Editable | Update `NewAccountCard` in Slice P6A-2 to display quota badges and individual refresh trigger. |
| `internal/ui/window.go` | Header Toolbar with `[📊 Làm mới Quota Tất cả]`, auto-refresh 60s indicator, and theme selector | `partial` | E0-P0A-SRC3 | Editable | Update layout in Slice P6A-2 to add global refresh button, theme selector, and countdown label. |
| `internal/ui/app.go` | Application controller wiring telemetry loop, handling async updates, maintaining 60 FPS non-blocking UI | `partial` | E0-P0A-SRC3 | Editable | Integrate `telemetry.AutoLoop` in Slice P6A-2; dispatch UI updates via `fyne.Do()` without freezing main thread. |
| `cmd/switcher/main.go` | CLI flag `--quota` rendering tabular quota distribution across all accounts | `partial` | E0-P0A-SRC4 | Editable | Implement flag parsing and tabwriter formatting in Slice P6A-3. |
| `cmd/switcher/app_cli.go` & `app_gui.go` | Dual-build support for pure CLI and native Fyne GUI | `correct` | E0-P0A-SRC4 | Editable | Maintain dual-build tags while exposing `--quota` in both modes. |
| `internal/reloader/` | Language Server hot-swap and session reload | `correct` | E0-P0A-SRC5 | Preserve-only | Do not touch; ensure quota operations do not trigger false reloads or process termination. |
| `internal/wincred/` | Win32 Credential Manager persistence | `correct` | E0-P0A-SRC6 | Preserve-only | Do not alter WinCred format or keys. |

## Status Refresh Log

| Refresh Event | Date | Triggering Slice | Status Transitions | Evidence IDs | Next Step / Plan Impact |
|---|---|---|---|---|---|
| P0 Baseline Audit | 2026-10-17 | P0 Baseline | Established initial matrix (telemetry: missing, ui: partial, cli: partial, connector/store/reloader: correct) | E0-P0A-SRC1..SRC6, E0-P0A-STORE1, E0-P0A-TEST1 | Proceed to Slice P6A-1 implementation without modifying preserve-only components. |

| Slice P6A-1 Completion | 2026-10-17 | Slice P6A-1 | internal/telemetry/: missing -> correct | E1-P1A-CODE1..CODE4, E1-P1A-TEST1..TEST3 | Quota telemetry engine, two-tier cache, parallel worker pool, and 60s loop fully operational. Proceed to Slice P6A-2 (UI Dashboard & Multi-Theme). |

