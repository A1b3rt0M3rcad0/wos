-- An authorized withdrawal resolves one exact pending unsigned TaskDelivery.
-- The delivery and withdrawal remain separate immutable records; WorkItem
-- navigation is rebuilt below so a resolved delivery points at exactly one
-- decision or withdrawal.
CREATE TABLE task_delivery_withdrawals (
    id TEXT PRIMARY KEY,
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    work_item_id TEXT NOT NULL,
    delivery_id TEXT NOT NULL,
    delivery_round TEXT NOT NULL CHECK (
        length(delivery_round) BETWEEN 1 AND 20
        AND delivery_round NOT GLOB '*[^0-9]*'
        AND delivery_round <> '0'
        AND substr(delivery_round,1,1) <> '0'
        AND (length(delivery_round) < 20 OR delivery_round <= '18446744073709551615')
    ),
    delivery_digest TEXT NOT NULL CHECK (length(delivery_digest)=71 AND substr(delivery_digest,1,7)='sha256:'),
    payload_json TEXT NOT NULL CHECK (json_valid(payload_json)),
    UNIQUE(namespace_id,outcome_id,id),
    UNIQUE(namespace_id,outcome_id,work_item_id,delivery_id),
    UNIQUE(namespace_id,outcome_id,work_item_id,delivery_id,id),
    FOREIGN KEY(namespace_id,outcome_id,work_item_id,delivery_id,delivery_round,delivery_digest)
        REFERENCES task_deliveries(namespace_id,outcome_id,work_item_id,id,delivery_round,delivery_digest)
);
CREATE INDEX task_delivery_withdrawals_history
    ON task_delivery_withdrawals(namespace_id,outcome_id,work_item_id,delivery_id,id);

-- Persist each correction's exact predecessor as a typed scoped relation. The
-- TaskDelivery payload remains canonical; this relation lets the database
-- retain and validate the immutable correction chain with foreign keys.
CREATE TABLE task_delivery_predecessors (
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    work_item_id TEXT NOT NULL,
    delivery_id TEXT NOT NULL,
    delivery_round TEXT NOT NULL CHECK (
        length(delivery_round) BETWEEN 1 AND 20
        AND delivery_round NOT GLOB '*[^0-9]*'
        AND delivery_round <> '0'
        AND delivery_round <> '1'
        AND substr(delivery_round,1,1) <> '0'
        AND (length(delivery_round) < 20 OR delivery_round <= '18446744073709551615')
    ),
    delivery_digest TEXT NOT NULL,
    predecessor_delivery_id TEXT NOT NULL,
    predecessor_round TEXT NOT NULL CHECK (
        length(predecessor_round) BETWEEN 1 AND 20
        AND predecessor_round NOT GLOB '*[^0-9]*'
        AND predecessor_round <> '0'
        AND substr(predecessor_round,1,1) <> '0'
        AND (length(predecessor_round) < 20 OR predecessor_round <= '18446744073709551615')
    ),
    predecessor_digest TEXT NOT NULL,
    PRIMARY KEY(namespace_id,outcome_id,work_item_id,delivery_id),
    CHECK (delivery_id <> predecessor_delivery_id),
    FOREIGN KEY(namespace_id,outcome_id,work_item_id,delivery_id,delivery_round,delivery_digest)
        REFERENCES task_deliveries(namespace_id,outcome_id,work_item_id,id,delivery_round,delivery_digest),
    FOREIGN KEY(namespace_id,outcome_id,work_item_id,predecessor_delivery_id,predecessor_round,predecessor_digest)
        REFERENCES task_deliveries(namespace_id,outcome_id,work_item_id,id,delivery_round,delivery_digest)
);
CREATE INDEX task_delivery_predecessors_by_parent
    ON task_delivery_predecessors(namespace_id,outcome_id,work_item_id,predecessor_delivery_id,delivery_id);

