# Phase 1B Plan

## Metadata
- Date: 2026-10-28
- Status: source implementation complete; post-refactor verification deferred by Owner
- Authority: reports/system-architect/rp_system-architect_261028_220000_by_antigravity_phase_1b_antigravity_backend_directory_refactoring_and_core_isolation.md (R2, f1a05ca)
- Companions: 2026-10-28-phase-1b-actual-status.md, 2026-10-28-phase-1b-evidence.md, 2026-10-28-phase-1b-benchmark.md

## Goal
Implement the R2 file map in ordered slices, preserving Antigravity behavior and isolating reusable core.

## Rules
Owner explicitly requires direct execution without agents. Implementation, QA and a separate source/evidence review are performed by this assistant; do not claim independent-agent acceptance. This overrides the previous lane-dispatch requirement. Preserve all R2 technical gates.
Use the Windows Fyne build scripts/build_windows.ps1 and real desktop; Docker/browser template clauses are N/A to this native application. Do not create WIP commits with broken import graphs. Commit each accepted slice, only owned paths.
Anvien executable currently fails before starting (0xC0000135); use R2 caller map plus source import scans, go list and git diff for impact verification, record the tool limitation.
Preserve untracked AGENTS.md and CLAUDE.md present at resume. No global environment changes or deletion of user data.

## Problem
Flat packages mix shared Win32 infrastructure and Antigravity token adapters; moving paths without updating callers/build metadata would break compilation.

## Scope
R2 tables 2.1, 2.2, 3 define the exact production/test/build touch map (104 baseline paths plus three outputs and build script). New plan/evidence files are task-owned. Historical Go QA packages outside cmd/internal must be checked because go test ./... includes them; import-only adjustments are allowed if required to keep their existing behavior compiling. No source algorithm changes.

## Non-Goals
Codex integration, IDEEngine, UI redesign, schema migration, dependency upgrades, behavioral cleanup.

## Requirements
Core cannot depend on engine/domain/UI. Token adapter preserves signatures, defaults and sentinel identities. Preferences schema/path unchanged. All moved callers updated atomically. Codex contains only an empty .gitkeep.

## Acceptance Criteria
R2 gates G, native four tests, release smoke, dependency/resource/import scan; original tests preserved, direct raw/token boundary coverage added; no claims for skipped live integrations.

## Checklist
- [ ] P0-A / 1B.0: baseline inventory, no-cache unit suite, standard build, native tests and isolated release smoke. No production edits until baseline succeeds or a documented baseline issue is repaired.

### P1: Ordered refactoring
Each slice below executes the shared six-step lifecycle: (1) code leaf actions, (2) inspect diff/import graph, (3) migrate/update behavior tests, (4) full standard build and no-cache suite, (5) native tests + isolated release smoke, (6) acceptance review, evidence/status update and owned commit.
Common preflight: data is existing stores/Wincred; no schema writes; render is native Fyne window; default credentials read-only, writes only to test targets; QA store override and auto_switch:false required; temporary artifacts in .tmp; no Docker/Playwright target.
Each task checkpoint is a local WIP note, not an unverified milestone commit. Evidence targets: reports/QA/phase-1b/1b-N and exact EN-PNA-VERIFY1 IDs below. Detailed file/caller lists are authoritative R2 tables, not inferred by directory name.

- [ ] P1-A / 1B.1: core and token adapter.
  - Goal/scope: 12 baseline moves; raw/token split, preferences package/qualifier, caller imports and version build paths; preserve UI/data behavior.
  - Leaf 1: move version/i18n/preferences using R2 map; input existing files, output core packages; verify their tests and embed data equality.
  - Leaf 2: split raw credential and engine token adapter; input cred.go/types.go, output exact R2 five-file credential boundary; verify raw roundtrip, validation, parse, errors.Is identity.
  - Leaf 3: update all R2 1B.1 callers and build version source/linker paths; verify go list and standard build.
  - Gate: P0 PASS. Acceptance: core has no domain dependency, all G/native/smoke gates pass. Refresh core status; commit only slice source and evidence. Evidence E1-P1A-VERIFY1.
- [ ] P2-A / 1B.2: move scanner/reloader/telemetry/connector (26 files).
  - Goal/scope: only R2 move targets and their caller imports (including unmoved manager/UI/tests).
  - Leaf 1: move four packages preserving contents; output engine subtree. Leaf 2: update exact imports, preserving still-unmoved domain paths. Verify focused tests then G/native/smoke.
  - Gate: P1 accepted/committed. Acceptance: no dangling moved import, behavior unchanged. Refresh backend status and commit slice; E2-P2A-VERIFY1.
