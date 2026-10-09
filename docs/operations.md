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
