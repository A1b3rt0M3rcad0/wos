# wosctl

API-only WOS client; `wos` remains the server/admin executable. No database is opened
by the client, and no instructions, scripts, agents or artifact URLs are executed.

```sh
go build -o wosctl ./packages/wos-cli/cmd/wosctl
wosctl init --server https://wos.example.test --namespace UUIDv7 --outcome UUIDv7 --credential-env WOS_TOKEN --output json
wosctl auth status --output json
wosctl work checkout UUIDv7 --version 2 --output json
wosctl work checkout --next --outcome UUIDv7 --output json
wosctl work status UUIDv7 --output json
```

`--workspace PATH` selects an existing project root. Configuration contains an environment
variable reference, never a token. Init checks the server and existing authorized Outcome.
New-protocol acquisition requires explicit Namespace cutover (C10); init does not enable it.
Changing server, Namespace or credential reference invalidates the local trust binding.

Files use the generated schemas in `schemas/`. The supplied plan's YAML examples were
illustrative; schema 1 uses a typed `contract`, `checkpoint` or `material` field and the
same public snake_case documentary DTOs as the API. Contracts retain exact string fencing
and immutable specs; local state is an observation, never authorization. `.yaml` and `.yml`
are equivalent, but two files with the same stem are rejected.

Parser limits: 256 KiB, depth 16, 10,000 value nodes, 64 KiB per scalar. UTF-8, a single
mapping, string keys, JSON decimal numbers and lowercase true/false are required. Unknown
fields, duplicates, aliases, anchors, merge keys and explicit/custom tags are rejected.

Checkout writes its exact destination-bound intent before sending and saves the receipt
before materialization. No retry changes an expected version or payload. A failure after
remote commit returns exit 9 with the receipt path and requires recovery of that acquisition;
no compensating release exists. State/outbox/receipts remain under `.wos/` and are ignored
by Git. File access uses `os.Root` confinement, rejects symlinks, uses same-directory atomic
replacement and cooperative locks. Actual Windows crash/lock behavior remains a release gate.

JSON stdout is a single structured result; diagnostics go to stderr. Exit codes: 0 success,
2 invalid input, 3 auth, 4 conflict, 5 authority lost, 6 transport uncertainty, 7 review pending,
8 no acquisition in the scanned page, 9 local persistence/materialization failure.

Available operations include status/refresh/resume (explicit `--takeover`), renew,
keepalive (`--foreground`), diff/validate/checkpoint/sync/submit/finalize/finish/recover,
contract queries/revoke and planning/review wrappers. `work recover --next` reconciles
an uncertain next acquisition without selecting another task. An abandoned local lock
can be removed only with explicit `work recover --break-lock`; this does not revoke
remote authority and requires the caller to stop the old local process first.

```sh
wosctl work validate UUIDv7 --output json
wosctl work checkpoint UUIDv7 --output json
wosctl work sync UUIDv7 --output json
wosctl work submit UUIDv7 --output json
wosctl work finalize UUIDv7 --submission UUIDv7 --output json
wosctl work recover UUIDv7 --output json
wosctl work keepalive UUIDv7 --foreground --output json
wosctl objective create --file objective.yaml --output json
wosctl roadmap publish --file publish-command.yaml --output json
wosctl review attest --file assessment-command.yaml --output json
```

Planning/review YAML uses the exact public command DTO, validated against the generated
catalog before sending. For example, objective creation supplies scope, title, priority,
and required_for_outcome. Assessment supplies submission_id, owner, criterion_id,
criterion_revision, expected_version, result and rationale. The server enforces reviewer
independence; the CLI does not grant approval from observed test results.

Documentary result records use unique local keys. Sync resolves and durably records their
server IDs; changing already registered material requires a new key. `finish` checkpoints,
syncs documentary items, submits and tries finalization using separate keys/receipts.
It never fabricates assessments; exit 7 means review is pending. Local JSON journal/receipt
files allow a bounded 512 KiB envelope; YAML and remote payloads retain the 256 KiB limit.
SDK mutations never follow redirects, preventing destination-bound credentials/intents
from being sent to a different endpoint.

## Distribution and versions

The coordinated WOS npm service package includes `wos` and `wosctl` on Linux amd64.
The skill package installs five skills separately. Native client assets contain the
executable, schema files, licenses, README and `release.json` identifying version,
commit, platform and SHA-256. `wosctl version --output json` works without credentials,
a workspace or a service. Windows ZIP preparation requires actual matching Windows
workspace/protocol/installer tests before publication; cross-compilation is not a
supported-platform claim. Standalone clients do not require Node.

Namespace administration is `protocol get`, `protocol set --file protocol.yaml`,
and `protocol reconcile --file reconcile.yaml`. Files are typed command mappings;
these writes have the same durable journal as planning writes. A serialization
conflict keeps the intent pending (exit 6); recover with the same destination/key,
without changing CAS versions. See `docs/work-contract-operations.md` for deployment
and `docs/work-contract-load-evidence.md` for the scoped local load measurements.


## Workspace migration

The schema-1 commands above remain available for legacy recovery. A schema-2
profile uses signed work/review operations with `--profile NAME` on every call.
To preserve an old workspace in a separate approved schema-2 destination:

```sh
wosctl workspace migrate --workspace ./legacy-project --to 2 --profile executor --dry-run
wosctl workspace migrate --workspace ./legacy-project --to 2 --profile executor --destination ./signed-project
```

Repeat the same command after an interruption. Changed drafts are preserved and
block automatic completion. Resolve uncertain or accepted-but-unmaterialized
legacy acquisitions in the original workspace first. Imported records are
explicitly `legacy_unsigned`, readable as history but refused by signed mutation.
Source files remain intact. See [the migration guide](../../docs/workspace-migration-v2.md)
for identity, limits and recovery requirements. Migration does not activate a
Namespace protocol or manufacture signatures.
