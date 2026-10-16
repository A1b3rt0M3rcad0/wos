# Multiagent and auxiliary subagent operation

## Two modes

| Mode                | Authority                                | Required behavior                                                                                                            |
| ------------------- | ---------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| Auxiliary subagents | One holder retains one WorkItem contract | External runtime delegates bounded implementation help; holder validates results and records/returns under its own authority |
| Independent workers | One authorized contract per WorkItem     | Each worker pins its own approved Profile/Principal, acquires work and maintains its own lease/material/return               |

Auxiliary labels do not create new authenticated holders or independent reviewers. Multiple keys/profiles for one Principal share server policy and quotas. A coordinator may observe others only within current granted scope.

## Process isolation contract

```sh
WOS_PROFILE=executor-a wos work next --outcome OUTCOME_REF
WOS_PROFILE=executor-b wos work next --outcome OUTCOME_REF
wos --profile reviewer review checkout --next
```

Each process resolves its Profile once; use an explicit contract reference whenever more than one local obligation applies. Do not use `profile use` as an identity switch inside concurrent automation. Server leases prevent duplicate WorkItem authority; they do not lock Git files, branches, devices or deployments.

## Implementation and integration

1. Central Profile selection must be stable under another process changing the interactive active default. Existing `locks_v2_*` and contract locks retain bounded per-Profile isolation.
2. Preserve `packages/wos-core/application/signed_quotas.go`, signed acquisition/lease services and storage guard behavior. Multiple credentials for one Principal must not bypass its quota.
3. Reuse explicit `depends_on` relations for execution dependencies. Plan order, parent Objective and grouping do not create hidden dependencies.
4. A worker receives only authorized Outcome/WorkItem/contract references, intent, acceptance, permitted files/tools and stop conditions. It reads focal CLI context, not the supervisor's full conversation by default.
5. The host chooses models, history inheritance, permissions and worktree separation. WOS neither promises filesystem isolation nor certifies every runtime because Skill installation succeeded.
6. Coordinator reads aggregate small status pages; request exact detail only for a blocked/review/uncertain item. Never require all workers to load a global transcript.

## Acceptance scenarios

- Two independent workers compete for one eligible item: only one acquires authority; the other receives a normal conflict/no-candidate result.
- Two workers on distinct items progress concurrently without mutating each other's files, bindings or return intentions.
- Change interactive Profile during a pinned worker's foreground command; its identity/destination stays unchanged.
- Same-Principal auxiliary subagent attempts independent review: server rejects it under independent-review policy.
- Stale fencing after explicit takeover cannot append valid work activity, renew old authority or deliver.
- Missing worker resumes from bounded server activity/material and original pending receipt; replacement acquisition is not used to repair transport ambiguity.

Use race/process tests and both SQL stores; multiple goroutines are not evidence of separately deployed hosts. Measure total model usage before claiming context/token savings.
