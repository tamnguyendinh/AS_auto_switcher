# Plan Update Report: Phase 6 (Go Engine Port, Fyne Native GUI, Standalone Binary) & Phase 7 (Go-Native Autonomous Quota Failover Daemon)

**Date:** 2026-10-15  
**Role:** Planner  
**Target Plan:** `docs/plans/2026-10-14-antigravity-multi-account-switcher/`  
**Authority:** Owner Directive via CEO Agent (`403f41a7-0810-44d9-b875-1e08c4ed13a0`), Architect Report `reports/architect/2026-10-14-autonomous-quota-failover-architecture.md`, Planner Skill `g:\antigravity_auto_switch\.agents\skills\planner\SKILL.md`  
**Status:** COMPLETE & COMMITTED  
**Verdict:** `[VERDICT: PASS / READY_FOR_ARCHITECT]`  

---

## 1. Executive Summary & Purpose

This report documents the formal expansion of the **Antigravity Multi-Account Switcher Plan Set** (`docs/plans/2026-10-14-antigravity-multi-account-switcher/`) to incorporate **Phase P6** and **Phase P7** directly from the approved Architect Specification (`reports/architect/2026-10-14-autonomous-quota-failover-architecture.md`).

While the Python prototype (Phases P0–P5) achieved 100% test pass rate (68/68 tests) and validated the 4-step closed-loop internal API switching architecture, running a continuous Python runtime imposes an unacceptable 40MB–80MB memory footprint, cold start delays, and interpreter dependencies. Furthermore, agentic developer workflows frequently encounter Google Cloud Code rate limits (HTTP 429 / gRPC `RESOURCE_EXHAUSTED`), breaking developer focus and subagent reasoning loops.

To solve these challenges permanently and build a production-grade developer tool, the plan set has been updated to incorporate:
1. **Phase P6 (High-Performance Go Engine, Fyne Native GUI & Standalone Windows Binary):**
   - Full port to compiled Go (Golang).
   - 100% Pure Go Native GUI (**Fyne**) with zero webview or browser runtime bloat (< 20MB RAM, < 50ms startup).
   - Direct Win32 `Advapi32.dll` integration using `golang.org/x/sys/windows` targeting `LegacyGeneric:target=gemini:antigravity` with zero CGo.
   - Single standalone compiled Windows executable: `antigravity-switch.exe` (10MB–15MB) with zero Python, pip, or external MSVC runtime dependencies.
2. **Phase P7 (Go-Native Autonomous Quota Failover Daemon & Auto-Switching Engine):**
   - Non-blocking background goroutine log tailer sniffing `%APPDATA%\Antigravity\logs\language_server.log`, detecting HTTP 429 and `RESOURCE_EXHAUSTED` in $\le 20\text{ms}$ with $< 0.05\%$ CPU.
   - Thread-safe `AccountPool` (`sync.RWMutex`) implementing account FSM (`ACTIVE`, `READY`, `QUOTA_EXHAUSTED`, `UNAUTH`, `ERROR`).
   - Round-robin / LRU scheduler with exponential backoff cooldowns (15m, 30m, 60m capped at 2h).
   - 15-second minimum dwell time hysteresis gate and sliding-window circuit breaker (max 4 switches per 120s) suppressing flapping storms.
   - Native Windows System Tray HUD with dynamic status dots (🟢 Healthy, 🟡 Active/Single, 🔴 All Exhausted with countdown tooltip) and discrete Windows Toast notifications.
   - Sub-second ($< 1.0\text{s}$) total automated failover cycle.
   - 100% IDE context preservation guarantee: open Monaco editor buffers (`isDirty = true`), terminal PTYs, and chat history remain completely unaffected.

---

## 2. Decomposed Implementation Slices

Adhering strictly to the Planner Skill structure, Phases P6 and P7 have been decomposed into concrete, atomic execution checklist items. Every slice is a complete mini-plan specifying Goal, Scope Boundary (Editable, Inspect-only, Preserve-only, Out of scope), Non-Goals, Pre-flight Questions (Data source, DB flow, Render location, Cleanup), Work Steps adhering to the invariant 6-step lifecycle (`Code -> Inspect -> Update tests -> Build -> QA -> Acceptance/Commit`), Implementation Gate, Acceptance Criteria, Evidence Targets, Actual-Status Update, and Commit Boundary.

### 2.1 Phase P6: Go Engine Port & Standalone Windows Binary

