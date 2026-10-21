ALTER TABLE namespace_grants ADD COLUMN permission_compatibility_version BIGINT NOT NULL DEFAULT 0
    CHECK (permission_compatibility_version IN (0,1));

CREATE TABLE permission_migration_results (
    namespace_id TEXT NOT NULL REFERENCES namespaces(id),
    principal_id TEXT NOT NULL REFERENCES principals(id),
    idempotency_key TEXT NOT NULL,
    fingerprint TEXT NOT NULL,
    result_json TEXT NOT NULL,
    PRIMARY KEY(namespace_id,principal_id,idempotency_key)
);
