-- A credential's initial permission policy and safe administrative metadata are
-- persisted in the same transaction as the bearer digest. Existing credentials
-- remain readable; only newly issued credentials require a metadata row.
CREATE TABLE credential_metadata (
    namespace_id TEXT NOT NULL,
    credential_id TEXT NOT NULL,
    name TEXT NOT NULL CHECK (length(name) BETWEEN 1 AND 100),
    description TEXT NOT NULL DEFAULT '' CHECK (length(description) <= 1000),
    created_at BIGINT NOT NULL,
    created_by TEXT NOT NULL REFERENCES principals(id),
    created_by_actor_json TEXT NOT NULL,
    role_template_id TEXT,
    role_template_version BIGINT NOT NULL DEFAULT 0 CHECK (role_template_version >= 0),
    policy_revision BIGINT NOT NULL CHECK (policy_revision > 0),
    PRIMARY KEY (namespace_id, credential_id),
    FOREIGN KEY (credential_id, namespace_id) REFERENCES credentials(id, namespace_id),
    FOREIGN KEY (namespace_id, credential_id) REFERENCES credential_policies(namespace_id, credential_id),
    CHECK ((role_template_id IS NULL AND role_template_version = 0) OR
           (role_template_id IS NOT NULL AND length(role_template_id) BETWEEN 1 AND 64 AND role_template_version > 0))
);

CREATE INDEX credential_metadata_namespace_created
    ON credential_metadata(namespace_id, created_at, credential_id);
