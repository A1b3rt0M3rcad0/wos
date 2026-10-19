# Operational workspace qualification

Implementation and acceptance are tracked separately. The [owner plan](owner-plan-2026-10-19.md)
is preserved verbatim; [ADR-038](../../adr/0038-operational-workspace.md) records the
accepted boundaries. The [108-scenario matrix](acceptance.json) identifies what
each assertion covers and what still needs evidence.

All 24 units record implementation. The matrix marks **74 scenarios verified**
and **34 partially verified**. The latter retain their specific missing UI
assertions, device checks or reference measurements; automated totals do not
silently satisfy those requirements or the independent comprehension study.

## Source and compatibility

Baseline: `5d945d3081f85927b80c686d695849bd1df73a5d`. Core state and projections
were introduced in `342dd6e`, transport exposure in `5c68f7e`, and the shared
workspace in `d5aecda`. The read-frame correction is `a8571c5`; operational
impact previews are implemented in `9b03c14` / `505f2e6`, with reviewed transport
contracts in `e36a8f4`. The final mobile reading correction is
`36d7d2ef7b3e663500a69008f8df335aa43981a8`. Its complete browser
qualification is recorded below. The capture-only follow-up is
`d458d4782d1813b75df4264605606837c9aff73d`; it changes only the opt-in helper
that restores header focus after returning from an action menu. All production
code and all 40 assets remain identical to the qualified functional source.
Subsequent evidence/documentation commits do not change the qualified source.

The earlier complete browser run used `6b2be17`. Its 161 journeys passed, but
visual review of the subsequent `076550b` capture found a nested-Task reading
gap at narrow widths. The retained before images at
[320 px](visual-regression-076550b2fbf8/task-320x740.png) and
[390 px](visual-regression-076550b2fbf8/task-390x844.png) include their original
source-bound JSON receipts. They are historical diagnosis, not final acceptance.
The correction keeps a single scrolling inspector on narrow screens, compact
44-pixel context controls and the original fonts, with new regressions for the
actual initial authored reading area and reachable actions at 320/390/430 px.

<details>
<summary>Implementation commits, oldest first</summary>

```text
4d33dce docs: record operational workspace plan and acceptance boundaries
342dd6e feat(core): add guarded operational workspace state and coherent reads
5c68f7e feat(api): expose atomic planning and contextual workspace operations
67f2917 test(core): verify exact newer-schema refusal during upgrade qualification
d5aecda feat(web): make the Outcome Board a contextual operational workspace
0a08d14 fix(web): preserve preparation drafts and compact native zoom controls
a8571c5 fix(core): pin workspace capability checks to the read frame time
5e9984c fix(web): keep initialization, compact inspection and dismissal coherent
9b03c14 feat(core): preview guarded conclusion and placement impacts
505f2e6 feat(web): review operational impacts before contextual changes
e36a8f4 feat(api): expose coherent operational impact reviews
362b421 test(web): verify conclusion acknowledgement blocks dispatch
9db6ae6 fix(web): fence late impact and preparation read failures
d61b20a fix(web): persist inspected routes before optional map expansion
43e0b4a fix(web): describe readiness without implying an active blocker
6b2be17 test(web): await the Board coverage status before delayed errors
076550b test(web): capture distinct Evidence views and narrow Board defaults
36d7d2e fix(web): preserve readable nested Task context on mobile
d458d47 test(web): restore header focus before gallery inspection
```

The final documentation/evidence commit follows this qualified implementation.

</details>

The reviewed API changes are additive. Existing creation defaults, signed
protocols, historical public schemas and released CLI contract fingerprints stay
compatible. New workspace projections quote versions and fencing integers
exactly; legacy numeric command fields retain their wire representation.
Generated adapters, catalog, HTTP schema, MCP and SDK are checked together.

Both persistent adapters use additive migrations 31 and 32. A source-stamped
upgrade verifies actual schema-30 signed fixtures, every preexisting table and
the exact old-binary refusal of schema 32. See [migration evidence](migrations.md).
No mixed-version rolling upgrade or in-place downgrade is claimed.

## Automated checks

The [backend receipt](backend-qualification.json) records **1,703 passing Go tests
and subtests**, nine explicitly identified skips and ten packages without tests
in the final real-PostgreSQL whole-repository run. Full vet, targeted race
qualification, generated contract parity and all 16 release/compatibility checks
passed. The final complete run started at `e36a8f4`; its 806-file Go/SQL/module
fingerprint remained identical through the UI-only descendants. Earlier
`d5aecda` and `a8571c5` development runs remain separate historical evidence.
The receipt identifies the commands, sources, counts and exact skip reasons.

The browser suite runs genuine isolated SQLite servers. It covers persistent
planning, Criterion/Evidence separation, signed authority, read-only and revoked
identities, frozen retries, local-draft recovery, scoped deep links, keyboard
navigation, native dialogs and Board inspection. Component-only fixtures are
identified separately and do not prove persistence or authorization.

