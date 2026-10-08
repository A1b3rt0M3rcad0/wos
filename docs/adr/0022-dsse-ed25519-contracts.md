# ADR-0022 — Versioned DSSE Ed25519 contract protocol

Status: accepted for implementation, 2026-10-08. Not an implementation claim.

Signed protocol 2 uses Ed25519 via crypto/ed25519, DSSE 1.0.2 pre-authentication encoding and RFC8785 JCS. Payload kinds are spec, authority, work return, review return, acceptance receipt and key enrollment, each with a distinct application/vnd.wos.*.v2+json type. Exactly one signature is admitted. The signed payload includes signer_key_id and expected identity/scope; outer keyid only locates an authorized registered key. Decode rejects duplicates/unknown fields/noncanonical payload bytes. Application derives its command from verified bytes, never an unsigned companion object or caller-supplied verified flag. Spec/grant signatures are separate. Spec, authority and request digests have no self-referential fields. V1 digest/fingerprint semantics stay unchanged. Public pure codec has no SQL, HTTP, keyring or private-key serialization. Payload cap 180 KiB, total request 256 KiB, local document 1 MiB with separately bounded technical blobs.

The owner authorized [P00–P13](../signed-contracts-implementation-plan.md). V1 behavior remains until explicit cutover. Native Windows, hosted CI, registry publication and deployed consumer pilots are separate evidence gates.
