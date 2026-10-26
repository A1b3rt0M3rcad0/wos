# ADR-046: Authenticated Task delivery and verification

Status: accepted for implementation in Draft PR #116 by the owner's
2026-10-21 instruction. Qualification and rollout remain pending; this ADR
does not claim that the capabilities are implemented.

## Context

The WOS has two distinct execution paths. Legacy leases and `contracts_v1` can
record work, while `signed_contracts_v2` already has a cryptographically bound submission
and independent review protocol. A common Kanban may display an operational
Verification stage, but the stage must correspond to an actual persisted
delivery or signed review case. Merely moving a card, adding Evidence, or
opening a review screen cannot create that fact.

`WorkSubmission` requires a real Work Contract and a matching specification
digest. `ReviewCase` and `ReviewContract` bind the signed review protocol's
contract, submission digest, policy, execution principals, and signature
provenance. Reusing either type for an unsigned lease would invent authority or
weaken those guarantees. Conversely, direct completion under a legacy lease
does not currently provide a persisted review decision that an Objective's
operational policy could rely on.

## Decision

### A bounded unsigned verification model

For `compatibility_v1`, preserve existing completion semantics. For
`operational_v2`, require every accepted Task to have a persisted delivery and
authorized decision. Introduce a minimal `TaskDelivery` and immutable
`TaskDeliveryDecision` for non-signed protocols. These records describe work
that was performed elsewhere; they do not schedule or execute work, create a
new Task lifecycle, or replace the Task's existing persisted lifecycle.

`Verification` is a projection of a real pending unsigned `TaskDelivery` or a
real signed `ReviewCase`. It is not a newly persisted Task status. A correction
requested from unsigned review returns the Task to `todo` without a lease; the
executor must explicitly acquire it again. Cancelled and withdrawn work is not
accepted.

### Protocol-specific provenance

The server derives the authority kind from the authenticated request and
persisted Task authority. Clients cannot select or forge it.

| Authority kind | Required binding | Material source |
| --- | --- | --- |
| `legacy_lease` | Exact claim, current fencing token, resolved credential, Principal, ActorRef, Task version and definition digest | A delivery-owned immutable summary and versioned Artifact/Evidence references; no invented Contract ID or `WorkSubmission` |
| `work_contract_v1` | Exact Work Contract and execution/lease authority, credential and Principal | A reference to the real `WorkSubmission`; do not duplicate or rewrite its material |
| `signed_contracts_v2` | Existing signed acquisition, signed submission and review bindings | The existing `ReviewCase`, `ReviewContract`, signed material, decisions and receipts; no native `TaskDelivery` |

For unsigned submissions, credential and actor provenance comes from the trusted
authentication/authority resolver, not from payload fields. A TaskDelivery
captures the ended execution's exact claim/fence or contract/execution,
credential and Principal, Task version, definition digest, applicable
Criterion IDs/revisions, and delivery round. A Work Contract reference is
present only for `work_contract_v1`; a WorkSubmission reference is required
only for that protocol. A Task governed by `signed_contracts_v2` is rejected by
unsigned delivery and decision commands.

`TaskDelivery` has a stable ID, scope, WorkItem ID, version, authority kind,
protocol binding, provenance, immutable material reference or snapshot,
criteria snapshot, round, superseded-delivery reference, server timestamps,
and current status. Status is limited to `pending`, `approved`,
`changes_requested`, `withdrawn`, and `superseded`. Each decision is a separate
append-only `TaskDeliveryDecision` with the delivery ID, decision kind, reviewer
identity, reason/findings, applicable assessment references, expected versions,
and server timestamp. A later correction creates a new delivery and decision
history; it never overwrites prior material or findings.

### Submission and authority handoff

`SubmitTaskForReview` is one authorized, versioned, idempotent Application
operation. Within one UnitOfWork under the Outcome guard it revalidates the
current Task, Objective/profile gates, live lease or contract, fencing token,
credential ownership, material references, current definition/criteria, and
absence of another pending review. It persists the delivery, ends the execution
authority, binds the pending delivery to the Task, advances versions, writes
domain/integration events, and records the exact receipt atomically.

For `contracts_v1`, compose private validated WorkSubmission and delivery-close
primitives in this operation. Do not call public `SubmitWorkResult` and then a
second command, leaving an intermediate state. Do not modify signed delivery
methods to accept unsigned envelopes. A failed transaction leaves neither an
unlinked delivery nor a released authority without its matching delivery; a
lost response is recovered with the same idempotency intent.

Add a `PendingDeliveryID` only for the unsigned path. It is mutually exclusive
with `PendingReviewCaseID`. The otherwise-invalid state of an `in_progress`
Task without a live lease is allowed only when its pending delivery is present,
valid, same-scope, and pending; it is not a general relaxation of the lease
invariant. The application and storage contracts both validate this binding.

### Authorized decision and correction

