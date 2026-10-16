---
name: wos-continuity
description: Resume WOS work through pinned CLI context, bounded server history and original-intention recovery instead of prior conversation or internal documents.
---

# WOS continuity

Persisted server state and valid authority determine work. A previous conversation, a local path or a historical receipt does not. Use the CLI as the only operational interface; never open, interpret, edit or delete internal operational documents.

1. Start with the approved Profile and minimal authorized task/contract intention. Pin `--profile NAME` or process-scoped `WOS_PROFILE`. `wos init` is installation-scoped and commands work from any directory; the interactive active default must not control concurrent worker identity.
2. Reconcile `wos --profile executor work recover --all-pending` (`review recover` for review) before new effects after uncertainty. A prepared-only return stays unsent. Missing local material is repaired from the original operation, not another acquisition.
3. Read `work list`, then focal `work context --for-agent --output json` and `work status`. Omit the contract argument only with one applicable local contract; otherwise select an exact/unique reference. Read immutable requirements, live authority/expiry/blockers, exact criteria and current review findings.
4. Use `work history --limit 10 --output json` only to answer a specific gap. After accepted local cleanup use the authorized server query with exact `--work WORK_ITEM_UUID` and Outcome scope as needed. Honor cursor/omission/truncation information; a missing page is not proof that work never occurred.
5. Locate actual artifact revisions/checksums, evidence and accepted result. A dirty tree, missing upload or unknown test result stays explicit. Activities report declarations, evidence records observations, and assessments record explicit criterion evaluation; none silently substitutes for another.
6. Resume the next permitted external action. Record new progress using `work log add`, not a copied transcript or duplicated manual pending list. Derive outstanding obligations from context/status/check and current review findings.
7. Stop on expiry, revocation or stale execution. Explicit same-holder resume rotates fencing without reviving expiry; refresh never takes over. A new valid acquisition after terminal authority is distinct and cannot repair an uncertain old return.
8. Finish only with validated actual material and explicit `work finish`. Preserve original request identity, destination, payload, signature and CAS during recovery. Cleanup requires verified signed closed acceptance; server history remains the authoritative durable record.

Use [continuity protocol](references/protocol.md) for read budgets, [protected signing and recovery](references/signed-protocol.md) for interrupted operations/trust rotation, and [correction](references/correction.md) for acknowledged original findings. Return a small resume packet: obligation, selected Profile/contract, durable progress/proof, authority, blocker and next relevant reference. Do not dump the full history or claim token savings without measurement.
