# Operate work in its contextual Kanban

The spatial Outcome Board remains the default map. Open **Objectives Kanban** from
an Outcome or **Tasks Kanban** from an Objective to operate its direct children.
Use **Context** to switch between direct Tasks and Objectives/Subobjectives. The create action follows that selection; ownership does not change.

## Inspect without leaving the lanes

The Kanban opens as a native modal over the Board. Click a card to open its compact
inspector. Search, filters, lanes, loaded pages and scroll remain available. Wide
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

The inspector omits the standalone Verify criteria/Add material buttons, the action
directory and technical audit disclosure. Relevant operations stay with their
work, responsibility or proof context. Administrative tools remain separate.

## Prepare before making work available

Define the Task in **Backlog**: instructions, expected result, criteria, dependencies,
priority, schedule and Objective placement. In **To do**, execution or a terminal
state, these planning fields are locked by the backend as well as the interface.
Evidence, assessment, incident reporting and authorized operations remain separate.
An eligible To do Task can return to Backlog through an explicit **Defer** operation
before replanning. Releasing a lease or reopening work does not unlock its definition.
Responsibility can be reassigned without granting execution authority.

## Review proof

Move a Task toward **Verification**, or follow its next acceptance requirement.
Review proof shows **What to do** and a checklist of the existing criteria.
For each criterion:

1. Choose **Add or link evidence**. Reuse existing Evidence, record an observation,
   or add new Evidence from an external source or Artifact.
2. Optionally create an **Artifact** describing the produced deliverable. An Artifact
   alone is neither criterion Evidence nor an assessment.
3. Select the criterion checkbox to open its explicit assessment. Choose the result,
   supporting Evidence and rationale; only an applicable `met` assessment checks it.

Each canonical form replaces the current modal, then returns to the refreshed proof
checklist. Linking Evidence does not check a criterion or complete work. Signed
execution/review still requires the authorized external host. Completing a Task,
achieving its Objective and accepting its Outcome remain distinct decisions.

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

The prepared operation replaces the Kanban modal. It shows its item, consequence and
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

[Current decisions and audit](simplification.md) · [Previous interaction contract](workspace-mode-contract.md) ·
[Qualification](simplification-validation.md) · [Screenshots](screenshots/focused-modal/README.md)
