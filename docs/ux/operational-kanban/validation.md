# Contextual workspace Kanban qualification

This is the owner-approved workspace-mode follow-up in draft PR #116.
[Current audit](workspace-mode-audit.md), [behavior contract](workspace-mode-contract.md)
and [A01–A30 acceptance matrix](workspace-mode-acceptance.json) describe its scope.
The previous modal/full-page presentation is superseded. Its receipts remain
[immutable historical evidence](https://github.com/A1b3rt0M3rcad0/wos/blob/75a56fe43ddd20ff55835fd2ba22ca8af9588d1c/docs/ux/operational-kanban/validation.md).

## Executed results

- Runtime source: `57e76f4741c4e483c304cf35c648550c0349c460`; clean runtime/test working tree and correctly stamped server binary.
- Complete local browser run: **220 passed, 2 failed**, zero skipped or flaky retries (821.6 seconds), two workers. The unmodified full receipt retains both failures.
- Corrective targeted qualification covers **both failed cases**, on identical production assets. The Objective assertion now targets the active requirements modal. The traced diagnostic passes that case but records a browser interruption in the administrative case; the separate one-worker administrative run passes without tracing. All original failures remain in the structured receipt. No timeout, assertion or browser test was removed. These runs are not a new all-green full run.
- The 222 distinct cases are covered across these local runs. **The final-head hosted browser job is the complete-suite acceptance gate**; its result is separate from the local receipts.
- JavaScript units: **78 passed**, zero failures/skips.
- Both Go packages `application` and `web` pass. Local default-environment tests that require PostgreSQL are explicitly listed as skipped in the structured receipt; hosted CI supplies the real PostgreSQL gate.
- **16 current screenshots**, HD/QHD/4K/mobile, each with source/viewport/browser receipt. The manifest binds source, stamped binary, asset hashes and image hashes. Later test/documentation-only commits do not change these assets; the production-file diff is checked before finalizing this receipt.
- Four real pre-change Task journeys passed and produced [before captures](before-workspace-mode/README.md). Focused and historical executions overlap the complete suite and are not additional acceptance counts.

[Structured results](test-results.json) retain each browser case, source, hashes,
Go results/skips and manual limitations. [Hosted PR checks](https://github.com/A1b3rt0M3rcad0/wos/pull/116/checks)
are the separate final-head CI gate; a historical green commit does not qualify a new head.

## Operational and exception coverage

The real-browser cycle creates, assigns, defines, activates, acquires and renews a
Task, then registers Artifact, Evidence, current Criterion Assessment and explicit
Conclusion. Persisted intermediate states are checked; completion does not achieve
the parent Objective. It runs at 1280×720, 2560×1440, 3840×2160 and 390×844.

Objective tests cover authorized Start, direct planned→Achieved, blocked acceptance
remediation and explicit open-work acknowledgements. Actual signed-review fixtures
prove Verification without browser signing; legacy submission does not invent that
fact. Issue resolution and Blocker release remain distinct writes.

Native drag and card menus prepare rather than write. Backlog→execution has two
explicit confirmations with a fresh To do projection between them. Nonmodal item
selection retains the same lane nodes. Only the canonical operation is modal.

Recovery tests cover conflict/input retention, persisted command with lost response
and exact body/key replay, confirmed write with failed refresh and reads-only retry,
revocation, partial pages, stale cursors and delayed origin focus restoration.

## Responsive and accessibility scope

The ten-width context matrix includes 320, 390, 430, 1024, 1280, 1366, 1440, 1920,
2560 and 3840 CSS-pixel widths. Actual Chromium Settings zoom 125%, 150%, 200% and
400% measures effective viewport/DPR; the 400% case opens a real operation, focuses
its last field, reaches Save and exits. It is separate from equivalent-width reflow.

Keyboard, Escape, Tab trapping, accessible card menus, emulated touch and scoped Axe
checks pass. No automatic Playwright retries, forced clicks, skipped browser tests or relaxed timeouts
were introduced. Physical-device touch, manual screen-reader combinations, other
engines, participant studies and production performance remain unqualified. These
checks do not establish global WCAG conformance or measured productivity gains.

No merge, release, tag, version change or publication is performed.
