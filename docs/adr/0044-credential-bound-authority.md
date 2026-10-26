# ADR-044 — Credential-bound legacy lease authority

Status: accepted for implementation by the owner's 2026-10-21 authorization of the operational plan; qualification and rollout remain pending.

## Context

The operational plan tightens legacy lease ownership and replay disclosure to
an authenticated credential. The current identity model already distinguishes
Principal, Credential, ActorRef, SigningKey, profile, contract, execution, and
review authority. The new lease binding must not collapse those identities or
turn assignment into execution authority. Signed contracts already bind a
specific credential and cryptographic identity; those accepted rules must not
be rewritten as a legacy credential lease.

Relevant constraints are recorded in ADR-0014 through ADR-0016, ADR-0018, and
ADR-0021 through ADR-0027. This decision extends legacy lease authority only; it
does not supersede the signed protocol or authorize a Namespace cutover.

## Proposed decision

### Authorization is cumulative and fails closed

An operation is authorized only when every applicable condition holds:

```text
current Principal grant
∩ current authenticated Credential and session policy
∩ permitted Namespace/Outcome scope
∩ capability of the selected protocol
∩ current execution or review authority
∩ current domain preconditions
```

Principal, Credential, ActorRef, role template, SigningKey, contract and
session remain distinct. Metadata supplied by a client, including a tenant or
credential identifier, is not authentication. Reading capabilities grants no
authority. Roles are versioned templates expanded to concrete permissions;
the caller cannot exercise permissions absent from either its Principal grant
or Credential policy. Unknown commands or permissions fail closed. New
credentials do not receive wildcard rights, and legacy grant expansion is
finite, versioned, and cannot silently grant newly introduced capabilities.

Compound operations require the permissions for each actual effect. For
example, creating a Task and its Criteria requires both creation and Criterion
definition rights; assessing or accepting proof is separate from submitting
work. A broad administrator role does not forge a signed identity, bypass
criteria, or erase separation-of-duty requirements.

### Legacy leases bind to the canonical authenticated credential

For a direct API credential, the lease holder is that authenticated Credential.
For browser sessions, resolve the root Credential through the server-side
session relationship and validate the complete parent chain, Principal,
Namespace, ActorRef, and revocation state. Store the root credential as the
legacy lease holder. Store a child session Credential separately as acquisition
provenance for audit. A client-supplied credential ID can never choose or
replace the holder identity.

Two authorized sessions derived from the same root Credential may continue that
credential's legacy lease. Two different Credentials belonging to the same
Principal are different holders and cannot renew, release, deliver, or recover
each other's authority solely because their Principal matches. This does not
change any existing Task assignee cardinality. Assignment expresses
responsibility; it neither reserves the Task nor transfers an active lease.

Preserve the existing legacy lease identity and fencing fields, including
Principal, ActorRef, ClaimID, execution identity, FencingToken, acquisition time,
and expiry. Add a credential binding and optional acquired-session provenance
without substituting a new concept for the lease. New signed contracts keep
their own immutable `SignedContractBinding.CredentialID`; do not rewrite that
binding to a browser root Credential.

### Lease transitions retain fencing, server time, and explicit recovery

Acquire/claim rechecks readiness, scope, live grants and Credential policy,
current domain gates, and absence of another valid authority in the same
transaction that reserves the Task. Renew and release require the same
canonical credential, Principal, claim/execution, and fencing generation,
unless a separately authorized intervention applies. Reclaim is allowed only
after expiry or explicit revocation of that authority and creates a new claim
and a strictly greater fencing token. Transfer is never implicit; any future
transfer must end the old authority and create an auditable new generation.

Use the transaction's authoritative clock after the required security and
Outcome coordination locks. `expires_at == now` is expired. Client time and a
timestamp captured before waiting on a lock cannot preserve an expired lease.
Preserve the existing lease duration defaults and bounds: 300 seconds default,
30 seconds minimum, and 3,600 seconds maximum. A per-lease duration is not a
maximum continuous work session; this decision introduces no total renewal
limit. Passage of time affects projections, not persisted Task lifecycle.

