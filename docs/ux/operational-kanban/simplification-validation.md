# Focused modal Kanban qualification — 2026-10-20

This receipt covers the seven owner-requested simplifications in draft PR #116.
It supersedes the presentation acceptance in [the previous receipt](validation.md),
without rewriting its historical results. See [audit and decisions](simplification.md)
and [ADR-041](../../adr/0041-focused-kanban-modal-and-backlog-planning.md).

## Source and environment

Runtime and browser tests: `fa82060cf661415d2253f32ac11dedfbf54296e1`.
Subsequent documentation-only commits do not change `packages/`, `tests/` or `tools/`.
The local API executable is stamped with that commit; screenshot receipts bind its
metadata, served assets and unmodified PNGs. Linux Chromium 154.0.8037.57, two
browser workers, isolated synthetic installations and unchanged test deadlines.
No retries, skips or forced clicks were introduced to obtain acceptance.

## Executed checks

| Check | Result / boundary |
|---|---|
| Complete browser suite | 229 passed; zero failed, skipped or flaky; one attempt per case ([structured receipt](simplification-browser-results.json)) |
| JavaScript units | 78 passed |
| `go test ./...` | Passed with the default local environment |
| `go vet ./...` | Passed |
| `go test -race ./packages/wos-core/domain ./packages/wos-core/application` | Passed |
| Generated mirrors/contracts | PostgreSQL, signing, MCP and OpenAPI generators leave no source diff; event catalog and plan archive checks passed |
| Outcome navigation race | Original navigation scenario and delayed-permission regression: five repetitions each, ten passed |
| Current screenshots | Six modal/proof capture cases and one Board capture case passed; these overlap the full browser suite, not seven additional distinct acceptance cases |

Local PostgreSQL DSN was unset: PostgreSQL-dependent tests skip locally and require
the separate real-database CI gate. A passing default Go suite does not establish
new real-PostgreSQL acceptance. Hosted checks qualify their own exact pushed head.
No release, merge, tag or version advancement is part of this change.

## What the scenarios establish

- Modal Objectives/Tasks collections preserve the mounted Board, camera, selected
  context and return focus. One native dialog is open at a time, including forms,
  conflicts and uncertain-operation recovery.
- Direct Overview remains usable after switching Outcome, including deliberately
  delayed permission responses. Context controls the collection and its one create
  action; captions do not repeat breadcrumb navigation.
- Selected Kanban cards omit the duplicate Verify criteria/Add material row,
  technical audit and action directory. Mobile lane controls cannot intercept the
  inspector's pointer input; assertions use real hit-testing.
- Review proof loads instructions and existing criteria. It can reuse Evidence or
  register new criterion-scoped material, assess explicitly and complete real work.
  Evidence does not check a criterion or accept its parent. Long instructions can
  load the canonical definition at the same Outcome revision without an audit panel.
- Task planning commands and capability reads reject definition changes outside
  Backlog without changing versions/events. To do and In progress agree. Explicit
  Defer permits replanning; operational proof, assignment and authorized execution
  retain their independent guards. Initial atomic definitions remain supported.
- Existing signed execution, readiness, permissions, independent review, strict
  completion, stale versions and exact-intent recovery regressions remain enabled.

## Visual evidence and limits

[Current gallery and source-bound manifest](screenshots/focused-modal/README.md):
Board 1366×768; Task Kanban 1280×720, 2560×1440, 3840×2160 and 390×844;
Objective Kanban 2560×1440; proof checklist 1280×900. The actual browser screenshots
retain the existing paper palette, typography, robots and physical post-it cards.
Separate native browser-zoom scenarios cover 125%, 150%, 200% and 400%.

These are bounded desktop Chromium/viewport, keyboard and automated accessibility
checks, not physical-device testing, a participant study or manual screen-reader
certification. No general WCAG compliance or productivity improvement is claimed.

## Diagnostic history

Intermediate failures were retained and corrected rather than hidden:

- A complete 227-case development run had 217 passes and ten failures, including
  fixtures that edited active Tasks, obsolete menu expectations and capability
  projection differences. Fixtures now prepare definitions before activation.
- A subsequent 228-case run had 227 passes and one failure: the previous Outcome's
  header was revealed before current permissions, losing a single Overview click.
  The header now appears only when permissions and rendered navigation agree.
- The mobile capture exposed lane menu controls painting above the inspector;
  the stacking correction has a pointer-target regression.

The final full-suite result above, rather than these diagnostic runs or earlier
PR CI, is the local browser acceptance gate for this runtime source.
