# Central installation and signed profiles

This guide describes the unreleased central workflow in `feat/central-agent-workflow`. Published `wosctl@0.4.2` does not implement it. Build the current source and use a server from the same source. [Legacy workspace migration](workspace-migration-v2.md) is a separate compatibility path.

## Initialize once, operate anywhere

```sh
wos init
wos profile list
```

The CLI stores operational material in a private OS data directory, independently of the current project. Linux uses `$XDG_DATA_HOME/wos` (default `~/.local/share/wos`), Windows uses `%LOCALAPPDATA%/WOS`, and macOS uses `~/Library/Application Support/wos`. The macOS path is implemented; native macOS distribution remains unqualified. `WOS_DATA_DIR` selects an explicit absolute root. No project `.wos` is discovered or created by the central workflow. Internal documents are owned by the CLI; agents do not open or edit them.

## Provision an approved identity

An administrator supplies the persistent server ID, issuer fingerprint, Tenant, credential, and authorized enrollment. Onboarding verifies these bindings and grants no server permission. Do not place administrative credentials in an executor profile.

For a host with a supported native keyring:

```sh
wos profile create executor --token-stdin --generate-signing-key \
  --server https://wos.example.test --server-id SERVER_UUID \
  --tenant TENANT_UUID --issuer-fingerprint sha256:APPROVED_DIGEST \
  --enrollment ENROLLMENT_UUID
```

Provide the token through protected standard input. Generated seeds and imported API credentials go only to the OS keyring. If its native service is unavailable, creation fails; it never falls back to plaintext storage. An existing seed may use `--signing-key-stdin --credential-ref env:WOS_TOKEN`; token and seed cannot share stdin in one invocation.

A host can instead provision protected references without a proposal file or keyring:

```sh
wos profile create executor \
  --server https://wos.example.test --server-id SERVER_UUID \
  --tenant TENANT_UUID --issuer-fingerprint sha256:APPROVED_DIGEST \
  --credential-ref env:WOS_TOKEN --private-key-ref env:WOS_SIGNING_SEED \
  --enrollment ENROLLMENT_UUID
```

The host supplies the named environment variables securely. Flags accept references, never secret values. The CLI derives the public fingerprint and verifies enrollment subject, purpose, and approved destination. Omit `--enrollment` only for a uniquely matching already registered live agent key. Protected mounted references are also supported by the [profile schema](../packages/wos-cli/schemas/profile-v2.schema.json): mounted files must be canonical private regular files outside the operational store, with no links and no other-account access. Windows checks a restricted DACL.

`--outcome OUTCOME_UUID` optionally pins a default Outcome. Without it, scoped commands can select `--outcome` explicitly or discover a unique match; ambiguity or an incomplete scan fails safely. Server and Tenant remain bound to the profile. A changed destination or credential requires new onboarding.

## Select and inspect profiles

```sh
wos profile use executor
wos profile list
wos profile show executor
wos auth status --output json
wos doctor --output json
wos --profile reviewer auth status --output json
```

Selection precedence is `--profile`, then `WOS_PROFILE`, then the persistent active profile. A sole profile is a fallback when none is active. Each invocation captures the selection once; another process changing the active profile cannot redirect work already running. Separate profiles keep credentials, signing keys, contracts and frozen intentions isolated. They do not create distinct Principals or bypass review independence and quotas. Profile names are case-sensitive; collisions and reserved names fail.

`profile show` reads nonsecret local configuration. `profile inspect`, `auth status`, and `doctor` check live identity and report compact public key status, restrictions, destination and pending-operation diagnostics without exposing secrets. These observations grant no authority; each mutation rechecks live permissions.

`profile remove NAME` refuses the active profile, pending intentions, and local contracts. Removal retires the selector while preserving fenced historical bytes and shared secret references; it does not revoke anything on the server. Explicit re-provisioning of the same name archives the retired store. Do not delete operational files to bypass these checks.

