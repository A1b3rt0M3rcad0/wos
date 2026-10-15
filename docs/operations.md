# Operating WOS

The standalone validation candidate supports Linux amd64 with SQLite or PostgreSQL 18.6. Shared contracts, race detection, HTTP/MCP and recovery have executable evidence in [backend verification](verification-2026-10-07.md) and [current UX acceptance](ux/final-acceptance.md). The native Windows CLI has separate hosted acceptance; cross-compiling a service does not establish runtime support. Release publication and integration into a consumer such as Woobe are separate actions.

## Local installation

```sh
go mod download
go build -trimpath -buildvcs=false -o bin/wos ./packages/wos-api/cmd/wos
./bin/wos config validate
./bin/wos server
```

Open `http://127.0.0.1:8080/app/`. Local mode accepts only loopback bind and Host values. It creates the Namespace configured by `WOS_LOCAL_NAMESPACE_ID` (default `0199d000-0000-7000-8000-000000000001`) with `WOS_LOCAL_NAMESPACE_NAME` (default `Local`). Local identity is composition configuration, not multiuser authentication. Special administrative privileges require `WOS_LOCAL_ADMIN_OVERRIDES=true`.

Core remains usable through `application.NewService` in trusted embedded composition. A host accepting third-party calls must use `NewAuthorizedService` and resolve authenticated identity outside the payload.

## Shared service

Configure `WOS_AUTH_MODE=api_token`, `WOS_BOOTSTRAP_NAMESPACE_ID` (UUIDv7), `WOS_BOOTSTRAP_NAMESPACE_NAME`, `WOS_LOCAL_PRINCIPAL_ID` and `WOS_BOOTSTRAP_TOKEN` (a random secret of at least 32 characters). `WOS_LISTEN` can then use a public interface. Terminate HTTPS at the ingress; the process does not terminate TLS. Keep tokens out of versioned commands, logs and URLs.

Bootstrap applies only to initial installation. If credentials already exist, it cannot restore grants or resurrect revoked tokens. Losing all administrative credentials requires controlled offline recovery; environment configuration provides no login bypass.

Each credential binds a Namespace, Principal and ActorRef. The browser exchanges it for an HttpOnly, SameSite Strict session lasting at most 12 hours. Revoking or expiring the original credential invalidates its sessions. Cookies are Secure outside loopback HTTP. Mutations reject foreign origins.

`GET /api/v1/namespaces/{namespace_id}/administration` returns the administrative version, grants, credential metadata and up to 100 recent audit entries per collection. `truncated` means omissions cannot imply completeness. `POST /api/v1/security/commands` accepts `AdministrativeIntent`: `namespace_id`, `expected_namespace_version`, `operation` and its specific fields, with `Idempotency-Key`. Operations are `set_grant`, `issue_credential`, `revoke_credential` and `create_namespace`. Creation copies only the creator's current permissions and issues a new Namespace credential valid for 12 hours. The returned administrative version belongs to the originating Namespace; the new one starts at version 1.

Issued secrets appear only in the first response. Replay returns a receipt with `token_omitted=true`. If the original response was lost, inspect the credential ID, revoke it and issue another through a new intent. No recoverable secret is persisted. Namespace CAS, current authorization, changes, audit and receipt are atomic. Outcome mutations revalidate grants and credentials inside their transaction before revealing replay or persisting state.

`WOS_INDEPENDENT_REVIEWER=true` requires the assessor/concluder not to have executed work in the same Outcome. The policy compares Principal over the full history; an ActorRef alias is not a new reviewer.

## PostgreSQL and containers

```sh
WOS_STORAGE_DRIVER=postgres WOS_POSTGRES_DSN='postgres://USER:PASSWORD@HOST:5432/DB?sslmode=require' ./bin/wos server
```

The DSN is secret configuration. The adapter uses pgx, Serializable transactions, an Outcome guard, a pool of up to 16 connections, a 5-second lock timeout and a 30-second statement timeout. Transient conflicts return `transaction_conflict`; expected versions never change silently. Storage acceptance uses real PostgreSQL. SQLite success does not establish PostgreSQL parity.

For the local Docker example:

```sh
cp .env.example .env
docker compose up --build -d
```

Open `http://localhost:8080/app/`. The supplied example bootstrap credential enables a local demonstration without additional edits; replace it before exposing an instance. `compose.yaml` binds only loopback on the host, enables HTTP/MCP and uses persistent SQLite by default. `Dockerfile` builds CGO-free binaries and runs as UID/GID `10001:10001`, with CA certificates and `/data`. Container and Compose acceptance are recorded with the delivered source and tested scope in [the final record](ux/final-acceptance.md).

For PostgreSQL, configure a nonempty password, `WOS_STORAGE_DRIVER=postgres` and a DSN using host `postgres`, then run `docker compose --profile postgres up --build`. Wait for database health before accepting service traffic. SQLite default Compose acceptance does not certify every optional deployment configuration.

