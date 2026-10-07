# ADR-008 — Namespace isolation and consumer context

Status: accepted, 2026-10-07.

Namespace grants authorize access. ExternalContext and external references address consumer state and are indexed with their JSON types; neither grants permission. ExecutionContext records request/run correlation. Principal is authenticated independently of the payload; ActorRef records authorized authorship. See ADR-014 and ADR-015 for remote enforcement and administrative receipts.

Evidence: `application/authorization.go; application/security.go; storage/*/context_contract_test.go`.