- [ ] P3-A / 1B.3: move model/store/manager/automation (25 files).
  - Goal/scope: remaining domain files, R2 callers and adapter model import; preferences excluded.
  - Leaf 1: move exact mapped files. Leaf 2: update all callers including historical compiled QA files when needed. Verify engine suite then G/native/smoke.
  - Gate: P2 accepted/committed. Acceptance: domain paths eliminated, core independent. Refresh backend status and commit slice; E3-P3A-VERIFY1.
- [ ] P4-A / 1B.4: integration audit.
  - Goal/scope: read all UI/entrypoint mappings and tagged files; edit only missed import/qualifier paths, no new behavior.
  - Work: inspect UI/entrypoint graph, run G/native/release smoke; no fake edits or empty commit if already correct.
  - Gate: P3 accepted. Acceptance: UI/startup/assets/preferences unchanged. Record E4-P4A-VERIFY1 and refresh status; evidence-only commit if applicable.
- [ ] P5-A / 1B.5: Codex placeholder.
  - Goal/scope: create internal/engines/codex/.gitkeep (zero bytes) only.
  - Work: create file, inspect directory, run G/native/smoke, record E5-P5A-VERIFY1 and commit.
  - Gate: P4 accepted. Acceptance: no subfolders/Go source/Codex imports.
- [ ] P6-A / 1B.6: final acceptance and cleanup.
  - Goal/scope: verify map, direct/transitive dependencies, import/build symbols, source diff, test coverage and release runtime; no code expansion.
  - Work: run final gates, review source/evidence using Supervisor protocol directly per Owner no-agent instruction; fix only rejected invariant; record truthful limitations.
  - Acceptance: all required behavior proven, no task-created dead artifacts; commits and worktree boundary accounted for. Final evidence E6-P6A-VERIFY1; update all ledgers.

## Risk Notes
GCC subprocess DLL startup failure on baseline; test with available Clang process-local CC. Native harness cannot coexist with release window. Release smoke needs ANTIGRAVITY_STORE_DIR; cwd alone is insufficient. Live tests may SKIP and may log sensitive data: redact before retaining evidence.
## P0 baseline repair addendum
Before moves, update only reports/QA/i18n-coverage/render_test.go: use actual Dictionary JSON key contract rather than obsolete fixed 53 count; align modal labels with already-accepted current locale source and modal-i18n-candidate tests; isolate account store before NewApp so host preferences cannot affect default locale. No production changes. Preserve/restore tracked screenshot outputs generated by baseline suite. Existing two UI intermittent failures passed focused rerun; standard build retest determines baseline outcome.

Baseline build addendum: windres still launches broken GCC despite process CC=Clang. Permit scripts/build_windows.ps1 to pass the explicitly selected CC as resource preprocessor (resolved executable, -E -xc -DRC_INVOKED). Preserve default windres behavior when CC unset. This is a toolchain-consistency fix, separate from refactor paths.

## Owner execution override
Latest Owner instruction: focus on implementation; tests deferred. Execute R2 source slices in order without running more tests/build/runtime gates now. No agent spawning. Mark implementation separately from acceptance; do not commit unverified implementation as accepted. Baseline unit suite and standard build passed after documented repairs; native baseline has unresolved failures and is not an acceptance claim.

## Current handoff
Source actions P1-A through P5-A implemented. Checklist acceptance boxes intentionally remain unchecked until deferred verification. No agents, no post-refactor build/test and no acceptance commit. Next work is deferred validation and any proven repairs, not more feature development.

## Automation failure repair — current Owner request
Claim: determine whether EdgeCase7 failure is refactor regression or timing/isolation, fix root cause, rerun full standard build. Source comparison confirms controller.go, retry.go, switch_gate_windows.go and automation_e2e_test.go are identical to f1a05ca except moved import paths. Repeated isolated EdgeCase7 runs reproduce both missing final switch and missing retry dispatch.
Hypothesis to prove: the server increments retry-received before client releases the shared SwitchGate; a following retry/switch calls TryExecute and is dropped as busy. Restrict repair to controller.go retry-send serialization/draining and runtime_regression_test.go deterministic gate handoff tests. Preserve cross-process TryExecute rejection and retry policy. Verify cancellation/OFF/manual boundaries, repeated EdgeCase7, automation suite, then scripts/build_windows.ps1 with explicit Clang CC. No agents, no global toolchain change, no commit/whole-refactor acceptance implied.

Reproduction also exposed a separate harness teardown race: EdgeCase7 exits once the credential email changes while the switch still writes timeline files. Extend the existing case's completion predicate to require IsSwitching=false and assert exactly one StatusSwitched with exactly three retries. Scope additionally includes automation_e2e_test.go, retaining its existing two-second limit.
