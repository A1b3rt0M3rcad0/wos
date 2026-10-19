# Operational workspace migrations

This change advances both storage adapters from schema 30 to schema 32 through
additive, checksummed migrations. The product remains relational current state
with an immutable event log and durable outbox. It does not replay events to
rebuild operational state.

## Additive changes

| Migration | Persisted change | Existing data |
| --- | --- | --- |
| `0031_entity_notes.sql` | `entity_notes`, scoped Artifact/Evidence reference rows, note ordering/correction indexes, and an index over existing subject events | Existing activity, events, Contracts, proofs and receipts retain their original records |
| `0032_workspace_execution_time.sql` | Two scoped expression indexes for the latest signed and unsigned execution headers | Existing Contract JSON, specifications, signatures and acquisition timestamps are not rewritten |

Notes are immutable participant declarations: updates, discoveries or
observations. The server records authenticated authorship and recording time;
an optional occurrence time remains the participant's declaration. A correction
appends another note with the same authenticated Principal, subject and criterion
context, retaining a supersession reference to the original.
Notes do not acquire a Task, replace signed execution activity, change lifecycle,
assess a criterion or accept an Outcome. Terminal, nonarchived work can receive
documentary notes; archival guards still apply.

An internal Evidence source using `wos.entity-note` resolves to the actual scoped
observation. A source label or URL cannot supply authenticated authorship or
prove verification. Unresolved legacy sources appear as unavailable rather than
inventing an author or observation. Evidence remains support for a separate
assessment and acceptance decision.

The acquisition indexes preserve canonical RFC3339Nano ordering, including UTC
offsets and up to nine fractional digits. Each history returns one indexed
candidate; the application compares actual acquisition times and uses the ID
only to break an exact timestamp tie. The read does not load Contract specs or
sort full execution histories.

## Qualification evidence

The [source-stamped receipt](migration-upgrade.json) qualifies backend source
`e36a8f498c567f9cc944ec2ab1da7a4bb79a0bc4` on SQLite and real PostgreSQL.
Both independent signed fixtures advance from schema 30 to schema 32. Candidate
Core production files, module files and helper source match that commit; the
helper separately reports its compiled source and both schema versions. The
receipt records baseline, helper, tool and production hashes and verifies the
current source again before publication. Transport and presentation qualification
remain separate. The [earlier a8571c5 receipt](migration-upgrade-a8571c5.json)
is retained as history; the current receipt repeats qualification after the
read-only impact previews and optional revision preconditions, with the same
schema-31/32 migration files.

| Assertion | SQLite | PostgreSQL |
| --- | --- | --- |
| Preexisting tables checked | 69 | 69 |
| Nonmigration tables preserved exactly | 68 | 68 |
| Original migration rows preserved | 30 | 30 |
| Original signed facts preserved | 39 | 39 |
| Completed Task unchanged after late note | Pass | Pass |
| Replay returns the same note; timeline contains it once | Pass | Pass |
| Baseline binary explicitly rejects schema 32 | Pass | Pass |

The qualification tool creates a private archive of baseline
`5d945d3081f85927b80c686d695849bd1df73a5d`. Production source files must match
that commit's Git blobs. Only the existing signed-review journey test is
instrumented to retain its validated database. The tool then:

1. Opens the actual baseline fixture and verifies schema 30 and signed facts.
2. Captures all original rows in all 69 preexisting tables.
3. Runs both new migrations and verifies the complete candidate migration chain,
   names and checksums against the candidate source.
4. Compares all 68 nonmigration tables exactly and verifies that all 30 original
   migration rows remain present.
5. Authenticates against the preserved fixture, appends a late note to its
   completed signed Task, replays the intent and finds the note once in the
   unified timeline. The original Task remains unchanged.
6. Runs the separately built baseline binary against the upgraded fixture and
   verifies explicit rejection of the newer schema.

Raw database copies, dumps, credentials and logs remain in a private temporary
directory. The committed receipt contains source and binary hashes, adapter
results and scenario assertions, without connection strings or local paths.
Migration evidence is separate from browser acceptance and independent human
usability review.

## Upgrade and rollback

Follow the existing [coordinated cutover procedure](../../operations.md#upgrade-cutover-for-the-operational-candidate):
pause mutations, retain pending CLI intentions, stop every old API and delivery
worker, and take a protected consistent backup before migrating. Rehearse restore
into a separate destination, then run `wos db migrate` with the intended candidate
and start only replicas of that qualified source. Confirm readiness, issuer
identity, authorization and pending intentions before reopening writers.

The old binary rejects schema 32 at startup. This does not fence an old process
that was already running, and it does not establish mixed-version rolling-upgrade
support. There is no in-place downgrade: restore the pre-upgrade backup into a
separate database and reconcile subsequent security changes, pending intentions
and external receiver effects before cutover. See the
[operator backup and restore guide](../../../tools/operations/README.md).
