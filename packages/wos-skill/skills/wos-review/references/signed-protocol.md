# Signed protocol and bounded recovery

Use this flow only after live Namespace protocol is `signed_contracts_v2`. A
schema-2 workspace does not activate a Namespace. The operator provisions the
persistent ServerID, independently approved issuer fingerprint, authorized
credential/enrollment and protected key reference. Check `wosctl version` against
the coordinated release; obtain `wos` (service) and `wosctl` (client) separately.

Initialize with explicit `--workspace-schema 2`, `--server`, `--server-id`,
`--namespace` and `--outcome`. Create an approved named profile using the actual
`profile-v2.schema.json` and operator enrollment. Tokens/private seeds belong to
host environment, a supported native keyring or protected mounted files outside
the workspace. Never put secret values in YAML, arguments, model context or logs.
Installing a skill neither provisions credentials nor grants server access.

Operational files are `.wos/project.yaml`,
`.wos/profiles/PROFILE/profile.yaml` and
`.wos/profiles/PROFILE/contract/CONTRACT_UUID.yaml`. One contract document contains
issued immutable specification/authority, editable execution or review draft,
and bounded technical local recovery state. No operational outbox/history
folders are required. `.yaml` and `.yml` are aliases; duplicate stems are refused.
Never edit issued sections, signatures, local markers or frozen intentions.

Execution: `wosctl --profile executor work checkout --next --count 1 --limit 25`,
then `work show CONTRACT_UUID --for-agent --output json`. Edit only the execution
draft following the packaged contract schema. `work finish CONTRACT_UUID` freezes,
signs, sends and reconciles the original return. Alternatively run `work sign`,
then explicit `work send`. Signing a delivery proves its origin, not its quality.
Returned execution can close the executor obligation while independent review
still blocks Task completion; parent Objective/Outcome completion stays explicit.

Review: an independently authorized Principal/profile uses
`wosctl --profile reviewer review checkout --next`, then
`review show REVIEW_CONTRACT_UUID --for-agent --output json`. Read exact assigned
case/delivery, findings, criterion revisions and required evidence. Edit the
review draft, then `review finish REVIEW_CONTRACT_UUID`. Approval, rejection or
replanning is an explicit signed decision, not inferred from passing tests. Two
keys or profile names for one Principal do not establish independence.

Always pass `--profile` on each call. Use focal show/list output; request omitted
technical proofs through harness code only when needed. `--count` freezes
independent intentions, not an atomic batch. Preserve partial results, cursors and
search completeness. Server Principal quotas apply across credentials/profiles.

After uncertainty use `work recover` or `review recover` for the selected profile.
The client checks the original-CID operation/receipt before any identical replay.
Recovery never automatically sends a prepared-only return or acquires replacement
work. Do not change CAS, UUID, payload, destination or credential to force success.
Accepted-but-unmaterialized acquisition publishes its original contract; cleanup
requires verified signed closed acceptance and preserves unconfirmed edits.

Use explicit `work renew/resume/refresh` or their review equivalents only when
needed. Foreground `work keepalive --all-active --foreground` is supervised by the
host, not a WOS scheduler. Resume rotates fencing without extending expiry;
refresh never takes over. Expired/revoked authority cannot be revived.

Migration is an explicit bounded dry-run and separate approved schema-2
destination. Legacy imports remain `legacy_unsigned`, never acquire retroactive
signatures, and signed mutators refuse them. Preserve unresolved journals/receipts.

After host issuer recovery, independently verify the new fingerprint, then use
`wosctl --profile executor profile trust --issuer-fingerprint sha256:APPROVED_DIGEST`.
It retains historical pins and exact pending returns. Do not blindly trust a
changed endpoint or re-sign an uncertain return. Native keyring availability and
all-runtime integration are not established by installer tests or cross-builds.

For HTTP/MCP use the actual typed signed command schemas. Acquisition is
`wos_acquire_signed_work_contract` / `wos_acquire_next_signed_work_contract`,
review uses `wos_acquire_signed_review_contract`, and frozen returns use
`wos_return_signed_work` / `wos_return_signed_review`. Signed envelopes and
verification belong in the protected harness, outside the language-model view.
The server authorizes every mutation from current grants/policy/authority.

## Editable signed review example

`review.decision` accepts exactly `approved`, `changes_requested` or
`inconclusive`. Use original criterion IDs/revisions from `review show`; prose
alone does not satisfy a required criterion. Edit only the draft:

```yaml
review:
  progress:
    summary: Examined the accepted commit and ran independent checks
    blockers: []
    next_action: Return the independent decision
  decision: approved
  material:
    reason: Describe what was actually checked
    reviewed_source_version: ACTUAL_COMMIT
    criterion_assessments:
      - criterion_id: ORIGINAL_CRITERION_UUID
        criterion_revision: "1"
        result: met
        rationale: Actual evidence for this original criterion
```

Assessment results are `met`, `not_met`, `inconclusive` or `waived`; only use
`met` after checking fulfillment. Do not invent evidence IDs or revisions.
`changes_requested` requires findings tied to original requirement references.
There is no `material.decision`; submission/case IDs come from verified issued
metadata. `work validate` and generic `--help` are not signed protocol operations;
use focal `show` and the coordinated packaged schema. If local preparation fails,
inspect pending state and recover before changing a draft. Never edit a frozen
intention after an uncertain send.

After changes_requested, use [the explicit correction flow](correction.md).
Correction is intentionally excluded from generic --next discovery.

Evidence semantics: `evidence_type: test_result` records test observations in its description/source; it must not contain `measurement`. Numeric evidence uses `evidence_type: measurement` with nonempty `measurement.value`. The same rule applies to executor and reviewer drafts. The CLI validates this before freezing a return; the server remains authoritative. Preserve any already-frozen rejected intention and its original files; do not edit signed bytes or invent a replacement acquisition to repair uncertainty.

For editable material shapes, read only the relevant execution/review section of `schemas/contract-v2.schema.json` in the verified CLI archive, or `node_modules/@a1b3rt0m3rcad0/wos/schemas/contract-v2.schema.json` in the npm installation. Preserve the issued result binding fields. Criterion-local evidence references attach selected registered evidence to the submission; an extra general evidence selection is unnecessary. Required assessments remain explicit typed review records at the original criterion revision.
