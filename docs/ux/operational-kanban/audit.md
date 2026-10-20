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
| UX-005 | Actions suspend/close Kanban and require manual Return. An uncertain response must retain the exact draft, rather than treating close as cancellation. | High / medium; P1, source-verified | Host existing forms inside the same dialog; retain canonical authorization, fencing/version/idempotency checks, conflict and exact-intent recovery. |
| UX-006 | Existing renewal/reacquisition commands were developer-only, leaving legacy reservation recovery undiscoverable in human flows. | High / medium; P1, source-verified | Explicit legacy-only human forms/capabilities; ownership, expiry and fresh fencing remain enforced. |
| UX-007 | A real concurrent edit test exposed optional stale map enrichment preventing the next authorized form from opening. | High / low; P1, reproduced | Kanban record hydration bypasses optional map expansion; conflict and exact-intent recovery stay on the same surface. |
| UX-008 | Native 400% zoom later reproduced an entry control hidden under Board chrome in a 320×225 CSS viewport. | High / low; P1, reproduced | Bound the Board inspector to the viewport at very short heights; retain ordinary HD/mobile layout. |

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

## Source traceability and exception paths

The original skill files retain their supplied language and MIT license. New
product documentation is English. The baseline source remains inspectable at
[`496be90`](https://github.com/A1b3rt0M3rcad0/wos/tree/496be90b8201d7fce9ea64eae2ffb4ecbcb58acb).

| Finding | Root-cause source | Recovery / disconfirming check |
|---|---|---|
| UX-001 | `workspace-kanban.js` selected-item append and `workspace-kanban.css` 340px inspection layout | The original Board inspector must remain usable; only Kanban selection changes. |
| UX-002 | `workspace-kanban.js` card button and read-only stages | A drag must issue zero commands until explicit confirmation; a viewer gets explanation, not write authority. |
| UX-003 | `workspace_capabilities.go` Objective branch versus `Service.StartObjective` and `Objective.CheckStart` | Missing criterion, inactive Outcome and real Blocker must still prevent starting. |
| UX-004 | `workspace-compact.js` moving work/people into generic management | Compact Board remains compact; focused Kanban shows actual instructions and distinct participants. No online-presence inference. |
| UX-005 | `app.js` Kanban `onAction` / canonical submit, `workspace-kanban.js` suspend/resume | Conflict retains input; a committed response loss retries the same immutable intent; unresolved intent cannot be dismissed. |
| UX-006 | `command-exposure.json`, lease commands and `work_item_lease.go` | A non-holder cannot renew. Expiry requires reacquisition with a new claim/fencing token. Contract protocols never get a parallel legacy lease. |
| UX-007 | `app.js` `showEntity` awaiting optional graph expansion before command preparation | Real concurrent edit → conflict → fresh creation works even if the map projection is older. The graph is not a second source of business authority. |
| UX-008 | `canvas.css` bounded stage and narrow-height inspector | Actual Chromium native zoom 400%, measured 320×225 CSS viewport; the trigger must receive a normal click. No force-click workaround. |

These are bounded reproductions and source mechanisms, not measured prevalence.
The existing automated signed journeys verify protocol semantics; the new complete
browser work journey uses supported legacy authority and authorized Assessment.
It does not claim independent signed review happened in the browser. The external
handoff fixture verifies that no such authority is fabricated.

## Validation-found regression in the intermediate implementation

Visual review of the intermediate `e403c80` completion form found its sticky footer
covering part of the final authored field. This was introduced during this follow-up,
not attributed to the original baseline. The canonical form now has a bounded
scrolling body and a separate footer in normal flex flow. The complete operation
journey asserts that the focused final field ends above the footer at every tested
viewport. The final-source gallery replaces intermediate captures.

## Acquired-contract continuation

The final source review also found that an in-progress contract-backed Task could
lack an execution capability in the common action list until review opened.
Requiring technical-record discovery here would fragment the handoff. A read-only
**Continue in authorized host** entry now opens the existing human-readable
contract/review component directly, with its real identity and protocol. It does
not synthesize a submit/finalize permission or run a host operation. Closing the
read returns to the focused Task in the same Kanban. A real acquired-contract
browser scenario asserts the current contract, no browser commands, one dialog,
no top-level technical JSON record and the contextual return.

The acquired-contract return scenario was expanded to assert the exact spatial
camera style. At intermediate source `9e7db06`, closing the embedded read changed
its horizontal translation from `-180.391px` to `-317.45px` in the controlled
1280×900 fixture. The old detail-close callback was revealing the Task on the
background map. The embedded-read close now invalidates its pending reads without
dismissing/revealing the spatial selection; leaving the Kanban restores a freshly
read origin without optional map expansion. The exact assertion is retained;
no tolerance, retry or timeout increase masks the regression.

## Asynchronous return focus

The subsequent complete regression exposed intermittent lost keyboard focus when
the origin inspector was rebuilt after closing the Kanban. Returning focus before
the refreshed read completed could target a detached control; a selector list
also preferred the inspector container over its action button. The return now
captures the opening trigger, awaits the real origin refresh, identifies its
replacement button and reopens its containing disclosure when necessary.

Two deterministic browser regressions hold the actual successful workspace
response until the dialog closes, then require the original control to be detached
and the replacement to receive focus. Both Outcome entry and Task criteria entry
are covered. This tests the asynchronous boundary rather than relying on an early
focus assertion that could pass before reconstruction.
