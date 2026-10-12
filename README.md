# WOS — Outcome Server

**Durable, agent-agnostic coordination for work that must survive sessions, processes, and handoffs.**

WOS stores the state, authority, plans, and verifiable evidence surrounding an intended result. Humans, AI agents, and applications decide what to do and execute work; WOS makes their commitments, progress, reviews, and history inspectable and recoverable.

WOS is a standalone Go server with an embeddable Core, an HTTP API, an MCP server, a human workspace, an API-only CLI (`wosctl`), and **five agent skills installable directly from the native `wos` CLI**, without Node.js or npm.

> **Release status:** The repository maintains a prepared version in `VERSION`; the documented `1.0.0` target is **not yet a certified/published stable release**. See [ROADMAP.md](ROADMAP.md) and [the release guide](docs/releases.md). A prepared version or successful local test is not a published release.

## What WOS provides

| Area | Capabilities |
| --- | --- |
| Outcomes and Objectives | Desired results, structured success criteria, explicit lifecycle and conclusions |
| Tasks | Bounded WorkItems, operational dependencies, readiness, issues and blockers |
| Planning | Draft and immutable published Roadmap revisions, activation and history |
| Contracts | Explicit work acquisition, leases, expiration, fencing, bounded context and durable receipts |
| Signed execution | Ed25519 profiles, immutable signed work/return envelopes, protected keys, independent review and correction |
| Continuity | Snapshot, focused context, timeline, graph, checkpoints, recovery and exact-intent replay |
| Evidence | Artifacts, observations, assessment history, decisions, provenance and contestable conclusions |
| Integrations | HTTP, MCP stdio/Streamable HTTP, Go SDK, durable events, triggers and signed webhook delivery |
| Interfaces | Human workspace, `wos` server/operator CLI, `wosctl` agent/work CLI, native agent skill installer |
| Persistence | SQLite, PostgreSQL, migrations, tested restore paths and transactional consistency |

### What WOS does not do

WOS **does not run or schedule agents**, make autonomous planning decisions, execute Tools, infer that evidence is approval, synchronize working directories, or replace an application's own business rules. A contract grants bounded authority over a WOS Task; it is **not** a Git branch lock, filesystem lock or permission to execute arbitrary host commands.

## Quick start: local server

### Docker Compose

```bash
git clone https://github.com/A1b3rt0M3rcad0/wos.git
cd wos
cp .env.example .env
docker compose up --build -d
```

Open **http://127.0.0.1:8080/app/** and sign in using `WOS_BOOTSTRAP_TOKEN` from `.env`. The default example binds to loopback, uses persisted SQLite and includes **development-only** credentials. Change all example credentials before configuring a shared or internet-accessible deployment.

```bash
docker compose logs -f wos
docker compose down            # persistent volumes are retained
```

For PostgreSQL, set `WOS_STORAGE_DRIVER=postgres`, `WOS_POSTGRES_PASSWORD` and `WOS_POSTGRES_DSN` for the `postgres` Compose service, then use `docker compose --profile postgres up --build -d`. See [operations](docs/operations.md) for deployment and upgrade rules.

### Build the native Go binary

```bash
go build -trimpath -o ./bin/wos ./packages/wos-api/cmd/wos
./bin/wos version
./bin/wos config validate
./bin/wos server
```

The only npm installation is `npm install -g wosctl@latest`. On Linux or Windows x64, this package supplies the unified `wos` command through a small platform/integrity launcher. The `wos skill` operation itself is implemented in Go and does not use Node.js logic or a network connection.

The source declares Go 1.27. The `wos` executable contains the HTTP/MCP server, embedded web application, and built-in skill bundles. Client operations such as `wos work` and `wos review` use the public API and do not open the server database; `wosctl` is retained as a compatibility alias.

**Supported release target:** Linux amd64 for the standalone service, SQLite or PostgreSQL 18.6, HTTP and MCP. The standalone `wosctl` client also has Linux/Windows amd64 native acceptance. Other platforms or runtimes require additional certification; compiling a binary alone does not provide it.

