# Qualifying a WOS client

WOS records state and coordinates authority. An external application or agent
executes work and decides when to request an explicit evaluation/conclusion.
Successful source-event registration does not qualify signed execution.

The original [client report](audit/client-operational-report-2026-10-12.txt)
describes documentary integration on source `2b8b6664afc18197c30b192e3c1a59164b7b698b`.
Its private application reports and desktop `localhost:8092` are separate from
the reproducible repository qualifications below. No application-specific rule
or inferred assessment is added to the Core.

## Select the protocol and identities

Build the current source for the central workflow; published `wosctl@0.4.2` implements the earlier interface. Use `wos --help`, `wos help work`, and `wos catalog --output json` before supplying credentials. The local catalog reports its schema version and executable version/commit. `wos version --output json` identifies the binary used for a run.

Follow [central installation and protected profiles](signed-workspace-profiles.md). Initialize once, verify ServerID/issuer pins through an independent channel, and enroll named identities with protected token/private-key references. Pin `--profile NAME` for each worker; no current project directory is needed. `init`, `profile create`, and `skill install` never activate a Namespace protocol or grant permission. Protocol activation and preflight require an authorized administrator; do not grant administration to an executor just to make diagnostics pass.

Execution is `work next → work context → external execution → typed log/evidence/artifact/result commands → work check → work finish`. Independent review uses a separate eligible Principal and the corresponding typed `review` commands. A different profile, credential or key alone creates no independence. Findings reference the exact issued requirements; corrections need an explicitly acknowledged original review and new authority. Preserve issued specs, frozen intentions and receipts. Never edit internal documents, silently change CAS/fences, or replace an uncertain request.

`work history --work TASK_ID --outcome OUTCOME_ID` survives local cleanup. History is bounded with revision/order-bound cursors. `outcome context` is not a CLI verb; use the issued focal `work context` and expand relevant references. The server's current attention projection is separate from the issued specification and local draft readiness.

For historical schema-1 bindings, explicitly use `wos --workspace PATH init ... --credential-env WOS_TOKEN` and the matching released guide. That compatibility mode does not create central profiles or signed authority. See [legacy conversion](workspace-migration-v2.md) before moving old material.

## Confirm commits, then redeliver safely

An external event should have a stable producer/event identity. Derive one
idempotency key from a serialized tuple of Namespace, Outcome, producer, event ID
and operation (for example a SHA-256 digest with a versioned prefix). Persist the
exact payload and key **before** sending. An event ID reused with different
content must raise a conflict; generating another key to conceal it is unsafe.
WOS does not globally deduplicate every equal `source_ref` or promise unlimited
idempotency retention.

If sending times out or the response is incomplete, record uncertainty. A GET
showing evidence or a HTTP status does not prove that this intention committed.
Replay the identical payload, original key and expected versions. Save the JSON
result's `command_id`, `outcome_revision`, entity ID and destination. When the
command ID is known, the SDK provides:

```go
observation, err := client.GetCommandReceiptObservation(ctx, namespaceID, commandID)
// Check err before interpreting observation. The server authenticates the
// Principal and the SDK validates command ID and positive commit revision.
if err == nil && observation.ResultOmitted {
    // Commit confirmed; complete result is unavailable in this bounded reply.
    // Preserve the compact receipt and query the specific referenced state.
}
```

`GetCommandReceipt` remains the full-response compatibility method; it returns
an explicit error for compact results and points to the observation API.
Standard Go HTTP transport is prevented from silently resending mutation bodies.
The SDK creates no new retry intention. Custom caller-provided transports must
also preserve this rule. Keep confirmed receipts in the client's durable queue;
an expired server cache or a missing receipt is not proof of non-commit. Define
retention/reconciliation before retrying beyond the supported cache horizon.

For ordinary documentary commands, renewed credentials of the same authorized
Principal/Actor can recover the original result, while the revoked token is
rejected. This does **not** transfer a signed contract to another credential ID:
signed obligations retain their original CID/Principal/key/fence binding. Follow
the signing rotation/recovery procedure and inspect the original durable
acceptance; do not substitute another profile to bypass it.

