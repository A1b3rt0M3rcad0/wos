# ADR-0021 — Independent profiles and minimal workspace

Status: accepted for implementation, 2026-10-08. Not an implementation claim.

ADR-019 remains the schema-1 historical contract. Schema 2 uses .wos/project.yaml and .wos/profiles/<name>/profile.yaml plus contract/<id>.yaml. Selection is explicit flag, process WOS_PROFILE, or the only unambiguous profile; no shared active-profile setting. A profile is a local destination/identity restriction, not authority. It contains secret references only, never token/private key bytes. Env and mounted-secret backends are explicit; OS keyring is preferred where available and failure never falls back to plaintext. Pending intentions are embedded in the profile before acquisition and in the contract afterward. No operational history directories. Interprocess locks use stable runtime identities (Unix advisory locks/Windows named mutex), not replaced YAML inodes. Cooperative locks need local content CAS to preserve edits by external editors. Profiles do not isolate processes running under the same OS account.

The owner authorized [P00–P13](../signed-contracts-implementation-plan.md). V1 behavior remains until explicit cutover. Native Windows, hosted CI, registry publication and deployed consumer pilots are separate evidence gates.


### Local destination integrity refinement (P07)

The minimal layout has no separate trust file. The selected profile therefore carries a standard HMAC-SHA256 integrity tag, keyed by its locally resolved API credential, over the exact origin, persistent server ID, Namespace, Principal, CredentialID, issuer pins, profile name and secret references. A separate domain-separated tag protects embedded pending intent bytes. Validate both tags before sending any credential over the network. Presentation preferences are not authenticated because they grant no authority. Changing a credential or identity requires explicit onboarding of a new binding; it never silently rebinds existing work.

This is client-side tamper detection, not an enrollment proof, domain authorization or additional server role. Possession of the bearer cannot replace a server signing key without the authorized enrollment challenge and Ed25519 possession proof. Processes able to obtain the credential under the same OS account can recompute the local tag; profiles are not an OS security boundary. First onboarding must independently approve the destination and issuer fingerprint. Standard library HMAC introduces no custom cryptographic primitive.

The file writer compares the complete previously read content before replacement and preserves observed editor changes. Stable runtime locks are implemented alongside P07 onboarding as a P08 foundation; they never lock a replaced YAML inode. Unix files remain private and are never unlinked. Windows global named mutexes cover service/interactive sessions and pin thread ownership; native platform verification is a separate gate. A noncooperating OS writer can race the final comparison/rename; this implementation does not claim a portable filesystem compare-and-swap primitive or isolation from another process under the same account.
