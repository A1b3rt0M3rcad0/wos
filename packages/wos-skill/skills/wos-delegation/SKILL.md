---
name: wos-delegation
description: Coordinate auxiliary subagents and independent WOS workers through pinned CLI Profiles and bounded WorkItem contracts in an authorized external runtime.
---

# WOS delegation

WOS does not create, run or schedule agents. Verify delegation is authorized and the external runtime exposes the needed tools. Skill installation configures instructions only; it grants no server identity or host permissions.

## Choose the authority mode

- **Auxiliary subagents:** collaborate on one holder's existing WorkItem obligation. The holder retains responsibility, checks results and records/returns through its own authorized CLI. A helper label does not transfer the contract or establish independent review.
- **Independent workers:** each acquires its own authorized contract for a distinct WorkItem. Each process pins an approved Profile/Principal with `--profile NAME` or process-scoped `WOS_PROFILE`. Credentials/keys under one Principal remain subject to that Principal's shared quotas and independence policy.

## Delegate bounded work

1. Define responsibilities and actual `depends_on` relationships within the authorized Outcome. Parent grouping and plan order do not create operational dependencies.
2. Supply only necessary intent, Outcome/WorkItem/contract reference, permitted files/tools, acceptance and stop conditions. Prefer fresh/minimal history where the host supports it; do not send the entire supervisor conversation by default.
3. Independent workers use `wos --profile worker-a work next --outcome REF` or explicit checkout, then `work context --for-agent --output json`. Auxiliary helpers receive the bounded instructions necessary to help; they do not acquire a second authority for the same WorkItem.
4. Use the CLI as the only WOS operational interface. Workers never open/interpret/edit internal contracts or Profile documents. They use typed activity/material commands, current context/status and explicit check/finish; UUIDs are optional exact references, not access grants.
5. Do not switch the global active Profile during concurrent work. `profile use` is interactive; pin each worker process and use an explicit contract reference when it holds more than one obligation.
6. Coordinate files/branches/deployments in the external host. WOS contracts protect work authority, not filesystem isolation. Native subagent history/worktree/tool isolation must be checked in that runtime.
7. Collect compact durable outputs: contract/result IDs, actual changed paths, evidence/artifact revisions, unresolved facts and next action. Read authorized bounded status pages rather than full transcripts or history dumps.
8. On interruption reconcile the original pinned Profile via `work recover --all-pending`; do not reacquire to repair uncertainty. Expired/revoked/stale authority stops effects. Preserve CAS/fencing and original signed bytes.

Independent review uses a genuinely distinct authorized Principal and a separate exact-material review obligation. Different names, keys, windows or subagents for one Principal do not count. A valid signature or executor return is not proof of quality or WorkItem completion.

Read [runtime differences](references/runtimes.md) only for host setup and [protected signing/recovery](references/signed-protocol.md) for authority safety. Use [correction](references/correction.md) only for an explicit original review finding. Measure total usage before claiming context/token savings; successful installation does not certify every agent runtime.
