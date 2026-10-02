CREATE TABLE relations (
    id TEXT PRIMARY KEY,
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT 'relation' CHECK (kind = 'relation'),
    version BIGINT NOT NULL CHECK (version >= 1),
    source_id TEXT NOT NULL,
    source_kind TEXT NOT NULL CHECK (source_kind IN ('objective','work_item')),
    relation_type TEXT NOT NULL CHECK (relation_type IN ('depends_on','relates_to','produces','derived_from')),
    target_id TEXT NOT NULL,
    target_kind TEXT NOT NULL,
    strength TEXT CHECK (strength IS NULL OR strength IN ('hard','advisory')),
    satisfaction TEXT CHECK (satisfaction IS NULL OR satisfaction = 'target_completed'),
    lifecycle TEXT NOT NULL CHECK (lifecycle IN ('active','removed')),
    removal_reason TEXT NOT NULL DEFAULT '',
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    UNIQUE (namespace_id, outcome_id, id),
    FOREIGN KEY (namespace_id, outcome_id, id, kind)
        REFERENCES entity_refs(namespace_id, outcome_id, id, kind),
    FOREIGN KEY (namespace_id, outcome_id, source_id, source_kind)
        REFERENCES entity_refs(namespace_id, outcome_id, id, kind),
    FOREIGN KEY (namespace_id, outcome_id, target_id, target_kind)
        REFERENCES entity_refs(namespace_id, outcome_id, id, kind),
    CHECK (source_id <> target_id OR source_kind <> target_kind),
    CHECK (
        relation_type <> 'depends_on'
        OR (
            source_kind IN ('objective','work_item')
            AND target_kind IN ('objective','work_item')
            AND strength IS NOT NULL
            AND satisfaction = 'target_completed'
        )
    )
);

CREATE UNIQUE INDEX uq_relations_active_semantic
    ON relations (
        namespace_id, outcome_id,
        source_id, source_kind, relation_type,
        target_id, target_kind
    )
    WHERE lifecycle = 'active';

CREATE INDEX idx_relations_source
    ON relations (namespace_id, outcome_id, source_id, source_kind, lifecycle);

CREATE INDEX idx_relations_target
    ON relations (namespace_id, outcome_id, target_id, target_kind, lifecycle);
