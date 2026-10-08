# ADR-0021 — Independent profiles and minimal workspace

Status: accepted for implementation, 2026-10-08. Not an implementation claim.

ADR-019 remains the schema-1 historical contract. Schema 2 uses .wos/project.yaml and .wos/profiles/<name>/profile.yaml plus contract/<id>.yaml. Selection is explicit flag, process WOS_PROFILE, or the only unambiguous profile; no shared active-profile setting. A profile is a local destination/identity restriction, not authority. It contains secret references only, never token/private key bytes. Env and mounted-secret backends are explicit; OS keyring is preferred where available and failure never falls back to plaintext. Pending intentions are embedded in the profile before acquisition and in the contract afterward. No operational history directories. Interprocess locks use stable runtime identities (Unix advisory locks/Windows named mutex), not replaced YAML inodes. Cooperative locks need local content CAS to preserve edits by external editors. Profiles do not isolate processes running under the same OS account.

The owner authorized [P00–P13](../signed-contracts-implementation-plan.md). V1 behavior remains until explicit cutover. Native Windows, hosted CI, registry publication and deployed consumer pilots are separate evidence gates.
