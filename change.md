# 📋 Changelog - AS Auto Switcher

All notable changes, new features, performance improvements, and bug fixes across releases of **AS Auto Switcher** are documented in this file.

---

## [Version 1.0.3.3] - 2026-10-31

### 🔄 Campaign 1: Reloader Hierarchical Thread Reactivation & 5m Window (`dee5506`)
- **Hierarchical Thread Reactivation:**
  - Upgraded IDE chat session resurrection logic to hierarchically reactivate conversation threads within a 5-minute active window.
  - Implemented thread state inspection with `killed` filter to bypass discarded and dead conversation threads, preventing zombie session reloads.
  - Enhanced session reloader resilience during automated account failover, preserving active developer context without chat corruption or state desynchronization.

### 📦 Campaign 2: Core Module Rename & Engine Isolation to `as-auto-switcher` (`aaf3782`)
- **Go Module Rename to `as-auto-switcher`:**
  - Transitioned the root Go module from legacy `antigravity-auto-switcher` to `as-auto-switcher` across all imports, build scripts, tests, and configurations.
- **Engine Subsystem Boundary Isolation:**
  - Architected strict separation of multi-AI engines under `internal/engines/antigravity` and `internal/engines/codex`.
  - Decoupled shared core components (`internal/core/` for preferences, versioning, credentials, and bilingual i18n) from platform-specific engine drivers, preventing cross-engine dependency leaks and circular imports.
  - Standardized diagnostic CLI flags (`--version`) and single-instance Windows mutex guards (`Global\AS_Auto_Switcher_SingleInstance_Mutex`).

### 🤖 Campaign 3: Codex Backend Architecture 6-Phase (`5f04ab6`..`f385096`)
- **Phase 1: Dual-Store Partitioning & Safe Migration (`5f04ab6`):**
  - Partitioned `accounts_store` into distinct directories: `accounts_store/antigravity` and `accounts_store/codex`.
  - Built atomic migration logic ensuring non-destructive transition of existing account credentials upon application launch.
- **Phase 2: Codex Models & Account Store Implementation (`0604445`):**
  - Implemented strong type-safe data models for OpenAI Codex accounts, tokens, session metadata, and quota states.
  - Passed strict G-CODEX pre-check invariants for storage persistence.
- **Phase 3: Runtime Adapter, JSON-RPC Client & Browser Detection (`3aa370a`):**
  - Implemented Codex runtime adapter communicating via robust JSON-RPC protocol.
  - Built proactive local browser and desktop process detection engine for Codex environment inspection.
- **Phase 4: OAuth Lifecycle Manager & Account Manager Coordination (`7362a26`):**
  - Built Codex OAuth authentication manager managing PKCE flows, token exchange, and refresh lifecycles.
  - Implemented multi-attempt concurrency coordination with mutex-protected state transitions.
- **Phase 5: Codex Add Account Modal & Card View Integration (`ccb6124`):**
  - Designed and delivered Fyne-based Codex Add Account modal dialog and interactive account card views.
  - Fully bound bilingual localization (Vietnamese and English) across all Codex UI elements.
- **Phase 6: Desktop Switching Boundary & Bootstrap Pipeline (`e5870f8`):**
  - Completed end-to-end desktop switching boundary and application bootstrap pipeline for seamless Codex integration.

### 🛡️ Campaign 4: 22-Trap Code Review Fixes for Codex Desktop Integration (`86703d2`)
- **Comprehensive Reliability & Security Hardening (22 Anti-Traps Cleared):**
  - *Trap 1-3 (Process & Target Identification):* Eradicated imprecise PID scanning; introduced reliable Windows Desktop process targeting, launcher contract verification, and JWT account ID validation.
  - *Trap 4-6 (Credential & State Isolation):* Resolved home/backend path ambiguities, added busy state and concurrency checks, and eliminated unsafe credential read-fallback assumptions.
  - *Trap 7-9 (Atomic Durability & Rollback):* Enforced journal updates before stage transitions, preserved original tokens during rollbacks, and eliminated dangerous file-deletion fallbacks.
  - *Trap 10-12 (Crash Recovery & Re-auth Generation):* Protected crash recovery against clobbering external modifications, bounded re-auth generations, and fixed no-op email-only comparisons.
  - *Trap 13-16 (Coordination & Cancellation Lifecycle):* Unified coordinating mutexes across OAuth, switch, and delete operations; resolved attempt cancellation race conditions and cleanly shut down child processes.
  - *Trap 17-19 (UI & Store Resilience):* Added automatic cleanup for expired failed saves, preserved manual switch capability when disabled, and bound Ready/Reauth status strictly to live runtime tokens.
  - *Trap 20-22 (Transport, Migration & Hygiene):* Hardened RPC transport channels against closed-channel sends and timeout stretching; guarded filesystem junction/reparse points; sanitized CLI arguments, ACLs, and logging redacting sensitive tokens.

