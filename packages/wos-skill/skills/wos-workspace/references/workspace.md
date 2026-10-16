# Record material through the CLI

All operations use the approved pinned Profile. Never inspect or change internal serialized documents. Material commands validate typed values, preserve immutable issued data and refuse changes to frozen/uncertain returns.

```sh
wos --profile executor work progress set --summary "Describe durable progress" --next-action "Describe the next permitted action"
wos --profile executor work artifact add --name "Change set" --uri ACTUAL_IMMUTABLE_URI --artifact-type code --source-version ACTUAL_COMMIT
wos --profile executor work evidence add --type test_result --description "Describe checks actually run and their result" --uri ACTUAL_REPORT_URI --criterion ORIGINAL_CRITERION_REF
wos --profile executor work evidence list --limit 10 --output json
wos --profile executor work result set --summary "Describe the actual delivered result"
wos --profile executor work status --for-agent --output json
wos --profile executor work check
wos --profile executor work finish
```

Replace placeholders only with actual references; do not invent a passing test, checksum or artifact. Add a contract reference or `--contract REF` when selection is ambiguous. Use `work log add --type validation --description TEXT` for an activity declaration about actual validation; this is separate from Evidence and CriterionAssessment.

`test_result` evidence is a factual test observation; numeric measurement is a different evidence type with its own validated shape. Unsupported material flags/types must be rejected rather than pushed into internal files. An evidence link to a criterion does not approve it. Source versions/checksums identify exact material, not a mutable working tree.

Execution `work log/evidence/artifact add` registers immediately on the Server and returns durable IDs. Reconcile an uncertain registration with `work recover --all-pending`; never repeat it as a new intent. Review material additions are local staging until accepted signed review finish, and return local keys instead. Consult command output for durable registered IDs versus local preparation. `work result set --summary TEXT` is required for delivery; progress alone is advisory. Use current context/status to derive obligations; do not maintain a second manual business pending list. A successful check is a preflight that can become stale. Server authorization and exact signed receipt decide acceptance. Independent review can remain pending after accepted execution; parent achievement is explicit.

After interruption use `work recover --all-pending` with the same Profile before changing material. Foreground keepalive is an explicitly supervised host process; there is no hidden WOS agent scheduler.
