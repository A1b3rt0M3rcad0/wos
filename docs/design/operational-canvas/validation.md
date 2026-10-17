# Operational workspace verification

## Complete workspace follow-up — 2026-10-17

The [all-page review and gallery](workspace/README.md) extends the Canvas delivery to sign-in, navigation, empty Workspace, Overview, every Plan/Work/Issues/Evidence/Activity subtab, documentary details, native forms, administration and technical tools. It preserves the original brand, robots, supplied post-its, tokens, fixed paper, existing structural grids and responsive scale. The specialist page scaffold is more compact so records appear sooner; no area or record is reordered.

The source correction includes functional checks beyond paint: Activity/Relations and Active Roadmaps search their actual loaded fields; priority appears only for Tasks/Objectives; Objective hierarchy avoids nested buttons; Roadmaps use bounded candidate pages; list and Board reads reject obsolete snapshots. Failed reads remain distinct from empty data. Artifact registration uses the canonical command fields and server-derived producer; Decisions use the existing proposal command. The scrollable conflict comparison is keyboard reachable and named. Roadmap Draft hydration builds all phase candidates before validating parents, so a child-first canonical response cannot silently detach a reference. An explicit parent removal also survives subsequent local reordering; neither operation rewrites a published revision or operational dependency.

In contract modes, Overview uses an explicitly neutral Open work count, Board uses persisted lifecycle with loaded-only counts, and selecting a Task reads signed-aware attention. List filters expose All Tasks and terminal Done/Cancelled groups; unsupported legacy-readiness links normalize to All Tasks with an explanation and corrected URL. No legacy lease projection is presented as current signed authority. Legacy mode keeps its existing readiness groups. Published plan labels and titles remain distinct from current records. These are presentation and command-adapter corrections; no backend schema, signing policy or coordination rule changes.

The [same-DOM dialog measurement](workspace/dialog-geometry.json), pinned to implementation commit `3fab26a`, covers 28 comparisons across seven native scenarios at 1440, 390, 320 and 2560 CSS px. All outer dialog/header/form/footer rectangles remain unchanged; 27 comparisons preserve every measured descendant. The single 320 px expanded Roadmap exception removes 25 px of horizontal body overflow by allowing existing fields to shrink; normal wrapping adds 45 px of scroll content height while preserving scroll position. This is an explicit overflow repair, not a claim of pixel identity for that case.

The final coordinated local run uses production inputs at `b745ec5`, including Task-specific guidance for secondary Work filters, with no later production changes. Go 1.27.2, Node 24.19.0, Playwright 1.62.1 and Chromium 149.0.7827.0 run on Linux amd64. PR checks identify the exact pushed head independently; a previous successful head is not treated as this follow-up's CI result.

| Check | Executed result |
| --- | --- |
| Native build, web tests and vet | Passed after rebuilding the embedded final assets |
| Web unit suite | 38 passed, no skips |
| Complete browser suite | 80 passed, no skips, in one final run with all three capture destinations enabled |
| Complete page/form/state journeys | Six passed within the full suite: every specialist subtab on desktop/mobile; documentary/planning/admin/technical dialogs; entry/empty/loading/error at minimum width; conflict and exact uncertain retry; real Artifact/Decision creation; existing Draft save and explicit parent removal/reordering |
| Signed specialist views | Both existing approved/changes-requested journeys pass with active-contract summary/List/Board/detail checks, protocol-specific filters and canonicalized old links; no browser domain writes from inspection |
| Same-Outcome response ordering | Two real-server regressions pass and fail against the actual earlier frontend: old Roadmap cache and old Board cursor page delivered during refreshed permissions cannot contaminate the new snapshot |
| Roadmap regression sensitivity | The actual earlier editor source fails the child-first parent assertion; the candidate preserves parent/target/after-links, published bytes and exactly two existing operational dependencies through unchanged save and explicit parent removal/reordering |
| Existing presentation safeguards | Strict hover/pressed geometry and contrast, selected states, touch insets, fixed-paper pixels, native dialog families, control borders/scroll/footer, keyboard, axe, 320 px reflow, larger text, Full HD/2K/4K proportions and missing assets remain enabled and pass |
| Documentation | 86 current-source PNGs, source/image SHA-256 manifests and local Markdown/HTML targets verified; whitespace checks pass |

The interaction harness lets Chromium complete its native hit-test scroll around the fixed header before measuring idle appearance, then retains strict idle/hover/pressed rectangle equality. Presentation audits run at the canonical viewport before documentation deliberately scrolls mobile collections. Child reference menus handle their own first Escape before the parent dialog closes. These synchronize actual browser behavior; no existing geometry, contrast, focus or axe assertion is removed.

### Browser CI follow-up

The first hosted browser run at `4378c8c` passed 77 journeys and exposed two intermittent failures after the complete local 79-test pass. Both were reproduced and corrected in production at `40499c4`.

The selected List/Board control interpolated foreground and background through an unreadable pair. Ink and paper now change together while border color keeps its existing transition; tokens, geometry and end states are unchanged. The existing interaction journey samples actual composited contrast on animation frames across both normal-motion role changes. Routing the earlier stylesheet into that same real journey fails at 2.62:1 after 44 ms; the corrected stylesheet passes the 4.5:1 requirement throughout.

