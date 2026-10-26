# Product UX review — 2026-10-26

This is a bounded, evidence-backed iteration on PR #116, not a declaration that the full WOS product audit is complete. The branch remains `fix/workspace-operational-density`; PR #116 remains Draft/open.

## Method and evidence

The review used the installed `laws-of-ux-auditor` and `product-design-and-ux` skills, inspected the client navigation/state code and current domain-backed browser fixtures, and exercised the application in Chromium against its real Go API and isolated SQLite test database. `agent-browser` was not available in this environment, so Playwright drove the real browser. Screenshots and JSON measurements are source-bound to production commit [`be6eb6e`](https://github.com/A1b3rt0M3rcad0/wos/commit/be6eb6eebbeb4fcdec31010e9eecbd4de31621fd); metadata records the browser build, binary hash, embedded asset hashes, fixture hashes, viewport geometry, focus target and screenshot hash for each capture.

Findings use **Observed** for behavior reproduced in the running product, **Inferred** for a risk derived from inspected code or test design, and **Not tested** where this run did not establish behavior.

## Findings and changes

| Priority | Evidence | Finding | Resolution |
| --- | --- | --- | --- |
| P2 | Observed | Browser Back from an open Outcome removed the Outcome query string but left the selected Outcome surface rendered. The user could not return to the authenticated Outcome chooser with browser navigation. | Fixed in [`app.js`](../../../../packages/wos-api/web/assets/app.js): the null route now invalidates pending reads, clears only Outcome-scoped state, restores the chooser, expands navigation, deselects the prior Outcome, and returns keyboard focus to it. Forward navigation restores the Outcome. |
| P3 | Observed in test execution | The contextual Kanban regression waited for criterion text anywhere in the inspection panel. The Task's criteria list already contained that text, so this did not prove the criterion inspector had loaded before refresh/context changes. | The test now waits for the active inspection heading to equal the criterion title, removing the race without weakening assertions. |
| P3 | Observed in gallery qualification | A few source-bound gallery steps attempted operations that the domain correctly disallows (editing planning after activation, or concluding an Objective that had unrelated unfinished work); other assertions did not expand intentionally collapsed sections. | Updated the fixture to establish valid domain preconditions and the capture flow to expand/restore disclosures as a user would. No domain guard was relaxed. |

The gallery's representative screenshots show the chooser after browser Back, the Board and a selected Task. All screenshots are under [`screenshots/`](screenshots/).

![Authenticated Outcome chooser after browser Back at 1440×900](screenshots/outcome-chooser-back-1440x900.png)

![Outcome Board at 1440×900](screenshots/board-1440x900.png)

![Task inspection at 360×800](screenshots/task-360x800.png)

## Browser and test verification

- The first complete Playwright suite passed **239/239** against the real local API/SQLite fixture in **17m54s**.
- A second complete run exposed one brittle assertion in `operational-workspace.spec.mjs`: after a successful HTTP 200 and a closed form, the helper demanded that the globally shared (now hidden) submit button already be enabled. The actual view showed the saved execution specification in the Task inspector; the assertion did not test persistence or user capability. Removed that hidden-button check while retaining the HTTP response, form closure, non-error notice, persisted API read, and subsequent interactions.
- The complete `operational-workspace.spec.mjs` file passed **25/25** across five repetitions after that correction.
- That rerun then exposed a second test race in `operational-density.spec.mjs`: two sequential checks queried the same dynamic `Priority` control; the snapshot taken at failure showed the field visible and correctly named. Repeated the QHD case 5/5 before changing the helper, then made visibility, viewport containment and pointer targeting a single retried measurement. The QHD case passed **10/10** after this adjustment. No product form behavior or geometric assertion was removed.
- The final complete Playwright rerun passed **239/239** against the same real local API/SQLite fixture in **18m54s** after both helper fixes.
- The full UX unit suite passed **83/83**; the navigation plus Outcome-search browser suites passed **10/10**; and the contextual Kanban viewport regression passed **10/10**.
- The source-bound operational gallery passed **14/14** at production commit `be6eb6e`. It covers login, chooser, Back/Forward, Board, Task, specialist views, forms and operational previews at 11 viewport sizes: 1280×720, 1366×768, 1440×900, 1920×1080, 2560×1440, 3840×2160, 320×740, 360×800, 390×844, 430×932 and 768×1024. It also exercises native browser zoom at 125%, 150% and 200%.
- Existing Playwright coverage includes axe checks, keyboard-only reference selection and confirmation, 200%/400% reflow equivalents, authenticated permission states, real command journeys, and Board movement. Passing axe tests are useful automated evidence, not a claim of full WCAG 2.2 AA conformance.

This iteration's final full suite is green. The audit itself remains bounded: the open items listed above are not represented as verified or fixed by the passing tests.

## Scope that remains open

This pass concentrates on navigation restoration, source-bound visual evidence, the Board/Task context and existing representative human operations. It does not establish that every critical human journey in the product has been completed or that two consecutive comprehensive audits found no new high-severity issue.

- **Not tested:** a manual screen-reader review with assistive technology, native Windows/macOS/Safari browser behavior, or a full user study.
- **Not tested:** every permission and conflict permutation, especially concurrent edits to acceptance criteria and the recovery of an in-progress human form after a version conflict.
- **Not tested:** a complete specialist workflow for every entity type at each viewport. Representative specialist states were captured at QHD and 360px; the primary login/chooser/Board/Task matrix covers all 11 viewports.
- **Inferred follow-up:** expand human-flow coverage from Outcome discovery through planning, task assignment, execution, evidence, independent assessment, conclusion, issue/blocker remediation, and return navigation. Verify each transition against the existing backend operation and authorization contract.
- **Inferred follow-up:** inspect all empty/error/loading/denied states at mobile and desktop, and run manual keyboard and screen-reader passes beyond automated axe checks.

An independent source-level human-flow review also identified these next items; they are coverage or communication risks, not browser-reproduced defects:

1. **P2 — Identify the exact delivery in historical assessments.** Current criterion history displays assessment result, criterion revision, rationale, reviewer and time, while the current assessment path can link to a submission. The historical rendering does not visibly identify the submission/delivery reviewed. Next validation: assess delivery A, submit delivery B under the same criterion revision, and verify the UI identifies which assessment applies to which delivery and labels stale assessment accurately.
2. **P2 — Recover from a concurrent criterion change in the Evidence form.** The form carries the criterion revision and backend rejects stale submissions, but a browser journey has not established that the user gets clear recovery while retaining the draft. Next validation: open the Evidence form in one session, change the Criterion in another, submit, and verify the explanation, preserved fields and safe reselection path.
3. **P2 — Complete browser coverage for review modes and waiver permissions.** Existing domain checks and an authorized waiver journey exist, but the complete attestation, evidence-review and external-evaluation paths and the unauthorized waiver experience have not all been verified end to end.

These items should be triaged and resolved or explicitly qualified in later iterations before claiming complete critical-flow coverage.

## Reproduction

From `tests/web`:

```sh
WOS_TEST_CHROMIUM_PATH=/usr/bin/chromium ./node_modules/.bin/playwright test navigation.spec.mjs
WOS_TEST_CHROMIUM_PATH=/usr/bin/chromium ./node_modules/.bin/playwright test contextual-kanban.spec.mjs --grep 'contextual Kanban preserves Board'
WOS_TEST_CHROMIUM_PATH=/usr/bin/chromium WOS_CAPTURE_OPERATIONAL_WORKSPACE=/tmp/wos-product-verified ./node_modules/.bin/playwright test operational-gallery.spec.mjs
```

The opt-in gallery verifies its checked-out source commit, running binary identity and embedded frontend asset hashes before it writes screenshots. Never reuse these screenshots as evidence for a different source revision without regenerating them.
