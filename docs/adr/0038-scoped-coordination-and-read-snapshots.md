# ADR 038 — Scoped signed coordination and coherent read snapshots

Status: accepted implementation decision; capacity qualification remains separate.

## Context

Signed commands previously acquired the Namespace protocol row exclusively,
including ordinary commands on independent Outcomes. Continuity and entity
queries opened the same Serializable write UnitOfWork as commands and acquired
the Outcome write guard. These choices retained correctness but unnecessarily
serialized work and made observational traffic compete with writers.

The canonical specification sections 16 and 18 require a coherent revision and
state snapshot. Immediate credential revocation and protocol/security cutover
ordering are also required; replacing the write transaction with unrelated
queries or removing security barriers would weaken those guarantees.

## Decision

Official adapters implement optional `ReadSnapshotManager` and
`SignedQuotaUnitOfWork` ports. Embedded third-party adapters retain the previous
guarded transaction path until they implement the corresponding port.

Read queries use one snapshot for state, revision and the fixed evaluation time.
Their coordination lookup is a plain SELECT: it neither creates a Namespace or
coordination row nor locks the Outcome for mutation. Read snapshots cannot
commit; application queries close them by rollback. PostgreSQL uses Repeatable
Read and retains shared Namespace and credential row barriers for authorization.
These barriers order security mutations against reads and reject stale security
snapshots after concurrent revocation. PostgreSQL cannot execute `FOR SHARE` in
a SQL read-only transaction, so this is an explicit exception to the canonical
specification's database-level `READ ONLY`: the transaction is observational and
cannot commit, while shared security row locks remain permitted. It is not an
unprotected read-replica authorization path.

Security read failures grant no authority. A missing or inactive credential or
grant remains forbidden; a database serialization/deadlock conflict retains its
transaction-conflict classification. Disguising that transient database failure
as credential revocation would prevent exact-intent reconciliation and retries.
This applies to transactional signed-policy reads during work, enrollment and
rotation. Only an absent or disallowed policy is an authorization denial.
SQL failures raised while scanning or iterating nested state, participants,
candidate work or issuer history retain the same adapter error classification
as failures raised when issuing the query; a deferred database error must not
escape as an unclassified server error.

Signing identity and signing-operation diagnostics use the same snapshot path.
An optional Namespace-version reader obtains the diagnostic CAS marker under a
shared barrier; polling CLI identity/status must not upgrade it to an exclusive
security mutation lock. Credential, parent-session and policy authorization is
still checked inside that coherent transaction.

SQLite snapshots use deferred, query-only transactions in a separate bounded
four-connection read pool. The existing single immediate-writer connection is
retained. WAL permits the snapshot and a writer to coexist; a read linearizes at
its database snapshot, and a new read after revocation sees the revoked state.
SQLite still has one writer per database, including across processes.

Ordinary signed commands take a shared Namespace protocol/security barrier.
Protocol changes and security administration retain their exclusive barrier.
Signed acquisition takes quota coordination before Outcome coordination: an
optional Namespace lane for an enabled Namespace ceiling, then a Principal lane
covering all its credentials. Work and review lanes are distinct. PostgreSQL
implements these lanes with transaction-scoped advisory locks over stable
hashed identities; a hash collision can only cause additional serialization.
SQLite's immediate transaction already excludes competing writers. Existing
Serializable command isolation remains unchanged. A snapshot that becomes stale
while waiting can still return the existing transaction-conflict error; callers
must repeat the same intent and idempotency key. No automatic retry changes the
caller intent, expected version, authority or fencing token.

Quota counts use indexed Principal/Credential predicates and request Namespace
counts only if its ceiling is enabled. They retain expiry evaluation at the
authoritative transaction time. No asynchronous cleanup or cached counter decides
whether authority is available. Renewals, revocation, independent review,
durable signed receipts and exact-intent replay retain their existing contracts.

Negative signed-return closure deliberately does not advance the Outcome revision
(ADR 034). Two Serializable contenders can consequently retain a snapshot from
before the winner even after waiting for the same Outcome row guard. The terminal
table's original-intent uniqueness still decides the winner atomically; Application
maps a competing immutable insertion to `idempotency_conflict`, matching the
already-visible-winner path, and rolls back its entire losing transaction and
signed receipt. This is not an automatic retry or a relaxation of acceptance
exclusion. Exact resolution retries retain their durable idempotent response.

## Boundaries and consequences

- Outcomes remain the domain mutation coordination boundary. One hot Outcome
  remains serialized; one configured shared Namespace quota is also a genuine
  shared resource.
- Existing configured authority ceilings retain their validated 1–100 bounds;
  absence of a Namespace acceptance policy has no Namespace-wide ceiling and
  retains the default Principal ceilings. This ADR makes no unlimited-capacity
  or throughput claim.
- Independent Principal acquisitions without a shared Namespace ceiling can
  take independent quota lanes, while commands still observe immediate security
  changes and the current signed protocol.
- PostgreSQL shared security locks intentionally make security administration
  wait for already-authorized queries. Query deadlines remain necessary.
- Historical capacity fixtures are not new qualification of this implementation.
  Benchmark and failure acceptance must report workload and source revision.

## Verification

The shared SQLite/PostgreSQL read-snapshot contract holds a snapshot while a
writer commits, verifies unchanged state and revision in that snapshot, then
verifies a fresh query sees the new revision. Missing reads cannot create state,
and read snapshots reject commit. PostgreSQL tests exercise independent Principal
quota lanes, exclusion within one Principal lane, and revocation ordering against
authorized snapshots. An actual signed acquisition on another Outcome and a CLI
identity diagnostic must complete while an independent Principal/Outcome and
the shared Namespace security barrier remain held. Existing signed quota,
cutover, revocation, lease expiry,
cursor/projection and exact-intent suites remain required regression gates.
Fault tests exercise policy conflicts and a real PostgreSQL `40001` both at a
Principal-group lookup and after a successful first nested-state row. These
verify the transactional error code and retained cause independently of a lucky
load-test schedule.
