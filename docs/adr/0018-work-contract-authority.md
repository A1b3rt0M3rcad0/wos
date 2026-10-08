# ADR-018 — Durable execution contracts and explicit cutover

Status: accepted for implementation, 2026-10-08. Delivery remains gated by C01–C12.

The owner authorized the attached [implementation plan](../work-contract-implementation-plan.md). It supersedes the legacy voluntary-release semantics only after explicit Namespace cutover. WOS remains a state/coordination service; consumers execute and supervise agents independently.

A WorkContract is a durable acquisition of an existing WorkItem. Its immutable specification snapshot is acquired in the same transaction as authority. One contract may be effectively valid per task. The contract incorporates the lease; CurrentLease is a compatibility projection, never another writable authority. Terminal causes are exclusively expiration, administrative revocation with reason, and accepted finalization. Progress and immutable submissions do not close authority. Terminal contracts never reactivate.

Contract content and lease have distinct CAS versions. Execution takeover by the same authenticated Principal explicitly increments the task fencing high-water mark and changes execution_id without implicitly extending time. Fencing is exact uint64, decimal strings in new wire contracts, with overflow rejection. Time is evaluated after Outcome guards using database authority on PostgreSQL. GETs project effective expiry without writes.

Finalization atomically records an exact submission, compatible criterion assessments, Conclusion, WorkItem done, contract completed, ordered events and an idempotent receipt. Old assessment material cannot silently accept a different submission. Objective/Outcome achievement remains separate. Replays require current authorization before returning protected historical results.

Migration is per Namespace: legacy → draining → contracts_v1. Draining blocks new claims and renewals, permits still-valid legacy completion, and cutover requires no valid legacy leases. No fabricated historical snapshots. Existing confirmed fingerprints/receipts remain intact. After cutover old execution commands fail explicitly and old servers must not be deployed. Revocation identifies a specific contract, never just a task.

## Required bypass inventory

- Claim/reclaim/renew/release/complete and privileged completion: reject incompatible legacy intents after cutover, including direct Application calls.
- WorkItem edits, criteria definition/revision/retirement and dependency changes: refuse material mutation under effective authority.
- Cancel WorkItem/Objective and achieve/abandon/archive Outcome: refuse affected active authority; administrative contract revocation is explicit beforehand.
- Assessment, documentary records and blockers remain authorized live facts, not silent specification changes.
- Core direct WorkItem methods must honor contract ownership markers; transports cannot be the only guard.

The historical architecture sections 7.1/7.3/7.4 describe legacy protocol. This ADR and execution plan govern migrated scopes. Additive persistence must preserve proof, versions and unsigned fencing across Memory/SQLite/PostgreSQL, restart and backup/restore.
