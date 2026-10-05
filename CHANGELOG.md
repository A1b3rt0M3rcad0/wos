# Changelog

## Unreleased — completion audit 2026-10-05

Completion branch, based on planning PR #13; not a released 0.1.

- Authorized per-request HTTP/MCP identity, Namespace grants, delegated authorship, browser sessions and revocation; transactional access revalidation and idempotent Namespace administration/audit.
- Bounded discovery, continuity, cursors, timeline, graph and focal work context; typed indexed ExternalContext and unique external references.
- Official MCP SDK with stdio/Streamable HTTP, shared generated command catalog, thin Go SDK and official web client.
- Durable declarative Trigger signals, atomic outbox, delivery leases/fencing, retries and signed webhooks.
- PostgreSQL pgx adapter, shared generated contracts and real PostgreSQL 18.6 contract/race execution in CI, including runtime transport/session scenarios and a clean integration restore.
- Independent reviewer policy, explicit structural reopen semantics, Apache 2.0, operation/contract documentation, Docker/Compose and metadata observations.


- Publish the complete v1 catalogue of 81 public integration fact types and its mapping check.
- Execute a public embedded Core host from an independent Go module.
- Expand HTTP/MCP/SDK/session scenarios to isolated PostgreSQL schemas and test clean-file/database restore of proof, planning and leases.

Open release obligations and actual evidence remain in `docs/auditoria-conclusao-2026-10-05.md`; partial browser/storage acceptance does not certify the full product release.
