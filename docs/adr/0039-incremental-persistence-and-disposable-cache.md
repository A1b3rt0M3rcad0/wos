# ADR 039 — Incremental owned-state persistence and disposable cache maintenance

Status: accepted implementation decision; final integration acceptance pending.

WOS keeps relational current state, immutable historical facts and a transactional
outbox. It does not become full Event Sourcing. API replication must not multiply
SQL work proportional to unchanged proof history.

Each SQL UnitOfWork now remembers a private snapshot of the aggregate-owned state
that it loaded or successfully saved. Immutable criterion revisions, assessments
and ordinal conclusion records are represented by digests detached from caller
slices. A subsequent save rejects changes to those facts and skips their SQL when
unchanged. Current criterion pointers, conclusion pointers and actor links are
written only when changed. New historical facts retain the existing insert,
foreign-key and immutable-content verification. A save without a prior snapshot
uses the existing full verification path. No snapshot survives the UnitOfWork or
acts as an authorization, replay, or cross-request cache.

Single-aggregate historical reads fetch assessment evidence and conclusion
assessment references in scoped set queries, avoiding one child query per
historical record. Full public aggregates still contain and validate their full
history: payload, hashing and domain-validation work remain proportional to the
requested history. This decision does not claim bounded memory or constant-time
full aggregate reads. Bounded focal projections remain the operational read path
for large Outcomes.

Signed acquisition can request only Principal and Credential activity when no
Namespace ceiling applies. Each count uses the existing scoped indexes and the
same transaction and evaluation time as authorization. Namespace diagnostics and
configured Namespace limits retain exact totals, with additional Namespace quota
indexes. Expiry remains authoritative without a maintenance write. Counters,
cache entries, or worker progress never grant authority.

`Store.PruneExpiredIdempotency(ctx, limit)` removes a bounded number (1..10000) of
completed expired rows from `idempotency_records`. PostgreSQL uses database time
and `FOR UPDATE SKIP LOCKED` in a single statement; SQLite uses a short writer
transaction. Multiple workers may safely clean disjoint rows without a leader.
The operation never deletes active/unfinished reservations, signed operation
results, signed protocol facts, terminal-intent resolutions, security receipts,
revocations, assessments, conclusions or integration deliveries. Durable signed
exact-intent replay is independent of the short cache and has no age-based TTL.

The new indexes are forward-only migration 0032. Original migration checksums and
historical signed bytes stay unchanged. No table partitioning, proof archival,
retention policy for audit history, hosted binary storage or cross-installation
Namespace transfer is introduced by this decision.

Verification includes growing immutable assessment histories, unchanged current
updates with a fixed SQL statement count, rejection of aliased in-place history
edits, new-assessment/current-pointer persistence, bounded cleanup preserving
live/pending rows and durable signed replay, plus the shared SQLite and real
PostgreSQL storage/concurrency contracts. Runtime scheduling and operator-visible
limits are documented with the process-role deployment decision.
