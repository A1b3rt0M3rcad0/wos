ALTER TABLE issues ADD CONSTRAINT issues_objective_owner_fk FOREIGN KEY (namespace_id, outcome_id, objective_id) REFERENCES objectives (namespace_id, outcome_id, id) ON DELETE RESTRICT;
ALTER TABLE issues ALTER COLUMN objective_id SET NOT NULL;
