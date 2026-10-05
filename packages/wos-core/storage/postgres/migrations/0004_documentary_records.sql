CREATE TABLE artifacts (
    id TEXT PRIMARY KEY,
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT 'artifact' CHECK (kind = 'artifact'),
    version BIGINT NOT NULL CHECK (version >= 1),
    artifact_type TEXT NOT NULL,
    name TEXT NOT NULL,
    uri TEXT NOT NULL,
    media_type TEXT NOT NULL DEFAULT '',
    checksum TEXT NOT NULL DEFAULT '',
    source_version TEXT NOT NULL DEFAULT '',
    producer_ref_json TEXT NOT NULL,
    produced_at BIGINT,
    registered_at BIGINT NOT NULL,
    lifecycle TEXT NOT NULL CHECK (lifecycle IN ('registered','withdrawn')),
    withdrawal_reason TEXT NOT NULL DEFAULT '',
    withdrawn_at BIGINT,
    updated_at BIGINT NOT NULL,
    UNIQUE (namespace_id, outcome_id, id),
    FOREIGN KEY (namespace_id, outcome_id, id, kind)
        REFERENCES entity_refs(namespace_id, outcome_id, id, kind),
    FOREIGN KEY (namespace_id, outcome_id)
        REFERENCES outcomes(namespace_id, id)
);

CREATE TABLE evidence (
    id TEXT PRIMARY KEY,
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT 'evidence' CHECK (kind = 'evidence'),
    version BIGINT NOT NULL CHECK (version >= 1),
    evidence_type TEXT NOT NULL CHECK (evidence_type IN (
        'measurement','test_result','inspection','attestation','source','external_evaluation'
    )),
    description TEXT NOT NULL,
    source_ref_json TEXT NOT NULL,
    producer_ref_json TEXT NOT NULL,
    captured_at BIGINT NOT NULL,
    registered_at BIGINT NOT NULL,
    artifact_id TEXT,
    measurement_json TEXT,
    source_version TEXT NOT NULL DEFAULT '',
    checksum TEXT NOT NULL DEFAULT '',
    lifecycle TEXT NOT NULL CHECK (lifecycle IN ('registered','retracted')),
    retraction_reason TEXT NOT NULL DEFAULT '',
    retracted_at BIGINT,
    updated_at BIGINT NOT NULL,
    UNIQUE (namespace_id, outcome_id, id),
    FOREIGN KEY (namespace_id, outcome_id, id, kind)
        REFERENCES entity_refs(namespace_id, outcome_id, id, kind),
    FOREIGN KEY (namespace_id, outcome_id)
        REFERENCES outcomes(namespace_id, id),
    FOREIGN KEY (namespace_id, outcome_id, artifact_id)
        REFERENCES artifacts(namespace_id, outcome_id, id)
);

CREATE TABLE decisions (
    id TEXT PRIMARY KEY,
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT 'decision' CHECK (kind = 'decision'),
    version BIGINT NOT NULL CHECK (version >= 1),
    title TEXT NOT NULL,
    proposal TEXT NOT NULL,
    chosen_alternative TEXT NOT NULL DEFAULT '',
    rationale TEXT NOT NULL DEFAULT '',
    alternatives_json TEXT NOT NULL DEFAULT '[]',
    lifecycle TEXT NOT NULL CHECK (lifecycle IN ('proposed','accepted','rejected','superseded')),
    proposed_by_json TEXT NOT NULL,
    decided_by_json TEXT,
    decided_at BIGINT,
    rejection_reason TEXT NOT NULL DEFAULT '',
    supersedes_decision_id TEXT,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    UNIQUE (namespace_id, outcome_id, id),
    FOREIGN KEY (namespace_id, outcome_id, id, kind)
        REFERENCES entity_refs(namespace_id, outcome_id, id, kind),
    FOREIGN KEY (namespace_id, outcome_id)
        REFERENCES outcomes(namespace_id, id),
    FOREIGN KEY (namespace_id, outcome_id, supersedes_decision_id)
        REFERENCES decisions(namespace_id, outcome_id, id),
    CHECK (supersedes_decision_id IS NULL OR supersedes_decision_id <> id)
);

CREATE UNIQUE INDEX idx_decision_direct_successor
    ON decisions (namespace_id, outcome_id, supersedes_decision_id)
    WHERE supersedes_decision_id IS NOT NULL;

CREATE TABLE evidence_links (
    id TEXT PRIMARY KEY,
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT 'evidence_link' CHECK (kind = 'evidence_link'),
    version BIGINT NOT NULL CHECK (version >= 1),
    evidence_id TEXT NOT NULL,
    target_id TEXT NOT NULL,
    target_kind TEXT NOT NULL CHECK (target_kind IN ('outcome','objective','work_item','issue','decision')),
    criterion_id TEXT,
    stance TEXT NOT NULL CHECK (stance IN ('supports','contradicts','context')),
    rationale TEXT NOT NULL,
    lifecycle TEXT NOT NULL CHECK (lifecycle IN ('active','retracted')),
    retraction_reason TEXT NOT NULL DEFAULT '',
    retracted_at BIGINT,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    UNIQUE (namespace_id, outcome_id, id),
    FOREIGN KEY (namespace_id, outcome_id, id, kind)
        REFERENCES entity_refs(namespace_id, outcome_id, id, kind),
    FOREIGN KEY (namespace_id, outcome_id)
        REFERENCES outcomes(namespace_id, id),
    FOREIGN KEY (namespace_id, outcome_id, evidence_id)
        REFERENCES evidence(namespace_id, outcome_id, id),
    FOREIGN KEY (namespace_id, outcome_id, target_id, target_kind)
        REFERENCES entity_refs(namespace_id, outcome_id, id, kind),
    FOREIGN KEY (namespace_id, outcome_id, criterion_id)
        REFERENCES success_criteria(namespace_id, outcome_id, id)
);

CREATE INDEX idx_artifacts_scope
    ON artifacts (namespace_id, outcome_id, lifecycle, id);

CREATE INDEX idx_evidence_scope
    ON evidence (namespace_id, outcome_id, lifecycle, evidence_type, id);

CREATE INDEX idx_evidence_artifact
    ON evidence (namespace_id, outcome_id, artifact_id);

CREATE INDEX idx_evidence_links_target
    ON evidence_links (namespace_id, outcome_id, target_kind, target_id, lifecycle, id);

CREATE INDEX idx_evidence_links_evidence
    ON evidence_links (namespace_id, outcome_id, evidence_id, lifecycle, id);

CREATE INDEX idx_decisions_scope
    ON decisions (namespace_id, outcome_id, lifecycle, id);
