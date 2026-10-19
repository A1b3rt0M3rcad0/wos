# CI follow-up: inspecting a card after dragging

[GitHub browser run](https://github.com/A1b3rt0M3rcad0/wos/actions/runs/39336999399)
on `e5ac3a2` passed 195 cases and failed the movement journey when a card press
did not open inspection. The existing local qualification is retained as
source-specific evidence rather than relabelled as a successful CI result.

## Cause and correction

A drag suppresses the trailing browser click for 250 ms. The guard also swallowed
a separate user press on a card when it arrived within that interval. The faster
CI sequence exposed this; the original sequence passed six local repetitions,
so repeating it alone did not establish that the defect was gone.

A new valid pointer press now clears the preceding gesture's suppression before
starting its own gesture. A drag still arms suppression at release, so dragging
cannot accidentally select a card. Mouse, touch, keyboard activation, pan and
zoom retain their existing interaction paths. There is no layout, persistence
or authorization change.

## Regression evidence

The added browser case holds only the application interaction clock inside the
suppression window. It uses real pointer movement/press/release, native timers,
the real API and recorded Outcome. It verifies both that the drag does not open
inspection and that the following explicit click does. Business time and command
responses are not mocked. The case failed before the fix and passed after it.

All three movement journeys passed five repetitions: **15 passed**, no skipped
or flaky cases. The original test and its assertions remain enabled; no sleeps,
retries or larger timeouts were added. Reproduce from a built source binary:

```sh
cd tests/web
WOS_TEST_CHROMIUM_PATH=/usr/bin/chromium \
npx playwright test board-movement.spec.mjs --repeat-each=5 --workers=2
```

The complete browser and CI checks for the correction are tracked in
[PR #116](https://github.com/A1b3rt0M3rcad0/wos/pull/116/checks).
The 29 gallery captures retain their original source receipts; this input-only
fix does not change their visual composition. The PR stays draft without merge
or release.
