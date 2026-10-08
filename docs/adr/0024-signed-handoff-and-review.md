# ADR-0024 — Atomic signed delivery and independent review

Status: accepted for implementation, 2026-10-08. Not an implementation claim.

For signed_contracts_v2 only, ADR-018 gains a fourth terminal: delivered means accepted execution handoff, not Task completion or voluntary release. Return atomically writes documents/submission/signature and either authorized explicit assessments + Conclusion + WorkItem done + completed, or delivered + pending ReviewCase and closed execution authority. ReviewCase fixes submission/digest/spec/policy/round. One ReviewContract per case has its own lease/fencing. Review approval uses reviewer authority, not executor lease, and rechecks current criteria/material/blockers/dependencies/parent state. Changes requested records immutable findings and enables a new execution/fence linked to previous material; terminal contracts never reactivate. Inconclusive reopens review availability without pretending approval/correction. Open review freezes material specification. All transitions, receipts, events and outbox share one UoW; no nested public command transactions.

The owner authorized [P00–P13](../signed-contracts-implementation-plan.md). V1 behavior remains until explicit cutover. Native Windows, hosted CI, registry publication and deployed consumer pilots are separate evidence gates.
