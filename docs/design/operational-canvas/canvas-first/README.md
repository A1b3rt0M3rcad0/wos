# Outcome Board workspace

The Outcome Board is the main spatial workspace for an Outcome. It occupies the viewport's available working region without a left sidebar, rather than appearing below a long page header. It changes the browser presentation of existing WOS state, not the graph engine, domain, authorization, execution protocol or storage model.

**Branch:** `feat/canvas-first-workspace`, based on `master` at `cba58d6`, in draft [PR #115](https://github.com/A1b3rt0M3rcad0/wos/pull/115). Implementation source: `5fb4e8af7e90876cedc243e8bb1ed27b1fe1e57a`. The sidebar-free Outcome Board and explicit Task Board naming are implemented and locally qualified through the recorded test cohorts below. This work excludes the closed Tenant/scaling PR. Review, merging and release remain separate; no merge is authorized.

## Review of the proposed plan

The owner's diagnosis correctly separates available height from horizontal responsiveness. A fixed minimum map height cannot guarantee that the graph and its controls fit below a global header, Outcome controls, area navigation and summary. Increasing that minimum or copying a second graph renderer would preserve the cause.

The accepted direction is a viewport-based flex shell with a compact Outcome header and progressive disclosure of supporting information. Woobe's Network is a structural reference for this decision, not a dependency or a source of copied UI code. WOS retains its own warm editorial paper, typography, fixed grid, original robots, supplied post-it artwork and domain-specific actions.

The review adds three safeguards to the proposed plan:

- **Preserve reachability.** Moving six areas into Views must retain their deep links, selected-area indication, browser history and keyboard access. Compact navigation is not removal of a capability.
- **Preserve usable content.** A viewport constraint must give the map, list, context panel and inspector deliberate scroll ownership. Clipping overflow to make a screenshot fit is not acceptance.
- **Measure the actual working region.** The requested 70–80% desktop-height target is a design objective, subject to measured available space, meaningful controls and text reflow. Native browser zoom, CSS viewports and screenshot raster sizes are recorded separately.

## Shell and navigation decisions

| Concern | Decision | Preserved behavior |
| --- | --- | --- |
| Main region | Outcome Board occupies the remaining viewport height and full workspace width with flex sizing and `min-height: 0`; it has no page-height-independent 540-unit minimum | Existing pan, zoom, fit, search, expansion and rendering |
| Outcome header | Keep the Outcome name, lifecycle and primary operation compact | Explicit activation/conclusion, current permissions and mutation preconditions |
| Area navigation | Keep a direct Outcome Board control; place Overview, Plan, Work, Issues, Evidence and Activity in Views | Existing routes, browser history and focused collections |
| Sidebar | Hide it only in the Outcome Board; Views → Overview restores Outcome/Workspace switching, creation, settings and tools | The sidebar and its operations remain in all six specialist areas |
| Summary and planning | Retain the information inside Outcome context rather than permanently stacking it above the map | Loaded counts and explicit planning/operational distinctions |
| Secondary actions | Keep sharing, refresh and criteria review in Outcome tools | Existing handlers, permissions and recovery behavior |
| Inspector | Reuse the existing detail/form/action DOM; use a workspace overlay in compact or short viewports | Current entity state, local form inputs, observed versions and exact uncertain intent |
| Task Board | Retain it as a specialized Task view inside Work, explicitly named Task Board | Protocol-appropriate Task grouping; distinct from the Outcome Board |

Specialist areas keep their sidebar, normal reading and collection scroll behavior. Login, the empty workspace, administration and native dialogs retain their existing responsibilities. The viewport shell and sidebar hiding are scoped to the selected Outcome's Outcome Board view; they must not turn every page into a clipped viewport. Existing `canvas` route values, DOM IDs and module paths remain unchanged for deep-link and implementation compatibility; the visible label is Outcome Board. The former Work Board control is labelled Task Board.

### Sidebar-free follow-up

The focused source-pinned journeys verify the requested navigation boundary:

- No left sidebar or reserved sidebar column in the Outcome Board at desktop, mobile or native page zoom.
- Views → Overview restores the sidebar and its Outcome/Workspace switching, creation and authorized settings/tools; every other specialist area also keeps its existing navigation.
- Returning to Outcome Board, reloading and browser Back/Forward preserve the selected scope and contextual navigation.
- Outcome Board and Task Board have distinct visible labels and accessible names, while existing `canvas` links remain valid.
- The same four desktop sizes, two compact sizes, real 125%/150% browser zoom, inspector/forms and original editorial assets remain usable.

The four desktop sizes, both compact sizes and both real zoom settings pass these checks at `5fb4e8a`. The current 22-image gallery records that source. The [manifest](screenshots.json) separates the full regression, test-helper correction and focused recheck while preserving their identical production input hashes.

The inspector is a labelled, nonmodal region. Opening it does not inert the map or create a dialog focus trap. Its body scrolls independently; a long detail-action footer has its own bounded scroll region so it cannot consume the readable body or hide later actions. Contextual command forms retain their reachable submit footer. The inspector overlays the workspace at widths of **1400 CSS px or less**, or heights of **800 CSS px or less**, preserving the map's dimensions. Above both thresholds it shares the available width beside the map. Escape respects a nested reference picker or other control first. Closing restores focus to a still-mounted origin; closing an inline form retains the established return to details. Native dialogs remain separate and keep their backdrop, focus containment and scroll behavior.

## Operational meaning stays unchanged

Cards and connections continue to represent loaded, persisted state at the selected Outcome revision. Position, color, grouping and Roadmap order do not grant authority or create dependencies. An unloaded relationship is not a statement that none exists. Evidence supports review but does not certify a criterion; assessments and conclusions remain explicit.

Cards expose concise **Loaded** indicators for outgoing operational dependencies, direct Blocker impacts and related EvidenceLinks. These counts come from deduplicated loaded connections, not the entire Outcome. Descriptions retain hard/advisory/unknown dependency strength and supporting/contradicting/context/unknown evidence stance. An Issue does not become a Blocker, inherited blocking is not inferred, and an EvidenceLink count is not a verified-criterion count. Selecting a card still retrieves its current context and permitted actions.

The existing [map contract](../README.md#read-the-map), [dependency actions](../dependencies.md), [post-it provenance](../postits/README.md) and [signed host profiles](../../../signed-workspace-profiles.md) remain authoritative. No new artwork is created or modified for this shell. Browser controls continue to use the existing command forms, reference searches and action registry rather than a second mutation implementation.

## Acceptance matrix

The initial desktop condition is an opened Outcome in Outcome Board: the graph/list and its controls are visible without scrolling the document to reach the workspace, with no sidebar or reserved sidebar column. The focused source-pinned journeys also inspect a card, close it, return to the map, open all six specialist areas and switch Outcomes through the restored sidebar.

| Environment | Required check | Status |
| --- | --- | --- |
| 1280 × 720 CSS px | Compact HD: immediate working region, usable controls, overlay inspection and no unintended page overflow | Passed at the implementation source |
| 1366 × 768 CSS px | Notebook HD: same checks with a long Outcome title, full title in Outcome context and contextual content | Passed at the implementation source |
| 1920 × 1080 CSS px | Full HD: proportional editorial scale and a viewport-filling working region | Passed at the implementation source |
| 2560 × 1440 CSS px | QHD: same composition and controls with measured graph-height share | Passed at the implementation source |
| Native browser zoom 125% | Navigation, text reflow, map controls, inspection and return at Chromium's real page-zoom setting | Passed at the implementation source |
| Native browser zoom 150% | Same checks; preserve existing lower-width behavior as CSS viewport changes | Passed at the implementation source |
| 390 × 844 and 320 × 740 CSS px | Readable default List, explicit Map alternative, accessible navigation and reachable inspector/actions | Passed at the implementation source |
| Keyboard | Progressive disclosures, card selection, nested Escape handling, sidebar exclusion and focus return | Passed in the focused source-pinned journeys |
| Existing flows | Deep links/history, native forms, conflict comparison, exact uncertain retry and signed handoffs | All 88 journeys covered across the full run and two corrected-fixture rechecks |

The default desktop graph viewport meets the requested 70–80% height target at `5fb4e8a`. Measurements below exclude the toolbar and footer. Every focused journey checks zero document scroll and zero document overflow independently of panel/list scrolling. The graph uses the full available working width rather than reserving a sidebar column.

| CSS viewport | Graph width | Graph top–bottom | Graph height | Share of viewport height | Inspector presentation |
| --- | --- | --- | --- | --- | --- |
| 1280 × 720 | 1246 px | 163.19–668 px | 504.81 px | 70.11% | Overlay; graph dimensions preserved |
| 1366 × 768 | 1332 px | 163.19–716 px | 552.81 px | 71.98% | Overlay; graph dimensions preserved |
| 1920 × 1080 | 1875.34 px | 216.19–1011.34 px | 795.16 px | 73.63% | Beside the graph |
| 2560 × 1440 | 2501.13 px | 286.94–1349.13 px | 1062.19 px | 73.76% | Beside the graph |

On wide/tall screens, opening the docked inspector reduces the graph width while preserving its height: 795.16 px at Full HD and 1062.19 px at QHD. Overlay inspection at the smaller desktop sizes preserves both graph dimensions. The graphic region and controls remain visible without document scrolling. The 70–80% target describes the default desktop composition, not every expanded panel or zoom level.

## Verification and capture provenance

On 2026-10-18, the focused run passed **8 browser journeys, zero failures and zero skips in 34.5 seconds** against implementation source `5fb4e8a`. It covers four desktop sizes, progressive navigation/context, two compact sizes and native zoom. It verifies the absent sidebar and reserved column, retained sidebar operations in all six specialist areas, keyboard exclusion of hidden navigation, Outcome switching through Overview and distinct Outcome Board/Task Board names. It also exercises reachable detail actions and inline form submission controls, independent scrolling, Escape/focus return, browser history/reload, map camera interactions and the absence of domain writes during exploration. The 1366 px case explicitly verifies a long Outcome title and its complete text inside Outcome context. The Issue count retains its correct singular label without implying that it counts only open Issues.

The full run passed **44 unit checks** and **86 of 88 browser journeys**, with no skips, in 6.8 minutes. Two browser journeys failed because their custom test helpers continued at form close, before the post-commit refresh and inspector restoration completed. The test-only correction at `b784113280bf2866f78080e3dcc2bd5d8545fd3a` waits for the submit control to re-enable before the next navigation. It changes no production code or assertion of product behavior.

Both affected journeys then passed their focused recheck: **2 passed, zero failures and zero skips in 22.4 seconds**. All 26 production input hashes match the frozen implementation across both runs. All 88 journeys are therefore covered across the full run and successful rechecks; this is not a claim of one clean 88-journey run. The [manifest](screenshots.json) records the two cohorts, log hashes and exact test-helper correction. The eight layout/capture journeys also passed inside the full run. Hosted PR checks qualify the pushed head separately.

The native web package's targeted tests and vet checks passed. This frontend qualification does not establish backend concurrency, schema behavior, production capacity or a release.

The earlier Canvas-first source `6695b79`, its 44-unit/88-browser regression, eight-journey focused result and sidebar-bearing captures remain available in the [pinned historical qualification](https://github.com/A1b3rt0M3rcad0/wos/blob/c12177ac92a2e4a1f7ed9bd45c296cd811a939f0/docs/design/operational-canvas/canvas-first/README.md). That record preserves its exact full-run/final-copy source difference; it does not qualify the later sidebar-free view.

Earlier [Canvas verification](../validation.md), [complete-workspace coverage](../workspace/README.md) and [responsive-scale proof](../../editorial/responsive-scale.md) qualify their recorded source baselines only. Their passing results are not presented as qualification of this shell.

The 22 current captures in [the gallery](#screenshots) use an isolated SQLite instance per browser journey and illustrative records created through real public commands. Each capture journey fetches the five modified production assets from the running server and compares their SHA-256 hashes with source. The server embeds the full implementation commit in its version output. The consolidated [source and capture manifest](screenshots.json) records hashes for all 312 embedded web assets and the server binary, fixture/test hashes, browser version, CSS viewport, device-pixel ratio, native zoom, capture method and image hash. The test-only helper correction leaves that server and all 22 captures unchanged.

The capture browser is full **Chromium 154.0.8037.57**, headless on Linux amd64, driven by Playwright 1.62.1. Normal captures use device scale factor 1. Fonts and visible images finish loading before capture. The images show the actual application viewport without compositing, annotation, resizing or CSS zoom. Desktop inspector captures at 1280/1366 px show overlays; Full HD/QHD captures show docked inspection.

True browser zoom changes the browser's layout viewport. The focused journey uses full Chromium in an isolated profile and changes its default page zoom through the browser Settings API. It verifies the resulting CSS viewport, matching device-pixel ratio, unchanged outer-window size and `visualViewport.scale === 1`. A smaller Playwright viewport, `deviceScaleFactor`, CSS `zoom`, CDP page-scale emulation and image resizing do not independently prove native browser zoom. This Chromium check is not manual operating-system magnification or other-browser acceptance.

| Browser page zoom | Actual CSS viewport | Device-pixel ratio | Graphic height | Captured framebuffer |
| --- | --- | --- | --- | --- |
| 125% | 1093 × 614 | 1.25 | 352.40 CSS px | 1366 × 768 px |
| 150% | 911 × 512 | 1.5 | 258.81 CSS px | 1366 × 768 px |

The zoom captures use `Page.captureScreenshot` on the complete visible surface without a CSS-sized clip. This preserves the real zoomed framebuffer, whose raster dimensions differ from its CSS viewport. Controls, inspector actions and focus return pass at both zoom settings; the desktop graph-height target is not imposed on the smaller zoomed CSS viewport.

## Screenshots

The 22 images below record the sidebar-free Outcome Board at `5fb4e8a`. Each link opens an unedited capture; the consolidated [manifest](screenshots.json) records individual hashes and measured geometry. Historical filenames containing `canvas` remain valid after the visible name changes.

| Environment or state | Captures |
| --- | --- |
| Compact HD, 1280 × 720 | [Outcome Board](screenshots/canvas-1280x720.png) · [Overlay inspector](screenshots/canvas-1280x720-inspector.png) |
| Notebook HD, 1366 × 768, long Outcome title | [Outcome Board](screenshots/canvas-1366x768.png) · [Overlay inspector](screenshots/canvas-1366x768-inspector.png) |
| Full HD, 1920 × 1080 | [Outcome Board](screenshots/canvas-1920x1080.png) · [Docked inspector](screenshots/canvas-1920x1080-inspector.png) |
| QHD, 2560 × 1440 | [Outcome Board](screenshots/canvas-2560x1440.png) · [Docked inspector](screenshots/canvas-2560x1440-inspector.png) |
| Progressive controls, 1366 × 768 | [Views](screenshots/canvas-views.png) · [Outcome context with active planning](screenshots/canvas-outcome-context.png) |
| Compact, 320 × 740 | [Scrolled List](screenshots/canvas-320x740-list.png) · [Inspector](screenshots/canvas-320x740-inspector.png) · [Contextual form](screenshots/canvas-320x740-form.png) · [Map alternative](screenshots/canvas-320x740-map.png) |
| Mobile, 390 × 844 | [Scrolled List](screenshots/canvas-390x844-list.png) · [Inspector](screenshots/canvas-390x844-inspector.png) · [Contextual form](screenshots/canvas-390x844-form.png) · [Map alternative](screenshots/canvas-390x844-map.png) |
| Native page zoom 125% | [Outcome Board](screenshots/canvas-native-zoom-125.png) · [Inspector](screenshots/canvas-native-zoom-125-inspector.png) |
| Native page zoom 150% | [Outcome Board](screenshots/canvas-native-zoom-150.png) · [Inspector](screenshots/canvas-native-zoom-150-inspector.png) |

![Notebook Outcome Board with compact navigation](screenshots/canvas-1366x768.png)

![Task inspection without leaving the HD Outcome Board](screenshots/canvas-1280x720-inspector.png)

## Reproduce

Use the repository's documented Go/Node toolchains and installed Chromium, then rebuild the embedded application before testing:

```sh
# Repository root
go build -trimpath -o bin/wos ./packages/wos-api/cmd/wos
go build -trimpath -o bin/wosctl ./packages/wos-cli/cmd/wosctl
cd tests/web
npm ci
npm run test:unit
npx playwright test
```

For the focused viewport/inspector/zoom suite and its captures:

```sh
WOS_CAPTURE_CANVAS_FIRST="$PWD/../../docs/design/operational-canvas/canvas-first/screenshots" \
  npx playwright test canvas-workspace-layout.spec.mjs
```

Set `WOS_TEST_CHROMIUM_PATH` when using an existing full Chromium binary. The native-zoom journey needs full Chromium's Settings API; headless-shell alone does not provide it. Capture is optional and does not replace test assertions. The checked-in [manifest](screenshots.json) identifies the executed capture command, exact source and binary; regenerate the manifest when replacing its images.

Representative-user usability, manual screen-reader acceptance and other browser engines remain unqualified. Automated checks and screenshots are not WCAG certification. No merge or release is part of this task.
