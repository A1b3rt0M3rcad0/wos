# WOS HTTP — Wave 05

Wave 05 exposes the M1 domain through the standalone WOS process. The HTTP
transport is an adapter over the existing Application services; it does not
duplicate lifecycle, criteria, lease, idempotency or concurrency rules.

## Start the local server

The local profile uses SQLite, local authentication and loopback binding by
default:

```bash
go run ./cmd/wos server
```

Default runtime values:

```text
listen:             127.0.0.1:8080
API prefix:         /api/v1
SQLite file:        ./data/wos.db
local principal:    local-user
request deadline:   15s
migrate on start:   true
```

Supported environment overrides in Wave 05:

```bash
export WOS_LISTEN=127.0.0.1:8080
export WOS_SQLITE_PATH=./data/wos.db
export WOS_LOCAL_PRINCIPAL_ID=local-user
go run ./cmd/wos server
```

MCP is disabled in the standalone default until its own implementation wave.

## Health

```bash
curl -i http://127.0.0.1:8080/livez
curl -i http://127.0.0.1:8080/readyz
```

`/livez` checks process liveness. `/readyz` checks that the SQLite storage
can report a compatible migrated schema.

## Addressing

Every domain route carries the Namespace explicitly:

```text
/api/v1/namespaces/{namespace_id}/outcomes/{outcome_id}
```

The server never derives a Namespace from external metadata.

All WOS IDs in this contract are normalized UUIDv7 values.

## Mutation contract

Every HTTP mutation requires an `Idempotency-Key` between 16 and 128 safe
ASCII characters. Retries of the same logical command must reuse the same key.

Versioned mutations also require a persisted aggregate version. Supply it
through either:

```http
If-Match: "work_item:<uuid>:v5"
```

or a JSON `expected_version` field. When both are supplied, they must match.

Successful individual aggregate responses return a strong ETag:

```http
ETag: "outcome:<uuid>:v2"
```

`X-Correlation-ID` is optional. WOS preserves a supplied value or creates one
for the response.

## Reproducible local flow

Set a UUIDv7 Namespace first:

```bash
NS=0199e100-0000-7000-8000-000000000001
BASE=http://127.0.0.1:8080/api/v1/namespaces/$NS
```

Create an Outcome:

```bash
curl -i -X POST "$BASE/outcomes" \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: docs-create-outcome-0001' \
  -d '{
    "title": "Validar a fatia HTTP do WOS",
    "desired_state": "Estado M1 pode ser criado e retomado via HTTP",
    "description": "Fluxo local sem dependência de Agent",
    "priority": "normal"
  }'
```

The response is `201 Created`, includes `Location`, a strong `ETag`, the
created entity, `command_id` and `outcome_revision`.

Before activation, the current domain contract requires at least one required
active criterion. Using the Outcome ETag returned above:

```bash
curl -i -X POST "$BASE/outcomes/$OUTCOME_ID/criteria" \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: docs-add-criterion-0001' \
  -H "If-Match: $OUTCOME_ETAG" \
  -d '{
    "title": "Validação humana",
    "required": true,
    "verification_mode": "attestation"
  }'
```

The criterion mutation returns the new ETag of its owner aggregate. Use that
ETag to activate the Outcome:

```bash
curl -i -X POST "$BASE/outcomes/$OUTCOME_ID/actions/activate" \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: docs-activate-outcome-0001' \
  -H "If-Match: $OUTCOME_ETAG" \
  -d '{}'
```

Create a WorkItem directly under the Outcome:

```bash
curl -i -X POST "$BASE/outcomes/$OUTCOME_ID/work-items" \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: docs-create-work-0001' \
  -d '{
    "title": "Executar prova HTTP",
    "priority": "normal",
    "lifecycle": "todo"
  }'
```

Claim it with the WorkItem ETag:

```bash
curl -i -X POST "$BASE/outcomes/$OUTCOME_ID/work-items/$WORK_ID/actions/claim" \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: docs-claim-work-0001' \
  -H "If-Match: $WORK_ETAG" \
  -d '{"lease_ttl_seconds":300}'
```

Complete it using the returned claim ID, fencing token and newest WorkItem
ETag:

```bash
curl -i -X POST "$BASE/outcomes/$OUTCOME_ID/work-items/$WORK_ID/actions/complete" \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: docs-complete-work-0001' \
  -H "If-Match: $WORK_ETAG" \
  -d '{
    "claim_id": "<claim-id>",
    "fencing_token": 1,
    "result_summary": "Prova concluída",
    "reason": "Execução confirmada pelo consumidor HTTP"
  }'
```

Read the M1 state projection:

```bash
curl -i "$BASE/outcomes/$OUTCOME_ID/state"
```

The state response is `Cache-Control: no-store` and contains the persisted
Outcome, its Objectives, its WorkItems and the current `outcome_revision`.

## Errors

Errors use a stable envelope:

```json
{
  "error": {
    "code": "version_conflict",
    "message": "version_conflict: expected version 3 but current version is 4",
    "retryable": false,
    "correlation_id": "request-42"
  }
}
```

Wave 05 mappings are:

| HTTP | Meaning |
| --- | --- |
| 400 | malformed JSON, invalid identifiers/format or invalid idempotency key |
| 404 | entity absent from the explicit Namespace/Outcome scope |
| 409 | lifecycle, precondition, lease or idempotency conflict |
| 412 | stale or mismatched `If-Match` / `expected_version` |
| 413 | request body exceeds the supported limit |
| 422 | syntactically valid request violates a domain field constraint |
| 428 | required aggregate version precondition is absent |
| 503 | request deadline or transient infrastructure timeout |
| 408 | caller/request cancellation |

Local auth intentionally uses the configured local Principal and does not
pretend to provide multi-user authentication. Token/OAuth authorization remains
a later wave.

## OpenAPI

The initial machine-readable contract is in
[`api/openapi.yaml`](../api/openapi.yaml).

The canonical architecture remains
[`docs/WOS_Design_Arquitetura_Planejamento_Atualizado.md`](./WOS_Design_Arquitetura_Planejamento_Atualizado.md).
