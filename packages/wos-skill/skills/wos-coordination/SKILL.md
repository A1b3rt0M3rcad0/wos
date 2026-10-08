---
name: wos-coordination
description: Coordinate bounded WOS contract work with focal snapshots, explicit authority, checkpoints and material-bound delivery. Use for authorized WOS tasks.
---

# WOS coordination

WOS persists state and coordinates authority. Your host executes work and controls tools. Installing this skill grants no access and launches no agents. Obtain the authorized Namespace/Outcome from the assignment; do not infer authorization from IDs, metadata or ActorRef.

1. Check live capabilities and Namespace protocol. For an assigned task, fetch focal work context; otherwise use bounded available-work pages within one explicit Outcome. Omitted data is not absence. Do not read a whole Outcome on every iteration.
2. Acquire with `wos_acquire_work_contract` or bounded `wos_acquire_next_work_contract`, exact expected version and a durable idempotency key. Save the returned immutable spec/digest, contract/execution IDs, string fencing token, content/lease versions and server expiry. A candidate listing does not grant execution.
3. Execute only the contracted obligation outside WOS. Open referenced artifacts only when necessary. Consult live authority and impediments before dependent effects. Renew with `wos_renew_work_contract` before expiry; renewal advances lease_version, not contract.version.
4. Persist material progress with `wos_sync_work_contract`. Checkpoints do not complete work or release authority. Include durable repository/commit/artifact references and mark dirty/unknown workspace facts truthfully; another agent does not inherit uncommitted files from a checkpoint.
5. Submit immutable material with `wos_submit_work_result`. Resolve local keys to canonical documentary IDs, identify exact artifact revisions/checksums, and explicitly supersede the latest submission when changing delivery. Evidence or passing tests are observations, not assessments.
6. Required assessments must bind the exact submission ID/digest and criterion revision. Respect independent reviewer policy. Finalize through `wos_finalize_work_contract`; completion remains separate from Objective/Outcome achievement. Pending review retains authority until completion, expiry or explicit administrative revocation.
7. Missing input or blockers require an explicit Issue/Blocker and stopping dependent effects. You may explicitly maintain renewal while waiting or stop renewal and allow expiry. There is no holder release/unlock command. Revocation uses authorized `wos_revoke_work_contract` with contract ID and reason.

Stop on expiry, revocation, stale execution or binding conflict. Same-holder takeover requires explicit `wos_resume_work_contract`, CAS and new execution/fencing; never treat refresh as takeover. After timeout replay the identical stored payload/key or recover its receipt. Do not change expected versions to force a retry.

Read [protocol and recovery](references/protocol.md) for envelopes, bounded expansion and uncertain commits. Use wos-workspace for the API-only wosctl/YAML workflow. End with compact persisted IDs, result/proof, authority disposition, remaining obligation and next relevant reference.
