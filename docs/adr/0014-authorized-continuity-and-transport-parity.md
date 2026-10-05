# ADR-014 — Authorized continuity and transport parity

Status: accepted, 2026-10-05.

Remote entry points use `NewAuthorizedService`. Identity is resolved per request and
all Application reads and mutations check current grants. Authorization precedes
idempotency reservation and replay. A revoked credential cannot expose an old result.
The trusted embedded `NewService` contract remains explicit and separate.

Credentials are opaque random secrets stored only as SHA-256 digests. Each credential
binds a Principal, Actor and Namespace; a Principal grant in another Namespace does
not extend that credential. Namespace context alone never authorizes access.

Continuity schema 1 bounds each section and the total response, declares counts and
omissions, and supports incremental cursors. Section and graph cursors bind to an
Outcome revision and are rejected after changes. Evaluation time is pinned while
expanding a snapshot, so lease/time projections do not silently mix instants.
Discovery uses stable creation-time/ID keysets; timeline uses immutable revision/index
keysets. These two collections are live, not materialized snapshot pagination.

HTTP and MCP call the same Application commands. MCP uses the official Go SDK,
stdio for a trusted local composition and Streamable HTTP with per-request identity.
Command DTO generation only adapts names and duration units, preserving existing
command fingerprints. Query payloads are limited to 256 KiB. Oversized successful
mutations return a commit receipt instead of misreporting a failure after commit.

PostgreSQL uses a genuine pgx-backed transaction manager, SERIALIZABLE transactions,
Outcome row locks, shared domain rules and explicitly generated SQL variants.
Generated repositories and contract tests are checked for drift in CI. Compilation
or SQLite success does not establish PostgreSQL parity.

Roadmap active slots retain their existing serialized replacement semantics. Two
authorized concurrent activations may both succeed and the later activation replaces
the earlier slot. Both facts remain in history. Slot compare-and-swap is not required
for this contract; aggregate/draft optimistic versions remain mandatory.

Terminal Outcomes retain the existing structural policy: reopen the Outcome explicitly
before reopening an Objective or changing required structural/criterion obligations.
Rejected commands leave the conclusion intact. Later assessments and evidence
retractions remain admissible and may contest a terminal conclusion. Structural
contestation projections also inspect obligation snapshots defensively, including
legacy/restored states; they do not implicitly relax the terminal mutation policy.
