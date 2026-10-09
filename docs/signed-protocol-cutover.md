# Explicit activation of signed work

Schema-2 hosts use an explicitly approved operator profile. Provision the server
issuer outside the repository, enroll each agent's own key, and configure a
Namespace acceptance policy before activation. Namespace administration remains
required; agent execution permission cannot activate a Namespace.

```sh
wosctl protocol get --profile operator
wosctl protocol set --profile operator --phase draining_to_signed_v2 --version <exact-version> --reason 'retire v1 writers'
wosctl protocol preflight --profile operator
# Stop every old server replica and direct SQL writer; retire their DB credentials.
# Finish or explicitly revoke remaining valid unsigned execution authority.
wosctl protocol set --profile operator --phase signed_contracts_v2 --version <exact-version> --writers-drained --reason 'writers and SQL credentials retired'
wosctl protocol recover --profile operator
```

The preflight is a bounded advisory snapshot. Activation checks the live state
again in the same transaction as the transition, while holding the Namespace
protocol and affected Outcome guards. It requires no valid legacy claim or
unsigned execution contract, a matching persistent issuer and explicit Namespace
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
writer epoch to 2 and forbids downgrades. Historical authorized replay does not
create new authority. Mixed old/new database writers are unsupported.

This document covers server activation and recoverable schema-2 operator changes.
Workspace schema-1 conversion, issuer replacement and final release acceptance
remain separate implementation items recorded in ROADMAP.md.


Startup checks database compatibility before bootstrap or serving, including
when automatic migrations are disabled. A newer schema is refused under the
migration transaction, without changing migration history. This protects
subsequent binaries that contain the guard. It cannot add a guard to already
released binaries: retire old processes and their database credentials before
cutover, and verify the actual historical binary separately.
