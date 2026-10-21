# ADR-043 — Versioned operational profiles and Objective gates

Status: accepted for implementation by the owner's 2026-10-21 authorization of the operational plan; qualification and rollout remain pending.

## Context

WOS already persists Outcome, Objective, and WorkItem lifecycles, while views
derive readiness, blocking, verification, and other operational projections.
The operational plan adds stricter gates for new work without retroactively
reinterpreting existing records. It also proposes deterministic Objective
completion under an explicit delegation. These rules must remain distinct from
the Namespace execution-authority protocol and from Outcome acceptance.

Existing decisions in ADR-0039 and ADR-0042 remain in force: strict completion
does not turn cancellation into acceptance; Task placement and Issue ownership
are structural; Issues, Blockers, and dependency relations have separate
semantics. Existing signed-contract decisions in ADR-0018 and ADR-0022 through
ADR-0027 remain authoritative for signed work.

## Decision

This ADR defines the profile and lifecycle-gate baseline. ADR-045 is the
detailed source for Objective completion authority, eligibility, and cycles;
ADR-046 is the detailed source for protocol-specific Task delivery and review.
The corresponding sections below summarize those boundaries without creating
competing variants.

### Operational profile is separate from authority protocol

Add a versioned operational profile to each Outcome:

- `compatibility_v1` preserves the historical behavior of existing Outcomes
  during an explicit migration period.
- `operational_v2` applies the operational gates in this decision.

The profile is a business-rule version, not a Namespace protocol or credential
mode. It does not select `legacy`, `contracts_v1`, or `signed_contracts_v2`, and
it cannot weaken the authority, signatures, review floor, or cutover rules of
those protocols. Each profile uses the genuine capabilities of its selected
protocol to satisfy the same applicable business obligations.

After rollout, newly created Outcomes default to `operational_v2` across HTTP,
MCP, SDK, CLI, and browser flows. Existing Outcomes enter the expand phase as
`compatibility_v1` and move only through an explicit, previewed, authorized
migration. Compatibility is a migration bridge, not an executor-selectable
bypass. Creation of new legacy-profile Outcomes by ordinary clients is to be
closed after the installation's open Outcomes are migrated. Historically
completed legacy records remain readable; reopening requires controlled
migration or an explicitly supported compatibility path.

### Hierarchy gates are structural lifecycle rules

`Outcome → Objective → [Subobjective | Task]` remains the structural hierarchy
defined by ADR-0042. A Task's Objective owner and an Issue's Objective owner are
not `depends_on` relations. The fact that a parent Objective must be in progress
before a child can start is a lifecycle gate, not an implicit edge in the
dependency graph. Do not persist hidden dependencies or propagate Blockers
through `depends_on` edges.

Conversely, a valid acyclic `depends_on` graph is not sufficient to prove that
execution can progress. Validation must reject combinations of hierarchy,
activation, and hard dependencies that create impossible waits, including a
Task depending on its own Objective/required ancestor or an Objective depending
on a descendant whose work requires that Objective to start. Validate the final
graph atomically during batch edits and structural reparenting.

Objective lifecycle remains `planned`, `in_progress`, `achieved`, or
`cancelled`; Task lifecycle remains `backlog`, `todo`, `in_progress`, `done`,
or `cancelled`. `ready`, `blocked`, `attention_needed`, and `accepted` are
derived projections, not additional persisted lifecycle values. Objective has
no synthetic Verification state. Task appears in Verification only when a real
pending delivery or signed review case exists.

In `operational_v2`, starting an Objective requires an active, non-archived
Outcome, applicable ancestors in progress, satisfied hard dependencies,
required criteria defined, and no applicable active Blocker. Activating a Task
from Backlog requires its required Objective and applicable ancestors in
progress, a live Outcome, a prepared definition, at least one active required
Criterion, satisfied scheduling/dependency gates, and no applicable Blocker.
Atomic creation directly into To Do applies the same gates to the final state
and rolls back completely on failure. Acquisition rechecks current readiness,
authorization, scope, and exclusive authority. The same domain predicates must
inform mutations, capabilities, ready-work queries, and workspace projections;
read projections do not grant permission.

Accepting one Task is never itself Objective or Outcome achievement. A
separately authorized derived Objective conclusion may follow under ADR-045;
Outcome acceptance remains a distinct explicit operation. Objective reopen,
start, and achievement are distinct operations. Reopening preserves
historical Conclusions, requires a reason and impact review, and does not
silently start the Objective. A Task under an achieved Objective can resume
only after the applicable Outcome and Objective/ancestor chain is explicitly
reopened and started.

### Task verification remains a real protocol operation

For `operational_v2`, acceptance of a Task requires an identifiable delivery
and an authorized persisted decision. The exact records, state invariants,
correction rounds, and transport contracts are defined by ADR-046. The legacy
lease path may receive a bounded native delivery record;
`contracts_v1` delivery references its actual
`WorkSubmission`; `signed_contracts_v2` continues to use its existing signed
submission and `ReviewCase`. Do not duplicate signed deliveries in a parallel
unsigned queue or accept a signed direct-return path where the frozen policy
requires independent review. In-flight signed contracts are not retroactively
rewritten.

Definitions may be prepared in Backlog. Starting work freezes the operational
definition according to ADR-0041 and the authorized command contract. Criteria
are definitions; Assessments are decisions bound to the exact Criterion
revision and applicable delivery/completion cycle. Evidence, Artifact,
Assessment, and Conclusion remain distinct. An Evidence record, checkbox,
percentage, Task completion, or waiver never fabricates a `met` Assessment.
Where a criterion requires evidence, the existing verification mode and
authorization remain mandatory.

