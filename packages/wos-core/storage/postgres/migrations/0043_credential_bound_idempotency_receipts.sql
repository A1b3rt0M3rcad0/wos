-- Protected work-operation receipts retain the canonical root credential that
-- authorized their mutation. For administrative revocation this is the issuer,
-- not the holder. Existing receipts stay NULL and are rejected for authenticated
-- replay because their authority is unproven.
ALTER TABLE idempotency_records ADD COLUMN authority_credential_id TEXT
  CHECK (authority_credential_id IS NULL OR length(authority_credential_id) = 36);
