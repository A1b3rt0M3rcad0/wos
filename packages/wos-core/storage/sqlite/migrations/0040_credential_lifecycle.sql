ALTER TABLE credential_metadata ADD COLUMN updated_at BIGINT NOT NULL DEFAULT 0 CHECK (updated_at >= 0);
ALTER TABLE credential_metadata ADD COLUMN metadata_revision BIGINT NOT NULL DEFAULT 1 CHECK (metadata_revision > 0);
ALTER TABLE credential_metadata ADD COLUMN last_used_at BIGINT CHECK (last_used_at IS NULL OR last_used_at >= 0);
ALTER TABLE credential_metadata ADD COLUMN replaced_by_credential_id TEXT;
ALTER TABLE credential_metadata ADD COLUMN replaces_credential_id TEXT;
ALTER TABLE credential_metadata ADD COLUMN rotation_overlap_until BIGINT CHECK (rotation_overlap_until IS NULL OR rotation_overlap_until >= 0);

UPDATE credential_metadata SET updated_at=created_at WHERE updated_at=0;

CREATE INDEX credential_metadata_namespace_last_used
    ON credential_metadata(namespace_id, last_used_at, credential_id);
