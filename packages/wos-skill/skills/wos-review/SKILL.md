---
name: wos-review
description: Review exact immutable WOS submissions against current criterion revisions and evidence using authorized independent assessment, without equating observations with approval.
---

# WOS review

Evidence and test observations do not mean a criterion is met. Assess explicit submitted material using the authorized reviewer Principal; ActorRef and installation do not grant permission.

## Signed Namespace review

Use your independently authorized reviewer profile: `review checkout --next`,
`review show ID --for-agent --output json`, edit the assigned review draft and
explicitly `review finish ID`. Check exact delivery, findings, required criteria
and evidence before deciding. Executor acceptance is not Task approval; a valid
signature is not evidence of quality. The UI never supplies your private key or
signs a decision for you. Read [signed protocol and recovery](references/signed-protocol.md)
for CAS, policy, protected signing and original-CID reconciliation.

## Legacy contracts_v1 only

1. Fetch `wos_get_work_submission` for the exact submission ID/digest, canonical artifact revisions/checksums and criterion evidence. Read the immutable contracted obligations, current criterion revisions and current documentary lifecycle. Expand only required proof references.
2. Verify artifact accessibility/material revision, registered evidence, unresolved impediments and required acceptance. Do not fetch arbitrary artifact URLs automatically or run artifact contents through WOS.
3. Record `wos_record_criterion_assessment` or permitted `wos_attest_criterion`, including submission_id, criterion ID/revision, expected WorkItem version, result, rationale and relevant canonical evidence IDs. The server binds the exact material digest and enforces independent reviewer/waiver policy.
4. If the submission changes, previous assessments cannot finalize the new delivery. Negative/pending criteria retain the contract reservation. Executor finalization is explicit via `wos_finalize_work_contract`; approval is not a holder release and does not automatically achieve parent aggregates.
5. Retracted proof and reopened dependencies must be rechecked. Report exact outstanding obligations rather than approving to make a workflow pass. Authorized administrative intervention is explicit revocation with contract ID/reason, preserving expiry/completion history.

Return a compact material-bound decision with assessment/evidence IDs and remaining action. Read [protocol and recovery](references/protocol.md) for safe retries. The human UI and wosctl review wrappers use the same Application authorization and immutable proof contract.
