-- Bind each Task-owned assessment used by a delivery decision to the exact
-- immutable delivery and correction round it reviewed. Historical assessments
-- remain unbound; they cannot approve a new pending delivery.
ALTER TABLE criterion_assessments ADD COLUMN task_delivery_id TEXT;
ALTER TABLE criterion_assessments ADD COLUMN task_delivery_digest TEXT NOT NULL DEFAULT '';
ALTER TABLE criterion_assessments ADD COLUMN task_delivery_round TEXT NOT NULL DEFAULT '0';

-- Enforce complete references and ensure that the delivery, criterion owner,
-- criterion revision, digest, and round all belong to the same immutable fact.
CREATE TRIGGER criterion_assessment_task_delivery_binding_insert
BEFORE INSERT ON criterion_assessments
WHEN
    (NEW.task_delivery_id IS NULL AND (NEW.task_delivery_digest <> '' OR NEW.task_delivery_round <> '0'))
    OR
    (NEW.task_delivery_id IS NOT NULL AND (
        NEW.task_delivery_digest = ''
        OR length(NEW.task_delivery_digest) <> 71
        OR substr(NEW.task_delivery_digest, 1, 7) <> 'sha256:'
        OR substr(NEW.task_delivery_digest, 8) GLOB '*[^0-9a-f]*'
        OR length(NEW.task_delivery_round) NOT BETWEEN 1 AND 20
        OR NEW.task_delivery_round GLOB '*[^0-9]*'
        OR NEW.task_delivery_round = '0'
        OR substr(NEW.task_delivery_round, 1, 1) = '0'
        OR (length(NEW.task_delivery_round) = 20 AND NEW.task_delivery_round > '18446744073709551615')
        OR NOT EXISTS (
            SELECT 1
            FROM task_deliveries d
            JOIN success_criteria c
              ON c.namespace_id = d.namespace_id
             AND c.outcome_id = d.outcome_id
             AND c.id = NEW.criterion_id
             AND c.owner_kind = 'work_item'
             AND c.owner_id = d.work_item_id
            JOIN task_delivery_criteria dc
              ON dc.namespace_id = d.namespace_id
             AND dc.outcome_id = d.outcome_id
             AND dc.delivery_id = d.id
             AND dc.criterion_id = NEW.criterion_id
             AND dc.criterion_revision = NEW.criterion_revision
            WHERE d.namespace_id = NEW.namespace_id
              AND d.outcome_id = NEW.outcome_id
              AND d.id = NEW.task_delivery_id
              AND d.delivery_digest = NEW.task_delivery_digest
              AND d.delivery_round = NEW.task_delivery_round
        )
    ))
BEGIN
    SELECT RAISE(ABORT, 'criterion assessment TaskDelivery binding is invalid');
END;

-- Keep the bound delivery identity immutable, and revalidate the complete
-- relation whenever any assessment field changes. In particular, changing
-- criterion_id or criterion_revision must not detach a bound assessment from
-- the TaskDelivery snapshot that it claims to review.
CREATE TRIGGER criterion_assessment_task_delivery_binding_update
BEFORE UPDATE ON criterion_assessments
WHEN
    NEW.task_delivery_id IS NOT OLD.task_delivery_id
    OR NEW.task_delivery_digest <> OLD.task_delivery_digest
    OR NEW.task_delivery_round <> OLD.task_delivery_round
    OR (NEW.task_delivery_id IS NULL AND (NEW.task_delivery_digest <> '' OR NEW.task_delivery_round <> '0'))
    OR (NEW.task_delivery_id IS NOT NULL AND (
        NEW.task_delivery_digest = ''
        OR length(NEW.task_delivery_digest) <> 71
        OR substr(NEW.task_delivery_digest, 1, 7) <> 'sha256:'
        OR substr(NEW.task_delivery_digest, 8) GLOB '*[^0-9a-f]*'
        OR length(NEW.task_delivery_round) NOT BETWEEN 1 AND 20
        OR NEW.task_delivery_round GLOB '*[^0-9]*'
        OR NEW.task_delivery_round = '0'
        OR substr(NEW.task_delivery_round, 1, 1) = '0'
        OR (length(NEW.task_delivery_round) = 20 AND NEW.task_delivery_round > '18446744073709551615')
        OR NOT EXISTS (
            SELECT 1
            FROM task_deliveries d
            JOIN success_criteria c
              ON c.namespace_id = d.namespace_id
             AND c.outcome_id = d.outcome_id
             AND c.id = NEW.criterion_id
             AND c.owner_kind = 'work_item'
             AND c.owner_id = d.work_item_id
            JOIN task_delivery_criteria dc
              ON dc.namespace_id = d.namespace_id
             AND dc.outcome_id = d.outcome_id
             AND dc.delivery_id = d.id
             AND dc.criterion_id = NEW.criterion_id
             AND dc.criterion_revision = NEW.criterion_revision
            WHERE d.namespace_id = NEW.namespace_id
              AND d.outcome_id = NEW.outcome_id
              AND d.id = NEW.task_delivery_id
              AND d.delivery_digest = NEW.task_delivery_digest
              AND d.delivery_round = NEW.task_delivery_round
        )
    ))
BEGIN
    SELECT RAISE(ABORT, 'criterion assessment TaskDelivery binding is immutable or invalid');
END;
