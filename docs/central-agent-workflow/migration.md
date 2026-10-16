# Migration and compatibility

## Goal and boundary

Replace automatic per-project operational state with installation-scoped state without deleting the original source or forging authority. Migration is an explicit operator command with a dry-run. It does not infer authorization from found files and does not transmit credentials to a newly discovered destination.

Existing regression anchors are `packages/wos-cli/migration_v2_test.go`, `migration_convert_v2_test.go`, `migration_legacy_v2.go`, `pending_acquisition_v2.go`, `publication_v2.go`, `profile_trust_v2.go` and immutable historical fixtures. Reuse their original-intention and protected-file rules.

## Operator commands

```sh
wos init --from /absolute/legacy-project --dry-run
wos init --from /absolute/legacy-project --writers-drained
```

The implemented initial import targets signed schema-2 state. Schema-1 state requires its explicit original recovery/conversion path first. New clients mark the preserved source as retired against further supported writes; older binaries cannot observe a new marker and **must be drained**. Relative external/mounted secret references are refused if they violate source/destination confinement. A source/credential failure is a safe blocker, not permission to relocate secret bytes.

## Required sequence

1. Initialize the installation directory and inventory a specifically selected source path read-only. Reject symlinks, duplicate stems, case-insensitive Profile collision and unknown content rather than overwriting it.
2. Dry-run reports source schema, Profiles, contracts, pending intentions, proposed destination names and blockers. Redact tokens/private keys/payloads from output.
3. Require an explicit apply action. Lock source/destination against cooperating writers; reject an existing conflicting destination. Copy verified bytes or apply a versioned conversion with an explicit mapping/report.
4. Preserve original ServerID/origin, Namespace, Principal, credential/key references, issuer history, exact issued proof, frozen requests, request IDs/idempotency keys and signed receipts. Moving state cannot turn a new server into the original destination.
5. Resolve relative protected secret references safely. Never copy private key/token values into general operational storage just to make relative references work.
6. Verify imported data before publishing the new Profile/index. Interrupted migration is resumable or fails with original source intact; no partial import becomes a new authoritative active Profile.
7. Leave source data in place. Only explicit later operator cleanup may remove a verified source, and never while unresolved intentions rely on it.

## Compatibility policy

- Ordinary central commands do not silently scan parent directories for an old `.wos` or fall back after a missing Profile.
- Existing explicit legacy workspace/recovery paths may remain for safe historical recovery. They are compatibility mechanisms, not the new primary agent workflow.
- Legacy unsigned history remains labelled `legacy_unsigned`; it is not re-signed as an originally signed contract and cannot authorize signed mutations.
- No Namespace protocol downgrade, server policy change or authority transfer is performed by local migration.
- Frozen activity append payloads/keys and accepted-but-unpublished results are recovery material too. Migration preserves them rather than retrying under a new scope or treating source retirement as server closure. Final-return terminal resolution cannot close an expired/rejected activity append; that unresolved state requires operator intervention and remains intact.
- If a source cannot be converted safely, provide a precise blocker and bounded original recovery command. Do not call migration complete for a file-only happy path.

## Acceptance

Cover dry-run zero writes, repeated apply, two servers/Namespaces, name collision, active/inactive contracts, pending acquisition/renewal/return/resolution, prepared-only return, relative protected secret references, disk full/process death, symlink escape and unknown files. Compare protected source/frozen bytes exactly before/after. Recover one original uncertain operation through the imported Profile without reacquiring or changing its destination. Run native Windows path/case/locking gates separately.
