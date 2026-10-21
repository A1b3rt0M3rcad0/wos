-- Contract stage: every Issue must have one explicit Objective owner.
-- The runner checks for orphans first and this INSERT copies every Issue so
-- unexpected ownership loss fails instead of silently dropping records.
PRAGMA defer_foreign_keys = ON;

-- Dropping the old parent would cascade-delete these rows from
-- issue_affected_refs. Preserve them explicitly across the SQLite table rebuild.
CREATE TEMP TABLE issue_affected_refs_migration_36 AS
SELECT namespace_id, outcome_id, issue_id, affected_id, affected_kind
FROM issue_affected_refs;

CREATE TABLE issues_required_objective (
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
    objective_id TEXT NOT NULL,
    assignee_ref_json TEXT,
    UNIQUE (namespace_id, outcome_id, id),
    FOREIGN KEY (namespace_id, outcome_id, id, kind)
        REFERENCES entity_refs(namespace_id, outcome_id, id, kind),
    FOREIGN KEY (namespace_id, outcome_id)
        REFERENCES outcomes(namespace_id, id),
    FOREIGN KEY (namespace_id, outcome_id, objective_id)
        REFERENCES objectives(namespace_id, outcome_id, id) ON DELETE RESTRICT,
    FOREIGN KEY (namespace_id, outcome_id, duplicate_of_issue_id)
        REFERENCES issues(namespace_id, outcome_id, id),
    CHECK (duplicate_of_issue_id IS NULL OR duplicate_of_issue_id <> id)
);

INSERT INTO issues_required_objective (
    id, namespace_id, outcome_id, kind, version, title, description, severity,
    lifecycle, reported_by_json, resolution_summary, duplicate_of_issue_id,
    created_at, updated_at, objective_id, assignee_ref_json
)
SELECT
    id, namespace_id, outcome_id, kind, version, title, description, severity,
    lifecycle, reported_by_json, resolution_summary, duplicate_of_issue_id,
    created_at, updated_at, objective_id, assignee_ref_json
FROM issues;

DROP TABLE issues;
ALTER TABLE issues_required_objective RENAME TO issues;

INSERT INTO issue_affected_refs (
    namespace_id, outcome_id, issue_id, affected_id, affected_kind
)
SELECT namespace_id, outcome_id, issue_id, affected_id, affected_kind
FROM issue_affected_refs_migration_36;

DROP TABLE issue_affected_refs_migration_36;
