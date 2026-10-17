# Architectural Specification Report: Phase 6A — Multi-Account Quota Telemetry Engine, Live Dashboard & Multi-Theme System

**Document Reference:** `reports/architect/2026-10-17-phase-6a-quota-telemetry-and-multi-theme-dashboard-architecture.md`  
**Date:** 2026-10-17  
**Author:** Senior System Architect  
**Authority:** Direct Human Owner Mandate via CEO Session (`01a14378-4a08-77c1-88d5-666ad8c3592d`)  
**Status:** APPROVED ARCHITECTURAL SPECIFICATION & UI TRACEABILITY BLUEPRINT  
**Target Milestone:** **Phase 6A (P6A)** — Bridging Phase P6 (Go Standalone Switcher) and Phase P7 (Autonomous Quota Failover Daemon)  

---

## 1. Executive Summary & Problem Formulation

### 1.1 The Fundamental Flaw of Blind Failover
During in-depth architectural evaluation of the Autonomous Quota Failover Daemon (Phase 7), the Human Owner identified a critical systemic deficiency in the baseline design:
> *The original Phase 7 failover mechanism was entirely **REACTIVE** and **BLIND ROUND-ROBIN**: upon intercepting an HTTP 429 quota exhaustion event, the switcher blindly selected the next account in the pool without knowing whether that account had remaining quota.*

This blind failover model suffers from three catastrophic failure modes in production:
1. **Cascading Flapping Storms (Bão chuyển đổi liên hoàn):** If the subsequent account in the queue was also exhausted previously (or possesses a smaller token limit), Antigravity immediately triggers another 429 on the very next query. This initiates an endless switching cascade, killing and restarting `language_server.exe` repeatedly and disrupting IDE stability.
2. **Speculative Cooldown Penalties (Đoán mò Cooldown):** The original design assigned arbitrary static cooldown timers (15m, 1h, 2h) without knowing when Google Cloud Code actually replenishes the account's quota.
3. **Multi-Model Quota Asymmetry:** In modern pair-programming workflows, Claude 3.5/3.7/4.6, Gemini 2.5/3, and GPT-OSS have separate, asymmetric quota pools. An account may have $0\%$ Gemini quota remaining while retaining $100\%$ Claude quota. Blind switching fails to leverage this granularity.

### 1.2 The Architectural Remedy: Phase 6A (P6A)
To solve this permanently, the Human Owner directed the formal establishment of **Phase 6A**:
- **Phase 6A** builds the **Multi-Account Quota Telemetry Engine**, a **Real-Time Fyne GUI Quota Dashboard**, a **60s Auto-Telemetry Loop**, and a **Multi-Theme System (Dark, Light, Warm)**.
- **Phase 6A** bridges Phase P6 (standalone binary) and Phase P7 (autonomous daemon), guaranteeing that when Phase P7 is activated, the failover controller possesses 100% deterministic, real-time knowledge of every account's exact quota headroom across all models.

---

## 2. Empirical Discovery & Reverse-Engineering Evidence

### 2.1 Discovery of Internal Google Cloud Code Quota Endpoints
Through binary reverse-engineering of Antigravity's production language server:
`C:\Users\TAM NGUYEN\AppData\Local\Programs\antigravity\resources\bin\language_server.exe`

Two authoritative internal RPC endpoints were uncovered on the Google Cloud Code backend:
1. `https://daily-cloudcode-pa.googleapis.com/v1internal:retrieveUserQuota`
2. `https://daily-cloudcode-pa.googleapis.com/v1internal:retrieveUserQuotaSummary`

### 2.2 Protocol Buffer & JSON Schema Definition
The Google endpoint returns fine-grained model bucket telemetry:
```protobuf
message RetrieveUserQuotaResponse {
  repeated BucketInfo buckets = 1;
  message BucketInfo {
    oneof remaining {
      int64 remaining_amount = 1;
      double remaining_fraction = 2; // e.g. 1.0 (100%), 0.2348 (23.5%), 0.0 (0%)
    }
    google.protobuf.Timestamp reset_time = 3; // Exact ISO 8601 UTC reset instant
    TokenType token_type = 4;                 // e.g. "WTUS"
    string model_id = 5;                      // e.g. "claude-sonnet-4-6", "gemini-2.5-pro"
  }
}
```

### 2.3 Live Parallel Benchmark on 7/7 Stored Accounts
A parallel worker pool executed token refreshes and `retrieveUserQuota` calls simultaneously across all 7 production accounts currently preserved in `accounts_store/`:

