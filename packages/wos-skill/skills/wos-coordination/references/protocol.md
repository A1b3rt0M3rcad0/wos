# Bounded context, authority and continuation

Use only the centralized CLI for agent operations. Start with pinned Profile, focal context and current status; request a bounded history page only to resolve a particular gap. `--limit` and `--cursor` are explicit on paged reads. Preserve evaluated-at/revision, omitted/truncated and search-completeness information. A missing item on one page is not absence.

The issued specification is immutable. Candidate discovery does not grant authority. A historical receipt proves the old operation, not current lease/permission. CAS, quotas, execution/fencing and current blockers remain authoritative. Stop external effects on expiry, revocation or stale fencing; never retry by changing expected versions.

Use typed progress/activity/material commands. Activities are participant declarations, Evidence records observations and exact sources, and CriterionAssessment explicitly evaluates current criterion revision/exact material. Changed delivery needs new review. WorkItem completion does not imply Objective/Outcome achievement.

After uncertainty use the original Profile's `work recover --all-pending` or `review recover --all-pending`. Preserve the original destination, request identity, payload and frozen signature. Recovery never automatically sends prepared-only material or acquires replacement work. Cleanup requires verified signed closed acceptance. See [protected signing and recovery](signed-protocol.md).
