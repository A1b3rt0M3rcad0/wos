# Correction of an original signed obligation

The coordinator reads the focal Task and supplies its exact ID, current version
and `correction_review_case_id`. Generic `work checkout --next` deliberately
skips corrections: acknowledging the original prior review is required.

```sh
wosctl --profile executor work checkout TASK_UUID --version CURRENT_VERSION --previous-review ORIGINAL_CASE_UUID
wosctl --profile executor work show NEW_CONTRACT_UUID --for-agent --output json
```

Read the original frozen findings and accepted material references. Edit only the
new execution draft. Its `material.correction_responses` must explicitly address
every original finding with `finding_id` and a nonempty `summary`; evidence IDs
or local keys are optional and must belong to actually returned material.
`work finish NEW_CONTRACT_UUID` closes this execution obligation after verified
acceptance; an independent reviewer still needs to assess the new exact submission.
Keep the original scope. New requirements require explicit replanning, not a
correction that silently broadens the contract. Never recreate old authority or
change a frozen request to retry it.
