# SPEC-ARCH-GLOBAL-HEADER-CONTROLS: Global Preference Controls Migration to Window Titlebar

| Metadata | Specification Details |
| :--- | :--- |
| **Status** | **APPROVED** (Direct Consensus & Directive from Human Owner in Mode 0) |
| **Version** | 1.0.0 |
| **Domain** | Desktop Application UI / Native Frameless Window Architecture |
| **Scope** | Window Header Titlebar, View-Specific Toolbars, Global Preferences Synchronization |
| **Owner** | System Architect |
| **Target Audience** | System Architects, Tactical Coders, QA Verification Engineers |

---

## 1. Executive Summary & Problem Domain

### 1.1 Problem Statement & Architectural Smell
In earlier iterations of the desktop client, global user preference controls—specifically the **Theme Selector** (Dark / Light / Warm) and the **Privacy Masking Switch** (Account Email / Sensitive Identifier Redaction)—were co-located inside the secondary toolbar of the primary engine tab.

This design suffered from significant architectural defects:
1. **Scope-Responsibility Mismatch:** Application theme and account privacy masking are global configuration concerns affecting the entire application surface. Embedding them inside an engine-specific tab implies local ownership and conceals them when switching to alternate tabs (such as the secondary engine view).
2. **Visual & Cognitive Clutter:** The secondary tab toolbar bore the combined cognitive load of engine operations, automated state switches, account management actions, and aesthetic toggles.
3. **Multi-Engine Inconsistency:** Alternate engine views lacked immediate visual access to theme and privacy toggles despite being styled and governed by the same underlying preference state.

### 1.2 Target Architectural State
The application architecture is restructured to elevate all global user preference controls to the **Top-Level Frameless Window Header (Titlebar)**. The secondary toolbar of the engine tab is strictly stripped of global controls, reserving its space exclusively for local engine automation and management workflows.

---

## 2. Component Hierarchy & Layer Boundaries