-- Reject malformed historical chains before adding their normalized rows.
CREATE TABLE task_delivery_predecessor_migration_guard (
    valid INTEGER NOT NULL CHECK (valid=1)
);
INSERT INTO task_delivery_predecessor_migration_guard(valid)
SELECT CASE WHEN EXISTS (
    SELECT 1 FROM task_deliveries current
    WHERE (current.delivery_round='1' AND json_extract(current.payload_json,'$.supersedes_delivery_id') IS NOT NULL)
       OR (current.delivery_round<>'1' AND NOT EXISTS (
            SELECT 1 FROM task_deliveries predecessor
            WHERE predecessor.namespace_id=current.namespace_id
              AND predecessor.outcome_id=current.outcome_id
              AND predecessor.work_item_id=current.work_item_id
              AND predecessor.id=json_extract(current.payload_json,'$.supersedes_delivery_id')
              AND (length(predecessor.delivery_round)<length(current.delivery_round)
                   OR (length(predecessor.delivery_round)=length(current.delivery_round) AND predecessor.delivery_round<current.delivery_round))
              AND NOT EXISTS (
                  SELECT 1 FROM task_deliveries nearer
                  WHERE nearer.namespace_id=current.namespace_id
                    AND nearer.outcome_id=current.outcome_id
                    AND nearer.work_item_id=current.work_item_id
                    AND (length(nearer.delivery_round)>length(predecessor.delivery_round)
                         OR (length(nearer.delivery_round)=length(predecessor.delivery_round) AND nearer.delivery_round>predecessor.delivery_round))
                    AND (length(nearer.delivery_round)<length(current.delivery_round)
                         OR (length(nearer.delivery_round)=length(current.delivery_round) AND nearer.delivery_round<current.delivery_round))
              )
       ))
) THEN 0 ELSE 1 END;
DROP TABLE task_delivery_predecessor_migration_guard;
INSERT INTO task_delivery_predecessors(namespace_id,outcome_id,work_item_id,delivery_id,delivery_round,delivery_digest,predecessor_delivery_id,predecessor_round,predecessor_digest)
SELECT current.namespace_id,current.outcome_id,current.work_item_id,current.id,current.delivery_round,current.delivery_digest,
       predecessor.id,predecessor.delivery_round,predecessor.delivery_digest
FROM task_deliveries current
JOIN task_deliveries predecessor
  ON predecessor.namespace_id=current.namespace_id
 AND predecessor.outcome_id=current.outcome_id
 AND predecessor.work_item_id=current.work_item_id
 AND predecessor.id=json_extract(current.payload_json,'$.supersedes_delivery_id')
WHERE current.delivery_round<>'1';

-- Rebuild the navigation table to add its withdrawal pointer, composite
-- foreign key and XOR resolution constraint while preserving existing rows.
CREATE TABLE task_delivery_work_state_replacement (
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    work_item_id TEXT NOT NULL,
    pending_delivery_id TEXT,
    latest_delivery_id TEXT,
    latest_decision_id TEXT,
    latest_withdrawal_id TEXT,
    delivery_round TEXT NOT NULL DEFAULT '0' CHECK (
        length(delivery_round) BETWEEN 1 AND 20
        AND delivery_round NOT GLOB '*[^0-9]*'
        AND (delivery_round='0' OR substr(delivery_round,1,1)<>'0')
        AND (length(delivery_round)<20 OR delivery_round<='18446744073709551615')
    ),
    PRIMARY KEY(namespace_id,outcome_id,work_item_id),
    FOREIGN KEY(namespace_id,outcome_id,work_item_id)
        REFERENCES work_items(namespace_id,outcome_id,id),
    FOREIGN KEY(namespace_id,outcome_id,work_item_id,pending_delivery_id)
        REFERENCES task_deliveries(namespace_id,outcome_id,work_item_id,id),
    FOREIGN KEY(namespace_id,outcome_id,work_item_id,latest_delivery_id)
        REFERENCES task_deliveries(namespace_id,outcome_id,work_item_id,id),
    FOREIGN KEY(namespace_id,outcome_id,work_item_id,latest_delivery_id,latest_decision_id)
        REFERENCES task_delivery_decisions(namespace_id,outcome_id,work_item_id,delivery_id,id),
    FOREIGN KEY(namespace_id,outcome_id,work_item_id,latest_delivery_id,latest_withdrawal_id)
        REFERENCES task_delivery_withdrawals(namespace_id,outcome_id,work_item_id,delivery_id,id),
    CHECK (
        (delivery_round='0' AND pending_delivery_id IS NULL AND latest_delivery_id IS NULL AND latest_decision_id IS NULL AND latest_withdrawal_id IS NULL)
        OR
        (delivery_round<>'0' AND latest_delivery_id IS NOT NULL AND
            ((pending_delivery_id IS NOT NULL AND pending_delivery_id=latest_delivery_id AND latest_decision_id IS NULL AND latest_withdrawal_id IS NULL)
             OR (pending_delivery_id IS NULL AND latest_decision_id IS NOT NULL AND latest_withdrawal_id IS NULL)
             OR (pending_delivery_id IS NULL AND latest_decision_id IS NULL AND latest_withdrawal_id IS NOT NULL)))
    )
);
INSERT INTO task_delivery_work_state_replacement(namespace_id,outcome_id,work_item_id,pending_delivery_id,latest_delivery_id,latest_decision_id,latest_withdrawal_id,delivery_round)
SELECT namespace_id,outcome_id,work_item_id,pending_delivery_id,latest_delivery_id,latest_decision_id,NULL,delivery_round
FROM task_delivery_work_state;
DROP TABLE task_delivery_work_state;
ALTER TABLE task_delivery_work_state_replacement RENAME TO task_delivery_work_state;
CREATE UNIQUE INDEX task_delivery_one_pending_per_work_item
    ON task_delivery_work_state(namespace_id,outcome_id,work_item_id)
    WHERE pending_delivery_id IS NOT NULL;