The [browser receipt](browser-qualification.json) records one clean complete run:
**164 browser journeys and 74 JavaScript unit tests passed**, with no failures,
skips, retries or flaky results. The functional source, binary digest, all 40
assets and 58 test/helper hashes were unchanged through that run. An interrupted
development run and corrected test-fixture assumptions are described separately;
they are not counted as successful qualification. A separate capture receipt
binds the opt-in harness to its own exact source and served assets.

The PostgreSQL matrix runs against a real PostgreSQL 18 server. Concurrent-writer
and expiry-boundary regressions qualify coherent revisions and evaluation time.
Mutation paths retain fresh authoritative time and their existing locks.

Historical baseline runs are characterized separately: 57 JavaScript unit tests
passed; 119 of 122 browser journeys passed in the first full run and the three
timeouts passed individually without changing baseline source. The earlier
baseline worker test that lost a process under concurrent runner load passed in
isolation. These reruns do not become a fictional clean full-baseline result.

## Visual evidence

The [current gallery](../../design/editorial/operational-workspace/README.md)
contains **75 images and 75 individual receipts**, produced by **11 passing
capture jobs** at `d458d478`. Its [manifest](../../design/editorial/operational-workspace/manifest.json)
verifies image/receipt hashes, PNG dimensions, exact source and fixture hashes,
binary identity and all 40 actual served JavaScript, CSS and HTML assets. The
production files match the complete `36d7d2e` functional run. Historical evidence
is never overwritten.

Representative visual review inspected the final desktop HD/QHD/4K Board and
Task inspector, mobile login and Task inspection at 320/390 px, desktop/mobile
actions, the mobile move preview and native 200% zoom. The portable browser
receipt names the eleven inspected images. The corrected mobile reading area
is visible; move labels no longer run into their values. Long content uses the
actual inspector scroller, including at native 200% zoom.

One intermediate capture run stopped after returning from the newly photographed
mobile action menu: the helper checked the offscreen Close button before focusing
it. The correction uses the same real focus sequence as the passing functional
regression; it changes no production file or asset. All eleven capture jobs then
passed in a fresh source-bound directory. The earlier failure remains identified
in the browser receipt rather than being counted as a successful run.

Coverage includes CSS viewports 1280 × 720, 1366 × 768, 1920 × 1080, 2560 × 1440,
3840 × 2160, 320 × 740, 390 × 844 and 430 × 932, plus actual Chromium native
browser zoom at 125%, 150% and 200%. Device scale and native zoom are reported
separately. Headless Chromium is not evidence of an actual mobile virtual keyboard
or browser engines that were not executed.

## Performance and independent comprehension

The [F08 report](performance.md) records fixture sizes, SQL, serialized bytes,
percentiles and actual runner conditions. On the final Core source, the quiet
ten-concurrent-read case measured **216.09 ms p95 on SQLite** and **183.16 ms
p95 on PostgreSQL**. An initial PostgreSQL attempt stopped during untimed
preparation: global `ANALYZE` scanned accumulated test schemas and exceeded the
statement timeout. A separately archived test-only overlay limited preparation
to the fixture schema; timed reads and production code were unchanged.

The earlier PostgreSQL **834.39 ms p95** result remains recorded and exceeded
the proposed **400 ms** comparison point. The final local samples fall below
that point, but neither a successful benchmark nor a faster retry establishes
the specified reference-environment latency target.

The prescribed 2-vCPU / 4-GiB / SSD / controlled-50-ms-RTT environment and ten
authenticated sessions remain unqualified. The local harness measures service
calls and serialization on a different runner. First-map rendering,
local-feedback and signed-workflow targets must not be inferred from those
timings. No production SLA is claimed.

The independent five-participant comprehension study has not run. Recruit people
who did not design the interface, provide the actual content to enter and do not
explain internal command names before each task. Record completion time, mistakes,
abandonment and assistance. The owner protocol measures:

- One Outcome, two Objectives, four Tasks and preparation Criteria: median at
  most three minutes, without IDs or a technical console.
- Finding a Task's five information groups: at least four of five participants
  within 30 seconds, without switching the global view.
- Correctly distinguishing declared responsibility from execution authority and
  review, and Evidence from assessment and acceptance.

Use the complete [owner protocol](owner-plan-2026-10-19.md#6-teste-de-compreensão-humana)
for the remaining tasks and thresholds. Automated journeys and author walkthroughs
do not count as participant results.

The English [study protocol](human-study.md) supplies concrete content and all
six participant tasks, neutral facilitator instructions, timing rules and the
observer's answer keys. Its [blank results template](human-study-results.template.json)
records `not_run`, zero participants and null measurements. Preparing these
materials is not execution of the study.

## Delivery boundary

All work stays in draft PR #116 on `fix/workspace-operational-density`. There is
no merge, tag or publication. Current metadata remains 1.1.0; the coordinated
release queue plans 1.2.0. Review, hosted CI and the remaining acceptance work
are reported explicitly rather than silently marked complete.
