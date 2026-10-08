# Contract operations

WOS coordinates durable state and work authority. Agents and humans choose work;
their host executes it. WOS never launches subagents, interprets YAML as a program,
runs an LLM or locks a Git branch/filesystem.

A WorkContract freezes the minimal executable specification and owns one renewable
lease on one WorkItem. Only completion, expiry or explicit administrative revocation
ends it. Losing authority preserves the WorkItem ID and recoverable in_progress state.
Lease CAS is separate from progress CAS; fencing is an exact unsigned decimal string.

For a new agent session, pass authorized Outcome/WorkItem/contract IDs plus essential
intent. Read the frozen spec once, then latest checkpoint, live authority/impediments
and only necessary proof references. Expand bounded history when required; observe
omissions and cursors. This reduces repeated context transfer but does not itself
measure or guarantee lower total token use. Native context isolation and shared-file
coordination belong to Codex, Claude, Hermes, OpenClaw or the other host.

Acquire explicitly → execute externally → checkpoint → submit immutable material →
review exact submission → finalize. Evidence is observation; assessment is acceptance.
A revised submission requires new proof binding. Work completion never automatically
achieves an Objective/Outcome. Pending review may retain explicitly renewed authority;
there is no voluntary holder release. Same-holder takeover rotates execution/fencing
without extending TTL; expiry requires a new eligible acquisition.

`wosctl` offers this flow through restricted YAML and a destination-bound write-ahead
journal. Credentials remain environment references. Refresh/recovery preserve edited
local drafts. Lost replies and post-commit disk failure reconcile the original receipt
or exact payload/key before another intent. A missing retained receipt does not prove
that a transaction failed. Registered local keys identify immutable material and cannot
silently be reused after edits. Refer to [the client guide](../packages/wos-cli/README.md)
and generated schemas for exact examples and exit codes.

The human interface shows contract authority, material progress, submission and review
in WorkItem details. Contract history loads on demand. Revocation requires an explicit
contract/reason and the server's administrative permission. Review uses submission-bound
assessments and the same independent reviewer policy as HTTP/MCP/CLI. A UI button does
not grant authorization.

Benefits: durable handoff without transcript transfer, bounded task authority, exact
replay after uncertain commits, no accidental progress-as-completion, portable instructions.
Tradeoffs: lease renewal/review latency, explicit recovery and conflict handling, host-owned
filesystem and external-effect coordination, deployment-controlled protocol migration.
Improve by measuring contract/context sizes and consumer costs, provisioning distinct
Principals for independent workers, and supervising foreground renewal at the host.

Linux race/protocol/recovery tests are available. Actual Windows crash/lock/reparse and
consumer execution gates must pass on their target environments; compilation and npm
skill installation are not substitutes. Registry publication is reported only after
published artifact identities/checksums are verified.

## Namespace activation

Read `GET /api/v1/namespaces/{namespace_id}/work-protocol` (or `wos_get_namespace_work_protocol`, `wosctl protocol get`). An absent metadata row means legacy, protocol version 1. Submit `set_namespace_work_protocol` with an existing Outcome `scope`, `expected_protocol_version: 1`, `phase: draining`, and a reason. Stop old server processes and drain their requests; withdraw their database credentials. Wait for existing leases to expire or finish under draining. Submit the next version with `phase: contracts_v1`, `writers_drained: true`. A valid legacy lease prevents activation. Retrying the original idempotency key returns the original receipt rather than executing a second migration.

`wosctl protocol set --file protocol.yaml` journals this operation like other commands. Its YAML file is the typed command mapping. `wosctl protocol reconcile --file reconcile.yaml` performs an explicitly requested bounded expiry scan; pass `after` from `next_after` until `search_complete` is true. Reconciliation preserves work identity and lifecycle. There is no automatic release, new acquisition, agent launch or background keepalive.

Protocol policy updates require an increasing `lease_policy.revision`; renewal evaluates the current policy. Cutover and policy changes require Namespace administration permission. Existing grants are not silently expanded; grant `work.contract.acquire`/`work.contract.revoke` explicitly where needed. New bootstrap administrator grants include both. See ADR-020 for deployment constraints.
