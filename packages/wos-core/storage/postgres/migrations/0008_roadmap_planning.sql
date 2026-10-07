CREATE TABLE roadmaps (
    id TEXT PRIMARY KEY,
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT 'roadmap' CHECK (kind = 'roadmap'),
    version BIGINT NOT NULL CHECK (version >= 1),
    plan_scope_kind TEXT NOT NULL CHECK (plan_scope_kind IN ('outcome','objective')),
    plan_scope_id TEXT NOT NULL,
    title TEXT NOT NULL,
    lifecycle TEXT NOT NULL CHECK (lifecycle IN ('open','archived')),
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    archived_at BIGINT,
    UNIQUE (namespace_id, outcome_id, id),
    FOREIGN KEY (namespace_id, outcome_id, id, kind)
        REFERENCES entity_refs(namespace_id, outcome_id, id, kind),
    FOREIGN KEY (namespace_id, outcome_id)
        REFERENCES outcomes(namespace_id, id)
);

CREATE INDEX idx_roadmaps_scope
ON roadmaps(namespace_id, outcome_id, plan_scope_kind, plan_scope_id);

CREATE TABLE roadmap_drafts (
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    roadmap_id TEXT NOT NULL,
    draft_version BIGINT NOT NULL CHECK (draft_version >= 1),
    base_revision_number BIGINT,
    lifecycle TEXT NOT NULL CHECK (lifecycle IN ('open','discarded')),
    nodes_json TEXT NOT NULL,
    after_links_json TEXT NOT NULL,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    discarded_at BIGINT,
    PRIMARY KEY (namespace_id, outcome_id, roadmap_id),
    FOREIGN KEY (namespace_id, outcome_id, roadmap_id)
        REFERENCES roadmaps(namespace_id, outcome_id, id)
        ON DELETE CASCADE
);

CREATE TABLE roadmap_revisions (
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    roadmap_id TEXT NOT NULL,
    revision_number BIGINT NOT NULL CHECK (revision_number >= 1),
    content_hash TEXT NOT NULL,
    published_by_json TEXT NOT NULL,
    published_at BIGINT NOT NULL,
    PRIMARY KEY (namespace_id, outcome_id, roadmap_id, revision_number),
    FOREIGN KEY (namespace_id, outcome_id, roadmap_id)
        REFERENCES roadmaps(namespace_id, outcome_id, id)
        ON DELETE CASCADE
);

CREATE TABLE roadmap_revision_nodes (
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    roadmap_id TEXT NOT NULL,
    revision_number BIGINT NOT NULL,
    node_key TEXT NOT NULL,
    node_type TEXT NOT NULL CHECK (node_type IN ('reference','phase','milestone')),
    parent_node_key TEXT,
    target_ref_json TEXT,
    title TEXT NOT NULL,
    position INTEGER NOT NULL CHECK (position >= 0),
    criterion_refs_json TEXT NOT NULL,
    criterion_snapshots_json TEXT NOT NULL,
    planned_start BIGINT,
    planned_end BIGINT,
    reference_snapshot_json TEXT,
    PRIMARY KEY (namespace_id, outcome_id, roadmap_id, revision_number, node_key),
    FOREIGN KEY (namespace_id, outcome_id, roadmap_id, revision_number)
        REFERENCES roadmap_revisions(namespace_id, outcome_id, roadmap_id, revision_number)
        ON DELETE CASCADE
);

CREATE TABLE roadmap_revision_after_links (
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    roadmap_id TEXT NOT NULL,
    revision_number BIGINT NOT NULL,
    node_key TEXT NOT NULL,
    after_node_key TEXT NOT NULL,
    PRIMARY KEY (
        namespace_id, outcome_id, roadmap_id, revision_number,
        node_key, after_node_key
    ),
    FOREIGN KEY (namespace_id, outcome_id, roadmap_id, revision_number)
        REFERENCES roadmap_revisions(namespace_id, outcome_id, roadmap_id, revision_number)
        ON DELETE CASCADE
);

CREATE TABLE roadmap_revision_dependencies (
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    roadmap_id TEXT NOT NULL,
    revision_number BIGINT NOT NULL,
    dependent_ref_json TEXT NOT NULL,
    prerequisite_ref_json TEXT NOT NULL,
    strength TEXT NOT NULL CHECK (strength IN ('hard','advisory')),
    satisfaction TEXT NOT NULL CHECK (satisfaction = 'target_completed'),
    PRIMARY KEY (
        namespace_id, outcome_id, roadmap_id, revision_number,
        dependent_ref_json, prerequisite_ref_json
    ),
    FOREIGN KEY (namespace_id, outcome_id, roadmap_id, revision_number)
        REFERENCES roadmap_revisions(namespace_id, outcome_id, roadmap_id, revision_number)
        ON DELETE CASCADE
);

CREATE TABLE roadmap_active_slots (
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    plan_scope_kind TEXT NOT NULL CHECK (plan_scope_kind IN ('outcome','objective')),
    plan_scope_id TEXT NOT NULL,
    roadmap_id TEXT NOT NULL,
    revision_number BIGINT NOT NULL CHECK (revision_number >= 1),
    activated_by_json TEXT NOT NULL,
    activated_at BIGINT NOT NULL,
    PRIMARY KEY (namespace_id, outcome_id, plan_scope_kind, plan_scope_id),
    FOREIGN KEY (namespace_id, outcome_id, roadmap_id, revision_number)
        REFERENCES roadmap_revisions(namespace_id, outcome_id, roadmap_id, revision_number)
);

CREATE TABLE roadmap_activation_history (
    id TEXT PRIMARY KEY,
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    plan_scope_kind TEXT NOT NULL CHECK (plan_scope_kind IN ('outcome','objective')),
    plan_scope_id TEXT NOT NULL,
    action TEXT NOT NULL CHECK (action IN ('activated','superseded','deactivated')),
    roadmap_id TEXT NOT NULL,
    revision_number BIGINT NOT NULL CHECK (revision_number >= 1),
    replacement_roadmap_id TEXT,
    replacement_revision_number BIGINT,
    actor_json TEXT NOT NULL,
    recorded_at BIGINT NOT NULL,
    FOREIGN KEY (namespace_id, outcome_id, roadmap_id, revision_number)
        REFERENCES roadmap_revisions(namespace_id, outcome_id, roadmap_id, revision_number)
);

CREATE INDEX idx_roadmap_activation_history_scope
ON roadmap_activation_history(
    namespace_id, outcome_id, plan_scope_kind, plan_scope_id, recorded_at, id
);
