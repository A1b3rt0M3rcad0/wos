---
name: wos-workspace
description: Operate an authorized local WOS task through API-only wosctl, restricted YAML drafts and durable recovery without loading whole conversations.
---

# WOS local workspace

The npm skill installer only copies instructions. It does not connect MCP, install credentials, start a daemon or launch subagents. Obtain wosctl from the verified coordinated release and check live capabilities/Namespace protocol. Server wos and client wosctl are separate executables.

## Signed Namespace workspace (schema 2)

Use the approved named profile on each call. `work checkout --next --count 1`
materializes one contract document; `work show ID --for-agent --output json`
provides bounded instructions. Edit only its execution draft, then `work finish ID`.
Use `review checkout/show/finish` with the independent reviewer profile. Preserve
issued documents and local recovery markers; never manually delete an uncertain
contract. `work recover`/`review recover` reconcile exact original intentions.
Read [signed files and recovery](references/signed-protocol.md) for onboarding,
protected secret references, migration, lease maintenance and issuer rotation.

## Legacy contracts_v1 workspace (schema 1) only

1. Initialize using explicit --server, --namespace, --outcome and --credential-env. The config stores an environment reference, never a token. Preserve the destination trust binding; changing configuration cannot redirect an existing intent or its credential.
2. Use `wosctl work checkout ID --version N --output json`, or bounded `--next --outcome ID`. `--dry-run` shows candidates only. Save the receipt/workspace location. Read contract.yaml once: immutable focal obligation, spec digest and original grant. state.json is only observed cache; verify current authority with status before effects.
3. Edit checkpoint.yaml for material progress/next action and result.yaml for typed material/documentary records. Use durable repository commits/artifact references. Preserve explicit dirty/unknown facts. YAML executes nothing; unsupported/unknown fields, anchors, aliases, tags and duplicate keys are rejected.
4. Use validate/diff, then checkpoint or sync. Sync resolves unique local keys to durable IDs; changing registered material requires a new key. Checkpoints never complete or release authority. Do not reinterpret observed_result as assessment.
5. Submit immutable material, obtain independent required assessment, and explicitly finalize. `finish` is a documented sync→submit→finalize sequence with separate receipts; exit 7 is pending review, not completion. Keepalive only runs when explicitly launched with --foreground and supervised/cancelled by the host.
6. On a new session use status and recover for the exact contract. Refresh preserves edited drafts. Resume --takeover is explicit and rotates execution/fencing without reviving expiry. Missing local files after acquisition are repaired from the original journal/receipt; never acquire another task to compensate.
7. On lost response replay the original stored payload/key. Recover --next repairs uncertain next acquisition. Do not change config/destination or CAS to force a retry. Remove an abandoned cooperative local lock only with explicit recover --break-lock after stopping the old process; it does not revoke remote authority.

JSON stdout is structured and diagnostics are stderr. Exit codes: 0 success, 2 invalid input, 3 auth, 4 conflict, 5 authority lost, 6 transport/commit uncertain, 7 review pending, 8 no acquisition in the scanned page, 9 local materialization failure. Read [file and recovery examples](references/workspace.md) only for the current operation. Never claim Windows/agent-runtime certification from cross-compilation or skill installation alone.
