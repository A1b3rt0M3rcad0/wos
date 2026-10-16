# WOS editorial workspace

This design brings the locally supplied `chess-style` and `wos-art-style` directions into the existing WOS human workspace, using Woobe's operational color palette and the owner-supplied WOS web identity. Its organizing idea is **a shared notebook for intent, work, and proof**: precise structure, human marks, and state that survives the next handoff.

This is a presentation change to the six existing areas, authentication screen, outcome selection, collections, board, forms, settings, and entity details. HTTP/MCP contracts, persisted lifecycle rules, authorization, independent review, and explicit completion remain the source of product behavior.

## Visual system

| Element | Meaning and implementation | Small screens |
| --- | --- | --- |
| Paper and grids | Woobe canvas `#EFEDE8`, workspace `#F5F3EF`, surface `#FFFDF9`, and elevated surface `#FFFEFC`; viewport-fixed 120 px graphic grid `#C2BCB2`, independent of structural columns | The same fixed paper under normal scrolling; connected sections become one ordered column |
| Sidebar | Serif workspace selector and Outcome titles, decorative numbered entries, ruled selection, registration mark, and organizer robot with an italic note | Compact workspace selector and existing collapsible, horizontally scrolling Outcome navigation; the larger vignette stays on desktop |
| Display | Instrument Serif, regular and italic; large on the login, restrained inside the operational workspace | Recomposed headlines that wrap; no horizontal type scaling |
| Utility type | DM Sans variable for controls; system monospace for metadata | Relative font sizes; readable controls and visible focus |
| Interaction | Copper `#A7441D` for links, relevant accents, and keyboard focus; selected surface `#F1DFD0` | Same roles, with explicit selected states |
| Primary actions | Charcoal `#292620`, hover `#403A32`, and on-primary text `#FFFDF9` | Same contrast and interaction states |
| Secondary actions | Padded utility buttons with a fine rule, lighter textual actions, and ruled serif action-menu rows; hover preserves contrast and selected states | At least 44 px targets; List/Board stays a framed switch; touch does not retain mouse-hover paint |
| Semantic state | Success `#246846`, warning `#865B13`, error `#B23834`, and info `#365F85`; every badge retains a text label | Meaning never relies on color alone |
| Brand | Supplied modular symbol in `#FF5500` beside the live name WOS in Instrument Serif; official browser icons; distinct from interaction copper | Original symbol proportions and readable editorial lettering; robot artwork remains monochrome |
| Robot vignettes | User-supplied monochrome pen-and-ink illustrations; each chapter has a relevant action | Smaller, reserved space alongside the introduction |
| Marginal notes | Brief italic annotations attached to the illustrations | Remain decorative and outside control hit areas |
| Forms and details | Gridded heading, small ink vignette, one scrolling body, persistent actions, complete control borders, and inset copper field focus | Related fields stack; close and save stay visible as nested content expands |
| Modal paper | Canvas headings/body, workspace secondary sections and footer, surface fields/cards, and shared border tokens; [surface inventory](modal-surfaces.md) | Identical color roles in action menus, forms, details, and settings |
| Motion | Content scrolls over a stationary grid; a small decorative arrival | Native scrolling without listeners or pinning; reduced motion makes the grid move with the document and disables animation |

The application uses the grammar of the style rather than a literal chess theme. Its operational tables and forms keep their practical density. Instrument Serif is a deliberate font selection, not a claim about the unidentified font in the style reference.

### Woobe palette integration

The owner approved the editorial direction and requested the Woobe colors supplied in [this palette specification](woobe-palette-source.md). That attachment is archived unmodified; its original "proposed" status describes the input specification. The implemented Light values are centralized as `--wos-*` tokens in `packages/wos-api/web/assets/colors.css`, with compatibility aliases for existing components. The chart and its legend use the same semantic tokens as the rest of the interface.

The supplied specification cites Woobe commit `3dbcb3e1a7559f4a50ebca320e411933d46e8012`. This is the provenance reported by the attachment, not a claim that this WOS change audited the Woobe repository. Source and stylesheet hashes are recorded in [assets.json](assets.json).

The palette overrides the original gray/green color suggestions in `chess-style`, preserving its typography, geometry, composition, and illustrative language. Copper is an interaction accent, charcoal remains the primary button color, and brand orange is reserved for the mark. Robot originals are not recolored. The workspace uses the Light values; the supplied Dark values remain in the source specification as a separate future theme.

### WOS brand kit

The header combines the original orange symbol with WOS rendered as live Instrument Serif text. Following the owner's feedback about repetition, it is the only on-page WOS mark: the workspace selector keeps its label, chapter indices identify their actual area, and pending reads use the observer robot beside the existing loading message. The no-JavaScript notice and checkpoint message retain editorial product-name typography; browser titles and accessible names remain plain text. All 94 supplied media files stay unchanged in the brand library, including the proposed Inter lockups retained as references. See [integration decisions, original kit notes, and asset provenance](brand-kit/README.md).

