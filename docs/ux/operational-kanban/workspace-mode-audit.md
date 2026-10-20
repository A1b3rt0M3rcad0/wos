# Workspace-mode UX and operational audit

Scope: PR #116, `fix/workspace-operational-density`, owner-approved
[workspace-mode plan](owner-plan-workspace-mode.md). This hybrid audit uses the
installed `laws-of-ux-auditor` and `product-design-and-ux` skills: task flows,
interface contracts, audit coverage/protocol, implementation patterns and WCAG
triage. It supersedes the prior [audit](audit.md) **for presentation only**.
The prior full-page item presentation remains historical evidence, not current guidance.

## Reproducible findings and corrections

Findings are observations from source, DOM and controlled browser/API fixtures,
not participant research. Priorities are qualitative impact/effort; confidence is
high for reproduced behavior, not for population-wide usability improvement.

| ID | Observed problem and user cost | Impact / effort | Implemented correction and evidence |
|---|---|---|---|
| K01 | The whole Kanban was a modal; selecting an item replaced its lanes. Users lost their planning surface. | High / medium | Nonmodal workspace region and compact inspector; lane node identity is asserted during selection. Before/after captures use the same actual Task journey. |
| K02 | Generic stages merged Backlog and To do, and suggested Objective verification. | High / medium | Three Objective lanes, five Task lanes; grouping uses actual lifecycle/display facts without changing public stage. Pure tests cover terminal precedence and review applicability. |
| K03 | A persistent move select competed with reading on every card. | Medium / low | Separate labelled Actions disclosure; click, keyboard and emulated touch prepare the same intention as native drag. No nested interactive controls. |
| K04 | Backlog directly to execution needed a legitimate intermediate activation. A one-step visual move would misrepresent authority. | High / medium | Two separately confirmed canonical commands, fresh projection between them; API assertions record To do before acquisition. |
| K05 | Blocked acceptance lacked a focused route from criterion to assessment and back to the intended operation. | High / medium | Requirements modal offers existing authorized remedies and rechecks the original intention after each confirmed command. No automatic acceptance, Start prerequisite or Evidence-as-Assessment shortcut. |
| K06 | Native drop preparation detached the dragged node; normal-flow hints moved its geometry. Actual browser drag stalled. | High / medium | Stable overlay hints; defer preparation until dragend. Real native drag test checks zero writes, trapped keyboard focus and cancellation. |
| K07 | Returning canonical content left a nested inspector inert; subsequent Board interaction could be inaccessible. | High / low | Explicitly restore nested host inertness on return/exit; delayed-read origin/focus regressions and full journeys cover it. |
| K08 | Expanded mobile filters consumed almost all lane height; at native 400% the Outcome heading intercepted the workspace exit. | High / low | Bounded independently scrolling controls, minimum useful lane height, short-viewport workspace stacking. Ten-width matrix and actual Chromium Settings zoom validate controls and return. |
| K10 | Global Outcome tools/actions still targeted the hidden, inert map inspector while the contextual workspace owned operations. | High / low | Suppress duplicate map controls only while Kanban is active; use the parent breadcrumb and contextual actions. Three viewport regressions assert suppression and restoration on return. |
| K09 | A confirmed write followed by a failed read could be confused with a failed command. | High / medium | Separate confirmed-command reconciliation from uncertain-write retry; test aborts the projection read and asserts one command only. Existing response-loss test retains identical body/key. |

Fitts supports labelled, adequately sized alternatives to precision dragging;
Hick-Hyman motivates grouping less frequent actions, not hiding necessary choices.
Context retention reduces the need to remember prior work. These are design
hypotheses with executable acceptance, not measured conversion or productivity gains.

## Domain audit

Existing workspace capabilities and canonical commands remain authoritative.
Readiness, dependencies, active blockers, permissions, expected versions, leases,
fencing, strict-work acknowledgements and signed-host boundaries are preserved.
A planned Objective may Achieve directly when authorized. Task conclusion does not
Achieve its Objective; Objective achievement does not accept its Outcome. An Issue
may resolve while its Blocker remains active. Criterion, Artifact, Evidence,
Assessment and Conclusion retain distinct operations and records.

The actual signed-review fixture proves Verification using a persisted pending
review case. A legacy v1 submission without that fact is deliberately **not**
shown there. Recorded assignment, execution authority and actual online presence
are not interchangeable. No backend API/schema, permission or business lifecycle
is introduced by this presentation change.

## Coverage and limits

Read, move, create, assign, acquire, renew, record material, assess, conclude,
remediate, hand off, conflict, uncertain response, confirmed-write/read failure,
revocation, partial pages, hierarchy and return are exercised against real APIs.
Viewport/zoom samples and source-bound screenshots are listed in
[qualification](validation.md) and [the acceptance matrix](workspace-mode-acceptance.json).
Keyboard and Axe evidence are bounded checks. Physical-device touch, manual
screen-reader sessions, other browser engines and participant comprehension remain
unqualified; emulated touch is explicitly labelled. No global WCAG conformance,
production load improvement or online-Agent claim is made.
