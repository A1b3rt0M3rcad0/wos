---
name: wos-delegation
description: Delegate authorized bounded WOS contract tasks through native host subagents while keeping supervisor context small. Use only when delegation is authorized and available.
---

# WOS delegation

WOS does not spawn, schedule or run agents. Verify the user's/repository's delegation authorization and the host's native tools before launching workers. If unavailable, execute sequentially or report that limitation.

1. Define independent WorkItems and acceptance obligations within the authorized Outcome. Plan order and parent grouping do not create execution dependencies; explicit depends_on does.
2. Send each worker only authorized scope/task IDs, essential intent, permitted files/tools, acceptance and stop conditions. It acquires its own WorkContract and reads its immutable focal snapshot and latest checkpoint from WOS. Do not send the full supervisor conversation by default.
3. Explicitly select fresh/minimal history if the host supports it. Separate windows can inherit history by default. Neither context nor filesystem isolation is guaranteed by WOS. Do not claim token savings without measurements.
4. Use distinct provisioned Principal credentials for independent holders. Actor aliases and two keys belonging to one Principal do not distinguish holders. A supervisor cannot silently transfer its contract to a worker with another authenticated identity.
5. Coordinate shared files outside WOS. Contracts protect WorkItems, not branches, files or deployments. Use isolated checkouts only when authorized; inspect combined edits before certification.
6. Workers renew only their valid authority, sync checkpoints, submit material and return concise IDs, changed files, test evidence, unresolved facts and next references. The host retains detailed logs outside the supervisor context.
7. On crash, read the last durable checkpoint. Before expiry, explicit same-holder takeover rotates execution/fencing without extending TTL. After expiry or revocation, acquire a new contract when eligible; the old generation cannot finalize. Review the new exact material and do not re-execute done work.

Read [runtime differences](references/runtimes.md) only when configuring delegation. Use wos-coordination for the authority protocol, wos-workspace for local recovery, and wos-review for independent proof. WOS never executes a YAML instruction or grants host tool permissions.
