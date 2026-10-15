# ADR 031 — Authoritative delivery health and supervised process state

Status: accepted for the operational hardening candidate; final acceptance is recorded in ROADMAP.

Durable webhook delivery has a different availability boundary from the Core API. A receiver outage must not make `/readyz` certify a different fact. HTTP readiness still checks schema and signing readiness. Delivery is at least once; consumers deduplicate `integration_event_id`.

The optional DeliveryHealth UnitOfWork port aggregates persisted queue status, due/lease ages, last claim, actual historical attempt count and last successful delivery. Its Namespace/optional Outcome predicate runs in one transaction with current credential, parent, actor, grant and policy checks. Namespace aggregates reject Outcome-restricted credentials. It reads neither webhook bodies nor receiver/secret configuration. Missing historical timestamps remain null. A migration adds a retained success timestamp, including after explicit redelivery; it does not reinterpret successful attempts as current success.

HTTP `GET /api/v1/namespaces/{namespace_id}/operations/delivery-health`, MCP `wos_delivery_health` and the Go SDK call the same Application query. Namespace administration is required; a scoped administrator must supply the permitted Outcome. Unsupported adapters fail explicitly rather than assert an empty queue. Local unauthenticated embedding does not provide an administrative credential snapshot.

The host injects a bounded process snapshot. It describes this process only: enabled, running, starts, start/tick timestamps and a classified last failure. It never asserts that another replica is running or the queue is empty. Fatal errors and panic terminate the worker visibly without taking down a healthy API. Cancellation is an intentional stop. Shutdown cancels delivery requests and bounds the wait by configured grace; unacknowledged leases are recoverable by fencing. Aggregated process metrics contain no Namespace, URL, event ID or payload label.

Operational health is healthy, degraded, disabled or unknown. The initial alarm policy is any exhausted delivery or a due item older than five minutes, configurable up to seven days. A stale/missing worker heartbeat is distinct from a successful delivery. External monitoring owns polling, notification and receiver availability objectives. WOS does not schedule agents or guarantee receiver availability.
