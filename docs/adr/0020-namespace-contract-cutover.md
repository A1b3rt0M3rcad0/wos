# ADR-020 — Explicit Namespace contract cutover

Accepted, 2026-10-08.

The public `set_namespace_work_protocol` command changes one Namespace through `legacy` → `draining` → `contracts_v1`. It carries the Scope of an existing anchor Outcome so it retains the existing command identity, authorization, idempotent receipt and Outcome event model. Every affected Outcome receives one coordination revision and an `outcome.work_protocol_changed` fact in the same transaction. No historical WorkContract or execution specification is fabricated. Work IDs, content versions, criteria, conclusions and fencing counters survive cutover.

New command transactions take a shared Namespace guard before authorization/idempotency and Outcome guards. Protocol commands take the exclusive guard first, then Outcome guards in deterministic ID order. PostgreSQL uses row locks; SQLite serializes writers. Receipt replay precedes phase rejection, preserving old completed commands. Draining prevents fresh legacy claim/reclaim but permits finishing or releasing existing legacy leases. Activation requires no valid legacy lease and explicit operator acknowledgement that old writers have been stopped. New WorkItems then use contracts automatically. Administrators can update versioned lease policy; renewal uses current policy without altering frozen execution material.

Old binaries do not understand the new guards. Deployments MUST stop old processes, drain requests and withdraw their database access before acknowledging `writers_drained`. The acknowledgement is an operational precondition, not automatic discovery or technical fencing of arbitrary old database clients. Running pre-contract binaries against a migrated database is unsupported. Backup restoration must include protocol metadata and receipts; silently reverting the phase is prohibited.

`reconcile_expired_work_contracts` is explicit, paginated housekeeping (1–100 per scan), not an agent runtime or scheduler. It records expiry, detaches authority while preserving recoverable work, and emits facts atomically. Administrative cancellation is permitted only after a contract is revoked or reconciled. Agents never gain a holder-release command.

Tradeoffs: Namespace cutover locks all its Outcomes briefly and requires an existing anchor Outcome. It favors a consistent migration over online mixed writer support. Large deployments should schedule a maintenance window and measure migration duration with their dataset. Actual Windows and independent adapter/platform gates remain separate from protocol migration.
