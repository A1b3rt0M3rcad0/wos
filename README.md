<p align="center">
  <img src="packages/wos-api/web/assets/brand/hand-drawn-chain/Hand-Drawn%20Orange%20Chain%20Icon.svg" width="320" alt="Hand-drawn chain logo" />
</p>

<p align="center"><strong>Shared state for people and agents. Verifiable outcomes.</strong></p>

<p align="center">
  <a href="#why-use-it">Why use it</a> ·
  <a href="#how-it-works">How it works</a> ·
  <a href="#quick-start">Quick start</a> ·
  <a href="docs/README.md">Documentation</a>
</p>

Keep the plan, the work, and its proof in one shared state. People, AI agents, and applications can pick up the same outcome, see what is authorized, and continue from the last recorded step.

The project provides a standalone Go service, an embedded web workspace, a native CLI, HTTP and MCP interfaces, and an embeddable Go Core. SQLite supports local operation; PostgreSQL supports shared deployments. Execution stays in the participant's own tools and environment.

## Why use it

Work often spans several people, agent sessions, and tools. Its context, ownership, delivery, and acceptance need to survive those handoffs.

| Need                                     | What the system provides                                                     |
| ---------------------------------------- | ---------------------------------------------------------------------------- |
| Continue after a session or process ends | Persisted outcomes, plans, task context, progress, and history               |
| Coordinate concurrent participants       | Explicit acquisition, scoped authority, expiry, and concurrency checks       |
| Know whether a delivery meets the goal   | Revisioned success criteria, evidence, assessments, and explicit conclusions |
| Recover after an uncertain response      | Durable receipts and reconciliation of the original intent                   |
| Use different tools over the same work   | Web, CLI, HTTP, MCP, and Go interfaces over shared application rules         |

Use it when the state of the work must remain available beyond the conversation or process that started it. Your application or agent chooses and performs actions; the server records state and enforces coordination rules.

## How it works

![A robot handing a task to another participant at a desk, with three robots waiting beside a shared work board.](docs/images/robots-queuing-for-paperwork.png)

The board represents shared state. Each paper represents a bounded task with instructions and delivery conditions. Participants explicitly request work, execute in their own environment, and return material for review.

1. **Define an Outcome:** describe the desired result and the criteria that make it verifiable.
2. **Plan the work:** create Objectives for intermediate conditions and Tasks for concrete actions. An optional Roadmap organizes versioned plans.
3. **Acquire authority:** a participant reads the context and acquires the authorization required by the workspace protocol.
4. **Execute and deliver:** perform the task outside the server and register evidence, artifacts, and the execution result.
5. **Review the material:** an authorized reviewer assesses the exact delivery and applicable criteria. A correction request leads to a new explicit acquisition.
6. **Conclude and continue:** explicitly conclude accepted results; the next participant reads persisted state and remaining obligations before continuing.

**Completing a Task does not automatically achieve an Objective or Outcome. Registering evidence does not approve a result.** Delivery, assessment, and conclusion are distinct records.

In the browser, a **Workspace** corresponds to the API's **Namespace**, the boundary for access and permissions. Each Outcome belongs to that space.

## Quick start

With Git and Docker Compose installed:

```sh
git clone https://github.com/A1b3rt0M3rcad0/wos.git
cd wos
cp .env.example .env
docker compose up --build -d
```

Open **http://127.0.0.1:8080/app/**. Enter the `WOS_BOOTSTRAP_TOKEN` value from `.env` in **Access credential**, then choose **Sign in**. The example uses persistent SQLite and a development credential for local demonstration. Configure individual credentials and permissions before sharing an instance; see [server operations](docs/operations.md#shared-service).

### Operate your first Outcome

1. Choose **+** beside **Outcomes**, enter a name and desired state, and select **Save draft**.
2. Add success criteria describing what must be checked before accepting the result.
3. Add Objectives in **Plan** and Tasks in **Work**, with instructions and references.
4. Activate the Outcome explicitly when preparation is ready; saving a draft does not activate it.
5. Open a Task to inspect context, dependencies, blockers, and permitted actions. For signed work, follow the execution command through an authorized local profile.
6. Register delivery material in **Evidence**, assess the applicable criteria, and explicitly conclude accepted results.

| Workspace area | Purpose                                              |
| -------------- | ---------------------------------------------------- |
| **Overview**   | Desired outcome, progress, and remaining preparation |
| **Plan**       | Objectives, Roadmaps, and published revisions        |
| **Work**       | Task list, board, readiness, and execution details   |
| **Issues**     | Problems and their separate blocking impacts         |
| **Evidence**   | Observations, deliverables, and assessment material  |
| **Activity**   | Events and the history of the Outcome                |

