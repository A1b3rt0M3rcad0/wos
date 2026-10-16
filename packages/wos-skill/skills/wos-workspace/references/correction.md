# Correct an original signed obligation

Generic `work next` deliberately excludes corrections requiring acknowledgement. The coordinator supplies the exact WorkItem, current version and original review case. Pin the executor Profile:

```sh
wos --profile executor work checkout TASK_UUID --version CURRENT_VERSION --previous-review ORIGINAL_CASE_UUID
wos --profile executor work context --for-agent --output json
wos --profile executor work correction add --finding FINDING_REF --summary "Describe the actual correction"
wos --profile executor work result set --summary "Describe the corrected delivery"
wos --profile executor work check
wos --profile executor work finish
```

Use `--evidence UUID,...` only for actual registered evidence. Address every original frozen finding by its original reference. Preserve the contracted scope and exact artifact versions. New requirements require explicit replanning rather than silently expanding correction work.

A verified accepted correction return closes that execution obligation; it still needs a new independent review of its exact submission/current criterion revisions. Do not reuse the old approval or recreate old authority. After uncertainty reconcile the original frozen return with `work recover --all-pending` before changing material.
