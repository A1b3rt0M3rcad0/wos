# Contextual workspace Kanban contract

Source plan: [owner-approved original](owner-plan-workspace-mode.md), based on
`75a56fe`. [ADR-040](../../adr/0040-kanban-workspace-mode-and-prepared-operations.md)
supersedes only the earlier modal/full-page presentation. Qualification is separate.

## Surfaces and input

The spatial Outcome Board is the default map. Opening a parent Kanban mounts a
nonmodal workspace region; the map remains mounted at its original geometry.
Selecting a card opens a compact nonmodal inspector, not a replacement page.
Wide layouts dock it; HD uses a bounded overlay; narrow layouts use a reading
panel with an explicit return. The lanes remain mounted, with their own scroll.

A card is a reading button, a drag affordance and a separate native disclosure
containing move actions. There are no nested interactive controls and no permanent
move select. Click, keyboard and emulated touch share the same intention path.
Native drag hints must not change the dragged card's geometry. Preparing an
operation waits until dragend; rendering during drop must not detach the source
before the browser finishes its gesture.

Only an operation or its requirements use a native modal. The existing canonical
form/detail/action nodes move into it; app.js retains their listeners, command
payloads, scope, expected version, fencing, draft and idempotency ownership. The
modal traps Tab/Shift+Tab, handles Escape through the existing exit guard and
returns to the item/card. A pending or uncertain mutation cannot be discarded.

## Lane projection and operation matrix

| Entity | Visible lane | Canonical fact | Prepared operation |
|---|---|---|---|
| Objective | Planned | planned | Start or directly Achieve when authorized |
| Objective | In progress | in_progress | Achieve when authorized |
| Objective | Achieved | achieved | Read accepted result; no invented reverse transition |
| Task | Backlog | backlog | Activate; assignment does not make execution authority |
| Task | To do | todo | Acquire under the actual protocol |
| Task | In progress | in_progress, without pending review | Renew/reacquire, release, inspect proof or conclude when authorized |
| Task | Verification | actual awaiting_review | Inspect delivery and follow authorized review; no browser signature |
| Task | Accepted | done | Read conclusion; no automatic parent acceptance |

Cancelled work is separate and appears only on explicit request. Terminal facts
win over stale review hints. The public `WorkspaceCardSummary.stage` remains
unchanged. UI grouping reads kind, lifecycle and display_state; no persisted state,
endpoint, authorization rule or migration is added.

Backlog → In progress prepares Activate first. After its confirmed response and a
fresh projection, the next acquisition form may be prepared; it requires another
explicit confirmation. A drop, menu, inspection or preparation performs no write.
The backend remains the final readiness/permission/version/authority gate.

## Requirements and recovery

A blocked intention presents current reasons and existing authorized remedies.
Criterion/reference inspection uses the same scoped read. Adding a criterion,
recording material, assessing it and concluding remain separate commands. After a
confirmed remedy the original intention is rechecked; no command is auto-submitted.
Assignment, authority and live presence are distinct. A contract is not proof that
an Agent is online. Issue resolution does not silently release a Blocker.

Version conflicts retain authored input and require explicit review. Uncertain
responses retain the same body/key. If the write is confirmed but refreshing
fails, another read reconciles the projection; the write is never repeated.
Scope/principal/revocation invalidates frames and pending reads and clears private
modal content. Leaving the Kanban restores a fresh origin and its rebuilt opening
control without expanding, fitting or panning the background map.

## Evidence boundary

Search and counts describe loaded direct children. Cursor/revision mismatches
require refresh; client pages never masquerade as authoritative totals. Automated
viewport, native zoom, keyboard, actual API and Axe evidence do not establish
participant usability, physical touch-device behavior or screen-reader conformance.
