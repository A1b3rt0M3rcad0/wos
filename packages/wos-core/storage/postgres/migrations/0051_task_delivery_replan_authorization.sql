-- A withdrawal resolves the current review round but does not itself reopen
-- planning or authorize another execution. This marker is set only when the
-- Task is explicitly deferred to Backlog, then survives activation and claim
-- until a new immutable delivery starts the next round.
ALTER TABLE task_delivery_work_state
    ADD COLUMN replan_authorized INTEGER NOT NULL DEFAULT 0
    CHECK (
        replan_authorized IN (0,1)
        AND (replan_authorized=0 OR (latest_withdrawal_id IS NOT NULL AND pending_delivery_id IS NULL))
    );

-- Before this marker existed, the only valid way to retain a withdrawn
-- delivery while entering Backlog was DeferWorkItem. Preserve that already
-- explicit replan transition when upgrading existing stores.
UPDATE task_delivery_work_state
SET replan_authorized=1
WHERE latest_withdrawal_id IS NOT NULL
  AND EXISTS (
      SELECT 1 FROM work_items w
      WHERE w.namespace_id=task_delivery_work_state.namespace_id
        AND w.outcome_id=task_delivery_work_state.outcome_id
        AND w.id=task_delivery_work_state.work_item_id
        AND w.lifecycle='backlog'
  );