### 🔀 Campaign 5: Codex Desktop Manual Switch 4-Phase (`06c4f9d`..`a52c576`)
- **Phase 1: Windows ChatGPT Desktop Lifecycle Adapter (`06c4f9d`):**
  - Created native Windows process lifecycle management adapter for OpenAI ChatGPT / Codex Desktop application.
- **Phase 2: Token Refresher Engine & Auth Validation (`e181dd2`):**
  - Implemented standalone token refresher engine verifying OAuth tokens and refreshing expired credentials safely prior to switching.
- **Phase 3: Session Reloader with Network Retries & Settling Window (`ca5d4a8`):**
  - Built resilient session reloader incorporating exponential backoff, network retries, and post-launch settling windows.
- **Phase 4: Fyne UI Async Codex Switch & Button Binding (`95bccb4`):**
  - Bound the UI manual switch buttons on Codex cards to asynchronous background execution workers, ensuring zero UI jank or freeze.

### 📊 Campaign 6: Dynamic Quota Matrix & Argon Expansion 4-Phase (`f7e4da7`..`c4bbea2`)
- **Phase 1: Dynamic Quota Data Foundation & Discovery Automation (`f7e4da7`):**
  - Developed dynamic quota discovery engine inspecting Claude 3.5 Sonnet, Claude 3.7 Sonnet, Claude 3.7 Sonnet Thought, and Gemini Pro models automatically.
  - Neutralized hardcoded model lists in favor of adaptive telemetry extraction.
- **Phase 2: Quota Matrix Layout Math & Ternary Display (`104dea4`):**
  - Implemented mathematical layout calculations for 2-line model quota matrix cells with compact vertical footprint.
  - Introduced ternary status rendering (`Active`, `Depleted`, `N/A`) with color-coded status badges.
- **Phase 3: Dynamic Card & Grid Expansion (`98b15e7`):**
  - Expanded account card containers dynamically based on active models, rendering flexible grid layouts.
  - Robust fallback handling ensuring seamless rendering across variable display resolutions.
- **Phase 4: Full Test Suite Sweep, Race Sweep & Windows Build (`85c2b43`, `c4bbea2`):**
  - Validated complete test suite under `go test -race` with 0 data races, achieving 100% green tests and successful Windows PE binary compilation.

### 📁 Campaign 7: Discovered Accounts Reorganization & Migration (`52c3a15`..`83368a8`)
- **Phase 1: Partitioned Discovered Store (`52c3a15`):**
  - Migrated legacy root `discovered_accounts/` store into engine-scoped directory `accounts_store/antigravity/discovered_accounts/`.
- **Phase 2: Manager Entrypoint & Startup Migration (`e5f0dd9`):**
  - Integrated automatic pre-flight migration during `account_manager` startup, safely relocating candidate account JSON files without data loss.
- **Phase 3: UI Presentation & Acceptance Test Synchronization (`947f6bd`):**
  - Synchronized modal presentation layers and acceptance test suites to read candidate accounts from the partitioned store path.
- **Phase 4: Full Build, Regression Sweep & Clean-up (`e0f9cb6`..`83368a8`):**
  - Eradicated legacy root directory, verified 100% data preservation across all 6 candidate accounts, and swept all repository tests green.

### 🏷️ SSOT Version Bump & System Synchronization
- Bumped application Single Source of Truth (SSOT) from `1.0.3.2` to `1.0.3.3` at `internal/core/version/version.go`.
- Synchronized Windows PE resource metadata (`cmd/switcher/app.rc`) to `1,0,3,3`.
- Synchronized headers across all documentation files (`README.md`, `README_VN.md`, `README_EN.md`).
- Synchronized version acceptance test assertions across `internal/core/version/version_test.go`, `cmd/switcher/version_test.go`, and `internal/ui/footer_bar_acceptance_test.go`.
- Recompiled official release Windows PE binary (`bin/AS_Auto_Switcher.exe`) with GUI subsystem and embedded 32x32 icon.

