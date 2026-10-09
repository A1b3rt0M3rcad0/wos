# ADR 028: host-managed issuer recovery with immutable public history

Status: accepted; implementation undergoing complete regression and integration.

The accepted signed-contract plan requires issuance after a restore that no
longer has the previous private seed (T14/T87). Normal runtime startup refuses a replacement seed; recovery requires a separate
host operation. Namespace administration
must not gain control of the instance issuer shared by other tenants.

Implement recovery as a host/deployment operation with database access and a
protected new local signer, outside HTTP/MCP. Require the exact persistent
ServerID, previous issuer ID/fingerprint, declared new issuer ID/fingerprint and
reason. Verify possession of the replacement seed before opening the transaction.
Never require the old private seed after a restore or overwrite old signed facts.

Persist the previous and new public identities and an immutable recovery receipt
in a transaction. Preserve ServerID and instance creation time. Replacement
compares the current issuer exactly; repeat of the same frozen recovery returns
its existing receipt. A changed reason, key, fingerprint or predecessor is a
conflict. A global issuer lock must order issuance against replacement, so an old
replica cannot emit a new accepted fact after cutover. Namespace policies and
historical agent/issuer key records remain intact.

Public trust queries expose bounded history without private material. Clients
require explicit out-of-band trust approval to append the replacement pin and
retain historical pins. Pending original-CID returns must remain recoverable
without re-signing or changing the frozen request. Profile trust updates use one atomic append with authenticated bounded prior
binding lineage; existing contracts and frozen intentions are never rewritten.

Acceptance requires Memory/SQLite/PostgreSQL CAS/replay, restore without the old
seed, verification of old specs, issuance using the approved replacement,
rejection by a stale runtime, client pending-receipt recovery, and full native
platform gates. This ADR does not waive old-writer retirement or final P12/P13
source-specific acceptance.