### Objective completion policy is explicit and deterministic

Introduce `ObjectiveCompletionMode` with `manual` and `automatic`; do not reuse
the Outcome's `completion_policy` (`compatible` / `strict_work`) for this
different question. `strict_work` defines which work is an obligation;
`completion_mode` defines whether an explicitly delegated rule may initiate an
Objective Conclusion once its obligations are met.

New `operational_v2` Objectives may use `automatic` only when the caller has the
dedicated permission to delegate completion. The selected mode must be visible
at creation. Without that permission, the client selects `manual` explicitly or
asks an administrator to configure it; omission must not silently delegate
authority. Existing Outcomes migrate to `manual` unless an authorized
preview-and-migration explicitly selects automatic. Changing profile or mode
uses expected versions, an authorized reason, and a specific permission;
policy cannot be reinterpreted while live execution or review is in flight.

Automatic achievement is a deterministic integrity rule, not an autonomous
planner, agent runtime, or judgment of semantic quality. Its detailed
authorization, eligibility report, provenance, cycle, transaction, and
reconciliation contract is defined by ADR-045. It is eligible only
when all of the following can be proven from a coherent, complete current
snapshot:

- the Objective is in progress, its Outcome is active and not archived, and
  the current automatic delegation is valid;
- the applicable work set is nonempty, every applicable Task is accepted with
  a valid current Conclusion, and every required descendant Objective is
  achieved under the configured `strict_work` semantics;
- all current required Objective Criteria have actual applicable `met`
  Assessments; the system never creates an Assessment from Task completion;
- there are no pending owned Issues, unsatisfied hard dependencies, applicable
  active Blockers, open execution authorities, or pending reviews;
- there is no cancelled work or waiver that requires an explicit manual
  decision, and the current proof and closure policies are satisfied.

The completion report distinguishes eligibility facts from the requesting
Principal's authority. It returns complete counts and bounded examples with
explicit partial-coverage reasons. An incomplete/truncated read is never
eligible. An empty Objective does not auto-complete. Cancelled work is not
accepted; any existing acknowledgement-based exclusion remains an explicit
manual decision. Outcome acceptance always remains explicit, even when all
Objectives are achieved.

The automatic Conclusion references real Objective-Criterion Assessments. Its
delegation records the authorizing Principal and Credential, policy revision,
scope, and cycle. The internal automation identity is narrowly limited to the
deterministic rule and cannot assess Criteria, create evidence, issue
credentials, or accept Outcomes. A human-final-approval policy selects manual
mode; the system must not manufacture an independent reviewer identity.

Reevaluation follows authorized mutations that can change eligibility and
runs under the existing Outcome coordination guard and UnitOfWork. The ordinary
case commits the triggering mutation, derived Conclusions, versions, events,
outbox records, and idempotent result atomically. A failed storage operation
rolls back the whole operation; a merely unmet precondition does not undo an
otherwise valid Task acceptance. Bounded reconciliation, if required for a
large cascade, rechecks only the deterministic rule through existing durable
mechanisms; it does not schedule or execute work. Reopening advances a
completion cycle so old Assessments do not silently satisfy new obligations.

### Presentation and recovery

Kanban lanes are projections. Drag-and-drop is the only lane-change interaction;
cards do not gain status selects, status menus, or separate move actions. The
drag interaction must remain operable by keyboard and touch using the same
prepared command path. A drop prepares an existing authorized operation and
never writes a status directly. Blocked operations identify the unmet predicate
and a valid next action. The server rechecks current versions, permissions,
parent state, and protocol authority on submission. Uncertain results retain
and reconcile the exact original intent; the client does not issue a replacement
mutation.

## Consequences

- New work receives consistent lifecycle and proof gates across every
  transport, while existing history is protected by an explicit migration
  boundary.
- Structural parent readiness cannot be confused with ownership, a Blocker,
  or an operational dependency.
- Automatic Objective conclusions require explicit delegation and real proof;
  Task completion cannot synthesize Objective assessment or Outcome acceptance.
- Implementation needs additional policy, completion-report, delivery/review,
  provenance, and reconciliation contracts. This ADR alone does not claim those
  capabilities exist.
- During migration, the application must explain the active profile and must
  not silently change the authority protocol or reinterpret an in-flight
  signed record.

## Rejected alternatives

- Select compatibility behavior from omitted client fields: this would create
  a permanent client-controlled bypass for new records.
- Encode parent readiness as hidden `depends_on` edges: hierarchy and
  prerequisite relations have different ownership and Blocker semantics.
- Persist UI-only stages such as Ready, Blocked, Accepted, or Verification:
  these are projections of lifecycle and actual operational records.
- Auto-achieve an Objective because all Tasks are done: Task completion does
  not prove Objective Criteria, and an empty/cancelled work set is not accepted
  work.
- Let Objective automation accept an Outcome or act as a general service
  Principal: these powers exceed the delegated deterministic rule.
- Relax signed return/review rules or create unsigned parallel review state
  for signed work: this would weaken already accepted protocol authority.

## Verification required before rollout

Before the profile is enabled, qualify migration
preview and zero-orphan behavior, all lifecycle gates across direct and
composed commands, complete bounded eligibility reads, actual criterion proof,
signed and unsigned verification parity, same-Outcome concurrency, replay,
reopening cycles, SQL restore, and HTTP/MCP/SDK/CLI parity. Preserve ADR-0018
and ADR-0022–ADR-0027 signed protocol evidence. The canonical design and
`AGENTS.md` record target behavior; `ROADMAP.md` records actual status. This ADR
does not claim the capabilities already exist.
