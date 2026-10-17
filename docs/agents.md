# Operate with agents

An agent acquires one WorkItem, performs the work in its own runtime, records what happened and the actual proof, and returns a delivery for the required review. A new session can continue from recorded state instead of reconstructing the conversation. WOS coordinates authority and persists shared state; it does not start agents or execute their tools.

> [!NOTE]
> This guide describes the centralized workflow in the current `master` source. Published `wosctl` v0.4.2 does not contain these new commands. Use a binary built from current source and check `wos version --output json` / `wos --help`. Do not assume installing `@latest` installs source behavior. See [source development](getting-started.md#build-from-source).

## 1. Install instructions in the external runtime

```sh
wos skill install --agent codex
wos skill list --agent codex
wos skill doctor --agent codex
```

Use `claude`, `hermes`, `openclaw`, or `--agent all` for their discovery layouts. `--agent generic --dir /absolute/path/to/skills` selects an explicit discovery root. `--project`, `--global`, and `--dry-run` control instruction installation, not WOS operational storage.

| Skill              | What it teaches                                                             |
| ------------------ | --------------------------------------------------------------------------- |
| `wos-workspace`    | Installation data, approved Profiles and typed CLI operation                |
| `wos-coordination` | Bounded discovery, WorkItem acquisition, progress and delivery              |
| `wos-delegation`   | Auxiliary helpers versus independently authorized workers                   |
| `wos-continuity`   | Current context, bounded durable history and interrupted-operation recovery |
| `wos-review`       | Independent evaluation of exact material and criterion revisions            |

Installation is offline and preserves user-modified instruction files. It grants no credential, server permission or runtime tool access. Configure the external runtime separately and reload its Skill discovery as needed. Official agent operation goes through the CLI; HTTP/MCP/SDK remain supported integration adapters for other authorized clients.

## 2. Initialize once and select an approved Profile

```sh
wos init
wos profile list
wos profile use executor
wos --profile executor profile show
wos --profile executor auth status --output json
wos --profile executor doctor --output json
```

An operator must first provision the Profile's trusted server origin/ServerID, Namespace, Principal, credential, registered signing key and issuer pin. The server must enable the appropriate signed protocol and grant the required operations. Follow the [central CLI provisioning guide](../packages/wos-cli/README.md#profiles-and-protected-onboarding) and [service operations](operations.md#shared-service). Creating a Profile or installing a Skill does not activate a Namespace or grant access.

Operational data belongs to one installation directory and commands work from any directory. No per-project `.wos/` is required. A Namespace is the authorization boundary called Workspace in the browser.

Selection resolves once per invocation: `--profile` overrides `WOS_PROFILE`, which overrides the persisted active Profile; a sole Profile can be selected if no default exists. `profile use` is convenient interactively. Concurrent agent processes must pin `--profile NAME` or process-scoped `WOS_PROFILE` so another process changing the active default cannot change their identity.

The CLI is the agent's only operational interface. Agents never open, interpret, edit or delete internal Profile/contract documents. The CLI owns protected storage, validation, local CAS, signing, request identity and recovery.

## 3. Acquire one WorkItem and read its obligation

```sh
wos --profile executor outcome list --limit 10 --output json
wos --profile executor work available --outcome OUTCOME_REF --limit 10 --output json
wos --profile executor work next --outcome OUTCOME_REF
wos --profile executor work context --for-agent --output json
wos --profile executor work status --for-agent --output json
```

Omit `--outcome` when an approved default or unambiguous selection applies. Use explicit `work checkout WORK_REF` to select a unique available title/ID prefix; exact UUIDs and observed `--version` remain supported. Listing candidates does not grant authority. Incomplete bounded search is not proof that no eligible work exists; honor cursors and completeness information.

`work list` also rediscovers bounded server execution headers after local cleanup. Filter by persisted `--status` and use `--holder PRINCIPAL_ID` only for authorized coordinator reads; `effective_status` also reflects current expiry. Delivered execution does not imply Task completion.

The contract is for one WorkItem. Outcome and Objective supply context, not additional execution authority. Read the immutable instructions, constraints, expected material and current success criteria. Omit a contract argument only if exactly one applicable local contract exists; otherwise use an explicit contract reference from `work list`.

Context includes bounded current coordination state, staged result/review material and summaries of the five newest WorkItem activities. Descriptions, captured version metadata and reference collections declare omissions; expand complete records through `work history`. Preserve every omission and each resource's revision/evaluation time; the separate resources are not one combined snapshot. Follow `history_follow_up` with its exact scope, `--order desc` and cursor when older activity is needed. The issued obligation remains complete even when descriptive live context is shortened.

Execute the obligation outside WOS. Consult live authority and blockers before dependent effects. A contract does not lock repository files or make an external deployment idempotent.

## 4. Record progress and actual proof

```sh
wos --profile executor work log add --type progress --description "Implemented the agreed change"
wos --profile executor work log add --type problem --description "Observed a failing expiry case"
wos --profile executor work log add --type correction --description "Corrected the expiry handling"
wos --profile executor work artifact add --name "Change set" --uri ACTUAL_IMMUTABLE_URI --artifact-type code --source-version ACTUAL_COMMIT
wos --profile executor work evidence add --type test_result --description "Describe the checks actually run and their result" --uri ACTUAL_REPORT_URI --criterion ORIGINAL_CRITERION_REF
wos --profile executor work evidence list --limit 10 --output json
wos --profile executor work result set --summary "Describe the actual delivered result"
```

Replace placeholders with real references. Activity types are `progress`, `discovery`, `problem`, `correction`, `validation`, and `result`. Execution log/evidence/artifact additions register durably on the Server; an uncertain response remains an exact recoverable intent. Do not retry it as a new append.

An activity is a participant's declaration. Evidence records an observation/source. An artifact identifies exact material. CriterionAssessment evaluates an exact criterion revision and submission. A passing test, signed record or evidence link does not approve a criterion. Source versions/checksums must identify material actually produced; WOS does not certify the truth of an external claim.

Use optional `work progress set --summary TEXT --next-action TEXT` for advisory progress. Delivery still requires `work result set --summary TEXT`. Context/status/check derive known obligations, blockers and review findings; do not maintain a redundant manual pending list.

## 5. Check and return the delivery

```sh
wos --profile executor work status --for-agent --output json
wos --profile executor work check --output json
wos --profile executor work finish --output json
```

Check is a preflight that may become stale; the Server decides acceptance under current authorization, CAS, policy and authority. Finish freezes, signs, sends and reconciles the original delivery. It preserves uncertain material and cleans local state only after verified signed closed acceptance.

An accepted execution can close the executor obligation while the WorkItem awaits independent review. WorkItem completion does not automatically achieve its Objective or Outcome.

## 6. Review from an independent Principal

```sh
wos --profile reviewer review checkout --next
wos --profile reviewer review context --for-agent --output json
```

Inspect the exact submission, immutable artifact revisions/checksums, assigned criteria and necessary evidence. Perform actual checks in the reviewer's external runtime. Different Profile names, keys or subagents for one Principal do not establish independence.

For approval, set the decision first, then explicitly assess each required original criterion:

```sh
wos --profile reviewer review decision set --decision approved --reason "Describe the independent checks"
wos --profile reviewer review assessment add --criterion ORIGINAL_CRITERION_REF --result met --rationale "Describe actual supporting observations"
wos --profile reviewer review check
wos --profile reviewer review finish
```

Use `--evidence UUID,...` only for registered proof. Review artifact/evidence additions are locally staged until accepted signed finish and return `local_key` values; assessments can reference their returned `local:LOCAL_KEY` values. Complementary reviewer records require a source version/checksum present in the accepted executor artifact set.

Before signing, use `review assessment list`, `review finding list`, and `review evidence/artifact list` for bounded staged material. Correct an assessment with `review assessment set --criterion REF --result met --rationale TEXT`; explicitly remove it with `review assessment remove --criterion REF`. Use `review finding set --finding REF --description TEXT` or `review finding remove --finding REF` for draft findings. A decision change never silently drops assessments. Remove a staged observation with `review evidence remove --evidence local:LOCAL_KEY` only after changing/removing every referencing draft assessment. Artifact removal uses `review artifact remove --name LOCAL_KEY`. These edits never rewrite accepted history and stop when a return is frozen. Local list cursors become stale after a draft edit; start a fresh list then.

For non-approval, set the decision/reason first, then record factual original-obligation findings and omit criterion assessments:

```sh
wos --profile reviewer review decision set --decision changes_requested --reason "Explain the required correction"
wos --profile reviewer review finding add --criterion ORIGINAL_CRITERION_REF --description "Describe the unmet obligation"
wos --profile reviewer review check
wos --profile reviewer review finish
```

A finding targets one criterion or one `--requirement`, never both. A changed submission requires a new exact-material review. Corrections explicitly acknowledge the original review case and every frozen finding; generic `work next` does not silently acquire them. See the installed Skill's correction reference and [signed return protocol](signed-cli-returns.md) for deeper authority constraints.

Inspect correction responses with `work correction list`. Before signing, correct a draft response with `work correction set --finding ORIGINAL_REF --summary TEXT`, or explicitly remove it with `work correction remove --finding ORIGINAL_REF`. Use context's `draft_digest` with `--expected-draft DIGEST` when writing from an earlier read.

## 7. Continue after interruption

```sh
wos --profile executor work recover --all-pending --output json
wos --profile executor work context --for-agent --output json
wos --profile executor work history --limit 10 --output json
wos --profile reviewer review recover --all-pending --output json
```

Read context/status first; request only the history needed for a specific gap. After accepted local cleanup, use `work history --work WORK_ITEM_UUID --outcome OUTCOME_REF --limit 10` to read durable server activity. That activity query is not a complete review-decision history.

History defaults to ascending chronology. Use `--order desc` for a bounded newest-first page; preserve that order with the returned cursor. Context supplies a matching follow-up for its newest-first page.

An exact historical execution contract can be inspected with `work context CONTRACT_UUID --outcome OUTCOME_REF` after local cleanup. `accepted_result` contains bounded accepted result/proof references verified against the pinned issuer receipt and exact material digest. Reading that result grants no new execution authority.

Recovery checks the original-CID operation/receipt before identical replay. It does not automatically send a prepared-only return, change payload/destination/CAS, acquire replacement work or delete uncertain data. A missing local acquisition is repaired from its original intent. Explicit resolution of a definitively rejected frozen return requires reasoned operator/host approval and verified durable proof; an HTTP error alone is insufficient.

Frozen activity appends do not yet have the final-return terminal-resolution protocol. If expiry or a definitive rejection prevents the original append from being accepted, stop and preserve its exact recovery state for an operator. Do not delete it or switch Profiles claiming it is resolved. See the [documented recovery limit](../packages/wos-cli/README.md#recovery-and-authority).

Expiry, revocation and stale fencing stop authority. Explicit resume rotates fencing without reviving expiry; refresh does not take over. Foreground keepalive runs only when explicitly launched and supervised by the host. See [CLI recovery](../packages/wos-cli/README.md#recovery-and-authority).

## Multiple agents and auxiliary helpers

An auxiliary subagent helps one holder execute its existing obligation; the holder retains responsibility and validates the result. Independent workers acquire different WorkItems under their own approved pinned Profiles/Principals. Explicit `depends_on` relations coordinate execution dependencies; parent grouping or plan ordering does not create them.

The external runtime controls delegation, tools, history and filesystem isolation. Give workers bounded intent, references, acceptance and stop conditions. Collect compact durable outputs rather than whole transcripts. An authorized coordinator can inspect scoped status/activity pages; it cannot acquire access merely by knowing another contract ID.

## Further reading

- [Central workflow plan](central-agent-workflow/README.md): architecture, migration, acceptance and the original owner briefs.
- [CLI reference](../packages/wos-cli/README.md): commands, Profiles, installation paths and compatibility.
- [Human workspace](workspace-ux.md) and [screenshots](design/editorial/README.md#main-pages).
- [HTTP](http.md), [MCP contracts](contracts.md) and the [Go client](../examples/remote-client): equivalent authorized integration contracts.
- [Client qualification](client-qualification.md), [ROADMAP](../ROADMAP.md) and [AGENTS](../AGENTS.md): verification limits and implementation status.
