# Signed workspace profiles (schema 2)

The administrator supplies the persistent server ID, issuer fingerprint, Namespace, Outcome, credential and authorized enrollment. Agent profiles contain secret references only. Onboarding does not grant permission on the server.

```sh
wosctl init --workspace-schema 2 --server https://wos.example.test \
  --server-id SERVER_UUID --namespace NAMESPACE_UUID --outcome OUTCOME_UUID
wosctl profile create executor_a --token-stdin --generate-signing-key \
  --enrollment ENROLLMENT_UUID --server https://wos.example.test \
  --server-id SERVER_UUID --issuer-fingerprint sha256:APPROVED_DIGEST
wosctl --profile executor_a auth status --output json
wosctl --profile executor_a profile inspect --output json
```

Provide the token through the host's protected standard input. No token or private-key flag accepts a secret value. Generated seeds and imported API credentials go only to the OS keyring. If its native credential service is unavailable, creation fails; configure explicit env or protected mounted references instead. Do not place administrative credentials in an executor profile.

A preprovisioned seed can be imported with `--signing-key-stdin --credential-ref env:WOS_TOKEN` instead of `--generate-signing-key`. The dedicated stdin accepts only a canonical base64 Ed25519 seed. Token and signing seed cannot share stdin in one invocation. Existing keyring entries survive interrupted provisioning; a different entry is rejected rather than overwritten.

For env/mounted secrets, supply a schema-2 profile proposal with `profile create NAME --file proposal.yaml` and the same explicit server approval flags. The generated [profile schema](../packages/wos-cli/schemas/profile-v2.schema.json) describes its fields. Provide an authorized `--enrollment` to register a new key, or an already registered active key whose fingerprint matches the selected private reference. Proposal references never transmit private key bytes to WOS. Env is provided by the host; mounted files must be outside the workspace, canonical regular files without links, and deny other accounts access. Windows uses a restricted DACL check whose native verification remains pending.

Selection is `--profile`, then process `WOS_PROFILE`, then the single profile. Multiple profiles without a selector fail with `profile_required`. `--workspace` takes precedence over `WOS_WORKSPACE`. No shared active-profile file is written. Profile names require exact case; case collisions and Windows reserved names are rejected.

The operational layout is:

```text
.wos/project.yaml
.wos/profiles/executor_a/profile.yaml
.wos/profiles/executor_a/contract/CONTRACT_UUID.yaml
```

Enrollment recovery is `wosctl --profile executor_a profile recover`. The frozen intention is authenticated and persisted before a network mutation. An unknown response remains `sent_unknown`; replay uses the original key and command, including original CAS values. Do not edit pending bytes or substitute another challenge. Server receipts make enrollment replay durable. Work acquisition uses its own durable frozen intentions, described below; enrollment is not reused as work authority.

After an operator explicitly recovers the instance issuer, approve its independently
verified replacement with `wosctl --profile executor_a profile trust --issuer-fingerprint sha256:<approved-fingerprint>`.
This appends public trust atomically without rewriting contracts or frozen returns;
old pins remain for historical verification. See [issuer recovery](issuer-recovery.md)
for host access, bounded trust history and interruption handling.

`profile inspect` and `auth status` return compact identity, selected public key status and credential restrictions. They are observations; each domain command rechecks current grants, credential policy, scope and authority. `doctor` verifies destination/issuer and local signing-key fingerprint without revealing secret values. It does not acquire or renew work.

`profile remove` refuses pending intentions and nonempty contract directories. It removes only known local profile files and empty directories, preserves shared secrets, and does not revoke a remote credential or contract. A changed API credential or profile destination needs explicit new onboarding; local integrity tags do not silently rebind existing work.

Stable locks live outside the operational workspace. Unix lock files are private, are never unlinked and release on process death. Windows uses global named mutexes across interactive/service sessions and fails closed if unavailable; cross-compilation is not native certification. The file writer preserves editor changes observed before replacement. A noncooperating OS writer can race the final comparison/rename; profiles do not isolate hostile processes under the same OS account.

P08 review checkout, operational lease maintenance and comprehensive file/process failure recovery remain in progress. P09 sign/send/finish is not implemented yet. A schema-2 project rejects legacy substitutions. The project does not yet claim a complete schema-2 operational workflow or native keyring certification.

## Work acquisition and recovery

```sh
wosctl --profile executor_a work checkout --next --count 3 --limit 25 --output json
wosctl --profile executor_a work checkout TASK_UUID --version EXACT_VERSION
wosctl --profile executor_a work recover --all-pending --output json
wosctl --profile executor_a work list --output json
wosctl --profile executor_a work show CONTRACT_UUID --for-agent --output json
```

`--count` (1–10) counts independent intentions; `--limit` (1–100) bounds each scan. The server still limits a Principal to three active execution contracts across credentials and profiles. A request for five may therefore finish partially. The bounded set is saved in the profile before the first request. Each item has its own UUID, idempotency key, frozen typed bytes and fingerprint; accepted contracts survive another item's failure. No batch compensation or fresh recovery search is performed.

