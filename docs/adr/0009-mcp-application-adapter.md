# ADR-009 — MCP as an Application adapter

Status: accepted, 2026-10-07.

The pinned official MCP SDK exposes stdio and stateless Streamable HTTP. MCP tools call the same Application services as HTTP. The command catalog, HTTP schemas and Go client are generated from the same command signatures; explicit scope, version, idempotency key and claim/fencing arguments survive transport changes. Protocol/session state never stores business state. Supported client/profile details and limits are in docs/contracts.md and docs/dependencies.md.

Evidence: `packages/wos-api/mcp; internal/server/mcp_runtime_test.go; mcp_identity_test.go; release_matrix_test.go`.