## Keep observations, evaluations and conclusions separate

`RegisterEvidence` records an observation. `CreateEvidenceLink` records a stance
such as supports, contradicts or context. Neither creates an assessment or
achieves an Outcome. A required criterion needs an explicit authorized
assessment for its current revision and evidence. Even a `met` assessment leaves
the Outcome active until an authorized explicit `AchieveOutcome` passes all
current gates. WOS does not evaluate the truth of an external payload.

The generic HTTP acceptance test demonstrates this sequence with an explicitly
labeled fixture evaluation. It asserts that evidence/link-only conclusion fails,
that assessment alone does not achieve, and that an explicit conclusion succeeds.
Clients must not present documentary delivery as independent work approval.

## Qualify the actual operational target

Probe `/livez` and `/readyz` for the **selected instance**. Then authenticate the
selected profile (`auth status`), inspect its signed identity/key (`doctor`),
read its Namespace protocol, and inspect a real acquired contract via
`work show ID --for-agent` or `review show ID --for-agent`. Inspect effective
status, expiry, current execution/fence and pending local intentions. A running
container or successful HTTP health probe alone does not certify acquisition or
review. `doctor` diagnoses identity/key/destination; it is not a production SLA,
backup certificate or claim that every pending server operation is known locally.

With the owning profile, `work recover --all-pending` and `review recover
--all-pending` reconcile the existing durable intentions and receipts. These are
explicit recovery actions, not new-work discovery probes. Retain original keys
and pending state if transport or local persistence remains uncertain.

SQLite must persist in the named Compose volume. Back up with `wos db backup
FILE` and restore to a **new** file using `wos db restore SOURCE DESTINATION`.
For PostgreSQL, use the database-version-compatible dump/restore tools and
protected credential references; test a clean isolated target. Check pending
execution, review and correction after restore, including original signed bytes,
versions/fence, receipt recovery and outbox/delivery identities. Never restore
over an active writer or claim production RPO/RTO from a test alone.

Observe metrics, conflicts, expired leases, unresolved pending intentions and
outbox retries under the deployment's retention/resource limits. The operating
procedures are in [work-contract operations](work-contract-operations.md).
No provider billing, production load envelope or manual human validation is
inferred from these functional checks.

## Reproducible verification

Supply a disposable real PostgreSQL DSN and the owned Docker container name for
restore tests; the PostgreSQL cases must actually run, not skip.

```sh
go test ./tests/acceptance -run 'TestClientEvidenceCommitLossRotationAndExplicitAchievement|TestSignedPublicEnrollmentHTTPAcquireMCPReturnSDKRead' -count=1
go test ./packages/wos-sdk-go -run TestCommandReceipt -count=1
go test ./packages/wos-cli -run 'TestHelp|TestOutcomeContext|TestUnknownHelp|TestJournal|TestReceiptCleanup|TestSDKRefuses' -count=1
go test ./packages/wos-core/storage/sqlite ./packages/wos-core/storage/postgres -run 'TestSignedReviewCorrectionJourneyWithoutExecutorAuthority|TestSignedContractsUseDatabaseAuthorityDespiteReplicaClockSkew|TestHostIssuerRecoveryPreservesHistoryIdentityAndReceiptAcrossRestore' -count=1
```

The real HTTP fixture runs on Memory/SQLite/PostgreSQL, loses the response **after
commit**, revokes the old token, replays through a same-Principal credential,
validates the original receipt and asserts exactly one evidence. It also rejects
changed semantic content under the same key and tests the explicit conclusion
sequence. Signed transport qualification crosses HTTP, MCP and SDK. Native
distribution qualification executes installed binaries outside the checkout
through execution, independent review, correction and recovery. Native Windows
and Linux receipts belong to their exact release source; cross-compilation is
not a substitute. Record source SHA, protocol, binary/version, target, executed
cases, skips/failures and limitations together.
