# Migration of a schema-1 workspace

Conversion uses a **separate, explicitly approved schema-2 destination**. Initialize
that destination and enroll its profile using the normal protected onboarding
procedure in `packages/wos-cli/README.md`. Source and destination must resolve to
the same persistent server, Namespace and currently selected credential identity.
Migration does not redirect credentials, generate issuer signatures or assume
that a copied legacy lease grants signed execution authority.

```sh
wosctl workspace migrate --workspace ./legacy-project --to 2 --profile executor --dry-run
wosctl workspace migrate --workspace ./legacy-project --to 2 --profile executor --destination ./signed-project
# After interruption, repeat the same command with the same approved destination.
```

Dry-run inventories exact bytes/digests of config, contract, checkpoint/result
drafts, journals and receipts. It never sends an intention or modifies a file.
Prepared/uncertain intentions and accepted acquisitions without a materialized
contract are reported separately. Reconcile them in the original schema-1
workspace before conversion; migration never acquires replacement work.

Conversion reads each original contract remotely before any publication. It
freezes the source inventory and expected target digests in a bounded pending
operation inside the destination profile. Network calls occur outside local
profile/contract locks. Existing target bytes must match the original frozen
conversion exactly; an editor change interrupts recovery and remains intact.
Only after all published documents and the source inventory match is the pending
marker cleared.

Each imported contract is explicitly `WOSLegacyUnsignedContract` /
`legacy_unsigned`. It combines the contract and drafts, retains exact known
operational record bytes in bounded chunks, and keeps its source reference.
`work list` and `work show` expose a small read-only historical summary. Sign,
finish and lease mutation reject these records. They never become signed specs
or agent returns retrospectively. Original records not attached to a contract,
custom files and the complete original workspace remain in the source.

**No original is deleted.** Compare both inventories and review retained records
before separately authorizing any discard/export of the original workspace.
This preservation is also the rollback path for unsupported/oversized documents.
Do not treat source path references as permission to open another workspace.
Limits: 512 inventory entries, depth 12, 32 MiB source bytes, 100 converted
contracts, 1 MiB per target document, 180 KiB pending manifest. Conflicts fail
explicitly; they do not trigger a partial discard or hidden limit expansion.

Implementation acceptance is tracked in ROADMAP.md. Unit inventory/publication
checks are preliminary evidence; actual transport, process-death, upgrade/restore
and native Windows gates must be recorded before claiming P10 complete.
