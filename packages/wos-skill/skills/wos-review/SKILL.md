---
name: wos-review
description: Review WOS task deliverables and revisioned success criteria with explicit evidence and immutable conclusions. Use for human/agent handoff, acceptance and contested results.
---

# WOS review

Evidence is an observation, not automatic approval. Inspect current criteria, their revisions, owner state and material artifacts before certifying work.

1. Read the focal WorkItem or owner entity and only relevant proof references. Validate deliverable provenance, exact commit/version and meaningful tests; do not treat a summary or screenshot as proof of unrelated behavior.
2. Follow the criterion's verification mode and reviewer policy. Record `wos_record_criterion_assessment` with current criterion revision/owner version, result, reasoning, required proof and evaluator metadata. A reviewer credential must have appropriate authority; independence policies can prohibit the executor from approving its own work.
3. Retired/revised criteria do not inherit old positive assessments. Evidence retracted or contradicted cannot justify a new positive conclusion. A waiver requires explicit permission and reason; do not use it to bypass failed verification.
4. WorkItem completion uses an executor's valid claim/fencing; Objective and Outcome certification are separate authorized commands (`wos_achieve_objective`, `wos_achieve_outcome`). Verify structural obligations and all required current criteria. Never infer achievement from done-count alone.
5. Report unresolved proof, issue/blocker or contested conclusion. Preserve immutable history; do not erase proof or auto-reopen terminal entities to make a view look consistent.

Humans can participate through the official WOS UI using the same state and authorization. Read [protocol and recovery](references/protocol.md) for live schemas and conflicts. Return a concise acceptance receipt with assessment/conclusion IDs and concrete remaining obligations.
