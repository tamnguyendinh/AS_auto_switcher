# Phase 1B — source implementation handoff

Status: IMPLEMENTED / VERIFICATION DEFERRED
Authority: R2 architecture report; latest Owner instructions forbid agents and defer tests.
Git base: f1a05ca. Changes are in the working tree, not committed or accepted.

## Implemented
- Shared core: raw Win32 credential API, preferences, i18n, version.
- Antigravity: nine packages including token adapter; original method signatures, credential target/user and error identities preserved.
- Updated UI/entrypoint/test/probe imports and two locale-file paths in historical QA Go packages.
- Build source path and version linker symbols moved to core/version; resource preprocessor honors explicitly selected CC.
- Codex only has an empty .gitkeep; no speculative IDEEngine.

## Invariant family
Core→platform dependencies only; engine→core; UI/entrypoint→engine/core. Data schemas and runtime paths preserved. Source scan shows no stale package import in Go files. All 104 mapped destinations exist; 12 core/53 Antigravity files present.

## Separate baseline repairs already made
Historical i18n coverage now checks the dictionary JSON contract instead of obsolete key count, matches current accepted labels, and isolates preference loading. The build-script CC resource change allows installed Clang to work around local GCC DLL startup failure.

## Deferred verification / remaining work
No post-refactor build, unit test, native GUI or release smoke was run, per Owner. Existing baseline native two-store failure remains recorded, not hidden. New raw tests not run; additional adapter error/empty-blob tests await QA work. Old historical PowerShell overlay replay scripts remain tied to old revisions and are not current acceptance commands.

## Cleanup and preservation
Owner received 11 former source directories for manual deletion; recursive inventory found no files in them. Automatic cleanup was blocked by policy. AGENTS.md and CLAUDE.md untracked at resume were preserved. Baseline screenshots are explicitly baseline evidence, not final acceptance. No user account/credential data was intentionally removed or migrated.

## Next
Run deferred validation when requested; fix only proven issues and then commit accepted source/evidence. Do not treat the existing bin executable as a refactored build.