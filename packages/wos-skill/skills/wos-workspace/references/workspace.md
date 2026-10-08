# File and recovery examples

.wos/config.yaml contains schema_version: 1, kind: WOSWorkspace, server_url and credential_ref under connection, namespace/default Outcome under scope, and .wos/work as workspace root. Documents use generated packages/wos-cli/schemas. The illustrative implementation-plan examples were refined to typed contract/checkpoint/material fields.

.wos/work/<work_id>/<contract_id>/ contains contract.yaml, checkpoint.yaml, result.yaml, state.json, outbox/ and receipts/. .yaml and .yml are equivalent; duplicate stems are ambiguous and rejected. Do not put operational state in Git by default.

- Timeout: work recover ID, replay exact original intent before editing or creating another.
- Acquire committed but disk failed: work recover ID or work recover --next; restore existing acquisition, not another lease.
- Reviewer pending: preserve submission; optional explicit foreground renewal supervised by host.
- Expired/revoked: stop effects; no takeover/release can restore old authority. New acquisition requires live eligibility and creates a new contract.
- Changed material: preserve local draft, inspect diff and use new documentary keys/new explicit submission.
- New session: status, latest checkpoint, exact necessary references; no prior chat dump required.
