# ADR-040: Contextual workspace Kanbans and prepared operations

Status: accepted for implementation by the owner's follow-up plan in draft PR #116.

## Decision

This supersedes only ADR-039's presentation decision to host the Kanban in a
native modal. The spatial Outcome Board remains the default map. Its contextual
Kanban is a nonmodal region of the same workspace; a compact nonmodal inspector
coexists with its lanes. A single native dialog owns a prepared operation, its
requirements, or the canonical form. Existing form nodes, listeners and recovery
remain owned by app.js; there is no duplicate command implementation.

Objectives have Planned, In progress and Achieved lanes. Tasks distinguish Backlog,
To do, In progress, actual Verification and Accepted. Cancellation is separate.
The existing public stage field is unchanged: local grouping uses the canonical
kind, lifecycle and display_state already returned by Application. Terminal facts
take precedence over stale review indications. Objectives acquire no review queue.

A drag or accessible card menu prepares an intention from a fresh, subject-scoped
capability. It never commits a lifecycle. Backlog-to-execution has two separately
confirmed commands. Requirements and related records can be inspected and remedied
before rechecking the original intention. Signed operations remain external.

## Invariants and recovery

No new persistent entity, lifecycle, endpoint, migration, permission or authority
is introduced. ADR-039's strict completion policy and cancellation acknowledgement
are unchanged. Criteria, material, assessment and conclusion remain distinct.
The canonical authorization/version/fencing/idempotency pipeline still decides.
Uncertain results preserve the original bytes and key. A confirmed command with a
failed refresh requires another read, never another write. Scope/revocation
invalidates contextual frames and in-flight reads. Map geometry stays mounted;
restoring the origin revalidates its read and focuses the rebuilt opening control.

## Qualification

The owner's [source plan](../ux/operational-kanban/owner-plan-workspace-mode.md)
contains A01–A30. Automated and manual evidence are recorded separately in the
workspace-mode qualification; participant research, physical devices and manual
screen readers must not be inferred from viewport tests or Axe.