| Slice ID | Slice Title | Core Packages / Targets | Primary Verification Gate | Target Commit |
|---|---|---|---|---|
| **P6-A** | Go Core Primitives & Native Win32 Credential Layer | `go.mod`, `internal/model/`, `internal/wincred/`, `internal/store/` | Round-trip credential read/write via `Advapi32.dll`; atomic JSON store with path sanitizer | `feat(P6-A): implement Go core primitives and native Win32 credential layer` |
| **P6-B** | Chrome Discovery Scanner & Language Server Connector | `internal/scanner/`, `internal/connector/` | Chrome profile extraction with strict `anvien` blacklist filtering; language server PID/port discovery and `net/http` loopback RPCs | `feat(P6-B): implement Chrome discovery scanner and language server connector` |
| **P6-C** | Core Orchestrator & Fyne Native GUI | `internal/reloader/`, `internal/manager/`, `internal/ui/`, `cmd/switcher/` | Closed-loop 4-step hot-swap coordination; Fyne pure Go GUI (< 20MB RAM, < 50ms startup); dual CLI/GUI flags | `feat(P6-C): implement core orchestrator, Fyne native GUI, and CLI entrypoint` |
| **P6-D** | Standalone Windows Binary Compilation & Benchmark | `scripts/build_windows.ps1`, `antigravity-switch.exe` | Static binary $\le 15\text{MB}$, startup $\le 10\text{ms}$, RAM $\le 12\text{MB}$ (CLI) / $\le 20\text{MB}$ (GUI), zero Python/MSVC dependencies | `feat(P6-D): compile and benchmark standalone Windows binary antigravity-switch.exe` |

### 2.2 Phase P7: Go-Native Autonomous Quota Failover Daemon

| Slice ID | Slice Title | Core Packages / Targets | Primary Verification Gate | Target Commit |
|---|---|---|---|---|
| **P7-A** | Goroutine Log Tailer & Quota Sensor Pipeline | `internal/sensor/` (`log_tailer.go`, `signatures.go`, `event_bus.go`) | Non-blocking incremental log tailer detecting 429/RESOURCE_EXHAUSTED in $\le 20\text{ms}$; unbuffered Go channel emission | `feat(P7-A): implement goroutine log tailer and quota sensor pipeline` |
| **P7-B** | Go-Native Failover Controller & Account Pool FSM | `internal/controller/` (`pool_manager.go`, `scheduler.go`, `cooldown.go`, `hysteresis.go`) | Thread-safe `AccountPool` FSM (`-race` clean); 15s hysteresis gate; exponential cooldown recovery | `feat(P7-B): implement Go-native failover controller, account pool FSM, and hysteresis engine` |
| **P7-C** | Native Windows System Tray & Toast HUD | `internal/tray/` (`tray_daemon.go`, `notifications.go`, `hotkeys.go`) | Fyne `desktop.App` / native systray with dynamic color dots; discrete Windows toast alerts; `Win+Alt+S` hotkeys | `feat(P7-C): implement native Windows system tray daemon, toast notifications, and global hotkeys` |
| **P7-D** | End-to-End Simulation QA & Quota Storm Stress Testing | `tests/e2e/` (`e2e_test.go`, `quota_storm_test.go`) | Quota Storm stress testing across 3+ accounts; failover cycle $< 1.0\text{s}$; 100% IDE context preservation | `feat(P7-D): verify end-to-end autonomous failover, quota storm stress tests, and context preservation` |

---

## 3. Plan Set Synchronization Audit

The standard five-file plan set in `docs/plans/2026-10-14-antigravity-multi-account-switcher/` has been fully updated and synchronized:

1. **`2026-10-14-antigravity-multi-account-switcher-plan.md`:**
   - Metadata status updated to `in_progress`.
   - Goal updated with Phase P6 and Phase P7 production milestones.
   - Scope updated with all Go packages and build artifacts.
   - Requirements updated with Requirement 6 (Standalone Go Engine & Fyne Native GUI) and Requirement 7 (Autonomous Failover Daemon & Systray HUD).
   - Acceptance Criteria expanded with criteria 9 through 13.
   - Phases P0 through P5 preserved 100% intact as completed (`- [x]`).
   - Appended Phase P6 (`P6-A`, `P6-B`, `P6-C`, `P6-D`) and Phase P7 (`P7-A`, `P7-B`, `P7-C`, `P7-D`) with complete mini-plan specifications.
   - Updated Pn items (`Pn-A`, `Pn-B`, `Pn-C`) to uncompleted (`- [ ]`) with Implementation Gate covering all planned phases (P1 through P7).
   - Risk Notes expanded with Notes 5, 6, 7, 8 (CGo-free Win32 compatibility, Fyne OpenGL headless handling, log tailer rotation seek resilience, hotkey conflict safety).

