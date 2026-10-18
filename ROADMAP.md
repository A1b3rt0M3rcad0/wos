# WOS Roadmap

**Status document:** live and mandatory  
**Canonical design:** `docs/WOS_Design_Arquitetura_Planejamento_Atualizado.md`  
**Last reviewed:** 2026-10-18
**Current target:** Stable 1.0 hardening; public npm package wosctl
**Current wave:** Compact login, stable Board inspection and visible camera controls; release 1.1.0 and human acceptance remain separately tracked

**Distribution extension:** npm service/skills and automated releases requested by the owner; see D1–D4 below.

**Workspace presentation follow-up:** owner-requested editorial redesign using the supplied `chess-style`, robot illustrations, `wos-art-style`, Woobe operational palette, and WOS web brand kit. This extends the existing human workspace UX program without changing domain semantics or the release target. Design, page captures, and reproducible checks are recorded in [the design guide](docs/design/editorial/README.md) and [validation](docs/design/editorial/validation.md). Automated validation and independent human acceptance are tracked separately below.

This file records the real implementation state of WOS. It must be kept synchronized with the repository by every agent that changes planned work.

The architecture document describes what should exist. This ROADMAP records what actually exists.

## Status legend

- ✅ **Done** — acceptance criteria satisfied and relevant verification passes.
- 🚧 **In progress** — implementation has started but acceptance criteria are not yet complete.
- ⬜ **Planned** — not started in a meaningful implementation sense.
- ⛔ **Blocked** — cannot proceed because of an explicit dependency or decision.
- 🧭 **Decision required** — an administrative/architectural decision is still open.

## Repository baseline

| Item | Status | Notes |
| --- | --- | --- |
| GitHub repository | ✅ | `A1b3rt0M3rcad0/wos`, default branch `master` |
| Canonical architecture specification | ✅ | Stored under `docs/`; revision 2 |
| Public README | ✅ | Executable features, draft PR status and remaining acceptance explicitly separated |
| Go module | ✅ | `github.com/A1b3rt0M3rcad0/wos` |
| Repository ignore rules | ✅ | Go, local DB/data, environment/secrets, IDE files |
| AGENTS development contract | ✅ | Root `AGENTS.md` |
| Live implementation roadmap | ✅ | Root `ROADMAP.md` |
| Product package topology | ✅ | `packages/wos-core` + `packages/wos-api`; ADR-011 |
| Current source license | ✅ | Owner-selected WOS Sustainable Use License 1.0 and separate brand policy; earlier Apache grants and third-party rights retained. See [licensing](docs/licensing.md). |

## Milestones

| Milestone | Status | Definition |
| --- | --- | --- |
| M1 — Local vertical slice | ✅ | Core + SQLite + HTTP with Outcome, Objective, WorkItem and criteria |
| M2 — Full coordination | ✅ | Dependencies, Blockers, Issues, leases and documentary records |
| M3 — Planning and continuity | ✅ | Roadmaps, snapshot/graph/timeline and MCP |
| M4 — Complete standalone | ✅ | PostgreSQL parity, triggers/outbox, auth and packaging |
| Release 0.1 validation candidate | ✅ | Thirteen canonical release gates have executable evidence; release tag publication is separate |

---

# Implementation waves

## Compact entry and Board inspection — 2026-10-18

