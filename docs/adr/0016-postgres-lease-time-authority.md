# ADR-016 — PostgreSQL lease time authority

Status: accepted, 2026-10-07.

Authorized services use the optional UnitOfWork TransactionClock port when provided. PostgreSQL returns clock_timestamp() after the command has acquired the Outcome guard, so replica clock skew and time waiting for locks cannot arbitrate lease expiry. Claim, renew, reclaim, release, completion/cancellation and operational queries share this authority. SQLite and memory use the deployment Clock. Trusted embedded NewService continues to honor its explicitly injected Clock for compatibility and deterministic tests; a distributed embedded host must provide a common authority or compose NewAuthorizedService. Local process time remains suitable for duration instrumentation. Section cursors pin the snapshot evaluation instant.

Evidence: `ports/transaction_clock.go; storage/postgres/clock.go; TestRemoteLeasesUseDatabaseAuthorityDespiteReplicaClockSkew`.
