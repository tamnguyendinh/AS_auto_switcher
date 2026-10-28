# Phase 1B Actual Status

Date: 2026-10-28
Status: IMPLEMENTED — verification deferred by Owner
Companion plan: 2026-10-28-phase-1b-plan.md

## Current Status Matrix
| Unit | Current state | Acceptance state | Next action |
|---|---|---|---|
| Core | 12 files: i18n, preferences, version, raw Wincred | Not runtime-verified after move | Deferred validation |
| Antigravity | 53 files in nine engine packages, including token adapter | Not runtime-verified after move | Deferred validation |
| UI/entrypoint | Imports updated; themes uses core preferences | Appearance/behavior not revalidated | Deferred validation |
| Codex | Only empty .gitkeep | Source inventory confirmed | No implementation in phase 1B |
| Build script | New version source/linker paths; explicit CC used for resource preprocessing | Pre-move compiler fix built successfully; post-move build deferred | Use available Clang for later validation |
| Old directories | No files remain; some empty nested directories | Owner will delete manually | Do not delete through tools |
| QA coverage | Existing tests migrated; raw boundary test authored | New tests not run; additional adapter error/empty-blob cases remain for QA pass | Deferred test work |

## Status Refresh Log
| Refresh | Basis | Change | Next |
|---|---|---|---|
| R0 | f1a05ca | Baseline flat source | Baseline investigation |
| R1 | Before moves | Clang standard build and serial all-package test passed after stale i18n test repair; native suite remained failing | Owner directed implementation first |
| R2 | Current worktree | R2 file moves/core split/caller updates implemented; 104 mapped destinations exist, no old mapped source files remain | Await deferred build/test; no acceptance claim |

## Ownership / authority
Owner prohibited agents and then explicitly deferred tests until after implementation. No independent-agent review is claimed. Existing untracked AGENTS.md and CLAUDE.md are preserved and excluded from task commits. No implementation commit has been made because verification/acceptance is deferred.

## Source inventory
12 core + 53 Antigravity + 37 UI + 4 cmd/resource + 1 Codex placeholder = 107 mapped/new paths, including generated syso. Source import scan found no old package imports in Go files across cmd/internal/reports. Historical PowerShell replay scripts and saved evidence remain historical and are not certified against the refactored tree.

## Remaining validation
Post-refactor standard build, no-cache unit suite, raw/token boundary tests, dependency checks, native GUI suite, release smoke and final source review. Prior baseline native failure: TestNativeTwoStoreRefreshAndResize waits for three cards and times out; an earlier resize drag failure did not recur in the isolated rerun. These are pre-move findings, not proof of refactor regression.