| # | Stored Account Email | Claude 4.6 (Opus/Sonnet) | Gemini 2.5 Pro / Flash | GPT-OSS | Google Reset Time (UTC) | Health Status |
|---|---|:---:|:---:|:---:|:---:|:---:|
| 1 | `daohien204ccyersin@gmail.com` | **100.0%** | **91.6%** | **100.0%** | `2026-10-17 11:29:41Z` | 🟢 Plentiful |
| 2 | `hiendao1760@gmail.com` (Active) | **100.0%** | **23.5%** | **100.0%** | `2026-10-17 07:19:14Z` | 🟡 Low Gemini |
| 3 | `hokinhdoanh.anvien.sales@gmail.com`| **100.0%** | **100.0%** | **100.0%** | `2026-10-17 12:13:34Z` | 🟢 Pristine |
| 4 | `ngau2hung@gmail.com` | **100.0%** | **100.0%** | **100.0%** | `2026-10-17 12:13:34Z` | 🟢 Pristine |
| 5 | `quantritaikhoanemail@gmail.com` | **100.0%** | **100.0%** | **100.0%** | `2026-10-17 12:13:34Z` | 🟢 Pristine |
| 6 | `skey.miennam@gmail.com` | **100.0%** | **100.0%** | **100.0%** | `2026-10-17 12:13:34Z` | 🟢 Pristine |
| 7 | `skey.vanphongmiennam@gmail.com` | **100.0%** | **100.0%** | **100.0%** | `2026-10-17 12:13:34Z` | 🟢 Pristine |

**Empirical Verification Metrics:**
- **Concurrency Duration:** **1.69 seconds** for all 7 accounts in parallel.
- **Success Rate:** **100% (7/7 accounts)** responded with HTTP 200 and complete bucket schemas.
- **Payload Footprint:** $\approx 10\text{ KB}$ network ingress total.
- **Conclusion:** Proves $100\%$ real-world feasibility.

---

## 3. Architecture Component Decomposition (Phase 6A)

Phase 6A introduces three cleanly decoupled subsystems:

```
+-----------------------------------------------------------------------------------+
|                            PHASE 6A ARCHITECTURE                                  |
+-----------------------------------------------------------------------------------+
|                                                                                   |
|  [ 1. BACKEND TELEMETRY ENGINE ] (internal/telemetry/)                            |
|    - Google OAuth2 Token Refresher (reusing internal/connector)                   |
|    - QuotaClient: HTTP REST client for v1internal:retrieveUserQuota               |
|    - Concurrent Worker Pool: parallelizes account queries with timeout context     |
|    - Two-Tier Cache: In-memory QuotaMatrix (sync.RWMutex) + Disk Snapshot          |
|    - 60-Second Auto-Telemetry Ticker Loop with Single-Flight Concurrency Gate     |
|                                                                                   |
|  [ 2. FYNE GUI DASHBOARD & MULTI-THEME ] (internal/ui/)                           |
|    - Toolbar Header: [📊 Làm mới Quota Tất cả], Auto-Refresh Indicator            |
|    - Account List Cards: Mini Quota Badges (Claude/Gemini/GPT) + Color Thresholds  |
|    - Account Action: Individual [🔄] Refresh Button per row                       |
|    - Theme Switcher Engine: 3 Modes (Dark Mode, Light Mode, Warm Mode)             |
|    - Async GUI Dispatcher: fyne.CurrentApp().QueueEvent non-blocking rendering     |
|                                                                                   |
|  [ 3. CLI INTEGRATION & BENCHMARKS ] (cmd/switcher/)                             |
|    - CLI Flag: `antigravity-switch.exe -quota` for tabular terminal output        |
|    - Memory & Latency Verification: RAM <= 20MB, Cold Startup <= 50ms             |
+-----------------------------------------------------------------------------------+
```

---

## 4. Detailed Specification: 60-Second Auto-Telemetry Loop

### 4.1 Requirement & Invariant
Per the Human Owner's explicit directive:
> *When running in GUI mode, the switcher MUST automatically re-measure and refresh the quota of all accounts in the pool every **60 seconds**.*

### 4.2 Concurrency & Safety Model
1. **Single-Flight Concurrency Gate:**
   - Managed via `atomic.Bool` (`isRefreshing`).
   - If a previous polling cycle is delayed by network lag, the subsequent 60s ticker event is **dropped immediately** (`CompareAndSwap(false, true)`). No request stacking.
2. **Manual Sync Reset:**
   - If the user clicks `[📊 Làm mới Quota Tất cả]`, the 60s ticker is reset immediately, preventing redundant double-calls.
3. **Resource Budget:**
   - Polling frequency: 7 requests / 60 seconds ($\approx 0.11\text{ req/sec}$).
   - Google rate limit consumption: $< 0.1\%$ of standard quota allowances.
   - Background CPU utilization: $\le 0.02\%$.

