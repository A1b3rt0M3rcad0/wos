-- Preserve authorization provenance and safe before/after state for guarded
-- administrative mutations. Credential identifiers are deliberately not
-- foreign keys so audit history survives credential cleanup.
ALTER TABLE security_audit ADD COLUMN author_credential_id TEXT;
ALTER TABLE security_audit ADD COLUMN root_credential_id TEXT;
ALTER TABLE security_audit ADD COLUMN reason TEXT NOT NULL DEFAULT '' CHECK (length(reason) <= 500);
ALTER TABLE security_audit ADD COLUMN before_json TEXT NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(before_json::jsonb) IS NOT NULL);
ALTER TABLE security_audit ADD COLUMN after_json TEXT NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(after_json::jsonb) IS NOT NULL);
