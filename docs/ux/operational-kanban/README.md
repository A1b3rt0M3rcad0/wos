# Operate work in its contextual Kanban

The spatial Outcome Board remains the default map. Open **Objectives Kanban** from
an Outcome or **Tasks Kanban** from an Objective to operate its direct children.
Switch to Direct Tasks or Subobjectives without changing their ownership.

## Inspect without leaving the lanes

Click a card to open its compact inspector. The Kanban remains a workspace, not a
modal. Its search, filters, lanes, loaded pages and scroll remain available. Wide
screens dock the inspector; HD and mobile provide a bounded reading panel. Use
Back to Kanban to close it, or Back to previous item when inspecting a related record.
Back to Canvas restores the map selection, camera and opening control.

The inspector provides:

- **Overview and Next operation:** current state, context and the next pertinent
  capability or a concrete explanation of its requirements.
- **What to do:** actual instructions, constraints, deliverables and definition.
- **Responsibility & authority:** declared responsibility and recorded execution
  authority. Assignment does not acquire a lease; a contract does not mean an Agent
  is online or executing.
- **Criteria & proof:** current definitions, applicable assessments and linked
  material. Open a criterion to inspect and assess its actual condition.
- **Dependencies & impediments:** related work, Issues and Blockers. Resolving an
  Issue does not release its Blocker; release is an explicit operation.
- **Activity history:** loaded on demand, with its frozen history cursor.
- **More actions and audit:** less frequent authorized operations and technical reads.

## Move to prepare an operation

Drag a card to a lane or open its **… Actions** menu and choose a destination.
The menu works without drag or hover. Drop hints explain possible operations;
current capabilities are read again before preparation. The card stays in its
persisted lane until an authorized command succeeds.

| Objectives | Tasks |
|---|---|
| Planned | Backlog: planned work, not yet available |
| In progress | To do: available; acquisition still requires readiness and authority |
| Achieved | In progress: acquired execution, not inferred online presence |
| No invented verification queue | Verification: actual pending review only |
| Cancelled is separate | Accepted: explicit Task completion; cancelled is separate |

A planned Objective can be achieved directly when the existing domain permits it;
Start is not an artificial prerequisite. Backlog → In progress has **two separately
confirmed commands**: make the Task available, then acquire its execution authority.
Reviewing proof never invents a submission lifecycle or moves a legacy Task into
Verification.

## Confirm or resolve requirements

The only modal is the prepared operation. It shows its item, consequence and
existing canonical form, or the current requirements and available preparation
actions. Add or inspect the relevant criterion, proof or impediment, then recheck
the intention. Each command needs its own confirmation.

| Record | Meaning |
|---|---|
| Criterion | A versioned condition that must be true |
| Artifact | A produced deliverable or reference |
| Evidence | An observation or source linked with an explicit stance |
| Assessment | An explicit evaluation against the current criterion definition |
| Conclusion | The explicit decision to conclude the entity |

Evidence does not approve a criterion. A waiver remains an exception, not verified
success. Completing a Task does not achieve its Objective; achieving an Objective
does not accept its Outcome. The existing strict-work policy and cancellation
acknowledgements still apply.

Legacy reservations can be renewed by their holder; expired recovery obtains new
fencing. Contract/signed acquisition, delivery and independent review continue in
the configured authorized host. The browser does not sign or execute work.

## Recover and retain context

A conflict retains your text and requires explicit review of the changed state.
An uncertain response retains its original body and idempotency key; use Retry the
same intention. Exit is guarded while submitting or uncertain. If a write is
confirmed but the refreshed view fails, retry reading the state, not the write.

Search and counters describe **loaded items**. Load more retains lane scroll and
uses the current authenticated cursor. A changed revision requires Refresh;
pages from different revisions are never silently merged.

[Interaction contract](workspace-mode-contract.md) · [Current audit](workspace-mode-audit.md) ·
[Qualification](validation.md) · [Screenshots](screenshots/README.md)
