# Operate an Outcome from its Board

The current Board interaction adds [contextual Kanbans and a compact inspector](../operational-kanban/README.md).
The map stays primary; Outcome and Objective inspectors open one navigable modal.
Details and activity start collapsed. New browser Outcomes visibly default to
strict work; historical/API omission retains compatible completion.

The Outcome Board is the working map of one requested result. Use it to build
the plan, understand the current work, record what happened and review what is
still unverified. Agents and people execute outside WOS; the Board shows the
shared state and the rules that apply to their next action.

This guide describes the source branch implementing PR #116. Published packages
may expose an earlier contract. See the [API and CLI reference](../../operational-workspace-api.md)
for exact requests, limits and compatibility, and the [acceptance matrix](acceptance.json)
for implemented behavior, executed checks and remaining qualification.

## Start with a result, then build the work

1. Create an Outcome with a short name and the state you want to achieve. It
   starts as a draft. Success conditions can be added now or later; WOS does not
   invent a Criterion from the description.
2. Open the Outcome Board. Select the Outcome and add an Objective, or add a
   Task directly when a separate Objective would add no useful structure.
3. Create Tasks from their Objective to keep that context filled in. A quick
   Task needs only a title and starts in `backlog`: planned, without a lease or
   execution contract.
4. Add instructions, constraints, deliverables, context references, scheduling
   and declared participants when the work becomes clearer. Use the item's
   contextual actions while the Task remains in Backlog; there is no need to recreate it.
5. Make a Task available explicitly when it is ready to enter the permitted
   acquisition workflow. Availability does not acquire execution authority. Task planning
   is then locked; an eligible To do Task must explicitly Defer to Backlog to replan.

Quick creation keeps the selected parent visible and offers more details without
requiring them up front. Objectives visibly default to **Required for Outcome**;
you can turn that off before saving. This flag always refers to the Outcome,
including for a subobjective. These human creation choices do not change old
records or the defaults of older API clients. The full Task form also offers an
explicit available-work choice.

To enter several items, choose the batch action, enter one title per line and
build a preview. Review each row's type, parent, requirement and optional success
conditions before confirming. A mixed plan can contain new Objectives and Tasks,
including Tasks beneath an Objective created in the same batch. Changing the
source list requires a fresh preview. A batch saves all its items or none; its
limit is 50 new items and 100 initial Criteria, subject to the existing byte and
domain limits.

## Read the map and the selected item

The default work map emphasizes **Outcome → Objectives → Tasks**. Ownership
answers where work belongs. It does not mean that one item must execute before
another. Tasks directly in the Outcome are supported.

Select a card to open the inspector. The Board and specialist views share this
inspector, with sections appropriate to the selected subject. A Task brings its
requested work, declared participants and recorded authority, current
impediments, Criteria and proof, delivery, relationships and recent activity into
one reading surface. Evidence, Issues, Decisions and other records have their
own pertinent facts rather than fictitious executors or Task checklists.

Cards keep a short operational summary. Separate counts describe work,
verification and waivers. A missing summary is shown as not loaded; a partial
page is not evidence that the remaining count is zero. Open the inspector for
the current facts and available actions.

Use the layer selector or a card's related-record control to explore documentary
records. A contextual preview is bounded; remaining records stay available
through the records layer, search and specialist views. Revealing records keeps
existing card positions and the camera. Pan, zoom, center and fit are visual
operations. Fit is explicit; list view provides an alternative to spatial
navigation. Search can find authorized items outside the loaded map.

The footer offers a small set of relevant actions and **More actions**. A blocked
action explains the observed condition. That explanation is a preview, not a
promise that a later command will succeed: the server checks current permissions,
versions, timing and domain rules again when you save.

## Define success, gather proof, then accept explicitly

| Record or action | What it means |
| --- | --- |
| Criterion | A condition that should be true, owned by an Outcome, Objective or Task |
| Evidence | An observation or source available for examination |
| Evidence link | How that Evidence supports, contradicts or provides context for a specific target |
| Assessment | An authorized evaluation of the relevant Criterion and current material |
| Delivery | Material submitted through the work protocol for review |
| Conclusion | An explicit decision that the item's current acceptance conditions are satisfied |

