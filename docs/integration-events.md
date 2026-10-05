# Public integration event catalogue v1

The machine-readable catalogue is [integration-events-v1.json](integration-events-v1.json). It enumerates every event type projected into `IntegrationFact` by the current Application command mapping. The envelope uses `schema_version: 1`; a Trigger produces a separate `TriggerFiring` with the configured product `signal_type` and the source fact. Administrative credential/grant/session audit is a separate Namespace contract.

Each fact carries a stable source ID, Namespace, Outcome, Outcome revision, event index, entity reference, Principal, declared Actor, command ID, timestamp and optional correlation ID. It has no internal DomainEvent payload. Consumers use authorized continuity queries to obtain the current details. Facts describe committed changes, not instructions to execute work.

| Event family | Public meaning |
| --- | --- |
| `outcome.*`, `objective.*` | Explicit lifecycle or definition changes. `achieved` and `conclusion_recorded` describe explicit certification with preserved proof, not inferred completion from work. |
| `work_item.*` | Work definition, availability, claim/lease or lifecycle change. Completion does not certify its Objective or Outcome. |
| `*.criterion_*`, `*.assessment_recorded` | Definition revision, retirement or explicit evaluation on the referenced owner. Assessment does not imply certification. |
| `relation.*` | Dependency added or removed; readiness must be queried after the change. |
| `issue.*`, `blocker.*` | Independent Issue/Blocker state changes. Resolving an Issue does not release its Blocker. |
| `artifact.*`, `evidence.*` | Documentary registration, withdrawal, retraction or evidence-link change. Retraction may contest a preserved conclusion; consumers query the contestation projection. |
| `decision.*` | Proposal, edit, acceptance or rejection of a Decision. |
| `roadmap.*` | Draft, immutable published revision, active slot or Roadmap lifecycle change. Published labels remain historical; current references are projected separately. |
| `trigger.*`, `integration_delivery.redelivered` | Trigger configuration/enablement or an authorized explicit redelivery request. These are facts, not an agent scheduler. |

One command can produce multiple facts at the same Outcome revision. For example, normal WorkItem completion emits release, completion and conclusion-recorded facts with distinct IDs and event indexes. Consumers must not assume a single event per revision. A configured Trigger firing has its own stable identity derived from Trigger ID/version and source event ID; each endpoint delivery preserves it across retries and explicit redelivery.

Names and required envelope semantics remain stable within v1. New types and optional metadata are additive changes; consumers should record and safely ignore unsupported types according to their subscription policy. Removing/renaming a type, changing field types, or changing the meaning of an existing type requires a new major schema/profile and an explicit migration. Persisted v1 facts, firings and outbox bodies are not rewritten when the catalogue evolves.

Delivery is at least once. Verify the signature and timestamp, then persist deduplication by `integration_event_id` atomically with consumer side effects. Acknowledge only after that consumer transaction commits. Network retries can repeat delivery even after successful processing. See [operations](operations.md) for signature, lease, retry and retention limits.

The CI check `python3 tools/eventcatalog/check.py` rejects a mapping/catalogue mismatch. The catalogue describes available types; it does not certify the complete failure/delivery acceptance matrix.
