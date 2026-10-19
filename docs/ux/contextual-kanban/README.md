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

## Strict completion

New browser-created Outcomes visibly default to strict work. API callers that
omit `completion_policy` retain compatible behavior. See ADR-039 for exactly which
children are obligations. A cancellation must have its own reason and a pinned,
explicit exclusion acknowledgement on the parent Conclusion. It remains cancelled,
not accepted. Policy changes on existing Outcomes and a dedicated aggregate Kanban
endpoint are not introduced.

## Evidence and acceptance

Implementation and verification are recorded separately. Existing PR qualification
is historical evidence, not automatically a pass for this change. New browser
journeys cover navigation, read-only opening, cancellation, viewport bounds,
focus return and criteria. Additional backend tests exercise strict rejection,
revision binding and durable cancellation acknowledgement across adapters.

The independent five-participant study is not executed by automation. Reference
latency, physical mobile keyboards and screen-reader/browser combinations remain
separate qualification. See the [existing study protocol](../operational-workspace/human-study.md).
