# Phase 1B Evidence Ledger

## Metadata
Date: 2026-10-28; companions: plan, actual-status, benchmark in this directory.

## Evidence Rules
Stable E<phase>-P<phase>A-<kind>1 IDs. Record exact commands/exit codes, redact sensitive values without hiding failures. Never report a skipped live integration as PASS. No independent-agent verdict is claimed after Owner prohibited agents.

## E0 - P0 Evidence
- E0-P0A-SRC1: HEAD f1a05ca; no production modifications; baseline untracked AGENTS.md/CLAUDE.md. R2 map and raw/token source/callers are authoritative.
- E0-P0A-ENV1: Go 1.26.3 windows/amd64; GCC 16.1.0 cc1/collect2 return -1073741515; standalone C probe also fails. Existing Clang 22.1.4 compiles same probe. Retrying baseline with process-local CC=Clang, no source/system change. Anvien analyze --force returns same DLL startup error.

## Closure Evidence
Pending actual implementation and verification.
## E1–E5 — implementation record (not acceptance)
- E1-P1A-SRC1: moved core source; extracted raw Wincred implementation without token/model imports; engine adapter keeps eight token methods, defaults, and shared sentinel identities. Preferences package and UI qualifiers changed, JSON/data paths preserved. Build script reads and injects version from core/version.
- E2-P2A-SRC1: moved 26 scanner/reloader/telemetry/connector files; updated caller imports in source/tests and historical Go probes.
- E3-P3A-SRC1: moved 25 model/store/manager/automation files; updated all Go callers and adapter model import.
- E4-P4A-SRC1: UI/entrypoint retain locations and logic except import/qualifier updates. Historical QA locale-file paths updated to core/i18n.
- E5-P5A-SRC1: Codex contains only a zero-byte .gitkeep. No engine.go or Codex integration added.
- E6-P6A-INVENTORY1: all 104 mapped destinations exist; no moved source remains at old path. Core=12 files, Antigravity=53, Codex=1. Source import search found no old imports. These static observations do not establish compile/runtime PASS.

## Baseline outcomes before Owner deferred tests
- Standard build with process-local CC=C:\msys64\ucrt64\bin\clang.exe passed after windres was taught to honor explicit CC. Output binary then was 29,273,088 bytes; it predates refactor and must not be delivered as the refactored build.
- go test -p 1 -count=1 -json ./... passed with isolated ANTIGRAVITY_STORE_DIR after repairing obsolete historical i18n assertions. Raw logs remain .tmp/phase-1b/baseline-final-unit.jsonl.
- Native baseline did not pass: TestNativeTwoStoreRefreshAndResize timed out waiting for card state. Earlier manual resize drag failure passed on isolated rerun. Evidence images in reports/QA/phase-1b/1b-0 are baseline-only, not refactor acceptance.
- Anvien and GCC child executables failed DLL startup with 0xC0000135; source inspection used instead of claiming graph verification.
- Restored only tracked PNG files overwritten by our baseline tests; preserved user data and unrelated files.

## Deferred by latest Owner instruction
No build/test/runtime suite run after moving source. New raw credential tests are authored but unexecuted. Additional adapter error-identity/invalid JSON/empty-blob coverage belongs to deferred QA work. Implementation remains uncommitted; no overall PASS or clean-worktree claim.