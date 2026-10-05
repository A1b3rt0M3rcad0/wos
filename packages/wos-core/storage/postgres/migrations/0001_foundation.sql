CREATE TABLE namespaces (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    lifecycle TEXT NOT NULL CHECK (lifecycle IN ('active','suspended')),
    version BIGINT NOT NULL CHECK (version >= 1),
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL
);

CREATE TABLE principals (
    id TEXT PRIMARY KEY,
    display_name TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('human','service','local')),
    lifecycle TEXT NOT NULL CHECK (lifecycle IN ('active','revoked')),
    created_at BIGINT NOT NULL
);

CREATE TABLE outcome_coordination (
    namespace_id TEXT NOT NULL REFERENCES namespaces(id),
    outcome_id TEXT NOT NULL,
    state_revision BIGINT NOT NULL CHECK (state_revision >= 0),
    PRIMARY KEY (namespace_id, outcome_id)
);

CREATE TABLE entity_refs (
    id TEXT PRIMARY KEY,
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN (
        'outcome','objective','work_item','issue','blocker','artifact',
        'evidence','decision','relation','evidence_link','roadmap','trigger'
    )),
    UNIQUE (namespace_id, outcome_id, id),
    UNIQUE (namespace_id, outcome_id, id, kind),
    FOREIGN KEY (namespace_id, outcome_id)
        REFERENCES outcome_coordination(namespace_id, outcome_id)
);

CREATE TABLE outcomes (
    id TEXT PRIMARY KEY,
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT 'outcome' CHECK (kind = 'outcome'),
    version BIGINT NOT NULL CHECK (version >= 1),
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    created_by_json TEXT NOT NULL DEFAULT '{}',
    metadata_json TEXT NOT NULL DEFAULT '{}',
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    priority TEXT NOT NULL CHECK (priority IN ('critical','high','normal','low')),
    desired_state TEXT NOT NULL,
    lifecycle TEXT NOT NULL CHECK (lifecycle IN ('draft','active','achieved','failed','abandoned')),
    external_context_json TEXT NOT NULL DEFAULT '{}',
    archived_at BIGINT,
    UNIQUE (namespace_id, outcome_id, id),
    UNIQUE (namespace_id, id),
    FOREIGN KEY (namespace_id, outcome_id, id, kind)
        REFERENCES entity_refs(namespace_id, outcome_id, id, kind),
    CHECK (id = outcome_id)
);

CREATE TABLE objectives (
    id TEXT PRIMARY KEY,
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT 'objective' CHECK (kind = 'objective'),
    version BIGINT NOT NULL CHECK (version >= 1),
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    created_by_json TEXT NOT NULL DEFAULT '{}',
    metadata_json TEXT NOT NULL DEFAULT '{}',
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    priority TEXT NOT NULL CHECK (priority IN ('critical','high','normal','low')),
    parent_objective_id TEXT,
    lifecycle TEXT NOT NULL CHECK (lifecycle IN ('planned','in_progress','achieved','cancelled')),
    required_for_outcome INTEGER NOT NULL DEFAULT 0 CHECK (required_for_outcome IN (0,1)),
    due_at BIGINT,
    UNIQUE (namespace_id, outcome_id, id),
    FOREIGN KEY (namespace_id, outcome_id, id, kind)
        REFERENCES entity_refs(namespace_id, outcome_id, id, kind),
    FOREIGN KEY (namespace_id, outcome_id)
        REFERENCES outcomes(namespace_id, id),
    FOREIGN KEY (namespace_id, outcome_id, parent_objective_id)
        REFERENCES objectives(namespace_id, outcome_id, id),
    CHECK (parent_objective_id IS NULL OR parent_objective_id <> id)
);

CREATE TABLE work_items (
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
    objective_id TEXT,
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

CREATE TABLE entity_actor_links (
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('owner','assignee')),
    actor_kind TEXT NOT NULL,
    actor_provider TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    actor_json TEXT NOT NULL,
    PRIMARY KEY (
        namespace_id, outcome_id, entity_id, role,
        actor_kind, actor_provider, actor_id
    ),
    FOREIGN KEY (namespace_id, outcome_id, entity_id)
        REFERENCES entity_refs(namespace_id, outcome_id, id)
);

