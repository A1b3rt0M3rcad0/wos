# Signed contracts v2 implementation contract

Baseline b75aa3e0ca6d649cafadf8450d9b5d606367dda4 / tree e51f8d10d4be81030a3220b8383121f2d45084c5 / product 0.2.0. Master was fetched and confirmed unchanged. The owner authorized execution of the complete supplied plan; its instructions are implementation guidance, not a claim of existing features. C01–C12 are preserved.

ADRs 021–026 govern profiles, signatures, credential policy, handoff/review, cleanup and cutover. Pure signing codec is packages/wos-core/signing; Application owns signed-command derivation and guards. Signing/private secret stores belong only to runtime/CLI composition. Existing wos/wosctl and one command catalog are retained.

Reserve migration 0020 for public signing keys/enrollments/credential and acceptance policies; 0021 for immutable issuances/returns/acceptances and review cases/contracts/decisions plus work markers. Reserve 0022 if explicit protocol metadata/index changes require it. Never modify migrations 0001–0019.

Bypass inventory: legacy claim/release/reclaim/renew/complete and admin completion; v1 SubmitWorkResult/FinalizeWorkContract; generic assessment/attestation/conclusion; material work/criterion/dependency edits; parent lifecycle closure; HTTP catalog/MCP/UI and public embedded Application. Signed protocol guards must live in Application/domain, with old receipt replay before phase rejection but current access checked first. Administrative interventions are explicit and audited, never fabricated signatures or normal review approval.

P00 fixtures are illustrative documents, not signed/tested envelopes. Executable signature vectors follow P01. R01–R24 and T01–T96 map to waves in original section 20.13; ROADMAP records actual per-wave verification. Direct return, review handoff, corrections and uncertain response are first-class fixtures. Direct means atomic done; delivered means executor responsibility closed while Task remains in_progress; correction means new authority.

Deferred extensions: voting/quorum, browser/delegated signing, remote KMS inside transactions, organizational independence without group binding, proprietary runtime integrations, scheduler/agent execution and claimed universal token savings. P12 supplies a controlled benchmark harness and reports unavailable provider telemetry instead of inventing costs. Publication remains separately gated, never inferred from a merge.
