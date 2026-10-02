# ADR-011 — Product package topology

- Status: Accepted
- Date: 2026-10-02
- Supersedes: ADR-010 for repository/package placement

## Context

WOS has two real distribution concerns: an embeddable Go engine and an executable/API surface that exposes that engine over supported protocols.

A separate `wos-server` product package would split process bootstrap from transports even though both belong to the same deployed WOS API artifact. That creates an extra release boundary without an independent product or deployment unit.

The architecture must preserve embedded use, keep protocol/process code out of Core, and avoid turning bootstrap into a second business layer.

## Decision

Product code lives under `packages/` in two explicit boundaries:

- `packages/wos-core` — public embeddable library containing Domain, Application, Ports, storage adapters and migrations.
- `packages/wos-api` — executable/API package containing HTTP/MCP transports, authentication adapters, protocol contracts, OpenAPI/JSON Schema artifacts, runtime bootstrap, process configuration and the `wos` binary.

Dependency direction is:

```text
wos-api ──> wos-core
```

`wos-core` must not import `wos-api`. HTTP and MCP remain adapters over the same Core Application services and do not write storage directly.

Inside `wos-api`, protocol and process concerns remain separated by subpackages such as `http/`, `authentication/`, `cmd/wos/` and `internal/server/`. This internal separation does not create additional product packages.

The Go module remains `github.com/A1b3rt0M3rcad0/wos`; this changes package import paths, not module identity.

## Consequences

External embedded consumers import Core through `github.com/A1b3rt0M3rcad0/wos/packages/wos-core/...`.

The standalone binary is built from `./packages/wos-api/cmd/wos`.

Storage remains part of the public Core boundary.

HTTP, future MCP, configuration, health endpoints and process lifecycle belong to the API distribution.

Repository boundary tests and CI enforce the two-package topology.
