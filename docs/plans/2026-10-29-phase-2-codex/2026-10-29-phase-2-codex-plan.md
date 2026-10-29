# Phase 2 Codex Plan

## Metadata
- Date: 2026-10-29
- Status: code and full build delivered; native acceptance reserved for Owner
- Evidence: 2026-10-29-phase-2-codex-evidence.md
- Benchmark: 2026-10-29-phase-2-codex-benchmark.md
- Actual status: 2026-10-29-phase-2-codex-actual-status.md

## Goal
Implement the reviewed Phase 2 report: Codex navigation, honest UI-only placeholder, contextual footer and 53 package-only Go skeleton files mirroring Antigravity.

## Rules
- Owner explicitly authorized implementation and prohibited spawning agents. Execute sequentially in this chat, including source review and QA; do not claim independent-agent acceptance.
- Owner subsequently requested code and build only and will test directly. This supersedes automated desktop QA/closure steps below: deliver the binary after build, leave native acceptance and milestone commit pending Owner test.
- Preserve existing dirty Phase 1B ledgers and Antigravity automation changes.
- Code, inspect, update tests, full Windows build, native runtime QA, review/record results.
- Runtime is a Windows-native Fyne/Win32 application, not a web/Docker application. Validate the real compiled Windows runtime and native pointer events; Docker/Playwright browser checks are inapplicable to this target.
- Use isolated test profiles; no real account mutations. No backend implementation, external account operations or persistent tab preference.
- Record task checkpoints here/evidence; do not commit incomplete work or other existing changes.

## Problem
Sidebar has an unused Codex enum but no second item/view. Footer timers unconditionally restore Antigravity labels. Codex engine contains only .gitkeep. Phase 1B recorded a failing automation build gate; current dirty fixes must be preserved and verified.

## Scope
UI sidebar, view routing, footer visibility, Codex-only view, embedded logo, bilingual strings, package-only Codex tree, directly affected tests and this plan/evidence.

## Non-Goals
Codex authentication, accounts, discovery, quota, automation, data storage or fabricated version/detection. Changes to Antigravity engine behavior or historical plan acceptance.

## Requirements
- Exactly 53 Go files, matching relative Antigravity names, each only package declaration.
- Default tab Antigravity. Switching preserves existing view instances, scroll and backend loop lifetimes.
- Codex Add/Refresh disabled; Pin shares application state and updates both buttons.
- Codex footer says not integrated, version unavailable; no fake version.
- Transient Antigravity messages remain associated with Antigravity. Codex footer remains contextual even when a background operation or timeout completes. Returning to Antigravity restores any unexpired message.
- Apply VN/EN and existing dark/light/warm palette to all new UI. Use Card, not nonexistent CardBg.
- Official OpenAI extension logo embedded from verified publisher asset.

## Acceptance Criteria
Current delivery: code implemented, full Windows build and code tests pass, exact skeleton inventory checked. Native switching, footer, pin, language/theme and layout testing is reserved for Owner under the latest instruction; do not represent it as completed. Milestone acceptance/commit remains pending that testing.

## Checklist
- [x] P0-A: Inspect current source/dirty files, classify surfaces, compare 53-file manifest, locate native harness and official asset. Gate: actual-status completed with E0-P0A-SRC1; next action P1-A.

### P1: Codex tab shell
- Phase boundary: all surfaces above; Antigravity engine preserve-only.
- One executable slice P1-A: this is one navigation feature; sidebar, body and footer must switch atomically. Skeleton is the inert deliverable for this same release and has no separate runtime behavior.
- [x] P1-A: Add Codex tab shell and inert mirror tree (code/build handoff complete under revised Owner scope; native acceptance pending).
  - Goal: user can switch between preserved Antigravity and truthful Codex placeholder.
  - Scope Boundary: edit internal/ui routing/view/assets, internal/core/i18n, internal/engines/codex and related tests. Inspect-only build script and native harness. Preserve Antigravity and user data.
  - Non-Goals: real Codex services, account switching or API calls.
  - Pre-flight Questions: strings from core i18n; palette from UI themes; sidebar/body/footer render in main Fyne window. No DB read/write. Windows runtime; browser/container N/A. Tests use temporary preferences/accounts. Only external side effect is downloading public official brand asset.
  - Work Steps:
    1. Code:
       - Task 1.1: Mirror actual Antigravity relative paths into Codex with package declarations. Inputs: verified 53-file inventory. Edit scope: Codex only. Output: 53 inert files. Verify exact paths/content. Checkpoint: wip(P1-A): task 1.1 - mirror created.
       - Task 1.2: Add official logo, localized Codex view and sidebar item. Inputs: palette, i18n, official asset. Edit scope: UI/assets and core/i18n. Output: disabled actions, placeholder, shared pin. Verify source routing and gofmt. Checkpoint: wip(P1-A): task 1.2 - Codex view wired.
       - Task 1.3: Centralize footer rendering and view switching. Inputs: selected tab and transient message. Edit scope: UI app/window/navigation. Output: contextual rendering without engine lifecycle calls. Verify all status show/clear paths route through renderer. Checkpoint: wip(P1-A): task 1.3 - contextual footer wired.
    2. Inspect: review diff, gofmt, caller graph; confirm Antigravity engine untouched.
    3. Update tests: navigation/scroll identity, background liveness, message expiry while Codex active, VN/EN/theme/pin, inert inventory. Adapt obsolete phase-one-only structural expectations to contextual containers without weakening behavior assertions.
    4. Build: scripts/build_windows.ps1 with existing working Clang and isolated store; go test ./... with fresh isolated data.
    5. QA: native test binary using existing Win32 pointer harness, screenshots and release binary smoke; evidence under reports/qa/phase-2-codex in Markdown and JSON.
    6. Acceptance: re-read diff and evidence using supervisor protocol locally (no agent), refresh ledgers, preserve unrelated files. Commit only accepted scoped files; no independent Supervisor claim.
  - Implementation Gate: P0 complete; manual imports/call paths used when Anvien fails.
  - Acceptance: exact inert mirror; UI tests/native behavior pass; build passes; no Antigravity source regression; no test profile modifies live accounts.
  - Evidence Targets: E1-P1A-SRC1, E1-P1A-BUILD1, E1-P1A-TEST1, E1-P1A-NATIVE1.
  - Actual-status Update: missing -> implemented -> verified only on corresponding evidence.
  - Commit Boundary: this slice only, after validation and local review; preserve unrelated dirty changes.
- [x] Pn-A: Local supervisor-protocol source/build handoff review; no subagent. Native acceptance remains with Owner.
- [x] Pn-B: Review plan-created outputs; active build evidence and requested package-only skeletons retained, no native probe created.
- [x] Pn-C: Record final code/build results and existing dirty files. Milestone commit deferred until Owner direct test.

## Risk Notes
Footer timeout can reveal the wrong engine; shared pin must update both buttons. Existing tests inspect container depth and old one-item sidebar. Native harness must not run alongside production window because pin resolves by title. Baseline automation failure may already have been repaired by existing dirty changes; do not overwrite those changes.
