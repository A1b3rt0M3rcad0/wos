# Workspace navigation tour

[Watch the tour](workspace-tour.gif) · [Open the static Canvas image](workspace-tour.png) · [Machine-readable provenance](workspace-tour.json)

The 28-second GIF shows the actual workspace running on a local WOS server with an isolated SQLite database. Its example Outcome, **Ship verified partial refunds**, connects implementation, review, and publication Tasks to a shared result. The seven frames are genuine browser screenshots, each held for four seconds; they are not a continuous-time recording. There are no UI overlays or invented application responses.

| Time    | Actual screen    | What to look for                                                                           |
| ------- | ---------------- | ------------------------------------------------------------------------------------------ |
| 0–4 s   | Canvas           | The Outcome, Objective, Tasks, and explicit connections share one workspace.               |
| 4–8 s   | Task inspector   | The implementation is Done, with its scope and external repository instructions preserved. |
| 8–12 s  | Work             | Implementation, review, and publication are separate Tasks with distinct current states.   |
| 12–16 s | Evidence         | A persisted example test-report observation is available for inspection.                   |
| 16–20 s | Evidence detail  | The report has recorded source and authorship information.                                 |
| 20–24 s | Overview         | Task progress is 33%, while criteria met remains 0%.                                       |
| 24–28 s | Success criteria | The required evidence-review criterion is explicit; the Outcome is still Active.           |

The fixture is illustrative. It does not implement a refund endpoint, run tests, access the example report URL, perform an independent review, or accept a delivery. One trusted local operator creates example records through public command APIs before navigation begins. No Assessment or Conclusion is created. In a real integration, participants execute and inspect the actual delivery outside WOS, then use their configured authority to record the review and explicit conclusion.

## Reproduce

Use the repository's Go toolchain, existing browser development dependencies, Chromium, and `ffmpeg`:

```sh
go build -o bin/wos ./packages/wos-api/cmd/wos
npm ci --prefix tests/web
WOS_TEST_CHROMIUM_PATH=/path/to/chromium node tools/documentation/record-workspace-tour.mjs
```

The [recorder](../../tools/documentation/record-workspace-tour.mjs) launches a loopback-only server, creates and removes its own temporary database, signs in before capturing, and retains no credential or browser session. It uses the existing browser navigation helper. The source application, brand, robots, supplied post-its, dependencies, and domain behavior are unchanged.

The capture uses a 1280 × 1100 CSS-pixel desktop viewport at device scale 1. The GIF is encoded at 1120 × 963 pixels, 2 fps, with a standard palette and resize; the static PNG retains 1280 × 1100 pixels. Native Canvas zoom/pan makes the main cards readable. Normal document scrolling keeps the three Work rows visible. The file's decoded duration is 28 seconds.

The JSON records the source commit, served web-input hashes, native binary hash, recorder/helper hashes, browser and Playwright versions, stage timestamps, source-frame hashes, and output hashes. The recorder verifies each served top-level JavaScript, CSS, HTML, and JSON input against the checkout. It does not claim stronger native binary-to-source build attestation.

Executed checks confirm the expected persisted records render; the primary Task card and all three Work rows are in the viewport; visible images decode; pages have no horizontal overflow; and authenticated navigation produces no failed same-origin responses, script errors, or domain mutations. The Outcome version is unchanged, with no criterion assessment or conclusion. This is a desktop product explanation, not a new browser-platform, signed-execution, accessibility, or production qualification.
