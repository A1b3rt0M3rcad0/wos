CREATE TABLE criterion_assessment_evidence (
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    assessment_id TEXT NOT NULL,
    evidence_id TEXT NOT NULL,
    PRIMARY KEY (namespace_id, outcome_id, assessment_id, evidence_id),
    FOREIGN KEY (namespace_id, outcome_id, assessment_id)
        REFERENCES criterion_assessments(namespace_id, outcome_id, id),
    FOREIGN KEY (namespace_id, outcome_id, evidence_id)
        REFERENCES evidence(namespace_id, outcome_id, id)
);

CREATE INDEX idx_criterion_assessment_evidence_evidence
    ON criterion_assessment_evidence (namespace_id, outcome_id, evidence_id, assessment_id);
