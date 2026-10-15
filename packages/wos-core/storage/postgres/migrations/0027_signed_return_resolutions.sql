-- Negative decisions are durable independently of successful operation caches.
CREATE TABLE signed_return_intent_resolutions (
 namespace_id TEXT NOT NULL,
 outcome_id TEXT NOT NULL,
 principal_id TEXT NOT NULL REFERENCES principals(id),
 original_credential_id TEXT NOT NULL,
 original_command_name TEXT NOT NULL CHECK(original_command_name IN ('ReturnSignedWork','ReturnSignedReview')),
 original_idempotency_key TEXT NOT NULL,
 original_request_id TEXT NOT NULL,
 issuer_key_id TEXT NOT NULL,
 state_json TEXT NOT NULL,
 PRIMARY KEY(namespace_id,principal_id,original_credential_id,original_command_name,original_idempotency_key),
 UNIQUE(namespace_id,principal_id,original_request_id),
 FOREIGN KEY(namespace_id,issuer_key_id) REFERENCES signing_keys(namespace_id,id),
 FOREIGN KEY(namespace_id,outcome_id) REFERENCES outcomes(namespace_id,id),
 FOREIGN KEY(original_credential_id,namespace_id) REFERENCES credentials(id,namespace_id)
);
CREATE INDEX signed_return_resolution_scope ON signed_return_intent_resolutions(namespace_id,outcome_id,principal_id,original_credential_id);
