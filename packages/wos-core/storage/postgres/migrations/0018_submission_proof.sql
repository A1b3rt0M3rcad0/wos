ALTER TABLE criterion_assessments ADD COLUMN submission_id TEXT;
ALTER TABLE criterion_assessments ADD COLUMN submission_digest TEXT NOT NULL DEFAULT '';
ALTER TABLE conclusions ADD COLUMN submission_id TEXT;
ALTER TABLE conclusions ADD COLUMN submission_digest TEXT NOT NULL DEFAULT '';
