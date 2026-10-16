# Canvas verification record

Verified on 2026-10-16 against the implementation on `feat/visual-operational-canvas`, based on `master` at `14b6708`. All data in the screenshots is illustrative, created through real commands on an isolated SQLite server. The PR's checks report the final commit's CI result; these local observations are not a release qualification.

## Automated evidence

| Check | Observed result |
| --- | --- |
| Native Go web tests | Passed: embedded assets and command-exposure contracts |
| Web unit suite | 38 passed: exact revision/scope boundaries, pagination, typed relationships, immutable models, deterministic non-overlapping layout, independent camera scaling, registry and navigation |
| Existing browser journeys | 41 passed, including signed approved/changes-requested handoffs, independent review, specialist views, long collections, modal families and responsive scale |
| Integrated Canvas journeys | 8 passed against the real server: default/no-Roadmap, deep links, contextual create/edit, real dependency/cycle/removal commands, Issue/Blocker separation, Evidence context linkage, partial-map pagination, off-page search, viewer behavior, conflict reconciliation, exact uncertain replay and Draft/publication/activation |
| Renderer interaction tests | 6 passed: local drag/pan/layout, keyboard edges, pagination/search cancellation, retained query on coherent refresh, synthetic plan selection, low-zoom axe checks and suppression of discarded callback failures |
| Context, inspector and reference picker | Passed: safe authored text, bounded omissions, original live form nodes and pending intent, focused/nested Escape ownership, cancellation of queued and late search results, multiple selection |
| Real authorization/concurrency ordering | 4 passed: closed detail cannot reopen, obsolete Task A comparison cannot replace Task B's form, an earlier successful refresh cannot restore private content after a genuine revoked-grant 403, and a dismissed Task's delayed history error cannot overwrite the current context or notice |
| Final visual regressions | 2 affected journeys rechecked: nested Evidence provenance stays readable; a plan opens with its actual phase visible at a readable zoom |
| Source integrity and release queue | 160 SVG hashes match the supplied archive; local documentation targets resolve; diff checks and the read-only release plan pass |

The Canvas viewer journey checks 1440 × 1000, 1920 × 1080, 2560 × 1440, 3840 × 2160 and 390 × 844 CSS px, including an open contextual panel and absence of outer horizontal overflow. UI typography grows above the reference width without changing the internal camera zoom. Isolated composition checks also cover 320 px. Large-scope integration seeds 135 Tasks; pure layout tests cover 301 nodes; inherited authorized collection searches use 1,001 records. These are specific test sizes, not a benchmark for arbitrary workloads.

Axe checks use WCAG 2/2.1/2.2 A/AA tags and reject serious or critical findings. Explicit distant zoom hides miniature secondary map actions while preserving selectable cards and full controls in the inspector/List. This does not establish complete WCAG conformance.

## Reproduce

Use the repository's Go toolchain and Node requirements. Rebuild the embedded web application before browser tests:

```sh
go build -trimpath -o bin/wos ./packages/wos-api/cmd/wos
go test ./packages/wos-api/web/...
cd tests/web
npm ci
npx playwright install chromium
npm test
```

`npm test` runs the web unit suite and all Playwright files against isolated local server processes. A prepared Chromium executable can be supplied with `WOS_TEST_CHROMIUM_PATH`. When the environment has a read-only home, set `XDG_CACHE_HOME` to a private writable directory for the CLI runtime locks.

To regenerate the nine gallery images from the real Canvas journeys:

```sh
WOS_CAPTURE_CANVAS="$(pwd)/../../docs/design/operational-canvas" npx playwright test canvas.spec.mjs
```

Captures wait for fonts and preserve the actual application rendering. Board images use the canvas shell's bounds. The mobile image shows the contextual inspector after an explicit selection. No screenshot is a mockup or composite.

## Scope and remaining acceptance

Live refresh is explicit and follows successful commands. It retains local camera/placement and the chosen plan source while fencing obsolete reads; it is not a WebSocket push subscription. Local placement is not saved as domain state. Initial graph depth is three; a selected graph entity adds one bounded focal neighborhood. Known omissions, cursors, loaded counts, depth limits and exact model revision remain visible. Complete history is not loaded for every card.

No domain schema, database migration, agent runner, signing model or CLI acquisition protocol is changed. Mutation authority, optimistic versions, cycle checks and receipts remain enforced by the existing application layer. New human dependency removal retains its original `outcome:write` authorization and requires an explicit reason/version.

Representative-user usability, manual screen-reader acceptance, native browser zoom and additional browser engines remain unmeasured. Automated Chromium checks and visual inspection do not substitute for those observations. The owner previously reported that representative users and screen-reader testers are unavailable; no human acceptance is claimed here.
