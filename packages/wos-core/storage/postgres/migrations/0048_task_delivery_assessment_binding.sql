-- Bind each Task-owned assessment used by a delivery decision to the exact
-- immutable delivery and correction round it reviewed. Historical assessments
-- remain unbound; they cannot approve a new pending delivery.
ALTER TABLE criterion_assessments ADD COLUMN task_delivery_id TEXT;
ALTER TABLE criterion_assessments ADD COLUMN task_delivery_digest TEXT NOT NULL DEFAULT '';
ALTER TABLE criterion_assessments ADD COLUMN task_delivery_round TEXT NOT NULL DEFAULT '0';

-- Enforce complete references and ensure that the delivery, criterion owner,
-- criterion revision, digest, and round all belong to the same immutable fact.
CREATE FUNCTION guard_criterion_assessment_task_delivery_binding() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        IF NEW.task_delivery_id IS DISTINCT FROM OLD.task_delivery_id
           OR NEW.task_delivery_digest IS DISTINCT FROM OLD.task_delivery_digest
           OR NEW.task_delivery_round IS DISTINCT FROM OLD.task_delivery_round THEN
            RAISE EXCEPTION 'criterion assessment TaskDelivery binding is immutable';
        END IF;
    END IF;

    IF NEW.task_delivery_id IS NULL THEN
        IF NEW.task_delivery_digest <> '' OR NEW.task_delivery_round <> '0' THEN
            RAISE EXCEPTION 'criterion assessment TaskDelivery binding is incomplete';
        END IF;
        RETURN NEW;
    END IF;

    IF NEW.task_delivery_digest !~ '^sha256:[0-9a-f]{64}$'
       OR NEW.task_delivery_round !~ '^[1-9][0-9]{0,19}$'
       OR (length(NEW.task_delivery_round) = 20 AND NEW.task_delivery_round > '18446744073709551615') THEN
        RAISE EXCEPTION 'criterion assessment TaskDelivery binding is malformed';
    END IF;

    IF NOT EXISTS (
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
    ) THEN
        RAISE EXCEPTION 'criterion assessment TaskDelivery binding is invalid';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER criterion_assessment_task_delivery_binding
BEFORE INSERT OR UPDATE ON criterion_assessments
FOR EACH ROW EXECUTE FUNCTION guard_criterion_assessment_task_delivery_binding();