Add a Criterion in its owner and state what must be true. The new human form
visibly suggests evidence review, with the real attestation and external
evaluation modes also available. Required, optional, retired, met and waived
remain different facts. A waiver records an authorized exception and its reason;
it does not become verified proof.

Open a Criterion to inspect its definition, current revision, linked Evidence
and applicable assessments. From there, **Add evidence** keeps its owner and
Criterion selected. Choose an external source, an existing Artifact, a recorded
observation or an existing Evidence record as the form permits. Review the
visible stance and explain its relevance. The registration and links save in
one transaction; an invalid link leaves no partial registration.

The same Evidence can have different stances on different targets. Contradictory
Evidence remains visible. Linking Evidence does not assess the Criterion. An
assessment of an older Criterion revision or a different submission/digest does
not approve the current work. Retracted proof is not counted as current proof.

An internal observation retains the real note, authenticated author and
registration time. It is a participant's recorded observation. A statement that
someone ran tests does not claim that the WOS server ran those tests. Artifact
and Evidence URLs are references; WOS does not fetch their content during these
operations.

Task completion does not automatically achieve an Objective. Objective
achievement does not automatically achieve an Outcome. Review and conclude each
level explicitly. Outcome preparation includes its own active required Criteria
and its actual required Objectives, including those outside the visible branch.
An eligible planned Objective can be achieved without an artificial Start step.
The conclusion form shows the current number of open child Objectives and Tasks,
a bounded list of their names and the scope consequences. Under `compatible`,
confirm explicitly when work remains open. Under `strict_work`, applicable open
work prevents completion; each cancelled exclusion requires a reasoned,
revision-pinned acknowledgement and is never counted as accepted. Concluding an Outcome closes its operational scope until
it is reopened. Concluding an Objective closes it to new children; existing
descendants retain their own workflow rules. Neither action silently completes,
cancels or waives child work or its Criteria.

The confirmation carries the observed Outcome revision. If another participant
changes that state, **Review current impact** reloads the impact, retains your
reason and clears the acknowledgement. Review it and confirm a new intention;
WOS does not silently reuse an obsolete preview.

## Record updates, problems and deliverables in context

**Record update** appends a documentary note to the selected item. Choose its
documentary type and enter the observation. Writing “finished” changes no
lifecycle; reporting a concern in a note creates no Issue or Blocker. To correct
your own note, append a correction linked to it. The original body and authorship
remain intact. Non-archived terminal work can retain documentary updates without
being reopened; archived scopes keep their write restrictions.

**Report issue** starts with the affected work selected. An Issue describes a
problem. Choose explicitly whether it prevents work from continuing. When it
does, the compound operation creates an explicit Blocker with a valid target and
scope. Task impacts are direct; Objective and Outcome scopes follow the server's
permitted propagation rules. For a subtree, **Preview blocking impact** shows the
actual target and affected open work, with explicit counts and coverage. Review
and acknowledge it before saving. A changed scope requires another preview; a
rejected compound intention leaves neither a partial Issue nor a partial Blocker.

Resolving an Issue alone does not release its Blockers. The contextual resolution
flow shows the recorded impacts and releases only those explicitly selected.
Other active Blockers can still prevent work. A new operational impediment on
terminal work may require reopening that work through its legitimate workflow.

Record an Artifact when material was produced or needs to be referenced. Choose
the real relation: production differs from general context. Propose a Decision
when a choice and its rationale need to be recorded. Registering material does
not submit a contracted delivery; accepting a Decision does not release an
associated Blocker automatically. Evidence about a Blocker belongs on its cause
or affected work, not on a fictitious direct Evidence link to the Blocker.

## Distinguish responsibility from authority

Owners and assignees express declared responsibility. They do not authenticate a
person, acquire a lease, grant a contract or establish that a process is running.
The inspector separates those declarations from current execution authority,
the last executor, the current reviewer and the authors of recorded facts.
Unknown authority remains unknown; WOS does not infer online presence.