## Migrate an existing schema-2 workspace

Stop every old writer first, including older CLI binaries, agents, and keepalive processes. Back up the source and preserve its protected references. Then use an explicit source:

```sh
wos init --from /absolute/old/project --writers-drained --dry-run
wos init --from /absolute/old/project --writers-drained
```

The dry run is read-only. Migration validates authenticated profiles/contracts, snapshots bounded exact bytes, coordinates supported locks, fences the source and stages isolated destination stores. It preserves pending payloads, idempotency keys, receipts, grants and local history; it does not send or re-sign them. Interrupted migration resumes only the same frozen copy. Source files remain for diagnosis. Collisions, changed bytes, unsupported schema, or uncertain ownership fail explicitly. Schema 1 is not silently converted.

The retirement fence protects supported new CLI processes, including queued writers. **Old binaries do not recognize it**, so `--writers-drained` is an operator prerequisite, not proof that old processes were stopped. Explicit `--workspace PATH` or `WOS_WORKSPACE` selects legacy compatibility; it never substitutes for central profile selection.

## Enrollment recovery, trust, and locks

Use `wos --profile executor profile recover` after interrupted enrollment. The authenticated frozen intention is persisted before mutation; an unknown response retains the original key, command and CAS. Do not replace a challenge or edit pending bytes. Work acquisition uses separate durable intentions and does not reuse enrollment as work authority.

After an operator explicitly recovers the instance issuer, approve an independently verified replacement with `wos --profile executor profile trust --issuer-fingerprint sha256:APPROVED_DIGEST`. Trust is appended atomically without rewriting historical contracts or returns. See [issuer recovery](issuer-recovery.md).

Stable locks live outside operational material. Unix private lock files are never unlinked and release on process death; Windows uses global named mutexes. Content comparison and atomic publication preserve observed conflicting edits. These controls coordinate cooperating processes and do not isolate hostile processes under the same OS account. Native keyring and platform release qualification remain separate gates.

## Work acquisition and recovery

```sh
wos --profile executor_a work checkout --next --count 3 --limit 25 --output json
wos --profile executor_a work checkout TASK_UUID --version EXACT_VERSION
wos --profile executor_a work recover --all-pending --output json
wos --profile executor_a work list --output json
wos --profile executor_a work show CONTRACT_UUID --for-agent --output json
```

`--count` (1–10) counts independent intentions; `--limit` (1–100) bounds each scan. The server enforces the configured Tenant Principal quota across credentials and profiles (default three active execution contracts), together with credential and Tenant ceilings. A request exceeding current capacity may therefore finish partially. The bounded set is saved in the profile before the first request. Each item has its own UUID, idempotency key, frozen typed bytes and fingerprint; accepted contracts survive another item's failure. No batch compensation or fresh recovery search is performed.

A search that found nothing is also durable. Repeating its intention returns the original empty result even after a new Task appears. An incomplete page retains `search_complete: false` and `next_cursor`; it does not declare global absence. Continuing that cursor is an explicit new checkout intention. Corrections and replanned deliveries require explicit Task checkout with `--previous-review REVIEW_CASE_UUID`; generic next search does not silently acknowledge their findings.

Preparation and technical response persistence use short stable profile locks. Requests run without a profile lock. Recovery merges only the matching frozen intention into the latest validated profile snapshot, preserving other processes' intentions and observed editor changes. An accepted result is persisted as `accepted_unmaterialized` before publishing its one contract file; the intention is removed only after materialization. Existing valid contract drafts are preserved. Corrupted, ambiguous or unexpected files fail explicitly and remain available for reconciliation.

For an unknown response, recovery first reads the original credential-bound durable operation and matches its command/fingerprint and exact response digest. This can reconcile acceptance after the agent signing key was revoked, without issuing another acquisition or extending expiry. Missing acceptance may be retried only with the original frozen command and live authorization. Destination, scope, CredentialID and issuer proof are checked; operation expansion is technical harness data and never current authority. An unknown/rejected unresolved intention is preserved, not rewritten with new CAS values.

