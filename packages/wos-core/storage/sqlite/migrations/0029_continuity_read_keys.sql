-- Exact Go JSON order is distinct from raw provider/ID collation. The bounded
-- Go backfill runs in this same migration transaction before its version row.
ALTER TABLE outcome_external_references ADD COLUMN continuity_sort_key TEXT COLLATE BINARY;
ALTER TABLE outcome_external_references ADD COLUMN continuity_key_digest TEXT;
CREATE UNIQUE INDEX external_continuity_digest ON outcome_external_references(namespace_id,outcome_id,continuity_key_digest);
CREATE INDEX external_continuity_order ON outcome_external_references(namespace_id,outcome_id,continuity_sort_key);
-- Focal owner queries must join the selected criterion, not scan every
-- assessment in an Outcome before applying the criterion's owner predicate.
CREATE INDEX criterion_assessment_by_criterion ON criterion_assessments(namespace_id,outcome_id,criterion_id,assessed_at,id);
