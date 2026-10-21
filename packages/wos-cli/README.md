# Native CLI

The `wos` CLI is the operational interface for agents. It discovers and acquires one WorkItem, exposes bounded context, records activity and proof, prepares delivery, and reconciles interrupted operations. The Server owns authorization and durable shared state; the external runtime executes work.

> [!NOTE]
> This README describes the current source branch. Published `wosctl` can differ; check `wos version`, `wos catalog` and server capabilities. The historical `wosctl` executable remains a compatibility alias.

For coherent inspectors, card summaries, documentary notes and atomic planning,
see [operational workspace contracts](../../docs/operational-workspace-api.md).
The `inspect` and restricted `planning apply` / `planning recover` groups use the
same authenticated API while signed execution and review keep their own channels.

```sh
go build -o ./bin/wos ./packages/wos-api/cmd/wos
./bin/wos version --output json
./bin/wos --help
```

Use the built executable as `wos` below. The binary also contains service/operator commands and the offline Skill installer; client operations access the API and never open the server database or execute artifact URLs, scripts or agents. Source builds use the repository's pinned Go toolchain; see [development setup](../../CONTRIBUTING.md).

## Initialize installation data

```sh
wos init
wos profile list
wos profile use executor
```

Initialize once and operate from any directory. Ordinary commands do not create a per-project `.wos/` or search parent directories for operational state.

| Platform | Default data directory                               |
| -------- | ---------------------------------------------------- |
| Linux    | `$XDG_DATA_HOME/wos`, otherwise `~/.local/share/wos` |
| macOS    | `~/Library/Application Support/wos`                  |
| Windows  | `%LOCALAPPDATA%/WOS`                                 |

An absolute `WOS_DATA_DIR` overrides the data directory for an explicit installation/test environment. Path resolution is not a platform release certification; published native assets currently target Linux/Windows x64.

## Profiles and protected onboarding

Each Profile owns its immutable server/Namespace/Principal binding, protected credential/key references, contracts and recovery state. Profiles can use different servers or Namespaces. An optional default Outcome is convenience scope, not new authority.

An operator supplies an exact trusted origin, persistent ServerID, independently approved issuer fingerprint, and authorized key enrollment:

```sh
wos profile create executor --server APPROVED_ORIGIN --server-id SERVER_UUID --namespace NAMESPACE_UUID --issuer-fingerprint sha256:APPROVED_DIGEST --enrollment ENROLLMENT_UUID --generate-signing-key --credential-ref env:WOS_EXECUTOR_TOKEN
wos profile show executor
wos --profile executor profile inspect
wos --profile executor auth status --output json
wos --profile executor doctor --output json
```

`--outcome UUID` is optional. Generated keys use the supported protected key mechanism. A headless host can instead use `--private-key-ref env:WOS_EXECUTOR_KEY` with `--credential-ref env:WOS_EXECUTOR_TOKEN`; the external key is a canonical base64 Ed25519 seed/private key. Do not combine reference mode with generation/stdin flags. Explicit stdin alternatives exist, but token and key cannot share stdin. Never pass secret values in arguments, source files, model context or logs.

An administrator must provision current grants, registered signing identity and the Namespace's protocol. Neither `init`, Profile creation nor Skill installation grants permission or activates signed contracts. An executor and independent reviewer need distinct eligible Principals; different labels/keys for one Principal do not establish independence.

| Command                                                     | Purpose                                                                                                     |
| ----------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------- |
| `profile list`                                              | List known Profiles and selection                                                                           |
| `profile show [NAME]`                                       | Read nonsecret local configuration                                                                          |
| `profile use NAME`                                          | Persist an interactive active default                                                                       |
| `profile inspect [NAME]`                                    | Verify live server identity and authorized status                                                           |
| `profile recover [NAME]`                                    | Reconcile original enrollment intent                                                                        |
| `profile trust --issuer-fingerprint sha256:APPROVED_DIGEST` | Explicit independently approved issuer update                                                               |
| `profile remove [NAME]`                                     | Safely remove selection only when contracts/pending intentions permit it; preserve external secrets/history |

Selection resolves once per invocation: `--profile` > `WOS_PROFILE` > persisted active Profile, with a sole-Profile fallback. Invalid or ambiguous selection fails. Concurrent workers pin `--profile NAME` or process-scoped `WOS_PROFILE`; do not use global `profile use` as their identity switch. One process changing the default cannot redirect another invocation.

## Discover, acquire and inspect

