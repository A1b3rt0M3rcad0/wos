# Baseline for the Issue and dependency plan

This baseline applies to the owner-authorized implementation in PR #116. The
starting source is `d80d8329e097b7252b65b9c75f49e00d6cdcfa39`; it remains Draft
and open. This step records existing behavior and onboarding tests before changing
Task ownership, Issue ownership, migrations or entry flows.

## Existing behavior confirmed in source

- `WorkItem.ObjectiveID` is optional. The Outcome can still contain direct Tasks;
  the current progress/projection model treats that as supported existing behavior.
- Issue stores severity, lifecycle, affected references and the immutable reporting
  actor. It does not yet store an Objective owner or a responsible assignee.
- Issue queries are Outcome-wide. Objective achievement currently checks existing
  criteria, Blockers and dependency gates, not pending owned Issues.
- Blockers represent an explicit impact on one target (or its structural subtree).
  Resolving an Issue does not clear its Blockers.
- `depends_on` is a directed, same-Outcome, cycle-checked relation. Its source is
  the dependent item; hard dependencies influence readiness, advisory links do not.
- Task planning is already frozen outside Backlog by ADR-041 and backend guards.
  This plan must preserve that policy for dependency edits and reparenting.
- The editorial embedded workspace and first-use flows are in the WOS API web
  assets. No separate onboarding application has been assumed.

## Onboarding and entry regression inventory

| Concern | Existing executable coverage |
|---|---|
| Login, Outcome creation, Task creation and authorization | `tests/web/accessibility.spec.mjs` (axe/login and keyboard flow), `tests/web/journey.spec.mjs` |
| First Task created in Objective context | `tests/web/progressive-entry.spec.mjs` (searchable reference and retained owner) |
| Long real operation and cross-session resumption | `tests/web/journey.spec.mjs` |
| Mobile quick creation and searched Objective placement | `tests/web/accessibility.spec.mjs` (390px) |
| Issue and explicit Blocker resolution | `tests/web/accessibility.spec.mjs`, `tests/web/journey.spec.mjs` |

The named suites ran against baseline `d80d8329e097b7252b65b9c75f49e00d6cdcfa39`
with Chromium 154: **12 passed** in 41.5s; zero failed, skipped or retried. This
includes login, first-use creation, searched Objective placement, desktop/mobile
reflow at 200% and 400%, keyboard restoration, and Issue/Blocker operations. No
onboarding code changed in this stage. Existing source-bound captures are listed in
[screenshot receipts](screenshots/README.md) and the [focused modal gallery](screenshots/focused-modal/README.md); they do not prove the future hierarchy contract.

## Gates

No Objective is created automatically to hide missing ownership. No legacy row is
assigned to the first matching Objective. No Issue is owned by inference from
`affected_refs`. Contract migrations remain gated on a deterministic, read-only
orphan report and an explicit mapping for every row in the target database.