2. **`2026-10-14-antigravity-multi-account-switcher-actual-status.md`:**
   - Metadata status updated to `in_progress`.
   - Target Scope expanded to include Phase 6 and Phase 7 Go components.
   - Relationship / Impact Evidence table updated with 14 new Go units.
   - Current Status Matrix updated with 14 new rows initially classified as `missing` with explicit Next Plan Decisions.
   - Status Refresh Log updated with row `R17` documenting incorporation of Phase 6 & Phase 7 from the Architect Report.
   - Phase Touch Map updated with touch modes and constraints for all 14 Go units.
   - Detailed Findings expanded with comprehensive file cards for each Go unit.
   - Next Phase Status Decisions table updated with rows for P6-A, P6-B, P6-C, P6-D, P7-A, P7-B, P7-C, P7-D.

3. **`2026-10-14-antigravity-multi-account-switcher-benchmark.md`:**
   - Preserved B0 through B5 baseline and final verification numbers.
   - Added `## B6 - P6 Benchmarks` table defining targets: binary size $\le 15.0\text{MB}$, CLI startup $\le 10\text{ms}$, GUI cold start $\le 50\text{ms}$, daemon RAM $\le 12.0\text{MB}$, GUI RAM $\le 20.0\text{MB}$, CPU $< 0.05\%$, external dependencies 0.
   - Added `## B7 - P7 Benchmarks` table defining targets: sensor detection $\le 20\text{ms}$, hysteresis dwell time $\ge 15\text{s}$, Quota Storm pass rate $100\%$, total failover latency $< 1.0\text{s}$, IDE context preservation $100\%$.

4. **`2026-10-14-antigravity-multi-account-switcher-evidence.md`:**
   - Preserved E0 through E5 and En-Pn closure evidence.
   - Added `## E6 - P6 Evidence: Go Engine Port & Standalone Binary` with planned targets `E6-P6A-WIN1`, `E6-P6A-STORE1`, `E6-P6A-MOD1`, `E6-P6B-SCAN1`, `E6-P6B-CONN1`, `E6-P6C-MGR1`, `E6-P6C-UI1`, `E6-P6D-BIN1`, `E6-P6D-PERF1`.
   - Added `## E7 - P7 Evidence: Go-Native Autonomous Quota Failover Daemon` with planned targets `E7-P7A-TAIL1`, `E7-P7A-SIG1`, `E7-P7B-POOL1`, `E7-P7B-HYST1`, `E7-P7C-TRAY1`, `E7-P7C-TOAST1`, `E7-P7D-STORM1`, `E7-P7D-LAT1`.

---

## 4. Compliance & Invariant Verification

- **Preservation of Completed Work:** Phases P0 through P5 in the plan set and codebase remain 100% untouched and preserved.
- **Strict Prohibition on Premature Code Modification:** Zero code files in `core/`, `scanner/`, `ui/`, `tests/` were modified. Zero Go source files were authored during this strategic planning slice.
- **Strict Blacklist Keyword Enforcement:** All scanner specifications enforce `["anvien"]` filtering with zero exceptions.
- **No CGo / No Webview Bloat:** Formally locked architecture to Fyne pure Go native GUI and `golang.org/x/sys/windows` native Win32 calls.

---

## 5. Summary & Handoff Packet

The plan update is complete and fully committed to Git. The project is primed for the next specialist lane (System Architect or Coder) to begin implementation starting with Slice `P6-A` (Go Core Primitives & Native Win32 Credential Layer).

- **Handoff Packet:**
  - Plan File: `docs/plans/2026-10-14-antigravity-multi-account-switcher/2026-10-14-antigravity-multi-account-switcher-plan.md`
  - Actual Status File: `docs/plans/2026-10-14-antigravity-multi-account-switcher/2026-10-14-antigravity-multi-account-switcher-actual-status.md`
  - Benchmark File: `docs/plans/2026-10-14-antigravity-multi-account-switcher/2026-10-14-antigravity-multi-account-switcher-benchmark.md`
  - Evidence File: `docs/plans/2026-10-14-antigravity-multi-account-switcher/2026-10-14-antigravity-multi-account-switcher-evidence.md`
  - Report Path: `reports/planner/2026-10-15-plan-update-phase6-phase7-report.md`
  - Next Action: CEO routes task to Coder lane for Slice `P6-A`.
