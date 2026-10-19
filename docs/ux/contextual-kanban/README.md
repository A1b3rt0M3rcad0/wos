# Contextual Kanbans and the operational inspector

This follows the [owner's preserved plan](owner-plan-2026-10-19.md) and
[ADR-039](../../adr/0039-contextual-kanbans-and-strict-completion.md).
The local [product-design-and-ux skill](../../../.agents/skills/product-design-and-ux/SKILL.md)
guides the interaction contracts; existing editorial styling and illustration assets remain.

## Experience contract

| Entry / user intent | Observable behavior | Persistence and recovery |
| --- | --- | --- |
| Select an Outcome | Compact inspector; Open Objectives Kanban is salient | No mutation or camera change from opening the modal |
| Select an Objective | Open Tasks Kanban; real parent remains visible | Direct Tasks and subobjectives stay separate |
| Enter an Objective from its Kanban | Same modal, breadcrumb and next level | Returning keeps the preceding filters and column scroll |
| Select a Task | Execution/review actions through actual capabilities; no Task Kanban | Existing command forms, host handoff and exact-intent recovery |
| Expand Details and management | Criteria, relationships, people and technical facts | No change to lifecycle |
| Expand Activity history | Fetch recent events on demand, with actual author and subject | Frozen-frontier pagination; history does not execute work |
| Verify criteria | Current definition and applicable assessment, then explicit assessment action | An assessment is not a Conclusion; waiver is not verification |
| Add material | Choose produced Artifact or verification Evidence | Existing atomic commands; external source is participant supplied |
| Open a partial child page | Loaded counts and an explicit partial notice | Cursor pins the Outcome revision; refresh after conflicts |
| Start a contextual change | Controlled return to the existing inspector/form | Return to Kanban preserves its navigation stack |
| Close modal / Escape | Board and originating focus return | No implicit mutation, lifecycle or relationship change |

The modal uses native focus isolation; the background is inert. Narrow screens
retain readable, horizontally scrollable columns and a contained item inspector.
All operational transitions use explicit existing actions; cross-column drag is
not enabled. Selection, search and presentation ordering grant no authority.

## Plan and operate

1. On the Board, select an Outcome and choose **Open Objectives Kanban**.
   Use **Direct Tasks** for work that belongs directly to the Outcome.
2. Select an Objective to inspect it, then choose **Open Tasks Kanban**.
   Its **Subobjectives** tab preserves the real hierarchy. The breadcrumb returns
   to the preceding context instead of opening another modal.
3. Select a Task to see its current state, explanation and permitted next action.
   **Details and management** holds the complete definition and relationships;
   **Activity history** loads actual events when expanded.
4. **Verify criteria** shows current assessments before offering an authorized
   assessment form. **Add material** distinguishes a produced Artifact from
   supporting or contradicting Evidence. Neither action accepts the work.
5. A contextual mutation temporarily returns to the existing inspector/form.
   **Return to Kanban** restores the visual navigation stack. **More actions**
   uses the same canonical action menu, not a second command implementation.

Search applies to **loaded cards**. Counts identify the loaded subset, and **Load
more** uses a revision-bound cursor. Selected details, criteria and lazy history must match the visible revision.
Refresh reloads the selected detail together with its columns. Changing the Outcome while paging requires
**Refresh**; stale pages are never merged. Required-Objective, priority, declared
participant and cancellation filters remain visual choices. Cancelled records
appear in a separate column only when requested.

| Displayed stage | Source of truth |
| --- | --- |
| To do | Current operational lifecycle, including backlog |
| In progress | Current operational lifecycle |
| Verification | An actual applicable pending review fact |
| Accepted | A terminal accepted lifecycle; stale review facts do not override it |
| Cancelled | A cancellation, never counted as acceptance |

An Objective whose Tasks are finished can still have an unassessed Criterion.
The parent requires its own review and explicit Conclusion. The UI does not
invent a parent review queue or silently turn completion into acceptance.

## Scope, permission and recovery

Read-only users can inspect without receiving mutation authority. The server
rechecks all capabilities and optimistic versions at dispatch. A scope/identity
change clears modal frames; old responses cannot restore their content. Revoked
credentials follow the existing private-state reset path. Closing a modal is a
presentation operation and cannot submit a form or change lifecycle.

The existing command form retains its typed conflict handling, signed-host handoff
and exact-intent recovery for uncertain results. A saved mutation followed by a
failed refresh is not retried as a new write. Column scroll, loaded search,
filters and origin focus are session presentation state, not persisted domain
objects. Modal launchers belong to Board inspection; specialist views keep their
existing single native dialog and contextual controls.

## Strict completion

New browser-created Outcomes visibly default to strict work. API callers that
omit `completion_policy` retain compatible behavior. See ADR-039 for exactly which
children are obligations. A cancellation must have its own reason and a pinned,
explicit exclusion acknowledgement on the parent Conclusion. It remains cancelled,
not accepted. Policy changes on existing Outcomes and a dedicated aggregate Kanban
endpoint are not introduced.

## Evidence and acceptance

[Current application captures](screenshots/README.md) ·
[Technical qualification](qualification.md) ·
[62-scenario acceptance matrix](acceptance.json).

Implementation and verification are recorded separately. Existing PR qualification
is historical evidence, not automatically a pass for this change. New browser
journeys cover navigation, read-only opening, cancellation, viewport bounds,
focus return and criteria. Additional backend tests exercise strict rejection,
revision binding and durable cancellation acknowledgement across adapters.

The independent five-participant study is not executed by automation. Reference
latency, physical mobile keyboards and screen-reader/browser combinations remain
separate qualification. See the [existing study protocol](../operational-workspace/human-study.md).

## Implementation boundaries

This is the bounded V1 allowed by the owner plan: it composes the existing
revision-pinned `workspace-cards` and `workspace-view` reads. There is no new
aggregate `workspace-kanban` API, persisted Card/Column model, cross-column
lifecycle drag, parent review protocol, upload service or agent runtime.
Incomplete loaded data stays explicitly partial. This increment is not a claim
of aggregate-wide constant-time rendering or a measured production latency SLO.

The skill informs the interaction contract and recovery design. It does not
replace the existing editorial brand or constitute evidence from user research.
