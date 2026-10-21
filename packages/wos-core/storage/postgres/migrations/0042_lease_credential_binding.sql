-- Persist the provenance required to bind a legacy WorkLease to the trusted
-- credentials resolved by Application. Existing leases remain explicitly
-- unbound: this expand migration does not infer credential ownership from a
-- principal or actor record.
ALTER TABLE work_items ADD COLUMN lease_credential_binding_version INTEGER NOT NULL DEFAULT 0
  CHECK (lease_credential_binding_version IN (0, 1));

ALTER TABLE work_items ADD COLUMN lease_holder_credential_id TEXT
  CHECK (
    (lease_claim_id IS NULL AND lease_credential_binding_version = 0 AND lease_holder_credential_id IS NULL)
    OR
    (lease_claim_id IS NOT NULL AND (
      (lease_credential_binding_version = 0 AND lease_holder_credential_id IS NULL)
      OR
      (lease_credential_binding_version = 1 AND lease_holder_credential_id IS NOT NULL AND length(lease_holder_credential_id) = 36)
    ))
  );

ALTER TABLE work_items ADD COLUMN lease_session_credential_id TEXT
  CHECK (
    (lease_claim_id IS NULL AND lease_credential_binding_version = 0 AND lease_holder_credential_id IS NULL AND lease_session_credential_id IS NULL)
    OR
    (lease_claim_id IS NOT NULL AND (
      (lease_credential_binding_version = 0 AND lease_holder_credential_id IS NULL AND lease_session_credential_id IS NULL)
      OR
      (lease_credential_binding_version = 1 AND lease_holder_credential_id IS NOT NULL AND length(lease_holder_credential_id) = 36 AND (lease_session_credential_id IS NULL OR length(lease_session_credential_id) = 36))
    ))
  );
