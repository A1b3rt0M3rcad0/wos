# ADR-005 — Relational state with an immutable event log

Status: accepted, 2026-10-07.

Current relational state, Domain Events, idempotency receipts and configured integration signals commit in one transaction. The event log supports audit and ordered continuity; it is not a promise of full event-sourced replay or arbitrary historical state reconstruction.

Evidence: `application/pipeline.go; storage/*/events.go; compound rollback and durable integration contracts`.