```sh
wos --profile executor outcome list --limit 10 --output json
wos --profile executor work available --outcome OUTCOME_REF --limit 10 --output json
wos --profile executor work next --outcome OUTCOME_REF
wos --profile executor work list --output json
wos --profile executor work context --for-agent --output json
wos --profile executor work status --for-agent --output json
```

Use `work checkout WORK_REF` for an explicit unique available title/ID prefix; exact UUID plus `--version` remains supported. Unique convenience references resolve to exact scope/ID/version before mutation. An incomplete bounded search cannot establish uniqueness or prove there is no work. Honor cursor/search-completeness information and use an exact ID when asked.

`work list` queries bounded server execution headers plus local contract metadata. Use `--status active|delivered|completed|expired|revoked` for a persisted-status filter, `--holder PRINCIPAL_ID` for authorized coordinator reads, and `--limit`/`--cursor` for paging. `effective_status` also reflects lease expiry. Headers and local absence grant no authority; an accepted execution can be delivered while its Task awaits review.

The contract grants authority for a WorkItem, with Outcome/Objective context. A candidate listing is not acquisition. The issued instructions, constraints and current criteria stay immutable. `context` includes the complete issued obligation without exposing internal proofs or serialized paths; `status` combines known current obligations and authority. Neither command grants permission.

`context` also includes a bounded current focal view of the Task, Outcome, Objective, dependencies, blockers and decisions, plus summaries of the five newest activity records across that WorkItem's execution attempts. Descriptions and reference/version collections declare omissions; use `work history` to expand complete records. Current coordination state does not rewrite the immutable specification. Each server resource exposes its own revision/evaluation time; these separate reads do not claim one combined transaction. Use `history_follow_up` with its WorkItem, Outcome, descending order and cursor to continue the frozen history traversal. `--limit` adjusts the context activity page up to 25 records.

Contract arguments may be omitted only with one applicable local execution/review contract. With multiple contracts, supply `[CONTRACT_REF]` or `--contract REF`. After receipt-confirmed local cleanup, historical reads can use the exact `--work WORK_ITEM_UUID` with explicit Outcome scope. A UUID is a reference, not authorization.

For an exact historical execution contract, `work context CONTRACT_UUID --outcome OUTCOME_REF` also restores its immutable specification and bounded `accepted_result` summary/evidence/artifact references from the Server. The CLI verifies the pinned issuer receipt and exact scope, contract, WorkItem, specification and submission digest; it does not recreate authority or infer acceptance from current Task state.

## Record execution through typed commands

```sh
wos --profile executor work log add --type progress --description "Implemented the agreed change"
wos --profile executor work artifact add --name "Change set" --uri ACTUAL_IMMUTABLE_URI --artifact-type code --source-version ACTUAL_COMMIT
wos --profile executor work evidence add --type test_result --description "Describe the checks actually run and their result" --uri ACTUAL_REPORT_URI --criterion ORIGINAL_CRITERION_REF
wos --profile executor work evidence list --limit 10 --output json
wos --profile executor work artifact list --limit 10 --output json
wos --profile executor work progress set --summary "Describe current progress" --next-action "Describe the next permitted action"
wos --profile executor work result set --summary "Describe the delivered result" --completion auto
```

Activity types are `progress`, `discovery`, `problem`, `correction`, `validation`, and `result`. Execution log/evidence/artifact additions register immediately on the Server under current contract authority, then publish returned IDs locally. Uncertain or locally interrupted publication preserves the exact original intention for recovery.

If the transport returns only a compact commit receipt, recovery expands the original durable response and validates its digest/fingerprint binding. Local recovery stores a compact authenticated publication record for the accepted Activity and command identity, without duplicating large documentary arrays. The exact original command/key and full server result remain unchanged.

Evidence types exposed by this command include `test_result`, `inspection`, `attestation`, `source`, and `external_evaluation`. An artifact's optional `--source-version`/`--content-hash` must be actual immutable material identifiers. An evidence link to a criterion does not approve it. Activities are declarations, evidence is observation/source, and CriterionAssessment is explicit evaluation. WOS does not certify that a reported external action occurred.

The CLI is the only agent interface: never open, interpret, edit or delete internal operational documents. Typed setters validate input, hold the contract lock, enforce local CAS and preserve issued/frozen sections. Progress is advisory; a result summary is required for delivery. Known obligations come from current criteria, dependencies, blockers and review findings rather than a duplicate manual pending list.

