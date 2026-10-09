# Explicit recovery of the instance issuer

This command is implemented in source; coordinated packaging and final release
acceptance remain P13. It is a host/deployment operation with direct database
access, never a tenant HTTP/MCP command. A Namespace administrator cannot replace
the issuer shared by other tenants.

Stop or drain serving replicas and retain a verified database backup. Provision
the new Ed25519 seed through the supported protected environment reference. Do
not put seeds in the public intention, project/profile YAML, arguments or logs.
Keep the existing Namespace policies and deployment acceptance floor.

Prepare `issuer-recovery.json` with the exact existing `server_id`,
`previous_issuer_id`, `previous_fingerprint`, a new UUIDv7 `new_issuer_id`, the
independently approved `new_fingerprint`, and `reason`. Obtain fingerprints through
a trusted operator channel, not an unapproved repository or endpoint. The new
key ID and seed must be new; historical IDs/fingerprints are not reused.

```sh
# The environment already contains the protected new seed reference and database config.
wos issuer recover --file issuer-recovery.json
# Repeating the identical file reconciles its original receipt.
```

The command verifies replacement possession before the transaction. It opens
storage without starting transports, bootstrapping credentials or silently
creating a new server identity. It compares the predecessor, persists old/new
public keys and an immutable receipt, and replaces only the current issuer.
`server_id` and instance creation time remain stable. A changed predecessor/key/
fingerprint/reason is a conflict; an old replay never reinstalls a superseded key.
The private old seed is unnecessary and no historical fact is re-signed.

Restart replicas with the approved replacement reference. Normal runtime startup
continues to reject silent seed replacement. Shared issuer reads and exclusive
replacement serialize current issuance; stale signer configuration fails closed.
Historical public keys remain available through authenticated bounded `trust`
queries, with `limit` and `cursor` and explicit search completion.

Client trust still requires explicit approval of the replacement fingerprint
while retaining old pins. **The automatic or interrupted-safe CLI pin-update
workflow is not implemented yet.** Existing profiles refuse an untrusted new
issuer; do not bypass their binding MAC, re-sign an uncertain return, edit issued
payloads or discard pending original-CID intentions. Client trust update and its
pending-return recovery acceptance remain required before claiming complete
operator recovery/P10 readiness.
