# Stable public compatibility contract

Published baseline: [v0.4.2](https://github.com/A1b3rt0M3rcad0/wos/releases/tag/v0.4.2), source b572269085494ddd4d6370716ff1cb360b46065f. The operational candidate is unreleased; its additive changes and old-client limits are reviewed in [the operational compatibility record](operational-contract-review.md). Product 1.0 is gated by the original plan and actual immutable-source acceptance; this document alone does not publish or certify that release.

| Identity | Supported compatibility |
| --- | --- |
| Product version | SemVer in VERSION, native executable metadata, npm wosctl, MCP serverInfo and OpenAPI info |
| HTTP | `/api/v1`; existing command names, DTO fields, entity identifiers, CAS/error/replay semantics retained |
| MCP | Stateless Application adapter; negotiated transport protocol is distinct from product SemVer |
| Go SDK/Core | Public module paths and the same Application services; wire precision and signed payload tests remain required |
| CLI | `wos <command>` is primary; wosctl remains an executable alias. The source central workflow uses installation-scoped defaults; `--workspace PATH` explicitly selects legacy operation |
| Workspace YAML | Schema 1/unsigned contracts retained where compatible; schema 2 profiles/signed contracts preferred for new signed deployments |
| Work protocol | Legacy and contracts_v1 are explicit compatibility modes; signed_contracts_v2 requires provisioning, enrollment and drained Namespace cutover |
| SQL | Existing checksummed migration sequence; newer-schema refusal and historical writer refusal remain required; rollback is restore to a separate database |
| Skills | Canonical bundle embedded in native wos; its release version follows the executable; discovery paths do not grant authorization |
| Platforms | Linux amd64 service; Linux/Windows amd64 native CLI; other service platforms are outside the certified scope |

The central default no longer discovers a project `.wos` directory. Migrate signed schema-2 material explicitly with drained writers, or retain `--workspace` for the original binding. Published 0.4.2 lacks the central commands and new activity reads; use matched source client/server as described in [profiles](../signed-workspace-profiles.md). Existing issued profile/contract bytes and exact pending requests are preserved.

No signed proof is fabricated for historical unsigned content. Cutover preserves existing history and original idempotent receipts, while rejecting incompatible old writes. A profile identifies server/issuer, Namespace, Principal, credential and protected key references; changing command spelling grants no new authority.

The API schemas and CLI contract/YAML JSON schemas are frozen in `public-contract.json`. The executable compatibility test normalizes only product info.version in generated OpenAPI; other wire/schema changes fail until reviewed and explicitly accepted. Golden signing/counter/YAML and HTTP/MCP/SDK tests cover semantic behavior beyond static shapes. New optional product features do not justify weakening CAS, scopes, fencing, independent review or replay protection.

Existing exit categories remain: 2 usage/validation, 3 authorization, 4 content/version conflict, 5 expired/stale authority, 6 transport/infrastructure, 7 unmet verification, 9 local material/recovery. JSON client output remains operation/committed/data/error/receipt information. Common human copy and command aliases do not rename wire identifiers or on-disk issued/protected schema fields.

Apache-2.0 remains the existing owner-selected license. It applies consistently to the public package, native archives and OCI labels. Historical service/skills npm source manifests are private compatibility fixtures. No Source-Available or proprietary claim replaces the actual license. Hosted managed instances and service sales are separate commercial operations; this stable plan makes no legal license change.

Operational qualification, actual released-binary refusal and the source-bound head gate are recorded in [operational acceptance](operational-acceptance.md). Historical reference manifests remain immutable; a new-schema database is never opened by an old writer as a rollback strategy.
