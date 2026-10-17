# Architecture overview

WOS is an independent state and coordination service organized around Outcomes. Humans, agents, and applications decide and perform work; the server persists their shared state and enforces the applicable rules.

This English overview describes established boundaries. [ROADMAP.md](../ROADMAP.md) records implementation status. The [original canonical specification](WOS_Design_Arquitetura_Planejamento_Atualizado.md) and [accepted ADRs](adr/) retain the detailed source requirements and architectural decisions.

## Domain model

| Concept               | Responsibility                                                              |
| --------------------- | --------------------------------------------------------------------------- |
| Tenant             | Isolation and authorization boundary; shown as a Workspace in the browser   |
| Outcome               | A desired, verifiable result with explicit lifecycle and conclusion         |
| Objective             | An intermediate condition, optionally organized in a parent/child hierarchy |
| WorkItem              | Bounded operational work; exposed as a Task in the workspace                |
| SuccessCriterion      | A revisioned condition to assess before an applicable conclusion            |
| Assessment            | An authorized evaluation of a particular criterion revision                 |
| Roadmap               | A versioned plan referencing live Objectives and Tasks                      |
| Issue / Blocker       | A recorded problem and its separately controlled blocking impact            |
| Evidence / Artifact   | An observation or source, and a deliverable or reference                    |
| Decision / Conclusion | An explicit choice, and the recorded basis for accepting a result           |

Task completion does not automatically achieve an Objective or Outcome. Evidence does not automatically assess a criterion. Publishing a Roadmap freezes a revision; activating it is a separate operation. Plan order does not create operational dependencies.

## Packages and interfaces

`packages/wos-api` owns server composition, HTTP, MCP, and the embedded web workspace. It depends on the public `packages/wos-core` packages. Core application services orchestrate domain behavior through ports implemented by storage adapters. Domain does not depend on transports, SQL, agent runtimes, or a particular consumer application.

SQLite and PostgreSQL implement the same functional storage contracts. The system stores relational current state, immutable Domain Events, and a durable outbox; it is not full Event Sourcing. The remote Go SDK and native CLI use public service contracts.

HTTP and MCP invoke the same application services, authorization, idempotency, and domain rules. The browser and native client provide different interaction flows over that state.

## Tenant placement and state ownership

A Tenant is the complete customer data and authorization boundary. PostgreSQL stores each Tenant in its own schema, including domain state, identities, credentials, public issuer history, immutable signed documents, receipts and delivery queues. A runtime and its connection pool bind to one catalog-resolved schema for their lifetime. Requests never choose a SQL schema through a payload or change `search_path` on a shared connection.

A small installation can serve one Tenant. A shared installation can route approved hostnames to independent Tenant runtimes; replicas of a Tenant use the same schema and issuer identity. A dedicated installation uses the same implementation against a dedicated database. Schemas share database CPU, memory and I/O; resource isolation requires appropriate infrastructure and capacity budgets.

The PostgreSQL catalog records Tenant placement and lifecycle outside the data schemas. Administrative storage operations provision, suspend, resume and delete Tenants with explicit version checks and audit records. Suspension preserves data. Deletion requires stopped runtimes and a suspended or incomplete provisioning state; the catalog retains a tombstone. External object storage, backup retention and billing remain operator responsibilities.

Live contracts use `tenant_id` and signed protocol 3. The ID records provenance and authenticates scope within the selected Tenant; it does not partition multiple customers' rows in a shared domain schema. Historical protocol-2 signed bytes retain their original fields and signatures for verification. See [Tenant operation and cutover](scaling/tenants.md).

## Authority and concurrency

Mutable aggregates use expected versions. Cross-entity invariants are coordinated through the owning Outcome. Operational dependency graphs remain acyclic. Work acquisition is explicit, bounded by its protocol, and subject to expiry and fencing; a stale holder cannot conclude current work.

Tenant, authenticated Principal, declared ActorRef, and external addressing context have distinct roles. Metadata supplied by a caller does not grant authority. Signed execution and independent review bind authorization to the applicable identities, contract, submission, and criterion revisions.

Domain mutations validate scope, coordinate the transaction, validate current authority and invariants, persist state/events/outbox changes, and commit the idempotent result atomically. External HTTP calls, agent execution, and artifact retrieval do not occur inside that transaction.

## Central CLI and durable execution activity

The CLI keeps a private installation independent of the project directory, with isolated destination/identity profiles and one selector captured per invocation. Explicit legacy workspaces remain a compatibility path; migration preserves authenticated bytes and original pending requests. See [ADR 035](adr/0035-central-cli-installation.md).

Authorized signed execution holders can append typed activity and documentary records atomically. The server assigns chronology and keeps history after local cleanup; this does not change issued instructions, authorize more work, or prove external execution. Current attention is a read projection of criteria, dependencies, blockers and correction requirements, not a second mutable pending checklist. HTTP, MCP and SDK use the same application service and bounded revision-aware reads. See [ADR 036](adr/0036-durable-execution-activity.md) and [activity contracts](work-activity.md).

## Continuation and integration

Continuity reads, graph projections, and cursors declare scope, revision, and omissions. A bounded response is not a promise of bounded work over every possible history. Interrupted mutations are reconciled using their original intent and receipts; retries do not silently change expected versions.

Triggers emit durable integration signals. Optional webhook delivery has at-least-once semantics, requiring consumer verification and deduplication. Neither triggers nor the server launch agents or execute tools.

See [public contracts](contracts.md), [operations](operations.md), [signed profiles](signed-workspace-profiles.md), and [client qualification](client-qualification.md) for operational details and limits.

## Licensing boundary

Current first-party source uses the WOS Sustainable Use License 1.0, with production and commercial application infrastructure permitted and commercial platform supply excluded. Software permissions, brand rights, upstream licenses and earlier Apache grants remain distinct. [ADR 037](adr/0037-source-available-licensing-and-brand-boundaries.md) records the owner's licensing decision; [the licensing guide](licensing.md) explains it. This supersedes the original specification's Open Source licensing direction without changing its domain or runtime boundaries.
