# ADR 033 — Namespace administration of signed Principal quotas

Status: accepted implementation decision; final-source acceptance pending.

WorkAcceptancePolicy gains optional max_active_work_per_principal and max_active_review_per_principal integers only at Namespace scope (OutcomeID and WorkItemID absent). Absent fields retain the published 3/1 defaults; explicit values must be 1..100. Zero never means unlimited. Existing JSON policies need no schema migration; narrower Outcome/WorkItem policies cannot change these ceilings.

Signed acquisition reads that policy in the same authorized UnitOfWork and Namespace protocol guard as current credential access and active counts. Principal, credential and Namespace ceilings apply independently across Outcomes. Additional CIDs/profiles of one Principal do not create extra Principal capacity or independent review identity. Increasing a ceiling grants no permissions, key enrollment, stronger review authority or exemption from current issuer/specification/lease/fencing checks. Lowering it keeps existing authority intact and prevents new acquisition while activity exceeds the new ceiling. Exact retries and bounded next acquisition use the same acquisition services.

Signing identity diagnostics return effective ceilings and only the caller's Principal/credential activity. Namespace limits are public configuration hints; no other Principal's contract list is exposed. Error messages identify the rejecting ceiling. CLI profile/doctor status includes these diagnostics. All execution-time authorization remains authoritative.

Focused tests cover legacy JSON/defaults, bounds/scope, unauthorized administration, Namespace/policy CAS, two CIDs across Outcomes, distinct Principals, current revocation, lowering with active contracts, independent credential/Namespace work and review ceilings, and 50 concurrent acquires. Memory/SQLite/real PostgreSQL pass; real empty-database restores preserve execution/review/correction authority in the same journey. Final full/native/compatibility gates remain required.
