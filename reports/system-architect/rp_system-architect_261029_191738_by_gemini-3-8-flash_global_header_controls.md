# System Architect Lane Report: Global Header Controls Migration Architecture & SPEC

| Report Metadata | Details |
| :--- | :--- |
| **Lane** | System Architect (Mode 0: Alignment $\rightarrow$ Mode 1: SPEC Authoring) |
| **Model** | Gemini 3.8 Flash |
| **Timestamp** | 2026-10-29 19:19:00 UTC+7 (261029_191738) |
| **Status** | Complete & Ready for Review |
| **Owner Authority** | Human Owner Explicit Directive & Consensus |

---

## 1. Scope of Work

This architecture session addressed the structural re-organization of global user preference controls across the native desktop application:
1. **Interactive Alignment (Mode 0):**
   - Engaged in structured two-way technical dialogue with the Human Owner regarding the relocation of the **Theme Selector** and **Privacy Masking Switch** from the secondary toolbar of the primary engine view to the top-level native window titlebar.
   - Grounded all proposals in the existing codebase reality (`internal/ui/window.go`, `internal/ui/app.go`, `internal/ui/codex_view.go`).
   - Conducted trade-off analysis between explicit full-text labels vs. compact icon buttons, geometry budget calculations across default ($1100\text{dp}$) and minimum ($640\text{dp}$) viewports, vertical centerline alignment invariants, and multi-engine view consistency.
   - Reached official consensus with the Human Owner on all design parameters.
2. **Deep Codebase Inspection & Specification Authoring (Mode 1):**
   - Authored the authoritative system specification: `Docs/SPEC/SPEC-ARCH-GLOBAL-HEADER-CONTROLS.md`.
   - Authored the formal Architecture Decision Record: `Docs/SPEC/ADR-001-MIGRATE-PREFERENCE-CONTROLS-TO-TITLEBAR.md`.
   - Strictly enforced all supreme iron rules: production-grade architecture from day one, zero specific function/variable names in normative SPEC sections, clear separation between architecture rules and implementation suggestions, and document lengths $\le 800$ lines.

---

## 2. Output Files Created

The following architectural artifacts were created:
1. **`Docs/SPEC/SPEC-ARCH-GLOBAL-HEADER-CONTROLS.md`**
   - Authoritative specification defining component boundaries, layer responsibilities, right-aligned titlebar control cluster sequence, geometry envelopes, vertical centerline alignment invariant ($y = (H_{container} - H_{element})/2$), drag area safety budget, primary engine toolbar simplification contract, state persistence invariants, zero-jank in-place visual refresh contracts, quality verification suite, and forbidden anti-patterns.
2. **`Docs/SPEC/ADR-001-MIGRATE-PREFERENCE-CONTROLS-TO-TITLEBAR.md`**
   - Architectural Decision Record detailing the context, decision, trade-offs, positive/negative consequences, mitigations, and rejected alternatives (localized toolbar duplication, icon-only toggle, modal settings).
3. **`reports/system-architect/rp_system-architect_261029_191738_by_gemini-3-8-flash_global_header_controls.md`**
   - This durable lane report documenting session scope, artifacts, architectural decisions, and commit references.

---

## 3. Decisions Made

1. **Relocation to Window Titlebar:** Both the Theme Selector (`🌙`, `☀️`, `☕`) and the Privacy Masking Switch are elevated from the local engine tab toolbar to the global Window Header Titlebar, placed to the left of the Locale Selector (`VN`/`EN`).
2. **Canonical Sequence & Visual Separation:** The right control cluster of the Titlebar is standardized in the sequence:
   $$\mathbf{[Mask\ Accounts\ Group]} \quad\Big|\quad \mathbf{[Theme\ Selector]} \quad\Big|\quad \mathbf{[Locale\ Selector]} \quad\Big|\quad \mathbf{[Window\ Buttons]}$$
   separated by subtle $1\text{dp} \times 14\text{dp}$ vertical delimiters.
3. **Explicit Full-Text Labeling for Privacy Masking:** In accordance with the Owner's directive, the privacy masking group retains its bold localized text label (`[Ẩn tài khoản: ○]` / `[Mask Accounts: ○]`) paired with the $23\text{dp} \times 12\text{dp}$ capsule toggle switch to maximize discoverability.
4. **Primary Engine Toolbar Simplification:** The engine tab toolbar right cluster is stripped of legacy dividers and theme selectors, retaining solely the engine-specific Automated Switching toggle (`[Tự động: ●]`).
5. **Drag Area Budget Protection Invariant:** Title text truncation with ellipsis is mandated to safeguard the titlebar drag area against collapse, guaranteeing a minimum of $40\text{dp}$ draggable area even at the $640\text{dp}$ minimum window width.
6. **Unified State & Multi-Tab Synchronization:** Theme and privacy masking states remain owned by the Global Application State layer with instant serialization to `ui_preferences.json`. All existing and future views (including the secondary engine view "Codex") inherit and reflect these states in-place with zero layout shifts.

---

## 4. Residual Open Questions

- **None.** All architectural boundaries, layout calculations, invariants, and implementation trade-offs were thoroughly analyzed, presented to the Human Owner, and finalized with explicit directive.

---

## 5. Commit Reference

- The newly authored architecture artifacts (`Docs/SPEC/*` and `reports/system-architect/*`) are staged and committed strictly following the artifact isolation rules.
- **Commit Reference:** `1aab9e1` (`docs(spec): add global header preference controls specification and ADR-001`)
