# Signed local returns

This source-branch guide targets the central CLI; published `wosctl@0.4.2` lacks the typed preparation commands shown below. Central signed profiles and explicit legacy-compatible schema-2 workspaces support `work sign`, `work send`, `work finish` and the corresponding `review` commands. Pin `--profile` in concurrent agent sessions; otherwise the installation uses its selected active profile. Internal drafts are CLI-owned.

```sh
wos --profile executor work show CONTRACT_ID --for-agent
wos --profile executor work progress set CONTRACT_ID --summary "Implemented the change" --next-action "Inspect the result"
wos --profile executor work evidence add CONTRACT_ID --description "Inspection notes" --uri https://example.test/review/123
wos --profile executor work result set CONTRACT_ID --summary "Implementation and inspection recorded" --completion auto
wos --profile executor work check CONTRACT_ID
wos --profile executor work finish CONTRACT_ID
wos --profile executor work recover --all-pending
```

For an explicit preparation/send split, use `work sign CONTRACT_ID` followed by `work send CONTRACT_ID` **instead of** `work finish`. Successful send may already clean up that contract; do not finish it again as a new delivery.

The harness reads the signing key from the approved protected reference. Model output does not contain secrets, signatures or frozen envelopes. Sign checks current own key/policy, original CredentialID, exact authority, fencing and Task/case versions. It does not renew a lease or silently change the draft's expected version. `--completion` must match the intent set through `work result set`. `check` is read-only: technical readiness is separate from quality assessment and server obligations. Core always performs decisive live authorization inside its transaction.

The contract and bounded profile state retain one immutable signed request, idempotency key and draft digest before sending. A missing profile write can be repaired from the original authenticated prepared file, with the same request; copying it to another CredentialID does not produce new authority. Unsigned local hints and issuer receipts alone are insufficient recovery provenance. `recover` reports an unsent prepared signature without sending it; `send` or `finish` explicitly sends that signature.

After a timeout, recover first reads the original signed acceptance. If no receipt exists it may replay only the same envelope/key, without signing, renewing, rebasing or compensating. A network/authorization error preserves the intention. Signing-key retirement does not prevent recovery of an already accepted receipt; live bearer/read authorization is still required.

A receipt must verify against pinned issuer keys and match destination, Principal, original request/key/digest, contract, Task, review target and final disposition. It must declare acceptance and a closed local obligation. Delivery for independent review closes the executor's local obligation even though the Task remains in progress. The independent reviewer needs its own profile, CID/key, acquisition and assessment; the executor's private key and process are not needed.

Only the matching contract is removed, after the receipt is persisted in bounded profile state. Progress/material edits after signing block deletion and leave the receipt in that file; output reports the existing remote commit and pending local cleanup. Retrying cleanup does not return work again. Another contract, profile, workspace, artifact or source tree is never recursively removed. Process death after unlink is recoverable from authenticated profile state.

Profile/contract locks coordinate cooperating harness processes; profiles do not isolate hostile operating-system writers. Observed content conflicts preserve data. Linux native process-death and actual Memory/SQLite/PostgreSQL HTTP journeys are covered; native Windows acceptance is recorded separately by hosted platform jobs.

## Explicit terminal resolution of a rejected return

After a definitive semantic rejection, preserve the original frozen return.
Recovery first looks for its accepted receipt and retries the exact original
intention. A missing receipt, timeout, EOF, 401/403/409 or server failure never
proves that it is safe to discard the intention.

When the operator explicitly chooses to close a return without acceptance:

```sh
wos --profile executor work resolution-status CONTRACT_ID
wos --profile executor work resolve CONTRACT_ID --reason "close the rejected material before a corrected return"
wos --profile executor work recover --all-pending
# After terminal proof and local cleanup, correct material with typed commands:
wos --profile executor work result set CONTRACT_ID --summary "Corrected delivery"
wos --profile executor work sign CONTRACT_ID
wos --profile executor work send CONTRACT_ID
```

Use `review` instead of `work` for an independent review return. `resolve`
requires advertised server support and the current original credential. It
freezes a **separate** signed resolution journal under `return_resolution`; the
parent payload, signature, request ID, idempotency key and CID remain unchanged.
A previously frozen resolution is recovered unchanged, including its reason.
The server either returns the original acceptance or commits an immutable
`closed_without_acceptance` decision. There is no local force/re-sign/delete
escape hatch.

The CLI verifies the pinned issuer, exact original and resolution bindings and
persists proof before touching the contract file. Negative closure archives the
exact original signed intention and current draft bytes under the profile's protected internal history, then clears only the old
pending signature. It retains current editable material and human edits. Native
cooperative locks, confined paths, exclusive creation, fsync and content CAS
protect publication; profiles are not isolation against a hostile OS writer.
Corruption or disk-full leaves recovery pending. Normal accepted-return cleanup
keeps its existing behavior. Output distinguishes original return acceptance
from a terminal resolution; neither proves external execution was undone.

A new return still needs valid live authority, policy, CAS and fencing. Expired
leases, revoked credentials, changed signer authority and changed versions use
existing explicit recovery/acquisition; resolution never renews or rebases them.
Stable assessment fields and known finding references are checked before freeze,
but the server always evaluates current authority and state.

## Agent-facing delivery and review

`work finish` reports `disposition`, `contract_status`, `work_item_lifecycle`, `task_completed`, and `cleanup_pending`. A confirmed delivery can close the executor's contract while the Task still awaits independent review. A receipt is acceptance of the operation, not proof of external execution.

The independent reviewer inspects the exact submission before preparing its separate return. Set the approved decision before adding required criterion assessments; adding assessments to the initial inconclusive draft is rejected:

```sh
wos --profile reviewer review next
wos --profile reviewer review context
wos --profile reviewer review decision set --decision approved --reason "The submitted material meets the criterion"
wos --profile reviewer review assessment add --criterion CRITERION_UUID --result met --rationale "Inspected the required behavior"
wos --profile reviewer review check
wos --profile reviewer review finish
```

Execution `log add`, `evidence add`, and `artifact add` append durable server activity immediately. Review evidence and artifacts are staged locally until the signed review return. Neither registration nor a local readiness check constitutes assessment or approval. Server `work history --work TASK_UUID --outcome OUTCOME_UUID` remains available after local cleanup; page cursors and revision bounds must be respected.

## Unresolved activity rejection

An uncertain activity append retains its exact authenticated pending request and is reconciled with `work recover`. The current protocol has terminal resolution for rejected final returns, not for an already frozen activity append that later becomes unauthorized or expired. Such an append remains pending; do not edit/discard its bytes, substitute a new request, or treat a different profile as resolution. Restore the applicable authorized recovery path when possible and retain the original material for operator diagnosis. This limitation is separate from a confirmed activity receipt and normal accepted-return cleanup.