---

## [Version 1.0.3.2] - 2026-10-30

### 🪟 System Tray Menu Optimization & Graceful Shutdown Parity
- **Elimination of Duplicate Quit Action (`exitItem.IsQuit = true`):**
  - Explicitly flagged `exitItem.IsQuit = true` on the System Tray menu exit item, neutralizing Fyne desktop driver's implicit `addMissingQuitForMenu()` fallback.
  - Completely eradicates the confusing duplicate exit actions ("Exit" & "Quit", or "Thoát" & "Quit") previously visible in the Windows taskbar tray context menu.
  - Preserves pure bilingual localization integrity (`i18n.Current().TrayExit`), ensuring the tray menu renders cleanly with exactly 4 items (`Show Switcher`, `Refresh Chrome`, `Separator`, `Exit` / `Thoát`).
- **Standardized Graceful Application Shutdown:**
  - Standardized the tray exit handler to execute `a.Close()` prior to invoking `a.app.Quit()`.
  - Ensures background telemetry loops, ticker routines, and window contexts are cleanly terminated and persistent preferences flushed to disk before UI process termination, achieving 100% graceful shutdown parity with the main window close intercept.

### 🚀 Instant Startup Quota Synchronization (t = 0) & Cold Start Zero-Delay
- **Immediate Quota Poll at Application Startup (t = 0):**
  - Enhanced `AutoTelemetryLoop` to trigger an immediate background quota poll at $t = 0$ upon initialization prior to commencing the regular 60-second polling loop.
  - Completely eliminates the 60-second cold-start quota blindness window, pulling fresh real-time quota telemetry from Google within milliseconds of launching the application.
  - Operates unconditionally at startup regardless of whether the autonomous AutoSwitch toggle is enabled or disabled, ensuring UI cards immediately reflect accurate quota metrics.
  - Automatically triggers account failover at $t = 0$ if the currently active account is exhausted, seamlessly rotating to a healthy backup account without requiring manual user refresh.
  - Protected by a single-flight atomic concurrency gate (`l.isRefreshing.CompareAndSwap`), eliminating duplicate requests if a manual refresh is triggered simultaneously.

### 🛡️ Concurrency Hardening & Fast Block Remediations
- **Thread-Safe Telemetry Cache Access (`QuotaCache` Data Race Resolution):**
  - Resolved a concurrent read/write data race on `c.filePath` flagged by Go ThreadSanitizer during high-concurrency telemetry operations.
  - Synchronized disk path access through `c.FilePath()` under read lock (`c.mu.RLock()`), and guarded path updates inside `SaveToDisk` and `LoadFromDisk` under write lock (`c.mu.Lock()`).
  - Achieved 100% data-race-free execution across all 38/38 telemetry test suites (`go test -race`).
- **UI Serialization & Atomic Widget Construction:**
  - Introduced dedicated `uiMu sync.Mutex` on `App` struct to serialize concurrent calls to `updateUI`, preventing re-entrant collisions between background telemetry callbacks, session syncs, and user interactions.
  - Adopted atomic slice construction (`newCards`, `newObjects`) before assigning to `a.accountCards` and `a.accountListContainer.Objects`, completely eliminating duplicate widget cards and preventing race conditions on the Fyne canvas object tree.

### ⚡ AutoSwitch Algorithm Optimization & Anti-Thrashing Guard
- **Strict Inequality Weekly Quota Threshold (> 2%):**
  - Raised the weekly quota candidate selection threshold in Stage 2 (Candidate Screening) from 1% to strictly **> 2%** (`ThresholdWeeklyDepleted = 0.02`), completely eliminating accounts at or below 2% (`<= 2%`, e.g., 0%, 0.5%, 1.0%, 1.5%, 2.0%).
  - Eradicates rapid oscillating account switching loops (**Thrashing / Flapping Loop**) that previously occurred when switching to marginal backup accounts on the verge of weekly exhaustion.
