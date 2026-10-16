# Editorial workspace validation

Validated locally on 2026-10-16 (America/Sao_Paulo), against a built native WOS server and isolated SQLite databases. Tooling: Go 1.27.2, Node 24.19.0, locked Playwright 1.62.1, and Chromium 154.0.8037.57 on Linux amd64.

## Executed checks

| Check | Result | Evidence and scope |
| --- | --- | --- |
| Native server build | Passed | `go build -p 4 -trimpath -o bin/wos ./packages/wos-api/cmd/wos`; embeds fonts, CSS, JS, grid, robots, and the WOS brand library |
| Browser unit tests | 14 passed | Action permissions, exact-intent handling, official copy, and six-area routing; the locale gate now includes `editorial.js` |
| Complete browser suite | 42 passed, no skips | Includes ten editorial/palette/brand/component/interaction/scale/modal tests and the 32 existing browser journeys |
| Matte modal surfaces | Passed | All four modal families at 1440 × 1000 and 390 × 844, expanded actions/instructions, technical commands, and all four administration operations; check surface roles, no pure-white element backgrounds, backdrop color, and focus inside the active modal. A separate same-DOM comparison with `e4a44de` confirms identical geometry/scroll/image attachment at 320/390/1440/2560 px; [inventory and measurement method](modal-surfaces.md) |
| Button/hover audit | Passed | Discover real controls across sign-in, every area, list/board, detail, expanded Task/reference controls, settings, and normal/technical action menus. Check nominal text contrast in idle/hover/pressed states, stable geometry, label insets, 44 px target height, selected-tab distinction, and disabled-hover stability. A touch-enabled context covers 390/320 px, keyboard focus, hover capability, and page width; [scope and inventory](interactions.md) |
| Browser CI diagnostics | Passed locally | GitHub annotations report failed expectations alongside line output. A reproduced Chromium Headless 149/151 glyph-rounding difference is handled by checking intrinsic headings/buttons through precise type, lines and containment while retaining layout-box comparisons; fonts are explicitly loaded before measuring. See [measurement rules](responsive-scale.md). That earlier CI helper refinement did not change application CSS |
| Fixed grid and sidebar | Passed | Compare visible paper pixels across 173 px of real scrolling at 1440/390 px widths; confirm content movement, stationary desktop sidebar, and document-relative reduced-motion fallback. Scroll a settings body without moving its grid. Select a long mobile Outcome by keyboard, then reach settings through a 15-Outcome desktop sidebar |
| Woobe palette | Passed | All 25 Light color tokens match the supplied README; browser assertions cover canvas, copper text/focus, charcoal primary/hover, selection, success, warning, and error roles |
| Supplied WOS brand | Passed | Three hand-drawn chain originals installed byte for byte; previous 94-file library preserved. At 1440/390/320 px verify the image-only accessible home link, original SVG proportions, no visible WOS wordmark/caption, zero-alpha openings, and no page overflow. Browser SVG/ICO/PNG assets decode and the compatibility favicon matches the new SVG. Pending reads retain one brand mark. [Current identity](brand-kit/hand-drawn-chain.md) |
| Nested forms and complete controls | Passed | At 1440 × 1000, 390 × 844, 320 × 740, landscape 740 × 390, and 768 × 1024 with a 28 px root font: measure full border/focus clearance against the scrolling body, hit testing, footer overlap, outer-dialog scrolling, control height, and page width. Exercise selects, keyboard reference selection, expanded instructions, repeatable rows, date input, permission checkbox, credential text field, persistent actions, and real Task save/detail |
| Automated accessibility | Passed within the browser suite | axe login, overview, common Task form, and reference combobox: no critical/serious violations in the configured WCAG 2/2.1/2.2 AA tag set; this is not a claim of complete conformance |
| Keyboard and reflow | Passed | Creation, reference selection, confirmation, focus restoration, mobile actions, 200% and 400% equivalent reflow |
| Editorial layout matrix | Passed | All six areas at 1920 × 1080, 1440 × 900, 1280 × 800, 768 × 1024, 390 × 844, 360 × 740, and 320 × 740; no page-level horizontal overflow |
| Responsive composition scale | Passed | The prior scale run established the unchanged 1440 CSS px reference; same-DOM, byte-identical PNG comparisons for sign-in and overview/Task forms at reference/smaller widths, including doubled root text. Measure proportional positions, dimensions, fonts, padding and column counts for sign-in/overview/form at 1920 × 1080, 2560 × 1440 and 3840 × 2160; preserve the five-column Work board, test all-area page overflow and persistent actions, and verify the cap at 5120 px. Review the actual captures; [formula, tolerances and reproduction](responsive-scale.md) |
| Fallbacks | Passed | Doubled root text size, reduced motion, blocked font/image requests, and a usable Task form without those assets; static login narrative and no-JavaScript notice checked at 1440, 390, and 320 px |
| Real workflow coverage | Passed | Outcome planning, list/board, permissions and revocation, HTTP/MCP work, evidence review, signed independent review/correction, immutable history, and reload/continuation |
| Go web/HTTP/server packages | Passed | `go test -p 4 ./packages/wos-api/web ./packages/wos-api/http ./packages/wos-api/internal/server` |
| Static analysis | Passed | `go vet -p 4` on the same three affected packages |
| Skill package | Passed in the earlier setup run | `scripts/validate_package.py`: structure, 62 tokens, 10 study crops, internal links, token output, and package hashes; the package is unchanged in this refinement |
| Git whitespace | Passed | `git diff --check` |
| Main-page gallery | 36 actual viewport captures | Regenerate every capture with the current hand-drawn chain header; retain the matte modal roles and desktop/mobile action/detail/technical views; [gallery](README.md#main-pages) and [capture/source hashes](screenshots.json) |

The gallery was generated by `editorial.spec.mjs` through public commands against a fresh database. It contains synthetic example titles and state, with no user credentials or external customer data. The chapter capture test checks loaded artwork and unexpected console/page errors. The signed-out namespace discovery's expected 401 is distinguished from an asset or application failure.

The earlier scale declaration comparison against `314f71c` is recorded in [responsive-scale.md](responsive-scale.md). The subsequent matte-modal change affects only surface/border paint and its dialog-scoped color alias in `editorial.css`; no markup, application JavaScript, dimensions, spacing, breakpoints, grid images, or backdrop rules change. The original robots, fonts, grid, palette source, and previous 94-file brand library remain unchanged. The authorized new identity adds three supplied originals and replaces the compatibility favicon with the new SVG; the current manifests verify 122 files.

The current gallery combines the approved editorial composition, Woobe Light colors, the newly supplied transparent chain logo alone in the header, and editorial product-name typography elsewhere. The [component review](component-review.md) records the top-down analysis, clipping diagnosis, fixed-paper/sidebar follow-up, and the [interaction review](interactions.md). All captures use the actual viewport so the grid, sidebar, backdrop, body, and footer match the visible interface at that scroll position. The palette README is archived byte for byte in [woobe-palette-source.md](woobe-palette-source.md), with separate [brand provenance](brand-kit/README.md). Robot, font, and earlier brand-library bytes remain unchanged; the new identity is preserved byte for byte. Build, the three targeted Go package tests, and vet were repeated after the responsive-scale, matte-modal, and new-brand refinements. The broader Go suite below records the earlier complete run; it was not repeated for these presentation refinements.

## Broader Go suite: original baseline and CI repair

`go test -p 4 -json ./...` executed the complete repository suite: **745 test/subtest outcomes passed, 75 skipped, and one failed**. The skips are service/platform-gated checks, including 53 PostgreSQL adapter cases; skipped checks are not counted as passed.

The failure is `TestActualV010DatasetMigrationsPreserveProofAndCommandFingerprints` in [legacy_contract_migration_test.go](../../../packages/wos-core/storage/sqlite/legacy_contract_migration_test.go). It also failed during environment onboarding, before this redesign. The test uses the wall clock and an original fixture whose claim receipt expires at `2026-10-15T19:16:36.279570Z`. After the seven-day expiry, the attempted replay reaches the already-completed WorkItem and returns:

```text
original 0.1 claim fingerprint changed
precondition_failed: work item is not ready: lifecycle_not_ready
```

The subsequent CI repair requested by the owner isolates the test's virtual time with `testing/synctest`, using the fixture's last command timestamp within its original receipt retention window. The historical fixture, system clock, fingerprint/proof assertions, and production code remain unchanged. The complete local Go suite now passes: **746 test/subtest outcomes passed, 75 skipped, zero failed**. See the [CI repair and verification record](../../verification-2026-10-16-ci.md) for the cause, correction, and hosted checks. The counts above remain the historical pre-repair baseline.

## Limits

- The matte-modal and final new-brand runs each passed all 42 browser tests together. The new-brand run includes zero-alpha checks for the four logo openings. The existing stale-cursor journey passed with its assertions and production reference-picker logic preserved; the prior scale run's timing failure remains a historical observation.
- Chromium/Linux automation and the listed viewport matrix do not certify other browser engines, physical devices, production deployment, or PostgreSQL parity.
- Screen-reader evaluation, representative-user review, visual approval by the owner, and full manual accessibility acceptance remain pending.
- Authored text in other scripts uses system fallbacks; the vendored Latin fonts match the official English interface.
- No merge, release publication, or cloud snapshot publication is implied by these checks.
