-- Additive signed-v2 execution/review storage; v1 tables and histories stay intact.
CREATE TABLE signed_work_contracts (
 id TEXT PRIMARY KEY,
 namespace_id TEXT NOT NULL,
 outcome_id TEXT NOT NULL,
 work_item_id TEXT NOT NULL,
 credential_id TEXT NOT NULL,
 holder_principal_id TEXT NOT NULL REFERENCES principals(id),
 status TEXT NOT NULL CHECK (status IN ('active','expired','revoked','completed','delivered')),
 version TEXT NOT NULL CHECK (version!='0'),
 lease_version TEXT NOT NULL CHECK (lease_version!='0'),
 fencing_token TEXT NOT NULL,
 expires_at BIGINT NOT NULL,
 state_json TEXT NOT NULL,
 spec_json TEXT NOT NULL,
 spec_digest TEXT NOT NULL,
 UNIQUE(namespace_id,outcome_id,id),
 UNIQUE(namespace_id,outcome_id,work_item_id,id),
 FOREIGN KEY(namespace_id,outcome_id,work_item_id) REFERENCES work_items(namespace_id,outcome_id,id)
);
CREATE UNIQUE INDEX signed_work_contracts_one_open ON signed_work_contracts(namespace_id,outcome_id,work_item_id) WHERE status='active';
CREATE INDEX signed_work_contracts_holder ON signed_work_contracts(namespace_id,outcome_id,holder_principal_id,id);
CREATE INDEX signed_work_contracts_work_history ON signed_work_contracts(namespace_id,outcome_id,work_item_id,id);
CREATE INDEX signed_work_contracts_expiry ON signed_work_contracts(status,expires_at,id);
CREATE TABLE signed_work_contract_checkpoints (
 id TEXT PRIMARY KEY,
 namespace_id TEXT NOT NULL,
 outcome_id TEXT NOT NULL,
 contract_id TEXT NOT NULL,
 work_item_id TEXT NOT NULL,
 sequence BIGINT NOT NULL CHECK (sequence>=1),
 payload_json TEXT NOT NULL,
 UNIQUE(namespace_id,outcome_id,contract_id,sequence),
 FOREIGN KEY(namespace_id,outcome_id,work_item_id,contract_id) REFERENCES signed_work_contracts(namespace_id,outcome_id,work_item_id,id)
);
CREATE INDEX signed_work_contract_checkpoint_page ON signed_work_contract_checkpoints(namespace_id,outcome_id,contract_id,id);
CREATE TABLE signed_work_contract_submissions (
 id TEXT PRIMARY KEY,
 namespace_id TEXT NOT NULL,
 outcome_id TEXT NOT NULL,
 contract_id TEXT NOT NULL,
 work_item_id TEXT NOT NULL,
 submission_digest TEXT NOT NULL,
 payload_json TEXT NOT NULL,
 UNIQUE(namespace_id,outcome_id,id),
 FOREIGN KEY(namespace_id,outcome_id,work_item_id,contract_id) REFERENCES signed_work_contracts(namespace_id,outcome_id,work_item_id,id)
);
CREATE INDEX signed_work_contract_submission_page ON signed_work_contract_submissions(namespace_id,outcome_id,contract_id,id);

CREATE INDEX signed_work_contracts_credential ON signed_work_contracts(namespace_id,credential_id,status,expires_at);
CREATE INDEX signed_work_contracts_quota ON signed_work_contracts(namespace_id,holder_principal_id,status,expires_at);
CREATE TABLE work_review_cases (
 id TEXT PRIMARY KEY, namespace_id TEXT NOT NULL, outcome_id TEXT NOT NULL,
 work_item_id TEXT NOT NULL, work_contract_id TEXT NOT NULL, submission_id TEXT NOT NULL,
 status TEXT NOT NULL CHECK(status IN ('pending','in_review','approved','changes_requested','cancelled','superseded')),
 version TEXT NOT NULL CHECK(version!='0'), state_json TEXT NOT NULL,
 UNIQUE(namespace_id,outcome_id,id),
 FOREIGN KEY(namespace_id,outcome_id,work_item_id,work_contract_id) REFERENCES signed_work_contracts(namespace_id,outcome_id,work_item_id,id),
 FOREIGN KEY(namespace_id,outcome_id,submission_id) REFERENCES signed_work_contract_submissions(namespace_id,outcome_id,id)
);
CREATE UNIQUE INDEX work_review_cases_one_open ON work_review_cases(namespace_id,outcome_id,work_item_id) WHERE status IN ('pending','in_review');
CREATE INDEX work_review_cases_queue ON work_review_cases(namespace_id,outcome_id,status,id);
CREATE TABLE work_review_contracts (
 id TEXT PRIMARY KEY, namespace_id TEXT NOT NULL, outcome_id TEXT NOT NULL,
 case_id TEXT NOT NULL, work_item_id TEXT NOT NULL, holder_principal_id TEXT NOT NULL REFERENCES principals(id), credential_id TEXT NOT NULL,
 status TEXT NOT NULL CHECK(status IN ('active','expired','revoked','completed')),
 version TEXT NOT NULL CHECK(version!='0'), lease_version TEXT NOT NULL CHECK(lease_version!='0'),
 expires_at BIGINT NOT NULL, state_json TEXT NOT NULL,
 UNIQUE(namespace_id,outcome_id,id),
 FOREIGN KEY(namespace_id,outcome_id,case_id) REFERENCES work_review_cases(namespace_id,outcome_id,id)
);
CREATE UNIQUE INDEX work_review_contracts_one_open ON work_review_contracts(namespace_id,outcome_id,case_id) WHERE status='active';
CREATE INDEX work_review_contracts_quota ON work_review_contracts(namespace_id,holder_principal_id,status,expires_at);
CREATE INDEX work_review_contracts_credential ON work_review_contracts(namespace_id,credential_id,status,expires_at);
CREATE TABLE signed_protocol_facts (
 id TEXT PRIMARY KEY, namespace_id TEXT NOT NULL, outcome_id TEXT NOT NULL,
 contract_id TEXT NOT NULL, contract_kind TEXT NOT NULL CHECK(contract_kind IN ('execution','review')),
 kind TEXT NOT NULL CHECK(kind IN ('specification','authority','return','acceptance')),
 principal_id TEXT NOT NULL REFERENCES principals(id), key_id TEXT NOT NULL,
 idempotency_key TEXT NOT NULL, request_id TEXT NOT NULL, payload_digest TEXT NOT NULL,
 state_json TEXT NOT NULL, UNIQUE(namespace_id,outcome_id,id),
 FOREIGN KEY(namespace_id,outcome_id) REFERENCES outcomes(namespace_id,id),
 FOREIGN KEY(namespace_id,key_id) REFERENCES signing_keys(namespace_id,id)
);
CREATE INDEX signed_protocol_facts_contract ON signed_protocol_facts(namespace_id,outcome_id,contract_id,kind,id);
CREATE UNIQUE INDEX signed_acceptance_operation ON signed_protocol_facts(namespace_id,principal_id,idempotency_key) WHERE kind='acceptance';
CREATE UNIQUE INDEX signed_return_operation ON signed_protocol_facts(namespace_id,principal_id,idempotency_key) WHERE kind='return';
CREATE UNIQUE INDEX signed_return_request ON signed_protocol_facts(namespace_id,request_id) WHERE kind='return';
CREATE UNIQUE INDEX signed_contract_specification ON signed_protocol_facts(namespace_id,outcome_id,contract_id) WHERE kind='specification';
