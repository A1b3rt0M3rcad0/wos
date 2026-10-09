# Signed local returns

Schema-2 workspaces support `work sign`, `work send`, `work finish` and the corresponding `review` commands. Choose an explicit profile when a workspace contains several profiles.

```sh
wosctl --profile executor work show CONTRACT_ID --for-agent
# Edit execution.progress and execution.material in this contract's YAML.
# execution.completion_intent is auto, direct or review.
wosctl --profile executor work sign CONTRACT_ID
wosctl --profile executor work send CONTRACT_ID
# finish prepares an unsigned draft when needed, then sends and cleans up.
wosctl --profile executor work finish CONTRACT_ID
wosctl --profile executor work recover --all-pending
```

The harness reads the signing key from the approved protected reference. Model output does not contain secrets, signatures or frozen envelopes. Sign checks current own key/policy, original CredentialID, exact authority, fencing and Task/case versions. It does not renew a lease or silently change the draft's expected version. `--completion` must match the explicitly edited execution draft. Core always performs decisive live authorization inside its transaction.

The contract and bounded profile state retain one immutable signed request, idempotency key and draft digest before sending. A missing profile write can be repaired from the original authenticated prepared file, with the same request; copying it to another CredentialID does not produce new authority. Unsigned local hints and issuer receipts alone are insufficient recovery provenance. `recover` reports an unsent prepared signature without sending it; `send` or `finish` explicitly sends that signature.

After a timeout, recover first reads the original signed acceptance. If no receipt exists it may replay only the same envelope/key, without signing, renewing, rebasing or compensating. A network/authorization error preserves the intention. Signing-key retirement does not prevent recovery of an already accepted receipt; live bearer/read authorization is still required.

A receipt must verify against pinned issuer keys and match destination, Principal, original request/key/digest, contract, Task, review target and final disposition. It must declare acceptance and a closed local obligation. Delivery for independent review closes the executor's local obligation even though the Task remains in progress. The independent reviewer needs its own profile, CID/key, acquisition and assessment; the executor's private key and process are not needed.

Only the matching contract is removed, after the receipt is persisted in bounded profile state. Progress/material edits after signing block deletion and leave the receipt in that file; output reports the existing remote commit and pending local cleanup. Retrying cleanup does not return work again. Another contract, profile, workspace, artifact or source tree is never recursively removed. Process death after unlink is recoverable from authenticated profile state.

Profile/contract locks coordinate cooperating harness processes; profiles do not isolate hostile operating-system writers. Observed content conflicts preserve data. Linux native process-death and actual Memory/SQLite/PostgreSQL HTTP journeys are covered; native Windows acceptance is recorded separately by hosted platform jobs.
