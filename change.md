# 📋 Changelog - AS Auto Switcher

All notable changes, new features, performance improvements, and bug fixes across releases of **AS Auto Switcher** are documented in this file.

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
