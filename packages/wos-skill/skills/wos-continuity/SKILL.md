---
name: wos-continuity
description: Resume WOS work and maintain a small active context using persisted checkpoints and on-demand expansion. Use on session restart, handoff, compaction or task boundary.
---

# WOS continuity

Keep coordination state outside the chat and reconstruct only the current task. WOS cannot erase existing messages or control the runtime's compaction.

1. Locate the assigned Outcome/WorkItem using IDs or bounded authorized discovery. Start with focal task context; read a small continuity snapshot only if you need overall orientation.
2. Preserve revision/evaluation markers and check omission/cursor fields. Expand only necessary sections. On `snapshot_changed`, refresh and reconsider the affected intention. WOS has no universal "changes since checkpoint" token: use current focal reads or bounded timeline filters when needed.
3. Read the actual task lease and persisted state before deciding whether work can continue. A remembered claim, stale version, lost response or old checkout does not prove current authority or completed external effects.
4. At a task boundary, persist artifacts, evidence, explicit decisions and task result using supported commands. For unfinished work, a checkpoint document may be registered as an Artifact reference; do not complete the task merely to store a checkpoint. Use stable, reviewer-accessible references instead of temporary sandbox paths for durable handoff.
5. Keep an active summary containing Outcome/task IDs, intent, constraints still in force, changed files, proof references, next action, unresolved issue, and claim/expiry if relevant. Never include tokens. Fetch entity versions fresh before a mutation.
6. Use the runtime's native summarization/compaction or a fresh authorized worker where supported. Check that essential restrictions and pending work remain represented. Raw traces can leave active context while remaining available in external artifacts.

Do not scan the whole repository, expand every section, replay all conversation or fetch a complete timeline by default. Restrict searches and report concise test results. Reducing supervisor context does not prove lower total tokens or higher correctness; measure context size, repeated reads, retries and acceptance quality in the consumer.

Read [protocol and recovery](references/protocol.md) when an interrupted command, stale snapshot or lost lease needs reconciliation.
