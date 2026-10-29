# ADR-001: Elevate Global Preference Controls (Theme & Privacy Masking) to Native Window Titlebar

| Metadata | Details |
| :--- | :--- |
| **Status** | **Approved** |
| **Date** | 2026-10-29 |
| **Decision Maker** | System Architect (in direct consensus with Human Owner) |
| **Target System** | Desktop Client Window Header & Primary Engine View Toolbar |
| **Related SPEC** | `Docs/SPEC/SPEC-ARCH-GLOBAL-HEADER-CONTROLS.md` |

---

## 1. Context & Problem Statement

In the desktop application, the **Theme Selector** (Dark / Light / Warm) and the **Privacy Masking Switch** (Email redaction for sensitive accounts) were originally housed within the secondary toolbar of the primary engine tab.

As the application expanded to support multi-tab operations (introducing the secondary engine view "Codex"), several architectural conflicts emerged:
1. **Misplaced Responsibility:** Theme and Account Privacy are global preferences that apply across the entire application shell, all tabs, dialog modals, and status bars. Confining them to the primary engine toolbar meant they disappeared whenever the user navigated to alternate views.
2. **Cognitive Overhead:** The primary engine toolbar was overloaded with operational action buttons, engine auto-switch toggles, privacy switches, and theme dropdowns all packed into one row.
3. **Multi-Engine Consistency:** The secondary engine tab (Codex) lacked immediate user controls for toggling themes or masking sensitive data, despite sharing the same core visual theme and persistent configuration store.

---

## 2. Decision

We decide to:
1. **Relocate both the Theme Selector and the Privacy Masking Control** from the primary engine tab toolbar to the **Top-Level Frameless Window Header (Titlebar)**, positioned immediately to the left of the Locale Selector (`VN`/`EN`).
2. **Standardize the Right Header Control Cluster** in the following canonical sequence:
   $$\mathbf{[Mask\ Accounts\ Group]} \quad\Big|\quad \mathbf{[Theme\ Selector]} \quad\Big|\quad \mathbf{[Locale\ Selector]} \quad\Big|\quad \mathbf{[Window\ Controls]}$$
   separated by subtle $1\text{dp} \times 14\text{dp}$ vertical delimiters.
3. **Adopt Full Localized Text Labeling for Mask Accounts** (`[Ẩn tài khoản: ○]` / `[Mask Accounts: ○]`) coupled with the compact capsule toggle switch to maximize discoverability and clarity for end-users.
4. **Strip the Primary Engine Toolbar** of global selectors, retaining solely the engine-specific Automated Switching toggle in its right cluster.
5. **Enforce Drag Area Budget Protection** by requiring title text ellipsis truncation to prevent right-cluster overflow or drag area starvation on compact window widths ($640\text{dp}$).

---

## 3. Consequences

### Positive
- **Global Availability:** Users can adjust application theme and privacy masking from any screen or tab without needing to switch back to the primary engine view.
- **Clean Separation of Concerns:** Titlebar manages global window framing and global user preferences; tab toolbars strictly manage tab-specific operational actions.
- **De-cluttered Work Area:** The primary engine toolbar is significantly cleaner, allowing immediate focus on account operations and engine automation status.
- **Unified Codex View Compatibility:** The secondary engine tab (and any future engine views) inherits seamless theme and privacy controls out-of-the-box.
- **Zero-Jank In-Place Updates:** State mutations trigger in-place Canvas refreshes across all active and background views without window reconstruction.

### Negative & Technical Costs
- **Titlebar Real Estate Consumption:** Adding the full-text mask accounts label, theme selector, and delimiters expands the right-aligned control cluster to $\approx 357\text{dp}$.
- **Constrained Drag Budget at Minimum Width:** At the absolute minimum supported width ($640\text{dp}$), the remaining draggable titlebar area contracts to $\approx 48\text{dp}$.
- **Refactoring Scope for Automated Tests:** Existing UI acceptance and layout tests asserting toolbar child counts and element locations require updating to reflect the new canonical header structure.

### Mitigations
- Enforce dynamic text truncation with ellipsis on the window title text so long titles or high-DPI scaling cannot squeeze out the drag area or push controls off-screen.
- Adopt a vertical centerline alignment layout algorithm for the right header cluster to guarantee sub-pixel alignment across elements of varying physical heights ($12\text{dp}$ to $26\text{dp}$).
- Maintain backward-compatible accessor contracts on the core application structure to minimize disruptions to downstream testing harnesses.

---

## 4. Alternatives Considered

### Alternative A: Keep Preferences in Primary Engine Toolbar and Replicate in Secondary Engine Toolbar
- **Drawbacks:** Violates DRY (Don't Repeat Yourself) principle; increases codebase duplication across views; causes confusion when switching between tabs if state updates lag or desynchronize.
- **Rejection Reason:** Poor scalability as more engine tabs are integrated.

### Alternative B: Compact Titlebar Mask Control (Icon Only, e.g., `[ 👁️ ○ ]`)
- **Benefits:** Saves $\approx 60\text{dp}$ of horizontal width on the titlebar, leaving a generous drag area even at $640\text{dp}$ width.
- **Drawbacks:** Requires reliance on hover tooltips for user discovery; less immediately explicit for new users.
- **Decision:** Evaluated during Mode 0 discussion with Human Owner; Owner explicitly decided in favor of the full explicit text label (`[Ẩn tài khoản: ○]`) for superior clarity.

### Alternative C: Hide Theme and Masking inside a Settings Menu / Modal Dialog
- **Drawbacks:** Adds friction to everyday quick tasks (switching themes or masking screens before sharing/recording); conceals critical privacy state from instant view.
- **Rejection Reason:** High friction and degraded user experience.