CREATE INDEX task_delivery_pending_queue
    ON task_delivery_work_state(namespace_id,outcome_id,work_item_id,latest_delivery_id)
    WHERE pending_delivery_id IS NOT NULL;

-- POSTGRESGEN_WITHDRAWAL_TRIGGERS_BEGIN
CREATE TRIGGER task_delivery_withdrawals_immutable_update BEFORE UPDATE ON task_delivery_withdrawals
BEGIN SELECT RAISE(ABORT, 'TaskDeliveryWithdrawal is immutable'); END;
CREATE TRIGGER task_delivery_withdrawals_immutable_delete BEFORE DELETE ON task_delivery_withdrawals
BEGIN SELECT RAISE(ABORT, 'TaskDeliveryWithdrawal is immutable'); END;
CREATE TRIGGER task_delivery_predecessors_immutable_update BEFORE UPDATE ON task_delivery_predecessors
BEGIN SELECT RAISE(ABORT, 'TaskDelivery predecessor is immutable'); END;
CREATE TRIGGER task_delivery_predecessors_immutable_delete BEFORE DELETE ON task_delivery_predecessors
BEGIN SELECT RAISE(ABORT, 'TaskDelivery predecessor is immutable'); END;
CREATE TRIGGER task_delivery_predecessor_insert_first AFTER INSERT ON task_deliveries
WHEN NEW.delivery_round='1'
BEGIN
    SELECT CASE WHEN json_extract(NEW.payload_json,'$.supersedes_delivery_id') IS NOT NULL
        THEN RAISE(ABORT, 'first TaskDelivery round cannot have a predecessor') END;
    SELECT CASE WHEN EXISTS (
        SELECT 1 FROM task_delivery_work_state s
        WHERE s.namespace_id=NEW.namespace_id AND s.outcome_id=NEW.outcome_id
          AND s.work_item_id=NEW.work_item_id
          AND (s.pending_delivery_id IS NOT NULL OR s.latest_delivery_id IS NOT NULL OR s.delivery_round<>'0')
    ) THEN RAISE(ABORT, 'first TaskDelivery round requires empty history') END;
