# Tenant addressing and signed protocol 3 compatibility review

This branch makes the owner-approved breaking change recorded in
[ADR 044](../adr/0044-tenant-schemas-and-protocol-cutover.md). The release queue
contains a **major** entry in
[tenant-schema-isolation.json](../../.changes/tenant-schema-isolation.json).
Neither this review nor its schema fingerprints publish a release or imply that
the complete PR qualification has passed. [ROADMAP](../../ROADMAP.md) records
implementation and verification status.

The reference is the actually released
[v0.4.2](https://github.com/A1b3rt0M3rcad0/wos/releases/tag/v0.4.2), source
`b572269085494ddd4d6370716ff1cb360b46065f`.
[Its frozen files](contracts/v0.4.2/) and historical fingerprint manifest remain
byte-for-byte unchanged. The earlier [operational review](operational-contract-review.md)
records additive changes before this breaking Tenant transition; it is historical
context, not a claim that old clients can submit current mutations.

## Explicitly reviewed changes

| Contract | Current behavior | Compatibility boundary |
| --- | --- | --- |
| Addressing | `Tenant`, `tenant_id`, `/tenants/`, Tenant command/type names and `tenant:admin` | Earlier Namespace names are not incoming mutation aliases |
| Signed protocol | Protocol 3, v3 payload purposes, `signed_contracts_v3`, `draining_to_signed_v3`, writer epoch 3 | Historical protocol-2 proofs remain verifiable in their original encoding; protocol-2 requests cannot authorize new mutations |
| Local Profiles | Explicit authenticated `profile migrate-tenant` conversion | Frozen or pending old intentions must be resolved with compatible clients before cutover; migration never edits a signature or invents acceptance |
| SQL placement | One complete Tenant per schema, catalog lifecycle and fixed-schema runtime | Migration 0033 preserves earlier migration checksums and immutable bytes; shared legacy state requires the explicit offline import |
| Provisioning | Protected host catalog commands allocate schemas | The old application-level creation path cannot add another Tenant to a bound data schema |

The current HTTP prefix remains `/api/v1`; product SemVer and the explicit
protocol discriminant carry this coordinated major break. Keeping that prefix
does not promise old field names or old clients will work. Match the server, CLI,
SDK and Skills source and follow the
[Tenant cutover procedure](../scaling/tenants.md#upgrade-an-existing-installation).

Local workspace document schema 2 remains distinct from signed protocol 3.
Existing UUID, counter, size, nesting, required-input, expiry, CAS, fencing,
independent-review and original-receipt semantics remain in force. Names and
placement do not grant authority or change Outcome acceptance semantics.

## Exact schema gate and reviewed projection

[public-contract.json](public-contract.json) freezes the nine current generated
OpenAPI/CLI schemas. The hash check normalizes **only** OpenAPI product
`info.version`; Tenant identifiers, protocol constants, purposes, constraints and
all other bytes remain part of each fingerprint. A later change still requires
an explicit review and fingerprint update.

The separate [release compatibility test](../../tools/releases/compatibility.test.mjs)
first verifies the original manifest hash and each available immutable released
schema hash. It then projects the released schema **in memory for comparison**:

- Rename the reviewed Namespace identifiers, routes, component references,
  command/permission names and their descriptive text to Tenant terminology.
- Map only the known WOS signed v2 purpose names to their v3 equivalents and the
  two signed protocol phase names to v3.
- Change `const: 2` to `const: 3` only at a `protocol_version` property. Local
  workspace versions, expected protocol versions and unrelated numeric values
  are retained.

This projection is not a request adapter, a migration algorithm or a way to edit
historical data. No released file, original signed envelope or pending intention
passes through it in production.

The ordinary additive comparison still rejects removal of every projected old
property or enum value, newly required inputs, changed old constraints and new
constraints on existing schema nodes. It preserves the projected original routes,
operations, path parameters, responses and annotations. The already-reviewed
central-workflow exception remains limited to its five optional signed-state
query filters. The signing-identity response retains its previously reviewed
annotation exception, with original query parameters and response-status keys
still checked.

Persisted local proof objects may list both v2 and v3 envelope purposes because
they retain historical material. New mutation envelopes advertise v3. An enum
entry alone does not authorize execution; the cryptographic and application
checks below enforce that distinction.

Negative controls prove the comparison still fails after a removed unrelated
field, a newly required field, a tighter existing bound, an extra constraint or a
removed enum value. They also prove the projection does not change local schema
2 or an unknown/unrelated v2 media type. The test additionally requires this
review's manifest marker, ADR reference and queued major version change.

## Semantic qualification remains separate

Static shape and hash checks cannot establish authority or safe migration. The
corresponding executable tests include:

- [Historical signing vectors](../../packages/wos-core/signing/historical_test.go):
  independently signed v2 bytes remain verifiable, cannot execute through the v3
  verifier, and cannot mix purposes, versions or ambiguous identity fields.
- [SQL cutover](../../packages/wos-core/storage/sqlite/tenant_migration_test.go):
  immutable signed bytes remain exact, only mutable configuration changes, and
  unsettled old authority prevents the rename before DDL.
- [Profile migration](../../packages/wos-cli/tenant_profile_migration_test.go):
  original binding/MAC checks, refusal of pending or tampered state and preservation
  of historical proof and unsigned backup bytes.
- [PostgreSQL placement import](../../packages/wos-core/storage/postgres/tenant_import_test.go):
  real migration 32→33, exact proof/row preservation, Tenant closure, source write
  fencing, refusal of RLS and unsupported sources, and atomic rollback after
  a later Tenant fails.
- [Catalog lifecycle](../../packages/wos-core/storage/postgres/tenant_catalog_test.go)
  and [hosted runtimes](../../packages/wos-api/internal/server/tenancy_runtime_test.go):
  schema/credential/session isolation, connection and lifecycle guards, deletion
  boundaries, explicit suspended-schema migration and shared-host admission.
- [Native operator commands](../../packages/wos-api/operations/tenants/command_test.go):
  versioned lifecycle, explicit drain acknowledgment, current catalog selection,
  source checks before import and secret-safe errors.

These tests require their declared real PostgreSQL and platform prerequisites;
a skipped test is not a pass. Source-bound normal/race and integrated results
must be reported with their actual commit or source hashes. Earlier scalability
measurements remain evidence for their original executable, not for this
contract change or a production capacity guarantee.