---

## 5. Detailed Specification: Multi-Theme System (Dark, Light, Warm)

### 5.1 Motivation & User Requirement
The existing Fyne GUI implements only a default dark theme. To deliver a complete, comfortable, and professional developer experience across varying lighting conditions, Phase 6A mandates three native themes:

| Theme Name | Target Environment | Palette Characteristics | Key Color Tokens |
|---|---|---|---|
| **🌙 Dark Mode** | Night / Dim lighting (Default) | Deep charcoal background, high-contrast crisp text, subtle borders | Background: `#1e1e24`<br>Card: `#2a2a32`<br>Text: `#f0f0f5`<br>Primary Accent: `#3b82f6` |
| **☀️ Light Mode** | Bright daylight / Office | Pure paper background, crisp dark slate text, subtle shadows | Background: `#f8fafc`<br>Card: `#ffffff`<br>Text: `#0f172a`<br>Primary Accent: `#2563eb` |
| **☕ Warm Mode** | Late-night reading / Blue-light reduction | Warm sepia/amber tint, gentle contrast, zero harsh blue light | Background: `#fbf0d9` (Warm Light) or `#241e17` (Warm Dark)<br>Card: `#eddcc1` / `#332a20`<br>Text: `#433422` / `#e8d8be`<br>Primary Accent: `#d97706` (Amber) |

### 5.2 Implementation via Fyne Theme Interface
Implement custom types satisfying `fyne.Theme`:
```go
type DarkTheme struct{}
type LightTheme struct{}
type WarmTheme struct{}
```
- Overriding `Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color`.
- Theme selector control: A 3-segment toggle or dropdown in the Header Toolbar:
  `[🎨 Giao diện: Tối | Sáng | Ấm]`
- Persist user preference to `accounts_store/.ui_preferences.json`.

---

## 6. Strict UI Design Specification & Traceability Acceptance Matrix

To ensure that downstream Coder implementations do not deviate from the approved design, the following **UI Traceability Matrix** is established as canonical acceptance law.

### 6.1 Visual Layout Architecture

```
+-----------------------------------------------------------------------------------+
|  Antigravity Multi-Account Switcher (v1.1)            [🎨 Tối | Sáng | Ấm] [ - ][ X ]|
+-----------------------------------------------------------------------------------+
|  👤 ĐANG NẠP RAM: hiendao1760@gmail.com                           [🟢 Hoạt động]  |
|  Hạn ngạch hiện tại: 🟣 Claude: 100% | 🔵 Gemini: 23.5% | ⚪ GPT: 100%             |
+-----------------------------------------------------------------------------------+
|  Thao tác hệ thống:                                                               |
|  [🔄 Đồng bộ Chrome]   [💾 Lưu phiên này]   [📊 Làm mới Quota Tất cả]              |
|                                             (Tự động: 60s | Cập nhật: 12:45:00)   |
+-----------------------------------------------------------------------------------+
|  DANH SÁCH TÀI KHOẢN TRONG KHO (7 tài khoản):                                     |
|                                                                                   |
|  ┌─────────────────────────────────────────────────────────────────────────────┐  |
|  │ 🟢 daohien204ccyersin@gmail.com                              [🔄] [⚡ Chuyển]│  |
|  │    Claude: [██████████] 100%   Gemini: [█████████░] 91.6%   GPT: 100%       │  |
|  ├─────────────────────────────────────────────────────────────────────────────┤  |
|  │ 🟢 hokinhdoanh.anvien.sales@gmail.com                        [🔄] [⚡ Chuyển]│  |
|  │    Claude: [██████████] 100%   Gemini: [██████████] 100%    GPT: 100%       │  |
|  ├─────────────────────────────────────────────────────────────────────────────┤  |
|  │ 🟡 hiendao1760@gmail.com (Active)                            [🔄] [Đang dùng]│  |
|  │    Claude: [██████████] 100%   Gemini: [██░░░░░░░░] 23.5%   GPT: 100%       │  |
|  │    ⚠️ Gemini sắp cạn (Google sẽ hồi phục lúc: 14:19 UTC - còn 1h 34p)       │  |
|  └─────────────────────────────────────────────────────────────────────────────┘  |
+-----------------------------------------------------------------------------------+
|  Trạng thái Antigravity: PID: 12450 | Port: 58921 | Đang kết nối mượt mà          |
+-----------------------------------------------------------------------------------+
```

### 6.2 UI Traceability Acceptance Checklist (Bảng Kiểm Tra Truy Vết UI)

Every item in this matrix MUST be verified by Supervisor and QA before Phase 6A can be marked complete.