END;
CREATE TRIGGER task_delivery_predecessor_insert_correction AFTER INSERT ON task_deliveries
WHEN NEW.delivery_round<>'1'
BEGIN
    SELECT CASE WHEN NOT EXISTS (
        SELECT 1 FROM task_delivery_work_state s
        JOIN task_deliveries predecessor
          ON predecessor.namespace_id=s.namespace_id AND predecessor.outcome_id=s.outcome_id
         AND predecessor.work_item_id=s.work_item_id AND predecessor.id=s.latest_delivery_id
        WHERE s.namespace_id=NEW.namespace_id AND s.outcome_id=NEW.outcome_id
          AND s.work_item_id=NEW.work_item_id AND s.pending_delivery_id IS NULL
          AND s.delivery_round=predecessor.delivery_round
          AND predecessor.id=json_extract(NEW.payload_json,'$.supersedes_delivery_id')
          AND (length(NEW.delivery_round)>length(predecessor.delivery_round)
               OR (length(NEW.delivery_round)=length(predecessor.delivery_round) AND NEW.delivery_round>predecessor.delivery_round))
    ) THEN RAISE(ABORT, 'TaskDelivery correction must supersede the latest resolved delivery') END;
    INSERT INTO task_delivery_predecessors(namespace_id,outcome_id,work_item_id,delivery_id,delivery_round,delivery_digest,predecessor_delivery_id,predecessor_round,predecessor_digest)
    SELECT NEW.namespace_id,NEW.outcome_id,NEW.work_item_id,NEW.id,NEW.delivery_round,NEW.delivery_digest,
           predecessor.id,predecessor.delivery_round,predecessor.delivery_digest
    FROM task_delivery_work_state s
    JOIN task_deliveries predecessor
      ON predecessor.namespace_id=s.namespace_id AND predecessor.outcome_id=s.outcome_id
     AND predecessor.work_item_id=s.work_item_id AND predecessor.id=s.latest_delivery_id
    WHERE s.namespace_id=NEW.namespace_id AND s.outcome_id=NEW.outcome_id
      AND s.work_item_id=NEW.work_item_id AND s.pending_delivery_id IS NULL
      AND s.delivery_round=predecessor.delivery_round
      AND predecessor.id=json_extract(NEW.payload_json,'$.supersedes_delivery_id');
END;
CREATE TRIGGER task_delivery_decision_resolution_guard BEFORE INSERT ON task_delivery_decisions
WHEN EXISTS (
    SELECT 1 FROM task_delivery_withdrawals w
    WHERE w.namespace_id=NEW.namespace_id AND w.outcome_id=NEW.outcome_id
      AND w.work_item_id=NEW.work_item_id AND w.delivery_id=NEW.delivery_id
) OR NOT EXISTS (
    SELECT 1 FROM task_delivery_work_state s
    WHERE s.namespace_id=NEW.namespace_id AND s.outcome_id=NEW.outcome_id
      AND s.work_item_id=NEW.work_item_id AND s.pending_delivery_id=NEW.delivery_id
      AND s.latest_delivery_id=NEW.delivery_id AND s.latest_withdrawal_id IS NULL
)
BEGIN SELECT RAISE(ABORT, 'TaskDelivery already resolved or is not pending'); END;
CREATE TRIGGER task_delivery_withdrawal_resolution_guard BEFORE INSERT ON task_delivery_withdrawals
WHEN EXISTS (
    SELECT 1 FROM task_delivery_decisions d
    WHERE d.namespace_id=NEW.namespace_id AND d.outcome_id=NEW.outcome_id
      AND d.work_item_id=NEW.work_item_id AND d.delivery_id=NEW.delivery_id
) OR NOT EXISTS (
    SELECT 1 FROM task_delivery_work_state s
    WHERE s.namespace_id=NEW.namespace_id AND s.outcome_id=NEW.outcome_id
      AND s.work_item_id=NEW.work_item_id AND s.pending_delivery_id=NEW.delivery_id
      AND s.latest_delivery_id=NEW.delivery_id AND s.latest_decision_id IS NULL
)
BEGIN SELECT RAISE(ABORT, 'TaskDelivery already resolved or is not pending'); END;
CREATE TRIGGER task_delivery_work_state_single_pending BEFORE UPDATE OF pending_delivery_id ON task_delivery_work_state
WHEN OLD.pending_delivery_id IS NOT NULL AND NEW.pending_delivery_id IS NOT NULL AND OLD.pending_delivery_id <> NEW.pending_delivery_id
BEGIN SELECT RAISE(ABORT, 'Task already has a pending TaskDelivery'); END;
-- POSTGRESGEN_WITHDRAWAL_TRIGGERS_END
