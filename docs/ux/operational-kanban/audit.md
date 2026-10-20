# Operational Kanban audit — 2026-10-20

Scope: PR #116, `fix/workspace-operational-density`, baseline `496be90`.
Hybrid audit: Chromium against a real isolated SQLite API, public commands and
workspace reads, plus source/contract inspection. Synthetic data only. Product
copy remains English. No brand changes, agent runtime, status setter or release.

Methods: the installed [UX auditor](../../../.agents/skills/laws-of-ux-auditor/SKILL.md)
(coverage A–J, evidence protocol, seven-law research limits, implementation,
WCAG 2.2 AA triage and prioritization) and
[product-design-and-ux](../../../.agents/skills/product-design-and-ux/SKILL.md)
(task/state model, inclusive interface contracts, IA, content and external handoff).
Bibliographic models explain possible mechanisms; they do not demonstrate user
success, accessibility conformance or a quantified productivity gain.

## Baseline evidence and priorities

[Browser measurements and actual API responses](baseline-observations.json) and
[HD reproduction](before-hd.png). Baseline browser probe passed at 1280×720,
2560×1440, 3840×2160 and 390×844 CSS pixels. No real participant research.

| ID | Verified observation and consequence | Impact / effort; priority | Intervention and acceptance |
|---|---|---|---|
| UX-001 | Selecting a child adds a 340px inspector beside the HD columns; on mobile it overlays 360 of 376px. Detail and columns compete. | High / medium; P1, high confidence | Single in-place item page with Back; preserve loaded filters, hierarchy and column/board scroll. No second permanent sidebar. |
| UX-002 | All cards are read buttons; no move controls or draggable elements. Execution requires leaving the modal for another surface. | High / medium; P1, high confidence | Drag or named action opens the legitimate operation, never commits a visual stage. Same-context canonical forms and recovery. |
| UX-003 | `StartObjective` exists in the application but is absent from workspace capabilities. | High / low; P1, high confidence in source | Add authorized capability evaluated against the same readiness gates as the existing command. No new command or lifecycle. |
| UX-004 | Instructions and full responsibility/authority are behind generic management disclosure; primary actions are chosen by numeric priority, including definition actions. | High / medium; P1, source-verified | Focused item headings explain work, responsibility, authority, readiness, proof, acceptance and history; group actions by user goal. |
| UX-006 | Existing renewal/reacquisition commands were developer-only, leaving legacy reservation recovery undiscoverable in human flows. | High / medium; P1, source-verified | Explicit legacy-only human forms/capabilities; ownership, expiry and fresh fencing remain enforced. |
| UX-007 | A real concurrent edit test exposed optional stale map enrichment preventing the next authorized form from opening. | High / low; P1, reproduced | Kanban record hydration bypasses optional map expansion; conflict and exact-intent recovery stay on the same surface. |
| UX-008 | Native 400% zoom later reproduced an entry control hidden under Board chrome in a 320×225 CSS viewport. | High / low; P1, reproduced | Bound the Board inspector to the viewport at very short heights; retain ordinary HD/mobile layout. |
| UX-005 | Actions suspend/close Kanban and require manual Return. An uncertain response must retain the exact draft, rather than treating close as cancellation. | High / medium; P1, source-verified | Host existing forms inside the same dialog; retain canonical authorization, fencing/version/idempotency checks, conflict and exact-intent recovery. |

Impact/effort are qualitative engineering estimates, not measured reach or ROI.
Ordering follows critical operational paths and shared root causes (recognition
and continuity, not a magic limit on visible controls).

## Interaction decisions before implementation

- Keep the spatial Board and its compact inspector. In Kanban, item details replace
  the column region; Back restores the board. Use one native modal, never stack
  operation dialogs. Reuse existing detail/forms/action nodes and their handlers.
- Move means **prepare an operation**. No optimistic status changes. To do may
  activate backlog or release a lease; In progress claims authorized ready work
  or starts an Objective; Accepted opens explicit completion/achievement. All
  require confirmation through existing forms. Invalid intent shows the real
  blockers/reasons and the next permissible action.
- Verification is a projection. Opening review does not submit, assess or accept
  anything. Signed delivery/review uses the authorized external host; do not
  fabricate browser signatures or impersonate an executor/reviewer.
- Assignment, recorded authority and live presence remain distinct. Evidence,
  Artifact, Criterion, Assessment and Conclusion remain distinct.
- Preserve scope, selected item, filters, column/board scroll and unsaved forms;
  changed revisions require explicit refresh/review. Permissions are rechecked
  by the canonical command pipeline. No automatic intent/version replacement.

## Coverage and evidence boundaries

| Axis | Baseline coverage | Final verification gate |
|---|---|---|
| A Orientation / navigation | Source and real Board → Kanban → child | In-place detail/back, hierarchy, filters, focus, route |
| B Hierarchy / density | Four viewports, actual DOM geometry | HD/QHD/4K/mobile captures, long content |
| C Operational flows | Domain/application commands, leases, signed protocol | Create/assign/claim/prove/assess/complete real journey |
| D Interaction / feedback | Read-only cards, modal action handoff | Drag plus keyboard/single-pointer equivalent; busy/error |
| E Forms | Canonical form, conflict/recovery pipeline inspected | Same form nodes, retained inputs, exact uncertain retry |
| F Responsive behavior | Four CSS viewports | Short heights, narrow reflow, native browser zoom |
| G Accessibility | Native dialog/button semantics inspected | Keyboard, focus, targets, Axe triage; no global WCAG claim |
| H Recovery | Version-pinned reads and existing tests inspected | Stale/blocked/forbidden/delayed/uncertain operations |
| I Rich workspace | Real columns and inspector measurements | Pan/camera preservation, column scroll, truthful stage |
| J Content / trust | Domain vocabulary and authority distinctions | Concrete next action, explicit proof/acceptance boundary |

Manual screen-reader/device sessions, independent novice usability and production
load profiling are not performed by this audit. Existing CI is a regression
baseline, not evidence that a novice understands the experience.