## Native agent skill installation — `wos skill`

The five maintained WOS skills ship **inside the Go `wos` executable**. Installation copies local instruction files into an agent's discovery directory. It performs no network request, needs no WOS server, does not run an agent and **does not require Node.js or npm**.

```bash
# Install all five WOS skills in the current project's Codex discovery directory.
wos skill install --agent codex

# Other supported discovery layouts.
wos skill install --agent claude
wos skill install --agent hermes
wos skill install --agent openclaw

# Preview first, or install for your user rather than one project.
wos skill install --agent codex --dry-run
wos skill install --agent codex --global

# Install into all four supported agent discovery directories.
wos skill install --agent all

# Explicit path for another Agent Skills-compatible runtime.
wos skill install --agent generic --dir /absolute/path/to/skills

# Inspect, update from this binary's embedded version, or safely remove.
wos skill list --agent codex
wos skill doctor --agent codex
wos skill update --agent codex
wos skill uninstall --agent codex
wos skill version
```

Use `--project /path/to/project` to select a project root instead of the current directory. Windows commands use native platform paths for `--project` and `--dir`.

| Agent selector | Project install directory | User-global install directory |
| --- | --- | --- |
| `codex` | `.agents/skills/` | `~/.agents/skills/` |
| `claude` | `.claude/skills/` | `~/.claude/skills/` |
| `hermes` | `.hermes/skills/` | `~/.hermes/skills/` |
| `openclaw` | `skills/` under the selected workspace | `~/.openclaw/skills/` |
| `generic` | Explicit `--dir` | Explicit `--dir` |

Installed skills:

- `wos-coordination` — bounded coordination, contract authority, progress and handoff.
- `wos-delegation` — task decomposition and controlled handoff to host-managed subagents.
- `wos-continuity` — continuation and original-intent recovery after session or transport loss.
- `wos-review` — material-bound assessment and independent review.
- `wos-workspace` — signed `wosctl`/YAML contract execution and recovery.

The installer embeds exactly the same source skill documents as the existing separately distributed skill package. It preflights every destination, tracks managed files with content hashes, refuses symlinks/unmanaged collisions and **preserves user modifications** instead of overwriting them. `--dry-run` is read-only; `update` installs the version bundled with the **invoking WOS binary**, not an unreviewed latest version from a registry. Concurrent writes into the same destination are not supported.

Installing instructions does **not** configure MCP, issue credentials, enroll signing keys or activate signed contracts. Those steps remain explicit and authorized. A runtime may require restart/reload to discover newly installed skills. The installer verifies files and supported layouts; execution by every version of each third-party runtime is a separate integration test.

## Concepts and lifecycle

```text
Namespace (security boundary)
└── Outcome (desired, verifiable result)
    ├── SuccessCriteria / Assessments / Conclusions
    ├── Objectives (optional hierarchy; distinct from Task dependencies)
    ├── WorkItems / ExecutionContracts / ReviewCases
    ├── Roadmaps (drafts, immutable published revisions, active slots)
    ├── Issues / Blockers / Relations
    ├── Evidence / Artifacts / Decisions
    └── Events / Triggers / Integration deliveries
```

- An **Outcome** describes the final state sought, not the number of tasks finished.
- An **Objective** describes a verifiable intermediate condition and may organize other Objectives.
- A **WorkItem** is a bounded operational task; its `ready`, `blocked` and `scheduled` states are derived projections, not arbitrary lifecycle values.
- A **Roadmap** references live Objectives and WorkItems. Publishing freezes a revision; activating it is a separate operation. Roadmap order does not automatically add dependencies.
- An **Issue** records a problem; a **Blocker** records a specific blocking impact. Resolving one does not implicitly resolve the other.
- **Evidence** records a source or observation, while a **CriterionAssessment** explicitly evaluates the applicable criterion revision. A passing test or signed delivery alone is not acceptance.
- A **Conclusion** preserves the exact assessed obligations; subsequent evidence can contest it without rewriting history.

