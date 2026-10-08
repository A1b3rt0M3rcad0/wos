-- Additive: legacy scope remains legacy until explicit drain/cutover.
ALTER TABLE work_items ADD COLUMN contracts_enabled INTEGER NOT NULL DEFAULT 0 CHECK (contracts_enabled IN (0,1));
ALTER TABLE work_items ADD COLUMN current_contract_id TEXT;
ALTER TABLE work_items ADD COLUMN execution_spec_json TEXT;
ALTER TABLE work_items ADD COLUMN fencing_token_decimal TEXT;
CREATE TABLE work_contracts (
 id TEXT PRIMARY KEY,
 namespace_id TEXT NOT NULL,
 outcome_id TEXT NOT NULL,
 work_item_id TEXT NOT NULL,
 holder_principal_id TEXT NOT NULL REFERENCES principals(id),
 status TEXT NOT NULL CHECK (status IN ('active','expired','revoked','completed')),
 version BIGINT NOT NULL CHECK (version>=1),
 lease_version BIGINT NOT NULL CHECK (lease_version>=1),
 fencing_token TEXT NOT NULL,
 expires_at BIGINT NOT NULL,
 state_json TEXT NOT NULL,
 spec_json TEXT NOT NULL,
 spec_digest TEXT NOT NULL,
 UNIQUE(namespace_id,outcome_id,id),
 UNIQUE(namespace_id,outcome_id,work_item_id,id),
 FOREIGN KEY(namespace_id,outcome_id,work_item_id) REFERENCES work_items(namespace_id,outcome_id,id)
);
CREATE UNIQUE INDEX work_contracts_one_open ON work_contracts(namespace_id,outcome_id,work_item_id) WHERE status='active';
CREATE INDEX work_contracts_holder ON work_contracts(namespace_id,outcome_id,holder_principal_id,id);
CREATE INDEX work_contracts_work_history ON work_contracts(namespace_id,outcome_id,work_item_id,id);
CREATE INDEX work_contracts_expiry ON work_contracts(status,expires_at,id);
CREATE TABLE work_contract_checkpoints (
 id TEXT PRIMARY KEY,
 namespace_id TEXT NOT NULL,
 outcome_id TEXT NOT NULL,
 contract_id TEXT NOT NULL,
 work_item_id TEXT NOT NULL,
 sequence BIGINT NOT NULL CHECK (sequence>=1),
 payload_json TEXT NOT NULL,
 UNIQUE(namespace_id,outcome_id,contract_id,sequence),
 FOREIGN KEY(namespace_id,outcome_id,work_item_id,contract_id) REFERENCES work_contracts(namespace_id,outcome_id,work_item_id,id)
);
CREATE INDEX work_contract_checkpoint_page ON work_contract_checkpoints(namespace_id,outcome_id,contract_id,id);
CREATE TABLE work_contract_submissions (
 id TEXT PRIMARY KEY,
 namespace_id TEXT NOT NULL,
 outcome_id TEXT NOT NULL,
 contract_id TEXT NOT NULL,
 work_item_id TEXT NOT NULL,
 submission_digest TEXT NOT NULL,
 payload_json TEXT NOT NULL,
 UNIQUE(namespace_id,outcome_id,id),
 FOREIGN KEY(namespace_id,outcome_id,work_item_id,contract_id) REFERENCES work_contracts(namespace_id,outcome_id,work_item_id,id)
);
CREATE INDEX work_contract_submission_page ON work_contract_submissions(namespace_id,outcome_id,contract_id,id);
