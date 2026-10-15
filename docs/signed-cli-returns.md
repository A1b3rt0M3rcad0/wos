# Signed local returns

Schema-2 workspaces support `work sign`, `work send`, `work finish` and the corresponding `review` commands. Choose an explicit profile when a workspace contains several profiles.

```sh
wos --profile executor work show CONTRACT_ID --for-agent
# Edit execution.progress and execution.material in this contract's YAML.
# execution.completion_intent is auto, direct or review.
wos --profile executor work sign CONTRACT_ID
wos --profile executor work send CONTRACT_ID
# finish prepares an unsigned draft when needed, then sends and cleans up.
wos --profile executor work finish CONTRACT_ID
wos --profile executor work recover --all-pending
```

The harness reads the signing key from the approved protected reference. Model output does not contain secrets, signatures or frozen envelopes. Sign checks current own key/policy, original CredentialID, exact authority, fencing and Task/case versions. It does not renew a lease or silently change the draft's expected version. `--completion` must match the explicitly edited execution draft. Core always performs decisive live authorization inside its transaction.

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
# After terminal proof and local cleanup, edit the retained draft explicitly:
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
exact original signed intention and current draft bytes under the profile's
`history/return-REQUEST_ID.yaml` and `.contract.yaml`, then clears only the old
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