Closing a Canvas form also launched a background detail restoration that could steal focus from a newly focused connection. Background restoration now preserves camera and current focus, and coherent Canvas redraws restore a surviving edge by its exact key. The new real-server barrier test holds the actual Task context response, focuses another edge, releases the old read, then activates and removes the intended advisory dependency with its observed version. Both Tasks keep their lifecycle. The earlier source fails the renderer and real-server focus assertions; the corrected source passes them and the original dependency lifecycle journey.

### Capture provenance

The 36 [main-page captures](../editorial/screenshots.json) are regenerated together. The [40 supplemental captures](workspace/screenshots.json) add specialist tabs, documentary and Roadmap surfaces, administration, entry/error states and mobile recovery. The [ten Canvas captures](screenshots.json) are also refreshed because navigation is shared. Every image is an actual browser viewport with illustrative records created through public commands on an isolated SQLite server. Mobile collection captures deliberately scroll the existing page to its working records. Only loading/error/response-order tests hold or fail requests; they do not replace domain records with mock data.

To regenerate the supplemental gallery after rebuilding the server, run from `tests/web`:

```sh
WOS_PRESENTATION_SCREENSHOTS="$PWD/../../docs/design/operational-canvas/workspace/screenshots" \
  npx playwright test presentation.spec.mjs
WOS_DESIGN_SCREENSHOTS="$PWD/../../docs/design/editorial/screenshots" \
  npx playwright test editorial.spec.mjs
```

Automated Chromium, axe, keyboard, scroll and reflow checks do not establish representative-user comprehension, manual screen-reader acceptance, other browser engines or native browser zoom certification. Those limits remain unchanged; no merge or release is performed.

## Original Canvas verification — 2026-10-16

Verified on 2026-10-16 against the implementation on `feat/visual-operational-canvas`, based on `master` at `14b6708`. All data in the screenshots is illustrative, created through real commands on an isolated SQLite server. The PR's checks report the final commit's CI result; these local observations are not a release qualification.

## Automated evidence

| Check                                   | Observed result                                                                                                                                                                                                                                                                                                                  |
| --------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Native Go web tests                     | Passed: embedded assets and command-exposure contracts                                                                                                                                                                                                                                                                           |
| Web unit suite                          | 38 passed: exact revision/scope boundaries, pagination, typed relationships, immutable models, deterministic non-overlapping layout, independent camera scaling, registry and navigation                                                                                                                                         |
| Existing browser journeys               | 41 passed, including signed approved/changes-requested handoffs, independent review, specialist views, long collections, modal families and responsive scale                                                                                                                                                                     |
| Integrated Canvas journeys              | 8 passed against the real server: default/no-Roadmap, deep links, contextual create/edit, real dependency/cycle/removal commands, Issue/Blocker separation, Evidence context linkage, partial-map pagination, off-page search, viewer behavior, conflict reconciliation, exact uncertain replay and Draft/publication/activation |
| Renderer interaction tests              | 7 passed: local drag/pan/layout, keyboard edges, pagination/search cancellation, retained query on coherent refresh, synthetic plan selection, low-zoom axe checks, suppression of discarded callback failures and loaded original paper images for every card kind                                                              |
| Context, inspector and reference picker | Passed: safe authored text, bounded omissions, original live form nodes and pending intent, focused/nested Escape ownership, cancellation of queued and late search results, multiple selection                                                                                                                                  |
| Real authorization/concurrency ordering | 4 passed: closed detail cannot reopen, obsolete Task A comparison cannot replace Task B's form, an earlier successful refresh cannot restore private content after a genuine revoked-grant 403, and a dismissed Task's delayed history error cannot overwrite the current context or notice                                      |
| Final visual regressions                | 2 affected journeys rechecked: nested Evidence provenance stays readable; a plan opens with its actual phase visible at a readable zoom                                                                                                                                                                                          |
| Original paper-card follow-up           | All 8 integrated Canvas journeys rechecked; all 11 mapped kinds plus the fallback decode their original SVGs in the renderer, source hashes and safe areas match, transparent HTML preserves the art, keyboard selection changes to original selected paper, and explicit mobile Map coexists with List                          |
| Source integrity and release queue      | 160 SVG hashes match the supplied archive; local documentation targets resolve; diff checks and the read-only release plan pass                                                                                                                                                                                                  |

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

To regenerate the ten gallery images from the real Canvas journeys:

```sh
WOS_CAPTURE_CANVAS="$(pwd)/../../docs/design/operational-canvas" npx playwright test canvas.spec.mjs
```

Captures wait for fonts and paper images to decode, preserving the actual application rendering. Board images use the canvas shell's bounds. Mobile captures show the contextual List inspector and an explicitly opened Map. No screenshot is a mockup or composite.

## Scope and remaining acceptance

Live refresh is explicit and follows successful commands. It retains local camera/placement and the chosen plan source while fencing obsolete reads; it is not a WebSocket push subscription. Local placement is not saved as domain state. Initial graph depth is three; a selected graph entity adds one bounded focal neighborhood. Known omissions, cursors, loaded counts, depth limits and exact model revision remain visible. Complete history is not loaded for every card.

No domain schema, database migration, agent runner, signing model or CLI acquisition protocol is changed. Mutation authority, optimistic versions, cycle checks and receipts remain enforced by the existing application layer. New human dependency removal retains its original `outcome:write` authorization and requires an explicit reason/version.

Representative-user usability, manual screen-reader acceptance, native browser zoom and additional browser engines remain unmeasured. Automated Chromium checks and visual inspection do not substitute for those observations. The owner previously reported that representative users and screen-reader testers are unavailable; no human acceptance is claimed here.
