CREATE TABLE issues (
    id TEXT PRIMARY KEY,
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT 'issue' CHECK (kind = 'issue'),
    version BIGINT NOT NULL CHECK (version >= 1),
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    severity TEXT NOT NULL CHECK (severity IN ('critical','major','minor','informational')),
    lifecycle TEXT NOT NULL CHECK (lifecycle IN ('open','investigating','resolved','wont_fix','duplicate')),
    reported_by_json TEXT NOT NULL,
    resolution_summary TEXT NOT NULL DEFAULT '',
    duplicate_of_issue_id TEXT,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    UNIQUE (namespace_id, outcome_id, id),
    FOREIGN KEY (namespace_id, outcome_id, id, kind)
        REFERENCES entity_refs(namespace_id, outcome_id, id, kind),
    FOREIGN KEY (namespace_id, outcome_id)
        REFERENCES outcomes(namespace_id, id),
    FOREIGN KEY (namespace_id, outcome_id, duplicate_of_issue_id)
        REFERENCES issues(namespace_id, outcome_id, id),
    CHECK (duplicate_of_issue_id IS NULL OR duplicate_of_issue_id <> id)
);

CREATE TABLE issue_affected_refs (
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    issue_id TEXT NOT NULL,
    affected_id TEXT NOT NULL,
    affected_kind TEXT NOT NULL,
    PRIMARY KEY (namespace_id, outcome_id, issue_id, affected_id, affected_kind),
    FOREIGN KEY (namespace_id, outcome_id, issue_id)
        REFERENCES issues(namespace_id, outcome_id, id) ON DELETE CASCADE,
    FOREIGN KEY (namespace_id, outcome_id, affected_id, affected_kind)
        REFERENCES entity_refs(namespace_id, outcome_id, id, kind)
);

CREATE TABLE blockers (
    id TEXT PRIMARY KEY,
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT 'blocker' CHECK (kind = 'blocker'),
    version BIGINT NOT NULL CHECK (version >= 1),
    blocked_id TEXT NOT NULL,
    blocked_kind TEXT NOT NULL CHECK (blocked_kind IN ('outcome','objective','work_item')),
    cause_id TEXT,
    cause_kind TEXT,
    external_cause_json TEXT,
    description TEXT NOT NULL DEFAULT '',
    lifecycle TEXT NOT NULL CHECK (lifecycle IN ('active','resolved','cancelled')),
    propagation TEXT NOT NULL CHECK (propagation IN ('direct','subtree')),
    resolved_at BIGINT,
    resolution_summary TEXT NOT NULL DEFAULT '',
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    UNIQUE (namespace_id, outcome_id, id),
    FOREIGN KEY (namespace_id, outcome_id, id, kind)
        REFERENCES entity_refs(namespace_id, outcome_id, id, kind),
    FOREIGN KEY (namespace_id, outcome_id)
        REFERENCES outcomes(namespace_id, id),
    FOREIGN KEY (namespace_id, outcome_id, blocked_id, blocked_kind)
        REFERENCES entity_refs(namespace_id, outcome_id, id, kind),
    FOREIGN KEY (namespace_id, outcome_id, cause_id, cause_kind)
        REFERENCES entity_refs(namespace_id, outcome_id, id, kind),
    CHECK ((cause_id IS NULL) = (cause_kind IS NULL)),
    CHECK (NOT (cause_id IS NOT NULL AND external_cause_json IS NOT NULL))
);

CREATE INDEX idx_issues_scope
    ON issues (namespace_id, outcome_id, lifecycle, severity, id);

CREATE INDEX idx_issue_affected_ref
    ON issue_affected_refs (namespace_id, outcome_id, affected_id, affected_kind, issue_id);

CREATE INDEX idx_blockers_scope
    ON blockers (namespace_id, outcome_id, lifecycle, blocked_kind, blocked_id, id);

CREATE INDEX idx_blockers_cause
    ON blockers (namespace_id, outcome_id, cause_kind, cause_id);
