---
name: wos-continuity
description: Recover a bounded WOS task using immutable contract specs, latest durable checkpoints and exact receipts instead of previous chat history.
---

# WOS continuity

Persisted WOS state, not a previous conversation or a local YAML file, determines work and authority. Start a new session with authorized scope/task/contract IDs and the minimum task intention.

1. Read `wos_get_work_contract` and its immutable spec through `wos_get_work_contract_spec` only when not already verified locally. Read the latest checkpoint/submission and live execution_allowed/reasons. Expand history with bounded pages only for a specific unresolved question.
2. Check evaluated_at, effective status, exact string fencing, execution ID, content/lease versions, truncated/omitted and next_cursor. An omitted record is not absent. Never execute using an expired local timestamp cache.
3. Locate durable repository/commit/artifact references. A dirty working tree, missing upload or unknown test result must remain explicit; a checkpoint does not transfer files or certify delivery.
4. Recover uncertain commits by `wos_get_command_receipt` or exact replay of the original key/payload/destination. A missing retained receipt does not prove failure. Reconcile entities/history before creating another intent; never automatically reacquire another task.
5. Same-holder takeover is explicit and rotates execution/fencing; it does not revive expiry. A new valid contract after expiry preserves the WorkItem ID and in_progress lifecycle but has a new spec/authority. Stop old execution effects.
6. Return a small resume packet: current obligation, durable progress/proof IDs, authority, blocker, next action and links. Avoid transcript dumps and full Outcome snapshots when a contract suffices.

For local files use wos-workspace and `wosctl work recover/status/diff`; refresh preserves edited result/checkpoint drafts. Read [protocol and recovery](references/protocol.md) when resolving transport ambiguity. Do not invent a release or a fourth terminal contract cause.
