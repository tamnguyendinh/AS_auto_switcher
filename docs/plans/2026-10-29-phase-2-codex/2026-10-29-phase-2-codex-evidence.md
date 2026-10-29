# Phase 2 Codex Evidence Ledger

## Metadata
Date: 2026-10-29. Companions: same-prefix plan, actual-status and benchmark.
## Evidence Rules
Record exact commands and results; source inspection is not runtime proof. Stable E<phase>-P<phase>A-<kind>1 IDs. No independent-agent acceptance (Owner prohibits agents).
## E0 - P0 Evidence
E0-P0A-SRC1: full report read; 53 declared names equal actual Antigravity paths. Read UI sidebar/window/app/theme/assets, i18n and native harness. Existing dirty files: three Phase 1B ledgers plus automation controller.go, automation_e2e_test.go and runtime_regression_test.go. Preserve all six. Prior ledger records EdgeCase7 failure, not re-established as current failure.
E0-P0A-GRAPH1: anvien file-detail internal/ui/app.go --repo . --json produced no output; imports and callers inspected directly.
E0-P0A-ASSET1: official publisher page https://marketplace.visualstudio.com/items?itemName=OpenAI.chatgpt exposes https://openai.gallerycdn.vsassets.io/extensions/openai/chatgpt/26.51021.22027/1792788319200/Microsoft.VisualStudio.Services.Icons.Default . Use original bytes, record hash after download.
## E1 - P1 Evidence
E1-P1A-SRC1: 53 Codex Go files created with only package declarations. Persistent Codex view and sidebar item added, with centralized footer visibility in navigation.go. Add/Refresh disabled, shared pin, bilingual content and dark/light/warm tokens. No Antigravity engine edits by this task. Asset hash matches E0-P0A-ASSET1: 2ce1c57dd3b312417106a487815ab1235bddce17e1fa9b32a2f048bd71ee25d3.

E1-P1A-TEST1: `go test ./internal/ui -run 'TestCodex|TestSidebarDefault|TestAcceptance_TC0[1-5]' -count=1` PASS, 9.689s. Covers tab/content preservation, active background liveness, loop instance/running state, message timeout on Codex, returning to pending/expired Antigravity feedback, localized content, themes, shared pin, skeleton path/content equality and existing footer regression.

During implementation, the initial compile found a Dictionary pointer/value mismatch; corrected the new view signature. A navigation test initially counted the pre-existing empty-state item as an account; corrected fixture to clear that item before adding test rows. Both issues are resolved by the fresh passing focused suite.

E1-P1A-DIFF1: `git diff --check` exit 0. UI source/diff reviewed; navigation has no lifecycle start/stop calls, all footer visibility transitions use renderFooter, new view has no engine imports.

E1-P1A-NATIVE1: NOT RUN. Computer Use could not capture the old minimized app (`FrameArrived timed out` after restore attempt). Owner then explicitly requested code/build only and stated they will test directly. No further desktop automation or native test executed. No runtime acceptance claimed.

E1-P1A-BUILD1: `scripts/build_windows.ps1` PASS, exit 0, no SkipTests. Output in reports/qa/phase-2-codex/build.log; process-local CC=C:\msys64\ucrt64\bin\clang.exe, CGO_ENABLED=1, isolated ANTIGRAVITY_STORE_DIR=.tmp/phase-2-codex/test-profile. Vet and all cmd/internal package tests pass; UI 35.442s. Antigravity automation passes from existing workspace changes (cached; not attributed to this task). Codex packages correctly report no tests to run; separate UI inventory test verifies their inert contents. GUI binary bin/antigravity-switch.exe is 29,313,024 bytes, compilation 7,324ms, PE icon verified. JSON result includes binary hash. Old running executable was stopped by the standard build script; new binary not launched, per Owner direct testing handoff.
## Closure Evidence
Owner acceptance and milestone commit deferred to direct testing. Existing six dirty Phase 1B/Antigravity files preserved. No agents spawned.
