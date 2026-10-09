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
Readiness compares each replica's configured public signer with the current
persistent issuer. A stale replica reports HTTP 503 after replacement. A restored
runtime without a configured signer also reports 503 if any Namespace has active
signed protocol, while liveness and authorized historical reads remain available.
No ephemeral replacement key is created; provision the approved signer before
routing new work to that replica.

Historical public keys remain available through authenticated bounded `trust`
queries, with `limit` and `cursor` and explicit search completion.

Client trust requires explicit approval of the replacement fingerprint while
retaining old pins. In an approved schema-2 workspace:

```sh
wosctl --profile executor profile trust --issuer-fingerprint sha256:<approved-new-fingerprint>
```

Use the actual profile name and complete independently approved fingerprint.
The command verifies the same origin, persistent ServerID, Namespace, Principal
and original CredentialID before one atomic profile update. It appends only the
new public pin and authenticated bounded binding lineage. Contract files,
original signatures, request IDs, idempotency keys and pending intentions remain
unchanged. Repeating approval reconciles the durable profile after interruption
without rewriting it. No network call runs under the profile publication lock.

At most ten issuer pins and nine prior binding records are retained per profile.
Reaching the bound requires explicit operator planning; the command does not
prune historical trust or discard uncertain work. Historical signatures remain
verifiable using retained public pins, including after losing the old issuer seed.
Proceed with `work send` for a prepared return, or `work recover` for a previously
sent uncertain return. Recovery never sends a prepared-only return automatically.
Do not re-sign uncertain work, edit issued payloads or delete pending intentions.

The actual HTTP/CLI journey covers rotation after local signing, refusal of
untrusted issuance, explicit approval without contract rewriting, new signed
acceptance, lost-response recovery, and independent review on Memory, SQLite
and PostgreSQL. Native child-process exit after profile publication also has a
regression fixture. These incremental checks do not replace final P12/P13 gates.
