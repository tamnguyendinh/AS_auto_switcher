# Supervisor Report: README alignment with the Go application

Verdict: PASS

## Scope and authority

- Review date: 2026-10-19, Asia/Bangkok (UTC+7).
- Reviewer: Codex (GPT-6).
- Target: `README.md`; documentation changes only.
- Claim reviewed: describe the current Go application, replace obsolete application instructions, and preserve the introduction explicitly requested by the owner.
- Authority: the owner's README correction request, clarification that the former application is no longer used, and subsequent explicit confirmation of the requested introductory sentence; current Go source and build script.
- No applicable `AGENTS.md` was found in the repository or its parent directory.
- Reviewed README SHA-256: `1067C998A45E274D90E8B2D22493D388478E8B2D2E1B22FD2E740028C4B31ADF`.

## Source-level clearance

- `README.md`: replaces the former launch/test workflow with Go executables, build tags, toolchain requirements, and existing Go test commands. No production source was edited by this task.
- Account discovery/capture: `internal/manager/account_manager.go:192`, `:251`, and `:306` establish credential capture and the union of Chrome and stored accounts, including accounts absent from Chrome.
- Switching: `internal/manager/account_manager.go:378`, `:488`, and `:543` establish user-triggered switching, the eight-second wait and verification against the new Language Server identity. `internal/reloader/active_threads.go:58` and `:70` establish the two-hour conversation scan; `internal/reloader/continuation.go:95` establishes three-second pacing.
- Quota/UI: `internal/telemetry/loop.go:13`, `internal/ui/app.go:107`, `internal/ui/widgets.go:179`, and `internal/ui/quota_badge.go` establish the refresh interval, current account actions, cycle display, missing-data behavior and color thresholds. `internal/ui/themes.go:24` defines the three themes.
- Storage: `internal/store/account_store.go:108` writes the token envelope directly as JSON. `internal/store/sanitizer.go:115` requires the environment override directory to exist. The README distinguishes these files from Windows Credential Manager.
- Entrypoints/build: `cmd/switcher/main.go:27`, `cmd/switcher/app_cli.go`, `cmd/switcher/app_gui.go`, and `scripts/build_windows.ps1:135` / `:165` establish command flags, build-tag behavior and executable names. The CLI quota implementation leaves GPT-OSS as `N/A`; the README states that limitation.
- Desktop integration: `internal/ui/tray.go`, `internal/ui/window.go`, and `internal/ui/single_instance_windows.go` establish tray controls, window controls and single-instance behavior.

## Evidence checked

Passed during this task:

- Documented CGO-disabled engine/CLI test command: all nine packages passed.
- `CGO_ENABLED=1 go test -tags gui ./...`: all ten packages passed.
- `scripts/build_windows.ps1 -Mode all -SkipTests -OutDir .tmp\readme-go-verification-20261019`: CLI and GUI builds succeeded; embedded icon checks passed. Tests were run separately as listed above.
- Build script runtime status probe: the newly built CLI detected an active identity, running Language Server and credential presence. No credential content is recorded here.
- Newly built CLI `--help`: command names matched the documentation.
- PowerShell parser: build script parsed without errors.
- `git diff --check -- README.md`: passed.
- README content inspection: no former application launch commands, test commands, activation badges or encrypted-vault claim remain.
- Anvien repository inventory: this repository is not indexed; direct current-source inspection was used.

Not run or certified:

- Live account switching, session capture, GUI interaction and third-party policy/security audit. These are not required to verify this documentation edit; no claim of fresh end-to-end runtime acceptance is made.
- The sentence beginning “An toàn tuyệt đối” is owner-supplied wording, retained after the owner explicitly confirmed prior discussions with the relevant parties. This review verifies that the requested wording is present. Builds and tests do not independently establish that absolute guarantee or Google's authorization.

## Invariant closure and retained artifacts

- The technical sections reflect the inspected Go implementation and distinguish implemented behavior from unavailable data or automatic account failover.
- Same-invariant surfaces checked: both entrypoints, CLI flags, GUI actions, telemetry, credential storage, continuation and build instructions. No unverified technical documentation surface remains within this scope.
- Concurrent edits to `internal/ui/window.go` and `internal/ui/quota_badge.go` introduce wider cards and work toward quota reset countdowns after the validation commands ran. They were not modified or accepted by this task. The README does not prescribe these dimensions or advertise the in-progress countdowns. Build/test results above describe their execution state, not acceptance of those separate edits.
- Two successful verification binaries remain under `.tmp/readme-go-verification-20261019/`. Automatic execution review rejected both recursive cleanup and a narrower request targeting only those files and the empty directory, without a detailed reason. No alternate deletion mechanism was attempted. These ignored outputs are retained build evidence and do not affect README content.

The documentation edit is accepted for the scope above. This verdict is not an independent security certification or confirmation of third-party policy.