### Component refinement

The [page-to-control review](component-review.md) records the clipping diagnosis and the revised hierarchy. One fixed, non-interactive grid layer spans the page behind sign-in, workspace, and the translucent sidebar. Dialog backgrounds keep the same viewport alignment while their bodies scroll. Connected overview columns, registration marks, small form/settings vignettes, and ruled disclosures extend the style into working components. Command forms, settings, and detail drawers share a single scrolling content region between their heading and actions. Field labels and help use consistent spacing; related selects and permission checkboxes reflow as groups.

Typography is served from the WOS binary. Vendored fonts are from `@fontsource/instrument-serif@5.3.0` and `@fontsource-variable/dm-sans@5.3.0`, distributed under the SIL Open Font License with notices in `packages/wos-api/web/assets/fonts/`. No browser request to a font service is required, and no production JavaScript package was added.

The [button and hover review](interactions.md) maps every renderer and control family to the shared interaction rules. Area/category tabs reserve their selected rule for the selected choice; hovering another tab does not make it look selected. Utility controls, text links, primary actions, and data rows retain distinct roles, stable hit areas, legible paint, and independent keyboard focus. Cards no longer move on hover.

Above the 1440 CSS px reference width, [responsive scaling](responsive-scale.md) grows the existing composition continuously, including typography, controls, sidebar, artwork, and spacing, with a cap at 3840 px. Reference and smaller viewports retain their existing dimensions and responsive rules.

## Illustration direction and provenance

The canonical art prompt is preserved in [the local art skill](../../../.agents/skills/wos-art-style/SKILL.md): minimalist black ink, imperfect organic contours, expressive limbs, hatching, playful proportions, monochrome, and transparent backgrounds. The existing 20 robot files are preserved byte for byte from the user-supplied `wos_20_robot_svgs_fixed.zip`. These SVG containers include embedded raster artwork; they are not presented as editable vector paths.

| Placement | Robot | Narrative role |
| --- | --- | --- |
| Login / Plan | `01_planner_map.svg` | Find the route from intent to delivery |
| Overview | `14_observer_metrics.svg` | Inspect the shared state |
| Work | `05_coordinator_kanban.svg` | Coordinate bounded tasks |
| Issues | `16_blocker_sentinel.svg` | Make an impediment visible |
| Evidence | `06_researcher_books.svg` | Inspect the basis for a conclusion |
| Activity | `09_documenter_notes.svg` | Keep a durable thread of changes |
| Command form | `09_documenter_notes.svg` | Record an explicit change to shared state |
| Workspace settings | `12_key_signature_guardian.svg` | Access and identity administration |
| Pending Outcome read | `14_observer_metrics.svg` | Observe shared state while a real request is pending |
| Outcome selection | `17_outcome_flagbearer.svg` | Choose the result to pursue |
| Sidebar | `18_workspace_organizer.svg` | Keep the workspace organized |

All these illustrations repeat a meaning already present in adjacent text and have empty alternatives. They do not convey status, permissions, or instructions. The other robots remain available as a library. The supplied study images in the `chess-style` skill are development references and are not embedded or reused as product artwork. [assets.json](assets.json) records the supplied artwork, font versions, dimensions, roles, and SHA-256 hashes; it does not imply an independent license grant for the supplied robot art.

## Main pages

These 36 captures use a real local server and isolated illustrative databases. The demo's records are created through public commands. They are examples, not product metrics, customer evidence, or release certification. Main desktop viewports are 1440 × 1000 CSS px, mobile viewports are 390 × 844, and the scale comparison includes Full HD, 2K, and 4K. All captures show the actual viewport, including fixed layers, scroll regions, and persistent actions. Pages continue below the captured opening view.

### Sign in

![Editorial sign-in page](screenshots/01-login.png)

### Overview

![Overview with real progress, context, and attention indicators](screenshots/02-overview.png)

### Plan

![Plan with Objectives and roadmap navigation](screenshots/03-plan.png)

### Work

![Task collection and filters](screenshots/04-work.png)

![Operational task board](screenshots/05-work-board.png)

### Issues

![Issues and explicit blocking impacts](screenshots/06-issues.png)

### Evidence

![Evidence records](screenshots/07-evidence.png)

### Activity

![Persisted activity](screenshots/08-activity.png)

### Detail, forms, and settings

| Task detail | Task form |
| --- | --- |
| ![Task detail drawer](screenshots/09-task-detail.png) | ![Task creation form](screenshots/10-task-form.png) |

![Workspace settings](screenshots/11-settings.png)

![Expanded task instructions with persistent save action](screenshots/15-expanded-task-form.png)

![Editorial action menu](screenshots/16-action-menu.png)

All modal families use the [same matte surface roles](modal-surfaces.md), including the technical command form:

