---
name: wos-workspace
description: Operate WOS through its centralized CLI, approved Profiles, bounded contract context and protected recovery without manipulating internal files.
---

# WOS workspace operation

Use the CLI as the only operational interface. WOS stores coordination state; your external runtime executes work. Skill installation grants no credentials, host permissions or server access. These instructions describe the centralized source-branch CLI; check the installed binary before using new commands.

## Initialize and select

```sh
wos init
wos profile list
wos profile use executor
wos --profile executor doctor
```

An operator must provision the trusted Profile first. Selection precedence is `--profile`, `WOS_PROFILE`, active default, then a sole-Profile fallback. Use the active default interactively; pin `--profile NAME` or process-scoped `WOS_PROFILE` for workers. Operational data is installation-scoped and commands work from any directory. Agent Skill discovery directories are unrelated to WOS operational storage.

## Operate one obligation

```sh
wos --profile executor work available --limit 10 --output json
wos --profile executor work next
wos --profile executor work context --for-agent --output json
# Perform the assigned work in the external runtime.
wos --profile executor work log add --type progress --description "Implemented the agreed change"
wos --profile executor work result set --summary "Describe the actual delivery"
wos --profile executor work status --for-agent --output json
wos --profile executor work check
wos --profile executor work finish
```

An omitted contract reference works only with one applicable local contract; if ambiguous, select explicitly from `work list`. Use `--outcome REF` when needed. Candidate discovery grants no authority. Acquire only a WorkItem, read the issued instructions/constraints/current criteria, and execute only that obligation. Never open, interpret or edit internal contract/Profile documents; use validated CLI commands for every operational change.

Read [recording material](references/workspace.md) only when adding progress, evidence or artifacts. Activity, evidence and assessment have different meanings. Do not claim completion because evidence exists or an executor return was accepted: independent review and explicit parent achievement remain separate.

Use context/status first and bounded history only to resolve a specific gap. Preserve cursors, truncation and incomplete-search information. Do not load a full Outcome or transcript on every iteration and do not maintain a duplicate manual pending list.

Tenant administration does not grant infrastructure authority. Tenant schema provisioning, suspension and deletion are protected host operations; agents must not invoke storage lifecycle commands to bypass a denied request. Runtime metrics require `installation:observe`; a hosted Tenant endpoint does not aggregate the whole installation. Use the scoped delivery diagnostic for the assigned Tenant/Outcome. Installation operator promotion is a host-controlled offline maintenance action.

## Resume safely

Reconcile `work recover --all-pending` with the original pinned Profile, then read current context/status and necessary durable references. Do not acquire replacement work, delete uncertain data or change expected versions to force a retry. The CLI owns signatures, frozen requests and receipt-driven cleanup. Read [protected signing and recovery](references/signed-protocol.md) for exact guardrails and [correction](references/correction.md) only when assigned an original review finding.

JSON stdout is structured; diagnostics are stderr. Report the selected contract, actual progress/proof references, remaining obligations, current authority and next action. Installing a Skill or cross-compiling a binary does not certify every runtime/platform.
