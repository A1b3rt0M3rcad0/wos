---
name: wos-coordination
description: Discover, acquire and follow bounded WorkItem obligations through the centralized WOS CLI while preserving authority, proof and independent review.
---

# WOS coordination

WOS persists shared work and authority; the external host executes tools and agents. Use the CLI as the only operational interface. An approved Profile is required; Scope IDs, metadata, ActorRef and Skill installation never grant access.

1. Pin the authorized Profile using `--profile NAME` or process-scoped `WOS_PROFILE`. `profile use` is an interactive default, not a concurrent worker identity switch. Run `wos init` once for installation data; do not create project operational state or manipulate internal documents.
2. Discover with `wos --profile executor work available --limit 10 --output json`, then acquire `work next` or an explicit `work checkout REF`. Add `--outcome REF` when selection requires it. UUIDs are supported; unique titles/prefixes are conveniences. Ambiguity or incomplete search requires an explicit target, never choosing the first match.
3. Read `work context --for-agent --output json` and `work status`. The contract belongs to one WorkItem; Outcome/Objective provide context. Preserve immutable issued instructions, constraints, current criterion revisions and acceptance policy. Listing is not acquisition and cached receipt is not current authority.
4. Execute only the contracted work outside WOS. Before dependent effects consult current blockers/authority. A lease does not lock files or deployments; coordinate shared files in the external host.
5. Append meaningful events with `work log add --type progress|discovery|problem|correction|validation|result --description TEXT`, selecting one actual type. Record real proof through `work evidence add` and `work artifact add`; use the relevant typed flags and immutable source versions. Activity is a declaration; evidence is an observation; assessment is explicit approval/evaluation.
6. Set the actual delivery with `work result set --summary TEXT`; inspect derived obligations via `work status`, then run `work check` and explicit `work finish`. Accepted execution may await independent review. WorkItem completion never automatically achieves an Objective/Outcome.
7. If blocked, stop dependent effects and report the exact impediment; do not fabricate success or a manual redundant pending list. Only explicitly authorized operations can change blockers/review policy. Maintain a valid lease only through explicit renewal/host-supervised foreground keepalive.
8. After timeout/interruption use `work recover --all-pending` for the same Profile. Never change CAS/fencing/payload, downgrade protocol, or reacquire to repair an uncertain result. Expiry/revocation/stale fencing stops authority; refresh is not takeover.

Contract arguments can be omitted only when exactly one applicable local contract exists. Use a contract reference for parallel obligations. Keep reads compact, follow bounded cursors only for a question, and report omitted data truthfully.

Read [coordination protocol](references/protocol.md) for bounded status and [protected signing and recovery](references/signed-protocol.md) for original-intention safety. Use [correction](references/correction.md) only after an explicit prior-review assignment. Return a compact packet of contract/result/proof references, authority disposition, remaining obligation and next action.
