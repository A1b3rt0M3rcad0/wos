-- Contract stage: rebuild every Task with an explicit Objective owner.
-- The migration runner checks for orphans first and this INSERT intentionally
-- copies every row so a concurrent or unexpected orphan cannot be discarded.
PRAGMA defer_foreign_keys = ON;

CREATE TABLE work_items_required_objective (
    id TEXT PRIMARY KEY,
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT 'work_item' CHECK (kind = 'work_item'),
    version BIGINT NOT NULL CHECK (version >= 1),
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    created_by_json TEXT NOT NULL DEFAULT '{}',
    metadata_json TEXT NOT NULL DEFAULT '{}',
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    priority TEXT NOT NULL CHECK (priority IN ('critical','high','normal','low')),
    objective_id TEXT NOT NULL,
    lifecycle TEXT NOT NULL CHECK (lifecycle IN ('backlog','todo','in_progress','done','cancelled')),
    due_at BIGINT,
    not_before BIGINT,
    result_summary TEXT NOT NULL DEFAULT '',
    last_fencing_token BIGINT NOT NULL DEFAULT 0 CHECK (last_fencing_token >= 0),
    lease_claim_id TEXT,
    lease_principal_id TEXT REFERENCES principals(id),
    lease_actor_json TEXT,
    lease_acquired_at BIGINT,
    lease_expires_at BIGINT,
    contracts_enabled INTEGER NOT NULL DEFAULT 0 CHECK (contracts_enabled IN (0,1)),
    current_contract_id TEXT,
    execution_spec_json TEXT,
    fencing_token_decimal TEXT,
    pending_review_case_id TEXT,
    latest_review_case_id TEXT,
    correction_review_case_id TEXT,
    UNIQUE (namespace_id, outcome_id, id),
    FOREIGN KEY (namespace_id, outcome_id, id, kind)
        REFERENCES entity_refs(namespace_id, outcome_id, id, kind),
    FOREIGN KEY (namespace_id, outcome_id)
        REFERENCES outcomes(namespace_id, id),
    FOREIGN KEY (namespace_id, outcome_id, objective_id)
        REFERENCES objectives(namespace_id, outcome_id, id),
    CHECK (
        (lease_claim_id IS NULL AND lease_principal_id IS NULL AND
         lease_actor_json IS NULL AND lease_acquired_at IS NULL AND lease_expires_at IS NULL)
        OR
        (lease_claim_id IS NOT NULL AND lease_principal_id IS NOT NULL AND
         lease_actor_json IS NOT NULL AND lease_acquired_at IS NOT NULL AND
         lease_expires_at IS NOT NULL AND lease_expires_at > lease_acquired_at)
    )
);

INSERT INTO work_items_required_objective (
    id, namespace_id, outcome_id, kind, version, created_at, updated_at,
    created_by_json, metadata_json, title, description, priority, objective_id,
    lifecycle, due_at, not_before, result_summary, last_fencing_token,
    lease_claim_id, lease_principal_id, lease_actor_json, lease_acquired_at,
    lease_expires_at, contracts_enabled, current_contract_id,
    execution_spec_json, fencing_token_decimal, pending_review_case_id,
    latest_review_case_id, correction_review_case_id
)
SELECT
    id, namespace_id, outcome_id, kind, version, created_at, updated_at,
    created_by_json, metadata_json, title, description, priority, objective_id,
    lifecycle, due_at, not_before, result_summary, last_fencing_token,
    lease_claim_id, lease_principal_id, lease_actor_json, lease_acquired_at,
    lease_expires_at, contracts_enabled, current_contract_id,
    execution_spec_json, fencing_token_decimal, pending_review_case_id,
    latest_review_case_id, correction_review_case_id
FROM work_items;

DROP TABLE work_items;
ALTER TABLE work_items_required_objective RENAME TO work_items;