CREATE TABLE success_criteria (
    id TEXT PRIMARY KEY,
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    owner_id TEXT NOT NULL,
    owner_kind TEXT NOT NULL CHECK (owner_kind IN ('outcome','objective','work_item')),
    criterion_revision BIGINT NOT NULL CHECK (criterion_revision >= 1),
    status TEXT NOT NULL CHECK (status IN ('active','retired')),
    created_at BIGINT NOT NULL,
    UNIQUE (namespace_id, outcome_id, id, owner_id),
    UNIQUE (namespace_id, outcome_id, id),
    FOREIGN KEY (namespace_id, outcome_id, owner_id, owner_kind)
        REFERENCES entity_refs(namespace_id, outcome_id, id, kind),
    FOREIGN KEY (namespace_id, outcome_id)
        REFERENCES outcomes(namespace_id, id)
);

CREATE TABLE criterion_revisions (
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    criterion_id TEXT NOT NULL,
    criterion_revision BIGINT NOT NULL,
    definition_json TEXT NOT NULL,
    required INTEGER NOT NULL CHECK (required IN (0,1)),
    verification_mode TEXT NOT NULL CHECK (
        verification_mode IN ('attestation','evidence_review','external_evaluation')
    ),
    created_at BIGINT NOT NULL,
    created_by_json TEXT NOT NULL DEFAULT '{}',
    PRIMARY KEY (namespace_id, outcome_id, criterion_id, criterion_revision),
    FOREIGN KEY (namespace_id, outcome_id, criterion_id)
        REFERENCES success_criteria(namespace_id, outcome_id, id)
);

CREATE TABLE criterion_assessments (
    id TEXT PRIMARY KEY,
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    criterion_id TEXT NOT NULL,
    criterion_revision BIGINT NOT NULL,
    result TEXT NOT NULL CHECK (result IN ('met','not_met','inconclusive','waived')),
    rationale TEXT NOT NULL,
    principal_id TEXT NOT NULL REFERENCES principals(id),
    actor_json TEXT NOT NULL,
    assessed_at BIGINT NOT NULL,
    evaluator_ref_json TEXT,
    supersedes_assessment_id TEXT,
    UNIQUE (namespace_id, outcome_id, id, criterion_id),
    UNIQUE (namespace_id, outcome_id, id),
    FOREIGN KEY (namespace_id, outcome_id, criterion_id, criterion_revision)
        REFERENCES criterion_revisions(namespace_id, outcome_id, criterion_id, criterion_revision),
    FOREIGN KEY (namespace_id, outcome_id, supersedes_assessment_id)
        REFERENCES criterion_assessments(namespace_id, outcome_id, id),
    FOREIGN KEY (namespace_id, outcome_id)
        REFERENCES outcomes(namespace_id, id)
);

CREATE TABLE criterion_current_assessments (
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    criterion_id TEXT NOT NULL,
    assessment_id TEXT NOT NULL,
    PRIMARY KEY (namespace_id, outcome_id, criterion_id),
    FOREIGN KEY (namespace_id, outcome_id, criterion_id)
        REFERENCES success_criteria(namespace_id, outcome_id, id),
    FOREIGN KEY (namespace_id, outcome_id, assessment_id, criterion_id)
        REFERENCES criterion_assessments(namespace_id, outcome_id, id, criterion_id)
);

CREATE TABLE conclusions (
    id TEXT PRIMARY KEY,
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    owner_id TEXT NOT NULL,
    ordinal BIGINT NOT NULL CHECK (ordinal >= 0),
    owner_version BIGINT,
    lifecycle_result TEXT NOT NULL,
    principal_id TEXT NOT NULL REFERENCES principals(id),
    actor_json TEXT NOT NULL,
    recorded_at BIGINT NOT NULL,
    rationale TEXT NOT NULL,
    obligations_snapshot_json TEXT NOT NULL DEFAULT '{}',
    UNIQUE (namespace_id, outcome_id, owner_id, ordinal),
    UNIQUE (namespace_id, outcome_id, id, owner_id),
    UNIQUE (namespace_id, outcome_id, id),
    FOREIGN KEY (namespace_id, outcome_id, owner_id)
        REFERENCES entity_refs(namespace_id, outcome_id, id),
    FOREIGN KEY (namespace_id, outcome_id)
        REFERENCES outcomes(namespace_id, id)
);

