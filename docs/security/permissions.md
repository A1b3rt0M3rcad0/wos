# Permission catalog

WOS authorization uses explicit permission values. A grant and a credential's
`permitted_operations` are both exact lists; the effective permission is their
intersection with scope, protocol authority, and domain preconditions. An empty
list grants no operations. Wildcards are not supported.

The catalog is defined once in `packages/wos-core/domain/permission_catalog.go`.
`ports.Permission.Valid()` and `ports.AllPermissions()` derive from that source.
The web grant editor centralizes English labels and short descriptions in
`packages/wos-api/web/assets/actions.js`, with display copy in `en-US.js`.

## Granular permissions

| Area | Permission values | Scope |
| --- | --- | --- |
| Reading | `state:read`, `credentials:read`, `audit:read` | Read application state, credential metadata, security audit, or operational diagnostics. |
| Objectives | `objective:create`, `objective:update`, `objective:start`, `objective:conclude`, `objective:reopen`, `objective:cancel`, `objective:configure_completion` | Create, define, start, conclude, reopen, cancel, or configure completion for Objectives. |
| Task planning | `task:create`, `task:update_definition`, `task:activate`, `task:defer`, `task:assign` | Create and prepare Tasks, change assignees, or move between planning operations where domain rules allow. Assignment does not grant execution authority. |
| Task execution | `task:claim`, `task:renew`, `task:release`, `task:deliver` | Acquire, renew, release, or deliver work through the existing execution protocol. |
| Task review | `task:review`, `task:accept`, `task:request_changes`, `task:withdraw_delivery` | Participate in review, accept a valid delivery, request correction, or withdraw a delivery where supported. Assessment and signed protocol checks remain separate. |
| Task control | `task:cancel`, `task:reopen`, `task:intervene` | Cancel or reopen a Task, or perform an explicitly audited intervention. |
| Criteria and records | `criterion:define`, `criterion:assess`, `criterion:waive`, `records:write` | Define criteria, record assessments, waive criteria explicitly, or add documentary records, Evidence, and Artifacts. Evidence does not assess a criterion. |
| Dependencies | `dependency:write` | Add, remove, or apply dependency relations. |
| Issues | `issue:create`, `issue:update`, `issue:assign`, `issue:investigate`, `issue:resolve`, `issue:reopen` | Report and manage Issues, including assignment, investigation, resolution, and reopening. |
| Blockers | `blocker:create`, `blocker:release`, `blocker:cancel` | Create, release, or cancel explicit Blockers. Issue resolution does not release a Blocker by itself. |
| Credential and policy administration | `credentials:issue`, `credentials:update`, `credentials:revoke`, `credentials:rotate`, `permissions:manage`, `policy:manage` | Issue or manage credential authority, grants, and operational or protocol policies. |

## Existing permission values

These published values remain recognized as exact catalog entries so stored
grants and policies can still be read without rewriting their data:

`work.contract.return`, `work.contract.complete_direct`,
`work.review.acquire`, `work.review.decide`, `identity.signing_key.enroll`,
`identity.signing_key.rotate`, `identity.signing_key.revoke`,
`work.contract.acquire`, `work.contract.revoke`, `state:read`,
`outcome:write`, `planning:write`, `work:write`, `records:write`,
`assessment:write`, `conclusion:write`, `namespace:admin`,
`integration:write`, `actor:delegate`, `work:admin_cancel`,
`work:admin_complete`, and `assessment:waive`.

This step does not rewrite or expand any existing grant or credential policy.
The versioned, finite compatibility expansion is a later rollout requirement
tracked by E17. Complete that migration before deploying granular command
requirements to an installation whose existing credentials rely on broad
legacy permissions. Never treat an old permission as a wildcard, and do not
give new executor credentials administrative, assessment, or review authority
by implication.

Permission lists are bounded to 128 entries in application validators and API
schemas. Entries must be known and unique. The catalog currently contains
fewer than 128 distinct values, so the cap is an upper bound rather than a
promise that every list can contain 128 authorized values.