An achieved Task does **not** automatically achieve its Objective or Outcome. WOS requires each result to be concluded explicitly under the applicable criteria and authority policy.

## Signed work and independent review

The recommended new-work protocol is `signed_contracts_v2` with workspace schema 2. It is an **explicit Namespace opt-in**; installing a skill or creating a profile does not activate it.

### 1. Set up an approved workspace

An administrator must first provision the server's persistent Ed25519 issuer, Namespace policy, credentials, reviewer permissions, and agent-key enrollment. The client receives the server ID, independently approved issuer fingerprint, Namespace/Outcome IDs and authorized enrollment reference.

```bash
wos init --workspace-schema 2 \
  --server https://wos.example.test \
  --server-id SERVER_UUID \
  --namespace NAMESPACE_UUID \
  --outcome OUTCOME_UUID

wos profile create executor \
  --token-stdin --generate-signing-key \
  --enrollment ENROLLMENT_UUID \
  --server https://wos.example.test \
  --server-id SERVER_UUID \
  --issuer-fingerprint sha256:APPROVED_FINGERPRINT

wos --profile executor auth status --output json
wos --profile executor doctor --output json
```

Give each independent executor or reviewer its **own authorized Principal/profile**. Secrets remain in protected OS keyring references, protected mounts or approved environment references; they must not appear in committed YAML or model prompts. The native keyring must be available for the `--generate-signing-key` example. Alternative protected references are documented in [signed workspace profiles](docs/signed-workspace-profiles.md).

### 2. Acquire, execute, and return a contract

```bash
wos --profile executor work checkout --next --count 1 --output json
wos --profile executor work list --output json
wos --profile executor work show CONTRACT_UUID --for-agent --output json

# Edit the permitted execution draft in the issued local contract YAML.
wos --profile executor work finish CONTRACT_UUID --output json

# Reconcile frozen operations after an interrupted or uncertain response.
wos --profile executor work recover --all-pending --output json
```

The local layout uses `.wos/project.yaml`, `.wos/profiles/<profile>/profile.yaml`, and only that profile's outstanding `contract/<id>.yaml` files. Issued specifications and signed original intentions are immutable. A verified receipt closes the corresponding local obligation; cleanup preserves unconfirmed edits. Delivery to independent review is **not** the same as Task approval.

### 3. Review and request correction

```bash
wos --profile reviewer review checkout --next --count 1 --output json
wos --profile reviewer review show REVIEW_CONTRACT_UUID --for-agent --output json

# Edit the exact review draft and criterion assessments, then:
wos --profile reviewer review finish REVIEW_CONTRACT_UUID --output json
wos --profile reviewer review recover --all-pending --output json
```

Approval is bound to the immutable submission and its current criterion revisions. `changes_requested` is a distinct outcome that may require a fresh correction contract; a valid signature never substitutes for material-quality review. Review authority and execution authority are independently enforced.

### Protocol activation and recovery

The administrator must drain legacy writers and outstanding unsigned authority before activating signed v2. The client supports `wos protocol get`, `set`, `preflight` and `recover` with exact CAS versions. Do **not** change Namespace protocol during routine `work checkout` or by skill installation. See [signed cutover](docs/signed-protocol-cutover.md), [issuer recovery](docs/issuer-recovery.md) and [signed returns](docs/signed-cli-returns.md).

Legacy `contracts_v1`/workspace-schema-1 operations remain supported for the historical path documented in [wosctl](packages/wos-cli/README.md). They cannot bypass a signed Namespace's execution authority.

## Interfaces and integration

### HTTP API

The HTTP API and MCP adapter call the same Application services and share authorization, optimistic concurrency and durable idempotency. The generated public command catalog and OpenAPI documentation live at:

- `GET /api/v1/commands`
- `POST /api/v1/commands/{name}` using `Idempotency-Key` and `{"command": ...}`
- `GET /api/v1/namespaces/{namespace_id}/outcomes`
- `GET /api/v1/namespaces/{namespace_id}/outcomes/{outcome_id}/continuity`
- `GET /livez`, `GET /readyz`
- [`packages/wos-api/openapi.yaml`](packages/wos-api/openapi.yaml)