Credential rotation creates a restricted replacement with explicit overlap; it
does not transfer an active lease by default. Revocation denies future
mutations/replays under that Credential immediately at the transaction's
revocation ordering, but does not pretend to stop an external process or
silently free its still-valid lease. Releasing a revoked holder's authority is
an explicit audited intervention/recovery that advances fencing before a new
worker can act. Signed-key rotation and signed-contract recovery continue to
follow their own protocol.

### Receipts are protected by the authority that created them

Protected operational receipts bind the authenticated Principal, canonical
Credential, scope, command, idempotency intention, normalized payload
fingerprint, and original result. Reauthorize the live credential and scope
inside the coherent transaction before disclosing a protected replay. A second
Credential of the same Principal cannot retrieve the first Credential's lease
or its original protected result merely by knowing an idempotency key.

Extend receipt/binding metadata additively and version it. Preserve previously
published intent fingerprints, encodings, signed receipts, and historical
results; do not silently replace their deduplication identity or fabricate
credential provenance for old records. When a legacy replay has no reliable
historical Credential binding, retain its original compatibility semantics
only where already authorized and do not present it as a v2 credential-bound
result.

Signed protocol receipts remain governed by their existing signed durable
result contract: they bind the original Credential and signed operation,
preserve original bytes and proof, reauthorize current access before protected
replay, and never acquire or extend authority during replay. This decision
does not change signed payloads, digests, DSSE envelopes, signer enrollment,
review independence, signing-key rotation, or accepted return/review flows.

### Persistence and concurrency use the existing transaction model

Persist credential binding using the existing UoW, Outcome guard, optimistic
versions, immutable events/outbox, and fencing model. PostgreSQL preserves the
established Namespace/security/protocol lock order before the Outcome guard
and uses database transaction time; SQLite preserves writer acquisition and
adapter semantics. Memory remains a test adapter and is not evidence of
multi-process concurrency. Do not add process-local global locks, Redis locks,
or another authority store.

Transactions remain short. They never encompass agent execution, remote
HTTP/MCP calls, Artifact URI fetching, or external runtime work. A failed
compound operation rolls back as one unit; an uncertain client result
reconciles the exact original intention and never silently claims a new lease.
Independent Tasks can still execute concurrently after their short state
mutations commit.

## Consequences

- Different API keys for one Principal no longer share legacy lease authority;
  sessions derived from the same canonical root Credential can still operate
  on that Credential's lease when authorized.
- The product must show assigned responsibility separately from the current
  lease holder and must explain expiry/recovery without claiming process
  presence.
- Credential rotation and revocation become explicit operational events;
  neither is a lease transfer or proof that external work stopped.
- Database migrations, receipt compatibility, web-session resolution, all
  transports, and real PostgreSQL concurrency require qualification before
  rollout.
- Signed protocol authority is unchanged. Legacy lease binding cannot be used
  as a fallback or substitute for signed execution/review authority.

## Rejected alternatives

- Bind a lease only to Principal: distinct credentials of one Principal would
  share authority and replay visibility.
- Bind browser leases to transient session credentials: a browser refresh
  would change ownership even though the root credential did not change.
- Accept `credential_id` from a request payload as the holder: this is
  caller-controlled identity, not authenticated provenance.
- Treat assignment, profile name, ActorRef, or role as execution authority:
  each expresses a different fact or restriction.
- Transfer a lease automatically during rotation, revocation, or assignee
  change: the old external process could still act and stale fencing would be
  unsafe.
- Rewrite signed bindings to match legacy root credentials or admit browser
  signing: this would break the immutable signed protocol and its trust model.
- Add a process mutex or separate distributed lock service: neither replaces
  the existing transaction, Outcome guard, and fencing guarantees across
  replicas.

## Verification required before rollout

Before rollout, test direct credentials and browser root/session
resolution, same-root sessions, distinct credentials under one Principal,
revocation races, rotation without implicit transfer, exact expiry at lock
wait, monotonically increasing fencing, protected replay after cache expiry,
historical compatibility, signed protocol invariance, and multi-process
SQLite/PostgreSQL concurrency and restore. Preserve existing signed protocol
goldens, receipts, review independence, key lifecycle, and P00–P13 evidence.
The canonical design and `AGENTS.md` record the target contract; `ROADMAP.md`
records actual implementation and qualification. This ADR does not claim the
capabilities already exist.
