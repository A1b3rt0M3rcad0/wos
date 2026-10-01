# ADR-010 — Public Core and internal Server boundary

- Status: Accepted
- Date: 2026-10-01

## Context

Go packages under `internal/` cannot be imported by external modules. WOS requires a real embedded mode in addition to the standalone server.

## Decision

Public domain/application/port contracts live under `core/`. Public storage adapters will live under `storage/`. Server composition and protocol transports remain internal implementation details.

Initial dependency direction is:

```text
transport -> application -> domain
                    |
                    v
                  ports <- adapters
```

## Consequences

The public Core is an API surface with compatibility obligations. Server and transport changes can evolve without forcing consumers to depend on their implementation packages.