`context` (or `show --for-agent`) returns the selected frozen instructions, constraints, criteria, current focal metadata and local progress. It omits issuer proofs, private references and other Outcome history. Read-only show/list/recover with no pending intentions do not acquire or renew work. A stored grant may have expired or lost authorization; final mutations must use their own live protocol checks.

Real HTTP fixtures on Memory, SQLite and PostgreSQL cover the second response lost after commit in a five-item batch, two eligible Tasks, preservation of an edited first draft, recovery without duplicate contracts and accepted-state reconciliation after key revocation. Shared storage tests cover durable empty search, pagination, cache-unavailable replay, exact original public response and Principal quota. Final acceptance gates are not inferred from those individual fixtures.

### Independent review checkout

Schema-2 `review checkout <case-id> --version <exact-case-version>` and
`review checkout --next --count N --limit L` use the selected profile's live
credential and agent key. Each bounded batch item has its own frozen intention;
review recovery reconciles the original operation before retrying a request.
The server retains empty search results and enforces historical/current
Principal and separation-group independence plus the configured review quota (default one active review per Principal).
A multi-item request can therefore stop at a quota while retaining its accepted
file and remaining intentions; it does not evade the quota by switching profiles.

A review draft starts with `decision: inconclusive` and retains the acquired case CAS. Use typed `review decision set` and `review assessment add` commands to prepare it.
`review list`, `review show <contract-id> --for-agent` and `review recover` do not
acquire, renew or approve work. Agent projection includes frozen instructions,
exact submitted material, omission references and public receipt fields, while
signature proofs stay in the local document for host verification. No returned
receipt is an evaluation of quality. Review sign/send/finish and lease maintenance use the same original-intention
recovery rules; their execution does not certify material quality.

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

### Signed returns and receipt-bound cleanup

```sh
wos --profile executor_a work sign CONTRACT_UUID --output json
wos --profile executor_a work send CONTRACT_UUID --output json
wos --profile executor_a work recover --output json
# Alternatively: work finish CONTRACT_UUID performs explicit sign/send/reconcile.
wos --profile reviewer_a review finish REVIEW_CONTRACT_UUID --output json
```

Signing freezes the exact draft, payload, signature, original CID, request ID and
idempotency key locally before mutation. Sending first reconciles any accepted
original receipt. Recovery never sends a prepared-only return automatically and
never re-signs an uncertain one. A remote commit can be confirmed while local
cleanup requires action: edited drafts remain intact. Only a verified closed
signed acceptance permits deletion of its matching unchanged contract document.
The profile survives, with no operational history directory. Executor delivery
may close its obligation while review remains pending; it is not Task approval.

## Signed acquisition quotas

A Tenant administrator can set optional `max_active_work_per_principal` and `max_active_review_per_principal` fields in a Tenant `WorkAcceptancePolicy` through the existing authenticated `set_acceptance_policy` security command, with current `expected_tenant_version` and policy `expected_version`. Defaults remain **3 work / 1 review** when fields are absent; explicit values are **1..100**. Outcome/WorkItem policy overrides of these fields are rejected.

Credential and Tenant active-contract ceilings still apply independently. Additional profiles or credentials of the same Principal share its capacity and cannot serve as an independent reviewer. Raising a quota does not add permissions or relax review separation, leases or fencing. Lowering it does not revoke issued contracts automatically; it blocks further acquisition until activity permits it. `wos --profile NAME auth status --output json` and doctor expose `quotas`, including current own activity, Principal/CID/Tenant ceilings and the effective minimum. No other Principal's contracts are listed. Quota rejection leaves the original WorkItem/Outcome revision unchanged; retry only the original intent after an authorized configuration or activity change.
