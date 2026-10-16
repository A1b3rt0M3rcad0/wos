-- Additive immutable signed-execution activity; v1 checkpoints stay intact.
CREATE TABLE work_activities (
 id TEXT PRIMARY KEY,
 namespace_id TEXT NOT NULL,
 outcome_id TEXT NOT NULL,
 contract_id TEXT NOT NULL,
 work_item_id TEXT NOT NULL,
 -- Fixed-width digits give exact unsigned numeric ordering in SQLite indexes.
 sequence NUMERIC(20,0) NOT NULL CHECK (sequence>0 AND sequence<=18446744073709551615),
 payload_json TEXT NOT NULL,
 UNIQUE(namespace_id,outcome_id,sequence),
 FOREIGN KEY(namespace_id,outcome_id,work_item_id,contract_id) REFERENCES signed_work_contracts(namespace_id,outcome_id,work_item_id,id)
);
CREATE INDEX work_activity_by_task ON work_activities(namespace_id,outcome_id,work_item_id,sequence);
CREATE INDEX work_activity_by_contract ON work_activities(namespace_id,outcome_id,contract_id,sequence);
