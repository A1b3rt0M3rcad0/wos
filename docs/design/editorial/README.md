# WOS editorial workspace

This design brings the locally supplied `chess-style` and `wos-art-style` directions into the existing WOS human workspace, using Woobe's operational color palette and the owner-supplied WOS web identity. Its organizing idea is **a shared notebook for intent, work, and proof**: precise structure, human marks, and state that survives the next handoff.

This is a presentation change to the six existing areas, authentication screen, outcome selection, collections, board, forms, settings, and entity details. HTTP/MCP contracts, persisted lifecycle rules, authorization, independent review, and explicit completion remain the source of product behavior.

## Visual system

| Element | Meaning and implementation | Small screens |
| --- | --- | --- |
| Paper | Woobe canvas `#EFEDE8`, workspace `#F5F3EF`, surface `#FFFDF9`, and elevated surface `#FFFEFC`; fine, locally served grid `#C2BCB2` | Same warm neutral hierarchy; no competing card stacks |
| Display | Instrument Serif, regular and italic; large on the login, restrained inside the operational workspace | Recomposed headlines that wrap; no horizontal type scaling |
| Utility type | DM Sans variable for controls; system monospace for metadata | Relative font sizes; readable controls and visible focus |
| Interaction | Copper `#A7441D` for links, relevant accents, and keyboard focus; selected surface `#F1DFD0` | Same roles, with explicit selected states |
| Primary actions | Charcoal `#292620`, hover `#403A32`, and on-primary text `#FFFDF9` | Same contrast and interaction states |
| Semantic state | Success `#246846`, warning `#865B13`, error `#B23834`, and info `#365F85`; every badge retains a text label | Meaning never relies on color alone |
| Brand | Supplied modular symbol in `#FF5500` beside the live name WOS in Instrument Serif; official browser icons; distinct from interaction copper | Original symbol proportions and readable editorial lettering; robot artwork remains monochrome |
| Robot vignettes | User-supplied monochrome pen-and-ink illustrations; each chapter has a relevant action | Smaller, reserved space alongside the introduction |
| Marginal notes | Brief italic annotations attached to the illustrations | Remain decorative and outside control hit areas |
| Motion | A small decorative arrival; content is visible in its final composition immediately | No parallax or pinned scrolling; reduced motion disables animation |

The application uses the grammar of the style rather than a literal chess theme. Its operational tables and forms keep their practical density. Instrument Serif is a deliberate font selection, not a claim about the unidentified font in the style reference.

### Woobe palette integration

The owner approved the editorial direction and requested the Woobe colors supplied in [this palette specification](woobe-palette-source.md). That attachment is archived unmodified; its original "proposed" status describes the input specification. The implemented Light values are centralized as `--wos-*` tokens in `packages/wos-api/web/assets/colors.css`, with compatibility aliases for existing components. The chart and its legend use the same semantic tokens as the rest of the interface.

The supplied specification cites Woobe commit `3dbcb3e1a7559f4a50ebca320e411933d46e8012`. This is the provenance reported by the attachment, not a claim that this WOS change audited the Woobe repository. Source and stylesheet hashes are recorded in [assets.json](assets.json).

The palette overrides the original gray/green color suggestions in `chess-style`, preserving its typography, geometry, composition, and illustrative language. Copper is an interaction accent, charcoal remains the primary button color, and brand orange is reserved for the mark. Robot originals are not recolored. The workspace uses the Light values; the supplied Dark values remain in the source specification as a separate future theme.

### WOS brand kit

The supplied WOS identity replaces the temporary letter/circle branding in the header, workspace selector, browser icons, and actual loading state. Following the owner's typography correction, the header combines the original orange symbol with WOS rendered as live Instrument Serif text. Chapter labels, the no-JavaScript notice, and the checkpoint message use the same editorial treatment for the product name. Browser titles and accessible names retain their plain text. All 94 supplied media files remain unchanged in the brand library, including the kit's proposed Inter wordmarks, which are references rather than active UI lockups. See [integration decisions, original kit notes, and asset provenance](brand-kit/README.md).

Typography is served from the WOS binary. Vendored fonts are from `@fontsource/instrument-serif@5.3.0` and `@fontsource-variable/dm-sans@5.3.0`, distributed under the SIL Open Font License with notices in `packages/wos-api/web/assets/fonts/`. No browser request to a font service is required, and no production JavaScript package was added.

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
| Outcome selection | `17_outcome_flagbearer.svg` | Choose the result to pursue |
| Sidebar | `18_workspace_organizer.svg` | Keep the workspace organized |

All these illustrations repeat a meaning already present in adjacent text and have empty alternatives. They do not convey status, permissions, or instructions. The other robots remain available as a library. The supplied study images in the `chess-style` skill are development references and are not embedded or reused as product artwork. [assets.json](assets.json) records the supplied artwork, font versions, dimensions, roles, and SHA-256 hashes; it does not imply an independent license grant for the supplied robot art.

## Main pages

These are captures of a real local server using an isolated illustrative database. The demo's records are created through public commands. They are examples, not product metrics, customer evidence, or release certification. Desktop captures use a 1440 × 1000 CSS px viewport with full-page output; mobile captures use 390 × 844 with full-page output.

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

### Mobile

| Overview | Work |
| --- | --- |
| ![Mobile overview](screenshots/12-mobile-overview.png) | ![Mobile task collection](screenshots/13-mobile-work.png) |

### Actual loading state

The brand symbol follows the existing pending server request. This capture holds that response in the test fixture; the application adds no deliberate waiting period.

![WOS mark during an actual pending Outcome read](screenshots/14-outcome-loading.png)

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
