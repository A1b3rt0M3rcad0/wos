# Protected signing and recovery

These instructions target the centralized CLI on this source branch. Verify the installed binary supports the documented commands; a published package may differ. The `wos` command is the native client installed by `wosctl`; the separately deployed service is WOS Server. Installing Skills copies instructions only: it does not provision identities, connect a runtime, start a daemon, or grant access.

## Setup and identity

Run `wos init` once. Operational data belongs to the installation, not a project. An operator provisions an approved Profile with an exact ServerID/origin, Tenant, authenticated Principal/credential, registered signing key and independently approved issuer fingerprint. Use `wos profile list` and `wos profile show NAME` to inspect nonsecret configuration. Never print credential values/private seeds or put them in arguments, model context, source files or logs.

Selection is explicit `--profile`, then `WOS_PROFILE`, then persisted active selection, with a sole-Profile fallback. `wos profile use NAME` changes the interactive default. Every concurrent worker pins `--profile NAME` or a process-scoped `WOS_PROFILE`; a Profile name is not authority or an independent Principal. Each invocation freezes its selection. Different Profiles can bind different servers/Tenants.

The CLI is the agent's only operational interface. Never open, interpret, edit or delete internal contract/Profile documents. The CLI owns serialization, protected paths, locking, local CAS, signatures and technical recovery state. Do not bypass it with a direct HTTP/MCP mutation, manual file changes, or a protocol downgrade.

## Tenant and protocol transition

Current commands bind `tenant_id` and use signed protocol 3. A Profile selects one
approved Tenant runtime; hostname routing does not grant access. Do not supply a
schema name, edit a Profile to switch Tenants, or infer authority from customer
metadata. Historical protocol-2 proofs preserve their original bytes for reading
and verification; they cannot authorize new mutations. An operator must resolve
old pending requests and active authority before storage cutover, then explicitly
migrate eligible local Profiles. Never rename fields inside signed material or
re-sign an uncertain old request as a new intention.

## Authority and material

Acquire one WorkItem with `work next` or explicit `work checkout REF`. Candidate listing grants no authority. Read `work context --for-agent --output json`; immutable issued obligations must not be rewritten. An omitted contract argument selects only a single applicable local contract; use an exact/unique reference when ambiguous. UUIDs remain supported. Bounded search cannot establish uniqueness if it is incomplete.

Use typed progress/result, activity, artifact and evidence commands. Activities are declarations; registered evidence is an observation/source; assessments evaluate exact current criteria/material. Neither passing tests, a valid signature, an activity log nor an accepted executor return implies independent approval or parent Objective/Outcome achievement. Do not create a redundant manual business pending list; context/status expose known obligations, blockers and review findings.

Run `work check` before `work finish`. Check can become stale; the Server authorizes acceptance. Finish freezes, signs, sends and reconciles the exact original return. Accepted execution may close the executor obligation while the WorkItem remains awaiting independent review. Keep artifact revisions/checksums factual and use only actual registered IDs.

## Interrupted operations

Use the same pinned Profile:

```sh
wos --profile executor work recover --all-pending
wos --profile executor work status --for-agent --output json
```

Use `review recover --all-pending` for review. Recovery checks the original-CID operation/receipt before identical replay. It never automatically sends a prepared-only return or acquires replacement work. Do not change destination, credential, payload, CAS, idempotency identity or fencing to force success. Missing local data after acquisition is repaired from the original intention, not a second acquisition. Cleanup needs verified signed closed acceptance; uncertain data remains protected.

After a definitive frozen-return rejection, `work resolution-status CONTRACT_UUID` diagnoses it. Only an explicit operator/host decision permits `work resolve CONTRACT_UUID --reason TEXT`; a durable verified closure proof is required before draft changes. Terminal resolution requires the exact contract UUID; `review` has equivalent commands. Timeout, EOF, 401/403/409 or 5xx alone never permits abandonment. Original acceptance wins.

Explicit `renew`, `resume` and `refresh` retain distinct meanings: resume rotates fencing without reviving expiry; refresh never takes over. Stop external effects on expiry, revocation or stale authority. Foreground keepalive must be explicitly launched/supervised by the host; WOS is not an agent scheduler. A contract does not lock repository files or make external side effects idempotent.

Issuer changes require independently approved fingerprint and explicit `profile trust`; never blindly trust a changed endpoint or re-sign uncertain bytes. Migration uses an explicit CLI dry-run/import with writers drained and original state preserved. Legacy unsigned data does not gain signatures or signed authority through import.
