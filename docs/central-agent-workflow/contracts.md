# Contract lifecycle and protected operational changes

## Existing contract

Work acquisition is for a WorkItem, with Outcome/Objective context and current authority. A WorkContract contains immutable issued specification/authority, the caller's mutable operational material, and technical recovery metadata. Review is a separate authorized obligation over an exact delivery.

The implementation must preserve `packages/wos-cli/contract_v2.go`, `yaml_v2.go`, `publication_v2.go`, `pending_acquisition_v2.go`, `pending_lease_v2.go`, `return_preflight_v2.go`, `return_send_v2.go`, and `return_resolution_v2.go` protections. Restricted serialization can remain an internal persistence detail; it is not the model's edit API.

## CLI-owned material

1. Add typed commands for progress summary, activities, artifacts, evidence, review decisions/assessments/findings and correction responses.
2. Validate fields and semantic combinations before any signing or network mutation. Reject unknown material and stale draft CAS.
3. Hold the selected Profile/contract lock for local mutation. Re-read, verify issued proof/binding and compare the expected original draft before atomically publishing changed operational data.
4. Never update issued specification, contract authority, execution ID, fencing, issuer proof, request identity, or signed bytes through a material command.
5. Once a return is frozen/prepared or uncertain, refuse material changes until its original intention is reconciled or explicitly resolved with verified durable server proof.
6. Distinguish local draft acceptance from server registration. Output must say which records are still local and what must happen to make them durable.
7. Provide explicit `set`/`remove`/`list` paths for erroneous draft assessments, findings and correction responses. Preserve record identity and declared obligation on replacement, fail on ambiguous selection, and reject deletion of staged evidence still referenced by an assessment. Never require internal document editing to repair authoring mistakes.
8. Bind local pagination to contract, section and draft digest. Bound staged context summaries and declare omissions; expand through typed list commands. An omitted original review specification must be fetched using its authenticated reference, issuer proof and exact digest rather than reconstructed from current mutable Task state.

## Finish and recovery

`work check` validates the current candidate and exposes known gates. `work finish` prepares, signs, sends and reconciles the original return using existing semantics. Accepted execution may close the execution obligation while the WorkItem awaits independent review. Neither contract completion nor a passing review automatically achieves an Objective/Outcome.

After timeout, use `work recover --all-pending` for the same pinned Profile. Recovery checks original-CID receipt/operation before exact replay. It never automatically sends a prepared-only return, creates replacement work, changes CAS/fencing to force success, or cleans uncertain files. `review recover` is the equivalent review path.

An activity's full durable response can exceed the transport's compact-receipt limit because captured artifact versions also appear in documentary metadata. Expand and validate the exact original registry result first. Persist only the derived Activity plus command identity/revision needed for local material publication, inside the existing protected response bound; preserve original request payload/key/fingerprint and server result. A local publication conflict must remain recoverable without another server append.

Explicit renew/resume/refresh remain distinct. Renew extends live authority under server rules. Resume rotates execution/fencing without reviving expiry. Refresh observes without takeover. Foreground keepalive is host-supervised and not a scheduler.

Definitively rejected frozen returns require the existing explicit `resolution-status` / reasoned `resolve` path; an HTTP error alone is not abandonment proof. Confirmed original acceptance wins over a later closure attempt. Keep historical request/signature bytes.

This terminal-resolution path does not cover activity appends. A frozen append that cannot be accepted after expiry or definitive rejection remains preserved for operator intervention; the branch provides no authenticated append-closure receipt. Do not clear it, change its payload/key, or treat another Profile as resolution. Such a closure needs a separate reviewed protocol extension before an automatic cleanup path can be advertised.

## Acceptance

- Contract/material setters preserve issued and recovery bytes; malformed input causes no partial write or frozen intention.
- Two local writers cannot lose one another's edits. Other Profiles can still operate while one contract is locked.
- Tests retain full disk/process death, atomic publication, pending acquisition/lease/return and issuer recovery coverage.
- Local cleanup requires verified signed closed acceptance. Server history and material remain available afterward.
- Legacy imports retain unsigned status and cannot acquire retroactive signatures or bypass a signed Namespace.

CAW-03 depends on central Profiles. A smaller CLI command surface must reuse these guards rather than create a second return protocol.
