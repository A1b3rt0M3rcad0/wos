# Changelog

## Unreleased — completion audit 2026-10-05

Completion branch, based on planning PR #13; not a released 0.1.

- Authorized per-request HTTP/MCP identity, Namespace grants, delegated authorship, browser sessions and revocation; transactional access revalidation and idempotent Namespace administration/audit.
- Bounded discovery, continuity, cursors, timeline, graph and focal work context; typed indexed ExternalContext and unique external references.
- Official MCP SDK with stdio/Streamable HTTP, shared generated command catalog, thin Go SDK and official web client.
- Durable declarative Trigger signals, atomic outbox, delivery leases/fencing, retries and signed webhooks.
- PostgreSQL pgx adapter, shared generated contracts and real PostgreSQL CI service (acceptance not yet obtained).
- Independent reviewer policy, explicit structural reopen semantics, Apache 2.0, operation/contract documentation, Docker/Compose and metadata observations.

Open release obligations and actual evidence remain in `docs/auditoria-conclusao-2026-10-05.md`; browser/product and PostgreSQL acceptance are not inferred from SQLite tests.
