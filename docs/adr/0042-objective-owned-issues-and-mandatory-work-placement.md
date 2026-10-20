# ADR-042: Objective-owned Issues and mandatory Task placement

Status: accepted for implementation by the repository owner's request
(2026-10-20). Implementation is staged in PR #116; see the live status in
`ROADMAP.md` and the [implementation plan](../ux/operational-kanban/issue-and-dependency-plan.md).

## Context

The current model allows a Task (`WorkItem`) to exist without an Objective and
does not assign an Issue to the Objective accountable for resolving it. An
Issue's affected references describe scope, but they do not provide stable
ownership. This makes Objective completion unable to account for its own open
Issues, and makes issue lists and assignment ambiguous.

Issues, Blockers, dependency relations, structural hierarchy, and evidence
have distinct meanings. The change must keep them distinct and preserve
existing execution, acceptance, idempotency, authorization, and audit rules.

## Decision

### Structural placement

The structural hierarchy is `Outcome → Objective → [Subobjective | Task]`.
Every Task has exactly one direct Objective in the same Outcome. The existing
Outcome scope remains the isolation and consistency boundary; it does not mean
that a Task is structurally attached directly to the Outcome. Objectives may
remain top-level or nested under another Objective.

Creating a Task, creating a plan batch, changing its Objective, or reparenting
it must preserve this invariant. A plan batch may create its Objective and
Tasks atomically. It must never create a temporary orphan or silently choose
an Objective. The existing Backlog-only Task-definition rule in ADR-041
continues to apply.

### Issue ownership and assignment

Every Issue belongs to exactly one Objective in its Outcome. `affected_refs`
remain an informative, independently validated set; they do not infer or
change ownership. Issue ownership is not an Objective child relationship,
dependency, Blocker, or execution assignment.

An Issue may have one optional declared assignee. The assignee is distinct from
`reported_by`, and assignment is not a lease or execution authority. Assignment
uses expected versions, authorization, audit events, and history; it does not
grant authority over Tasks.

### Objective achievement gate

Achieving an Objective requires that no Issue owned by that Objective is
pending. `open` and `investigating` Issues are pending. `resolved` and
`wont_fix` are terminal; `wont_fix` requires a reason and does not satisfy or
waive a success criterion. A `duplicate` Issue satisfies this gate only when
its canonical duplicate chain terminates at a `resolved` or `wont_fix` Issue.
The chain must be valid, finite, and cycle-free.

The same predicate is used by command validation and capability/precondition
projections. This gate does not make an Issue a Blocker, and an Issue alone
does not prevent Task completion. Task criteria, applicable Blockers,
hard dependencies, execution authority, delivery, and review retain their
existing independent rules. An ancestor is not blocked by every descendant
Issue; existing structural Objective-achievement obligations determine which
child Objectives must first be achieved.

Reopening an owned Issue while its owner Objective is achieved is denied. The
Objective must first be explicitly reopened through its existing authorized
operation. Closing or reopening an Issue never changes a Blocker implicitly.

### Blockers and dependencies remain separate

A Blocker represents an explicit impediment on one named Outcome, Objective,
or Task target. Its `direct` or `subtree` propagation follows structural
Objective hierarchy only. It does not propagate through `depends_on` edges.
Resolving an Issue does not release any Blocker; a caller explicitly selects
and confirms each Blocker release using the existing compound operation.
Reporting an Issue does not create a Blocker unless the caller explicitly
reports one.

`depends_on` remains a directed relation from dependent `source_ref` to
prerequisite `target_ref`. `hard` dependencies affect readiness and existing
completion gates; `advisory` dependencies are contextual and do not block.
Both remain acyclic and within the same Outcome. A Task cannot hard-depend on
its own Objective. Dependency edits use the actual dependent source and retain
ADR-041's Backlog-only definition guard for outgoing Task dependencies. An
incoming dependency is an edit to its true source, not to the selected target.

Batch dependency edits validate the entire proposed graph atomically, including
cycles, duplicate active edges, scope, relation types, source editability, and
the resulting graph. They do not perform automatic defer, release, or lifecycle
changes.

### Storage rollout and compatibility

Schema changes use expand, explicit backfill, validation, and contract stages.
The audit and backfill are read-only/dry-run by default and identify candidates
without assigning them. Any actual mapping must be explicit, deterministic,
reviewable, scoped, and recorded. Ambiguous or unmapped legacy records stop the
contract migration; no migration may guess an Objective from `affected_refs`,
title, hierarchy proximity, or a sole available Objective.

The final contract can make Task placement and Issue ownership required only
after all supported storage adapters and persisted data have passed the
zero-orphan gate. API, HTTP, MCP, Go SDK, and CLI changes must share the same
application rules. Existing request replay, expected versions, authorization,
and Domain Event semantics remain in force.

## Consequences

- Issue lists and filters can use explicit Objective ownership and optional
  assignee without guessing from affected references.
- An Objective's own unresolved operational problems are visible at its
  achievement boundary, while a Task may still be completed when an unrelated
  Issue exists and its own actual gates are satisfied.
- Task creation flows must establish an Objective before Task creation or
  atomically create both.
- Existing installations require a reviewed mapping before the final schema
  constraint can be applied. A safe deployment may pause at the expanded
  schema until that data gate passes.
- The UI may project workflows as lanes, but every move remains an explicit
  authorized domain operation; no lifecycle setter is introduced.

## Rejected alternatives

- Infer Issue ownership from the first `affected_ref`: an Issue can affect
  several entities and scope does not establish accountability.
- Keep direct-Outcome Tasks as a permanent second hierarchy path: it makes
  planning and responsibility ambiguous and prevents consistent Objective
  reporting.
- Turn every Issue into a Blocker: detection and operational impact are
  separate facts.
- Let Issues cascade through dependency edges: dependency readiness already
  represents prerequisite state and is not structural Blocker propagation.
- Infer a legacy owner during migration: this can silently assign
  responsibility to the wrong Objective and invalidate the user's plan.

## Verification

Implementation acceptance includes domain and application tests, memory,
SQLite, and PostgreSQL storage contracts, migration/backfill safety, HTTP/MCP/
SDK/CLI parity, race checks for graph updates, browser operations, keyboard
access, and the onboarding regression suite. The ROADMAP must identify any
gate that remains unverified; this ADR alone does not mark implementation
complete.
