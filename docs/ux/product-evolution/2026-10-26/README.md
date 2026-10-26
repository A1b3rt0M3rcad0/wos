# Product UX review — 2026-10-26

This is a bounded, evidence-backed product iteration, not a declaration that the full WOS audit is complete. PR #116 had already merged as `840292c` before the newer work in this branch was pushed; no merge or release was performed for these changes. The follow-up is reviewable in [Draft PR #117](https://github.com/A1b3rt0M3rcad0/wos/pull/117) on `fix/workspace-operational-density`.

## Method and evidence

The review used the installed `laws-of-ux-auditor` and `product-design-and-ux` skills, inspected client navigation/state code and domain-backed browser fixtures, and exercised WOS in Chromium against an isolated SQLite database. `agent-browser` was not available, so Playwright drove Chromium. The original gallery remains source-bound to [`be6eb6e`](https://github.com/A1b3rt0M3rcad0/wos/commit/be6eb6eebbeb4fcdec31010e9eecbd4de31621fd). The later criterion-history screenshots are separately bound to their frontend source commit and record the API binary hash and exact browser asset delivery mode; see their JSON sidecars.

Findings use **Observed** for behavior reproduced in the running product, **Inferred** for a risk derived from inspected code or test design, and **Not tested** where this run did not establish behavior.

## Findings and changes

| Priority | Evidence | Finding | Resolution |
| --- | --- | --- | --- |
| P2 | Observed | Browser Back from an open Outcome removed the Outcome query string but left the selected Outcome surface rendered. The user could not return to the authenticated Outcome chooser with browser navigation. | Fixed in [`app.js`](../../../../packages/wos-api/web/assets/app.js): the null route now invalidates pending reads, clears only Outcome-scoped state, restores the chooser, expands navigation, deselects the prior Outcome, and returns keyboard focus to it. Forward navigation restores the Outcome. |
| P2 | Observed | Historical Criterion assessments showed the result, revision, rationale, reviewer and date, but not the immutable TaskDelivery they evaluated, even though the backend persisted the exact delivery ID, digest and round. Opening the history also appended the whole technical entity payload. | The focused history now shows the delivery round and ID, reveals the full digest on demand, displays any exact Submission reference, and labels linked Evidence with a direct inspection action. Criterion-history inspection no longer appends the unrelated raw entity payload. |
| P2 | Observed | After a correction round, the old assessment remained visible beside the new pending delivery without enough context to explain why it could not satisfy the current review; the basic human assessment form also did not bind a new assessment to the exact current delivery. | The current round and ID now appear beside the stale assessment, with the current digest available on demand. Review proof exposes the exact pending delivery; new human assessments bind its ID, digest and round. The human can request changes against that delivery or accept it only after current required criteria have applicable met assessments. |
| P3 | Observed in test execution | The contextual Kanban regression waited for criterion text anywhere in the inspection panel. The Task's criteria list already contained that text, so this did not prove the criterion inspector had loaded before refresh/context changes. | The test now waits for the active inspection heading to equal the criterion title, removing the race without weakening assertions. |
| P3 | Observed in gallery qualification | A few source-bound gallery steps attempted operations that the domain correctly disallows (editing planning after activation, or concluding an Objective that had unrelated unfinished work); other assertions did not expand intentionally collapsed sections. | Updated the fixture to establish valid domain preconditions and the capture flow to expand/restore disclosures as a user would. No domain guard was relaxed. |

The gallery's representative screenshots show the chooser after browser Back, the Board and a selected Task. All screenshots are under [`screenshots/`](screenshots/).

![Authenticated Outcome chooser after browser Back at 1440×900](screenshots/outcome-chooser-back-1440x900.png)

![Outcome Board at 1440×900](screenshots/board-1440x900.png)

![Task inspection at 360×800](screenshots/task-360x800.png)

### Criterion assessment and delivery provenance

The focused history now distinguishes the assessment from the exact delivery and linked Evidence. The full digest remains available without overwhelming the default reading view.

![Criterion assessment tied to the exact Task delivery at 1440×1000](screenshots/criterion-history-delivery-1440.png)

![Criterion delivery provenance in the mobile inspector at 360×800](screenshots/criterion-history-delivery-360.png)

## Browser and test verification

- The first complete Playwright suite passed **239/239** against the real local API/SQLite fixture in **17m54s**.
- A second complete run exposed one brittle assertion in `operational-workspace.spec.mjs`: after a successful HTTP 200 and a closed form, the helper demanded that the globally shared (now hidden) submit button already be enabled. The actual view showed the saved execution specification in the Task inspector; the assertion did not test persistence or user capability. Removed that hidden-button check while retaining the HTTP response, form closure, non-error notice, persisted API read, and subsequent interactions.
- The complete `operational-workspace.spec.mjs` file passed **25/25** across five repetitions after that correction.
- That rerun then exposed a second test race in `operational-density.spec.mjs`: two sequential checks queried the same dynamic `Priority` control; the snapshot taken at failure showed the field visible and correctly named. Repeated the QHD case 5/5 before changing the helper, then made visibility, viewport containment and pointer targeting a single retried measurement. The QHD case passed **10/10** after this adjustment. No product form behavior or geometric assertion was removed.
- The complete Playwright rerun for the preceding source revision passed **239/239** against the local API/SQLite fixture in **18m54s** after both helper fixes. This full-suite result predates the criterion-history implementation below.
- The full UX unit suite passed **83/83** after the criterion-history implementation.
- The new real-browser delivery-history and human review journeys passed **12/12** over three repetitions at 1440×1000 and 360×800. They create a Task, acquire a real lease, submit delivery A, assess its exact ID/digest/round, return it for correction, submit delivery B, confirm that A's stale assessment cannot accept B, assess B through the human Review proof form, and accept B through the existing domain operation. A separate journey requests changes through the human form and asserts its exact delivery ID, digest, round and correction findings in the HTTP command.
- Axe checks scoped to assessment history, Review proof, assessment, acceptance and request-changes forms found no critical or serious violations tagged through WCAG 2.2 AA. The mobile journeys also verify operation/modal bounds and no horizontal overflow. This is automated evidence, not a complete accessibility certification.
- The navigation plus Outcome-search browser suites passed **10/10**, and the contextual Kanban viewport regression passed **10/10** on the preceding iteration.
- The source-bound operational gallery passed **14/14** at production commit `be6eb6e`. It covers login, chooser, Back/Forward, Board, Task, specialist views, forms and operational previews at 11 viewport sizes: 1280×720, 1366×768, 1440×900, 1920×1080, 2560×1440, 3840×2160, 320×740, 360×800, 390×844, 430×932 and 768×1024. It also exercises native browser zoom at 125%, 150% and 200%.
- Existing Playwright coverage includes axe checks, keyboard-only reference selection and confirmation, 200%/400% reflow equivalents, authenticated permission states, real command journeys, and Board movement. Passing axe tests are useful automated evidence, not a claim of full WCAG 2.2 AA conformance.

The focused checks for the current iteration are green. A full Playwright suite has not yet been rerun on the new head, and the audit remains bounded: open items listed below are not represented as verified or fixed by the passing tests.

## Scope that remains open

This pass concentrates on navigation restoration, source-bound visual evidence, the Board/Task context and existing representative human operations. It does not establish that every critical human journey in the product has been completed or that two consecutive comprehensive audits found no new high-severity issue.

- **Not tested:** a manual screen-reader review with assistive technology, native Windows/macOS/Safari browser behavior, or a full user study.
- **Not tested:** every permission and conflict permutation, especially concurrent edits to acceptance criteria and the recovery of an in-progress human form after a version conflict.
- **Not tested:** a complete specialist workflow for every entity type at each viewport. Representative specialist states were captured at QHD and 360px; the primary login/chooser/Board/Task matrix covers all 11 viewports.
- **Inferred follow-up:** expand human-flow coverage from Outcome discovery through planning, task assignment, execution, evidence, independent assessment, conclusion, issue/blocker remediation, and return navigation. Verify each transition against the existing backend operation and authorization contract.
- **Inferred follow-up:** inspect all empty/error/loading/denied states at mobile and desktop, and run manual keyboard and screen-reader passes beyond automated axe checks.

An independent source-level human-flow review also identified these next items; they are coverage or communication risks, not browser-reproduced defects:

1. **P2 — Recover from a concurrent Criterion change in the Evidence form.** The form carries the criterion revision and backend rejects stale submissions, but a browser journey has not established that the user gets a clear recovery while retaining the draft. Next validation: open the Evidence form in one session, change the Criterion in another, submit, and verify the explanation, preserved fields and safe reselection path.
2. **P2 — Complete browser coverage for review modes and waiver permissions.** Existing domain checks and an authorized waiver journey exist, but the complete attestation, evidence-review and external-evaluation paths and the unauthorized waiver experience have not all been verified end to end.
3. **Not yet repeated on the current head:** the complete Playwright suite and source-bound operational gallery. The prior full-suite and gallery results belong to earlier source revisions; the current qualification is the focused 12-case browser run plus 83 unit tests.

## Current build limitation

This executor's `/usr/bin/go` is not a working Go toolchain (`go version` responds `Go: Unknown option: version`), so it could not rebuild the Go server with the changed embedded assets. For the new browser journeys, Playwright serves the exact checkout versions of `app.js`, `actions.js`, `en-US.js`, `human-forms.js`, `workspace-kanban.js`, `workspace-kanban.css`, `workspace-view.js` and `workspace-view.css` into real Chromium, while `bin/wos` supplies the authenticated HTTP API and isolated SQLite state. The screenshot sidecars record that split and the API binary's SHA-256. CI must still build the current Go binary and exercise the assets in the embedded production package before this iteration is considered fully build-verified.

These items should be triaged and resolved or explicitly qualified in later iterations before claiming complete critical-flow coverage.

## Reproduction

From `tests/web`:

```sh
WOS_TEST_CHROMIUM_PATH=/usr/bin/chromium ./node_modules/.bin/playwright test navigation.spec.mjs
WOS_TEST_CHROMIUM_PATH=/usr/bin/chromium ./node_modules/.bin/playwright test contextual-kanban.spec.mjs --grep 'contextual Kanban preserves Board'
WOS_TEST_CHROMIUM_PATH=/usr/bin/chromium WOS_CAPTURE_OPERATIONAL_WORKSPACE=/tmp/wos-product-verified ./node_modules/.bin/playwright test operational-gallery.spec.mjs
```

The opt-in gallery verifies its checked-out source commit, running binary identity and embedded frontend asset hashes before it writes screenshots. Never reuse these screenshots as evidence for a different source revision without regenerating them.
