# Contextual Kanban qualification

The [owner plan](owner-plan-2026-10-19.md) is preserved byte-for-byte. Its
[62-scenario matrix](acceptance.json) records implementation evidence and remaining
acceptance separately: **56 verified for their automated scope, six partially
verified**. This does not substitute for the independent comprehension study.

The [machine-readable receipt](qualification.json) contains the actual complete
browser run, per-case outcomes, Go results, explicit skips and source bindings.
[Application captures](screenshots/README.md) show the working interface.

## CI follow-up

The subsequent GitHub browser run on `e5ac3a2` passed 195 cases and failed the
post-pan card-inspection case. Its [gesture follow-up](ci-gesture-fix.md) records
the reproduced production defect and correction. The next CI run on `f856f93` passed 196 cases and exposed a separate deep-link
route race; the same follow-up records its controlled reproduction and correction.
Earlier local suites below remain evidence for their own sources, not proof that
these CI runs were green.

## Technical checks

- Complete 193-case Playwright suite at `198fa43`, without retries: see the receipt's exact
  expected, unexpected, skipped and flaky counts. This includes the previous
  workspace journeys and the new hierarchy, criteria, material, pagination,
  focus, cancelled-history and native zoom cases. Two additional strict-exclusion/
  revocation regressions at `5282237` passed separately; these are not presented
  as a single 195-case execution.
- Final contextual increment at `356c5e6`: **20 browser cases passed**, including
  the selected-detail/history revision regression, with 29 fresh captures.
- Whole-repository Go: **1,124 passing tests/subtests; 126 explicit skips** in the
  default environment, plus ten packages without tests. The skips include
  PostgreSQL-dependent tests without a DSN, native keyring/full-disk fixtures
  and opt-in load/process scenarios. These are not counted as passes.
- Actual PostgreSQL storage suite: **450 passing tests/subtests; two opt-in load
  matrices skipped**. This is an additional adapter qualification, not a number
  to add to the default whole-repository count: suites overlap.
- Targeted core race checks: seven passing tests/subtests for stage precedence
  and strict completion. This is not a full-repository race claim.
- CLI recovery: the CI-discovered test-fixture data race was fixed with a
  mutex protecting the handler's request count and idempotency-key set. The
  specific case passed ten repeated race runs; the broader CLI race run has
  its own receipt, with native credential fixtures explicitly skipped locally.
- JavaScript units: **74 passed**. Package, skill, native-test-runner and release
  compatibility checks: **32 passed**. Full vet and generated adapter/catalog/
  SDK/event-catalog parity are checked without weakening the gates.

The [compatibility review](../../stable/operational-contract-review.md#contextual-kanbans-and-opt-in-strict-work--2026-10-19)
explains the additive fields and current candidate fingerprint. Historical
released schemas and all CLI YAML fingerprints remain intact. Migration 0033 is
forward-only; this work does not certify a mixed-version deployment or downgrade.

The final read-coherence regression first reproduced a selected Objective still
showing its old title after the columns had refreshed. The correction refreshes
parent, page and selected detail as one revision-matched set and fences separate
criterion/history reads. The final targeted Kanban suite is recorded separately
from the earlier complete 193-case run; their overlapping cases are not added.

## Source and visual evidence

The 29 paired PNG/JSON captures come from source
`356c5e650027d7e5ca703fa3c4494133ccb87e30`, with a matching stamped binary and
clean production/test tree at capture time. Each capture verifies all 43 served
CSS/JavaScript/entry HTML assets against source before writing the original
browser pixels. The [manifest](screenshots/manifest.json) binds image and receipt
checksums. Subsequent documentation commits preserve these bytes.

The viewport matrix is 1024×768, 1280×720, 1366×768, 1920×1080, 2560×1440,
3840×2160, 320×740, 390×844 and 430×932. Native browser zoom at 125%, 150% and
200% is exercised separately. Desktop, mobile, criteria and material captures
were visually inspected; the authored nested-Task reading area at 320 px also
has an assertion on the actual initial visible prose, not merely its DOM presence.

## Development findings retained

Earlier runs are diagnostic history, never silently added together as a clean
suite. The receipt retains the 191-case development run (172 passed, 19 failed),
the 21-case follow-up (14 passed, seven failed), and the subsequent broad run.
The initial failures exposed both old tests assuming always-expanded content and
actual disclosure-target/prose-order defects. Tests now use the real disclosure
controls; geometry, permission and lifecycle assertions remain enabled.

An additional broad run exposed timing in two older keyboard journeys: form
Escape requires an explicitly focused form target, and the reference-picker
journey must wait for its own debounced query before selecting a result. These
synchronizations retain the original focus, persistence and recovery assertions.
An initial package check also rejected the unreviewed current OpenAPI digest;
its reviewed update passes the unchanged historical shape and exact-hash gates.

## Remaining acceptance

- Independent five-participant comprehension study: **not performed**. Use the
  [study protocol](../operational-workspace/human-study.md); no synthetic result
  is presented as a participant observation.
- Physical mobile keyboard/touch and manual screen-reader/browser combinations.
- Reference large-workload latency and SQL query-count profiling. The 105-record
  fixture proves bounded pagination and full strict obligations, not an SLO or
  production scalability result.
- The matrix identifies the narrower remaining keyboard-only/transport assertion
  scope. Two additional browser regressions do cover strict cancellation
  acknowledgement through persisted Conclusion and genuine credential revocation
  while the Kanban is open. Their separate receipt is not relabelled as another
  complete-suite run.

The dedicated aggregate Kanban endpoint is deferred using the owner's allowed
bounded V1. Loaded search/filter results and partial coverage are labelled;
unknown scope is never presented as a verified zero. No new Cloud layer, agent
runtime, file-upload backend, automatic parent review or lifecycle drag is added.
The PR remains draft, without merge, tag or publication.

Local documentation links and anchors are checked in the [link receipt](documentation-links.json).