- **Platform Quota Invariant Standardization (Mandatory "AND" Conjunction):**
  - Standardized eligible candidate health criteria to strictly enforce the **"AND"** conjunction across all tracked quotas:
    - Accounts with 5-hour rolling cycle: **5h > 2% AND Weekly > 2%** (`ApplicableValid`).
    - Accounts with Weekly-only rolling cycle: **Weekly > 2%** (`NotApplicable`).
- **Strict 3-Stage Sequential Pipeline Invariant Preservation:**
  - Preserved the unidirectional 3-stage execution pipeline without coupling or side effects:
    - **Stage 1 (Detection):** Preserves 100% existing trigger mechanisms (proactive telemetry evaluation, 429 quota exhaustion, and GUI toggle OFF).
    - **Stage 2 (Screening & Hygiene):** Applies 6 candidate screening filters to produce a clean `eligible` slice strictly satisfying health invariants.
    - **Stage 3 (Burn-down Sorter & Preflight):** Prioritizes candidates with lowest weekly fraction (`w1 < w2`), validates quota freshness via preflight probe, and commits atomic account switch.
- **Architectural Trace Comments:**
  - Embedded explicit 3-stage boundary trace comments across `model_quota.go`, `sorter.go`, and `controller.go` for long-term codebase maintainability and architectural clarity.

### 🏷️ SSOT Version Bump & System Synchronization
- Bumped application Single Source of Truth (SSOT) from `1.0.3.1` to `1.0.3.2` at `internal/core/version/version.go`.
- Synchronized Windows PE resource metadata (`cmd/switcher/app.rc`) to `1,0,3,2`.
- Synchronized headers across all documentation files (`README.md`, `README_VN.md`, `README_EN.md`).
- Updated footer bar version formatting acceptance test assertion (`internal/ui/footer_bar_acceptance_test.go`).

---

## [Version 1.0.3.1] - 2026-10-29

### ✨ New Features & UX Enhancements
- **Multi-Platform Sidebar Navigation:**
  - Integrated compact left sidebar (136px) enabling seamless switching between AI tool suites: **Antigravity Tab** (`Tab 0`) and **Codex Tab** (`Tab 1`).
  - Standardized application window width to `1100px` (with a minimum width of `640px`) for optimal readability and spacious information layout.
  - Zero-jank in-place tab switching preserving state across functional modules.
- **Codex UI & Skeleton Architecture Integration:**
  - Integrated Codex tab view (`codexView`) with branded emblem (`codex_logo.png`), contextual status badge cards, and dynamic footer components.
  - Scaffolded 53 Go package skeleton files for the Codex backend mirroring Antigravity's architectural layout.
- **Active Conversation Threads Window Expansion:**
  - Upgraded active conversation scan and preservation window from 5 minutes to 1 hour, and subsequently expanded to 2 hours (`2h window`).
  - Preserves developer chat context and active IDE sessions seamlessly during automated background account switching.
- **Add Account Modal Standardization:**
  - Decoupled candidate badge tags (`[OAuth Ready]`, `[Added]`, `[Needs OAuth]`) from the primary account listing.
  - Completed 100% bilingual localization (Vietnamese & English) eliminating all hardcoded UI strings.

### 🏛️ Architecture & Decision Records (ADR)
- **Global Header Preference Controls Specification:**
  - Authored system architecture specification `SPEC-ARCH-GLOBAL-HEADER-CONTROLS.md` and Architectural Decision Record `ADR-001-MIGRATE-PREFERENCE-CONTROLS-TO-TITLEBAR.md`.
  - Designed migration of `Theme Select` and `Mask Accounts Toggle` from Antigravity Row 2 Toolbar to Row 1 Header (to the left of i18n Dropdown) as Global Preferences shared across all application tabs (Antigravity & Codex).
  - Defined window drag area budget (`dragArea`), standardized `1x14px BorderSubtle` dividers, and enforced vertical centerline alignment invariant ($y = (H - h)/2$).
- **Core Isolation & Backend Directory Restructuring (Phase 1B):**
  - Fully isolated `internal/core` (i18n, preferences, version, wincred) from GUI presentation layers, eliminating circular dependencies and reinforcing modularity.

---

## [Version 1.0.3.0] - 2026-10-24

