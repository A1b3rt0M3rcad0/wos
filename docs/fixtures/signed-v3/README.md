# Protocol-3 interoperability fixture

The [independent DSSE/Ed25519 vector](../../../packages/wos-core/signing/testdata/dsse-vector-v3.json) contains the exact canonical payload, DSSE pre-authentication encoding, public key, signature and payload digest for the version-3 work-return purpose. Python cryptography 50.0.0 produced this vector from the public RFC8032 example seed. It tests the codec, not authorization or a complete operational return.

The [version-2 vector](../../../packages/wos-core/signing/testdata/dsse-vector.json) remains unchanged. Automated tests verify it through the explicit historical reader, preserve its bytes and proof, and reject it at the current mutation boundary. Historical records are never re-signed to rename a scope.

Current issued bindings use `tenant_id`, `protocol_version: 3` and version-3 purpose strings. The [codec contract](../../../packages/wos-core/signing/README.md) describes strict mutation decoding and historical read projections. The older `signed-v2/` directory contains historical design placeholders with deliberately invalid proofs; those documents remain examples, not executable acceptance evidence.
