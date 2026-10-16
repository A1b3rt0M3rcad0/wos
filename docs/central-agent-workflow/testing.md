# Verification and acceptance matrix

This is the required verification strategy, not a report of tests already run. Record actual commands, source revision, compiler, platform, backend and explicit skips in the branch validation record/ROADMAP. Do not claim final-source hosted checks from an earlier commit.

## Local and service suites

| Scope                    | Cases                                                                                                                  | Evidence required                                              |
| ------------------------ | ---------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------- |
| Central storage/Profiles | init idempotence, any cwd, two servers, precedence, changed default during pinned process, protected secret references | CLI unit/process/race output; no project operational files     |
| Reference resolution     | exact ID/title/prefix, duplicates, incomplete search, zero/multiple contracts, execution vs review                     | Deterministic fail-closed results and preserved mutations      |
| Typed material commands  | malformed fields, stale draft CAS, immutable issued data, frozen return refusal                                        | No partial writes, no changed signature/request identity       |
| Draft corrections        | assessment/finding/response set/remove/list, referenced-evidence deletion refusal, stale/cross-section cursors         | CLI-only correction without proof changes or dangling links   |
| Focal continuation       | current scope, staged material, latest bounded activity, descending cursor, explicit omissions, separate revisions    | Read-only queries; complete issued obligations retained       |
| Activity append/history  | replay/conflict, sequence, paging, rollback, current authority, cleanup/restart durability                             | Memory, SQLite and real PostgreSQL shared tests                |
| Evidence/review          | measurement types, exact revision/checksum, criterion binding, self-review rejection, non-approval findings            | Existing regression plus CLI-only end-to-end checks            |
| Recovery                 | pending acquire/lease/return, prepared-only return, uncertain acceptance, explicit resolution, issuer rotation         | Original-CID replay and verified receipt cleanup               |
| Migration                | dry-run, collision, pending intent, original bytes, disk/process interruption                                          | Explicit source/destination evidence and safe unchanged source |
| Protocol parity          | append/history/context scope, error, cursor and result equivalence                                                     | HTTP/MCP/SDK integration results and schema generation checks  |
| Skills                   | five bounded bundles, local references, no direct internal edits, installer ownership                                  | Native installer plus Node compatibility tests                 |

Suggested relevant commands, subject to the final source tree:

```sh
go test ./packages/wos-cli ./packages/wos-skill
go test -race ./packages/wos-cli ./packages/wos-skill
node --test packages/wos-skill/test/installer.test.mjs
go test ./packages/wos-core/application ./packages/wos-api/http ./packages/wos-api/mcp ./packages/wos-sdk-go
```

SQL contract tests require the real database setup documented by repository tooling; an environment-driven PostgreSQL skip is pending parity, not a passing backend. Use repository generators/checks for generated source and public schemas. Run the repository's full required checks before claiming integration ready.

## End-to-end acceptance scenario

1. Start an isolated real server/database with the same source. Provision executor and independent reviewer Principals/keys through current server policy; retain approved issuer/ServerID pins outside model context.
2. Initialize once. Operate from a different directory. Select executor interactively, then pin it explicitly for the agent process.
3. Discover bounded available work and acquire one WorkItem. Inspect compact context and exact current criteria.
4. Perform real external work. Append a progress record, a discovered problem, its correction and actual test observation. Add exact immutable artifact/proof references through the CLI.
5. Start a fresh agent session without the old transcript. Read context/status and one bounded relevant history page; resume without a duplicate acquisition.
   Correct a prepared draft through typed setters/removals, retain issuer proof bytes, and reject stale list cursors after that edit. Follow context's newest-first activity cursor using the same descending order and frozen upper revision.
6. Check and finish. Inject/loss-simulate a response at the tested boundary, reconcile the exact original intent and verify one server result, not two.
7. Independently inspect the exact submission and criterion revisions. Return changes requested for one actual unmet obligation; acquire an explicit acknowledged correction and return a new exact delivery, then approve from the independent Principal.
8. Verify WorkItem lifecycle, unchanged parent achievement, durable activity/material/assessment history and accepted local cleanup.
9. Run two pinned independent workers on distinct WorkItems while another process changes the interactive default. Verify authority/data isolation; also prove competing acquisition has one winner.

Do not describe a scenario as completed unless all observed steps ran. Simulated response loss, goroutines, process tests and separate deployed hosts are different evidence levels and must stay labelled.

Exercise the documented boundary for activity rejection/expiry: the exact pending append stays intact and prevents unsafe replacement. Final-return resolution is not an append-closure protocol; do not mark authenticated terminal append closure complete without a reviewed implementation and its own acceptance evidence.

Cover a compact HTTP commit receipt with a valid full original result larger than 256 KiB, followed by local publication failure. Verify exact original registry/digest/fingerprint binding, bounded compact local publication state, successful recovery without a second append, and byte-exact artifact versions/checksums. Historical accepted-result tests must reject a valid issuer signature that binds different submitted material. Context tests must bound activity descriptions/version metadata/inner references, preserve omissions/cursors/exact counters, and avoid any acquire/renew/write side effect.

## Exit gates

No unexplained failure, no undocumented new contract, no weakened signing/review/concurrency behavior, no unsafe migration, no Skills example that requires internal document manipulation, and no completion claim based only on created files. Unsupported services/platforms, human acceptance and provider usage remain explicit limits.