In a signed workspace, continue execution or review with an authorized host
Profile. The handoff pins the real Workspace, Outcome, target and version;
choose a Profile whose server, identity and permissions match that work.
The browser does not choose a credential, receive a private key, sign
for an agent or fall back to unsigned acquisition. Signed contract activity
retains its authority checks; documentary notes are a separate operation.

## Change structure deliberately

Dragging a paper changes its local visual position. To change persisted
ownership, use the Task's Objective action or the Objective's parent action.
Choose **Preview move** to review the old placement, selected destination,
affected work, applicable Blockers before and after, and Task readiness where
pertinent. The preview uses the same guarded rules as the eventual command;
an ineligible destination explains the preparation it needs. Acknowledge the
observed impact before saving. Changing the destination invalidates that preview.
A Task can move directly under its Outcome by explicitly clearing its
Objective. Its identity, Criteria, Evidence and history are preserved.

Structural commands check scope, current versions, cycles, dependencies and
execution/review freezes. Moving a planned Objective also checks its affected
subtree. A blocked move revokes no contract. If the impact preview is stale,
review the new state before confirming a new intention.

**Add dependency** is a separate action with an explicit direction: “A depends
on B.” Hard dependencies affect operational readiness; advisory dependencies
remain informative. A Task cannot depend hard on its own Objective because that
creates an impossible completion condition. Older invalid configurations remain
visible for deliberate correction; WOS does not remove them silently.

## Read history and coverage honestly

Recent activity loads when the inspector opens. The timeline identifies domain
events, documentary notes and authorized contract activity, without showing the
same WorkActivity twice as an update and its registration event. Descendant
entries identify their actual subject rather than becoming actions of the parent.

Older pages share a frozen historical frontier. New facts do not enter the middle
of that sequence. Refresh explicitly to start from current activity. A structural
change that affects descendant membership can require restarting that history
query. Registration time determines recorded order; an optional observation time
is displayed separately. Open a note to read its complete body beyond the bounded
preview.

Coverage is per section: complete, partial, not loaded, unavailable or not
applicable. **Load more** expands the named information, not the whole database.
Neither the visible graph nor an ended bounded page proves that every entity or
document has been loaded.

## Recover without losing or duplicating an intention

A preparation error describes a missing domain condition. Use its contextual
remedy, return to the retained draft and review it again. Adding the missing
Criterion does not assess it or automatically resubmit a conclusion. A genuine
version conflict instead requires comparing base, local and current state before
forming a new save intention. Structural moves and assessments are reexamined;
their versions are never rebased automatically.

If a response is uncertain, recover the original request with its original bytes,
scope and idempotency key. Do not edit that request and resend it under the same
key. If the write succeeded but refreshing the view failed, refresh the view
rather than creating the record again. Changing identity or scope invalidates
obsolete reads and drafts so private content is not restored by a late response.

The CLI offers the same allowlisted planning and documentary operations through
`planning apply` and `planning recover`; `inspect` reads the shared projections.
See the [contract reference](../../operational-workspace-api.md#use-the-cli-and-recover-safely)
for files and commands. Signed execution and review keep their separate Profile
workflow.

## Self-hosting and qualification

These capabilities belong to the existing WOS product. They require no Cloud
service or separate dashboard database. Memory, SQLite and PostgreSQL use the
same Application rules. Persistent adapters add notes and bounded-read indexes
through additive migrations; existing creation defaults and signed protocols are
preserved. Back up before upgrading and follow the [migration evidence](migrations.md).

The [acceptance matrix](acceptance.json) separates source implementation, executed
automated checks, browser qualification and pending work. Performance targets are
measurements to establish in the stated environment, not an SLA. Screenshots and
automated journeys do not replace the owner's requested independent
five-participant comprehension study; that study remains explicitly pending.
The [study protocol](human-study.md) provides concrete tasks and a blank results
template for that independent evaluation.
