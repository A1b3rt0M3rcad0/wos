# ADR-045: Policy-driven Objective completion

Status: accepted for implementation in Draft PR #116 by the owner's
2026-10-21 instruction. Qualification and rollout remain pending; this ADR
does not claim that the capabilities are implemented.

## Context

The existing `CompletionPolicy` distinguishes compatible completion from strict
work accounting. It does not say whether a person or an explicitly delegated
rule initiates an Objective conclusion. Earlier repository rules prohibit
automatic Objective achievement and allow a manual `AchieveObjective` from
`planned`; the operational plan changes those rules for explicitly configured
operational Outcomes. The change must preserve historical behavior during
migration, retain the distinction between Objective and Outcome decisions, and
avoid turning completion into a judgment performed by an agent or heuristic.

An Objective can contain nested Objectives, Tasks, criteria, owned Issues,
dependencies, and Blockers. A client page or loaded Board sample cannot prove
that all applicable work is complete. Reopening also creates a new operational
cycle in which old conclusions and assessments must remain historical rather
than silently satisfying current obligations.

## Decision

### Separate the policy axes

Keep Outcome `completion_policy` (`compatible` or `strict_work`) responsible
for the scope of work that participates in a completion gate. Add a distinct
Objective `completion_mode` (`manual` or `automatic`) to define how an eligible
Objective may be concluded. Neither field substitutes for the other.

Automatic completion is valid only for `operational_v2` and requires
`strict_work`. Manual completion remains available under the applicable
profile's ordinary authorization and conclusion gates. Both manual and
automatic achievement in `operational_v2` require an `in_progress` Objective;
the profile's lifecycle gates are detailed in ADR-043. `automatic` is a
delegation of a fixed deterministic rule; it does not authorize an agent,
service, or API caller to make a semantic judgment about whether work is good.

### Defaults, authorization, and migration

- A newly created Objective in the operational profile may default visibly to
  `automatic` only when its creator is authorized for
  `objective:configure_completion`. The creator explicitly delegates this
  policy as part of creation. If that authority is absent, the caller must
  choose `manual` or receive an authorization error; the server must not
  silently grant the delegation.
- Existing Objectives migrate to `manual`. Moving an existing Objective to
  `automatic`, or changing its mode, requires the specific configuration
  permission, `expected_version`, a reason, and an auditable policy revision.
  The update must be rejected while an open execution or review could be
  reinterpreted, unless a reviewed migration explicitly drains or resolves it.
- The authorization record binds the authorizing Principal and credential,
  Objective and Outcome scope, policy revision, and server timestamp. The
  automation identity cannot authorize its own policy, issue credentials, add
  proof, assess criteria, or accept an Outcome.
- Where an organization requires a human final decision or a policy requires
  an independent reviewer for the Objective conclusion, the mode is `manual`.
  Automation must not impersonate an independent human or reviewer.

### Deterministic eligibility

Automatic achievement is an Application decision over a complete, coherent
snapshot under the Outcome coordination guard. A bounded sample may be returned
for display, but it cannot establish eligibility. The report exposes separate
`eligible`, `automatic_allowed`, and `manual_allowed` results, with the
Objective version, Outcome revision, current completion policy and cycle,
complete obligation totals, and reason codes. If completeness or consistency
cannot be established, the Objective remains unchanged.

The automatic predicate requires all of the following:

1. The Objective is `in_progress`; its Outcome is active and not archived; and
   the current policy delegation is valid and has not been superseded.
2. The applicable work set is non-empty. Work includes Tasks owned by the
   Objective and its existing Objective subtree. An Objective with no Task or
   other descendant work does not close automatically; a parent with actual
   work in descendants is not treated as empty merely because it has no direct
   Task.
3. Every applicable Task has a valid accepted completion, including its real
   delivery/review where required. A cancelled Task is not accepted and blocks
   automatic achievement; excluding cancelled work requires a manual decision
   and explicit acknowledgement.
4. Required descendant Objectives are achieved under the applicable strict-work
   rules. No parent, child, Task, Issue, or Blocker lifecycle is changed merely
   because another item completed.
5. Every required active Objective criterion has a current, applicable
   `met` assessment for the current criterion revision and completion cycle.
   Work completion does not create or imply an assessment. Evidence presence,
   `waived`, `not_met`, or `inconclusive` does not satisfy this automatic gate.
6. The Objective has no pending owned Issue, including a duplicate whose
   canonical chain is not closed; every hard dependency is satisfied; no
   applicable Blocker is active; and no execution authority or review is open
   in the applicable work set.

