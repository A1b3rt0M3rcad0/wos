# ADR-007 — Aggregate versions and Outcome coordination

Status: accepted, 2026-10-07.

Mutable aggregates require their observed version. Cross-aggregate writes acquire the Outcome coordination guard. SQLite obtains its writer transaction immediately; PostgreSQL uses SERIALIZABLE transactions and a row lock. PostgreSQL serialization failures are explicit transaction_conflict errors. A caller may retry the same intent/key, but WOS never changes expected_version. Writes within one Outcome serialize; independent Outcomes remain separate coordination scopes.

Evidence: `application/pipeline.go; storage/*/repositories.go; tests/boundary/dependency_concurrency_test.go; release_concurrency_test.go`.
