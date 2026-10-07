---
name: wos-delegation
description: Delegate bounded WOS work to native subagents while keeping supervisor context small and coordinating claims and shared files. Use when the assignment authorizes delegation and the runtime supports it.
---

# WOS delegation

WOS does not launch agents. Inspect the runtime's native delegation tools and applicable repository/user permissions before creating workers. If delegation is unavailable or not authorized, execute sequentially or report the limitation; do not invent a universal spawn API.

1. Define explicit WorkItems for independent deliverables, acceptance obligations and dependencies within the authorized Outcome. Parent grouping and plan order do not imply execution dependencies. Register real `depends_on` relations only when needed.
2. Delegate only ready work. Give each worker the authorized Namespace/Outcome/task IDs, objective and acceptance constraints, permitted files/tools, stop conditions and evidence requirements. Ask it to fetch focal context from WOS rather than forwarding the entire chat.
3. Select fresh/minimal history explicitly if supported. A separate context window can inherit the parent history by default. Do not claim context isolation, filesystem isolation or reduced total tokens without runtime evidence.
4. Assign one executor to acquire each task claim. Use distinct provisioned Principal/ActorRef credentials where available. Sharing one credential does not distinguish agent authority. Never transfer a supervisor's claim to a different authenticated identity by assumption.
5. Coordinate shared-file ownership outside WOS. Claims cover WorkItems, not filesystem locks. Workers may share a sandbox; avoid overlapping edits and inspect merged changes. Use isolated checkouts only when permitted by the user/project/runtime, not as an automatic requirement.
6. Each worker renews its own lease, handles versions/fencing and persists result/proof. Require a concise return: task status, artifact/evidence IDs, changed files, tests, unresolved issue and next reference. Retrieve detailed logs only for a review or failure.
7. Re-read the relevant state before dependent work or certification. Do not re-execute done tasks. A crashed worker requires an explicit expired-lease reclaim with a new fencing token; it is not permission to silently complete its work.

Read [runtime differences](references/runtimes.md) when configuring delegation. Use wos-coordination for claims/recovery and wos-review for final evaluation. WOS does not issue tool permissions, provide a model, schedule workers or own their conversations.
