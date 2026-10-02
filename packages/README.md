# WOS product packages

`packages/` contains the three product boundaries of WOS.

| Package | Responsibility | Deployment role |
| --- | --- | --- |
| `wos-core` | Domain, Application services, Ports, storage adapters and migrations | Embeddable Go library; not a standalone process |
| `wos-api` | HTTP/MCP transports, API authentication adapters and protocol contracts | Reusable protocol layer; composed by Server or another host |
| `wos-server` | Standalone runtime, configuration, process bootstrap and `wos` binary | Independently runnable WOS server |

## Dependency rules

```text
wos-server ──> wos-api ──> wos-core
     └────────────────────> wos-core
```

- `wos-core` must never import `wos-api` or `wos-server`.
- `wos-api` may depend on `wos-core`, but must not depend on `wos-server`.
- `wos-server` is the composition root and may depend on both.
- Protocol transports do not own business rules and do not write storage directly.
- Storage adapters remain inside the Core release boundary because embedded consumers need them directly.
- Repository-wide tests, documentation and development tooling remain outside `packages/`.

This organization mirrors Woobe's product-package approach while preserving WOS's Go-specific embedded mode.