| UI Element ID | Element Description | Location | Visual & Behavioral Acceptance Criteria | Severity on Violation |
|---|---|---|---|---|
| **UI-TR-01** | Global Quota Refresh Button | Header Toolbar | Button labeled `[📊 Làm mới Quota Tất cả]`. Clicking triggers concurrent refresh across all accounts. Displays visual spinning/loading state during fetch. | **BLOCKER** |
| **UI-TR-02** | Row-Level Single Refresh Button | Each Account Row | Icon button `[🔄]` placed immediately adjacent to the account email. Clicking triggers refresh ONLY for that specific account. | **BLOCKER** |
| **UI-TR-03** | Auto-Refresh 60s Indicator | Header Toolbar / Status | Label displaying `Tự động: 60s` and last updated timestamp (`HH:mm:ss`). Pulses softly upon auto-refresh completion. | **BLOCKER** |
| **UI-TR-04** | Mini Quota Progress Badges | Each Account Row | Displays percentage badges for Claude, Gemini, and GPT. Three-tier color coding:<br>• Green (`#22c55e`): $> 50\%$<br>• Amber (`#f59e0b`): $15\% - 50\%$<br>• Red (`#ef4444`): $< 15\%$ | **BLOCKER** |
| **UI-TR-05** | Quota Recovery Countdown Text | Account Row with Quota $<15\%$ | Text label beneath progress bar showing exact local or UTC recovery time: `Google sẽ hồi phục lúc: HH:mm (còn Xh Yp)`. | **HIGH** |
| **UI-TR-06** | Multi-Theme Switcher Control | Window Titlebar / Header | Segmented button or selector for `Tối (Dark)`, `Sáng (Light)`, and `Ấm (Warm)`. Switching instantly alters background, cards, text, and progress bars without restarting app. | **BLOCKER** |
| **UI-TR-07** | Warm Mode Color Palette | Theme Engine | Warm Mode must eliminate harsh blue light: warm sepia/amber tints, reduced contrast, soothing background. | **HIGH** |
| **UI-TR-08** | Non-Blocking UI Invariant | Entire UI Lifecycle | Zero UI freezing during network calls. GUI must maintain responsive 60 FPS scrolling and interaction while telemetry is actively polling in background goroutines. | **BLOCKER** |

---

## 7. Execution Slices for Phase 6A

The execution of Phase 6A is broken into three sequential, independently verifiable slices:

### **Slice P6A-1: Quota Telemetry Engine & 60s Loop (`internal/telemetry/`)**
- Implement `internal/telemetry/client.go`: HTTP client for Google Cloud Code `v1internal:retrieveUserQuota`.
- Implement `internal/telemetry/pool.go`: Parallel worker pool for $N$ accounts with `context.WithTimeout`.
- Implement `internal/telemetry/loop.go`: `time.Ticker(60 * time.Second)` with single-flight concurrency gate.
- Implement `internal/telemetry/cache.go`: Two-tier cache (RAM QuotaMatrix + `accounts_store/.quota_cache.json`).
- Unit test suite with mock Google server achieving $> 90\%$ test coverage.

### **Slice P6A-2: UI Quota Dashboard & Multi-Theme System (`internal/ui/`)**
- Implement `internal/ui/themes.go`: Custom `DarkTheme`, `LightTheme`, and `WarmTheme` conforming to `fyne.Theme`.
- Implement `internal/ui/quota_badge.go`: Mini progress bars with 3-tier color coding.
- Update `internal/ui/window.go`: Add toolbar buttons (`[📊 Làm mới Quota Tất cả]`, theme selector), add row-level `[🔄]` buttons.
- Connect asynchronous event loop for non-blocking UI rendering.

### **Slice P6A-3: CLI Integration, Traceability QA & Final Benchmarking (`cmd/switcher/`)**
- Implement CLI flag `antigravity-switch.exe -quota` for tabular terminal output.
- Run UI Traceability Checklist (`UI-TR-01` through `UI-TR-08`).
- Benchmark memory footprint ($\le 20\text{ MB}$ RAM) and startup latency ($\le 50\text{ ms}$).

---

## 8. Strategic Handoff & Directive to Planner

This specification establishes the uncompromised architecture for Phase 6A.
- **Handoff Target:** CEO Session (`01a14378-4a08-77c1-88d5-666ad8c3592d`).
- **Required CEO Action:** Issue directive to **Planner Lane** to incorporate **Phase 6A** into `docs/plans/` with the exact execution slices and UI Traceability Matrix specified herein.
- **Subsequent Action:** Dispatch Coder and Supervisor for Slice P6A-1.
