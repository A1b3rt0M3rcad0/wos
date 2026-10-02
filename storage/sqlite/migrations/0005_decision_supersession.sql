CREATE UNIQUE INDEX uq_decisions_direct_successor
    ON decisions (namespace_id, outcome_id, supersedes_decision_id)
    WHERE supersedes_decision_id IS NOT NULL;
