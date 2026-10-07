---
name: wos-coordination
description: Coordinate a bounded task through WOS with focal context, explicit claims, evidence and safe retries. Use when an authorized WOS Outcome or WorkItem is part of the assignment.
---

# WOS coordination

WOS stores state; you choose and execute work using your runtime tools. An installed skill neither connects MCP nor grants permission. Obtain the authorized Namespace and Outcome from the assignment or bounded discovery. Never invent existing IDs or derive authority from metadata.

1. If a WorkItem is assigned, read `wos_get_work_context` directly. Otherwise use `wos_list_ready_work` with a small limit such as 5 and choose only within the assignment. Use `wos_get_continuity` with a small section limit when broader orientation is necessary, not on every iteration.
2. Read the task, linked Objective/criteria, operational state, dependencies, decisions and proof references. Expand only what execution requires. Check `truncated`, `omitted`, cursors, `evaluated_at` and `outcome_revision`; omitted data is not absence. Focal context currently may include accepted decisions across the Outcome.
3. Acquire `wos_claim_work_item` with the current expected version, explicit TTL and stable idempotency key. Save the returned claim ID, fencing token, version and expiry. A readiness listing is not an execution grant; the claim command revalidates it.
4. Execute only the claimed task outside WOS. Use relevant source excerpts and bounded test output. Renew through `wos_renew_work_item_lease` before expiry, using current returned version/claim/fencing. Stop if authority or lease is lost; do not continue external side effects on a stale claim.
5. Register the material deliverable with `wos_register_artifact` and observations/test proof with `wos_register_evidence`; link evidence using `wos_create_evidence_link` where applicable. References must be accessible to intended reviewers. Keep credentials and full logs outside documentary text.
6. Complete explicitly through `wos_complete_work_item` with current expected version, claim ID, fencing token, concise result summary and reason. Required criteria must have valid current assessments before completion. Work completion does not achieve Objective/Outcome automatically.
7. If essential input is missing or an impediment exists, register an Issue and an explicit Blocker on the relevant target when appropriate, then stop/release according to the live command contract. Do not invent a free-form lifecycle `blocked` or silently certify success.

Discover live argument schemas through MCP tools/list or HTTP `/api/v1/commands`. Tool names may carry a client-side prefix. Mutation envelopes contain `idempotency_key` and `command`; do not guess TTL units, DTO names or a new version. Read [protocol and recovery](references/protocol.md) only when connecting, mutating or recovering from errors.

End with a short receipt: task/Outcome IDs, persisted result, artifact/proof IDs, current version/lease disposition, remaining issue and next relevant reference. Use WOS persisted state, not a previous chat, as continuity authority.