```text
┌─────────────────────────────────────────────────────────────────────────────┐
│ LAYER 1: FRAMELESS WINDOW ROOT CONTAINER                                    │
│                                                                             │
│ ┌─────────────────────────────────────────────────────────────────────────┐ │
│ │ TOP SECTION: WINDOW HEADER BAR (Global Scope - Persistent on All Tabs)  │ │
│ │                                                                         │ │
│ │ [Brand Icon + Title] ──── [Drag Area] ──── [Right Control Cluster]     │ │
│ │                                             │                           │ │
│ │                                             ├── Privacy Masking Control │ │
│ │                                             ├── Delimiter 1             │ │
│ │                                             ├── Theme Selector          │ │
│ │                                             ├── Delimiter 2             │ │
│ │                                             ├── Locale Selector         │ │
│ │                                             ├── Delimiter 3             │ │
│ │                                             └── Native Window Buttons   │ │
│ └─────────────────────────────────────────────────────────────────────────┘ │
│ ├─────────────────────────────────────────────────────────────────────────┤ │
│ │ Horizontal Divider (1dp border subtle)                                  │ │
│ ├─────────────────────────────────────────────────────────────────────────┤ │
│ │ WORKSPACE BODY (Split View)                                             │ │
│ │ ┌───────────────┐ ┌───────────────────────────────────────────────────┐ │ │
│ │ │ SIDEBAR NAV   │ │ TAB CONTENT STACK                                 │ │ │
│ │ │ (Persistent)  │ │                                                   │ │ │
│ │ │               │ │ ┌───────────────────────────────────────────────┐ │ │ │
│ │ │ • Engine 1    │ │ │ TAB 1: PRIMARY ENGINE VIEW                    │ │ │ │
│ │ │ • Engine 2    │ │ │ ┌───────────────────────────────────────────┐ │ │ │ │
│ │ │               │ │ │ │ Toolbar: [Engine Actions] ── [Auto Switch]│ │ │ │ │
│ │ │               │ │ │ └───────────────────────────────────────────┘ │ │ │ │
│ │ │               │ │ │ Card Grid (Honors Global Privacy Masking)   │ │ │ │ │
│ │ │               │ │ └───────────────────────────────────────────────┘ │ │ │
│ │ │               │ │ ┌───────────────────────────────────────────────┐ │ │ │
│ │ │               │ │ │ TAB 2: SECONDARY ENGINE VIEW                  │ │ │ │
│ │ │               │ │ │ (Honors Global Theme & Privacy Masking)       │ │ │ │
│ │ │               │ │ └───────────────────────────────────────────────┘ │ │ │
│ │ └───────────────┘ └───────────────────────────────────────────────────┘ │ │
│ ├─────────────────────────────────────────────────────────────────────────┤ │
│ │ FOOTER SECTION: Status & Telemetry Bar (Persistent)                     │ │
│ └─────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 2.1 Boundary Ownership Matrix

| System Component | Owning Architectural Layer | Scope | Lifecycle & Visibility |
| :--- | :--- | :--- | :--- |
| **Window Header Titlebar** | Window Shell Layer | Global Application | Always visible across all tabs |
| **Privacy Masking Control** | Global Preferences Subsystem | Global Application | Always visible on Header Titlebar |
| **Theme Selector** | Global Preferences Subsystem | Global Application | Always visible on Header Titlebar |
| **Locale Selector** | Global i18n Subsystem | Global Application | Always visible on Header Titlebar |
| **Native Window Buttons** | Native Platform Window Subsystem | Window Frame | Always visible on Header Titlebar |
| **Auto-Switch Control** | Engine Automation Subsystem | Engine 1 View Local | Visible only within Engine 1 Tab |
| **Engine Action Buttons** | Engine Management Subsystem | Engine 1 View Local | Visible only within Engine 1 Tab |
| **Account Cards** | Presentation Subsystem | View Local Content | Rendered in Engine 1 View |

---

## 3. Layout Contracts & Geometry Envelopes

### 3.1 Header Right Control Cluster Layout Contract

The right side of the Window Header Titlebar MUST adhere to the following sequence and geometric specifications:

$$\mathbf{[Mask\ Accounts\ Group]} \quad\Big|\quad \mathbf{[Theme\ Selector\ Group]} \quad\Big|\quad \mathbf{[Locale\ Selector\ Group]} \quad\Big|\quad \mathbf{[Window\ Buttons\ Group]}$$

#### Detailed Group Layout Sequence (Left to Right):
1. **Privacy Masking Group:**
   - **Text Label:** Bold typography displaying the localized privacy masking string (e.g., "Ẩn tài khoản:" / "Mask Accounts:").
   - **Capsule Toggle Switch:** Standard capsule pill switch with physical footprint of $23\text{dp} \times 12\text{dp}$.
   - Horizontal spacing between label and capsule toggle: Standard UI padding ($4\text{dp}$).
2. **Subtle Delimiter 1:**
   - Structural vertical separator of dimension $1\text{dp} \times 14\text{dp}$, rendered using the current theme's subtle border color.
   - Symmetric padding of $4\text{dp}$ on left and right flanks.
3. **Theme Selector Group:**
   - Compact selection dropdown containing 3 symbolic representations: Moon (`🌙`), Sun (`☀️`), and Warm Coffee (`☕`).
   - Dimension envelope: Width $58\text{dp}$, Height $20\text{dp}$.
4. **Subtle Delimiter 2:**
   - Structural vertical separator of dimension $1\text{dp} \times 14\text{dp}$ (identical to Delimiter 1).
5. **Locale Selector Group:**
   - Dropdown control containing two-letter ISO language tags (`VN` / `EN`).
   - Dimension envelope: Width $54\text{dp}$, Height $22\text{dp}$.
6. **Subtle Delimiter 3:**
   - Structural vertical separator of dimension $1\text{dp} \times 14\text{dp}$ separating preferences from native window operations.
7. **Window Actions Group:**
   - Three equal-sized rectangular buttons: Minimize (`—`), Maximize/Restore (`□` / `❐`), and Close (`✕`).
   - Dimension envelope: $32\text{dp} \times 26\text{dp}$ per button (total width: $96\text{dp}$).

---

### 3.2 Vertical Centerline Alignment Invariant

> **INVARIANT LAW [INV-UI-VCENTER-01]:**  
> All elements contained within the Header Right Control Cluster MUST be dynamically centered along the vertical axis of the header bar container:
>
> $$y_{\text{offset}} = \frac{H_{\text{container}} - H_{\text{element}}}{2}$$
>
> - The standard Titlebar height is constrained to $26\text{dp} - 28\text{dp}$.
> - The Capsule Toggle ($H=12\text{dp}$), Delimiters ($H=14\text{dp}$), Theme Selector ($H=20\text{dp}$), Locale Selector ($H=22\text{dp}$), and Window Buttons ($H=26\text{dp}$) MUST share the identical horizontal centerline.
> - Maximum allowable baseline deviation between the centerlines of any two adjacent elements in the cluster is strictly **$\le 1.0\text{dp}$**.

---

### 3.3 Window Width & Drag Area Budget Invariant

The application supports responsive resizing between minimum supported width and default launch width.

#### Geometry Breakdown Table:

| Viewport Profile | Width Budget | Left Branding Cluster | Right Control Cluster | Resulting Drag Area Budget | Safety Margin Status |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Default Width** | $1100.0\text{dp}$ | $\approx 235\text{dp}$ | $\approx 357\text{dp}$ | **$\approx 508\text{dp}$** | Optimal ($> 45\%$) |
| **Intermediate Width**| $800.0\text{dp}$ | $\approx 235\text{dp}$ | $\approx 357\text{dp}$ | **$\approx 208\text{dp}$** | Safe ($> 25\%$) |
| **Minimum Boundary**| $640.0\text{dp}$ | $\approx 235\text{dp}$ | $\approx 357\text{dp}$ | **$\approx 48\text{dp}$** | Constrained Threshold |

> **INVARIANT LAW [INV-UI-DRAG-AREA-02]:**  
> To guarantee that the window remains movable via mouse dragging under all circumstances:
> 1. The Left Branding Cluster Title text MUST implement automatic **Ellipsis Truncation** at small viewport widths.
> 2. The Right Control Cluster MUST NOT wrap into multi-line or collapse into invisible states.
> 3. The center Drag Area container MUST be protected against layout collapse, maintaining a minimum width of at least $40\text{dp}$ even under localized text expansion or high system DPI scaling.

---

### 3.4 Primary Engine Toolbar Simplification Contract

Following the migration of Theme Selector and Privacy Masking to the Window Header:

> **INVARIANT LAW [INV-UI-TABTOOLBAR-03]:**  
> 1. The right cluster of the Primary Engine Toolbar MUST retain only the **Engine Automated Switching Component** (comprising its bold label and capsule toggle switch).
> 2. All legacy separators and global selector wrappers previously hosted in the engine toolbar right cluster are strictly abolished.
> 3. The left cluster of the Primary Engine Toolbar retains its operational action buttons (Add Account, Refresh List, Refresh Quota, Pin Window).

---

## 4. State Ownership, Persistence & Synchronization Invariants

### 4.1 State Hierarchy
- **Single Source of Truth:** The Global Application State layer holds canonical in-memory state for:
  - `ActiveTheme` (`dark` | `light` | `warm`)
  - `PrivacyMaskingEnabled` (`boolean`)
  - `CurrentLocale` (`vi` | `en`)
- **Immediate Disk Persistence:** Any user interaction with the Theme Selector or Privacy Masking Capsule Switch MUST trigger immediate serialization to the persistent configuration store (`ui_preferences.json`).

### 4.2 In-Place Visual Synchronization (Zero-Jank Contract)
1. **Theme Switching Lifecycle:**
   - Mutation of the active theme triggers palette re-computation.
   - All Canvas rectangles, monogram badges, delimiters, card backgrounds, and border strokes MUST re-query their colors from the newly activated palette and issue a visual refresh.
   - Re-theming MUST apply globally across the Header, Sidebar, Primary Engine View, Secondary Engine View (Codex), and Dialog Modals without requiring application restart or complete window reconstruction.
2. **Privacy Masking Lifecycle:**
   - Mutation of the privacy masking switch immediately updates the internal boolean flag.
   - All visible account cards in the active view MUST mutate their displayed text representations in-place (switching between unmasked account strings and masked patterns).
   - In-place mutation MUST occur with zero layout shifts (card bounding boxes, heights, and margins remain strictly invariant).
   - Any inactive or hidden views (e.g., Secondary Engine View) MUST inspect the canonical privacy masking flag prior to presenting account records upon activation.

---

## 5. View Contracts & Multi-Tab Isolation

### 5.1 Primary Engine View Contract
- Must consume `PrivacyMaskingEnabled` state on every card build and in-place update.
- Must not expose local overrides for theme or masking.
- Must limit its right-aligned toolbar actions strictly to engine auto-switch mechanics.

### 5.2 Secondary Engine View (Codex) Contract
- Must inherit global palette styles upon instantiation and during runtime theme switch broadcasts.
- When account management or sensitive credentials are introduced to the secondary engine view in subsequent phases, it MUST consume the canonical `PrivacyMaskingEnabled` flag from the global state layer.

---

## 6. Testing & Acceptance Criteria (Quality Gate)

### 6.1 Viewport & Geometry Verification Suite
- **TC-SPEC-UI-01 [Header Composition]:** Verify that the Window Header contains the Privacy Masking Group, Theme Selector, Locale Selector, and Window Buttons in exact specified order from left to right.
- **TC-SPEC-UI-02 [Vertical Centerline Alignment]:** Measure the bounding box of the Privacy Capsule Toggle ($12\text{dp}$), Theme Select ($20\text{dp}$), and Locale Select ($22\text{dp}$). Vertical centerline offset between any two elements must not exceed $1.0\text{dp}$.
- **TC-SPEC-UI-03 [Drag Area Integrity at 640dp]:** Resize window viewport to $640\text{dp} \times 400\text{dp}$. Assert that drag area width remains $\ge 40\text{dp}$ and window remains movable via mouse dragging.
- **TC-SPEC-UI-04 [Tab Toolbar Simplification]:** Assert that the Primary Engine Toolbar right cluster contains solely the Auto-Switch component and contains zero delimiters or theme selectors.

### 6.2 State & Synchronization Verification Suite
- **TC-SPEC-UI-05 [Global Theme Propagation]:** Toggle theme across `🌙`, `☀️`, `☕`. Verify that Header elements, Sidebar elements, Tab 1 cards, and Tab 2 (Codex) background update colors instantaneously.
- **TC-SPEC-UI-06 [Global Privacy Masking Propagation]:** Toggle privacy masking switch. Verify that all email displays in Tab 1 update between masked format (`t***@domain.com`) and plain format without card re-layout jank.
- **TC-SPEC-UI-07 [Dynamic Locale Switch]:** Toggle language between `VN` and `EN`. Verify that the Privacy Masking label on the Header updates dynamically between "Ẩn tài khoản:" and "Mask Accounts:" without altering cluster alignment.

---

## 7. Forbidden Anti-Patterns (Architectural Prohibitions)

1. **FORBIDDEN:** Re-introducing global preference toggles into individual tab content containers or view toolbars.
2. **FORBIDDEN:** Hardcoding fixed element widths that cause text clipping or overflow when switching locales.
3. **FORBIDDEN:** Allowing the Header Right Control Cluster to wrap onto a second line under any supported viewport dimension.
4. **FORBIDDEN:** Permitting drag area starvation below $40\text{dp}$ on the minimum supported window width ($640\text{dp}$).
5. **FORBIDDEN:** Destructive whole-window tearing or recreation during runtime theme or masking state changes.

---

## 8. Non-Authoritative Implementation Guidance

*(This section provides non-binding suggestions for tactical implementation to assist downstream engineering lanes; it does not constitute architectural law).*

- In the Fyne desktop implementation, the Right Control Cluster can be structured using a dedicated lightweight layout implementation that iterates children, computes maximum row height, and applies $(H_{\text{container}} - H_{\text{child}}) / 2$ to `child.Move(pos)`.
- Existing public helper methods and getters on the main application object (such as accessors for the privacy toggle widget or its label) should be preserved or aliased to prevent breaking existing integration test suites.
- When registering delimiters, wrapping them in fixed-size grid wrappers (e.g., $1\text{dp} \times 14\text{dp}$) ensures standard container layouts do not collapse them to $0\text{dp}$ dimensions.
