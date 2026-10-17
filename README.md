<p align="center">
  <img src="docs/images/logo.svg" width="240" alt="Hand-drawn chain logo" />
</p>

<p align="center"><strong>Meaningful outcomes. Work together.</strong></p>

<p align="center">
  <a href="https://github.com/A1b3rt0M3rcad0/wos/releases/latest"><img src="docs/images/badges/release.svg" alt="Published baseline 0.4.2" /></a>
  <a href="https://www.npmjs.com/package/wosctl"><img src="docs/images/badges/npm.svg" alt="npm version" /></a>
  <a href="https://github.com/A1b3rt0M3rcad0/wos/actions/workflows/ci.yml"><img src="https://github.com/A1b3rt0M3rcad0/wos/actions/workflows/ci.yml/badge.svg?branch=master" alt="CI on master" /></a>
  <a href=".go-version"><img src="docs/images/badges/go.svg" alt="Go 1.27.2" /></a>
  <a href="LICENSE"><img src="docs/images/badges/license.svg" alt="Apache 2.0 license" /></a>
</p>

<p align="center">
  <a href="#how-it-works">Product tour</a> ·
  <a href="#why-use-it">Why use it</a> ·
  <a href="#for-developers-and-oaas">Developers and OaaS</a> ·
  <a href="#quick-start">Quick start</a> ·
  <a href="#operate-with-agents">Agents</a> ·
  <a href="#project-status">Status</a> ·
  <a href="docs/README.md">Documentation</a>
</p>

A shared workspace and coordination backend for teams building software with people and AI agents. Define the result you want, break it into work, and keep the delivery, evidence and review together across tools and sessions.

An agent can finish a Task without delivering the result you asked for. Here, "the code is done" and "the result meets our criteria" are separate records. A person or another authorized agent checks the material before the Outcome is explicitly accepted. When independent review is required by policy, the reviewer must be a different authenticated identity.

Use the browser to plan and inspect work, the CLI to guide coding agents such as Codex or Claude Code, or the APIs to connect your own product. Your runtimes execute and schedule the work; the server persists state and enforces coordination rules.

## How it works

Suppose your team needs to **ship partial refunds**. Agree on the expected behavior, assign implementation and testing Tasks, and record the exact change and test results. A completed Task is one step toward the result. Criteria still need assessment, and the Outcome needs an explicit conclusion.

<p align="center">
  <a href="docs/images/workspace-tour.gif"><img src="docs/images/workspace-tour.gif" width="960" alt="Workspace navigation through the Canvas, Tasks, Evidence and criteria" /></a>
</p>

The tour navigates sample records in the real application: **Canvas → Task details → Work → Evidence → Overview and criteria**. It shows a completed Task alongside criteria still awaiting assessment. Navigation does not execute the work or demonstrate a signed independent review. [Static view](docs/images/workspace-tour.png) · [Recording details](docs/images/workspace-tour.md) · [All desktop and mobile pages](docs/design/operational-canvas/workspace/README.md).

| Step | What you do | What stays recorded |
| --- | --- | --- |
| Define the result | Create an Outcome and its success criteria | What must be true before you accept the result |
| Plan the work | Add Tasks, optional Objectives and explicit dependencies | Who needs to do what, and what is blocked |
| Execute | A participant acquires a Task and works in their own tools | Bounded authority, instructions and reported progress |
| Deliver | Register the change, output and supporting material | The submitted version, Artifact references and Evidence |
| Review | Inspect the actual delivery and assess the applicable criteria | The review decision and its supporting observations |
| Conclude | Explicitly accept the Outcome when its requirements are satisfied | The basis for accepting the result |

**Task completion does not achieve an Outcome. Attaching Evidence does not approve a criterion.**

<details>
<summary>The shared-work metaphor</summary>

<p align="center">
  <img src="docs/images/robots-queuing-for-paperwork.png" width="280" alt="Robots sharing work at a desk" />
</p>

The board represents shared state; each paper represents a bounded piece of work. The product tour above shows the actual interface.

</details>

## Why use it

| When this happens | What changes in practice |
| --- | --- |
| Two agents try to take the same Task | One current acquisition grants authority at a time. Expiry and fencing checks reject stale deliveries. |
| An agent says "done," but nobody has checked the result | Keep delivery and assessment separate; require an eligible independent reviewer when the policy calls for one. |
| A session ends and its context disappears | The next participant reads the recorded instructions, activity and remaining obligations. |
| A request times out and you cannot tell whether it succeeded | Inspect its durable receipt and reconcile the original request instead of guessing or submitting a different operation. |
| People and agents work in different tools | The workspace, CLI, HTTP and MCP interfaces use the same persisted state and domain rules. |

## For developers and OaaS