The report must also make visible cancellations, waivers, incomplete scans,
pending reconciliation, and any other condition that requires a human decision.
It may not convert a missing, paginated, or failed query into a zero count.

### Conclusion provenance and Outcome boundary

After the predicate is revalidated in the same UnitOfWork, the system may write
an Objective `Conclusion` that references the actual current Objective
assessments used by the gate. It must not synthesize Evidence, Criterion
assessments, acceptance decisions, or an independent reviewer. Use the existing
automation ActorKind with a stable WOS provider and identifier; record the
delegating Principal, credential, scope, policy revision, and evaluation time.
The automation authority is limited to applying this predicate and writing the
derived Objective conclusion.

Outcome conclusion is always explicit. Achieving every required Objective,
including through this policy, never calls `AchieveOutcome`, and never creates
an Outcome conclusion on behalf of its owner.

### Revisions, cycles, and transaction behavior

- Reopening an Objective increments a monotonic completion cycle. Assessments
  and conclusions from an earlier cycle remain in history and cannot satisfy
  the new cycle. Reaffirming proof requires a new authorized assessment with
  current criterion revision and cycle provenance; do not copy old assessments.
- Evaluate and conclude under the same Outcome guard and UnitOfWork as the
  mutation that makes the Objective eligible. Reuse a private achievement
  primitive shared with the manual command; do not recursively invoke a public
  handler or make an HTTP/MCP call from inside the transaction.
- Use expected versions and the current Outcome revision. A stale or incomplete
  snapshot cannot close the Objective. A composite command advances the Outcome
  revision once; each changed Objective advances its own version once.
- Derived candidates and any cascade are processed in stable order to a fixed
  point. The initial implementation limit is 64 derived achievements per
  transaction. If a complete cascade cannot fit the configured bound, do not
  report the remaining Objectives as achieved or partially eligible. Persist a
  durable, idempotent reconciliation obligation through the existing durable
  integration mechanism, or disable automatic mode for scopes that cannot be
  reconciled safely. Reconciliation only re-evaluates this rule; it does not
  select or execute work.
- An idempotent replay returns the original command result and derived
  conclusions. It must not repeat the evaluation as a new intent or create
  duplicate conclusions/events.
- A precondition failure for Objective eligibility does not undo an otherwise
  valid Task acceptance. Storage, corruption, or transaction failures roll
  back the complete mutation and its derived conclusions.

### Relationship to prior rules

For `operational_v2`, this decision supersedes the prior blanket prohibition on
Task-triggered Objective achievement in ADR-001 and the older manual-only
Objective completion text in the canonical architecture. It is consistent
with the updated `AGENTS.md` and the lifecycle/profile gates in ADR-043.
`compatibility_v1` retains its historical behavior until explicitly migrated.
This decision does not weaken the rule that Outcome acceptance is explicit,
nor the separation among Evidence, assessment, delivery, review, and
Conclusion.

## Consequences

- Objective eligibility becomes a canonical, complete read model instead of a
  frontend percentage or page-local calculation.
- Automatic completion is useful for teams that delegate deterministic
  closure, while human-final-decision policies remain explicit and supported.
- Existing data remains manual until an authorized, previewed migration opts
  in; deployment does not silently reinterpret old Objectives.
- Implementations need cycle-aware assessment applicability, durable
  provenance, transaction-level revalidation, and deterministic cascade tests.
- Outcome lifecycle and conclusion remain independently controlled.

## Rejected alternatives

- Reusing `CompletionPolicy` for who concludes: work-scope accounting and
  decision authority are different policy axes.
- Auto-achieving an empty Objective: there is no operational work whose
  completion justifies the derived conclusion.
- Inferring Objective assessments from accepted Tasks: child completion does
  not prove the Objective's own criteria.
- Treating cancelled or waived work as accepted for automation: exclusion
  requires an explicit human decision.
- Running an agent, LLM, or scheduler to decide or execute the remaining work:
  WOS records and enforces coordination state; external runtimes remain
  responsible for choosing and performing work.
- Automatically achieving an Outcome when its Objectives are achieved: the
  Outcome remains an explicit owner decision.

## Verification required before rollout

Cover empty and descendant-work Objectives, exact current Criterion revisions
and cycles, pending and cancelled work, owned Issues and duplicate chains,
hard dependencies, Blockers, open authority/review, incomplete/paginated scans,
unauthorized delegation, profile migration defaults, concurrent final
acceptances, concurrent Task creation, stale versions, cascade limits,
idempotent replay, and rollback on storage failure. Verify that no path
automatically concludes an Outcome and that existing manual policies retain
their authorized gates.
