# Operate work without leaving the Kanban

The spatial Outcome Board remains the central map. Open an Outcome's Objectives
Kanban or an Objective's Tasks Kanban to plan and operate its **direct** children.
Subobjectives and direct Outcome Tasks stay separate. Counts and search describe
loaded cards; use Load more rather than assuming the visible page is the whole plan.

## Read and operate an item

Select a card to replace the columns with its focused work page. No second sidebar
or additional modal is opened. Back to Kanban restores its filters, horizontal
position and column scroll; Back to previous item returns from a criterion or
related record. Escape leaves an item first, then closes the Kanban; breadcrumbs
return to earlier Kanban contexts. An unresolved mutation keeps its form.

Collection filters and parent-creation actions are hidden while an item is focused;
Refresh stays available. A missing assessment directs the primary action to
review criteria, rather than presenting blocked completion as the next operation.

The work page brings together:

- **Next operation:** a current server capability, with a confirmation form or a
  concrete explanation of its requirements. It never silently changes status.
- **What to do:** authored instructions, constraints, expected deliverables and
  context. An empty definition is described honestly; use its definition action.
- **Responsibility and execution authority:** declared assignees/owners, recorded
  executor/reviewer and expiry. Assignment grants no lease or permission. Recorded
  authority does not establish live presence or that an external process is running.
- **What is missing:** readiness, dependencies, real blockers and pending criteria.
  Open their references in the same work page, resolve the correct entity and return.
- **Prove and verify:** register the produced Artifact, record or link Evidence,
  inspect each Criterion, and explicitly record an Assessment. Evidence alone never
  marks a criterion met. A waiver is an exception, not verified success.
- **Execute and conclude:** acquire current authority, perform the work externally,
  record its actual result and explicitly conclude when all backend gates permit it.
- **What happened:** Activity history loads on demand; its continuation stays bound
  to the actual frozen history cursor. Authored updates and decisions remain records.

Less frequent operations remain grouped by purpose and in All item actions. The
same existing forms, permission checks, expected versions, fencing and idempotency
pipeline are used inside the Kanban.

## Move a card to prepare an operation

Drag onto a column, or use the card's labelled **Move to…** select with a single
pointer or keyboard. A drop is a request to prepare the next operation. Review
and confirm the existing form; only a persisted response changes the projection.

| Target column | Legitimate next operation |
|---|---|
| To do | Activate a backlog Task or explicitly release a live legacy reservation, when available. Objectives have no invented reverse transition. |
| In progress | Acquire a ready Task's lease/contract, reacquire expired legacy authority, or Start Objective. A backlog Task must first be made available. |
| Verification | Inspect delivery, criteria and review requirements. This does not invent a submission lifecycle. Signed submission/review stays external. |
| Accepted | Explicitly Complete Task or Achieve Objective, subject to current criteria, assessments, dependencies, blockers, authority and conclusion policy. |

Invalid or unauthorized moves leave the persisted work unchanged. Required
criteria, blockers and execution authority are revalidated on the server, including
when they change between opening and saving a form. Completing a Task never
achieves its Objective; achieving an Objective never automatically accepts the Outcome.

## Reservations and external execution

For a legacy Task, Reserve task acquires the real exclusive time-limited lease.
Extend my reservation retains its claim and fencing token. Once expired, use
Reacquire expired reservation: the existing command issues a new claim and higher
fencing token. The previous holder cannot complete with stale authority.

Contract-backed work uses **Continue in authorized host**. Use the supported
configured external host for acquisition, signed delivery, independent review,
correction and continuation. The existing handoff shows the exact record/context.
The browser does not generate signatures, assume an agent identity or execute work.
These operations are not advertised as browser-complete when the protocol requires
an external actor. Legacy recovery actions are unavailable under contract protocols.

## Recover without losing intent

A version conflict retains your input and offers the canonical comparison/remedy
flow. Refresh and review the actual state; the UI never silently replaces the
expected version or retries a changed semantic operation.

If a response may have been lost after commit, the form retains its pending intent.
Retry uses the **same body and idempotency key**. Escape/Close cannot discard an
unresolved intent. Confirmed saves refresh parent, child, ancestors and cards against
one coherent revision. Optional spatial graph enrichment cannot prevent this
operation; the Board can be refreshed independently after returning.

[Audit and priorities](audit.md) · [Verification](validation.md)
