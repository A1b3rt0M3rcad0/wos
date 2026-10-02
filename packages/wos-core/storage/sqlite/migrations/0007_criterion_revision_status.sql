ALTER TABLE criterion_revisions
ADD COLUMN status TEXT NOT NULL DEFAULT 'active'
CHECK (status IN ('active','retired'));
