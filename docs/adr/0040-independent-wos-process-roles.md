# ADR 040 — Independent WOS process roles and bounded maintenance

Status: accepted for the owner-requested scalability implementation; executed acceptance and remaining qualification are recorded in ROADMAP.

## Context

The standalone server previously hosted every enabled transport and the optional delivery pool in one process. PostgreSQL supports durable fenced delivery claims across processes, but the composition root could not run a delivery worker independently. Replicating API instances also replicates their connection pools and optional workers. An installation must grow without adding a separate domain implementation or an agent execution runtime.

## Decision

The existing `wos server` command and image support `complete`, `api` and `worker` process roles, selected by `WOS_PROCESS_ROLE` or an explicit `--role` flag. `complete` is the backward-compatible default and retains the optional `WOS_DELIVERY_WORKER_ENABLED` setting. `api` exposes the current product transports and rejects a simultaneously enabled delivery pool. `worker` inherently enables delivery and exposes only `/livez`, `/readyz` and authorized `/metrics` on `WOS_LISTEN`.

A separate worker requires PostgreSQL and scoped API authentication. SQLite remains a supported single-process deployment; no shared-file multiwriter topology is introduced. Worker startup checks the installed schema but never migrates, bootstraps a Namespace, installs endpoint metadata or creates an issuer identity. Provision and migrate the installation through the existing administrative deployment procedure before starting workers. Workers use the same durable queue and fenced claim/acknowledgement protocol as complete processes. Endpoint allowlists and secret references must agree with the installation's configuration; workers do not need its bootstrap secret or issuer private key. An optionally configured issuer seed must match the persisted identity.

The operational listener must remain private. Health explicitly describes the answering process. API readiness continues checking storage/schema and signed issuer readiness, independently of receiver availability or a failed optional local delivery pool. Worker readiness requires the compatible schema and a running local pool; it does not claim that all queues or remote workers are healthy. Durable scoped queue health remains an authenticated Application query through an API process.

A fatal pool exit terminates a worker-only process for an external supervisor to restart. In complete mode the healthy API remains available and local worker failure is visible, preserving ADR 031. Shutdown stops acquisitions, cancels bounded delivery requests and waits for independent bounded acknowledgement/lease recovery. It does not revoke external WorkItem authority or guarantee that a receiver did not observe an interrupted request. Consumers still deduplicate stable Integration Event identities.

Complete and worker roles perform periodic bounded cleanup through an optional storage maintenance port. Each SQL batch removes at most 256 completed expired short-lived idempotency cache entries. A pass continues while batches are full, bounded by five seconds and at most 4,096 batches, then yields for one minute. Partial batches yield immediately so idle installations do not scan repeatedly; busy installations can drain accumulated expiry without holding a large delete transaction. Storage coordinates concurrent passes with transactional bounded selection; no global leader is required. Durable signed operation results, closed intent receipts, proofs, active operations and security records are excluded. API-only roles perform no background maintenance.

Connection pools remain per process. Read-only deployment diagnostics can compare a declared process count times the configured pool plus reserved connections against an operator-declared database budget. This conservative estimate uses the largest pool for mixed process roles and neither discovers replicas nor certifies database capacity. Operators must account for every API, worker, migration/backup client and other database consumer before autoscaling.

## Consequences and verification

The same product can run as one process or independently scaled API and worker pools. No broker, new microservice, scheduler, sticky sessions or private Cloud dependency is required. Supervisors, load balancing and database high availability remain deployment responsibilities. Role tests cover unsafe composition refusal, worker refusal to initialize storage, absence of business routes, running-pool readiness, fatal exit and cancellation. PostgreSQL acceptance is exercised against a real isolated schema; SQLite tests cannot qualify the separate-worker topology.