A search that found nothing is also durable. Repeating its intention returns the original empty result even after a new Task appears. An incomplete page retains `search_complete: false` and `next_cursor`; it does not declare global absence. Continuing that cursor is an explicit new checkout intention. Corrections and replanned deliveries require explicit Task checkout with `--previous-review REVIEW_CASE_UUID`; generic next search does not silently acknowledge their findings.

Preparation and technical response persistence use short stable profile locks. Requests run without a profile lock. Recovery merges only the matching frozen intention into the latest validated profile snapshot, preserving other processes' intentions and observed editor changes. An accepted result is persisted as `accepted_unmaterialized` before publishing its one contract file; the intention is removed only after materialization. Existing valid contract drafts are preserved. Corrupted, ambiguous or unexpected files fail explicitly and remain available for reconciliation.

For an unknown response, recovery first reads the original credential-bound durable operation and matches its command/fingerprint and exact response digest. This can reconcile acceptance after the agent signing key was revoked, without issuing another acquisition or extending expiry. Missing acceptance may be retried only with the original frozen command and live authorization. Destination, scope, CredentialID and issuer proof are checked; operation expansion is technical harness data and never current authority. An unknown/rejected unresolved intention is preserved, not rewritten with new CAS values.

`show --for-agent` returns the selected frozen instructions, constraints, criteria, current focal metadata and local progress. It omits issuer proofs, private references and other Outcome history. Read-only show/list/recover with no pending intentions do not acquire or renew work. A stored grant may have expired or lost authorization; final mutations must use their own live protocol checks.

Real HTTP fixtures on Memory, SQLite and PostgreSQL cover the second response lost after commit in a five-item batch, two eligible Tasks, preservation of an edited first draft, recovery without duplicate contracts and accepted-state reconciliation after key revocation. Shared storage tests cover durable empty search, pagination, cache-unavailable replay, exact original public response and Principal quota. The remaining P08/P09 and native gates above are not inferred from those fixtures.

### Independent review checkout

Schema-2 `review checkout <case-id> --version <exact-case-version>` and
`review checkout --next --count N --limit L` use the selected profile's live
credential and agent key. Each bounded batch item has its own frozen intention;
review recovery reconciles the original operation before retrying a request.
The server retains empty search results and enforces historical/current
Principal and separation-group independence plus one active review per Principal.
A multi-item request can therefore stop at a quota while retaining its accepted
file and remaining intentions; it does not evade the quota by switching profiles.

Review YAML starts with `decision: inconclusive` and stores the acquired case CAS.
`review list`, `review show <contract-id> --for-agent` and `review recover` do not
acquire, renew or approve work. Agent projection includes frozen instructions,
exact submitted material, omission references and public receipt fields, while
signature proofs stay in the local document for host verification. No returned
receipt is an evaluation of quality. Review sign/send/finish and lease maintenance
are separate subsequent implementation steps.

### Lease maintenance

`work renew <contract-id> --ttl 300` and `review renew <contract-id>` freeze the
original grant, lease CAS, destination, actor fingerprint and independent
idempotency key in the profile before contacting WOS. An uncertain response is
recovered with `work recover` or `review recover`: lookup of the accepted operation
comes first, and the accepted grant is merged into the latest observed draft.
Draft edits are retained. A changed authority, unresolved contract intention or
pending/accepted final return pauses maintenance instead of replacing intent.

`work resume <contract-id>` / `review resume <contract-id>` explicitly take over
execution with a new fence; `--ttl` is rejected because takeover never extends
expiry. Explicit `refresh` only reads current state/proof and preserves semantic
Task/case CAS. It does not acquire, renew or silently rebase changed material.

`work keepalive --all-active --foreground --interval 60` (or `review keepalive`)
runs in the host CLI until interrupted. `--once` performs one bounded pass for
supervision and testing. Items fail independently; revoked/expired authority and
pending final returns are skipped and their files retained. Uncertain intentions
require explicit recovery. Profile locks are short and are released during all
network calls and waits. WOS itself runs no keepalive daemon or agent scheduler.
Agent show reports a pending lease operation without expanding its technical blob.

Signed Go SDK mutations accept the operation registry's bounded technical result
size (1 MiB plus 64 KiB transport overhead); ordinary v1 limits remain 256 KiB.
This changes neither signature validation nor agent context budgets.

### Interrupted local publication

Initial v2 document creation stages and fsyncs complete YAML, then publishes it
exclusively; existing destinations are never overwritten. Accepted contract
materialization uses a temporary filename bound to the original pending ID. It
repairs only the exact constructor prefix of that accepted intention, or retains
a complete verified staged draft. Unknown/different contents remain untouched
for explicit reconciliation. Ordinary reads can remove a recognized temporary
alias only when its identity proves exactly two links inside the confined root;
the destination retains all current contents. Additional/outside links, symlinks,
reparse points and ambiguous `.yaml`/`.yml` destinations remain rejected.

Technical staging entries are bounded and are not agent context or additional
operational directories. Recovery tolerates process death after stage fsync,
exclusive publication and staging unlink without a new acquisition. User edits
and a destination renamed to `.yml` remain intact. General initial onboarding
stages that never reached publication and unrelated replacement temporaries are
preserved for diagnosis; they are not automatically attributed to accepted work.
Cooperative stable locks plus observed content CAS protect replacement writes;
profiles do not provide isolation against a hostile writer with the same OS user.
