-- Bind new contracts_v1 authorities to the trusted canonical credential.
-- Existing contracts remain explicitly unbound; ownership is never inferred
-- from the holder Principal or Actor.
ALTER TABLE work_contracts ADD COLUMN credential_binding_version INTEGER NOT NULL DEFAULT 0
  CHECK (credential_binding_version IN (0, 1));

ALTER TABLE work_contracts ADD COLUMN holder_credential_id TEXT
  CHECK (
    (credential_binding_version = 0 AND holder_credential_id IS NULL)
    OR
    (credential_binding_version = 1 AND holder_credential_id IS NOT NULL AND length(holder_credential_id) = 36)
  );

ALTER TABLE work_contracts ADD COLUMN session_credential_id TEXT
  CHECK (
    (credential_binding_version = 0 AND session_credential_id IS NULL)
    OR
    (credential_binding_version = 1 AND (session_credential_id IS NULL OR length(session_credential_id) = 36))
  );
