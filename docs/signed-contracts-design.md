# Signed contracts v2 implementation contract

Baseline b75aa3e0ca6d649cafadf8450d9b5d606367dda4 / tree e51f8d10d4be81030a3220b8383121f2d45084c5 / product 0.2.0. Master was fetched and confirmed unchanged. The owner authorized execution of the complete supplied plan; its instructions are implementation guidance, not a claim of existing features. C01–C12 are preserved.

ADRs 021–026 govern profiles, signatures, credential policy, handoff/review, cleanup and cutover. Pure signing codec is packages/wos-core/signing; Application owns signed-command derivation and guards. Signing/private secret stores belong only to runtime/CLI composition. Existing wos/wosctl and one command catalog are retained.

Reserve migration 0020 for public signing keys/enrollments/credential and acceptance policies; 0021 for immutable issuances/returns/acceptances and review cases/contracts/decisions plus work markers. Reserve 0022 if explicit protocol metadata/index changes require it. Never modify migrations 0001–0019.

Bypass inventory: legacy claim/release/reclaim/renew/complete and admin completion; v1 SubmitWorkResult/FinalizeWorkContract; generic assessment/attestation/conclusion; material work/criterion/dependency edits; parent lifecycle closure; HTTP catalog/MCP/UI and public embedded Application. Signed protocol guards must live in Application/domain, with old receipt replay before phase rejection but current access checked first. Administrative interventions are explicit and audited, never fabricated signatures or normal review approval.

P00 fixtures are illustrative documents, not signed/tested envelopes. Executable signature vectors follow P01. R01–R24 and T01–T96 map to waves in original section 20.13; ROADMAP records actual per-wave verification. Direct return, review handoff, corrections and uncertain response are first-class fixtures. Direct means atomic done; delivered means executor responsibility closed while Task remains in_progress; correction means new authority.

Deferred extensions: voting/quorum, browser/delegated signing, remote KMS inside transactions, organizational independence without group binding, proprietary runtime integrations, scheduler/agent execution and claimed universal token savings. P12 supplies a controlled benchmark harness and reports unavailable provider telemetry instead of inventing costs. Publication remains separately gated, never inferred from a merge.

## Signed execution and review persistence foundation (P03, partial)

Migration 0021 adds separate signed execution, checkpoint/submission, review case/authority and immutable signed-fact tables. Existing v1 tables and migration checksums remain unchanged. Separate ports prevent `delivered` from being written into the legacy execution table, whose original status constraint remains intact. A delivered execution is terminal; the Task stays `in_progress`. Its review authority has a different Principal, execution, fence and lease. Expired/revoked/inconclusive review releases the case to pending; approved/changes_requested/cancelled/superseded cases remain terminal.

Storage checks exact byte digests, sizes, identity/scope, immutable targets and CAS. It does not assert that a detached proof is verified. Application must verify the signature with the expected purpose, server identity, registry role and live credential policy before deriving any signed mutation. Canonical payload bytes and detached proof are stored together; no unsigned companion command or persisted `verified` flag is authoritative. Signing specifications do not contain their own digest.

Review participant history includes execution/correction Principals and configured separation groups. It does not establish organizational independence or prevent administrator-created Sybil identities. Existing deployment-wide Outcome independence remains a stronger additional gate.

The generated signed execution adapters reuse v1 execution storage mechanics with separate maps/tables and explicit signed bindings. `tools/signedgen/generate.py` runs before PostgreSQL generation; CI checks generated outputs. v2 SQL content/lease versions use exact decimal text in SQLite and NUMERIC(20,0) in PostgreSQL. Existing v1 fingerprints/wire representations remain unchanged.

Tests exercise delivery without Task completion, terminal rejection, independent review, exact expiry, fenced takeover without TTL extension, immutable submission/spec/policy/round, rollback, unique open case, scoped pages/active counts, signed bytes across actual PostgreSQL clean restore and embedded unsigned task/assessment/parent completion bypasses. This foundation does not yet expose signed acquisition/return/review commands, configure persistent server trust, activate a Namespace or install workspace schema 2. Those are remaining implementation gates, not available features.
