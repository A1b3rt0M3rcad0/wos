-- Accepted signed-command responses survive short-cache expiry. This table is
-- immutable operational state; it contains public responses and no secret keys.
CREATE TABLE signed_operation_results (
 namespace_id TEXT NOT NULL,
 outcome_id TEXT NOT NULL,
 principal_id TEXT NOT NULL REFERENCES principals(id),
 credential_id TEXT NOT NULL,
 idempotency_key TEXT NOT NULL,
 command_name TEXT NOT NULL,
 fingerprint TEXT NOT NULL,
 command_id TEXT NOT NULL,
 outcome_revision TEXT NOT NULL CHECK(outcome_revision!='0'),
 response_json TEXT NOT NULL,
 recorded_at BIGINT NOT NULL,
 PRIMARY KEY(namespace_id,principal_id,idempotency_key),
 FOREIGN KEY(namespace_id,outcome_id) REFERENCES outcomes(namespace_id,id),
 FOREIGN KEY(credential_id,namespace_id) REFERENCES credentials(id,namespace_id)
);
CREATE INDEX signed_operation_results_scope ON signed_operation_results(namespace_id,outcome_id,principal_id,credential_id,idempotency_key);
