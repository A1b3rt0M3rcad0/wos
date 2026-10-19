# Documentation

Start with the [project README](../README.md) to understand the purpose, run the service, and operate your first Outcome. Choose a guide for the task you need to complete.

## Use the workspace

| Task                                                | Guide                                                                  |
| --------------------------------------------------- | ---------------------------------------------------------------------- |
| Install the service, sign in, or troubleshoot setup | [Getting started](getting-started.md)                                  |
| Run one complete isolated workflow                  | [Unsigned local example](../examples/quickstart/README.md)             |
| Create Outcomes, plan, execute, and assess work     | [Workspace guide](workspace-ux.md)                                     |
| Use contextual Objectives/Tasks Kanbans, compact inspection and strict completion | [Contextual Kanbans](ux/contextual-kanban/README.md) |
| Use progressive creation and the shared contextual inspector | [Operational workspace](ux/operational-workspace/README.md) |
| Connect Objectives, Tasks, Evidence and problems without confusing completion with acceptance | [Understand the relationships](workspace-relationships.md) |
| Find Outcomes through live search and filters      | [Outcome discovery](workspace-ux.md#find-another-outcome)               |
| View the current desktop and mobile interface       | [Current Board and workspace screenshots](design/editorial/operational-workspace/README.md) |
| Inspect earlier specialist pages, native forms and recovery states | [Earlier complete workspace gallery](design/operational-canvas/workspace/README.md) |
| Understand the shared human/agent workflow          | [How it works](../README.md#how-it-works)                              |
| See real navigation through the earlier interface   | [Recorded workspace tour](images/workspace-tour.md)                    |
| Connect a development team or an OaaS product       | [Developers and OaaS](../README.md#for-developers-and-oaas)             |
| Inspect the README illustration's provenance        | [Original artwork record](images/assets.json)                          |

## Connect agents and applications

| Task                                               | Guide                                                                                               |
| -------------------------------------------------- | --------------------------------------------------------------------------------------------------- |
| Install skills and operate with an agent           | [Agent guide](agents.md)                                                                            |
| Record typed activity, evidence, and current obligations | [Durable work activity](work-activity.md) |
| Review the central workflow implementation and acceptance plan | [Implementation package](central-agent-workflow/README.md) |
| Look up client commands                            | [Native CLI](../packages/wos-cli/README.md)                                                         |
| Configure credentials and signed-contract profiles | [Signed profiles](signed-workspace-profiles.md)                                                     |
| Review deliveries and recover uncertain responses  | [Signed returns](signed-cli-returns.md)                                                             |
| Prepare or migrate a workspace/protocol            | [Protocol cutover](signed-protocol-cutover.md) and [workspace migration](workspace-migration-v2.md) |
| Use HTTP or connect MCP                            | [HTTP](http.md), [MCP contracts](contracts.md), and [OpenAPI](../packages/wos-api/openapi.yaml)     |
| Integrate a consumer and reconcile receipts        | [Client qualification](client-qualification.md)                                                     |
| Consume events and webhooks                        | [Integration events](integration-events.md)                                                         |
| Integrate contextual notes, atomic plans and coherent workspace queries | [Operational workspace API](operational-workspace-api.md) |

## Operate the server

- [Operations](operations.md): local setup, authenticated services, databases, backup, and restore.
- [Issuer recovery](issuer-recovery.md): identity and trust for signed contracts.
- [Releases and distribution](releases.md): public installation, platforms, and publication.
- [Operational reference](stable/operations-reference.md): capacity limits, retention, and diagnostics.
- [Measurements](benchmarks.md): benchmark scope and limits; technical evidence is separate from production guarantees.

## Develop and understand the project

- [Contributing](../CONTRIBUTING.md): setup, checks, and pull requests.
- [Security](../SECURITY.md): deployment boundaries and private-reporting guidance.
- [AGENTS.md](../AGENTS.md): development contract and required verification.
- [ROADMAP.md](../ROADMAP.md): implementation state, current work, and pending acceptance.
- [Architecture overview](architecture.md): domain model, invariants, and package boundaries.
- [Architecture decisions](adr/): accepted choices and rationale.
- [Dependencies](dependencies.md): toolchain and libraries.
- [Licensing](licensing.md): production/application use, platform-resale restrictions, brand rights, earlier grants and distribution notices.
- [Current interface validation](ux/operational-workspace/qualification.md): source-bound automated checks, migrations, performance, captures and remaining acceptance; [earlier operational Canvas](design/operational-canvas/validation.md) and [editorial baselines](design/editorial/validation.md) remain dated separately.

Maintained product copy and user guides use English. Original owner-supplied specifications, style packages, authored user content, and historical evidence retain their source language.

## Historical records

Dated reports, original implementation plans, ADRs, and evidence under `audit/`, `ux/`, and `stable/` describe specific revisions. They do not replace current guides or automatically establish acceptance of later versions. The [original canonical specification](WOS_Design_Arquitetura_Planejamento_Atualizado.md) is preserved as a source document; the English architecture overview explains the implemented boundaries.

The [editorial gallery](design/editorial/README.md#main-pages) preserves an earlier visual baseline; the current Outcome Board gallery is linked above. Older captures and the two October 5 reports removed from active documentation remain in Git history; acceptance references point to the preserving commit.
