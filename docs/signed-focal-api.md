# Signed v2 focal API

This API reads persisted state. It does not run agents, certify quality, select a reviewer or grant execution authority. Namespace activation remains an explicit operation; configuring an issuer alone never activates v2.

## Public identity and enrollment

HTTP `GET /api/v1/security/signing-identity` and MCP `wos_get_signing_identity` return the authenticated credential's public identity, Namespace CAS version, current policy and at most 100 public key records. `keys_truncated` explicitly reports a bounded list. An optional `enrollment_id` expands only a challenge bound to the authenticated Principal and credential. Private signing material and bearer tokens are absent.

HTTP `POST /api/v1/security/signing-commands`, MCP `wos_manage_signing_identity` and SDK `SigningMutation` use the same transactional SecurityService. HTTP accepts the command itself and the `Idempotency-Key` header; MCP accepts `{idempotency_key, command}`. New versions use decimal strings, including zero for an unused CAS field. The caller cannot choose another authenticated identity.

Operations are `set_credential_policy`, `set_acceptance_policy`, `set_review_group`, `create_enrollment`, `create_rotation_enrollment`, `register_signing_key` and `revoke_signing_key`. Administrator authorization issues a short, one-use enrollment. Registration verifies an Ed25519 possession proof over that exact challenge. Rotation requires the previous authorized key or explicit administrative recovery. A token alone is insufficient to replace a signing key.

## Trust

HTTP `GET /api/v1/namespaces/{namespace_id}/signed-trust`, MCP `wos_get_signed_trust` and SDK `GetSignedTrust` return the persistent public server identity and Namespace protocol metadata. Consumers must establish and persist trust during explicit onboarding. Public metadata does not authorize sending secrets to an origin taken from an unknown repository.

## Scoped resources

HTTP GET paths are:

- `/api/v1/namespaces/{namespace_id}/outcomes/{outcome_id}/signed-state/{resource}`
- `/api/v1/namespaces/{namespace_id}/outcomes/{outcome_id}/signed-state/{resource}/{entity_id}`

MCP `wos_read_signed_state` and SDK `ReadSignedState` accept `scope`, `resource`, optional `id`, `contract_kind`, expected `digest`, `idempotency_key`, `limit` and `cursor`. MCP also exposes `wos://namespaces/{namespace_id}/outcomes/{outcome_id}/signed-state/{resource}/{entity_id}` resource templates, with optional digest/contract_kind query parameters. Collections use the tool for pagination.

| Resource | Identifier | Result |
| --- | --- | --- |
| execution | Execution contract ID | Compact IDs, versions, fence, persisted/effective status, expiry and proof pointers. |
| review | Review contract ID | Compact independent review authority state and case pointer. |
| specification | Contract ID plus execution/review kind | Exact issuer envelope, verified digest and public signer record. |
| authority | Contract ID plus execution/review kind | Latest issued grant, separately from immutable specification. |
| case | ReviewCase ID | Persisted target, submission/spec digests, round, independence history and explicit disposition. |
| correction | Closed changes-requested ReviewCase ID | Accepted signed review decision and immutable target case. |
| fact | Signed fact ID | Exact stored verified envelope and historical public key. |
| submission | Accepted v2 submission ID | Canonical material as base64, exact submission digest and issuer acceptance pointer. |
| receipt | Idempotency key | Durable signed acceptance for the authenticated Principal in the exact requested Outcome. |
| cases | No ID | Bounded historical cases. |
| review_queue | No ID | Bounded open cases without a currently valid review lease. |

Metadata omits specification and material deliberately; pointers support focal expansion. Digests bind expansion to exact bytes. A retired key may verify historical facts but cannot authorize a new operation. Accepted material is separate from the original signed return: documentary local keys have been resolved into server identifiers. The issuer receipt binds that accepted material's digest. Querying a fact does not substitute a caller-provided key for the registry.

Collections scan at most `limit` records (1–100, default 25); `next_cursor`, `scanned` and `search_complete` describe the bounded scan. A review queue page can have fewer returned cases than scanned records because active leases are excluded. Follow the cursor even when such a page is empty. Neither effective expiry nor queue membership persists a transition or guarantees acquisition, independence or quota eligibility.

## Authorization and compatibility

Reads check the actual live bearer, grant and policy in the transaction at authoritative database time. Permitted-Outcome restrictions apply to old focal reads and SearchOutcomes too. Search filtering happens in storage before pagination; clients cannot widen it. A scope-policy change invalidates an old restricted search cursor. Namespace-wide state reads/command receipts are rejected for an Outcome-restricted credential; public identity/protocol metadata remains separately bounded.

New signed metadata counters are decimal strings. Signed envelope payloads remain exact opaque base64; metadata formatting never rewrites signed bytes. Historic v1 numeric encoding and fingerprints remain unchanged.

Open review freezes unsigned task/parent intent, criteria, dependencies and lifecycle operations. Active signed execution authority also freezes material edits. Unsigned assessments cannot rewrite a signed accepted Task obligation even after independent completion. Observational records remain writable under their existing permissions. Administrative intervention and a newly issued authority preserve history.

## Validation status

The cross-transport fixture uses real HTTP and MCP clients plus the Go SDK on Memory, SQLite and actual PostgreSQL. It authorizes enrollment via HTTP/SDK, registers possession proof via MCP, acquires via HTTP/SDK, returns the exact signed envelope via MCP and verifies the issuer receipt. A separate SDK reviewer completes after the original bearer is revoked; the original execution remains delivered and HTTP metadata shows Task completion. The harness owns ephemeral test keys; no LLM receives them. This is protocol acceptance, not a certified external-agent integration or a context/cost benchmark.

During repository validation, the accumulated PostgreSQL test catalog exceeded the development container's former 64 MiB `/dev/shm` limit. The test container now uses 512 MiB with the same persistent volume; parallel scans/maintenance are disabled in that development container. This is environment configuration, not a production database change. SQLite `Options.StartupTimeout` now separately bounds connection setup/migrations (default 30 seconds); `BusyTimeout` still controls writer contention. An explicitly longer legacy BusyTimeout retains its former startup allowance unless StartupTimeout is supplied. A tiny explicit startup deadline still preserves cancellation; enlarging startup allowance does not enlarge the writer wait.