A timed-out mutation may have **committed successfully**. Reconcile the original receipt or retry the **identical request, expected version and idempotency key**. Do not increase a stale version automatically to force a write.

### MCP

The server supports Streamable HTTP (default path `/mcp` when enabled) and stdio:

```bash
# Local MCP server, with trusted loopback identity and no HTTP socket.
WOS_MCP_ENABLED=true wos mcp stdio

# Remote service exposes /mcp when WOS_MCP_ENABLED=true.
```

Remote MCP requests use their own authenticated credentials and the same Namespace permissions as HTTP. Tools expose explicit commands and bounded queries; WOS never silently executes the requested agent work. The active supported protocol profiles and transport examples are recorded in [public contracts](docs/contracts.md).

### Go Core / SDK

- **Embedded mode:** import the public `packages/wos-core` domain/application/storage packages in a trusted Go host.
- **Remote mode:** use the lightweight `packages/wos-sdk-go` HTTP client for commands/queries.
- **CLI mode:** use native `wos` for explicit API-only workspace operations and durable local recovery.

See [embedded](examples/embedded) and [remote client](examples/remote-client) examples. Trusted embedded composition is not a shortcut around security for an untrusted multitenant host: use the authorized service and resolve identities outside model-controlled payloads.

### Durable integration events

WOS emits immutable Domain Events and durable Integration Events. Declarative triggers produce signals; an optional worker delivers signed webhooks with at-least-once semantics. Consumers must verify HMAC signatures and deduplicate by integration-event identity. Triggers **do not launch agents or execute work**.

## Concurrency, integrity, and limits

- **Optimistic concurrency:** mutating aggregates require applicable expected versions.
- **Coordination:** cross-entity invariants are guarded by the owning Outcome; dependency graphs cannot form cycles.
- **Leases:** execution has expiry and fencing; stale holders cannot conclude current work.
- **Idempotency:** remote mutations use stable keys, durable operation results and exact-intent recovery.
- **Read consistency:** continuity and graph cursors bind scope/revision and report truncation/omissions.
- **Authorization:** Namespace, authenticated Principal, declared ActorRef and external context are different concepts; metadata never grants authority.
- **Immutable records:** plans, published submissions, signatures, historical assessments and conclusions are not silently overwritten.

HTTP/MCP payloads and many query results are bounded to 256 KiB; page sizes and YAML drafts have additional limits. A bounded response size does **not** imply bounded database work: large histories and highly contended Outcomes remain subject to measured capacity limits. See [benchmarks](docs/benchmarks.md), [contract load](docs/work-contract-load-evidence.md) and [signed acceptance](docs/signed-acceptance-matrix.json).

## Deploying and operating WOS

**Authentication:** local mode is intentionally loopback-only. Shared mode uses `WOS_AUTH_MODE=api_token`, scoped credentials and Namespace grants. Browser sessions are secure, bounded and revocation-aware. The WOS process does not terminate TLS; expose it behind a correctly configured TLS ingress.

**Databases:** SQLite is suitable for local/single-service operation; PostgreSQL is the recommended shared-service store. Existing migrations are numbered, checksummed and forward-only. A mismatched or future schema must not be served as ready.

**Backup and restore:** SQLite uses `wos db backup` and `wos db restore` (new destination); PostgreSQL uses `pg_dump`/`pg_restore` into a clean database. Preserve all history, security and outbox tables; restoring only current entity rows destroys continuation guarantees. Backup validation and restore drills are deployment responsibilities.

**Issuer and client trust:** signed mode uses an explicitly configured persistent server issuer. Rotation/recovery is an offline authorized operation; pinned historical public keys remain needed to verify accepted documents. See [issuer recovery](docs/issuer-recovery.md).

**Observability:** JSON logs and `/metrics` expose command/query/HTTP/delivery counters and timing information without logging secret values or documentary payloads. Current runtime metrics are not yet a production SLO or complete percentile-tracing system.

