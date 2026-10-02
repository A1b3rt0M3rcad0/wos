ALTER TABLE criterion_revisions
ADD COLUMN status TEXT NOT NULL DEFAULT 'active'
CHECK (status IN ('active','retired'));

UPDATE criterion_revisions
SET status = (
    SELECT c.status
    FROM success_criteria c
    WHERE c.namespace_id = criterion_revisions.namespace_id
      AND c.outcome_id = criterion_revisions.outcome_id
      AND c.id = criterion_revisions.criterion_id
      AND c.criterion_revision = criterion_revisions.criterion_revision
)
WHERE EXISTS (
    SELECT 1
    FROM success_criteria c
    WHERE c.namespace_id = criterion_revisions.namespace_id
      AND c.outcome_id = criterion_revisions.outcome_id
      AND c.id = criterion_revisions.criterion_id
      AND c.criterion_revision = criterion_revisions.criterion_revision
);
