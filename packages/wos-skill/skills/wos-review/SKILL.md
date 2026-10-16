---
name: wos-review
description: Review exact WOS submissions and current criteria using an independent Principal and typed CLI decisions, findings and assessments without internal file edits.
---

# WOS independent review

Use the CLI as the only operational interface. Evidence and test observations do not mean a criterion is met. Approval evaluates the exact submitted material against the original current criterion revision. Actor labels, Skill installation and multiple keys for one Principal do not grant independent authority.

1. Use a genuinely independent authorized reviewer Principal/Profile. Pin `--profile reviewer` or process-scoped `WOS_PROFILE`; the interactive active Profile is not an isolation boundary. `wos init` initializes installation data, not project state.
2. Acquire the review obligation with `wos --profile reviewer review checkout --next`. Read `review show --for-agent --output json` for exact delivery, assigned case, original findings/criteria, evidence and source bindings. Supply a contract reference when more than one obligation applies.
3. Inspect the actual artifact version/checksum and run necessary checks outside WOS. Expand only proof references needed for a criterion, using bounded results. Never fabricate an observation or automatically execute arbitrary referenced artifacts.
4. After the actual checks, set `review decision set --decision approved --reason TEXT`, then record each required criterion explicitly with `review assessment add --criterion REF --result met --rationale TEXT`, using original IDs/revisions. Add `--evidence UUID,...` only for actually registered proof; staged review observations use the returned `local:LOCAL_KEY` reference.
5. For changes requested, first set `review decision set --decision changes_requested --reason TEXT`, then use `review finding add --criterion REF --description TEXT` or `--requirement ORIGINAL_REF`, never both for one finding. `inconclusive` also needs a factual reason/findings. Set its reason before adding findings. Non-approval decisions omit criterion assessments, including passing ones; observations belong in the reason/evidence.
6. Use typed CLI material operations only; never open, interpret or edit internal review/contract documents. Review artifact/evidence additions remain local staged material until verified signed finish; inspect their returned local keys. Complementary records require an immutable source version/checksum in the accepted executor artifact set. If none exists, omit those additions and assess the accepted material with actual checks/rationale.
7. Run `review check`, then explicitly finish with `wos --profile reviewer review finish`. A signature proves origin, not quality. Executor acceptance, reviewer decision, Task completion and Objective/Outcome achievement remain distinct transitions. Changed delivery requires new exact-material review.
8. Recover an uncertain result with `review recover --all-pending` on the original Profile. Never change payload/CAS/fencing, re-sign frozen bytes or delete uncertain material to make the workflow pass. A prepared-only return remains unsent; receipt-driven cleanup requires verified signed acceptance.

Read [review protocol](references/protocol.md) for material/decision constraints and [protected signing and recovery](references/signed-protocol.md) for interruptions and trust. Read [correction](references/correction.md) when returning an original finding to the coordinator. Return a compact decision with checked material, assessment/evidence references and outstanding obligations; do not dump whole histories or infer absence from omitted pages.