**Choose an action** and item details expose actions allowed by your permissions and the current state. See the [workspace guide](docs/workspace-ux.md) and [current screenshots](docs/design/editorial/README.md#main-pages).

## Operate with agents

The public npm package `wosctl` installs the native `wos` command. The npm launcher requires Node.js 22 or later and supports Linux/Windows x64 clients:

```sh
npm install -g wosctl@latest
wos version
wos skill install --agent codex
```

Use `claude`, `hermes`, or `openclaw` for another discovery layout, `--agent all` for all four, or `--agent generic --dir /absolute/path/to/skills` for an explicit destination. Native archives are also available from [releases](https://github.com/A1b3rt0M3rcad0/wos/releases/latest).

The binary includes five skills: coordination, delegation, continuity, review, and workspace operation. Skill installation is offline and preserves local modifications. Configure the agent runtime, MCP connection, credentials, and protocol separately.

### Signed execution and review

An administrator first configures the Namespace protocol, server identity, credentials, key enrollment, and authorized profiles. Follow [signed workspace profiles](docs/signed-workspace-profiles.md) before using this example:

```sh
wos --profile executor work checkout --next --count 1 --output json
wos --profile executor work show CONTRACT_UUID --for-agent --output json

# Execute externally and complete the permitted local execution draft.
wos --profile executor work finish CONTRACT_UUID --output json
```

Use the contract ID returned by acquisition. Its context contains instructions, constraints, and criteria. Each executor and independent reviewer uses their own authorized identity. Sending a delivery may leave it pending review; it does not approve the work.

Reconcile an interrupted or uncertain operation using its original intent:

```sh
wos --profile executor work recover --all-pending --output json
```

Keep the original key, destination, payload, and expected version during recovery. The [agent guide](docs/agents.md) covers execution, independent review, and continuation.

## Run and maintain the service

```sh
docker compose logs -f wos   # Follow service logs.
docker compose stop         # Stop the service.
docker compose start        # Resume with the same data.
docker compose down         # Remove containers; keep volumes.
```

The `wos_data` volume stores SQLite data. `down -v` removes volumes, so avoid it when preserving state. Sign in to the same Workspace to resume an Outcome. A shared link opens context but does not grant access.

See [operations](docs/operations.md) for authentication, PostgreSQL, backup, and restore. The server's validated runtime target is Linux amd64; Windows support applies to the client.

<details>
<summary>Build and run from Go source</summary>

Use the Go version declared in [go.mod](go.mod), currently 1.27:

```sh
go build -trimpath -o bin/wos ./packages/wos-api/cmd/wos
./bin/wos config validate
./bin/wos server
```

For this default trusted-local mode, open `http://127.0.0.1:8080/app/`, expand **Local development server**, enter Namespace `0199d000-0000-7000-8000-000000000001`, and choose **Open local workspace**. Use authenticated service configuration for multiple users.

The binary includes the server, web application, and skills. `wosctl` remains a compatibility client alias.

</details>

## Integrate and extend

| Interface           | Reference                                                                      |
| ------------------- | ------------------------------------------------------------------------------ |
| Web workspace       | [Human workflows](docs/workspace-ux.md)                                        |
| Native CLI          | [Commands and protocols](packages/wos-cli/README.md)                           |
| HTTP API            | [HTTP guide](docs/http.md) and [OpenAPI](packages/wos-api/openapi.yaml)        |
| MCP                 | [Protocol profiles and transports](docs/contracts.md) — `/mcp` when enabled    |
| Go                  | [Embedded Core](examples/embedded) and [remote client](examples/remote-client) |
| Events and webhooks | [Consumer integration](docs/integration-events.md)                             |

The service works independently of an AI provider or agent runtime. Integrations choose and execute their own actions while sharing the same persisted coordination state.

## Documentation and project status

- [Documentation index](docs/README.md): guides organized by task.
- [Architecture](docs/architecture.md): state model, boundaries, and execution rules.
- [Development contract](AGENTS.md): repository rules and required checks.
- [Roadmap](ROADMAP.md): implementation status and remaining acceptance work.
- [Releases](docs/releases.md): installation, platform scope, and publication requirements.

These instructions and screenshots describe this branch. [VERSION](VERSION) identifies its source version; npm `@latest` follows the published package. Stable 1.0 acceptance is tracked separately in the roadmap.

Licensed under [Apache 2.0](LICENSE).
