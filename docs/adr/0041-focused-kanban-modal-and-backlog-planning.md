# ADR-041: Focused Kanban modal and Backlog-only Task planning

Status: accepted by the repository owner's explicit follow-up (2026-10-20).
Supersedes ADR-040 only for presentation and Task definition editing.

## Decision

The contextual Kanban is a native modal over the persistent spatial Outcome
Board. An operation temporarily replaces that modal: two modal dialogs are never
open together. Returning preserves the Kanban context, filters, selection and
scroll. The Board camera remains mounted and unchanged.

Kanban inspectors omit the standalone Verify criteria/Add material controls, technical
audit disclosure and More actions directory. Meaningful work, responsibility,
criteria/proof, impediments and history remain. The header uses a plain context
caption rather than breadcrumbs. A context selector controls both the entity
collection and its creation action. Overview is direct navigation; specialist
views remain reachable outside the Board.

Review proof contains instructions and the existing criteria checklist. A check
represents an applicable, explicit `met` Assessment, never Evidence presence.
Each criterion can open the existing evidence editor (new or existing Evidence,
including Artifact-backed observations) and assessment form. Creating an Artifact
alone does not link Evidence or assess a criterion. Signed review remains external.

Task definitions can change only in Backlog: title/description/priority,
instructions/execution specification, not-before, Objective placement, criterion
definitions and outgoing dependencies. Commands and capability reads share the
gate; all transports must enforce it. Incoming dependencies owned by other work
are not edits of this Task's definition. Atomic creation with initial criteria
and `lifecycle=todo` remains supported: the submitted initial definition is sealed
as a single creation, without a post-activation editing window.

The existing explicit Defer command remains the audited way to return eligible
To do work to Backlog for replanning. Release and Reopen return to To do and do
not unlock planning. No implicit defer, lease revocation or status setter is added.
Assignment is operational responsibility, distinct from the agreed definition
and execution authority; evidence, artifacts, assessments, issue/blocker reporting,
notes, acquisition, delivery, review and explicit conclusions remain governed by
their existing independent rules. No migration or new persisted lifecycle is needed.

## Card movement follow-up

The owner subsequently requests drag-and-drop-only lane movement on cards.
Remove the card stage menu/select and its duplicate destination controls.
A drop still prepares the existing operation with current capabilities and
explicit confirmation; it never persists a visual status directly. Contextual
operations remain in the selected-item inspector, including keyboard/touch
access to their existing canonical commands. No new domain transition is added.

## Verification

Test definition mutation denial after activation, unchanged versions/state on
rejection, capability agreement, explicit defer/replanning and completion with
current proof. Test one-open-dialog, cancellation/return, contextual creation,
keyboard/focus, small/large viewports, and the real evidence/assessment flow.
Qualification receipts must distinguish new evidence from historical PR checks.