`progress set` and `result set` save local draft material; a durable chronology requires `log add`, and the signed return persists the final result. Use the `draft_digest` from context with `--expected-draft DIGEST` to reject changes based on an outdated draft. Result and progress setters never change issued authority or a frozen return.

```sh
wos --profile executor work status --for-agent --output json
wos --profile executor work check --output json
wos --profile executor work finish --output json
```

`check` is a preflight; the Server still validates current policy, authority and versions. `finish` reuses signed freeze/send/receipt semantics. An accepted executor return may await independent review; it does not automatically complete parent Objectives/Outcomes. Low-level `sign` and `send` remain explicit advanced operations.

## Independent review and correction

```sh
wos --profile reviewer review checkout --next
wos --profile reviewer review context --for-agent --output json
wos --profile reviewer review decision set --decision approved --reason "Describe independent checks"
wos --profile reviewer review assessment add --criterion ORIGINAL_CRITERION_REF --result met --rationale "Describe actual supporting observations"
wos --profile reviewer review check
wos --profile reviewer review finish
```

Set approval before adding its required original criterion assessments. `--evidence UUID,...` selects actual registered proof. `review artifact/evidence add` stages local review material and returns local keys with `remote_committed: false`; accepted signed finish makes it durable. Assessment references can use returned `local:LOCAL_KEY` values. Complementary records require the accepted executor artifact's exact source version/checksum.

For non-approval, first set `review decision set --decision changes_requested|inconclusive --reason TEXT`, then add an original-obligation finding using `review finding add --criterion REF --description TEXT` (or `--requirement ORIGINAL_REF`, never both). Omit criterion assessments. The initial empty review reason must be filled before finding validation. A signature proves origin, not quality. New material needs new exact-submission/current-criterion review.

Review context exposes the staged decision, reason, assessments, findings and documentary references with explicit omissions. Large contracts expand their authenticated original-specification reference through the CLI; the issued review document stays unchanged. Before freezing a return, correct draft mistakes through these commands:

| Draft record | Inspect | Correct or remove |
| --- | --- | --- |
| Assessment | `review assessment list --limit 10` | `review assessment set --criterion REF --result met --rationale TEXT`; `review assessment remove --criterion REF` |
| Finding | `review finding list --limit 10` | `review finding set --finding REF --description TEXT`; `review finding remove --finding REF` |
| Correction response | `work correction list --limit 10` | `work correction set --finding ORIGINAL_REF --summary TEXT`; `work correction remove --finding ORIGINAL_REF` |
| Reviewer evidence | `review evidence list --limit 10` | `review evidence remove --evidence local:LOCAL_KEY`, then add its corrected observation |
| Reviewer artifact | `review artifact list --limit 10` | `review artifact remove --name LOCAL_KEY`, then add its corrected reference |

`add` rejects duplicate draft assessments/responses; `set` explicitly replaces or creates one. Finding `set` preserves the finding ID and existing obligation unless an explicit criterion/requirement is supplied. Changing an approved decision does not silently discard assessments: remove them explicitly first. Evidence referenced by a staged assessment cannot be removed until the assessment is changed or removed. These commands change unsubmitted local material, never accepted server history. Execution assessment setters use the same verbs where the Server's return policy permits them. Local list pages are bounded to 64 KiB and their cursors bind to the contract, section and draft digest; after an edit, start a fresh list.

Corrections explicitly acknowledge the original case, which generic next acquisition excludes:

```sh
wos --profile executor work checkout TASK_UUID --version CURRENT_VERSION --previous-review ORIGINAL_CASE_UUID
wos --profile executor work context --for-agent --output json
wos --profile executor work correction add --finding ORIGINAL_FINDING_REF --summary "Describe the actual correction"
wos --profile executor work result set --summary "Describe the corrected delivery"
wos --profile executor work check
wos --profile executor work finish
```

Address every frozen finding and preserve scope. New requirements require explicit replanning; old approval does not carry forward.

The server must advertise the new activity/attention capabilities. An older server fails clearly; the CLI never falls back to unsigned execution to bypass the missing capability.

## Bounded history and output

```sh
wos --profile executor work history --limit 10 --output json
wos --profile executor work history --work WORK_ITEM_UUID --outcome OUTCOME_REF --limit 10 --output json
wos --profile executor work history --work WORK_ITEM_UUID --outcome OUTCOME_REF --order desc --limit 5 --output json
```

