# Operational Kanban qualification

## Scope and evidence

This is the 2026-10-20 follow-up in draft PR #116. It extends the original contextual
Kanban implementation; older receipts/captures still describe their exact sources.
[Audit](audit.md) records the pre-change investigation. [Operation guide](README.md)
describes the implemented user flow. Final source/run identifiers and captures are
recorded below.

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

- Complete local browser regression at `5ef10fc`: **210 passed**, zero skipped, zero flaky retries (498.9 seconds). The API binary was built from this source; its runtime commit metadata was `unknown` and its SHA-256 is retained in the structured result.
- After rebuilding the same source with the correct server commit stamp, **32 contextual/operational cases passed** and produced the final captures. This changes version metadata only; the production source is identical.
- The nine-width context matrix and both delayed-origin focus regressions also passed **three repetitions: 33 executions**, zero skipped or flaky retries. These are overlapping scenarios, not 33 additional acceptance cases. The deterministic delayed-response tests wait for the original trigger to detach before asserting focus on its replacement.
- The complete run includes the full create/assign/acquire/prove/assess/conclude journey at HD/QHD/4K/mobile, existing contract/signed-protocol checks, native browser zoom through 400%, permission/conflict/recovery cases and contextual navigation. Intermediate failures are not counted as passing qualification.
- JavaScript units: **76 passed**. Both affected Go packages (`application` and `web`) passed; workspace capability/race checks passed. Local default tests explicitly skip tests requiring an actual PostgreSQL service; full CI supplies PostgreSQL and remains the acceptance gate for those checks.
- The supplied audit skill passed its structure/link validator and **four package tests**.
- **16 final-source screenshots** have clean runtime-source receipts, binary/source binding, image hashes and 44 asset-file hashes in the [gallery manifest](screenshots/manifest.json). Source `5ef10fc` is the final runtime implementation; later documentation commits do not change those assets.
- Historical successful runs at `f765f5d` (207 cases) and `ee31cb8` (30 focused cases) retain their separate source/statistics in the structured result. They are not added to the final count.

[Structured browser results](test-results.json) · [Gallery](screenshots/README.md) · [Live GitHub checks](https://github.com/A1b3rt0M3rcad0/wos/pull/116/checks).

GitHub check status is recorded live on the PR, separately from these local receipts. No merge, tag, version advance or publication is performed.
