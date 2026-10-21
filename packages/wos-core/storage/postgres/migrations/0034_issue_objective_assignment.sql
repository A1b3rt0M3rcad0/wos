ALTER TABLE issues ADD COLUMN objective_id TEXT;
ALTER TABLE issues ADD COLUMN assignee_ref_json TEXT;

CREATE INDEX idx_issues_scope_objective_lifecycle
    ON issues (namespace_id, outcome_id, objective_id, lifecycle, id);
