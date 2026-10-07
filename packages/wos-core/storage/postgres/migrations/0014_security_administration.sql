CREATE TABLE security_admin_results (
 namespace_id TEXT NOT NULL REFERENCES namespaces(id), principal_id TEXT NOT NULL REFERENCES principals(id), idempotency_key TEXT NOT NULL,
 fingerprint TEXT NOT NULL, result_json TEXT NOT NULL, PRIMARY KEY(namespace_id,principal_id,idempotency_key)
);
CREATE TABLE security_audit (
 namespace_id TEXT NOT NULL REFERENCES namespaces(id), namespace_version BIGINT NOT NULL,
 principal_id TEXT NOT NULL REFERENCES principals(id), actor_json TEXT NOT NULL,
 operation TEXT NOT NULL, target TEXT NOT NULL, recorded_at BIGINT NOT NULL,
 PRIMARY KEY(namespace_id,namespace_version)
);