CREATE TABLE conclusion_assessments (
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    conclusion_id TEXT NOT NULL,
    assessment_id TEXT NOT NULL,
    PRIMARY KEY (namespace_id, outcome_id, conclusion_id, assessment_id),
    FOREIGN KEY (namespace_id, outcome_id, conclusion_id)
        REFERENCES conclusions(namespace_id, outcome_id, id),
    FOREIGN KEY (namespace_id, outcome_id, assessment_id)
        REFERENCES criterion_assessments(namespace_id, outcome_id, id)
);

CREATE TABLE aggregate_current_conclusions (
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    owner_id TEXT NOT NULL,
    conclusion_id TEXT NOT NULL,
    PRIMARY KEY (namespace_id, outcome_id, owner_id),
    FOREIGN KEY (namespace_id, outcome_id, owner_id)
        REFERENCES entity_refs(namespace_id, outcome_id, id),
    FOREIGN KEY (namespace_id, outcome_id, conclusion_id, owner_id)
        REFERENCES conclusions(namespace_id, outcome_id, id, owner_id)
);

CREATE TABLE domain_events (
    id TEXT PRIMARY KEY,
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    schema_version BIGINT NOT NULL CHECK (schema_version >= 1),
    outcome_revision BIGINT NOT NULL CHECK (outcome_revision >= 1),
    event_index BIGINT NOT NULL CHECK (event_index >= 0),
    aggregate_id TEXT NOT NULL,
    aggregate_kind TEXT NOT NULL,
    aggregate_version_before BIGINT,
    aggregate_version_after BIGINT,
    command_id TEXT NOT NULL,
    principal_id TEXT NOT NULL REFERENCES principals(id),
    actor_json TEXT NOT NULL,
    recorded_at BIGINT NOT NULL,
    correlation_id TEXT,
    causation_id TEXT,
    execution_context_json TEXT NOT NULL,
    payload_json TEXT NOT NULL,
    UNIQUE (namespace_id, outcome_id, outcome_revision, event_index),
    UNIQUE (namespace_id, outcome_id, id),
    FOREIGN KEY (namespace_id, outcome_id, aggregate_id, aggregate_kind)
        REFERENCES entity_refs(namespace_id, outcome_id, id, kind),
    FOREIGN KEY (namespace_id, outcome_id)
        REFERENCES outcomes(namespace_id, id)
);

CREATE TABLE idempotency_records (
    namespace_id TEXT NOT NULL REFERENCES namespaces(id),
    principal_id TEXT NOT NULL REFERENCES principals(id),
    command_name TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('processing','completed')),
    command_id TEXT,
    outcome_revision BIGINT,
    response_json TEXT,
    created_at BIGINT NOT NULL,
    expires_at BIGINT NOT NULL,
    PRIMARY KEY (namespace_id, principal_id, command_name, idempotency_key),
    CHECK (
        status <> 'completed'
        OR (command_id IS NOT NULL AND outcome_revision IS NOT NULL AND response_json IS NOT NULL)
    )
);

CREATE INDEX idx_objectives_scope
    ON objectives (namespace_id, outcome_id, lifecycle, created_at, id);

CREATE INDEX idx_objectives_parent
    ON objectives (namespace_id, outcome_id, parent_objective_id);

CREATE INDEX idx_work_scope
    ON work_items (namespace_id, outcome_id, lifecycle, priority, id);

CREATE INDEX idx_work_objective
    ON work_items (namespace_id, outcome_id, objective_id);

CREATE INDEX idx_criteria_owner
    ON success_criteria (namespace_id, outcome_id, owner_id, status);

CREATE INDEX idx_event_aggregate
    ON domain_events (
        namespace_id, outcome_id, aggregate_id, outcome_revision, event_index
    );

CREATE INDEX idx_idempotency_expiry
    ON idempotency_records (expires_at);
