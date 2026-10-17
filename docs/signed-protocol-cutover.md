# Explicit activation of signed work

Schema-2 hosts use an explicitly approved operator profile. Provision the server
issuer outside the repository, enroll each agent's own key, and configure a
Tenant acceptance policy before activation. Tenant administration remains
required; agent execution permission cannot activate a Tenant.

```sh
wos protocol get --profile operator
wos protocol set --profile operator --phase draining_to_signed_v3 --version <exact-version> --reason 'retire v1 writers'
wos protocol preflight --profile operator
# Stop every old server replica and direct SQL writer; retire their DB credentials.
# Finish or explicitly revoke remaining valid unsigned execution authority.
wos protocol set --profile operator --phase signed_contracts_v3 --version <exact-version> --writers-drained --reason 'writers and SQL credentials retired'
wos protocol recover --profile operator
```

The preflight is a bounded advisory snapshot. Activation checks the live state
again in the same transaction as the transition, while holding the Tenant
protocol and affected Outcome guards. It requires no valid legacy claim or
unsigned execution contract, a matching persistent issuer and explicit Tenant
acceptance policy. Entering signed drain clears the previous writer attestation.
The operator must make a fresh acknowledgement. WOS cannot remotely certify
that unrelated SQL clients have been retired.

Each schema-2 change freezes its exact scope, phase, CAS version, reason and drain
acknowledgement in the profile before sending. After an uncertain response,
`protocol recover` reads the credential-bound original accepted operation first.
It never changes CAS, creates another intention or silently upgrades/downgrades.
An accepted operation is saved locally before clearing its pending marker.

Legacy history and accepted receipts retain their original encoding. New
unsigned acquisition is refused during signed drain. Signed activation advances
writer epoch to 3 and forbids downgrades. Historical authorized replay does not
create new authority. Mixed old/new database writers are unsupported.

New signed mutations use protocol 3 and bind `tenant_id`. Schema-2 refers to the
local CLI document format, not permission to send signed protocol 2. Historical
protocol-2 envelopes retain their original `namespace_id` fields and bytes for
read-only verification; never edit or re-sign them to imitate protocol 3.

For a previously signed service, first settle execution/review authority, open
review cases and pending client intentions on the old version. Then follow the
[explicit storage and Tenant cutover](scaling/tenants.md#upgrade-an-existing-installation).
This activation sequence is not a live migration of pending protocol-2 work.

This document covers server activation and recoverable schema-2 operator changes.
Workspace schema-1 conversion, issuer replacement and final release acceptance
remain separate implementation items recorded in ROADMAP.md.


Startup checks database compatibility before bootstrap or serving, including
when automatic migrations are disabled. A newer schema is refused under the
migration transaction, without changing migration history. This protects
subsequent binaries that contain the guard. It cannot add a guard to already
released binaries: retire old processes and their database credentials before
cutover, and verify the actual historical binary separately.

## Historical v0.2 upgrade and old writer acceptance

The following fixture and original results cover the earlier protocol-2 branch.
They preserve its original identifiers and do not qualify schema-per-Tenant or
protocol-3 cutover.

CI builds the original `b75aa3e0ca6d649cafadf8450d9b5d606367dda4` source,
creates an unsigned contract, immutable submission, criteria and seven original
command receipts through its HTTP API, then restores the closed SQLite database
or an actual PostgreSQL custom dump into a clean target. The current binary
checks the original rows, versions, digests and exact receipts before and after
explicit signed cutover. Activation with the old valid contract is refused; the
operator revokes it explicitly before activating v2. No historical document is
signed retroactively. The artifact records both source/binary hashes and backup
hashes. Reproduce with explicitly built binaries:

```sh
python3 tests/acceptance/historical_upgrade.py --binary bin/wos-upgrade-test \
  --old-binary /path/to/pinned-wos-v020 --source FULL_CURRENT_COMMIT \
  --report historical-upgrade.json
```

This test verifies the exact old binary's readiness rejection and incompatible
work acquisition refusal. WOS 0.2.0 predates the new startup guard: it can bind
an HTTP port against the newer database, but `/readyz` returns 503 and unsigned
acquisition rejects the unknown signed protocol without committing history.
This is not a claim that every route of an old executable is disabled. Retire
old writers, remove them from traffic and revoke their database credentials; a
new migration cannot retrofit an old executable's startup behavior. The current
binary separately refuses future schemas before bootstrap or serving. Native
Windows and restore with signed execution/review/correction pending have their
own gates and are not inferred from this historical fixture.