**Retention:** idempotency retention defaults to seven days. Domain history and administrative receipts are not automatically purged; plan database capacity and define a retention/archive policy before sustained high-volume use.

## Development and validation

```bash
go test ./...
go test -race ./...
go vet ./...

# Native offline skill-installer tests (no Node.js runtime needed).
go test ./packages/wos-skill/...

# Deterministic HTTP standalone journey (after building bin/wos).
python3 tests/acceptance/journey.py --binary bin/wos --client http
```

Storage and signed-protocol acceptance require a real PostgreSQL test instance configured through `WOS_TEST_POSTGRES_DSN` and, where required, `WOS_TEST_POSTGRES_CONTAINER`. Do not point the test suite to a production database. Browser and release packaging have separate repository workflows and dependencies. A skipped optional test is not a successful execution of that test.

The CI/release system consumes declared changes from reviewed source, automatically records only version metadata and runs exact-source Core/browser/CLI/distribution checks. It validates native receipts and publishes qualified artifacts following the WKS model; manual dispatch is reserved for recovery. See [releases](docs/releases.md).

### 1.0.0 release criteria

The intended stable `1.0.0` should only be tagged after all of the following have corresponding same-source evidence:

1. Required CI checks pass on the selected commit (including real SQL, race, migration/restore, browser, native CLI and distribution).
2. Public HTTP/MCP/CLI contracts, release versions, metadata and backward compatibility are frozen and consistent.
3. Signed work/review/correction, exact recovery, authority revocation and multi-actor handoff complete a real end-to-end journey using packaged binaries.
4. Critical T01–T96 acceptance assertions are audited; incomplete scenarios are explicitly excluded from 1.0 claims or closed by additional tests.
5. PostgreSQL contention, continuity-read cost, growth and supported operational limits have measured acceptance thresholds.
6. Backup/restore, retention, issuer handling, observability and supported deployment security are verified for the advertised production profile.
7. Essential human workflows receive manual UX/accessibility validation, with unsupported claims explicitly withheld.
8. Versioned artifacts are built from the accepted source, hashes/native receipts agree, and explicit publication plus fresh-install verification succeeds.

**A prepared `0.3.0` is not evidence that these gates are already complete.** Source-of-truth implementation state is [ROADMAP.md](ROADMAP.md); historical test reports are source-specific, not blanket certification for later commits.

For external clients, follow [client qualification](docs/client-qualification.md): explicit protocols/profiles, receipt-based recovery, operation/restore probes and separate evidence, assessments and conclusions.

## Repository layout

```text
packages/
  wos-core/        domain, application, ports, Memory/SQLite/PostgreSQL adapters
  wos-api/         Go service, HTTP/MCP, embedded human workspace
  wos-cli/         native API-only wosctl and YAML signed workspaces
  wos-sdk-go/      remote Go SDK
  wos-skill/       five canonical agent skills + native Go installer + optional npm distribution
  wos-npm/         optional npm wrapper for the precompiled service
examples/          trusted embedded and standalone remote clients
docs/              architecture, ADRs, contracts, operations, evidence, releases
tests/             acceptance, boundary, browser and distribution fixtures
tools/             generation, benchmarking and release automation
```

## Reference documentation

- [Development contract](AGENTS.md)
- [Live implementation roadmap](ROADMAP.md)
- [Architecture and decisions](docs/WOS_Design_Arquitetura_Planejamento_Atualizado.md)
- [HTTP/MCP public contracts](docs/contracts.md)
- [Client operations](packages/wos-cli/README.md)
- [Signed profiles](docs/signed-workspace-profiles.md)
- [Work contracts](docs/work-contract-operations.md)
- [Backup, deployment and telemetry](docs/operations.md)
- [Current UX acceptance](docs/ux/final-acceptance.md)
- [Release and publication](docs/releases.md)

## License

WOS is distributed under [Apache License 2.0](LICENSE). Third-party notices accompany release artifacts.