For a development team, keep your repository, coding agents and CI. Use this service to coordinate work across them: make acceptance criteria explicit, issue bounded work, record immutable change references and test reports, and inspect what still needs review. An authorized participant can resume from recorded state after a session ends.

For developers building **Outcome-as-a-Service (OaaS)** products, treat a requested result as an Outcome. For example, your product could accept "add partial refunds to my store," record agreed criteria and expose progress while your workers implement and verify the change. The repository supplies the coordination layer; you build and operate the customer-facing service.

| Your product or runtime owns | The coordination service provides |
| --- | --- |
| Customer intake and agreement on the requested result | Outcomes, success criteria, Objectives and Tasks |
| Agent selection, scheduling, tools and execution | Explicit work acquisition, scoped authority and current obligations |
| Producing deliverables and inspecting their contents | Artifact references, Evidence, assessments and explicit conclusions |
| Customer access, provisioning, pricing and billing | Namespace authorization boundaries, recorded state, history and integration events |

Provision credentials and grants for each integration; a customer ID in metadata does not grant access. Webhooks report recorded changes with at-least-once delivery. Your consumer verifies and deduplicates them before deciding what to do next. See [security boundaries](SECURITY.md) and [events and webhooks](docs/integration-events.md).

### Choose an interface

| Interface | Use it for | Start here |
| --- | --- | --- |
| Browser workspace | Human planning, Canvas navigation, review and inspection | [Workspace guide](docs/workspace-ux.md) |
| Native CLI and five Skills | Coding agents acquiring work, recording progress and returning deliveries | [Agent guide](docs/agents.md) |
| HTTP API and remote Go client | Connecting a service, customer portal, job or integration | [HTTP API](docs/http.md) · [Client example](examples/remote-client) |
| MCP | Exposing authorized coordination tools to an MCP client | [MCP contracts](docs/contracts.md) |
| Public Go Core | Embedding the domain and application services in a Go host | [Architecture](docs/architecture.md#packages-and-interfaces) |

Official agent Skills operate through the CLI. MCP and HTTP are additional integration interfaces. An embedded host is responsible for its authentication and authorization boundaries; see [deployment modes](docs/operations.md).

## Quick start

### Run the current workspace

You need Git, Docker and Docker Compose v2. The server path below builds the current `master` source, including the interface shown in the tour.

```sh
git clone --branch master https://github.com/A1b3rt0M3rcad0/wos.git
cd wos
cp .env.example .env
docker compose up --build -d
```

**You should see:** the login page at **http://127.0.0.1:8080/app/**. Copy `WOS_BOOTSTRAP_TOKEN` from `.env` into **Access credential**, then choose **Sign in**. The unchanged example opens the **Main** Workspace.

> [!WARNING]
> The example contains a public development credential. Keep it on localhost. Before sharing a deployment, configure private credentials, permissions and HTTPS using the [shared-service guide](docs/operations.md#shared-service). Do not print credentials in logs.

Create an Outcome with **+** beside **Outcomes**. Add a required success criterion and a Task, then explicitly activate the Outcome when preparation is ready. Inspect the work in **Canvas**, **Work** and **Evidence**. **Choose an action** exposes the operations your permissions and the current state allow. A new installation starts with your own records, not the tour's sample data.

<a id="requirements-and-platforms"></a>
<details>
<summary>Requirements, platforms and troubleshooting</summary>

| Component | Supported path |
| --- | --- |
| Server | Linux amd64, including the published container |
| Native CLI | Linux x64 and Windows x64 release archives |
| macOS / ARM | No qualified native CLI distribution; use a browser with a supported server. Container emulation is not qualified. |
| npm launcher | Node.js 22+; native archives do not need Node.js |
| Database | SQLite for local use; PostgreSQL for shared deployments |
| Source build | Stable Go 1.27.2, pinned in [.go-version](.go-version) |

| Symptom | Check |
| --- | --- |
| Port 8080 is occupied | Run `WOS_HTTP_PORT=8081 docker compose up --build -d`, then open port 8081. |
| Docker reports permission denied or cannot connect | Start Docker and use its supported user-access setup. Check `docker info` before retrying. |
| Sign-in rejects the credential | Copy the whole value from the `.env` used for first initialization. Changing the bootstrap value does not reset an existing credential; follow [credential recovery](docs/getting-started.md#sign-in-and-persisted-data). |

For startup failures, inspect `docker compose logs --tail=100 wos`. [Service lifecycle, backups, source builds and setup help](docs/getting-started.md).

</details>

Want an executable example before configuring agents? The [isolated local lifecycle](examples/quickstart/README.md) creates one Outcome, delivers a file, assesses a criterion and concludes the result. It uses one trusted operator and the unsigned API; it does not demonstrate independent review.

## Concepts in one minute

| Term | Meaning | Partial-refund example |
| --- | --- | --- |
| Outcome | The desired result, accepted explicitly | Customers can receive partial refunds |
| Success criterion | A revisioned condition that must be assessed | Refunds above the remaining amount are rejected |
| Objective | An optional intermediate condition | Refund behavior is implemented and verified |
| Task | Concrete work for a participant; `WorkItem` in the API | Implement amount validation |
| Contract | Bounded authority issued at checkout in a contract-enabled Workspace | Permission to deliver one Task under the issued constraints |
| Artifact / Evidence | A deliverable reference, and an observation or source used to check it | An exact commit, and the recorded validation test report |
| Assessment / Conclusion | Evaluation of a criterion, and the basis for accepting the result | The amount check passed; accept the Outcome after all requirements are satisfied |
| Workspace / Namespace | The same access boundary: Workspace in the UI, Namespace in API/CLI configuration | The team's authorized space |

A Roadmap is an optional versioned plan referencing existing Objectives and Tasks. Publishing it does not execute work or create dependencies automatically.

## Operate with agents

> [!NOTE]
> The published image and `wosctl@0.4.2` predate the current workspace and centralized agent commands. Build this source for the flow below. For the published version, follow [its tagged README](https://github.com/A1b3rt0M3rcad0/wos/blob/v0.4.2/README.md) or [try its prebuilt container](docs/getting-started.md#published-container).

Build the CLI from the repository root with stable Go 1.27.2, then install its instruction files in your external agent:

```sh
go build -o bin/wos ./packages/wos-api/cmd/wos
./bin/wos init
./bin/wos skill install --agent codex
```

Use `claude`, `hermes`, `openclaw` or `all` for the other supported layouts. The five Skills guide workspace operation, coordination, delegation, continuity and review. They install instructions; they do not start an agent or grant access.

An administrator must [provision approved Profiles](docs/signed-workspace-profiles.md#provision-an-approved-identity), credentials and signing identities, and enable the appropriate Workspace protocol. After provisioning, an executor can acquire work and read its context from any directory:

```sh
./bin/wos --profile executor work available
./bin/wos --profile executor work next
./bin/wos --profile executor work context
```

The agent performs the work in its own tools, then records actual progress, Artifact references and Evidence through typed CLI commands. `work check` reports readiness and obligations; `work finish` signs, submits and reconciles the delivery. Delivery alone is not approval. Where independent review is required, another eligible Principal inspects the exact submission and records its decision and criterion assessments.

Profiles keep destinations and protected local state separate. Concurrent agents pin `--profile` on every invocation. After an uncertain response, use the CLI's recovery commands to reconcile the original request; preserve pending material. Two Profile names do not establish two independent identities.

Follow the [complete agent workflow](docs/agents.md) for recording proof, review and recovery. [CLI reference](packages/wos-cli/README.md) · [Protected Profiles](docs/signed-workspace-profiles.md) · [Signed return safety](docs/signed-cli-returns.md).

## Fit and limits

Use a checklist or issue tracker when human conventions are enough. Add this service when participants need machine-readable authority, delivery, review and recovery rules across tools. Keep Jira or Linear for the reporting they provide, and keep your agent framework for runtime orchestration; integrations remain your responsibility.

The server checks recorded authority and state. It cannot prove that claimed external execution happened. Reviewers must inspect the actual material, and Task acquisition does not lock your Git branch or filesystem. See [security boundaries](SECURITY.md).

## Project status

| Item | State |
| --- | --- |
| Published baseline | [v0.4.2](https://github.com/A1b3rt0M3rcad0/wos/releases/tag/v0.4.2), also available as `wosctl@0.4.2` and on GHCR |
| This README and tour | Describe current source, including changes not yet published in that release |
| Compatibility | Existing HTTP/CLI contracts and checked forward migrations follow the [compatibility policy](docs/stable/compatibility.md); no blanket 1.0 certification |
| Stable 1.0 acceptance | Tracked separately in [ROADMAP.md](ROADMAP.md) |

## Documentation and support

- [Documentation index](docs/README.md): setup, workspace use, agents and integration references.
- [Workspace gallery](docs/design/operational-canvas/workspace/README.md) and [Canvas guide](docs/design/operational-canvas/README.md): pages, forms, mobile states and interaction details.
- [Server operations](docs/operations.md), [architecture](docs/architecture.md) and [operational acceptance](docs/stable/operational-acceptance.md).
- [Report bugs or ask questions](https://github.com/A1b3rt0M3rcad0/wos/issues): include versions and a minimal reproduction, without secrets.
- [Contribute](CONTRIBUTING.md) or [report a security concern](SECURITY.md#report-a-vulnerability).

[Release process](docs/releases.md) · [Apache 2.0 license](LICENSE)