![Technical command form with matte paper and warm field surfaces](screenshots/35-technical-command.png)

### Mobile

| Overview | Work |
| --- | --- |
| ![Mobile overview](screenshots/12-mobile-overview.png) | ![Mobile task collection](screenshots/13-mobile-work.png) |

| Mobile Task form | Mobile settings |
| --- | --- |
| ![Mobile Task form with separate scroll body and actions](screenshots/17-mobile-task-form.png) | ![Mobile settings with persistent save action](screenshots/18-mobile-settings.png) |

| Mobile action menu | Mobile Task detail |
| --- | --- |
| ![Matte mobile action menu](screenshots/33-mobile-action-menu.png) | ![Matte mobile Task detail](screenshots/34-mobile-task-detail.png) |

![Mobile technical command form](screenshots/36-mobile-technical-command.png)

### Actual loading state

The observer vignette accompanies the existing pending server request. This capture holds that response in the test fixture; the application adds no deliberate waiting period.

![Observer robot during an actual pending Outcome read](screenshots/14-outcome-loading.png)

### Fixed paper and editorial navigation

These two viewport captures show the same workspace before and after scrolling 173 px. The content moves while the grid lines and desktop sidebar keep their viewport positions. The extra Outcomes are illustrative records, not invented product features. With reduced motion, the grid follows the document instead.

| Before scrolling | After scrolling |
| --- | --- |
| ![Workspace at the top with three Outcomes and the editorial sidebar](screenshots/19-fixed-paper-top.png) | ![Workspace scrolled over the same stationary grid](screenshots/20-fixed-paper-scrolled.png) |

![Expanded mobile Outcome navigation with a long title brought into view](screenshots/21-mobile-outcome-navigation.png)

### Button interaction states

| Hover another area while Work remains selected | Hover Board while List remains selected |
| --- | --- |
| ![Padded Issues hover without the selected Work underline](screenshots/22-area-tab-hover.png) | ![Readable unselected Board hover beside selected List](screenshots/23-view-switch-hover.png) |

| Stable board card hover | Expanded form utility hover |
| --- | --- |
| ![Board card with a restrained border and no movement](screenshots/24-board-card-hover.png) | ![Padded add-instructions action within the expanded Task form](screenshots/25-form-controls-hover.png) |

![Keyboard focus inside the mobile tab strip with no persistent mouse-hover paint](screenshots/26-mobile-tabs-focus.png)

### Responsive scale

The same composition at the reference width and the three requested desktop sizes; [sizing rules and verification](responsive-scale.md). The 16:9 captures expose the same document region at larger dimensions, with less vertical content than the taller reference.

| Reference · 1440 × 1000 | Full HD · 1920 × 1080 |
| --- | --- |
| ![Unchanged reference composition](screenshots/27-scale-reference.png) | ![Proportional Full HD composition](screenshots/28-scale-full-hd.png) |

| 2K · 2560 × 1440 | 4K · 3840 × 2160 |
| --- | --- |
| ![Proportional 2K composition](screenshots/29-scale-2k.png) | ![Controlled 4K composition](screenshots/30-scale-4k.png) |

| 2K Task form | 2K Work board |
| --- | --- |
| ![Scaled fields and persistent form actions](screenshots/31-scale-2k-task-form.png) | ![Scaled Work columns and cards](screenshots/32-scale-2k-work-board.png) |

## Reproduce and validate

From the repository root, activate the Go toolchain documented for your environment, then run:

```sh
go build -trimpath -o bin/wos ./packages/wos-api/cmd/wos
go test ./packages/wos-api/web ./packages/wos-api/http ./packages/wos-api/internal/server
go vet ./packages/wos-api/web ./packages/wos-api/http ./packages/wos-api/internal/server
```

From `tests/web`:

```sh
npm ci
npm test
```

The browser suite supports an installed Chromium via `WOS_TEST_CHROMIUM_PATH`. In a sandbox with a read-only home, set writable `XDG_CACHE_HOME` and npm cache paths. To regenerate this gallery after a build:

```sh
# Working directory: tests/web. Defaults to Playwright's installed browser.
WOS_DESIGN_SCREENSHOTS="$PWD/../../docs/design/editorial/screenshots" \
  npx playwright test editorial.spec.mjs
```

The design tests exercise each chapter, illustrations, actual records, empty states through the existing suite, usable forms, and missing font/image fallbacks. The layout matrix covers 1920 × 1080, 1440 × 900, 1280 × 800, 768 × 1024, 390 × 844, 360 × 740, and 320 × 740; every area is checked for page overflow. Separate checks cover doubled root text size, reduced motion, keyboard navigation, and automated accessibility. JavaScript remains required for the workspace application; the static sign-in narrative and a clear no-JavaScript notice remain available when it is disabled.

See [validation.md](validation.md) for executed checks and their limits. Manual screen-reader, representative-user, other browser engines, and production deployment acceptance remain separate.
