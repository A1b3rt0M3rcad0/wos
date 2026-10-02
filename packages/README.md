# WOS product packages

`packages/` contains two product boundaries.

| Package | Responsibility | Deployment role |
| --- | --- | --- |
| `wos-core` | Domain, Application services, Ports, storage adapters and migrations | Public embeddable Go library; not a standalone process |
| `wos-api` | HTTP/MCP transports, authentication adapters, protocol contracts, standalone runtime/configuration and the `wos` binary | Executable/API distribution of WOS |

## Dependency rule

```text
wos-api ──> wos-core
```

`wos-core` is the reusable engine. It must never import `wos-api`.

`wos-api` owns every process/protocol concern required to expose Core as a standalone WOS service. Internal separation remains explicit:

```text
wos-api/
├── authentication/
├── http/
├── mcp/
├── contracts/
├── openapi.yaml
├── cmd/
│   └── wos/
└── internal/
    └── server/
```

HTTP, MCP and server bootstrap are internal architectural concerns of the API product package, not separate release units. Protocol handlers call the same Core Application services and never own business rules or write storage directly.

Storage adapters remain inside `wos-core` because embedded consumers can use them without depending on the standalone API/runtime.
