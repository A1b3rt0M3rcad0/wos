# ADR-019 — API-only wosctl and bounded YAML workspaces

Status: accepted for implementation, 2026-10-08. Not an implementation claim.

Create packages/wos-cli and executable wosctl. The existing wos executable remains the server/admin entrypoint. Client dependencies point to public SDK/contracts, never storage. YAML is optional; HTTP/MCP consumers require no local files. WorkContract, checkpoint draft and result use schema_version 1, immutable spec separated from editable progress. Credentials are referenced through environment variables and never persisted in payloads, logs or journals.

Adopt the restricted YAML 1.2 profile in plan section 15: one UTF-8 mapping, known fields, string keys, no duplicate keys/aliases/anchors/merge keys/custom tags; 256 KiB, depth 16 and bounded nodes/scalars. Library version will be pinned and fixtures must prove profile behavior. SHA-256 of RFC8785 JCS over typed spec/submission objects supplies semantic integrity, not authentication. Decimal fencing strings avoid JavaScript precision loss.

Local write-ahead journal persists exact destination-bound intent and idempotency key before network mutation. Confirmed receipt is durable before completion marking. Recovery reuses exact intent, preserves local edits and does not release or acquire silently. Filesystem materialization is not atomic with remote acquisition; failure leaves recoverable authority. Per-contract cooperative locks and atomic per-file writes cannot substitute remote fencing. Reject path escapes and symlink/reparse traversal. Destination changes cannot redirect credentials or pending intents silently.

Keepalive is explicit, foreground, cancelable and host-supervised. Installing skills/packages does not start agents, daemons or configure MCP. JSON stdout and stable exit categories support agents. Linux amd64 and Windows amd64 require actual platform execution before support claims; cross-compilation is preparation only. Publication requires evidence separate from build. macOS and declarative GitOps remain later extensions.