### ✨ New Features & UX Enhancements
- **Automatic Tier Detection (Tier Badge):** Replaced static `"Quota"` label in the top-left cell of the quota matrix with live, auto-detected subscription tier badges (`PRO`, `ULTRA`, `FREE`, `BIZ`, `--`).
- **Visual Hierarchy & Color Coding:**
  - `ULTRA`: Vibrant Amber (`#f59e0b`) highlighting top-tier Google AI Ultra accounts.
  - `PRO`: Primary Sky Blue (`#38bdf8`) for primary Google One AI Pro / Premium daily-driver accounts.
  - `FREE`: Muted Gray (`#888899`) for standard personal accounts.
  - `BIZ`: Emerald Green (`#10b981`) for Google Workspace / Enterprise accounts.
  - `--`: Subtle Muted Gray for unauthenticated or initializing accounts.

### 🛠️ Technical Improvements & Architecture
- **Schema Drift Remediation (`GetUserStatus`):** Fixed type-assertion crashes caused by Antigravity 2.x upgrading `planStatus` to nested JSON maps (`planInfo.planName`, `planInfo.teamsTier`), maintaining backwards-compatible support for both legacy flat strings and nested maps.
- **Dual-Source Telemetry:**
  - Integrated Google Cloud Code endpoint `/v1internal:loadCodeAssist` to detect precise subscription tiers across the entire account inventory (including inactive accounts).
  - Connected local RPC `/GetUserStatus` for instantaneous active IDE telemetry.
- **Zero-Flicker In-Place Updates:** Added canvas-level `UpdateTier` on Fyne widgets, preserving fixed column width (`col1W = 48px`) without UI jitter.
- **Changelog Tracking Initialization:** Added root `change.md` to track release history.

---

## [Version 1.0.2.0] - 2026-10-23

### ✨ New Features & Documentation
- **Comprehensive Multilingual Documentation:** Added complete bilingual documentation including `README_VN.md` (Vietnamese) and `README_EN.md` (English) with seamless cross-navigation.
- **SSOT Version Auto-Synchronization:**
  - Established `internal/core/version/version.go` as the Single Source of Truth (SSOT).
  - Automated version synchronization across `cmd/switcher/app.rc` (Win32 resource metadata) and headers of all `README.md` files.
  - Supported `-SyncOnly` flag in `scripts/build_windows.ps1` for sub-200ms version synchronization.
- **Contract Test Suite:** Added `TestVersion_ReadmeSync` unit test to automatically detect and prevent version drift across documentation and binaries.

---

## [Version 1.0.1.0] - 2026-10-21

### ⚡ Performance Optimization (Startup & Memory)
- **Cold Startup Boost:** Cut cold application startup time to sub-500ms by streamlining configuration loading and account store initialization.
- **Memory Footprint Reduction:** Reduced idle background RAM consumption by over 40% via automated cleanup of unused canvas and graphic resources.
- **Win32 Discovery & Singleflight:** Utilized low-level Windows APIs for rapid and reliable detection of Antigravity Language Server processes, eliminating duplicate telemetry calls.
- **Double-Check Active Protection:** Implemented two-layer safety guard preventing accidental deletion or switching of actively running accounts.

---

## [Version 1.0.0.0] - 2026-10-18

### 🚀 Initial Release
- **Standalone Zero-Console Architecture:** Packaged entire application into a single executable `bin/antigravity-switch.exe` without popping up a console terminal window on launch.
- **Modern Desktop GUI (Fyne):** Visual card-based account manager displaying real-time states `[✔ Active]`, `[Switch]`, and `[Needs OAuth]`.
- **Dual-Cycle Quota Monitoring Matrix:**
  - Concurrent tracking of both 5-hour (5h) and Weekly rolling cycles for Gemini and Claude.
  - Mini ProgressBars dynamically color-coded by remaining capacity (Green > 50%, Yellow 15-50%, Red < 15%).
- **Autonomous AutoSwitch Daemon:**
  - Real-time detection of quota exhaustion.
  - Intelligent **Burn-down Sorter** algorithm prioritizing optimal backup accounts.
  - Exponential retry-backoff preventing congestion and ensuring continuous developer productivity.
- **System Utilities:** Always-on-Top window pinning, System Tray minimization, and 3 built-in themes (Dark, Light, Warm).
