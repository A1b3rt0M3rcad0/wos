# Operational Kanban qualification

## Scope and evidence

This is the 2026-10-20 follow-up in draft PR #116. It extends the original contextual
Kanban implementation; older receipts/captures still describe their exact sources.
[Audit](audit.md) records the pre-change investigation. [Operation guide](README.md)
describes the implemented user flow. Final source/run identifiers and captures are
recorded below when the final regression completes.

No participant study, manual screen-reader session, physical touch-device test or
production throughput claim is substituted by these automated tests.

## Executable acceptance

| Contract | Evidence |
|---|---|
| Real create → assign → define → activate → acquire → renew → Artifact → Evidence → Assessment → Conclusion | `tests/web/kanban-operations.spec.mjs` repeats the actual browser/API journey at 1280×720, 2560×1440, 3840×2160, 390×844 |
| Assignment does not acquire authority; evidence does not assess; Task completion does not achieve Objective | Persisted state asserted between the steps in the same journey |
| Drag / single-pointer / keyboard alternatives prepare operations | Native drag and named select; blocked and read-only fixtures assert zero command dispatches |
| Objective start and assessed achievement are real commands | Current Criterion, explicit Start and Achieve; Outcome remains active |
| Issues and Blockers remain distinct | Resolve the Issue, assert Blocker stays active, explicitly confirm release, then acquire newly available work |
| Exact-intent recovery and conflict handling | Concurrent real API edit retains local form input; actual commit followed by aborted response, then identical body/key retry and one created Task |
| External signed/contract authority is not invented | Real contract-protocol fixture: movement shows external host handoff, no browser mutation/lease/contract |
| Current Objective gates / lease ownership and expiry | `workspace_objective_capability_test.go`; existing readiness, domain, authorization and race suites remain |
| Context preservation, partial pages, revision coherence, cancellation exclusions and credential revocation | Existing contextual Kanban and guard suites retained; expected surface updated to the single embedded dialog |
| HD, QHD, 4K, narrow widths and actual native zoom | Existing nine-width contextual suite plus native Chromium Settings zoom 125%, 150%, 200%, 400%; viewport/DPR are measured by the existing helper |
| Accessibility triage | Native modal/controls, keyboard Escape/back/focus assertions, scoped Axe checks; no global WCAG conformance claim |

No retries, skipped browser tests, forced clicks or enlarged timeouts are introduced
for this follow-up. Intermediate failures exposed real integration issues or stale
assumptions in tests; they are not reported as passing final qualification.

## Final run

Pending final-source browser regression, capture receipts and GitHub CI. Local
focused/whole affected-package checks are intermediate evidence only until then.
