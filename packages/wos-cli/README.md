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

C07 foundation currently exposes init, capabilities, auth status, checkout, status, and
contract queries/revoke. Remaining workspace/planning operations are implemented in C08/C09.
