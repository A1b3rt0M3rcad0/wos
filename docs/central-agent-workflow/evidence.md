# Evidence, artifacts and assessment

## Keep the records distinct

| Record              | Meaning                                                                     | Does not establish                     |
| ------------------- | --------------------------------------------------------------------------- | -------------------------------------- |
| Activity            | A participant reports something happened                                    | Independent truth or quality           |
| Artifact            | A delivered material reference at an exact revision/checksum                | That required criteria were met        |
| Evidence            | Observation/source, with provenance and optional material binding           | Acceptance merely because it exists    |
| EvidenceLink        | This observation supports, contradicts or contextualizes a target/criterion | A global stance for the whole evidence |
| CriterionAssessment | Authorized explicit evaluation of a criterion's exact revision/material     | Automatic parent achievement           |

## Existing integration points

`packages/wos-core/domain/documentary.go` and `signed_material.go` define documentary and return material. `packages/wos-core/application/documentary_records.go`, `documentary_compound.go`, `signed_return.go`, and `signed_review_return.go` own registration and exact-submission validation. Storage uses `packages/wos-core/storage/{memory,sqlite,postgres}/documentary.go`. CLI validators include `evidence_preflight_v2_test.go`, `finding_preflight_v2_test.go`, and `return_preflight_v2.go`; do not regress their accepted constraints.

## Operational commands and durable meaning

Expose `work artifact add`, `work evidence add/list`, and criterion linkage through typed flags. Include artifact URI/type plus actual immutable source version/content checksum when available; evidence type/description/source and optional criterion/activity links must be validated. Execution `work artifact/evidence add` registers immediately via `AppendWorkActivity` and returns durable IDs; uncertain operations remain exact recoverable intentions. Review artifact/evidence additions are local staging in `SignedReviewMaterial` and return `local_key` with `remote_committed: false`; they become durable only on accepted signed review finish. Assessments can reference registered UUIDs or returned staged `local:LOCAL_KEY` references.

The CLI must support linking evidence to the originating activity and the executed WorkItem, and selecting proof for current success criteria. A file path without an immutable version is not an exact artifact revision. A commit mentioned only in an activity summary does not substitute for an artifact source binding.

```sh
wos work artifact add --name "Change set" --uri ACTUAL_URI --artifact-type code --source-version ACTUAL_COMMIT
wos work evidence add --type test_result --description "Describe actual checks and results" --uri ACTUAL_REPORT_URI --criterion ORIGINAL_CRITERION_REF
wos work evidence list --limit 10 --output json
```

Use the same pinned Profile on each command. After local cleanup, server reads use the exact `--work WORK_ITEM_UUID` and explicit Outcome scope when necessary. Set `work result set --summary TEXT` before check/finish; advisory progress alone is insufficient.

## Validation steps

1. Reuse Domain evidence type/measurement validation. `test_result` describes actual test observations; numerical `measurement` requires its measurement value and cannot masquerade as a test result.
2. Resolve local material keys only through CLI-owned registration/return flow. Reject unknown or repeated keys and preserve canonical returned IDs.
3. Validate target scope, artifact revision/checksum, original criterion ID/revision and EvidenceLink stance. Never fetch arbitrary artifact URLs inside a transaction or execute an artifact in WOS.
4. Preserve immutable proof and explicit retraction/supersession. Corrected evidence is a new observation; history is not overwritten.
5. Approval binds the exact submitted material and original current criterion revision. Changed material requires a new explicit review; no approval carries forward silently.
6. Reviewer complementary records require a source version/checksum in the delivered artifact set. If no matching exact source exists, omit complementary records rather than inventing a binding.

## Review and correction gates

Only an authorized independent reviewer can meet required criteria under independent-review policy. Different profile names, keys, sessions or subagents belonging to one Principal do not establish independence. `changes_requested`/`inconclusive` carry reason/findings, not passing criterion assessments; approval carries actual explicit assessments. A finding names one original criterion or requirement, never both.

Correction acquisition explicitly acknowledges the original review case. Responses address every original frozen finding and preserve scope; new requirements require replanning. Generic next-work discovery cannot silently absorb corrections.

## Acceptance

Run malformed-measurement, duplicate material key, wrong scope, stale criterion revision, changed submission, forged source binding, self-review and correction round-trip tests. Prove malformed commands leave issued/frozen bytes and pending operations unchanged. Check server history/evidence remain accessible after accepted local cleanup. Document deferred relation forms explicitly rather than claiming every association is implemented.
