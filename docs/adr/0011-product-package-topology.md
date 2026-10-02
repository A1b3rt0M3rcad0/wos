# ADR-011 — Product package topology

- Status: Accepted
- Date: 2026-10-02
- Supersedes: ADR-010 for repository/package placement

## Context

WOS has three distinct distribution concerns: an embeddable Go Core, reusable protocol adapters/contracts, and a standalone server process. Keeping these concerns spread across root-level `core/`, `storage/`, `api/`, `internal/`, and `cmd/` obscures the product boundary and diverges from the package-oriented repository organization used by Woobe.

The refactor must preserve embedded use, keep protocol code out of the domain, and prevent the standalone server from becoming a second business layer.

## Decision

Product code lives under `packages/` in three explicit boundaries:

- `packages/wos-core` — public embeddable library containing Domain, Application, Ports, storage adapters and migrations.
- `packages/wos-api` — reusable protocol layer containing HTTP/MCP transports, protocol contracts, OpenAPI/JSON Schema artifacts and API-facing authentication adapters.
- `packages/wos-server` — standalone composition root, executable, runtime bootstrap, process configuration, observability and server-only integrations.

Dependency direction is:

```text
wos-server ──> wos-api ──> wos-core
     └────────────────────> wos-core
```

`wos-core` must not import `wos-api` or `wos-server`. `wos-api` must not import `wos-server`. Transports call the same Application services and do not write storage directly.

The Go module remains `github.com/A1b3rt0M3rcad0/wos`; this changes package import paths, not module identity.

## Consequences

External embedded consumers import Core through `github.com/A1b3rt0M3rcad0/wos/packages/wos-core/...`.

The standalone binary is built from `./packages/wos-server/cmd/wos`.

Storage remains part of the public Core release boundary instead of becoming an artificial standalone package or service.

HTTP and future MCP adapters have an explicit reusable package boundary independent from process bootstrap.

Repository boundary tests and CI enforce the new topology.
