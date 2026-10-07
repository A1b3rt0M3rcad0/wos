CREATE TABLE credentials (
 id TEXT PRIMARY KEY, namespace_id TEXT NOT NULL REFERENCES namespaces(id), principal_id TEXT NOT NULL REFERENCES principals(id),
 actor_json TEXT NOT NULL, digest TEXT NOT NULL UNIQUE,
 expires_at BIGINT NOT NULL, revoked INTEGER NOT NULL DEFAULT 0 CHECK (revoked IN (0,1))
);
CREATE TABLE namespace_grants (
 namespace_id TEXT NOT NULL REFERENCES namespaces(id), principal_id TEXT NOT NULL REFERENCES principals(id),
 permissions_json TEXT NOT NULL, PRIMARY KEY(namespace_id,principal_id)
);
CREATE INDEX namespace_grants_principal ON namespace_grants(principal_id,namespace_id);