**Status: In progress in [PR #115](https://github.com/A1b3rt0M3rcad0/wos/pull/115).**
The owner reports avoidable login scrolling on smaller screens and a contextual
inspector that disrupts the Outcome Board header. The owner also requests clear
centering/zoom controls, using Woobe's Network as an interaction reference.

- [ ] Fit the default login composition in compact desktop/mobile viewports
  without clipping fields, shrinking touch targets or disabling natural reflow
  when local access, enlarged text or an on-screen keyboard needs more room.
- [ ] Keep the Outcome/Board header and toolbar stable during inspection;
  constrain the existing detail/form/action surfaces to the graphic region.
- [ ] Expose persistent map camera controls, preserving keyboard, pan, selected
  context and the distinction between centering, fit and local auto-layout.
- [ ] Verify normal/compact/zoomed entry, inspector focus/forms and camera
  interactions, then update the captures and source-bound qualification.

The existing editorial tokens, original art, commands, authority and recovery
contracts remain unchanged. Merge was authorized by the owner; additional UI
corrections precede integration and exact-source checks. Version 1.1.0 remains
prepared and unpublished while this PR is open.

## Coordinated release and README versions — 2026-10-18

**Status: Prepared and locally verified in draft [PR #115](https://github.com/A1b3rt0M3rcad0/wos/pull/115), no merge.**
The owner requests preparing the next release in this PR and keeping the npm
package and README synchronized. Initial read-only GitHub and npm checks
confirmed that v1.0.0 was already published while the README still advertised
v0.4.2. The queued Outcome Board feature requires a minor release, now prepared
as 1.1.0.

- [x] Prepare VERSION, npm metadata, both OpenAPI versions, manifest, changelog
  and release notes together, consuming the reviewed change queue.
- [x] Generate the README source-version section during release preparation and
  use live published-version badges instead of manual release/npm numbers.
- [x] Include generated README metadata in automatic release preparation and
  reject version drift through release/compatibility checks.
- [x] Verify release preparation, immutable compatibility and actual package
  contents before updating the draft PR. Publication follows integration and
  existing exact-source release gates; no merge or registry publication here.

The 32 package/Skill/distribution/release contract tests pass. After adding
independent assertions for the actual README version and release-notes target,
all 16 release/compatibility checks pass again. Preparation produces 1.1.0 and
consumes the queue; a repeated plan returns null.

Actual 1.1.0 artifacts built from committed source
`21d0a6f8e4235ce00cb8c5541050bee7bd24eaa3` pass the distribution verifier:
offline npm installation, matching service/client/Skill versions, source and
artifact hashes, original legal files, native Skill operations, signed execution
and independent review/recovery, HTTP/MCP/UI readiness, shutdown and corruption
rejection. The local runner uses writable npm/XDG cache paths under `/tmp`.
Windows archives are cross-built and checked structurally here; native Windows
execution and the remaining exact-source release gates belong to hosted CI.
No registry upload, tag creation, release publication or merge was performed.

## Contextual Outcome actions and compact navigation — 2026-10-18

**Status: Implemented and locally verified in draft [PR #115](https://github.com/A1b3rt0M3rcad0/wos/pull/115), no merge.**
The owner requests fewer repeated operations and more useful space in the
specialist sidebar. Outcome-wide tools and the general action menu belong in
Outcome Board; individual records and collections retain their own actions.

- [x] Restrict Outcome tools and Choose an action to Outcome Board; preserve
  same-view Refresh, Overview review and existing native entity/collection forms.
- [x] Use one visible Outcome search field with automatic reads, immediate Enter
  and filters, cancellation, query-bound pagination and authorization handling.
- [x] Compact the sidebar's spacing and original robot/caption while preserving
  selection, keyboard focus, mobile navigation and functional targets.
- [x] Validate the view matrix, discovery ordering/pagination, native forms,
  permissions and recovery; refresh source-bound Board and sidebar captures.

Production source `c3c3139f5fb1d71bebdc80c14de5cc915c23de98` passes 44 unit
checks and 91 of 92 browser journeys in the full run. Its only failure was a
modal-palette test selecting a Work row after navigating to Overview. Test-only
correction `62cbc03a917a6e4a501f14a69b3184cfb2d510b8` explicitly returns to Work;
the affected desktop/mobile journey passes its focused recheck with all existing
assertions retained. All 92 journeys are covered across those runs, with zero
skips and identical production inputs; this is not one clean 92-case execution.

All eight viewport journeys and all four new discovery journeys pass within the
full run. The [26 source-bound captures](docs/design/operational-canvas/canvas-first/README.md#screenshots)
include desktop Overview/Work and compact Overview sidebars alongside the Board,
inspector/forms and native 125%/150% browser zoom. Native web tests and vet pass.
The [manifest](docs/design/operational-canvas/canvas-first/screenshots.json)
records exact cohorts, source/binary/image hashes and six served assets checked
against source per capture. Hosted checks qualify the pushed head independently.

No domain, backend, protocol, schema or release change is included. The earlier
qualifications below remain historical. Representative-user and manual
screen-reader acceptance remain separate.

## Outcome Board workspace follow-up — 2026-10-18

**Status: Implemented and locally verified in draft [PR #115](https://github.com/A1b3rt0M3rcad0/wos/pull/115), no merge.**
The owner identifies the spatial Board as the working environment itself. The
left sidebar belongs in Overview and the other specialist views rather than
inside this working region. This supersedes retaining a sidebar in the initial
Canvas-first presentation below.

- [x] Hide the sidebar only for the active Outcome Board and give its space to
  the working region; retain Overview/specialist navigation and Outcome switching.
- [x] Rename visible and accessible Canvas controls to Outcome Board; distinguish
  the specialized Task Board in Work and preserve existing internal routes.
- [x] Verify focus when navigation becomes hidden, desktop/mobile/native zoom,
  inherited journeys, hosted failure corrections and refreshed source-bound captures.

Current operation and evidence are maintained in the
[workspace review](docs/design/operational-canvas/canvas-first/README.md).
The earlier source-bound qualification below is historical, not proof of this
follow-up. No domain, protocol, backend, storage or release change is included.

Production source `5fb4e8af7e90876cedc243e8bb1ed27b1fe1e57a` passes all eight
viewport journeys, including hidden-sidebar focus exclusion, restoration in all
six specialist areas and real Outcome switching through Overview. The full run
passes 44 unit checks and 86 of 88 browser journeys. Its two custom journey
helpers returned before post-commit refresh completed; test-only correction
`b784113280bf2866f78080e3dcc2bd5d8545fd3a` waits for the submit control to
re-enable, and both focused rechecks pass. All 26 production inputs are unchanged
between runs. This covers all 88 journeys across the regression and recheck;
it is not a claim of a single clean 88-case execution. Native web tests and vet
pass. The previous hosted plan-disclosure and proportional-navigation failures
also pass locally with explicit completion and geometry checks. The
[22 captures from this source](https://github.com/A1b3rt0M3rcad0/wos/blob/15593b763a820d97026db2354b8013424798c2e9/docs/design/operational-canvas/canvas-first/README.md#screenshots)
qualify the production source, while hosted checks qualify the pushed head
independently. Representative-user and manual screen-reader acceptance remain
separate.

## Canvas-first workspace — 2026-10-18

**Status: Implemented and locally verified; draft review, no merge.** The work is
based on `master` at `cba58d6`. This presentation follow-up is independent from the
closed scalable-installation PR #114 and introduces no backend, schema, CLI or
execution-protocol changes. The [plan review and acceptance record](docs/design/operational-canvas/canvas-first/README.md)
identify the implementation and validation boundaries.

- [x] Make the active Canvas occupy the viewport remainder, with explicit internal
  scroll ownership instead of a fixed minimum map height.
- [x] Replace the horizontal area row with Canvas and compact Views navigation;
  retain specialist views, deep links and the distinct Work Board.
- [x] Keep Outcome identity/status and priority actions compact; expose existing
  summary, planning and secondary tools progressively.
- [x] Preserve the existing inspector/form nodes, exact pending intent, keyboard
  return and camera; overlay inspection on compact screens without document scroll.
- [x] Show actual loaded dependency, Blocker and EvidenceLink indicators without
  treating Evidence, note position or visual connections as criterion verification.
- [x] Validate desktop HD/Full HD/QHD, mobile reflow, real 125%/150% browser zoom,
  navigation/permissions/races and source-bound screenshots before qualification.

The supplied plan's 70–80% map-height target is measured against the actual CSS
viewport; controls, reading and keyboard access take precedence at reduced sizes.
The existing editorial palette, typography, fixed grid, robot and post-it assets
remain authoritative. Automated checks and representative-user acceptance stay
distinct; earlier screenshots do not qualify this structural change.

The complete local regression passes 44 unit checks and 88 browser journeys with
no skips. Its recorded production hashes precede only the final Issues count
wording/pluralization correction. Eight final-source layout journeys pass at
`6695b7984d1e3447139c71dff15fa2ba61948b90`, including that correction, a long
Outcome name, mobile inspector/forms, and native 125%/150% browser zoom. The
actual graph occupies 70.11%, 71.98%, 73.63% and 73.76% of the four requested
desktop viewports. Native web race tests and vet pass. The 22 unannotated captures,
served-asset checks and exact input/image hashes are recorded in the
[original capture manifest](https://github.com/A1b3rt0M3rcad0/wos/blob/c12177ac92a2e4a1f7ed9bd45c296cd811a939f0/docs/design/operational-canvas/canvas-first/screenshots.json).
Hosted checks qualify the pushed PR head independently. Representative-user,
manual screen-reader and other-browser acceptance remain unclaimed.

## Visual operational workspace — 2026-10-16

**Status:** merged with the owner's authorization as [PR #110](https://github.com/A1b3rt0M3rcad0/wos/pull/110) into `master` at `c321b95` on 2026-10-17. The implementation was verified on `feat/visual-operational-canvas`, based on `master` at `14b6708`. This is a presentation extension of the existing human workspace and centralized-agent APIs, not a new backend wave or a release qualification.

The implementation brings existing Outcome context into a default canvas with real Objective hierarchy, Task ownership, typed operational connections, optional Roadmap overlays and contextual use of existing detail/forms/actions. Native ES modules retain the approved editorial identity, existing responsive scale and original supplied post-it artwork. Domain models, signed authority, independent review, authorization, CAS and exact-intent recovery remain authoritative. See [architecture, behavior and decisions](docs/design/operational-canvas/README.md), [workspace operation](docs/workspace-ux.md) and [source-asset provenance](docs/design/operational-canvas/postits/README.md).

Automated acceptance is recorded in [verification](docs/design/operational-canvas/validation.md); PR review and independent human acceptance remain separate:

- [x] Default canvas and contextual inspector preserve the existing actions/forms and keep purpose-based areas, Task list/Board and keyboard/mobile list alternatives available.
- [x] Live scoped graph, continuity pages and selected context are revision-consistent, bounded per request and explicit about partial/depth-limited results; authorized search can find unloaded records.
- [x] Hierarchy/ownership, hard/advisory/unknown dependencies, Issues/Blockers and Evidence/criterion assessments retain their distinct semantics and permitted mutation paths.
- [x] Optional Draft and published Roadmap overlays preserve planning hierarchy/order separately from operations; viewing, publication and activation remain distinct.
- [x] Existing palette, typography, fixed paper grid, original brand/robots, post-it proportions/safe areas and responsive scale remain consistent across desktop and compact layouts.
- [x] Every Map card uses an original supplied post-it image with live accessible content; paper shapes cover operational and planning records, selected silhouettes have no extra rectangular frame, and compact Map remains available beside the readable List alternative.
- [x] Real-server small/large-scope, permission, stale-read, conflict/recovery, signed-handoff, keyboard/focus and mobile checks complete; integrated screenshots and precise verification limits are recorded.

Earlier workspace and centralized-agent verification records below describe their own baselines. They do not establish acceptance of this new presentation. Independent human usability and screen-reader validation remain separate from automated checks and screenshots.

### Complete workspace page follow-up — 2026-10-17

The owner requested completion of every remaining page in the same branch and [PR #110](https://github.com/A1b3rt0M3rcad0/wos/pull/110), initially without merging, and subsequently authorized its merge. The [coverage matrix and gallery](docs/design/operational-canvas/workspace/README.md) include all specialist tabs, authentication/navigation, entry states, details, native forms, settings and technical tools. Source and executed results belong to [verification](docs/design/operational-canvas/validation.md#complete-workspace-follow-up--2026-10-17).

- [x] Apply the shared editorial hierarchy and matte paper to every remaining area; preserve structural grids, original brand/robots/post-its, native surfaces, fixed paper and the existing responsive scale.
- [x] Make collection metadata and filters appropriate to the actual record; bound Roadmap discovery, search real loaded Activity/Relations/Active Roadmap fields, separate historical plan labels from live records, and retain explicit empty/loading/error recovery.
- [x] Preserve signed-aware current Task authority in specialist views; use neutral summary counts and persisted-lifecycle Board columns in contract modes without extrapolating legacy lease expiry.
- [x] Correct native Artifact/Decision creation adapters and keyboard access to conflict comparison; preserve canonical commands, authenticated authorship, permissions, CAS, receipts and exact uncertain intent.
- [x] Verify child-first Roadmap Draft hydration and an explicit parent removal through local reordering; keep published revisions and operational dependencies unchanged.
- [x] Reproduce and repair the first hosted browser run's transient selected-button contrast and delayed Canvas form-restoration focus; retain strict frame contrast and exact edge focus through coherent redraws and actual authorized removal.
- [x] Complete the final coordinated validation: 80 browser journeys and 38 unit checks pass without skips, native web tests/vet pass, and all 86 documentation captures are refreshed against production inputs at `b745ec5`, including Task-specific guidance in secondary Work filters. Record per-file hashes and viewport metadata in the gallery manifests.

Exact final-head hosted checks are reported by [PR checks](https://github.com/A1b3rt0M3rcad0/wos/pull/110/checks); earlier green commits are separate evidence. Representative-user review, manual screen-reader use and other browser engines remain unqualified. This follow-up changes no backend schema, CLI/Skill execution protocol, release version or release status.

### Product explanation and browser tour — 2026-10-17

The owner requested a more useful README demonstration and a clearer explanation for developers and Outcome-as-a-Service (OaaS) products. This documentation follow-up replaces the README's terminal animation with real workspace navigation, connects the result/work/delivery/review model to a concrete software example, and presents OaaS as an integration pattern with external execution, customer management and billing. Detailed CLI provisioning, signing and recovery remain in their existing guides. The published v0.4.2 and current-source distinction remains explicit; no domain, runtime or commercial-service capability is added.

- [x] Verify the seven-state real navigation tour (28 seconds), static alternative and capture provenance; review README claims, local links and preserved incoming anchors; inspect GFM output in Chromium at 1280, 390 and 320 CSS pixels with details closed/open and all images decoded. Record executed scope and limits in [README validation](docs/readme-validation.md).

## Owner-authorized release preparation — 2026-10-17

The owner authorized merging the README clarification and initiating the next coordinated release. The release tool consumes the four queued changes and prepares version **1.0.0**: the licensing compatibility break requires a major bump under `.changes/README.md`; the agent workflow, operational hardening and visual workspace additions are included. README release status explicitly distinguishes preparation from public availability. Publication must use the existing source-gated workflow and immutable npm, GHCR and GitHub artifacts. This version number does not complete the separate stable-1.0 acceptance program; published baseline remains v0.4.2 until verified publication succeeds.

Publication follow-up verified on 2026-10-18: GitHub Release v1.0.0 was published
on 2026-10-17 at source `cba58d6d609377df141d1369dccaaaf3b3e331e3`, and public
npm `wosctl@latest` resolves to 1.0.0. The preparation status above is historical;
the next source release is tracked in the coordinated-release wave above.

## Source-available licensing and brand — 2026-10-17

The owner requests production and application use with Woobe-style restrictions against commercial platform supply under the original brand or a replacement brand. [ADR 037](docs/adr/0037-source-available-licensing-and-brand-boundaries.md) adopts the WOS Sustainable Use License 1.0 and [trademark policy](TRADEMARKS.md). Current first-party terms permit internal production, commercial applications/OaaS, private modifications and genuine technical services, while excluding resale, paid hosted/managed platform access, OEM and commercial white label. Qualified legal review is separate from technical consistency checks.

- [x] Record the owner decision, software/brand boundaries, examples and contribution authority; preserve the exact prior Apache text and all upstream licenses/notices.
- [x] Propagate terms and notices through npm, native archives, container packaging and offline Skills, with meaningful integrity, installed-file and preservation checks.
- [x] Pass 30 focused Node tests, race checks for both affected Go packages, actual archive/npm/20-installed-Skill byte comparisons and the complete packaged smoke verifier. Check current-document links and incoming README anchors, and inspect GFM at 1280/390/320 CSS pixels. Record working-tree provenance, unexecuted container/Windows runtime limits and separate exact-head qualification in [licensing validation](docs/licensing-validation.md).

Earlier Apache grants remain effective for their material, including portions retained in later builds. This change does not revoke past rights, relicense other owners' material or rewrite immutable releases. Dated Apache-selection/qualification statements below record their historical baselines and are superseded for new material by this owner decision. The major release-queue entry records a licensing compatibility break; it does not prepare, publish or certify product 1.0, and no release version is changed here.

## Centralized agent operations — 2026-10-16

**Status:** merged by the maintainer as [PR #109](https://github.com/A1b3rt0M3rcad0/wos/pull/109) into `master` at `14b6708` on 2026-10-16; release publication remains separate. The owner requests coordinated CLI, five Skills, backend, README and documentation changes. The supplied implementation brief and strategic summary define the scope; the detailed plan is maintained under `docs/central-agent-workflow/`.

- [x] Initialize one local installation data directory, persistent active profiles, explicit per-process profile selection and distinct protected destinations, without project `.wos` discovery.
- [x] Migrate existing protected material safely with drained writers, preserved source/issued bytes/pending intentions, collision checks and interruption recovery.
- [x] Provide bounded discovery, unambiguous short references, optional unique-contract selection, CLI-managed progress/results/evidence/artifacts and preflight/finish without agent YAML editing.
- [x] Persist authorized chronological activity and documentary references independently of local files; expose scoped paged history and compact known-obligation projections through HTTP, MCP and SDK.
- [x] Update all five Skills for CLI-only execution, bounded continuation, multiagent roles, true independent review and exact-intent recovery.
- [x] Update public and operational guides, source/release boundaries, executable examples and a detailed implementation-plan ZIP.
- [x] Complete local migration, authority, concurrency, recovery, Memory/SQLite/real PostgreSQL, transport/client parity and end-to-end validation; record native/platform and optional-load limits separately in [validation](docs/central-agent-workflow/validation.md). Full normal/race suites each passed 1,501 test/subtest entries; browser checks passed 42 journeys and 14 unit checks. Hosted exact-PR checks remain reported by CI.

Architecture decisions: [central installation](docs/adr/0035-central-cli-installation.md) and [durable execution activity](docs/adr/0036-durable-execution-activity.md). WOS remains passive state and coordination; runtimes execute and schedule agents.

CI follow-up: the released-v0.4.2 upgrade gate compares every installed migration's version, name and checksum with the complete candidate source chain, replacing its stale candidate-schema-29 assertion. The original schema-26 fixture, historical rows, signed proofs, issuer checks and old-binary refusal remain mandatory. The Windows migration fixture closes its inspection handle before directory publication, preserving its assertions; native test diagnostics are retained separately from successful platform acceptance proofs. Exact-head hosted qualification remains reported by the PR checks.


## Editorial workspace presentation — 2026-10-15

**Scope:** the existing six-area human workspace, sign-in, outcome selection, task list and board, entity details, forms, and settings. This presentation follow-up does not mark an additional backend wave or a stable release complete.

- [x] Apply continuous paper surfaces, editorial typography, fine geometry, and supplied monochrome robot vignettes; follow the owner's approved direction with Woobe's warm neutrals, charcoal, copper, and semantic colors from the supplied palette specification.
- [x] Preserve existing scoped actions, explicit lifecycle changes, independent review, URL navigation, and API contracts.
- [x] Serve licensed font assets locally with system fallbacks; preserve the supplied skill packages and robot artwork.
- [x] Document the visual system, art direction, asset provenance, and page gallery generated from an isolated real-server dataset.
- [x] Apply the supplied WOS web identity to the header and browser/touch icons; preserve the 94 original media assets and master silhouette in all 80 SVGs. The original symbol accompanies live Instrument Serif lettering. Following the repetition feedback, keep one on-page brand mark, functional area/workspace labels, and an observer robot during pending reads.
- [x] Review page-to-control composition and correct shared button borders, continuous grids, connected overview sections, dialog sizing/scrolling, select/focus clipping, grouped fields, nested editors, and persistent actions; document the [component review](docs/design/editorial/component-review.md). Preserve the approved direction and Woobe palette.
- [x] Anchor the shared 120 px paper grid to the viewport behind scrolling content and dialogs, with a document-relative reduced-motion fallback. Refine the sidebar with serif workspace/Outcome names, decorative numbering, ruled states, and an organizer vignette; retain mobile collapse, keyboard selection, native controls, and long-list access to settings.
- [x] Review all button renderers and interaction families: unify idle/hover/pressed paint, preserve selected-tab distinction and text contrast, add label insets and touch-sized targets, prevent disabled hover and persistent touch-hover paint, and remove board-card movement. Record the [interaction inventory and verification scope](docs/design/editorial/interactions.md).
- [x] Complete final presentation validation: 40 browser tests and 14 browser unit tests passed, including button/hover contrast and geometry, touch navigation, fixed-paper pixel comparisons, sidebar navigation, palette, brand, and nested-form geometry/interactions; targeted Go tests and vet passed. Refresh and extend the gallery to 26 viewport captures. Preserve the original Go baseline and its service/platform skips in [validation](docs/design/editorial/validation.md).
- [x] Repair the historical migration test's wall-clock dependency using an isolated virtual clock and the fixture's original receipt timestamps; preserve fixture bytes, fingerprint/proof assertions, and production retention semantics. The complete local Go suite now passes with 746 passing test/subtest outcomes and 75 explicit skips; hosted verification is tracked in [the CI repair record](docs/verification-2026-10-16-ci.md).
- [x] Preserve the approved 1440 CSS px composition and smaller-screen behavior while scaling dimensions continuously above the reference, capped at the 3840/1440 ratio. Verify pixel identity locally for reference/smaller pages and forms, proportional login/overview/form geometry, unchanged board columns, all-area overflow, and the cap; add Full HD/2K/4K captures and [scale documentation](docs/design/editorial/responsive-scale.md).
- [x] Unify all four native modal families with matte canvas, workspace secondary areas/footers, surface fields/cards, and existing border tokens. Preserve layout, scrolling, focus, backdrop, and editorial grid; verify 16 same-DOM geometry comparisons, desktop/mobile modal states, the complete 42-browser/14-unit suite, and targeted Go tests/vet. Refresh nine modal captures and add four views in the [surface inventory](docs/design/editorial/modal-surfaces.md).
- [x] Apply the subsequently supplied hand-drawn chain SVG/PNG/ICO identity and show only its symbol in the header home link, preserving its accessible name. Verify original proportions, zero-alpha internal openings, desktop/mobile reflow, and browser-icon decoding; preserve the previous 94-file library and refresh all 36 header-bearing captures. Record the [current brand and provenance](docs/design/editorial/brand-kit/hand-drawn-chain.md).
- [x] Rewrite the public README in English around purpose, shared state, setup, and the actual human/agent workflow, with a larger image-only brand introduction, the owner's original queue illustration, and a task-oriented documentation index. Translate maintained guides and local art instructions; preserve original specifications and historical evidence. Remove two superseded October 5 reports and 19 old UI captures while preserving historical proof through pinned Git links. Verify local Markdown references and the GFM preview at desktop/mobile widths; the earlier setup run also checked native CLI help/dry-run and Compose configuration.
- [x] Replace only the planner illustration with the owner's supplied blank-map SVG at its existing path and 247 × 275 dimensions. Preserve the supplied bytes, document the authorized exception in the asset manifest, and refresh login/Plan captures. Repeat the build, Go web tests, 14 browser unit checks, and four affected desktop/mobile/responsive browser journeys without changing layout or behavior.
- [x] Synchronize the presentation branch with current master and reconcile the README installation/release updates while retaining the English illustrated guide and image-only introduction. Keep the deterministic virtual-clock migration regression: upstream expiry relocation conflicts with that historical window, so preserve original fixture rows and use the existing isolated clock. Verify local documentation links, browser unit checks, the focused migration regression, and the native build; PR integration remains a separate owner action.
- [x] Refine the English README for first-time users: a 240 px image-only logo, 360 px original robot metaphor, real Work/Evidence/login captures, concrete scenarios and concepts, explicit platform/release boundaries, and a real 24-second unsigned local demo with a rendered GIF and provenance. Add getting-started/troubleshooting, contributor and security guides; verify the public 0.4.2 container through readiness and browser sign-in, execute the isolated lifecycle, check links and asset hashes, and inspect GFM previews at 1280/390/320 px. The demonstration uses one trusted operator and does not claim signed independent review. See [README validation](docs/readme-validation.md).
- [ ] Independent human review, screen-reader validation, and other-browser acceptance; not implied by automated checks or screenshots.

Publication and merging remain separate owner actions. No merge is authorized by this design task.

## Wave 01 — Public Core Foundation

**Status:** ✅ Done

**Goal:** establish a Go project usable as a public library and standalone server, with verifiable boundaries.

### Completed

- [x] Create standalone `wos` repository.
- [x] Define Go module path.
- [x] Add public architecture README.
- [x] Store canonical architecture and implementation plan.
- [x] Add repository `.gitignore`.
- [x] Add root `AGENTS.md`.
- [x] Add live `ROADMAP.md`.

### Implementation checklist

- [x] Establish `packages/wos-core/domain`.
- [x] Establish `packages/wos-core/ports`.
- [x] Add foundational ID / UUIDv7 value object.
- [x] Add `Scope`.
- [x] Add `EntityRef`.
- [x] Add `ActorRef`.
- [x] Add aggregate `Version` and Outcome revision types.
- [x] Add `Clock` port.
- [x] Add `IDGenerator` port.
- [x] Add stable domain/application error model.
- [x] Add transport-neutral `CommandContext` separating authenticated Principal, declared `ActorRef`, and optional external execution correlation.
- [x] Add `packages/wos-api/cmd/wos/main.go`.
- [x] Add `packages/wos-api/internal/server` bootstrap/configuration skeleton.
- [x] Add version command/output.
- [x] Add configuration validation skeleton.
- [x] Create `docs/adr/`.
- [x] Materialize foundational ADRs relevant to Wave 01.
- [x] Add static dependency-boundary verification.
- [x] Add unit tests for value objects and enum/ID serialization.
- [x] Enforce canonical flat `EntityRef` JSON addressing (`namespace_id`, `outcome_id`, `kind`, `id`).
- [x] Add an external compile test/example proving that public Core can be imported outside the module's internal packages.
- [x] Add GitHub Actions CI using the Go version declared by `go.mod`.
- [x] Ensure `go test ./...` passes.
- [x] Ensure process can print version and validate configuration.
- [x] Ensure `go vet ./...`, `go test -race ./...`, standalone build and smoke commands pass in CI.

### Verification

GitHub Actions validated commit `b5f7b0868fbf9673f5512fe8ae41cd2a68a2c744` with Go 1.27.1 on Linux. The successful Wave 01 foundation job executed:

- `gofmt` cleanliness check;
- `go vet ./...`;
- `go test ./...`;
- `go test -race ./...`;
- `go build -o ./bin/wos ./cmd/wos`;
- `./bin/wos version`;
- `./bin/wos config validate`.

The test suite includes the external-module Core import contract and static dependency-boundary verification, so the four Wave 01 completion gates are satisfied on the declared Go toolchain.

### Completion gate

Wave 01 is complete only when:

- `go test ./...` passes;
- external Core import example compiles;
- no prohibited dependency enters Core;
- the binary can print its version and validate configuration.

**Target integration commit:**  
`chore: bootstrap standalone WOS module and public core boundaries`

---

## Wave 02 — Transactions, Memory and Initial Domain

**Status:** ✅ Done

Implement Outcome, Objective, WorkItem and criteria with pure rules; application services; repository/transaction ports; transactional memory adapter; rollback; initial claim/complete semantics and Outcome coordination revision in memory.

### Implementation checklist

- [x] Implement Outcome lifecycle rules.
- [x] Implement Objective lifecycle rules.
- [x] Implement WorkItem lifecycle rules.
- [x] Keep archive state orthogonal to Outcome lifecycle.
- [x] Add versioned SuccessCriterion definitions.
- [x] Invalidate current assessment when a criterion revision changes.
- [x] Add minimum attestation assessments for Wave 02.
- [x] Enforce required-criterion gates before achievement.
- [x] Preserve explicit Conclusions and historical Conclusions on reopen.
- [x] Preserve immutable CriterionAssessment history while keeping a separate current-assessment projection.
- [x] Add basic WorkItem claim/complete semantics with Principal, TTL and fencing token.
- [x] Add declarative Outcome/Objective owners and WorkItem assignees, separate from lease ownership.
- [x] Keep WorkItem completion separate from Objective achievement.
- [x] Add aggregate optimistic versions.
- [x] Add repository ports for Outcome, Objective and WorkItem.
- [x] Add TransactionManager / UnitOfWork / CoordinationStore ports.
- [x] Add public transactional `packages/wos-core/storage/memory` adapter.
- [x] Add rollback behavior to memory transactions.
- [x] Add in-memory Outcome coordination guard/revision marker.
- [x] Add application services over ports instead of direct adapter access.
- [x] Expose the complete Wave 02 lifecycle command surface through Application services.
- [x] Enforce `required_for_outcome` before Outcome achievement.
- [x] Add human-only end-to-end scenario with no DB, network or Agent dependency.
- [x] Add failed-command rollback test for state and `outcome_revision`.
- [x] Validate aggregate lifecycle/lease/conclusion/criterion invariants at the domain boundary.
- [x] Reject structurally invalid aggregates before memory adapter saves.
- [x] Pass repository CI on the declared Go toolchain.
- [x] Review CI findings and close remaining Wave 02 contract gaps.

### Verification

GitHub Actions validated implementation commit `6590842a76b9fd5d6d4a370f1a8f8f4b0e5e41d2` with Go 1.27.1 on Linux. The successful run executed:

- `gofmt` cleanliness;
- `go vet ./...`;
- `go test ./...`;
- `go test -race ./...`;
- standalone binary build;
- `wos version` smoke test;
- `wos config validate` smoke test.

The suite covers pure lifecycle rules, criterion revision/assessment semantics, ownership/assignment separation, optimistic versions, transactional memory rollback, invalid aggregate rejection, Outcome revision markers, required-objective achievement gates and a full human-only scenario without database, network or Agent dependencies.

### Scope boundary

Wave 02 does **not** add Domain Events, IdempotencyStore, command replay or event timeline behavior. Those remain Wave 03.

**Completion gate:** a human-only scenario can create/activate an Outcome and organize Objectives/WorkItems without database, network or Agent dependencies.

**Target commit:**  
`feat(core): implement outcome objective work item and criterion lifecycles`

---

## Wave 03 — Event Log and Idempotency

**Status:** ✅ Done

Add command envelope, immutable Domain Events, ordered Outcome revisions, IdempotencyStore, normalized fingerprints and transactional replay.

### Implementation checklist

- [x] Add immutable DomainEvent envelope with command, actor, execution and ordering metadata.
- [x] Add event validation and 256 KiB payload bound.
- [x] Add CausationID to transport-neutral CommandContext.
- [x] Keep administrative audit records structurally separate from Outcome Domain Events.
- [x] Add IdempotencyIdentity and safe key validation.
- [x] Add persisted StoredCommandResult contract.
- [x] Add DomainEventLog and IdempotencyStore ports to UnitOfWork.
- [x] Add transactional in-memory Event Log.
- [x] Add transactional in-memory idempotency reservations/results.
- [x] Ensure rollback discards events and idempotency state.
- [x] Add normalized SHA-256 command fingerprints excluding retry-observational context.
- [x] Normalize declarative actor sets before fingerprinting.
- [x] Wrap all Wave 02 application mutations in the transactional audit/idempotency pipeline.
- [x] Preserve original IDs, versions, revision and command_id on successful replay.
- [x] Reject same idempotency identity with a different fingerprint.
- [x] Emit multiple ordered Events under one outcome_revision for compound command effects.
- [x] Implement explicit no-op semantics without version/revision advancement or misleading Event.
- [x] Add memory timeline snapshot ordered by (outcome_revision,event_index).
- [x] Add replay/conflict/no-op/rollback/event-ordering tests.
- [x] Pass repository CI on the declared Go toolchain.
- [x] Review CI findings and close remaining Wave 03 contract gaps.

### Verification

GitHub Actions validated commit `d9d380712721b58c7484c4a59c9244894cac73da` with Go 1.27.1 on Linux. The successful `WOS verification` run executed:

- `gofmt` cleanliness;
- `go vet ./...`;
- `go test ./...`;
- `go test -race ./...`;
- standalone binary build;
- `wos version` smoke test;
- `wos config validate` smoke test.

The suite verifies same-payload replay, different-payload conflict, transactional rollback of Event/idempotency state, multiple ordered Events under one Outcome revision, original result replay and explicit no-op behavior without misleading Events.

### Scope boundary

Wave 03 does **not** add SQLite persistence, HTTP/MCP transport requirements, Integration Events, trigger firings or outbox delivery. Those remain later waves.

**Completion gate:** repeated CreateWorkItem returns the original ID without duplicate Events, and timeline ordering is stable.

**Target commit:**  
`feat(core): add transactional audit events and command idempotency`

---

## Wave 04 — SQLite Persistence and Migrations

**Status:** ✅ Done

Implement SQLite adapter, schema migrations, SQL UnitOfWork, registry/coordination rows, repository contracts, idempotency/event persistence, restart-safe state and SQLite writer coordination.

### Implementation checklist

- [x] Pin the SQLite `database/sql` driver used by the adapter.
- [x] Record the SQLite driver/writer-acquisition decision in ADR 0006.
- [x] Enable `foreign_keys` for every adapter connection.
- [x] Enable WAL and synchronous NORMAL for the local durability profile.
- [x] Configure bounded `busy_timeout` integrated with command contexts.
- [x] Acquire the SQLite writer lock at transaction start with the immediate transaction profile.
- [x] Restrict the Store to one writer connection instead of pretending SQLite has parallel writers.
- [x] Add embedded, forward-only numbered migrations.
- [x] Add `schema_migrations` with SHA-256 checksum verification.
- [x] Add the canonical UTC microsecond timestamp codec.
- [x] Add Namespace, Principal, entity registry and Outcome coordination tables.
- [x] Add normalized relational persistence for Outcome, Objective and WorkItem.
- [x] Add normalized ownership/assignee links.
- [x] Persist revisioned SuccessCriteria and immutable CriterionAssessment history/current projection.
- [x] Persist current and historical Conclusions.
- [x] Persist Domain Events in the same SQL UnitOfWork as current state.
- [x] Persist idempotency reservations/results in the same SQL UnitOfWork.
- [x] Enforce optimistic writes with `UPDATE ... WHERE version = ?`.
- [x] Distinguish not-found from stale-version conflicts after zero-row updates.
- [x] Persist lease principals before FK-protected WorkItem lease writes.
- [x] Add restart-safe reconstruction of Wave 02 aggregates and nested state.
- [x] Add persisted idempotent replay across process restart.
- [x] Add compound rollback coverage for state + Outcome revision + Event + idempotency.
- [x] Add cross-scope foreign-key rejection coverage.
- [x] Add SQLite connection-profile and migration-idempotency tests.
- [x] Add consistent `VACUUM ... INTO` backup plus closed-destination restore.
- [x] Add writer-contention/busy-timeout context coverage.
- [x] Add transaction-cancellation coverage.
- [x] Add explicit claim-by-new-principal FK coverage.
- [x] Pin and verify the Go module graph with CI module hygiene.
- [x] Pass `go vet`, full unit/contract tests and race detector.

### Verification

GitHub Actions validated implementation commit `1895c642d2c7155c737d1d3adf9d95923d0889d0` with Go 1.27.1 on Linux in run `36963863527`.

The successful verification executed:

- module graph/tidy verification;
- `gofmt` cleanliness;
- `go vet ./...`;
- `go test ./...`;
- `go test -race ./...`;
- standalone binary build;
- `wos version` smoke test;
- `wos config validate` smoke test.

The SQLite suite verifies the full Wave 02 scenario after close/reopen, Outcome revision preservation, nested criteria/assessment/conclusion reconstruction, persisted Event ordering, replay after restart, stale `expected_version` rejection, compound rollback, scope foreign keys, backup/restore, writer contention, transaction cancellation and lease-principal integrity.

### Scope boundary

Wave 04 does **not** add the HTTP transport, transport-level authentication/local principal resolution, ETags/preconditions, OpenAPI or the first durable state query. Those remain Wave 05.

**Completion gate:** satisfied — the Wave 02 scenario survives process restart and versioned writes reject stale `expected_version`.

**Target commit:**  
`feat(storage): add SQLite persistence migrations and outcome coordination guards`

---

## Wave 05 — HTTP Vertical Slice

**Status:** ✅ Done

Expose the M1 domain through HTTP with DTOs, auth-local mode, version preconditions, idempotency keys, ETags, error mapping, OpenAPI and first state query.

### Completed

- [x] Add standalone SQLite + local-auth HTTP runtime and `wos server` command.
- [x] Add `/livez` and storage-backed `/readyz` health endpoints.
- [x] Expose Outcome, Objective, WorkItem and SuccessCriterion operations through versioned HTTP routes.
- [x] Require `Idempotency-Key` on remote mutations.
- [x] Translate aggregate versions through strong ETags and `If-Match` / `expected_version`.
- [x] Preserve/request correlation IDs and return a stable JSON error envelope.
- [x] Add the first coherent Outcome state query.
- [x] Add OpenAPI 3.1 contract under `packages/wos-api/openapi.yaml`.
- [x] Add reproducible HTTP documentation under `docs/http.md`.
- [x] Add transport contract coverage for replay, stale versions, no-op writes, invalid payloads, scope isolation and payload bounds.
- [x] Add durable SQLite HTTP restart coverage.
- [x] Add standalone HTTP runtime smoke verification to CI.
- [x] Pass formatting, vet, unit/contract tests, race detector, standalone build and runtime smoke verification.

### Verification

GitHub Actions run `36967299973` validated head `9a6cc4af2526420b6a8a6b6bbff209bed480f24a` with the complete verification job, including the standalone HTTP runtime smoke test. The suite includes a durable create/update/restart/read scenario over SQLite and an M1 HTTP contract scenario for Outcome, Objective, WorkItem and criteria.

**Completion gate:** satisfied — the standalone binary can create, modify, persist, restart and query durable state through HTTP, with reproducible documentation and a machine-readable OpenAPI contract.

**Target commit:**  
`feat(http): expose versioned outcome coordination API`

---

## Wave 06 — Dependencies, Graph and Readiness

**Status:** ✅ Done

Implement canonical `depends_on` relations, hard/advisory semantics, DAG validation, Objective hierarchy constraints, deterministic readiness, `not_before`, readiness reasons and ready-work queries.

### Completed

- [x] Add typed Outcome-local `Relation/depends_on` domain model with hard/advisory strength and `target_completed` satisfaction semantics.
- [x] Reject self-dependencies, invalid endpoint kinds, cross-Outcome references, duplicate active semantic edges and direct/indirect dependency cycles.
- [x] Validate DAG mutations while holding the Outcome coordination guard so opposite concurrent edge insertion cannot commit a cycle.
- [x] Preserve explicit audit reasons for dependency changes and require a reason when adding a dependency to an in-progress source.
- [x] Prevent dependency changes on terminal sources until they are reopened.
- [x] Enforce hard dependencies on Objective achievement and WorkItem completion without allowing advisory dependencies to block transitions.
- [x] Add deterministic Objective/WorkItem readiness policies and explicit readiness reasons.
- [x] Persist and restore WorkItem `not_before` in memory/SQLite and honor the exact `not_before <= evaluated_at` boundary.
- [x] Add deterministic `ListReadyWork` ordering by priority, creation time and ID.
- [x] Persist Relations through memory and SQLite UnitOfWork repositories with migration `0002_relations.sql`.
- [x] Include Relations in Outcome state and expose relation read/list operations.
- [x] Expose HTTP `/relations`, relation removal and `/ready-work` through the same Application services.
- [x] Extend OpenAPI and HTTP documentation for dependency/readiness contracts.
- [x] Prove hard/advisory behavior, cancelled-target semantics, dependency removal, in-progress mutation rules and indirect cycle rejection.
- [x] Prove memory/SQLite readiness parity, SQLite restart durability and concurrent opposite-edge rejection.
- [x] Pass formatting, vet, unit/contract tests, race detector, standalone build and runtime smoke verification.

### Verification

CI run `37008538950` (#97) validated implementation head `8549346c96e6e2c7f940b3c681158ac0064e011c` with module hygiene, formatting, vet, the full unit/contract suite, race detector, standalone build, version/configuration smoke checks and standalone HTTP runtime smoke verification.

**Completion gate:** satisfied — memory and SQLite produce equivalent readiness results for the tested state/time boundaries, and concurrent opposite dependency insertion cannot commit a dependency cycle.

**Target commit:**  
`feat(core): add dependency validation and deterministic work readiness`

---

## Wave 07 — Issues and Blockers

**Status:** ✅ Done and merged into `master`

Implement independent Issue and Blocker aggregates, typed causes/targets, direct/subtree propagation, explicit resolution, inherited blocking projection and bounded compound commands.

### Completed

- [x] Add independent Issue and Blocker aggregates with explicit versions and lifecycles.
- [x] Add Issue severity, affected references, investigation, resolution, wont-fix, duplicate and reopen semantics.
- [x] Reject duplicate-Issue cycles with a bounded chain walk.
- [x] Add Blocker targets for Outcome, Objective and WorkItem with direct/subtree propagation rules.
- [x] Support persisted local causes and explicit external causes while keeping Issue and Blocker lifecycles independent.
- [x] Reject new active Blockers on terminal targets.
- [x] Derive deterministic direct/inherited blocking state with explicit origin information.
- [x] Apply active blocking to WorkItem readiness without hiding the underlying Blockers.
- [x] Ensure resolving an Issue never resolves any Blocker implicitly.
- [x] Ensure resolving one Blocker never releases another active impediment.
- [x] Add `ReportIssueWithBlocker` as one atomic multi-aggregate command with separate per-aggregate Domain Events.
- [x] Add bounded `ResolveIssueAndBlockers` with explicit Blocker IDs, versions and `release_confirmed` values.
- [x] Keep compound mutations under one `command_id` and one `outcome_revision` while emitting one Event per changed aggregate.
- [x] Add versioned Issue detail updates and active-Blocker description updates with explicit no-op semantics.
- [x] Validate Issue `affected_refs` against persisted entities before mutation so memory and SQLite fail consistently.
- [x] Keep Decision as a reserved Blocker-cause domain kind but reject it at the Application boundary until persisted Decision records exist in Wave 09.
- [x] Persist Issues, affected references and Blockers through memory and SQLite UnitOfWork repositories.
- [x] Add SQLite migration and restart-safe reconstruction of Issue/Blocker state and blocking projection.
- [x] Extend Outcome state snapshots with Issues, Blockers and `blocking_states`.
- [x] Expose Issue/Blocker create/read/list/update/lifecycle operations through HTTP.
- [x] Expose the atomic report and explicit multi-aggregate resolution commands through HTTP.
- [x] Preserve idempotent replay, ETags and optimistic version preconditions for the new transport surface.
- [x] Extend OpenAPI and HTTP documentation through Wave 07.
- [x] Prove inherited subtree blocking, independent resolution, terminal-target rejection and duplicate-cycle rejection.
- [x] Prove compound rollback when one expected version is stale and prove unlisted Blockers remain active.
- [x] Prove the end-to-end HTTP contract, including replay, state projection, explicit release and external causes.
- [x] Pass formatting, vet, full unit/contract tests, race detector, standalone build and runtime smoke verification.

### Verification

GitHub Actions run `37020264539` validated implementation head `c21e454071f2159538af57e5e927330c52d7c979` with module hygiene, formatting, `go vet ./...`, the complete unit/contract suite, `go test -race ./...`, standalone build and all smoke checks.

The verified suite includes direct/inherited blocking, Issue/Blocker lifecycle independence, SQLite restart durability, atomic compound commands, rollback on stale multi-aggregate versions, explicit release confirmation, idempotent HTTP replay, versioned HTTP mutations and readiness recovery only after every applicable Blocker is closed.

### Scope boundary

`Decision` is already a valid Blocker-cause kind in the domain model, but Wave 07 does not invent non-persisted Decisions. The Application layer therefore rejects a Decision cause until the documentary-record implementation in Wave 09 can resolve that reference identically in every storage adapter.

**Completion gate:** satisfied — snapshots explain direct/inherited target/cause state, and resolving an Issue or one Blocker never silently resolves any other active Blocker.

**Target integration commit:**  
`feat(core): separate issues from blockers and implement blocking projections`

---

## Wave 08 — Leases and Fencing

**Status:** ✅ Done and merged into `master`

Harden WorkItem execution coordination with Claim/Renew/Release/Reclaim, principal-bound leases, TTL, fencing tokens, expiration semantics, attention-needed projection and audited administrative override.

### Initial implementation

- [x] Preserve the existing Claim/Release contract and harden lease TTL validation.
- [x] Add owner-bound lease renewal without rotating claim identity or fencing token.
- [x] Add explicit reclaim of expired in-progress work with a new claim ID and monotonically increasing fencing token.
- [x] Reject stale claimant completion after reclaim.
- [x] Add Application commands for Renew/Reclaim through the existing transaction, idempotency and Domain Event pipeline.
- [x] Emit explicit `work_item.lease_renewed` and `work_item.reclaimed` events.
- [x] Add deterministic `lease_status` and WorkItem `display_state` projection, including `attention_needed` for expired leases.

### Remaining work

- [x] Integrate the operational projection into the broader Outcome continuity/state queries.
- [x] Add audited administrative override semantics and the required authorization boundary.
- [x] Expose Renew/Reclaim and operational lease state through HTTP, OpenAPI and HTTP documentation.
- [x] Add SQLite restart/parity and concentrated concurrent-claim/reclaim contract coverage for the hardened semantics.
- [x] Complete the Wave 08 end-to-end transport and concurrency verification before marking the wave done.

### Administrative override boundary

- [x] Normal cancellation cannot silently terminate an in-progress leased WorkItem.
- [x] Add explicit `work:admin_cancel` and `work:admin_complete` permissions behind a transport-neutral Authorizer port.
- [x] Default privileged authorization to deny unless a deployment composes an explicit authorizer.
- [x] Keep local standalone overrides disabled unless `WOS_LOCAL_ADMIN_OVERRIDES=true` is explicitly configured.
- [x] Audit privileged state changes through Outcome-scoped Domain Events with principal, actor, command and reason.
- [x] Administrative completion overrides lease ownership/expiry only; Blocker, hard-dependency and completion-evidence gates remain active.

**Completion gate:** satisfied on the implementation branch — only one current lease can survive claim/reclaim races, fencing tokens advance monotonically, stale claimants cannot complete after reclaim, and bypassing lease ownership requires an explicit privileged permission and auditable reason.

**Target commit:**  
`feat(coordination): add work item leases and fencing tokens`

---

## Wave 09 — Documentary Records and Decisions

**Status:** ✅ Done and merged into `master`

**Goal:** preserve durable facts, deliverables and explicit choices with enough provenance that another consumer can understand the current state without relying on previous chat/session history.

### Domain contract

- [x] Add `Artifact` aggregate with immutable documentary identity/content metadata and `registered|withdrawn` lifecycle.
- [x] Add `Evidence` aggregate with immutable observation content and `registered|retracted` lifecycle.
- [x] Add `EvidenceLink` aggregate with link-local stance (`supports|contradicts|context`), rationale and `active|retracted` lifecycle.
- [x] Add `Decision` aggregate with `proposed|accepted|rejected|superseded` lifecycle.
- [x] Permit editing only while a Decision is `proposed`; accepted/rejected/superseded decision content is immutable.
- [x] Require explicit reason for Evidence retraction, Artifact withdrawal and Decision rejection.
- [x] Preserve producer/source provenance without claiming external authenticity or truth.
- [x] Keep Artifact registration reference-only: WOS must not dereference URIs or store arbitrary artifact bytes.
- [x] Keep EvidenceLink separate from criterion assessment: linking Evidence never marks a SuccessCriterion as met.
- [x] Validate all local targets as same Namespace + Outcome and compatible entity kinds.
- [x] Allow Evidence to carry different stances toward different targets.
- [x] Model Decision supersession as a dedicated semantic relation, not as a generic `Relation`.
- [x] Enforce no supersession cycles and at most one accepted direct successor for an accepted Decision.
- [x] Supersede an accepted Decision atomically when the successor Decision is accepted.

### Application and transaction contract

- [x] Add create/read/list commands for Artifact, Evidence, EvidenceLink and Decision.
- [x] Add explicit retract/withdraw commands with optimistic `expected_version`.
- [x] Add Decision update/accept/reject/supersede commands.
- [x] Route all Wave 09 mutations through the existing transaction + idempotency + Domain Event pipeline.
- [x] Hold the Outcome coordination guard for cross-aggregate reference validation and Decision supersession.
- [x] Emit canonical events: `artifact.registered`, `artifact.withdrawn`, `evidence.registered`, `evidence.retracted`, `evidence.link_created`, `evidence.link_retracted`, `decision.proposed`, `decision.updated`, `decision.accepted`, `decision.rejected`, `decision.superseded`.
- [x] Preserve no-op semantics without advancing aggregate version/outcome revision or emitting misleading events.

### Storage contract

- [x] Extend UnitOfWork/ports with Artifact, Evidence, EvidenceLink and Decision repositories.
- [x] Implement transactional memory repositories with clone/rollback parity.
- [x] Add SQLite migration `0004_documentary_records.sql`.
- [x] Persist provenance, immutable content fields, lifecycle, versions and Decision supersession constraints.
- [x] Register Wave 09 aggregates in the local entity registry.
- [x] Preserve all records and supersession state across SQLite restart.
- [x] Reject cross-scope references at both application and persistence boundaries where possible.

### HTTP and contract surface

- [x] Expose versioned HTTP routes for Artifact, Evidence, EvidenceLink and Decision operations.
- [x] Require `Idempotency-Key` on remote mutations.
- [x] Preserve ETag / `If-Match` semantics for mutable lifecycle/version transitions.
- [x] Extend OpenAPI 3.1 with Wave 09 schemas, enums, requests, responses and error behavior.
- [x] Extend `docs/http.md` with reproducible documentary-record scenarios.

### Verification

- [x] Prove registering an Artifact cannot conclude an Objective, WorkItem or Outcome.
- [x] Prove the same Evidence can support one target and contradict another.
- [x] Prove Evidence retraction does not rewrite the original observation.
- [x] Prove Artifact withdrawal does not delete historical metadata.
- [x] Prove accepted Decision content cannot be edited.
- [x] Prove Decision rejection requires a reason.
- [x] Prove two concurrent attempts to supersede the same accepted Decision cannot both commit accepted successors.
- [x] Prove supersession is atomic: successor acceptance and predecessor superseded state commit/rollback together.
- [x] Prove supersession cycles are rejected.
- [x] Prove cross-Outcome and cross-Namespace documentary links are rejected.
- [x] Prove memory and SQLite behavior parity for the Wave 09 lifecycle.
- [x] Prove SQLite restart preserves documentary history and current Decision state.
- [x] Pass module hygiene, formatting, vet, full unit/contract tests, race detector, standalone build and HTTP runtime smoke verification.

**Completion gate:** another consumer can reconstruct the current factual/documentary context and active choices of an Outcome from WOS state alone, including provenance, contradictory Evidence, withdrawn/retracted records and Decision supersession history, without previous chat history.

**Target commit:**  
`feat(records): add evidence artifacts and explicit decision history`
---

## Wave 10 — Assessments and Verifiable Conclusions

**Status:** ✅ Done and merged into `master`

**Goal:** make every positive conclusion reproducible from immutable criterion definitions, immutable assessments and an explicit obligation snapshot, while preserving later contradictory evidence/assessments as contestations rather than silently rewriting lifecycle.

### Domain contract

- [x] Preserve an immutable definition snapshot for every SuccessCriterion revision.
- [x] Keep the current SuccessCriterion definition separate from its immutable revision history.
- [x] Extend CriterionAssessment with explicit `evidence_ids` and optional `evaluator_ref`.
- [x] Support `met|not_met|inconclusive|waived` without conflating EvidenceLink with assessment.
- [x] Require `evidence_review` assessments to reference at least one active, same-Outcome Evidence.
- [x] Require `external_evaluation` assessments to include evaluator identity/version metadata and evidence required by the caller contract.
- [x] Require explicit authorization and reason for `waived`.
- [x] Keep CriterionAssessment immutable and retain supersession history.
- [x] Clear the current-assessment binding when a criterion is revised or retired; retain assessment history.
- [x] Introduce first-class Conclusion identity/history for Outcome, Objective and WorkItem.
- [x] Store the exact owner version, lifecycle result, criterion revisions, assessment IDs and structural obligations used by a Conclusion.
- [x] Reopening clears only the current Conclusion binding; historical Conclusions remain immutable.
- [x] Project `conclusion_contested` and concrete causes when current Evidence/assessments contradict the recorded Conclusion.
- [x] Never auto-reopen or auto-change lifecycle because a Conclusion becomes contested.

### Application and authorization contract

- [x] Replace the Wave 02 attestation-only command path with a generic RecordCriterionAssessment path.
- [x] Preserve attestation as a compatibility/application specialization where useful, but not as the domain's only assessment mode.
- [x] Add `assessment:waive` permission to the Authorizer contract.
- [x] Validate assessment Evidence against Namespace/Outcome, lifecycle and criterion revision inside the Outcome transaction guard.
- [x] Require expected owner version when updating the current assessment binding.
- [x] Detect concurrent assessments/current-binding replacements with optimistic version conflict.
- [x] Make AchieveOutcome/AchieveObjective/CompleteWorkItem build explicit immutable Conclusion records.
- [x] Enforce required Objective obligations for Outcome achievement.
- [x] Capture structural obligation snapshots before the terminal transition commits.
- [x] Emit canonical assessment/conclusion Domain Events through the existing transaction/idempotency pipeline.

### Storage contract

- [x] Treat `criterion_revisions`, `criterion_assessments`, `conclusions` and conclusion-assessment links as append-only historical records.
- [x] Persist assessment Evidence references and evaluator metadata.
- [x] Persist current-assessment/current-conclusion bindings separately from historical rows.
- [x] Add a Wave 10 SQLite migration for fields/relations missing from the predeclared foundation schema.
- [x] Reject assessment references to nonexistent criterion revisions.
- [x] Preserve criterion revision history, assessment history, current bindings and Conclusions across restart.
- [x] Keep memory and SQLite behavior equivalent.

### HTTP and contract surface

- [x] Expose generic assessment recording for all verification modes.
- [x] Require Idempotency-Key for assessment/conclusion mutations and ETag/If-Match for owner-bound current state.
- [x] Expose immutable assessment history and current bindings.
- [x] Expose current/historical Conclusions and contestation state.
- [x] Extend OpenAPI 3.1 and `docs/http.md` with Wave 10 contracts.

### Verification

- [x] Prove an assessment for criterion revision N never satisfies revision N+1.
- [x] Prove revising a criterion clears only the current binding and preserves old assessments.
- [x] Prove `evidence_review` without Evidence is rejected.
- [x] Prove retracted Evidence cannot support a new positive assessment/conclusion.
- [x] Prove a waiver without `assessment:waive` authorization is rejected.
- [x] Prove two concurrent assessments cannot silently replace the same current binding.
- [x] Prove required Objectives block Outcome achievement when not achieved.
- [x] Prove a Conclusion stores exactly the criterion revisions and assessment IDs used at commit time.
- [x] Prove later `not_met` assessment or Evidence retraction contests but does not erase/reopen a terminal entity.
- [x] Prove memory/SQLite restart preserves immutable validation history.
- [x] Pass module hygiene, gofmt, vet, unit/contract tests, race detector, standalone build and HTTP runtime smoke.

**Completion gate:** achievement records exactly which criterion revisions, assessments and structural obligations justified the conclusion, and later contradictory facts are visible as contestations without historical rewriting.

**Target commit:**  
`feat(validation): enforce versioned criterion assessments and explicit achievement`

---

## Wave 11 — Roadmaps and Planning History

**Status:** ✅ Done; integrated through PR #13

**Goal:** preserve planning and re-planning as durable history without creating a second operational graph or changing live execution state implicitly.

### Domain contract

- [x] Introduce Roadmap as an Outcome-scoped aggregate with `scope_kind=outcome|objective`, `scope_id`, title, lifecycle `open|archived` and optimistic `version`.
- [x] Keep Roadmap ownership separate from operational entities: Roadmap references Objectives/WorkItems and never owns or duplicates them.
- [x] Introduce editable Roadmap drafts with `draft_version` independent from aggregate `version` and published `revision_number`.
- [x] Introduce immutable RoadmapRevision records with content hash and publication metadata.
- [x] Introduce stable `node_key` identity across revisions.
- [x] Support RoadmapNode kinds `reference|phase|milestone`.
- [x] For reference nodes, allow only Objective or WorkItem targets from the same Outcome and permitted Roadmap scope.
- [x] Keep phase/milestone planning-only; they do not gain operational lifecycle, Blockers or leases.
- [x] Support optional parent grouping, presentation `position`, `after` ordering links, planned start/end and criterion references.
- [x] Reject duplicate reference targets within one revision.
- [x] Reject cycles in parent grouping and `after` plan ordering.
- [x] Snapshot published reference labels/scope so historical revisions remain understandable after live entities change.
- [x] Snapshot informative live dependency edges between referenced entities at publication time; do not duplicate operational dependencies as mutable Roadmap state.

### Publication and activation contract

- [x] Draft editing requires both `expected_roadmap_version` and `expected_draft_version`.
- [x] Published revision content is immutable.
- [x] Publishing computes/stores a deterministic content hash.
- [x] Publishing validates all references, scope restrictions, criterion references and plan DAG invariants.
- [x] `PublishRoadmapRevision` may include an explicit batch of operational dependency changes and applies plan publication + dependency changes atomically.
- [x] Absence of a plan edge or dependency never implies an execution command.
- [x] Introduce one active Roadmap revision slot per scope.
- [x] Activation only accepts a published revision belonging to the matching scope.
- [x] Activation history is append-only and distinguishes activation/supersession/deactivation without mutating published revision content.
- [x] Archiving a Roadmap clears its active slot in the same transaction.
- [x] Reopening a Roadmap never silently reactivates a prior revision.

### Storage contract

- [x] Add repository ports for Roadmap aggregate, drafts, published revisions and active-slot/history access.
- [x] Implement Memory adapter with deep-copy and append-only publication/history enforcement.
- [x] Add SQLite migrations/tables for roadmaps, drafts, revisions, nodes, plan edges, reference snapshots, active slots and activation history.
- [x] Enforce unique active slot per scope and unique published revision number per Roadmap.
- [x] Preserve all published revisions and activation history across restart.
- [x] Keep Memory/SQLite behavior equivalent.

### Application contract

- [x] Create Roadmap in Outcome or Objective scope.
- [x] Open a draft from empty content or an existing published revision.
- [x] Edit draft metadata/nodes/ordering with optimistic concurrency.
- [x] Discard draft explicitly; discarded draft cannot later be published.
- [x] Publish a draft atomically into an immutable RoadmapRevision.
- [x] Activate/deactivate a published revision for its scope.
- [x] Archive/reopen Roadmap with explicit lifecycle commands.
- [x] Ensure removing a node from a plan does not cancel/delete the referenced Objective/WorkItem.
- [x] Ensure cancelling operational work does not mutate historical Roadmap revisions.
- [x] Emit canonical Roadmap draft/publication/activation/archive Domain Events.

### HTTP and contract surface

- [x] Expose Roadmap create/read/list operations scoped to Outcome.
- [x] Expose draft open/edit/discard/publish operations with ETag/If-Match + Idempotency-Key.
- [x] Expose revision history and immutable revision reads.
- [x] Expose active Roadmap slots for Outcome and Objective scopes.
- [x] Extend OpenAPI 3.1 and `docs/http.md` with planning semantics and examples.

### Verification

- [x] Prove historical revisions remain byte/semantically stable after later draft edits.
- [x] Prove concurrent publication of the same draft has one winner.
- [x] Prove stale draft/roadmap versions fail explicitly.
- [x] Prove invalid cross-Outcome and invalid Objective-subtree references are rejected.
- [x] Prove parent and `after` cycles are rejected.
- [x] Prove one entity cannot appear twice as a reference in one revision.
- [x] Prove active slot uniqueness under concurrent activation.
- [x] Prove archived Roadmap clears the active slot and reopen does not reactivate it.
- [x] Prove node removal does not cancel live WorkItems/Objectives.
- [x] Prove cancelling live work does not rewrite prior revisions.
- [x] Prove title/criterion/reference snapshots survive live-entity mutation and SQLite restart.
- [x] Pass module hygiene, gofmt, vet, tests, race detector, standalone build and HTTP runtime smoke.

**Completion gate:** historical published revisions remain recoverable and immutable; each scope has at most one active published revision; Roadmap state never duplicates or silently mutates operational work.

**Target commit:**  
`feat(planning): add immutable roadmap revisions and scoped activation`

---

## Wave 12 — Continuity Queries

**Status:** ✅ Done on the validation candidate; final integration through PR #14

Implement coherent Outcome state snapshots, graph queries, timeline, search/filter, pagination, readiness explanations, plan projection, current decisions, contestations, progress metrics and bounded output.

**Completion gate:** another consumer can resume the complete scenario from snapshot + cursors without the previous Session.

**Target commit:**  
`feat(queries): add coherent outcome state graph and timeline projections`

---

## Wave 13 — First-Class MCP

**Status:** ✅ Done on the validation candidate; final integration through PR #14

Expose the Application layer through the pinned official Go MCP SDK with tools/resources, schemas, stdio and Streamable HTTP, explicit scope/version/claim arguments and parity with HTTP semantics.

**Completion gate:** a real MCP client can inspect, claim, update and resume an Outcome without duplicated domain logic.

**Target commit:**  
`feat(mcp): expose outcome coordination tools resources and protocol profiles`

---

## Wave 14 — PostgreSQL Parity

**Status:** ✅ Done on the validation candidate; final integration through PR #14

Implement PostgreSQL migrations, repositories, query store, row guard, isolation rules, pool/timeouts and the full shared storage/concurrency contract.

**Completion gate:** SQLite and PostgreSQL satisfy the same functional contract and concurrency suite.

**Target commit:**  
`feat(storage): add PostgreSQL adapter with coordination contract parity`

---

## Wave 15 — Authentication, Grants and External Context

**Status:** ✅ Done on the validation candidate; final integration through PR #14

Implement token authentication, Namespace grants, Authorizer, Principal/ActorRef delegation, ExternalContext indexing, ExecutionContext separation, external refs and administrative audit.

**Completion gate:** namespace isolation and delegated authorship are enforced independently of consumer metadata.

**Target commit:**  
`feat(auth): add namespace grants delegated actors and external context indexing`

---

## Wave 16 — Triggers, Integration Events and Delivery

**Status:** ✅ Done on the validation candidate; final integration through PR #14

Implement declarative event Triggers, public Integration Event mapping, TriggerFiring deduplication, outbox delivery, webhook signing, endpoint policy, worker leases, retries/exhaustion and redelivery.

**Completion gate:** consumers receive durable at-least-once signals with stable identity, while WOS still never executes WorkItems or LLMs autonomously.

**Target commit:**  
`feat(integration): add declarative event triggers and durable webhook delivery`

---

## Wave 17 — Packaging, SDK and Examples

**Status:** ✅ Done on the validation candidate; final integration through PR #14

Add complete binary commands, Docker/Compose, health/readiness, graceful shutdown, Go client, embedded/standalone/Woobe/product-MCP examples, backup/restore and operator documentation.

**Completion gate:** a third party can run WOS without Woobe and resume an Outcome using published documentation.

**Target commit:**  
`feat(server): package standalone deployment embedded usage and Go client`

---

## Wave 18 — Release, Validation and Demonstration

**Status:** ✅ Done on the validation candidate; final integration through PR #14

Complete CI, contract suites, race detector, benchmark fixtures, failure injection, migration upgrade validation, schema review, transport/storage compatibility matrix, changelog and real multi-actor continuation demonstration.

**Completion gate:** every Release 0.1 criterion has verifiable evidence and no missing feature is represented as complete.

**Target commit:**  
`test: validate WOS release contracts concurrency recovery and performance`

---

# Release 0.1 acceptance checklist

- [x] Standalone WOS starts without Woobe.
- [x] External embedded Go application compiles and executes.
- [x] Every remote mutation is protected by idempotency.
- [x] Stale aggregate versions are rejected.
- [x] Concurrent claim and dependency-cycle races preserve invariants.
- [x] SQLite and PostgreSQL pass the same storage contract suite.
- [x] HTTP and MCP use the same Application services.
- [x] Snapshots are coherent, bounded and explicit about omissions.
- [x] Published Roadmap revisions are immutable and recoverable.
- [x] Assessments preserve criteria, revisions and Evidence used.
- [x] Triggers produce durable signals without autonomous execution.
- [x] Documentation covers license, auth, backup, limits and supported MCP profile.
- [x] Restart/restore preserves state required to resume work.

# Current integration and validation

## Distribution extension — npm service, portable skills, automated releases

| Stage | Status | Acceptance |
| --- | --- | --- |
| D1 — Design and release policy | ✅ | [Complete plan](docs/distribution-plan.md) and ADR-017 define boundaries, tradeoffs, PR sequence, compatibility and external requirements |
| D2 — Portable skills installer | ✅ | Five portable skills, four managed discovery roots and generic path; safe install/update/remove and installed tarball verification. Codex pilot is distinct from universal runtime certification. |
| D3 — Service distribution | ✅ | Native metadata/integrity, versioned tarballs and Linux archive; real offline-installed service passes HTTP/MCP/UI, SIGTERM, exit status and corruption checks; Docker release metadata supported |
| D4 — Automated release | ✅ | Changes/version PR and reusable exact-source gates; verified immutable npm/GHCR/GitHub publication adapters and partial-failure replay; workflow lint, 15 Node contracts, reproducible assets and full local WOS gates pass. Hosted publication remains separately blocked |
| External publication | Explicitly deferred | Hosted Actions now execute; npm/GHCR/GitHub writes require explicit publish=true and authorized registry identity. No registry publication claimed. |

Do not conflate a portable skill with executed integration in every agent, nor implemented CI/CD with a successful hosted release. Original Waves 01–18 remain accepted; this is distribution work outside the original completion claim.

[Distribution verification](docs/verification-distribution-2026-10-07.md) records 437 Go cases with zero skipped tests, race, three browser journeys, the 18-step independent journey, installed npm service/skills, cross-path reproducibility and versioned read-only/authenticated container smoke. Nine Go packages without test files are not executed test cases. Publication requires working hosted infrastructure and titular credentials; a prepared 0.1.0 manifest is not a published npm version.

The first release is prepared as `0.1.0` in `release-manifest.json` with [version notes](docs/releases/0.1.0.md). D1–D4 integrate through PRs #16–#19; version preparation is [PR #20](https://github.com/A1b3rt0M3rcad0/wos/pull/20). The release job still requires the hosted gates before npm/GHCR/final GitHub publication. A GitHub draft with locally verified artifacts is preparation only, not a completed registry release.

Wave 11 was merged through [PR #13](https://github.com/A1b3rt0M3rcad0/wos/pull/13) at `f009b76dd4924490f0162839391218202abaeb79`. [PR #14](https://github.com/A1b3rt0M3rcad0/wos/pull/14), rebased in scope onto `master`, carries Waves 12–18 and the final hardening. Completion status above refers to the tested candidate; integration is confirmed by the PR's merged state, not by an unchecked draft or an earlier CI head.

[Verification dated 2026-10-07](docs/verification-2026-10-07.md) maps all thirteen canonical Release 0.1 gates to code and executed tests. The supported runtime matrix is Linux amd64, SQLite and PostgreSQL 18.6, HTTP and MCP; embedded and independent client examples are executable. The [agent guide](docs/agents.md) explains continuation, credentials, version conflicts, uncertain commits, leases and fencing.

1. Integration requires all final-head verification gates. Hosted Actions did not start on 2026-10-07 because GitHub reported failed account payments/spending limit. Equivalent gates were executed locally, including both Compose profiles, reproducible build and non-root/read-only container; this infrastructure failure is documented in the current verification and is not reported as a green hosted CI.
2. Validate the candidate in the consumer deployment using the documented commands, namespaces and credentials.
3. Publish a versioned release only when the maintainer elects to distribute it. No version tag or release asset is inferred from a merged PR.

## Acceptance evidence — 2026-10-07

- `internal/server/release_matrix_test.go`: 78 command schemas/idempotency guards, three proof modes and lost-response reconciliation across HTTP/MCP and both databases; real SDK clients.
- `storage/{sqlite,postgres}/release_concurrency_test.go` and `tests/boundary/dependency_concurrency_test.go`: separate connections/pools for claims, assessments, active slots, idempotency and opposite dependency edges.
- `storage/{sqlite,postgres}/bulk_history_test.go`: bulk/single read equivalence for owners, criteria, Evidence, current/historical Conclusions and reopen; immutable publication with live plan readiness and bounded graph references.
- `storage/postgres/clock_test.go`: authorized lease arbitration uses database time despite a replica clock skew of one year.
- Shared migration/restart/backup/restore contracts execute on real PostgreSQL with the test container; the independent 18-step standalone journey resumes the same Outcome after clean restore.
- Browser acceptance covers planning/replanning, human/agent continuation, proof review, Objectives, Issues/Blockers, conflicts, contestations, archive/resume, keyboard/focus and mobile rendering.
- In-flight HTTP shutdown and interrupted webhook retry preserve committed state and delivery identity. Query/guard/delivery metadata metrics exclude documentary payloads and credentials.
- Graph reachability, cursor scope and public command serialization fuzzing passed; measured query counts, pool wait and concurrent read throughput are in [benchmarks](docs/benchmarks.md).

The October 5 [audit](https://github.com/A1b3rt0M3rcad0/wos/blob/ec9aa1ecc836aa7b239f489cd70f8c6ad75930f1/docs/auditoria-conclusao-2026-10-05.md) and [verification](https://github.com/A1b3rt0M3rcad0/wos/blob/ec9aa1ecc836aa7b239f489cd70f8c6ad75930f1/docs/verification-2026-10-05.md) remain dated historical evidence in Git history; they have been removed from the active documentation. Its broader product checklist includes complete deployed Woobe acceptance and consumer distribution work. These remain explicit consumer obligations; they are not claims that WOS runs a deployed Woobe product. Current WOS release-gate acceptance is the October 7 matrix.

## Human workspace redesign — Jira UX / Guild.ai visual reference

**Status:** ✅ Implemented and locally verified (2026-10-07); integration through [PR #15](https://github.com/A1b3rt0M3rcad0/wos/pull/15).

Acceptance: persistent contextual navigation; Outcome summary, item list and operational board; readable entity/history/plan details; scoped human actions without exposing internal payloads by default; search/filter and pagination preserving snapshot semantics; keyboard/mobile behavior; existing human/agent journeys plus board/list regressions; real screenshots and PR integration. This is a presentation extension to Waves 12/17/18 and keeps WOS domain transitions, authorization and transport contracts intact.

Verified against the resulting repository: three Playwright journeys pass, including the real 32-item workspace, paginated board/list, proof selectors, cancellation, delayed Outcome responses, keyboard and mobile. API/server/boundary tests pass with real PostgreSQL enabled, plus vet/build/module syntax. The frontend is embedded with no new dependency or Domain/HTTP/MCP change. [UX guide and limits](docs/workspace-ux.md), [actual screenshots](https://github.com/A1b3rt0M3rcad0/wos/blob/ec9aa1ecc836aa7b239f489cd70f8c6ad75930f1/docs/ui-workspace/README.md) and audit logs make the result reviewable. Hosted CI remains blocked before steps by GitHub account billing; it is not claimed green. PR integration is established by its merged state.

## Execution contracts and API-only client — C01–C12

Owner authorized execution on 2026-10-08 of [the supplied plan](docs/work-contract-implementation-plan.md), baseline bcd1714ad7cded666890d0b5c9f711772c4f8d7a (confirmed unchanged). ADR-018 explicitly supersedes legacy release/override semantics in migrated Namespaces; ADR-019 defines client/workspace boundaries. Original Waves 01–18 and distribution evidence remain historical.

| Wave | State | Required gate |
| --- | --- | --- |
| C01 | Accepted design and typed fixtures | Plan, ADRs, bypass inventory and typed preliminary fixtures |
| C02 | Domain implemented and tested | Pure domain validity/terminal/overflow tests |
| C03 | Implemented in Memory; tested | Transactional Memory acquire/renew/revoke/takeover, replay and exclusivity |
| C04 | SQL parity and restore tested | SQLite/PostgreSQL parity, indexes, exact fencing, restore |
| C05 | Implemented and transport-tested | Typed HTTP/MCP/SDK, bounded queries, capabilities, drift |
| C06 | Implementado e testado | Sync documental atômico com local keys, checkpoints, submissões imutáveis, avaliação vinculada ao material, finalize e proteções de bypass; testes de aplicação, HTTP/MCP e restauração SQL. |
| C07 | Implementado e testado em Linux | Cliente API-only, YAML restrito/schemas, checkout/dry-run, planejamento/revisão via DTOs tipados; percurso real HTTP/CLI. |
| C08 | Implemented; actual native Linux/Windows gates executed | Durable recovery, stable native process locks and protected keyring execution passed on integrated precursor sources. Final 0.3 receipts are a separate gate. |
| C09 | Interface/skills implementadas e verificadas | Human contract UI, skills and client planning |
| C10 | Implemented and locally verified | Explicit Namespace phases, current lease policy, old receipt replay, shared/exclusive writer guards and bounded expiry reconciliation. Actual SQLite/PostgreSQL transition/restart tests pass; operators must stop old binaries before activation. |
| C11 | Implemented and tested | Actual HTTP/MCP/SDK/CLI and SQL restore, bounded reads and workload measurements. Woobe-specific runtime validation remains external. |
| C12 | Implemented; 0.3 prepared | Service/client/five skills and immutable packages/native assets/notices; final combined-source verification and explicit publication remain separate. |

T01–T88 in the plan are acceptance traceability, not passed tests. Actual Windows execution and deployed Woobe are not currently available; neither may be inferred from cross-compilation or HTTP parity. Existing Actions billing/npm identity blockers remain separate operational limitations; useful implementation continues independently.

C02: `domain/work_contract.go`, `work_submission.go` and `work_contract_test.go` cover exact expiry, all terminal causes, takeover, separate CAS, legacy direct release/override rejection, unsigned fencing including initial-claim overflow, JCS vectors and immutable-result digests. Existing domain/application/memory suites pass. Persistence and Namespace activation are not yet implemented.

C03: optional `WorkContractUnitOfWork` plus bounded repositories, transactional acquisition/snapshot, expiry+reacquisition, renewal, explicit takeover and privileged revocation. Memory deep-copies immutable material and CAS guards content/lease independently. Application race tests prove one concurrent winner, original replay, renew replay without extension, exact-deadline rejection and stale takeover fencing. WorkContract events are facts about their owning WorkItem; payload records contract identity and acquisition/result, avoiding a second entity_refs graph identity. Namespace cutover remains C10; no automatic migration occurs on acquire.

C04: additive migration 0016, one-open-contract uniqueness, holder/history/expiry/page indexes and scope FKs. Immutable spec is stored separately and never updated by renewal. New writers persist full unsigned task fencing as canonical decimal; original signed field is a legacy hint (zero above signed range), never the authority when decimal exists. PostgreSQL contract token uses NUMERIC(20,0); SQLite uses exact text. Both complete storage suites pass with race on real PostgreSQL 18.6 and clean SQLite/PostgreSQL restore. New cases cover independent connections, replay after restore, expired authority, >2^53 and uint64 maximum. PostgreSQL authorized contract acquire/renew/query also pass with replica clock skew of a year. Expiration is recorded during reacquisition, effective at expires_at, independently of maintenance workers.

C05: generated HTTP/MCP/SDK catalogue now includes 83 commands; contract state/spec/history, submissions, checkpoint pages, capabilities, own-principal receipt lookup and available/recoverable candidates are exposed. Authority DTO is a named `authority` object with decimal-string fencing; new wire contract was not released previously. Acquire-next scans bounded indexed headers in deterministic priority/creation/ID order, declares incomplete search and preserves empty-intent replay. Work context expands current contract only when revision matches, refusing a mixed snapshot. Contract spec cap is 128 KiB; command/query caps remain 256 KiB. Independent Go SDK + real Streamable HTTP MCP client proves HTTP acquisition → MCP takeover → stale HTTP renewal rejection, exact >2^53 fencing, own receipt, bounded history and search continuation; race passes. SQL contract suites remain verified; broader multi-storage transport proof follows C11.

C06: testes com race detector em aplicação, SQLite/PostgreSQL reais e cliente HTTP/MCP independente; restauração preserva submissões, avaliações, conclusões e recibos. O modo novo ainda depende do cutover explícito C10; CLI e validação Windows seguem pendentes.

C09: painel de contrato/último checkpoint/submissão/revisão/revogação, histórico sob demanda e exclusão dos comandos legados para WorkItems migradas. Cinco skills e quatro destinos npm testados; três jornadas Chromium existentes passaram. A jornada Chromium específica de cutover + checkpoint + revisão de submissão vinculada + revogação privilegiada passou em C10. Instalação de skill não certifica execução em cada runtime.

C10: ADR-020 documents anchor-Outcome auditing, no historical contract fabrication, operational old-writer retirement and no silent downgrade. Migrated work preserves IDs/versions/fencing; new work supports typed ExecutionSpec. Domain permits administrative cancellation only after authority detachment. CLI protocol get/set/reconcile uses authenticated public API and durable command journal. New optional command fields omit nil values to preserve legacy command fingerprints.

C11: context query reads selected work under one guard and preserves current proof, with historical proof expansion declared. Contract history reads metadata only; large latest checkpoint/submission has explicit expansion paths. Redirected mutations stay sent_unknown; acquisition recovery orders prepared intents. Frozen spec includes Outcome/Objective intent for clean sessions. Load fixture counts transaction conflicts and bounded identical-intent retries instead of hiding them; same-Outcome PostgreSQL contention is material. The 54-case report identifies tested implementation commit 6179387 and preserves the full measured matrix in docs/work-contract-load-evidence.json. Four Chromium journeys and all five skill installation tests passed. A full race run exposed the new transitive-module dependency in the external consumer test; the corrected real external consumer test passed. Final all-package integrated verification remains a C12 gate.

C12: WOS npm now contains wos + wosctl; separate client archives include schemas/notices/source metadata. Publisher requires native Linux/Windows receipts matching source and binary hashes, and hosted release depends on the CLI matrix. Version 0.2.0 prepared coherently; five skills install 20 managed copies across four discovery roots. Local authorizer grants ordinary acquisition to its single Principal, and revocation only with existing admin opt-in; Namespace administration still requires an authenticated Namespace administrator. Offline packaging uses an isolated API-token bootstrap instance and does not relax runtime authorization. Exact implementation commit 2ebe93fa18cbc2ec3d9570fd2725cec5ff9d3ddf passed all-package race/vet, four Chromium journeys, 16 Node tests, offline packaging, cross-path reproducibility, native Linux acceptance and non-root/read-only container verification. See docs/verification-work-contracts-2026-10-08.md; native Windows, Woobe and external publication are still pending.

## Signed contracts and independent minimal profiles — P00–P13

Owner authorized execution of [the new plan](docs/signed-contracts-implementation-plan.md), baseline b75aa3e confirmed against origin/master. C01–C12 above remain historical. ADRs 021–026 explicitly change behavior only for signed_contracts_v2. Product remains state/coordination, not agent execution.

| Wave | Actual state | Gate |
| --- | --- | --- |
| P00 | Design accepted; implementation baseline frozen | Full original plan, ADRs, bypass/tables inventory, migration reservations and four design fixtures; no proposed command advertised as existing. |
| P01 | Codec implemented and locally verified | Public pure signing package, six payload purposes, typed bindings/counters, exact verified DTO, local mapping/proof and independent Python vector. Race signing/domain, vet and two 15-second fuzz campaigns passed; no runtime authorization claim. |
| P02 | Implemented and tested | Principal-bound enrollment/rotation/revocation, credential policies and durable security receipts integrated with signed operations. |
| P03 | Implemented; essential release gates in P12 | Signed issuance/runtime trust and storage foundations integrated with subsequent delivery/review/cutover work; final-source gates are required for each release. |
| P04 | Implemented; essential release gates in P12 | Atomic signed return implemented: exact envelope/spec verification, documentary local keys, fresh explicit direct assessments, delivery/review handoff and durable signed acceptance. |
| P05 | Implemented; cross-transport and release gates in P12 | Independent signed review/leases, exact immutable targets, correction rounds, Task completion and explicit administrative interventions. Exhaustive cross-transport/fault acceptance remains P06/P12. |
| P06 | Implemented; essential release gates in P12 | Exact signed schemas/counters, live Outcome scope filtering, public identity/enrollment, bounded focal proofs/material, unsigned guards and actual HTTP/MCP/SDK independent review journey. Full race/vet/restore and generator gates passed locally. |
| P07 | Implemented; native protected-store gates executed | Approved protected profiles and exact workspace binding, Linux Secret Service and Windows Credential Manager actual hosted execution. Same-source native receipts are required for each release. |
| P08 | Implemented; fault and release gates in P12 | Stable locks, bounded execution/review batches, embedded intentions, verified atomic materialization, renew/resume/refresh and supervised keepalive integrated. Process/file failure gates execute on the source selected for release. |
| P09 | Implemented and tested | Protected signing/return/finish, original-CID recovery and verified receipt cleanup. Native death and real Linux ENOSPC fixtures; same-source release gates remain mandatory. |
| P10 | Implemented and tested | Explicit cutover, bounded preserving workspace migration, actual v0.2 historical upgrade and old-writer readiness refusal, pending SQL restores, host issuer recovery and client pin approval. |
| P11 | Implemented and tested | Five progressive guides, signed UI projections and guarded actions, exact correction acknowledgement and installed-binary independent correction journey. |
| P12 | Implemented; release-source acceptance is gated | Real SQL/race/restore, native platforms/keyrings, installed correction, bounded load/fuzz/faults and fresh-context Codex pilot. A/B/C Codex functional controls ran with one task each and identical external checks; provider usage/cost and exhaustive assertion audit remain explicitly unclaimed. |
| P13 | Coordinated 0.3.0 implemented; publication explicit | Version/manifest/OpenAPI/notices/upgrade/skills aligned; same-source verifier precedes explicit registry publication. |

R01–R24 / T01–T96 traceability is in plan section 20.13; acceptance is not inferred from fixture/file/commit counts. Native Windows, hosted Actions billing, registry identity and actual Woobe/model telemetry were unavailable in the previous phase and must be rechecked, not assumed fixed. No secret values are requested or persisted by this design wave.

P01: crypto/ed25519 + existing pinned JCS, no new product dependency. Independent Python cryptography 50.0.0 generated the published RFC8032-seed DSSE vector; Go matches exact bytes/signature. Tests reject duplicate keys, lone surrogates, noncanonical bytes/base64, unsafe numeric integers, malformed key sizes, multiple signatures, wrong purpose/inner signer, unknown verified DTO fields and unsigned companion commands. Exact uint64 counters remain strings through maximum. Semantic scalar cap is distinct from technical envelope base64. Linux race/domain and vet passed; fuzz canonical/envelope ran 15 seconds each (50,477 and 552,465 executions). Actual Windows remains unexecuted. No mutable-domain/transport behavior changes in this phase.

P02: additive migration 0020 stores only public signing identities, bounded one-use enrollments, credential/acceptance policies, Principal separation groups and immutable security receipts. Application requires authenticated credentials and a live transactional access snapshot, uses authoritative transaction time, verifies possession before deriving mutations and audits Namespace CAS changes. Rotation requires the previous authorized active key; it retires that key while preserving historical verification and the stable Principal/floor. Explicit administrator enrollment plus audited key revocation supports recovery without treating a bearer token as key-replacement authority. Missing legacy policy does not authorize registration. Memory now implements security ports; SQLite and PostgreSQL share generated repository contracts. Focused race contracts exercise forged key hints, another challenge, scoped views, replay versus second consumption, rotation/revocation, policy/grant intersection, concurrent enrollment and stale authenticated contexts after bearer revocation. v2 work execution, quotas, scoped read filtering and transport exposure are subsequent waves, not asserted by P02.

P02 verification: full repository `go test -race -p=1 ./...` passed with actual PostgreSQL and clean pg_dump/pg_restore fixtures enabled. Final identity/concurrency tests passed under race detection on Memory, SQLite and PostgreSQL after authoritative-clock and scoped-view changes; vet passed for changed Application/domain/ports/storage packages. No native Windows or hosted Actions execution is inferred.

P03 foundation (partial wave): delivered is a signed-only terminal execution status, while review cases/authorities preserve independent Principal/execution/fencing/lease state and immutable submission/spec/policy/round/history. Migration 0021 adds separate signed tables, immutable payload/proof facts and indexed scoped counts without rewriting v1 data. v2 execution/review SQL versions remain exact above 2^63/2^53 (SQLite decimal text, PostgreSQL NUMERIC(20,0)); the specification payload excludes a circular self-digest. Both generators are checked by CI. Open reviews block embedded unsigned Task edits, assessments and parent completion; active signed authorities block material edits. Full repository race suite passed against real PostgreSQL with clean restore enabled, plus vet and actionlint. A test-only pg_dump client option bounds parallel catalog workers after Docker shared-memory exhaustion; production settings are unchanged. Domain/storage fixtures are not signed acquisition/return endpoints, activation or workspace support. Remaining P03 gates are server identity/trust composition, signed issuance/acquisition/renewal and full protocol integration; P04–P13 remain pending.

P03 signed issuance implemented and locally verified: persistent immutable public server identity and deployment-local Ed25519 signer, secret seed environment reference, quoted signed work-spec/material counters and distinct v2 material digest, epoch-2 metadata constraints with row-preserving migration 0023, authenticated acquisition with frozen strongest acceptance floor, indexed Principal/credential/Namespace counts, exact signed specification/grant, renewable grant and fenced takeover without TTL extension. Targeted race tests passed on Memory/SQLite/PostgreSQL for acquisition/quota/replay/revocation and lease/spec separation; restart tests pin server public trust and reject silent seed replacement. Generated HTTP/MCP/Go SDK catalog now includes the three acquisition/lease commands. Security/trust/profile query exposure and full transport acceptance remain P06; public protocol activation and signed delivery/review remain gated to following waves. No Namespace is automatically activated.

P03 issuance verification: full `go test -race -p=1 ./...` and `go vet ./...` passed with actual PostgreSQL and clean restore enabled. Generated HTTP/MCP/Go SDK catalog and boundary tests passed after regeneration; integration event catalogue checks passed. Hosted Actions and native Windows remain external gates, not inferred from local results.

P04 return verification: shared transaction composition reuses documentary validation without nested commands; v1 finalization cannot finalize a signed Domain contract. Targeted race fixtures exercise actual credential/key policies, exact signed material with newly registered artifact/evidence, required assessments, review-floor rejection, full authority preservation on invalid references, signed receipt binding, revoked-bearer rejection and durable receipt replay after SQL idempotency-cache expiry. Memory, SQLite and actual PostgreSQL targeted tests passed; full repository `go test -race -p=1 ./...` and `go vet ./...` passed with clean PostgreSQL restore enabled. Final refinements (Tasks with and without required criteria, accepted-material digest and optional conclusion reason) passed targeted race tests on all three adapters, transport/boundary checks and vet. Broader fault injection, cross-transport flows, corrections, profiles and release acceptance remain P05–P13.


P05 verification: independent signed review/renewal/takeover, inconclusive release, accepted changes and a new correction executor, live blocker rollback, supplemental same-source evidence, administrative review revocation and case cancellation/supersession passed shared race fixtures in Memory/SQLite/actual PostgreSQL. The original executor credential is revoked before correction/review completion; original delivered contracts and signed submissions remain immutable. Replanning after intervention requires explicit case acknowledgement and a new fenced authority. Full repository race tests passed, including PostgreSQL clean restore and architectural boundaries; generated HTTP/MCP/SDK and 102-event catalog have no drift. Metadata counter transport and complete focal/bypass acceptance remain P06; exhaustive T01–T96, native Windows and real consumer telemetry remain P12 gates.


P06 first-part verification: exact new signed command and mutation counters (including Task/lease fences) round-trip above 2^53 and at uint64 maximum without changing v1 numeric encoding or immutable issued proof. Strict signed DTOs reject nested/case aliases even under an otherwise valid signature. Generated OpenAPI includes purpose-specific exact-field schemas. Shared signed acquisition/return/review/correction and legacy lease/idempotency race fixtures passed in Memory/SQLite/actual PostgreSQL; Core, command/HTTP/runtime, boundaries and vet passed. Focal queries, permitted-Outcome filtering, full remaining guards and cross-transport acceptance are still in progress; this is not P06 completion.


P06 focal verification in progress: bounded contract/case/correction/receipt/submission queries expose verified exact envelopes and public historical keys, with digest-bound expansions and explicit scan cursors. Public HTTP/MCP/SDK signing administration uses the same transactional enrollment/policy service; identity metadata includes Namespace CAS and key-list truncation. Outcome restrictions filter storage before search pagination and revalidate within legacy/focal read snapshots, including Namespace receipt policy changes. Active signed authority/open reviews freeze parent material and obligations; accepted signed Task assessments reject unsigned rewrites after completion. Legacy material-edit behavior remains preserved. Available-work projections exclude pending review/correction and resolve signed authorities separately from v1 repositories. Shared Memory/SQLite/actual PostgreSQL targeted race fixtures passed for focal queries and changes-requested acceptance. Real HTTP acquisition, MCP return and an independent SDK reviewer after executor bearer revocation passed on all three adapters, preserving exact receipt bytes/fencing and the original delivered execution. Final full repository race suite and final public bypass refinements are being completed; no hosted CI/native Windows/publication/pilot claim.

P06 verification discovered two environment/bootstrap limits: the dedicated PostgreSQL container's accumulated test catalog required more than 64 MiB shared memory (94 MiB observed), so its data volume was preserved while recreating the container with 512 MiB and retaining the old stopped container for reversal. Development-container parallel scan/maintenance settings are zero. No production data/configuration was changed. SQLite startup/migration cancellation previously reused a minimum five-second writer timeout and intermittently interrupted migration 24 under race detection. `Options.StartupTimeout` separates the bounded migration budget (30-second default) from unchanged writer contention. A targeted cancellation/contention/indexed-context race test passed; final full-suite rerun remains required before P06 integration.

The full frozen-source run subsequently passed SQLite (107.122s) but the historical cross-transport PostgreSQL fixture used a one-second HTTP request budget and timed out during its normal CLI submission under race instrumentation. That acceptance fixture now uses ten seconds, matching the signed transport fixture; dedicated deadline/cancellation assertions and production request-timeout configuration are unchanged. A complete rerun of the resulting source is required before merge.


P06 final verification: the frozen source 0eb0f66b040c239c4e8559af009f54b209613a51 passed full `go test -race -p=1 ./...` with explicit actual PostgreSQL DSN and clean restore container settings after the recorded environment/bootstrap fixes. PostgreSQL full contracts passed in 115.665s and SQLite in 107.122s; the final cross-transport acceptance suite passed in 14.409s. Earlier interrupted/mixed-source/SHM-exhausted runs are not counted as acceptance. Final scoped/focal/embedded/public unsigned guards and possession-based HTTP/MCP/SDK review after executor revocation passed on all three adapters; final vet/actionlint and generated artifacts/catalog had no drift. Hosted Actions still fail before steps because account payments/spending limit; no hosted/native Windows/publication/pilot claim. P07–P13 remain pending and are the next work, not declared complete by P06.


P07 foundation: schema-2 project/profile and one-contract documents, explicit flag/env/unambiguous profile selection, exact-case collisions, confined regular-file checks, bounded semantic versus technical YAML scalars, exact uint64 counters, proof verification, editable human material names, content comparison before replacement, and explicit env/mounted/OS-keyring secret backends. Profiles carry authenticated local destination/pending bindings without adding operational directories. The pinned MIT keyring dependency is CLI-only; notices include Windows dependencies. Targeted race tests verify changed destination/identity/references/pending rejection, signature round trips and tampering, editor conflict preservation, YAML limits, symlink/hardlink rejection and redacted backend errors. This is a foundation, not completed P07: CLI enrollment/profile commands, stable process locks, durable recovery, sign/send/finish, cutover, skills/UI and release gates remain pending. Native Windows and actual OS-keyring execution remain unproven.


P07 onboarding continuation: `init --workspace-schema 2` / `project init`, explicit `profile create/list/inspect/recover/remove`, protected `--token-stdin`, generated OS-keyring seeds or dedicated `--signing-key-stdin`, and preprovisioned env/mounted proposals. First onboarding requires independent exact origin/server ID/issuer fingerprint approval; existing keyring entries are reused after interruption and never silently overwritten. Credentials/keys stay out of arguments, YAML and diagnostic output. New profiles persist authenticated frozen enrollment intent before registration; recovery replays the exact durable security intention. Actual HTTP/Core acceptance on Memory/SQLite/PostgreSQL commits enrollment, repeatedly drops its response, preserves `sent_unknown`, replays the original intention and consumes it once. Live identity inspection is compact; profiles do not advertise server permissions as authority. Removal refuses pending intentions or remaining contract files, preserves shared secrets and does not impersonate server revocation. A signed project cannot silently dispatch legacy operations.

P08 lock foundation was intentionally implemented with onboarding to prevent concurrent reservations: canonical root plus case-collision-safe profile and contract identities, private stable Unix advisory lock files outside the workspace, and Windows named mutexes with thread ownership. Native Linux process fixtures verify exclusion after YAML replacement, independent profiles, case collisions and automatic release after owner death. These do not establish Windows-native behavior. Acquisition recovery after server cache expiry, batches, file cleanup and the exhaustive T49–T72 matrix remain unimplemented gates, not inferred from enrollment recovery.

P07 verification: full repository race suite passed with explicit actual PostgreSQL DSN and clean restore container, including the real HTTP enrollment response-loss/replay journey on all adapters. Vet, generated schemas/catalog, actionlint and Windows cross-compilation passed. The dedicated protected-seed import fixture subsequently passed race/vet and does not claim native keyring execution. Hosted Actions still cannot start because account billing/spending limit; native Windows, registry publication and actual consumer/model telemetry remain external acceptance gates.


P08 server recovery foundation: additive migration 0025 and a shared public Core port retain immutable signed-operation responses independently of short cache expiry. The common pipeline reauthorizes current credential/policy/scope and selected acquisition/lease key, then binds replay to exact Principal/CredentialID/Outcome/command/fingerprint and original command ID/revision. Replays return original grant/fencing/CAS/expiry after renewal/takeover and do not emit another event or acquire/renew work. A bounded credential-bound `operation` expansion supports harness reconciliation; initial agent context does not include its technical blob. Actual Memory/SQLite/PostgreSQL race contracts passed cache expiry plus an unavailable-cache adapter, changed payload/command/CID rejection, original grant preservation and clean SQLite/PostgreSQL restore. HTTP/SDK recovery expansion and existing independent review/return fixtures passed. The full frozen-source race suite passed with actual PostgreSQL (166.753s), SQLite (105.092s) and cross-transport acceptance (13.110s). The final SDK-only refinement also passed race/vet: operation expansion permits the full bounded 1 MiB registry response plus base64/metadata overhead while ordinary v1 responses retain their 256 KiB limit. Exact committed-source verification is recorded in the integration PR. Workspace pending/materialization/batch/sign/send/cleanup and T57–T72 remain subsequent work; durable enrollment alone did not satisfy work recovery.


P08 compatibility refinement: before reserving/deleting a short-cache entry, the new pipeline can recover a still-present pre-P08 accepted signed response even when expired. It resolves the original CID from the historical contract, rejects substitution, and promotes the exact original result once. Read-only focal recovery does not persist a promotion. Actual SQL race/restore fixtures cover expired legacy rows plus another CID sharing Principal/Actor/key permission. Ambiguous or unresolvable historical cache intentions require explicit reconciliation; no retroactive proof is fabricated.


P08 work acquisition continuation: public `AcquireNextSignedWorkContract` scans bounded candidates under the Outcome guard and persists empty/acquired results with independent intentions. Replay does not become a new search after a Task appears; incomplete pages preserve cursor/completeness. The signed Namespace creation path now enables contracts for new Tasks, including during signed drain, without granting acquisition during drain. Schema-2 work checkout/count/recover/list/show persists the bounded independent set before network calls, uses short profile locks and observed content snapshots, verifies pinned issuer documents, persists accepted-unmaterialized state and preserves existing edited drafts. Recovery first reconciles credential-bound accepted state and exact command fingerprint; accepted acquisition remains recoverable after key revocation, with no acquisition call. Five-item/second-response-loss actual HTTP fixtures passed on Memory/SQLite/PostgreSQL, materializing exactly two original eligible contracts and preserving a first draft edit. Shared tests passed empty replay, incomplete cursor, unavailable cache, original public response, technical expansion above 256 KiB and quota. Snapshot/independent-intention race fixtures passed. Review checkout, lease maintenance, crash-in-file-write/cleanup fault coverage, P09 returns, P10 cutover and P11–P13 remain pending; this is not completion of P08. Full frozen-source `go test -race -p=1 ./...` passed with explicit actual PostgreSQL and clean restore (PostgreSQL 152.920s, SQLite 119.659s, acceptance 19.707s). Vet, actionlint, generated contracts/catalog and Windows cross-compilation passed. The owner explicitly accepts hosted jobs being refused and authorized continued local validation/integration; refused jobs are not counted as executed CI or native Windows acceptance.

P08 independent review continuation: bounded public signed-next review acquisition shares the same transaction, historical/current independence checks and quota with specific-case checkout. Empty/acquired results use the immutable credential-bound operation registry. Schema-2 review checkout/recover/list/show reuses independent profile intentions, verifies issuer/target documents, materializes an inconclusive draft and retains exact acquired case CAS. Agent projection verifies nested delivery/work documents and keeps authentication proofs outside model output. Actual HTTP loss-after-commit recovery acquires no replacement and preserves the editable draft; the subsequent independent SDK review still completes after executor credential revocation. Targeted shared race journeys passed Memory/SQLite/actual PostgreSQL, including original review authority replay after takeover with unavailable short cache. Full frozen-source race suite passed with actual PostgreSQL/clean restore (165.153s), SQLite (127.266s), acceptance (20.951s) and architectural boundaries. Vet, actionlint, 102-event catalog and Windows cross-compilation passed; Windows-native and hosted jobs are not counted as executed. Lease maintenance, file-write/cleanup fault acceptance and P09–P13 remain pending.

P08 lease maintenance continuation: schema-2 work/review renew, resume and explicit refresh, profile-embedded immutable lease intentions, accepted-operation lookup before uncertain replay, and verified authority-only merge preserving observed editable drafts. Resume rejects TTL overrides and retains expiry; explicit refresh preserves semantic Task/case CAS. Foreground/one-pass keepalive handles items independently, retains revoked/expired files and pauses unresolved final/lease operations without a server scheduler. Actual HTTP response-loss fixtures passed Memory/SQLite/PostgreSQL, preserving a draft edited after the failed renewal and reconciling without another renewal call; takeover fencing/expiry, two-contract keepalive, one revoked contract alongside a renewed live contract, and review renewal/resume/refresh/keepalive followed by independent SDK approval passed. Unit race checks bind exact frozen authority/CAS and reject competing intentions/final markers. Signed SDK commands now allow bounded technical responses above 256 KiB without widening legacy limits. The profile-lock/network assertion passed in the actual HTTP journey. Full final-source race suite passed with explicit actual PostgreSQL/clean restore environment (unchanged Core/storage fixtures cached), final acceptance 38.017s and architectural boundaries. Vet, actionlint, catalog and Windows cross-compilation passed; no Windows-native or hosted execution is inferred. Foreground intervals are bounded to 300 seconds and must be shorter than the requested TTL. Atomic file publication/abandoned staging reconciliation, P09 receipt cleanup and P10–P13 remain pending.

P08 atomic materialization continuation: complete fsynced staging plus exclusive publication replaces direct writes to final paths during initial v2 creation. Accepted contracts use an original-intention-bound staging path and recover constructor-prefix partial writes, verified complete drafts, published two-alias pairs and post-unlink process death without reacquisition. Exact two-alias confinement preserves current destination contents; additional/outside links and differing staged data remain rejected/preserved. `.yml` destination renames retain draft contents rather than creating another `.yaml`. Technical directory scanning is bounded. Native process-death race fixtures and full CLI race suite passed on Linux; actual Memory/SQLite/PostgreSQL HTTP recovery repaired half-written materialization from the original accepted response while preserving the first edited draft. Generic pre-publication onboarding/replacement stages remain preserved for diagnosis, and noncooperative final-compare/rename races are not claimed as an OS isolation guarantee. Final alias/concurrent-reader refinements passed the full frozen-source race suite with explicit actual PostgreSQL/clean restore environment (unchanged Core/storage fixtures cached), acceptance 31.240s and architectural boundaries. Vet, actionlint, 102-event catalog and Windows cross-compilation passed; Windows-native/hosted execution is not inferred. P09 receipt cleanup and P10–P13 remain pending; this is not final product acceptance.

P09 credential-bound cleanup foundation: focal signed execution/review metadata now exposes the original public CredentialID, and historical work/review acceptance fallback reauthorizes and compares that original CID before disclosing a mutation replay when both cache layers have no row. Existing signed payloads/signatures are not rewritten. Shared Memory/SQLite/actual PostgreSQL port fixtures retain authoritative signed facts while modeling absent cache layers: original ownership replays exact acceptance after key retirement; another CID with the same Principal/Actor/key permissions is rejected for execution and review. Actual public HTTP projection retains original execution CID after independent completion. Targeted race tests passed (SQLite 19.480s, PostgreSQL 13.584s, acceptance 23.505s). CLI refresh/keepalive now checks original CID as well. Full frozen-source race suite passed with explicit actual PostgreSQL/restore (PostgreSQL 167.946s, SQLite unchanged-source cached, acceptance 29.440s, boundaries 6.281s); vet, actionlint and catalog passed. An earlier full run failed only after the environment disk filled and is not counted; disposable Docker build cache was reclaimed without dropping database data. Sign/send/finish and receipt-bound deletion remain subsequent P09 work.

P12 hosted platform recheck: PR #49 hosted browser, packages and Linux platform jobs executed successfully after prior account refusals. Windows executed and failed, but the distribution runner hid Go test stdout whenever dependency-download stderr existed. Diagnostics now retain both output streams so the native failure can be investigated; Windows remains unaccepted until the corrected native job passes.

P09 CLI return continuation: work/review sign freezes exact payload, signature, editable-draft digest and original-CID local provenance before network mutation. Fresh focal authority/Task/case CAS and own active key/policy are preflighted without implicit renewal/rebase. Send reconciles original signed acceptance before replaying identical bytes; prepared-only recovery never sends automatically. Confirmed acceptance is persisted before cleanup, and remote commit remains explicit when local cleanup fails. Cleanup verifies issuer/targets/request/disposition/closed obligation, preserves unconfirmed edits and removes only the corresponding regular contract file. Native child-process exit fixtures at receipt persistence and unlink passed on Linux. Actual HTTP Memory/SQLite/PostgreSQL execution delivery with lost response preserved edits and recovered without another return; independent CLI review signs/finishes after executor bearer revocation, deletes its file and SDK replays the original accepted decision. Unit fixtures reject copied-CID provenance, valid-local-MAC/invalid-signature and wrong/open receipts. Final frozen-source full race suite passed with explicit real PostgreSQL/restore environment (unchanged Core/storage cached), acceptance 41.013s and native Linux cleanup fixtures. Vet, actionlint, generated workspace schemas and the 102-event catalog passed. Extended P12 fault/direct/correction/platform acceptance remains required.

P12 native Windows diagnosis continuation: the hosted diagnostic job identified rejection of native relative backslash paths by a Linux-only confinement rule, Windows drive letters encoded as file-URI authorities for SQLite, and a mounted-secret fixture relying on Unix chmod/canonical spelling. Native relative paths now follow the platform local-path rules while absolute paths, volumes/ADS and symlink/reparse escapes remain rejected. SQLite emits a local-drive URI path. The mounted-secret fixture resolves its canonical spelling and explicitly establishes a restricted Windows DACL instead of weakening product secret validation. Linux targeted race and Windows cross-compilation passed; hosted native Windows execution is still required for acceptance.

The first hosted Windows fix passed native CLI filesystem/secret tests and SDK tests; SQLite acceptance still failed at open. Dependency inspection showed that the Go SQLite VFS receives the parsed filename directly and does not remove the RFC-style leading slash before a drive. The DSN now uses an escaped opaque file:C:/ URI to retain the native drive path, with all pragmas unchanged; the native rerun remains decisive.

Hosted Windows now passes native Go CLI/SDK/protocol checks for the opaque-drive fix. The next failure was skill frontmatter checked against LF after Git autocrlf produced CRLF. Repository attributes now keep text LF on all platforms, preserving canonical skill and embedded migration bytes; the integrated P09/native installer rerun is required.

Integrated native Windows Go checks including P09 returns/cleanup passed. The remaining installer test expected POSIX absolute strings on Windows even though installation/update/edit protection all passed; expected discovery roots now use the native absolute-path representation without changing their documented root mapping. The full platform receipt remains pending until every check and source-identity build passes.

P10 activation foundation: Namespace CAS supports contracts_v1 -> draining_to_signed_v2 -> signed_contracts_v2. Signed drain clears the earlier writer acknowledgement and refuses new unsigned acquisition. Activation rechecks live legacy claims, unsigned contracts, matching persistent issuer and an explicit Namespace acceptance policy under the existing transaction/coordination guards. Read-only administrative protocol_preflight is advisory and shared across transports. Authenticated signed-phase operator intentions use the immutable original-CID operation registry, without requiring an agent signing key or changing v1 fingerprints. Schema-2 protocol get/preflight/set/recover persists exact phase/version/reason/acknowledgement before sending and reconciles an uncertain response before any identical replay. Targeted Memory/SQLite/PostgreSQL cutover tests passed (SQLite 2.790s, PostgreSQL 1.899s); the actual HTTP/CLI journey also passed on all three adapters (33.601s), including activation response loss and recovery without another mutation. Full frozen-source race suite passed with actual PostgreSQL/clean restore (220.613s), SQLite (131.714s), then the final guard-order source passed full race again (PostgreSQL 183.647s, SQLite 125.448s, acceptance 54.303s), vet, actionlint and catalog. Namespace is locked before Outcome in protocol preflight, matching activation order. Final CLI-only precision/printable-error changes passed the CLI suite (7.804s), complete acceptance (40.291s), boundaries and CLI vet; quoted local CAS above the JavaScript limit and confirmed target/version validation have regression tests. Workspace v1 migration, historical upgrade/restore, issuer recovery and P11–P13 remain pending.

PR #51 integrated Windows portability: the full native Linux/Windows workflow run 37874587697 passed Go CLI/SDK/protocol, five-skill installer and executable identity checks. Receipts identify synthetic PR merge source 4080b609ae2583645fabb538395e8d0b0f28f4ad; this is executed native evidence for that source, not final-release proof. Earlier hosted billing refusals are historical; current runs execute.

Local Compose onboarding: the owner requested a copy-only .env.example setup. A committed development template provides a localhost bootstrap credential, valid Namespace ID and persistent SQLite defaults; Compose accepts a configurable localhost port and optional protected runtime signing references. PostgreSQL remains opt-in. Config interpolation and actionlint pass. Actual non-root image build/start, readiness, frontend, browser-session login, Namespace discovery and authenticated create passed; restart retained the original Outcome and idempotent receipt. The sandbox required a temporary build-only host-network/proxy/CA override, outside the repository; normal Compose settings remain portable. The immediate post-restart probe ran before readiness and was retried successfully; it is not counted as a product failure. Signing activation remains explicit and no shared issuer key is shipped.

P10 workspace migration continuation: read-only bounded inventory preserves exact draft/journal/receipt bytes and reports prepared/uncertain intentions and accepted acquisitions without materialization without network calls. Explicit separate schema-2 destination checks current approved server/Namespace/CID and remote original contracts, freezes a bounded profile migration intention and publishes explicitly legacy_unsigned single documents. Source files remain untouched; neither issuer nor agent signatures are fabricated. Recovery preserves source/target edits and reconciles exact publication before clearing the marker; repeating a verified migration recognizes existing matching files without a new intention. Draft interpretation uses the observed bytes, not a second read. Generated unsigned-local schema is packaged separately from signed contracts. Linux native child-process death after publication, original materialization, changed source/target recovery and decimal counters above JavaScript precision passed race tests (CLI 1.521s); actual HTTP/CLI conversion plus signed activation/execution/review passed Memory/SQLite/real PostgreSQL (35.679s), including refused signed mutation of legacy imports. Full frozen-source race suite passed with real PostgreSQL/clean restore (236.833s), SQLite (161.677s), CLI (17.842s), acceptance (87.457s) and boundaries; vet, actionlint, 102-event catalogue and Compose interpolation passed. Hosted Windows execution is required for this source. Historical database upgrade/restore, issuer recovery and P11–P13 remain pending; no P10 completion claim.

P10 database compatibility continuation: migration checks the highest persisted
schema under its transaction/ PostgreSQL advisory lock and rejects a database
newer than the compiled migrations without altering history. Runtime also checks
schema compatibility before bootstrap/serving when migrations are disabled.
Targeted SQLite and actual PostgreSQL future-schema/older-upgrade race checks
passed (6.957s/3.535s). Runtime restart/future-schema race tests passed (4.531s). Full race
regression passed with real PostgreSQL/restore (303.254s), SQLite (180.163s),
acceptance (79.802s) and CLI (18.613s); runtime passed (47.088s) on a sequential
package rerun with unchanged Core/storage/acceptance tests cached. The initial
parallel run timed out in the existing 30-second go-run MCP subprocess discovery
and is not counted as a full pass. Vet, actionlint and catalogue passed. PR #55
hosted browser, packages, full verification and native Linux/Windows passed. This adds forward protection, not retroactive protection to released
v0.2.0 executables; actual old-binary and historical dataset acceptance remain P10.


P11 agent guides started: all five portable skills now select live Namespace
protocol explicitly. Signed work uses protected schema-2 named profiles,
focal agent projections, distinct execution/review obligations and exact-CID
recovery; contracts_v1 instructions are explicitly retained only for legacy
Namespaces. Progressive bundled references explain immutable issued documents,
editable single-file drafts, quotas/partial batches, supervised leases, signed
acceptance versus quality, explicit trust rotation and unsigned migration.
No skill installs credentials, starts agents or certifies runtime integration.
All five installer tests passed: discovery roots, repeatable all-agent install/update,
protection of local edits, progressive references/frontmatter and actual main-skill
MCP tool names. UI/docs alignment remains required; this is not P11 completion
or final native/consumer acceptance.

P11 guide alignment removes stale claims that schema-2 review, lease maintenance
and sign/send/finish are unimplemented. Client/skill READMEs explicitly separate
legacy recipes from the protected signed flow; signed workspace docs explain
immutable original-CID returns and receipt-bound cleanup, distinguish executor
acceptance from review/quality, and retain native-keyring/final-source gates.
Registry publication is not inferred from source guide changes.
P10 issuer recovery design started: ADR 028 records the missing host-managed
replacement path required by T14/T87, separate from tenant administration and
without rewriting old signed facts. Implementation must retain persistent
ServerID/public history, freeze CAS and provenance, order global issuance against
replacement, and explicitly update client trust without breaking pending returns.
No recovery command is implemented by this design note; acceptance remains open.


Issuer recovery foundation work in progress: pure host composition proves the
approved replacement signer before its transaction and rejects tenant identity
contexts. Memory/SQLite/PostgreSQL persist immutable public issuer history and
frozen recovery receipts, preserve ServerID/instance creation time, CAS the
predecessor and replay a receipt without reinstalling a superseded issuer. SQL
shared issuer reads order issuance against the exclusive recovery lock. Targeted
race/clean restore tests passed (SQLite 4.106s, PostgreSQL 16.661s), verifying old
public proofs after clearing the old private key and rejecting stale configuration.
The source host executable command and paginated public trust projection are
implemented; actual runtime restart/HTTP tests passed SQLite and PostgreSQL
(4.215s), including refusal of silent replacement, wrong declared pin, exact
receipt replay and private-seed absence from the receipt. Client pin updates,
actual signed issuance/return after rotation and complete regression remain
unimplemented or unaccepted. This increment is not integrated and is not P10
completion.



PR #56 integrated workspace migration together with PR #55 compatibility checks.
The combined source d3e9d9e passed actual Memory/SQLite/PostgreSQL signed CLI
acceptance (48.186s), migration/precision race checks (1.959s), and runtime
schema/MCP subprocess checks (7.695s). All hosted browser, package, full
verification and native Linux/Windows checks passed on its synthetic PR merge;
these receipts do not certify a later final-release source.


P10 issuer recovery/client trust continuation: schema-2 `profile trust` verifies
explicit independently approved current fingerprint and the unchanged origin,
ServerID, Namespace, Principal and original CID. One atomic profile append retains
old pins and authenticated bounded prior binding MACs; contract bytes, signatures,
request IDs and pending intentions are unchanged. No trust RPC holds the profile
publication guard. Native Linux child-process exit after publication, `.yml`
mutation, repeated approval without rewrites, tampered lineage and moved-origin
rejection passed targeted race tests. The actual HTTP/CLI journey now rotates
after preparing a return, rejects stale issuance and unapproved trust, clears the
old private issuer, approves both local profiles, sends the original return under
the new issuer and recovers a lost response before independent review. This
passed Memory/SQLite/real PostgreSQL (42.384s). Shared recovery/restore tests also
verify reused-fingerprint parity and that replaying an old host receipt after a
second rotation never reinstalls its retired key. The host command checks schema
compatibility even with migrations disabled, before mutation; future-history
sentinels remain untouched. Combined targeted race passed SQLite 3.023s,
PostgreSQL 20.374s, runtime 5.592s and CLI 2.207s. Full regression, hosted native
checks and integration remain pending; this does not complete P10 or P12.


P10/P12 T15 readiness continuation: host readiness observes the current instance
issuer and whether any Namespace requires signed issuance in one read snapshot.
A signed runtime without a configured signer and a stale replica after explicit
issuer recovery report HTTP 503; ordinary unsigned startup remains ready without
a signing key. Authenticated historical trust remains readable and liveness is
independent. No key is generated silently. The actual SDK fixture provisions a
valid credential policy/agent key, explicitly activates signed protocol, restarts
without the issuer and verifies readiness/liveness, unchanged public trust,
negative issuer preflight and explicit new-issuance refusal on SQLite/PostgreSQL.
Targeted race passed (10.827s); initial fixture runs lacked an enrollment grant,
used a too-short idempotency key and expected readiness in the trust resource
rather than protocol_preflight. Those failed fixtures are not counted as passes;
the product assertions remain intact. Broader runtime regression and hosted
gates remain required before integration; historical dataset and P11–P13 remain
open.
PR #57 integrated host issuer recovery and interruption-safe client trust on
master f06d63c. Full frozen-source race suite passed with real PostgreSQL/clean
restore (214.673s), SQLite (126.291s), runtime (47.389s), CLI (8.862s), actual
signed acceptance (41.944s) and architectural boundaries (6.441s). Vet,
actionlint and catalogue passed. All hosted browser/package/full/native checks
passed. Native Linux/Windows receipts from run 37885413406 identify synthetic PR
merge source 78bf2e2d005ce64d02032e75608c823bd9730450 and version 0.2.0;
these are incremental source-specific proofs, not final-release certification.
P10 historical dataset/old binary and P11 UI/P12/P13 remain required. P11 guide
work proceeds against stable signed interfaces while those independent gates
continue; no wave-completion claim is inferred from this merge.


P11 signed UI projection work started: the workspace observes live Namespace
protocol, shows a public issuer/credential/key/onboarding panel, and reads focal
signed execution/review cases instead of routing signed Tasks through legacy
contract queries. Pending review, acceptance versus quality and protected-profile
next steps are explicit. Legacy execution/raw Task assessment actions and signed
proof-bearing commands are excluded from browser forms; a fresh protocol read
before mutation blocks a stale-form substitution. Parent planning/assessment
remains separate. Browser fixtures use the actual protected CLI with separate executor/reviewer
Principals, exact accepted material and both approved/changes_requested decisions.
Corrections expose accepted findings on existing obligations; no unsigned browser
mutation replaces review. All six browser journeys passed (30.1s), including
legacy cutover and planning/assessment regression. Native onboarding remains
explicit through profiles; runtime pilot and final source acceptance remain open.

PR #58 integrated the five signed-protocol skills and client-guide alignment on
master c91c694. All hosted browser/package/full/native Linux/Windows checks
passed on combined source 129797f. Installer acceptance is not certification of
execution in every consumer agent; P11 UI and final operational acceptance remain
open. T15 readiness source 2941bba passed the complete runtime race suite
(51.364s), actual Memory/SQLite/PostgreSQL signed acceptance (43.777s), vet,
actionlint and catalogue. Merging the guide baseline changes documentation only;
new hosted source checks remain required for the readiness PR.

P10 T81/T82/T83/T88 actual historical fixture: the pinned v0.2.0 executable
produces schema-19 unsigned contracts/material/criteria/seven receipts. Closed
SQLite backup API and PostgreSQL custom dump restore into clean targets; schema
26 upgrade preserves original rows, WorkItem versions and exact replay results
without historical signatures. Premature activation with an active v1 contract
is refused, followed by explicit revocation/cutover. The exact old process may
bind but returns readiness 503 and rejects unsigned acquisition with unknown
work protocol (422), committing no history; full startup refusal cannot be
retrofitted and deployment must retire old writers/revoke database credentials.
Both local drivers passed on service source 5461815. Initial fixture copied
SQLite without closing a reader/checkpointing WAL; using the backup API fixed
the fixture, not product code. Its initial guessed error status 400 was corrected
to the observed precise compatibility rejection 422/invalid_argument. CI now
builds both exact sources and uploads provenance; hosted validation pending.
Signed pending-state restore and final P12/P13 acceptance remain separate.
P11/P12 signed transport consistency: Namespace protocol CAS counters in signed
trust are quoted separately from the literal protocol-2 marker; signed schema
counter types match exact decimal metadata while unsigned encoders/schemas stay
unchanged. Max uint64/typed-client and protocol-marker tests passed. Application, command
and SDK race suites passed (2.322s/1.080s/2.218s); actual signed Memory/SQLite/
PostgreSQL acceptance passed (35.559s). OpenAPI regenerated without altering
v1 command encoders. Hosted/native checks remain required on this source.
PR #59 readiness integrated after all five hosted checks passed on 95e3921;
master merge 5461815. Historical upgrade/restore and final release gates remain open.

P12/P13 installed signed package acceptance now exercises protected enrollment,
signed execution checkout/show/finish, independent review finish and both
recover operations after cleanup outside the source checkout. The fresh Codex
pilot identified stale CLI/skill protocol metadata and missing explicit review
draft examples; preferred protocol/schema now advertise v2 while supported
legacy arrays preserve v1 compatibility. Five guides document exact review
decisions and criterion assessments. Required final source/platform checks and
release version preparation remain open.
Local development offline npm install completed the signed workflow and verified
Task DONE after independent review; all five skill installer tests passed and
the CLI version contract passed race (1.041s). The invalid review enum diagnostic
now lists exact allowed values. Local artifacts built from a dirty development
checkout are not final immutable-source release proofs; hosted checks must
repeat acceptance on the committed source.
P12 native keyring gate started: platform CI now requires an actual OS credential
service. An isolated Linux DBus/Secret Service fixture and native Windows
Credential Manager run protected reference read/write, exact profile MAC and
signing-key fingerprint checks, signing/verification and child-process reads. The actual executor/reviewer
HTTP/CLI lost-response fixture selects those native references in platform CI,
without passing credentials/seeds through child process environments.
Randomly named synthetic entries are deleted; no plaintext fallback, private
process arguments or repository secrets are introduced. Native receipt verification
rejects proofs without the executed keyring gate. Local Linux lacks the daemon;
actual native execution remains pending until hosted jobs finish. Native receipts
now list actual passing Go test count and individual skips, including unavailable
PostgreSQL on client-only hosts, instead of implying every suite ran. Local
Memory/SQLite fallback transport passed (24.272s); PostgreSQL was explicitly
skipped there. Distribution gate unit tests and negative secret redaction pass;
those are not counted as real local keyring execution.

The first native keyring jobs reached actual protected reads and the real
HTTP/CLI journey on Linux/Windows, but the standalone signing fixture omitted
the required signer_key_id from its payload. Both failed and produced no accepted
platform proof. The fixture now supplies the exact selected identity; signing
validation is unchanged and the native gates must rerun.
P12 T86 pending-state restore started: the existing authenticated signed
execution/review/correction journey now takes clean SQL backups at each pending
authority stage, restores into clean targets and resumes the same workflow.
Version/fencing/lease metadata and exact authority envelope bytes are compared
across restore, then delivery, review takeover, correction and approval continue
against restored storage. Memory is explicitly excluded from database restore
claims. Both clean restores passed with the complete authenticated correction/approval
journey: SQLite race 13.231s, PostgreSQL race 30.151s with three empty target
databases. The initial correction-stage fixture used the already revoked
original executor to read metadata; it now uses the current authorized
correction executor, retaining denial assertions. Hosted checks remain pending.

P12 signed workload matrix covers 1/4 Outcomes, 2/8 concurrent consumers and
0/16 fully accepted historical Tasks per Outcome, with independent attestation
review and exact original approval replay. All 16 SQLite/PostgreSQL measurements
passed locally; issuance remained bounded. DONE is absent before review and
exactly one completion event/outbox delivery is committed after approval/replay.
The initial real PostgreSQL workload failed on unclassified serialization errors
in signed quota/read queries. Storage SQL boundaries now map those failures to
transaction_conflict; three repeated multi-Outcome race runs passed (67.870s)
without replacing CID/envelopes or relaxing guards. The final full matrix passed
in SQLite/PostgreSQL (PostgreSQL 98.285s, non-race measurements). Updated query
error boundaries require fresh source verification. The report includes honest
SQL-count/data-density limitations; no production latency/cost SLA is claimed.

The actual managed Codex pilot used fresh executor and reviewer task histories,
distinct profiles/Principals and the installed npm skill. Four actual Node tests,
25 independent whitespace checks and five invalid inputs passed; required
criteria were independently met, Task is DONE and both contract directories empty.
Go binary VCS metadata records separate exact development service/client sources,
not final coordinated release evidence. No OS privilege isolation or provider
token/billing savings is claimed. T01–T96 traceability maps tests/resources and
explicit assertion-audit/experiment gaps without treating test names as scenario
acceptance. CI now requires all 16 signed measurements on its committed source.
P13 separates coordinated verification from registry publication. Manifest
integration runs all four same-source workflows and uploads hash-checked assets,
actual native receipts and package acceptance. Real registry publication requires
explicit workflow_dispatch publish=true, so finishing the plan does not silently
publish a version. Registry credentials and final source acceptance remain separate.
Existing hosted Linux/Windows execution replaces stale billing-only claims in
release instructions; no registry publication is claimed.
Native keyring gate rerun passed all five hosted checks on PR64 head 6a26bcc.
The downloaded Linux/Windows receipts explicitly record native_keyring_executed,
actual passing counts and individual skips. PR62 historical upgrade and PR63
pending execution/review/correction restores are integrated after their hosted
gates passed. This documentation merge requires a fresh source check before
PR64 integration; final release receipts still require the final combined source.

P12 T60 now has real Linux ENOSPC acceptance: a disconnected disposable Docker
container supplies a 1 MiB tmpfs, capacity is exhausted, receipt persistence
fails with ENOSPC, exact contract/profile/artifact bytes remain unchanged, then
freeing only owned filler capacity allows receipt-driven cleanup without a new
return. Local Go1.27.2 execution passed. The first runner retained mktemp mode
0600 on its test binary; explicit chmod 755 fixes container execution, without
changing product file protections. Ordinary/native Windows suites explicitly
do not certify this Linux capacity experiment. CI requires the actual named
PASS and retains the log. Permission-denied/process-death tests remain separate.
P12 T32 signed replica-clock experiment passed three PostgreSQL race runs
(3.058s) on Go 1.27.2. Two service instances use clocks +/-365 days; acquisition
and issuer authority deadlines use database time, renewal remains current,
takeover preserves expiry and increases fencing, and old authority is denied.
This is an actual shared-database test, not a claim of separately deployed hosts.
Existing unsigned clock tests remain intact; final source/platform gates apply.
The first hosted signed-load gate exposed generated-source drift: signedgen
regenerated v1-based signed work storage without the new SQL error boundaries.
The generator now preserves the signed-only mapping; v1 source is unchanged.
A Go 1.27.2 targeted PostgreSQL clean-restore check hit its 40-second pg_dump
timeout on the long-lived test database catalog; rerun uses a new owned disposable
database instead of changing the timeout or deleting existing fixture data.
P12 active fuzz campaigns execute canonical JSON/DSSE decoding and restricted
YAML parsing, with two workers and a 10-second budget each. Go 1.27.2 local
source 0ee312f passed actual mutations: canonical 21,915, envelope 91,292,
YAML 45,747. These are bounded runs, not exhaustive security proof or a coverage
percentage. CI now rejects seed-only/zero-execution evidence and uploads raw
Go JSON plus target counts/source/compiler. Hosted final-source campaign remains
required; no parser/security behavior was weakened to pass.
P12 T95 measurement tooling validates controlled A/B/C task and quality scopes,
identical B/C models/harnesses, decimal costs/provenance and retained failed,
review/correction/integration attempts. Unknown usage remains null; no bytes-to-
tokens conversion or fabricated savings. Five negative/aggregation test cases
pass. Managed Codex C succeeded, but A/B controls and provider usage/billing are
not exposed here; comparative savings remain unmeasured and explicitly pending.
P12 T34 exercises real SQLite/PostgreSQL rollback boundaries for artifact,
evidence, submission, contract, Task, return fact, Outcome revision, acceptance
fact, Domain/Integration Event, idempotency reservation/completion, durable
operation recording and review-case insertion. Late receipt-signing and pre-
commit failures retain every optional SQL capability/clock through concrete
unit embedding. All affected table rows compare exactly with the baseline; the
original immutable signed request succeeds after faults are removed. Go1.27.2
race suites pass on both SQL backends. This is before-commit fault injection,
not proof of network loss after commit or every statement inside each adapter.
Installed signed acceptance now drives an actual subprocess correction round:
planner HTTP commands, executor/reviewer CLI processes, changes_requested, fresh
contract with exact --previous-review acknowledgement, explicit frozen finding
response, required original criterion assessment, Task DONE and empty profiles.
The first fixture used generic --next, which intentionally skips corrections;
it now acknowledges the original case rather than weakening acquisition guards.
The fixture used the wrong criterion DTO field and leaked a block-local variable
into final assertions; both fixture errors are corrected without relaxing strict
schema/cleanup checks. Five progressive guides and the human UI now show the
required correction acknowledgement. This automated correction scenario is
distinct from the managed Codex pilot, which accepted its initial delivery.

Signed SQL boundary regression rerun passed with Go1.27.2 and a new owned clean
PostgreSQL database (SQLite 17.945s, PostgreSQL 12.789s), resolving catalog-only
pg_dump timeout diagnosis without changing restore deadlines or existing data.
P12 T90/T91 signed-load acceptance also checks one completion worker recovery
row per backend: leave the committed signal leased without acknowledgement,
reopen storage connections, reclaim the exact body/ID after expiry with increased
fencing, reject the old acknowledgement and finish without a duplicate signal.
Both real SQL race cases pass (SQLite 5.441s, PostgreSQL 4.380s). This models
worker absence and storage restart; it does not claim an OS kill experiment.
P13 coordinated 0.3.0 preparation updates VERSION, both npm packages and service
compatibility, release manifest, changelog and upgrade notes together. Existing
Namespace phases/unsigned history remain explicit compatibility paths, so this
opt-in protocol addition uses a minor bump. This release is prepared, not
registry-published or certified from parent-commit receipts. Remaining P12 PRs
are integrated; all same-source final gates must pass for the source selected for release.

The first 0.3 hosted source gate rejected stale OpenAPI release metadata (0.2).
Preparation now validates and updates that generated metadata before writing
coordinated version state; unchanged public paths remain exact and stale inputs
fail before consuming the queue. Release automation includes the OpenAPI file
in its commit and explicitly dispatches four verification workflows for its
bot-created PR, which GITHUB_TOKEN events otherwise do not start. Publication
remains separately explicit. The failing gate is rerun rather than disabled.

Combined integration preserves the independent feature PR heads/commits and
checks generation drift on their merged source. T01–T96 now map actual Go JSON
execution (including generated PostgreSQL counterparts and individual skips);
passing names never automatically accept all scenario assertions. Hosted final
source verification remains required. Historical progress entries below their
phase tables describe earlier states and do not override the current tables.

P12 managed Codex 0.3.0 pilot found a concrete invalid-material trap: a
`test_result` evidence draft containing measurement passed local parsing and
failed remotely after immutable freezing. Client execution/review parsing now
shares Domain measurement/type validation before network/signing; human command,
MCP/OpenAPI and packaged draft schemas expose the conditional rule. HTTP maps
`evidence_error` to nonretryable 422 rather than 500. Regression checks retain
exact original contract/profile bytes and prove no pending intention is created
on malformed material. The failed experimental fixture is preserved; this
change does not rewrite any previously frozen payload or manufacture acceptance.

P12 fresh Codex retry also exposed a redundant criterion-local evidence selection
requirement. Resolving explicit registered `criterion_local_evidence` now includes
that same evidence in the submission's general evidence set. Original signed
request bytes remain unchanged; unknown local keys are still rejected, current
criteria/assessment guards remain authoritative. SQLite/PostgreSQL direct/review
atomic return race cases pass without duplicated `evidence_local_keys`.

P13 npm/standalone packages now include the same generated workspace schemas as
the CLI archives, so npm agents need no source checkout to discover typed
editable material. Package metadata matches the live client signed-v2/schema-2
capabilities and retains explicit legacy compatibility; installed-package
verification checks schema inclusion and protocol metadata against the binary.

P12 T95 actual managed Codex A/B/C calibration is retained under
`docs/audit/controlled-codex-2026-10-09`: one common Unicode normalizer task,
fresh executor/reviewer histories for B/C, same inherited models/native harness
and 43 external checks for all strategies. Parent-observed wall times include
orchestration and C development/remediation; first malformed return and second
criterion-reference failure remain recorded. C recovered unchanged original
signed bytes after same-database/issuer upgrade, then received independent
revision-1 approval; Task DONE and both empty contract directories were checked.
Provider tokens/cache/tool counts/cost remain null, no Outcome achievement or
savings is inferred. The reporter now validates observed timezone-bearing timing.

P12 T72 bounded mixed local operation experiment executes actual three-intent
acquisition, original-CID lease preparation, accepted receipt recovery and
cleanup concurrently in one workspace. A separately held profile blocks only
its own acquisition; every independent path finishes before release. Exact
maintenance contract bytes, batch contents and empty accepted profiles are
verified. Three native Linux race repetitions passed; platform CI exercises the
same test on its actual host. Fixtures are explicit local signed receipt unit
data; SQL/network/load proof remains in the independent real-service gates.
Signed v2 has maintenance/return operations, not the legacy `sync` command.

P12 managed correction review found a second malformed-draft trap: one finding
contained both criterion_id and requirement_ref. Shared Domain structural
validation now rejects missing/ambiguous obligations before local signing or
network preflight; generated editable schemas and progressive skill guides
expose the exclusive choice. Race regression cases prove original profile and
contract bytes remain unchanged. A previously frozen rejected experimental
return stays preserved and is not relabeled as accepted.
A subsequent controlled non-approval attempt correctly failed the server guard
because it included criterion assessments. Editable review parsing now rejects
that combination before freezing; the progressive guides explicitly require
findings/reason for non-approval and assessments only for approval. The original
rejected requests remain preserved, with no fabricated receipt or acceptance.
The final managed correction review uncovered an immutable-source prerequisite:
complementary reviewer records need an existing delivered artifact version or
checksum. A commit mentioned only in evidence does not satisfy it. The CLI now
shares the server's source-binding guard before network/signing; guides explain
when to omit complementary records and use actual checks in assessment rationale.
The rejected review is retained, rather than modified after freezing.

P12 managed Codex correction is now actually exercised separately from A/B/C:
controlled incomplete delivery, independent changes_requested at the original
criterion, fresh executor correction response and fresh independent round-2
approval. Parent queried Task DONE (version9), unchanged revision-1 criteria,
clean tested commit, 43 external checks and verified cleanup for accepted CIDs.
Exactly three rejected review documents remain preserved with unchanged hashes;
these are not counted as pending cleanup of accepted obligations. See
`docs/audit/signed-codex-correction-2026-10-09.json` for exact source/provenance.
Shared OS/filesystem and unknown provider billing remain explicit limits.
Local Compose copied .env.example unchanged and passed readiness/liveness/UI
checks on its documented source, preserving existing projects and data.

## Human workspace UX/DX program — 2026-10-09

Owner requested execution of `docs/ux/original-plan-2026-10-09.md` against baseline `5ea787c`. This is a new program; prior backend acceptance is not frontend usability evidence.

| Lot | State | Acceptance |
| --- | --- | --- |
| UX-FND-01 | ✅ | Explicit 100-command inventory, copy inventory, glossary, journey contracts and repeatable datasets; human baseline not measured |
| UX-LOC-01 | ✅ | Official en-US catalog, shell, typed labels/UTC dates and user-content preservation; four presentation checks and six real browser journeys passed |
| UX-ACT-01 | ✅ | Explicit permission/state/protocol action registry; common action and revoked-authority browser checks passed |
| UX-API-01 | ✅ | Eleven-kind authorized bounded metadata search; Memory/SQLite/PostgreSQL 1,001 Task + Evidence parity, revocation/cursor/HTTP-MCP tests passed locally |
| UX-CMP-01 | ✅ | Typed server-searched reference pickers; 1,001-item, duplicate-title, keyboard and stale-cursor browser checks passed |
| UX-FRM-01–03 | ✅ | Dedicated forms and persisted Draft preparation checklist; explicit activation retained |
| UX-NAV-01 | ✅ | Six purpose areas, Work status filters, scoped deep links with reload/back/forward; browser checks passed |
| UX-PLN-01 | ✅ | Visual phases/milestones/references, draft diff, reason, publication and separate activation; immutable history browser check passed |
| UX-EVD-01 / UX-ISS-01 | ✅ | Exact-definition assessments and independent Issue/Blocker resolution passed real-browser journeys |
| UX-OPS-01 / UX-ADM-01 | ✅ | Permission-bound typed settings/advanced editors and copied host-profile handoff; legacy/unsigned/signed regressions passed |
| UX-CAS-01 | ✅ | Retained edits, explicit comparison/reconfirmation and identical-byte/key uncertain replay passed real-browser checks |
| UX-QA-01 / UX-REL-01 | ✅ automated / pending manual | 31 browsers, 14 web units, 712 Go test/subtest passes in both normal/race, real PostgreSQL, offline/reproducible packages and Docker Compose; representative users, screen reader and native zoom pending |

Owner confirmed that no representative users or manual screen-reader testers are available; that validation remains pending. Human baseline and representative-user/screen-reader measurements remain **not measured**. Automation timings cannot establish human comprehension or WCAG certification. No UX lot is complete solely because this plan or its inventory exists.

### Historical implementation checkpoints

The following paragraphs record intermediate slices. Current acceptance is in the table above and the final record below.

UX-API-01 implementation: additive `SearchReferences` Application query and bounded metadata port; typed HTTP/MCP adapters, all eleven operational reference kinds, revision-bound keyset cursors and authorized exact-ID resolution. Storage/query verification is in progress; completion not claimed. Fresh baseline browser recheck: six tests passed with writable XDG paths, recorded in `docs/ux/baseline-browser.json`.

UX-API-01 local acceptance: all three stores passed eleven reference types and 1,001 Task/Evidence datasets, exact-ID resolution, scoped cursors, Unicode semantics and cached credential revocation. Real HTTP/MCP query parity and empty-page telemetry tests passed. Human latency/comprehension remain unmeasured; automated query latency is separate. Search uses existing scope/ID indexes and bounded projections; no migration or domain change. PR #80 integrated after all five hosted checks passed (merge `dab7adb`). PR #79 foundation integrated after all five hosted checks passed (merge `0be1d7d`).

UX-LOC-01 starts after the accepted foundation and authorized reference query contract. Dynamic official copy and static shell fallbacks bind to `en-US.js`; dates explicitly label UTC, wire/user content is unchanged, and free text is no longer formatted as an enum merely because its value matches one. The technical command picker remains pending UX-ACT-01 and is not claimed resolved by translation.

UX-LOC-01 acceptance: six embedded-server browser journeys passed on 2026-10-09, including legacy leases, unsigned contracts, signed independent review, evidence contestation and mobile checks. Portuguese fixture/user titles are deliberately retained. Historical audit documents are exempt from official-copy translation.

UX-ACT-01 API/registry slice: effective scoped permission hints exposed through HTTP/MCP, explicit presentation surfaces and permissions checked against all catalog operations, lifecycle/material/protocol/lease-holder guards tested. Common-client form/picker wiring remains under browser acceptance. PR #81 English migration integrated after all five hosted checks passed (merge `49e33f4`).

UX human-flow slice acceptance: sixteen embedded-server browser tests passed on 2026-10-09 (ten common journeys plus six protocol/domain regressions). Dedicated Outcome/Objective/Task inputs, reference search, visual Roadmap, exact-revision verification, compound/independent impediment handling and immutable intent retries are implemented. Six-area navigation, Draft checklist, final accessibility/distribution gates and representative-user/manual screen-reader checks remain outstanding. See `docs/ux/dedicated-human-journeys.md`.

UX navigation/search slice acceptance: new six-area, Draft checklist, reload/back/forward/item and 1,001-Task global title/priority browser checks passed; compatibility journeys passed after purpose-navigation migration. Switching to Board invalidates outstanding List renders. Additive Task priority query passed Memory, SQLite and real PostgreSQL tests; HTTP/MCP permission-hint shape parity also passed. API access, CAS and domain state semantics remain unchanged. Hosted PR #83 browser/package/Ubuntu/Windows checks passed, while core CI was interrupted at a Docker Hub anonymous-pull rate limit; a failed-job rerun was requested. No CI application failure is inferred from that infrastructure response.

UX operation/accessibility slice: 31 real-server browser tests and 14 unit checks passed locally. Official copy now covers common modules and additional static shell controls; authored content is preserved. Typed participants/external context grant no authority. Objective parent relationships, linked progress, entity-specific empty states and Task post-create detail are implemented. Host-profile clipboard handoff, revoked read/write/replay prevention and late-context failures passed. Axe reported no serious/critical violations in tested login/Overview/Task/picker states after contrast correction; keyboard, 390px and 200%/400% equivalent viewport reflow passed. Native browser zoom, representative-user usability and manual screen-reader checks are not claimed measured. Full backend/distribution and final documentation gates remain outstanding.

### Final executable acceptance — 2026-10-09

PRs #79–85 are integrated. All executable correction lots are implemented and the final documentation/capture slice records 31 real-server browser passes, 14 web unit passes, 17 package/release contract passes and full Go normal/race acceptance (712 test/subtest passes each, 18 passing packages, explicit opt-in/environment skips). Memory/SQLite/real PostgreSQL, permission/query parity, cursor/revocation/CAS and signed independent review passed. Vet, module hygiene, generated contract drift, event catalog, actionlint and report checks passed. Actual full-disk recovery passed separately. All five distribution archives were byte-identical across separate source paths; offline npm installation exercised service/client/skills and signed host profiles. The real Dockerfile/Compose default SQLite path passed startup, English embedded assets/CSP, authorization/HTTP/MCP and persistence across restart after copying `.env.example` unchanged (port 18109 and build-only cloud CA override). No runtime/schema/release version change or publication is implied.

Accepted executable source: `315fcadbcb4a6acc614cf908e04b5d55d54e9db9`; final documentation adds no application behavior. Source-bound reports and English current screenshots are in `docs/ux/final-acceptance.md` and the historical gallery at `ec9aa1ecc836aa7b239f489cd70f8c6ad75930f1`; current screenshots are in `docs/design/editorial/README.md`. Hosted browser/packages/Ubuntu/native-Windows passed for #83–85; core jobs were refused by Docker Hub anonymous-pull rate limiting, not treated as green. Owner permitted continuing despite refused jobs; local required gates passed before integration.

**Pending human validation:** representative-user comprehension/unaided creation and baseline timing, manual screen-reader use and native browser 200%/400% zoom. Owner explicitly confirmed no representative users or screen-reader testers are available and requested this be recorded as pending. Automated axe/keyboard/reflow and screenshots do not certify WCAG or human usability percentages. Implementation is ready for that validation and human/agent operation; the pending validation is not marked complete.

## Native `wos skill` CLI — candidate for 1.0 hardening (2026-10-12)

**State:** implemented as a Go distribution/client change; local standalone
installer tests passed. Exact-source hosted Go/distribution verification and
reviewable PR integration are still required before marking this gate complete.

- The `wos` Go binary imports the canonical Go-embedded agent skills bundle.
- `wos skill install|update|list|doctor|uninstall|version` supports codex,
  claude, hermes, openclaw, all and explicit generic destinations, as project,
  global or explicit paths. It makes no network/API calls and needs no Node.
- Native and optional npm installers share managed hashes and preserve
  user-modified/unmanaged/symlink targets; JSON map insertion order is not
  interpreted as a content modification.
- Required acceptance: Go package tests, full Go build, actual packaged
  binary skill install/doctor/uninstall, cross-installer compatibility and
  offline reproducibility on the immutable source commit.
- This does not certify native agent execution in every runtime, provision
  credentials or activate signed Namespace protocols.


## Native WOS CLI npm distribution — 2026-10-12

**State:** proposed for v0.4.0; exact-source CI and real npm registry publication not yet accepted.

- Add `@woobeai/wos-cli` with Go `wos` and `wosctl` executables for Linux/Windows amd64 and a minimal platform-integrity npm launcher.
- The standalone native `wos skill` installer is still executed by Go; no Node.js code performs skill installation or accesses the WOS server.
- Require native Windows/Linux execution of skill install, doctor and uninstall, verified native receipts, coordinated version metadata and reproducible offline npm installation.
- For the 0.4.0 candidate, publish this CLI package only through a source-gated GitHub workflow after all backend/browser/CLI/distribution checks; do not mark service, GHCR, skills, or GitHub release as published.
- Future releases use the existing explicit coordinated publication workflow; no default auto-publication of future versions is authorized.

### npm CLI publication scope correction — 2026-10-12

The first 0.4.0 publication attempts authenticated as npm account `woobeai` but `@a1b3rt0m3rcad0/wos-cli` failed with `E404 Scope not found`. The CLI-only artifact now uses `@woobeai/wos-cli`; server and separate skills package retain their existing names without asserting npm ownership. The obsolete fixed-SHA failed recovery workflow was removed. The new package must pass exact-source complete CI and publish with independently verified registry integrity before this entry can be marked complete. This change does not retroactively mark the failed attempts as successful.

## Stable 1.0 hardening — owner plan, 2026-10-12

Source plan: `docs/stable/original-plan-2026-10-12.md`, baseline `40d1703` / 0.4.0. The owner's current naming instruction overrides the historical CLI scope: **`npm install -g wosctl@latest`**. The package remains the native Linux/Windows amd64 Go CLI; WOS does not execute agents. Apache-2.0 remains the established license. Version 1.0.0 is not prepared or published merely because the plan exists.

| Gate | Current state |
| --- | --- |
| P0-01 CLI name/publication | Public wosctl 0.4.1 passed source/integrity and installed Linux/Windows consumers; 0.4.2 was published 2026-10-13 at b572269; operational changes remain unreleased |
| P0-02 stable contracts | Product metadata/OpenAPI synchronization, public-schema fingerprint freeze and compatibility regression gates integrated; unfinished 1.0 assertion audit is separate |
| P0-03 signed assertions | Authority/live-policy/enrollment slices and actual Codex pilot integrated; remaining T01–T96 assertion audit still open, with no blanket acceptance claim |
| P0-04 license | Apache-2.0 retained and release/distribution license consistency checks passed |
| P0-05 protection/release | CI-gated immutable coordinated 0.4.1 release public; active master rule requires PRs, preparer respects it, publication requires exact-source core/browser/packages/native gates |
| P1-01–04 operational/security | Bounded capacity/retention/restore reference and negative-authority evidence integrated; client report adds real API/SDK, Compose restart/new-file restore and native signed qualification without production SLA claims |
| P1-05 human validation | Remains pending under prior owner instruction; no participants or manual screen reader available; automated success is separate |
| P1-06 independent consumer | Actual public installed 0.4.1 signed consumers passed Linux/Windows; current fixes also passed source-bound native image qualification, separate from current operational-candidate acceptance; 0.4.2 publication completed 2026-10-13 |

Owner clarification: no scoped replacement and no separate public npm service/skills package. `npm install -g wosctl@latest` supplies `wos`/`wosctl`; skill installation remains native `wos skill install --agent codex`. The active build/publication/version pipeline is being reduced to that one npm identity while retaining native archives and container deployment. Historical unpublished Node package directories are compatibility fixtures, not distribution claims.

Owner operation-name clarification: all public operations use `wos <command>`. The unified Go wos root now composes server/database/skills plus the existing native profile/work/review client groups; wosctl remains a compatibility alias. Native/browser/distribution tests must exercise the unified root, not merely install a renamed npm wrapper.

Unified package verification reached signed independent review/correction successfully; corruption testing now replaces an owned binary atomically to avoid Linux ETXTBSY while preserving checksum rejection. The actual browser handoff regression selects `wos` instructions and invokes the unified Go binary.

Actual offline unified package acceptance passed, including unsigned checkout, signed execution/correction/independent review, native skills/all/generic destinations, shutdown and corruption rejection. Further entrypoint checks cover global flags before commands, matching copied `wos --profile … work/review …` instructions, and the existing doctor client operation.

### C — enrollment boundary assertions (T10)

The shared enrollment test now independently verifies correctly signed wrong-purpose/wrong-Principal proofs, another authenticated Principal attempting the same challenge, and a separate approved challenge with a persisted elapsed deadline while the credential is still valid (PostgreSQL uses authoritative transaction time). Each must fail with forbidden, leave enrollment unconsumed/version unchanged, and leave its key absent. The original approved challenge must still succeed afterward. Local Memory/SQLite/real PostgreSQL execution passed all 22 tests/subtests without skips; exact-source hosted acceptance is still required. The elapsed challenge is a controlled persisted fixture, not a claim that PostgreSQL obeys the injected Go clock.

### B — public contract candidate freeze

While A awaits hosted/public registry gates, independent contract hardening proceeds without declaring A accepted. MCP serverInfo now receives the actual service build version rather than 0.1.0-dev. Both OpenAPI metadata versions follow product SemVer and release preparation synchronizes them. Exact public API/CLI YAML schema fingerprints and license/one-package checks detect unreviewed drift. Compatibility matrix retains unsigned history, explicit signed cutover, legacy alias and exit semantics. Apache-2.0 is retained; historical service/skill manifests are private. VERSION remains 0.4.0; stable publication and manual validation are not asserted.

Contract-candidate local verification passed 22 package/release/compatibility checks and the real MCP client test against Memory/SQLite/real PostgreSQL. The runtime now advertises the injected product version. The source remains 0.4.0 and stable gates are not inferred from these focused checks.

### C — signed assertion audit, first authority slice

The audit adds 21 separately named, correctly signed return probes across direct/independent-review and with/without required criteria. Probes change server, Namespace, Outcome, Principal, a registered foreign Principal's signing key, contract, task, execution, fencing, specification/authority digests, contract/lease/work/policy versions, purpose, completion intent, documentary scopes and local-key bindings. Each probe verifies the signature independently, requires application rejection, and verifies the original active grant, task version/lifecycle and Outcome revision remain unchanged. The original accepted return then executes, so negative requests cannot consume the real obligation.

Local Memory/SQLite/real PostgreSQL acceptance passed 354 tests/subtests. Generated PostgreSQL tests remain aligned. This is an explicit assertion-level slice of T04/T18–20/T24/T33–37; it does not mark all T01–T96 assertions accepted or imply the manual human and registry gates passed. Hosted exact-source verification is still required.


### T89 — actual fresh-context Codex pilot evidence

The authorized managed Codex pilot completed independent execution/review, real changes requested, a fresh correction contract and final approval with public `wosctl@0.4.0` source 347e4a4. The corrected frozen artifact passed 21 independent checks; four acceptance receipts and two submission digests were verified independently, final work is done, contracts empty and recovery clean. The first invalid signed review failed HTTP422 and remains preserved; it is not silently counted as success. Sanitized report and byte-identical artifacts are archived under docs/audit/codex-managed-pilot-2026-10-12. Shared filesystem is not privilege isolation, artifact refs are empty and provider token/cost figures are null. This closes the bounded actual managed-agent pilot evidence on that public source, not the final-source 1.0 acceptance gate or human validation.

### Browser regression found during release hardening

PR #94's browser gate exposed a delayed search firing after explicit Work-status navigation and Load more, resetting the list to its first 25 items. Section rendering now cancels the queued search timer when an explicit render consumes the current filter state. Existing generation/render-ID guards continue rejecting stale in-flight responses. A deterministic queued-search/pagination regression and exact-source browser checks are required; this failure is not waived as a CI transient.

Pagination regression verification: the deterministic queued-timer test fails against the original accepted hosted binary (unexpected stale reference request), and passes against the corrected binary. Three repeats each of the regression and full workspace/pagination journey passed; complete local browser suite passed 32 journeys plus 14 web unit tests. Exact-source hosted gates remain required before integration.

### D — dense continuity measurements, first operational slice

The shared SQLite/PostgreSQL benchmark now accepts 100/1,000/10,000 WorkItems per Outcome and measures focal work context beside the full continuity projection, with allocation/response size and empirical p50/p95/p99. Local 1,000-item execution passed for SQLite and real PostgreSQL (100 samples per sequential query, GOMAXPROCS=3); each rich WorkItem has an actual criterion and explicit assessment. PostgreSQL rich snapshot mean/p95/p99: 82.48/120.33/136.92 ms; focal context: 15.35/20.91/28.78 ms, 2,225 bytes versus 32,643. Four-reader evidence is not claimed: this run used three readers and measured 24.36 PostgreSQL snapshots/s. SQLite reported 32 SQL statements/rich snapshot and 34/focal context; PostgreSQL SQL count was unavailable and is not invented. This demonstrates response caps do not bound full reconstruction; agents should prefer focal context and expand proof by reference. Larger-history envelopes, production percentiles, CPU profiles, restore/retention and final-source records remain separate acceptance work.


Dense capacity is now an immutable-source CI fixture: 100/1,000 tasks on SQLite/real PostgreSQL, with separate full/focal projections, empirical quantiles and CPU/heap profiles. The report rejects missing/zero/failing cases and changes to compiler inputs; CI output logs are not confused with source changes. PostgreSQL SQL counts remain unavailable unless pg_stat_statements is configured. Operational reference documents retention by data class, hourly backup and clean restore/upgrade procedures, explicit RPO/RTO targets rather than guarantees, metrics resets/missing production percentiles, external collector diagnostics and production secret/TLS boundaries. `.env.example` and the copy-ready local Compose flow remain unchanged. Reference target acceptance is limited to the measured unprofiled reads; final clean-source profiling and hosted gate integration are still required.


Clean-source capacity profiling passed source 09de362daaed7696970f760ba6fa7cdd7ab25e60: 16 real store/history/query cases with 100 samples each, plus CPU/heap profiles. Tracked receipt: docs/stable/capacity-2026-10-12.json. At 1,000 tasks, profiled PostgreSQL rich/focal p95 was 97.32/26.20 ms, SQLite 117.73/21.66 ms; three-reader throughput 26.24/22.83 snapshots/s. The reference focal target is 35 ms; unavailable PostgreSQL SQL counts and observed tail spikes remain explicit. Go/fixture CPU/RSS excludes the independent database process. No 10,000-item, signed-writer-at-1,000 or production-SLA acceptance is inferred.

Public registry wosctl 0.4.0 now matches the original source 347e4a492d50f69a1aeb0a0c0bb3fde255999ad0 tarball SRI. Read-only preflight allowed rerunning the original failed publisher with no duplicate upload. Run 38297440817 completed successfully, including actual public latest installation and native skill management on Linux and Windows. Full public signed-journey acceptance is an additional check in PR #94, separate from those original installer receipts.

### Owner correction — use the WKS CI-controlled release model

The owner requested the WKS model rather than additional GitHub settings steps. Branch protection remains an optional repository administration hardening action, not a prerequisite for this delivery. No effective master protection is asserted: the integration received HTTP 403 and the owner's earlier intent to configure it is not evidence of activation. Required acceptance stays enforced in the publication workflow's dependency graph on the exact source.

The WKS workflows were inspected directly. WOS now automatically publishes after a prepared release manifest is integrated and its core/browser/packages/native Linux/native Windows gates pass. Publication consumes the qualified artifact, not another build. Workflow dispatch remains for recovery. The owner authorization supersedes historical notes requiring a separate approval for every future publication. This does not automatically promote the current 0.4.0 source to 1.0.0 or mark pending human validation complete.

Initial wosctl publication run 38297440817 passed all source gates but public version 0.4.0 remained absent. Registry metadata shows a 0.0.0-stage placeholder, consistent with npm staged publishing. The original approved artifact was recovered and independently verified: source 347e4a492d50f69a1aeb0a0c0bb3fde255999ad0; SHA-256 b06b95561624f34354554186a4388b375f9e2fbe70659a96196481c05bbc7e15. No duplicate upload or rebuilt version is authorized by recovery. A read-only recovery workflow verifies the original source gates, native receipts and public integrity, then runs a clean installed consumer on Linux and Windows. Future failed metadata verification preserves original package bytes and a pending diagnostic receipt. Public consumer acceptance now includes authenticated signed execution, independent review, correction and recovery using the installed binaries outside the checkout. Execution is not yet asserted until the public version is approved/available.

Recovery verification locally accepted the real original hosted artifacts and native Linux/Windows receipts, rejected a foreign source and differing registry integrity without registry mutation. The installed original hosted tarball completed the signed executor/reviewer/correction/recovery journey outside source on Linux. Registry-installed acceptance remains pending; that offline execution is not represented as public npm installation. Release/compatibility tests passed 13 checks and actionlint passed the affected workflows.

PRs #92, #93 and #95 passed all five hosted checks and are integrated. Public wosctl 0.4.0 is SRI-verified with successful registry-installed native Linux/Windows skill receipts in run 38297440817. Full registry-installed signed journey is checked additionally through the read-only recovery workflow. Branch-rule setup is not an outstanding owner action under the WKS-model clarification.


### WKS automation follow-up: remove bot-PR settings dependency

After PR #94 integration, GitHub rejected automatic release PR creation: "GitHub Actions is not permitted to create or approve pull requests" (run 38305750972). The owner requested the WKS model without additional GitHub settings. Version preparation now validates the declared reviewed changes and records only allowlisted mechanical version/notes/manifest/OpenAPI metadata through a normal fast-forward master push. It uses no force push, administrative bypass or PR-approval permission. A concurrent master update fails safely and queues a new preparation from its push. GITHUB_TOKEN suppresses sibling push workflows, so the preparer explicitly dispatches exact-source release checks/publication. All backend/browser/packages/native gates remain prerequisites. This replaces the failed bot-PR step; no additional owner configuration is requested.

Read-only reconciliation run 38306230907 passed: original public wosctl 0.4.0 source/SRI/native receipts and actual signed executor/reviewer/correction/recovery consumers on Linux and Windows. The successful Windows fixture does not add production Windows server support.


### Client operational report — command discovery correction

The 2026-10-12 client report confirmed documentary integration but found that local --help and outcome context did not work. Local root/group help and a version/commit-bound CLI catalog now execute before workspace/credential/server access, through both primary wos and its client alias. Unsupported outcome context returns explicit legacy outcome get and signed work-show alternatives. CLI documentation now reflects the unified wosctl package and bundled native skill installer. Focused offline discovery/root-routing tests passed; signed-client qualification and exact-source hosted integration remain separate evidence.

### Generic client qualification — actual API commits and receipts

The supplied client report is archived unchanged in docs/audit/client-operational-report-2026-10-12.txt. A real authenticated HTTP/SDK test now loses the response after RegisterEvidence commits, revokes the old token, retries the identical intention with another credential of the same Principal/Actor, obtains the original receipt/evidence and rejects changed payload under the same key. Exactly one evidence remains. Evidence and a supports link cannot assess or conclude an Outcome; explicit authorized assessment still does not auto-achieve it, and an explicit conclusion is required. Memory/SQLite/real PostgreSQL passed. This exposed standard Go HTTP automatic mutation replay after a reused-connection failure; SDK mutations now disable transport body replay and leave reconciliation explicit. Typed compact receipt observations preserve result_omitted; the legacy full-response API refuses to silently accept missing results. Focused HTTP/SDK/CLI race execution passed 13 tests/subtests with no skips; independent review/correction, database-clock skew and issuer-history restore execution passed 13 tests/subtests on Memory/SQLite/real PostgreSQL with no skips. Native and hosted checks remain required; the client's private Lipo reports and desktop instance are not available in this workspace and are not presented as independently reproduced.


The combined CLI/SDK candidate source 5fdd6949c433d734816ac709a844e892486644d1 passed actual Compose startup from the public .env.example (owned host port override), authenticated documentary commit/replay/receipt, container restart persistence and backup restored into a new SQLite file. Its copied native binary completed approved enrollment, signed execution/independent review, changes requested/fresh correction, required assessment and receipt-bound cleanup/recovery outside the checkout. Source-bound sanitized receipts and supporting focused Go results are archived in docs/audit/client-qualification-2026-10-12. The SDK/core files match the qualified image source. Desktop localhost:8092 and private external reports are not certified; no production SLA or isolation claim is inferred.

### Release preparation respects active master rules

Run 38308328553 proved that master now requires a PR (GH013); the preceding direct metadata push was rejected and did not change master. Preparation now writes a source-specific release branch, creates a PR when allowed, or records the branch for the connected authorized agent. No rule bypass, force push or additional owner setting is required. A prepared branch is not a published release. Merging the metadata PR triggers the existing exact-source publication gates.

### C — live policy assertion gap closure

The assertion audit found that nominal direct/review floor cases did not independently demonstrate a policy changing after issuance. The shared signed-return matrix now verifies deployment and Outcome floors override a weaker Task policy, current credential hardening is enforced, relaxing current policy cannot weaken the frozen review floor, and another credential of the same Principal cannot replace the original owner. Each mathematically valid probe must fail specifically with forbidden (not parser/stale-policy failure), preserve original authority/task/revision, and permit the original correctly updated request afterward. Local Memory/SQLite/real PostgreSQL acceptance passed 418 tests/subtests, with no skips. This closes explicit T07/T38/T39 authority assertions; exact-source hosted verification remains required.

### Client operational report correction integration

PR #103 (local discovery), #104 (explicit SDK recovery/compact receipts and generic client qualification), #101 (enrollment), #102 (actual Codex pilot archive) and #96 (capacity/operations) are integrated after their source checks. All five recommendations in the generic client report have repository-level corrections/procedures and observable functional evidence. The private client desktop and its unprovided reports remain outside the tested target. The 0.4.2 metadata consumes the reviewed change declarations and passes release/compatibility validation. Historical status at preparation: public acceptance of 0.4.2 was pending. Publication completed 2026-10-13 at b572269 (v0.4.2); 1.0/human validation remain unasserted.

### Operational hardening — single PR, owner review required

The owner supplied the operational correction plan on 2026-10-15. The work starts from b572269085494ddd4d6370716ff1cb360b46065f; v0.4.2 was published on 2026-10-13 (the earlier pending-publication entry is historical). At intake F01–F08 were in progress; the completion/qualification record below supersedes those dated intermediate states. The original plan and baseline evidence are archived in docs/audit/operational-hardening-2026-10-15. All changes belong to one branch and one Draft PR; merge and release publication are not authorized. The HTTP 422 CLI reproduction confirms preservation of the frozen return and refusal to sign again without authoritative resolution. Full new-source acceptance remains pending.

F01 foundation: ADR-034 (renumbered to preserve the existing package-identity ADR-030), separate terminal-resolution domain/port, immutable Memory and SQLite/PostgreSQL storage and forward-only migration 0027 are implemented. The focused repository contract passed Memory/SQLite/real PostgreSQL: rollback leaves no decision, committed receipt survives, foreign Principal lookup is absent and overwrite is rejected. Application arbitration, transport/CLI and the full race/restore matrix are still pending.

F01 arbitration: current authenticated CID, signer, policy, original DSSE/digests and authority are revalidated; Namespace/Outcome serialization makes acceptance and terminal closure exclusive. Work/review acceptance paths consult durable closure state. Focused race runs passed Memory/SQLite/real PostgreSQL, including identical/distinct resolutions and live authority rejection. Existing atomic returns and the extended independent review/correction journey passed with actual SQLite clean-file and PostgreSQL clean-database restores. Transport/CLI exposure and crash/ENOSPC coverage remain in progress; this is not full F01 acceptance.

F01 transport: generated HTTP/MCP/SDK command and exact signed schemas now expose terminal resolution, with additive capability discovery and typed proof lookup. A real authenticated journey passed Memory/SQLite/real PostgreSQL: HTTP 422, SDK resolution, identical MCP replay, HTTP cryptographic proof and deterministic 409/signed_intent_closed on late HTTP and MCP returns. New migration 0027 also checks the execution/review contract reference. CLI integration and remaining F02–F08 work are pending.

F01 CLI: work/review resolve and resolution-status use an additive authenticated resolution journal, immutable original payload and pinned signed proof. Confirmed negative decisions archive exact draft/signature bytes before releasing the pending marker; current human edits remain editable. Ambiguous responses retry only the same resolution; output distinguishes closure from acceptance. Full CLI race suite passed after the change; focused resolution/prevalidation checks passed after the read-only status refinement. Stable assessment/finding preflight and bundled skill references are aligned. Real new-resolution process death/ENOSPC, terminal restore and native cross-platform final-source qualification remain pending.

F01 failure qualification: real Linux process death at each archive/unfreeze boundary recovered using the persisted proof. A disconnected 1 MiB Docker tmpfs produced actual ENOSPC at proof persistence, draft archive, receipt archive and unfreeze/profile cleanup; original bytes remained recoverable. SQLite clean-file and PostgreSQL clean-database restore retained the exact terminal proof and rejected late returns. Actual persisted 30-second lease expiry, without changing rows/signatures/database time, rejected both closure and return without revision/authority writes on Memory/SQLite/real PostgreSQL under race. The native unified binary completed the real rejected review → explicit resolve → retained draft → corrected independent review/correction journey. The active resolution fuzz campaign passed 72,082 inputs. Source-file-bound intermediate logs are archived; final-source qualification remains pending. A bounded Windows VHD runner and native binary smoke are added to CI but are not presented as locally executed Windows evidence.

Initial PR source checks found two integration issues: the new signed command needed explicit profile-only browser exposure, now recorded; the immutable 0.1 fixture cache TTL had elapsed on 2026-10-15. The migration test now relocates only cache expiry in its disposable verified copy, preserving payload/hash/receipts/timestamps and production retention. Focused migration/exposure regressions pass. Public schema fingerprint review remains deliberately pending until final additive contracts settle.

The first native Windows VHD job created, verified and detached the owned 32 MiB volume successfully, but PowerShell split the unquoted Go test flags before execution. Flags now use a literal argument array; native Windows full-disk acceptance remains pending its rerun. This was a runner invocation issue, not a passing Windows recovery result.

Native Windows then executed the capacity test: proof-persistence failed safely on disk full, while three smaller cleanup writes still fit the unallocated remainder after a rejected 64 KiB write. The fixture now exhausts remaining 4 KiB/512-byte/single-byte allocation units as well, to exercise actual NTFS disk full at every publication window. No production behavior or assertion is relaxed; rerun remains required.

F02 HTTP: explicit connection/header/read/write/idle/request bounds, nonblocking request admission with two health slots, body framing limits and aggregate metrics are implemented. Stateful MCP standalone GET streams have dedicated socket/context lifetimes; real MCP JSON/SSE testing caught and corrected an ordinary read deadline cutting a stream early. Shutdown force-closes active HTTP connections after grace. Real TCP tests cover slow headers/body, oversized input, idle sockets, excess connections, cancellation/panic slot release, health capacity and shutdown. A real committed Outcome with a lost deadline response is recovered by original receipt and exact retry, with no implicit SDK replay. Full server/HTTP/observability race suites passed; focused race passed again after final refinements, and vet/Compose example validation passed. An unchanged v0.4.2 binary kept the incomplete body open without response after 17 seconds; the candidate returned the existing deadline error and closed within that window. Intermediate source-bound evidence is archived; final source and cross-platform gates remain pending.

F03 delivery diagnostics are implemented as a scoped SQL read port, authenticated administrative HTTP/MCP/SDK query and a separate process snapshot. The worker now reports fatal exit/panic and distinguishes intentional cancellation; API readiness remains independent of receiver delivery. The new authorization test exposed zero-Outcome metadata permission being insufficient for a Namespace-wide queue collection; the query now resolves the current parent credential policy inside the same snapshot and rejects Outcome-restricted credentials before aggregation. Focused SQL tests pass on SQLite and real PostgreSQL; the complete new runtime/parity test set is still being qualified. Bounded parallel delivery and its scale measurements remain pending. CI on 0107c654 passed Core/race/fuzz, browser and native Linux/Windows (including actual NTFS full-disk recovery); packages still require the planned final public-contract fingerprint review.

F03 diagnostics gate: the complete focused race run now passes on SQLite and real PostgreSQL, including old backlog with worker disabled while readiness remains 200, fatal worker exit while readiness remains 200, panic/cancellation classification, revoked and Outcome-restricted credentials, exact history across manual redelivery, exhaustion, and authenticated HTTP/SDK/MCP parity. The source-file-bound log is archived in intermediate-f03. This completes the diagnostic implementation slice; concurrency, load/replica measurements and exact final-source gates remain pending.

F03 bounded concurrency slice: a fixed delivery pool with default 1 and validated cap 64 is implemented; SQLite emits a parallel-write advisory. The 100-message race matrix exercises one lane, four lanes and two independent handles/replicas over four Outcomes, verifies HMAC, bounded inflight requests, backlog drainage and consumer deduplication after deliberate ACK loss. It exposed an unclassified PostgreSQL attempt-update serialization failure, now mapped, and unnecessary SSI conflicts in row-local queue operations. PostgreSQL delivery claims/ACKs now use Read Committed with SKIP LOCKED/exact fenced CAS; domain/security transactions remain Serializable (ADR 031). The 100-message pool matrix passes; actual process-kill and 1,000-message qualification are still running/pending. No throughput/SLA claim is made from this intermediate run.

F03 concurrency gate passes: actual process kill/reclaim on both SQL backends fences the old worker (two receiver requests, one deduplicated effect and one authoritative success). The 100-message race matrix passes on SQLite/PostgreSQL. The PostgreSQL 1,000-message matrix over four Outcomes measured 24.77 messages/s with one lane, 94.23 with four lanes, and 95.90 with two replicas of two lanes; these are synthetic loopback measurements with a 2 ms receiver and deliberate ACK loss, not production capacity/SLA. Inflight never exceeded the configured total; every message reached the deduplicating receiver. SQL plans show SQLite covering search versus scan; unvacuumed PostgreSQL preferred a scan, while ordinary VACUUM ANALYZE enabled index-only search (15 versus 345 buffer hits in the maintained fixture). The index comparison rolls back its temporary DROP in the owned fixture. Logs and production-file hashes are archived; final PR-source and cross-platform gates remain pending.

F04 read-port contract slice: ContinuityReadRepository and a bounded metadata-page builder exist. Memory's ten direct sections match the current legacy service byte-for-byte across consecutive pages, including Unicode truncation, HTML/quote/backslash external-reference ordering, relation/link payloads and exact total/omission counts; focused race passes. Application endpoints still use the old path until SQL/projection integration, so no performance correction is yet claimed. The unchanged v0.4.2 archive now has a fast domain-validated preload benchmark (not fabricated proof) and exploratory raw SQLite/PostgreSQL measurements at 100/1,000/10,000 rich items, three repetitions. Shared-host overlap/outliers are explicitly recorded; candidate comparison and final-source gates remain pending. ADR 032 records the cursor/projection strategy in progress.

F04 SQL metadata slice: SQLite and generated PostgreSQL now page all ten direct sections with scoped COUNT and LIMIT, without aggregate hydration; shared row scanners avoid per-item relation/link fetches. Migration 0029 adds exact Go JSON sort/digest keys for external references and backfills in bounded batches inside the migration transaction; schema 26 migration and digest continuation pass on both databases. The immutable v0.4.2 producer generated a complete fixed rich snapshot; Memory/SQLite/PostgreSQL match all fields (only read evaluation timestamps normalized) and consecutive direct pages under race. Exact provenance is archived. The new focal assessment join index still requires dense SQL-plan/measurement qualification. Application routes, derived/full continuity, focal and graph optimization remain in progress; no page-allocation target is yet accepted.

F04 direct Application slice passes focused race/parity on Memory/SQLite/real PostgreSQL. Direct section queries use metadata Outcome presence, current authorization, coherent coordination/time and scoped LIMIT pages. A test adapter panics if this path hydrates Outcome proof or any WorkItem; old digest cursors, new v2 cursors, long external references, future evaluation time, missing Outcome and revision changes all pass. Direct-page measurements at 100/1,000/10,000 keep SQLite allocation around 85 KB/op and PostgreSQL around 93 KB/op (old 10,000 section about 126/130 MB/op). Raw timing/outliers are retained; final-source qualification remains required and full/derived continuity, graph and current-proof focal reads are not yet migrated. PostgreSQL focal initial outliers require controlled final measurement, not a threshold waiver.

F04 full/derived continuity implementation is now under qualification. A shared pure classifier preserves the full OutcomeState API while optimized reads use scoped operational headers, required-proof SQL counts and only the current conclusion's referenced assessments/evidence. Selected blocker bodies are fetched in bounded batches with a byte budget; full snapshots reduce pages deterministically to retain the 256 KiB response bound. Golden/parity, inherited blocker body/order, old derived digests, real state/current-proof disagreement, retracted evidence, lease expiry without a write and frozen cursor time pass focused race tests on Memory/SQLite/real PostgreSQL. The legacy derived hash test found and fixed []byte being hashed as base64 rather than raw JSON. Full snapshot 10,000-rich-item laboratory target is set before candidate measurement: at least 30% lower p95 and 50% lower allocation than the archived same-fixture baseline. Graph and focal current-proof migration remain pending; no final-source acceptance is asserted.

F04 projected-snapshot regression slice passes sequential focused race execution on Application/Memory/SQLite/real PostgreSQL. The first PostgreSQL 10,000-item run hit statement timeout in the progress query: EXPLAIN showed a correlated evidence subplan and estimated cost 166,662 with JIT. Progress now materializes scoped required/current/invalid-source sets once (same failed fixture: estimated cost 3,702, actual 91 ms, no JIT; no global database setting changed). Proof/retraction/cursor/size semantics pass after the query correction. One shared-host SQLite migration exceeded its existing 30-second open deadline; the isolated test and complete focused sequential rerun passed unchanged, with no timeout relaxed. Raw failures/plans are retained. The corrected snapshot capacity rerun is pending; graph and focal work still remain.

F04 focal/graph semantic slice passes focused race tests on Memory/SQLite/real PostgreSQL. Current-only owner proof, exact history omission counts and bounded relevant records match the legacy fallback even with 1,000 assessments and a hard dependency outside the 100-item displayed page. Graph adjacency batches, directions, depth-boundary induced edges, cycles, kind filters, old-compatible cursors and revision invalidation match legacy results without full aggregate hydration. An actual SQLite 10,000-item rerun exposed a materialized-current-proof regression (p95 27.37 seconds); the failing measurement is retained, not accepted. SQLite now uses indexed point joins while PostgreSQL retains the scoped materialized-set query via an explicit generator template. Focused semantics pass after correction; embedded-adapter capacity reruns, dense query-plan evidence and final-source gates remain pending.

F04 follow-up: SQLite full snapshots at 10,000 now measure p95 214–231 ms and about 44.5 MB/op across three exploratory repetitions. A fresh PostgreSQL fixture still timed out before statistics warmed; a warm 83 ms EXPLAIN is diagnostic, not acceptance. PostgreSQL now counts required proof separately and uses hashed membership for met/waived/retracted-source sets, removing the inter-CTE join susceptible to badly underestimated cardinality. Fresh-fixture and final controlled measurements remain pending.

F05 is implemented with focused acceptance: Namespace-only optional Principal limits 1..100 retain old JSON defaults 3/1, are changed through authenticated Namespace/policy CAS and are enforced independently of CID/Namespace ceilings in signed work/review acquisition. Read-only identity/CLI diagnostics show effective ceilings and own activity. Memory/SQLite/real PostgreSQL passed bounds/scope/unauthorized/stale-CAS, two CIDs/two Outcomes, distinct Principals, revocation and lowering-active tests; the 50-consumer test granted exactly four remaining slots under a limit of seven (108 PostgreSQL conflicts retried with unchanged intentions in one run). Real PostgreSQL empty-database restores preserved execution/review/correction authority. A first local invocation omitted the container restore flag and failed in the test reopen path; the required real-restore invocation passes, with no production authorization relaxed. Final exact-source/public-contract gates remain pending.

F02 reconnect follow-up: a hosted MCP check returned an empty 200 after a bounded standalone stream; local race repetition reproduced it once in 30. Bounded SSE now enforces Connection: close at the actual write/flush boundary (the SDK otherwise overwrites keep-alive headers), with a 25 ms write-deadline propagation margin beyond the context lifetime. Thirty repeated real stateful stream/JSON reconnect journeys pass under race and assert connection closure; ordinary request limits remain unchanged. Final full-source transport checks remain required.

F04 fresh-plan diagnosis confirmed repeated scoped index scans with ID join filters under one-row cardinality estimates. PostgreSQL now uses lateral primary-key point lookups, retaining Namespace/Outcome equality on joins. On a rolled-back unanalysed copy of actual validated 10,000-item data, the aggregate executes in 439 ms versus 15.64 seconds for the intermediate lateral wide-index variant; earlier full benchmark timeouts remain recorded. No planner settings/deadlines were relaxed. The current read-model race slice passes Application/Memory/SQLite/real PostgreSQL, including complete continuity/golden/cursor/focal/graph tests. Full fixed-sample benchmark and final exact-source acceptance remain pending.

F06 implementation/profile slice passes: the original local SQLite Compose/env contract is retained. The opt-in shared PostgreSQL/TLS reference uses unique external instance credentials and issuer files, persistent volumes, health checks, resource limits and optional bounded webhook workers. Real local/HTTPS requests verified UI serving, absent/invalid credential rejection, Namespace isolation, secure browser cookies, cross-origin rejection, data persistence and unchanged issuer identity across restart. The initial shared bind-mounted entrypoint lacked non-root readability; it is now included in the image with explicit permissions. Loopback probes bypass proxies and internal service proxy bypass is explicit. Read-only deployment diagnostics and the per-API PostgreSQL pool setting pass repeated and complete server/CLI race tests; declarations do not claim external verification. `.go-version` now pins Go 1.27.2 for CI/release and the Docker drift contract passes. The signed contention harness supports 1/4 Outcomes and 2/8/50 consumers, exact one-winner version collisions, full outbox pagination and CPU/RSS/pool/conflict reports; its two-consumer smoke passes both SQL adapters. The immutable-source full matrix and final literal Compose build/source acceptance remain pending.

F06 fixture qualification correction: the initial 4-Outcome/2-consumer assignment only wrote to two Outcomes. The preliminary matrix was stopped and is not acceptance evidence. Tasks now cover every declared Outcome, active_outcomes is asserted/reported, and the report consumer rejects a dimension mismatch. The corrected four-active-Outcome/two-consumer scenario passes SQLite and real PostgreSQL, including the one-winner collision and exact retries. Full immutable-source matrix rerun remains pending.

F06 immutable contention reference completes all 12 SQLite/PostgreSQL combinations for 1/4 active Outcomes and 2/8/50 consumers on e6dae02, including real independent review, current original criterion assessment, exact approval replay, full outbox identity counts, one-winner version collisions and unchanged-intention conflict retries. Reports retain pool wait and whole-process CPU/peak RSS with fixture/setup boundaries. At 50 consumers this laboratory's SQLite signed-journey p95 is about 4.65–6.16 seconds; PostgreSQL about 103.67–143.08 seconds with roughly 5,000 conflicts/retries. These are measured contention limits, not a claim that PostgreSQL is always faster, a production SLA or a supported user count. PostgreSQL remains the shared/replica backend; production sizing must account for Namespace/security guard contention. Two-API-process recovery and final-source qualification remain pending.

F04 graph filter bound: a compact valid query can contain tens of thousands of empty/duplicate kind filters and exceed SQL bind limits despite a bounded response. SQL adapters now intersect/deduplicate only the existing supported edge kinds, preserving legacy empty/unknown/inclusion meaning and cursor fingerprints while limiting filter bind count. Legacy-oracle parity with 40,000 empty kinds, mixed known kinds and normal graph fixtures passes Memory/SQLite/real PostgreSQL under race.

F03/F06 actual API-process recovery follow-up: a two-process/four-lane pilot exposed the queue-wide expired-attempt sweep blocking recovery; the surviving worker visibly stopped after 96/100 acknowledgements. Recovery now locks each selected parent before its current attempt, matching ACK order, and bounds crashed-budget cleanup to 25 per transaction. The independent locked-expired-row regression fails before and passes three race repetitions after the fix; SQLite/PostgreSQL health, crash-budget and worker race contracts pass. Intermediate real API pilots now recover 100/100 after SIGKILL (four external repeats deduplicated), and restore eight delivered-before-snapshot signals into a new PostgreSQL database with unchanged identity and consumer dedupe. Exact-source qualification and remaining F07/F08 work are still pending.

F07 operator-tool slice passes six procedure tests and actual Memory/SQLite/PostgreSQL signed authority journeys using the opt-in backup/restore tools. Execution/review/correction stages retain exact authority bytes, versions and fences; terminal closure, current revocation, issuer history/recovery receipts and durable signals survive new-destination restore. The first second-stage PostgreSQL helper reused the initial source DSN; it now derives the actual current source database and the complete suite passes without authorization changes. The tools preserve private manifests/checksums and public identity pins, exclude signer seeds, refuse live/existing destinations and provide a complete external wrapper/schedule/workspace/issuer/monitoring runbook. A two-real-API new-database pilot demonstrates delivered-before-snapshot repeats with durable consumer dedupe. Actual v0.4.2 source upgrade, final-source pilots/metrics and remaining F04/F08 gates are pending; no production RPO/RTO or 1.0 certification is asserted.

F04 controlled measurement v2 is prepared before running: the identical domain-validated rich fixture/harness is copied to immutable baseline/candidate archives; both perform ANALYZE outside timing. Cold unanalysed failures remain separate evidence and are not waived. SQL statement tracing and pool-wait deltas are now real on both adapters, with whole-precompiled-child CPU/peak RSS labelled inclusive of setup and exclusive of database CPU. The predeclared comparison uses the median of three equal-sample p95 repetitions (80 samples at 100/1,000; 40 at 10,000), retains every repetition and enforces simple/full 10,000 targets of p95 ratio <=0.70 and allocation ratio <=0.50, plus focal 1,000 p95 ratio <=1.10. No target acceptance is claimed before the run.

F04 controlled v2 reference completes every 100/1,000/10,000 rich-read case on SQLite and real PostgreSQL (immutable aa7ccc3 vs b572269; three equal-sample repetitions, identical harness/statistics preparation, isolated fixture DBs). Every predeclared target passes: simple/full 10,000 median-p95 ratio <=0.70 and allocation ratio <=0.50; focal 1,000 ratio <=1.10. Raw p50/p95/p99, actual SQL statement counts/pool waits, bytes/allocations/response sizes and whole-precompiled-child CPU/peak RSS are archived. Full snapshots retain unavoidable scoped classification/proof work and about 44–45 MB transient allocation at 10,000; no constant-cost/full-scale claim. Cold failure evidence is retained independently. Dense assessment-index plan justification and final exact-source gates remain pending.

F08 public-shape review retains exact released v0.4.2 schema/fingerprint references while freezing the additive candidate. Strict fingerprint and structural compatibility tests pass: all old command routes, accepted properties, required inputs, enum values and bounds remain; the identity read now documents its actual response plus quotas. Review caught the new resolution DSSE purposes missing from generated payload enums; codec/schema validity regression and regenerated OpenAPI/YAML schemas now agree. The minor release queue declaration does not advance VERSION, merge the PR or publish a release. New-resolution pending files require the updated CLI; old strict clients must fail safely. Actual v0.4.2 signed-state upgrade and final-source gates remain pending.

F04 migration-index qualification passes on SQLite and real PostgreSQL, including race: an actual 4,096-assessment/32-owner domain fixture compares the exact focal omission-count query with/without only the new index (DDL rolled back). SQLite uses a covering index and PostgreSQL an index-only scoped scan for the selected 128 assessments. PostgreSQL diagnostic execution improves from 1.460 to 0.360 ms in the first run and 1.679 to 0.145 ms under race; no planner setting/deadline was relaxed, and maintenance is scoped to fixture tables. Raw plans are archived. This closes the dense-history index justification; final source gates/remaining F07 upgrade qualification are pending.

F07/F08 actual released-source upgrade passes SQLite and PostgreSQL: a Git-blob-verified b572269 producer creates real schema-26 execution/review/correction history; the new operator tools restore to new destinations and candidate 9709da8 migrates to schema 29. All original columns/rows in 67 tables and old migration checksums/rows are retained. The current codec verifies 39 original DSSE facts (12 specs, 13 authority, seven returns/seven receipts) without re-signing. Actual old binary migration is refused, missing configured private seed fails explicitly, and original state remains unchanged. The first driver omitted required bootstrap configuration and failed before migration; corrected fixture declarations pass with no production validation change. Reports archive hashes and bounded fixture RTO only. CI now includes this source-pinned journey, and PR workflows check out the actual head SHA while release workflow source overrides remain intact. Complete final-source/backend/browser/packages/native/Compose acceptance remains pending.

F04 explicit cursor-after-restore qualification passes under race: SQLite new-file and real PostgreSQL new-database restore retain byte-identical continuation pages for legacy digests and v2 keysets. Consecutive no-hydration pages, frozen revision/evaluation time, long external digest, wrong scope/missing Outcome and later revision invalidation remain covered. Derived cursors also survive restore while leases are active, then keep their original frozen classification as fresh reads detect expiry without a write. The terminal-resolution ADR is uniquely numbered 034, retaining the original existing package-identity ADR 030 unchanged. Final-source workflow/Compose acceptance still follows.


## Operational hardening — implementation and qualification closure (2026-10-16)

All F01–F08 implementation work in the owner's single-PR plan is complete within its stated product boundary. [Current acceptance and support limits](docs/stable/operational-acceptance.md) and [permanent qualification receipts](docs/audit/operational-hardening-2026-10-15/qualification/) supersede the dated intermediate pending notes above. There is one branch/PR #106, kept Draft/open: no merge, tag, version preparation or release publication was performed.

| Item | Executed qualification |
| --- | --- |
| F01 terminal return resolution | Memory/SQLite/real PostgreSQL transactional races, accepted-versus-closed replay, current scope/revocation/issuer/lease/fence, HTTP/MCP/SDK, clean restore, Linux/Windows native recovery and all real-full-disk cleanup windows |
| F02 HTTP/MCP bounds | Hostile TCP/admission/reserved health, bounded ordinary/stream lifetimes, repeated real reconnect, lost committed response/receipt reconciliation, cancellation/panic and shutdown |
| F03 delivery operations | Authorized durable health, process diagnostics, HMAC/SSRF/retry/redelivery, bounded 100/1,000 load, locked-expired-row regression, two actual APIs/four lanes, SIGKILL/new-DB restore and durable receiver dedupe |
| F04 reads | Immutable v0.4.2 golden/parity, bounded direct/current-proof/adjacency reads, old/new cursors across real restore, frozen evaluation/expiry without writes, 256 KiB/page limits, scoped dense index plans and passing controlled 100/1,000/10,000 comparison targets |
| F05 quotas | Defaults/1..100 Namespace-only policy, independent Principal/CID/Namespace ceilings, 50 concurrent consumers/two CIDs/two Outcomes, CAS/current revocation/lowering and execution/review/correction authority after restore |
| F06 profiles | Exact env-example clone local Compose build (free host port override), persistent data, non-root image/source label; shared PostgreSQL/TLS startup/identity/auth/cookie/origin/restart and read-only diagnostics; full 12-case 1/4 active Outcome x 2/8/50 signed matrix |
| F07 continuity | Consistent private backups, independently approved manifest/issuer pins, new-destination-only restore, complete read-only logical verification, real signed/issuer/revocation/outbox journeys, workspace/issuer/external schedule-monitoring procedures and measured fixture-only RTO/RPO |
| F08 compatibility/source | Original migrations 0001–0026/license retained; strict candidate and immutable 0.4.2 shapes; actual schema-26 producer -> schema 29 with 67 original tables/39 verified old DSSE facts; old source and downloaded released native binary refuse future schema; configured missing seed fails explicitly |

Complete green source-6e91208 workflows: core 38840856862 (unit/race/real SQL/clean restores/fuzz/transport/container/capacity/actual old upgrades/operator pilots), browser 38840856701, packages/reproducibility 38840856684 and native Linux/Windows 38840856635. Native receipts explicitly confirm protected keyring execution; Windows full-disk recovery used a 32 MiB isolated VHD. Code subject 2a46475 adds the separately passing old/new cursor restore regressions; all 542 production/validation inputs match that green source byte-for-byte. The 295 read Core/SQL/module/toolchain/harness files match measured aa7ccc3 exactly. The archive records hashes and never mislabels an older benchmark as a newly executed final-head run.

Exact final PR-head CI must be green before delivery; [head checks](https://github.com/A1b3rt0M3rcad0/wos/pull/106/checks) and their emitted artifacts identify that SHA independently of these historical/code-equivalence receipts. The final documentation/evidence commit is requalified by those same required workflows. This is a source acceptance gate, not a waiver based on earlier runs.

Controlled read targets pass on both SQL adapters: at 10,000 simple/full median-p95 ratio <=0.70 and allocation <=0.50; focal 1,000 ratio <=1.10. Full derived snapshots still allocate roughly 44–45 MB for global scoped classification/proof. The complete 50-consumer signed matrix measures SQLite p95 4.06–4.18 seconds and PostgreSQL 91.42–140.45 seconds with about 5,000 conflicts/exact retries. Those are explicit laboratory contention limits, not a production SLA, invariably faster PostgreSQL, a supported user/session count or thousands-of-writers certification. Backups/retention/alarms are opt-in external operations; restore cannot reverse receiver effects or security changes after a stale snapshot.

The remaining human comprehension/usability and manual screen-reader/zoom validation is pending under the owner's prior explicit instruction. Product 1.0 is not certified/prepared/published. The public package remains wosctl, public operations remain wos <command>, skills remain embedded, and Apache-2.0 is unchanged. Merge remains prohibited for this delivery.