## MCP

Enable `WOS_MCP_ENABLED=true` for Streamable HTTP at `/mcp`. The service uses official Go SDK v1.8.0 and stateless JSON responses; business state belongs in the database. Remote calls use `Authorization: Bearer ...`. Tools revalidate identity and authority on each call.

For local stdio, use `WOS_MCP_ENABLED=true ./bin/wos mcp stdio`. This opens no HTTP socket. stdout is reserved for the protocol and logs use stderr. OAuth/IdP is not implemented. See [public contracts](contracts.md) and the real-client tests.

## Backup, restore and upgrade

For SQLite backup, run `WOS_SQLITE_PATH=./data/wos.db ./bin/wos db backup ./backup/wos.db`. The adapter uses `VACUUM INTO` for a consistent snapshot of all tables. Never copy only the main database file while WAL is active.

For SQLite restore, stop the service and workers, then use `./bin/wos db restore ./backup/wos.db ./data/restored.db`. The destination must be new; restoration never overwrites an existing installation. Configure the restored path, apply `db migrate`, check `/readyz` and validate continuity before accepting traffic.

For PostgreSQL, use `pg_dump --format=custom` and `pg_restore --single-transaction` into a new database with the matching client version. Include the full schema, history, grants, credentials, idempotency and outbox. Restoring only current entities loses coordination state. Shared acceptance exercises clean restore and continuity; historical upgrade evidence has its own source-bound report.

Numbered migrations are checksummed and forward-only. `db migrate` applies pending migrations; opening the service also migrates. Unknown schemas cannot become ready. Destructive SQL downgrade is unsupported. Operational rollback uses the older binary and a pre-upgrade backup in a separate database, explicitly reconciling subsequent facts.

Leases retain absolute UTC times and fencing. Synchronize clocks after restore; expired leases require explicit reclaim. Restore may redeliver an integration event the consumer already received, so deduplicate by `integration_event_id`. Session memory is not authoritative completion state.

## Integrations and observability

Configure `WOS_WEBHOOK_ENDPOINTS` as an array of `{id,namespace_id,url,secret_ref,key_id}`. Secrets belong in `WOS_WEBHOOK_SECRETS`, with random values of at least 32 characters. Trigger commands contain no secret values. `WOS_DELIVERY_WORKER_ENABLED=true` starts the service worker; `WOS_WEBHOOK_ALLOW_LOOPBACK=true` explicitly enables local delivery tests.

HTTPS targets use a configured host allowlist, DNS/IP checks, blocked private addresses and redirects, a 10-second timeout and a 64 KiB response-read limit. Triggers make no external calls inside domain transactions. Facts, signals and delivery state persist atomically. The worker uses a 30-second fenced lease, up to six attempts, exponential backoff starting at 2 seconds, crash recovery and authorized explicit redelivery. Consumers receive the same stable event ID on retries.

The signature is `X-WOS-Signature = sha256=<HMAC-SHA256(secret, timestamp + "." + eventID + "." + body)>`, with `X-WOS-Event-ID`, `X-WOS-Timestamp` and `X-WOS-Key-ID`. Verify the raw body, the five-minute window and durable deduplication. Go consumers can use `integration.VerifySignature`. Retain old Key-IDs as needed for pending attempts during secret rotation.

JSON logs record command, Principal/ActorRef, correlation, revision, duration, transaction/guard acquisition, commit, classified error and replay. Queries record identity metadata, revision, duration and error outside Domain. HTTP logs include method, path without query, duration, status and size; they exclude bodies, authentication headers, tokens and documentary payloads.

`/metrics` returns JSON schema 1 with command/replay/conflict/request counters and `command_timing`, `query_timing`, `http_timing` and `delivery_timing`. Timing values are nanosecond sums, counts, maxima and errors. Commands include `transaction_wait_ns`, `guard_ns` and `commit_ns`; guard timing includes acquisition and coordination preparation, not just blocked time. SQLite contention primarily occurs at transaction acquisition. Delivery labels describe classified results, never secrets or URLs. Remote metrics require `namespace:admin`. Counters reset on process restart. This version has no histogram/percentile or OpenTelemetry exporter; benchmark SQL/pool-wait measurements are not a production endpoint.

Outcome idempotency retention defaults to seven days. History and administrative receipts are not automatically purged. Plan capacity and backups; archiving retains history. See [query contracts](contracts.md) for response limits.

In remote PostgreSQL composition, database time arbitrates leases after guard acquisition; replica clocks measure durations. A trusted embedded host choosing `NewService` supplies deployment time authority; `NewAuthorizedService` uses transactional PostgreSQL time. See ADR-016.

## Builds behind an HTTPS proxy

The Dockerfile uses the pinned official Go image and copies its standard CA bundle into Alpine. A proxy with a private CA can supply `docker build --secret id=go-ca-bundle,src=/path/to/bundle.pem .`. The temporary mount is used only for verified-TLS Go downloads; it is not copied into the runtime and is unnecessary for ordinary installations. Build proxy and DNS settings belong to the host.

