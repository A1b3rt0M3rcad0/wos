ALTER TABLE conclusions ADD COLUMN public_id TEXT;

CREATE UNIQUE INDEX idx_conclusions_public_id
    ON conclusions (namespace_id, outcome_id, public_id)
    WHERE public_id IS NOT NULL;