Server activity survives contract completion and local cleanup. History defaults to chronological `--order asc`; `--order desc` reads newest records first without scanning all earlier pages. Preserve the same scope, order and filters with `next_cursor`; the traversal remains bound to its initial upper revision. Read current context/status first and expand only a specific gap. Execution activity is not a complete review-decision journal.

JSON stdout is one structured result and diagnostics are stderr. Prefer `--for-agent --output json` for focal model context. No token, private key or complete unrelated proof payload belongs in compact output.

| Exit | Meaning                                                |
| ---- | ------------------------------------------------------ |
| 0    | Success                                                |
| 2    | Invalid input/selection                                |
| 3    | Authentication/authorization failure                   |
| 4    | Conflict requiring inspection                          |
| 5    | Authority lost                                         |
| 6    | Transport/commit uncertainty; preserve original intent |
| 7    | Review/acceptance remains pending                      |
| 8    | No acquisition in the scanned page                     |
| 9    | Local persistence/materialization failure              |

## Recovery and authority

```sh
wos --profile executor work recover --all-pending --output json
wos --profile reviewer review recover --all-pending --output json
```

Recovery checks the original-CID operation/receipt before exact replay. A prepared-only return stays unsent. Never reacquire replacement work to repair a missing local acquisition, change CAS/fencing/payload/destination to force success, or manually remove uncertain state. Cleanup requires verified signed closed acceptance.

A definitively rejected frozen return can be inspected with `work resolution-status CONTRACT_UUID`. Only an explicit operator/host decision permits `work resolve CONTRACT_UUID --reason TEXT`; verified durable server closure is required before draft changes. Terminal resolution requires the exact contract UUID, not a short reference; review has equivalent operations. Timeout, EOF, 401/403/409 or 5xx alone never authorizes abandonment. An accepted original wins over a later closure request.

The terminal-resolution protocol currently applies to signed final returns, not activity appends. If an original frozen append can no longer be accepted because its authority expired or was definitively rejected, preserve its exact pending payload/key and escalate to the operator; this branch does not supply an authenticated terminal closure for that append. Do not clear internal state, renew around the intention, or use another Profile as if it repaired the original obligation. A separate Profile may perform other explicitly authorized work while the original recovery state remains intact.

Renew, resume and refresh are distinct. Resume rotates execution/fencing without extending or reviving expiry; refresh never takes over. Stop external effects on expired/revoked/stale authority. Explicit foreground keepalive is supervised by the host, not a WOS agent scheduler. Contracts do not lock external files or make side effects idempotent. Independently verify changed issuer identity before `profile trust`; never re-sign an uncertain return under a new pin to bypass its original binding.

## Migrate existing local state

```sh
wos init --from /absolute/legacy-project --dry-run
wos init --from /absolute/legacy-project --writers-drained
```

Initial central import targets signed schema-2 state. Drain all old writers first: supported new clients retire the original source against further writes, but older binaries cannot honor a new marker. Import preserves original source, issued proofs, signatures, MACs, pending intentions, receipts and identity/destination binding. Unknown content, name collision or secret-reference confinement failures block unsafe import. External secret values are never copied merely to make an import succeed.

Schema-1 state requires its explicit original recovery/conversion first. `--workspace`/`WOS_WORKSPACE` remain explicit compatibility/recovery mechanisms; they are not automatic per-directory discovery. Imported legacy records remain `legacy_unsigned` and cannot authorize signed mutations or acquire retroactive signatures. No local migration activates or downgrades Namespace protocol.

See [migration plan](../../docs/central-agent-workflow/migration.md), [schema-1 conversion](../../docs/workspace-migration-v2.md), and [historical contract operations](../../docs/work-contract-operations.md). Those low-level compatibility references do not replace the central agent workflow.

## Development and distribution

```sh
go test ./packages/wos-cli ./packages/wos-skill
go test -race ./packages/wos-cli ./packages/wos-skill
node --test packages/wos-skill/test/installer.test.mjs
```

Real PostgreSQL, native Windows, crash/full-disk recovery, protocol parity and final-source package checks are distinct qualification gates. File installation/cross-compilation alone does not certify a platform or agent runtime. Standalone native clients do not require Node; the published npm launcher requires its documented Node version.

Read [agent operation](../../docs/agents.md), [central implementation plan](../../docs/central-agent-workflow/README.md), [ROADMAP](../../ROADMAP.md), and [client qualification](../../docs/client-qualification.md) for verified status and remaining limits. Merge, release and registry publication are separate owner actions.