For the stable-plan capacity envelope, data-class retention, hourly backup targets, clean restore checklist and reference diagnostics/alerts, see [the operational reference](stable/operations-reference.md).

## HTTP connection and request bounds

The standalone server uses explicit reference bounds, configurable through the
same environment in native and Compose deployments:

| Setting | Default | Meaning |
| --- | --- | --- |
| `WOS_HTTP_READ_HEADER_TIMEOUT` | `5s` | Receive complete request headers |
| `WOS_HTTP_READ_TIMEOUT` | `30s` | Socket read bound |
| `WOS_HTTP_REQUEST_TIMEOUT` | `15s` | Request context; body reads are interrupted at this deadline plus a small timer margin |
| `WOS_HTTP_WRITE_TIMEOUT` | `45s` | Ordinary response socket deadline |
| `WOS_HTTP_IDLE_TIMEOUT` | `60s` | Keep-alive idle bound and dedicated standalone MCP GET stream lifetime |
| `WOS_HTTP_MAX_HEADER_BYTES` | `65536` | Header parsing limit (Go also reserves a small parser buffer) |
| `WOS_HTTP_MAX_INFLIGHT_REQUESTS` | `128` | Nonblocking ordinary request admission |
| `WOS_MAX_CONNECTIONS` | `256` | TCP connection admission, including incomplete headers and idle clients |
| `WOS_SHUTDOWN_TIMEOUT` | `15s` | Drain grace, followed by forced connection close |

`wos config validate` rejects invalid/overflowed durations, zero/negative limits,
read shorter than header, write not greater than request or shorter than read, headers outside
1 KiB–1 MiB, requests outside 1–4096, and connections outside requests+2–65536.
Shutdown may deliberately be shorter than a request: the server forces close
at the end of that grace and the caller must reconcile an uncertain mutation.

Ordinary saturation returns JSON `overloaded`, HTTP 503 and `Retry-After: 1`
before invoking a command. Two separate request slots remain available for
`GET /livez` and `GET /readyz`; health still reports actual storage/issuer health.
Connection saturation closes newly accepted sockets without starting HTTP work.
Reserved health requests do not defeat a flood that consumes all TCP capacity;
public ingress must enforce TLS, connection/rate/header/body/time limits and
capacity appropriate to the deployment. Forwarded client IP is not trusted for
authorization or built-in identity limits.

Request bodies have a global 272 KiB framing bound; command/MCP/session decoders
retain narrower limits, including the 256 KiB command contract. Socket body-read
and context deadlines are distinct. Stateless Streamable HTTP MCP uses JSON
responses. If a host enables stateful protocol transport, standalone GET SSE
uses its own bounded idle-timeout lifetime and both socket deadlines; the client
reconnects without implicitly repeating a mutation. Protocol sessions do not own
WOS business state. Metrics distinguish in-flight requests/connections,
admission rejection, request cancellation/deadline and connection timeout, with
fixed aggregate labels and no token, webhook URL, prompt or documentary content.
Idle/stream expiry is not by itself a service incident.

A deadline, disconnect or overload never proves that a prior attempt did not
commit. Keep payload, CAS, fence and idempotency key unchanged; recover the
original authenticated receipt before deciding what to do next. Existing
`deadline_exceeded` error semantics remain in place. No SDK mutation retry is
introduced by these limits.

## Delivery health (operational hardening candidate)

API health and webhook delivery health are separate. A healthy `/readyz` means the Core can serve; poll the administrative `GET /api/v1/namespaces/{namespace_id}/operations/delivery-health` for durable queue facts. Supply `?outcome_id=UUID` for an Outcome-restricted credential. The equivalent MCP tool is `wos_delivery_health`, and the Go SDK method is `GetDeliveryHealth`. Authentication and current Namespace administration are required. A `403` is not evidence of an empty queue.

The response separates `queue_state` from this process's `worker` status. Ages are seconds; unavailable historical timestamps are null. `attempts_total` counts retained attempts across explicit redelivery. `last_successful_delivery` survives redelivery, but does not certify a currently pending delivery as successful. The response excludes destinations' URLs, secret references and event bodies. Worker fatal errors are classified and exposed; restart the deployment after repairing the cause. Another replica may still deliver while this process reports degraded.

Set `WOS_DELIVERY_BACKLOG_MAX_AGE=5m` (default) to choose the due-age alarm threshold. External monitoring should alert on `queue_state=degraded`, any `exhausted`, or a worker expected by the deployment that is disabled, stopped or has no fresh tick. `/metrics` includes the aggregate `delivery_worker` process snapshot and delivery attempt timings. Poll a Namespace/Outcome diagnostic for queue counts, rather than treating a process heartbeat as queue state. External receivers must deduplicate `integration_event_id`; a successful receiver side effect followed by ACK loss can be delivered again.