`AcceptTaskDelivery` and `RequestTaskDeliveryChanges` require the exact delivery
ID plus current Task and delivery versions. The reviewer must be authorized by
the current policy. Where independent review is required, a reviewer Principal
must differ from every Principal in the execution/correction history;
different API keys for the same Principal do not establish independence.
Acceptance validates the exact current delivery, applicable current
assessments linked to the relevant Criterion revisions and delivery/cycle,
policy, dependencies, Blockers and other domain gates. It does not require the
executor's ended lease; reviewer authority is a separate act.

Approval appends a decision, marks the delivery approved, writes the Task's
Conclusion and `done` lifecycle, and clears `PendingDeliveryID` atomically.
The Conclusion references the actual material and current assessment records;
it does not synthesize Evidence or `met` assessments. Only one decision may win
for a delivery version. An exact idempotent replay returns that decision;
a conflicting second decision fails.

A correction decision requires a reason and actionable findings. It records
`changes_requested`, clears the pending pointer, and returns the Task to `todo`
without acquiring a new lease. The next submission has a higher round and
explicitly supersedes the prior delivery. Prior assessments do not automatically
apply to the new delivery. Implementation fixes preserve the sealed definition;
scope/criterion changes use the existing authorized Backlog replanning path.
Withdrawal is an explicit authorized operation with a reason and history.

For the unsigned path, an explicit, separately authorized reviewer action may
decide on their own Principal's work only when the effective policy does not
require independence. The `signed_contracts_v2` protocol remains at least as
restrictive as its frozen `independent_review` policy and does not gain an
unsigned bypass.

### Signed protocol remains authoritative

Never create `TaskDelivery` for `signed_contracts_v2`. Derive its verification
view from the real signed `ReviewCase` and existing signed records. Preserve DSSE bytes,
digests, frozen specification and acceptance policy, signing keys, original
credential, execution/fencing bindings, reviewer separation, receipts, and
correction history. A signed direct return or completion cannot satisfy a
profile that requires review. The browser may show status and prepare an
authorized-host handoff; it cannot sign or acquire signed authority.

Do not relax `WorkSubmission.Validate`, the signed contract's envelope or
`WorkContract.Deliver` requirements, or `ReviewCase.Validate` to fit the
unsigned path. An unsigned claim must not be assigned a fake Contract ID,
submission digest, signature, or review case.

### Read models, authorization, and integration

Expose one `TaskVerificationView` that identifies the actual source kind and
record ID, round, material, executor, reviewer, status, current requirements,
blocked reasons, and available authorized actions. For signed work, fields are
projected from signed records; for unsigned work, from TaskDelivery. The view
must not infer Verification from a stale pointer, Evidence presence, lifecycle
alone, page counts, or a UI drag gesture. It is additive and does not merge the
underlying records.

HTTP, MCP, SDK, CLI, and browser operations adapt the same Application use
cases. Each write validates authenticated scope, permission, expected versions,
idempotency, Outcome coordination, and exact protocol provenance. Events and
receipts distinguish submitted, approved, correction requested, and withdrawn
operations. No transport gets a status setter or a path around canonical
authorization.

## Consequences

- Unsigned work receives a real persisted Verification stage without pretending
  that it used the cryptographic signed-review protocol.
- A delivered Task no longer retains execution authority while awaiting review.
- Evidence, Artifact, CriterionAssessment, TaskDelivery, signed WorkSubmission,
  review decision, and Conclusion remain separate records with separate
  semantics.
- Storage requires one-open-delivery constraints, scoped links, append-only
  decision history, restart/restore coverage, and equivalent SQLite and
  PostgreSQL behavior.
- Browser, HTTP, MCP, SDK, and CLI can present one operational view while
  preserving each protocol's actual source of authority.

## Rejected alternatives

- Treat drag-and-drop or a visual lane change as a delivery or review: display
  state is not a domain fact.
- Reuse `WorkSubmission` for a legacy lease: it requires an actual contract and
  contract specification digest.
- Reuse or weaken `ReviewCase` for unsigned review: it represents the signed
  review protocol and carries signature, digest, policy and authority
  invariants.
- Persist a second Task lifecycle or generic review status: Verification is a
  projection from a pending real delivery/review record.
- Accept a Task when Evidence or an Artifact is merely present: proof must be
  evaluated under the applicable Criterion and delivery policy.
- Preserve the execution lease while another participant reviews: delivery
  ends the executor's authority; review uses separate authorization.
- Auto-reacquire after corrections: another execution is a new explicit claim
  with fresh readiness, authorization and fencing checks.

## Verification required before rollout

Test legacy and contracts_v1 deliveries separately; reject signed authority on
unsigned commands; preserve signed fixtures/receipts; validate credentials,
claims, contracts and fencing; reject stale definition/criterion snapshots;
guarantee one pending delivery and mutual exclusion with signed review; exercise
atomic rollback, timeout/replay, restart and restore; race competing decisions;
test review independence by Principal rather than key; prove correction rounds
do not inherit assessments or acquire leases; verify no direct completion or
administrative route bypasses required review; and confirm HTTP/MCP/SDK/CLI
parity with storage-contract tests for SQLite and PostgreSQL.
